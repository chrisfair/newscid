<script setup lang="ts">
import { ref, reactive, watch, onMounted } from 'vue'
import {
  Search,
  X,
  RotateCcw,
  Database,
  User,
  Calendar,
  Trophy,
  Filter,
  CheckCircle2,
  Sliders,
  FileSearch,
  Hash
} from '@lucide/vue'
import { CountSearchMatches } from '../../wailsjs/go/main/App'
import { scid } from '../../wailsjs/go/models'

const props = defineProps<{
  dbName?: string
  totalGames: number
  initialCriteria?: Partial<scid.SearchCriteria>
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'applySearch', criteria: scid.SearchCriteria): void
}>()

const criteria = reactive<scid.SearchCriteria>({
  white: '',
  black: '',
  player: '',
  event: '',
  site: '',
  round: '',
  dateFrom: '',
  dateTo: '',
  yearFrom: undefined,
  yearTo: undefined,
  result: '',
  eco: '',
  ecoFrom: '',
  ecoTo: '',
  minElo: undefined,
  maxElo: undefined,
  whiteEloMin: undefined,
  whiteEloMax: undefined,
  blackEloMin: undefined,
  blackEloMax: undefined,
  minMoves: undefined,
  maxMoves: undefined,
  hasVariations: false,
  ignoreDeleted: true,
  onlyDeleted: false
})

const isCounting = ref(false)
const matchCount = ref<number | null>(null)

onMounted(() => {
  if (props.initialCriteria) {
    Object.assign(criteria, props.initialCriteria)
  }
})

function resetFilters() {
  criteria.white = ''
  criteria.black = ''
  criteria.player = ''
  criteria.event = ''
  criteria.site = ''
  criteria.round = ''
  criteria.dateFrom = ''
  criteria.dateTo = ''
  criteria.yearFrom = undefined
  criteria.yearTo = undefined
  criteria.result = ''
  criteria.eco = ''
  criteria.ecoFrom = ''
  criteria.ecoTo = ''
  criteria.minElo = undefined
  criteria.maxElo = undefined
  criteria.whiteEloMin = undefined
  criteria.whiteEloMax = undefined
  criteria.blackEloMin = undefined
  criteria.blackEloMax = undefined
  criteria.minMoves = undefined
  criteria.maxMoves = undefined
  criteria.hasVariations = false
  criteria.ignoreDeleted = true
  criteria.onlyDeleted = false
  matchCount.value = null
}

function cleanCriteria(): scid.SearchCriteria {
  const c: any = {}
  if (criteria.player?.trim()) c.player = criteria.player.trim()
  if (criteria.white?.trim()) c.white = criteria.white.trim()
  if (criteria.black?.trim()) c.black = criteria.black.trim()
  if (criteria.event?.trim()) c.event = criteria.event.trim()
  if (criteria.site?.trim()) c.site = criteria.site.trim()
  if (criteria.round?.trim()) c.round = criteria.round.trim()
  if (criteria.dateFrom?.trim()) c.dateFrom = criteria.dateFrom.trim()
  if (criteria.dateTo?.trim()) c.dateTo = criteria.dateTo.trim()
  if (criteria.yearFrom && criteria.yearFrom > 0) c.yearFrom = Number(criteria.yearFrom)
  if (criteria.yearTo && criteria.yearTo > 0) c.yearTo = Number(criteria.yearTo)
  if (criteria.result) c.result = criteria.result
  if (criteria.eco?.trim()) c.eco = criteria.eco.trim()
  if (criteria.ecoFrom?.trim()) c.ecoFrom = criteria.ecoFrom.trim()
  if (criteria.ecoTo?.trim()) c.ecoTo = criteria.ecoTo.trim()
  if (criteria.minElo && criteria.minElo > 0) c.minElo = Number(criteria.minElo)
  if (criteria.maxElo && criteria.maxElo > 0) c.maxElo = Number(criteria.maxElo)
  if (criteria.whiteEloMin && criteria.whiteEloMin > 0) c.whiteEloMin = Number(criteria.whiteEloMin)
  if (criteria.whiteEloMax && criteria.whiteEloMax > 0) c.whiteEloMax = Number(criteria.whiteEloMax)
  if (criteria.blackEloMin && criteria.blackEloMin > 0) c.blackEloMin = Number(criteria.blackEloMin)
  if (criteria.blackEloMax && criteria.blackEloMax > 0) c.blackEloMax = Number(criteria.blackEloMax)
  if (criteria.minMoves && criteria.minMoves > 0) c.minMoves = Number(criteria.minMoves)
  if (criteria.maxMoves && criteria.maxMoves > 0) c.maxMoves = Number(criteria.maxMoves)
  if (criteria.hasVariations) c.hasVariations = true
  if (criteria.onlyDeleted) c.onlyDeleted = true
  if (criteria.ignoreDeleted !== undefined) c.ignoreDeleted = criteria.ignoreDeleted
  return c
}

async function handleCheckCount() {
  try {
    isCounting.value = true
    const c = cleanCriteria()
    const count = await CountSearchMatches(c)
    matchCount.value = count
  } catch (err) {
    console.error('Failed to count search matches:', err)
  } finally {
    isCounting.value = false
  }
}

function handleSearch() {
  const c = cleanCriteria()
  emit('applySearch', c)
  emit('close')
}
</script>

<template>
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/75 backdrop-blur-xs p-4">
    <div
      class="bg-zinc-900 border border-zinc-700 rounded-xl shadow-2xl w-full max-w-3xl max-h-[90vh] flex flex-col overflow-hidden text-zinc-200 animate-in fade-in zoom-in-95 duration-150"
      @keydown.esc="emit('close')"
    >
      <!-- Header -->
      <div class="flex items-center justify-between px-5 py-3.5 bg-zinc-950 border-b border-zinc-800">
        <div class="flex items-center gap-2.5">
          <div class="w-8 h-8 rounded-lg bg-emerald-500/20 border border-emerald-500/40 flex items-center justify-center text-emerald-400">
            <FileSearch class="w-4 h-4" />
          </div>
          <div>
            <h2 class="font-bold text-base text-zinc-100 flex items-center gap-2">
              Header / General Database Search
              <span class="text-xs px-2 py-0.5 rounded-full bg-zinc-800 border border-zinc-700 text-zinc-400 font-mono font-normal">
                Scid vs PC Filter
              </span>
            </h2>
            <div class="flex items-center gap-2 text-xs text-zinc-400 mt-0.5">
              <Database class="w-3.5 h-3.5 text-zinc-500" />
              <span>Target Database: <strong class="text-emerald-400">{{ dbName || 'No Database' }}</strong> ({{ totalGames.toLocaleString() }} total games)</span>
            </div>
          </div>
        </div>
        <button
          class="p-1.5 rounded-lg text-zinc-400 hover:text-white hover:bg-zinc-800 transition-colors cursor-pointer"
          @click="emit('close')"
        >
          <X class="w-4 h-4" />
        </button>
      </div>

      <!-- Body / Form Fields (scrollable) -->
      <div class="p-5 overflow-y-auto space-y-5 text-xs flex-1">
        <!-- 1. PLAYERS & RATINGS -->
        <div class="p-4 rounded-lg bg-zinc-950/70 border border-zinc-800 space-y-3">
          <div class="flex items-center gap-2 text-sm font-semibold text-zinc-200 border-b border-zinc-800/80 pb-2">
            <User class="w-4 h-4 text-emerald-400" />
            <span>Players & Rating Limits</span>
          </div>

          <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
            <!-- Either Player -->
            <div class="md:col-span-2">
              <label class="block text-zinc-400 mb-1">Either Player (White or Black contains)</label>
              <input
                v-model="criteria.player"
                type="text"
                placeholder="e.g. Kasparov or Fischer"
                class="w-full bg-zinc-900 border border-zinc-700 rounded-lg px-3 py-1.5 text-xs text-zinc-200 focus:outline-hidden focus:border-emerald-500"
              />
            </div>

            <!-- White Player & Elo -->
            <div>
              <label class="block text-zinc-400 mb-1">White Player Name</label>
              <input
                v-model="criteria.white"
                type="text"
                placeholder="White player..."
                class="w-full bg-zinc-900 border border-zinc-700 rounded-lg px-3 py-1.5 text-xs text-zinc-200 focus:outline-hidden focus:border-emerald-500 mb-2"
              />
              <div class="flex items-center gap-2">
                <div class="flex-1">
                  <label class="block text-[11px] text-zinc-500 mb-0.5">White Min Elo</label>
                  <input
                    v-model.number="criteria.whiteEloMin"
                    type="number"
                    placeholder="Min"
                    class="w-full bg-zinc-900 border border-zinc-700 rounded-lg px-2.5 py-1 text-xs text-zinc-200 focus:outline-hidden focus:border-emerald-500"
                  />
                </div>
                <div class="flex-1">
                  <label class="block text-[11px] text-zinc-500 mb-0.5">White Max Elo</label>
                  <input
                    v-model.number="criteria.whiteEloMax"
                    type="number"
                    placeholder="Max"
                    class="w-full bg-zinc-900 border border-zinc-700 rounded-lg px-2.5 py-1 text-xs text-zinc-200 focus:outline-hidden focus:border-emerald-500"
                  />
                </div>
              </div>
            </div>

            <!-- Black Player & Elo -->
            <div>
              <label class="block text-zinc-400 mb-1">Black Player Name</label>
              <input
                v-model="criteria.black"
                type="text"
                placeholder="Black player..."
                class="w-full bg-zinc-900 border border-zinc-700 rounded-lg px-3 py-1.5 text-xs text-zinc-200 focus:outline-hidden focus:border-emerald-500 mb-2"
              />
              <div class="flex items-center gap-2">
                <div class="flex-1">
                  <label class="block text-[11px] text-zinc-500 mb-0.5">Black Min Elo</label>
                  <input
                    v-model.number="criteria.blackEloMin"
                    type="number"
                    placeholder="Min"
                    class="w-full bg-zinc-900 border border-zinc-700 rounded-lg px-2.5 py-1 text-xs text-zinc-200 focus:outline-hidden focus:border-emerald-500"
                  />
                </div>
                <div class="flex-1">
                  <label class="block text-[11px] text-zinc-500 mb-0.5">Black Max Elo</label>
                  <input
                    v-model.number="criteria.blackEloMax"
                    type="number"
                    placeholder="Max"
                    class="w-full bg-zinc-900 border border-zinc-700 rounded-lg px-2.5 py-1 text-xs text-zinc-200 focus:outline-hidden focus:border-emerald-500"
                  />
                </div>
              </div>
            </div>

            <!-- Overall Elo filter -->
            <div class="md:col-span-2 flex items-center gap-3 pt-1 border-t border-zinc-800/60">
              <span class="text-zinc-400 whitespace-nowrap">Overall Rating Filter:</span>
              <div class="flex items-center gap-2 flex-1">
                <input
                  v-model.number="criteria.minElo"
                  type="number"
                  placeholder="Min Elo (either player)"
                  class="flex-1 bg-zinc-900 border border-zinc-700 rounded-lg px-2.5 py-1 text-xs text-zinc-200 focus:outline-hidden focus:border-emerald-500"
                />
                <span class="text-zinc-500">to</span>
                <input
                  v-model.number="criteria.maxElo"
                  type="number"
                  placeholder="Max Elo (either player)"
                  class="flex-1 bg-zinc-900 border border-zinc-700 rounded-lg px-2.5 py-1 text-xs text-zinc-200 focus:outline-hidden focus:border-emerald-500"
                />
              </div>
            </div>
          </div>
        </div>

        <!-- 2. EVENT, SITE & DATES -->
        <div class="p-4 rounded-lg bg-zinc-950/70 border border-zinc-800 space-y-3">
          <div class="flex items-center gap-2 text-sm font-semibold text-zinc-200 border-b border-zinc-800/80 pb-2">
            <Trophy class="w-4 h-4 text-amber-400" />
            <span>Event, Location & Date Range</span>
          </div>

          <div class="grid grid-cols-1 md:grid-cols-3 gap-3">
            <div>
              <label class="block text-zinc-400 mb-1">Event</label>
              <input
                v-model="criteria.event"
                type="text"
                placeholder="e.g. Olympiad, Candidates"
                class="w-full bg-zinc-900 border border-zinc-700 rounded-lg px-3 py-1.5 text-xs text-zinc-200 focus:outline-hidden focus:border-emerald-500"
              />
            </div>
            <div>
              <label class="block text-zinc-400 mb-1">Site / City</label>
              <input
                v-model="criteria.site"
                type="text"
                placeholder="e.g. London, Moscow"
                class="w-full bg-zinc-900 border border-zinc-700 rounded-lg px-3 py-1.5 text-xs text-zinc-200 focus:outline-hidden focus:border-emerald-500"
              />
            </div>
            <div>
              <label class="block text-zinc-400 mb-1">Round</label>
              <input
                v-model="criteria.round"
                type="text"
                placeholder="e.g. 1 or Finals"
                class="w-full bg-zinc-900 border border-zinc-700 rounded-lg px-3 py-1.5 text-xs text-zinc-200 focus:outline-hidden focus:border-emerald-500"
              />
            </div>
          </div>

          <!-- Date / Year Range -->
          <div class="grid grid-cols-1 md:grid-cols-2 gap-3 pt-1 border-t border-zinc-800/60">
            <div>
              <label class="block text-zinc-400 mb-1">Year Range</label>
              <div class="flex items-center gap-2">
                <input
                  v-model.number="criteria.yearFrom"
                  type="number"
                  placeholder="From (e.g. 1970)"
                  class="w-full bg-zinc-900 border border-zinc-700 rounded-lg px-2.5 py-1 text-xs text-zinc-200 focus:outline-hidden focus:border-emerald-500"
                />
                <span class="text-zinc-500">—</span>
                <input
                  v-model.number="criteria.yearTo"
                  type="number"
                  placeholder="To (e.g. 2024)"
                  class="w-full bg-zinc-900 border border-zinc-700 rounded-lg px-2.5 py-1 text-xs text-zinc-200 focus:outline-hidden focus:border-emerald-500"
                />
              </div>
            </div>
            <div>
              <label class="block text-zinc-400 mb-1">Specific Date (YYYY.MM.DD)</label>
              <div class="flex items-center gap-2">
                <input
                  v-model="criteria.dateFrom"
                  type="text"
                  placeholder="From YYYY.MM.DD"
                  class="w-full bg-zinc-900 border border-zinc-700 rounded-lg px-2.5 py-1 text-xs text-zinc-200 focus:outline-hidden focus:border-emerald-500"
                />
                <span class="text-zinc-500">—</span>
                <input
                  v-model="criteria.dateTo"
                  type="text"
                  placeholder="To YYYY.MM.DD"
                  class="w-full bg-zinc-900 border border-zinc-700 rounded-lg px-2.5 py-1 text-xs text-zinc-200 focus:outline-hidden focus:border-emerald-500"
                />
              </div>
            </div>
          </div>
        </div>

        <!-- 3. GAME DETAILS & ECO -->
        <div class="p-4 rounded-lg bg-zinc-950/70 border border-zinc-800 space-y-3">
          <div class="flex items-center gap-2 text-sm font-semibold text-zinc-200 border-b border-zinc-800/80 pb-2">
            <Sliders class="w-4 h-4 text-blue-400" />
            <span>Game Results, ECO & Move Length</span>
          </div>

          <div class="grid grid-cols-1 md:grid-cols-3 gap-3">
            <!-- Result -->
            <div>
              <label class="block text-zinc-400 mb-1">Game Result</label>
              <select
                v-model="criteria.result"
                class="w-full bg-zinc-900 border border-zinc-700 rounded-lg px-3 py-1.5 text-xs text-zinc-200 focus:outline-hidden focus:border-emerald-500"
              >
                <option value="">Any Result</option>
                <option value="1-0">1-0 (White Wins)</option>
                <option value="0-1">0-1 (Black Wins)</option>
                <option value="1/2-1/2">1/2-1/2 (Draw)</option>
                <option value="*">* (Underway / Unknown)</option>
              </select>
            </div>

            <!-- ECO Code / Prefix -->
            <div>
              <label class="block text-zinc-400 mb-1">ECO Code / Prefix</label>
              <input
                v-model="criteria.eco"
                type="text"
                placeholder="e.g. B20 or B"
                class="w-full bg-zinc-900 border border-zinc-700 rounded-lg px-3 py-1.5 text-xs text-zinc-200 focus:outline-hidden focus:border-emerald-500"
              />
            </div>

            <!-- ECO Range -->
            <div>
              <label class="block text-zinc-400 mb-1">ECO Range (From — To)</label>
              <div class="flex items-center gap-1.5">
                <input
                  v-model="criteria.ecoFrom"
                  type="text"
                  placeholder="e.g. B00"
                  class="w-full bg-zinc-900 border border-zinc-700 rounded-lg px-2 py-1.5 text-xs text-zinc-200 focus:outline-hidden focus:border-emerald-500"
                />
                <span class="text-zinc-500">—</span>
                <input
                  v-model="criteria.ecoTo"
                  type="text"
                  placeholder="e.g. B99"
                  class="w-full bg-zinc-900 border border-zinc-700 rounded-lg px-2 py-1.5 text-xs text-zinc-200 focus:outline-hidden focus:border-emerald-500"
                />
              </div>
            </div>

            <!-- Moves Length -->
            <div class="md:col-span-3">
              <label class="block text-zinc-400 mb-1">Game Length (Number of Full Moves)</label>
              <div class="flex items-center gap-3">
                <input
                  v-model.number="criteria.minMoves"
                  type="number"
                  placeholder="Min Moves (e.g. 15)"
                  class="flex-1 bg-zinc-900 border border-zinc-700 rounded-lg px-3 py-1.5 text-xs text-zinc-200 focus:outline-hidden focus:border-emerald-500"
                />
                <span class="text-zinc-500">to</span>
                <input
                  v-model.number="criteria.maxMoves"
                  type="number"
                  placeholder="Max Moves (e.g. 60)"
                  class="flex-1 bg-zinc-900 border border-zinc-700 rounded-lg px-3 py-1.5 text-xs text-zinc-200 focus:outline-hidden focus:border-emerald-500"
                />
              </div>
            </div>
          </div>
        </div>

        <!-- 4. FLAGS & OPTIONS -->
        <div class="p-4 rounded-lg bg-zinc-950/70 border border-zinc-800 flex flex-wrap items-center gap-6">
          <label class="flex items-center gap-2 cursor-pointer select-none">
            <input
              v-model="criteria.ignoreDeleted"
              type="checkbox"
              class="rounded border-zinc-700 text-emerald-500 focus:ring-0 focus:outline-hidden"
            />
            <span class="text-zinc-300">Exclude deleted games</span>
          </label>

          <label class="flex items-center gap-2 cursor-pointer select-none">
            <input
              v-model="criteria.hasVariations"
              type="checkbox"
              class="rounded border-zinc-700 text-emerald-500 focus:ring-0 focus:outline-hidden"
            />
            <span class="text-zinc-300">Must contain variations / commentary</span>
          </label>

          <label class="flex items-center gap-2 cursor-pointer select-none">
            <input
              v-model="criteria.onlyDeleted"
              type="checkbox"
              class="rounded border-zinc-700 text-amber-500 focus:ring-0 focus:outline-hidden"
            />
            <span class="text-zinc-300">Only search deleted games</span>
          </label>
        </div>
      </div>

      <!-- Footer with Match Counter & Action Buttons -->
      <div class="flex items-center justify-between px-5 py-3.5 bg-zinc-950 border-t border-zinc-800">
        <div class="flex items-center gap-3">
          <button
            class="px-3 py-1.5 rounded-lg bg-zinc-800 hover:bg-zinc-700 text-zinc-300 font-medium transition-colors cursor-pointer flex items-center gap-1.5"
            :disabled="isCounting"
            @click="handleCheckCount"
          >
            <Hash class="w-3.5 h-3.5 text-emerald-400" />
            <span>{{ isCounting ? 'Scanning...' : 'Check Matches' }}</span>
          </button>

          <div v-if="matchCount !== null" class="flex items-center gap-1.5 px-3 py-1 rounded-full bg-emerald-500/10 border border-emerald-500/30 text-emerald-400 font-mono text-xs">
            <CheckCircle2 class="w-3.5 h-3.5" />
            <span>Found <strong>{{ matchCount.toLocaleString() }}</strong> matching games</span>
          </div>

          <button
            class="text-xs text-zinc-500 hover:text-zinc-300 underline cursor-pointer ml-1"
            @click="resetFilters"
          >
            Reset All
          </button>
        </div>

        <div class="flex items-center gap-2">
          <button
            class="px-4 py-1.5 rounded-lg bg-zinc-800 hover:bg-zinc-700 text-zinc-300 font-medium transition-colors cursor-pointer"
            @click="emit('close')"
          >
            Cancel
          </button>
          <button
            class="px-5 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white font-medium shadow-md shadow-emerald-950 transition-colors cursor-pointer flex items-center gap-2"
            @click="handleSearch"
          >
            <Search class="w-4 h-4" />
            <span>Search Database</span>
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
