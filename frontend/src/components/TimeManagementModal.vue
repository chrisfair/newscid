<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import {
  X,
  Clock,
  Timer,
  AlertTriangle,
  Zap,
  TrendingDown,
  TrendingUp,
  Activity,
  Flame,
  Award,
  Search,
  Users,
  ChevronRight,
  ShieldAlert,
  Info,
  CheckCircle2
} from '@lucide/vue'
import {
  AnalyzeGameTimeManagement,
  AnalyzePlayerTimeManagement
} from '../../wailsjs/go/main/App'
import { stats } from '../../wailsjs/go/models'

const props = withDefaults(defineProps<{
  gameId?: number
  defaultPlayerName?: string
}>(), {
  gameId: 0,
  defaultPlayerName: ''
})

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'jumpToPly', ply: number): void
}>()

// Tabs
type TabMode = 'game' | 'player'
const activeTab = ref<TabMode>('game')

// Game Report State
const gameReport = ref<stats.GameTimeManagementReport | null>(null)
const isGameLoading = ref<boolean>(false)
const gameError = ref<string>('')

// Player Report State
const playerNameInput = ref<string>(props.defaultPlayerName || '')
const playerDossier = ref<stats.PlayerTimeManagementDossier | null>(null)
const isPlayerLoading = ref<boolean>(false)
const playerError = ref<string>('')

// Move filter in Game tab
const moveFilter = ref<'all' | 'tanks' | 'panics' | 'scramble'>('all')

const filteredMoves = computed(() => {
  if (!gameReport.value || !gameReport.value.moves) return []
  if (moveFilter.value === 'tanks') {
    return gameReport.value.moves.filter(m => m.isTank)
  }
  if (moveFilter.value === 'panics') {
    return gameReport.value.moves.filter(m => m.isPanic)
  }
  if (moveFilter.value === 'scramble') {
    return gameReport.value.moves.filter(m => m.isScramble)
  }
  return gameReport.value.moves
})

const maxMoveTimeSec = computed(() => {
  if (!gameReport.value || !gameReport.value.moves) return 60
  let maxT = 10
  for (const m of gameReport.value.moves) {
    if (m.timeSpentSec > maxT) maxT = m.timeSpentSec
  }
  return Math.max(maxT, 30)
})

async function fetchGameAnalysis() {
  isGameLoading.value = true
  gameError.value = ''
  try {
    const rep = await AnalyzeGameTimeManagement(props.gameId)
    gameReport.value = rep
    if (!playerNameInput.value && rep.whitePlayer && rep.whitePlayer !== '?') {
      playerNameInput.value = rep.whitePlayer
    }
  } catch (err: unknown) {
    gameError.value = err instanceof Error ? err.message : String(err)
  } finally {
    isGameLoading.value = false
  }
}

async function fetchPlayerAnalysis() {
  const name = playerNameInput.value.trim()
  if (!name) return

  isPlayerLoading.value = true
  playerError.value = ''
  try {
    const dos = await AnalyzePlayerTimeManagement(name, 100)
    playerDossier.value = dos
  } catch (err: unknown) {
    playerError.value = err instanceof Error ? err.message : String(err)
  } finally {
    isPlayerLoading.value = false
  }
}

function formatSec(sec: number): string {
  if (sec <= 0) return '0s'
  const m = Math.floor(sec / 60)
  const s = Math.round(sec % 60)
  if (m > 0) {
    return `${m}m ${s < 10 ? '0' : ''}${s}s`
  }
  return `${s}s`
}

function getMoveTimeBarWidth(sec: number): string {
  const pct = Math.min(100, Math.max(2, (sec / maxMoveTimeSec.value) * 100))
  return `${pct}%`
}

onMounted(() => {
  fetchGameAnalysis()
})
</script>

<template>
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/75 backdrop-blur-sm p-4 select-none">
    <div
      class="bg-theme-panel border border-theme-border rounded-xl shadow-2xl flex flex-col w-full max-w-5xl overflow-hidden animate-in fade-in zoom-in-95 duration-150"
      style="height: 88vh; max-height: 850px;"
    >
      <!-- Header -->
      <div class="px-6 py-3.5 border-b border-theme-border flex items-center justify-between bg-theme-panel-header">
        <div class="flex items-center gap-3">
          <div class="p-2 bg-indigo-500/10 text-indigo-400 rounded-lg border border-indigo-500/20">
            <Clock class="w-5 h-5 text-indigo-400" />
          </div>
          <div>
            <div class="flex items-center gap-2">
              <h2 class="text-base font-bold text-theme-text-primary tracking-wide">Time Management & Tanking Dossier</h2>
              <span class="px-2 py-0.5 text-xs font-semibold rounded bg-indigo-500/20 text-indigo-300 border border-indigo-500/30">
                %clk &amp; %emt Analytics
              </span>
            </div>
            <p class="text-xs text-theme-text-muted mt-0.5">
              Deep clock telemetry, long tank outcomes, panic moves, and zeitnot vulnerability audit.
            </p>
          </div>
        </div>

        <div class="flex items-center gap-3">
          <!-- Navigation Tabs -->
          <div class="flex items-center bg-theme-panel border border-theme-border rounded-lg p-0.5 text-xs">
            <button
              class="px-3 py-1 rounded transition-colors flex items-center gap-1.5"
              :class="activeTab === 'game' ? 'bg-indigo-600 text-white font-semibold shadow' : 'text-theme-text-muted hover:text-theme-text-primary'"
              @click="activeTab = 'game'"
            >
              <Activity class="w-3.5 h-3.5" />
              Game Timeline
            </button>
            <button
              class="px-3 py-1 rounded transition-colors flex items-center gap-1.5"
              :class="activeTab === 'player' ? 'bg-indigo-600 text-white font-semibold shadow' : 'text-theme-text-muted hover:text-theme-text-primary'"
              @click="activeTab = 'player'; if (!playerDossier && playerNameInput) fetchPlayerAnalysis()"
            >
              <Users class="w-3.5 h-3.5" />
              Player Historical Dossier
            </button>
          </div>

          <button
            class="text-theme-text-muted hover:text-theme-text-primary p-1.5 rounded-lg hover:bg-theme-hover transition-colors"
            @click="emit('close')"
          >
            <X class="w-5 h-5" />
          </button>
        </div>
      </div>

      <!-- TAB 1: CURRENT GAME TIMELINE & TANKING AUDIT -->
      <div v-if="activeTab === 'game'" class="flex-1 flex flex-col overflow-hidden">
        <!-- Game Summary Bar -->
        <div v-if="gameReport" class="px-6 py-2.5 bg-theme-panel-header/40 border-b border-theme-border/60 flex items-center justify-between text-xs">
          <div class="flex items-center gap-4">
            <div class="flex items-center gap-1.5">
              <span class="w-2.5 h-2.5 rounded-full bg-white border border-gray-400"></span>
              <span class="font-semibold text-theme-text-primary">{{ gameReport.whitePlayer || 'White' }}</span>
              <span class="text-theme-text-muted text-[11px]">
                (Avg: {{ formatSec(gameReport.whiteStats.avgMoveTimeSec) }} / Longest: {{ formatSec(gameReport.whiteStats.longestTankSec) }})
              </span>
            </div>
            <span class="text-theme-text-muted font-bold">&bull;</span>
            <div class="flex items-center gap-1.5">
              <span class="w-2.5 h-2.5 rounded-full bg-black border border-gray-600"></span>
              <span class="font-semibold text-theme-text-primary">{{ gameReport.blackPlayer || 'Black' }}</span>
              <span class="text-theme-text-muted text-[11px]">
                (Avg: {{ formatSec(gameReport.blackStats.avgMoveTimeSec) }} / Longest: {{ formatSec(gameReport.blackStats.longestTankSec) }})
              </span>
            </div>
          </div>

          <!-- Time control badge -->
          <div class="flex items-center gap-2">
            <span v-if="gameReport.timeControl" class="px-2 py-0.5 rounded bg-theme-panel font-mono text-[11px] text-theme-text-muted border border-theme-border">
              {{ gameReport.timeControl }}
            </span>
            <span v-if="!gameReport.whiteHasClock && !gameReport.blackHasClock" class="text-amber-400 text-[11px] flex items-center gap-1">
              <AlertTriangle class="w-3 h-3" /> No %clk tags found in PGN
            </span>
          </div>
        </div>

        <!-- Metrics cards row -->
        <div v-if="gameReport" class="px-6 py-3 border-b border-theme-border grid grid-cols-2 md:grid-cols-4 gap-3 text-xs bg-theme-panel">
          <div class="p-2.5 rounded-lg border border-theme-border bg-theme-panel-header/30">
            <div class="text-theme-text-muted flex items-center justify-between mb-1">
              <span>Deep Tanks (&ge;2m)</span>
              <Timer class="w-3.5 h-3.5 text-amber-400" />
            </div>
            <div class="flex items-baseline gap-2">
              <span class="text-base font-bold text-amber-400">{{ gameReport.tankDossier.length }}</span>
              <span class="text-[11px] text-theme-text-muted">
                (White: {{ gameReport.whiteStats.tankCount }} / Black: {{ gameReport.blackStats.tankCount }})
              </span>
            </div>
          </div>

          <div class="p-2.5 rounded-lg border border-theme-border bg-theme-panel-header/30">
            <div class="text-theme-text-muted flex items-center justify-between mb-1">
              <span>Panic Moves (&le;5s)</span>
              <Zap class="w-3.5 h-3.5 text-purple-400" />
            </div>
            <div class="flex items-baseline gap-2">
              <span class="text-base font-bold text-purple-400">{{ gameReport.panicIncidents.length }}</span>
              <span class="text-[11px] text-theme-text-muted">
                (White: {{ gameReport.whiteStats.panicMoveCount }} / Black: {{ gameReport.blackStats.panicMoveCount }})
              </span>
            </div>
          </div>

          <div class="p-2.5 rounded-lg border border-theme-border bg-theme-panel-header/30">
            <div class="text-theme-text-muted flex items-center justify-between mb-1">
              <span>Zeitnot Moves (&le;30s)</span>
              <Flame class="w-3.5 h-3.5 text-rose-400" />
            </div>
            <div class="flex items-baseline gap-2">
              <span class="text-base font-bold text-rose-400">
                {{ gameReport.scrambleSummary.whiteScrambleMoves + gameReport.scrambleSummary.blackScrambleMoves }}
              </span>
              <span class="text-[11px] text-theme-text-muted">
                ({{ gameReport.scrambleSummary.whiteScrambleBlunders + gameReport.scrambleSummary.blackScrambleBlunders }} blunders)
              </span>
            </div>
          </div>

          <div class="p-2.5 rounded-lg border border-theme-border bg-theme-panel-header/30">
            <div class="text-theme-text-muted flex items-center justify-between mb-1">
              <span>Tank ROI (Success Rate)</span>
              <Award class="w-3.5 h-3.5 text-emerald-400" />
            </div>
            <div class="flex items-baseline gap-2">
              <span class="text-base font-bold text-emerald-400">
                {{
                  gameReport.tankDossier.length > 0
                    ? Math.round(((gameReport.tankDossier.filter(t => t.outcome === 'Masterstroke' || t.outcome === 'Solid').length) / gameReport.tankDossier.length) * 100)
                    : 100
                }}%
              </span>
              <span class="text-[11px] text-theme-text-muted">Tanks with good moves</span>
            </div>
          </div>
        </div>

        <!-- Filter bar -->
        <div class="px-6 py-2 border-b border-theme-border flex items-center justify-between text-xs bg-theme-panel-header/20">
          <div class="flex items-center gap-2">
            <span class="text-theme-text-muted">Filter:</span>
            <button
              class="px-2.5 py-0.5 rounded transition-colors"
              :class="moveFilter === 'all' ? 'bg-indigo-600 text-white font-medium' : 'text-theme-text-muted hover:text-theme-text-primary'"
              @click="moveFilter = 'all'"
            >
              All Moves ({{ gameReport?.moves.length || 0 }})
            </button>
            <button
              class="px-2.5 py-0.5 rounded transition-colors flex items-center gap-1"
              :class="moveFilter === 'tanks' ? 'bg-amber-600 text-white font-medium' : 'text-theme-text-muted hover:text-theme-text-primary'"
              @click="moveFilter = 'tanks'"
            >
              <span>Tanks</span>
              <span class="text-[10px] px-1 rounded bg-black/30">({{ gameReport?.tankDossier.length || 0 }})</span>
            </button>
            <button
              class="px-2.5 py-0.5 rounded transition-colors flex items-center gap-1"
              :class="moveFilter === 'panics' ? 'bg-purple-600 text-white font-medium' : 'text-theme-text-muted hover:text-theme-text-primary'"
              @click="moveFilter = 'panics'"
            >
              <span>Panics</span>
              <span class="text-[10px] px-1 rounded bg-black/30">({{ gameReport?.panicIncidents.length || 0 }})</span>
            </button>
            <button
              class="px-2.5 py-0.5 rounded transition-colors flex items-center gap-1"
              :class="moveFilter === 'scramble' ? 'bg-rose-600 text-white font-medium' : 'text-theme-text-muted hover:text-theme-text-primary'"
              @click="moveFilter = 'scramble'"
            >
              <span>Scramble</span>
              <span class="text-[10px] px-1 rounded bg-black/30">
                ({{ (gameReport?.scrambleSummary.whiteScrambleMoves || 0) + (gameReport?.scrambleSummary.blackScrambleMoves || 0) }})
              </span>
            </button>
          </div>

          <span class="text-[11px] text-theme-text-muted">Click any move to jump directly on main board</span>
        </div>

        <!-- Main Content Area: Split List & Recommendations -->
        <div class="flex-1 flex flex-col md:flex-row overflow-hidden">
          <!-- Move List Timeline -->
          <div class="flex-1 overflow-y-auto p-4 space-y-1">
            <div
              v-if="isGameLoading"
              class="h-48 flex flex-col items-center justify-center gap-2 text-theme-text-muted text-xs"
            >
              <div class="w-6 h-6 border-2 border-indigo-400 border-t-transparent rounded-full animate-spin"></div>
              <span>Auditing game clocks...</span>
            </div>

            <div
              v-else-if="filteredMoves.length === 0"
              class="h-48 flex items-center justify-center text-theme-text-muted text-xs"
            >
              No moves match the selected filter.
            </div>

            <div
              v-for="m in filteredMoves"
              :key="m.ply"
              class="flex items-center gap-3 px-3 py-1.5 rounded-lg border border-transparent hover:border-theme-border hover:bg-theme-hover cursor-pointer transition-all text-xs"
              @click="emit('jumpToPly', m.ply)"
            >
              <!-- Ply & Move Number -->
              <span class="w-12 font-mono text-theme-text-muted text-[11px]">
                {{ m.turn === 'white' ? `${m.moveNumber}.` : `${m.moveNumber}...` }}
              </span>

              <!-- Move SAN -->
              <span class="w-14 font-semibold text-theme-text-primary">
                {{ m.san }}
              </span>

              <!-- Time Spent Bar -->
              <div class="flex-1 flex items-center gap-2">
                <div class="flex-1 bg-theme-panel h-2 rounded-full overflow-hidden border border-theme-border/50">
                  <div
                    class="h-full rounded-full transition-all duration-300"
                    :class="[
                      m.isTank ? 'bg-amber-500' :
                      m.isPanic ? 'bg-purple-500' :
                      m.isScramble ? 'bg-rose-500' : 'bg-indigo-500/70'
                    ]"
                    :style="{ width: getMoveTimeBarWidth(m.timeSpentSec) }"
                  ></div>
                </div>
                <span class="w-16 text-right font-mono text-[11px] text-theme-text-muted">
                  {{ formatSec(m.timeSpentSec) }}
                </span>
              </div>

              <!-- Remaining Clock -->
              <span v-if="m.clockStr" class="w-20 text-right font-mono text-[11px] text-theme-text-muted">
                {{ m.clockStr }}
              </span>
              <span v-else class="w-20 text-right text-[11px] text-theme-text-muted/40">-</span>

              <!-- Status Flag Badge -->
              <div class="w-28 flex justify-end">
                <span
                  v-if="m.flag === 'tank_success'"
                  class="px-1.5 py-0.5 rounded text-[10px] font-bold bg-amber-500/20 text-amber-300 border border-amber-500/30"
                >
                  TANK SUCCESS
                </span>
                <span
                  v-else-if="m.flag === 'tank_blunder'"
                  class="px-1.5 py-0.5 rounded text-[10px] font-bold bg-rose-500/20 text-rose-300 border border-rose-500/30"
                >
                  TANK BLUNDER
                </span>
                <span
                  v-else-if="m.flag === 'panic_blunder'"
                  class="px-1.5 py-0.5 rounded text-[10px] font-bold bg-purple-500/20 text-purple-300 border border-purple-500/30"
                >
                  PANIC BLUNDER
                </span>
                <span
                  v-else-if="m.isScramble"
                  class="px-1.5 py-0.5 rounded text-[10px] font-bold bg-rose-500/10 text-rose-400 border border-rose-500/20"
                >
                  ZEITNOT
                </span>
                <span
                  v-else-if="m.isTank"
                  class="px-1.5 py-0.5 rounded text-[10px] font-medium bg-amber-500/10 text-amber-400"
                >
                  TANK
                </span>
              </div>
            </div>
          </div>

          <!-- Practical Coaching & Recommendations Side Panel -->
          <div class="w-full md:w-80 border-t md:border-t-0 md:border-l border-theme-border bg-theme-panel-header/20 p-4 flex flex-col gap-4 overflow-y-auto">
            <div>
              <h3 class="text-xs font-bold text-theme-text-primary tracking-wide flex items-center gap-1.5 mb-2">
                <ShieldAlert class="w-3.5 h-3.5 text-indigo-400" />
                Coaching Dossier
              </h3>
              <div class="space-y-2">
                <div
                  v-for="(rec, idx) in gameReport?.recommendations || []"
                  :key="idx"
                  class="p-2.5 rounded-lg bg-theme-panel border border-theme-border text-xs leading-relaxed text-theme-text-primary"
                >
                  {{ rec }}
                </div>
              </div>
            </div>

            <!-- Long Tanks Log -->
            <div v-if="gameReport && gameReport.tankDossier.length > 0">
              <h3 class="text-xs font-bold text-theme-text-primary tracking-wide flex items-center gap-1.5 mb-2">
                <Timer class="w-3.5 h-3.5 text-amber-400" />
                Critical Tanks ({{ gameReport.tankDossier.length }})
              </h3>
              <div class="space-y-1.5">
                <div
                  v-for="t in gameReport.tankDossier"
                  :key="t.ply"
                  class="p-2 rounded border border-theme-border bg-theme-panel text-xs cursor-pointer hover:border-amber-500/50 transition-colors"
                  @click="emit('jumpToPly', t.ply)"
                >
                  <div class="flex items-center justify-between mb-1">
                    <span class="font-bold text-amber-300">Move {{ t.moveNumber }} ({{ t.san }})</span>
                    <span class="text-[10px] font-mono text-theme-text-muted">{{ formatSec(t.timeSpentSec) }}</span>
                  </div>
                  <p class="text-[11px] text-theme-text-muted leading-tight">
                    {{ t.description }}
                  </p>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- TAB 2: PLAYER HISTORICAL DOSSIER -->
      <div v-else class="flex-1 flex flex-col overflow-hidden p-6 gap-6">
        <!-- Player search bar -->
        <div class="flex items-center gap-3">
          <div class="relative flex-1 max-w-md">
            <Search class="absolute left-3 top-2.5 w-4 h-4 text-theme-text-muted" />
            <input
              v-model="playerNameInput"
              type="text"
              placeholder="Search player name across database (e.g. Carlsen, Kasparov)..."
              class="w-full pl-9 pr-4 py-2 bg-theme-panel border border-theme-border rounded-lg text-xs text-theme-text-primary focus:outline-none focus:border-indigo-500 transition-colors"
              @keydown.enter="fetchPlayerAnalysis"
            />
          </div>
          <button
            class="px-4 py-2 rounded-lg bg-indigo-600 hover:bg-indigo-500 text-white font-medium text-xs transition-colors flex items-center gap-1.5 shadow-sm"
            :disabled="isPlayerLoading"
            @click="fetchPlayerAnalysis"
          >
            <Activity v-if="!isPlayerLoading" class="w-3.5 h-3.5" />
            <div v-else class="w-3.5 h-3.5 border-2 border-white border-t-transparent rounded-full animate-spin"></div>
            <span>Generate Dossier</span>
          </button>
        </div>

        <div v-if="playerError" class="p-4 rounded-lg bg-rose-500/10 border border-rose-500/30 text-xs text-rose-300">
          {{ playerError }}
        </div>

        <!-- Dossier Content -->
        <div v-if="playerDossier" class="flex-1 flex flex-col gap-6 overflow-y-auto">
          <!-- Overview stats -->
          <div class="grid grid-cols-2 md:grid-cols-4 gap-4">
            <div class="p-4 rounded-xl border border-theme-border bg-theme-panel">
              <div class="text-xs text-theme-text-muted mb-1">Average Move Time</div>
              <div class="text-xl font-bold text-theme-text-primary">{{ formatSec(playerDossier.averageMoveTimeSec) }}</div>
              <div class="text-[11px] text-theme-text-muted mt-1">
                Open: {{ formatSec(playerDossier.openingAvgSec) }} &bull; Mid: {{ formatSec(playerDossier.middlegameAvgSec) }} &bull; End: {{ formatSec(playerDossier.endgameAvgSec) }}
              </div>
            </div>

            <div class="p-4 rounded-xl border border-theme-border bg-theme-panel">
              <div class="text-xs text-theme-text-muted mb-1">Tank Success Rate</div>
              <div class="text-xl font-bold text-emerald-400">{{ Math.round(playerDossier.tankSuccessRate) }}%</div>
              <div class="text-[11px] text-theme-text-muted mt-1">
                {{ playerDossier.totalTanks }} total tanks analyzed
              </div>
            </div>

            <div class="p-4 rounded-xl border border-theme-border bg-theme-panel">
              <div class="text-xs text-theme-text-muted mb-1">Scramble Frequency</div>
              <div class="text-xl font-bold text-rose-400">{{ Math.round(playerDossier.scrambleFrequencyPct) }}%</div>
              <div class="text-[11px] text-theme-text-muted mt-1">
                {{ playerDossier.scrambleBlunderCount }} errors committed in zeitnot
              </div>
            </div>

            <div class="p-4 rounded-xl border border-theme-border bg-theme-panel">
              <div class="text-xs text-theme-text-muted mb-1">Sample Coverage</div>
              <div class="text-xl font-bold text-indigo-400">{{ playerDossier.gamesWithClock }} games</div>
              <div class="text-[11px] text-theme-text-muted mt-1">
                out of {{ playerDossier.gamesAnalyzed }} matches sampled
              </div>
            </div>
          </div>

          <!-- Habit Flags & Coaching -->
          <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div class="p-4 rounded-xl border border-theme-border bg-theme-panel flex flex-col gap-3">
              <h3 class="text-xs font-bold text-theme-text-primary flex items-center gap-2">
                <AlertTriangle class="w-4 h-4 text-amber-400" />
                Detected Psychological &amp; Pacing Habits
              </h3>
              <div v-if="playerDossier.habitFlags.length === 0" class="text-xs text-theme-text-muted italic py-4">
                No negative habits detected! Clock pacing is exemplary.
              </div>
              <div v-else class="space-y-2">
                <div
                  v-for="(flag, idx) in playerDossier.habitFlags"
                  :key="idx"
                  class="p-2.5 rounded-lg bg-amber-500/10 border border-amber-500/20 text-xs text-amber-200"
                >
                  {{ flag }}
                </div>
              </div>
            </div>

            <div class="p-4 rounded-xl border border-theme-border bg-theme-panel flex flex-col gap-3">
              <h3 class="text-xs font-bold text-theme-text-primary flex items-center gap-2">
                <ShieldAlert class="w-4 h-4 text-indigo-400" />
                Grandmaster Coaching Prescription
              </h3>
              <div class="space-y-2">
                <div
                  v-for="(tip, idx) in playerDossier.coachingDossier"
                  :key="idx"
                  class="p-2.5 rounded-lg bg-indigo-500/10 border border-indigo-500/20 text-xs text-indigo-200 leading-relaxed"
                >
                  {{ tip }}
                </div>
              </div>
            </div>
          </div>
        </div>

        <div v-else-if="!isPlayerLoading" class="flex-1 flex flex-col items-center justify-center text-theme-text-muted text-xs gap-2">
          <Clock class="w-8 h-8 text-theme-text-muted/40" />
          <span>Enter a player name and click "Generate Dossier" to analyze historical clock habits.</span>
        </div>
      </div>
    </div>
  </div>
</template>
