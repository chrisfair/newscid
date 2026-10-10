<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import {
  BarChart3,
  TrendingUp,
  X,
  Search,
  Calendar,
  Award,
  Layers,
  Clock,
  PieChart,
  ArrowUpRight,
  ArrowDownRight,
  Minus
} from '@lucide/vue'
import {
  GetDatabaseDemographics,
  GetRatingEvolution,
  GetDatabasePlayers,
  GetPersonas
} from '../../wailsjs/go/main/App'
import { stats } from '../../wailsjs/go/models'

const props = withDefaults(defineProps<{
  dbName?: string
  initialPlayer?: string
}>(), {
  dbName: 'Active Database',
  initialPlayer: ''
})

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'loadGame', gameId: number): void
}>()

// Active Tab: 'demographics' | 'rating'
const activeTab = ref<'demographics' | 'rating'>(props.initialPlayer ? 'rating' : 'demographics')

// Demographics State
const demographics = ref<stats.DatabaseDemographics | null>(null)
const loadingDemo = ref(true)

// Rating Tracker State
const playerQuery = ref(props.initialPlayer || '')
const candidatePlayers = ref<string[]>([])
const ratingResult = ref<stats.RatingTrackerResult | null>(null)
const loadingRating = ref(false)
const hoveredPoint = ref<stats.RatingPoint | null>(null)

// Load demographics
async function loadDemographics() {
  loadingDemo.value = true
  try {
    const res = await GetDatabaseDemographics()
    demographics.value = res
  } catch (err) {
    console.error('Failed to load demographics:', err)
  } finally {
    loadingDemo.value = false
  }
}

// Load rating tracker for player
async function loadRatingEvolution(name: string) {
  if (!name.trim()) return
  loadingRating.value = true
  playerQuery.value = name
  hoveredPoint.value = null
  try {
    const res = await GetRatingEvolution(name.trim())
    ratingResult.value = res
  } catch (err) {
    console.error('Failed to load rating evolution:', err)
  } finally {
    loadingRating.value = false
  }
}

onMounted(async () => {
  await loadDemographics()

  try {
    const dbPlayers = await GetDatabasePlayers()
    const personas = await GetPersonas()
    const set = new Set<string>()
    if (personas) {
      personas.forEach(p => set.add(p.displayName))
    }
    if (dbPlayers) {
      dbPlayers.slice(0, 50).forEach(p => set.add(p))
    }
    candidatePlayers.value = Array.from(set)
  } catch (e) {
    candidatePlayers.value = []
  }

  if (props.initialPlayer) {
    await loadRatingEvolution(props.initialPlayer)
  } else if (candidatePlayers.value.length > 0) {
    await loadRatingEvolution(candidatePlayers.value[0])
  }
})

// Rating Chart Geometry
const chartPoints = computed(() => {
  if (!ratingResult.value || !ratingResult.value.points || ratingResult.value.points.length === 0) {
    return []
  }
  const pts = ratingResult.value.points
  const minR = ratingResult.value.minRating - 20
  const maxR = ratingResult.value.maxRating + 20
  const range = Math.max(1, maxR - minR)

  const width = 640
  const height = 180

  return pts.map((pt, i) => {
    const x = pts.length > 1 ? (i / (pts.length - 1)) * width : width / 2
    const y = height - ((pt.rating - minR) / range) * height
    return { x, y, pt }
  })
})

const chartSvgPath = computed(() => {
  const pts = chartPoints.value
  if (pts.length === 0) return ''
  return pts.reduce((acc, curr, idx) => {
    return `${acc} ${idx === 0 ? 'M' : 'L'} ${curr.x.toFixed(1)},${curr.y.toFixed(1)}`
  }, '')
})

const ratingDiff = computed(() => {
  if (!ratingResult.value) return 0
  return ratingResult.value.endRating - ratingResult.value.startRating
})
</script>

<template>
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/75 backdrop-blur-sm p-4 animate-fade-in">
    <div class="bg-zinc-900 border border-zinc-800 rounded-xl shadow-2xl w-full max-w-5xl flex flex-col max-h-[92vh] overflow-hidden">
      <!-- Modal Header -->
      <div class="px-5 py-3 border-b border-zinc-800 flex items-center justify-between bg-zinc-950/60">
        <div class="flex items-center gap-3">
          <div class="p-2 rounded-lg bg-emerald-500/10 border border-emerald-500/20 text-emerald-400">
            <BarChart3 class="w-5 h-5" />
          </div>
          <div>
            <h2 class="text-base font-semibold text-zinc-100 flex items-center gap-2">
              Database Statistics & Demographics
              <span class="text-xs px-2 py-0.5 rounded-full bg-emerald-500/20 text-emerald-300 font-mono">
                {{ dbName }}
              </span>
            </h2>
            <p class="text-xs text-zinc-400">
              Aggregated demographic breakdowns, Elo distribution, and player historical rating curves.
            </p>
          </div>
        </div>

        <!-- Tab Switcher -->
        <div class="flex items-center gap-1 bg-zinc-800/80 p-1 rounded-lg border border-zinc-700/60 text-xs">
          <button
            class="px-3 py-1 rounded-md font-medium transition-colors flex items-center gap-1.5 cursor-pointer"
            :class="activeTab === 'demographics' ? 'bg-emerald-600 text-white shadow-xs' : 'text-zinc-400 hover:text-zinc-200'"
            @click="activeTab = 'demographics'"
          >
            <BarChart3 class="w-3.5 h-3.5" />
            <span>Demographics</span>
          </button>
          <button
            class="px-3 py-1 rounded-md font-medium transition-colors flex items-center gap-1.5 cursor-pointer"
            :class="activeTab === 'rating' ? 'bg-emerald-600 text-white shadow-xs' : 'text-zinc-400 hover:text-zinc-200'"
            @click="activeTab = 'rating'"
          >
            <TrendingUp class="w-3.5 h-3.5" />
            <span>Rating Tracker</span>
          </button>
        </div>

        <button
          class="p-1.5 rounded-lg text-zinc-400 hover:text-zinc-200 hover:bg-zinc-800 transition-colors cursor-pointer"
          @click="emit('close')"
        >
          <X class="w-5 h-5" />
        </button>
      </div>

      <!-- Main Body -->
      <div class="flex-1 overflow-y-auto p-5">
        <!-- TAB 1: DEMOGRAPHICS -->
        <div v-if="activeTab === 'demographics'" class="flex flex-col gap-5">
          <!-- Overview Cards -->
          <div class="grid grid-cols-2 md:grid-cols-4 gap-3">
            <div class="bg-zinc-800/40 border border-zinc-700/40 rounded-xl p-3.5 flex flex-col">
              <span class="text-[11px] uppercase tracking-wider text-zinc-500 font-semibold">Total Games</span>
              <span class="text-2xl font-bold text-zinc-100 font-mono mt-1">
                {{ demographics?.totalGames || 0 }}
              </span>
              <span class="text-[11px] text-zinc-400 mt-0.5">
                {{ demographics?.ratedGames || 0 }} rated games
              </span>
            </div>

            <div class="bg-zinc-800/40 border border-zinc-700/40 rounded-xl p-3.5 flex flex-col">
              <span class="text-[11px] uppercase tracking-wider text-zinc-500 font-semibold">Average Rating</span>
              <span class="text-2xl font-bold text-emerald-400 font-mono mt-1">
                {{ demographics?.avgRating ? `${demographics.avgRating} Elo` : 'Unrated' }}
              </span>
              <span class="text-[11px] text-zinc-400 mt-0.5">Player average Elo</span>
            </div>

            <div class="bg-zinc-800/40 border border-zinc-700/40 rounded-xl p-3.5 flex flex-col">
              <span class="text-[11px] uppercase tracking-wider text-zinc-500 font-semibold">Average Length</span>
              <span class="text-2xl font-bold text-sky-400 font-mono mt-1">
                {{ demographics?.avgMoves ? `${demographics.avgMoves.toFixed(1)} moves` : '0 moves' }}
              </span>
              <span class="text-[11px] text-zinc-400 mt-0.5">Per game duration</span>
            </div>

            <div class="bg-zinc-800/40 border border-zinc-700/40 rounded-xl p-3.5 flex flex-col">
              <span class="text-[11px] uppercase tracking-wider text-zinc-500 font-semibold">Game Results</span>
              <div class="flex items-center gap-1.5 mt-2">
                <span class="px-2 py-0.5 rounded text-xs bg-zinc-200 text-zinc-900 font-bold font-mono" title="White Wins">
                  {{ Math.round(demographics?.whiteWinPct || 0) }}%
                </span>
                <span class="px-2 py-0.5 rounded text-xs bg-zinc-700 text-zinc-200 font-bold font-mono" title="Draws">
                  {{ Math.round(demographics?.drawPct || 0) }}%
                </span>
                <span class="px-2 py-0.5 rounded text-xs bg-zinc-950 text-zinc-300 border border-zinc-800 font-bold font-mono" title="Black Wins">
                  {{ Math.round(demographics?.blackWinPct || 0) }}%
                </span>
              </div>
              <span class="text-[11px] text-zinc-400 mt-1">White / Draw / Black</span>
            </div>
          </div>

          <!-- Charts Grid -->
          <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <!-- Elo Rating Distribution -->
            <div class="bg-zinc-850/60 border border-zinc-800 rounded-xl p-4 flex flex-col">
              <h3 class="text-xs font-semibold text-zinc-200 uppercase tracking-wider mb-3 flex items-center gap-1.5">
                <Award class="w-4 h-4 text-emerald-400" />
                Elo Rating Distribution
              </h3>
              <div class="flex-1 flex flex-col gap-2">
                <div
                  v-for="bin in demographics?.eloDist || []"
                  :key="bin.label"
                  class="flex items-center gap-3 text-xs"
                >
                  <span class="w-24 text-zinc-400 font-mono text-[11px] shrink-0">{{ bin.label }}</span>
                  <div class="flex-1 bg-zinc-900 rounded-full h-3 overflow-hidden border border-zinc-800 flex">
                    <div
                      class="bg-emerald-500 h-full rounded-full transition-all duration-500"
                      :style="{ width: `${bin.pct}%` }"
                    />
                  </div>
                  <span class="w-16 text-right font-mono text-[11px] text-zinc-300 shrink-0">
                    {{ bin.count }} ({{ Math.round(bin.pct) }}%)
                  </span>
                </div>
              </div>
            </div>

            <!-- Decade Distribution -->
            <div class="bg-zinc-850/60 border border-zinc-800 rounded-xl p-4 flex flex-col">
              <h3 class="text-xs font-semibold text-zinc-200 uppercase tracking-wider mb-3 flex items-center gap-1.5">
                <Calendar class="w-4 h-4 text-amber-400" />
                Chronological Decades
              </h3>
              <div class="flex-1 flex flex-col gap-2">
                <div
                  v-for="bin in demographics?.decadeDist || []"
                  :key="bin.label"
                  class="flex items-center gap-3 text-xs"
                >
                  <span class="w-20 text-zinc-400 font-mono text-[11px] shrink-0">{{ bin.label }}</span>
                  <div class="flex-1 bg-zinc-900 rounded-full h-3 overflow-hidden border border-zinc-800 flex">
                    <div
                      class="bg-amber-500 h-full rounded-full transition-all duration-500"
                      :style="{ width: `${bin.pct}%` }"
                    />
                  </div>
                  <span class="w-16 text-right font-mono text-[11px] text-zinc-300 shrink-0">
                    {{ bin.count }} ({{ Math.round(bin.pct) }}%)
                  </span>
                </div>
              </div>
            </div>

            <!-- ECO Sections -->
            <div class="bg-zinc-850/60 border border-zinc-800 rounded-xl p-4 flex flex-col">
              <h3 class="text-xs font-semibold text-zinc-200 uppercase tracking-wider mb-3 flex items-center gap-1.5">
                <Layers class="w-4 h-4 text-sky-400" />
                ECO Opening Sections
              </h3>
              <div class="flex-1 flex flex-col gap-2">
                <div
                  v-for="bin in demographics?.ecoSectionDist || []"
                  :key="bin.label"
                  class="flex items-center gap-3 text-xs"
                >
                  <span class="w-36 text-zinc-300 font-medium text-[11px] truncate shrink-0">{{ bin.label }}</span>
                  <div class="flex-1 bg-zinc-900 rounded-full h-3 overflow-hidden border border-zinc-800 flex">
                    <div
                      class="bg-sky-500 h-full rounded-full transition-all duration-500"
                      :style="{ width: `${bin.pct}%` }"
                    />
                  </div>
                  <span class="w-16 text-right font-mono text-[11px] text-zinc-300 shrink-0">
                    {{ bin.count }} ({{ Math.round(bin.pct) }}%)
                  </span>
                </div>
              </div>
            </div>

            <!-- Game Move Lengths -->
            <div class="bg-zinc-850/60 border border-zinc-800 rounded-xl p-4 flex flex-col">
              <h3 class="text-xs font-semibold text-zinc-200 uppercase tracking-wider mb-3 flex items-center gap-1.5">
                <Clock class="w-4 h-4 text-purple-400" />
                Move Length Distribution
              </h3>
              <div class="flex-1 flex flex-col gap-2">
                <div
                  v-for="bin in demographics?.moveLengthDist || []"
                  :key="bin.label"
                  class="flex items-center gap-3 text-xs"
                >
                  <span class="w-20 text-zinc-400 font-mono text-[11px] shrink-0">{{ bin.label }} moves</span>
                  <div class="flex-1 bg-zinc-900 rounded-full h-3 overflow-hidden border border-zinc-800 flex">
                    <div
                      class="bg-purple-500 h-full rounded-full transition-all duration-500"
                      :style="{ width: `${bin.pct}%` }"
                    />
                  </div>
                  <span class="w-16 text-right font-mono text-[11px] text-zinc-300 shrink-0">
                    {{ bin.count }} ({{ Math.round(bin.pct) }}%)
                  </span>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- TAB 2: RATING TRACKER -->
        <div v-else class="flex flex-col gap-4">
          <!-- Player Selector & Search -->
          <div class="flex items-center gap-3 bg-zinc-850/60 border border-zinc-800 rounded-xl p-3 flex-wrap">
            <div class="flex-1 min-w-[240px] relative">
              <Search class="w-4 h-4 text-zinc-400 absolute left-3 top-2.5" />
              <input
                v-model="playerQuery"
                type="text"
                placeholder="Enter player name or persona (e.g. Kasparov, Morphy)..."
                class="w-full bg-zinc-900 border border-zinc-700 rounded-lg pl-9 pr-3 py-1.5 text-xs text-zinc-100 placeholder-zinc-500 focus:outline-none focus:border-emerald-500"
                @keydown.enter="loadRatingEvolution(playerQuery)"
              />
            </div>
            <button
              class="px-4 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white font-medium text-xs transition-colors cursor-pointer"
              @click="loadRatingEvolution(playerQuery)"
            >
              Track Evolution
            </button>

            <!-- Quick Candidate Chips -->
            <div v-if="candidatePlayers.length > 0" class="flex items-center gap-1.5 flex-wrap">
              <span class="text-[11px] text-zinc-500">Popular:</span>
              <button
                v-for="p in candidatePlayers.slice(0, 4)"
                :key="p"
                class="px-2 py-0.5 rounded-full bg-zinc-800 hover:bg-zinc-700 text-zinc-300 text-[11px] border border-zinc-700 transition-colors cursor-pointer"
                @click="loadRatingEvolution(p)"
              >
                {{ p }}
              </button>
            </div>
          </div>

          <!-- Player Rating Summary Cards -->
          <div v-if="ratingResult" class="grid grid-cols-2 md:grid-cols-4 gap-3">
            <div class="bg-zinc-800/40 border border-zinc-700/40 rounded-xl p-3 flex flex-col">
              <span class="text-[10px] uppercase text-zinc-500 font-semibold">Start Rating</span>
              <span class="text-xl font-bold text-zinc-200 font-mono mt-0.5">
                {{ ratingResult.startRating || 'N/A' }}
              </span>
              <span class="text-[10px] text-zinc-400">First recorded game</span>
            </div>

            <div class="bg-zinc-800/40 border border-zinc-700/40 rounded-xl p-3 flex flex-col">
              <span class="text-[10px] uppercase text-zinc-500 font-semibold">Peak Elo</span>
              <span class="text-xl font-bold text-amber-400 font-mono mt-0.5">
                {{ ratingResult.peakRating || 'N/A' }}
              </span>
              <span class="text-[10px] text-zinc-400">Achieved on {{ ratingResult.peakDate || 'Unknown' }}</span>
            </div>

            <div class="bg-zinc-800/40 border border-zinc-700/40 rounded-xl p-3 flex flex-col">
              <span class="text-[10px] uppercase text-zinc-500 font-semibold">Current / End Rating</span>
              <span class="text-xl font-bold text-zinc-200 font-mono mt-0.5">
                {{ ratingResult.endRating || 'N/A' }}
              </span>
              <span class="text-[10px] text-zinc-400">Latest recorded game</span>
            </div>

            <div class="bg-zinc-800/40 border border-zinc-700/40 rounded-xl p-3 flex flex-col">
              <span class="text-[10px] uppercase text-zinc-500 font-semibold">Net Progress</span>
              <div class="flex items-center gap-1 mt-0.5">
                <ArrowUpRight v-if="ratingDiff > 0" class="w-4 h-4 text-emerald-400" />
                <ArrowDownRight v-else-if="ratingDiff < 0" class="w-4 h-4 text-rose-400" />
                <Minus v-else class="w-4 h-4 text-zinc-400" />
                <span
                  class="text-xl font-bold font-mono"
                  :class="ratingDiff > 0 ? 'text-emerald-400' : ratingDiff < 0 ? 'text-rose-400' : 'text-zinc-400'"
                >
                  {{ ratingDiff > 0 ? `+${ratingDiff}` : ratingDiff }}
                </span>
              </div>
              <span class="text-[10px] text-zinc-400">{{ ratingResult.ratedGames }} rated games</span>
            </div>
          </div>

          <!-- Interactive SVG Rating Evolution Chart -->
          <div v-if="ratingResult && ratingResult.points.length > 0" class="bg-zinc-850/60 border border-zinc-800 rounded-xl p-4 flex flex-col">
            <div class="flex items-center justify-between mb-2">
              <h3 class="text-xs font-semibold text-zinc-200 uppercase tracking-wider flex items-center gap-1.5">
                <TrendingUp class="w-4 h-4 text-emerald-400" />
                Historical Elo Rating Curve
              </h3>
              <span class="text-xs text-zinc-400 font-mono">
                Min: {{ ratingResult.minRating }} • Max: {{ ratingResult.maxRating }}
              </span>
            </div>

            <!-- SVG Chart Area -->
            <div class="w-full overflow-x-auto py-2">
              <svg
                viewBox="0 0 640 180"
                class="w-full h-48 overflow-visible"
              >
                <!-- Grid Lines -->
                <line x1="0" y1="30" x2="640" y2="30" stroke="#27272a" stroke-dasharray="4" />
                <line x1="0" y1="90" x2="640" y2="90" stroke="#27272a" stroke-dasharray="4" />
                <line x1="0" y1="150" x2="640" y2="150" stroke="#27272a" stroke-dasharray="4" />

                <!-- Trend Line -->
                <path
                  :d="chartSvgPath"
                  fill="none"
                  stroke="#10b981"
                  stroke-width="2.5"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                />

                <!-- Interactive Points -->
                <g v-for="(p, idx) in chartPoints" :key="idx">
                  <circle
                    :cx="p.x"
                    :cy="p.y"
                    r="4"
                    class="fill-emerald-400 stroke-zinc-900 stroke-2 hover:r-6 hover:fill-emerald-300 transition-all cursor-pointer"
                    @mouseenter="hoveredPoint = p.pt"
                  />
                </g>
              </svg>
            </div>

            <!-- Point Hover Details Tooltip / Card -->
            <div
              v-if="hoveredPoint"
              class="p-2.5 rounded-lg bg-zinc-900 border border-emerald-500/50 text-xs flex items-center justify-between animate-fade-in"
            >
              <div class="flex items-center gap-2">
                <span class="font-bold text-emerald-400 font-mono text-sm">{{ hoveredPoint.rating }} Elo</span>
                <span class="text-zinc-300">vs {{ hoveredPoint.opponent }} ({{ hoveredPoint.opponentElo || 'Unrated' }})</span>
                <span class="px-1.5 py-0.5 rounded text-[10px] font-mono bg-zinc-800 text-zinc-300 font-bold">
                  {{ hoveredPoint.result }}
                </span>
                <span class="text-zinc-500 font-mono text-[11px]">{{ hoveredPoint.eco }}</span>
              </div>
              <div class="text-zinc-400 text-[11px]">
                {{ hoveredPoint.date }} • {{ hoveredPoint.event }}
              </div>
            </div>
            <div v-else class="text-[11px] text-zinc-500 italic text-center py-1">
              Hover over points to inspect specific games, opponents, and dates.
            </div>
          </div>

          <div
            v-else-if="!loadingRating"
            class="p-8 text-center text-zinc-500 text-xs border border-dashed border-zinc-800 rounded-xl"
          >
            No rated games found for "{{ playerQuery }}". Enter another player name above.
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
