package wtradio

import (
	"regexp"
	"strconv"
	"strings"
)

var (
	markupPattern    = regexp.MustCompile(`<[^>]*>`)
	locationPattern  = regexp.MustCompile(`(?i)\[([a-z]{1,2})(\d{1,2})(?:,\s*alt\.\s*([+-]?\d+(?:\.\d+)?)\s*m)?\]`)
	gridLabelPattern = regexp.MustCompile(`^([A-Za-z]{1,2})(\d{1,2})$`)
)

func IsRTB(message string) bool {
	switch Normalize(message) {
	case "heading to the base",
		"heading to base",
		"heading to the airfield",
		"heading back to the base",
		"returning to the base",
		"returning to base",
		"returning to the airfield",
		"returning to airfield",
		"going back to the base",
		"going home":
		return true
	}
	return false
}

func MarkKind(message string) (string, bool) {
	normalized := Normalize(message)
	switch normalized {
	case "guide on me",
		"follow me",
		"move after me":
		return "guide", true
	case "attention to the map",
		"attention to the designated grid zone",
		"attention to designated grid zone":
		return "attention", true
	case "cover me":
		return "cover", true
	case "need help",
		"help me":
		return "help", true
	}
	return "", false
}

func Normalize(message string) string {
	message = locationPattern.ReplaceAllString(StripMarkup(message), "")
	return strings.Trim(strings.ToLower(strings.TrimSpace(message)), " .!?")
}

func StripMarkup(message string) string {
	return strings.TrimSpace(markupPattern.ReplaceAllString(message, ""))
}

func ExtractGrid(message string) (string, bool) {
	match := locationPattern.FindStringSubmatch(message)
	if match == nil {
		return "", false
	}
	return strings.ToUpper(match[1]) + match[2], true
}

func ExtractAltitudeMeters(message string) (float64, bool) {
	match := locationPattern.FindStringSubmatch(message)
	if match == nil || match[3] == "" {
		return 0, false
	}
	altitude, err := strconv.ParseFloat(match[3], 64)
	if err != nil {
		return 0, false
	}
	return altitude, true
}

func ParseGrid(label string) (column string, row int, ok bool) {
	match := gridLabelPattern.FindStringSubmatch(strings.TrimSpace(label))
	if match == nil {
		return "", 0, false
	}
	row, err := strconv.Atoi(match[2])
	if err != nil || row < 1 {
		return "", 0, false
	}
	return strings.ToUpper(match[1]), row, true
}
