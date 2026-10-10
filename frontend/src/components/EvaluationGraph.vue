<script setup lang="ts">
import { computed } from 'vue'
import { Sparkles, TrendingUp, AlertOctagon, AlertCircle, HelpCircle, Clock } from '@lucide/vue'

export interface MoveAnalysisItem {
  ply: number
  san: string
  scoreCp: number
  deltaCp?: number
  nag?: number
  comment?: string
}

export interface GameAnalysisData {
  whiteAccuracy: number
  blackAccuracy: number
  whiteBlunders: number
  whiteMistakes: number
  whiteInaccs: number
  blackBlunders: number
  blackMistakes: number
  blackInaccs: number
  moveAnalyses: MoveAnalysisItem[]
}

const props = defineProps<{
  analysis: GameAnalysisData | null
  currentPly: number
  isAnalyzingGame?: boolean
  progress?: { currentPly: number; totalPlies: number; percent: number } | null
  moves?: Array<{ ply: number; clock?: string; emt?: string; san?: string }>
}>()

const emit = defineEmits<{
  (e: 'jump', ply: number): void
  (e: 'annotate'): void
}>()

// Graph coordinate dimensions
const width = 600
const height = 110
const padding = 15

// Clamped evaluations between -600 cp and +600 cp for visible graph
function clampCp(cp: number): number {
  return Math.max(-600, Math.min(600, cp))
}

function getY(cp: number): number {
  const clamped = clampCp(cp)
  // 600 -> padding, -600 -> height - padding, 0 -> height / 2
  const midY = height / 2
  const range = (height - 2 * padding) / 2
  return midY - (clamped / 600) * range
}

function getX(idx: number, total: number): number {
  if (total <= 1) return padding
  return padding + (idx / (total - 1)) * (width - 2 * padding)
}

const points = computed(() => {
  if (!props.analysis || props.analysis.moveAnalyses.length === 0) {
    return []
  }
  const moves = props.analysis.moveAnalyses
  const total = moves.length
  return moves.map((m, idx) => ({
    ply: m.ply,
    san: m.san,
    scoreCp: m.scoreCp,
    nag: m.nag,
    x: getX(idx, total),
    y: getY(m.scoreCp),
  }))
})

// SVG path for line
const linePath = computed(() => {
  if (points.value.length === 0) return ''
  return points.value.reduce((acc, pt, idx) => {
    return `${acc} ${idx === 0 ? 'M' : 'L'} ${pt.x.toFixed(1)} ${pt.y.toFixed(1)}`
  }, '')
})

// SVG path for area fill
const areaPath = computed(() => {
  if (points.value.length === 0) return ''
  const midY = height / 2
  const first = points.value[0]
  const last = points.value[points.value.length - 1]
  const line = points.value.map(pt => `L ${pt.x.toFixed(1)} ${pt.y.toFixed(1)}`).join(' ')
  return `M ${first.x.toFixed(1)} ${midY} ${line} L ${last.x.toFixed(1)} ${midY} Z`
})

// Active ply indicator X
const currentMarkerX = computed(() => {
  if (points.value.length === 0) return null
  const idx = props.currentPly - 1
  if (idx < 0) return points.value[0]?.x ?? null
  if (idx >= points.value.length) return points.value[points.value.length - 1]?.x ?? null
  return points.value[idx]?.x ?? null
})

function formatCp(cp: number): string {
  const sign = cp > 0 ? '+' : ''
  return `${sign}${(cp / 100).toFixed(2)}`
}

// Clock & Time Trouble Analysis
function parseClockSeconds(clockStr?: string): number | null {
  if (!clockStr) return null
  const parts = clockStr.split(':').map(p => parseFloat(p))
  if (parts.length === 3) {
    return parts[0] * 3600 + parts[1] * 60 + parts[2]
  } else if (parts.length === 2) {
    return parts[0] * 60 + parts[1]
  } else if (parts.length === 1 && !isNaN(parts[0])) {
    return parts[0]
  }
  return null
}

const clockPoints = computed(() => {
  if (!props.moves || props.moves.length === 0) return []
  const hasClocks = props.moves.some(m => !!m.clock || !!m.emt)
  if (!hasClocks) return []
  const validMoves = props.moves.filter(m => !!m.clock || !!m.emt)
  const total = validMoves.length
  return validMoves.map((m, idx) => {
    const secs = parseClockSeconds(m.clock) ?? 0
    return {
      ply: m.ply,
      san: m.san || '',
      clock: m.clock || '',
      secs,
      isTimeTrouble: secs > 0 && secs < 30,
      x: getX(idx, total)
    }
  })
})

const timeTroubleMoves = computed(() => {
  return clockPoints.value.filter(pt => pt.isTimeTrouble)
})
</script>

<template>
  <div class="evaluation-graph bg-zinc-950 border border-zinc-800 rounded-xl p-3 flex flex-col gap-2 select-none">
    <!-- Header with Accuracy & Annotate button -->
    <div class="flex items-center justify-between text-xs">
      <div class="flex items-center gap-2">
        <TrendingUp class="w-4 h-4 text-emerald-400" />
        <span class="font-bold text-zinc-200">Evaluation Curve</span>
        <span v-if="analysis && analysis.moveAnalyses && analysis.moveAnalyses.length > 0" class="text-[11px] text-zinc-400">
          White: <span class="font-bold text-emerald-400">{{ analysis.whiteAccuracy }}%</span> |
          Black: <span class="font-bold text-emerald-400">{{ analysis.blackAccuracy }}%</span>
        </span>
      </div>

      <button
        class="flex items-center gap-1.5 px-2.5 py-1 rounded-lg bg-zinc-900 hover:bg-zinc-800 border border-zinc-800 text-[11px] font-semibold text-amber-400 transition-colors cursor-pointer"
        :class="{ 'animate-pulse opacity-80 cursor-wait': isAnalyzingGame }"
        :disabled="isAnalyzingGame"
        @click="emit('annotate')"
      >
        <Sparkles class="w-3.5 h-3.5" :class="{ 'animate-spin': isAnalyzingGame }" />
        <span>{{ isAnalyzingGame ? (progress ? `Analyzing (${progress.percent}%)...` : 'Analyzing Game...') : (analysis && analysis.moveAnalyses && analysis.moveAnalyses.length > 0 ? 'Re-Annotate' : 'Annotate Game') }}</span>
      </button>
    </div>

    <!-- Blunder & Mistake Stats Bar -->
    <div v-if="analysis && analysis.moveAnalyses && analysis.moveAnalyses.length > 0" class="flex items-center justify-between text-[11px] px-2 py-1 bg-zinc-900/60 rounded-lg border border-zinc-800/80 font-mono">
      <div class="flex items-center gap-3">
        <span class="text-zinc-500">White:</span>
        <span class="text-rose-400 flex items-center gap-0.5" title="Blunders (??)">
          <span class="font-bold">{{ analysis.whiteBlunders }}</span> ??
        </span>
        <span class="text-amber-400 flex items-center gap-0.5" title="Mistakes (?)">
          <span class="font-bold">{{ analysis.whiteMistakes }}</span> ?
        </span>
        <span class="text-yellow-400 flex items-center gap-0.5" title="Inaccuracies (?!)">
          <span class="font-bold">{{ analysis.whiteInaccs }}</span> ?!
        </span>
      </div>

      <div class="flex items-center gap-3">
        <span class="text-zinc-500">Black:</span>
        <span class="text-rose-400 flex items-center gap-0.5" title="Blunders (??)">
          <span class="font-bold">{{ analysis.blackBlunders }}</span> ??
        </span>
        <span class="text-amber-400 flex items-center gap-0.5" title="Mistakes (?)">
          <span class="font-bold">{{ analysis.blackMistakes }}</span> ?
        </span>
        <span class="text-yellow-400 flex items-center gap-0.5" title="Inaccuracies (?!)">
          <span class="font-bold">{{ analysis.blackInaccs }}</span> ?!
        </span>
      </div>
    </div>

    <!-- Clock Sparkline & Time Trouble Banner (if clock data present) -->
    <div
      v-if="clockPoints.length > 0"
      class="clock-sparkline-bar flex items-center justify-between px-2.5 py-1 bg-zinc-900/70 rounded-lg border border-zinc-800/80 text-[11px] font-mono shadow-xs"
    >
      <div class="flex items-center gap-1.5 text-zinc-300">
        <Clock class="w-3.5 h-3.5 text-sky-400" />
        <span class="text-zinc-400">Clock Dynamics:</span>
        <span class="text-zinc-200 font-semibold">{{ clockPoints.length }} timed moves</span>
      </div>

      <div v-if="timeTroubleMoves.length > 0" class="flex items-center gap-1.5 text-amber-400 font-semibold">
        <span class="w-2 h-2 rounded-full bg-amber-400 animate-pulse" />
        <span>{{ timeTroubleMoves.length }} time trouble scramble{{ timeTroubleMoves.length > 1 ? 's' : '' }} (&lt;30s)</span>
      </div>
      <div v-else class="text-emerald-400/80 text-[10px] flex items-center gap-1">
        <span>✓ No severe time scrambles</span>
      </div>
    </div>

    <!-- SVG Graph Container -->
    <div class="relative w-full h-28 bg-zinc-900/90 rounded-lg border border-zinc-800/80 overflow-hidden flex items-center justify-center">
      <template v-if="points.length > 0">
        <svg
          :viewBox="`0 0 ${width} ${height}`"
          class="w-full h-full cursor-crosshair"
          preserveAspectRatio="none"
        >
          <!-- Gradients -->
          <defs>
            <linearGradient id="evalGrad" x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stop-color="#10b981" stop-opacity="0.4" />
              <stop offset="50%" stop-color="#10b981" stop-opacity="0.05" />
              <stop offset="50%" stop-color="#f43f5e" stop-opacity="0.05" />
              <stop offset="100%" stop-color="#f43f5e" stop-opacity="0.4" />
            </linearGradient>
          </defs>

          <!-- Zero Line -->
          <line
            :x1="padding"
            :y1="height / 2"
            :x2="width - padding"
            :y2="height / 2"
            stroke="#52525b"
            stroke-dasharray="3 3"
            stroke-width="1"
          />

          <!-- Area fill -->
          <path :d="areaPath" fill="url(#evalGrad)" />

          <!-- Evaluation Curve Line -->
          <path
            :d="linePath"
            fill="none"
            stroke="#10b981"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
          />

          <!-- Active Ply Vertical Line Marker -->
          <line
            v-if="currentMarkerX !== null"
            :x1="currentMarkerX"
            y1="0"
            :x2="currentMarkerX"
            :y2="height"
            stroke="#fbbf24"
            stroke-width="2"
            stroke-dasharray="2 2"
          />

          <!-- Interactive Ply Points & Blunder Markers -->
          <g v-for="pt in points" :key="pt.ply" @click="emit('jump', pt.ply)">
            <!-- Blunder (??) Marker -->
            <circle
              v-if="pt.nag === 4"
              :cx="pt.x"
              :cy="pt.y"
              r="4.5"
              fill="#ef4444"
              stroke="#7f1d1d"
              stroke-width="1.5"
            >
              <title>Ply {{ pt.ply }} ({{ pt.san }}): Blunder (??) [{{ formatCp(pt.scoreCp) }}]</title>
            </circle>

            <!-- Mistake (?) Marker -->
            <circle
              v-else-if="pt.nag === 2"
              :cx="pt.x"
              :cy="pt.y"
              r="3.5"
              fill="#f97316"
              stroke="#7c2d12"
              stroke-width="1"
            >
              <title>Ply {{ pt.ply }} ({{ pt.san }}): Mistake (?) [{{ formatCp(pt.scoreCp) }}]</title>
            </circle>

            <!-- Invisible hit target for clicking -->
            <circle
              :cx="pt.x"
              :cy="pt.y"
              r="8"
              fill="transparent"
              class="cursor-pointer"
            >
              <title>Ply {{ pt.ply }} ({{ pt.san }}): {{ formatCp(pt.scoreCp) }}</title>
            </circle>
          </g>
        </svg>
      </template>

      <!-- Progress indicator while analyzing -->
      <div v-else-if="isAnalyzingGame" class="flex flex-col items-center justify-center gap-2 text-zinc-300 w-full max-w-xs px-4 text-center">
        <div class="flex items-center gap-2 text-xs font-semibold text-amber-400">
          <Sparkles class="w-4 h-4 animate-spin text-amber-400" />
          <span>Analyzing Position (Ply {{ progress?.currentPly || 1 }} / {{ progress?.totalPlies || '...' }})</span>
        </div>
        <div class="w-full bg-zinc-800 rounded-full h-2 overflow-hidden border border-zinc-700/80">
          <div
            class="bg-gradient-to-r from-amber-500 to-emerald-500 h-2 rounded-full transition-all duration-300"
            :style="{ width: `${progress?.percent || 5}%` }"
          />
        </div>
        <span class="text-[10px] text-zinc-400 font-mono">{{ progress?.percent || 0 }}% completed</span>
      </div>

      <!-- Placeholder if game not analyzed yet -->
      <div v-else class="flex flex-col items-center justify-center gap-1.5 text-zinc-500 text-xs">
        <Sparkles class="w-5 h-5 text-zinc-600" />
        <span>Click "Annotate Game" to generate centipawn advantage curve & blunder report</span>
      </div>
    </div>
  </div>
</template>
