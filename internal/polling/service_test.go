package polling

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/NolanMullins/wt-modern-8111/internal/identity"
	"github.com/NolanMullins/wt-modern-8111/internal/telemetry"
	"github.com/NolanMullins/wt-modern-8111/internal/warthunder"
)

// newTestService builds a service whose identity resolver never touches the
// user's persisted callsign file.
func newTestService() *Service {
	return NewServiceWithIdentity(nil, identity.NewResolver(""))
}

func TestHUDPrimingDiscardsHistoryThenAcceptsNewRecords(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Query().Get("lastEvt") {
		case "0":
			_, _ = fmt.Fprint(writer, `{"events":[{"id":5,"msg":"history"}],"damage":[]}`)
		case "5":
			_, _ = fmt.Fprint(writer, `{"events":[{"id":6,"msg":"new"}],"damage":[]}`)
		default:
			t.Errorf("unexpected lastEvt %q", request.URL.Query().Get("lastEvt"))
		}
	}))
	defer server.Close()

	service := NewService(warthunder.NewClient(server.URL, time.Second))
	service.pollHUD(context.Background())
	service.publishNow()
	if len(service.Snapshot().Feed) != 0 {
		t.Fatalf("priming feed length = %d, want 0", len(service.Snapshot().Feed))
	}
	service.pollHUD(context.Background())
	service.publishNow()
	feed := service.Snapshot().Feed
	if len(feed) != 1 || feed[0].Message != "new" {
		t.Fatalf("feed = %#v, want one new HUD event", feed)
	}
}

func TestRepeatedStateFailuresResetFeedSession(t *testing.T) {
	service := newTestService()
	service.lastChatID = 12
	service.lastEventID = 8
	service.lastDamageID = 4
	service.chatPrimed = true
	service.hudPrimed = true
	service.raw.Feed = []telemetry.FeedEntry{{Key: "event:8"}}
	service.feedKeys["event:8"] = struct{}{}

	for range 3 {
		service.recordFailure("state", context.DeadlineExceeded)
	}

	if service.lastChatID != 0 || service.lastEventID != 0 || service.lastDamageID != 0 {
		t.Fatalf("cursors were not reset: chat=%d event=%d damage=%d", service.lastChatID, service.lastEventID, service.lastDamageID)
	}
	if service.chatPrimed || service.hudPrimed || len(service.raw.Feed) != 0 {
		t.Fatalf("feed session was not reset: chatPrimed=%v hudPrimed=%v feed=%d", service.chatPrimed, service.hudPrimed, len(service.raw.Feed))
	}
}

func TestMapObjectResponseFromOldEpochIsDiscarded(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		close(started)
		<-release
		writer.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(writer, `[{"type":"aircraft","icon":"Player","x":0.2,"y":0.3}]`)
	}))
	defer server.Close()

	service := NewService(warthunder.NewClient(server.URL, time.Second))
	done := make(chan struct{})
	go func() {
		service.pollMapObjects(context.Background())
		close(done)
	}()
	<-started
	service.mu.Lock()
	service.resetMapSessionLocked()
	service.mu.Unlock()
	close(release)
	<-done

	service.mu.RLock()
	defer service.mu.RUnlock()
	if len(service.raw.MapObjects) != 0 {
		t.Fatalf("old epoch committed %d map objects", len(service.raw.MapObjects))
	}
}

func TestGameSessionResetInvalidatesMapImage(t *testing.T) {
	service := newTestService()
	service.mapImage = []byte("old")
	service.mapImageType = "image/png"
	service.mapGeneration = 1

	service.mu.Lock()
	service.resetGameSessionLocked()
	service.mu.Unlock()

	if _, _, _, ok := service.MapImage(); ok {
		t.Fatal("old map image remained available after session reset")
	}
}

func TestRTBActivatesOnLiveHeadingToBasePreset(t *testing.T) {
	service := newTestService()
	// The preset War Thunder actually emits is "Heading to the base."
	command := warthunder.FeedRecord{
		Message: "Heading to the base.",
		Sender:  "DEERSLUG",
		Mode:    "Team",
	}

	// The callsign is deduced from the damage feed, not from chat.
	service.identity.SetCallsign("=GRIND= DEERSLUG")

	service.processChatRecordLocked(command, false)
	if service.returnToAirfield {
		t.Fatal("backlog command must not activate navigation")
	}

	service.processChatRecordLocked(command, true)
	if !service.returnToAirfield {
		t.Fatal("pilot RTB command did not activate navigation")
	}

	service.resetRTBLocked()
	service.processChatRecordLocked(warthunder.FeedRecord{
		Message: "Heading to the base.",
		Sender:  "TEAMMATE",
		Mode:    "Team",
	}, true)
	if service.returnToAirfield {
		t.Fatal("teammate RTB command activated pilot navigation")
	}
}

func TestGuideOnMeMarksAlliesOnly(t *testing.T) {
	service := newTestService()
	service.identity.SetCallsign("=GRIND= DEERSLUG")

	service.processChatRecordLocked(warthunder.FeedRecord{
		ID:      1,
		Message: "Guide on me!",
		Sender:  "DEERSLUG",
		Mode:    "Team",
	}, true)
	if len(service.allyMarks) != 0 {
		t.Fatal("the local pilot must not be marked on their own map")
	}

	service.processChatRecordLocked(warthunder.FeedRecord{
		ID:      2,
		Message: "Guide on me!<color=#FF96966E> [C4]</color>",
		Sender:  "FISHY THUNDER",
		Mode:    "Team",
	}, true)
	if len(service.allyMarks) != 1 || service.allyMarks[0].Kind != "guide" {
		t.Fatalf("expected one ally guide mark, got %+v", service.allyMarks)
	}
	if service.allyMarks[0].Sender != "FISHY THUNDER" {
		t.Fatalf("unexpected sender %q", service.allyMarks[0].Sender)
	}
	if service.allyMarks[0].Grid != "C4" {
		t.Fatalf("expected teammate coordinate payload, got %+v", service.allyMarks[0])
	}
	if lifetime := service.allyMarks[0].ExpiresAt.Sub(service.allyMarks[0].CreatedAt); lifetime != 35*time.Second {
		t.Fatalf("ally mark lifetime = %v, want 30s visible plus 5s fade", lifetime)
	}
}

func TestAllyMarkResolvesGridReference(t *testing.T) {
	service := newTestService()
	service.identity.SetCallsign("=GRIND= DEERSLUG")
	service.raw.MapInfo = warthunder.MapInfo{
		GridSteps: []float64{13100, 13100},
		GridZero:  []float64{-65536, 65536},
		MapMin:    []float64{-65536, -65536},
		MapMax:    []float64{65536, 65536},
	}

	service.processChatRecordLocked(warthunder.FeedRecord{
		ID:      3,
		Message: "Attention to the map!<color=#FF96966E> [C4]</color>",
		Sender:  "TEAMMATE",
		Mode:    "Team",
	}, true)

	if len(service.allyMarks) != 1 {
		t.Fatalf("expected one mark, got %d", len(service.allyMarks))
	}
	mark := service.allyMarks[0]
	if mark.Grid != "C4" || !mark.Located || mark.X == nil || mark.Y == nil {
		t.Fatalf("grid not resolved: %+v", mark)
	}
	if *mark.X < 0 || *mark.X > 1 || *mark.Y < 0 || *mark.Y > 1 {
		t.Fatalf("resolved position out of range: %v %v", *mark.X, *mark.Y)
	}
	// The colour markup must not leak into the displayed message.
	if strings.Contains(mark.Message, "<color") {
		t.Fatalf("markup leaked into message: %q", mark.Message)
	}
}

func TestAllyMarksExpire(t *testing.T) {
	service := newTestService()
	service.allyMarks = []telemetry.AllyMark{
		{Key: "old", ExpiresAt: time.Now().Add(-time.Second)},
		{Key: "fresh", ExpiresAt: time.Now().Add(time.Minute)},
	}

	service.pruneAllyMarksLocked(time.Now())

	if len(service.allyMarks) != 1 || service.allyMarks[0].Key != "fresh" {
		t.Fatalf("expired marks not pruned: %+v", service.allyMarks)
	}
}

func TestRTBCommandVariantsAreRecognised(t *testing.T) {
	for _, message := range []string{
		"Heading to the base.",
		"heading to the base",
		"Returning to the base.",
		"Heading to the airfield.",
	} {
		if !isRTBCommand(message) {
			t.Errorf("expected %q to be an RTB command", message)
		}
	}
	for _, message := range []string{
		"Guide on me!",
		"Attack the D point!",
		"heading to the base of the mountain and beyond",
	} {
		if isRTBCommand(message) {
			t.Errorf("did not expect %q to be an RTB command", message)
		}
	}
}

func TestShotDownAsVictimMarksAircraftDestroyed(t *testing.T) {
	service := newTestService()
	service.identity.SetCallsign("=GRIND= DEERSLUG")
	service.returnToAirfield = true

	service.processDamageRecordLocked(warthunder.FeedRecord{
		Message: "airplaneguy24 (MiG-23M) shot down =GRIND= DEERSLUG (J-7D)",
	})

	if !service.destroyed {
		t.Fatal("shot-down victim was not marked destroyed")
	}
	if service.returnToAirfield {
		t.Fatal("RTB remained active after aircraft loss")
	}
}

func TestFuelIncreaseAfterLossStartsNewAircraft(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(writer, `{"valid":true,"Mfuel, kg":500}`)
	}))
	defer server.Close()

	service := NewServiceWithIdentity(
		warthunder.NewClient(server.URL, time.Second),
		identity.NewResolver(""),
	)
	service.destroyed = true
	service.raw.State = map[string]any{"Mfuel, kg": float64(100)}

	service.pollState(context.Background())

	if service.destroyed {
		t.Fatal("new aircraft remained marked destroyed")
	}
}

func TestRTBResetsWhenVehicleBecomesInvalid(t *testing.T) {
	service := newTestService()
	service.returnToAirfield = true
	service.raw.ReturnToAirfield = true
	service.raw.State = map[string]any{"valid": false}
	service.raw.Indicators = map[string]any{"valid": false}

	service.updateRTBStateLocked()

	if service.returnToAirfield || service.raw.ReturnToAirfield {
		t.Fatal("RTB remained active after vehicle became invalid")
	}
}

func TestRTBResetsAfterConfirmedAirfieldLanding(t *testing.T) {
	service := newTestService()
	playerX, playerY := 0.5, 0.52
	runwayStartX, runwayStartY := 0.5, 0.4
	runwayEndX, runwayEndY := 0.5, 0.6
	service.returnToAirfield = true
	service.raw.State = map[string]any{
		"valid":     true,
		"IAS, km/h": 55.0,
		"gear, %":   100.0,
		"Vy, m/s":   0.0,
	}
	service.raw.Indicators = map[string]any{
		"valid":          true,
		"radio_altitude": 0.0,
	}
	service.raw.MapInfo = warthunder.MapInfo{
		Valid:  true,
		MapMin: []float64{0, 0},
		MapMax: []float64{10000, 10000},
	}
	service.raw.MapObjects = []warthunder.MapObject{
		{Type: "aircraft", Icon: "Player", X: &playerX, Y: &playerY},
		{Type: "airfield", SX: &runwayStartX, SY: &runwayStartY, EX: &runwayEndX, EY: &runwayEndY},
	}

	for range 9 {
		service.updateRTBStateLocked()
	}
	if !service.returnToAirfield {
		t.Fatal("RTB reset before landing confirmation window")
	}
	service.updateRTBStateLocked()
	if service.returnToAirfield || service.raw.ReturnToAirfield {
		t.Fatal("RTB remained active after confirmed landing")
	}
}
