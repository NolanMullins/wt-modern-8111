//go:build !windows

package main

func configureLogging() func() {
	return func() {}
}
