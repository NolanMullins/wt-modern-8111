package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/NolanMullins/wt-modern-8111/internal/polling"
	"github.com/NolanMullins/wt-modern-8111/internal/telemetry"
)

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
