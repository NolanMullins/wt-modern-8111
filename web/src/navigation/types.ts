import type { Snapshot } from '../types'

export interface SelectedTarget {
  key: string
  name: string
  x: number
  y: number
  generation?: number
  basis: 'map selection' | 'mission objective' | 'game target'
  object?: {
    index: number
    type: string
    icon?: string
    color?: string
  }
}

export type NavigationSolution = NonNullable<Snapshot['navigation']>
