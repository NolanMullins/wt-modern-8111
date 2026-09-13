import { describe, expect, it } from 'vitest'
import { gridColumnLabel, visibleGridCells } from './frame'

describe('gridColumnLabel', () => {
  it('uses lettered columns like War Thunder radio coordinates', () => {
    expect(gridColumnLabel(0)).toBe('A')
    expect(gridColumnLabel(25)).toBe('Z')
    expect(gridColumnLabel(26)).toBe('AA')
  })

  it('anchors visible cells to the map grid origin', () => {
    expect(visibleGridCells(-65_536, 65_536, -59_904, 13_100, 1).slice(0, 2))
      .toEqual([
        { index: 0, boundary: -59_904, center: -53_354 },
        { index: 1, boundary: -46_804, center: -40_254 },
      ])
    expect(visibleGridCells(-65_536, 65_536, 59_904, 13_100, -1).slice(0, 2))
      .toEqual([
        { index: 0, boundary: 59_904, center: 53_354 },
        { index: 1, boundary: 46_804, center: 40_254 },
      ])
  })
})
