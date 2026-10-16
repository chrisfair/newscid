<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import {
  X,
  Scissors,
  CheckCircle2,
  AlertTriangle,
  BookOpen,
  Sparkles,
  Sliders,
  TrendingDown,
  Layers,
  ChevronRight,
  ShieldCheck,
  Zap,
  Save,
  Trash2
} from '@lucide/vue'
import {
  ListRepertoires,
  ShortenRepertoire
} from '../../wailsjs/go/main/App'
import { repertoire } from '../../wailsjs/go/models'

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'repertoireSaved', repId: string): void
}>()

// State
const repertoires = ref<repertoire.RepertoireSummary[]>([])
const selectedRepId = ref<string>('')
const isLoadingRepertoires = ref<boolean>(false)
const isPruning = ref<boolean>(false)
const errorMsg = ref<string>('')
const successMsg = ref<string>('')

// Pruning Options
const targetCoverage = ref<number>(80)
const minFrequency = ref<number>(3.0)
const maxDepth = ref<number>(12)
const preserveAnnotated = ref<boolean>(true)
const saveAsNew = ref<boolean>(true)

// Result State
const shortenerResult = ref<repertoire.ShortenerResult | null>(null)
const activeView = ref<'pruned' | 'mainlines'>('pruned')

const selectedRepSummary = computed(() => {
  return repertoires.value.find(r => r.id === selectedRepId.value) || null
})

async function loadRepertoires() {
  isLoadingRepertoires.value = true
  errorMsg.value = ''
  try {
    const list = await ListRepertoires()
    repertoires.value = list
    if (list.length > 0 && !selectedRepId.value) {
      selectedRepId.value = list[0].id
    }
  } catch (err: unknown) {
    errorMsg.value = err instanceof Error ? err.message : String(err)
  } finally {
    isLoadingRepertoires.value = false
  }
}

async function handleRunPrune() {
  if (!selectedRepId.value) return
  isPruning.value = true
  errorMsg.value = ''
  successMsg.value = ''

  try {
    const opts = new repertoire.ShortenerOptions({
      repertoireId: selectedRepId.value,
      targetCoveragePct: targetCoverage.value,
      minFrequencyPct: minFrequency.value,
      maxDepth: maxDepth.value,
      preserveAnnotated: preserveAnnotated.value,
      saveAsNew: saveAsNew.value
    })

    const res = await ShortenRepertoire(opts)
    shortenerResult.value = res
    if (saveAsNew.value) {
      successMsg.value = `Successfully generated optimized repertoire: ${res.prunedRepertoireId}`
      emit('repertoireSaved', res.prunedRepertoireId)
    } else {
      successMsg.value = `Successfully updated repertoire ${res.originalId}!`
      emit('repertoireSaved', res.originalId)
    }
  } catch (err: unknown) {
    errorMsg.value = err instanceof Error ? err.message : String(err)
  } finally {
    isPruning.value = false
  }
}

onMounted(() => {
  loadRepertoires()
})
</script>

<template>
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/75 backdrop-blur-sm p-4 select-none">
    <div
      class="bg-theme-panel border border-theme-border rounded-xl shadow-2xl flex flex-col w-full max-w-4xl overflow-hidden animate-in fade-in zoom-in-95 duration-150"
      style="height: 84vh; max-height: 800px;"
    >
      <!-- Header -->
      <div class="px-6 py-3.5 border-b border-theme-border flex items-center justify-between bg-theme-panel-header">
        <div class="flex items-center gap-3">
          <div class="p-2 bg-rose-500/10 text-rose-400 rounded-lg border border-rose-500/20">
            <Scissors class="w-5 h-5 text-rose-400" />
          </div>
          <div>
            <div class="flex items-center gap-2">
              <h2 class="text-base font-bold text-theme-text-primary tracking-wide">Repertoire Shortener</h2>
              <span class="px-2 py-0.5 text-xs font-semibold rounded bg-rose-500/20 text-rose-300 border border-rose-500/30">
                Pareto 80/20 Pruner
              </span>
            </div>
            <p class="text-xs text-theme-text-muted mt-0.5">
              Prune obscure sidelines and low-frequency branches to prioritize 80% of over-the-board positions.
            </p>
          </div>
        </div>

        <button
          class="text-theme-text-muted hover:text-theme-text-primary p-1.5 rounded-lg hover:bg-theme-hover transition-colors"
          @click="emit('close')"
        >
          <X class="w-5 h-5" />
        </button>
      </div>

      <!-- Main Layout: Sidebar controls + Results view -->
      <div class="flex-1 flex flex-col md:flex-row overflow-hidden">
        <!-- Controls Column -->
        <div class="w-full md:w-80 border-b md:border-b-0 md:border-r border-theme-border bg-theme-panel-header/30 p-5 flex flex-col justify-between overflow-y-auto">
          <div class="space-y-4">
            <!-- Repertoire Selector -->
            <div>
              <label class="block text-xs font-bold text-theme-text-primary mb-1.5 flex items-center gap-1.5">
                <BookOpen class="w-3.5 h-3.5 text-amber-400" />
                Select Repertoire
              </label>
              <select
                v-model="selectedRepId"
                class="w-full px-3 py-2 bg-theme-panel border border-theme-border rounded-lg text-xs text-theme-text-primary focus:outline-none focus:border-amber-500"
              >
                <option v-for="r in repertoires" :key="r.id" :value="r.id">
                  {{ r.name }} ({{ r.numMoves }} moves)
                </option>
              </select>
            </div>

            <!-- Target Coverage Slider (Pareto Principle) -->
            <div>
              <div class="flex items-center justify-between text-xs mb-1">
                <span class="font-bold text-theme-text-primary">Target Coverage:</span>
                <span class="font-mono font-bold text-rose-400">{{ targetCoverage }}%</span>
              </div>
              <input
                v-model.number="targetCoverage"
                type="range"
                min="60"
                max="95"
                step="5"
                class="w-full accent-rose-500 cursor-pointer h-1.5 bg-theme-border rounded-lg"
              />
              <span class="text-[11px] text-theme-text-muted mt-1 block">
                Standard Pareto 80/20 cuts the bottom 20% rarest opponent deviations.
              </span>
            </div>

            <!-- Min Frequency Cutoff -->
            <div>
              <div class="flex items-center justify-between text-xs mb-1">
                <span class="font-bold text-theme-text-primary">Min Move Frequency:</span>
                <span class="font-mono font-bold text-amber-400">{{ minFrequency }}%</span>
              </div>
              <input
                v-model.number="minFrequency"
                type="range"
                min="1.0"
                max="10.0"
                step="0.5"
                class="w-full accent-amber-500 cursor-pointer h-1.5 bg-theme-border rounded-lg"
              />
              <span class="text-[11px] text-theme-text-muted mt-1 block">
                Prunes lines occurring in less than {{ minFrequency }}% of tournament games.
              </span>
            </div>

            <!-- Max Depth -->
            <div>
              <div class="flex items-center justify-between text-xs mb-1">
                <span class="font-bold text-theme-text-primary">Max Plies Depth:</span>
                <span class="font-mono font-bold text-indigo-400">{{ maxDepth }} plies</span>
              </div>
              <input
                v-model.number="maxDepth"
                type="range"
                min="6"
                max="24"
                step="2"
                class="w-full accent-indigo-500 cursor-pointer h-1.5 bg-theme-border rounded-lg"
              />
              <span class="text-[11px] text-theme-text-muted mt-1 block">
                Avoids extreme 20+ move theoretical weeds in sub-variations.
              </span>
            </div>

            <!-- Options Checkboxes -->
            <div class="space-y-2 pt-2 border-t border-theme-border/60">
              <label class="flex items-center gap-2 cursor-pointer text-xs text-theme-text-primary">
                <input
                  v-model="preserveAnnotated"
                  type="checkbox"
                  class="rounded border-theme-border bg-theme-panel text-rose-500 focus:ring-rose-400 h-3.5 w-3.5"
                />
                <span>Preserve Annotated Moves</span>
              </label>

              <label class="flex items-center gap-2 cursor-pointer text-xs text-theme-text-primary">
                <input
                  v-model="saveAsNew"
                  type="checkbox"
                  class="rounded border-theme-border bg-theme-panel text-rose-500 focus:ring-rose-400 h-3.5 w-3.5"
                />
                <span>Save as New (Keep Original)</span>
              </label>
            </div>
          </div>

          <!-- Prune Action Button -->
          <button
            class="w-full mt-4 py-2.5 px-4 rounded-xl bg-rose-600 hover:bg-rose-500 text-white font-bold text-xs transition-colors flex items-center justify-center gap-2 shadow-lg"
            :disabled="isPruning || !selectedRepId"
            @click="handleRunPrune"
          >
            <Scissors v-if="!isPruning" class="w-4 h-4" />
            <div v-else class="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin"></div>
            <span>{{ isPruning ? 'Analyzing & Pruning...' : 'Prune Repertoire (80/20)' }}</span>
          </button>
        </div>

        <!-- Results & Inspection View -->
        <div class="flex-1 flex flex-col overflow-hidden p-6 gap-4">
          <!-- Notification banners -->
          <div v-if="errorMsg" class="p-3 rounded-lg bg-rose-500/10 border border-rose-500/30 text-xs text-rose-300">
            {{ errorMsg }}
          </div>
          <div v-if="successMsg" class="p-3 rounded-lg bg-emerald-500/10 border border-emerald-500/30 text-xs text-emerald-300 flex items-center gap-2">
            <CheckCircle2 class="w-4 h-4 text-emerald-400" />
            <span>{{ successMsg }}</span>
          </div>

          <div v-if="shortenerResult" class="flex-1 flex flex-col gap-4 overflow-hidden">
            <!-- Reduction Scorecard -->
            <div class="grid grid-cols-2 md:grid-cols-4 gap-3 text-xs">
              <div class="p-3 rounded-xl border border-theme-border bg-theme-panel">
                <div class="text-theme-text-muted mb-1">Study Workload Saved</div>
                <div class="text-xl font-bold text-rose-400">{{ Math.round(shortenerResult.reductionPct) }}%</div>
                <div class="text-[11px] text-theme-text-muted mt-1">
                  {{ shortenerResult.movesRemoved }} moves pruned
                </div>
              </div>

              <div class="p-3 rounded-xl border border-theme-border bg-theme-panel">
                <div class="text-theme-text-muted mb-1">OTB Coverage Retained</div>
                <div class="text-xl font-bold text-emerald-400">{{ Math.round(shortenerResult.coverageRetainedPct) }}%</div>
                <div class="text-[11px] text-theme-text-muted mt-1">
                  High-leverage core positions
                </div>
              </div>

              <div class="p-3 rounded-xl border border-theme-border bg-theme-panel">
                <div class="text-theme-text-muted mb-1">Original Size</div>
                <div class="text-xl font-bold text-theme-text-primary">{{ shortenerResult.originalMoveCount }} moves</div>
                <div class="text-[11px] text-theme-text-muted mt-1">
                  Complete branch tree
                </div>
              </div>

              <div class="p-3 rounded-xl border border-theme-border bg-theme-panel">
                <div class="text-theme-text-muted mb-1">Streamlined Size</div>
                <div class="text-xl font-bold text-indigo-400">{{ shortenerResult.prunedMoveCount }} moves</div>
                <div class="text-[11px] text-theme-text-muted mt-1">
                  Compact 80/20 repertoire
                </div>
              </div>
            </div>

            <!-- View switch tabs -->
            <div class="flex items-center gap-2 border-b border-theme-border pb-2 text-xs">
              <button
                class="px-3 py-1 rounded-lg transition-colors flex items-center gap-1.5"
                :class="activeView === 'pruned' ? 'bg-rose-600 text-white font-medium' : 'text-theme-text-muted hover:text-theme-text-primary'"
                @click="activeView = 'pruned'"
              >
                <span>Pruned Sidelines</span>
                <span class="text-[10px] px-1.5 py-0.2 rounded bg-black/30">({{ shortenerResult.prunedBranches.length }})</span>
              </button>
              <button
                class="px-3 py-1 rounded-lg transition-colors flex items-center gap-1.5"
                :class="activeView === 'mainlines' ? 'bg-indigo-600 text-white font-medium' : 'text-theme-text-muted hover:text-theme-text-primary'"
                @click="activeView = 'mainlines'"
              >
                <span>Core High-Yield Lines</span>
                <span class="text-[10px] px-1.5 py-0.2 rounded bg-black/30">({{ shortenerResult.highYieldLines.length }})</span>
              </button>
            </div>

            <!-- List content -->
            <div class="flex-1 overflow-y-auto space-y-2 pr-1">
              <!-- Pruned lines view -->
              <div v-if="activeView === 'pruned'" class="space-y-2">
                <div
                  v-if="shortenerResult.prunedBranches.length === 0"
                  class="p-6 text-center text-xs text-theme-text-muted"
                >
                  No sidelines met the pruning criteria. Your repertoire is already concise!
                </div>
                <div
                  v-for="(b, idx) in shortenerResult.prunedBranches"
                  :key="idx"
                  class="p-3 rounded-lg border border-theme-border bg-theme-panel text-xs flex flex-col md:flex-row md:items-center justify-between gap-2"
                >
                  <div>
                    <div class="font-mono font-bold text-rose-300">{{ b.sanLine }}</div>
                    <div class="text-[11px] text-theme-text-muted mt-0.5">{{ b.reason }}</div>
                  </div>
                  <div class="flex items-center gap-3 shrink-0">
                    <span class="px-2 py-0.5 rounded bg-theme-panel-header text-[10px] font-mono text-theme-text-muted border border-theme-border">
                      Depth: {{ b.depth }} plies
                    </span>
                    <span v-if="b.frequencyPct > 0" class="text-[11px] text-amber-400 font-semibold">
                      {{ b.frequencyPct.toFixed(1) }}% games
                    </span>
                  </div>
                </div>
              </div>

              <!-- Mainlines view -->
              <div v-else class="space-y-2">
                <div
                  v-if="shortenerResult.highYieldLines.length === 0"
                  class="p-6 text-center text-xs text-theme-text-muted"
                >
                  No custom line comments in retained moves.
                </div>
                <div
                  v-for="(h, idx) in shortenerResult.highYieldLines"
                  :key="idx"
                  class="p-3 rounded-lg border border-theme-border bg-theme-panel text-xs flex items-center justify-between"
                >
                  <div>
                    <div class="font-bold text-indigo-300 font-mono">{{ h.sanLine }}</div>
                    <div class="text-[11px] text-theme-text-muted mt-0.5">{{ h.keyIdea }}</div>
                  </div>
                  <span class="px-2 py-0.5 rounded bg-emerald-500/20 text-emerald-300 text-[10px] font-bold border border-emerald-500/30">
                    CORE MAINLINE
                  </span>
                </div>
              </div>
            </div>
          </div>

          <!-- Empty placeholder -->
          <div v-else class="flex-1 flex flex-col items-center justify-center text-theme-text-muted text-xs gap-3">
            <Layers class="w-10 h-10 text-theme-text-muted/40" />
            <div class="text-center max-w-sm">
              <p class="font-medium text-theme-text-primary">Pareto Repertoire Pruner</p>
              <p class="mt-1 text-theme-text-muted">
                Configure your coverage cutoff on the left and click "Prune Repertoire" to condense your opening lines.
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
