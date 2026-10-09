<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import {
  Trophy,
  Award,
  Calendar,
  MapPin,
  Users,
  Swords,
  Copy,
  Check,
  Search,
  RefreshCw,
  FileText,
  ChevronDown,
  Grid,
  ListOrdered,
  X
} from '@lucide/vue'
import { GetTournamentEvents, GetTournamentCrosstable, ExportCrosstableText } from '../../wailsjs/go/main/App'
import { tournament } from '../../wailsjs/go/models'
import { ClipboardSetText } from '../../wailsjs/runtime/runtime'

const props = defineProps<{
  show: boolean
  activeEvent?: string
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'loadGame', gameId: number): void
}>()

// State
const loading = ref(false)
const eventsList = ref<tournament.EventSummary[]>([])
const selectedEvent = ref<tournament.EventSummary | null>(null)
const crosstable = ref<tournament.TournamentCrosstable | null>(null)
const activeView = ref<'matrix' | 'swiss' | 'standings' | 'text'>('matrix')
const eventSearchQuery = ref('')
const playerSearchQuery = ref('')
const copied = ref(false)
const showEventDropdown = ref(false)

// Filtered events for selector
const filteredEvents = computed(() => {
  const q = eventSearchQuery.value.trim().toLowerCase()
  if (!q) return eventsList.value
  return eventsList.value.filter(
    (ev) =>
      ev.event.toLowerCase().includes(q) ||
      (ev.site && ev.site.toLowerCase().includes(q)) ||
      (ev.date && ev.date.includes(q))
  )
})

// Filtered standings
const filteredStandings = computed(() => {
  if (!crosstable.value || !crosstable.value.standings) return []
  const q = playerSearchQuery.value.trim().toLowerCase()
  if (!q) return crosstable.value.standings
  return crosstable.value.standings.filter((p) =>
    p.playerName.toLowerCase().includes(q)
  )
})

async function fetchEvents(preferredEvent?: string) {
  loading.value = true
  try {
    const list = await GetTournamentEvents()
    eventsList.value = list || []

    if (eventsList.value.length > 0) {
      let match = null
      if (preferredEvent) {
        match = eventsList.value.find(
          (e) => e.event.toLowerCase() === preferredEvent.toLowerCase()
        )
      }
      if (!match) {
        match = eventsList.value[0]
      }
      await selectEvent(match)
    } else {
      selectedEvent.value = null
      crosstable.value = null
    }
  } catch (err) {
    console.error('Failed to load tournament events:', err)
  } finally {
    loading.value = false
  }
}

async function selectEvent(ev: tournament.EventSummary) {
  selectedEvent.value = ev
  showEventDropdown.value = false
  loading.value = true
  try {
    const ct = await GetTournamentCrosstable(ev.event, ev.site, ev.date)
    crosstable.value = ct

    // Auto-select view mode based on tournament type
    if (ct.type === 'Swiss') {
      activeView.value = 'swiss'
    } else {
      activeView.value = 'matrix'
    }
  } catch (err) {
    console.error('Failed to load crosstable for event:', ev.event, err)
  } finally {
    loading.value = false
  }
}

async function copyCrosstable() {
  if (!crosstable.value) return
  try {
    const txt = await ExportCrosstableText(
      crosstable.value.event,
      crosstable.value.site,
      crosstable.value.date
    )
    await ClipboardSetText(txt || crosstable.value.crossTableText)
    copied.value = true
    setTimeout(() => {
      copied.value = false
    }, 2000)
  } catch (err) {
    console.error('Failed to copy crosstable text:', err)
  }
}

function handleCellClick(gameId?: number) {
  if (gameId && gameId > 0) {
    emit('loadGame', gameId)
    emit('close')
  }
}

function getRankBadgeClass(rank: number): string {
  if (rank === 1) return 'bg-amber-500/20 text-amber-300 border-amber-500/40'
  if (rank === 2) return 'bg-zinc-300/20 text-zinc-200 border-zinc-400/40'
  if (rank === 3) return 'bg-amber-700/20 text-amber-400 border-amber-700/40'
  return 'bg-zinc-800 text-zinc-400 border-zinc-700'
}

function getResultCellClass(res: string): string {
  if (!res) return 'text-zinc-600'
  if (res === 'X') return 'bg-zinc-800/80 text-zinc-600 font-bold'
  if (res.includes('1') && !res.includes('0') && !res.includes('½')) {
    return 'bg-emerald-500/20 text-emerald-300 font-medium hover:bg-emerald-500/30'
  }
  if (res.includes('0') && !res.includes('1') && !res.includes('½')) {
    return 'bg-red-500/15 text-red-400 hover:bg-red-500/25'
  }
  if (res.includes('½')) {
    return 'bg-amber-500/15 text-amber-300 hover:bg-amber-500/25'
  }
  return 'text-zinc-300 hover:bg-zinc-700/50'
}

watch(
  () => props.show,
  (val) => {
    if (val) {
      fetchEvents(props.activeEvent)
    }
  }
)

onMounted(() => {
  if (props.show) {
    fetchEvents(props.activeEvent)
  }
})
</script>

<template>
  <div
    v-if="show"
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/75 backdrop-blur-sm p-4 animate-in fade-in duration-200"
    @click.self="emit('close')"
  >
    <div
      class="bg-zinc-900 border border-zinc-700/90 rounded-xl shadow-2xl w-full max-w-6xl max-h-[92vh] flex flex-col overflow-hidden text-zinc-200"
    >
      <!-- Modal Header -->
      <div class="px-6 py-4 border-b border-zinc-800 flex items-center justify-between bg-zinc-950/60">
        <div class="flex items-center gap-3">
          <div class="p-2 rounded-lg bg-amber-500/10 text-amber-400 border border-amber-500/20">
            <Trophy class="w-5 h-5" />
          </div>
          <div>
            <h2 class="text-base font-semibold text-zinc-100 flex items-center gap-2">
              Tournament Crosstables & Standings
            </h2>
            <p class="text-xs text-zinc-400">
              Round-Robin matrices, Swiss pairings, Sonneborn-Berger, and Buchholz tiebreak standings
            </p>
          </div>
        </div>

        <button
          class="p-1.5 rounded-lg text-zinc-400 hover:text-zinc-100 hover:bg-zinc-800 transition-colors cursor-pointer"
          @click="emit('close')"
        >
          <X class="w-5 h-5" />
        </button>
      </div>

      <!-- Controls & Event Selector Bar -->
      <div class="px-6 py-3 border-b border-zinc-800/80 bg-zinc-900/90 flex flex-wrap items-center justify-between gap-4">
        <!-- Event Dropdown -->
        <div class="relative flex-1 min-w-[300px] max-w-lg">
          <button
            class="w-full px-3 py-2 bg-zinc-800/80 hover:bg-zinc-800 border border-zinc-700 rounded-lg text-left text-xs font-medium text-zinc-200 flex items-center justify-between transition-colors cursor-pointer"
            @click="showEventDropdown = !showEventDropdown"
          >
            <span class="truncate flex items-center gap-2">
              <Calendar class="w-3.5 h-3.5 text-amber-400 shrink-0" />
              <span v-if="selectedEvent" class="truncate">
                {{ selectedEvent.event }}
                <span v-if="selectedEvent.site" class="text-zinc-400">({{ selectedEvent.site }})</span>
              </span>
              <span v-else class="text-zinc-500">Select tournament event...</span>
            </span>
            <ChevronDown class="w-4 h-4 text-zinc-400 shrink-0" />
          </button>

          <!-- Dropdown menu -->
          <div
            v-if="showEventDropdown"
            class="absolute top-full left-0 mt-1 w-full max-h-72 bg-zinc-900 border border-zinc-700 rounded-lg shadow-2xl py-1 z-50 flex flex-col overflow-hidden text-xs"
          >
            <div class="p-2 border-b border-zinc-800">
              <div class="relative">
                <Search class="w-3.5 h-3.5 absolute left-2.5 top-2.5 text-zinc-400" />
                <input
                  v-model="eventSearchQuery"
                  type="text"
                  placeholder="Filter events..."
                  class="w-full pl-8 pr-3 py-1.5 bg-zinc-950 border border-zinc-700 rounded text-xs text-zinc-200 placeholder-zinc-500 focus:outline-none focus:border-amber-500"
                />
              </div>
            </div>

            <div class="overflow-y-auto max-h-56 divide-y divide-zinc-800/50">
              <button
                v-for="ev in filteredEvents"
                :key="ev.event + ev.site + ev.date"
                class="w-full text-left px-3 py-2 hover:bg-zinc-800 flex items-center justify-between text-zinc-200 transition-colors cursor-pointer"
                :class="{ 'bg-zinc-800/60 font-semibold text-amber-300': selectedEvent?.event === ev.event }"
                @click="selectEvent(ev)"
              >
                <div class="truncate pr-2">
                  <div class="truncate text-zinc-100">{{ ev.event }}</div>
                  <div class="text-[11px] text-zinc-400 flex items-center gap-2 mt-0.5">
                    <span v-if="ev.site">{{ ev.site }}</span>
                    <span v-if="ev.date">• {{ ev.date }}</span>
                  </div>
                </div>
                <div class="flex items-center gap-1.5 shrink-0 text-[10px]">
                  <span class="px-1.5 py-0.5 rounded bg-zinc-800 border border-zinc-700 text-zinc-300">
                    {{ ev.numGames }} g
                  </span>
                  <span
                    class="px-1.5 py-0.5 rounded text-[10px] font-medium"
                    :class="ev.type === 'Swiss' ? 'bg-blue-500/20 text-blue-300' : 'bg-amber-500/20 text-amber-300'"
                  >
                    {{ ev.type }}
                  </span>
                </div>
              </button>

              <div v-if="filteredEvents.length === 0" class="p-4 text-center text-zinc-500 text-xs">
                No matching events found.
              </div>
            </div>
          </div>
        </div>

        <!-- View Mode Switcher -->
        <div class="flex items-center bg-zinc-950/80 p-1 rounded-lg border border-zinc-800 text-xs">
          <button
            class="px-3 py-1.5 rounded-md flex items-center gap-1.5 transition-colors cursor-pointer"
            :class="activeView === 'matrix' ? 'bg-amber-500/20 text-amber-300 font-medium' : 'text-zinc-400 hover:text-zinc-200'"
            @click="activeView = 'matrix'"
          >
            <Grid class="w-3.5 h-3.5" />
            <span>Matrix</span>
          </button>
          <button
            class="px-3 py-1.5 rounded-md flex items-center gap-1.5 transition-colors cursor-pointer"
            :class="activeView === 'swiss' ? 'bg-amber-500/20 text-amber-300 font-medium' : 'text-zinc-400 hover:text-zinc-200'"
            @click="activeView = 'swiss'"
          >
            <Swords class="w-3.5 h-3.5" />
            <span>Swiss</span>
          </button>
          <button
            class="px-3 py-1.5 rounded-md flex items-center gap-1.5 transition-colors cursor-pointer"
            :class="activeView === 'standings' ? 'bg-amber-500/20 text-amber-300 font-medium' : 'text-zinc-400 hover:text-zinc-200'"
            @click="activeView = 'standings'"
          >
            <ListOrdered class="w-3.5 h-3.5" />
            <span>Standings</span>
          </button>
          <button
            class="px-3 py-1.5 rounded-md flex items-center gap-1.5 transition-colors cursor-pointer"
            :class="activeView === 'text' ? 'bg-amber-500/20 text-amber-300 font-medium' : 'text-zinc-400 hover:text-zinc-200'"
            @click="activeView = 'text'"
          >
            <FileText class="w-3.5 h-3.5" />
            <span>Text / ASCII</span>
          </button>
        </div>

        <!-- Actions -->
        <div class="flex items-center gap-2">
          <button
            class="px-2.5 py-1.5 bg-zinc-800 hover:bg-zinc-700 text-zinc-300 rounded-lg text-xs flex items-center gap-1.5 border border-zinc-700 transition-colors cursor-pointer"
            title="Refresh Tournament Events"
            @click="fetchEvents(selectedEvent?.event)"
          >
            <RefreshCw class="w-3.5 h-3.5" :class="{ 'animate-spin': loading }" />
          </button>
          <button
            class="px-3 py-1.5 bg-zinc-800 hover:bg-zinc-700 text-zinc-200 rounded-lg text-xs flex items-center gap-1.5 border border-zinc-700 transition-colors cursor-pointer font-medium"
            @click="copyCrosstable"
          >
            <component :is="copied ? Check : Copy" class="w-3.5 h-3.5 text-amber-400" />
            <span>{{ copied ? 'Copied!' : 'Copy Crosstable' }}</span>
          </button>
        </div>
      </div>

      <!-- Tournament Metadata Summary Badge Bar -->
      <div
        v-if="crosstable"
        class="px-6 py-2.5 bg-zinc-950/40 border-b border-zinc-800/80 flex flex-wrap items-center justify-between text-xs gap-3"
      >
        <div class="flex items-center gap-4 flex-wrap">
          <span class="font-semibold text-zinc-100 flex items-center gap-1.5">
            <Trophy class="w-4 h-4 text-amber-400" />
            {{ crosstable.event }}
          </span>
          <span v-if="crosstable.site" class="text-zinc-400 flex items-center gap-1">
            <MapPin class="w-3 h-3 text-zinc-500" />
            {{ crosstable.site }}
          </span>
          <span v-if="crosstable.date" class="text-zinc-400 flex items-center gap-1">
            <Calendar class="w-3 h-3 text-zinc-500" />
            {{ crosstable.date }}
          </span>
        </div>

        <div class="flex items-center gap-2 flex-wrap">
          <span
            v-if="crosstable.category > 0"
            class="px-2 py-0.5 rounded-full text-[11px] font-semibold bg-amber-500/20 text-amber-300 border border-amber-500/30"
          >
            Category {{ crosstable.category }}
          </span>
          <span
            v-if="crosstable.averageElo > 0"
            class="px-2 py-0.5 rounded-full text-[11px] font-medium bg-zinc-800 text-zinc-300 border border-zinc-700"
          >
            Avg Elo: {{ crosstable.averageElo }}
          </span>
          <span class="px-2 py-0.5 rounded-full text-[11px] font-medium bg-zinc-800 text-zinc-300 border border-zinc-700">
            {{ crosstable.numPlayers }} Players
          </span>
          <span class="px-2 py-0.5 rounded-full text-[11px] font-medium bg-zinc-800 text-zinc-300 border border-zinc-700">
            {{ crosstable.numGames }} Games
          </span>
          <span class="px-2 py-0.5 rounded-full text-[11px] font-semibold bg-emerald-500/15 text-emerald-300 border border-emerald-500/30">
            {{ crosstable.type }}
          </span>
        </div>
      </div>

      <!-- Main Body Content -->
      <div class="flex-1 overflow-auto p-4 bg-zinc-900/50">
        <!-- Loading State -->
        <div v-if="loading" class="h-64 flex flex-col items-center justify-center gap-3 text-zinc-400">
          <RefreshCw class="w-8 h-8 animate-spin text-amber-400" />
          <span class="text-xs">Computing tournament cross-table and tiebreaks...</span>
        </div>

        <!-- No Tournament Data -->
        <div
          v-else-if="!crosstable || crosstable.numGames === 0"
          class="h-64 flex flex-col items-center justify-center gap-2 text-zinc-500 text-xs"
        >
          <Award class="w-10 h-10 text-zinc-600" />
          <p>No tournament games found for the selected event in this database.</p>
        </div>

        <!-- Tab 1: Round-Robin Matrix View -->
        <div v-else-if="activeView === 'matrix'" class="overflow-x-auto">
          <table class="w-full border-collapse text-xs select-none">
            <thead>
              <tr class="bg-zinc-950/80 border-b border-zinc-700 text-zinc-400">
                <th class="py-2 px-2 text-center w-8">#</th>
                <th class="py-2 px-3 text-left">Player</th>
                <th class="py-2 px-2 text-center w-14">Elo</th>
                <!-- Matrix Columns (1..N) -->
                <th
                  v-for="p in crosstable.standings"
                  :key="'col-' + p.rank"
                  class="py-2 px-2 text-center w-10 border-l border-zinc-800/80"
                  :title="p.playerName"
                >
                  {{ p.rank }}
                </th>
                <th class="py-2 px-3 text-center font-bold text-zinc-200 border-l border-zinc-700">Pts</th>
                <th class="py-2 px-2 text-center border-l border-zinc-800/80" title="Sonneborn-Berger tiebreak">SB</th>
                <th class="py-2 px-2 text-center border-l border-zinc-800/80" title="Buchholz tiebreak">Buch</th>
                <th class="py-2 px-2 text-center border-l border-zinc-800/80" title="Tournament Performance Rating">TPR</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-zinc-800/60 font-mono">
              <tr
                v-for="(p, rowIdx) in crosstable.standings"
                :key="p.playerName"
                class="hover:bg-zinc-800/40 transition-colors"
              >
                <!-- Rank -->
                <td class="py-2 px-2 text-center">
                  <span
                    class="inline-flex items-center justify-center w-5 h-5 rounded-full border text-[11px] font-bold"
                    :class="getRankBadgeClass(p.rank)"
                  >
                    {{ p.rank }}
                  </span>
                </td>
                <!-- Player Name -->
                <td class="py-2 px-3 text-left font-sans truncate max-w-[200px]" :title="p.playerName">
                  <span v-if="p.title" class="font-bold text-amber-400 mr-1 text-[11px]">{{ p.title }}</span>
                  <span class="text-zinc-100 font-medium">{{ p.playerName }}</span>
                </td>
                <!-- Elo -->
                <td class="py-2 px-2 text-center text-zinc-400">
                  {{ p.elo > 0 ? p.elo : '----' }}
                </td>

                <!-- Result Cells (0..N-1) -->
                <td
                  v-for="(cell, colIdx) in crosstable.matrix[rowIdx]"
                  :key="'cell-' + rowIdx + '-' + colIdx"
                  class="py-1 px-1 text-center border-l border-zinc-800/60 cursor-pointer transition-colors"
                  :class="getResultCellClass(cell.result)"
                  :title="cell.gameId ? `Game #${cell.gameId} (${cell.result}) - Click to open` : ''"
                  @click="handleCellClick(cell.gameId)"
                >
                  <span class="inline-block py-0.5 px-1 rounded text-center w-full">
                    {{ cell.result || '·' }}
                  </span>
                </td>

                <!-- Points -->
                <td class="py-2 px-3 text-center font-bold text-amber-300 border-l border-zinc-700 bg-amber-500/5">
                  {{ p.points % 1 === 0 ? p.points.toFixed(0) : p.points.toFixed(1) }}
                </td>
                <!-- SB -->
                <td class="py-2 px-2 text-center text-zinc-400 border-l border-zinc-800/60 text-[11px]">
                  {{ p.sonnebornBerger.toFixed(2) }}
                </td>
                <!-- Buchholz -->
                <td class="py-2 px-2 text-center text-zinc-400 border-l border-zinc-800/60 text-[11px]">
                  {{ p.buchholz.toFixed(1) }}
                </td>
                <!-- TPR -->
                <td class="py-2 px-2 text-center font-semibold border-l border-zinc-800/60 text-[11px]" :class="p.performanceRating >= (p.elo || 2400) ? 'text-emerald-400' : 'text-zinc-400'">
                  {{ p.performanceRating > 0 ? p.performanceRating : '----' }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <!-- Tab 2: Swiss Round-by-Round View -->
        <div v-else-if="activeView === 'swiss'" class="overflow-x-auto">
          <table class="w-full border-collapse text-xs select-none">
            <thead>
              <tr class="bg-zinc-950/80 border-b border-zinc-700 text-zinc-400">
                <th class="py-2 px-2 text-center w-8">#</th>
                <th class="py-2 px-3 text-left">Player</th>
                <th class="py-2 px-2 text-center w-14">Elo</th>
                <!-- Round Columns -->
                <th
                  v-for="r in crosstable.numRounds"
                  :key="'round-' + r"
                  class="py-2 px-2 text-center border-l border-zinc-800/80 min-w-[70px]"
                >
                  R{{ r }}
                </th>
                <th class="py-2 px-3 text-center font-bold text-zinc-200 border-l border-zinc-700">Pts</th>
                <th class="py-2 px-2 text-center border-l border-zinc-800/80" title="Buchholz">Buch</th>
                <th class="py-2 px-2 text-center border-l border-zinc-800/80" title="Buchholz Cut 1">Buch-1</th>
                <th class="py-2 px-2 text-center border-l border-zinc-800/80" title="Sonneborn-Berger">SB</th>
                <th class="py-2 px-2 text-center border-l border-zinc-800/80" title="TPR">TPR</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-zinc-800/60 font-mono">
              <tr
                v-for="p in crosstable.standings"
                :key="'swiss-' + p.playerName"
                class="hover:bg-zinc-800/40 transition-colors"
              >
                <!-- Rank -->
                <td class="py-2 px-2 text-center">
                  <span
                    class="inline-flex items-center justify-center w-5 h-5 rounded-full border text-[11px] font-bold"
                    :class="getRankBadgeClass(p.rank)"
                  >
                    {{ p.rank }}
                  </span>
                </td>
                <!-- Player Name -->
                <td class="py-2 px-3 text-left font-sans truncate max-w-[200px]" :title="p.playerName">
                  <span v-if="p.title" class="font-bold text-amber-400 mr-1 text-[11px]">{{ p.title }}</span>
                  <span class="text-zinc-100 font-medium">{{ p.playerName }}</span>
                </td>
                <!-- Elo -->
                <td class="py-2 px-2 text-center text-zinc-400">
                  {{ p.elo > 0 ? p.elo : '----' }}
                </td>

                <!-- Round Results -->
                <td
                  v-for="r in crosstable.numRounds"
                  :key="'rnd-' + p.rank + '-' + r"
                  class="py-1 px-1.5 text-center border-l border-zinc-800/60"
                >
                  <template v-if="p.roundResults && p.roundResults[r - 1]">
                    <button
                      class="px-2 py-0.5 rounded text-[11px] font-medium border flex items-center justify-center gap-1 mx-auto cursor-pointer transition-colors"
                      :class="
                        p.roundResults[r - 1].score === 1.0
                          ? 'bg-emerald-500/20 text-emerald-300 border-emerald-500/30 hover:bg-emerald-500/30'
                          : p.roundResults[r - 1].score === 0.5
                          ? 'bg-amber-500/20 text-amber-300 border-amber-500/30 hover:bg-amber-500/30'
                          : 'bg-red-500/15 text-red-400 border-red-500/30 hover:bg-red-500/25'
                      "
                      :title="`${p.playerName} vs ${p.roundResults[r - 1].opponentName} (${p.roundResults[r - 1].result}) - Click to open`"
                      @click="handleCellClick(p.roundResults[r - 1].gameId)"
                    >
                      <span class="text-[9px] uppercase font-bold text-zinc-400">
                        {{ p.roundResults[r - 1].color }}
                      </span>
                      <span>{{ p.roundResults[r - 1].opponentRank }}</span>
                      <span class="font-bold">({{ p.roundResults[r - 1].result }})</span>
                    </button>
                  </template>
                  <span v-else class="text-zinc-600 text-xs">--</span>
                </td>

                <!-- Points -->
                <td class="py-2 px-3 text-center font-bold text-amber-300 border-l border-zinc-700 bg-amber-500/5">
                  {{ p.points % 1 === 0 ? p.points.toFixed(0) : p.points.toFixed(1) }}
                </td>
                <!-- Buchholz -->
                <td class="py-2 px-2 text-center text-zinc-400 border-l border-zinc-800/60 text-[11px]">
                  {{ p.buchholz.toFixed(1) }}
                </td>
                <!-- Buchholz Cut 1 -->
                <td class="py-2 px-2 text-center text-zinc-400 border-l border-zinc-800/60 text-[11px]">
                  {{ p.buchholzCut1.toFixed(1) }}
                </td>
                <!-- SB -->
                <td class="py-2 px-2 text-center text-zinc-400 border-l border-zinc-800/60 text-[11px]">
                  {{ p.sonnebornBerger.toFixed(2) }}
                </td>
                <!-- TPR -->
                <td class="py-2 px-2 text-center font-semibold border-l border-zinc-800/60 text-[11px]" :class="p.performanceRating >= (p.elo || 2400) ? 'text-emerald-400' : 'text-zinc-400'">
                  {{ p.performanceRating > 0 ? p.performanceRating : '----' }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <!-- Tab 3: Standings Table View -->
        <div v-else-if="activeView === 'standings'" class="overflow-x-auto">
          <div class="mb-3 flex items-center justify-between">
            <div class="relative w-64">
              <Search class="w-3.5 h-3.5 absolute left-2.5 top-2.5 text-zinc-400" />
              <input
                v-model="playerSearchQuery"
                type="text"
                placeholder="Filter players in tournament..."
                class="w-full pl-8 pr-3 py-1.5 bg-zinc-950 border border-zinc-700 rounded-lg text-xs text-zinc-200 placeholder-zinc-500 focus:outline-none focus:border-amber-500"
              />
            </div>
            <div class="text-xs text-zinc-400">
              Showing {{ filteredStandings.length }} of {{ crosstable.standings.length }} competitors
            </div>
          </div>

          <table class="w-full border-collapse text-xs select-none">
            <thead>
              <tr class="bg-zinc-950/80 border-b border-zinc-700 text-zinc-400">
                <th class="py-2 px-3 text-center w-12">Place</th>
                <th class="py-2 px-3 text-left">Competitor</th>
                <th class="py-2 px-3 text-center">Elo Rating</th>
                <th class="py-2 px-3 text-center">Played</th>
                <th class="py-2 px-3 text-center text-emerald-400">+</th>
                <th class="py-2 px-3 text-center text-amber-400">=</th>
                <th class="py-2 px-3 text-center text-red-400">-</th>
                <th class="py-2 px-3 text-center font-bold text-zinc-100">Score</th>
                <th class="py-2 px-3 text-center" title="Sonneborn-Berger">Sonneborn-Berger</th>
                <th class="py-2 px-3 text-center" title="Buchholz Score">Buchholz</th>
                <th class="py-2 px-3 text-center" title="Performance Rating">Perf. Rating (TPR)</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-zinc-800/60 font-mono">
              <tr
                v-for="p in filteredStandings"
                :key="'standing-' + p.playerName"
                class="hover:bg-zinc-800/40 transition-colors"
              >
                <td class="py-2 px-3 text-center">
                  <span
                    class="inline-flex items-center justify-center w-5 h-5 rounded-full border text-[11px] font-bold"
                    :class="getRankBadgeClass(p.rank)"
                  >
                    {{ p.rank }}
                  </span>
                </td>
                <td class="py-2 px-3 text-left font-sans">
                  <span v-if="p.title" class="font-bold text-amber-400 mr-1 text-[11px]">{{ p.title }}</span>
                  <span class="text-zinc-100 font-semibold">{{ p.playerName }}</span>
                </td>
                <td class="py-2 px-3 text-center text-zinc-300">
                  {{ p.elo > 0 ? p.elo : '----' }}
                </td>
                <td class="py-2 px-3 text-center text-zinc-400">{{ p.played }}</td>
                <td class="py-2 px-3 text-center font-semibold text-emerald-400">{{ p.wins }}</td>
                <td class="py-2 px-3 text-center font-semibold text-amber-400">{{ p.draws }}</td>
                <td class="py-2 px-3 text-center font-semibold text-red-400">{{ p.losses }}</td>
                <td class="py-2 px-3 text-center font-bold text-amber-300 bg-amber-500/5 text-sm">
                  {{ p.points % 1 === 0 ? p.points.toFixed(0) : p.points.toFixed(1) }}
                </td>
                <td class="py-2 px-3 text-center text-zinc-300">{{ p.sonnebornBerger.toFixed(2) }}</td>
                <td class="py-2 px-3 text-center text-zinc-300">{{ p.buchholz.toFixed(1) }}</td>
                <td class="py-2 px-3 text-center font-bold" :class="p.performanceRating >= (p.elo || 2400) ? 'text-emerald-400' : 'text-zinc-300'">
                  {{ p.performanceRating > 0 ? p.performanceRating : '----' }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <!-- Tab 4: Text / ASCII Export View -->
        <div v-else-if="activeView === 'text'" class="h-full flex flex-col">
          <div class="mb-2 flex items-center justify-between">
            <span class="text-xs text-zinc-400">Plain-text ASCII tournament cross-table:</span>
            <button
              class="px-2.5 py-1 bg-zinc-800 hover:bg-zinc-700 text-zinc-200 rounded text-xs flex items-center gap-1.5 border border-zinc-700 transition-colors cursor-pointer"
              @click="copyCrosstable"
            >
              <component :is="copied ? Check : Copy" class="w-3.5 h-3.5 text-amber-400" />
              <span>{{ copied ? 'Copied' : 'Copy Text' }}</span>
            </button>
          </div>
          <pre
            class="flex-1 p-4 bg-zinc-950 border border-zinc-800 rounded-lg font-mono text-xs text-zinc-300 overflow-x-auto whitespace-pre select-all leading-relaxed"
          >{{ crosstable.crossTableText }}</pre>
        </div>
      </div>

      <!-- Modal Footer -->
      <div class="px-6 py-3 border-t border-zinc-800 flex items-center justify-between bg-zinc-950/60 text-xs">
        <div class="text-zinc-400 flex items-center gap-2">
          <span class="w-2 h-2 rounded-full bg-emerald-400 inline-block animate-pulse"></span>
          <span>Click any match result cell in Matrix or Swiss view to load that game immediately</span>
        </div>
        <button
          class="px-4 py-2 bg-zinc-800 hover:bg-zinc-700 text-zinc-200 rounded-lg font-medium transition-colors cursor-pointer"
          @click="emit('close')"
        >
          Close
        </button>
      </div>
    </div>
  </div>
</template>
