<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { getPieceSVG } from '../utils/pieces'

interface SquareInfo {
  square: string
  file: number
  rank: number
  isLight: boolean
  piece: string | null // e.g. "wK", "bP"
}

const props = withDefaults(defineProps<{
  fen: string
  legalMoves?: string[] // UCI strings like "e2e4", "e7e8q"
  pieceSet?: string
  boardTheme?: {
    light: string
    dark: string
    highlight: string
  }
  flipped?: boolean
  lastMove?: { from: string; to: string } | null
  inCheck?: boolean
}>(), {
  legalMoves: () => [],
  pieceSet: 'cburnett',
  boardTheme: () => ({
    light: '#e3c896',
    dark: '#ab7a4e',
    highlight: '#f7e597',
  }),
  flipped: false,
  lastMove: null,
  inCheck: false,
})

const emit = defineEmits<{
  (e: 'move', uci: string): void
}>()

const selectedSquare = ref<string | null>(null)
const promoPending = ref<{ from: string; to: string; color: 'w' | 'b' } | null>(null)

// Parse FEN into 64-square map
const boardMap = computed(() => {
  const map: Record<string, string> = {}
  const parts = props.fen.split(' ')
  const piecePlacement = parts[0] || ''
  const rows = piecePlacement.split('/')

  for (let rIdx = 0; rIdx < 8; rIdx++) {
    const rank = 7 - rIdx
    let file = 0
    const row = rows[rIdx] || ''
    for (const ch of row) {
      if (ch >= '1' && ch <= '8') {
        file += parseInt(ch, 10)
      } else {
        const sq = String.fromCharCode('a'.charCodeAt(0) + file) + (rank + 1)
        const isUpper = ch === ch.toUpperCase()
        const color = isUpper ? 'w' : 'b'
        map[sq] = color + ch.toUpperCase()
        file++
      }
    }
  }
  return map
})

// Squares ordered according to flip
const displayedSquares = computed(() => {
  const squares: SquareInfo[] = []
  const rankIndices = props.flipped ? [0, 1, 2, 3, 4, 5, 6, 7] : [7, 6, 5, 4, 3, 2, 1, 0]
  const fileIndices = props.flipped ? [7, 6, 5, 4, 3, 2, 1, 0] : [0, 1, 2, 3, 4, 5, 6, 7]

  for (const r of rankIndices) {
    for (const f of fileIndices) {
      const sq = String.fromCharCode('a'.charCodeAt(0) + f) + (r + 1)
      const isLight = (r + f) % 2 !== 0
      squares.push({
        square: sq,
        file: f,
        rank: r,
        isLight,
        piece: boardMap.value[sq] || null,
      })
    }
  }
  return squares
})

// Current turn from FEN
const turn = computed(() => {
  const parts = props.fen.split(' ')
  return parts[1] === 'b' ? 'b' : 'w'
})

// Filter legal moves from selected square
const destinationSquares = computed(() => {
  if (!selectedSquare.value) return new Set<string>()
  const set = new Set<string>()
  const prefix = selectedSquare.value
  for (const m of props.legalMoves) {
    if (m.startsWith(prefix)) {
      set.add(m.slice(2, 4))
    }
  }
  return set
})

function onSquareClick(sq: SquareInfo) {
  if (promoPending.value) return

  // If a piece from current side was clicked and nothing selected, select it
  if (!selectedSquare.value) {
    if (sq.piece && sq.piece.startsWith(turn.value)) {
      selectedSquare.value = sq.square
    }
    return
  }

  // If clicking the same square, deselect
  if (selectedSquare.value === sq.square) {
    selectedSquare.value = null
    return
  }

  // If clicking another piece of our color, reselect
  if (sq.piece && sq.piece.startsWith(turn.value)) {
    selectedSquare.value = sq.square
    return
  }

  // Check if target is a legal destination
  if (destinationSquares.value.has(sq.square)) {
    tryMakeMove(selectedSquare.value, sq.square)
  }

  selectedSquare.value = null
}

function tryMakeMove(from: string, to: string) {
  const movingPiece = boardMap.value[from]
  const isPawn = movingPiece && movingPiece.endsWith('P')
  const toRank = to[1]

  // Check for promotion
  if (isPawn && (toRank === '8' || toRank === '1')) {
    promoPending.value = { from, to, color: movingPiece.startsWith('w') ? 'w' : 'b' }
    return
  }

  // Regular move
  emit('move', `${from}${to}`)
}

function selectPromoPiece(promo: string) {
  if (!promoPending.value) return
  const { from, to } = promoPending.value
  emit('move', `${from}${to}${promo.toLowerCase()}`)
  promoPending.value = null
  selectedSquare.value = null
}

// Drag & Drop
function onDragStart(e: DragEvent, sq: SquareInfo) {
  if (!sq.piece || !sq.piece.startsWith(turn.value) || promoPending.value) {
    e.preventDefault()
    return
  }
  selectedSquare.value = sq.square
  if (e.dataTransfer) {
    e.dataTransfer.setData('text/plain', sq.square)
    e.dataTransfer.effectAllowed = 'move'
  }
}

function onDragOver(e: DragEvent) {
  e.preventDefault()
  if (e.dataTransfer) {
    e.dataTransfer.dropEffect = 'move'
  }
}

function onDrop(e: DragEvent, targetSq: SquareInfo) {
  e.preventDefault()
  const from = e.dataTransfer?.getData('text/plain')
  if (from && destinationSquares.value.has(targetSq.square)) {
    tryMakeMove(from, targetSq.square)
  }
  selectedSquare.value = null
}

// Clear selection on FEN change
watch(() => props.fen, () => {
  selectedSquare.value = null
  promoPending.value = null
})
</script>

<template>
  <div class="chessboard-container select-none">
    <div
      class="board-grid relative"
      :style="{
        '--light-color': boardTheme.light,
        '--dark-color': boardTheme.dark,
        '--highlight-color': boardTheme.highlight,
      }"
    >
      <div
        v-for="sq in displayedSquares"
        :key="sq.square"
        class="square relative flex items-center justify-center cursor-pointer transition-colors duration-75"
        :class="{
          'bg-light': sq.isLight,
          'bg-dark': !sq.isLight,
          'selected-square': selectedSquare === sq.square,
          'last-move-square': lastMove && (lastMove.from === sq.square || lastMove.to === sq.square),
          'check-square': inCheck && sq.piece === (turn + 'K'),
        }"
        @click="onSquareClick(sq)"
        @dragover="onDragOver"
        @drop="onDrop($event, sq)"
      >
        <!-- Rank coordinate (left edge) -->
        <span
          v-if="(!flipped && sq.file === 0) || (flipped && sq.file === 7)"
          class="coord rank-coord absolute top-0.5 left-1 text-[10px] font-bold select-none opacity-60 pointer-events-none"
          :class="sq.isLight ? 'text-dark-label' : 'text-light-label'"
        >
          {{ sq.rank + 1 }}
        </span>

        <!-- File coordinate (bottom edge) -->
        <span
          v-if="(!flipped && sq.rank === 0) || (flipped && sq.rank === 7)"
          class="coord file-coord absolute bottom-0.5 right-1 text-[10px] font-bold select-none opacity-60 pointer-events-none"
          :class="sq.isLight ? 'text-dark-label' : 'text-light-label'"
        >
          {{ String.fromCharCode('a'.charCodeAt(0) + sq.file) }}
        </span>

        <!-- Piece Image -->
        <img
          v-if="sq.piece"
          :src="getPieceSVG(pieceSet, sq.piece[0] as 'w' | 'b', sq.piece[1])"
          :alt="sq.piece"
          class="piece-img w-[88%] h-[88%] object-contain select-none z-10 filter drop-shadow-sm active:scale-105 transition-transform"
          draggable="true"
          @dragstart="onDragStart($event, sq)"
        />

        <!-- Legal Move Destination Hint -->
        <div
          v-if="destinationSquares.has(sq.square)"
          class="dest-hint absolute pointer-events-none z-20 flex items-center justify-center"
        >
          <!-- Small dot for empty squares -->
          <div
            v-if="!sq.piece"
            class="w-3.5 h-3.5 rounded-full bg-emerald-500/60 shadow-sm"
          />
          <!-- Ring for captures -->
          <div
            v-else
            class="w-full h-full rounded-full border-4 border-emerald-500/70"
          />
        </div>
      </div>

      <!-- Promotion Modal Overlay -->
      <div
        v-if="promoPending"
        class="promo-overlay absolute inset-0 bg-black/60 backdrop-blur-xs flex items-center justify-center z-30 animate-fade-in"
      >
        <div class="promo-dialog bg-zinc-900 border border-zinc-700 p-3 rounded-xl shadow-2xl flex gap-2">
          <button
            v-for="p in ['Q', 'R', 'B', 'N']"
            :key="p"
            class="promo-btn p-2 rounded-lg hover:bg-zinc-800 border border-transparent hover:border-emerald-500 transition-all cursor-pointer flex flex-col items-center"
            @click.stop="selectPromoPiece(p)"
          >
            <img
              :src="getPieceSVG(pieceSet, promoPending.color, p)"
              :alt="p"
              class="w-12 h-12 object-contain"
            />
            <span class="text-xs text-zinc-400 font-mono mt-1">{{ p }}</span>
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.chessboard-container {
  display: flex;
  justify-content: center;
  align-items: center;
  width: 100%;
  max-width: min(75vh, 600px);
  aspect-ratio: 1;
}

.board-grid {
  display: grid;
  grid-template-columns: repeat(8, 1fr);
  grid-template-rows: repeat(8, 1fr);
  width: 100%;
  height: 100%;
  border-radius: 6px;
  overflow: hidden;
  box-shadow: 0 10px 25px -5px rgba(0, 0, 0, 0.5), 0 8px 10px -6px rgba(0, 0, 0, 0.4);
}

.bg-light {
  background-color: var(--light-color);
}

.bg-dark {
  background-color: var(--dark-color);
}

.text-dark-label {
  color: var(--dark-color);
}

.text-light-label {
  color: var(--light-color);
}

.selected-square {
  background-color: var(--highlight-color) !important;
}

.last-move-square {
  position: relative;
}

.last-move-square::before {
  content: '';
  position: absolute;
  inset: 0;
  background-color: rgba(245, 215, 110, 0.45);
  pointer-events: none;
}

.check-square {
  background: radial-gradient(circle, rgba(239, 68, 68, 0.8) 0%, rgba(220, 38, 38, 0.4) 60%, transparent 100%) !important;
}

.dest-hint {
  inset: 0;
}
</style>
