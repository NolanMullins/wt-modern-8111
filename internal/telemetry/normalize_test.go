package telemetry

import (
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/NolanMullins/wt-modern-8111/internal/warthunder"
)

func TestSnapshotJSONMaintainsFrontendEnvelope(t *testing.T) {
	snapshot := BuildSnapshot(1, time.Now(), "live", RawData{}, nil)
	body, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatal(err)
	}
	if string(envelope["version"]) != "1" {
		t.Fatalf("version = %s, want 1", envelope["version"])
	}
	for _, field := range []string{"feed", "allyMarks", "pilot"} {
		if value := string(envelope[field]); value == "" || value == "null" {
			t.Fatalf("%s must be present and non-null: %s", field, body)
		}
	}
}

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
	if snapshot.Ground.SpeedKMH != nil {
		t.Fatalf("air snapshot ground speed = %v, want nil", snapshot.Ground.SpeedKMH)
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

func TestBuildSnapshotNormalizesGroundVehicleTelemetryAndMapCounts(t *testing.T) {
	playerX, playerY := 0.25, 0.75
	friendlyX, friendlyY := 0.3, 0.7
	hostileX, hostileY := 0.7, 0.3
	zoneX, zoneY := 0.5, 0.5
	spawnX, spawnY := 0.1, 0.9
	raw := RawData{
		State: map[string]any{"valid": true},
		Indicators: map[string]any{
			"valid":            true,
			"army":             "ground",
			"type":             "tankModels/ussr_t_80u",
			"speed":            10.0,
			"compass":          91.5,
			"rpm":              1900.0,
			"gear":             4.0,
			"cruise_control":   0.0,
			"first_stage_ammo": 23.0,
			"crew_current":     3.0,
			"crew_total":       3.0,
			"driver_state":     0.0,
			"gunner_state":     0.0,
			"stabilizer":       1.0,
			"lws":              -1.0,
			"ircm":             1.0,
		},
		MapInfo: warthunder.MapInfo{Valid: true, HUDType: 1},
		MapObjects: []warthunder.MapObject{
			{Type: "ground_model", Icon: "Player", X: &playerX, Y: &playerY},
			{Type: "ground_model", Color: "#174DFF", X: &friendlyX, Y: &friendlyY},
			{Type: "ground_model", Color: "#fa0C00", X: &hostileX, Y: &hostileY},
			{Type: "capture_zone", X: &zoneX, Y: &zoneY},
			{Type: "respawn_base_tank", X: &spawnX, Y: &spawnY},
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

	if snapshot.Ground.SpeedKMH == nil || math.Abs(*snapshot.Ground.SpeedKMH-10) > 0.001 {
		t.Fatalf("ground speed = %v, want 10 km/h", snapshot.Ground.SpeedKMH)
	}
	if snapshot.Ground.HeadingDeg == nil || *snapshot.Ground.HeadingDeg != 91.5 {
		t.Fatalf("ground heading = %v, want 91.5", snapshot.Ground.HeadingDeg)
	}
	if snapshot.Ground.Ammo == nil || *snapshot.Ground.Ammo != 23 {
		t.Fatalf("ground ammo = %v, want 23", snapshot.Ground.Ammo)
	}
	if snapshot.Ground.CrewCurrent == nil || *snapshot.Ground.CrewCurrent != 3 ||
		snapshot.Ground.CrewTotal == nil || *snapshot.Ground.CrewTotal != 3 {
		t.Fatalf("ground crew = %v/%v, want 3/3", snapshot.Ground.CrewCurrent, snapshot.Ground.CrewTotal)
	}
	if snapshot.Ground.LWS != nil {
		t.Fatalf("negative LWS sentinel normalized as %v", snapshot.Ground.LWS)
	}
	if snapshot.Map.HUDType != 1 {
		t.Fatalf("map HUD type = %d, want 1", snapshot.Map.HUDType)
	}
	unavailableAmmo := buildGround(
		Vehicle{Class: "ground"},
		RawData{Indicators: map[string]any{"first_stage_ammo": -1.0}},
	)
	if unavailableAmmo.Ammo != nil {
		t.Fatalf("negative ammo sentinel normalized as %v", unavailableAmmo.Ammo)
	}
	if snapshot.Map.Counts.FriendlyGround != 1 || snapshot.Map.Counts.HostileGround != 1 ||
		snapshot.Map.Counts.CaptureZone != 1 || snapshot.Map.Counts.GroundSpawn != 1 {
		t.Fatalf("ground map counts = %#v", snapshot.Map.Counts)
	}
	if snapshot.Navigation != nil {
		t.Fatalf("ground snapshot selected air RTB navigation: %#v", snapshot.Navigation)
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

func TestMapColorAffiliationHandlesLiveGroundColors(t *testing.T) {
	for _, color := range []string{"#174DFF", "#043FFF", "#67D756"} {
		if !friendlyMapColor(color) {
			t.Errorf("friendlyMapColor(%q) = false", color)
		}
	}
	for _, color := range []string{"#fa0C00", "#fb655C"} {
		if !hostileMapColor(color) {
			t.Errorf("hostileMapColor(%q) = false", color)
		}
	}
	if hostileMapColor("#faC81E") {
		t.Error("player yellow classified as hostile")
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
