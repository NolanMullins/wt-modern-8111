import { useState } from 'react'
import { useBattleMode, vehicleModeForSnapshot } from './battleMode'
import { FeedPanel } from './features/feed/FeedPanel'
import { FlightPanel } from './features/flight/FlightPanel'
import { GroundHeatmapPanel } from './features/ground/GroundHeatmapPanel'
import { GroundMobilityPanel } from './features/ground/GroundMobilityPanel'
import { GroundSystemsPanel } from './features/ground/GroundSystemsPanel'
import { DashboardHeader } from './features/header/DashboardHeader'
import { MapPanel } from './features/map/MapPanel'
import { MissionPanel } from './features/mission/MissionPanel'
import { NavigationPanel } from './features/navigation/NavigationPanel'
import { useSelectedNavigation } from './features/navigation/useSelectedNavigation'
import { SystemsPanel } from './features/systems/SystemsPanel'
import { useHeatmapImage } from './map/useHeatmapImage'
import { useFuelEstimate } from './useFuelEstimate'
import { useTelemetry } from './useTelemetry'

function App() {
  const { snapshot, transport, error } = useTelemetry()
  const fuelEstimate = useFuelEstimate(snapshot)
  const battleMode = useBattleMode(snapshot)
  const vehicleMode = vehicleModeForSnapshot(snapshot)
  const [heatmapEnabled, setHeatmapEnabled] = useState(false)
  const heatmap = useHeatmapImage(snapshot, battleMode === 'ground' && heatmapEnabled)
  const {
    navigation,
    selectedTarget,
    selectTarget,
    clearTarget,
    radioRTB,
  } = useSelectedNavigation(snapshot)

  return (
    <main className="dashboard" data-battle-mode={battleMode}>
      <DashboardHeader
        snapshot={snapshot}
        battleMode={battleMode}
        transport={transport}
        error={error}
      />
      <section className="workspace">
        {vehicleMode === 'ground'
          ? <GroundMobilityPanel snapshot={snapshot} />
          : <FlightPanel snapshot={snapshot} />}
        <NavigationPanel
          navigation={navigation}
          radioRTB={radioRTB}
          canClear={selectedTarget !== null}
          onClear={clearTarget}
        />
        {vehicleMode === 'ground'
          ? <GroundSystemsPanel snapshot={snapshot} />
          : <SystemsPanel snapshot={snapshot} fuelEstimate={fuelEstimate} />}
        <MapPanel
          snapshot={snapshot}
          battleMode={battleMode}
          heatmap={heatmap}
          navigation={navigation}
          selectedTarget={selectedTarget}
          onSelectTarget={selectTarget}
          fuelEstimate={fuelEstimate}
        />
        {battleMode === 'ground'
          ? (
              <GroundHeatmapPanel
                enabled={heatmapEnabled}
                heatmap={heatmap}
                onToggle={() => setHeatmapEnabled((enabled) => !enabled)}
              />
            )
          : <MissionPanel snapshot={snapshot} onSelectTarget={selectTarget} />}
        <FeedPanel snapshot={snapshot} />
      </section>
    </main>
  )
}

export default App
