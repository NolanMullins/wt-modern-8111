import { FeedPanel } from './features/feed/FeedPanel'
import { FlightPanel } from './features/flight/FlightPanel'
import { DashboardHeader } from './features/header/DashboardHeader'
import { MapPanel } from './features/map/MapPanel'
import { MissionPanel } from './features/mission/MissionPanel'
import { NavigationPanel } from './features/navigation/NavigationPanel'
import { useSelectedNavigation } from './features/navigation/useSelectedNavigation'
import { SystemsPanel } from './features/systems/SystemsPanel'
import { useFuelEstimate } from './useFuelEstimate'
import { useTelemetry } from './useTelemetry'

function App() {
  const { snapshot, transport } = useTelemetry()
  const fuelEstimate = useFuelEstimate(snapshot)
  const {
    navigation,
    selectedTarget,
    selectTarget,
    clearTarget,
    radioRTB,
  } = useSelectedNavigation(snapshot)

  return (
    <main className="dashboard">
      <DashboardHeader snapshot={snapshot} transport={transport} />
      <section className="workspace">
        <FlightPanel snapshot={snapshot} />
        <NavigationPanel
          navigation={navigation}
          radioRTB={radioRTB}
          canClear={selectedTarget !== null}
          onClear={clearTarget}
        />
        <SystemsPanel snapshot={snapshot} fuelEstimate={fuelEstimate} />
        <MapPanel
          snapshot={snapshot}
          navigation={navigation}
          selectedTarget={selectedTarget}
          onSelectTarget={selectTarget}
          fuelEstimate={fuelEstimate}
        />
        <MissionPanel snapshot={snapshot} onSelectTarget={selectTarget} />
        <FeedPanel snapshot={snapshot} />
      </section>
    </main>
  )
}

export default App
