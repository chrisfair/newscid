import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import PlayerSimulationModal from './PlayerSimulationModal.vue'
import { playerbot, persona } from '../../wailsjs/go/models'

const mockPlayers = ['Kasparov, Garry', 'Carlsen, Magnus', 'Fischer, Robert J.']

const mockPersonaObj = new persona.Persona({
  id: 'persona-1',
  displayName: 'Christopher Fair',
  identifiers: ['Fair, Christopher', 'ChristopherFair'],
  estimatedElo: 2150,
  title: 'CM',
})

const mockPersonality = new playerbot.PlayerPersonality({
  name: 'Kasparov, Garry',
  estimatedElo: 2851,
  peakElo: 2851,
  totalGames: 120,
  wins: 80,
  draws: 30,
  losses: 10,
  winRate: 79.2,
  topWhiteECOs: ['B90', 'E97'],
  topBlackECOs: ['B80', 'E99'],
})

const mockPersonaPersonality = new playerbot.PlayerPersonality({
  name: 'Christopher Fair',
  estimatedElo: 2150,
  peakElo: 2200,
  totalGames: 50,
  wins: 30,
  draws: 10,
  losses: 10,
  winRate: 70.0,
  topWhiteECOs: ['C50', 'B01'],
  topBlackECOs: ['B20', 'C00'],
})

vi.mock('../../wailsjs/go/main/App', () => ({
  GetDatabasePlayers: vi.fn(async () => mockPlayers),
  GetPlayerPersonality: vi.fn(async (_name: string) => mockPersonality),
  GetPersonas: vi.fn(async () => [mockPersonaObj]),
  GetPersonaPersonality: vi.fn(async (_id: string) => mockPersonaPersonality),
}))

describe('PlayerSimulationModal.vue', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders opponent tabs and loads initial database player personality', async () => {
    const wrapper = mount(PlayerSimulationModal, {
      props: { activeDbName: 'Bases/Standard.si4' }
    })
    await flushPromises()

    expect(wrapper.text()).toContain('Database Opponents')
    expect(wrapper.text()).toContain('Unified Personas')
    expect(wrapper.text()).toContain('Kasparov, Garry')
    expect(wrapper.text()).toContain('2851 Elo')
    expect(wrapper.text()).toContain('79.2%')
  })

  it('switches to Unified Personas tab and displays personas', async () => {
    const wrapper = mount(PlayerSimulationModal)
    await flushPromises()

    const personasTabBtn = wrapper.findAll('button').find(b => b.text().includes('Unified Personas'))
    expect(personasTabBtn).toBeDefined()
    await personasTabBtn!.trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('Choose Persona to Spar Against:')
    expect(wrapper.text()).toContain('Christopher Fair')
    expect(wrapper.text()).toContain('~2150 Elo')
  })

  it('analyzes persona and emits startMatch with persona personality', async () => {
    const wrapper = mount(PlayerSimulationModal)
    await flushPromises()

    // Switch to personas tab
    const personasTabBtn = wrapper.findAll('button').find(b => b.text().includes('Unified Personas'))
    await personasTabBtn!.trigger('click')
    await flushPromises()

    // Click on Christopher Fair persona card
    const personaCard = wrapper.findAll('button').find(b => b.text().includes('Christopher Fair'))
    expect(personaCard).toBeDefined()
    await personaCard!.trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('Christopher Fair')
    expect(wrapper.text()).toContain('2150 Elo')
    expect(wrapper.text()).toContain('70.0%')

    // Click "Start Sparring Match"
    const startBtn = wrapper.findAll('button').find(b => b.text().includes('Start Sparring Match'))
    expect(startBtn).toBeDefined()
    await startBtn!.trigger('click')

    expect(wrapper.emitted('startMatch')).toBeTruthy()
    const payload = wrapper.emitted('startMatch')![0][0] as any
    expect(payload.playerName).toBe('Christopher Fair')
    expect(payload.personality.estimatedElo).toBe(2150)
  })

  it('initializes in Personas mode when initialPersona prop is provided', async () => {
    const wrapper = mount(PlayerSimulationModal, {
      props: { initialPersona: mockPersonaObj }
    })
    await flushPromises()

    expect(wrapper.text()).toContain('Christopher Fair')
    expect(wrapper.text()).toContain('2150 Elo')
  })
})
