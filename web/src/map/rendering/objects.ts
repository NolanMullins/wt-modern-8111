import type { MapObject } from '../../types'
import { mapToCanvas, pointInRect, type MapRect } from '../geometry'
import { drawAirDefenseIcon, drawPlayer, drawTargetPoint } from './glyphs'

export function drawObjectLayer(
  context: CanvasRenderingContext2D,
  rect: MapRect,
  viewport: MapRect,
  objects: MapObject[],
) {
  const symbolMapSize = viewport.size
  const ground = objects.filter((object) => object.type === 'ground_model')
  objects
    .filter((object) => object.type !== 'ground_model' && object.type !== 'aircraft')
    .forEach((object) => drawObject(context, rect, symbolMapSize, object))
  drawGroundClusters(context, rect, viewport, symbolMapSize, ground)
  objects
    .filter((object) => object.type === 'aircraft')
    .forEach((object) => drawObject(context, rect, symbolMapSize, object))
}

function drawObject(
  context: CanvasRenderingContext2D,
  rect: MapRect,
  symbolMapSize: number,
  object: MapObject,
) {
  const ratio = pixelRatio()
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
    const start = mapToCanvas({ x: sx, y: sy }, rect)
    const end = mapToCanvas({ x: ex, y: ey }, rect)
    context.save()
    context.strokeStyle = object.color ?? '#174dff'
    context.lineWidth = 3 * ratio
    context.beginPath()
    context.moveTo(start.x, start.y)
    context.lineTo(end.x, end.y)
    context.stroke()
    context.restore()
    return
  }

  const { x, y } = object
  if (typeof x !== 'number' || typeof y !== 'number' || !Number.isFinite(x) || !Number.isFinite(y)) return
  const screen = mapToCanvas({ x, y }, rect)
  if (object.icon === 'Player') {
    drawPlayer(context, screen.x, screen.y, object, symbolMapSize)
    return
  }
  if (object.type === 'point_of_interest') {
    drawTargetPoint(context, screen.x, screen.y, symbolMapSize)
    return
  }
  context.save()
  context.translate(screen.x, screen.y)
  context.fillStyle = object.color ?? '#f00c00'
  context.strokeStyle = '#080808'
  context.lineWidth = ratio
  const size = Math.max(8 * ratio, symbolMapSize * 0.014)
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
  } else if (isAirDefense(object)) {
    drawAirDefenseIcon(context, size, object.icon!)
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

function drawGroundClusters(
  context: CanvasRenderingContext2D,
  rect: MapRect,
  viewport: MapRect,
  symbolMapSize: number,
  objects: MapObject[],
) {
  const ratio = pixelRatio()
  const glyphExtent = Math.max(10 * ratio, symbolMapSize * 0.014)
  const clusters: Array<{ x: number; y: number; objects: MapObject[] }> = []
  const airDefense = objects.filter(isAirDefense)
  const otherGround = objects.filter((object) => !airDefense.includes(object))
  for (const object of otherGround) {
    const { x, y } = object
    if (typeof x !== 'number' || typeof y !== 'number' || !Number.isFinite(x) || !Number.isFinite(y)) continue
    const screen = mapToCanvas({ x, y }, rect)
    if (!pointInRect(screen, viewport, glyphExtent)) continue
    const cluster = clusters.find(
      (item) => Math.hypot(item.x - screen.x, item.y - screen.y) < 14 * ratio,
    )
    if (cluster) {
      cluster.objects.push(object)
      if (
        pointInRect(screen, viewport) &&
        !pointInRect({ x: cluster.x, y: cluster.y }, viewport)
      ) {
        cluster.x = screen.x
        cluster.y = screen.y
      }
    } else {
      clusters.push({ x: screen.x, y: screen.y, objects: [object] })
    }
  }
  airDefense.forEach((object) => drawObject(context, rect, symbolMapSize, object))
  for (const cluster of clusters) {
    if (cluster.objects.length === 1) {
      drawObject(context, rect, symbolMapSize, cluster.objects[0])
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

function isAirDefense(object: MapObject) {
  return object.icon === 'SAM' || object.icon === 'SPAA' || object.icon === 'Airdefence'
}

function pixelRatio() {
  return Math.min(window.devicePixelRatio || 1, 2)
}
