import type { NavigationSolution } from '../../navigation'
import type { Snapshot } from '../../types'
import { mapToCanvas, type MapRect } from '../geometry'
import { cornerSquarePath } from './glyphs'

const allyMarkLabels: Record<string, string> = {
  guide: 'GUIDE ON ME',
  attention: 'ATTENTION',
  cover: 'COVER ME',
  help: 'NEEDS HELP',
}
const allyMarkFadeMilliseconds = 5_000

export function drawNavigationOverlay(
  context: CanvasRenderingContext2D,
  rect: MapRect,
  snapshot: Snapshot,
  navigation?: NavigationSolution,
) {
  const player = (snapshot.map.objects ?? []).find((object) => object.icon === 'Player')
  if (!player || !navigation || typeof player.x !== 'number' || typeof player.y !== 'number') return
  const ratio = pixelRatio()
  const start = mapToCanvas({ x: player.x, y: player.y }, rect)
  const target = mapToCanvas({ x: navigation.targetX, y: navigation.targetY }, rect)
  context.save()
  context.strokeStyle = '#f4b13d'
  context.lineWidth = 2 * ratio
  context.setLineDash([6 * ratio, 6 * ratio])
  context.beginPath()
  context.moveTo(start.x, start.y)
  context.lineTo(target.x, target.y)
  context.stroke()
  context.setLineDash([])
  drawSelectedTarget(context, target.x, target.y, navigation.name, ratio)
  context.restore()
}

export function drawAllyMarkOverlay(
  context: CanvasRenderingContext2D,
  rect: MapRect,
  snapshot: Snapshot,
) {
  const marks = (snapshot.allyMarks ?? []).filter(
    (mark) =>
      mark.located &&
      typeof mark.x === 'number' &&
      typeof mark.y === 'number' &&
      new Date(mark.expiresAt).getTime() > Date.now(),
  )
  if (marks.length === 0) return
  const ratio = pixelRatio()
  const now = Date.now()

  marks.forEach((mark) => {
    const position = mapToCanvas({ x: mark.x as number, y: mark.y as number }, rect)
    const age = (now - new Date(mark.createdAt).getTime()) / 1000
    const remaining = new Date(mark.expiresAt).getTime() - now
    const opacity = Math.min(1, Math.max(0, remaining / allyMarkFadeMilliseconds))
    const pulse = age < 6 ? 1 + 0.25 * Math.sin(age * Math.PI * 2) : 1
    const radius = 13 * ratio * pulse

    context.save()
    context.globalAlpha = opacity
    context.strokeStyle = '#39d921'
    context.fillStyle = 'rgba(57, 217, 33, 0.16)'
    context.lineWidth = 2 * ratio

    context.beginPath()
    context.arc(position.x, position.y, radius, 0, Math.PI * 2)
    context.fill()
    context.stroke()

    context.beginPath()
    context.moveTo(position.x - radius - 5 * ratio, position.y)
    context.lineTo(position.x + radius + 5 * ratio, position.y)
    context.moveTo(position.x, position.y - radius - 5 * ratio)
    context.lineTo(position.x, position.y + radius + 5 * ratio)
    context.stroke()

    const label = allyMarkLabels[mark.kind] ?? mark.kind.toUpperCase()
    context.font = `${10 * ratio}px "Inter", system-ui, sans-serif`
    context.textAlign = 'center'
    context.textBaseline = 'bottom'
    context.fillStyle = '#0b0f0a'
    context.strokeStyle = '#0b0f0a'
    context.lineWidth = 3 * ratio
    const text = `${label} · ${mark.sender}`
    context.strokeText(text, position.x, position.y - radius - 8 * ratio)
    context.fillStyle = '#8dfa77'
    context.fillText(text, position.x, position.y - radius - 8 * ratio)
    context.restore()
  })
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

function pixelRatio() {
  return Math.min(window.devicePixelRatio || 1, 2)
}
