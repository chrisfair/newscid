import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import PieceSafetyModal from './PieceSafetyModal.vue'

vi.mock('../../wailsjs/go/main/App', () => ({
  AnalyzeBoardSafety: vi.fn().mockResolvedValue({
    fen: 'r1bqk2r/pppp1ppp/2n5/2b1p3/2B1n3/3P1N2/PPP2PPP/RNBQ1RK1 b kq - 0 6',
    turn: 'white',
    pieces: [
      {
        square: 'e4',
        piece: 'N',
        color: 'black',
        pieceValue: 320,
        attackers: ['d3'],
        defenders: [],
        status: 'hanging',
        netLoss: 320,
        explanation: 'Black Knight on e4 is completely hanging with 0 defenders! Attacked by d3.',
      },
    ],
    hangingPieces: [
      {
        square: 'e4',
        piece: 'N',
        color: 'black',
        pieceValue: 320,
        attackers: ['d3'],
        defenders: [],
        status: 'hanging',
        netLoss: 320,
        explanation: 'Black Knight on e4 is completely hanging with 0 defenders! Attacked by d3.',
      },
    ],
    underdefendedPieces: [],
    loosePieces: [],
    hangingSquares: ['e4'],
    totalHangingWhite: 0,
    totalHangingBlack: 1,
    criticalThreat: {
      square: 'e4',
      piece: 'N',
      color: 'black',
      pieceValue: 320,
      attackers: ['d3'],
      defenders: [],
      status: 'hanging',
      netLoss: 320,
      explanation: 'Black Knight on e4 is completely hanging with 0 defenders! Attacked by d3.',
    },
    blunderCount: 1,
  }),
  GetSafetyDrillBatch: vi.fn().mockResolvedValue([
    {
      id: 'hang_02_italian_hanging_knight',
      title: 'Free Knight on e4',
      fen: 'r1bqk2r/pppp1ppp/2n5/2b1p3/2B1n3/3P1N2/PPP2PPP/RNBQ1RK1 b kq - 0 6',
      turn: 'white',
      mode: 'punish_blunder',
      prompt: 'Black left a minor piece en prise with 0 defenders. Click the hanging piece!',
      targetColor: 'black',
      correctSquares: ['e4'],
      winningMoveUCI: 'd3e4',
      winningMoveSAN: 'dxe4',
      explanation: "Black's Knight on e4 is attacked by White's d3 pawn and defended by nothing!",
      difficulty: 'beginner',
      source: 'Italian Game Blunder',
    },
  ]),
  GenerateSafetyDrillsFromDatabase: vi.fn().mockResolvedValue([]),
  SubmitSafetyDrillAnswer: vi.fn().mockResolvedValue({
    isCorrect: true,
    score: 100,
    correctSquares: ['e4'],
    missedSquares: [],
    wrongSquares: [],
    explanation: "Black's Knight on e4 is attacked by White's d3 pawn and defended by nothing!",
    bestMoveUCI: 'd3e4',
    bestMoveSAN: 'dxe4',
  }),
}))

describe('PieceSafetyModal.vue', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders modal with title, tabs, and loads initial safety drill', async () => {
    const wrapper = mount(PieceSafetyModal, {
      props: {
        initialFEN: 'r1bqk2r/pppp1ppp/2n5/2b1p3/2B1n3/3P1N2/PPP2PPP/RNBQ1RK1 b kq - 0 6',
      },
    })

    expect(wrapper.text()).toContain('Piece Safety Scanner')
    expect(wrapper.text()).toContain("Don't Hang Pieces")
    expect(wrapper.text()).toContain('Rapid Scanner (Blitz)')

    await new Promise((resolve) => setTimeout(resolve, 50))
    expect(wrapper.text()).toContain('Free Knight on e4')
  })

  it('allows clicking squares, submitting answer, and receiving positive feedback', async () => {
    const wrapper = mount(PieceSafetyModal, {
      props: {
        initialFEN: 'r1bqk2r/pppp1ppp/2n5/2b1p3/2B1n3/3P1N2/PPP2PPP/RNBQ1RK1 b kq - 0 6',
      },
    })

    await new Promise((resolve) => setTimeout(resolve, 50))

    // Click submit button
    const checkBtn = wrapper.findAll('button').find((b) => b.text().includes('Check Answer'))
    expect(checkBtn?.exists()).toBe(true)
    await checkBtn?.trigger('click')

    await new Promise((resolve) => setTimeout(resolve, 50))
    expect(wrapper.text()).toContain('Outstanding! All Hanging Pieces Identified.')
  })

  it('emits close event when close button is clicked', async () => {
    const wrapper = mount(PieceSafetyModal)
    const closeBtn = wrapper.find('button[class*="hover:bg-zinc-800"]')
    expect(closeBtn.exists()).toBe(true)
    await closeBtn.trigger('click')
    expect(wrapper.emitted('close')).toBeTruthy()
  })
})
