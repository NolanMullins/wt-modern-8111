//go:build !windows

package autostart

import "errors"

var errUnsupported = errors.New("automatic startup is only available on Windows")

func Enabled() (bool, error) {
	return false, errUnsupported
}

func Set(bool) error {
	return errUnsupported
}
