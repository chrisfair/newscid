<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import {
  Users,
  User,
  Plus,
  Trash2,
  Save,
  Database,
  ExternalLink,
  Search,
  Sparkles,
  Bot,
  Trophy,
  CheckCircle2,
  AlertCircle,
  X,
  RefreshCw,
  FolderOpen,
  Globe
} from '@lucide/vue'
import {
  GetPersonas,
  GetPersona,
  SavePersona,
  DeletePersona,
  AddPersonaIdentifier,
  RemovePersonaIdentifier,
  SuggestPersonaIdentifiers,
  GetPersonaStatistics,
  AggregatePersonaGames,
  OpenPersonaDatabase,
  FetchPersonaOnlineGames
} from '../../wailsjs/go/main/App'
import { persona } from '../../wailsjs/go/models'

const props = defineProps<{
  activeDbName?: string
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'sparPersona', p: persona.Persona): void
  (e: 'openPersonaDb', p: persona.Persona): void
}>()

const personasList = ref<persona.Persona[]>([])
const selectedPersona = ref<persona.Persona | null>(null)
const searchFilter = ref('')
const loading = ref(true)
const statusMessage = ref<{ type: 'success' | 'error' | 'info'; text: string } | null>(null)

// Editing state
const editDisplayName = ref('')
const editTitle = ref('')
const editBio = ref('')
const editEstimatedElo = ref<number>(2400)
const editChessCom = ref('')
const editLichess = ref('')
const editFideId = ref('')
const editDatabasePath = ref('')
const newIdentifierInput = ref('')

// Suggestions & Stats
const suggestedAliases = ref<string[]>([])
const loadingSuggestions = ref(false)
const personaStats = ref<persona.PersonaStatistics | null>(null)
const loadingStats = ref(false)
const isAggregating = ref(false)
const isFetchingOnline = ref(false)
const autoFetchOnline = ref(true)

onMounted(async () => {
  await loadPersonas()
})

async function loadPersonas() {
  loading.value = true
  try {
    const list = await GetPersonas()
    personasList.value = list || []
    if (personasList.value.length > 0 && !selectedPersona.value) {
      selectPersona(personasList.value[0])
    }
  } catch (err: any) {
    statusMessage.value = { type: 'error', text: `Failed to load personas: ${err?.message || err}` }
  } finally {
    loading.value = false
  }
}

const filteredPersonas = computed(() => {
  const q = searchFilter.value.toLowerCase().trim()
  if (!q) return personasList.value
  return personasList.value.filter(p =>
    p.displayName.toLowerCase().includes(q) ||
    p.identifiers.some(id => id.toLowerCase().includes(q))
  )
})

function selectPersona(p: persona.Persona, preserveStatus: boolean = false) {
  selectedPersona.value = p
  editDisplayName.value = p.displayName
  editTitle.value = p.title || ''
  editBio.value = p.bio || ''
  editEstimatedElo.value = p.estimatedElo || 2400
  editChessCom.value = p.chessComUsername || ''
  editLichess.value = p.lichessUsername || ''
  editFideId.value = p.fideId || ''
  editDatabasePath.value = p.databasePath || ''
  if (!preserveStatus) {
    statusMessage.value = null
  }
  fetchSuggestions()
  fetchStatistics()
}

function handleCreateNew() {
  const newP = new persona.Persona({
    id: '',
    displayName: 'New Player Persona',
    identifiers: [],
    title: 'Player',
    estimatedElo: 2200,
  })
  selectedPersona.value = newP
  editDisplayName.value = 'New Player Persona'
  editTitle.value = 'Player'
  editBio.value = ''
  editEstimatedElo.value = 2200
  editChessCom.value = ''
  editLichess.value = ''
  editFideId.value = ''
  editDatabasePath.value = ''
  personaStats.value = null
  suggestedAliases.value = []
}

async function handleSave() {
  if (!editDisplayName.value.trim()) {
    statusMessage.value = { type: 'error', text: 'Display name cannot be empty.' }
    return
  }

  try {
    const pToSave = new persona.Persona({
      id: selectedPersona.value?.id || '',
      displayName: editDisplayName.value.trim(),
      identifiers: selectedPersona.value?.identifiers || [],
      title: editTitle.value.trim(),
      bio: editBio.value.trim(),
      estimatedElo: Number(editEstimatedElo.value) || 2400,
      chessComUsername: editChessCom.value.trim(),
      lichessUsername: editLichess.value.trim(),
      fideId: editFideId.value.trim(),
      databasePath: editDatabasePath.value.trim(),
    })

    const saved = await SavePersona(pToSave)
    await loadPersonas()
    const updated = personasList.value.find(p => p.id === saved.id)
    if (updated) selectPersona(updated, true)
    statusMessage.value = { type: 'success', text: `Persona "${saved.displayName}" saved successfully!` }

    if (autoFetchOnline.value && (editChessCom.value.trim() || editLichess.value.trim())) {
      await handleFetchOnlineGames(saved.id)
    }
  } catch (err: any) {
    statusMessage.value = { type: 'error', text: `Save failed: ${err?.message || err}` }
  }
}

async function handleDelete() {
  if (!selectedPersona.value?.id) return
  if (!confirm(`Are you sure you want to delete persona "${selectedPersona.value.displayName}"?`)) return

  try {
    await DeletePersona(selectedPersona.value.id)
    statusMessage.value = { type: 'success', text: 'Persona deleted.' }
    selectedPersona.value = null
    await loadPersonas()
  } catch (err: any) {
    statusMessage.value = { type: 'error', text: `Delete failed: ${err?.message || err}` }
  }
}

async function handleAddIdentifier(ident?: string) {
  const val = ident || newIdentifierInput.value.trim()
  if (!val) return

  if (selectedPersona.value && selectedPersona.value.id) {
    try {
      const updated = await AddPersonaIdentifier(selectedPersona.value.id, val)
      selectedPersona.value = updated
      newIdentifierInput.value = ''
      statusMessage.value = { type: 'success', text: `Added identifier "${val}"` }
      await loadPersonas()
      fetchSuggestions()
      fetchStatistics()
    } catch (err: any) {
      statusMessage.value = { type: 'error', text: `Could not add identifier: ${err?.message || err}` }
    }
  } else if (selectedPersona.value) {
    if (!selectedPersona.value.identifiers.includes(val)) {
      selectedPersona.value.identifiers.push(val)
      newIdentifierInput.value = ''
    }
  }
}

async function handleRemoveIdentifier(ident: string) {
  if (!selectedPersona.value) return

  if (selectedPersona.value.id) {
    try {
      const updated = await RemovePersonaIdentifier(selectedPersona.value.id, ident)
      selectedPersona.value = updated
      statusMessage.value = { type: 'info', text: `Removed identifier "${ident}"` }
      await loadPersonas()
      fetchSuggestions()
      fetchStatistics()
    } catch (err: any) {
      statusMessage.value = { type: 'error', text: `Could not remove identifier: ${err?.message || err}` }
    }
  } else {
    selectedPersona.value.identifiers = selectedPersona.value.identifiers.filter(i => i !== ident)
  }
}

async function fetchSuggestions() {
  if (!selectedPersona.value) return
  loadingSuggestions.value = true
  try {
    const list = await SuggestPersonaIdentifiers(
      editDisplayName.value,
      selectedPersona.value.identifiers || []
    )
    suggestedAliases.value = list || []
  } catch (err) {
    suggestedAliases.value = []
  } finally {
    loadingSuggestions.value = false
  }
}

async function fetchStatistics() {
  if (!selectedPersona.value?.id) {
    personaStats.value = null
    return
  }
  loadingStats.value = true
  try {
    const stats = await GetPersonaStatistics(selectedPersona.value.id)
    personaStats.value = stats
  } catch (err) {
    personaStats.value = null
  } finally {
    loadingStats.value = false
  }
}

async function handleAggregateGames() {
  if (!selectedPersona.value?.id) return
  isAggregating.value = true
  statusMessage.value = null
  try {
    const res = await AggregatePersonaGames(selectedPersona.value.id)
    await loadPersonas()
    const updated = personasList.value.find(p => p.id === selectedPersona.value?.id)
    if (updated) selectPersona(updated, true)
    statusMessage.value = {
      type: 'success',
      text: `Aggregated ${res.newlyAdded} new game(s) into dedicated database (${res.duplicatesSkipped} duplicates skipped). Total in persona base: ${res.totalPersonaGames} games.`
    }
  } catch (err: any) {
    statusMessage.value = { type: 'error', text: `Aggregation failed: ${err?.message || err}` }
  } finally {
    isAggregating.value = false
  }
}

async function handleFetchOnlineGames(targetPersonaId?: string) {
  const pId = targetPersonaId || selectedPersona.value?.id
  if (!pId) return
  if (!editChessCom.value.trim() && !editLichess.value.trim()) {
    statusMessage.value = {
      type: 'error',
      text: 'Please specify a Chess.com or Lichess username before fetching online games.'
    }
    return
  }

  isFetchingOnline.value = true
  statusMessage.value = {
    type: 'info',
    text: 'Connecting to Chess.com & Lichess to download games...'
  }

  try {
    const res = await FetchPersonaOnlineGames(pId, 0)
    let summary = `Fetched ${res.imported} game(s) into persona database (${res.chessComGames} Chess.com, ${res.lichessGames} Lichess)`
    if (res.duplicatesSkipped > 0) {
      summary += `, ${res.duplicatesSkipped} duplicates skipped`
    }
    if (res.errors && res.errors.length > 0) {
      summary += `. Note: ${res.errors.join('; ')}`
    }

    await loadPersonas()
    const updated = personasList.value.find(p => p.id === pId)
    if (updated) {
      selectPersona(updated, true)
    } else {
      fetchStatistics()
    }

    statusMessage.value = {
      type: res.imported > 0 || res.totalDownloaded > 0 ? 'success' : 'info',
      text: summary
    }
  } catch (err: any) {
    statusMessage.value = {
      type: 'error',
      text: `Online fetch failed: ${err?.message || err}`
    }
  } finally {
    isFetchingOnline.value = false
  }
}

async function handleOpenDatabase() {
  if (!selectedPersona.value?.id) return
  try {
    await OpenPersonaDatabase(selectedPersona.value.id)
    emit('openPersonaDb', selectedPersona.value)
    emit('close')
  } catch (err: any) {
    statusMessage.value = { type: 'error', text: `Could not open database: ${err?.message || err}` }
  }
}

function handleStartSparring() {
  if (!selectedPersona.value) return
  emit('sparPersona', selectedPersona.value)
  emit('close')
}
</script>

<template>
  <div class="fixed inset-0 z-50 bg-black/70 backdrop-blur-sm flex items-center justify-center p-4">
    <div class="modal-card bg-zinc-900 border border-zinc-700/80 rounded-2xl w-full max-w-4xl shadow-2xl overflow-hidden flex flex-col max-h-[90vh]">
      <!-- Header -->
      <div class="flex items-center justify-between p-4 bg-zinc-950 border-b border-zinc-800">
        <div class="flex items-center gap-2.5">
          <div class="w-8 h-8 rounded-xl bg-emerald-500/20 border border-emerald-500/40 flex items-center justify-center text-emerald-400">
            <Users class="w-4 h-4" />
          </div>
          <div>
            <h2 class="text-base font-bold text-zinc-100 flex items-center gap-2">
              <span>Player Personas & Unified Designators</span>
              <span class="px-2 py-0.5 rounded-full text-[10px] font-mono bg-emerald-950/80 border border-emerald-800/60 text-emerald-300">
                Multi-Alias Consolidation
              </span>
            </h2>
            <p class="text-xs text-zinc-400">
              Unify multiple designators (OTB, Chess.com, Lichess) under a single persona with dedicated database archiving.
            </p>
          </div>
        </div>

        <div class="flex items-center gap-2">
          <button
            class="px-3 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white text-xs font-semibold flex items-center gap-1.5 transition-colors cursor-pointer"
            @click="handleCreateNew"
          >
            <Plus class="w-3.5 h-3.5" />
            <span>New Persona</span>
          </button>
          <button
            class="p-1.5 rounded-lg text-zinc-400 hover:text-zinc-200 hover:bg-zinc-800 transition-colors cursor-pointer"
            @click="emit('close')"
          >
            <X class="w-5 h-5" />
          </button>
        </div>
      </div>

      <!-- Main Layout: Left Personas List / Right Persona Editor -->
      <div class="flex-1 flex overflow-hidden">
        <!-- Left Sidebar: Persona List -->
        <div class="w-72 bg-zinc-950/60 border-r border-zinc-800 flex flex-col shrink-0">
          <div class="p-3 border-b border-zinc-800/80">
            <div class="relative">
              <Search class="w-3.5 h-3.5 absolute left-2.5 top-2.5 text-zinc-500" />
              <input
                v-model="searchFilter"
                type="text"
                placeholder="Filter personas..."
                class="w-full bg-zinc-900 border border-zinc-700/80 rounded-lg pl-8 pr-2.5 py-1.5 text-xs text-zinc-200 placeholder-zinc-500 focus:outline-hidden focus:border-emerald-500"
              />
            </div>
          </div>

          <div class="flex-1 overflow-y-auto p-2 space-y-1.5">
            <div
              v-for="p in filteredPersonas"
              :key="p.id"
              class="p-2.5 rounded-xl border transition-all cursor-pointer flex items-center gap-2.5"
              :class="selectedPersona?.id === p.id
                ? 'bg-emerald-950/40 border-emerald-500/50 shadow-xs'
                : 'bg-zinc-900/60 border-zinc-800/80 hover:bg-zinc-800/60 hover:border-zinc-700'"
              @click="selectPersona(p)"
            >
              <div
                class="w-8 h-8 rounded-lg flex items-center justify-center font-bold text-xs text-white shrink-0"
                :style="{ backgroundColor: p.avatarColor || '#10b981' }"
              >
                {{ p.displayName.charAt(0).toUpperCase() }}
              </div>
              <div class="min-w-0 flex-1">
                <div class="flex items-center justify-between">
                  <span class="font-bold text-xs text-zinc-200 truncate">{{ p.displayName }}</span>
                  <span v-if="p.title" class="text-[10px] px-1 rounded bg-zinc-800 text-zinc-400 font-mono">
                    {{ p.title }}
                  </span>
                </div>
                <div class="flex items-center gap-2 text-[10px] text-zinc-400 mt-0.5">
                  <span>{{ p.identifiers?.length || 0 }} aliases</span>
                  <span v-if="p.estimatedElo" class="text-emerald-400 font-mono">~{{ p.estimatedElo }}</span>
                </div>
              </div>
            </div>

            <div v-if="filteredPersonas.length === 0" class="p-6 text-center text-xs text-zinc-500">
              No personas found.
            </div>
          </div>
        </div>

        <!-- Right Content: Persona Details & Alias Manager -->
        <div v-if="selectedPersona" class="flex-1 overflow-y-auto p-5 space-y-5 bg-zinc-900/40">
          <!-- Status Banner -->
          <div
            v-if="statusMessage"
            class="p-3 rounded-xl text-xs flex items-center gap-2 animate-fade-in"
            :class="{
              'bg-emerald-950/80 border border-emerald-800/60 text-emerald-200': statusMessage.type === 'success',
              'bg-rose-950/80 border border-rose-800/60 text-rose-200': statusMessage.type === 'error',
              'bg-blue-950/80 border border-blue-800/60 text-blue-200': statusMessage.type === 'info',
            }"
          >
            <CheckCircle2 v-if="statusMessage.type === 'success'" class="w-4 h-4 text-emerald-400 shrink-0" />
            <AlertCircle v-else class="w-4 h-4 text-rose-400 shrink-0" />
            <span class="flex-1">{{ statusMessage.text }}</span>
          </div>

          <!-- Section 1: Core Persona Identity -->
          <div class="bg-zinc-950/70 border border-zinc-800/90 rounded-xl p-4 space-y-3">
            <h3 class="text-xs font-bold text-zinc-300 uppercase tracking-wider flex items-center gap-1.5">
              <User class="w-3.5 h-3.5 text-emerald-400" />
              <span>Persona Identity</span>
            </h3>

            <div class="grid grid-cols-3 gap-3">
              <div class="col-span-2 space-y-1">
                <label class="text-[11px] font-semibold text-zinc-400">Unified Display Name</label>
                <input
                  v-model="editDisplayName"
                  type="text"
                  placeholder="e.g. Christopher Fair or Garry Kasparov"
                  class="w-full bg-zinc-900 border border-zinc-700/80 rounded-lg px-3 py-1.5 text-xs text-zinc-100 focus:outline-hidden focus:border-emerald-500"
                />
              </div>
              <div class="space-y-1">
                <label class="text-[11px] font-semibold text-zinc-400">Title / Subtitle</label>
                <input
                  v-model="editTitle"
                  type="text"
                  placeholder="e.g. GM, IM, NM, Club Player"
                  class="w-full bg-zinc-900 border border-zinc-700/80 rounded-lg px-3 py-1.5 text-xs text-zinc-100 focus:outline-hidden focus:border-emerald-500"
                />
              </div>
            </div>

            <div class="grid grid-cols-4 gap-3">
              <div class="space-y-1">
                <label class="text-[11px] font-semibold text-zinc-400">Estimated Elo</label>
                <input
                  v-model="editEstimatedElo"
                  type="number"
                  min="600"
                  max="3500"
                  class="w-full bg-zinc-900 border border-zinc-700/80 rounded-lg px-3 py-1.5 text-xs text-zinc-100 font-mono focus:outline-hidden focus:border-emerald-500"
                />
              </div>
              <div class="space-y-1">
                <label class="text-[11px] font-semibold text-zinc-400">Chess.com Username</label>
                <input
                  v-model="editChessCom"
                  type="text"
                  placeholder="e.g. ChristopherFair"
                  class="w-full bg-zinc-900 border border-zinc-700/80 rounded-lg px-3 py-1.5 text-xs text-zinc-100 focus:outline-hidden focus:border-emerald-500"
                />
              </div>
              <div class="space-y-1">
                <label class="text-[11px] font-semibold text-zinc-400">Lichess Username</label>
                <input
                  v-model="editLichess"
                  type="text"
                  placeholder="e.g. QuenyenHighElf"
                  class="w-full bg-zinc-900 border border-zinc-700/80 rounded-lg px-3 py-1.5 text-xs text-zinc-100 focus:outline-hidden focus:border-emerald-500"
                />
              </div>
              <div class="space-y-1">
                <label class="text-[11px] font-semibold text-zinc-400">FIDE / USCF ID</label>
                <input
                  v-model="editFideId"
                  type="text"
                  placeholder="e.g. 15000123"
                  class="w-full bg-zinc-900 border border-zinc-700/80 rounded-lg px-3 py-1.5 text-xs text-zinc-100 focus:outline-hidden focus:border-emerald-500"
                />
              </div>
            </div>

            <!-- Online Auto-fetch Option -->
            <div class="flex items-center justify-between pt-2 border-t border-zinc-800/80 text-xs">
              <label class="flex items-center gap-2 text-zinc-300 cursor-pointer select-none">
                <input
                  v-model="autoFetchOnline"
                  type="checkbox"
                  class="rounded border-zinc-700 bg-zinc-900 text-emerald-500 focus:ring-emerald-500 focus:ring-offset-zinc-900"
                />
                <span class="text-zinc-300 text-[11px] font-medium flex items-center gap-1.5">
                  <Globe class="w-3.5 h-3.5 text-emerald-400" />
                  <span>Auto-fetch online games from Chess.com & Lichess on save</span>
                </span>
              </label>
              <span v-if="editChessCom || editLichess" class="text-[10px] text-emerald-400 font-mono">
                Handles auto-link to persona identifiers & dedicated base
              </span>
            </div>
          </div>

          <!-- Section 2: Unified Identifiers & Designators (Aliases) -->
          <div class="bg-zinc-950/70 border border-zinc-800/90 rounded-xl p-4 space-y-3">
            <div class="flex items-center justify-between">
              <h3 class="text-xs font-bold text-zinc-300 uppercase tracking-wider flex items-center gap-1.5">
                <Sparkles class="w-3.5 h-3.5 text-amber-400" />
                <span>Database Designators & Aliases ({{ selectedPersona.identifiers?.length || 0 }})</span>
              </h3>
              <span class="text-[11px] text-zinc-500">Games matching ANY of these belong to this Persona</span>
            </div>

            <!-- Active Aliases Pills -->
            <div class="flex flex-wrap gap-1.5 min-h-[36px] p-2 bg-zinc-900/80 border border-zinc-800 rounded-lg">
              <div
                v-for="alias in selectedPersona.identifiers"
                :key="alias"
                class="px-2.5 py-1 rounded-full text-xs bg-zinc-800 border border-zinc-700 text-zinc-200 flex items-center gap-1.5 hover:border-zinc-500 transition-colors"
              >
                <span class="font-mono">{{ alias }}</span>
                <button
                  class="text-zinc-500 hover:text-rose-400 transition-colors cursor-pointer"
                  title="Remove alias"
                  @click="handleRemoveIdentifier(alias)"
                >
                  <X class="w-3 h-3" />
                </button>
              </div>
              <span v-if="!selectedPersona.identifiers || selectedPersona.identifiers.length === 0" class="text-xs text-zinc-500 italic py-1 px-1">
                No designators attached. Add aliases below to unify games under this persona.
              </span>
            </div>

            <!-- Add Custom Alias -->
            <div class="flex items-center gap-2">
              <input
                v-model="newIdentifierInput"
                type="text"
                placeholder="Enter exact player name as found in database (e.g. Fair, Christopher)..."
                class="flex-1 bg-zinc-900 border border-zinc-700/80 rounded-lg px-3 py-1.5 text-xs text-zinc-100 placeholder-zinc-500 focus:outline-hidden focus:border-emerald-500"
                @keyup.enter="handleAddIdentifier()"
              />
              <button
                class="px-3 py-1.5 rounded-lg bg-zinc-800 hover:bg-zinc-700 text-zinc-200 text-xs font-medium border border-zinc-700 flex items-center gap-1 transition-colors cursor-pointer"
                @click="handleAddIdentifier()"
              >
                <Plus class="w-3.5 h-3.5" />
                <span>Add Alias</span>
              </button>
            </div>

            <!-- Suggested Database Matches -->
            <div v-if="suggestedAliases.length > 0" class="pt-2 border-t border-zinc-800/80 space-y-2">
              <div class="flex items-center justify-between text-[11px] text-zinc-400">
                <span class="font-semibold flex items-center gap-1">
                  <Database class="w-3 h-3 text-emerald-400" />
                  <span>Found Candidate Names in Open Database:</span>
                </span>
                <span>Click to link</span>
              </div>
              <div class="flex flex-wrap gap-1.5 max-h-24 overflow-y-auto">
                <button
                  v-for="suggested in suggestedAliases"
                  :key="suggested"
                  class="px-2 py-0.5 rounded-md text-[11px] font-mono bg-emerald-950/40 hover:bg-emerald-900/60 border border-emerald-800/50 hover:border-emerald-600 text-emerald-300 flex items-center gap-1 transition-colors cursor-pointer"
                  @click="handleAddIdentifier(suggested)"
                >
                  <Plus class="w-3 h-3 text-emerald-400" />
                  <span>{{ suggested }}</span>
                </button>
              </div>
            </div>
          </div>

          <!-- Section 3: Dedicated Persona Database & Extraction -->
          <div class="bg-zinc-950/70 border border-zinc-800/90 rounded-xl p-4 space-y-3">
            <div class="flex items-center justify-between">
              <h3 class="text-xs font-bold text-zinc-300 uppercase tracking-wider flex items-center gap-1.5">
                <Database class="w-3.5 h-3.5 text-blue-400" />
                <span>Dedicated Persona Database (.si4)</span>
              </h3>
              <span class="text-[11px] text-zinc-500">Autonomous repository for all games featuring this persona</span>
            </div>

            <div class="p-3 bg-zinc-900/80 border border-zinc-800 rounded-lg flex items-center justify-between text-xs">
              <div class="space-y-0.5 min-w-0 pr-3">
                <span class="text-[11px] text-zinc-400 font-semibold block">Dedicated Database Path:</span>
                <span class="font-mono text-zinc-200 truncate block text-[11px]">
                  {{ selectedPersona.databasePath || 'Will be auto-created upon aggregation' }}
                </span>
              </div>

              <div class="flex items-center gap-2 shrink-0">
                <button
                  class="px-3 py-1.5 rounded-lg bg-zinc-800 hover:bg-zinc-700 text-zinc-200 text-xs font-medium border border-zinc-700 flex items-center gap-1.5 transition-colors cursor-pointer"
                  title="Open this persona's dedicated database into NewScid"
                  @click="handleOpenDatabase"
                >
                  <FolderOpen class="w-3.5 h-3.5 text-blue-400" />
                  <span>Open Database</span>
                </button>

                <button
                  class="px-3 py-1.5 rounded-lg bg-blue-600 hover:bg-blue-500 text-white text-xs font-semibold flex items-center gap-1.5 transition-colors cursor-pointer disabled:opacity-50"
                  :disabled="isAggregating"
                  title="Extract all games matching any of the persona's aliases from active database into dedicated base"
                  @click="handleAggregateGames"
                >
                  <RefreshCw class="w-3.5 h-3.5" :class="{ 'animate-spin': isAggregating }" />
                  <span>{{ isAggregating ? 'Aggregating...' : 'Sync from Active Base' }}</span>
                </button>

                <button
                  class="px-3 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white text-xs font-semibold flex items-center gap-1.5 transition-colors cursor-pointer disabled:opacity-50"
                  :disabled="isFetchingOnline || (!editChessCom && !editLichess)"
                  title="Fetch games directly from Chess.com and Lichess into dedicated database"
                  @click="handleFetchOnlineGames()"
                >
                  <Globe class="w-3.5 h-3.5" :class="{ 'animate-spin': isFetchingOnline }" />
                  <span>{{ isFetchingOnline ? 'Fetching Online...' : 'Fetch Online Games' }}</span>
                </button>
              </div>
            </div>
          </div>

          <!-- Section 4: Consolidated Statistics & Alias Breakdown -->
          <div v-if="personaStats && personaStats.totalGames > 0" class="bg-zinc-950/70 border border-zinc-800/90 rounded-xl p-4 space-y-3">
            <h3 class="text-xs font-bold text-zinc-300 uppercase tracking-wider flex items-center gap-1.5">
              <Trophy class="w-3.5 h-3.5 text-amber-400" />
              <span>Consolidated Record ({{ personaStats.totalGames }} Games Across All Aliases)</span>
            </h3>

            <div class="grid grid-cols-4 gap-2 text-center text-xs">
              <div class="p-2.5 bg-zinc-900/80 rounded-lg border border-zinc-800">
                <span class="text-[10px] text-zinc-500 uppercase font-semibold block">Total Games</span>
                <span class="text-base font-bold text-zinc-100 font-mono">{{ personaStats.totalGames }}</span>
              </div>
              <div class="p-2.5 bg-zinc-900/80 rounded-lg border border-zinc-800">
                <span class="text-[10px] text-zinc-500 uppercase font-semibold block">Score Record</span>
                <span class="text-base font-bold text-emerald-400 font-mono">
                  +{{ personaStats.wins }} ={{ personaStats.draws }} -{{ personaStats.losses }}
                </span>
              </div>
              <div class="p-2.5 bg-zinc-900/80 rounded-lg border border-zinc-800">
                <span class="text-[10px] text-zinc-500 uppercase font-semibold block">Win Rate</span>
                <span class="text-base font-bold text-zinc-100 font-mono">{{ personaStats.winRate.toFixed(1) }}%</span>
              </div>
              <div class="p-2.5 bg-zinc-900/80 rounded-lg border border-zinc-800">
                <span class="text-[10px] text-zinc-500 uppercase font-semibold block">Peak Elo</span>
                <span class="text-base font-bold text-purple-400 font-mono">{{ personaStats.peakElo || '?' }}</span>
              </div>
            </div>

            <!-- Per-Alias Breakdown -->
            <div v-if="personaStats.identifierStats?.length > 0" class="space-y-1.5">
              <span class="text-[11px] font-semibold text-zinc-400 block">Games by Database Designator:</span>
              <div class="border border-zinc-800 rounded-lg overflow-hidden text-xs">
                <table class="w-full text-left">
                  <thead class="bg-zinc-900 text-zinc-400 text-[10px] font-mono border-b border-zinc-800">
                    <tr>
                      <th class="p-2 font-semibold">Designator / Alias</th>
                      <th class="p-2 font-semibold text-center">Games</th>
                      <th class="p-2 font-semibold text-center">Wins</th>
                      <th class="p-2 font-semibold text-center">Draws</th>
                      <th class="p-2 font-semibold text-center">Losses</th>
                      <th class="p-2 font-semibold text-right">Win Rate</th>
                    </tr>
                  </thead>
                  <tbody class="divide-y divide-zinc-800/60 font-mono text-[11px]">
                    <tr v-for="istat in personaStats.identifierStats" :key="istat.identifier" class="hover:bg-zinc-800/40">
                      <td class="p-2 font-semibold text-zinc-200">{{ istat.identifier }}</td>
                      <td class="p-2 text-center text-zinc-300">{{ istat.gameCount }}</td>
                      <td class="p-2 text-center text-emerald-400">{{ istat.wins }}</td>
                      <td class="p-2 text-center text-zinc-400">{{ istat.draws }}</td>
                      <td class="p-2 text-center text-rose-400">{{ istat.losses }}</td>
                      <td class="p-2 text-right text-zinc-200">{{ istat.winRate.toFixed(1) }}%</td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </div>
          </div>

          <!-- Section 5: Modal Actions (Save, Spar, Delete) -->
          <div class="flex items-center justify-between pt-2 border-t border-zinc-800">
            <button
              v-if="selectedPersona.id"
              class="px-3 py-1.5 rounded-lg bg-zinc-800 hover:bg-rose-950/80 text-zinc-400 hover:text-rose-300 text-xs font-medium border border-zinc-700 hover:border-rose-800 transition-colors flex items-center gap-1.5 cursor-pointer"
              @click="handleDelete"
            >
              <Trash2 class="w-3.5 h-3.5" />
              <span>Delete Persona</span>
            </button>
            <div v-else />

            <div class="flex items-center gap-2">
              <button
                v-if="selectedPersona.id"
                class="px-3.5 py-1.5 rounded-lg bg-purple-600 hover:bg-purple-500 text-white text-xs font-semibold flex items-center gap-1.5 transition-colors cursor-pointer"
                title="Spar against AI bot simulating this unified persona"
                @click="handleStartSparring"
              >
                <Bot class="w-3.5 h-3.5" />
                <span>Spar As/Against Persona</span>
              </button>

              <button
                class="px-4 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white text-xs font-semibold flex items-center gap-1.5 transition-colors cursor-pointer shadow-md"
                @click="handleSave"
              >
                <Save class="w-3.5 h-3.5" />
                <span>Save Persona</span>
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
