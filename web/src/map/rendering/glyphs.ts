import type { MapObject } from '../../types'

export function cornerSquarePath(
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

export function drawTargetPoint(
  context: CanvasRenderingContext2D,
  x: number,
  y: number,
  mapSize: number,
) {
  const ratio = pixelRatio()
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

export function drawAirDefenseIcon(
  context: CanvasRenderingContext2D,
  size: number,
  icon: string,
) {
  const color = context.fillStyle
  const label = icon === 'SAM' ? 'SAM' : icon === 'SPAA' ? 'AAA' : 'AD'
  const width = size * 2.2
  const height = size * 1.15
  const radius = size * 0.25
  context.save()
  context.lineJoin = 'round'
  context.lineCap = 'round'
  context.fillStyle = 'rgba(7, 9, 10, 0.9)'
  roundedRectPath(context, -width / 2, -height / 2, width, height, radius)
  context.fill()
  context.strokeStyle = '#07090a'
  context.lineWidth = Math.max(2, size * 0.35)
  context.stroke()
  context.strokeStyle = color
  context.lineWidth = Math.max(1, size * 0.15)
  context.stroke()

  context.fillStyle = '#fff'
  context.font = `900 ${size * 0.55}px "Arial Narrow", Bahnschrift, sans-serif`
  context.textAlign = 'center'
  context.textBaseline = 'middle'
  context.fillText(label, 0, size * 0.05)
  context.restore()
}

export function drawPlayer(
  context: CanvasRenderingContext2D,
  x: number,
  y: number,
  object: MapObject,
  mapSize: number,
  groundMode = false,
) {
  const ratio = pixelRatio()
  const length = Math.max(22 * ratio, mapSize * 0.04)
  const width = length * 0.28
  context.save()
  context.translate(x, y)
  context.rotate(Math.atan2(object.dx ?? 0, -(object.dy ?? -1)))
  context.fillStyle = '#fff'
  context.strokeStyle = '#262626'
  context.lineWidth = 2 * ratio
  context.beginPath()
  if (groundMode) {
    context.moveTo(0, -length * 0.6)
    context.lineTo(width, -length * 0.25)
    context.lineTo(width, length * 0.42)
    context.lineTo(-width, length * 0.42)
    context.lineTo(-width, -length * 0.25)
  } else {
    context.moveTo(0, -length * 0.6)
    context.lineTo(width, length * 0.42)
    context.lineTo(0, length * 0.2)
    context.lineTo(-width, length * 0.42)
  }
  context.closePath()
  context.fill()
  context.stroke()
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

function pixelRatio() {
  return Math.min(window.devicePixelRatio || 1, 2)
}
