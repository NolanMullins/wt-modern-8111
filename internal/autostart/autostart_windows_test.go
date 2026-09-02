//go:build windows

package autostart

import (
	"path/filepath"
	"testing"
)

func TestStableExecutableRejectsGoRunBinary(t *testing.T) {
	temporary := filepath.Join(`C:\Users\pilot\AppData\Local`, "Temp")
	executable := filepath.Join(temporary, "go-build123", "exe", "wt-modern.exe")

	if _, err := stableExecutablePath(executable, temporary); err == nil {
		t.Fatal("temporary go-run executable was accepted for autostart")
	}
}

func TestStableExecutableAcceptsInstalledBinary(t *testing.T) {
	temporary := filepath.Join(`C:\Users\pilot\AppData\Local`, "Temp")
	executable := filepath.Join(
		`C:\Users\pilot\AppData\Local`,
		"Programs",
		"WT Modern 8111",
		"wt-modern.exe",
	)

	if got, err := stableExecutablePath(executable, temporary); err != nil || got != executable {
		t.Fatalf("stableExecutablePath() = %q, %v", got, err)
	}
}
