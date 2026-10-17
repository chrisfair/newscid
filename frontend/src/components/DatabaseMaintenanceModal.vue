<script setup lang="ts">
import { ref, onMounted } from 'vue'
import {
  Wrench,
  BarChart3,
  Archive,
  ArrowUpDown,
  Copy,
  CheckCircle2,
  AlertTriangle,
  XCircle,
  UploadCloud,
  RefreshCw,
  X,
  FileText,
  Sliders,
  FolderOpen,
  Check,
  Trash2
} from '@lucide/vue'
import {
  GetDatabaseStatistics,
  CompactDatabase,
  SortDatabase,
  FindDuplicateGames,
  DeleteDuplicateGames,
  VerifyDatabase,
  ImportPGNBatch,
  ImportPGNFileBatch,
  OpenPGNFileDialog
} from '../../wailsjs/go/main/App'

const props = defineProps<{
  dbName?: string
  dbPath?: string
  totalGames: number
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'dbUpdated'): void
}>()

type TabType = 'overview' | 'compact' | 'sort' | 'duplicates' | 'verify' | 'import'
const activeTab = ref<TabType>('overview')

// Loading states
const loading = ref(false)
const statusMessage = ref('')
const errorMessage = ref('')

// Overview / Statistics State
interface NameCount {
  name: string
  count: number
}
interface StatsData {
  totalGames: number
  activeGames: number
  deletedGames: number
  earliestYear: number
  latestYear: number
  whiteWins: number
  blackWins: number
  draws: number
  unknownResults: number
  averageElo: number
  maxElo: number
  minElo: number
  topOpenings: NameCount[]
  topPlayers: NameCount[]
  totalSizeBytes: number
  fileSizes: Record<string, number>
}
const stats = ref<StatsData | null>(null)

// Compaction State
interface CompactResult {
  originalGames: number
  compactedGames: number
  purgedGames: number
  bytesReclaimed: number
  originalSg4Size: number
  compactedSg4Size: number
}
const compactResult = ref<CompactResult | null>(null)

// Sort State
const sortField = ref('date')
const sortAscending = ref(true)

// Duplicates State
const dupExactMoves = ref(true)
const dupMatchPlayers = ref(true)
const dupMatchDate = ref(false)
const dupMatchEvent = ref(false)
const dupMatchResult = ref(false)

interface DuplicateGroup {
  key: string
  gameIds: number[]
  count: number
  games: any[]
}
const duplicateGroups = ref<DuplicateGroup[]>([])
const duplicatesScanned = ref(false)
const deletedDuplicatesCount = ref<number | null>(null)

// Verification State
interface VerificationReport {
  totalGames: number
  activeGames: number
  deletedGames: number
  corruptedGames: number
  status: 'OK' | 'WARNING' | 'CORRUPTED'
  errors: string[]
  warnings: string[]
  fileSizes: Record<string, number>
  nameCounts: Record<string, number>
}
const verifyReport = ref<VerificationReport | null>(null)

// Batch Import State
const pgnBatchText = ref('')
interface ImportResult {
  totalParsed: number
  imported: number
  skipped: number
  errors: number
  durationMs: number
  firstGameId: number
  lastGameId: number
}
const importResult = ref<ImportResult | null>(null)

// Lifecycle
onMounted(() => {
  loadStatistics()
})

function formatBytes(bytes: number): string {
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  if (bytes < 1024 * 1024 * 1024) return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
  return (bytes / (1024 * 1024 * 1024)).toFixed(2) + ' GB'
}

async function loadStatistics() {
  loading.value = true
  errorMessage.value = ''
  try {
    const res = await GetDatabaseStatistics()
    stats.value = res as any
  } catch (err: any) {
    errorMessage.value = err?.message || 'Failed to load database statistics'
  } finally {
    loading.value = false
  }
}

async function handleCompact() {
  loading.value = true
  errorMessage.value = ''
  statusMessage.value = ''
  compactResult.value = null
  try {
    const res = await CompactDatabase()
    compactResult.value = res as any
    statusMessage.value = `Successfully compacted database. Reclaimed ${formatBytes(res.bytesReclaimed)}.`
    emit('dbUpdated')
    await loadStatistics()
  } catch (err: any) {
    errorMessage.value = err?.message || 'Database compaction failed'
  } finally {
    loading.value = false
  }
}

async function handleSort() {
  loading.value = true
  errorMessage.value = ''
  statusMessage.value = ''
  try {
    await SortDatabase({
      field: sortField.value as any,
      ascending: sortAscending.value
    })
    statusMessage.value = `Database index sorted successfully by ${sortField.value} (${sortAscending.value ? 'ascending' : 'descending'}).`
    emit('dbUpdated')
  } catch (err: any) {
    errorMessage.value = err?.message || 'Database sorting failed'
  } finally {
    loading.value = false
  }
}

async function handleScanDuplicates() {
  loading.value = true
  errorMessage.value = ''
  statusMessage.value = ''
  duplicatesScanned.value = false
  deletedDuplicatesCount.value = null
  try {
    const res = await FindDuplicateGames({
      exactMoves: dupExactMoves.value,
      matchPlayers: dupMatchPlayers.value,
      matchDate: dupMatchDate.value,
      matchEvent: dupMatchEvent.value,
      matchResult: dupMatchResult.value
    })
    duplicateGroups.value = (res || []) as any[]
    duplicatesScanned.value = true
    if (duplicateGroups.value.length === 0) {
      statusMessage.value = 'No duplicate games detected matching the criteria.'
    } else {
      const totalDupCopies = duplicateGroups.value.reduce((acc, g) => acc + (g.count - 1), 0)
      statusMessage.value = `Found ${duplicateGroups.value.length} duplicate groups (${totalDupCopies} redundant copies).`
    }
  } catch (err: any) {
    errorMessage.value = err?.message || 'Scanning duplicates failed'
  } finally {
    loading.value = false
  }
}

async function handleDeleteDuplicates() {
  if (duplicateGroups.value.length === 0) return
  loading.value = true
  errorMessage.value = ''
  statusMessage.value = ''
  try {
    const deletedCount = await DeleteDuplicateGames({
      exactMoves: dupExactMoves.value,
      matchPlayers: dupMatchPlayers.value,
      matchDate: dupMatchDate.value,
      matchEvent: dupMatchEvent.value,
      matchResult: dupMatchResult.value
    })
    deletedDuplicatesCount.value = deletedCount
    statusMessage.value = `Marked ${deletedCount} redundant duplicate games as deleted.`
    duplicateGroups.value = []
    emit('dbUpdated')
    await loadStatistics()
  } catch (err: any) {
    errorMessage.value = err?.message || 'Deleting duplicates failed'
  } finally {
    loading.value = false
  }
}

async function handleVerify() {
  loading.value = true
  errorMessage.value = ''
  statusMessage.value = ''
  try {
    const res = await VerifyDatabase()
    verifyReport.value = res as any
  } catch (err: any) {
    errorMessage.value = err?.message || 'Integrity check failed'
  } finally {
    loading.value = false
  }
}

async function handleImportText() {
  if (!pgnBatchText.value.trim()) {
    errorMessage.value = 'Please paste PGN games to import.'
    return
  }
  loading.value = true
  errorMessage.value = ''
  statusMessage.value = ''
  importResult.value = null
  try {
    const res = await ImportPGNBatch(pgnBatchText.value)
    importResult.value = res as any
    statusMessage.value = `Imported ${res.imported} games in ${res.durationMs}ms.`
    pgnBatchText.value = ''
    emit('dbUpdated')
    await loadStatistics()
  } catch (err: any) {
    errorMessage.value = err?.message || 'Batch PGN text import failed'
  } finally {
    loading.value = false
  }
}

async function handleImportFile() {
  errorMessage.value = ''
  statusMessage.value = ''
  try {
    const filePath = await OpenPGNFileDialog()
    if (!filePath) return
    loading.value = true
    const res = await ImportPGNFileBatch(filePath)
    importResult.value = res as any
    statusMessage.value = `Imported ${res.imported} games from ${filePath} in ${res.durationMs}ms.`
    emit('dbUpdated')
    await loadStatistics()
  } catch (err: any) {
    errorMessage.value = err?.message || 'PGN file import failed'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="fixed inset-0 bg-black/70 backdrop-blur-xs flex items-center justify-center z-50 p-4 animate-fade-in" @click.self="emit('close')">
    <div class="bg-zinc-900 border border-zinc-700/80 w-full max-w-4xl max-h-[90vh] rounded-2xl shadow-2xl flex flex-col overflow-hidden text-zinc-200">
      
      <!-- MODAL HEADER -->
      <div class="px-6 py-4 border-b border-zinc-800 flex items-center justify-between bg-zinc-950/60">
        <div class="flex items-center gap-3">
          <div class="w-9 h-9 rounded-xl bg-emerald-500/10 border border-emerald-500/30 flex items-center justify-center text-emerald-400">
            <Wrench class="w-5 h-5" />
          </div>
          <div>
            <h2 class="font-bold text-base text-zinc-100 flex items-center gap-2">
              Database Maintenance & Tools
              <span v-if="dbName" class="text-xs font-normal text-emerald-400 bg-emerald-950/40 border border-emerald-800/50 px-2 py-0.5 rounded-full">
                {{ dbName }} ({{ totalGames.toLocaleString() }} games)
              </span>
            </h2>
            <p class="text-xs text-zinc-400">Manage, optimize, defragment, sort, and verify Scid chess database records</p>
          </div>
        </div>

        <button
          class="p-1.5 rounded-lg text-zinc-400 hover:text-white hover:bg-zinc-800 transition-colors cursor-pointer"
          @click="emit('close')"
        >
          <X class="w-5 h-5" />
        </button>
      </div>

      <!-- TABS NAVIGATION -->
      <div class="flex border-b border-zinc-800 bg-zinc-950/40 px-4 text-xs font-medium overflow-x-auto">
        <button
          class="px-4 py-2.5 border-b-2 flex items-center gap-2 transition-colors cursor-pointer"
          :class="activeTab === 'overview' ? 'border-emerald-500 text-emerald-400' : 'border-transparent text-zinc-400 hover:text-zinc-200'"
          @click="activeTab = 'overview'"
        >
          <BarChart3 class="w-4 h-4" />
          Overview & Stats
        </button>

        <button
          class="px-4 py-2.5 border-b-2 flex items-center gap-2 transition-colors cursor-pointer"
          :class="activeTab === 'compact' ? 'border-emerald-500 text-emerald-400' : 'border-transparent text-zinc-400 hover:text-zinc-200'"
          @click="activeTab = 'compact'"
        >
          <Archive class="w-4 h-4" />
          Compact & Defrag
        </button>

        <button
          class="px-4 py-2.5 border-b-2 flex items-center gap-2 transition-colors cursor-pointer"
          :class="activeTab === 'sort' ? 'border-emerald-500 text-emerald-400' : 'border-transparent text-zinc-400 hover:text-zinc-200'"
          @click="activeTab = 'sort'"
        >
          <ArrowUpDown class="w-4 h-4" />
          Sort Database
        </button>

        <button
          class="px-4 py-2.5 border-b-2 flex items-center gap-2 transition-colors cursor-pointer"
          :class="activeTab === 'duplicates' ? 'border-emerald-500 text-emerald-400' : 'border-transparent text-zinc-400 hover:text-zinc-200'"
          @click="activeTab = 'duplicates'"
        >
          <Copy class="w-4 h-4" />
          Duplicate Games
        </button>

        <button
          class="px-4 py-2.5 border-b-2 flex items-center gap-2 transition-colors cursor-pointer"
          :class="activeTab === 'verify' ? 'border-emerald-500 text-emerald-400' : 'border-transparent text-zinc-400 hover:text-zinc-200'"
          @click="activeTab = 'verify'"
        >
          <CheckCircle2 class="w-4 h-4" />
          Integrity Check
        </button>

        <button
          class="px-4 py-2.5 border-b-2 flex items-center gap-2 transition-colors cursor-pointer"
          :class="activeTab === 'import' ? 'border-emerald-500 text-emerald-400' : 'border-transparent text-zinc-400 hover:text-zinc-200'"
          @click="activeTab = 'import'"
        >
          <UploadCloud class="w-4 h-4" />
          Batch PGN Import
        </button>
      </div>

      <!-- NOTIFICATIONS -->
      <div v-if="statusMessage" class="mx-6 mt-4 p-3 bg-emerald-950/40 border border-emerald-800/60 rounded-xl text-xs text-emerald-300 flex items-center gap-2">
        <Check class="w-4 h-4 shrink-0 text-emerald-400" />
        <span>{{ statusMessage }}</span>
      </div>
      <div v-if="errorMessage" class="mx-6 mt-4 p-3 bg-red-950/40 border border-red-800/60 rounded-xl text-xs text-red-300 flex items-center gap-2">
        <AlertTriangle class="w-4 h-4 shrink-0 text-red-400" />
        <span>{{ errorMessage }}</span>
      </div>

      <!-- TAB CONTENTS -->
      <div class="flex-1 overflow-y-auto p-6 space-y-6">

        <!-- 1. OVERVIEW & STATS -->
        <div v-if="activeTab === 'overview'" class="space-y-6">
          <div class="flex items-center justify-between">
            <h3 class="font-bold text-sm text-zinc-100 flex items-center gap-2">
              <BarChart3 class="w-4 h-4 text-emerald-400" />
              Database Analytics & Demographic Overview
            </h3>
            <button
              class="px-3 py-1 bg-zinc-800 hover:bg-zinc-700 text-zinc-300 rounded-lg text-xs flex items-center gap-1.5 transition-colors cursor-pointer"
              :disabled="loading"
              @click="loadStatistics"
            >
              <RefreshCw class="w-3.5 h-3.5" :class="{ 'animate-spin': loading }" />
              Refresh
            </button>
          </div>

          <div v-if="stats" class="grid grid-cols-2 sm:grid-cols-4 gap-3">
            <div class="bg-zinc-950/60 border border-zinc-800/80 p-3.5 rounded-xl">
              <span class="text-[11px] text-zinc-500 uppercase tracking-wider font-semibold">Total Games</span>
              <p class="text-xl font-bold text-zinc-100 mt-0.5">{{ stats.totalGames.toLocaleString() }}</p>
              <span class="text-[10px] text-zinc-500">Active: {{ stats.activeGames.toLocaleString() }} | Deleted: {{ stats.deletedGames.toLocaleString() }}</span>
            </div>

            <div class="bg-zinc-950/60 border border-zinc-800/80 p-3.5 rounded-xl">
              <span class="text-[11px] text-zinc-500 uppercase tracking-wider font-semibold">Year Range</span>
              <p class="text-xl font-bold text-zinc-100 mt-0.5">
                {{ stats.earliestYear > 0 && stats.earliestYear < 9999 ? stats.earliestYear : '—' }}
                <span class="text-sm font-normal text-zinc-400">to</span>
                {{ stats.latestYear > 0 ? stats.latestYear : '—' }}
              </p>
              <span class="text-[10px] text-zinc-500">Chronological span</span>
            </div>

            <div class="bg-zinc-950/60 border border-zinc-800/80 p-3.5 rounded-xl">
              <span class="text-[11px] text-zinc-500 uppercase tracking-wider font-semibold">Elo Ratings</span>
              <p class="text-xl font-bold text-zinc-100 mt-0.5">{{ stats.averageElo > 0 ? stats.averageElo : '—' }}</p>
              <span class="text-[10px] text-zinc-500">Max: {{ stats.maxElo > 0 ? stats.maxElo : '—' }} | Min: {{ stats.minElo < 9999 ? stats.minElo : '—' }}</span>
            </div>

            <div class="bg-zinc-950/60 border border-zinc-800/80 p-3.5 rounded-xl">
              <span class="text-[11px] text-zinc-500 uppercase tracking-wider font-semibold">Storage Size</span>
              <p class="text-xl font-bold text-zinc-100 mt-0.5">{{ formatBytes(stats.totalSizeBytes) }}</p>
              <span class="text-[10px] text-zinc-500">Index (.si4): {{ formatBytes(stats.fileSizes?.si4 || 0) }}</span>
            </div>
          </div>

          <!-- Result Distribution -->
          <div v-if="stats && stats.totalGames > 0" class="bg-zinc-950/40 border border-zinc-800/80 p-4 rounded-xl space-y-2">
            <span class="text-xs font-semibold text-zinc-300">Result Breakdown</span>
            <div class="h-3 w-full bg-zinc-800 rounded-full overflow-hidden flex">
              <div
                class="bg-emerald-500 h-full"
                :style="{ width: ((stats.whiteWins / stats.totalGames) * 100) + '%' }"
                :title="`White Wins: ${stats.whiteWins} (${((stats.whiteWins / stats.totalGames) * 100).toFixed(1)}%)`"
              />
              <div
                class="bg-zinc-400 h-full"
                :style="{ width: ((stats.draws / stats.totalGames) * 100) + '%' }"
                :title="`Draws: ${stats.draws} (${((stats.draws / stats.totalGames) * 100).toFixed(1)}%)`"
              />
              <div
                class="bg-blue-500 h-full"
                :style="{ width: ((stats.blackWins / stats.totalGames) * 100) + '%' }"
                :title="`Black Wins: ${stats.blackWins} (${((stats.blackWins / stats.totalGames) * 100).toFixed(1)}%)`"
              />
              <div
                class="bg-amber-500/60 h-full"
                :style="{ width: ((stats.unknownResults / stats.totalGames) * 100) + '%' }"
                :title="`Ongoing/Unknown: ${stats.unknownResults}`"
              />
            </div>
            <div class="flex items-center justify-between text-[11px] text-zinc-400 pt-1">
              <span class="flex items-center gap-1.5"><span class="w-2.5 h-2.5 rounded-xs bg-emerald-500 inline-block" /> 1-0 White ({{ stats.whiteWins.toLocaleString() }})</span>
              <span class="flex items-center gap-1.5"><span class="w-2.5 h-2.5 rounded-xs bg-zinc-400 inline-block" /> 1/2-1/2 Draw ({{ stats.draws.toLocaleString() }})</span>
              <span class="flex items-center gap-1.5"><span class="w-2.5 h-2.5 rounded-xs bg-blue-500 inline-block" /> 0-1 Black ({{ stats.blackWins.toLocaleString() }})</span>
              <span class="flex items-center gap-1.5"><span class="w-2.5 h-2.5 rounded-xs bg-amber-500/60 inline-block" /> * Other ({{ stats.unknownResults.toLocaleString() }})</span>
            </div>
          </div>

          <!-- Top Openings & Top Players -->
          <div v-if="stats" class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div class="bg-zinc-950/40 border border-zinc-800/80 p-4 rounded-xl space-y-3">
              <span class="text-xs font-semibold text-zinc-200 flex items-center gap-2">
                <Sliders class="w-3.5 h-3.5 text-purple-400" />
                Most Frequent ECO Openings
              </span>
              <div v-if="stats.topOpenings && stats.topOpenings.length > 0" class="space-y-1.5">
                <div
                  v-for="(item, idx) in stats.topOpenings"
                  :key="idx"
                  class="flex items-center justify-between text-xs py-1 border-b border-zinc-900 last:border-0"
                >
                  <span class="font-mono text-zinc-300 font-medium">{{ item.name }}</span>
                  <span class="text-zinc-400 font-mono">{{ item.count.toLocaleString() }} games</span>
                </div>
              </div>
              <div v-else class="text-xs text-zinc-500 italic py-2">No ECO codes recorded in database.</div>
            </div>

            <div class="bg-zinc-950/40 border border-zinc-800/80 p-4 rounded-xl space-y-3">
              <span class="text-xs font-semibold text-zinc-200 flex items-center gap-2">
                <FileText class="w-3.5 h-3.5 text-blue-400" />
                Most Frequent Players
              </span>
              <div v-if="stats.topPlayers && stats.topPlayers.length > 0" class="space-y-1.5">
                <div
                  v-for="(item, idx) in stats.topPlayers"
                  :key="idx"
                  class="flex items-center justify-between text-xs py-1 border-b border-zinc-900 last:border-0"
                >
                  <span class="text-zinc-300 truncate max-w-[200px]">{{ item.name }}</span>
                  <span class="text-zinc-400 font-mono">{{ item.count.toLocaleString() }} games</span>
                </div>
              </div>
              <div v-else class="text-xs text-zinc-500 italic py-2">No player records found.</div>
            </div>
          </div>
        </div>

        <!-- 2. COMPACT & DEFRAGMENT -->
        <div v-if="activeTab === 'compact'" class="space-y-5">
          <div class="bg-zinc-950/50 border border-zinc-800 p-5 rounded-2xl space-y-3">
            <h3 class="font-bold text-sm text-zinc-100 flex items-center gap-2">
              <Archive class="w-4 h-4 text-emerald-400" />
              Physical Compaction & Storage Reclamation
            </h3>
            <p class="text-xs text-zinc-400 leading-relaxed">
              In Scid format, games flagged for deletion remain in the index (<code class="text-emerald-400 font-mono">.si4</code>) and binary game move stream (<code class="text-emerald-400 font-mono">.sg4</code>).
              Compacting defragments the database by rewriting both files without deleted records, updating offsets, and shrinking disk consumption.
            </p>

            <div v-if="stats" class="p-3 bg-zinc-900 rounded-xl flex items-center justify-between text-xs">
              <span class="text-zinc-400">Current Deleted Games:</span>
              <span class="font-bold font-mono" :class="stats.deletedGames > 0 ? 'text-amber-400' : 'text-zinc-500'">
                {{ stats.deletedGames.toLocaleString() }} games ready to purge
              </span>
            </div>

            <div class="pt-2">
              <button
                class="px-5 py-2.5 bg-emerald-600 hover:bg-emerald-500 text-white font-medium rounded-xl text-xs flex items-center gap-2 transition-colors cursor-pointer shadow-lg shadow-emerald-950/50"
                :disabled="loading"
                @click="handleCompact"
              >
                <Archive class="w-4 h-4" />
                <span v-if="loading">Compacting Database...</span>
                <span v-else>Compact & Defragment Database Now</span>
              </button>
            </div>
          </div>

          <div v-if="compactResult" class="bg-emerald-950/30 border border-emerald-800/60 p-4 rounded-xl space-y-2 text-xs">
            <h4 class="font-bold text-emerald-300 flex items-center gap-1.5">
              <CheckCircle2 class="w-4 h-4 text-emerald-400" /> Compaction Summary
            </h4>
            <div class="grid grid-cols-2 sm:grid-cols-3 gap-3 text-zinc-300 pt-1">
              <div>Games Purged: <span class="font-mono font-bold text-emerald-400">{{ compactResult.purgedGames }}</span></div>
              <div>Active Games Remaining: <span class="font-mono font-bold text-zinc-100">{{ compactResult.compactedGames }}</span></div>
              <div>Disk Space Reclaimed: <span class="font-mono font-bold text-emerald-400">{{ formatBytes(compactResult.bytesReclaimed) }}</span></div>
            </div>
          </div>
        </div>

        <!-- 3. SORT DATABASE -->
        <div v-if="activeTab === 'sort'" class="space-y-5">
          <div class="bg-zinc-950/50 border border-zinc-800 p-5 rounded-2xl space-y-4">
            <h3 class="font-bold text-sm text-zinc-100 flex items-center gap-2">
              <ArrowUpDown class="w-4 h-4 text-emerald-400" />
              Reorder Index (.si4) Records
            </h3>
            <p class="text-xs text-zinc-400 leading-relaxed">
              Because each index record stores an absolute byte pointer to game data in <code class="text-emerald-400 font-mono">.sg4</code>, sorting only re-indexes the lightweight <code class="text-emerald-400 font-mono">.si4</code> file, completing in sub-second time.
            </p>

            <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div class="space-y-1.5">
                <label class="text-[11px] font-semibold text-zinc-400 uppercase tracking-wider">Sort Attribute</label>
                <select
                  v-model="sortField"
                  class="w-full bg-zinc-900 border border-zinc-700/80 rounded-xl px-3 py-2 text-xs text-zinc-200 focus:outline-hidden focus:border-emerald-500"
                >
                  <option value="date">Date</option>
                  <option value="eco">ECO Code</option>
                  <option value="white">White Player</option>
                  <option value="black">Black Player</option>
                  <option value="player">Player (White then Black)</option>
                  <option value="whiteElo">White Elo Rating</option>
                  <option value="blackElo">Black Elo Rating</option>
                  <option value="maxElo">Highest Elo Rating</option>
                  <option value="event">Event Name</option>
                  <option value="site">Site Location</option>
                  <option value="round">Round</option>
                  <option value="result">Result</option>
                  <option value="moves">Number of Moves</option>
                </select>
              </div>

              <div class="space-y-1.5">
                <label class="text-[11px] font-semibold text-zinc-400 uppercase tracking-wider">Direction</label>
                <div class="flex items-center gap-4 pt-1.5">
                  <label class="flex items-center gap-2 text-xs cursor-pointer">
                    <input type="radio" v-model="sortAscending" :value="true" class="text-emerald-500 accent-emerald-500" />
                    <span>Ascending (A-Z, Oldest first)</span>
                  </label>
                  <label class="flex items-center gap-2 text-xs cursor-pointer">
                    <input type="radio" v-model="sortAscending" :value="false" class="text-emerald-500 accent-emerald-500" />
                    <span>Descending (Z-A, Newest first)</span>
                  </label>
                </div>
              </div>
            </div>

            <div class="pt-2">
              <button
                class="px-5 py-2.5 bg-emerald-600 hover:bg-emerald-500 text-white font-medium rounded-xl text-xs flex items-center gap-2 transition-colors cursor-pointer shadow-lg shadow-emerald-950/50"
                :disabled="loading"
                @click="handleSort"
              >
                <ArrowUpDown class="w-4 h-4" />
                <span v-if="loading">Sorting Index...</span>
                <span v-else>Sort Database Index Now</span>
              </button>
            </div>
          </div>
        </div>

        <!-- 4. DUPLICATE GAMES -->
        <div v-if="activeTab === 'duplicates'" class="space-y-5">
          <div class="bg-zinc-950/50 border border-zinc-800 p-5 rounded-2xl space-y-4">
            <h3 class="font-bold text-sm text-zinc-100 flex items-center gap-2">
              <Copy class="w-4 h-4 text-emerald-400" />
              Duplicate Game Detection & Removal
            </h3>
            <p class="text-xs text-zinc-400 leading-relaxed">
              Detect identical games using cryptographic move hashes or metadata combinations. Duplicate copies can be marked deleted in bulk while preserving the initial game.
            </p>

            <div class="grid grid-cols-2 sm:grid-cols-3 gap-3">
              <label class="flex items-center gap-2 p-2.5 bg-zinc-900 border border-zinc-800 rounded-xl text-xs cursor-pointer">
                <input type="checkbox" v-model="dupExactMoves" class="accent-emerald-500" />
                <span>Exact Move Hashes (SHA-256)</span>
              </label>
              <label class="flex items-center gap-2 p-2.5 bg-zinc-900 border border-zinc-800 rounded-xl text-xs cursor-pointer">
                <input type="checkbox" v-model="dupMatchPlayers" class="accent-emerald-500" />
                <span>Match Player Names</span>
              </label>
              <label class="flex items-center gap-2 p-2.5 bg-zinc-900 border border-zinc-800 rounded-xl text-xs cursor-pointer">
                <input type="checkbox" v-model="dupMatchDate" class="accent-emerald-500" />
                <span>Match Exact Date</span>
              </label>
              <label class="flex items-center gap-2 p-2.5 bg-zinc-900 border border-zinc-800 rounded-xl text-xs cursor-pointer">
                <input type="checkbox" v-model="dupMatchEvent" class="accent-emerald-500" />
                <span>Match Event Name</span>
              </label>
              <label class="flex items-center gap-2 p-2.5 bg-zinc-900 border border-zinc-800 rounded-xl text-xs cursor-pointer">
                <input type="checkbox" v-model="dupMatchResult" class="accent-emerald-500" />
                <span>Match Game Result</span>
              </label>
            </div>

            <div class="flex items-center gap-3 pt-2">
              <button
                class="px-4 py-2 bg-emerald-600 hover:bg-emerald-500 text-white font-medium rounded-xl text-xs flex items-center gap-2 transition-colors cursor-pointer"
                :disabled="loading"
                @click="handleScanDuplicates"
              >
                <RefreshCw class="w-3.5 h-3.5" :class="{ 'animate-spin': loading }" />
                Scan for Duplicates
              </button>

              <button
                v-if="duplicateGroups.length > 0"
                class="px-4 py-2 bg-red-600/90 hover:bg-red-500 text-white font-medium rounded-xl text-xs flex items-center gap-2 transition-colors cursor-pointer"
                :disabled="loading"
                @click="handleDeleteDuplicates"
              >
                <Trash2 class="w-3.5 h-3.5" />
                Delete Redundant Copies
              </button>
            </div>
          </div>

          <!-- Duplicate Results Table -->
          <div v-if="duplicateGroups.length > 0" class="space-y-3">
            <h4 class="font-bold text-xs text-zinc-300 uppercase tracking-wider">Identified Duplicate Clusters</h4>
            <div class="space-y-2 max-h-72 overflow-y-auto pr-1">
              <div
                v-for="(grp, idx) in duplicateGroups"
                :key="idx"
                class="p-3 bg-zinc-950/60 border border-zinc-800 rounded-xl space-y-2"
              >
                <div class="flex items-center justify-between text-xs">
                  <span class="font-semibold text-emerald-400">Cluster #{{ idx + 1 }}: {{ grp.count }} identical games</span>
                  <span class="text-[10px] text-zinc-500 font-mono">Game IDs: {{ grp.gameIds.join(', ') }}</span>
                </div>
                <div class="text-[11px] text-zinc-300">
                  <div v-for="g in grp.games" :key="g.id" class="flex items-center gap-2 py-0.5 text-zinc-400 font-mono">
                    <span class="text-zinc-500">#{{ g.id }}</span>
                    <span class="text-zinc-200">{{ g.white }} vs {{ g.black }}</span>
                    <span>[{{ g.result }}]</span>
                    <span class="text-zinc-500">{{ g.date }}</span>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- 5. INTEGRITY CHECK (VERIFY) -->
        <div v-if="activeTab === 'verify'" class="space-y-5">
          <div class="bg-zinc-950/50 border border-zinc-800 p-5 rounded-2xl space-y-4">
            <h3 class="font-bold text-sm text-zinc-100 flex items-center gap-2">
              <CheckCircle2 class="w-4 h-4 text-emerald-400" />
              Comprehensive Database Verification
            </h3>
            <p class="text-xs text-zinc-400 leading-relaxed">
              Verify header magic signatures, index size consistency, boundary offsets in the game move stream, and cross-reference NameBase dictionaries.
            </p>

            <button
              class="px-5 py-2.5 bg-emerald-600 hover:bg-emerald-500 text-white font-medium rounded-xl text-xs flex items-center gap-2 transition-colors cursor-pointer shadow-lg shadow-emerald-950/50"
              :disabled="loading"
              @click="handleVerify"
            >
              <CheckCircle2 class="w-4 h-4" />
              <span v-if="loading">Running Integrity Check...</span>
              <span v-else>Run Integrity Check</span>
            </button>
          </div>

          <div v-if="verifyReport" class="bg-zinc-950/60 border border-zinc-800 p-5 rounded-2xl space-y-4 text-xs">
            <div class="flex items-center justify-between">
              <span class="font-bold text-zinc-200">Database Status:</span>
              <span
                class="px-3 py-1 rounded-full font-bold text-xs uppercase tracking-wider"
                :class="{
                  'bg-emerald-950 text-emerald-300 border border-emerald-700': verifyReport.status === 'OK',
                  'bg-amber-950 text-amber-300 border border-amber-700': verifyReport.status === 'WARNING',
                  'bg-red-950 text-red-300 border border-red-700': verifyReport.status === 'CORRUPTED',
                }"
              >
                {{ verifyReport.status }}
              </span>
            </div>

            <div class="grid grid-cols-2 sm:grid-cols-4 gap-3 text-zinc-300">
              <div class="bg-zinc-900 p-2.5 rounded-lg">Total Games: <span class="font-bold">{{ verifyReport.totalGames }}</span></div>
              <div class="bg-zinc-900 p-2.5 rounded-lg">Active Games: <span class="font-bold text-emerald-400">{{ verifyReport.activeGames }}</span></div>
              <div class="bg-zinc-900 p-2.5 rounded-lg">Deleted Games: <span class="font-bold text-amber-400">{{ verifyReport.deletedGames }}</span></div>
              <div class="bg-zinc-900 p-2.5 rounded-lg">Corrupted: <span class="font-bold" :class="verifyReport.corruptedGames > 0 ? 'text-red-400' : 'text-zinc-400'">{{ verifyReport.corruptedGames }}</span></div>
            </div>

            <div v-if="verifyReport.warnings && verifyReport.warnings.length > 0" class="space-y-1 text-amber-300 bg-amber-950/30 border border-amber-800/40 p-3 rounded-xl">
              <span class="font-bold block">Warnings:</span>
              <ul class="list-disc list-inside space-y-0.5 font-mono text-[11px]">
                <li v-for="(w, idx) in verifyReport.warnings" :key="idx">{{ w }}</li>
              </ul>
            </div>

            <div v-if="verifyReport.errors && verifyReport.errors.length > 0" class="space-y-1 text-red-300 bg-red-950/30 border border-red-800/40 p-3 rounded-xl">
              <span class="font-bold block">Errors:</span>
              <ul class="list-disc list-inside space-y-0.5 font-mono text-[11px]">
                <li v-for="(e, idx) in verifyReport.errors" :key="idx">{{ e }}</li>
              </ul>
            </div>
          </div>
        </div>

        <!-- 6. BATCH PGN IMPORT -->
        <div v-if="activeTab === 'import'" class="space-y-5">
          <div class="bg-zinc-950/50 border border-zinc-800 p-5 rounded-2xl space-y-4">
            <h3 class="font-bold text-sm text-zinc-100 flex items-center gap-2">
              <UploadCloud class="w-4 h-4 text-emerald-400" />
              High-Performance Batch PGN Import
            </h3>
            <p class="text-xs text-zinc-400 leading-relaxed">
              Batch-append multiple games from an external <code class="text-emerald-400 font-mono">.pgn</code> file directly into the active database, or paste raw PGN content below.
            </p>

            <div class="flex items-center gap-3">
              <button
                class="px-4 py-2 bg-emerald-600 hover:bg-emerald-500 text-white font-medium rounded-xl text-xs flex items-center gap-2 transition-colors cursor-pointer"
                :disabled="loading"
                @click="handleImportFile"
              >
                <FolderOpen class="w-4 h-4" />
                Select & Import PGN File...
              </button>
            </div>

            <div class="space-y-2 pt-2">
              <label class="text-[11px] font-semibold text-zinc-400 uppercase tracking-wider">Or Paste PGN Games Text</label>
              <textarea
                v-model="pgnBatchText"
                rows="6"
                placeholder="[Event &quot;Tournament&quot;]&#10;[Site &quot;City&quot;]&#10;1. e4 e5 2. Nf3 Nc6 1-0&#10;&#10;[Event &quot;Round 2&quot;]..."
                class="w-full bg-zinc-900 border border-zinc-700/80 rounded-xl p-3 text-xs font-mono text-zinc-200 placeholder-zinc-500 focus:outline-hidden focus:border-emerald-500"
              />
              <button
                class="px-4 py-2 bg-zinc-800 hover:bg-zinc-700 text-zinc-200 font-medium rounded-xl text-xs flex items-center gap-2 transition-colors cursor-pointer"
                :disabled="loading || !pgnBatchText.trim()"
                @click="handleImportText"
              >
                <UploadCloud class="w-3.5 h-3.5" />
                Import Pasted PGN Text
              </button>
            </div>
          </div>

          <div v-if="importResult" class="bg-emerald-950/30 border border-emerald-800/60 p-4 rounded-xl space-y-2 text-xs">
            <h4 class="font-bold text-emerald-300 flex items-center gap-1.5">
              <CheckCircle2 class="w-4 h-4 text-emerald-400" /> Batch Import Complete
            </h4>
            <div class="grid grid-cols-2 sm:grid-cols-4 gap-3 text-zinc-300 pt-1">
              <div>Parsed: <span class="font-mono font-bold text-zinc-100">{{ importResult.totalParsed }}</span></div>
              <div>Imported: <span class="font-mono font-bold text-emerald-400">{{ importResult.imported }}</span></div>
              <div>Errors: <span class="font-mono font-bold" :class="importResult.errors > 0 ? 'text-red-400' : 'text-zinc-400'">{{ importResult.errors }}</span></div>
              <div>Duration: <span class="font-mono font-bold text-zinc-100">{{ importResult.durationMs }}ms</span></div>
            </div>
          </div>
        </div>

      </div>

      <!-- MODAL FOOTER -->
      <div class="px-6 py-3 border-t border-zinc-800 bg-zinc-950/60 flex items-center justify-between">
        <span class="text-[11px] text-zinc-500">NewScid Native Database Maintenance Engine</span>
        <button
          class="px-4 py-1.5 bg-zinc-800 hover:bg-zinc-700 text-zinc-300 font-medium rounded-lg text-xs transition-colors cursor-pointer"
          @click="emit('close')"
        >
          Close
        </button>
      </div>

    </div>
  </div>
</template>
