package heatmap

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const maxHeatmapSize = 32 << 20

var (
	ErrUnknownMap = errors.New("unknown heatmap map")
	ErrNoData     = errors.New("heatmap data unavailable")
)

type Result struct {
	Map     Map
	PNG     []byte
	BasePNG []byte
}

type cachedResult struct {
	result    Result
	err       error
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

	mu    sync.Mutex
	cache map[string]cachedResult

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
		resolve:     ResolveMap,
	}
}

func (client *Client) Fetch(ctx context.Context, mapImage []byte) (Result, error) {
	mapData, err := client.resolve(mapImage)
	if err != nil {
		return Result{}, err
	}
	client.mu.Lock()
	cached, ok := client.cache[mapData.Level]
	if ok && cached.expiresAt.After(client.now()) {
		client.mu.Unlock()
		return cachedResponse(cached)
	}
	client.mu.Unlock()

	// The upstream asks clients not to issue concurrent requests.
	client.fetchMu.Lock()
	defer client.fetchMu.Unlock()
	client.mu.Lock()
	cached, ok = client.cache[mapData.Level]
	if ok && cached.expiresAt.After(client.now()) {
		client.mu.Unlock()
		return cachedResponse(cached)
	}
	client.mu.Unlock()

	endpoint, err := url.Parse(client.baseURL + "/heat")
	if err != nil {
		return Result{}, fmt.Errorf("parse heatmap URL: %w", err)
	}
	query := endpoint.Query()
	query.Set("level", mapData.Level)
	query.Set("scoreIntensity", "32")
	query.Set("countIntensity", "32")
	endpoint.RawQuery = query.Encode()

	body, status, err := client.fetchPNG(ctx, endpoint.String())
	if err != nil {
		return Result{}, client.handleFetchError(ctx, mapData.Level, "fetch heatmap", err)
	}
	if status == http.StatusNoContent || status == http.StatusNotFound {
		return Result{}, client.cacheError(
			mapData.Level,
			fmt.Errorf("%w for %s", ErrNoData, mapData.Level),
			client.negativeTTL,
		)
	}
	if status != http.StatusOK {
		return Result{}, client.cacheError(
			mapData.Level,
			fmt.Errorf("fetch heatmap: HTTP %d", status),
			client.errorTTL,
		)
	}

	baseURL := client.baseURL + "/minimap/2048/" + mapData.Level
	base, status, err := client.fetchPNG(ctx, baseURL)
	if err != nil {
		return Result{}, client.handleFetchError(ctx, mapData.Level, "fetch heatmap base", err)
	}
	if status == http.StatusNoContent || status == http.StatusNotFound {
		return Result{}, client.cacheError(
			mapData.Level,
			fmt.Errorf("%w for %s", ErrNoData, mapData.Level),
			client.negativeTTL,
		)
	}
	if status != http.StatusOK {
		return Result{}, client.cacheError(
			mapData.Level,
			fmt.Errorf("fetch heatmap base: HTTP %d", status),
			client.errorTTL,
		)
	}

	result := Result{Map: mapData, PNG: body, BasePNG: base}
	client.mu.Lock()
	client.cache[mapData.Level] = cachedResult{
		result:    cloneResult(result),
		expiresAt: client.now().Add(client.ttl),
	}
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
	client.cache[level] = cachedResult{err: err, expiresAt: client.now().Add(ttl)}
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

func cachedResponse(cached cachedResult) (Result, error) {
	if cached.err != nil {
		return Result{}, cached.err
	}
	return cloneResult(cached.result), nil
}

func cloneResult(result Result) Result {
	return Result{
		Map:     result.Map,
		PNG:     append([]byte(nil), result.PNG...),
		BasePNG: append([]byte(nil), result.BasePNG...),
	}
}
