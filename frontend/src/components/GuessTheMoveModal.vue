<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import {
  Award,
  X,
  Lightbulb,
  ArrowRight,
  RotateCcw,
  CheckCircle2,
  AlertCircle,
  HelpCircle,
  Trophy,
  Play
} from '@lucide/vue'
import Chessboard from './Chessboard.vue'
import {
  StartGuessTheMove,
  GetGuessTheMoveState,
  SubmitGuessTheMove,
  GetGuessTheMoveHint,
  SkipGuessTheMove
} from '../../wailsjs/go/main/App'
import { training } from '../../wailsjs/go/models'

const props = withDefaults(defineProps<{
  activeGameId?: number
  gameHeader?: {
    white?: string
    black?: string
    event?: string
    date?: string
    result?: string
  }
  currentPieceSet?: string
  boardTheme?: {
    light: string
    dark: string
    highlight: string
  }
}>(), {
  activeGameId: 0,
  gameHeader: () => ({
    white: 'White',
    black: 'Black',
    event: 'Game',
    date: 'Unknown',
    result: '*'
  }),
  currentPieceSet: 'cburnett',
  boardTheme: () => ({
    light: '#e3c896',
    dark: '#ab7a4e',
    highlight: '#f7e597',
  })
})

const emit = defineEmits<{
  (e: 'close'): void
}>()

// Session State
const selectedSide = ref<'white' | 'black'>('white')
const session = ref<training.SolitaireSession | null>(null)
const currentFEN = ref('rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1')
const moveInput = ref('')
const isLoading = ref(false)
const errorMessage = ref('')
const lastFeedback = ref<{
  type: 'success' | 'error' | 'info' | null
  message: string
  points?: number
}>({
  type: null,
  message: ''
})
const currentHint = ref<{ text: string; level: number } | null>(null)
const isFinished = ref(false)
const summary = ref<training.PerformanceSummary | null>(null)

const isWhiteUser = computed(() => {
  if (!session.value) return selectedSide.value === 'white'
  return session.value.userSide === 1 || selectedSide.value === 'white'
})

const currentTurnText = computed(() => {
  if (!session.value || isFinished.value) return 'Game Finished'
  const isWhite = isWhiteUser.value
  return isWhite ? 'Your turn to move (White)' : 'Your turn to move (Black)'
})

async function initSession() {
  isLoading.value = true
  errorMessage.value = ''
  lastFeedback.value = { type: null, message: '' }
  currentHint.value = null
  isFinished.value = false
  summary.value = null

  try {
    const res = await StartGuessTheMove(props.activeGameId, selectedSide.value)
    session.value = res
    if (res && res.moves && res.moves.length > 0) {
      if (res.currentPlyIdx < res.moves.length) {
        currentFEN.value = res.moves[res.currentPlyIdx].fen
      } else {
        currentFEN.value = res.moves[res.moves.length - 1].fen
      }
    }
  } catch (err: any) {
    errorMessage.value = `Failed to start Solitaire Chess: ${err}`
  } finally {
    isLoading.value = false
  }
}

async function handleBoardMove(uci: string) {
  if (isFinished.value || !session.value) return
  await submitMoveGuess(uci)
}

async function handleSubmitInput() {
  if (!moveInput.value.trim() || isFinished.value) return
  await submitMoveGuess(moveInput.value.trim())
}

async function submitMoveGuess(guess: string) {
  isLoading.value = true
  errorMessage.value = ''
  currentHint.value = null

  try {
    const fb = await SubmitGuessTheMove(guess)
    if (fb.isMatch) {
      lastFeedback.value = {
        type: 'success',
        message: `Correct! ${fb.actualMoveSAN} was played.` +
          (fb.opponentReplySAN ? ` Opponent replied ${fb.opponentReplySAN}.` : ''),
        points: fb.pointsEarned
      }
      moveInput.value = ''
      currentFEN.value = fb.currentFEN
      if (session.value) {
        session.value.score = fb.totalScore
        session.value.correctGuesses++
      }
      if (fb.isFinished) {
        isFinished.value = true
        summary.value = fb.summary || null
      }
    } else {
      lastFeedback.value = {
        type: 'error',
        message: `Not the played move! Try another move or request a hint.`
      }
    }
  } catch (err: any) {
    errorMessage.value = `Submission error: ${err}`
  } finally {
    isLoading.value = false
  }
}

async function requestHint() {
  if (isFinished.value) return
  try {
    const res = await GetGuessTheMoveHint()
    // res can be a string or array depending on binding
    let text = ''
    let level = 1
    if (typeof res === 'string') {
      text = res
    } else if (res && typeof res === 'object') {
      text = res[0] || (res as any).text || String(res)
      level = res[1] || (res as any).level || 1
    }
    currentHint.value = { text, level }
  } catch (err: any) {
    errorMessage.value = `Hint error: ${err}`
  }
}

async function handleSkip() {
  if (isFinished.value) return
  isLoading.value = true
  errorMessage.value = ''
  currentHint.value = null

  try {
    const fb = await SkipGuessTheMove()
    lastFeedback.value = {
      type: 'info',
      message: `Move revealed: ${fb.actualMoveSAN}.` +
        (fb.opponentReplySAN ? ` Opponent replied ${fb.opponentReplySAN}.` : ''),
      points: 0
    }
    moveInput.value = ''
    currentFEN.value = fb.currentFEN
    if (fb.isFinished) {
      isFinished.value = true
      summary.value = fb.summary || null
    }
  } catch (err: any) {
    errorMessage.value = `Skip error: ${err}`
  } finally {
    isLoading.value = false
  }
}

onMounted(async () => {
  await initSession()
})
</script>

<template>
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/75 backdrop-blur-sm p-4 animate-fade-in">
    <div class="bg-zinc-900 border border-zinc-800 rounded-xl shadow-2xl w-full max-w-4xl flex flex-col max-h-[92vh] overflow-hidden">
      <!-- Modal Header -->
      <div class="px-5 py-3 border-b border-zinc-800 flex items-center justify-between bg-zinc-950/60">
        <div class="flex items-center gap-2.5">
          <div class="p-2 rounded-lg bg-amber-500/10 border border-amber-500/20 text-amber-400">
            <Award class="w-5 h-5" />
          </div>
          <div>
            <h2 class="text-base font-semibold text-zinc-100 flex items-center gap-2">
              Guess the Move
              <span class="text-xs px-2 py-0.5 rounded-full bg-amber-500/20 text-amber-300 font-mono">
                Solitaire Chess
              </span>
            </h2>
            <p class="text-xs text-zinc-400">
              {{ gameHeader.white }} vs {{ gameHeader.black }} • {{ gameHeader.event }} ({{ gameHeader.date }})
            </p>
          </div>
        </div>
        <button
          class="p-1.5 rounded-lg text-zinc-400 hover:text-zinc-200 hover:bg-zinc-800 transition-colors cursor-pointer"
          @click="emit('close')"
        >
          <X class="w-5 h-5" />
        </button>
      </div>

      <!-- Main Body -->
      <div class="flex-1 overflow-y-auto p-4 flex flex-col md:flex-row gap-5 items-center md:items-start justify-center">
        <!-- Left: Interactive Chessboard -->
        <div class="flex flex-col items-center gap-2 shrink-0">
          <div class="rounded-lg overflow-hidden border border-zinc-700/60 shadow-lg w-80 h-80 sm:w-96 sm:h-96">
            <Chessboard
              :fen="currentFEN"
              :piece-set="currentPieceSet"
              :board-theme="boardTheme"
              :flipped="!isWhiteUser"
              @move="handleBoardMove"
            />
          </div>
          <p class="text-[11px] text-zinc-500">
            Tip: You can play your guess directly on the board or type below.
          </p>
        </div>

        <!-- Right: Control Panel & Status -->
        <div class="flex-1 flex flex-col gap-3.5 w-full min-w-0">
          <!-- Setup Bar: Side Selection -->
          <div class="bg-zinc-950/40 border border-zinc-800/80 rounded-lg p-3 flex items-center justify-between">
            <div class="text-xs text-zinc-300 font-medium">Train Playing As:</div>
            <div class="flex items-center gap-2">
              <button
                class="px-3 py-1 rounded text-xs font-medium transition-colors cursor-pointer"
                :class="selectedSide === 'white' ? 'bg-amber-600 text-white' : 'bg-zinc-800 text-zinc-400 hover:bg-zinc-700'"
                @click="selectedSide = 'white'; initSession()"
              >
                White
              </button>
              <button
                class="px-3 py-1 rounded text-xs font-medium transition-colors cursor-pointer"
                :class="selectedSide === 'black' ? 'bg-amber-600 text-white' : 'bg-zinc-800 text-zinc-400 hover:bg-zinc-700'"
                @click="selectedSide = 'black'; initSession()"
              >
                Black
              </button>
              <button
                class="p-1 rounded bg-zinc-800 hover:bg-zinc-700 text-zinc-400 hover:text-zinc-200 transition-colors ml-1 cursor-pointer"
                title="Restart Session"
                @click="initSession"
              >
                <RotateCcw class="w-4 h-4" />
              </button>
            </div>
          </div>

          <!-- Score Card -->
          <div class="grid grid-cols-3 gap-2 text-center">
            <div class="bg-zinc-800/40 border border-zinc-700/40 rounded-lg p-2.5">
              <span class="text-[10px] uppercase text-zinc-500 font-semibold block">Score</span>
              <span class="text-lg font-bold text-amber-400 font-mono">
                {{ session?.score || 0 }}
              </span>
            </div>
            <div class="bg-zinc-800/40 border border-zinc-700/40 rounded-lg p-2.5">
              <span class="text-[10px] uppercase text-zinc-500 font-semibold block">Guesses</span>
              <span class="text-lg font-bold text-emerald-400 font-mono">
                {{ session?.correctGuesses || 0 }} / {{ session?.totalTurns || 0 }}
              </span>
            </div>
            <div class="bg-zinc-800/40 border border-zinc-700/40 rounded-lg p-2.5">
              <span class="text-[10px] uppercase text-zinc-500 font-semibold block">Hints Used</span>
              <span class="text-lg font-bold text-sky-400 font-mono">
                {{ session?.hintsUsed || 0 }}
              </span>
            </div>
          </div>

          <!-- Turn Banner -->
          <div
            v-if="!isFinished"
            class="px-3 py-2 rounded-lg bg-zinc-800/80 border border-zinc-700 flex items-center justify-between text-xs"
          >
            <span class="font-semibold text-zinc-200">{{ currentTurnText }}</span>
            <span class="text-zinc-400 text-[11px]">
              Turn {{ (session?.currentPlyIdx || 0) + 1 }} of {{ session?.moves?.length || 0 }}
            </span>
          </div>

          <!-- Feedback Alert -->
          <div
            v-if="lastFeedback.type"
            class="p-3 rounded-lg text-xs flex items-start gap-2 border animate-fade-in"
            :class="{
              'bg-emerald-950/40 border-emerald-700/50 text-emerald-300': lastFeedback.type === 'success',
              'bg-rose-950/40 border-rose-700/50 text-rose-300': lastFeedback.type === 'error',
              'bg-sky-950/40 border-sky-700/50 text-sky-300': lastFeedback.type === 'info',
            }"
          >
            <CheckCircle2 v-if="lastFeedback.type === 'success'" class="w-4 h-4 shrink-0 mt-0.5 text-emerald-400" />
            <AlertCircle v-else class="w-4 h-4 shrink-0 mt-0.5 text-rose-400" />
            <div class="flex-1">
              <span>{{ lastFeedback.message }}</span>
              <span v-if="lastFeedback.points && lastFeedback.points > 0" class="ml-1 font-bold text-emerald-400">
                (+{{ lastFeedback.points }} pts)
              </span>
            </div>
          </div>

          <!-- Hint Callout -->
          <div
            v-if="currentHint"
            class="p-3 rounded-lg bg-amber-950/30 border border-amber-700/40 text-xs text-amber-200 flex items-start gap-2 animate-fade-in"
          >
            <Lightbulb class="w-4 h-4 shrink-0 mt-0.5 text-amber-400" />
            <div>
              <span class="font-semibold">Hint (Level {{ currentHint.level }}):</span>
              <p class="mt-0.5 text-amber-300/90">{{ currentHint.text }}</p>
            </div>
          </div>

          <!-- Error Alert -->
          <div v-if="errorMessage" class="p-2 rounded bg-rose-950/60 border border-rose-700 text-rose-200 text-xs">
            {{ errorMessage }}
          </div>

          <!-- Input Controls (Active Session) -->
          <div v-if="!isFinished" class="flex flex-col gap-2 mt-auto">
            <form @submit.prevent="handleSubmitInput" class="flex gap-2">
              <input
                v-model="moveInput"
                type="text"
                placeholder="Enter move (e.g. e4, Nf3, O-O, e2e4)..."
                class="flex-1 bg-zinc-950 border border-zinc-700 rounded-lg px-3 py-2 text-sm text-zinc-100 placeholder-zinc-500 focus:outline-none focus:border-amber-500"
                :disabled="isLoading"
              />
              <button
                type="submit"
                class="px-4 py-2 rounded-lg bg-amber-600 hover:bg-amber-500 text-white font-medium text-xs flex items-center gap-1.5 transition-colors cursor-pointer disabled:opacity-50"
                :disabled="isLoading || !moveInput.trim()"
              >
                <span>Guess</span>
                <ArrowRight class="w-3.5 h-3.5" />
              </button>
            </form>

            <div class="flex items-center gap-2">
              <button
                class="flex-1 px-3 py-2 rounded-lg bg-zinc-800 hover:bg-zinc-700 border border-zinc-700 text-zinc-300 hover:text-zinc-100 text-xs font-medium flex items-center justify-center gap-1.5 transition-colors cursor-pointer"
                @click="requestHint"
              >
                <Lightbulb class="w-3.5 h-3.5 text-amber-400" />
                <span>Get Hint</span>
              </button>
              <button
                class="px-3 py-2 rounded-lg bg-zinc-800 hover:bg-zinc-700 border border-zinc-700 text-zinc-400 hover:text-zinc-200 text-xs transition-colors cursor-pointer"
                title="Skip this turn and reveal actual move"
                @click="handleSkip"
              >
                Reveal & Skip
              </button>
            </div>
          </div>

          <!-- Game Completion Summary -->
          <div
            v-else
            class="bg-gradient-to-br from-amber-950/40 via-zinc-900 to-zinc-950 border border-amber-500/40 rounded-xl p-5 flex flex-col items-center text-center gap-3 animate-fade-in"
          >
            <div class="p-3 rounded-full bg-amber-500/20 border border-amber-500/40 text-amber-400">
              <Trophy class="w-8 h-8" />
            </div>
            <div>
              <h3 class="text-base font-bold text-zinc-100">
                {{ summary?.ratingTitle || 'Training Finished' }}
              </h3>
              <p class="text-xs text-zinc-400 mt-1 max-w-sm">
                {{ summary?.feedback || 'Great job playing through the master game!' }}
              </p>
            </div>

            <div class="w-full bg-zinc-800/70 border border-zinc-700/60 rounded-lg p-3 grid grid-cols-2 gap-2 text-xs">
              <div>
                <span class="text-zinc-500 block">Final Score</span>
                <span class="font-bold text-amber-400 text-base font-mono">
                  {{ summary?.score || session?.score || 0 }} / {{ summary?.maxScore || (session?.totalTurns || 0) * 10 }}
                </span>
              </div>
              <div>
                <span class="text-zinc-500 block">Move Accuracy</span>
                <span class="font-bold text-emerald-400 text-base font-mono">
                  {{ Math.round(summary?.accuracyPct || 0) }}%
                </span>
              </div>
            </div>

            <div class="flex items-center gap-3 w-full mt-2">
              <button
                class="flex-1 py-2 px-3 rounded-lg bg-amber-600 hover:bg-amber-500 text-white font-medium text-xs transition-colors cursor-pointer"
                @click="initSession"
              >
                Play Again
              </button>
              <button
                class="flex-1 py-2 px-3 rounded-lg bg-zinc-800 hover:bg-zinc-700 border border-zinc-700 text-zinc-300 text-xs transition-colors cursor-pointer"
                @click="emit('close')"
              >
                Close
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
