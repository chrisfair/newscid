<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { X, Dices, RotateCcw, Check, Sparkles } from '@lucide/vue'
import { GetChess960Position, GenerateRandomChess960 } from '../../wailsjs/go/main/App'
import { getPieceSVG } from '../utils/pieces'

interface Chess960Pos {
  id: number
  fen: string
  pieces: string
  isClassic: boolean
}

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'loadPosition', pos: Chess960Pos): void
}>()

const positionId = ref<number>(518)
const currentPos = ref<Chess960Pos | null>(null)
const loading = ref<boolean>(false)
const error = ref<string>('')

async function fetchPosition(id: number) {
  if (id < 0 || id > 959) return
  loading.value = true
  error.value = ''
  try {
    const res = await GetChess960Position(id)
    currentPos.value = res
    positionId.value = res.id
  } catch (err: unknown) {
    error.value = String(err)
  } finally {
    loading.value = false
  }
}

async function rollRandom() {
  loading.value = true
  error.value = ''
  try {
    const res = await GenerateRandomChess960()
    currentPos.value = res
    positionId.value = res.id
  } catch (err: unknown) {
    error.value = String(err)
  } finally {
    loading.value = false
  }
}

function handleInput() {
  let val = Math.max(0, Math.min(959, positionId.value || 0))
  positionId.value = val
  fetchPosition(val)
}

function loadIntoBoard() {
  if (!currentPos.value) return
  emit('loadPosition', currentPos.value)
  emit('close')
}

onMounted(() => {
  fetchPosition(518)
})
</script>

<template>
  <div class="fixed inset-0 bg-black/75 backdrop-blur-xs flex items-center justify-center z-50 p-4">
    <div class="bg-zinc-900 border border-zinc-800 rounded-xl shadow-2xl w-full max-w-lg overflow-hidden flex flex-col animate-fade-in">
      <!-- Modal Header -->
      <div class="flex items-center justify-between px-5 py-4 border-b border-zinc-800 bg-zinc-950/60">
        <div class="flex items-center gap-2.5">
          <div class="w-8 h-8 rounded-lg bg-amber-500/10 border border-amber-500/20 flex items-center justify-center text-amber-400">
            <Sparkles class="w-4 h-4" />
          </div>
          <div>
            <h2 class="text-base font-semibold text-zinc-100">Chess960 (Fischer Random)</h2>
            <p class="text-xs text-zinc-400">Select or generate an official FIDE Chess960 starting setup</p>
          </div>
        </div>
        <button
          class="text-zinc-400 hover:text-zinc-200 p-1.5 rounded-lg hover:bg-zinc-800 transition-colors"
          @click="emit('close')"
        >
          <X class="w-5 h-5" />
        </button>
      </div>

      <!-- Modal Body -->
      <div class="p-5 flex flex-col gap-5">
        <!-- Controls row -->
        <div class="flex items-center gap-3">
          <div class="flex-1 flex items-center gap-2 bg-zinc-950 px-3 py-2 rounded-lg border border-zinc-800">
            <span class="text-xs font-mono text-zinc-400">ID:</span>
            <input
              v-model.number="positionId"
              type="number"
              min="0"
              max="959"
              class="w-20 bg-transparent text-sm font-mono font-bold text-amber-400 focus:outline-none"
              @change="handleInput"
            />
            <span class="text-xs text-zinc-500">/ 959</span>
          </div>

          <button
            class="flex items-center gap-1.5 px-3 py-2 bg-amber-600 hover:bg-amber-500 text-zinc-950 font-semibold text-xs rounded-lg transition-colors shadow-sm"
            :disabled="loading"
            @click="rollRandom"
          >
            <Dices class="w-4 h-4" />
            <span>Random</span>
          </button>

          <button
            class="flex items-center gap-1.5 px-3 py-2 bg-zinc-800 hover:bg-zinc-700 text-zinc-300 text-xs rounded-lg transition-colors border border-zinc-700"
            :disabled="loading"
            @click="fetchPosition(518)"
          >
            <RotateCcw class="w-3.5 h-3.5" />
            <span>Classic (#518)</span>
          </button>
        </div>

        <!-- Slider -->
        <div class="flex flex-col gap-1.5">
          <div class="flex justify-between text-[11px] text-zinc-400 font-mono">
            <span>#0</span>
            <span class="text-amber-400 font-semibold">Position #{{ positionId }}</span>
            <span>#959</span>
          </div>
          <input
            v-model.number="positionId"
            type="range"
            min="0"
            max="959"
            class="w-full accent-amber-500 cursor-pointer"
            @input="handleInput"
          />
        </div>

        <!-- Rank 1 Layout Preview -->
        <div v-if="currentPos" class="flex flex-col gap-2">
          <div class="flex items-center justify-between">
            <span class="text-xs font-medium text-zinc-300">Initial Pieces Rank:</span>
            <span
              v-if="currentPos.isClassic"
              class="px-2 py-0.5 rounded text-[10px] font-bold bg-emerald-500/20 text-emerald-400 border border-emerald-500/30"
            >
              Standard Classical Layout
            </span>
          </div>

          <!-- 8-square preview bar -->
          <div class="grid grid-cols-8 gap-1 p-2 bg-zinc-950 rounded-lg border border-zinc-800">
            <div
              v-for="(piece, idx) in currentPos.pieces.split('')"
              :key="idx"
              class="aspect-square flex flex-col items-center justify-center rounded p-1"
              :class="idx % 2 === 0 ? 'bg-zinc-800/80' : 'bg-zinc-900/80'"
            >
              <img
                :src="getPieceSVG('cburnett', 'w', piece)"
                :alt="piece"
                class="w-7 h-7 object-contain drop-shadow"
              />
              <span class="text-[9px] font-mono text-zinc-400 mt-0.5">{{ String.fromCharCode(97 + idx) }}</span>
            </div>
          </div>

          <!-- FEN string box -->
          <div class="bg-zinc-950 p-2.5 rounded-lg border border-zinc-800 flex flex-col gap-1">
            <span class="text-[10px] font-mono uppercase text-zinc-500">Starting FEN:</span>
            <code class="text-xs font-mono text-zinc-300 break-all select-all">{{ currentPos.fen }}</code>
          </div>
        </div>

        <div v-if="error" class="p-3 bg-red-950/40 border border-red-800/50 rounded-lg text-xs text-red-300">
          {{ error }}
        </div>
      </div>

      <!-- Modal Footer -->
      <div class="flex items-center justify-end gap-2.5 px-5 py-3.5 border-t border-zinc-800 bg-zinc-950/60">
        <button
          class="px-3.5 py-1.5 text-xs text-zinc-400 hover:text-zinc-200 transition-colors"
          @click="emit('close')"
        >
          Cancel
        </button>
        <button
          class="flex items-center gap-1.5 px-4 py-2 bg-emerald-600 hover:bg-emerald-500 text-white text-xs font-medium rounded-lg transition-colors shadow-sm"
          :disabled="!currentPos || loading"
          @click="loadIntoBoard"
        >
          <Check class="w-4 h-4" />
          <span>Load Into Board</span>
        </button>
      </div>
    </div>
  </div>
</template>
