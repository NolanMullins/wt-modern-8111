import { describe, expect, it } from 'vitest'
import {
  inGameTarget,
  missionTarget,
  targetFromMapObject,
} from '../navigation'
import { navigationSnapshot } from './testFixtures'

describe('target construction and inference', () => {
  it('infers an airfield using distance to its runway segment', () => {
    const snapshot = navigationSnapshot([
      { type: 'point_of_interest', x: 0.2, y: 0.429 },
      { type: 'airfield', color: '#174dff', sx: 0.1, sy: 0.4, ex: 0.9, ey: 0.4 },
    ])

    expect(inGameTarget(snapshot)).toMatchObject({
      name: 'Friendly airfield',
      x: 0.5,
      y: 0.4,
      basis: 'game target',
      object: { index: 1, type: 'airfield' },
    })
  })

  it('keeps the point of interest when inference exceeds radius 0.03', () => {
    const snapshot = navigationSnapshot([
      { type: 'point_of_interest', x: 0.2, y: 0.431 },
      { type: 'airfield', color: '#174dff', sx: 0.1, sy: 0.4, ex: 0.9, ey: 0.4 },
    ])

    expect(inGameTarget(snapshot)).toMatchObject({
      name: 'In-game target point',
      x: 0.2,
      y: 0.431,
      basis: 'game target',
      object: { index: 0, type: 'point_of_interest' },
    })
  })

  it('chooses mission targets by physical rather than normalized distance', () => {
    const snapshot = navigationSnapshot(
      [
        { type: 'aircraft', icon: 'Player', x: 0, y: 0 },
        { type: 'bombing_point', color: '#fa0c00', x: 0.1, y: 0 },
        { type: 'bombing_point', color: '#fa0c00', x: 0, y: 0.2 },
      ],
      { mapMax: [100_000, 10_000] },
    )
    const objective = { primary: true, status: 'active', text: 'Bomb enemy bases' }

    expect(missionTarget(snapshot, objective)).toMatchObject({
      name: objective.text,
      x: 0,
      y: 0.2,
      basis: 'mission objective',
      object: { index: 2 },
    })
  })

  it.each([
    [{ type: 'ground_model', icon: 'SAM', x: 0.1, y: 0.2 }, 'SAM position'],
    [{ type: 'ground_model', icon: 'SPAA', x: 0.1, y: 0.2 }, 'Air-defense position'],
    [{ type: 'bombing_point', x: 0.1, y: 0.2 }, 'Strike point'],
  ])('preserves the label for $0', (object, label) => {
    expect(targetFromMapObject(object, 3)?.name).toBe(label)
  })
})
