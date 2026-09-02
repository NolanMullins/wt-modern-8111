import { useEffect, useState } from 'react'
import type { Snapshot } from '../types'

export type HeatmapStatus = 'idle' | 'loading' | 'ready' | 'unavailable' | 'error'

export interface HeatmapState {
  generation?: number
  status: HeatmapStatus
  mapName?: string
  image?: CanvasImageSource
  baseImage?: CanvasImageSource
}

export function useHeatmapImage(
  snapshot: Snapshot | null,
  enabled: boolean,
): HeatmapState {
  const generation = snapshot?.map.imageRevision ?? snapshot?.map.generation
  const canLoad = enabled &&
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
      fetch(`/api/v1/heatmap/${generation}`, {
        cache: 'no-store',
        signal: controller.signal,
      }),
      fetch(`/api/v1/historical-map/${generation}`, {
        cache: 'no-store',
        signal: controller.signal,
      }),
    ])
      .then(async ([heatResponse, mapResponse]) => {
        if (heatResponse.status === 404 || mapResponse.status === 404) {
          if (!cancelled) setState({ generation, status: 'unavailable' })
          return
        }
        if (!heatResponse.ok) throw new Error(`heatmap HTTP ${heatResponse.status}`)
        if (!mapResponse.ok) throw new Error(`historical map HTTP ${mapResponse.status}`)
        const [heatBlob, mapBlob] = await Promise.all([
          heatResponse.blob(),
          mapResponse.blob(),
        ])
        if (cancelled) return
        const [heatImage, decodedBase] = await Promise.all([
          createImageBitmap(heatBlob),
          createImageBitmap(mapBlob),
        ])
        baseImage = decodedBase
        if (cancelled) {
          heatImage.close()
          baseImage.close()
          return
        }
        const gradient = buildHeatmapGradient(heatImage)
        heatImage.close()
        if (!cancelled) {
          setState({
            generation,
            status: 'ready',
            mapName: heatResponse.headers.get('X-WT-Heatmap-Map') ?? undefined,
            image: gradient,
            baseImage,
          })
        }
      })
      .catch((error: unknown) => {
        if (!cancelled && !(error instanceof DOMException && error.name === 'AbortError')) {
          setState({ generation, status: 'error' })
        }
      })

    return () => {
      cancelled = true
      controller.abort()
      baseImage?.close()
    }
  }, [canLoad, generation])

  if (!enabled) return { status: 'idle' }
  if (!canLoad) return { status: 'unavailable' }
  if (state.generation !== generation) return { generation, status: 'loading' }
  return state
}

function buildHeatmapGradient(image: ImageBitmap): HTMLCanvasElement {
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
  for (let pass = 0; pass < 8; pass += 1) {
    compressedContext.drawImage(image, 0, 0, width, height)
  }

  const gradient = document.createElement('canvas')
  gradient.width = width
  gradient.height = height
  const gradientContext = gradient.getContext('2d')
  if (!gradientContext) return compressed
  gradientContext.filter = 'blur(9px) saturate(170%) contrast(125%)'
  gradientContext.drawImage(compressed, 0, 0)
  gradientContext.filter = 'none'
  gradientContext.globalAlpha = 0.5
  gradientContext.drawImage(compressed, 0, 0)
  return gradient
}
