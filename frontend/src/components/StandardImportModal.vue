<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import {
  Database,
  FileUp,
  FolderOpen,
  ShieldAlert,
  BookOpen,
  UserCheck,
  Sparkles,
  CheckCircle2,
  X,
  Lock,
  RefreshCw,
  Unlock,
  FileText,
} from '@lucide/vue'
import {
  ImportStandardDatabase,
  OpenFileDialog,
  CancelActiveOperation,
} from '../../wailsjs/go/main/App'
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'

const props = defineProps<{
  defaultDbName?: string
  referenceDbName?: string
  initialMode?: 'original_and_master' | 'original_only' | 'user_db' | 'board_only'
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'importedDatabase', result: any): void
  (e: 'loadPgnOnBoard', pgnText: string): void
}>()

const importMode = ref<'original_and_master' | 'original_only' | 'user_db' | 'board_only'>(
  props.initialMode || 'original_and_master'
)
const sourcePath = ref('')
const pgnText = ref('')
const archiveName = ref('')
const masterDbName = ref(props.referenceDbName || 'Master_Reference')
const userDbName = ref(props.defaultDbName || 'My_Games')
const setMasterAsRef = ref(true)
const setUserAsDefault = ref(true)
const openAfterImport = ref(true)

const isImporting = ref(false)
const errorMsg = ref<string | null>(null)
const importResult = ref<any | null>(null)
const progressInfo = ref<{ target: string; source: string; scanned: number; imported: number; skipped: number } | null>(null)

const inferredArchivePlaceholder = computed(() => {
  if (sourcePath.value) {
    const base = sourcePath.value.split(/[/\\]/).pop() || 'Imported_Archive'
    return base.replace(/\.(si4|sg4|sn4|pgn)$/i, '').replace(/[^a-zA-Z0-9_-]/g, '_')
  }
  return 'Candidates_2026_Original'
})

async function handleBrowseFile() {
  errorMsg.value = null
  try {
    const chosen = await OpenFileDialog()
    if (chosen) {
      sourcePath.value = chosen
      if (!archiveName.value.trim()) {
        const base = chosen.split(/[/\\]/).pop() || ''
        archiveName.value = base.replace(/\.(si4|sg4|sn4|pgn)$/i, '').replace(/[^a-zA-Z0-9_-]/g, '_')
      }
    }
  } catch (err: any) {
    errorMsg.value = err?.message || String(err)
  }
}

async function handleRunImport() {
  errorMsg.value = null
  importResult.value = null

  if (importMode.value === 'board_only') {
    if (!pgnText.value.trim()) {
      errorMsg.value = 'Please paste a PGN game into the text box below to load it onto the board.'
      return
    }
    emit('loadPgnOnBoard', pgnText.value.trim())
    return
  }

  if (!sourcePath.value.trim() && !pgnText.value.trim()) {
    errorMsg.value = 'Please select a .si4 / .pgn file from disk or paste PGN text below.'
    return
  }

  isImporting.value = true
  progressInfo.value = null
  try {
    const res = await ImportStandardDatabase({
      sourcePath: sourcePath.value.trim(),
      pgnText: pgnText.value.trim(),
      archiveName: archiveName.value.trim() || inferredArchivePlaceholder.value,
      importMode: importMode.value,
      masterDbName: masterDbName.value.trim() || 'Master_Reference',
      userDbName: userDbName.value.trim() || 'My_Games',
      setMasterAsRef: setMasterAsRef.value,
      setUserAsDefault: setUserAsDefault.value,
      openAfterImport: openAfterImport.value,
    })
    importResult.value = res
    emit('importedDatabase', res)
  } catch (err: any) {
    errorMsg.value = err?.message || String(err)
  } finally {
    isImporting.value = false
  }
}

async function handleCancelImport() {
  try {
    await CancelActiveOperation()
  } catch {
    // ignore
  }
}

onMounted(() => {
  EventsOn('import:progress', (data: any) => {
    if (data) {
      progressInfo.value = data
    }
  })
})

onUnmounted(() => {
  EventsOff('import:progress')
})
</script>

<template>
  <div
    class="fixed inset-0 bg-black/75 backdrop-blur-xs flex items-center justify-center z-[110] p-4 animate-fade-in"
    @click.self="emit('close')"
  >
    <div class="bg-zinc-900 border border-zinc-700 w-full max-w-3xl rounded-2xl shadow-2xl flex flex-col max-h-[90vh] overflow-hidden">
      <!-- Header -->
      <div class="px-5 py-3.5 bg-zinc-950 border-b border-zinc-800 flex items-center justify-between">
        <div class="flex items-center gap-2.5">
          <div class="w-8 h-8 rounded-xl bg-emerald-500/15 border border-emerald-500/40 flex items-center justify-center text-emerald-400">
            <FileUp class="w-4 h-4" />
          </div>
          <div>
            <h2 class="text-sm font-bold text-zinc-100">Standardized Database &amp; PGN Import</h2>
            <p class="text-[11px] text-zinc-400">
              Cleanly separate Original Source Archives, your Master Reference DB, and your Personal User DB
            </p>
          </div>
        </div>
        <button
          class="text-zinc-400 hover:text-white p-1 rounded-lg hover:bg-zinc-800 cursor-pointer"
          aria-label="Close Import Modal"
          @click="emit('close')"
        >
          <X class="w-4 h-4" />
        </button>
      </div>

      <div class="p-5 overflow-y-auto space-y-4 text-xs">
        <!-- 3-Tier Database Architecture Explainer Banner -->
        <div class="grid grid-cols-1 md:grid-cols-3 gap-2.5 bg-zinc-950/90 border border-zinc-800 rounded-xl p-3">
          <div class="p-2.5 rounded-lg bg-rose-950/20 border border-rose-500/25 flex flex-col gap-1">
            <div class="flex items-center justify-between">
              <span class="font-bold text-rose-300 flex items-center gap-1.5">
                <Lock class="w-3.5 h-3.5 text-rose-400" />
                <span>1. Original Archive</span>
              </span>
              <span class="px-1.5 py-0.2 rounded text-[9px] font-mono bg-rose-950/70 text-rose-300 border border-rose-700/40">🔒 Locked</span>
            </div>
            <p class="text-[11px] text-zinc-400 leading-relaxed">
              Untouched copy of the imported database or PGN archive. Locked so your original source is never contaminated by manual edits.
            </p>
          </div>

          <div class="p-2.5 rounded-lg bg-cyan-950/20 border border-cyan-500/25 flex flex-col gap-1">
            <div class="flex items-center justify-between">
              <span class="font-bold text-cyan-300 flex items-center gap-1.5">
                <BookOpen class="w-3.5 h-3.5 text-cyan-400" />
                <span>2. Master Reference</span>
              </span>
              <span class="px-1.5 py-0.2 rounded text-[9px] font-mono bg-cyan-950/70 text-cyan-300 border border-cyan-700/40">🔄 Sync-Only</span>
            </div>
            <p class="text-[11px] text-zinc-400 leading-relaxed">
              Unified, deduplicated encyclopedia built from your Original Archives. Queried in the background by the <strong>Opening Tree</strong> &amp; <strong>ECO Explorer</strong>.
            </p>
          </div>

          <div class="p-2.5 rounded-lg bg-amber-950/20 border border-amber-500/25 flex flex-col gap-1">
            <div class="flex items-center justify-between">
              <span class="font-bold text-amber-300 flex items-center gap-1.5">
                <UserCheck class="w-3.5 h-3.5 text-amber-400" />
                <span>3. Personal User DB</span>
              </span>
              <span class="px-1.5 py-0.2 rounded text-[9px] font-mono bg-emerald-950/70 text-emerald-300 border border-emerald-700/40">🔓 Editable</span>
            </div>
            <p class="text-[11px] text-zinc-400 leading-relaxed">
              <strong>Your personal chess life</strong> (your own games, repertoire notes &amp; annotations). Opened at startup and powers Cyclical Drill &amp; Blunder Vault.
            </p>
          </div>
        </div>

        <!-- Step 1: Choose How to Import -->
        <div>
          <label class="block text-[11px] font-bold uppercase tracking-wider text-zinc-400 mb-2">
            Step 1 — Choose Destination Tier
          </label>
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-2">
            <button
              type="button"
              class="text-left p-3 rounded-xl border transition-all cursor-pointer flex flex-col gap-1"
              :class="importMode === 'original_and_master'
                ? 'bg-emerald-950/40 border-emerald-500 ring-1 ring-emerald-500/60'
                : 'bg-zinc-950/60 border-zinc-800 hover:border-zinc-700'"
              @click="importMode = 'original_and_master'"
            >
              <div class="flex items-center justify-between">
                <span class="font-bold text-zinc-100 flex items-center gap-1.5">
                  <Sparkles class="w-3.5 h-3.5 text-emerald-400" />
                  <span>Standard Reference Import (Original + Master)</span>
                </span>
                <span class="px-1.5 py-0.2 rounded text-[9px] font-mono bg-emerald-500/20 text-emerald-300">Recommended</span>
              </div>
              <p class="text-[11px] text-zinc-400">
                Saves a locked <strong>Original Archive</strong> copy AND merges/deduplicates all games into your <strong>Master Reference DB</strong>.
              </p>
            </button>

            <button
              type="button"
              class="text-left p-3 rounded-xl border transition-all cursor-pointer flex flex-col gap-1"
              :class="importMode === 'user_db'
                ? 'bg-amber-950/40 border-amber-500 ring-1 ring-amber-500/60'
                : 'bg-zinc-950/60 border-zinc-800 hover:border-zinc-700'"
              @click="importMode = 'user_db'"
            >
              <div class="flex items-center justify-between">
                <span class="font-bold text-zinc-100 flex items-center gap-1.5">
                  <Unlock class="w-3.5 h-3.5 text-amber-400" />
                  <span>Import into My Personal User DB</span>
                </span>
                <span class="px-1.5 py-0.2 rounded text-[9px] font-mono bg-amber-500/20 text-amber-300">Editable</span>
              </div>
              <p class="text-[11px] text-zinc-400">
                Appends &amp; deduplicates your own OTB or study games into your editable <strong>Personal User DB ({{ userDbName }})</strong>.
              </p>
            </button>

            <button
              type="button"
              class="text-left p-3 rounded-xl border transition-all cursor-pointer flex flex-col gap-1"
              :class="importMode === 'original_only'
                ? 'bg-rose-950/40 border-rose-500 ring-1 ring-rose-500/60'
                : 'bg-zinc-950/60 border-zinc-800 hover:border-zinc-700'"
              @click="importMode = 'original_only'"
            >
              <div class="flex items-center justify-between">
                <span class="font-bold text-zinc-100 flex items-center gap-1.5">
                  <Lock class="w-3.5 h-3.5 text-rose-400" />
                  <span>Standalone Original Archive Only</span>
                </span>
                <span class="px-1.5 py-0.2 rounded text-[9px] font-mono bg-rose-500/20 text-rose-300">Read-Only</span>
              </div>
              <p class="text-[11px] text-zinc-400">
                Imports into managed storage as an isolated, locked <strong>Original Archive</strong> without merging into Master.
              </p>
            </button>

            <button
              type="button"
              class="text-left p-3 rounded-xl border transition-all cursor-pointer flex flex-col gap-1"
              :class="importMode === 'board_only'
                ? 'bg-sky-950/40 border-sky-500 ring-1 ring-sky-500/60'
                : 'bg-zinc-950/60 border-zinc-800 hover:border-zinc-700'"
              @click="importMode = 'board_only'"
            >
              <div class="flex items-center justify-between">
                <span class="font-bold text-zinc-100 flex items-center gap-1.5">
                  <FileText class="w-3.5 h-3.5 text-sky-400" />
                  <span>Load Single PGN Game onto Board</span>
                </span>
                <span class="px-1.5 py-0.2 rounded text-[9px] font-mono bg-sky-500/20 text-sky-300">Board Scratch</span>
              </div>
              <p class="text-[11px] text-zinc-400">
                Loads pasted PGN text onto the analysis chessboard immediately without modifying any database.
              </p>
            </button>
          </div>
        </div>

        <!-- Step 2: Select Source File or Paste PGN -->
        <div class="space-y-3 bg-zinc-950/70 border border-zinc-800 rounded-xl p-3.5">
          <label class="block text-[11px] font-bold uppercase tracking-wider text-zinc-400">
            Step 2 — Select Source File (.si4 / .pgn) or Paste PGN Text
          </label>

          <div v-if="importMode !== 'board_only'" class="flex flex-col sm:flex-row items-stretch sm:items-center gap-2">
            <input
              v-model="sourcePath"
              type="text"
              placeholder="Path to external .pgn or .si4 database file..."
              class="flex-1 bg-zinc-900 border border-zinc-700 rounded-lg px-3 py-2 text-xs text-zinc-100 font-mono focus:outline-none focus:border-emerald-500"
            />
            <button
              type="button"
              class="px-3.5 py-2 rounded-lg bg-zinc-800 hover:bg-zinc-700 border border-zinc-600 text-zinc-100 font-semibold flex items-center justify-center gap-1.5 cursor-pointer shrink-0"
              @click="handleBrowseFile"
            >
              <FolderOpen class="w-3.5 h-3.5 text-emerald-400" />
              <span>Browse File...</span>
            </button>
          </div>

          <!-- Naming & Role Options -->
          <div v-if="importMode !== 'board_only'" class="grid grid-cols-1 sm:grid-cols-2 gap-3 pt-1">
            <div v-if="importMode === 'original_and_master' || importMode === 'original_only'">
              <label class="block text-[11px] text-zinc-400 mb-1">Original Archive Name (Locked Source Copy):</label>
              <input
                v-model="archiveName"
                type="text"
                :placeholder="inferredArchivePlaceholder"
                class="w-full bg-zinc-900 border border-zinc-700 rounded-lg px-2.5 py-1.5 text-xs text-zinc-100 font-mono focus:outline-none focus:border-emerald-500"
              />
            </div>

            <div v-if="importMode === 'original_and_master'">
              <label class="block text-[11px] text-zinc-400 mb-1">Target Master Reference Database:</label>
              <input
                v-model="masterDbName"
                type="text"
                placeholder="Master_Reference"
                class="w-full bg-zinc-900 border border-zinc-700 rounded-lg px-2.5 py-1.5 text-xs text-cyan-300 font-mono focus:outline-none focus:border-cyan-500"
              />
            </div>

            <div v-if="importMode === 'user_db'">
              <label class="block text-[11px] text-zinc-400 mb-1">Target Personal User Database (Editable):</label>
              <input
                v-model="userDbName"
                type="text"
                placeholder="My_Games"
                class="w-full bg-zinc-900 border border-zinc-700 rounded-lg px-2.5 py-1.5 text-xs text-amber-300 font-mono focus:outline-none focus:border-amber-500"
              />
            </div>
          </div>

          <div v-if="importMode !== 'board_only'" class="flex flex-wrap items-center gap-4 pt-1 text-[11px] text-zinc-300">
            <label v-if="importMode === 'original_and_master'" class="flex items-center gap-1.5 cursor-pointer">
              <input v-model="setMasterAsRef" type="checkbox" class="rounded accent-emerald-500" />
              <span>Set <strong>{{ masterDbName || 'Master_Reference' }}</strong> as Default Reference DB (for Opening Tree)</span>
            </label>
            <label v-if="importMode === 'user_db'" class="flex items-center gap-1.5 cursor-pointer">
              <input v-model="setUserAsDefault" type="checkbox" class="rounded accent-amber-500" />
              <span>Set <strong>{{ userDbName || 'My_Games' }}</strong> as Default Startup User DB</span>
            </label>
            <label class="flex items-center gap-1.5 cursor-pointer">
              <input v-model="openAfterImport" type="checkbox" class="rounded accent-emerald-500" />
              <span>Open in Game Browser after import</span>
            </label>
          </div>

          <!-- Optional or Primary PGN Textarea -->
          <div>
            <label class="block text-[11px] text-zinc-400 mb-1">
              {{ importMode === 'board_only' ? 'Paste PGN Game Text:' : 'Or Paste Raw PGN Games (optional if a file is selected above):' }}
            </label>
            <textarea
              v-model="pgnText"
              rows="5"
              placeholder="[Event &quot;Candidates 2026&quot;]&#10;[White &quot;Caruana, Fabiano&quot;]&#10;[Black &quot;Nakamura, Hikaru&quot;]&#10;&#10;1. e4 e5 2. Nf3 Nc6 3. Bb5 a6 *"
              class="w-full bg-zinc-900 border border-zinc-700 rounded-lg p-2.5 text-xs font-mono text-zinc-200 focus:outline-none focus:border-emerald-500"
            />
          </div>
        </div>

        <!-- Live Progress Banner -->
        <div
          v-if="isImporting"
          class="p-3 rounded-xl bg-emerald-950/30 border border-emerald-500/40 flex items-center justify-between gap-3"
        >
          <div class="flex items-center gap-2.5">
            <RefreshCw class="w-4 h-4 text-emerald-400 animate-spin shrink-0" />
            <div>
              <div class="font-bold text-emerald-300">Importing &amp; Deduplicating Games...</div>
              <div v-if="progressInfo" class="text-[11px] text-zinc-300 font-mono">
                Target: {{ progressInfo.target }} • Scanned: {{ progressInfo.scanned.toLocaleString() }} • Added: +{{ progressInfo.imported.toLocaleString() }} • Dupes Skipped: {{ progressInfo.skipped.toLocaleString() }}
              </div>
            </div>
          </div>
          <button
            type="button"
            class="px-3 py-1 rounded-lg bg-rose-600 hover:bg-rose-500 text-white font-semibold text-xs cursor-pointer"
            @click="handleCancelImport"
          >
            Cancel
          </button>
        </div>

        <!-- Error Banner -->
        <div
          v-if="errorMsg"
          class="p-3 rounded-xl bg-rose-950/50 border border-rose-500/40 text-rose-200 flex items-center gap-2"
        >
          <ShieldAlert class="w-4 h-4 text-rose-400 shrink-0" />
          <span>{{ errorMsg }}</span>
        </div>

        <!-- Result Banner -->
        <div
          v-if="importResult"
          class="p-3.5 rounded-xl bg-emerald-950/40 border border-emerald-500/40 text-emerald-200 space-y-1.5"
        >
          <div class="flex items-center gap-2 font-bold text-emerald-300">
            <CheckCircle2 class="w-4 h-4 text-emerald-400 shrink-0" />
            <span>Import Complete!</span>
          </div>
          <p class="text-xs text-zinc-200 leading-relaxed">{{ importResult.summary }}</p>
        </div>
      </div>

      <!-- Footer -->
      <div class="px-5 py-3 bg-zinc-950 border-t border-zinc-800 flex items-center justify-between gap-2">
        <span class="text-[11px] text-zinc-500 font-mono">
          Managed Storage: ~/.local/share/newscid/databases
        </span>
        <div class="flex items-center gap-2">
          <button
            type="button"
            class="px-4 py-1.5 rounded-lg bg-zinc-800 hover:bg-zinc-700 text-zinc-300 text-xs font-medium cursor-pointer"
            @click="emit('close')"
          >
            {{ importResult ? 'Done' : 'Cancel' }}
          </button>
          <button
            type="button"
            :disabled="isImporting"
            class="px-4 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-500 disabled:opacity-50 text-white text-xs font-bold flex items-center gap-1.5 cursor-pointer shadow-md"
            @click="handleRunImport"
          >
            <FileUp class="w-3.5 h-3.5" />
            <span>
              {{
                importMode === 'board_only'
                  ? 'Load PGN onto Board'
                  : importMode === 'original_and_master'
                    ? 'Import Original & Merge into Master'
                    : importMode === 'user_db'
                      ? 'Import into My Personal User DB'
                      : 'Import Standalone Original Archive'
              }}
            </span>
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
