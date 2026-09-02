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
  heatmapImage?: CanvasImageSource
  snapshot: Snapshot | null
  navigation?: NavigationSolution
  camera: MapCamera
}

export function drawMapFrame(
  context: CanvasRenderingContext2D,
  { width, height, image, heatmapImage, snapshot, navigation, camera }: MapFrame,
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
  drawHeatmap(context, rect, heatmapImage)
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
    drawAllyMarkOverlay(context, rect, snapshot)
  }

  function drawHeatmap(
    context: CanvasRenderingContext2D,
    rect: MapRect,
    image: CanvasImageSource | undefined,
  ) {
    if (!image) return
    context.save()
    context.globalAlpha = 0.9
    context.drawImage(image, rect.x, rect.y, rect.size, rect.size)
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
  context.font = `700 ${Math.max(11 * ratio, viewport.size * 0.018)}px Arial`
  for (let row = 0; row <= rows; row += 1) {
    const y = rect.y + (row / rows) * rect.size
    context.beginPath()
    context.moveTo(rect.x, y)
    context.lineTo(rect.x + rect.size, y)
    context.stroke()
    if (row < rows) {
      context.textAlign = 'left'
      context.textBaseline = 'middle'
      const centerY = y + rect.size / rows / 2
      if (centerY >= viewport.y && centerY <= viewport.y + viewport.size) {
        context.fillText(rowLabel(row), viewport.x + 5 * ratio, centerY)
      }
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
      const centerX = x + rect.size / columns / 2
      if (centerX >= viewport.x && centerX <= viewport.x + viewport.size) {
        context.fillText(String(column + 1), centerX, viewport.y + 5 * ratio)
      }
    }
  }
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
