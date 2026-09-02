package polling

import (
	"math"

	"github.com/NolanMullins/wt-modern-8111/internal/telemetry"
	"github.com/NolanMullins/wt-modern-8111/internal/warthunder"
)

func (s *Service) canonicalRawDataLocked() telemetry.RawData {
	raw := s.raw
	if !s.groundMapInfo.Valid || isGroundArmy(raw.Indicators) {
		return raw
	}
	raw.MapObjects = transformMapObjects(raw.MapObjects, raw.MapInfo, s.groundMapInfo)
	raw.AllyMarks = transformAllyMarks(raw.AllyMarks, raw.MapInfo, s.groundMapInfo)
	groundInfo := cloneMapInfo(s.groundMapInfo)
	groundInfo.Valid = raw.MapInfo.Valid
	groundInfo.ValidPresent = raw.MapInfo.ValidPresent
	groundInfo.Generation = raw.MapInfo.Generation
	groundInfo.HUDType = raw.MapInfo.HUDType
	raw.MapInfo = groundInfo
	return raw
}

func transformMapObjects(
	objects []warthunder.MapObject,
	current warthunder.MapInfo,
	ground warthunder.MapInfo,
) []warthunder.MapObject {
	if !validTransformBounds(current, ground) {
		return append([]warthunder.MapObject(nil), objects...)
	}
	result := make([]warthunder.MapObject, len(objects))
	for index, object := range objects {
		result[index] = object
		result[index].X, result[index].Y = transformPoint(object.X, object.Y, current, ground)
		result[index].SX, result[index].SY = transformPoint(object.SX, object.SY, current, ground)
		result[index].EX, result[index].EY = transformPoint(object.EX, object.EY, current, ground)
		result[index].DX, result[index].DY = transformDirection(object.DX, object.DY, current, ground)
	}
	return result
}

func transformAllyMarks(
	marks []telemetry.AllyMark,
	current warthunder.MapInfo,
	ground warthunder.MapInfo,
) []telemetry.AllyMark {
	result := append([]telemetry.AllyMark(nil), marks...)
	for index := range result {
		result[index].X, result[index].Y = transformPoint(
			result[index].X,
			result[index].Y,
			current,
			ground,
		)
	}
	return result
}

func transformPoint(
	x, y *float64,
	current warthunder.MapInfo,
	ground warthunder.MapInfo,
) (*float64, *float64) {
	if x == nil || y == nil || !validTransformBounds(current, ground) {
		return cloneNumber(x), cloneNumber(y)
	}
	currentWidth := current.MapMax[0] - current.MapMin[0]
	currentHeight := current.MapMax[1] - current.MapMin[1]
	groundWidth := ground.MapMax[0] - ground.MapMin[0]
	groundHeight := ground.MapMax[1] - ground.MapMin[1]
	worldX := current.MapMin[0] + *x*currentWidth
	worldY := current.MapMax[1] - *y*currentHeight
	groundX := (worldX - ground.MapMin[0]) / groundWidth
	groundY := (ground.MapMax[1] - worldY) / groundHeight
	return &groundX, &groundY
}

func transformDirection(
	dx, dy *float64,
	current warthunder.MapInfo,
	ground warthunder.MapInfo,
) (*float64, *float64) {
	if dx == nil || dy == nil || !validTransformBounds(current, ground) {
		return cloneNumber(dx), cloneNumber(dy)
	}
	x := *dx * (current.MapMax[0] - current.MapMin[0]) /
		(ground.MapMax[0] - ground.MapMin[0])
	y := *dy * (current.MapMax[1] - current.MapMin[1]) /
		(ground.MapMax[1] - ground.MapMin[1])
	length := math.Hypot(x, y)
	if length > 0 {
		x /= length
		y /= length
	}
	return &x, &y
}

func validTransformBounds(current, ground warthunder.MapInfo) bool {
	return len(current.MapMin) >= 2 && len(current.MapMax) >= 2 &&
		len(ground.MapMin) >= 2 && len(ground.MapMax) >= 2 &&
		current.MapMax[0] > current.MapMin[0] &&
		current.MapMax[1] > current.MapMin[1] &&
		ground.MapMax[0] > ground.MapMin[0] &&
		ground.MapMax[1] > ground.MapMin[1]
}

func cloneMapInfo(info warthunder.MapInfo) warthunder.MapInfo {
	info.GridSize = append([]float64(nil), info.GridSize...)
	info.GridSteps = append([]float64(nil), info.GridSteps...)
	info.GridZero = append([]float64(nil), info.GridZero...)
	info.MapMin = append([]float64(nil), info.MapMin...)
	info.MapMax = append([]float64(nil), info.MapMax...)
	return info
}

func cloneNumber(value *float64) *float64 {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}
