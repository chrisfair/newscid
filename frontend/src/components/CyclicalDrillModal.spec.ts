import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import CyclicalDrillModal from './CyclicalDrillModal.vue'
import Chessboard from './Chessboard.vue'
import * as AppBindings from '../../wailsjs/go/main/App'

vi.mock('../../wailsjs/go/main/App', () => ({
  GetCyclicalDashboard: vi.fn(),
  GetLichessThemesCatalog: vi.fn(),
  CreateCyclicalDrillSet: vi.fn(),
  DeleteCyclicalDrillSet: vi.fn(),
  RecordCyclicalPuzzleAttempt: vi.fn(),
  StartNextCyclicalCycle: vi.fn(),
  CheckCyclicalEntitlement: vi.fn(),
  GetPersonas: vi.fn(),
}))

const mockCatalog = [
  { key: 'fork', name: 'Fork / Double Attack', category: 'Motifs', description: 'Attack two targets at once.' },
  { key: 'pin', name: 'Pin', category: 'Motifs', description: 'Pin an enemy piece.' },
  { key: 'backRankMate', name: 'Back Rank Mate', category: 'Mates', description: 'Mate on the back rank.' },
  { key: 'sacrifice', name: 'Sacrifice', category: 'Advanced', description: 'Material sacrifice.' },
  { key: 'quietMove', name: 'Quiet Move', category: 'Advanced', description: 'Quiet tactical preparation.' },
]

const mockDashboard = {
  sets: [
    {
      id: 'set-1',
      name: 'My Autumn Blunders & Tactics',
      playerName: 'ChristopherFair',
      themeFilter: ['fork', 'backRankMate'],
      createdAt: 1760000000,
      currentCycle: 3,
      targetDays: 7,
      puzzles: [
        {
          id: 'puz-easy-1',
          sourceGameId: 12,
          white: 'ChristopherFair',
          black: 'ClubRival',
          event: 'Rapid Championship',
          date: '2026.10.10',
          ply: 24,
          fen: '6k1/5ppp/8/8/8/8/5PPP/4R1K1 w - - 0 1',
          turn: 'w',
          solutionUci: ['e1e8'],
          solutionSan: ['Re8#'],
          playedBlunderUci: 'h2h3',
          themes: ['backRankMate', 'mateIn1'],
          primaryTheme: 'backRankMate',
          difficultyTier: 'Easy',
          difficultyOrder: 1,
          estimatedRating: 1120,
          evalDeltaCp: 900,
          explanation: 'Immediate back-rank mate with Re8#.',
        },
        {
          id: 'puz-hard-2',
          sourceGameId: 18,
          white: 'ChristopherFair',
          black: 'MasterBot',
          event: 'Classical League',
          date: '2026.10.14',
          ply: 36,
          fen: 'r1bqk2r/pppp1ppp/2n2n2/4p3/2B1P3/5N2/PPPP1PPP/RNBQK2R w KQkq - 4 4',
          turn: 'w',
          solutionUci: ['f3g5', 'd7d5', 'e4d5'],
          solutionSan: ['Ng5', 'd5', 'exd5'],
          playedBlunderUci: 'd2d3',
          themes: ['fork', 'sacrifice'],
          primaryTheme: 'fork',
          difficultyTier: 'Hard',
          difficultyOrder: 2,
          estimatedRating: 1940,
          evalDeltaCp: 420,
          explanation: 'Attacking f7 and winning central control.',
        },
      ],
      cycles: [
        {
          cycleNumber: 1,
          completedAt: '2026-10-12',
          totalPuzzles: 2,
          firstTryCorrect: 1,
          totalSolved: 2,
          accuracyPct: 50.0,
          totalTimeSec: 120,
          targetTimeSec: 120,
          avgSecPerPuzzle: 60.0,
          speedupVsCycle1: 1.0,
          easyAccuracyPct: 100.0,
          mediumAccuracyPct: 75.0,
          hardAccuracyPct: 0.0,
        },
        {
          cycleNumber: 2,
          completedAt: '2026-10-16',
          totalPuzzles: 2,
          firstTryCorrect: 2,
          totalSolved: 2,
          accuracyPct: 100.0,
          totalTimeSec: 52,
          targetTimeSec: 60,
          avgSecPerPuzzle: 26.0,
          speedupVsCycle1: 2.31,
          easyAccuracyPct: 100.0,
          mediumAccuracyPct: 90.0,
          hardAccuracyPct: 100.0,
        },
      ],
      activeProgress: {
        cycleNumber: 3,
        currentIndex: 0,
        elapsedSec: 15,
        firstTryCorrect: 0,
        totalAttempted: 0,
      },
    },
  ],
  themeStats: [
    {
      themeKey: 'backRankMate',
      themeName: 'Back Rank Mate',
      category: 'Mates',
      attempts: 12,
      firstWins: 11,
      fixed: 1,
      failed: 1,
      accuracyPct: 91.7,
      avgTimeSec: 8.4,
      ratingAvg: 1450,
      performanceRating: 1867,
    },
    {
      themeKey: 'fork',
      themeName: 'Fork / Double Attack',
      category: 'Motifs',
      attempts: 16,
      firstWins: 13,
      fixed: 2,
      failed: 3,
      accuracyPct: 81.3,
      avgTimeSec: 12.1,
      ratingAvg: 1520,
      performanceRating: 1833,
    },
    {
      themeKey: 'quietMove',
      themeName: 'Quiet Move',
      category: 'Advanced',
      attempts: 10,
      firstWins: 3,
      fixed: 4,
      failed: 7,
      accuracyPct: 30.0,
      avgTimeSec: 28.5,
      ratingAvg: 1750,
      performanceRating: 1550,
    },
  ],
  radarAxes: [
    {
      themeKey: 'backRankMate',
      themeName: 'Back Rank Mate',
      category: 'Mates',
      attempts: 12,
      firstWins: 11,
      fixed: 1,
      failed: 1,
      accuracyPct: 91.7,
      avgTimeSec: 8.4,
      ratingAvg: 1450,
      performanceRating: 1867,
    },
    {
      themeKey: 'fork',
      themeName: 'Fork / Double Attack',
      category: 'Motifs',
      attempts: 16,
      firstWins: 13,
      fixed: 2,
      failed: 3,
      accuracyPct: 81.3,
      avgTimeSec: 12.1,
      ratingAvg: 1520,
      performanceRating: 1833,
    },
    {
      themeKey: 'quietMove',
      themeName: 'Quiet Move',
      category: 'Advanced',
      attempts: 10,
      firstWins: 3,
      fixed: 4,
      failed: 7,
      accuracyPct: 30.0,
      avgTimeSec: 28.5,
      ratingAvg: 1750,
      performanceRating: 1550,
    },
  ],
  strongThemes: [
    {
      themeKey: 'backRankMate',
      themeName: 'Back Rank Mate',
      category: 'Mates',
      attempts: 12,
      firstWins: 11,
      fixed: 1,
      failed: 1,
      accuracyPct: 91.7,
      avgTimeSec: 8.4,
      ratingAvg: 1450,
      performanceRating: 1867,
    },
  ],
  weakThemes: [
    {
      themeKey: 'quietMove',
      themeName: 'Quiet Move',
      category: 'Advanced',
      attempts: 10,
      firstWins: 3,
      fixed: 4,
      failed: 7,
      accuracyPct: 30.0,
      avgTimeSec: 28.5,
      ratingAvg: 1750,
      performanceRating: 1550,
    },
  ],
  globalPerformanceRating: 1765,
  totalPuzzlesAttempted: 38,
  totalCyclesCompleted: 2,
  improvementSummary: 'Cycle 2 achieved 2.31x faster solve speed and +50% first-try accuracy gain over Cycle 1.',
  methodAttribution:
    "Implements High-Frequency Cyclical Pattern Repetition—similar to GM Axel Smith & IM Hans Tikkanen's 'The Woodpecker Method'.",
}

describe('CyclicalDrillModal.vue', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked((AppBindings as any).GetCyclicalDashboard).mockResolvedValue(
      JSON.parse(JSON.stringify(mockDashboard)),
    )
    vi.mocked((AppBindings as any).GetLichessThemesCatalog).mockResolvedValue(mockCatalog)
    vi.mocked((AppBindings as any).CheckCyclicalEntitlement).mockResolvedValue({
      featureId: 'cyclical_tactical_drill',
      unlocked: true,
      enforceGate: false,
      supporterActive: true,
      bughouseOnlineSync: true,
      tierRequired: 'Supporter',
      message: 'Included in NewScid Workstation (Future bughouse-online Supporter Feature)',
    })
    vi.mocked((AppBindings as any).GetPersonas).mockResolvedValue([
      { id: 'p-1', displayName: 'ChristopherFair' },
    ])
    vi.mocked((AppBindings as any).CreateCyclicalDrillSet).mockResolvedValue(
      JSON.parse(JSON.stringify(mockDashboard.sets[0])),
    )
    vi.mocked((AppBindings as any).RecordCyclicalPuzzleAttempt).mockResolvedValue(
      JSON.parse(JSON.stringify(mockDashboard.sets[0])),
    )
  })

  it('renders header, legal cyclical training attribution, supporter entitlement badge, and Cycle Sets improvement history', async () => {
    const wrapper = mount(CyclicalDrillModal)
    await flushPromises()

    expect(wrapper.get('[data-testid="cyclical-modal-title"]').text()).toContain(
      'Cyclical Tactical Drill & Theme Mastery',
    )
    expect(wrapper.get('[data-testid="method-attribution-banner"]').text()).toContain(
      'The Woodpecker Method',
    )
    expect(wrapper.get('[data-testid="entitlement-badge"]').text()).toContain(
      'bughouse-online',
    )
    expect(wrapper.get('[data-testid="global-perf-rating"]').text()).toContain('1765 Elo')

    // Switch to Cycle Sets & Improvement History tab
    await wrapper.get('[data-testid="tab-sets"]').trigger('click')
    await flushPromises()

    const setCard = wrapper.get('[data-testid="cycle-set-card-set-1"]')
    expect(setCard.text()).toContain('My Autumn Blunders & Tactics')
    expect(setCard.text()).toContain('Cycle 3 / 7')

    const historyTable = wrapper.get('[data-testid="improvement-history-set-1"]')
    expect(historyTable.text()).toContain('Cycle 1')
    expect(historyTable.text()).toContain('Cycle 2')
    expect(historyTable.text()).toContain('2.31x faster')
    expect(historyTable.text()).toContain('100.0%')
  })

  it('switches to Lichess Theme Radar & Proficiency tab and renders 9-axis SVG radar, Strengths/Weaknesses, and theme ratings', async () => {
    const wrapper = mount(CyclicalDrillModal)
    await flushPromises()

    await wrapper.get('[data-testid="tab-themes"]').trigger('click')
    await flushPromises()

    const radarSvg = wrapper.find('[data-testid="lichess-radar-svg"]')
    expect(radarSvg.exists()).toBe(true)
    expect(wrapper.get('[data-testid="radar-performance-polygon"]').attributes('points')).toBeTruthy()
    expect(radarSvg.text()).toContain('Back Rank Mate (1867)')
    expect(radarSvg.text()).toContain('Fork / Double Attack (1833)')

    const strongPanel = wrapper.get('[data-testid="strong-themes-panel"]')
    expect(strongPanel.text()).toContain('Back Rank Mate')
    expect(strongPanel.text()).toContain('1867 Elo')

    const weakPanel = wrapper.get('[data-testid="weak-themes-panel"]')
    expect(weakPanel.text()).toContain('Quiet Move')
    expect(weakPanel.text()).toContain('1550 Elo')

    // Verify 1-click Build Cycle Set for Weak Theme calls CreateCyclicalDrillSet
    await wrapper.get('[data-testid="drill-weak-theme-quietMove"]').trigger('click')
    await flushPromises()

    expect((AppBindings as any).CreateCyclicalDrillSet).toHaveBeenCalledWith(
      'Quiet Move Mastery Cycle',
      'ChristopherFair',
      ['quietMove'],
      20,
    )
  })

  it('creates a new Cycle Set filtered by specific Lichess themes and player history', async () => {
    const wrapper = mount(CyclicalDrillModal)
    await flushPromises()

    await wrapper.get('[data-testid="tab-sets"]').trigger('click')
    await flushPromises()

    await wrapper.get('[data-testid="set-name-input"]').setValue('Fork & Mate Speed Cycle')
    await wrapper.get('[data-testid="player-name-input"]').setValue('ChristopherFair')
    await wrapper.get('[data-testid="set-size-32"]').trigger('click')
    await wrapper.get('[data-testid="theme-chip-fork"]').trigger('click')
    await wrapper.get('[data-testid="theme-chip-backRankMate"]').trigger('click')

    await wrapper.get('[data-testid="create-set-submit-btn"]').trigger('click')
    await flushPromises()

    expect((AppBindings as any).CreateCyclicalDrillSet).toHaveBeenCalledWith(
      'Fork & Mate Speed Cycle',
      'ChristopherFair',
      ['fork', 'backRankMate'],
      32,
    )
  })

  it('solves puzzles on the Active Drill Board ordered Easy -> Hard and calls RecordCyclicalPuzzleAttempt', async () => {
    const wrapper = mount(CyclicalDrillModal)
    await flushPromises()

    // Starts on Puzzle #1 (Easy tier)
    expect(wrapper.get('[data-testid="puzzle-difficulty-badge"]').text()).toBe('Easy')
    expect(wrapper.get('[data-testid="puzzle-order-indicator"]').text()).toContain('#1 / 2')

    const board = wrapper.findComponent(Chessboard)
    expect(board.exists()).toBe(true)

    // Emit correct winning move e1e8
    board.vm.$emit('move', 'e1e8')
    await flushPromises()

    expect((AppBindings as any).RecordCyclicalPuzzleAttempt).toHaveBeenCalledWith(
      'set-1',
      'puz-easy-1',
      true,
      expect.any(Number),
    )
    expect(wrapper.get('[data-testid="feedback-correct"]').text()).toContain('Solved cleanly')
    expect(wrapper.get('[data-testid="puzzle-explanation"]').text()).toContain('Immediate back-rank mate')

    // Advance to Puzzle #2 (Hard tier, multi-move with automatic opponent reply)
    await wrapper.get('[data-testid="next-puzzle-btn"]').trigger('click')
    await flushPromises()

    expect(wrapper.get('[data-testid="puzzle-difficulty-badge"]').text()).toBe('Hard')
    expect(wrapper.get('[data-testid="puzzle-order-indicator"]').text()).toContain('#2 / 2')

    // Play first solver move f3g5 (opponent automatically replies d7d5)
    board.vm.$emit('move', 'f3g5')
    await flushPromises()
    expect(wrapper.get('[data-testid="feedback-step"]').text()).toContain('Opponent replied d7d5')

    // Play second solver move e4d5 to complete the puzzle
    board.vm.$emit('move', 'e4d5')
    await flushPromises()
    expect((AppBindings as any).RecordCyclicalPuzzleAttempt).toHaveBeenCalledWith(
      'set-1',
      'puz-hard-2',
      true,
      expect.any(Number),
    )
  })
})
