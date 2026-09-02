import { describe, expect, it } from 'vitest'
import { navigationSnapshot } from './navigation/testFixtures'
import type { Snapshot } from './types'
import { estimateCalibratedFuel } from './useFuelEstimate'

describe('estimateCalibratedFuel', () => {
  it('suspends a military-power calibration while afterburner is active', () => {
    const snapshot = fuelSnapshot(110)

    expect(estimateCalibratedFuel(snapshot, 1)).toMatchObject({
      state: 'unavailable',
      label: 'Afterburner outside calibrated fuel model',
    })
  })

  it('calculates direct-return fuel in the calibrated regime', () => {
    const snapshot = fuelSnapshot(100)

    expect(estimateCalibratedFuel(snapshot, 1)).toMatchObject({
      state: 'safe',
      bingoFuelKg: 72,
      marginKg: 928,
    })
  })
})

function fuelSnapshot(throttlePercent: number): Snapshot {
  return {
    ...navigationSnapshot([
      { type: 'aircraft', icon: 'Player', x: 0.1, y: 0.5 },
      {
        type: 'airfield',
        color: '#174DFF',
        sx: 0.2,
        sy: 0.4,
        ex: 0.2,
        ey: 0.6,
      },
    ], {
      mapMin: [0, 0],
      mapMax: [100_000, 100_000],
      tasKmh: 500,
    }),
    systems: {
      status: 'Nominal',
      severity: 'good',
      warnings: [],
      engines: [{
        index: 1,
        throttlePercent,
        status: 'running',
        running: true,
      }],
      fuelKg: 1000,
    },
  }
}
