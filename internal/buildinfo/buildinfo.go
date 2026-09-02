package buildinfo

import (
	"strconv"
	"strings"
)

// Version is replaced from a release tag through the Go linker.
var Version = "dev"

func Current() string {
	version := strings.TrimSpace(Version)
	version = strings.TrimPrefix(version, "v")
	if version == "" {
		return "dev"
	}
	return version
}

func Release() bool {
	parts := strings.Split(Current(), ".")
	if len(parts) != 3 {
		return false
	}
	for _, part := range parts {
		if value, err := strconv.Atoi(part); err != nil || value < 0 {
			return false
		}
	}
	return true
}
