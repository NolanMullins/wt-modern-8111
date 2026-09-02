package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPortalRunning(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/v1/status" {
			http.NotFound(writer, request)
			return
		}
		_, _ = writer.Write([]byte(`{"state":"live","mode":"live"}`))
	}))
	defer server.Close()

	if !portalRunning(server.URL) {
		t.Fatal("running portal was not detected")
	}
}

func TestPortalRunningRejectsOtherServices(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()

	if portalRunning(server.URL) {
		t.Fatal("unrelated HTTP service was detected as the portal")
	}
}
