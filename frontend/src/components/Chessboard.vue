<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { getPieceSVG } from '../utils/pieces'
import { playMoveSound, playCaptureSound, playCheckSound, playCastleSound } from '../utils/sound'

export interface BoardArrow {
  from: string
  to: string
  color?: string // '#10b981' (green), '#ef4444' (red), '#3b82f6' (blue), '#eab308' (yellow)
  style?: 'solid' | 'dashed'
}

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
  arrows?: BoardArrow[]
  threatSquares?: string[]
  blindfold?: boolean
  showEvalBar?: boolean
  evalScore?: number | null // centipawns
  evalMate?: number | null  // moves to mate
  boardSize?: number | null // dynamic responsive board dimension
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
  arrows: () => [],
  threatSquares: () => [],
  blindfold: false,
  showEvalBar: false,
  evalScore: null,
  evalMate: null,
  boardSize: null,
})

const emit = defineEmits<{
  (e: 'move', uci: string): void
}>()

const selectedSquare = ref<string | null>(null)
const promoPending = ref<{ from: string; to: string; color: 'w' | 'b' } | null>(null)

// Custom right-click user arrows and highlights (PyChess / Lichess style)
const customArrows = ref<BoardArrow[]>([])
const customHighlightedSquares = ref<Set<string>>(new Set())
const rightClickStartSquare = ref<string | null>(null)

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

// Count total pieces on board for audio capture detection
const pieceCount = computed(() => Object.keys(boardMap.value).length)

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

const threatSquaresSet = computed(() => new Set(props.threatSquares))

// Merged list of props.arrows + custom right-click drawn arrows
const allArrows = computed(() => {
  return [...props.arrows, ...customArrows.value]
})

// Evaluation bar calculation
const evalWhitePercent = computed(() => {
  if (props.evalMate !== null && props.evalMate !== undefined) {
    return props.evalMate > 0 ? 100 : 0
  }
  if (props.evalScore === null || props.evalScore === undefined) {
    return 50
  }
  const cp = props.evalScore
  // Sigmoid formula: 50 + 50 * (2 / (1 + exp(-0.0036 * cp)) - 1)
  const val = 50 + 50 * (2 / (1 + Math.exp(-0.0036 * cp)) - 1)
  return Math.max(5, Math.min(95, val))
})

const evalLabel = computed(() => {
  if (props.evalMate !== null && props.evalMate !== undefined) {
    return `M${Math.abs(props.evalMate)}`
  }
  if (props.evalScore === null || props.evalScore === undefined) {
    return ''
  }
  const val = props.evalScore / 100
  return val > 0 ? `+${val.toFixed(1)}` : val.toFixed(1)
})

function getSquareCoords(sq: string): { x: number; y: number } {
  const f = sq.charCodeAt(0) - 'a'.charCodeAt(0)
  const r = parseInt(sq[1], 10) - 1

  const col = props.flipped ? (7 - f) : f
  const row = props.flipped ? r : (7 - r)

  return {
    x: (col + 0.5) * 12.5,
    y: (row + 0.5) * 12.5,
  }
}

function getSquareTarget(fromSq: string, toSq: string): { x: number; y: number } {
  const start = getSquareCoords(fromSq)
  const end = getSquareCoords(toSq)

  const dx = end.x - start.x
  const dy = end.y - start.y
  const dist = Math.sqrt(dx * dx + dy * dy)
  if (dist === 0) return end

  // Back off arrowhead slightly from center of target square
  const offset = 3.6
  return {
    x: end.x - (dx / dist) * offset,
    y: end.y - (dy / dist) * offset,
  }
}

function getMarkerId(color?: string): string {
  if (!color) return 'green'
  const c = color.toLowerCase()
  if (c.includes('ef4444') || c.includes('red')) return 'red'
  if (c.includes('3b82f6') || c.includes('blue')) return 'blue'
  if (c.includes('eab308') || c.includes('yellow')) return 'yellow'
  return 'green'
}

function onSquareClick(sq: SquareInfo) {
  // Left click clears custom drawn arrows and square highlights
  if (customArrows.value.length > 0 || customHighlightedSquares.value.size > 0) {
    customArrows.value = []
    customHighlightedSquares.value = new Set()
  }

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

function onSquareMouseDown(e: MouseEvent, sq: SquareInfo) {
  if (e.button === 2) {
    // Right click start
    rightClickStartSquare.value = sq.square
  }
}

function onSquareMouseUp(e: MouseEvent, sq: SquareInfo) {
  if (e.button === 2 && rightClickStartSquare.value) {
    const from = rightClickStartSquare.value
    const to = sq.square
    rightClickStartSquare.value = null

    if (from === to) {
      // Toggle highlight on square
      const next = new Set(customHighlightedSquares.value)
      if (next.has(from)) {
        next.delete(from)
      } else {
        next.add(from)
      }
      customHighlightedSquares.value = next
    } else {
      // Toggle arrow
      const existingIdx = customArrows.value.findIndex(a => a.from === from && a.to === to)
      if (existingIdx >= 0) {
        customArrows.value.splice(existingIdx, 1)
      } else {
        customArrows.value.push({ from, to, color: '#eab308' })
      }
    }
  }
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

  emit('move', `${from}${to}`)
}

function confirmPromotion(promoPiece: string) {
  if (!promoPending.value) return
  const { from, to } = promoPending.value
  emit('move', `${from}${to}${promoPiece.toLowerCase()}`)
  promoPending.value = null
}

function onDragStart(e: DragEvent, sq: SquareInfo) {
  if (promoPending.value) {
    e.preventDefault()
    return
  }
  if (!sq.piece || !sq.piece.startsWith(turn.value)) {
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

// Sound effects on position change
let prevPieceCount = 0
let prevFEN = ''

watch(() => props.fen, (newFEN) => {
  selectedSquare.value = null
  promoPending.value = null

  if (prevFEN && prevFEN !== newFEN) {
    const currentCount = pieceCount.value
    if (props.inCheck) {
      playCheckSound()
    } else if (currentCount < prevPieceCount) {
      playCaptureSound()
    } else if (props.lastMove && (
      (props.lastMove.from === 'e1' && (props.lastMove.to === 'g1' || props.lastMove.to === 'c1')) ||
      (props.lastMove.from === 'e8' && (props.lastMove.to === 'g8' || props.lastMove.to === 'c8'))
    )) {
      playCastleSound()
    } else {
      playMoveSound()
    }
  }

  prevFEN = newFEN
  prevPieceCount = pieceCount.value
}, { immediate: true })
</script>

<template>
  <div
    class="chessboard-wrapper isolate flex items-stretch gap-2.5 select-none justify-center"
    :class="{ 'shrink-0': !!boardSize, 'w-full h-full max-w-full max-h-full': !boardSize }"
    :style="{
      width: boardSize ? `${boardSize + (showEvalBar ? 24 : 0)}px` : undefined,
      height: boardSize ? `${boardSize}px` : undefined,
    }"
    @contextmenu.prevent
  >
    <!-- PyChess-style Vertical Evaluation Gauge -->
    <div
      v-if="showEvalBar"
      class="eval-bar w-3.5 flex flex-col justify-between bg-zinc-950 rounded-sm overflow-hidden border border-zinc-700/80 relative shadow-inner select-none transition-all duration-300"
      :style="{ height: boardSize ? `${boardSize}px` : '100%' }"
      :title="`Evaluation: ${evalLabel}`"
    >
      <div
        class="eval-black w-full bg-zinc-900 transition-all duration-300 relative flex items-start justify-center"
        :style="{ height: `${100 - evalWhitePercent}%` }"
      >
        <span v-if="evalWhitePercent < 50" class="text-[8px] font-mono font-bold text-zinc-300 mt-1 select-none">
          {{ evalLabel }}
        </span>
      </div>
      <div
        class="eval-white w-full bg-zinc-100 transition-all duration-300 relative flex items-end justify-center"
        :style="{ height: `${evalWhitePercent}%` }"
      >
        <span v-if="evalWhitePercent >= 50" class="text-[8px] font-mono font-bold text-zinc-900 mb-1 select-none">
          {{ evalLabel }}
        </span>
      </div>
    </div>

    <!-- Main Chessboard Container -->
    <div
      class="chessboard-container relative select-none flex items-center justify-center"
      :class="{ 'shrink-0': !!boardSize, 'w-full h-full max-w-full max-h-full aspect-square': !boardSize }"
      :style="{
        width: boardSize ? `${boardSize}px` : undefined,
        height: boardSize ? `${boardSize}px` : undefined,
      }"
    >
      <div
        class="board-grid relative"
        :style="{
          '--light-color': boardTheme.light,
          '--dark-color': boardTheme.dark,
          '--highlight-color': boardTheme.highlight,
          '--board-size': boardSize ? `${boardSize}px` : undefined,
          width: boardSize ? `${boardSize}px` : '100%',
          height: boardSize ? `${boardSize}px` : '100%',
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
            'threat-square': threatSquaresSet.has(sq.square),
            'custom-mark-square': customHighlightedSquares.has(sq.square),
          }"
          @click="onSquareClick(sq)"
          @mousedown="onSquareMouseDown($event, sq)"
          @mouseup="onSquareMouseUp($event, sq)"
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

          <!-- Piece Image (Normal mode) -->
          <img
            v-if="sq.piece && !blindfold"
            :src="getPieceSVG(pieceSet, sq.piece[0] as 'w' | 'b', sq.piece[1])"
            :alt="sq.piece"
            class="piece-img w-[88%] h-[88%] object-contain select-none z-10 filter drop-shadow-sm active:scale-105 transition-transform"
            draggable="true"
            @dragstart="onDragStart($event, sq)"
          />

          <!-- Blindfold Mode Marker -->
          <div
            v-else-if="sq.piece && blindfold"
            class="blindfold-dot w-2.5 h-2.5 rounded-full opacity-20 pointer-events-none z-10"
            :class="sq.piece[0] === 'w' ? 'bg-white' : 'bg-black'"
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

        <!-- Interactive SVG Arrows Overlay (Engine moves, hints, threats, and user drawings) -->
        <svg
          v-if="allArrows.length > 0"
          class="board-arrows-overlay absolute inset-0 pointer-events-none z-25 w-full h-full"
          viewBox="0 0 100 100"
        >
          <defs>
            <marker id="arrow-green" viewBox="0 0 10 10" refX="6" refY="5" markerWidth="6" markerHeight="6" orient="auto-start-reverse">
              <path d="M 0 1.5 L 9 5 L 0 8.5 z" fill="#10b981" />
            </marker>
            <marker id="arrow-red" viewBox="0 0 10 10" refX="6" refY="5" markerWidth="6" markerHeight="6" orient="auto-start-reverse">
              <path d="M 0 1.5 L 9 5 L 0 8.5 z" fill="#ef4444" />
            </marker>
            <marker id="arrow-blue" viewBox="0 0 10 10" refX="6" refY="5" markerWidth="6" markerHeight="6" orient="auto-start-reverse">
              <path d="M 0 1.5 L 9 5 L 0 8.5 z" fill="#3b82f6" />
            </marker>
            <marker id="arrow-yellow" viewBox="0 0 10 10" refX="6" refY="5" markerWidth="6" markerHeight="6" orient="auto-start-reverse">
              <path d="M 0 1.5 L 9 5 L 0 8.5 z" fill="#eab308" />
            </marker>
          </defs>
          <line
            v-for="(arrow, idx) in allArrows"
            :key="idx"
            :x1="getSquareCoords(arrow.from).x"
            :y1="getSquareCoords(arrow.from).y"
            :x2="getSquareTarget(arrow.from, arrow.to).x"
            :y2="getSquareTarget(arrow.from, arrow.to).y"
            :stroke="arrow.color || '#10b981'"
            stroke-width="2.3"
            stroke-linecap="round"
            :stroke-dasharray="arrow.style === 'dashed' ? '3,2' : undefined"
            :marker-end="`url(#arrow-${getMarkerId(arrow.color)})`"
            opacity="0.85"
          />
        </svg>

        <!-- Promotion Modal Overlay -->
        <div
          v-if="promoPending"
          class="promo-overlay absolute inset-0 bg-black/60 backdrop-blur-xs flex items-center justify-center z-30 animate-fade-in"
        >
          <div class="promo-dialog bg-zinc-900 border border-zinc-700 p-3 rounded-xl shadow-2xl flex gap-2">
            <button
              v-for="p in ['Q', 'R', 'B', 'N']"
              :key="p"
              class="w-14 h-14 bg-zinc-800 hover:bg-zinc-700 active:scale-95 rounded-lg flex items-center justify-center border border-zinc-600 transition-all p-1"
              @click="confirmPromotion(p)"
            >
              <img
                :src="getPieceSVG(pieceSet, promoPending.color, p)"
                :alt="p"
                class="w-full h-full object-contain filter drop-shadow"
              />
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.chessboard-wrapper {
  max-width: 100%;
  max-height: 100%;
}

.chessboard-container {
  max-width: 100%;
  max-height: 100%;
  aspect-ratio: 1 / 1;
}

.board-grid {
  display: grid;
  grid-template-columns: repeat(8, 1fr);
  grid-template-rows: repeat(8, 1fr);
  aspect-ratio: 1 / 1;
  width: 100%;
  height: 100%;
  max-width: 100%;
  max-height: 100%;
  border-radius: 4px;
  overflow: hidden;
  box-shadow: 0 10px 25px -5px rgba(0, 0, 0, 0.5), 0 8px 10px -6px rgba(0, 0, 0, 0.4);
}

.square {
  width: 100%;
  height: 100%;
  position: relative;
}

.bg-light {
  background-color: var(--light-color);
}

.bg-dark {
  background-color: var(--dark-color);
}

.text-light-label {
  color: var(--light-color);
}

.text-dark-label {
  color: var(--dark-color);
}

.selected-square {
  background-color: var(--highlight-color) !important;
}

.last-move-square::before {
  content: '';
  position: absolute;
  inset: 0;
  background-color: rgba(255, 235, 59, 0.35);
  pointer-events: none;
}

.check-square::before {
  content: '';
  position: absolute;
  inset: 0;
  background: radial-gradient(circle, rgba(239, 68, 68, 0.8) 0%, rgba(220, 38, 38, 0.2) 75%);
  pointer-events: none;
}

/* PyChess Threat / Spy Mode Highlight */
.threat-square::after {
  content: '';
  position: absolute;
  inset: 0;
  background: radial-gradient(circle, rgba(239, 68, 68, 0.45) 0%, rgba(239, 68, 68, 0.1) 80%);
  border: 2px dashed rgba(239, 68, 68, 0.7);
  pointer-events: none;
  animation: pulse 1.8s infinite;
}

/* User Right-Click Circle Highlight */
.custom-mark-square::after {
  content: '';
  position: absolute;
  inset: 4px;
  border-radius: 9999px;
  border: 3.5px solid rgba(234, 179, 8, 0.85);
  pointer-events: none;
}

.dest-hint {
  width: 100%;
  height: 100%;
}
</style>
