//go:build windows

package updater

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"golang.org/x/sys/windows"
)

func Launch(stagedPath string) error {
	target, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate installed executable: %w", err)
	}
	command := exec.Command(
		stagedPath,
		"-apply-update", target,
		"-wait-pid", strconv.Itoa(os.Getpid()),
	)
	if err := command.Start(); err != nil {
		return fmt.Errorf("start update helper: %w", err)
	}
	return nil
}

func Restart(target string) error {
	command := exec.Command(target, "-open=false")
	if err := command.Start(); err != nil {
		return fmt.Errorf("restart application: %w", err)
	}
	return nil
}

func Apply(target string, processID uint32) error {
	if target == "" || processID == 0 {
		return fmt.Errorf("invalid update target")
	}
	if err := waitForProcess(processID, time.Minute); err != nil {
		return err
	}
	staged, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate staged executable: %w", err)
	}
	target, err = filepath.Abs(target)
	if err != nil {
		return fmt.Errorf("resolve update target: %w", err)
	}
	backup, err := replaceExecutable(staged, target)
	if err != nil {
		return err
	}
	command := exec.Command(target, "-open=false", "-cleanup-update", staged)
	if err := command.Start(); err != nil {
		if rollbackErr := os.Rename(backup, target); rollbackErr != nil {
			return fmt.Errorf(
				"restart updated application: %v; restore previous version: %w",
				err,
				rollbackErr,
			)
		}
		return fmt.Errorf("restart updated application: %w", err)
	}
	return nil
}

func replaceExecutable(staged, target string) (string, error) {
	next := target + ".new"
	backup := target + ".old"
	_ = os.Remove(next)
	_ = os.Remove(backup)
	if err := copyExecutable(staged, next); err != nil {
		return "", err
	}
	if err := copyExecutable(target, backup); err != nil {
		_ = os.Remove(next)
		return "", fmt.Errorf("back up installed executable: %w", err)
	}
	if err := os.Rename(next, target); err != nil {
		_ = os.Remove(backup)
		return "", fmt.Errorf("activate update: %w", err)
	}
	return backup, nil
}

func Cleanup(stagedPath string) {
	if stagedPath == "" {
		return
	}
	current, err := os.Executable()
	if err != nil {
		return
	}
	go func() {
		time.Sleep(3 * time.Second)
		_ = os.Remove(stagedPath)
		_ = os.Remove(current + ".old")
	}()
}

func waitForProcess(processID uint32, timeout time.Duration) error {
	process, err := windows.OpenProcess(windows.SYNCHRONIZE, false, processID)
	if err != nil {
		if err == windows.ERROR_INVALID_PARAMETER {
			return nil
		}
		return fmt.Errorf("open running application: %w", err)
	}
	defer windows.CloseHandle(process)
	result, err := windows.WaitForSingleObject(process, uint32(timeout.Milliseconds()))
	if err != nil {
		return fmt.Errorf("wait for running application: %w", err)
	}
	if result == uint32(windows.WAIT_TIMEOUT) {
		return fmt.Errorf("running application did not exit before update timeout")
	}
	if result != uint32(windows.WAIT_OBJECT_0) {
		return fmt.Errorf("unexpected process wait result %d", result)
	}
	return nil
}

func copyExecutable(source, destination string) error {
	input, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("open staged executable: %w", err)
	}
	defer input.Close()
	output, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o755)
	if err != nil {
		return fmt.Errorf("create replacement executable: %w", err)
	}
	_, copyErr := io.Copy(output, input)
	syncErr := output.Sync()
	closeErr := output.Close()
	if copyErr != nil {
		return fmt.Errorf("copy replacement executable: %w", copyErr)
	}
	if syncErr != nil {
		return fmt.Errorf("sync replacement executable: %w", syncErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close replacement executable: %w", closeErr)
	}
	return nil
}
