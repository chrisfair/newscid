<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { useNotation, formatMoveNotation } from '../utils/notation'
import { BookOpen, Database, RefreshCw, ExternalLink, ChevronsLeft, ChevronLeft, FolderOpen } from '@lucide/vue'
import { GetPolyglotBooks, QueryPolyglotBook } from '../../wailsjs/go/main/App'
import { polyglot } from '../../wailsjs/go/models'

export interface MoveStat {
  move: string
  uci: string
  count: number
  whiteWins: number
  draws: number
  blackWins: number
  scorePct: number
  avgElo?: number
  eco?: string
}

export interface TreeData {
  fen: string
  totalGames: number
  moves: MoveStat[]
}

const props = defineProps<{
  fen?: string
  treeData: TreeData | null
  loading?: boolean
  dbOpen?: boolean
  dbName?: string
  ecoCode?: string
  openingName?: string
  discoveredDatabases?: Array<{ name: string; basePath: string; numGames: number; category?: string }>
}>()

const emit = defineEmits<{
  (e: 'playMove', uci: string): void
  (e: 'refresh'): void
  (e: 'openDb'): void
  (e: 'openDiscoveredDb', path: string): void
  (e: 'goFirst'): void
  (e: 'goPrev'): void
  (e: 'popOutWindow'): void
}>()

// Active tab: 'db' or 'book'
const activeTab = ref<'db' | 'book'>('db')

// Polyglot state
const availableBooks = ref<string[]>([])
const selectedBook = ref<string>('')
const bookMoves = ref<polyglot.BookMove[]>([])
const loadingBook = ref(false)

const movesList = computed(() => props.treeData?.moves || [])
const totalGames = computed(() => props.treeData?.totalGames || 0)

async function loadAvailableBooks() {
  try {
    const books = await GetPolyglotBooks()
    if (books && books.length > 0) {
      availableBooks.value = books
      if (!selectedBook.value) {
        selectedBook.value = books[0]
      }
    }
  } catch (err) {
    console.error('Failed to load polyglot books:', err)
  }
}

async function queryCurrentBook() {
  const targetFen = props.fen || props.treeData?.fen
  if (!selectedBook.value || !targetFen) {
    bookMoves.value = []
    return
  }

  loadingBook.value = true
  try {
    const res = await QueryPolyglotBook(targetFen, selectedBook.value)
    bookMoves.value = res || []
  } catch (err) {
    console.error('Failed to query polyglot book:', err)
    bookMoves.value = []
  } finally {
    loadingBook.value = false
  }
}

watch(() => [props.fen, props.treeData?.fen], () => {
  if (activeTab.value === 'book') {
    queryCurrentBook()
  }
})

watch(selectedBook, () => {
  queryCurrentBook()
})

watch(activeTab, (tab) => {
  if (tab === 'book') {
    queryCurrentBook()
  } else {
    emit('refresh')
  }
})

onMounted(async () => {
  await loadAvailableBooks()
  if (activeTab.value === 'book') {
    await queryCurrentBook()
  } else {
    emit('refresh')
  }
})

function formatBookName(path: string) {
  const parts = path.split('/')
  return parts[parts.length - 1] || path
}
</script>

<template>
  <div class="opening-tree flex flex-col h-full bg-zinc-900 border border-zinc-800 rounded-xl overflow-hidden shadow-lg select-none">
    <!-- Header with DB / Polyglot Book Tab Selector & Window Pop-out -->
    <div class="tree-header flex items-center justify-between p-2 bg-zinc-950 border-b border-zinc-800">
      <div class="flex items-center gap-1.5">
        <!-- DB Tree Tab -->
        <button
          class="px-2.5 py-1 rounded-lg text-xs font-semibold uppercase tracking-wider transition-colors cursor-pointer flex items-center gap-1.5"
          :class="activeTab === 'db' ? 'bg-zinc-800 text-emerald-400' : 'text-zinc-500 hover:text-zinc-300'"
          @click="activeTab = 'db'"
        >
          <Database class="w-3.5 h-3.5" />
          <span>{{ dbOpen ? 'DB Tree' : 'Master Tree' }}</span>
          <span class="px-1.5 py-0.2 rounded-full text-[10px] font-mono bg-zinc-900 text-zinc-400 font-normal">
            {{ totalGames }}
          </span>
        </button>

        <!-- Polyglot Book Tab -->
        <button
          class="px-2.5 py-1 rounded-lg text-xs font-semibold uppercase tracking-wider transition-colors cursor-pointer flex items-center gap-1.5"
          :class="activeTab === 'book' ? 'bg-zinc-800 text-purple-400' : 'text-zinc-500 hover:text-zinc-300'"
          @click="activeTab = 'book'"
        >
          <BookOpen class="w-3.5 h-3.5" />
          <span>Opening Book (.bin)</span>
          <span v-if="bookMoves.length > 0" class="px-1.5 py-0.2 rounded-full text-[10px] font-mono bg-zinc-900 text-purple-400 font-normal">
            {{ bookMoves.length }}
          </span>
        </button>
      </div>

      <div class="flex items-center gap-1">
        <!-- Book selector dropdown when Book tab is active -->
        <select
          v-if="activeTab === 'book' && availableBooks.length > 0"
          v-model="selectedBook"
          class="bg-zinc-900 border border-zinc-700 rounded-lg px-2 py-0.5 text-xs text-zinc-300 font-mono focus:outline-hidden focus:border-purple-500 max-w-[120px] truncate"
        >
          <option v-for="b in availableBooks" :key="b" :value="b">
            {{ formatBookName(b) }}
          </option>
        </select>

        <!-- Navigation Buttons -->
        <button
          class="p-1 rounded-lg text-zinc-400 hover:text-zinc-100 hover:bg-zinc-800 transition-colors cursor-pointer"
          title="Jump to Game Start Position"
          @click="emit('goFirst')"
        >
          <ChevronsLeft class="w-3.5 h-3.5" />
        </button>
        <button
          class="p-1 rounded-lg text-zinc-400 hover:text-zinc-100 hover:bg-zinc-800 transition-colors cursor-pointer"
          title="Step Back 1 Move"
          @click="emit('goPrev')"
        >
          <ChevronLeft class="w-3.5 h-3.5" />
        </button>

        <button
          class="p-1 rounded-lg text-zinc-400 hover:text-zinc-100 hover:bg-zinc-800 transition-colors cursor-pointer"
          title="Refresh Opening Tree"
          @click="activeTab === 'book' ? queryCurrentBook() : emit('refresh')"
        >
          <RefreshCw class="w-3.5 h-3.5" :class="{ 'animate-spin': loading || loadingBook }" />
        </button>

        <!-- Pop out dedicated window -->
        <button
          class="p-1 rounded-lg text-emerald-400 hover:text-emerald-300 hover:bg-zinc-800 transition-colors cursor-pointer ml-1"
          title="Open Dedicated Opening Tree Window (F3)"
          @click="emit('popOutWindow')"
        >
          <ExternalLink class="w-3.5 h-3.5" />
        </button>
      </div>
    </div>

    <!-- Sub-info bar: ECO & Opening name if present -->
    <div v-if="ecoCode || openingName" class="px-3 py-1 bg-zinc-950/70 border-b border-zinc-800 flex items-center gap-2 text-xs">
      <span v-if="ecoCode" class="font-mono text-emerald-400 font-bold bg-emerald-950/60 px-1 rounded text-[10px]">
        {{ ecoCode }}
      </span>
      <span v-if="openingName" class="text-zinc-300 truncate text-[11px]">
        {{ openingName }}
      </span>
    </div>

    <!-- Tree Table: DB Tree View -->
    <div v-if="activeTab === 'db'" class="table-container flex-1 overflow-y-auto">
      <div v-if="movesList.length === 0" class="text-zinc-400 text-xs text-center py-6 px-4 space-y-3">
        <template v-if="!dbOpen">
          <p class="font-medium text-zinc-300">No database is currently open.</p>
          <p class="text-[11px] text-zinc-500">Select a discovered database below or open a file:</p>

          <!-- Quick Open Discovered Bases -->
          <div v-if="discoveredDatabases && discoveredDatabases.length > 0" class="flex flex-wrap justify-center gap-1.5 max-w-sm mx-auto">
            <button
              v-for="d in discoveredDatabases.slice(0, 3)"
              :key="d.basePath"
              class="px-2.5 py-1 bg-zinc-800 hover:bg-zinc-700 border border-zinc-700 rounded-lg text-[11px] text-zinc-300 transition-all cursor-pointer flex items-center gap-1"
              @click="emit('openDiscoveredDb', d.basePath)"
            >
              <Database class="w-3 h-3 text-emerald-400" />
              <span>{{ d.name }}</span>
            </button>
          </div>

          <div class="flex items-center justify-center gap-2 pt-1">
            <button
              class="px-3 py-1.5 bg-emerald-600/20 text-emerald-400 hover:bg-emerald-600/30 border border-emerald-500/40 rounded-lg text-xs font-semibold transition-all cursor-pointer inline-flex items-center gap-1.5"
              @click="emit('openDb')"
            >
              <FolderOpen class="w-3.5 h-3.5" />
              <span>Open Database</span>
            </button>
            <button
              class="px-3 py-1.5 bg-purple-600/20 text-purple-400 hover:bg-purple-600/30 border border-purple-500/40 rounded-lg text-xs font-semibold transition-all cursor-pointer inline-flex items-center gap-1.5"
              @click="activeTab = 'book'"
            >
              <BookOpen class="w-3.5 h-3.5" />
              <span>Opening Book</span>
            </button>
          </div>
        </template>
        <template v-else>
          <p class="italic text-zinc-400">0 database games match this position.</p>
          <p class="text-[11px] text-zinc-500">The current board position is out of database repertoire.</p>
          <div class="flex items-center justify-center gap-2 pt-2">
            <button
              class="px-2.5 py-1 bg-zinc-800 hover:bg-zinc-700 border border-zinc-700 rounded-lg text-[11px] text-zinc-300 transition-colors cursor-pointer flex items-center gap-1"
              @click="emit('goFirst')"
            >
              <ChevronsLeft class="w-3 h-3" />
              <span>Start</span>
            </button>
            <button
              class="px-2.5 py-1 bg-zinc-800 hover:bg-zinc-700 border border-zinc-700 rounded-lg text-[11px] text-zinc-300 transition-colors cursor-pointer flex items-center gap-1"
              @click="emit('goPrev')"
            >
              <ChevronLeft class="w-3 h-3" />
              <span>Step Back</span>
            </button>
            <button
              class="px-2.5 py-1 bg-purple-600/20 text-purple-400 hover:bg-purple-600/30 border border-purple-500/40 rounded-lg text-[11px] font-semibold transition-colors cursor-pointer flex items-center gap-1"
              @click="activeTab = 'book'"
            >
              <BookOpen class="w-3 h-3" />
              <span>Check Book</span>
            </button>
          </div>
        </template>
      </div>

      <table v-else class="w-full text-left font-mono text-xs border-collapse">
        <thead class="bg-zinc-950/80 sticky top-0 text-zinc-400 select-none z-10 border-b border-zinc-800/80">
          <tr>
            <th class="py-2 px-3 font-semibold">Move</th>
            <th class="py-2 px-2 font-semibold text-right">Games</th>
            <th class="py-2 px-2 font-semibold">Score %</th>
            <th class="py-2 px-2 font-semibold text-right text-zinc-500">W / D / L</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-zinc-800/40">
          <tr
            v-for="m in movesList"
            :key="m.move"
            class="hover:bg-emerald-950/30 transition-colors cursor-pointer group"
            :title="`Click to play ${m.move} on board`"
            @click="emit('playMove', m.uci || m.move)"
          >
            <!-- Move SAN / Styled -->
            <td class="py-2 px-3 font-bold text-emerald-400 group-hover:text-emerald-300 flex items-center gap-1.5">
              <span>{{ formatMoveNotation(m.move, m.uci || '') }}</span>
            </td>

            <!-- Games Count -->
            <td class="py-2 px-2 text-right text-zinc-300 font-medium">
              {{ m.count }}
            </td>

            <!-- Score % Bar -->
            <td class="py-2 px-2 min-w-[120px]">
              <div class="flex items-center gap-2">
                <span class="w-9 text-right text-[11px] text-zinc-400">
                  {{ m.scorePct.toFixed(1) }}%
                </span>
                <div
                  v-if="(m.whiteWins + m.draws + m.blackWins) > 0 && m.count > 0"
                  class="flex-1 h-2 rounded-full overflow-hidden bg-zinc-800 flex"
                >
                  <!-- White win bar -->
                  <div
                    class="bg-zinc-300 h-full"
                    :style="{ width: `${(m.whiteWins / m.count) * 100}%` }"
                    :title="`White Wins: ${m.whiteWins}`"
                  />
                  <!-- Draw bar -->
                  <div
                    class="bg-zinc-500 h-full"
                    :style="{ width: `${(m.draws / m.count) * 100}%` }"
                    :title="`Draws: ${m.draws}`"
                  />
                  <!-- Black win bar -->
                  <div
                    class="bg-zinc-900 h-full border-l border-zinc-700"
                    :style="{ width: `${(m.blackWins / m.count) * 100}%` }"
                    :title="`Black Wins: ${m.blackWins}`"
                  />
                </div>
                <div
                  v-else
                  class="flex-1 h-2 rounded-full overflow-hidden bg-zinc-800 flex"
                >
                  <div
                    class="bg-emerald-500 h-full"
                    :style="{ width: `${Math.min(100, Math.max(0, m.scorePct))}%` }"
                    :title="`Score Percentage: ${m.scorePct.toFixed(1)}%`"
                  />
                </div>
              </div>
            </td>

            <!-- W / D / L Breakdown -->
            <td class="py-2 px-2 text-right text-[11px] text-zinc-400 whitespace-nowrap">
              <template v-if="(m.whiteWins + m.draws + m.blackWins) > 0">
                <span class="text-zinc-200">{{ m.whiteWins }}</span>
                <span class="text-zinc-500">/</span>
                <span class="text-zinc-400">{{ m.draws }}</span>
                <span class="text-zinc-500">/</span>
                <span class="text-zinc-400">{{ m.blackWins }}</span>
              </template>
              <template v-else>
                <span class="text-purple-400 font-mono text-[10px]">Book</span>
              </template>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Tree Table: Polyglot Opening Book View -->
    <div v-else-if="activeTab === 'book'" class="table-container flex-1 overflow-y-auto">
      <div v-if="availableBooks.length === 0" class="text-zinc-500 text-xs italic text-center py-8 space-y-1">
        <p>No Polyglot opening books (.bin) found on system.</p>
        <p class="text-[11px] text-zinc-600">Place .bin books in /usr/share/scid/books or ~/.scidvspc/books</p>
      </div>

      <div v-else-if="bookMoves.length === 0" class="text-zinc-400 text-xs italic text-center py-6 space-y-3">
        <p>Position out of book in {{ formatBookName(selectedBook) }}.</p>
        <div class="flex items-center justify-center gap-2">
          <button
            class="px-2.5 py-1 bg-zinc-800 hover:bg-zinc-700 border border-zinc-700 rounded-lg text-[11px] text-zinc-300 transition-colors cursor-pointer flex items-center gap-1"
            @click="emit('goFirst')"
          >
            <ChevronsLeft class="w-3 h-3" />
            <span>Start</span>
          </button>
          <button
            class="px-2.5 py-1 bg-zinc-800 hover:bg-zinc-700 border border-zinc-700 rounded-lg text-[11px] text-zinc-300 transition-colors cursor-pointer flex items-center gap-1"
            @click="emit('goPrev')"
          >
            <ChevronLeft class="w-3 h-3" />
            <span>Step Back</span>
          </button>
        </div>
      </div>

      <table v-else class="w-full text-left font-mono text-xs border-collapse">
        <thead class="bg-zinc-950/80 sticky top-0 text-zinc-400 select-none z-10 border-b border-zinc-800/80">
          <tr>
            <th class="py-2 px-3 font-semibold">Book Move</th>
            <th class="py-2 px-2 font-semibold text-right">Weight</th>
            <th class="py-2 px-2 font-semibold">Frequency %</th>
            <th class="py-2 px-2 font-semibold text-right text-zinc-500">Learn</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-zinc-800/40">
          <tr
            v-for="bm in bookMoves"
            :key="bm.uci"
            class="hover:bg-purple-950/30 transition-colors cursor-pointer group"
            :title="`Click to play book move ${bm.san} on board`"
            @click="emit('playMove', bm.uci || bm.san)"
          >
            <!-- Move SAN / Styled -->
            <td class="py-2 px-3 font-bold text-purple-400 group-hover:text-purple-300 flex items-center gap-1.5">
              <span>{{ formatMoveNotation(bm.san, bm.uci) }}</span>
            </td>

            <!-- Weight -->
            <td class="py-2 px-2 text-right text-zinc-300 font-medium">
              {{ bm.weight.toLocaleString() }}
            </td>

            <!-- Frequency % Bar -->
            <td class="py-2 px-2 min-w-[120px]">
              <div class="flex items-center gap-2">
                <span class="w-9 text-right text-[11px] text-zinc-400">
                  {{ bm.percentage.toFixed(1) }}%
                </span>
                <div class="flex-1 h-2 rounded-full overflow-hidden bg-zinc-800 flex">
                  <div
                    class="bg-purple-500 h-full rounded-full"
                    :style="{ width: `${bm.percentage}%` }"
                    :title="`${bm.percentage.toFixed(1)}%`"
                  />
                </div>
              </div>
            </td>

            <!-- Learn stat -->
            <td class="py-2 px-2 text-right text-[11px] text-zinc-500">
              {{ bm.learn || 0 }}
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<style scoped>
.table-container::-webkit-scrollbar {
  width: 6px;
}
.table-container::-webkit-scrollbar-thumb {
  background-color: rgba(255, 255, 255, 0.1);
  border-radius: 4px;
}
</style>
