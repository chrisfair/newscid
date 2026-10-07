<script setup lang="ts">
import { computed } from 'vue'
import { Play, Square, Cpu } from '@lucide/vue'

export interface EvalInfo {
  depth: number
  scoreCp: number
  mateIn: number
  nps: number
  nodes: number
  pv: string[]
  bestMove: string
}

const props = withDefaults(defineProps<{
  evalInfo: EvalInfo | null
  isAnalyzing: boolean
  availableEngines?: string[]
  selectedEngine?: string
}>(), {
  evalInfo: null,
  isAnalyzing: false,
  availableEngines: () => [],
  selectedEngine: '',
})

const emit = defineEmits<{
  (e: 'toggleAnalysis'): void
  (e: 'selectEngine', path: string): void
}>()

// Calculate score string (+1.45, -0.80, M3, -M2)
const formattedScore = computed(() => {
  if (!props.evalInfo) return '0.0'
  if (props.evalInfo.mateIn !== 0) {
    return props.evalInfo.mateIn > 0
      ? `M${props.evalInfo.mateIn}`
      : `-M${Math.abs(props.evalInfo.mateIn)}`
  }
  const score = props.evalInfo.scoreCp / 100.0
  return score > 0 ? `+${score.toFixed(2)}` : score.toFixed(2)
})

// Evaluation bar percentage (0% = Black winning, 50% = equal, 100% = White winning)
const evalBarPercent = computed(() => {
  if (!props.evalInfo) return 50
  if (props.evalInfo.mateIn !== 0) {
    return props.evalInfo.mateIn > 0 ? 98 : 2
  }
  // Sigmoid-like scale for centipawns (-1000 to +1000)
  const cp = props.evalInfo.scoreCp
  const clamped = Math.max(-1000, Math.min(1000, cp))
  return 50 + (clamped / 1000) * 45
})

// PV line string
const pvLineString = computed(() => {
  if (!props.evalInfo || !props.evalInfo.pv || props.evalInfo.pv.length === 0) {
    return 'Waiting for engine...'
  }
  return props.evalInfo.pv.slice(0, 10).join(' ')
})
</script>

<template>
  <div class="engine-panel bg-zinc-900 border border-zinc-800 rounded-xl p-3 shadow-lg flex flex-col gap-2">
    <!-- Top Row: Engine controls & Evaluation score badge -->
    <div class="flex items-center justify-between">
      <div class="flex items-center gap-2">
        <button
          class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg font-medium text-xs transition-colors cursor-pointer"
          :class="isAnalyzing
            ? 'bg-amber-600/30 text-amber-300 border border-amber-500/50 hover:bg-amber-600/40'
            : 'bg-emerald-600/30 text-emerald-300 border border-emerald-500/50 hover:bg-emerald-600/40'"
          @click="emit('toggleAnalysis')"
        >
          <Square v-if="isAnalyzing" class="w-3.5 h-3.5 fill-current" />
          <Play v-else class="w-3.5 h-3.5 fill-current" />
          <span>{{ isAnalyzing ? 'Stop' : 'Analyze' }}</span>
        </button>

        <div v-if="availableEngines.length > 1" class="flex items-center gap-1">
          <Cpu class="w-3.5 h-3.5 text-zinc-500" />
          <select
            :value="selectedEngine"
            class="bg-zinc-800 border border-zinc-700 text-zinc-300 text-xs rounded px-1.5 py-1"
            @change="emit('selectEngine', ($event.target as HTMLSelectElement).value)"
          >
            <option v-for="eng in availableEngines" :key="eng" :value="eng">
              {{ eng.split('/').pop() }}
            </option>
          </select>
        </div>
      </div>

      <!-- Score Badge -->
      <div class="flex items-center gap-2">
        <span class="text-xs text-zinc-500 font-mono">D: {{ evalInfo?.depth || 0 }}</span>
        <span
          class="px-2.5 py-1 rounded font-mono font-bold text-sm tracking-wider shadow-inner"
          :class="(evalInfo?.scoreCp || 0) >= 0 ? 'bg-zinc-200 text-zinc-900' : 'bg-zinc-950 text-zinc-100 border border-zinc-800'"
        >
          {{ formattedScore }}
        </span>
      </div>
    </div>

    <!-- Visual Eval Bar -->
    <div class="eval-bar-wrapper h-2 w-full bg-zinc-950 rounded-full overflow-hidden border border-zinc-800 flex shadow-inner">
      <div
        class="eval-bar-fill h-full bg-zinc-100 transition-all duration-300 ease-out"
        :style="{ width: `${evalBarPercent}%` }"
      />
    </div>

    <!-- PV / Principal Variation line -->
    <div class="pv-container bg-zinc-950/70 border border-zinc-800/60 rounded-lg p-2 font-mono text-xs text-zinc-300 truncate">
      <span class="text-zinc-500 font-semibold mr-1.5">PV:</span>
      <span class="text-emerald-400 select-all">{{ pvLineString }}</span>
    </div>
  </div>
</template>
