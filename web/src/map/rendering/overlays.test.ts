import { describe, expect, it } from 'vitest'
import type { AllyMark } from '../../types'
import { allyMarkGridBounds, allyMarkPresentation } from './overlays'

const mark: AllyMark = {
  key: 'chat-1',
  kind: 'guide',
  subject: 'sender',
  sender: 'ALLY',
  message: 'Guide on me!',
  grid: 'C5',
  altitudeM: 600,
  located: true,
  createdAt: '2026-01-01T00:00:00Z',
  expiresAt: '2026-01-01T00:01:00Z',
}

describe('allyMarkPresentation', () => {
  it('labels sender locations independently from target pings', () => {
    expect(allyMarkPresentation(mark).label).toBe('ALLY · ALLY · C5 · 600 M')
    expect(allyMarkPresentation({ ...mark, subject: 'target' }).label)
      .toBe('PING · ALLY · C5')
    expect(allyMarkPresentation({ ...mark, kind: 'cover' }).label)
      .toBe('HELP · ALLY · C5 · 600 M')
  })

  it('represents reported coordinates as a grid area', () => {
    expect(allyMarkGridBounds(mark, {
      mapMin: [-65_536, -65_536],
      mapMax: [65_536, 65_536],
      gridSteps: [13_100, 13_100],
    })).toEqual({
      x: 0.3997802734375,
      y: 0.19989013671875,
      width: 0.099945068359375,
      height: 0.099945068359375,
    })
  })
})
