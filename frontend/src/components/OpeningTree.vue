<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { useNotation, formatMoveNotation } from '../utils/notation'
import { BookOpen, Database, RefreshCw } from '@lucide/vue'
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
  treeData: TreeData | null
  loading?: boolean
}>()

const emit = defineEmits<{
  (e: 'playMove', san: string): void
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
  if (!selectedBook.value || !props.treeData?.fen) {
    bookMoves.value = []
    return
  }

  loadingBook.value = true
  try {
    const res = await QueryPolyglotBook(props.treeData.fen, selectedBook.value)
    bookMoves.value = res || []
  } catch (err) {
    console.error('Failed to query polyglot book:', err)
    bookMoves.value = []
  } finally {
    loadingBook.value = false
  }
}

watch(() => props.treeData?.fen, () => {
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
  }
})

onMounted(async () => {
  await loadAvailableBooks()
  if (activeTab.value === 'book') {
    await queryCurrentBook()
  }
})

function formatBookName(path: string) {
  const parts = path.split('/')
  return parts[parts.length - 1] || path
}
</script>

<template>
  <div class="opening-tree flex flex-col h-full bg-zinc-900 border border-zinc-800 rounded-xl overflow-hidden shadow-lg">
    <!-- Header with DB / Polyglot Book Tab Selector -->
    <div class="tree-header flex items-center justify-between p-2 bg-zinc-950 border-b border-zinc-800">
      <div class="flex items-center gap-1.5">
        <!-- DB Tree Tab -->
        <button
          class="px-2.5 py-1 rounded-lg text-xs font-semibold uppercase tracking-wider transition-colors cursor-pointer flex items-center gap-1.5"
          :class="activeTab === 'db' ? 'bg-zinc-800 text-emerald-400' : 'text-zinc-500 hover:text-zinc-300'"
          @click="activeTab = 'db'"
        >
          <Database class="w-3.5 h-3.5" />
          <span>DB Tree</span>
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

      <div class="flex items-center gap-2">
        <!-- Book selector dropdown when Book tab is active -->
        <select
          v-if="activeTab === 'book' && availableBooks.length > 0"
          v-model="selectedBook"
          class="bg-zinc-900 border border-zinc-700 rounded-lg px-2 py-0.5 text-xs text-zinc-300 font-mono focus:outline-hidden focus:border-purple-500 max-w-[140px] truncate"
        >
          <option v-for="b in availableBooks" :key="b" :value="b">
            {{ formatBookName(b) }}
          </option>
        </select>

        <div v-if="loading || loadingBook" class="animate-spin w-3.5 h-3.5 border-2 border-emerald-500 border-t-transparent rounded-full" />
      </div>
    </div>

    <!-- Tree Table: DB Tree View -->
    <div v-if="activeTab === 'db'" class="table-container flex-1 overflow-y-auto">
      <div v-if="movesList.length === 0" class="text-zinc-500 text-xs italic text-center py-8">
        No database games match this position.
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
            @click="emit('playMove', m.move)"
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
                <div class="flex-1 h-2 rounded-full overflow-hidden bg-zinc-800 flex">
                  <!-- White win bar -->
                  <div
                    class="bg-zinc-300 h-full"
                    :style="{ width: `${(m.whiteWins / m.count) * 100}%` }"
                    title="White Wins"
                  />
                  <!-- Draw bar -->
                  <div
                    class="bg-zinc-500 h-full"
                    :style="{ width: `${(m.draws / m.count) * 100}%` }"
                    title="Draws"
                  />
                  <!-- Black win bar -->
                  <div
                    class="bg-zinc-900 h-full border-l border-zinc-700"
                    :style="{ width: `${(m.blackWins / m.count) * 100}%` }"
                    title="Black Wins"
                  />
                </div>
              </div>
            </td>

            <!-- W / D / L Breakdown -->
            <td class="py-2 px-2 text-right text-[11px] text-zinc-400 whitespace-nowrap">
              <span class="text-zinc-200">{{ m.whiteWins }}</span>
              <span class="text-zinc-500">/</span>
              <span class="text-zinc-400">{{ m.draws }}</span>
              <span class="text-zinc-500">/</span>
              <span class="text-zinc-400">{{ m.blackWins }}</span>
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

      <div v-else-if="bookMoves.length === 0" class="text-zinc-500 text-xs italic text-center py-8">
        Position out of book in {{ formatBookName(selectedBook) }}.
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
            @click="emit('playMove', bm.san)"
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
