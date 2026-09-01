import type { MapObject } from '../types'
import { objectPosition, type Point } from './geometry'

export interface MapObjectHit {
  object: MapObject
  index: number
  position: Point
  distance: number
}

export function nearestSelectableMapObjectHit(
  objects: readonly MapObject[],
  point: Point,
  radius: number,
): MapObjectHit | undefined {
  let nearest: MapObjectHit | undefined
  objects.forEach((object, index) => {
    if (object.icon === 'Player') return
    const position = objectPosition(object)
    if (!position) return
    const distance = Math.hypot(position.x - point.x, position.y - point.y)
    if (distance > radius || (nearest && distance >= nearest.distance)) return
    nearest = { object, index, position, distance }
  })
  return nearest
}
