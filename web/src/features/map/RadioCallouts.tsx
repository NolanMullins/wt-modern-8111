import { useEffect, useState } from 'react'
import type { AllyMark } from '../../types'
import { activeRadioCallouts, calloutOpacity } from './radioCalloutModel'

export function RadioCallouts({ marks }: { marks: AllyMark[] }) {
  const [now, setNow] = useState(0)
  useEffect(() => {
    const timer = window.setInterval(() => setNow(Date.now()), 250)
    return () => window.clearInterval(timer)
  }, [])

  const active = activeRadioCallouts(marks, now)

  if (active.length === 0) return null

  return (
    <aside className="radio-callouts" aria-live="polite">
      <small>Team radio</small>
      {active.map((mark) => (
        <div
          className={`radio-callout radio-callout-${mark.subject}`}
          key={mark.key}
          style={{ opacity: calloutOpacity(mark, now) }}
        >
          <strong>{calloutTitle(mark)}</strong>
          <span>{mark.sender}</span>
          <em>{locationLabel(mark)}</em>
        </div>
      ))}
    </aside>
  )
}

function calloutTitle(mark: AllyMark) {
  if (mark.subject === 'target') return 'MAP PING'
  if (mark.kind === 'cover' || mark.kind === 'help') return 'SUPPORT REQUEST'
  return 'ALLY POSITION'
}

function locationLabel(mark: AllyMark) {
  if (!mark.located || !mark.grid) return 'Position unavailable'
  if (mark.subject === 'sender' && mark.altitudeM !== undefined) {
    return `${mark.grid} · ${Math.round(mark.altitudeM)} m`
  }
  return mark.grid
}
