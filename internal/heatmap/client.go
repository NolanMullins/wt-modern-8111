package heatmap

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/disintegration/imaging"
)

const maxHeatmapSize = 32 << 20

const (
	maxHeatmapDimension = 8192
	maxHeatmapPixels    = 50 << 20
	maxBaseDimension    = 4096
	maxBasePixels       = 16 << 20
	minimumLayerSamples = 5
	maxCachedTeams      = 32
	maxCachedBases      = 16
)

var (
	ErrUnknownMap = errors.New("unknown heatmap map")
	ErrNoData     = errors.New("heatmap data unavailable")
)

type Result struct {
	Map       Map
	FiringPNG []byte
	VictimPNG []byte
	BasePNG   []byte
}

type cachedResult struct {
	result    Result
	err       error
	expiresAt time.Time
}

type cachedPNG struct {
	body      []byte
	expiresAt time.Time
}

type Client struct {
	baseURL     string
	http        *http.Client
	now         func() time.Time
	ttl         time.Duration
	negativeTTL time.Duration
	errorTTL    time.Duration
	minInterval time.Duration

	mu        sync.Mutex
	cache     map[string]cachedResult
	baseCache map[string]cachedPNG

	fetchMu   sync.Mutex
	lastFetch time.Time
	resolve   func([]byte) (Map, error)
}

func NewClient(baseURL string, timeout time.Duration) *Client {
	return &Client{
		baseURL:     strings.TrimRight(baseURL, "/"),
		http:        &http.Client{Timeout: timeout},
		now:         time.Now,
		ttl:         6 * time.Hour,
		negativeTTL: time.Hour,
		errorTTL:    time.Minute,
		minInterval: 2 * time.Second,
		cache:       make(map[string]cachedResult),
		baseCache:   make(map[string]cachedPNG),
		resolve:     ResolveMap,
	}
}

func (client *Client) Fetch(ctx context.Context, mapImage []byte, killerTeam int) (Result, error) {
	if killerTeam != 1 && killerTeam != 2 {
		return Result{}, fmt.Errorf("invalid killer team %d", killerTeam)
	}
	mapData, err := client.resolve(mapImage)
	if err != nil {
		return Result{}, err
	}
	cacheKey := fmt.Sprintf("%s:team:%d", mapData.Level, killerTeam)
	client.mu.Lock()
	client.pruneCacheLocked()
	cached, ok := client.cache[cacheKey]
	base := client.baseCache[mapData.Level]
	if ok && (cached.err != nil || len(base.body) > 0) {
		client.mu.Unlock()
		return cachedResponse(cached, base.body)
	}
	client.mu.Unlock()

	// The upstream asks clients not to issue concurrent requests.
	client.fetchMu.Lock()
	defer client.fetchMu.Unlock()
	client.mu.Lock()
	client.pruneCacheLocked()
	cached, ok = client.cache[cacheKey]
	base = client.baseCache[mapData.Level]
	if ok && (cached.err != nil || len(base.body) > 0) {
		client.mu.Unlock()
		return cachedResponse(cached, base.body)
	}
	client.mu.Unlock()

	endpoint, err := url.Parse(client.baseURL + "/heat")
	if err != nil {
		return Result{}, fmt.Errorf("parse heatmap URL: %w", err)
	}
	query := endpoint.Query()
	query.Set("level", mapData.Level)
	query.Set("scoreIntensity", "1")
	query.Set("countIntensity", "1")
	query.Set("killerTeam", strconv.Itoa(killerTeam))
	endpoint.RawQuery = query.Encode()

	body, status, err := client.fetchPNG(ctx, endpoint.String())
	if err != nil {
		return Result{}, client.handleFetchError(ctx, cacheKey, "fetch heatmap", err)
	}
	if status == http.StatusNoContent || status == http.StatusNotFound {
		return Result{}, client.cacheError(
			cacheKey,
			fmt.Errorf("%w for %s", ErrNoData, mapData.Level),
			client.negativeTTL,
		)
	}
	if status != http.StatusOK {
		return Result{}, client.cacheError(
			cacheKey,
			fmt.Errorf("fetch heatmap: HTTP %d", status),
			client.errorTTL,
		)
	}
	if err := validatePNGDimensions(body, maxHeatmapDimension, maxHeatmapPixels); err != nil {
		return Result{}, client.cacheError(
			cacheKey,
			fmt.Errorf("validate heatmap: %w", err),
			client.errorTTL,
		)
	}

	baseBody := append([]byte(nil), base.body...)
	if len(baseBody) == 0 {
		baseURL := client.baseURL + "/minimap/2048/" + mapData.Level
		baseBody, status, err = client.fetchPNG(ctx, baseURL)
		if err != nil {
			return Result{}, client.handleFetchError(ctx, cacheKey, "fetch heatmap base", err)
		}
		if status == http.StatusNoContent || status == http.StatusNotFound {
			return Result{}, client.cacheError(
				cacheKey,
				fmt.Errorf("%w for %s", ErrNoData, mapData.Level),
				client.negativeTTL,
			)
		}
		if status != http.StatusOK {
			return Result{}, client.cacheError(
				cacheKey,
				fmt.Errorf("fetch heatmap base: HTTP %d", status),
				client.errorTTL,
			)
		}
		if err := validatePNGDimensions(baseBody, maxBaseDimension, maxBasePixels); err != nil {
			return Result{}, client.cacheError(
				cacheKey,
				fmt.Errorf("validate heatmap base: %w", err),
				client.errorTTL,
			)
		}
		baseBody, err = resizeBaseMap(baseBody)
		if err != nil {
			return Result{}, client.cacheError(
				cacheKey,
				fmt.Errorf("resize heatmap base: %w", err),
				client.errorTTL,
			)
		}
	}

	firing, victim, err := splitHeatmap(body)
	if err != nil {
		return Result{}, client.cacheError(
			cacheKey,
			fmt.Errorf("decode heatmap: %w", err),
			client.errorTTL,
		)
	}
	result := Result{Map: mapData, FiringPNG: firing, VictimPNG: victim, BasePNG: baseBody}
	client.mu.Lock()
	cacheResult := cloneResult(result)
	cacheResult.BasePNG = nil
	client.cache[cacheKey] = cachedResult{
		result:    cacheResult,
		expiresAt: client.now().Add(client.ttl),
	}
	client.baseCache[mapData.Level] = cachedPNG{
		body:      append([]byte(nil), baseBody...),
		expiresAt: client.now().Add(client.ttl),
	}
	client.trimCacheLocked()
	client.mu.Unlock()
	return result, nil
}

func (client *Client) fetchPNG(ctx context.Context, endpoint string) ([]byte, int, error) {
	if wait := client.requestWait(); wait > 0 {
		timer := time.NewTimer(wait)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return nil, 0, ctx.Err()
		case <-timer.C:
		}
	}
	client.lastFetch = client.now()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("create request: %w", err)
	}
	request.Header.Set(
		"User-Agent",
		"WT-Modern-8111 heatmap integration (+https://github.com/NolanMullins/wt-modern-8111)",
	)
	response, err := client.http.Do(request)
	if err != nil {
		return nil, 0, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, response.StatusCode, nil
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxHeatmapSize+1))
	if err != nil {
		return nil, response.StatusCode, fmt.Errorf("read response: %w", err)
	}
	if len(body) > maxHeatmapSize {
		return nil, response.StatusCode, fmt.Errorf("response exceeds %d bytes", maxHeatmapSize)
	}
	if !bytes.HasPrefix(body, []byte("\x89PNG\r\n\x1a\n")) {
		return nil, response.StatusCode, errors.New("response is not PNG")
	}
	return body, response.StatusCode, nil
}

func (client *Client) requestWait() time.Duration {
	if client.lastFetch.IsZero() {
		return 0
	}
	return max(0, client.minInterval-client.now().Sub(client.lastFetch))
}

func (client *Client) cacheError(level string, err error, ttl time.Duration) error {
	client.mu.Lock()
	client.pruneCacheLocked()
	client.cache[level] = cachedResult{err: err, expiresAt: client.now().Add(ttl)}
	client.trimCacheLocked()
	client.mu.Unlock()
	return err
}

func (client *Client) handleFetchError(
	ctx context.Context,
	level, operation string,
	err error,
) error {
	wrapped := fmt.Errorf("%s: %w", operation, err)
	if ctx.Err() != nil {
		return wrapped
	}
	return client.cacheError(level, wrapped, client.errorTTL)
}

func cachedResponse(cached cachedResult, base []byte) (Result, error) {
	if cached.err != nil {
		return Result{}, cached.err
	}
	result := cloneResult(cached.result)
	result.BasePNG = append([]byte(nil), base...)
	return result, nil
}

func cloneResult(result Result) Result {
	return Result{
		Map:       result.Map,
		FiringPNG: append([]byte(nil), result.FiringPNG...),
		VictimPNG: append([]byte(nil), result.VictimPNG...),
		BasePNG:   append([]byte(nil), result.BasePNG...),
	}
}

func splitHeatmap(body []byte) ([]byte, []byte, error) {
	source, err := png.Decode(bytes.NewReader(body))
	if err != nil {
		return nil, nil, err
	}
	bounds := source.Bounds()
	const binSize = 4
	width := (bounds.Dx() + binSize - 1) / binSize
	height := (bounds.Dy() + binSize - 1) / binSize
	firingCounts := make([]uint16, width*height)
	victimCounts := make([]uint16, width*height)
	switch typed := source.(type) {
	case *image.NRGBA:
		accumulateNRGBA(typed, firingCounts, victimCounts, width, binSize)
	case *image.RGBA:
		accumulateRGBA(typed, firingCounts, victimCounts, width, binSize)
	default:
		accumulateGeneric(source, firingCounts, victimCounts, width, binSize)
	}
	outputBounds := image.Rect(0, 0, width, height)
	firing := image.NewNRGBA(outputBounds)
	victims := image.NewNRGBA(outputBounds)
	for index, count := range firingCounts {
		if count >= minimumLayerSamples {
			offset := index * 4
			firing.Pix[offset] = 255
			firing.Pix[offset+1] = 78
			firing.Pix[offset+3] = sampleAlpha(int(count))
		}
		if victimCount := victimCounts[index]; victimCount >= minimumLayerSamples {
			offset := index * 4
			victims.Pix[offset+1] = 166
			victims.Pix[offset+2] = 255
			victims.Pix[offset+3] = sampleAlpha(int(victimCount))
		}
	}
	firingPNG, err := encodePNG(firing)
	if err != nil {
		return nil, nil, err
	}
	victimPNG, err := encodePNG(victims)
	if err != nil {
		return nil, nil, err
	}
	return firingPNG, victimPNG, nil
}

func accumulateNRGBA(
	source *image.NRGBA,
	firingCounts, victimCounts []uint16,
	width, binSize int,
) {
	bounds := source.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		offset := (y - bounds.Min.Y) * source.Stride
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			alpha := source.Pix[offset+3]
			pixel := color.RGBA{
				R: premultiply(source.Pix[offset], alpha),
				G: premultiply(source.Pix[offset+1], alpha),
				B: premultiply(source.Pix[offset+2], alpha),
				A: alpha,
			}
			accumulateSample(pixel, x-bounds.Min.X, y-bounds.Min.Y,
				firingCounts, victimCounts, width, binSize)
			offset += 4
		}
	}
}

func accumulateRGBA(
	source *image.RGBA,
	firingCounts, victimCounts []uint16,
	width, binSize int,
) {
	bounds := source.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		offset := (y - bounds.Min.Y) * source.Stride
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			accumulateSample(
				color.RGBA{
					R: source.Pix[offset],
					G: source.Pix[offset+1],
					B: source.Pix[offset+2],
					A: source.Pix[offset+3],
				},
				x-bounds.Min.X,
				y-bounds.Min.Y,
				firingCounts,
				victimCounts,
				width,
				binSize,
			)
			offset += 4
		}
	}
}

func accumulateGeneric(
	source image.Image,
	firingCounts, victimCounts []uint16,
	width, binSize int,
) {
	bounds := source.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			pixel := color.RGBAModel.Convert(source.At(x, y)).(color.RGBA)
			accumulateSample(pixel, x-bounds.Min.X, y-bounds.Min.Y,
				firingCounts, victimCounts, width, binSize)
		}
	}
}

func accumulateSample(
	pixel color.RGBA,
	x, y int,
	firingCounts, victimCounts []uint16,
	width, binSize int,
) {
	// The upstream writes four attribution lines into this fixed corner.
	if width*binSize >= 1024 && x < 512 && y < 96 {
		return
	}
	score, ok := heatScore(pixel)
	if !ok {
		return
	}
	count := int(pixel.A)
	index := (y/binSize)*width + x/binSize
	// score = kills - deaths and count = kills + deaths.
	firingCounts[index] = saturatingAdd(firingCounts[index], max(0, (count+score)/2))
	victimCounts[index] = saturatingAdd(victimCounts[index], max(0, (count-score)/2))
}

func heatScore(pixel color.RGBA) (int, bool) {
	if pixel.A == 0 {
		return 0, false
	}
	score := int(pixel.R)
	if pixel.R > 0 && pixel.B > 0 {
		return 0, false
	}
	if pixel.B > 0 {
		score = -int(pixel.B)
	}
	if abs(score) > int(pixel.A) {
		return 0, false
	}
	return score, true
}

func premultiply(value, alpha uint8) uint8 {
	return uint8((uint16(value)*uint16(alpha) + 127) / 255)
}

func sampleAlpha(count int) uint8 {
	return uint8(min(
		255,
		64+int(math.Round(math.Sqrt(float64(count-minimumLayerSamples+1))*64)),
	))
}

func encodePNG(source image.Image) ([]byte, error) {
	var encoded bytes.Buffer
	encoder := png.Encoder{CompressionLevel: png.BestSpeed}
	if err := encoder.Encode(&encoded, source); err != nil {
		return nil, err
	}
	return encoded.Bytes(), nil
}

func resizeBaseMap(body []byte) ([]byte, error) {
	source, err := png.Decode(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	if source.Bounds().Dx() <= 1024 && source.Bounds().Dy() <= 1024 {
		return body, nil
	}
	return encodePNG(imaging.Fit(source, 1024, 1024, imaging.Lanczos))
}

func validatePNGDimensions(body []byte, maxDimension, maxPixels int) error {
	config, err := png.DecodeConfig(bytes.NewReader(body))
	if err != nil {
		return err
	}
	if config.Width <= 0 || config.Height <= 0 ||
		config.Width > maxDimension || config.Height > maxDimension ||
		config.Width > maxPixels/config.Height {
		return fmt.Errorf("image dimensions %dx%d exceed limits", config.Width, config.Height)
	}
	return nil
}

func saturatingAdd(current uint16, value int) uint16 {
	return uint16(min(int(^uint16(0)), int(current)+value))
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func (client *Client) pruneCacheLocked() {
	now := client.now()
	for key, cached := range client.cache {
		if !cached.expiresAt.After(now) {
			delete(client.cache, key)
		}
	}
	for key, cached := range client.baseCache {
		if !cached.expiresAt.After(now) {
			delete(client.baseCache, key)
		}
	}
}

func (client *Client) trimCacheLocked() {
	for len(client.cache) > maxCachedTeams {
		deleteOldestResult(client.cache)
	}
	for len(client.baseCache) > maxCachedBases {
		deleteOldestPNG(client.baseCache)
	}
}

func deleteOldestResult(cache map[string]cachedResult) {
	var oldestKey string
	var oldest time.Time
	for key, value := range cache {
		if oldestKey == "" || value.expiresAt.Before(oldest) {
			oldestKey, oldest = key, value.expiresAt
		}
	}
	delete(cache, oldestKey)
}

func deleteOldestPNG(cache map[string]cachedPNG) {
	var oldestKey string
	var oldest time.Time
	for key, value := range cache {
		if oldestKey == "" || value.expiresAt.Before(oldest) {
			oldestKey, oldest = key, value.expiresAt
		}
	}
	delete(cache, oldestKey)
}
