<script setup lang="ts">
import { ref, computed } from 'vue'
import { Play, Square, Cpu, ChevronDown, Check } from '@lucide/vue'

export interface EvalInfo {
  depth: number
  scoreCp: number
  mateIn: number
  nps: number
  nodes: number
  pv: string[]
  bestMove: string
}

export interface EngineOption {
  name: string
  cmd: string
  resolvedCmd?: string
  uci?: boolean
  elo?: number
  available: boolean
}

const props = withDefaults(defineProps<{
  evalInfo: EvalInfo | null
  isAnalyzing: boolean
  availableEngines?: string[]
  discoveredEngines?: EngineOption[]
  selectedEngine?: string
}>(), {
  evalInfo: null,
  isAnalyzing: false,
  availableEngines: () => [],
  discoveredEngines: () => [],
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
// Engine options list
const engineOptions = computed(() => {
  if (props.discoveredEngines && props.discoveredEngines.length > 0) {
    return props.discoveredEngines.map(e => ({
      id: e.name || e.cmd,
      name: e.name || e.cmd.split('/').pop() || 'Engine',
      available: e.available,
      uci: e.uci !== false,
      elo: e.elo,
    }))
  }
  return props.availableEngines.map(p => ({
    id: p,
    name: p.split('/').pop() || p,
    available: true,
    uci: true,
    elo: 0,
  }))
})

// Dropdown state
const showEngineDropdown = ref(false)

const activeEngineName = computed(() => {
  const found = engineOptions.value.find(e => e.id === props.selectedEngine || e.name === props.selectedEngine)
  if (found) return found.name
  if (props.selectedEngine) {
    const lastPart = props.selectedEngine.split('/').pop()
    if (lastPart) return lastPart
  }
  const first = engineOptions.value.find(e => e.available)
  if (first) return first.name
  return 'Select Engine'
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

        <!-- Custom Readable Engine Selector Dropdown -->
        <div v-if="engineOptions.length > 0" class="relative">
          <button
            class="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg bg-zinc-800 hover:bg-zinc-700/90 text-white text-xs font-semibold border border-zinc-650 transition-colors cursor-pointer shadow-xs"
            title="Click to change chess engine"
            @click="showEngineDropdown = !showEngineDropdown"
          >
            <Cpu class="w-3.5 h-3.5 text-emerald-400 shrink-0" />
            <span class="font-bold text-zinc-100 max-w-[150px] truncate">{{ activeEngineName }}</span>
            <ChevronDown class="w-3 h-3 text-zinc-400 shrink-0 ml-0.5" />
          </button>

          <!-- Backdrop -->
          <div
            v-if="showEngineDropdown"
            class="fixed inset-0 z-40"
            @click="showEngineDropdown = false"
          />

          <!-- Popover Menu (opens upward from engine panel) -->
          <div
            v-if="showEngineDropdown"
            class="absolute bottom-full left-0 mb-1.5 w-64 max-h-64 overflow-y-auto bg-zinc-900 border border-zinc-700 rounded-xl shadow-2xl p-1 z-50 text-xs divide-y divide-zinc-800"
          >
            <div class="px-2.5 py-1 text-[10px] font-bold text-zinc-400 uppercase tracking-wider bg-zinc-950/60 rounded-t-lg">
              Installed Chess Engines ({{ engineOptions.filter(e => e.available).length }})
            </div>
            <div class="py-1 space-y-0.5">
              <button
                v-for="eng in engineOptions"
                :key="eng.id"
                class="w-full text-left px-2.5 py-1.5 rounded-lg flex items-center justify-between transition-colors group cursor-pointer"
                :class="[
                  !eng.available ? 'opacity-40 cursor-not-allowed' : 'hover:bg-zinc-800 text-zinc-200',
                  (selectedEngine === eng.id || selectedEngine === eng.name) ? 'bg-emerald-950/50 text-emerald-300 font-bold border border-emerald-800/40' : ''
                ]"
                :disabled="!eng.available"
                @click="emit('selectEngine', eng.id); showEngineDropdown = false"
              >
                <div class="flex items-center gap-2 min-w-0">
                  <div
                    class="w-2 h-2 rounded-full shrink-0"
                    :class="eng.available ? 'bg-emerald-400 shadow-[0_0_6px_rgba(52,211,153,0.8)]' : 'bg-zinc-600'"
                  />
                  <div class="flex flex-col min-w-0">
                    <span class="truncate text-xs font-medium text-zinc-100 group-hover:text-emerald-300">
                      {{ eng.name }}
                    </span>
                    <span v-if="!eng.available" class="text-[9px] text-zinc-500">binary not found</span>
                  </div>
                </div>
                <div class="flex items-center gap-1.5 shrink-0">
                  <span v-if="eng.elo" class="text-[10px] font-mono text-zinc-400">[{{ eng.elo }}]</span>
                  <Check
                    v-if="selectedEngine === eng.id || selectedEngine === eng.name"
                    class="w-3.5 h-3.5 text-emerald-400 shrink-0"
                  />
                </div>
              </button>
            </div>
          </div>
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
