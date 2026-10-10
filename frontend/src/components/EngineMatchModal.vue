<script setup lang="ts">
import { ref, reactive, computed, onMounted, onUnmounted } from 'vue'
import {
  Swords,
  X,
  Play,
  Square,
  Trophy,
  Cpu,
  Clock,
  Settings,
  Layers,
  CheckCircle2,
  AlertCircle,
  ExternalLink,
  ChevronRight
} from '@lucide/vue'
import {
  StartEngineMatch,
  StopEngineMatch,
  IsEngineMatchRunning
} from '../../wailsjs/go/main/App'
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'
import { tournament } from '../../wailsjs/go/models'
import { EngineOption } from './EnginePanel.vue'

const props = withDefaults(defineProps<{
  discoveredEngines?: EngineOption[]
  availableEngines?: string[]
  dbName?: string
}>(), {
  discoveredEngines: () => [],
  availableEngines: () => [],
  dbName: 'Active Database'
})

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'matchFinished', summary: tournament.EngineMatchSummary): void
}>()

const form = reactive({
  engine1Name: '',
  engine1Path: '',
  engine2Name: '',
  engine2Path: '',
  rounds: 4,
  timeControlMs: 500,
  maxPlies: 150,
  alternateColors: true,
  adjudicateEvalCp: 1000,
  autoSaveToDb: true
})

export interface EngineMatchProgress {
  currentRound: number
  totalRounds: number
  currentPly: number
  lastMoveUci: string
  lastMoveSan: string
  currentFen: string
  whiteEngine: string
  blackEngine: string
  engine1Score: number
  engine2Score: number
  lastEval?: {
    depth?: number
    scoreCp: number
    mate?: number
    pv?: string[]
  }
}

const isRunning = ref(false)
const errorMessage = ref('')
const liveProgress = ref<EngineMatchProgress | null>(null)
const completedGames = ref<tournament.EngineGameResult[]>([])
const matchSummary = ref<tournament.EngineMatchSummary | null>(null)

const score1 = computed(() => {
  if (matchSummary.value) return matchSummary.value.engine1Score
  return completedGames.value.reduce((acc, g) => {
    const isE1White = (g.round % 2 !== 0) || !form.alternateColors
    if (g.result === '1-0') return acc + (isE1White ? 1 : 0)
    if (g.result === '0-1') return acc + (isE1White ? 0 : 1)
    if (g.result === '1/2-1/2') return acc + 0.5
    return acc
  }, 0)
})

const score2 = computed(() => {
  if (matchSummary.value) return matchSummary.value.engine2Score
  return completedGames.value.reduce((acc, g) => {
    const isE1White = (g.round % 2 !== 0) || !form.alternateColors
    if (g.result === '1-0') return acc + (isE1White ? 0 : 1)
    if (g.result === '0-1') return acc + (isE1White ? 1 : 0)
    if (g.result === '1/2-1/2') return acc + 0.5
    return acc
  }, 0)
})

onMounted(async () => {
  // Pre-populate engines
  if (props.discoveredEngines.length >= 2) {
    form.engine1Name = props.discoveredEngines[0].name
    form.engine1Path = props.discoveredEngines[0].cmd
    form.engine2Name = props.discoveredEngines[1].name
    form.engine2Path = props.discoveredEngines[1].cmd
  } else if (props.discoveredEngines.length === 1) {
    form.engine1Name = props.discoveredEngines[0].name
    form.engine1Path = props.discoveredEngines[0].cmd
    form.engine2Name = 'Stockfish Alternative'
    form.engine2Path = props.discoveredEngines[0].cmd
  } else {
    form.engine1Name = 'Engine Alpha'
    form.engine2Name = 'Engine Beta'
  }

  isRunning.value = await IsEngineMatchRunning()

  EventsOn('enginematch:progress', (data: any) => {
    liveProgress.value = data
  })

  EventsOn('enginematch:gameend', (data: any) => {
    if (data && data.game) {
      completedGames.value.push(data.game)
    }
  })
})

onUnmounted(() => {
  EventsOff('enginematch:progress')
  EventsOff('enginematch:gameend')
})

function handleEngine1Select(path: string) {
  const found = props.discoveredEngines.find(e => e.cmd === path)
  if (found) {
    form.engine1Name = found.name
    form.engine1Path = found.cmd
  }
}

function handleEngine2Select(path: string) {
  const found = props.discoveredEngines.find(e => e.cmd === path)
  if (found) {
    form.engine2Name = found.name
    form.engine2Path = found.cmd
  }
}

async function handleStartMatch() {
  errorMessage.value = ''
  matchSummary.value = null
  completedGames.value = []
  liveProgress.value = null
  isRunning.value = true

  try {
    const cfg: any = {
      engine1: {
        name: form.engine1Name || 'Engine 1',
        path: form.engine1Path
      },
      engine2: {
        name: form.engine2Name || 'Engine 2',
        path: form.engine2Path
      },
      rounds: Number(form.rounds) || 2,
      timeControlMs: Number(form.timeControlMs) || 500,
      maxPlies: Number(form.maxPlies) || 150,
      alternateColors: Boolean(form.alternateColors),
      adjudicateEvalCp: Number(form.adjudicateEvalCp) || 0,
      autoSaveToDb: Boolean(form.autoSaveToDb)
    }

    const summary = await StartEngineMatch(cfg)
    matchSummary.value = summary
    emit('matchFinished', summary)
  } catch (err: any) {
    errorMessage.value = `Engine match failed: ${err}`
  } finally {
    isRunning.value = false
  }
}

async function handleStopMatch() {
  try {
    await StopEngineMatch()
    isRunning.value = false
  } catch (err: any) {
    console.error('Failed to stop engine match:', err)
  }
}
</script>

<template>
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/75 backdrop-blur-sm p-4 animate-fade-in">
    <div class="bg-zinc-900 border border-zinc-800 rounded-xl shadow-2xl w-full max-w-2xl flex flex-col max-h-[90vh] overflow-hidden">
      <!-- Modal Header -->
      <div class="px-5 py-3 border-b border-zinc-800 flex items-center justify-between bg-zinc-950/60">
        <div class="flex items-center gap-2.5">
          <div class="p-2 rounded-lg bg-amber-500/10 border border-amber-500/20 text-amber-400">
            <Swords class="w-5 h-5" />
          </div>
          <div>
            <h2 class="text-base font-semibold text-zinc-100 flex items-center gap-2">
              Automated Engine vs Engine Match
              <span class="text-xs px-2 py-0.5 rounded-full bg-zinc-800 text-zinc-300 font-mono">
                Tournament Mode
              </span>
            </h2>
            <p class="text-xs text-zinc-400">
              Run automated chess matches between two UCI engines with color alternation, adjudication, and database recording.
            </p>
          </div>
        </div>
        <button
          class="p-1.5 rounded-lg text-zinc-400 hover:text-zinc-200 hover:bg-zinc-800 transition-colors cursor-pointer"
          @click="emit('close')"
        >
          <X class="w-5 h-5" />
        </button>
      </div>

      <!-- Live Match Scoreboard Banner (When active or completed) -->
      <div v-if="isRunning || completedGames.length > 0" class="p-4 bg-zinc-950 border-b border-zinc-800 flex flex-col gap-3">
        <div class="flex items-center justify-between">
          <div class="flex-1 flex flex-col items-center">
            <span class="text-xs font-semibold text-zinc-200 truncate max-w-[180px]">{{ form.engine1Name || 'Engine 1' }}</span>
            <span class="text-2xl font-bold font-mono text-amber-400">{{ score1.toFixed(1) }}</span>
          </div>
          <div class="flex flex-col items-center px-4">
            <span class="text-[10px] uppercase font-bold tracking-wider text-zinc-500">
              {{ isRunning ? 'In Progress' : 'Match Complete' }}
            </span>
            <span class="text-xs font-mono text-zinc-400">
              Round {{ liveProgress ? liveProgress.currentRound : completedGames.length }} / {{ form.rounds }}
            </span>
          </div>
          <div class="flex-1 flex flex-col items-center">
            <span class="text-xs font-semibold text-zinc-200 truncate max-w-[180px]">{{ form.engine2Name || 'Engine 2' }}</span>
            <span class="text-2xl font-bold font-mono text-cyan-400">{{ score2.toFixed(1) }}</span>
          </div>
        </div>

        <!-- Live Telemetry / Evaluation -->
        <div v-if="liveProgress && isRunning" class="p-2.5 rounded-lg bg-zinc-900 border border-zinc-800 flex items-center justify-between text-xs">
          <div class="flex items-center gap-2">
            <Cpu class="w-4 h-4 text-emerald-400 animate-pulse" />
            <span class="text-zinc-300 font-medium">Ply {{ liveProgress.currentPly }}:</span>
            <span class="font-mono text-amber-300 font-semibold">{{ liveProgress.lastMoveSan }}</span>
          </div>
          <div v-if="liveProgress.lastEval" class="flex items-center gap-3 text-[11px] font-mono text-zinc-400">
            <span>Depth: <strong class="text-zinc-200">{{ liveProgress.lastEval.depth }}</strong></span>
            <span>Eval: <strong :class="liveProgress.lastEval.scoreCp >= 0 ? 'text-emerald-400' : 'text-rose-400'">
              {{ (liveProgress.lastEval.scoreCp / 100).toFixed(2) }}
            </strong></span>
          </div>
        </div>
      </div>

      <!-- Config Form -->
      <div class="p-5 flex-1 overflow-y-auto flex flex-col gap-4 text-xs">
        <!-- Engine 1 & Engine 2 Selectors -->
        <div class="grid grid-cols-2 gap-4">
          <!-- Engine 1 Card -->
          <div class="p-3 rounded-lg bg-zinc-950 border border-zinc-800 flex flex-col gap-2">
            <div class="flex items-center justify-between">
              <span class="font-semibold text-amber-400 flex items-center gap-1.5">
                <Cpu class="w-3.5 h-3.5" />
                Engine 1
              </span>
              <span class="text-[10px] text-zinc-500 font-mono">Plays White First</span>
            </div>
            <div class="flex flex-col gap-1">
              <label class="text-[11px] text-zinc-400">Name</label>
              <input
                v-model="form.engine1Name"
                type="text"
                placeholder="e.g. Stockfish 16"
                class="w-full bg-zinc-900 border border-zinc-700 rounded px-2.5 py-1 text-zinc-100 font-medium focus:outline-none focus:border-amber-500"
                :disabled="isRunning"
              />
            </div>
            <div class="flex flex-col gap-1">
              <label class="text-[11px] text-zinc-400">Installed Engine / Executable</label>
              <select
                class="w-full bg-zinc-900 border border-zinc-700 rounded px-2.5 py-1 text-zinc-200 focus:outline-none focus:border-amber-500"
                :disabled="isRunning"
                @change="handleEngine1Select(($event.target as HTMLSelectElement).value)"
              >
                <option value="">-- Choose Installed Engine --</option>
                <option
                  v-for="eng in discoveredEngines"
                  :key="eng.cmd"
                  :value="eng.cmd"
                  :selected="form.engine1Path === eng.cmd"
                >
                  {{ eng.name }} ({{ eng.cmd }})
                </option>
              </select>
            </div>
          </div>

          <!-- Engine 2 Card -->
          <div class="p-3 rounded-lg bg-zinc-950 border border-zinc-800 flex flex-col gap-2">
            <div class="flex items-center justify-between">
              <span class="font-semibold text-cyan-400 flex items-center gap-1.5">
                <Cpu class="w-3.5 h-3.5" />
                Engine 2
              </span>
              <span class="text-[10px] text-zinc-500 font-mono">Plays Black First</span>
            </div>
            <div class="flex flex-col gap-1">
              <label class="text-[11px] text-zinc-400">Name</label>
              <input
                v-model="form.engine2Name"
                type="text"
                placeholder="e.g. LCZero / Alternate"
                class="w-full bg-zinc-900 border border-zinc-700 rounded px-2.5 py-1 text-zinc-100 font-medium focus:outline-none focus:border-cyan-500"
                :disabled="isRunning"
              />
            </div>
            <div class="flex flex-col gap-1">
              <label class="text-[11px] text-zinc-400">Installed Engine / Executable</label>
              <select
                class="w-full bg-zinc-900 border border-zinc-700 rounded px-2.5 py-1 text-zinc-200 focus:outline-none focus:border-cyan-500"
                :disabled="isRunning"
                @change="handleEngine2Select(($event.target as HTMLSelectElement).value)"
              >
                <option value="">-- Choose Installed Engine --</option>
                <option
                  v-for="eng in discoveredEngines"
                  :key="eng.cmd"
                  :value="eng.cmd"
                  :selected="form.engine2Path === eng.cmd"
                >
                  {{ eng.name }} ({{ eng.cmd }})
                </option>
              </select>
            </div>
          </div>
        </div>

        <!-- Match Options Grid -->
        <div class="grid grid-cols-3 gap-3">
          <div class="flex flex-col gap-1">
            <label class="text-[11px] text-zinc-300 font-medium">Number of Rounds</label>
            <input
              v-model.number="form.rounds"
              type="number"
              min="1"
              max="100"
              class="w-full bg-zinc-950 border border-zinc-700 rounded px-2.5 py-1.5 text-zinc-100 font-mono"
              :disabled="isRunning"
            />
          </div>
          <div class="flex flex-col gap-1">
            <label class="text-[11px] text-zinc-300 font-medium">Time Per Move (ms)</label>
            <input
              v-model.number="form.timeControlMs"
              type="number"
              min="50"
              step="50"
              class="w-full bg-zinc-950 border border-zinc-700 rounded px-2.5 py-1.5 text-zinc-100 font-mono"
              :disabled="isRunning"
            />
          </div>
          <div class="flex flex-col gap-1">
            <label class="text-[11px] text-zinc-300 font-medium">Max Plies / Game</label>
            <input
              v-model.number="form.maxPlies"
              type="number"
              min="20"
              max="500"
              class="w-full bg-zinc-950 border border-zinc-700 rounded px-2.5 py-1.5 text-zinc-100 font-mono"
              :disabled="isRunning"
            />
          </div>
        </div>

        <!-- Checkbox Options -->
        <div class="flex flex-col gap-2 pt-1">
          <label class="flex items-center gap-2 text-zinc-300 cursor-pointer">
            <input
              v-model="form.alternateColors"
              type="checkbox"
              class="rounded bg-zinc-950 border-zinc-700 text-amber-500 focus:ring-0"
              :disabled="isRunning"
            />
            <span>Alternate colors every round (Round 1: White, Round 2: Black...)</span>
          </label>
          <label class="flex items-center gap-2 text-zinc-300 cursor-pointer">
            <input
              v-model="form.autoSaveToDb"
              type="checkbox"
              class="rounded bg-zinc-950 border-zinc-700 text-amber-500 focus:ring-0"
              :disabled="isRunning"
            />
            <span>Auto-save all completed match games into active database ({{ dbName }})</span>
          </label>
        </div>

        <!-- Error Feedback -->
        <div v-if="errorMessage" class="p-2.5 rounded-lg bg-rose-950/60 border border-rose-700 text-rose-200 text-xs">
          {{ errorMessage }}
        </div>

        <!-- Completed Games Table -->
        <div v-if="completedGames.length > 0" class="flex flex-col gap-2 mt-2">
          <span class="font-semibold text-zinc-300 flex items-center justify-between">
            <span>Completed Match Games ({{ completedGames.length }})</span>
            <span class="text-[11px] text-zinc-500 font-normal">Recorded into tournament base</span>
          </span>
          <div class="max-h-36 overflow-y-auto border border-zinc-800 rounded-lg divide-y divide-zinc-800">
            <div
              v-for="g in completedGames"
              :key="g.round"
              class="px-3 py-1.5 flex items-center justify-between text-[11px]"
            >
              <div class="flex items-center gap-2">
                <span class="font-mono text-zinc-500">R{{ g.round }}</span>
                <span class="font-medium text-zinc-300">{{ g.whiteEngine }}</span>
                <span class="text-zinc-500">vs</span>
                <span class="font-medium text-zinc-300">{{ g.blackEngine }}</span>
              </div>
              <div class="flex items-center gap-3">
                <span class="font-mono font-bold" :class="g.result === '1-0' ? 'text-amber-400' : g.result === '0-1' ? 'text-cyan-400' : 'text-zinc-400'">
                  {{ g.result }}
                </span>
                <span class="text-zinc-500 text-[10px]">({{ g.termination }}, {{ g.plies }} plies)</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Modal Footer -->
      <div class="px-5 py-3 border-t border-zinc-800 flex items-center justify-between bg-zinc-950/60">
        <span class="text-xs text-zinc-500">
          Scid vs. PC Tournament Parity
        </span>
        <div class="flex items-center gap-3">
          <button
            type="button"
            class="px-4 py-2 rounded-lg bg-zinc-800 hover:bg-zinc-700 text-zinc-300 font-medium text-xs transition-colors cursor-pointer"
            @click="emit('close')"
          >
            Close
          </button>
          <button
            v-if="isRunning"
            type="button"
            class="px-5 py-2 rounded-lg bg-rose-600 hover:bg-rose-500 text-white font-medium text-xs flex items-center gap-1.5 transition-colors cursor-pointer"
            @click="handleStopMatch"
          >
            <Square class="w-3.5 h-3.5" />
            <span>Stop Match</span>
          </button>
          <button
            v-else
            type="button"
            class="px-5 py-2 rounded-lg bg-amber-600 hover:bg-amber-500 text-white font-medium text-xs flex items-center gap-1.5 transition-colors cursor-pointer disabled:opacity-50"
            :disabled="isRunning"
            @click="handleStartMatch"
          >
            <Play class="w-3.5 h-3.5 fill-current" />
            <span>Start Tournament Match</span>
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
