import { useEffect, useRef, useState } from 'react'
import { navigationToTarget, nearestFriendlyAirfield } from './navigation'
import type { Snapshot } from './types'

const calibrationSeconds = 8

interface FuelSample {
  at: number
  fuelKg: number
}

export interface FuelEstimate {
  state: 'building' | 'safe' | 'caution' | 'bingo' | 'emergency' | 'unavailable' | 'unlimited'
  label: string
  burnKgPerHour?: number
  enduranceSeconds?: number
  bingoFuelKg?: number
  marginKg?: number
  baseRangeKm?: number
  baseETASeconds?: number
  sampleSeconds?: number
}

export function useFuelEstimate(snapshot: Snapshot | null): FuelEstimate {
  const samples = useRef<FuelSample[]>([])
  const calibration = useRef<{ vehicle?: string; burnKgPerSecond?: number }>({})
  const lastCalculation = useRef(0)
  const [estimate, setEstimate] = useState<FuelEstimate>({
    state: 'building',
    label: 'Hold 100% throttle to calibrate',
  })

  useEffect(() => {
    const fuelKg = snapshot?.systems.fuelKg
    if (!snapshot || fuelKg === undefined || !Number.isFinite(fuelKg)) {
      samples.current = []
      // This hook intentionally maintains a rolling sample window from the
      // external SSE stream; publishing its derived state belongs in effect.
      // oxlint-disable-next-line react/set-state-in-effect
      setEstimate({ state: 'unavailable', label: 'Fuel estimate unavailable' })
      return
    }

    const at = new Date(snapshot.capturedAt).getTime()
    if (calibration.current.vehicle !== snapshot.vehicle.type) {
      calibration.current = { vehicle: snapshot.vehicle.type }
      samples.current = []
    }

    if (at - lastCalculation.current < 1000) return
    lastCalculation.current = at
    const burnKgPerSecond = calibration.current.burnKgPerSecond
    if (burnKgPerSecond !== undefined) {
      // oxlint-disable-next-line react/set-state-in-effect
      setEstimate(calculateFuelEstimate(snapshot, burnKgPerSecond))
      return
    }

    const throttleState = calibrationThrottleState(snapshot)
    if (throttleState !== 'military') {
      samples.current = []
      // oxlint-disable-next-line react/set-state-in-effect
      setEstimate({
        state: 'building',
        label: throttleState === 'afterburner'
          ? 'Afterburner excluded · set 100%'
          : 'Hold 100% throttle to calibrate',
      })
      return
    }

    const previous = samples.current.at(-1)
    if (previous && fuelKg > previous.fuelKg + 1) samples.current = []
    samples.current.push({ at, fuelKg })
    const first = samples.current[0]
    const last = samples.current.at(-1)
    const sampleSeconds = first && last ? (last.at - first.at) / 1000 : 0
    const fuelUsed = first && last ? first.fuelKg - last.fuelKg : 0
    if (sampleSeconds >= calibrationSeconds && fuelUsed <= 0.05) {
      // Full throttle for the whole window with no measurable burn means the
      // mission is running unlimited fuel, so no bingo state can ever apply.
      // oxlint-disable-next-line react/set-state-in-effect
      setEstimate({ state: 'unlimited', label: 'Unlimited fuel enabled' })
      return
    }
    if (sampleSeconds < calibrationSeconds) {
      // oxlint-disable-next-line react/set-state-in-effect
      setEstimate({
        state: 'building',
        label: `Calibrating at 100% · ${Math.min(calibrationSeconds, Math.round(sampleSeconds))}/${calibrationSeconds}s`,
        sampleSeconds,
      })
      return
    }

    calibration.current.burnKgPerSecond = fuelUsed / sampleSeconds
    samples.current = []
    // oxlint-disable-next-line react/set-state-in-effect
    setEstimate(calculateFuelEstimate(snapshot, calibration.current.burnKgPerSecond))
  }, [snapshot])

  return estimate
}

function calculateFuelEstimate(snapshot: Snapshot, burnKgPerSecond: number): FuelEstimate {
  if (snapshot.systems.status === 'Aircraft lost') {
    return { state: 'unavailable', label: 'Aircraft lost' }
  }
  const fuelKg = snapshot.systems.fuelKg
  if (fuelKg === undefined || !Number.isFinite(fuelKg) || burnKgPerSecond <= 0) {
    return { state: 'unavailable', label: 'Fuel estimate unavailable' }
  }
  const burnKgPerHour = burnKgPerSecond * 3600
  const enduranceSeconds = fuelKg / burnKgPerSecond
  const airfield = nearestFriendlyAirfield(snapshot)
  const baseNavigation = navigationToTarget(snapshot, airfield ?? null)
  if (baseNavigation?.etaSeconds === undefined) {
    return {
      state: 'unavailable',
      label: 'No friendly airfield solution',
      burnKgPerHour,
      enduranceSeconds,
    }
  }

  const tripFuelKg = baseNavigation.etaSeconds * burnKgPerSecond
  const bingoFuelKg = tripFuelKg
  const marginKg = fuelKg - bingoFuelKg
  let state: FuelEstimate['state'] = 'safe'
  let label = 'Fuel safe'
  if (marginKg <= -burnKgPerSecond * 60) {
    state = 'emergency'
    label = 'Below recovery fuel'
  } else if (marginKg <= 0) {
    state = 'bingo'
    label = 'Bingo fuel'
  } else if (marginKg <= burnKgPerSecond * 120) {
    state = 'caution'
    label = 'Approaching bingo'
  }

  return {
    state,
    label,
    burnKgPerHour,
    enduranceSeconds,
    bingoFuelKg,
    marginKg,
    baseRangeKm: baseNavigation.rangeKm,
    baseETASeconds: baseNavigation.etaSeconds,
  }
}

function calibrationThrottleState(snapshot: Snapshot): 'military' | 'afterburner' | 'other' {
  const throttles = snapshot.systems.engines
    .map((engine) => engine.throttlePercent)
    .filter((throttle): throttle is number => throttle !== undefined && Number.isFinite(throttle))
  if (throttles.some((throttle) => throttle > 101)) return 'afterburner'
  if (throttles.length > 0 && throttles.every((throttle) => throttle >= 99 && throttle <= 101)) {
    return 'military'
  }
  return 'other'
}
