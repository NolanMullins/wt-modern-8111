import { format, heading, signed } from '../../shared/formatters'
import { PanelHead, Readout } from '../../shared/presentation'
import type { Snapshot } from '../../types'

export function FlightPanel({ snapshot }: { snapshot: Snapshot | null }) {
  const flight = snapshot?.flight
  return (
    <section className="panel flight-panel">
      <PanelHead
        title="Primary flight"
        status={flight?.iasKmh !== undefined ? 'Valid' : 'Unavailable'}
        good={flight?.iasKmh !== undefined}
      />
      <div className="metric-grid">
        <Readout className="metric" label="IAS" value={format(flight?.iasKmh)} unit="km/h" />
        <Readout className="metric" label="TAS" value={format(flight?.tasKmh)} unit="km/h" />
        <Readout className="metric" label="Altitude" value={format(flight?.altitudeM)} unit="m" />
        <Readout className="metric" label="Mach" value={format(flight?.mach, 2)} />
        <Readout className="metric" label="Heading" value={heading(flight?.headingDeg)} unit="deg" />
        <Readout className="metric" label="Vertical speed" value={signed(flight?.verticalSpeedMps)} unit="m/s" />
        <Readout className="metric" label="AoA" value={format(flight?.aoaDeg, 1)} unit="deg" />
        <Readout className="metric" label="Load" value={format(flight?.gLoad, 2)} unit="g" />
      </div>
    </section>
  )
}
