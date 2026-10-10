<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import {
  Search,
  ChevronLeft,
  ChevronRight,
  Plus,
  Trash2,
  Save,
  Database,
  Filter,
  Sliders,
  X
} from '@lucide/vue'

export interface GameSummary {
  id: number
  white: string
  black: string
  result: string
  date: string
  event: string
  site: string
  round: string
  whiteElo: number
  blackElo: number
  eco: string
  moves: number
  deleted: boolean
}

const props = defineProps<{
  games: GameSummary[]
  totalGames: number
  currentPage: number
  pageSize: number
  activeGameId?: number
  dbName?: string
  activeCriteria?: Record<string, any>
}>()

const emit = defineEmits<{
  (e: 'loadGame', id: number): void
  (e: 'pageChange', page: number): void
  (e: 'search', criteria: any): void
  (e: 'openAdvancedSearch'): void
  (e: 'newGame'): void
  (e: 'saveGame'): void
  (e: 'deleteGame', id: number): void
}>()

const searchPlayer = ref('')
const filterResult = ref('')
const searchECO = ref('')

// Sync props.activeCriteria with inline inputs
watch(() => props.activeCriteria, (c) => {
  if (c) {
    searchPlayer.value = c.player || c.white || ''
    filterResult.value = c.result || ''
    searchECO.value = c.eco || (c.ecoFrom ? `${c.ecoFrom}-${c.ecoTo || ''}` : '')
  } else {
    searchPlayer.value = ''
    filterResult.value = ''
    searchECO.value = ''
  }
}, { immediate: true })

function onSearchInput() {
  emit('search', {
    ...props.activeCriteria,
    player: searchPlayer.value.trim() || undefined,
    result: filterResult.value || undefined,
    eco: searchECO.value.trim() || undefined,
  })
}

function clearFilters() {
  searchPlayer.value = ''
  filterResult.value = ''
  searchECO.value = ''
  emit('search', {})
}

const hasActiveCriteria = computed(() => {
  if (!props.activeCriteria) return false
  return Object.values(props.activeCriteria).some(v => v !== undefined && v !== '' && v !== false)
})

const activeBadges = computed(() => {
  if (!props.activeCriteria) return []
  const list: string[] = []
  const c = props.activeCriteria
  if (c.player) list.push(`Player: ${c.player}`)
  if (c.white) list.push(`White: ${c.white}`)
  if (c.black) list.push(`Black: ${c.black}`)
  if (c.result) list.push(`Result: ${c.result}`)
  if (c.eco) list.push(`ECO: ${c.eco}`)
  if (c.ecoFrom || c.ecoTo) list.push(`ECO: ${c.ecoFrom || ''}–${c.ecoTo || ''}`)
  if (c.yearFrom || c.yearTo) list.push(`Year: ${c.yearFrom || ''}–${c.yearTo || ''}`)
  if (c.dateFrom || c.dateTo) list.push(`Date: ${c.dateFrom || ''}–${c.dateTo || ''}`)
  if (c.minElo) list.push(`Elo ≥ ${c.minElo}`)
  if (c.maxElo) list.push(`Elo ≤ ${c.maxElo}`)
  if (c.whiteEloMin) list.push(`White Elo ≥ ${c.whiteEloMin}`)
  if (c.blackEloMin) list.push(`Black Elo ≥ ${c.blackEloMin}`)
  if (c.event) list.push(`Event: ${c.event}`)
  if (c.site) list.push(`Site: ${c.site}`)
  if (c.round) list.push(`Round: ${c.round}`)
  if (c.minMoves || c.maxMoves) list.push(`Moves: ${c.minMoves || 1}–${c.maxMoves || '∞'}`)
  if (c.hasVariations) list.push('Has Variations')
  if (c.onlyDeleted) list.push('Deleted Games')
  return list
})

const totalPages = () => Math.max(1, Math.ceil(props.totalGames / props.pageSize))
</script>

<template>
  <div class="game-browser flex flex-col h-full bg-zinc-900 border border-zinc-800 rounded-xl overflow-hidden shadow-xl">
    <!-- Header with DB title and search controls -->
    <div class="browser-header flex flex-wrap items-center justify-between gap-3 p-3 bg-zinc-950 border-b border-zinc-800">
      <div class="flex items-center gap-2">
        <Database class="w-4 h-4 text-emerald-400" />
        <span class="font-bold text-sm text-zinc-200">{{ dbName || 'No Database Open' }}</span>
        <span class="text-xs text-zinc-500 font-mono">({{ totalGames }} games)</span>
      </div>

      <!-- Search Inputs -->
      <div class="flex items-center gap-2 flex-wrap">
        <div class="relative">
          <Search class="w-3.5 h-3.5 absolute left-2.5 top-1/2 -translate-y-1/2 text-zinc-500" />
          <input
            v-model="searchPlayer"
            type="text"
            placeholder="Quick player..."
            class="bg-zinc-800 border border-zinc-700 rounded-lg pl-8 pr-3 py-1 text-xs text-zinc-200 placeholder-zinc-500 focus:outline-hidden focus:border-emerald-500 w-32"
            @input="onSearchInput"
          />
        </div>

        <select
          v-model="filterResult"
          class="bg-zinc-800 border border-zinc-700 rounded-lg pl-2 pr-6 py-1 text-xs text-zinc-100 font-medium focus:outline-hidden focus:border-emerald-500 cursor-pointer"
          @change="onSearchInput"
        >
          <option value="" class="bg-zinc-900 text-zinc-100">Any Result</option>
          <option value="1-0" class="bg-zinc-900 text-zinc-100">1-0</option>
          <option value="0-1" class="bg-zinc-900 text-zinc-100">0-1</option>
          <option value="1/2-1/2" class="bg-zinc-900 text-zinc-100">½-½</option>
        </select>

        <input
          v-model="searchECO"
          type="text"
          placeholder="ECO"
          class="bg-zinc-800 border border-zinc-700 rounded-lg px-2 py-1 text-xs text-zinc-200 placeholder-zinc-500 focus:outline-hidden focus:border-emerald-500 w-16 font-mono"
          @input="onSearchInput"
        />

        <!-- Advanced Search Button -->
        <button
          class="px-2.5 py-1 rounded-lg bg-zinc-800 hover:bg-zinc-700 text-zinc-300 hover:text-white border border-zinc-700 text-xs flex items-center gap-1.5 transition-colors cursor-pointer"
          title="Advanced Header / General Search (Ctrl+H)"
          @click="emit('openAdvancedSearch')"
        >
          <Sliders class="w-3 h-3 text-emerald-400" />
          <span>Advanced</span>
        </button>

        <button
          v-if="hasActiveCriteria"
          class="text-xs text-zinc-400 hover:text-red-400 underline cursor-pointer ml-1"
          @click="clearFilters"
        >
          Clear
        </button>
      </div>

      <!-- Quick Action Buttons -->
      <div class="flex items-center gap-1.5">
        <button
          class="flex items-center gap-1 px-2.5 py-1 rounded-lg bg-zinc-800 hover:bg-zinc-700 text-zinc-300 text-xs font-medium transition-colors cursor-pointer"
          title="New Blank Game"
          @click="emit('newGame')"
        >
          <Plus class="w-3.5 h-3.5 text-emerald-400" />
          <span>New</span>
        </button>

        <button
          class="flex items-center gap-1 px-2.5 py-1 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white text-xs font-medium transition-colors cursor-pointer"
          title="Save Current Game into DB"
          @click="emit('saveGame')"
        >
          <Save class="w-3.5 h-3.5" />
          <span>Save</span>
        </button>
      </div>
    </div>

    <!-- Active Filter Banner -->
    <div
      v-if="hasActiveCriteria && activeBadges.length > 0"
      class="bg-emerald-950/40 border-b border-emerald-800/40 px-3 py-1.5 flex items-center justify-between text-xs"
    >
      <div class="flex items-center gap-1.5 flex-wrap">
        <span class="text-emerald-400 font-semibold flex items-center gap-1 text-[11px]">
          <Filter class="w-3 h-3" /> Active Filter ({{ totalGames.toLocaleString() }} matches):
        </span>
        <span
          v-for="badge in activeBadges"
          :key="badge"
          class="px-2 py-0.5 rounded bg-zinc-800 border border-zinc-700 text-zinc-300 font-mono text-[11px]"
        >
          {{ badge }}
        </span>
      </div>
      <button
        class="text-zinc-400 hover:text-red-400 flex items-center gap-1 cursor-pointer transition-colors text-[11px] ml-2 whitespace-nowrap"
        @click="clearFilters"
      >
        <X class="w-3 h-3" />
        <span>Clear</span>
      </button>
    </div>

    <!-- Games Table -->
    <div class="games-table-container flex-1 overflow-y-auto">
      <div v-if="games.length === 0" class="text-zinc-500 text-xs italic text-center py-12">
        No games match the current query or database is empty.
      </div>

      <table v-else class="w-full text-left font-mono text-xs border-collapse">
        <thead class="bg-zinc-950 sticky top-0 text-zinc-400 select-none z-10 border-b border-zinc-800">
          <tr>
            <th class="py-2 px-3 w-12 font-semibold">#</th>
            <th class="py-2 px-3 font-semibold">White</th>
            <th class="py-2 px-3 font-semibold">Black</th>
            <th class="py-2 px-2 font-semibold text-center w-16">Result</th>
            <th class="py-2 px-2 font-semibold w-24">Date</th>
            <th class="py-2 px-2 font-semibold">Event</th>
            <th class="py-2 px-2 font-semibold w-16">ECO</th>
            <th class="py-2 px-2 font-semibold text-right w-16">Moves</th>
            <th class="py-2 px-2 w-10"></th>
          </tr>
        </thead>
        <tbody class="divide-y divide-zinc-800/40">
          <tr
            v-for="g in games"
            :key="g.id"
            class="transition-colors cursor-pointer"
            :class="[
              activeGameId === g.id ? 'bg-emerald-950/40 border-l-2 border-emerald-500' : 'hover:bg-zinc-800/50',
              g.deleted ? 'opacity-50 line-through' : '',
            ]"
            @click="emit('loadGame', g.id)"
          >
            <td class="py-2 px-3 text-zinc-500 font-medium">{{ g.id }}</td>
            <td class="py-2 px-3 font-semibold text-zinc-100 flex items-center gap-1.5">
              <span>{{ g.white }}</span>
              <span v-if="g.whiteElo" class="text-[10px] text-zinc-500 font-normal">({{ g.whiteElo }})</span>
            </td>
            <td class="py-2 px-3 font-semibold text-zinc-200">
              <span>{{ g.black }}</span>
              <span v-if="g.blackElo" class="text-[10px] text-zinc-500 font-normal ml-1.5">({{ g.blackElo }})</span>
            </td>
            <td class="py-2 px-2 text-center font-bold" :class="g.result === '1-0' ? 'text-zinc-200' : g.result === '0-1' ? 'text-zinc-400' : 'text-zinc-500'">
              {{ g.result }}
            </td>
            <td class="py-2 px-2 text-zinc-400 text-[11px]">{{ g.date }}</td>
            <td class="py-2 px-2 text-zinc-400 truncate max-w-[160px]">{{ g.event }}</td>
            <td class="py-2 px-2 text-emerald-400 font-medium">{{ g.eco }}</td>
            <td class="py-2 px-2 text-right text-zinc-400">{{ g.moves }}</td>
            <td class="py-2 px-2 text-center">
              <button
                class="text-zinc-500 hover:text-red-400 p-1 rounded transition-colors"
                title="Toggle Deleted"
                @click.stop="emit('deleteGame', g.id)"
              >
                <Trash2 class="w-3.5 h-3.5" />
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Pagination Footer -->
    <div class="browser-footer flex items-center justify-between px-3 py-2 bg-zinc-950 border-t border-zinc-800 text-xs text-zinc-400">
      <div class="flex items-center gap-2">
        <span>Page {{ currentPage }} of {{ totalPages() }}</span>
      </div>

      <div class="flex items-center gap-1">
        <button
          class="p-1 rounded-lg hover:bg-zinc-800 text-zinc-400 hover:text-zinc-200 disabled:opacity-30 disabled:pointer-events-none transition-colors"
          :disabled="currentPage <= 1"
          @click="emit('pageChange', currentPage - 1)"
        >
          <ChevronLeft class="w-4 h-4" />
        </button>
        <button
          class="p-1 rounded-lg hover:bg-zinc-800 text-zinc-400 hover:text-zinc-200 disabled:opacity-30 disabled:pointer-events-none transition-colors"
          :disabled="currentPage >= totalPages()"
          @click="emit('pageChange', currentPage + 1)"
        >
          <ChevronRight class="w-4 h-4" />
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.games-table-container::-webkit-scrollbar {
  width: 6px;
}
.games-table-container::-webkit-scrollbar-thumb {
  background-color: rgba(255, 255, 255, 0.1);
  border-radius: 4px;
}
</style>
