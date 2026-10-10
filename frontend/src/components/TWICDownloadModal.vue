<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import {
  Download,
  X,
  Check,
  AlertCircle,
  Loader2,
  RefreshCw,
  ExternalLink,
  Layers,
  FileCheck
} from '@lucide/vue'
import {
  DownloadTWICIssue,
  DownloadTWICRange,
  CheckTWICIssue,
  GetLatestTWICIssue
} from '../../wailsjs/go/main/App'
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'
import { online } from '../../wailsjs/go/models'

const props = withDefaults(defineProps<{
  dbName?: string
}>(), {
  dbName: 'Active Database'
})

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'imported', totalGames: number): void
}>()

const downloadMode = ref<'single' | 'range'>('single')
const singleIssue = ref<number>(1560)
const rangeStart = ref<number>(1555)
const rangeEnd = ref<number>(1560)

const isDetectingLatest = ref(false)
const isCheckingIssue = ref(false)
const checkStatus = ref<string | null>(null)
const checkStatusOk = ref<boolean>(false)

const isDownloading = ref(false)
const progress = ref<{
  currentIssue: number
  processedIssues: number
  totalIssues: number
  gamesImported: number
} | null>(null)

const errorMessage = ref('')
const successMessage = ref('')
const batchResult = ref<online.TWICBatchResult | null>(null)
const singleResult = ref<online.TWICIssueResult | null>(null)

onMounted(async () => {
  EventsOn('twic:progress', (data: any) => {
    progress.value = data
  })

  // Detect latest issue on mount
  await detectLatest()
})

onUnmounted(() => {
  EventsOff('twic:progress')
})

async function detectLatest() {
  isDetectingLatest.value = true
  errorMessage.value = ''
  try {
    const latest = await GetLatestTWICIssue()
    if (latest > 0) {
      singleIssue.value = latest
      rangeEnd.value = latest
      rangeStart.value = Math.max(1, latest - 5)
    }
  } catch (err: any) {
    console.error('Failed to detect latest TWIC issue:', err)
  } finally {
    isDetectingLatest.value = false
  }
}

async function verifyIssueOnline() {
  const issue = downloadMode.value === 'single' ? singleIssue.value : rangeStart.value
  isCheckingIssue.value = true
  checkStatus.value = null
  try {
    const exists = await CheckTWICIssue(issue)
    checkStatusOk.value = exists
    if (exists) {
      checkStatus.value = `Issue #${issue} is verified and ready on the server!`
    } else {
      checkStatus.value = `Issue #${issue} not found on the server (HTTP 404).`
    }
  } catch (err: any) {
    checkStatusOk.value = false
    checkStatus.value = `Verification error: ${err}`
  } finally {
    isCheckingIssue.value = false
  }
}

async function handleDownload() {
  errorMessage.value = ''
  successMessage.value = ''
  singleResult.value = null
  batchResult.value = null
  isDownloading.value = true
  progress.value = null

  try {
    if (downloadMode.value === 'single') {
      const res = await DownloadTWICIssue(singleIssue.value)
      singleResult.value = res
      if (res.error) {
        errorMessage.value = `Failed to download TWIC #${singleIssue.value}: ${res.error}`
      } else {
        successMessage.value = `Successfully downloaded & imported ${res.gamesImported.toLocaleString()} games from TWIC #${singleIssue.value}!`
        emit('imported', res.gamesImported)
      }
    } else {
      const res = await DownloadTWICRange(rangeStart.value, rangeEnd.value)
      batchResult.value = res
      successMessage.value = `Successfully downloaded & imported ${res.totalImported.toLocaleString()} games across ${res.issues.length} TWIC issues!`
      emit('imported', res.totalImported)
    }
  } catch (err: any) {
    errorMessage.value = `TWIC Download failed: ${err}`
  } finally {
    isDownloading.value = false
  }
}
</script>

<template>
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/75 backdrop-blur-sm p-4 animate-fade-in">
    <div class="bg-zinc-900 border border-zinc-800 rounded-xl shadow-2xl w-full max-w-xl flex flex-col overflow-hidden">
      <!-- Modal Header -->
      <div class="px-5 py-3 border-b border-zinc-800 flex items-center justify-between bg-zinc-950/60">
        <div class="flex items-center gap-2.5">
          <div class="p-2 rounded-lg bg-cyan-500/10 border border-cyan-500/20 text-cyan-400">
            <Download class="w-5 h-5" />
          </div>
          <div>
            <h2 class="text-base font-semibold text-zinc-100 flex items-center gap-2">
              The Week In Chess (TWIC) Downloader
              <span class="text-xs px-2 py-0.5 rounded-full bg-cyan-950 text-cyan-300 border border-cyan-800 font-mono">
                {{ dbName }}
              </span>
            </h2>
            <p class="text-xs text-zinc-400">
              Download weekly master tournament bulletins edited by Mark Crowther directly into your database.
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

      <!-- Mode Selector Tabs -->
      <div class="px-5 pt-3 pb-0 flex items-center gap-2 border-b border-zinc-800 bg-zinc-950/30">
        <button
          class="px-4 py-2 border-b-2 text-xs font-medium transition-colors cursor-pointer flex items-center gap-1.5"
          :class="downloadMode === 'single'
            ? 'border-cyan-400 text-cyan-400 font-semibold'
            : 'border-transparent text-zinc-400 hover:text-zinc-200'"
          @click="downloadMode = 'single'"
        >
          <FileCheck class="w-3.5 h-3.5" />
          <span>Single Bulletin</span>
        </button>
        <button
          class="px-4 py-2 border-b-2 text-xs font-medium transition-colors cursor-pointer flex items-center gap-1.5"
          :class="downloadMode === 'range'
            ? 'border-cyan-400 text-cyan-400 font-semibold'
            : 'border-transparent text-zinc-400 hover:text-zinc-200'"
          @click="downloadMode = 'range'"
        >
          <Layers class="w-3.5 h-3.5" />
          <span>Issue Range / Batch</span>
        </button>
      </div>

      <!-- Main Form Body -->
      <div class="p-5 flex flex-col gap-4 text-xs">
        <!-- Single Mode Inputs -->
        <div v-if="downloadMode === 'single'" class="flex flex-col gap-2">
          <label class="text-[11px] font-medium text-zinc-300">TWIC Issue Number</label>
          <div class="flex items-center gap-2">
            <input
              v-model.number="singleIssue"
              type="number"
              min="1"
              max="9999"
              placeholder="e.g. 1560"
              class="w-36 bg-zinc-950 border border-zinc-700 rounded-lg px-3 py-1.5 text-zinc-100 placeholder-zinc-500 focus:outline-none focus:border-cyan-500 font-mono text-sm"
              :disabled="isDownloading"
            />
            <button
              type="button"
              class="px-3 py-1.5 rounded-lg bg-zinc-800 hover:bg-zinc-700 text-zinc-300 font-medium text-xs flex items-center gap-1.5 transition-colors cursor-pointer disabled:opacity-50"
              :disabled="isDetectingLatest || isDownloading"
              @click="detectLatest"
            >
              <RefreshCw class="w-3.5 h-3.5" :class="{ 'animate-spin': isDetectingLatest }" />
              <span>{{ isDetectingLatest ? 'Detecting...' : 'Detect Latest' }}</span>
            </button>
            <button
              type="button"
              class="px-3 py-1.5 rounded-lg bg-zinc-800 hover:bg-zinc-700 text-zinc-300 font-medium text-xs flex items-center gap-1.5 transition-colors cursor-pointer disabled:opacity-50"
              :disabled="isCheckingIssue || isDownloading"
              @click="verifyIssueOnline"
            >
              <Check class="w-3.5 h-3.5 text-cyan-400" />
              <span>Verify Online</span>
            </button>
          </div>
          <span class="text-[11px] text-zinc-500">
            Bulletin URL: <code class="text-zinc-400 font-mono">https://theweekinchess.com/zips/twic{{ singleIssue }}g.zip</code>
          </span>
        </div>

        <!-- Range Mode Inputs -->
        <div v-if="downloadMode === 'range'" class="flex flex-col gap-2">
          <div class="grid grid-cols-2 gap-3">
            <div class="flex flex-col gap-1">
              <label class="text-[11px] font-medium text-zinc-300">Start Issue</label>
              <input
                v-model.number="rangeStart"
                type="number"
                min="1"
                max="9999"
                class="w-full bg-zinc-950 border border-zinc-700 rounded-lg px-3 py-1.5 text-zinc-100 font-mono text-sm focus:outline-none focus:border-cyan-500"
                :disabled="isDownloading"
              />
            </div>
            <div class="flex flex-col gap-1">
              <label class="text-[11px] font-medium text-zinc-300">End Issue</label>
              <input
                v-model.number="rangeEnd"
                type="number"
                min="1"
                max="9999"
                class="w-full bg-zinc-950 border border-zinc-700 rounded-lg px-3 py-1.5 text-zinc-100 font-mono text-sm focus:outline-none focus:border-cyan-500"
                :disabled="isDownloading"
              />
            </div>
          </div>
          <div class="flex items-center justify-between text-[11px] text-zinc-500">
            <span>
              Total Issues: <strong class="text-zinc-300">{{ Math.max(0, rangeEnd - rangeStart + 1) }}</strong>
            </span>
            <button
              type="button"
              class="text-cyan-400 hover:text-cyan-300 underline cursor-pointer"
              @click="detectLatest"
            >
              Set End to Latest
            </button>
          </div>
        </div>

        <!-- Online Check Feedback -->
        <div
          v-if="checkStatus"
          class="p-2.5 rounded-lg border text-xs flex items-center gap-2"
          :class="checkStatusOk
            ? 'bg-emerald-950/60 border-emerald-700 text-emerald-200'
            : 'bg-amber-950/60 border-amber-700 text-amber-200'"
        >
          <Check v-if="checkStatusOk" class="w-4 h-4 text-emerald-400 shrink-0" />
          <AlertCircle v-else class="w-4 h-4 text-amber-400 shrink-0" />
          <span>{{ checkStatus }}</span>
        </div>

        <!-- Download & Import Progress Bar -->
        <div v-if="isDownloading && progress" class="flex flex-col gap-1.5 p-3 rounded-lg bg-zinc-950 border border-zinc-800">
          <div class="flex items-center justify-between text-xs text-zinc-300">
            <span class="flex items-center gap-1.5 font-medium">
              <Loader2 class="w-3.5 h-3.5 animate-spin text-cyan-400" />
              Downloading TWIC #{{ progress.currentIssue }}...
            </span>
            <span class="font-mono text-[11px] text-zinc-400">
              {{ progress.processedIssues }} / {{ progress.totalIssues }} issues
            </span>
          </div>
          <div class="w-full bg-zinc-800 rounded-full h-2 overflow-hidden">
            <div
              class="bg-cyan-500 h-full rounded-full transition-all duration-300"
              :style="{ width: `${(progress.processedIssues / progress.totalIssues) * 100}%` }"
            />
          </div>
          <div class="text-[11px] text-zinc-500 flex justify-between">
            <span>Games imported so far:</span>
            <strong class="text-cyan-400 font-mono">{{ progress.gamesImported.toLocaleString() }}</strong>
          </div>
        </div>

        <!-- Result Alerts -->
        <div v-if="errorMessage" class="p-2.5 rounded-lg bg-rose-950/60 border border-rose-700 text-rose-200 text-xs">
          {{ errorMessage }}
        </div>
        <div v-if="successMessage" class="p-2.5 rounded-lg bg-emerald-950/60 border border-emerald-700 text-emerald-200 text-xs flex items-center gap-2">
          <Check class="w-4 h-4 text-emerald-400 shrink-0" />
          <span>{{ successMessage }}</span>
        </div>

        <!-- Batch Results Breakdown -->
        <div v-if="batchResult && batchResult.issues.length > 0" class="max-h-40 overflow-y-auto border border-zinc-800 rounded-lg divide-y divide-zinc-800 text-[11px]">
          <div
            v-for="iss in batchResult.issues"
            :key="iss.issueNumber"
            class="px-3 py-1.5 flex items-center justify-between"
          >
            <div class="flex items-center gap-2">
              <span class="font-mono font-medium text-zinc-300">TWIC #{{ iss.issueNumber }}</span>
              <span v-if="iss.error" class="text-rose-400">{{ iss.error }}</span>
              <span v-else class="text-zinc-500">{{ iss.gamesImported.toLocaleString() }} games</span>
            </div>
            <span class="font-mono text-zinc-500 text-[10px]">{{ iss.durationMs }}ms</span>
          </div>
        </div>
      </div>

      <!-- Modal Footer -->
      <div class="px-5 py-3 border-t border-zinc-800 flex items-center justify-between bg-zinc-950/60">
        <a
          href="https://theweekinchess.com"
          target="_blank"
          class="text-xs text-zinc-400 hover:text-cyan-400 flex items-center gap-1 transition-colors"
        >
          <span>The Week In Chess</span>
          <ExternalLink class="w-3 h-3" />
        </a>
        <div class="flex items-center gap-2.5">
          <button
            type="button"
            class="px-4 py-2 rounded-lg bg-zinc-800 hover:bg-zinc-700 text-zinc-300 font-medium text-xs transition-colors cursor-pointer"
            :disabled="isDownloading"
            @click="emit('close')"
          >
            Close
          </button>
          <button
            type="button"
            class="px-5 py-2 rounded-lg bg-cyan-600 hover:bg-cyan-500 text-white font-medium text-xs flex items-center gap-1.5 transition-colors cursor-pointer disabled:opacity-50"
            :disabled="isDownloading"
            @click="handleDownload"
          >
            <Loader2 v-if="isDownloading" class="w-4 h-4 animate-spin" />
            <Download v-else class="w-4 h-4" />
            <span>{{ isDownloading ? 'Downloading...' : 'Download & Import' }}</span>
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
