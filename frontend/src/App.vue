<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import MenuBar from './components/MenuBar.vue'
import HeaderBar, { PieceSetOption, BoardThemeOption, DiscoveredDatabase } from './components/HeaderBar.vue'
import Chessboard from './components/Chessboard.vue'
import MoveNotation, { FlatMove } from './components/MoveNotation.vue'
import OpeningTree, { TreeData } from './components/OpeningTree.vue'
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
import { Bot } from '@lucide/vue'
import { playerbot, persona, scid, tournament } from '../wailsjs/go/models'

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
const activeTab = ref<'notation' | 'tree' | 'info'>('notation')
const bottomDrawerOpen = ref(true)

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

const evalScore = computed(() => evalInfo.value?.scoreCp ?? 0)
const evalMate = computed(() => evalInfo.value?.mateIn ?? 0)

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



// Resizable Panes State
const leftColWidth = ref<number>(Number(localStorage.getItem('newscid:leftColWidth')) || 560)
const bottomDrawerHeight = ref<number>(Number(localStorage.getItem('newscid:bottomDrawerHeight')) || 240)
const isResizingH = ref(false)
const isResizingV = ref(false)
const isMdScreen = ref(window.innerWidth >= 768)

function handleWindowResize() {
  isMdScreen.value = window.innerWidth >= 768
}

function startHorizontalResize(e: MouseEvent) {
  e.preventDefault()
  isResizingH.value = true
  const startX = e.clientX
  const startWidth = leftColWidth.value
  document.body.style.cursor = 'col-resize'
  document.body.style.userSelect = 'none'

  function onMouseMove(moveEvent: MouseEvent) {
    const delta = moveEvent.clientX - startX
    const minW = 360
    const maxW = Math.max(minW, window.innerWidth - 300)
    leftColWidth.value = Math.max(minW, Math.min(maxW, startWidth + delta))
  }

  function onMouseUp() {
    isResizingH.value = false
    document.body.style.cursor = ''
    document.body.style.userSelect = ''
    window.removeEventListener('mousemove', onMouseMove)
    window.removeEventListener('mouseup', onMouseUp)
    localStorage.setItem('newscid:leftColWidth', String(leftColWidth.value))
  }

  window.addEventListener('mousemove', onMouseMove)
  window.addEventListener('mouseup', onMouseUp)
}

function resetHorizontalResize() {
  leftColWidth.value = 560
  localStorage.setItem('newscid:leftColWidth', '560')
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
  }

  function onMouseUp() {
    isResizingV.value = false
    document.body.style.cursor = ''
    document.body.style.userSelect = ''
    window.removeEventListener('mousemove', onMouseMove)
    window.removeEventListener('mouseup', onMouseUp)
    localStorage.setItem('newscid:bottomDrawerHeight', String(bottomDrawerHeight.value))
  }

  window.addEventListener('mousemove', onMouseMove)
  window.addEventListener('mouseup', onMouseUp)
}

function resetVerticalResize() {
  bottomDrawerHeight.value = 240
  localStorage.setItem('newscid:bottomDrawerHeight', '240')
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
    window.addEventListener('resize', handleWindowResize)

    // 1. Load sets and themes
    pieceSets.value = (await GetPieceSets()) as PieceSetOption[]
    boardThemes.value = (await GetBoardThemes()) as BoardThemeOption[]

    // 2. Discover user databases and system bases
    discoveredDatabases.value = (await GetDiscoveredDatabases()) as DiscoveredDatabase[]
    systemBases.value = await GetScidSystemBases()

    // 3. Discover available UCI engines and configured Scid engines
    discoveredEngines.value = (await GetDiscoveredEngines()) as EngineOption[]
    availableEngines.value = await GetAvailableEngines()

    if (discoveredEngines.value.length > 0) {
      const firstAvailable = discoveredEngines.value.find(e => e.available)
      if (firstAvailable) {
        selectedEngine.value = firstAvailable.name || firstAvailable.cmd
      }
    } else if (availableEngines.value.length > 0) {
      selectedEngine.value = availableEngines.value[0]
    }

    // 4. Listen to live engine events
    EventsOn('engine:eval', (data: EvalInfo) => {
      evalInfo.value = data
    })

    const initMPV = await GetEngineMultiPV()
    if (initMPV > 0) {
      engineMultiPv.value = initMPV
    }

    // 5. Auto-open user's database or first discovered base
    let autoBase: string | null = null
    if (discoveredDatabases.value.length > 0) {
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
    }

    // Update board state for start position
    await updateBoardState(startingFEN)
  } catch (err) {
    console.error('Initialization error:', err)
  }

  window.addEventListener('keydown', handleKeydown)
})

onUnmounted(() => {
  window.removeEventListener('keydown', handleKeydown)
  window.removeEventListener('resize', handleWindowResize)
  EventsOff('engine:eval')
  if (playInterval) clearInterval(playInterval)
})

watch(currentFEN, (newFen) => {
  updateTablebase(newFen)
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

    // If live engine analysis is active, send position
    if (isAnalyzing.value) {
      AnalyzePosition(state.fen, [], 20).catch(console.error)
    }
  } catch (err) {
    console.error('Error updating board state:', err)
  }
}

async function handleMove(uci: string) {
  try {
    let moveSAN = uci
    try {
      moveSAN = await FormatMove(currentFEN.value, uci, 'san')
    } catch {
      // fallback to uci
    }

    const nextState = await MakeMove(currentFEN.value, uci)
    const from = uci.slice(0, 2)
    const to = uci.slice(2, 4)

    // Truncate forward history if moving from middle of game
    history.value = history.value.slice(0, currentPly.value + 1)
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

    // If simulated match is active and game is not finished, trigger bot response!
    if (activeSimulation.value && !isBotThinking.value) {
      if (nextState.isCheckmate) {
        const winner = isWhite ? 'Player' : activeSimulation.value.playerName
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
    }
    return
  }

  // Single Keys
  if (e.key === 'F1') {
    e.preventDefault()
    showHelpModal.value = true
  } else if (e.key === 'F2') {
    e.preventDefault()
    toggleAnalysis()
  } else if (e.key === 'F6') {
    e.preventDefault()
    showEcoBrowserModal.value = !showEcoBrowserModal.value
  } else if (e.key === 'F7') {
    e.preventDefault()
    showTournamentCrosstableModal.value = !showTournamentCrosstableModal.value
  } else if (e.key === '?') {
    e.preventDefault()
    showShortcutsModal.value = true
  } else if (e.key === 'ArrowLeft') {
    goPrev()
  } else if (e.key === 'ArrowRight') {
    goNext()
  } else if (e.key === 'Home') {
    goFirst()
  } else if (e.key === 'End') {
    goLast()
  } else if (e.key === 'f' || e.key === 'F') {
    flipped.value = !flipped.value
  } else if (e.key === ' ') {
    e.preventDefault()
    toggleAutoplay()
  }
}

async function fetchOpeningTree(fen: string) {
  treeLoading.value = true
  try {
    const res = await GetOpeningTree(fen)
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
    const gameObj: any = {
      tags: gameTags.value,
      startFen: history.value[0].fen,
      result: gameTags.value.Result || '*',
      plyCount: flatMoves.value.length,
    }
    const newId = await SaveGame(gameObj)
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

// Full Game Analysis (Annotate)
async function handleAnnotateGame() {
  if (isAnalyzingGame.value) return
  if (flatMoves.value.length === 0) {
    alert('There are no moves in the current game to annotate.')
    return
  }
  isAnalyzingGame.value = true
  try {
    const gameObj: any = {
      tags: gameTags.value,
      startFen: history.value[0].fen,
      result: gameTags.value.Result || '*',
      plyCount: flatMoves.value.length,
    }
    const pgn = await ExportPGN(gameObj)
    const analysis = await AnnotatePGN(pgn, 150)
    gameAnalysis.value = analysis as GameAnalysisData

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
    alert(`Annotation Complete!\nWhite Accuracy: ${analysis.whiteAccuracy}%\nBlack Accuracy: ${analysis.blackAccuracy}%\nBlunders: White ${analysis.whiteBlunders}, Black ${analysis.blackBlunders}`)
  } catch (err) {
    alert(`Game annotation failed: ${err}`)
  } finally {
    isAnalyzingGame.value = false
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
    const gameObj: any = {
      tags: gameTags.value,
      startFen: history.value[0].fen,
      result: gameTags.value.Result || '*',
      plyCount: flatMoves.value.length,
    }
    const pgn = await ExportPGN(gameObj)
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
      const winner = isBotWhite ? activeSimulation.value.playerName : 'Player'
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
</script>

<template>
  <div class="app-root flex flex-col h-screen w-screen bg-zinc-950 text-zinc-100 overflow-hidden font-sans">
    <!-- Desktop Application Menu Bar (Scid vs. PC Parity) -->
    <MenuBar
      :db-name="dbInfo?.name"
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
      @opening-trainer="showOpeningTrainerModal = true"
      @progressive-tactics="showProgressiveTacticsModal = true"
      @mine-puzzles="showPuzzleMinerModal = true"
      @search-header="showSearchModal = true"
      @search-position="handleSearchPosition"
      @toggle-flip="flipped = !flipped"
      @change-piece-set="currentPieceSet = $event"
      @change-theme="currentThemeId = $event"
      @open-settings="handleOpenSettings"
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
      @engine-match="showEngineMatchModal = true"
      @export-game="showExportGameModal = true"
      @standardize-names="handleStandardizeNames"
      :sound-enabled="soundEnabled"
      :blindfold="blindfoldMode"
      :threat-mode="threatMode"
      :show-eval-bar="showEvalBar"
      @toggle-sound="toggleSound"
      @toggle-blindfold="toggleBlindfold"
      @toggle-threat-mode="toggleThreatMode"
      @toggle-eval-bar="toggleEvalBar"
      @chess960="showChess960Modal = true"
      @coordinate-trainer="showCoordinateTrainerModal = true"
    />


    <!-- Top Header Bar -->
    <HeaderBar
      ref="headerBarRef"
      :db-name="dbInfo?.name"
      :system-bases="systemBases"
      :discovered-databases="discoveredDatabases"
      :piece-sets="pieceSets"
      :board-themes="boardThemes"
      :current-piece-set="currentPieceSet"
      :current-theme-id="currentThemeId"
      :flipped="flipped"
      @open-d-b="handleOpenDB"
      @create-d-b="handleCreateDB"
      @open-system-base="handleOpenSystemBase"
      @open-discovered-base="handleOpenDiscoveredBase"
      @import-p-g-n="handleImportPGN"
      @export-p-g-n="handleExportPGN"
      @change-piece-set="currentPieceSet = $event"
      @change-theme="currentThemeId = $event"
      @toggle-flip="flipped = !flipped"
    />

    <!-- Main Workspace (Split Grid) -->
    <div class="workspace-grid flex-1 flex flex-col md:flex-row overflow-hidden p-2 gap-1 relative">
      <!-- Left Column: Chessboard & Engine Control (Resizable) -->
      <div
        class="left-col flex flex-col items-center justify-between gap-2 w-full md:w-auto shrink-0 overflow-y-auto"
        :style="{ width: isMdScreen ? `${leftColWidth}px` : '100%' }"
      >
        <!-- Player Simulation Sparring Match Status -->
        <div
          v-if="activeSimulation"
          class="w-full bg-gradient-to-r from-purple-950/80 via-zinc-900 to-purple-950/80 border border-purple-500/40 rounded-xl p-2.5 flex items-center justify-between text-xs shadow-md animate-fade-in"
        >
          <div class="flex items-center gap-2 min-w-0">
            <div class="w-7 h-7 rounded-lg bg-purple-500/20 border border-purple-500/40 flex items-center justify-center text-purple-300 shrink-0">
              <Bot class="w-4 h-4" :class="{ 'animate-pulse text-purple-400': isBotThinking }" />
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

        <!-- Interactive Chessboard -->
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
          @move="handleMove"
        />

        <!-- Live Engine Analysis Bar -->
        <div class="w-full">
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
        title="Drag to resize panes (Double-click to reset)"
        @mousedown="startHorizontalResize"
        @dblclick="resetHorizontalResize"
      >
        <div
          class="w-1 h-8 rounded-full bg-zinc-700 group-hover:bg-emerald-400 group-active:bg-emerald-400 transition-colors"
          :class="{ 'bg-emerald-400': isResizingH }"
        />
      </div>

      <!-- Right Column: Tabbed Panels (Moves, Opening Tree, Game Info) -->
      <div class="right-col flex-1 min-w-0 flex flex-col overflow-hidden bg-zinc-900 border border-zinc-800 rounded-xl shadow-lg">
        <!-- Tabs Header -->
        <div class="tabs-header flex items-center justify-between p-2 bg-zinc-950 border-b border-zinc-800">
          <div class="flex items-center gap-1">
            <button
              class="px-3 py-1.5 rounded-lg text-xs font-semibold transition-all cursor-pointer"
              :class="activeTab === 'notation'
                ? 'bg-emerald-600 text-white shadow-sm'
                : 'text-zinc-400 hover:text-zinc-200 hover:bg-zinc-800'"
              @click="activeTab === 'notation'"
            >
              Notation & Analysis
            </button>
            <button
              class="px-3 py-1.5 rounded-lg text-xs font-semibold transition-all cursor-pointer"
              :class="activeTab === 'tree'
                ? 'bg-emerald-600 text-white shadow-sm'
                : 'text-zinc-400 hover:text-zinc-200 hover:bg-zinc-800'"
              @click="activeTab === 'tree'"
            >
              Opening Tree
            </button>
            <button
              class="px-3 py-1.5 rounded-lg text-xs font-semibold transition-all cursor-pointer"
              :class="activeTab === 'info'
                ? 'bg-emerald-600 text-white shadow-sm'
                : 'text-zinc-400 hover:text-zinc-200 hover:bg-zinc-800'"
              @click="activeTab === 'info'"
            >
              Game Info
            </button>
          </div>

          <!-- Bottom Drawer Toggle -->
          <button
            class="text-xs text-zinc-400 hover:text-zinc-200 px-2 py-1 rounded bg-zinc-800/80 hover:bg-zinc-800 transition-colors cursor-pointer"
            @click="bottomDrawerOpen = !bottomDrawerOpen"
          >
            {{ bottomDrawerOpen ? 'Hide Database ▼' : 'Show Database ▲' }}
          </button>
        </div>

        <!-- Tab Content -->
        <div class="tab-content flex-1 overflow-hidden p-2">
          <!-- 1. Notation & Evaluation Curve View -->
          <div v-show="activeTab === 'notation'" class="h-full flex flex-col gap-2">
            <!-- Evaluation Graph -->
            <EvaluationGraph
              :analysis="gameAnalysis"
              :current-ply="currentPly"
              :is-analyzing-game="isAnalyzingGame"
              @jump="jumpToPly"
              @annotate="handleAnnotateGame"
            />

            <!-- Move Notation List -->
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
              />
            </div>
          </div>

          <!-- 2. Opening Tree View -->
          <div v-show="activeTab === 'tree'" class="h-full">
            <OpeningTree
              :tree-data="treeData"
              :loading="treeLoading"
              @play-move="handleMove"
            />
          </div>

          <!-- 3. Game Info & Header View -->
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
      v-if="bottomDrawerOpen"
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

    <!-- Bottom Panel: Database Games Browser (Collapsible & Resizable) -->
    <div
      v-if="bottomDrawerOpen"
      class="bottom-drawer shrink-0 px-2 pb-2 transition-all duration-75 overflow-hidden"
      :style="{ height: `${bottomDrawerHeight}px` }"
    >
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
  </div>
</template>
