package heatmap

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestClientFetchesAndCachesHeatmap(t *testing.T) {
	requests := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests++
		if request.URL.Path == "/heat" &&
			request.URL.Query().Get("level") != "levels/avg_abandoned_town.bin" {
			t.Errorf("level = %q", request.URL.Query().Get("level"))
		}
		if request.URL.Path == "/heat" &&
			(request.URL.Query().Get("killerTeam") != "2" ||
				request.URL.Query().Get("scoreIntensity") != "1" ||
				request.URL.Query().Get("countIntensity") != "1") {
			t.Errorf("heatmap query = %q", request.URL.RawQuery)
		}
		if requests == 2 &&
			request.URL.Path != "/minimap/2048/levels/avg_abandoned_town.bin" {
			t.Errorf("minimap path = %q", request.URL.Path)
		}
		if request.Header.Get("User-Agent") == "" {
			t.Error("missing User-Agent")
		}
		writer.Header().Set("Content-Type", "image/png")
		_, _ = writer.Write(testHeatmapPNG(t))
	}))
	defer upstream.Close()

	client := NewClient(upstream.URL, time.Second)
	client.minInterval = 0
	client.resolve = func([]byte) (Map, error) {
		return Map{Level: "levels/avg_abandoned_town.bin", Name: "Abandoned Town"}, nil
	}

	first, err := client.Fetch(context.Background(), []byte("map"), 2)
	if err != nil {
		t.Fatal(err)
	}
	first.FiringPNG[0] = 0
	second, err := client.Fetch(context.Background(), []byte("map"), 2)
	if err != nil {
		t.Fatal(err)
	}
	if requests != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}
	if second.FiringPNG[0] != 0x89 {
		t.Fatal("cached image was not cloned")
	}
	if len(second.VictimPNG) == 0 {
		t.Fatal("victim layer was not generated")
	}
	if len(second.BasePNG) == 0 {
		t.Fatal("base minimap was not loaded")
	}
	if len(client.cache["levels/avg_abandoned_town.bin:team:2"].result.BasePNG) != 0 {
		t.Fatal("team cache duplicated the base minimap")
	}
	if len(client.baseCache["levels/avg_abandoned_town.bin"].body) == 0 {
		t.Fatal("base minimap was not cached independently")
	}
}

func TestClientPrunesExpiredAndBoundsCaches(t *testing.T) {
	client := NewClient("http://127.0.0.1", time.Second)
	now := time.Now()
	client.now = func() time.Time { return now }
	client.cache["expired"] = cachedResult{expiresAt: now.Add(-time.Second)}
	client.baseCache["expired"] = cachedPNG{expiresAt: now.Add(-time.Second)}
	client.pruneCacheLocked()
	if len(client.cache) != 0 || len(client.baseCache) != 0 {
		t.Fatal("expired cache entries were retained")
	}
	for index := 0; index < maxCachedTeams+2; index++ {
		client.cache[fmt.Sprintf("team-%d", index)] = cachedResult{
			expiresAt: now.Add(time.Duration(index) * time.Second),
		}
	}
	for index := 0; index < maxCachedBases+2; index++ {
		client.baseCache[fmt.Sprintf("base-%d", index)] = cachedPNG{
			expiresAt: now.Add(time.Duration(index) * time.Second),
		}
	}
	client.trimCacheLocked()
	if len(client.cache) != maxCachedTeams || len(client.baseCache) != maxCachedBases {
		t.Fatalf("cache sizes = %d/%d", len(client.cache), len(client.baseCache))
	}
}

func TestSplitHeatmapSeparatesExactFiringAndVictimPositions(t *testing.T) {
	source := image.NewRGBA(image.Rect(0, 0, 16, 1))
	source.SetRGBA(0, 0, encodedHeatPixel(5, 5))
	source.SetRGBA(4, 0, encodedHeatPixel(-6, 6))
	source.SetRGBA(8, 0, encodedHeatPixel(1, 1))
	source.SetRGBA(12, 0, color.RGBA{R: 255, G: 255, B: 255, A: 255})
	var body bytes.Buffer
	if err := png.Encode(&body, source); err != nil {
		t.Fatal(err)
	}
	firingBody, victimBody, err := splitHeatmap(body.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	firing, err := png.Decode(bytes.NewReader(firingBody))
	if err != nil {
		t.Fatal(err)
	}
	victims, err := png.Decode(bytes.NewReader(victimBody))
	if err != nil {
		t.Fatal(err)
	}
	_, _, _, firingAlpha := firing.At(0, 0).RGBA()
	_, _, _, victimAlpha := victims.At(0, 0).RGBA()
	if firingAlpha == 0 {
		t.Fatal("positive-score firing position is transparent")
	}
	if firingAlpha <= victimAlpha {
		t.Fatalf("positive score firing alpha = %d, victim alpha = %d", firingAlpha, victimAlpha)
	}
	_, _, _, firingAlpha = firing.At(1, 0).RGBA()
	_, _, _, victimAlpha = victims.At(1, 0).RGBA()
	if victimAlpha == 0 {
		t.Fatal("negative-score victim position is transparent")
	}
	if victimAlpha <= firingAlpha {
		t.Fatalf("negative score victim alpha = %d, firing alpha = %d", victimAlpha, firingAlpha)
	}
	if _, _, _, alpha := firing.At(2, 0).RGBA(); alpha != 0 {
		t.Fatal("low-confidence pixel was treated as a visible sample")
	}
	if _, _, _, alpha := firing.At(3, 0).RGBA(); alpha != 0 {
		t.Fatal("watermark-like pixel was treated as a sample")
	}
}

func TestValidatePNGDimensionsRejectsOversizedImage(t *testing.T) {
	source := image.NewNRGBA(image.Rect(0, 0, 5000, 1))
	var body bytes.Buffer
	if err := png.Encode(&body, source); err != nil {
		t.Fatal(err)
	}
	if err := validatePNGDimensions(body.Bytes(), 4096, 16<<20); err == nil {
		t.Fatal("oversized image was accepted")
	}
}

func TestClientReportsNoData(t *testing.T) {
	requests := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		requests++
		writer.WriteHeader(http.StatusNoContent)
	}))
	defer upstream.Close()
	client := NewClient(upstream.URL, time.Second)
	client.minInterval = 0
	client.resolve = func([]byte) (Map, error) {
		return Map{Level: "levels/avg_abandoned_town.bin"}, nil
	}

	_, err := client.Fetch(context.Background(), []byte("map"), 1)
	if !errors.Is(err, ErrNoData) {
		t.Fatalf("error = %v, want ErrNoData", err)
	}
	_, err = client.Fetch(context.Background(), []byte("map"), 1)
	if !errors.Is(err, ErrNoData) {
		t.Fatalf("cached error = %v, want ErrNoData", err)
	}
	if requests != 1 {
		t.Fatalf("requests = %d, want 1", requests)
	}
}

func TestClientDoesNotCacheCancellation(t *testing.T) {
	client := NewClient("http://127.0.0.1:1", time.Second)
	client.minInterval = 0
	client.resolve = func([]byte) (Map, error) {
		return Map{Level: "levels/avg_abandoned_town.bin"}, nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.Fetch(ctx, []byte("map"), 1)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
	client.mu.Lock()
	defer client.mu.Unlock()
	if len(client.cache) != 0 {
		t.Fatalf("canceled request cached %d entries", len(client.cache))
	}
}

func TestClientCachesUpstreamTimeout(t *testing.T) {
	var requests atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		requests.Add(1)
		time.Sleep(50 * time.Millisecond)
	}))
	defer upstream.Close()
	client := NewClient(upstream.URL, 10*time.Millisecond)
	client.minInterval = 0
	client.resolve = func([]byte) (Map, error) {
		return Map{Level: "levels/avg_abandoned_town.bin"}, nil
	}

	if _, err := client.Fetch(context.Background(), []byte("map"), 1); err == nil {
		t.Fatal("first request unexpectedly succeeded")
	}
	if _, err := client.Fetch(context.Background(), []byte("map"), 1); err == nil {
		t.Fatal("cached request unexpectedly succeeded")
	}
	if requests.Load() != 1 {
		t.Fatalf("requests = %d, want 1", requests.Load())
	}
}

func testHeatmapPNG(t *testing.T) []byte {
	t.Helper()
	source := image.NewRGBA(image.Rect(0, 0, 2, 1))
	source.SetRGBA(0, 0, encodedHeatPixel(1, 1))
	source.SetRGBA(1, 0, encodedHeatPixel(-1, 1))
	var body bytes.Buffer
	if err := png.Encode(&body, source); err != nil {
		t.Fatal(err)
	}
	return body.Bytes()
}

func encodedHeatPixel(score, count int) color.RGBA {
	pixel := color.RGBA{
		G: uint8(min(score*score, 255) / 2),
		A: uint8(count),
	}
	if score > 0 {
		pixel.R = uint8(score)
	} else {
		pixel.B = uint8(-score)
	}
	return pixel
}
