<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import {
  ShieldAlert,
  Search,
  Globe,
  Database,
  Activity,
  Calendar,
  Clock,
  AlertTriangle,
  CheckCircle,
  Copy,
  ExternalLink,
  X,
  TrendingUp,
} from '@lucide/vue'
import {
  AnalyzePlayerFairPlay,
  DownloadAndAnalyzePlayerFairPlay,
  GetDatabasePlayers,
} from '../../wailsjs/go/main/App'

const props = defineProps<{
  dbOpen?: boolean
  initialPlayer?: string
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'select-game', gameId: number): void
  (e: 'imported', count: number): void
}>()

interface GameFairPlayRecord {
  gameId: number
  date: string
  event: string
  site: string
  timeControlRaw: string
  timeControlCategory: string
  playerColor: string
  playerElo: number
  opponentName: string
  opponentElo: number
  result: string
  playerScore: number
  totalPlies: number
  nonBookMoves: number
  hasClockData: boolean
  hasEvalData: boolean
  avgMoveTimeSec: number
  moveTimeStdDevSec: number
  moveTimeCV: number
  roboticCadencePct: number
  delayedRecaptureCount: number
  acpl: number
  top1AgreementPct: number
  criticalAccuracyPct: number
  blunderCount: number
  intrinsicPerformanceElo: number
  gameIssueScore: number
  flags: string[]
}

interface TimeControlIssueProfile {
  category: string
  gamesAnalyzed: number
  daysActive: number
  daysWithElevatedIssue: number
  dailyConsistencyPct: number
  winRatePct: number
  avgOfficialElo: number
  intrinsicPerformanceElo: number
  eloOverperformanceDelta: number
  avgAcpl: number
  top1AgreementPct: number
  criticalAccuracyPct: number
  blunderFreeGamePct: number
  avgMoveTimeSec: number
  avgMoveTimeCV: number
  roboticCadenceGamePct: number
  consistentIssueScore: number
  zScore: number
  consistencyClassification: string
  summary: string
}

interface DailyFairPlaySummary {
  date: string
  gamesCount: number
  timeControls: string[]
  primaryTimeControl: string
  winRatePct: number
  avgAcpl: number
  top1AgreementPct: number
  avgMoveTimeCV: number
  intrinsicPerformance: number
  dailyIssueScore: number
  peakGameId: number
  classification: string
}

interface CrossControlDivergence {
  hasMultipleControls: boolean
  baselineControl: string
  suspectControl: string
  iprDelta: number
  acplDelta: number
  cadenceCvDelta: number
  divergenceScore: number
  explanation: string
}

interface PlayerFairPlayDossier {
  playerName: string
  totalGamesAnalyzed: number
  totalDaysActive: number
  overallIssueScore: number
  highestRiskTimeControl: string
  highestControlIssueScore: number
  overallClassification: string
  timeControlProfiles: TimeControlIssueProfile[]
  dailyTimeline: DailyFairPlaySummary[]
  crossDivergence: CrossControlDivergence
  flaggedGames: GameFairPlayRecord[]
  allGames: GameFairPlayRecord[]
  keyFindings: string[]
  reportMarkdown: string
}

const sourceMode = ref<'database' | 'online'>(props.dbOpen ? 'database' : 'online')
const playerName = ref<string>(props.initialPlayer || '')
const onlinePlatform = ref<'lichess' | 'chesscom' | 'both'>('lichess')
const maxGames = ref<number>(250)
const dbPlayers = ref<string[]>([])

const isAnalyzing = ref<boolean>(false)
const errorMsg = ref<string>('')
const copiedReport = ref<boolean>(false)
const dossier = ref<PlayerFairPlayDossier | null>(null)
const activeViewTab = ref<'matrix' | 'daily' | 'games' | 'report'>('matrix')
const selectedControlFilter = ref<string>('ALL')

onMounted(async () => {
  if (props.dbOpen) {
    try {
      const list = await GetDatabasePlayers()
      if (Array.isArray(list)) {
        dbPlayers.value = list.slice(0, 200)
        if (!playerName.value && list.length > 0) {
          playerName.value = list[0]
        }
      }
    } catch {
      // Ignore in test environment
    }
  }
})

const filteredFlaggedGames = computed(() => {
  if (!dossier.value) return []
  const list = dossier.value.flaggedGames || []
  if (selectedControlFilter.value === 'ALL') return list
  return list.filter(g => g.timeControlCategory === selectedControlFilter.value)
})

function scoreBadgeClass(score: number): string {
  if (score >= 75) return 'bg-rose-500/20 text-rose-300 border-rose-500/40'
  if (score >= 55) return 'bg-amber-500/20 text-amber-300 border-amber-500/40'
  if (score >= 38) return 'bg-yellow-500/15 text-yellow-200 border-yellow-500/30'
  return 'bg-emerald-500/15 text-emerald-300 border-emerald-500/30'
}

function scoreBarClass(score: number): string {
  if (score >= 75) return 'bg-rose-500'
  if (score >= 55) return 'bg-amber-500'
  if (score >= 38) return 'bg-yellow-400'
  return 'bg-emerald-500'
}

async function handleRunInspection() {
  const target = playerName.value.trim()
  if (!target) {
    errorMsg.value = 'Please enter a player name or online username.'
    return
  }
  isAnalyzing.value = true
  errorMsg.value = ''
  copiedReport.value = false

  try {
    if (sourceMode.value === 'online') {
      const res = await DownloadAndAnalyzePlayerFairPlay(onlinePlatform.value, target, Number(maxGames.value) || 250)
      dossier.value = res
      emit('imported', res?.totalGamesAnalyzed ?? 0)
    } else {
      const res = await AnalyzePlayerFairPlay(target, Number(maxGames.value) || 250)
      dossier.value = res
    }
  } catch (err: any) {
    errorMsg.value = typeof err === 'string' ? err : (err?.message || 'Failed to run fair-play & time-control analysis')
  } finally {
    isAnalyzing.value = false
  }
}

async function handleCopyReport() {
  if (!dossier.value?.reportMarkdown) return
  try {
    await navigator.clipboard.writeText(dossier.value.reportMarkdown)
    copiedReport.value = true
    setTimeout(() => {
      copiedReport.value = false
    }, 2500)
  } catch {
    copiedReport.value = true
  }
}
</script>

<template>
  <div
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/75 backdrop-blur-xs p-4"
    data-testid="fairplay-modal"
  >
    <div class="bg-zinc-900 border border-zinc-700 rounded-2xl shadow-2xl w-full max-w-5xl max-h-[92vh] flex flex-col overflow-hidden text-zinc-100">
      <!-- Header -->
      <div class="flex items-center justify-between px-5 py-4 bg-zinc-950 border-b border-zinc-800 shrink-0">
        <div class="flex items-center gap-3">
          <div class="p-2 rounded-xl bg-amber-500/15 border border-amber-500/30 text-amber-400">
            <ShieldAlert class="w-5 h-5" />
          </div>
          <div>
            <h2 class="text-base font-bold text-white flex items-center gap-2">
              <span>Longitudinal Fair-Play &amp; Time-Control Consistency Inspector</span>
              <span class="text-[10px] font-mono uppercase px-2 py-0.5 rounded-full bg-amber-500/20 text-amber-300 border border-amber-500/30">
                Multi-Control &amp; Everyday Dossier
              </span>
            </h2>
            <p class="text-xs text-zinc-400">
              Measures consistent statistical anomalies across Bullet, Blitz, Rapid &amp; Classical and tracks everyday session consistency (IPR, ACPL &amp; Move-Time CV)
            </p>
          </div>
        </div>
        <button
          type="button"
          class="p-1.5 rounded-lg text-zinc-400 hover:text-white hover:bg-zinc-800 cursor-pointer"
          @click="emit('close')"
        >
          <X class="w-5 h-5" />
        </button>
      </div>

      <!-- Body -->
      <div class="p-5 overflow-y-auto space-y-4 flex-1 text-xs">
        <!-- Control Bar: Source Mode + Player Input -->
        <div class="bg-zinc-950 border border-zinc-800 rounded-xl p-4 space-y-3">
          <div class="flex flex-wrap items-center justify-between gap-2">
            <div class="flex items-center gap-2">
              <button
                type="button"
                data-testid="mode-database-btn"
                class="px-3 py-1.5 rounded-lg font-medium flex items-center gap-1.5 transition-colors cursor-pointer"
                :class="sourceMode === 'database' ? 'bg-amber-600 text-white' : 'bg-zinc-900 text-zinc-400 hover:text-zinc-200'"
                @click="sourceMode = 'database'"
              >
                <Database class="w-3.5 h-3.5" />
                <span>Analyze Player in Open Database</span>
              </button>
              <button
                type="button"
                data-testid="mode-online-btn"
                class="px-3 py-1.5 rounded-lg font-medium flex items-center gap-1.5 transition-colors cursor-pointer"
                :class="sourceMode === 'online' ? 'bg-sky-600 text-white' : 'bg-zinc-900 text-zinc-400 hover:text-zinc-200'"
                @click="sourceMode = 'online'"
              >
                <Globe class="w-3.5 h-3.5" />
                <span>Fetch &amp; Inspect Online Player (All Time Controls)</span>
              </button>
            </div>

            <div class="text-[11px] text-zinc-400">
              Objective statistical metrics (Ken Regan IPR / Move-Time CV / Cross-Control Z-Score)
            </div>
          </div>

          <div class="grid grid-cols-1 md:grid-cols-12 gap-3 items-end">
            <!-- Platform selector when in Online mode -->
            <div v-if="sourceMode === 'online'" class="md:col-span-3 space-y-1">
              <label class="block text-[11px] text-zinc-400">Platform</label>
              <select
                v-model="onlinePlatform"
                data-testid="online-platform-select"
                class="w-full bg-zinc-900 border border-zinc-700 rounded-lg px-2.5 py-2 text-xs text-zinc-100"
              >
                <option value="lichess">Lichess.org (Clocks + Evals)</option>
                <option value="chesscom">Chess.com (All Archives)</option>
                <option value="both">Both Lichess &amp; Chess.com</option>
              </select>
            </div>

            <!-- Player Name / Username -->
            <div :class="sourceMode === 'online' ? 'md:col-span-4' : 'md:col-span-7'" class="space-y-1">
              <label class="block text-[11px] text-zinc-400">
                {{ sourceMode === 'online' ? 'Online Username' : 'Player Name in Database' }}
              </label>
              <input
                v-model="playerName"
                list="fairplay-db-players"
                data-testid="fairplay-player-input"
                type="text"
                placeholder="Enter username or select player..."
                class="w-full bg-zinc-900 border border-zinc-700 rounded-lg px-3 py-2 text-xs text-zinc-100 focus:outline-hidden focus:border-amber-500"
                @keyup.enter="handleRunInspection"
              />
              <datalist id="fairplay-db-players">
                <option v-for="p in dbPlayers" :key="p" :value="p" />
              </datalist>
            </div>

            <!-- Max Games Sample -->
            <div class="md:col-span-2 space-y-1">
              <label class="block text-[11px] text-zinc-400">Max Games</label>
              <select
                v-model.number="maxGames"
                data-testid="fairplay-max-games"
                class="w-full bg-zinc-900 border border-zinc-700 rounded-lg px-2.5 py-2 text-xs text-zinc-100"
              >
                <option :value="100">100 games</option>
                <option :value="250">250 games</option>
                <option :value="500">500 games</option>
                <option :value="1000">1,000 games</option>
              </select>
            </div>

            <!-- Action Button -->
            <div class="md:col-span-3">
              <button
                type="button"
                data-testid="run-fairplay-btn"
                :disabled="isAnalyzing"
                class="w-full px-4 py-2 rounded-lg bg-amber-600 hover:bg-amber-500 text-white font-bold flex items-center justify-center gap-2 shadow-lg cursor-pointer disabled:opacity-50"
                @click="handleRunInspection"
              >
                <Search class="w-4 h-4" />
                <span>{{ isAnalyzing ? 'Inspecting Games...' : (sourceMode === 'online' ? 'Fetch & Inspect' : 'Run Inspection') }}</span>
              </button>
            </div>
          </div>
        </div>

        <!-- Error Message -->
        <div
          v-if="errorMsg"
          data-testid="fairplay-error"
          class="p-3 rounded-xl bg-rose-950/70 border border-rose-500/40 flex items-center gap-2 text-rose-200"
        >
          <AlertTriangle class="w-4 h-4 text-rose-400 shrink-0" />
          <span>{{ errorMsg }}</span>
        </div>

        <!-- Dossier Results -->
        <div v-if="dossier" class="space-y-4" data-testid="fairplay-dossier">
          <!-- Top Executive Summary Cards -->
          <div class="grid grid-cols-1 md:grid-cols-4 gap-3">
            <div class="bg-zinc-950 border border-zinc-800 rounded-xl p-3.5">
              <div class="text-[11px] text-zinc-400">Player &amp; Longitudinal Scope</div>
              <div class="text-sm font-bold text-white mt-0.5 truncate">{{ dossier.playerName }}</div>
              <div class="text-[11px] text-zinc-400 mt-1">
                <span class="text-zinc-200 font-semibold">{{ dossier.totalGamesAnalyzed }}</span> games across
                <span class="text-zinc-200 font-semibold">{{ dossier.totalDaysActive }}</span> active day(s)
              </div>
            </div>

            <div class="bg-zinc-950 border border-zinc-800 rounded-xl p-3.5">
              <div class="text-[11px] text-zinc-400">Highest-Issue Time Control</div>
              <div class="flex items-center justify-between mt-0.5">
                <span class="text-sm font-bold text-amber-300" data-testid="highest-risk-tc">
                  {{ dossier.highestRiskTimeControl || 'N/A' }}
                </span>
                <span
                  class="px-2 py-0.5 rounded-full text-[11px] font-mono font-bold border"
                  :class="scoreBadgeClass(dossier.highestControlIssueScore)"
                >
                  {{ dossier.highestControlIssueScore }} / 100
                </span>
              </div>
              <div class="text-[11px] text-zinc-400 mt-1">
                Consistent Time-Control Issue Score
              </div>
            </div>

            <div class="bg-zinc-950 border border-zinc-800 rounded-xl p-3.5">
              <div class="text-[11px] text-zinc-400">Cross-Control Divergence</div>
              <div class="flex items-center justify-between mt-0.5">
                <span class="text-sm font-bold text-sky-300">
                  <template v-if="dossier.crossDivergence?.hasMultipleControls">
                    {{ dossier.crossDivergence.suspectControl }} vs {{ dossier.crossDivergence.baselineControl }}
                  </template>
                  <template v-else>Single Control</template>
                </span>
                <span
                  v-if="dossier.crossDivergence?.hasMultipleControls"
                  class="text-xs font-mono font-bold text-sky-400"
                >
                  {{ dossier.crossDivergence.iprDelta >= 0 ? `+${dossier.crossDivergence.iprDelta}` : dossier.crossDivergence.iprDelta }} IPR
                </span>
              </div>
              <div class="text-[11px] text-zinc-400 mt-1 truncate">
                Divergence Index: <span class="text-zinc-200 font-semibold">{{ dossier.crossDivergence?.divergenceScore ?? 0 }}/100</span>
              </div>
            </div>

            <div class="bg-zinc-950 border border-zinc-800 rounded-xl p-3.5">
              <div class="text-[11px] text-zinc-400">Overall Statistical Assessment</div>
              <div class="flex items-center justify-between mt-0.5">
                <span
                  class="px-2 py-0.5 rounded-full text-[11px] font-bold border"
                  :class="scoreBadgeClass(dossier.overallIssueScore)"
                  data-testid="overall-classification"
                >
                  {{ dossier.overallClassification }}
                </span>
                <span class="text-sm font-mono font-bold text-white">{{ dossier.overallIssueScore }}</span>
              </div>
              <div class="w-full h-1.5 bg-zinc-800 rounded-full overflow-hidden mt-2">
                <div
                  class="h-full transition-all"
                  :class="scoreBarClass(dossier.overallIssueScore)"
                  :style="{ width: `${Math.min(100, Math.max(4, dossier.overallIssueScore))}%` }"
                />
              </div>
            </div>
          </div>

          <!-- Cross-Time-Control Divergence Alert Banner -->
          <div
            v-if="dossier.crossDivergence?.hasMultipleControls"
            data-testid="cross-control-banner"
            class="p-3.5 rounded-xl bg-sky-950/40 border border-sky-500/30 flex items-start justify-between gap-3"
          >
            <div class="space-y-1">
              <div class="font-bold text-sky-300 flex items-center gap-1.5">
                <TrendingUp class="w-4 h-4" />
                <span>Cross-Time-Control Divergence Analysis ({{ dossier.crossDivergence.suspectControl }} vs. {{ dossier.crossDivergence.baselineControl }} Baseline)</span>
              </div>
              <p class="text-zinc-300 leading-relaxed">{{ dossier.crossDivergence.explanation }}</p>
            </div>
            <span class="px-2.5 py-1 rounded-lg bg-sky-500/20 border border-sky-500/40 text-sky-200 font-mono font-bold shrink-0">
              {{ dossier.crossDivergence.divergenceScore }} / 100
            </span>
          </div>

          <!-- Navigation Tabs for Dossier Views -->
          <div class="flex items-center justify-between border-b border-zinc-800 pb-2">
            <div class="flex items-center gap-2">
              <button
                type="button"
                data-testid="tab-matrix"
                class="px-3 py-1.5 rounded-lg font-semibold flex items-center gap-1.5 cursor-pointer"
                :class="activeViewTab === 'matrix' ? 'bg-amber-500/20 text-amber-300 border border-amber-500/40' : 'text-zinc-400 hover:text-zinc-200'"
                @click="activeViewTab = 'matrix'"
              >
                <Clock class="w-3.5 h-3.5" />
                <span>Per-Time-Control Issue Scores ({{ dossier.timeControlProfiles?.length || 0 }})</span>
              </button>
              <button
                type="button"
                data-testid="tab-daily"
                class="px-3 py-1.5 rounded-lg font-semibold flex items-center gap-1.5 cursor-pointer"
                :class="activeViewTab === 'daily' ? 'bg-amber-500/20 text-amber-300 border border-amber-500/40' : 'text-zinc-400 hover:text-zinc-200'"
                @click="activeViewTab = 'daily'"
              >
                <Calendar class="w-3.5 h-3.5" />
                <span>Everyday Consistency Timeline ({{ dossier.dailyTimeline?.length || 0 }} Days)</span>
              </button>
              <button
                type="button"
                data-testid="tab-games"
                class="px-3 py-1.5 rounded-lg font-semibold flex items-center gap-1.5 cursor-pointer"
                :class="activeViewTab === 'games' ? 'bg-amber-500/20 text-amber-300 border border-amber-500/40' : 'text-zinc-400 hover:text-zinc-200'"
                @click="activeViewTab = 'games'"
              >
                <Activity class="w-3.5 h-3.5" />
                <span>Flagged Games ({{ dossier.flaggedGames?.length || 0 }})</span>
              </button>
            </div>

            <button
              type="button"
              data-testid="copy-fairplay-report-btn"
              class="px-3 py-1.5 rounded-lg bg-zinc-800 hover:bg-zinc-700 text-zinc-200 font-medium flex items-center gap-1.5 cursor-pointer"
              @click="handleCopyReport"
            >
              <CheckCircle v-if="copiedReport" class="w-3.5 h-3.5 text-emerald-400" />
              <Copy v-else class="w-3.5 h-3.5 text-amber-400" />
              <span>{{ copiedReport ? 'Copied Markdown Report!' : 'Copy Official Statistical Report' }}</span>
            </button>
          </div>

          <!-- TAB 1: Per-Time-Control Consistent Issue Matrix -->
          <div v-if="activeViewTab === 'matrix'" class="space-y-3" data-testid="time-control-matrix">
            <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
              <div
                v-for="prof in dossier.timeControlProfiles"
                :key="prof.category"
                :data-testid="`tc-card-${prof.category}`"
                class="bg-zinc-950 border border-zinc-800 rounded-xl p-4 space-y-3"
              >
                <div class="flex items-center justify-between">
                  <div>
                    <div class="text-sm font-bold text-white flex items-center gap-2">
                      <span>{{ prof.category }}</span>
                      <span class="text-[11px] font-normal text-zinc-400">
                        ({{ prof.gamesAnalyzed }} games • {{ prof.daysActive }} day{{ prof.daysActive === 1 ? '' : 's' }})
                      </span>
                    </div>
                    <div class="text-[11px] text-zinc-400 mt-0.5">
                      {{ prof.consistencyClassification }}
                    </div>
                  </div>
                  <div class="text-right">
                    <div
                      class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-lg border font-mono font-bold text-xs"
                      :class="scoreBadgeClass(prof.consistentIssueScore)"
                    >
                      <span>Issue Score: {{ prof.consistentIssueScore }}</span>
                      <span class="text-[10px] opacity-80">({{ prof.zScore >= 0 ? `+${prof.zScore}` : prof.zScore }}σ)</span>
                    </div>
                  </div>
                </div>

                <!-- Score bar -->
                <div class="w-full h-2 bg-zinc-800 rounded-full overflow-hidden">
                  <div
                    class="h-full transition-all"
                    :class="scoreBarClass(prof.consistentIssueScore)"
                    :style="{ width: `${Math.min(100, Math.max(5, prof.consistentIssueScore))}%` }"
                  />
                </div>

                <!-- Key metrics grid -->
                <div class="grid grid-cols-3 gap-2 pt-1 text-[11px]">
                  <div class="bg-zinc-900/90 rounded-lg p-2 border border-zinc-800/80">
                    <div class="text-zinc-400">Official vs IPR</div>
                    <div class="font-mono font-bold text-zinc-100 mt-0.5">
                      {{ prof.avgOfficialElo }} → <span class="text-amber-300">{{ prof.intrinsicPerformanceElo }}</span>
                      <span class="text-[10px] ml-1" :class="prof.eloOverperformanceDelta > 200 ? 'text-rose-400' : 'text-zinc-400'">
                        ({{ prof.eloOverperformanceDelta >= 0 ? `+${prof.eloOverperformanceDelta}` : prof.eloOverperformanceDelta }})
                      </span>
                    </div>
                  </div>

                  <div class="bg-zinc-900/90 rounded-lg p-2 border border-zinc-800/80">
                    <div class="text-zinc-400">ACPL &amp; Top-1 %</div>
                    <div class="font-mono font-bold text-zinc-100 mt-0.5">
                      {{ prof.avgAcpl }} cp • <span class="text-emerald-400">{{ prof.top1AgreementPct }}%</span>
                    </div>
                  </div>

                  <div class="bg-zinc-900/90 rounded-lg p-2 border border-zinc-800/80">
                    <div class="text-zinc-400">Move-Time CV</div>
                    <div class="font-mono font-bold mt-0.5" :class="prof.avgMoveTimeCV <= 0.38 ? 'text-rose-400' : 'text-zinc-100'">
                      {{ prof.avgMoveTimeCV }} <span class="text-[10px] text-zinc-400">({{ prof.avgMoveTimeSec }}s avg)</span>
                    </div>
                  </div>
                </div>

                <div class="flex items-center justify-between text-[11px] text-zinc-400 pt-1 border-t border-zinc-800/80">
                  <span>
                    Daily Consistency: <strong class="text-zinc-200">{{ prof.dailyConsistencyPct }}%</strong>
                    ({{ prof.daysWithElevatedIssue }}/{{ prof.daysActive }} days elevated)
                  </span>
                  <span>
                    Win Rate: <strong class="text-zinc-200">{{ prof.winRatePct }}%</strong>
                  </span>
                </div>
              </div>
            </div>
          </div>

          <!-- TAB 2: Everyday Consistency Timeline -->
          <div v-if="activeViewTab === 'daily'" class="bg-zinc-950 border border-zinc-800 rounded-xl overflow-hidden" data-testid="daily-timeline-table">
            <table class="w-full text-left border-collapse">
              <thead>
                <tr class="border-b border-zinc-800 text-[11px] text-zinc-400 bg-zinc-900/60">
                  <th class="py-2.5 px-3">Date</th>
                  <th class="py-2.5 px-3">Time Controls</th>
                  <th class="py-2.5 px-3">Games</th>
                  <th class="py-2.5 px-3">Win %</th>
                  <th class="py-2.5 px-3">Avg ACPL</th>
                  <th class="py-2.5 px-3">Top-1 %</th>
                  <th class="py-2.5 px-3">Move-Time CV</th>
                  <th class="py-2.5 px-3">Daily IPR</th>
                  <th class="py-2.5 px-3">Daily Issue Score</th>
                  <th class="py-2.5 px-3 text-right">Peak Game</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-zinc-800/70 text-xs">
                <tr v-for="day in dossier.dailyTimeline" :key="day.date" class="hover:bg-zinc-900/50">
                  <td class="py-2.5 px-3 font-mono font-semibold text-white">{{ day.date }}</td>
                  <td class="py-2.5 px-3">
                    <span
                      v-for="tc in day.timeControls"
                      :key="tc"
                      class="inline-block mr-1 px-1.5 py-0.5 rounded bg-zinc-800 text-zinc-300 text-[10px] font-mono"
                    >
                      {{ tc }}
                    </span>
                  </td>
                  <td class="py-2.5 px-3 font-mono">{{ day.gamesCount }}</td>
                  <td class="py-2.5 px-3 font-mono">{{ day.winRatePct }}%</td>
                  <td class="py-2.5 px-3 font-mono">{{ day.avgAcpl }} cp</td>
                  <td class="py-2.5 px-3 font-mono">{{ day.top1AgreementPct }}%</td>
                  <td class="py-2.5 px-3 font-mono" :class="day.avgMoveTimeCV <= 0.38 ? 'text-rose-400 font-bold' : ''">
                    {{ day.avgMoveTimeCV }}
                  </td>
                  <td class="py-2.5 px-3 font-mono text-amber-300 font-semibold">{{ day.intrinsicPerformance }}</td>
                  <td class="py-2.5 px-3">
                    <span
                      class="px-2 py-0.5 rounded-full text-[11px] font-mono font-bold border"
                      :class="scoreBadgeClass(day.dailyIssueScore)"
                    >
                      {{ day.dailyIssueScore }}
                    </span>
                  </td>
                  <td class="py-2.5 px-3 text-right">
                    <button
                      type="button"
                      class="px-2 py-1 rounded bg-zinc-800 hover:bg-zinc-700 text-amber-300 text-[11px] font-mono inline-flex items-center gap-1 cursor-pointer"
                      @click="emit('select-game', day.peakGameId)"
                    >
                      <span>#{{ day.peakGameId }}</span>
                      <ExternalLink class="w-3 h-3" />
                    </button>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>

          <!-- TAB 3: Flagged Games Dossier -->
          <div v-if="activeViewTab === 'games'" class="space-y-2.5" data-testid="flagged-games-list">
            <div class="flex items-center gap-2">
              <span class="text-[11px] text-zinc-400">Filter by Time Control:</span>
              <button
                v-for="tc in ['ALL', 'Bullet', 'Blitz', 'Rapid', 'Classical']"
                :key="tc"
                type="button"
                class="px-2.5 py-1 rounded text-[11px] font-mono cursor-pointer"
                :class="selectedControlFilter === tc ? 'bg-amber-600 text-white' : 'bg-zinc-950 text-zinc-400 hover:text-zinc-200'"
                @click="selectedControlFilter = tc"
              >
                {{ tc }}
              </button>
            </div>

            <div
              v-for="g in filteredFlaggedGames"
              :key="g.gameId"
              class="bg-zinc-950 border border-zinc-800 rounded-xl p-3.5 flex flex-col md:flex-row md:items-center justify-between gap-3"
            >
              <div class="space-y-1">
                <div class="flex items-center gap-2 flex-wrap">
                  <span class="font-mono font-bold text-amber-400">Game #{{ g.gameId }}</span>
                  <span class="px-2 py-0.5 rounded bg-zinc-800 text-zinc-200 font-mono text-[10px]">
                    {{ g.timeControlCategory }} ({{ g.timeControlRaw || 'Standard' }})
                  </span>
                  <span class="text-zinc-300 font-semibold">
                    vs. {{ g.opponentName }} ({{ g.opponentElo }})
                  </span>
                  <span class="text-zinc-500 font-mono">{{ g.date }}</span>
                </div>
                <div class="text-[11px] text-zinc-400 font-mono">
                  Official {{ g.playerElo }} → <strong class="text-amber-300">IPR {{ g.intrinsicPerformanceElo }}</strong>
                  • ACPL <strong class="text-zinc-200">{{ g.acpl }}</strong>
                  • Top-1 <strong class="text-emerald-400">{{ g.top1AgreementPct }}%</strong>
                  • Move-Time CV <strong :class="g.moveTimeCV <= 0.38 ? 'text-rose-400' : 'text-zinc-200'">{{ g.moveTimeCV }}</strong>
                </div>
                <div v-if="g.flags && g.flags.length > 0" class="flex flex-wrap gap-1.5 pt-1">
                  <span
                    v-for="(fl, idx) in g.flags"
                    :key="idx"
                    class="px-2 py-0.5 rounded bg-rose-500/15 border border-rose-500/30 text-rose-300 text-[10px]"
                  >
                    {{ fl }}
                  </span>
                </div>
              </div>

              <div class="flex items-center gap-2 shrink-0">
                <span
                  class="px-2.5 py-1 rounded-lg border font-mono font-bold text-xs"
                  :class="scoreBadgeClass(g.gameIssueScore)"
                >
                  Score {{ g.gameIssueScore }}
                </span>
                <button
                  type="button"
                  data-testid="inspect-flagged-game-btn"
                  class="px-3 py-1.5 rounded-lg bg-zinc-800 hover:bg-zinc-700 text-white text-xs font-medium flex items-center gap-1 cursor-pointer"
                  @click="emit('select-game', g.gameId)"
                >
                  <span>Open on Board</span>
                  <ExternalLink class="w-3.5 h-3.5" />
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Footer -->
      <div class="flex items-center justify-between px-5 py-3 bg-zinc-950 border-t border-zinc-800 text-[11px] text-zinc-400 shrink-0">
        <span>
          Statistical indicators are objective mathematical correlations (ACPL, IPR &amp; Move-Time CV) for training &amp; fair-play review.
        </span>
        <button
          type="button"
          class="px-4 py-1.5 rounded-xl bg-zinc-800 hover:bg-zinc-700 text-zinc-200 text-xs font-medium cursor-pointer"
          @click="emit('close')"
        >
          Close
        </button>
      </div>
    </div>
  </div>
</template>
