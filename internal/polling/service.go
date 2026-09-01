package polling

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/NolanMullins/wt-modern-8111/internal/identity"
	"github.com/NolanMullins/wt-modern-8111/internal/telemetry"
	"github.com/NolanMullins/wt-modern-8111/internal/warthunder"
)

const (
	fastInterval            = 100 * time.Millisecond
	slowInterval            = time.Second
	maxFeedItems            = 50
	allyMarkVisibleDuration = 30 * time.Second
	allyMarkFadeDuration    = 5 * time.Second
	allyMarkTTL             = allyMarkVisibleDuration + allyMarkFadeDuration
	maxAllyMarks            = 12
)

type Service struct {
	mu       sync.RWMutex
	client   *warthunder.Client
	mode     string
	raw      telemetry.RawData
	sources  map[string]*sourceRecord
	snapshot telemetry.Snapshot
	sequence uint64

	mapImage      []byte
	mapImageType  string
	mapGeneration int
	mapEpoch      uint64

	lastChatID       int
	lastEventID      int
	lastDamageID     int
	chatPrimed       bool
	hudPrimed        bool
	feedKeys         map[string]struct{}
	stateFailures    int
	identity         *identity.Resolver
	returnToAirfield bool
	landingSamples   int
	allyMarks        []telemetry.AllyMark
	destroyed        bool
}

type sourceRecord struct {
	lastSuccess time.Time
	lastError   error
}

func NewService(client *warthunder.Client) *Service {
	return NewServiceWithIdentity(client, identity.NewResolver(identity.DefaultStorePath()))
}

// NewServiceWithIdentity allows injecting a callsign resolver, primarily so
// tests can avoid touching the user's persisted identity file.
func NewServiceWithIdentity(client *warthunder.Client, resolver *identity.Resolver) *Service {
	service := &Service{
		client:   client,
		mode:     "live",
		sources:  make(map[string]*sourceRecord),
		feedKeys: make(map[string]struct{}),
		identity: resolver,
	}
	service.publish(time.Now())
	return service
}

func NewFixtureService(directory string) (*Service, error) {
	service := &Service{
		mode:     "fixture",
		sources:  make(map[string]*sourceRecord),
		feedKeys: make(map[string]struct{}),
		identity: identity.NewResolver(""),
	}
	files := []struct {
		name   string
		target any
		source string
	}{
		{"state.json", &service.raw.State, "state"},
		{"indicators.json", &service.raw.Indicators, "indicators"},
		{"map-info.json", &service.raw.MapInfo, "mapInfo"},
		{"map-objects.json", &service.raw.MapObjects, "mapObjects"},
		{"mission.json", &service.raw.Mission, "mission"},
	}
	now := time.Now()
	for _, file := range files {
		path := filepath.Join(directory, file.name)
		body, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read fixture %s: %w", path, err)
		}
		if err := json.Unmarshal(body, file.target); err != nil {
			return nil, fmt.Errorf("decode fixture %s: %w", path, err)
		}
		service.sources[file.source] = &sourceRecord{lastSuccess: now}
	}
	service.publish(now)
	return service, nil
}

func (s *Service) Start(ctx context.Context) {
	if s.client == nil {
		return
	}
	go s.run(ctx)
}

func (s *Service) Snapshot() telemetry.Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.snapshot
}

func (s *Service) MapImage() ([]byte, string, int, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.mapImage) == 0 {
		return nil, "", 0, false
	}
	return append([]byte(nil), s.mapImage...), s.mapImageType, s.mapGeneration, true
}

func (s *Service) run(ctx context.Context) {
	workers := []struct {
		interval time.Duration
		poll     func(context.Context)
	}{
		{fastInterval, s.pollState},
		{fastInterval, s.pollIndicators},
		{fastInterval, s.pollMapObjects},
		{slowInterval, s.pollMapInfo},
		{slowInterval, s.pollMission},
		{slowInterval, s.pollChat},
		{slowInterval, s.pollHUD},
		{fastInterval, func(context.Context) { s.publishNow() }},
	}
	var wait sync.WaitGroup
	for _, worker := range workers {
		wait.Add(1)
		go func() {
			defer wait.Done()
			s.runWorker(ctx, worker.interval, worker.poll)
		}()
	}
	<-ctx.Done()
	wait.Wait()
}

func (s *Service) runWorker(ctx context.Context, interval time.Duration, poll func(context.Context)) {
	poll(ctx)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			poll(ctx)
		}
	}
}

func (s *Service) pollState(ctx context.Context) {
	if value, err := s.client.State(ctx); err != nil {
		s.recordFailure("state", err)
	} else {
		s.mu.Lock()
		previousFuel, previousFuelOK := mapNumber(s.raw.State, "Mfuel, kg")
		currentFuel, currentFuelOK := mapNumber(value, "Mfuel, kg")
		if s.destroyed && previousFuelOK && currentFuelOK && currentFuel > previousFuel+1 {
			// A fuel increase after a loss means a new aircraft has spawned.
			s.destroyed = false
		}
		s.raw.State = value
		s.recordSuccessLocked("state")
		s.mu.Unlock()
	}
}

func (s *Service) pollIndicators(ctx context.Context) {
	if value, err := s.client.Indicators(ctx); err != nil {
		s.recordFailure("indicators", err)
	} else {
		s.mu.Lock()
		previousType, _ := s.raw.Indicators["type"].(string)
		currentType, _ := value["type"].(string)
		if previousType != "" && currentType != "" && previousType != currentType {
			s.resetRTBLocked()
			s.destroyed = false
		}
		// A fresh valid airframe means the pilot has respawned.
		if s.destroyed && boolValue(value["valid"]) && !boolValue(s.raw.Indicators["valid"]) {
			s.destroyed = false
		}
		s.raw.Indicators = value
		s.recordSuccessLocked("indicators")
		s.mu.Unlock()
	}
}

func (s *Service) pollMapObjects(ctx context.Context) {
	epoch := s.currentMapEpoch()
	if value, err := s.client.MapObjects(ctx); err != nil {
		s.recordFailure("mapObjects", err)
	} else {
		s.mu.Lock()
		if epoch != s.mapEpoch {
			s.mu.Unlock()
			return
		}
		s.raw.MapObjects = value
		s.recordSuccessLocked("mapObjects")
		s.mu.Unlock()
	}
}

func (s *Service) pollMapInfo(ctx context.Context) {
	if value, err := s.client.MapInfo(ctx); err != nil {
		s.recordFailure("mapInfo", err)
	} else {
		s.mu.Lock()
		previousGeneration := s.raw.MapInfo.Generation
		if value.Generation != previousGeneration {
			s.resetMapSessionLocked()
		}
		s.raw.MapInfo = value
		s.recordSuccessLocked("mapInfo")
		s.mu.Unlock()
		if value.Valid && value.Generation != s.currentMapGeneration() {
			s.pollMapImage(ctx, value.Generation)
		}
	}
}

func (s *Service) pollMission(ctx context.Context) {
	if value, err := s.client.Mission(ctx); err != nil {
		s.recordFailure("mission", err)
	} else {
		s.mu.Lock()
		s.raw.Mission = value
		s.recordSuccessLocked("mission")
		s.mu.Unlock()
	}
}

func (s *Service) pollChat(ctx context.Context) {
	if records, err := s.client.GameChat(ctx, s.chatCursor()); err != nil {
		s.recordFailure("chat", err)
	} else {
		s.mu.Lock()
		wasPrimed := s.chatPrimed
		for _, record := range records {
			s.processChatRecordLocked(record, wasPrimed)
		}
		if s.chatPrimed {
			s.appendFeedLocked("chat", records)
		}
		for _, record := range records {
			if record.ID > s.lastChatID {
				s.lastChatID = record.ID
			}
		}
		s.chatPrimed = true
		s.recordSuccessLocked("chat")
		s.mu.Unlock()
	}
}

func (s *Service) pollHUD(ctx context.Context) {
	eventCursor, damageCursor := s.hudCursors()
	if messages, err := s.client.HUDMessages(ctx, eventCursor, damageCursor); err != nil {
		s.recordFailure("hud", err)
	} else {
		s.mu.Lock()
		vehicleType, _ := s.raw.Indicators["type"].(string)
		for _, record := range messages.Damage {
			// Identity deduction must also consume the backlog, since older
			// damage lines are the richest source of callsign evidence.
			s.identity.Observe(record.Message, vehicleType)
		}
		if s.hudPrimed {
			for _, record := range messages.Damage {
				s.processDamageRecordLocked(record)
			}
			s.appendFeedLocked("event", messages.Events)
			s.appendFeedLocked("damage", messages.Damage)
		}
		for _, record := range messages.Events {
			if record.ID > s.lastEventID {
				s.lastEventID = record.ID
			}
		}
		for _, record := range messages.Damage {
			if record.ID > s.lastDamageID {
				s.lastDamageID = record.ID
			}
		}
		s.hudPrimed = true
		s.recordSuccessLocked("hud")
		s.mu.Unlock()
	}
}

func (s *Service) pollMapImage(ctx context.Context, generation int) {
	image, err := s.client.MapImage(ctx, generation)
	if err != nil {
		s.recordFailure("mapImage", err)
		return
	}
	s.mu.Lock()
	s.mapImage = append(s.mapImage[:0], image.Body...)
	s.mapImageType = image.ContentType
	s.mapGeneration = generation
	s.recordSuccessLocked("mapImage")
	s.mu.Unlock()
}

func (s *Service) appendFeedLocked(kind string, records []warthunder.FeedRecord) {
	now := time.Now()
	for _, record := range records {
		key := fmt.Sprintf("%s:%d", kind, record.ID)
		if _, exists := s.feedKeys[key]; exists {
			continue
		}
		s.feedKeys[key] = struct{}{}
		s.raw.Feed = append(s.raw.Feed, telemetry.FeedEntry{
			Key:     key,
			Kind:    kind,
			Time:    record.Time,
			AddedAt: now,
			Sender:  record.Sender,
			Message: record.Message,
			Enemy:   record.Enemy,
		})
	}
	sort.SliceStable(s.raw.Feed, func(i, j int) bool {
		if s.raw.Feed[i].Time != s.raw.Feed[j].Time {
			return s.raw.Feed[i].Time > s.raw.Feed[j].Time
		}
		return s.raw.Feed[i].AddedAt.After(s.raw.Feed[j].AddedAt)
	})
	if len(s.raw.Feed) > maxFeedItems {
		removed := s.raw.Feed[maxFeedItems:]
		s.raw.Feed = s.raw.Feed[:maxFeedItems]
		for _, entry := range removed {
			delete(s.feedKeys, entry.Key)
		}
	}
}

func (s *Service) recordFailure(name string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record := s.sourceLocked(name)
	record.lastError = err
	if name == "state" {
		s.stateFailures++
		if s.stateFailures == 3 {
			s.resetGameSessionLocked()
		}
	}
}

func (s *Service) recordSuccessLocked(name string) {
	record := s.sourceLocked(name)
	record.lastSuccess = time.Now()
	record.lastError = nil
	if name == "state" {
		s.stateFailures = 0
	}
}

func (s *Service) sourceLocked(name string) *sourceRecord {
	record, ok := s.sources[name]
	if !ok {
		record = &sourceRecord{}
		s.sources[name] = record
	}
	return record
}

func (s *Service) publish(now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.publishLocked(now)
}

func (s *Service) publishNow() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.updateRTBStateLocked()
	s.pruneAllyMarksLocked(time.Now())
	callsign, confirmed := s.identity.Callsign()
	s.raw.Pilot = telemetry.Pilot{Callsign: callsign, Confirmed: confirmed}
	s.raw.AllyMarks = s.allyMarks
	s.raw.Destroyed = s.destroyed
	s.publishLocked(time.Now())
}

func (s *Service) publishLocked(now time.Time) {
	s.sequence++
	s.snapshot = telemetry.BuildSnapshot(s.sequence, now, s.mode, s.raw, s.sourceStatusesLocked(now))
}

func (s *Service) sourceStatusesLocked(now time.Time) map[string]telemetry.SourceStatus {
	statuses := make(map[string]telemetry.SourceStatus, len(s.sources))
	for name, source := range s.sources {
		status := telemetry.SourceStatus{State: "unavailable"}
		if !source.lastSuccess.IsZero() {
			lastSuccess := source.lastSuccess
			age := now.Sub(lastSuccess).Milliseconds()
			status.LastSuccess = &lastSuccess
			status.AgeMS = &age
			status.State = "fresh"
			if name != "mapImage" && age > 2*slowInterval.Milliseconds() {
				status.State = "stale"
			}
		}
		if source.lastError != nil {
			status.State = "error"
			status.Error = source.lastError.Error()
		}
		statuses[name] = status
	}
	return statuses
}

func (s *Service) currentMapGeneration() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.mapGeneration
}

func (s *Service) currentMapEpoch() uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.mapEpoch
}

func (s *Service) chatCursor() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastChatID
}

func (s *Service) hudCursors() (int, int) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastEventID, s.lastDamageID
}

func (s *Service) resetFeedSessionLocked() {
	s.lastChatID = 0
	s.lastEventID = 0
	s.lastDamageID = 0
	s.chatPrimed = false
	s.hudPrimed = false
	s.raw.Feed = make([]telemetry.FeedEntry, 0)
	s.feedKeys = make(map[string]struct{})
}

func (s *Service) resetMapSessionLocked() {
	s.mapEpoch++
	s.raw.MapObjects = make([]warthunder.MapObject, 0)
	s.sources["mapObjects"] = &sourceRecord{}
	s.resetFeedSessionLocked()
	s.resetRTBLocked()
	s.allyMarks = nil
	s.destroyed = false
	if s.identity != nil {
		s.identity.ResetSession()
	}
}

func (s *Service) resetGameSessionLocked() {
	s.resetMapSessionLocked()
	s.mapImage = nil
	s.mapImageType = ""
	s.mapGeneration = 0
}

// processChatRecordLocked interprets team radio presets. Ownership is resolved
// via the deduced pilot callsign rather than by assuming the first sender seen
// is the local player.
func (s *Service) processChatRecordLocked(record warthunder.FeedRecord, activate bool) {
	if record.Enemy || record.Sender == "" || !strings.EqualFold(record.Mode, "Team") {
		return
	}
	message := stripMarkup(record.Message)
	mine := s.identity.Matches(record.Sender)

	if isRTBCommand(message) {
		if activate && mine {
			s.returnToAirfield = true
			s.landingSamples = 0
		}
		return
	}

	if !activate || mine {
		// Ally marks describe teammates only; never mark the local aircraft,
		// and never replay backlog messages from before startup.
		return
	}
	if kind, ok := allyMarkKind(message); ok {
		s.addAllyMarkLocked(kind, record)
	}
}

// addAllyMarkLocked records a teammate callout, resolving a map position from
// the grid reference War Thunder appends to received radio messages. The local
// pilot's copy does not contain this payload, which is why local calls are
// filtered before reaching this function.
func (s *Service) addAllyMarkLocked(kind string, record warthunder.FeedRecord) {
	now := time.Now()
	mark := telemetry.AllyMark{
		Key:       fmt.Sprintf("chat-%d", record.ID),
		Kind:      kind,
		Sender:    record.Sender,
		Message:   stripMarkup(record.Message),
		CreatedAt: now,
		ExpiresAt: now.Add(allyMarkTTL),
	}
	if grid, ok := extractGrid(record.Message); ok {
		mark.Grid = grid
		if x, y, ok := gridToNormalized(grid, s.raw.MapInfo); ok {
			mark.X = &x
			mark.Y = &y
			mark.Located = true
		}
	}
	s.allyMarks = append(s.allyMarks, mark)
	s.pruneAllyMarksLocked(now)
}

func (s *Service) pruneAllyMarksLocked(now time.Time) {
	kept := s.allyMarks[:0]
	for _, mark := range s.allyMarks {
		if mark.ExpiresAt.After(now) {
			kept = append(kept, mark)
		}
	}
	s.allyMarks = append(make([]telemetry.AllyMark, 0, len(kept)), kept...)
	if len(s.allyMarks) > maxAllyMarks {
		s.allyMarks = s.allyMarks[len(s.allyMarks)-maxAllyMarks:]
	}
}

func (s *Service) processDamageRecordLocked(record warthunder.FeedRecord) {
	vehicleType, _ := s.raw.Indicators["type"].(string)
	s.identity.Observe(record.Message, vehicleType)

	if !s.identity.IsLocalPilotLoss(record.Message) {
		return
	}
	s.destroyed = true
	s.resetRTBLocked()
}

func (s *Service) updateRTBStateLocked() {
	if !s.returnToAirfield {
		s.raw.ReturnToAirfield = false
		return
	}
	if !mapBool(s.raw.State, "valid") && !mapBool(s.raw.Indicators, "valid") {
		s.resetRTBLocked()
		s.raw.ReturnToAirfield = false
		return
	}
	if landedAtAirfield(s.raw) {
		s.landingSamples++
		if s.landingSamples >= 10 {
			s.resetRTBLocked()
		}
	} else {
		s.landingSamples = 0
	}
	s.raw.ReturnToAirfield = s.returnToAirfield
}

func (s *Service) resetRTBLocked() {
	s.returnToAirfield = false
	s.landingSamples = 0
	s.raw.ReturnToAirfield = false
}

// isRTBCommand matches the War Thunder radio presets that announce a return to
// base. The live preset text is "Heading to the base."; the other spellings are
// accepted defensively across localisations and vehicle types.
func isRTBCommand(message string) bool {
	normalized := normalizeCommand(message)
	switch normalized {
	case "heading to the base",
		"heading to base",
		"heading to the airfield",
		"heading back to the base",
		"returning to the base",
		"returning to base",
		"returning to the airfield",
		"returning to airfield",
		"going back to the base",
		"going home":
		return true
	}
	return false
}

// allyMarkKind classifies teammate callouts that are worth putting on the map.
func allyMarkKind(message string) (string, bool) {
	normalized := normalizeCommand(message)
	switch {
	case strings.HasPrefix(normalized, "guide on me"),
		strings.HasPrefix(normalized, "follow me"):
		return "guide", true
	case strings.HasPrefix(normalized, "attention to the map"),
		strings.HasPrefix(normalized, "attention to the designated grid zone"):
		return "attention", true
	case strings.HasPrefix(normalized, "cover me"):
		return "cover", true
	case strings.HasPrefix(normalized, "need help"),
		strings.HasPrefix(normalized, "help me"):
		return "help", true
	}
	return "", false
}

func normalizeCommand(message string) string {
	return strings.Trim(strings.ToLower(strings.TrimSpace(stripMarkup(message))), " .!?")
}

var markupPattern = regexp.MustCompile(`<[^>]*>`)

// stripMarkup removes the colour tags War Thunder embeds in some presets, such
// as "Attention to the map!<color=#FF96966E> [C4]</color>".
func stripMarkup(message string) string {
	return strings.TrimSpace(markupPattern.ReplaceAllString(message, ""))
}

var gridPattern = regexp.MustCompile(`\[([A-Za-z]{1,2})(\d{1,2})\]`)

// extractGrid pulls a grid reference such as "C4" out of a preset message.
func extractGrid(message string) (string, bool) {
	match := gridPattern.FindStringSubmatch(message)
	if match == nil {
		return "", false
	}
	return strings.ToUpper(match[1]) + match[2], true
}

// gridToNormalized converts a grid label into normalized map coordinates using
// the grid geometry reported by /map_info.json. Columns are letters running
// east, rows are numbers running south, matching the in-game map.
func gridToNormalized(grid string, info warthunder.MapInfo) (float64, float64, bool) {
	match := gridPattern.FindStringSubmatch("[" + grid + "]")
	if match == nil {
		return 0, 0, false
	}
	if len(info.GridSteps) < 2 || len(info.GridZero) < 2 ||
		len(info.MapMin) < 2 || len(info.MapMax) < 2 {
		return 0, 0, false
	}
	stepX, stepY := info.GridSteps[0], info.GridSteps[1]
	spanX := info.MapMax[0] - info.MapMin[0]
	spanY := info.MapMax[1] - info.MapMin[1]
	if stepX == 0 || stepY == 0 || spanX == 0 || spanY == 0 {
		return 0, 0, false
	}

	column := 0
	for _, symbol := range strings.ToUpper(match[1]) {
		column = column*26 + int(symbol-'A') + 1
	}
	column--
	row, err := strconv.Atoi(match[2])
	if err != nil || row < 1 {
		return 0, 0, false
	}
	row--

	// Aim at the centre of the cell.
	worldX := info.GridZero[0] + (float64(column)+0.5)*stepX
	worldY := info.GridZero[1] - (float64(row)+0.5)*math.Abs(stepY)

	x := (worldX - info.MapMin[0]) / spanX
	y := (info.MapMax[1] - worldY) / spanY
	if x < 0 || x > 1 || y < 0 || y > 1 {
		return 0, 0, false
	}
	return x, y, true
}

func landedAtAirfield(raw telemetry.RawData) bool {
	ias, iasOK := mapNumber(raw.State, "IAS, km/h")
	gear, gearOK := mapNumber(raw.State, "gear, %")
	if !iasOK || !gearOK || gear < 90 || ias > 160 {
		return false
	}
	radioAltitude, radioOK := mapNumber(raw.Indicators, "radio_altitude")
	verticalSpeed, verticalOK := mapNumber(raw.State, "Vy, m/s")
	if radioOK {
		if radioAltitude > 10 || (verticalOK && math.Abs(verticalSpeed) > 5) {
			return false
		}
	} else if ias > 60 {
		return false
	}

	if len(raw.MapInfo.MapMin) < 2 || len(raw.MapInfo.MapMax) < 2 {
		return false
	}
	var player *warthunder.MapObject
	for index := range raw.MapObjects {
		if raw.MapObjects[index].Icon == "Player" {
			player = &raw.MapObjects[index]
			break
		}
	}
	if player == nil || player.X == nil || player.Y == nil {
		return false
	}
	mapWidth := raw.MapInfo.MapMax[0] - raw.MapInfo.MapMin[0]
	mapHeight := raw.MapInfo.MapMax[1] - raw.MapInfo.MapMin[1]
	if mapWidth <= 0 || mapHeight <= 0 {
		return false
	}
	playerX := (*player.X)*mapWidth + raw.MapInfo.MapMin[0]
	playerY := (*player.Y)*mapHeight + raw.MapInfo.MapMin[1]
	for _, object := range raw.MapObjects {
		if object.Type != "airfield" || object.SX == nil || object.SY == nil || object.EX == nil || object.EY == nil {
			continue
		}
		startX := (*object.SX)*mapWidth + raw.MapInfo.MapMin[0]
		startY := (*object.SY)*mapHeight + raw.MapInfo.MapMin[1]
		endX := (*object.EX)*mapWidth + raw.MapInfo.MapMin[0]
		endY := (*object.EY)*mapHeight + raw.MapInfo.MapMin[1]
		if pointSegmentDistance(playerX, playerY, startX, startY, endX, endY) <= 1500 {
			return true
		}
	}
	return false
}

func pointSegmentDistance(px, py, startX, startY, endX, endY float64) float64 {
	dx := endX - startX
	dy := endY - startY
	if dx == 0 && dy == 0 {
		return math.Hypot(px-startX, py-startY)
	}
	t := ((px-startX)*dx + (py-startY)*dy) / (dx*dx + dy*dy)
	t = max(0, min(1, t))
	return math.Hypot(px-(startX+t*dx), py-(startY+t*dy))
}

func mapNumber(values map[string]any, key string) (float64, bool) {
	value, ok := values[key].(float64)
	return value, ok && !math.IsNaN(value) && !math.IsInf(value, 0)
}

func mapBool(values map[string]any, key string) bool {
	value, _ := values[key].(bool)
	return value
}

func boolValue(raw any) bool {
	value, _ := raw.(bool)
	return value
}
