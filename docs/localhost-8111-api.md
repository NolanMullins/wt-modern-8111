# War Thunder `localhost:8111` API research

Last researched: 2026-09-01

## Purpose and confidence labels

War Thunder exposes an undocumented local HTTP service at
`http://127.0.0.1:8111`. This document records the public interface needed by
WT Modern 8111 before implementation begins.

Gaijin does not publish a complete, versioned contract for this service. The
reference below is therefore based on direct captures and implementations in
public projects. It uses these labels:

- **Observed**: present in a captured response or the game's built-in page and
  corroborated by public client code.
- **Reported**: described by community projects but not confirmed by a capture
  reviewed for this document.
- **Unknown**: requires a live capture against the current game.

Response schemas must be treated as open and vehicle-dependent. Consumers
should preserve unknown fields and must not assume that a field seen for one
aircraft exists for another.

## Service availability

The service is available only while War Thunder is running. Depending on the
installation and game mode, the corresponding **Allow remote access** setting
may also need to be enabled in War Thunder's battle settings. The default port
is `8111`.

In the 2026-09-01 live capture, the service listened on `0.0.0.0:8111` and a
request to the gaming PC's Ethernet address succeeded. Direct access from
another LAN device is therefore possible when War Thunder is configured this
way; it is not safe to assume that the service is loopback-only. Binding may
depend on the game's remote-access setting and still needs verification with
that setting disabled.

The service has no dedicated health endpoint. Availability and game state must
be inferred from request success and payloads such as `valid` and
`mission.status`.

Observed response headers include permissive CORS headers:

```http
Access-Control-Allow-Origin: *
Access-Control-Allow-Methods: GET, POST, OPTIONS
Access-Control-Allow-Headers: origin, content-type, accept
```

No useful POST operation has been identified. WT Modern 8111 should consider
the upstream service read-only and use GET requests only. An `OPTIONS /state`
request returned `200`, an empty body, and the CORS headers above in the live
capture; POST was deliberately not tested.

## Endpoint inventory

All known upstream endpoints are GET endpoints.

| Path | Media type | Purpose | Confidence |
| --- | --- | --- | --- |
| `/` | `text/html` | Gaijin's built-in tactical map page | Observed |
| `/state` | `application/json` | Flight-model and physics telemetry | Observed |
| `/indicators` | `application/json` | Cockpit or vehicle instrument values | Observed |
| `/map_info.json` | `application/json` | Map bounds, grid, and generation metadata | Observed |
| `/map_obj.json` | `application/json` | Objects currently exposed on the tactical map | Observed |
| `/map.img` | `image/png` or `image/jpeg` | Tactical map background raster | Observed |
| `/map.img?gen=<id>` | image | Cache-busted map raster for a generation | Observed |
| `/mission.json` | `application/json` | Mission status and objectives | Observed |
| `/hudmsg?lastEvt=<id>&lastDmg=<id>` | `application/json` | Incremental HUD event and damage records | Observed |
| `/gamechat?lastId=<id>` | `application/json` | Incremental game chat records | Observed |
| `/loc/map/primary_objectives?fmt=js` | JavaScript | Localized primary-objective strings | Observed |
| `/loc/map/secondary_objectives?fmt=js` | JavaScript | Localized secondary-objective strings | Observed |

The following intuitive variants have been observed to return 404 and should
not be used: `/map_info`, `/map_obj`, `/state.json`, `/indicators.json`, and
`/map`.

## Live capture: JH-7 Air Test Flight

A live capture was taken on 2026-09-01 while a JH-7 was flying in Air Test
Flight. The game version is not exposed by the API and was not available for
this capture. Sanitized snapshot fixtures are stored in
[`fixtures/air-test-flight-jh-7/`](fixtures/air-test-flight-jh-7/).

Observed at one synchronized snapshot:

| Endpoint | Observation |
| --- | --- |
| `/state` | `valid: true`; two engines; 613-byte JSON response |
| `/indicators` | `valid: true`, `army: "air"`, `type: "jh_7"`; 1,350-byte JSON response |
| `/map_info.json` | `valid: true`, `hud_type: 0`, `map_generation: 1`, world bounds `-32768..32768` on both axes |
| `/map_obj.json` | 35 objects: 5 aircraft, 1 airfield, 25 ground models, and 4 bombing points |
| `/mission.json` | `{"objectives": null, "status": "running"}` |
| `/map.img` | 128 x 128, 32-bit ARGB PNG, 1,519 bytes |
| `/hudmsg` | Empty `events` and `damage` arrays when queried beyond the current cursor |
| `/gamechat` | Empty array when queried beyond the current cursor |

The objective-localization endpoints returned executable JavaScript assignments
with the English values `"PRIMARY GOALS"` and `"SECONDARY GOALS"`, not JSON.

## `/state`

`/state` is the primary source for normalized flight-model values. Its unusual
schema embeds a unit in each key:

```json
{
  "valid": true,
  "H, m": 4936,
  "TAS, km/h": 237,
  "IAS, km/h": 185,
  "M": 0.2,
  "AoA, deg": 1.1,
  "AoS, deg": -0.1,
  "Ny": 1.0,
  "Vy, m/s": 3.2,
  "Mfuel, kg": 750,
  "Mfuel0, kg": 2620,
  "throttle 1, %": 100,
  "RPM 1": 1957,
  "manifold pressure 1, atm": 0.81,
  "oil temp 1, C": 73
}
```

Common field families include:

| Domain | Observed keys |
| --- | --- |
| Flight | `H, m`, `TAS, km/h`, `IAS, km/h`, `M`, `AoA, deg`, `AoS, deg`, `Ny`, `Vy, m/s` |
| Rotation | `Wx, deg/s`, `Wy, deg/s`, `Wz, deg/s` |
| Controls | `aileron, %`, `elevator, %`, `rudder, %`, `flaps, %`, `gear, %`, `airbrake, %` |
| Fuel | `Mfuel, kg`, `Mfuel0, kg` |
| Engines | `throttle N, %`, `power N, hp`, `RPM N`, `manifold pressure N, atm`, temperatures and thrust |

`N` is a one-based engine number. Fields for nonexistent engines are generally
omitted. Parsers should discover engines from present keys rather than assume a
fixed count.

The JH-7 capture contained both engine 1 and engine 2 fields, including
throttle, RPM, manifold pressure, oil temperature, thrust, and efficiency. It
reported `power N, hp: 0.0` while each engine produced nonzero thrust, showing
that a present numeric field can still be inapplicable to the current propulsion
type.

Observed ground-vehicle captures may return only `{"valid": false}`. Whether
any ground modes produce a valid `/state` payload remains unknown.

### Parsing requirements

- Match the entire key, including its unit.
- Treat every metric other than `valid` as optional.
- Store raw data alongside normalized data so new fields are not lost.
- Do not convert a missing metric to zero.
- Parse values only when their JSON type is numeric; do not accept arbitrary
  strings as telemetry.

## `/indicators`

`/indicators` reports instrument values and the internal vehicle identifier.
Unlike `/state`, its units are usually not included in the key.

```json
{
  "valid": true,
  "type": "so_4050_vautour_2a_iaf",
  "speed": 0.0,
  "altitude_hour": 63.06,
  "aviahorizon_roll": -0.08,
  "aviahorizon_pitch": -4.28,
  "compass": 355.83,
  "fuel": 3182.0,
  "gears": 0.5,
  "flaps": 0.0,
  "throttle": 0.0,
  "mach": 0.0,
  "g_meter": 0.92
}
```

Observed field families include:

- Movement and controls: `speed`, `pedals`, `stick_elevator`,
  `stick_ailerons`, `vario`
- Attitude and navigation: `aviahorizon_roll`, `aviahorizon_pitch`, `bank`,
  `turn`, `compass`
- Engine: `rpm`, numbered RPM fields, `manifold_pressure`, oil, water, and
  carburetor temperatures
- Fuel: `fuel`, numbered fuel fields, `fuel_pressure`, `fuel_consume`
- Systems: `gears`, `flaps`, `trimmer`, `radiator`, `supercharger`,
  `prop_pitch`, `wing_sweep_lever`
- Weapons: `weapon1` through `weapon3` and numbered ammunition counters

Field presence and meaning vary by cockpit and vehicle. For example, a
variable-sweep control is exposed only where applicable. Some equipment fields
use `-1` as an unavailable/not-equipped sentinel. A negative value must not be
treated generically as valid sensor data.

The JH-7 capture demonstrates that indicator names are instrument channels, not
necessarily canonical aircraft state:

- `speed` was `189.13` while `/state` IAS was `681 km/h`; multiplying the
  indicator value by 3.6 gives approximately the same IAS. For this aircraft,
  `speed` therefore behaved as meters per second.
- `altitude_10k` matched `/state` altitude while `altitude_min` contained only
  the lower-scale portion of the cockpit instrument.
- `/state` reported gear and flaps at `0%`, while `indicators.gears` and
  `indicators.flaps` were both `0.5`. These indicator values must not be assumed
  to be normalized extension percentages.
- `water_temperature` was about `665` in a jet. The label alone is not enough
  to assign a physical unit or warning threshold.
- Multiple aliases were present for the same control, including `throttle`,
  `throttle1`, `throttle_1`, and `throttle1_1`.

Prefer unit-bearing `/state` values for the primary flight strip. Use
`/indicators` as a vehicle-specific instrument source only after a field has an
observed interpretation.

`type` is the best live key available for aircraft-specific configuration. The
API does not expose a stable display name or performance envelope such as VNE,
critical AoA, or structural G limits. Those require a separately maintained,
versioned aircraft database if added later.

## `/map_info.json`

Two observed schema generations demonstrate that types and optional fields can
change:

```json
{
  "grid_steps": ["3250.0", "3250.0"],
  "grid_zero": ["-32768.0", "32768.0"],
  "map_generation": "1",
  "map_max": ["32768.0", "32768.0"],
  "map_min": ["-32768.0", "-32768.0"]
}
```

```json
{
  "grid_size": [1600.0, 1600.0],
  "grid_steps": [225.0, 225.0],
  "grid_zero": [1519.45, 2497.25],
  "hud_type": 1,
  "map_generation": 1,
  "map_max": [4096.0, 4096.0],
  "map_min": [0.0, 0.0],
  "valid": true
}
```

The normalizer must safely accept numeric strings or numbers for map metadata.
`grid_size`, `hud_type`, and `valid` are optional. Invalid or incomplete bounds
must produce an explicit map-unavailable state rather than fabricated defaults.

`map_generation` identifies the current map raster. Fetch `/map.img` once when
this value first becomes valid and again only when it changes.

## `/map_obj.json`

The response is an array. A common point-object shape is:

```json
{
  "type": "ground_model",
  "color": "#fac81e",
  "color[]": [250, 200, 30],
  "blink": 0,
  "icon": "Player",
  "icon_bg": "none",
  "x": 0.421074,
  "y": 0.56343,
  "dx": 0.939689,
  "dy": -0.342029
}
```

Observed object `type` values include `ground_model`, `aircraft`, `airfield`,
`defending_point`, `bombing_point`, `respawn_base_fighter`, and
`respawn_base_bomber`.

Observed icons include `Player`, `Fighter`, `Bomber`, `Assault`, `Tracked`,
`Wheeled`, `Airdefence`, `MediumTank`, `Structure`, `waypoint`,
`capture_zone`, objective icons, and respawn-base icons. The JH-7 Test Flight
also exposed `LightTank`, `SPAA`, and `SAM`. Clients must render an
unknown-object fallback because this list is not a closed enum.

Most point objects use normalized `x` and `y` coordinates in the range 0 to 1.
Rotatable objects may include a direction vector in `dx` and `dy`. Airfield
objects use runway endpoints `sx`, `sy`, `ex`, and `ey` instead of a single
point.

`blink` has been observed as:

- `0`: no blink
- `1`: normal blink
- `2`: strong blink

It is reported, but not yet verified per game mode, that objects hidden from
the in-game tactical map are omitted rather than included with a hidden flag.
The application must display only what the endpoint returns and must not infer
or synthesize unseen contacts.

## Map coordinates, distance, and bearing

For a map object with normalized coordinates `(x, y)`:

```text
pixel_x = image_width * x
pixel_y = image_height * y

world_x = map_min_x + x * (map_max_x - map_min_x)
world_y = map_min_y + y * (map_max_y - map_min_y)
```

For source `(sx, sy)` and destination `(tx, ty)`, both normalized:

```text
dx_world = (tx - sx) * (map_max_x - map_min_x)
dy_world = (ty - sy) * (map_max_y - map_min_y)
distance = sqrt(dx_world^2 + dy_world^2)
```

The implementation used by existing map clients derives north-up bearing as:

```text
bearing = normalize_degrees(atan2(dx_world, -dy_world))
```

and object heading as:

```text
heading = normalize_degrees(atan2(object.dx, -object.dy))
```

The object-heading formula is now directly verified. Across 100 live JH-7
samples, its result differed from `/indicators.compass` by an average of
`0.000408` degrees and a maximum of `0.001041` degrees. This confirms the map
direction vector and Y-axis convention for player heading.

Destination bearing uses the same coordinate convention, but cardinal-point
selection has not yet been manually compared with an in-game indication.
`grid_zero` to official map-grid labels is also unresolved.

There is no known latitude/longitude mapping. Bounds represent game-world map
units, not Earth coordinates.

## `/map.img`

The URL suffix does not guarantee the image encoding. Both `image/png` and
`image/jpeg` have been observed, so the proxy must forward the upstream
`Content-Type`.

The image is static for a `map_generation`. It should be retained across
temporary telemetry failures and replaced atomically after a new generation
has been downloaded successfully.

The JH-7 capture returned a 128 x 128 PNG with no observed cache headers.
`/map.img` and `/map.img?gen=1` were byte-identical. Image dimensions must not
be hard-coded.

### Built-in map renderer and assets

A direct audit of the built-in page on 2026-09-01 established how Gaijin
composes the tactical picture:

- The page creates a square 640 x 640 canvas and stretches `/map.img` across
  it.
- The captured Air Test Flight raster is a grayscale question-mark
  placeholder, not terrain. It is 91.94% `#404040` and 6.69% `#808080`.
- Grid lines and alphanumeric labels are generated from `map_min`, `map_max`,
  and `grid_steps`. The current built-in JavaScript does not use `grid_zero`.
- Airfields are colored lines drawn from normalized `sx`, `sy`, `ex`, and `ey`
  endpoints.
- The player is a white direction arrow with a dark outline.
- Other object colors come directly from their `/map_obj.json` entries.
- Other symbols use a custom `icons.ttf` font. The live service returned this
  588,472-byte resource as `text/plain` without a CORS header.
- The built-in page requests mission, objects, map metadata, chat, HUD feeds,
  indicators, and state every 500 ms, while redrawing the cached map every
  25 ms.

WT Modern should not package Gaijin's font or captured map imagery. The
companion can proxy user-local runtime assets same-origin, or the project can
ship an independently designed tactical symbol set. Map layouts must preserve
the square coordinate geometry; stretching the raster or normalized object
space into a rectangular panel distorts distance and bearing.

## `/mission.json`

Observed payloads include both populated and null objectives:

```json
{
  "status": "running",
  "objectives": [
    {
      "primary": true,
      "status": "in_progress",
      "text": "Take off"
    }
  ]
}
```

```json
{
  "status": "running",
  "objectives": null
}
```

The client must treat `objectives` as nullable. `"running"` is observed.
Sources disagree between `"fail"` and `"failed"` for a failed mission, so all
unrecognized status strings must remain representable until live captures
establish the current set.

## Incremental feeds

### `/hudmsg`

Request:

```text
/hudmsg?lastEvt=<highest-seen-event-id>&lastDmg=<highest-seen-damage-id>
```

Response:

```json
{
  "events": [],
  "damage": [
    {
      "id": 161,
      "msg": "example damage message",
      "sender": "",
      "enemy": false,
      "mode": ""
    }
  ]
}
```

### `/gamechat`

Request:

```text
/gamechat?lastId=<highest-seen-chat-id>
```

Entries may contain `id`, `msg`, `sender`, `enemy`, `mode`, and possibly
`time`. Cursor IDs reportedly reset when the game starts, not at every match.
The poller must detect a cursor rollback or game restart rather than assuming
IDs are scoped to one mission.

Messages and player names are untrusted display text. They must be rendered as
text, never inserted as HTML.

## Polling and caching

Recommended starting rates balance responsiveness with a conservative load on
the game's local server:

| Data | Endpoint | Starting rate |
| --- | --- | --- |
| Flight state | `/state` | 5-10 Hz |
| Instruments | `/indicators` | 5-10 Hz |
| Map objects | `/map_obj.json` | 5-10 Hz |
| Map metadata | `/map_info.json` | 1 Hz |
| Mission | `/mission.json` | 1 Hz |
| HUD feed | `/hudmsg` | 1-2 Hz |
| Chat feed | `/gamechat` | 1-2 Hz |
| Map raster | `/map.img?gen=<id>` | On generation change only |

Network polling and UI rendering must be decoupled. The frontend should render
the newest immutable snapshot at display speed while a local service polls the
game independently.

Each endpoint must have its own status, last-success timestamp, and error.
Failure of one endpoint must not discard valid data from others. Requests
should not overlap with a previous poll of the same endpoint; a slow response
should skip or delay that endpoint's next tick.

During a 10.775-second live trace, `/state`, `/indicators`, and
`/map_obj.json` were each requested 100 times sequentially:

| Endpoint | Errors | Mean response | 95th percentile | Payload changed |
| --- | --- | --- | --- | --- |
| `/state` | 0 | 3.21 ms | 4.28 ms | 99 of 99 comparisons |
| `/indicators` | 0 | 2.66 ms | 3.23 ms | 99 of 99 comparisons |
| `/map_obj.json` | 0 | 2.82 ms | 3.21 ms | 99 of 99 comparisons |

This confirms that roughly 9 Hz polling was viable on the tested machine and
that all three endpoints updated at every observed interval during active
flight. It does not establish a universal maximum rate.

## Connection and game-state model

There is no single upstream status value. The application should distinguish:

1. **Service offline**: connection refused or repeated request timeout.
2. **Service online, no active vehicle**: requests succeed but both primary
   telemetry payloads are invalid.
3. **Mission loading**: endpoints are reachable but map or mission data is
   temporarily incomplete.
4. **In mission**: at least one telemetry source is valid and/or mission status
   is running.
5. **Partially degraded**: the mission remains usable while one or more
   endpoints are stale or failing.

Keep last-known-good data during short failures, mark it stale, and show its
age. Never present stale navigation data as live. A new `map_generation`,
vehicle `type`, cursor rollback, or transition through an invalid state should
be considered a session-boundary signal.

## Browser and second-device architecture

A browser can often fetch the service directly because of its CORS headers. If
War Thunder is listening on `0.0.0.0`, a tablet can use the gaming PC's LAN
address instead of `localhost`. Direct access is still not sufficient for the
intended product architecture:

- An HTTPS-hosted page cannot reliably fetch a plain HTTP telemetry endpoint
  because of browser mixed-content protections.
- On a tablet, `localhost` means the tablet, so clients would have to discover
  and retain the gaming PC's address.
- Enabling direct LAN access exposes the raw game service to every device that
  can reach the port.
- The game service is not a stable, versioned API and must never be exposed to
  the internet.
- Normalization, rate control, and stale-data handling should be implemented
  once rather than in every browser client.

The recommended architecture is a local companion service on the gaming PC:

```text
War Thunder :8111
       |
       | local HTTP polling
       v
WT Modern companion service
  - validates and normalizes payloads
  - caches immutable snapshots
  - derives navigation values
  - proxies the map image
  - serves the frontend
       |
       | same-origin HTTP/WebSocket on trusted LAN
       v
desktop browser / tablet
```

The companion should bind to loopback by default. LAN binding must be explicit,
display the exact reachable URL, and be paired with host-firewall guidance.
The product must never suggest router port forwarding or internet exposure.

Serving the frontend and application API from the same local origin avoids
browser CORS and mixed-content problems. WebSocket or Server-Sent Events can
push cached snapshots to clients; the companion still polls War Thunder over
HTTP at the rates above.

## Simulator Battles findings and limitations

The endpoint names appear consistent across game modes, but no reviewed source
provides a current, controlled Air AB/RB/SB availability matrix.

What can be stated now:

- Air Test Flight exposes valid `/state`, `/indicators`, map, map-object, and
  mission data during active JH-7 flight.
- Telemetry shape varies by vehicle and cockpit, not just by game mode.
- `/state` and `/indicators` overlap but are not interchangeable.
- Instrument units can differ by aircraft. Existing clients include
  aircraft-specific handling for feet versus meters.
- Multi-engine fields are sparse and aircraft-dependent.
- Map contacts should be assumed to contain no more information than the game
  chooses to expose locally.
- Performance limits are not provided by the live service.
- Offline, hangar, loading, spawn, death, and spectating transitions are not
  represented by a documented state machine.

These limitations support a capability-driven UI: show a metric only when its
source is present, valid, and fresh. Aircraft-specific layouts should wait
until the initial map-first milestone and live compatibility matrix are proven.

## Architecture lessons from existing projects

| Project | Relevant lesson |
| --- | --- |
| [WT-8111-Neo](https://github.com/CreeperUX/WT-8111-Neo) | Local companion service proxies `8111`, serves the web app, and makes LAN access explicit. Its API reference carefully separates observed and expected data. |
| [WarThunder8111](https://github.com/mukasc/WarThunder8111) | A shared telemetry layer can support both an overlay and a browser dashboard, but renderer security workarounds are a warning against direct browser polling. |
| [War Thunder Flight Desk](https://github.com/MistressOfDNS/warthunder-ui) | Separates fast and slow endpoint groups, normalizes awkward state keys once, and preserves per-endpoint errors during partial failure. |
| [wt-iox-api](https://github.com/IsraelSiq/wt-iox-api) | Uses an async poller, shared in-memory state, REST/WebSocket delivery, and a mock War Thunder server for development without the game. |
| [wthud](https://github.com/wysiwyng/wthud) and [wthud2](https://github.com/wysiwyng/wthud2) | Demonstrate explicit game-state inference, aircraft-specific configuration, and the need for unit normalization. |
| [War Thunder BYOH](https://github.com/SpaceCapo/warthunder-byoh) | Decouples approximately 6 Hz data capture from 60 Hz rendering and uses external flight-model data for limits absent from the API. |

The reusable pattern is a thin, testable polling and normalization core plus a
map-first web client. The visual design should not be copied.

## Unknowns requiring live capture

1. Endpoint and field availability for more Air Test Flight aircraft and in
   Air AB, RB, and SB.
2. Behavior while spawning, dying, spectating, switching aircraft, and leaving
   a mission.
3. Whether exposed enemy objects exactly match what the in-game map shows in
   every mode.
4. The current complete set of object types, icons, mission statuses, and
   sentinel values.
5. Exact instrument units by aircraft, especially imperial cockpits.
6. Destination-bearing cardinal points against an in-game indication; the
   player-heading transform is verified.
7. Conversion from `grid_zero` and `grid_steps` to official map-grid labels.
8. Whether `hud_type` changes icons or coordinate transforms.
9. Whether HUD `events` is populated in current game versions.
10. Response headers, encodings, timeouts, and map image media type across
    session transitions.
11. Whether POST has any meaning despite being listed by the CORS header.

The procedure for answering these questions is in
[the live-capture protocol](sim-live-capture.md).

## Sources

Primary technical sources reviewed:

- [WT-8111-Neo API reference](https://github.com/CreeperUX/WT-8111-Neo/blob/main/WT_8111_API_REFERENCE.md),
  last checked by its maintainers on 2026-05-31
- [WarThunder localhost documentation](https://github.com/lucasvmx/WarThunder-localhost-documentation)
- [War Thunder Flight Desk server](https://github.com/MistressOfDNS/warthunder-ui/blob/main/server.mjs)
- [wt-iox-api](https://github.com/IsraelSiq/wt-iox-api)
- [War Thunder BYOH](https://github.com/SpaceCapo/warthunder-byoh)
- [wthud](https://github.com/wysiwyng/wthud) and
  [wthud2](https://github.com/wysiwyng/wthud2)
- [WarThunder8111](https://github.com/mukasc/WarThunder8111)

Policy context:

- [Gaijin forum discussion about tools using port 8111](https://forum.warthunder.com/t/tools-using-data-provided-on-port-8111/106664)

The forum discussion reports inconsistent staff interpretations. It is not a
guarantee of future policy. The project's safety boundary is therefore stricter
than merely "other tools do it": read only the local HTTP interface and avoid
memory access, injection, hooks, and frame capture.
