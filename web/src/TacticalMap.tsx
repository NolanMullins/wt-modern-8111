import { useEffect, useMemo, useRef, useState } from 'react'
import type {
  ChangeEvent,
  KeyboardEvent,
  PointerEvent,
  WheelEvent,
} from 'react'
import {
  mapPointTarget,
  navigationToTarget,
  targetFromMapObject,
  type NavigationSolution,
  type SelectedTarget,
} from './navigation'
import {
  clientToCameraMapPoint,
  defaultMapCamera,
  panCamera,
  zoomCameraAt,
  type MapCamera,
} from './map/camera'
import { clientToMapPoint, type Point } from './map/geometry'
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

interface DragState {
  pointerId: number
  start: Point
  last: Point
  moved: boolean
}

export function TacticalMap({
  snapshot,
  navigation,
  selectedTarget,
  onSelectTarget,
}: TacticalMapProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const targetPickerRef = useRef<HTMLSelectElement>(null)
  const dragRef = useRef<DragState | undefined>(undefined)
  const cameraRef = useRef<MapCamera>(defaultMapCamera)
  const drawRef = useRef<() => void>(() => {})
  const drawFrameRef = useRef<number | undefined>(undefined)
  const [dragging, setDragging] = useState(false)
  const mapImage = useMapImage(snapshot)
  const keyboardTargets = useMemo(() => (snapshot?.map.objects ?? [])
    .map((object, index) => ({
      index,
      target: targetFromMapObject(object, index, snapshot?.map.generation),
    }))
    .filter((entry): entry is { index: number; target: SelectedTarget } =>
      entry.target !== undefined,
    ), [snapshot])

  function requestMapDraw() {
    if (drawFrameRef.current !== undefined) return
    drawFrameRef.current = window.requestAnimationFrame(() => {
      drawFrameRef.current = undefined
      drawRef.current()
    })
  }

  useEffect(() => {
    cameraRef.current = defaultMapCamera
    requestMapDraw()
  }, [snapshot?.map.generation])

  useEffect(() => () => {
    if (drawFrameRef.current !== undefined) {
      window.cancelAnimationFrame(drawFrameRef.current)
    }
  }, [])

  useEffect(() => {
    const canvas = canvasRef.current
    if (!canvas) return
    const preventPageScroll = (event: globalThis.WheelEvent) => event.preventDefault()
    canvas.addEventListener('wheel', preventPageScroll, { passive: false })
    return () => canvas.removeEventListener('wheel', preventPageScroll)
  }, [])

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
        camera: cameraRef.current,
      })
    }
    drawRef.current = draw

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
      drawRef.current = () => {}
      resize.disconnect()
      if (redrawTimer !== undefined) window.clearInterval(redrawTimer)
    }
  }, [mapImage, navigation, snapshot])

  const selectTargetAt = (client: Point) => {
    if (!snapshot?.map.valid) return
    const canvas = canvasRef.current
    if (!canvas) return
    const bounds = canvas.getBoundingClientRect()
    const mapSize = Math.min(bounds.width, bounds.height)
    const camera = cameraRef.current
    const point = clientToCameraMapPoint(client, bounds, camera)
    if (!point) return
    const hitRadius = 26 / (mapSize * camera.zoom)
    const hit = nearestSelectableMapObjectHit(snapshot.map.objects, point, hitRadius)

    const target = hit
      ? targetFromMapObject(hit.object, hit.index, snapshot.map.generation)
      : mapPointTarget(point.x, point.y, snapshot.map.generation)
    if (target) onSelectTarget(target)
  }

  const startPan = (event: PointerEvent<HTMLCanvasElement>) => {
    if (event.button !== 0 || !snapshot?.map.valid) return
    if (dragRef.current) {
      dragRef.current.moved = true
      return
    }
    event.currentTarget.setPointerCapture(event.pointerId)
    const point = { x: event.clientX, y: event.clientY }
    dragRef.current = {
      pointerId: event.pointerId,
      start: point,
      last: point,
      moved: false,
    }
  }

  const movePan = (event: PointerEvent<HTMLCanvasElement>) => {
    const drag = dragRef.current
    if (!drag || drag.pointerId !== event.pointerId) return
    const current = { x: event.clientX, y: event.clientY }
    const movement = {
      x: current.x - drag.last.x,
      y: current.y - drag.last.y,
    }
    if (!drag.moved && Math.hypot(current.x - drag.start.x, current.y - drag.start.y) >= 4) {
      drag.moved = true
      setDragging(true)
    }
    drag.last = current
    if (drag.moved) {
      const bounds = event.currentTarget.getBoundingClientRect()
      cameraRef.current = panCamera(
        cameraRef.current,
        movement,
        Math.min(bounds.width, bounds.height),
      )
      requestMapDraw()
    }
  }

  const finishPan = (event: PointerEvent<HTMLCanvasElement>) => {
    const drag = dragRef.current
    if (!drag || drag.pointerId !== event.pointerId) return
    if (event.currentTarget.hasPointerCapture(event.pointerId)) {
      event.currentTarget.releasePointerCapture(event.pointerId)
    }
    dragRef.current = undefined
    setDragging(false)
    if (!drag.moved) {
      selectTargetAt({ x: event.clientX, y: event.clientY })
    }
  }

  const cancelPan = (event: PointerEvent<HTMLCanvasElement>) => {
    if (dragRef.current?.pointerId !== event.pointerId) return
    dragRef.current = undefined
    setDragging(false)
  }

  const zoomMap = (event: WheelEvent<HTMLCanvasElement>) => {
    if (!snapshot?.map.valid) return
    const bounds = event.currentTarget.getBoundingClientRect()
    const client = { x: event.clientX, y: event.clientY }
    const viewportPoint = clientToMapPoint(client, bounds)
    if (!viewportPoint) return
    const unit = event.deltaMode === 1
      ? 16
      : event.deltaMode === 2
        ? Math.min(bounds.width, bounds.height)
        : 1
    const factor = Math.exp(-event.deltaY * unit * 0.0015)
    const camera = cameraRef.current
    const anchor = clientToCameraMapPoint(client, bounds, camera)
    if (!anchor) return
    cameraRef.current = zoomCameraAt(anchor, viewportPoint, camera.zoom * factor)
    requestMapDraw()
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
        aria-label={`${label} Drag to pan, use the mouse wheel to zoom, or press Enter to focus the target list.`}
        data-dragging={dragging}
        onKeyDown={openKeyboardTargets}
        onPointerCancel={cancelPan}
        onPointerDown={startPan}
        onPointerMove={movePan}
        onPointerUp={finishPan}
        onWheel={zoomMap}
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
