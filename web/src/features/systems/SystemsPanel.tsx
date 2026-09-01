import { format, signedKg } from '../../shared/formatters'
import { PanelHead, Readout } from '../../shared/presentation'
import type { Snapshot } from '../../types'
import type { FuelEstimate } from '../../useFuelEstimate'

export function SystemsPanel({
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
    item.status === reference?.status &&
    item.throttlePercent === reference?.throttlePercent &&
    item.rpm === reference?.rpm &&
    item.oilTempC === reference?.oilTempC &&
    item.thrustKgf === reference?.thrustKgf,
  )
  const configuration = [systems?.gearPercent, systems?.flapsPercent, systems?.airbrakePercent]
  const configurationKnown = configuration.every((value) => value !== undefined)
  const clean = configurationKnown && configuration.every((value) => value === 0)
  const configurationLabel = configurationKnown
    ? (clean ? 'clean configuration' : 'configuration extended')
    : 'configuration unavailable'
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
            {reference?.throttlePercent !== undefined
              ? `Throttle ${format(reference.throttlePercent)}%`
              : 'Throttle unavailable'} · {configurationLabel}
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
          <Readout className="system-stat" label="RPM" value={format(reference?.rpm)} />
          <Readout className="system-stat" label="Oil" value={format(reference?.oilTempC)} unit="C" />
          <Readout
            className="system-stat"
            label="Thrust"
            value={format(reference?.thrustKgf)}
            unit={linked ? `kgf × ${engines.length}` : 'kgf'}
          />
          <Readout
            className="system-stat"
            label="Fuel"
            value={format(systems?.fuelKg)}
            unit={systems?.fuelPercent !== undefined
              ? `kg · ${format(systems.fuelPercent)}%`
              : 'kg'}
          />
        </div>
      </div>
    </section>
  )
}
