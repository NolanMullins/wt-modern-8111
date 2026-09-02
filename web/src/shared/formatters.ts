const integer = new Intl.NumberFormat('en-US', { maximumFractionDigits: 0 })
const decimal = new Intl.NumberFormat('en-US', {
  minimumFractionDigits: 1,
  maximumFractionDigits: 1,
})

export function format(value: number | undefined, digits = 0) {
  if (value === undefined || !Number.isFinite(value)) return '—'
  return digits === 0 ? integer.format(value) : value.toFixed(digits)
}

export function oneDecimal(value: number | undefined) {
  return value === undefined || !Number.isFinite(value) ? '—' : decimal.format(value)
}

export function signed(value: number | undefined) {
  if (value === undefined || !Number.isFinite(value)) return '—'
  return `${value > 0 ? '+' : ''}${decimal.format(value)}`
}

export function signedKg(value: number | undefined) {
  if (value === undefined || !Number.isFinite(value)) return '—'
  return `${value > 0 ? '+' : ''}${integer.format(value)} kg`
}

export function heading(value: number | undefined) {
  if (value === undefined || !Number.isFinite(value)) return '—'
  const rounded = Math.round(value) % 360
  return String(rounded === 0 ? 360 : rounded).padStart(3, '0')
}

export function duration(seconds: number | undefined) {
  if (seconds === undefined || !Number.isFinite(seconds)) return '—'
  const rounded = Math.max(0, Math.round(seconds))
  return `${String(Math.floor(rounded / 60)).padStart(2, '0')}:${String(rounded % 60).padStart(2, '0')}`
}
