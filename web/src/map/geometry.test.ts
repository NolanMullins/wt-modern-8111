import { describe, expect, it } from 'vitest'
import {
  clientToMapPoint,
  mapToCanvas,
  objectPosition,
  pointInRect,
  scaledDistance,
  squareRect,
} from './geometry'

describe('squareRect', () => {
  it.each([
    [100, 100, { x: 0, y: 0, size: 100 }],
    [200, 100, { x: 50, y: 0, size: 100 }],
    [100, 201, { x: 0, y: 51, size: 100 }],
  ])('centers a square within %d x %d', (width, height, expected) => {
    expect(squareRect(width, height)).toEqual(expected)
  })
})

describe('map coordinate transforms', () => {
  const bounds = { left: 10, top: 20, width: 200, height: 100 }

  it('maps centered viewport corners into normalized coordinates', () => {
    expect(clientToMapPoint({ x: 60, y: 20 }, bounds)).toEqual({ x: 0, y: 0 })
    expect(clientToMapPoint({ x: 160, y: 120 }, bounds)).toEqual({ x: 1, y: 1 })
  })

  it('rejects pointer positions outside the square map', () => {
    expect(clientToMapPoint({ x: 59, y: 50 }, bounds)).toBeUndefined()
    expect(clientToMapPoint({ x: 161, y: 50 }, bounds)).toBeUndefined()
  })

  it('maps normalized positions into canvas coordinates', () => {
    expect(mapToCanvas({ x: 0.25, y: 0.75 }, { x: 50, y: 0, size: 100 }))
      .toEqual({ x: 75, y: 75 })
  })

  it('classifies points against a square viewport', () => {
    const rect = { x: 50, y: 20, size: 100 }

    expect(pointInRect({ x: 50, y: 120 }, rect)).toBe(true)
    expect(pointInRect({ x: 49.9, y: 120 }, rect)).toBe(false)
    expect(pointInRect({ x: 49, y: 120 }, rect, 2)).toBe(true)
  })

  it('scales distance by rectangular world dimensions', () => {
    expect(scaledDistance(
      { x: 0, y: 0 },
      { x: 0.1, y: 0.1 },
      { width: 100_000, height: 10_000 },
    )).toBeCloseTo(Math.hypot(10_000, 1_000))
  })
})

describe('objectPosition', () => {
  it('uses the midpoint of an airfield runway', () => {
    const position = objectPosition({
      type: 'airfield',
      sx: 0.2,
      sy: 0.4,
      ex: 0.4,
      ey: 0.8,
    })

    expect(position?.x).toBeCloseTo(0.3)
    expect(position?.y).toBeCloseTo(0.6)
  })
})
