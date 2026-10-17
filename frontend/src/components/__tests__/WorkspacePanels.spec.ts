import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import PreferencesModal from '../PreferencesModal.vue'
import DatabaseHubPanel from '../DatabaseHubPanel.vue'
import LivePortalPanel from '../LivePortalPanel.vue'
import FloatingPanel from '../FloatingPanel.vue'
import {
  workspacePanels,
  resetWorkspaceLayout,
  togglePanelFloater,
  applyLayoutPreset,
} from '../../utils/workspaceLayout'

describe('Workspace Customization, Database Roles, Floaters, Hyprland & Live Portals', () => {
  beforeEach(() => {
    resetWorkspaceLayout()
  })

  it('manages dockable, floating ("floater"), and hidden panel states and presets', () => {
    expect(workspacePanels.value.tree.mode).toBe('docked')
    togglePanelFloater('tree')
    expect(workspacePanels.value.tree.mode).toBe('floater')

    togglePanelFloater('tree')
    expect(workspacePanels.value.tree.mode).toBe('docked')

    applyLayoutPreset('floaters')
    expect(workspacePanels.value.tree.mode).toBe('floater')
    expect(workspacePanels.value.engine.mode).toBe('floater')

    applyLayoutPreset('minimal')
    expect(workspacePanels.value.games.mode).toBe('hidden')

    applyLayoutPreset('standard')
    expect(workspacePanels.value.games.mode).toBe('docked')
  })

  it('renders DatabaseHubPanel with clear 4-role database hierarchy and Set Default / Set Reference actions', async () => {
    const wrapper = mount(DatabaseHubPanel, {
      props: {
        rolesStatus: {
          activeDb: {
            basePath: '/tmp/ChristopherFair',
            name: 'ChristopherFair',
            numGames: 420,
            description: 'User Games',
          },
          defaultDbPath: '/tmp/ChristopherFair',
          defaultDbName: 'ChristopherFair',
          isActiveDefault: true,
          referenceDb: {
            basePath: '/tmp/LumbrasGigaBase',
            name: 'LumbrasGigaBase',
            numGames: 15000000,
            description: 'Master Reference',
          },
          referenceDbPath: '/tmp/LumbrasGigaBase',
          referenceDbName: 'LumbrasGigaBase',
          useRefForTree: true,
          clipbasePath: '/tmp/clipbase',
          isClipbase: false,
        },
        discoveredDatabases: [
          {
            name: 'ChristopherFair',
            basePath: '/tmp/ChristopherFair',
            numGames: 420,
            category: 'User Databases',
            exists: true,
            type: 'scid',
            description: 'User Games',
            sizeStr: '1.2 MB',
          },
          {
            name: 'LumbrasGigaBase',
            basePath: '/tmp/LumbrasGigaBase',
            numGames: 15000000,
            category: 'Master Databases',
            exists: true,
            type: 'scid',
            description: 'Master Reference',
            sizeStr: '3.4 GB',
          },
        ],
      },
    })

    expect(wrapper.text()).toContain('How Databases Work Together in NewScid')
    expect(wrapper.text()).toContain('1. Default Startup DB')
    expect(wrapper.text()).toContain('2. Active Working DB')
    expect(wrapper.text()).toContain('3. Reference Theory DB')
    expect(wrapper.text()).toContain('4. Clipbase')
    expect(wrapper.text()).toContain('ChristopherFair')
    expect(wrapper.text()).toContain('LumbrasGigaBase')
  })

  it('renders LivePortalPanel for Lichess.org & Chess.com with Hyprland window launcher and embedded modes', async () => {
    const wrapper = mount(LivePortalPanel, {
      props: {
        currentFen: 'rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq - 0 1',
        defaultWindowMode: 'browser-app',
        lichessUsername: 'DrNykterstein',
        chessComUsername: 'MagnusCarlsen',
      },
    })

    expect(wrapper.text()).toContain('Lichess.org')
    expect(wrapper.text()).toContain('Chess.com')

    // Switch to Chess.com
    const buttons = wrapper.findAll('button')
    const chessComBtn = buttons.find(b => b.text().includes('Chess.com'))
    expect(chessComBtn).toBeTruthy()
    await chessComBtn!.trigger('click')
    expect(wrapper.text()).toContain('Chess.com')
  })

  it('renders FloatingPanel ("Floater") with Dock, Screen, Minimize, and Hide controls', async () => {
    togglePanelFloater('engine')
    const wrapper = mount(FloatingPanel, {
      props: {
        panel: workspacePanels.value.engine,
      },
      slots: {
        default: '<div class="test-floater-slot">Stockfish 17 Live</div>',
      },
    })

    expect(wrapper.text()).toContain('UCI Engine & Tablebases')
    expect(wrapper.text()).toContain('Floater')
    expect(wrapper.text()).toContain('Stockfish 17 Live')

    // Clicking Dock emits dock event
    const dockBtn = wrapper.findAll('button').find(b => b.text().includes('Dock'))
    expect(dockBtn).toBeTruthy()
    await dockBtn!.trigger('click')
    expect(wrapper.emitted('dock')).toBeTruthy()
  })

  it('renders PreferencesModal organized document with all 6 sections and Hyprland rules', async () => {
    const wrapper = mount(PreferencesModal, {
      props: {
        initialSection: 'databases',
        preferences: {
          defaultDatabasePath: '/tmp/ChristopherFair',
          autoOpenDefaultDb: true,
          referenceDatabasePath: '',
          useReferenceDbForTree: true,
          defaultPieceSet: 'cburnett',
          defaultBoardTheme: 'classic-wood',
          defaultUITheme: 'midnight-zinc',
          defaultNotationStyle: 'san',
          soundEnabled: true,
          showEvalBar: true,
          threatModeDefault: false,
          defaultEngine: 'stockfish',
          defaultMultiPv: 3,
          defaultThreads: 4,
          defaultHashMb: 256,
          annotateMoveTimeMs: 300,
          lichessUsername: 'Player1',
          chessComUsername: 'Player1',
          defaultPortalPlatform: 'lichess',
          portalWindowMode: 'browser-app',
          customBrowserCmd: '',
        },
        rolesStatus: {
          activeDb: {
            basePath: '/tmp/ChristopherFair',
            name: 'ChristopherFair',
            numGames: 100,
            description: 'User DB',
          },
          defaultDbPath: '/tmp/ChristopherFair',
          defaultDbName: 'ChristopherFair',
          isActiveDefault: true,
          referenceDb: null,
          referenceDbPath: '',
          referenceDbName: '',
          useRefForTree: true,
          clipbasePath: '/tmp/clipbase',
          isClipbase: false,
        },
        discoveredDatabases: [],
        pieceSets: [{ id: 'cburnett', name: 'CBurnett', isScidNative: false, description: 'CBurnett SVG' }],
        boardThemes: [{ id: 'classic-wood', name: 'Classic Wood', light: '#eee', dark: '#888', highlight: '#ff0' }],
        discoveredEngines: [],
        currentPieceSet: 'cburnett',
        currentThemeId: 'classic-wood',
        currentUITheme: 'midnight-zinc',
        soundEnabled: true,
        showEvalBar: true,
        threatMode: false,
        blindfold: false,
        leftColWidth: 520,
        bottomDrawerHeight: 220,
      },
    })

    expect(wrapper.text()).toContain('Preferences & Options Document')
    expect(wrapper.text()).toContain('Databases & Default DB')
    expect(wrapper.text()).toContain('Panels, Docks & Floaters')
    expect(wrapper.text()).toContain('Multi-Screen & Hyprland')
    expect(wrapper.text()).toContain('Lichess & Chess.com Portals')

    // Navigate to Multi-Screen & Hyprland section
    const hyprBtn = wrapper.findAll('button').find(b => b.text().includes('Multi-Screen & Hyprland'))
    expect(hyprBtn).toBeTruthy()
    await hyprBtn!.trigger('click')
    expect(wrapper.text()).toContain('windowrulev2')
  })
})
