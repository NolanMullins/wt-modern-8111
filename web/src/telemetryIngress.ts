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
  if (!current || candidate.sequence > current.sequence) return candidate
  if (candidate.sequence === current.sequence) return current

  const currentTime = Date.parse(current.capturedAt)
  const candidateTime = Date.parse(candidate.capturedAt)
  return candidateTime > currentTime ? candidate : current
}

function isSnapshot(value: unknown): value is Snapshot {
  if (!isRecord(value)) return false
  return value.version === 1 &&
    isFiniteNumber(value.sequence) &&
    value.sequence >= 0 &&
    isTimestamp(value.capturedAt) &&
    isRecord(value.connection) &&
    isRecord(value.vehicle) &&
    isRecord(value.flight) &&
    isRecord(value.systems) &&
    isRecord(value.mission) &&
    isRecord(value.map) &&
    Array.isArray(value.feed) &&
    isRecord(value.pilot) &&
    Array.isArray(value.allyMarks)
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

function isFiniteNumber(value: unknown): value is number {
  return typeof value === 'number' && Number.isFinite(value)
}

function isTimestamp(value: unknown): value is string {
  return typeof value === 'string' && Number.isFinite(Date.parse(value))
}
