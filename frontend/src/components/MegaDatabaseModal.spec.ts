import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import MegaDatabaseModal from './MegaDatabaseModal.vue'
import * as AppBindings from '../../wailsjs/go/main/App'

vi.mock('../../wailsjs/go/main/App', () => ({
  GetMegaDatabaseCatalog: vi.fn(),
  FetchLichessDatabaseFeed: vi.fn(),
  ImportMegaDatabase: vi.fn(),
  BuildConsolidatedDatabase: vi.fn(),
  ConsolidateOnlineCatalogDatabases: vi.fn(),
  CancelActiveOperation: vi.fn(),
}))

vi.mock('../../wailsjs/runtime/runtime', () => ({
  EventsOn: vi.fn(),
  EventsOff: vi.fn(),
}))

const mockCatalog = [
  {
    id: 'lumbras-otb-2026',
    name: "Lumbra's GigaBase — OTB 2026 Release",
    provider: 'LumbrasGigaBase',
    category: 'OTB Master',
    format: '7z-pgn',
    url: 'https://huggingface.co/datasets/Lumbra74/LumbrasGigabaseCCBYNC40/resolve/main/LumbrasGigabase_OTB_2026_partial_release.7z',
    sizeEstimate: '127 MB (.7z)',
    gamesEstimate: '~350,000 OTB games',
    description: 'Latest 2026 Over-The-Board tournament games.',
    supportsLimit: true,
    year: 2026,
    isLatest: true,
  },
  {
    id: 'lichess-open-2026-09',
    name: 'Lichess Open Rated Database Stream — Sep 2026',
    provider: 'Lichess Open',
    category: 'Lichess Monthly Stream',
    format: 'pgn.zst',
    url: 'https://database.lichess.org/standard/lichess_db_standard_rated_2026-09.pgn.zst',
    sizeEstimate: 'Live HTTP Zstd Stream',
    gamesEstimate: '90M+ games',
    description: 'Streams directly from database.lichess.org using on-the-fly zstd decompression.',
    supportsLimit: true,
    year: 2026,
    isLatest: true,
  },
  {
    id: 'lumbras-otb-2025',
    name: "Lumbra's GigaBase — OTB 2025 Complete (Older Archive)",
    provider: 'LumbrasGigaBase',
    category: 'OTB Master',
    format: '7z-pgn',
    url: 'https://huggingface.co/datasets/Lumbra74/LumbrasGigabaseCCBYNC40/resolve/main/LumbrasGigabase_OTB_2025.7z',
    sizeEstimate: '111 MB (.7z)',
    gamesEstimate: '~550,000 OTB games',
    description: 'Older 2025 archive.',
    supportsLimit: true,
    year: 2025,
    isLatest: false,
  },
]

describe('MegaDatabaseModal.vue', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(AppBindings.GetMegaDatabaseCatalog).mockResolvedValue(mockCatalog as any)
    vi.mocked(AppBindings.FetchLichessDatabaseFeed).mockResolvedValue([
      'https://database.lichess.org/broadcast/lichess_db_broadcast_2026-09.pgn.zst',
    ])
    vi.mocked(AppBindings.ImportMegaDatabase).mockResolvedValue({
      sourceId: 'lumbras-otb-2026',
      sourceName: "Lumbra's GigaBase — OTB 2026 Release",
      gamesScanned: 5000,
      gamesImported: 5000,
    } as any)
    vi.mocked(AppBindings.BuildConsolidatedDatabase).mockResolvedValue({
      targetName: 'Consolidated_Reference',
      targetBasePath: '/home/user/.local/share/newscid/databases/Consolidated_Reference',
      sourcesScanned: 3,
      totalGamesScanned: 1200,
      uniqueGamesSaved: 1050,
      duplicatesRemoved: 150,
      duplicatesById: 90,
      duplicatesByContent: 60,
    } as any)
    vi.mocked(AppBindings.CancelActiveOperation).mockResolvedValue(true)
  })

  it('defaults to 2026 / Latest Archives Only and reveals older 2025 archives when unchecked', async () => {
    const wrapper = mount(MegaDatabaseModal, {
      props: { dbOpen: true, dbName: 'MyPrepBase' },
    })
    await flushPromises()

    expect(wrapper.text()).toContain("Lumbra's GigaBase — OTB 2026 Release")
    expect(wrapper.text()).toContain('Lichess Open Rated Database Stream — Sep 2026')
    // Older 2025 archive is hidden by default
    expect(wrapper.find('[data-testid="source-lumbras-otb-2025"]').exists()).toBe(false)

    // Uncheck "2026 / Latest Archives Only" to manually view older 2025 archive
    await wrapper.get('[data-testid="toggle-latest-2026"] input').setValue(false)
    expect(wrapper.find('[data-testid="source-lumbras-otb-2025"]').exists()).toBe(true)

    // Filter to Lichess only
    await wrapper.get('[data-testid="tab-lichess"]').trigger('click')
    expect(wrapper.find('[data-testid="source-lumbras-otb-2026"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="source-lichess-open-2026-09"]').exists()).toBe(true)
  })

  it('triggers ImportMegaDatabase with selected source, maxGames, and minElo', async () => {
    const wrapper = mount(MegaDatabaseModal, {
      props: { dbOpen: true, dbName: 'MyPrepBase' },
    })
    await flushPromises()

    await wrapper.get('[data-testid="source-lichess-open-2026-09"]').trigger('click')
    await wrapper.get('[data-testid="min-elo-select"]').setValue('2400')
    await wrapper.get('[data-testid="start-mega-import-btn"]').trigger('click')
    await flushPromises()

    expect(AppBindings.ImportMegaDatabase).toHaveBeenCalledWith({
      sourceId: 'lichess-open-2026-09',
      customUrl: '',
      maxGames: 25000,
      minElo: 2400,
      createNewDatabase: '',
    })
    expect(wrapper.find('[data-testid="megadb-success-msg"]').exists()).toBe(true)
    expect(wrapper.emitted('imported')).toBeTruthy()
  })

  it('builds a Consolidated Reference Database removing duplicates by ID and move sequence', async () => {
    const wrapper = mount(MegaDatabaseModal, {
      props: { dbOpen: true, dbName: 'MyPrepBase' },
    })
    await flushPromises()

    await wrapper.get('[data-testid="tab-consolidate"]').trigger('click')
    expect(wrapper.text()).toContain('Consolidated Online & Reference Database Builder')

    await wrapper.get('[data-testid="start-mega-import-btn"]').trigger('click')
    await flushPromises()

    expect(AppBindings.BuildConsolidatedDatabase).toHaveBeenCalledWith({
      targetName: 'Consolidated_Reference',
      targetDescription: 'Consolidated Online & Master Reference Database (Deduplicated by ID & Move Sequence)',
      sourceBasePaths: [],
      setAsReference: true,
      setAsDefaultUser: false,
      minElo: 0,
      onlyMostRecentArchives: true,
    })
    expect(wrapper.get('[data-testid="megadb-success-msg"]').text()).toContain('1050 unique games saved')
    expect(wrapper.get('[data-testid="megadb-success-msg"]').text()).toContain('150 duplicates removed')
  })

  it('allows cancelling an active long-running import operation', async () => {
    let resolveImport!: (val: any) => void
    vi.mocked(AppBindings.ImportMegaDatabase).mockReturnValue(
      new Promise(resolve => {
        resolveImport = resolve
      })
    )

    const wrapper = mount(MegaDatabaseModal, {
      props: { dbOpen: true, dbName: 'MyPrepBase' },
    })
    await flushPromises()

    await wrapper.get('[data-testid="start-mega-import-btn"]').trigger('click')
    expect(wrapper.find('[data-testid="cancel-mega-import-btn"]').exists()).toBe(true)

    await wrapper.get('[data-testid="cancel-mega-import-btn"]').trigger('click')
    expect(AppBindings.CancelActiveOperation).toHaveBeenCalled()

    resolveImport({
      sourceId: 'lumbras-otb-2026',
      gamesScanned: 1200,
      gamesImported: 750,
      cancelled: true,
    })
    await flushPromises()
    expect(wrapper.get('[data-testid="megadb-success-msg"]').text()).toContain('Import cancelled cleanly — 750 games imported so far have been saved!')
  })
})


