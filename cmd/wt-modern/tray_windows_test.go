//go:build windows

package main

import (
	"sync/atomic"
	"testing"
)

func TestDashboardClickSuppressesReleaseAfterDoubleClick(t *testing.T) {
	var suppressNext atomic.Bool
	if !shouldOpenDashboard(&suppressNext) {
		t.Fatal("first release should open the dashboard")
	}

	suppressNext.Store(true)
	if shouldOpenDashboard(&suppressNext) {
		t.Fatal("release after double-click should be suppressed")
	}
	if !shouldOpenDashboard(&suppressNext) {
		t.Fatal("suppression should apply to only one release")
	}
}
