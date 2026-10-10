<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import {
  FileText,
  FolderOpen,
  Save,
  Trash2,
  Undo2,
  Redo2,
  Sliders,
  Play,
  Award,
  Trophy,
  Search,
  Filter,
  Wrench,
  HelpCircle,
  Cpu,
  Palette,
  Shuffle,
  ChevronRight,
  Sparkles,
  Swords,
  Copy,
  ClipboardPaste,
  RotateCcw,
  BookOpen,
  Zap,
  Pickaxe,
  Info,
  Globe,
  Bot,
  Check,
  Users,
  ExternalLink,
  BarChart3,
  FileEdit,
  Download
} from '@lucide/vue'
import { PieceSetOption, BoardThemeOption } from './HeaderBar.vue'
import { useNotation, type NotationStyle } from '../utils/notation'

const { currentStyle, setNotationStyle, options: notationOptions } = useNotation()

const props = defineProps<{
  dbName?: string
  pieceSets: PieceSetOption[]
  boardThemes: BoardThemeOption[]
  currentPieceSet: string
  currentThemeId: string
  trialMode: boolean
  isAnalyzing: boolean
}>()

const emit = defineEmits<{
  (e: 'newGame'): void
  (e: 'openDB'): void
  (e: 'createDB'): void
  (e: 'closeDB'): void
  (e: 'saveGame'): void
  (e: 'deleteGame'): void
  (e: 'firstGame'): void
  (e: 'prevGame'): void
  (e: 'nextGame'): void
  (e: 'lastGame'): void
  (e: 'randomGame'): void
  (e: 'importPGN'): void
  (e: 'exportPGN'): void
  (e: 'copyFEN'): void
  (e: 'pasteFEN'): void
  (e: 'setupBoard'): void
  (e: 'toggleTrialMode'): void
  (e: 'undoMove'): void
  (e: 'redoMove'): void
  (e: 'toggleAnalysis'): void
  (e: 'annotateGame'): void
  (e: 'playVsEngine'): void
  (e: 'tacticsTrainer'): void
  (e: 'openingTrainer'): void
  (e: 'progressiveTactics'): void
  (e: 'minePuzzles'): void
  (e: 'searchHeader'): void
  (e: 'searchPosition'): void
  (e: 'toggleFlip'): void
  (e: 'changePieceSet', id: string): void
  (e: 'changeTheme', id: string): void
  (e: 'openSettings'): void
  (e: 'showShortcuts'): void
  (e: 'showAbout'): void
  (e: 'showHelp'): void
  (e: 'openWebDocs'): void
  (e: 'databaseMaintenance'): void
  (e: 'onlineSync'): void
  (e: 'simulatePlayer'): void
  (e: 'playerPersonas'): void
  (e: 'showCrosstable'): void
  (e: 'ecoBrowser'): void
  (e: 'guessTheMove'): void
  (e: 'databaseStats'): void
  (e: 'editTags'): void
  (e: 'twicDownloader'): void
  (e: 'engineMatch'): void
  (e: 'exportGame'): void
  (e: 'standardizeNames'): void
}>()


const activeMenu = ref<string | null>(null)
let justSwitchedMenu = false

function toggleMenu(menu: string) {
  if (activeMenu.value === menu) {
    if (justSwitchedMenu) {
      justSwitchedMenu = false
      return
    }
    activeMenu.value = null
  } else {
    activeMenu.value = menu
    justSwitchedMenu = false
  }
}

function handleMenuHover(menu: string) {
  if (activeMenu.value !== null && activeMenu.value !== menu) {
    activeMenu.value = menu
    justSwitchedMenu = true
  }
}

function closeAll() {
  activeMenu.value = null
  justSwitchedMenu = false
}

function selectItem(fn: () => void) {
  closeAll()
  fn()
}

function handleGlobalClick(e: MouseEvent) {
  const target = e.target as HTMLElement
  if (!target || typeof target.closest !== 'function') return
  if (!target.closest('.app-menubar')) {
    closeAll()
  }
}

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape' && activeMenu.value !== null) {
    closeAll()
  }
}

onMounted(() => {
  window.addEventListener('click', handleGlobalClick)
  window.addEventListener('keydown', handleKeydown)
})

onUnmounted(() => {
  window.removeEventListener('click', handleGlobalClick)
  window.removeEventListener('keydown', handleKeydown)
})
</script>

<template>
  <div class="app-menubar relative z-40 bg-zinc-950 border-b border-zinc-800/80 px-2 py-0.5 flex items-center gap-1 text-xs select-none">
    <!-- FILE MENU -->
    <div class="relative">
      <button
        class="px-2.5 py-1 rounded hover:bg-zinc-800 text-zinc-300 hover:text-white transition-colors cursor-pointer"
        :class="{ 'bg-zinc-800 text-white': activeMenu === 'file' }"
        @click="toggleMenu('file')"
        @mouseenter="handleMenuHover('file')"
      >
        File
      </button>

      <div
        v-if="activeMenu === 'file'"
        class="absolute top-full left-0 mt-0.5 w-56 bg-zinc-900 border border-zinc-700/80 rounded-lg shadow-2xl py-1 z-50 text-xs"
      >
        <button
          class="w-full text-left px-3 py-1.5 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between cursor-pointer"
          @click="selectItem(() => emit('openDB'))"
        >
          <span class="flex items-center gap-2"><FolderOpen class="w-3.5 h-3.5 text-emerald-400" /> Open Database...</span>
          <span class="text-[10px] text-zinc-500 font-mono">Ctrl+O</span>
        </button>
        <button
          class="w-full text-left px-3 py-1.5 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between cursor-pointer"
          @click="selectItem(() => emit('createDB'))"
        >
          <span class="flex items-center gap-2"><FileText class="w-3.5 h-3.5 text-zinc-400" /> New Database...</span>
        </button>
        <button
          v-if="dbName"
          class="w-full text-left px-3 py-1.5 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between cursor-pointer"
          @click="selectItem(() => emit('closeDB'))"
        >
          <span class="flex items-center gap-2"><Trash2 class="w-3.5 h-3.5 text-amber-400" /> Close Database</span>
        </button>
        <button
          v-if="dbName"
          class="w-full text-left px-3 py-1.5 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between cursor-pointer"
          @click="selectItem(() => emit('databaseMaintenance'))"
        >
          <span class="flex items-center gap-2"><Wrench class="w-3.5 h-3.5 text-emerald-400" /> Database Maintenance...</span>
          <span class="text-[10px] text-zinc-500 font-mono">Ctrl+M</span>
        </button>
        <button
          class="w-full text-left px-3 py-1.5 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between cursor-pointer"
          @click="selectItem(() => emit('onlineSync'))"
        >
          <span class="flex items-center gap-2"><Globe class="w-3.5 h-3.5 text-cyan-400" /> Online Account Sync...</span>
          <span class="text-[10px] text-zinc-500 font-mono">Chess.com/Lichess</span>
        </button>

        <div class="h-px bg-zinc-800 my-1" />

        <button
          class="w-full text-left px-3 py-1.5 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between cursor-pointer"
          @click="selectItem(() => emit('importPGN'))"
        >
          <span class="flex items-center gap-2"><ClipboardPaste class="w-3.5 h-3.5 text-zinc-400" /> Import PGN Game...</span>
          <span class="text-[10px] text-zinc-500 font-mono">Ctrl+I</span>
        </button>
        <button
          class="w-full text-left px-3 py-1.5 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between cursor-pointer"
          @click="selectItem(() => emit('exportPGN'))"
        >
          <span class="flex items-center gap-2"><Copy class="w-3.5 h-3.5 text-zinc-400" /> Export PGN Game</span>
          <span class="text-[10px] text-zinc-500 font-mono">Ctrl+E</span>
        </button>
        <button
          class="w-full text-left px-3 py-1.5 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between cursor-pointer"
          @click="selectItem(() => emit('exportGame'))"
        >
          <span class="flex items-center gap-2"><Download class="w-3.5 h-3.5 text-emerald-400" /> Export Game (HTML / LaTeX)...</span>
          <span class="text-[10px] text-zinc-500 font-mono">Ctrl+Shift+E</span>
        </button>
        <button
          class="w-full text-left px-3 py-1.5 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between cursor-pointer"
          @click="selectItem(() => emit('copyFEN'))"
        >
          <span class="flex items-center gap-2"><Copy class="w-3.5 h-3.5 text-zinc-400" /> Copy FEN to Clipboard</span>
          <span class="text-[10px] text-zinc-500 font-mono">Ctrl+C</span>
        </button>
        <button
          class="w-full text-left px-3 py-1.5 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between cursor-pointer"
          @click="selectItem(() => emit('pasteFEN'))"
        >
          <span class="flex items-center gap-2"><ClipboardPaste class="w-3.5 h-3.5 text-zinc-400" /> Paste FEN from Clipboard</span>
          <span class="text-[10px] text-zinc-500 font-mono">Ctrl+V</span>
        </button>
      </div>
    </div>

    <!-- EDIT MENU -->
    <div class="relative">
      <button
        class="px-2.5 py-1 rounded hover:bg-zinc-800 text-zinc-300 hover:text-white transition-colors cursor-pointer"
        :class="{ 'bg-zinc-800 text-white': activeMenu === 'edit' }"
        @click="toggleMenu('edit')"
        @mouseenter="handleMenuHover('edit')"
      >
        Edit
      </button>

      <div
        v-if="activeMenu === 'edit'"
        class="absolute top-full left-0 mt-0.5 w-60 bg-zinc-900 border border-zinc-700/80 rounded-lg shadow-2xl py-1 z-50 text-xs"
      >
        <button
          class="w-full text-left px-3 py-1.5 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between cursor-pointer"
          @click="selectItem(() => emit('setupBoard'))"
        >
          <span class="flex items-center gap-2"><Sliders class="w-3.5 h-3.5 text-purple-400" /> Setup Board / FEN Editor...</span>
          <span class="text-[10px] text-zinc-500 font-mono">Ctrl+B</span>
        </button>
        <button
          class="w-full text-left px-3 py-1.5 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between cursor-pointer"
          @click="selectItem(() => emit('toggleTrialMode'))"
        >
          <span class="flex items-center gap-2">
            <span class="w-2 h-2 rounded-full" :class="trialMode ? 'bg-amber-400' : 'bg-zinc-600'" />
            Trial Mode {{ trialMode ? '(Active)' : '(Off)' }}
          </span>
          <span class="text-[10px] text-zinc-500 font-mono">Ctrl+Space</span>
        </button>

        <div class="h-px bg-zinc-800 my-1" />

        <button
          class="w-full text-left px-3 py-1.5 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between cursor-pointer"
          @click="selectItem(() => emit('undoMove'))"
        >
          <span class="flex items-center gap-2"><Undo2 class="w-3.5 h-3.5 text-zinc-400" /> Undo Move</span>
          <span class="text-[10px] text-zinc-500 font-mono">Ctrl+Z</span>
        </button>
        <button
          class="w-full text-left px-3 py-1.5 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between cursor-pointer"
          @click="selectItem(() => emit('redoMove'))"
        >
          <span class="flex items-center gap-2"><Redo2 class="w-3.5 h-3.5 text-zinc-400" /> Redo Move</span>
          <span class="text-[10px] text-zinc-500 font-mono">Ctrl+Y</span>
        </button>
      </div>
    </div>

    <!-- GAME MENU -->
    <div class="relative">
      <button
        class="px-2.5 py-1 rounded hover:bg-zinc-800 text-zinc-300 hover:text-white transition-colors cursor-pointer"
        :class="{ 'bg-zinc-800 text-white': activeMenu === 'game' }"
        @click="toggleMenu('game')"
        @mouseenter="handleMenuHover('game')"
      >
        Game
      </button>

      <div
        v-if="activeMenu === 'game'"
        class="absolute top-full left-0 mt-0.5 w-56 bg-zinc-900 border border-zinc-700/80 rounded-lg shadow-2xl py-1 z-50 text-xs"
      >
        <button
          class="w-full text-left px-3 py-1.5 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between cursor-pointer"
          @click="selectItem(() => emit('newGame'))"
        >
          <span class="flex items-center gap-2"><FileText class="w-3.5 h-3.5 text-emerald-400" /> New Game</span>
          <span class="text-[10px] text-zinc-500 font-mono">Ctrl+N</span>
        </button>
        <button
          class="w-full text-left px-3 py-1.5 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between cursor-pointer"
          @click="selectItem(() => emit('saveGame'))"
        >
          <span class="flex items-center gap-2"><Save class="w-3.5 h-3.5 text-blue-400" /> Save Current Game</span>
          <span class="text-[10px] text-zinc-500 font-mono">Ctrl+S</span>
        </button>
        <button
          class="w-full text-left px-3 py-1.5 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between cursor-pointer"
          @click="selectItem(() => emit('editTags'))"
        >
          <span class="flex items-center gap-2"><FileEdit class="w-3.5 h-3.5 text-emerald-400" /> Edit Game Headers...</span>
          <span class="text-[10px] text-zinc-500 font-mono">Ctrl+T</span>
        </button>
        <button
          class="w-full text-left px-3 py-1.5 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between cursor-pointer"
          @click="selectItem(() => emit('deleteGame'))"
        >
          <span class="flex items-center gap-2"><Trash2 class="w-3.5 h-3.5 text-red-400" /> Delete Game</span>
          <span class="text-[10px] text-zinc-500 font-mono">Ctrl+Del</span>
        </button>

        <div class="h-px bg-zinc-800 my-1" />

        <button
          class="w-full text-left px-3 py-1.5 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between cursor-pointer"
          @click="selectItem(() => emit('firstGame'))"
        >
          <span>First Game</span>
          <span class="text-[10px] text-zinc-500 font-mono">Ctrl+Home</span>
        </button>
        <button
          class="w-full text-left px-3 py-1.5 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between cursor-pointer"
          @click="selectItem(() => emit('prevGame'))"
        >
          <span>Previous Game</span>
          <span class="text-[10px] text-zinc-500 font-mono">Ctrl+Up</span>
        </button>
        <button
          class="w-full text-left px-3 py-1.5 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between cursor-pointer"
          @click="selectItem(() => emit('nextGame'))"
        >
          <span>Next Game</span>
          <span class="text-[10px] text-zinc-500 font-mono">Ctrl+Down</span>
        </button>
        <button
          class="w-full text-left px-3 py-1.5 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between cursor-pointer"
          @click="selectItem(() => emit('lastGame'))"
        >
          <span>Last Game</span>
          <span class="text-[10px] text-zinc-500 font-mono">Ctrl+End</span>
        </button>
        <button
          class="w-full text-left px-3 py-1.5 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between cursor-pointer"
          @click="selectItem(() => emit('randomGame'))"
        >
          <span class="flex items-center gap-2"><Shuffle class="w-3.5 h-3.5 text-zinc-400" /> Random Game</span>
          <span class="text-[10px] text-zinc-500 font-mono">Ctrl+?</span>
        </button>
      </div>
    </div>

    <!-- SEARCH MENU -->
    <div class="relative">
      <button
        class="px-2.5 py-1 rounded hover:bg-zinc-800 text-zinc-300 hover:text-white transition-colors cursor-pointer"
        :class="{ 'bg-zinc-800 text-white': activeMenu === 'search' }"
        @click="toggleMenu('search')"
        @mouseenter="handleMenuHover('search')"
      >
        Search
      </button>

      <div
        v-if="activeMenu === 'search'"
        class="absolute top-full left-0 mt-0.5 w-64 bg-zinc-900 border border-zinc-700/80 rounded-lg shadow-2xl py-1 z-50 text-xs"
      >
        <button
          class="w-full text-left px-3 py-1.5 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between cursor-pointer"
          @click="selectItem(() => emit('searchHeader'))"
        >
          <span class="flex items-center gap-2"><Filter class="w-3.5 h-3.5 text-blue-400" /> Header / General Search...</span>
          <span class="text-[10px] text-zinc-500 font-mono">Ctrl+H</span>
        </button>
        <button
          class="w-full text-left px-3 py-1.5 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between cursor-pointer"
          @click="selectItem(() => emit('searchPosition'))"
        >
          <span class="flex items-center gap-2"><Search class="w-3.5 h-3.5 text-emerald-400" /> Search Current Position</span>
        </button>
      </div>
    </div>

    <!-- TOOLS MENU -->
    <div class="relative">
      <button
        class="px-2.5 py-1 rounded hover:bg-zinc-800 text-zinc-300 hover:text-white transition-colors cursor-pointer"
        :class="{ 'bg-zinc-800 text-white': activeMenu === 'tools' }"
        @click="toggleMenu('tools')"
        @mouseenter="handleMenuHover('tools')"
      >
        Tools
      </button>

      <div
        v-if="activeMenu === 'tools'"
        class="absolute top-full left-0 mt-0.5 w-64 bg-zinc-900 border border-zinc-700/80 rounded-lg shadow-2xl py-1 z-50 text-xs"
      >
        <button
          class="w-full text-left px-3 py-1.5 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between cursor-pointer"
          @click="selectItem(() => emit('annotateGame'))"
        >
          <span class="flex items-center gap-2"><Sparkles class="w-3.5 h-3.5 text-amber-400" /> Full Game Analysis (Annotate)</span>
        </button>
        <button
          class="w-full text-left px-3 py-1.5 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between cursor-pointer"
          @click="selectItem(() => emit('toggleAnalysis'))"
        >
          <span class="flex items-center gap-2"><Cpu class="w-3.5 h-3.5 text-emerald-400" /> {{ isAnalyzing ? 'Stop Engine Analysis' : 'Start Live Analysis' }}</span>
          <span class="text-[10px] text-zinc-500 font-mono">F2</span>
        </button>
        <div class="h-px bg-zinc-800 my-1" />
        <button
          class="w-full text-left px-3 py-1.5 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between cursor-pointer"
          @click="selectItem(() => emit('minePuzzles'))"
        >
          <span class="flex items-center gap-2"><Pickaxe class="w-3.5 h-3.5 text-amber-400" /> Mine Puzzles from Database Games...</span>
        </button>
        <button
          v-if="dbName"
          class="w-full text-left px-3 py-1.5 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between cursor-pointer"
          @click="selectItem(() => emit('databaseMaintenance'))"
        >
          <span class="flex items-center gap-2"><Wrench class="w-3.5 h-3.5 text-emerald-400" /> Database Maintenance & Tools...</span>
        </button>
        <button
          v-if="dbName"
          class="w-full text-left px-3 py-1.5 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between cursor-pointer"
          @click="selectItem(() => emit('standardizeNames'))"
        >
          <span class="flex items-center gap-2"><Sparkles class="w-3.5 h-3.5 text-purple-400" /> Standardize Player Names (Spellcheck)...</span>
        </button>
        <button
          class="w-full text-left px-3 py-1.5 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between cursor-pointer"
          @click="selectItem(() => emit('onlineSync'))"
        >
          <span class="flex items-center gap-2"><Globe class="w-3.5 h-3.5 text-cyan-400" /> Online Account Sync (Chess.com & Lichess)...</span>
        </button>
        <button
          class="w-full text-left px-3 py-1.5 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between cursor-pointer"
          @click="selectItem(() => emit('twicDownloader'))"
        >
          <span class="flex items-center gap-2"><Download class="w-3.5 h-3.5 text-cyan-400" /> Download TWIC Bulletins...</span>
        </button>
        <button
          class="w-full text-left px-3 py-1.5 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between cursor-pointer"
          @click="selectItem(() => emit('playerPersonas'))"
        >
          <span class="flex items-center gap-2"><Users class="w-3.5 h-3.5 text-purple-400" /> Player Personas & Aliases...</span>
        </button>
        <button
          class="w-full text-left px-3 py-1.5 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between cursor-pointer"
          @click="selectItem(() => emit('showCrosstable'))"
        >
          <span class="flex items-center gap-2"><Trophy class="w-3.5 h-3.5 text-amber-400" /> Tournament Crosstables & Standings...</span>
          <span class="text-[10px] text-zinc-500 font-mono">F7</span>
        </button>
        <button
          class="w-full text-left px-3 py-1.5 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between cursor-pointer"
          @click="selectItem(() => emit('ecoBrowser'))"
        >
          <span class="flex items-center gap-2"><BookOpen class="w-3.5 h-3.5 text-emerald-400" /> ECO Opening Classification Browser...</span>
          <span class="text-[10px] text-zinc-500 font-mono">F6</span>
        </button>
        <button
          class="w-full text-left px-3 py-1.5 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between cursor-pointer"
          @click="selectItem(() => emit('databaseStats'))"
        >
          <span class="flex items-center gap-2"><BarChart3 class="w-3.5 h-3.5 text-emerald-400" /> Database Demographics & Rating Tracker...</span>
          <span class="text-[10px] text-zinc-500 font-mono">Ctrl+D</span>
        </button>
      </div>
    </div>

    <!-- PLAY MENU -->
    <div class="relative">
      <button
        class="px-2.5 py-1 rounded hover:bg-zinc-800 text-zinc-300 hover:text-white transition-colors cursor-pointer"
        :class="{ 'bg-zinc-800 text-white': activeMenu === 'play' }"
        @click="toggleMenu('play')"
        @mouseenter="handleMenuHover('play')"
      >
        Play
      </button>

      <div
        v-if="activeMenu === 'play'"
        class="absolute top-full left-0 mt-0.5 w-68 bg-zinc-900 border border-zinc-700/80 rounded-lg shadow-2xl py-1 z-50 text-xs"
      >
        <button
          class="w-full text-left px-3 py-1.5 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between cursor-pointer"
          @click="selectItem(() => emit('playVsEngine'))"
        >
          <span class="flex items-center gap-2"><Swords class="w-3.5 h-3.5 text-amber-400" /> Play Against Engine...</span>
        </button>
        <button
          class="w-full text-left px-3 py-1.5 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between cursor-pointer"
          @click="selectItem(() => emit('engineMatch'))"
        >
          <span class="flex items-center gap-2"><Swords class="w-3.5 h-3.5 text-cyan-400" /> Engine vs Engine Match...</span>
        </button>
        <button
          class="w-full text-left px-3 py-1.5 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between cursor-pointer"
          @click="selectItem(() => emit('simulatePlayer'))"
        >
          <span class="flex items-center gap-2"><Bot class="w-3.5 h-3.5 text-purple-400" /> Simulate Database Player (Sparring)...</span>
        </button>
        <button
          class="w-full text-left px-3 py-1.5 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between cursor-pointer"
          @click="selectItem(() => emit('playerPersonas'))"
        >
          <span class="flex items-center gap-2"><Users class="w-3.5 h-3.5 text-emerald-400" /> Player Personas (Spar & Aggregate)...</span>
        </button>
        <button
          class="w-full text-left px-3 py-1.5 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between cursor-pointer"
          @click="selectItem(() => emit('tacticsTrainer'))"
        >
          <span class="flex items-center gap-2"><Award class="w-3.5 h-3.5 text-blue-400" /> Tactics Puzzle Trainer...</span>
        </button>
        <button
          class="w-full text-left px-3 py-1.5 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between cursor-pointer"
          @click="selectItem(() => emit('guessTheMove'))"
        >
          <span class="flex items-center gap-2"><Award class="w-3.5 h-3.5 text-amber-400" /> Guess the Move (Solitaire Chess)...</span>
          <span class="text-[10px] text-zinc-500 font-mono">Ctrl+G</span>
        </button>
        <div class="h-px bg-zinc-800 my-1" />
        <button
          class="w-full text-left px-3 py-1.5 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between cursor-pointer"
          @click="selectItem(() => emit('progressiveTactics'))"
        >
          <span class="flex items-center gap-2"><Zap class="w-3.5 h-3.5 text-amber-400" /> Progressive Tactics Challenge...</span>
        </button>
        <button
          class="w-full text-left px-3 py-1.5 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between cursor-pointer"
          @click="selectItem(() => emit('openingTrainer'))"
        >
          <span class="flex items-center gap-2"><BookOpen class="w-3.5 h-3.5 text-emerald-400" /> Opening Builder & Trainer...</span>
        </button>
      </div>
    </div>


    <!-- OPTIONS MENU -->
    <div class="relative">
      <button
        class="px-2.5 py-1 rounded hover:bg-zinc-800 text-zinc-300 hover:text-white transition-colors cursor-pointer"
        :class="{ 'bg-zinc-800 text-white': activeMenu === 'options' }"
        @click="toggleMenu('options')"
        @mouseenter="handleMenuHover('options')"
      >
        Options
      </button>

      <div
        v-if="activeMenu === 'options'"
        class="absolute top-full left-0 mt-0.5 w-64 bg-zinc-900 border border-zinc-700/80 rounded-lg shadow-2xl py-1 z-50 text-xs"
      >
        <button
          class="w-full text-left px-3 py-1.5 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between cursor-pointer"
          @click="selectItem(() => emit('toggleFlip'))"
        >
          <span class="flex items-center gap-2"><RotateCcw class="w-3.5 h-3.5 text-zinc-400" /> Flip Board</span>
          <span class="text-[10px] text-zinc-500 font-mono">F</span>
        </button>
        <div class="h-px bg-zinc-800 my-1" />
        <button
          class="w-full text-left px-3 py-1.5 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between cursor-pointer"
          @click="selectItem(() => emit('openSettings'))"
        >
          <span class="flex items-center gap-2"><Palette class="w-3.5 h-3.5 text-emerald-400" /> Board Themes & Piece Sets...</span>
        </button>
        <div class="h-px bg-zinc-800 my-1" />
        <div class="px-3 py-1 text-[10px] text-zinc-500 font-bold uppercase tracking-wider">Notation Style</div>
        <button
          v-for="opt in notationOptions"
          :key="opt.id"
          class="w-full text-left px-3 py-1 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between cursor-pointer"
          @click="selectItem(() => setNotationStyle(opt.id))"
        >
          <span class="flex items-center gap-2">
            <span :class="currentStyle === opt.id ? 'text-amber-400 font-bold' : 'text-zinc-400'">{{ opt.name }}</span>
          </span>
          <Check v-if="currentStyle === opt.id" class="w-3 h-3 text-amber-400" />
        </button>
      </div>
    </div>

    <!-- HELP MENU -->
    <div class="relative">
      <button
        class="px-2.5 py-1 rounded hover:bg-zinc-800 text-zinc-300 hover:text-white transition-colors cursor-pointer"
        :class="{ 'bg-zinc-800 text-white': activeMenu === 'help' }"
        @click="toggleMenu('help')"
        @mouseenter="handleMenuHover('help')"
      >
        Help
      </button>

      <div
        v-if="activeMenu === 'help'"
        class="absolute top-full left-0 mt-0.5 w-60 bg-zinc-900 border border-zinc-700/80 rounded-lg shadow-2xl py-1 z-50 text-xs"
      >
        <button
          class="w-full text-left px-3 py-1.5 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between cursor-pointer"
          @click="selectItem(() => emit('showHelp'))"
        >
          <span class="flex items-center gap-2"><BookOpen class="w-3.5 h-3.5 text-emerald-400" /> Online Help & Guides...</span>
          <span class="text-[10px] text-zinc-500 font-mono">F1</span>
        </button>
        <button
          class="w-full text-left px-3 py-1.5 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between cursor-pointer"
          @click="selectItem(() => emit('openWebDocs'))"
        >
          <span class="flex items-center gap-2"><ExternalLink class="w-3.5 h-3.5 text-cyan-400" /> Web Documentation (Online)...</span>
        </button>
        <div class="h-px bg-zinc-800 my-1" />
        <button
          class="w-full text-left px-3 py-1.5 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between cursor-pointer"
          @click="selectItem(() => emit('showShortcuts'))"
        >
          <span class="flex items-center gap-2"><HelpCircle class="w-3.5 h-3.5 text-emerald-400" /> Keyboard Shortcuts</span>
          <span class="text-[10px] text-zinc-500 font-mono">?</span>
        </button>
        <div class="h-px bg-zinc-800 my-1" />
        <button
          class="w-full text-left px-3 py-1.5 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between cursor-pointer"
          @click="selectItem(() => emit('showAbout'))"
        >
          <span class="flex items-center gap-2"><Info class="w-3.5 h-3.5 text-zinc-400" /> About NewScid</span>
        </button>
      </div>
    </div>

    <!-- TRIAL MODE BADGE INDICATOR -->
    <div v-if="trialMode" class="ml-auto flex items-center gap-1.5 px-2.5 py-0.5 rounded-full bg-amber-500/20 border border-amber-500/40 text-amber-300 text-[11px] font-semibold animate-pulse">
      <span class="w-1.5 h-1.5 rounded-full bg-amber-400" />
      <span>Trial Mode Active</span>
      <button
        class="ml-1 text-[10px] underline hover:text-white cursor-pointer"
        @click="emit('toggleTrialMode')"
      >
        Exit
      </button>
    </div>
  </div>
</template>
