import { PanelHead } from '../../shared/presentation'
import type { HeatmapState } from '../../map/useHeatmapImage'
import type { TeamPreference, usePlayerTeam } from '../../map/usePlayerTeam'

export function GroundHeatmapPanel({
  enabled,
  heatmap,
  playerTeam,
  showEnemyFiring,
  showFriendlyLosses,
  onToggle,
  onShowEnemyFiring,
  onShowFriendlyLosses,
}: {
  enabled: boolean
  heatmap: HeatmapState
  playerTeam: ReturnType<typeof usePlayerTeam>
  showEnemyFiring: boolean
  showFriendlyLosses: boolean
  onToggle: () => void
  onShowEnemyFiring: (shown: boolean) => void
  onShowFriendlyLosses: (shown: boolean) => void
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
            Orange marks confirmed firing positions for the historical enemy team.
            Blue marks where that team killed players on your side. A cell is shown only
            when at least five exact replay events support it. Broad interpolation is
            disabled.
          </p>
        </div>
        <label className="heatmap-team-control">
          <span>Your replay side</span>
          <select
            value={playerTeam.preference}
            onChange={(event) =>
              playerTeam.setPreference(event.currentTarget.value as TeamPreference)}
          >
            <option value="auto">
              {playerTeam.detection.status === 'detected'
                ? `Automatic (Team ${playerTeam.detection.team})`
                : 'Automatic detection'}
            </option>
            <option value="1">Team 1 override</option>
            <option value="2">Team 2 override</option>
          </select>
          <small>
            {playerTeam.preference === 'auto'
              ? playerTeam.detection.detail
              : `Manual override active: you are Team ${playerTeam.selectedTeam}`}
          </small>
        </label>
        <div className="heatmap-layer-controls" aria-label="Historical heatmap layers">
          <label>
            <input
              type="checkbox"
              checked={showEnemyFiring}
              onChange={(event) => onShowEnemyFiring(event.currentTarget.checked)}
            />
            <i className="heatmap-firing-swatch" />
            Enemy firing positions
          </label>
          <label>
            <input
              type="checkbox"
              checked={showFriendlyLosses}
              onChange={(event) => onShowFriendlyLosses(event.currentTarget.checked)}
            />
            <i className="heatmap-victim-swatch" />
            Your team killed here
          </label>
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
  if (heatmap.status === 'waiting-team') return 'Select team'
  if (heatmap.status === 'loading') return 'Loading'
  if (heatmap.status === 'ready') return 'Overlay active'
  if (heatmap.status === 'unavailable') return 'No map data'
  return 'Service unavailable'
}
