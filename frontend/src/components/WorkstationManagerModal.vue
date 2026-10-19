<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import {
  Database,
  HardDrive,
  RotateCcw,
  FolderOpen,
  Copy,
  Check,
  Lock,
  Unlock,
  RefreshCw,
  Filter,
  UserPlus,
  Layers,
  AlertTriangle,
  CheckCircle2,
  X,
  Upload,
  ShieldAlert,
  Archive,
  Play
} from '@lucide/vue'
import {
  BuildFocusedDatabase,
  GetWorkstationStorageReport,
  ImportAssetToWorkstation,
  SelectAndImportWorkstationAsset,
  FactoryResetNewScid,
  GetNotablePlayersDropdown,
  SetDatabaseWritePolicy,
  CancelActiveOperation,
} from '../../wailsjs/go/main/App'
import { ClipboardSetText } from '../../wailsjs/runtime/runtime'

export interface DiscoveredDBItem {
  name: string
  basePath: string
  type: string
  category: string
  numGames: number
  description: string
  sizeStr: string
  writePolicy?: string
  isLocked?: boolean
}

export interface NotablePlayerItem {
  name: string
  title: string
  era: string
  aliases: string[]
  matchedNames?: string[]
  gameCount: number
  peakElo?: number
  yearStart?: number
  yearEnd?: number
  isCustom?: boolean
}

export interface StoredAssetInfo {
  name: string
  path: string
  category: string
  sizeBytes: number
  sizeFormatted: string
  modifiedAt: string
}

export interface CategoryStorageInfo {
  categoryId: number
  key: string
  label: string
  path: string
  description: string
  fileCount: number
  totalBytes: number
  sizeFormatted: string
  assets: StoredAssetInfo[]
}

export interface StorageReport {
  dataRootDir: string
  configRootDir: string
  totalBytes: number
  totalSizeStr: string
  totalFiles: number
  categories: CategoryStorageInfo[]
}

export interface SubDatabaseResult {
  targetBasePath: string
  targetName: string
  sourcesScanned: number
  totalGamesScanned: number
  gamesMatched: number
  duplicatesSkipped: number
}

export interface FactoryResetResult {
  resetCompletedAt: string
  backupCreated: boolean
  backupArchivePath?: string
  wipedPaths: string[]
  filesRemoved: number
  bytesFreed: number
  sizeFreedStr: string
  message: string
}

const props = defineProps<{
  initialTab?: 'builder' | 'storage' | 'reset'
  activeDbPath?: string
  activeDbName?: string
  discoveredDatabases?: Array<DiscoveredDBItem>
  initialFilter?: {
    player?: string
    players?: string[]
    eco?: string
    openingMoves?: string[]
  }
  currentOpeningMoves?: string[]
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'openDatabase', path: string): void
  (e: 'refreshDatabases'): void
  (e: 'factoryResetCompleted'): void
}>()

type TabType = 'builder' | 'storage' | 'reset'
const activeTab = ref<TabType>(props.initialTab || 'builder')

// Global messages
const loading = ref(false)
const statusMessage = ref('')
const errorMessage = ref('')

// ==========================================
// Tab 1: Multi-Database Sub-Database Builder
// ==========================================
const localPolicies = ref<Record<string, string>>({})

const allAvailableDatabases = computed<DiscoveredDBItem[]>(() => {
  const list: DiscoveredDBItem[] = []
  const seen = new Set<string>()

  if (props.discoveredDatabases) {
    for (const db of props.discoveredDatabases) {
      if (!db.basePath) continue
      seen.add(db.basePath)
      list.push({
        ...db,
        writePolicy: localPolicies.value[db.basePath] || db.writePolicy || (db.isLocked ? 'locked' : 'editable'),
      })
    }
  }

  if (props.activeDbPath && !seen.has(props.activeDbPath)) {
    list.unshift({
      name: props.activeDbName || props.activeDbPath.split('/').pop() || 'Active Database',
      basePath: props.activeDbPath,
      type: 'Scid',
      category: 'User Databases',
      numGames: 0,
      description: 'Currently active database',
      sizeStr: '',
      writePolicy: localPolicies.value[props.activeDbPath] || 'editable',
    })
  }

  return list
})

const selectedSources = ref<string[]>([])
const targetName = ref('Focused_Study_DB')
const targetDescription = ref('Focused sub-database built across multiple databases')
const playersInput = ref('')
const playerColor = ref<'any' | 'white' | 'black'>('any')
const ecoFilter = ref('')
const openingMovesInput = ref('')
const minElo = ref<number>(0)
const yearFrom = ref<number>(0)
const yearTo = ref<number>(0)
const resultFilter = ref<string>('')
const deduplicate = ref<boolean>(true)
const maxGames = ref<number>(0)

const notablePlayers = ref<NotablePlayerItem[]>([])
const buildResult = ref<SubDatabaseResult | null>(null)

const ecoPresets = [
  { label: 'Sicilian B20-B99', code: 'B20-B99' },
  { label: 'Ruy Lopez C60-C99', code: 'C60-C99' },
  { label: "Queen's Gambit D06-D69", code: 'D06-D69' },
  { label: "King's Indian E60-E99", code: 'E60-E99' },
  { label: 'French C00-C19', code: 'C00-C19' },
  { label: 'Caro-Kann B10-B19', code: 'B10-B19' },
]

function initializeFiltersFromProps() {
  if (props.activeDbPath) {
    selectedSources.value = [props.activeDbPath]
  } else if (allAvailableDatabases.value.length > 0) {
    selectedSources.value = [allAvailableDatabases.value[0].basePath]
  }

  if (props.initialFilter) {
    const pList: string[] = []
    if (props.initialFilter.player) {
      pList.push(props.initialFilter.player)
    }
    if (props.initialFilter.players?.length) {
      for (const p of props.initialFilter.players) {
        if (!pList.includes(p)) pList.push(p)
      }
    }
    if (pList.length > 0) {
      playersInput.value = pList.join(', ')
    }
    if (props.initialFilter.eco) {
      ecoFilter.value = props.initialFilter.eco
    }
    if (props.initialFilter.openingMoves?.length) {
      openingMovesInput.value = props.initialFilter.openingMoves.join(' ')
    }
  }
}

function toggleSourceDb(basePath: string) {
  const idx = selectedSources.value.indexOf(basePath)
  if (idx >= 0) {
    selectedSources.value.splice(idx, 1)
  } else {
    selectedSources.value.push(basePath)
  }
}

function selectAllUserDbs() {
  const paths = allAvailableDatabases.value
    .filter((db) => db.category !== 'Tutorial & Tactics')
    .map((db) => db.basePath)
  selectedSources.value = paths.length > 0 ? paths : allAvailableDatabases.value.map((d) => d.basePath)
}

function selectActiveDbOnly() {
  if (props.activeDbPath) {
    selectedSources.value = [props.activeDbPath]
  } else if (allAvailableDatabases.value.length > 0) {
    selectedSources.value = [allAvailableDatabases.value[0].basePath]
  }
}

function formatWritePolicyLabel(policy?: string): string {
  switch (policy) {
    case 'locked':
      return 'Locked'
    case 'sync_only':
      return 'Sync-Only'
    default:
      return 'Editable'
  }
}

function nextWritePolicy(current?: string): string {
  if (current === 'editable' || !current) return 'sync_only'
  if (current === 'sync_only') return 'locked'
  return 'editable'
}

async function handleToggleWritePolicy(db: DiscoveredDBItem) {
  errorMessage.value = ''
  const next = nextWritePolicy(db.writePolicy)
  try {
    await SetDatabaseWritePolicy(db.basePath, next)
    localPolicies.value[db.basePath] = next
    statusMessage.value = `Updated write policy for ${db.name} to ${formatWritePolicyLabel(next)}`
    emit('refreshDatabases')
  } catch (err: any) {
    errorMessage.value = err?.message || String(err)
  }
}

function addPlayerChip(player: NotablePlayerItem) {
  const current = playersInput.value
    .split(',')
    .map((s) => s.trim())
    .filter(Boolean)

  if (!current.includes(player.name)) {
    current.push(player.name)
  }
  playersInput.value = current.join(', ')
}

function applyEcoPreset(code: string) {
  ecoFilter.value = code
}

function useMovesOnBoard() {
  if (props.currentOpeningMoves && props.currentOpeningMoves.length > 0) {
    openingMovesInput.value = props.currentOpeningMoves.join(' ')
  }
}

async function loadNotablePlayers() {
  try {
    const res = await GetNotablePlayersDropdown()
    if (Array.isArray(res)) {
      notablePlayers.value = res
    }
  } catch {
    // Ignore errors when no DB is open
  }
}

async function handleBuildSubDatabase() {
  errorMessage.value = ''
  statusMessage.value = ''
  buildResult.value = null

  if (!targetName.value.trim()) {
    errorMessage.value = 'Please provide a target database name.'
    return
  }

  loading.value = true
  try {
    const players = playersInput.value
      .split(',')
      .map((s) => s.trim())
      .filter(Boolean)

    const openingMoves = openingMovesInput.value
      .trim()
      .split(/\s+/)
      .filter(Boolean)

    const filterPayload = {
      targetName: targetName.value.trim(),
      targetDescription: targetDescription.value.trim(),
      sourceBasePaths: [...selectedSources.value],
      players,
      playerColor: playerColor.value,
      ecoFilter: ecoFilter.value.trim(),
      openingMoves,
      minElo: Number(minElo.value) || 0,
      yearFrom: Number(yearFrom.value) || 0,
      yearTo: Number(yearTo.value) || 0,
      resultFilter: resultFilter.value,
      deduplicate: deduplicate.value,
      maxGames: Number(maxGames.value) || 0,
    }

    const res = await BuildFocusedDatabase(filterPayload)
    buildResult.value = res
    statusMessage.value = `Built focused database "${res.targetName}" (${res.gamesMatched} games matched).`
    emit('refreshDatabases')
  } catch (err: any) {
    errorMessage.value = err?.message || String(err)
  } finally {
    loading.value = false
  }
}

function handleOpenBuiltDatabase() {
  if (buildResult.value?.targetBasePath) {
    emit('openDatabase', buildResult.value.targetBasePath)
  }
}

// ==========================================
// Tab 2: Managed Storage & Asset Importer
// ==========================================
const storageReport = ref<StorageReport | null>(null)
const customAssetPaths = ref<Record<string, string>>({})
const copiedPathKey = ref<string>('')
const importedAssetNotice = ref<StoredAssetInfo | null>(null)

async function loadStorageReport() {
  errorMessage.value = ''
  loading.value = true
  try {
    const rep = await GetWorkstationStorageReport()
    storageReport.value = rep
  } catch (err: any) {
    errorMessage.value = err?.message || String(err)
  } finally {
    loading.value = false
  }
}

async function copyDirectoryPath(key: string, path: string) {
  try {
    await ClipboardSetText(path)
    copiedPathKey.value = key
    setTimeout(() => {
      if (copiedPathKey.value === key) copiedPathKey.value = ''
    }, 2000)
  } catch {
    // Fallback ignore
  }
}

async function handleSelectAndImportAsset(categoryKey: string) {
  errorMessage.value = ''
  statusMessage.value = ''
  loading.value = true
  try {
    const info = await SelectAndImportWorkstationAsset(categoryKey)
    if (info) {
      importedAssetNotice.value = info
      statusMessage.value = `Copied ${info.name} (${info.sizeFormatted}) into ${categoryKey}.`
      await loadStorageReport()
      if (categoryKey === 'databases') {
        emit('refreshDatabases')
      }
    }
  } catch (err: any) {
    errorMessage.value = err?.message || String(err)
  } finally {
    loading.value = false
  }
}

async function handleCopyCustomPathIntoNewScid(categoryKey: string) {
  const srcPath = (customAssetPaths.value[categoryKey] || '').trim()
  if (!srcPath) {
    errorMessage.value = 'Please enter a source file path to copy into NewScid.'
    return
  }

  errorMessage.value = ''
  statusMessage.value = ''
  loading.value = true
  try {
    const info = await ImportAssetToWorkstation(categoryKey, srcPath)
    if (info) {
      importedAssetNotice.value = info
      customAssetPaths.value[categoryKey] = ''
      statusMessage.value = `Copied ${info.name} (${info.sizeFormatted}) into ${categoryKey}.`
      await loadStorageReport()
      if (categoryKey === 'databases') {
        emit('refreshDatabases')
      }
    }
  } catch (err: any) {
    errorMessage.value = err?.message || String(err)
  } finally {
    loading.value = false
  }
}

// ==========================================
// Tab 3: Factory Reset (with Opt-In Backup)
// ==========================================
const createOptInBackup = ref<boolean>(false) // MUST default to false (opt-in!)
const keepEngines = ref<boolean>(false)
const keepBooks = ref<boolean>(false)
const factoryResetResult = ref<FactoryResetResult | null>(null)

async function handleFactoryReset() {
  errorMessage.value = ''
  statusMessage.value = ''
  factoryResetResult.value = null
  loading.value = true

  try {
    const res = await FactoryResetNewScid({
      createOptInBackup: createOptInBackup.value,
      keepEngines: keepEngines.value,
      keepBooks: keepBooks.value,
    })
    factoryResetResult.value = res
    statusMessage.value = res?.message || 'Factory reset completed.'
    emit('factoryResetCompleted')
    emit('refreshDatabases')
  } catch (err: any) {
    errorMessage.value = err?.message || String(err)
  } finally {
    loading.value = false
  }
}

function switchTab(tab: TabType) {
  activeTab.value = tab
  errorMessage.value = ''
  statusMessage.value = ''
  if (tab === 'storage') {
    loadStorageReport()
  }
}

watch(
  () => props.initialTab,
  (newTab) => {
    if (newTab) {
      switchTab(newTab)
    }
  }
)

onMounted(async () => {
  initializeFiltersFromProps()
  await loadNotablePlayers()
  if (activeTab.value === 'storage') {
    await loadStorageReport()
  }
})
</script>

<template>
  <div class="modal-backdrop" @click.self="emit('close')">
    <div class="modal-container" data-testid="workstation-manager-modal">
      <!-- Header -->
      <div class="modal-header">
        <div class="header-title">
          <HardDrive :size="20" class="header-icon" />
          <div>
            <h2>Workstation Manager, Focused Sub-DB Builder &amp; Storage</h2>
            <p class="header-subtitle">
              Build focused multi-database studies, inspect managed directories &amp; assets, or reset to factory settings
            </p>
          </div>
        </div>
        <button class="close-btn" data-testid="close-modal-btn" @click="emit('close')" title="Close">
          <X :size="18" />
        </button>
      </div>

      <!-- Navigation Tabs -->
      <div class="modal-tabs">
        <button
          class="tab-btn"
          :class="{ active: activeTab === 'builder' }"
          data-testid="tab-builder"
          @click="switchTab('builder')"
        >
          <Filter :size="15" />
          <span>Multi-DB Sub-Database Builder</span>
        </button>
        <button
          class="tab-btn"
          :class="{ active: activeTab === 'storage' }"
          data-testid="tab-storage"
          @click="switchTab('storage')"
        >
          <FolderOpen :size="15" />
          <span>Managed Storage &amp; Assets</span>
        </button>
        <button
          class="tab-btn tab-danger"
          :class="{ active: activeTab === 'reset' }"
          data-testid="tab-reset"
          @click="switchTab('reset')"
        >
          <RotateCcw :size="15" />
          <span>Factory Reset</span>
        </button>
      </div>

      <!-- Status / Error Banners -->
      <div v-if="errorMessage" class="banner error-banner" data-testid="workstation-error-banner">
        <AlertTriangle :size="16" />
        <span>{{ errorMessage }}</span>
      </div>
      <div v-if="statusMessage" class="banner success-banner" data-testid="workstation-status-banner">
        <CheckCircle2 :size="16" />
        <span>{{ statusMessage }}</span>
      </div>

      <!-- Body -->
      <div class="modal-body">
        <!-- ========================================== -->
        <!-- TAB 1: MULTI-DB SUB-DATABASE BUILDER       -->
        <!-- ========================================== -->
        <div v-if="activeTab === 'builder'" class="tab-panel" data-testid="panel-builder">
          <div class="builder-grid">
            <!-- Left Column: Source Databases & Write Policy Lock -->
            <div class="section-card">
              <div class="section-header-row">
                <h3><Database :size="16" /> 1. Source Databases &amp; Lock Policy</h3>
                <div class="quick-btn-group">
                  <button
                    type="button"
                    class="mini-btn"
                    data-testid="select-all-user-dbs-btn"
                    @click="selectAllUserDbs"
                  >
                    Select All User DBs
                  </button>
                  <button
                    type="button"
                    class="mini-btn"
                    data-testid="select-active-db-btn"
                    @click="selectActiveDbOnly"
                  >
                    Select Active DB
                  </button>
                </div>
              </div>
              <p class="section-hint">
                Select one or more source databases to scan. Reference databases default to
                <strong>Sync-Only</strong> so manual edits never corrupt master archives.
              </p>

              <div v-if="allAvailableDatabases.length === 0" class="empty-box">
                No discovered databases yet. Open or import a database first.
              </div>

              <div v-else class="source-db-list" data-testid="source-db-list">
                <div
                  v-for="db in allAvailableDatabases"
                  :key="db.basePath"
                  class="source-db-item"
                  :class="{ selected: selectedSources.includes(db.basePath) }"
                >
                  <label class="source-db-label">
                    <input
                      type="checkbox"
                      :checked="selectedSources.includes(db.basePath)"
                      :data-testid="`source-db-checkbox-${db.name}`"
                      @change="toggleSourceDb(db.basePath)"
                    />
                    <div class="source-db-meta">
                      <span class="source-db-name">{{ db.name }}</span>
                      <span class="source-db-sub">
                        {{ db.category }} • {{ (db.numGames || 0).toLocaleString() }} games
                        <template v-if="db.sizeStr"> • {{ db.sizeStr }}</template>
                      </span>
                    </div>
                  </label>

                  <button
                    type="button"
                    class="policy-badge-btn"
                    :class="`policy-${db.writePolicy || 'editable'}`"
                    :data-testid="`toggle-policy-btn-${db.name}`"
                    title="Click to cycle write policy (Editable -> Sync-Only -> Locked)"
                    @click="handleToggleWritePolicy(db)"
                  >
                    <Lock v-if="db.writePolicy === 'locked' || db.writePolicy === 'sync_only'" :size="12" />
                    <Unlock v-else :size="12" />
                    <span>{{ formatWritePolicyLabel(db.writePolicy) }}</span>
                  </button>
                </div>
              </div>

              <!-- Target Focused Database -->
              <div class="target-db-box">
                <h4><Layers :size="15" /> 2. Target Focused Database</h4>
                <div class="form-row">
                  <label>Target Database Name (.si4)</label>
                  <input
                    v-model="targetName"
                    type="text"
                    class="text-input"
                    data-testid="target-name-input"
                    placeholder="e.g. Fischer_Najdorf_Study or My_Sicilian_Prep"
                  />
                </div>
                <div class="form-row">
                  <label>Description</label>
                  <input
                    v-model="targetDescription"
                    type="text"
                    class="text-input"
                    data-testid="target-desc-input"
                    placeholder="Optional description for the new focused database"
                  />
                </div>
              </div>
            </div>

            <!-- Right Column: Focus Filters (Players, Openings, Rating/Year) -->
            <div class="section-card">
              <h3><Filter :size="16" /> 3. Focus Filters (Players &amp; Opening Lines)</h3>

              <!-- Players / Users Filter -->
              <div class="filter-group">
                <div class="filter-label-row">
                  <label>Players / Users (comma-separated)</label>
                  <select v-model="playerColor" class="select-input compact-select" data-testid="player-color-select">
                    <option value="any">Any Color</option>
                    <option value="white">White Only</option>
                    <option value="black">Black Only</option>
                  </select>
                </div>
                <input
                  v-model="playersInput"
                  type="text"
                  class="text-input"
                  data-testid="players-filter-input"
                  placeholder="e.g. Bobby Fischer, Magnus Carlsen, DrNykterstein"
                />

                <!-- Quick-add Famous / Notable Players Chips -->
                <div v-if="notablePlayers.length > 0" class="chip-section">
                  <span class="chip-caption"><UserPlus :size="12" /> Quick-Add Notable Players:</span>
                  <div class="chip-list" data-testid="notable-player-chips">
                    <button
                      v-for="p in notablePlayers.slice(0, 10)"
                      :key="p.name"
                      type="button"
                      class="chip-btn"
                      :data-testid="`player-chip-${p.name}`"
                      @click="addPlayerChip(p)"
                    >
                      <span>{{ p.name }}</span>
                      <span v-if="p.gameCount > 0" class="chip-count">{{ p.gameCount }}</span>
                    </button>
                  </div>
                </div>
              </div>

              <!-- Opening Line / ECO / Move Sequence -->
              <div class="filter-group">
                <label>ECO Code or Range</label>
                <input
                  v-model="ecoFilter"
                  type="text"
                  class="text-input"
                  data-testid="eco-filter-input"
                  placeholder="e.g. B90 or B20-B99"
                />
                <div class="chip-list">
                  <button
                    v-for="preset in ecoPresets"
                    :key="preset.code"
                    type="button"
                    class="chip-btn"
                    :data-testid="`eco-preset-${preset.code}`"
                    @click="applyEcoPreset(preset.code)"
                  >
                    {{ preset.label }}
                  </button>
                </div>
              </div>

              <div class="filter-group">
                <div class="filter-label-row">
                  <label>Opening Move Prefix Sequence (SAN or UCI)</label>
                  <button
                    v-if="props.currentOpeningMoves && props.currentOpeningMoves.length > 0"
                    type="button"
                    class="mini-btn accent"
                    data-testid="use-board-moves-btn"
                    @click="useMovesOnBoard"
                  >
                    Use Moves on Board
                  </button>
                </div>
                <input
                  v-model="openingMovesInput"
                  type="text"
                  class="text-input"
                  data-testid="opening-moves-input"
                  placeholder="e.g. 1.e4 c5 2.Nf3 d6 3.d4 cxd4 4.Nxd4 Nf6 5.Nc3 a6"
                />
              </div>

              <!-- Rating / Year / Result / Deduplicate -->
              <div class="filter-row-3">
                <div class="form-row">
                  <label>Min Elo</label>
                  <input
                    v-model.number="minElo"
                    type="number"
                    min="0"
                    max="3500"
                    step="50"
                    class="text-input"
                    data-testid="min-elo-input"
                  />
                </div>
                <div class="form-row">
                  <label>Year From</label>
                  <input
                    v-model.number="yearFrom"
                    type="number"
                    min="0"
                    max="2100"
                    class="text-input"
                    data-testid="year-from-input"
                    placeholder="e.g. 1950"
                  />
                </div>
                <div class="form-row">
                  <label>Year To</label>
                  <input
                    v-model.number="yearTo"
                    type="number"
                    min="0"
                    max="2100"
                    class="text-input"
                    data-testid="year-to-input"
                    placeholder="e.g. 2026"
                  />
                </div>
              </div>

              <div class="filter-row-2">
                <div class="form-row">
                  <label>Result Filter</label>
                  <select v-model="resultFilter" class="select-input" data-testid="result-filter-select">
                    <option value="">Any Result</option>
                    <option value="1-0">White Wins (1-0)</option>
                    <option value="0-1">Black Wins (0-1)</option>
                    <option value="1/2-1/2">Draws (1/2-1/2)</option>
                  </select>
                </div>
                <div class="form-row checkbox-row">
                  <label class="checkbox-label">
                    <input
                      v-model="deduplicate"
                      type="checkbox"
                      data-testid="deduplicate-checkbox"
                    />
                    <span>Deduplicate across source databases</span>
                  </label>
                </div>
              </div>

              <div class="action-bar">
                <button
                  type="button"
                  class="primary-btn"
                  data-testid="build-focused-db-btn"
                  :disabled="loading"
                  @click="handleBuildSubDatabase"
                >
                  <RefreshCw v-if="loading" :size="16" class="spin" />
                  <Layers v-else :size="16" />
                  <span>{{ loading ? 'Building Sub-Database...' : 'Build Focused Sub-Database' }}</span>
                </button>
                <button
                  v-if="loading"
                  type="button"
                  class="mini-btn"
                  data-testid="cancel-build-subdb-btn"
                  @click="CancelActiveOperation()"
                >
                  ✕ Cancel Build
                </button>
              </div>

              <!-- Build Result Summary Card -->
              <div v-if="buildResult" class="result-card" data-testid="build-result-card">
                <div class="result-card-header">
                  <CheckCircle2 :size="18" class="success-icon" />
                  <div>
                    <h4>Focused Database Ready: {{ buildResult.targetName }}</h4>
                    <code class="mono-path">{{ buildResult.targetBasePath }}.si4</code>
                  </div>
                </div>
                <div class="result-metrics">
                  <div class="metric-item">
                    <span class="metric-label">Games Matched</span>
                    <span class="metric-val highlight" data-testid="result-games-matched">
                      {{ buildResult.gamesMatched.toLocaleString() }}
                    </span>
                  </div>
                  <div class="metric-item">
                    <span class="metric-label">Total Scanned</span>
                    <span class="metric-val" data-testid="result-games-scanned">
                      {{ buildResult.totalGamesScanned.toLocaleString() }}
                    </span>
                  </div>
                  <div class="metric-item">
                    <span class="metric-label">Duplicates Skipped</span>
                    <span class="metric-val" data-testid="result-duplicates-skipped">
                      {{ buildResult.duplicatesSkipped.toLocaleString() }}
                    </span>
                  </div>
                </div>
                <button
                  type="button"
                  class="open-built-btn"
                  data-testid="open-built-db-btn"
                  @click="handleOpenBuiltDatabase"
                >
                  <Play :size="14" />
                  <span>Open Focused Database Now</span>
                </button>
              </div>
            </div>
          </div>
        </div>

        <!-- ========================================== -->
        <!-- TAB 2: MANAGED STORAGE & ASSET IMPORTER    -->
        <!-- ========================================== -->
        <div v-if="activeTab === 'storage'" class="tab-panel" data-testid="panel-storage">
          <div v-if="storageReport" class="storage-summary-bar" data-testid="storage-summary-bar">
            <div class="storage-root-pill">
              <span class="root-label">Data Directory:</span>
              <code data-testid="data-root-dir">{{ storageReport.dataRootDir }}</code>
              <button
                type="button"
                class="icon-btn"
                title="Copy Data Root Path"
                @click="copyDirectoryPath('dataRoot', storageReport.dataRootDir)"
              >
                <Check v-if="copiedPathKey === 'dataRoot'" :size="13" />
                <Copy v-else :size="13" />
              </button>
            </div>
            <div class="storage-root-pill">
              <span class="root-label">Config Directory:</span>
              <code data-testid="config-root-dir">{{ storageReport.configRootDir }}</code>
              <button
                type="button"
                class="icon-btn"
                title="Copy Config Root Path"
                @click="copyDirectoryPath('configRoot', storageReport.configRootDir)"
              >
                <Check v-if="copiedPathKey === 'configRoot'" :size="13" />
                <Copy v-else :size="13" />
              </button>
            </div>
            <div class="storage-totals">
              <span><strong>{{ storageReport.totalFiles }}</strong> files</span>
              <span>•</span>
              <span><strong>{{ storageReport.totalSizeStr }}</strong> used</span>
            </div>
          </div>

          <div v-if="storageReport" class="categories-grid" data-testid="storage-categories-grid">
            <div
              v-for="cat in storageReport.categories"
              :key="cat.key"
              class="category-card"
              :data-testid="`storage-category-${cat.key}`"
            >
              <div class="category-card-header">
                <div>
                  <h4>{{ cat.label }}</h4>
                  <p class="category-desc">{{ cat.description }}</p>
                </div>
                <div class="category-badges">
                  <span class="badge-pill">{{ cat.fileCount }} files</span>
                  <span class="badge-pill accent">{{ cat.sizeFormatted }}</span>
                </div>
              </div>

              <!-- Full directory path + Copy Path -->
              <div class="path-box">
                <code class="mono-path" :data-testid="`category-path-${cat.key}`">{{ cat.path }}</code>
                <button
                  type="button"
                  class="mini-btn"
                  :data-testid="`copy-path-btn-${cat.key}`"
                  @click="copyDirectoryPath(cat.key, cat.path)"
                >
                  <Check v-if="copiedPathKey === cat.key" :size="12" />
                  <Copy v-else :size="12" />
                  <span>{{ copiedPathKey === cat.key ? 'Copied' : 'Copy Path' }}</span>
                </button>
              </div>

              <!-- Import / Copy Asset Controls -->
              <div class="import-controls-row">
                <button
                  type="button"
                  class="mini-btn accent"
                  :data-testid="`select-import-btn-${cat.key}`"
                  @click="handleSelectAndImportAsset(cat.key)"
                >
                  <Upload :size="12" />
                  <span>Import / Copy File into NewScid</span>
                </button>
                <div class="inline-copy-input">
                  <input
                    v-model="customAssetPaths[cat.key]"
                    type="text"
                    class="text-input compact-input"
                    :data-testid="`custom-asset-input-${cat.key}`"
                    placeholder="/path/to/external/file..."
                  />
                  <button
                    type="button"
                    class="mini-btn"
                    :data-testid="`copy-asset-btn-${cat.key}`"
                    @click="handleCopyCustomPathIntoNewScid(cat.key)"
                  >
                    Copy Path into NewScid
                  </button>
                </div>
              </div>

              <!-- Stored Assets List -->
              <div v-if="cat.assets && cat.assets.length > 0" class="assets-table-wrap">
                <div
                  v-for="asset in cat.assets.slice(0, 8)"
                  :key="asset.path"
                  class="asset-row"
                >
                  <span class="asset-name">{{ asset.name }}</span>
                  <span class="asset-meta">{{ asset.sizeFormatted }}</span>
                </div>
                <div v-if="cat.assets.length > 8" class="asset-more">
                  + {{ cat.assets.length - 8 }} more files in {{ cat.key }}
                </div>
              </div>
              <div v-else class="empty-assets">
                Directory is currently empty.
              </div>
            </div>
          </div>
        </div>

        <!-- ========================================== -->
        <!-- TAB 3: FACTORY RESET                       -->
        <!-- ========================================== -->
        <div v-if="activeTab === 'reset'" class="tab-panel" data-testid="panel-reset">
          <div class="reset-container">
            <div class="danger-warning-banner" data-testid="factory-reset-warning">
              <ShieldAlert :size="28" class="danger-icon" />
              <div>
                <h3>Factory Reset — Wipe All Workstation Data &amp; Start Over</h3>
                <p>
                  <strong>Factory Reset</strong> permanently wipes all user databases, opponent personas,
                  opening repertoires, training vaults, and saved settings in NewScid's directories back to
                  <strong>FACTORY SETTINGS (empty/nonexistent)</strong> so you can start over from scratch.
                </p>
              </div>
            </div>

            <div class="reset-options-card">
              <h4><Archive :size="16" /> Backup &amp; Preservation Options</h4>

              <!-- Opt-In Backup Checkbox (MUST default to false!) -->
              <label class="reset-option-row highlight-opt-in">
                <input
                  v-model="createOptInBackup"
                  type="checkbox"
                  data-testid="opt-in-backup-checkbox"
                />
                <div>
                  <span class="option-title">
                    Opt-In: Back up my current databases and settings to a compressed (.tar.gz) archive before wiping
                  </span>
                  <span class="option-sub">
                    Unchecked by default. Check this box if you want NewScid to save a timestamped
                    <code>.tar.gz</code> backup archive before resetting.
                  </span>
                </div>
              </label>

              <label class="reset-option-row">
                <input
                  v-model="keepEngines"
                  type="checkbox"
                  data-testid="keep-engines-checkbox"
                />
                <div>
                  <span class="option-title">Keep copied UCI engine binaries</span>
                  <span class="option-sub">Preserve engine binaries stored inside NewScid's managed engines folder.</span>
                </div>
              </label>

              <label class="reset-option-row">
                <input
                  v-model="keepBooks"
                  type="checkbox"
                  data-testid="keep-books-checkbox"
                />
                <div>
                  <span class="option-title">Keep copied Opening &amp; Endgame books</span>
                  <span class="option-sub">Preserve Polyglot (.bin) books and Syzygy tablebases.</span>
                </div>
              </label>

              <div class="reset-actions">
                <button
                  type="button"
                  class="danger-reset-btn"
                  data-testid="confirm-factory-reset-btn"
                  :disabled="loading"
                  @click="handleFactoryReset"
                >
                  <RotateCcw :size="16" />
                  <span>Wipe Content &amp; Reset to Factory Settings</span>
                </button>
              </div>
            </div>

            <!-- Factory Reset Result Summary -->
            <div v-if="factoryResetResult" class="result-card" data-testid="factory-reset-result-card">
              <div class="result-card-header">
                <CheckCircle2 :size="18" class="success-icon" />
                <div>
                  <h4>Workstation Restored to Clean Factory Settings</h4>
                  <p>{{ factoryResetResult.message }}</p>
                </div>
              </div>
              <div class="result-metrics">
                <div class="metric-item">
                  <span class="metric-label">Files Removed</span>
                  <span class="metric-val" data-testid="reset-files-removed">
                    {{ factoryResetResult.filesRemoved }}
                  </span>
                </div>
                <div class="metric-item">
                  <span class="metric-label">Disk Space Freed</span>
                  <span class="metric-val" data-testid="reset-size-freed">
                    {{ factoryResetResult.sizeFreedStr }}
                  </span>
                </div>
                <div v-if="factoryResetResult.backupCreated" class="metric-item">
                  <span class="metric-label">Opt-In Backup Archive</span>
                  <code class="mono-path" data-testid="reset-backup-path">
                    {{ factoryResetResult.backupArchivePath }}
                  </code>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.modal-backdrop {
  position: fixed;
  inset: 0;
  background: rgba(8, 11, 18, 0.78);
  backdrop-filter: blur(4px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1200;
  padding: 20px;
}

.modal-container {
  background: #141923;
  border: 1px solid #2a3346;
  border-radius: 12px;
  width: 100%;
  max-width: 1080px;
  max-height: 90vh;
  display: flex;
  flex-direction: column;
  color: #e4e9f2;
  box-shadow: 0 22px 60px rgba(0, 0, 0, 0.65);
  overflow: hidden;
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 22px;
  background: #19202e;
  border-bottom: 1px solid #263044;
}

.header-title {
  display: flex;
  align-items: center;
  gap: 12px;
}

.header-icon {
  color: #60a5fa;
}

.header-title h2 {
  margin: 0;
  font-size: 1.1rem;
  font-weight: 700;
}

.header-subtitle {
  margin: 2px 0 0;
  font-size: 0.8rem;
  color: #94a3b8;
}

.close-btn {
  background: transparent;
  border: none;
  color: #94a3b8;
  cursor: pointer;
  padding: 6px;
  border-radius: 6px;
}

.close-btn:hover {
  background: #263044;
  color: #fff;
}

.modal-tabs {
  display: flex;
  gap: 8px;
  padding: 10px 22px;
  background: #111620;
  border-bottom: 1px solid #232c3d;
}

.tab-btn {
  display: flex;
  align-items: center;
  gap: 7px;
  padding: 8px 14px;
  border-radius: 7px;
  border: 1px solid transparent;
  background: transparent;
  color: #94a3b8;
  font-size: 0.85rem;
  font-weight: 600;
  cursor: pointer;
}

.tab-btn:hover {
  background: #1b2333;
  color: #e2e8f0;
}

.tab-btn.active {
  background: #1e293b;
  border-color: #3b82f6;
  color: #60a5fa;
}

.tab-btn.tab-danger.active {
  border-color: #ef4444;
  color: #f87171;
}

.banner {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 22px;
  font-size: 0.85rem;
}

.error-banner {
  background: rgba(239, 68, 68, 0.15);
  color: #fca5a5;
  border-bottom: 1px solid rgba(239, 68, 68, 0.35);
}

.success-banner {
  background: rgba(34, 197, 94, 0.14);
  color: #86efac;
  border-bottom: 1px solid rgba(34, 197, 94, 0.35);
}

.modal-body {
  padding: 20px 22px;
  overflow-y: auto;
  flex: 1;
}

.builder-grid {
  display: grid;
  grid-template-columns: 1fr 1.15fr;
  gap: 18px;
}

.section-card {
  background: #181f2c;
  border: 1px solid #263044;
  border-radius: 10px;
  padding: 16px;
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.section-header-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.section-card h3 {
  margin: 0;
  font-size: 0.95rem;
  display: flex;
  align-items: center;
  gap: 7px;
  color: #f1f5f9;
}

.section-hint {
  margin: 0;
  font-size: 0.78rem;
  color: #94a3b8;
  line-height: 1.4;
}

.quick-btn-group {
  display: flex;
  gap: 6px;
}

.mini-btn {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 5px 10px;
  border-radius: 6px;
  border: 1px solid #334155;
  background: #1e293b;
  color: #cbd5e1;
  font-size: 0.75rem;
  font-weight: 600;
  cursor: pointer;
}

.mini-btn:hover {
  background: #283548;
  color: #fff;
}

.mini-btn.accent {
  border-color: #2563eb;
  background: rgba(37, 99, 235, 0.2);
  color: #93c5fd;
}

.source-db-list {
  display: flex;
  flex-direction: column;
  gap: 7px;
  max-height: 240px;
  overflow-y: auto;
}

.source-db-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 8px 10px;
  border-radius: 7px;
  background: #121722;
  border: 1px solid #222b3b;
}

.source-db-item.selected {
  border-color: #3b82f6;
  background: rgba(59, 130, 246, 0.08);
}

.source-db-label {
  display: flex;
  align-items: center;
  gap: 9px;
  cursor: pointer;
  flex: 1;
}

.source-db-meta {
  display: flex;
  flex-direction: column;
}

.source-db-name {
  font-size: 0.85rem;
  font-weight: 600;
  color: #e2e8f0;
}

.source-db-sub {
  font-size: 0.73rem;
  color: #94a3b8;
}

.policy-badge-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 8px;
  border-radius: 999px;
  font-size: 0.72rem;
  font-weight: 600;
  cursor: pointer;
  border: 1px solid transparent;
}

.policy-editable {
  background: rgba(34, 197, 94, 0.15);
  color: #86efac;
  border-color: rgba(34, 197, 94, 0.35);
}

.policy-sync_only {
  background: rgba(245, 158, 11, 0.15);
  color: #fcd34d;
  border-color: rgba(245, 158, 11, 0.35);
}

.policy-locked {
  background: rgba(239, 68, 68, 0.15);
  color: #fca5a5;
  border-color: rgba(239, 68, 68, 0.35);
}

.target-db-box {
  border-top: 1px solid #263044;
  padding-top: 12px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.target-db-box h4 {
  margin: 0;
  font-size: 0.88rem;
  display: flex;
  align-items: center;
  gap: 6px;
}

.form-row,
.filter-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.form-row label,
.filter-group label {
  font-size: 0.78rem;
  font-weight: 600;
  color: #cbd5e1;
}

.filter-label-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.text-input,
.select-input {
  background: #10151f;
  border: 1px solid #2b364b;
  border-radius: 6px;
  padding: 7px 10px;
  color: #f1f5f9;
  font-size: 0.83rem;
}

.compact-select {
  padding: 4px 8px;
  font-size: 0.76rem;
}

.chip-section {
  display: flex;
  flex-direction: column;
  gap: 5px;
  margin-top: 2px;
}

.chip-caption {
  font-size: 0.73rem;
  color: #94a3b8;
  display: flex;
  align-items: center;
  gap: 4px;
}

.chip-list {
  display: flex;
  flex-wrap: wrap;
  gap: 5px;
}

.chip-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 3px 8px;
  border-radius: 999px;
  background: #1e293b;
  border: 1px solid #334155;
  color: #cbd5e1;
  font-size: 0.73rem;
  cursor: pointer;
}

.chip-btn:hover {
  border-color: #60a5fa;
  color: #fff;
}

.chip-count {
  background: #0f172a;
  color: #60a5fa;
  padding: 1px 5px;
  border-radius: 999px;
  font-size: 0.68rem;
}

.filter-row-3 {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 10px;
}

.filter-row-2 {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
  align-items: end;
}

.checkbox-row {
  justify-content: center;
}

.checkbox-label {
  display: flex;
  align-items: center;
  gap: 7px;
  cursor: pointer;
  font-size: 0.8rem;
}

.primary-btn {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 10px 16px;
  border-radius: 7px;
  border: none;
  background: linear-gradient(135deg, #2563eb, #1d4ed8);
  color: #fff;
  font-weight: 700;
  font-size: 0.88rem;
  cursor: pointer;
}

.primary-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.result-card {
  background: rgba(16, 185, 129, 0.08);
  border: 1px solid rgba(16, 185, 129, 0.35);
  border-radius: 8px;
  padding: 12px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.result-card-header {
  display: flex;
  align-items: center;
  gap: 10px;
}

.result-card-header h4 {
  margin: 0;
  font-size: 0.88rem;
  color: #a7f3d0;
}

.result-metrics {
  display: flex;
  gap: 16px;
  flex-wrap: wrap;
}

.metric-item {
  display: flex;
  flex-direction: column;
}

.metric-label {
  font-size: 0.72rem;
  color: #94a3b8;
}

.metric-val {
  font-size: 0.95rem;
  font-weight: 700;
}

.metric-val.highlight {
  color: #34d399;
}

.open-built-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 7px 12px;
  border-radius: 6px;
  background: #059669;
  color: #fff;
  border: none;
  font-size: 0.8rem;
  font-weight: 600;
  cursor: pointer;
}

/* Tab 2: Storage */
.storage-summary-bar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 12px 14px;
  background: #181f2c;
  border: 1px solid #263044;
  border-radius: 8px;
  margin-bottom: 16px;
}

.storage-root-pill {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 0.78rem;
}

.root-label {
  color: #94a3b8;
  font-weight: 600;
}

.categories-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 14px;
}

.category-card {
  background: #181f2c;
  border: 1px solid #263044;
  border-radius: 9px;
  padding: 14px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.category-card-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 8px;
}

.category-card-header h4 {
  margin: 0;
  font-size: 0.9rem;
}

.category-desc {
  margin: 2px 0 0;
  font-size: 0.74rem;
  color: #94a3b8;
}

.category-badges {
  display: flex;
  gap: 5px;
}

.badge-pill {
  padding: 2px 7px;
  border-radius: 999px;
  background: #0f172a;
  border: 1px solid #263044;
  font-size: 0.7rem;
  color: #cbd5e1;
}

.badge-pill.accent {
  color: #60a5fa;
}

.path-box {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  background: #10151f;
  border: 1px solid #222b3b;
  padding: 6px 10px;
  border-radius: 6px;
}

.mono-path {
  font-family: monospace;
  font-size: 0.75rem;
  color: #93c5fd;
  word-break: break-all;
}

.import-controls-row {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.inline-copy-input {
  display: flex;
  gap: 6px;
}

.compact-input {
  flex: 1;
  padding: 5px 8px;
  font-size: 0.76rem;
}

.assets-table-wrap {
  border-top: 1px solid #222b3b;
  padding-top: 8px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.asset-row {
  display: flex;
  justify-content: space-between;
  font-size: 0.75rem;
  color: #cbd5e1;
}

.asset-meta {
  color: #94a3b8;
}

.empty-assets {
  font-size: 0.74rem;
  color: #64748b;
  font-style: italic;
}

/* Tab 3: Factory Reset */
.reset-container {
  max-width: 720px;
  margin: 0 auto;
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.danger-warning-banner {
  display: flex;
  align-items: flex-start;
  gap: 14px;
  padding: 16px;
  border-radius: 10px;
  background: rgba(239, 68, 68, 0.12);
  border: 1px solid rgba(239, 68, 68, 0.45);
}

.danger-icon {
  color: #f87171;
  flex-shrink: 0;
}

.danger-warning-banner h3 {
  margin: 0 0 6px;
  font-size: 1rem;
  color: #fca5a5;
}

.danger-warning-banner p {
  margin: 0;
  font-size: 0.84rem;
  color: #e2e8f0;
  line-height: 1.45;
}

.reset-options-card {
  background: #181f2c;
  border: 1px solid #263044;
  border-radius: 10px;
  padding: 18px;
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.reset-options-card h4 {
  margin: 0;
  display: flex;
  align-items: center;
  gap: 7px;
  font-size: 0.92rem;
}

.reset-option-row {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 10px 12px;
  border-radius: 8px;
  background: #121722;
  border: 1px solid #222b3b;
  cursor: pointer;
}

.reset-option-row.highlight-opt-in {
  border-color: #3b82f6;
  background: rgba(59, 130, 246, 0.08);
}

.option-title {
  display: block;
  font-size: 0.85rem;
  font-weight: 600;
  color: #f1f5f9;
}

.option-sub {
  display: block;
  font-size: 0.75rem;
  color: #94a3b8;
  margin-top: 2px;
}

.danger-reset-btn {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 11px 16px;
  border-radius: 8px;
  border: 1px solid #ef4444;
  background: linear-gradient(135deg, #dc2626, #b91c1c);
  color: #fff;
  font-size: 0.9rem;
  font-weight: 700;
  cursor: pointer;
}

.danger-reset-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}
</style>
