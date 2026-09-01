import { EmptyState, PanelHead } from '../../shared/presentation'
import type { Snapshot } from '../../types'

export function FeedPanel({ snapshot }: { snapshot: Snapshot | null }) {
  const feed = (snapshot?.feed ?? []).slice(0, 6)
  return (
    <section className="panel feed-panel">
      <PanelHead title="Comms and events" status={feed.length ? `${feed.length} latest` : 'No records'} />
      <div className="feed-body">
        {feed.length
          ? feed.map((entry) => (
              <div className="feed-row" key={entry.key}>
                <time>{feedTime(entry.time, entry.addedAt)}</time>
                <span className={`feed-type ${entry.kind}`}>{entry.kind}</span>
                <span>{entry.sender && <b>{entry.sender}: </b>}{entry.message}</span>
              </div>
            ))
          : <EmptyState>No chat, HUD events, or damage records received.</EmptyState>}
      </div>
    </section>
  )
}

function feedTime(missionSeconds: number | undefined, addedAt: string) {
  if (missionSeconds !== undefined) {
    const rounded = Math.max(0, Math.round(missionSeconds))
    return `T+${String(Math.floor(rounded / 60)).padStart(2, '0')}:${String(rounded % 60).padStart(2, '0')}`
  }
  return new Date(addedAt).toLocaleTimeString([], {
    hour: '2-digit',
    minute: '2-digit',
  })
}
