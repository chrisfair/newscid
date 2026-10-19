<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { Database, Download, X, CheckCircle, AlertCircle, Globe, Filter, Sparkles, HardDrive, Layers } from '@lucide/vue'
import {
  GetMegaDatabaseCatalog,
  FetchLichessDatabaseFeed,
  ImportMegaDatabase,
  BuildConsolidatedDatabase,
  ConsolidateOnlineCatalogDatabases,
} from '../../wailsjs/go/main/App'
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'

const props = defineProps<{
  dbOpen?: boolean
  dbName?: string
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'imported', count: number): void
}>()

interface MegaDatabaseSource {
  id: string
  name: string
  provider: string
  category: string
  format: string
  url: string
  sizeEstimate: string
  gamesEstimate: string
  description: string
  supportsLimit: boolean
}

interface MegaImportProgress {
  phase: string
  sourceName: string
  bytesDownloaded: number
  totalBytes: number
  gamesScanned: number
  gamesImported: number
  percent: number
  message: string
}

const catalog = ref<MegaDatabaseSource[]>([])
const activeTab = ref<'all' | 'LumbrasGigaBase' | 'Lichess' | 'custom' | 'consolidate'>('all')
const selectedSourceId = ref<string>('lumbras-otb-2026')
const customUrl = ref<string>('')
const lichessLiveUrls = ref<string[]>([])
const lichessFeedType = ref<'broadcast' | 'standard'>('broadcast')
const loadingFeed = ref(false)

const maxGames = ref<number>(25000)
const minElo = ref<number>(0)
const createNewDb = ref<boolean>(false)
const newDbName = ref<string>('')

// Consolidated Database Builder state
const consolidateTargetName = ref<string>('Consolidated_Reference')
const consolidateIncludeLocal = ref<boolean>(true)
const consolidateSelectedOnlineIds = ref<string[]>([])
const consolidateSetAsRef = ref<boolean>(true)
const consolidateSetAsUser = ref<boolean>(false)

const isImporting = ref<boolean>(false)
const progress = ref<MegaImportProgress | null>(null)
const resultMsg = ref<string>('')
const errorMsg = ref<string>('')

const filteredCatalog = computed(() => {
  if (activeTab.value === 'all') return catalog.value
  if (activeTab.value === 'LumbrasGigaBase') {
    return catalog.value.filter(s => s.provider === 'LumbrasGigaBase')
  }
  if (activeTab.value === 'Lichess') {
    return catalog.value.filter(s => s.provider.startsWith('Lichess'))
  }
  return catalog.value
})

const selectedSource = computed(() => {
  return catalog.value.find(s => s.id === selectedSourceId.value) || null
})

onMounted(async () => {
  try {
    const list = await GetMegaDatabaseCatalog()
    if (Array.isArray(list) && list.length > 0) {
      catalog.value = list
      selectedSourceId.value = list[0].id
    }
  } catch (err) {
    console.error('Failed to load mega database catalog:', err)
  }

  if (!props.dbOpen) {
    createNewDb.value = true
    newDbName.value = 'lumbras_otb_2026'
  }

  try {
    EventsOn('megadb:progress', (data: MegaImportProgress) => {
      progress.value = data
    })
  } catch {
    // Ignore in test environment if runtime is mocked
  }
})

onUnmounted(() => {
  try {
    EventsOff('megadb:progress')
  } catch {
    // Ignore
  }
})

function selectSource(src: MegaDatabaseSource) {
  selectedSourceId.value = src.id
  resultMsg.value = ''
  errorMsg.value = ''
  if (src.provider === 'Lichess Open' && maxGames.value === 0) {
    maxGames.value = 25000
  }
  if (!props.dbOpen || createNewDb.value) {
    newDbName.value = src.id.replace(/-/g, '_')
  }
}

async function handleLoadLichessFeed(feed: 'broadcast' | 'standard') {
  lichessFeedType.value = feed
  loadingFeed.value = true
  try {
    const urls = await FetchLichessDatabaseFeed(feed, 12)
    lichessLiveUrls.value = Array.isArray(urls) ? urls : []
    if (lichessLiveUrls.value.length > 0 && !customUrl.value) {
      customUrl.value = lichessLiveUrls.value[0]
    }
  } catch (err) {
    console.error('Failed to fetch Lichess feed:', err)
  } finally {
    loadingFeed.value = false
  }
}

function toggleConsolidateOnlineSource(id: string) {
  const idx = consolidateSelectedOnlineIds.value.indexOf(id)
  if (idx >= 0) {
    consolidateSelectedOnlineIds.value.splice(idx, 1)
  } else {
    consolidateSelectedOnlineIds.value.push(id)
  }
}

async function handleStartImport() {
  isImporting.value = true
  resultMsg.value = ''
  errorMsg.value = ''

  if (activeTab.value === 'consolidate') {
    progress.value = {
      phase: 'consolidating',
      sourceName: consolidateTargetName.value || 'Consolidated_Reference',
      bytesDownloaded: 0,
      totalBytes: 0,
      gamesScanned: 0,
      gamesImported: 0,
      percent: 15,
      message: 'Consolidating databases & removing duplicates (by Game ID and move sequence)...',
    }
    try {
      let res: any
      if (consolidateSelectedOnlineIds.value.length > 0) {
        res = await ConsolidateOnlineCatalogDatabases(
          [...consolidateSelectedOnlineIds.value],
          consolidateIncludeLocal.value,
          consolidateTargetName.value.trim() || 'Consolidated_Reference',
          Number(maxGames.value) || 25000,
          Number(minElo.value) || 0,
          consolidateSetAsRef.value
        )
      } else {
        res = await BuildConsolidatedDatabase({
          targetName: consolidateTargetName.value.trim() || 'Consolidated_Reference',
          targetDescription: 'Consolidated Online & Master Reference Database (Deduplicated by ID & Move Sequence)',
          sourceBasePaths: [],
          setAsReference: consolidateSetAsRef.value,
          setAsDefaultUser: consolidateSetAsUser.value,
          minElo: Number(minElo.value) || 0,
        })
      }
      const saved = res?.uniqueGamesSaved ?? 0
      const removed = res?.duplicatesRemoved ?? 0
      const byId = res?.duplicatesById ?? 0
      const byMoves = res?.duplicatesByContent ?? 0
      resultMsg.value = `Consolidated "${res?.targetName || consolidateTargetName.value}": ${saved} unique games saved, ${removed} duplicates removed (${byId} by ID, ${byMoves} by identical moves)!`
      emit('imported', saved)
    } catch (err: any) {
      errorMsg.value = typeof err === 'string' ? err : (err?.message || 'Failed to consolidate databases')
    } finally {
      isImporting.value = false
    }
    return
  }

  progress.value = {
    phase: 'downloading',
    sourceName: activeTab.value === 'custom' ? customUrl.value : (selectedSource.value?.name || ''),
    bytesDownloaded: 0,
    totalBytes: 0,
    gamesScanned: 0,
    gamesImported: 0,
    percent: 5,
    message: 'Starting download & stream...',
  }

  try {
    const res = await ImportMegaDatabase({
      sourceId: activeTab.value === 'custom' ? 'custom' : selectedSourceId.value,
      customUrl: activeTab.value === 'custom' ? customUrl.value.trim() : '',
      maxGames: Number(maxGames.value) || 0,
      minElo: Number(minElo.value) || 0,
      createNewDatabase: createNewDb.value ? newDbName.value.trim() : '',
    })

    if (res?.databasePath) {
      resultMsg.value = `Extracted & opened native Scid database: ${res.databasePath}`
    } else {
      resultMsg.value = `Successfully imported ${res?.gamesImported ?? 0} games (${res?.gamesScanned ?? 0} scanned)!`
    }
    emit('imported', res?.gamesImported ?? 0)
  } catch (err: any) {
    errorMsg.value = typeof err === 'string' ? err : (err?.message || 'Failed to download/import database')
  } finally {
    isImporting.value = false
  }
}
</script>

<template>
  <div
    class="fixed inset-0 bg-black/85 backdrop-blur-sm flex items-center justify-center z-70 p-4 animate-fade-in select-none"
    @click.self="!isImporting && emit('close')"
  >
    <div class="bg-zinc-900 border border-zinc-700 rounded-2xl w-full max-w-3xl max-h-[90vh] shadow-2xl overflow-hidden flex flex-col">
      <!-- Header -->
      <div class="flex items-center justify-between px-5 py-3.5 bg-zinc-950 border-b border-zinc-800 shrink-0">
        <div class="flex items-center gap-2.5">
          <div class="w-8 h-8 rounded-lg bg-emerald-500/15 border border-emerald-500/30 flex items-center justify-center">
            <Database class="w-4 h-4 text-emerald-400" />
          </div>
          <div>
            <h2 class="text-sm font-bold text-zinc-100">One-Click Master &amp; Mega Database Downloader</h2>
            <p class="text-[11px] text-zinc-400">
              Lumbra's GigaBase (&sim;15M+ OTB/Online) &bull; Lichess Open &amp; Broadcasts (.pgn.zst) &bull; Lichess Elite (2400+) &bull; Consolidated Reference DB
            </p>
          </div>
        </div>
        <button
          data-testid="megadb-close-btn"
          :disabled="isImporting"
          class="text-zinc-400 hover:text-white p-1 rounded-lg hover:bg-zinc-800 transition-colors cursor-pointer disabled:opacity-40"
          @click="emit('close')"
        >
          <X class="w-5 h-5" />
        </button>
      </div>

      <!-- Provider Filter Tabs -->
      <div class="flex items-center gap-1.5 px-5 py-2.5 bg-zinc-950/60 border-b border-zinc-800 shrink-0 flex-wrap">
        <button
          type="button"
          data-testid="tab-all"
          class="px-3 py-1.5 rounded-lg text-xs font-semibold transition-colors cursor-pointer"
          :class="activeTab === 'all' ? 'bg-emerald-600 text-white' : 'bg-zinc-900 text-zinc-400 hover:text-zinc-200'"
          @click="activeTab = 'all'"
        >
          All Databases ({{ catalog.length }})
        </button>
        <button
          type="button"
          data-testid="tab-lumbras"
          class="px-3 py-1.5 rounded-lg text-xs font-semibold transition-colors cursor-pointer"
          :class="activeTab === 'LumbrasGigaBase' ? 'bg-emerald-600 text-white' : 'bg-zinc-900 text-zinc-400 hover:text-zinc-200'"
          @click="activeTab = 'LumbrasGigaBase'"
        >
          Lumbra's GigaBase
        </button>
        <button
          type="button"
          data-testid="tab-lichess"
          class="px-3 py-1.5 rounded-lg text-xs font-semibold transition-colors cursor-pointer"
          :class="activeTab === 'Lichess' ? 'bg-emerald-600 text-white' : 'bg-zinc-900 text-zinc-400 hover:text-zinc-200'"
          @click="activeTab = 'Lichess'"
        >
          Lichess Open / Elite / Broadcasts
        </button>
        <button
          type="button"
          data-testid="tab-custom"
          class="px-3 py-1.5 rounded-lg text-xs font-semibold transition-colors cursor-pointer"
          :class="activeTab === 'custom' ? 'bg-emerald-600 text-white' : 'bg-zinc-900 text-zinc-400 hover:text-zinc-200'"
          @click="activeTab = 'custom'; handleLoadLichessFeed('broadcast')"
        >
          Live Monthly Feed / Custom URL
        </button>
        <button
          type="button"
          data-testid="tab-consolidate"
          class="px-3 py-1.5 rounded-lg text-xs font-bold transition-colors cursor-pointer flex items-center gap-1"
          :class="activeTab === 'consolidate' ? 'bg-cyan-600 text-white' : 'bg-cyan-950/50 text-cyan-300 border border-cyan-500/40 hover:bg-cyan-900/60'"
          @click="activeTab = 'consolidate'"
        >
          <Layers class="w-3.5 h-3.5" />
          <span>⚡ Consolidate All Online DBs</span>
        </button>
      </div>

      <!-- Body -->
      <div class="p-5 space-y-4 overflow-y-auto flex-1 text-xs text-zinc-300">
        <!-- Curated Catalog Grid -->
        <div v-if="activeTab !== 'custom' && activeTab !== 'consolidate'" class="grid grid-cols-1 md:grid-cols-2 gap-2.5">
          <button
            v-for="src in filteredCatalog"
            :key="src.id"
            type="button"
            :data-testid="`source-${src.id}`"
            class="text-left p-3 rounded-xl border transition-all cursor-pointer flex flex-col justify-between gap-2"
            :class="selectedSourceId === src.id
              ? 'bg-emerald-950/40 border-emerald-500 shadow-md'
              : 'bg-zinc-950 border-zinc-800 hover:border-zinc-700'"
            @click="selectSource(src)"
          >
            <div class="space-y-1">
              <div class="flex items-center justify-between gap-2">
                <span class="font-bold text-zinc-100 text-xs">{{ src.name }}</span>
                <span
                  class="px-1.5 py-0.5 rounded text-[10px] font-mono uppercase shrink-0"
                  :class="src.provider === 'LumbrasGigaBase'
                    ? 'bg-amber-500/15 text-amber-300 border border-amber-500/30'
                    : 'bg-sky-500/15 text-sky-300 border border-sky-500/30'"
                >
                  {{ src.provider }}
                </span>
              </div>
              <p class="text-[11px] text-zinc-400 leading-snug">{{ src.description }}</p>
            </div>
            <div class="flex items-center justify-between text-[10px] text-zinc-400 pt-1 border-t border-zinc-800/80">
              <span class="font-semibold text-emerald-400">{{ src.gamesEstimate }}</span>
              <span class="font-mono">{{ src.sizeEstimate }}</span>
            </div>
          </button>
        </div>

        <!-- Consolidated Online & Local Reference Database Builder Tab -->
        <div v-else-if="activeTab === 'consolidate'" class="bg-zinc-950 border border-cyan-500/40 rounded-xl p-4 space-y-3">
          <div class="flex items-center justify-between">
            <div>
              <h3 class="font-bold text-sm text-cyan-300 flex items-center gap-1.5">
                <Layers class="w-4 h-4 text-cyan-400" />
                <span>Consolidated Online &amp; Reference Database Builder</span>
              </h3>
              <p class="text-[11px] text-zinc-400 mt-0.5">
                Combines all online and local databases into a single unified <code class="text-cyan-300">.si4</code> database, removing all duplicate games whether they share the same <strong>Online Game ID / URL</strong> or have <strong>identical move sequences</strong>.
              </p>
            </div>
          </div>

          <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
            <div class="space-y-1">
              <label class="block text-[11px] text-zinc-400">Consolidated Database Name</label>
              <input
                v-model="consolidateTargetName"
                data-testid="consolidate-target-name"
                type="text"
                placeholder="Consolidated_Reference"
                class="w-full bg-zinc-900 border border-zinc-700 rounded-lg px-3 py-1.5 text-xs text-zinc-100 font-mono"
              />
            </div>

            <div class="space-y-1.5 pt-1">
              <label class="flex items-center gap-2 text-xs text-cyan-300 font-semibold cursor-pointer">
                <input
                  v-model="consolidateSetAsRef"
                  data-testid="consolidate-set-ref"
                  type="checkbox"
                  class="rounded accent-cyan-500"
                />
                <span>Set as Default REFERENCE Database (Opening Tree / Theory)</span>
              </label>
              <label class="flex items-center gap-2 text-xs text-amber-300 cursor-pointer">
                <input
                  v-model="consolidateSetAsUser"
                  data-testid="consolidate-set-user"
                  type="checkbox"
                  class="rounded accent-amber-500"
                />
                <span>Set as Default USER Database</span>
              </label>
              <label class="flex items-center gap-2 text-xs text-zinc-300 cursor-pointer">
                <input
                  v-model="consolidateIncludeLocal"
                  type="checkbox"
                  class="rounded accent-emerald-500"
                />
                <span>Include all installed local &amp; Persona databases</span>
              </label>
            </div>
          </div>

          <div v-if="catalog.length > 0" class="space-y-1.5 pt-1">
            <div class="text-[11px] text-zinc-400">
              Optional: Also stream &amp; merge online catalog feeds into the Consolidated Database:
            </div>
            <div class="grid grid-cols-1 md:grid-cols-2 gap-1.5">
              <label
                v-for="src in catalog"
                :key="'cons-' + src.id"
                class="flex items-center gap-2 p-2 rounded-lg bg-zinc-900 border border-zinc-800 hover:border-cyan-500/40 cursor-pointer text-[11px]"
              >
                <input
                  type="checkbox"
                  :checked="consolidateSelectedOnlineIds.includes(src.id)"
                  class="rounded accent-cyan-500"
                  @change="toggleConsolidateOnlineSource(src.id)"
                />
                <span class="font-semibold text-zinc-200 truncate">{{ src.name }}</span>
                <span class="ml-auto text-[10px] font-mono text-emerald-400">{{ src.gamesEstimate }}</span>
              </label>
            </div>
          </div>
        </div>

        <!-- Custom URL / Live Lichess Monthly Feed Tab -->
        <div v-else class="bg-zinc-950 border border-zinc-800 rounded-xl p-4 space-y-3">
          <div class="flex items-center justify-between">
            <span class="font-bold text-zinc-200 flex items-center gap-1.5">
              <Globe class="w-4 h-4 text-sky-400" />
              <span>Direct Archive URL (.pgn.zst, .zip, .7z, or .pgn)</span>
            </span>
            <div class="flex items-center gap-1.5">
              <button
                type="button"
                class="px-2.5 py-1 rounded bg-zinc-800 hover:bg-zinc-700 text-[11px] text-zinc-200 cursor-pointer"
                @click="handleLoadLichessFeed('broadcast')"
              >
                Load Lichess Broadcast Months
              </button>
              <button
                type="button"
                class="px-2.5 py-1 rounded bg-zinc-800 hover:bg-zinc-700 text-[11px] text-zinc-200 cursor-pointer"
                @click="handleLoadLichessFeed('standard')"
              >
                Load Lichess Rated Months
              </button>
            </div>
          </div>

          <input
            v-model="customUrl"
            data-testid="custom-mega-url"
            type="text"
            placeholder="https://database.lichess.org/broadcast/lichess_db_broadcast_2026-09.pgn.zst"
            class="w-full bg-zinc-900 border border-zinc-700 rounded-lg px-3 py-2 text-xs text-zinc-100 font-mono focus:outline-hidden focus:border-emerald-500"
          />

          <div v-if="lichessLiveUrls.length > 0" class="space-y-1.5">
            <div class="text-[11px] text-zinc-400">
              Click any live monthly archive from <code class="text-sky-400">database.lichess.org/{{ lichessFeedType }}/list.txt</code>:
            </div>
            <div class="max-h-32 overflow-y-auto grid grid-cols-1 md:grid-cols-2 gap-1.5">
              <button
                v-for="u in lichessLiveUrls"
                :key="u"
                type="button"
                class="text-left px-2.5 py-1.5 rounded bg-zinc-900 hover:bg-zinc-800 border border-zinc-800 text-[11px] font-mono text-zinc-300 truncate cursor-pointer"
                @click="customUrl = u"
              >
                {{ u.split('/').pop() }}
              </button>
            </div>
          </div>
        </div>

        <!-- Stream & Filter Settings -->
        <div class="bg-zinc-950 border border-zinc-800 rounded-xl p-4 space-y-3">
          <div class="flex items-center gap-1.5 text-zinc-200 font-bold">
            <Filter class="w-3.5 h-3.5 text-emerald-400" />
            <span>Import &amp; Smart Stream Filter Options</span>
          </div>

          <div class="grid grid-cols-1 md:grid-cols-3 gap-3">
            <!-- Max Games Cap -->
            <div class="space-y-1">
              <label class="block text-[11px] text-zinc-400">Max Games to Import (0 = All)</label>
              <select
                v-model.number="maxGames"
                data-testid="max-games-select"
                class="w-full bg-zinc-900 border border-zinc-700 rounded-lg px-2.5 py-1.5 text-xs text-zinc-100"
              >
                <option :value="5000">5,000 games (Fastest preview)</option>
                <option :value="25000">25,000 games (Recommended)</option>
                <option :value="100000">100,000 games</option>
                <option :value="500000">500,000 games</option>
                <option :value="0">All games in archive (Unlimited)</option>
              </select>
            </div>

            <!-- Minimum Elo Filter -->
            <div class="space-y-1">
              <label class="block text-[11px] text-zinc-400">Minimum Player Rating (Elo)</label>
              <select
                v-model.number="minElo"
                data-testid="min-elo-select"
                class="w-full bg-zinc-900 border border-zinc-700 rounded-lg px-2.5 py-1.5 text-xs text-zinc-100"
              >
                <option :value="0">Any Rating (All games)</option>
                <option :value="2000">2000+ Elo (Candidate Master+)</option>
                <option :value="2200">2200+ Elo (Master+)</option>
                <option :value="2400">2400+ Elo (IM / GM Elite)</option>
                <option :value="2600">2600+ Elo (Super-GM)</option>
              </select>
            </div>

            <!-- Target Database -->
            <div class="space-y-1">
              <label class="block text-[11px] text-zinc-400 flex items-center gap-1.5">
                <input
                  v-model="createNewDb"
                  type="checkbox"
                  class="rounded border-zinc-700 bg-zinc-900 text-emerald-500"
                />
                <span>Create New .si4 Database</span>
              </label>
              <input
                v-if="createNewDb"
                v-model="newDbName"
                data-testid="new-db-name-input"
                type="text"
                placeholder="e.g. lumbras_otb_2026"
                class="w-full bg-zinc-900 border border-zinc-700 rounded-lg px-2.5 py-1.5 text-xs text-zinc-100"
              />
              <div v-else class="px-2.5 py-1.5 rounded-lg bg-zinc-900 border border-zinc-800 text-zinc-400 truncate">
                Append to: <span class="text-zinc-200 font-semibold">{{ dbName || 'Active Database' }}</span>
              </div>
            </div>
          </div>
        </div>

        <!-- Live Progress Bar -->
        <div v-if="isImporting || progress" class="bg-zinc-950 border border-zinc-800 rounded-xl p-3.5 space-y-2">
          <div class="flex items-center justify-between text-xs">
            <span class="font-semibold text-emerald-400">{{ progress?.message || 'Processing...' }}</span>
            <span class="font-mono text-zinc-400">
              {{ progress?.gamesImported ?? 0 }} imported / {{ progress?.gamesScanned ?? 0 }} scanned
            </span>
          </div>
          <div class="w-full h-2 bg-zinc-800 rounded-full overflow-hidden">
            <div
              class="h-full bg-emerald-500 transition-all duration-300"
              :style="{ width: `${Math.min(100, Math.max(5, progress?.percent || 10))}%` }"
            />
          </div>
        </div>

        <!-- Success / Error Banners -->
        <div
          v-if="resultMsg"
          data-testid="megadb-success-msg"
          class="p-3 rounded-xl bg-emerald-950/70 border border-emerald-500/40 flex items-center gap-2.5 text-emerald-200"
        >
          <CheckCircle class="w-4 h-4 text-emerald-400 shrink-0" />
          <span class="font-semibold">{{ resultMsg }}</span>
        </div>

        <div
          v-if="errorMsg"
          data-testid="megadb-error-msg"
          class="p-3 rounded-xl bg-rose-950/70 border border-rose-500/40 flex items-center gap-2.5 text-rose-200"
        >
          <AlertCircle class="w-4 h-4 text-rose-400 shrink-0" />
          <span>{{ errorMsg }}</span>
        </div>
      </div>

      <!-- Footer -->
      <div class="flex items-center justify-between px-5 py-3.5 bg-zinc-950 border-t border-zinc-800 shrink-0">
        <div class="flex items-center gap-2 text-[11px] text-zinc-400">
          <HardDrive class="w-3.5 h-3.5 text-emerald-400" />
          <span>Supports native <code class="text-zinc-300">.pgn.zst</code> streaming, <code class="text-zinc-300">.7z</code>, <code class="text-zinc-300">.zip</code> &amp; <code class="text-zinc-300">.si4</code></span>
        </div>
        <div class="flex items-center gap-2">
          <button
            type="button"
            :disabled="isImporting"
            class="px-4 py-2 rounded-xl bg-zinc-800 hover:bg-zinc-700 text-zinc-300 text-xs font-medium cursor-pointer disabled:opacity-40"
            @click="emit('close')"
          >
            Close
          </button>
          <button
            type="button"
            data-testid="start-mega-import-btn"
            :disabled="isImporting"
            class="px-5 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-500 text-white text-xs font-bold flex items-center gap-2 shadow-lg cursor-pointer disabled:opacity-50"
            @click="handleStartImport"
          >
            <Layers v-if="activeTab === 'consolidate'" class="w-4 h-4" />
            <Download v-else class="w-4 h-4" />
            <span>{{ isImporting ? (activeTab === 'consolidate' ? 'Consolidating Databases...' : 'Downloading & Importing...') : (activeTab === 'consolidate' ? 'Build Consolidated Database' : 'Download & Import Now') }}</span>
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
