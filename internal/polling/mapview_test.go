package polling

import (
	"math"
	"testing"

	"github.com/NolanMullins/wt-modern-8111/internal/telemetry"
	"github.com/NolanMullins/wt-modern-8111/internal/warthunder"
)

func TestCanonicalRawDataTransformsCASIntoGroundMap(t *testing.T) {
	x, y := 0.5, 0.5
	markX, markY := 0.25, 0.75
	service := newTestService()
	service.raw.Indicators = map[string]any{"army": "air"}
	service.raw.MapInfo = warthunder.MapInfo{
		Valid:        true,
		ValidPresent: true,
		Generation:   7,
		HUDType:      1,
		MapMin:       []float64{-1000, -1000},
		MapMax:       []float64{3000, 3000},
	}
	service.raw.MapObjects = []warthunder.MapObject{
		{Type: "aircraft", Icon: "Player", X: &x, Y: &y},
	}
	service.raw.AllyMarks = []telemetry.AllyMark{{X: &markX, Y: &markY}}
	service.groundMapInfo = warthunder.MapInfo{
		Valid:      true,
		Generation: 2,
		GridSteps:  []float64{100, 100},
		MapMin:     []float64{0, 0},
		MapMax:     []float64{1000, 1000},
	}

	raw := service.canonicalRawDataLocked()

	if raw.MapObjects[0].X == nil || raw.MapObjects[0].Y == nil ||
		math.Abs(*raw.MapObjects[0].X-1) > 0.001 ||
		math.Abs(*raw.MapObjects[0].Y) > 0.001 {
		t.Fatalf("player position = %v, %v, want 1, 0", raw.MapObjects[0].X, raw.MapObjects[0].Y)
	}
	if raw.AllyMarks[0].X == nil || raw.AllyMarks[0].Y == nil ||
		math.Abs(*raw.AllyMarks[0].X) > 0.001 ||
		math.Abs(*raw.AllyMarks[0].Y-1) > 0.001 {
		t.Fatalf("ally mark = %v, %v, want 0, 1", raw.AllyMarks[0].X, raw.AllyMarks[0].Y)
	}
	if raw.MapInfo.GridSteps[0] != 100 {
		t.Fatalf("grid steps = %v", raw.MapInfo.GridSteps)
	}
	if !raw.MapInfo.Valid || !raw.MapInfo.ValidPresent ||
		raw.MapInfo.Generation != 7 || raw.MapInfo.HUDType != 1 {
		t.Fatalf("current map identity was replaced: %#v", raw.MapInfo)
	}
	if service.raw.MapObjects[0].X == nil || *service.raw.MapObjects[0].X != 0.5 {
		t.Fatal("canonicalization mutated raw CAS objects")
	}
}

func TestCanonicalRawDataLeavesGroundViewUnchanged(t *testing.T) {
	x, y := 0.4, 0.6
	service := newTestService()
	service.raw.Indicators = map[string]any{"army": "tank"}
	service.raw.MapObjects = []warthunder.MapObject{{X: &x, Y: &y}}
	service.groundMapInfo = warthunder.MapInfo{
		Valid:  true,
		MapMin: []float64{0, 0},
		MapMax: []float64{1000, 1000},
	}

	raw := service.canonicalRawDataLocked()

	if raw.MapObjects[0].X == nil || *raw.MapObjects[0].X != 0.4 {
		t.Fatalf("ground position changed: %v", raw.MapObjects[0].X)
	}
}
