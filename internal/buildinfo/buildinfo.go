package buildinfo

import (
	"strconv"
	"strings"
)

// Version is replaced from a release tag through the Go linker.
var Version = "dev"

// Revision is replaced with the source commit through the Go linker.
var Revision = "unknown"

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

func ShortRevision() string {
	revision := strings.TrimSpace(Revision)
	if revision == "" {
		return "unknown"
	}
	if len(revision) > 12 {
		return revision[:12]
	}
	return revision
}
