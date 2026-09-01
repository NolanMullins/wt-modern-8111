package warthunder

import (
	"reflect"
	"testing"
)

func TestParseMapInfoAcceptsNumericStrings(t *testing.T) {
	info, err := parseMapInfo(map[string]any{
		"valid":          true,
		"grid_steps":     []any{"6400.0", "6400.0"},
		"map_generation": "2",
		"map_min":        []any{"-32768", "-32768"},
		"map_max":        []any{"32768", "32768"},
	})
	if err != nil {
		t.Fatalf("parseMapInfo: %v", err)
	}
	if info.Generation != 2 {
		t.Fatalf("generation = %d, want 2", info.Generation)
	}
	if !reflect.DeepEqual(info.GridSteps, []float64{6400, 6400}) {
		t.Fatalf("grid steps = %#v", info.GridSteps)
	}
	if !reflect.DeepEqual(info.MapMin, []float64{-32768, -32768}) {
		t.Fatalf("map min = %#v", info.MapMin)
	}
}

func TestParseMapInfoRejectsInvalidArrayValue(t *testing.T) {
	_, err := parseMapInfo(map[string]any{
		"grid_steps": []any{6400.0, true},
	})
	if err == nil {
		t.Fatal("parseMapInfo returned nil error for invalid grid_steps")
	}
}

func TestParseMapInfoInfersLegacyValidityFromGeometry(t *testing.T) {
	info, err := parseMapInfo(map[string]any{
		"grid_steps":     []any{"6400.0", "6400.0"},
		"map_generation": "1",
		"map_min":        []any{"-32768", "-32768"},
		"map_max":        []any{"32768", "32768"},
	})
	if err != nil {
		t.Fatalf("parseMapInfo: %v", err)
	}
	if !info.Valid {
		t.Fatal("legacy map geometry was not inferred valid")
	}

	explicitlyInvalid, err := parseMapInfo(map[string]any{
		"valid":      false,
		"grid_steps": []any{6400.0, 6400.0},
		"map_min":    []any{-32768.0, -32768.0},
		"map_max":    []any{32768.0, 32768.0},
	})
	if err != nil {
		t.Fatalf("parse explicitly invalid map info: %v", err)
	}
	if explicitlyInvalid.Valid {
		t.Fatal("explicit valid=false was overridden")
	}
}
