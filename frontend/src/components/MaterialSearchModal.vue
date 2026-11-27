<script setup lang="ts">
import { ref } from 'vue'
import { X, Search, Layers, Sparkles } from '@lucide/vue'
import {
  searchMaterialAndStructure,
  defaultPieceFilter,
  type MaterialPattern,
  type MaterialSearchHit,
} from '../utils/v21Api'

const props = defineProps<{
  show: boolean
  dbName?: string
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'loadGameAtPly', gameId: number, ply: number): void
}>()

const pattern = ref<MaterialPattern>('opposite_bishops')
const minPly = ref(12)
const minSacrificeCp = ref(250)
const maxResults = ref(100)
const loading = ref(false)
const errorMsg = ref('')
const hits = ref<MaterialSearchHit[]>([])
const searched = ref(false)

const presets: { id: MaterialPattern; label: string; desc: string }[] = [
  { id: 'opposite_bishops', label: 'Opposite-Colored Bishops', desc: 'Pure opposite-color bishop endgames' },
  { id: 'same_bishops', label: 'Same-Colored Bishops', desc: 'Pure same-color bishop endgames' },
  { id: 'queen_vs_rook_minor', label: 'Queen vs. Rook + Minor', desc: 'Asymmetric queen imbalances' },
  { id: 'exchange_imbalance', label: 'Exchange Imbalance', desc: 'Rook vs. minor piece battles' },
  { id: 'two_bishops', label: 'Bishop Pair', desc: 'Two bishops vs. knight/bishop' },
  { id: 'iqp', label: 'Isolated Queen Pawn (IQP)', desc: 'Classic d4/d5 isolated pawn positions' },
  { id: 'carlsbad', label: 'Carlsbad Structure', desc: 'Minority attack pawn skeleton' },
  { id: 'maroczy_bind', label: 'Maroczy Bind', desc: 'c4 + e4 central clamp' },
  { id: 'winning_sacrifice', label: 'Winning Sacrifices', desc: 'Side drops material and still wins' },
]

async function runSearch() {
  loading.value = true
  errorMsg.value = ''
  searched.value = false
  try {
    const res = await searchMaterialAndStructure({
      pattern: pattern.value,
      white: defaultPieceFilter(),
      black: defaultPieceFilter(),
      usePieceCounts: false,
      minPly: minPly.value,
      maxPly: 0,
      minSacrificeCp: minSacrificeCp.value,
      maxResults: maxResults.value,
      maxGamesToScan: 5000,
    })
    hits.value = res || []
    searched.value = true
  } catch (err: any) {
    errorMsg.value = err?.message || String(err)
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div v-if="show" class="fixed inset-0 z-50 bg-black/70 backdrop-blur-sm flex items-center justify-center p-4">
    <div class="bg-zinc-900 border border-zinc-700 rounded-xl shadow-2xl w-full max-w-4xl max-h-[86vh] flex flex-col overflow-hidden">
      <div class="flex items-center justify-between px-5 py-3.5 border-b border-zinc-800 bg-zinc-950/60">
        <div class="flex items-center gap-2.5">
          <Layers class="w-5 h-5 text-cyan-400" />
          <div>
            <h2 class="text-sm font-bold text-zinc-100">Material &amp; Pawn-Structure Search</h2>
            <p class="text-[11px] text-zinc-400">
              Parallel multi-core pattern scanner{{ dbName ? ` — ${dbName}` : '' }}
            </p>
          </div>
        </div>
        <button class="p-1.5 rounded hover:bg-zinc-800 text-zinc-400 hover:text-zinc-100 cursor-pointer" @click="emit('close')">
          <X class="w-4 h-4" />
        </button>
      </div>

      <div class="p-5 overflow-y-auto space-y-4">
        <div class="grid grid-cols-2 sm:grid-cols-3 gap-2">
          <button
            v-for="p in presets"
            :key="p.id"
            class="text-left p-2.5 rounded-lg border transition-colors cursor-pointer"
            :class="pattern === p.id
              ? 'bg-cyan-500/15 border-cyan-500/60 text-cyan-200'
              : 'bg-zinc-950/50 border-zinc-800 text-zinc-300 hover:border-zinc-700'"
            @click="pattern = p.id"
          >
            <div class="text-xs font-semibold">{{ p.label }}</div>
            <div class="text-[10px] text-zinc-400 mt-0.5">{{ p.desc }}</div>
          </button>
        </div>

        <div class="flex flex-wrap items-end gap-4 bg-zinc-950/40 border border-zinc-800 rounded-lg p-3">
          <label class="text-xs text-zinc-400 flex flex-col gap-1">
            Min Ply
            <input v-model.number="minPly" type="number" min="1" max="120" class="w-24 bg-zinc-900 border border-zinc-700 rounded px-2 py-1 text-zinc-100 text-xs" />
          </label>
          <label v-if="pattern === 'winning_sacrifice'" class="text-xs text-zinc-400 flex flex-col gap-1">
            Min Sacrifice (cp)
            <input v-model.number="minSacrificeCp" type="number" step="50" min="100" max="1200" class="w-28 bg-zinc-900 border border-zinc-700 rounded px-2 py-1 text-zinc-100 text-xs" />
          </label>
          <label class="text-xs text-zinc-400 flex flex-col gap-1">
            Max Results
            <input v-model.number="maxResults" type="number" min="10" max="500" class="w-24 bg-zinc-900 border border-zinc-700 rounded px-2 py-1 text-zinc-100 text-xs" />
          </label>
          <button
            class="ml-auto px-4 py-1.5 rounded-lg bg-cyan-600 hover:bg-cyan-500 disabled:opacity-50 text-white text-xs font-semibold flex items-center gap-1.5 cursor-pointer"
            :disabled="loading"
            @click="runSearch"
          >
            <Search class="w-3.5 h-3.5" />
            {{ loading ? 'Scanning...' : 'Scan Database' }}
          </button>
        </div>

        <div v-if="errorMsg" class="text-xs text-red-400 bg-red-950/40 border border-red-800/50 rounded-lg p-3">
          {{ errorMsg }}
        </div>

        <div v-if="searched" class="space-y-2">
          <div class="text-xs text-zinc-400 flex items-center justify-between">
            <span>Found <strong class="text-zinc-100">{{ hits.length }}</strong> matching game(s)</span>
            <span class="text-[11px] text-zinc-500">Click any game to jump directly to the matching ply</span>
          </div>

          <div v-if="hits.length === 0" class="text-xs text-zinc-500 text-center py-8 border border-dashed border-zinc-800 rounded-lg">
            No games matched this motif in the currently open database.
          </div>

          <div v-else class="border border-zinc-800 rounded-lg overflow-hidden divide-y divide-zinc-800 max-h-72 overflow-y-auto">
            <button
              v-for="hit in hits"
              :key="`${hit.gameId}-${hit.matchPly}`"
              class="w-full text-left px-3.5 py-2.5 hover:bg-zinc-800/70 flex items-center justify-between gap-4 transition-colors cursor-pointer"
              @click="emit('loadGameAtPly', hit.gameId, hit.matchPly); emit('close')"
            >
              <div class="min-w-0">
                <div class="text-xs font-semibold text-zinc-100 truncate">
                  #{{ hit.gameId }} · {{ hit.white }} vs {{ hit.black }}
                  <span class="text-zinc-400 font-normal">({{ hit.result }})</span>
                </div>
                <div class="text-[11px] text-cyan-300/90 flex items-center gap-1.5 mt-0.5">
                  <Sparkles class="w-3 h-3 shrink-0" />
                  <span>Move {{ Math.ceil(hit.matchPly / 2) }} — {{ hit.summary }}</span>
                </div>
              </div>
              <span class="text-[11px] text-zinc-400 font-mono shrink-0">{{ hit.date || hit.eco }}</span>
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
