import { describe, expect, it } from 'vitest'
import { gridRowLabel, visibleGridCells } from './frame'

describe('War Thunder map grid', () => {
  it('uses lettered rows like War Thunder radio coordinates', () => {
    expect(gridRowLabel(0)).toBe('A')
    expect(gridRowLabel(25)).toBe('Z')
    expect(gridRowLabel(26)).toBe('AA')
  })

  it('anchors visible cells to map_min like the built-in 8111 page', () => {
    expect(visibleGridCells(-65_536, 65_536, 13_100).slice(0, 2))
      .toEqual([
        { index: 0, boundary: -65_536, center: -58_986 },
        { index: 1, boundary: -52_436, center: -45_886 },
      ])
  })
})
