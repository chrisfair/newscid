import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import FairPlayModal from './FairPlayModal.vue'
import * as AppBindings from '../../wailsjs/go/main/App'

vi.mock('../../wailsjs/go/main/App', () => ({
  AnalyzePlayerFairPlay: vi.fn(),
  DownloadAndAnalyzePlayerFairPlay: vi.fn(),
  GetDatabasePlayers: vi.fn(),
}))

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
