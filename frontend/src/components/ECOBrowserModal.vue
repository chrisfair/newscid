<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import {
  BookOpen,
  Search,
  X,
  Play,
  Database,
  Filter,
  Layers,
  ChevronRight
} from '@lucide/vue'
import { GetECOSections, SearchECO } from '../../wailsjs/go/main/App'
import { eco } from '../../wailsjs/go/models'

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'playOpening', moves: string): void
  (e: 'searchEcoInDb', ecoCode: string): void
}>()

const activeSection = ref('ALL')
const searchQuery = ref('')
const sections = ref<eco.SectionSummary[]>([])
const entries = ref<eco.Entry[]>([])
const isLoading = ref(false)
const selectedEntry = ref<eco.Entry | null>(null)

async function loadSections() {
  try {
    const s = await GetECOSections()
    if (s && s.length > 0) {
      sections.value = s
    }
  } catch (err) {
    console.error('Failed to load ECO sections:', err)
  }
}

async function performSearch() {
  isLoading.value = true
  try {
    const res = await SearchECO(activeSection.value, searchQuery.value, 150)
    entries.value = res || []
    if (entries.value.length > 0 && !selectedEntry.value) {
      selectedEntry.value = entries.value[0]
    }
  } catch (err) {
    console.error('Failed to search ECO entries:', err)
  } finally {
    isLoading.value = false
  }
}

watch([activeSection, searchQuery], () => {
  performSearch()
})

onMounted(async () => {
  await loadSections()
  await performSearch()
})

function handlePlayOnBoard(moves: string) {
  emit('playOpening', moves)
  emit('close')
}

function handleSearchInDb(code: string) {
  emit('searchEcoInDb', code)
  emit('close')
}
</script>

<template>
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/75 backdrop-blur-xs p-4">
    <div
      class="bg-zinc-900 border border-zinc-700 rounded-xl shadow-2xl w-full max-w-4xl max-h-[88vh] flex flex-col overflow-hidden text-zinc-200 animate-in fade-in zoom-in-95 duration-150"
      @keydown.esc="emit('close')"
    >
      <!-- Header -->
      <div class="flex items-center justify-between px-5 py-3.5 bg-zinc-950 border-b border-zinc-800">
        <div class="flex items-center gap-2.5">
          <div class="w-8 h-8 rounded-lg bg-emerald-500/20 border border-emerald-500/40 flex items-center justify-center text-emerald-400">
            <BookOpen class="w-4 h-4" />
          </div>
          <div>
            <h2 class="font-bold text-base text-zinc-100 flex items-center gap-2">
              ECO Opening Classification Browser
              <span class="text-xs px-2 py-0.5 rounded-full bg-zinc-800 border border-zinc-700 text-zinc-400 font-mono font-normal">
                Scid vs PC Taxonomy
              </span>
            </h2>
            <p class="text-xs text-zinc-400 mt-0.5">
              Explore the Encyclopedia of Chess Openings (A00–E99) with full move trees and database integration.
            </p>
          </div>
        </div>
        <button
          class="p-1.5 rounded-lg text-zinc-400 hover:text-white hover:bg-zinc-800 transition-colors cursor-pointer"
          @click="emit('close')"
        >
          <X class="w-4 h-4" />
        </button>
      </div>

      <!-- Section Tabs & Search Filter -->
      <div class="p-3 bg-zinc-950/70 border-b border-zinc-800 flex flex-wrap items-center justify-between gap-3">
        <!-- Section Tabs -->
        <div class="flex items-center gap-1.5 overflow-x-auto text-xs">
          <button
            class="px-3 py-1.5 rounded-lg font-medium transition-colors cursor-pointer"
            :class="activeSection === 'ALL' ? 'bg-emerald-600 text-white shadow-xs' : 'bg-zinc-800/80 text-zinc-400 hover:bg-zinc-800 hover:text-zinc-200'"
            @click="activeSection = 'ALL'"
          >
            All Openings
          </button>
          <button
            v-for="s in sections"
            :key="s.letter"
            class="px-2.5 py-1.5 rounded-lg font-medium transition-colors cursor-pointer flex items-center gap-1.5"
            :class="activeSection === s.letter ? 'bg-emerald-600 text-white shadow-xs' : 'bg-zinc-800/80 text-zinc-400 hover:bg-zinc-800 hover:text-zinc-200'"
            @click="activeSection = s.letter"
            :title="s.description"
          >
            <span class="font-bold">{{ s.letter }}</span>
            <span class="text-[10px] opacity-75 font-mono">({{ s.count }})</span>
          </button>
        </div>

        <!-- Search Bar -->
        <div class="relative w-72">
          <Search class="w-3.5 h-3.5 absolute left-3 top-1/2 -translate-y-1/2 text-zinc-500" />
          <input
            v-model="searchQuery"
            type="text"
            placeholder="Search ECO code, name, or moves..."
            class="w-full bg-zinc-900 border border-zinc-700 rounded-lg pl-8 pr-3 py-1.5 text-xs text-zinc-200 placeholder-zinc-500 focus:outline-hidden focus:border-emerald-500"
          />
        </div>
      </div>

      <!-- Main Content: Left List, Right Detail -->
      <div class="flex-1 flex overflow-hidden">
        <!-- Openings List -->
        <div class="w-7/12 border-r border-zinc-800 flex flex-col overflow-hidden bg-zinc-900/60">
          <div class="flex-1 overflow-y-auto divide-y divide-zinc-800/40">
            <div
              v-if="isLoading"
              class="flex items-center justify-center p-8 text-xs text-zinc-500"
            >
              Loading ECO openings...
            </div>
            <div
              v-else-if="entries.length === 0"
              class="flex flex-col items-center justify-center p-8 text-center text-xs text-zinc-500 space-y-1"
            >
              <span>No openings match your criteria.</span>
              <span class="text-[11px] text-zinc-600">Try adjusting your search query or section tab.</span>
            </div>
            <div
              v-for="entry in entries"
              :key="entry.code + entry.name"
              class="p-2.5 hover:bg-zinc-800/60 transition-colors cursor-pointer flex items-center justify-between text-xs group"
              :class="{ 'bg-emerald-950/30 border-l-2 border-emerald-500': selectedEntry?.code === entry.code && selectedEntry?.name === entry.name }"
              @click="selectedEntry = entry"
            >
              <div class="flex-1 min-w-0 pr-3">
                <div class="flex items-center gap-2">
                  <span class="font-bold font-mono px-1.5 py-0.5 rounded bg-zinc-800 border border-zinc-700 text-emerald-400 text-[11px]">
                    {{ entry.code }}
                  </span>
                  <span class="font-medium text-zinc-200 truncate group-hover:text-white">
                    {{ entry.name }}
                  </span>
                </div>
                <div class="text-[11px] text-zinc-400 font-mono mt-1 truncate">
                  {{ entry.moves || '1. (irregular/uncommon moves)' }}
                </div>
              </div>
              <ChevronRight class="w-4 h-4 text-zinc-600 group-hover:text-zinc-300 shrink-0" />
            </div>
          </div>

          <div class="p-2 bg-zinc-950/60 border-t border-zinc-800 text-[11px] text-zinc-500 flex justify-between items-center px-4">
            <span>Showing {{ entries.length }} results</span>
            <span v-if="entries.length >= 150" class="text-amber-400/80">Capped at 150 items</span>
          </div>
        </div>

        <!-- Detail Preview Panel -->
        <div class="w-5/12 p-5 flex flex-col justify-between overflow-y-auto bg-zinc-950/30">
          <div v-if="selectedEntry" class="space-y-4">
            <div>
              <div class="flex items-center gap-2">
                <span class="text-lg font-bold font-mono px-2 py-0.5 rounded-md bg-emerald-500/20 border border-emerald-500/40 text-emerald-400">
                  {{ selectedEntry.code }}
                </span>
                <span class="text-xs px-2 py-0.5 rounded bg-zinc-800 text-zinc-400">
                  Section {{ selectedEntry.code[0] }}
                </span>
              </div>
              <h3 class="text-base font-bold text-zinc-100 mt-2">
                {{ selectedEntry.name }}
              </h3>
            </div>

            <!-- Moves Sequence Card -->
            <div class="p-3 rounded-lg bg-zinc-900 border border-zinc-800 space-y-1.5">
              <span class="text-[11px] font-semibold text-zinc-400 uppercase tracking-wider block">
                Standard Move Line
              </span>
              <div class="font-mono text-xs text-emerald-300 leading-relaxed break-words bg-zinc-950/60 p-2.5 rounded border border-zinc-800">
                {{ selectedEntry.moves || '1. (any starting moves)' }}
              </div>
            </div>

            <!-- Section Description -->
            <div class="text-xs text-zinc-400 space-y-1">
              <span class="font-semibold text-zinc-300">Classification Context:</span>
              <p v-if="selectedEntry.code.startsWith('A')">
                Flank openings & irregular beginnings, including English, Réti, Bird, Benoni, Dutch, and King's Indian Attack.
              </p>
              <p v-else-if="selectedEntry.code.startsWith('B')">
                Semi-open games other than French: Sicilian Defense, Caro-Kann, Pirc, Modern, Scandinavian, and Alekhine.
              </p>
              <p v-else-if="selectedEntry.code.startsWith('C')">
                Open games with 1.e4 e5 (Ruy Lopez, Italian, Petrov, King's Gambit) and French Defense (1.e4 e6).
              </p>
              <p v-else-if="selectedEntry.code.startsWith('D')">
                Closed & Semi-closed games with 1.d4 d5 (Queen's Gambit, Slav), and Grünfeld Defense.
              </p>
              <p v-else-if="selectedEntry.code.startsWith('E')">
                Indian Defenses: King's Indian, Nimzo-Indian, Queen's Indian, Catalan, and Bogo-Indian.
              </p>
            </div>
          </div>

          <div v-else class="flex items-center justify-center h-full text-xs text-zinc-500 italic">
            Select an opening to view details
          </div>

          <!-- Bottom Actions -->
          <div v-if="selectedEntry" class="pt-4 border-t border-zinc-800 space-y-2">
            <button
              class="w-full px-4 py-2 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white font-medium text-xs shadow-md transition-colors cursor-pointer flex items-center justify-center gap-2"
              @click="handlePlayOnBoard(selectedEntry.moves)"
            >
              <Play class="w-3.5 h-3.5 fill-current" />
              <span>Play Line on Board</span>
            </button>

            <button
              class="w-full px-4 py-2 rounded-lg bg-zinc-800 hover:bg-zinc-700 text-zinc-200 font-medium text-xs transition-colors cursor-pointer flex items-center justify-center gap-2"
              @click="handleSearchInDb(selectedEntry.code)"
            >
              <Database class="w-3.5 h-3.5 text-emerald-400" />
              <span>Search Database for {{ selectedEntry.code }}</span>
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
