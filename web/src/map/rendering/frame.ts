import { vehicleModeForSnapshot } from '../../battleMode'
import type { NavigationSolution } from '../../navigation'
import type { Snapshot } from '../../types'
import { mapRectForCamera, type MapCamera } from '../camera'
import { squareRect, type MapRect } from '../geometry'
import { drawObjectLayer } from './objects'
import { drawAllyMarkOverlay, drawNavigationOverlay } from './overlays'

interface MapFrame {
  width: number
  height: number
  image: CanvasImageSource | null
  heatmapImages?: CanvasImageSource[]
  snapshot: Snapshot | null
  navigation?: NavigationSolution
  camera: MapCamera
}

export function drawMapFrame(
  context: CanvasRenderingContext2D,
  { width, height, image, heatmapImages, snapshot, navigation, camera }: MapFrame,
) {
  const viewport = squareRect(width, height)
  const rect = mapRectForCamera(viewport, camera)
  context.clearRect(0, 0, width, height)
  context.fillStyle = '#111519'
  context.fillRect(0, 0, width, height)
  context.save()
  context.beginPath()
  context.rect(viewport.x, viewport.y, viewport.size, viewport.size)
  context.clip()
  drawBackground(context, rect, image)
  drawHeatmaps(context, rect, heatmapImages)
  if (snapshot) {
    drawGrid(context, viewport, rect, snapshot)
    drawObjectLayer(
      context,
      rect,
      viewport,
      snapshot.map.objects ?? [],
      vehicleModeForSnapshot(snapshot) === 'ground',
    )
    drawNavigationOverlay(context, rect, snapshot, navigation)
    drawAllyMarkOverlay(context, rect, viewport, snapshot)
  }

  function drawHeatmaps(
    context: CanvasRenderingContext2D,
    rect: MapRect,
    images: CanvasImageSource[] | undefined,
  ) {
    if (!images?.length) return
    context.save()
    context.globalAlpha = 0.88
    for (const image of images) {
      context.drawImage(image, rect.x, rect.y, rect.size, rect.size)
    }
    context.restore()
  }
  context.restore()
}

function drawBackground(
  context: CanvasRenderingContext2D,
  rect: MapRect,
  image: CanvasImageSource | null,
) {
  if (image) {
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

function drawGrid(
  context: CanvasRenderingContext2D,
  viewport: MapRect,
  rect: MapRect,
  snapshot: Snapshot,
) {
  const { mapMin, mapMax, gridSteps, gridZero } = snapshot.map
  if (!mapMin || !mapMax || !gridSteps ||
    mapMin.length < 2 || mapMax.length < 2 || gridSteps.length < 2) return
  const width = mapMax[0] - mapMin[0]
  const height = mapMax[1] - mapMin[1]
  const gridOrigin = gridZero && gridZero.length >= 2
    ? gridZero
    : [mapMin[0], mapMax[1]]
  if (![width, height, gridSteps[0], gridSteps[1], ...gridOrigin].every(Number.isFinite) ||
    width <= 0 || height <= 0 || gridSteps[0] <= 0 || gridSteps[1] <= 0) return
  const columns = visibleGridCells(mapMin[0], mapMax[0], gridOrigin[0], gridSteps[0], 1)
  const rows = visibleGridCells(mapMin[1], mapMax[1], gridOrigin[1], gridSteps[1], -1)
  const ratio = Math.min(window.devicePixelRatio || 1, 2)

  context.save()
  context.strokeStyle = 'rgba(255,255,255,.16)'
  context.fillStyle = 'rgba(238,240,238,.9)'
  context.shadowColor = 'rgba(0,0,0,.95)'
  context.shadowBlur = 3 * ratio
  context.lineWidth = ratio
  context.font = `700 ${Math.max(11 * ratio, viewport.size * 0.018)}px Arial`
  for (const row of rows) {
    const y = rect.y + ((mapMax[1] - row.boundary) / height) * rect.size
    context.beginPath()
    context.moveTo(rect.x, y)
    context.lineTo(rect.x + rect.size, y)
    context.stroke()
    if (row.center >= mapMin[1]) {
      context.textAlign = 'left'
      context.textBaseline = 'middle'
      const centerY = rect.y + ((mapMax[1] - row.center) / height) * rect.size
      if (centerY >= viewport.y && centerY <= viewport.y + viewport.size) {
        context.fillText(String(row.index + 1), viewport.x + 5 * ratio, centerY)
      }
    }
  }
  for (const column of columns) {
    const x = rect.x + ((column.boundary - mapMin[0]) / width) * rect.size
    context.beginPath()
    context.moveTo(x, rect.y)
    context.lineTo(x, rect.y + rect.size)
    context.stroke()
    if (column.center <= mapMax[0]) {
      context.textAlign = 'center'
      context.textBaseline = 'top'
      const centerX = rect.x + ((column.center - mapMin[0]) / width) * rect.size
      if (centerX >= viewport.x && centerX <= viewport.x + viewport.size) {
        context.fillText(gridColumnLabel(column.index), centerX, viewport.y + 5 * ratio)
      }
    }
  }
  context.restore()
}

export function visibleGridCells(
  min: number,
  max: number,
  zero: number,
  step: number,
  direction: 1 | -1,
) {
  const firstIndex = Math.max(
    0,
    Math.ceil(direction === 1 ? (min - zero) / step : (zero - max) / step),
  )
  const cells: Array<{ index: number; boundary: number; center: number }> = []
  for (let index = firstIndex; index < firstIndex + 100; index += 1) {
    const boundary = zero + direction * index * step
    if ((direction === 1 && boundary > max) ||
      (direction === -1 && boundary < min)) break
    cells.push({
      index,
      boundary,
      center: boundary + direction * step / 2,
    })
  }
  return cells
}

export function gridColumnLabel(index: number) {
  let label = ''
  let value = index + 1
  while (value > 0) {
    value -= 1
    label = String.fromCharCode(65 + (value % 26)) + label
    value = Math.floor(value / 26)
  }
  return label
}
