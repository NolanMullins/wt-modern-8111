import { useEffect, useState } from 'react'
import type { Snapshot } from '../types'
import type { PlayerTeam } from './usePlayerTeam'

export type HeatmapStatus =
  | 'idle'
  | 'waiting-team'
  | 'loading'
  | 'ready'
  | 'unavailable'
  | 'error'

export interface HeatmapState {
  generation?: number
  enemyTeam?: PlayerTeam
  status: HeatmapStatus
  mapName?: string
  firingImage?: CanvasImageSource
  victimImage?: CanvasImageSource
  baseImage?: CanvasImageSource
}

export function useHeatmapImage(
  snapshot: Snapshot | null,
  enabled: boolean,
  enemyTeam?: PlayerTeam,
): HeatmapState {
  const generation = snapshot?.map.imageRevision ?? snapshot?.map.generation
  const canLoad = enabled &&
    enemyTeam !== undefined &&
    snapshot?.connection.mode === 'live' &&
    snapshot.map.valid &&
    generation !== undefined
  const [state, setState] = useState<HeatmapState>({ status: 'idle' })

  useEffect(() => {
    if (!canLoad || generation === undefined) return
    let cancelled = false
    let baseImage: ImageBitmap | undefined
    const controller = new AbortController()

    Promise.all([
      fetch(`/api/v1/heatmap/${generation}?team=${enemyTeam}&layer=firing`, {
        cache: 'no-store',
        signal: controller.signal,
      }),
      fetch(`/api/v1/heatmap/${generation}?team=${enemyTeam}&layer=victims`, {
        cache: 'no-store',
        signal: controller.signal,
      }),
      fetch(`/api/v1/historical-map/${generation}?team=${enemyTeam}`, {
        cache: 'no-store',
        signal: controller.signal,
      }),
    ])
      .then(async ([firingResponse, victimResponse, mapResponse]) => {
        if ([firingResponse, victimResponse, mapResponse].some((response) =>
          response.status === 404
        )) {
          if (!cancelled) setState({ generation, enemyTeam, status: 'unavailable' })
          return
        }
        if (!firingResponse.ok) {
          throw new Error(`firing heatmap HTTP ${firingResponse.status}`)
        }
        if (!victimResponse.ok) {
          throw new Error(`victim heatmap HTTP ${victimResponse.status}`)
        }
        if (!mapResponse.ok) throw new Error(`historical map HTTP ${mapResponse.status}`)
        const [firingBlob, victimBlob, mapBlob] = await Promise.all([
          firingResponse.blob(),
          victimResponse.blob(),
          mapResponse.blob(),
        ])
        if (cancelled) return
        const [firingSource, victimSource, decodedBase] = await Promise.all([
          createImageBitmap(firingBlob),
          createImageBitmap(victimBlob),
          createImageBitmap(mapBlob),
        ])
        baseImage = decodedBase
        if (cancelled) {
          firingSource.close()
          victimSource.close()
          baseImage.close()
          return
        }
        const firingImage = buildPreciseLayer(firingSource)
        const victimImage = buildPreciseLayer(victimSource)
        firingSource.close()
        victimSource.close()
        if (!cancelled) {
          setState({
            generation,
            enemyTeam,
            status: 'ready',
            mapName: firingResponse.headers.get('X-WT-Heatmap-Map') ?? undefined,
            firingImage,
            victimImage,
            baseImage,
          })
        }
      })
      .catch((error: unknown) => {
        if (!cancelled && !(error instanceof DOMException && error.name === 'AbortError')) {
          setState({ generation, enemyTeam, status: 'error' })
        }
      })

    return () => {
      cancelled = true
      controller.abort()
      baseImage?.close()
      setState((current) =>
        current.generation === generation && current.enemyTeam === enemyTeam
          ? { status: 'idle' }
          : current
      )
    }
  }, [canLoad, enemyTeam, generation])

  if (!enabled) return { status: 'idle' }
  if (enemyTeam === undefined) return { status: 'waiting-team' }
  if (!canLoad) return { status: 'unavailable' }
  if (state.generation !== generation || state.enemyTeam !== enemyTeam) {
    return { generation, enemyTeam, status: 'loading' }
  }
  return state
}

function buildPreciseLayer(image: ImageBitmap): HTMLCanvasElement {
  const maxSize = 1024
  const scale = Math.min(1, maxSize / Math.max(image.width, image.height))
  const width = Math.max(1, Math.round(image.width * scale))
  const height = Math.max(1, Math.round(image.height * scale))
  const compressed = document.createElement('canvas')
  compressed.width = width
  compressed.height = height
  const compressedContext = compressed.getContext('2d')
  if (!compressedContext) return compressed
  compressedContext.globalCompositeOperation = 'screen'
  for (let pass = 0; pass < 4; pass += 1) {
    compressedContext.drawImage(image, 0, 0, width, height)
  }

  const gradient = document.createElement('canvas')
  gradient.width = width
  gradient.height = height
  const gradientContext = gradient.getContext('2d')
  if (!gradientContext) return compressed
  gradientContext.filter = 'blur(1px) saturate(135%) contrast(115%)'
  gradientContext.drawImage(compressed, 0, 0)
  gradientContext.filter = 'none'
  gradientContext.globalAlpha = 0.75
  gradientContext.drawImage(compressed, 0, 0)
  return gradient
}
