package buildinfo

import "testing"

func TestCurrentNormalizesVersion(t *testing.T) {
	previous := Version
	t.Cleanup(func() { Version = previous })

	Version = "v1.2.3"
	if got := Current(); got != "1.2.3" {
		t.Fatalf("Current() = %q, want 1.2.3", got)
	}

	if !Release() {
		t.Fatal("semantic version was not recognized as a release")
	}
}

func TestDevelopmentVersionIsNotRelease(t *testing.T) {
	previous := Version
	t.Cleanup(func() { Version = previous })

	for _, version := range []string{"dev", "1.0.0-dev", "commit-sha"} {
		Version = version
		if Release() {
			t.Errorf("Release() accepted %q", version)
		}
	}
}

func TestShortRevision(t *testing.T) {
	previous := Revision
	t.Cleanup(func() { Revision = previous })

	Revision = "0123456789abcdef"
	if got := ShortRevision(); got != "0123456789ab" {
		t.Fatalf("ShortRevision() = %q", got)
	}
}
