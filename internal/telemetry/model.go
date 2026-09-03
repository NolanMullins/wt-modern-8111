package telemetry

import (
	"time"

	"github.com/NolanMullins/wt-modern-8111/internal/warthunder"
)

type Snapshot struct {
	Version    int         `json:"version"`
	Sequence   uint64      `json:"sequence"`
	CapturedAt time.Time   `json:"capturedAt"`
	Connection Connection  `json:"connection"`
	Vehicle    Vehicle     `json:"vehicle"`
	Flight     Flight      `json:"flight"`
	Ground     Ground      `json:"ground"`
	Systems    Systems     `json:"systems"`
	Navigation *Navigation `json:"navigation"`
	Mission    Mission     `json:"mission"`
	Map        Map         `json:"map"`
	Feed       []FeedEntry `json:"feed"`
	Pilot      Pilot       `json:"pilot"`
	AllyMarks  []AllyMark  `json:"allyMarks"`
}

// Pilot describes the locally deduced identity of the player.
type Pilot struct {
	Callsign  string `json:"callsign,omitempty"`
	Confirmed bool   `json:"confirmed"`
}

// AllyMark is a time-limited callout raised by a teammate radio command.
type AllyMark struct {
	Key       string    `json:"key"`
	Kind      string    `json:"kind"`
	Sender    string    `json:"sender"`
	Message   string    `json:"message"`
	Grid      string    `json:"grid,omitempty"`
	X         *float64  `json:"x,omitempty"`
	Y         *float64  `json:"y,omitempty"`
	Located   bool      `json:"located"`
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type Connection struct {
	State   string                  `json:"state"`
	Mode    string                  `json:"mode"`
	Sources map[string]SourceStatus `json:"sources"`
}

type SourceStatus struct {
	State       string     `json:"state"`
	LastSuccess *time.Time `json:"lastSuccess,omitempty"`
	AgeMS       *int64     `json:"ageMs,omitempty"`
	Error       string     `json:"error,omitempty"`
}

type Vehicle struct {
	Type  string `json:"type,omitempty"`
	Class string `json:"class,omitempty"`
}

type Flight struct {
	IASKMH           *float64 `json:"iasKmh,omitempty"`
	TASKMH           *float64 `json:"tasKmh,omitempty"`
	AltitudeM        *float64 `json:"altitudeM,omitempty"`
	RadioAltitudeM   *float64 `json:"radioAltitudeM,omitempty"`
	Mach             *float64 `json:"mach,omitempty"`
	HeadingDeg       *float64 `json:"headingDeg,omitempty"`
	VerticalSpeedMPS *float64 `json:"verticalSpeedMps,omitempty"`
	AOADeg           *float64 `json:"aoaDeg,omitempty"`
	GLoad            *float64 `json:"gLoad,omitempty"`
}

type Ground struct {
	SpeedKMH      *float64 `json:"speedKmh,omitempty"`
	HeadingDeg    *float64 `json:"headingDeg,omitempty"`
	EngineRPM     *float64 `json:"engineRpm,omitempty"`
	Gear          *float64 `json:"gear,omitempty"`
	CruiseControl *float64 `json:"cruiseControl,omitempty"`
	Ammo          *float64 `json:"ammo,omitempty"`
	CrewCurrent   *float64 `json:"crewCurrent,omitempty"`
	CrewTotal     *float64 `json:"crewTotal,omitempty"`
	DriverState   *float64 `json:"driverState,omitempty"`
	GunnerState   *float64 `json:"gunnerState,omitempty"`
	Stabilizer    *float64 `json:"stabilizer,omitempty"`
	LWS           *float64 `json:"lws,omitempty"`
	IRCM          *float64 `json:"ircm,omitempty"`
	EngineBroken  *float64 `json:"engineBroken,omitempty"`
	SpeedWarning  *float64 `json:"speedWarning,omitempty"`
}

type Systems struct {
	Status          string   `json:"status"`
	Severity        string   `json:"severity"`
	Warnings        []string `json:"warnings"`
	Engines         []Engine `json:"engines"`
	FuelKG          *float64 `json:"fuelKg,omitempty"`
	FuelPercent     *float64 `json:"fuelPercent,omitempty"`
	GearPercent     *float64 `json:"gearPercent,omitempty"`
	FlapsPercent    *float64 `json:"flapsPercent,omitempty"`
	AirbrakePercent *float64 `json:"airbrakePercent,omitempty"`
}

type Engine struct {
	Index           int      `json:"index"`
	ThrottlePercent *float64 `json:"throttlePercent,omitempty"`
	RPM             *float64 `json:"rpm,omitempty"`
	OilTempC        *float64 `json:"oilTempC,omitempty"`
	ThrustKGF       *float64 `json:"thrustKgf,omitempty"`
	Status          string   `json:"status"`
	Running         bool     `json:"running"`
}

type Navigation struct {
	Name       string   `json:"name"`
	BearingDeg float64  `json:"bearingDeg"`
	RangeKM    float64  `json:"rangeKm"`
	ETASeconds *float64 `json:"etaSeconds,omitempty"`
	TargetX    float64  `json:"targetX"`
	TargetY    float64  `json:"targetY"`
	Basis      string   `json:"basis"`
}

type Mission struct {
	Status     string                 `json:"status,omitempty"`
	Objectives []warthunder.Objective `json:"objectives"`
}

type Map struct {
	Valid         bool                   `json:"valid"`
	Generation    int                    `json:"generation,omitempty"`
	ImageRevision int                    `json:"imageRevision,omitempty"`
	HUDType       int                    `json:"hudType,omitempty"`
	GridSize      []float64              `json:"gridSize,omitempty"`
	GridSteps     []float64              `json:"gridSteps,omitempty"`
	GridZero      []float64              `json:"gridZero,omitempty"`
	MapMin        []float64              `json:"mapMin,omitempty"`
	MapMax        []float64              `json:"mapMax,omitempty"`
	Objects       []warthunder.MapObject `json:"objects"`
	Counts        MapCounts              `json:"counts"`
}

type MapCounts struct {
	Total          int `json:"total"`
	HostileAir     int `json:"hostileAir"`
	Ground         int `json:"ground"`
	FriendlyGround int `json:"friendlyGround"`
	HostileGround  int `json:"hostileGround"`
	AirDefense     int `json:"airDefense"`
	CaptureZone    int `json:"captureZone"`
	GroundSpawn    int `json:"groundSpawn"`
	StrikePoint    int `json:"strikePoint"`
	Airfield       int `json:"airfield"`
}

type FeedEntry struct {
	Key     string    `json:"key"`
	Kind    string    `json:"kind"`
	Time    float64   `json:"time,omitempty"`
	AddedAt time.Time `json:"addedAt"`
	Sender  string    `json:"sender,omitempty"`
	Message string    `json:"message"`
	Enemy   bool      `json:"enemy,omitempty"`
}

type RawData struct {
	State            map[string]any
	Indicators       map[string]any
	MapInfo          warthunder.MapInfo
	MapObjects       []warthunder.MapObject
	MapImageRevision int
	Mission          warthunder.Mission
	Feed             []FeedEntry
	ReturnToAirfield bool
	Pilot            Pilot
	AllyMarks        []AllyMark
	Destroyed        bool
}
