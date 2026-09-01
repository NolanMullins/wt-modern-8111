import type { MapObject, Snapshot } from './types'

export interface SelectedTarget {
  key: string
  name: string
  x: number
  y: number
  generation?: number
  basis: 'map selection' | 'mission objective' | 'game target'
  object?: {
    index: number
    type: string
    icon?: string
    color?: string
  }
}

export type NavigationSolution = NonNullable<Snapshot['navigation']>
const targetInferenceRadius = 0.03

export function navigationToTarget(
  snapshot: Snapshot | null,
  target: SelectedTarget | null,
): NavigationSolution | undefined {
  if (!snapshot || !target || !snapshot.map.valid) return undefined
  if (
    target.generation !== undefined &&
    snapshot.map.generation !== undefined &&
    target.generation !== snapshot.map.generation
  ) return undefined

  const player = snapshot.map.objects.find(
    (object) => object.icon === 'Player' && finite(object.x) && finite(object.y),
  )
  if (!player || !finite(player.x) || !finite(player.y)) return undefined
  const bounds = mapDimensions(snapshot)
  if (!bounds) return undefined

  const targetPosition = resolveTargetPosition(snapshot, target)
  if (!targetPosition) return undefined
  const dx = (targetPosition.x - player.x) * bounds.width
  const dy = (targetPosition.y - player.y) * bounds.height
  const rangeKm = Math.hypot(dx, dy) / 1000
  const bearingDeg = (Math.atan2(dx, -dy) * 180 / Math.PI + 360) % 360
  const tas = snapshot.flight.tasKmh
  const etaSeconds = finite(tas) && tas > 0 ? rangeKm / tas * 3600 : undefined
  return {
    name: target.name,
    bearingDeg,
    rangeKm,
    etaSeconds,
    targetX: targetPosition.x,
    targetY: targetPosition.y,
    basis: target.basis,
  }
}

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

function targetableMapObject(object: MapObject) {
  if (object.icon === 'Player') return false
  return object.type === 'airfield' ||
    object.type === 'aircraft' ||
    object.type === 'bombing_point' ||
    object.type === 'defending_point' ||
    object.type === 'ground_model'
}

function distanceToObject(
  point: { x: number; y: number },
  object: MapObject,
): number | undefined {
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
      : Math.max(0, Math.min(1, ((point.x - object.sx) * dx + (point.y - object.sy) * dy) / lengthSquared))
    return Math.hypot(
      point.x - (object.sx + interpolation * dx),
      point.y - (object.sy + interpolation * dy),
    )
  }
  const position = objectPosition(object)
  return position ? Math.hypot(point.x - position.x, point.y - position.y) : undefined
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

export function objectPosition(object: MapObject): { x: number; y: number } | undefined {
  if (object.type === 'airfield') {
    if (finite(object.sx) && finite(object.sy) && finite(object.ex) && finite(object.ey)) {
      return { x: (object.sx + object.ex) / 2, y: (object.sy + object.ey) / 2 }
    }
  }
  if (finite(object.x) && finite(object.y)) return { x: object.x, y: object.y }
  return undefined
}

// Resolve against the nearest matching object from the previous live position.
// This follows moving units while surviving insertions and removals elsewhere
// in War Thunder's indexless map-object array.
export function resolveTargetPosition(
  snapshot: Snapshot,
  target: SelectedTarget,
): { x: number; y: number } | undefined {
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
  const nearest = indexed(snapshot.map.objects)
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
  return nearest?.distance <= 0.025 ? nearest : undefined
}

function sameObjectKind(object: MapObject, identity: NonNullable<SelectedTarget['object']>) {
  return object.type === identity.type &&
    object.icon === identity.icon &&
    object.color === identity.color
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
  return candidates
    .filter(({ object }) => objectPosition(object) !== undefined)
    .sort((left, right) => {
      const a = objectPosition(left.object)!
      const b = objectPosition(right.object)!
      return Math.hypot(a.x - playerX, a.y - playerY) -
        Math.hypot(b.x - playerX, b.y - playerY)
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
