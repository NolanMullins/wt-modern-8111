import type { MapObject, Snapshot } from '../types'

interface SnapshotOptions {
  generation?: number
  mapMin?: number[]
  mapMax?: number[]
  tasKmh?: number
}

export function navigationSnapshot(
  objects: MapObject[],
  {
    generation = 7,
    mapMin = [0, 0],
    mapMax = [100_000, 50_000],
    tasKmh = 500,
  }: SnapshotOptions = {},
): Snapshot {
  return {
    flight: { tasKmh },
    map: {
      valid: true,
      generation,
      mapMin,
      mapMax,
      objects,
    },
    mission: { objectives: [] },
  } as unknown as Snapshot
}
