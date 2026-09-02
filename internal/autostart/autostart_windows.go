//go:build windows

package autostart

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

const runKeyPath = `Software\Microsoft\Windows\CurrentVersion\Run`

func Enabled() (bool, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.QUERY_VALUE)
	if errors.Is(err, registry.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	defer key.Close()
	value, _, err := key.GetStringValue(applicationName)
	if errors.Is(err, registry.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	executable, err := stableExecutable()
	if err != nil {
		return false, err
	}
	return strings.EqualFold(value, command(executable)), nil
}

func Set(enabled bool) error {
	if enabled {
		key, _, err := registry.CreateKey(
			registry.CURRENT_USER,
			runKeyPath,
			registry.QUERY_VALUE|registry.SET_VALUE,
		)
		if err != nil {
			return err
		}
		defer key.Close()
		executable, err := stableExecutable()
		if err != nil {
			return err
		}
		return key.SetStringValue(applicationName, command(executable))
	}

	key, err := registry.OpenKey(
		registry.CURRENT_USER,
		runKeyPath,
		registry.QUERY_VALUE|registry.SET_VALUE,
	)
	if errors.Is(err, registry.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer key.Close()
	err = key.DeleteValue(applicationName)
	if errors.Is(err, registry.ErrNotExist) {
		return nil
	}
	return err
}

func stableExecutable() (string, error) {
	executable, err := os.Executable()
	if err != nil {
		return "", err
	}
	return stableExecutablePath(executable, os.TempDir())
}

func stableExecutablePath(executable, temporary string) (string, error) {
	relative, err := filepath.Rel(temporary, executable)
	if err == nil && relative != "." &&
		relative != ".." &&
		!strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("build the Windows executable before enabling automatic startup")
	}
	return executable, nil
}
