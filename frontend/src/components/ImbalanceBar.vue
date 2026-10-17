<script setup lang="ts">
import { ref, computed } from 'vue'
import { Shield, Sparkles, AlertTriangle, ArrowRight, Activity } from '@lucide/vue'
import type { imbalance } from '../../wailsjs/go/models'

const props = defineProps<{
  report: imbalance.Report | null
  compact?: boolean
}>()

const matDelta = computed(() => props.report?.materialDelta ?? 0)

const matLabel = computed(() => {
  if (matDelta.value > 0) return `+${matDelta.value} White`
  if (matDelta.value < 0) return `+${Math.abs(matDelta.value)} Black`
  return 'Equal'
})

const whiteKingColor = computed(() => {
  const s = props.report?.white?.kingSafety ?? 100
  if (s >= 75) return 'bg-emerald-500'
  if (s >= 50) return 'bg-amber-500'
  return 'bg-red-500'
})

const blackKingColor = computed(() => {
  const s = props.report?.black?.kingSafety ?? 100
  if (s >= 75) return 'bg-emerald-500'
  if (s >= 50) return 'bg-amber-500'
  return 'bg-red-500'
})

const isExpanded = ref(!props.compact)
</script>

<template>
  <div
    v-if="report"
    class="w-full bg-zinc-900/90 border border-zinc-800 rounded-xl p-2 flex flex-col gap-1.5 text-xs shadow-md select-none transition-all"
  >
    <!-- Header: Material & Dynamic Summary (Clickable to collapse/expand) -->
    <div
      class="flex items-center justify-between text-[11px] cursor-pointer hover:opacity-90"
      @click="isExpanded = !isExpanded"
      title="Click to toggle positional breakdown"
    >
      <div class="flex items-center gap-1.5">
        <span class="font-bold text-zinc-100 flex items-center gap-1">
          <Activity class="w-3.5 h-3.5 text-cyan-400" /> Imbalances:
        </span>
        <span
          class="px-1.5 py-0.2 rounded font-mono font-bold text-[10px]"
          :class="{
            'bg-zinc-800 text-zinc-300': matDelta === 0,
            'bg-emerald-500/20 text-emerald-300 border border-emerald-500/30': matDelta > 0,
            'bg-purple-500/20 text-purple-300 border border-purple-500/30': matDelta < 0
          }"
        >
          {{ matLabel }}
        </span>
      </div>
      <div class="flex items-center gap-2 overflow-hidden">
        <p class="text-zinc-400 text-[10px] italic truncate max-w-[200px]">
          {{ report.positionalSummary }}
        </p>
        <span class="text-zinc-500 hover:text-zinc-300 text-[10px] font-mono shrink-0">
          {{ isExpanded ? '▲' : '▼' }}
        </span>
      </div>
    </div>

    <!-- Comparative Metrics Grid -->
    <div v-show="isExpanded" class="grid grid-cols-2 gap-2 text-[11px] pt-1 border-t border-zinc-800/80">
      <!-- White Metrics -->
      <div class="bg-zinc-950/60 p-2 rounded-lg border border-zinc-800 flex flex-col gap-1.5">
        <div class="flex items-center justify-between font-semibold text-zinc-200">
          <span class="flex items-center gap-1">⚪ White</span>
          <span v-if="report.white.hasBishopPair" class="text-[9px] px-1.5 py-0.2 rounded bg-amber-500/20 text-amber-300 border border-amber-500/30 font-bold">
            2B Pair
          </span>
        </div>

        <div class="flex items-center justify-between text-[10px] text-zinc-400">
          <span class="flex items-center gap-1">
            <Shield class="w-3 h-3 text-zinc-500" /> King Shelter:
          </span>
          <div class="flex items-center gap-1.5">
            <div class="w-16 h-1.5 rounded-full bg-zinc-800 overflow-hidden">
              <div class="h-full rounded-full transition-all" :class="whiteKingColor" :style="{ width: `${report.white.kingSafety}%` }" />
            </div>
            <span class="font-mono">{{ report.white.kingSafety }}%</span>
          </div>
        </div>

        <div class="flex items-center justify-between text-[10px] text-zinc-400">
          <span>Pawns Structure:</span>
          <div class="flex items-center gap-1">
            <span v-if="report.white.pawnStructure.passedPawns > 0" class="text-emerald-400 font-semibold" title="Passed Pawns">
              {{ report.white.pawnStructure.passedPawns }} Passed
            </span>
            <span v-if="report.white.pawnStructure.isolatedPawns > 0" class="text-amber-400" title="Isolated Pawns">
              {{ report.white.pawnStructure.isolatedPawns }} Iso
            </span>
            <span v-if="report.white.pawnStructure.doubledPawns > 0" class="text-red-400" title="Doubled Pawns">
              {{ report.white.pawnStructure.doubledPawns }} Dbl
            </span>
            <span v-if="report.white.pawnStructure.passedPawns === 0 && report.white.pawnStructure.isolatedPawns === 0 && report.white.pawnStructure.doubledPawns === 0" class="text-zinc-500">
              Healthy
            </span>
          </div>
        </div>
      </div>

      <!-- Black Metrics -->
      <div class="bg-zinc-950/60 p-2 rounded-lg border border-zinc-800 flex flex-col gap-1.5">
        <div class="flex items-center justify-between font-semibold text-zinc-200">
          <span class="flex items-center gap-1">⚫ Black</span>
          <span v-if="report.black.hasBishopPair" class="text-[9px] px-1.5 py-0.2 rounded bg-amber-500/20 text-amber-300 border border-amber-500/30 font-bold">
            2B Pair
          </span>
        </div>

        <div class="flex items-center justify-between text-[10px] text-zinc-400">
          <span class="flex items-center gap-1">
            <Shield class="w-3 h-3 text-zinc-500" /> King Shelter:
          </span>
          <div class="flex items-center gap-1.5">
            <div class="w-16 h-1.5 rounded-full bg-zinc-800 overflow-hidden">
              <div class="h-full rounded-full transition-all" :class="blackKingColor" :style="{ width: `${report.black.kingSafety}%` }" />
            </div>
            <span class="font-mono">{{ report.black.kingSafety }}%</span>
          </div>
        </div>

        <div class="flex items-center justify-between text-[10px] text-zinc-400">
          <span>Pawns Structure:</span>
          <div class="flex items-center gap-1">
            <span v-if="report.black.pawnStructure.passedPawns > 0" class="text-emerald-400 font-semibold" title="Passed Pawns">
              {{ report.black.pawnStructure.passedPawns }} Passed
            </span>
            <span v-if="report.black.pawnStructure.isolatedPawns > 0" class="text-amber-400" title="Isolated Pawns">
              {{ report.black.pawnStructure.isolatedPawns }} Iso
            </span>
            <span v-if="report.black.pawnStructure.doubledPawns > 0" class="text-red-400" title="Doubled Pawns">
              {{ report.black.pawnStructure.doubledPawns }} Dbl
            </span>
            <span v-if="report.black.pawnStructure.passedPawns === 0 && report.black.pawnStructure.isolatedPawns === 0 && report.black.pawnStructure.doubledPawns === 0" class="text-zinc-500">
              Healthy
            </span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
