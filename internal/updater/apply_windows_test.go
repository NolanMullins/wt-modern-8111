//go:build windows

package updater

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReplaceExecutableKeepsRollbackCopy(t *testing.T) {
	directory := t.TempDir()
	staged := filepath.Join(directory, "staged.exe")
	target := filepath.Join(directory, "installed.exe")
	if err := os.WriteFile(staged, []byte("new version"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("old version"), 0o755); err != nil {
		t.Fatal(err)
	}

	backup, err := replaceExecutable(staged, target)
	if err != nil {
		t.Fatal(err)
	}
	installed, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	previous, err := os.ReadFile(backup)
	if err != nil {
		t.Fatal(err)
	}
	if string(installed) != "new version" || string(previous) != "old version" {
		t.Fatalf("installed=%q backup=%q", installed, previous)
	}
}
