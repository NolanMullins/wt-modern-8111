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
		"Attention to the map!":                 "attention",
		"Attention to the designated grid zone": "attention",
		"Cover me!":                             "cover",
		"Need help!":                            "help",
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
}

func TestMarkupAndGridParsing(t *testing.T) {
	message := "Attention to the map!<color=#FF96966E> [c4]</color>"
	if got := StripMarkup(message); got != "Attention to the map! [c4]" {
		t.Fatalf("StripMarkup() = %q", got)
	}
	if got, ok := ExtractGrid(message); !ok || got != "C4" {
		t.Fatalf("ExtractGrid() = %q, %v; want C4, true", got, ok)
	}
	if column, row, ok := ParseGrid("aa12"); !ok || column != "AA" || row != 12 {
		t.Fatalf("ParseGrid() = %q, %d, %v", column, row, ok)
	}
	if _, _, ok := ParseGrid("A0"); ok {
		t.Fatal("row zero must be rejected")
	}
}
