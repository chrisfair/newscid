import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import ImbalanceBar from './ImbalanceBar.vue'

describe('ImbalanceBar.vue', () => {
  const dummyReport = {
    fen: 'rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1',
    white: {
      materialPoints: 39,
      hasBishopPair: true,
      pawnStructure: { doubledPawns: 0, isolatedPawns: 0, passedPawns: 0, maxPassedRank: 0 },
      kingSafety: 100,
      controlledTerritory: 0,
      openFilesControlled: 0,
    },
    black: {
      materialPoints: 39,
      hasBishopPair: true,
      pawnStructure: { doubledPawns: 0, isolatedPawns: 0, passedPawns: 0, maxPassedRank: 0 },
      kingSafety: 100,
      controlledTerritory: 0,
      openFilesControlled: 0,
    },
    materialDelta: 0,
    positionalSummary: 'Material is equal; dynamic play revolves around space and activity.',
  }

  it('renders report with material delta and bishop pair', () => {
    const wrapper = mount(ImbalanceBar, {
      props: {
        report: dummyReport as any,
      },
    })
    expect(wrapper.text()).toContain('Equal')
    expect(wrapper.text()).toContain('2B Pair')
    expect(wrapper.text()).toContain('King Shelter')
    expect(wrapper.text()).toContain('Material is equal')
  })
})
