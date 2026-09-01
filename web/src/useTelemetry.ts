import { useEffect, useState } from 'react'
import type { Snapshot } from './types'

type TransportState = 'connecting' | 'streaming' | 'reconnecting'

export function useTelemetry() {
  const [snapshot, setSnapshot] = useState<Snapshot | null>(null)
  const [transport, setTransport] = useState<TransportState>('connecting')

  useEffect(() => {
    let active = true

    fetch('/api/v1/snapshot', { cache: 'no-store' })
      .then((response) => {
        if (!response.ok) throw new Error(`snapshot HTTP ${response.status}`)
        return response.json() as Promise<Snapshot>
      })
      .then((value) => {
        if (active) setSnapshot(value)
      })
      .catch(() => {
        if (active) setTransport('reconnecting')
      })

    const events = new EventSource('/api/v1/events')
    events.addEventListener('open', () => {
      if (active) setTransport('streaming')
    })
    events.addEventListener('snapshot', (event) => {
      if (!active || !(event instanceof MessageEvent)) return
      setSnapshot(JSON.parse(event.data) as Snapshot)
      setTransport('streaming')
    })
    events.addEventListener('error', () => {
      if (active) setTransport('reconnecting')
    })

    return () => {
      active = false
      events.close()
    }
  }, [])
  return { snapshot, transport }
}
}
