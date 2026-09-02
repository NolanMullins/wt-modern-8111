import { useEffect, useState } from 'react'
import type { Snapshot } from '../types'

export interface LoadedMap {
  generation: number
  image: HTMLImageElement
}

export function useMapImage(snapshot: Snapshot | null): LoadedMap | null {
  const [mapImage, setMapImage] = useState<LoadedMap | null>(null)

  useEffect(() => {
    if (
      snapshot?.connection.mode !== 'live' ||
      !snapshot.map.valid ||
      snapshot.map.generation === undefined
    ) {
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

  return mapImage
}
