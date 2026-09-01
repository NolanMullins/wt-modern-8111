import { TacticalMap } from '../../TacticalMap'
import type { NavigationSolution, SelectedTarget } from '../../navigation'
import { format, signedKg } from '../../shared/formatters'
import { Legend } from '../../shared/presentation'
import type { Snapshot } from '../../types'
import type { FuelEstimate } from '../../useFuelEstimate'

export function MapPanel({
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
      {calibrated && <em>{format(estimate.burnKgPerHour)} kg/h at 100% · no reserve</em>}
    </div>
  )
}
