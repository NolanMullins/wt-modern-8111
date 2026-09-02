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
	if !activate || mine {
		return
	}
	if kind, ok := wtradio.MarkKind(message); ok {
		s.addAllyMarkLocked(kind, record)
	}
}

func (s *Service) addAllyMarkLocked(kind string, record warthunder.FeedRecord) {
	now := s.now()
	mark := telemetry.AllyMark{
		Key:       fmt.Sprintf("chat-%d", record.ID),
		Kind:      kind,
		Sender:    record.Sender,
		Message:   wtradio.StripMarkup(record.Message),
		CreatedAt: now,
		ExpiresAt: now.Add(allyMarkTTL),
	}
	if grid, ok := wtradio.ExtractGrid(record.Message); ok {
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

func gridToNormalized(grid string, info warthunder.MapInfo) (float64, float64, bool) {
	columnLabel, row, ok := wtradio.ParseGrid(grid)
	if !ok {
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
	for _, symbol := range columnLabel {
		column = column*26 + int(symbol-'A') + 1
	}
	column--
	row--
	worldX := info.GridZero[0] + (float64(column)+0.5)*stepX
	worldY := info.GridZero[1] - (float64(row)+0.5)*math.Abs(stepY)
	x := (worldX - info.MapMin[0]) / spanX
	y := (info.MapMax[1] - worldY) / spanY
	if x < 0 || x > 1 || y < 0 || y > 1 {
		return 0, 0, false
	}
	return x, y, true
}
