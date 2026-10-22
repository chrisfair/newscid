<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import {
  Globe,
  ExternalLink,
  Tv,
  Zap,
  Compass,
  RefreshCw,
  Shield,
  Download,
  Monitor,
  Sparkles,
  Radio,
  CheckCircle2,
  Database,
  Play
} from '@lucide/vue'
import {
  LaunchPortalWindow,
  GetLivePortalCaptureStatus,
  ConfigureLivePortalAutoCapture,
  CheckAndCaptureLivePortalGames,
  CaptureOnlineGameFromURLOrPGN
} from '../../wailsjs/go/main/App'
import { BrowserOpenURL, EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'

const props = defineProps<{
  currentFen: string
  lichessUsername?: string
  chessComUsername?: string
  defaultWindowMode?: string
}>()

const emit = defineEmits<{
  (e: 'openOnlineSync'): void
  (e: 'openFairPlay'): void
  (e: 'detachPortalScreen'): void
  (e: 'floatPortal'): void
  (e: 'gameCaptured', payload: any): void
  (e: 'loadGameById', gameId: number): void
  (e: 'loadPgnToBoard', pgn: string): void
}>()

const activePlatform = ref<'lichess' | 'chesscom'>('lichess')
const embedMode = ref<'tv' | 'puzzle' | 'bridge'>('tv')
const windowMode = ref<string>(props.defaultWindowMode || 'browser-app')
const customUrl = ref('')
const statusMessage = ref<string | null>(null)
const isErrorToast = ref(false)
const iframeKey = ref(1)

// Live Auto-Capture State
const lichessUserInput = ref(props.lichessUsername || '')
const chessComUserInput = ref(props.chessComUsername || '')
const autoCaptureEnabled = ref(true)
const autoLoadToBoard = ref(true)
const watcherActive = ref(false)
const targetDbName = ref('My_Games')
const sessionCapturedCount = ref(0)
const lastCapturedGameId = ref(0)
const lastCapturedSummary = ref('')
const isCheckingNow = ref(false)
const isCapturingUrl = ref(false)
const showAccountSetup = ref(false)

watch(() => props.lichessUsername, (val) => {
  if (val && !lichessUserInput.value) lichessUserInput.value = val
})
watch(() => props.chessComUsername, (val) => {
  if (val && !chessComUserInput.value) chessComUserInput.value = val
})

const hasConfiguredUsername = computed(() => {
  return Boolean(lichessUserInput.value.trim() || chessComUserInput.value.trim())
})

const encodedFenForLichess = computed(() => {
  const fen = (props.currentFen || 'rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1').trim()
  return fen.replace(/\s+/g, '_')
})

const encodedFenForChessCom = computed(() => {
  const fen = (props.currentFen || 'rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1').trim()
  return encodeURIComponent(fen)
})

const lichessAnalysisUrl = computed(() => `https://lichess.org/analysis/${encodedFenForLichess.value}`)
const chessComAnalysisUrl = computed(() => `https://www.chess.com/analysis?fen=${encodedFenForChessCom.value}`)

const activeEmbedSrc = computed(() => {
  if (embedMode.value === 'tv') {
    return 'https://lichess.org/tv/frame?theme=brown&bg=dark'
  }
  return 'https://lichess.org/training/frame?theme=brown&bg=dark'
})

function showToast(msg: string, isErr = false) {
  statusMessage.value = msg
  isErrorToast.value = isErr
  setTimeout(() => {
    if (statusMessage.value === msg) {
      statusMessage.value = null
    }
  }, 5000)
}

function applyCaptureStatus(st: any) {
  if (!st) return
  autoCaptureEnabled.value = Boolean(st.enabled)
  autoLoadToBoard.value = Boolean(st.autoLoadToBoard)
  watcherActive.value = Boolean(st.watcherActive)
  if (st.lichessUsername) lichessUserInput.value = st.lichessUsername
  if (st.chessComUsername) chessComUserInput.value = st.chessComUsername
  if (st.targetDatabaseName) targetDbName.value = st.targetDatabaseName
  sessionCapturedCount.value = st.sessionCapturedCount || 0
  if (st.lastCapturedGameId) lastCapturedGameId.value = st.lastCapturedGameId
  if (st.lastCapturedGameSummary) lastCapturedSummary.value = st.lastCapturedGameSummary
}

async function refreshCaptureStatus() {
  try {
    const st = await GetLivePortalCaptureStatus()
    applyCaptureStatus(st)
    if (!st?.lichessUsername && !st?.chessComUsername) {
      showAccountSetup.value = true
    } else if (st?.enabled && !st?.watcherActive) {
      const updated = await ConfigureLivePortalAutoCapture(
        lichessUserInput.value.trim(),
        chessComUserInput.value.trim(),
        true,
        autoLoadToBoard.value
      )
      applyCaptureStatus(updated)
    }
  } catch {
    // Ignore in test environments
  }
}

async function handleSaveAutoCaptureConfig() {
  try {
    const st = await ConfigureLivePortalAutoCapture(
      lichessUserInput.value.trim(),
      chessComUserInput.value.trim(),
      autoCaptureEnabled.value,
      autoLoadToBoard.value
    )
    applyCaptureStatus(st)
    showAccountSetup.value = false
    if (autoCaptureEnabled.value && hasConfiguredUsername.value) {
      showToast(`Live Auto-Capture active! Finished games on ${[
        lichessUserInput.value.trim() ? `Lichess (@${lichessUserInput.value.trim()})` : '',
        chessComUserInput.value.trim() ? `Chess.com (@${chessComUserInput.value.trim()})` : ''
      ].filter(Boolean).join(' & ')} will be automatically added to '${targetDbName.value}'.`)
      await handleCheckAndCaptureNow(true)
    } else {
      showToast('Live Portal Auto-Capture settings saved.')
    }
  } catch (err: any) {
    showToast(`Failed to save Auto-Capture settings: ${err?.message || err}`, true)
  }
}

async function handleToggleAutoCapture() {
  autoCaptureEnabled.value = !autoCaptureEnabled.value
  await handleSaveAutoCaptureConfig()
}

async function handleToggleAutoLoad() {
  autoLoadToBoard.value = !autoLoadToBoard.value
  await handleSaveAutoCaptureConfig()
}

async function handleCheckAndCaptureNow(silentIfZero = false) {
  if (isCheckingNow.value) return
  if (!hasConfiguredUsername.value) {
    showAccountSetup.value = true
    showToast('Enter your Lichess or Chess.com username first so NewScid knows which live games to capture.', true)
    return
  }
  isCheckingNow.value = true
  try {
    // Save any newly typed username first
    await ConfigureLivePortalAutoCapture(
      lichessUserInput.value.trim(),
      chessComUserInput.value.trim(),
      autoCaptureEnabled.value,
      autoLoadToBoard.value
    )
    const res = await CheckAndCaptureLivePortalGames()
    if (res?.status) {
      applyCaptureStatus(res.status)
    }
    if (res && res.newGamesImported > 0) {
      showToast(`✅ Auto-captured ${res.newGamesImported} new game(s) into '${res.targetDatabaseName}': ${res.latestGameSummary}`)
      emit('gameCaptured', res)
    } else if (!silentIfZero) {
      showToast(`Checked Lichess & Chess.com — all finished games are already in '${targetDbName.value}' (${res?.duplicatesSkipped || 0} recent verified).`)
    }
  } catch (err: any) {
    if (!silentIfZero) {
      showToast(`Capture check error: ${err?.message || err}`, true)
    }
  } finally {
    isCheckingNow.value = false
  }
}

async function handleCaptureUrlOrPgn() {
  const input = customUrl.value.trim()
  if (!input || isCapturingUrl.value) return
  isCapturingUrl.value = true
  try {
    const res = await CaptureOnlineGameFromURLOrPGN(input)
    if (res?.status) {
      applyCaptureStatus(res.status)
    }
    if (res && res.newGamesImported > 0) {
      showToast(`✅ Captured & saved to '${res.targetDatabaseName}' (Game #${res.latestGameId}): ${res.latestGameSummary}`)
      emit('gameCaptured', res)
    } else if (res && res.latestGamePgn) {
      showToast(`Game already in '${res.targetDatabaseName}' — loaded '${res.latestGameSummary}' onto the board!`)
      emit('loadPgnToBoard', res.latestGamePgn)
    }
    customUrl.value = ''
  } catch (err: any) {
    showToast(`Could not capture game from URL/PGN: ${err?.message || err}`, true)
  } finally {
    isCapturingUrl.value = false
  }
}

async function openPortalUrl(platform: 'lichess' | 'chesscom', url: string) {
  showToast(`Launching ${platform === 'chesscom' ? 'Chess.com' : 'Lichess.org'} window (Auto-Capture watching for finished games)...`)
  try {
    if (autoCaptureEnabled.value && hasConfiguredUsername.value) {
      await ConfigureLivePortalAutoCapture(
        lichessUserInput.value.trim(),
        chessComUserInput.value.trim(),
        true,
        autoLoadToBoard.value
      )
    }
    await LaunchPortalWindow(platform, url, windowMode.value)
    showToast(`Opened ${url} — finished games will be automatically added to '${targetDbName.value}'.`)
  } catch {
    try {
      BrowserOpenURL(url)
      showToast(`Opened ${url} in system browser`)
    } catch {
      window.open(url, '_blank')
    }
  }
}

function handleLaunchCustomUrl() {
  let u = customUrl.value.trim()
  if (!u) return
  if (!u.startsWith('http://') && !u.startsWith('https://')) {
    u = 'https://' + u
  }
  const plat = u.toLowerCase().includes('chess.com') ? 'chesscom' : 'lichess'
  openPortalUrl(plat, u)
}

onMounted(async () => {
  await refreshCaptureStatus()
  try {
    if (typeof window !== 'undefined' && (window as any).runtime?.EventsOnMultiple) {
      EventsOn('portal:game-captured', (res: any) => {
        if (res?.status) {
          applyCaptureStatus(res.status)
        }
        if (res?.newGamesImported > 0) {
          showToast(`🎉 Auto-added new finished game to '${res.targetDatabaseName}': ${res.latestGameSummary}`)
        }
      })
    }
  } catch {}
})

onUnmounted(() => {
  EventsOff('portal:game-captured')
})
</script>

<template>
  <div class="live-portal-panel h-full flex flex-col gap-2.5 text-xs overflow-y-auto p-1">
    <!-- Top Platform & Window Launcher Bar -->
    <div class="bg-zinc-950 border border-zinc-800 rounded-xl p-2.5 flex flex-wrap items-center justify-between gap-2">
      <div class="flex items-center gap-1.5">
        <button
          class="px-3 py-1.5 rounded-lg font-bold flex items-center gap-1.5 transition-all cursor-pointer"
          :class="activePlatform === 'lichess'
            ? 'bg-emerald-600 text-white shadow-sm'
            : 'bg-zinc-900 text-zinc-400 hover:text-zinc-200 border border-zinc-800'"
          @click="activePlatform = 'lichess'"
        >
          <Globe class="w-3.5 h-3.5" />
          <span>Lichess.org</span>
        </button>

        <button
          class="px-3 py-1.5 rounded-lg font-bold flex items-center gap-1.5 transition-all cursor-pointer"
          :class="activePlatform === 'chesscom'
            ? 'bg-emerald-600 text-white shadow-sm'
            : 'bg-zinc-900 text-zinc-400 hover:text-zinc-200 border border-zinc-800'"
          @click="activePlatform = 'chesscom'"
        >
          <Globe class="w-3.5 h-3.5" />
          <span>Chess.com</span>
        </button>
      </div>

      <div class="flex items-center gap-1.5">
        <select
          v-model="windowMode"
          class="bg-zinc-900 border border-zinc-700 rounded-lg px-2 py-1 text-[11px] text-zinc-200 focus:outline-none focus:border-emerald-500"
          title="Choose how external portal windows open on Hyprland / multi-screen"
        >
          <option value="browser-app">Standalone App Window (--app, keeps login)</option>
          <option value="native-window">Native NewScid WebKit Window</option>
        </select>

        <button
          class="px-2.5 py-1 rounded-lg bg-cyan-950/70 hover:bg-cyan-900/70 text-cyan-300 border border-cyan-500/40 font-semibold flex items-center gap-1 cursor-pointer transition-colors"
          title="Detach this portal panel to a separate Hyprland screen"
          @click="emit('detachPortalScreen')"
        >
          <Monitor class="w-3.5 h-3.5" />
          <span>Screen Window</span>
        </button>
      </div>
    </div>

    <!-- ========================================================================= -->
    <!-- LIVE AUTO-CAPTURE TO PERSONAL DATABASE BAR                                -->
    <!-- ========================================================================= -->
    <div class="bg-zinc-950/95 border border-emerald-500/30 rounded-xl p-2.5 space-y-2">
      <div class="flex flex-wrap items-center justify-between gap-2">
        <div class="flex items-center gap-2 flex-wrap">
          <button
            type="button"
            class="px-2.5 py-1 rounded-lg font-bold flex items-center gap-1.5 border transition-colors cursor-pointer"
            :class="autoCaptureEnabled && hasConfiguredUsername
              ? 'bg-emerald-950/80 border-emerald-500/50 text-emerald-300 hover:bg-emerald-900/80'
              : 'bg-zinc-900 border-zinc-700 text-zinc-400 hover:text-zinc-200'"
            title="Automatically detect finished games on Lichess.org & Chess.com and add them to your Personal User Database without manual syncing"
            @click="handleToggleAutoCapture"
          >
            <Radio class="w-3.5 h-3.5" :class="autoCaptureEnabled && hasConfiguredUsername ? 'text-emerald-400 animate-pulse' : 'text-zinc-500'" />
            <span>Auto-Capture to DB: {{ autoCaptureEnabled ? 'ON' : 'OFF' }}</span>
          </button>

          <span class="text-[11px] text-zinc-300 flex items-center gap-1">
            <Database class="w-3 h-3 text-amber-400" />
            <span>Target: <strong class="text-amber-300 font-mono">{{ targetDbName }}</strong></span>
          </span>

          <label class="flex items-center gap-1 text-[11px] text-zinc-400 hover:text-zinc-200 cursor-pointer select-none ml-1">
            <input
              type="checkbox"
              :checked="autoLoadToBoard"
              class="rounded border-zinc-700 bg-zinc-900 text-emerald-500 focus:ring-0 cursor-pointer"
              @change="handleToggleAutoLoad"
            />
            <span>Auto-load finished game onto board</span>
          </label>
        </div>

        <div class="flex items-center gap-1.5">
          <span
            v-if="sessionCapturedCount > 0"
            class="px-2 py-0.5 rounded bg-emerald-950/80 border border-emerald-500/40 text-emerald-300 font-mono text-[10px]"
          >
            +{{ sessionCapturedCount }} auto-saved
          </span>

          <button
            type="button"
            class="px-2.5 py-1 rounded-lg bg-emerald-600/20 hover:bg-emerald-600/30 border border-emerald-500/40 text-emerald-300 font-semibold flex items-center gap-1 cursor-pointer transition-colors disabled:opacity-50"
            :disabled="isCheckingNow"
            title="Immediately check Lichess.org & Chess.com for your just-finished game and add it to your database"
            @click="handleCheckAndCaptureNow(false)"
          >
            <RefreshCw class="w-3 h-3" :class="isCheckingNow ? 'animate-spin' : ''" />
            <span>{{ isCheckingNow ? 'Checking...' : '⚡ Capture Latest Now' }}</span>
          </button>

          <button
            type="button"
            class="px-2 py-1 rounded-lg bg-zinc-900 hover:bg-zinc-800 border border-zinc-700 text-zinc-300 text-[11px] cursor-pointer"
            @click="showAccountSetup = !showAccountSetup"
          >
            {{ hasConfiguredUsername ? 'Accounts ▾' : 'Set Usernames ▾' }}
          </button>
        </div>
      </div>

      <!-- Configured Account Summary or Inline Username Setup -->
      <div v-if="!showAccountSetup && hasConfiguredUsername" class="flex flex-wrap items-center justify-between gap-2 pt-1 border-t border-zinc-800/80 text-[11px] text-zinc-400">
        <div class="flex items-center gap-3 flex-wrap">
          <span v-if="lichessUserInput" class="flex items-center gap-1">
            <CheckCircle2 class="w-3 h-3 text-emerald-400" />
            <span>Lichess: <strong class="text-zinc-200 font-mono">@{{ lichessUserInput }}</strong></span>
          </span>
          <span v-if="chessComUserInput" class="flex items-center gap-1">
            <CheckCircle2 class="w-3 h-3 text-emerald-400" />
            <span>Chess.com: <strong class="text-zinc-200 font-mono">@{{ chessComUserInput }}</strong></span>
          </span>
          <span class="text-[10px] text-zinc-500">
            (Polls automatically while you play — no manual sync needed)
          </span>
        </div>

        <div v-if="lastCapturedSummary" class="flex items-center gap-1.5">
          <span class="text-[10px] text-emerald-300 truncate max-w-[260px]">
            Last: {{ lastCapturedSummary }}
          </span>
          <button
            v-if="lastCapturedGameId > 0"
            type="button"
            class="px-1.5 py-0.5 rounded bg-emerald-900/60 hover:bg-emerald-800 text-emerald-200 text-[10px] font-semibold flex items-center gap-0.5 cursor-pointer"
            @click="emit('loadGameById', lastCapturedGameId)"
          >
            <Play class="w-2.5 h-2.5" />
            <span>Load #{{ lastCapturedGameId }}</span>
          </button>
        </div>
      </div>

      <!-- Inline Username Configuration Drawer -->
      <div v-if="showAccountSetup || !hasConfiguredUsername" class="pt-1.5 border-t border-zinc-800 grid grid-cols-1 sm:grid-cols-3 gap-2 items-end">
        <div>
          <label class="block text-[10px] text-zinc-400 mb-0.5 font-semibold">Lichess.org Username</label>
          <input
            v-model="lichessUserInput"
            type="text"
            placeholder="e.g. DrNykterstein"
            class="w-full bg-zinc-900 border border-zinc-700 rounded-lg px-2.5 py-1 text-xs text-zinc-100 placeholder-zinc-500 focus:outline-none focus:border-emerald-500"
            @keydown.enter="handleSaveAutoCaptureConfig"
          />
        </div>
        <div>
          <label class="block text-[10px] text-zinc-400 mb-0.5 font-semibold">Chess.com Username</label>
          <input
            v-model="chessComUserInput"
            type="text"
            placeholder="e.g. MagnusCarlsen"
            class="w-full bg-zinc-900 border border-zinc-700 rounded-lg px-2.5 py-1 text-xs text-zinc-100 placeholder-zinc-500 focus:outline-none focus:border-emerald-500"
            @keydown.enter="handleSaveAutoCaptureConfig"
          />
        </div>
        <div class="flex items-center gap-1.5">
          <button
            type="button"
            class="flex-1 px-3 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white font-bold text-xs cursor-pointer transition-colors"
            @click="handleSaveAutoCaptureConfig"
          >
            Save &amp; Enable Auto-Capture
          </button>
        </div>
      </div>
    </div>

    <!-- Status Toast -->
    <div
      v-if="statusMessage"
      class="px-3 py-1.5 rounded-lg text-[11px] flex items-center justify-between border"
      :class="isErrorToast
        ? 'bg-rose-950/80 border-rose-500/40 text-rose-200'
        : 'bg-emerald-950/80 border-emerald-500/40 text-emerald-200'"
    >
      <span>{{ statusMessage }}</span>
    </div>

    <!-- Quick Native Window Launchers (Lichess.org / Chess.com) -->
    <div class="bg-zinc-950/90 border border-zinc-800 rounded-xl p-3 space-y-2.5">
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-2">
          <Sparkles class="w-4 h-4 text-emerald-400" />
          <span class="font-bold text-zinc-100">
            {{ activePlatform === 'lichess' ? 'Lichess.org Direct Screen Windows' : 'Chess.com Direct Screen Windows' }}
          </span>
        </div>
        <span class="text-[10px] font-mono text-emerald-400">
          Finished games auto-save to {{ targetDbName }}
        </span>
      </div>

      <!-- Lichess Quick Actions -->
      <div v-if="activePlatform === 'lichess'" class="grid grid-cols-2 sm:grid-cols-3 gap-2">
        <button
          class="p-2.5 rounded-xl bg-zinc-900 hover:bg-zinc-800 border border-zinc-800 hover:border-emerald-500/50 text-left transition-all cursor-pointer group"
          @click="openPortalUrl('lichess', 'https://lichess.org')"
        >
          <div class="flex items-center justify-between text-emerald-400 font-bold mb-0.5">
            <span>Lichess Play Lobby</span>
            <ExternalLink class="w-3.5 h-3.5 opacity-70 group-hover:opacity-100" />
          </div>
          <p class="text-[10px] text-zinc-400">Play live games — auto-captured into {{ targetDbName }} when finished</p>
        </button>

        <button
          class="p-2.5 rounded-xl bg-zinc-900 hover:bg-zinc-800 border border-zinc-800 hover:border-cyan-500/50 text-left transition-all cursor-pointer group"
          @click="openPortalUrl('lichess', lichessAnalysisUrl)"
        >
          <div class="flex items-center justify-between text-cyan-400 font-bold mb-0.5">
            <span>Send Board to Lichess</span>
            <Compass class="w-3.5 h-3.5 opacity-70 group-hover:opacity-100" />
          </div>
          <p class="text-[10px] text-zinc-400">Open current NewScid FEN in Lichess Analysis</p>
        </button>

        <button
          class="p-2.5 rounded-xl bg-zinc-900 hover:bg-zinc-800 border border-zinc-800 hover:border-amber-500/50 text-left transition-all cursor-pointer group"
          @click="openPortalUrl('lichess', 'https://lichess.org/broadcast')"
        >
          <div class="flex items-center justify-between text-amber-400 font-bold mb-0.5">
            <span>Live Broadcasts</span>
            <Tv class="w-3.5 h-3.5 opacity-70 group-hover:opacity-100" />
          </div>
          <p class="text-[10px] text-zinc-400">Watch live FIDE &amp; super-GM tournament relays</p>
        </button>
      </div>

      <!-- Chess.com Quick Actions -->
      <div v-else class="grid grid-cols-2 sm:grid-cols-3 gap-2">
        <button
          class="p-2.5 rounded-xl bg-zinc-900 hover:bg-zinc-800 border border-zinc-800 hover:border-emerald-500/50 text-left transition-all cursor-pointer group"
          @click="openPortalUrl('chesscom', 'https://www.chess.com/play/online')"
        >
          <div class="flex items-center justify-between text-emerald-400 font-bold mb-0.5">
            <span>Chess.com Live Play</span>
            <ExternalLink class="w-3.5 h-3.5 opacity-70 group-hover:opacity-100" />
          </div>
          <p class="text-[10px] text-zinc-400">Play live games — auto-captured into {{ targetDbName }} when finished</p>
        </button>

        <button
          class="p-2.5 rounded-xl bg-zinc-900 hover:bg-zinc-800 border border-zinc-800 hover:border-cyan-500/50 text-left transition-all cursor-pointer group"
          @click="openPortalUrl('chesscom', chessComAnalysisUrl)"
        >
          <div class="flex items-center justify-between text-cyan-400 font-bold mb-0.5">
            <span>Send Board to Chess.com</span>
            <Compass class="w-3.5 h-3.5 opacity-70 group-hover:opacity-100" />
          </div>
          <p class="text-[10px] text-zinc-400">Open current NewScid FEN in Chess.com Analysis</p>
        </button>

        <button
          class="p-2.5 rounded-xl bg-zinc-900 hover:bg-zinc-800 border border-zinc-800 hover:border-amber-500/50 text-left transition-all cursor-pointer group"
          @click="openPortalUrl('chesscom', 'https://www.chess.com/events')"
        >
          <div class="flex items-center justify-between text-amber-400 font-bold mb-0.5">
            <span>Chess.com Events &amp; TV</span>
            <Tv class="w-3.5 h-3.5 opacity-70 group-hover:opacity-100" />
          </div>
          <p class="text-[10px] text-zinc-400">Follow live Championships, Titled Tuesday &amp; CCT</p>
        </button>
      </div>

      <!-- Custom URL / Game Link Input + 1-Click Capture Game to DB -->
      <div class="flex items-center gap-1.5 pt-1 flex-wrap sm:flex-nowrap">
        <input
          v-model="customUrl"
          type="text"
          placeholder="Paste any Lichess/Chess.com game URL (e.g. lichess.org/AbCdEfGh or chess.com/game/live/...)..."
          class="flex-1 bg-zinc-900 border border-zinc-800 rounded-lg px-2.5 py-1.5 text-xs text-zinc-100 placeholder-zinc-500 focus:outline-none focus:border-emerald-500"
          @keydown.enter="handleCaptureUrlOrPgn"
        />
        <button
          type="button"
          class="px-3 py-1.5 rounded-lg bg-amber-600 hover:bg-amber-500 text-white font-semibold shrink-0 cursor-pointer transition-colors disabled:opacity-50"
          :disabled="!customUrl.trim() || isCapturingUrl"
          title="Download this specific Lichess or Chess.com game URL and add it directly to your Personal Database"
          @click="handleCaptureUrlOrPgn"
        >
          {{ isCapturingUrl ? 'Capturing...' : '📥 Capture Game to DB' }}
        </button>
        <button
          type="button"
          class="px-3 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white font-semibold shrink-0 cursor-pointer transition-colors"
          @click="handleLaunchCustomUrl"
        >
          Open Window
        </button>
      </div>
    </div>

    <!-- Embedded Live Widget + Sync Bridge -->
    <div class="flex-1 min-h-[260px] bg-zinc-950 border border-zinc-800 rounded-xl flex flex-col overflow-hidden">
      <div class="px-3 py-1.5 border-b border-zinc-800 flex items-center justify-between bg-zinc-900/60">
        <div class="flex items-center gap-1.5">
          <button
            class="px-2.5 py-1 rounded-md text-[11px] font-semibold flex items-center gap-1 cursor-pointer transition-colors"
            :class="embedMode === 'tv' ? 'bg-emerald-600 text-white' : 'text-zinc-400 hover:text-zinc-200 hover:bg-zinc-800'"
            @click="embedMode = 'tv'"
          >
            <Tv class="w-3 h-3" />
            <span>Embedded Live GM TV</span>
          </button>
          <button
            class="px-2.5 py-1 rounded-md text-[11px] font-semibold flex items-center gap-1 cursor-pointer transition-colors"
            :class="embedMode === 'puzzle' ? 'bg-emerald-600 text-white' : 'text-zinc-400 hover:text-zinc-200 hover:bg-zinc-800'"
            @click="embedMode = 'puzzle'"
          >
            <Zap class="w-3 h-3" />
            <span>Embedded Live Puzzle</span>
          </button>
          <button
            class="px-2.5 py-1 rounded-md text-[11px] font-semibold flex items-center gap-1 cursor-pointer transition-colors"
            :class="embedMode === 'bridge' ? 'bg-emerald-600 text-white' : 'text-zinc-400 hover:text-zinc-200 hover:bg-zinc-800'"
            @click="embedMode = 'bridge'"
          >
            <Shield class="w-3 h-3" />
            <span>Historical Backfill &amp; Fair-Play Bridge</span>
          </button>
        </div>

        <button
          v-if="embedMode !== 'bridge'"
          class="p-1 rounded hover:bg-zinc-800 text-zinc-400 hover:text-zinc-200 cursor-pointer"
          title="Reload embedded live board"
          @click="iframeKey++"
        >
          <RefreshCw class="w-3.5 h-3.5" />
        </button>
      </div>

      <!-- Iframe for Lichess TV / Puzzle -->
      <div v-if="embedMode !== 'bridge'" class="flex-1 flex items-center justify-center bg-zinc-950 p-2 overflow-hidden">
        <iframe
          :key="`${embedMode}-${iframeKey}`"
          :src="activeEmbedSrc"
          class="w-full h-full min-h-[240px] max-w-[440px] border-0 rounded-lg shadow-inner bg-zinc-900"
          allowtransparency="true"
          frameborder="0"
        />
      </div>

      <!-- Account Sync & Fair-Play Bridge -->
      <div v-else class="flex-1 p-4 space-y-3 overflow-y-auto">
        <p class="text-zinc-300 leading-relaxed">
          While <strong>Auto-Capture to DB</strong> automatically saves new games as you finish playing them in the Live Portal, you can also run a full historical backfill or multi-time-control Fair-Play audit below:
        </p>

        <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
          <button
            class="p-3.5 rounded-xl bg-zinc-900 hover:bg-zinc-800 border border-zinc-800 hover:border-cyan-500/50 text-left transition-all cursor-pointer"
            @click="emit('openOnlineSync')"
          >
            <div class="flex items-center gap-2 text-cyan-400 font-bold mb-1">
              <Download class="w-4 h-4" />
              <span>Historical Account Sync (Lichess &amp; Chess.com)</span>
            </div>
            <p class="text-[11px] text-zinc-400">
              Backfill thousands of older historical games from your online archives into your database.
            </p>
          </button>

          <button
            class="p-3.5 rounded-xl bg-zinc-900 hover:bg-zinc-800 border border-zinc-800 hover:border-amber-500/50 text-left transition-all cursor-pointer"
            @click="emit('openFairPlay')"
          >
            <div class="flex items-center gap-2 text-amber-400 font-bold mb-1">
              <Shield class="w-4 h-4" />
              <span>Fair-Play &amp; Time-Control Consistency Audit</span>
            </div>
            <p class="text-[11px] text-zinc-400">
              Download any player's games across Bullet, Blitz, Rapid &amp; Classical to inspect consistent issue scores.
            </p>
          </button>
        </div>

        <div class="p-3 rounded-xl bg-zinc-900/70 border border-zinc-800/80 text-[11px] font-mono text-zinc-400 space-y-1">
          <div class="text-zinc-200 font-bold">Live Synced Position (FEN):</div>
          <div class="truncate text-emerald-400">{{ currentFen }}</div>
        </div>
      </div>
    </div>
  </div>
</template>
