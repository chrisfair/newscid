<script setup lang="ts">
import { ref, computed } from 'vue'
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
  Copy
} from '@lucide/vue'
import { LaunchPortalWindow } from '../../wailsjs/go/main/App'
import { BrowserOpenURL } from '../../wailsjs/runtime/runtime'

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
}>()

const activePlatform = ref<'lichess' | 'chesscom'>('lichess')
const embedMode = ref<'tv' | 'puzzle' | 'bridge'>('tv')
const windowMode = ref<string>(props.defaultWindowMode || 'browser-app')
const customUrl = ref('')
const statusMessage = ref<string | null>(null)
const iframeKey = ref(1)

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

async function openPortalUrl(platform: 'lichess' | 'chesscom', url: string) {
  statusMessage.value = `Launching ${platform === 'chesscom' ? 'Chess.com' : 'Lichess.org'} window...`
  try {
    await LaunchPortalWindow(platform, url, windowMode.value)
    statusMessage.value = `Opened ${url} in ${windowMode.value === 'native-window' ? 'Native NewScid Window' : 'Standalone App Window'}`
  } catch (err: any) {
    try {
      BrowserOpenURL(url)
      statusMessage.value = `Opened ${url} in system browser`
    } catch {
      window.open(url, '_blank')
    }
  }
  setTimeout(() => {
    statusMessage.value = null
  }, 4000)
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

    <!-- Status Toast -->
    <div
      v-if="statusMessage"
      class="px-3 py-1.5 rounded-lg bg-emerald-950/80 border border-emerald-500/40 text-emerald-200 text-[11px] flex items-center justify-between"
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
        <span class="text-[10px] font-mono text-zinc-400">Hyprland Multi-Monitor Ready</span>
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
          <p class="text-[10px] text-zinc-400">Open full Lichess pairing lobby &amp; live games</p>
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
          <p class="text-[10px] text-zinc-400">Open dedicated Chess.com online play window</p>
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

      <!-- Custom URL / Game Link Input -->
      <div class="flex items-center gap-1.5 pt-1">
        <input
          v-model="customUrl"
          type="text"
          placeholder="Paste any Chess.com or Lichess URL (e.g. lichess.org/study/... or chess.com/game/live/...)..."
          class="flex-1 bg-zinc-900 border border-zinc-800 rounded-lg px-2.5 py-1.5 text-xs text-zinc-100 placeholder-zinc-500 focus:outline-none focus:border-emerald-500"
          @keydown.enter="handleLaunchCustomUrl"
        />
        <button
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
            <span>Account Sync &amp; Fair-Play Bridge</span>
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
          Pull your everyday games from <strong>Chess.com</strong> and <strong>Lichess.org</strong> directly into your NewScid Default or Active Database, or run a multi-time-control Fair-Play audit on any online opponent.
        </p>

        <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
          <button
            class="p-3.5 rounded-xl bg-zinc-900 hover:bg-zinc-800 border border-zinc-800 hover:border-cyan-500/50 text-left transition-all cursor-pointer"
            @click="emit('openOnlineSync')"
          >
            <div class="flex items-center gap-2 text-cyan-400 font-bold mb-1">
              <Download class="w-4 h-4" />
              <span>Sync Online Accounts (Lichess &amp; Chess.com)</span>
            </div>
            <p class="text-[11px] text-zinc-400">
              Download and deduplicate your latest online games into your working database.
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
