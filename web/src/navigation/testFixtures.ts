import type { MapObject, Snapshot } from '../types'

interface SnapshotOptions {
  generation?: number
  mapMin?: number[]
  mapMax?: number[]
  tasKmh?: number
  groundSpeedKmh?: number
  vehicleClass?: string
  hudType?: number
}

export function navigationSnapshot(
  objects: MapObject[],
  {
    generation = 7,
    mapMin = [0, 0],
    mapMax = [100_000, 50_000],
    tasKmh = 500,
    groundSpeedKmh,
    vehicleClass = 'air',
    hudType,
  }: SnapshotOptions = {},
): Snapshot {
  return {
    vehicle: { class: vehicleClass },
    flight: { tasKmh },
    ground: { speedKmh: groundSpeedKmh },
    map: {
      valid: true,
      generation,
      hudType,
      mapMin,
      mapMax,
      objects,
    },
    mission: { objectives: [] },
  } as unknown as Snapshot
}
