import { useEffect, useState } from 'react'
import { parseSnapshotPayload, selectNewerSnapshot } from './telemetryIngress'
import type { Snapshot } from './types'

type TransportState = 'connecting' | 'streaming' | 'reconnecting'

export function useTelemetry() {
  const [snapshot, setSnapshot] = useState<Snapshot | null>(null)
  const [transport, setTransport] = useState<TransportState>('connecting')
  const [error, setError] = useState<string>()

  useEffect(() => {
    let active = true

    const acceptPayload = (payload: string) => {
      const result = parseSnapshotPayload(payload)
      if (!result.ok) {
        setError(result.error)
        setTransport('reconnecting')
        return false
      }
      setError(undefined)
      setSnapshot((current) => selectNewerSnapshot(current, result.snapshot))
      return true
    }

    fetch('/api/v1/snapshot', { cache: 'no-store' })
      .then((response) => {
        if (!response.ok) throw new Error(`snapshot HTTP ${response.status}`)
        return response.text()
      })
      .then((payload) => {
        if (active) acceptPayload(payload)
      })
      .catch((requestError: unknown) => {
        if (active) {
          setError(requestError instanceof Error ? requestError.message : 'Snapshot request failed')
          setTransport('reconnecting')
        }
      })

    const events = new EventSource('/api/v1/events')
    events.addEventListener('open', () => {
      if (active) setTransport('streaming')
    })
    events.addEventListener('snapshot', (event) => {
      if (!active || !(event instanceof MessageEvent)) return
      if (acceptPayload(event.data)) setTransport('streaming')
    })
    events.addEventListener('error', () => {
      if (active) {
        setError('Telemetry stream disconnected')
        setTransport('reconnecting')
      }
    })

    return () => {
      active = false
      events.close()
    }
  }, [])
  return { snapshot, transport, error }
}
