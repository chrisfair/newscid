<script setup lang="ts">
import { ref, computed } from 'vue'
import { Play, Square, Cpu, ChevronDown, Check, Sliders, Layers } from '@lucide/vue'

export interface PVLine {
  multipv: number
  depth: number
  scoreCp: number
  mateIn: number
  nps: number
  nodes: number
  pv: string[]
  moveUci: string
  san: string
  pvFormatted: string
}

export interface EvalInfo {
  depth: number
  scoreCp: number
  mateIn: number
  nps: number
  nodes: number
  pv: string[]
  bestMove: string
  multiPv?: number
  lines?: PVLine[]
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
  multiPv?: number
}>(), {
  evalInfo: null,
  isAnalyzing: false,
  availableEngines: () => [],
  discoveredEngines: () => [],
  selectedEngine: '',
  multiPv: 1,
})

const emit = defineEmits<{
  (e: 'toggleAnalysis'): void
  (e: 'selectEngine', path: string): void
  (e: 'changeMultiPV', lines: number): void
  (e: 'changeThreads', threads: number): void
  (e: 'changeHash', mb: number): void
  (e: 'previewMove', uci: string): void
}>()

// Dropdown and settings state
const showEngineDropdown = ref(false)
const showSettingsDropdown = ref(false)
const selectedThreads = ref(1)
const selectedHash = ref(128)

// Active MultiPV count
const currentMultiPv = computed(() => {
  return props.multiPv || props.evalInfo?.multiPv || 1
})

// Format scores for any PVLine or root eval
function formatScoreVal(scoreCp: number, mateIn: number): string {
  if (mateIn !== 0) {
    return mateIn > 0 ? `M${mateIn}` : `-M${Math.abs(mateIn)}`
  }
  const score = scoreCp / 100.0
  return score > 0 ? `+${score.toFixed(2)}` : score.toFixed(2)
}

// Calculate root score string (+1.45, -0.80, M3, -M2)
const formattedScore = computed(() => {
  if (!props.evalInfo) return '0.0'
  return formatScoreVal(props.evalInfo.scoreCp, props.evalInfo.mateIn)
})

// Evaluation bar percentage (0% = Black winning, 50% = equal, 100% = White winning)
const evalBarPercent = computed(() => {
  if (!props.evalInfo) return 50
  if (props.evalInfo.mateIn !== 0) {
    return props.evalInfo.mateIn > 0 ? 98 : 2
  }
  const cp = props.evalInfo.scoreCp
  const clamped = Math.max(-1000, Math.min(1000, cp))
  return 50 + (clamped / 1000) * 45
})

// Single PV line string
const pvLineString = computed(() => {
  if (!props.evalInfo || !props.evalInfo.pv || props.evalInfo.pv.length === 0) {
    return 'Waiting for engine...'
  }
  return props.evalInfo.pv.slice(0, 10).join(' ')
})

// Sorted Multi-PV lines
const sortedLines = computed<PVLine[]>(() => {
  if (!props.evalInfo || !props.evalInfo.lines || props.evalInfo.lines.length === 0) {
    // If no lines array but root PV exists, construct line 1
    if (props.evalInfo && props.evalInfo.pv && props.evalInfo.pv.length > 0) {
      return [{
        multipv: 1,
        depth: props.evalInfo.depth,
        scoreCp: props.evalInfo.scoreCp,
        mateIn: props.evalInfo.mateIn,
        nps: props.evalInfo.nps,
        nodes: props.evalInfo.nodes,
        pv: props.evalInfo.pv,
        moveUci: props.evalInfo.pv[0],
        san: props.evalInfo.pv[0],
        pvFormatted: props.evalInfo.pv.slice(0, 10).join(' ')
      }]
    }
    return []
  }
  return [...props.evalInfo.lines].sort((a, b) => a.multipv - b.multipv)
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

function getRankBadgeStyle(rank: number) {
  switch (rank) {
    case 1:
      return 'bg-emerald-950/70 text-emerald-300 border-emerald-700/60'
    case 2:
      return 'bg-sky-950/70 text-sky-300 border-sky-700/60'
    case 3:
      return 'bg-amber-950/70 text-amber-300 border-amber-700/60'
    case 4:
      return 'bg-purple-950/70 text-purple-300 border-purple-700/60'
    case 5:
      return 'bg-rose-950/70 text-rose-300 border-rose-700/60'
    default:
      return 'bg-zinc-800 text-zinc-300 border-zinc-700'
  }
}

function handleSelectMultiPV(n: number) {
  emit('changeMultiPV', n)
}

function handleApplyThreads(t: number) {
  selectedThreads.value = t
  emit('changeThreads', t)
}

function handleApplyHash(h: number) {
  selectedHash.value = h
  emit('changeHash', h)
}
</script>

<template>
  <div class="engine-panel bg-zinc-900 border border-zinc-800 rounded-xl p-3 shadow-lg flex flex-col gap-2.5">
    
    <!-- Top Row: Engine controls & Evaluation score badge -->
    <div class="flex items-center justify-between gap-2 flex-wrap">
      
      <!-- Left side: Play/Stop + Engine selector + MultiPV quick pills -->
      <div class="flex items-center gap-2 flex-wrap">
        
        <!-- Toggle Analysis Button -->
        <button
          class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg font-medium text-xs transition-colors cursor-pointer shrink-0"
          :class="isAnalyzing
            ? 'bg-amber-600/30 text-amber-300 border border-amber-500/50 hover:bg-amber-600/40'
            : 'bg-emerald-600/30 text-emerald-300 border border-emerald-500/50 hover:bg-emerald-600/40'"
          @click="emit('toggleAnalysis')"
        >
          <Square v-if="isAnalyzing" class="w-3.5 h-3.5 fill-current" />
          <Play v-else class="w-3.5 h-3.5 fill-current" />
          <span>{{ isAnalyzing ? 'Stop' : 'Analyze' }}</span>
        </button>

        <!-- Engine Selector Dropdown -->
        <div v-if="engineOptions.length > 0" class="relative">
          <button
            class="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg bg-zinc-800 hover:bg-zinc-700/90 text-white text-xs font-semibold border border-zinc-650 transition-colors cursor-pointer shadow-xs"
            title="Click to change chess engine"
            @click="showEngineDropdown = !showEngineDropdown; showSettingsDropdown = false"
          >
            <Cpu class="w-3.5 h-3.5 text-emerald-400 shrink-0" />
            <span class="font-bold text-zinc-100 max-w-[130px] truncate">{{ activeEngineName }}</span>
            <ChevronDown class="w-3 h-3 text-zinc-400 shrink-0 ml-0.5" />
          </button>

          <!-- Backdrop -->
          <div
            v-if="showEngineDropdown"
            class="fixed inset-0 z-40"
            @click="showEngineDropdown = false"
          />

          <!-- Engine Menu -->
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

        <!-- Multi-PV Lines Quick Selector -->
        <div class="flex items-center bg-zinc-950 rounded-lg p-0.5 border border-zinc-800 shadow-inner">
          <span class="text-[10px] text-zinc-500 font-semibold px-1.5 uppercase tracking-wider flex items-center gap-1">
            <Layers class="w-3 h-3 text-cyan-400" />
            Lines:
          </span>
          <button
            v-for="n in [1, 2, 3, 4, 5]"
            :key="n"
            class="px-2 py-0.5 text-xs font-mono rounded font-medium transition-all cursor-pointer"
            :class="currentMultiPv === n 
              ? 'bg-cyan-600 text-white shadow-xs font-bold' 
              : 'text-zinc-400 hover:text-zinc-100 hover:bg-zinc-800/80'"
            :title="`Analyze ${n} candidate principal variation line${n > 1 ? 's' : ''}`"
            @click="handleSelectMultiPV(n)"
          >
            {{ n }}
          </button>
        </div>

        <!-- Engine Advanced Settings Popover Toggle -->
        <div class="relative">
          <button
            class="p-1.5 rounded-lg bg-zinc-800 hover:bg-zinc-700/80 text-zinc-300 hover:text-white border border-zinc-700 transition-colors cursor-pointer"
            title="Engine configuration (Threads, Hash)"
            @click="showSettingsDropdown = !showSettingsDropdown; showEngineDropdown = false"
          >
            <Sliders class="w-3.5 h-3.5" />
          </button>

          <!-- Backdrop -->
          <div
            v-if="showSettingsDropdown"
            class="fixed inset-0 z-40"
            @click="showSettingsDropdown = false"
          />

          <!-- Settings Popover -->
          <div
            v-if="showSettingsDropdown"
            class="absolute bottom-full left-0 mb-1.5 w-60 bg-zinc-900 border border-zinc-700 rounded-xl shadow-2xl p-3 z-50 text-xs space-y-3"
          >
            <div class="font-semibold text-xs text-white border-b border-zinc-800 pb-1.5 flex items-center justify-between">
              <span>Engine Parameters</span>
              <span class="text-[10px] text-zinc-500 font-mono">UCI Options</span>
            </div>

            <!-- Threads -->
            <div>
              <label class="text-[11px] text-zinc-400 block mb-1">CPU Threads</label>
              <div class="flex items-center gap-1">
                <button
                  v-for="t in [1, 2, 4, 8, 16]"
                  :key="t"
                  class="flex-1 py-1 rounded text-center font-mono text-[11px] border transition-colors cursor-pointer"
                  :class="selectedThreads === t
                    ? 'bg-emerald-600 text-white border-emerald-500 font-bold'
                    : 'bg-zinc-950 text-zinc-300 border-zinc-800 hover:bg-zinc-800'"
                  @click="handleApplyThreads(t)"
                >
                  {{ t }}
                </button>
              </div>
            </div>

            <!-- Hash Size -->
            <div>
              <label class="text-[11px] text-zinc-400 block mb-1">Hash Memory (MB)</label>
              <div class="grid grid-cols-3 gap-1">
                <button
                  v-for="h in [16, 64, 128, 256, 512, 1024]"
                  :key="h"
                  class="py-1 rounded text-center font-mono text-[11px] border transition-colors cursor-pointer"
                  :class="selectedHash === h
                    ? 'bg-emerald-600 text-white border-emerald-500 font-bold'
                    : 'bg-zinc-950 text-zinc-300 border-zinc-800 hover:bg-zinc-800'"
                  @click="handleApplyHash(h)"
                >
                  {{ h }}
                </button>
              </div>
            </div>
          </div>
        </div>

      </div>

      <!-- Right side: Score & Depth Badge -->
      <div class="flex items-center gap-2 shrink-0">
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

    <!-- Multi-PV Lines View (when Lines > 1) -->
    <div
      v-if="currentMultiPv > 1"
      class="multi-pv-container space-y-1.5 max-h-48 overflow-y-auto pr-0.5"
    >
      <div
        v-for="line in sortedLines"
        :key="line.multipv"
        class="flex items-start gap-2 p-1.5 rounded-lg bg-zinc-950/80 border border-zinc-800/80 hover:border-zinc-700 hover:bg-zinc-850/90 transition-colors font-mono text-xs cursor-pointer group"
        :title="`Click to preview candidate move ${line.san || line.moveUci}`"
        @click="emit('previewMove', line.moveUci)"
      >
        <!-- Rank Badge -->
        <span
          class="px-1.5 py-0.5 rounded text-[10px] font-bold border shrink-0 mt-0.5"
          :class="getRankBadgeStyle(line.multipv)"
        >
          #{{ line.multipv }}
        </span>

        <!-- Score pill -->
        <span
          class="px-1.5 py-0.5 rounded text-[11px] font-semibold font-mono shrink-0 mt-0.5"
          :class="line.scoreCp >= 0 
            ? 'bg-zinc-800 text-emerald-300' 
            : 'bg-zinc-900 text-rose-300 border border-zinc-800'"
        >
          {{ formatScoreVal(line.scoreCp, line.mateIn) }}
        </span>

        <!-- Depth -->
        <span class="text-[10px] text-zinc-500 font-mono shrink-0 mt-0.5">
          d{{ line.depth }}
        </span>

        <!-- Candidate Move & Continuation -->
        <div class="flex-1 min-w-0 truncate">
          <span class="font-bold text-white group-hover:text-cyan-300 transition-colors mr-1.5">
            {{ line.san || line.moveUci }}
          </span>
          <span class="text-zinc-400 group-hover:text-zinc-200 transition-colors text-[11px]">
            {{ line.pvFormatted || line.pv?.join(' ') }}
          </span>
        </div>
      </div>

      <div
        v-if="sortedLines.length === 0"
        class="text-xs text-zinc-500 italic p-2 text-center"
      >
        Waiting for engine analysis ({{ currentMultiPv }} lines)...
      </div>
    </div>

    <!-- Single PV Line View (when Lines == 1) -->
    <div
      v-else
      class="pv-container bg-zinc-950/70 border border-zinc-800/60 rounded-lg p-2 font-mono text-xs text-zinc-300 truncate"
    >
      <span class="text-zinc-500 font-semibold mr-1.5">PV:</span>
      <span class="text-emerald-400 select-all">{{ pvLineString }}</span>
    </div>

  </div>
</template>

<style scoped>
.multi-pv-container::-webkit-scrollbar {
  width: 4px;
}
.multi-pv-container::-webkit-scrollbar-thumb {
  background: rgba(255, 255, 255, 0.1);
  border-radius: 4px;
}
</style>
