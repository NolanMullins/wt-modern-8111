package autostart

import "testing"

func TestCommandQuotesExecutableAndDisablesBrowserLaunch(t *testing.T) {
	got := command(`C:\Program Files\WT Modern\wt-modern.exe`)
	want := `"C:\Program Files\WT Modern\wt-modern.exe" -open=false`
	if got != want {
		t.Fatalf("command = %q, want %q", got, want)
	}
}
