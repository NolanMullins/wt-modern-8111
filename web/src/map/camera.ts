import {
  clientToMapPoint,
  type MapRect,
  type Point,
  type ViewportBounds,
} from './geometry'

export interface MapCamera {
  center: Point
  zoom: number
}

export const defaultMapCamera: MapCamera = {
  center: { x: 0.5, y: 0.5 },
  zoom: 1,
}

export const maxMapZoom = 8

export function mapRectForCamera(viewport: MapRect, camera: MapCamera): MapRect {
  const normalized = clampCamera(camera)
  const size = viewport.size * normalized.zoom
  return {
    x: viewport.x + viewport.size / 2 - normalized.center.x * size,
    y: viewport.y + viewport.size / 2 - normalized.center.y * size,
    size,
  }
}

export function clientToCameraMapPoint(
  client: Point,
  bounds: ViewportBounds,
  camera: MapCamera,
): Point | undefined {
  const viewportPoint = clientToMapPoint(client, bounds)
  if (!viewportPoint) return undefined
  const normalized = clampCamera(camera)
  const span = 1 / normalized.zoom
  return {
    x: normalized.center.x - span / 2 + viewportPoint.x * span,
    y: normalized.center.y - span / 2 + viewportPoint.y * span,
  }
}

export function zoomCameraAt(
  anchor: Point,
  viewportPoint: Point,
  zoom: number,
): MapCamera {
  const nextZoom = Math.max(1, Math.min(maxMapZoom, zoom))
  const span = 1 / nextZoom
  return clampCamera({
    zoom: nextZoom,
    center: {
      x: anchor.x + (0.5 - viewportPoint.x) * span,
      y: anchor.y + (0.5 - viewportPoint.y) * span,
    },
  })
}

export function panCamera(
  camera: MapCamera,
  movementPixels: Point,
  viewportSize: number,
): MapCamera {
  if (viewportSize <= 0) return camera
  return clampCamera({
    ...camera,
    center: {
      x: camera.center.x - movementPixels.x / (viewportSize * camera.zoom),
      y: camera.center.y - movementPixels.y / (viewportSize * camera.zoom),
    },
  })
}

export function clampCamera(camera: MapCamera): MapCamera {
  const zoom = Math.max(1, Math.min(maxMapZoom, camera.zoom))
  const halfSpan = 1 / zoom / 2
  return {
    zoom,
    center: {
      x: Math.max(halfSpan, Math.min(1 - halfSpan, camera.center.x)),
      y: Math.max(halfSpan, Math.min(1 - halfSpan, camera.center.y)),
    },
  }
}
