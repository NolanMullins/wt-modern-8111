package heatmap

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestResolveKnownAndFuzzyMapHash(t *testing.T) {
	known := groundMapHashes[1]
	resolved, err := resolveHash(known.hash)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Level != "levels/avg_abandoned_town.bin" || resolved.Name != "Abandoned Town" {
		t.Fatalf("resolved map = %#v", resolved)
	}

	fuzzy := "0" + known.hash[1:]
	resolved, err = resolveHash(fuzzy)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Level != "levels/avg_abandoned_town.bin" {
		t.Fatalf("fuzzy map = %#v", resolved)
	}

	cas := airInGroundMapHashes[3]
	resolved, err = resolveHash(cas.hash)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Level != "levels/avg_american_valley.bin" ||
		resolved.Name != "American Desert" {
		t.Fatalf("CAS map = %#v", resolved)
	}
}

func TestResolveLiveMapWhenRequested(t *testing.T) {
	endpoint := os.Getenv("WT_MAP_IMAGE_URL")
	if endpoint == "" {
		t.Skip("WT_MAP_IMAGE_URL is not set")
	}
	response, err := http.Get(endpoint)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := ResolveMap(body)
	if err != nil {
		if errors.Is(err, ErrUnknownMap) {
			t.Logf("live map is not in the Ground Realistic heatmap catalog: %v", err)
			return
		}
		t.Fatal(err)
	}
	t.Logf("resolved live map as %s (%s)", resolved.Name, resolved.Level)
}

func TestDifferenceHashUsesHorizontalComparisons(t *testing.T) {
	source := image.NewGray(image.Rect(0, 0, 16, 15))
	for y := range 15 {
		for x := range 16 {
			source.SetGray(x, y, color.Gray{Y: uint8(x * 16)})
		}
	}
	var body bytes.Buffer
	if err := png.Encode(&body, source); err != nil {
		t.Fatal(err)
	}

	hash, err := differenceHash(body.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if hash != strings.Repeat("0", 57) {
		t.Fatalf("hash = %q", hash)
	}
}
