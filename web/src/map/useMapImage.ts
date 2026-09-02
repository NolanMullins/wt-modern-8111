import { useEffect, useState } from 'react'
import type { Snapshot } from '../types'

export interface LoadedMap {
  revision: number
  image: HTMLImageElement
}

export function useMapImage(snapshot: Snapshot | null, groundMap: boolean): LoadedMap | null {
  const [mapImage, setMapImage] = useState<LoadedMap | null>(null)

  useEffect(() => {
    if (
      snapshot?.connection.mode !== 'live' ||
      !snapshot.map.valid ||
      (snapshot.map.imageRevision ?? snapshot.map.generation) === undefined
    ) {
      return
    }
    const revision = snapshot.map.imageRevision ?? snapshot.map.generation!
    let cancelled = false
    let retry: number | undefined
    const load = () => {
      const image = new Image()
      image.onload = () => {
        if (!cancelled) setMapImage({ revision, image })
      }
      image.onerror = () => {
        if (!cancelled) retry = window.setTimeout(load, 1000)
      }
      const endpoint = groundMap ? 'ground-map' : 'map'
      image.src = `/api/v1/${endpoint}/${revision}?attempt=${Date.now()}`
    }
    load()
    return () => {
      cancelled = true
      if (retry !== undefined) window.clearTimeout(retry)
    }
  }, [
    snapshot?.connection.mode,
    snapshot?.map.generation,
    snapshot?.map.imageRevision,
    snapshot?.map.valid,
    groundMap,
  ])

  return mapImage
}
