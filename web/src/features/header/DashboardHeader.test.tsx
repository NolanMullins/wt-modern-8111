import { renderToStaticMarkup } from 'react-dom/server'
import { describe, expect, it } from 'vitest'
import { DashboardHeader } from './DashboardHeader'

describe('DashboardHeader', () => {
  it('shows the companion version', () => {
    const markup = renderToStaticMarkup(
      <DashboardHeader
        snapshot={null}
        battleMode="air"
        transport="connecting"
        appVersion="1.2.3"
      />,
    )

    expect(markup).toContain('v1.2.3')
  })

  it('labels development builds without presenting them as releases', () => {
    const markup = renderToStaticMarkup(
      <DashboardHeader
        snapshot={null}
        battleMode="air"
        transport="connecting"
        appVersion="dev"
      />,
    )

    expect(markup).toContain('development build')
    expect(markup).not.toContain('vdev')
  })

  it('omits an unavailable version', () => {
    const markup = renderToStaticMarkup(
      <DashboardHeader
        snapshot={null}
        battleMode="air"
        transport="connecting"
        appVersion="unknown"
      />,
    )

    expect(markup).not.toContain('unknown')
  })
})
