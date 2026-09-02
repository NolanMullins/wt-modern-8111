package updater

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestNewerVersion(t *testing.T) {
	for _, test := range []struct {
		current   string
		candidate string
		want      bool
	}{
		{"1.0.0", "1.0.1", true},
		{"1.2.0", "2.0.0", true},
		{"1.2.3", "1.2.3", false},
		{"2.0.0", "1.9.9", false},
		{"dev", "1.0.0", false},
	} {
		if got := newerVersion(test.current, test.candidate); got != test.want {
			t.Errorf("newerVersion(%q, %q) = %v, want %v", test.current, test.candidate, got, test.want)
		}
	}
}

func TestCheckAndStageVerifiedRelease(t *testing.T) {
	binary := []byte("versioned executable")
	sum := sha256.Sum256(binary)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/latest":
			writer.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintf(writer, `{
				"tag_name":"v1.1.0",
				"assets":[
					{"name":"%s","state":"uploaded","digest":"sha256:%s","browser_download_url":"%s/binary"},
					{"name":"%s","state":"uploaded","browser_download_url":"%s/checksums"}
				]
			}`, binaryName, hex.EncodeToString(sum[:]), serverURL(request), checksumsName, serverURL(request))
		case "/binary":
			_, _ = writer.Write(binary)
		case "/checksums":
			_, _ = fmt.Fprintf(writer, "%s  %s\n", hex.EncodeToString(sum[:]), binaryName)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	manager := &Manager{
		currentVersion: "1.0.0",
		apiURL:         server.URL + "/latest",
		http:           server.Client(),
		stagingDir:     t.TempDir(),
		targetPath:     filepath.Join(t.TempDir(), "wt-modern.exe"),
		allowInsecure:  true,
	}

	release, staged, err := manager.CheckAndStage(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if release == nil || release.Version != "1.1.0" {
		t.Fatalf("release = %+v", release)
	}
	contents, err := os.ReadFile(staged)
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != string(binary) {
		t.Fatalf("staged contents = %q", contents)
	}
}

func TestCheckAndStageRejectsChecksumMismatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/latest":
			_, _ = fmt.Fprintf(writer, `{
				"tag_name":"v1.1.0",
				"assets":[
					{"name":"%s","state":"uploaded","browser_download_url":"%s/binary"},
					{"name":"%s","state":"uploaded","browser_download_url":"%s/checksums"}
				]
			}`, binaryName, serverURL(request), checksumsName, serverURL(request))
		case "/binary":
			_, _ = writer.Write([]byte("tampered"))
		case "/checksums":
			_, _ = fmt.Fprintf(writer, "%064d  %s\n", 0, binaryName)
		}
	}))
	defer server.Close()
	manager := &Manager{
		currentVersion: "1.0.0",
		apiURL:         server.URL + "/latest",
		http:           server.Client(),
		stagingDir:     filepath.Join(t.TempDir(), "updates"),
		targetPath:     filepath.Join(t.TempDir(), "wt-modern.exe"),
		allowInsecure:  true,
	}

	if _, _, err := manager.CheckAndStage(context.Background()); err == nil {
		t.Fatal("checksum mismatch was accepted")
	}
}

func serverURL(request *http.Request) string {
	return "http://" + request.Host
}
