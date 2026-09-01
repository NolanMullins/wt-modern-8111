package polling

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/NolanMullins/wt-modern-8111/internal/identity"
	"github.com/NolanMullins/wt-modern-8111/internal/telemetry"
	"github.com/NolanMullins/wt-modern-8111/internal/warthunder"
)

const (
	fastInterval = 100 * time.Millisecond
	slowInterval = time.Second
	maxFeedItems = 50
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
	identity         *identity.Resolver
	now              func() time.Time
	returnToAirfield bool
	landingSamples   int
	allyMarks        []telemetry.AllyMark
	destroyed        bool
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
		now:      time.Now,
	}
	service.publish(service.now())
	return service
}

func NewFixtureService(directory string) (*Service, error) {
	service := &Service{
		mode:     "fixture",
		sources:  make(map[string]*sourceRecord),
		feedKeys: make(map[string]struct{}),
		identity: identity.NewResolver(""),
		now:      time.Now,
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
	now := service.now()
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
	epoch := s.currentMapEpoch()
	image, err := s.client.MapImage(ctx, generation)
	if err != nil {
		s.recordFailure("mapImage", err)
		return
	}
	s.mu.Lock()
	if epoch != s.mapEpoch {
		s.mu.Unlock()
		return
	}
	s.mapImage = append(s.mapImage[:0], image.Body...)
	s.mapImageType = image.ContentType
	s.mapGeneration = generation
	s.recordSuccessLocked("mapImage")
	s.mu.Unlock()
}

func (s *Service) publish(now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.publishLocked(now)
}

func (s *Service) publishNow() {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	s.updateRTBStateLocked()
	s.pruneAllyMarksLocked(now)
	callsign, confirmed := s.identity.Callsign()
	s.raw.Pilot = telemetry.Pilot{Callsign: callsign, Confirmed: confirmed}
	s.raw.AllyMarks = s.allyMarks
	s.raw.Destroyed = s.destroyed
	s.publishLocked(now)
}

func (s *Service) publishLocked(now time.Time) {
	s.sequence++
	s.snapshot = telemetry.BuildSnapshot(s.sequence, now, s.mode, s.raw, s.sourceStatusesLocked(now))
}
