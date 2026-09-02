import { useEffect, useMemo, useRef } from 'react'
import type { ChangeEvent, KeyboardEvent, MouseEvent } from 'react'
import {
  mapPointTarget,
  navigationToTarget,
  targetFromMapObject,
  type NavigationSolution,
  type SelectedTarget,
} from './navigation'
import { clientToMapPoint } from './map/geometry'
import { nearestSelectableMapObjectHit } from './map/hitTesting'
import { drawMapFrame } from './map/rendering/frame'
import { useMapImage } from './map/useMapImage'
import type { Snapshot } from './types'

interface TacticalMapProps {
  snapshot: Snapshot | null
  navigation?: NavigationSolution
  selectedTarget: SelectedTarget | null
  onSelectTarget: (target: SelectedTarget) => void
}

export function TacticalMap({
  snapshot,
  navigation,
  selectedTarget,
  onSelectTarget,
}: TacticalMapProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const targetPickerRef = useRef<HTMLSelectElement>(null)
  const mapImage = useMapImage(snapshot)
  const keyboardTargets = useMemo(() => (snapshot?.map.objects ?? [])
    .map((object, index) => ({
      index,
      target: targetFromMapObject(object, index, snapshot?.map.generation),
    }))
    .filter((entry): entry is { index: number; target: SelectedTarget } =>
      entry.target !== undefined,
    ), [snapshot])

  useEffect(() => {
    const canvas = canvasRef.current
    if (!canvas) return
    const context = canvas.getContext('2d')
    if (!context) return

    const draw = () => {
      const bounds = canvas.getBoundingClientRect()
      const ratio = Math.min(window.devicePixelRatio || 1, 2)
      const width = Math.max(1, Math.round(bounds.width * ratio))
      const height = Math.max(1, Math.round(bounds.height * ratio))
      if (canvas.width !== width || canvas.height !== height) {
        canvas.width = width
        canvas.height = height
      }

      const currentImage = mapImage && mapImage.generation === snapshot?.map.generation ? mapImage.image : null
      drawMapFrame(context, {
        width,
        height,
        image: currentImage,
        snapshot,
        navigation,
      })
    }

    const resize = new ResizeObserver(draw)
    resize.observe(canvas)
    draw()
    const finalExpiry = Math.max(
      0,
      ...(snapshot?.allyMarks.map((mark) => new Date(mark.expiresAt).getTime()) ?? []),
    )
    let redrawTimer: number | undefined
    if (finalExpiry > Date.now()) {
      redrawTimer = window.setInterval(() => {
        draw()
        if (Date.now() >= finalExpiry && redrawTimer !== undefined) {
          window.clearInterval(redrawTimer)
          redrawTimer = undefined
        }
      }, 250)
    }
    return () => {
      resize.disconnect()
      if (redrawTimer !== undefined) window.clearInterval(redrawTimer)
    }
  }, [mapImage, navigation, snapshot])

  const selectTarget = (event: MouseEvent<HTMLCanvasElement>) => {
    if (!snapshot?.map.valid) return
    const canvas = canvasRef.current
    if (!canvas) return
    const bounds = canvas.getBoundingClientRect()
    const mapSize = Math.min(bounds.width, bounds.height)
    const point = clientToMapPoint(
      { x: event.clientX, y: event.clientY },
      bounds,
    )
    if (!point) return
    const hitRadius = 26 / mapSize
    const hit = nearestSelectableMapObjectHit(snapshot.map.objects, point, hitRadius)

    const target = hit
      ? targetFromMapObject(hit.object, hit.index, snapshot.map.generation)
      : mapPointTarget(point.x, point.y, snapshot.map.generation)
    if (target) onSelectTarget(target)
  }

  const selectKeyboardTarget = (event: ChangeEvent<HTMLSelectElement>) => {
    const index = Number(event.currentTarget.value)
    const selection = keyboardTargets.find((entry) => entry.index === index)
    if (selection) onSelectTarget(selection.target)
    event.currentTarget.value = ''
  }

  const openKeyboardTargets = (event: KeyboardEvent<HTMLCanvasElement>) => {
    if (event.key !== 'Enter' && event.key !== ' ') return
    event.preventDefault()
    targetPickerRef.current?.focus()
  }

  const selectedNavigation = navigation ??
    navigationToTarget(snapshot, selectedTarget)
  const label = selectedNavigation
    ? `War Thunder tactical map. Active destination: ${selectedNavigation.name}`
    : 'War Thunder tactical map. Click any object or point to select a destination.'

  return (
    <>
      <canvas
        ref={canvasRef}
        aria-label={`${label} Press Enter to focus the target list.`}
        onClick={selectTarget}
        onKeyDown={openKeyboardTargets}
        tabIndex={0}
      />
      <select
        ref={targetPickerRef}
        aria-label="Select a tactical map target"
        className="map-target-picker"
        defaultValue=""
        onChange={selectKeyboardTarget}
      >
        <option value="">Select map target</option>
        {keyboardTargets.map(({ index, target }) => (
          <option key={target.key} value={index}>{target.name}</option>
        ))}
      </select>
    </>
  )
}
