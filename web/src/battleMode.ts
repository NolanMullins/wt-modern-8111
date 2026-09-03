import { useEffect, useState } from 'react'
import type { Snapshot } from './types'

export type BattleMode = 'air' | 'ground'
export type VehicleMode = 'air' | 'ground'

interface BattleModeState {
  mode: BattleMode
  generation?: number
}

export function battleModeForSnapshot(snapshot: Snapshot | null): BattleMode {
  return vehicleModeForSnapshot(snapshot) === 'ground' ||
      snapshot?.map?.hudType === 1 ||
      (snapshot?.map?.counts.groundSpawn ?? 0) > 0
    ? 'ground'
    : 'air'
}

export function retainBattleMode(
  previous: BattleModeState,
  snapshot: Snapshot | null,
): BattleModeState {
  if (!snapshot) return previous
  const detected = battleModeForSnapshot(snapshot)
  const generation = snapshot.map?.generation
  const generationChanged = generation !== undefined &&
    previous.generation !== undefined &&
    generation !== previous.generation
  const sessionEnded = !snapshot.map?.valid &&
    snapshot.mission.status !== 'running' &&
    (snapshot.connection.state === 'hangar' || snapshot.connection.state === 'waiting-for-game')
  if (generationChanged || sessionEnded) return { mode: detected, generation }
  const next: BattleModeState = {
    mode: previous.mode === 'ground' || detected === 'ground' ? 'ground' : 'air',
    generation: generation ?? previous.generation,
  }
  return next.mode === previous.mode && next.generation === previous.generation
    ? previous
    : next
}

export function useBattleMode(snapshot: Snapshot | null): BattleMode {
  const detected = battleModeForSnapshot(snapshot)
  const [retained, setRetained] = useState<BattleModeState>({ mode: 'air' })
  useEffect(() => {
    // oxlint-disable-next-line react/set-state-in-effect
    setRetained((previous) => retainBattleMode(previous, snapshot))
  }, [snapshot])
  return detected === 'ground' ? 'ground' : retained.mode
}

export function vehicleModeForSnapshot(snapshot: Snapshot | null): VehicleMode {
  const vehicleClass = snapshot?.vehicle.class?.trim().toLowerCase()
  const vehicleType = snapshot?.vehicle.type?.trim().toLowerCase()
  return vehicleClass === 'ground' ||
      vehicleClass === 'tank' ||
      vehicleType?.startsWith('tankmodels/')
    ? 'ground'
    : 'air'
}

export function displayVehicleName(vehicleType: string | undefined) {
  const name = vehicleType?.split('/').at(-1)?.trim()
  return name ? name.replaceAll('_', ' ').toUpperCase() : 'NO VEHICLE'
}
