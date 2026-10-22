import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import PersonaModal from './PersonaModal.vue'
import { persona } from '../../wailsjs/go/models'

const mockPersonas = [
  new persona.Persona({
    id: 'persona-1',
    displayName: 'Christopher Fair',
    identifiers: ['Fair, Christopher', 'ChristopherFair', 'QuenyenHighElf'],
    title: 'Candidate Master',
    bio: 'Club player and developer',
    estimatedElo: 2150,
    chessComUsername: 'ChristopherFair',
    lichessUsername: 'QuenyenHighElf',
    fideId: '1501234',
    databasePath: '/home/christopher/personas/ChristopherFair.si4',
    avatarColor: '#10b981',
  }),
  new persona.Persona({
    id: 'persona-2',
    displayName: 'Garry Kasparov',
    identifiers: ['Kasparov, Garry', 'Kasparov,G'],
    title: 'World Champion',
    estimatedElo: 2851,
  })
]

const mockStats = new persona.PersonaStatistics({
  personaId: 'persona-1',
  totalGames: 42,
  wins: 25,
  draws: 10,
  losses: 7,
  winRate: 71.4,
  peakElo: 2210,
  identifierStats: [
    new persona.IdentifierStat({
      identifier: 'Fair, Christopher',
      gameCount: 30,
      wins: 18,
      draws: 8,
      losses: 4,
      winRate: 73.3,
    }),
    new persona.IdentifierStat({
      identifier: 'ChristopherFair',
      gameCount: 12,
      wins: 7,
      draws: 2,
      losses: 3,
      winRate: 66.7,
    })
  ]
})

vi.mock('../../wailsjs/go/main/App', () => ({
  GetPersonas: vi.fn(async () => mockPersonas),
  GetPersona: vi.fn(async (id: string) => mockPersonas.find(p => p.id === id)),
  SavePersona: vi.fn(async (p: any) => p),
  DeletePersona: vi.fn(async (_id: string) => {}),
  AddPersonaIdentifier: vi.fn(async (id: string, ident: string) => {
    const p = mockPersonas.find(x => x.id === id)
    if (p && !p.identifiers.includes(ident)) p.identifiers.push(ident)
    return p
  }),
  RemovePersonaIdentifier: vi.fn(async (id: string, ident: string) => {
    const p = mockPersonas.find(x => x.id === id)
    if (p) p.identifiers = p.identifiers.filter(x => x !== ident)
    return p
  }),
  SuggestPersonaIdentifiers: vi.fn(async () => ['Fair, C.', 'Fair, Christopher A.']),
  GetPersonaStatistics: vi.fn(async () => mockStats),
  AggregatePersonaGames: vi.fn(async () => ({
    personaId: 'persona-1',
    sourceDatabase: 'current.si4',
    targetDatabase: 'persona.si4',
    newlyAdded: 5,
    duplicatesSkipped: 12,
    totalPersonaGames: 47,
  })),
  OpenPersonaDatabase: vi.fn(async () => {}),
  FetchPersonaOnlineGames: vi.fn(async () => ({
    profileName: 'Christopher Fair',
    totalDownloaded: 15,
    imported: 15,
    duplicatesSkipped: 3,
    lichessGames: 10,
    chessComGames: 5,
    errors: [],
  })),
}))

describe('PersonaModal.vue', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('loads personas on mount and selects the first persona', async () => {
    const wrapper = mount(PersonaModal, {
      props: { activeDbName: 'MyBase.si4' }
    })
    await flushPromises()

    expect(wrapper.text()).toContain('Christopher Fair')
    expect(wrapper.text()).toContain('Garry Kasparov')
    expect(wrapper.find('input[placeholder*="John Smith"]').exists()).toBe(true)
    const displayNameInput = wrapper.find('input[placeholder*="John Smith"]').element as HTMLInputElement
    expect(displayNameInput.value).toBe('Christopher Fair')
  })

  it('filters personas list by search query', async () => {
    const wrapper = mount(PersonaModal)
    await flushPromises()

    const searchInput = wrapper.find('input[placeholder="Filter personas..."]')
    await searchInput.setValue('Kasparov')

    expect(wrapper.text()).toContain('Garry Kasparov')
    expect(wrapper.text()).not.toContain('Christopher Fair')
  })

  it('displays consolidated statistics for selected persona', async () => {
    const wrapper = mount(PersonaModal)
    await flushPromises()

    expect(wrapper.text()).toContain('Consolidated Record (42 Games Across All Aliases)')
    expect(wrapper.text()).toContain('+25 =10 -7')
    expect(wrapper.text()).toContain('71.4%')
    expect(wrapper.text()).toContain('2210')
  })

  it('displays suggested candidate database matches', async () => {
    const wrapper = mount(PersonaModal)
    await flushPromises()

    expect(wrapper.text()).toContain('Found Candidate Names in Open Database:')
    expect(wrapper.text()).toContain('Fair, C.')
    expect(wrapper.text()).toContain('Fair, Christopher A.')
  })

  it('emits openPersonaDb when Open Database is clicked', async () => {
    const wrapper = mount(PersonaModal)
    await flushPromises()

    const openDbBtn = wrapper.findAll('button').find(b => b.text().includes('Open Database'))
    expect(openDbBtn).toBeDefined()
    await openDbBtn!.trigger('click')
    await flushPromises()

    expect(wrapper.emitted('openPersonaDb')).toBeTruthy()
    expect(wrapper.emitted('close')).toBeTruthy()
  })

  it('emits sparPersona when Spar As/Against Persona is clicked', async () => {
    const wrapper = mount(PersonaModal)
    await flushPromises()

    const sparBtn = wrapper.findAll('button').find(b => b.text().includes('Spar As/Against Persona'))
    expect(sparBtn).toBeDefined()
    await sparBtn!.trigger('click')

    expect(wrapper.emitted('sparPersona')).toBeTruthy()
    expect(wrapper.emitted('sparPersona')![0][0]).toMatchObject({
      displayName: 'Christopher Fair',
    })
    expect(wrapper.emitted('close')).toBeTruthy()
  })

  it('creates a blank new persona form when New Persona is clicked', async () => {
    const wrapper = mount(PersonaModal)
    await flushPromises()

    const newBtn = wrapper.findAll('button').find(b => b.text().includes('New Persona'))
    expect(newBtn).toBeDefined()
    await newBtn!.trigger('click')

    const displayNameInput = wrapper.find('input[placeholder*="John Smith"]').element as HTMLInputElement
    expect(displayNameInput.value).toBe('New Player Persona')
  })

  it('fetches online games when Fetch Online Games button is clicked', async () => {
    const wrapper = mount(PersonaModal)
    await flushPromises()

    const fetchBtn = wrapper.findAll('button').find(b => b.text().includes('Fetch Online Games'))
    expect(fetchBtn).toBeDefined()
    expect(fetchBtn!.attributes('disabled')).toBeUndefined()

    await fetchBtn!.trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('Fetched 15 game(s) into persona database')
    expect(wrapper.text()).toContain('5 Chess.com, 10 Lichess')
  })

  it('auto-fetches online games upon saving when auto-fetch is checked and online usernames exist', async () => {
    const wrapper = mount(PersonaModal)
    await flushPromises()

    const saveBtn = wrapper.findAll('button').find(b => b.text().includes('Save Persona'))
    expect(saveBtn).toBeDefined()

    await saveBtn!.trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('Fetched 15 game(s) into persona database')
  })
})
