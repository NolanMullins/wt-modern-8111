package wtradio

import "testing"

func TestIsRTB(t *testing.T) {
	for _, message := range []string{
		"Heading to the base.",
		"heading to the base",
		"Returning to the base.",
		"Heading to the airfield.",
	} {
		if !IsRTB(message) {
			t.Errorf("expected %q to be an RTB command", message)
		}
	}
	for _, message := range []string{
		"Guide on me!",
		"Attack the D point!",
		"heading to the base of the mountain and beyond",
	} {
		if IsRTB(message) {
			t.Errorf("did not expect %q to be an RTB command", message)
		}
	}
}

func TestMarkKind(t *testing.T) {
	tests := map[string]string{
		"Guide on me!":                          "guide",
		"Follow me!":                            "guide",
		"Move after me!":                        "guide",
		"On me!":                                "guide",
		"Attention to the map!":                 "attention",
		"Attention to the designated grid zone": "attention",
		"Cover me!":                             "cover",
		"Need help!":                            "help",
		"Need backup!":                          "help",
		"Help me!":                              "help",
	}
	for message, want := range tests {
		got, ok := MarkKind(message)
		if !ok || got != want {
			t.Errorf("MarkKind(%q) = %q, %v; want %q, true", message, got, ok, want)
		}
	}
	if _, ok := MarkKind("Attack the A point!"); ok {
		t.Fatal("attack command must not create an ally mark")
	}
	if _, ok := MarkKind("Cover me in the next match"); ok {
		t.Fatal("ordinary team chat must not create an ally mark")
	}
}

func TestMarkupAndGridParsing(t *testing.T) {
	message := "Guide on me!<color=#FF96966E> [c4, alt. 600 m]</color>"
	if got := StripMarkup(message); got != "Guide on me! [c4, alt. 600 m]" {
		t.Fatalf("StripMarkup() = %q", got)
	}
	if got, ok := ExtractGrid(message); !ok || got != "C4" {
		t.Fatalf("ExtractGrid() = %q, %v; want C4, true", got, ok)
	}
	if got, ok := ExtractAltitudeMeters(message); !ok || got != 600 {
		t.Fatalf("ExtractAltitudeMeters() = %v, %v; want 600, true", got, ok)
	}
	if column, row, ok := ParseGrid("aa12"); !ok || column != "AA" || row != 12 {
		t.Fatalf("ParseGrid() = %q, %d, %v", column, row, ok)
	}
	if _, _, ok := ParseGrid("A0"); ok {
		t.Fatal("row zero must be rejected")
	}
}
