import { format } from '../../shared/formatters'
import { PanelHead, Readout } from '../../shared/presentation'
import type { Snapshot } from '../../types'

export function GroundSystemsPanel({ snapshot }: { snapshot: Snapshot | null }) {
  const ground = snapshot?.ground
  const connected = snapshot?.connection.state === 'live' || snapshot?.connection.state === 'degraded'
  const crew = ground?.crewCurrent !== undefined && ground.crewTotal !== undefined
    ? `${format(ground.crewCurrent)}/${format(ground.crewTotal)} crew operational`
    : 'Crew status unavailable'
  const critical = ground?.engineBroken === 1 ||
    ground?.driverState === 2 ||
    ground?.gunnerState === 2
  return (
    <section className="panel systems-panel ground-systems-panel">
      <PanelHead
        title="Ground combat"
        status={critical ? 'Damage reported' : connected ? 'Ground RB' : 'Unavailable'}
        severity={critical ? 'critical' : connected ? 'good' : 'unknown'}
      />
      <div className="systems-body">
        <div className="engine-summary ground-summary">
          <strong>{crew}</strong>
          <span>
            {ground?.ammo !== undefined
              ? `${format(ground.ammo)} first-stage rounds`
              : 'Vehicle telemetry varies by ground vehicle'}
          </span>
          <span>Use exposed contacts and team callouts for navigation</span>
        </div>
        <div className="system-stats">
          <Readout className="system-stat" label="Ammo" value={format(ground?.ammo)} />
          <Readout
            className="system-stat"
            label="Stabilizer"
            value={equipmentLabel(ground?.stabilizer)}
          />
          <Readout className="system-stat" label="LWS" value={equipmentLabel(ground?.lws)} />
          <Readout className="system-stat" label="IRCM" value={equipmentLabel(ground?.ircm)} />
        </div>
      </div>
    </section>
  )
}

function equipmentLabel(value: number | undefined) {
  if (value === undefined) return '—'
  return value > 0 ? 'Ready' : 'Off'
}
