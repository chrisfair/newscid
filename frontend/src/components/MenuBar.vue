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
  Search,
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
  RotateCcw
} from '@lucide/vue'
import { PieceSetOption, BoardThemeOption } from './HeaderBar.vue'

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
  (e: 'searchPosition'): void
  (e: 'toggleFlip'): void
  (e: 'changePieceSet', id: string): void
  (e: 'changeTheme', id: string): void
  (e: 'showShortcuts'): void
}>()

const activeMenu = ref<string | null>(null)

function toggleMenu(menu: string) {
  if (activeMenu.value === menu) {
    activeMenu.value = null
  } else {
    activeMenu.value = menu
  }
}

function handleMenuHover(menu: string) {
  if (activeMenu.value !== null) {
    activeMenu.value = menu
  }
}

function closeAll() {
  activeMenu.value = null
}

function handleGlobalClick(e: MouseEvent) {
  const target = e.target as HTMLElement
  if (!target.closest('.app-menubar')) {
    closeAll()
  }
}

onMounted(() => {
  window.addEventListener('click', handleGlobalClick)
})

onUnmounted(() => {
  window.removeEventListener('click', handleGlobalClick)
})
</script>

<template>
  <div class="app-menubar bg-zinc-950 border-b border-zinc-800/80 px-2 py-0.5 flex items-center gap-1 text-xs select-none z-30">
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
          @click="emit('openDB'); closeAll()"
        >
          <span class="flex items-center gap-2"><FolderOpen class="w-3.5 h-3.5 text-emerald-400" /> Open Database...</span>
          <span class="text-[10px] text-zinc-500 font-mono">Ctrl+O</span>
        </button>
        <button
          class="w-full text-left px-3 py-1.5 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between cursor-pointer"
          @click="emit('createDB'); closeAll()"
        >
          <span class="flex items-center gap-2"><FileText class="w-3.5 h-3.5 text-zinc-400" /> New Database...</span>
        </button>
        <button
          v-if="dbName"
          class="w-full text-left px-3 py-1.5 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between cursor-pointer"
          @click="emit('closeDB'); closeAll()"
        >
          <span class="flex items-center gap-2"><Trash2 class="w-3.5 h-3.5 text-amber-400" /> Close Database</span>
        </button>

        <div class="h-px bg-zinc-800 my-1" />

        <button
          class="w-full text-left px-3 py-1.5 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between cursor-pointer"
          @click="emit('importPGN'); closeAll()"
        >
          <span class="flex items-center gap-2"><ClipboardPaste class="w-3.5 h-3.5 text-zinc-400" /> Import PGN Game...</span>
          <span class="text-[10px] text-zinc-500 font-mono">Ctrl+I</span>
        </button>
        <button
          class="w-full text-left px-3 py-1.5 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between cursor-pointer"
          @click="emit('exportPGN'); closeAll()"
        >
          <span class="flex items-center gap-2"><Copy class="w-3.5 h-3.5 text-zinc-400" /> Export PGN Game</span>
          <span class="text-[10px] text-zinc-500 font-mono">Ctrl+E</span>
        </button>
        <button
          class="w-full text-left px-3 py-1.5 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between cursor-pointer"
          @click="emit('copyFEN'); closeAll()"
        >
          <span class="flex items-center gap-2"><Copy class="w-3.5 h-3.5 text-zinc-400" /> Copy FEN to Clipboard</span>
          <span class="text-[10px] text-zinc-500 font-mono">Ctrl+C</span>
        </button>
        <button
          class="w-full text-left px-3 py-1.5 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between cursor-pointer"
          @click="emit('pasteFEN'); closeAll()"
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
          @click="emit('setupBoard'); closeAll()"
        >
          <span class="flex items-center gap-2"><Sliders class="w-3.5 h-3.5 text-purple-400" /> Setup Board / FEN Editor...</span>
          <span class="text-[10px] text-zinc-500 font-mono">Ctrl+B</span>
        </button>
        <button
          class="w-full text-left px-3 py-1.5 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between cursor-pointer"
          @click="emit('toggleTrialMode'); closeAll()"
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
          @click="emit('undoMove'); closeAll()"
        >
          <span class="flex items-center gap-2"><Undo2 class="w-3.5 h-3.5 text-zinc-400" /> Undo Move</span>
          <span class="text-[10px] text-zinc-500 font-mono">Ctrl+Z</span>
        </button>
        <button
          class="w-full text-left px-3 py-1.5 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between cursor-pointer"
          @click="emit('redoMove'); closeAll()"
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
          @click="emit('newGame'); closeAll()"
        >
          <span class="flex items-center gap-2"><FileText class="w-3.5 h-3.5 text-emerald-400" /> New Game</span>
          <span class="text-[10px] text-zinc-500 font-mono">Ctrl+N</span>
        </button>
        <button
          class="w-full text-left px-3 py-1.5 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between cursor-pointer"
          @click="emit('saveGame'); closeAll()"
        >
          <span class="flex items-center gap-2"><Save class="w-3.5 h-3.5 text-blue-400" /> Save Current Game</span>
          <span class="text-[10px] text-zinc-500 font-mono">Ctrl+S</span>
        </button>
        <button
          class="w-full text-left px-3 py-1.5 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between cursor-pointer"
          @click="emit('deleteGame'); closeAll()"
        >
          <span class="flex items-center gap-2"><Trash2 class="w-3.5 h-3.5 text-red-400" /> Delete Game</span>
          <span class="text-[10px] text-zinc-500 font-mono">Ctrl+Del</span>
        </button>

        <div class="h-px bg-zinc-800 my-1" />

        <button
          class="w-full text-left px-3 py-1.5 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between cursor-pointer"
          @click="emit('firstGame'); closeAll()"
        >
          <span>First Game</span>
          <span class="text-[10px] text-zinc-500 font-mono">Ctrl+Home</span>
        </button>
        <button
          class="w-full text-left px-3 py-1.5 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between cursor-pointer"
          @click="emit('prevGame'); closeAll()"
        >
          <span>Previous Game</span>
          <span class="text-[10px] text-zinc-500 font-mono">Ctrl+Up</span>
        </button>
        <button
          class="w-full text-left px-3 py-1.5 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between cursor-pointer"
          @click="emit('nextGame'); closeAll()"
        >
          <span>Next Game</span>
          <span class="text-[10px] text-zinc-500 font-mono">Ctrl+Down</span>
        </button>
        <button
          class="w-full text-left px-3 py-1.5 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between cursor-pointer"
          @click="emit('lastGame'); closeAll()"
        >
          <span>Last Game</span>
          <span class="text-[10px] text-zinc-500 font-mono">Ctrl+End</span>
        </button>
        <button
          class="w-full text-left px-3 py-1.5 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between cursor-pointer"
          @click="emit('randomGame'); closeAll()"
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
        class="absolute top-full left-0 mt-0.5 w-60 bg-zinc-900 border border-zinc-700/80 rounded-lg shadow-2xl py-1 z-50 text-xs"
      >
        <button
          class="w-full text-left px-3 py-1.5 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between cursor-pointer"
          @click="emit('searchPosition'); closeAll()"
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
          @click="emit('annotateGame'); closeAll()"
        >
          <span class="flex items-center gap-2"><Sparkles class="w-3.5 h-3.5 text-amber-400" /> Full Game Analysis (Annotate)</span>
        </button>
        <button
          class="w-full text-left px-3 py-1.5 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between cursor-pointer"
          @click="emit('toggleAnalysis'); closeAll()"
        >
          <span class="flex items-center gap-2"><Cpu class="w-3.5 h-3.5 text-emerald-400" /> {{ isAnalyzing ? 'Stop Engine Analysis' : 'Start Live Analysis' }}</span>
          <span class="text-[10px] text-zinc-500 font-mono">F2</span>
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
        class="absolute top-full left-0 mt-0.5 w-60 bg-zinc-900 border border-zinc-700/80 rounded-lg shadow-2xl py-1 z-50 text-xs"
      >
        <button
          class="w-full text-left px-3 py-1.5 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between cursor-pointer"
          @click="emit('playVsEngine'); closeAll()"
        >
          <span class="flex items-center gap-2"><Swords class="w-3.5 h-3.5 text-amber-400" /> Play Against Engine...</span>
        </button>
        <button
          class="w-full text-left px-3 py-1.5 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between cursor-pointer"
          @click="emit('tacticsTrainer'); closeAll()"
        >
          <span class="flex items-center gap-2"><Award class="w-3.5 h-3.5 text-blue-400" /> Tactics Puzzle Trainer...</span>
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
        class="absolute top-full left-0 mt-0.5 w-56 bg-zinc-900 border border-zinc-700/80 rounded-lg shadow-2xl py-1 z-50 text-xs"
      >
        <button
          class="w-full text-left px-3 py-1.5 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between cursor-pointer"
          @click="emit('toggleFlip'); closeAll()"
        >
          <span class="flex items-center gap-2"><RotateCcw class="w-3.5 h-3.5 text-zinc-400" /> Flip Board</span>
          <span class="text-[10px] text-zinc-500 font-mono">F</span>
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
        class="absolute top-full left-0 mt-0.5 w-52 bg-zinc-900 border border-zinc-700/80 rounded-lg shadow-2xl py-1 z-50 text-xs"
      >
        <button
          class="w-full text-left px-3 py-1.5 hover:bg-zinc-800 text-zinc-200 flex items-center justify-between cursor-pointer"
          @click="emit('showShortcuts'); closeAll()"
        >
          <span class="flex items-center gap-2"><HelpCircle class="w-3.5 h-3.5 text-emerald-400" /> Keyboard Shortcuts</span>
          <span class="text-[10px] text-zinc-500 font-mono">?</span>
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
