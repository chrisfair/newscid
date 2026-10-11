<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { X, Award, Flame, Check, AlertOctagon, RotateCcw, ArrowRight, Eye } from '@lucide/vue'
import Chessboard from './Chessboard.vue'
import { GetDueBlunderPuzzles, GetAllBlunderPuzzles, RecordBlunderReview } from '../../wailsjs/go/main/App'
import type { blundervault } from '../../wailsjs/go/models'
import { BoardThemeOption } from './HeaderBar.vue'
import { playCaptureSound, playMoveSound } from '../utils/sound'

const props = defineProps<{
  boardTheme: BoardThemeOption
  pieceSet: string
}>()

const emit = defineEmits<{
  (e: 'close'): void
}>()

const loading = ref(true)
const puzzles = ref<blundervault.Puzzle[]>([])
const currentIndex = ref(0)
const revealed = ref(false)
const userFeedback = ref<'correct' | 'wrong' | null>(null)
const solvedCount = ref(0)
const currentBoardFEN = ref('')

const currentPuzzle = computed(() => {
  if (puzzles.value.length === 0 || currentIndex.value >= puzzles.value.length) {
    return null
  }
  return puzzles.value[currentIndex.value]
})

async function fetchPuzzles() {
  loading.value = true
  try {
    let list = await GetDueBlunderPuzzles()
    if (!list || list.length === 0) {
      list = await GetAllBlunderPuzzles()
    }
    puzzles.value = list || []
    currentIndex.value = 0
    loadCurrentPuzzle()
  } catch (err) {
    console.error('Error fetching blunder puzzles:', err)
  } finally {
    loading.value = false
  }
}

function loadCurrentPuzzle() {
  revealed.value = false
  userFeedback.value = null
  if (currentPuzzle.value) {
    currentBoardFEN.value = currentPuzzle.value.fen
  }
}

function handleMove(uci: string) {
  if (!currentPuzzle.value || revealed.value) return

  if (uci === currentPuzzle.value.bestMove) {
    userFeedback.value = 'correct'
    revealed.value = true
    solvedCount.value++
    playCaptureSound()
  } else {
    userFeedback.value = 'wrong'
    playMoveSound()
  }
}

function revealSolution() {
  revealed.value = true
}

async function rateAndAdvance(quality: number) {
  if (!currentPuzzle.value) return

  try {
    await RecordBlunderReview(currentPuzzle.value.id, quality)
  } catch (err) {
    console.error('Record review error:', err)
  }

  if (currentIndex.value < puzzles.value.length - 1) {
    currentIndex.value++
    loadCurrentPuzzle()
  } else {
    // Finished all
    currentIndex.value++
  }
}

onMounted(() => {
  fetchPuzzles()
})
</script>

<template>
  <div
    class="fixed inset-0 bg-black/75 backdrop-blur-xs flex items-center justify-center z-50 p-4 animate-fade-in select-none"
    @click.self="emit('close')"
  >
    <div
      class="bg-zinc-900 border border-zinc-700/80 w-full max-w-2xl rounded-2xl p-5 shadow-2xl flex flex-col gap-4 text-xs"
    >
      <!-- Header -->
      <div class="flex items-center justify-between border-b border-zinc-800 pb-3">
        <div class="flex items-center gap-2.5">
          <div class="w-8 h-8 rounded-lg bg-red-500/20 border border-red-500/40 flex items-center justify-center text-red-400">
            <Flame class="w-4 h-4" />
          </div>
          <div>
            <h3 class="font-bold text-sm text-zinc-100 flex items-center gap-2">
              Personal Blunder Vault
              <span v-if="puzzles.length > 0" class="text-[10px] px-2 py-0.5 rounded-full bg-red-500/20 text-red-300 font-mono">
                {{ currentIndex < puzzles.length ? `${currentIndex + 1} / ${puzzles.length}` : 'Complete' }}
              </span>
            </h3>
            <p class="text-[11px] text-zinc-400">Spaced repetition review of your actual game blunders</p>
          </div>
        </div>
        <button
          class="p-1.5 rounded-lg text-zinc-400 hover:text-zinc-200 hover:bg-zinc-800 transition-colors cursor-pointer"
          @click="emit('close')"
        >
          <X class="w-4 h-4" />
        </button>
      </div>

      <!-- Loading State -->
      <div v-if="loading" class="py-12 text-center text-zinc-400 animate-pulse">
        Loading personal blunder flashcards...
      </div>

      <!-- Empty State -->
      <div v-else-if="puzzles.length === 0" class="py-12 flex flex-col items-center justify-center gap-3 text-center">
        <div class="w-12 h-12 rounded-full bg-emerald-500/20 border border-emerald-500/40 flex items-center justify-center text-emerald-400">
          <Check class="w-6 h-6" />
        </div>
        <h4 class="font-bold text-sm text-zinc-200">No Blunders in Vault Yet!</h4>
        <p class="text-zinc-400 text-xs max-w-sm">
          Play sparring matches, import your games from Lichess/Chess.com, or mine your database to queue your mistakes into this spaced-repetition trainer.
        </p>
        <button
          class="mt-2 px-4 py-2 rounded-xl bg-zinc-800 hover:bg-zinc-700 text-zinc-200 text-xs font-semibold cursor-pointer"
          @click="emit('close')"
        >
          Return to Workspace
        </button>
      </div>

      <!-- Completed Deck State -->
      <div v-else-if="currentIndex >= puzzles.length" class="py-10 flex flex-col items-center justify-center gap-3 text-center">
        <div class="w-12 h-12 rounded-full bg-amber-500/20 border border-amber-500/40 flex items-center justify-center text-amber-400">
          <Award class="w-6 h-6" />
        </div>
        <h4 class="font-bold text-base text-zinc-100">Workout Complete!</h4>
        <p class="text-zinc-400 text-xs max-w-sm">
          You reviewed all due blunder flashcards in your deck. SuperMemo SM-2 intervals have been scheduled.
        </p>
        <button
          class="mt-3 px-5 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-500 text-white font-medium text-xs cursor-pointer shadow-lg"
          @click="emit('close')"
        >
          Done
        </button>
      </div>

      <!-- Active Puzzle Workout -->
      <div v-else-if="currentPuzzle" class="flex flex-col md:flex-row items-center gap-5">
        <!-- Interactive Chessboard -->
        <div class="shrink-0 flex items-center justify-center w-80 h-80 sm:w-96 sm:h-96">
          <Chessboard
            :fen="currentBoardFEN"
            :board-theme="boardTheme"
            :piece-set="pieceSet"
            :flipped="currentPuzzle.userColor === 'black'"
            @move="handleMove"
          />
        </div>

        <!-- Right Side: Puzzle Context & Evaluation -->
        <div class="flex-1 flex flex-col justify-between self-stretch gap-3">
          <!-- Game Context Card -->
          <div class="p-3 bg-zinc-950/80 rounded-xl border border-zinc-800 flex flex-col gap-1.5">
            <div class="flex items-center justify-between text-[11px]">
              <span class="font-bold text-zinc-300">You played: {{ currentPuzzle.userColor.toUpperCase() }}</span>
              <span class="text-red-400 font-mono font-bold">{{ currentPuzzle.deltaCp }} cp mistake</span>
            </div>
            <p class="text-zinc-400 text-[11px] truncate">
              {{ currentPuzzle.event }} ({{ currentPuzzle.date }})
            </p>
            <div class="h-px bg-zinc-800 my-1" />
            <div class="text-[11px] text-zinc-300">
              In your game, you played: <span class="font-mono text-red-400 font-bold">{{ currentPuzzle.playedMove }}</span>
            </div>
            <p class="text-xs text-amber-300 font-medium mt-1">
              Find the winning move you missed!
            </p>
          </div>

          <!-- Feedback Status -->
          <div
            v-if="userFeedback === 'correct'"
            class="p-3 rounded-xl bg-emerald-500/20 border border-emerald-500/40 text-emerald-300 text-xs font-semibold flex items-center gap-2"
          >
            <Check class="w-4 h-4" /> Correct! You found the master move: {{ currentPuzzle.bestMove }}
          </div>
          <div
            v-else-if="userFeedback === 'wrong'"
            class="p-3 rounded-xl bg-red-500/20 border border-red-500/40 text-red-300 text-xs font-semibold flex items-center gap-2"
          >
            <AlertOctagon class="w-4 h-4" /> Not the best move. Try again or reveal solution!
          </div>

          <!-- Solution Display -->
          <div
            v-if="revealed"
            class="p-3 bg-zinc-950 border border-zinc-800 rounded-xl text-xs flex flex-col gap-1"
          >
            <div class="text-zinc-400 text-[11px]">Best engine move:</div>
            <div class="font-mono text-emerald-400 font-bold text-sm">{{ currentPuzzle.bestMove }}</div>
            <p v-if="currentPuzzle.explanation" class="text-zinc-300 text-xs mt-1">
              {{ currentPuzzle.explanation }}
            </p>
          </div>

          <!-- Action & Rating Controls -->
          <div class="flex flex-col gap-2 mt-auto">
            <button
              v-if="!revealed"
              class="w-full py-2 rounded-xl bg-zinc-800 hover:bg-zinc-700 text-zinc-300 text-xs font-medium cursor-pointer flex items-center justify-center gap-2 transition-colors"
              @click="revealSolution"
            >
              <Eye class="w-3.5 h-3.5 text-zinc-400" /> Reveal Solution
            </button>

            <!-- SM-2 Rating Buttons (visible when revealed) -->
            <div v-else class="flex flex-col gap-1.5">
              <span class="text-[10px] text-zinc-500 uppercase tracking-wider font-semibold text-center">
                Rate your recall (Spaced Repetition):
              </span>
              <div class="grid grid-cols-4 gap-1.5">
                <button
                  class="py-2 px-1 rounded-lg bg-red-950/60 hover:bg-red-900 border border-red-800/60 text-red-300 text-[11px] font-semibold cursor-pointer"
                  @click="rateAndAdvance(1)"
                >
                  Again (1d)
                </button>
                <button
                  class="py-2 px-1 rounded-lg bg-amber-950/60 hover:bg-amber-900 border border-amber-800/60 text-amber-300 text-[11px] font-semibold cursor-pointer"
                  @click="rateAndAdvance(3)"
                >
                  Hard (3d)
                </button>
                <button
                  class="py-2 px-1 rounded-lg bg-blue-950/60 hover:bg-blue-900 border border-blue-800/60 text-blue-300 text-[11px] font-semibold cursor-pointer"
                  @click="rateAndAdvance(4)"
                >
                  Good (6d)
                </button>
                <button
                  class="py-2 px-1 rounded-lg bg-emerald-950/60 hover:bg-emerald-900 border border-emerald-800/60 text-emerald-300 text-[11px] font-semibold cursor-pointer"
                  @click="rateAndAdvance(5)"
                >
                  Easy (14d)
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
