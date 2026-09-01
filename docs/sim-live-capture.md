# Simulator live-capture protocol

## Goal

This protocol converts the unknowns in the
[API research](localhost-8111-api.md) into reproducible observations. Captures
should be collected before the production normalizer, navigation calculations,
or polling service are treated as stable.

Do not capture or publish private chat without redacting player names and
messages.

## Capture principles

- Record raw responses before normalization.
- Record response headers as well as bodies.
- Use the same game version for a comparison set.
- Change one factor at a time.
- Record aircraft, mode, map, mission phase, cockpit unit system, and local
  time with every sample.
- Label a field as unavailable rather than assigning it a default value.
- Repeat surprising results in a fresh session.

## Minimum scenario matrix

| Scenario | Purpose |
| --- | --- |
| Game not running | Connection-refused baseline |
| Game running in hangar | Reachable-but-invalid baseline |
| Air Test Flight, single-engine prop | Prop engine fields and transition timing |
| Air Test Flight, single-engine jet | Jet-specific state and indicator fields |
| Air Test Flight, multi-engine aircraft | Engine-index discovery |
| Air Arcade | Mode comparison |
| Air Realistic | Mode comparison and map visibility |
| Air Simulator | Primary product mode and map visibility |
| Spawn selection/loading | Incomplete map and mission payload behavior |
| Spawned and stationary | Initial valid-data transition |
| Normal flight | Stable polling and units |
| Death/spectating | Player marker and validity transitions |
| Return to hangar | Session-end and cursor behavior |

If practical, add one variable-sweep aircraft and one aircraft with imperial
cockpit instruments.

## Endpoints to record

For each scenario, capture:

```text
/
/state
/indicators
/map_info.json
/map_obj.json
/map.img
/mission.json
/hudmsg?lastEvt=0&lastDmg=0
/gamechat?lastId=0
/loc/map/primary_objectives?fmt=js
/loc/map/secondary_objectives?fmt=js
```

For `/map.img`, save the binary response and record its `Content-Type`,
`Content-Length`, cache headers, and current `map_generation`.

## One-shot PowerShell capture

Run from a new, scenario-specific directory. Replace the metadata values before
each capture.

```powershell
$baseUrl = "http://127.0.0.1:8111"
$metadata = [ordered]@{
  capturedAt = (Get-Date).ToString("o")
  gameVersion = "<record manually>"
  mode = "Air Simulator"
  phase = "normal flight"
  aircraft = "<record internal and display names>"
  map = "<record manually>"
}
$metadata | ConvertTo-Json | Set-Content -Encoding utf8 metadata.json

$jsonEndpoints = [ordered]@{
  state = "/state"
  indicators = "/indicators"
  mapInfo = "/map_info.json"
  mapObjects = "/map_obj.json"
  mission = "/mission.json"
  hudMessages = "/hudmsg?lastEvt=0&lastDmg=0"
  gameChat = "/gamechat?lastId=0"
}

foreach ($entry in $jsonEndpoints.GetEnumerator()) {
  $response = Invoke-WebRequest -Uri ($baseUrl + $entry.Value) -TimeoutSec 2
  $response.Content | Set-Content -Encoding utf8 ($entry.Key + ".json")
  $response.Headers | ConvertTo-Json -Depth 4 |
    Set-Content -Encoding utf8 ($entry.Key + ".headers.json")
}

$map = Invoke-WebRequest -Uri ($baseUrl + "/map.img") -TimeoutSec 5
[System.IO.File]::WriteAllBytes(
  (Join-Path $PWD "map.img"),
  $map.Content
)
$map.Headers | ConvertTo-Json -Depth 4 |
  Set-Content -Encoding utf8 map.headers.json
```

PowerShell versions differ in how binary `Invoke-WebRequest` content is
represented. Confirm the saved map opens as an image. If it does not, use the
system `curl.exe` for only that request:

```powershell
curl.exe --silent --show-error --dump-header map.headers.txt `
  --output map.img http://127.0.0.1:8111/map.img
```

An offline capture is expected to fail. Record the exception type and elapsed
time instead of creating success-shaped JSON.

## Short transition trace

A one-shot sample cannot reveal loading and lifecycle behavior. For transitions
such as hangar to spawn or death to spectator, sample `/state`, `/indicators`,
`/map_info.json`, `/map_obj.json`, and `/mission.json` at 5 Hz for at least 15
seconds before and after the transition.

Each record should include:

```json
{
  "requestedAt": "ISO-8601 timestamp",
  "completedAt": "ISO-8601 timestamp",
  "endpoint": "/state",
  "statusCode": 200,
  "contentType": "application/json",
  "elapsedMs": 3.2,
  "body": {}
}
```

Store one JSON object per line so an interrupted trace remains readable. Never
issue another request to the same endpoint while its previous request is still
pending.

## Navigation verification

To verify normalized coordinates, distance, and bearing:

1. Use a map with clearly identifiable runway endpoints or grid intersections.
2. Record `map_info.json`, `map_obj.json`, and the map image in the same
   `map_generation`.
3. Compare the normalized player coordinate to its pixel location on the raw
   image.
4. Select points directly north, east, south, and west of the player.
5. Compare calculated bearing with the in-game compass/map indication.
6. Fly a known map-grid distance and compare calculated world distance.
7. Record enough samples to determine how `grid_zero` and `grid_steps` create
   the displayed alphanumeric grid.

Do not ship bearing or grid labels based only on visual intuition; record the
expected and calculated values.

## Contact-visibility verification

For each Air AB/RB/SB mode:

1. Capture map objects with no known enemy contact.
2. Capture before and after an enemy becomes visible on the in-game tactical
   map.
3. Capture after the contact is lost.
4. Compare added and removed objects, including every field.
5. Confirm that WT Modern 8111 would display exactly the locally exposed
   objects and no inferred contacts.

This test must be performed in ordinary play without coordinating behavior that
violates game rules.

## Field compatibility table

For each aircraft, build a generated table with:

| Field | Endpoint | JSON type | Observed unit | Min | Max | Missing phases | Sentinel |
| --- | --- | --- | --- | --- | --- | --- | --- |
| Example: `IAS, km/h` | `/state` | number | km/h | 0 | ... | hangar | none seen |

Pay particular attention to:

- altitude in feet versus meters
- fuel mass versus instrument quantity
- compass convention and wraparound
- vertical-speed sign
- gear/flap ranges and partial positions
- propeller pitch, mixture, radiator, and supercharger fields
- variable wing sweep
- absent versus negative not-equipped values
- multi-engine numbering gaps

## Acceptance criteria for the API baseline

The research phase is complete enough to implement the first milestone when:

- Air Simulator has a full hangar-to-flight-to-hangar trace.
- At least one prop, one jet, and one multi-engine aircraft have field tables.
- The map image and player marker align for two maps.
- Bearing cardinal directions and distance scale are verified.
- Map contact visibility is compared across Air AB/RB/SB.
- Offline, invalid, loading, live, stale, and partial-failure states have
  captured examples.
- `map_generation` behavior is observed across a mission change.
- Cursor behavior is observed across at least two matches without restarting
  the game.
- Every production fixture is redacted and records the game version that
  produced it.

Captures that meet these criteria can become fixtures for the future companion
service and mock War Thunder server.
