import { describe, expect, it } from 'vitest'
import {
  clientToCameraMapPoint,
  defaultMapCamera,
  mapRectForCamera,
  panCamera,
  zoomCameraAt,
} from './camera'

describe('map camera', () => {
  it('preserves the full-map transform at zoom one', () => {
    expect(mapRectForCamera(
      { x: 50, y: 0, size: 100 },
      defaultMapCamera,
    )).toEqual({ x: 50, y: 0, size: 100 })
  })

  it('keeps the cursor map point fixed while zooming', () => {
    const bounds = { left: 0, top: 0, width: 100, height: 100 }
    const cursor = { x: 75, y: 25 }
    const anchor = clientToCameraMapPoint(cursor, bounds, defaultMapCamera)!
    const zoomed = zoomCameraAt(anchor, { x: 0.75, y: 0.25 }, 2)

    expect(clientToCameraMapPoint(cursor, bounds, zoomed)).toEqual(anchor)
  })

  it('pans opposite the pointer movement and clamps to the map', () => {
    const camera = { center: { x: 0.5, y: 0.5 }, zoom: 2 }

    expect(panCamera(camera, { x: 20, y: -10 }, 100)).toEqual({
      center: { x: 0.4, y: 0.55 },
      zoom: 2,
    })
    expect(panCamera(camera, { x: 1000, y: 1000 }, 100)).toEqual({
      center: { x: 0.25, y: 0.25 },
      zoom: 2,
    })
  })

  it('maps viewport coordinates through the visible camera span', () => {
    const camera = { center: { x: 0.75, y: 0.25 }, zoom: 2 }
    const bounds = { left: 0, top: 0, width: 100, height: 100 }

    expect(clientToCameraMapPoint({ x: 0, y: 0 }, bounds, camera))
      .toEqual({ x: 0.5, y: 0 })
    expect(clientToCameraMapPoint({ x: 100, y: 100 }, bounds, camera))
      .toEqual({ x: 1, y: 0.5 })
  })
})
