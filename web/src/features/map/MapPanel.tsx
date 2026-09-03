import { TacticalMap } from '../../TacticalMap'
import { type BattleMode, vehicleModeForSnapshot } from '../../battleMode'
import type { HeatmapState } from '../../map/useHeatmapImage'
import type { NavigationSolution, SelectedTarget } from '../../navigation'
import { format, signedKg } from '../../shared/formatters'
import { Legend } from '../../shared/presentation'
import type { Snapshot } from '../../types'
import type { FuelEstimate } from '../../useFuelEstimate'

export function MapPanel({
  snapshot,
  battleMode,
  heatmap,
  showEnemyFiring,
  showFriendlyLosses,
  navigation,
  selectedTarget,
  onSelectTarget,
  fuelEstimate,
}: {
  snapshot: Snapshot | null
  battleMode: BattleMode
  heatmap: HeatmapState
  showEnemyFiring: boolean
  showFriendlyLosses: boolean
  navigation?: NavigationSolution
  selectedTarget: SelectedTarget | null
  onSelectTarget: (target: SelectedTarget) => void
  fuelEstimate: FuelEstimate
}) {
  const counts = snapshot?.map.counts
  const groundMode = battleMode === 'ground'
  const groundVehicle = vehicleModeForSnapshot(snapshot) === 'ground'
  return (
    <section className="panel map-panel">
      <TacticalMap
        snapshot={snapshot}
        navigation={navigation}
        selectedTarget={selectedTarget}
        onSelectTarget={onSelectTarget}
        heatmapImages={heatmap.status === 'ready'
          ? [
              showFriendlyLosses ? heatmap.victimImage : undefined,
              showEnemyFiring ? heatmap.firingImage : undefined,
            ].filter((image): image is CanvasImageSource => image !== undefined)
          : undefined}
        mapImageOverride={heatmap.status === 'ready' ? heatmap.baseImage : undefined}
        groundMap={groundMode}
      />
      <div className="map-overlay">
        <div className="map-name">
          <small>Full mission frame</small>
          <strong>Tactical map</strong>
        </div>
        {!groundVehicle && <BingoFuelHUD estimate={fuelEstimate} />}
        <div className="map-mode">
          {snapshot?.connection.mode === 'fixture' ? 'Captured map fixture' : 'Drag to pan · scroll to zoom'}
          <br />
          {snapshot?.map.valid ? `${counts?.total ?? 0} exposed objects` : 'Map unavailable'}
        </div>
      </div>
      {groundMode
        ? (
            <div className="map-legend">
              <Legend color="#fff" label={groundVehicle ? 'Own vehicle' : 'Ownship CAS'} />
              <Legend color="#174dff" label={`${counts?.friendlyGround ?? 0} friendly ground`} />
              <Legend color="#fa0c00" label={`${counts?.hostileGround ?? 0} hostile ground`} />
              <Legend color="#f4b13d" label={`${counts?.captureZone ?? 0} capture zones`} shape="target" />
              <Legend color="#f00c00" label={`${counts?.hostileAir ?? 0} air contacts`} shape="air" />
              {heatmap.status === 'ready' && (
                <>
                  {showEnemyFiring && (
                    <Legend color="#ff4e00" label="enemy firing positions" shape="heat" />
                  )}
                  {showFriendlyLosses && (
                    <Legend color="#00a6ff" label="friendly loss positions" shape="heat" />
                  )}
                </>
              )}
            </div>
          )
        : (
            <div className="map-legend">
              <Legend color="#fff" label="Ownship" />
              <Legend color="#f00c00" label={`${counts?.hostileAir ?? 0} hostile air`} shape="air" />
              <Legend color="#fa0c00" label={`${counts?.ground ?? 0} ground`} />
              <Legend color="#fa0c00" label={`${counts?.airDefense ?? 0} air defense`} shape="airdef" />
              <Legend color="#fa0c00" label={`${counts?.strikePoint ?? 0} strike points`} shape="target" />
              <Legend color="#174dff" label={`${counts?.airfield ?? 0} airfield`} shape="airfield" />
            </div>
          )}
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
