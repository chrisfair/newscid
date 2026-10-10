<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { X, Play, RotateCcw, Target, Trophy, Flame } from '@lucide/vue'
import { playMoveSound, playCaptureSound } from '../utils/sound'

const emit = defineEmits<{
  (e: 'close'): void
}>()

const HIGH_SCORE_KEY = 'newscid:coordinate_high_score'

const perspective = ref<'w' | 'b'>('w')
const duration = ref<number>(30) // seconds
const timeLeft = ref<number>(30)
const isRunning = ref<boolean>(false)
const targetSquare = ref<string>('e4')

const score = ref<number>(0)
const totalClicks = ref<number>(0)
const currentStreak = ref<number>(0)
const bestStreak = ref<number>(0)
const highScore = ref<number>(parseInt(localStorage.getItem(HIGH_SCORE_KEY) || '0', 10))

const lastClickedSquare = ref<string | null>(null)
const lastClickStatus = ref<'correct' | 'wrong' | null>(null)

let timerInterval: number | null = null

const accuracy = computed(() => {
  if (totalClicks.value === 0) return 100
  return Math.round((score.value / totalClicks.value) * 100)
})

const files = ['a', 'b', 'c', 'd', 'e', 'f', 'g', 'h']
const ranks = ['1', '2', '3', '4', '5', '6', '7', '8']

function pickRandomSquare(): string {
  const f = files[Math.floor(Math.random() * 8)]
  const r = ranks[Math.floor(Math.random() * 8)]
  const sq = `${f}${r}`
  if (sq === targetSquare.value) {
    return pickRandomSquare()
  }
  return sq
}

function startGame() {
  score.value = 0
  totalClicks.value = 0
  currentStreak.value = 0
  bestStreak.value = 0
  timeLeft.value = duration.value
  isRunning.value = true
  lastClickStatus.value = null
  lastClickedSquare.value = null
  targetSquare.value = pickRandomSquare()

  if (timerInterval) clearInterval(timerInterval)
  timerInterval = window.setInterval(() => {
    if (timeLeft.value > 0) {
      timeLeft.value--
    } else {
      endGame()
    }
  }, 1000)
}

function endGame() {
  isRunning.value = false
  if (timerInterval) {
    clearInterval(timerInterval)
    timerInterval = null
  }
  if (score.value > highScore.value) {
    highScore.value = score.value
    localStorage.setItem(HIGH_SCORE_KEY, score.value.toString())
  }
  playCaptureSound()
}

function onSquareClick(sq: string) {
  if (!isRunning.value) return

  totalClicks.value++
  lastClickedSquare.value = sq

  if (sq === targetSquare.value) {
    score.value++
    currentStreak.value++
    if (currentStreak.value > bestStreak.value) {
      bestStreak.value = currentStreak.value
    }
    lastClickStatus.value = 'correct'
    playMoveSound()
    targetSquare.value = pickRandomSquare()
  } else {
    currentStreak.value = 0
    lastClickStatus.value = 'wrong'
  }

  setTimeout(() => {
    lastClickStatus.value = null
  }, 350)
}

function resetGame() {
  if (timerInterval) {
    clearInterval(timerInterval)
    timerInterval = null
  }
  isRunning.value = false
  timeLeft.value = duration.value
  score.value = 0
  totalClicks.value = 0
  currentStreak.value = 0
  lastClickStatus.value = null
  lastClickedSquare.value = null
}

onUnmounted(() => {
  if (timerInterval) clearInterval(timerInterval)
})

onMounted(() => {
  targetSquare.value = pickRandomSquare()
})
</script>

<template>
  <div class="fixed inset-0 bg-black/75 backdrop-blur-xs flex items-center justify-center z-50 p-4">
    <div class="bg-zinc-900 border border-zinc-800 rounded-xl shadow-2xl w-full max-w-lg overflow-hidden flex flex-col animate-fade-in">
      <!-- Modal Header -->
      <div class="flex items-center justify-between px-5 py-4 border-b border-zinc-800 bg-zinc-950/60">
        <div class="flex items-center gap-2.5">
          <div class="w-8 h-8 rounded-lg bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center text-emerald-400">
            <Target class="w-4 h-4" />
          </div>
          <div>
            <h2 class="text-base font-semibold text-zinc-100">Board Vision & Coordinate Trainer</h2>
            <p class="text-xs text-zinc-400">Train your instant chess square recognition speed</p>
          </div>
        </div>
        <button
          class="text-zinc-400 hover:text-zinc-200 p-1.5 rounded-lg hover:bg-zinc-800 transition-colors"
          @click="emit('close')"
        >
          <X class="w-5 h-5" />
        </button>
      </div>

      <!-- Training Header Status Bar -->
      <div class="p-5 flex flex-col gap-4">
        <!-- Target Coordinate Banner -->
        <div class="bg-zinc-950 border border-zinc-800 rounded-xl p-4 flex items-center justify-between shadow-inner">
          <div class="flex flex-col">
            <span class="text-[11px] font-mono uppercase text-zinc-400">Click Square:</span>
            <span class="text-3xl font-mono font-black text-amber-400 tracking-wider">
              {{ isRunning ? targetSquare : '—' }}
            </span>
          </div>

          <div class="flex items-center gap-4">
            <!-- Timer Display -->
            <div class="flex flex-col items-center">
              <span class="text-[10px] font-mono text-zinc-400 uppercase">Time</span>
              <span
                class="text-xl font-mono font-bold"
                :class="timeLeft <= 5 && isRunning ? 'text-red-400 animate-pulse' : 'text-zinc-200'"
              >
                {{ timeLeft }}s
              </span>
            </div>

            <!-- Score Display -->
            <div class="flex flex-col items-center">
              <span class="text-[10px] font-mono text-zinc-400 uppercase">Score</span>
              <span class="text-xl font-mono font-bold text-emerald-400">{{ score }}</span>
            </div>

            <!-- Streak Display -->
            <div class="flex flex-col items-center">
              <span class="text-[10px] font-mono text-zinc-400 uppercase">Streak</span>
              <div class="flex items-center gap-1 text-amber-400 font-mono font-bold text-base">
                <Flame class="w-4 h-4 fill-amber-400" />
                <span>{{ currentStreak }}</span>
              </div>
            </div>
          </div>
        </div>

        <!-- 8x8 Clickable Training Board -->
        <div class="aspect-square w-full max-w-[340px] mx-auto grid grid-cols-8 grid-rows-8 rounded-lg overflow-hidden border border-zinc-700 shadow-lg select-none">
          <template v-for="r in (perspective === 'w' ? [7,6,5,4,3,2,1,0] : [0,1,2,3,4,5,6,7])" :key="`r-${r}`">
            <button
              v-for="f in (perspective === 'w' ? [0,1,2,3,4,5,6,7] : [7,6,5,4,3,2,1,0])"
              :key="`sq-${f}-${r}`"
              class="relative aspect-square flex items-center justify-center transition-all duration-75 cursor-pointer"
              :class="[
                (r + f) % 2 !== 0 ? 'bg-[#e3c896] hover:bg-[#d8bd8b]' : 'bg-[#ab7a4e] hover:bg-[#9d6f45]',
                lastClickedSquare === `${files[f]}${ranks[r]}` && lastClickStatus === 'correct' ? '!bg-emerald-500 ring-2 ring-emerald-300 z-10' : '',
                lastClickedSquare === `${files[f]}${ranks[r]}` && lastClickStatus === 'wrong' ? '!bg-red-500 ring-2 ring-red-300 z-10' : ''
              ]"
              :disabled="!isRunning"
              @click="onSquareClick(`${files[f]}${ranks[r]}`)"
            >
              <!-- Edge rank coordinate -->
              <span
                v-if="(perspective === 'w' && f === 0) || (perspective === 'b' && f === 7)"
                class="absolute top-0.5 left-1 text-[9px] font-bold opacity-60 pointer-events-none"
                :class="(r + f) % 2 !== 0 ? 'text-[#ab7a4e]' : 'text-[#e3c896]'"
              >
                {{ ranks[r] }}
              </span>
              <!-- Edge file coordinate -->
              <span
                v-if="(perspective === 'w' && r === 0) || (perspective === 'b' && r === 7)"
                class="absolute bottom-0.5 right-1 text-[9px] font-bold opacity-60 pointer-events-none"
                :class="(r + f) % 2 !== 0 ? 'text-[#ab7a4e]' : 'text-[#e3c896]'"
              >
                {{ files[f] }}
              </span>
            </button>
          </template>
        </div>

        <!-- Options row -->
        <div class="flex items-center justify-between text-xs text-zinc-400 bg-zinc-950 p-3 rounded-lg border border-zinc-800">
          <div class="flex items-center gap-2">
            <span>Side:</span>
            <button
              class="px-2.5 py-1 rounded font-medium transition-colors"
              :class="perspective === 'w' ? 'bg-zinc-800 text-white border border-zinc-700' : 'text-zinc-500 hover:text-zinc-300'"
              :disabled="isRunning"
              @click="perspective = 'w'"
            >
              White
            </button>
            <button
              class="px-2.5 py-1 rounded font-medium transition-colors"
              :class="perspective === 'b' ? 'bg-zinc-800 text-white border border-zinc-700' : 'text-zinc-500 hover:text-zinc-300'"
              :disabled="isRunning"
              @click="perspective = 'b'"
            >
              Black
            </button>
          </div>

          <div class="flex items-center gap-2">
            <span>Duration:</span>
            <select
              v-model.number="duration"
              class="bg-zinc-900 border border-zinc-700 rounded px-2 py-1 text-xs text-zinc-200 focus:outline-none"
              :disabled="isRunning"
              @change="timeLeft = duration"
            >
              <option :value="15">15s</option>
              <option :value="30">30s</option>
              <option :value="60">60s</option>
            </select>
          </div>

          <div class="flex items-center gap-1.5 text-zinc-300">
            <Trophy class="w-3.5 h-3.5 text-amber-400" />
            <span>Best: <strong class="text-amber-400">{{ highScore }}</strong></span>
          </div>
        </div>
      </div>

      <!-- Modal Footer -->
      <div class="flex items-center justify-between px-5 py-3.5 border-t border-zinc-800 bg-zinc-950/60">
        <div class="text-xs text-zinc-400">
          Accuracy: <strong class="text-zinc-200">{{ accuracy }}%</strong>
        </div>

        <div class="flex items-center gap-2.5">
          <button
            v-if="isRunning"
            class="flex items-center gap-1.5 px-3 py-1.5 bg-zinc-800 hover:bg-zinc-700 text-zinc-300 text-xs rounded-lg transition-colors border border-zinc-700"
            @click="resetGame"
          >
            <RotateCcw class="w-3.5 h-3.5" />
            <span>Reset</span>
          </button>

          <button
            v-if="!isRunning"
            class="flex items-center gap-1.5 px-4 py-2 bg-emerald-600 hover:bg-emerald-500 text-white text-xs font-semibold rounded-lg transition-colors shadow-sm"
            @click="startGame"
          >
            <Play class="w-4 h-4 fill-white" />
            <span>{{ totalClicks > 0 ? 'Play Again' : 'Start Trainer' }}</span>
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
