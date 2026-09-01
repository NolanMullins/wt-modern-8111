import { useEffect, useMemo, useRef, useState } from 'react'
import { TacticalMap } from './TacticalMap'
import {
  inGameTarget,
  missionTarget,
  navigationToTarget,
  reconcileSelectedTarget,
  type NavigationSolution,
  type SelectedTarget,
} from './navigation'
import type { Snapshot } from './types'
import { useFuelEstimate, type FuelEstimate } from './useFuelEstimate'
import { useTelemetry } from './useTelemetry'

const integer = new Intl.NumberFormat('en-US', { maximumFractionDigits: 0 })
const decimal = new Intl.NumberFormat('en-US', { minimumFractionDigits: 1, maximumFractionDigits: 1 })

function App() {
  const { snapshot, transport } = useTelemetry()
  const connection = connectionView(snapshot, transport)
  const [selectedTarget, setSelectedTarget] = useState<SelectedTarget | null>(null)
  const lastGameTargetKey = useRef<string | null>(null)
  const gameTarget = inGameTarget(snapshot)
  const gameTargetKey = gameTarget?.key ?? null

  useEffect(() => {
    if (!snapshot) return
    const previousGameTargetKey = lastGameTargetKey.current
    lastGameTargetKey.current = gameTargetKey
    // The telemetry snapshot is an external stream. Reconcile object identity
    // here so moving units remain selected even if map-object ordering changes.
    // oxlint-disable-next-line react/set-state-in-effect
    setSelectedTarget((current) => {
      if (gameTarget && gameTargetKey !== previousGameTargetKey) return gameTarget
      if (
        !gameTarget &&
        current?.basis === 'game target' &&
        current.object?.type === 'point_of_interest'
      ) {
        return { ...current, object: undefined }
      }
      return reconcileSelectedTarget(snapshot, current)
    })
  }, [gameTarget, gameTargetKey, snapshot])
  const effectiveTarget = selectedTarget &&
    (selectedTarget.generation === undefined ||
      snapshot?.map.generation === undefined ||
      selectedTarget.generation === snapshot.map.generation)
    ? selectedTarget
    : null
  const manualNavigation = useMemo(
    () => navigationToTarget(snapshot, effectiveTarget),
    [effectiveTarget, snapshot],
  )
  const navigation = manualNavigation ?? snapshot?.navigation
  const fuelEstimate = useFuelEstimate(snapshot)

  return (
    <main className="dashboard">
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

      <section className="workspace">
        <FlightPanel snapshot={snapshot} />
        <NavigationPanel
          navigation={navigation}
          radioRTB={snapshot?.navigation != null}
          canClear={effectiveTarget !== null}
          onClear={() => setSelectedTarget(null)}
        />
        <SystemsPanel snapshot={snapshot} fuelEstimate={fuelEstimate} />
        <MapPanel
          snapshot={snapshot}
          navigation={navigation}
          selectedTarget={effectiveTarget}
          onSelectTarget={setSelectedTarget}
          fuelEstimate={fuelEstimate}
        />
        <MissionPanel snapshot={snapshot} onSelectTarget={setSelectedTarget} />
        <FeedPanel snapshot={snapshot} />
      </section>
    </main>
  )
}

function FlightPanel({ snapshot }: { snapshot: Snapshot | null }) {
  const flight = snapshot?.flight
  return (
    <section className="panel flight-panel">
      <PanelHead title="Primary flight" status={flight?.iasKmh !== undefined ? 'Valid' : 'Unavailable'} good={flight?.iasKmh !== undefined} />
      <div className="metric-grid">
        <Metric label="IAS" value={format(flight?.iasKmh)} unit="km/h" />
        <Metric label="TAS" value={format(flight?.tasKmh)} unit="km/h" />
        <Metric label="Altitude" value={format(flight?.altitudeM)} unit="m" />
        <Metric label="Mach" value={format(flight?.mach, 2)} />
        <Metric label="Heading" value={heading(flight?.headingDeg)} unit="deg" />
        <Metric label="Vertical speed" value={signed(flight?.verticalSpeedMps)} unit="m/s" />
        <Metric label="AoA" value={format(flight?.aoaDeg, 1)} unit="deg" />
        <Metric label="Load" value={format(flight?.gLoad, 2)} unit="g" />
      </div>
    </section>
  )
}

function NavigationPanel({
  navigation,
  radioRTB,
  canClear,
  onClear,
}: {
  navigation?: NavigationSolution
  radioRTB: boolean
  canClear: boolean
  onClear: () => void
}) {
  return (
    <section className="panel nav-panel">
      <PanelHead
        title="Selected navigation"
        status={navigation ? (radioRTB ? 'Radio RTB' : navigation.basis) : 'No destination'}
      />
      <div className="nav-body">
        <div className="destination">
          <span>Active destination</span>
          <strong>{navigation?.name ?? 'Select on map or mission'}</strong>
          {canClear && !radioRTB && (
            <button className="clear-target" type="button" onClick={onClear}>Clear</button>
          )}
        </div>
        <NavStat label="Bearing" value={navigation ? heading(navigation.bearingDeg) : '—'} unit="deg" />
        <NavStat label="Range" value={navigation ? decimal.format(navigation.rangeKm) : '—'} unit="km" />
        <NavStat label="ETA" value={duration(navigation?.etaSeconds)} />
      </div>
    </section>
  )
}

function SystemsPanel({
  snapshot,
  fuelEstimate,
}: {
  snapshot: Snapshot | null
  fuelEstimate: FuelEstimate
}) {
  const systems = snapshot?.systems
  const engines = systems?.engines ?? []
  const live = engines.filter((item) => item.status === 'running')
  const reference = live[0] ?? engines[0]
  const failed = engines.filter((item) => item.status === 'failed')
  const linked = engines.length > 1 && engines.every((item) =>
    item.throttlePercent === reference?.throttlePercent &&
    item.rpm === reference?.rpm &&
    item.oilTempC === reference?.oilTempC &&
    item.thrustKgf === reference?.thrustKgf,
  )
  const configuration = [systems?.gearPercent, systems?.flapsPercent, systems?.airbrakePercent]
  const configurationKnown = configuration.every((value) => value !== undefined)
  const clean = configurationKnown && configuration.every((value) => value === 0)
  const configurationLabel = configurationKnown ? (clean ? 'clean configuration' : 'configuration extended') : 'configuration unavailable'

  const engineLabel = engines.length
    ? `Engine${engines.length > 1 ? 's' : ''} ${engines.map((item) => item.index).join(' + ')}${linked ? ' linked' : ''}`
    : 'No engine data'

  return (
    <section className="panel systems-panel">
      <PanelHead
        title="Aircraft systems"
        status={systems?.status ?? 'Unavailable'}
        severity={systems?.severity ?? 'unknown'}
      />
      <div className="systems-body">
        <div className="engine-summary">
          <strong>{engineLabel}</strong>
          <span>
            {reference?.throttlePercent !== undefined ? `Throttle ${format(reference.throttlePercent)}%` : 'Throttle unavailable'} · {configurationLabel}
          </span>
          {failed.length > 0 && (
            <span className="system-alert">
              {failed.map((item) => `Engine ${item.index} out`).join(' · ')}
            </span>
          )}
          <span className={`bingo-indicator bingo-${fuelEstimate.state}`}>
            {fuelEstimate.bingoFuelKg !== undefined
              ? `${fuelEstimate.label} · Bingo ${format(fuelEstimate.bingoFuelKg)} kg · Margin ${signedKg(fuelEstimate.marginKg)} · direct return`
              : fuelEstimate.label}
          </span>
        </div>
        <div className="system-stats">
          <SystemStat label="RPM" value={format(reference?.rpm)} />
          <SystemStat label="Oil" value={format(reference?.oilTempC)} unit="C" />
          <SystemStat label="Thrust" value={format(reference?.thrustKgf)} unit={linked ? `kgf × ${engines.length}` : 'kgf'} />
          <SystemStat label="Fuel" value={format(systems?.fuelKg)} unit={systems?.fuelPercent !== undefined ? `kg · ${format(systems.fuelPercent)}%` : 'kg'} />
        </div>
      </div>
    </section>
  )
}

function MapPanel({
  snapshot,
  navigation,
  selectedTarget,
  onSelectTarget,
  fuelEstimate,
}: {
  snapshot: Snapshot | null
  navigation?: NavigationSolution
  selectedTarget: SelectedTarget | null
  onSelectTarget: (target: SelectedTarget) => void
  fuelEstimate: FuelEstimate
}) {
  const counts = snapshot?.map.counts
  return (
    <section className="panel map-panel">
      <TacticalMap
        snapshot={snapshot}
        navigation={navigation}
        selectedTarget={selectedTarget}
        onSelectTarget={onSelectTarget}
      />
      <div className="map-overlay">
        <div className="map-name">
          <small>Full mission frame</small>
          <strong>Tactical map</strong>
        </div>
        <BingoFuelHUD estimate={fuelEstimate} />
        <div className="map-mode">
          {snapshot?.connection.mode === 'fixture' ? 'Captured map fixture' : 'Click object or map point'}
          <br />
          {snapshot?.map.valid ? `${counts?.total ?? 0} exposed objects` : 'Map unavailable'}
        </div>
      </div>
      <div className="map-legend">
        <Legend color="#fff" label="Ownship" />
        <Legend color="#f00c00" label={`${counts?.hostileAir ?? 0} hostile air`} shape="air" />
        <Legend color="#fa0c00" label={`${counts?.ground ?? 0} ground`} />
        <Legend color="#fa0c00" label={`${counts?.airDefense ?? 0} air defense`} shape="airdef" />
        <Legend color="#fa0c00" label={`${counts?.strikePoint ?? 0} strike points`} shape="target" />
        <Legend color="#174dff" label={`${counts?.airfield ?? 0} airfield`} shape="airfield" />
      </div>
      {!snapshot?.map.valid && <div className="map-warning">Waiting for valid map metadata</div>}
    </section>
  )
}

function BingoFuelHUD({ estimate }: { estimate: FuelEstimate }) {
  const calibrated = estimate.burnKgPerHour !== undefined
  const primary = estimate.bingoFuelKg !== undefined
    ? `${format(estimate.bingoFuelKg)} KG`
    : estimate.state === 'unlimited'
      ? 'UNLIMITED'
      : estimate.state === 'building'
        ? estimate.sampleSeconds !== undefined
          ? `CAL ${Math.round(estimate.sampleSeconds)}/8`
          : 'CAL'
        : 'NO SOLUTION'
  const detail = estimate.marginKg !== undefined
    ? `Margin ${signedKg(estimate.marginKg)}`
    : estimate.label
  return (
    <div className={`bingo-hud bingo-${estimate.state}`}>
      <small>Bingo fuel</small>
      <strong>{primary}</strong>
      <span>{detail}</span>
      {calibrated && (
        <em>{format(estimate.burnKgPerHour)} kg/h at 100% · no reserve</em>
      )}
    </div>
  )
}

function MissionPanel({
  snapshot,
  onSelectTarget,
}: {
  snapshot: Snapshot | null
  onSelectTarget: (target: SelectedTarget) => void
}) {
  const objectives = distinctObjectives(snapshot?.mission.objectives ?? [])
  const primary = objectives.filter((objective) => objective.primary)
  const secondary = objectives.filter((objective) => !objective.primary)
  return (
    <section className="panel mission-panel">
      <PanelHead title="Active mission" status={snapshot?.mission.status || 'Unavailable'} good={snapshot?.mission.status === 'running'} />
      <div className="mission-body">
        <ObjectiveGroup
          title="Primary objectives"
          objectives={primary}
          snapshot={snapshot}
          onSelectTarget={onSelectTarget}
          primary
        />
        <ObjectiveGroup
          title="Secondary objectives"
          objectives={secondary}
          snapshot={snapshot}
          onSelectTarget={onSelectTarget}
        />
      </div>
    </section>
  )
}

function distinctObjectives(objectives: Snapshot['mission']['objectives']) {
  const distinct = new Map<string, Snapshot['mission']['objectives'][number]>()
  for (const objective of objectives) {
    const key = `${objective.primary}:${objective.text}`
    const existing = distinct.get(key)
    if (!existing || existing.status === 'undefined') distinct.set(key, objective)
  }
  return [...distinct.values()]
}

function FeedPanel({ snapshot }: { snapshot: Snapshot | null }) {
  const feed = (snapshot?.feed ?? []).slice(0, 6)
  return (
    <section className="panel feed-panel">
      <PanelHead title="Comms and events" status={feed.length ? `${feed.length} latest` : 'No records'} />
      <div className="feed-body">
        {feed.length ? feed.map((entry) => (
          <div className="feed-row" key={entry.key}>
            <time>{feedTime(entry.time, entry.addedAt)}</time>
            <span className={`feed-type ${entry.kind}`}>{entry.kind}</span>
            <span>{entry.sender && <b>{entry.sender}: </b>}{entry.message}</span>
          </div>
        )) : <EmptyState>No chat, HUD events, or damage records received.</EmptyState>}
      </div>
    </section>
  )
}

function ObjectiveGroup({ title, objectives, snapshot, onSelectTarget, primary = false }: {
  title: string
  objectives: Snapshot['mission']['objectives']
  snapshot: Snapshot | null
  onSelectTarget: (target: SelectedTarget) => void
  primary?: boolean
}) {
  return (
    <section className="objective-group">
      <h3>{title}</h3>
      {objectives.length ? objectives.map((objective, index) => {
        const target = snapshot ? missionTarget(snapshot, objective) : undefined
        return (
          <button
            className="objective"
            disabled={!target}
            key={`${objective.text}-${index}`}
            onClick={() => target && onSelectTarget(target)}
            title={target ? 'Set as active destination' : 'No map position is exposed for this objective'}
            type="button"
          >
            <i className={primary ? 'primary' : ''}>{primary ? 'P' : 'S'}</i>
            <span className="objective-copy">
              <strong>{objective.text}</strong>
              <span>{target ? `${objective.status.replaceAll('_', ' ')} · select destination` : objective.status.replaceAll('_', ' ')}</span>
            </span>
          </button>
        )
      }) : <p>No {primary ? 'primary' : 'secondary'} objectives exposed.</p>}
    </section>
  )
}

function PanelHead({ title, status, good = false, severity }: { title: string; status: string; good?: boolean; severity?: 'good' | 'caution' | 'critical' | 'unknown' }) {
  const tone = severity ?? (good ? 'good' : undefined)
  return <div className="panel-head"><span>{title}</span><strong className={tone ? `status-${tone}` : ''}>{status}</strong></div>
}

function Metric({ label, value, unit }: { label: string; value: string; unit?: string }) {
  return <div className="metric"><small>{label}</small><strong>{value}{unit && <em>{unit}</em>}</strong></div>
}

function NavStat({ label, value, unit }: { label: string; value: string; unit?: string }) {
  return <div className="nav-stat"><small>{label}</small><strong>{value}{unit && <em>{unit}</em>}</strong></div>
}

function SystemStat({ label, value, unit }: { label: string; value: string; unit?: string }) {
  return <div className="system-stat"><small>{label}</small><strong>{value}{unit && <em>{unit}</em>}</strong></div>
}

function Legend({ color, label, shape }: { color: string; label: string; shape?: string }) {
  return <span className="legend-item"><i className={shape} style={{ background: color, color }} />{label}</span>
}

function EmptyState({ children }: { children: string }) {
  return <div className="empty-state">{children}</div>
}

function connectionView(snapshot: Snapshot | null, transport: string) {
  if (!snapshot) return { state: 'offline', label: 'Connecting', detail: 'waiting for companion' }
  if (snapshot.connection.mode === 'fixture') return { state: 'fixture', label: 'Fixture', detail: 'captured JH-7 session' }
  if (transport !== 'streaming') return { state: 'stale', label: 'Reconnecting', detail: 'holding last good snapshot' }
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

function format(value: number | undefined, digits = 0) {
  if (value === undefined || !Number.isFinite(value)) return '—'
  return digits === 0 ? integer.format(value) : value.toFixed(digits)
}

function signed(value: number | undefined) {
  if (value === undefined || !Number.isFinite(value)) return '—'
  return `${value > 0 ? '+' : ''}${decimal.format(value)}`
}

function signedKg(value: number | undefined) {
  if (value === undefined || !Number.isFinite(value)) return '—'
  return `${value > 0 ? '+' : ''}${integer.format(value)} kg`
}

function heading(value: number | undefined) {
  if (value === undefined || !Number.isFinite(value)) return '—'
  const rounded = Math.round(value) % 360
  return String(rounded === 0 ? 360 : rounded).padStart(3, '0')
}

function duration(seconds: number | undefined) {
  if (seconds === undefined || !Number.isFinite(seconds)) return '—'
  const rounded = Math.max(0, Math.round(seconds))
  return `${String(Math.floor(rounded / 60)).padStart(2, '0')}:${String(rounded % 60).padStart(2, '0')}`
}

function feedTime(missionSeconds: number | undefined, addedAt: string) {
  if (missionSeconds !== undefined) {
    const rounded = Math.max(0, Math.round(missionSeconds))
    return `T+${String(Math.floor(rounded / 60)).padStart(2, '0')}:${String(rounded % 60).padStart(2, '0')}`
  }
  return new Date(addedAt).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}

export default App
