<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch, nextTick } from 'vue'
import MenuBar from './components/MenuBar.vue'
import HeaderBar, { PieceSetOption, BoardThemeOption, DiscoveredDatabase } from './components/HeaderBar.vue'
import Chessboard from './components/Chessboard.vue'
import MoveNotation, { FlatMove } from './components/MoveNotation.vue'
import OpeningTree, { TreeData } from './components/OpeningTree.vue'
import OpeningTreeModal from './components/OpeningTreeModal.vue'
import EnginePanel, { EvalInfo, EngineOption } from './components/EnginePanel.vue'
import GameBrowser, { GameSummary } from './components/GameBrowser.vue'
import BoardSetupModal from './components/BoardSetupModal.vue'
import PlayEngineModal from './components/PlayEngineModal.vue'
import TacticsTrainerModal from './components/TacticsTrainerModal.vue'
import GuessTheMoveModal from './components/GuessTheMoveModal.vue'
import EvaluationGraph, { GameAnalysisData } from './components/EvaluationGraph.vue'
import KeyboardShortcutsModal from './components/KeyboardShortcutsModal.vue'
import OpeningTrainerModal from './components/OpeningTrainerModal.vue'
import ProgressiveTacticsModal from './components/ProgressiveTacticsModal.vue'
import PuzzleMinerModal from './components/PuzzleMinerModal.vue'
import HeaderSearchModal from './components/HeaderSearchModal.vue'
import AboutModal from './components/AboutModal.vue'
import HelpModal from './components/HelpModal.vue'
import DatabaseMaintenanceModal from './components/DatabaseMaintenanceModal.vue'
import OnlineSyncModal from './components/OnlineSyncModal.vue'
import PlayerSimulationModal from './components/PlayerSimulationModal.vue'
import PersonaModal from './components/PersonaModal.vue'
import TournamentCrosstableModal from './components/TournamentCrosstableModal.vue'
import ECOBrowserModal from './components/ECOBrowserModal.vue'
import DatabaseStatsModal from './components/DatabaseStatsModal.vue'
import EditTagsModal from './components/EditTagsModal.vue'
import TWICDownloadModal from './components/TWICDownloadModal.vue'
import EngineMatchModal from './components/EngineMatchModal.vue'
import ExportGameModal from './components/ExportGameModal.vue'
import Chess960Modal from './components/Chess960Modal.vue'
import CoordinateTrainerModal from './components/CoordinateTrainerModal.vue'
import CommandPalette, { CommandItem } from './components/CommandPalette.vue'
import ImbalanceBar from './components/ImbalanceBar.vue'
import ExplainMoveModal from './components/ExplainMoveModal.vue'
import BlunderVaultModal from './components/BlunderVaultModal.vue'
import OpponentScoutModal from './components/OpponentScoutModal.vue'
import AntiPersonaModal from './components/AntiPersonaModal.vue'
import PieceSafetyModal from './components/PieceSafetyModal.vue'
import IntuitionStormModal from './components/IntuitionStormModal.vue'
import TimeManagementModal from './components/TimeManagementModal.vue'
import RepertoireShortenerModal from './components/RepertoireShortenerModal.vue'
import ProphylaxisModal from './components/ProphylaxisModal.vue'
import SupportModal from './components/SupportModal.vue'
import MegaDatabaseModal from './components/MegaDatabaseModal.vue'
import FairPlayModal from './components/FairPlayModal.vue'
import FloatingPanel from './components/FloatingPanel.vue'
import LivePortalPanel from './components/LivePortalPanel.vue'
import DatabaseHubPanel, { type DatabaseRolesStatusView } from './components/DatabaseHubPanel.vue'
import PreferencesModal, { type AppPreferencesModel } from './components/PreferencesModal.vue'
import {
  workspacePanels,
  floatingPanels,
  setPanelMode,
  setPanelWorkspace,
  setPanelDockZone,
  togglePanelFloater,
  togglePanelVisibility,
  bringFloaterToFront,
  updateFloaterRect,
  applyLayoutPreset,
  resetWorkspaceLayout,
  serializeWorkspacePanelsForPrefs,
  hydrateWorkspacePanels,
  setOnPanelsChangedHook,
  type PanelId
} from './utils/workspaceLayout'
import { initUITheme, applyUITheme, toggleDarkLight, currentUITheme, UI_THEMES } from './utils/theme'
import { Bot, Sparkles, Brain, Shield, Maximize2, Minimize2, AlertTriangle, Crosshair, ExternalLink, X, PanelRight } from '@lucide/vue'
import { playerbot, persona, scid, tournament, imbalance, explainer, blundervault, scout } from '../wailsjs/go/models'

// Import Wails backend bindings
import {
  OpenDatabase,
  CreateDatabase,
  CloseDatabase,
  GetDatabaseInfo,
  StandardizePlayerNames,
  GetGames,
  GetGame,
  SaveGame,
  DeleteGame,
  OpenFileDialog,
  SaveFileDialog,
  ParseFEN,
  MakeMove,
  GetOpeningTree,
  ClassifyOpening,
  StartEngine,
  StopEngine,
  AnalyzePosition,
  GetAvailableEngines,
  GetDiscoveredEngines,
  GetDiscoveredDatabases,
  GetPieceSets,
  GetBoardThemes,
  GetScidSystemBases,
  ParsePGN,
  ExportPGN,
  GetBestMove,
  AnnotateCurrentGame,
  AnnotatePGN,
  SearchPosition,
  ProbeTablebase,
  LoadTacticsPuzzle,
  SetEngineMultiPV,
  GetEngineMultiPV,
  SetEngineThreads,
  SetEngineHash,
  FormatMove,
  SetGameNAG,
  ClearGameNAGs,
  AddGameComment,
  PromoteVariation,
  DeleteVariation,
  GetSimulatedMove,
  GetPersonaSimulatedMove,
  GetPositionalImbalance,
  ExplainMove,
  GetDueBlunderPuzzles,
  GetAllBlunderPuzzles,
  RecordBlunderReview,
  AddBlunderToVault,
  GenerateOpponentScoutingReport,
  GetPersona,
  GetSupportStatus,
  GetAppPreferences,
  SaveAppPreferences,
  SetDefaultDatabase,
  SetReferenceDatabase,
  OpenDefaultDatabase,
  OpenClipbase,
  GetDatabaseRolesStatus,
  GetWindowStartupConfig,
  LaunchDetachedPanelWindow,
  LaunchPanelOnWorkspace,
  MoveCurrentWindowToWorkspace,
  UpdateDatabaseFromLocal,
  LaunchPortalWindow,
  PublishWorkspaceState,
  GetSharedWorkspaceState,
} from '../wailsjs/go/main/App'
import { EventsOn, EventsOff, ClipboardSetText, ClipboardGetText, BrowserOpenURL } from '../wailsjs/runtime/runtime'

const startingFEN = 'rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1'

const headerBarRef = ref<InstanceType<typeof HeaderBar> | null>(null)

function handleOpenSettings() {
  headerBarRef.value?.openSettings()
}

// App State
const currentFEN = ref(startingFEN)
const legalMoves = ref<string[]>([])
const inCheck = ref(false)
const lastMove = ref<{ from: string; to: string } | null>(null)
const flipped = ref(false)
const activeTab = ref<'notation' | 'tree' | 'games' | 'engine' | 'db-roles' | 'portal' | 'info'>('notation')
const bottomDrawerOpen = ref(true)
const showMoreTabsDropdown = ref(false)

// History & Move Tree
const history = ref<{ fen: string; lastMove: { from: string; to: string } | null }[]>([
  { fen: startingFEN, lastMove: null },
])
const currentPly = ref(0)
const isPlaying = ref(false)
let playInterval: number | null = null

// Game metadata
const gameTags = ref<Record<string, string>>({
  White: 'White',
  Black: 'Black',
  Date: '????.??.??',
  Result: '*',
  Event: 'Casual Game',
})
const flatMoves = ref<FlatMove[]>([])
const activeGameId = ref<number | undefined>(undefined)

// Multi-Game Tabbed Workspace State
export interface GameTab {
  id: string
  title: string
  activeGameId?: number
  gameTags: Record<string, string>
  history: { fen: string; lastMove: { from: string; to: string } | null }[]
  flatMoves: FlatMove[]
  currentPly: number
  gameAnalysis: GameAnalysisData | null
}

const openGameTabs = ref<GameTab[]>([
  {
    id: 'tab-1',
    title: 'Game 1',
    activeGameId: undefined,
    gameTags: {
      White: 'White',
      Black: 'Black',
      Date: '????.??.??',
      Result: '*',
      Event: 'Casual Game',
    },
    history: [{ fen: startingFEN, lastMove: null }],
    flatMoves: [],
    currentPly: 0,
    gameAnalysis: null,
  }
])
const activeGameTabIdx = ref(0)

function saveCurrentTabSnapshot() {
  if (activeGameTabIdx.value < 0 || activeGameTabIdx.value >= openGameTabs.value.length) return
  const cur = openGameTabs.value[activeGameTabIdx.value]
  cur.activeGameId = activeGameId.value
  cur.gameTags = { ...gameTags.value }
  cur.history = [...history.value]
  cur.flatMoves = [...flatMoves.value]
  cur.currentPly = currentPly.value
  cur.gameAnalysis = gameAnalysis.value ? JSON.parse(JSON.stringify(gameAnalysis.value)) : null
  if (gameTags.value.White && gameTags.value.Black && (gameTags.value.White !== 'White' || gameTags.value.Black !== 'Black')) {
    cur.title = `${gameTags.value.White} vs ${gameTags.value.Black}`
  }
}

function restoreTabState(tab: GameTab) {
  activeGameId.value = tab.activeGameId
  gameTags.value = { ...tab.gameTags }
  history.value = [...tab.history]
  flatMoves.value = [...tab.flatMoves]
  currentPly.value = tab.currentPly
  gameAnalysis.value = tab.gameAnalysis ? JSON.parse(JSON.stringify(tab.gameAnalysis)) : null
  jumpToPly(tab.currentPly)
}

function switchGameTab(idx: number) {
  if (idx === activeGameTabIdx.value || idx < 0 || idx >= openGameTabs.value.length) return
  saveCurrentTabSnapshot()
  activeGameTabIdx.value = idx
  restoreTabState(openGameTabs.value[idx])
}

function openNewGameTab() {
  saveCurrentTabSnapshot()
  const newNum = openGameTabs.value.length + 1
  const newTab: GameTab = {
    id: `tab-${Date.now()}-${newNum}`,
    title: `Game ${newNum}`,
    activeGameId: undefined,
    gameTags: {
      White: 'White',
      Black: 'Black',
      Date: new Date().toISOString().split('T')[0].replace(/-/g, '.'),
      Result: '*',
      Event: 'Casual Game',
    },
    history: [{ fen: startingFEN, lastMove: null }],
    flatMoves: [],
    currentPly: 0,
    gameAnalysis: null,
  }
  openGameTabs.value.push(newTab)
  activeGameTabIdx.value = openGameTabs.value.length - 1
  restoreTabState(newTab)
}

function closeGameTab(idx: number) {
  if (openGameTabs.value.length <= 1) {
    handleNewGame()
    openGameTabs.value[0].title = 'Game 1'
    return
  }
  if (activeGameTabIdx.value === idx) {
    const nextIdx = idx === 0 ? 1 : idx - 1
    restoreTabState(openGameTabs.value[nextIdx])
    openGameTabs.value.splice(idx, 1)
    activeGameTabIdx.value = idx === 0 ? 0 : idx - 1
  } else {
    openGameTabs.value.splice(idx, 1)
    if (activeGameTabIdx.value > idx) {
      activeGameTabIdx.value--
    }
  }
}

// Database State
const dbInfo = ref<{ basePath: string; name: string; numGames: number; description: string } | null>(null)
const systemBases = ref<string[]>([])
const discoveredDatabases = ref<DiscoveredDatabase[]>([])
const gamesList = ref<GameSummary[]>([])
const totalGames = ref(0)
const currentPage = ref(1)
const pageSize = ref(20)
const searchFilter = ref<{ player?: string; result?: string; eco?: string }>({})

// Engine State
const isAnalyzing = ref(false)
const availableEngines = ref<string[]>([])
const discoveredEngines = ref<EngineOption[]>([])
const selectedEngine = ref('')
const evalInfo = ref<EvalInfo | null>(null)
const engineMultiPv = ref(1)

// Game Analysis & Blunder Detection
const gameAnalysis = ref<GameAnalysisData | null>(null)
const isAnalyzingGame = ref(false)
const annotationProgress = ref<{ currentPly: number; totalPlies: number; percent: number } | null>(null)

// Trial Mode
const trialMode = ref(false)
const trialSavedState = ref<{
  fen: string
  history: { fen: string; lastMove: { from: string; to: string } | null }[]
  flatMoves: FlatMove[]
  currentPly: number
  tags: Record<string, string>
} | null>(null)

// Modals State
const showSetupBoardModal = ref(false)
const showPlayEngineModal = ref(false)
const showTacticsModal = ref(false)
const showOpeningTrainerModal = ref(false)
const showProgressiveTacticsModal = ref(false)
const showPuzzleMinerModal = ref(false)
const showShortcutsModal = ref(false)
const showAboutModal = ref(false)
const showHelpModal = ref(false)
const showSearchModal = ref(false)
const showMaintenanceModal = ref(false)
const showOnlineSyncModal = ref(false)
const showPlayerSimulationModal = ref(false)
const showPersonaModal = ref(false)
const showTournamentCrosstableModal = ref(false)
const showEcoBrowserModal = ref(false)
const showGuessTheMoveModal = ref(false)
const showDatabaseStatsModal = ref(false)
const showEditTagsModal = ref(false)
const showTwicModal = ref(false)
const showEngineMatchModal = ref(false)
const showExportGameModal = ref(false)
const showChess960Modal = ref(false)
const showCoordinateTrainerModal = ref(false)
const showCommandPalette = ref(false)
const showExplainMoveModal = ref(false)
const showBlunderVaultModal = ref(false)
const showOpponentScoutModal = ref(false)
const showAntiPersonaModal = ref(false)
const showPieceSafetyModal = ref(false)
const showIntuitionStormModal = ref(false)
const showTimeManagementModal = ref(false)
const showRepertoireShortenerModal = ref(false)
const showProphylaxisModal = ref(false)
const showSupportModal = ref(false)
const showMegaDatabaseModal = ref(false)
const showFairPlayModal = ref(false)
const showPreferencesModal = ref(false)
const preferencesSection = ref('databases')
const antiPersonaTargetId = ref<string>('')
const antiPersonaTargetName = ref<string>('')
const showOpeningTreeModal = ref(false)

const appPreferences = ref<AppPreferencesModel>({
  defaultDatabasePath: '',
  referenceDatabasePath: '',
  autoOpenDefaultDb: true,
  useReferenceDbForTree: true,
  defaultPieceSet: 'cburnett',
  defaultBoardTheme: 'classic-wood',
  defaultUITheme: 'dark',
  defaultNotationStyle: 'san',
  soundEnabled: true,
  showEvalBar: true,
  threatModeDefault: false,
  blindfoldDefault: false,
  boardFlippedDefault: false,
  compactLayoutMode: 'auto',
  leftColWidth: 520,
  bottomDrawerHeight: 220,
  bottomDrawerOpen: true,
  mainWindowWorkspace: '',
  autoRestoreWorkspaceWindows: false,
  panelConfigs: {},
  defaultEngine: '',
  defaultMultiPv: 1,
  defaultThreads: 2,
  defaultHashMb: 64,
  annotateMoveTimeMs: 150,
  lichessUsername: '',
  chessComUsername: '',
  defaultPortalPlatform: 'lichess',
  portalWindowMode: 'browser-app',
  customBrowserCmd: '',
})

const prefsInitialized = ref(false)
let persistConfigTimer: number | null = null
const isUpdatingDbFromLocal = ref(false)
const localMergeStatusMsg = ref<string | null>(null)

function schedulePersistUserConfig() {
  if (!prefsInitialized.value || windowStartupMode.value.mode !== 'main') return
  if (persistConfigTimer) window.clearTimeout(persistConfigTimer)
  persistConfigTimer = window.setTimeout(async () => {
    try {
      const payload: AppPreferencesModel = {
        ...appPreferences.value,
        defaultPieceSet: currentPieceSet.value,
        defaultBoardTheme: currentThemeId.value,
        defaultUITheme: currentUITheme.value,
        soundEnabled: soundEnabled.value,
        showEvalBar: showEvalBar.value,
        threatModeDefault: threatMode.value,
        blindfoldDefault: blindfoldMode.value,
        boardFlippedDefault: flipped.value,
        compactLayoutMode: compactModeOverride.value,
        leftColWidth: leftColWidth.value,
        bottomDrawerHeight: bottomDrawerHeight.value,
        bottomDrawerOpen: bottomDrawerOpen.value,
        defaultEngine: selectedEngine.value || appPreferences.value.defaultEngine,
        defaultMultiPv: engineMultiPv.value || appPreferences.value.defaultMultiPv || 1,
        panelConfigs: serializeWorkspacePanelsForPrefs(),
      }
      const saved = await SaveAppPreferences(payload as any)
      if (saved) {
        appPreferences.value = saved as AppPreferencesModel
      }
    } catch {
      // Ignore in test environment
    }
  }, 180)
}

setOnPanelsChangedHook(() => {
  schedulePersistUserConfig()
})

const dbRolesStatus = ref<DatabaseRolesStatusView | null>(null)
const windowStartupMode = ref<{
  mode: string
  panelId: string
  workspace?: string
  portalUrl: string
  portalTitle: string
  initialFen: string
  initialDb: string
}>({
  mode: 'main',
  panelId: '',
  workspace: '',
  portalUrl: '',
  portalTitle: '',
  initialFen: '',
  initialDb: '',
})
let multiScreenSyncTimer: number | null = null
let lastSyncedTimestamp = 0
const windowInstanceId = `win-${Date.now()}-${Math.random().toString(36).slice(2, 6)}`

async function refreshDatabaseRolesStatus() {
  try {
    const status = await GetDatabaseRolesStatus()
    if (status) {
      dbRolesStatus.value = status as DatabaseRolesStatusView
    }
  } catch {
    // Ignore in test/mock environments
  }
}

async function publishCurrentWorkspaceState() {
  try {
    const now = Date.now()
    lastSyncedTimestamp = now
    await PublishWorkspaceState({
      updatedAtMs: now,
      sourceId: windowInstanceId,
      fen: currentFEN.value,
      lastMoveFrom: lastMove.value?.from || '',
      lastMoveTo: lastMove.value?.to || '',
      currentPly: currentPly.value,
      dbPath: dbInfo.value?.basePath || '',
      dbName: dbInfo.value?.name || '',
      activeGameId: activeGameId.value || 0,
      gameTags: { ...gameTags.value },
    } as any)
  } catch {
    // Ignore sync write errors
  }
}

function openPreferencesModal(section = 'databases') {
  preferencesSection.value = section || 'databases'
  showPreferencesModal.value = true
}

function openDatabaseHubView() {
  setPanelMode('db-roles', 'docked')
  setPanelDockZone('db-roles', 'right')
  activeTab.value = 'db-roles'
  refreshDatabaseRolesStatus()
}

function openPortalView() {
  if (workspacePanels.value.portal.mode === 'hidden') {
    setPanelMode('portal', 'docked')
  }
  if (workspacePanels.value.portal.mode === 'docked') {
    setPanelDockZone('portal', 'right')
    activeTab.value = 'portal'
  } else {
    bringFloaterToFront('portal')
  }
}

async function handleSetDefaultDatabase(path: string) {
  try {
    const status = await SetDefaultDatabase(path)
    if (status) {
      dbRolesStatus.value = status as DatabaseRolesStatusView
    }
    const prefs = await GetAppPreferences()
    if (prefs) {
      appPreferences.value = prefs as AppPreferencesModel
    }
  } catch (err) {
    console.error('Failed to set default database:', err)
  }
}

async function handleSetCurrentAsDefaultDB() {
  if (!dbInfo.value?.basePath) return
  await handleSetDefaultDatabase(dbInfo.value.basePath)
}

async function handleOpenDefaultDatabase() {
  try {
    const info = await OpenDefaultDatabase()
    if (info) {
      dbInfo.value = info
      currentPage.value = 1
      await refreshGames()
      if (gamesList.value.length > 0) {
        await handleLoadGame(1)
      }
      await refreshDatabaseRolesStatus()
    }
  } catch (err: any) {
    alert(err?.message || err)
  }
}

async function handleSetReferenceDatabase(path: string, useForTree: boolean) {
  try {
    const status = await SetReferenceDatabase(path, useForTree)
    if (status) {
      dbRolesStatus.value = status as DatabaseRolesStatusView
    }
    const prefs = await GetAppPreferences()
    if (prefs) {
      appPreferences.value = prefs as AppPreferencesModel
    }
    fetchOpeningTree(currentFEN.value)
  } catch (err) {
    console.error('Failed to set reference database:', err)
  }
}

async function handleOpenClipbase() {
  try {
    const info = await OpenClipbase()
    if (info) {
      dbInfo.value = info
      currentPage.value = 1
      await refreshGames()
      await refreshDatabaseRolesStatus()
    }
  } catch (err: any) {
    alert(`Failed to open Clipbase: ${err?.message || err}`)
  }
}

async function handleSaveAppPreferences(prefs: AppPreferencesModel) {
  try {
    const merged = {
      ...prefs,
      panelConfigs: serializeWorkspacePanelsForPrefs(),
    }
    const saved = await SaveAppPreferences(merged as any)
    if (saved) {
      appPreferences.value = saved as AppPreferencesModel
    }
    await refreshDatabaseRolesStatus()
    fetchOpeningTree(currentFEN.value)
  } catch (err) {
    console.error('Failed to save preferences:', err)
  }
}

async function handleDetachPanelScreen(panelId: PanelId, workspace?: string) {
  const panel = workspacePanels.value[panelId]
  const ws = workspace !== undefined ? workspace : (panel?.targetWorkspace || '')
  if (panel) {
    if (workspace !== undefined) {
      setPanelWorkspace(panelId, ws)
    }
    if (panelId === 'board') {
      setPanelMode('board', 'detached')
    }
  }
  const title = panel
    ? `NewScid — ${panel.title}${ws ? ` [WS ${ws}]` : ' [Screen Panel]'}`
    : `NewScid — ${panelId}${ws ? ` [WS ${ws}]` : ' [Screen Panel]'}`
  await publishCurrentWorkspaceState()
  try {
    if (typeof LaunchPanelOnWorkspace === 'function') {
      await LaunchPanelOnWorkspace(panelId, ws, title, currentFEN.value, dbInfo.value?.basePath || '')
    } else {
      await LaunchDetachedPanelWindow(panelId, title, currentFEN.value, dbInfo.value?.basePath || '')
    }
  } catch (err: any) {
    alert(`Failed to launch screen window: ${err?.message || err}`)
  }
}

async function handleMoveCurrentWindowToWorkspace(workspace: string) {
  if (!workspace) return
  try {
    windowStartupMode.value.workspace = workspace
    appPreferences.value.mainWindowWorkspace = workspace
    await MoveCurrentWindowToWorkspace(workspace)
  } catch (err) {
    console.error('Failed to move window to workspace:', err)
  }
}

async function handleUpdateDatabaseFromLocal(targetPath = '') {
  if (isUpdatingDbFromLocal.value) return
  isUpdatingDbFromLocal.value = true
  localMergeStatusMsg.value = 'Scanning local databases and merging new games...'
  try {
    const stats = await UpdateDatabaseFromLocal(targetPath, [])
    discoveredDatabases.value = (await GetDiscoveredDatabases()) as DiscoveredDatabase[]
    await refreshGames()
    await refreshDatabaseRolesStatus()
    fetchOpeningTree(currentFEN.value)
    if (stats) {
      localMergeStatusMsg.value = `Updated! +${stats.imported || 0} new games added (${stats.duplicatesSkipped || 0} existing skipped across ${stats.sourcesProcessed || 0} local DBs; total: ${stats.targetTotalGames || totalGames.value} games).`
    } else {
      localMergeStatusMsg.value = 'Database update completed.'
    }
  } catch (err: any) {
    localMergeStatusMsg.value = `Update error: ${err?.message || err}`
  } finally {
    isUpdatingDbFromLocal.value = false
  }
}

async function handleLaunchPortalWindow(platform: 'lichess' | 'chesscom', customUrl?: string) {
  const url = customUrl || (platform === 'chesscom' ? 'https://www.chess.com/play/online' : 'https://lichess.org')
  try {
    await LaunchPortalWindow(platform, url, appPreferences.value.portalWindowMode || 'browser-app')
  } catch {
    try {
      BrowserOpenURL(url)
    } catch {
      window.open(url, '_blank')
    }
  }
}

function openAntiPersonaModal(target?: persona.Persona | { id: string; displayName?: string } | string) {
  if (typeof target === 'string') {
    antiPersonaTargetId.value = ''
    antiPersonaTargetName.value = target
  } else if (target) {
    antiPersonaTargetId.value = target.id || ''
    antiPersonaTargetName.value = target.displayName || ''
  } else {
    antiPersonaTargetId.value = ''
    antiPersonaTargetName.value = ''
  }
  showAntiPersonaModal.value = true
}

function handleOpenTreeWindow() {
  showOpeningTreeModal.value = true
  activeTab.value = 'tree'
  fetchOpeningTree(currentFEN.value)
}

const zenMode = ref(false)
function toggleZenMode() {
  zenMode.value = !zenMode.value
}

// Positional Imbalance State
const imbalanceReport = ref<imbalance.Report | null>(null)
async function updateImbalance(fen: string) {
  try {
    const report = await GetPositionalImbalance(fen)
    imbalanceReport.value = report
  } catch (e) {
    console.warn('Failed to get imbalance:', e)
  }
}

// Move Explainer Target State
const explainMoveTarget = ref<{ beforeFen: string; moveUci: string; scoreCp?: number; bestMove?: string } | null>(null)
function handleExplainMove(ply: number) {
  const m = flatMoves.value.find(item => item.ply === ply)
  if (!m) {
    if (flatMoves.value.length > 0) {
      const lastM = flatMoves.value[flatMoves.value.length - 1]
      const prevFen = history.value[lastM.ply - 1]?.fen || startingFEN
      explainMoveTarget.value = {
        beforeFen: prevFen,
        moveUci: lastM.uci || lastM.san,
        scoreCp: evalInfo.value?.scoreCp,
        bestMove: evalInfo.value?.bestMove
      }
      showExplainMoveModal.value = true
    }
    return
  }
  const prevFen = history.value[ply - 1]?.fen || startingFEN
  explainMoveTarget.value = {
    beforeFen: prevFen,
    moveUci: m.uci || m.san,
    scoreCp: evalInfo.value?.scoreCp,
    bestMove: evalInfo.value?.bestMove
  }
  showExplainMoveModal.value = true
}

// Blunder Radar Guardrail State
const blunderRadarAlert = ref<string | null>(null)
let radarTimer: any = null
function triggerBlunderRadar(warning: string) {
  blunderRadarAlert.value = warning
  if (radarTimer) clearTimeout(radarTimer)
  radarTimer = setTimeout(() => {
    blunderRadarAlert.value = null
  }, 4500)
}

function handleLoadVaultPuzzle(fen: string) {
  showBlunderVaultModal.value = false
  handleSetupBoardApply(fen)
}

// Theory Boundary & Novelty State
const isTheoreticalNovelty = computed(() => {
  if (currentPly.value <= 1) return false
  if (!treeData.value || treeData.value.moves.length === 0) {
    return currentPly.value >= 4 && currentPly.value <= 40
  }
  return false
})

// PyChess & Modern Analysis Features
const soundEnabled = ref(localStorage.getItem('newscid:soundEnabled') !== 'false')
function toggleSound() {
  soundEnabled.value = !soundEnabled.value
  localStorage.setItem('newscid:soundEnabled', String(soundEnabled.value))
}

const blindfoldMode = ref(false)
function toggleBlindfold() {
  blindfoldMode.value = !blindfoldMode.value
}

const threatMode = ref(false)
function toggleThreatMode() {
  threatMode.value = !threatMode.value
}

const showEvalBar = ref(localStorage.getItem('newscid:showEvalBar') !== 'false')
function toggleEvalBar() {
  showEvalBar.value = !showEvalBar.value
  localStorage.setItem('newscid:showEvalBar', String(showEvalBar.value))
}

const threatSquares = computed(() => {
  if (!threatMode.value || !legalMoves.value) return []
  const set = new Set<string>()
  for (const m of legalMoves.value) {
    if (m.length >= 4) {
      set.add(m.slice(2, 4))
    }
  }
  return Array.from(set)
})

const evalScore = computed(() => {
  if (evalInfo.value) return evalInfo.value.scoreCp
  if (imbalanceReport.value) {
    return (imbalanceReport.value as any).estimatedEvalCp ?? (imbalanceReport.value.materialDelta * 100)
  }
  return 0
})
const evalMate = computed(() => (evalInfo.value?.mateIn ? evalInfo.value.mateIn : null))

const engineArrows = computed(() => {
  if (!isAnalyzing.value || !evalInfo.value?.bestMove || evalInfo.value.bestMove.length < 4) {
    return []
  }
  const from = evalInfo.value.bestMove.slice(0, 2)
  const to = evalInfo.value.bestMove.slice(2, 4)
  return [{ from, to, color: '#38bdf8' }]
})

function handleLoadChess960(pos: { id: number; fen: string; pieces: string; isClassic: boolean }) {
  activeGameId.value = undefined
  gameAnalysis.value = null
  gameTags.value = {
    White: 'Player 1',
    Black: 'Player 2',
    Date: new Date().toISOString().split('T')[0].replace(/-/g, '.'),
    Result: '*',
    Event: `Chess960 #${pos.id} (${pos.pieces})`,
    SetUp: '1',
    FEN: pos.fen,
  }
  history.value = [{ fen: pos.fen, lastMove: null }]
  flatMoves.value = []
  jumpToPly(0)
  updateBoardState(pos.fen)
  showChess960Modal.value = false
}

const tablebaseResult = ref<any>(null)
const selectedSimulationPersona = ref<persona.Persona | undefined>(undefined)
const showImportModal = ref(false)
const importPGNText = ref('')

function countPiecesInFEN(fen: string): number {
  if (!fen) return 32
  const placement = fen.split(' ')[0] || ''
  let count = 0
  for (const ch of placement) {
    if ((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')) {
      count++
    }
  }
  return count
}

async function updateTablebase(fen: string) {
  if (countPiecesInFEN(fen) <= 7) {
    try {
      const res = await ProbeTablebase(fen)
      tablebaseResult.value = res
    } catch {
      tablebaseResult.value = null
    }
  } else {
    tablebaseResult.value = null
  }
}

// Player Simulation Sparring State
const activeSimulation = ref<{
  playerName: string
  personaId?: string
  userColor: 'white' | 'black'
  personality: playerbot.PlayerPersonality
  timeControlMinutes: number
} | null>(null)
const isBotThinking = ref(false)
const botMoveStatus = ref<string | null>(null)



// Resizable Panes & Adaptive Sizing State
const windowWidth = ref(typeof window !== 'undefined' ? window.innerWidth : 1280)
const windowHeight = ref(typeof window !== 'undefined' ? window.innerHeight : 800)
const compactModeOverride = ref<'auto' | 'compact' | 'full'>('auto')
const userForcedBottomDrawer = ref(false)
const hasUserManuallyResizedLeft = ref(false)

// Automatically activate compact layout on screens/tiles smaller than 1180px wide or 860px tall
const isSmallScreen = computed(() => windowHeight.value < 860 || windowWidth.value < 1180)
const effectiveCompact = computed(() =>
  compactModeOverride.value === 'compact' ||
  (compactModeOverride.value === 'auto' && isSmallScreen.value)
)

const showBottomGamesDrawer = computed(() =>
  !zenMode.value &&
  workspacePanels.value.games.mode === 'docked' &&
  workspacePanels.value.games.dockZone === 'bottom' &&
  (effectiveCompact.value ? userForcedBottomDrawer.value : bottomDrawerOpen.value)
)

const showLeftImbalancePanel = computed(() =>
  !effectiveCompact.value &&
  workspacePanels.value.imbalance.mode === 'docked' &&
  workspacePanels.value.imbalance.dockZone === 'left'
)

const showLeftEnginePanel = computed(() =>
  !effectiveCompact.value &&
  workspacePanels.value.engine.mode === 'docked' &&
  workspacePanels.value.engine.dockZone === 'left'
)

function toggleCompactLayout() {
  if (effectiveCompact.value) {
    compactModeOverride.value = 'full'
  } else {
    compactModeOverride.value = 'compact'
  }
  hasUserManuallyResizedLeft.value = false
  nextTick(() => updateResponsiveBoardSize())
}

function toggleBottomGamesDrawer() {
  if (effectiveCompact.value) {
    userForcedBottomDrawer.value = !userForcedBottomDrawer.value
  } else {
    bottomDrawerOpen.value = !bottomDrawerOpen.value
  }
  nextTick(() => updateResponsiveBoardSize())
}

function getInitialLeftColWidth(): number {
  const saved = Number(localStorage.getItem('newscid:leftColWidth'))
  const screenW = typeof window !== 'undefined' ? window.innerWidth : 1280
  const maxW = Math.max(280, screenW - 320)
  if (saved && saved >= 260 && saved <= maxW && screenW >= 1200) {
    return saved
  }
  if (screenW < 1000) return Math.max(280, Math.floor(screenW * 0.52))
  if (screenW < 1440) return Math.max(380, Math.floor(screenW * 0.48))
  return 560
}

function getInitialBottomDrawerHeight(): number {
  const saved = Number(localStorage.getItem('newscid:bottomDrawerHeight'))
  const screenH = typeof window !== 'undefined' ? window.innerHeight : 800
  const maxH = Math.max(120, Math.floor(screenH * 0.4))
  if (saved && saved >= 100 && saved <= maxH) {
    return saved
  }
  if (screenH < 800) return 155
  if (screenH < 960) return 195
  return 235
}

const leftColWidth = ref<number>(getInitialLeftColWidth())
const bottomDrawerHeight = ref<number>(getInitialBottomDrawerHeight())
const isResizingH = ref(false)
const isResizingV = ref(false)
const isMdScreen = ref(typeof window !== 'undefined' ? window.innerWidth >= 768 : true)

// Responsive Board Sizing State
const computedBoardSize = ref<number>(460)
const leftColRef = ref<HTMLElement | null>(null)
const workspaceRef = ref<HTMLElement | null>(null)
let resizeObserver: ResizeObserver | null = null

function updateResponsiveBoardSize() {
  if (typeof window === 'undefined') return

  const screenW = window.innerWidth
  const screenH = window.innerHeight
  windowWidth.value = screenW
  windowHeight.value = screenH

  const maxDrawerH = Math.max(120, Math.floor(screenH * 0.4))
  if (bottomDrawerHeight.value > maxDrawerH) {
    bottomDrawerHeight.value = maxDrawerH
  }

  // 1. Measure available workspace & left-column dimensions
  const topBarsH = zenMode.value
    ? 0
    : (34 + (effectiveCompact.value ? 0 : 48) + ((effectiveCompact.value && openGameTabs.value.length <= 1) ? 0 : 34))
  const bottomDrawerH = showBottomGamesDrawer.value ? (bottomDrawerHeight.value + 8) : 0

  const wsClientH = workspaceRef.value?.clientHeight || 0
  const wsClientW = workspaceRef.value?.clientWidth || 0
  const leftClientH = leftColRef.value?.clientHeight || 0

  const wsH = (screenW >= 768 && leftClientH > 120)
    ? leftClientH
    : wsClientH > 120
      ? (wsClientH - 16)
      : Math.max(220, screenH - topBarsH - bottomDrawerH - 16)
  const wsW = wsClientW > 200
    ? (wsClientW - 16)
    : Math.max(280, screenW - 16)

  // 2. Measure actual DOM height + fallback estimates for non-board items inside left-col
  let measuredNonBoardH = 0
  let measuredCount = 0
  if (leftColRef.value) {
    for (const child of Array.from(leftColRef.value.children)) {
      const el = child as HTMLElement
      if (!el.classList.contains('chessboard-wrapper-host')) {
        measuredNonBoardH += el.offsetHeight
        measuredCount++
      }
    }
  }

  let estimatedNonBoardH = 0
  let estCount = 0
  if (activeSimulation.value) { estimatedNonBoardH += 52; estCount++ }
  if (isTheoreticalNovelty.value) { estimatedNonBoardH += 34; estCount++ }
  if (blunderRadarAlert.value) { estimatedNonBoardH += 34; estCount++ }
  if (showLeftImbalancePanel.value && imbalanceReport.value) {
    estimatedNonBoardH += 44
    estCount++
  }
  if (showLeftEnginePanel.value) {
    estimatedNonBoardH += (isAnalyzing.value ? 168 : 136)
    estCount++
  }

  const itemCount = Math.max(measuredCount, estCount)
  const gapsH = itemCount * 6 // gap-1.5 = 6px per flex gap
  const nonBoardH = Math.max(measuredNonBoardH, estimatedNonBoardH) + gapsH + 12

  const availH = Math.max(180, wsH - nonBoardH)
  const evalBarExtra = showEvalBar.value ? 26 : 4

  // If this OS window is a dedicated detached Chessboard window (--window-panel board)
  if (windowStartupMode.value.mode === 'panel' && windowStartupMode.value.panelId === 'board') {
    const detH = Math.max(200, screenH - 115)
    const detW = Math.max(200, screenW - evalBarExtra - 24)
    computedBoardSize.value = Math.max(180, Math.min(820, Math.floor(Math.min(detH, detW))))
    return
  }

  // If the Chessboard is floating or detached to another workspace in the main window
  if (workspacePanels.value.board && workspacePanels.value.board.mode !== 'docked') {
    if (!hasUserManuallyResizedLeft.value) {
      leftColWidth.value = (showLeftEnginePanel.value || showLeftImbalancePanel.value) ? 340 : 235
    }
    return
  }

  if (screenW >= 768) {
    const minRightW = screenW < 920 ? 250 : screenW < 1150 ? 300 : 350
    const maxBoardW = Math.max(180, wsW - minRightW - evalBarExtra - 12)

    if (hasUserManuallyResizedLeft.value) {
      const maxLeftW = Math.max(240, wsW - minRightW)
      if (leftColWidth.value > maxLeftW) {
        leftColWidth.value = maxLeftW
      }
      const availW = Math.max(180, leftColWidth.value - evalBarExtra - 4)
      computedBoardSize.value = Math.max(180, Math.min(800, Math.floor(Math.min(availH, availW))))
    } else {
      const idealBoard = Math.max(180, Math.min(800, Math.floor(Math.min(availH, maxBoardW))))
      computedBoardSize.value = idealBoard
      leftColWidth.value = idealBoard + evalBarExtra + 4
    }
  } else {
    // Stacked vertical layout for narrow windows (< 768px)
    const idealBoard = Math.max(180, Math.floor(Math.min(wsW - evalBarExtra - 4, wsH * 0.54)))
    computedBoardSize.value = idealBoard
  }
}

function handleWindowResize() {
  windowWidth.value = window.innerWidth
  windowHeight.value = window.innerHeight
  isMdScreen.value = window.innerWidth >= 768
  nextTick(() => updateResponsiveBoardSize())
}

function startHorizontalResize(e: MouseEvent) {
  e.preventDefault()
  isResizingH.value = true
  hasUserManuallyResizedLeft.value = true
  const startX = e.clientX
  const startWidth = leftColWidth.value
  document.body.style.cursor = 'col-resize'
  document.body.style.userSelect = 'none'

  function onMouseMove(moveEvent: MouseEvent) {
    const delta = moveEvent.clientX - startX
    const minW = 240
    const maxW = Math.max(minW, window.innerWidth - 260)
    leftColWidth.value = Math.max(minW, Math.min(maxW, startWidth + delta))
    updateResponsiveBoardSize()
  }

  function onMouseUp() {
    isResizingH.value = false
    document.body.style.cursor = ''
    document.body.style.userSelect = ''
    window.removeEventListener('mousemove', onMouseMove)
    window.removeEventListener('mouseup', onMouseUp)
    localStorage.setItem('newscid:leftColWidth', String(leftColWidth.value))
    updateResponsiveBoardSize()
  }

  window.addEventListener('mousemove', onMouseMove)
  window.addEventListener('mouseup', onMouseUp)
}

function resetHorizontalResize() {
  hasUserManuallyResizedLeft.value = false
  localStorage.removeItem('newscid:leftColWidth')
  nextTick(() => updateResponsiveBoardSize())
}

function startVerticalResize(e: MouseEvent) {
  e.preventDefault()
  isResizingV.value = true
  const startY = e.clientY
  const startHeight = bottomDrawerHeight.value
  document.body.style.cursor = 'row-resize'
  document.body.style.userSelect = 'none'

  function onMouseMove(moveEvent: MouseEvent) {
    const delta = startY - moveEvent.clientY
    const minH = 120
    const maxH = Math.max(minH, window.innerHeight - 250)
    bottomDrawerHeight.value = Math.max(minH, Math.min(maxH, startHeight + delta))
    updateResponsiveBoardSize()
  }

  function onMouseUp() {
    isResizingV.value = false
    document.body.style.cursor = ''
    document.body.style.userSelect = ''
    window.removeEventListener('mousemove', onMouseMove)
    window.removeEventListener('mouseup', onMouseUp)
    localStorage.setItem('newscid:bottomDrawerHeight', String(bottomDrawerHeight.value))
    updateResponsiveBoardSize()
  }

  window.addEventListener('mousemove', onMouseMove)
  window.addEventListener('mouseup', onMouseUp)
}

function resetVerticalResize() {
  bottomDrawerHeight.value = getInitialBottomDrawerHeight()
  localStorage.setItem('newscid:bottomDrawerHeight', String(bottomDrawerHeight.value))
  nextTick(() => updateResponsiveBoardSize())
}

// Customization
const pieceSets = ref<PieceSetOption[]>([])
const boardThemes = ref<BoardThemeOption[]>([])
const currentPieceSet = ref('cburnett')
const currentThemeId = ref('classic-wood')

const currentTheme = computed(() => {
  return boardThemes.value.find(t => t.id === currentThemeId.value) || {
    id: 'classic-wood',
    name: 'Classic Wood',
    light: '#e3c896',
    dark: '#ab7a4e',
    highlight: '#f7e597',
  }
})

// Opening Tree Data
const treeData = ref<TreeData | null>(null)
const treeLoading = ref(false)

// Initialize
onMounted(async () => {
  try {
    initUITheme()
    window.addEventListener('resize', handleWindowResize)

    // 0. Check window startup config (main workspace vs detached Hyprland screen panel vs live portal)
    try {
      const winCfg = await GetWindowStartupConfig()
      if (winCfg && winCfg.mode) {
        windowStartupMode.value = winCfg as any
      }
    } catch {
      // Default to main mode
    }

    // Load saved user preferences & panel workspace layout from ~/.config/newscid/preferences.json
    try {
      const prefs = await GetAppPreferences()
      if (prefs) {
        appPreferences.value = prefs as AppPreferencesModel
        if (prefs.defaultPieceSet) currentPieceSet.value = prefs.defaultPieceSet
        if (prefs.defaultBoardTheme) currentThemeId.value = prefs.defaultBoardTheme
        if (prefs.defaultUITheme) applyUITheme(prefs.defaultUITheme as any)
        if (typeof prefs.soundEnabled === 'boolean') soundEnabled.value = prefs.soundEnabled
        if (typeof prefs.showEvalBar === 'boolean') showEvalBar.value = prefs.showEvalBar
        if (typeof prefs.threatModeDefault === 'boolean') threatMode.value = prefs.threatModeDefault
        if (typeof (prefs as any).blindfoldDefault === 'boolean') blindfoldMode.value = (prefs as any).blindfoldDefault
        if (typeof (prefs as any).boardFlippedDefault === 'boolean') flipped.value = (prefs as any).boardFlippedDefault
        const cmpMode = (prefs as any).compactLayoutMode
        if (cmpMode === 'auto' || cmpMode === 'compact' || cmpMode === 'full') {
          compactModeOverride.value = cmpMode
        }
        if ((prefs as any).leftColWidth && (prefs as any).leftColWidth >= 240) {
          leftColWidth.value = (prefs as any).leftColWidth
        }
        if ((prefs as any).bottomDrawerHeight && (prefs as any).bottomDrawerHeight >= 100) {
          bottomDrawerHeight.value = (prefs as any).bottomDrawerHeight
        }
        if (typeof (prefs as any).bottomDrawerOpen === 'boolean') {
          bottomDrawerOpen.value = (prefs as any).bottomDrawerOpen
        }
        if ((prefs as any).panelConfigs) {
          hydrateWorkspacePanels((prefs as any).panelConfigs)
        }
      }
    } catch {
      // Ignore in test environment
    }

    // 1. Load sets and themes
    pieceSets.value = (await GetPieceSets()) as PieceSetOption[]
    boardThemes.value = (await GetBoardThemes()) as BoardThemeOption[]

    // 2. Discover user databases and system bases
    discoveredDatabases.value = (await GetDiscoveredDatabases()) as DiscoveredDatabase[]
    systemBases.value = await GetScidSystemBases()

    // 3. Discover available UCI engines and configured Scid engines
    discoveredEngines.value = (await GetDiscoveredEngines()) as EngineOption[]
    availableEngines.value = await GetAvailableEngines()

    if (appPreferences.value.defaultEngine) {
      selectedEngine.value = appPreferences.value.defaultEngine
    } else if (discoveredEngines.value.length > 0) {
      const firstAvailable = discoveredEngines.value.find(e => e.available)
      if (firstAvailable) {
        selectedEngine.value = firstAvailable.name || firstAvailable.cmd
      }
    } else if (availableEngines.value.length > 0) {
      selectedEngine.value = availableEngines.value[0]
    }

    // 4. Listen to live engine events & annotation progress
    EventsOn('engine:eval', (data: EvalInfo) => {
      evalInfo.value = data
    })
    EventsOn('annotate:progress', (data: { currentPly: number; totalPlies: number; percent: number }) => {
      annotationProgress.value = data
    })

    const initMPV = await GetEngineMultiPV()
    if (initMPV > 0) {
      engineMultiPv.value = initMPV
    } else if (appPreferences.value.defaultMultiPv > 0) {
      engineMultiPv.value = appPreferences.value.defaultMultiPv
    }

    prefsInitialized.value = true

    // 5. Auto-open Default Database (or initial DB passed via --db flag, or first discovered user base)
    let autoBase: string | null = null
    if (windowStartupMode.value.initialDb) {
      autoBase = windowStartupMode.value.initialDb
    } else if (appPreferences.value.autoOpenDefaultDb && appPreferences.value.defaultDatabasePath) {
      autoBase = appPreferences.value.defaultDatabasePath
    } else if (discoveredDatabases.value.length > 0) {
      const userBase = discoveredDatabases.value.find(d => d.name === 'ChristopherFair' && d.exists)
        || discoveredDatabases.value.find(d => d.category === 'User Databases' && d.exists && d.numGames > 0)
        || discoveredDatabases.value.find(d => d.exists && d.numGames > 0 && d.numGames < 100000)
      if (userBase) {
        autoBase = userBase.basePath
      }
    }
    if (!autoBase && systemBases.value.length > 0) {
      autoBase = systemBases.value[0]
    }

    if (autoBase) {
      await handleOpenDiscoveredBase(autoBase)
      if (!appPreferences.value.defaultDatabasePath) {
        await handleSetDefaultDatabase(autoBase)
      }
    }
    await refreshDatabaseRolesStatus()

    if (windowStartupMode.value.initialFen) {
      await updateBoardState(windowStartupMode.value.initialFen)
    }

    // Start bidirectional live state synchronization across main and detached Hyprland screen windows!
    if (windowStartupMode.value.mode === 'panel') {
      if (windowStartupMode.value.panelId === 'engine' && !isAnalyzing.value) {
        toggleAnalysis()
      }
    } else {
      await publishCurrentWorkspaceState()
      // Auto-restore detached workspace windows on startup if enabled
      if (appPreferences.value.autoRestoreWorkspaceWindows) {
        for (const p of Object.values(workspacePanels.value)) {
          if (p.mode === 'detached') {
            handleDetachPanelScreen(p.id, p.targetWorkspace || '')
          }
        }
      }
    }

    multiScreenSyncTimer = window.setInterval(async () => {
      try {
        const shared = await GetSharedWorkspaceState()
        if (shared && shared.updatedAtMs > lastSyncedTimestamp && shared.sourceId !== windowInstanceId) {
          lastSyncedTimestamp = shared.updatedAtMs
          if (shared.dbPath && shared.dbPath !== dbInfo.value?.basePath) {
            await handleOpenDiscoveredBase(shared.dbPath)
          }
          if (shared.activeGameId && shared.activeGameId > 0 && shared.activeGameId !== activeGameId.value) {
            await handleLoadGame(shared.activeGameId)
          } else if (shared.fen && shared.fen !== currentFEN.value) {
            lastMove.value = shared.lastMoveFrom && shared.lastMoveTo
              ? { from: shared.lastMoveFrom, to: shared.lastMoveTo }
              : null
            await updateBoardState(shared.fen)
          }
        }
      } catch {
        // Ignore sync poll errors
      }
    }, 550)

    // 6. Check voluntary support status (exempts author machine via Chess.com & Lichess IDs; 24h cooldown on dismiss)
    const checkSupportNag = async () => {
      try {
        const supStatus = await GetSupportStatus()
        if (supStatus && supStatus.shouldShowNag && windowStartupMode.value.mode === 'main') {
          showSupportModal.value = true
        }
      } catch {
        // Ignore support status errors
      }
    }
    await checkSupportNag()
    setInterval(checkSupportNag, 30 * 60 * 1000)

    // Setup ResizeObserver for adaptive right-sizing of the chessboard
    nextTick(() => {
      updateResponsiveBoardSize()
      if (typeof ResizeObserver !== 'undefined') {
        resizeObserver = new ResizeObserver(() => {
          updateResponsiveBoardSize()
        })
        if (leftColRef.value) resizeObserver.observe(leftColRef.value)
        if (workspaceRef.value) resizeObserver.observe(workspaceRef.value)
      }
    })
  } catch (err) {
    console.error('Initialization error:', err)
  }

  window.addEventListener('keydown', handleKeydown)
})

onUnmounted(() => {
  window.removeEventListener('keydown', handleKeydown)
  window.removeEventListener('resize', handleWindowResize)
  if (resizeObserver) {
    resizeObserver.disconnect()
    resizeObserver = null
  }
  EventsOff('engine:eval')
  EventsOff('annotate:progress')
  if (playInterval) clearInterval(playInterval)
  if (multiScreenSyncTimer) clearInterval(multiScreenSyncTimer)
  if (persistConfigTimer) clearTimeout(persistConfigTimer)
})

watch([showExportGameModal, showGuessTheMoveModal, showTimeManagementModal], ([exportOpen, guessOpen, timeOpen]) => {
  if ((exportOpen || guessOpen || timeOpen) && flatMoves.value.length > 0) {
    ParsePGN(buildCurrentGamePGN()).catch(() => {})
  }
})

watch([leftColWidth, bottomDrawerHeight, bottomDrawerOpen, userForcedBottomDrawer, effectiveCompact, showBottomGamesDrawer, showLeftEnginePanel, showLeftImbalancePanel, isAnalyzing, zenMode, showEvalBar, imbalanceReport, activeSimulation, isTheoreticalNovelty, blunderRadarAlert, () => workspacePanels.value.board?.mode], () => {
  nextTick(() => updateResponsiveBoardSize())
})

watch([
  currentPieceSet,
  currentThemeId,
  currentUITheme,
  soundEnabled,
  showEvalBar,
  threatMode,
  blindfoldMode,
  flipped,
  compactModeOverride,
  leftColWidth,
  bottomDrawerHeight,
  bottomDrawerOpen,
  selectedEngine,
  engineMultiPv,
], () => {
  schedulePersistUserConfig()
})

watch(currentFEN, (newFen) => {
  updateTablebase(newFen)
})

watch(activeTab, (newTab) => {
  if (newTab === 'tree') {
    fetchOpeningTree(currentFEN.value)
  } else if (newTab === 'db-roles') {
    refreshDatabaseRolesStatus()
  }
})

// ==========================================
// Board & Move Logic
// ==========================================

async function updateBoardState(fen: string) {
  try {
    const state = await ParseFEN(fen)
    currentFEN.value = state.fen
    legalMoves.value = state.legalMoves
    inCheck.value = state.inCheck

    // Update opening tree for this position
    fetchOpeningTree(state.fen)

    // Update positional imbalance metrics
    await updateImbalance(state.fen)

    // If live engine analysis is active, send position; otherwise clear stale evalInfo from prior plies/bot searches
    if (isAnalyzing.value) {
      AnalyzePosition(state.fen, [], 20).catch(console.error)
    } else {
      evalInfo.value = null
    }

    publishCurrentWorkspaceState()
  } catch (err) {
    console.error('Error updating board state:', err)
  }
}

async function handleMove(moveInput: string) {
  try {
    const evalBefore = evalScore.value
    let moveSAN = moveInput
    try {
      moveSAN = await FormatMove(currentFEN.value, moveInput, 'san')
    } catch {
      // fallback
    }

    const nextState = await MakeMove(currentFEN.value, moveInput)
    const uci = nextState.moveUci || moveInput
    if (nextState.moveSan) {
      moveSAN = nextState.moveSan
    }
    const from = uci.slice(0, 2)
    const to = uci.slice(2, 4)

    // Truncate forward history and flatMoves if moving from middle of game
    history.value = history.value.slice(0, currentPly.value + 1)
    flatMoves.value = flatMoves.value.slice(0, currentPly.value)
    history.value.push({
      fen: nextState.fen,
      lastMove: { from, to },
    })

    currentPly.value = history.value.length - 1
    lastMove.value = { from, to }

    // Add to flatMoves if moving in free mode
    const moveNum = Math.floor((currentPly.value - 1) / 2) + 1
    const isWhite = (currentPly.value % 2) !== 0
    flatMoves.value.push({
      index: flatMoves.value.length,
      ply: currentPly.value,
      moveNumber: moveNum,
      isWhite,
      san: moveSAN,
      uci,
      fen: nextState.fen,
      variationLevel: 0,
    })

    await updateBoardState(nextState.fen)

    // Blunder Radar evaluation check
    const evalAfter = evalScore.value
    const swing = isWhite ? (evalBefore - evalAfter) : (evalAfter - evalBefore)
    if (swing >= 200) {
      triggerBlunderRadar(`⚠️ Blunder Radar: Move dropped evaluation by ${(swing / 100).toFixed(1)} pawns!`)
    }

    // If simulated match is active and game is not finished, trigger bot response!
    if (activeSimulation.value && !isBotThinking.value) {
      if (nextState.isCheckmate) {
        const winner = 'Player'
        gameTags.value.Result = isWhite ? '1-0' : '0-1'
        botMoveStatus.value = `Checkmate! ${winner} wins.`
      } else if (nextState.isStalemate) {
        gameTags.value.Result = '1/2-1/2'
        botMoveStatus.value = 'Game drawn by stalemate.'
      } else {
        await triggerSimulatedOpponentMove()
      }
    }
  } catch (err) {
    console.error('Move execution failed:', err)
  }
}

function jumpToPly(ply: number) {
  if (ply < 0 || ply >= history.value.length) return
  currentPly.value = ply
  const state = history.value[ply]
  lastMove.value = state.lastMove
  updateBoardState(state.fen)
}

function goFirst() {
  jumpToPly(0)
}

function goPrev() {
  if (currentPly.value > 0) {
    jumpToPly(currentPly.value - 1)
  }
}

function goNext() {
  if (currentPly.value < history.value.length - 1) {
    jumpToPly(currentPly.value + 1)
  }
}

function goLast() {
  jumpToPly(history.value.length - 1)
}

function toggleAutoplay() {
  if (isPlaying.value) {
    if (playInterval) clearInterval(playInterval)
    isPlaying.value = false
  } else {
    isPlaying.value = true
    playInterval = window.setInterval(() => {
      if (currentPly.value < history.value.length - 1) {
        goNext()
      } else {
        toggleAutoplay()
      }
    }, 1000)
  }
}

// Trial Mode
function toggleTrialMode() {
  if (trialMode.value) {
    // Exit trial mode: revert to saved state
    if (trialSavedState.value) {
      history.value = trialSavedState.value.history
      flatMoves.value = trialSavedState.value.flatMoves
      currentPly.value = trialSavedState.value.currentPly
      gameTags.value = trialSavedState.value.tags
      updateBoardState(trialSavedState.value.fen)
      trialSavedState.value = null
    }
    trialMode.value = false
  } else {
    // Enter trial mode: save current state
    trialSavedState.value = {
      fen: currentFEN.value,
      history: JSON.parse(JSON.stringify(history.value)),
      flatMoves: JSON.parse(JSON.stringify(flatMoves.value)),
      currentPly: currentPly.value,
      tags: { ...gameTags.value },
    }
    trialMode.value = true
  }
}

// Global Keyboard Shortcuts
function handleKeydown(e: KeyboardEvent) {
  if (e.target instanceof HTMLInputElement || e.target instanceof HTMLTextAreaElement) {
    return
  }

  // Ctrl Combos
  if (e.ctrlKey || e.metaKey) {
    if (e.key === ',') {
      e.preventDefault()
      showPreferencesModal.value = !showPreferencesModal.value
      return
    }
    if (e.key === 'b' || e.key === 'B') {
      e.preventDefault()
      showSetupBoardModal.value = true
    } else if (e.key === ' ' || e.code === 'Space') {
      e.preventDefault()
      toggleTrialMode()
    } else if (e.key === 'o' || e.key === 'O') {
      e.preventDefault()
      handleOpenDB()
    } else if (e.key === 'h' || e.key === 'H') {
      e.preventDefault()
      showSearchModal.value = true
    } else if (e.key === 'n' || e.key === 'N') {
      e.preventDefault()
      handleNewGame()
    } else if (e.key === 's' || e.key === 'S') {
      e.preventDefault()
      handleSaveCurrentGame()
    } else if (e.key === 'Delete') {
      e.preventDefault()
      if (activeGameId.value) handleDeleteGame(activeGameId.value)
    } else if (e.key === 'z' || e.key === 'Z') {
      e.preventDefault()
      goPrev()
    } else if (e.key === 'y' || e.key === 'Y') {
      e.preventDefault()
      goNext()
    } else if (e.key === 'i' || e.key === 'I') {
      e.preventDefault()
      handleImportPGN()
    } else if (e.shiftKey && (e.key === 'e' || e.key === 'E')) {
      e.preventDefault()
      showExportGameModal.value = true
    } else if (e.key === 'e' || e.key === 'E') {
      e.preventDefault()
      handleExportPGN()
    } else if (e.key === 'c' || e.key === 'C') {
      e.preventDefault()
      handleCopyFEN()
    } else if (e.key === 'v' || e.key === 'V') {
      e.preventDefault()
      handlePasteFEN()
    } else if (e.key === 'g' || e.key === 'G') {
      e.preventDefault()
      showGuessTheMoveModal.value = !showGuessTheMoveModal.value
    } else if (e.key === 'd' || e.key === 'D') {
      e.preventDefault()
      showDatabaseStatsModal.value = !showDatabaseStatsModal.value
    } else if (e.key === 't' || e.key === 'T') {
      e.preventDefault()
      handleOpenEditTags()
    } else if (e.key === 'k' || e.key === 'K') {
      e.preventDefault()
      showCommandPalette.value = !showCommandPalette.value
    }
    return
  }

  // Single Keys
  if (e.key === 'F1') {
    e.preventDefault()
    showHelpModal.value = true
    return
  } else if (e.key === 'F2') {
    e.preventDefault()
    toggleAnalysis()
    return
  } else if (e.key === 'F3') {
    e.preventDefault()
    handleOpenTreeWindow()
    return
  } else if (e.key === 'F6') {
    e.preventDefault()
    showEcoBrowserModal.value = !showEcoBrowserModal.value
    return
  } else if (e.key === 'F7') {
    e.preventDefault()
    showTournamentCrosstableModal.value = !showTournamentCrosstableModal.value
    return
  } else if (e.key === 'F10') {
    e.preventDefault()
    toggleDarkLight()
    return
  } else if (e.key === '?') {
    e.preventDefault()
    showShortcutsModal.value = true
    return
  }

  // Guard single-key board navigation, flip, and autoplay when any modal is open
  if (
    showSetupBoardModal.value ||
    showPlayEngineModal.value ||
    showTacticsModal.value ||
    showOpeningTrainerModal.value ||
    showProgressiveTacticsModal.value ||
    showPuzzleMinerModal.value ||
    showShortcutsModal.value ||
    showAboutModal.value ||
    showHelpModal.value ||
    showSearchModal.value ||
    showMaintenanceModal.value ||
    showOnlineSyncModal.value ||
    showPlayerSimulationModal.value ||
    showPersonaModal.value ||
    showTournamentCrosstableModal.value ||
    showEcoBrowserModal.value ||
    showGuessTheMoveModal.value ||
    showDatabaseStatsModal.value ||
    showEditTagsModal.value ||
    showTwicModal.value ||
    showEngineMatchModal.value ||
    showExportGameModal.value ||
    showChess960Modal.value ||
    showCoordinateTrainerModal.value ||
    showCommandPalette.value ||
    showExplainMoveModal.value ||
    showBlunderVaultModal.value ||
    showOpponentScoutModal.value ||
    showAntiPersonaModal.value ||
    showPieceSafetyModal.value ||
    showIntuitionStormModal.value ||
    showTimeManagementModal.value ||
    showRepertoireShortenerModal.value ||
    showProphylaxisModal.value ||
    showSupportModal.value ||
    showMegaDatabaseModal.value ||
    showFairPlayModal.value ||
    showPreferencesModal.value ||
    showImportModal.value
  ) {
    return
  }

  if (e.key === 'ArrowLeft') {
    goPrev()
  } else if (e.key === 'ArrowRight') {
    goNext()
  } else if (e.key === 'Home') {
    goFirst()
  } else if (e.key === 'End') {
    goLast()
  } else if (e.key === 'f' || e.key === 'F') {
    flipped.value = !flipped.value
  } else if (e.key === 'z' || e.key === 'Z') {
    toggleZenMode()
  } else if (e.key === ' ') {
    e.preventDefault()
    toggleAutoplay()
  }
}

async function fetchOpeningTree(fen?: any) {
  const targetFen = (typeof fen === 'string' && fen.trim()) ? fen.trim() : (currentFEN.value || startingFEN)
  treeLoading.value = true
  try {
    const res = await GetOpeningTree(targetFen)
    treeData.value = res as TreeData
  } catch (err) {
    console.error('Error fetching opening tree:', err)
  } finally {
    treeLoading.value = false
  }
}

// ==========================================
// Database Operations
// ==========================================

async function refreshGames() {
  try {
    const offset = (currentPage.value - 1) * pageSize.value
    const page = await GetGames(offset, pageSize.value, searchFilter.value as any)
    gamesList.value = page.games as GameSummary[]
    totalGames.value = page.total
  } catch (err) {
    console.error('Error loading games:', err)
  }
}

async function handleOpenDB() {
  try {
    const path = await OpenFileDialog()
    if (!path) return
    const info = await OpenDatabase(path)
    dbInfo.value = info
    currentPage.value = 1
    await refreshGames()
    if (gamesList.value.length > 0) {
      await handleLoadGame(1)
    }
    await refreshDatabaseRolesStatus()
    await publishCurrentWorkspaceState()
  } catch (err) {
    alert(`Failed to open database: ${err}`)
  }
}

async function handleCreateDB() {
  try {
    const path = await SaveFileDialog('mygames')
    if (!path) return
    const info = await CreateDatabase(path, 'My Chess Database')
    dbInfo.value = info
    currentPage.value = 1
    await refreshGames()
    await refreshDatabaseRolesStatus()
    await publishCurrentWorkspaceState()
  } catch (err) {
    alert(`Failed to create database: ${err}`)
  }
}

async function handleCloseDB() {
  try {
    await CloseDatabase()
    dbInfo.value = null
    gamesList.value = []
    totalGames.value = 0
    currentPage.value = 1
    handleNewGame()
    await refreshDatabaseRolesStatus()
    await publishCurrentWorkspaceState()
  } catch (err) {
    alert(`Failed to close database: ${err}`)
  }
}

async function handleOpenSystemBase(path: string) {
  try {
    const info = await OpenDatabase(path)
    dbInfo.value = info
    currentPage.value = 1
    await refreshGames()
    if (gamesList.value.length > 0) {
      await handleLoadGame(1)
    }
    await refreshDatabaseRolesStatus()
    await publishCurrentWorkspaceState()
  } catch (err) {
    console.error('Failed to open sample base:', err)
  }
}

async function handleOpenDiscoveredBase(path: string) {
  try {
    const info = await OpenDatabase(path)
    dbInfo.value = info
    currentPage.value = 1
    searchFilter.value = {}
    await refreshGames()
    if (gamesList.value.length > 0) {
      await handleLoadGame(1)
    }
    bottomDrawerOpen.value = true
    await refreshDatabaseRolesStatus()
    await publishCurrentWorkspaceState()
  } catch (err) {
    console.error('Failed to open database:', err)
    alert(`Failed to open database: ${err}`)
  }
}

async function handleDatabaseUpdated() {
  await refreshGames()
  const info = await GetDatabaseInfo()
  if (info) {
    dbInfo.value = info
    totalGames.value = info.numGames
  }
  await refreshDatabaseRolesStatus()
}

async function handleLoadGame(id: number) {
  try {
    const g = await GetGame(id)
    activeGameId.value = id
    gameTags.value = g.tags || {}
    gameAnalysis.value = null

    // Build history from game move tree
    const newHistory: { fen: string; lastMove: { from: string; to: string } | null }[] = [
      { fen: g.startFen || startingFEN, lastMove: null },
    ]
    const moves: FlatMove[] = []

    let curr = g.rootMove
    let ply = 1
    while (curr) {
      const from = curr.uci.slice(0, 2)
      const to = curr.uci.slice(2, 4)
      newHistory.push({
        fen: curr.fen,
        lastMove: { from, to },
      })
      moves.push({
        index: moves.length,
        ply,
        moveNumber: Math.floor((ply - 1) / 2) + 1,
        isWhite: ply % 2 !== 0,
        san: curr.san,
        uci: curr.uci,
        fen: curr.fen,
        comment: curr.comment,
        clock: (curr as any).clock,
        emt: (curr as any).emt,
        nags: curr.nags,
        variationLevel: 0,
      })
      curr = curr.next
      ply++
    }

    history.value = newHistory
    flatMoves.value = moves

    jumpToPly(0)

    const curTab = openGameTabs.value[activeGameTabIdx.value]
    if (curTab) {
      curTab.title = `${g.tags?.White || 'White'} vs ${g.tags?.Black || 'Black'}`
      curTab.activeGameId = id
      curTab.gameTags = { ...g.tags }
      curTab.history = [...newHistory]
      curTab.flatMoves = [...moves]
      curTab.currentPly = 0
      curTab.gameAnalysis = null
    }
  } catch (err) {
    alert(`Error loading game ${id}: ${err}`)
  }
}

async function handleSaveCurrentGame() {
  if (!dbInfo.value) {
    const shouldCreate = confirm('No database is currently open. Would you like to create a new database to save this game?')
    if (shouldCreate) {
      await handleCreateDB()
      if (!dbInfo.value) return
    } else {
      return
    }
  }
  try {
    const pgn = buildCurrentGamePGN()
    const parsed = await ParsePGN(pgn)
    const gameObj: any = {
      ...(parsed || {}),
      tags: { ...(parsed?.tags || {}), ...gameTags.value },
      startFen: history.value[0]?.fen || parsed?.startFen || startingFEN,
      result: gameTags.value.Result || parsed?.result || '*',
      plyCount: flatMoves.value.length,
    }
    const newId = await SaveGame(gameObj)
    activeGameId.value = newId
    alert(`Game saved successfully as Game #${newId}`)
    await refreshGames()
  } catch (err) {
    alert(`Error saving game: ${err}`)
  }
}

async function handleDeleteGame(id: number) {
  if (!confirm(`Are you sure you want to delete game #${id}?`)) return
  try {
    await DeleteGame(id, true)
    await refreshGames()
  } catch (err) {
    alert(`Error deleting game: ${err}`)
  }
}

async function handleDeleteCurrentGame() {
  if (activeGameId.value) {
    await handleDeleteGame(activeGameId.value)
  } else if (!dbInfo.value) {
    alert('No database is currently open.')
  } else {
    alert('No game from the database is currently loaded to delete.')
  }
}

function handleSearch(criteria: any) {
  searchFilter.value = criteria || {}
  currentPage.value = 1
  refreshGames()
}

function handlePageChange(p: number) {
  currentPage.value = p
  refreshGames()
}

function handleNewGame() {
  activeGameId.value = undefined
  gameAnalysis.value = null
  gameTags.value = {
    White: 'Player 1',
    Black: 'Player 2',
    Date: new Date().toISOString().split('T')[0].replace(/-/g, '.'),
    Result: '*',
    Event: 'Casual Game',
  }
  history.value = [{ fen: startingFEN, lastMove: null }]
  flatMoves.value = []
  jumpToPly(0)

  const curTab = openGameTabs.value[activeGameTabIdx.value]
  if (curTab) {
    curTab.title = `Game ${activeGameTabIdx.value + 1}`
    curTab.activeGameId = undefined
    curTab.gameTags = { ...gameTags.value }
    curTab.history = [{ fen: startingFEN, lastMove: null }]
    curTab.flatMoves = []
    curTab.currentPly = 0
    curTab.gameAnalysis = null
  }
}

// Game Navigation
function handleFirstGame() {
  if (totalGames.value > 0) {
    handleLoadGame(1)
  } else {
    alert('No database is open or no games available.')
  }
}

function handlePrevGame() {
  if (activeGameId.value && activeGameId.value > 1) {
    handleLoadGame(activeGameId.value - 1)
  } else if (!activeGameId.value && totalGames.value > 0) {
    handleLoadGame(1)
  } else if (activeGameId.value === 1) {
    alert('Already at the first game in the database.')
  } else {
    alert('No database is open or no games available.')
  }
}

function handleNextGame() {
  if (activeGameId.value && activeGameId.value < totalGames.value) {
    handleLoadGame(activeGameId.value + 1)
  } else if (!activeGameId.value && totalGames.value > 0) {
    handleLoadGame(1)
  } else if (activeGameId.value && activeGameId.value >= totalGames.value) {
    alert('Already at the last game in the database.')
  } else {
    alert('No database is open or no games available.')
  }
}

function handleLastGame() {
  if (totalGames.value > 0) {
    handleLoadGame(totalGames.value)
  } else {
    alert('No database is open or no games available.')
  }
}

function handleRandomGame() {
  if (totalGames.value > 0) {
    const rand = Math.floor(Math.random() * totalGames.value) + 1
    handleLoadGame(rand)
  } else {
    alert('No database is open or no games available.')
  }
}

// Position Search
async function handleSearchPosition() {
  if (!dbInfo.value) {
    alert('Please open a database before searching for board positions.')
    return
  }
  try {
    const matches = await SearchPosition(currentFEN.value)
    if (!matches || matches.length === 0) {
      alert('No games found in the database reaching this board position.')
      return
    }
    gamesList.value = matches as GameSummary[]
    totalGames.value = matches.length
    currentPage.value = 1
    bottomDrawerOpen.value = true
    alert(`Found ${matches.length} game(s) reaching this position! Filter applied to game browser.`)
  } catch (err) {
    alert(`Position search failed: ${err}`)
  }
}

// Setup Board
function handleSetupBoard() {
  showSetupBoardModal.value = true
}

async function handleSetupBoardApply(newFen: string) {
  try {
    await updateBoardState(newFen)
    history.value = [{ fen: newFen, lastMove: null }]
    flatMoves.value = []
    currentPly.value = 0
    gameAnalysis.value = null
    gameTags.value = {
      White: 'Custom Position',
      Black: 'Custom Position',
      Date: new Date().toISOString().split('T')[0].replace(/-/g, '.'),
      Result: '*',
      Event: 'Board Setup',
    }
  } catch (err) {
    alert(`Failed to set up position: ${err}`)
  }
}

// Copy / Paste FEN
async function handleCopyFEN() {
  try {
    let success = false
    try {
      success = await ClipboardSetText(currentFEN.value)
    } catch {
      // fallback
    }
    if (!success && navigator.clipboard) {
      await navigator.clipboard.writeText(currentFEN.value)
      success = true
    }
    alert(`FEN copied to clipboard:\n${currentFEN.value}`)
  } catch (err) {
    console.error('Failed to copy FEN:', err)
    alert(`Failed to copy FEN: ${err}`)
  }
}

async function handlePasteFEN() {
  try {
    let text = ''
    try {
      text = await ClipboardGetText()
    } catch {
      // fallback
    }
    if (!text && navigator.clipboard) {
      text = await navigator.clipboard.readText()
    }
    if (!text || !text.trim()) {
      alert('Clipboard does not contain valid text.')
      return
    }
    await handleSetupBoardApply(text.trim())
  } catch (err) {
    alert(`Failed to paste FEN: ${err}`)
  }
}

// Helper to build full PGN string including headers and moves
function buildCurrentGamePGN(): string {
  const tags = { ...gameTags.value }
  if (!tags.Event) tags.Event = 'Casual Game'
  if (!tags.Site) tags.Site = 'NewScid'
  if (!tags.Date) tags.Date = new Date().toISOString().split('T')[0].replace(/-/g, '.')
  if (!tags.Round) tags.Round = '1'
  if (!tags.White) tags.White = 'White'
  if (!tags.Black) tags.Black = 'Black'
  if (!tags.Result) tags.Result = '*'

  const startFen = history.value[0]?.fen
  if (startFen && startFen !== startingFEN) {
    tags.SetUp = '1'
    tags.FEN = startFen
  }

  let pgn = ''
  const standardTagOrder = ['Event', 'Site', 'Date', 'Round', 'White', 'Black', 'Result']
  for (const tag of standardTagOrder) {
    pgn += `[${tag} "${tags[tag] || '?' }"]\n`
  }
  for (const [k, v] of Object.entries(tags)) {
    if (!standardTagOrder.includes(k) && v) {
      pgn += `[${k} "${v}"]\n`
    }
  }
  pgn += '\n'

  let moveText = ''
  for (let i = 0; i < flatMoves.value.length; i++) {
    const m = flatMoves.value[i]
    if (m.isWhite) {
      moveText += `${m.moveNumber}. ${m.san}`
    } else {
      if (i === 0) {
        moveText += `${m.moveNumber}... ${m.san}`
      } else {
        moveText += ` ${m.san}`
      }
    }

    if (m.nags && m.nags.length > 0) {
      for (const nag of m.nags) {
        moveText += ` $${nag}`
      }
    }

    const commentTokens: string[] = []
    if (m.comment) {
      const clean = m.comment.replace(/\[%clk\s+[^\]]+\]/gi, '').replace(/\[%emt\s+[^\]]+\]/gi, '').trim()
      if (clean) {
        commentTokens.push(clean)
      }
    }
    if (m.clock) {
      commentTokens.push(`[%clk ${m.clock}]`)
    }
    if (m.emt) {
      commentTokens.push(`[%emt ${m.emt}]`)
    }
    if (commentTokens.length > 0) {
      moveText += ` {${commentTokens.join(' ')}}`
    }

    moveText += ' '
  }

  const result = tags.Result || '*'
  moveText += result

  return pgn + moveText.trim() + '\n'
}

// Full Game Analysis (Annotate)
async function handleAnnotateGame() {
  if (isAnalyzingGame.value) return
  if (flatMoves.value.length === 0) {
    alert('There are no moves in the current game to annotate. Play some moves on the board or load a game first.')
    return
  }
  isAnalyzingGame.value = true
  annotationProgress.value = { currentPly: 0, totalPlies: flatMoves.value.length, percent: 0 }
  try {
    const pgn = buildCurrentGamePGN()
    const analysis = await AnnotatePGN(pgn, 150)
    gameAnalysis.value = analysis as GameAnalysisData

    // Update active tab snapshot
    const curTab = openGameTabs.value[activeGameTabIdx.value]
    if (curTab) {
      curTab.gameAnalysis = analysis as GameAnalysisData
    }

    // Update flatMoves with NAGs and comments
    if (analysis.moveAnalyses) {
      for (const ma of analysis.moveAnalyses) {
        const move = flatMoves.value.find(m => m.ply === ma.ply)
        if (move) {
          move.comment = ma.comment
          if (ma.nag) {
            move.nags = [ma.nag]
          }
        }
      }
    }
    activeTab.value = 'notation'
    alert(`Annotation Complete!\nWhite Accuracy: ${analysis.whiteAccuracy}%\nBlack Accuracy: ${analysis.blackAccuracy}%\nBlunders: White ${analysis.whiteBlunders}, Black ${analysis.blackBlunders}`)
  } catch (err) {
    alert(`Game annotation failed: ${err}`)
  } finally {
    isAnalyzingGame.value = false
    annotationProgress.value = null
  }
}

// Tactics Trainer
function handleTacticsTrainer() {
  showTacticsModal.value = true
}

async function handleLoadTacticsIntoApp(fen: string) {
  await handleSetupBoardApply(fen)
}

// Play Against Engine
function handlePlayVsEngine() {
  showPlayEngineModal.value = true
}

async function handleImportGameFromMatch(pgn: string) {
  importPGNText.value = pgn
  await confirmImportPGN()
}

async function handlePlayOpeningMoves(movesStr: string) {
  if (!movesStr) return
  const pgn = `[Event "ECO Opening Line"]\n[Result "*"]\n\n${movesStr} *`
  importPGNText.value = pgn
  await confirmImportPGN()
}

function handleSearchEcoInDb(ecoCode: string) {
  if (!ecoCode) return
  searchFilter.value = { ...searchFilter.value, eco: ecoCode }
  currentPage.value = 1
  refreshGames()
}

// ==========================================
// PGN Import / Export
// ==========================================

async function handleImportPGN() {
  showImportModal.value = true
}

async function confirmImportPGN() {
  if (!importPGNText.value.trim()) return
  try {
    const parsed = await ParsePGN(importPGNText.value)
    showImportModal.value = false
    importPGNText.value = ''
    gameAnalysis.value = null

    gameTags.value = parsed.tags || {}
    const newHistory: { fen: string; lastMove: { from: string; to: string } | null }[] = [
      { fen: parsed.startFen || startingFEN, lastMove: null },
    ]
    const moves: FlatMove[] = []

    let curr = parsed.rootMove
    let ply = 1
    while (curr) {
      const from = curr.uci.slice(0, 2)
      const to = curr.uci.slice(2, 4)
      newHistory.push({
        fen: curr.fen,
        lastMove: { from, to },
      })
      moves.push({
        index: moves.length,
        ply,
        moveNumber: Math.floor((ply - 1) / 2) + 1,
        isWhite: ply % 2 !== 0,
        san: curr.san,
        uci: curr.uci,
        fen: curr.fen,
        comment: curr.comment,
        clock: (curr as any).clock,
        emt: (curr as any).emt,
        nags: curr.nags,
        variationLevel: 0,
      })
      curr = curr.next
      ply++
    }

    history.value = newHistory
    flatMoves.value = moves
    jumpToPly(0)
  } catch (err) {
    alert(`Failed to parse PGN: ${err}`)
  }
}

async function handleExportPGN() {
  try {
    const pgn = buildCurrentGamePGN()
    let success = false
    try {
      success = await ClipboardSetText(pgn)
    } catch {
      // fallback
    }
    if (!success && navigator.clipboard) {
      await navigator.clipboard.writeText(pgn)
      success = true
    }
    alert('PGN copied to clipboard!')
  } catch (err) {
    alert(`Error exporting PGN: ${err}`)
  }
}

// ==========================================
// Engine Control
// ==========================================

async function toggleAnalysis() {
  if (isAnalyzing.value) {
    await StopEngine()
    isAnalyzing.value = false
  } else {
    isAnalyzing.value = true
    AnalyzePosition(currentFEN.value, [], 20).catch(err => {
      alert(`Engine error: ${err}`)
      isAnalyzing.value = false
    })
  }
}

async function handleSelectEngine(path: string) {
  selectedEngine.value = path
  if (isAnalyzing.value) {
    await StopEngine()
    await StartEngine(path)
    AnalyzePosition(currentFEN.value, [], 20)
  }
}

async function handleChangeMultiPV(lines: number) {
  engineMultiPv.value = lines
  try {
    await SetEngineMultiPV(lines)
    if (isAnalyzing.value) {
      await AnalyzePosition(currentFEN.value, [], 20)
    }
  } catch (err) {
    console.error('Failed to set MultiPV:', err)
  }
}

async function handleChangeThreads(threads: number) {
  try {
    await SetEngineThreads(threads)
    if (isAnalyzing.value) {
      await AnalyzePosition(currentFEN.value, [], 20)
    }
  } catch (err) {
    console.error('Failed to set engine threads:', err)
  }
}

async function handleChangeHash(mb: number) {
  try {
    await SetEngineHash(mb)
    if (isAnalyzing.value) {
      await AnalyzePosition(currentFEN.value, [], 20)
    }
  } catch (err) {
    console.error('Failed to set engine hash:', err)
  }
}

function handlePreviewCandidateMove(uci: string) {
  if (!uci || uci.length < 4) return
  const from = uci.slice(0, 2)
  const to = uci.slice(2, 4)
  lastMove.value = { from, to }
}

// ==========================================
// Annotation, NAG & Variation Operations
// ==========================================

async function handleSetNAG(ply: number, nag: number) {
  const move = flatMoves.value.find(m => m.ply === ply)
  if (move) {
    move.nags = [nag]
  }
  try {
    await SetGameNAG(ply, nag)
  } catch (err) {
    // Ignore backend missing active game if moving in freeboard mode
  }
}

async function handleClearNAGs(ply: number) {
  const move = flatMoves.value.find(m => m.ply === ply)
  if (move) {
    move.nags = []
  }
  try {
    await ClearGameNAGs(ply)
  } catch (err) {
    // Ignore backend missing active game if moving in freeboard mode
  }
}

async function handleSetComment(ply: number, comment: string) {
  const move = flatMoves.value.find(m => m.ply === ply)
  if (move) {
    move.comment = comment
  }
  try {
    await AddGameComment(ply, comment)
  } catch (err) {
    // Ignore backend missing active game if moving in freeboard mode
  }
}

async function handlePromoteVariation(ply: number, varIdx: number) {
  try {
    await PromoteVariation(ply, varIdx)
    if (activeGameId.value) {
      await handleLoadGame(activeGameId.value)
    }
  } catch (err) {
    alert(`Could not promote variation: ${err}`)
  }
}

async function handleDeleteVariation(ply: number, varIdx: number) {
  try {
    await DeleteVariation(ply, varIdx)
    if (activeGameId.value) {
      await handleLoadGame(activeGameId.value)
    }
  } catch (err) {
    alert(`Could not delete variation: ${err}`)
  }
}

// ==========================================
// Player Simulation Sparring Bot Logic
// ==========================================

async function handleStartSimulationMatch(payload: {
  playerName: string
  personaId?: string
  userColor: 'white' | 'black'
  timeControlMinutes: number
  personality: playerbot.PlayerPersonality
}) {
  showPlayerSimulationModal.value = false
  activeSimulation.value = payload
  botMoveStatus.value = `Sparring Match vs ${payload.playerName} (Elo ~${payload.personality.estimatedElo || 2400})`

  // Start fresh game with proper tags
  activeGameId.value = undefined
  gameAnalysis.value = null
  const whiteName = payload.userColor === 'white' ? 'Player' : `${payload.playerName} (Simulated)`
  const blackName = payload.userColor === 'black' ? 'Player' : `${payload.playerName} (Simulated)`
  gameTags.value = {
    White: whiteName,
    Black: blackName,
    WhiteElo: payload.userColor === 'black' ? String(payload.personality.peakElo || payload.personality.estimatedElo || 2400) : '1500',
    BlackElo: payload.userColor === 'white' ? String(payload.personality.peakElo || payload.personality.estimatedElo || 2400) : '1500',
    Date: new Date().toISOString().split('T')[0].replace(/-/g, '.'),
    Result: '*',
    Event: `Simulated Match vs ${payload.playerName}`,
  }
  history.value = [{ fen: startingFEN, lastMove: null }]
  flatMoves.value = []
  flipped.value = payload.userColor === 'black'
  await updateBoardState(startingFEN)

  // If simulated bot is White, trigger bot's opening move!
  if (payload.userColor === 'black') {
    await triggerSimulatedOpponentMove()
  }
}

function stopSimulation() {
  activeSimulation.value = null
  botMoveStatus.value = null
  isBotThinking.value = false
}

async function triggerSimulatedOpponentMove() {
  if (!activeSimulation.value || isBotThinking.value) return
  isBotThinking.value = true
  botMoveStatus.value = `${activeSimulation.value.playerName} is thinking...`
  try {
    const isBotWhite = activeSimulation.value.userColor === 'black'
    await new Promise(r => setTimeout(r, 400))
    let simMove: any
    if (activeSimulation.value.personaId) {
      simMove = await GetPersonaSimulatedMove(
        activeSimulation.value.personaId,
        currentFEN.value,
        isBotWhite
      )
    } else {
      simMove = await GetSimulatedMove(
        activeSimulation.value.playerName,
        currentFEN.value,
        isBotWhite
      )
    }
    if (!simMove || !simMove.uci) {
      botMoveStatus.value = `${activeSimulation.value.playerName} has no moves.`
      return
    }

    botMoveStatus.value = simMove.commentary || `${activeSimulation.value.playerName} played ${simMove.san || simMove.uci}`

    const nextState = await MakeMove(currentFEN.value, simMove.uci)
    const from = simMove.uci.slice(0, 2)
    const to = simMove.uci.slice(2, 4)

    history.value.push({
      fen: nextState.fen,
      lastMove: { from, to },
    })
    currentPly.value = history.value.length - 1
    lastMove.value = { from, to }

    const moveNum = Math.floor((currentPly.value - 1) / 2) + 1
    const isWhite = (currentPly.value % 2) !== 0
    flatMoves.value.push({
      index: flatMoves.value.length,
      ply: currentPly.value,
      moveNumber: moveNum,
      isWhite,
      san: simMove.san || simMove.uci,
      uci: simMove.uci,
      fen: nextState.fen,
      comment: simMove.commentary,
      variationLevel: 0,
    })

    await updateBoardState(nextState.fen)

    if (nextState.isCheckmate) {
      const winner = activeSimulation.value.playerName
      gameTags.value.Result = isBotWhite ? '1-0' : '0-1'
      botMoveStatus.value = `Checkmate! ${winner} wins.`
    } else if (nextState.isStalemate) {
      gameTags.value.Result = '1/2-1/2'
      botMoveStatus.value = 'Game drawn by stalemate.'
    }
  } catch (err: any) {
    console.error('Simulated bot move failed:', err)
    botMoveStatus.value = `Bot move error: ${err?.message || err}`
  } finally {
    isBotThinking.value = false
  }
}

function handleSparPersona(p: persona.Persona) {
  selectedSimulationPersona.value = p
  showPersonaModal.value = false
  showPlayerSimulationModal.value = true
}

async function handleOpenPersonaDb(p: persona.Persona) {
  if (p.databasePath) {
    showPersonaModal.value = false
    await handleOpenDiscoveredBase(p.databasePath)
  }
}

async function handleSparBotFromAnti(personaId: string, playerName: string) {
  showAntiPersonaModal.value = false
  if (personaId) {
    try {
      const p = await GetPersona(personaId)
      if (p) {
        selectedSimulationPersona.value = p
        showPlayerSimulationModal.value = true
        return
      }
    } catch {}
  }
  showPlayerSimulationModal.value = true
}

function handleTrainAntiRepertoire() {
  showAntiPersonaModal.value = false
  showOpeningTrainerModal.value = true
}

function handleOpenWebDocs() {
  try {
    BrowserOpenURL('https://github.com/christopherpatrickfair/newscid#readme')
  } catch (err) {
    window.open('https://github.com/christopherpatrickfair/newscid#readme', '_blank')
  }
}

function handleOpenEditTags() {
  if (!activeGameId.value && totalGames.value > 0) {
    handleLoadGame(1)
  }
  showEditTagsModal.value = true
}

function handleTagsUpdated(summary: scid.GameSummary) {
  gameTags.value = {
    ...gameTags.value,
    White: summary.white,
    Black: summary.black,
    WhiteElo: summary.whiteElo ? String(summary.whiteElo) : '',
    BlackElo: summary.blackElo ? String(summary.blackElo) : '',
    Event: summary.event,
    Site: summary.site,
    Date: summary.date,
    Round: summary.round,
    Result: summary.result,
    ECO: summary.eco
  }
  refreshGames()
}

function handleTwicImported(count: number) {
  refreshGames()
}

function handleMatchFinished(summary: tournament.EngineMatchSummary) {
  refreshGames()
}

async function handleStandardizeNames() {
  if (!dbInfo.value?.name) return
  try {
    const summary = await StandardizePlayerNames()
    if (!summary) return
    if (summary.correctionsMade === 0) {
      alert(`✓ Player Name Standardization:\nAll ${summary.totalPlayersChecked} player names in ${dbInfo.value.name} are already standardized!`)
    } else {
      alert(`✓ Player Name Standardization Complete:\n• Checked: ${summary.totalPlayersChecked} players\n• Corrections: ${summary.correctionsMade} player names standardized\n• Games updated: ${summary.gamesUpdated}\n\nDatabase index and name records updated successfully.`)
      refreshGames()
    }
  } catch (err: any) {
    alert(`Failed to standardize player names: ${err?.message || err}`)
  }
}

// Universal Command Palette & Spotlight Commands List
const commandPaletteItems = computed<CommandItem[]>(() => [
  {
    id: 'cmd-palette',
    title: 'Command Palette',
    subtitle: 'Search all commands, study tools, and database features',
    icon: 'search',
    category: 'Actions',
    shortcut: 'Ctrl+K',
    action: () => { showCommandPalette.value = true }
  },
  {
    id: 'zen-mode',
    title: zenMode.value ? 'Exit Zen Mode' : 'Enter Zen / Classical Study Mode',
    subtitle: 'Distraction-free focus board layout for calculation and analysis',
    icon: 'maximize-2',
    category: 'Options',
    shortcut: 'Z',
    action: toggleZenMode
  },
  {
    id: 'theme-toggle-dark-light',
    title: 'Toggle Dark / Light Mode',
    subtitle: 'Switch application interface between dark and light appearance',
    icon: 'sun',
    category: 'Options',
    shortcut: 'F10',
    action: toggleDarkLight,
  },
  ...UI_THEMES.map((t) => ({
    id: `ui-theme-${t.id}`,
    title: `UI Theme: ${t.name}`,
    subtitle: `${t.description} (${t.category.toUpperCase()})`,
    icon: 'palette',
    category: 'Options' as const,
    action: () => applyUITheme(t.id),
  })),
  {
    id: 'explain-move',
    title: 'Explain This Move',
    subtitle: 'Natural language conceptual breakdown of tactical motifs & plans',
    icon: 'sparkles',
    category: 'Analysis',
    action: () => handleExplainMove(currentPly.value)
  },
  {
    id: 'blunder-vault',
    title: 'Personal Blunder Vault',
    subtitle: 'SuperMemo SM-2 Spaced Repetition personal mistake trainer',
    icon: 'brain',
    category: 'Training',
    action: () => { showBlunderVaultModal.value = true }
  },
  {
    id: 'opponent-scout',
    title: 'Opponent Scouting Card & Prep Matrix',
    subtitle: 'Generate preparation dossier and repertoire vulnerability report',
    icon: 'shield',
    category: 'Training',
    action: () => { showOpponentScoutModal.value = true }
  },
  {
    id: 'anti-persona-exploiter',
    title: 'Opponent Opening Exploiter & Custom Book',
    subtitle: 'Evaluate opponent vulnerabilities and compile weaponized Polyglot opening book',
    icon: 'crosshair',
    category: 'Training',
    action: () => { openAntiPersonaModal() }
  },
  {
    id: 'new-game',
    title: 'New Game',
    subtitle: 'Clear board and start fresh game',
    icon: 'file-text',
    category: 'Actions',
    shortcut: 'Ctrl+N',
    action: handleNewGame
  },
  {
    id: 'open-db',
    title: 'Open Database...',
    subtitle: 'Open an existing Scid database',
    icon: 'folder-open',
    category: 'Actions',
    shortcut: 'Ctrl+O',
    action: handleOpenDB
  },
  {
    id: 'save-game',
    title: 'Save Current Game',
    subtitle: 'Save or append game into database',
    icon: 'save',
    category: 'Actions',
    shortcut: 'Ctrl+S',
    action: handleSaveCurrentGame
  },
  {
    id: 'setup-board',
    title: 'Setup Board / FEN Editor',
    subtitle: 'Custom position setup and FEN verification',
    icon: 'sliders',
    category: 'Actions',
    shortcut: 'Ctrl+B',
    action: () => { showSetupBoardModal.value = true }
  },
  {
    id: 'search-games',
    title: 'Search Database Games',
    subtitle: 'Filter games by player, rating, result, event, ECO',
    icon: 'search',
    category: 'Actions',
    shortcut: 'Ctrl+H',
    action: () => { showSearchModal.value = true }
  },
  {
    id: 'tactics-trainer',
    title: 'Tactics Puzzle Trainer',
    subtitle: 'Solve tactical puzzles with rating progression',
    icon: 'award',
    category: 'Training',
    action: handleTacticsTrainer
  },
  {
    id: 'guess-the-move',
    title: 'Guess the Move (Solitaire Chess)',
    subtitle: 'Play master games move by move and score prediction accuracy',
    icon: 'award',
    category: 'Training',
    shortcut: 'Ctrl+G',
    action: () => { showGuessTheMoveModal.value = true }
  },
  {
    id: 'progressive-tactics',
    title: 'Progressive Tactics Challenge',
    subtitle: 'Survival puzzle rush ladder against time',
    icon: 'zap',
    category: 'Training',
    action: () => { showProgressiveTacticsModal.value = true }
  },
  {
    id: 'opening-tree',
    title: 'Opening Tree Window (F3)',
    subtitle: 'Explore master opening statistics, candidate moves & Polyglot books',
    icon: 'book-open',
    category: 'Openings',
    shortcut: 'F3',
    action: handleOpenTreeWindow
  },
  {
    id: 'opening-trainer',
    title: 'Opening Repertoire Builder & Trainer',
    subtitle: 'Learn and quiz your opening lines',
    icon: 'book-open',
    category: 'Openings',
    action: () => { showOpeningTrainerModal.value = true }
  },
  {
    id: 'coordinate-trainer',
    title: 'Coordinate & Vision Trainer',
    subtitle: 'Practice board algebraic square recognition speed',
    icon: 'target',
    category: 'Training',
    action: () => { showCoordinateTrainerModal.value = true }
  },
  {
    id: 'piece-safety',
    title: "Piece Safety Scanner (Don't Hang Pieces)",
    subtitle: 'Rapid tactical blunder detection and piece safety audit',
    icon: 'shield',
    category: 'Training',
    action: () => { showPieceSafetyModal.value = true }
  },
  {
    id: 'intuition-storm',
    title: 'Intuition Storm (Speed Reflex Trainer)',
    subtitle: 'Rapid instinctual evaluation reflex trainer under fast countdown',
    icon: 'zap',
    category: 'Training',
    action: () => { showIntuitionStormModal.value = true }
  },
  {
    id: 'time-management',
    title: 'Time Management & Tanking Dossier',
    subtitle: 'Deep clock telemetry, tanks, panics, and zeitnot vulnerability audit',
    icon: 'clock',
    category: 'Tools',
    action: () => { showTimeManagementModal.value = true }
  },
  {
    id: 'repertoire-shortener',
    title: 'Repertoire Shortener (Pareto 80/20 Pruner)',
    subtitle: 'Prune low-frequency opening sidelines to master the 80% core lines',
    icon: 'scissors',
    category: 'Training',
    action: () => { showRepertoireShortenerModal.value = true }
  },
  {
    id: 'prophylaxis-drill',
    title: 'Quiet Moves & Prophylaxis Drill',
    subtitle: 'Non-forcing tactical trainer: shut down opponent threats and find calm masterstrokes',
    icon: 'shield',
    category: 'Training',
    action: () => { showProphylaxisModal.value = true }
  },
  {
    id: 'chess960',
    title: 'Chess960 (Fischer Random)',
    subtitle: 'Generate any of the 960 starting positions',
    icon: 'dices',
    category: 'Actions',
    action: () => { showChess960Modal.value = true }
  },
  {
    id: 'play-engine',
    title: 'Play Against Engine',
    subtitle: 'Spar with UCI engine at customized strength and time controls',
    icon: 'swords',
    category: 'Actions',
    action: handlePlayVsEngine
  },
  {
    id: 'simulate-player',
    title: 'Simulate Database Player (Sparring)',
    subtitle: 'Play against an AI modeled on a database grandmaster',
    icon: 'bot',
    category: 'Actions',
    action: () => { showPlayerSimulationModal.value = true }
  },
  {
    id: 'player-personas',
    title: 'Player Personas & Aliases',
    subtitle: 'Manage aggregate styles, biographical info, and FIDE ratings',
    icon: 'users',
    category: 'Training',
    action: () => { showPersonaModal.value = true }
  },
  {
    id: 'engine-match',
    title: 'Engine vs Engine Match',
    subtitle: 'Automated tournament duel between UCI engines',
    icon: 'swords',
    category: 'Actions',
    action: () => { showEngineMatchModal.value = true }
  },
  {
    id: 'tournament-crosstable',
    title: 'Tournament Crosstables & Standings',
    subtitle: 'Calculate berger tables, tiebreaks, and round-by-round results',
    icon: 'trophy',
    category: 'Actions',
    shortcut: 'F7',
    action: () => { showTournamentCrosstableModal.value = true }
  },
  {
    id: 'eco-browser',
    title: 'ECO Opening Repertoire Browser',
    subtitle: 'Explore full encyclopedia of chess openings A00-E99',
    icon: 'book-open',
    category: 'Openings',
    shortcut: 'F6',
    action: () => { showEcoBrowserModal.value = true }
  },
  {
    id: 'database-stats',
    title: 'Database Demographics & Rating Tracker',
    subtitle: 'Visual rating history and performance metrics',
    icon: 'bar-chart-3',
    category: 'Actions',
    shortcut: 'Ctrl+D',
    action: () => { showDatabaseStatsModal.value = true }
  },
  {
    id: 'twic-downloader',
    title: 'Download TWIC Bulletins',
    subtitle: 'Download and import weekly grandmaster tournaments',
    icon: 'download',
    category: 'Actions',
    action: () => { showTwicModal.value = true }
  },
  {
    id: 'export-game',
    title: 'Export Game (PGN / HTML / LaTeX / Cards)',
    subtitle: 'Publish publication-grade diagrams and social cards',
    icon: 'download',
    category: 'Actions',
    shortcut: 'Ctrl+Shift+E',
    action: () => { showExportGameModal.value = true }
  },
  {
    id: 'toggle-threat',
    title: threatMode.value ? 'Disable Threat Heatmap' : 'Enable Threat Heatmap',
    subtitle: 'Highlight attacked and tactical danger squares on board',
    icon: 'eye',
    category: 'Options',
    action: toggleThreatMode
  },
  {
    id: 'toggle-blindfold',
    title: blindfoldMode.value ? 'Disable Blindfold Mode' : 'Enable Blindfold Mode',
    subtitle: 'Hide piece graphics to train visualization',
    icon: 'eye-off',
    category: 'Options',
    action: toggleBlindfold
  },
  {
    id: 'toggle-eval-bar',
    title: showEvalBar.value ? 'Hide Evaluation Bar' : 'Show Evaluation Bar',
    subtitle: 'Toggle live win-probability eval gauge beside board',
    icon: 'bar-chart-3',
    category: 'Options',
    action: toggleEvalBar
  },
  {
    id: 'new-workspace-tab',
    title: 'New Workspace Tab',
    subtitle: 'Open an independent chess game and analysis tab',
    icon: 'plus',
    category: 'Actions',
    action: openNewGameTab
  },
  {
    id: 'support-newscid',
    title: 'Support NewScid ($5+)',
    subtitle: 'Voluntary author support via Stripe & disable startup reminder',
    icon: 'heart',
    category: 'Actions',
    action: () => { showSupportModal.value = true }
  },
  {
    id: 'mega-database-downloader',
    title: 'Download Master & Mega Databases (Lumbra / Lichess)',
    subtitle: "One-click download & zstd streaming for Lumbra's GigaBase, Lichess Open, Broadcasts & Elite",
    icon: 'download',
    category: 'Actions',
    action: () => { showMegaDatabaseModal.value = true }
  },
  {
    id: 'fairplay-timecontrol-inspector',
    title: 'Fair-Play & Time-Control Consistency Inspector',
    subtitle: 'Longitudinal multi-time-control (Bullet/Blitz/Rapid/Classical) & everyday anomaly dossier',
    icon: 'shield',
    category: 'Tools',
    action: () => { showFairPlayModal.value = true }
  },
  {
    id: 'preferences-document',
    title: 'Preferences & Options Document',
    subtitle: 'Configure Default Database, Panels/Floaters, Hyprland Multi-Screen, Portals & Appearance',
    icon: 'sliders',
    category: 'Options',
    shortcut: 'Ctrl+,',
    action: () => openPreferencesModal('databases')
  },
  {
    id: 'database-hub-roles',
    title: 'Database Hub & Relationships (Default / Active / Reference / Clipbase)',
    subtitle: 'Understand how databases relate, set Default Startup DB, or assign Reference Theory DB',
    icon: 'database',
    category: 'Actions',
    action: () => openDatabaseHubView()
  },
  {
    id: 'set-default-db',
    title: 'Set Current Database as Default Startup Database',
    subtitle: 'Automatically open this database whenever NewScid starts',
    icon: 'star',
    category: 'Actions',
    action: () => handleSetCurrentAsDefaultDB()
  },
  {
    id: 'live-portal-lichess-chesscom',
    title: 'Lichess.org & Chess.com Live Portal',
    subtitle: 'Open in-app Live TV/Puzzle portal or launch dedicated multi-screen Hyprland windows',
    icon: 'globe',
    category: 'Tools',
    action: () => openPortalView()
  },
  {
    id: 'float-panels-preset',
    title: 'Float Key Panels (Floaters Preset)',
    subtitle: 'Undock Opening Tree and Live Engine into draggable floating windows',
    icon: 'layers',
    category: 'Options',
    action: () => applyLayoutPreset('floaters')
  },
  {
    id: 'reset-docked-panels',
    title: 'Reset All Panels to Docked Layout',
    subtitle: 'Restore standard docked workspace layout',
    icon: 'layout',
    category: 'Options',
    action: () => resetWorkspaceLayout()
  }
])
</script>

<template>
  <div class="app-root flex flex-col h-screen w-screen bg-zinc-950 text-zinc-100 overflow-hidden font-sans">
    <!-- ========================================================================= -->
    <!-- MODE A: Detached Multi-Screen Hyprland Panel Window (--window-panel <id>) -->
    <!-- ========================================================================= -->
    <div
      v-if="windowStartupMode.mode === 'panel'"
      class="flex-1 flex flex-col h-full w-full overflow-hidden bg-zinc-950"
    >
      <!-- Multi-Screen Sync Status Header -->
      <div class="flex items-center justify-between px-3 py-2 bg-zinc-900 border-b border-zinc-800 text-xs shrink-0">
        <div class="flex items-center gap-2 min-w-0">
          <span class="px-2 py-0.5 rounded bg-purple-600/20 border border-purple-500/40 text-purple-300 font-bold text-[10px] uppercase tracking-wider shrink-0">
            Hyprland Screen Panel
          </span>
          <span class="font-bold text-zinc-100 truncate">
            {{ windowStartupMode.portalTitle || `NewScid — ${windowStartupMode.panelId}` }}
          </span>
          <span v-if="dbInfo?.name" class="text-zinc-400 font-mono text-[11px] shrink-0">
            • DB: {{ dbInfo.name }}
          </span>
        </div>
        <div class="flex items-center gap-2 shrink-0">
          <label class="flex items-center gap-1 text-[11px] text-zinc-400">
            <span>Workspace:</span>
            <select
              :value="windowStartupMode.workspace || ''"
              class="bg-zinc-800 border border-zinc-700 rounded px-1.5 py-0.5 text-[11px] text-purple-300 font-mono cursor-pointer"
              title="Relocate this detached panel window to a specific Hyprland workspace (1..10)"
              @change="handleMoveCurrentWindowToWorkspace(($event.target as HTMLSelectElement).value)"
            >
              <option value="">Current</option>
              <option v-for="ws in 10" :key="ws" :value="String(ws)">WS {{ ws }}</option>
            </select>
          </label>
          <span class="text-[10px] text-emerald-400 font-mono flex items-center gap-1">
            <span class="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse" />
            Live Sync
          </span>
          <button
            class="px-2 py-1 rounded bg-zinc-800 hover:bg-zinc-700 text-zinc-300 text-[11px] cursor-pointer"
            @click="openPreferencesModal('multiscreen')"
          >
            Preferences
          </button>
        </div>
      </div>

      <!-- Full-Window Rendered Panel Content -->
      <div class="flex-1 overflow-hidden p-2">
        <div
          v-if="windowStartupMode.panelId === 'board'"
          class="h-full w-full flex flex-col items-center justify-between gap-2 overflow-hidden"
        >
          <div class="w-full flex items-center justify-between px-3 py-1.5 bg-zinc-900/90 border border-zinc-800 rounded-lg text-xs shrink-0">
            <div class="flex items-center gap-2 truncate">
              <span class="font-bold text-zinc-100">{{ gameTags.White || 'White' }}</span>
              <span class="text-zinc-500">vs</span>
              <span class="font-bold text-zinc-100">{{ gameTags.Black || 'Black' }}</span>
              <span class="text-emerald-400 font-mono ml-1">{{ gameTags.Result || '*' }}</span>
            </div>
            <div class="flex items-center gap-1 shrink-0">
              <button class="px-2 py-0.5 rounded bg-zinc-800 hover:bg-zinc-700 text-zinc-200 cursor-pointer" title="First Move" @click="goFirst">|◀</button>
              <button class="px-2 py-0.5 rounded bg-zinc-800 hover:bg-zinc-700 text-zinc-200 cursor-pointer" title="Previous Move" @click="goPrev">◀</button>
              <button class="px-2 py-0.5 rounded bg-zinc-800 hover:bg-zinc-700 text-zinc-200 cursor-pointer" title="Next Move" @click="goNext">▶</button>
              <button class="px-2 py-0.5 rounded bg-zinc-800 hover:bg-zinc-700 text-zinc-200 cursor-pointer" title="Last Move" @click="goLast">▶|</button>
              <button class="px-2 py-0.5 rounded bg-zinc-800 hover:bg-zinc-700 text-amber-300 cursor-pointer ml-1" title="Flip Board (F)" @click="flipped = !flipped">Flip</button>
              <button class="px-2 py-0.5 rounded bg-zinc-800 hover:bg-zinc-700 text-sky-300 cursor-pointer" title="Toggle Eval Bar" @click="toggleEvalBar">Eval</button>
            </div>
          </div>
          <div class="flex-1 flex items-center justify-center w-full overflow-hidden">
            <Chessboard
              :fen="currentFEN"
              :legal-moves="legalMoves"
              :piece-set="currentPieceSet"
              :board-theme="currentTheme"
              :flipped="flipped"
              :last-move="lastMove"
              :in-check="inCheck"
              :sound-enabled="soundEnabled"
              :blindfold="blindfoldMode"
              :show-eval-bar="showEvalBar"
              :eval-score="evalScore"
              :eval-mate="evalMate"
              :threat-squares="threatSquares"
              :arrows="engineArrows"
              :board-size="computedBoardSize"
              @move="handleMove"
            />
          </div>
        </div>

        <OpeningTree
          v-else-if="windowStartupMode.panelId === 'tree'"
          :fen="currentFEN"
          :tree-data="treeData"
          :loading="treeLoading"
          :db-open="!!dbInfo || !!dbRolesStatus?.referenceDbPath"
          :db-name="dbRolesStatus?.useRefForTree && dbRolesStatus?.referenceDbName ? `${dbRolesStatus.referenceDbName} (Ref)` : dbInfo?.name"
          :eco-code="gameTags.ECO"
          :opening-name="gameTags.Opening || gameTags.Event"
          :discovered-databases="discoveredDatabases"
          @play-move="handleMove"
          @refresh="fetchOpeningTree(currentFEN)"
          @open-db="handleOpenDB"
          @open-discovered-db="handleOpenDiscoveredBase"
          @go-first="goFirst"
          @go-prev="goPrev"
          @pop-out-window="handleOpenTreeWindow"
        />

        <div v-else-if="windowStartupMode.panelId === 'engine'" class="h-full flex flex-col gap-3">
          <EnginePanel
            :eval-info="evalInfo"
            :is-analyzing="isAnalyzing"
            :available-engines="availableEngines"
            :discovered-engines="discoveredEngines"
            :selected-engine="selectedEngine"
            :multi-pv="engineMultiPv"
            :tablebase-result="tablebaseResult"
            @toggle-analysis="toggleAnalysis"
            @select-engine="handleSelectEngine"
            @change-multi-p-v="handleChangeMultiPV"
            @change-threads="handleChangeThreads"
            @change-hash="handleChangeHash"
            @preview-move="handlePreviewCandidateMove"
          />
          <ImbalanceBar :report="imbalanceReport" />
        </div>

        <div v-else-if="windowStartupMode.panelId === 'imbalance'" class="h-full flex flex-col gap-3 p-2">
          <ImbalanceBar :report="imbalanceReport" />
        </div>

        <div v-else-if="windowStartupMode.panelId === 'info'" class="h-full overflow-y-auto p-4 space-y-4">
          <div class="grid grid-cols-2 gap-3 text-xs font-mono">
            <div class="p-3 bg-zinc-900 rounded-lg border border-zinc-800">
              <span class="text-zinc-500 uppercase font-semibold">White Player</span>
              <p class="text-base font-bold text-zinc-100 mt-1">{{ gameTags.White || '?' }}</p>
              <span v-if="gameTags.WhiteElo" class="text-xs text-emerald-400">Elo: {{ gameTags.WhiteElo }}</span>
            </div>
            <div class="p-3 bg-zinc-900 rounded-lg border border-zinc-800">
              <span class="text-zinc-500 uppercase font-semibold">Black Player</span>
              <p class="text-base font-bold text-zinc-100 mt-1">{{ gameTags.Black || '?' }}</p>
              <span v-if="gameTags.BlackElo" class="text-xs text-emerald-400">Elo: {{ gameTags.BlackElo }}</span>
            </div>
          </div>
          <div class="p-3 bg-zinc-900 rounded-lg border border-zinc-800 space-y-2 text-xs">
            <div class="flex justify-between"><span class="text-zinc-500 font-semibold">Event:</span><span class="text-zinc-200">{{ gameTags.Event || '?' }}</span></div>
            <div class="flex justify-between"><span class="text-zinc-500 font-semibold">Site:</span><span class="text-zinc-200">{{ gameTags.Site || '?' }}</span></div>
            <div class="flex justify-between"><span class="text-zinc-500 font-semibold">Date:</span><span class="text-zinc-200">{{ gameTags.Date || '????.??.??' }}</span></div>
            <div class="flex justify-between"><span class="text-zinc-500 font-semibold">Result:</span><span class="text-emerald-400 font-bold">{{ gameTags.Result || '*' }}</span></div>
            <div class="flex justify-between"><span class="text-zinc-500 font-semibold">ECO:</span><span class="text-zinc-200 font-mono">{{ gameTags.ECO || 'A00' }}</span></div>
          </div>
        </div>

        <div v-else-if="windowStartupMode.panelId === 'notation'" class="h-full flex flex-col gap-2">
          <EvaluationGraph
            :analysis="gameAnalysis"
            :current-ply="currentPly"
            :is-analyzing-game="isAnalyzingGame"
            :progress="annotationProgress"
            :moves="flatMoves"
            @jump="jumpToPly"
            @annotate="handleAnnotateGame"
          />
          <div class="flex-1 overflow-hidden">
            <MoveNotation
              :moves="flatMoves"
              :current-ply="currentPly"
              :is-playing="isPlaying"
              :result="gameTags.Result || '*'"
              @jump="jumpToPly"
              @first="goFirst"
              @prev="goPrev"
              @next="goNext"
              @last="goLast"
              @toggle-play="toggleAutoplay"
              @copy-p-g-n="handleExportPGN"
              @set-n-a-g="handleSetNAG"
              @clear-n-a-gs="handleClearNAGs"
              @set-comment="handleSetComment"
              @promote-variation="handlePromoteVariation"
              @delete-variation="handleDeleteVariation"
              @explain-move="handleExplainMove"
            />
          </div>
        </div>

        <GameBrowser
          v-else-if="windowStartupMode.panelId === 'games'"
          :games="gamesList"
          :total-games="totalGames"
          :current-page="currentPage"
          :page-size="pageSize"
          :active-game-id="activeGameId"
          :db-name="dbInfo?.name"
          :active-criteria="searchFilter"
          @load-game="handleLoadGame"
          @page-change="handlePageChange"
          @search="handleSearch"
          @open-advanced-search="showSearchModal = true"
          @new-game="handleNewGame"
          @save-game="handleSaveCurrentGame"
          @delete-game="handleDeleteGame"
        />

        <LivePortalPanel
          v-else-if="windowStartupMode.panelId === 'portal'"
          :current-fen="currentFEN"
          :default-window-mode="appPreferences.portalWindowMode"
          :lichess-username="appPreferences.lichessUsername"
          :chess-com-username="appPreferences.chessComUsername"
          @open-online-sync="showOnlineSyncModal = true"
          @open-fair-play="showFairPlayModal = true"
          @detach-portal-screen="handleDetachPanelScreen('portal')"
          @float-portal="togglePanelFloater('portal')"
        />

        <DatabaseHubPanel
          v-else-if="windowStartupMode.panelId === 'db-roles'"
          :roles-status="dbRolesStatus"
          :discovered-databases="discoveredDatabases"
          :is-updating-from-local="isUpdatingDbFromLocal"
          :local-merge-status="localMergeStatusMsg"
          @open-d-b="handleOpenDB"
          @create-d-b="handleCreateDB"
          @open-discovered-base="handleOpenDiscoveredBase"
          @set-default-d-b="handleSetDefaultDatabase"
          @set-reference-d-b="handleSetReferenceDatabase"
          @open-default-d-b="handleOpenDefaultDatabase"
          @open-clipbase="handleOpenClipbase"
          @open-mega-downloader="showMegaDatabaseModal = true"
          @open-online-sync="showOnlineSyncModal = true"
          @open-maintenance="showMaintenanceModal = true"
          @update-from-local="handleUpdateDatabaseFromLocal"
        />
      </div>
    </div>

    <!-- ========================================================================= -->
    <!-- MODE B: Main Multi-Panel Workspace (Docked + Floating + Multi-Screen)     -->
    <!-- ========================================================================= -->
    <template v-else>
      <!-- Desktop Application Menu Bar (Reorganized & Scid vs. PC Parity) -->
      <MenuBar
        :db-name="dbInfo?.name"
        :default-db-name="dbRolesStatus?.defaultDbName"
        :is-active-default-db="dbRolesStatus?.isActiveDefault"
        :reference-db-name="dbRolesStatus?.referenceDbName"
        :piece-sets="pieceSets"
        :board-themes="boardThemes"
        :current-piece-set="currentPieceSet"
        :current-theme-id="currentThemeId"
        :trial-mode="trialMode"
        :is-analyzing="isAnalyzing"
        @new-game="handleNewGame"
        @open-d-b="handleOpenDB"
        @create-d-b="handleCreateDB"
        @close-d-b="handleCloseDB"
        @save-game="handleSaveCurrentGame"
        @delete-game="handleDeleteCurrentGame"
        @first-game="handleFirstGame"
        @prev-game="handlePrevGame"
        @next-game="handleNextGame"
        @last-game="handleLastGame"
        @random-game="handleRandomGame"
        @import-p-g-n="handleImportPGN"
        @export-p-g-n="handleExportPGN"
        @copy-f-e-n="handleCopyFEN"
        @paste-f-e-n="handlePasteFEN"
        @setup-board="handleSetupBoard"
        @toggle-trial-mode="toggleTrialMode"
        @undo-move="goPrev"
        @redo-move="goNext"
        @toggle-analysis="toggleAnalysis"
        @annotate-game="handleAnnotateGame"
        @play-vs-engine="handlePlayVsEngine"
        @tactics-trainer="handleTacticsTrainer"
        @opening-tree="handleOpenTreeWindow"
        @open-tree-window="handleOpenTreeWindow"
        @opening-trainer="showOpeningTrainerModal = true"
        @progressive-tactics="showProgressiveTacticsModal = true"
        @mine-puzzles="showPuzzleMinerModal = true"
        @search-header="showSearchModal = true"
        @search-position="handleSearchPosition"
        @toggle-flip="flipped = !flipped"
        @change-piece-set="currentPieceSet = $event"
        @change-theme="currentThemeId = $event"
        @open-settings="handleOpenSettings"
        @open-preferences="openPreferencesModal"
        @open-database-hub="openDatabaseHubView"
        @set-current-as-default-d-b="handleSetCurrentAsDefaultDB"
        @open-default-d-b="handleOpenDefaultDatabase"
        @open-clipbase="handleOpenClipbase"
        @open-portal-panel="openPortalView"
        @launch-portal-window="handleLaunchPortalWindow"
        @detach-panel-screen="handleDetachPanelScreen"
        @show-shortcuts="showShortcutsModal = true"
        @show-about="showAboutModal = true"
        @show-help="showHelpModal = true"
        @open-web-docs="handleOpenWebDocs"
        @database-maintenance="showMaintenanceModal = true"
        @online-sync="showOnlineSyncModal = true"
        @simulate-player="showPlayerSimulationModal = true"
        @player-personas="showPersonaModal = true"
        @show-crosstable="showTournamentCrosstableModal = true"
        @eco-browser="showEcoBrowserModal = true"
        @guess-the-move="showGuessTheMoveModal = true"
        @database-stats="showDatabaseStatsModal = true"
        @edit-tags="handleOpenEditTags"
        @twic-downloader="showTwicModal = true"
        @mega-downloader="showMegaDatabaseModal = true"
        @engine-match="showEngineMatchModal = true"
        @export-game="showExportGameModal = true"
        @standardize-names="handleStandardizeNames"
        :sound-enabled="soundEnabled"
        :blindfold="blindfoldMode"
        :threat-mode="threatMode"
        :show-eval-bar="showEvalBar"
        :zen-mode="zenMode"
        :compact-layout="effectiveCompact"
        @toggle-sound="toggleSound"
        @toggle-blindfold="toggleBlindfold"
        @toggle-threat-mode="toggleThreatMode"
        @toggle-eval-bar="toggleEvalBar"
        @chess960="showChess960Modal = true"
        @coordinate-trainer="showCoordinateTrainerModal = true"
        @command-palette="showCommandPalette = true"
        @blunder-vault="showBlunderVaultModal = true"
        @opponent-scout="showOpponentScoutModal = true"
        @anti-persona="openAntiPersonaModal()"
        @toggle-zen-mode="toggleZenMode"
        @toggle-compact-layout="toggleCompactLayout"
        @explain-move="handleExplainMove(currentPly)"
        :current-u-i-theme="currentUITheme"
        @change-u-i-theme="applyUITheme"
        @toggle-dark-light="toggleDarkLight"
        @piece-safety="showPieceSafetyModal = true"
        @intuition-storm="showIntuitionStormModal = true"
        @time-management="showTimeManagementModal = true"
        @repertoire-shortener="showRepertoireShortenerModal = true"
        @prophylaxis="showProphylaxisModal = true"
        @show-support="showSupportModal = true"
        @fair-play-inspector="showFairPlayModal = true"
      />

      <!-- Top Header Bar (Auto-hidden in Compact / Small-Screen Mode to maximize board height) -->
      <HeaderBar
        v-if="!zenMode && !effectiveCompact"
        ref="headerBarRef"
        :db-name="dbInfo?.name"
        :default-db-name="dbRolesStatus?.defaultDbName"
        :is-active-default-db="dbRolesStatus?.isActiveDefault"
        :reference-db-name="dbRolesStatus?.referenceDbName"
        :system-bases="systemBases"
        :discovered-databases="discoveredDatabases"
        :piece-sets="pieceSets"
        :board-themes="boardThemes"
        :current-piece-set="currentPieceSet"
        :current-theme-id="currentThemeId"
        :current-u-i-theme="currentUITheme"
        :flipped="flipped"
        @open-d-b="handleOpenDB"
        @create-d-b="handleCreateDB"
        @open-system-base="handleOpenSystemBase"
        @open-discovered-base="handleOpenDiscoveredBase"
        @import-p-g-n="handleImportPGN"
        @export-p-g-n="handleExportPGN"
        @change-piece-set="currentPieceSet = $event"
        @change-theme="currentThemeId = $event"
        @change-u-i-theme="applyUITheme"
        @toggle-dark-light="toggleDarkLight"
        @toggle-flip="flipped = !flipped"
        @set-current-as-default-d-b="handleSetCurrentAsDefaultDB"
        @open-database-hub="openDatabaseHubView"
        @open-portal-panel="openPortalView"
        @launch-portal-window="handleLaunchPortalWindow"
        @open-preferences="openPreferencesModal"
      />

      <!-- Multi-Game Tabbed Workspace Strip (Auto-hidden on compact screens when only 1 game tab is open) -->
      <div
        v-if="!zenMode && (!effectiveCompact || openGameTabs.length > 1)"
        class="workspace-tabs-bar flex items-center justify-between gap-1.5 px-3 py-1 bg-zinc-950 border-b border-zinc-800/80 text-xs overflow-x-auto select-none"
      >
        <div class="flex items-center gap-1.5 overflow-x-auto">
          <div
            v-for="(tab, idx) in openGameTabs"
            :key="tab.id"
            class="group flex items-center gap-2 px-3 py-1.5 rounded-t-lg border-t border-x cursor-pointer transition-all max-w-[240px]"
            :class="activeGameTabIdx === idx
              ? 'bg-zinc-900 border-zinc-700 text-zinc-100 font-semibold shadow-xs'
              : 'bg-zinc-950/60 border-transparent text-zinc-400 hover:text-zinc-200 hover:bg-zinc-900/50'"
            @click="switchGameTab(idx)"
          >
            <span class="w-1.5 h-1.5 rounded-full shrink-0" :class="activeGameTabIdx === idx ? 'bg-emerald-400' : 'bg-zinc-600'" />
            <span class="truncate">{{ tab.title }}</span>
            <button
              v-if="openGameTabs.length > 1"
              class="w-4 h-4 ml-1 rounded hover:bg-zinc-800 hover:text-rose-400 flex items-center justify-center text-zinc-500 opacity-60 group-hover:opacity-100 transition-all text-[11px]"
              title="Close tab"
              @click.stop="closeGameTab(idx)"
            >
              ✕
            </button>
          </div>

          <button
            class="px-2.5 py-1 rounded-md hover:bg-zinc-800 text-zinc-400 hover:text-zinc-100 transition-colors flex items-center gap-1 text-xs cursor-pointer ml-1"
            title="Open new game tab"
            @click="openNewGameTab"
          >
            <span class="font-bold text-sm leading-none">+</span>
            <span class="text-[11px]">New Tab</span>
          </button>
        </div>

        <!-- Database Relationship Status Pill in Tab Strip -->
        <div class="hidden lg:flex items-center gap-2 text-[11px] shrink-0">
          <button
            class="px-2 py-0.5 rounded bg-zinc-900 hover:bg-zinc-800 border border-zinc-800 text-zinc-400 hover:text-zinc-200 flex items-center gap-1.5 cursor-pointer transition-colors"
            title="Click to view and configure how Default, Active, Reference, and Clipbase databases relate"
            @click="openDatabaseHubView"
          >
            <span class="text-emerald-400 font-semibold">Active: {{ dbInfo?.name || 'None' }}</span>
            <span v-if="dbRolesStatus?.isActiveDefault" class="text-amber-400 font-bold" title="Default Startup DB">★ Default</span>
            <span v-if="dbRolesStatus?.useRefForTree && dbRolesStatus?.referenceDbName" class="text-sky-400 border-l border-zinc-700 pl-1.5">
              Tree Ref: {{ dbRolesStatus.referenceDbName }}
            </span>
          </button>
        </div>
      </div>

      <!-- Main Workspace (Split Grid) -->
      <div
        ref="workspaceRef"
        class="workspace-grid flex-1 flex flex-col md:flex-row overflow-hidden p-1.5 md:p-2 gap-1 relative min-h-0"
      >
        <!-- Left Column: Auto-Sized Chessboard & Optional Docked Left Panels -->
        <div
          ref="leftColRef"
          class="left-col flex flex-col items-center justify-start pt-1 gap-1.5 w-full md:w-auto shrink-0 md:h-full overflow-hidden"
          :style="{ width: isMdScreen ? `${leftColWidth}px` : '100%' }"
        >
          <!-- Player Simulation Sparring Match Status -->
          <div
            v-if="activeSimulation"
            class="w-full bg-gradient-to-r from-purple-950/80 via-zinc-900 to-purple-950/80 border border-purple-500/40 rounded-xl px-2.5 py-1.5 flex items-center justify-between text-xs shadow-md animate-fade-in"
          >
            <div class="flex items-center gap-2 min-w-0">
              <div class="w-6 h-6 rounded-lg bg-purple-500/20 border border-purple-500/40 flex items-center justify-center text-purple-300 shrink-0">
                <Bot class="w-3.5 h-3.5" :class="{ 'animate-pulse text-purple-400': isBotThinking }" />
              </div>
              <div class="truncate">
                <div class="flex items-center gap-1.5">
                  <span class="font-bold text-zinc-100">{{ activeSimulation.playerName }}</span>
                  <span class="px-1.5 py-0.5 rounded text-[10px] bg-purple-900/60 text-purple-200 font-mono">
                    ~{{ activeSimulation.personality.estimatedElo || 2400 }} Elo
                  </span>
                  <span v-if="isBotThinking" class="text-[10px] text-amber-400 animate-pulse">Thinking...</span>
                </div>
                <p class="text-[11px] text-zinc-400 truncate">{{ botMoveStatus || 'Sparring match active' }}</p>
              </div>
            </div>
            <button
              class="px-2 py-1 rounded bg-zinc-800 hover:bg-zinc-700 text-zinc-300 text-[11px] border border-zinc-700 hover:border-zinc-600 transition-colors shrink-0 ml-2 cursor-pointer"
              @click="stopSimulation"
            >
              End Sparring
            </button>
          </div>

          <!-- Theory Novelty Pill -->
          <div
            v-if="isTheoreticalNovelty"
            class="w-full flex items-center justify-between px-3 py-1 bg-amber-500/10 border border-amber-500/30 rounded-xl text-xs text-amber-300 font-semibold shadow-xs"
          >
            <span class="flex items-center gap-1.5">
              <Sparkles class="w-3.5 h-3.5 text-amber-400" />
              <span>Theoretical Novelty (N)</span>
            </span>
            <span class="text-[10px] text-amber-400/80 font-mono">Out of Book • 0 Games</span>
          </div>

          <!-- Blunder Radar Alert Banner -->
          <div
            v-if="blunderRadarAlert"
            class="w-full flex items-center gap-2 px-3 py-1 bg-rose-950/90 border border-rose-500/50 rounded-xl text-xs text-rose-200 font-semibold shadow-md animate-pulse"
          >
            <AlertTriangle class="w-4 h-4 text-rose-400 shrink-0" />
            <span class="truncate">{{ blunderRadarAlert }}</span>
          </div>

          <!-- Interactive Auto-Sized Chessboard (or Detached/Floating Board Card) -->
          <div
            v-if="!workspacePanels.board || workspacePanels.board.mode === 'docked'"
            class="chessboard-wrapper-host relative group/board flex items-center justify-center w-full shrink-0"
          >
            <div class="absolute right-2 top-1.5 z-20 hidden group-hover/board:flex items-center gap-1 bg-zinc-900/95 border border-zinc-700 rounded-lg px-1.5 py-0.5 text-[10px] shadow-lg">
              <button
                class="text-amber-300 hover:text-amber-200 px-1 cursor-pointer font-semibold"
                title="Undock Chessboard into a draggable Floater window"
                @click="togglePanelFloater('board')"
              >
                Float ❐
              </button>
              <button
                class="text-purple-300 hover:text-purple-200 px-1 cursor-pointer font-semibold"
                title="Detach Chessboard into its own Hyprland Screen Window"
                @click="handleDetachPanelScreen('board')"
              >
                Screen ↗
              </button>
              <select
                :value="workspacePanels.board?.targetWorkspace || ''"
                class="bg-zinc-800 border border-zinc-700 rounded px-1 py-0.2 text-[10px] text-purple-200 font-mono cursor-pointer"
                title="Detach & relocate Chessboard to a specific Hyprland Workspace (1..10)"
                @change="handleDetachPanelScreen('board', ($event.target as HTMLSelectElement).value)"
              >
                <option value="">WS ▾</option>
                <option v-for="ws in 10" :key="ws" :value="String(ws)">WS {{ ws }}</option>
              </select>
            </div>
            <Chessboard
              :fen="currentFEN"
              :legal-moves="legalMoves"
              :piece-set="currentPieceSet"
              :board-theme="currentTheme"
              :flipped="flipped"
              :last-move="lastMove"
              :in-check="inCheck"
              :sound-enabled="soundEnabled"
              :blindfold="blindfoldMode"
              :show-eval-bar="showEvalBar"
              :eval-score="evalScore"
              :eval-mate="evalMate"
              :threat-squares="threatSquares"
              :arrows="engineArrows"
              :board-size="computedBoardSize"
              @move="handleMove"
            />
          </div>
          <div
            v-else
            class="chessboard-wrapper-host w-full bg-zinc-900/90 border border-purple-500/30 rounded-xl p-3 flex flex-col items-center justify-center gap-2 text-center shadow-md shrink-0"
          >
            <div class="text-xs font-bold text-purple-300">
              Chessboard {{ workspacePanels.board.mode === 'floater' ? 'Floating' : 'Detached' }}
              <span v-if="workspacePanels.board.targetWorkspace" class="font-mono text-emerald-300">
                (WS {{ workspacePanels.board.targetWorkspace }})
              </span>
            </div>
            <p class="text-[11px] text-zinc-400">
              Moves and games stay synchronized across all windows.
            </p>
            <div class="flex flex-wrap items-center justify-center gap-1.5">
              <button
                class="px-2.5 py-1 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white text-[11px] font-semibold cursor-pointer"
                @click="setPanelMode('board', 'docked')"
              >
                Dock Board Here
              </button>
              <button
                v-if="workspacePanels.board.mode === 'detached'"
                class="px-2 py-1 rounded-lg bg-purple-950/60 hover:bg-purple-900/60 border border-purple-500/40 text-purple-200 text-[11px] cursor-pointer"
                @click="handleDetachPanelScreen('board', workspacePanels.board.targetWorkspace)"
              >
                Re-Launch Window ↗
              </button>
            </div>
          </div>

          <!-- Interactive Material & Positional Imbalance Bar (Auto-hidden under board on compact screens) -->
          <div
            v-if="showLeftImbalancePanel"
            class="w-full relative group/imbalance"
          >
            <div class="absolute right-2 top-1.5 z-10 hidden group-hover/imbalance:flex items-center gap-1 bg-zinc-900/90 border border-zinc-700 rounded px-1 py-0.5 text-[10px]">
              <button
                class="text-amber-300 hover:text-amber-200 px-1 cursor-pointer"
                title="Undock as Floating Window (Floater)"
                @click="togglePanelFloater('imbalance')"
              >
                Float
              </button>
              <button
                class="text-zinc-400 hover:text-rose-300 px-1 cursor-pointer"
                title="Hide Imbalance Panel"
                @click="setPanelMode('imbalance', 'hidden')"
              >
                ✕
              </button>
            </div>
            <ImbalanceBar :report="imbalanceReport" :compact="true" />
          </div>

          <!-- Live Engine Analysis Bar (Auto-moved to Right Column 'Engine' tab on compact screens) -->
          <div
            v-if="showLeftEnginePanel"
            class="w-full relative group/engine"
          >
            <div class="absolute right-2 top-1.5 z-10 hidden group-hover/engine:flex items-center gap-1 bg-zinc-900/90 border border-zinc-700 rounded px-1.5 py-0.5 text-[10px]">
              <button
                class="text-amber-300 hover:text-amber-200 px-1 cursor-pointer"
                title="Undock Engine into a draggable Floater window"
                @click="togglePanelFloater('engine')"
              >
                Float
              </button>
              <button
                class="text-purple-300 hover:text-purple-200 px-1 cursor-pointer"
                title="Detach Engine into separate Hyprland Screen Window"
                @click="handleDetachPanelScreen('engine')"
              >
                Screen ↗
              </button>
              <button
                class="text-zinc-400 hover:text-rose-300 px-1 cursor-pointer"
                title="Hide Engine Panel"
                @click="setPanelMode('engine', 'hidden')"
              >
                ✕
              </button>
            </div>
            <EnginePanel
              :eval-info="evalInfo"
              :is-analyzing="isAnalyzing"
              :available-engines="availableEngines"
              :discovered-engines="discoveredEngines"
              :selected-engine="selectedEngine"
              :multi-pv="engineMultiPv"
              :tablebase-result="tablebaseResult"
              @toggle-analysis="toggleAnalysis"
              @select-engine="handleSelectEngine"
              @change-multi-p-v="handleChangeMultiPV"
              @change-threads="handleChangeThreads"
              @change-hash="handleChangeHash"
              @preview-move="handlePreviewCandidateMove"
            />
          </div>
        </div>

        <!-- Horizontal Drag Splitter (between Left and Right Panes) -->
        <div
          class="hidden md:flex items-center justify-center w-2 cursor-col-resize hover:bg-emerald-500/20 active:bg-emerald-500/40 rounded transition-colors group select-none relative z-10 shrink-0"
          :class="{ 'bg-emerald-500/30': isResizingH }"
          title="Drag to resize panes (Double-click to auto-fit board)"
          @mousedown="startHorizontalResize"
          @dblclick="resetHorizontalResize"
        >
          <div
            class="w-1 h-8 rounded-full bg-zinc-700 group-hover:bg-emerald-400 group-active:bg-emerald-400 transition-colors"
            :class="{ 'bg-emerald-400': isResizingH }"
          />
        </div>

        <!-- Right Column: Customizable Docked Tabbed Panels -->
        <div class="right-col flex-1 min-w-0 min-h-0 flex flex-col overflow-hidden bg-zinc-900 border border-zinc-800 rounded-xl shadow-lg">
          <!-- Tabs Header + Panel Float / Detach / Customize Controls -->
          <div class="tabs-header flex items-center justify-between gap-1.5 p-1.5 bg-zinc-950 border-b border-zinc-800">
            <div class="flex items-center gap-1 overflow-x-auto min-w-0">
              <button
                v-if="workspacePanels.notation.mode === 'docked'"
                class="px-2 py-1 rounded-lg text-xs font-semibold transition-all cursor-pointer whitespace-nowrap"
                :class="activeTab === 'notation'
                  ? 'bg-emerald-600 text-white shadow-sm'
                  : 'text-zinc-400 hover:text-zinc-200 hover:bg-zinc-800'"
                @click="activeTab = 'notation'"
              >
                {{ effectiveCompact ? 'Moves' : 'Notation & Analysis' }}
              </button>

              <button
                v-if="workspacePanels.tree.mode === 'docked'"
                class="px-2 py-1 rounded-lg text-xs font-semibold transition-all cursor-pointer whitespace-nowrap"
                :class="activeTab === 'tree'
                  ? 'bg-emerald-600 text-white shadow-sm'
                  : 'text-zinc-400 hover:text-zinc-200 hover:bg-zinc-800'"
                @click="activeTab = 'tree'"
              >
                {{ effectiveCompact ? 'Tree' : 'Opening Tree' }}
              </button>

              <!-- Games Tab (automatically shown when bottom drawer is collapsed or on compact screens) -->
              <button
                v-if="!showBottomGamesDrawer || workspacePanels.games.dockZone === 'right' || activeTab === 'games'"
                class="px-2 py-1 rounded-lg text-xs font-semibold transition-all cursor-pointer whitespace-nowrap flex items-center gap-1"
                :class="activeTab === 'games'
                  ? 'bg-emerald-600 text-white shadow-sm'
                  : 'text-zinc-400 hover:text-zinc-200 hover:bg-zinc-800'"
                @click="activeTab = 'games'"
              >
                <span>Games</span>
                <span v-if="totalGames > 0" class="text-[10px] font-mono opacity-80">({{ totalGames }})</span>
              </button>

              <!-- Engine Tab (automatically shown on compact screens when Engine is moved out from under the board) -->
              <button
                v-if="!showLeftEnginePanel || workspacePanels.engine.dockZone === 'right' || activeTab === 'engine'"
                class="px-2 py-1 rounded-lg text-xs font-semibold transition-all cursor-pointer whitespace-nowrap flex items-center gap-1"
                :class="activeTab === 'engine'
                  ? 'bg-emerald-600 text-white shadow-sm'
                  : 'text-zinc-400 hover:text-zinc-200 hover:bg-zinc-800'"
                @click="activeTab = 'engine'"
              >
                <span>Engine</span>
                <span
                  v-if="isAnalyzing"
                  class="px-1 py-0.2 rounded text-[10px] font-mono font-bold bg-emerald-500/20 text-emerald-300"
                >
                  {{ evalScore >= 0 ? `+${(evalScore / 100).toFixed(1)}` : (evalScore / 100).toFixed(1) }}
                </span>
              </button>

              <!-- Secondary Tabs: Shown inline on large screens, or when active, or via 'More ▾' on compact screens -->
              <button
                v-if="(!effectiveCompact && workspacePanels['db-roles'].mode === 'docked') || activeTab === 'db-roles'"
                class="px-2 py-1 rounded-lg text-xs font-semibold transition-all cursor-pointer whitespace-nowrap flex items-center gap-1"
                :class="activeTab === 'db-roles'
                  ? 'bg-emerald-600 text-white shadow-sm'
                  : 'text-zinc-400 hover:text-zinc-200 hover:bg-zinc-800'"
                @click="activeTab = 'db-roles'"
              >
                <span>{{ effectiveCompact ? 'DBs' : 'Databases & Roles' }}</span>
                <span v-if="dbRolesStatus?.isActiveDefault" class="text-amber-300 text-[10px]">★</span>
              </button>

              <button
                v-if="(!effectiveCompact && workspacePanels.portal.mode === 'docked') || activeTab === 'portal'"
                class="px-2 py-1 rounded-lg text-xs font-semibold transition-all cursor-pointer whitespace-nowrap flex items-center gap-1"
                :class="activeTab === 'portal'
                  ? 'bg-sky-600 text-white shadow-sm'
                  : 'text-sky-400/90 hover:text-sky-200 hover:bg-zinc-800'"
                @click="activeTab = 'portal'"
              >
                <span>{{ effectiveCompact ? 'Portal' : 'Lichess / Chess.com' }}</span>
              </button>

              <button
                v-if="(!effectiveCompact && workspacePanels.info.mode === 'docked') || activeTab === 'info'"
                class="px-2 py-1 rounded-lg text-xs font-semibold transition-all cursor-pointer whitespace-nowrap"
                :class="activeTab === 'info'
                  ? 'bg-emerald-600 text-white shadow-sm'
                  : 'text-zinc-400 hover:text-zinc-200 hover:bg-zinc-800'"
                @click="activeTab = 'info'"
              >
                {{ effectiveCompact ? 'Info' : 'Game Info' }}
              </button>

              <!-- Compact More Panels Popover Button -->
              <div v-if="effectiveCompact" class="relative shrink-0">
                <button
                  class="px-2 py-1 rounded-lg text-[11px] font-semibold bg-zinc-900 hover:bg-zinc-800 text-zinc-400 hover:text-zinc-200 border border-zinc-800 cursor-pointer whitespace-nowrap"
                  title="More workspace panels (Databases, Lichess/Chess.com Portal, Game Info)"
                  @click="showMoreTabsDropdown = !showMoreTabsDropdown"
                >
                  More ▾
                </button>
                <div
                  v-if="showMoreTabsDropdown"
                  class="fixed inset-0 z-40"
                  @click="showMoreTabsDropdown = false"
                />
                <div
                  v-if="showMoreTabsDropdown"
                  class="absolute top-full left-0 mt-1 w-48 bg-zinc-900 border border-zinc-700 rounded-xl shadow-2xl py-1 z-50 text-xs"
                >
                  <button
                    class="w-full text-left px-3 py-1.5 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between cursor-pointer"
                    @click="activeTab = 'db-roles'; showMoreTabsDropdown = false"
                  >
                    <span>Databases &amp; Roles</span>
                    <span v-if="dbRolesStatus?.isActiveDefault" class="text-amber-300 text-[10px]">★</span>
                  </button>
                  <button
                    class="w-full text-left px-3 py-1.5 hover:bg-zinc-800 text-sky-300 flex items-center justify-between cursor-pointer"
                    @click="activeTab = 'portal'; showMoreTabsDropdown = false"
                  >
                    <span>Lichess / Chess.com</span>
                  </button>
                  <button
                    class="w-full text-left px-3 py-1.5 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between cursor-pointer"
                    @click="activeTab = 'info'; showMoreTabsDropdown = false"
                  >
                    <span>Game Info &amp; Tags</span>
                  </button>
                  <div class="h-px bg-zinc-800 my-1" />
                  <button
                    class="w-full text-left px-3 py-1.5 hover:bg-zinc-800 text-zinc-300 flex items-center justify-between cursor-pointer"
                    @click="openNewGameTab(); showMoreTabsDropdown = false"
                  >
                    <span>+ New Game Tab</span>
                  </button>
                  <button
                    class="w-full text-left px-3 py-1.5 hover:bg-zinc-800 text-zinc-400 flex items-center justify-between cursor-pointer"
                    @click="toggleBottomGamesDrawer(); showMoreTabsDropdown = false"
                  >
                    <span>{{ showBottomGamesDrawer ? 'Hide Bottom DB Drawer' : 'Show Bottom DB Drawer' }}</span>
                  </button>
                </div>
              </div>
            </div>

            <!-- Active Tab Panel Controls (Float / Detach to Hyprland Screen / Workspace / Hide / Bottom Drawer Toggle) -->
            <div class="flex items-center gap-1 shrink-0">
              <button
                class="text-[11px] text-amber-300 hover:text-amber-200 px-1.5 py-1 rounded bg-amber-950/40 hover:bg-amber-900/50 border border-amber-500/30 transition-colors cursor-pointer"
                :title="`Undock '${activeTab}' into a draggable Floating Window (Floater)`"
                @click="togglePanelFloater(activeTab as PanelId)"
              >
                <span class="hidden sm:inline">Float </span>❐
              </button>
              <button
                class="text-[11px] text-purple-300 hover:text-purple-200 px-1.5 py-1 rounded bg-purple-950/40 hover:bg-purple-900/50 border border-purple-500/30 transition-colors cursor-pointer"
                :title="`Detach '${activeTab}' into a native OS window for another Hyprland monitor`"
                @click="handleDetachPanelScreen(activeTab as PanelId)"
              >
                <span class="hidden sm:inline">Screen </span>↗
              </button>
              <select
                :value="workspacePanels[activeTab as PanelId]?.targetWorkspace || ''"
                class="bg-zinc-900 border border-purple-500/30 rounded px-1 py-1 text-[10px] text-purple-300 font-mono cursor-pointer"
                :title="`Detach & relocate '${activeTab}' to a specific Hyprland Workspace (1..10)`"
                @change="handleDetachPanelScreen(activeTab as PanelId, ($event.target as HTMLSelectElement).value)"
              >
                <option value="">WS ▾</option>
                <option v-for="ws in 10" :key="ws" :value="String(ws)">WS {{ ws }}</option>
              </select>
              <button
                v-if="!effectiveCompact"
                class="text-[11px] text-zinc-400 hover:text-rose-300 px-1.5 py-1 rounded bg-zinc-800/80 hover:bg-zinc-800 transition-colors cursor-pointer"
                :title="`Hide '${activeTab}' panel (restore anytime from Windows menu)`"
                @click="setPanelMode(activeTab as PanelId, 'hidden')"
              >
                ✕
              </button>
              <button
                v-if="!effectiveCompact"
                class="text-xs text-zinc-400 hover:text-zinc-200 px-2 py-1 rounded bg-zinc-800/80 hover:bg-zinc-800 transition-colors cursor-pointer ml-0.5"
                @click="toggleBottomGamesDrawer()"
              >
                {{ showBottomGamesDrawer ? 'Hide DB ▼' : 'Show DB ▲' }}
              </button>
            </div>
          </div>

          <!-- Tab Content -->
          <div class="tab-content flex-1 overflow-hidden p-2 min-h-0">
            <!-- 1. Notation & Evaluation Curve View -->
            <div v-show="activeTab === 'notation'" class="h-full flex flex-col gap-2">
              <EvaluationGraph
                :analysis="gameAnalysis"
                :current-ply="currentPly"
                :is-analyzing-game="isAnalyzingGame"
                :progress="annotationProgress"
                :moves="flatMoves"
                @jump="jumpToPly"
                @annotate="handleAnnotateGame"
              />

              <div class="flex-1 overflow-hidden">
                <MoveNotation
                  :moves="flatMoves"
                  :current-ply="currentPly"
                  :is-playing="isPlaying"
                  :result="gameTags.Result || '*'"
                  @jump="jumpToPly"
                  @first="goFirst"
                  @prev="goPrev"
                  @next="goNext"
                  @last="goLast"
                  @toggle-play="toggleAutoplay"
                  @copy-p-g-n="handleExportPGN"
                  @set-n-a-g="handleSetNAG"
                  @clear-n-a-gs="handleClearNAGs"
                  @set-comment="handleSetComment"
                  @promote-variation="handlePromoteVariation"
                  @delete-variation="handleDeleteVariation"
                  @explain-move="handleExplainMove"
                />
              </div>
            </div>

            <!-- 2. Opening Tree View -->
            <div v-show="activeTab === 'tree'" class="h-full">
              <OpeningTree
                :fen="currentFEN"
                :tree-data="treeData"
                :loading="treeLoading"
                :db-open="!!dbInfo || !!dbRolesStatus?.referenceDbPath"
                :db-name="dbRolesStatus?.useRefForTree && dbRolesStatus?.referenceDbName ? `${dbRolesStatus.referenceDbName} (Reference)` : dbInfo?.name"
                :eco-code="gameTags.ECO"
                :opening-name="gameTags.Opening || gameTags.Event"
                :discovered-databases="discoveredDatabases"
                @play-move="handleMove"
                @refresh="fetchOpeningTree(currentFEN)"
                @open-db="handleOpenDB"
                @open-discovered-db="handleOpenDiscoveredBase"
                @go-first="goFirst"
                @go-prev="goPrev"
                @pop-out-window="handleOpenTreeWindow"
              />
            </div>

            <!-- 2b. Database Games Browser Tab (Used in Compact / Small-Screen Mode when bottom drawer is collapsed) -->
            <div v-show="activeTab === 'games'" class="h-full overflow-hidden">
              <GameBrowser
                :games="gamesList"
                :total-games="totalGames"
                :current-page="currentPage"
                :page-size="pageSize"
                :active-game-id="activeGameId"
                :db-name="dbInfo?.name"
                :active-criteria="searchFilter"
                @load-game="handleLoadGame"
                @page-change="handlePageChange"
                @search="handleSearch"
                @open-advanced-search="showSearchModal = true"
                @new-game="handleNewGame"
                @save-game="handleSaveCurrentGame"
                @delete-game="handleDeleteGame"
              />
            </div>

            <!-- 2c. Live Engine & Imbalance Tab (Used in Compact / Small-Screen Mode when Engine is moved out from under board) -->
            <div v-show="activeTab === 'engine'" class="h-full overflow-y-auto flex flex-col gap-2.5">
              <EnginePanel
                :eval-info="evalInfo"
                :is-analyzing="isAnalyzing"
                :available-engines="availableEngines"
                :discovered-engines="discoveredEngines"
                :selected-engine="selectedEngine"
                :multi-pv="engineMultiPv"
                :tablebase-result="tablebaseResult"
                @toggle-analysis="toggleAnalysis"
                @select-engine="handleSelectEngine"
                @change-multi-p-v="handleChangeMultiPV"
                @change-threads="handleChangeThreads"
                @change-hash="handleChangeHash"
                @preview-move="handlePreviewCandidateMove"
              />
              <ImbalanceBar :report="imbalanceReport" />
            </div>

            <!-- 3. Databases & Roles Hub View (Default / Active / Reference / Clipbase) -->
            <div v-show="activeTab === 'db-roles'" class="h-full overflow-hidden">
              <DatabaseHubPanel
                :roles-status="dbRolesStatus"
                :discovered-databases="discoveredDatabases"
                :is-updating-from-local="isUpdatingDbFromLocal"
                :local-merge-status="localMergeStatusMsg"
                @open-d-b="handleOpenDB"
                @create-d-b="handleCreateDB"
                @open-discovered-base="handleOpenDiscoveredBase"
                @set-default-d-b="handleSetDefaultDatabase"
                @set-reference-d-b="handleSetReferenceDatabase"
                @open-default-d-b="handleOpenDefaultDatabase"
                @open-clipbase="handleOpenClipbase"
                @open-mega-downloader="showMegaDatabaseModal = true"
                @open-online-sync="showOnlineSyncModal = true"
                @open-maintenance="showMaintenanceModal = true"
                @update-from-local="handleUpdateDatabaseFromLocal"
              />
            </div>

            <!-- 4. Lichess.org & Chess.com Live Portal View -->
            <div v-show="activeTab === 'portal'" class="h-full overflow-hidden">
              <LivePortalPanel
                :current-fen="currentFEN"
                :default-window-mode="appPreferences.portalWindowMode"
                :lichess-username="appPreferences.lichessUsername"
                :chess-com-username="appPreferences.chessComUsername"
                @open-online-sync="showOnlineSyncModal = true"
                @open-fair-play="showFairPlayModal = true"
                @detach-portal-screen="handleDetachPanelScreen('portal')"
                @float-portal="togglePanelFloater('portal')"
              />
            </div>

            <!-- 5. Game Info & Header View -->
            <div v-show="activeTab === 'info'" class="h-full overflow-y-auto p-4 space-y-4">
              <div class="grid grid-cols-2 gap-3 text-xs font-mono">
                <div class="p-3 bg-zinc-950 rounded-lg border border-zinc-800">
                  <span class="text-zinc-500 uppercase font-semibold">White Player</span>
                  <p class="text-base font-bold text-zinc-100 mt-1">{{ gameTags.White || '?' }}</p>
                  <span v-if="gameTags.WhiteElo" class="text-xs text-emerald-400">Elo: {{ gameTags.WhiteElo }}</span>
                </div>
                <div class="p-3 bg-zinc-950 rounded-lg border border-zinc-800">
                  <span class="text-zinc-500 uppercase font-semibold">Black Player</span>
                  <p class="text-base font-bold text-zinc-100 mt-1">{{ gameTags.Black || '?' }}</p>
                  <span v-if="gameTags.BlackElo" class="text-xs text-emerald-400">Elo: {{ gameTags.BlackElo }}</span>
                </div>
              </div>

              <div class="p-3 bg-zinc-950 rounded-lg border border-zinc-800 space-y-2 text-xs">
                <div class="flex justify-between">
                  <span class="text-zinc-500 font-semibold">Event:</span>
                  <span class="text-zinc-200">{{ gameTags.Event || '?' }}</span>
                </div>
                <div class="flex justify-between">
                  <span class="text-zinc-500 font-semibold">Site:</span>
                  <span class="text-zinc-200">{{ gameTags.Site || '?' }}</span>
                </div>
                <div class="flex justify-between">
                  <span class="text-zinc-500 font-semibold">Date:</span>
                  <span class="text-zinc-200">{{ gameTags.Date || '????.??.??' }}</span>
                </div>
                <div class="flex justify-between">
                  <span class="text-zinc-500 font-semibold">Result:</span>
                  <span class="text-emerald-400 font-bold">{{ gameTags.Result || '*' }}</span>
                </div>
                <div class="flex justify-between">
                  <span class="text-zinc-500 font-semibold">ECO:</span>
                  <span class="text-zinc-200 font-mono">{{ gameTags.ECO || 'A00' }}</span>
                </div>
                <div v-if="gameTags.Annotator" class="flex justify-between">
                  <span class="text-zinc-500 font-semibold">Annotator:</span>
                  <span class="text-amber-400 font-medium">{{ gameTags.Annotator }}</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Vertical Drag Splitter (between Workspace and Database Drawer) -->
      <div
        v-if="showBottomGamesDrawer"
        class="flex items-center justify-center h-2 cursor-row-resize hover:bg-emerald-500/20 active:bg-emerald-500/40 transition-colors group select-none relative z-10 shrink-0"
        :class="{ 'bg-emerald-500/30': isResizingV }"
        title="Drag to resize database drawer (Double-click to reset)"
        @mousedown="startVerticalResize"
        @dblclick="resetVerticalResize"
      >
        <div
          class="h-1 w-12 rounded-full bg-zinc-700 group-hover:bg-emerald-400 group-active:bg-emerald-400 transition-colors"
          :class="{ 'bg-emerald-400': isResizingV }"
        />
      </div>

      <!-- Bottom Panel: Database Games Browser (Auto-collapsed into 'Games' right tab on smaller screens) -->
      <div
        v-if="showBottomGamesDrawer"
        class="bottom-drawer shrink-0 px-2 pb-2 transition-all duration-75 overflow-hidden relative group/gamesdrawer"
        :style="{ height: `${bottomDrawerHeight}px` }"
      >
        <div class="absolute right-4 top-2 z-20 hidden group-hover/gamesdrawer:flex items-center gap-1 bg-zinc-900/95 border border-zinc-700 rounded px-1.5 py-0.5 text-[10px]">
          <button
            class="text-amber-300 hover:text-amber-200 px-1 cursor-pointer"
            title="Undock Database Games List as a Floater Window"
            @click="togglePanelFloater('games')"
          >
            Float ❐
          </button>
          <button
            class="text-purple-300 hover:text-purple-200 px-1 cursor-pointer"
            title="Detach Database Games List into separate Hyprland Screen Window"
            @click="handleDetachPanelScreen('games')"
          >
            Screen ↗
          </button>
        </div>
        <GameBrowser
          :games="gamesList"
          :total-games="totalGames"
          :current-page="currentPage"
          :page-size="pageSize"
          :active-game-id="activeGameId"
          :db-name="dbInfo?.name"
          :active-criteria="searchFilter"
          @load-game="handleLoadGame"
          @page-change="handlePageChange"
          @search="handleSearch"
          @open-advanced-search="showSearchModal = true"
          @new-game="handleNewGame"
          @save-game="handleSaveCurrentGame"
          @delete-game="handleDeleteGame"
        />
      </div>

      <!-- ===================================================================== -->
      <!-- FLOATING PANELS ("FLOATERS") LAYER                                    -->
      <!-- ===================================================================== -->
      <FloatingPanel
        v-for="fp in floatingPanels"
        :key="fp.id"
        :panel="fp"
        @focus="bringFloaterToFront"
        @update-rect="updateFloaterRect"
        @dock="(id, zone) => setPanelDockZone(id, zone || fp.defaultDockZone)"
        @detach-screen="handleDetachPanelScreen"
        @hide="(id) => setPanelMode(id, 'hidden')"
      >
        <div v-if="fp.id === 'board'" class="h-full w-full flex items-center justify-center p-2 overflow-hidden">
          <Chessboard
            :fen="currentFEN"
            :legal-moves="legalMoves"
            :piece-set="currentPieceSet"
            :board-theme="currentTheme"
            :flipped="flipped"
            :last-move="lastMove"
            :in-check="inCheck"
            :sound-enabled="soundEnabled"
            :blindfold="blindfoldMode"
            :show-eval-bar="showEvalBar"
            :eval-score="evalScore"
            :eval-mate="evalMate"
            :threat-squares="threatSquares"
            :arrows="engineArrows"
            :board-size="Math.max(180, Math.min(fp.floaterRect.width - (showEvalBar ? 54 : 24), fp.floaterRect.height - 54))"
            @move="handleMove"
          />
        </div>

        <div v-else-if="fp.id === 'notation'" class="h-full flex flex-col gap-2 p-2">
          <EvaluationGraph
            :analysis="gameAnalysis"
            :current-ply="currentPly"
            :is-analyzing-game="isAnalyzingGame"
            :progress="annotationProgress"
            :moves="flatMoves"
            @jump="jumpToPly"
            @annotate="handleAnnotateGame"
          />
          <div class="flex-1 overflow-hidden">
            <MoveNotation
              :moves="flatMoves"
              :current-ply="currentPly"
              :is-playing="isPlaying"
              :result="gameTags.Result || '*'"
              @jump="jumpToPly"
              @first="goFirst"
              @prev="goPrev"
              @next="goNext"
              @last="goLast"
              @toggle-play="toggleAutoplay"
              @copy-p-g-n="handleExportPGN"
              @set-n-a-g="handleSetNAG"
              @clear-n-a-gs="handleClearNAGs"
              @set-comment="handleSetComment"
              @promote-variation="handlePromoteVariation"
              @delete-variation="handleDeleteVariation"
              @explain-move="handleExplainMove"
            />
          </div>
        </div>

        <div v-else-if="fp.id === 'tree'" class="h-full p-2">
          <OpeningTree
            :fen="currentFEN"
            :tree-data="treeData"
            :loading="treeLoading"
            :db-open="!!dbInfo || !!dbRolesStatus?.referenceDbPath"
            :db-name="dbRolesStatus?.useRefForTree && dbRolesStatus?.referenceDbName ? `${dbRolesStatus.referenceDbName} (Ref)` : dbInfo?.name"
            :eco-code="gameTags.ECO"
            :opening-name="gameTags.Opening || gameTags.Event"
            :discovered-databases="discoveredDatabases"
            @play-move="handleMove"
            @refresh="fetchOpeningTree(currentFEN)"
            @open-db="handleOpenDB"
            @open-discovered-db="handleOpenDiscoveredBase"
            @go-first="goFirst"
            @go-prev="goPrev"
            @pop-out-window="handleOpenTreeWindow"
          />
        </div>

        <div v-else-if="fp.id === 'db-roles'" class="h-full p-2">
          <DatabaseHubPanel
            :roles-status="dbRolesStatus"
            :discovered-databases="discoveredDatabases"
            :is-updating-from-local="isUpdatingDbFromLocal"
            :local-merge-status="localMergeStatusMsg"
            @open-d-b="handleOpenDB"
            @create-d-b="handleCreateDB"
            @open-discovered-base="handleOpenDiscoveredBase"
            @set-default-d-b="handleSetDefaultDatabase"
            @set-reference-d-b="handleSetReferenceDatabase"
            @open-default-d-b="handleOpenDefaultDatabase"
            @open-clipbase="handleOpenClipbase"
            @open-mega-downloader="showMegaDatabaseModal = true"
            @open-online-sync="showOnlineSyncModal = true"
            @open-maintenance="showMaintenanceModal = true"
            @update-from-local="handleUpdateDatabaseFromLocal"
          />
        </div>

        <div v-else-if="fp.id === 'portal'" class="h-full p-2">
          <LivePortalPanel
            :current-fen="currentFEN"
            :default-window-mode="appPreferences.portalWindowMode"
            :lichess-username="appPreferences.lichessUsername"
            :chess-com-username="appPreferences.chessComUsername"
            @open-online-sync="showOnlineSyncModal = true"
            @open-fair-play="showFairPlayModal = true"
            @detach-portal-screen="handleDetachPanelScreen('portal')"
            @float-portal="togglePanelFloater('portal')"
          />
        </div>

        <div v-else-if="fp.id === 'engine'" class="h-full p-2">
          <EnginePanel
            :eval-info="evalInfo"
            :is-analyzing="isAnalyzing"
            :available-engines="availableEngines"
            :discovered-engines="discoveredEngines"
            :selected-engine="selectedEngine"
            :multi-pv="engineMultiPv"
            :tablebase-result="tablebaseResult"
            @toggle-analysis="toggleAnalysis"
            @select-engine="handleSelectEngine"
            @change-multi-p-v="handleChangeMultiPV"
            @change-threads="handleChangeThreads"
            @change-hash="handleChangeHash"
            @preview-move="handlePreviewCandidateMove"
          />
        </div>

        <div v-else-if="fp.id === 'imbalance'" class="h-full p-2">
          <ImbalanceBar :report="imbalanceReport" />
        </div>

        <div v-else-if="fp.id === 'games'" class="h-full p-2">
          <GameBrowser
            :games="gamesList"
            :total-games="totalGames"
            :current-page="currentPage"
            :page-size="pageSize"
            :active-game-id="activeGameId"
            :db-name="dbInfo?.name"
            :active-criteria="searchFilter"
            @load-game="handleLoadGame"
            @page-change="handlePageChange"
            @search="handleSearch"
            @open-advanced-search="showSearchModal = true"
            @new-game="handleNewGame"
            @save-game="handleSaveCurrentGame"
            @delete-game="handleDeleteGame"
          />
        </div>

        <div v-else-if="fp.id === 'info'" class="h-full overflow-y-auto p-3 space-y-2 text-xs">
          <div class="font-bold text-zinc-100">{{ gameTags.White || '?' }} vs {{ gameTags.Black || '?' }}</div>
          <div class="text-zinc-400">{{ gameTags.Event || '?' }} ({{ gameTags.Date || '?' }}) • {{ gameTags.Result || '*' }}</div>
        </div>
      </FloatingPanel>
    </template>

    <!-- ===================================================================== -->
    <!-- PREFERENCES & OPTIONS DOCUMENT MODAL (Ctrl+,)                         -->
    <!-- ===================================================================== -->
    <PreferencesModal
      v-if="showPreferencesModal"
      :initial-section="preferencesSection"
      :preferences="appPreferences"
      :roles-status="dbRolesStatus"
      :discovered-databases="discoveredDatabases"
      :piece-sets="pieceSets"
      :board-themes="boardThemes"
      :discovered-engines="discoveredEngines"
      :current-piece-set="currentPieceSet"
      :current-theme-id="currentThemeId"
      :current-u-i-theme="currentUITheme"
      :sound-enabled="soundEnabled"
      :show-eval-bar="showEvalBar"
      :threat-mode="threatMode"
      :blindfold="blindfoldMode"
      :left-col-width="leftColWidth"
      :bottom-drawer-height="bottomDrawerHeight"
      :is-updating-from-local="isUpdatingDbFromLocal"
      :local-merge-status="localMergeStatusMsg"
      @close="showPreferencesModal = false"
      @save-preferences="handleSaveAppPreferences"
      @set-default-d-b="handleSetDefaultDatabase"
      @set-reference-d-b="handleSetReferenceDatabase"
      @open-default-d-b="handleOpenDefaultDatabase"
      @open-clipbase="handleOpenClipbase"
      @open-discovered-base="handleOpenDiscoveredBase"
      @change-piece-set="currentPieceSet = $event"
      @change-theme="currentThemeId = $event"
      @change-u-i-theme="applyUITheme"
      @toggle-sound="toggleSound"
      @toggle-eval-bar="toggleEvalBar"
      @toggle-threat-mode="toggleThreatMode"
      @toggle-blindfold="toggleBlindfold"
      @update-left-width="(w: number) => { leftColWidth = w }"
      @update-bottom-height="(h: number) => { bottomDrawerHeight = h }"
      @detach-panel-screen="handleDetachPanelScreen"
      @move-main-workspace="handleMoveCurrentWindowToWorkspace"
      @update-from-local="handleUpdateDatabaseFromLocal"
      @launch-portal-window="handleLaunchPortalWindow"
    />

    <!-- Setup Board / FEN Editor Modal (Ctrl+B) -->
    <BoardSetupModal
      v-if="showSetupBoardModal"
      :initial-fen="currentFEN"
      :piece-set="currentPieceSet"
      :board-theme="currentTheme"
      @apply="handleSetupBoardApply"
      @close="showSetupBoardModal = false"
    />

    <!-- Play Against Engine Modal -->
    <PlayEngineModal
      v-if="showPlayEngineModal"
      :discovered-engines="discoveredEngines"
      :available-engines="availableEngines"
      :current-piece-set="currentPieceSet"
      :board-theme="currentTheme"
      @import-game="handleImportGameFromMatch"
      @close="showPlayEngineModal = false"
    />

    <!-- Tactics Puzzle Trainer Modal -->
    <TacticsTrainerModal
      v-if="showTacticsModal"
      :system-bases="systemBases"
      :current-piece-set="currentPieceSet"
      :board-theme="currentTheme"
      @load-into-app="handleLoadTacticsIntoApp"
      @close="showTacticsModal = false"
    />

    <!-- Dedicated Opening Tree & Repertoire Explorer Window (F3) -->
    <OpeningTreeModal
      :is-open="showOpeningTreeModal"
      :fen="currentFEN"
      :tree-data="treeData"
      :loading="treeLoading"
      :db-open="!!dbInfo || !!dbRolesStatus?.referenceDbPath"
      :db-name="dbRolesStatus?.useRefForTree && dbRolesStatus?.referenceDbName ? `${dbRolesStatus.referenceDbName} (Ref)` : dbInfo?.name"
      :eco-code="gameTags.ECO"
      :opening-name="gameTags.Opening || gameTags.Event"
      :discovered-databases="discoveredDatabases"
      @play-move="handleMove"
      @refresh="fetchOpeningTree(currentFEN)"
      @open-db="handleOpenDB"
      @open-discovered-db="handleOpenDiscoveredBase"
      @go-first="goFirst"
      @go-prev="goPrev"
      @close="showOpeningTreeModal = false"
    />

    <!-- Opening Repertoire Builder & Trainer Modal -->
    <OpeningTrainerModal
      v-if="showOpeningTrainerModal"
      :current-piece-set="currentPieceSet"
      :board-theme="currentTheme"
      @load-into-app="handleSetupBoardApply"
      @close="showOpeningTrainerModal = false"
    />

    <!-- Progressive Tactics Challenge Modal -->
    <ProgressiveTacticsModal
      v-if="showProgressiveTacticsModal"
      :current-piece-set="currentPieceSet"
      :board-theme="currentTheme"
      @load-into-app="handleSetupBoardApply"
      @close="showProgressiveTacticsModal = false"
    />

    <!-- Database Puzzle Miner Modal -->
    <PuzzleMinerModal
      v-if="showPuzzleMinerModal"
      :db-name="dbInfo?.name"
      :current-piece-set="currentPieceSet"
      :board-theme="currentTheme"
      @load-into-app="handleSetupBoardApply"
      @close="showPuzzleMinerModal = false"
    />

    <!-- Keyboard Shortcuts Modal -->
    <KeyboardShortcutsModal
      v-if="showShortcutsModal"
      @close="showShortcutsModal = false"
    />

    <!-- About NewScid Modal -->
    <AboutModal
      v-if="showAboutModal"
      @close="showAboutModal = false"
      @support="showAboutModal = false; showSupportModal = true"
    />

    <!-- Voluntary Author Support ($5+ Stripe) Modal -->
    <SupportModal
      v-if="showSupportModal"
      @close="showSupportModal = false"
    />

    <!-- Online Help & User Documentation Modal (F1) -->
    <HelpModal
      v-if="showHelpModal"
      @close="showHelpModal = false"
      @open-shortcuts="showHelpModal = false; showShortcutsModal = true"
    />

    <!-- Header / General Search Modal (Ctrl+H) -->
    <HeaderSearchModal
      v-if="showSearchModal"
      :db-name="dbInfo?.name"
      :total-games="totalGames"
      :initial-criteria="searchFilter"
      @apply-search="handleSearch"
      @close="showSearchModal = false"
    />

    <!-- Database Maintenance & Tools Modal -->
    <DatabaseMaintenanceModal
      v-if="showMaintenanceModal"
      :db-name="dbInfo?.name"
      :db-path="dbInfo?.basePath"
      :total-games="totalGames"
      @db-updated="handleDatabaseUpdated"
      @close="showMaintenanceModal = false"
    />

    <!-- Online Account Sync & Merge Modal -->
    <OnlineSyncModal
      v-if="showOnlineSyncModal"
      :active-db-path="dbInfo?.basePath"
      :active-db-name="dbInfo?.name"
      @db-updated="handleDatabaseUpdated"
      @close="showOnlineSyncModal = false"
    />

    <!-- Player Simulation Sparring Bot Modal -->
    <PlayerSimulationModal
      v-if="showPlayerSimulationModal"
      :active-db-name="dbInfo?.name"
      :initial-persona="selectedSimulationPersona"
      @start-match="handleStartSimulationMatch"
      @close="showPlayerSimulationModal = false; selectedSimulationPersona = undefined"
    />

    <!-- Player Personas & Unified Designators Modal -->
    <PersonaModal
      v-if="showPersonaModal"
      :active-db-name="dbInfo?.name"
      @spar-persona="handleSparPersona"
      @open-persona-db="handleOpenPersonaDb"
      @exploit-persona="openAntiPersonaModal($event)"
      @close="showPersonaModal = false"
    />

    <!-- Tournament Crosstables & Standings Modal (F7) -->
    <TournamentCrosstableModal
      v-if="showTournamentCrosstableModal"
      :show="showTournamentCrosstableModal"
      :active-event="gameTags.Event"
      @load-game="handleLoadGame"
      @close="showTournamentCrosstableModal = false"
    />

    <!-- ECO Opening Classification Browser Modal (F6) -->
    <ECOBrowserModal
      v-if="showEcoBrowserModal"
      @close="showEcoBrowserModal = false"
      @play-opening="handlePlayOpeningMoves"
      @search-eco-in-db="handleSearchEcoInDb"
    />

    <!-- Guess the Move (Solitaire Chess) Training Modal -->
    <GuessTheMoveModal
      v-if="showGuessTheMoveModal"
      :active-game-id="activeGameId || 0"
      :game-header="{
        white: gameTags.White || 'White',
        black: gameTags.Black || 'Black',
        event: gameTags.Event || 'Game',
        date: gameTags.Date || 'Unknown',
        result: gameTags.Result || '*'
      }"
      :current-piece-set="currentPieceSet"
      :board-theme="currentTheme"
      @close="showGuessTheMoveModal = false"
    />

    <!-- Database Demographics & Rating Tracker Modal (Ctrl+D) -->
    <DatabaseStatsModal
      v-if="showDatabaseStatsModal"
      :db-name="dbInfo?.name"
      :initial-player="gameTags.White || ''"
      @close="showDatabaseStatsModal = false"
      @load-game="handleLoadGame"
    />

    <!-- Edit Game Headers & Tags Modal (Ctrl+T) -->
    <EditTagsModal
      v-if="showEditTagsModal"
      :game-id="activeGameId || 1"
      :initial-tags="gameTags"
      @tags-updated="handleTagsUpdated"
      @close="showEditTagsModal = false"
    />

    <!-- The Week In Chess (TWIC) Downloader Modal -->
    <TWICDownloadModal
      v-if="showTwicModal"
      :db-name="dbInfo?.name"
      @imported="handleTwicImported"
      @close="showTwicModal = false"
    />

    <!-- One-Click Master & Mega Database Downloader Modal (LumbrasGigaBase & Lichess Open/Elite) -->
    <MegaDatabaseModal
      v-if="showMegaDatabaseModal"
      :db-open="!!dbInfo?.basePath"
      :db-name="dbInfo?.name"
      @imported="handleTwicImported"
      @close="showMegaDatabaseModal = false"
    />

    <!-- Longitudinal Fair-Play & Time-Control Consistency Inspector Modal -->
    <FairPlayModal
      v-if="showFairPlayModal"
      :db-open="!!dbInfo?.basePath"
      :initial-player="gameTags.White && gameTags.White !== '?' ? gameTags.White : ''"
      @select-game="(gid: number) => { showFairPlayModal = false; handleLoadGame(gid) }"
      @imported="handleTwicImported"
      @close="showFairPlayModal = false"
    />

    <!-- Engine vs Engine Match / Tournament Modal -->
    <EngineMatchModal
      v-if="showEngineMatchModal"
      :discovered-engines="discoveredEngines"
      :available-engines="availableEngines"
      :db-name="dbInfo?.name"
      @match-finished="handleMatchFinished"
      @close="showEngineMatchModal = false"
    />

    <!-- Export Game (HTML & LaTeX) Modal -->
    <ExportGameModal
      v-if="showExportGameModal"
      :is-open="showExportGameModal"
      @close="showExportGameModal = false"
    />

    <!-- Chess960 (Fischer Random) Modal -->
    <Chess960Modal
      v-if="showChess960Modal"
      @close="showChess960Modal = false"
      @load-position="handleLoadChess960"
    />

    <!-- Board Vision & Coordinate Trainer Modal -->
    <CoordinateTrainerModal
      v-if="showCoordinateTrainerModal"
      @close="showCoordinateTrainerModal = false"
    />

    <!-- Import PGN Modal -->
    <div
      v-if="showImportModal"
      class="fixed inset-0 bg-black/60 backdrop-blur-xs flex items-center justify-center z-50 p-4 animate-fade-in"
      @click.self="showImportModal = false"
    >
      <div class="bg-zinc-900 border border-zinc-700 w-full max-w-lg rounded-2xl p-5 shadow-2xl flex flex-col gap-3">
        <h3 class="font-bold text-sm text-zinc-100">Import PGN Game</h3>
        <textarea
          v-model="importPGNText"
          rows="8"
          placeholder="Paste PGN tags and moves here..."
          class="w-full bg-zinc-950 border border-zinc-700 rounded-lg p-3 text-xs font-mono text-zinc-200 placeholder-zinc-500 focus:outline-hidden focus:border-emerald-500"
        />
        <div class="flex justify-end gap-2">
          <button
            class="px-3 py-1.5 rounded-lg bg-zinc-800 text-zinc-300 text-xs font-medium hover:bg-zinc-700 cursor-pointer"
            @click="showImportModal = false"
          >
            Cancel
          </button>
          <button
            class="px-4 py-1.5 rounded-lg bg-emerald-600 text-white text-xs font-medium hover:bg-emerald-500 cursor-pointer"
            @click="confirmImportPGN"
          >
            Load Game
          </button>
        </div>
      </div>
    </div>

    <!-- Zen Mode Floating Exit Indicator -->
    <button
      v-if="zenMode"
      class="fixed bottom-4 right-4 z-50 px-3 py-1.5 rounded-xl bg-zinc-900/90 hover:bg-zinc-800 text-zinc-200 border border-zinc-700 text-xs font-semibold flex items-center gap-1.5 shadow-2xl cursor-pointer backdrop-blur-sm"
      @click="toggleZenMode"
    >
      <Minimize2 class="w-3.5 h-3.5 text-emerald-400" />
      <span>Exit Zen Mode (Z)</span>
    </button>

    <!-- Universal Command Palette Modal (Ctrl+K) -->
    <CommandPalette
      :is-open="showCommandPalette"
      :commands="commandPaletteItems"
      @close="showCommandPalette = false"
    />

    <!-- Natural Language Move Explainer Modal -->
    <ExplainMoveModal
      v-if="showExplainMoveModal"
      :before-fen="explainMoveTarget?.beforeFen || startingFEN"
      :move-uci="explainMoveTarget?.moveUci || ''"
      :score-cp="explainMoveTarget?.scoreCp"
      :best-move="explainMoveTarget?.bestMove"
      @close="showExplainMoveModal = false"
    />

    <!-- Personal Blunder Vault (SuperMemo SM-2 Spaced Repetition) Modal -->
    <BlunderVaultModal
      v-if="showBlunderVaultModal"
      :board-theme="currentTheme"
      :piece-set="currentPieceSet"
      @close="showBlunderVaultModal = false"
    />

    <!-- Opponent Scouting Dossier & Repertoire Prep Modal -->
    <OpponentScoutModal
      v-if="showOpponentScoutModal"
      :active-db-name="dbInfo?.name"
      @load-game="handleLoadGame"
      @exploit-player="openAntiPersonaModal($event)"
      @close="showOpponentScoutModal = false"
    />

    <!-- Opponent Opening Exploiter & Custom Weaponized Book Generator Modal -->
    <AntiPersonaModal
      v-if="showAntiPersonaModal"
      :initial-persona-id="antiPersonaTargetId"
      :initial-player-name="antiPersonaTargetName"
      :board-theme="currentTheme"
      :piece-set="currentPieceSet"
      @close="showAntiPersonaModal = false"
      @spar-bot="handleSparBotFromAnti"
      @train-repertoire="handleTrainAntiRepertoire"
    />

    <!-- Piece Safety Scanner & Blunder Prevention Modal -->
    <PieceSafetyModal
      v-if="showPieceSafetyModal"
      :initial-f-e-n="currentFEN"
      :piece-set="currentPieceSet"
      :board-theme="currentTheme"
      @close="showPieceSafetyModal = false"
      @load-position="(fen) => handleSetupBoardApply(fen)"
    />

    <!-- Intuition Storm (Speed Reflex Trainer) Modal -->
    <IntuitionStormModal
      v-if="showIntuitionStormModal"
      :initial-f-e-n="currentFEN"
      :piece-set="currentPieceSet"
      :board-theme="currentTheme"
      @close="showIntuitionStormModal = false"
      @load-position="(fen) => handleSetupBoardApply(fen)"
    />

    <!-- Time Management & Tanking Dossier Modal -->
    <TimeManagementModal
      v-if="showTimeManagementModal"
      :game-id="activeGameId || 0"
      @close="showTimeManagementModal = false"
      @jump-to-ply="(ply) => { jumpToPly(ply); showTimeManagementModal = false; }"
    />

    <!-- Repertoire Shortener (Pareto 80/20 Pruner) Modal -->
    <RepertoireShortenerModal
      v-if="showRepertoireShortenerModal"
      @close="showRepertoireShortenerModal = false"
      @repertoire-saved="() => {}"
    />

    <!-- Quiet Moves & Prophylaxis Drill Modal -->
    <ProphylaxisModal
      v-if="showProphylaxisModal"
      :initial-f-e-n="currentFEN"
      :piece-set="currentPieceSet"
      :board-theme="currentTheme"
      @close="showProphylaxisModal = false"
      @load-position="(fen) => handleSetupBoardApply(fen)"
    />
  </div>
</template>
