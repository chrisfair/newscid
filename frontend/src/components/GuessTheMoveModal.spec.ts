import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import GuessTheMoveModal from './GuessTheMoveModal.vue'
import * as AppApi from '../../wailsjs/go/main/App'

vi.mock('../../wailsjs/go/main/App', () => ({
  StartGuessTheMove: vi.fn().mockResolvedValue({
    white: 'Morphy, Paul',
    black: 'Duke of Brunswick',
    event: 'Paris Opera',
    date: '1858.11.02',
    result: '1-0',
    userSide: 1,
    currentPlyIdx: 0,
    score: 0,
    correctGuesses: 0,
    totalTurns: 3,
    hintsUsed: 0,
    currentHints: 0,
    moves: [
      { ply: 0, turn: 'white', san: 'e4', uci: 'e2e4', fen: 'rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1' },
      { ply: 1, turn: 'black', san: 'e5', uci: 'e7e5', fen: 'rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1' },
      { ply: 2, turn: 'white', san: 'Nf3', uci: 'g1f3', fen: 'rnbqkbnr/pppp1ppp/8/4p3/4P3/8/PPPP1PPP/RNBQKBNR w KQkq - 0 2' },
    ],
    isFinished: false
  }),
  SubmitGuessTheMove: vi.fn().mockImplementation((guess: string) => {
    if (guess === 'e4' || guess === 'e2e4') {
      return Promise.resolve({
        isMatch: true,
        actualMoveSAN: 'e4',
        actualMoveUCI: 'e2e4',
        pointsEarned: 10,
        totalScore: 10,
        currentFEN: 'rnbqkbnr/pppp1ppp/8/4p3/4P3/5N2/PPPP1PPP/RNBQKB1R b KQkq - 1 2',
        opponentReplySAN: 'e5',
        opponentReplyUCI: 'e7e5',
        isFinished: false
      })
    }
    return Promise.resolve({
      isMatch: false,
      actualMoveSAN: 'e4',
      actualMoveUCI: 'e2e4',
      pointsEarned: 0,
      totalScore: 0,
      currentFEN: 'rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1',
      isFinished: false
    })
  }),
  GetGuessTheMoveHint: vi.fn().mockResolvedValue('Piece type: A Pawn was moved.'),
  SkipGuessTheMove: vi.fn().mockResolvedValue({
    isMatch: false,
    actualMoveSAN: 'e4',
    actualMoveUCI: 'e2e4',
    pointsEarned: 0,
    totalScore: 0,
    currentFEN: 'rnbqkbnr/pppp1ppp/8/4p3/4P3/5N2/PPPP1PPP/RNBQKB1R b KQkq - 1 2',
    opponentReplySAN: 'e5',
    opponentReplyUCI: 'e7e5',
    isFinished: true,
    summary: {
      score: 10,
      maxScore: 30,
      percentage: 33.3,
      accuracyPct: 50.0,
      correctGuesses: 1,
      totalTurns: 2,
      hintsUsed: 0,
      ratingTitle: 'Club Player',
      feedback: 'Good tactical eye.'
    }
  }),
  GetGuessTheMoveState: vi.fn().mockResolvedValue(null)
}))

describe('GuessTheMoveModal.vue', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders modal with title, game header, and starts session on mount', async () => {
    const wrapper = mount(GuessTheMoveModal, {
      props: {
        activeGameId: 1,
        gameHeader: {
          white: 'Morphy, Paul',
          black: 'Duke of Brunswick',
          event: 'Paris Opera',
          date: '1858.11.02',
          result: '1-0'
        }
      }
    })
    await flushPromises()

    expect(wrapper.text()).toContain('Guess the Move')
    expect(wrapper.text()).toContain('Solitaire Chess')
    expect(wrapper.text()).toContain('Morphy, Paul vs Duke of Brunswick')
    expect(AppApi.StartGuessTheMove).toHaveBeenCalledWith(1, 'white')
  })

  it('handles wrong move guess and displays error feedback', async () => {
    const wrapper = mount(GuessTheMoveModal, {
      props: { activeGameId: 1 }
    })
    await flushPromises()

    const input = wrapper.find('input[placeholder*="Enter move"]')
    expect(input.exists()).toBe(true)
    await input.setValue('d4')

    const form = wrapper.find('form')
    await form.trigger('submit.prevent')
    await flushPromises()

    expect(AppApi.SubmitGuessTheMove).toHaveBeenCalledWith('d4')
    expect(wrapper.text()).toContain('Not the played move')
  })

  it('handles correct move guess and awards points', async () => {
    const wrapper = mount(GuessTheMoveModal, {
      props: { activeGameId: 1 }
    })
    await flushPromises()

    const input = wrapper.find('input[placeholder*="Enter move"]')
    await input.setValue('e4')

    const form = wrapper.find('form')
    await form.trigger('submit.prevent')
    await flushPromises()

    expect(AppApi.SubmitGuessTheMove).toHaveBeenCalledWith('e4')
    expect(wrapper.text()).toContain('Correct! e4 was played')
    expect(wrapper.text()).toContain('Opponent replied e5')
  })

  it('fetches progressive hint when Get Hint button is clicked', async () => {
    const wrapper = mount(GuessTheMoveModal, {
      props: { activeGameId: 1 }
    })
    await flushPromises()

    const hintBtn = wrapper.findAll('button').find(b => b.text().includes('Get Hint'))
    expect(hintBtn).toBeDefined()
    await hintBtn!.trigger('click')
    await flushPromises()

    expect(AppApi.GetGuessTheMoveHint).toHaveBeenCalled()
    expect(wrapper.text()).toContain('Piece type: A Pawn was moved')
  })

  it('handles skip move and displays completion summary when finished', async () => {
    const wrapper = mount(GuessTheMoveModal, {
      props: { activeGameId: 1 }
    })
    await flushPromises()

    const skipBtn = wrapper.findAll('button').find(b => b.text().includes('Reveal & Skip'))
    expect(skipBtn).toBeDefined()
    await skipBtn!.trigger('click')
    await flushPromises()

    expect(AppApi.SkipGuessTheMove).toHaveBeenCalled()
    expect(wrapper.text()).toContain('Move revealed: e4')
    expect(wrapper.text()).toContain('Club Player')
    expect(wrapper.text()).toContain('Good tactical eye')
  })
})
