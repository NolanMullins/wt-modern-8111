import type { MapObject } from '../types'

export interface Point {
  x: number
  y: number
}

export interface MapRect extends Point {
  size: number
}

export interface ViewportBounds {
  left: number
  top: number
  width: number
  height: number
}

export function squareRect(width: number, height: number): MapRect {
  const size = Math.min(width, height)
  return {
    x: Math.round((width - size) / 2),
    y: Math.round((height - size) / 2),
    size,
  }
}

export function mapToCanvas(point: Point, rect: MapRect): Point {
  return {
    x: rect.x + point.x * rect.size,
    y: rect.y + point.y * rect.size,
  }
}

export function scaledDistance(
  first: Point,
  second: Point,
  dimensions: { width: number; height: number },
) {
  return Math.hypot(
    (first.x - second.x) * dimensions.width,
    (first.y - second.y) * dimensions.height,
  )
}

export function clientToMapPoint(
  client: Point,
  bounds: ViewportBounds,
): Point | undefined {
  const size = Math.min(bounds.width, bounds.height)
  if (size <= 0) return undefined
  const left = bounds.left + (bounds.width - size) / 2
  const top = bounds.top + (bounds.height - size) / 2
  const point = {
    x: (client.x - left) / size,
    y: (client.y - top) / size,
  }
  return point.x >= 0 && point.x <= 1 && point.y >= 0 && point.y <= 1
    ? point
    : undefined
}

export function objectPosition(object: MapObject): Point | undefined {
  if (
    object.type === 'airfield' &&
    finite(object.sx) &&
    finite(object.sy) &&
    finite(object.ex) &&
    finite(object.ey)
  ) {
    return {
      x: (object.sx + object.ex) / 2,
      y: (object.sy + object.ey) / 2,
    }
  }
  return finite(object.x) && finite(object.y)
    ? { x: object.x, y: object.y }
    : undefined
}

export function distanceToMapObject(point: Point, object: MapObject): number | undefined {
  if (
    object.type === 'airfield' &&
    finite(object.sx) &&
    finite(object.sy) &&
    finite(object.ex) &&
    finite(object.ey)
  ) {
    const dx = object.ex - object.sx
    const dy = object.ey - object.sy
    const lengthSquared = dx * dx + dy * dy
    const interpolation = lengthSquared === 0
      ? 0
      : Math.max(
          0,
          Math.min(
            1,
            ((point.x - object.sx) * dx + (point.y - object.sy) * dy) / lengthSquared,
          ),
        )
    return Math.hypot(
      point.x - (object.sx + interpolation * dx),
      point.y - (object.sy + interpolation * dy),
    )
  }
  const position = objectPosition(object)
  return position ? Math.hypot(point.x - position.x, point.y - position.y) : undefined
}

function finite(value: number | undefined): value is number {
  return value !== undefined && Number.isFinite(value)
}
