# T-80U Ground Realistic fixture

Synthetic fixture for exercising the ground-battle UI without a running game.
Its field names and map-object types follow the open, vehicle-dependent
`localhost:8111` contract, but the values are not a live capture.

- Vehicle identifier: `tankModels/ussr_t_80u`
- Vehicle class: `ground`
- Intended mode: Ground Realistic Battles
- Mission status: `running`

Ground telemetry varies by vehicle and game version. The dashboard therefore
keeps every metric optional and remains useful as a map-first display when
`/state` or `/indicators` omits a value.
