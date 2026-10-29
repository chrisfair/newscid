<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import {
  X,
  Crosshair,
  ShieldAlert,
  Download,
  BookOpen,
  Sword,
  Copy,
  Check,
  ChevronRight,
  TrendingDown,
  Sparkles,
  RefreshCw,
  Eye,
  Sliders,
  AlertCircle
} from '@lucide/vue'
import Chessboard from './Chessboard.vue'
import {
  GetPersonas,
  GetDatabasePlayers,
  EvaluatePersonaWeaknesses,
  EvaluateOpponentWeaknesses,
  ExportAntiPersonaPolyglotBook,
  ExportAntiPersonaPGN,
  SaveBookFileDialog,
  SavePGNFileDialog
} from '../../wailsjs/go/main/App'
import type { persona } from '../../wailsjs/go/models'

const props = defineProps<{
  initialPersonaId?: string
  initialPlayerName?: string
  boardTheme?: { light: string; dark: string; highlight: string }
  pieceSet?: string
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'sparBot', personaId: string, playerName: string, targetColor: string): void
  (e: 'trainRepertoire', pgn: string, title: string): void
}>()

// State
const personasList = ref<persona.Persona[]>([])
const selectedPersonaId = ref<string>(props.initialPersonaId || '')
const customPlayerName = ref<string>(props.initialPlayerName || '')
const isCustomMode = ref<boolean>(!props.initialPersonaId && !!props.initialPlayerName)
const targetColor = ref<'white' | 'black' | 'both'>('white')
const maxDepth = ref<number>(14)
const minGames = ref<number>(1)

const loading = ref<boolean>(false)
const error = ref<string>('')
const report = ref<persona.AntiPersonaReport | null>(null)
const selectedVulnIndex = ref<number>(0)
const copiedPGN = ref<boolean>(false)
const exportSuccessMessage = ref<string>('')

onMounted(async () => {
  await loadPersonas()
  if (selectedPersonaId.value || customPlayerName.value) {
    generateAntiBook()
  }
})

async function loadPersonas() {
  try {
    const list = await GetPersonas()
    personasList.value = list || []
    if (!selectedPersonaId.value && !customPlayerName.value && personasList.value.length > 0) {
      selectedPersonaId.value = personasList.value[0].id
    }
  } catch (err) {
    console.error('Failed to load personas:', err)
  }
}

async function generateAntiBook() {
  error.value = ''
  exportSuccessMessage.value = ''
  loading.value = true
  report.value = null
  selectedVulnIndex.value = 0

  try {
    let res: persona.AntiPersonaReport
    if (isCustomMode.value) {
      if (!customPlayerName.value.trim()) {
        error.value = 'Please specify a player name.'
        loading.value = false
        return
      }
      res = await EvaluateOpponentWeaknesses(
        customPlayerName.value.trim(),
        targetColor.value,
        maxDepth.value,
        minGames.value
      )
    } else {
      if (!selectedPersonaId.value) {
        error.value = 'Please select a persona.'
        loading.value = false
        return
      }
      res = await EvaluatePersonaWeaknesses(
        selectedPersonaId.value,
        targetColor.value,
        maxDepth.value,
        minGames.value
      )
    }
    report.value = res
  } catch (err: any) {
    error.value = String(err?.message || err)
  } finally {
    loading.value = false
  }
}

const currentVuln = computed(() => {
  if (!report.value || !report.value.vulnerabilities || report.value.vulnerabilities.length === 0) {
    return null
  }
  return report.value.vulnerabilities[selectedVulnIndex.value] || report.value.vulnerabilities[0]
})

const currentBoardFEN = computed(() => {
  if (currentVuln.value) {
    return currentVuln.value.fen
  }
  return 'rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1'
})

const currentMoveArrow = computed(() => {
  if (!currentVuln.value || !currentVuln.value.moveUci || currentVuln.value.moveUci.length < 4) {
    return []
  }
  const uci = currentVuln.value.moveUci
  return [
    {
      from: uci.substring(0, 2),
      to: uci.substring(2, 4),
      color: '#10b981', // Emerald green arrow for our refutation move
      style: 'solid' as const
    }
  ]
})

async function handleSavePolyglot() {
  if (!report.value) return
  try {
    const slug = (report.value.displayName || 'opponent').replace(/\s+/g, '_').replace(/,/g, '')
    const defaultName = `Anti_${slug}.bin`
    const path = await SaveBookFileDialog(defaultName)
    if (!path) return

    const savedPath = await ExportAntiPersonaPolyglotBook(report.value, path)
    exportSuccessMessage.value = `Opening Book successfully saved to: ${savedPath}`
    setTimeout(() => {
      exportSuccessMessage.value = ''
    }, 6000)
  } catch (err: any) {
    error.value = `Failed to export book: ${err?.message || err}`
  }
}

async function handleSavePGN() {
  if (!report.value) return
  try {
    const slug = (report.value.displayName || 'opponent').replace(/\s+/g, '_').replace(/,/g, '')
    const defaultName = `Anti_${slug}.pgn`
    const path = await SavePGNFileDialog(defaultName)
    if (!path) return

    const savedPath = await ExportAntiPersonaPGN(report.value, path)
    exportSuccessMessage.value = `PGN Repertoire successfully saved to: ${savedPath}`
    setTimeout(() => {
      exportSuccessMessage.value = ''
    }, 6000)
  } catch (err: any) {
    error.value = `Failed to export PGN: ${err?.message || err}`
  }
}

function handleCopyPGN() {
  if (!report.value || !report.value.pgnRepertoire) return
  navigator.clipboard.writeText(report.value.pgnRepertoire)
  copiedPGN.value = true
  setTimeout(() => {
    copiedPGN.value = false
  }, 2500)
}

function handleTrainRepertoire() {
  if (!report.value || !report.value.pgnRepertoire) return
  const title = `Anti-${report.value.displayName} Counter Repertoire`
  emit('trainRepertoire', report.value.pgnRepertoire, title)
}

function handleSparBot() {
  if (!report.value) return
  emit('sparBot', selectedPersonaId.value, report.value.displayName, targetColor.value)
}
</script>

<template>
  <div
    class="fixed inset-0 bg-black/80 backdrop-blur-xs flex items-center justify-center z-50 p-4 animate-fade-in select-none text-zinc-200"
    @click.self="emit('close')"
  >
    <div
      class="bg-zinc-900 border border-zinc-700/80 w-full max-w-5xl rounded-2xl p-5 shadow-2xl flex flex-col gap-4 max-h-[90vh] overflow-hidden"
    >
      <!-- Header -->
      <div class="flex items-center justify-between border-b border-zinc-800 pb-3 shrink-0">
        <div class="flex items-center gap-3">
          <div
            class="w-10 h-10 rounded-xl bg-rose-500/20 border border-rose-500/40 flex items-center justify-center text-rose-400 shadow-inner"
          >
            <Crosshair class="w-5 h-5" />
          </div>
          <div>
            <div class="flex items-center gap-2">
              <h3 class="font-bold text-base text-zinc-100">Opponent Opening Exploiter</h3>
              <span class="text-[10px] font-semibold uppercase tracking-wider bg-rose-500/20 text-rose-300 px-2 py-0.5 rounded-full border border-rose-500/30">
                Custom Book Generator
              </span>
            </div>
            <p class="text-xs text-zinc-400">
              Evaluates statistical vulnerabilities across opponent games and compiles a Polyglot counter-book
            </p>
          </div>
        </div>
        <button
          class="p-2 rounded-xl text-zinc-400 hover:text-zinc-200 hover:bg-zinc-800 transition-colors cursor-pointer"
          @click="emit('close')"
        >
          <X class="w-5 h-5" />
        </button>
      </div>

      <!-- Controls Configuration Bar -->
      <div class="bg-zinc-950 p-3 rounded-xl border border-zinc-800 flex flex-wrap items-center justify-between gap-3 shrink-0 text-xs">
        <!-- Target Identity Selector -->
        <div class="flex items-center gap-2">
          <div class="flex bg-zinc-900 p-0.5 rounded-lg border border-zinc-800">
            <button
              :class="[
                'px-2.5 py-1 rounded-md text-xs font-medium cursor-pointer transition-colors',
                !isCustomMode ? 'bg-rose-600 text-white shadow-xs' : 'text-zinc-400 hover:text-zinc-200'
              ]"
              @click="isCustomMode = false"
            >
              Persona
            </button>
            <button
              :class="[
                'px-2.5 py-1 rounded-md text-xs font-medium cursor-pointer transition-colors',
                isCustomMode ? 'bg-rose-600 text-white shadow-xs' : 'text-zinc-400 hover:text-zinc-200'
              ]"
              @click="isCustomMode = true"
            >
              Player Name
            </button>
          </div>

          <div v-if="!isCustomMode" class="min-w-[180px]">
            <select
              v-model="selectedPersonaId"
              class="w-full bg-zinc-900 border border-zinc-700 rounded-lg px-2.5 py-1.5 text-xs text-zinc-200 focus:outline-hidden focus:border-rose-500"
            >
              <option v-for="p in personasList" :key="p.id" :value="p.id">
                {{ p.displayName }} ({{ p.identifiers.length }} aliases)
              </option>
            </select>
          </div>

          <div v-else class="min-w-[200px]">
            <input
              v-model="customPlayerName"
              type="text"
              placeholder="e.g. Kasparov, Garry"
              class="w-full bg-zinc-900 border border-zinc-700 rounded-lg px-2.5 py-1.5 text-xs text-zinc-200 focus:outline-hidden focus:border-rose-500 placeholder-zinc-500"
              @keydown.enter="generateAntiBook"
            />
          </div>
        </div>

        <!-- Target Color -->
        <div class="flex items-center gap-1.5">
          <span class="text-zinc-400">Target Color:</span>
          <div class="flex bg-zinc-900 p-0.5 rounded-lg border border-zinc-800">
            <button
              :class="[
                'px-2 py-1 rounded-md text-[11px] font-medium cursor-pointer transition-colors',
                targetColor === 'white' ? 'bg-zinc-700 text-white' : 'text-zinc-400 hover:text-zinc-200'
              ]"
              title="Opponent plays White, we prepare as Black"
              @click="targetColor = 'white'"
            >
              ⚪ White
            </button>
            <button
              :class="[
                'px-2 py-1 rounded-md text-[11px] font-medium cursor-pointer transition-colors',
                targetColor === 'black' ? 'bg-zinc-700 text-white' : 'text-zinc-400 hover:text-zinc-200'
              ]"
              title="Opponent plays Black, we prepare as White"
              @click="targetColor = 'black'"
            >
              ⚫ Black
            </button>
            <button
              :class="[
                'px-2 py-1 rounded-md text-[11px] font-medium cursor-pointer transition-colors',
                targetColor === 'both' ? 'bg-zinc-700 text-white' : 'text-zinc-400 hover:text-zinc-200'
              ]"
              title="Prepare against opponent as both White and Black"
              @click="targetColor = 'both'"
            >
              Dual
            </button>
          </div>
        </div>

        <!-- Depth & Min Games -->
        <div class="flex items-center gap-3">
          <div class="flex items-center gap-1.5">
            <span class="text-zinc-400">Depth:</span>
            <input
              v-model.number="maxDepth"
              type="number"
              min="6"
              max="24"
              class="w-12 bg-zinc-900 border border-zinc-700 rounded px-1.5 py-0.5 text-center text-xs"
            />
            <span class="text-[10px] text-zinc-500">plies</span>
          </div>

          <button
            :disabled="loading"
            class="px-4 py-1.5 rounded-lg bg-rose-600 hover:bg-rose-500 disabled:opacity-50 text-white font-medium text-xs flex items-center gap-1.5 transition-colors cursor-pointer shadow-md"
            @click="generateAntiBook"
          >
            <RefreshCw v-if="loading" class="w-3.5 h-3.5 animate-spin" />
            <Crosshair v-else class="w-3.5 h-3.5" />
            Evaluate & Synthesize
          </button>
        </div>
      </div>

      <!-- Success Notification Banner -->
      <div
        v-if="exportSuccessMessage"
        class="bg-emerald-950/40 border border-emerald-500/40 px-3 py-2 rounded-xl text-emerald-300 text-xs flex items-center gap-2 animate-fade-in"
      >
        <Check class="w-4 h-4 text-emerald-400 shrink-0" />
        <span>{{ exportSuccessMessage }}</span>
      </div>

      <!-- Error State -->
      <div v-if="error" class="p-3 bg-red-950/40 border border-red-800/60 rounded-xl text-red-300 text-xs flex items-center gap-2">
        <AlertCircle class="w-4 h-4 text-red-400 shrink-0" />
        <span>{{ error }}</span>
      </div>

      <!-- Loading State -->
      <div v-if="loading" class="flex-1 flex flex-col items-center justify-center py-16 gap-3 text-zinc-400">
        <div class="relative w-12 h-12 flex items-center justify-center">
          <div class="w-12 h-12 rounded-full border-2 border-rose-500/20 border-t-rose-500 animate-spin"></div>
          <Crosshair class="w-5 h-5 text-rose-400 absolute" />
        </div>
        <p class="text-sm font-medium text-zinc-300">Evaluating opening move trees and identifying statistical vulnerabilities...</p>
        <p class="text-xs text-zinc-500">Replaying persona games, calculating branch loss rates, and forging Polyglot refutations</p>
      </div>

      <!-- Main Results Display -->
      <div v-else-if="report" class="flex-1 flex flex-col gap-3 min-h-0 overflow-hidden">
        <!-- Overview Dossier Cards -->
        <div class="grid grid-cols-4 gap-2.5 shrink-0 text-xs">
          <div class="bg-zinc-950 p-2.5 rounded-xl border border-zinc-800 flex flex-col justify-between">
            <span class="text-zinc-400 text-[11px]">Analyzed Games</span>
            <div class="text-base font-bold text-zinc-100 mt-0.5">
              {{ report.totalGamesAnalyzed }}
              <span class="text-[10px] text-zinc-500 font-normal">games as {{ report.targetColor }}</span>
            </div>
          </div>

          <div class="bg-zinc-950 p-2.5 rounded-xl border border-zinc-800 flex flex-col justify-between">
            <span class="text-zinc-400 text-[11px]">Opponent Record</span>
            <div class="text-xs font-semibold text-zinc-200 mt-0.5">
              <span class="text-emerald-400">{{ report.personaRecord.wins }}W</span> /
              <span class="text-zinc-400">{{ report.personaRecord.draws }}D</span> /
              <span class="text-rose-400">{{ report.personaRecord.losses }}L</span>
              <span class="text-[10px] text-zinc-500 ml-1">({{ report.personaRecord.winRate }}%)</span>
            </div>
          </div>

          <div class="bg-zinc-950 p-2.5 rounded-xl border border-zinc-800 flex flex-col justify-between">
            <span class="text-zinc-400 text-[11px]">Critical Weaknesses</span>
            <div class="text-base font-bold text-rose-400 mt-0.5 flex items-center gap-1.5">
              <ShieldAlert class="w-4 h-4" />
              <span>{{ report.vulnerabilities.length }} Lines</span>
            </div>
          </div>

          <div class="bg-zinc-950 p-2.5 rounded-xl border border-zinc-800 flex flex-col justify-between">
            <span class="text-zinc-400 text-[11px]">Counter Book</span>
            <div class="text-base font-bold text-cyan-400 mt-0.5 flex items-center gap-1.5">
              <BookOpen class="w-4 h-4" />
              <span>{{ report.polyglotEntryCount }} Entries</span>
            </div>
          </div>
        </div>

        <!-- Split View: Vulnerabilities List (Left) + Interactive Board & Advice (Right) -->
        <div class="flex-1 grid grid-cols-12 gap-3 min-h-0 overflow-hidden">
          <!-- Left Column: Vulnerability Lines List -->
          <div class="col-span-6 bg-zinc-950/70 rounded-xl border border-zinc-800 p-2 flex flex-col min-h-0 overflow-y-auto">
            <div class="px-2 py-1 text-[11px] font-semibold uppercase tracking-wider text-zinc-400 flex items-center justify-between border-b border-zinc-800/80 mb-2">
              <span>Identified Achilles' Heel Openings</span>
              <span>Exploit Score</span>
            </div>

            <div v-if="report.vulnerabilities.length === 0" class="py-8 text-center text-zinc-500 text-xs">
              No severe opening vulnerabilities detected under current thresholds.
            </div>

            <div v-else class="flex flex-col gap-1.5">
              <div
                v-for="(vuln, idx) in report.vulnerabilities"
                :key="idx"
                :class="[
                  'p-2.5 rounded-xl border transition-all cursor-pointer flex flex-col gap-1.5',
                  selectedVulnIndex === idx
                    ? 'bg-rose-950/30 border-rose-500/60 shadow-xs'
                    : 'bg-zinc-900/60 border-zinc-800/80 hover:bg-zinc-800/60 hover:border-zinc-700'
                ]"
                @click="selectedVulnIndex = idx"
              >
                <!-- Line Header -->
                <div class="flex items-center justify-between">
                  <div class="flex items-center gap-2 min-w-0">
                    <span class="font-mono font-bold text-[11px] px-1.5 py-0.5 rounded bg-zinc-800 text-amber-300 shrink-0">
                      {{ vuln.eco }}
                    </span>
                    <span class="font-semibold text-xs text-zinc-200 truncate">
                      {{ vuln.openingName }}
                    </span>
                  </div>

                  <div class="flex items-center gap-1.5 shrink-0">
                    <span class="text-[11px] font-mono font-bold px-1.5 py-0.5 rounded bg-rose-500/20 text-rose-300 border border-rose-500/30">
                      {{ vuln.exploitationScore }}
                    </span>
                  </div>
                </div>

                <!-- Refutation Move & Stats -->
                <div class="flex items-center justify-between text-[11px]">
                  <div class="flex items-center gap-1.5">
                    <span class="text-zinc-400">Weapon:</span>
                    <span class="font-mono font-bold text-emerald-400 bg-emerald-950/40 px-1.5 py-0.5 rounded border border-emerald-500/30">
                      {{ vuln.moveSan }}!
                    </span>
                  </div>

                  <div class="text-zinc-400">
                    Opponent:
                    <span class="text-rose-400 font-semibold">{{ vuln.personaRecord.winRate }}% win rate</span>
                    <span class="text-zinc-500 text-[10px]"> ({{ vuln.personaRecord.total }} games, {{ vuln.personaRecord.losses }}L)</span>
                  </div>
                </div>

                <!-- Moves Sequence Preview -->
                <div class="text-[10px] font-mono text-zinc-400 truncate bg-zinc-950/80 px-2 py-1 rounded border border-zinc-900">
                  {{ vuln.moveHistorySan.join(' ') }}
                </div>
              </div>
            </div>
          </div>

          <!-- Right Column: Board & Tactical Insight -->
          <div class="col-span-6 flex flex-col gap-3 min-h-0 overflow-y-auto">
            <!-- Board Preview Card -->
            <div class="bg-zinc-950 p-3 rounded-xl border border-zinc-800 flex flex-col items-center justify-center">
              <div class="w-full flex items-center justify-between text-xs text-zinc-400 mb-2 px-1">
                <span class="font-semibold text-zinc-200 flex items-center gap-1.5">
                  <Eye class="w-3.5 h-3.5 text-emerald-400" />
                  Target Position After <span class="text-emerald-300 font-mono font-bold">{{ currentVuln?.moveSan }}!</span>
                </span>
                <span class="font-mono text-[10px] text-zinc-500">Ply {{ currentVuln?.ply }}</span>
              </div>

              <!-- Interactive Chessboard -->
              <div class="w-[280px] h-[280px] flex items-center justify-center">
                <Chessboard
                  :fen="currentBoardFEN"
                  :board-theme="boardTheme"
                  :piece-set="pieceSet"
                  :board-size="280"
                  :flipped="targetColor === 'white'"
                  :arrows="currentMoveArrow"
                />
              </div>
            </div>

            <!-- Tactical Advice Card -->
            <div v-if="currentVuln" class="bg-rose-950/20 border border-rose-500/30 rounded-xl p-3 flex flex-col gap-2 text-xs">
              <div class="flex items-center gap-1.5 font-bold text-rose-300 text-xs">
                <Sparkles class="w-4 h-4 text-rose-400" />
                Exploitation Tactical Insight:
              </div>
              <p class="text-zinc-300 text-[11px] leading-relaxed">
                {{ currentVuln.tacticalInsight }}
              </p>

              <!-- Suggested Opponent Replies -->
              <div v-if="currentVuln.suggestedReplies && currentVuln.suggestedReplies.length > 0" class="mt-1 pt-2 border-t border-rose-500/20 text-[11px]">
                <span class="text-zinc-400 font-medium">Opponent's frequent responses:</span>
                <div class="flex flex-wrap gap-1.5 mt-1">
                  <span
                    v-for="(rep, idx) in currentVuln.suggestedReplies"
                    :key="idx"
                    class="font-mono text-[10px] bg-zinc-900 border border-zinc-800 px-2 py-0.5 rounded text-zinc-300"
                  >
                    {{ rep }}
                  </span>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Footer Actions Bar -->
        <div class="border-t border-zinc-800 pt-3 flex items-center justify-between shrink-0 text-xs">
          <div class="flex items-center gap-2">
            <button
              class="px-3 py-1.5 rounded-lg bg-zinc-800 hover:bg-zinc-700 text-zinc-200 font-medium flex items-center gap-1.5 transition-colors cursor-pointer"
              @click="handleCopyPGN"
            >
              <Check v-if="copiedPGN" class="w-3.5 h-3.5 text-emerald-400" />
              <Copy v-else class="w-3.5 h-3.5" />
              {{ copiedPGN ? 'PGN Copied!' : 'Copy Repertoire PGN' }}
            </button>

            <button
              class="px-3 py-1.5 rounded-lg bg-zinc-800 hover:bg-zinc-700 text-zinc-200 font-medium flex items-center gap-1.5 transition-colors cursor-pointer"
              @click="handleSavePGN"
            >
              <Download class="w-3.5 h-3.5" />
              Save .pgn
            </button>
          </div>

          <div class="flex items-center gap-2">
            <button
              class="px-3.5 py-1.5 rounded-lg bg-cyan-600 hover:bg-cyan-500 text-white font-medium flex items-center gap-1.5 transition-colors cursor-pointer shadow-md"
              @click="handleTrainRepertoire"
            >
              <BookOpen class="w-3.5 h-3.5" />
              Train Repertoire
            </button>

            <button
              class="px-3.5 py-1.5 rounded-lg bg-rose-600 hover:bg-rose-500 text-white font-medium flex items-center gap-1.5 transition-colors cursor-pointer shadow-md"
              @click="handleSavePolyglot"
            >
              <Download class="w-3.5 h-3.5" />
              Save Polyglot (.bin) Book
            </button>

            <button
              class="px-3.5 py-1.5 rounded-lg bg-amber-600 hover:bg-amber-500 text-white font-medium flex items-center gap-1.5 transition-colors cursor-pointer shadow-md"
              @click="handleSparBot"
            >
              <Sword class="w-3.5 h-3.5" />
              Spar Against Persona
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
