package telemetry

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/NolanMullins/wt-modern-8111/internal/warthunder"
)

func BuildSnapshot(sequence uint64, now time.Time, mode string, raw RawData, sources map[string]SourceStatus) Snapshot {
	fresh := freshRawData(raw, sources)
	snapshot := Snapshot{
		Version:    1,
		Sequence:   sequence,
		CapturedAt: now,
		Connection: Connection{State: connectionState(raw, sources), Mode: mode, Sources: sources},
		Vehicle: Vehicle{
			Type:  stringValue(fresh.Indicators["type"]),
			Class: stringValue(fresh.Indicators["army"]),
		},
		Flight: Flight{
			IASKMH:           number(fresh.State, "IAS, km/h"),
			TASKMH:           number(fresh.State, "TAS, km/h"),
			AltitudeM:        number(fresh.State, "H, m"),
			RadioAltitudeM:   number(fresh.Indicators, "radio_altitude"),
			Mach:             firstNumber(number(fresh.State, "M"), number(fresh.Indicators, "mach")),
			HeadingDeg:       heading(fresh),
			VerticalSpeedMPS: number(fresh.State, "Vy, m/s"),
			AOADeg:           number(fresh.State, "AoA, deg"),
			GLoad:            number(fresh.State, "Ny"),
		},
		Systems:   buildSystems(fresh.State, fresh.Indicators, raw.Destroyed),
		Mission:   buildMission(fresh.Mission),
		Map:       buildMap(fresh.MapInfo, fresh.MapObjects),
		Feed:      append(make([]FeedEntry, 0, len(fresh.Feed)), fresh.Feed...),
		Pilot:     raw.Pilot,
		AllyMarks: append(make([]AllyMark, 0, len(raw.AllyMarks)), raw.AllyMarks...),
	}
	snapshot.Navigation = buildNavigation(snapshot.Map, snapshot.Flight, raw.ReturnToAirfield)
	return snapshot
}

// buildSystems derives real aircraft system health rather than assuming that
// the presence of engine telemetry implies a healthy aircraft. An engine is
// treated as failed when the pilot is commanding power but the engine produces
// neither RPM nor thrust, which is the signature of an engine that has been
// shot out.
func buildSystems(state map[string]any, indicators map[string]any, destroyed bool) Systems {
	systems := Systems{
		Engines:         make([]Engine, 0),
		Warnings:        make([]string, 0),
		FuelKG:          firstNumber(number(indicators, "fuel"), number(state, "Mfuel, kg")),
		GearPercent:     number(state, "gear, %"),
		FlapsPercent:    number(state, "flaps, %"),
		AirbrakePercent: number(state, "airbrake, %"),
	}
	if fuel := systems.FuelKG; fuel != nil {
		if initial := number(state, "Mfuel0, kg"); initial != nil && *initial > 0 {
			value := *fuel / *initial * 100
			systems.FuelPercent = &value
		}
	}
	for index := 1; index <= 4; index++ {
		engine := Engine{
			Index:           index,
			ThrottlePercent: number(state, fmt.Sprintf("throttle %d, %%", index)),
			RPM:             number(state, fmt.Sprintf("RPM %d", index)),
			OilTempC:        number(state, fmt.Sprintf("oil temp %d, C", index)),
			ThrustKGF:       number(state, fmt.Sprintf("thrust %d, kgs", index)),
		}
		if engine.ThrottlePercent == nil && engine.RPM == nil && engine.OilTempC == nil && engine.ThrustKGF == nil {
			continue
		}
		power := number(state, fmt.Sprintf("power %d, hp", index))
		engine.Status, engine.Running = engineStatus(engine, power)
		if engine.Status == engineStatusFailed {
			systems.Warnings = append(systems.Warnings, fmt.Sprintf("Engine %d out", index))
		}
		systems.Engines = append(systems.Engines, engine)
	}

	if fire := number(indicators, "fire"); fire != nil && *fire > 0 {
		systems.Warnings = append(systems.Warnings, "Fire")
	}
	if systems.FuelPercent != nil && *systems.FuelPercent <= 10 {
		systems.Warnings = append(systems.Warnings, "Fuel low")
	}

	systems.Status, systems.Severity = systemsSummary(
		systems,
		destroyed || frozenAirborneWreck(state, systems),
	)
	return systems
}

const (
	engineStatusRunning = "running"
	engineStatusIdle    = "idle"
	engineStatusFailed  = "failed"
	engineStatusUnknown = "unknown"
)

func engineStatus(engine Engine, power *float64) (string, bool) {
	rpm := engine.RPM
	thrust := engine.ThrustKGF
	if rpm == nil && thrust == nil && power == nil {
		return engineStatusUnknown, false
	}

	// Producing output in any form means the engine is alive. Props report
	// power/RPM, jets report thrust/RPM, so any positive signal counts.
	producing := (rpm != nil && *rpm > 100) ||
		(thrust != nil && *thrust > 1) ||
		(power != nil && *power > 1)
	if producing {
		return engineStatusRunning, true
	}

	// No output at all. If the pilot is commanding throttle, the engine has
	// failed; otherwise it is deliberately shut down.
	if engine.ThrottlePercent != nil && *engine.ThrottlePercent > 5 {
		return engineStatusFailed, false
	}
	return engineStatusIdle, false
}

func systemsSummary(systems Systems, destroyed bool) (string, string) {
	if destroyed {
		return "Aircraft lost", "critical"
	}
	if len(systems.Engines) == 0 {
		return "Unavailable", "unknown"
	}
	failed := 0
	known := 0
	for _, engine := range systems.Engines {
		if engine.Status == engineStatusUnknown {
			continue
		}
		known++
		if engine.Status == engineStatusFailed {
			failed++
		}
	}
	switch {
	case known == 0:
		return "Unavailable", "unknown"
	case failed > 0 && failed == known:
		return "Engine out", "critical"
	case failed > 0:
		return fmt.Sprintf("%d engine%s out", failed, plural(failed)), "critical"
	case len(systems.Warnings) > 0:
		return systems.Warnings[0], "caution"
	default:
		return "Nominal", "good"
	}
}

// frozenAirborneWreck detects the post-loss state War Thunder sometimes leaves
// behind: valid remains true, altitude freezes well above the terrain, all
// motion and engine output become zero, and the gear remains retracted. A
// parked aircraft is excluded by its extended gear.
func frozenAirborneWreck(state map[string]any, systems Systems) bool {
	ias := number(state, "IAS, km/h")
	tas := number(state, "TAS, km/h")
	altitude := number(state, "H, m")
	if ias == nil || tas == nil || altitude == nil ||
		*ias > 1 || *tas > 1 || *altitude < 50 {
		return false
	}
	if systems.GearPercent != nil && *systems.GearPercent >= 50 {
		return false
	}
	for _, engine := range systems.Engines {
		if engine.Running {
			return false
		}
	}
	return len(systems.Engines) > 0
}

func plural(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}

func buildMission(mission warthunder.Mission) Mission {
	result := Mission{Status: mission.Status, Objectives: []warthunder.Objective{}}
	if mission.Objectives != nil {
		result.Objectives = append(result.Objectives, (*mission.Objectives)...)
	}
	return result
}

func buildMap(info warthunder.MapInfo, objects []warthunder.MapObject) Map {
	result := Map{
		Valid:      info.Valid,
		Generation: info.Generation,
		GridSize:   append([]float64(nil), info.GridSize...),
		GridSteps:  append([]float64(nil), info.GridSteps...),
		GridZero:   append([]float64(nil), info.GridZero...),
		MapMin:     append([]float64(nil), info.MapMin...),
		MapMax:     append([]float64(nil), info.MapMax...),
		Objects:    append(make([]warthunder.MapObject, 0, len(objects)), objects...),
	}
	for _, object := range objects {
		result.Counts.Total++
		switch {
		case object.Type == "aircraft" && object.Icon != "Player":
			result.Counts.HostileAir++
		case object.Type == "ground_model":
			result.Counts.Ground++
		}
		if object.Icon == "SAM" || object.Icon == "SPAA" {
			result.Counts.AirDefense++
		}
		if object.Type == "bombing_point" {
			result.Counts.StrikePoint++
		}
		if object.Type == "airfield" {
			result.Counts.Airfield++
		}
	}
	return result
}

func buildNavigation(mapData Map, flight Flight, returnToAirfield bool) *Navigation {
	// Normal map targets are selected explicitly in the frontend. The backend
	// only creates a destination for the pilot's RTB radio command.
	if !returnToAirfield || !mapData.Valid || len(mapData.MapMin) < 2 || len(mapData.MapMax) < 2 {
		return nil
	}
	var player *warthunder.MapObject
	type candidate struct {
		name     string
		priority int
		x        float64
		y        float64
	}
	targets := make([]candidate, 0)
	for index := range mapData.Objects {
		object := &mapData.Objects[index]
		if object.Icon == "Player" {
			player = object
		}
		if object.Type == "airfield" &&
			friendlyMapColor(object.Color) &&
			object.SX != nil && object.SY != nil &&
			object.EX != nil && object.EY != nil {
			targets = append(targets, candidate{
				name:     "Nearest airfield",
				priority: 0,
				x:        (*object.SX + *object.EX) / 2,
				y:        (*object.SY + *object.EY) / 2,
			})
		}
	}
	if player == nil || player.X == nil || player.Y == nil || len(targets) == 0 {
		return nil
	}
	width := mapData.MapMax[0] - mapData.MapMin[0]
	height := mapData.MapMax[1] - mapData.MapMin[1]
	if math.IsNaN(width) || math.IsInf(width, 0) || math.IsNaN(height) || math.IsInf(height, 0) || width <= 0 || height <= 0 {
		return nil
	}
	sort.Slice(targets, func(i, j int) bool {
		if targets[i].priority != targets[j].priority {
			return targets[i].priority < targets[j].priority
		}
		return worldDistance(*player.X, *player.Y, targets[i].x, targets[i].y, width, height) <
			worldDistance(*player.X, *player.Y, targets[j].x, targets[j].y, width, height)
	})
	target := targets[0]
	dx := (target.x - *player.X) * width
	dy := (target.y - *player.Y) * height
	distanceM := math.Hypot(dx, dy)
	bearing := math.Mod(math.Atan2(dx, -dy)*180/math.Pi+360, 360)
	result := &Navigation{
		Name:       target.name,
		BearingDeg: bearing,
		RangeKM:    distanceM / 1000,
		TargetX:    target.x,
		TargetY:    target.y,
		Basis:      "map bounds",
	}
	if flight.TASKMH != nil && *flight.TASKMH > 0 {
		seconds := result.RangeKM / *flight.TASKMH * 3600
		result.ETASeconds = &seconds
	}
	return result
}

func friendlyMapColor(color string) bool {
	switch strings.ToLower(strings.TrimSpace(color)) {
	case "#174dff", "#39d921":
		return true
	}
	return false
}

func heading(raw RawData) *float64 {
	if value := number(raw.Indicators, "compass"); value != nil {
		normalized := math.Mod(*value+360, 360)
		return &normalized
	}
	for _, object := range raw.MapObjects {
		if object.Icon == "Player" && object.DX != nil && object.DY != nil {
			value := math.Mod(math.Atan2(*object.DX, -*object.DY)*180/math.Pi+360, 360)
			return &value
		}
	}
	return nil
}

func connectionState(raw RawData, sources map[string]SourceStatus) string {
	stateFresh := sourceFresh(sources, "state")
	indicatorsFresh := sourceFresh(sources, "indicators")
	stateValid := stateFresh && boolMapValue(raw.State, "valid")
	indicatorsValid := indicatorsFresh && boolMapValue(raw.Indicators, "valid")
	if stateValid || indicatorsValid {
		for _, name := range []string{"state", "indicators", "mapObjects"} {
			if !sourceFresh(sources, name) {
				return "degraded"
			}
		}
		return "live"
	}
	if stateFresh || indicatorsFresh {
		return "hangar"
	}
	return "waiting-for-game"
}

func freshRawData(raw RawData, sources map[string]SourceStatus) RawData {
	result := RawData{Feed: make([]FeedEntry, 0)}
	if sourceFresh(sources, "state") {
		result.State = raw.State
	}
	if sourceFresh(sources, "indicators") {
		result.Indicators = raw.Indicators
	}
	if sourceFresh(sources, "mapInfo") {
		result.MapInfo = raw.MapInfo
	}
	if sourceFresh(sources, "mapObjects") {
		result.MapObjects = raw.MapObjects
	}
	if sourceFresh(sources, "mission") {
		result.Mission = raw.Mission
	}
	if sourceFresh(sources, "chat") || sourceFresh(sources, "hud") {
		result.Feed = raw.Feed
	}
	return result
}

func sourceFresh(sources map[string]SourceStatus, name string) bool {
	status, ok := sources[name]
	return ok && status.State == "fresh"
}

func number(values map[string]any, key string) *float64 {
	if values == nil {
		return nil
	}
	raw, ok := values[key]
	if !ok {
		return nil
	}
	switch value := raw.(type) {
	case float64:
		result := value
		return &result
	case int:
		result := float64(value)
		return &result
	case jsonNumber:
		parsed, err := strconv.ParseFloat(string(value), 64)
		if err == nil {
			return &parsed
		}
	}
	return nil
}

type jsonNumber string

func stringValue(raw any) string {
	value, _ := raw.(string)
	return value
}

func boolMapValue(values map[string]any, key string) bool {
	value, _ := values[key].(bool)
	return value
}

func firstNumber(values ...*float64) *float64 {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func worldDistance(sourceX, sourceY, targetX, targetY, width, height float64) float64 {
	return math.Hypot((targetX-sourceX)*width, (targetY-sourceY)*height)
}
