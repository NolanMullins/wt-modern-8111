package playerteam

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDetectorTracksCurrentBattleTeamAndClear(t *testing.T) {
	root := t.TempDir()
	logs := filepath.Join(root, ".game_logs")
	if err := os.Mkdir(logs, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(logs, "game.clog")
	key := testKey()
	initial := append(bytes.Repeat([]byte{' '}, keySampleSize-1024),
		[]byte(
			"cloud_server::load_last_user_id: last id is 4304199\n"+
				`[MATCHING] < {"userId":"4304199","presences":{"in_game_ex":{"team":2}}}`+"\n",
		)...,
	)
	writeEncrypted(t, path, initial, key)

	detector := newDetector([]string{root})
	result := detector.Detect()
	if result.Status != "detected" || result.Team != 2 {
		t.Fatalf("result = %+v, want detected Team 2", result)
	}

	appendEncrypted(t, path, []byte(
		`[MATCHING] < {"userId":"4304199","presences":{"in_game_ex":null}}`+"\n",
	), int64(len(initial)), key)
	result = detector.Detect()
	if result.Status != "unknown" || result.Team != 0 {
		t.Fatalf("result after clear = %+v, want unknown", result)
	}
}

func TestDetectorReportsMissingLogs(t *testing.T) {
	result := newDetector([]string{t.TempDir()}).Detect()
	if result.Status != "unavailable" {
		t.Fatalf("result = %+v, want unavailable", result)
	}
}

func TestDetectorRetriesAfterYoungLogGrows(t *testing.T) {
	root := t.TempDir()
	logs := filepath.Join(root, ".game_logs")
	if err := os.Mkdir(logs, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(logs, "game.clog")
	key := testKey()
	writeEncrypted(t, path, bytes.Repeat([]byte{' '}, 1024), key)
	detector := newDetector([]string{root})
	if result := detector.Detect(); result.Status != "unknown" {
		t.Fatalf("young log result = %+v, want unknown", result)
	}

	body := append(bytes.Repeat([]byte{' '}, keySampleSize-1024),
		[]byte(
			"cloud_server::load_last_user_id: last id is 4304199\n"+
				`[MATCHING] < {"userId":"4304199","presences":{"in_game_ex":{"team":1}}}`+"\n",
		)...,
	)
	writeEncrypted(t, path, body, key)
	if result := detector.Detect(); result.Status != "detected" || result.Team != 1 {
		t.Fatalf("grown log result = %+v, want detected Team 1", result)
	}
}

func TestDetectorDoesNotReusePreviousLogKey(t *testing.T) {
	root := t.TempDir()
	logs := filepath.Join(root, ".game_logs")
	if err := os.Mkdir(logs, 0o755); err != nil {
		t.Fatal(err)
	}
	firstPath := filepath.Join(logs, "first.clog")
	firstKey := testKey()
	first := append(bytes.Repeat([]byte{' '}, keySampleSize-1024),
		[]byte(
			"cloud_server::load_last_user_id: last id is 4304199\n"+
				`[MATCHING] < {"userId":"4304199","presences":{"in_game_ex":{"team":1}}}`+"\n",
		)...,
	)
	writeEncrypted(t, firstPath, first, firstKey)
	detector := newDetector([]string{root})
	if result := detector.Detect(); result.Team != 1 {
		t.Fatalf("first log result = %+v", result)
	}

	secondPath := filepath.Join(logs, "second.clog")
	writeEncrypted(t, secondPath, bytes.Repeat([]byte{0x7f}, 64<<10), testKeyWithOffset())
	updated := time.Now().Add(time.Second)
	if err := os.Chtimes(secondPath, updated, updated); err != nil {
		t.Fatal(err)
	}
	result := detector.Detect()
	if result.Status != "unknown" || result.Team != 0 {
		t.Fatalf("new undecodable log reused stale team: %+v", result)
	}
	if detector.logPath == secondPath {
		t.Fatal("undecodable log was committed with stale detector state")
	}
}

func TestSteamLibraryPaths(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "steamapps")
	if err := os.Mkdir(path, 0o755); err != nil {
		t.Fatal(err)
	}
	body := []byte(`"libraryfolders" {
		"0" { "path" "C:\\Program Files (x86)\\Steam" }
		"1" { "path" "D:\\Games\\Steam" }
	}`)
	if err := os.WriteFile(filepath.Join(path, "libraryfolders.vdf"), body, 0o600); err != nil {
		t.Fatal(err)
	}
	libraries := steamLibraryPaths(root)
	if len(libraries) != 2 ||
		libraries[0] != `C:\Program Files (x86)\Steam` ||
		libraries[1] != `D:\Games\Steam` {
		t.Fatalf("libraries = %#v", libraries)
	}
}

func TestDetectsLocalGameLog(t *testing.T) {
	if os.Getenv("WT_MODERN_TEST_LOCAL_TEAM") == "" {
		t.Skip("set WT_MODERN_TEST_LOCAL_TEAM=1 to inspect locally installed War Thunder logs")
	}
	result := NewDetector().Detect()
	if result.Status == "error" || result.Status == "unavailable" {
		t.Fatalf("local detection failed: %+v", result)
	}
	t.Logf("local result: %+v", result)
}

func testKey() [xorKeyLength]byte {
	var key [xorKeyLength]byte
	for index := range key {
		key[index] = byte(index*17 + 3)
	}
	return key
}

func testKeyWithOffset() [xorKeyLength]byte {
	key := testKey()
	for index := range key {
		key[index]++
	}
	return key
}

func writeEncrypted(t *testing.T, path string, body []byte, key [xorKeyLength]byte) {
	t.Helper()
	encrypted := append([]byte(nil), body...)
	decrypt(encrypted, 0, key)
	if err := os.WriteFile(path, encrypted, 0o600); err != nil {
		t.Fatal(err)
	}
}

func appendEncrypted(
	t *testing.T,
	path string,
	body []byte,
	offset int64,
	key [xorKeyLength]byte,
) {
	t.Helper()
	encrypted := append([]byte(nil), body...)
	decrypt(encrypted, offset, key)
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if _, err := file.Write(encrypted); err != nil {
		t.Fatal(err)
	}
	updated := time.Now().Add(time.Second)
	if err := os.Chtimes(path, updated, updated); err != nil {
		t.Fatal(err)
	}
}
