import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import GettingStartedWizardModal from './GettingStartedWizardModal.vue'
import * as AppAPI from '../../wailsjs/go/main/App'

vi.mock('../../wailsjs/go/main/App', () => ({
  GetUserProfileStatus: vi.fn(),
  SaveUserProfileSetup: vi.fn(),
  SyncUserHomeGames: vi.fn(),
  GetPolyglotBooks: vi.fn(),
  GetRepertoires: vi.fn(),
  CreateRepertoire: vi.fn(),
  CompileRepertoireToPolyglotBook: vi.fn(),
  GetPersonas: vi.fn(),
  SavePersona: vi.fn(),
  BuildPersonaOpeningBook: vi.fn(),
  ExportAntiPersonaPolyglotBook: vi.fn(),
  AssignPersonaBooks: vi.fn(),
  OpenPolyglotBookDialog: vi.fn(),
  OpenFileDialog: vi.fn(),
  ImportStandardDatabase: vi.fn(),
  BuildConsolidatedDatabase: vi.fn(),
  GetDiscoveredDatabases: vi.fn(),
  SetDefaultDatabase: vi.fn(),
  SetReferenceDatabase: vi.fn(),
  EnsurePersonalUserDatabase: vi.fn(),
  FetchPersonaOnlineGames: vi.fn(),
  AggregatePersonaGames: vi.fn(),
}))

describe('GettingStartedWizardModal.vue', () => {
  beforeEach(() => {
    vi.clearAllMocks()

    vi.mocked(AppAPI.GetUserProfileStatus).mockResolvedValue({
      onboardingCompleted: false,
      displayName: 'Christopher Fair',
      title: 'CM',
      estimatedElo: 1850,
      fideId: '',
      lichessUsername: 'chrisfair',
      chessComUsername: 'chrisfair',
      aliases: ['Fair, Christopher', 'Christopher Fair', 'chrisfair'],
      personaId: 'p-chris',
      homeDbPath: '/tmp/My_Games',
      homeDbName: 'My_Games',
      homeDbGames: 42,
      referenceDbPath: '/tmp/Consolidated_Reference',
      referenceDbName: 'Consolidated_Reference',
      referenceDbGames: 15000,
      focusPolyglotBookPath: '/tmp/books/My_Repertoire.bin',
      focusWhiteRepertoireId: 'rep-w',
      focusBlackRepertoireId: 'rep-b',
    } as any)

    vi.mocked(AppAPI.GetPolyglotBooks).mockResolvedValue([
      '/tmp/books/My_Repertoire.bin',
      '/tmp/books/Anti_Club_Rival.bin',
    ])

    vi.mocked(AppAPI.GetDiscoveredDatabases).mockResolvedValue([
      { name: 'My_Games', basePath: '/tmp/My_Games', numGames: 42, category: 'Personal' },
      { name: 'Consolidated_Reference', basePath: '/tmp/Consolidated_Reference', numGames: 15000, category: 'Reference' },
    ] as any)

    vi.mocked(AppAPI.GetRepertoires).mockResolvedValue([
      { id: 'rep-w', name: 'White Vienna', color: 'w', numMoves: 12 },
      { id: 'rep-b', name: 'Black Caro-Kann', color: 'b', numMoves: 8 },
    ] as any)

    vi.mocked(AppAPI.GetPersonas).mockResolvedValue([
      {
        id: 'p-chris',
        displayName: 'Christopher Fair',
        identifiers: ['Fair, Christopher', 'chrisfair'],
        estimatedElo: 1850,
        isUserProfile: true,
        bookPath: '/tmp/books/Christopher_Fair_Book.bin',
      },
      {
        id: 'p-rival',
        displayName: 'Club Rival',
        identifiers: ['Rival, Club'],
        estimatedElo: 2000,
        isUserProfile: false,
        bookPath: '',
      },
    ] as any)
  })

  it('loads user profile status and navigates across all 5 wizard steps', async () => {
    const wrapper = mount(GettingStartedWizardModal, {
      props: {
        discoveredDatabases: [
          { name: 'My_Games', basePath: '/tmp/My_Games', numGames: 42, category: 'Personal' },
          { name: 'Consolidated_Reference', basePath: '/tmp/Consolidated_Reference', numGames: 15000, category: 'Reference' },
        ],
      },
    })

    await flushPromises()

    // Step 1: Player Profile
    expect(wrapper.text()).toContain('Why Define Your Player Identity?')
    const nameInput = wrapper.find('[data-testid="wizard-display-name-input"]')
    expect((nameInput.element as HTMLInputElement).value).toBe('Christopher Fair')

    vi.mocked(AppAPI.SaveUserProfileSetup).mockResolvedValue({
      onboardingCompleted: false,
      displayName: 'Christopher Fair',
      homeDbName: 'My_Games',
      homeDbGames: 42,
    } as any)

    const nextBtn = wrapper.find('[data-testid="wizard-next-btn"]')
    await nextBtn.trigger('click')
    await flushPromises()

    expect(AppAPI.SaveUserProfileSetup).toHaveBeenCalledTimes(1)

    // Step 2: Home Database & Migrating Games
    expect(wrapper.text()).toContain('1. Home DB (My_Games)')
    expect(wrapper.text()).toContain('42 games')

    vi.mocked(AppAPI.SyncUserHomeGames).mockResolvedValue({
      totalDownloaded: 10,
      imported: 8,
      duplicatesSkipped: 2,
      chessComGames: 4,
      lichessGames: 6,
      errors: [],
    } as any)

    await wrapper.find('[data-testid="wizard-sync-online-btn"]').trigger('click')
    await flushPromises()
    expect(AppAPI.SyncUserHomeGames).toHaveBeenCalledWith(250)

    // Go to Step 3: My Focus Books
    await wrapper.find('[data-testid="wizard-step-tab-3"]').trigger('click')
    await flushPromises()
    expect(wrapper.text()).toContain('How "Books" Work for Your Own Study')

    vi.mocked(AppAPI.CompileRepertoireToPolyglotBook).mockResolvedValue('/tmp/books/White_Vienna.bin' as any)
    await wrapper.find('[data-testid="wizard-compile-white-rep-btn"]').trigger('click')
    await flushPromises()
    expect(AppAPI.CompileRepertoireToPolyglotBook).toHaveBeenCalledWith('rep-w', true)

    // Go to Step 4: Books for Any Persona
    await wrapper.find('[data-testid="wizard-step-tab-4"]').trigger('click')
    await flushPromises()
    expect(wrapper.text()).toContain('How to Define a Book for Any Persona')

    vi.mocked(AppAPI.BuildPersonaOpeningBook).mockResolvedValue('/tmp/books/Club_Rival_Book.bin' as any)
    await wrapper.find('[data-testid="wizard-build-persona-book-btn"]').trigger('click')
    await flushPromises()
    expect(AppAPI.BuildPersonaOpeningBook).toHaveBeenCalled()

    // Go to Step 5: How to Use NewScid
    await wrapper.find('[data-testid="wizard-step-tab-5"]').trigger('click')
    await flushPromises()
    expect(wrapper.text()).toContain('Workstation Ready for Christopher Fair')

    await wrapper.find('[data-testid="wizard-complete-btn"]').trigger('click')
    await flushPromises()
    expect(wrapper.emitted('close')).toBeTruthy()
  })
})
