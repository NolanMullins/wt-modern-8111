package warthunder

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const maxResponseSize = 10 << 20

type Client struct {
	baseURL string
	http    *http.Client
}

type MapInfo struct {
	Valid        bool      `json:"valid"`
	ValidPresent bool      `json:"-"`
	GridSize     []float64 `json:"grid_size,omitempty"`
	GridSteps    []float64 `json:"grid_steps,omitempty"`
	GridZero     []float64 `json:"grid_zero,omitempty"`
	HUDType      int       `json:"hud_type,omitempty"`
	Generation   int       `json:"map_generation,omitempty"`
	MapMin       []float64 `json:"map_min,omitempty"`
	MapMax       []float64 `json:"map_max,omitempty"`
}

type MapObject struct {
	Type       string    `json:"type"`
	Color      string    `json:"color,omitempty"`
	ColorArray []float64 `json:"color[],omitempty"`
	Blink      int       `json:"blink,omitempty"`
	Icon       string    `json:"icon,omitempty"`
	IconBG     string    `json:"icon_bg,omitempty"`
	X          *float64  `json:"x,omitempty"`
	Y          *float64  `json:"y,omitempty"`
	DX         *float64  `json:"dx,omitempty"`
	DY         *float64  `json:"dy,omitempty"`
	SX         *float64  `json:"sx,omitempty"`
	SY         *float64  `json:"sy,omitempty"`
	EX         *float64  `json:"ex,omitempty"`
	EY         *float64  `json:"ey,omitempty"`
}

type Objective struct {
	Primary bool   `json:"primary"`
	Status  string `json:"status"`
	Text    string `json:"text"`
}

type Mission struct {
	Status     string       `json:"status"`
	Objectives *[]Objective `json:"objectives"`
}

type FeedRecord struct {
	ID      int     `json:"id"`
	Message string  `json:"msg"`
	Sender  string  `json:"sender"`
	Enemy   bool    `json:"enemy"`
	Mode    string  `json:"mode"`
	Time    float64 `json:"time,omitempty"`
}

type HUDMessages struct {
	Events []FeedRecord `json:"events"`
	Damage []FeedRecord `json:"damage"`
}

type Image struct {
	Body        []byte
	ContentType string
}

func NewClient(baseURL string, timeout time.Duration) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		http: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *Client) State(ctx context.Context) (map[string]any, error) {
	var value map[string]any
	return value, c.getJSON(ctx, "/state", &value)
}

func (c *Client) Indicators(ctx context.Context) (map[string]any, error) {
	var value map[string]any
	return value, c.getJSON(ctx, "/indicators", &value)
}

func (c *Client) MapObjects(ctx context.Context) ([]MapObject, error) {
	var value []MapObject
	return value, c.getJSON(ctx, "/map_obj.json", &value)
}

func (c *Client) MapInfo(ctx context.Context) (MapInfo, error) {
	var raw map[string]any
	if err := c.getJSON(ctx, "/map_info.json", &raw); err != nil {
		return MapInfo{}, err
	}
	return parseMapInfo(raw)
}

func (c *Client) Mission(ctx context.Context) (Mission, error) {
	var value Mission
	return value, c.getJSON(ctx, "/mission.json", &value)
}

func (c *Client) GameChat(ctx context.Context, lastID int) ([]FeedRecord, error) {
	var value []FeedRecord
	return value, c.getJSON(ctx, "/gamechat?lastId="+strconv.Itoa(lastID), &value)
}

func (c *Client) HUDMessages(ctx context.Context, lastEventID, lastDamageID int) (HUDMessages, error) {
	path := fmt.Sprintf("/hudmsg?lastEvt=%d&lastDmg=%d", lastEventID, lastDamageID)
	var value HUDMessages
	return value, c.getJSON(ctx, path, &value)
}

func (c *Client) MapImage(ctx context.Context, generation int) (Image, error) {
	path := "/map.img?gen=" + strconv.Itoa(generation)
	request, err := c.request(ctx, path)
	if err != nil {
		return Image{}, err
	}
	response, err := c.http.Do(request)
	if err != nil {
		return Image{}, fmt.Errorf("GET %s: %w", path, err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return Image{}, fmt.Errorf("GET %s: HTTP %d", path, response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxResponseSize+1))
	if err != nil {
		return Image{}, fmt.Errorf("read %s: %w", path, err)
	}
	if len(body) > maxResponseSize {
		return Image{}, fmt.Errorf("read %s: response exceeds %d bytes", path, maxResponseSize)
	}
	contentType := response.Header.Get("Content-Type")
	if contentType != "image/png" && contentType != "image/jpeg" {
		return Image{}, fmt.Errorf("GET %s: unsupported content type %q", path, contentType)
	}
	return Image{Body: body, ContentType: contentType}, nil
}

func (c *Client) getJSON(ctx context.Context, path string, target any) error {
	request, err := c.request(ctx, path)
	if err != nil {
		return err
	}
	response, err := c.http.Do(request)
	if err != nil {
		return fmt.Errorf("GET %s: %w", path, err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: HTTP %d", path, response.StatusCode)
	}
	decoder := json.NewDecoder(io.LimitReader(response.Body, maxResponseSize))
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}

func (c *Client) request(ctx context.Context, path string) (*http.Request, error) {
	parsed, err := url.Parse(c.baseURL + path)
	if err != nil {
		return nil, fmt.Errorf("parse War Thunder URL: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create GET %s: %w", path, err)
	}
	request.Header.Set("Accept", "application/json, image/png, image/jpeg")
	return request, nil
}

func parseMapInfo(raw map[string]any) (MapInfo, error) {
	_, validPresent := raw["valid"]
	value := MapInfo{Valid: boolValue(raw["valid"]), ValidPresent: validPresent}
	var err error
	if value.GridSize, err = numberArray(raw["grid_size"]); err != nil {
		return MapInfo{}, fmt.Errorf("grid_size: %w", err)
	}
	if value.GridSteps, err = numberArray(raw["grid_steps"]); err != nil {
		return MapInfo{}, fmt.Errorf("grid_steps: %w", err)
	}
	if value.GridZero, err = numberArray(raw["grid_zero"]); err != nil {
		return MapInfo{}, fmt.Errorf("grid_zero: %w", err)
	}
	if value.MapMin, err = numberArray(raw["map_min"]); err != nil {
		return MapInfo{}, fmt.Errorf("map_min: %w", err)
	}
	if value.MapMax, err = numberArray(raw["map_max"]); err != nil {
		return MapInfo{}, fmt.Errorf("map_max: %w", err)
	}
	value.Generation = int(numberValue(raw["map_generation"]))
	value.HUDType = int(numberValue(raw["hud_type"]))
	if !value.ValidPresent {
		value.Valid = validMapGeometry(value)
	}
	return value, nil
}

func validMapGeometry(info MapInfo) bool {
	if len(info.MapMin) < 2 || len(info.MapMax) < 2 || len(info.GridSteps) < 2 {
		return false
	}
	for index := 0; index < 2; index++ {
		if math.IsNaN(info.MapMin[index]) || math.IsInf(info.MapMin[index], 0) ||
			math.IsNaN(info.MapMax[index]) || math.IsInf(info.MapMax[index], 0) ||
			math.IsNaN(info.GridSteps[index]) || math.IsInf(info.GridSteps[index], 0) ||
			info.MapMax[index] <= info.MapMin[index] || info.GridSteps[index] <= 0 {
			return false
		}
	}
	return true
}

func numberArray(raw any) ([]float64, error) {
	if raw == nil {
		return nil, nil
	}
	values, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("expected array, got %T", raw)
	}
	result := make([]float64, 0, len(values))
	for _, value := range values {
		switch typed := value.(type) {
		case float64:
			result = append(result, typed)
		case string:
			number, err := strconv.ParseFloat(typed, 64)
			if err != nil {
				return nil, fmt.Errorf("parse %q: %w", typed, err)
			}
			result = append(result, number)
		default:
			return nil, fmt.Errorf("expected number or numeric string, got %T", value)
		}
	}
	return result, nil
}

func numberValue(raw any) float64 {
	switch typed := raw.(type) {
	case float64:
		return typed
	case string:
		value, _ := strconv.ParseFloat(typed, 64)
		return value
	default:
		return 0
	}
}

func boolValue(raw any) bool {
	value, _ := raw.(bool)
	return value
}
