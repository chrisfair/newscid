import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import AcademyModal from './AcademyModal.vue'

const mockOverview = {
  profile: {
    playerName: 'Challenger',
    rating: 950,
    peakRating: 980,
    currentTier: 1,
    unlockedTiers: [1],
    records: { t1_toby: { wins: 2, draws: 0, losses: 1 } },
    completedLessons: {},
    bossExams: {},
    customPersonalities: []
  },
  tiers: [
    { tier: 1, name: 'Coffeehouse', ratingRange: '700–1000', targetElo: 950, description: 'Tier 1 desc', bossId: 't1_milo_boss' },
    { tier: 2, name: 'Club Floor', ratingRange: '1000–1300', targetElo: 1250, description: 'Tier 2 desc', bossId: 't2_vera_boss' }
  ],
  personalities: [
    {
      id: 't1_toby',
      name: 'Toby',
      title: 'Pawn Pusher',
      tier: 1,
      isBoss: false,
      elo: 750,
      avatarColor: '#f59e0b',
      avatarSymbol: '♙',
      styleTag: 'Overextends Pawns',
      bio: 'Pushes pawns early.',
      source: 'builtin',
      sliders: { aggression: 65, tradeTendency: 35, tacticalSharpness: 20, positionalSolidity: 20, blunderRate: 70 }
    }
  ],
  lessons: [
    {
      id: 't1_piece_safety',
      tier: 1,
      order: 1,
      title: 'Hanging Pieces',
      category: 'tactics',
      estimatedMinutes: 4,
      summary: 'Scan for undefended pieces.',
      steps: [
        {
          fen: '6k1/5Q2/6K1/8/8/8/8/8 w - - 0 1',
          title: 'Mate in 1',
          instruction: 'Deliver checkmate!',
          expectedMoves: ['f7g7'],
          hint: 'Play Qg7#',
          successFeedback: 'Checkmate!'
        }
      ]
    }
  ]
}

describe('AcademyModal.vue', () => {
  beforeEach(() => {
    ;(window as any).go = {
      main: {
        App: {
          GetAcademyOverview: vi.fn().mockResolvedValue(mockOverview),
          SearchDatabasePlayers: vi.fn().mockResolvedValue([{ name: 'Kasparov, Garry', gameCount: 42, peakElo: 2851 }]),
          BuildAcademyPersonalityFromDB: vi.fn().mockResolvedValue({
            id: 'db_kasparov',
            name: 'Kasparov, Garry',
            title: 'Database Clone',
            tier: 6,
            isBoss: false,
            elo: 2851,
            avatarColor: '#06b6d4',
            avatarSymbol: '♜',
            styleTag: 'Tactical Attacker',
            bio: 'Built from 42 games.',
            source: 'database',
            sourceGames: 42,
            sliders: { aggression: 90, tradeTendency: 40, tacticalSharpness: 98, positionalSolidity: 90, blunderRate: 2 }
          }),
          CompleteAcademyLesson: vi.fn().mockResolvedValue({
            ...mockOverview.profile,
            completedLessons: { t1_piece_safety: true }
          }),
          TestOutAcademyTier: vi.fn().mockResolvedValue({
            ...mockOverview.profile,
            currentTier: 2,
            unlockedTiers: [1, 2]
          }),
          SaveCustomPersonality: vi.fn().mockResolvedValue({
            id: 'custom_bot',
            name: 'Club Bot',
            elo: 1400,
            source: 'custom'
          }),
          DeleteCustomPersonality: vi.fn().mockResolvedValue(undefined)
        }
      }
    }
  })

  it('renders Career Ladder tiers, lessons, and personalities', async () => {
    const wrapper = mount(AcademyModal, {
      props: { show: true, dbName: 'master.si4', dbNumGames: 100 }
    })
    await flushPromises()
    expect(wrapper.text()).toContain('NewScid Academy & Career Ladder')
    expect(wrapper.text()).toContain('Tier 1: Coffeehouse')
    expect(wrapper.text()).toContain('Hanging Pieces')
    expect(wrapper.text()).toContain('Toby')
  })

  it('builds a personality from the open database in the guided DB tab', async () => {
    const wrapper = mount(AcademyModal, {
      props: { show: true, initialTab: 'build-db', dbName: 'master.si4', dbNumGames: 42 }
    })
    await flushPromises()
    expect(wrapper.text()).toContain('Kasparov, Garry')
    const playerBtn = wrapper.findAll('button').find(b => b.text().includes('Kasparov, Garry'))
    expect(playerBtn).toBeTruthy()
    await playerBtn!.trigger('click')
    await flushPromises()
    expect((window as any).go.main.App.BuildAcademyPersonalityFromDB).toHaveBeenCalledWith('Kasparov, Garry')
    expect(wrapper.text()).toContain('Ready to Play!')
  })

  it('opens an interactive lesson and emits startCareerGame on challenge', async () => {
    const wrapper = mount(AcademyModal, {
      props: { show: true }
    })
    await flushPromises()
    const challengeBtn = wrapper.findAll('button').find(b => b.text().includes('Challenge'))
    expect(challengeBtn).toBeTruthy()
    await challengeBtn!.trigger('click')
    expect(wrapper.emitted('startCareerGame')).toBeTruthy()
  })
})
