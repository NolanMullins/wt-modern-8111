# JH-7 Air Test Flight fixture

Sanitized responses captured from a live, active Air Test Flight on 2026-09-01.

## Capture context

- Vehicle identifier: `jh_7`
- Vehicle class: `air`
- Mission status: `running`
- Game version: not exposed by the API and not recorded
- Map generation: `1`
- Personal chat and HUD history: not included

These files are representative samples, not complete schemas. Fields remain
optional and aircraft-dependent.

The accompanying timed trace was retained as a session artifact rather than
committed because a single snapshot is sufficient for future parser fixtures.
The trace comprised 100 samples each of `/state`, `/indicators`, and
`/map_obj.json` over 10.775 seconds with no request errors.
