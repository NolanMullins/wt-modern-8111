export interface SourceStatus {
  state: 'fresh' | 'stale' | 'error' | 'unavailable'
  lastSuccess?: string
  ageMs?: number
  error?: string
}

export interface MapObject {
  type: string
  color?: string
  blink?: number
  icon?: string
  icon_bg?: string
  x?: number
  y?: number
  dx?: number
  dy?: number
  sx?: number
  sy?: number
  ex?: number
  ey?: number
}

export interface Snapshot {
  version: number
  sequence: number
  capturedAt: string
  connection: {
    state: 'waiting-for-game' | 'hangar' | 'live' | 'degraded'
    mode: 'live' | 'fixture'
    sources: Record<string, SourceStatus>
  }
  vehicle: {
    type?: string
    class?: string
  }
  flight: {
    iasKmh?: number
    tasKmh?: number
    altitudeM?: number
    radioAltitudeM?: number
    mach?: number
    headingDeg?: number
    verticalSpeedMps?: number
    aoaDeg?: number
    gLoad?: number
  }
  systems: {
    status: string
    severity: 'good' | 'caution' | 'critical' | 'unknown'
    warnings: string[]
    engines: Array<{
      index: number
      throttlePercent?: number
      rpm?: number
      oilTempC?: number
      thrustKgf?: number
      status: 'running' | 'idle' | 'failed' | 'unknown'
      running: boolean
    }>
    fuelKg?: number
    fuelPercent?: number
    gearPercent?: number
    flapsPercent?: number
    airbrakePercent?: number
  }
  navigation?: {
    name: string
    bearingDeg: number
    rangeKm: number
    etaSeconds?: number
    targetX: number
    targetY: number
    basis: string
  }
  mission: {
    status?: string
    objectives: Array<{
      primary: boolean
      status: string
      text: string
    }>
  }
  map: {
    valid: boolean
    generation?: number
    gridSize?: number[]
    gridSteps?: number[]
    gridZero?: number[]
    mapMin?: number[]
    mapMax?: number[]
    objects: MapObject[]
    counts: {
      total: number
      hostileAir: number
      ground: number
      airDefense: number
      strikePoint: number
      airfield: number
    }
  }
  feed: Array<{
    key: string
    kind: 'chat' | 'event' | 'damage'
    time?: number
    addedAt: string
    sender?: string
    message: string
    enemy?: boolean
  }>
  pilot: {
    callsign?: string
    confirmed: boolean
  }
  allyMarks: Array<{
    key: string
    kind: 'guide' | 'attention' | 'cover' | 'help'
    sender: string
    message: string
    grid?: string
    x?: number
    y?: number
    located: boolean
    createdAt: string
    expiresAt: string
  }>
}
