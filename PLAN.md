# WT Modern 8111 plan

Last updated: 2026-09-01

## Product objective

Build a map-first electronic flight bag for War Thunder Simulator pilots that
is substantially easier to read and operate than the built-in `localhost:8111`
interface.

The first release should require no command line, development runtime, account,
or cloud service. The intended experience is:

> Install it, launch it, and it works.

### Passive second-screen constraint

The pilot may be unable to interact with WT Modern while actively flying. The
primary experience must therefore work as a passive, glanceable display:

- The map follows the aircraft and maintains a useful tactical framing
  automatically.
- Active mission objectives, chat/events, navigation, engine state, primary
  flight data, and connection health remain visible without opening menus.
- Context changes automatically when mission, aircraft, or selected-destination
  data changes; destinations themselves are selected explicitly.
- Touch and pointer interactions are optional setup conveniences, not required
  for normal in-flight use.
- Controls used during flight must be large and sparse. Radio RTB remains the
  only automatic navigation override.

## First milestone

1. Connect to War Thunder's local `8111` service.
2. Display the live tactical map.
3. Show player position and heading.
4. Render exposed map objects and objectives.
5. Present a compact primary flight-data strip.
6. Support mouse and touch pan and zoom.
7. Allow explicit selection of a target, supported mission objective, or
   arbitrary map point.
8. Calculate bearing and distance to the active destination.
9. Work well on landscape desktop and tablet displays.
10. Handle offline, hangar, loading, live, and degraded states.

Aircraft-specific layouts, performance-envelope warnings, flight recording,
and advanced route planning are outside this milestone.

## Architecture decision

WT Modern 8111 will be a lightweight local companion application with an
embedded responsive web frontend.

```text
War Thunder
localhost:8111
      |
      | fixed, allowlisted HTTP polling
      v
WT Modern companion - one executable
+--------------------------------------+
| Pollers and connection state         |
| Payload validation and normalization |
| Immutable latest snapshot            |
| Map-image cache                      |
| Local HTTP/SSE server :17711         |
| Embedded web frontend                |
+--------------------------------------+
      |
      +--> Same PC: http://127.0.0.1:17711
      |
      +--> Tablet: http://<gaming-pc-ip>:17711
```

The browser never polls War Thunder directly. The companion is the only
component that communicates with `8111`.

### Why this architecture

- The end user installs and launches one application.
- No Node.js, Python, Docker, or command-line setup is required.
- War Thunder's irregular payloads are normalized in one place.
- All browser clients share one polling workload and cached map image.
- The UI remains usable during temporary endpoint failures.
- The same responsive frontend works on a second monitor and tablet.
- Telemetry remains on the local machine and trusted LAN.
- The frontend can be developed independently against a mock War Thunder
  server.

### Approaches not selected

- **Direct static frontend:** possible in some configurations, but exposes the
  raw API, duplicates normalization in the browser, complicates address
  discovery, and encounters mixed-content restrictions.
- **Cloud-hosted frontend or backend:** unnecessary, cannot naturally access
  local telemetry, and weakens privacy.
- **Electron:** offers convenient native-window support but adds a bundled
  Chromium runtime and substantial resource and download overhead.
- **Docker:** inappropriate for the intended nontechnical end user.
- **Windows service:** creates administrator, installation, and lifecycle
  complexity without helping the first milestone.
- **Database in the first milestone:** configuration and transient telemetry do
  not yet require one.

## Technology stack

### Companion service

- **Language:** Go
- **Distribution:** one native executable with embedded frontend assets
- **HTTP:** Go's standard HTTP stack
- **Live delivery:** Server-Sent Events carrying normalized JSON snapshots
- **Concurrency:** independent endpoint pollers publishing immutable snapshots
- **Configuration:** JSON under the user's local application-data directory

Go is selected for its small deployment footprint, straightforward concurrency,
low idle resource use, and lack of a runtime installation requirement.

### Frontend

- **Language:** TypeScript
- **UI:** React
- **Build:** Vite
- **Layout:** responsive CSS Grid
- **Map rendering:** Canvas 2D
- **Transport:** Server-Sent Events with an HTTP snapshot fallback

The tactical map is a static raster plus normalized objects, not a geographic
tile set. A Canvas renderer avoids unnecessary GIS abstractions and decouples
high-frequency map drawing from React component rendering.

## End-user experience

### Same-PC flow

1. Install WT Modern 8111.
2. Launch it from the Start menu.
3. The companion starts in the background.
4. The dashboard opens in the default browser.
5. The application detects War Thunder and reconnects automatically.

Launching the executable again should open the existing dashboard instead of
starting a second companion.

A production tray menu should provide:

- Open dashboard
- Current connection state
- Connect another device
- Start with Windows
- Quit

### Tablet flow

The dashboard provides a **Connect another device** action that:

1. Explicitly enables LAN access.
2. Selects an appropriate private network interface.
3. Displays a QR code and short URL.
4. Explains any Windows Firewall prompt.
5. Uses a temporary pairing token.

LAN mode is opt-in. The application must never instruct users to configure
router port forwarding or expose the service to the internet.

### Offline behavior

The dashboard opens even if War Thunder is unavailable. It presents a useful
state rather than a raw network error:

```text
starting
  -> waiting-for-game
  -> hangar-or-no-vehicle
  -> loading-mission
  -> live
```

Any active state can become `degraded` when one or more sources fail or become
stale. Recovery is automatic.

## Companion service design

### War Thunder client

Only explicit upstream paths are supported:

```text
/state
/indicators
/map_info.json
/map_obj.json
/map.img
/mission.json
/hudmsg
/gamechat
```

Do not implement a generic user-controlled proxy path.

### Polling schedule

| Group | Endpoints | Initial rate |
| --- | --- | --- |
| Fast | `/state`, `/indicators`, `/map_obj.json` | 10 Hz |
| Slow | `/map_info.json`, `/mission.json` | 1 Hz |
| Incremental | `/hudmsg`, `/gamechat` | 1-2 Hz when enabled |
| On change | `/map.img` | When `map_generation` changes |

Each endpoint has:

- An independent timeout
- A last-success timestamp
- A last-attempt timestamp
- Its own error and freshness state
- A last-known-good payload
- At most one request in flight

One endpoint failure must not invalidate healthy data from another endpoint.

### Normalization

War Thunder payloads are converted into a stable, versioned application model.
The UI should not need to parse keys such as `"IAS, km/h"` or decide which
cockpit-specific alias represents a value.

Example:

```json
{
  "sequence": 1842,
  "capturedAt": "2026-09-01T18:53:32Z",
  "connection": {
    "state": "live",
    "sources": {
      "state": { "fresh": true, "ageMs": 18 },
      "indicators": { "fresh": true, "ageMs": 15 },
      "mapObjects": { "fresh": true, "ageMs": 12 }
    }
  },
  "vehicle": {
    "type": "jh_7",
    "class": "air"
  },
  "flight": {
    "iasKmh": 681,
    "tasKmh": 861,
    "altitudeM": 5471,
    "verticalSpeedMps": -12.9,
    "mach": 0.78,
    "headingDeg": 359.89,
    "angleOfAttackDeg": 3.1,
    "gLoad": 1.01
  }
}
```

Normalization rules:

- Missing data remains absent or `null`; it never silently becomes zero.
- Prefer unit-bearing `/state` fields for the primary flight strip.
- Use `/indicators` only when a channel has an observed interpretation.
- Keep raw snapshots internally for diagnostics.
- Preserve unknown upstream fields during capture and debugging.
- Model source freshness explicitly.
- Discover engines and capabilities from present fields.
- Never derive or synthesize contacts the game did not expose.
- Do not introduce aircraft warning thresholds without a trusted,
  version-matched aircraft database.

### Snapshot store

Pollers publish into one immutable in-memory snapshot. The server broadcasts the
latest coherent snapshot to all clients.

This separates:

- War Thunder polling rate
- Snapshot publication rate
- Browser transport rate
- Display rendering rate

The map image is cached separately and keyed by `map_generation`.

### Local application API

Initial API:

```text
GET /api/v1/status
GET /api/v1/snapshot
GET /api/v1/map/{generation}
WS  /api/v1/live
```

The first implementation should send a complete normalized snapshot at up to
10 Hz. The payload is small enough that a delta protocol would add complexity
without meaningful benefit.

Telemetry responses should use `Cache-Control: no-store`. Map images may be
cached by generation.

## Frontend design

```text
Application shell
|-- Connection-state screen
|-- Tactical map
|   |-- Background image
|   |-- Grid
|   |-- Airfields and objectives
|   |-- Tactical objects
|   |-- Player position and heading
|   |-- Selected point
|   `-- Measurement and route layer
|-- Primary flight-data strip
|-- Navigation panel
|-- Mission summary
`-- Settings and tablet connection
```

### Map renderer

- Draw the map raster and tactical layers with Canvas 2D.
- Maintain a pure viewport transform for normalized, world, image, and screen
  coordinates.
- Test every forward and inverse coordinate conversion.
- Support mouse wheel, drag, touch pan, and pinch zoom.
- Render from the latest snapshot using `requestAnimationFrame`.
- Keep React out of the per-symbol drawing loop.
- Perform pointer hit testing in map coordinates.
- Render unknown object types with a safe fallback symbol.
- Never hard-code map image dimensions.

### UI state

React owns:

- Connection and stale-data presentation
- Flight strip
- Navigation readout
- Mission information
- Layer controls
- Settings

The selected map point and display preferences may initially live in browser
storage. Shared routes and waypoint libraries can move into companion-managed
storage when those features are introduced.

## Local files

Initial storage:

```text
%APPDATA%\wt-modern-8111\
`-- identity.json
```

Current persisted data:

- Confirmed or explicitly configured pilot callsign

Telemetry, chat messages, and other player names remain in memory for the live
dashboard and are not written to disk. The `-callsign` and `-forget-callsign`
flags replace or clear the persisted identity.

## Security and privacy

- Bind the companion to `127.0.0.1` by default.
- Enable only a selected private LAN interface when requested.
- Require pairing for LAN browser clients.
- Validate origins and authentication for state-changing requests.
- Poll only fixed, allowlisted War Thunder paths.
- Process team game chat locally for the event feed and radio-command features.
- Keep game chat in memory only.
- Never expose the service to the public internet.
- Never read game memory.
- Never inject into or hook the game process.
- Never capture game frames to derive unavailable information.
- Keep telemetry local; no account or cloud connection is required.

War Thunder itself was observed listening on `0.0.0.0:8111` when remote access
was enabled. WT Modern must not assume that the upstream service is private
merely because it is described as `localhost`.

## Proposed repository structure

```text
wt-modern-8111/
|-- cmd/
|   `-- wt-modern/
|       `-- main.go
|-- internal/
|   |-- app/
|   |-- configuration/
|   |-- navigation/
|   |-- platform/
|   |-- polling/
|   |-- server/
|   |-- telemetry/
|   `-- warthunder/
|-- web/
|   |-- src/
|   |   |-- connection/
|   |   |-- flight/
|   |   |-- map/
|   |   |-- mission/
|   |   `-- navigation/
|   `-- package.json
|-- docs/
|-- fixtures/
|-- go.mod
|-- PLAN.md
`-- README.md
```

The existing sanitized captures under
[`docs/fixtures/air-test-flight-jh-7/`](docs/fixtures/air-test-flight-jh-7/)
will seed the initial parser tests and mock server. Fixtures can move to the
root `fixtures/` directory when executable code is bootstrapped.

## Testing strategy

Development and continuous integration must not require War Thunder.

1. **Fixture parsing:** validate captured payload shapes and optional fields.
2. **Normalization:** verify canonical units, missing fields, duplicate aliases,
   sentinels, and multi-engine discovery.
3. **Navigation:** test normalized/world/screen transforms, heading, bearing,
   distance, and angle wraparound.
4. **Lifecycle:** simulate offline, hangar, loading, live, stale, degraded, and
   recovered states.
5. **Server contracts:** validate HTTP and Server-Sent Events payloads.
6. **Frontend components:** test flight, connection, mission, and navigation
   displays against normalized snapshots.
7. **Map interaction:** test pan, zoom, touch, point selection, and resize.
8. **Browser integration:** run the complete UI against a mock `8111` server.
9. **Manual game validation:** run a documented smoke test before releases.

## Delivery

Release artifacts:

- Signed Windows installer for normal users
- Portable executable for advanced users
- No external runtime dependencies

The initial release does not require an automatic updater. A signed updater can
be added after the application and release process are stable.

Code signing is important to the usability goal because unsigned executables
create Windows SmartScreen friction.

## Implementation phases

### Phase 0 - API baseline

- [x] Research the known `8111` endpoints and response formats.
- [x] Define polling, caching, and connection-state expectations.
- [x] Capture an active JH-7 Air Test Flight.
- [x] Verify player heading against the map direction vector.
- [ ] Capture hangar, loading, death, spectator, and return-to-hangar states.
- [ ] Capture at least one propeller and one additional multi-engine aircraft.
- [ ] Compare exposed map contacts in Air AB, RB, and SB.

### Phase 1 - Companion foundation

- [x] Bootstrap the Go module and frontend workspace.
- [x] Implement explicit fixture-backed development mode.
- [x] Implement typed raw endpoint clients.
- [x] Implement independent fast and slow polling schedules.
- [x] Implement source health, freshness, and connection state.
- [x] Implement the immutable snapshot store.
- [x] Serve the embedded frontend and versioned local API.

### Phase 2 - Functional map proof of concept

- [x] Render the live map image.
- [x] Render map objects and airfields.
- [x] Render player position and heading.
- [ ] Evaluate optional setup-only pan and zoom without weakening automatic framing.
- [x] Automatically select the nearest exposed strike point.
- [x] Show bearing, distance, and ETA to the active point.
- [x] Handle `map_generation` changes without flashing stale imagery.

### Phase 3 - Flight and mission presentation

- [x] Implement the primary flight-data strip.
- [x] Add mission and objective presentation.
- [ ] Add unit preferences.
- [x] Add explicit stale and unavailable states per metric.
- [x] Tune responsive desktop and landscape-tablet layouts.

### Phase 4 - End-user packaging

- [ ] Add single-instance behavior and automatic browser launch.
- [ ] Add a production tray lifecycle.
- [ ] Add explicit LAN mode and QR pairing.
- [ ] Add Windows Firewall guidance.
- [ ] Build a portable executable.
- [ ] Build and sign a Windows installer.
- [ ] Test installation and first launch on a clean Windows machine.

## Deferred work

Do not add these until the first milestone proves the map-first UX:

- Aircraft-specific dashboards
- External flight-model database
- VNE, stall, or structural-limit warnings
- Flight recording and replay
- Shared route and waypoint libraries
- User-authored formulas
- Cloud synchronization
- Public internet access
- Native in-game overlay

## Current architectural commitments

1. One local companion executable
2. Go backend
3. React and TypeScript frontend
4. Embedded static frontend
5. Canvas-based tactical map
6. Server-Sent Events snapshot delivery
7. No database initially
8. Loopback by default and explicit paired LAN mode
9. Mock War Thunder service from the beginning
10. Browser-based display instead of Electron or a cloud frontend
