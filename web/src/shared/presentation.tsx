import type { ReactNode } from 'react'

export function PanelHead({
  title,
  status,
  good = false,
  severity,
}: {
  title: string
  status: string
  good?: boolean
  severity?: 'good' | 'caution' | 'critical' | 'unknown'
}) {
  const tone = severity ?? (good ? 'good' : undefined)
  return (
    <div className="panel-head">
      <span>{title}</span>
      <strong className={tone ? `status-${tone}` : ''}>{status}</strong>
    </div>
  )
}

export function Readout({
  className,
  label,
  value,
  unit,
}: {
  className: 'metric' | 'nav-stat' | 'system-stat'
  label: string
  value: string
  unit?: string
}) {
  return (
    <div className={className}>
      <small>{label}</small>
      <strong>{value}{unit && <em>{unit}</em>}</strong>
    </div>
  )
}

export function Legend({
  color,
  label,
  shape,
}: {
  color: string
  label: string
  shape?: string
}) {
  return (
    <span className="legend-item">
      <i className={shape} style={{ background: color, color }} />
      {label}
    </span>
  )
}

export function EmptyState({ children }: { children: ReactNode }) {
  return <div className="empty-state">{children}</div>
}
