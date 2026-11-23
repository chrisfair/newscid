import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import BlunderVaultModal from './BlunderVaultModal.vue'
import Chessboard from './Chessboard.vue'
import * as AppBindings from '../../wailsjs/go/main/App'

vi.mock('../../wailsjs/go/main/App', () => ({
  GetDueBlunderPuzzles: vi.fn(),
  GetAllBlunderPuzzles: vi.fn(),
  RecordBlunderReview: vi.fn(),
}))

const samplePuzzles = [
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
    explanation: 'Control the center immediately.',
    repetitions: 0,
    intervalDays: 1,
    easeFactor: 2.5,
    status: 'due',
  },
  {
    id: 'puzzle-2',
    fen: 'rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq - 0 1',
    playerName: 'Hero',
    userColor: 'black',
    playedMove: 'f7f6',
    bestMove: 'c7c5',
    deltaCp: -140,
    event: 'Rapid Open',
    date: '2026.05.02',
    repetitions: 0,
    intervalDays: 1,
    easeFactor: 2.5,
    status: 'due',
  },
]

describe('BlunderVaultModal.vue', () => {
  const dummyTheme = {
    id: 'classic-wood',
    name: 'Classic Wood',
    light: '#f0d9b5',
    dark: '#b58863',
    highlight: '#cdd26a',
  }

  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(AppBindings.GetDueBlunderPuzzles).mockResolvedValue([...samplePuzzles] as any)
    vi.mocked(AppBindings.GetAllBlunderPuzzles).mockResolvedValue([])
    vi.mocked(AppBindings.RecordBlunderReview).mockResolvedValue({
      id: 'puzzle-1',
      repetitions: 1,
      intervalDays: 1,
    } as any)
  })

  it('handles wrong move, correct move, SM-2 rating, advancing to next puzzle, and workout completion', async () => {
    const wrapper = mount(BlunderVaultModal, {
      props: {
        boardTheme: dummyTheme,
        pieceSet: 'cburnett',
      },
    })
    await flushPromises()

    expect(wrapper.text()).toContain('Personal Blunder Vault')
    expect(wrapper.text()).toContain('Blitz Cup')
    expect(wrapper.text()).toContain('f2f3')

    // Wrong move shows feedback
    const board = wrapper.findComponent(Chessboard)
    board.vm.$emit('move', 'a2a3')
    await wrapper.vm.$nextTick()
    expect(wrapper.text()).toContain('Not the best move')

    // Correct move reveals solution and SM-2 buttons
    board.vm.$emit('move', 'e2e4')
    await wrapper.vm.$nextTick()
    expect(wrapper.text()).toContain('Correct! You found the master move: e2e4')
    expect(wrapper.text()).toContain('Control the center immediately.')

    // Rate puzzle 1 -> advances to puzzle 2
    const goodBtn = wrapper.findAll('button').find((b) => b.text().includes('Good (6d)'))
    await goodBtn!.trigger('click')
    await flushPromises()
    expect(AppBindings.RecordBlunderReview).toHaveBeenCalledWith('puzzle-1', 4)
    expect(wrapper.text()).toContain('Rapid Open')

    // Reveal and rate puzzle 2 -> completes deck
    const revealBtn = wrapper.findAll('button').find((b) => b.text().includes('Reveal Solution'))
    await revealBtn!.trigger('click')
    const easyBtn = wrapper.findAll('button').find((b) => b.text().includes('Easy (14d)'))
    await easyBtn!.trigger('click')
    await flushPromises()

    expect(AppBindings.RecordBlunderReview).toHaveBeenCalledWith('puzzle-2', 5)
    expect(wrapper.text()).toContain('Workout Complete!')

    const doneBtn = wrapper.findAll('button').find((b) => b.text().trim() === 'Done')
    await doneBtn!.trigger('click')
    expect(wrapper.emitted('close')).toBeTruthy()
  })

  it('falls back to GetAllBlunderPuzzles and renders empty vault state when none exist', async () => {
    vi.mocked(AppBindings.GetDueBlunderPuzzles).mockResolvedValueOnce([])
    vi.mocked(AppBindings.GetAllBlunderPuzzles).mockResolvedValueOnce([])

    const wrapper = mount(BlunderVaultModal, {
      props: {
        boardTheme: dummyTheme,
        pieceSet: 'cburnett',
      },
    })
    await flushPromises()

    expect(AppBindings.GetAllBlunderPuzzles).toHaveBeenCalled()
    expect(wrapper.text()).toContain('No Blunders in Vault Yet!')
  })
})
