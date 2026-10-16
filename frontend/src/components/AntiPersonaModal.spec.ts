import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import AntiPersonaModal from './AntiPersonaModal.vue'

vi.mock('../../wailsjs/go/main/App', () => ({
  GetPersonas: vi.fn().mockResolvedValue([
    {
      id: 'p-kasparov',
      displayName: 'Garry Kasparov',
      identifiers: ['Kasparov, Garry', 'GarryKasparov'],
    },
  ]),
  GetDatabasePlayers: vi.fn().mockResolvedValue(['Kasparov, Garry', 'Karpov, Anatoly']),
  EvaluatePersonaWeaknesses: vi.fn().mockResolvedValue({
    personaId: 'p-kasparov',
    displayName: 'Garry Kasparov',
    targetColor: 'white',
    totalGamesAnalyzed: 12,
    personaRecord: { total: 12, wins: 8, draws: 2, losses: 2, winRate: 75.0 },
    vulnerabilities: [
      {
        ply: 4,
        eco: 'B22',
        openingName: 'Sicilian Defense: Alapin Variation',
        moveSan: 'd5',
        moveUci: 'd7d5',
        moveHistorySan: ['e4', 'c5', 'c3', 'd5'],
        fen: 'rnbqkbnr/pp2pppp/8/3p4/4P3/2P5/PP1P1PPP/RNBQKBNR w KQkq - 0 3',
        personaRecord: { total: 4, wins: 0, draws: 1, losses: 3, winRate: 12.5 },
        exploitationScore: 92.5,
        tacticalInsight: 'Opponent scores only 12.5% across 4 games. Playing d5 creates immediate structural disadvantage.',
        suggestedReplies: ['exd5 (3)', 'e5 (1)'],
      },
    ],
    topRefutationLines: ['[B22] e4 c5 c3 d5 -> d5 (Score: 12.5%)'],
    pgnRepertoire: '[Event "Anti-Persona Weaponized Repertoire"]\n1. e4 c5 2. c3 d5! *\n',
    polyglotEntryCount: 16,
    generatedAt: 1700000000000,
  }),
  EvaluateOpponentWeaknesses: vi.fn().mockResolvedValue({
    personaId: '',
    displayName: 'Karpov, Anatoly',
    targetColor: 'white',
    totalGamesAnalyzed: 10,
    personaRecord: { total: 10, wins: 5, draws: 3, losses: 2, winRate: 65.0 },
    vulnerabilities: [],
    topRefutationLines: [],
    pgnRepertoire: '[Event "Anti-Persona Weaponized Repertoire"]\n*\n',
    polyglotEntryCount: 8,
    generatedAt: 1700000000000,
  }),
  ExportAntiPersonaPolyglotBook: vi.fn().mockResolvedValue('/home/user/.local/share/newscid/books/Anti_Garry_Kasparov.bin'),
  ExportAntiPersonaPGN: vi.fn().mockResolvedValue('/home/user/.local/share/newscid/repertoires/Anti_Garry_Kasparov.pgn'),
  SaveBookFileDialog: vi.fn().mockResolvedValue('/tmp/Anti_Book.bin'),
  SavePGNFileDialog: vi.fn().mockResolvedValue('/tmp/Anti_Repertoire.pgn'),
}))

describe('AntiPersonaModal.vue', () => {
  it('renders modal and generates weaponized anti-persona opening report', async () => {
    const wrapper = mount(AntiPersonaModal, {
      props: {
        initialPersonaId: 'p-kasparov',
      },
    })

    // Wait for mock async evaluation
    await new Promise((r) => setTimeout(r, 30))
    await wrapper.vm.$nextTick()

    expect(wrapper.text()).toContain('Opponent Opening Exploiter')
    expect(wrapper.text()).toContain('Analyzed Games')
    expect(wrapper.text()).toContain('12')
    expect(wrapper.text()).toContain('Critical Weaknesses')
    expect(wrapper.text()).toContain('1 Lines')
    expect(wrapper.text()).toContain('Weaponized Book')
    expect(wrapper.text()).toContain('16 Entries')

    // Check vulnerability card details
    expect(wrapper.text()).toContain('B22')
    expect(wrapper.text()).toContain('Sicilian Defense: Alapin Variation')
    expect(wrapper.text()).toContain('d5!')
    expect(wrapper.text()).toContain('12.5% win rate')
    expect(wrapper.text()).toContain('92.5')
    expect(wrapper.text()).toContain('Exploitation Tactical Insight')
  })

  it('emits trainRepertoire and sparBot events', async () => {
    const wrapper = mount(AntiPersonaModal, {
      props: {
        initialPersonaId: 'p-kasparov',
      },
    })

    await new Promise((r) => setTimeout(r, 30))
    await wrapper.vm.$nextTick()

    // Find and click Train Repertoire button
    const trainBtn = wrapper.findAll('button').find((b) => b.text().includes('Train Repertoire'))
    expect(trainBtn).toBeDefined()
    await trainBtn!.trigger('click')

    expect(wrapper.emitted('trainRepertoire')).toBeTruthy()
    const trainArgs = wrapper.emitted('trainRepertoire')![0]
    expect(trainArgs[0]).toContain('Anti-Persona Weaponized Repertoire')

    // Find and click Spar Against Persona button
    const sparBtn = wrapper.findAll('button').find((b) => b.text().includes('Spar Against Persona'))
    expect(sparBtn).toBeDefined()
    await sparBtn!.trigger('click')

    expect(wrapper.emitted('sparBot')).toBeTruthy()
  })
})
