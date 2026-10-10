<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { useNotation, formatMoveNotation } from '../utils/notation'
import { BookOpen, Database, RefreshCw, X, ChevronsLeft, ChevronLeft, FolderOpen, ExternalLink } from '@lucide/vue'
import { GetPolyglotBooks, QueryPolyglotBook } from '../../wailsjs/go/main/App'
import { polyglot } from '../../wailsjs/go/models'
import type { TreeData, MoveStat } from './OpeningTree.vue'

const props = defineProps<{
  isOpen: boolean
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
  (e: 'close'): void
  (e: 'playMove', uci: string): void
  (e: 'refresh'): void
  (e: 'openDb'): void
  (e: 'openDiscoveredDb', path: string): void
  (e: 'goFirst'): void
  (e: 'goPrev'): void
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
  if (props.isOpen && activeTab.value === 'book') {
    queryCurrentBook()
  }
})

watch(selectedBook, () => {
  if (props.isOpen && activeTab.value === 'book') {
    queryCurrentBook()
  }
})

watch(activeTab, (tab) => {
  if (tab === 'book') {
    queryCurrentBook()
  } else {
    emit('refresh')
  }
})

watch(() => props.isOpen, (open) => {
  if (open) {
    loadAvailableBooks()
    if (activeTab.value === 'book') {
      queryCurrentBook()
    } else {
      emit('refresh')
    }
  }
})

onMounted(async () => {
  await loadAvailableBooks()
  if (props.isOpen && activeTab.value === 'book') {
    await queryCurrentBook()
  }
})

function formatBookName(path: string) {
  const parts = path.split('/')
  return parts[parts.length - 1] || path
}
</script>

<template>
  <div
    v-if="isOpen"
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/75 backdrop-blur-xs p-4"
    @click.self="emit('close')"
  >
    <div
      class="bg-zinc-900 border border-zinc-700/80 rounded-2xl shadow-2xl w-full max-w-4xl max-h-[85vh] flex flex-col overflow-hidden text-zinc-200"
      @keydown.esc="emit('close')"
    >
      <!-- Modal Header -->
      <div class="px-5 py-3.5 bg-zinc-950 border-b border-zinc-800 flex items-center justify-between select-none">
        <div class="flex items-center gap-3">
          <div class="p-2 rounded-xl bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
            <BookOpen class="w-5 h-5" />
          </div>
          <div>
            <div class="flex items-center gap-2">
              <h2 class="text-base font-bold text-white tracking-tight">Opening Tree &amp; Repertoire Explorer</h2>
              <span class="px-1.5 py-0.5 rounded text-[10px] font-mono bg-zinc-800 text-zinc-400 font-semibold border border-zinc-700/60">F3</span>
            </div>
            <div class="flex items-center gap-2 mt-0.5 text-xs text-zinc-400">
              <span v-if="ecoCode" class="font-mono text-emerald-400 font-bold bg-emerald-950/60 px-1.5 py-0.2 rounded border border-emerald-800/40">
                {{ ecoCode }}
              </span>
              <span v-if="openingName" class="text-zinc-300 font-medium truncate max-w-md">
                {{ openingName }}
              </span>
              <span class="text-zinc-500">• {{ dbOpen ? `Base: ${dbName || 'Open Base'}` : 'Master Repertoire Tree' }}</span>
            </div>
          </div>
        </div>

        <!-- Quick Navigation & Close -->
        <div class="flex items-center gap-1.5">
          <button
            class="px-2.5 py-1.5 rounded-lg bg-zinc-800/80 hover:bg-zinc-700 text-xs text-zinc-300 font-medium flex items-center gap-1.5 transition-colors cursor-pointer"
            title="Jump to Game Start / Root Position"
            @click="emit('goFirst')"
          >
            <ChevronsLeft class="w-3.5 h-3.5" />
            <span>Root</span>
          </button>
          <button
            class="px-2.5 py-1.5 rounded-lg bg-zinc-800/80 hover:bg-zinc-700 text-xs text-zinc-300 font-medium flex items-center gap-1.5 transition-colors cursor-pointer"
            title="Step Back 1 Move"
            @click="emit('goPrev')"
          >
            <ChevronLeft class="w-3.5 h-3.5" />
            <span>Back</span>
          </button>
          <button
            class="p-1.5 rounded-lg bg-zinc-800/80 hover:bg-zinc-700 text-zinc-300 transition-colors cursor-pointer ml-1"
            title="Refresh Tree"
            @click="activeTab === 'book' ? queryCurrentBook() : emit('refresh')"
          >
            <RefreshCw class="w-4 h-4" :class="{ 'animate-spin': loading || loadingBook }" />
          </button>
          <button
            class="p-1.5 rounded-lg bg-zinc-800/80 hover:bg-rose-900/60 hover:text-rose-200 text-zinc-400 transition-colors cursor-pointer ml-2"
            title="Close Tree Window (Esc)"
            @click="emit('close')"
          >
            <X class="w-4 h-4" />
          </button>
        </div>
      </div>

      <!-- Tab Selectors & Subheader -->
      <div class="px-5 py-2.5 bg-zinc-950/60 border-b border-zinc-800 flex items-center justify-between">
        <div class="flex items-center gap-2">
          <!-- DB Tree Tab Button -->
          <button
            class="px-3.5 py-1.5 rounded-lg text-xs font-semibold uppercase tracking-wider transition-colors cursor-pointer flex items-center gap-2"
            :class="activeTab === 'db' ? 'bg-zinc-800 text-emerald-400 shadow-sm border border-emerald-500/30' : 'text-zinc-400 hover:text-zinc-200 hover:bg-zinc-800/50'"
            @click="activeTab = 'db'"
          >
            <Database class="w-3.5 h-3.5" />
            <span>{{ dbOpen ? 'Database Tree' : 'Master Tree' }}</span>
            <span class="px-1.5 py-0.2 rounded-full text-[10px] font-mono bg-zinc-900 text-zinc-300">
              {{ totalGames }} {{ dbOpen ? 'games' : 'weight' }}
            </span>
          </button>

          <!-- Polyglot Book Tab Button -->
          <button
            class="px-3.5 py-1.5 rounded-lg text-xs font-semibold uppercase tracking-wider transition-colors cursor-pointer flex items-center gap-2"
            :class="activeTab === 'book' ? 'bg-zinc-800 text-purple-400 shadow-sm border border-purple-500/30' : 'text-zinc-400 hover:text-zinc-200 hover:bg-zinc-800/50'"
            @click="activeTab = 'book'"
          >
            <BookOpen class="w-3.5 h-3.5" />
            <span>Opening Book (.bin)</span>
            <span v-if="bookMoves.length > 0" class="px-1.5 py-0.2 rounded-full text-[10px] font-mono bg-zinc-900 text-purple-400">
              {{ bookMoves.length }}
            </span>
          </button>
        </div>

        <!-- Polyglot Book Selector Dropdown when Book tab active -->
        <div v-if="activeTab === 'book' && availableBooks.length > 0" class="flex items-center gap-2">
          <span class="text-xs text-zinc-400">Book:</span>
          <select
            v-model="selectedBook"
            class="bg-zinc-900 border border-zinc-700 rounded-lg px-2.5 py-1 text-xs text-zinc-200 font-mono focus:outline-hidden focus:border-purple-500 max-w-[200px] truncate"
          >
            <option v-for="b in availableBooks" :key="b" :value="b">
              {{ formatBookName(b) }}
            </option>
          </select>
        </div>
      </div>

      <!-- Main Body -->
      <div class="flex-1 overflow-y-auto p-4 min-h-[350px]">
        <!-- 1. DB Tree Content -->
        <div v-if="activeTab === 'db'" class="h-full flex flex-col">
          <!-- State: No Database Open -->
          <div v-if="!dbOpen" class="flex-1 flex flex-col items-center justify-center py-10 px-4 text-center space-y-4">
            <div class="p-3 bg-zinc-800/80 rounded-2xl border border-zinc-700/60 text-emerald-400">
              <Database class="w-8 h-8" />
            </div>
            <div class="max-w-md space-y-1">
              <h3 class="text-base font-bold text-white">No Database Is Currently Open</h3>
              <p class="text-xs text-zinc-400">
                To explore master statistics, open a Scid or PGN database, or quickly select one of the discovered databases below:
              </p>
            </div>

            <!-- Quick Open Discovered Bases -->
            <div v-if="discoveredDatabases && discoveredDatabases.length > 0" class="flex flex-wrap justify-center gap-2 max-w-xl">
              <button
                v-for="d in discoveredDatabases.slice(0, 5)"
                :key="d.basePath"
                class="px-3 py-1.5 bg-zinc-800 hover:bg-zinc-700 border border-zinc-700/80 hover:border-emerald-500/50 rounded-xl text-xs text-zinc-200 transition-all cursor-pointer flex items-center gap-1.5"
                @click="emit('openDiscoveredDb', d.basePath)"
              >
                <Database class="w-3 h-3 text-emerald-400" />
                <span class="font-semibold">{{ d.name }}</span>
                <span class="text-[10px] text-zinc-400 font-mono">({{ d.numGames.toLocaleString() }} games)</span>
              </button>
            </div>

            <div class="flex items-center gap-3 pt-2">
              <button
                class="px-4 py-2 bg-emerald-600 hover:bg-emerald-500 text-white rounded-xl text-xs font-semibold shadow-md transition-all cursor-pointer flex items-center gap-2"
                @click="emit('openDb')"
              >
                <FolderOpen class="w-4 h-4" />
                <span>Browse Database File (.si4, .pgn)...</span>
              </button>
              <button
                class="px-4 py-2 bg-zinc-800 hover:bg-zinc-700 text-zinc-200 rounded-xl text-xs font-semibold border border-zinc-700 transition-all cursor-pointer flex items-center gap-2"
                @click="activeTab = 'book'"
              >
                <BookOpen class="w-4 h-4 text-purple-400" />
                <span>Switch to Opening Book (.bin)</span>
              </button>
            </div>
          </div>

          <!-- State: Database Open, but 0 matching games -->
          <div v-else-if="movesList.length === 0" class="flex-1 flex flex-col items-center justify-center py-10 px-4 text-center space-y-4">
            <div class="max-w-md space-y-2">
              <p class="text-sm font-semibold text-zinc-300">0 matching database games at this board position.</p>
              <p class="text-xs text-zinc-500">
                The current board position is out of book for database <span class="font-bold text-zinc-400">{{ dbName }}</span>.
              </p>
            </div>
            <div class="flex items-center gap-3">
              <button
                class="px-3 py-1.5 bg-zinc-800 hover:bg-zinc-700 border border-zinc-700 text-zinc-200 rounded-lg text-xs font-semibold transition-all cursor-pointer flex items-center gap-1.5"
                @click="emit('goFirst')"
              >
                <ChevronsLeft class="w-3.5 h-3.5" />
                <span>Go to Start Position</span>
              </button>
              <button
                class="px-3 py-1.5 bg-zinc-800 hover:bg-zinc-700 border border-zinc-700 text-zinc-200 rounded-lg text-xs font-semibold transition-all cursor-pointer flex items-center gap-1.5"
                @click="emit('goPrev')"
              >
                <ChevronLeft class="w-3.5 h-3.5" />
                <span>Step Back 1 Move</span>
              </button>
              <button
                class="px-3 py-1.5 bg-purple-600/20 text-purple-400 hover:bg-purple-600/30 border border-purple-500/40 rounded-lg text-xs font-semibold transition-all cursor-pointer flex items-center gap-1.5"
                @click="activeTab = 'book'"
              >
                <BookOpen class="w-3.5 h-3.5" />
                <span>Check Opening Book (.bin)</span>
              </button>
            </div>
          </div>

          <!-- State: Move Table -->
          <div v-else class="table-container flex-1 overflow-y-auto">
            <table class="w-full text-left font-mono text-xs border-collapse">
              <thead class="bg-zinc-950 sticky top-0 text-zinc-400 select-none z-10 border-b border-zinc-800">
                <tr>
                  <th class="py-2.5 px-3 font-semibold">Move</th>
                  <th class="py-2.5 px-3 font-semibold text-right">Games</th>
                  <th class="py-2.5 px-3 font-semibold text-right">Frequency %</th>
                  <th class="py-2.5 px-4 font-semibold">Score % Bar</th>
                  <th class="py-2.5 px-3 font-semibold text-right text-zinc-500">W / D / L</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-zinc-800/60">
                <tr
                  v-for="m in movesList"
                  :key="m.move"
                  class="hover:bg-emerald-950/30 transition-colors cursor-pointer group"
                  :title="`Click to play ${m.move} on board`"
                  @click="emit('playMove', m.uci || m.move)"
                >
                  <!-- Move SAN -->
                  <td class="py-2.5 px-3 font-bold text-emerald-400 group-hover:text-emerald-300 flex items-center gap-2">
                    <span class="text-sm">{{ formatMoveNotation(m.move, m.uci || '') }}</span>
                  </td>

                  <!-- Games Count -->
                  <td class="py-2.5 px-3 text-right text-zinc-200 font-semibold">
                    {{ m.count.toLocaleString() }}
                  </td>

                  <!-- Frequency % -->
                  <td class="py-2.5 px-3 text-right text-zinc-400">
                    {{ totalGames > 0 ? ((m.count / totalGames) * 100).toFixed(1) : 0 }}%
                  </td>

                  <!-- Score % Bar -->
                  <td class="py-2.5 px-4 min-w-[160px]">
                    <div class="flex items-center gap-2">
                      <span class="w-10 text-right text-xs font-semibold text-zinc-300">
                        {{ m.scorePct.toFixed(1) }}%
                      </span>
                      <div
                        v-if="(m.whiteWins + m.draws + m.blackWins) > 0 && m.count > 0"
                        class="flex-1 h-2.5 rounded-full overflow-hidden bg-zinc-800 flex shadow-inner"
                      >
                        <!-- White win bar -->
                        <div
                          class="bg-zinc-200 h-full"
                          :style="{ width: `${(m.whiteWins / m.count) * 100}%` }"
                          :title="`White Wins: ${m.whiteWins} (${((m.whiteWins / m.count) * 100).toFixed(1)}%)`"
                        />
                        <!-- Draw bar -->
                        <div
                          class="bg-zinc-500 h-full"
                          :style="{ width: `${(m.draws / m.count) * 100}%` }"
                          :title="`Draws: ${m.draws} (${((m.draws / m.count) * 100).toFixed(1)}%)`"
                        />
                        <!-- Black win bar -->
                        <div
                          class="bg-zinc-950 h-full border-l border-zinc-700"
                          :style="{ width: `${(m.blackWins / m.count) * 100}%` }"
                          :title="`Black Wins: ${m.blackWins} (${((m.blackWins / m.count) * 100).toFixed(1)}%)`"
                        />
                      </div>
                      <div
                        v-else
                        class="flex-1 h-2.5 rounded-full overflow-hidden bg-zinc-800 flex shadow-inner"
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
                  <td class="py-2.5 px-3 text-right text-xs text-zinc-400 whitespace-nowrap">
                    <template v-if="(m.whiteWins + m.draws + m.blackWins) > 0">
                      <span class="text-zinc-200 font-medium">{{ m.whiteWins }}</span>
                      <span class="text-zinc-600 px-1">/</span>
                      <span class="text-zinc-400">{{ m.draws }}</span>
                      <span class="text-zinc-600 px-1">/</span>
                      <span class="text-zinc-400">{{ m.blackWins }}</span>
                    </template>
                    <template v-else>
                      <span class="text-purple-400 font-mono text-xs">Book</span>
                    </template>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>

        <!-- 2. Polyglot Book Content -->
        <div v-else-if="activeTab === 'book'" class="h-full flex flex-col">
          <div v-if="availableBooks.length === 0" class="flex-1 flex flex-col items-center justify-center py-10 px-4 text-center space-y-2">
            <BookOpen class="w-8 h-8 text-zinc-600" />
            <p class="text-sm font-semibold text-zinc-400">No Polyglot opening books (.bin) found on system.</p>
            <p class="text-xs text-zinc-500">Place .bin books in /usr/share/scid/books or ~/.scidvspc/books</p>
          </div>

          <div v-else-if="bookMoves.length === 0" class="flex-1 flex flex-col items-center justify-center py-10 px-4 text-center space-y-3">
            <p class="text-sm text-zinc-400 italic">
              Position out of book in {{ formatBookName(selectedBook) }}.
            </p>
            <div class="flex items-center gap-2">
              <button
                class="px-3 py-1.5 bg-zinc-800 hover:bg-zinc-700 border border-zinc-700 text-zinc-200 rounded-lg text-xs font-semibold transition-all cursor-pointer flex items-center gap-1.5"
                @click="emit('goFirst')"
              >
                <ChevronsLeft class="w-3.5 h-3.5" />
                <span>Start Position</span>
              </button>
              <button
                class="px-3 py-1.5 bg-zinc-800 hover:bg-zinc-700 border border-zinc-700 text-zinc-200 rounded-lg text-xs font-semibold transition-all cursor-pointer flex items-center gap-1.5"
                @click="emit('goPrev')"
              >
                <ChevronLeft class="w-3.5 h-3.5" />
                <span>Step Back</span>
              </button>
            </div>
          </div>

          <div v-else class="table-container flex-1 overflow-y-auto">
            <table class="w-full text-left font-mono text-xs border-collapse">
              <thead class="bg-zinc-950 sticky top-0 text-zinc-400 select-none z-10 border-b border-zinc-800">
                <tr>
                  <th class="py-2.5 px-3 font-semibold">Book Move</th>
                  <th class="py-2.5 px-3 font-semibold text-right">Weight</th>
                  <th class="py-2.5 px-4 font-semibold">Frequency %</th>
                  <th class="py-2.5 px-3 font-semibold text-right text-zinc-500">Learn</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-zinc-800/60">
                <tr
                  v-for="bm in bookMoves"
                  :key="bm.uci"
                  class="hover:bg-purple-950/30 transition-colors cursor-pointer group"
                  :title="`Click to play book move ${bm.san} on board`"
                  @click="emit('playMove', bm.uci || bm.san)"
                >
                  <!-- Move SAN -->
                  <td class="py-2.5 px-3 font-bold text-purple-400 group-hover:text-purple-300 flex items-center gap-2">
                    <span class="text-sm">{{ formatMoveNotation(bm.san, bm.uci) }}</span>
                  </td>

                  <!-- Weight -->
                  <td class="py-2.5 px-3 text-right text-zinc-200 font-semibold">
                    {{ bm.weight.toLocaleString() }}
                  </td>

                  <!-- Frequency % Bar -->
                  <td class="py-2.5 px-4 min-w-[160px]">
                    <div class="flex items-center gap-2">
                      <span class="w-10 text-right text-xs font-semibold text-zinc-300">
                        {{ bm.percentage.toFixed(1) }}%
                      </span>
                      <div class="flex-1 h-2.5 rounded-full overflow-hidden bg-zinc-800 flex shadow-inner">
                        <div
                          class="bg-purple-500 h-full rounded-full"
                          :style="{ width: `${bm.percentage}%` }"
                          :title="`${bm.percentage.toFixed(1)}%`"
                        />
                      </div>
                    </div>
                  </td>

                  <!-- Learn Stat -->
                  <td class="py-2.5 px-3 text-right text-xs text-zinc-500">
                    {{ bm.learn || 0 }}
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>

      <!-- Footer Bar -->
      <div class="px-5 py-2.5 bg-zinc-950 border-t border-zinc-800 flex items-center justify-between text-xs text-zinc-500 select-none">
        <div class="flex items-center gap-3">
          <span>Click any move row to play it on the main board</span>
          <span>•</span>
          <span>F3: Toggle Window</span>
          <span>•</span>
          <span>Esc: Close</span>
        </div>
        <button
          class="px-3 py-1 bg-zinc-800 hover:bg-zinc-700 text-zinc-300 rounded-lg font-medium transition-colors cursor-pointer"
          @click="emit('close')"
        >
          Close
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.table-container::-webkit-scrollbar {
  width: 6px;
}
.table-container::-webkit-scrollbar-thumb {
  background-color: rgba(255, 255, 255, 0.15);
  border-radius: 4px;
}
</style>
