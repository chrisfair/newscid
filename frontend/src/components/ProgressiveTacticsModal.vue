<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import Chessboard from './Chessboard.vue'
import {
  Flame,
  X,
  Heart,
  Timer,
  Award,
  Zap,
  RotateCcw,
  CheckCircle,
  AlertTriangle,
  Play,
  Trophy,
  ArrowRight
} from '@lucide/vue'
import { GetProgressivePuzzle, ParseFEN, MakeMove } from '../../wailsjs/go/main/App'
import { main } from '../../wailsjs/go/models'

const props = defineProps<{
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

// Game configuration
type GameMode = 'survival' | 'rush'
const gameMode = ref<GameMode>('survival')
const isGameActive = ref(false)
const isGameOver = ref(false)

// Progressive state
const currentLevel = ref(1)
const score = ref(0)
const highScore = ref(Number(localStorage.getItem('newscid:progressive_highscore')) || 0)
const lives = ref(3)
const combo = ref(0)
const puzzlesSolved = ref(0)

// Rush mode timer (seconds)
const rushTimeLeft = ref(180)
let timerInterval: number | null = null

// Current puzzle
const puzzle = ref<main.TacticsPuzzle | null>(null)
const currentFEN = ref('')
const legalMoves = ref<string[]>([])
const lastMove = ref<{ from: string; to: string } | null>(null)
const inCheck = ref(false)
const moveIndex = ref(0)

const feedback = ref<{ type: 'success' | 'error' | null; message: string }>({
  type: null,
  message: '',
})

const puzzleTurn = computed(() => {
  return puzzle.value?.turn === 'b' ? 'b' : 'w'
})

const isBoardFlipped = computed(() => {
  return puzzleTurn.value === 'b'
})

onMounted(() => {
  // ready to start
})

onUnmounted(() => {
  stopTimer()
})

function startTimer() {
  stopTimer()
  if (gameMode.value === 'rush') {
    rushTimeLeft.value = 180
    timerInterval = window.setInterval(() => {
      if (rushTimeLeft.value > 0) {
        rushTimeLeft.value--
      } else {
        triggerGameOver('Time is up!')
      }
    }, 1000)
  }
}

function stopTimer() {
  if (timerInterval) {
    clearInterval(timerInterval)
    timerInterval = null
  }
}

function formatRushTime(sec: number): string {
  const m = Math.floor(sec / 60)
  const s = sec % 60
  return `${m}:${s < 10 ? '0' : ''}${s}`
}

async function startChallenge(mode: GameMode) {
  gameMode.value = mode
  currentLevel.value = 1
  score.value = 0
  combo.value = 0
  lives.value = 3
  puzzlesSolved.value = 0
  isGameOver.value = false
  isGameActive.value = true

  startTimer()
  await loadNextPuzzle()
}

async function loadNextPuzzle() {
  try {
    feedback.value = { type: null, message: '' }
    moveIndex.value = 0

    const p = await GetProgressivePuzzle(currentLevel.value)
    puzzle.value = p

    currentFEN.value = p.startFen
    lastMove.value = null

    const state = await ParseFEN(p.startFen)
    legalMoves.value = state.legalMoves
    inCheck.value = state.inCheck
  } catch (err) {
    console.error('Error loading progressive puzzle:', err)
  }
}

async function handleMove(uci: string) {
  if (!puzzle.value || isGameOver.value || !isGameActive.value) return

  const expected = puzzle.value.moves[moveIndex.value]

  if (uci.toLowerCase() === expected?.toLowerCase()) {
    // Correct Move!
    const nextState = await MakeMove(currentFEN.value, uci)
    const from = uci.slice(0, 2)
    const to = uci.slice(2, 4)
    lastMove.value = { from, to }
    currentFEN.value = nextState.fen
    legalMoves.value = nextState.legalMoves
    inCheck.value = nextState.inCheck
    moveIndex.value++

    if (moveIndex.value >= puzzle.value.moves.length) {
      // Puzzle Solved!
      handlePuzzleSolved()
      return
    }

    // Opponent reply in puzzle
    const oppMove = puzzle.value.moves[moveIndex.value]
    if (oppMove) {
      setTimeout(async () => {
        try {
          const oppState = await MakeMove(currentFEN.value, oppMove)
          const oppFrom = oppMove.slice(0, 2)
          const oppTo = oppMove.slice(2, 4)
          lastMove.value = { from: oppFrom, to: oppTo }
          currentFEN.value = oppState.fen
          legalMoves.value = oppState.legalMoves
          inCheck.value = oppState.inCheck
          moveIndex.value++

          if (moveIndex.value >= puzzle.value!.moves.length) {
            handlePuzzleSolved()
          }
        } catch (err) {
          console.error('Opponent move error:', err)
        }
      }, 400)
    }
  } else {
    // Strike / Incorrect Move!
    combo.value = 0
    feedback.value = { type: 'error', message: 'Strike! Move is incorrect.' }

    if (gameMode.value === 'survival') {
      lives.value--
      if (lives.value <= 0) {
        triggerGameOver('3 Strikes! Out of lives.')
      } else {
        // Reset current puzzle to try again
        setTimeout(() => {
          loadNextPuzzle()
        }, 800)
      }
    } else {
      // Rush mode penalty
      rushTimeLeft.value = Math.max(0, rushTimeLeft.value - 10)
      setTimeout(() => {
        loadNextPuzzle()
      }, 800)
    }
  }
}

function handlePuzzleSolved() {
  combo.value++
  puzzlesSolved.value++
  const pts = 10 * currentLevel.value * Math.min(combo.value, 5)
  score.value += pts

  if (score.value > highScore.value) {
    highScore.value = score.value
    localStorage.setItem('newscid:progressive_highscore', String(highScore.value))
  }

  feedback.value = {
    type: 'success',
    message: `Level ${currentLevel.value} Cleared! (+${pts} pts)`,
  }

  // Advance level
  currentLevel.value++

  setTimeout(() => {
    loadNextPuzzle()
  }, 750)
}

function triggerGameOver(reason: string) {
  stopTimer()
  isGameActive.value = false
  isGameOver.value = true
  feedback.value = { type: 'error', message: `Game Over: ${reason}` }
}
</script>

<template>
  <div class="fixed inset-0 bg-black/80 backdrop-blur-xs flex items-center justify-center z-50 p-4 animate-fade-in select-none">
    <div class="bg-zinc-900 border border-zinc-700/80 rounded-2xl w-full max-w-4xl shadow-2xl overflow-hidden flex flex-col max-h-[92vh]">
      <!-- Header -->
      <div class="flex items-center justify-between px-5 py-3.5 bg-zinc-950 border-b border-zinc-800">
        <div class="flex items-center gap-3">
          <Zap class="w-5 h-5 text-amber-400" />
          <div>
            <h2 class="text-sm font-bold text-zinc-100">Progressive Tactics Challenge</h2>
            <p class="text-[11px] text-zinc-500 font-mono">Survival Ladder & 3-Minute Tactics Rush</p>
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
        <!-- Left: Chessboard -->
        <div class="flex flex-col items-center gap-3">
          <!-- Level & Turn Banner -->
          <div class="w-full flex items-center justify-between bg-zinc-950 px-4 py-2 rounded-xl border border-zinc-800 text-xs">
            <div class="flex items-center gap-2">
              <span class="w-2.5 h-2.5 rounded-full" :class="puzzleTurn === 'w' ? 'bg-white' : 'bg-zinc-900 border border-zinc-600'" />
              <span class="font-bold text-zinc-200">
                {{ puzzleTurn === 'w' ? 'White to move' : 'Black to move' }}
              </span>
            </div>
            <span class="text-xs font-bold text-amber-400 font-mono">
              {{ puzzle?.theme || `Level ${currentLevel}` }}
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
            <AlertTriangle v-else class="w-4 h-4 shrink-0 text-rose-400" />
            <span>{{ feedback.message }}</span>
          </div>
        </div>

        <!-- Right: Stats & Game Mode Launcher -->
        <div class="flex-1 flex flex-col gap-4">
          <!-- Active Challenge HUD (when playing) -->
          <div v-if="isGameActive" class="flex flex-col gap-3">
            <!-- Stats Grid -->
            <div class="grid grid-cols-2 gap-2">
              <!-- Level & Score -->
              <div class="bg-zinc-950 p-3 rounded-xl border border-zinc-800 flex flex-col items-center">
                <span class="text-[10px] uppercase font-semibold text-zinc-500">Current Level</span>
                <span class="text-2xl font-black text-amber-400">{{ currentLevel }}</span>
              </div>
              <div class="bg-zinc-950 p-3 rounded-xl border border-zinc-800 flex flex-col items-center">
                <span class="text-[10px] uppercase font-semibold text-zinc-500">Total Score</span>
                <span class="text-2xl font-black text-emerald-400">{{ score }}</span>
              </div>
            </div>

            <!-- Mode-specific HUD (Lives vs Timer) -->
            <div v-if="gameMode === 'survival'" class="bg-zinc-950 p-3 rounded-xl border border-zinc-800 flex items-center justify-between">
              <span class="text-xs font-semibold text-zinc-400">Remaining Lives</span>
              <div class="flex items-center gap-1.5">
                <Heart
                  v-for="l in 3"
                  :key="l"
                  class="w-5 h-5 transition-colors"
                  :class="l <= lives ? 'text-rose-500 fill-current' : 'text-zinc-700'"
                />
              </div>
            </div>

            <div v-else class="bg-zinc-950 p-3 rounded-xl border border-zinc-800 flex items-center justify-between">
              <span class="text-xs font-semibold text-zinc-400">Time Remaining</span>
              <div class="flex items-center gap-2 font-mono text-base font-bold text-amber-400">
                <Timer class="w-4 h-4" />
                <span>{{ formatRushTime(rushTimeLeft) }}</span>
              </div>
            </div>

            <!-- Combo & Streak -->
            <div class="bg-zinc-950 p-3 rounded-xl border border-zinc-800 flex items-center justify-between text-xs">
              <span class="text-zinc-500">Combo Multiplier:</span>
              <span class="font-bold text-amber-400 flex items-center gap-1">
                <Flame class="w-4 h-4" /> {{ Math.min(combo, 5) }}x
              </span>
            </div>

            <!-- Surrender / Stop Challenge -->
            <button
              class="w-full py-2 rounded-xl bg-zinc-800 hover:bg-zinc-700 text-zinc-300 text-xs font-semibold flex items-center justify-center gap-1.5 cursor-pointer mt-auto"
              @click="triggerGameOver('Challenge stopped by player.')"
            >
              <span>End Challenge</span>
            </button>
          </div>

          <!-- Game Over Modal / Summary -->
          <div v-else-if="isGameOver" class="flex-1 flex flex-col items-center justify-center gap-4 bg-zinc-950 p-6 rounded-2xl border border-zinc-800">
            <Trophy class="w-12 h-12 text-amber-400" />
            <h3 class="text-base font-bold text-zinc-100">Challenge Complete!</h3>
            
            <div class="grid grid-cols-2 gap-3 w-full max-w-xs text-xs">
              <div class="p-3 rounded-xl bg-zinc-900 border border-zinc-800 text-center">
                <span class="text-zinc-500">Puzzles Solved</span>
                <p class="text-lg font-bold text-emerald-400">{{ puzzlesSolved }}</p>
              </div>
              <div class="p-3 rounded-xl bg-zinc-900 border border-zinc-800 text-center">
                <span class="text-zinc-500">Highest Level</span>
                <p class="text-lg font-bold text-amber-400">{{ currentLevel }}</p>
              </div>
            </div>

            <div class="p-3 rounded-xl bg-zinc-900 border border-zinc-800 w-full max-w-xs flex justify-between items-center text-xs">
              <span class="text-zinc-400">Final Score:</span>
              <span class="text-base font-bold text-emerald-400">{{ score }}</span>
            </div>

            <div class="flex gap-2 w-full max-w-xs">
              <button
                class="flex-1 py-2.5 rounded-xl bg-emerald-600 hover:bg-emerald-500 text-white font-bold text-xs flex items-center justify-center gap-2 cursor-pointer shadow-lg shadow-emerald-600/30"
                @click="startChallenge(gameMode)"
              >
                <RotateCcw class="w-4 h-4" />
                <span>Play Again</span>
              </button>
            </div>
          </div>

          <!-- Challenge Setup / Selection (Start Screen) -->
          <div v-else class="flex-1 flex flex-col gap-4">
            <!-- High Score Badge -->
            <div class="bg-amber-500/10 border border-amber-500/30 p-3 rounded-xl flex items-center justify-between text-xs">
              <div class="flex items-center gap-2 text-amber-300 font-semibold">
                <Trophy class="w-4 h-4 text-amber-400" />
                <span>Personal Best High Score</span>
              </div>
              <span class="font-bold text-amber-400 font-mono">{{ highScore }} pts</span>
            </div>

            <!-- Mode 1: Survival Mode -->
            <div
              class="bg-zinc-950 p-4 rounded-xl border border-zinc-800 hover:border-emerald-500/60 transition-all cursor-pointer flex flex-col gap-2 group"
              @click="startChallenge('survival')"
            >
              <div class="flex items-center justify-between">
                <div class="flex items-center gap-2">
                  <Heart class="w-4 h-4 text-rose-500 fill-current" />
                  <span class="text-xs font-bold text-zinc-100 group-hover:text-emerald-400 transition-colors">Tactics Survival Ladder</span>
                </div>
                <ArrowRight class="w-4 h-4 text-zinc-500 group-hover:text-emerald-400 transition-colors" />
              </div>
              <p class="text-[11px] text-zinc-400">
                Start at Level 1 with simple checkmates, progressively scaling up to Grandmaster combinations. 3 strikes and you're out!
              </p>
            </div>

            <!-- Mode 2: Tactics Rush -->
            <div
              class="bg-zinc-950 p-4 rounded-xl border border-zinc-800 hover:border-amber-500/60 transition-all cursor-pointer flex flex-col gap-2 group"
              @click="startChallenge('rush')"
            >
              <div class="flex items-center justify-between">
                <div class="flex items-center gap-2">
                  <Timer class="w-4 h-4 text-amber-400" />
                  <span class="text-xs font-bold text-zinc-100 group-hover:text-amber-400 transition-colors">3-Minute Tactics Rush</span>
                </div>
                <ArrowRight class="w-4 h-4 text-zinc-500 group-hover:text-amber-400 transition-colors" />
              </div>
              <p class="text-[11px] text-zinc-400">
                Fast-paced blitz puzzle sprint. Solve as many puzzles as possible in 3 minutes. Wrong moves deduct 10 seconds!
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
