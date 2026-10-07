<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { getPieceSVG } from '../utils/pieces'
import { RotateCcw, Trash2, Check, X, ShieldAlert, Sparkles } from '@lucide/vue'

const props = defineProps<{
  initialFen: string
  pieceSet: string
  boardTheme: {
    light: string
    dark: string
    highlight: string
  }
}>()

const emit = defineEmits<{
  (e: 'apply', fen: string): void
  (e: 'close'): void
}>()

// Palette options
type PaletteItem = 'wK' | 'wQ' | 'wR' | 'wB' | 'wN' | 'wP' | 'bK' | 'bQ' | 'bR' | 'bB' | 'bN' | 'bP' | 'empty'
const selectedTool = ref<PaletteItem>('wK')

// Board state: 8 ranks x 8 files
// board[r][f] where r=0 is rank 8 (top), r=7 is rank 1 (bottom), f=0 is 'a', f=7 is 'h'
const grid = ref<(string | null)[][]>(Array.from({ length: 8 }, () => Array(8).fill(null)))

// FEN components
const activeTurn = ref<'w' | 'b'>('w')
const castleWK = ref(true)
const castleWQ = ref(true)
const castleBK = ref(true)
const castleBQ = ref(true)
const enPassant = ref('-')
const halfmove = ref(0)
const fullmove = ref(1)

const fenString = ref('')
const fenError = ref('')

// Initialize from prop
function loadFromFen(fen: string) {
  try {
    const parts = fen.trim().split(/\s+/)
    if (parts.length < 1) return

    const placement = parts[0]
    const rows = placement.split('/')
    if (rows.length !== 8) {
      fenError.value = 'Invalid rank count in FEN'
      return
    }

    const newGrid: (string | null)[][] = Array.from({ length: 8 }, () => Array(8).fill(null))
    for (let r = 0; r < 8; r++) {
      let f = 0
      for (const ch of rows[r]) {
        if (ch >= '1' && ch <= '8') {
          f += parseInt(ch, 10)
        } else {
          const color = ch === ch.toUpperCase() ? 'w' : 'b'
          newGrid[r][f] = color + ch.toUpperCase()
          f++
        }
      }
    }
    grid.value = newGrid

    activeTurn.value = parts[1] === 'b' ? 'b' : 'w'
    const castling = parts[2] || '-'
    castleWK.value = castling.includes('K')
    castleWQ.value = castling.includes('Q')
    castleBK.value = castling.includes('k')
    castleBQ.value = castling.includes('q')

    enPassant.value = parts[3] || '-'
    halfmove.value = parseInt(parts[4] || '0', 10) || 0
    fullmove.value = parseInt(parts[5] || '1', 10) || 1

    fenError.value = ''
    syncFenFromState()
  } catch (err: any) {
    fenError.value = err?.message || 'Error parsing FEN'
  }
}

function syncFenFromState() {
  let placement = ''
  for (let r = 0; r < 8; r++) {
    let emptyCount = 0
    for (let f = 0; f < 8; f++) {
      const p = grid.value[r][f]
      if (!p) {
        emptyCount++
      } else {
        if (emptyCount > 0) {
          placement += emptyCount
          emptyCount = 0
        }
        const ch = p[1]
        placement += p[0] === 'w' ? ch.toUpperCase() : ch.toLowerCase()
      }
    }
    if (emptyCount > 0) {
      placement += emptyCount
    }
    if (r < 7) placement += '/'
  }

  let castling = ''
  if (castleWK.value) castling += 'K'
  if (castleWQ.value) castling += 'Q'
  if (castleBK.value) castling += 'k'
  if (castleBQ.value) castling += 'q'
  if (!castling) castling = '-'

  fenString.value = `${placement} ${activeTurn.value} ${castling} ${enPassant.value || '-'} ${halfmove.value} ${fullmove.value}`
  validateBoard()
}

// King validation
const whiteKingsCount = computed(() => {
  let count = 0
  for (let r = 0; r < 8; r++) {
    for (let f = 0; f < 8; f++) {
      if (grid.value[r][f] === 'wK') count++
    }
  }
  return count
})

const blackKingsCount = computed(() => {
  let count = 0
  for (let r = 0; r < 8; r++) {
    for (let f = 0; f < 8; f++) {
      if (grid.value[r][f] === 'bK') count++
    }
  }
  return count
})

const isValidPosition = computed(() => {
  return whiteKingsCount.value === 1 && blackKingsCount.value === 1 && !fenError.value
})

function validateBoard() {
  if (whiteKingsCount.value === 0) {
    fenError.value = 'White king is missing'
  } else if (whiteKingsCount.value > 1) {
    fenError.value = 'White has more than one king'
  } else if (blackKingsCount.value === 0) {
    fenError.value = 'Black king is missing'
  } else if (blackKingsCount.value > 1) {
    fenError.value = 'Black has more than one king'
  } else {
    fenError.value = ''
  }
}

// Square interaction
function handleSquareClick(r: number, f: number) {
  if (selectedTool.value === 'empty') {
    grid.value[r][f] = null
  } else {
    grid.value[r][f] = selectedTool.value
  }
  syncFenFromState()
}

function handleSquareRightClick(e: MouseEvent, r: number, f: number) {
  e.preventDefault()
  grid.value[r][f] = null
  syncFenFromState()
}

function handleFenInput() {
  loadFromFen(fenString.value)
}

function setStartingPosition() {
  loadFromFen('rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1')
}

function clearBoard() {
  grid.value = Array.from({ length: 8 }, () => Array(8).fill(null))
  castleWK.value = false
  castleWQ.value = false
  castleBK.value = false
  castleBQ.value = false
  enPassant.value = '-'
  halfmove.value = 0
  fullmove.value = 1
  syncFenFromState()
}

function applyPosition() {
  if (!isValidPosition.value) return
  emit('apply', fenString.value)
  emit('close')
}

watch(() => props.initialFen, (newFen) => {
  loadFromFen(newFen || 'rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1')
}, { immediate: true })
</script>

<template>
  <div class="fixed inset-0 bg-black/75 backdrop-blur-xs flex items-center justify-center z-50 p-4 animate-fade-in select-none">
    <div class="bg-zinc-900 border border-zinc-700/80 rounded-2xl w-full max-w-4xl shadow-2xl overflow-hidden flex flex-col max-h-[92vh]">
      <!-- Header -->
      <div class="flex items-center justify-between px-5 py-3.5 bg-zinc-950 border-b border-zinc-800">
        <div class="flex items-center gap-2">
          <Sparkles class="w-5 h-5 text-purple-400" />
          <h2 class="text-sm font-bold text-zinc-100">Setup Board & Position Editor</h2>
          <span class="text-[11px] text-zinc-500 font-mono">Scid vs. PC Parity</span>
        </div>
        <button
          class="text-zinc-400 hover:text-white p-1 rounded-lg hover:bg-zinc-800 transition-colors"
          @click="emit('close')"
        >
          <X class="w-5 h-5" />
        </button>
      </div>

      <!-- Body Grid: Left Board + Right Controls -->
      <div class="p-5 flex-1 overflow-y-auto flex flex-col md:flex-row gap-6">
        <!-- Left: 8x8 Board & Piece Palette -->
        <div class="flex flex-col items-center gap-4">
          <!-- 8x8 Grid -->
          <div
            class="setup-board grid grid-cols-8 grid-rows-8 w-80 h-80 border-2 border-zinc-700 rounded-lg overflow-hidden shadow-xl"
            :style="{
              '--light-color': boardTheme.light,
              '--dark-color': boardTheme.dark,
            }"
          >
            <template v-for="r in 8" :key="r">
              <div
                v-for="f in 8"
                :key="f"
                class="setup-square relative flex items-center justify-center cursor-pointer transition-transform duration-75 hover:scale-105 z-0 hover:z-10"
                :class="(r - 1 + f - 1) % 2 === 0 ? 'bg-light' : 'bg-dark'"
                @click="handleSquareClick(r - 1, f - 1)"
                @contextmenu="handleSquareRightClick($event, r - 1, f - 1)"
              >
                <!-- Rank/File label -->
                <span
                  v-if="f === 1"
                  class="absolute top-0.5 left-1 text-[9px] font-bold opacity-50 pointer-events-none"
                  :class="(r - 1 + f - 1) % 2 === 0 ? 'text-zinc-800' : 'text-zinc-200'"
                >
                  {{ 8 - (r - 1) }}
                </span>
                <span
                  v-if="r === 8"
                  class="absolute bottom-0.5 right-1 text-[9px] font-bold opacity-50 pointer-events-none"
                  :class="(r - 1 + f - 1) % 2 === 0 ? 'text-zinc-800' : 'text-zinc-200'"
                >
                  {{ String.fromCharCode('a'.charCodeAt(0) + f - 1) }}
                </span>

                <!-- Placed Piece -->
                <img
                  v-if="grid[r - 1][f - 1]"
                  :src="getPieceSVG(pieceSet, grid[r - 1][f - 1]![0] as 'w' | 'b', grid[r - 1][f - 1]![1])"
                  :alt="grid[r - 1][f - 1]!"
                  class="w-[85%] h-[85%] object-contain pointer-events-none drop-shadow-sm"
                />
              </div>
            </template>
          </div>

          <!-- Piece Palette -->
          <div class="flex flex-col gap-2 bg-zinc-950 p-2.5 rounded-xl border border-zinc-800 w-full">
            <span class="text-[11px] font-semibold text-zinc-400 uppercase tracking-wider text-center">Piece Palette (Click tool, then click square)</span>
            
            <!-- White Pieces -->
            <div class="flex justify-center gap-1.5">
              <button
                v-for="p in (['wK', 'wQ', 'wR', 'wB', 'wN', 'wP'] as PaletteItem[])"
                :key="p"
                class="w-10 h-10 rounded-lg flex items-center justify-center p-1 border transition-all cursor-pointer"
                :class="selectedTool === p ? 'bg-purple-600/30 border-purple-500 shadow-md ring-2 ring-purple-500/30' : 'bg-zinc-900 border-zinc-800 hover:bg-zinc-800'"
                @click="selectedTool = p"
              >
                <img :src="getPieceSVG(pieceSet, 'w', p[1])" class="w-full h-full object-contain" />
              </button>
            </div>

            <!-- Black Pieces -->
            <div class="flex justify-center gap-1.5">
              <button
                v-for="p in (['bK', 'bQ', 'bR', 'bB', 'bN', 'bP'] as PaletteItem[])"
                :key="p"
                class="w-10 h-10 rounded-lg flex items-center justify-center p-1 border transition-all cursor-pointer"
                :class="selectedTool === p ? 'bg-purple-600/30 border-purple-500 shadow-md ring-2 ring-purple-500/30' : 'bg-zinc-900 border-zinc-800 hover:bg-zinc-800'"
                @click="selectedTool = p"
              >
                <img :src="getPieceSVG(pieceSet, 'b', p[1])" class="w-full h-full object-contain" />
              </button>
            </div>

            <!-- Eraser / Clear tool -->
            <div class="flex justify-center mt-1">
              <button
                class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg border text-xs font-medium transition-all cursor-pointer"
                :class="selectedTool === 'empty' ? 'bg-rose-600/30 border-rose-500 text-rose-300 ring-2 ring-rose-500/30' : 'bg-zinc-900 border-zinc-800 text-zinc-400 hover:text-zinc-200 hover:bg-zinc-800'"
                @click="selectedTool = 'empty'"
              >
                <Trash2 class="w-3.5 h-3.5" />
                <span>Eraser Tool (or Right-Click)</span>
              </button>
            </div>
          </div>
        </div>

        <!-- Right: Parameters, Turn, Castling, FEN -->
        <div class="flex-1 flex flex-col gap-4">
          <!-- Quick Presets -->
          <div class="flex gap-2">
            <button
              class="flex-1 flex items-center justify-center gap-1.5 py-1.5 px-3 rounded-lg bg-zinc-800 hover:bg-zinc-700 text-xs font-semibold text-zinc-200 transition-colors cursor-pointer"
              @click="setStartingPosition"
            >
              <RotateCcw class="w-3.5 h-3.5 text-emerald-400" />
              <span>Standard Start Pos</span>
            </button>
            <button
              class="flex-1 flex items-center justify-center gap-1.5 py-1.5 px-3 rounded-lg bg-zinc-800 hover:bg-zinc-700 text-xs font-semibold text-rose-300 transition-colors cursor-pointer"
              @click="clearBoard"
            >
              <Trash2 class="w-3.5 h-3.5" />
              <span>Clear Board</span>
            </button>
          </div>

          <!-- Turn Selection -->
          <div class="bg-zinc-950 p-3 rounded-xl border border-zinc-800 flex flex-col gap-2">
            <span class="text-xs font-semibold text-zinc-400">Side to Move</span>
            <div class="grid grid-cols-2 gap-2">
              <button
                class="flex items-center justify-center gap-2 py-2 rounded-lg text-xs font-semibold border transition-all cursor-pointer"
                :class="activeTurn === 'w' ? 'bg-white text-zinc-950 border-white shadow-md' : 'bg-zinc-900 text-zinc-400 border-zinc-800 hover:bg-zinc-800'"
                @click="activeTurn = 'w'; syncFenFromState()"
              >
                <span class="w-2.5 h-2.5 rounded-full bg-zinc-300 border border-zinc-400" />
                <span>White to move</span>
              </button>
              <button
                class="flex items-center justify-center gap-2 py-2 rounded-lg text-xs font-semibold border transition-all cursor-pointer"
                :class="activeTurn === 'b' ? 'bg-zinc-800 text-white border-zinc-600 shadow-md' : 'bg-zinc-900 text-zinc-400 border-zinc-800 hover:bg-zinc-800'"
                @click="activeTurn = 'b'; syncFenFromState()"
              >
                <span class="w-2.5 h-2.5 rounded-full bg-zinc-900 border border-zinc-600" />
                <span>Black to move</span>
              </button>
            </div>
          </div>

          <!-- Castling Rights -->
          <div class="bg-zinc-950 p-3 rounded-xl border border-zinc-800 flex flex-col gap-2">
            <span class="text-xs font-semibold text-zinc-400">Castling Availability</span>
            <div class="grid grid-cols-2 gap-3 text-xs">
              <label class="flex items-center gap-2 text-zinc-300 cursor-pointer">
                <input
                  v-model="castleWK"
                  type="checkbox"
                  class="rounded bg-zinc-900 border-zinc-700 text-purple-600 focus:ring-purple-500"
                  @change="syncFenFromState"
                />
                <span>White O-O (Kingside)</span>
              </label>
              <label class="flex items-center gap-2 text-zinc-300 cursor-pointer">
                <input
                  v-model="castleWQ"
                  type="checkbox"
                  class="rounded bg-zinc-900 border-zinc-700 text-purple-600 focus:ring-purple-500"
                  @change="syncFenFromState"
                />
                <span>White O-O-O (Queenside)</span>
              </label>
              <label class="flex items-center gap-2 text-zinc-300 cursor-pointer">
                <input
                  v-model="castleBK"
                  type="checkbox"
                  class="rounded bg-zinc-900 border-zinc-700 text-purple-600 focus:ring-purple-500"
                  @change="syncFenFromState"
                />
                <span>Black O-O (Kingside)</span>
              </label>
              <label class="flex items-center gap-2 text-zinc-300 cursor-pointer">
                <input
                  v-model="castleBQ"
                  type="checkbox"
                  class="rounded bg-zinc-900 border-zinc-700 text-purple-600 focus:ring-purple-500"
                  @change="syncFenFromState"
                />
                <span>Black O-O-O (Queenside)</span>
              </label>
            </div>
          </div>

          <!-- En Passant, Halfmove, Fullmove -->
          <div class="grid grid-cols-3 gap-2">
            <div class="bg-zinc-950 p-2.5 rounded-xl border border-zinc-800 flex flex-col gap-1">
              <label class="text-[11px] font-semibold text-zinc-400">En Passant</label>
              <input
                v-model="enPassant"
                type="text"
                placeholder="-"
                class="bg-zinc-900 border border-zinc-700 rounded px-2 py-1 text-xs font-mono text-zinc-200 focus:outline-hidden focus:border-purple-500"
                @input="syncFenFromState"
              />
            </div>
            <div class="bg-zinc-950 p-2.5 rounded-xl border border-zinc-800 flex flex-col gap-1">
              <label class="text-[11px] font-semibold text-zinc-400">Halfmove Clock</label>
              <input
                v-model.number="halfmove"
                type="number"
                min="0"
                class="bg-zinc-900 border border-zinc-700 rounded px-2 py-1 text-xs font-mono text-zinc-200 focus:outline-hidden focus:border-purple-500"
                @input="syncFenFromState"
              />
            </div>
            <div class="bg-zinc-950 p-2.5 rounded-xl border border-zinc-800 flex flex-col gap-1">
              <label class="text-[11px] font-semibold text-zinc-400">Move Number</label>
              <input
                v-model.number="fullmove"
                type="number"
                min="1"
                class="bg-zinc-900 border border-zinc-700 rounded px-2 py-1 text-xs font-mono text-zinc-200 focus:outline-hidden focus:border-purple-500"
                @input="syncFenFromState"
              />
            </div>
          </div>

          <!-- Raw FEN Input -->
          <div class="bg-zinc-950 p-3 rounded-xl border border-zinc-800 flex flex-col gap-1.5">
            <div class="flex items-center justify-between">
              <span class="text-xs font-semibold text-zinc-400">FEN Output / Direct Input</span>
              <span v-if="isValidPosition" class="text-[11px] text-emerald-400 font-medium flex items-center gap-1">
                <Check class="w-3.5 h-3.5" /> Valid Position
              </span>
            </div>
            <textarea
              v-model="fenString"
              rows="2"
              class="w-full bg-zinc-900 border border-zinc-700 rounded-lg p-2 text-xs font-mono text-zinc-100 focus:outline-hidden focus:border-purple-500 resize-none"
              @change="handleFenInput"
            />
            <div v-if="fenError" class="flex items-center gap-1.5 text-rose-400 text-xs font-medium">
              <ShieldAlert class="w-4 h-4 shrink-0" />
              <span>{{ fenError }}</span>
            </div>
          </div>
        </div>
      </div>

      <!-- Footer Buttons -->
      <div class="px-5 py-3 bg-zinc-950 border-t border-zinc-800 flex items-center justify-between">
        <div class="text-xs text-zinc-500">
          White Kings: <span :class="whiteKingsCount === 1 ? 'text-emerald-400' : 'text-rose-400 font-bold'">{{ whiteKingsCount }}</span> |
          Black Kings: <span :class="blackKingsCount === 1 ? 'text-emerald-400' : 'text-rose-400 font-bold'">{{ blackKingsCount }}</span>
        </div>
        <div class="flex gap-2">
          <button
            class="px-4 py-2 rounded-xl bg-zinc-800 hover:bg-zinc-700 text-zinc-300 text-xs font-semibold transition-colors cursor-pointer"
            @click="emit('close')"
          >
            Cancel
          </button>
          <button
            class="px-5 py-2 rounded-xl text-xs font-semibold flex items-center gap-2 transition-all cursor-pointer shadow-lg"
            :class="isValidPosition ? 'bg-purple-600 hover:bg-purple-500 text-white shadow-purple-600/30' : 'bg-zinc-800 text-zinc-500 cursor-not-allowed'"
            :disabled="!isValidPosition"
            @click="applyPosition"
          >
            <Check class="w-4 h-4" />
            <span>Apply Position</span>
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.bg-light {
  background-color: var(--light-color);
}
.bg-dark {
  background-color: var(--dark-color);
}
</style>
