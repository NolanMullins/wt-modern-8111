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

	"github.com/NolanMullins/wt-modern-8111/internal/heatmap"
	"github.com/NolanMullins/wt-modern-8111/internal/polling"
	"github.com/NolanMullins/wt-modern-8111/internal/telemetry"
)

type fakeSource struct {
	snapshot telemetry.Snapshot
	image    []byte
	mime     string
	gen      int
}

type fakeHeatmapSource struct {
	result heatmap.Result
	err    error
}

type noGroundSource struct {
	*fakeSource
}

func (source noGroundSource) GroundMapImage() ([]byte, string, int, bool) {
	return nil, "", 0, false
}

func (source fakeHeatmapSource) Fetch(context.Context, []byte) (heatmap.Result, error) {
	return source.result, source.err
}

func (source *fakeSource) Snapshot() telemetry.Snapshot {
	return source.snapshot
}

func (source *fakeSource) MapImage() ([]byte, string, int, bool) {
	return source.image, source.mime, source.gen, len(source.image) > 0
}

func (source *fakeSource) GroundMapImage() ([]byte, string, int, bool) {
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

func TestStatusIncludesApplicationVersion(t *testing.T) {
	handler, err := New(&fakeSource{snapshot: telemetry.Snapshot{
		Connection: telemetry.Connection{State: "live", Mode: "live"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Header().Get("X-WT-Modern-Version") == "" {
		t.Fatal("status response is missing version header")
	}
	var status struct {
		State      string `json:"state"`
		Mode       string `json:"mode"`
		AppVersion string `json:"appVersion"`
		Revision   string `json:"revision"`
	}
	if err := json.NewDecoder(response.Body).Decode(&status); err != nil {
		t.Fatal(err)
	}
	if status.State != "live" || status.Mode != "live" ||
		status.AppVersion == "" || status.Revision == "" {
		t.Fatalf("unexpected status: %+v", status)
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

func TestHeatmapEndpointContract(t *testing.T) {
	source := &fakeSource{image: []byte("map"), mime: "image/jpeg", gen: 2}
	provider := fakeHeatmapSource{result: heatmap.Result{
		Map:     heatmap.Map{Level: "levels/avg_abandoned_town.bin", Name: "Abandoned Town"},
		PNG:     []byte("\x89PNG\r\n\x1a\nheat"),
		BasePNG: []byte("\x89PNG\r\n\x1a\nbase"),
	}}
	handler, err := newWithHeatmaps(source, provider)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/heatmap/2", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	if response.Header().Get("Content-Type") != "image/png" {
		t.Fatalf("content type = %q", response.Header().Get("Content-Type"))
	}
	if response.Header().Get("X-WT-Heatmap-Map") != "Abandoned Town" {
		t.Fatalf("map name = %q", response.Header().Get("X-WT-Heatmap-Map"))
	}
}

func TestGroundMapFallsBackToCurrentView(t *testing.T) {
	source := noGroundSource{&fakeSource{image: []byte("cas"), mime: "image/jpeg", gen: 3}}
	handler, err := New(source)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/ground-map/3", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK ||
		response.Header().Get("X-WT-Map-Fallback") != "current-view" ||
		response.Body.String() != "cas" {
		t.Fatalf("ground map fallback = %d, headers=%v", response.Code, response.Header())
	}
}

func TestHeatmapRequiresGroundReference(t *testing.T) {
	source := noGroundSource{&fakeSource{image: []byte("cas"), mime: "image/jpeg", gen: 3}}
	handler, err := newWithHeatmaps(source, fakeHeatmapSource{})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/heatmap/3", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("heatmap status = %d, want 404", response.Code)
	}
}

func TestGroundMapEndpointContract(t *testing.T) {
	source := &fakeSource{image: []byte("ground"), mime: "image/png", gen: 3}
	handler, err := New(source)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/ground-map/3", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || response.Body.String() != "ground" {
		t.Fatalf("ground map response = %d %q", response.Code, response.Body.String())
	}
}

func TestHeatmapEndpointReportsUnavailableData(t *testing.T) {
	source := &fakeSource{image: []byte("map"), mime: "image/jpeg", gen: 2}
	handler, err := newWithHeatmaps(
		source,
		fakeHeatmapSource{err: heatmap.ErrNoData},
	)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/heatmap/2", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", response.Code)
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
