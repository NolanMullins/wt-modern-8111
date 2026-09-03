import { useEffect, useMemo, useState } from 'react'

export type PlayerTeam = 1 | 2
export type TeamPreference = 'auto' | '1' | '2'

export interface PlayerTeamDetection {
  team?: PlayerTeam
  status: 'detected' | 'unknown' | 'unavailable' | 'error'
  source?: string
  updatedAt?: string
  detail?: string
}

const preferenceKey = 'wt-modern-8111.heatmap-player-team'

const waitingDetection: PlayerTeamDetection = {
  status: 'unknown',
  detail: 'Waiting for the current battle team assignment',
}

export function usePlayerTeam(active: boolean, sessionKey?: number) {
  const [preference, setPreferenceState] = useState<TeamPreference>(() => {
    const stored = window.localStorage.getItem(preferenceKey)
    return stored === '1' || stored === '2' ? stored : 'auto'
  })
  const [detectionState, setDetectionState] = useState<{
    sessionKey?: number
    value: PlayerTeamDetection
  }>({
    value: waitingDetection,
  })
  const detection = active && detectionState.sessionKey === sessionKey
    ? detectionState.value
    : waitingDetection

  useEffect(() => {
    if (!active) return
    const controller = new AbortController()
    const load = () => {
      fetch('/api/v1/player-team', {
        cache: 'no-store',
        signal: controller.signal,
      })
        .then(async (response) => {
          if (!response.ok) throw new Error(`player team HTTP ${response.status}`)
          const value = await response.json() as PlayerTeamDetection
          if (value.team !== undefined && value.team !== 1 && value.team !== 2) {
            throw new Error(`invalid player team ${value.team}`)
          }
          setDetectionState({ sessionKey, value })
        })
        .catch((error: unknown) => {
          if (!(error instanceof DOMException && error.name === 'AbortError')) {
            setDetectionState({
              sessionKey,
              value: {
                status: 'error',
                detail: 'Could not read the current War Thunder team assignment',
              },
            })
          }
        })
    }
    load()
    const interval = window.setInterval(load, 2000)
    return () => {
      controller.abort()
      window.clearInterval(interval)
    }
  }, [active, sessionKey])

  const selectedTeam = useMemo<PlayerTeam | undefined>(() => {
    if (preference === '1') return 1
    if (preference === '2') return 2
    return detection.status === 'detected' ? detection.team : undefined
  }, [detection, preference])

  const setPreference = (next: TeamPreference) => {
    setPreferenceState(next)
    if (next === 'auto') {
      window.localStorage.removeItem(preferenceKey)
    } else {
      window.localStorage.setItem(preferenceKey, next)
    }
  }

  return {
    detection,
    preference,
    selectedTeam,
    enemyTeam: selectedTeam === undefined ? undefined : (3 - selectedTeam) as PlayerTeam,
    setPreference,
  }
}
