<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import HeaderBar, { PieceSetOption, BoardThemeOption } from './components/HeaderBar.vue'
import Chessboard from './components/Chessboard.vue'
import MoveNotation, { FlatMove } from './components/MoveNotation.vue'
import OpeningTree, { TreeData } from './components/OpeningTree.vue'
import EnginePanel, { EvalInfo } from './components/EnginePanel.vue'
import GameBrowser, { GameSummary } from './components/GameBrowser.vue'

// Import Wails backend bindings
import {
  OpenDatabase,
  CreateDatabase,
  CloseDatabase,
  GetDatabaseInfo,
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
  GetPieceSets,
  GetBoardThemes,
  GetScidSystemBases,
  ParsePGN,
  ExportPGN,
} from '../wailsjs/go/main/App'
import { EventsOn, EventsOff } from '../wailsjs/runtime/runtime'

const startingFEN = 'rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1'

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
const gamesList = ref<GameSummary[]>([])
const totalGames = ref(0)
const currentPage = ref(1)
const pageSize = ref(20)
const searchFilter = ref<{ player?: string; result?: string; eco?: string }>({})

// Engine State
const isAnalyzing = ref(false)
const availableEngines = ref<string[]>([])
const selectedEngine = ref('')
const evalInfo = ref<EvalInfo | null>(null)

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

// PGN Import Modal
const showImportModal = ref(false)
const importPGNText = ref('')

// Initialize
onMounted(async () => {
  try {
    // 1. Load sets and themes
    pieceSets.value = (await GetPieceSets()) as PieceSetOption[]
    boardThemes.value = (await GetBoardThemes()) as BoardThemeOption[]

    // 2. Discover system sample databases (Tactics, Mate in 1, etc.)
    systemBases.value = await GetScidSystemBases()

    // 3. Discover available UCI engines
    availableEngines.value = await GetAvailableEngines()
    if (availableEngines.value.length > 0) {
      selectedEngine.value = availableEngines.value[0]
    }

    // 4. Listen to live engine events
    EventsOn('engine:eval', (data: EvalInfo) => {
      evalInfo.value = data
    })

    // 5. If system bases exist, auto-open Tactics base for immediate exploration
    if (systemBases.value.length > 0) {
      await handleOpenSystemBase(systemBases.value[0])
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
  EventsOff('engine:eval')
  if (playInterval) clearInterval(playInterval)
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
      san: uci, // Fallback SAN
      uci,
      fen: nextState.fen,
      variationLevel: 0,
    })

    await updateBoardState(nextState.fen)
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

function handleKeydown(e: KeyboardEvent) {
  if (e.target instanceof HTMLInputElement || e.target instanceof HTMLTextAreaElement) {
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

async function handleLoadGame(id: number) {
  try {
    const g = await GetGame(id)
    activeGameId.value = id
    gameTags.value = g.tags || {}

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

    // Jump to start of game (or end of game)
    jumpToPly(0)
  } catch (err) {
    alert(`Error loading game ${id}: ${err}`)
  }
}

async function handleSaveCurrentGame() {
  if (!dbInfo.value) {
    alert('Please open or create a database first before saving.')
    return
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

function handleSearch(criteria: { player?: string; result?: string; eco?: string }) {
  searchFilter.value = criteria
  currentPage.value = 1
  refreshGames()
}

function handlePageChange(p: number) {
  currentPage.value = p
  refreshGames()
}

function handleNewGame() {
  activeGameId.value = undefined
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
    await navigator.clipboard.writeText(pgn)
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
</script>

<template>
  <div class="app-root flex flex-col h-screen w-screen bg-zinc-950 text-zinc-100 overflow-hidden font-sans">
    <!-- Top Header Bar -->
    <HeaderBar
      :db-name="dbInfo?.name"
      :system-bases="systemBases"
      :piece-sets="pieceSets"
      :board-themes="boardThemes"
      :current-piece-set="currentPieceSet"
      :current-theme-id="currentThemeId"
      :flipped="flipped"
      @open-d-b="handleOpenDB"
      @create-d-b="handleCreateDB"
      @open-system-base="handleOpenSystemBase"
      @import-p-g-n="handleImportPGN"
      @export-p-g-n="handleExportPGN"
      @change-piece-set="currentPieceSet = $event"
      @change-theme="currentThemeId = $event"
      @toggle-flip="flipped = !flipped"
    />

    <!-- Main Workspace (Split Grid) -->
    <div class="workspace-grid flex-1 flex flex-col md:flex-row overflow-hidden p-2 gap-2">
      <!-- Left Column: Chessboard & Engine Control -->
      <div class="left-col flex flex-col items-center justify-between gap-2 max-w-[620px] w-full shrink-0">
        <!-- Interactive Chessboard -->
        <Chessboard
          :fen="currentFEN"
          :legal-moves="legalMoves"
          :piece-set="currentPieceSet"
          :board-theme="currentTheme"
          :flipped="flipped"
          :last-move="lastMove"
          :in-check="inCheck"
          @move="handleMove"
        />

        <!-- Live Engine Analysis Bar -->
        <div class="w-full">
          <EnginePanel
            :eval-info="evalInfo"
            :is-analyzing="isAnalyzing"
            :available-engines="availableEngines"
            :selected-engine="selectedEngine"
            @toggle-analysis="toggleAnalysis"
            @select-engine="handleSelectEngine"
          />
        </div>
      </div>

      <!-- Right Column: Tabbed Panels (Moves, Opening Tree, Game Info) -->
      <div class="right-col flex-1 flex flex-col overflow-hidden bg-zinc-900 border border-zinc-800 rounded-xl shadow-lg">
        <!-- Tabs Header -->
        <div class="tabs-header flex items-center justify-between p-2 bg-zinc-950 border-b border-zinc-800">
          <div class="flex items-center gap-1">
            <button
              class="px-3 py-1.5 rounded-lg text-xs font-semibold transition-all cursor-pointer"
              :class="activeTab === 'notation'
                ? 'bg-emerald-600 text-white shadow-sm'
                : 'text-zinc-400 hover:text-zinc-200 hover:bg-zinc-800'"
              @click="activeTab = 'notation'"
            >
              Notation & Moves
            </button>
            <button
              class="px-3 py-1.5 rounded-lg text-xs font-semibold transition-all cursor-pointer"
              :class="activeTab === 'tree'
                ? 'bg-emerald-600 text-white shadow-sm'
                : 'text-zinc-400 hover:text-zinc-200 hover:bg-zinc-800'"
              @click="activeTab = 'tree'"
            >
              Opening Tree
            </button>
            <button
              class="px-3 py-1.5 rounded-lg text-xs font-semibold transition-all cursor-pointer"
              :class="activeTab === 'info'
                ? 'bg-emerald-600 text-white shadow-sm'
                : 'text-zinc-400 hover:text-zinc-200 hover:bg-zinc-800'"
              @click="activeTab = 'info'"
            >
              Game Info
            </button>
          </div>

          <!-- Bottom Drawer Toggle -->
          <button
            class="text-xs text-zinc-400 hover:text-zinc-200 px-2 py-1 rounded bg-zinc-800/80 hover:bg-zinc-800 transition-colors"
            @click="bottomDrawerOpen = !bottomDrawerOpen"
          >
            {{ bottomDrawerOpen ? 'Hide Database ▼' : 'Show Database ▲' }}
          </button>
        </div>

        <!-- Tab Content -->
        <div class="tab-content flex-1 overflow-hidden p-2">
          <!-- 1. Notation View -->
          <div v-show="activeTab === 'notation'" class="h-full">
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
            />
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
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Bottom Panel: Database Games Browser (Collapsible) -->
    <div
      v-if="bottomDrawerOpen"
      class="bottom-drawer h-[240px] shrink-0 px-2 pb-2 transition-all duration-200"
    >
      <GameBrowser
        :games="gamesList"
        :total-games="totalGames"
        :current-page="currentPage"
        :page-size="pageSize"
        :active-game-id="activeGameId"
        :db-name="dbInfo?.name"
        @load-game="handleLoadGame"
        @page-change="handlePageChange"
        @search="handleSearch"
        @new-game="handleNewGame"
        @save-game="handleSaveCurrentGame"
        @delete-game="handleDeleteGame"
      />
    </div>

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
            class="px-3 py-1.5 rounded-lg bg-zinc-800 text-zinc-300 text-xs font-medium hover:bg-zinc-700"
            @click="showImportModal = false"
          >
            Cancel
          </button>
          <button
            class="px-4 py-1.5 rounded-lg bg-emerald-600 text-white text-xs font-medium hover:bg-emerald-500"
            @click="confirmImportPGN"
          >
            Load Game
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
