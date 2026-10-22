<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import {
  Settings,
  Database,
  LayoutGrid,
  Monitor,
  Globe,
  Palette,
  Cpu,
  Check,
  Star,
  BookOpen,
  ExternalLink,
  RotateCcw,
  Search,
  X,
  Copy,
  Sparkles,
  RefreshCw
} from '@lucide/vue'
import type { PieceSetOption, BoardThemeOption, DiscoveredDatabase } from './HeaderBar.vue'
import type { EngineOption } from './EnginePanel.vue'
import type { DatabaseRolesStatusView } from './DatabaseHubPanel.vue'
import {
  workspacePanels,
  setPanelMode,
  setPanelDockZone,
  setPanelWorkspace,
  applyLayoutPreset,
  resetWorkspaceLayout,
  type PanelId,
  type PanelMode,
  type DockZone,
  type LayoutPresetId
} from '../utils/workspaceLayout'
import { UI_THEMES } from '../utils/theme'
import { useNotation } from '../utils/notation'

export interface AppPreferencesModel {
  defaultDatabasePath: string
  referenceDatabasePath: string
  autoOpenDefaultDb: boolean
  useReferenceDbForTree: boolean
  defaultPieceSet: string
  defaultBoardTheme: string
  defaultUITheme: string
  defaultNotationStyle: string
  soundEnabled: boolean
  showEvalBar: boolean
  threatModeDefault: boolean
  blindfoldDefault?: boolean
  boardFlippedDefault?: boolean
  compactLayoutMode?: string
  leftColWidth?: number
  bottomDrawerHeight?: number
  bottomDrawerOpen?: boolean
  mainWindowWorkspace?: string
  autoRestoreWorkspaceWindows?: boolean
  panelConfigs?: Record<string, any>
  defaultEngine: string
  defaultMultiPv: number
  defaultThreads: number
  defaultHashMb: number
  annotateMoveTimeMs: number
  lichessUsername: string
  chessComUsername: string
  defaultPortalPlatform: string
  portalWindowMode: string
  customBrowserCmd: string
}

const props = defineProps<{
  initialSection?: string
  preferences: AppPreferencesModel
  rolesStatus: DatabaseRolesStatusView | null
  discoveredDatabases: DiscoveredDatabase[]
  pieceSets: PieceSetOption[]
  boardThemes: BoardThemeOption[]
  discoveredEngines: EngineOption[]
  currentPieceSet: string
  currentThemeId: string
  currentUITheme: string
  soundEnabled: boolean
  showEvalBar: boolean
  threatMode: boolean
  blindfold: boolean
  leftColWidth: number
  bottomDrawerHeight: number
  isUpdatingFromLocal?: boolean
  updatingDbPath?: string | null
  localMergeStatus?: string | null
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'savePreferences', prefs: AppPreferencesModel): void
  (e: 'setDefaultDB', path: string): void
  (e: 'setReferenceDB', path: string, useForTree: boolean): void
  (e: 'openDefaultDB'): void
  (e: 'openClipbase'): void
  (e: 'openDiscoveredBase', path: string): void
  (e: 'changePieceSet', id: string): void
  (e: 'changeTheme', id: string): void
  (e: 'changeUITheme', id: string): void
  (e: 'toggleSound'): void
  (e: 'toggleEvalBar'): void
  (e: 'toggleThreatMode'): void
  (e: 'toggleBlindfold'): void
  (e: 'updateLeftWidth', w: number): void
  (e: 'updateBottomHeight', h: number): void
  (e: 'detachPanelScreen', panelId: PanelId, workspace?: string): void
  (e: 'moveMainToWorkspace', workspace: string): void
  (e: 'updateFromLocal', targetPath?: string): void
  (e: 'cancelUpdateLocal'): void
  (e: 'launchPortalWindow', platform: 'lichess' | 'chesscom', url: string): void
}>()

const { currentStyle, setNotationStyle, options: notationOptions } = useNotation()

const activeSection = ref<string>(props.initialSection || 'databases')
const searchQuery = ref('')
const localPrefs = ref<AppPreferencesModel>({ ...props.preferences })
const savedBanner = ref(false)
const copiedHyprland = ref(false)

watch(
  () => props.preferences,
  (next) => {
    if (next) localPrefs.value = { ...next }
  },
  { deep: true }
)

const sections = [
  {
    id: 'databases',
    title: '1. Databases & Default DB',
    subtitle: 'Default startup DB, Reference DB & roles',
    icon: Database,
    keywords: 'default database reference active clipbase startup si4 pgn role',
  },
  {
    id: 'panels',
    title: '2. Panels, Docks & Floaters',
    subtitle: 'Customize, float, dock, resize or hide panels',
    icon: LayoutGrid,
    keywords: 'panels dockable floaters floating removable resize layout splitters',
  },
  {
    id: 'multiscreen',
    title: '3. Multi-Screen & Hyprland',
    subtitle: 'Detach panels across monitors & workspaces',
    icon: Monitor,
    keywords: 'hyprland multi screen monitor wayland windowrulev2 detach tile',
  },
  {
    id: 'portals',
    title: '4. Lichess & Chess.com Portals',
    subtitle: 'Direct windows, live TV & account bridges',
    icon: Globe,
    keywords: 'lichess chess.com portal window browser live tv online',
  },
  {
    id: 'appearance',
    title: '5. Board, Pieces & Appearance',
    subtitle: 'UI themes, board colors, piece sets & audio',
    icon: Palette,
    keywords: 'theme dark light pieces board sound eval bar blindfold notation',
  },
  {
    id: 'engines',
    title: '6. UCI Engines & Analysis',
    subtitle: 'Stockfish threads, hash, MultiPV & annotation',
    icon: Cpu,
    keywords: 'engine stockfish multipv threads hash annotation syzygy tablebase',
  },
]

const filteredSections = computed(() => {
  const q = searchQuery.value.trim().toLowerCase()
  if (!q) return sections
  return sections.filter(
    s =>
      s.title.toLowerCase().includes(q) ||
      s.subtitle.toLowerCase().includes(q) ||
      s.keywords.toLowerCase().includes(q)
  )
})

const panelsArray = computed(() => Object.values(workspacePanels.value))

const hyprlandConfigSnippet = `# ~/.config/hypr/hyprland.conf — NewScid Multi-Screen Rules
# Place main NewScid board on primary monitor (e.g. DP-1 / workspace 1)
windowrulev2 = workspace 1, title:^(NewScid - Modern Chess Information Database)$

# Place detached NewScid Screen Panels on secondary monitor (e.g. DP-2 / workspace 2)
windowrulev2 = monitor DP-2, title:^(NewScid — .* \\[Screen Panel\\])$
windowrulev2 = tile, title:^(NewScid — .* \\[Screen Panel\\])$

# Place Lichess.org / Chess.com Portal Windows on monitor 2 or 3
windowrulev2 = monitor DP-2, class:^(newscid-portal-.*)$
windowrulev2 = monitor DP-2, title:^(NewScid Live Portal.*)$`

function handleSave() {
  emit('savePreferences', { ...localPrefs.value })
  savedBanner.value = true
  setTimeout(() => {
    savedBanner.value = false
  }, 2500)
}

function handleDefaultDbSelect(e: Event) {
  const val = (e.target as HTMLSelectElement).value
  localPrefs.value.defaultDatabasePath = val
  emit('setDefaultDB', val)
  handleSave()
}

function handleReferenceDbSelect(e: Event) {
  const val = (e.target as HTMLSelectElement).value
  localPrefs.value.referenceDatabasePath = val
  emit('setReferenceDB', val, localPrefs.value.useReferenceDbForTree)
  handleSave()
}

async function copyHyprlandSnippet() {
  try {
    if (navigator.clipboard) {
      await navigator.clipboard.writeText(hyprlandConfigSnippet)
    }
    copiedHyprland.value = true
    setTimeout(() => {
      copiedHyprland.value = false
    }, 2500)
  } catch {
    // ignore
  }
}
</script>

<template>
  <div
    class="fixed inset-0 z-50 bg-black/75 backdrop-blur-xs flex items-center justify-center p-3 sm:p-6 animate-fade-in"
    @click.self="emit('close')"
  >
    <div class="bg-zinc-900 border border-zinc-700/90 rounded-2xl shadow-2xl w-full max-w-6xl h-[88vh] flex flex-col overflow-hidden text-xs">
      <!-- Top Document Header -->
      <div class="px-5 py-3.5 bg-zinc-950 border-b border-zinc-800 flex items-center justify-between shrink-0">
        <div class="flex items-center gap-3">
          <div class="w-9 h-9 rounded-xl bg-emerald-600/20 border border-emerald-500/40 flex items-center justify-center text-emerald-400">
            <Settings class="w-5 h-5" />
          </div>
          <div>
            <h2 class="text-base font-extrabold text-zinc-100 tracking-tight">
              NewScid Preferences &amp; Options Document
            </h2>
            <p class="text-[11px] text-zinc-400">
              Centralized reference &amp; configuration for Database Roles, Dockable/Floating Panels, Hyprland Multi-Screen, Lichess/Chess.com Portals, and Engines
            </p>
          </div>
        </div>

        <div class="flex items-center gap-2">
          <span
            v-if="savedBanner"
            class="px-2.5 py-1 rounded-lg bg-emerald-500/20 border border-emerald-500/40 text-emerald-300 font-semibold flex items-center gap-1"
          >
            <Check class="w-3.5 h-3.5" /> Preferences Saved
          </span>
          <button
            class="px-3.5 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white font-bold cursor-pointer transition-colors"
            @click="handleSave"
          >
            Save Preferences
          </button>
          <button
            class="p-1.5 rounded-lg bg-zinc-900 hover:bg-zinc-800 text-zinc-400 hover:text-zinc-100 border border-zinc-800 cursor-pointer"
            title="Close Preferences (Esc)"
            @click="emit('close')"
          >
            <X class="w-4 h-4" />
          </button>
        </div>
      </div>

      <!-- Body: Sidebar Navigation + Well-Organized Document Content -->
      <div class="flex-1 flex overflow-hidden min-h-0">
        <!-- Left Table of Contents Sidebar -->
        <aside class="w-64 sm:w-72 bg-zinc-950/90 border-r border-zinc-800 flex flex-col shrink-0">
          <div class="p-3 border-b border-zinc-800/80">
            <div class="relative">
              <Search class="w-3.5 h-3.5 text-zinc-500 absolute left-2.5 top-2" />
              <input
                v-model="searchQuery"
                type="text"
                placeholder="Search options & docs..."
                class="w-full bg-zinc-900 border border-zinc-800 rounded-lg pl-8 pr-3 py-1.5 text-xs text-zinc-100 placeholder-zinc-500 focus:outline-none focus:border-emerald-500"
              />
            </div>
          </div>

          <nav class="flex-1 overflow-y-auto p-2 space-y-1">
            <button
              v-for="sec in filteredSections"
              :key="sec.id"
              class="w-full text-left p-2.5 rounded-xl transition-all flex items-start gap-2.5 cursor-pointer"
              :class="activeSection === sec.id
                ? 'bg-emerald-950/60 border border-emerald-500/40 text-emerald-200 shadow-xs'
                : 'hover:bg-zinc-900 text-zinc-400 hover:text-zinc-200 border border-transparent'"
              @click="activeSection = sec.id"
            >
              <component
                :is="sec.icon"
                class="w-4 h-4 mt-0.5 shrink-0"
                :class="activeSection === sec.id ? 'text-emerald-400' : 'text-zinc-500'"
              />
              <div class="min-w-0">
                <div class="font-bold text-xs truncate" :class="activeSection === sec.id ? 'text-white' : 'text-zinc-200'">
                  {{ sec.title }}
                </div>
                <div class="text-[10px] text-zinc-400 truncate mt-0.5">
                  {{ sec.subtitle }}
                </div>
              </div>
            </button>
          </nav>

          <div class="p-3 border-t border-zinc-800/80 bg-zinc-950 text-[11px] text-zinc-500 flex items-center justify-between">
            <span>Shortcut: <strong class="text-zinc-300 font-mono">Ctrl+,</strong></span>
            <button
              class="text-rose-400 hover:text-rose-300 underline cursor-pointer"
              @click="resetWorkspaceLayout()"
            >
              Reset Panels
            </button>
          </div>
        </aside>

        <!-- Right Document Content Area -->
        <main class="flex-1 overflow-y-auto p-6 space-y-6 bg-zinc-900">
          <!-- ======================================================== -->
          <!-- SECTION 1: DATABASES & DEFAULT DATABASE                  -->
          <!-- ======================================================== -->
          <section v-if="activeSection === 'databases'" class="space-y-5 max-w-4xl">
            <div>
              <h3 class="text-lg font-extrabold text-zinc-100 flex items-center gap-2">
                <Database class="w-5 h-5 text-emerald-400" />
                <span>1. Database Architecture &amp; Default Database</span>
              </h3>
              <p class="text-zinc-300 mt-1 leading-relaxed">
                In NewScid, databases have <strong>four distinct roles</strong> so you always know which database opens on startup, where your saved games go, and where opening tree statistics come from.
              </p>
            </div>

            <!-- How Databases Are Related Document Card -->
            <div class="bg-zinc-950 border border-zinc-800 rounded-xl p-4 space-y-3">
              <div class="font-bold text-zinc-100 text-sm">How Your Databases Relate:</div>
              <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
                <div class="p-3 rounded-lg bg-zinc-900 border border-amber-500/30">
                  <div class="font-bold text-amber-400 flex items-center gap-1.5">
                    <Star class="w-3.5 h-3.5 fill-amber-400" />
                    <span>Default Startup Database</span>
                  </div>
                  <p class="text-[11px] text-zinc-300 mt-1">
                    Your primary "home" database. When <em>Auto-Open Default Database</em> is enabled, NewScid automatically opens this database as your Active Working Database every time you launch the app.
                  </p>
                </div>

                <div class="p-3 rounded-lg bg-zinc-900 border border-emerald-500/30">
                  <div class="font-bold text-emerald-400 flex items-center gap-1.5">
                    <Database class="w-3.5 h-3.5" />
                    <span>Active Working Database</span>
                  </div>
                  <p class="text-[11px] text-zinc-300 mt-1">
                    The database currently displayed in the bottom <strong>Games Browser</strong>. Pressing <code>Ctrl+S</code> saves games here, and <code>Ctrl+H</code> searches games here.
                  </p>
                </div>

                <div class="p-3 rounded-lg bg-zinc-900 border border-cyan-500/30">
                  <div class="font-bold text-cyan-400 flex items-center gap-1.5">
                    <BookOpen class="w-3.5 h-3.5" />
                    <span>Reference Theory Database</span>
                  </div>
                  <p class="text-[11px] text-zinc-300 mt-1">
                    A large Grandmaster/Mega database (such as <em>Lumbra's GigaBase</em>, <em>Caïssa</em>, or <em>TWIC</em>) queried by the <strong>Opening Tree (F3)</strong> while you keep your personal games in the Active Working Database.
                  </p>
                </div>

                <div class="p-3 rounded-lg bg-zinc-900 border border-purple-500/30">
                  <div class="font-bold text-purple-400 flex items-center gap-1.5">
                    <Sparkles class="w-3.5 h-3.5" />
                    <span>Clipbase (Temporary Scratchpad)</span>
                  </div>
                  <p class="text-[11px] text-zinc-300 mt-1">
                    A dedicated scratchpad base for temporary PGN imports, fair-play inspections, or ad-hoc game analysis without modifying your Default Database.
                  </p>
                </div>
              </div>
            </div>

            <!-- Default Database Selector -->
            <div class="bg-zinc-950 border border-zinc-800 rounded-xl p-4 space-y-4">
              <div class="flex flex-wrap items-center justify-between gap-2">
                <div>
                  <label class="font-bold text-sm text-zinc-100 flex items-center gap-1.5">
                    <Star class="w-4 h-4 text-amber-400 fill-amber-400" />
                    <span>Default Startup Database</span>
                  </label>
                  <p class="text-[11px] text-zinc-400">
                    Choose which database loads automatically when NewScid starts.
                  </p>
                </div>

                <div class="flex items-center gap-2">
                  <button
                    v-if="rolesStatus?.activeDb?.basePath"
                    class="px-3 py-1.5 rounded-lg bg-amber-600/20 hover:bg-amber-600/30 text-amber-300 border border-amber-500/40 font-semibold cursor-pointer transition-colors"
                    @click="localPrefs.defaultDatabasePath = rolesStatus.activeDb.basePath; emit('setDefaultDB', rolesStatus.activeDb.basePath); handleSave()"
                  >
                    ⭐ Set Current Active DB ({{ rolesStatus.activeDb.name }}) as Default
                  </button>
                  <button
                    v-if="localPrefs.defaultDatabasePath"
                    class="px-3 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white font-semibold cursor-pointer transition-colors"
                    @click="emit('openDefaultDB')"
                  >
                    Open Default DB Now
                  </button>
                </div>
              </div>

              <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
                <div>
                  <span class="block text-[11px] text-zinc-400 mb-1">Select from Installed Databases:</span>
                  <select
                    :value="localPrefs.defaultDatabasePath"
                    class="w-full bg-zinc-900 border border-zinc-700 rounded-lg px-3 py-2 text-xs text-zinc-100 focus:outline-none focus:border-amber-500"
                    @change="handleDefaultDbSelect"
                  >
                    <option value="">-- Automatic (First User Database) --</option>
                    <option
                      v-for="db in discoveredDatabases"
                      :key="db.basePath"
                      :value="db.basePath"
                    >
                      {{ db.name }} ({{ db.numGames }} games) — {{ db.category }}
                    </option>
                  </select>
                </div>

                <div class="flex items-end pb-1">
                  <label class="flex items-center gap-2 cursor-pointer select-none text-zinc-200">
                    <input
                      v-model="localPrefs.autoOpenDefaultDb"
                      type="checkbox"
                      class="w-4 h-4 rounded accent-emerald-500"
                      @change="handleSave"
                    />
                    <span>Automatically open Default Database on application startup</span>
                  </label>
                </div>
              </div>
            </div>

            <!-- Reference Database Selector -->
            <div class="bg-zinc-950 border border-zinc-800 rounded-xl p-4 space-y-4">
              <div>
                <label class="font-bold text-sm text-zinc-100 flex items-center gap-1.5">
                  <BookOpen class="w-4 h-4 text-cyan-400" />
                  <span>Reference Theory Database (for Opening Tree)</span>
                </label>
                <p class="text-[11px] text-zinc-400">
                  When set, the Opening Tree (F3) queries this reference database first so you get master statistics even while working in a small personal database.
                </p>
              </div>

              <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
                <div>
                  <span class="block text-[11px] text-zinc-400 mb-1">Select Reference Database:</span>
                  <select
                    :value="localPrefs.referenceDatabasePath"
                    class="w-full bg-zinc-900 border border-zinc-700 rounded-lg px-3 py-2 text-xs text-zinc-100 focus:outline-none focus:border-cyan-500"
                    @change="handleReferenceDbSelect"
                  >
                    <option value="">-- Use Active Working Database --</option>
                    <option
                      v-for="db in discoveredDatabases"
                      :key="db.basePath"
                      :value="db.basePath"
                    >
                      {{ db.name }} ({{ db.numGames }} games) — {{ db.category }}
                    </option>
                  </select>
                </div>

                <div class="flex items-end pb-1">
                  <label class="flex items-center gap-2 cursor-pointer select-none text-zinc-200">
                    <input
                      v-model="localPrefs.useReferenceDbForTree"
                      type="checkbox"
                      class="w-4 h-4 rounded accent-cyan-500"
                      @change="emit('setReferenceDB', localPrefs.referenceDatabasePath, localPrefs.useReferenceDbForTree); handleSave()"
                    />
                    <span>Use Reference Database for Opening Tree (F3) statistics</span>
                  </label>
                </div>
              </div>

              <!-- Update Any Database with Local Databases Card -->
              <div class="pt-2 border-t border-zinc-800/80 flex flex-wrap items-center justify-between gap-2">
                <div>
                  <div class="font-bold text-zinc-200 flex items-center gap-1.5">
                    <RefreshCw class="w-3.5 h-3.5 text-emerald-400" :class="{ 'animate-spin': isUpdatingFromLocal }" />
                    <span>Update Databases with Local Games Already on Computer</span>
                  </div>
                  <p class="text-[11px] text-zinc-400">
                    Deduplicates and merges all games from local <code class="text-emerald-400">.si4</code> and <code class="text-emerald-400">.pgn</code> databases (including Persona bases) into your Default, Reference, or Active database.
                  </p>
                </div>
                <div class="flex items-center gap-2 flex-wrap">
                  <button
                    v-if="rolesStatus?.activeDb?.basePath"
                    class="px-3 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white font-semibold flex items-center gap-1.5 cursor-pointer transition-colors"
                    :disabled="isUpdatingFromLocal"
                    @click="emit('updateFromLocal', rolesStatus.activeDb.basePath)"
                  >
                    <RefreshCw class="w-3.5 h-3.5" :class="{ 'animate-spin': isUpdatingFromLocal && (!updatingDbPath || updatingDbPath === rolesStatus.activeDb.basePath) }" />
                    <span>{{ isUpdatingFromLocal && (!updatingDbPath || updatingDbPath === rolesStatus.activeDb.basePath) ? 'Updating...' : `Update Active DB (${rolesStatus.activeDb.name})` }}</span>
                  </button>
                  <button
                    v-if="localPrefs.referenceDatabasePath"
                    class="px-3 py-1.5 rounded-lg bg-cyan-600/20 hover:bg-cyan-600/30 text-cyan-300 border border-cyan-500/40 font-semibold flex items-center gap-1.5 cursor-pointer transition-colors"
                    :disabled="isUpdatingFromLocal"
                    @click="emit('updateFromLocal', localPrefs.referenceDatabasePath)"
                  >
                    <RefreshCw class="w-3.5 h-3.5" :class="{ 'animate-spin': isUpdatingFromLocal && updatingDbPath === localPrefs.referenceDatabasePath }" />
                    <span>{{ isUpdatingFromLocal && updatingDbPath === localPrefs.referenceDatabasePath ? 'Updating...' : 'Update Reference / Mega DB' }}</span>
                  </button>
                </div>
              </div>
              <div
                v-if="localMergeStatus"
                class="p-2.5 rounded-lg bg-emerald-950/40 border border-emerald-500/40 text-emerald-200 text-xs flex items-center justify-between gap-3"
              >
                <div class="flex items-center gap-2">
                  <RefreshCw class="w-3.5 h-3.5 text-emerald-400 shrink-0" :class="{ 'animate-spin': isUpdatingFromLocal }" />
                  <span>{{ localMergeStatus }}</span>
                </div>
                <button
                  v-if="isUpdatingFromLocal"
                  class="px-2.5 py-1 rounded bg-rose-950 hover:bg-rose-900 text-rose-300 border border-rose-500/50 text-[11px] font-bold cursor-pointer transition-colors shrink-0"
                  @click="emit('cancelUpdateLocal')"
                >
                  🛑 Cancel
                </button>
              </div>
            </div>
          </section>

          <!-- ======================================================== -->
          <!-- SECTION 2: PANELS, DOCKS & FLOATERS                      -->
          <!-- ======================================================== -->
          <section v-if="activeSection === 'panels'" class="space-y-5 max-w-4xl">
            <div>
              <h3 class="text-lg font-extrabold text-zinc-100 flex items-center gap-2">
                <LayoutGrid class="w-5 h-5 text-emerald-400" />
                <span>2. Adjustable, Dockable, Removable &amp; Floating ("Floater") Panels</span>
              </h3>
              <p class="text-zinc-300 mt-1 leading-relaxed">
                Every workspace panel in NewScid — <strong>including the Chessboard itself</strong> — can be <strong>Docked</strong>, converted into a draggable/resizable <strong>Floater</strong>, <strong>Detached to a specific Hyprland Workspace (1..10)</strong>, or <strong>Hidden</strong>. All changes are saved automatically to your user config (<code class="text-emerald-400">~/.config/newscid/preferences.json</code>).
              </p>
            </div>

            <!-- Workspace Layout Presets -->
            <div class="bg-zinc-950 border border-zinc-800 rounded-xl p-4 space-y-3">
              <div class="font-bold text-sm text-zinc-100">One-Click Workspace Layout Presets</div>
              <div class="grid grid-cols-2 sm:grid-cols-4 gap-2.5">
                <button
                  class="p-3 rounded-xl bg-zinc-900 hover:bg-zinc-800 border border-zinc-800 hover:border-emerald-500/50 text-left cursor-pointer transition-all"
                  @click="applyLayoutPreset('standard')"
                >
                  <div class="font-bold text-emerald-400">Standard Study</div>
                  <p class="text-[10px] text-zinc-400 mt-0.5">Classic docked layout with Board, Engine, Tabs &amp; Bottom DB</p>
                </button>

                <button
                  class="p-3 rounded-xl bg-zinc-900 hover:bg-zinc-800 border border-zinc-800 hover:border-amber-500/50 text-left cursor-pointer transition-all"
                  @click="applyLayoutPreset('floaters')"
                >
                  <div class="font-bold text-amber-400">Pro Floaters Mode</div>
                  <p class="text-[10px] text-zinc-400 mt-0.5">Opening Tree, Engine &amp; Live Portal as draggable Floaters</p>
                </button>

                <button
                  class="p-3 rounded-xl bg-zinc-900 hover:bg-zinc-800 border border-zinc-800 hover:border-cyan-500/50 text-left cursor-pointer transition-all"
                  @click="applyLayoutPreset('hyprland-multiscreen')"
                >
                  <div class="font-bold text-cyan-400">Hyprland Multi-Screen</div>
                  <p class="text-[10px] text-zinc-400 mt-0.5">Clean primary screen ready for secondary monitor panels</p>
                </button>

                <button
                  class="p-3 rounded-xl bg-zinc-900 hover:bg-zinc-800 border border-zinc-800 hover:border-purple-500/50 text-left cursor-pointer transition-all"
                  @click="applyLayoutPreset('minimal')"
                >
                  <div class="font-bold text-purple-400">Minimal Board + Moves</div>
                  <p class="text-[10px] text-zinc-400 mt-0.5">Hides Engine, Imbalance &amp; Bottom Drawer for pure focus</p>
                </button>
              </div>
            </div>

            <!-- Per-Panel Configuration Table -->
            <div class="bg-zinc-950 border border-zinc-800 rounded-xl overflow-hidden">
              <div class="px-4 py-2.5 bg-zinc-900/80 border-b border-zinc-800 flex items-center justify-between">
                <span class="font-bold text-zinc-100">Customize Individual Panels (Docked / Floater / Hidden / Screen Workspace)</span>
                <button
                  class="px-2.5 py-1 rounded bg-zinc-800 hover:bg-zinc-700 text-zinc-300 text-[11px] flex items-center gap-1 cursor-pointer"
                  @click="resetWorkspaceLayout()"
                >
                  <RotateCcw class="w-3 h-3" />
                  <span>Reset Default Layout</span>
                </button>
              </div>

              <div class="divide-y divide-zinc-800/80">
                <div
                  v-for="panel in panelsArray"
                  :key="panel.id"
                  class="p-3 flex flex-wrap items-center justify-between gap-3 hover:bg-zinc-900/50 transition-colors"
                >
                  <div class="min-w-0 flex-1">
                    <div class="font-bold text-zinc-100 flex items-center gap-2">
                      <span>{{ panel.title }}</span>
                      <span
                        class="px-1.5 py-0.5 rounded text-[9px] font-mono uppercase"
                        :class="{
                          'bg-emerald-500/20 text-emerald-300 border border-emerald-500/30': panel.mode === 'docked',
                          'bg-amber-500/20 text-amber-300 border border-amber-500/30': panel.mode === 'floater',
                          'bg-cyan-500/20 text-cyan-300 border border-cyan-500/30': panel.mode === 'detached',
                          'bg-zinc-800 text-zinc-400 border border-zinc-700': panel.mode === 'hidden',
                        }"
                      >
                        {{ panel.mode }}{{ panel.targetWorkspace ? ` (WS ${panel.targetWorkspace})` : '' }}
                      </span>
                    </div>
                    <p class="text-[11px] text-zinc-400 mt-0.5">{{ panel.description }}</p>
                  </div>

                  <div class="flex items-center gap-2 shrink-0 flex-wrap">
                    <!-- Mode Selector -->
                    <div class="flex items-center bg-zinc-900 border border-zinc-800 rounded-lg p-0.5">
                      <button
                        class="px-2 py-1 rounded-md text-[11px] font-semibold cursor-pointer transition-colors"
                        :class="panel.mode === 'docked' ? 'bg-emerald-600 text-white' : 'text-zinc-400 hover:text-zinc-200'"
                        @click="setPanelMode(panel.id, 'docked')"
                      >
                        Docked
                      </button>
                      <button
                        class="px-2 py-1 rounded-md text-[11px] font-semibold cursor-pointer transition-colors"
                        :class="panel.mode === 'floater' ? 'bg-amber-600 text-white' : 'text-zinc-400 hover:text-zinc-200'"
                        @click="setPanelMode(panel.id, 'floater')"
                      >
                        Floater
                      </button>
                      <button
                        class="px-2 py-1 rounded-md text-[11px] font-semibold cursor-pointer transition-colors"
                        :class="panel.mode === 'detached' ? 'bg-cyan-600 text-white' : 'text-zinc-400 hover:text-zinc-200'"
                        @click="emit('detachPanelScreen', panel.id, panel.targetWorkspace)"
                      >
                        Detached
                      </button>
                      <button
                        class="px-2 py-1 rounded-md text-[11px] font-semibold cursor-pointer transition-colors"
                        :class="panel.mode === 'hidden' ? 'bg-rose-600/80 text-white' : 'text-zinc-400 hover:text-zinc-200'"
                        @click="setPanelMode(panel.id, 'hidden')"
                      >
                        Hidden
                      </button>
                    </div>

                    <!-- Dock Zone Selector -->
                    <select
                      v-if="panel.mode === 'docked' && panel.id !== 'board'"
                      :value="panel.dockZone"
                      class="bg-zinc-900 border border-zinc-700 rounded-lg px-2 py-1 text-[11px] text-zinc-200 focus:outline-none focus:border-emerald-500"
                      title="Select Dock Zone"
                      @change="setPanelDockZone(panel.id, ($event.target as HTMLSelectElement).value as DockZone)"
                    >
                      <option value="left">Left Dock (Under Board)</option>
                      <option value="right">Right Dock (Tabbed Pane)</option>
                      <option value="bottom">Bottom Dock (Drawer)</option>
                    </select>

                    <!-- Hyprland Workspace Selector -->
                    <select
                      :value="panel.targetWorkspace || ''"
                      class="bg-zinc-900 border border-cyan-500/40 rounded-lg px-2 py-1 text-[11px] text-cyan-300 font-mono focus:outline-none focus:border-cyan-400"
                      title="Assign Hyprland Workspace (1..10) and launch/move panel there"
                      @change="setPanelWorkspace(panel.id, ($event.target as HTMLSelectElement).value); if (($event.target as HTMLSelectElement).value) emit('detachPanelScreen', panel.id, ($event.target as HTMLSelectElement).value)"
                    >
                      <option value="">Workspace: Auto</option>
                      <option v-for="ws in 10" :key="ws" :value="String(ws)">Workspace {{ ws }}</option>
                    </select>

                    <!-- Detach to Screen Button -->
                    <button
                      class="px-2.5 py-1 rounded-lg bg-cyan-950/60 hover:bg-cyan-900/70 text-cyan-300 border border-cyan-500/40 text-[11px] font-semibold flex items-center gap-1 cursor-pointer transition-colors"
                      title="Open this panel in a dedicated OS window for multi-screen / Hyprland"
                      @click="emit('detachPanelScreen', panel.id, panel.targetWorkspace)"
                    >
                      <ExternalLink class="w-3 h-3" />
                      <span>Screen</span>
                    </button>
                  </div>
                </div>
              </div>
            </div>

            <!-- Pane Size Sliders -->
            <div class="bg-zinc-950 border border-zinc-800 rounded-xl p-4 grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div>
                <div class="flex justify-between mb-1">
                  <span class="font-bold text-zinc-200">Left Board Column Width</span>
                  <span class="font-mono text-emerald-400">{{ leftColWidth }}px</span>
                </div>
                <input
                  type="range"
                  min="320"
                  max="900"
                  :value="leftColWidth"
                  class="w-full accent-emerald-500"
                  @input="emit('updateLeftWidth', Number(($event.target as HTMLInputElement).value))"
                />
              </div>

              <div>
                <div class="flex justify-between mb-1">
                  <span class="font-bold text-zinc-200">Bottom Drawer Height</span>
                  <span class="font-mono text-emerald-400">{{ bottomDrawerHeight }}px</span>
                </div>
                <input
                  type="range"
                  min="120"
                  max="480"
                  :value="bottomDrawerHeight"
                  class="w-full accent-emerald-500"
                  @input="emit('updateBottomHeight', Number(($event.target as HTMLInputElement).value))"
                />
              </div>
            </div>
          </section>

          <!-- ======================================================== -->
          <!-- SECTION 3: MULTI-SCREEN & HYPRLAND INTEGRATION           -->
          <!-- ======================================================== -->
          <section v-if="activeSection === 'multiscreen'" class="space-y-5 max-w-4xl">
            <div>
              <h3 class="text-lg font-extrabold text-zinc-100 flex items-center gap-2">
                <Monitor class="w-5 h-5 text-cyan-400" />
                <span>3. Multi-Screen &amp; Hyprland Workspace Orchestration</span>
              </h3>
              <p class="text-zinc-300 mt-1 leading-relaxed">
                Spread NewScid across multiple monitors or Hyprland workspaces. Detached Screen Panels (including a standalone <strong>Chessboard</strong> window) run as native OS windows, can be placed on specific Hyprland workspaces (`1..10`), and <strong>automatically synchronize board positions, moves, and database state in real time</strong>.
              </p>
            </div>

            <!-- Main Window Workspace & Auto-Restore Config -->
            <div class="bg-zinc-950 border border-zinc-800 rounded-xl p-4 flex flex-wrap items-center justify-between gap-4">
              <div>
                <div class="font-bold text-sm text-zinc-100">Main Window Workspace &amp; Startup Persistence</div>
                <p class="text-[11px] text-zinc-400">
                  Choose which Hyprland workspace hosts the main NewScid window and whether detached panels re-launch on startup.
                </p>
              </div>
              <div class="flex items-center gap-3 flex-wrap">
                <select
                  :value="localPrefs.mainWindowWorkspace || ''"
                  class="bg-zinc-900 border border-cyan-500/40 rounded-lg px-3 py-1.5 text-xs text-cyan-300 font-mono"
                  @change="localPrefs.mainWindowWorkspace = ($event.target as HTMLSelectElement).value; emit('moveMainToWorkspace', localPrefs.mainWindowWorkspace || ''); handleSave()"
                >
                  <option value="">Main Window: Current Workspace</option>
                  <option v-for="ws in 10" :key="ws" :value="String(ws)">Move Main Window to Workspace {{ ws }}</option>
                </select>

                <label class="flex items-center gap-2 text-xs text-zinc-200 cursor-pointer">
                  <input
                    v-model="localPrefs.autoRestoreWorkspaceWindows"
                    type="checkbox"
                    class="w-4 h-4 accent-cyan-500"
                    @change="handleSave"
                  />
                  <span>Auto-restore detached workspace windows on startup</span>
                </label>
              </div>
            </div>

            <!-- One-Click Detached Screen Launchers -->
            <div class="bg-zinc-950 border border-zinc-800 rounded-xl p-4 space-y-3">
              <div class="font-bold text-sm text-zinc-100">Launch Dedicated Screen Windows (Synchronized Live)</div>
              <div class="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-2.5">
                <button
                  class="p-3 rounded-xl bg-zinc-900 hover:bg-zinc-800 border border-zinc-800 hover:border-emerald-500/50 text-left cursor-pointer transition-all"
                  @click="emit('detachPanelScreen', 'board', workspacePanels.board?.targetWorkspace)"
                >
                  <div class="font-bold text-emerald-400 flex items-center justify-between">
                    <span>Chessboard &amp; Clocks Screen</span>
                    <ExternalLink class="w-3.5 h-3.5" />
                  </div>
                  <p class="text-[10px] text-zinc-400 mt-1">Detach the interactive Chessboard to its own OS window / Hyprland workspace</p>
                </button>

                <button
                  class="p-3 rounded-xl bg-zinc-900 hover:bg-zinc-800 border border-zinc-800 hover:border-cyan-500/50 text-left cursor-pointer transition-all"
                  @click="emit('detachPanelScreen', 'tree', workspacePanels.tree?.targetWorkspace)"
                >
                  <div class="font-bold text-cyan-400 flex items-center justify-between">
                    <span>Opening Tree Screen</span>
                    <ExternalLink class="w-3.5 h-3.5" />
                  </div>
                  <p class="text-[10px] text-zinc-400 mt-1">Full-window Opening Tree &amp; Reference DB explorer on Screen 2</p>
                </button>

                <button
                  class="p-3 rounded-xl bg-zinc-900 hover:bg-zinc-800 border border-zinc-800 hover:border-emerald-500/50 text-left cursor-pointer transition-all"
                  @click="emit('detachPanelScreen', 'engine', workspacePanels.engine?.targetWorkspace)"
                >
                  <div class="font-bold text-emerald-400 flex items-center justify-between">
                    <span>UCI Engine &amp; Telemetry</span>
                    <ExternalLink class="w-3.5 h-3.5" />
                  </div>
                  <p class="text-[10px] text-zinc-400 mt-1">Dedicated Stockfish MultiPV &amp; Syzygy Tablebase monitor</p>
                </button>

                <button
                  class="p-3 rounded-xl bg-zinc-900 hover:bg-zinc-800 border border-zinc-800 hover:border-amber-500/50 text-left cursor-pointer transition-all"
                  @click="emit('detachPanelScreen', 'games', workspacePanels.games?.targetWorkspace)"
                >
                  <div class="font-bold text-amber-400 flex items-center justify-between">
                    <span>Database Games Browser</span>
                    <ExternalLink class="w-3.5 h-3.5" />
                  </div>
                  <p class="text-[10px] text-zinc-400 mt-1">Full-screen database list — clicking any game loads it on Screen 1</p>
                </button>

                <button
                  class="p-3 rounded-xl bg-zinc-900 hover:bg-zinc-800 border border-zinc-800 hover:border-purple-500/50 text-left cursor-pointer transition-all"
                  @click="emit('detachPanelScreen', 'notation', workspacePanels.notation?.targetWorkspace)"
                >
                  <div class="font-bold text-purple-400 flex items-center justify-between">
                    <span>Notation &amp; Eval Graph</span>
                    <ExternalLink class="w-3.5 h-3.5" />
                  </div>
                  <p class="text-[10px] text-zinc-400 mt-1">Dedicated move tree, annotations &amp; evaluation curve window</p>
                </button>

                <button
                  class="p-3 rounded-xl bg-zinc-900 hover:bg-zinc-800 border border-zinc-800 hover:border-emerald-500/50 text-left cursor-pointer transition-all"
                  @click="emit('launchPortalWindow', 'lichess', 'https://lichess.org')"
                >
                  <div class="font-bold text-emerald-300 flex items-center justify-between">
                    <span>Lichess.org Screen Window</span>
                    <Globe class="w-3.5 h-3.5" />
                  </div>
                  <p class="text-[10px] text-zinc-400 mt-1">Open Lichess.org in its own standalone Hyprland window</p>
                </button>

                <button
                  class="p-3 rounded-xl bg-zinc-900 hover:bg-zinc-800 border border-zinc-800 hover:border-emerald-500/50 text-left cursor-pointer transition-all"
                  @click="emit('launchPortalWindow', 'chesscom', 'https://www.chess.com/play/online')"
                >
                  <div class="font-bold text-emerald-300 flex items-center justify-between">
                    <span>Chess.com Screen Window</span>
                    <Globe class="w-3.5 h-3.5" />
                  </div>
                  <p class="text-[10px] text-zinc-400 mt-1">Open Chess.com in its own standalone Hyprland window</p>
                </button>
              </div>
            </div>

            <!-- Hyprland Config Rules Helper -->
            <div class="bg-zinc-950 border border-zinc-800 rounded-xl p-4 space-y-2.5">
              <div class="flex items-center justify-between">
                <div>
                  <div class="font-bold text-sm text-zinc-100">Hyprland Multi-Monitor Window Rules (<code>hyprland.conf</code>)</div>
                  <p class="text-[11px] text-zinc-400">
                    Optional rules to automatically route detached NewScid panels and Lichess/Chess.com portals to specific monitors or workspaces.
                  </p>
                </div>
                <button
                  class="px-3 py-1.5 rounded-lg bg-zinc-800 hover:bg-zinc-700 text-zinc-200 border border-zinc-700 font-semibold flex items-center gap-1.5 cursor-pointer"
                  @click="copyHyprlandSnippet"
                >
                  <Copy class="w-3.5 h-3.5 text-emerald-400" />
                  <span>{{ copiedHyprland ? 'Copied!' : 'Copy Rules' }}</span>
                </button>
              </div>
              <pre class="p-3 rounded-lg bg-zinc-900 border border-zinc-800 text-[11px] font-mono text-emerald-300 overflow-x-auto select-all">{{ hyprlandConfigSnippet }}</pre>
            </div>
          </section>

          <!-- ======================================================== -->
          <!-- SECTION 4: LICHESS & CHESS.COM PORTALS                   -->
          <!-- ======================================================== -->
          <section v-if="activeSection === 'portals'" class="space-y-5 max-w-4xl">
            <div>
              <h3 class="text-lg font-extrabold text-zinc-100 flex items-center gap-2">
                <Globe class="w-5 h-5 text-emerald-400" />
                <span>4. Lichess.org &amp; Chess.com Live Portal Options</span>
              </h3>
              <p class="text-zinc-300 mt-1 leading-relaxed">
                Configure your online usernames, default portal platform, and how standalone Chess.com and Lichess windows open on your desktop.
              </p>
            </div>

            <div class="bg-zinc-950 border border-zinc-800 rounded-xl p-4 space-y-4">
              <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
                <div>
                  <label class="block font-bold text-zinc-200 mb-1">Default Lichess.org Username</label>
                  <input
                    v-model="localPrefs.lichessUsername"
                    type="text"
                    placeholder="e.g. DrNykterstein"
                    class="w-full bg-zinc-900 border border-zinc-700 rounded-lg px-3 py-2 text-xs text-zinc-100 focus:outline-none focus:border-emerald-500"
                  />
                </div>

                <div>
                  <label class="block font-bold text-zinc-200 mb-1">Default Chess.com Username</label>
                  <input
                    v-model="localPrefs.chessComUsername"
                    type="text"
                    placeholder="e.g. MagnusCarlsen"
                    class="w-full bg-zinc-900 border border-zinc-700 rounded-lg px-3 py-2 text-xs text-zinc-100 focus:outline-none focus:border-emerald-500"
                  />
                </div>

                <div>
                  <label class="block font-bold text-zinc-200 mb-1">Default Portal Platform</label>
                  <select
                    v-model="localPrefs.defaultPortalPlatform"
                    class="w-full bg-zinc-900 border border-zinc-700 rounded-lg px-3 py-2 text-xs text-zinc-100 focus:outline-none focus:border-emerald-500"
                  >
                    <option value="lichess">Lichess.org</option>
                    <option value="chesscom">Chess.com</option>
                  </select>
                </div>

                <div>
                  <label class="block font-bold text-zinc-200 mb-1">External Portal Window Mode</label>
                  <select
                    v-model="localPrefs.portalWindowMode"
                    class="w-full bg-zinc-900 border border-zinc-700 rounded-lg px-3 py-2 text-xs text-zinc-100 focus:outline-none focus:border-emerald-500"
                  >
                    <option value="browser-app">Standalone Chromeless Browser Window (--app, keeps login cookies)</option>
                    <option value="native-window">Native NewScid WebKit Window</option>
                  </select>
                </div>
              </div>

              <div>
                <label class="block font-bold text-zinc-200 mb-1">Custom Browser / Launcher Command (Optional)</label>
                <input
                  v-model="localPrefs.customBrowserCmd"
                  type="text"
                  placeholder="Leave blank for auto-detect (chromium/brave/firefox --app), or e.g. firefox --new-window"
                  class="w-full bg-zinc-900 border border-zinc-700 rounded-lg px-3 py-2 text-xs text-zinc-100 focus:outline-none focus:border-emerald-500"
                />
              </div>
            </div>
          </section>

          <!-- ======================================================== -->
          <!-- SECTION 5: BOARD, PIECES & APPEARANCE                    -->
          <!-- ======================================================== -->
          <section v-if="activeSection === 'appearance'" class="space-y-5 max-w-4xl">
            <div>
              <h3 class="text-lg font-extrabold text-zinc-100 flex items-center gap-2">
                <Palette class="w-5 h-5 text-emerald-400" />
                <span>5. Board, Piece Sets, Themes &amp; Audio</span>
              </h3>
              <p class="text-zinc-300 mt-1 leading-relaxed">
                Customize the application color palette, chessboard squares, vector piece set, algebraic notation format, and visual training aids.
              </p>
            </div>

            <!-- UI Themes -->
            <div class="bg-zinc-950 border border-zinc-800 rounded-xl p-4 space-y-3">
              <div class="font-bold text-sm text-zinc-100">Application UI Theme</div>
              <div class="grid grid-cols-2 sm:grid-cols-4 gap-2">
                <button
                  v-for="t in UI_THEMES"
                  :key="t.id"
                  class="p-2.5 rounded-xl border text-left transition-all cursor-pointer flex items-center justify-between"
                  :class="currentUITheme === t.id
                    ? 'bg-emerald-950/50 border-emerald-500 text-white font-bold'
                    : 'bg-zinc-900 border-zinc-800 text-zinc-300 hover:bg-zinc-800'"
                  @click="emit('changeUITheme', t.id); localPrefs.defaultUITheme = t.id"
                >
                  <span class="flex items-center gap-2 truncate">
                    <span class="w-3 h-3 rounded-full shrink-0" :style="{ backgroundColor: t.preview.accent }" />
                    <span class="truncate">{{ t.name }}</span>
                  </span>
                  <Check v-if="currentUITheme === t.id" class="w-3.5 h-3.5 text-emerald-400 shrink-0" />
                </button>
              </div>
            </div>

            <!-- Piece Set & Board Theme -->
            <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div class="bg-zinc-950 border border-zinc-800 rounded-xl p-4 space-y-2.5">
                <div class="font-bold text-sm text-zinc-100">Chess Piece Set</div>
                <div class="space-y-1.5 max-h-56 overflow-y-auto pr-1">
                  <button
                    v-for="ps in pieceSets"
                    :key="ps.id"
                    class="w-full p-2.5 rounded-lg border text-left flex items-center justify-between cursor-pointer transition-colors"
                    :class="currentPieceSet === ps.id
                      ? 'bg-emerald-950/50 border-emerald-500 text-white font-bold'
                      : 'bg-zinc-900 border-zinc-800 text-zinc-300 hover:bg-zinc-800'"
                    @click="emit('changePieceSet', ps.id); localPrefs.defaultPieceSet = ps.id"
                  >
                    <div>
                      <div>{{ ps.name }}</div>
                      <div class="text-[10px] text-zinc-400 font-normal">{{ ps.description }}</div>
                    </div>
                    <Check v-if="currentPieceSet === ps.id" class="w-4 h-4 text-emerald-400" />
                  </button>
                </div>
              </div>

              <div class="bg-zinc-950 border border-zinc-800 rounded-xl p-4 space-y-2.5">
                <div class="font-bold text-sm text-zinc-100">Chessboard Square Theme</div>
                <div class="space-y-1.5 max-h-56 overflow-y-auto pr-1">
                  <button
                    v-for="bt in boardThemes"
                    :key="bt.id"
                    class="w-full p-2.5 rounded-lg border text-left flex items-center justify-between cursor-pointer transition-colors"
                    :class="currentThemeId === bt.id
                      ? 'bg-emerald-950/50 border-emerald-500 text-white font-bold'
                      : 'bg-zinc-900 border-zinc-800 text-zinc-300 hover:bg-zinc-800'"
                    @click="emit('changeTheme', bt.id); localPrefs.defaultBoardTheme = bt.id"
                  >
                    <div class="flex items-center gap-2.5">
                      <div class="flex rounded overflow-hidden border border-zinc-700">
                        <span class="w-4 h-4" :style="{ backgroundColor: bt.light }" />
                        <span class="w-4 h-4" :style="{ backgroundColor: bt.dark }" />
                      </div>
                      <span>{{ bt.name }}</span>
                    </div>
                    <Check v-if="currentThemeId === bt.id" class="w-4 h-4 text-emerald-400" />
                  </button>
                </div>
              </div>
            </div>

            <!-- Visual Toggles & Notation Style -->
            <div class="bg-zinc-950 border border-zinc-800 rounded-xl p-4 grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div class="space-y-2.5">
                <div class="font-bold text-sm text-zinc-100">Board Visuals &amp; Audio</div>
                <label class="flex items-center gap-2 cursor-pointer text-zinc-200">
                  <input type="checkbox" :checked="soundEnabled" class="w-4 h-4 accent-emerald-500" @change="emit('toggleSound')" />
                  <span>Move &amp; Capture Sound Effects</span>
                </label>
                <label class="flex items-center gap-2 cursor-pointer text-zinc-200">
                  <input type="checkbox" :checked="showEvalBar" class="w-4 h-4 accent-emerald-500" @change="emit('toggleEvalBar')" />
                  <span>Show Live Evaluation Bar Beside Board</span>
                </label>
                <label class="flex items-center gap-2 cursor-pointer text-zinc-200">
                  <input type="checkbox" :checked="threatMode" class="w-4 h-4 accent-emerald-500" @change="emit('toggleThreatMode')" />
                  <span>Threat / Attacked Squares Highlight Overlay</span>
                </label>
                <label class="flex items-center gap-2 cursor-pointer text-zinc-200">
                  <input type="checkbox" :checked="blindfold" class="w-4 h-4 accent-emerald-500" @change="emit('toggleBlindfold')" />
                  <span>Blindfold Calculation Mode (Hide Pieces)</span>
                </label>
              </div>

              <div class="space-y-2">
                <div class="font-bold text-sm text-zinc-100">Move Notation Format</div>
                <div class="space-y-1.5">
                  <button
                    v-for="opt in notationOptions"
                    :key="opt.id"
                    class="w-full px-3 py-2 rounded-lg border text-left flex items-center justify-between cursor-pointer"
                    :class="currentStyle === opt.id
                      ? 'bg-amber-950/40 border-amber-500/50 text-amber-300 font-bold'
                      : 'bg-zinc-900 border-zinc-800 text-zinc-300 hover:bg-zinc-800'"
                    @click="setNotationStyle(opt.id)"
                  >
                    <span>{{ opt.name }} ({{ opt.example }})</span>
                    <Check v-if="currentStyle === opt.id" class="w-3.5 h-3.5 text-amber-400" />
                  </button>
                </div>
              </div>
            </div>
          </section>

          <!-- ======================================================== -->
          <!-- SECTION 6: UCI ENGINES & ANALYSIS                        -->
          <!-- ======================================================== -->
          <section v-if="activeSection === 'engines'" class="space-y-5 max-w-4xl">
            <div>
              <h3 class="text-lg font-extrabold text-zinc-100 flex items-center gap-2">
                <Cpu class="w-5 h-5 text-emerald-400" />
                <span>6. UCI Engine &amp; Analysis Configuration</span>
              </h3>
              <p class="text-zinc-300 mt-1 leading-relaxed">
                Configure default parameters for Stockfish and other UCI chess engines.
              </p>
            </div>

            <div class="bg-zinc-950 border border-zinc-800 rounded-xl p-4 grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div>
                <label class="block font-bold text-zinc-200 mb-1">Default UCI Engine</label>
                <select
                  v-model="localPrefs.defaultEngine"
                  class="w-full bg-zinc-900 border border-zinc-700 rounded-lg px-3 py-2 text-xs text-zinc-100 focus:outline-none focus:border-emerald-500"
                >
                  <option value="">-- First Available System Engine (Stockfish) --</option>
                  <option
                    v-for="eng in discoveredEngines"
                    :key="eng.cmd"
                    :value="eng.name || eng.cmd"
                  >
                    {{ eng.name }} ({{ eng.cmd }})
                  </option>
                </select>
              </div>

              <div>
                <label class="block font-bold text-zinc-200 mb-1">Default MultiPV Lines (1–5)</label>
                <input
                  v-model.number="localPrefs.defaultMultiPv"
                  type="number"
                  min="1"
                  max="5"
                  class="w-full bg-zinc-900 border border-zinc-700 rounded-lg px-3 py-2 text-xs text-zinc-100 focus:outline-none focus:border-emerald-500"
                />
              </div>

              <div>
                <label class="block font-bold text-zinc-200 mb-1">Engine CPU Threads</label>
                <input
                  v-model.number="localPrefs.defaultThreads"
                  type="number"
                  min="1"
                  max="64"
                  class="w-full bg-zinc-900 border border-zinc-700 rounded-lg px-3 py-2 text-xs text-zinc-100 focus:outline-none focus:border-emerald-500"
                />
              </div>

              <div>
                <label class="block font-bold text-zinc-200 mb-1">Engine Hash Table Size (MB)</label>
                <input
                  v-model.number="localPrefs.defaultHashMb"
                  type="number"
                  min="16"
                  max="8192"
                  step="16"
                  class="w-full bg-zinc-900 border border-zinc-700 rounded-lg px-3 py-2 text-xs text-zinc-100 focus:outline-none focus:border-emerald-500"
                />
              </div>

              <div>
                <label class="block font-bold text-zinc-200 mb-1">Full-Game Annotation Time per Ply (ms)</label>
                <input
                  v-model.number="localPrefs.annotateMoveTimeMs"
                  type="number"
                  min="50"
                  max="5000"
                  step="50"
                  class="w-full bg-zinc-900 border border-zinc-700 rounded-lg px-3 py-2 text-xs text-zinc-100 focus:outline-none focus:border-emerald-500"
                />
              </div>
            </div>
          </section>
        </main>
      </div>
    </div>
  </div>
</template>
