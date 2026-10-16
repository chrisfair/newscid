<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Info, X, ExternalLink, GitCommit, Calendar, Cpu, CheckCircle } from '@lucide/vue'
import { GetVersionInfo } from '../../wailsjs/go/main/App'

defineEmits<{
  (e: 'close'): void
}>()

const versionData = ref({
  version: '1.2.0',
  gitCommit: 'unknown',
  buildDate: 'unknown',
  channel: 'stable',
  goVersion: 'go1.26',
  platform: 'linux/amd64',
  compiler: 'gc',
})

onMounted(async () => {
  try {
    const info = await GetVersionInfo()
    if (info) {
      versionData.value = info
    }
  } catch (err) {
    console.error('Failed to load version info:', err)
  }
})
</script>

<template>
  <div class="fixed inset-0 bg-black/75 backdrop-blur-xs flex items-center justify-center z-50 p-4 animate-fade-in select-none">
    <div class="bg-zinc-900 border border-zinc-700/80 rounded-2xl w-full max-w-md shadow-2xl overflow-hidden flex flex-col">
      <!-- Header -->
      <div class="flex items-center justify-between px-5 py-3.5 bg-zinc-950 border-b border-zinc-800">
        <div class="flex items-center gap-2">
          <Info class="w-5 h-5 text-emerald-400" />
          <h2 class="text-sm font-bold text-zinc-100">About NewScid</h2>
        </div>
        <button
          class="text-zinc-400 hover:text-white p-1 rounded-lg hover:bg-zinc-800 transition-colors cursor-pointer"
          @click="$emit('close')"
        >
          <X class="w-5 h-5" />
        </button>
      </div>

      <!-- Body -->
      <div class="p-6 flex flex-col items-center text-center space-y-4">
        <!-- Logo / App Icon -->
        <div class="w-16 h-16 rounded-2xl bg-zinc-800 border border-zinc-700 flex items-center justify-center shadow-lg p-2">
          <img src="/appicon.png" alt="NewScid Logo" class="w-full h-full object-contain" />
        </div>

        <div>
          <h3 class="text-lg font-bold text-zinc-100">NewScid</h3>
          <p class="text-xs text-zinc-400 mt-0.5">Modern Chess Information Database</p>
          <div class="mt-2 inline-flex items-center gap-1.5 px-3 py-1 rounded-full bg-emerald-500/10 border border-emerald-500/30 text-emerald-400 text-xs font-semibold">
            <span class="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse" />
            <span>v{{ versionData.version }}</span>
            <span class="text-zinc-500 font-mono">({{ versionData.channel }})</span>
          </div>
        </div>

        <!-- Metadata Cards -->
        <div class="w-full bg-zinc-950 rounded-xl border border-zinc-800 p-3.5 space-y-2.5 text-xs text-left">
          <div class="flex items-center justify-between">
            <span class="text-zinc-400 flex items-center gap-1.5"><GitCommit class="w-3.5 h-3.5 text-zinc-500" /> Commit</span>
            <span class="font-mono text-zinc-200">{{ versionData.gitCommit }}</span>
          </div>
          <div class="flex items-center justify-between">
            <span class="text-zinc-400 flex items-center gap-1.5"><Calendar class="w-3.5 h-3.5 text-zinc-500" /> Build Date</span>
            <span class="font-mono text-zinc-200">{{ versionData.buildDate }}</span>
          </div>
          <div class="flex items-center justify-between">
            <span class="text-zinc-400 flex items-center gap-1.5"><Cpu class="w-3.5 h-3.5 text-zinc-500" /> Runtime</span>
            <span class="font-mono text-zinc-200">{{ versionData.platform }} ({{ versionData.goVersion }})</span>
          </div>
          <div class="flex items-center justify-between">
            <span class="text-zinc-400 flex items-center gap-1.5"><CheckCircle class="w-3.5 h-3.5 text-emerald-500" /> Scid Compatibility</span>
            <span class="font-mono text-zinc-300">.si4, .sn4, .sg4, .pgn</span>
          </div>
        </div>

        <!-- License & Credits -->
        <div class="text-[11px] text-zinc-500 space-y-1">
          <p>Scid vs. PC rewrite powered by Wails v2, Go & Vue 3</p>
          <p>Licensed under GNU General Public License v3.0 (GPL-3.0)</p>
        </div>

        <button
          class="w-full py-2 px-4 rounded-xl bg-emerald-600 hover:bg-emerald-500 text-white text-xs font-semibold transition-colors cursor-pointer"
          @click="$emit('close')"
        >
          Close
        </button>
      </div>
    </div>
  </div>
</template>
