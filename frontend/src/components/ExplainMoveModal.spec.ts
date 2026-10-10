import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import ExplainMoveModal from './ExplainMoveModal.vue'

vi.mock('../../wailsjs/go/main/App', () => ({
  ExplainMove: vi.fn().mockResolvedValue({
    moveUci: 'e1g1',
    moveSan: 'O-O',
    assessment: 'Positional Improvement',
    summary: 'White castles kingside',
    tacticalMotifs: ['Castling'],
    strategicConcepts: ['King Safety'],
    detailedText: 'White castles kingside, tucking the king into safety and activating the rook.',
  }),
}))

describe('ExplainMoveModal.vue', () => {
  it('renders modal and fetches explanation on mount', async () => {
    const wrapper = mount(ExplainMoveModal, {
      props: {
        beforeFen: 'rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1',
        moveUci: 'e1g1',
        scoreCp: 30,
        bestMove: 'e1g1',
      },
    })

    // wait for mock resolve
    await new Promise((r) => setTimeout(r, 20))
    await wrapper.vm.$nextTick()

    expect(wrapper.text()).toContain('Explain Move')
    expect(wrapper.text()).toContain('White castles kingside')
    expect(wrapper.text()).toContain('Castling')
    expect(wrapper.text()).toContain('King Safety')
  })
})
