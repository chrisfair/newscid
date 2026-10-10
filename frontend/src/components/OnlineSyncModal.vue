<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import {
  Globe,
  Plus,
  Trash2,
  Save,
  RefreshCw,
  FolderOpen,
  CheckCircle2,
  AlertTriangle,
  XCircle,
  X,
  User,
  Shield,
  Layers,
  Calendar,
  Clock,
  Eye,
  EyeOff,
  Check,
  Database
} from '@lucide/vue'
import {
  GetSyncProfiles,
  SaveSyncProfile,
  DeleteSyncProfile,
  SyncOnlineProfile,
  FetchLichessPreview,
  FetchChessComPreview,
  OpenFileDialog,
  ImportPlayerGames
} from '../../wailsjs/go/main/App'
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'
import { online } from '../../wailsjs/go/models'

const props = defineProps<{
  activeDbPath?: string
  activeDbName?: string
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'dbUpdated'): void
}>()

// State
const profiles = ref<online.Profile[]>([])
const selectedProfileId = ref<string>('')
const loadingProfiles = ref(true)
const isSaving = ref(false)
const isSyncing = ref(false)
const syncProgressText = ref('')
const statusMessage = ref('')
const errorMessage = ref('')
const showToken = ref(false)

// Previews
const chessComArchivesCount = ref<number | null>(null)
const testingChessCom = ref(false)
const lichessPreviewResult = ref<string | null>(null)
const testingLichess = ref(false)

// Active Editing Profile Model
const currentProfile = ref<online.Profile>(
  new online.Profile({
    id: '',
    name: 'My Online Games',
    targetDatabase: '',
    chessComUsername: '',
    lichessUsername: '',
    lichessToken: '',
    lastSyncTime: 0,
    lastSyncGames: 0,
    ratedOnly: false,
    autoDeduplicate: true,
    createdAt: 0,
    updatedAt: 0
  })
)

// Active Modal Tab: 'profiles' (My accounts) or 'import-opponent' (Import other users)
const activeTab = ref<'profiles' | 'import-opponent'>('profiles')

// Opponent / GM Import State
const opponentForm = ref({
  playerName: '',
  chessComUsername: '',
  lichessUsername: '',
  targetMode: 'active' as 'active' | 'dedicated' | 'blended',
  dedicatedPath: '',
  blendedPath: '',
  createProfile: true,
  ratedOnly: false,
  autoDeduplicate: true,
  maxGames: 200,
})
const isImportingOpponent = ref(false)
const opponentImportResult = ref<online.SyncResult | null>(null)

async function pickDedicatedPath() {
  try {
    const path = await OpenFileDialog()
    if (path) {
      opponentForm.value.dedicatedPath = path
    }
  } catch (err: any) {
    console.error('Error selecting dedicated database path:', err)
  }
}

async function pickBlendedPath() {
  try {
    const path = await OpenFileDialog()
    if (path) {
      opponentForm.value.blendedPath = path
    }
  } catch (err: any) {
    console.error('Error selecting blended database path:', err)
  }
}

async function handleImportOpponentGames() {
  if (!opponentForm.value.chessComUsername.trim() && !opponentForm.value.lichessUsername.trim()) {
    errorMessage.value = 'Please provide at least one username (Chess.com or Lichess.org)'
    return
  }

  if (opponentForm.value.targetMode === 'active' && !props.activeDbPath) {
    errorMessage.value = 'No database is currently open. Please open a database or choose Dedicated Database mode.'
    return
  }

  if (opponentForm.value.targetMode === 'dedicated' && !opponentForm.value.dedicatedPath.trim()) {
    errorMessage.value = 'Please select or enter the Dedicated Database path.'
    return
  }

  isImportingOpponent.value = true
  errorMessage.value = ''
  statusMessage.value = 'Connecting to online chess servers...'
  opponentImportResult.value = null

  let cancelEvents: (() => void) | null = null
  if (typeof window !== 'undefined' && (window as any).runtime?.EventsOnMultiple) {
    cancelEvents = EventsOn('online:fetch-progress', (progress: any) => {
      if (progress && progress.message) {
        statusMessage.value = progress.message
      }
    })
  }

  try {
    const displayName = opponentForm.value.playerName.trim() || opponentForm.value.chessComUsername.trim() || opponentForm.value.lichessUsername.trim()
    const res = await ImportPlayerGames(new online.ImportPlayerOptions({
      playerName: displayName,
      chessComUsername: opponentForm.value.chessComUsername.trim(),
      lichessUsername: opponentForm.value.lichessUsername.trim(),
      targetMode: opponentForm.value.targetMode,
      dedicatedPath: opponentForm.value.dedicatedPath.trim(),
      blendedPath: opponentForm.value.blendedPath.trim(),
      createProfile: opponentForm.value.createProfile,
      ratedOnly: opponentForm.value.ratedOnly,
      autoDeduplicate: opponentForm.value.autoDeduplicate,
      maxGames: Number(opponentForm.value.maxGames) || 0,
    }))

    opponentImportResult.value = res
    if (res.errors && res.errors.length > 0) {
      errorMessage.value = `Import finished with warnings: ${res.errors.join('; ')}`
    } else {
      statusMessage.value = `Successfully imported ${res.imported} games for ${displayName}!`
    }

    emit('dbUpdated')
    await loadProfiles()
  } catch (err: any) {
    errorMessage.value = `Opponent import failed: ${err?.message || err}`
  } finally {
    if (cancelEvents) {
      cancelEvents()
    }
    if (typeof window !== 'undefined' && (window as any).runtime?.EventsOff) {
      EventsOff('online:fetch-progress')
    }
    isImportingOpponent.value = false
  }
}

// Target database mode: 'active' or 'custom'
const targetMode = ref<'active' | 'custom'>('active')

// Last Sync Result
const lastSyncResult = ref<online.SyncResult | null>(null)

const activeProfile = computed(() => {
  return profiles.value.find(p => p.id === selectedProfileId.value)
})

onMounted(async () => {
  await loadProfiles()
})

async function loadProfiles() {
  loadingProfiles.value = true
  errorMessage.value = ''
  try {
    const list = await GetSyncProfiles()
    profiles.value = list || []
    if (profiles.value.length > 0) {
      if (!selectedProfileId.value || !profiles.value.some(p => p.id === selectedProfileId.value)) {
        selectProfile(profiles.value[0])
      }
    } else {
      createNewProfile()
    }
  } catch (err: any) {
    errorMessage.value = `Failed to load profiles: ${err?.message || err}`
  } finally {
    loadingProfiles.value = false
  }
}

function selectProfile(p: online.Profile) {
  selectedProfileId.value = p.id
  currentProfile.value = new online.Profile({
    id: p.id,
    name: p.name,
    targetDatabase: p.targetDatabase,
    chessComUsername: p.chessComUsername,
    lichessUsername: p.lichessUsername,
    lichessToken: p.lichessToken,
    lastSyncTime: p.lastSyncTime,
    lastSyncGames: p.lastSyncGames,
    ratedOnly: p.ratedOnly,
    autoDeduplicate: p.autoDeduplicate,
    createdAt: p.createdAt,
    updatedAt: p.updatedAt
  })

  targetMode.value = p.targetDatabase ? 'custom' : 'active'
  lastSyncResult.value = null
  chessComArchivesCount.value = null
  lichessPreviewResult.value = null
  statusMessage.value = ''
  errorMessage.value = ''
}

function createNewProfile() {
  selectedProfileId.value = ''
  currentProfile.value = new online.Profile({
    id: '',
    name: `New Sync Profile ${profiles.value.length + 1}`,
    targetDatabase: '',
    chessComUsername: '',
    lichessUsername: '',
    lichessToken: '',
    lastSyncTime: 0,
    lastSyncGames: 0,
    ratedOnly: false,
    autoDeduplicate: true,
    createdAt: 0,
    updatedAt: 0
  })
  targetMode.value = 'active'
  lastSyncResult.value = null
  chessComArchivesCount.value = null
  lichessPreviewResult.value = null
  statusMessage.value = ''
  errorMessage.value = ''
}

async function handleBrowseTargetDB() {
  try {
    const path = await OpenFileDialog()
    if (path) {
      currentProfile.value.targetDatabase = path
    }
  } catch (err: any) {
    errorMessage.value = `Failed to select database file: ${err?.message || err}`
  }
}

async function testChessCom() {
  if (!currentProfile.value.chessComUsername.trim()) return
  testingChessCom.value = true
  errorMessage.value = ''
  chessComArchivesCount.value = null
  try {
    const archives = await FetchChessComPreview(currentProfile.value.chessComUsername.trim())
    chessComArchivesCount.value = archives ? archives.length : 0
  } catch (err: any) {
    errorMessage.value = `Chess.com verify error: ${err?.message || err}`
  } finally {
    testingChessCom.value = false
  }
}

async function testLichess() {
  if (!currentProfile.value.lichessUsername.trim()) return
  testingLichess.value = true
  errorMessage.value = ''
  lichessPreviewResult.value = null
  try {
    const pgn = await FetchLichessPreview(currentProfile.value.lichessUsername.trim(), 2)
    if (pgn && pgn.includes('[Event')) {
      lichessPreviewResult.value = 'Found games on Lichess.org'
    } else {
      lichessPreviewResult.value = 'User exists, no games found'
    }
  } catch (err: any) {
    errorMessage.value = `Lichess verify error: ${err?.message || err}`
  } finally {
    testingLichess.value = false
  }
}

async function handleSaveProfile() {
  if (!currentProfile.value.name.trim()) {
    errorMessage.value = 'Profile name cannot be empty'
    return
  }

  if (!currentProfile.value.chessComUsername.trim() && !currentProfile.value.lichessUsername.trim()) {
    errorMessage.value = 'Please provide at least one username (Chess.com or Lichess.org)'
    return
  }

  isSaving.value = true
  errorMessage.value = ''
  statusMessage.value = ''

  try {
    // If targetMode is active, clear targetDatabase
    if (targetMode.value === 'active') {
      currentProfile.value.targetDatabase = ''
    }

    await SaveSyncProfile(currentProfile.value)
    statusMessage.value = 'Profile saved successfully'

    const list = await GetSyncProfiles()
    profiles.value = list || []

    // Ensure selection stays on saved profile
    if (!currentProfile.value.id && profiles.value.length > 0) {
      const newest = profiles.value[profiles.value.length - 1]
      selectProfile(newest)
    } else if (currentProfile.value.id) {
      const updated = profiles.value.find(p => p.id === currentProfile.value.id)
      if (updated) selectProfile(updated)
    }
  } catch (err: any) {
    errorMessage.value = `Failed to save profile: ${err?.message || err}`
  } finally {
    isSaving.value = false
  }
}

async function handleDeleteProfile() {
  if (!currentProfile.value.id) return
  if (!confirm(`Are you sure you want to delete profile "${currentProfile.value.name}"?`)) return

  errorMessage.value = ''
  statusMessage.value = ''
  try {
    await DeleteSyncProfile(currentProfile.value.id)
    await loadProfiles()
  } catch (err: any) {
    errorMessage.value = `Failed to delete profile: ${err?.message || err}`
  }
}

async function handleSyncNow() {
  if (!currentProfile.value.name.trim()) {
    errorMessage.value = 'Please enter a valid profile name'
    return
  }
  if (!currentProfile.value.chessComUsername.trim() && !currentProfile.value.lichessUsername.trim()) {
    errorMessage.value = 'Please provide at least one platform username to sync'
    return
  }

  if (targetMode.value === 'active' && !props.activeDbPath) {
    errorMessage.value = 'No database is currently open. Please open a database or choose a specific database file.'
    return
  }

  // Save profile first before running sync
  await handleSaveProfile()
  if (errorMessage.value) return

  isSyncing.value = true
  syncProgressText.value = 'Connecting to online platforms and fetching games...'
  lastSyncResult.value = null
  statusMessage.value = ''
  errorMessage.value = ''

  try {
    const res = await SyncOnlineProfile(currentProfile.value.id)
    lastSyncResult.value = res

    if (res.errors && res.errors.length > 0) {
      errorMessage.value = `Sync finished with ${res.errors.length} issue(s): ${res.errors.join('; ')}`
    } else {
      statusMessage.value = `Successfully synced ${res.imported} games into database!`
    }

    // Refresh profiles to display updated sync metrics
    const list = await GetSyncProfiles()
    profiles.value = list || []
    const updated = profiles.value.find(p => p.id === currentProfile.value.id)
    if (updated) {
      currentProfile.value.lastSyncTime = updated.lastSyncTime
      currentProfile.value.lastSyncGames = updated.lastSyncGames
    }

    // Notify parent to refresh database view if active DB was updated
    if (targetMode.value === 'active' || (props.activeDbPath && currentProfile.value.targetDatabase.includes(props.activeDbPath))) {
      emit('dbUpdated')
    }
  } catch (err: any) {
    errorMessage.value = `Sync failed: ${err?.message || err}`
  } finally {
    isSyncing.value = false
    syncProgressText.value = ''
  }
}

function formatDate(ms: number): string {
  if (!ms || ms <= 0) return 'Never synced'
  return new Date(ms).toLocaleString(undefined, {
    dateStyle: 'medium',
    timeStyle: 'short'
  })
}
</script>

<template>
  <div class="fixed inset-0 z-50 bg-black/80 backdrop-blur-xs flex items-center justify-center p-4">
    <div class="bg-zinc-900 border border-zinc-700/80 rounded-2xl shadow-2xl w-full max-w-4xl h-[90vh] max-h-[760px] flex flex-col overflow-hidden text-zinc-200">
      
      <!-- Modal Header -->
      <div class="px-6 py-4 border-b border-zinc-800 flex items-center justify-between bg-zinc-950/60">
        <div class="flex items-center gap-3">
          <div class="w-10 h-10 rounded-xl bg-cyan-950/80 border border-cyan-700/50 flex items-center justify-center text-cyan-400 shadow-inner">
            <Globe class="w-5 h-5" />
          </div>
          <div>
            <h2 class="text-base font-semibold text-white tracking-wide flex items-center gap-2">
              Online Account Sync & Merge
              <span class="text-xs px-2 py-0.5 rounded-full bg-cyan-900/40 text-cyan-300 border border-cyan-700/40 font-mono">
                Chess.com & Lichess
              </span>
            </h2>
            <p class="text-xs text-zinc-400">
              Directly download, synchronize, and merge your online games into local Scid databases.
            </p>
          </div>
        </div>
        <button
          class="p-2 text-zinc-400 hover:text-white hover:bg-zinc-800 rounded-lg transition-colors cursor-pointer"
          @click="emit('close')"
        >
          <X class="w-5 h-5" />
        </button>
      </div>

      <!-- Modal Navigation Tabs -->
      <div class="flex items-center gap-2 px-6 pt-2.5 border-b border-zinc-800 bg-zinc-950/40 text-xs">
        <button
          class="pb-2.5 px-3 font-semibold border-b-2 transition-colors flex items-center gap-2 cursor-pointer"
          :class="activeTab === 'profiles'
            ? 'border-cyan-400 text-white font-bold'
            : 'border-transparent text-zinc-400 hover:text-zinc-200'"
          @click="activeTab = 'profiles'"
        >
          <User class="w-4 h-4 text-cyan-400" />
          <span>My Account Profiles ({{ profiles.length }})</span>
        </button>

        <button
          class="pb-2.5 px-3 font-semibold border-b-2 transition-colors flex items-center gap-2 cursor-pointer"
          :class="activeTab === 'import-opponent'
            ? 'border-purple-400 text-white font-bold'
            : 'border-transparent text-zinc-400 hover:text-zinc-200'"
          @click="activeTab = 'import-opponent'"
        >
          <Globe class="w-4 h-4 text-purple-400" />
          <span>Import Another User / Opponent Games</span>
          <span class="px-1.5 py-0.5 rounded bg-purple-950/80 border border-purple-800 text-[10px] text-purple-300 font-mono">
            Dedicated & Blended
          </span>
        </button>
      </div>

      <!-- Modal Body (Two-Column Layout for Profiles) -->
      <div v-if="activeTab === 'profiles'" class="flex-1 flex overflow-hidden">
        
        <!-- Left Column: Profiles List -->
        <div class="w-72 border-r border-zinc-800 bg-zinc-950/30 flex flex-col p-4">
          <div class="flex items-center justify-between mb-3">
            <span class="text-xs font-semibold text-zinc-400 uppercase tracking-wider">Sync Profiles</span>
            <button
              class="px-2.5 py-1 text-xs rounded-md bg-cyan-600/20 hover:bg-cyan-600/30 text-cyan-300 border border-cyan-500/30 flex items-center gap-1.5 transition-colors cursor-pointer"
              @click="createNewProfile"
            >
              <Plus class="w-3.5 h-3.5" />
              <span>Add</span>
            </button>
          </div>

          <!-- Profile List -->
          <div class="flex-1 overflow-y-auto space-y-2 pr-1">
            <div
              v-for="p in profiles"
              :key="p.id"
              class="p-3 rounded-xl border text-left cursor-pointer transition-all"
              :class="selectedProfileId === p.id 
                ? 'bg-cyan-950/40 border-cyan-500/60 shadow-md ring-1 ring-cyan-500/30' 
                : 'bg-zinc-900/60 border-zinc-800 hover:border-zinc-700 hover:bg-zinc-800/50'"
              @click="selectProfile(p)"
            >
              <div class="flex items-center justify-between">
                <span class="font-medium text-sm text-zinc-100 truncate">{{ p.name }}</span>
                <span v-if="p.lastSyncGames > 0" class="text-[10px] px-1.5 py-0.5 rounded bg-zinc-800 text-zinc-300 font-mono">
                  +{{ p.lastSyncGames }}
                </span>
              </div>

              <!-- Badges for configured platforms -->
              <div class="flex items-center gap-1.5 mt-2">
                <span
                  v-if="p.chessComUsername"
                  class="text-[10px] px-1.5 py-0.5 rounded bg-emerald-950/60 text-emerald-300 border border-emerald-800/40 font-medium"
                >
                  Chess.com
                </span>
                <span
                  v-if="p.lichessUsername"
                  class="text-[10px] px-1.5 py-0.5 rounded bg-blue-950/60 text-blue-300 border border-blue-800/40 font-medium"
                >
                  Lichess
                </span>
              </div>

              <!-- Last sync date -->
              <div class="flex items-center gap-1 text-[11px] text-zinc-500 mt-2">
                <Clock class="w-3 h-3 text-zinc-600" />
                <span>{{ formatDate(p.lastSyncTime) }}</span>
              </div>
            </div>

            <div v-if="profiles.length === 0 && !loadingProfiles" class="p-4 text-center text-xs text-zinc-500">
              No profiles created yet. Click "+ Add" to set up your online accounts.
            </div>
          </div>
        </div>

        <!-- Right Column: Profile Editor & Sync Execution -->
        <div class="flex-1 overflow-y-auto p-6 space-y-6">
          
          <!-- Status / Error Notification Banners -->
          <div
            v-if="errorMessage"
            class="p-3 rounded-xl bg-red-950/50 border border-red-800/60 text-red-200 text-xs flex items-start gap-2.5"
          >
            <AlertTriangle class="w-4 h-4 text-red-400 shrink-0 mt-0.5" />
            <div class="flex-1 whitespace-pre-line">{{ errorMessage }}</div>
            <button class="text-red-400 hover:text-red-200" @click="errorMessage = ''">
              <X class="w-4 h-4" />
            </button>
          </div>

          <div
            v-if="statusMessage"
            class="p-3 rounded-xl bg-emerald-950/50 border border-emerald-800/60 text-emerald-200 text-xs flex items-center gap-2.5"
          >
            <CheckCircle2 class="w-4 h-4 text-emerald-400 shrink-0" />
            <span class="flex-1">{{ statusMessage }}</span>
            <button class="text-emerald-400 hover:text-emerald-200" @click="statusMessage = ''">
              <X class="w-4 h-4" />
            </button>
          </div>

          <!-- Section: Profile Name -->
          <div class="space-y-1.5">
            <label class="text-xs font-semibold text-zinc-300 flex items-center gap-1.5">
              Profile Name
            </label>
            <input
              v-model="currentProfile.name"
              type="text"
              placeholder="e.g. My Online Blitz & Rapid Games"
              class="w-full px-3 py-2 bg-zinc-950 border border-zinc-700/80 rounded-lg text-sm text-zinc-100 placeholder-zinc-500 focus:outline-none focus:border-cyan-500 focus:ring-1 focus:ring-cyan-500"
            />
          </div>

          <!-- Section: Online Accounts -->
          <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
            
            <!-- Chess.com Configuration Card -->
            <div class="p-4 rounded-xl bg-zinc-950/50 border border-zinc-800 space-y-3">
              <div class="flex items-center justify-between">
                <div class="flex items-center gap-2">
                  <div class="w-2.5 h-2.5 rounded-full bg-emerald-400" />
                  <span class="font-medium text-xs text-zinc-200">Chess.com Account</span>
                </div>
                <span class="text-[10px] text-zinc-500">Public Archives</span>
              </div>

              <div>
                <label class="text-[11px] text-zinc-400 block mb-1">Username</label>
                <div class="flex items-center gap-2">
                  <input
                    v-model="currentProfile.chessComUsername"
                    type="text"
                    placeholder="e.g. MagnusCarlsen"
                    class="flex-1 px-3 py-1.5 bg-zinc-900 border border-zinc-700 rounded-lg text-xs text-zinc-100 placeholder-zinc-600 focus:outline-none focus:border-emerald-500"
                  />
                  <button
                    class="px-2.5 py-1.5 text-xs bg-zinc-800 hover:bg-zinc-700 text-zinc-300 rounded-lg transition-colors cursor-pointer shrink-0 disabled:opacity-50"
                    :disabled="testingChessCom || !currentProfile.chessComUsername.trim()"
                    @click="testChessCom"
                  >
                    <RefreshCw v-if="testingChessCom" class="w-3.5 h-3.5 animate-spin" />
                    <span v-else>Verify</span>
                  </button>
                </div>
              </div>

              <div v-if="chessComArchivesCount !== null" class="text-[11px] text-emerald-400 flex items-center gap-1.5">
                <Check class="w-3.5 h-3.5" />
                <span>Found {{ chessComArchivesCount }} monthly archives</span>
              </div>
            </div>

            <!-- Lichess.org Configuration Card -->
            <div class="p-4 rounded-xl bg-zinc-950/50 border border-zinc-800 space-y-3">
              <div class="flex items-center justify-between">
                <div class="flex items-center gap-2">
                  <div class="w-2.5 h-2.5 rounded-full bg-blue-400" />
                  <span class="font-medium text-xs text-zinc-200">Lichess.org Account</span>
                </div>
                <span class="text-[10px] text-zinc-500">ND-PGN Streaming</span>
              </div>

              <div>
                <label class="text-[11px] text-zinc-400 block mb-1">Username</label>
                <div class="flex items-center gap-2">
                  <input
                    v-model="currentProfile.lichessUsername"
                    type="text"
                    placeholder="e.g. DrNykterstein"
                    class="flex-1 px-3 py-1.5 bg-zinc-900 border border-zinc-700 rounded-lg text-xs text-zinc-100 placeholder-zinc-600 focus:outline-none focus:border-blue-500"
                  />
                  <button
                    class="px-2.5 py-1.5 text-xs bg-zinc-800 hover:bg-zinc-700 text-zinc-300 rounded-lg transition-colors cursor-pointer shrink-0 disabled:opacity-50"
                    :disabled="testingLichess || !currentProfile.lichessUsername.trim()"
                    @click="testLichess"
                  >
                    <RefreshCw v-if="testingLichess" class="w-3.5 h-3.5 animate-spin" />
                    <span v-else>Verify</span>
                  </button>
                </div>
              </div>

              <div>
                <label class="text-[11px] text-zinc-400 block mb-1">
                  Personal API Token <span class="text-zinc-500 font-normal">(Optional, for private games)</span>
                </label>
                <div class="relative">
                  <input
                    v-model="currentProfile.lichessToken"
                    :type="showToken ? 'text' : 'password'"
                    placeholder="lip_..."
                    class="w-full pl-3 pr-8 py-1.5 bg-zinc-900 border border-zinc-700 rounded-lg text-xs text-zinc-100 placeholder-zinc-600 focus:outline-none focus:border-blue-500"
                  />
                  <button
                    type="button"
                    class="absolute right-2 top-1/2 -translate-y-1/2 text-zinc-400 hover:text-zinc-200 cursor-pointer"
                    @click="showToken = !showToken"
                  >
                    <EyeOff v-if="showToken" class="w-3.5 h-3.5" />
                    <Eye v-else class="w-3.5 h-3.5" />
                  </button>
                </div>
              </div>

              <div v-if="lichessPreviewResult" class="text-[11px] text-blue-400 flex items-center gap-1.5">
                <Check class="w-3.5 h-3.5" />
                <span>{{ lichessPreviewResult }}</span>
              </div>
            </div>

          </div>

          <!-- Section: Target Database Destination -->
          <div class="p-4 rounded-xl bg-zinc-950/50 border border-zinc-800 space-y-3">
            <span class="text-xs font-semibold text-zinc-300 flex items-center gap-2">
              <Database class="w-3.5 h-3.5 text-cyan-400" />
              Target Scid Database Merge Destination
            </span>

            <div class="space-y-2">
              <!-- Option 1: Active Database -->
              <label class="flex items-start gap-3 p-2.5 rounded-lg border border-zinc-800 hover:bg-zinc-900/60 cursor-pointer">
                <input
                  v-model="targetMode"
                  type="radio"
                  value="active"
                  class="mt-1 accent-cyan-500"
                />
                <div>
                  <div class="text-xs font-medium text-zinc-200">
                    Currently Open Database
                  </div>
                  <div class="text-[11px] text-zinc-400 mt-0.5">
                    Merge downloaded games into the active session database:
                    <span v-if="props.activeDbName" class="text-cyan-300 font-mono font-medium ml-1">
                      {{ props.activeDbName }}
                    </span>
                    <span v-else class="text-amber-400 ml-1">
                      (No database currently open)
                    </span>
                  </div>
                </div>
              </label>

              <!-- Option 2: Custom Database File -->
              <label class="flex items-start gap-3 p-2.5 rounded-lg border border-zinc-800 hover:bg-zinc-900/60 cursor-pointer">
                <input
                  v-model="targetMode"
                  type="radio"
                  value="custom"
                  class="mt-1 accent-cyan-500"
                />
                <div class="flex-1">
                  <div class="text-xs font-medium text-zinc-200">
                    Specific Local Scid Database (.si4)
                  </div>
                  <div class="text-[11px] text-zinc-400 mt-0.5 mb-2">
                    Specify or browse for a Scid database. (A new database will be created automatically if it does not exist yet).
                  </div>

                  <div v-if="targetMode === 'custom'" class="flex items-center gap-2">
                    <input
                      v-model="currentProfile.targetDatabase"
                      type="text"
                      placeholder="/path/to/my_online_games"
                      class="flex-1 px-3 py-1.5 bg-zinc-900 border border-zinc-700 rounded-lg text-xs text-zinc-100 placeholder-zinc-600 focus:outline-none focus:border-cyan-500 font-mono"
                    />
                    <button
                      class="px-3 py-1.5 text-xs bg-zinc-800 hover:bg-zinc-700 text-zinc-200 rounded-lg flex items-center gap-1.5 transition-colors cursor-pointer shrink-0"
                      @click.prevent="handleBrowseTargetDB"
                    >
                      <FolderOpen class="w-3.5 h-3.5 text-zinc-400" />
                      <span>Browse...</span>
                    </button>
                  </div>
                </div>
              </label>
            </div>
          </div>

          <!-- Section: Sync Preferences -->
          <div class="p-4 rounded-xl bg-zinc-950/50 border border-zinc-800 space-y-3">
            <span class="text-xs font-semibold text-zinc-300">Sync & Filter Preferences</span>

            <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
              <label class="flex items-center gap-2.5 p-2 rounded-lg hover:bg-zinc-900/60 cursor-pointer">
                <input
                  v-model="currentProfile.autoDeduplicate"
                  type="checkbox"
                  class="rounded accent-cyan-500"
                />
                <div>
                  <span class="text-xs text-zinc-200 font-medium block">Automatic Deduplication</span>
                  <span class="text-[10px] text-zinc-400">Silently skip games already present in the database</span>
                </div>
              </label>

              <label class="flex items-center gap-2.5 p-2 rounded-lg hover:bg-zinc-900/60 cursor-pointer">
                <input
                  v-model="currentProfile.ratedOnly"
                  type="checkbox"
                  class="rounded accent-cyan-500"
                />
                <div>
                  <span class="text-xs text-zinc-200 font-medium block">Rated Games Only</span>
                  <span class="text-[10px] text-zinc-400">Ignore unrated / casual friendly games</span>
                </div>
              </label>
            </div>

            <!-- Incremental Sync Note -->
            <div class="p-2.5 rounded-lg bg-zinc-900/60 border border-zinc-800/80 text-[11px] text-zinc-400 flex items-center gap-2">
              <Calendar class="w-4 h-4 text-cyan-400 shrink-0" />
              <span>
                <strong>Smart Incremental Sync:</strong>
                Subsequent syncs only download games played after the last sync timestamp
                <span v-if="currentProfile.lastSyncTime > 0" class="text-cyan-300 font-medium">
                  ({{ formatDate(currentProfile.lastSyncTime) }})
                </span>.
              </span>
            </div>
          </div>

          <!-- Section: Live Sync Results Card -->
          <div
            v-if="lastSyncResult"
            class="p-4 rounded-xl bg-zinc-950/70 border border-cyan-800/50 space-y-3"
          >
            <div class="flex items-center justify-between">
              <span class="text-xs font-semibold text-white flex items-center gap-1.5">
                <CheckCircle2 class="w-4 h-4 text-emerald-400" />
                Latest Sync Summary
              </span>
              <span class="text-[11px] text-zinc-400 font-mono">
                Duration: {{ lastSyncResult.durationMs }}ms
              </span>
            </div>

            <div class="grid grid-cols-2 sm:grid-cols-5 gap-2">
              <div class="p-2.5 rounded-lg bg-zinc-900/80 border border-zinc-800 text-center">
                <div class="text-[10px] text-zinc-400">Downloaded</div>
                <div class="text-base font-semibold text-white font-mono mt-0.5">
                  {{ lastSyncResult.totalDownloaded }}
                </div>
              </div>
              <div class="p-2.5 rounded-lg bg-zinc-900/80 border border-zinc-800 text-center">
                <div class="text-[10px] text-zinc-400">Lichess</div>
                <div class="text-base font-semibold text-blue-300 font-mono mt-0.5">
                  {{ lastSyncResult.lichessGames }}
                </div>
              </div>
              <div class="p-2.5 rounded-lg bg-zinc-900/80 border border-zinc-800 text-center">
                <div class="text-[10px] text-zinc-400">Chess.com</div>
                <div class="text-base font-semibold text-emerald-300 font-mono mt-0.5">
                  {{ lastSyncResult.chessComGames }}
                </div>
              </div>
              <div class="p-2.5 rounded-lg bg-zinc-900/80 border border-zinc-800 text-center">
                <div class="text-[10px] text-zinc-400">Imported</div>
                <div class="text-base font-semibold text-cyan-300 font-mono mt-0.5">
                  {{ lastSyncResult.imported }}
                </div>
              </div>
              <div class="p-2.5 rounded-lg bg-zinc-900/80 border border-zinc-800 text-center">
                <div class="text-[10px] text-zinc-400">Deduplicated</div>
                <div class="text-base font-semibold text-amber-300 font-mono mt-0.5">
                  {{ lastSyncResult.duplicatesSkipped }}
                </div>
              </div>
            </div>
          </div>

        </div>

      </div>

      <!-- Tab 2: Import Another User / Opponent Games -->
      <div v-else-if="activeTab === 'import-opponent'" class="flex-1 overflow-y-auto p-6 space-y-5">
        <div class="p-4 rounded-xl bg-purple-950/30 border border-purple-800/40 flex items-start gap-3">
          <Globe class="w-5 h-5 text-purple-400 shrink-0 mt-0.5" />
          <div class="text-xs space-y-1">
            <h4 class="font-bold text-white text-sm">Download & Merge Any Player's Online Games</h4>
            <p class="text-zinc-300 leading-relaxed">
              Study any opponent, club rival, coach, or grandmaster. Choose to merge their games into your currently active database, maintain an isolated dedicated database, or maintain both simultaneously (blended database).
            </p>
          </div>
        </div>

        <!-- Form: Player Identity -->
        <div class="grid grid-cols-1 md:grid-cols-3 gap-3">
          <div>
            <label class="text-xs font-semibold text-zinc-300 block mb-1">Player Display Name</label>
            <input
              v-model="opponentForm.playerName"
              type="text"
              placeholder="e.g. Hikaru Nakamura, RivalSmith"
              class="w-full px-3 py-2 bg-zinc-950 border border-zinc-700 rounded-xl text-xs text-zinc-100 placeholder-zinc-500 focus:outline-none focus:ring-2 focus:ring-purple-500/50"
            />
          </div>

          <div>
            <label class="text-xs font-semibold text-zinc-300 block mb-1">Chess.com Username</label>
            <input
              v-model="opponentForm.chessComUsername"
              type="text"
              placeholder="e.g. hikaru"
              class="w-full px-3 py-2 bg-zinc-950 border border-zinc-700 rounded-xl text-xs text-zinc-100 placeholder-zinc-500 focus:outline-none focus:ring-2 focus:ring-purple-500/50"
            />
          </div>

          <div>
            <label class="text-xs font-semibold text-zinc-300 block mb-1">Lichess.org Username</label>
            <input
              v-model="opponentForm.lichessUsername"
              type="text"
              placeholder="e.g. drnykterstein"
              class="w-full px-3 py-2 bg-zinc-950 border border-zinc-700 rounded-xl text-xs text-zinc-100 placeholder-zinc-500 focus:outline-none focus:ring-2 focus:ring-purple-500/50"
            />
          </div>
        </div>

        <!-- Destination Strategy Selector -->
        <div class="space-y-3">
          <label class="text-xs font-bold text-zinc-200 block uppercase tracking-wider">
            Database Destination & Merge Strategy
          </label>

          <div class="grid grid-cols-1 md:grid-cols-3 gap-3">
            <!-- Mode 1: Active DB -->
            <label
              class="p-3.5 rounded-xl border cursor-pointer transition-all flex flex-col justify-between"
              :class="opponentForm.targetMode === 'active'
                ? 'bg-purple-950/40 border-purple-500 shadow-md ring-1 ring-purple-500/30'
                : 'bg-zinc-950 border-zinc-800 hover:border-zinc-700'"
            >
              <div class="flex items-start gap-2.5">
                <input
                  v-model="opponentForm.targetMode"
                  type="radio"
                  value="active"
                  class="mt-1 accent-purple-500"
                />
                <div>
                  <span class="text-xs font-bold text-white block">Blend into Active DB</span>
                  <p class="text-[11px] text-zinc-400 mt-1">
                    Appends games directly into:
                    <span class="text-purple-300 font-mono block mt-0.5 truncate">{{ props.activeDbName || 'Current open database' }}</span>
                  </p>
                </div>
              </div>
            </label>

            <!-- Mode 2: Dedicated DB -->
            <label
              class="p-3.5 rounded-xl border cursor-pointer transition-all flex flex-col justify-between"
              :class="opponentForm.targetMode === 'dedicated'
                ? 'bg-purple-950/40 border-purple-500 shadow-md ring-1 ring-purple-500/30'
                : 'bg-zinc-950 border-zinc-800 hover:border-zinc-700'"
            >
              <div class="flex items-start gap-2.5">
                <input
                  v-model="opponentForm.targetMode"
                  type="radio"
                  value="dedicated"
                  class="mt-1 accent-purple-500"
                />
                <div>
                  <span class="text-xs font-bold text-white block">Dedicated Standalone DB</span>
                  <p class="text-[11px] text-zinc-400 mt-1">
                    Stores the player's games in their own isolated database file (.si4).
                  </p>
                </div>
              </div>
            </label>

            <!-- Mode 3: Blended DB (Both) -->
            <label
              class="p-3.5 rounded-xl border cursor-pointer transition-all flex flex-col justify-between"
              :class="opponentForm.targetMode === 'blended'
                ? 'bg-purple-950/40 border-purple-500 shadow-md ring-1 ring-purple-500/30'
                : 'bg-zinc-950 border-zinc-800 hover:border-zinc-700'"
            >
              <div class="flex items-start gap-2.5">
                <input
                  v-model="opponentForm.targetMode"
                  type="radio"
                  value="blended"
                  class="mt-1 accent-purple-500"
                />
                <div>
                  <span class="text-xs font-bold text-white block">Dedicated + Blended Master</span>
                  <p class="text-[11px] text-zinc-400 mt-1">
                    Maintains their separate base AND merges their games into your master database.
                  </p>
                </div>
              </div>
            </label>
          </div>

          <!-- File pickers for dedicated and blended paths -->
          <div v-if="opponentForm.targetMode === 'dedicated' || opponentForm.targetMode === 'blended'" class="space-y-2 pt-2">
            <div>
              <label class="text-xs text-zinc-400 block mb-1">Dedicated Database Path (.si4):</label>
              <div class="flex items-center gap-2">
                <input
                  v-model="opponentForm.dedicatedPath"
                  type="text"
                  placeholder="e.g. /home/user/ChessDataBases/Hikaru_Games.si4"
                  class="flex-1 px-3 py-1.5 bg-zinc-950 border border-zinc-700 rounded-lg text-xs text-zinc-100 placeholder-zinc-600 focus:outline-none focus:ring-1 focus:ring-purple-500"
                />
                <button
                  type="button"
                  class="px-3 py-1.5 bg-zinc-800 hover:bg-zinc-700 text-zinc-200 rounded-lg text-xs flex items-center gap-1.5 transition-colors cursor-pointer"
                  @click="pickDedicatedPath"
                >
                  <FolderOpen class="w-3.5 h-3.5 text-purple-400" />
                  <span>Browse</span>
                </button>
              </div>
            </div>

            <div v-if="opponentForm.targetMode === 'blended'">
              <label class="text-xs text-zinc-400 block mb-1">Master Blended Database Path (leave empty to use active database):</label>
              <div class="flex items-center gap-2">
                <input
                  v-model="opponentForm.blendedPath"
                  type="text"
                  :placeholder="props.activeDbPath || '/home/user/ChessDataBases/Master.si4'"
                  class="flex-1 px-3 py-1.5 bg-zinc-950 border border-zinc-700 rounded-lg text-xs text-zinc-100 placeholder-zinc-600 focus:outline-none focus:ring-1 focus:ring-purple-500"
                />
                <button
                  type="button"
                  class="px-3 py-1.5 bg-zinc-800 hover:bg-zinc-700 text-zinc-200 rounded-lg text-xs flex items-center gap-1.5 transition-colors cursor-pointer"
                  @click="pickBlendedPath"
                >
                  <FolderOpen class="w-3.5 h-3.5 text-purple-400" />
                  <span>Browse</span>
                </button>
              </div>
            </div>
          </div>
        </div>

        <!-- Preferences Checkboxes & Limit -->
        <div class="p-3.5 rounded-xl bg-zinc-950 border border-zinc-800 grid grid-cols-1 md:grid-cols-4 gap-3 text-xs items-center">
          <label class="flex items-center gap-2 cursor-pointer">
            <input v-model="opponentForm.autoDeduplicate" type="checkbox" class="rounded accent-purple-500" />
            <span class="text-zinc-300">Auto-deduplicate</span>
          </label>
          <label class="flex items-center gap-2 cursor-pointer">
            <input v-model="opponentForm.ratedOnly" type="checkbox" class="rounded accent-purple-500" />
            <span class="text-zinc-300">Rated only</span>
          </label>
          <label class="flex items-center gap-2 cursor-pointer">
            <input v-model="opponentForm.createProfile" type="checkbox" class="rounded accent-purple-500" />
            <span class="text-zinc-300">Save as study profile</span>
          </label>
          <div class="flex items-center gap-2">
            <span class="text-zinc-400">Limit:</span>
            <select v-model="opponentForm.maxGames" class="bg-zinc-900 border border-zinc-700/80 rounded px-2 py-1 text-xs text-zinc-200 font-mono">
              <option :value="100">100 games</option>
              <option :value="200">200 (Rec.)</option>
              <option :value="500">500 games</option>
              <option :value="1000">1,000 games</option>
              <option :value="0">All games</option>
            </select>
          </div>
        </div>

        <!-- Import Result Card -->
        <div v-if="opponentImportResult" class="p-4 rounded-xl bg-purple-950/40 border border-purple-500/50 space-y-2">
          <div class="flex items-center justify-between text-xs">
            <span class="font-bold text-white flex items-center gap-1.5">
              <CheckCircle2 class="w-4 h-4 text-emerald-400" />
              Import Complete for {{ opponentImportResult.profileName }}
            </span>
            <span class="text-zinc-400 font-mono">{{ opponentImportResult.durationMs }}ms</span>
          </div>
          <div class="grid grid-cols-4 gap-2 text-center text-xs">
            <div class="p-2 bg-zinc-900 rounded-lg"><div class="font-bold text-white font-mono">{{ opponentImportResult.totalDownloaded }}</div><div class="text-[10px] text-zinc-400">Downloaded</div></div>
            <div class="p-2 bg-zinc-900 rounded-lg"><div class="font-bold text-cyan-300 font-mono">{{ opponentImportResult.imported }}</div><div class="text-[10px] text-zinc-400">Imported</div></div>
            <div class="p-2 bg-zinc-900 rounded-lg"><div class="font-bold text-amber-300 font-mono">{{ opponentImportResult.duplicatesSkipped }}</div><div class="text-[10px] text-zinc-400">Deduplicated</div></div>
            <div class="p-2 bg-zinc-900 rounded-lg"><div class="font-bold text-blue-300 font-mono">{{ opponentImportResult.lichessGames + opponentImportResult.chessComGames }}</div><div class="text-[10px] text-zinc-400">Online Total</div></div>
          </div>
        </div>
      </div>

      <!-- Modal Footer Action Bar -->
      <div class="px-6 py-4 border-t border-zinc-800 bg-zinc-950/80 flex items-center justify-between">
        <!-- Profiles footer buttons -->
        <template v-if="activeTab === 'profiles'">
          <div>
            <button
              v-if="currentProfile.id"
              class="px-3 py-2 text-xs rounded-xl bg-red-950/40 hover:bg-red-900/60 text-red-300 border border-red-800/40 flex items-center gap-1.5 transition-colors cursor-pointer"
              :disabled="isSyncing || isSaving"
              @click="handleDeleteProfile"
            >
              <Trash2 class="w-3.5 h-3.5" />
              <span>Delete Profile</span>
            </button>
          </div>

          <div class="flex items-center gap-3">
            <button
              class="px-4 py-2 text-xs rounded-xl bg-zinc-800 hover:bg-zinc-700 text-zinc-200 border border-zinc-700 flex items-center gap-1.5 transition-colors cursor-pointer"
              :disabled="isSyncing || isSaving"
              @click="handleSaveProfile"
            >
              <Save class="w-3.5 h-3.5" />
              <span>{{ isSaving ? 'Saving...' : 'Save Profile' }}</span>
            </button>

            <button
              class="px-5 py-2 text-xs font-semibold rounded-xl bg-cyan-600 hover:bg-cyan-500 active:bg-cyan-700 text-white flex items-center gap-2 transition-all shadow-lg shadow-cyan-950/50 cursor-pointer disabled:opacity-50"
              :disabled="isSyncing || isSaving"
              @click="handleSyncNow"
            >
              <RefreshCw v-if="isSyncing" class="w-4 h-4 animate-spin" />
              <Globe v-else class="w-4 h-4" />
              <span>{{ isSyncing ? (syncProgressText || 'Syncing...') : 'Sync & Merge Now' }}</span>
            </button>
          </div>
        </template>

        <!-- Import Opponent footer buttons -->
        <template v-else-if="activeTab === 'import-opponent'">
          <div class="text-xs text-zinc-400">
            <span>Destination: </span>
            <span class="text-purple-300 font-semibold uppercase">{{ opponentForm.targetMode }} mode</span>
          </div>

          <div class="flex items-center gap-3">
            <button
              class="px-4 py-2 text-xs rounded-xl bg-zinc-800 hover:bg-zinc-700 text-zinc-200 border border-zinc-700 transition-colors cursor-pointer"
              @click="emit('close')"
            >
              Close
            </button>

            <button
              class="px-5 py-2 text-xs font-bold rounded-xl bg-gradient-to-r from-purple-600 to-indigo-600 hover:from-purple-500 hover:to-indigo-500 text-white flex items-center gap-2 transition-all shadow-lg shadow-purple-950/50 cursor-pointer disabled:opacity-50"
              :disabled="isImportingOpponent"
              @click="handleImportOpponentGames"
            >
              <RefreshCw v-if="isImportingOpponent" class="w-4 h-4 animate-spin" />
              <Globe v-else class="w-4 h-4" />
              <span>{{ isImportingOpponent ? 'Downloading & Merging...' : 'Download & Merge Player Games' }}</span>
            </button>
          </div>
        </template>
      </div>

    </div>
  </div>
</template>
