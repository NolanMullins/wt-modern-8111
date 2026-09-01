import { objectPosition, scaledDistance, type Point } from '../map/geometry'
import type { MapObject, Snapshot } from '../types'
import type { SelectedTarget } from './types'

const targetInferenceRadius = 0.03

export function targetFromMapObject(
  object: MapObject,
  index: number,
  generation?: number,
): SelectedTarget | undefined {
  if (object.icon === 'Player') return undefined
  const position = objectPosition(object)
  if (!position) return undefined
  const label = objectLabel(object)
  return {
    key: `${object.type}-${object.icon ?? 'object'}-${index}`,
    name: label,
    x: position.x,
    y: position.y,
    generation,
    basis: 'map selection',
    object: {
      index,
      type: object.type,
      icon: object.icon,
      color: object.color,
    },
  }
}

export function mapPointTarget(
  x: number,
  y: number,
  generation?: number,
): SelectedTarget {
  return {
    key: `point-${x.toFixed(4)}-${y.toFixed(4)}`,
    name: 'Selected map point',
    x,
    y,
    generation,
    basis: 'map selection',
  }
}

export function inGameTarget(snapshot: Snapshot | null): SelectedTarget | undefined {
  if (!snapshot?.map.valid) return undefined
  const entry = indexed(snapshot.map.objects).find(
    ({ object }) => object.type === 'point_of_interest' && objectPosition(object) !== undefined,
  )
  if (!entry) return undefined
  const position = objectPosition(entry.object)!
  const inferred = indexed(snapshot.map.objects)
    .filter(({ object }) => targetableMapObject(object))
    .map((candidate) => ({
      ...candidate,
      distance: distanceToObject(position, candidate.object),
    }))
    .filter((candidate) => candidate.distance !== undefined)
    .sort((left, right) => left.distance! - right.distance!)[0]
  if (inferred?.distance !== undefined && inferred.distance <= targetInferenceRadius) {
    const target = targetFromMapObject(
      inferred.object,
      inferred.index,
      snapshot.map.generation,
    )
    if (target) {
      return {
        ...target,
        key: `game-target-${position.x.toFixed(6)}-${position.y.toFixed(6)}-${target.key}`,
        basis: 'game target',
      }
    }
  }
  return {
    key: `game-target-${position.x.toFixed(6)}-${position.y.toFixed(6)}`,
    name: 'In-game target point',
    x: position.x,
    y: position.y,
    generation: snapshot.map.generation,
    basis: 'game target',
    object: {
      index: entry.index,
      type: entry.object.type,
      icon: entry.object.icon,
      color: entry.object.color,
    },
  }
}

export function missionTarget(
  snapshot: Snapshot,
  objective: Snapshot['mission']['objectives'][number],
): SelectedTarget | undefined {
  const message = objective.text.toLowerCase()
  let candidates: Array<{ object: MapObject; index: number }> = []
  const objects = snapshot.map.objects

  if (message.includes('attacker')) {
    candidates = indexed(objects).filter(({ object }) =>
      object.type === 'aircraft' && object.icon === 'Assault' && hostile(object),
    )
  }
  if (candidates.length === 0 && (message.includes('base') || message.includes('bomb'))) {
    candidates = indexed(objects).filter(({ object }) =>
      object.type === 'bombing_point' && hostile(object),
    )
  }
  if (candidates.length === 0 && (message.includes('defend') || message.includes('cover'))) {
    candidates = indexed(objects).filter(({ object }) => object.type === 'defending_point')
  }
  if (candidates.length === 0) {
    candidates = indexed(objects).filter(({ object }) =>
      object.type === 'bombing_point' || object.type === 'defending_point',
    )
  }

  const nearest = nearestToPlayer(snapshot, candidates)
  if (!nearest) return undefined
  const target = targetFromMapObject(nearest.object, nearest.index, snapshot.map.generation)
  if (!target) return undefined
  return {
    ...target,
    key: `objective-${objective.text}-${target.key}`,
    name: objective.text,
    basis: 'mission objective',
  }
}

export function nearestFriendlyAirfield(snapshot: Snapshot): SelectedTarget | undefined {
  const candidates = indexed(snapshot.map.objects).filter(({ object }) =>
    object.type === 'airfield' && friendly(object),
  )
  const nearest = nearestToPlayer(snapshot, candidates)
  if (!nearest) return undefined
  const target = targetFromMapObject(nearest.object, nearest.index, snapshot.map.generation)
  return target ? { ...target, name: 'Nearest friendly airfield' } : undefined
}

function targetableMapObject(object: MapObject) {
  if (object.icon === 'Player') return false
  return object.type === 'airfield' ||
    object.type === 'aircraft' ||
    object.type === 'bombing_point' ||
    object.type === 'defending_point' ||
    object.type === 'ground_model'
}

function distanceToObject(point: Point, object: MapObject): number | undefined {
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

function objectLabel(object: MapObject): string {
  switch (object.type) {
    case 'airfield': return friendly(object) ? 'Friendly airfield' : 'Enemy airfield'
    case 'bombing_point': return 'Strike point'
    case 'defending_point': return 'Defending point'
    case 'respawn_base_fighter': return 'Fighter ingress'
    case 'respawn_base_bomber': return 'Bomber ingress'
    case 'point_of_interest': return 'In-game target point'
    case 'aircraft': return object.icon ? `${object.icon} aircraft` : 'Aircraft'
    case 'ground_model':
      if (object.icon === 'SAM') return 'SAM position'
      if (object.icon === 'SPAA' || object.icon === 'Airdefence') return 'Air-defense position'
      return object.icon ? object.icon.replaceAll('_', ' ') : 'Ground unit'
    default: return object.icon?.replaceAll('_', ' ') ?? object.type.replaceAll('_', ' ')
  }
}

function nearestToPlayer(
  snapshot: Snapshot,
  candidates: Array<{ object: MapObject; index: number }>,
) {
  const player = snapshot.map.objects.find(
    (object) => object.icon === 'Player' && finite(object.x) && finite(object.y),
  )
  if (!player || !finite(player.x) || !finite(player.y)) return candidates[0]
  const playerX = player.x
  const playerY = player.y
  const dimensions = mapDimensions(snapshot)
  if (!dimensions) return candidates[0]
  return candidates
    .filter(({ object }) => objectPosition(object) !== undefined)
    .sort((left, right) => {
      const a = objectPosition(left.object)!
      const b = objectPosition(right.object)!
      const playerPosition = { x: playerX, y: playerY }
      return scaledDistance(a, playerPosition, dimensions) -
        scaledDistance(b, playerPosition, dimensions)
    })[0]
}

function indexed(objects: MapObject[]) {
  return objects.map((object, index) => ({ object, index }))
}

function mapDimensions(snapshot: Snapshot) {
  const min = snapshot.map.mapMin
  const max = snapshot.map.mapMax
  if (!min || !max || min.length < 2 || max.length < 2) return undefined
  const width = max[0] - min[0]
  const height = max[1] - min[1]
  return finite(width) && finite(height) && width > 0 && height > 0
    ? { width, height }
    : undefined
}

function hostile(object: MapObject) {
  const color = object.color?.toLowerCase() ?? ''
  return color === '#fa0c00' || color === '#f00c00' || color === '#fa0c00'
}

function friendly(object: MapObject) {
  const color = object.color?.toLowerCase() ?? ''
  return color === '#174dff' || color === '#39d921'
}

function finite(value: number | undefined): value is number {
  return value !== undefined && Number.isFinite(value)
}
