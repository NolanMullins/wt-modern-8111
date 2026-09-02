package wtradio

import (
	"regexp"
	"strconv"
	"strings"
)

var (
	markupPattern    = regexp.MustCompile(`<[^>]*>`)
	gridPattern      = regexp.MustCompile(`\[([A-Za-z]{1,2})(\d{1,2})\]`)
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
	switch {
	case strings.HasPrefix(normalized, "guide on me"),
		strings.HasPrefix(normalized, "follow me"):
		return "guide", true
	case strings.HasPrefix(normalized, "attention to the map"),
		strings.HasPrefix(normalized, "attention to the designated grid zone"):
		return "attention", true
	case strings.HasPrefix(normalized, "cover me"):
		return "cover", true
	case strings.HasPrefix(normalized, "need help"),
		strings.HasPrefix(normalized, "help me"):
		return "help", true
	}
	return "", false
}

func Normalize(message string) string {
	return strings.Trim(strings.ToLower(strings.TrimSpace(StripMarkup(message))), " .!?")
}

func StripMarkup(message string) string {
	return strings.TrimSpace(markupPattern.ReplaceAllString(message, ""))
}

func ExtractGrid(message string) (string, bool) {
	match := gridPattern.FindStringSubmatch(message)
	if match == nil {
		return "", false
	}
	return strings.ToUpper(match[1]) + match[2], true
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
