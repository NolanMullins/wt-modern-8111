import { describe, expect, it } from 'vitest'
import { mapPointTarget, navigationToTarget } from '../navigation'
import { navigationSnapshot } from './testFixtures'

describe('navigationToTarget', () => {
  it('calculates range, bearing, and ETA in physical map dimensions', () => {
    const snapshot = navigationSnapshot(
      [{ type: 'aircraft', icon: 'Player', x: 0.1, y: 0.2 }],
      { mapMax: [100_000, 50_000], tasKmh: 400 },
    )
    const target = mapPointTarget(0.2, 0.1, 7)

    const navigation = navigationToTarget(snapshot, target)

    expect(navigation?.targetX).toBe(0.2)
    expect(navigation?.targetY).toBe(0.1)
    expect(navigation?.rangeKm).toBeCloseTo(Math.hypot(10_000, -5_000) / 1000)
    expect(navigation?.bearingDeg).toBeCloseTo(63.4349488)
    expect(navigation?.etaSeconds).toBeCloseTo(
      Math.hypot(10_000, -5_000) / 1000 / 400 * 3600,
    )
  })

  it('rejects a target from another map generation', () => {
    const snapshot = navigationSnapshot([
      { type: 'aircraft', icon: 'Player', x: 0.1, y: 0.2 },
    ])

    expect(navigationToTarget(snapshot, mapPointTarget(0.2, 0.1, 8))).toBeUndefined()
  })
})
