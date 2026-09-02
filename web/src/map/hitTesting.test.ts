import { describe, expect, it } from 'vitest'
import { nearestSelectableMapObjectHit } from './hitTesting'
import type { MapObject } from '../types'

describe('nearestSelectableMapObjectHit', () => {
  const objects: MapObject[] = [
    { type: 'aircraft', icon: 'Player', x: 0.5, y: 0.5 },
    { type: 'ground_model', icon: 'SAM', x: 0.51, y: 0.5 },
    { type: 'ground_model', icon: 'LightTank', x: 0.52, y: 0.5 },
    { type: 'ground_model', icon: 'invalid' },
  ]

  it('selects the closest non-player object within the radius', () => {
    const hit = nearestSelectableMapObjectHit(objects, { x: 0.5, y: 0.5 }, 0.03)

    expect(hit?.index).toBe(1)
    expect(hit?.distance).toBeCloseTo(0.01)
    expect(hit?.object.icon).toBe('SAM')
  })

  it('returns no hit outside the radius', () => {
    expect(nearestSelectableMapObjectHit(objects, { x: 0.8, y: 0.8 }, 0.03))
      .toBeUndefined()
  })

  it('selects an airfield from a visible runway endpoint', () => {
    const runway: MapObject = {
      type: 'airfield',
      sx: 0.1,
      sy: 0.2,
      ex: 0.5,
      ey: 0.2,
    }

    expect(nearestSelectableMapObjectHit([runway], { x: 0.49, y: 0.2 }, 0.02)?.object)
      .toBe(runway)
  })
})
