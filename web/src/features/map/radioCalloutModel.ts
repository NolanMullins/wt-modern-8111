import type { AllyMark } from '../../types'

export function activeRadioCallouts(marks: AllyMark[], now: number) {
  return [...marks]
    .filter((mark) => now === 0 || new Date(mark.expiresAt).getTime() > now)
    .sort((left, right) =>
      new Date(right.createdAt).getTime() - new Date(left.createdAt).getTime(),
    )
    .slice(0, 3)
}

export function calloutOpacity(mark: AllyMark, now: number) {
  if (now === 0) return 1
  const remaining = new Date(mark.expiresAt).getTime() - now
  return Math.min(1, Math.max(0, remaining / 5_000))
}
