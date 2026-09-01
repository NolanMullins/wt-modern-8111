package identity

import (
	"path/filepath"
	"testing"
)

func TestDeducesCallsignFromVehicleCorrelation(t *testing.T) {
	resolver := NewResolver("")
	// Teammates flying other aircraft must not be selected.
	resolver.Observe("=M22WT= FISHY THUNDER (AV-8C) destroyed Truck", "j_7d")
	if _, confirmed := resolver.Callsign(); confirmed {
		t.Fatal("unrelated vehicle should not confirm a callsign")
	}

	resolver.Observe("=GRIND= DEERSLUG (J-7D) destroyed Studebaker US6", "j_7d")
	if _, confirmed := resolver.Callsign(); confirmed {
		t.Fatal("single sighting should not confirm")
	}

	if !resolver.Observe("=GRIND= DEERSLUG (J-7D) has crashed.", "j_7d") {
		t.Fatal("second corroborating sighting should confirm")
	}
	callsign, confirmed := resolver.Callsign()
	if !confirmed || callsign != "=GRIND= DEERSLUG" {
		t.Fatalf("got %q confirmed=%v", callsign, confirmed)
	}
}

func TestMatchesChatSenderWithoutClanTag(t *testing.T) {
	resolver := NewResolver("")
	resolver.SetCallsign("=GRIND= DEERSLUG")

	if !resolver.Matches("DEERSLUG") {
		t.Fatal("chat sender without clan tag should match")
	}
	if resolver.Matches("FISHY THUNDER") {
		t.Fatal("teammate must not match")
	}
	if resolver.Matches("") {
		t.Fatal("empty sender must not match")
	}
}

func TestMentionsLocalPilotIgnoresSubstringCollisions(t *testing.T) {
	resolver := NewResolver("")
	resolver.SetCallsign("SLUG")

	if resolver.MentionsLocalPilot("=GRIND= DEERSLUG (J-7D) has crashed.") {
		t.Fatal("substring collision must not count as the local pilot")
	}
	if !resolver.MentionsLocalPilot("SLUG (J-7D) has crashed.") {
		t.Fatal("exact actor should match")
	}
}

func TestDetectsLocalPilotLossAsActorOrVictim(t *testing.T) {
	resolver := NewResolver("")
	resolver.SetCallsign("=GRIND= DEERSLUG")

	for _, message := range []string{
		"=GRIND= DEERSLUG (J-7D) has crashed.",
		"airplaneguy24 (MiG-23M) shot down =GRIND= DEERSLUG (J-7D)",
		"airplaneguy24 (MiG-23M) destroyed =GRIND= DEERSLUG (J-7D)",
	} {
		if !resolver.IsLocalPilotLoss(message) {
			t.Errorf("expected local loss for %q", message)
		}
	}

	for _, message := range []string{
		"=GRIND= DEERSLUG (J-7D) shot down -TRFEM- StonksMeister (AJS37)",
		"airplaneguy24 (MiG-23M) critically damaged =GRIND= DEERSLUG (J-7D)",
	} {
		if resolver.IsLocalPilotLoss(message) {
			t.Errorf("did not expect local loss for %q", message)
		}
	}
}

func TestPersistsAndReloadsCallsign(t *testing.T) {
	path := filepath.Join(t.TempDir(), "identity.json")
	NewResolver(path).SetCallsign("=GRIND= DEERSLUG")

	callsign, confirmed := NewResolver(path).Callsign()
	if !confirmed || callsign != "=GRIND= DEERSLUG" {
		t.Fatalf("reload got %q confirmed=%v", callsign, confirmed)
	}
}

func TestVehicleMatchesIgnoresSeparatorsAndDecorations(t *testing.T) {
	cases := []struct {
		display   string
		indicator string
		want      bool
	}{
		{"J-7D", "j_7d", true},
		{"◊MiG-21MF", "mig_21mf", true},
		{"AV-8C", "j_7d", false},
	}
	for _, testCase := range cases {
		if got := vehicleMatches(testCase.display, testCase.indicator); got != testCase.want {
			t.Errorf("vehicleMatches(%q, %q) = %v", testCase.display, testCase.indicator, got)
		}
	}
}
