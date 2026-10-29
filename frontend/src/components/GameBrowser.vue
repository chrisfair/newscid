<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
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
  X,
  Star,
  Lock,
  Unlock,
  RefreshCw,
  Crosshair,
  BookOpen,
  UserPlus,
  Layers,
  RotateCcw
} from '@lucide/vue'
import {
  GetNotablePlayersDropdown,
  AddCustomNotablePlayer,
  RemoveNotablePlayer,
  ResetNotablePlayersToDefault
} from '../../wailsjs/go/main/App'

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

export interface NotablePlayerOption {
  name: string
  title: string
  era: string
  aliases: string[]
  matchedNames: string[]
  gameCount: number
  peakElo: number
  yearStart: number
  yearEnd: number
  isCustom: boolean
}

const props = defineProps<{
  games: GameSummary[]
  totalGames: number
  currentPage: number
  pageSize: number
  activeGameId?: number
  dbName?: string
  activeCriteria?: Record<string, any>
  currentFen?: string
  writePolicy?: string
  isLocked?: boolean
}>()

const emit = defineEmits<{
  (e: 'loadGame', id: number): void
  (e: 'pageChange', page: number): void
  (e: 'search', criteria: any): void
  (e: 'openAdvancedSearch'): void
  (e: 'newGame'): void
  (e: 'saveGame'): void
  (e: 'deleteGame', id: number): void
  (e: 'cycleWritePolicy'): void
  (e: 'openSubDbBuilder', filter?: any): void
  (e: 'openStandardImport'): void
  (e: 'openMegaDownloader'): void
  (e: 'ensurePersonalDb'): void
  (e: 'openOnlineSync'): void
}>()

const startingFENPrefix = 'rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR'
const currentBoardMoveNumber = computed(() => {
  if (!props.currentFen || props.currentFen.startsWith(startingFENPrefix)) {
    return 0
  }
  const parts = props.currentFen.trim().split(/\s+/)
  const moveNum = parts.length >= 6 ? parseInt(parts[5], 10) : 0
  return Number.isFinite(moveNum) && moveNum > 0 ? moveNum : 1
})

const searchPlayer = ref('')
const selectedFamousPlayer = ref('')
const filterResult = ref('')
const searchOpeningInput = ref('')
const selectedOpeningPreset = ref('')
const matchBoardPosition = ref(false)

// Famous & Custom Players Dropdown state
const notablePlayers = ref<NotablePlayerOption[]>([])
const showCustomizePlayers = ref(false)
const newCustomPlayerName = ref('')
const newCustomPlayerTitle = ref('')
const newCustomPlayerAliases = ref('')

const OPENING_PRESETS = [
  { label: 'Any Opening', value: '', ecoFrom: '', ecoTo: '', moves: '' },
  { label: 'Sicilian Defense (B20–B99)', value: 'B20-B99', ecoFrom: 'B20', ecoTo: 'B99', moves: '' },
  { label: 'Ruy Lopez / Spanish (C60–C99)', value: 'C60-C99', ecoFrom: 'C60', ecoTo: 'C99', moves: '' },
  { label: 'Italian Game (C50–C59)', value: 'C50-C59', ecoFrom: 'C50', ecoTo: 'C59', moves: '' },
  { label: 'French Defense (C00–C19)', value: 'C00-C19', ecoFrom: 'C00', ecoTo: 'C19', moves: '' },
  { label: 'Caro-Kann Defense (B10–B19)', value: 'B10-B19', ecoFrom: 'B10', ecoTo: 'B19', moves: '' },
  { label: "Queen's Gambit (D06–D69)", value: 'D06-D69', ecoFrom: 'D06', ecoTo: 'D69', moves: '' },
  { label: "King's Indian Defense (E60–E99)", value: 'E60-E99', ecoFrom: 'E60', ecoTo: 'E99', moves: '' },
  { label: "Nimzo / Queen's Indian (E10–E59)", value: 'E10-E59', ecoFrom: 'E10', ecoTo: 'E59', moves: '' },
  { label: 'English Opening (A10–A39)', value: 'A10-A39', ecoFrom: 'A10', ecoTo: 'A39', moves: '' },
  { label: 'Flank / Réti (A00–A09)', value: 'A00-A09', ecoFrom: 'A00', ecoTo: 'A09', moves: '' },
]

async function loadNotablePlayers() {
  try {
    const fn = (window as any)?.go?.main?.App?.GetNotablePlayersDropdown || GetNotablePlayersDropdown
    if (typeof fn === 'function') {
      const list = await fn()
      if (Array.isArray(list)) {
        notablePlayers.value = list
      }
    }
  } catch {
    // Ignore in test environments where Wails runtime is not mounted
  }
}

onMounted(() => {
  loadNotablePlayers()
})

watch(() => [props.dbName, props.totalGames], () => {
  loadNotablePlayers()
})

// Sync props.activeCriteria with inline inputs
watch(() => props.activeCriteria, (c) => {
  if (c) {
    searchPlayer.value = c.player || c.white || ''
    filterResult.value = c.result || ''
    if (Array.isArray(c.openingMoves) && c.openingMoves.length > 0) {
      searchOpeningInput.value = c.openingMoves.join(' ')
    } else if (c.eco) {
      searchOpeningInput.value = c.eco
    } else if (c.ecoFrom || c.ecoTo) {
      searchOpeningInput.value = `${c.ecoFrom || ''}-${c.ecoTo || ''}`
    } else {
      searchOpeningInput.value = ''
    }
    matchBoardPosition.value = Boolean(c.positionFen)
  } else {
    searchPlayer.value = ''
    selectedFamousPlayer.value = ''
    filterResult.value = ''
    searchOpeningInput.value = ''
    selectedOpeningPreset.value = ''
    matchBoardPosition.value = false
  }
}, { immediate: true })

// If matchBoardPosition is toggled on and the board FEN changes, update search automatically
watch(() => props.currentFen, (newFen) => {
  if (matchBoardPosition.value && newFen) {
    onSearchInput()
  }
})

function parseOpeningInput(raw: string): { eco?: string; ecoFrom?: string; ecoTo?: string; openingMoves?: string[] } {
  const trimmed = raw.trim()
  if (!trimmed) return {}

  // Check if ECO range like B20-B99
  const rangeMatch = trimmed.match(/^([A-Ea-e]\d{2})\s*[-–]\s*([A-Ea-e]\d{2})$/)
  if (rangeMatch) {
    return {
      ecoFrom: rangeMatch[1].toUpperCase(),
      ecoTo: rangeMatch[2].toUpperCase()
    }
  }

  // Check if single ECO prefix like B, B9, B90
  const ecoMatch = trimmed.match(/^[A-Ea-e]\d{0,2}$/)
  if (ecoMatch) {
    return { eco: trimmed.toUpperCase() }
  }

  // Otherwise treat as an opening move sequence (e.g. "1.e4 c5 2.Nf3" or "e4 e5 Nf3")
  const tokens = trimmed
    .replace(/\d+\.+/g, ' ')
    .split(/\s+/)
    .map(t => t.trim())
    .filter(Boolean)
  if (tokens.length > 0) {
    return { openingMoves: tokens }
  }
  return {}
}

function onSearchInput() {
  const openingParsed = parseOpeningInput(searchOpeningInput.value)
  let playersList: string[] | undefined = undefined

  if (selectedFamousPlayer.value) {
    const found = notablePlayers.value.find(p => p.name === selectedFamousPlayer.value)
    if (found && searchPlayer.value.trim().toLowerCase() === found.name.toLowerCase()) {
      playersList = found.matchedNames?.length
        ? found.matchedNames
        : [found.name, ...(found.aliases || [])]
    } else {
      selectedFamousPlayer.value = ''
    }
  }

  emit('search', {
    ...props.activeCriteria,
    player: searchPlayer.value.trim() || undefined,
    players: playersList,
    result: filterResult.value || undefined,
    eco: openingParsed.eco,
    ecoFrom: openingParsed.ecoFrom,
    ecoTo: openingParsed.ecoTo,
    openingMoves: openingParsed.openingMoves,
    positionFen: matchBoardPosition.value && props.currentFen ? props.currentFen : undefined
  })
}

function onSelectFamousPlayer() {
  if (!selectedFamousPlayer.value) {
    searchPlayer.value = ''
    onSearchInput()
    return
  }
  const found = notablePlayers.value.find(p => p.name === selectedFamousPlayer.value)
  if (found) {
    searchPlayer.value = found.name
    const playersList = found.matchedNames?.length
      ? found.matchedNames
      : [found.name, ...(found.aliases || [])]
    const openingParsed = parseOpeningInput(searchOpeningInput.value)
    emit('search', {
      ...props.activeCriteria,
      player: found.name,
      players: playersList,
      result: filterResult.value || undefined,
      eco: openingParsed.eco,
      ecoFrom: openingParsed.ecoFrom,
      ecoTo: openingParsed.ecoTo,
      openingMoves: openingParsed.openingMoves,
      positionFen: matchBoardPosition.value && props.currentFen ? props.currentFen : undefined
    })
  }
}

function onSelectOpeningPreset() {
  searchOpeningInput.value = selectedOpeningPreset.value
  onSearchInput()
}

function togglePositionSearch() {
  matchBoardPosition.value = !matchBoardPosition.value
  onSearchInput()
}

async function handleAddCustomPlayer() {
  const name = newCustomPlayerName.value.trim()
  if (!name) return
  const aliases = newCustomPlayerAliases.value
    .split(',')
    .map(a => a.trim())
    .filter(Boolean)
  try {
    const list = await AddCustomNotablePlayer(name, aliases, newCustomPlayerTitle.value.trim() || 'Tracked Player', '')
    if (Array.isArray(list)) {
      notablePlayers.value = list
    }
    newCustomPlayerName.value = ''
    newCustomPlayerTitle.value = ''
    newCustomPlayerAliases.value = ''
    selectedFamousPlayer.value = name
    onSelectFamousPlayer()
  } catch (e) {
    console.error('Failed to add custom notable player:', e)
  }
}

async function handleRemoveNotablePlayer(name: string) {
  try {
    const list = await RemoveNotablePlayer(name)
    if (Array.isArray(list)) {
      notablePlayers.value = list
    }
    if (selectedFamousPlayer.value === name) {
      selectedFamousPlayer.value = ''
    }
  } catch (e) {
    console.error('Failed to remove notable player:', e)
  }
}

async function handleResetNotablePlayers() {
  try {
    const list = await ResetNotablePlayersToDefault()
    if (Array.isArray(list)) {
      notablePlayers.value = list
    }
  } catch (e) {
    console.error('Failed to reset notable players:', e)
  }
}

function clearFilters() {
  searchPlayer.value = ''
  selectedFamousPlayer.value = ''
  filterResult.value = ''
  searchOpeningInput.value = ''
  selectedOpeningPreset.value = ''
  matchBoardPosition.value = false
  emit('search', {})
}

function handleExtractSubDb() {
  const openingParsed = parseOpeningInput(searchOpeningInput.value)
  emit('openSubDbBuilder', {
    player: searchPlayer.value.trim() || undefined,
    eco: openingParsed.eco || (openingParsed.ecoFrom ? `${openingParsed.ecoFrom}-${openingParsed.ecoTo}` : undefined),
    openingMoves: openingParsed.openingMoves
  })
}

const effectivePolicy = computed(() => props.writePolicy || (props.isLocked ? 'locked' : 'editable'))

const policyBadge = computed(() => {
  if (effectivePolicy.value === 'locked') {
    return {
      label: 'Locked (Read-Only)',
      shortLabel: 'Locked',
      classes: 'bg-rose-950/60 border-rose-500/40 text-rose-300 hover:bg-rose-900/60',
      tooltip: 'Strictly Locked (Read-Only) — Protected against all edits and syncs. Click to change write policy.'
    }
  }
  if (effectivePolicy.value === 'sync_only') {
    return {
      label: 'Sync-Only Protected',
      shortLabel: 'Sync-Only',
      classes: 'bg-cyan-950/60 border-cyan-500/40 text-cyan-300 hover:bg-cyan-900/60',
      tooltip: 'Reference Database (Sync-Only) — Protected against accidental manual edits; allows Online & Local Sync. Click to change write policy.'
    }
  }
  return {
    label: 'Editable',
    shortLabel: 'Editable',
    classes: 'bg-emerald-950/60 border-emerald-500/40 text-emerald-300 hover:bg-emerald-900/60',
    tooltip: 'Personal / Editable Database — Direct modifications allowed. Click to lock or set Sync-Only.'
  }
})

const hasActiveCriteria = computed(() => {
  if (!props.activeCriteria) return false
  return Object.entries(props.activeCriteria).some(([k, v]) => {
    if (k === 'ignoreDeleted') return false
    if (Array.isArray(v)) return v.length > 0
    return v !== undefined && v !== '' && v !== false
  })
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
  if (Array.isArray(c.openingMoves) && c.openingMoves.length > 0) {
    list.push(`Moves: ${c.openingMoves.join(' ')}`)
  }
  if (c.positionFen) {
    list.push('Board Position Match')
  }
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
    <!-- Top Header Row: DB Title, Write-Policy Lock Badge, and Game Actions -->
    <div class="browser-header flex flex-wrap items-center justify-between gap-2 px-3 py-2 bg-zinc-950 border-b border-zinc-800/80">
      <div class="flex items-center gap-2 flex-wrap">
        <Database class="w-4 h-4 text-emerald-400" />
        <span class="font-bold text-sm text-zinc-200">{{ dbName || 'No Database Open' }}</span>
        <span class="text-xs text-zinc-500 font-mono">({{ totalGames.toLocaleString() }} games)</span>

        <!-- Database Lock / Write Policy Toggle Badge -->
        <button
          v-if="dbName"
          type="button"
          class="px-2 py-0.5 rounded-md border text-[11px] font-semibold flex items-center gap-1 transition-colors cursor-pointer"
          :class="policyBadge.classes"
          :title="policyBadge.tooltip"
          @click="emit('cycleWritePolicy')"
        >
          <Lock v-if="effectivePolicy === 'locked'" class="w-3 h-3" />
          <RefreshCw v-else-if="effectivePolicy === 'sync_only'" class="w-3 h-3" />
          <Unlock v-else class="w-3 h-3" />
          <span>{{ policyBadge.shortLabel }}</span>
        </button>
      </div>

      <!-- Quick Action Buttons -->
      <div class="flex items-center gap-1.5 flex-wrap">
        <!-- 1-Click Extract to Sub-Database Button -->
        <button
          type="button"
          class="flex items-center gap-1 px-2.5 py-1 rounded-lg bg-indigo-950/60 hover:bg-indigo-900/70 text-indigo-300 border border-indigo-500/40 text-xs font-medium transition-colors cursor-pointer"
          title="Create a focused smaller database from this search or multiple databases"
          @click="handleExtractSubDb"
        >
          <Layers class="w-3.5 h-3.5 text-indigo-400" />
          <span>Extract Sub-DB</span>
        </button>

        <button
          type="button"
          class="flex items-center gap-1 px-2.5 py-1 rounded-lg bg-zinc-800 hover:bg-zinc-700 text-zinc-300 text-xs font-medium transition-colors cursor-pointer"
          title="New Blank Game"
          @click="emit('newGame')"
        >
          <Plus class="w-3.5 h-3.5 text-emerald-400" />
          <span>New</span>
        </button>

        <button
          type="button"
          class="flex items-center gap-1 px-2.5 py-1 rounded-lg text-xs font-medium transition-colors cursor-pointer"
          :class="effectivePolicy === 'editable'
            ? 'bg-emerald-600 hover:bg-emerald-500 text-white'
            : 'bg-zinc-800 text-zinc-500 border border-zinc-700'"
          :title="effectivePolicy === 'editable' ? 'Save Current Game into DB' : `Database is ${policyBadge.label} — click the lock badge to enable direct edits`"
          @click="emit('saveGame')"
        >
          <Lock v-if="effectivePolicy !== 'editable'" class="w-3 h-3 text-amber-400" />
          <Save v-else class="w-3.5 h-3.5" />
          <span>Save</span>
        </button>
      </div>
    </div>

    <!-- Intuitive Search Bar: Grouped into 3 Labeled Clusters (1. Player & Legend, 2. Opening & Moves, 3. Position & Result) -->
    <div class="px-3 py-2 bg-zinc-950/80 border-b border-zinc-800 flex flex-wrap items-center gap-2">
      <!-- Player & Famous Legends -->
      <div class="flex flex-wrap items-center gap-1.5 bg-zinc-900/90 border border-zinc-800 rounded-xl px-2 py-1">
        <span class="text-[10px] font-bold uppercase tracking-wider text-amber-400/90 select-none">Player:</span>
        <div class="relative">
          <Search class="w-3.5 h-3.5 absolute left-2.5 top-1/2 -translate-y-1/2 text-zinc-500" />
          <input
            v-model="searchPlayer"
            type="text"
            placeholder="Player (e.g. Fischer, Carlsen)..."
            class="bg-zinc-800 border border-zinc-700 rounded-lg pl-8 pr-2.5 py-1 text-xs text-zinc-200 placeholder-zinc-500 focus:outline-hidden focus:border-emerald-500 w-40"
            @input="onSearchInput"
          />
        </div>

        <!-- Famous & Custom Players Quick-Lookup Dropdown -->
        <div class="relative flex items-center gap-1">
          <select
            v-model="selectedFamousPlayer"
            aria-label="Famous Players Quick Lookup"
            class="bg-zinc-800 border border-amber-500/40 rounded-lg pl-2 pr-6 py-1 text-xs text-amber-200 font-medium focus:outline-hidden focus:border-amber-400 cursor-pointer max-w-[175px]"
            title="Quickly look up games by Famous Players or your custom tracked players"
            @change="onSelectFamousPlayer"
          >
            <option value="" class="bg-zinc-900 text-zinc-200">★ Famous Players...</option>
            <option
              v-for="p in notablePlayers"
              :key="p.name"
              :value="p.name"
              class="bg-zinc-900 text-zinc-100"
            >
              {{ p.gameCount > 0 ? `★ ${p.name} (${p.gameCount})` : `☆ ${p.name}` }}{{ p.isCustom ? ' [Custom]' : '' }}
            </option>
          </select>

          <button
            type="button"
            class="px-2 py-1 rounded-lg bg-zinc-800 hover:bg-zinc-700 border border-zinc-700 text-amber-300 text-xs flex items-center gap-1 cursor-pointer transition-colors"
            title="Customize Famous & Tracked Players Dropdown (Add or remove players)"
            @click="showCustomizePlayers = !showCustomizePlayers"
          >
            <UserPlus class="w-3.5 h-3.5" />
          </button>

          <!-- Customize Famous Players Popover -->
          <div
            v-if="showCustomizePlayers"
            class="absolute top-full left-0 mt-1.5 w-80 bg-zinc-900 border border-zinc-700 rounded-xl shadow-2xl p-3 z-50 text-xs space-y-2.5"
          >
            <div class="flex items-center justify-between border-b border-zinc-800 pb-1.5">
              <span class="font-bold text-amber-300 flex items-center gap-1.5">
                <Star class="w-3.5 h-3.5 fill-amber-400 text-amber-400" />
                <span>Customize Players Dropdown</span>
              </span>
              <button
                type="button"
                class="text-zinc-400 hover:text-white cursor-pointer"
                @click="showCustomizePlayers = false"
              >
                <X class="w-3.5 h-3.5" />
              </button>
            </div>

            <div class="space-y-1.5">
              <input
                v-model="newCustomPlayerName"
                type="text"
                placeholder="Player name (e.g. Paul Morphy or Hikaru)"
                class="w-full bg-zinc-950 border border-zinc-700 rounded-lg px-2.5 py-1 text-xs text-zinc-200"
              />
              <div class="flex gap-1.5">
                <input
                  v-model="newCustomPlayerTitle"
                  type="text"
                  placeholder="Note / Title (optional)"
                  class="flex-1 bg-zinc-950 border border-zinc-700 rounded-lg px-2.5 py-1 text-xs text-zinc-200"
                />
                <button
                  type="button"
                  class="px-2.5 py-1 rounded-lg bg-amber-600 hover:bg-amber-500 text-white font-semibold cursor-pointer"
                  @click="handleAddCustomPlayer"
                >
                  + Add
                </button>
              </div>
              <input
                v-model="newCustomPlayerAliases"
                type="text"
                placeholder="Optional aliases (e.g. Fischer, Robert J)"
                class="w-full bg-zinc-950 border border-zinc-700 rounded-lg px-2.5 py-1 text-[11px] text-zinc-300"
              />
            </div>

            <div class="max-h-36 overflow-y-auto divide-y divide-zinc-800/60 border border-zinc-800 rounded-lg bg-zinc-950/60">
              <div
                v-for="p in notablePlayers"
                :key="p.name"
                class="px-2.5 py-1.5 flex items-center justify-between gap-2"
              >
                <div class="truncate">
                  <span class="font-semibold text-zinc-200">{{ p.name }}</span>
                  <span v-if="p.gameCount > 0" class="ml-1 text-[10px] font-mono text-emerald-400">({{ p.gameCount }})</span>
                  <span v-if="p.isCustom" class="ml-1 text-[10px] text-amber-400">[Custom]</span>
                </div>
                <button
                  type="button"
                  class="text-zinc-500 hover:text-rose-400 cursor-pointer"
                  title="Remove from dropdown"
                  @click="handleRemoveNotablePlayer(p.name)"
                >
                  <X class="w-3.5 h-3.5" />
                </button>
              </div>
            </div>

            <div class="flex items-center justify-between pt-1 border-t border-zinc-800">
              <button
                type="button"
                class="text-[11px] text-zinc-400 hover:text-zinc-200 flex items-center gap-1 cursor-pointer"
                @click="handleResetNotablePlayers"
              >
                <RotateCcw class="w-3 h-3" />
                <span>Restore Default Legends</span>
              </button>
              <button
                type="button"
                class="text-[11px] text-emerald-400 font-semibold cursor-pointer"
                @click="showCustomizePlayers = false"
              >
                Done
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- Opening Family, ECO & Move Sequence -->
      <div class="flex flex-wrap items-center gap-1.5 bg-zinc-900/90 border border-zinc-800 rounded-xl px-2 py-1">
        <span class="text-[10px] font-bold uppercase tracking-wider text-emerald-400/90 select-none">Opening:</span>
        <select
          v-model="selectedOpeningPreset"
          aria-label="Opening Family Preset"
          class="bg-zinc-800 border border-zinc-700 rounded-lg pl-2 pr-6 py-1 text-xs text-zinc-200 font-medium focus:outline-hidden focus:border-emerald-500 cursor-pointer max-w-[155px]"
          title="Filter by popular opening family"
          @change="onSelectOpeningPreset"
        >
          <option
            v-for="preset in OPENING_PRESETS"
            :key="preset.label"
            :value="preset.value"
            class="bg-zinc-900 text-zinc-100"
          >
            {{ preset.label }}
          </option>
        </select>

        <div class="relative">
          <BookOpen class="w-3.5 h-3.5 absolute left-2.5 top-1/2 -translate-y-1/2 text-zinc-500" />
          <input
            v-model="searchOpeningInput"
            type="text"
            placeholder="ECO (B90) or moves (1.e4 c5)..."
            class="bg-zinc-800 border border-zinc-700 rounded-lg pl-8 pr-2.5 py-1 text-xs text-zinc-200 placeholder-zinc-500 focus:outline-hidden focus:border-emerald-500 w-40 font-mono"
            title="Type an ECO code (e.g. B90 or B20-B99) OR opening moves (e.g. 1.e4 c5 2.Nf3)"
            @input="onSearchInput"
          />
        </div>
      </div>

      <!-- ♟️ Board Position, Result & Advanced Filters -->
      <div class="flex flex-wrap items-center gap-1.5 bg-zinc-900/90 border border-zinc-800 rounded-xl px-2 py-1">
        <button
          type="button"
          class="px-2.5 py-1 rounded-lg border text-xs font-semibold flex items-center gap-1.5 transition-colors cursor-pointer"
          :class="matchBoardPosition
            ? 'bg-emerald-600/30 border-emerald-400 text-emerald-200 shadow-xs'
            : 'bg-zinc-800 hover:bg-zinc-700 border-zinc-700 text-zinc-300'"
          title="Find all games in the database that reach the exact position currently on the chessboard"
          @click="togglePositionSearch"
        >
          <Crosshair class="w-3.5 h-3.5" :class="matchBoardPosition ? 'text-emerald-300' : 'text-zinc-400'" />
          <span>Match Board Position</span>
          <span
            v-if="currentBoardMoveNumber > 0"
            class="px-1 py-0.2 rounded text-[10px] font-mono bg-emerald-950/80 text-emerald-300 border border-emerald-700/50"
          >
            Move {{ currentBoardMoveNumber }}
          </span>
        </button>

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

        <button
          type="button"
          class="px-2.5 py-1 rounded-lg bg-zinc-800 hover:bg-zinc-700 text-zinc-300 hover:text-white border border-zinc-700 text-xs flex items-center gap-1.5 transition-colors cursor-pointer"
          title="Advanced Header, Material, Structure & Position Search (Ctrl+H)"
          @click="emit('openAdvancedSearch')"
        >
          <Sliders class="w-3 h-3 text-emerald-400" />
          <span>More Filters</span>
        </button>

        <button
          v-if="hasActiveCriteria"
          type="button"
          class="text-xs text-zinc-400 hover:text-red-400 underline cursor-pointer ml-1"
          @click="clearFilters"
        >
          Clear
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

    <!-- Games Table or Guided Empty-State Setup Hub -->
    <div class="games-table-container flex-1 overflow-y-auto">
      <div v-if="games.length === 0" class="p-4 flex flex-col items-center justify-center min-h-[160px]">
        <!-- Filtered 0 matches state -->
        <div v-if="hasActiveCriteria" class="text-center space-y-2 py-4">
          <p class="text-zinc-400 text-xs">
            No games in <strong>{{ dbName || 'the active database' }}</strong> match the current search filter.
          </p>
          <button
            type="button"
            class="px-3 py-1 rounded-lg bg-zinc-800 hover:bg-zinc-700 border border-zinc-700 text-emerald-300 text-xs font-semibold cursor-pointer"
            @click="clearFilters"
          >
            Clear Search Filters
          </button>
        </div>

        <!-- Guided 3-Tier Database Onboarding Hub when no DB is open or DB is empty -->
        <div v-else class="w-full max-w-4xl space-y-2.5">
          <div class="flex items-center justify-between text-xs px-1">
            <span class="font-bold text-zinc-200">
              {{ dbName ? `'${dbName}' is currently empty (0 games)` : 'Welcome to NewScid — Set Up Your 3-Tier Database Workspace' }}
            </span>
            <span class="text-[11px] text-zinc-500">Original Archives (Locked) • Master Reference (Sync-Only) • Personal User DB (Editable)</span>
          </div>
          <div class="grid grid-cols-1 md:grid-cols-3 gap-2.5 text-xs">
            <button
              type="button"
              class="text-left p-3 rounded-xl bg-zinc-950/80 hover:bg-zinc-800/80 border border-cyan-500/30 hover:border-cyan-400 transition-all cursor-pointer flex flex-col justify-between gap-1.5"
              @click="emit('openMegaDownloader')"
            >
              <div class="font-bold text-cyan-300 flex items-center justify-between">
                <span>1. Build Master Reference DB</span>
                <span class="text-[9px] font-mono px-1.5 py-0.2 rounded bg-cyan-950 text-cyan-300">2026 Archives</span>
              </div>
              <p class="text-[11px] text-zinc-400 leading-snug">
                Download latest 2026 master/elite archives &amp; consolidate into your deduplicated <strong>Master Reference DB</strong> for the Opening Tree.
              </p>
            </button>

            <button
              type="button"
              class="text-left p-3 rounded-xl bg-zinc-950/80 hover:bg-zinc-800/80 border border-emerald-500/30 hover:border-emerald-400 transition-all cursor-pointer flex flex-col justify-between gap-1.5"
              @click="emit('openStandardImport')"
            >
              <div class="font-bold text-emerald-300 flex items-center justify-between">
                <span>2. Standard Import (.pgn / .si4)</span>
                <span class="text-[9px] font-mono px-1.5 py-0.2 rounded bg-emerald-950 text-emerald-300">Original + Master</span>
              </div>
              <p class="text-[11px] text-zinc-400 leading-snug">
                Import any external database as a locked <strong>Original Archive</strong> and automatically merge &amp; deduplicate into <strong>Master Reference</strong>.
              </p>
            </button>

            <button
              type="button"
              class="text-left p-3 rounded-xl bg-zinc-950/80 hover:bg-zinc-800/80 border border-amber-500/30 hover:border-amber-400 transition-all cursor-pointer flex flex-col justify-between gap-1.5"
              @click="emit('ensurePersonalDb')"
            >
              <div class="font-bold text-amber-300 flex items-center justify-between">
                <span>3. Personal User DB (My_Games)</span>
                <span class="text-[9px] font-mono px-1.5 py-0.2 rounded bg-amber-950 text-amber-300">Editable</span>
              </div>
              <p class="text-[11px] text-zinc-400 leading-snug">
                Initialize or open your editable <strong>Personal User Database</strong> for your own games, annotations, Cyclical Drill &amp; Blunder Vault.
              </p>
            </button>
          </div>
        </div>
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
                :title="effectivePolicy === 'editable' ? 'Toggle Deleted' : `Database is ${policyBadge.label}`"
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
