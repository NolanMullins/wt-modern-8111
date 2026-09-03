package polling

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/NolanMullins/wt-modern-8111/internal/identity"
	"github.com/NolanMullins/wt-modern-8111/internal/warthunder"
)

func TestSameGenerationGroundCASTransitionKeepsCanonicalGroundMap(t *testing.T) {
	type viewState struct {
		army    string
		vehicle string
		image   string
		min     float64
		max     float64
		step    float64
	}
	var (
		mu    sync.RWMutex
		state = viewState{
			army:    "tank",
			vehicle: "tankModels/us_m1_abrams",
			image:   "ground-one",
			min:     0,
			max:     4096,
			step:    225,
		}
	)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		mu.RLock()
		current := state
		mu.RUnlock()
		switch request.URL.Path {
		case "/indicators":
			writer.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintf(
				writer,
				`{"valid":true,"army":%q,"type":%q}`,
				current.army,
				current.vehicle,
			)
		case "/map_info.json":
			writer.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintf(
				writer,
				`{
					"valid":true,
					"map_generation":7,
					"map_min":[%f,%f],
					"map_max":[%f,%f],
					"grid_steps":[%f,%f]
				}`,
				current.min,
				current.min,
				current.max,
				current.max,
				current.step,
				current.step,
			)
		case "/map.img":
			writer.Header().Set("Content-Type", "image/jpeg")
			_, _ = writer.Write([]byte(current.image))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	service := NewServiceWithIdentity(
		warthunder.NewClient(server.URL, time.Second),
		identity.NewResolver(""),
	)
	pollView := func() {
		t.Helper()
		service.pollIndicators(context.Background())
		service.pollMapInfo(context.Background())
	}

	pollView()
	assertMapImages(t, service, "ground-one", "ground-one")

	mu.Lock()
	state = viewState{
		army:    "air",
		vehicle: "f_16c",
		image:   "expanded-cas",
		min:     -32768,
		max:     32768,
		step:    6500,
	}
	mu.Unlock()
	pollView()
	assertMapImages(t, service, "expanded-cas", "ground-one")

	mu.Lock()
	state = viewState{
		army:    "tank",
		vehicle: "tankModels/us_m1_abrams",
		image:   "ground-two",
		min:     0,
		max:     4096,
		step:    225,
	}
	mu.Unlock()
	pollView()
	assertMapImages(t, service, "ground-two", "ground-two")
}

func assertMapImages(t *testing.T, service *Service, currentWant, groundWant string) {
	t.Helper()
	current, _, _, ok := service.MapImage()
	if !ok || string(current) != currentWant {
		t.Fatalf("current map = %q available=%t, want %q", current, ok, currentWant)
	}
	ground, _, _, ok := service.GroundMapImage()
	if !ok || string(ground) != groundWant {
		t.Fatalf("ground map = %q available=%t, want %q", ground, ok, groundWant)
	}
}
