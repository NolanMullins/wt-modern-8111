import { PanelHead } from '../../shared/presentation'
import type { HeatmapState } from '../../map/useHeatmapImage'

export function GroundHeatmapPanel({
  enabled,
  heatmap,
  onToggle,
}: {
  enabled: boolean
  heatmap: HeatmapState
  onToggle: () => void
}) {
  return (
    <section className="panel mission-panel ground-heatmap-panel">
      <PanelHead
        title="Historical map intelligence"
        status={heatmapStatus(enabled, heatmap)}
        good={heatmap.status === 'ready'}
      />
      <div className="heatmap-panel-body">
        <div className="heatmap-panel-copy">
          <small>Ground Realistic battles</small>
          <strong>{heatmap.mapName ?? 'Community combat history'}</strong>
          <p>
            Red highlights positions where players secure more kills. Blue highlights
            positions where players die more often. Brighter areas have more samples.
            Ground RB always uses the tank-map view so the historical layer stays aligned,
            including while flying CAS.
          </p>
        </div>
        <div className="heatmap-scale" aria-label="Historical heatmap color key">
          <span className="heatmap-death">Death-heavy</span>
          <span>Balanced</span>
          <span className="heatmap-kill">Kill-heavy</span>
        </div>
        <button
          className="heatmap-panel-toggle"
          disabled={heatmap.status === 'loading'}
          onClick={onToggle}
          type="button"
        >
          {enabled ? 'Hide historical overlay' : 'Load historical overlay'}
        </button>
        <a
          href="https://thunder.nanachi.party/about"
          rel="noreferrer"
          target="_blank"
        >
          Data rendered by War Thunder Heatmaps
        </a>
      </div>
    </section>
  )
}

function heatmapStatus(enabled: boolean, heatmap: HeatmapState) {
  if (!enabled) return 'Off'
  if (heatmap.status === 'loading') return 'Loading'
  if (heatmap.status === 'ready') return 'Overlay active'
  if (heatmap.status === 'unavailable') return 'No map data'
  return 'Service unavailable'
}
