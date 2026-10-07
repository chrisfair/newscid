<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import Chessboard from './Chessboard.vue'
import {
  BookOpen,
  X,
  Play,
  CheckCircle,
  AlertCircle,
  RotateCcw,
  Plus,
  Trash2,
  Edit3,
  Layers,
  Sparkles,
  Flame,
  Award,
  Lightbulb
} from '@lucide/vue'
import {
  GetRepertoires,
  GetRepertoire,
  SaveRepertoire,
  DeleteRepertoire,
  VerifyRepertoireMove,
  RecordRepertoirePractice,
  AddRepertoireMove,
  ParseFEN,
  MakeMove
} from '../../wailsjs/go/main/App'
import { repertoire, main } from '../../wailsjs/go/models'

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

const startingFEN = 'rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1'

// Repertoire list & selection
const repertoires = ref<repertoire.RepertoireSummary[]>([])
const selectedRepId = ref<string>('')
const activeRep = ref<repertoire.Repertoire | null>(null)

// Mode: 'drill' (Test & Spaced Practice) or 'builder' (Create & Add Lines)
const mode = ref<'drill' | 'builder'>('drill')

// Board State
const currentFEN = ref(startingFEN)
const legalMoves = ref<string[]>([])
const lastMove = ref<{ from: string; to: string } | null>(null)
const inCheck = ref(false)
const moveHistory = ref<{ ply: number; san: string; uci: string }[]>([])

// Drill State & Gamification
const drillScore = ref(0)
const drillStreak = ref(0)
const drillTotalMoves = ref(0)
const drillCorrectMoves = ref(0)
const feedback = ref<{ type: 'success' | 'error' | 'hint' | null; message: string }>({
  type: null,
  message: '',
})
const expectedAlternatives = ref<repertoire.RepertoireMove[]>([])

// Builder State
const noteInput = ref('')
const newRepModal = ref(false)
const newRepName = ref('')
const newRepColor = ref<'w' | 'b'>('w')
const newRepDesc = ref('')

const isBoardFlipped = computed(() => {
  return activeRep.value?.color === 'b'
})

const accuracy = computed(() => {
  if (drillTotalMoves.value === 0) return 100
  return Math.round((drillCorrectMoves.value / drillTotalMoves.value) * 100)
})

const currentPositionCandidates = computed(() => {
  if (!activeRep.value || !activeRep.value.positions) return []
  const core = coreFEN(currentFEN.value)
  return activeRep.value.positions[core] || []
})

function coreFEN(fen: string): string {
  const parts = fen.split(' ')
  if (parts.length >= 4) {
    return `${parts[0]} ${parts[1]} ${parts[2]} ${parts[3]}`
  }
  return fen
}

onMounted(async () => {
  await loadRepertoires()
  if (repertoires.value.length > 0) {
    await selectRepertoire(repertoires.value[0].id)
  }
})

async function loadRepertoires() {
  try {
    const list = await GetRepertoires()
    repertoires.value = list
  } catch (err) {
    console.error('Error loading repertoires:', err)
  }
}

async function selectRepertoire(id: string) {
  try {
    selectedRepId.value = id
    const rep = await GetRepertoire(id)
    activeRep.value = rep
    resetBoard()
  } catch (err) {
    console.error('Error selecting repertoire:', err)
  }
}

async function resetBoard() {
  currentFEN.value = startingFEN
  lastMove.value = null
  moveHistory.value = []
  feedback.value = { type: null, message: '' }
  expectedAlternatives.value = []

  const state = await ParseFEN(startingFEN)
  legalMoves.value = state.legalMoves
  inCheck.value = state.inCheck

  // If Black repertoire in drill mode, White plays first move!
  if (mode.value === 'drill' && activeRep.value?.color === 'b') {
    handleOpponentMove()
  }
}

async function handleMove(uci: string) {
  try {
    if (mode.value === 'drill') {
      await handleDrillMove(uci)
    } else {
      await handleBuilderMove(uci)
    }
  } catch (err) {
    console.error('Move handling error:', err)
  }
}

// ------------------------------------
// DRILL MODE (Spaced Repetition Test)
// ------------------------------------
async function handleDrillMove(uci: string) {
  if (!activeRep.value) return

  const fenBefore = currentFEN.value
  drillTotalMoves.value++

  // Check against repertoire
  const check = await VerifyRepertoireMove(activeRep.value.id, fenBefore, uci)
  expectedAlternatives.value = check.alternatives || []

  if (check.isRepertoire && check.playedMove) {
    // Correct repertoire move!
    drillCorrectMoves.value++
    drillScore.value += 10
    drillStreak.value++
    RecordRepertoirePractice(activeRep.value.id, fenBefore, uci, true)

    const nextState = await MakeMove(fenBefore, uci)
    const from = uci.slice(0, 2)
    const to = uci.slice(2, 4)
    lastMove.value = { from, to }
    currentFEN.value = nextState.fen
    legalMoves.value = nextState.legalMoves
    inCheck.value = nextState.inCheck
    moveHistory.value.push({ ply: moveHistory.value.length + 1, san: check.playedMove.san || uci, uci })

    feedback.value = {
      type: 'success',
      message: check.playedMove.comment
        ? `Excellent! ${check.playedMove.san} (${check.playedMove.comment})`
        : `Correct repertoire move: ${check.playedMove.san}!`,
    }

    // Opponent reply from repertoire
    setTimeout(() => {
      handleOpponentMove()
    }, 450)
  } else {
    // Incorrect move / not in repertoire
    drillStreak.value = 0
    RecordRepertoirePractice(activeRep.value.id, fenBefore, uci, false)

    let altText = ''
    if (check.alternatives && check.alternatives.length > 0) {
      altText = ` Expected: ${check.alternatives.map(a => a.san || a.uci).join(' or ')}`
    }
    feedback.value = {
      type: 'error',
      message: `Move not in repertoire!${altText}`,
    }
  }
}

async function handleOpponentMove() {
  if (!activeRep.value) return
  const fenBefore = currentFEN.value
  const core = coreFEN(fenBefore)

  // Find if there are moves in repertoire from this opponent position
  const options = activeRep.value.positions[core]
  if (options && options.length > 0) {
    // Pick an opponent candidate move
    const chosen = options[Math.floor(Math.random() * options.length)]
    const nextState = await MakeMove(fenBefore, chosen.uci)
    const from = chosen.uci.slice(0, 2)
    const to = chosen.uci.slice(2, 4)
    lastMove.value = { from, to }
    currentFEN.value = nextState.fen
    legalMoves.value = nextState.legalMoves
    inCheck.value = nextState.inCheck
    moveHistory.value.push({ ply: moveHistory.value.length + 1, san: chosen.san || chosen.uci, uci: chosen.uci })
  } else {
    // End of repertoire branch
    feedback.value = {
      type: 'success',
      message: 'Repertoire line completed! Great work mastering this variation.',
    }
  }
}


// ------------------------------------
// BUILDER MODE (Add & Curate Lines)
// ------------------------------------
async function handleBuilderMove(uci: string) {
  const fenBefore = currentFEN.value
  const nextState = await MakeMove(fenBefore, uci)
  const from = uci.slice(0, 2)
  const to = uci.slice(2, 4)
  lastMove.value = { from, to }
  currentFEN.value = nextState.fen
  legalMoves.value = nextState.legalMoves
  inCheck.value = nextState.inCheck
  moveHistory.value.push({ ply: moveHistory.value.length + 1, san: uci, uci })

  // Auto-record to repertoire if played by repertoire side
  if (activeRep.value) {
    const isWhiteTurn = fenBefore.split(' ')[1] === 'w'
    const isRepTurn = (activeRep.value.color === 'w' && isWhiteTurn) || (activeRep.value.color === 'b' && !isWhiteTurn)
    if (isRepTurn) {
      await AddRepertoireMove(activeRep.value.id, fenBefore, uci, uci, nextState.fen, noteInput.value)
      noteInput.value = ''
      await selectRepertoire(activeRep.value.id)
    }
  }
}

async function addManualMove(moveSAN: string) {
  if (!activeRep.value || !moveSAN.trim()) return
  try {
    const parsedState = await MakeMove(currentFEN.value, moveSAN)
    await AddRepertoireMove(activeRep.value.id, currentFEN.value, moveSAN, moveSAN, parsedState.fen, noteInput.value)
    noteInput.value = ''
    await selectRepertoire(activeRep.value.id)
    await resetBoard()
  } catch (err) {
    alert(`Invalid move: ${err}`)
  }
}

async function createNewRepertoire() {
  if (!newRepName.value.trim()) return
  try {
    const rep: any = {
      id: `${newRepColor.value}-${newRepName.value.toLowerCase().replace(/\s+/g, '-')}`,
      name: newRepName.value,
      color: newRepColor.value,
      description: newRepDesc.value,
      positions: {},
    }
    await SaveRepertoire(rep)
    await loadRepertoires()
    await selectRepertoire(rep.id)
    newRepModal.value = false
    newRepName.value = ''
    newRepDesc.value = ''
  } catch (err) {
    alert(`Failed to create repertoire: ${err}`)
  }
}

async function deleteCurrentRepertoire() {
  if (!activeRep.value) return
  if (!confirm(`Are you sure you want to delete repertoire "${activeRep.value.name}"?`)) return
  try {
    await DeleteRepertoire(activeRep.value.id)
    await loadRepertoires()
    if (repertoires.value.length > 0) {
      await selectRepertoire(repertoires.value[0].id)
    } else {
      activeRep.value = null
    }
  } catch (err) {
    alert(`Failed to delete repertoire: ${err}`)
  }
}

function showHint() {
  if (expectedAlternatives.value.length > 0) {
    const hintMoves = expectedAlternatives.value.map(a => a.san || a.uci).join(', ')
    feedback.value = {
      type: 'hint',
      message: `Hint: Candidate repertoire moves: ${hintMoves}`,
    }
  } else {
    feedback.value = {
      type: 'hint',
      message: 'No repertoire moves recorded from this position yet.',
    }
  }
}


function loadIntoMainApp() {
  emit('loadIntoApp', currentFEN.value)
  emit('close')
}

watch(mode, () => {
  resetBoard()
})
</script>

<template>
  <div class="fixed inset-0 bg-black/80 backdrop-blur-xs flex items-center justify-center z-50 p-4 animate-fade-in select-none">
    <div class="bg-zinc-900 border border-zinc-700/80 rounded-2xl w-full max-w-5xl shadow-2xl overflow-hidden flex flex-col max-h-[92vh]">
      <!-- Header -->
      <div class="flex items-center justify-between px-5 py-3.5 bg-zinc-950 border-b border-zinc-800">
        <div class="flex items-center gap-3">
          <BookOpen class="w-5 h-5 text-emerald-400" />
          <div>
            <h2 class="text-sm font-bold text-zinc-100">Opening Repertoire Builder & Trainer</h2>
            <p class="text-[11px] text-zinc-500 font-mono">Curate lines, drill spaced repetition, and master your opening tree</p>
          </div>
        </div>

        <!-- Mode Toggle & Close -->
        <div class="flex items-center gap-2">
          <div class="bg-zinc-900 border border-zinc-700 rounded-lg p-0.5 flex text-xs">
            <button
              class="px-3 py-1 rounded-md font-semibold transition-all cursor-pointer flex items-center gap-1.5"
              :class="mode === 'drill' ? 'bg-emerald-600 text-white shadow-sm' : 'text-zinc-400 hover:text-zinc-200'"
              @click="mode = 'drill'"
            >
              <Play class="w-3.5 h-3.5 fill-current" />
              <span>Drill / Practice</span>
            </button>
            <button
              class="px-3 py-1 rounded-md font-semibold transition-all cursor-pointer flex items-center gap-1.5"
              :class="mode === 'builder' ? 'bg-emerald-600 text-white shadow-sm' : 'text-zinc-400 hover:text-zinc-200'"
              @click="mode = 'builder'"
            >
              <Edit3 class="w-3.5 h-3.5" />
              <span>Repertoire Builder</span>
            </button>
          </div>

          <button
            class="text-zinc-400 hover:text-white p-1 rounded-lg hover:bg-zinc-800 transition-colors ml-2 cursor-pointer"
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
          <!-- Turn & Repertoire Color Banner -->
          <div class="w-full flex items-center justify-between bg-zinc-950 px-4 py-2 rounded-xl border border-zinc-800 text-xs">
            <div class="flex items-center gap-2">
              <span class="w-2.5 h-2.5 rounded-full" :class="activeRep?.color === 'w' ? 'bg-white' : 'bg-zinc-900 border border-zinc-600'" />
              <span class="font-bold text-zinc-200">{{ activeRep?.name || 'Select Repertoire' }}</span>
            </div>
            <span class="text-[11px] font-mono text-zinc-400">
              {{ activeRep?.color === 'w' ? 'Playing White' : 'Playing Black' }}
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
              'bg-amber-950/60 border-amber-500/50 text-amber-300': feedback.type === 'hint',
            }"
          >
            <CheckCircle v-if="feedback.type === 'success'" class="w-4 h-4 shrink-0 text-emerald-400" />
            <AlertCircle v-else-if="feedback.type === 'error'" class="w-4 h-4 shrink-0 text-rose-400" />
            <Lightbulb v-else class="w-4 h-4 shrink-0 text-amber-400" />
            <span>{{ feedback.message }}</span>
          </div>
        </div>

        <!-- Right: Repertoire Selector & Controls -->
        <div class="flex-1 flex flex-col gap-4">
          <!-- Repertoire Selector Dropdown + New Button -->
          <div class="bg-zinc-950 p-3 rounded-xl border border-zinc-800 flex flex-col gap-2">
            <div class="flex items-center justify-between">
              <label class="text-xs font-semibold text-zinc-400">Active Opening Repertoire</label>
              <button
                class="flex items-center gap-1 text-[11px] text-emerald-400 hover:text-emerald-300 font-semibold cursor-pointer"
                @click="newRepModal = true"
              >
                <Plus class="w-3.5 h-3.5" />
                <span>New Repertoire</span>
              </button>
            </div>

            <select
              :value="selectedRepId"
              class="w-full bg-zinc-900 border border-zinc-700 rounded-lg p-2 text-xs text-zinc-100 focus:outline-hidden focus:border-emerald-500"
              @change="selectRepertoire(($event.target as HTMLSelectElement).value)"
            >
              <option v-for="r in repertoires" :key="r.id" :value="r.id">
                [{{ r.color.toUpperCase() }}] {{ r.name }} ({{ r.numMoves }} moves)
              </option>
            </select>
          </div>

          <!-- Drill Mode Panel -->
          <div v-if="mode === 'drill'" class="flex-1 flex flex-col gap-3">
            <!-- Drill Stats -->
            <div class="grid grid-cols-3 gap-2">
              <div class="bg-zinc-950 p-3 rounded-xl border border-zinc-800 flex flex-col items-center">
                <span class="text-[10px] uppercase font-semibold text-zinc-500">Streak</span>
                <span class="text-lg font-bold text-amber-400 flex items-center gap-1">
                  <Flame class="w-4 h-4" /> {{ drillStreak }}
                </span>
              </div>
              <div class="bg-zinc-950 p-3 rounded-xl border border-zinc-800 flex flex-col items-center">
                <span class="text-[10px] uppercase font-semibold text-zinc-500">Accuracy</span>
                <span class="text-lg font-bold text-emerald-400">{{ accuracy }}%</span>
              </div>
              <div class="bg-zinc-950 p-3 rounded-xl border border-zinc-800 flex flex-col items-center">
                <span class="text-[10px] uppercase font-semibold text-zinc-500">Score</span>
                <span class="text-lg font-bold text-zinc-100 flex items-center gap-1">
                  <Award class="w-4 h-4 text-emerald-400" /> {{ drillScore }}
                </span>
              </div>
            </div>

            <!-- Drill Action Buttons -->
            <div class="flex gap-2">
              <button
                class="flex-1 py-2 rounded-xl bg-zinc-800 hover:bg-zinc-700 text-amber-300 text-xs font-semibold flex items-center justify-center gap-1.5 cursor-pointer"
                @click="showHint"
              >
                <Lightbulb class="w-4 h-4" />
                <span>Show Repertoire Hint</span>
              </button>
              <button
                class="flex-1 py-2 rounded-xl bg-zinc-800 hover:bg-zinc-700 text-zinc-300 text-xs font-semibold flex items-center justify-center gap-1.5 cursor-pointer"
                @click="resetBoard"
              >
                <RotateCcw class="w-4 h-4 text-emerald-400" />
                <span>Restart Drill</span>
              </button>
            </div>

            <!-- Played Moves List in this Drill -->
            <div class="bg-zinc-950 p-3 rounded-xl border border-zinc-800 flex-1 overflow-y-auto min-h-36 max-h-56 font-mono text-xs flex flex-col gap-1">
              <span class="text-[10px] font-semibold text-zinc-500 uppercase">Played Line</span>
              <div class="grid grid-cols-2 gap-x-4 gap-y-1 mt-1 text-zinc-300">
                <template v-for="(m, idx) in moveHistory" :key="m.ply">
                  <span v-if="idx % 2 === 0" class="text-zinc-500">
                    {{ Math.floor(idx / 2) + 1 }}. <span class="text-zinc-200 font-semibold">{{ m.san }}</span>
                  </span>
                  <span v-else class="text-zinc-200 font-semibold">
                    {{ m.san }}
                  </span>
                </template>
              </div>
            </div>
          </div>

          <!-- Builder Mode Panel -->
          <div v-else class="flex-1 flex flex-col gap-3">
            <!-- Position Candidate Moves in Repertoire -->
            <div class="bg-zinc-950 p-3 rounded-xl border border-zinc-800 flex flex-col gap-2">
              <span class="text-xs font-semibold text-zinc-400">Repertoire Moves for Current Position</span>
              <div v-if="currentPositionCandidates.length > 0" class="space-y-1.5">
                <div
                  v-for="c in currentPositionCandidates"
                  :key="c.uci"
                  class="flex items-center justify-between px-3 py-1.5 rounded-lg bg-zinc-900 border border-zinc-800 text-xs"
                >
                  <div class="flex items-center gap-2">
                    <span class="font-bold text-emerald-400 font-mono">{{ c.san || c.uci }}</span>
                    <span v-if="c.comment" class="text-[11px] text-zinc-400">({{ c.comment }})</span>
                  </div>
                  <span class="text-[10px] text-zinc-500 font-mono">
                    Recall: {{ c.practice > 0 ? Math.round((c.correct / c.practice) * 100) : 0 }}% ({{ c.practice }} drills)
                  </span>
                </div>
              </div>
              <div v-else class="text-xs text-zinc-500 italic p-1">
                No repertoire moves recorded from this position. Make a move on the board to record it.
              </div>
            </div>

            <!-- Add Note / Comment -->
            <div class="bg-zinc-950 p-3 rounded-xl border border-zinc-800 flex flex-col gap-2">
              <label class="text-xs font-semibold text-zinc-400">Add Strategic Note to Next Move</label>
              <input
                v-model="noteInput"
                type="text"
                placeholder="e.g. Pin the knight with Bg5, or fight for d5 break..."
                class="w-full bg-zinc-900 border border-zinc-700 rounded-lg p-2 text-xs text-zinc-100 placeholder-zinc-500 focus:outline-hidden focus:border-emerald-500"
              />
            </div>

            <!-- Builder Action Buttons -->
            <div class="flex gap-2">
              <button
                class="flex-1 py-2 rounded-xl bg-zinc-800 hover:bg-zinc-700 text-zinc-300 text-xs font-semibold flex items-center justify-center gap-1.5 cursor-pointer"
                @click="resetBoard"
              >
                <RotateCcw class="w-4 h-4 text-emerald-400" />
                <span>Reset to Start</span>
              </button>
              <button
                class="py-2 px-3 rounded-xl bg-zinc-800 hover:bg-rose-900/60 text-rose-400 text-xs font-semibold flex items-center gap-1.5 cursor-pointer"
                @click="deleteCurrentRepertoire"
              >
                <Trash2 class="w-4 h-4" />
                <span>Delete Repertoire</span>
              </button>
            </div>
          </div>

          <!-- Bottom: Load into main workspace -->
          <button
            class="w-full py-2 rounded-xl bg-zinc-950 hover:bg-zinc-800 border border-zinc-800 text-zinc-300 text-xs font-medium flex items-center justify-center gap-2 cursor-pointer transition-colors mt-auto"
            @click="loadIntoMainApp"
          >
            <span>Analyze Position in Main App</span>
          </button>
        </div>
      </div>
    </div>

    <!-- Create New Repertoire Modal -->
    <div
      v-if="newRepModal"
      class="fixed inset-0 bg-black/75 backdrop-blur-xs flex items-center justify-center z-60 p-4"
      @click.self="newRepModal = false"
    >
      <div class="bg-zinc-900 border border-zinc-700 rounded-2xl w-full max-w-md p-5 flex flex-col gap-4 shadow-2xl">
        <h3 class="font-bold text-sm text-zinc-100">Create New Opening Repertoire</h3>
        
        <div class="space-y-1">
          <label class="text-xs font-semibold text-zinc-400">Repertoire Name</label>
          <input
            v-model="newRepName"
            type="text"
            placeholder="e.g. White: Ruy Lopez & Scotch"
            class="w-full bg-zinc-950 border border-zinc-700 rounded-lg p-2 text-xs text-zinc-100 focus:outline-hidden focus:border-emerald-500"
          />
        </div>

        <div class="space-y-1">
          <label class="text-xs font-semibold text-zinc-400">Color</label>
          <div class="grid grid-cols-2 gap-2">
            <button
              class="py-2 rounded-lg text-xs font-semibold border transition-all cursor-pointer flex items-center justify-center gap-2"
              :class="newRepColor === 'w' ? 'bg-white text-zinc-950 border-white' : 'bg-zinc-950 text-zinc-400 border-zinc-800'"
              @click="newRepColor = 'w'"
            >
              <span>White Repertoire</span>
            </button>
            <button
              class="py-2 rounded-lg text-xs font-semibold border transition-all cursor-pointer flex items-center justify-center gap-2"
              :class="newRepColor === 'b' ? 'bg-zinc-800 text-white border-zinc-600' : 'bg-zinc-950 text-zinc-400 border-zinc-800'"
              @click="newRepColor = 'b'"
            >
              <span>Black Repertoire</span>
            </button>
          </div>
        </div>

        <div class="space-y-1">
          <label class="text-xs font-semibold text-zinc-400">Description</label>
          <textarea
            v-model="newRepDesc"
            rows="2"
            placeholder="Key plans, favorite variations..."
            class="w-full bg-zinc-950 border border-zinc-700 rounded-lg p-2 text-xs text-zinc-100 focus:outline-hidden focus:border-emerald-500"
          />
        </div>

        <div class="flex justify-end gap-2 mt-2">
          <button
            class="px-4 py-1.5 rounded-lg bg-zinc-800 hover:bg-zinc-700 text-zinc-300 text-xs font-medium cursor-pointer"
            @click="newRepModal = false"
          >
            Cancel
          </button>
          <button
            class="px-4 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white text-xs font-semibold cursor-pointer"
            @click="createNewRepertoire"
          >
            Create
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
