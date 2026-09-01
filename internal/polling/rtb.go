package polling

import (
	"math"

	"github.com/NolanMullins/wt-modern-8111/internal/telemetry"
	"github.com/NolanMullins/wt-modern-8111/internal/warthunder"
)

const landingConfirmationSamples = 10

func (s *Service) processDamageRecordLocked(record warthunder.FeedRecord) {
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
		if s.landingSamples >= landingConfirmationSamples {
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
		if object.Type != "airfield" || object.SX == nil || object.SY == nil ||
			object.EX == nil || object.EY == nil {
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
