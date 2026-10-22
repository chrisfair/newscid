<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import Chessboard from './Chessboard.vue'
import {
  Pickaxe,
  X,
  Play,
  CheckCircle,
  AlertCircle,
  RotateCcw,
  Sparkles,
  ChevronRight,
  User,
  Database,
  Search
} from '@lucide/vue'
import { MinePuzzlesFromDatabase, ParseFEN, MakeMove } from '../../wailsjs/go/main/App'
import { puzzleminer } from '../../wailsjs/go/models'

const props = defineProps<{
  dbName?: string
  currentPieceSet: string
  boardTheme: {
    light: string
    dark: string
    highlight: string
  }
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'loadIntoApp', fen: string): void
}>()

// Mining Settings
const playerName = ref('John Smith')
const maxGames = ref(50)
const isMining = ref(false)

// Mined Puzzles
const minedPuzzles = ref<puzzleminer.MinedPuzzle[]>([])
const selectedIndex = ref<number>(-1)

// Board state for active puzzle
const currentFEN = ref('rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1')
const legalMoves = ref<string[]>([])
const lastMove = ref<{ from: string; to: string } | null>(null)
const inCheck = ref(false)
const feedback = ref<{ type: 'success' | 'error' | null; message: string }>({
  type: null,
  message: '',
})
const isSolved = ref(false)

const activePuzzle = computed(() => {
  if (selectedIndex.value >= 0 && selectedIndex.value < minedPuzzles.value.length) {
    return minedPuzzles.value[selectedIndex.value]
  }
  return null
})

const isBoardFlipped = computed(() => {
  return activePuzzle.value?.turn === 'b'
})

onMounted(() => {
  // auto-trigger initial mining if DB open
  startMining()
})

async function startMining() {
  isMining.value = true
  feedback.value = { type: null, message: '' }
  try {
    const list = await MinePuzzlesFromDatabase(playerName.value, maxGames.value)
    minedPuzzles.value = list
    if (list.length > 0) {
      await selectPuzzle(0)
    }
  } catch (err) {
    feedback.value = { type: 'error', message: `Mining error: ${err}` }
  } finally {
    isMining.value = false
  }
}

async function selectPuzzle(idx: number) {
  if (idx < 0 || idx >= minedPuzzles.value.length) return
  selectedIndex.value = idx
  isSolved.value = false
  feedback.value = { type: null, message: '' }

  const p = minedPuzzles.value[idx]
  currentFEN.value = p.startFen
  lastMove.value = null

  const state = await ParseFEN(p.startFen)
  legalMoves.value = state.legalMoves
  inCheck.value = state.inCheck
}

async function handleMove(uci: string) {
  if (!activePuzzle.value || isSolved.value) return

  const solution = activePuzzle.value.bestMoveUci
  if (uci.toLowerCase() === solution.toLowerCase()) {
    // Solved!
    isSolved.value = true
    const nextState = await MakeMove(currentFEN.value, uci)
    const from = uci.slice(0, 2)
    const to = uci.slice(2, 4)
    lastMove.value = { from, to }
    currentFEN.value = nextState.fen
    legalMoves.value = nextState.legalMoves
    inCheck.value = nextState.inCheck

    feedback.value = {
      type: 'success',
      message: `Excellent! You found the winning tactical move: ${activePuzzle.value.bestMoveSan || uci}!`,
    }
  } else {
    feedback.value = {
      type: 'error',
      message: 'Not the winning move. Try again or look for tactical checks and captures!',
    }
  }
}

function loadIntoMainApp() {
  if (activePuzzle.value) {
    emit('loadIntoApp', activePuzzle.value.startFen)
    emit('close')
  }
}
</script>

<template>
  <div class="fixed inset-0 bg-black/80 backdrop-blur-xs flex items-center justify-center z-50 p-4 animate-fade-in select-none">
    <div class="bg-zinc-900 border border-zinc-700/80 rounded-2xl w-full max-w-5xl shadow-2xl overflow-hidden flex flex-col max-h-[92vh]">
      <!-- Header -->
      <div class="flex items-center justify-between px-5 py-3.5 bg-zinc-950 border-b border-zinc-800">
        <div class="flex items-center gap-3">
          <Pickaxe class="w-5 h-5 text-amber-400" />
          <div>
            <h2 class="text-sm font-bold text-zinc-100">Database Puzzle Miner & Builder</h2>
            <p class="text-[11px] text-zinc-500 font-mono">Mine tactical turning points & blunder punishments directly from your games</p>
          </div>
        </div>

        <button
          class="text-zinc-400 hover:text-white p-1 rounded-lg hover:bg-zinc-800 transition-colors cursor-pointer"
          @click="emit('close')"
        >
          <X class="w-5 h-5" />
        </button>
      </div>

      <!-- Main Body -->
      <div class="p-5 flex-1 overflow-y-auto flex flex-col md:flex-row gap-6">
        <!-- Left: Chessboard & Active Puzzle -->
        <div class="flex flex-col items-center gap-3">
          <!-- Active Puzzle Header -->
          <div class="w-full flex items-center justify-between bg-zinc-950 px-4 py-2 rounded-xl border border-zinc-800 text-xs">
            <div class="flex items-center gap-2">
              <span class="w-2.5 h-2.5 rounded-full" :class="activePuzzle?.turn === 'w' ? 'bg-white' : 'bg-zinc-900 border border-zinc-600'" />
              <span class="font-bold text-zinc-200">
                {{ activePuzzle ? `${activePuzzle.white} vs ${activePuzzle.black}` : 'Select a Mined Puzzle' }}
              </span>
            </div>
            <span v-if="activePuzzle" class="text-[11px] font-mono text-amber-400 font-bold">
              {{ activePuzzle.theme }}
            </span>
          </div>

          <!-- Board -->
          <div class="w-80 h-80 sm:w-96 sm:h-96">
            <Chessboard
              :fen="currentFEN"
              :legal-moves="legalMoves"
              :piece-set="currentPieceSet"
              :board-theme="boardTheme"
              :flipped="isBoardFlipped"
              :last-move="lastMove"
              :in-check="inCheck"
              @move="handleMove"
            />
          </div>

          <!-- Feedback Banner -->
          <div
            v-if="feedback.message"
            class="w-full p-2.5 rounded-xl border text-xs font-semibold flex items-center gap-2 transition-all animate-fade-in"
            :class="{
              'bg-emerald-950/60 border-emerald-500/50 text-emerald-300': feedback.type === 'success',
              'bg-rose-950/60 border-rose-500/50 text-rose-300': feedback.type === 'error',
            }"
          >
            <CheckCircle v-if="feedback.type === 'success'" class="w-4 h-4 shrink-0 text-emerald-400" />
            <AlertCircle v-else class="w-4 h-4 shrink-0 text-rose-400" />
            <span>{{ feedback.message }}</span>
          </div>
        </div>

        <!-- Right: Mining Filters & Mined Puzzles List -->
        <div class="flex-1 flex flex-col gap-4">
          <!-- Mining Configuration Controls -->
          <div class="bg-zinc-950 p-3 rounded-xl border border-zinc-800 flex flex-col gap-2.5">
            <div class="flex items-center justify-between text-xs">
              <span class="font-semibold text-zinc-400 flex items-center gap-1.5">
                <Database class="w-3.5 h-3.5 text-emerald-400" />
                <span>Source Database: <strong class="text-zinc-200">{{ dbName || 'Current Database' }}</strong></span>
              </span>
            </div>

            <div class="grid grid-cols-2 gap-2 text-xs">
              <div>
                <label class="text-[10px] text-zinc-500 uppercase font-semibold">Filter by Player</label>
                <input
                  v-model="playerName"
                  type="text"
                  placeholder="e.g. John Smith (or leave empty)"
                  class="w-full bg-zinc-900 border border-zinc-700 rounded-lg p-1.5 text-xs text-zinc-100 placeholder-zinc-500 focus:outline-hidden focus:border-amber-500"
                />
              </div>

              <div>
                <label class="text-[10px] text-zinc-500 uppercase font-semibold">Max Games to Scan</label>
                <select
                  v-model.number="maxGames"
                  class="w-full bg-zinc-900 border border-zinc-700 rounded-lg p-1.5 text-xs text-zinc-100 focus:outline-hidden focus:border-amber-500"
                >
                  <option :value="25">25 Games</option>
                  <option :value="50">50 Games</option>
                  <option :value="100">100 Games</option>
                  <option :value="200">200 Games</option>
                </select>
              </div>
            </div>

            <button
              class="w-full py-2 rounded-xl bg-amber-500 hover:bg-amber-400 text-zinc-950 font-bold text-xs flex items-center justify-center gap-1.5 shadow-md transition-all cursor-pointer"
              :class="{ 'opacity-70 cursor-wait': isMining }"
              :disabled="isMining"
              @click="startMining"
            >
              <Pickaxe class="w-4 h-4" />
              <span>{{ isMining ? 'Scanning Database for Tactics...' : 'Mine Tactical Puzzles' }}</span>
            </button>
          </div>

          <!-- Mined Puzzles Results List -->
          <div class="bg-zinc-950 p-3 rounded-xl border border-zinc-800 flex-1 overflow-y-auto min-h-48 max-h-72 flex flex-col gap-2">
            <div class="flex items-center justify-between text-xs">
              <span class="font-bold text-zinc-300 uppercase tracking-wider text-[10px]">
                Mined Puzzles ({{ minedPuzzles.length }})
              </span>
              <span v-if="minedPuzzles.length > 0" class="text-[11px] text-emerald-400 font-mono">
                Click any puzzle to play
              </span>
            </div>

            <div v-if="minedPuzzles.length > 0" class="space-y-1.5">
              <div
                v-for="(p, idx) in minedPuzzles"
                :key="p.id"
                class="p-2 rounded-lg border transition-all cursor-pointer flex items-center justify-between"
                :class="selectedIndex === idx ? 'bg-amber-500/10 border-amber-500/60' : 'bg-zinc-900 border-zinc-800 hover:bg-zinc-800/80'"
                @click="selectPuzzle(idx)"
              >
                <div class="flex items-center gap-2 min-w-0">
                  <span class="w-2 h-2 rounded-full" :class="p.turn === 'w' ? 'bg-white' : 'bg-zinc-900 border border-zinc-600'" />
                  <div class="truncate">
                    <p class="text-xs font-semibold text-zinc-200 truncate">{{ p.white }} vs {{ p.black }}</p>
                    <p class="text-[10px] text-zinc-500 truncate">{{ p.description }}</p>
                  </div>
                </div>

                <div class="flex items-center gap-2 shrink-0">
                  <span class="px-2 py-0.5 rounded text-[10px] font-semibold bg-zinc-800 text-amber-300">
                    {{ p.theme }}
                  </span>
                  <ChevronRight class="w-4 h-4 text-zinc-500" />
                </div>
              </div>
            </div>

            <div v-else-if="!isMining" class="flex-1 flex flex-col items-center justify-center text-zinc-500 text-xs italic gap-1 p-4">
              <Sparkles class="w-6 h-6 text-zinc-600" />
              <span>No tactical turning points mined yet. Click "Mine Tactical Puzzles" to scan.</span>
            </div>
          </div>

          <!-- Bottom: Load into main workspace -->
          <button
            class="w-full py-2 rounded-xl bg-zinc-950 hover:bg-zinc-800 border border-zinc-800 text-zinc-300 text-xs font-medium flex items-center justify-center gap-2 cursor-pointer transition-colors mt-auto"
            :disabled="!activePuzzle"
            @click="loadIntoMainApp"
          >
            <span>Analyze Tactical Moment in Main Workspace</span>
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
