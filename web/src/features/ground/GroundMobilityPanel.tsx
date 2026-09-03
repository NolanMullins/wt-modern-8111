import { format, heading } from '../../shared/formatters'
import { PanelHead, Readout } from '../../shared/presentation'
import type { Snapshot } from '../../types'

export function GroundMobilityPanel({ snapshot }: { snapshot: Snapshot | null }) {
  const ground = snapshot?.ground
  const available = ground?.speedKmh !== undefined || ground?.headingDeg !== undefined
  return (
    <section className="panel flight-panel ground-mobility-panel">
      <PanelHead
        title="Ground mobility"
        status={available ? 'Telemetry valid' : 'Map-first mode'}
        good={available}
      />
      <div className="metric-grid">
        <Readout className="metric" label="Speed" value={format(ground?.speedKmh)} unit="km/h" />
        <Readout className="metric" label="Heading" value={heading(ground?.headingDeg)} unit="deg" />
        <Readout className="metric" label="Engine" value={format(ground?.engineRpm)} unit="rpm" />
        <Readout className="metric" label="Gear" value={format(ground?.gear)} />
        <Readout className="metric" label="Cruise" value={format(ground?.cruiseControl)} unit="%" />
        <Readout
          className="metric"
          label="Crew"
          value={crewLabel(ground?.crewCurrent, ground?.crewTotal)}
        />
        <Readout className="metric" label="Driver" value={crewStateLabel(ground?.driverState)} />
        <Readout className="metric" label="Gunner" value={crewStateLabel(ground?.gunnerState)} />
      </div>
    </section>
  )
}

function crewLabel(current: number | undefined, total: number | undefined) {
  return current === undefined || total === undefined
    ? '—'
    : `${format(current)}/${format(total)}`
}

function crewStateLabel(state: number | undefined) {
  if (state === undefined) return '—'
  if (state === 0) return 'Ready'
  if (state === 1) return 'Wounded'
  return 'Out'
}
