# Passive second-screen UX concepts

These concepts are built for a pilot who cannot interact with the display while
flying. They do not contain navigation buttons, drawers, tabs, map controls, or
click-dependent information.

Open the HTML files through a local HTTP server. They use the sanitized JH-7
map and telemetry fixtures and clearly identify mission, navigation, and feed
content as illustrative. These static concepts never claim live freshness and
never poll War Thunder directly; runtime data remains the future companion's
responsibility.

## Actual War Thunder map findings

The concepts are based on a direct audit of Gaijin's built-in `8111` page:

- `/map.img` is a small raster stretched to the map canvas.
- In the captured Air Test Flight, it is a 128×128 grayscale question-mark
  placeholder dominated by `#404040` and `#808080`, not a terrain chart.
- The tactical grid is generated from `map_min`, `map_max`, and `grid_steps`.
  This matches Gaijin's current built-in drawing code; it does not use
  `grid_zero`, whose exact label semantics remain unresolved.
- Airfields are lines built from `sx`, `sy`, `ex`, and `ey`.
- The player is a white directional arrow with a dark outline.
- Tactical-object colors come from each `/map_obj.json` entry.
- Gaijin's `icons.ttf` is served locally as `text/plain` without CORS headers.
  Production should proxy it same-origin at runtime or use a separately
  designed, distributable symbol set. It should not be copied into the
  repository.
- The built-in page explicitly presents primary and secondary objectives, game
  chat, HUD events, damage messages, indicators, and raw state.

## Shared information contract

Every concept shows each important value in one place only:

- Actual map raster/grid/object model with automatic full-mission framing
- Primary and secondary objectives
- Combined chronological chat, HUD event, and damage feed
- Automatic active-destination navigation
- IAS, TAS, altitude, Mach, heading, vertical speed, AoA, and G
- Combined twin-engine status, RPM, oil temperature, thrust, and fuel
- Connection and source state

The navigation example is calculated from the captured player position to the
nearest captured bombing point: 157 degrees, 10.9 km, and approximately 46
seconds at the captured TAS. Mission objectives and feed messages are visibly
marked illustrative because the captured Test Flight returned null objectives
and empty incremental feeds.

The visual system includes distinct fixture, live, stale, and offline status
styles. Only the truthful prototype/fixture state is used by these static
concepts; live and freshness states must be driven by the complete normalized
snapshot in the production application.

## Concepts

| File | Layout thesis |
| --- | --- |
| `01-balanced-flight-desk.html` | Follow-framed map with balanced aircraft and mission columns |
| `02-briefing-columns.html` | Full-mission map with an action-priority event model |
| `03-right-seat-kneeboard.html` | Two-tier aircraft/mission hierarchy with the map on the right |
| `04-command-center.html` | Threat-framed map and warning-priority feed |
| `05-split-operations.html` | Full-mission map between persistent mission and comms columns |

The visual language uses War Thunder's black surfaces, white condensed display
type, hard red accents, thin dividers, and restrained battlefield amber. Figma's
2026 trends are applied through bold typography, immersive composition,
asymmetry, responsive personalization, subtle depth, and purposeful motion—not
through decorative novelty.
