package playerteam

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	xorKeyLength  = 128
	keySampleSize = 4 << 20
	initialTail   = 16 << 20
	appendOverlap = 4 << 10
	activeLogAge  = 2 * time.Minute
)

var (
	errLogNotReady      = errors.New("game log is not ready for team detection")
	userIDPattern       = regexp.MustCompile(`load_last_user_id: last id is ([0-9]+)`)
	presenceTeamPattern = regexp.MustCompile(`"in_game_ex":\{"team":([12])`)
)

type Result struct {
	Team      int       `json:"team,omitempty"`
	Status    string    `json:"status"`
	Source    string    `json:"source,omitempty"`
	UpdatedAt time.Time `json:"updatedAt,omitempty"`
	Detail    string    `json:"detail,omitempty"`
}

type Detector struct {
	mu sync.Mutex

	roots    []string
	logPath  string
	logSize  int64
	key      [xorKeyLength]byte
	userID   string
	result   Result
	lastScan time.Time
}

func NewDetector() *Detector {
	return newDetector(discoverRoots())
}

func newDetector(roots []string) *Detector {
	return &Detector{
		roots:  append([]string(nil), roots...),
		result: Result{Status: "unavailable", Detail: "War Thunder game log not found"},
	}
}

func (detector *Detector) Detect() Result {
	detector.mu.Lock()
	defer detector.mu.Unlock()

	path, info, err := detector.latestLog()
	if err != nil {
		detector.result = Result{Status: "unavailable", Detail: err.Error()}
		return detector.result
	}
	if path == detector.logPath && info.Size() == detector.logSize &&
		info.ModTime().Equal(detector.lastScan) {
		detector.expireInactiveTeam(info.ModTime())
		return detector.result
	}

	if path != detector.logPath || info.Size() < detector.logSize {
		if err := detector.startLog(path, info); err != nil {
			if errors.Is(err, errLogNotReady) {
				detector.result = Result{
					Status: "unknown",
					Source: "game-log",
					Detail: "Waiting for the current War Thunder battle log",
				}
				return detector.result
			}
			detector.result = Result{Status: "error", Source: "game-log", Detail: err.Error()}
			return detector.result
		}
	} else if err := detector.scanAppend(path, info); err != nil {
		detector.result = Result{Status: "error", Source: "game-log", Detail: err.Error()}
		return detector.result
	}
	detector.lastScan = info.ModTime()
	detector.expireInactiveTeam(info.ModTime())
	return detector.result
}

func (detector *Detector) expireInactiveTeam(logModified time.Time) {
	if detector.result.Status == "detected" && time.Since(logModified) > activeLogAge {
		detector.result = Result{
			Status: "unknown",
			Source: "game-log",
			Detail: "Waiting for an active War Thunder battle",
		}
	}
}

func (detector *Detector) startLog(path string, info os.FileInfo) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open game log: %w", err)
	}
	defer file.Close()

	sampleLength := min(info.Size(), keySampleSize)
	if sampleLength < 64<<10 {
		return errLogNotReady
	}
	sample := make([]byte, sampleLength)
	if _, err := io.ReadFull(file, sample); err != nil {
		return fmt.Errorf("read game log key sample: %w", err)
	}
	keys, err := deriveXORKeyCandidates(file, info.Size(), sample)
	if err != nil {
		return err
	}
	var (
		detectedKey [xorKeyLength]byte
		userID      string
	)
	for _, key := range keys {
		plaintext := append([]byte(nil), sample...)
		decrypt(plaintext, 0, key)
		if matchedUserID := latestMatch(plaintext, userIDPattern); matchedUserID != "" {
			detectedKey = key
			userID = matchedUserID
			break
		}
	}
	if userID == "" {
		return errLogNotReady
	}
	detector.key = detectedKey
	detector.userID = userID

	start := max(int64(0), info.Size()-initialTail)
	body := make([]byte, info.Size()-start)
	if _, err := file.ReadAt(body, start); err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("read game log: %w", err)
	}
	decrypt(body, start, detector.key)

	detector.logPath = path
	detector.logSize = info.Size()
	detector.result = Result{
		Status: "unknown",
		Source: "game-log",
		Detail: "Waiting for the current battle team assignment",
	}
	detector.scanPlaintext(body)
	return nil
}

func (detector *Detector) scanAppend(path string, info os.FileInfo) error {
	start := max(int64(0), detector.logSize-appendOverlap)
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open game log: %w", err)
	}
	defer file.Close()

	body := make([]byte, info.Size()-start)
	if _, err := file.ReadAt(body, start); err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("read game log update: %w", err)
	}
	decrypt(body, start, detector.key)
	detector.logSize = info.Size()
	detector.scanPlaintext(body)
	return nil
}

func (detector *Detector) scanPlaintext(body []byte) {
	if latestUserID := latestMatch(body, userIDPattern); latestUserID != "" &&
		latestUserID != detector.userID {
		detector.userID = latestUserID
		detector.result = Result{
			Status: "unknown",
			Source: "game-log",
			Detail: "Waiting for the current battle team assignment",
		}
	}
	if detector.userID == "" {
		return
	}

	for _, rawLine := range strings.Split(string(body), "\n") {
		line := strings.TrimSpace(rawLine)
		if !strings.Contains(line, detector.userID) {
			continue
		}
		if strings.Contains(line, `"in_game_ex":null`) {
			detector.result = Result{
				Status: "unknown",
				Source: "game-log",
				Detail: "Waiting for the current battle team assignment",
			}
			continue
		}
		if match := presenceTeamPattern.FindStringSubmatch(line); len(match) == 2 {
			detector.setTeam(match[1])
		}
	}
}

func (detector *Detector) setTeam(raw string) {
	team, err := strconv.Atoi(raw)
	if err != nil || (team != 1 && team != 2) {
		return
	}
	detector.result = Result{
		Team:      team,
		Status:    "detected",
		Source:    "game-log",
		UpdatedAt: time.Now().UTC(),
		Detail:    fmt.Sprintf("Confirmed from the active War Thunder battle (Team %d)", team),
	}
}

func (detector *Detector) latestLog() (string, os.FileInfo, error) {
	var (
		latestPath string
		latestInfo os.FileInfo
	)
	for _, root := range detector.roots {
		matches, err := filepath.Glob(filepath.Join(root, ".game_logs", "*.clog"))
		if err != nil {
			continue
		}
		for _, match := range matches {
			info, err := os.Stat(match)
			if err != nil || info.IsDir() {
				continue
			}
			if latestInfo == nil || info.ModTime().After(latestInfo.ModTime()) {
				latestPath, latestInfo = match, info
			}
		}
	}
	if latestInfo == nil {
		return "", nil, errors.New("War Thunder game log not found")
	}
	return latestPath, latestInfo, nil
}

func deriveXORKey(sample []byte) [xorKeyLength]byte {
	// CLOG uses a repeating XOR key; spaces are the most frequent plaintext byte
	// at every key position in a sufficiently mature text log.
	var frequencies [xorKeyLength][256]uint32
	addKeyFrequencies(&frequencies, sample, 0)
	return keyFromFrequencies(frequencies)
}

func deriveXORKeyCandidates(
	file *os.File,
	size int64,
	first []byte,
) ([][xorKeyLength]byte, error) {
	candidates := [][xorKeyLength]byte{deriveXORKey(first)}
	var combined [xorKeyLength][256]uint32
	starts := []int64{
		0,
		max(0, size/2-keySampleSize/2),
		max(0, size-keySampleSize),
	}
	seen := make(map[int64]struct{}, len(starts))
	for _, start := range starts {
		if _, ok := seen[start]; ok {
			continue
		}
		seen[start] = struct{}{}
		length := min(int64(keySampleSize), size-start)
		sample := make([]byte, length)
		if _, err := file.ReadAt(sample, start); err != nil && !errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("read game log key sample: %w", err)
		}
		addKeyFrequencies(&combined, sample, start)
	}
	combinedKey := keyFromFrequencies(combined)
	if combinedKey != candidates[0] {
		candidates = append(candidates, combinedKey)
	}
	return candidates, nil
}

func addKeyFrequencies(
	frequencies *[xorKeyLength][256]uint32,
	sample []byte,
	offset int64,
) {
	for index, value := range sample {
		position := (int(offset) + index) % xorKeyLength
		frequencies[position][value]++
	}
}

func keyFromFrequencies(frequencies [xorKeyLength][256]uint32) [xorKeyLength]byte {
	var key [xorKeyLength]byte
	for position := range xorKeyLength {
		var (
			bestByte  byte
			bestCount uint32
		)
		for value, count := range frequencies[position] {
			if count > bestCount {
				bestByte, bestCount = byte(value), count
			}
		}
		key[position] = bestByte ^ ' '
	}
	return key
}

func decrypt(body []byte, offset int64, key [xorKeyLength]byte) {
	for index := range body {
		body[index] ^= key[(int(offset)+index)%xorKeyLength]
	}
}

func latestMatch(body []byte, pattern *regexp.Regexp) string {
	matches := pattern.FindAllSubmatch(body, -1)
	if len(matches) == 0 {
		return ""
	}
	return string(matches[len(matches)-1][1])
}

func discoverRoots() []string {
	seen := make(map[string]struct{})
	roots := make([]string, 0, 16)
	add := func(root string) {
		root = filepath.Clean(strings.TrimSpace(root))
		if root == "." || root == "" {
			return
		}
		if _, ok := seen[strings.ToLower(root)]; ok {
			return
		}
		if info, err := os.Stat(filepath.Join(root, ".game_logs")); err == nil && info.IsDir() {
			seen[strings.ToLower(root)] = struct{}{}
			roots = append(roots, root)
		}
	}

	add(os.Getenv("WT_MODERN_WAR_THUNDER_DIR"))
	if executable, err := os.Executable(); err == nil {
		add(filepath.Dir(executable))
	}
	steamRoots := platformSteamRoots()
	if runtime.GOOS == "windows" {
		for drive := 'C'; drive <= 'Z'; drive++ {
			prefix := string(drive) + `:\`
			add(filepath.Join(prefix, "Program Files", "War Thunder"))
			add(filepath.Join(prefix, "Program Files (x86)", "War Thunder"))
			add(filepath.Join(prefix, "Games", "War Thunder"))
			steamRoots = append(
				steamRoots,
				filepath.Join(prefix, "SteamLibrary"),
				filepath.Join(prefix, "Program Files (x86)", "Steam"),
				filepath.Join(prefix, "Program Files", "Steam"),
			)
		}
	}
	for _, steamRoot := range steamRoots {
		add(filepath.Join(steamRoot, "steamapps", "common", "War Thunder"))
		for _, library := range steamLibraryPaths(steamRoot) {
			add(filepath.Join(library, "steamapps", "common", "War Thunder"))
		}
	}
	return roots
}

var steamLibraryPathPattern = regexp.MustCompile(`(?m)"path"\s+"([^"]+)"`)

func steamLibraryPaths(steamRoot string) []string {
	body, err := os.ReadFile(filepath.Join(steamRoot, "steamapps", "libraryfolders.vdf"))
	if err != nil {
		return nil
	}
	matches := steamLibraryPathPattern.FindAllSubmatch(body, -1)
	result := make([]string, 0, len(matches))
	for _, match := range matches {
		value := strings.ReplaceAll(string(match[1]), `\\`, `\`)
		if value != "" {
			result = append(result, value)
		}
	}
	return result
}
