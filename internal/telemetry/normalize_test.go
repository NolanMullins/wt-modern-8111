package telemetry

import (
	"math"
	"testing"
	"time"

	"github.com/NolanMullins/wt-modern-8111/internal/warthunder"
)

func TestBuildSnapshotNormalizesFixtureWithoutAutomaticNavigation(t *testing.T) {
	playerX, playerY := 0.360958, 0.353662
	playerDX, playerDY := -0.002011, -0.999998
	targetX, targetY := 0.427345, 0.50683
	raw := RawData{
		State: map[string]any{
			"valid":         true,
			"IAS, km/h":     681.0,
			"TAS, km/h":     861.0,
			"H, m":          5471.0,
			"Mfuel, kg":     4946.0,
			"Mfuel0, kg":    9912.0,
			"throttle 1, %": 100.0,
			"RPM 1":         12375.0,
			"oil temp 1, C": 69.0,
			"thrust 1, kgs": 2892.0,
			"throttle 2, %": 100.0,
			"RPM 2":         12375.0,
			"oil temp 2, C": 69.0,
			"thrust 2, kgs": 2892.0,
		},
		Indicators: map[string]any{
			"valid":   true,
			"army":    "air",
			"type":    "jh_7",
			"compass": 359.886078,
		},
		MapInfo: warthunder.MapInfo{
			Valid:  true,
			MapMin: []float64{-32768, -32768},
			MapMax: []float64{32768, 32768},
		},
		MapObjects: []warthunder.MapObject{
			{Type: "aircraft", Icon: "Player", X: &playerX, Y: &playerY, DX: &playerDX, DY: &playerDY},
			{Type: "bombing_point", Icon: "bombing_point", X: &targetX, Y: &targetY},
		},
	}
	now := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	fresh := now
	zero := int64(0)
	sources := map[string]SourceStatus{
		"state":      {State: "fresh", LastSuccess: &fresh, AgeMS: &zero},
		"indicators": {State: "fresh", LastSuccess: &fresh, AgeMS: &zero},
		"mapInfo":    {State: "fresh", LastSuccess: &fresh, AgeMS: &zero},
		"mapObjects": {State: "fresh", LastSuccess: &fresh, AgeMS: &zero},
	}

	snapshot := BuildSnapshot(7, now, "fixture", raw, sources)

	if snapshot.Connection.State != "live" {
		t.Fatalf("connection state = %q, want live", snapshot.Connection.State)
	}
	if snapshot.Vehicle.Type != "jh_7" {
		t.Fatalf("vehicle type = %q, want jh_7", snapshot.Vehicle.Type)
	}
	if len(snapshot.Systems.Engines) != 2 {
		t.Fatalf("engine count = %d, want 2", len(snapshot.Systems.Engines))
	}
	if snapshot.Systems.FuelPercent == nil || math.Abs(*snapshot.Systems.FuelPercent-49.8991) > 0.01 {
		t.Fatalf("fuel percent = %v, want about 49.9", snapshot.Systems.FuelPercent)
	}
	if snapshot.Navigation != nil {
		t.Fatalf("navigation = %#v, want explicit user selection", snapshot.Navigation)
	}
}

func TestBuildSnapshotMarksCriticalSourceFailureDegraded(t *testing.T) {
	now := time.Now()
	sources := map[string]SourceStatus{
		"state":      {State: "fresh"},
		"indicators": {State: "fresh"},
		"mapObjects": {State: "error", Error: "timeout"},
	}

	raw := RawData{
		State:      map[string]any{"valid": true},
		Indicators: map[string]any{"valid": true},
	}
	snapshot := BuildSnapshot(1, now, "live", raw, sources)
	if snapshot.Connection.State != "degraded" {
		t.Fatalf("connection state = %q, want degraded", snapshot.Connection.State)
	}
}

func TestBuildSnapshotRTBSelectsNearestFriendlyAirfield(t *testing.T) {
	playerX, playerY := 0.25, 0.25
	runwayStartX, runwayStartY := 0.5, 0.5
	runwayEndX, runwayEndY := 0.5, 0.6
	enemyStartX, enemyStartY := 0.3, 0.3
	enemyEndX, enemyEndY := 0.3, 0.4
	strikeX, strikeY := 0.3, 0.3
	raw := RawData{
		State:      map[string]any{"valid": true, "TAS, km/h": 360.0},
		Indicators: map[string]any{"valid": true},
		MapInfo: warthunder.MapInfo{
			Valid:  true,
			MapMin: []float64{0, 0},
			MapMax: []float64{10000, 10000},
		},
		MapObjects: []warthunder.MapObject{
			{Type: "aircraft", Icon: "Player", X: &playerX, Y: &playerY},
			{Type: "bombing_point", X: &strikeX, Y: &strikeY},
			{
				Type: "airfield", Color: "#fa0C00",
				SX: &enemyStartX, SY: &enemyStartY, EX: &enemyEndX, EY: &enemyEndY,
			},
			{
				Type: "airfield", Color: "#174DFF",
				SX: &runwayStartX, SY: &runwayStartY, EX: &runwayEndX, EY: &runwayEndY,
			},
		},
		ReturnToAirfield: true,
	}

	sources := map[string]SourceStatus{
		"state":      {State: "fresh"},
		"indicators": {State: "fresh"},
		"mapInfo":    {State: "fresh"},
		"mapObjects": {State: "fresh"},
	}
	snapshot := BuildSnapshot(1, time.Now(), "live", raw, sources)
	if snapshot.Navigation == nil {
		t.Fatal("navigation is nil")
	}
	if snapshot.Navigation.Name != "Nearest airfield" {
		t.Fatalf("navigation name = %q, want Nearest airfield", snapshot.Navigation.Name)
	}
	if snapshot.Navigation.TargetX != 0.5 || snapshot.Navigation.TargetY != 0.55 {
		t.Fatalf("navigation target = %.2f, %.2f", snapshot.Navigation.TargetX, snapshot.Navigation.TargetY)
	}
}

func TestBuildSnapshotRTBDoesNotSelectEnemyAirfield(t *testing.T) {
	playerX, playerY := 0.25, 0.25
	startX, startY := 0.3, 0.3
	endX, endY := 0.3, 0.4
	raw := RawData{
		State:      map[string]any{"valid": true, "TAS, km/h": 360.0},
		Indicators: map[string]any{"valid": true},
		MapInfo: warthunder.MapInfo{
			Valid:  true,
			MapMin: []float64{0, 0},
			MapMax: []float64{10000, 10000},
		},
		MapObjects: []warthunder.MapObject{
			{Type: "aircraft", Icon: "Player", X: &playerX, Y: &playerY},
			{
				Type: "airfield", Color: "#fa0C00",
				SX: &startX, SY: &startY, EX: &endX, EY: &endY,
			},
		},
		ReturnToAirfield: true,
	}
	sources := map[string]SourceStatus{
		"state":      {State: "fresh"},
		"indicators": {State: "fresh"},
		"mapInfo":    {State: "fresh"},
		"mapObjects": {State: "fresh"},
	}

	snapshot := BuildSnapshot(1, time.Now(), "live", raw, sources)

	if snapshot.Navigation != nil {
		t.Fatalf("enemy airfield selected for RTB: %#v", snapshot.Navigation)
	}
}

func TestBuildSnapshotDoesNotSelectAirfieldWithoutRTBCommand(t *testing.T) {
	playerX, playerY := 0.25, 0.25
	runwayStartX, runwayStartY := 0.5, 0.5
	runwayEndX, runwayEndY := 0.5, 0.6
	raw := RawData{
		State:      map[string]any{"valid": true, "TAS, km/h": 360.0},
		Indicators: map[string]any{"valid": true},
		MapInfo: warthunder.MapInfo{
			Valid:  true,
			MapMin: []float64{0, 0},
			MapMax: []float64{10000, 10000},
		},
		MapObjects: []warthunder.MapObject{
			{Type: "aircraft", Icon: "Player", X: &playerX, Y: &playerY},
			{Type: "airfield", SX: &runwayStartX, SY: &runwayStartY, EX: &runwayEndX, EY: &runwayEndY},
		},
	}
	sources := map[string]SourceStatus{
		"state":      {State: "fresh"},
		"indicators": {State: "fresh"},
		"mapInfo":    {State: "fresh"},
		"mapObjects": {State: "fresh"},
	}
	snapshot := BuildSnapshot(1, time.Now(), "live", raw, sources)
	if snapshot.Navigation != nil {
		t.Fatalf("navigation = %#v, want nil before RTB command", snapshot.Navigation)
	}
}

func TestBuildSnapshotExpiresStaleRawTelemetry(t *testing.T) {
	raw := RawData{
		State:      map[string]any{"valid": true, "IAS, km/h": 700.0},
		Indicators: map[string]any{"valid": true, "type": "stale_aircraft"},
		MapObjects: []warthunder.MapObject{{Type: "aircraft", Icon: "Player"}},
	}
	sources := map[string]SourceStatus{
		"state":      {State: "error", Error: "connection refused"},
		"indicators": {State: "error", Error: "connection refused"},
		"mapObjects": {State: "error", Error: "connection refused"},
	}

	snapshot := BuildSnapshot(1, time.Now(), "live", raw, sources)
	if snapshot.Connection.State != "waiting-for-game" {
		t.Fatalf("connection state = %q, want waiting-for-game", snapshot.Connection.State)
	}
	if snapshot.Flight.IASKMH != nil || snapshot.Vehicle.Type != "" {
		t.Fatalf("stale telemetry leaked into snapshot: IAS=%v vehicle=%q", snapshot.Flight.IASKMH, snapshot.Vehicle.Type)
	}
	if snapshot.Map.Objects == nil || len(snapshot.Map.Objects) != 0 {
		t.Fatalf("map objects = %#v, want non-nil empty slice", snapshot.Map.Objects)
	}
	if snapshot.Systems.Engines == nil {
		t.Fatal("engines serialized as nil")
	}
}

func TestBuildSnapshotDoesNotNavigateInvalidMap(t *testing.T) {
	playerX, playerY := 0.25, 0.25
	targetX, targetY := 0.5, 0.5
	raw := RawData{
		State: map[string]any{"valid": true},
		MapInfo: warthunder.MapInfo{
			Valid:  false,
			MapMin: []float64{0, 0},
			MapMax: []float64{10000, 10000},
		},
		MapObjects: []warthunder.MapObject{
			{Type: "aircraft", Icon: "Player", X: &playerX, Y: &playerY},
			{Type: "bombing_point", X: &targetX, Y: &targetY},
		},
	}
	sources := map[string]SourceStatus{
		"state":      {State: "fresh"},
		"mapInfo":    {State: "fresh"},
		"mapObjects": {State: "fresh"},
	}
	snapshot := BuildSnapshot(1, time.Now(), "live", raw, sources)
	if snapshot.Navigation != nil {
		t.Fatalf("navigation = %#v, want nil for invalid map", snapshot.Navigation)
	}
}

func TestBuildSystemsReportsEngineOutWhenThrottleCommandedButNoOutput(t *testing.T) {
	state := map[string]any{
		"throttle 1, %": float64(100),
		"RPM 1":         float64(0),
		"thrust 1, kgs": float64(0),
		"power 1, hp":   float64(0),
		"oil temp 1, C": float64(74),
	}

	systems := buildSystems(state, nil, false)

	if systems.Status != "Engine out" || systems.Severity != "critical" {
		t.Fatalf("got status=%q severity=%q", systems.Status, systems.Severity)
	}
	if len(systems.Engines) != 1 || systems.Engines[0].Status != engineStatusFailed {
		t.Fatalf("engine not marked failed: %+v", systems.Engines)
	}
}

func TestBuildSystemsReportsNominalForHealthyJet(t *testing.T) {
	// Live J-7D sample: jets report 0 hp but positive thrust and RPM.
	state := map[string]any{
		"throttle 1, %": float64(100),
		"RPM 1":         float64(11385),
		"thrust 1, kgs": float64(3733),
		"power 1, hp":   float64(0),
	}

	systems := buildSystems(state, nil, false)

	if systems.Status != "Nominal" || systems.Severity != "good" {
		t.Fatalf("healthy jet reported as %q/%q", systems.Status, systems.Severity)
	}
}

func TestBuildSystemsPrefersHighResolutionIndicatorFuel(t *testing.T) {
	state := map[string]any{
		"Mfuel, kg":     float64(2296),
		"Mfuel0, kg":    float64(3484),
		"throttle 1, %": float64(100),
		"RPM 1":         float64(11385),
	}
	indicators := map[string]any{"fuel": float64(2295.42)}

	systems := buildSystems(state, indicators, false)

	if systems.FuelKG == nil || *systems.FuelKG != 2295.42 {
		t.Fatalf("fuel = %v, want high-resolution indicator value", systems.FuelKG)
	}
}

func TestBuildSystemsTreatsShutdownEngineAsIdleNotFailed(t *testing.T) {
	state := map[string]any{
		"throttle 1, %": float64(0),
		"RPM 1":         float64(0),
		"thrust 1, kgs": float64(0),
	}

	systems := buildSystems(state, nil, false)

	if systems.Engines[0].Status != engineStatusIdle {
		t.Fatalf("expected idle, got %q", systems.Engines[0].Status)
	}
}

func TestBuildSystemsReportsAircraftLostWhenDestroyed(t *testing.T) {
	state := map[string]any{"throttle 1, %": float64(100), "RPM 1": float64(11385)}

	systems := buildSystems(state, nil, true)

	if systems.Status != "Aircraft lost" || systems.Severity != "critical" {
		t.Fatalf("got status=%q severity=%q", systems.Status, systems.Severity)
	}
}

func TestBuildSystemsDetectsFrozenAirborneWreck(t *testing.T) {
	state := map[string]any{
		"valid":         true,
		"IAS, km/h":     float64(0),
		"TAS, km/h":     float64(0),
		"H, m":          float64(1636),
		"gear, %":       float64(0),
		"throttle 1, %": float64(0),
		"RPM 1":         float64(0),
		"thrust 1, kgs": float64(0),
	}

	systems := buildSystems(state, nil, false)

	if systems.Status != "Aircraft lost" || systems.Severity != "critical" {
		t.Fatalf("frozen wreck reported as %q/%q", systems.Status, systems.Severity)
	}
}

func TestBuildSystemsDoesNotTreatParkedAircraftAsWreck(t *testing.T) {
	state := map[string]any{
		"IAS, km/h":     float64(0),
		"TAS, km/h":     float64(0),
		"H, m":          float64(1636),
		"gear, %":       float64(100),
		"throttle 1, %": float64(0),
		"RPM 1":         float64(0),
		"thrust 1, kgs": float64(0),
	}

	systems := buildSystems(state, nil, false)

	if systems.Status == "Aircraft lost" {
		t.Fatal("parked aircraft was reported lost")
	}
}
