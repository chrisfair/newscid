<script setup lang="ts">
import { computed } from 'vue'
import {
  ChevronLeft,
  ChevronRight,
  ChevronsLeft,
  ChevronsRight,
  Play,
  Pause,
  RotateCcw,
  Copy,
  FileText
} from '@lucide/vue'

export interface FlatMove {
  index: number
  ply: number
  moveNumber: number
  isWhite: boolean
  san: string
  uci: string
  fen: string
  comment?: string
  nags?: number[]
  variationLevel: number
}

const props = withDefaults(defineProps<{
  moves: FlatMove[]
  currentPly: number
  isPlaying?: boolean
  result?: string
}>(), {
  moves: () => [],
  currentPly: 0,
  isPlaying: false,
  result: '*',
})

const emit = defineEmits<{
  (e: 'jump', ply: number): void
  (e: 'first'): void
  (e: 'prev'): void
  (e: 'next'): void
  (e: 'last'): void
  (e: 'togglePlay'): void
  (e: 'copyPGN'): void
}>()

// Group moves into pairs for standard score-sheet layout
const movePairs = computed(() => {
  const pairs: { moveNumber: number; white?: FlatMove; black?: FlatMove }[] = []
  const mainMoves = props.moves.filter(m => m.variationLevel === 0)

  let currentPair: { moveNumber: number; white?: FlatMove; black?: FlatMove } | null = null

  for (const m of mainMoves) {
    if (m.isWhite) {
      if (currentPair) {
        pairs.push(currentPair)
      }
      currentPair = { moveNumber: m.moveNumber, white: m }
    } else {
      if (!currentPair) {
        currentPair = { moveNumber: m.moveNumber, black: m }
      } else {
        currentPair.black = m
        pairs.push(currentPair)
        currentPair = null
      }
    }
  }

  if (currentPair) {
    pairs.push(currentPair)
  }

  return pairs
})

function formatNAG(nag: number): string {
  switch (nag) {
    case 1: return '!'
    case 2: return '?'
    case 3: return '!!'
    case 4: return '??'
    case 5: return '!?'
    case 6: return '?!'
    case 10: return '='
    case 14: return '+='
    case 15: return '=+'
    case 16: return '±'
    case 17: return '∓'
    case 18: return '+-'
    case 19: return '-+'
    default: return `$${nag}`
  }
}
</script>

<template>
  <div class="move-notation flex flex-col h-full bg-zinc-900 border border-zinc-800 rounded-xl overflow-hidden shadow-lg">
    <!-- Move Navigation Header -->
    <div class="nav-toolbar flex items-center justify-between p-2 bg-zinc-950 border-b border-zinc-800">
      <div class="flex items-center gap-1">
        <button
          class="nav-btn p-1.5 rounded-lg text-zinc-400 hover:text-white hover:bg-zinc-800 transition-colors"
          title="First Move (Home)"
          @click="emit('first')"
        >
          <ChevronsLeft class="w-4 h-4" />
        </button>
        <button
          class="nav-btn p-1.5 rounded-lg text-zinc-400 hover:text-white hover:bg-zinc-800 transition-colors"
          title="Previous Move (Left Arrow)"
          @click="emit('prev')"
        >
          <ChevronLeft class="w-4 h-4" />
        </button>
        <button
          class="nav-btn p-1.5 rounded-lg text-zinc-400 hover:text-white hover:bg-zinc-800 transition-colors"
          :title="isPlaying ? 'Pause' : 'Autoplay (Space)'"
          @click="emit('togglePlay')"
        >
          <Pause v-if="isPlaying" class="w-4 h-4 text-emerald-400" />
          <Play v-else class="w-4 h-4" />
        </button>
        <button
          class="nav-btn p-1.5 rounded-lg text-zinc-400 hover:text-white hover:bg-zinc-800 transition-colors"
          title="Next Move (Right Arrow)"
          @click="emit('next')"
        >
          <ChevronRight class="w-4 h-4" />
        </button>
        <button
          class="nav-btn p-1.5 rounded-lg text-zinc-400 hover:text-white hover:bg-zinc-800 transition-colors"
          title="Last Move (End)"
          @click="emit('last')"
        >
          <ChevronsRight class="w-4 h-4" />
        </button>
      </div>

      <div class="flex items-center gap-1">
        <button
          class="p-1.5 rounded-lg text-zinc-400 hover:text-white hover:bg-zinc-800 transition-colors text-xs flex items-center gap-1"
          title="Copy PGN"
          @click="emit('copyPGN')"
        >
          <Copy class="w-3.5 h-3.5" />
          <span>PGN</span>
        </button>
      </div>
    </div>

    <!-- Moves Scrollable Container -->
    <div class="moves-container flex-1 p-3 overflow-y-auto font-mono text-sm space-y-1">
      <div v-if="movePairs.length === 0" class="text-zinc-500 text-xs italic text-center py-6">
        No moves played yet. Play a move on the board or load a game.
      </div>

      <div
        v-for="pair in movePairs"
        :key="pair.moveNumber"
        class="move-row flex items-baseline gap-2 py-0.5 px-2 rounded-md hover:bg-zinc-800/40"
      >
        <!-- Move number -->
        <span class="move-num text-zinc-500 text-xs w-7 select-none text-right">
          {{ pair.moveNumber }}.
        </span>

        <!-- White Move -->
        <div class="flex-1 flex items-center gap-1">
          <button
            v-if="pair.white"
            class="move-btn px-2 py-0.5 rounded text-left font-semibold transition-all cursor-pointer"
            :class="pair.white.ply === currentPly
              ? 'bg-emerald-600 text-white shadow-sm'
              : 'text-zinc-200 hover:bg-zinc-700/60'"
            @click="emit('jump', pair.white.ply)"
          >
            {{ pair.white.san }}
            <span
              v-for="nag in pair.white.nags"
              :key="nag"
              class="nag text-emerald-400 font-bold ml-0.5 text-xs"
            >
              {{ formatNAG(nag) }}
            </span>
          </button>
        </div>

        <!-- Black Move -->
        <div class="flex-1 flex items-center gap-1">
          <button
            v-if="pair.black"
            class="move-btn px-2 py-0.5 rounded text-left font-semibold transition-all cursor-pointer"
            :class="pair.black.ply === currentPly
              ? 'bg-emerald-600 text-white shadow-sm'
              : 'text-zinc-200 hover:bg-zinc-700/60'"
            @click="emit('jump', pair.black.ply)"
          >
            {{ pair.black.san }}
            <span
              v-for="nag in pair.black.nags"
              :key="nag"
              class="nag text-emerald-400 font-bold ml-0.5 text-xs"
            >
              {{ formatNAG(nag) }}
            </span>
          </button>
        </div>
      </div>

      <!-- Result at the end -->
      <div v-if="result && result !== '*'" class="text-center font-bold text-zinc-300 py-3 tracking-widest text-sm border-t border-zinc-800/60 mt-3">
        {{ result }}
      </div>
    </div>
  </div>
</template>

<style scoped>
.moves-container::-webkit-scrollbar {
  width: 6px;
}
.moves-container::-webkit-scrollbar-thumb {
  background-color: rgba(255, 255, 255, 0.1);
  border-radius: 4px;
}
</style>
