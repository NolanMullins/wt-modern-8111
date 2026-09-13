import type { NavigationSolution } from '../../navigation'
import type { AllyMark, Snapshot } from '../../types'
import { mapToCanvas, type MapRect } from '../geometry'
import { cornerSquarePath } from './glyphs'

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
  viewport: MapRect,
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
    const presentation = allyMarkPresentation(mark)

    context.save()
    context.globalAlpha = opacity
    context.strokeStyle = presentation.color
    context.fillStyle = presentation.fill
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

    context.font = `${10 * ratio}px "Inter", system-ui, sans-serif`
    context.textAlign = 'center'
    context.textBaseline = 'bottom'
    context.fillStyle = '#0b0f0a'
    context.strokeStyle = '#0b0f0a'
    context.lineWidth = 3 * ratio
    const text = presentation.label
    const halfWidth = context.measureText(text).width / 2
    const textX = Math.min(
      viewport.x + viewport.size - halfWidth - 4 * ratio,
      Math.max(viewport.x + halfWidth + 4 * ratio, position.x),
    )
    const textY = Math.max(
      viewport.y + 12 * ratio,
      position.y - radius - 8 * ratio,
    )
    context.strokeText(text, textX, textY)
    context.fillStyle = presentation.color
    context.fillText(text, textX, textY)
    context.restore()
  })
}

export function allyMarkPresentation(mark: AllyMark) {
  const location = [
    mark.grid,
    mark.subject === 'sender' && mark.altitudeM !== undefined
      ? `${Math.round(mark.altitudeM)} M`
      : undefined,
  ].filter(Boolean).join(' · ')
  const prefix = mark.subject === 'target'
    ? 'PING'
    : mark.kind === 'cover' || mark.kind === 'help'
      ? 'HELP'
      : 'ALLY'
  return {
    color: mark.subject === 'target' ? '#ffd166' : '#8dfa77',
    fill: mark.subject === 'target'
      ? 'rgba(255, 209, 102, 0.18)'
      : 'rgba(57, 217, 33, 0.16)',
    label: [prefix, mark.sender, location].filter(Boolean).join(' · '),
  }
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
