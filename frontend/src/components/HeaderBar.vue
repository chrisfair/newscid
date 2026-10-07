<script setup lang="ts">
import { ref } from 'vue'
import {
  FolderOpen,
  PlusCircle,
  FileDown,
  FileUp,
  Settings,
  Palette,
  RotateCcw,
  BookOpen,
  Database
} from '@lucide/vue'

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

const props = defineProps<{
  dbName?: string
  systemBases: string[]
  pieceSets: PieceSetOption[]
  boardThemes: BoardThemeOption[]
  currentPieceSet: string
  currentThemeId: string
  flipped: boolean
}>()

const emit = defineEmits<{
  (e: 'openDB'): void
  (e: 'createDB'): void
  (e: 'openSystemBase', path: string): void
  (e: 'importPGN'): void
  (e: 'exportPGN'): void
  (e: 'changePieceSet', id: string): void
  (e: 'changeTheme', id: string): void
  (e: 'toggleFlip'): void
}>()

const showSettingsModal = ref(false)
const showSystemBasesDropdown = ref(false)
</script>

<template>
  <header class="app-header bg-zinc-950 border-b border-zinc-800 px-4 py-2 flex items-center justify-between z-40 select-none">
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

        <!-- System Bases Dropdown -->
        <div v-if="systemBases.length > 0" class="relative">
          <button
            class="header-btn flex items-center gap-1 px-2.5 py-1 rounded-lg bg-zinc-900 hover:bg-zinc-800 text-zinc-300 text-xs font-medium border border-zinc-800 transition-colors"
            @click="showSystemBasesDropdown = !showSystemBasesDropdown"
          >
            <BookOpen class="w-3.5 h-3.5 text-blue-400" />
            <span>Sample Bases</span>
          </button>

          <div
            v-if="showSystemBasesDropdown"
            class="absolute top-full left-0 mt-1 w-48 bg-zinc-900 border border-zinc-700 rounded-lg shadow-2xl py-1 z-50 text-xs"
          >
            <button
              v-for="base in systemBases"
              :key="base"
              class="w-full text-left px-3 py-1.5 hover:bg-zinc-800 text-zinc-200 capitalize flex items-center justify-between"
              @click="emit('openSystemBase', base); showSystemBasesDropdown = false"
            >
              <span>{{ base.split('/').pop() }}</span>
              <Database class="w-3 h-3 text-zinc-500" />
            </button>
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

      <button
        class="p-1.5 rounded-lg bg-zinc-900 hover:bg-zinc-800 text-zinc-300 border border-zinc-800 transition-colors"
        title="Themes & Piece Sets"
        @click="showSettingsModal = true"
      >
        <Palette class="w-4 h-4 text-emerald-400" />
      </button>
    </div>

    <!-- Settings Modal -->
    <div
      v-if="showSettingsModal"
      class="fixed inset-0 bg-black/60 backdrop-blur-xs flex items-center justify-center z-50 animate-fade-in p-4"
      @click.self="showSettingsModal = false"
    >
      <div class="bg-zinc-900 border border-zinc-700 w-full max-w-2xl rounded-2xl shadow-2xl p-5 flex flex-col gap-4 max-h-[85vh] overflow-y-auto">
        <div class="flex items-center justify-between border-b border-zinc-800 pb-3">
          <div class="flex items-center gap-2">
            <Palette class="w-5 h-5 text-emerald-400" />
            <h2 class="text-base font-bold text-zinc-100">Piece Sets & Board Themes</h2>
          </div>
          <button
            class="text-zinc-400 hover:text-white text-sm"
            @click="showSettingsModal = false"
          >
            ✕
          </button>
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
  </header>
</template>
