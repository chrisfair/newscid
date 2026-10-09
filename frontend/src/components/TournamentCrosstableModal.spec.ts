import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import TournamentCrosstableModal from './TournamentCrosstableModal.vue'
import * as App from '../../wailsjs/go/main/App'
import * as runtime from '../../wailsjs/runtime/runtime'

vi.mock('../../wailsjs/go/main/App', () => ({
  GetTournamentEvents: vi.fn(),
  GetTournamentCrosstable: vi.fn(),
  ExportCrosstableText: vi.fn(),
}))

vi.mock('../../wailsjs/runtime/runtime', () => ({
  ClipboardSetText: vi.fn(),
}))

describe('TournamentCrosstableModal.vue', () => {
  const mockEvents = [
    {
      event: 'Tata Steel Masters 2024',
      site: 'Wijk aan Zee NED',
      date: '2024.01.13',
      year: 2024,
      numGames: 6,
      numPlayers: 4,
      rounds: 3,
      type: 'Round-Robin',
    },
    {
      event: 'Reykjavik Open 2024',
      site: 'Reykjavik ISL',
      date: '2024.03.15',
      year: 2024,
      numGames: 20,
      numPlayers: 10,
      rounds: 5,
      type: 'Swiss',
    },
  ]

  const mockCrosstable = {
    event: 'Tata Steel Masters 2024',
    site: 'Wijk aan Zee NED',
    date: '2024.01.13',
    category: 21,
    averageElo: 2760,
    numPlayers: 4,
    numGames: 6,
    numRounds: 3,
    type: 'Round-Robin',
    standings: [
      {
        rank: 1,
        playerName: 'Wei Yi',
        title: 'GM',
        elo: 2740,
        points: 2.5,
        played: 3,
        wins: 2,
        draws: 1,
        losses: 0,
        sonnebornBerger: 3.25,
        buchholz: 4.5,
        buchholzCut1: 3.0,
        performanceRating: 2850,
        roundResults: [
          { round: '1', opponentRank: 2, opponentName: 'Nakamura, Hikaru', color: 'w', result: '1', score: 1.0, gameId: 101 },
          { round: '2', opponentRank: 3, opponentName: 'Praggnanandhaa, R', color: 'b', result: '½', score: 0.5, gameId: 102 },
          { round: '3', opponentRank: 4, opponentName: 'Firouzja, Alireza', color: 'w', result: '1', score: 1.0, gameId: 103 },
        ],
        scoresAgainst: { 'Nakamura, Hikaru': 1.0, 'Praggnanandhaa, R': 0.5, 'Firouzja, Alireza': 1.0 },
      },
      {
        rank: 2,
        playerName: 'Nakamura, Hikaru',
        title: 'GM',
        elo: 2780,
        points: 2.0,
        played: 3,
        wins: 2,
        draws: 0,
        losses: 1,
        sonnebornBerger: 2.5,
        buchholz: 4.0,
        buchholzCut1: 2.5,
        performanceRating: 2790,
        roundResults: [
          { round: '1', opponentRank: 1, opponentName: 'Wei Yi', color: 'b', result: '0', score: 0.0, gameId: 101 },
          { round: '2', opponentRank: 4, opponentName: 'Firouzja, Alireza', color: 'w', result: '1', score: 1.0, gameId: 104 },
          { round: '3', opponentRank: 3, opponentName: 'Praggnanandhaa, R', color: 'w', result: '1', score: 1.0, gameId: 105 },
        ],
        scoresAgainst: { 'Wei Yi': 0.0, 'Firouzja, Alireza': 1.0, 'Praggnanandhaa, R': 1.0 },
      },
      {
        rank: 3,
        playerName: 'Praggnanandhaa, R',
        title: 'GM',
        elo: 2720,
        points: 1.0,
        played: 3,
        wins: 0,
        draws: 2,
        losses: 1,
        sonnebornBerger: 1.5,
        buchholz: 4.5,
        buchholzCut1: 2.5,
        performanceRating: 2680,
        roundResults: [],
        scoresAgainst: {},
      },
      {
        rank: 4,
        playerName: 'Firouzja, Alireza',
        title: 'GM',
        elo: 2760,
        points: 0.5,
        played: 3,
        wins: 0,
        draws: 1,
        losses: 2,
        sonnebornBerger: 0.5,
        buchholz: 5.0,
        buchholzCut1: 3.0,
        performanceRating: 2540,
        roundResults: [],
        scoresAgainst: {},
      },
    ],
    matrix: [
      [
        { result: 'X', score: 0, color: '', gameId: 0 },
        { result: '1', score: 1.0, color: 'w', gameId: 101 },
        { result: '½', score: 0.5, color: 'b', gameId: 102 },
        { result: '1', score: 1.0, color: 'w', gameId: 103 },
      ],
      [
        { result: '0', score: 0, color: 'b', gameId: 101 },
        { result: 'X', score: 0, color: '', gameId: 0 },
        { result: '1', score: 1.0, color: 'w', gameId: 105 },
        { result: '1', score: 1.0, color: 'w', gameId: 104 },
      ],
      [
        { result: '½', score: 0.5, color: 'w', gameId: 102 },
        { result: '0', score: 0, color: 'b', gameId: 105 },
        { result: 'X', score: 0, color: '', gameId: 0 },
        { result: '½', score: 0.5, color: 'w', gameId: 106 },
      ],
      [
        { result: '0', score: 0, color: 'b', gameId: 103 },
        { result: '0', score: 0, color: 'b', gameId: 104 },
        { result: '½', score: 0.5, color: 'b', gameId: 106 },
        { result: 'X', score: 0, color: '', gameId: 0 },
      ],
    ],
    crossTableText: 'Tata Steel Masters 2024\n1. Wei Yi 2.5\n2. Nakamura, Hikaru 2.0',
  }

  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(App.GetTournamentEvents).mockResolvedValue(mockEvents as any)
    vi.mocked(App.GetTournamentCrosstable).mockResolvedValue(mockCrosstable as any)
    vi.mocked(App.ExportCrosstableText).mockResolvedValue(mockCrosstable.crossTableText)
  })

  it('renders modal and loads tournament events when show is true', async () => {
    const wrapper = mount(TournamentCrosstableModal, {
      props: {
        show: true,
      },
    })
    await flushPromises()

    expect(App.GetTournamentEvents).toHaveBeenCalled()
    expect(App.GetTournamentCrosstable).toHaveBeenCalledWith(
      'Tata Steel Masters 2024',
      'Wijk aan Zee NED',
      '2024.01.13'
    )
    expect(wrapper.text()).toContain('Tournament Crosstables & Standings')
    expect(wrapper.text()).toContain('Tata Steel Masters 2024')
    expect(wrapper.text()).toContain('Category 21')
    expect(wrapper.text()).toContain('Avg Elo: 2760')
    expect(wrapper.text()).toContain('Wei Yi')
    expect(wrapper.text()).toContain('Nakamura, Hikaru')
  })

  it('switches between Matrix, Swiss, Standings, and Text views', async () => {
    const wrapper = mount(TournamentCrosstableModal, {
      props: {
        show: true,
      },
    })
    await flushPromises()

    // Standings view
    const standingsTab = wrapper.findAll('button').find((b) => b.text().includes('Standings'))
    expect(standingsTab).toBeDefined()
    await standingsTab!.trigger('click')
    expect(wrapper.text()).toContain('Sonneborn-Berger')
    expect(wrapper.text()).toContain('Perf. Rating (TPR)')

    // Swiss view
    const swissTab = wrapper.findAll('button').find((b) => b.text().includes('Swiss'))
    expect(swissTab).toBeDefined()
    await swissTab!.trigger('click')
    expect(wrapper.text()).toContain('R1')
    expect(wrapper.text()).toContain('R2')
    expect(wrapper.text()).toContain('R3')

    // Text view
    const textTab = wrapper.findAll('button').find((b) => b.text().includes('Text / ASCII'))
    expect(textTab).toBeDefined()
    await textTab!.trigger('click')
    expect(wrapper.text()).toContain('Plain-text ASCII tournament cross-table')
    expect(wrapper.text()).toContain('Wei Yi 2.5')
  })

  it('emits loadGame and close when clicking an interactive result cell', async () => {
    const wrapper = mount(TournamentCrosstableModal, {
      props: {
        show: true,
      },
    })
    await flushPromises()

    // Matrix view: find cell for Game #101
    const cell = wrapper.findAll('td').find((td) => td.attributes('title')?.includes('Game #101'))
    expect(cell).toBeDefined()
    await cell!.trigger('click')

    expect(wrapper.emitted('loadGame')).toBeTruthy()
    expect(wrapper.emitted('loadGame')![0]).toEqual([101])
    expect(wrapper.emitted('close')).toBeTruthy()
  })

  it('copies crosstable text to clipboard', async () => {
    const wrapper = mount(TournamentCrosstableModal, {
      props: {
        show: true,
      },
    })
    await flushPromises()

    const copyBtn = wrapper.findAll('button').find((b) => b.text().includes('Copy Crosstable'))
    expect(copyBtn).toBeDefined()
    await copyBtn!.trigger('click')
    await flushPromises()

    expect(runtime.ClipboardSetText).toHaveBeenCalledWith(
      expect.stringContaining('Tata Steel Masters 2024')
    )
  })

  it('filters standings by player search query', async () => {
    const wrapper = mount(TournamentCrosstableModal, {
      props: {
        show: true,
      },
    })
    await flushPromises()

    // Switch to Standings tab
    const standingsTab = wrapper.findAll('button').find((b) => b.text().includes('Standings'))
    await standingsTab!.trigger('click')

    const searchInput = wrapper.find('input[placeholder="Filter players in tournament..."]')
    await searchInput.setValue('Nakamura')

    expect(wrapper.text()).toContain('Nakamura, Hikaru')
    expect(wrapper.text()).not.toContain('Firouzja, Alireza')
  })
})
