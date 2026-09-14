import { renderToStaticMarkup } from 'react-dom/server'
import { describe, expect, it } from 'vitest'
import type { AllyMark } from '../../types'
import {
  activeRadioCallouts,
  calloutOpacity,
} from './radioCalloutModel'
import { RadioCallouts } from './RadioCallouts'

const baseMark: AllyMark = {
  key: 'chat-1',
  kind: 'guide',
  subject: 'sender',
  source: 'chat',
  precision: 'grid',
  sender: 'ALLY',
  message: 'Guide on me! [C5, alt. 600 m]',
  grid: 'C5',
  altitudeM: 600,
  x: 0.5,
  y: 0.5,
  located: true,
  createdAt: '2099-01-01T00:00:00Z',
  expiresAt: '2099-01-01T00:01:00Z',
}

describe('RadioCallouts', () => {
  it('distinguishes sender positions from target pings', () => {
    const markup = renderToStaticMarkup(
      <RadioCallouts
        marks={[
          baseMark,
          {
            ...baseMark,
            key: 'chat-2',
            kind: 'attention',
            subject: 'target',
            message: 'Attention to the map! [B4]',
            grid: 'B4',
            altitudeM: undefined,
          },
        ]}
      />,
    )

    expect(markup).toContain('ALLY POSITION')
    expect(markup).toContain('C5 · 600 m')
    expect(markup).toContain('MAP PING')
    expect(markup).toContain('B4')
  })

  it('labels cover calls as sender support requests', () => {
    const markup = renderToStaticMarkup(
      <RadioCallouts marks={[{ ...baseMark, kind: 'cover' }]} />,
    )

    expect(markup).toContain('SUPPORT REQUEST')
    expect(markup).not.toContain('MAP PING')
  })

  it('expires and fades callouts from the client clock', () => {
    const expiry = new Date(baseMark.expiresAt).getTime()

    expect(activeRadioCallouts([baseMark], expiry + 1)).toEqual([])
    expect(calloutOpacity(baseMark, expiry - 2_500)).toBe(0.5)
  })

  it('shows a known grid even before map metadata locates it', () => {
    const markup = renderToStaticMarkup(
      <RadioCallouts marks={[{ ...baseMark, located: false, x: undefined, y: undefined }]} />,
    )

    expect(markup).toContain('C5 · 600 m')
    expect(markup).not.toContain('Position unavailable')
  })
})
