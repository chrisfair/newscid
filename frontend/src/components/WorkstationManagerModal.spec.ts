import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import WorkstationManagerModal from './WorkstationManagerModal.vue'
import * as AppBindings from '../../wailsjs/go/main/App'
import * as RuntimeBindings from '../../wailsjs/runtime/runtime'

vi.mock('../../wailsjs/go/main/App', () => ({
  BuildFocusedDatabase: vi.fn(),
  GetWorkstationStorageReport: vi.fn(),
  ImportAssetToWorkstation: vi.fn(),
  SelectAndImportWorkstationAsset: vi.fn(),
  FactoryResetNewScid: vi.fn(),
  GetNotablePlayersDropdown: vi.fn(),
  SetDatabaseWritePolicy: vi.fn(),
}))

vi.mock('../../wailsjs/runtime/runtime', () => ({
  ClipboardSetText: vi.fn(),
}))

const mockDiscoveredDatabases = [
  {
    name: 'LumbrasGigaBase_2026',
    basePath: '/home/user/.local/share/newscid/databases/LumbrasGigaBase_2026',
    type: 'Scid',
    category: 'User Databases',
    numGames: 350000,
    description: 'Master reference archive',
    sizeStr: '125 MB',
    writePolicy: 'sync_only',
    isLocked: false,
  },
  {
    name: 'MyClubGames',
    basePath: '/home/user/.local/share/newscid/databases/MyClubGames',
    type: 'Scid',
    category: 'User Databases',
    numGames: 420,
    description: 'Personal games',
    sizeStr: '180 KB',
    writePolicy: 'editable',
    isLocked: false,
  },
]

const mockNotablePlayers = [
  {
    name: 'Bobby Fischer',
    title: '11th World Champion (1972-1975)',
    era: '1955-1992',
    aliases: ['Fischer, Robert James', 'Fischer, Bobby'],
    gameCount: 820,
  },
  {
    name: 'Magnus Carlsen',
    title: '16th World Champion (2013-2023)',
    era: '2000-present',
    aliases: ['Carlsen, Magnus', 'DrNykterstein'],
    gameCount: 4100,
  },
]

const mockStorageReport = {
  dataRootDir: '/home/user/.local/share/newscid',
  configRootDir: '/home/user/.config/newscid',
  totalBytes: 134217728,
  totalSizeStr: '128.00 MB',
  totalFiles: 14,
  categories: [
    {
      categoryId: 0,
      key: 'databases',
      label: 'Chess Databases',
      path: '/home/user/.local/share/newscid/databases',
      description: 'Managed Scid (.si4/.sg4/.sn4) and PGN (.pgn) chess game databases',
      fileCount: 6,
      totalBytes: 131072000,
      sizeFormatted: '125.00 MB',
      assets: [
        {
          name: 'LumbrasGigaBase_2026.si4',
          path: '/home/user/.local/share/newscid/databases/LumbrasGigaBase_2026.si4',
          category: 'databases',
          sizeBytes: 131072000,
          sizeFormatted: '125.00 MB',
          modifiedAt: '2026-10-19T12:00:00Z',
        },
      ],
    },
    {
      categoryId: 2,
      key: 'engines',
      label: 'UCI Chess Engines',
      path: '/home/user/.local/share/newscid/engines',
      description: 'Local UCI chess engine binaries managed by NewScid',
      fileCount: 1,
      totalBytes: 3145728,
      sizeFormatted: '3.00 MB',
      assets: [],
    },
  ],
}

describe('WorkstationManagerModal.vue', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(AppBindings.GetNotablePlayersDropdown).mockResolvedValue(mockNotablePlayers as any)
    vi.mocked(AppBindings.GetWorkstationStorageReport).mockResolvedValue(mockStorageReport as any)
    vi.mocked(AppBindings.SetDatabaseWritePolicy).mockResolvedValue({} as any)
    vi.mocked(RuntimeBindings.ClipboardSetText).mockResolvedValue(true as any)
  })

  it('renders Multi-Database Sub-Database Builder, selects source DBs, and calls BuildFocusedDatabase', async () => {
    vi.mocked(AppBindings.BuildFocusedDatabase).mockResolvedValue({
      targetBasePath: '/home/user/.local/share/newscid/databases/Fischer_Najdorf_Study',
      targetName: 'Fischer_Najdorf_Study',
      sourcesScanned: 2,
      totalGamesScanned: 350420,
      gamesMatched: 64,
      duplicatesSkipped: 3,
    } as any)

    const wrapper = mount(WorkstationManagerModal, {
      props: {
        activeDbPath: '/home/user/.local/share/newscid/databases/LumbrasGigaBase_2026',
        activeDbName: 'LumbrasGigaBase_2026',
        discoveredDatabases: mockDiscoveredDatabases,
        currentOpeningMoves: ['1.e4', 'c5', '2.Nf3', 'd6'],
      },
    })
    await flushPromises()

    expect(wrapper.find('[data-testid="panel-builder"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('LumbrasGigaBase_2026')
    expect(wrapper.text()).toContain('MyClubGames')

    // Select all user DBs
    await wrapper.get('[data-testid="select-all-user-dbs-btn"]').trigger('click')

    // Add Bobby Fischer via notable player chip
    await wrapper.get('[data-testid="player-chip-Bobby Fischer"]').trigger('click')

    // Apply Sicilian ECO preset and board moves
    await wrapper.get('[data-testid="eco-preset-B20-B99"]').trigger('click')
    await wrapper.get('[data-testid="use-board-moves-btn"]').trigger('click')

    // Set target name and build
    await wrapper.get('[data-testid="target-name-input"]').setValue('Fischer_Najdorf_Study')
    await wrapper.get('[data-testid="build-focused-db-btn"]').trigger('click')
    await flushPromises()

    expect(AppBindings.BuildFocusedDatabase).toHaveBeenCalledWith(
      expect.objectContaining({
        targetName: 'Fischer_Najdorf_Study',
        sourceBasePaths: [
          '/home/user/.local/share/newscid/databases/LumbrasGigaBase_2026',
          '/home/user/.local/share/newscid/databases/MyClubGames',
        ],
        players: ['Bobby Fischer'],
        ecoFilter: 'B20-B99',
        openingMoves: ['1.e4', 'c5', '2.Nf3', 'd6'],
        deduplicate: true,
      })
    )

    expect(wrapper.find('[data-testid="build-result-card"]').exists()).toBe(true)
    expect(wrapper.get('[data-testid="result-games-matched"]').text()).toBe('64')

    // Click Open Focused Database Now
    await wrapper.get('[data-testid="open-built-db-btn"]').trigger('click')
    expect(wrapper.emitted('openDatabase')?.[0]).toEqual([
      '/home/user/.local/share/newscid/databases/Fischer_Najdorf_Study',
    ])
  })

  it('switches to Managed Storage & Assets tab, displays paths, and copies an asset via ImportAssetToWorkstation', async () => {
    vi.mocked(AppBindings.ImportAssetToWorkstation).mockResolvedValue({
      name: 'stockfish-17',
      path: '/home/user/.local/share/newscid/engines/stockfish-17',
      category: 'engines',
      sizeBytes: 41943040,
      sizeFormatted: '40.00 MB',
      modifiedAt: '2026-10-19T13:00:00Z',
    } as any)

    const wrapper = mount(WorkstationManagerModal, {
      props: {
        discoveredDatabases: mockDiscoveredDatabases,
      },
    })
    await flushPromises()

    await wrapper.get('[data-testid="tab-storage"]').trigger('click')
    await flushPromises()

    expect(AppBindings.GetWorkstationStorageReport).toHaveBeenCalled()
    expect(wrapper.get('[data-testid="data-root-dir"]').text()).toBe('/home/user/.local/share/newscid')
    expect(wrapper.get('[data-testid="config-root-dir"]').text()).toBe('/home/user/.config/newscid')
    expect(wrapper.get('[data-testid="category-path-engines"]').text()).toBe(
      '/home/user/.local/share/newscid/engines'
    )

    // Copy custom engine binary path into NewScid
    await wrapper.get('[data-testid="custom-asset-input-engines"]').setValue('/usr/local/bin/stockfish-17')
    await wrapper.get('[data-testid="copy-asset-btn-engines"]').trigger('click')
    await flushPromises()

    expect(AppBindings.ImportAssetToWorkstation).toHaveBeenCalledWith('engines', '/usr/local/bin/stockfish-17')
    expect(wrapper.find('[data-testid="workstation-status-banner"]').text()).toContain('stockfish-17')
  })

  it('switches to Factory Reset tab, defaults opt-in backup checkbox to unchecked, and runs FactoryResetNewScid', async () => {
    vi.mocked(AppBindings.FactoryResetNewScid).mockResolvedValue({
      resetCompletedAt: '2026-10-19T14:00:00Z',
      backupCreated: true,
      backupArchivePath: '/home/user/.local/share/newscid/backups/newscid-factory-backup-20261019.tar.gz',
      wipedPaths: ['/home/user/.local/share/newscid/databases'],
      filesRemoved: 18,
      bytesFreed: 134217728,
      sizeFreedStr: '128.00 MB',
      message: 'Factory reset complete (18 files removed, 128.00 MB freed).',
    } as any)

    const wrapper = mount(WorkstationManagerModal, {
      props: {
        initialTab: 'reset',
      },
    })
    await flushPromises()

    const optInCheckbox = wrapper.get('[data-testid="opt-in-backup-checkbox"]')
    expect((optInCheckbox.element as HTMLInputElement).checked).toBe(false)

    // Opt-in to backup and keep engines
    await optInCheckbox.setValue(true)
    await wrapper.get('[data-testid="keep-engines-checkbox"]').setValue(true)

    await wrapper.get('[data-testid="confirm-factory-reset-btn"]').trigger('click')
    await flushPromises()

    expect(AppBindings.FactoryResetNewScid).toHaveBeenCalledWith({
      createOptInBackup: true,
      keepEngines: true,
      keepBooks: false,
    })

    expect(wrapper.find('[data-testid="factory-reset-result-card"]').exists()).toBe(true)
    expect(wrapper.get('[data-testid="reset-files-removed"]').text()).toBe('18')
    expect(wrapper.get('[data-testid="reset-backup-path"]').text()).toContain('newscid-factory-backup')
    expect(wrapper.emitted('factoryResetCompleted')).toBeTruthy()
  })
})
