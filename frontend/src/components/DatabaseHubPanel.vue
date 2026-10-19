<script setup lang="ts">
import { ref, computed } from 'vue'
import {
  Database,
  Star,
  BookOpen,
  ClipboardList,
  FolderOpen,
  PlusCircle,
  Download,
  Globe,
  Wrench,
  ArrowRight,
  Check,
  RefreshCw,
  Layers
} from '@lucide/vue'
import { BuildConsolidatedDatabase } from '../../wailsjs/go/main/App'
import type { DiscoveredDatabase } from './HeaderBar.vue'

export interface DatabaseRolesStatusView {
  activeDb?: { basePath: string; name: string; numGames: number; description: string } | null
  defaultDbPath: string
  defaultDbName: string
  isActiveDefault: boolean
  referenceDb?: { basePath: string; name: string; numGames: number; description: string } | null
  referenceDbPath: string
  referenceDbName: string
  useRefForTree: boolean
  clipbasePath: string
  isClipbase: boolean
}

const props = defineProps<{
  rolesStatus: DatabaseRolesStatusView | null
  discoveredDatabases: DiscoveredDatabase[]
  isUpdatingFromLocal?: boolean
  localMergeStatus?: string | null
}>()

const emit = defineEmits<{
  (e: 'openDiscoveredBase', path: string): void
  (e: 'openDB'): void
  (e: 'createDB'): void
  (e: 'setDefaultDB', path: string): void
  (e: 'openDefaultDB'): void
  (e: 'setReferenceDB', path: string, useForTree: boolean): void
  (e: 'openClipbase'): void
  (e: 'openMegaDownloader'): void
  (e: 'openOnlineSync'): void
  (e: 'openMaintenance'): void
  (e: 'updateFromLocal', targetPath?: string): void
  (e: 'cycleWritePolicy', path: string, currentPolicy?: string): void
  (e: 'openWorkstationManager', tab?: 'builder' | 'storage' | 'reset'): void
}>()

const isConsolidating = ref(false)
const consolidateBanner = ref<string | null>(null)

async function handleConsolidateAllToReference() {
  if (isConsolidating.value) return
  isConsolidating.value = true
  consolidateBanner.value = 'Consolidating all databases & removing duplicate games (by ID and move sequence)...'
  try {
    const res = await BuildConsolidatedDatabase({
      targetName: 'Consolidated_Reference',
      targetDescription: 'Consolidated Online & Master Reference Database (Deduplicated by ID & Move Sequence)',
      sourceBasePaths: [],
      setAsReference: true,
      setAsDefaultUser: false,
      minElo: 0,
    })
    if (res) {
      consolidateBanner.value = `Consolidated "${res.targetName}": ${res.uniqueGamesSaved} unique games saved, ${res.duplicatesRemoved} duplicates removed (${res.duplicatesById} by ID, ${res.duplicatesByContent} by identical moves) across ${res.sourcesScanned} databases — now set as Default Reference DB!`
      if (res.targetBasePath) {
        emit('setReferenceDB', res.targetBasePath, true)
      }
    }
  } catch (err: any) {
    consolidateBanner.value = `Consolidation note: ${err?.message || String(err)}`
  } finally {
    isConsolidating.value = false
  }
}

function normalizePath(p?: string): string {
  if (!p) return ''
  return p.replace(/\.(si4|sg4|sn4|pgn)$/i, '').trim()
}

function isDefaultPath(path: string): boolean {
  if (!props.rolesStatus?.defaultDbPath) return false
  return normalizePath(path) === normalizePath(props.rolesStatus.defaultDbPath)
}

function isActivePath(path: string): boolean {
  if (!props.rolesStatus?.activeDb?.basePath) return false
  return normalizePath(path) === normalizePath(props.rolesStatus.activeDb.basePath)
}

function isReferencePath(path: string): boolean {
  if (!props.rolesStatus?.referenceDbPath) return false
  return normalizePath(path) === normalizePath(props.rolesStatus.referenceDbPath)
}

const groupedDbs = computed(() => {
  const groups: Record<string, DiscoveredDatabase[]> = {}
  for (const db of props.discoveredDatabases || []) {
    const cat = db.category || 'Other Databases'
    if (!groups[cat]) groups[cat] = []
    groups[cat].push(db)
  }
  return groups
})

function formatCount(n: number): string {
  if (!n || n <= 0) return '0 games'
  if (n >= 1000000) return `${(n / 1000000).toFixed(2)}M games`
  if (n >= 1000) return `${(n / 1000).toFixed(1)}k games`
  return `${n} games`
}
</script>

<template>
  <div class="database-hub-panel h-full overflow-y-auto p-2 space-y-3 text-xs">
    <!-- How Databases Relate Visual Diagram Banner -->
    <div class="bg-zinc-950 border border-zinc-800 rounded-xl p-3.5 space-y-3">
      <div class="flex flex-wrap items-center justify-between gap-2">
        <div>
          <h3 class="text-sm font-extrabold text-zinc-100 flex items-center gap-2">
            <Database class="w-4 h-4 text-emerald-400" />
            <span>How Databases Work Together in NewScid</span>
          </h3>
          <p class="text-[11px] text-zinc-400 mt-0.5">
            Select any database as your <strong>Default USER Database</strong> or your <strong>Default REFERENCE Database</strong> (including a deduplicated Consolidated Database), while each <strong>Persona</strong> maintains its own dedicated database.
          </p>
        </div>

        <div class="flex items-center gap-1.5 flex-wrap">
          <button
            data-testid="consolidate-all-dbs-btn"
            class="px-2.5 py-1.5 rounded-lg bg-cyan-600/20 hover:bg-cyan-600/30 text-cyan-300 border border-cyan-500/40 font-bold flex items-center gap-1.5 cursor-pointer transition-colors"
            :disabled="isConsolidating"
            title="Combine all online & local databases into a single deduplicated Consolidated Reference Database (removing duplicates by Game ID and move sequence)"
            @click="handleConsolidateAllToReference"
          >
            <Layers class="w-3.5 h-3.5 text-cyan-400" :class="{ 'animate-spin': isConsolidating }" />
            <span>{{ isConsolidating ? 'Consolidating...' : '⚡ Consolidate All DBs' }}</span>
          </button>
          <button
            class="px-2.5 py-1.5 rounded-lg bg-indigo-600/20 hover:bg-indigo-600/30 text-indigo-300 border border-indigo-500/40 font-bold flex items-center gap-1.5 cursor-pointer transition-colors"
            title="Create a smaller focused database from multiple source databases (by opening line or player)"
            @click="emit('openWorkstationManager', 'builder')"
          >
            <span>⚡ Build Sub-DB</span>
          </button>
          <button
            class="px-2.5 py-1.5 rounded-lg bg-zinc-900 hover:bg-zinc-800 text-amber-300 border border-amber-500/30 font-semibold flex items-center gap-1.5 cursor-pointer transition-colors"
            title="View where NewScid stores databases, engines, opening books & endgame tablebases, or copy files into NewScid"
            @click="emit('openWorkstationManager', 'storage')"
          >
            <span>📁 Storage &amp; Assets</span>
          </button>
          <button
            class="px-2.5 py-1.5 rounded-lg bg-rose-950/50 hover:bg-rose-900/60 text-rose-300 border border-rose-500/40 font-semibold flex items-center gap-1.5 cursor-pointer transition-colors"
            title="Factory Reset all databases and settings to empty scratch state (with opt-in backup)"
            @click="emit('openWorkstationManager', 'reset')"
          >
            <span>🧹 Factory Reset</span>
          </button>
          <button
            class="px-2.5 py-1.5 rounded-lg bg-emerald-600/20 hover:bg-emerald-600/30 text-emerald-300 border border-emerald-500/40 font-bold flex items-center gap-1.5 cursor-pointer transition-colors"
            :disabled="isUpdatingFromLocal"
            title="Update the active database with all non-duplicate games from local databases already on your machine"
            @click="emit('updateFromLocal', rolesStatus?.activeDb?.basePath || '')"
          >
            <RefreshCw class="w-3.5 h-3.5 text-emerald-400" :class="{ 'animate-spin': isUpdatingFromLocal }" />
            <span>{{ isUpdatingFromLocal ? 'Updating from Local DBs...' : 'Update Active DB from Local' }}</span>
          </button>
          <button
            class="px-2.5 py-1.5 rounded-lg bg-zinc-900 hover:bg-zinc-800 text-zinc-200 border border-zinc-700 font-semibold flex items-center gap-1.5 cursor-pointer transition-colors"
            @click="emit('openDB')"
          >
            <FolderOpen class="w-3.5 h-3.5 text-emerald-400" />
            <span>Open File...</span>
          </button>
          <button
            class="px-2.5 py-1.5 rounded-lg bg-zinc-900 hover:bg-zinc-800 text-zinc-200 border border-zinc-700 font-semibold flex items-center gap-1.5 cursor-pointer transition-colors"
            @click="emit('createDB')"
          >
            <PlusCircle class="w-3.5 h-3.5 text-blue-400" />
            <span>New DB...</span>
          </button>
        </div>
      </div>

      <div
        v-if="consolidateBanner"
        data-testid="consolidate-status-banner"
        class="p-2.5 rounded-lg bg-cyan-950/40 border border-cyan-500/40 text-cyan-200 text-xs flex items-center justify-between gap-2"
      >
        <div class="flex items-center gap-2">
          <Layers class="w-3.5 h-3.5 text-cyan-400 shrink-0" :class="{ 'animate-spin': isConsolidating }" />
          <span>{{ consolidateBanner }}</span>
        </div>
        <button
          class="text-[10px] text-cyan-300 hover:text-white underline cursor-pointer"
          @click="consolidateBanner = null"
        >
          Dismiss
        </button>
      </div>

      <div
        v-if="localMergeStatus"
        class="p-2.5 rounded-lg bg-emerald-950/40 border border-emerald-500/40 text-emerald-200 text-xs flex items-center gap-2"
      >
        <RefreshCw class="w-3.5 h-3.5 text-emerald-400 shrink-0" :class="{ 'animate-spin': isUpdatingFromLocal }" />
        <span>{{ localMergeStatus }}</span>
      </div>

      <!-- 4 Database Role Cards -->
      <div class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-4 gap-2.5">
        <!-- 1. DEFAULT USER DATABASE -->
        <div class="p-3 rounded-xl bg-zinc-900/90 border border-amber-500/40 flex flex-col justify-between gap-2">
          <div>
            <div class="flex items-center justify-between">
              <span class="text-[10px] font-bold uppercase tracking-wider text-amber-400 flex items-center gap-1">
                <Star class="w-3.5 h-3.5 fill-amber-400 text-amber-400" />
                1. Default Startup DB (User)
              </span>
              <span
                v-if="rolesStatus?.isActiveDefault"
                class="px-1.5 py-0.5 rounded bg-amber-500/20 text-amber-300 text-[9px] font-mono"
              >
                Loaded
              </span>
            </div>
            <div class="font-bold text-sm text-zinc-100 mt-1 truncate">
              {{ rolesStatus?.defaultDbName || 'Not Configured Yet' }}
            </div>
            <p class="text-[11px] text-zinc-400 mt-1 leading-snug">
              Your personal startup database (editable by default) as opposed to the Reference DB.
            </p>
            <select
              v-if="discoveredDatabases.length > 0"
              data-testid="select-default-user-db"
              :value="normalizePath(rolesStatus?.defaultDbPath)"
              class="mt-1.5 w-full bg-zinc-950 border border-amber-500/30 rounded px-2 py-1 text-[11px] text-zinc-200"
              @change="(e) => emit('setDefaultDB', (e.target as HTMLSelectElement).value)"
            >
              <option value="">-- Select Any DB as Default USER DB --</option>
              <option
                v-for="db in discoveredDatabases"
                :key="'def-' + db.basePath"
                :value="normalizePath(db.basePath)"
              >
                [{{ db.category }}] {{ db.name }} ({{ formatCount(db.numGames) }})
              </option>
            </select>
          </div>

          <div class="flex items-center gap-1.5 pt-1 flex-wrap">
            <button
              v-if="rolesStatus?.defaultDbPath && !rolesStatus?.isActiveDefault"
              class="flex-1 px-2 py-1 rounded bg-amber-600 hover:bg-amber-500 text-white font-semibold text-[11px] cursor-pointer transition-colors"
              @click="emit('openDefaultDB')"
            >
              Load Default DB
            </button>
            <button
              v-if="rolesStatus?.activeDb?.basePath && !rolesStatus?.isActiveDefault"
              class="flex-1 px-2 py-1 rounded bg-zinc-800 hover:bg-zinc-700 text-amber-300 border border-amber-500/30 font-semibold text-[11px] cursor-pointer transition-colors"
              @click="emit('setDefaultDB', rolesStatus.activeDb.basePath)"
            >
              ⭐ Set Current as Default
            </button>
            <button
              v-if="rolesStatus?.defaultDbPath"
              class="px-2 py-1 rounded bg-zinc-800 hover:bg-zinc-700 text-emerald-300 border border-emerald-500/30 font-semibold text-[11px] flex items-center gap-1 cursor-pointer transition-colors"
              :disabled="isUpdatingFromLocal"
              title="Update Default DB with non-duplicate games from local databases"
              @click="emit('updateFromLocal', rolesStatus.defaultDbPath)"
            >
              <RefreshCw class="w-3 h-3" :class="{ 'animate-spin': isUpdatingFromLocal }" />
              <span>Update Local</span>
            </button>
            <span
              v-if="rolesStatus?.isActiveDefault"
              class="text-[11px] text-amber-300 font-semibold flex items-center gap-1"
            >
              <Check class="w-3.5 h-3.5" /> Active DB is your Default DB
            </span>
          </div>
        </div>

        <!-- 2. ACTIVE WORKING DATABASE -->
        <div class="p-3 rounded-xl bg-zinc-900/90 border border-emerald-500/40 flex flex-col justify-between gap-2">
          <div>
            <div class="flex items-center justify-between">
              <span class="text-[10px] font-bold uppercase tracking-wider text-emerald-400 flex items-center gap-1">
                <Database class="w-3.5 h-3.5 text-emerald-400" />
                2. Active Working DB
              </span>
              <span v-if="rolesStatus?.activeDb" class="text-[10px] font-mono text-emerald-300">
                {{ formatCount(rolesStatus.activeDb.numGames) }}
              </span>
            </div>
            <div class="font-bold text-sm text-zinc-100 mt-1 truncate">
              {{ rolesStatus?.activeDb?.name || 'No Database Open' }}
            </div>
            <p class="text-[11px] text-zinc-400 mt-1 leading-snug">
              Drives the <strong>Games Browser</strong>, <strong>Search (Ctrl+H)</strong>, and <strong>Save Game (Ctrl+S)</strong>.
            </p>
          </div>

          <div class="flex items-center gap-1.5 pt-1 flex-wrap">
            <button
              v-if="rolesStatus?.activeDb"
              class="px-2 py-1 rounded bg-zinc-800 hover:bg-zinc-700 text-zinc-200 border border-zinc-700 text-[11px] flex items-center gap-1 cursor-pointer"
              @click="emit('openMaintenance')"
            >
              <Wrench class="w-3 h-3 text-emerald-400" />
              <span>Maintenance</span>
            </button>
            <button
              class="px-2 py-1 rounded bg-zinc-800 hover:bg-zinc-700 text-zinc-200 border border-zinc-700 text-[11px] flex items-center gap-1 cursor-pointer"
              @click="emit('openOnlineSync')"
            >
              <Globe class="w-3 h-3 text-cyan-400" />
              <span>Sync Online</span>
            </button>
            <button
              v-if="rolesStatus?.activeDb?.basePath"
              class="px-2 py-1 rounded bg-emerald-950/60 hover:bg-emerald-900/70 text-emerald-300 border border-emerald-500/40 text-[11px] font-semibold flex items-center gap-1 cursor-pointer"
              :disabled="isUpdatingFromLocal"
              title="Merge all non-duplicate local games into Active DB"
              @click="emit('updateFromLocal', rolesStatus.activeDb.basePath)"
            >
              <RefreshCw class="w-3 h-3" :class="{ 'animate-spin': isUpdatingFromLocal }" />
              <span>Update Local</span>
            </button>
          </div>
        </div>

        <!-- 3. REFERENCE DATABASE (MEGA / CONSOLIDATED / MASTER BASE) -->
        <div class="p-3 rounded-xl bg-zinc-900/90 border border-cyan-500/40 flex flex-col justify-between gap-2">
          <div>
            <div class="flex items-center justify-between">
              <span class="text-[10px] font-bold uppercase tracking-wider text-cyan-400 flex items-center gap-1">
                <BookOpen class="w-3.5 h-3.5 text-cyan-400" />
                3. Reference Theory DB (Default Ref)
              </span>
              <span v-if="rolesStatus?.referenceDb" class="text-[10px] font-mono text-cyan-300">
                {{ formatCount(rolesStatus.referenceDb.numGames) }}
              </span>
            </div>
            <div class="font-bold text-sm text-zinc-100 mt-1 truncate">
              {{ rolesStatus?.referenceDbName || 'Uses Active DB / Polyglot' }}
            </div>
            <p class="text-[11px] text-zinc-400 mt-1 leading-snug">
              Select the <strong>Consolidated DB</strong> or any online/master database as your Default Reference DB for Opening Tree (F3).
            </p>
            <select
              v-if="discoveredDatabases.length > 0"
              data-testid="select-default-ref-db"
              :value="normalizePath(rolesStatus?.referenceDbPath)"
              class="mt-1.5 w-full bg-zinc-950 border border-cyan-500/30 rounded px-2 py-1 text-[11px] text-zinc-200"
              @change="(e) => emit('setReferenceDB', (e.target as HTMLSelectElement).value, true)"
            >
              <option value="">-- Select Any DB as Default REFERENCE DB --</option>
              <option
                v-for="db in discoveredDatabases"
                :key="'ref-' + db.basePath"
                :value="normalizePath(db.basePath)"
              >
                [{{ db.category }}] {{ db.name }} ({{ formatCount(db.numGames) }})
              </option>
            </select>
          </div>

          <div class="flex items-center justify-between gap-1.5 pt-1 flex-wrap">
            <label
              v-if="rolesStatus?.referenceDbPath"
              class="flex items-center gap-1.5 text-[11px] text-cyan-300 cursor-pointer select-none"
            >
              <input
                type="checkbox"
                :checked="rolesStatus.useRefForTree"
                class="rounded accent-cyan-500"
                @change="emit('setReferenceDB', rolesStatus.referenceDbPath, !rolesStatus.useRefForTree)"
              />
              <span>Use for Tree</span>
            </label>
            <button
              v-if="rolesStatus?.referenceDbPath"
              class="px-2 py-1 rounded bg-cyan-950/60 hover:bg-cyan-900/70 text-cyan-300 border border-cyan-500/40 font-semibold text-[11px] flex items-center gap-1 cursor-pointer transition-colors"
              :disabled="isUpdatingFromLocal"
              title="Update Reference / Mega Database with all non-duplicate games from local databases"
              @click="emit('updateFromLocal', rolesStatus.referenceDbPath)"
            >
              <RefreshCw class="w-3 h-3" :class="{ 'animate-spin': isUpdatingFromLocal }" />
              <span>Update Local</span>
            </button>
            <button
              v-if="rolesStatus?.activeDb?.basePath && !isReferencePath(rolesStatus.activeDb.basePath)"
              class="px-2 py-1 rounded bg-zinc-800 hover:bg-zinc-700 text-cyan-300 border border-cyan-500/30 font-semibold text-[11px] cursor-pointer transition-colors"
              @click="emit('setReferenceDB', rolesStatus.activeDb.basePath, true)"
            >
              📚 Use Active as Ref
            </button>
            <button
              v-else
              class="px-2 py-1 rounded bg-zinc-800 hover:bg-zinc-700 text-emerald-300 border border-emerald-500/30 font-semibold text-[11px] cursor-pointer transition-colors"
              @click="emit('openMegaDownloader')"
            >
              Download Mega DB
            </button>
          </div>
        </div>

        <!-- 4. CLIPBASE (SCRATCHPAD) -->
        <div class="p-3 rounded-xl bg-zinc-900/90 border border-purple-500/40 flex flex-col justify-between gap-2">
          <div>
            <div class="flex items-center justify-between">
              <span class="text-[10px] font-bold uppercase tracking-wider text-purple-400 flex items-center gap-1">
                <ClipboardList class="w-3.5 h-3.5 text-purple-400" />
                4. Clipbase Scratchpad
              </span>
              <span
                v-if="rolesStatus?.isClipbase"
                class="px-1.5 py-0.5 rounded bg-purple-500/20 text-purple-300 text-[9px] font-mono"
              >
                Active
              </span>
            </div>
            <div class="font-bold text-sm text-zinc-100 mt-1 truncate">
              Clipbase (Scratchpad)
            </div>
            <p class="text-[11px] text-zinc-400 mt-1 leading-snug">
              Temporary workspace for quick PGN paste, testing variations, or fair-play downloads without touching your Default DB.
            </p>
          </div>

          <div class="flex items-center gap-1.5 pt-1">
            <button
              class="w-full px-2 py-1 rounded bg-purple-600/80 hover:bg-purple-500 text-white font-semibold text-[11px] cursor-pointer transition-colors"
              @click="emit('openClipbase')"
            >
              {{ rolesStatus?.isClipbase ? 'Clipbase Currently Open' : 'Switch to Clipbase' }}
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- All Discovered Databases Table with Role Assignment Buttons -->
    <div class="bg-zinc-950 border border-zinc-800 rounded-xl overflow-hidden">
      <div class="px-3.5 py-2.5 bg-zinc-900/70 border-b border-zinc-800 flex items-center justify-between">
        <span class="font-bold text-zinc-100">Installed &amp; Discovered Databases — Assign Roles</span>
        <button
          class="px-2.5 py-1 rounded-lg bg-emerald-600/20 hover:bg-emerald-600/30 text-emerald-300 border border-emerald-500/30 font-semibold text-[11px] flex items-center gap-1 cursor-pointer"
          @click="emit('openMegaDownloader')"
        >
          <Download class="w-3 h-3" />
          <span>Download Lumbra's GigaBase / Lichess Mega DB</span>
        </button>
      </div>

      <div class="divide-y divide-zinc-800/80">
        <div v-for="(dbs, category) in groupedDbs" :key="category" class="p-2.5 space-y-1.5">
          <div class="text-[10px] font-bold uppercase tracking-wider text-zinc-400 px-1">
            {{ category }} ({{ dbs.length }})
          </div>

          <div
            v-for="db in dbs"
            :key="db.basePath"
            class="p-2.5 rounded-lg bg-zinc-900/70 hover:bg-zinc-900 border transition-colors flex flex-wrap items-center justify-between gap-2"
            :class="isActivePath(db.basePath) ? 'border-emerald-500/60 bg-emerald-950/15' : 'border-zinc-800/80'"
          >
            <div class="min-w-0 flex-1">
              <div class="flex items-center gap-2 flex-wrap">
                <span class="font-bold text-zinc-100 text-xs">{{ db.name }}</span>
                <span class="text-[10px] font-mono text-emerald-400 font-semibold">
                  {{ formatCount(db.numGames) }}
                </span>
                <span v-if="db.sizeStr" class="text-[10px] font-mono text-zinc-500">({{ db.sizeStr }})</span>

                <!-- Role Badges -->
                <span
                  v-if="isDefaultPath(db.basePath)"
                  class="px-1.5 py-0.5 rounded bg-amber-500/20 border border-amber-500/40 text-amber-300 text-[10px] font-bold flex items-center gap-1"
                >
                  ⭐ Default Startup DB
                </span>
                <span
                  v-if="isActivePath(db.basePath)"
                  class="px-1.5 py-0.5 rounded bg-emerald-500/20 border border-emerald-500/40 text-emerald-300 text-[10px] font-bold"
                >
                  📂 Active Working DB
                </span>
                <span
                  v-if="isReferencePath(db.basePath)"
                  class="px-1.5 py-0.5 rounded bg-cyan-500/20 border border-cyan-500/40 text-cyan-300 text-[10px] font-bold"
                >
                  📚 Reference Theory DB
                </span>
              </div>
              <div class="text-[10px] text-zinc-500 font-mono truncate mt-0.5">
                {{ db.description ? `${db.description} • ` : '' }}{{ db.basePath }}
              </div>
            </div>

            <!-- Role Assignment Actions -->
            <div class="flex items-center gap-1.5 shrink-0 flex-wrap">
              <button
                class="px-2 py-1 rounded-md text-[11px] font-semibold border transition-colors cursor-pointer"
                :class="db.writePolicy === 'locked' || db.isLocked
                  ? 'bg-rose-950/60 border-rose-500/40 text-rose-300'
                  : db.writePolicy === 'sync_only'
                    ? 'bg-cyan-950/60 border-cyan-500/40 text-cyan-300'
                    : 'bg-emerald-950/50 border-emerald-500/40 text-emerald-300'"
                title="Click to cycle write policy: Editable (Personal) -> Sync-Only (Reference) -> Locked (Read-Only)"
                @click="emit('cycleWritePolicy', db.basePath, db.writePolicy)"
              >
                {{ db.writePolicy === 'locked' || db.isLocked ? '🔒 Locked' : db.writePolicy === 'sync_only' ? '🔄 Sync-Only' : '🔓 Editable' }}
              </button>

              <button
                class="px-2.5 py-1 rounded-md text-[11px] font-semibold transition-colors cursor-pointer"
                :class="isActivePath(db.basePath)
                  ? 'bg-emerald-600 text-white'
                  : 'bg-zinc-800 hover:bg-emerald-600 text-zinc-200 hover:text-white'"
                @click="emit('openDiscoveredBase', db.basePath)"
              >
                {{ isActivePath(db.basePath) ? '✓ Active' : 'Open Active' }}
              </button>

              <button
                class="px-2 py-1 rounded-md text-[11px] font-semibold border transition-colors cursor-pointer"
                :class="isDefaultPath(db.basePath)
                  ? 'bg-amber-500/20 border-amber-500/50 text-amber-300'
                  : 'bg-zinc-800/80 hover:bg-amber-950/60 border-zinc-700 text-zinc-300 hover:text-amber-300'"
                title="Set as Default Database (auto-opens on startup)"
                @click="emit('setDefaultDB', isDefaultPath(db.basePath) ? '' : db.basePath)"
              >
                {{ isDefaultPath(db.basePath) ? '⭐ Default' : '☆ Set Default' }}
              </button>

              <button
                class="px-2 py-1 rounded-md text-[11px] font-semibold border transition-colors cursor-pointer"
                :class="isReferencePath(db.basePath)
                  ? 'bg-cyan-500/20 border-cyan-500/50 text-cyan-300'
                  : 'bg-zinc-800/80 hover:bg-cyan-950/60 border-zinc-700 text-zinc-300 hover:text-cyan-300'"
                title="Set as Reference Database for Opening Tree & Opponent Prep"
                @click="emit('setReferenceDB', isReferencePath(db.basePath) ? '' : db.basePath, true)"
              >
                {{ isReferencePath(db.basePath) ? '📚 Reference' : 'Set Reference' }}
              </button>

              <button
                class="px-2 py-1 rounded-md text-[11px] font-semibold border border-emerald-500/40 bg-emerald-950/40 hover:bg-emerald-900/60 text-emerald-300 flex items-center gap-1 transition-colors cursor-pointer"
                :disabled="isUpdatingFromLocal"
                title="Update this database with all non-duplicate games from other local databases already on your computer"
                @click="emit('updateFromLocal', db.basePath)"
              >
                <RefreshCw class="w-3 h-3" :class="{ 'animate-spin': isUpdatingFromLocal }" />
                <span>Update Local</span>
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
