<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import Chessboard from './Chessboard.vue'
import { Award, X, ChevronLeft, ChevronRight, Shuffle, Lightbulb, RotateCcw, CheckCircle, AlertTriangle, Flame } from '@lucide/vue'
import { LoadTacticsPuzzle, ParseFEN, MakeMove } from '../../wailsjs/go/main/App'
import { main } from '../../wailsjs/go/models'

const props = defineProps<{
  systemBases: string[]
  currentPieceSet: string
  boardTheme: {
    light: string
    dark: string
    highlight: string
  }
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'loadIntoApp', fen: string, pgn?: string): void
}>()

interface PuzzleData {
  gameId: number
  totalGames: number
  baseName: string
  event: string
  white: string
  black: string
  startFen: string
  moves: string[]
  sanMoves: string[]
  turn: string
  theme: string
}

// State
const selectedBase = ref('/usr/share/scid/bases/tactics')
const currentPuzzleId = ref(1)
const puzzle = ref<PuzzleData | null>(null)
const currentFEN = ref('')
const legalMoves = ref<string[]>([])
const lastMove = ref<{ from: string; to: string } | null>(null)
const inCheck = ref(false)
const moveIndex = ref(0) // tracks how many solution moves have been played

// Gamification & Feedback
const streak = ref(0)
const score = ref(0)
const solvedCount = ref(0)
const feedback = ref<{ type: 'success' | 'error' | 'hint' | null; message: string }>({
  type: null,
  message: '',
})
const isSolved = ref(false)
const showHint = ref(false)

const puzzleTurn = computed(() => {
  return puzzle.value?.turn === 'b' ? 'b' : 'w'
})

const isBoardFlipped = computed(() => {
  return puzzleTurn.value === 'b'
})

onMounted(() => {
  if (props.systemBases.length > 0) {
    selectedBase.value = props.systemBases[0]
  }
  loadPuzzle(1)
})

async function loadPuzzle(id: number) {
  try {
    feedback.value = { type: null, message: '' }
    isSolved.value = false
    showHint.value = false
    moveIndex.value = 0

    const p = await LoadTacticsPuzzle(selectedBase.value, id) as PuzzleData
    puzzle.value = p
    currentPuzzleId.value = p.gameId

    currentFEN.value = p.startFen
    lastMove.value = null

    const state = await ParseFEN(p.startFen)
    legalMoves.value = state.legalMoves
    inCheck.value = state.inCheck
  } catch (err) {
    feedback.value = { type: 'error', message: `Failed to load puzzle: ${err}` }
  }
}

async function handleMove(uci: string) {
  if (!puzzle.value || isSolved.value) return

  const expectedMove = puzzle.value.moves[moveIndex.value]

  // Compare move
  if (uci.toLowerCase() === expectedMove?.toLowerCase()) {
    // Correct move!
    feedback.value = { type: 'success', message: 'Correct move!' }

    const nextState = await MakeMove(currentFEN.value, uci)
    const from = uci.slice(0, 2)
    const to = uci.slice(2, 4)
    lastMove.value = { from, to }
    currentFEN.value = nextState.fen
    legalMoves.value = nextState.legalMoves
    inCheck.value = nextState.inCheck
    moveIndex.value++

    // Check if puzzle is complete
    if (moveIndex.value >= puzzle.value.moves.length) {
      finishPuzzle()
      return
    }

    // Play opponent's automatic reply if there are more moves
    const opponentMove = puzzle.value.moves[moveIndex.value]
    if (opponentMove) {
      setTimeout(async () => {
        try {
          const oppState = await MakeMove(currentFEN.value, opponentMove)
          const oppFrom = opponentMove.slice(0, 2)
          const oppTo = opponentMove.slice(2, 4)
          lastMove.value = { from: oppFrom, to: oppTo }
          currentFEN.value = oppState.fen
          legalMoves.value = oppState.legalMoves
          inCheck.value = oppState.inCheck
          moveIndex.value++

          if (moveIndex.value >= puzzle.value!.moves.length) {
            finishPuzzle()
          }
        } catch (err) {
          console.error('Opponent move error:', err)
        }
      }, 450)
    }
  } else {
    // Incorrect move
    feedback.value = { type: 'error', message: 'Incorrect move! Try again.' }
    streak.value = 0
  }
}

function finishPuzzle() {
  isSolved.value = true
  feedback.value = { type: 'success', message: 'Puzzle Solved! Excellent tactic! (+10 pts)' }
  streak.value++
  score.value += 10
  solvedCount.value++
}

function handleNextPuzzle() {
  if (!puzzle.value) return
  const nextId = currentPuzzleId.value < puzzle.value.totalGames ? currentPuzzleId.value + 1 : 1
  loadPuzzle(nextId)
}

function handlePrevPuzzle() {
  if (!puzzle.value) return
  const prevId = currentPuzzleId.value > 1 ? currentPuzzleId.value - 1 : puzzle.value.totalGames
  loadPuzzle(prevId)
}

function handleRandomPuzzle() {
  if (!puzzle.value) return
  const randId = Math.floor(Math.random() * puzzle.value.totalGames) + 1
  loadPuzzle(randId)
}

function restartCurrentPuzzle() {
  if (puzzle.value) {
    loadPuzzle(puzzle.value.gameId)
  }
}

function triggerHint() {
  if (!puzzle.value || isSolved.value) return
  const nextMoveSAN = puzzle.value.sanMoves[moveIndex.value] || puzzle.value.moves[moveIndex.value]
  showHint.value = true
  feedback.value = { type: 'hint', message: `Hint: The winning move begins with ${nextMoveSAN.slice(0, 2)}... (Solution: ${nextMoveSAN})` }
}

function handleBaseChange(base: string) {
  selectedBase.value = base
  loadPuzzle(1)
}

function loadIntoMainApp() {
  if (!puzzle.value) return
  emit('loadIntoApp', puzzle.value.startFen)
  emit('close')
}
</script>

<template>
  <div class="fixed inset-0 bg-black/75 backdrop-blur-xs flex items-center justify-center z-50 p-4 animate-fade-in select-none">
    <div class="bg-zinc-900 border border-zinc-700/80 rounded-2xl w-full max-w-4xl shadow-2xl overflow-hidden flex flex-col max-h-[92vh]">
      <!-- Header -->
      <div class="flex items-center justify-between px-5 py-3.5 bg-zinc-950 border-b border-zinc-800">
        <div class="flex items-center gap-2">
          <Award class="w-5 h-5 text-blue-400" />
          <h2 class="text-sm font-bold text-zinc-100">Tactics Puzzle Trainer</h2>
          <span class="text-[11px] text-zinc-500 font-mono">Scid vs. PC Tactics Bases</span>
        </div>

        <!-- Streak & Score Badges -->
        <div class="flex items-center gap-3">
          <div class="flex items-center gap-1.5 px-3 py-1 rounded-full bg-amber-500/10 border border-amber-500/30 text-amber-300 text-xs font-semibold">
            <Flame class="w-4 h-4 text-amber-400" />
            <span>Streak: {{ streak }}</span>
          </div>
          <div class="flex items-center gap-1.5 px-3 py-1 rounded-full bg-emerald-500/10 border border-emerald-500/30 text-emerald-300 text-xs font-semibold">
            <CheckCircle class="w-4 h-4 text-emerald-400" />
            <span>Solved: {{ solvedCount }} ({{ score }} pts)</span>
          </div>
          <button
            class="text-zinc-400 hover:text-white p-1 rounded-lg hover:bg-zinc-800 transition-colors"
            @click="emit('close')"
          >
            <X class="w-5 h-5" />
          </button>
        </div>
      </div>

      <!-- Main Body -->
      <div class="p-5 flex-1 overflow-y-auto flex flex-col md:flex-row gap-6">
        <!-- Left: Chessboard -->
        <div class="flex flex-col items-center gap-3">
          <!-- Turn Banner -->
          <div
            class="w-full flex items-center justify-between px-4 py-2 rounded-xl border text-xs font-semibold"
            :class="puzzleTurn === 'w' ? 'bg-zinc-950 text-zinc-100 border-zinc-800' : 'bg-zinc-800 text-white border-zinc-700'"
          >
            <div class="flex items-center gap-2">
              <span class="w-3 h-3 rounded-full" :class="puzzleTurn === 'w' ? 'bg-white border border-zinc-400' : 'bg-zinc-900 border border-zinc-600'" />
              <span>{{ puzzleTurn === 'w' ? 'White' : 'Black' }} to move and win!</span>
            </div>
            <span class="text-[11px] text-zinc-400 font-mono">Move {{ Math.floor(moveIndex / 2) + 1 }} of {{ Math.ceil((puzzle?.moves.length || 1) / 2) }}</span>
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
              'bg-amber-950/60 border-amber-500/50 text-amber-300': feedback.type === 'hint',
            }"
          >
            <CheckCircle v-if="feedback.type === 'success'" class="w-4 h-4 shrink-0 text-emerald-400" />
            <AlertTriangle v-else-if="feedback.type === 'error'" class="w-4 h-4 shrink-0 text-rose-400" />
            <Lightbulb v-else class="w-4 h-4 shrink-0 text-amber-400" />
            <span>{{ feedback.message }}</span>
          </div>
        </div>

        <!-- Right: Puzzle Controls & Database -->
        <div class="flex-1 flex flex-col gap-4">
          <!-- Database Selector -->
          <div class="bg-zinc-950 p-3 rounded-xl border border-zinc-800 flex flex-col gap-1.5">
            <label class="text-xs font-semibold text-zinc-400">Puzzle Database Collection</label>
            <div class="grid grid-cols-2 gap-2">
              <button
                v-for="b in [
                  { label: 'Tactics (928)', path: '/usr/share/scid/bases/tactics' },
                  { label: 'Mate in 1 (1,212)', path: '/usr/share/scid/bases/matein1' },
                  { label: 'Mate in 2 (15,623)', path: '/usr/share/scid/bases/matein2' },
                  { label: 'Mate in 3 (7,862)', path: '/usr/share/scid/bases/matein3' },
                  { label: 'Mate in 4+ (3,417)', path: '/usr/share/scid/bases/matein4andmore' },
                  { label: 'Endings (67)', path: '/usr/share/scid/bases/endings' },
                ]"
                :key="b.path"
                class="px-2.5 py-1.5 rounded-lg border text-xs font-medium text-left truncate transition-all cursor-pointer"
                :class="selectedBase.includes(b.path) ? 'bg-blue-600/30 border-blue-500 text-blue-200' : 'bg-zinc-900 border-zinc-800 text-zinc-400 hover:bg-zinc-800'"
                @click="handleBaseChange(b.path)"
              >
                {{ b.label }}
              </button>
            </div>
          </div>

          <!-- Puzzle Metadata Card -->
          <div class="bg-zinc-950 p-4 rounded-xl border border-zinc-800 flex flex-col gap-2">
            <div class="flex items-center justify-between">
              <span class="text-xs font-bold text-zinc-200 uppercase tracking-wider">
                Puzzle #{{ puzzle?.gameId }} of {{ puzzle?.totalGames }}
              </span>
              <span class="text-[11px] font-mono text-zinc-500">{{ puzzle?.baseName }}</span>
            </div>

            <div class="space-y-1 text-xs text-zinc-400">
              <div v-if="puzzle?.event && puzzle.event !== '?'" class="flex justify-between">
                <span class="text-zinc-500">Event / Theme:</span>
                <span class="text-zinc-200 font-medium">{{ puzzle.event }}</span>
              </div>
              <div v-if="puzzle?.white && puzzle.white !== '?'" class="flex justify-between">
                <span class="text-zinc-500">White:</span>
                <span class="text-zinc-200">{{ puzzle.white }}</span>
              </div>
              <div v-if="puzzle?.black && puzzle.black !== '?'" class="flex justify-between">
                <span class="text-zinc-500">Black:</span>
                <span class="text-zinc-200">{{ puzzle.black }}</span>
              </div>
            </div>
          </div>

          <!-- Navigation & Hint Controls -->
          <div class="flex flex-col gap-2">
            <div class="flex gap-2">
              <button
                class="flex-1 py-2 rounded-xl bg-zinc-800 hover:bg-zinc-700 text-zinc-300 text-xs font-semibold flex items-center justify-center gap-1.5 cursor-pointer"
                @click="handlePrevPuzzle"
              >
                <ChevronLeft class="w-4 h-4" />
                <span>Previous</span>
              </button>
              <button
                class="flex-1 py-2 rounded-xl bg-zinc-800 hover:bg-zinc-700 text-zinc-300 text-xs font-semibold flex items-center justify-center gap-1.5 cursor-pointer"
                @click="handleRandomPuzzle"
              >
                <Shuffle class="w-4 h-4 text-emerald-400" />
                <span>Random</span>
              </button>
              <button
                class="flex-1 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-500 text-white text-xs font-semibold flex items-center justify-center gap-1.5 cursor-pointer shadow-lg shadow-emerald-600/20"
                @click="handleNextPuzzle"
              >
                <span>Next</span>
                <ChevronRight class="w-4 h-4" />
              </button>
            </div>

            <div class="flex gap-2">
              <button
                class="flex-1 py-1.5 rounded-lg bg-zinc-800 hover:bg-zinc-700 text-amber-300 text-xs font-medium flex items-center justify-center gap-1.5 cursor-pointer"
                @click="triggerHint"
              >
                <Lightbulb class="w-3.5 h-3.5" />
                <span>Show Hint / Solution</span>
              </button>
              <button
                class="flex-1 py-1.5 rounded-lg bg-zinc-800 hover:bg-zinc-700 text-zinc-300 text-xs font-medium flex items-center justify-center gap-1.5 cursor-pointer"
                @click="restartCurrentPuzzle"
              >
                <RotateCcw class="w-3.5 h-3.5 text-blue-400" />
                <span>Reset Puzzle</span>
              </button>
            </div>
          </div>

          <!-- Load into Main Workspace -->
          <button
            class="w-full py-2 rounded-xl bg-zinc-950 hover:bg-zinc-800 border border-zinc-800 text-zinc-300 text-xs font-medium flex items-center justify-center gap-2 cursor-pointer transition-colors mt-auto"
            @click="loadIntoMainApp"
          >
            <span>Analyze in Main Workspace</span>
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
