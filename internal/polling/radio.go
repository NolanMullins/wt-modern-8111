package polling

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/NolanMullins/wt-modern-8111/internal/telemetry"
	"github.com/NolanMullins/wt-modern-8111/internal/warthunder"
	"github.com/NolanMullins/wt-modern-8111/internal/wtradio"
)

const (
	allyMarkVisibleDuration = 30 * time.Second
	allyMarkFadeDuration    = 5 * time.Second
	allyMarkTTL             = allyMarkVisibleDuration + allyMarkFadeDuration
	mapSignalMatchWindow    = 3 * time.Second
	maxAllyMarks            = 12
)

func (s *Service) processChatRecordLocked(record warthunder.FeedRecord, activate bool) {
	if record.Enemy || record.Sender == "" || !strings.EqualFold(record.Mode, "Team") {
		return
	}
	message := wtradio.StripMarkup(record.Message)
	mine := s.identity.Matches(record.Sender)

	if wtradio.IsRTB(message) {
		if activate && mine {
			s.returnToAirfield = true
			s.landingSamples = 0
		}
		return
	}
	kind, marked := wtradio.MarkKind(message)
	if !activate || !marked {
		return
	}
	if mine {
		_, hasGrid := wtradio.ExtractGrid(record.Message)
		if kind != "attention" || !hasGrid {
			return
		}
	}
	s.addAllyMarkLocked(kind, record)
}

func (s *Service) addAllyMarkLocked(kind string, record warthunder.FeedRecord) {
	now := s.now()
	mark := telemetry.AllyMark{
		Key:       fmt.Sprintf("chat-%d", record.ID),
		Kind:      kind,
		Sender:    record.Sender,
		Message:   wtradio.StripMarkup(record.Message),
		Subject:   "sender",
		Source:    "chat",
		Precision: "unavailable",
		CreatedAt: now,
		ExpiresAt: now.Add(allyMarkTTL),
	}
	if kind == "attention" {
		mark.Subject = "target"
	}
	if grid, ok := wtradio.ExtractGrid(record.Message); ok {
		mark.Grid = grid
		mark.Precision = "grid"
		if x, y, area, ok := normalizedGridPosition(grid, s.raw.MapInfo); ok {
			mark.X = &x
			mark.Y = &y
			mark.Area = area
			mark.Located = true
		}
	}
	if altitude, ok := wtradio.ExtractAltitudeMeters(record.Message); ok {
		mark.AltitudeM = &altitude
	}
	if kind == "attention" && s.fuseChatWithExactPointLocked(&mark, now) {
		s.pruneAllyMarksLocked(now)
		return
	}
	s.allyMarks = append(s.allyMarks, mark)
	s.pruneAllyMarksLocked(now)
}

func (s *Service) processMapObjectsLocked(objects []warthunder.MapObject, activate bool) {
	current := make(map[string]struct{})
	for _, object := range objects {
		key, ok := pointSignalKey(object)
		if !ok {
			continue
		}
		current[key] = struct{}{}
		if activate {
			if _, existed := s.pointSignalKeys[key]; !existed {
				s.addExactPointLocked(key, object)
			}
		}
	}
	s.pointSignalKeys = current
}

func (s *Service) addExactPointLocked(key string, object warthunder.MapObject) {
	now := s.now()
	markKey := "map-" + key
	for index := range s.allyMarks {
		if s.allyMarks[index].Key != markKey {
			continue
		}
		s.allyMarks[index] = telemetry.AllyMark{
			Key:       markKey,
			Kind:      "attention",
			Message:   "Exact map signal",
			Subject:   "target",
			Source:    "telemetry",
			Precision: "exact",
			X:         cloneFloat(object.X),
			Y:         cloneFloat(object.Y),
			Located:   true,
			CreatedAt: now,
			ExpiresAt: now.Add(allyMarkTTL),
		}
		s.fuseExistingExactWithChatLocked(index, now)
		return
	}
	if s.fuseExactPointWithChatLocked(markKey, object, now) {
		s.pruneAllyMarksLocked(now)
		return
	}
	s.allyMarks = append(s.allyMarks, telemetry.AllyMark{
		Key:       markKey,
		Kind:      "attention",
		Message:   "Exact map signal",
		Subject:   "target",
		Source:    "telemetry",
		Precision: "exact",
		X:         cloneFloat(object.X),
		Y:         cloneFloat(object.Y),
		Located:   true,
		CreatedAt: now,
		ExpiresAt: now.Add(allyMarkTTL),
	})
	s.pruneAllyMarksLocked(now)
}

func (s *Service) fuseChatWithExactPointLocked(mark *telemetry.AllyMark, now time.Time) bool {
	for index := len(s.allyMarks) - 1; index >= 0; index-- {
		exact := &s.allyMarks[index]
		if exact.Subject != "target" || exact.Precision != "exact" ||
			exact.Sender != "" ||
			!signalTimesMatch(exact.CreatedAt, mark.CreatedAt) ||
			!pointMatchesGrid(exact.X, exact.Y, mark.Grid, s.raw.MapInfo) {
			continue
		}
		enrichExactMark(exact, *mark, now)
		return true
	}
	return false
}

func (s *Service) fuseExactPointWithChatLocked(
	key string,
	object warthunder.MapObject,
	now time.Time,
) bool {
	for index := len(s.allyMarks) - 1; index >= 0; index-- {
		mark := &s.allyMarks[index]
		if mark.Subject != "target" || mark.Precision != "grid" ||
			!signalTimesMatch(mark.CreatedAt, now) ||
			!pointMatchesGrid(object.X, object.Y, mark.Grid, s.raw.MapInfo) {
			continue
		}
		mark.Key = key
		mark.X = cloneFloat(object.X)
		mark.Y = cloneFloat(object.Y)
		mark.Source = "fused"
		mark.Precision = "exact"
		mark.ExpiresAt = now.Add(allyMarkTTL)
		return true
	}
	return false
}

func (s *Service) fuseExistingExactWithChatLocked(exactIndex int, now time.Time) bool {
	exact := &s.allyMarks[exactIndex]
	for chatIndex := len(s.allyMarks) - 1; chatIndex >= 0; chatIndex-- {
		if chatIndex == exactIndex {
			continue
		}
		chat := s.allyMarks[chatIndex]
		if chat.Source != "chat" || chat.Subject != "target" ||
			chat.Precision != "grid" ||
			!signalTimesMatch(exact.CreatedAt, chat.CreatedAt) ||
			!pointMatchesGrid(exact.X, exact.Y, chat.Grid, s.raw.MapInfo) {
			continue
		}
		enrichExactMark(exact, chat, now)
		s.allyMarks = append(s.allyMarks[:chatIndex], s.allyMarks[chatIndex+1:]...)
		return true
	}
	return false
}

func enrichExactMark(exact *telemetry.AllyMark, chat telemetry.AllyMark, now time.Time) {
	exact.Sender = chat.Sender
	exact.Message = chat.Message
	exact.Grid = chat.Grid
	exact.AltitudeM = cloneFloat(chat.AltitudeM)
	exact.Source = "fused"
	exact.ExpiresAt = now.Add(allyMarkTTL)
}

func signalTimesMatch(left, right time.Time) bool {
	delta := left.Sub(right)
	return delta >= -mapSignalMatchWindow && delta <= mapSignalMatchWindow
}

func pointSignalKey(object warthunder.MapObject) (string, bool) {
	if object.Type != "point_of_interest" || object.X == nil || object.Y == nil ||
		math.IsNaN(*object.X) || math.IsNaN(*object.Y) ||
		*object.X < 0 || *object.X > 1 || *object.Y < 0 || *object.Y > 1 {
		return "", false
	}
	return fmt.Sprintf("%.6f-%.6f", *object.X, *object.Y), true
}

func pointMatchesGrid(x, y *float64, grid string, info warthunder.MapInfo) bool {
	if x == nil || y == nil || grid == "" {
		return false
	}
	minX, minY, maxX, maxY, ok := normalizedGridBounds(grid, info)
	return ok && *x >= minX && *x <= maxX && *y >= minY && *y <= maxY
}

func cloneFloat(value *float64) *float64 {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
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

func (s *Service) resolveAllyMarksLocked() {
	for index := range s.allyMarks {
		mark := &s.allyMarks[index]
		if mark.Located || mark.Grid == "" {
			continue
		}
		if x, y, ok := gridToNormalized(mark.Grid, s.raw.MapInfo); ok {
			mark.X = &x
			mark.Y = &y
			_, _, mark.Area, _ = normalizedGridPosition(mark.Grid, s.raw.MapInfo)
			mark.Located = true
		}
	}
	remove := make(map[int]struct{})
	now := s.now()
	for index := range s.allyMarks {
		mark := &s.allyMarks[index]
		if mark.Source == "chat" && mark.Subject == "target" &&
			mark.Precision == "grid" &&
			s.fuseChatWithExactPointLocked(mark, now) {
			remove[index] = struct{}{}
		}
	}
	if len(remove) > 0 {
		kept := make([]telemetry.AllyMark, 0, len(s.allyMarks)-len(remove))
		for index, mark := range s.allyMarks {
			if _, discard := remove[index]; !discard {
				kept = append(kept, mark)
			}
		}
		s.allyMarks = kept
	}
}

func gridToNormalized(grid string, info warthunder.MapInfo) (float64, float64, bool) {
	x, y, _, ok := normalizedGridPosition(grid, info)
	return x, y, ok
}

func normalizedGridPosition(
	grid string,
	info warthunder.MapInfo,
) (float64, float64, *telemetry.MapArea, bool) {
	minX, minY, maxX, maxY, ok := normalizedGridBounds(grid, info)
	if !ok {
		return 0, 0, nil, false
	}
	return (minX + maxX) / 2, (minY + maxY) / 2, &telemetry.MapArea{
		MinX: minX,
		MinY: minY,
		MaxX: maxX,
		MaxY: maxY,
	}, true
}

func normalizedGridBounds(
	grid string,
	info warthunder.MapInfo,
) (float64, float64, float64, float64, bool) {
	rowLabel, column, ok := wtradio.ParseGrid(grid)
	if !ok {
		return 0, 0, 0, 0, false
	}
	if len(info.GridSteps) < 2 || len(info.MapMin) < 2 || len(info.MapMax) < 2 {
		return 0, 0, 0, 0, false
	}
	stepX, stepY := info.GridSteps[0], info.GridSteps[1]
	spanX := info.MapMax[0] - info.MapMin[0]
	spanY := info.MapMax[1] - info.MapMin[1]
	if stepX == 0 || stepY == 0 || spanX == 0 || spanY == 0 {
		return 0, 0, 0, 0, false
	}

	row := 0
	for _, symbol := range rowLabel {
		row = row*26 + int(symbol-'A') + 1
	}
	row--
	column--
	minX := float64(column) * math.Abs(stepX) / spanX
	minY := float64(row) * math.Abs(stepY) / spanY
	maxX := minX + math.Abs(stepX)/spanX
	maxY := minY + math.Abs(stepY)/spanY
	if minX < 0 || minX >= 1 || minY < 0 || minY >= 1 {
		return 0, 0, 0, 0, false
	}
	return minX, minY, math.Min(1, maxX), math.Min(1, maxY), true
}
