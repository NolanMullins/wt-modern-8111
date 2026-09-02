package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/NolanMullins/wt-modern-8111/internal/polling"
	appserver "github.com/NolanMullins/wt-modern-8111/internal/server"
	"github.com/NolanMullins/wt-modern-8111/internal/warthunder"
)

func main() {
	var (
		address        = flag.String("address", "127.0.0.1:17711", "HTTP listen address")
		wtURL          = flag.String("wt-url", "http://127.0.0.1:8111", "War Thunder service URL")
		fixtureDir     = flag.String("fixture", "", "directory containing captured JSON fixtures")
		openUI         = flag.Bool("open", true, "open the dashboard in the default browser")
		callsign       = flag.String("callsign", "", "set and remember the local pilot callsign")
		forgetCallsign = flag.Bool("forget-callsign", false, "clear the remembered pilot callsign")
	)
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var (
		service *polling.Service
		err     error
	)
	if *fixtureDir != "" {
		service, err = polling.NewFixtureService(*fixtureDir)
	} else {
		service = polling.NewService(warthunder.NewClient(*wtURL, 300*time.Millisecond))
	}
	if err != nil {
		log.Fatalf("initialize telemetry service: %v", err)
	}
	if *forgetCallsign {
		if err := service.ClearCallsign(); err != nil {
			log.Fatalf("clear callsign: %v", err)
		}
	}
	if *callsign != "" {
		service.SetCallsign(*callsign)
	}
	service.Start(ctx)

	handler, err := appserver.New(service)
	if err != nil {
		log.Fatalf("initialize HTTP server: %v", err)
	}

	server := &http.Server{
		Addr:              *address,
		Handler:           handler,
		BaseContext:       func(net.Listener) context.Context { return ctx },
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	errs := make(chan error, 1)
	go func() {
		log.Printf("WT Modern 8111 listening at http://%s", *address)
		errs <- server.ListenAndServe()
	}()

	if *openUI {
		go func() {
			timer := time.NewTimer(300 * time.Millisecond)
			defer timer.Stop()
			select {
			case <-ctx.Done():
			case <-timer.C:
				if err := openBrowser("http://" + *address); err != nil {
					log.Printf("open browser: %v", err)
				}
			}
		}()
	}

	select {
	case <-ctx.Done():
	case err := <-errs:
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("serve: %v", err)
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}

func openBrowser(url string) error {
	var command *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		command = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		command = exec.Command("open", url)
	default:
		command = exec.Command("xdg-open", url)
	}
	if err := command.Start(); err != nil {
		return fmt.Errorf("start browser command: %w", err)
	}
	return nil
}
