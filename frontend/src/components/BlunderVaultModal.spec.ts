import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import BlunderVaultModal from './BlunderVaultModal.vue'

vi.mock('../../wailsjs/go/main/App', () => ({
  GetDueBlunderPuzzles: vi.fn().mockResolvedValue([
    {
      id: 'puzzle-1',
      fen: 'rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1',
      playerName: 'Hero',
      userColor: 'white',
      playedMove: 'f2f3',
      bestMove: 'e2e4',
      deltaCp: -180,
      event: 'Blitz Cup',
      date: '2026.05.01',
      repetitions: 0,
      intervalDays: 1,
      easeFactor: 2.5,
      status: 'due',
    },
  ]),
  GetAllBlunderPuzzles: vi.fn().mockResolvedValue([]),
  RecordBlunderReview: vi.fn().mockResolvedValue({
    id: 'puzzle-1',
    repetitions: 1,
    intervalDays: 1,
  }),
}))

describe('BlunderVaultModal.vue', () => {
  const dummyTheme = {
    id: 'classic-wood',
    name: 'Classic Wood',
    light: '#f0d9b5',
    dark: '#b58863',
    highlight: '#cdd26a',
  }

  it('renders modal with active puzzle and allows solution reveal', async () => {
    const wrapper = mount(BlunderVaultModal, {
      props: {
        boardTheme: dummyTheme,
        pieceSet: 'cburnett',
      },
    })

    // wait for mock fetch
    await new Promise((r) => setTimeout(r, 20))
    await wrapper.vm.$nextTick()

    expect(wrapper.text()).toContain('Personal Blunder Vault')
    expect(wrapper.text()).toContain('Blitz Cup')
    expect(wrapper.text()).toContain('f2f3')

    // Reveal Solution
    const revealBtn = wrapper.findAll('button').find((b) => b.text().includes('Reveal Solution'))
    expect(revealBtn).toBeDefined()
    await revealBtn!.trigger('click')

    expect(wrapper.text()).toContain('e2e4')
    expect(wrapper.text()).toContain('Again (1d)')
    expect(wrapper.text()).toContain('Good (6d)')
  })
})
