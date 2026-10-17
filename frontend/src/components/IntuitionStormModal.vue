<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import {
  X,
  Zap,
  Flame,
  Trophy,
  RotateCcw,
  ChevronRight,
  Eye,
  CheckCircle2,
  AlertTriangle,
  Volume2,
  VolumeX,
  Sparkles,
  Database,
  Timer,
  TrendingUp,
  Activity
} from '@lucide/vue'
import Chessboard from './Chessboard.vue'
import {
  GetIntuitionDrillBatch,
  GenerateIntuitionDrillsFromDatabase,
  SubmitIntuitionAnswer
} from '../../wailsjs/go/main/App'
import { training } from '../../wailsjs/go/models'
import { playMoveSound, playCaptureSound, playCheckSound, soundEnabled, toggleSound } from '../utils/sound'

const props = withDefaults(defineProps<{
  initialFEN?: string
  pieceSet?: string
  boardTheme?: {
    light: string
    dark: string
    highlight: string
  }
}>(), {
  initialFEN: 'rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1',
  pieceSet: 'cburnett',
  boardTheme: () => ({
    light: '#e3c896',
    dark: '#ab7a4e',
    highlight: '#f7e597'
  })
})

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'loadPosition', fen: string): void
}>()

const HIGH_SCORE_KEY = 'newscid:intuition_high_score'
const RATING_KEY = 'newscid:intuition_rating'

// Mode
const sourceMode = ref<'curated' | 'database'>('curated')

// Drill State
const challenges = ref<training.IntuitionChallenge[]>([])
const currentIndex = ref<number>(0)
const currentChallenge = computed(() => challenges.value[currentIndex.value] || null)
const currentFEN = computed(() => currentChallenge.value?.fen || props.initialFEN)

// Timing & Selection
const elapsedMs = ref<number>(0)
const remainingMs = ref<number>(5000)
const maxTimeMs = ref<number>(5000)
let timerInterval: number | null = null
const isAnswering = ref<boolean>(false)
const autoAdvance = ref<boolean>(true)
const autoAdvanceTimer = ref<number | null>(null)

// Result state
const result = ref<training.IntuitionResult | null>(null)
const isLoading = ref<boolean>(false)

// Gamification stats
const score = ref<number>(0)
const streak = ref<number>(0)
const bestStreak = ref<number>(0)
const totalAnswered = ref<number>(0)
const totalCorrect = ref<number>(0)
const reflexTimes = ref<number[]>([])
const highScore = ref<number>(parseInt(localStorage.getItem(HIGH_SCORE_KEY) || '0', 10))
const instinctRating = ref<number>(parseInt(localStorage.getItem(RATING_KEY) || '1500', 10))

const accuracyPct = computed(() => {
  if (totalAnswered.value === 0) return 0
  return Math.round((totalCorrect.value / totalAnswered.value) * 100)
})

const averageReflexMs = computed(() => {
  if (reflexTimes.value.length === 0) return 0
  const sum = reflexTimes.value.reduce((a, b) => a + b, 0)
  return Math.round(sum / reflexTimes.value.length)
})

const timerProgressPct = computed(() => {
  return Math.max(0, Math.min(100, (remainingMs.value / maxTimeMs.value) * 100))
})

const timerColorClass = computed(() => {
  if (timerProgressPct.value > 50) return 'bg-emerald-500'
  if (timerProgressPct.value > 20) return 'bg-amber-500'
  return 'bg-red-500 animate-pulse'
})

// Start timer for current question
function startTimer() {
  stopTimer()
  remainingMs.value = maxTimeMs.value
  elapsedMs.value = 0
  const startTime = performance.now()

  timerInterval = window.setInterval(() => {
    const elapsed = Math.round(performance.now() - startTime)
    elapsedMs.value = elapsed
    remainingMs.value = Math.max(0, maxTimeMs.value - elapsed)

    if (remainingMs.value <= 0) {
      stopTimer()
      // Time out! Counts as wrong guess
      if (!result.value && isAnswering.value) {
        handleTimeout()
      }
    }
  }, 30)
}

function stopTimer() {
  if (timerInterval !== null) {
    clearInterval(timerInterval)
    timerInterval = null
  }
}

async function loadChallenges() {
  isLoading.value = true
  stopTimer()
  clearAutoAdvance()
  result.value = null

  try {
    let list: training.IntuitionChallenge[] = []
    if (sourceMode.value === 'database') {
      list = await GenerateIntuitionDrillsFromDatabase(20)
    } else {
      list = await GetIntuitionDrillBatch(20)
    }
    challenges.value = list
    currentIndex.value = 0
    if (challenges.value.length > 0) {
      isAnswering.value = true
      startTimer()
    }
  } catch (err) {
    console.error('Failed to load intuition challenges:', err)
  } finally {
    isLoading.value = false
  }
}

async function handleAnswer(choice: 'white_winning' | 'equal' | 'black_winning') {
  if (!currentChallenge.value || result.value || !isAnswering.value) return

  stopTimer()
  isAnswering.value = false
  const takenMs = elapsedMs.value
  reflexTimes.value.push(takenMs)
  totalAnswered.value++

  try {
    const res = await SubmitIntuitionAnswer(currentChallenge.value.id, choice, takenMs)
    result.value = res

    if (res.isCorrect) {
      streak.value++
      if (streak.value > bestStreak.value) {
        bestStreak.value = streak.value
      }
      totalCorrect.value++
      score.value += res.pointsAwarded

      if (score.value > highScore.value) {
        highScore.value = score.value
        localStorage.setItem(HIGH_SCORE_KEY, score.value.toString())
      }

      // Rating adjustment
      const gain = Math.max(8, Math.round(24 * (1 - takenMs / 5000)))
      instinctRating.value += gain
      localStorage.setItem(RATING_KEY, instinctRating.value.toString())

      playCheckSound()
    } else {
      streak.value = 0
      instinctRating.value = Math.max(800, instinctRating.value - 16)
      localStorage.setItem(RATING_KEY, instinctRating.value.toString())
      playCaptureSound()
    }

    if (autoAdvance.value) {
      autoAdvanceTimer.value = window.setTimeout(() => {
        nextChallenge()
      }, res.isCorrect ? 1400 : 2500)
    }
  } catch (err) {
    console.error('Error submitting intuition answer:', err)
  }
}

function handleTimeout() {
  if (!currentChallenge.value || result.value) return
  isAnswering.value = false
  streak.value = 0
  totalAnswered.value++
  reflexTimes.value.push(maxTimeMs.value)
  instinctRating.value = Math.max(800, instinctRating.value - 20)
  localStorage.setItem(RATING_KEY, instinctRating.value.toString())

  result.value = new training.IntuitionResult({
    isCorrect: false,
    userChoice: 'equal',
    actualCategory: currentChallenge.value.category,
    evaluationCp: currentChallenge.value.evaluationCp,
    scoreFormatted: (currentChallenge.value.evaluationCp > 0 ? '+' : '') + (currentChallenge.value.evaluationCp / 100).toFixed(2),
    pointsAwarded: 0,
    speedBonus: 0,
    explanation: `⏱️ Time Expired! In Intuition Storm speed is everything. Actual: ${formatCategory(currentChallenge.value.category)}. ${currentChallenge.value.keyFeature}`,
    keyFeature: currentChallenge.value.keyFeature
  })

  playCaptureSound()

  if (autoAdvance.value) {
    autoAdvanceTimer.value = window.setTimeout(() => {
      nextChallenge()
    }, 2500)
  }
}

function formatCategory(cat: string): string {
  if (cat === 'white_winning') return 'White is Winning (+)'
  if (cat === 'black_winning') return 'Black is Winning (-)'
  return 'Equal (=)'
}

function clearAutoAdvance() {
  if (autoAdvanceTimer.value !== null) {
    clearTimeout(autoAdvanceTimer.value)
    autoAdvanceTimer.value = null
  }
}

function nextChallenge() {
  clearAutoAdvance()
  result.value = null

  if (currentIndex.value + 1 < challenges.value.length) {
    currentIndex.value++
    isAnswering.value = true
    startTimer()
    playMoveSound()
  } else {
    // Reload a fresh batch
    loadChallenges()
  }
}

function restartSession() {
  clearAutoAdvance()
  stopTimer()
  score.value = 0
  streak.value = 0
  totalAnswered.value = 0
  totalCorrect.value = 0
  reflexTimes.value = []
  result.value = null
  loadChallenges()
}

function handleKeyDown(e: KeyboardEvent) {
  if (e.target instanceof HTMLInputElement || e.target instanceof HTMLTextAreaElement) return

  if (e.key === 'Escape') {
    emit('close')
    return
  }

  // Intuition rapid keys:
  // 1 or ArrowLeft -> White Winning
  // 2 or ArrowDown or Space -> Equal
  // 3 or ArrowRight -> Black Winning
  if (isAnswering.value && !result.value) {
    if (e.key === '1' || e.key === 'ArrowLeft') {
      e.preventDefault()
      handleAnswer('white_winning')
      return
    }
    if (e.key === '2' || e.key === 'ArrowDown' || e.key === ' ') {
      e.preventDefault()
      handleAnswer('equal')
      return
    }
    if (e.key === '3' || e.key === 'ArrowRight') {
      e.preventDefault()
      handleAnswer('black_winning')
      return
    }
  }

  // If result is displayed and user hits Enter or Space, immediately proceed to next
  if (result.value && (e.key === 'Enter' || e.key === ' ')) {
    e.preventDefault()
    nextChallenge()
  }
}

function exploreInMainBoard() {
  if (currentChallenge.value) {
    emit('loadPosition', currentChallenge.value.fen)
    emit('close')
  }
}

onMounted(() => {
  window.addEventListener('keydown', handleKeyDown)
  loadChallenges()
})

onUnmounted(() => {
  window.removeEventListener('keydown', handleKeyDown)
  stopTimer()
  clearAutoAdvance()
})
</script>

<template>
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/75 backdrop-blur-sm p-4 select-none">
    <div
      class="bg-theme-panel border border-theme-border rounded-xl shadow-2xl flex flex-col w-full max-w-4xl overflow-hidden animate-in fade-in zoom-in-95 duration-150"
      style="max-height: 94vh;"
    >
      <!-- Header -->
      <div class="px-6 py-3.5 border-b border-theme-border flex items-center justify-between bg-theme-panel-header">
        <div class="flex items-center gap-3">
          <div class="p-2 bg-amber-500/10 text-amber-400 rounded-lg border border-amber-500/20">
            <Zap class="w-5 h-5 fill-amber-500/30" />
          </div>
          <div>
            <div class="flex items-center gap-2">
              <h2 class="text-base font-bold text-theme-text-primary tracking-wide">Intuition Storm</h2>
              <span class="px-2 py-0.5 text-xs font-semibold rounded bg-amber-500/20 text-amber-300 border border-amber-500/30">
                Speed Reflex Trainer
              </span>
            </div>
            <p class="text-xs text-theme-text-muted mt-0.5">
              Decide White Winning (+), Equal (=), or Black Winning (-) in 5 seconds or less.
            </p>
          </div>
        </div>

        <div class="flex items-center gap-3">
          <!-- Sound toggle -->
          <button
            class="p-2 text-theme-text-muted hover:text-theme-text-primary rounded-lg transition-colors border border-theme-border/50 hover:bg-theme-hover"
            :title="soundEnabled ? 'Mute Sounds' : 'Unmute Sounds'"
            @click="toggleSound()"
          >
            <Volume2 v-if="soundEnabled" class="w-4 h-4 text-emerald-400" />
            <VolumeX v-else class="w-4 h-4 text-rose-400" />
          </button>

          <!-- Source Toggle -->
          <div class="flex items-center bg-theme-panel border border-theme-border rounded-lg p-0.5 text-xs">
            <button
              class="px-2.5 py-1 rounded transition-colors flex items-center gap-1.5"
              :class="sourceMode === 'curated' ? 'bg-amber-500 text-black font-semibold shadow' : 'text-theme-text-muted hover:text-theme-text-primary'"
              @click="sourceMode = 'curated'; restartSession()"
            >
              <Sparkles class="w-3.5 h-3.5" />
              Grandmaster
            </button>
            <button
              class="px-2.5 py-1 rounded transition-colors flex items-center gap-1.5"
              :class="sourceMode === 'database' ? 'bg-amber-500 text-black font-semibold shadow' : 'text-theme-text-muted hover:text-theme-text-primary'"
              @click="sourceMode = 'database'; restartSession()"
            >
              <Database class="w-3.5 h-3.5" />
              Active DB
            </button>
          </div>

          <button
            class="text-theme-text-muted hover:text-theme-text-primary p-1.5 rounded-lg hover:bg-theme-hover transition-colors"
            @click="emit('close')"
          >
            <X class="w-5 h-5" />
          </button>
        </div>
      </div>

      <!-- Stats ribbon -->
      <div class="px-6 py-2 bg-theme-panel-header/50 border-b border-theme-border/60 flex items-center justify-between text-xs">
        <div class="flex items-center gap-6">
          <div class="flex items-center gap-1.5 font-medium">
            <Flame class="w-4 h-4 text-orange-500" :class="{ 'animate-bounce': streak >= 3 }" />
            <span class="text-theme-text-muted">Streak:</span>
            <span class="font-bold text-orange-400 text-sm">{{ streak }}</span>
            <span v-if="bestStreak > 0" class="text-theme-text-muted text-[11px]">(Best: {{ bestStreak }})</span>
          </div>

          <div class="flex items-center gap-1.5 font-medium">
            <Trophy class="w-4 h-4 text-amber-400" />
            <span class="text-theme-text-muted">Score:</span>
            <span class="font-bold text-amber-300 text-sm">{{ score }}</span>
            <span v-if="highScore > 0" class="text-theme-text-muted text-[11px]">(Record: {{ highScore }})</span>
          </div>

          <div class="flex items-center gap-1.5 font-medium">
            <TrendingUp class="w-4 h-4 text-indigo-400" />
            <span class="text-theme-text-muted">Instinct Rating:</span>
            <span class="font-bold text-indigo-300 text-sm">{{ instinctRating }}</span>
          </div>
        </div>

        <div class="flex items-center gap-6 text-theme-text-muted">
          <div class="flex items-center gap-1">
            <Activity class="w-3.5 h-3.5 text-blue-400" />
            <span>Reflex:</span>
            <span class="font-semibold text-theme-text-primary">{{ averageReflexMs }}ms</span>
          </div>

          <div>
            <span>Accuracy:</span>
            <span class="font-semibold text-theme-text-primary ml-1">{{ accuracyPct }}%</span>
            <span class="text-[10px] ml-0.5">({{ totalCorrect }}/{{ totalAnswered }})</span>
          </div>

          <!-- Auto-advance toggle -->
          <label class="flex items-center gap-1.5 cursor-pointer hover:text-theme-text-primary select-none">
            <input
              v-model="autoAdvance"
              type="checkbox"
              class="rounded border-theme-border bg-theme-panel text-amber-500 focus:ring-amber-400 h-3.5 w-3.5"
            />
            <span class="text-[11px]">Auto-Next</span>
          </label>
        </div>
      </div>

      <!-- Countdown Progress Bar -->
      <div class="w-full bg-theme-panel h-1.5 overflow-hidden">
        <div
          class="h-full transition-all duration-75 ease-linear"
          :class="timerColorClass"
          :style="{ width: `${timerProgressPct}%` }"
        ></div>
      </div>

      <!-- Main Body -->
      <div class="p-6 flex-1 flex flex-col md:flex-row gap-6 items-center justify-center overflow-y-auto">
        <!-- Chessboard container -->
        <div class="flex flex-col items-center">
          <div class="relative rounded-lg overflow-hidden shadow-xl border border-theme-border/80" style="width: 380px; height: 380px;">
            <Chessboard
              :fen="currentFEN"
              :piece-set="pieceSet"
              :board-theme="boardTheme"
              :flipped="currentChallenge?.sideToMove === 'black' || currentFEN.split(' ')[1] === 'b'"
              :interactive="false"
              class="w-full h-full"
            />

            <!-- Loading overlay -->
            <div
              v-if="isLoading"
              class="absolute inset-0 bg-theme-panel/80 backdrop-blur-sm flex flex-col items-center justify-center gap-3 z-20"
            >
              <div class="w-8 h-8 border-2 border-amber-400 border-t-transparent rounded-full animate-spin"></div>
              <span class="text-xs font-semibold text-theme-text-muted tracking-wide">Gathering Positions...</span>
            </div>
          </div>

          <!-- Position metadata caption -->
          <div v-if="currentChallenge" class="mt-2 text-center text-xs text-theme-text-muted truncate max-w-sm">
            <span v-if="currentChallenge.whitePlayer && currentChallenge.blackPlayer" class="font-medium text-theme-text-primary">
              {{ currentChallenge.whitePlayer }} vs {{ currentChallenge.blackPlayer }}
            </span>
            <span v-if="currentChallenge.event" class="block text-[11px] text-theme-text-muted/75 mt-0.5">
              {{ currentChallenge.event }}
            </span>
          </div>
        </div>

        <!-- Decision & Reflex Dashboard -->
        <div class="flex-1 flex flex-col justify-between w-full max-w-md h-full py-2">
          <!-- Timer Display Badge -->
          <div class="flex items-center justify-between mb-4">
            <div class="flex items-center gap-2">
              <Timer class="w-4 h-4" :class="remainingMs <= 1500 ? 'text-rose-400 animate-spin' : 'text-amber-400'" />
              <span class="text-sm font-bold tracking-wider font-mono" :class="remainingMs <= 1500 ? 'text-rose-400 font-extrabold' : 'text-theme-text-primary'">
                {{ (remainingMs / 1000).toFixed(1) }}s
              </span>
            </div>
            <span class="text-xs text-theme-text-muted">
              Problem {{ currentIndex + 1 }} of {{ challenges.length }}
            </span>
          </div>

          <!-- Action Evaluation Cards (White Winning / Equal / Black Winning) -->
          <div class="space-y-3 my-auto">
            <!-- White Winning Button -->
            <button
              class="w-full group px-4 py-3.5 rounded-xl border text-left transition-all duration-150 flex items-center justify-between relative overflow-hidden"
              :class="[
                result
                  ? result.actualCategory === 'white_winning'
                    ? 'bg-emerald-500/15 border-emerald-500 text-emerald-300 ring-2 ring-emerald-500/50'
                    : result.userChoice === 'white_winning'
                    ? 'bg-rose-500/15 border-rose-500 text-rose-300'
                    : 'opacity-40 border-theme-border bg-theme-panel/40'
                  : 'bg-theme-panel hover:bg-theme-hover border-theme-border hover:border-amber-500/60 text-theme-text-primary hover:shadow-lg hover:-translate-y-0.5 active:translate-y-0'
              ]"
              :disabled="!isAnswering || !!result"
              @click="handleAnswer('white_winning')"
            >
              <div class="flex items-center gap-3">
                <div class="w-8 h-8 rounded-lg bg-white/10 flex items-center justify-center font-bold text-sm text-white border border-white/20">
                  +
                </div>
                <div>
                  <div class="font-bold text-sm">White is Winning</div>
                  <div class="text-[11px] text-theme-text-muted">Decisive advantage (Eval &ge; +1.50)</div>
                </div>
              </div>
              <div class="flex items-center gap-2">
                <span class="text-xs font-mono font-semibold px-2 py-0.5 rounded bg-theme-panel/80 border border-theme-border text-theme-text-muted group-hover:text-amber-300">
                  [1]
                </span>
                <CheckCircle2 v-if="result && result.actualCategory === 'white_winning'" class="w-5 h-5 text-emerald-400" />
              </div>
            </button>

            <!-- Equal Button -->
            <button
              class="w-full group px-4 py-3.5 rounded-xl border text-left transition-all duration-150 flex items-center justify-between relative overflow-hidden"
              :class="[
                result
                  ? result.actualCategory === 'equal'
                    ? 'bg-emerald-500/15 border-emerald-500 text-emerald-300 ring-2 ring-emerald-500/50'
                    : result.userChoice === 'equal'
                    ? 'bg-rose-500/15 border-rose-500 text-rose-300'
                    : 'opacity-40 border-theme-border bg-theme-panel/40'
                  : 'bg-theme-panel hover:bg-theme-hover border-theme-border hover:border-amber-500/60 text-theme-text-primary hover:shadow-lg hover:-translate-y-0.5 active:translate-y-0'
              ]"
              :disabled="!isAnswering || !!result"
              @click="handleAnswer('equal')"
            >
              <div class="flex items-center gap-3">
                <div class="w-8 h-8 rounded-lg bg-gray-500/20 flex items-center justify-center font-bold text-sm text-gray-300 border border-gray-500/30">
                  =
                </div>
                <div>
                  <div class="font-bold text-sm">Balanced / Equal</div>
                  <div class="text-[11px] text-theme-text-muted">Dynamic equality (-1.50 &lt; Eval &lt; +1.50)</div>
                </div>
              </div>
              <div class="flex items-center gap-2">
                <span class="text-xs font-mono font-semibold px-2 py-0.5 rounded bg-theme-panel/80 border border-theme-border text-theme-text-muted group-hover:text-amber-300">
                  [2]
                </span>
                <CheckCircle2 v-if="result && result.actualCategory === 'equal'" class="w-5 h-5 text-emerald-400" />
              </div>
            </button>

            <!-- Black Winning Button -->
            <button
              class="w-full group px-4 py-3.5 rounded-xl border text-left transition-all duration-150 flex items-center justify-between relative overflow-hidden"
              :class="[
                result
                  ? result.actualCategory === 'black_winning'
                    ? 'bg-emerald-500/15 border-emerald-500 text-emerald-300 ring-2 ring-emerald-500/50'
                    : result.userChoice === 'black_winning'
                    ? 'bg-rose-500/15 border-rose-500 text-rose-300'
                    : 'opacity-40 border-theme-border bg-theme-panel/40'
                  : 'bg-theme-panel hover:bg-theme-hover border-theme-border hover:border-amber-500/60 text-theme-text-primary hover:shadow-lg hover:-translate-y-0.5 active:translate-y-0'
              ]"
              :disabled="!isAnswering || !!result"
              @click="handleAnswer('black_winning')"
            >
              <div class="flex items-center gap-3">
                <div class="w-8 h-8 rounded-lg bg-black/40 flex items-center justify-center font-bold text-sm text-white border border-gray-700">
                  &minus;
                </div>
                <div>
                  <div class="font-bold text-sm">Black is Winning</div>
                  <div class="text-[11px] text-theme-text-muted">Decisive advantage (Eval &le; -1.50)</div>
                </div>
              </div>
              <div class="flex items-center gap-2">
                <span class="text-xs font-mono font-semibold px-2 py-0.5 rounded bg-theme-panel/80 border border-theme-border text-theme-text-muted group-hover:text-amber-300">
                  [3]
                </span>
                <CheckCircle2 v-if="result && result.actualCategory === 'black_winning'" class="w-5 h-5 text-emerald-400" />
              </div>
            </button>
          </div>

          <!-- Feedback card -->
          <div
            v-if="result"
            class="mt-4 p-3.5 rounded-xl border animate-in fade-in slide-in-from-bottom-2 duration-150"
            :class="result.isCorrect ? 'bg-emerald-500/10 border-emerald-500/30' : 'bg-rose-500/10 border-rose-500/30'"
          >
            <div class="flex items-center justify-between mb-1.5">
              <div class="flex items-center gap-2">
                <CheckCircle2 v-if="result.isCorrect" class="w-4 h-4 text-emerald-400" />
                <AlertTriangle v-else class="w-4 h-4 text-rose-400" />
                <span class="font-bold text-xs" :class="result.isCorrect ? 'text-emerald-300' : 'text-rose-300'">
                  {{ result.isCorrect ? 'CORRECT JUDGMENT' : 'MISJUDGMENT' }}
                </span>
              </div>
              <div class="flex items-center gap-2">
                <span class="px-2 py-0.5 rounded bg-theme-panel/80 font-mono font-bold text-xs text-amber-300 border border-amber-500/30">
                  Eval: {{ result.scoreFormatted }}
                </span>
                <span v-if="result.isCorrect" class="text-xs font-bold text-emerald-400">
                  +{{ result.pointsAwarded }} pts
                </span>
              </div>
            </div>
            <p class="text-xs text-theme-text-primary leading-relaxed">
              {{ result.explanation }}
            </p>
          </div>

          <!-- Bottom Action Controls -->
          <div class="mt-4 pt-3 border-t border-theme-border/60 flex items-center justify-between gap-3">
            <button
              class="px-3 py-1.5 text-xs text-theme-text-muted hover:text-theme-text-primary rounded-lg border border-theme-border hover:bg-theme-hover transition-colors flex items-center gap-1.5"
              @click="exploreInMainBoard"
            >
              <Eye class="w-3.5 h-3.5" />
              Explore on Main Board
            </button>

            <div class="flex items-center gap-2">
              <button
                class="px-3 py-1.5 text-xs text-theme-text-muted hover:text-theme-text-primary rounded-lg border border-theme-border hover:bg-theme-hover transition-colors flex items-center gap-1.5"
                @click="restartSession"
              >
                <RotateCcw class="w-3.5 h-3.5" />
                Reset
              </button>

              <button
                class="px-4 py-1.5 text-xs font-semibold rounded-lg bg-amber-500 hover:bg-amber-400 text-black transition-colors flex items-center gap-1.5 shadow-sm"
                @click="nextChallenge"
              >
                <span>Next</span>
                <ChevronRight class="w-4 h-4" />
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
