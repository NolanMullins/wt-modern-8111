//go:build !windows

package main

import (
	"context"
)

func traySupported() bool {
	return false
}

func runTray(
	context.Context,
	string,
	context.CancelFunc,
	<-chan error,
) error {
	return nil
}
