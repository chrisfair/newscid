<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import {
  X,
  Shield,
  ShieldAlert,
  ShieldCheck,
  Flame,
  Trophy,
  RotateCcw,
  ChevronRight,
  Eye,
  CheckCircle2,
  AlertTriangle,
  Play,
  Zap,
  Swords,
  Pickaxe,
  Search,
  Volume2,
  VolumeX,
  Sparkles
} from '@lucide/vue'
import Chessboard, { type BoardArrow } from './Chessboard.vue'
import {
  AnalyzeBoardSafety,
  GetSafetyDrillBatch,
  GenerateSafetyDrillsFromDatabase,
  SubmitSafetyDrillAnswer
} from '../../wailsjs/go/main/App'
import { training } from '../../wailsjs/go/models'
import { playMoveSound, playCaptureSound, playCheckSound } from '../utils/sound'

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

const HIGH_SCORE_KEY = 'newscid:piece_safety_high_score'
const RATING_KEY = 'newscid:piece_safety_rating'

// Mode and tab state
type DrillModeTab = 'rapid' | 'defend' | 'punish' | 'mine' | 'audit'
const activeTab = ref<DrillModeTab>('rapid')
const difficulty = ref<string>('all')

// Drill State
const drills = ref<training.SafetyDrill[]>([])
const currentIndex = ref<number>(0)
const currentDrill = computed(() => drills.value[currentIndex.value] || null)
const currentFEN = ref<string>(props.initialFEN)

// Selection & feedback
const selectedSquares = ref<Set<string>>(new Set())
const feedback = ref<training.SafetyDrillFeedback | null>(null)
const isRevealed = ref<boolean>(false)
const isLoading = ref<boolean>(false)
const fullAuditReport = ref<training.BoardSafetyReport | null>(null)

// Gamification stats
const score = ref<number>(0)
const streak = ref<number>(0)
const bestStreak = ref<number>(0)
const totalAnswered = ref<number>(0)
const totalCorrect = ref<number>(0)
const highScore = ref<number>(parseInt(localStorage.getItem(HIGH_SCORE_KEY) || '0', 10))
const userRating = ref<number>(parseInt(localStorage.getItem(RATING_KEY) || '1500', 10))

// Timer
const timerSeconds = ref<number>(15)
const maxTime = ref<number>(15)
let timerId: number | null = null
const isTimerActive = ref<boolean>(false)

// Dynamic arrows for attacks/defenders
const dynamicArrows = computed<BoardArrow[]>(() => {
  const arrows: BoardArrow[] = []
  if (!fullAuditReport.value) return arrows

  for (const piece of fullAuditReport.value.pieces) {
    const isTarget = selectedSquares.value.has(piece.square) || isRevealed.value || piece.status === 'hanging'
    if (!isTarget) continue

    // Red arrows for attackers
    for (const atkSq of piece.attackers) {
      arrows.push({
        from: atkSq,
        to: piece.square,
        color: '#ef4444',
        style: 'solid'
      })
    }

    // Green arrows for defenders
    for (const defSq of piece.defenders) {
      arrows.push({
        from: defSq,
        to: piece.square,
        color: '#10b981',
        style: 'dashed'
      })
    }
  }

  return arrows
})

// Highlight threat squares on board
const boardThreatSquares = computed<string[]>(() => {
  if (isRevealed.value && fullAuditReport.value) {
    return fullAuditReport.value.hangingSquares
  }
  return Array.from(selectedSquares.value)
})

// Computed performance
const accuracyPct = computed(() => {
  if (totalAnswered.value === 0) return 100
  return Math.round((totalCorrect.value / totalAnswered.value) * 100)
})

const rankTitle = computed(() => {
  const r = userRating.value
  if (r >= 2200) return 'Grandmaster of Vision'
  if (r >= 2000) return 'Iron Fortress'
  if (r >= 1800) return 'Tactical Hawk'
  if (r >= 1600) return 'Vigilant Guard'
  if (r >= 1400) return 'Board Sentinel'
  return 'Careful Apprentice'
})

// Fetch batch of drills
async function loadDrillBatch() {
  isLoading.value = true
  try {
    if (activeTab.value === 'mine') {
      const result = await GenerateSafetyDrillsFromDatabase(15)
      drills.value = result || []
    } else {
      const diff = difficulty.value === 'all' ? '' : difficulty.value
      const result = await GetSafetyDrillBatch(20, diff)
      drills.value = result || []
    }
    currentIndex.value = 0
    setupCurrentDrill()
  } catch (err) {
    console.error('Failed to load safety drills:', err)
  } finally {
    isLoading.value = false
  }
}

// Setup active drill
async function setupCurrentDrill() {
  selectedSquares.value.clear()
  feedback.value = null
  isRevealed.value = false
  fullAuditReport.value = null

  if (activeTab.value === 'audit') {
    currentFEN.value = props.initialFEN
    await runAuditForPosition(currentFEN.value)
    return
  }

  const drill = currentDrill.value
  if (!drill) return

  currentFEN.value = drill.fen
  resetTimer()

  // Pre-calculate full audit report for rich explanations
  try {
    const report = await AnalyzeBoardSafety(drill.fen)
    fullAuditReport.value = report
  } catch (e) {
    console.error('Audit calculation error:', e)
  }
}

// Run audit for arbitrary FEN
async function runAuditForPosition(fen: string) {
  try {
    isLoading.value = true
    const report = await AnalyzeBoardSafety(fen)
    fullAuditReport.value = report
    isRevealed.value = true
  } catch (e) {
    console.error('Failed to run position safety audit:', e)
  } finally {
    isLoading.value = false
  }
}

// Timer mechanics
function resetTimer() {
  if (timerId) clearInterval(timerId)
  if (activeTab.value !== 'rapid') {
    isTimerActive.value = false
    return
  }
  timerSeconds.value = maxTime.value
  isTimerActive.value = true
  timerId = window.setInterval(() => {
    if (timerSeconds.value > 0) {
      timerSeconds.value--
    } else {
      // Time expired: auto-submit
      if (!feedback.value) {
        checkAnswer()
      }
    }
  }, 1000)
}

// Square selection toggle
function handleSquareClick(sq: string) {
  if (isRevealed.value) return

  if (selectedSquares.value.has(sq)) {
    selectedSquares.value.delete(sq)
  } else {
    selectedSquares.value.add(sq)
    playMoveSound()
  }
}

// Submit answer
async function checkAnswer() {
  if (timerId) clearInterval(timerId)
  isTimerActive.value = false

  const drill = currentDrill.value
  if (!drill) return

  const squares = Array.from(selectedSquares.value)
  try {
    const res = await SubmitSafetyDrillAnswer(drill, squares, '')
    feedback.value = res
    isRevealed.value = true
    totalAnswered.value++

    if (res.isCorrect) {
      playCaptureSound()
      streak.value++
      if (streak.value > bestStreak.value) {
        bestStreak.value = streak.value
      }
      totalCorrect.value++

      // Score calculation with speed multiplier
      const timeBonus = activeTab.value === 'rapid' ? timerSeconds.value * 5 : 0
      const roundScore = res.score + (streak.value * 10) + timeBonus
      score.value += roundScore

      // Rating boost
      userRating.value += Math.min(25, 10 + streak.value * 2)

      if (score.value > highScore.value) {
        highScore.value = score.value
        localStorage.setItem(HIGH_SCORE_KEY, highScore.value.toString())
      }
      localStorage.setItem(RATING_KEY, userRating.value.toString())
    } else {
      playCheckSound()
      streak.value = 0
      userRating.value = Math.max(800, userRating.value - 12)
      localStorage.setItem(RATING_KEY, userRating.value.toString())
    }
  } catch (err) {
    console.error('Failed to submit safety drill:', err)
  }
}

// Next drill
function nextDrill() {
  if (currentIndex.value < drills.value.length - 1) {
    currentIndex.value++
    setupCurrentDrill()
  } else {
    loadDrillBatch()
  }
}

// Reveal full audit without scoring
async function revealAudit() {
  if (timerId) clearInterval(timerId)
  isTimerActive.value = false
  isRevealed.value = true
  if (!fullAuditReport.value && currentFEN.value) {
    await runAuditForPosition(currentFEN.value)
  }
}

// Keyboard shortcuts (Enter = submit, Space = next)
function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    emit('close')
  } else if (e.key === 'Enter') {
    if (!isRevealed.value) {
      checkAnswer()
    } else {
      nextDrill()
    }
  } else if (e.key === ' ' && isRevealed.value) {
    e.preventDefault()
    nextDrill()
  }
}

onMounted(() => {
  window.addEventListener('keydown', handleKeydown)
  loadDrillBatch()
})

onUnmounted(() => {
  window.removeEventListener('keydown', handleKeydown)
  if (timerId) clearInterval(timerId)
})

watch(activeTab, () => {
  loadDrillBatch()
})

watch(difficulty, () => {
  loadDrillBatch()
})
</script>

<template>
  <div class="fixed inset-0 bg-black/80 backdrop-blur-xs flex items-center justify-center z-50 p-4 select-none">
    <div class="bg-zinc-900 border border-zinc-700/80 rounded-2xl shadow-2xl w-full max-w-6xl max-h-[92vh] flex flex-col overflow-hidden text-zinc-100 animate-in fade-in zoom-in-95 duration-200">
      
      <!-- HEADER -->
      <div class="flex items-center justify-between px-6 py-4 border-b border-zinc-800 bg-zinc-950/60">
        <div class="flex items-center gap-3">
          <div class="p-2 rounded-xl bg-emerald-500/10 border border-emerald-500/20 text-emerald-400">
            <ShieldAlert class="w-6 h-6" />
          </div>
          <div>
            <h2 class="text-base font-semibold tracking-wide flex items-center gap-2">
              Piece Safety Scanner
              <span class="text-xs font-normal px-2 py-0.5 rounded-full bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
                Don't Hang Pieces
              </span>
            </h2>
            <p class="text-xs text-zinc-400">Identify undefended, underprotected, and hanging pieces before blunders happen</p>
          </div>
        </div>

        <!-- STATS BADGES -->
        <div class="flex items-center gap-3">
          <div class="flex items-center gap-1.5 px-3 py-1 rounded-lg bg-zinc-800/80 border border-zinc-700 text-xs">
            <Flame class="w-4 h-4 text-amber-400" />
            <span class="text-zinc-400">Streak:</span>
            <span class="font-bold text-amber-400">{{ streak }}</span>
          </div>

          <div class="flex items-center gap-1.5 px-3 py-1 rounded-lg bg-zinc-800/80 border border-zinc-700 text-xs">
            <Trophy class="w-4 h-4 text-yellow-400" />
            <span class="text-zinc-400">Score:</span>
            <span class="font-bold text-yellow-400">{{ score }}</span>
          </div>

          <div class="flex items-center gap-1.5 px-3 py-1 rounded-lg bg-zinc-800/80 border border-zinc-700 text-xs">
            <ShieldCheck class="w-4 h-4 text-emerald-400" />
            <span class="text-zinc-400">Rating:</span>
            <span class="font-bold text-emerald-400">{{ userRating }}</span>
          </div>

          <button
            class="p-2 rounded-lg hover:bg-zinc-800 text-zinc-400 hover:text-zinc-200 transition-colors ml-2"
            @click="emit('close')"
          >
            <X class="w-5 h-5" />
          </button>
        </div>
      </div>

      <!-- MODE SELECTOR TABS -->
      <div class="flex items-center justify-between px-6 py-2.5 bg-zinc-950/40 border-b border-zinc-800 text-xs">
        <div class="flex items-center gap-1.5">
          <button
            class="px-3 py-1.5 rounded-lg font-medium transition-all flex items-center gap-2"
            :class="activeTab === 'rapid' ? 'bg-emerald-600 text-white shadow-xs' : 'text-zinc-400 hover:bg-zinc-800 hover:text-zinc-200'"
            @click="activeTab = 'rapid'"
          >
            <Zap class="w-3.5 h-3.5" /> Rapid Scanner (Blitz)
          </button>

          <button
            class="px-3 py-1.5 rounded-lg font-medium transition-all flex items-center gap-2"
            :class="activeTab === 'defend' ? 'bg-emerald-600 text-white shadow-xs' : 'text-zinc-400 hover:bg-zinc-800 hover:text-zinc-200'"
            @click="activeTab = 'defend'"
          >
            <Shield class="w-3.5 h-3.5" /> Defend Your Camp
          </button>

          <button
            class="px-3 py-1.5 rounded-lg font-medium transition-all flex items-center gap-2"
            :class="activeTab === 'punish' ? 'bg-emerald-600 text-white shadow-xs' : 'text-zinc-400 hover:bg-zinc-800 hover:text-zinc-200'"
            @click="activeTab = 'punish'"
          >
            <Swords class="w-3.5 h-3.5" /> Punish Blunders
          </button>

          <button
            class="px-3 py-1.5 rounded-lg font-medium transition-all flex items-center gap-2"
            :class="activeTab === 'mine' ? 'bg-emerald-600 text-white shadow-xs' : 'text-zinc-400 hover:bg-zinc-800 hover:text-zinc-200'"
            @click="activeTab = 'mine'"
          >
            <Pickaxe class="w-3.5 h-3.5" /> Mine from Active Database
          </button>

          <button
            class="px-3 py-1.5 rounded-lg font-medium transition-all flex items-center gap-2"
            :class="activeTab === 'audit' ? 'bg-emerald-600 text-white shadow-xs' : 'text-zinc-400 hover:bg-zinc-800 hover:text-zinc-200'"
            @click="activeTab = 'audit'"
          >
            <Search class="w-3.5 h-3.5" /> Position Safety Audit
          </button>
        </div>

        <!-- DIFFICULTY SELECTOR -->
        <div v-if="activeTab !== 'audit'" class="flex items-center gap-2">
          <span class="text-zinc-500 font-mono">Difficulty:</span>
          <select
            v-model="difficulty"
            class="bg-zinc-800 border border-zinc-700 text-zinc-200 px-2.5 py-1 rounded-md text-xs cursor-pointer focus:outline-hidden focus:border-emerald-500"
          >
            <option value="all">All Difficulties</option>
            <option value="beginner">Beginner (Direct Free Pieces)</option>
            <option value="intermediate">Intermediate (Exchanges & Pins)</option>
          </select>
        </div>
      </div>

      <!-- MAIN CONTENT GRID -->
      <div class="flex-1 grid grid-cols-1 lg:grid-cols-12 gap-6 p-6 overflow-y-auto">
        
        <!-- LEFT: CHESSBOARD COLUMN (7 cols) -->
        <div class="lg:col-span-7 flex flex-col items-center justify-center">
          <div class="relative w-full max-w-[500px] aspect-square rounded-xl overflow-hidden shadow-2xl border border-zinc-700/60 bg-zinc-950">
            <Chessboard
              :fen="currentFEN"
              :piece-set="pieceSet"
              :board-theme="boardTheme"
              :arrows="dynamicArrows"
              :threat-squares="boardThreatSquares"
              @move="() => {}"
            />

            <!-- Interactive Clickable Overlay Squares -->
            <div class="absolute inset-0 grid grid-cols-8 grid-rows-8 z-20">
              <div
                v-for="idx in 64"
                :key="idx"
                class="cursor-pointer transition-all hover:bg-emerald-500/20 flex items-center justify-center relative"
                :class="{
                  'ring-4 ring-inset ring-amber-400 bg-amber-400/20': selectedSquares.has(
                    String.fromCharCode('a'.charCodeAt(0) + ((idx - 1) % 8)) + (8 - Math.floor((idx - 1) / 8))
                  ),
                  'ring-4 ring-inset ring-emerald-500 bg-emerald-500/30': isRevealed && feedback?.correctSquares?.includes(
                    String.fromCharCode('a'.charCodeAt(0) + ((idx - 1) % 8)) + (8 - Math.floor((idx - 1) / 8))
                  ),
                  'ring-4 ring-inset ring-rose-500 bg-rose-500/30': isRevealed && feedback?.wrongSquares?.includes(
                    String.fromCharCode('a'.charCodeAt(0) + ((idx - 1) % 8)) + (8 - Math.floor((idx - 1) / 8))
                  )
                }"
                @click="handleSquareClick(
                  String.fromCharCode('a'.charCodeAt(0) + ((idx - 1) % 8)) + (8 - Math.floor((idx - 1) / 8))
                )"
              >
                <!-- Hanging Piece Alert Pulse -->
                <span
                  v-if="isRevealed && fullAuditReport?.hangingSquares?.includes(
                    String.fromCharCode('a'.charCodeAt(0) + ((idx - 1) % 8)) + (8 - Math.floor((idx - 1) / 8))
                  )"
                  class="absolute w-3 h-3 rounded-full bg-rose-500 animate-ping opacity-75"
                />
              </div>
            </div>
          </div>

          <!-- RAPID TIMER PROGRESS BAR -->
          <div v-if="activeTab === 'rapid' && isTimerActive" class="w-full max-w-[500px] mt-3">
            <div class="flex justify-between text-[11px] text-zinc-400 mb-1">
              <span>Time Remaining</span>
              <span :class="timerSeconds <= 5 ? 'text-rose-400 font-bold animate-pulse' : 'text-zinc-300'">
                {{ timerSeconds }}s
              </span>
            </div>
            <div class="w-full h-1.5 bg-zinc-800 rounded-full overflow-hidden">
              <div
                class="h-full transition-all duration-1000 ease-linear rounded-full"
                :class="timerSeconds > 5 ? 'bg-emerald-500' : 'bg-rose-500'"
                :style="{ width: `${(timerSeconds / maxTime) * 100}%` }"
              />
            </div>
          </div>
        </div>

        <!-- RIGHT: CHALLENGE & AUDIT DOSSIER (5 cols) -->
        <div class="lg:col-span-5 flex flex-col gap-4 justify-between">
          
          <!-- TOP CARD: PROMPT & DRILL INFO -->
          <div class="bg-zinc-950/70 border border-zinc-800 rounded-xl p-5 shadow-inner">
            <div class="flex items-center justify-between mb-3">
              <span class="text-xs font-semibold uppercase tracking-wider text-emerald-400 flex items-center gap-1.5">
                <Sparkles class="w-3.5 h-3.5" />
                {{ currentDrill?.source || 'Safety Challenge' }}
              </span>
              <span class="text-[11px] px-2 py-0.5 rounded-full bg-zinc-800 text-zinc-400 border border-zinc-700">
                Drill #{{ currentIndex + 1 }} of {{ drills.length }}
              </span>
            </div>

            <h3 class="text-sm font-semibold text-zinc-100 mb-2">
              {{ currentDrill?.title || 'Scan for Vulnerabilities' }}
            </h3>

            <p class="text-xs text-zinc-300 leading-relaxed mb-4 bg-zinc-900/90 p-3 rounded-lg border border-zinc-800/80">
              {{ currentDrill?.prompt || 'Click all squares containing hanging or underdefended pieces!' }}
            </p>

            <!-- SELECTED SQUARES CHIPS -->
            <div class="flex items-center gap-2 flex-wrap min-h-[32px]">
              <span class="text-[11px] text-zinc-500">Selected Squares:</span>
              <span
                v-if="selectedSquares.size === 0"
                class="text-[11px] text-zinc-600 italic"
              >
                (Click squares on board to select)
              </span>
              <span
                v-for="sq in Array.from(selectedSquares)"
                :key="sq"
                class="px-2 py-0.5 rounded-md bg-amber-500/20 border border-amber-500/30 text-amber-300 font-mono text-xs cursor-pointer hover:bg-amber-500/30"
                @click="handleSquareClick(sq)"
              >
                {{ sq }} ✕
              </span>
            </div>
          </div>

          <!-- FEEDBACK & RESULT CARD -->
          <div
            v-if="feedback"
            class="p-4 rounded-xl border transition-all"
            :class="feedback.isCorrect ? 'bg-emerald-950/30 border-emerald-500/40 text-emerald-200' : 'bg-rose-950/30 border-rose-500/40 text-rose-200'"
          >
            <div class="flex items-center gap-2 mb-2">
              <CheckCircle2 v-if="feedback.isCorrect" class="w-5 h-5 text-emerald-400" />
              <AlertTriangle v-else class="w-5 h-5 text-rose-400" />
              <span class="font-semibold text-sm">
                {{ feedback.isCorrect ? 'Outstanding! All Hanging Pieces Identified.' : 'Missed Vulnerabilities!' }}
              </span>
            </div>
            <p class="text-xs leading-relaxed text-zinc-300">
              {{ feedback.explanation }}
            </p>
            <div v-if="feedback.bestMoveSAN" class="mt-2 text-xs font-mono bg-zinc-900/80 px-2.5 py-1.5 rounded-md border border-zinc-800 text-emerald-400">
              Winning Refutation: <strong>{{ feedback.bestMoveSAN }}</strong> ({{ feedback.bestMoveUCI }})
            </div>
          </div>

          <!-- FULL SAFETY AUDIT BREAKDOWN -->
          <div v-if="fullAuditReport" class="flex-1 bg-zinc-950/70 border border-zinc-800 rounded-xl p-4 overflow-y-auto max-h-[220px]">
            <h4 class="text-xs font-semibold text-zinc-300 mb-2 flex items-center justify-between">
              <span>Position Safety Audit</span>
              <span class="text-[11px] text-zinc-500 font-normal">
                {{ fullAuditReport.hangingPieces.length }} Hanging, {{ fullAuditReport.loosePieces.length }} Loose
              </span>
            </h4>

            <div v-if="fullAuditReport.hangingPieces.length === 0" class="text-xs text-emerald-400/90 py-2">
              ✔ No hanging pieces detected. Camp is defensively sound.
            </div>

            <div class="space-y-2">
              <div
                v-for="p in fullAuditReport.hangingPieces"
                :key="p.square"
                class="p-2.5 rounded-lg bg-rose-500/10 border border-rose-500/20 text-xs flex items-start gap-2.5"
              >
                <AlertTriangle class="w-4 h-4 text-rose-400 shrink-0 mt-0.5" />
                <div class="flex-1">
                  <div class="flex items-center justify-between font-mono font-medium text-rose-300">
                    <span>{{ p.color }} {{ p.piece }} on {{ p.square }}</span>
                    <span class="text-rose-400 font-bold">-{{ p.netLoss }} cp</span>
                  </div>
                  <p class="text-[11px] text-zinc-400 mt-0.5">{{ p.explanation }}</p>
                  <div class="flex items-center gap-3 text-[10px] text-zinc-500 mt-1 font-mono">
                    <span>Attackers: {{ p.attackers.join(', ') || 'none' }}</span>
                    <span>Defenders: {{ p.defenders.join(', ') || '0' }}</span>
                  </div>
                </div>
              </div>

              <!-- Loose Pieces (LPDO) -->
              <div
                v-for="p in fullAuditReport.loosePieces"
                :key="p.square"
                class="p-2 rounded-lg bg-zinc-900 border border-zinc-800 text-xs flex items-center justify-between text-zinc-400"
              >
                <span>Loose {{ p.color }} {{ p.piece }} on {{ p.square }}</span>
                <span class="text-[10px] px-1.5 py-0.5 rounded bg-zinc-800 text-zinc-400 font-mono">0 Defenders</span>
              </div>
            </div>
          </div>

          <!-- ACTION BUTTONS -->
          <div class="flex items-center gap-3 pt-2">
            <button
              v-if="!isRevealed"
              class="flex-1 py-3 rounded-xl bg-emerald-600 hover:bg-emerald-500 text-white font-semibold text-xs transition-all shadow-lg flex items-center justify-center gap-2 cursor-pointer"
              @click="checkAnswer"
            >
              <CheckCircle2 class="w-4 h-4" /> Check Answer (Enter)
            </button>

            <button
              v-else
              class="flex-1 py-3 rounded-xl bg-emerald-600 hover:bg-emerald-500 text-white font-semibold text-xs transition-all shadow-lg flex items-center justify-center gap-2 cursor-pointer"
              @click="nextDrill"
            >
              Next Challenge (Space) <ChevronRight class="w-4 h-4" />
            </button>

            <button
              v-if="!isRevealed"
              class="px-4 py-3 rounded-xl bg-zinc-800 hover:bg-zinc-700 text-zinc-300 hover:text-white text-xs transition-colors flex items-center gap-2 cursor-pointer"
              @click="revealAudit"
            >
              <Eye class="w-4 h-4" /> Reveal Audit
            </button>
          </div>
        </div>

      </div>

      <!-- FOOTER INFO BAR -->
      <div class="px-6 py-2.5 bg-zinc-950/80 border-t border-zinc-800 flex items-center justify-between text-[11px] text-zinc-500">
        <div class="flex items-center gap-4">
          <span>Rank: <strong class="text-zinc-300">{{ rankTitle }}</strong></span>
          <span>Accuracy: <strong class="text-zinc-300">{{ accuracyPct }}%</strong> ({{ totalCorrect }}/{{ totalAnswered }})</span>
          <span>Best Streak: <strong class="text-zinc-300">{{ bestStreak }}</strong></span>
        </div>
        <div class="flex items-center gap-2 font-mono">
          <span>Keyboard: [Enter] Submit • [Space] Next • [Esc] Close</span>
        </div>
      </div>

    </div>
  </div>
</template>
