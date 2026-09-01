import type { Snapshot } from '../types'
import { resolveTargetPosition } from './tracking'
import type { NavigationSolution, SelectedTarget } from './types'

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

function finite(value: number | undefined): value is number {
  return value !== undefined && Number.isFinite(value)
}
