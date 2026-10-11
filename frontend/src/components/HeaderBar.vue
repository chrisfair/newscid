<script setup lang="ts">
import { ref, computed } from 'vue'
import {
  FolderOpen,
  PlusCircle,
  FileDown,
  FileUp,
  Settings,
  Palette,
  RotateCcw,
  BookOpen,
  Database,
  ChevronDown,
  Sun,
  Moon,
  Check
} from '@lucide/vue'
import { UI_THEMES, type UITheme } from '../utils/theme'

export interface PieceSetOption {
  id: string
  name: string
  isScidNative: boolean
  description: string
}

export interface BoardThemeOption {
  id: string
  name: string
  light: string
  dark: string
  highlight: string
}

export interface DiscoveredDatabase {
  name: string
  basePath: string
  type: string
  category: string
  numGames: number
  description: string
  sizeStr: string
  exists: boolean
}

const props = defineProps<{
  dbName?: string
  systemBases: string[]
  discoveredDatabases?: DiscoveredDatabase[]
  pieceSets: PieceSetOption[]
  boardThemes: BoardThemeOption[]
  currentPieceSet: string
  currentThemeId: string
  currentUITheme?: string
  flipped: boolean
}>()

const emit = defineEmits<{
  (e: 'openDB'): void
  (e: 'createDB'): void
  (e: 'openSystemBase', path: string): void
  (e: 'openDiscoveredBase', path: string): void
  (e: 'importPGN'): void
  (e: 'exportPGN'): void
  (e: 'changePieceSet', id: string): void
  (e: 'changeTheme', id: string): void
  (e: 'changeUITheme', id: string): void
  (e: 'toggleDarkLight'): void
  (e: 'toggleFlip'): void
}>()

const isCurrentLight = computed(() => props.currentUITheme === 'light' || props.currentUITheme === 'retro')

const showSettingsModal = ref(false)
const showBasesDropdown = ref(false)

const groupedDatabases = computed(() => {
  const groups: Record<string, DiscoveredDatabase[]> = {}
  if (props.discoveredDatabases && props.discoveredDatabases.length > 0) {
    for (const db of props.discoveredDatabases) {
      const cat = db.category || 'Other Databases'
      if (!groups[cat]) {
        groups[cat] = []
      }
      groups[cat].push(db)
    }
    return groups
  }

  if (props.systemBases && props.systemBases.length > 0) {
    groups['Tutorial & Tactics'] = props.systemBases.map(b => ({
      name: b.split('/').pop() || b,
      basePath: b,
      type: 'scid',
      category: 'Tutorial & Tactics',
      numGames: 0,
      description: '',
      sizeStr: '',
      exists: true,
    }))
  }
  return groups
})

function formatGameCount(count: number): string {
  if (count <= 0) return ''
  if (count >= 1000000) {
    return `${(count / 1000000).toFixed(1)}M games`
  }
  if (count >= 1000) {
    return `${(count / 1000).toFixed(1)}k games`
  }
  return `${count} games`
}

defineExpose({
  openSettings: () => {
    showSettingsModal.value = true
  }
})
</script>

<template>
  <header class="app-header relative z-30 bg-zinc-950 border-b border-zinc-800 px-4 py-2 flex items-center justify-between select-none">
    <!-- Left: Brand Logo & Database Status -->
    <div class="flex items-center gap-3">
      <div class="flex items-center gap-2">
        <div class="w-8 h-8 rounded-lg bg-emerald-600 flex items-center justify-center font-bold text-white shadow-md text-base">
          ♘
        </div>
        <span class="font-extrabold text-base tracking-tight text-zinc-100">
          New<span class="text-emerald-400">Scid</span>
        </span>
      </div>

      <div class="h-4 w-px bg-zinc-800" />

      <!-- Database Actions -->
      <div class="flex items-center gap-1.5">
        <button
          class="header-btn flex items-center gap-1 px-2.5 py-1 rounded-lg bg-zinc-900 hover:bg-zinc-800 text-zinc-300 text-xs font-medium border border-zinc-800 transition-colors"
          @click="emit('openDB')"
        >
          <FolderOpen class="w-3.5 h-3.5 text-emerald-400" />
          <span>Open DB</span>
        </button>

        <button
          class="header-btn flex items-center gap-1 px-2.5 py-1 rounded-lg bg-zinc-900 hover:bg-zinc-800 text-zinc-300 text-xs font-medium border border-zinc-800 transition-colors"
          @click="emit('createDB')"
        >
          <PlusCircle class="w-3.5 h-3.5 text-zinc-400" />
          <span>Create DB</span>
        </button>

        <!-- Installed & Discovered Databases Dropdown -->
        <div v-if="Object.keys(groupedDatabases).length > 0" class="relative">
          <button
            class="header-btn flex items-center gap-1.5 px-2.5 py-1 rounded-lg bg-zinc-900 hover:bg-zinc-800 text-zinc-300 text-xs font-medium border border-zinc-800 transition-colors"
            @click="showBasesDropdown = !showBasesDropdown"
          >
            <Database class="w-3.5 h-3.5 text-blue-400" />
            <span>Installed Bases</span>
            <ChevronDown class="w-3 h-3 text-zinc-500" />
          </button>

          <!-- Backdrop click listener -->
          <div
            v-if="showBasesDropdown"
            class="fixed inset-0 z-40"
            @click="showBasesDropdown = false"
          />

          <div
            v-if="showBasesDropdown"
            class="absolute top-full left-0 mt-1 min-w-[340px] max-w-md w-max max-h-[440px] overflow-y-auto bg-zinc-900 border border-zinc-700 rounded-xl shadow-2xl py-1 z-50 text-xs divide-y divide-zinc-800"
          >
            <div v-for="(dbs, category) in groupedDatabases" :key="category" class="py-1">
              <div class="px-3 py-1 text-[10px] font-bold text-zinc-400 uppercase tracking-wider bg-zinc-950/70 flex items-center justify-between">
                <span>{{ category }}</span>
                <span class="text-zinc-500 font-mono">{{ dbs.length }}</span>
              </div>
              <button
                v-for="db in dbs"
                :key="db.basePath"
                class="w-full text-left px-3 py-2 hover:bg-zinc-800 text-zinc-200 transition-colors flex items-center justify-between group cursor-pointer"
                @click="emit('openDiscoveredBase', db.basePath); showBasesDropdown = false"
              >
                <div class="flex flex-col min-w-0 pr-2">
                  <div class="flex items-center gap-1.5">
                    <span class="font-medium truncate text-zinc-100 group-hover:text-emerald-400 transition-colors">{{ db.name }}</span>
                    <span
                      class="px-1 py-0.2 rounded text-[9px] font-mono uppercase"
                      :class="db.type === 'pgn' ? 'bg-amber-950/60 text-amber-400 border border-amber-800/50' : 'bg-emerald-950/60 text-emerald-400 border border-emerald-800/50'"
                    >
                      {{ db.type }}
                    </span>
                  </div>
                  <span v-if="db.description" class="text-[10px] text-zinc-400 truncate mt-0.5">{{ db.description }}</span>
                </div>
                <div class="flex flex-col items-end shrink-0 text-[10px] font-mono">
                  <span v-if="db.numGames > 0" class="text-emerald-400 font-semibold">{{ formatGameCount(db.numGames) }}</span>
                  <span v-if="db.sizeStr" class="text-zinc-500">{{ db.sizeStr }}</span>
                </div>
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Center: Active Database Title -->
    <div class="hidden md:flex items-center gap-2">
      <span class="text-xs text-zinc-500 uppercase tracking-wider font-semibold">Active Base:</span>
      <span class="text-xs font-mono font-bold text-zinc-200 bg-zinc-900 border border-zinc-800 px-2 py-0.5 rounded-md">
        {{ dbName || 'No Base Loaded' }}
      </span>
    </div>

    <!-- Right: Import/Export, Board Flip & Settings -->
    <div class="flex items-center gap-2">
      <button
        class="p-1.5 rounded-lg bg-zinc-900 hover:bg-zinc-800 text-zinc-300 text-xs flex items-center gap-1 border border-zinc-800 transition-colors"
        title="Import PGN"
        @click="emit('importPGN')"
      >
        <FileUp class="w-3.5 h-3.5 text-zinc-400" />
        <span class="hidden sm:inline">Import</span>
      </button>

      <button
        class="p-1.5 rounded-lg bg-zinc-900 hover:bg-zinc-800 text-zinc-300 text-xs flex items-center gap-1 border border-zinc-800 transition-colors"
        title="Export PGN"
        @click="emit('exportPGN')"
      >
        <FileDown class="w-3.5 h-3.5 text-zinc-400" />
        <span class="hidden sm:inline">Export</span>
      </button>

      <button
        class="p-1.5 rounded-lg bg-zinc-900 hover:bg-zinc-800 text-zinc-300 border border-zinc-800 transition-colors"
        :class="flipped ? 'text-emerald-400 border-emerald-500/50' : 'text-zinc-400'"
        title="Flip Board (F)"
        @click="emit('toggleFlip')"
      >
        <RotateCcw class="w-4 h-4" />
      </button>

      <!-- Quick Dark / Light Toggle -->
      <button
        class="p-1.5 rounded-lg bg-zinc-900 hover:bg-zinc-800 text-zinc-300 border border-zinc-800 transition-colors"
        :title="isCurrentLight ? 'Switch to Dark Mode' : 'Switch to Light Mode'"
        @click="emit('toggleDarkLight')"
      >
        <Sun v-if="isCurrentLight" class="w-4 h-4 text-amber-400" />
        <Moon v-else class="w-4 h-4 text-cyan-400" />
      </button>

      <button
        class="p-1.5 rounded-lg bg-zinc-900 hover:bg-zinc-800 text-zinc-300 border border-zinc-800 transition-colors"
        title="Themes & Appearance"
        @click="showSettingsModal = true"
      >
        <Palette class="w-4 h-4 text-emerald-400" />
      </button>
    </div>

    <!-- Settings Modal -->
    <Teleport to="body">
      <div
        v-if="showSettingsModal"
        class="fixed inset-0 bg-black/60 backdrop-blur-xs flex items-center justify-center z-[100] animate-fade-in p-4"
        @click.self="showSettingsModal = false"
      >
        <div class="bg-zinc-900 border border-zinc-700 w-full max-w-2xl rounded-2xl shadow-2xl p-5 flex flex-col gap-4 max-h-[85vh] overflow-y-auto">
          <div class="flex items-center justify-between border-b border-zinc-800 pb-3">
            <div class="flex items-center gap-2">
              <Palette class="w-5 h-5 text-emerald-400" />
              <h2 class="text-base font-bold text-zinc-100">Appearance, Themes &amp; Piece Sets</h2>
            </div>
            <button
              class="text-zinc-400 hover:text-white text-sm"
              @click="showSettingsModal = false"
            >
              ✕
            </button>
          </div>

          <!-- Application UI Theme Section -->
          <div>
            <div class="flex items-center justify-between mb-2">
              <h3 class="text-xs font-semibold text-zinc-400 uppercase tracking-wider">
                Application UI Theme (Dark &amp; Light)
              </h3>
              <span class="text-[11px] text-zinc-500 font-mono">
                Current: <span class="text-emerald-400 font-semibold">{{ UI_THEMES.find(t => t.id === currentUITheme)?.name || 'Modern Dark' }}</span>
              </span>
            </div>
            <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-2">
              <div
                v-for="theme in UI_THEMES"
                :key="theme.id"
                class="p-3 rounded-xl border cursor-pointer transition-all flex flex-col justify-between gap-2"
                :class="currentUITheme === theme.id
                  ? 'bg-emerald-950/40 border-emerald-500 shadow-md ring-1 ring-emerald-500'
                  : 'bg-zinc-800/60 border-zinc-700/60 hover:bg-zinc-800 hover:border-zinc-600'"
                @click="emit('changeUITheme', theme.id)"
              >
                <!-- Mini UI Theme Preview Swatch -->
                <div
                  class="h-10 rounded-lg p-1.5 flex flex-col justify-between overflow-hidden border shadow-inner"
                  :style="{ backgroundColor: theme.preview.bg, borderColor: theme.preview.border }"
                >
                  <div class="flex items-center justify-between">
                    <!-- Mini Window Bar -->
                    <div class="flex items-center gap-1">
                      <span class="w-1.5 h-1.5 rounded-full" :style="{ backgroundColor: theme.preview.accent }" />
                      <span class="w-6 h-1 rounded-sm" :style="{ backgroundColor: theme.preview.card }" />
                    </div>
                    <span
                      class="text-[9px] font-mono px-1 rounded font-bold uppercase tracking-wider"
                      :style="{ color: theme.preview.text, backgroundColor: theme.preview.card }"
                    >
                      {{ theme.category }}
                    </span>
                  </div>
                  <!-- Mini Text Line Preview -->
                  <div class="flex items-center gap-1.5">
                    <span class="w-10 h-1.5 rounded-sm" :style="{ backgroundColor: theme.preview.accent }" />
                    <span class="w-14 h-1.5 rounded-sm opacity-80" :style="{ backgroundColor: theme.preview.text }" />
                  </div>
                </div>

                <div>
                  <div class="flex items-center justify-between">
                    <span class="font-bold text-xs text-zinc-100 flex items-center gap-1">
                      {{ theme.name }}
                    </span>
                    <span
                      v-if="currentUITheme === theme.id"
                      class="flex items-center gap-0.5 text-[10px] text-emerald-400 font-semibold"
                    >
                      <Check class="w-3 h-3" /> Active
                    </span>
                  </div>
                  <p class="text-[10px] text-zinc-400 mt-0.5 leading-snug line-clamp-1">
                    {{ theme.description }}
                  </p>
                </div>
              </div>
            </div>
          </div>

          <!-- Piece Sets Section -->
          <div>
            <h3 class="text-xs font-semibold text-zinc-400 uppercase tracking-wider mb-2">Piece Sets</h3>
            <div class="grid grid-cols-2 sm:grid-cols-3 gap-2">
              <div
                v-for="set in pieceSets"
                :key="set.id"
                class="p-2.5 rounded-xl border cursor-pointer transition-all flex flex-col justify-between"
                :class="currentPieceSet === set.id
                  ? 'bg-emerald-950/40 border-emerald-500 shadow-md ring-1 ring-emerald-500'
                  : 'bg-zinc-800/60 border-zinc-700/60 hover:bg-zinc-800 hover:border-zinc-600'"
                @click="emit('changePieceSet', set.id)"
              >
                <div>
                  <div class="flex items-center justify-between">
                    <span class="font-bold text-xs text-zinc-200">{{ set.name }}</span>
                    <span
                      v-if="set.isScidNative"
                      class="text-[9px] px-1.5 py-0.5 rounded bg-emerald-900/60 text-emerald-300 font-semibold"
                    >
                      SCID
                    </span>
                  </div>
                  <p class="text-[10px] text-zinc-500 mt-1 leading-snug">{{ set.description }}</p>
                </div>
              </div>
            </div>
          </div>

          <!-- Board Themes Section -->
          <div>
            <h3 class="text-xs font-semibold text-zinc-400 uppercase tracking-wider mb-2">Board Themes</h3>
            <div class="grid grid-cols-2 sm:grid-cols-4 gap-2">
              <div
                v-for="theme in boardThemes"
                :key="theme.id"
                class="p-2 rounded-xl border cursor-pointer transition-all flex items-center gap-3"
                :class="currentThemeId === theme.id
                  ? 'bg-emerald-950/40 border-emerald-500 shadow-md ring-1 ring-emerald-500'
                  : 'bg-zinc-800/60 border-zinc-700/60 hover:bg-zinc-800 hover:border-zinc-600'"
                @click="emit('changeTheme', theme.id)"
              >
                <!-- 2x2 Mini Board Swatch -->
                <div class="w-8 h-8 rounded grid grid-cols-2 grid-rows-2 overflow-hidden shadow-xs border border-black/20 shrink-0">
                  <div :style="{ backgroundColor: theme.light }" />
                  <div :style="{ backgroundColor: theme.dark }" />
                  <div :style="{ backgroundColor: theme.dark }" />
                  <div :style="{ backgroundColor: theme.light }" />
                </div>
                <span class="text-xs font-medium text-zinc-200 truncate">{{ theme.name }}</span>
              </div>
            </div>
          </div>

          <div class="flex justify-end pt-2 border-t border-zinc-800">
            <button
              class="px-4 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white font-medium text-xs transition-colors cursor-pointer"
              @click="showSettingsModal = false"
            >
              Done
            </button>
          </div>
        </div>
      </div>
    </Teleport>
  </header>
</template>
