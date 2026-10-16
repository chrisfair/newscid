import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import TimeManagementModal from './TimeManagementModal.vue'

vi.mock('../../wailsjs/go/main/App', () => ({
  AnalyzeGameTimeManagement: vi.fn().mockResolvedValue({
    gameId: 1,
    whitePlayer: 'Carlsen, Magnus',
    blackPlayer: 'Nakamura, Hikaru',
    event: 'Speed Chess 2024',
    timeControl: '300+0',
    whiteHasClock: true,
    blackHasClock: true,
    moves: [
      {
        ply: 1,
        moveNumber: 1,
        turn: 'white',
        san: 'e4',
        uci: 'e2e4',
        clockRemainingSec: 300,
        timeSpentSec: 2,
        clockStr: '0:05:00',
        phase: 'opening',
        isTank: false,
        isPanic: false,
        isScramble: false,
        evalCp: 20,
        evalDropCp: 0,
        evaluationClassification: 'good',
        flag: 'normal',
      },
      {
        ply: 5,
        moveNumber: 3,
        turn: 'white',
        san: 'd4',
        uci: 'd2d4',
        clockRemainingSec: 160,
        timeSpentSec: 138,
        clockStr: '0:02:40',
        phase: 'opening',
        isTank: true,
        isPanic: false,
        isScramble: false,
        evalCp: 35,
        evalDropCp: 0,
        evaluationClassification: 'best',
        flag: 'tank_success',
      },
    ],
    whiteStats: {
      totalMoves: 2,
      totalTimeSpentSec: 140,
      avgMoveTimeSec: 70,
      medianMoveTimeSec: 70,
      openingAvgTimeSec: 70,
      middlegameAvgTimeSec: 0,
      endgameAvgTimeSec: 0,
      longestTankSec: 138,
      longestTankPly: 5,
      tankCount: 1,
      tankBlunderCount: 0,
      panicMoveCount: 0,
      timeTroubleMoves: 0,
      timeTroubleBlunders: 0,
      correlationTimeVsLoss: 0,
    },
    blackStats: {
      totalMoves: 0,
      totalTimeSpentSec: 0,
      avgMoveTimeSec: 0,
      medianMoveTimeSec: 0,
      openingAvgTimeSec: 0,
      middlegameAvgTimeSec: 0,
      endgameAvgTimeSec: 0,
      longestTankSec: 0,
      longestTankPly: 0,
      tankCount: 0,
      tankBlunderCount: 0,
      panicMoveCount: 0,
      timeTroubleMoves: 0,
      timeTroubleBlunders: 0,
      correlationTimeVsLoss: 0,
    },
    tankDossier: [
      {
        ply: 5,
        moveNumber: 3,
        player: 'Carlsen, Magnus',
        color: 'white',
        san: 'd4',
        timeSpentSec: 138,
        clockBeforeSec: 298,
        clockAfterSec: 160,
        evalDropCp: 0,
        outcome: 'Masterstroke',
        description: 'Carlsen spent 138s on move 3...d4. Outcome: Masterstroke.',
      },
    ],
    panicIncidents: [],
    scrambleSummary: {
      whiteEnteredScramblePly: 0,
      blackEnteredScramblePly: 0,
      whiteScrambleMoves: 0,
      blackScrambleMoves: 0,
      whiteScrambleBlunders: 0,
      blackScrambleBlunders: 0,
    },
    recommendations: [
      '✅ Clean clock discipline demonstrated by both sides!',
    ],
  }),
  AnalyzePlayerTimeManagement: vi.fn().mockResolvedValue({
    playerName: 'Carlsen, Magnus',
    gamesAnalyzed: 50,
    gamesWithClock: 45,
    averageMoveTimeSec: 8.5,
    openingAvgSec: 4.2,
    middlegameAvgSec: 12.1,
    endgameAvgSec: 6.3,
    totalTanks: 12,
    tankSuccessRate: 83.3,
    panicMoveCount: 1,
    panicErrorRate: 0.1,
    scrambleFrequencyPct: 15.0,
    scrambleBlunderCount: 2,
    habitFlags: [
      'Fast Opening Pacing: Converts opening preparation efficiently',
    ],
    coachingDossier: [
      'Superb practical time distribution across all game phases.',
    ],
  }),
}))

describe('TimeManagementModal.vue', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders modal header, summary, and loads game telemetry', async () => {
    const wrapper = mount(TimeManagementModal, {
      props: { gameId: 1 },
    })

    expect(wrapper.text()).toContain('Time Management & Tanking Dossier')
    expect(wrapper.text()).toContain('%clk & %emt Analytics')

    await flushPromises()
    await new Promise(r => setTimeout(r, 60))

    expect(wrapper.text()).toContain('Carlsen, Magnus')
    expect(wrapper.text()).toContain('Nakamura, Hikaru')
    expect(wrapper.text()).toContain('Deep Tanks (≥2m)')
    expect(wrapper.text()).toContain('Tank ROI (Success Rate)')
    expect(wrapper.text()).toContain('TANK SUCCESS')
  })

  it('filters moves by tanks', async () => {
    const wrapper = mount(TimeManagementModal, {
      props: { gameId: 1 },
    })
    await flushPromises()
    await new Promise(r => setTimeout(r, 60))

    const buttons = wrapper.findAll('button')
    const tankFilter = buttons.find(b => b.text().includes('Tanks'))
    expect(tankFilter).toBeDefined()
    await tankFilter!.trigger('click')

    expect(wrapper.text()).toContain('d4')
    expect(wrapper.text()).not.toContain('e4')
  })

  it('emits jumpToPly when a move row is clicked', async () => {
    const wrapper = mount(TimeManagementModal, {
      props: { gameId: 1 },
    })
    await flushPromises()
    await new Promise(r => setTimeout(r, 60))

    // Click on move 3. d4 (ply 5)
    const moveRows = wrapper.findAll('.cursor-pointer')
    const d4Row = moveRows.find(r => r.text().includes('d4'))
    expect(d4Row).toBeDefined()
    await d4Row!.trigger('click')

    expect(wrapper.emitted('jumpToPly')).toBeTruthy()
    expect(wrapper.emitted('jumpToPly')![0]).toEqual([5])
  })

  it('switches to player historical dossier tab and renders historical stats', async () => {
    const wrapper = mount(TimeManagementModal, {
      props: { defaultPlayerName: 'Carlsen, Magnus' },
    })
    await flushPromises()
    await new Promise(r => setTimeout(r, 60))

    const playerTabBtn = wrapper.findAll('button').find(b => b.text().includes('Player Historical Dossier'))
    expect(playerTabBtn).toBeDefined()
    await playerTabBtn!.trigger('click')

    await flushPromises()
    await new Promise(r => setTimeout(r, 60))

    expect(wrapper.text()).toContain('Average Move Time')
    expect(wrapper.text()).toContain('Tank Success Rate')
    expect(wrapper.text()).toContain('83%')
    expect(wrapper.text()).toContain('Fast Opening Pacing')
  })

  it('emits close on clicking close button', async () => {
    const wrapper = mount(TimeManagementModal)
    const closeBtn = wrapper.findAll('button').find(b => b.find('svg').exists() && !b.text())
    expect(closeBtn).toBeDefined()
    await closeBtn!.trigger('click')

    expect(wrapper.emitted('close')).toBeTruthy()
  })
})
