<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { X, Trophy, RotateCcw, Lightbulb, Undo2 } from '@lucide/vue'
import Chessboard from './Chessboard.vue'
import {
  getEndgameDrills,
  evaluateEndgameMove,
  type EndgameDrill,
  type EndgameEvalFeedback,
} from '../utils/v21Api'

const props = defineProps<{
  show: boolean
  pieceSet?: string
  boardTheme?: { light: string; dark: string; highlight: string }
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'loadPosition', fen: string): void
}>()

const category = ref('all')
const drills = ref<EndgameDrill[]>([])
const activeDrill = ref<EndgameDrill | null>(null)
const currentFEN = ref('')
const legalMoves = ref<string[]>([])
const history = ref<string[]>([])
const feedback = ref<EndgameEvalFeedback | null>(null)
const showHint = ref(false)
const useTablebase = ref(true)
const thinking = ref(false)

const categories = [
  { id: 'all', label: 'All' },
  { id: 'rook', label: 'Rook' },
  { id: 'pawn', label: 'Pawn' },
  { id: 'minor', label: 'Minor' },
  { id: 'queen', label: 'Queen' },
]

async function loadCatalog() {
  drills.value = await getEndgameDrills(category.value)
  if (!activeDrill.value && drills.value.length > 0) {
    await selectDrill(drills.value[0])
  }
}

async function refreshLegalMoves(fen: string) {
  const app = (window as any)?.go?.main?.App
  if (app?.ParseFEN) {
    try {
      const state = await app.ParseFEN(fen)
      legalMoves.value = state?.legalMoves || []
      return
    } catch {
      // ignore
    }
  }
  legalMoves.value = []
}

async function selectDrill(d: EndgameDrill) {
  activeDrill.value = d
  currentFEN.value = d.fen
  history.value = [d.fen]
  feedback.value = null
  showHint.value = false
  await refreshLegalMoves(d.fen)
}

async function handleUserMove(uci: string) {
  if (!activeDrill.value || thinking.value) return
  const app = (window as any)?.go?.main?.App
  if (!app?.MakeMove) return

  thinking.value = true
  try {
    const afterUser = await app.MakeMove(currentFEN.value, uci)
    if (!afterUser?.fen) return
    currentFEN.value = afterUser.fen
    history.value.push(afterUser.fen)

    const evalRes = await evaluateEndgameMove(
      afterUser.fen,
      activeDrill.value.objective,
      activeDrill.value.sideToPlay,
      useTablebase.value
    )
    feedback.value = evalRes

    // Auto-play opponent's tablebase/heuristic reply if game continues
    if (evalRes?.bestReplyUci && evalRes.status !== 'won' && evalRes.status !== 'drawn' && evalRes.status !== 'lost') {
      const afterOpp = await app.MakeMove(afterUser.fen, evalRes.bestReplyUci)
      if (afterOpp?.fen) {
        currentFEN.value = afterOpp.fen
        history.value.push(afterOpp.fen)
      }
    }
    await refreshLegalMoves(currentFEN.value)
  } finally {
    thinking.value = false
  }
}

async function retryLastMove() {
  if (history.value.length <= 1) return
  // Pop opponent reply + user move if possible
  const popCount = history.value.length >= 3 ? 2 : 1
  history.value.splice(history.value.length - popCount, popCount)
  currentFEN.value = history.value[history.value.length - 1]
  feedback.value = null
  await refreshLegalMoves(currentFEN.value)
}

const isFlipped = computed(() => activeDrill.value?.sideToPlay === 'black')

watch(() => props.show, (v) => {
  if (v) loadCatalog()
})
watch(category, () => {
  if (props.show) loadCatalog()
})
</script>

<template>
  <div v-if="show" class="fixed inset-0 z-50 bg-black/70 backdrop-blur-sm flex items-center justify-center p-4">
    <div class="bg-zinc-900 border border-zinc-700 rounded-xl shadow-2xl w-full max-w-5xl max-h-[90vh] flex flex-col overflow-hidden">
      <div class="flex items-center justify-between px-5 py-3.5 border-b border-zinc-800 bg-zinc-950/60">
        <div class="flex items-center gap-2.5">
          <Trophy class="w-5 h-5 text-amber-400" />
          <div>
            <h2 class="text-sm font-bold text-zinc-100">Endgame Sparring &amp; Theoretical Trainer</h2>
            <p class="text-[11px] text-zinc-400">Practice essential endings with Syzygy WDL/DTZ feedback</p>
          </div>
        </div>
        <button class="p-1.5 rounded hover:bg-zinc-800 text-zinc-400 hover:text-zinc-100 cursor-pointer" @click="emit('close')">
          <X class="w-4 h-4" />
        </button>
      </div>

      <div class="grid grid-cols-1 md:grid-cols-12 gap-4 p-4 overflow-y-auto">
        <!-- Left Drill List -->
        <div class="md:col-span-5 flex flex-col gap-3">
          <div class="flex flex-wrap gap-1.5">
            <button
              v-for="c in categories"
              :key="c.id"
              class="px-2.5 py-1 rounded text-xs font-medium cursor-pointer transition-colors"
              :class="category === c.id ? 'bg-amber-500/20 text-amber-300 border border-amber-500/50' : 'bg-zinc-950 text-zinc-400 border border-zinc-800 hover:text-zinc-200'"
              @click="category = c.id"
            >
              {{ c.label }}
            </button>
          </div>

          <div class="space-y-1.5 max-h-[56vh] overflow-y-auto pr-1">
            <button
              v-for="d in drills"
              :key="d.id"
              class="w-full text-left p-2.5 rounded-lg border transition-colors cursor-pointer"
              :class="activeDrill?.id === d.id
                ? 'bg-amber-500/15 border-amber-500/60 text-zinc-100'
                : 'bg-zinc-950/50 border-zinc-800 text-zinc-300 hover:border-zinc-700'"
              @click="selectDrill(d)"
            >
              <div class="flex items-center justify-between gap-2">
                <span class="text-xs font-semibold truncate">{{ d.title }}</span>
                <span
                  class="text-[10px] px-1.5 py-0.5 rounded uppercase font-bold shrink-0"
                  :class="d.objective === 'win' ? 'bg-emerald-500/20 text-emerald-300' : 'bg-sky-500/20 text-sky-300'"
                >
                  {{ d.objective }}
                </span>
              </div>
              <div class="text-[10px] text-zinc-400 mt-0.5">
                Play as {{ d.sideToPlay }} · {{ d.difficulty }}
              </div>
            </button>
          </div>
        </div>

        <!-- Right Interactive Sparring Board -->
        <div v-if="activeDrill" class="md:col-span-7 flex flex-col gap-3">
          <div class="flex items-center justify-center bg-zinc-950/60 border border-zinc-800 rounded-xl p-3">
            <Chessboard
              :fen="currentFEN"
              :legal-moves="legalMoves"
              :flipped="isFlipped"
              :piece-set="pieceSet"
              :board-theme="boardTheme"
              :board-size="360"
              @move="handleUserMove"
            />
          </div>

          <div class="bg-zinc-950/60 border border-zinc-800 rounded-lg p-3 space-y-2">
            <div class="text-xs font-semibold text-zinc-100">{{ activeDrill.title }}</div>
            <p class="text-xs text-zinc-400">{{ activeDrill.keyConcept }}</p>

            <div
              v-if="feedback"
              class="text-xs rounded-md p-2.5 border"
              :class="feedback.lostObjective
                ? 'bg-red-950/40 border-red-700/60 text-red-200'
                : 'bg-emerald-950/40 border-emerald-700/60 text-emerald-200'"
            >
              <strong>{{ feedback.wdlText }}</strong> — {{ feedback.message }}
            </div>

            <div v-if="showHint" class="text-xs text-amber-200 bg-amber-950/40 border border-amber-700/50 rounded-md p-2.5">
              Hint: {{ activeDrill.hint }} (Try <strong>{{ activeDrill.firstMoveSan }}</strong>)
            </div>

            <div class="flex flex-wrap items-center gap-2 pt-1">
              <button class="px-2.5 py-1.5 rounded bg-zinc-800 hover:bg-zinc-700 text-xs text-zinc-200 flex items-center gap-1 cursor-pointer" @click="showHint = !showHint">
                <Lightbulb class="w-3.5 h-3.5 text-amber-400" /> Hint
              </button>
              <button class="px-2.5 py-1.5 rounded bg-zinc-800 hover:bg-zinc-700 text-xs text-zinc-200 flex items-center gap-1 cursor-pointer" @click="retryLastMove">
                <Undo2 class="w-3.5 h-3.5" /> Retry Move
              </button>
              <button class="px-2.5 py-1.5 rounded bg-zinc-800 hover:bg-zinc-700 text-xs text-zinc-200 flex items-center gap-1 cursor-pointer" @click="selectDrill(activeDrill)">
                <RotateCcw class="w-3.5 h-3.5" /> Reset
              </button>
              <label class="ml-auto flex items-center gap-1.5 text-[11px] text-zinc-400 cursor-pointer">
                <input v-model="useTablebase" type="checkbox" class="accent-amber-500" />
                Syzygy WDL/DTZ
              </label>
              <button
                class="px-3 py-1.5 rounded bg-amber-600 hover:bg-amber-500 text-xs font-semibold text-white cursor-pointer"
                @click="emit('loadPosition', currentFEN); emit('close')"
              >
                Open on Main Board
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
