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
  ChevronLeft,
  ChevronsRight,
  ChevronsLeft,
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
  Download,
  Eye,
  EyeOff,
  Volume2,
  VolumeX,
  Dices,
  Target,
  Shield,
  Maximize2,
  Brain,
  Sun,
  Moon,
  Crosshair,
  Clock,
  Scissors,
  Heart
} from '@lucide/vue'
import { PieceSetOption, BoardThemeOption } from './HeaderBar.vue'
import { useNotation, type NotationStyle } from '../utils/notation'
import { UI_THEMES } from '../utils/theme'

const { currentStyle, setNotationStyle, options: notationOptions } = useNotation()

const props = defineProps<{
  dbName?: string
  pieceSets: PieceSetOption[]
  boardThemes: BoardThemeOption[]
  currentPieceSet: string
  currentThemeId: string
  currentUITheme?: string
  trialMode: boolean
  isAnalyzing: boolean
  soundEnabled?: boolean
  blindfold?: boolean
  threatMode?: boolean
  showEvalBar?: boolean
  zenMode?: boolean
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
  (e: 'toggleSound'): void
  (e: 'toggleBlindfold'): void
  (e: 'toggleThreatMode'): void
  (e: 'toggleEvalBar'): void
  (e: 'chess960'): void
  (e: 'coordinateTrainer'): void
  (e: 'commandPalette'): void
  (e: 'blunderVault'): void
  (e: 'opponentScout'): void
  (e: 'antiPersona'): void
  (e: 'toggleZenMode'): void
  (e: 'explainMove'): void
  (e: 'openTreeWindow'): void
  (e: 'changeUITheme', id: string): void
  (e: 'toggleDarkLight'): void
  (e: 'pieceSafety'): void
  (e: 'intuitionStorm'): void
  (e: 'timeManagement'): void
  (e: 'repertoireShortener'): void
  (e: 'prophylaxis'): void
  (e: 'showSupport'): void
  (e: 'megaDownloader'): void
  (e: 'fairPlayInspector'): void
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
  <div class="app-menubar relative z-40 bg-zinc-950 border-b border-zinc-800/80 px-2.5 py-1 flex items-center gap-1 text-xs select-none">
    <!-- FILE MENU -->
    <div class="relative">
      <button
        class="px-3 py-1.5 rounded-md hover:bg-zinc-800 text-zinc-300 hover:text-white transition-colors cursor-pointer font-medium"
        :class="{ 'bg-zinc-800 text-white shadow-xs': activeMenu === 'file' }"
        @click="toggleMenu('file')"
        @mouseenter="handleMenuHover('file')"
      >
        File
      </button>

      <div
        v-if="activeMenu === 'file'"
        class="absolute top-full left-0 mt-1 min-w-[320px] w-max bg-zinc-900 border border-zinc-700/80 rounded-xl shadow-2xl py-1.5 z-50 text-xs"
      >
        <button
          class="w-full text-left px-3.5 py-2 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => emit('openDB'))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap"><FolderOpen class="w-4 h-4 shrink-0 text-emerald-400" /> Open Database...</span>
          <span class="text-[11px] text-zinc-500 font-mono tracking-wider shrink-0">Ctrl+O</span>
        </button>
        <button
          class="w-full text-left px-3.5 py-2 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => emit('createDB'))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap"><FileText class="w-4 h-4 shrink-0 text-zinc-400" /> New Database...</span>
        </button>
        <button
          v-if="dbName"
          class="w-full text-left px-3.5 py-2 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => emit('closeDB'))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap"><Trash2 class="w-4 h-4 shrink-0 text-amber-400" /> Close Database</span>
        </button>
        <button
          v-if="dbName"
          class="w-full text-left px-3.5 py-2 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => emit('databaseMaintenance'))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap"><Wrench class="w-4 h-4 shrink-0 text-emerald-400" /> Database Maintenance...</span>
          <span class="text-[11px] text-zinc-500 font-mono tracking-wider shrink-0">Ctrl+M</span>
        </button>
        <button
          class="w-full text-left px-3.5 py-2 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => emit('onlineSync'))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap"><Globe class="w-4 h-4 shrink-0 text-cyan-400" /> Online Account Sync...</span>
          <span class="text-[11px] text-zinc-500 font-mono tracking-wider shrink-0">Chess.com / Lichess</span>
        </button>
        <button
          class="w-full text-left px-3.5 py-2 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => emit('megaDownloader'))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap"><Download class="w-4 h-4 shrink-0 text-emerald-400" /> Download Master &amp; Mega Databases...</span>
          <span class="text-[11px] text-emerald-400 font-mono tracking-wider shrink-0">Lumbra / Lichess</span>
        </button>

        <div class="h-px bg-zinc-800 my-1 mx-2" />

        <button
          class="w-full text-left px-3.5 py-2 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => emit('importPGN'))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap"><ClipboardPaste class="w-4 h-4 shrink-0 text-zinc-400" /> Import PGN Game...</span>
          <span class="text-[11px] text-zinc-500 font-mono tracking-wider shrink-0">Ctrl+I</span>
        </button>
        <button
          class="w-full text-left px-3.5 py-2 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => emit('exportPGN'))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap"><Copy class="w-4 h-4 shrink-0 text-zinc-400" /> Export PGN Game</span>
          <span class="text-[11px] text-zinc-500 font-mono tracking-wider shrink-0">Ctrl+E</span>
        </button>
        <button
          class="w-full text-left px-3.5 py-2 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => emit('exportGame'))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap"><Download class="w-4 h-4 shrink-0 text-emerald-400" /> Export Game (HTML / LaTeX)...</span>
          <span class="text-[11px] text-zinc-500 font-mono tracking-wider shrink-0">Ctrl+Shift+E</span>
        </button>
        <button
          class="w-full text-left px-3.5 py-2 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => emit('copyFEN'))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap"><Copy class="w-4 h-4 shrink-0 text-zinc-400" /> Copy FEN to Clipboard</span>
          <span class="text-[11px] text-zinc-500 font-mono tracking-wider shrink-0">Ctrl+C</span>
        </button>
        <button
          class="w-full text-left px-3.5 py-2 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => emit('pasteFEN'))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap"><ClipboardPaste class="w-4 h-4 shrink-0 text-zinc-400" /> Paste FEN from Clipboard</span>
          <span class="text-[11px] text-zinc-500 font-mono tracking-wider shrink-0">Ctrl+V</span>
        </button>
      </div>
    </div>

    <!-- EDIT MENU -->
    <div class="relative">
      <button
        class="px-3 py-1.5 rounded-md hover:bg-zinc-800 text-zinc-300 hover:text-white transition-colors cursor-pointer font-medium"
        :class="{ 'bg-zinc-800 text-white shadow-xs': activeMenu === 'edit' }"
        @click="toggleMenu('edit')"
        @mouseenter="handleMenuHover('edit')"
      >
        Edit
      </button>

      <div
        v-if="activeMenu === 'edit'"
        class="absolute top-full left-0 mt-1 min-w-[320px] w-max bg-zinc-900 border border-zinc-700/80 rounded-xl shadow-2xl py-1.5 z-50 text-xs"
      >
        <button
          class="w-full text-left px-3.5 py-2 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => emit('setupBoard'))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap"><Sliders class="w-4 h-4 shrink-0 text-purple-400" /> Setup Board / FEN Editor...</span>
          <span class="text-[11px] text-zinc-500 font-mono tracking-wider shrink-0">Ctrl+B</span>
        </button>
        <button
          class="w-full text-left px-3.5 py-2 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => emit('toggleTrialMode'))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap">
            <span class="w-2.5 h-2.5 rounded-full shrink-0" :class="trialMode ? 'bg-amber-400 shadow-[0_0_6px_rgba(251,191,36,0.6)]' : 'bg-zinc-600'" />
            <span>Trial Mode {{ trialMode ? '(Active)' : '(Off)' }}</span>
          </span>
          <span class="text-[11px] text-zinc-500 font-mono tracking-wider shrink-0">Ctrl+Space</span>
        </button>

        <div class="h-px bg-zinc-800 my-1 mx-2" />

        <button
          class="w-full text-left px-3.5 py-2 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => emit('undoMove'))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap"><Undo2 class="w-4 h-4 shrink-0 text-zinc-400" /> Undo Move</span>
          <span class="text-[11px] text-zinc-500 font-mono tracking-wider shrink-0">Ctrl+Z</span>
        </button>
        <button
          class="w-full text-left px-3.5 py-2 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => emit('redoMove'))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap"><Redo2 class="w-4 h-4 shrink-0 text-zinc-400" /> Redo Move</span>
          <span class="text-[11px] text-zinc-500 font-mono tracking-wider shrink-0">Ctrl+Y</span>
        </button>
      </div>
    </div>

    <!-- GAME MENU -->
    <div class="relative">
      <button
        class="px-3 py-1.5 rounded-md hover:bg-zinc-800 text-zinc-300 hover:text-white transition-colors cursor-pointer font-medium"
        :class="{ 'bg-zinc-800 text-white shadow-xs': activeMenu === 'game' }"
        @click="toggleMenu('game')"
        @mouseenter="handleMenuHover('game')"
      >
        Game
      </button>

      <div
        v-if="activeMenu === 'game'"
        class="absolute top-full left-0 mt-1 min-w-[320px] w-max bg-zinc-900 border border-zinc-700/80 rounded-xl shadow-2xl py-1.5 z-50 text-xs"
      >
        <button
          class="w-full text-left px-3.5 py-2 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => emit('newGame'))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap"><FileText class="w-4 h-4 shrink-0 text-emerald-400" /> New Game</span>
          <span class="text-[11px] text-zinc-500 font-mono tracking-wider shrink-0">Ctrl+N</span>
        </button>
        <button
          class="w-full text-left px-3.5 py-2 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => emit('saveGame'))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap"><Save class="w-4 h-4 shrink-0 text-blue-400" /> Save Current Game</span>
          <span class="text-[11px] text-zinc-500 font-mono tracking-wider shrink-0">Ctrl+S</span>
        </button>
        <button
          class="w-full text-left px-3.5 py-2 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => emit('editTags'))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap"><FileEdit class="w-4 h-4 shrink-0 text-emerald-400" /> Edit Game Headers...</span>
          <span class="text-[11px] text-zinc-500 font-mono tracking-wider shrink-0">Ctrl+T</span>
        </button>
        <button
          class="w-full text-left px-3.5 py-2 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => emit('deleteGame'))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap"><Trash2 class="w-4 h-4 shrink-0 text-red-400" /> Delete Game</span>
          <span class="text-[11px] text-zinc-500 font-mono tracking-wider shrink-0">Ctrl+Del</span>
        </button>

        <div class="h-px bg-zinc-800 my-1 mx-2" />

        <button
          class="w-full text-left px-3.5 py-2 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => emit('firstGame'))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap"><ChevronsLeft class="w-4 h-4 shrink-0 text-zinc-400" /> First Game</span>
          <span class="text-[11px] text-zinc-500 font-mono tracking-wider shrink-0">Ctrl+Home</span>
        </button>
        <button
          class="w-full text-left px-3.5 py-2 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => emit('prevGame'))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap"><ChevronLeft class="w-4 h-4 shrink-0 text-zinc-400" /> Previous Game</span>
          <span class="text-[11px] text-zinc-500 font-mono tracking-wider shrink-0">Ctrl+Up</span>
        </button>
        <button
          class="w-full text-left px-3.5 py-2 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => emit('nextGame'))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap"><ChevronRight class="w-4 h-4 shrink-0 text-zinc-400" /> Next Game</span>
          <span class="text-[11px] text-zinc-500 font-mono tracking-wider shrink-0">Ctrl+Down</span>
        </button>
        <button
          class="w-full text-left px-3.5 py-2 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => emit('lastGame'))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap"><ChevronsRight class="w-4 h-4 shrink-0 text-zinc-400" /> Last Game</span>
          <span class="text-[11px] text-zinc-500 font-mono tracking-wider shrink-0">Ctrl+End</span>
        </button>
        <button
          class="w-full text-left px-3.5 py-2 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => emit('randomGame'))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap"><Shuffle class="w-4 h-4 shrink-0 text-zinc-400" /> Random Game</span>
          <span class="text-[11px] text-zinc-500 font-mono tracking-wider shrink-0">Ctrl+?</span>
        </button>

        <div class="h-px bg-zinc-800 my-1 mx-2" />

        <button
          class="w-full text-left px-3.5 py-2 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => emit('openTreeWindow'))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap"><BookOpen class="w-4 h-4 shrink-0 text-emerald-400" /> Opening Tree Window...</span>
          <span class="text-[11px] text-zinc-500 font-mono tracking-wider shrink-0">F3</span>
        </button>
      </div>
    </div>

    <!-- SEARCH MENU -->
    <div class="relative">
      <button
        class="px-3 py-1.5 rounded-md hover:bg-zinc-800 text-zinc-300 hover:text-white transition-colors cursor-pointer font-medium"
        :class="{ 'bg-zinc-800 text-white shadow-xs': activeMenu === 'search' }"
        @click="toggleMenu('search')"
        @mouseenter="handleMenuHover('search')"
      >
        Search
      </button>

      <div
        v-if="activeMenu === 'search'"
        class="absolute top-full left-0 mt-1 min-w-[320px] w-max bg-zinc-900 border border-zinc-700/80 rounded-xl shadow-2xl py-1.5 z-50 text-xs"
      >
        <button
          class="w-full text-left px-3.5 py-2 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => emit('searchHeader'))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap"><Filter class="w-4 h-4 shrink-0 text-blue-400" /> Header / General Search...</span>
          <span class="text-[11px] text-zinc-500 font-mono tracking-wider shrink-0">Ctrl+H</span>
        </button>
        <button
          class="w-full text-left px-3.5 py-2 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => emit('searchPosition'))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap"><Search class="w-4 h-4 shrink-0 text-emerald-400" /> Search Current Position</span>
        </button>
      </div>
    </div>

    <!-- TOOLS MENU -->
    <div class="relative">
      <button
        class="px-3 py-1.5 rounded-md hover:bg-zinc-800 text-zinc-300 hover:text-white transition-colors cursor-pointer font-medium"
        :class="{ 'bg-zinc-800 text-white shadow-xs': activeMenu === 'tools' }"
        @click="toggleMenu('tools')"
        @mouseenter="handleMenuHover('tools')"
      >
        Tools
      </button>

      <div
        v-if="activeMenu === 'tools'"
        class="absolute top-full left-0 mt-1 min-w-[380px] w-max bg-zinc-900 border border-zinc-700/80 rounded-xl shadow-2xl py-1.5 z-50 text-xs"
      >
        <button
          class="w-full text-left px-3.5 py-2 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => emit('annotateGame'))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap"><Sparkles class="w-4 h-4 shrink-0 text-amber-400" /> Full Game Analysis (Annotate)</span>
        </button>
        <button
          class="w-full text-left px-3.5 py-2 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => emit('toggleAnalysis'))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap"><Cpu class="w-4 h-4 shrink-0 text-emerald-400" /> {{ isAnalyzing ? 'Stop Engine Analysis' : 'Start Live Analysis' }}</span>
          <span class="text-[11px] text-zinc-500 font-mono tracking-wider shrink-0">F2</span>
        </button>
        <button
          class="w-full text-left px-3.5 py-2 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => emit('openTreeWindow'))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap"><BookOpen class="w-4 h-4 shrink-0 text-emerald-400" /> Opening Tree Window...</span>
          <span class="text-[11px] text-zinc-500 font-mono tracking-wider shrink-0">F3</span>
        </button>
        <div class="h-px bg-zinc-800 my-1 mx-2" />
        <button
          class="w-full text-left px-3.5 py-2 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => emit('minePuzzles'))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap"><Pickaxe class="w-4 h-4 shrink-0 text-amber-400" /> Mine Puzzles from Database Games...</span>
        </button>
        <button
          v-if="dbName"
          class="w-full text-left px-3.5 py-2 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => emit('databaseMaintenance'))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap"><Wrench class="w-4 h-4 shrink-0 text-emerald-400" /> Database Maintenance & Tools...</span>
        </button>
        <button
          v-if="dbName"
          class="w-full text-left px-3.5 py-2 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => emit('standardizeNames'))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap"><Sparkles class="w-4 h-4 shrink-0 text-purple-400" /> Standardize Player Names (Spellcheck)...</span>
        </button>
        <button
          class="w-full text-left px-3.5 py-2 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => emit('onlineSync'))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap"><Globe class="w-4 h-4 shrink-0 text-cyan-400" /> Online Account Sync (Chess.com & Lichess)...</span>
        </button>
        <button
          class="w-full text-left px-3.5 py-2 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => emit('twicDownloader'))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap"><Download class="w-4 h-4 shrink-0 text-cyan-400" /> Download TWIC Bulletins...</span>
        </button>
        <button
          class="w-full text-left px-3.5 py-2 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => emit('megaDownloader'))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap"><Download class="w-4 h-4 shrink-0 text-emerald-400" /> Download Master &amp; Mega Databases (Lumbra / Lichess)...</span>
        </button>
        <button
          class="w-full text-left px-3.5 py-2 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => emit('playerPersonas'))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap"><Users class="w-4 h-4 shrink-0 text-purple-400" /> Player Personas & Aliases...</span>
        </button>
        <button
          class="w-full text-left px-3.5 py-2 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => emit('showCrosstable'))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap"><Trophy class="w-4 h-4 shrink-0 text-amber-400" /> Tournament Crosstables & Standings...</span>
          <span class="text-[11px] text-zinc-500 font-mono tracking-wider shrink-0">F7</span>
        </button>
        <button
          class="w-full text-left px-3.5 py-2 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => emit('ecoBrowser'))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap"><BookOpen class="w-4 h-4 shrink-0 text-emerald-400" /> ECO Opening Classification Browser...</span>
          <span class="text-[11px] text-zinc-500 font-mono tracking-wider shrink-0">F6</span>
        </button>
        <button
          class="w-full text-left px-3.5 py-2 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => emit('databaseStats'))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap"><BarChart3 class="w-4 h-4 shrink-0 text-emerald-400" /> Database Demographics & Rating Tracker...</span>
          <span class="text-[11px] text-zinc-500 font-mono tracking-wider shrink-0">Ctrl+D</span>
        </button>
        <div class="h-px bg-zinc-800 my-1 mx-2" />
        <button
          class="w-full text-left px-3.5 py-2 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => emit('explainMove'))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap"><Sparkles class="w-4 h-4 shrink-0 text-amber-400" /> Explain Current Move (Concept Engine)...</span>
        </button>
        <button
          class="w-full text-left px-3.5 py-2 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => emit('blunderVault'))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap"><Brain class="w-4 h-4 shrink-0 text-rose-400" /> Personal Blunder Vault (SM-2 Trainer)...</span>
        </button>
        <button
          class="w-full text-left px-3.5 py-2 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => emit('opponentScout'))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap"><Shield class="w-4 h-4 shrink-0 text-cyan-400" /> Opponent Scouting Card & Prep Matrix...</span>
        </button>
        <button
          class="w-full text-left px-3.5 py-2 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => emit('antiPersona'))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap"><Crosshair class="w-4 h-4 shrink-0 text-rose-400" /> Opponent Opening Exploiter & Custom Book...</span>
        </button>
        <button
          class="w-full text-left px-3.5 py-2 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => emit('timeManagement'))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap"><Clock class="w-4 h-4 shrink-0 text-indigo-400" /> Time Management & Tanking Dossier...</span>
        </button>
        <button
          class="w-full text-left px-3.5 py-2 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => emit('fairPlayInspector'))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap"><Shield class="w-4 h-4 shrink-0 text-amber-400" /> Fair-Play &amp; Time-Control Consistency Inspector...</span>
        </button>
      </div>
    </div>

    <!-- PLAY MENU -->
    <div class="relative">
      <button
        class="px-3 py-1.5 rounded-md hover:bg-zinc-800 text-zinc-300 hover:text-white transition-colors cursor-pointer font-medium"
        :class="{ 'bg-zinc-800 text-white shadow-xs': activeMenu === 'play' }"
        @click="toggleMenu('play')"
        @mouseenter="handleMenuHover('play')"
      >
        Play
      </button>

      <div
        v-if="activeMenu === 'play'"
        class="absolute top-full left-0 mt-1 min-w-[360px] w-max bg-zinc-900 border border-zinc-700/80 rounded-xl shadow-2xl py-1.5 z-50 text-xs"
      >
        <button
          class="w-full text-left px-3.5 py-2 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => emit('playVsEngine'))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap"><Swords class="w-4 h-4 shrink-0 text-amber-400" /> Play Against Engine...</span>
        </button>
        <button
          class="w-full text-left px-3.5 py-2 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => emit('engineMatch'))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap"><Swords class="w-4 h-4 shrink-0 text-cyan-400" /> Engine vs Engine Match...</span>
        </button>
        <button
          class="w-full text-left px-3.5 py-2 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => emit('simulatePlayer'))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap"><Bot class="w-4 h-4 shrink-0 text-purple-400" /> Simulate Database Player (Sparring)...</span>
        </button>
        <button
          class="w-full text-left px-3.5 py-2 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => emit('playerPersonas'))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap"><Users class="w-4 h-4 shrink-0 text-emerald-400" /> Player Personas (Spar & Aggregate)...</span>
        </button>
        <button
          class="w-full text-left px-3.5 py-2 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => emit('tacticsTrainer'))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap"><Award class="w-4 h-4 shrink-0 text-blue-400" /> Tactics Puzzle Trainer...</span>
        </button>
        <button
          class="w-full text-left px-3.5 py-2 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => emit('guessTheMove'))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap"><Award class="w-4 h-4 shrink-0 text-amber-400" /> Guess the Move (Solitaire Chess)...</span>
          <span class="text-[11px] text-zinc-500 font-mono tracking-wider shrink-0">Ctrl+G</span>
        </button>
        <div class="h-px bg-zinc-800 my-1 mx-2" />
        <button
          class="w-full text-left px-3.5 py-2 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => emit('progressiveTactics'))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap"><Zap class="w-4 h-4 shrink-0 text-amber-400" /> Progressive Tactics Challenge...</span>
        </button>
        <button
          class="w-full text-left px-3.5 py-2 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => emit('openingTrainer'))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap"><BookOpen class="w-4 h-4 shrink-0 text-emerald-400" /> Opening Builder & Trainer...</span>
        </button>
        <button
          class="w-full text-left px-3.5 py-2 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => emit('repertoireShortener'))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap"><Scissors class="w-4 h-4 shrink-0 text-rose-400" /> Repertoire Shortener (Pareto 80/20)...</span>
        </button>
        <div class="h-px bg-zinc-800 my-1 mx-2" />
        <button
          class="w-full text-left px-3.5 py-2 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => emit('chess960'))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap"><Dices class="w-4 h-4 shrink-0 text-purple-400" /> Chess960 (Fischer Random)...</span>
        </button>
        <button
          class="w-full text-left px-3.5 py-2 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => emit('coordinateTrainer'))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap"><Target class="w-4 h-4 shrink-0 text-emerald-400" /> Coordinate & Vision Trainer...</span>
        </button>
        <button
          class="w-full text-left px-3.5 py-2 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => emit('pieceSafety'))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap"><Shield class="w-4 h-4 shrink-0 text-emerald-400" /> Piece Safety Scanner (Don't Hang Pieces)...</span>
        </button>
        <button
          class="w-full text-left px-3.5 py-2 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => emit('intuitionStorm'))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap"><Zap class="w-4 h-4 shrink-0 text-amber-400" /> Intuition Storm (Speed Reflex Trainer)...</span>
        </button>
        <button
          class="w-full text-left px-3.5 py-2 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => emit('prophylaxis'))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap"><Shield class="w-4 h-4 shrink-0 text-emerald-400" /> Quiet Moves &amp; Prophylaxis Drill...</span>
        </button>
      </div>
    </div>

    <!-- WINDOWS MENU -->
    <div class="relative">
      <button
        class="px-3 py-1.5 rounded-md hover:bg-zinc-800 text-zinc-300 hover:text-white transition-colors cursor-pointer font-medium"
        :class="{ 'bg-zinc-800 text-white shadow-xs': activeMenu === 'windows' }"
        @click="toggleMenu('windows')"
        @mouseenter="handleMenuHover('windows')"
      >
        Windows
      </button>

      <div
        v-if="activeMenu === 'windows'"
        class="absolute top-full left-0 mt-1 min-w-[340px] w-max bg-zinc-900 border border-zinc-700/80 rounded-xl shadow-2xl py-1.5 z-50 text-xs"
      >
        <button
          class="w-full text-left px-3.5 py-2 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => emit('openTreeWindow'))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap"><BookOpen class="w-4 h-4 shrink-0 text-emerald-400" /> Opening Tree &amp; Repertoire...</span>
          <span class="text-[11px] text-zinc-500 font-mono tracking-wider shrink-0">F3</span>
        </button>
        <button
          class="w-full text-left px-3.5 py-2 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => emit('showCrosstable'))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap"><Trophy class="w-4 h-4 shrink-0 text-amber-400" /> Tournament Crosstable...</span>
          <span class="text-[11px] text-zinc-500 font-mono tracking-wider shrink-0">Ctrl+T</span>
        </button>
        <button
          class="w-full text-left px-3.5 py-2 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => emit('ecoBrowser'))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap"><BookOpen class="w-4 h-4 shrink-0 text-blue-400" /> ECO Opening Directory...</span>
        </button>
        <button
          class="w-full text-left px-3.5 py-2 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => emit('playerPersonas'))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap"><Users class="w-4 h-4 shrink-0 text-emerald-400" /> Player Personas Explorer...</span>
        </button>
        <div class="h-px bg-zinc-800 my-1 mx-2" />
        <button
          class="w-full text-left px-3.5 py-2 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => emit('databaseStats'))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap"><BarChart3 class="w-4 h-4 shrink-0 text-cyan-400" /> Database Summary &amp; Analytics...</span>
        </button>
      </div>
    </div>

    <!-- OPTIONS MENU -->
    <div class="relative">
      <button
        class="px-3 py-1.5 rounded-md hover:bg-zinc-800 text-zinc-300 hover:text-white transition-colors cursor-pointer font-medium"
        :class="{ 'bg-zinc-800 text-white shadow-xs': activeMenu === 'options' }"
        @click="toggleMenu('options')"
        @mouseenter="handleMenuHover('options')"
      >
        Options
      </button>

      <div
        v-if="activeMenu === 'options'"
        class="absolute top-full left-0 mt-1 min-w-[320px] w-max bg-zinc-900 border border-zinc-700/80 rounded-xl shadow-2xl py-1.5 z-50 text-xs"
      >
        <button
          class="w-full text-left px-3.5 py-2 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => emit('toggleFlip'))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap"><RotateCcw class="w-4 h-4 shrink-0 text-zinc-400" /> Flip Board</span>
          <span class="text-[11px] text-zinc-500 font-mono tracking-wider shrink-0">F</span>
        </button>
        <div class="h-px bg-zinc-800 my-1 mx-2" />
        <button
          class="w-full text-left px-3.5 py-2 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => emit('openSettings'))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap"><Palette class="w-4 h-4 shrink-0 text-emerald-400" /> Themes &amp; Piece Sets...</span>
        </button>

        <div class="h-px bg-zinc-800 my-1 mx-2" />
        <div class="px-3.5 py-1 text-[10px] text-zinc-500 font-bold uppercase tracking-wider">UI Theme (Dark / Light)</div>

        <button
          class="w-full text-left px-3.5 py-2 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => emit('toggleDarkLight'))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap">
            <Sun v-if="currentUITheme === 'light' || currentUITheme === 'retro'" class="w-4 h-4 shrink-0 text-amber-400" />
            <Moon v-else class="w-4 h-4 shrink-0 text-cyan-400" />
            <span>Toggle Dark / Light Mode</span>
          </span>
          <span class="text-[11px] text-zinc-500 font-mono tracking-wider shrink-0">F10</span>
        </button>

        <div class="px-2 py-1 grid grid-cols-2 gap-1 bg-zinc-950/40 rounded-lg mx-2 border border-zinc-800/60 my-1">
          <button
            v-for="t in UI_THEMES"
            :key="t.id"
            class="px-2 py-1.5 rounded text-[11px] flex items-center justify-between transition-colors cursor-pointer"
            :class="currentUITheme === t.id
              ? 'bg-emerald-950/60 text-emerald-300 font-semibold border border-emerald-500/40'
              : 'text-zinc-400 hover:text-zinc-200 hover:bg-zinc-800/80'"
            @click="selectItem(() => emit('changeUITheme', t.id))"
          >
            <span class="flex items-center gap-1.5 truncate">
              <span class="w-2 h-2 rounded-full shrink-0" :style="{ backgroundColor: t.preview.accent }" />
              <span class="truncate">{{ t.name }}</span>
            </span>
            <Check v-if="currentUITheme === t.id" class="w-3 h-3 text-emerald-400 shrink-0" />
          </button>
        </div>

        <div class="h-px bg-zinc-800 my-1 mx-2" />
        <div class="px-3.5 py-1 text-[10px] text-zinc-500 font-bold uppercase tracking-wider">Audio &amp; Visuals</div>

        <button
          class="w-full text-left px-3.5 py-2 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => emit('toggleSound'))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap">
            <Volume2 v-if="soundEnabled" class="w-4 h-4 shrink-0 text-emerald-400" />
            <VolumeX v-else class="w-4 h-4 shrink-0 text-zinc-500" />
            <span>Sound Effects {{ soundEnabled ? '(Enabled)' : '(Muted)' }}</span>
          </span>
          <Check v-if="soundEnabled" class="w-3.5 h-3.5 text-emerald-400 shrink-0" />
        </button>

        <button
          class="w-full text-left px-3.5 py-2 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => emit('toggleEvalBar'))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap">
            <BarChart3 class="w-4 h-4 shrink-0" :class="showEvalBar ? 'text-cyan-400' : 'text-zinc-500'" />
            <span>Evaluation Gauge {{ showEvalBar ? '(Visible)' : '(Hidden)' }}</span>
          </span>
          <Check v-if="showEvalBar" class="w-3.5 h-3.5 text-cyan-400 shrink-0" />
        </button>

        <button
          class="w-full text-left px-3.5 py-2 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => emit('toggleThreatMode'))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap">
            <Eye class="w-4 h-4 shrink-0" :class="threatMode ? 'text-amber-400' : 'text-zinc-500'" />
            <span>Threat / Spy Highlights {{ threatMode ? '(Active)' : '(Off)' }}</span>
          </span>
          <Check v-if="threatMode" class="w-3.5 h-3.5 text-amber-400 shrink-0" />
        </button>

        <button
          class="w-full text-left px-3.5 py-2 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => emit('toggleBlindfold'))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap">
            <EyeOff class="w-4 h-4 shrink-0" :class="blindfold ? 'text-purple-400' : 'text-zinc-500'" />
            <span>Blindfold Mode {{ blindfold ? '(Active)' : '(Off)' }}</span>
          </span>
          <Check v-if="blindfold" class="w-3.5 h-3.5 text-purple-400 shrink-0" />
        </button>

        <div class="h-px bg-zinc-800 my-1 mx-2" />
        <div class="px-3.5 py-1 text-[10px] text-zinc-500 font-bold uppercase tracking-wider">Productivity & Focus</div>

        <button
          class="w-full text-left px-3.5 py-2 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => emit('commandPalette'))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap"><Search class="w-4 h-4 shrink-0 text-emerald-400" /> Universal Command Palette...</span>
          <span class="text-[11px] text-zinc-500 font-mono tracking-wider shrink-0">Ctrl+K</span>
        </button>

        <button
          class="w-full text-left px-3.5 py-2 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => emit('toggleZenMode'))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap">
            <Maximize2 class="w-4 h-4 shrink-0" :class="zenMode ? 'text-emerald-400' : 'text-zinc-500'" />
            <span>Zen / Study Mode {{ zenMode ? '(Active)' : '(Off)' }}</span>
          </span>
          <span class="text-[11px] text-zinc-500 font-mono tracking-wider shrink-0">Z</span>
        </button>

        <div class="h-px bg-zinc-800 my-1 mx-2" />
        <div class="px-3.5 py-1 text-[10px] text-zinc-500 font-bold uppercase tracking-wider">Notation Style</div>
        <button
          v-for="opt in notationOptions"
          :key="opt.id"
          class="w-full text-left px-3.5 py-1.5 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => setNotationStyle(opt.id))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap">
            <span :class="currentStyle === opt.id ? 'text-amber-400 font-bold' : 'text-zinc-400'">{{ opt.name }}</span>
          </span>
          <Check v-if="currentStyle === opt.id" class="w-3.5 h-3.5 text-amber-400 shrink-0" />
        </button>
      </div>
    </div>

    <!-- HELP MENU -->
    <div class="relative">
      <button
        class="px-3 py-1.5 rounded-md hover:bg-zinc-800 text-zinc-300 hover:text-white transition-colors cursor-pointer font-medium"
        :class="{ 'bg-zinc-800 text-white shadow-xs': activeMenu === 'help' }"
        @click="toggleMenu('help')"
        @mouseenter="handleMenuHover('help')"
      >
        Help
      </button>

      <div
        v-if="activeMenu === 'help'"
        class="absolute top-full left-0 mt-1 min-w-[300px] w-max bg-zinc-900 border border-zinc-700/80 rounded-xl shadow-2xl py-1.5 z-50 text-xs"
      >
        <button
          class="w-full text-left px-3.5 py-2 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => emit('showHelp'))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap"><BookOpen class="w-4 h-4 shrink-0 text-emerald-400" /> Online Help & Guides...</span>
          <span class="text-[11px] text-zinc-500 font-mono tracking-wider shrink-0">F1</span>
        </button>
        <button
          class="w-full text-left px-3.5 py-2 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => emit('openWebDocs'))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap"><ExternalLink class="w-4 h-4 shrink-0 text-cyan-400" /> Web Documentation (Online)...</span>
        </button>
        <div class="h-px bg-zinc-800 my-1 mx-2" />
        <button
          class="w-full text-left px-3.5 py-2 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => emit('showShortcuts'))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap"><HelpCircle class="w-4 h-4 shrink-0 text-emerald-400" /> Keyboard Shortcuts</span>
          <span class="text-[11px] text-zinc-500 font-mono tracking-wider shrink-0">?</span>
        </button>
        <div class="h-px bg-zinc-800 my-1 mx-2" />
        <button
          class="w-full text-left px-3.5 py-2 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => emit('showSupport'))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap"><Heart class="w-4 h-4 shrink-0 text-rose-400" /> Support NewScid ($5+)...</span>
        </button>
        <button
          class="w-full text-left px-3.5 py-2 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between gap-6 cursor-pointer transition-colors whitespace-nowrap"
          @click="selectItem(() => emit('showAbout'))"
        >
          <span class="flex items-center gap-2.5 whitespace-nowrap"><Info class="w-4 h-4 shrink-0 text-zinc-400" /> About NewScid</span>
        </button>
      </div>
    </div>

    <!-- Command Palette & Support Quick Triggers -->
    <div class="ml-auto mr-2 flex items-center gap-1.5">
      <button
        class="px-2.5 py-1 rounded-lg bg-zinc-900 border border-zinc-800 hover:border-rose-500/50 text-zinc-300 hover:text-rose-300 text-xs flex items-center gap-1.5 cursor-pointer transition-colors shadow-xs"
        title="Support the Author ($5+)"
        @click="emit('showSupport')"
      >
        <Heart class="w-3.5 h-3.5 text-rose-400 fill-rose-400/20" />
        <span class="hidden md:inline text-[11px] font-medium">Support</span>
      </button>

      <button
        class="px-2.5 py-1 rounded-lg bg-zinc-900 border border-zinc-800 hover:border-zinc-700 text-zinc-400 hover:text-zinc-200 text-xs flex items-center gap-2 cursor-pointer transition-colors shadow-xs"
        title="Universal Command Palette & Spotlight (Ctrl+K)"
        @click="emit('commandPalette')"
      >
        <Search class="w-3.5 h-3.5 text-emerald-400" />
        <span class="hidden sm:inline text-[11px]">Command Palette</span>
        <kbd class="px-1.5 py-0.5 rounded bg-zinc-800 text-[10px] font-mono text-zinc-400 border border-zinc-700">Ctrl+K</kbd>
      </button>
    </div>

    <!-- TRIAL MODE BADGE INDICATOR -->
    <div v-if="trialMode" class="ml-auto flex items-center gap-1.5 px-2.5 py-1 rounded-full bg-amber-500/20 border border-amber-500/40 text-amber-300 text-[11px] font-semibold animate-pulse">
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
