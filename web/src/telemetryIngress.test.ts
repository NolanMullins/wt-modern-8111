import { describe, expect, it } from 'vitest'
import { parseSnapshotPayload, selectNewerSnapshot } from './telemetryIngress'
import type { Snapshot } from './types'

describe('parseSnapshotPayload', () => {
  it('accepts the versioned snapshot envelope', () => {
    const snapshot = makeSnapshot(4, '2026-09-01T22:00:04Z')

    expect(parseSnapshotPayload(JSON.stringify(snapshot))).toEqual({
      ok: true,
      snapshot,
    })
  })

  it('rejects malformed and unsupported snapshots', () => {
    expect(parseSnapshotPayload('{')).toMatchObject({ ok: false })
    expect(parseSnapshotPayload(JSON.stringify({
      ...makeSnapshot(4, '2026-09-01T22:00:04Z'),
      version: 2,
    }))).toEqual({
      ok: false,
      error: 'Snapshot payload does not match the v1 envelope',
    })
    expect(parseSnapshotPayload(JSON.stringify({
      ...makeSnapshot(4, '2026-09-01T22:00:04Z'),
      systems: { engines: null },
    }))).toMatchObject({ ok: false })
  })
})

describe('selectNewerSnapshot', () => {
  it('does not let an older bootstrap response replace streamed data', () => {
    const streamed = makeSnapshot(12, '2026-09-01T22:00:12Z')
    const bootstrap = makeSnapshot(10, '2026-09-01T22:00:10Z')

    expect(selectNewerSnapshot(streamed, bootstrap)).toBe(streamed)
  })

  it('accepts a restarted service with a newer capture time', () => {
    const beforeRestart = makeSnapshot(50, '2026-09-01T22:00:50Z')
    const afterRestart = makeSnapshot(1, '2026-09-01T22:01:00Z')

    expect(selectNewerSnapshot(beforeRestart, afterRestart)).toBe(afterRestart)
  })

  it('rejects a delayed pre-restart snapshot with a higher sequence', () => {
    const afterRestart = makeSnapshot(1, '2026-09-01T22:01:00Z')
    const delayedOldProcess = makeSnapshot(50, '2026-09-01T22:00:50Z')

    expect(selectNewerSnapshot(afterRestart, delayedOldProcess)).toBe(afterRestart)
  })
})

function makeSnapshot(sequence: number, capturedAt: string): Snapshot {
  return {
    version: 1,
    sequence,
    capturedAt,
    connection: { state: 'live', mode: 'live', sources: {} },
    vehicle: {},
    flight: {},
    systems: {
      status: 'Nominal',
      severity: 'good',
      warnings: [],
      engines: [],
    },
    mission: { objectives: [] },
    map: {
      valid: true,
      objects: [],
      counts: {
        total: 0,
        hostileAir: 0,
        ground: 0,
        airDefense: 0,
        strikePoint: 0,
        airfield: 0,
      },
    },
    feed: [],
    pilot: { confirmed: false },
    allyMarks: [],
  }
}
