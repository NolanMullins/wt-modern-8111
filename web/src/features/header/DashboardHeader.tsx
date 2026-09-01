import type { Snapshot } from '../../types'

export function DashboardHeader({
  snapshot,
  transport,
  error,
}: {
  snapshot: Snapshot | null
  transport: string
  error?: string
}) {
  const connection = connectionView(snapshot, transport, error)
  return (
    <header className="app-header">
      <i className="brand-flag" aria-hidden="true" />
      <div className="brand">
        <strong>WT Modern 8111</strong>
        <span>Simulator flight desk</span>
      </div>
      <div className="header-context">
        <b>{snapshot?.vehicle.type?.replaceAll('_', ' ').toUpperCase() || 'NO VEHICLE'}</b>
        {' · '}
        {snapshot?.connection.mode === 'fixture' ? 'Captured fixture' : 'Local companion'}
      </div>
      <div className="connection" data-state={connection.state}>
        {connection.label}
        <span>{connection.detail}</span>
      </div>
    </header>
  )
}

function connectionView(snapshot: Snapshot | null, transport: string, error?: string) {
  if (!snapshot) {
    return transport === 'reconnecting'
      ? { state: 'offline', label: 'Reconnecting', detail: error ?? 'waiting for companion' }
      : { state: 'offline', label: 'Connecting', detail: 'waiting for companion' }
  }
  if (snapshot.connection.mode === 'fixture') {
    return { state: 'fixture', label: 'Fixture', detail: 'captured session' }
  }
  if (transport !== 'streaming') {
    return {
      state: 'stale',
      label: 'Reconnecting',
      detail: error ?? 'holding last good snapshot',
    }
  }
  const labels: Record<string, string> = {
    live: 'Live',
    degraded: 'Degraded',
    hangar: 'Hangar',
    'waiting-for-game': 'Waiting',
  }
  if (snapshot.connection.state === 'live') {
    return { state: 'live', label: 'Live', detail: 'sources fresh' }
  }
  const ages = Object.values(snapshot.connection.sources)
    .map((source) => source.ageMs)
    .filter((age): age is number => age !== undefined)
  const age = ages.length ? Math.max(...ages) : 0
  return {
    state: 'stale',
    label: labels[snapshot.connection.state] ?? snapshot.connection.state,
    detail: age ? `oldest source ${Math.ceil(age / 1000)}s` : 'sources fresh',
  }
}
