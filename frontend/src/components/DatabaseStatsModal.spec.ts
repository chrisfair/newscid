import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import DatabaseStatsModal from './DatabaseStatsModal.vue'
import * as AppApi from '../../wailsjs/go/main/App'

vi.mock('../../wailsjs/go/main/App', () => ({
  GetDatabaseDemographics: vi.fn().mockResolvedValue({
    totalGames: 1200,
    ratedGames: 1100,
    avgRating: 2450,
    avgMoves: 41.5,
    whiteWins: 450,
    draws: 400,
    blackWins: 350,
    whiteWinPct: 37.5,
    drawPct: 33.3,
    blackWinPct: 29.2,
    decadeDist: [
      { label: '1980s', count: 300, pct: 25.0 },
      { label: '1990s', count: 500, pct: 41.7 },
      { label: '2000s', count: 400, pct: 33.3 },
    ],
    eloDist: [
      { label: '2400 - 2599', count: 800, pct: 72.7 },
      { label: '2600+', count: 300, pct: 27.3 },
    ],
    ecoSectionDist: [
      { label: 'A (Flank)', count: 200, pct: 16.7 },
      { label: 'B (Semi-Open)', count: 400, pct: 33.3 },
      { label: 'C (Open & French)', count: 350, pct: 29.2 },
      { label: 'D (Closed)', count: 150, pct: 12.5 },
      { label: 'E (Indian)', count: 100, pct: 8.3 },
    ],
    topEcos: [
      { label: 'B90', count: 120, pct: 10.0 },
      { label: 'C88', count: 90, pct: 7.5 },
    ],
    moveLengthDist: [
      { label: '21 - 40', count: 600, pct: 50.0 },
      { label: '41 - 60', count: 400, pct: 33.3 },
    ]
  }),
  GetRatingEvolution: vi.fn().mockImplementation((player: string) => {
    return Promise.resolve({
      player,
      totalGames: 20,
      ratedGames: 20,
      startRating: 2600,
      endRating: 2840,
      peakRating: 2851,
      peakDate: '1999.07.15',
      minRating: 2600,
      maxRating: 2851,
      points: [
        { date: '1985.11.09', year: 1985, rating: 2600, opponent: 'Karpov', opponentElo: 2720, result: '1-0', event: 'WCh', eco: 'B44', gameId: 1 },
        { date: '1999.07.15', year: 1999, rating: 2851, opponent: 'Topalov', opponentElo: 2750, result: '1-0', event: 'Wijk aan Zee', eco: 'B07', gameId: 15 },
        { date: '2005.03.10', year: 2005, rating: 2840, opponent: 'Anand', opponentElo: 2780, result: '1/2-1/2', event: 'Linares', eco: 'B90', gameId: 20 },
      ]
    })
  }),
  GetDatabasePlayers: vi.fn().mockResolvedValue(['Kasparov, Garry', 'Karpov, Anatoly']),
  GetPersonas: vi.fn().mockResolvedValue([
    { id: '1', displayName: 'Garry Kasparov', identifiers: ['Kasparov, Garry'] }
  ])
}))

describe('DatabaseStatsModal.vue', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders demographics overview cards, charts, and metrics', async () => {
    const wrapper = mount(DatabaseStatsModal, {
      props: { dbName: 'MasterGames.si4' }
    })
    await flushPromises()

    expect(wrapper.text()).toContain('Database Statistics & Demographics')
    expect(wrapper.text()).toContain('MasterGames.si4')
    expect(wrapper.text()).toContain('1200')
    expect(wrapper.text()).toContain('2450 Elo')
    expect(wrapper.text()).toContain('41.5 moves')
    expect(wrapper.text()).toContain('Elo Rating Distribution')
    expect(wrapper.text()).toContain('Chronological Decades')
    expect(AppApi.GetDatabaseDemographics).toHaveBeenCalled()
  })

  it('switches to Rating Tracker tab and displays player rating curve', async () => {
    const wrapper = mount(DatabaseStatsModal)
    await flushPromises()

    const ratingTabBtn = wrapper.findAll('.flex button').find(b => b.text().includes('Rating Tracker'))
    expect(ratingTabBtn).toBeDefined()
    await ratingTabBtn!.trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('Historical Elo Rating Curve')
    expect(wrapper.text()).toContain('Start Rating')
    expect(wrapper.text()).toContain('Peak Elo')
    expect(wrapper.text()).toContain('2851')
    expect(wrapper.find('svg').exists()).toBe(true)
  })

  it('queries new player rating evolution when search is submitted', async () => {
    const wrapper = mount(DatabaseStatsModal, {
      props: { initialPlayer: 'Kasparov, Garry' }
    })
    await flushPromises()

    const input = wrapper.find('input[placeholder*="Enter player name"]')
    expect(input.exists()).toBe(true)
    await input.setValue('Karpov, Anatoly')

    const trackBtn = wrapper.findAll('button').find(b => b.text().includes('Track Evolution'))
    expect(trackBtn).toBeDefined()
    await trackBtn!.trigger('click')
    await flushPromises()

    expect(AppApi.GetRatingEvolution).toHaveBeenCalledWith('Karpov, Anatoly')
  })
})
