package server

import (
	"context"
	"encoding/json"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/NolanMullins/wt-modern-8111/internal/polling"
	"github.com/NolanMullins/wt-modern-8111/internal/telemetry"
)

type fakeSource struct {
	snapshot telemetry.Snapshot
	image    []byte
	mime     string
	gen      int
}

func (source *fakeSource) Snapshot() telemetry.Snapshot {
	return source.snapshot
}

func (source *fakeSource) MapImage() ([]byte, string, int, bool) {
	return source.image, source.mime, source.gen, len(source.image) > 0
}

func TestFixtureSnapshotAndFrontend(t *testing.T) {
	fixtures := filepath.Join("..", "..", "docs", "fixtures", "air-test-flight-jh-7")
	service, err := polling.NewFixtureService(fixtures)
	if err != nil {
		t.Fatalf("NewFixtureService: %v", err)
	}

	handler, err := New(service)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	snapshotRequest := httptest.NewRequest(http.MethodGet, "/api/v1/snapshot", nil)
	snapshotResponse := httptest.NewRecorder()
	handler.ServeHTTP(snapshotResponse, snapshotRequest)
	if snapshotResponse.Code != http.StatusOK {
		t.Fatalf("snapshot status = %d, want 200", snapshotResponse.Code)
	}
	var snapshot telemetry.Snapshot
	if err := json.NewDecoder(snapshotResponse.Body).Decode(&snapshot); err != nil {
		t.Fatalf("decode snapshot: %v", err)
	}
	if snapshot.Connection.Mode != "fixture" || snapshot.Vehicle.Type != "jh_7" {
		t.Fatalf("unexpected fixture snapshot: mode=%q vehicle=%q", snapshot.Connection.Mode, snapshot.Vehicle.Type)
	}
	if snapshot.Feed == nil {
		t.Fatal("fixture feed serialized as nil")
	}

	frontendRequest := httptest.NewRequest(http.MethodGet, "/", nil)
	frontendResponse := httptest.NewRecorder()
	handler.ServeHTTP(frontendResponse, frontendRequest)
	if frontendResponse.Code != http.StatusOK {
		t.Fatalf("frontend status = %d, want 200", frontendResponse.Code)
	}
	if frontendResponse.Header().Get("Content-Security-Policy") == "" {
		t.Fatal("frontend response is missing Content-Security-Policy")
	}
}

func TestSnapshotStreamContractAndCancellation(t *testing.T) {
	source := &fakeSource{snapshot: telemetry.Snapshot{Version: 1, Sequence: 7}}
	handler, err := New(source)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/events", nil)
	ctx, cancel := context.WithCancel(request.Context())
	request = request.WithContext(ctx)
	response := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		handler.ServeHTTP(response, request)
		close(done)
	}()

	time.Sleep(250 * time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("snapshot stream did not stop after request cancellation")
	}
	if contentType := response.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "text/event-stream") {
		t.Fatalf("content type = %q", contentType)
	}
	body := response.Body.String()
	if strings.Count(body, "event: snapshot\n") != 1 || !strings.Contains(body, `"sequence":7`) {
		t.Fatalf("unexpected stream body %q", body)
	}
}

func TestMapEndpointContract(t *testing.T) {
	source := &fakeSource{image: []byte("map"), mime: "image/jpeg", gen: 2}
	handler, err := New(source)
	if err != nil {
		t.Fatal(err)
	}
	for path, want := range map[string]int{
		"/api/v1/map/nope": http.StatusBadRequest,
		"/api/v1/map/1":    http.StatusNotFound,
		"/api/v1/map/2":    http.StatusOK,
	} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != want {
			t.Errorf("%s status = %d, want %d", path, response.Code, want)
		}
		if want == http.StatusOK && response.Header().Get("Content-Type") != "image/jpeg" {
			t.Errorf("map content type = %q", response.Header().Get("Content-Type"))
		}
	}
}

func TestStaticJavaScriptUsesExecutableContentType(t *testing.T) {
	assets := fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("<html></html>")},
		"app.js":     &fstest.MapFile{Data: []byte("export {}")},
	}
	request := httptest.NewRequest(http.MethodGet, "/app.js", nil)
	response := httptest.NewRecorder()

	spaHandler(fs.FS(assets)).ServeHTTP(response, request)

	if contentType := response.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "text/javascript") {
		t.Fatalf("JavaScript content type = %q", contentType)
	}
}
