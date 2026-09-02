import { useEffect, useMemo, useRef, useState } from 'react'
import {
  inGameTarget,
  navigationToTarget,
  reconcileSelectedTarget,
  type SelectedTarget,
} from '../../navigation'
import type { Snapshot } from '../../types'

export function useSelectedNavigation(snapshot: Snapshot | null) {
  const [selectedTarget, setSelectedTarget] = useState<SelectedTarget | null>(null)
  const lastGameTargetKey = useRef<string | null>(null)
  const radioRTBWasActive = useRef(false)
  const gameTarget = useMemo(() => inGameTarget(snapshot), [snapshot])
  const gameTargetKey = gameTarget?.key ?? null

  useEffect(() => {
    if (!snapshot) return
    const previousGameTargetKey = lastGameTargetKey.current
    const radioRTBActive = snapshot.navigation != null
    const radioRTBActivated = radioRTBActive && !radioRTBWasActive.current
    lastGameTargetKey.current = gameTargetKey
    radioRTBWasActive.current = radioRTBActive
    // oxlint-disable-next-line react/set-state-in-effect
    setSelectedTarget((current) => {
      if (radioRTBActivated) return null
      if (gameTarget && gameTargetKey !== previousGameTargetKey) return gameTarget
      if (
        !gameTarget &&
        current?.basis === 'game target' &&
        current.object?.type === 'point_of_interest'
      ) {
        return { ...current, object: undefined }
      }
      return reconcileSelectedTarget(snapshot, current)
    })
  }, [gameTarget, gameTargetKey, snapshot])

  const effectiveTarget = selectedTarget &&
    (selectedTarget.generation === undefined ||
      snapshot?.map.generation === undefined ||
      selectedTarget.generation === snapshot.map.generation)
    ? selectedTarget
    : null
  const selectedNavigation = useMemo(
    () => navigationToTarget(snapshot, effectiveTarget),
    [effectiveTarget, snapshot],
  )
  const radioRTB = selectedNavigation === undefined && snapshot?.navigation != null

  return {
    navigation: selectedNavigation ?? snapshot?.navigation,
    selectedTarget: effectiveTarget,
    selectTarget: setSelectedTarget,
    clearTarget: () => setSelectedTarget(null),
    radioRTB,
  }
}
