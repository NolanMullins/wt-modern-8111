import { describe, expect, it } from 'vitest'
import {
  battleModeForSnapshot,
  displayVehicleName,
  retainBattleMode,
  vehicleModeForSnapshot,
} from './battleMode'
import type { Snapshot } from './types'

describe('battle mode selection', () => {
  it.each(['ground', 'tank'])('selects ground mode for the %s army class', (vehicleClass) => {
    expect(battleModeForSnapshot({
      vehicle: { class: vehicleClass },
    } as Snapshot)).toBe('ground')
  })

  it('falls back to the tank model prefix when army is unavailable', () => {
    expect(battleModeForSnapshot({
      vehicle: { type: 'tankModels/ussr_t_80u' },
    } as Snapshot)).toBe('ground')
  })

  it('keeps Ground RB context while the player flies CAS', () => {
    const snapshot = {
      vehicle: { class: 'air', type: 'su_25' },
      map: { hudType: 1, counts: {} },
    } as Snapshot

    expect(battleModeForSnapshot(snapshot)).toBe('ground')
    expect(vehicleModeForSnapshot(snapshot)).toBe('air')
  })

  it('retains Ground RB through missing map signals until the map changes', () => {
    const tank = snapshot({
      vehicle: { class: 'tank' },
      map: { generation: 2, hudType: 1 },
    })
    const casDuringMapRefresh = snapshot({
      vehicle: { class: 'air' },
      map: { generation: 2 },
    })
    const nextAirMission = snapshot({
      vehicle: { class: 'air' },
      map: { generation: 3, hudType: 0 },
    })

    const groundState = retainBattleMode({ mode: 'air' }, tank)
    expect(retainBattleMode(groundState, casDuringMapRefresh).mode).toBe('ground')
    expect(retainBattleMode(groundState, nextAirMission).mode).toBe('air')
  })

  it('preserves air mode and formats internal vehicle paths', () => {
    expect(battleModeForSnapshot(null)).toBe('air')
    expect(displayVehicleName('tankModels/ussr_t_80u')).toBe('USSR T 80U')
  })
})

function snapshot({
  vehicle,
  map,
}: {
  vehicle: Snapshot['vehicle']
  map: Pick<Snapshot['map'], 'generation' | 'hudType'>
}): Snapshot {
  return {
    version: 1,
    sequence: 1,
    capturedAt: '2026-09-01T20:00:00Z',
    vehicle,
    flight: {},
    ground: {},
    systems: {
      status: 'Unavailable',
      severity: 'unknown',
      warnings: [],
      engines: [],
    },
    map: {
      valid: true,
      objects: [],
      counts: {
        total: 0,
        hostileAir: 0,
        ground: 0,
        airDefense: 0,
        strikePoint: 0,
        airfield: 0,
      },
      ...map,
    },
    mission: { status: 'running', objectives: [] },
    connection: { state: 'live', mode: 'live', sources: {} },
    feed: [],
    pilot: { confirmed: false },
    allyMarks: [],
  }
}
