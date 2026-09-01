# WT Modern 8111

Modern, Sim-focused frontend for War Thunder's `localhost:8111` telemetry and
tactical map interface.

The project is intended to feel like an electronic flight bag or mission
computer rather than a generic telemetry dashboard. The first milestone is a
map-first, glanceable second-screen experience for Simulator Battles.

## Current status

The first end-to-end application slice is implemented:

- Go companion polling War Thunder's local service
- Normalized, versioned snapshot API
- Server-Sent Events for live browser updates
- Generation-aware tactical-map image cache
- Click-to-select map objects, map points, and supported mission objectives
- Bearing, range, and ETA for the selected destination
- Radio-command RTB navigation to the nearest friendly airfield
- Rolling zero-reserve bingo-fuel estimate for direct airfield recovery
- Damage-aware engine and aircraft-loss status
- Chronological chat, HUD event, and damage feed
- Embedded React/TypeScript briefing-columns frontend
- Explicit fixture mode using the captured JH-7 session

- [Project architecture and implementation plan](PLAN.md)
- [localhost:8111 API research](docs/localhost-8111-api.md)
- [Simulator live-capture protocol](docs/sim-live-capture.md)
- [Passive UX concepts](concepts/README.md)

## Architecture

The application keeps transport, domain behavior, and presentation separate:

- `internal/warthunder`: bounded client and upstream endpoint types
- `internal/polling`: polling orchestration with separate health, feed, session,
  radio-mark, and RTB modules
- `internal/wtradio`: reusable radio-command and grid parsing
- `internal/telemetry`: normalized versioned snapshot model and derivation
- `internal/server`: local JSON, SSE, map-image, and embedded-SPA delivery
- `web/src/features`: feature-owned dashboard components and state hooks
- `web/src/navigation`: target inference, tracking, and navigation calculation
- `web/src/map`: coordinate geometry, hit testing, image loading, and rendering
  layers
- `web/src/shared`: presentation primitives and formatting

`App.tsx` and `TacticalMap.tsx` are composition adapters; domain and rendering
behavior belongs in reusable modules with focused tests.

## Development

Requirements:

- Go 1.27 or later
- Node.js 24 or later
- War Thunder for live data, or the included fixture for offline development

Install and build the embedded frontend:

```powershell
npm --prefix .\web ci
npm --prefix .\web run build
```

Run against War Thunder:

```powershell
go run .\cmd\wt-modern
```

Run against the captured JH-7 fixture:

```powershell
go run .\cmd\wt-modern -fixture .\docs\fixtures\air-test-flight-jh-7
```

The dashboard opens at `http://127.0.0.1:17711`. Pass `-open=false` to suppress
automatic browser launch.

Validation:

```powershell
go test .\...
npm --prefix .\web run typecheck
npm --prefix .\web run lint
npm --prefix .\web test
npm --prefix .\web run build
```

## First milestone

1. Connect to the local War Thunder service.
2. Display the live tactical map.
3. Show player position and heading.
4. Render exposed map objects and objectives.
5. Present a compact primary flight-data strip.
6. Automatically frame the current tactical picture.
7. Select an exposed target, mission objective, or arbitrary map point.
8. Calculate bearing, distance, and ETA to that destination.
9. Work well on landscape desktop and tablet displays.
10. Handle game-offline, hangar, loading, and partial-data states.

## Product principles

- Sim-first
- Map-first
- Glanceable
- Low visual noise
- Contextual
- Second-screen friendly
- Responsive
- Modern but restrained

## Safety boundary

WT Modern 8111 should consume only data deliberately exposed by War Thunder's
local HTTP service. It must not read game memory, inject into the game process,
hook rendering, or derive information from captured game frames.
