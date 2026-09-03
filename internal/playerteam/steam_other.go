//go:build !windows

package playerteam

import (
	"os"
	"path/filepath"
	"runtime"
)

func platformSteamRoots() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	if runtime.GOOS == "darwin" {
		return []string{
			filepath.Join(home, "Library", "Application Support", "Steam"),
		}
	}
	return []string{
		filepath.Join(home, ".local", "share", "Steam"),
		filepath.Join(home, ".steam", "steam"),
		filepath.Join(
			home,
			".var",
			"app",
			"com.valvesoftware.Steam",
			".local",
			"share",
			"Steam",
		),
	}
}
