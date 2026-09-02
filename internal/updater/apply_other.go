//go:build !windows

package updater

import "errors"

var errUnsupported = errors.New("automatic updates are only available on Windows")

func Launch(string) error {
	return errUnsupported
}

func Apply(string, uint32) error {
	return errUnsupported
}

func Restart(string) error {
	return errUnsupported
}

func Cleanup(string) {}
