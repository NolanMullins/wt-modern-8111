import { objectPosition, type Point } from '../map/geometry'
import type { MapObject, Snapshot } from '../types'
import type { SelectedTarget } from './types'

const targetTrackingRadius = 0.025

export function resolveTargetPosition(
  snapshot: Snapshot,
  target: SelectedTarget,
): Point | undefined {
  if (!target.object) return { x: target.x, y: target.y }
  return trackedObject(snapshot, target)?.position
}

export function reconcileSelectedTarget(
  snapshot: Snapshot,
  target: SelectedTarget | null,
): SelectedTarget | null {
  if (!target) return null
  if (
    target.generation !== undefined &&
    snapshot.map.generation !== undefined &&
    target.generation !== snapshot.map.generation
  ) return null
  if (!target.object) return target

  const tracked = trackedObject(snapshot, target)
  if (!tracked) return null
  if (
    tracked.index === target.object.index &&
    tracked.position.x === target.x &&
    tracked.position.y === target.y
  ) return target

  return {
    ...target,
    x: tracked.position.x,
    y: tracked.position.y,
    object: { ...target.object, index: tracked.index },
  }
}

function trackedObject(snapshot: Snapshot, target: SelectedTarget) {
  const identity = target.object
  if (!identity) return undefined
  const nearest = snapshot.map.objects
    .map((object, index) => ({ object, index }))
    .filter(({ object }) => sameObjectKind(object, identity))
    .map((entry) => {
      const position = objectPosition(entry.object)
      return position
        ? {
            ...entry,
            position,
            distance: Math.hypot(position.x - target.x, position.y - target.y),
          }
        : undefined
    })
    .filter((entry): entry is NonNullable<typeof entry> => entry !== undefined)
    .sort((left, right) => left.distance - right.distance)[0]
  return nearest?.distance <= targetTrackingRadius ? nearest : undefined
}

function sameObjectKind(
  object: MapObject,
  identity: NonNullable<SelectedTarget['object']>,
) {
  return object.type === identity.type &&
    object.icon === identity.icon &&
    object.color === identity.color
}
