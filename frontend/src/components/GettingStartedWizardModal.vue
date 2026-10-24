<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import {
  UserCheck,
  Database,
  BookOpen,
  Users,
  Compass,
  CheckCircle2,
  ChevronRight,
  ChevronLeft,
  Sparkles,
  Globe,
  FolderOpen,
  Plus,
  X,
  Zap,
  Target,
  Swords,
  Cpu,
  Shield,
  RefreshCw,
} from '@lucide/vue'
import {
  GetUserProfileStatus,
  SaveUserProfileSetup,
  SyncUserHomeGames,
  ImportStandardDatabase,
  OpenFileDialog,
  SelectAndImportWorkstationAsset,
  GetRepertoires,
  SaveRepertoire,
  GetPolyglotBooks,
  CompileRepertoireToPolyglotBook,
  GetPersonas,
  SavePersona,
  FetchPersonaOnlineGames,
  AggregatePersonaGames,
  BuildPersonaOpeningBook,
  EvaluatePersonaWeaknesses,
  ExportAntiPersonaPolyglotBook,
  AssignPersonaBooks,
  GetDiscoveredDatabases,
  SetReferenceDatabase,
} from '../../wailsjs/go/main/App'

const props = defineProps<{
  initialStep?: number
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'profileUpdated', status: any): void
  (e: 'openHomeDB'): void
  (e: 'openOpeningTrainer'): void
  (e: 'openPortal', platform: 'lichess' | 'chesscom'): void
  (e: 'openSimulatePlayer'): void
  (e: 'openPersonas'): void
  (e: 'openMegaCatalog'): void
}>()

const currentStep = ref(props.initialStep || 1)
const isSaving = ref(false)
const statusBanner = ref<string | null>(null)
const errorBanner = ref<string | null>(null)

// Step 1: Player Profile State
const displayName = ref('')
const playerTitle = ref('Club Player')
const estimatedElo = ref(1500)
const lichessUsername = ref('')
const chessComUsername = ref('')
const fideId = ref('')
const customAliases = ref<string[]>([])
const newAliasInput = ref('')
const homeDbName = ref('My_Games')
const userPersonaId = ref('')

// Step 2: Home DB & Migration State
const homeDbPath = ref('')
const homeDbGames = ref(0)
const referenceDbPath = ref('')
const referenceDbName = ref('')
const referenceDbGames = ref(0)
const discoveredDbs = ref<any[]>([])
const syncMaxGames = ref(250)
const isSyncingOnline = ref(false)
const pgnSourcePath = ref('')
const pgnPasteText = ref('')
const isImportingPgn = ref(false)

// Step 3: Focus Books & Repertoires State
const repertoires = ref<any[]>([])
const polyglotBooks = ref<string[]>([])
const focusWhiteRepId = ref('white-openings-italian')
const focusBlackRepId = ref('black-openings-sicilian-kid')
const focusBookPath = ref('')
const newRepName = ref('')
const newRepColor = ref<'w' | 'b'>('w')
const newRepDesc = ref('')
const isCompilingBook = ref(false)

// Step 4: Opponent Personas & Persona Books State
const personas = ref<any[]>([])
const selectedPersonaId = ref('')
const newOpponentName = ref('')
const newOpponentLichess = ref('')
const newOpponentChessCom = ref('')
const newOpponentElo = ref(1600)
const isWorkingPersona = ref(false)

const steps = [
  { id: 1, title: '1. My Player Profile', subtitle: 'Identity & Handles', icon: UserCheck },
  { id: 2, title: '2. Home Database', subtitle: 'Setup & Migrate Games', icon: Database },
  { id: 3, title: '3. My Focus Books', subtitle: 'Repertoire & .bin Book', icon: BookOpen },
  { id: 4, title: '4. Persona Books', subtitle: 'Opponents & Bots', icon: Users },
  { id: 5, title: '5. Quick-Start Guide', subtitle: 'Daily Workflows', icon: Compass },
]

const autoPreviewAliases = computed(() => {
  const set = new Set<string>()
  const name = displayName.value.trim()
  if (name) {
    set.add(name)
    if (!name.includes(',')) {
      const parts = name.split(/\s+/)
      if (parts.length >= 2) {
        const last = parts[parts.length - 1]
        const first = parts.slice(0, -1).join(' ')
        set.add(`${last}, ${first}`)
        if (first.length > 0) {
          set.add(`${last}, ${first[0]}`)
        }
      }
    }
  }
  if (lichessUsername.value.trim()) set.add(lichessUsername.value.trim())
  if (chessComUsername.value.trim()) set.add(chessComUsername.value.trim())
  for (const a of customAliases.value) {
    if (a.trim()) set.add(a.trim())
  }
  return Array.from(set)
})

const whiteRepertoires = computed(() =>
  repertoires.value.filter(r => !r.color || r.color.toLowerCase().startsWith('w'))
)
const blackRepertoires = computed(() =>
  repertoires.value.filter(r => r.color && r.color.toLowerCase().startsWith('b'))
)

const selectedPersona = computed(() =>
  personas.value.find(p => p.id === selectedPersonaId.value) || null
)

function formatBookBasename(p: string) {
  if (!p) return 'None'
  return p.split(/[/\\]/).pop() || p
}

async function loadAllWizardData() {
  try {
    const [status, reps, books, pers, dbs] = await Promise.all([
      GetUserProfileStatus().catch(() => null),
      GetRepertoires().catch(() => []),
      GetPolyglotBooks().catch(() => []),
      GetPersonas().catch(() => []),
      GetDiscoveredDatabases().catch(() => []),
    ])

    if (status) {
      displayName.value = status.displayName || ''
      playerTitle.value = status.title || 'Club Player'
      estimatedElo.value = status.estimatedElo || 1500
      lichessUsername.value = status.lichessUsername || ''
      chessComUsername.value = status.chessComUsername || ''
      fideId.value = status.fideId || ''
      customAliases.value = Array.isArray(status.aliases) ? [...status.aliases] : []
      userPersonaId.value = status.personaId || ''
      homeDbPath.value = status.homeDbPath || ''
      if (status.homeDbName) homeDbName.value = status.homeDbName
      homeDbGames.value = status.homeDbGames || 0
      referenceDbPath.value = status.referenceDbPath || ''
      referenceDbName.value = status.referenceDbName || ''
      referenceDbGames.value = status.referenceDbGames || 0
      if (status.focusWhiteRepertoireId) focusWhiteRepId.value = status.focusWhiteRepertoireId
      if (status.focusBlackRepertoireId) focusBlackRepId.value = status.focusBlackRepertoireId
      if (status.focusPolyglotBookPath) focusBookPath.value = status.focusPolyglotBookPath
    }

    repertoires.value = reps || []
    polyglotBooks.value = books || []
    if (!focusBookPath.value && polyglotBooks.value.length > 0) {
      focusBookPath.value = polyglotBooks.value[0]
    }

    personas.value = pers || []
    if (!selectedPersonaId.value && personas.value.length > 0) {
      const nonSelf = personas.value.find((p: any) => !p.isUserProfile) || personas.value[0]
      if (nonSelf) selectedPersonaId.value = nonSelf.id
    }

    discoveredDbs.value = dbs || []
  } catch (err) {
    console.error('Failed to load wizard data:', err)
  }
}

function addCustomAlias() {
  const val = newAliasInput.value.trim()
  if (!val) return
  if (!customAliases.value.includes(val)) {
    customAliases.value.push(val)
  }
  newAliasInput.value = ''
}

function removeCustomAlias(alias: string) {
  customAliases.value = customAliases.value.filter(a => a !== alias)
}

async function saveProfileState(markComplete = false) {
  isSaving.value = true
  errorBanner.value = null
  try {
    const res = await SaveUserProfileSetup({
      displayName: displayName.value.trim(),
      title: playerTitle.value.trim() || 'Club Player',
      estimatedElo: Number(estimatedElo.value) || 1500,
      lichessUsername: lichessUsername.value.trim(),
      chessComUsername: chessComUsername.value.trim(),
      fideId: fideId.value.trim(),
      aliases: autoPreviewAliases.value,
      homeDatabaseName: homeDbName.value.trim() || 'My_Games',
      ensureHomeDb: true,
      focusWhiteRepertoireId: focusWhiteRepId.value,
      focusBlackRepertoireId: focusBlackRepId.value,
      focusPolyglotBookPath: focusBookPath.value,
      completeOnboarding: markComplete,
    })
    if (res) {
      userPersonaId.value = res.personaId || userPersonaId.value
      homeDbPath.value = res.homeDbPath || homeDbPath.value
      homeDbGames.value = res.homeDbGames ?? homeDbGames.value
      emit('profileUpdated', res)
    }
    const pers = await GetPersonas().catch(() => [])
    personas.value = pers || []
    return res
  } catch (err: any) {
    errorBanner.value = err?.message || String(err)
    return null
  } finally {
    isSaving.value = false
  }
}

async function handleNextStep() {
  statusBanner.value = null
  errorBanner.value = null
  if (currentStep.value === 1 || currentStep.value === 3) {
    await saveProfileState(false)
  }
  if (currentStep.value < 5) {
    currentStep.value++
  } else {
    await handleFinishWizard()
  }
}

async function handleFinishWizard() {
  await saveProfileState(true)
  emit('close')
}

// Step 2 Handlers: Online Sync, PGN Import, Reference DB
async function handleSyncOnlineIntoHome() {
  errorBanner.value = null
  statusBanner.value = null
  await saveProfileState(false)
  if (!lichessUsername.value.trim() && !chessComUsername.value.trim()) {
    errorBanner.value = 'Enter your Lichess or Chess.com username in Step 1 first so NewScid knows which account to sync.'
    return
  }
  isSyncingOnline.value = true
  try {
    const res: any = await SyncUserHomeGames(Number(syncMaxGames.value) || 250)
    await loadAllWizardData()
    statusBanner.value = `Synced online games into ${homeDbName.value}: +${res?.imported || 0} new games added (${res?.duplicatesSkipped || 0} duplicates skipped).`
  } catch (err: any) {
    errorBanner.value = `Online sync error: ${err?.message || String(err)}`
  } finally {
    isSyncingOnline.value = false
  }
}

async function handleBrowseMigrationFile() {
  try {
    const chosen = await OpenFileDialog()
    if (chosen) {
      pgnSourcePath.value = chosen
    }
  } catch (err: any) {
    errorBanner.value = err?.message || String(err)
  }
}

async function handleMigrateFileIntoHome() {
  errorBanner.value = null
  statusBanner.value = null
  if (!pgnSourcePath.value.trim() && !pgnPasteText.value.trim()) {
    errorBanner.value = 'Select a .pgn / .si4 file or paste PGN text first.'
    return
  }
  isImportingPgn.value = true
  try {
    const res: any = await ImportStandardDatabase({
      sourcePath: pgnSourcePath.value.trim(),
      pgnText: pgnPasteText.value.trim(),
      importMode: 'user_db',
      archiveName: '',
      masterDbName: 'Master_Reference',
      userDbName: homeDbName.value.trim() || 'My_Games',
      setMasterAsRef: false,
      setUserAsDefault: true,
      openAfterImport: true,
    } as any)
    pgnPasteText.value = ''
    await loadAllWizardData()
    statusBanner.value = res?.summary || `Imported games into ${homeDbName.value}!`
  } catch (err: any) {
    errorBanner.value = `Import error: ${err?.message || String(err)}`
  } finally {
    isImportingPgn.value = false
  }
}

async function handleSelectReferenceDb(basePath: string) {
  if (!basePath) return
  try {
    await SetReferenceDatabase(basePath, true)
    await loadAllWizardData()
    statusBanner.value = `Set "${formatBookBasename(basePath)}" as your background Reference Database for the Opening Tree.`
  } catch (err: any) {
    errorBanner.value = err?.message || String(err)
  }
}

// Step 3 Handlers: Repertoires & Polyglot Books
async function handleCreateQuickRepertoire() {
  const name = newRepName.value.trim()
  if (!name) {
    errorBanner.value = 'Enter a name for your new repertoire.'
    return
  }
  errorBanner.value = null
  try {
    const slug = `${newRepColor.value}-${name.toLowerCase().replace(/[^a-z0-9]+/g, '-')}`
    await SaveRepertoire({
      id: slug,
      name,
      color: newRepColor.value,
      description: newRepDesc.value.trim() || `Personal ${newRepColor.value === 'w' ? 'White' : 'Black'} Focus Repertoire`,
      positions: {},
    } as any)
    repertoires.value = await GetRepertoires()
    if (newRepColor.value === 'w') {
      focusWhiteRepId.value = slug
    } else {
      focusBlackRepId.value = slug
    }
    newRepName.value = ''
    newRepDesc.value = ''
    await saveProfileState(false)
    statusBanner.value = `Created repertoire "${name}" and set it as your ${newRepColor.value === 'w' ? 'White' : 'Black'} focus!`
  } catch (err: any) {
    errorBanner.value = err?.message || String(err)
  }
}

async function handleCompileRepToBook(repId: string) {
  if (!repId) return
  isCompilingBook.value = true
  errorBanner.value = null
  statusBanner.value = null
  try {
    const savedPath: any = await CompileRepertoireToPolyglotBook(repId, true)
    polyglotBooks.value = await GetPolyglotBooks()
    if (typeof savedPath === 'string' && savedPath) {
      focusBookPath.value = savedPath
    }
    await saveProfileState(false)
    statusBanner.value = `Compiled Focus Repertoire into Polyglot opening book: ${formatBookBasename(String(savedPath))}`
  } catch (err: any) {
    errorBanner.value = err?.message || String(err)
  } finally {
    isCompilingBook.value = false
  }
}

async function handleCompileMyGamesToBook() {
  isCompilingBook.value = true
  errorBanner.value = null
  statusBanner.value = null
  try {
    const st = await saveProfileState(false)
    const pid = st?.personaId || userPersonaId.value
    if (!pid) {
      errorBanner.value = 'Set your Display Name in Step 1 first so NewScid can compile your personal opening book.'
      return
    }
    const savedPath: any = await BuildPersonaOpeningBook(pid, 'both', 20)
    polyglotBooks.value = await GetPolyglotBooks()
    if (typeof savedPath === 'string' && savedPath) {
      focusBookPath.value = savedPath
      await saveProfileState(false)
    }
    statusBanner.value = `Compiled your personal Polyglot book from your games: ${formatBookBasename(String(savedPath))}`
  } catch (err: any) {
    errorBanner.value = err?.message || String(err)
  } finally {
    isCompilingBook.value = false
  }
}

async function handleImportExternalBook() {
  errorBanner.value = null
  try {
    const asset = await SelectAndImportWorkstationAsset('opening_books')
    if (asset && asset.path) {
      polyglotBooks.value = await GetPolyglotBooks()
      focusBookPath.value = asset.path
      await saveProfileState(false)
      statusBanner.value = `Imported "${asset.name}" and set it as your active Focus Polyglot Book!`
    }
  } catch (err: any) {
    errorBanner.value = err?.message || String(err)
  }
}

// Step 4 Handlers: Opponent Personas & Persona Books
async function handleCreateOpponentPersona() {
  const name = newOpponentName.value.trim()
  if (!name) {
    errorBanner.value = 'Enter the opponent or player name.'
    return
  }
  isWorkingPersona.value = true
  errorBanner.value = null
  statusBanner.value = null
  try {
    const saved = await SavePersona({
      id: '',
      displayName: name,
      identifiers: [name],
      databasePath: '',
      title: 'Opponent',
      bio: 'Created via Getting Started Setup Wizard',
      avatarColor: '',
      estimatedElo: Number(newOpponentElo.value) || 1600,
      chessComUsername: newOpponentChessCom.value.trim(),
      lichessUsername: newOpponentLichess.value.trim(),
      fideId: '',
      createdAt: 0,
      updatedAt: 0,
    } as any)
    personas.value = await GetPersonas()
    if (saved && saved.id) {
      selectedPersonaId.value = saved.id
    }
    newOpponentName.value = ''
    newOpponentLichess.value = ''
    newOpponentChessCom.value = ''
    statusBanner.value = `Created Persona "${name}" with dedicated database!`
  } catch (err: any) {
    errorBanner.value = err?.message || String(err)
  } finally {
    isWorkingPersona.value = false
  }
}

async function handleFetchPersonaOnline(personaId: string) {
  if (!personaId) return
  isWorkingPersona.value = true
  errorBanner.value = null
  statusBanner.value = null
  try {
    const res = await FetchPersonaOnlineGames(personaId, 200)
    statusBanner.value = `Fetched +${res?.imported || 0} online games into Persona database!`
  } catch (err: any) {
    errorBanner.value = err?.message || String(err)
  } finally {
    isWorkingPersona.value = false
  }
}

async function handleExtractPersonaFromOpenDB(personaId: string) {
  if (!personaId) return
  isWorkingPersona.value = true
  errorBanner.value = null
  statusBanner.value = null
  try {
    const res = await AggregatePersonaGames(personaId)
    statusBanner.value = `Extracted +${res?.newlyAdded || 0} games (${res?.totalPersonaGames || 0} total) into Persona database!`
  } catch (err: any) {
    errorBanner.value = err?.message || String(err)
  } finally {
    isWorkingPersona.value = false
  }
}

async function handleBuildPersonaOwnBook(personaId: string) {
  if (!personaId) return
  isWorkingPersona.value = true
  errorBanner.value = null
  statusBanner.value = null
  try {
    const savedPath: any = await BuildPersonaOpeningBook(personaId, 'both', 20)
    personas.value = await GetPersonas()
    polyglotBooks.value = await GetPolyglotBooks()
    statusBanner.value = `Built Persona Opening Book (.bin): ${formatBookBasename(String(savedPath))} and attached to Persona!`
  } catch (err: any) {
    errorBanner.value = err?.message || String(err)
  } finally {
    isWorkingPersona.value = false
  }
}

async function handleBuildAntiPersonaBook(personaId: string) {
  if (!personaId) return
  isWorkingPersona.value = true
  errorBanner.value = null
  statusBanner.value = null
  try {
    const rep = await EvaluatePersonaWeaknesses(personaId, 'both', 16, 1)
    const savedPath = await ExportAntiPersonaPolyglotBook(rep, '')
    personas.value = await GetPersonas()
    polyglotBooks.value = await GetPolyglotBooks()
    statusBanner.value = `Compiled Anti-Persona Counter-Book: ${formatBookBasename(savedPath)} (${rep?.polyglotEntryCount || 0} positions)!`
  } catch (err: any) {
    errorBanner.value = err?.message || String(err)
  } finally {
    isWorkingPersona.value = false
  }
}

async function handleAssignPersonaBooks(p: any) {
  if (!p || !p.id) return
  errorBanner.value = null
  try {
    await AssignPersonaBooks(
      p.id,
      p.bookPath || '',
      p.antiBookPath || '',
      p.whiteRepertoireId || '',
      p.blackRepertoireId || ''
    )
    personas.value = await GetPersonas()
    statusBanner.value = `Updated assigned books & repertoires for ${p.displayName}!`
  } catch (err: any) {
    errorBanner.value = err?.message || String(err)
  }
}

onMounted(() => {
  loadAllWizardData()
})
</script>

<template>
  <div
    class="fixed inset-0 z-50 bg-black/80 backdrop-blur-xs flex items-center justify-center p-3"
    data-testid="getting-started-wizard-modal"
    @click.self="emit('close')"
  >
    <div class="bg-zinc-900 border border-zinc-700/80 rounded-2xl shadow-2xl w-full max-w-4xl max-h-[92vh] flex flex-col overflow-hidden text-xs">
      <!-- Top Header -->
      <div class="px-5 py-3.5 bg-zinc-950 border-b border-zinc-800 flex items-center justify-between gap-3">
        <div class="flex items-center gap-2.5">
          <div class="w-9 h-9 rounded-xl bg-emerald-500/15 border border-emerald-500/30 flex items-center justify-center text-emerald-400">
            <Sparkles class="w-5 h-5" />
          </div>
          <div>
            <h2 class="text-base font-extrabold text-white tracking-tight">
              Getting Started &amp; Player Setup Wizard
            </h2>
            <p class="text-[11px] text-zinc-400">
              Set up your Player Identity, Home Database, Focus Opening Book, and Opponent Personas in one place
            </p>
          </div>
        </div>
        <button
          data-testid="wizard-close-btn"
          class="p-1.5 rounded-lg text-zinc-400 hover:text-white hover:bg-zinc-800 transition-colors cursor-pointer"
          title="Close Setup Wizard"
          @click="emit('close')"
        >
          <X class="w-4 h-4" />
        </button>
      </div>

      <!-- Stepper Navigation Bar -->
      <div class="px-4 py-2.5 bg-zinc-950/60 border-b border-zinc-800/80 grid grid-cols-5 gap-1.5">
        <button
          v-for="s in steps"
          :key="s.id"
          :data-testid="`wizard-step-tab-${s.id}`"
          class="px-2.5 py-2 rounded-xl border text-left transition-all cursor-pointer flex items-center gap-2"
          :class="
            currentStep === s.id
              ? 'bg-emerald-950/50 border-emerald-500/60 text-emerald-200 shadow-xs'
              : currentStep > s.id
              ? 'bg-zinc-900/80 border-zinc-700 text-zinc-200 hover:bg-zinc-800'
              : 'bg-zinc-950/60 border-zinc-800/80 text-zinc-400 hover:bg-zinc-900 hover:text-zinc-200'
          "
          @click="currentStep = s.id"
        >
          <component
            :is="currentStep > s.id ? CheckCircle2 : s.icon"
            class="w-4 h-4 shrink-0"
            :class="currentStep === s.id ? 'text-emerald-400' : currentStep > s.id ? 'text-emerald-500' : 'text-zinc-500'"
          />
          <div class="min-w-0">
            <div class="font-bold truncate text-[11px]">{{ s.title }}</div>
            <div class="text-[10px] text-zinc-400 truncate">{{ s.subtitle }}</div>
          </div>
        </button>
      </div>

      <!-- Status / Error Banners -->
      <div v-if="statusBanner" class="mx-5 mt-3 px-3.5 py-2 rounded-xl bg-emerald-950/60 border border-emerald-500/40 text-emerald-200 flex items-center justify-between gap-2">
        <div class="flex items-center gap-2">
          <CheckCircle2 class="w-4 h-4 text-emerald-400 shrink-0" />
          <span>{{ statusBanner }}</span>
        </div>
        <button class="text-emerald-300 hover:text-white cursor-pointer" @click="statusBanner = null">✕</button>
      </div>
      <div v-if="errorBanner" class="mx-5 mt-3 px-3.5 py-2 rounded-xl bg-rose-950/60 border border-rose-500/40 text-rose-200 flex items-center justify-between gap-2">
        <span>{{ errorBanner }}</span>
        <button class="text-rose-300 hover:text-white cursor-pointer" @click="errorBanner = null">✕</button>
      </div>

      <!-- Step Content Body -->
      <div class="flex-1 overflow-y-auto p-5 space-y-4">
        <!-- ========================================================= -->
        <!-- STEP 1: MY PLAYER PROFILE (WHO AM I?)                     -->
        <!-- ========================================================= -->
        <div v-if="currentStep === 1" class="space-y-4" data-testid="wizard-step-1">
          <div class="p-3.5 rounded-xl bg-emerald-950/25 border border-emerald-500/30 space-y-1">
            <h3 class="text-sm font-bold text-emerald-300 flex items-center gap-2">
              <UserCheck class="w-4 h-4" />
              <span>Why Define Your Player Identity?</span>
            </h3>
            <p class="text-zinc-300 leading-relaxed">
              In PGN files and online platforms, the same player often appears under multiple strings (e.g.
              <code class="text-emerald-300">Fair, Christopher</code>, <code class="text-emerald-300">Christopher Fair</code>, and your Lichess/Chess.com usernames).
              Setting up your profile creates your <strong>Personal Persona</strong>, links your online handles for <strong>automatic game capture</strong>, and initializes your <strong>Home Database</strong>.
            </p>
          </div>

          <div class="grid grid-cols-1 md:grid-cols-3 gap-3.5">
            <div class="md:col-span-2 space-y-1">
              <label class="block text-[11px] font-bold text-zinc-300">Your Full Name / Display Name</label>
              <input
                v-model="displayName"
                data-testid="wizard-display-name-input"
                type="text"
                placeholder="e.g. Christopher Fair"
                class="w-full px-3 py-2 rounded-lg bg-zinc-950 border border-zinc-700 text-white focus:border-emerald-500 focus:outline-hidden"
              />
            </div>

            <div class="grid grid-cols-2 gap-2">
              <div class="space-y-1">
                <label class="block text-[11px] font-bold text-zinc-300">Title / Rank</label>
                <select
                  v-model="playerTitle"
                  class="w-full px-2.5 py-2 rounded-lg bg-zinc-950 border border-zinc-700 text-white focus:border-emerald-500 focus:outline-hidden"
                >
                  <option value="Club Player">Club Player</option>
                  <option value="Tournament Player">Tournament Player</option>
                  <option value="NM">NM</option>
                  <option value="CM">CM</option>
                  <option value="FM">FM</option>
                  <option value="IM">IM</option>
                  <option value="GM">GM</option>
                  <option value="WFM">WFM</option>
                  <option value="WIM">WIM</option>
                  <option value="WGM">WGM</option>
                </select>
              </div>
              <div class="space-y-1">
                <label class="block text-[11px] font-bold text-zinc-300">Est. Rating (Elo)</label>
                <input
                  v-model.number="estimatedElo"
                  type="number"
                  min="400"
                  max="3500"
                  step="50"
                  class="w-full px-2.5 py-2 rounded-lg bg-zinc-950 border border-zinc-700 text-white focus:border-emerald-500 focus:outline-hidden"
                />
              </div>
            </div>
          </div>

          <div class="grid grid-cols-1 md:grid-cols-3 gap-3.5">
            <div class="space-y-1">
              <label class="block text-[11px] font-bold text-zinc-300">Lichess.org Username (Optional)</label>
              <input
                v-model="lichessUsername"
                data-testid="wizard-lichess-input"
                type="text"
                placeholder="e.g. chrisfair"
                class="w-full px-3 py-2 rounded-lg bg-zinc-950 border border-zinc-700 text-white focus:border-emerald-500 focus:outline-hidden"
              />
            </div>
            <div class="space-y-1">
              <label class="block text-[11px] font-bold text-zinc-300">Chess.com Username (Optional)</label>
              <input
                v-model="chessComUsername"
                data-testid="wizard-chesscom-input"
                type="text"
                placeholder="e.g. chrisfair"
                class="w-full px-3 py-2 rounded-lg bg-zinc-950 border border-zinc-700 text-white focus:border-emerald-500 focus:outline-hidden"
              />
            </div>
            <div class="space-y-1">
              <label class="block text-[11px] font-bold text-zinc-300">FIDE / USCF ID (Optional)</label>
              <input
                v-model="fideId"
                type="text"
                placeholder="e.g. 12345678"
                class="w-full px-3 py-2 rounded-lg bg-zinc-950 border border-zinc-700 text-white focus:border-emerald-500 focus:outline-hidden"
              />
            </div>
          </div>

          <!-- PGN Name Aliases Preview & Custom Adder -->
          <div class="p-3.5 rounded-xl bg-zinc-950 border border-zinc-800 space-y-2.5">
            <div class="flex items-center justify-between">
              <div>
                <div class="font-bold text-zinc-200">Unified PGN Name Variations &amp; Aliases</div>
                <div class="text-[11px] text-zinc-400">
                  NewScid automatically matches all of these names in databases as <strong>you</strong>:
                </div>
              </div>
              <div class="flex items-center gap-1.5">
                <input
                  v-model="newAliasInput"
                  type="text"
                  placeholder="Add extra alias / old handle..."
                  class="px-2.5 py-1 rounded-lg bg-zinc-900 border border-zinc-700 text-white text-xs"
                  @keydown.enter.prevent="addCustomAlias"
                />
                <button
                  type="button"
                  class="px-2.5 py-1 rounded-lg bg-zinc-800 hover:bg-zinc-700 text-zinc-200 font-semibold cursor-pointer"
                  @click="addCustomAlias"
                >
                  + Add
                </button>
              </div>
            </div>

            <div class="flex flex-wrap gap-1.5">
              <span
                v-for="alias in autoPreviewAliases"
                :key="alias"
                class="px-2.5 py-1 rounded-full bg-emerald-950/60 border border-emerald-500/40 text-emerald-200 font-medium flex items-center gap-1.5"
              >
                <span>{{ alias }}</span>
                <button
                  v-if="customAliases.includes(alias)"
                  type="button"
                  class="text-emerald-400 hover:text-white cursor-pointer"
                  @click="removeCustomAlias(alias)"
                >
                  ✕
                </button>
              </span>
              <span v-if="autoPreviewAliases.length === 0" class="text-zinc-500 italic">
                Type your name or online usernames above to generate your unified PGN aliases...
              </span>
            </div>
          </div>
        </div>

        <!-- ========================================================= -->
        <!-- STEP 2: HOME DATABASE & GAME MIGRATION                    -->
        <!-- ========================================================= -->
        <div v-if="currentStep === 2" class="space-y-4" data-testid="wizard-step-2">
          <!-- 3-Tier Explanation Diagram -->
          <div class="grid grid-cols-1 md:grid-cols-3 gap-2.5">
            <div class="p-3 rounded-xl bg-emerald-950/30 border border-emerald-500/40 space-y-1">
              <div class="font-extrabold text-emerald-300 flex items-center justify-between">
                <span>1. Home DB ({{ homeDbName }})</span>
                <span class="px-1.5 py-0.5 rounded bg-emerald-500/20 text-[10px]">EDITABLE</span>
              </div>
              <p class="text-[11px] text-zinc-300">
                Your personal database for <strong>your own games</strong>, annotations, and study variations. Opens automatically at startup.
              </p>
              <div class="text-[11px] font-bold text-emerald-400 pt-1">
                Currently: {{ homeDbGames }} games
              </div>
            </div>

            <div class="p-3 rounded-xl bg-cyan-950/25 border border-cyan-500/30 space-y-1">
              <div class="font-extrabold text-cyan-300 flex items-center justify-between">
                <span>2. Reference DB (Theory)</span>
                <span class="px-1.5 py-0.5 rounded bg-cyan-500/20 text-[10px]">SYNC-ONLY</span>
              </div>
              <p class="text-[11px] text-zinc-300">
                Large Master/TWIC archive that powers your <strong>Opening Tree</strong> in the background while you keep your Home DB open.
              </p>
              <div class="text-[11px] font-bold text-cyan-400 pt-1">
                Currently: {{ referenceDbName ? `${referenceDbName} (${referenceDbGames} games)` : 'Not assigned yet' }}
              </div>
            </div>

            <div class="p-3 rounded-xl bg-purple-950/25 border border-purple-500/30 space-y-1">
              <div class="font-extrabold text-purple-300 flex items-center justify-between">
                <span>3. Persona Databases</span>
                <span class="px-1.5 py-0.5 rounded bg-purple-500/20 text-[10px]">ISOLATED</span>
              </div>
              <p class="text-[11px] text-zinc-300">
                Each opponent, friend, or Grandmaster you study gets a dedicated database so their games never clutter your Home DB.
              </p>
              <div class="text-[11px] font-bold text-purple-400 pt-1">
                {{ personas.length }} Persona(s) configured
              </div>
            </div>
          </div>

          <!-- Migration Option A & B -->
          <div class="grid grid-cols-1 md:grid-cols-2 gap-3.5">
            <!-- Card A: Sync from Lichess / Chess.com -->
            <div class="p-4 rounded-xl bg-zinc-950 border border-zinc-800 space-y-3 flex flex-col justify-between">
              <div class="space-y-1.5">
                <h4 class="text-sm font-bold text-white flex items-center gap-2">
                  <Globe class="w-4 h-4 text-emerald-400" />
                  <span>Option A: Pull My Games from Lichess &amp; Chess.com</span>
                </h4>
                <p class="text-[11px] text-zinc-400">
                  Downloads your online games and merges them into your <strong>{{ homeDbName }}</strong> Home Database with automatic duplicate removal.
                </p>
                <div class="text-[11px] text-zinc-300 pt-1">
                  Linked Handles:
                  <strong class="text-emerald-300">{{ lichessUsername || 'No Lichess handle' }}</strong> /
                  <strong class="text-emerald-300">{{ chessComUsername || 'No Chess.com handle' }}</strong>
                </div>
              </div>

              <div class="flex items-center gap-2 pt-2">
                <select
                  v-model.number="syncMaxGames"
                  class="px-2.5 py-1.5 rounded-lg bg-zinc-900 border border-zinc-700 text-zinc-200"
                >
                  <option :value="100">Latest 100 games</option>
                  <option :value="250">Latest 250 games</option>
                  <option :value="500">Latest 500 games</option>
                  <option :value="1000">Latest 1000 games</option>
                </select>
                <button
                  type="button"
                  data-testid="wizard-sync-online-btn"
                  :disabled="isSyncingOnline"
                  class="flex-1 px-3 py-2 rounded-lg bg-emerald-600 hover:bg-emerald-500 disabled:opacity-50 text-white font-bold flex items-center justify-center gap-1.5 cursor-pointer transition-colors"
                  @click="handleSyncOnlineIntoHome"
                >
                  <RefreshCw class="w-3.5 h-3.5" :class="{ 'animate-spin': isSyncingOnline }" />
                  <span>{{ isSyncingOnline ? 'Syncing Games...' : `Sync Online Games into ${homeDbName}` }}</span>
                </button>
              </div>
            </div>

            <!-- Card B: Migrate from PGN or .si4 File -->
            <div class="p-4 rounded-xl bg-zinc-950 border border-zinc-800 space-y-2.5 flex flex-col justify-between">
              <div class="space-y-1.5">
                <h4 class="text-sm font-bold text-white flex items-center gap-2">
                  <FolderOpen class="w-4 h-4 text-amber-400" />
                  <span>Option B: Migrate Games from .pgn / .si4 File</span>
                </h4>
                <p class="text-[11px] text-zinc-400">
                  Import an existing PGN file, Scid <code>.si4</code> database, or pasted PGN games directly into your <strong>{{ homeDbName }}</strong> Home Database.
                </p>

                <div class="flex gap-1.5 pt-1">
                  <input
                    v-model="pgnSourcePath"
                    type="text"
                    placeholder="Path to .pgn or .si4 file..."
                    class="flex-1 px-2.5 py-1.5 rounded-lg bg-zinc-900 border border-zinc-700 text-white"
                  />
                  <button
                    type="button"
                    class="px-2.5 py-1.5 rounded-lg bg-zinc-800 hover:bg-zinc-700 text-zinc-200 font-semibold cursor-pointer"
                    @click="handleBrowseMigrationFile"
                  >
                    Browse...
                  </button>
                </div>

                <textarea
                  v-model="pgnPasteText"
                  rows="2"
                  placeholder="Or paste raw PGN games here..."
                  class="w-full px-2.5 py-1.5 rounded-lg bg-zinc-900 border border-zinc-700 text-white font-mono text-[11px]"
                />
              </div>

              <button
                type="button"
                data-testid="wizard-import-pgn-btn"
                :disabled="isImportingPgn"
                class="w-full px-3 py-2 rounded-lg bg-amber-600 hover:bg-amber-500 disabled:opacity-50 text-white font-bold cursor-pointer transition-colors"
                @click="handleMigrateFileIntoHome"
              >
                {{ isImportingPgn ? 'Importing into Home DB...' : `Import & Deduplicate into ${homeDbName}` }}
              </button>
            </div>
          </div>

          <!-- Reference DB Assignment Bar -->
          <div class="p-3.5 rounded-xl bg-zinc-950 border border-zinc-800 flex flex-wrap items-center justify-between gap-3">
            <div class="space-y-0.5">
              <div class="font-bold text-cyan-300">Background Reference Database (For Opening Tree Statistics)</div>
              <div class="text-[11px] text-zinc-400">
                Select a large database for Opening Tree theory, or download master archives from the Online Catalog:
              </div>
            </div>
            <div class="flex items-center gap-2 flex-wrap">
              <select
                :value="referenceDbPath"
                class="px-2.5 py-1.5 rounded-lg bg-zinc-900 border border-zinc-700 text-zinc-200"
                @change="handleSelectReferenceDb(($event.target as HTMLSelectElement).value)"
              >
                <option value="">— Select Local Database as Reference —</option>
                <option v-for="db in discoveredDbs" :key="db.basePath" :value="db.basePath">
                  {{ db.name }} ({{ db.numGames }} games)
                </option>
              </select>
              <button
                type="button"
                class="px-3 py-1.5 rounded-lg bg-cyan-600/20 hover:bg-cyan-600/30 border border-cyan-500/40 text-cyan-300 font-bold cursor-pointer"
                @click="emit('openMegaCatalog')"
              >
                🌐 Open Master Database Catalog...
              </button>
            </div>
          </div>
        </div>

        <!-- ========================================================= -->
        <!-- STEP 3: MY FOCUS BOOKS & REPERTOIRES                      -->
        <!-- ========================================================= -->
        <div v-if="currentStep === 3" class="space-y-4" data-testid="wizard-step-3">
          <div class="p-3.5 rounded-xl bg-indigo-950/25 border border-indigo-500/30 space-y-1">
            <h3 class="text-sm font-bold text-indigo-300 flex items-center gap-2">
              <BookOpen class="w-4 h-4" />
              <span>How "Books" Work for Your Own Study</span>
            </h3>
            <p class="text-zinc-300 leading-relaxed">
              NewScid supports two kinds of books for your personal preparation:
              <strong>1) Focus Repertoires</strong> (your curated White &amp; Black move trees that you drill with spaced repetition), and
              <strong>2) Polyglot Opening Books (<code>.bin</code>)</strong> (fast reference move-weight tables displayed in the Opening Tree).
              You can also compile any Focus Repertoire or your own games into a <code>.bin</code> book with one click!
            </p>
          </div>

          <div class="grid grid-cols-1 md:grid-cols-2 gap-3.5">
            <!-- White & Black Focus Repertoire Selection -->
            <div class="p-4 rounded-xl bg-zinc-950 border border-zinc-800 space-y-3">
              <h4 class="text-sm font-bold text-white">1. My Focus Training Repertoires</h4>

              <div class="space-y-1">
                <label class="block text-[11px] font-bold text-zinc-300">♔ White Focus Repertoire</label>
                <div class="flex gap-1.5">
                  <select
                    v-model="focusWhiteRepId"
                    data-testid="wizard-white-rep-select"
                    class="flex-1 px-2.5 py-1.5 rounded-lg bg-zinc-900 border border-zinc-700 text-white"
                  >
                    <option v-for="r in whiteRepertoires" :key="r.id" :value="r.id">
                      {{ r.name }} ({{ r.numMoves }} moves)
                    </option>
                  </select>
                  <button
                    type="button"
                    data-testid="wizard-compile-white-rep-btn"
                    class="px-2.5 py-1.5 rounded-lg bg-indigo-600/20 hover:bg-indigo-600/30 border border-indigo-500/40 text-indigo-300 font-semibold cursor-pointer"
                    title="Compile this White Repertoire into a Polyglot .bin book"
                    @click="handleCompileRepToBook(focusWhiteRepId)"
                  >
                    ⚡ To .bin
                  </button>
                </div>
              </div>

              <div class="space-y-1">
                <label class="block text-[11px] font-bold text-zinc-300">♚ Black Focus Repertoire</label>
                <div class="flex gap-1.5">
                  <select
                    v-model="focusBlackRepId"
                    data-testid="wizard-black-rep-select"
                    class="flex-1 px-2.5 py-1.5 rounded-lg bg-zinc-900 border border-zinc-700 text-white"
                  >
                    <option v-for="r in blackRepertoires" :key="r.id" :value="r.id">
                      {{ r.name }} ({{ r.numMoves }} moves)
                    </option>
                  </select>
                  <button
                    type="button"
                    class="px-2.5 py-1.5 rounded-lg bg-indigo-600/20 hover:bg-indigo-600/30 border border-indigo-500/40 text-indigo-300 font-semibold cursor-pointer"
                    title="Compile this Black Repertoire into a Polyglot .bin book"
                    @click="handleCompileRepToBook(focusBlackRepId)"
                  >
                    ⚡ To .bin
                  </button>
                </div>
              </div>

              <!-- Create a New Repertoire inline -->
              <div class="pt-2 border-t border-zinc-800/80 space-y-2">
                <div class="text-[11px] font-bold text-zinc-300">+ Create a New Focus Repertoire</div>
                <div class="flex gap-1.5">
                  <select
                    v-model="newRepColor"
                    class="px-2 py-1.5 rounded-lg bg-zinc-900 border border-zinc-700 text-white"
                  >
                    <option value="w">White</option>
                    <option value="b">Black</option>
                  </select>
                  <input
                    v-model="newRepName"
                    data-testid="wizard-new-rep-name"
                    type="text"
                    placeholder="e.g. My Queen's Gambit & London"
                    class="flex-1 px-2.5 py-1.5 rounded-lg bg-zinc-900 border border-zinc-700 text-white"
                  />
                  <button
                    type="button"
                    data-testid="wizard-create-rep-btn"
                    class="px-3 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white font-bold cursor-pointer"
                    @click="handleCreateQuickRepertoire"
                  >
                    Create
                  </button>
                </div>
              </div>
            </div>

            <!-- Focus Polyglot Opening Book (.bin) -->
            <div class="p-4 rounded-xl bg-zinc-950 border border-zinc-800 space-y-3 flex flex-col justify-between">
              <div class="space-y-2">
                <h4 class="text-sm font-bold text-white">2. My Focus Polyglot Opening Book (.bin)</h4>
                <p class="text-[11px] text-zinc-400">
                  Selected automatically when you open the <strong>Opening Tree $\rightarrow$ Polyglot Book</strong> tab:
                </p>

                <select
                  v-model="focusBookPath"
                  data-testid="wizard-focus-book-select"
                  class="w-full px-2.5 py-2 rounded-lg bg-zinc-900 border border-zinc-700 text-white"
                >
                  <option value="">— No Default Polyglot .bin Book Selected —</option>
                  <option v-for="b in polyglotBooks" :key="b" :value="b">
                    {{ formatBookBasename(b) }} ({{ b }})
                  </option>
                </select>
              </div>

              <div class="space-y-2 pt-2 border-t border-zinc-800/80">
                <div class="text-[11px] font-bold text-zinc-300">Build or Import a Polyglot (.bin) Book:</div>
                <div class="grid grid-cols-1 gap-2">
                  <button
                    type="button"
                    data-testid="wizard-compile-my-games-book-btn"
                    :disabled="isCompilingBook"
                    class="w-full px-3 py-2 rounded-lg bg-purple-600/25 hover:bg-purple-600/35 border border-purple-500/40 text-purple-200 font-bold flex items-center justify-center gap-1.5 cursor-pointer transition-colors"
                    @click="handleCompileMyGamesToBook"
                  >
                    <Zap class="w-3.5 h-3.5 text-purple-400" />
                    <span>Compile Personal .bin Book from My Games ({{ homeDbName }})</span>
                  </button>
                  <button
                    type="button"
                    class="w-full px-3 py-2 rounded-lg bg-zinc-900 hover:bg-zinc-800 border border-zinc-700 text-zinc-200 font-semibold flex items-center justify-center gap-1.5 cursor-pointer transition-colors"
                    @click="handleImportExternalBook"
                  >
                    <FolderOpen class="w-3.5 h-3.5 text-amber-400" />
                    <span>Import External Polyglot (.bin) Book from Disk...</span>
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- ========================================================= -->
        <!-- STEP 4: PERSONA BOOKS & OPPONENT PREPARATION              -->
        <!-- ========================================================= -->
        <div v-if="currentStep === 4" class="space-y-4" data-testid="wizard-step-4">
          <div class="p-3.5 rounded-xl bg-purple-950/25 border border-purple-500/30 space-y-1">
            <h3 class="text-sm font-bold text-purple-300 flex items-center gap-2">
              <Users class="w-4 h-4" />
              <span>How to Define a Book for Any Persona</span>
            </h3>
            <p class="text-zinc-300 leading-relaxed">
              Every Persona (an opponent, club rival, or GM) can have <strong>two books</strong>:
              <strong>1) Their Personal Book</strong> (compiled from their games or assigned from a <code>.bin</code> book / repertoire so you can spar against their exact openings in <em>Simulate Player</em>), and
              <strong>2) An Anti-Persona Counter-Book (<code>Anti_&lt;Name&gt;.bin</code>)</strong> that targets the exact lines where they lose most often!
            </p>
          </div>

          <div class="grid grid-cols-1 md:grid-cols-3 gap-3.5">
            <!-- Left Column: Create or Pick a Persona -->
            <div class="p-3.5 rounded-xl bg-zinc-950 border border-zinc-800 space-y-3">
              <div class="font-bold text-white">Select or Add a Persona</div>

              <select
                v-model="selectedPersonaId"
                data-testid="wizard-persona-select"
                class="w-full px-2.5 py-2 rounded-lg bg-zinc-900 border border-zinc-700 text-white"
              >
                <option value="">— Select Persona —</option>
                <option v-for="p in personas" :key="p.id" :value="p.id">
                  {{ p.isUserProfile ? '👤 [ME] ' : '🎯 ' }}{{ p.displayName }} ({{ p.estimatedElo || 1500 }} Elo)
                </option>
              </select>

              <div class="pt-2 border-t border-zinc-800 space-y-2">
                <div class="text-[11px] font-bold text-zinc-300">+ Quick-Add Opponent Persona</div>
                <input
                  v-model="newOpponentName"
                  data-testid="wizard-new-opponent-name"
                  type="text"
                  placeholder="Opponent / GM Name..."
                  class="w-full px-2.5 py-1.5 rounded-lg bg-zinc-900 border border-zinc-700 text-white"
                />
                <div class="grid grid-cols-2 gap-1.5">
                  <input
                    v-model="newOpponentLichess"
                    type="text"
                    placeholder="Lichess handle"
                    class="px-2 py-1.5 rounded-lg bg-zinc-900 border border-zinc-700 text-white"
                  />
                  <input
                    v-model="newOpponentChessCom"
                    type="text"
                    placeholder="Chess.com handle"
                    class="px-2 py-1.5 rounded-lg bg-zinc-900 border border-zinc-700 text-white"
                  />
                </div>
                <button
                  type="button"
                  data-testid="wizard-create-opponent-btn"
                  :disabled="isWorkingPersona"
                  class="w-full px-3 py-1.5 rounded-lg bg-purple-600 hover:bg-purple-500 text-white font-bold flex items-center justify-center gap-1 cursor-pointer"
                  @click="handleCreateOpponentPersona"
                >
                  <Plus class="w-3.5 h-3.5" />
                  <span>Create Opponent Persona</span>
                </button>
              </div>
            </div>

            <!-- Right 2 Columns: Configure Books for Selected Persona -->
            <div v-if="selectedPersona" class="md:col-span-2 p-4 rounded-xl bg-zinc-950 border border-zinc-800 space-y-3">
              <div class="flex items-center justify-between border-b border-zinc-800 pb-2.5">
                <div>
                  <div class="text-sm font-extrabold text-white flex items-center gap-2">
                    <span>{{ selectedPersona.displayName }}</span>
                    <span v-if="selectedPersona.isUserProfile" class="px-2 py-0.5 rounded-full bg-emerald-500/20 text-emerald-300 text-[10px]">
                      MY PROFILE
                    </span>
                  </div>
                  <div class="text-[11px] text-zinc-400">
                    Aliases: {{ (selectedPersona.identifiers || []).join(', ') || selectedPersona.displayName }}
                  </div>
                </div>
              </div>

              <!-- Step A: Populate Their Games -->
              <div class="space-y-1.5">
                <div class="text-[11px] font-bold text-zinc-300">
                  Step A: Populate {{ selectedPersona.displayName }}'s Games (Feeds Their Historical Opening Tree)
                </div>
                <div class="flex flex-wrap gap-2">
                  <button
                    type="button"
                    :disabled="isWorkingPersona"
                    class="px-3 py-1.5 rounded-lg bg-zinc-900 hover:bg-zinc-800 border border-zinc-700 text-emerald-300 font-semibold cursor-pointer"
                    @click="handleFetchPersonaOnline(selectedPersona.id)"
                  >
                    ☁️ Fetch Their Lichess / Chess.com Games
                  </button>
                  <button
                    type="button"
                    :disabled="isWorkingPersona"
                    class="px-3 py-1.5 rounded-lg bg-zinc-900 hover:bg-zinc-800 border border-zinc-700 text-amber-300 font-semibold cursor-pointer"
                    @click="handleExtractPersonaFromOpenDB(selectedPersona.id)"
                  >
                    📥 Extract Their Games from Open DB
                  </button>
                </div>
              </div>

              <!-- Step B: Auto-Compile Their Opening Book (.bin) or Anti-Book (.bin) -->
              <div class="space-y-1.5 pt-2 border-t border-zinc-800/80">
                <div class="text-[11px] font-bold text-zinc-300">
                  Step B: One-Click Compile Polyglot Books (.bin) from {{ selectedPersona.displayName }}'s Games
                </div>
                <div class="grid grid-cols-1 sm:grid-cols-2 gap-2">
                  <button
                    type="button"
                    data-testid="wizard-build-persona-book-btn"
                    :disabled="isWorkingPersona"
                    class="px-3 py-2 rounded-lg bg-purple-600/25 hover:bg-purple-600/35 border border-purple-500/40 text-purple-200 font-bold flex items-center justify-center gap-1.5 cursor-pointer"
                    @click="handleBuildPersonaOwnBook(selectedPersona.id)"
                  >
                    <BookOpen class="w-3.5 h-3.5 text-purple-400" />
                    <span>Build Persona's Own Book (.bin)</span>
                  </button>
                  <button
                    type="button"
                    data-testid="wizard-build-antipersona-book-btn"
                    :disabled="isWorkingPersona"
                    class="px-3 py-2 rounded-lg bg-rose-600/25 hover:bg-rose-600/35 border border-rose-500/40 text-rose-200 font-bold flex items-center justify-center gap-1.5 cursor-pointer"
                    @click="handleBuildAntiPersonaBook(selectedPersona.id)"
                  >
                    <Target class="w-3.5 h-3.5 text-rose-400" />
                    <span>Build Anti-Persona Exploit Book (.bin)</span>
                  </button>
                </div>
              </div>

              <!-- Step C: Explicitly Assign Any Book or Repertoire to This Persona -->
              <div class="space-y-2 pt-2 border-t border-zinc-800/80">
                <div class="text-[11px] font-bold text-zinc-300">
                  Step C: Or Manually Assign a Specific Book / Repertoire to {{ selectedPersona.displayName }}
                </div>
                <div class="grid grid-cols-1 sm:grid-cols-3 gap-2">
                  <div>
                    <label class="block text-[10px] text-zinc-400 mb-0.5">Assigned Polyglot Book (.bin)</label>
                    <select
                      v-model="selectedPersona.bookPath"
                      class="w-full px-2 py-1.5 rounded-lg bg-zinc-900 border border-zinc-700 text-white"
                      @change="handleAssignPersonaBooks(selectedPersona)"
                    >
                      <option value="">(Use Historical Game Tree)</option>
                      <option v-for="b in polyglotBooks" :key="b" :value="b">
                        {{ formatBookBasename(b) }}
                      </option>
                    </select>
                  </div>
                  <div>
                    <label class="block text-[10px] text-zinc-400 mb-0.5">Assigned White Repertoire</label>
                    <select
                      v-model="selectedPersona.whiteRepertoireId"
                      class="w-full px-2 py-1.5 rounded-lg bg-zinc-900 border border-zinc-700 text-white"
                      @change="handleAssignPersonaBooks(selectedPersona)"
                    >
                      <option value="">(None)</option>
                      <option v-for="r in whiteRepertoires" :key="r.id" :value="r.id">
                        {{ r.name }}
                      </option>
                    </select>
                  </div>
                  <div>
                    <label class="block text-[10px] text-zinc-400 mb-0.5">Assigned Black Repertoire</label>
                    <select
                      v-model="selectedPersona.blackRepertoireId"
                      class="w-full px-2 py-1.5 rounded-lg bg-zinc-900 border border-zinc-700 text-white"
                      @change="handleAssignPersonaBooks(selectedPersona)"
                    >
                      <option value="">(None)</option>
                      <option v-for="r in blackRepertoires" :key="r.id" :value="r.id">
                        {{ r.name }}
                      </option>
                    </select>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- ========================================================= -->
        <!-- STEP 5: QUICK-START DAILY WORKFLOW GUIDE                  -->
        <!-- ========================================================= -->
        <div v-if="currentStep === 5" class="space-y-4" data-testid="wizard-step-5">
          <!-- Profile Summary Banner -->
          <div class="p-3.5 rounded-xl bg-emerald-950/30 border border-emerald-500/40 flex flex-wrap items-center justify-between gap-3">
            <div class="space-y-0.5">
              <div class="text-sm font-extrabold text-emerald-300">
                ✓ Workstation Ready for {{ displayName || 'Your Player Profile' }}
              </div>
              <div class="text-[11px] text-zinc-300">
                Home DB: <strong>{{ homeDbName }}</strong> ({{ homeDbGames }} games) ·
                Reference DB: <strong>{{ referenceDbName || 'None' }}</strong> ·
                Focus Book: <strong>{{ formatBookBasename(focusBookPath) }}</strong>
              </div>
            </div>
          </div>

          <div class="grid grid-cols-1 md:grid-cols-2 gap-3.5">
            <!-- Workflow 1 -->
            <div class="p-4 rounded-xl bg-zinc-950 border border-zinc-800 space-y-2.5 flex flex-col justify-between">
              <div class="space-y-1">
                <div class="text-sm font-bold text-white flex items-center gap-2">
                  <Globe class="w-4 h-4 text-emerald-400" />
                  <span>1. Play Online &amp; Auto-Capture</span>
                </div>
                <p class="text-[11px] text-zinc-400 leading-relaxed">
                  Launch a dedicated Lichess or Chess.com portal window. As soon as you finish a game, NewScid automatically imports it into <strong>{{ homeDbName }}</strong> and loads it on the board for review.
                </p>
              </div>
              <div class="flex gap-2">
                <button
                  type="button"
                  class="flex-1 px-3 py-1.5 rounded-lg bg-emerald-600/20 hover:bg-emerald-600/30 border border-emerald-500/40 text-emerald-300 font-bold cursor-pointer"
                  @click="handleFinishWizard(); emit('openPortal', 'lichess')"
                >
                  Launch Lichess Portal
                </button>
                <button
                  type="button"
                  class="flex-1 px-3 py-1.5 rounded-lg bg-amber-600/20 hover:bg-amber-600/30 border border-amber-500/40 text-amber-300 font-bold cursor-pointer"
                  @click="handleFinishWizard(); emit('openPortal', 'chesscom')"
                >
                  Launch Chess.com Portal
                </button>
              </div>
            </div>

            <!-- Workflow 2 -->
            <div class="p-4 rounded-xl bg-zinc-950 border border-zinc-800 space-y-2.5 flex flex-col justify-between">
              <div class="space-y-1">
                <div class="text-sm font-bold text-white flex items-center gap-2">
                  <Cpu class="w-4 h-4 text-cyan-400" />
                  <span>2. Review &amp; Annotate My Games</span>
                </div>
                <p class="text-[11px] text-zinc-400 leading-relaxed">
                  Open your <strong>{{ homeDbName }}</strong> database, run <strong>Annotate Game</strong> with Stockfish, and any blunders found are automatically added to your spaced-repetition <strong>Blunder Vault</strong>.
                </p>
              </div>
              <button
                type="button"
                class="w-full px-3 py-1.5 rounded-lg bg-cyan-600/20 hover:bg-cyan-600/30 border border-cyan-500/40 text-cyan-300 font-bold cursor-pointer"
                @click="handleFinishWizard(); emit('openHomeDB')"
              >
                Open My Home Database ({{ homeDbName }})
              </button>
            </div>

            <!-- Workflow 3 -->
            <div class="p-4 rounded-xl bg-zinc-950 border border-zinc-800 space-y-2.5 flex flex-col justify-between">
              <div class="space-y-1">
                <div class="text-sm font-bold text-white flex items-center gap-2">
                  <BookOpen class="w-4 h-4 text-indigo-400" />
                  <span>3. Build &amp; Drill My Focus Repertoire</span>
                </div>
                <p class="text-[11px] text-zinc-400 leading-relaxed">
                  Open the <strong>Opening Repertoire Trainer</strong> to add lines from the board or drill your White &amp; Black focus repertoires with automatic mastery tracking.
                </p>
              </div>
              <button
                type="button"
                class="w-full px-3 py-1.5 rounded-lg bg-indigo-600/20 hover:bg-indigo-600/30 border border-indigo-500/40 text-indigo-300 font-bold cursor-pointer"
                @click="handleFinishWizard(); emit('openOpeningTrainer')"
              >
                Open Repertoire Trainer
              </button>
            </div>

            <!-- Workflow 4 -->
            <div class="p-4 rounded-xl bg-zinc-950 border border-zinc-800 space-y-2.5 flex flex-col justify-between">
              <div class="space-y-1">
                <div class="text-sm font-bold text-white flex items-center gap-2">
                  <Swords class="w-4 h-4 text-purple-400" />
                  <span>4. Spar vs. Persona &amp; Scout Weaknesses</span>
                </div>
                <p class="text-[11px] text-zinc-400 leading-relaxed">
                  Play against any Persona's opening book and Elo-calibrated style in <strong>Simulate Player</strong>, or generate an <strong>Anti-Persona Book</strong> before a match.
                </p>
              </div>
              <div class="flex gap-2">
                <button
                  type="button"
                  class="flex-1 px-3 py-1.5 rounded-lg bg-purple-600/20 hover:bg-purple-600/30 border border-purple-500/40 text-purple-300 font-bold cursor-pointer"
                  @click="handleFinishWizard(); emit('openSimulatePlayer')"
                >
                  Spar vs. Persona Bot
                </button>
                <button
                  type="button"
                  class="flex-1 px-3 py-1.5 rounded-lg bg-zinc-800 hover:bg-zinc-700 text-zinc-200 font-bold cursor-pointer"
                  @click="handleFinishWizard(); emit('openPersonas')"
                >
                  Manage Personas
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Footer Navigation -->
      <div class="px-5 py-3 bg-zinc-950 border-t border-zinc-800 flex items-center justify-between gap-3">
        <button
          v-if="currentStep > 1"
          type="button"
          data-testid="wizard-prev-btn"
          class="px-3.5 py-2 rounded-xl bg-zinc-900 hover:bg-zinc-800 border border-zinc-700 text-zinc-200 font-semibold flex items-center gap-1.5 cursor-pointer"
          @click="currentStep--"
        >
          <ChevronLeft class="w-4 h-4" />
          <span>Previous Step</span>
        </button>
        <div v-else class="text-[11px] text-zinc-500">
          You can reopen this wizard anytime from the top bar or <strong>File $\rightarrow$ Getting Started Wizard</strong>.
        </div>

        <div class="flex items-center gap-2">
          <button
            type="button"
            data-testid="wizard-finish-btn"
            class="px-3.5 py-2 rounded-xl bg-zinc-900 hover:bg-zinc-800 border border-zinc-700 text-zinc-300 font-semibold cursor-pointer"
            @click="handleFinishWizard"
          >
            Save &amp; Close
          </button>
          <button
            v-if="currentStep < 5"
            type="button"
            data-testid="wizard-next-btn"
            :disabled="isSaving"
            class="px-4 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-500 text-white font-bold flex items-center gap-1.5 cursor-pointer shadow-xs"
            @click="handleNextStep"
          >
            <span>{{ isSaving ? 'Saving...' : 'Save & Continue' }}</span>
            <ChevronRight class="w-4 h-4" />
          </button>
          <button
            v-else
            type="button"
            data-testid="wizard-complete-btn"
            class="px-4 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-500 text-white font-bold flex items-center gap-1.5 cursor-pointer shadow-xs"
            @click="handleFinishWizard"
          >
            <CheckCircle2 class="w-4 h-4" />
            <span>Finish Setup &amp; Start Using NewScid</span>
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
