//go:build windows

package main

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/NolanMullins/wt-modern-8111/internal/autostart"
	"github.com/gogpu/systray"
	"golang.org/x/sys/windows"
)

var getDoubleClickTime = windows.NewLazySystemDLL("user32.dll").NewProc("GetDoubleClickTime")

func traySupported() bool {
	return true
}

func runTray(
	ctx context.Context,
	dashboardURL string,
	stop context.CancelFunc,
	serverErrors <-chan error,
) error {
	tray := systray.New()
	var lastDashboardOpen atomic.Int64
	doubleClickMilliseconds := systemDoubleClickMilliseconds()
	openDashboard := func() {
		now := time.Now().UnixMilli()
		previous := lastDashboardOpen.Swap(now)
		if previous != 0 && now-previous <= doubleClickMilliseconds {
			return
		}
		if err := openBrowser(dashboardURL); err != nil {
			tray.ShowNotification("WT Modern 8111", "Could not open the dashboard.")
		}
	}
	menu := systray.NewMenu()
	menu.Add("Open Dashboard", openDashboard)
	menu.AddSeparator()

	startupEnabled, startupErr := autostart.Enabled()
	var startupItem *systray.MenuItem
	startupItem = menu.AddCheckbox("Start with Windows", startupEnabled, func() {
		next := !startupItem.IsChecked()
		if err := autostart.Set(next); err != nil {
			tray.ShowNotification("WT Modern 8111", "Could not update the Windows startup setting.")
			return
		}
		startupItem.SetChecked(next)
	})
	startupItem.SetDisabled(startupErr != nil)
	menu.AddSeparator()
	menu.Add("Quit", func() {
		stop()
	})

	icon := trayIcon()
	tray.SetIcon(icon).
		SetDarkModeIcon(icon).
		SetTooltip("WT Modern 8111").
		SetMenu(menu)
	tray.OnClick(openDashboard)
	if !showTrayIcon(tray, 30*time.Second) {
		stop()
		tray.Remove()
		return fmt.Errorf("tray icon could not be created")
	}

	serverResult := make(chan error, 1)
	go func() {
		select {
		case <-ctx.Done():
			tray.Remove()
		case err := <-serverErrors:
			if err != nil && err != http.ErrServerClosed {
				serverResult <- fmt.Errorf("serve: %w", err)
			}
			stop()
			tray.Remove()
		}
	}()

	if err := tray.Run(); err != nil {
		return err
	}
	select {
	case err := <-serverResult:
		return err
	default:
		return nil
	}
}

func systemDoubleClickMilliseconds() int64 {
	milliseconds, _, _ := getDoubleClickTime.Call()
	if milliseconds == 0 {
		return 500
	}
	return int64(milliseconds)
}

func showTrayIcon(tray *systray.SystemTray, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		tray.Show()
		_, _, width, height := tray.Bounds()
		if width > 0 && height > 0 {
			return true
		}
		time.Sleep(250 * time.Millisecond)
	}
	return false
}

func trayIcon() []byte {
	const size = 32
	icon := image.NewRGBA(image.Rect(0, 0, size, size))
	dark := color.RGBA{R: 17, G: 21, B: 25, A: 255}
	red := color.RGBA{R: 215, G: 59, B: 50, A: 255}
	white := color.RGBA{R: 243, G: 244, B: 242, A: 255}
	for y := 2; y < size-2; y++ {
		for x := 2; x < size-2; x++ {
			icon.SetRGBA(x, y, dark)
		}
	}
	for y := 2; y < size-2; y++ {
		for x := 2; x < 7; x++ {
			icon.SetRGBA(x, y, red)
		}
	}
	for x := 11; x < 28; x++ {
		for y := 8; y < 12; y++ {
			icon.SetRGBA(x, y, white)
		}
	}
	for x := 18; x < 22; x++ {
		for y := 8; y < 25; y++ {
			icon.SetRGBA(x, y, white)
		}
	}
	var buffer bytes.Buffer
	_ = png.Encode(&buffer, icon)
	return buffer.Bytes()
}
