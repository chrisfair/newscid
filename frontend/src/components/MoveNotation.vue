<script setup lang="ts">
import { ref, computed } from 'vue'
import {
  ChevronLeft,
  ChevronRight,
  ChevronsLeft,
  ChevronsRight,
  Play,
  Pause,
  Copy,
  MessageSquare,
  MessageSquarePlus,
  Trash2,
  ArrowUpCircle,
  Tag,
  Check,
  X,
  SlidersHorizontal
} from '@lucide/vue'
import { useNotation, formatMoveNotation, type NotationStyle } from '../utils/notation'

export interface VariationLine {
  id: number
  moves: FlatMove[]
}

export interface FlatMove {
  index: number
  ply: number
  moveNumber: number
  isWhite: boolean
  san: string
  uci: string
  fen: string
  comment?: string
  nags?: number[]
  variationLevel: number
  variations?: VariationLine[]
}

const props = withDefaults(defineProps<{
  moves: FlatMove[]
  currentPly: number
  isPlaying?: boolean
  result?: string
}>(), {
  moves: () => [],
  currentPly: 0,
  isPlaying: false,
  result: '*',
})

const emit = defineEmits<{
  (e: 'jump', ply: number): void
  (e: 'first'): void
  (e: 'prev'): void
  (e: 'next'): void
  (e: 'last'): void
  (e: 'togglePlay'): void
  (e: 'copyPGN'): void
  (e: 'setNAG', ply: number, nag: number): void
  (e: 'clearNAGs', ply: number): void
  (e: 'setComment', ply: number, comment: string): void
  (e: 'promoteVariation', ply: number, varIdx: number): void
  (e: 'deleteVariation', ply: number, varIdx: number): void
}>()

const { currentStyle, setNotationStyle, options: notationOptions } = useNotation()

// Palette & Comment states
const showPalette = ref(true)
const showCommentModal = ref(false)
const editingCommentPly = ref<number | null>(null)
const commentInput = ref('')

// Find move by ply
function findMoveByPly(ply: number): FlatMove | undefined {
  return props.moves.find(m => m.ply === ply)
}

const currentMove = computed(() => {
  return findMoveByPly(props.currentPly)
})

// NAG Definitions
interface NAGItem {
  id: number
  symbol: string
  label: string
  category: 'quality' | 'eval'
}

const QUALITY_NAGS: NAGItem[] = [
  { id: 1, symbol: '!', label: 'Good move', category: 'quality' },
  { id: 2, symbol: '?', label: 'Mistake', category: 'quality' },
  { id: 3, symbol: '!!', label: 'Brilliant', category: 'quality' },
  { id: 4, symbol: '??', label: 'Blunder', category: 'quality' },
  { id: 5, symbol: '!?', label: 'Interesting', category: 'quality' },
  { id: 6, symbol: '?!', label: 'Dubious', category: 'quality' },
]

const EVAL_NAGS: NAGItem[] = [
  { id: 10, symbol: '=', label: 'Equal', category: 'eval' },
  { id: 14, symbol: '⩲', label: 'White slightly better (+/=)', category: 'eval' },
  { id: 15, symbol: '⩱', label: 'Black slightly better (=/+)', category: 'eval' },
  { id: 16, symbol: '±', label: 'White clearly better', category: 'eval' },
  { id: 17, symbol: '∓', label: 'Black clearly better', category: 'eval' },
  { id: 18, symbol: '+-', label: 'White winning', category: 'eval' },
  { id: 19, symbol: '-+', label: 'Black winning', category: 'eval' },
  { id: 13, symbol: '∞', label: 'Unclear', category: 'eval' },
  { id: 44, symbol: '⯹', label: 'With compensation', category: 'eval' },
]

function formatNAG(nag: number): string {
  switch (nag) {
    case 1: return '!'
    case 2: return '?'
    case 3: return '!!'
    case 4: return '??'
    case 5: return '!?'
    case 6: return '?!'
    case 10: return '='
    case 14: return '⩲'
    case 15: return '⩱'
    case 16: return '±'
    case 17: return '∓'
    case 18: return '+-'
    case 19: return '-+'
    case 13: return '∞'
    case 44: return '⯹'
    default: return `$${nag}`
  }
}

function handleNAGClick(nag: number) {
  if (props.currentPly <= 0) return
  emit('setNAG', props.currentPly, nag)
}

function handleClearNAGs() {
  if (props.currentPly <= 0) return
  emit('clearNAGs', props.currentPly)
}

function openCommentModal(ply?: number) {
  const targetPly = ply !== undefined ? ply : props.currentPly
  if (targetPly <= 0) return
  editingCommentPly.value = targetPly
  const m = findMoveByPly(targetPly)
  commentInput.value = m?.comment || ''
  showCommentModal.value = true
}

function saveComment() {
  if (editingCommentPly.value !== null) {
    emit('setComment', editingCommentPly.value, commentInput.value.trim())
  }
  showCommentModal.value = false
  editingCommentPly.value = null
  commentInput.value = ''
}

function removeComment() {
  if (editingCommentPly.value !== null) {
    emit('setComment', editingCommentPly.value, '')
  }
  showCommentModal.value = false
  editingCommentPly.value = null
  commentInput.value = ''
}

// Group main line moves into pairs for standard scorebook layout
const movePairs = computed(() => {
  const pairs: { moveNumber: number; white?: FlatMove; black?: FlatMove }[] = []
  const mainMoves = props.moves.filter(m => m.variationLevel === 0)

  let currentPair: { moveNumber: number; white?: FlatMove; black?: FlatMove } | null = null

  for (const m of mainMoves) {
    if (m.isWhite) {
      if (currentPair) {
        pairs.push(currentPair)
      }
      currentPair = { moveNumber: m.moveNumber, white: m }
    } else {
      if (!currentPair) {
        currentPair = { moveNumber: m.moveNumber, black: m }
      } else {
        currentPair.black = m
        pairs.push(currentPair)
        currentPair = null
      }
    }
  }

  if (currentPair) {
    pairs.push(currentPair)
  }

  return pairs
})

function formatMove(m?: FlatMove): string {
  if (!m) return ''
  return formatMoveNotation(m.san, m.uci, m.isWhite, currentStyle.value)
}
</script>

<template>
  <div class="move-notation flex flex-col h-full bg-zinc-900 border border-zinc-800 rounded-xl overflow-hidden shadow-lg select-none">
    <!-- Move Navigation Header -->
    <div class="nav-toolbar flex flex-wrap items-center justify-between p-2 bg-zinc-950 border-b border-zinc-800 gap-1">
      <div class="flex items-center gap-1">
        <button
          class="nav-btn p-1.5 rounded-lg text-zinc-400 hover:text-white hover:bg-zinc-800 transition-colors"
          title="First Move (Home)"
          @click="emit('first')"
        >
          <ChevronsLeft class="w-4 h-4" />
        </button>
        <button
          class="nav-btn p-1.5 rounded-lg text-zinc-400 hover:text-white hover:bg-zinc-800 transition-colors"
          title="Previous Move (Left Arrow)"
          @click="emit('prev')"
        >
          <ChevronLeft class="w-4 h-4" />
        </button>
        <button
          class="nav-btn p-1.5 rounded-lg text-zinc-400 hover:text-white hover:bg-zinc-800 transition-colors"
          :title="isPlaying ? 'Pause' : 'Autoplay (Space)'"
          @click="emit('togglePlay')"
        >
          <Pause v-if="isPlaying" class="w-4 h-4 text-emerald-400" />
          <Play v-else class="w-4 h-4" />
        </button>
        <button
          class="nav-btn p-1.5 rounded-lg text-zinc-400 hover:text-white hover:bg-zinc-800 transition-colors"
          title="Next Move (Right Arrow)"
          @click="emit('next')"
        >
          <ChevronRight class="w-4 h-4" />
        </button>
        <button
          class="nav-btn p-1.5 rounded-lg text-zinc-400 hover:text-white hover:bg-zinc-800 transition-colors"
          title="Last Move (End)"
          @click="emit('last')"
        >
          <ChevronsRight class="w-4 h-4" />
        </button>
      </div>

      <!-- Right controls: Notation selector & PGN copy -->
      <div class="flex items-center gap-1">
        <!-- Notation Style Selector -->
        <div class="flex items-center bg-zinc-900 border border-zinc-800 rounded-lg px-2 py-1">
          <span class="text-[10px] text-zinc-500 font-bold uppercase tracking-wider mr-1.5">Style</span>
          <select
            :value="currentStyle"
            class="bg-transparent text-xs text-zinc-200 font-semibold focus:outline-none cursor-pointer"
            @change="setNotationStyle(($event.target as HTMLSelectElement).value as NotationStyle)"
          >
            <option v-for="opt in notationOptions" :key="opt.id" :value="opt.id" class="bg-zinc-900 text-zinc-100">
              {{ opt.name }}
            </option>
          </select>
        </div>

        <!-- Annotation Palette Toggle -->
        <button
          class="p-1.5 rounded-lg text-zinc-400 hover:text-white hover:bg-zinc-800 transition-colors"
          :class="showPalette ? 'text-amber-400 bg-zinc-800/60' : ''"
          title="Toggle Scorebook Annotation Palette"
          @click="showPalette = !showPalette"
        >
          <Tag class="w-3.5 h-3.5" />
        </button>

        <!-- Copy PGN -->
        <button
          class="p-1.5 rounded-lg text-zinc-400 hover:text-white hover:bg-zinc-800 transition-colors text-xs flex items-center gap-1"
          title="Copy PGN"
          @click="emit('copyPGN')"
        >
          <Copy class="w-3.5 h-3.5" />
          <span class="hidden sm:inline">PGN</span>
        </button>
      </div>
    </div>

    <!-- Collapsible Scorebook Annotation Palette Toolbar -->
    <div
      v-if="showPalette"
      class="annotation-palette bg-zinc-950/80 border-b border-zinc-800/80 p-2 flex flex-wrap items-center gap-1 text-xs"
    >
      <!-- Move Quality NAGs -->
      <div class="flex items-center gap-0.5 bg-zinc-900/90 border border-zinc-800 rounded-lg p-0.5 shadow-sm">
        <button
          v-for="item in QUALITY_NAGS"
          :key="item.id"
          class="px-1.5 py-0.5 rounded text-xs font-bold transition-all"
          :class="currentMove?.nags?.includes(item.id)
            ? 'bg-amber-500 text-zinc-950 shadow-sm'
            : 'text-zinc-300 hover:bg-zinc-800 hover:text-white'"
          :title="`${item.symbol} (${item.label})`"
          @click="handleNAGClick(item.id)"
        >
          {{ item.symbol }}
        </button>
      </div>

      <!-- Evaluation NAGs -->
      <div class="flex items-center gap-0.5 bg-zinc-900/90 border border-zinc-800 rounded-lg p-0.5 shadow-sm">
        <button
          v-for="item in EVAL_NAGS"
          :key="item.id"
          class="px-1.5 py-0.5 rounded text-xs font-bold transition-all"
          :class="currentMove?.nags?.includes(item.id)
            ? 'bg-emerald-500 text-zinc-950 shadow-sm'
            : 'text-zinc-300 hover:bg-zinc-800 hover:text-white'"
          :title="`${item.symbol} (${item.label})`"
          @click="handleNAGClick(item.id)"
        >
          {{ item.symbol }}
        </button>
      </div>

      <!-- Clear NAG button -->
      <button
        v-if="currentMove?.nags && currentMove.nags.length > 0"
        class="px-1.5 py-1 rounded-lg text-xs font-medium text-rose-400 hover:bg-rose-950/50 hover:text-rose-300 transition-colors flex items-center gap-0.5"
        title="Clear NAGs on this move"
        @click="handleClearNAGs"
      >
        <X class="w-3 h-3" />
        <span>Clear</span>
      </button>

      <!-- Add / Edit Comment Button -->
      <button
        class="ml-auto px-2 py-1 rounded-lg text-xs font-medium bg-zinc-800/80 hover:bg-zinc-700/80 text-zinc-200 transition-colors flex items-center gap-1"
        :class="currentMove?.comment ? 'text-sky-300 border border-sky-500/30' : ''"
        :title="currentMove?.comment ? 'Edit move comment' : 'Add comment to move'"
        @click="openCommentModal()"
      >
        <MessageSquare class="w-3.5 h-3.5 text-sky-400" />
        <span>{{ currentMove?.comment ? 'Edit Comment' : 'Add Comment' }}</span>
      </button>
    </div>

    <!-- Moves Scrollable Container -->
    <div class="moves-container flex-1 p-3 overflow-y-auto font-mono text-sm space-y-1">
      <div v-if="movePairs.length === 0" class="text-zinc-500 text-xs italic text-center py-6">
        No moves played yet. Play a move on the board or load a game.
      </div>

      <div
        v-for="pair in movePairs"
        :key="pair.moveNumber"
        class="move-row-wrapper"
      >
        <div class="move-row flex items-baseline gap-2 py-0.5 px-2 rounded-md hover:bg-zinc-800/40">
          <!-- Move number -->
          <span class="move-num text-zinc-500 text-xs w-7 select-none text-right">
            {{ pair.moveNumber }}.
          </span>

          <!-- White Move -->
          <div class="flex-1 flex flex-col items-start">
            <div class="flex items-center gap-1">
              <button
                v-if="pair.white"
                class="move-btn px-2 py-0.5 rounded text-left font-semibold transition-all cursor-pointer flex items-center gap-1"
                :class="pair.white.ply === currentPly
                  ? 'bg-emerald-600 text-white shadow-sm ring-1 ring-emerald-400'
                  : 'text-zinc-200 hover:bg-zinc-700/60'"
                @click="emit('jump', pair.white.ply)"
                @dblclick="openCommentModal(pair.white.ply)"
              >
                <span>{{ formatMove(pair.white) }}</span>
                <span
                  v-for="nag in pair.white.nags"
                  :key="nag"
                  class="nag text-amber-400 font-bold ml-0.5 text-xs"
                >
                  {{ formatNAG(nag) }}
                </span>
                <span v-if="pair.white.comment" class="w-1.5 h-1.5 rounded-full bg-sky-400 ml-1 inline-block" title="Has comment" />
              </button>
            </div>

            <!-- White Inline Comment -->
            <div
              v-if="pair.white?.comment"
              class="comment-box mt-0.5 ml-2 p-1 px-2 rounded bg-zinc-950/70 border border-zinc-800/60 text-xs text-zinc-400 font-sans italic flex items-start gap-1"
              @click="openCommentModal(pair.white.ply)"
            >
              <MessageSquare class="w-3 h-3 text-sky-400 shrink-0 mt-0.5" />
              <span>{{ pair.white.comment }}</span>
            </div>

            <!-- White Variations (if any) -->
            <div
              v-if="pair.white?.variations && pair.white.variations.length > 0"
              class="variations-container mt-1 ml-4 pl-2 border-l-2 border-zinc-700/60 space-y-1 font-sans text-xs"
            >
              <div
                v-for="(vLine, vIdx) in pair.white.variations"
                :key="vLine.id || vIdx"
                class="variation-line flex items-center gap-1 text-zinc-400 bg-zinc-950/40 p-1 rounded"
              >
                <span class="text-zinc-500 font-mono text-[10px]">({{ vIdx + 1 }})</span>
                <span class="text-zinc-300">
                  <span
                    v-for="vm in vLine.moves"
                    :key="vm.ply"
                    class="hover:text-emerald-400 cursor-pointer mr-1"
                    :class="vm.ply === currentPly ? 'text-emerald-400 font-bold' : ''"
                    @click="emit('jump', vm.ply)"
                  >
                    {{ vm.isWhite ? `${vm.moveNumber}. ` : '' }}{{ formatMove(vm) }}
                  </span>
                </span>

                <div class="ml-auto flex items-center gap-1">
                  <button
                    class="p-0.5 rounded hover:text-emerald-300 hover:bg-zinc-800 transition-colors"
                    title="Promote variation to main line"
                    @click.stop="emit('promoteVariation', pair.white!.ply, vIdx)"
                  >
                    <ArrowUpCircle class="w-3.5 h-3.5" />
                  </button>
                  <button
                    class="p-0.5 rounded hover:text-rose-400 hover:bg-zinc-800 transition-colors"
                    title="Delete variation"
                    @click.stop="emit('deleteVariation', pair.white!.ply, vIdx)"
                  >
                    <Trash2 class="w-3.5 h-3.5" />
                  </button>
                </div>
              </div>
            </div>
          </div>

          <!-- Black Move -->
          <div class="flex-1 flex flex-col items-start">
            <div class="flex items-center gap-1">
              <button
                v-if="pair.black"
                class="move-btn px-2 py-0.5 rounded text-left font-semibold transition-all cursor-pointer flex items-center gap-1"
                :class="pair.black.ply === currentPly
                  ? 'bg-emerald-600 text-white shadow-sm ring-1 ring-emerald-400'
                  : 'text-zinc-200 hover:bg-zinc-700/60'"
                @click="emit('jump', pair.black.ply)"
                @dblclick="openCommentModal(pair.black.ply)"
              >
                <span>{{ formatMove(pair.black) }}</span>
                <span
                  v-for="nag in pair.black.nags"
                  :key="nag"
                  class="nag text-amber-400 font-bold ml-0.5 text-xs"
                >
                  {{ formatNAG(nag) }}
                </span>
                <span v-if="pair.black.comment" class="w-1.5 h-1.5 rounded-full bg-sky-400 ml-1 inline-block" title="Has comment" />
              </button>
            </div>

            <!-- Black Inline Comment -->
            <div
              v-if="pair.black?.comment"
              class="comment-box mt-0.5 ml-2 p-1 px-2 rounded bg-zinc-950/70 border border-zinc-800/60 text-xs text-zinc-400 font-sans italic flex items-start gap-1"
              @click="openCommentModal(pair.black.ply)"
            >
              <MessageSquare class="w-3 h-3 text-sky-400 shrink-0 mt-0.5" />
              <span>{{ pair.black.comment }}</span>
            </div>

            <!-- Black Variations (if any) -->
            <div
              v-if="pair.black?.variations && pair.black.variations.length > 0"
              class="variations-container mt-1 ml-4 pl-2 border-l-2 border-zinc-700/60 space-y-1 font-sans text-xs"
            >
              <div
                v-for="(vLine, vIdx) in pair.black.variations"
                :key="vLine.id || vIdx"
                class="variation-line flex items-center gap-1 text-zinc-400 bg-zinc-950/40 p-1 rounded"
              >
                <span class="text-zinc-500 font-mono text-[10px]">({{ vIdx + 1 }})</span>
                <span class="text-zinc-300">
                  <span
                    v-for="vm in vLine.moves"
                    :key="vm.ply"
                    class="hover:text-emerald-400 cursor-pointer mr-1"
                    :class="vm.ply === currentPly ? 'text-emerald-400 font-bold' : ''"
                    @click="emit('jump', vm.ply)"
                  >
                    {{ vm.isWhite ? `${vm.moveNumber}. ` : '' }}{{ formatMove(vm) }}
                  </span>
                </span>

                <div class="ml-auto flex items-center gap-1">
                  <button
                    class="p-0.5 rounded hover:text-emerald-300 hover:bg-zinc-800 transition-colors"
                    title="Promote variation to main line"
                    @click.stop="emit('promoteVariation', pair.black!.ply, vIdx)"
                  >
                    <ArrowUpCircle class="w-3.5 h-3.5" />
                  </button>
                  <button
                    class="p-0.5 rounded hover:text-rose-400 hover:bg-zinc-800 transition-colors"
                    title="Delete variation"
                    @click.stop="emit('deleteVariation', pair.black!.ply, vIdx)"
                  >
                    <Trash2 class="w-3.5 h-3.5" />
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Result at the end -->
      <div v-if="result && result !== '*'" class="text-center font-bold text-zinc-300 py-3 tracking-widest text-sm border-t border-zinc-800/60 mt-3">
        {{ result }}
      </div>
    </div>

    <!-- Move Comment Modal -->
    <div
      v-if="showCommentModal"
      class="comment-modal-overlay fixed inset-0 z-50 bg-black/60 backdrop-blur-sm flex items-center justify-center p-4"
      @click.self="showCommentModal = false"
    >
      <div class="comment-modal bg-zinc-900 border border-zinc-700/80 rounded-2xl w-full max-w-md shadow-2xl p-4 space-y-4">
        <div class="flex items-center justify-between border-b border-zinc-800 pb-2">
          <div class="flex items-center gap-2">
            <MessageSquare class="w-4 h-4 text-sky-400" />
            <h3 class="text-sm font-bold text-zinc-100">
              Move {{ findMoveByPly(editingCommentPly || 0)?.moveNumber }}{{ findMoveByPly(editingCommentPly || 0)?.isWhite ? '.' : '...' }} {{ formatMove(findMoveByPly(editingCommentPly || 0)) }} Comment
            </h3>
          </div>
          <button
            class="text-zinc-400 hover:text-white p-1 rounded-lg hover:bg-zinc-800 transition-colors"
            @click="showCommentModal = false"
          >
            <X class="w-4 h-4" />
          </button>
        </div>

        <div>
          <textarea
            v-model="commentInput"
            rows="4"
            class="w-full bg-zinc-950 border border-zinc-700 rounded-xl p-3 text-sm text-zinc-100 placeholder-zinc-500 focus:outline-none focus:ring-2 focus:ring-sky-500/50 resize-none font-sans"
            placeholder="Type move commentary, strategic thoughts, or tactical notes..."
            autofocus
            @keydown.ctrl.enter="saveComment"
          />
          <div class="text-[11px] text-zinc-500 mt-1 flex justify-between">
            <span>Press Ctrl+Enter to save</span>
            <span>{{ commentInput.length }} chars</span>
          </div>
        </div>

        <div class="flex items-center justify-between pt-2 border-t border-zinc-800">
          <button
            v-if="findMoveByPly(editingCommentPly || 0)?.comment"
            class="px-3 py-1.5 rounded-lg text-xs font-semibold text-rose-400 hover:bg-rose-950/60 transition-colors flex items-center gap-1"
            @click="removeComment"
          >
            <Trash2 class="w-3.5 h-3.5" />
            <span>Delete</span>
          </button>
          <div v-else />

          <div class="flex items-center gap-2">
            <button
              class="px-3 py-1.5 rounded-lg text-xs font-semibold text-zinc-400 hover:text-white hover:bg-zinc-800 transition-colors"
              @click="showCommentModal = false"
            >
              Cancel
            </button>
            <button
              class="px-4 py-1.5 rounded-lg text-xs font-semibold bg-sky-600 hover:bg-sky-500 text-white shadow-md transition-colors flex items-center gap-1"
              @click="saveComment"
            >
              <Check class="w-3.5 h-3.5" />
              <span>Save Comment</span>
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.moves-container::-webkit-scrollbar {
  width: 6px;
}
.moves-container::-webkit-scrollbar-thumb {
  background-color: rgba(255, 255, 255, 0.1);
  border-radius: 4px;
}
</style>
