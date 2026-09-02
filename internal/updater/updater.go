package updater

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	defaultAPIURL = "https://api.github.com/repos/NolanMullins/wt-modern-8111/releases/latest"
	binaryName    = "wt-modern-windows-amd64.exe"
	checksumsName = "checksums.txt"
	maxAPIBytes   = 2 << 20
	maxAssetBytes = 128 << 20
)

var ErrDevelopmentBuild = errors.New("automatic updates require a versioned release build")

type Release struct {
	Version      string
	BinaryURL    string
	BinaryDigest string
	ChecksumsURL string
}

type Manager struct {
	currentVersion string
	apiURL         string
	apiHTTP        *http.Client
	assetHTTP      *http.Client
	stagingDir     string
	targetPath     string
	allowInsecure  bool
}

func New(currentVersion string) (*Manager, error) {
	cache, err := os.UserCacheDir()
	if err != nil {
		return nil, fmt.Errorf("locate update cache: %w", err)
	}
	executable, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("locate installed executable: %w", err)
	}
	return &Manager{
		currentVersion: currentVersion,
		apiURL:         defaultAPIURL,
		apiHTTP:        &http.Client{Timeout: 30 * time.Second},
		assetHTTP:      &http.Client{Timeout: 10 * time.Minute},
		stagingDir:     filepath.Join(cache, "wt-modern-8111", "updates"),
		targetPath:     executable,
	}, nil
}

func (manager *Manager) CheckAndStage(ctx context.Context) (*Release, string, error) {
	if !isSemanticVersion(manager.currentVersion) {
		return nil, "", ErrDevelopmentBuild
	}
	release, err := manager.latest(ctx)
	if err != nil || release == nil {
		return release, "", err
	}
	if err := probeWritable(filepath.Dir(manager.targetPath)); err != nil {
		return nil, "", fmt.Errorf("installed application cannot be updated: %w", err)
	}
	path, err := manager.downloadAndVerify(ctx, *release)
	if err != nil {
		return nil, "", err
	}
	return release, path, nil
}

func (manager *Manager) latest(ctx context.Context) (*Release, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, manager.apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create release request: %w", err)
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("X-GitHub-Api-Version", "2026-03-10")
	request.Header.Set("User-Agent", "WT-Modern-8111/"+manager.currentVersion)
	response, err := manager.apiHTTP.Do(request)
	if err != nil {
		return nil, fmt.Errorf("check latest release: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("check latest release: HTTP %d", response.StatusCode)
	}
	body, err := readLimited(response.Body, maxAPIBytes)
	if err != nil {
		return nil, fmt.Errorf("read latest release: %w", err)
	}
	var document struct {
		TagName    string `json:"tag_name"`
		Draft      bool   `json:"draft"`
		Prerelease bool   `json:"prerelease"`
		Assets     []struct {
			Name   string `json:"name"`
			URL    string `json:"browser_download_url"`
			State  string `json:"state"`
			Digest string `json:"digest"`
		} `json:"assets"`
	}
	if err := json.Unmarshal(body, &document); err != nil {
		return nil, fmt.Errorf("decode latest release: %w", err)
	}
	version := strings.TrimPrefix(strings.TrimSpace(document.TagName), "v")
	if document.Draft || document.Prerelease || !newerVersion(manager.currentVersion, version) {
		return nil, nil
	}
	release := &Release{Version: version}
	for _, asset := range document.Assets {
		if asset.State != "uploaded" {
			continue
		}
		switch asset.Name {
		case binaryName:
			release.BinaryURL = asset.URL
			release.BinaryDigest = asset.Digest
		case checksumsName:
			release.ChecksumsURL = asset.URL
		}
	}
	if release.BinaryURL == "" || release.ChecksumsURL == "" {
		return nil, fmt.Errorf("release %s is missing update assets", document.TagName)
	}
	if !manager.allowInsecure {
		if err := validateGitHubAssetURL(release.BinaryURL); err != nil {
			return nil, err
		}
		if err := validateGitHubAssetURL(release.ChecksumsURL); err != nil {
			return nil, err
		}
	}
	return release, nil
}

func (manager *Manager) downloadAndVerify(ctx context.Context, release Release) (string, error) {
	checksums, err := manager.download(ctx, release.ChecksumsURL, maxAPIBytes)
	if err != nil {
		return "", fmt.Errorf("download checksums: %w", err)
	}
	expected, err := checksumFor(checksums, binaryName)
	if err != nil {
		return "", err
	}
	binary, err := manager.download(ctx, release.BinaryURL, maxAssetBytes)
	if err != nil {
		return "", fmt.Errorf("download update: %w", err)
	}
	actual := sha256.Sum256(binary)
	actualHex := hex.EncodeToString(actual[:])
	if !strings.EqualFold(expected, actualHex) {
		return "", fmt.Errorf("update checksum mismatch")
	}
	if release.BinaryDigest != "" &&
		!strings.EqualFold(release.BinaryDigest, "sha256:"+actualHex) {
		return "", fmt.Errorf("update digest mismatch")
	}
	if err := os.MkdirAll(manager.stagingDir, 0o755); err != nil {
		return "", fmt.Errorf("create update cache: %w", err)
	}
	destination := filepath.Join(manager.stagingDir, "wt-modern-"+release.Version+".exe")
	temporary := destination + ".tmp"
	if err := os.WriteFile(temporary, binary, 0o755); err != nil {
		return "", fmt.Errorf("write staged update: %w", err)
	}
	_ = os.Remove(destination)
	if err := os.Rename(temporary, destination); err != nil {
		_ = os.Remove(temporary)
		return "", fmt.Errorf("finalize staged update: %w", err)
	}
	return destination, nil
}

func probeWritable(directory string) error {
	file, err := os.CreateTemp(directory, ".wt-modern-update-*")
	if err != nil {
		return err
	}
	name := file.Name()
	if err := file.Close(); err != nil {
		_ = os.Remove(name)
		return err
	}
	return os.Remove(name)
}

func (manager *Manager) download(ctx context.Context, location string, limit int64) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, location, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/octet-stream")
	request.Header.Set("User-Agent", "WT-Modern-8111/"+manager.currentVersion)
	response, err := manager.assetHTTP.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", response.StatusCode)
	}
	return readLimited(response.Body, limit)
}

func readLimited(reader io.Reader, limit int64) ([]byte, error) {
	body, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > limit {
		return nil, fmt.Errorf("response exceeds %d bytes", limit)
	}
	return body, nil
}

func checksumFor(contents []byte, name string) (string, error) {
	scanner := bufio.NewScanner(strings.NewReader(string(contents)))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) == 2 && strings.TrimPrefix(fields[1], "*") == name {
			if len(fields[0]) != sha256.Size*2 {
				return "", fmt.Errorf("invalid checksum for %s", name)
			}
			if _, err := hex.DecodeString(fields[0]); err != nil {
				return "", fmt.Errorf("invalid checksum for %s: %w", name, err)
			}
			return fields[0], nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("read checksums: %w", err)
	}
	return "", fmt.Errorf("checksum for %s not found", name)
}

func validateGitHubAssetURL(location string) error {
	parsed, err := url.Parse(location)
	if err != nil {
		return fmt.Errorf("parse release asset URL: %w", err)
	}
	if parsed.Scheme != "https" || !strings.EqualFold(parsed.Hostname(), "github.com") {
		return fmt.Errorf("release asset is not hosted by GitHub")
	}
	return nil
}

func newerVersion(current, candidate string) bool {
	currentParts, currentOK := parseVersion(current)
	candidateParts, candidateOK := parseVersion(candidate)
	if !currentOK || !candidateOK {
		return false
	}
	for index := range currentParts {
		if candidateParts[index] != currentParts[index] {
			return candidateParts[index] > currentParts[index]
		}
	}
	return false
}

func isSemanticVersion(version string) bool {
	_, ok := parseVersion(version)
	return ok
}

func parseVersion(version string) ([3]int, bool) {
	var result [3]int
	parts := strings.Split(strings.TrimPrefix(strings.TrimSpace(version), "v"), ".")
	if len(parts) != len(result) {
		return result, false
	}
	for index, part := range parts {
		value, err := strconv.Atoi(part)
		if err != nil || value < 0 {
			return result, false
		}
		result[index] = value
	}
	return result, true
}
