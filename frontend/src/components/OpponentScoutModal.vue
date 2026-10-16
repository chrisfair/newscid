<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { X, Search, ShieldAlert, Crosshair, Trophy, Lightbulb, TrendingDown, TrendingUp, Sword } from '@lucide/vue'
import { GenerateOpponentScoutingReport, GetDatabasePlayers } from '../../wailsjs/go/main/App'
import type { scout } from '../../wailsjs/go/models'

const props = defineProps<{
  initialPlayer?: string
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'exploitPlayer', playerName: string): void
}>()

const playerName = ref(props.initialPlayer || '')
const knownPlayers = ref<string[]>([])
const loading = ref(false)
const error = ref('')
const report = ref<scout.Report | null>(null)

async function fetchPlayers() {
  try {
    const list = await GetDatabasePlayers()
    knownPlayers.value = (list || []).slice(0, 100)
    if (!playerName.value && knownPlayers.value.length > 0) {
      playerName.value = knownPlayers.value[0]
      generateReport()
    } else if (playerName.value) {
      generateReport()
    }
  } catch (err) {
    console.error('Failed to get database players:', err)
  }
}

async function generateReport() {
  if (!playerName.value.trim()) return

  loading.value = true
  error.value = ''
  report.value = null
  try {
    const res = await GenerateOpponentScoutingReport(playerName.value.trim())
    report.value = res
  } catch (err: unknown) {
    error.value = String(err)
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchPlayers()
})
</script>

<template>
  <div
    class="fixed inset-0 bg-black/75 backdrop-blur-xs flex items-center justify-center z-50 p-4 animate-fade-in select-none"
    @click.self="emit('close')"
  >
    <div
      class="bg-zinc-900 border border-zinc-700/80 w-full max-w-2xl rounded-2xl p-5 shadow-2xl flex flex-col gap-4 text-xs max-h-[85vh] overflow-y-auto"
    >
      <!-- Header -->
      <div class="flex items-center justify-between border-b border-zinc-800 pb-3 shrink-0">
        <div class="flex items-center gap-2.5">
          <div class="w-8 h-8 rounded-lg bg-cyan-500/20 border border-cyan-500/40 flex items-center justify-center text-cyan-300">
            <Crosshair class="w-4 h-4" />
          </div>
          <div>
            <h3 class="font-bold text-sm text-zinc-100">Opponent Scouting Card & Prep Matrix</h3>
            <p class="text-[11px] text-zinc-400">Tactical vulnerabilities and opening tendencies dossier</p>
          </div>
        </div>
        <button
          class="p-1.5 rounded-lg text-zinc-400 hover:text-zinc-200 hover:bg-zinc-800 transition-colors cursor-pointer"
          @click="emit('close')"
        >
          <X class="w-4 h-4" />
        </button>
      </div>

      <!-- Player Search Bar -->
      <div class="flex items-center gap-2 bg-zinc-950 p-2 rounded-xl border border-zinc-800 shrink-0">
        <input
          v-model="playerName"
          type="text"
          placeholder="Enter player name (e.g. Carlsen, Magnus)..."
          class="w-full bg-transparent text-xs text-zinc-200 placeholder-zinc-500 focus:outline-hidden px-2"
          @keydown.enter="generateReport"
        />
        <button
          class="px-4 py-1.5 rounded-lg bg-cyan-600 hover:bg-cyan-500 text-white font-medium text-xs cursor-pointer transition-colors shrink-0"
          @click="generateReport"
        >
          Scout Opponent
        </button>
      </div>

      <!-- Loading / Error States -->
      <div v-if="loading" class="py-12 text-center text-zinc-400 animate-pulse">
        Compiling opponent dossier from database games...
      </div>

      <div v-else-if="error" class="p-3 bg-red-950/40 border border-red-800/60 rounded-xl text-red-300 text-xs">
        {{ error }}
      </div>

      <!-- Report Content -->
      <div v-else-if="report" class="flex flex-col gap-4">
        <!-- Opening Exploiter Banner -->
        <div class="flex items-center justify-between bg-rose-950/20 border border-rose-500/30 p-2.5 rounded-xl">
          <div class="flex items-center gap-2">
            <span class="text-rose-400 font-semibold text-xs flex items-center gap-1.5">
              <Crosshair class="w-4 h-4" /> Opening Exploiter:
            </span>
            <span class="text-[11px] text-zinc-300">Compile tailored Polyglot counter-book</span>
          </div>
          <button
            class="px-3 py-1 rounded-lg bg-rose-600 hover:bg-rose-500 text-white font-medium text-xs flex items-center gap-1.5 transition-colors cursor-pointer shadow-xs"
            @click="emit('exploitPlayer', playerName)"
          >
            <Sword class="w-3.5 h-3.5" />
            Generate Weaponized Anti-Book
          </button>
        </div>

        <!-- Overview Stats Cards -->
        <div class="grid grid-cols-2 gap-3">
          <!-- As White Card -->
          <div class="bg-zinc-950/80 p-3 rounded-xl border border-zinc-800 flex flex-col gap-1.5">
            <div class="flex items-center justify-between font-semibold text-zinc-200">
              <span>⚪ Record as White</span>
              <span class="text-xs font-mono font-bold text-emerald-400">{{ report.whiteRecord.winRate }}%</span>
            </div>
            <div class="text-[11px] text-zinc-400">
              {{ report.whiteRecord.total }} Games:
              <span class="text-emerald-400">{{ report.whiteRecord.wins }}W</span> /
              <span class="text-zinc-400">{{ report.whiteRecord.draws }}D</span> /
              <span class="text-red-400">{{ report.whiteRecord.losses }}L</span>
            </div>
          </div>

          <!-- As Black Card -->
          <div class="bg-zinc-950/80 p-3 rounded-xl border border-zinc-800 flex flex-col gap-1.5">
            <div class="flex items-center justify-between font-semibold text-zinc-200">
              <span>⚫ Record as Black</span>
              <span class="text-xs font-mono font-bold text-cyan-400">{{ report.blackRecord.winRate }}%</span>
            </div>
            <div class="text-[11px] text-zinc-400">
              {{ report.blackRecord.total }} Games:
              <span class="text-emerald-400">{{ report.blackRecord.wins }}W</span> /
              <span class="text-zinc-400">{{ report.blackRecord.draws }}D</span> /
              <span class="text-red-400">{{ report.blackRecord.losses }}L</span>
            </div>
          </div>
        </div>

        <!-- Tactical Recommendations Box -->
        <div v-if="report.repertoireRecommendations.length > 0" class="p-3.5 bg-cyan-950/20 border border-cyan-500/30 rounded-xl flex flex-col gap-2">
          <div class="font-bold text-cyan-300 text-xs flex items-center gap-1.5">
            <Lightbulb class="w-4 h-4 text-cyan-400" /> Prep Strategy & Recommendations:
          </div>
          <ul class="flex flex-col gap-1.5 text-zinc-300 text-[11px] list-disc list-inside">
            <li v-for="(rec, idx) in report.repertoireRecommendations" :key="idx">
              {{ rec }}
            </li>
          </ul>
        </div>

        <!-- Sharpest vs Weakest Lines -->
        <div class="grid grid-cols-2 gap-3">
          <!-- Sharpest Lines -->
          <div class="bg-zinc-950/60 p-3 rounded-xl border border-zinc-800 flex flex-col gap-2">
            <div class="font-semibold text-zinc-200 text-xs flex items-center gap-1.5 text-emerald-400">
              <TrendingUp class="w-3.5 h-3.5" /> Sharpest Weapons (Avoid)
            </div>
            <div v-if="report.sharpestLines.length === 0" class="text-zinc-500 text-[11px]">
              No line data available.
            </div>
            <div
              v-for="line in report.sharpestLines"
              :key="line.eco"
              class="flex items-center justify-between p-2 rounded-lg bg-zinc-900 border border-zinc-800 text-[11px]"
            >
              <span class="font-mono font-bold text-zinc-200">{{ line.eco }}</span>
              <div class="flex items-center gap-2">
                <span class="text-zinc-400">{{ line.games }} g</span>
                <span class="font-mono text-emerald-400 font-bold">{{ line.winRate }}%</span>
              </div>
            </div>
          </div>

          <!-- Weakest Lines (Attack Targets) -->
          <div class="bg-zinc-950/60 p-3 rounded-xl border border-zinc-800 flex flex-col gap-2">
            <div class="font-semibold text-zinc-200 text-xs flex items-center gap-1.5 text-red-400">
              <TrendingDown class="w-3.5 h-3.5" /> Weakest Lines (Target)
            </div>
            <div v-if="report.weakestLines.length === 0" class="text-zinc-500 text-[11px]">
              No line data available.
            </div>
            <div
              v-for="line in report.weakestLines"
              :key="line.eco"
              class="flex items-center justify-between p-2 rounded-lg bg-zinc-900 border border-zinc-800 text-[11px]"
            >
              <span class="font-mono font-bold text-zinc-200">{{ line.eco }}</span>
              <div class="flex items-center gap-2">
                <span class="text-zinc-400">{{ line.games }} g</span>
                <span class="font-mono text-red-400 font-bold">{{ line.winRate }}%</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
