<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import {
  X,
  Shield,
  Eye,
  CheckCircle2,
  AlertTriangle,
  RotateCcw,
  ChevronRight,
  Sparkles,
  Database,
  Flame,
  Trophy,
  TrendingUp,
  Radar,
  ArrowRight,
  Crosshair,
  Volume2,
  VolumeX,
  Lightbulb
} from '@lucide/vue'
import Chessboard, { type BoardArrow } from './Chessboard.vue'
import {
  GetProphylaxisDrillBatch,
  GenerateProphylaxisDrillsFromDatabase,
  SubmitProphylaxisAnswer
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

const HIGH_SCORE_KEY = 'newscid:prophylaxis_high_score'
const RATING_KEY = 'newscid:prophylaxis_rating'

// Mode and tabs
const sourceMode = ref<'curated' | 'database'>('curated')
const selectedCategory = ref<string>('all')
const showThreatRadar = ref<boolean>(false)

// Drill State
const drills = ref<training.ProphylaxisDrill[]>([])
const currentIndex = ref<number>(0)
const currentDrill = computed(() => drills.value[currentIndex.value] || null)
const currentFEN = ref<string>(props.initialFEN)

// Submission & Feedback
const feedback = ref<training.ProphylaxisFeedback | null>(null)
const isRevealed = ref<boolean>(false)
const isLoading = ref<boolean>(false)
const moveInput = ref<string>('')

// Gamification stats
const score = ref<number>(0)
const streak = ref<number>(0)
const bestStreak = ref<number>(0)
const totalAnswered = ref<number>(0)
const totalCorrect = ref<number>(0)
const highScore = ref<number>(parseInt(localStorage.getItem(HIGH_SCORE_KEY) || '0', 10))
const userRating = ref<number>(parseInt(localStorage.getItem(RATING_KEY) || '1500', 10))

// Threat Radar Arrows
const boardArrows = computed<BoardArrow[]>(() => {
  const arrows: BoardArrow[] = []
  if (!currentDrill.value) return arrows

  // Show opponent threat arrow if threat radar is active or position is revealed
  if ((showThreatRadar.value || isRevealed.value) && currentDrill.value.threatArrowFrom && currentDrill.value.threatArrowTo) {
    arrows.push({
      from: currentDrill.value.threatArrowFrom,
      to: currentDrill.value.threatArrowTo,
      color: 'rgba(239, 68, 68, 0.85)' // Red threat arrow
    })
  }

  // Show winning move arrow if revealed or answered correctly
  if ((isRevealed.value || feedback.value?.isCorrect) && currentDrill.value.winningMoveUCI && currentDrill.value.winningMoveUCI.length >= 4) {
    const from = currentDrill.value.winningMoveUCI.substring(0, 2)
    const to = currentDrill.value.winningMoveUCI.substring(2, 4)
    arrows.push({
      from,
      to,
      color: 'rgba(34, 197, 94, 0.85)' // Emerald solution arrow
    })
  }

  return arrows
})

async function loadDrills() {
  isLoading.value = true
  feedback.value = null
  isRevealed.value = false
  moveInput.value = ''

  try {
    let list: training.ProphylaxisDrill[] = []
    if (sourceMode.value === 'database') {
      list = await GenerateProphylaxisDrillsFromDatabase(15)
    } else {
      list = await GetProphylaxisDrillBatch(15, selectedCategory.value)
    }
    drills.value = list
    currentIndex.value = 0
    if (list.length > 0) {
      currentFEN.value = list[0].fen
    }
  } catch (err) {
    console.error('Failed to load prophylaxis drills:', err)
  } finally {
    isLoading.value = false
  }
}

async function handleMove(uci: string) {
  if (!currentDrill.value || feedback.value) return

  totalAnswered.value++
  try {
    const res = await SubmitProphylaxisAnswer(currentDrill.value.id, uci)
    feedback.value = res

    if (res.isCorrect) {
      streak.value++
      if (streak.value > bestStreak.value) bestStreak.value = streak.value
      totalCorrect.value++
      score.value += res.pointsAwarded
      if (score.value > highScore.value) {
        highScore.value = score.value
        localStorage.setItem(HIGH_SCORE_KEY, score.value.toString())
      }
      userRating.value += 16
      localStorage.setItem(RATING_KEY, userRating.value.toString())
      playCheckSound()
    } else {
      streak.value = 0
      userRating.value = Math.max(900, userRating.value - 12)
      localStorage.setItem(RATING_KEY, userRating.value.toString())
      playCaptureSound()
    }
  } catch (err) {
    console.error('Error submitting prophylaxis move:', err)
  }
}

function handleInputSubmit() {
  const val = moveInput.value.trim().toLowerCase()
  if (!val) return
  handleMove(val)
}

function revealSolution() {
  if (!currentDrill.value || isRevealed.value) return
  isRevealed.value = true
  if (!feedback.value) {
    feedback.value = new training.ProphylaxisFeedback({
      isCorrect: false,
      pointsAwarded: 0,
      bestMoveUCI: currentDrill.value.winningMoveUCI,
      bestMoveSAN: currentDrill.value.winningMoveSAN,
      explanation: currentDrill.value.explanation,
      opponentThreat: currentDrill.value.opponentThreat
    })
    streak.value = 0
  }
}

function nextDrill() {
  feedback.value = null
  isRevealed.value = false
  moveInput.value = ''

  if (currentIndex.value + 1 < drills.value.length) {
    currentIndex.value++
    currentFEN.value = drills.value[currentIndex.value].fen
    playMoveSound()
  } else {
    loadDrills()
  }
}

function exploreInMainBoard() {
  if (currentDrill.value) {
    emit('loadPosition', currentDrill.value.fen)
    emit('close')
  }
}

onMounted(() => {
  loadDrills()
})
</script>

<template>
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/75 backdrop-blur-sm p-4 select-none">
    <div
      class="bg-theme-panel border border-theme-border rounded-xl shadow-2xl flex flex-col w-full max-w-4xl overflow-hidden animate-in fade-in zoom-in-95 duration-150"
      style="height: 88vh; max-height: 850px;"
    >
      <!-- Header -->
      <div class="px-6 py-3.5 border-b border-theme-border flex items-center justify-between bg-theme-panel-header">
        <div class="flex items-center gap-3">
          <div class="p-2 bg-emerald-500/10 text-emerald-400 rounded-lg border border-emerald-500/20">
            <Shield class="w-5 h-5 text-emerald-400" />
          </div>
          <div>
            <div class="flex items-center gap-2">
              <h2 class="text-base font-bold text-theme-text-primary tracking-wide">Quiet Moves &amp; Prophylaxis Drill</h2>
              <span class="px-2 py-0.5 text-xs font-semibold rounded bg-emerald-500/20 text-emerald-300 border border-emerald-500/30">
                Non-Forcing Tactics Trainer
              </span>
            </div>
            <p class="text-xs text-theme-text-muted mt-0.5">
              Anticipate the opponent's counterplay and find the quiet masterstroke that defuses threats.
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
              :class="sourceMode === 'curated' ? 'bg-emerald-600 text-white font-semibold shadow' : 'text-theme-text-muted hover:text-theme-text-primary'"
              @click="sourceMode = 'curated'; loadDrills()"
            >
              <Sparkles class="w-3.5 h-3.5" />
              Grandmaster
            </button>
            <button
              class="px-2.5 py-1 rounded transition-colors flex items-center gap-1.5"
              :class="sourceMode === 'database' ? 'bg-emerald-600 text-white font-semibold shadow' : 'text-theme-text-muted hover:text-theme-text-primary'"
              @click="sourceMode = 'database'; loadDrills()"
            >
              <Database class="w-3.5 h-3.5" />
              Active DB
            </button>
          </div>

          <button
            class="text-theme-text-muted hover:text-theme-text-primary p-1.5 rounded-lg hover:bg-theme-hover transition-colors"
            title="Close"
            @click="emit('close')"
          >
            <X class="w-5 h-5" />
          </button>
        </div>
      </div>

      <!-- Stats ribbon & Controls -->
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
            <TrendingUp class="w-4 h-4 text-emerald-400" />
            <span class="text-theme-text-muted">Rating:</span>
            <span class="font-bold text-emerald-300 text-sm">{{ userRating }}</span>
          </div>
        </div>

        <div class="flex items-center gap-4">
          <!-- Threat Radar Toggle -->
          <button
            class="px-3 py-1 rounded-lg text-xs font-medium transition-all flex items-center gap-1.5 border"
            :class="showThreatRadar ? 'bg-rose-500/20 text-rose-300 border-rose-500/50 shadow-sm' : 'bg-theme-panel text-theme-text-muted hover:text-theme-text-primary border-theme-border'"
            @click="showThreatRadar = !showThreatRadar"
          >
            <Radar class="w-3.5 h-3.5" :class="{ 'animate-spin': showThreatRadar }" />
            <span>Threat Radar</span>
            <span class="text-[10px] px-1 rounded" :class="showThreatRadar ? 'bg-rose-500/30' : 'bg-theme-panel-header'">
              {{ showThreatRadar ? 'ON' : 'OFF' }}
            </span>
          </button>

          <!-- Category filter -->
          <select
            v-model="selectedCategory"
            class="px-2.5 py-1 bg-theme-panel border border-theme-border rounded-lg text-xs text-theme-text-primary focus:outline-none focus:border-emerald-500"
            @change="loadDrills"
          >
            <option value="all">All Prophylaxis Themes</option>
            <option value="king_safety">King Safety &amp; Unpin</option>
            <option value="piece_restraint">Piece Restraint</option>
            <option value="blockade">Blockade</option>
            <option value="quiet_regroup">Quiet Regrouping</option>
          </select>
        </div>
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
              :flipped="currentDrill?.sideToMove === 'black' || currentFEN.split(' ')[1] === 'b'"
              :interactive="!feedback"
              :arrows="boardArrows"
              class="w-full h-full"
              @move="handleMove"
            />

            <!-- Loading overlay -->
            <div
              v-if="isLoading"
              class="absolute inset-0 bg-theme-panel/80 backdrop-blur-sm flex flex-col items-center justify-center gap-3 z-20"
            >
              <div class="w-8 h-8 border-2 border-emerald-400 border-t-transparent rounded-full animate-spin"></div>
              <span class="text-xs font-semibold text-theme-text-muted tracking-wide">Gathering Positions...</span>
            </div>
          </div>

          <!-- Position metadata caption -->
          <div v-if="currentDrill" class="mt-2 text-center text-xs text-theme-text-muted truncate max-w-sm">
            <span class="font-medium text-theme-text-primary">{{ currentDrill.title }}</span>
            <span v-if="currentDrill.source" class="block text-[11px] text-theme-text-muted/75 mt-0.5">
              {{ currentDrill.source }}
            </span>
          </div>
        </div>

        <!-- Problem Prompt & Prophylaxis Solution Panel -->
        <div class="flex-1 flex flex-col justify-between w-full max-w-md h-full py-1">
          <!-- Turn & Theme Banner -->
          <div>
            <div class="flex items-center justify-between mb-3">
              <span class="px-2.5 py-1 rounded-md text-xs font-bold uppercase tracking-wider" :class="currentDrill?.turn === 'white' ? 'bg-white text-black' : 'bg-gray-800 text-white border border-gray-700'">
                {{ currentDrill?.turn === 'white' ? 'White to Move' : 'Black to Move' }}
              </span>
              <span v-if="currentDrill?.categoryLabel" class="text-xs font-semibold text-emerald-400 flex items-center gap-1">
                <Shield class="w-3.5 h-3.5" />
                {{ currentDrill.categoryLabel }}
              </span>
            </div>

            <!-- Strategic Prompt -->
            <div class="p-3.5 rounded-xl border border-theme-border bg-theme-panel-header/30 mb-3">
              <div class="text-xs font-semibold text-theme-text-muted mb-1 flex items-center gap-1.5">
                <Lightbulb class="w-3.5 h-3.5 text-amber-400" />
                Strategic Objective
              </div>
              <p class="text-xs text-theme-text-primary leading-relaxed font-medium">
                {{ currentDrill?.prompt || 'Anticipate the threat and find the quiet masterstroke.' }}
              </p>
            </div>

            <!-- Threat Radar Alert (Visible if toggle on) -->
            <div
              v-if="showThreatRadar && currentDrill?.opponentThreat"
              class="p-3 rounded-xl border border-rose-500/30 bg-rose-500/10 text-xs text-rose-200 animate-in fade-in duration-150 mb-3"
            >
              <div class="font-bold flex items-center gap-1.5 mb-1 text-rose-300">
                <Radar class="w-3.5 h-3.5" />
                Detected Enemy Threat:
              </div>
              <p class="leading-relaxed text-[11px]">
                {{ currentDrill.opponentThreat }}
              </p>
            </div>
          </div>

          <!-- Move input fallback -->
          <div v-if="!feedback" class="my-auto space-y-3">
            <div class="text-xs text-theme-text-muted">
              Make your move on the board, or type move notation below:
            </div>
            <div class="flex items-center gap-2">
              <input
                v-model="moveInput"
                type="text"
                placeholder="e.g. Kh1, a3, Ne2, g1h1..."
                class="flex-1 px-3 py-2 bg-theme-panel border border-theme-border rounded-lg text-xs text-theme-text-primary focus:outline-none focus:border-emerald-500"
                @keydown.enter="handleInputSubmit"
              />
              <button
                class="px-4 py-2 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white text-xs font-semibold transition-colors"
                @click="handleInputSubmit"
              >
                Submit
              </button>
            </div>
          </div>

          <!-- Feedback card -->
          <div
            v-if="feedback"
            class="p-4 rounded-xl border animate-in fade-in slide-in-from-bottom-2 duration-150 my-auto"
            :class="feedback.isCorrect ? 'bg-emerald-500/10 border-emerald-500/30' : 'bg-rose-500/10 border-rose-500/30'"
          >
            <div class="flex items-center justify-between mb-2">
              <div class="flex items-center gap-2">
                <CheckCircle2 v-if="feedback.isCorrect" class="w-4 h-4 text-emerald-400" />
                <AlertTriangle v-else class="w-4 h-4 text-rose-400" />
                <span class="font-bold text-xs" :class="feedback.isCorrect ? 'text-emerald-300' : 'text-rose-300'">
                  {{ feedback.isCorrect ? 'MASTERFUL PROPHYLAXIS' : 'MISSED THE QUIET MOVE' }}
                </span>
              </div>
              <span class="px-2 py-0.5 rounded bg-theme-panel font-mono font-bold text-xs text-emerald-300 border border-theme-border">
                Solution: {{ feedback.bestMoveSAN }}
              </span>
            </div>
            <p class="text-xs text-theme-text-primary leading-relaxed mb-2">
              {{ feedback.explanation }}
            </p>
            <div v-if="feedback.opponentThreat" class="pt-2 border-t border-theme-border/60 text-[11px] text-theme-text-muted">
              <span class="font-semibold text-rose-300">Defused Threat:</span> {{ feedback.opponentThreat }}
            </div>
          </div>

          <!-- Bottom Action Controls -->
          <div class="mt-4 pt-3 border-t border-theme-border/60 flex items-center justify-between gap-3">
            <div class="flex items-center gap-2">
              <button
                class="px-3 py-1.5 text-xs text-theme-text-muted hover:text-theme-text-primary rounded-lg border border-theme-border hover:bg-theme-hover transition-colors flex items-center gap-1.5"
                @click="exploreInMainBoard"
              >
                <Eye class="w-3.5 h-3.5" />
                Explore on Main Board
              </button>

              <button
                v-if="!feedback && !isRevealed"
                class="px-3 py-1.5 text-xs text-theme-text-muted hover:text-theme-text-primary rounded-lg border border-theme-border hover:bg-theme-hover transition-colors flex items-center gap-1.5"
                @click="revealSolution"
              >
                <Lightbulb class="w-3.5 h-3.5 text-amber-400" />
                Show Solution
              </button>
            </div>

            <button
              class="px-4 py-1.5 text-xs font-semibold rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white transition-colors flex items-center gap-1.5 shadow-sm"
              @click="nextDrill"
            >
              <span>Next Drill</span>
              <ChevronRight class="w-4 h-4" />
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
