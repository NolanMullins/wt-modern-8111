import type { Snapshot } from './types'

export type SnapshotParseResult =
  | { ok: true; snapshot: Snapshot }
  | { ok: false; error: string }

export function parseSnapshotPayload(payload: string): SnapshotParseResult {
  let value: unknown
  try {
    value = JSON.parse(payload)
  } catch (error: unknown) {
    const detail = error instanceof Error ? error.message : 'unknown parse error'
    return { ok: false, error: `Invalid snapshot JSON: ${detail}` }
  }
  if (!isSnapshot(value)) {
    return { ok: false, error: 'Snapshot payload does not match the v1 envelope' }
  }
  return { ok: true, snapshot: value }
}

export function selectNewerSnapshot(
  current: Snapshot | null,
  candidate: Snapshot,
): Snapshot {
  if (!current) return candidate
  const currentTime = Date.parse(current.capturedAt)
  const candidateTime = Date.parse(candidate.capturedAt)
  if (candidateTime !== currentTime) {
    return candidateTime > currentTime ? candidate : current
  }
  return candidate.sequence > current.sequence ? candidate : current
}

function isSnapshot(value: unknown): value is Snapshot {
  if (!isRecord(value)) return false
  const connection = value.connection
  const systems = value.systems
  const mission = value.mission
  const map = value.map
  const pilot = value.pilot
  const ground = value.ground
  return value.version === 1 &&
    isFiniteNumber(value.sequence) &&
    value.sequence >= 0 &&
    isTimestamp(value.capturedAt) &&
    isConnection(connection) &&
    isRecord(value.vehicle) &&
    isRecord(value.flight) &&
    (ground === undefined || isGround(ground)) &&
    isSystems(systems) &&
    isMission(mission) &&
    isMap(map) &&
    Array.isArray(value.feed) &&
    value.feed.every(isFeedEntry) &&
    isRecord(pilot) &&
    typeof pilot.confirmed === 'boolean' &&
    Array.isArray(value.allyMarks) &&
    value.allyMarks.every(isAllyMark)
}

function isGround(value: unknown) {
  if (!isRecord(value)) return false
  return [
    'speedKmh',
    'headingDeg',
    'engineRpm',
    'gear',
    'cruiseControl',
    'ammo',
    'crewCurrent',
    'crewTotal',
    'driverState',
    'gunnerState',
    'stabilizer',
    'lws',
    'ircm',
    'engineBroken',
    'speedWarning',
  ].every((key) => value[key] === undefined || isFiniteNumber(value[key]))
}

function isConnection(value: unknown) {
  return isRecord(value) &&
    includes(['waiting-for-game', 'hangar', 'live', 'degraded'], value.state) &&
    includes(['live', 'fixture'], value.mode) &&
    isRecord(value.sources)
}

function isSystems(value: unknown) {
  return isRecord(value) &&
    typeof value.status === 'string' &&
    includes(['good', 'caution', 'critical', 'unknown'], value.severity) &&
    isStringArray(value.warnings) &&
    Array.isArray(value.engines) &&
    value.engines.every((engine) =>
      isRecord(engine) &&
      isFiniteNumber(engine.index) &&
      includes(['running', 'idle', 'failed', 'unknown'], engine.status) &&
      typeof engine.running === 'boolean',
    )
}

function isMission(value: unknown) {
  return isRecord(value) &&
    Array.isArray(value.objectives) &&
    value.objectives.every((objective) =>
      isRecord(objective) &&
      typeof objective.primary === 'boolean' &&
      typeof objective.status === 'string' &&
      typeof objective.text === 'string',
    )
}

function isMap(value: unknown) {
  if (!isRecord(value) || !isRecord(value.counts)) return false
  const counts = value.counts
  return typeof value.valid === 'boolean' &&
    (value.imageRevision === undefined || isFiniteNumber(value.imageRevision)) &&
    (value.hudType === undefined || isFiniteNumber(value.hudType)) &&
    Array.isArray(value.objects) &&
    value.objects.every((object) => isRecord(object) && typeof object.type === 'string') &&
    ['total', 'hostileAir', 'ground', 'airDefense', 'strikePoint', 'airfield']
      .every((key) => isFiniteNumber(counts[key])) &&
    ['friendlyGround', 'hostileGround', 'captureZone', 'groundSpawn']
      .every((key) => counts[key] === undefined || isFiniteNumber(counts[key]))
}

function isFeedEntry(value: unknown) {
  return isRecord(value) &&
    typeof value.key === 'string' &&
    includes(['chat', 'event', 'damage'], value.kind) &&
    typeof value.addedAt === 'string' &&
    typeof value.message === 'string'
}

function isAllyMark(value: unknown) {
  return isRecord(value) &&
    typeof value.key === 'string' &&
    typeof value.kind === 'string' &&
    typeof value.sender === 'string' &&
    typeof value.message === 'string' &&
    typeof value.located === 'boolean' &&
    isTimestamp(value.createdAt) &&
    isTimestamp(value.expiresAt)
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

function isFiniteNumber(value: unknown): value is number {
  return typeof value === 'number' && Number.isFinite(value)
}

function isStringArray(value: unknown): value is string[] {
  return Array.isArray(value) && value.every((entry) => typeof entry === 'string')
}

function includes<const Value extends string>(
  values: readonly Value[],
  value: unknown,
): value is Value {
  return typeof value === 'string' && values.includes(value as Value)
}

function isTimestamp(value: unknown): value is string {
  return typeof value === 'string' && Number.isFinite(Date.parse(value))
}
