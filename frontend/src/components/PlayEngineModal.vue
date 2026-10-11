<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import Chessboard from './Chessboard.vue'
import { Swords, X, Play, RotateCcw, Flag, Award, Cpu, CheckCircle, AlertCircle } from '@lucide/vue'
import { EngineOption } from './EnginePanel.vue'
import { ParseFEN, MakeMove, GetBestMove, StartEngine } from '../../wailsjs/go/main/App'

const props = defineProps<{
  discoveredEngines: EngineOption[]
  availableEngines: string[]
  currentPieceSet: string
  boardTheme: {
    light: string
    dark: string
    highlight: string
  }
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'importGame', pgn: string): void
}>()

const startingFEN = 'rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1'

// Setup settings
const isPlaying = ref(false)
const selectedEngineName = ref('')
const playerColor = ref<'w' | 'b'>('w')
const timePerMove = ref(1000) // ms
const gameClockMinutes = ref(5)

// Match state
const fen = ref(startingFEN)
const legalMoves = ref<string[]>([])
const inCheck = ref(false)
const lastMove = ref<{ from: string; to: string } | null>(null)
const engineThinking = ref(false)
const gameOverMessage = ref<string | null>(null)
const gameResult = ref<'*' | '1-0' | '0-1' | '1/2-1/2'>('*')

// Move history
const moveHistory = ref<{ ply: number; san: string; uci: string }[]>([])

// Clocks (seconds remaining)
const playerTime = ref(300)
const engineTime = ref(300)
let timerInterval: number | null = null

const isPlayerTurn = computed(() => {
  const parts = fen.value.split(' ')
  const turn = parts[1] || 'w'
  return turn === playerColor.value
})

const isBoardFlipped = computed(() => {
  return playerColor.value === 'b'
})

onMounted(() => {
  // Pick default engine
  const avail = props.discoveredEngines.find(e => e.available)
  if (avail) {
    selectedEngineName.value = avail.name || avail.cmd
  } else if (props.availableEngines.length > 0) {
    selectedEngineName.value = props.availableEngines[0]
  }
})

onUnmounted(() => {
  stopClock()
})

function startClock() {
  stopClock()
  timerInterval = window.setInterval(() => {
    if (gameOverMessage.value) {
      stopClock()
      return
    }
    if (isPlayerTurn.value) {
      if (playerTime.value > 0) {
        playerTime.value--
      } else {
        gameOverMessage.value = 'Game Over: You lost on time!'
        gameResult.value = playerColor.value === 'w' ? '0-1' : '1-0'
        stopClock()
      }
    } else {
      if (engineTime.value > 0) {
        engineTime.value--
      } else {
        gameOverMessage.value = 'Game Over: Engine lost on time!'
        gameResult.value = playerColor.value === 'w' ? '1-0' : '0-1'
        stopClock()
      }
    }
  }, 1000)
}

function stopClock() {
  if (timerInterval) {
    clearInterval(timerInterval)
    timerInterval = null
  }
}

function formatTime(seconds: number): string {
  const m = Math.floor(seconds / 60)
  const s = seconds % 60
  return `${m}:${s < 10 ? '0' : ''}${s}`
}

async function startMatch() {
  try {
    if (selectedEngineName.value) {
      await StartEngine(selectedEngineName.value).catch(console.error)
    }

    fen.value = startingFEN
    lastMove.value = null
    inCheck.value = false
    moveHistory.value = []
    gameOverMessage.value = null
    gameResult.value = '*'
    playerTime.value = gameClockMinutes.value * 60
    engineTime.value = gameClockMinutes.value * 60
    isPlaying.value = true

    const state = await ParseFEN(startingFEN)
    legalMoves.value = state.legalMoves

    startClock()

    // If player is Black, engine makes the first move!
    if (playerColor.value === 'b') {
      triggerEngineMove()
    }
  } catch (err) {
    alert(`Failed to start match: ${err}`)
  }
}

async function handleMove(uci: string) {
  if (!isPlaying.value || !isPlayerTurn.value || engineThinking.value || gameOverMessage.value) {
    return
  }

  try {
    const nextState = await MakeMove(fen.value, uci)
    const from = uci.slice(0, 2)
    const to = uci.slice(2, 4)
    lastMove.value = { from, to }
    fen.value = nextState.fen
    legalMoves.value = nextState.legalMoves
    inCheck.value = nextState.inCheck

    moveHistory.value.push({
      ply: moveHistory.value.length + 1,
      san: uci,
      uci,
    })

    if (nextState.isCheckmate) {
      gameOverMessage.value = 'Game Over: You won by checkmate!'
      gameResult.value = playerColor.value === 'w' ? '1-0' : '0-1'
      stopClock()
      return
    }

    if (nextState.isStalemate) {
      gameOverMessage.value = 'Game Over: Draw by stalemate.'
      gameResult.value = '1/2-1/2'
      stopClock()
      return
    }

    // Trigger engine reply
    triggerEngineMove()
  } catch (err) {
    console.error('Move error:', err)
  }
}

async function triggerEngineMove() {
  if (gameOverMessage.value) return
  engineThinking.value = true

  try {
    const best = await GetBestMove(fen.value, [], timePerMove.value)
    if (!best) {
      engineThinking.value = false
      return
    }

    const nextState = await MakeMove(fen.value, best)
    const from = best.slice(0, 2)
    const to = best.slice(2, 4)
    lastMove.value = { from, to }
    fen.value = nextState.fen
    legalMoves.value = nextState.legalMoves
    inCheck.value = nextState.inCheck

    moveHistory.value.push({
      ply: moveHistory.value.length + 1,
      san: best,
      uci: best,
    })

    if (nextState.isCheckmate) {
      gameOverMessage.value = 'Game Over: Engine won by checkmate.'
      gameResult.value = playerColor.value === 'w' ? '0-1' : '1-0'
      stopClock()
    } else if (nextState.isStalemate) {
      gameOverMessage.value = 'Game Over: Draw by stalemate.'
      gameResult.value = '1/2-1/2'
      stopClock()
    }
  } catch (err) {
    console.error('Engine move error:', err)
  } finally {
    engineThinking.value = false
  }
}

function resignMatch() {
  if (!confirm('Are you sure you want to resign?')) return
  gameOverMessage.value = 'Game Over: You resigned.'
  gameResult.value = playerColor.value === 'w' ? '0-1' : '1-0'
  stopClock()
}

function offerDraw() {
  if (confirm('Engine accepts the draw.')) {
    gameOverMessage.value = 'Game Over: Drawn by agreement.'
    gameResult.value = '1/2-1/2'
    stopClock()
  }
}

function exportMatchToApp() {
  // Generate simple PGN
  let pgn = `[Event "Play vs Engine"]\n`
  pgn += `[Site "NewScid"]\n`
  pgn += `[Date "${new Date().toISOString().split('T')[0].replace(/-/g, '.')}"]\n`
  pgn += `[White "${playerColor.value === 'w' ? 'Human' : selectedEngineName.value}"]\n`
  pgn += `[Black "${playerColor.value === 'b' ? 'Human' : selectedEngineName.value}"]\n`
  pgn += `[Result "${gameResult.value}"]\n\n`

  for (let i = 0; i < moveHistory.value.length; i++) {
    if (i % 2 === 0) {
      pgn += `${Math.floor(i / 2) + 1}. `
    }
    pgn += `${moveHistory.value[i].san} `
  }
  pgn += `${gameResult.value}\n`

  emit('importGame', pgn)
  emit('close')
}
</script>

<template>
  <div class="fixed inset-0 bg-black/75 backdrop-blur-xs flex items-center justify-center z-50 p-4 animate-fade-in select-none">
    <div class="bg-zinc-900 border border-zinc-700/80 rounded-2xl w-full max-w-4xl shadow-2xl overflow-hidden flex flex-col max-h-[92vh]">
      <!-- Header -->
      <div class="flex items-center justify-between px-5 py-3.5 bg-zinc-950 border-b border-zinc-800">
        <div class="flex items-center gap-2">
          <Swords class="w-5 h-5 text-amber-400" />
          <h2 class="text-sm font-bold text-zinc-100">Play Against Engine</h2>
          <span class="text-[11px] text-zinc-500 font-mono">Interactive UCI Match</span>
        </div>
        <button
          class="text-zinc-400 hover:text-white p-1 rounded-lg hover:bg-zinc-800 transition-colors"
          @click="emit('close')"
        >
          <X class="w-5 h-5" />
        </button>
      </div>

      <!-- Main Body -->
      <div class="p-5 flex-1 overflow-y-auto flex flex-col md:flex-row gap-6">
        <!-- Left: Chessboard -->
        <div class="flex flex-col items-center gap-3 shrink-0 w-80 sm:w-96 mx-auto md:mx-0">
          <!-- Opponent Bar -->
          <div class="w-full flex items-center justify-between bg-zinc-950 px-4 py-2 rounded-xl border border-zinc-800 text-xs">
            <div class="flex items-center gap-2">
              <Cpu class="w-4 h-4 text-emerald-400" />
              <span class="font-bold text-zinc-200">{{ selectedEngineName || 'Engine' }}</span>
              <span v-if="engineThinking" class="text-[10px] text-amber-400 animate-pulse font-medium">Thinking...</span>
            </div>
            <div class="font-mono text-sm font-bold" :class="!isPlayerTurn && isPlaying ? 'text-amber-400' : 'text-zinc-400'">
              {{ formatTime(engineTime) }}
            </div>
          </div>

          <!-- Board -->
          <div class="w-80 h-80 sm:w-96 sm:h-96 shrink-0 relative flex items-center justify-center">
            <Chessboard
              :fen="fen"
              :legal-moves="legalMoves"
              :piece-set="currentPieceSet"
              :board-theme="boardTheme"
              :flipped="isBoardFlipped"
              :last-move="lastMove"
              :in-check="inCheck"
              @move="handleMove"
            />
          </div>

          <!-- Player Bar -->
          <div class="w-full flex items-center justify-between bg-zinc-950 px-4 py-2 rounded-xl border border-zinc-800 text-xs">
            <div class="flex items-center gap-2">
              <span class="w-2.5 h-2.5 rounded-full" :class="playerColor === 'w' ? 'bg-white' : 'bg-zinc-900 border border-zinc-600'" />
              <span class="font-bold text-zinc-200">You ({{ playerColor === 'w' ? 'White' : 'Black' }})</span>
              <span v-if="isPlayerTurn && isPlaying && !gameOverMessage" class="text-[10px] text-emerald-400 font-medium">Your Turn</span>
            </div>
            <div class="font-mono text-sm font-bold" :class="isPlayerTurn && isPlaying ? 'text-emerald-400' : 'text-zinc-400'">
              {{ formatTime(playerTime) }}
            </div>
          </div>
        </div>

        <!-- Right: Settings & Match Control -->
        <div class="flex-1 min-w-0 flex flex-col gap-4">
          <!-- Game Over Alert -->
          <div v-if="gameOverMessage" class="bg-emerald-950/40 border border-emerald-500/50 p-3 rounded-xl flex items-center gap-3">
            <Award class="w-6 h-6 text-amber-400 shrink-0" />
            <div>
              <p class="text-xs font-bold text-zinc-100">{{ gameOverMessage }}</p>
              <p class="text-[11px] text-emerald-300">Result: {{ gameResult }}</p>
            </div>
            <button
              class="ml-auto text-xs px-3 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white font-semibold cursor-pointer shrink-0"
              @click="exportMatchToApp"
            >
              Load in Main View
            </button>
          </div>

          <!-- Configuration panel (when match is not active) -->
          <div v-if="!isPlaying" class="flex flex-col gap-3">
            <!-- Engine Selection -->
            <div class="bg-zinc-950 p-3 rounded-xl border border-zinc-800 flex flex-col gap-1.5">
              <label class="text-xs font-semibold text-zinc-400">Select UCI Engine</label>
              <select
                v-model="selectedEngineName"
                class="bg-zinc-900 border border-zinc-700 rounded-lg p-2 text-xs text-zinc-100 focus:outline-hidden focus:border-emerald-500"
              >
                <option
                  v-for="eng in discoveredEngines.filter(e => e.available)"
                  :key="eng.cmd"
                  :value="eng.name || eng.cmd"
                >
                  {{ eng.name || eng.cmd }}
                </option>
              </select>
            </div>

            <!-- Color Selection -->
            <div class="bg-zinc-950 p-3 rounded-xl border border-zinc-800 flex flex-col gap-1.5">
              <label class="text-xs font-semibold text-zinc-400">Your Color</label>
              <div class="grid grid-cols-2 gap-2">
                <button
                  class="py-2 rounded-lg text-xs font-semibold border transition-all cursor-pointer flex items-center justify-center gap-2"
                  :class="playerColor === 'w' ? 'bg-white text-zinc-950 border-white' : 'bg-zinc-900 text-zinc-400 border-zinc-800'"
                  @click="playerColor = 'w'"
                >
                  <span class="w-3 h-3 rounded-full bg-zinc-200 border border-zinc-400" />
                  <span>Play White</span>
                </button>
                <button
                  class="py-2 rounded-lg text-xs font-semibold border transition-all cursor-pointer flex items-center justify-center gap-2"
                  :class="playerColor === 'b' ? 'bg-zinc-800 text-white border-zinc-600' : 'bg-zinc-900 text-zinc-400 border-zinc-800'"
                  @click="playerColor = 'b'"
                >
                  <span class="w-3 h-3 rounded-full bg-zinc-900 border border-zinc-600" />
                  <span>Play Black</span>
                </button>
              </div>
            </div>

            <!-- Thinking Time / Difficulty -->
            <div class="bg-zinc-950 p-3 rounded-xl border border-zinc-800 flex flex-col gap-1.5">
              <label class="text-xs font-semibold text-zinc-400">Engine Thinking Time</label>
              <div class="grid grid-cols-4 gap-1.5 text-xs">
                <button
                  v-for="opt in [
                    { label: 'Casual', ms: 500 },
                    { label: 'Blitz', ms: 1000 },
                    { label: 'Rapid', ms: 2000 },
                    { label: 'Deep', ms: 3000 }
                  ]"
                  :key="opt.ms"
                  class="py-1.5 rounded-lg border text-center transition-all cursor-pointer font-medium"
                  :class="timePerMove === opt.ms ? 'bg-emerald-600 border-emerald-500 text-white' : 'bg-zinc-900 border-zinc-800 text-zinc-400 hover:bg-zinc-800'"
                  @click="timePerMove = opt.ms"
                >
                  {{ opt.label }}
                </button>
              </div>
            </div>

            <!-- Time Control -->
            <div class="bg-zinc-950 p-3 rounded-xl border border-zinc-800 flex flex-col gap-1.5">
              <label class="text-xs font-semibold text-zinc-400">Clock Time (Minutes)</label>
              <div class="flex gap-2">
                <button
                  v-for="min in [3, 5, 10, 15]"
                  :key="min"
                  class="flex-1 py-1.5 rounded-lg border text-xs font-medium transition-all cursor-pointer"
                  :class="gameClockMinutes === min ? 'bg-emerald-600 border-emerald-500 text-white' : 'bg-zinc-900 border-zinc-800 text-zinc-400 hover:bg-zinc-800'"
                  @click="gameClockMinutes = min"
                >
                  {{ min }}m
                </button>
              </div>
            </div>

            <!-- Start Match Button -->
            <button
              class="w-full py-2.5 rounded-xl bg-amber-500 hover:bg-amber-400 text-zinc-950 font-bold text-xs flex items-center justify-center gap-2 shadow-lg shadow-amber-500/20 transition-all cursor-pointer mt-2"
              @click="startMatch"
            >
              <Play class="w-4 h-4 fill-current" />
              <span>Start Match</span>
            </button>
          </div>

          <!-- In-Game Controls & Moves -->
          <div v-else class="flex-1 flex flex-col gap-3 min-h-0">
            <!-- Active Controls -->
            <div class="flex gap-2">
              <button
                class="flex-1 py-1.5 rounded-lg bg-zinc-800 hover:bg-zinc-700 text-zinc-300 text-xs font-medium flex items-center justify-center gap-1.5 cursor-pointer"
                @click="offerDraw"
              >
                <Award class="w-3.5 h-3.5 text-zinc-400" />
                <span>Offer Draw</span>
              </button>
              <button
                class="flex-1 py-1.5 rounded-lg bg-zinc-800 hover:bg-rose-900/60 text-rose-300 text-xs font-medium flex items-center justify-center gap-1.5 cursor-pointer"
                @click="resignMatch"
              >
                <Flag class="w-3.5 h-3.5 text-rose-400" />
                <span>Resign</span>
              </button>
              <button
                class="flex-1 py-1.5 rounded-lg bg-zinc-800 hover:bg-zinc-700 text-zinc-300 text-xs font-medium flex items-center justify-center gap-1.5 cursor-pointer"
                @click="isPlaying = false; stopClock()"
              >
                <RotateCcw class="w-3.5 h-3.5 text-emerald-400" />
                <span>Rematch / Setup</span>
              </button>
            </div>

            <!-- Move History -->
            <div class="bg-zinc-950 p-3 rounded-xl border border-zinc-800 flex-1 overflow-y-auto min-h-40 max-h-60 flex flex-col gap-1 font-mono text-xs">
              <span class="text-[10px] font-semibold text-zinc-500 uppercase">Match Moves</span>
              <div class="grid grid-cols-2 gap-x-4 gap-y-1 mt-1 text-zinc-300">
                <template v-for="(m, idx) in moveHistory" :key="m.ply">
                  <span v-if="idx % 2 === 0" class="text-zinc-500">
                    {{ Math.floor(idx / 2) + 1 }}. <span class="text-zinc-200">{{ m.san }}</span>
                  </span>
                  <span v-else class="text-zinc-200">
                    {{ m.san }}
                  </span>
                </template>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
