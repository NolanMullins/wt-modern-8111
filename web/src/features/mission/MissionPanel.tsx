import { missionTarget, type SelectedTarget } from '../../navigation'
import { PanelHead } from '../../shared/presentation'
import type { Snapshot } from '../../types'

export function MissionPanel({
  snapshot,
  onSelectTarget,
}: {
  snapshot: Snapshot | null
  onSelectTarget: (target: SelectedTarget) => void
}) {
  const objectives = distinctObjectives(snapshot?.mission.objectives ?? [])
  const primary = objectives.filter((objective) => objective.primary)
  const secondary = objectives.filter((objective) => !objective.primary)
  return (
    <section className="panel mission-panel">
      <PanelHead
        title="Active mission"
        status={snapshot?.mission.status || 'Unavailable'}
        good={snapshot?.mission.status === 'running'}
      />
      <div className="mission-body">
        <ObjectiveGroup
          title="Primary objectives"
          objectives={primary}
          snapshot={snapshot}
          onSelectTarget={onSelectTarget}
          primary
        />
        <ObjectiveGroup
          title="Secondary objectives"
          objectives={secondary}
          snapshot={snapshot}
          onSelectTarget={onSelectTarget}
        />
      </div>
    </section>
  )
}

function distinctObjectives(objectives: Snapshot['mission']['objectives']) {
  const distinct = new Map<string, Snapshot['mission']['objectives'][number]>()
  for (const objective of objectives) {
    const key = `${objective.primary}:${objective.text}`
    const existing = distinct.get(key)
    if (!existing || existing.status === 'undefined') distinct.set(key, objective)
  }
  return [...distinct.values()]
}

function ObjectiveGroup({
  title,
  objectives,
  snapshot,
  onSelectTarget,
  primary = false,
}: {
  title: string
  objectives: Snapshot['mission']['objectives']
  snapshot: Snapshot | null
  onSelectTarget: (target: SelectedTarget) => void
  primary?: boolean
}) {
  return (
    <section className="objective-group">
      <h3>{title}</h3>
      {objectives.length
        ? objectives.map((objective, index) => {
            const target = snapshot ? missionTarget(snapshot, objective) : undefined
            return (
              <button
                className="objective"
                disabled={!target}
                key={`${objective.text}-${index}`}
                onClick={() => target && onSelectTarget(target)}
                title={target
                  ? 'Set as active destination'
                  : 'No map position is exposed for this objective'}
                type="button"
              >
                <i className={primary ? 'primary' : ''}>{primary ? 'P' : 'S'}</i>
                <span className="objective-copy">
                  <strong>{objective.text}</strong>
                  <span>
                    {target
                      ? `${objective.status.replaceAll('_', ' ')} · select destination`
                      : objective.status.replaceAll('_', ' ')}
                  </span>
                </span>
              </button>
            )
          })
        : <p>No {primary ? 'primary' : 'secondary'} objectives exposed.</p>}
    </section>
  )
}
