import { describe, expect, it } from 'vitest'
import {
  navigationToTarget,
  reconcileSelectedTarget,
  resolveTargetPosition,
  targetFromMapObject,
} from '../navigation'
import type { MapObject } from '../types'
import { navigationSnapshot } from './testFixtures'

describe('moving target tracking', () => {
  const initialObject: MapObject = {
    type: 'ground_model',
    icon: 'LightTank',
    color: '#fa0c00',
    x: 0.5,
    y: 0.5,
  }

  it('follows the nearest matching identity through array insertion and movement', () => {
    const target = targetFromMapObject(initialObject, 0, 7)!
    const snapshot = navigationSnapshot([
      { type: 'defending_point', x: 0.2, y: 0.2 },
      { ...initialObject, x: 0.51, y: 0.5 },
      { type: 'aircraft', icon: 'Player', x: 0.1, y: 0.1 },
    ])

    expect(resolveTargetPosition(snapshot, target)).toEqual({ x: 0.51, y: 0.5 })
    expect(reconcileSelectedTarget(snapshot, target)).toMatchObject({
      x: 0.51,
      y: 0.5,
      object: { index: 1 },
    })

    expect(navigationToTarget(snapshot, target)).toMatchObject({
      targetX: 0.51,
      targetY: 0.5,
    })
  })

  it('drops an ambiguous identity instead of switching to a neighbor', () => {
    const target = targetFromMapObject(initialObject, 0, 7)!
    const snapshot = navigationSnapshot([
      { type: 'defending_point', x: 0.2, y: 0.2 },
      { ...initialObject, x: 0.51, y: 0.5 },
      { ...initialObject, x: 0.52, y: 0.5 },
    ])

    expect(reconcileSelectedTarget(snapshot, target)).toBeNull()
  })

  it('drops objects beyond the tracking radius or with changed identity', () => {
    const target = targetFromMapObject(initialObject, 0, 7)!
    const tooFar = navigationSnapshot([{ ...initialObject, x: 0.526 }])
    const changedColor = navigationSnapshot([{ ...initialObject, x: 0.51, color: '#174dff' }])

    expect(reconcileSelectedTarget(tooFar, target)).toBeNull()
    expect(resolveTargetPosition(changedColor, target)).toBeUndefined()
  })

  it('keeps point targets and invalidates object targets on generation changes', () => {
    const point = {
      key: 'point',
      name: 'Point',
      x: 0.3,
      y: 0.4,
      generation: 7,
      basis: 'map selection' as const,
    }
    const objectTarget = targetFromMapObject(initialObject, 0, 7)!
    const nextGeneration = navigationSnapshot([initialObject], { generation: 8 })

    expect(reconcileSelectedTarget(navigationSnapshot([]), point)).toBe(point)
    expect(reconcileSelectedTarget(nextGeneration, objectTarget)).toBeNull()
  })
})
