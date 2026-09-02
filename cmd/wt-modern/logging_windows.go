//go:build windows

package main

import (
	"io"
	"log"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"
)

var getConsoleWindow = windows.NewLazySystemDLL("kernel32.dll").NewProc("GetConsoleWindow")

func configureLogging() func() {
	if console, _, _ := getConsoleWindow.Call(); console != 0 {
		return func() {}
	}
	directory := os.Getenv("LOCALAPPDATA")
	if directory == "" {
		return func() {}
	}
	directory = filepath.Join(directory, "wt-modern-8111")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return func() {}
	}
	path := filepath.Join(directory, "wt-modern.log")
	if info, err := os.Stat(path); err == nil && info.Size() > 2<<20 {
		_ = os.Remove(path + ".old")
		_ = os.Rename(path, path+".old")
	}
	file, err := os.OpenFile(
		path,
		os.O_CREATE|os.O_APPEND|os.O_WRONLY,
		0o644,
	)
	if err != nil {
		return func() {}
	}
	log.SetOutput(file)
	return func() {
		log.SetOutput(io.Discard)
		_ = file.Close()
	}
}
