package server

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/NolanMullins/wt-modern-8111/internal/buildinfo"
	"github.com/NolanMullins/wt-modern-8111/internal/telemetry"
	"github.com/NolanMullins/wt-modern-8111/internal/webui"
)

type Source interface {
	Snapshot() telemetry.Snapshot
	MapImage() ([]byte, string, int, bool)
}

func New(service Source) (http.Handler, error) {
	assets, err := webui.FS()
	if err != nil {
		return nil, fmt.Errorf("open embedded frontend: %w", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/status", func(writer http.ResponseWriter, request *http.Request) {
		writeJSON(writer, struct {
			telemetry.Connection
			AppVersion string `json:"appVersion"`
			Revision   string `json:"revision"`
		}{
			Connection: service.Snapshot().Connection,
			AppVersion: buildinfo.Current(),
			Revision:   buildinfo.ShortRevision(),
		})
	})
	mux.HandleFunc("GET /api/v1/snapshot", func(writer http.ResponseWriter, request *http.Request) {
		writeJSON(writer, service.Snapshot())
	})
	mux.HandleFunc("GET /api/v1/events", func(writer http.ResponseWriter, request *http.Request) {
		streamSnapshots(writer, request, service)
	})
	mux.HandleFunc("GET /api/v1/map/{generation}", func(writer http.ResponseWriter, request *http.Request) {
		serveMap(writer, request, service)
	})
	mux.Handle("/", spaHandler(assets))
	return securityHeaders(mux), nil
}

func writeJSON(writer http.ResponseWriter, value any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.Header().Set("Cache-Control", "no-store")
	if err := json.NewEncoder(writer).Encode(value); err != nil {
		http.Error(writer, "encode response", http.StatusInternalServerError)
	}
}

func streamSnapshots(writer http.ResponseWriter, request *http.Request, service Source) {
	flusher, ok := writer.(http.Flusher)
	if !ok {
		http.Error(writer, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	writer.Header().Set("Content-Type", "text/event-stream")
	writer.Header().Set("Cache-Control", "no-store")
	writer.Header().Set("Connection", "keep-alive")

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	var sequence uint64
	for {
		snapshot := service.Snapshot()
		if snapshot.Sequence != sequence {
			payload, err := json.Marshal(snapshot)
			if err != nil {
				return
			}
			if _, err := fmt.Fprintf(writer, "event: snapshot\ndata: %s\n\n", payload); err != nil {
				return
			}
			flusher.Flush()
			sequence = snapshot.Sequence
		}
		select {
		case <-request.Context().Done():
			return
		case <-ticker.C:
		}
	}
}

func serveMap(writer http.ResponseWriter, request *http.Request, service Source) {
	requested, err := strconv.Atoi(request.PathValue("generation"))
	if err != nil {
		http.Error(writer, "invalid generation", http.StatusBadRequest)
		return
	}
	body, contentType, generation, ok := service.MapImage()
	if !ok || requested != generation {
		http.Error(writer, "map image unavailable", http.StatusNotFound)
		return
	}
	writer.Header().Set("Content-Type", contentType)
	writer.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write(body)
}

func spaHandler(assets fs.FS) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		name := strings.TrimPrefix(path.Clean(request.URL.Path), "/")
		if name == "." || name == "" {
			name = "index.html"
		}
		body, err := fs.ReadFile(assets, name)
		if err != nil {
			body, err = fs.ReadFile(assets, "index.html")
			if err != nil {
				http.Error(writer, "frontend has not been built", http.StatusServiceUnavailable)
				return
			}
			name = "index.html"
		}
		if contentType := assetContentType(name); contentType != "" {
			writer.Header().Set("Content-Type", contentType)
		}
		if name == "index.html" {
			writer.Header().Set("Cache-Control", "no-cache")
		} else {
			writer.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}
		_, _ = writer.Write(body)
	})
}

var assetContentTypes = map[string]string{
	".css":   "text/css; charset=utf-8",
	".html":  "text/html; charset=utf-8",
	".ico":   "image/x-icon",
	".js":    "text/javascript; charset=utf-8",
	".json":  "application/json",
	".png":   "image/png",
	".svg":   "image/svg+xml",
	".webp":  "image/webp",
	".woff2": "font/woff2",
}

func assetContentType(name string) string {
	extension := strings.ToLower(path.Ext(name))
	if contentType, ok := assetContentTypes[extension]; ok {
		return contentType
	}
	return mime.TypeByExtension(extension)
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("X-WT-Modern-Version", buildinfo.Current())
		writer.Header().Set("X-Content-Type-Options", "nosniff")
		writer.Header().Set("Referrer-Policy", "no-referrer")
		writer.Header().Set("Content-Security-Policy", "default-src 'self'; img-src 'self' blob:; style-src 'self'; script-src 'self'; connect-src 'self'")
		next.ServeHTTP(writer, request)
	})
}
