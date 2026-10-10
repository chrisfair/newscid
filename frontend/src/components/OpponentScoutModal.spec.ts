import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import OpponentScoutModal from './OpponentScoutModal.vue'

vi.mock('../../wailsjs/go/main/App', () => ({
  GetDatabasePlayers: vi.fn().mockResolvedValue(['Carlsen, Magnus', 'Nakamura, Hikaru']),
  GenerateOpponentScoutingReport: vi.fn().mockResolvedValue({
    playerName: 'Carlsen, Magnus',
    totalGames: 10,
    whiteRecord: { total: 5, wins: 4, draws: 1, losses: 0, winRate: 90.0 },
    blackRecord: { total: 5, wins: 3, draws: 2, losses: 0, winRate: 80.0 },
    topOpeningsAsWhite: [{ eco: 'B90', games: 3, wins: 3, losses: 0, draws: 0, winRate: 100.0 }],
    topOpeningsAsBlack: [],
    sharpestLines: [{ eco: 'B90', games: 3, wins: 3, losses: 0, draws: 0, winRate: 100.0 }],
    weakestLines: [],
    repertoireRecommendations: ['As White, primary weapon is B90.'],
  }),
}))

describe('OpponentScoutModal.vue', () => {
  it('renders modal and generates opponent scouting dossier', async () => {
    const wrapper = mount(OpponentScoutModal, {
      props: {
        initialPlayer: 'Carlsen, Magnus',
      },
    })

    // wait for mock fetch
    await new Promise((r) => setTimeout(r, 20))
    await wrapper.vm.$nextTick()

    expect(wrapper.text()).toContain('Opponent Scouting Card')
    expect(wrapper.text()).toContain('Record as White')
    expect(wrapper.text()).toContain('90%')
    expect(wrapper.text()).toContain('B90')
    expect(wrapper.text()).toContain('primary weapon is B90')
  })
})
