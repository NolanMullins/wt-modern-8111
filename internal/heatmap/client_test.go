package heatmap

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
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
		if requests == 2 &&
			request.URL.Path != "/minimap/2048/levels/avg_abandoned_town.bin" {
			t.Errorf("minimap path = %q", request.URL.Path)
		}
		if request.Header.Get("User-Agent") == "" {
			t.Error("missing User-Agent")
		}
		writer.Header().Set("Content-Type", "image/png")
		_, _ = writer.Write([]byte("\x89PNG\r\n\x1a\nheatmap"))
	}))
	defer upstream.Close()

	client := NewClient(upstream.URL, time.Second)
	client.minInterval = 0
	client.resolve = func([]byte) (Map, error) {
		return Map{Level: "levels/avg_abandoned_town.bin", Name: "Abandoned Town"}, nil
	}

	first, err := client.Fetch(context.Background(), []byte("map"))
	if err != nil {
		t.Fatal(err)
	}
	first.PNG[0] = 0
	second, err := client.Fetch(context.Background(), []byte("map"))
	if err != nil {
		t.Fatal(err)
	}
	if requests != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}
	if second.PNG[0] != 0x89 {
		t.Fatal("cached image was not cloned")
	}
	if len(second.BasePNG) == 0 {
		t.Fatal("base minimap was not loaded")
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

	_, err := client.Fetch(context.Background(), []byte("map"))
	if !errors.Is(err, ErrNoData) {
		t.Fatalf("error = %v, want ErrNoData", err)
	}
	_, err = client.Fetch(context.Background(), []byte("map"))
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

	_, err := client.Fetch(ctx, []byte("map"))
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
	requests := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		requests++
		time.Sleep(50 * time.Millisecond)
	}))
	defer upstream.Close()
	client := NewClient(upstream.URL, 10*time.Millisecond)
	client.minInterval = 0
	client.resolve = func([]byte) (Map, error) {
		return Map{Level: "levels/avg_abandoned_town.bin"}, nil
	}

	if _, err := client.Fetch(context.Background(), []byte("map")); err == nil {
		t.Fatal("first request unexpectedly succeeded")
	}
	if _, err := client.Fetch(context.Background(), []byte("map")); err == nil {
		t.Fatal("cached request unexpectedly succeeded")
	}
	if requests != 1 {
		t.Fatalf("requests = %d, want 1", requests)
	}
}
