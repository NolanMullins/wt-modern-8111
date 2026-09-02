import type { NavigationSolution } from '../../navigation'
import { duration, heading, oneDecimal } from '../../shared/formatters'
import { PanelHead, Readout } from '../../shared/presentation'

export function NavigationPanel({
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
          {canClear && (
            <button className="clear-target" type="button" onClick={onClear}>Clear</button>
          )}
        </div>
        <Readout
          className="nav-stat"
          label="Bearing"
          value={navigation ? heading(navigation.bearingDeg) : '—'}
          unit="deg"
        />
        <Readout
          className="nav-stat"
          label="Range"
          value={oneDecimal(navigation?.rangeKm)}
          unit="km"
        />
        <Readout className="nav-stat" label="ETA" value={duration(navigation?.etaSeconds)} />
      </div>
    </section>
  )
}
