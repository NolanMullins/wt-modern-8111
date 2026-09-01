import { useEffect, useRef, useState } from 'react'
import type { MouseEvent } from 'react'
import {
  mapPointTarget,
  navigationToTarget,
  objectPosition,
  targetFromMapObject,
  type NavigationSolution,
  type SelectedTarget,
} from './navigation'
import type { MapObject, Snapshot } from './types'

interface TacticalMapProps {
  snapshot: Snapshot | null
  navigation?: NavigationSolution
  selectedTarget: SelectedTarget | null
  onSelectTarget: (target: SelectedTarget) => void
}

interface MapRect {
  x: number
  y: number
  size: number
}

interface LoadedMap {
  generation: number
  image: HTMLImageElement
}

export function TacticalMap({
  snapshot,
  navigation,
  selectedTarget,
  onSelectTarget,
}: TacticalMapProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const [mapImage, setMapImage] = useState<LoadedMap | null>(null)

  useEffect(() => {
    if (snapshot?.connection.mode !== 'live' || !snapshot.map.valid || snapshot.map.generation === undefined) {
      return
    }
    const generation = snapshot.map.generation
    let cancelled = false
    let retry: number | undefined
    const load = () => {
      const image = new Image()
      image.onload = () => {
        if (!cancelled) setMapImage({ generation, image })
      }
      image.onerror = () => {
        if (!cancelled) retry = window.setTimeout(load, 1000)
      }
      image.src = `/api/v1/map/${generation}?attempt=${Date.now()}`
    }
    load()
    return () => {
      cancelled = true
      if (retry !== undefined) window.clearTimeout(retry)
    }
  }, [snapshot?.connection.mode, snapshot?.map.generation, snapshot?.map.valid])

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

      const rect = squareRect(width, height)
      context.clearRect(0, 0, width, height)
      context.fillStyle = '#111519'
      context.fillRect(0, 0, width, height)
      context.save()
      context.beginPath()
      context.rect(rect.x, rect.y, rect.size, rect.size)
      context.clip()
      const currentImage = mapImage && mapImage.generation === snapshot?.map.generation ? mapImage.image : null
      drawBackground(context, rect, currentImage)
      if (snapshot) {
        drawGrid(context, rect, snapshot)
        drawObjects(context, rect, snapshot, navigation)
      }
      context.restore()
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
    const mapLeft = (bounds.width - mapSize) / 2
    const mapTop = (bounds.height - mapSize) / 2
    const clickX = event.clientX - bounds.left
    const clickY = event.clientY - bounds.top
    const x = (clickX - mapLeft) / mapSize
    const y = (clickY - mapTop) / mapSize
    if (x < 0 || x > 1 || y < 0 || y > 1) return

    const hitRadius = 26 / mapSize
    const hit = snapshot.map.objects
      .map((object, index) => ({
        object,
        index,
        position: objectPosition(object),
      }))
      .filter((candidate) => candidate.position && candidate.object.icon !== 'Player')
      .map((candidate) => ({
        ...candidate,
        distance: Math.hypot(candidate.position!.x - x, candidate.position!.y - y),
      }))
      .filter((candidate) => candidate.distance <= hitRadius)
      .sort((left, right) => left.distance - right.distance)[0]

    const target = hit
      ? targetFromMapObject(hit.object, hit.index, snapshot.map.generation)
      : mapPointTarget(x, y, snapshot.map.generation)
    if (target) onSelectTarget(target)
  }

  const selectedNavigation = navigation ??
    navigationToTarget(snapshot, selectedTarget)
  const label = selectedNavigation
    ? `War Thunder tactical map. Active destination: ${selectedNavigation.name}`
    : 'War Thunder tactical map. Click any object or point to select a destination.'

  return <canvas ref={canvasRef} aria-label={label} onClick={selectTarget} />
}

function squareRect(width: number, height: number): MapRect {
  const size = Math.min(width, height)
  return {
    x: Math.round((width - size) / 2),
    y: Math.round((height - size) / 2),
    size,
  }
}

function drawBackground(context: CanvasRenderingContext2D, rect: MapRect, image: HTMLImageElement | null) {
  if (image?.complete && image.naturalWidth) {
    context.drawImage(image, rect.x, rect.y, rect.size, rect.size)
    return
  }
  context.fillStyle = '#404040'
  context.fillRect(rect.x, rect.y, rect.size, rect.size)
  context.save()
  context.globalAlpha = 0.42
  context.fillStyle = '#808080'
  context.textAlign = 'center'
  context.textBaseline = 'middle'
  context.font = `900 ${rect.size * 0.55}px Arial`
  context.fillText('?', rect.x + rect.size / 2, rect.y + rect.size * 0.51)
  context.restore()
}

function drawGrid(context: CanvasRenderingContext2D, rect: MapRect, snapshot: Snapshot) {
  const { mapMin, mapMax, gridSteps } = snapshot.map
  if (!mapMin || !mapMax || !gridSteps || mapMin.length < 2 || mapMax.length < 2 || gridSteps.length < 2) return
  const width = mapMax[0] - mapMin[0]
  const height = mapMax[1] - mapMin[1]
  if (![width, height, gridSteps[0], gridSteps[1]].every(Number.isFinite) ||
    width <= 0 || height <= 0 || gridSteps[0] <= 0 || gridSteps[1] <= 0) return
  const columns = Math.min(100, Math.max(1, Math.ceil(width / gridSteps[0])))
  const rows = Math.min(100, Math.max(1, Math.ceil(height / gridSteps[1])))
  const ratio = Math.min(window.devicePixelRatio || 1, 2)

  context.save()
  context.strokeStyle = 'rgba(255,255,255,.16)'
  context.fillStyle = 'rgba(238,240,238,.9)'
  context.shadowColor = 'rgba(0,0,0,.95)'
  context.shadowBlur = 3 * ratio
  context.lineWidth = ratio
  context.font = `700 ${Math.max(11 * ratio, rect.size * 0.018)}px Arial`
  for (let row = 0; row <= rows; row += 1) {
    const y = rect.y + (row / rows) * rect.size
    context.beginPath()
    context.moveTo(rect.x, y)
    context.lineTo(rect.x + rect.size, y)
    context.stroke()
    if (row < rows) {
      context.textAlign = 'left'
      context.textBaseline = 'middle'
      context.fillText(rowLabel(row), rect.x + 5 * ratio, y + rect.size / rows / 2)
    }
  }
  for (let column = 0; column <= columns; column += 1) {
    const x = rect.x + (column / columns) * rect.size
    context.beginPath()
    context.moveTo(x, rect.y)
    context.lineTo(x, rect.y + rect.size)
    context.stroke()
    if (column < columns) {
      context.textAlign = 'center'
      context.textBaseline = 'top'
      context.fillText(String(column + 1), x + rect.size / columns / 2, rect.y + 5 * ratio)
    }
  }
  context.restore()
}

function drawObjects(
  context: CanvasRenderingContext2D,
  rect: MapRect,
  snapshot: Snapshot,
  navigation?: NavigationSolution,
) {
  const objects = snapshot.map.objects ?? []
  const ground = objects.filter((object) => object.type === 'ground_model')
  objects
    .filter((object) => object.type !== 'ground_model' && object.type !== 'aircraft')
    .forEach((object) => drawObject(context, rect, object))
  drawGroundClusters(context, rect, ground)
  objects
    .filter((object) => object.type === 'aircraft')
    .forEach((object) => drawObject(context, rect, object))
  drawNavigation(context, rect, snapshot, navigation)
  drawAllyMarks(context, rect, snapshot)
}

const allyMarkLabels: Record<string, string> = {
  guide: 'GUIDE ON ME',
  attention: 'ATTENTION',
  cover: 'COVER ME',
  help: 'NEEDS HELP',
}
const allyMarkFadeMilliseconds = 5_000

// drawAllyMarks renders teammate radio callouts. Only located marks appear on
// the map; unlocated callouts are surfaced in the feed instead.
function drawAllyMarks(context: CanvasRenderingContext2D, rect: MapRect, snapshot: Snapshot) {
  const marks = (snapshot.allyMarks ?? []).filter(
    (mark) =>
      mark.located &&
      typeof mark.x === 'number' &&
      typeof mark.y === 'number' &&
      new Date(mark.expiresAt).getTime() > Date.now(),
  )
  if (marks.length === 0) return
  const ratio = Math.min(window.devicePixelRatio || 1, 2)
  const now = Date.now()

  marks.forEach((mark) => {
    const x = rect.x + (mark.x as number) * rect.size
    const y = rect.y + (mark.y as number) * rect.size
    const age = (now - new Date(mark.createdAt).getTime()) / 1000
    const remaining = new Date(mark.expiresAt).getTime() - now
    const opacity = Math.min(1, Math.max(0, remaining / allyMarkFadeMilliseconds))
    // Pulse for the first few seconds so a fresh call draws the eye.
    const pulse = age < 6 ? 1 + 0.25 * Math.sin(age * Math.PI * 2) : 1
    const radius = 13 * ratio * pulse

    context.save()
    context.globalAlpha = opacity
    context.strokeStyle = '#39d921'
    context.fillStyle = 'rgba(57, 217, 33, 0.16)'
    context.lineWidth = 2 * ratio

    context.beginPath()
    context.arc(x, y, radius, 0, Math.PI * 2)
    context.fill()
    context.stroke()

    context.beginPath()
    context.moveTo(x - radius - 5 * ratio, y)
    context.lineTo(x + radius + 5 * ratio, y)
    context.moveTo(x, y - radius - 5 * ratio)
    context.lineTo(x, y + radius + 5 * ratio)
    context.stroke()

    const label = allyMarkLabels[mark.kind] ?? mark.kind.toUpperCase()
    context.font = `${10 * ratio}px "Inter", system-ui, sans-serif`
    context.textAlign = 'center'
    context.textBaseline = 'bottom'
    context.fillStyle = '#0b0f0a'
    context.strokeStyle = '#0b0f0a'
    context.lineWidth = 3 * ratio
    const text = `${label} · ${mark.sender}`
    context.strokeText(text, x, y - radius - 8 * ratio)
    context.fillStyle = '#8dfa77'
    context.fillText(text, x, y - radius - 8 * ratio)
    context.restore()
  })
}

function drawObject(context: CanvasRenderingContext2D, rect: MapRect, object: MapObject) {
  const ratio = Math.min(window.devicePixelRatio || 1, 2)
  if (object.type === 'airfield') {
    const { sx, sy, ex, ey } = object
    if (
      typeof sx !== 'number' ||
      typeof sy !== 'number' ||
      typeof ex !== 'number' ||
      typeof ey !== 'number' ||
      !Number.isFinite(sx) ||
      !Number.isFinite(sy) ||
      !Number.isFinite(ex) ||
      !Number.isFinite(ey)
    ) return
    context.save()
    context.strokeStyle = object.color ?? '#174dff'
    context.lineWidth = 3 * ratio
    context.beginPath()
    context.moveTo(rect.x + sx * rect.size, rect.y + sy * rect.size)
    context.lineTo(rect.x + ex * rect.size, rect.y + ey * rect.size)
    context.stroke()
    context.restore()
    return
  }

  const { x, y } = object
  if (typeof x !== 'number' || typeof y !== 'number' || !Number.isFinite(x) || !Number.isFinite(y)) return
  const screenX = rect.x + x * rect.size
  const screenY = rect.y + y * rect.size
  if (object.icon === 'Player') {
    drawPlayer(context, screenX, screenY, object, rect.size)
    return
  }
  if (object.type === 'point_of_interest') {
    drawTargetPoint(context, screenX, screenY, rect.size)
    return
  }
  context.save()
  context.translate(screenX, screenY)
  context.fillStyle = object.color ?? '#f00c00'
  context.strokeStyle = '#080808'
  context.lineWidth = ratio
  const size = Math.max(8 * ratio, rect.size * 0.014)
  if (object.type === 'aircraft') {
    context.rotate(Math.atan2(object.dx ?? 0, -(object.dy ?? -1)))
    context.beginPath()
    context.moveTo(0, -size)
    context.lineTo(size, size * 0.65)
    context.lineTo(0, size * 0.45)
    context.lineTo(-size, size * 0.65)
    context.closePath()
  } else if (object.icon === 'bombing_point') {
    context.beginPath()
    context.arc(0, 0, size * 0.8, 0, Math.PI * 2)
    context.moveTo(-size, 0)
    context.lineTo(size, 0)
    context.moveTo(0, -size)
    context.lineTo(0, size)
  } else if (object.icon === 'SAM' || object.icon === 'SPAA' || object.icon === 'Airdefence') {
    drawAirDefenseIcon(context, size, object.icon)
    context.restore()
    return
  } else {
    context.beginPath()
    context.rect(-size * 0.65, -size * 0.65, size * 1.3, size * 1.3)
  }
  context.fill()
  context.stroke()
  context.restore()
}

function drawTargetPoint(
  context: CanvasRenderingContext2D,
  x: number,
  y: number,
  mapSize: number,
) {
  const ratio = Math.min(window.devicePixelRatio || 1, 2)
  const halfSize = Math.max(10 * ratio, mapSize * 0.017)
  const arm = halfSize * 0.48
  context.save()
  context.translate(x, y)
  context.strokeStyle = 'rgba(255, 255, 255, 0.55)'
  context.lineWidth = 5 * ratio
  cornerSquarePath(context, halfSize, arm)
  context.stroke()
  context.strokeStyle = '#080808'
  context.lineWidth = 2.5 * ratio
  cornerSquarePath(context, halfSize, arm)
  context.stroke()
  context.restore()
}

function cornerSquarePath(
  context: CanvasRenderingContext2D,
  halfSize: number,
  arm: number,
) {
  context.beginPath()
  context.moveTo(-halfSize + arm, -halfSize)
  context.lineTo(-halfSize, -halfSize)
  context.lineTo(-halfSize, -halfSize + arm)
  context.moveTo(halfSize - arm, -halfSize)
  context.lineTo(halfSize, -halfSize)
  context.lineTo(halfSize, -halfSize + arm)
  context.moveTo(halfSize, halfSize - arm)
  context.lineTo(halfSize, halfSize)
  context.lineTo(halfSize - arm, halfSize)
  context.moveTo(-halfSize + arm, halfSize)
  context.lineTo(-halfSize, halfSize)
  context.lineTo(-halfSize, halfSize - arm)
}

function drawAirDefenseIcon(
  context: CanvasRenderingContext2D,
  size: number,
  icon: string,
) {
  const color = context.fillStyle
  const label = icon === 'SAM' ? 'SAM' : icon === 'SPAA' ? 'AAA' : 'AD'
  const width = size * 3.4
  const height = size * 1.75
  const radius = size * 0.35
  context.save()
  context.lineJoin = 'round'
  context.lineCap = 'round'
  context.fillStyle = 'rgba(7, 9, 10, 0.9)'
  roundedRectPath(context, -width / 2, -height / 2, width, height, radius)
  context.fill()
  context.strokeStyle = '#07090a'
  context.lineWidth = Math.max(3, size * 0.5)
  context.stroke()
  context.strokeStyle = color
  context.lineWidth = Math.max(1.5, size * 0.2)
  context.stroke()

  context.fillStyle = '#fff'
  context.font = `900 ${size * 0.9}px "Arial Narrow", Bahnschrift, sans-serif`
  context.textAlign = 'center'
  context.textBaseline = 'middle'
  context.fillText(label, 0, size * 0.05)
  context.restore()
}

function roundedRectPath(
  context: CanvasRenderingContext2D,
  x: number,
  y: number,
  width: number,
  height: number,
  radius: number,
) {
  context.beginPath()
  context.moveTo(x + radius, y)
  context.lineTo(x + width - radius, y)
  context.quadraticCurveTo(x + width, y, x + width, y + radius)
  context.lineTo(x + width, y + height - radius)
  context.quadraticCurveTo(x + width, y + height, x + width - radius, y + height)
  context.lineTo(x + radius, y + height)
  context.quadraticCurveTo(x, y + height, x, y + height - radius)
  context.lineTo(x, y + radius)
  context.quadraticCurveTo(x, y, x + radius, y)
  context.closePath()
}

function drawGroundClusters(context: CanvasRenderingContext2D, rect: MapRect, objects: MapObject[]) {
  const ratio = Math.min(window.devicePixelRatio || 1, 2)
  const clusters: Array<{ x: number; y: number; objects: MapObject[] }> = []
  const airDefense = objects.filter((object) =>
    object.icon === 'SAM' || object.icon === 'SPAA' || object.icon === 'Airdefence',
  )
  const otherGround = objects.filter((object) => !airDefense.includes(object))
  for (const object of otherGround) {
    const { x, y } = object
    if (typeof x !== 'number' || typeof y !== 'number' || !Number.isFinite(x) || !Number.isFinite(y)) continue
    const screenX = rect.x + x * rect.size
    const screenY = rect.y + y * rect.size
    const cluster = clusters.find((item) => Math.hypot(item.x - screenX, item.y - screenY) < 14 * ratio)
    if (cluster) {
      cluster.objects.push(object)
    } else {
      clusters.push({ x: screenX, y: screenY, objects: [object] })
    }
  }
  airDefense.forEach((object) => drawObject(context, rect, object))
  for (const cluster of clusters) {
    if (cluster.objects.length === 1) {
      drawObject(context, rect, cluster.objects[0])
      continue
    }
    context.save()
    context.fillStyle = cluster.objects[0].color ?? '#fa0c00'
    context.strokeStyle = '#080808'
    context.lineWidth = 2 * ratio
    context.beginPath()
    context.arc(cluster.x, cluster.y, 10 * ratio, 0, Math.PI * 2)
    context.fill()
    context.stroke()
    context.fillStyle = '#fff'
    context.font = `800 ${10 * ratio}px Arial`
    context.textAlign = 'center'
    context.textBaseline = 'middle'
    context.fillText(String(cluster.objects.length), cluster.x, cluster.y)
    context.restore()
  }
}

function drawNavigation(
  context: CanvasRenderingContext2D,
  rect: MapRect,
  snapshot: Snapshot,
  navigation?: NavigationSolution,
) {
  const player = (snapshot.map.objects ?? []).find((object) => object.icon === 'Player')
  if (!player || !navigation || typeof player.x !== 'number' || typeof player.y !== 'number') return
  const ratio = Math.min(window.devicePixelRatio || 1, 2)
  const startX = rect.x + player.x * rect.size
  const startY = rect.y + player.y * rect.size
  const targetX = rect.x + navigation.targetX * rect.size
  const targetY = rect.y + navigation.targetY * rect.size
  context.save()
  context.strokeStyle = '#f4b13d'
  context.lineWidth = 2 * ratio
  context.setLineDash([6 * ratio, 6 * ratio])
  context.beginPath()
  context.moveTo(startX, startY)
  context.lineTo(targetX, targetY)
  context.stroke()
  context.setLineDash([])
  drawSelectedTarget(context, targetX, targetY, navigation.name, ratio)
  context.restore()
}

function drawSelectedTarget(
  context: CanvasRenderingContext2D,
  x: number,
  y: number,
  name: string,
  ratio: number,
) {
  const halfSize = 17 * ratio
  const arm = 7 * ratio
  context.save()
  context.translate(x, y)
  context.strokeStyle = '#ffd166'
  context.lineWidth = 2 * ratio
  context.lineCap = 'square'
  cornerSquarePath(context, halfSize, arm)
  context.stroke()

  context.font = `800 ${9 * ratio}px "Inter", system-ui, sans-serif`
  context.textAlign = 'center'
  context.textBaseline = 'bottom'
  context.strokeStyle = '#090b0c'
  context.lineWidth = 4 * ratio
  const label = `ACTIVE · ${name.toUpperCase()}`
  context.strokeText(label, 0, -halfSize - 5 * ratio)
  context.fillStyle = '#ffd166'
  context.fillText(label, 0, -halfSize - 5 * ratio)
  context.restore()
}

function drawPlayer(context: CanvasRenderingContext2D, x: number, y: number, object: MapObject, mapSize: number) {
  const ratio = Math.min(window.devicePixelRatio || 1, 2)
  const length = Math.max(22 * ratio, mapSize * 0.04)
  const width = length * 0.28
  context.save()
  context.translate(x, y)
  context.rotate(Math.atan2(object.dx ?? 0, -(object.dy ?? -1)))
  context.fillStyle = '#fff'
  context.strokeStyle = '#262626'
  context.lineWidth = 2 * ratio
  context.beginPath()
  context.moveTo(0, -length * 0.6)
  context.lineTo(width, length * 0.42)
  context.lineTo(0, length * 0.2)
  context.lineTo(-width, length * 0.42)
  context.closePath()
  context.fill()
  context.stroke()
  context.restore()
}

function rowLabel(index: number) {
  let label = ''
  let value = index + 1
  while (value > 0) {
    value -= 1
    label = String.fromCharCode(65 + (value % 26)) + label
    value = Math.floor(value / 26)
  }
  return label
}
