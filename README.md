# WT Modern 8111

A browser dashboard for War Thunder Simulator Battles.

WT Modern 8111 reads telemetry and tactical-map data from `localhost:8111` and
serves a map-first interface on the same PC.

![WT Modern 8111 dashboard](docs/images/dashboard.png)

## Features

- Live tactical map with player position, aircraft, ground units, airfields,
  objectives, and air defenses
- Clickable map objects and arbitrary navigation points
- Automatic detection of War Thunder map targets, including nearby-entity
  inference and tracking for moving targets
- Bearing, distance, and ETA to the active destination
- IAS, TAS, Mach, altitude, heading, vertical speed, AoA, and G-load
- Engine state, configuration, fuel quantity, and learned fuel consumption
- Direct-return bingo fuel estimate to the nearest friendly airfield
- Simulator radio support for `Guide on me`, `Cover me`,
  `Attention to the map`, and return-to-base calls
- Mission, chat, HUD-event, and damage feed
- Captured fixture mode for offline development

![Tactical map and active destination](docs/images/tactical-map.png)

## Run from source

Requirements:

- Go 1.27 or later
- Node.js 24 or later

Build the embedded frontend:

```powershell
npm --prefix .\web ci
npm --prefix .\web run build
```

Run the companion while War Thunder is open:

```powershell
go run .\cmd\wt-modern
```

The dashboard is available at `http://127.0.0.1:17711`. On Windows, the tray
icon opens it; use `-open=true` to open it immediately at launch.

On Windows, the companion runs from the notification area. Left-click the tray
icon or choose **Open Dashboard**. The tray menu also provides
**Start with Windows** and **Quit**. Build the executable before enabling
automatic startup; `go run` uses a temporary executable that Windows removes.

Set or replace the remembered pilot callsign:

```powershell
go run .\cmd\wt-modern -callsign "=SQUAD= PilotName"
```

Clear the remembered callsign:

```powershell
go run .\cmd\wt-modern -forget-callsign
```

Run with the included offline fixture:

```powershell
go run .\cmd\wt-modern -fixture .\docs\fixtures\air-test-flight-jh-7
```

## Development

```powershell
npm --prefix .\web run typecheck
npm --prefix .\web run lint
npm --prefix .\web test
npm --prefix .\web run build
go test .\...
```

Build the Windows application without a console window:

```powershell
.\scripts\build-windows.ps1
```

The executable is written to `dist\wt-modern.exe`.

Build the portable executable, installer, and checksums:

```powershell
.\scripts\build-installer.ps1 -Version 1.0.0
```

## Releases and updates

Version tags such as `v1.0.0` publish:

- `wt-modern-setup.exe` — per-user installer
- `wt-modern-windows-amd64.exe` — portable application
- `checksums.txt` — SHA-256 release checksums

Release builds check GitHub automatically at startup and every six hours. A
new executable is downloaded, verified against `checksums.txt`, installed after
the running process exits, and restarted without opening a console window.

The Go companion handles polling, normalization, local identity, and SSE
delivery. The React frontend is split into feature, navigation, map-rendering,
and shared UI modules. CI runs frontend validation, Go vet, race-enabled tests,
and a full companion build.

## Current scope

- Data source: War Thunder's local HTTP API
- Air-defense identification: broad `SAM` and `AAA` classes; the API omits the
  vehicle and weapon details required for range rings
- Game mode priority: Simulator Battles
- Distribution: source build

## Documentation

- [Architecture and implementation plan](PLAN.md)
- [localhost:8111 API research](docs/localhost-8111-api.md)
- [Live-capture protocol](docs/sim-live-capture.md)
