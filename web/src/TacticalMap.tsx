import { useEffect, useRef } from 'react'
import type { MouseEvent } from 'react'
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
  const mapImage = useMapImage(snapshot)

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
    return () => resize.disconnect()
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

  const selectedNavigation = navigation ??
    navigationToTarget(snapshot, selectedTarget)
  const label = selectedNavigation
    ? `War Thunder tactical map. Active destination: ${selectedNavigation.name}`
    : 'War Thunder tactical map. Click any object or point to select a destination.'

  return <canvas ref={canvasRef} aria-label={label} onClick={selectTarget} />
}
