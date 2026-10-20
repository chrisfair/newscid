import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import FairPlayModal from './FairPlayModal.vue'
import * as AppBindings from '../../wailsjs/go/main/App'

vi.mock('../../wailsjs/go/main/App', () => ({
  AnalyzePlayerFairPlay: vi.fn(),
  DownloadAndAnalyzePlayerFairPlay: vi.fn(),
  AnalyzeHardMoveBenchmark: vi.fn(),
  DownloadAndAnalyzeHardMoveBenchmark: vi.fn(),
  GetDatabasePlayers: vi.fn(),
}))

const mockHardMoveReport = {
  benchmarkPlayer: 'SuspectPlayer',
  timeControlFilter: 'ALL',
  hardMoveProbCutoff: 0.30,
  totalGamesScanned: 42,
  selfGamesQualified: 40,
  opponentGamesQualified: 40,
  totalHardMovesAnalyzed: 380,
  selfAvgHardHitRatePct: 24.5,
  opponentAvgHardHitRatePct: 31.2,
  selfExpectedHitRatePct: 24.0,
  opponentExpectedHitRatePct: 24.0,
  topTwoBandThreshold: 2.15,
  topHalfPctThreshold: 2.55,
  selfInTopTwoBands: 1,
  selfTopTwoRatePct: 2.5,
  opponentInTopTwoBands: 6,
  opponentTopTwoRatePct: 15.0,
  expectedOpponentTopTwo: 1.0,
  excessOpponentGames: 5.0,
  excessGamesLowEstimate: 3,
  excessGamesHighEstimate: 7,
  excessPctOfAllGames: 12.5,
  excessPValue: 0.001,
  oddsOneIn: 1000,
  selfInTop100Count: 18,
  opponentInTop100Count: 22,
  bands: [
    {
      bandIndex: 1,
      bandName: 'Band 1: Normal (<90%)',
      description: 'Everyday human games',
      thresholdScore: 0,
      selfCount: 36,
      selfPct: 90.0,
      opponentCount: 30,
      opponentPct: 75.0,
      expectedOppCount: 36.0,
      excessOppCount: 0,
    },
    {
      bandIndex: 5,
      bandName: 'Band 5: Top 0.5% Band B',
      description: 'Extreme outlier',
      thresholdScore: 2.55,
      selfCount: 0,
      selfPct: 0.0,
      opponentCount: 4,
      opponentPct: 10.0,
      expectedOppCount: 0.2,
      excessOppCount: 3.8,
    },
  ],
  selectiveSimulation: [
    {
      label: '0 Swaps (Your Real Benchmark)',
      swappedMovesSpec: '0 engine moves',
      avgSwappedPerGame: 0,
      topTwoBandsCaughtPct: 2.5,
      topFivePctCaughtPct: 5.0,
      avgUnusualnessScore: 0.45,
    },
    {
      label: '100% of Hard Moves Swapped',
      swappedMovesSpec: 'Full engine assistance',
      avgSwappedPerGame: 6.2,
      topTwoBandsCaughtPct: 95.0,
      topFivePctCaughtPct: 100.0,
      avgUnusualnessScore: 4.12,
    },
  ],
  bannedValidation: {
    accountsChecked: 10,
    confirmedBannedCount: 2,
    gamesVsBanned: 5,
    bannedInTopTwoBands: 3,
    bannedTopTwoRatePct: 60.0,
    openTopTwoRatePct: 8.5,
    multiplierVsOpen: 7.1,
  },
  top100Games: [
    {
      gameId: 19,
      date: '2026-10-16',
      timeControl: 'Rapid',
      side: 'opponent',
      playerName: 'CheaterAccount99',
      anonymousLabel: 'Opponent #001',
      playerElo: 1680,
      opponentName: 'SuspectPlayer',
      result: '0-1',
      totalNonBookMoves: 26,
      hardMoveCount: 8,
      hardMovesFound: 7,
      hardMoveHitRatePct: 87.5,
      expectedHardHits: 1.9,
      tailProbability: 0.0001,
      unusualnessScore: 4.0,
      zScore: 4.2,
      bandIndex: 5,
      bandName: 'Band 5: Extreme Outlier (Top 0.5%)',
      isBannedOpponent: true,
      banSource: 'Chess.com Closed: Fair Play',
    },
  ],
  verdictHeadline: 'Your opponents games are more unusual than your benchmark by ~5 games beyond chance.',
  verdictFindings: ['Hard-Move Filter (p <= 30%): Isolated 380 non-obvious positions.'],
}

const mockDossier = {
  playerName: 'SuspectPlayer',
  totalGamesAnalyzed: 42,
  totalDaysActive: 6,
  overallIssueScore: 81.4,
  highestRiskTimeControl: 'Rapid',
  highestControlIssueScore: 88.5,
  overallClassification: 'High Consistent Time-Control Anomaly',
  timeControlProfiles: [
    {
      category: 'Bullet',
      gamesAnalyzed: 20,
      daysActive: 5,
      daysWithElevatedIssue: 0,
      dailyConsistencyPct: 0,
      winRatePct: 48.0,
      avgOfficialElo: 1650,
      intrinsicPerformanceElo: 1620,
      eloOverperformanceDelta: -30,
      avgAcpl: 54.2,
      top1AgreementPct: 44.5,
      criticalAccuracyPct: 41.0,
      blunderFreeGamePct: 15.0,
      avgMoveTimeSec: 1.1,
      avgMoveTimeCV: 0.94,
      roboticCadenceGamePct: 0,
      consistentIssueScore: 18.5,
      zScore: -0.65,
      consistencyClassification: 'Normal Human Variance',
      summary: 'Bullet normal human baseline',
    },
    {
      category: 'Rapid',
      gamesAnalyzed: 22,
      daysActive: 6,
      daysWithElevatedIssue: 6,
      dailyConsistencyPct: 100.0,
      winRatePct: 90.9,
      avgOfficialElo: 1680,
      intrinsicPerformanceElo: 2890,
      eloOverperformanceDelta: 1210,
      avgAcpl: 11.4,
      top1AgreementPct: 86.2,
      criticalAccuracyPct: 91.0,
      blunderFreeGamePct: 95.5,
      avgMoveTimeSec: 5.1,
      avgMoveTimeCV: 0.24,
      roboticCadenceGamePct: 90.9,
      consistentIssueScore: 88.5,
      zScore: 4.17,
      consistencyClassification: 'High Consistent Time-Control Anomaly',
      summary: 'Rapid consistent issue across 6 active days',
    },
  ],
  dailyTimeline: [
    {
      date: '2026-10-16',
      gamesCount: 7,
      timeControls: ['Bullet', 'Rapid'],
      primaryTimeControl: 'Rapid',
      winRatePct: 85.7,
      avgAcpl: 14.2,
      top1AgreementPct: 82.0,
      avgMoveTimeCV: 0.27,
      intrinsicPerformance: 2780,
      dailyIssueScore: 84.0,
      peakGameId: 19,
      classification: 'High Consistent Time-Control Anomaly',
    },
  ],
  crossDivergence: {
    hasMultipleControls: true,
    baselineControl: 'Bullet',
    suspectControl: 'Rapid',
    iprDelta: 1270,
    acplDelta: 42.8,
    cadenceCvDelta: 0.70,
    divergenceScore: 95.0,
    explanation: 'Comparing Rapid vs. Bullet baseline: Intrinsic Performance jumps by +1270 Elo.',
  },
  flaggedGames: [
    {
      gameId: 19,
      date: '2026-10-16',
      event: 'Rated Rapid game',
      site: 'https://lichess.org/r19',
      timeControlRaw: '600+0',
      timeControlCategory: 'Rapid',
      playerColor: 'white',
      playerElo: 1680,
      opponentName: 'StrongIM',
      opponentElo: 2350,
      result: '1-0',
      playerScore: 1.0,
      totalPlies: 64,
      nonBookMoves: 26,
      hasClockData: true,
      hasEvalData: true,
      avgMoveTimeSec: 5.0,
      moveTimeStdDevSec: 1.1,
      moveTimeCV: 0.22,
      roboticCadencePct: 92.0,
      delayedRecaptureCount: 3,
      acpl: 9.8,
      top1AgreementPct: 88.5,
      criticalAccuracyPct: 92.0,
      blunderCount: 0,
      intrinsicPerformanceElo: 3010,
      gameIssueScore: 94.0,
      flags: ['Robotic flat move-time cadence (5.0s ± 0.22 CV)', 'Ultra-low ACPL (9.8 cp)'],
    },
  ],
  allGames: [],
  keyFindings: ['Rapid consistent issue across 6 active days'],
  reportMarkdown: '# Longitudinal Fair-Play Report',
}

describe('FairPlayModal.vue', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(AppBindings.GetDatabasePlayers).mockResolvedValue(['SuspectPlayer', 'Magnus Carlsen'])
    vi.mocked(AppBindings.AnalyzePlayerFairPlay).mockResolvedValue(mockDossier as any)
    vi.mocked(AppBindings.DownloadAndAnalyzePlayerFairPlay).mockResolvedValue(mockDossier as any)
    vi.mocked(AppBindings.AnalyzeHardMoveBenchmark).mockResolvedValue(mockHardMoveReport as any)
    vi.mocked(AppBindings.DownloadAndAnalyzeHardMoveBenchmark).mockResolvedValue(mockHardMoveReport as any)
  })

  it('analyzes a database player and renders Per-Time-Control Consistent Issue Scores and Cross-Control Divergence', async () => {
    const wrapper = mount(FairPlayModal, {
      props: { dbOpen: true, initialPlayer: 'SuspectPlayer' },
    })
    await flushPromises()

    await wrapper.get('[data-testid="run-fairplay-btn"]').trigger('click')
    await flushPromises()

    expect(AppBindings.AnalyzePlayerFairPlay).toHaveBeenCalledWith('SuspectPlayer', 250)
    expect(wrapper.get('[data-testid="highest-risk-tc"]').text()).toBe('Rapid')
    expect(wrapper.get('[data-testid="tc-card-Rapid"]').text()).toContain('88.5')
    expect(wrapper.get('[data-testid="tc-card-Bullet"]').text()).toContain('18.5')
    expect(wrapper.get('[data-testid="cross-control-banner"]').text()).toContain('+1270 Elo')
  })

  it('switches to Hard-Move & Self-Benchmark tab, renders excess estimate, simulation, and supports Anonymous Mode', async () => {
    const wrapper = mount(FairPlayModal, {
      props: { dbOpen: true, initialPlayer: 'SuspectPlayer' },
    })
    await flushPromises()
    await wrapper.get('[data-testid="run-fairplay-btn"]').trigger('click')
    await flushPromises()

    await wrapper.get('[data-testid="tab-hardmove"]').trigger('click')
    const view = wrapper.get('[data-testid="hardmove-benchmark-view"]')
    expect(view.text()).toContain('3–7 games')
    expect(view.text()).toContain('~1 in 1,000')
    expect(view.text()).toContain('CheaterAccount99')
    expect(view.text()).toContain('100% of Hard Moves Swapped')

    const anonBtn = view.findAll('button').find(b => b.text().includes('Anonymous Mode'))
    expect(anonBtn).toBeDefined()
    await anonBtn!.trigger('click')
    expect(view.text()).toContain('Opponent #001')
    expect(view.text()).not.toContain('CheaterAccount99')
  })

  it('switches to Everyday Consistency Timeline and Flagged Games tabs and emits select-game', async () => {
    const wrapper = mount(FairPlayModal, {
      props: { dbOpen: true, initialPlayer: 'SuspectPlayer' },
    })
    await flushPromises()
    await wrapper.get('[data-testid="run-fairplay-btn"]').trigger('click')
    await flushPromises()

    await wrapper.get('[data-testid="tab-daily"]').trigger('click')
    expect(wrapper.get('[data-testid="daily-timeline-table"]').text()).toContain('2026-10-16')

    await wrapper.get('[data-testid="tab-games"]').trigger('click')
    expect(wrapper.get('[data-testid="flagged-games-list"]').text()).toContain('StrongIM')

    await wrapper.get('[data-testid="inspect-flagged-game-btn"]').trigger('click')
    expect(wrapper.emitted('select-game')?.[0]).toEqual([19])
  })
})
