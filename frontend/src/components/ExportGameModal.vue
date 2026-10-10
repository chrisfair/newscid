<template>
  <div v-if="isOpen" class="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center z-50 p-4">
    <div class="bg-zinc-900 border border-zinc-700 rounded-xl shadow-2xl w-full max-w-4xl max-h-[90vh] flex flex-col overflow-hidden text-zinc-100">
      
      <!-- Modal Header -->
      <div class="flex items-center justify-between px-6 py-4 border-b border-zinc-800 bg-zinc-950/50">
        <div class="flex items-center gap-3">
          <div class="p-2 rounded-lg bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
            <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12" />
            </svg>
          </div>
          <div>
            <h2 class="text-lg font-bold">Export Game (HTML & LaTeX)</h2>
            <p class="text-xs text-zinc-400">Publish high-resolution games with vector diagrams and rich annotations</p>
          </div>
        </div>
        <button
          @click="close"
          class="p-1.5 text-zinc-400 hover:text-zinc-100 hover:bg-zinc-800 rounded-lg transition-colors"
          title="Close (Esc)"
        >
          <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>

      <!-- Controls & Settings Bar -->
      <div class="p-6 border-b border-zinc-800 bg-zinc-900/60 flex flex-col gap-4">
        <div class="flex flex-wrap items-center justify-between gap-4">
          <!-- Format Picker -->
          <div class="flex rounded-lg bg-zinc-950 p-1 border border-zinc-800">
            <button
              @click="format = 'html'"
              :class="format === 'html' ? 'bg-zinc-800 text-emerald-400 font-semibold shadow-sm' : 'text-zinc-400 hover:text-zinc-200'"
              class="px-4 py-1.5 rounded-md text-sm transition-all flex items-center gap-2"
            >
              <span>🌐 Modern HTML</span>
            </button>
            <button
              @click="format = 'latex'"
              :class="format === 'latex' ? 'bg-zinc-800 text-emerald-400 font-semibold shadow-sm' : 'text-zinc-400 hover:text-zinc-200'"
              class="px-4 py-1.5 rounded-md text-sm transition-all flex items-center gap-2"
            >
              <span>📄 LaTeX Article</span>
            </button>
          </div>

          <!-- HTML Specific Settings -->
          <div v-if="format === 'html'" class="flex flex-wrap items-center gap-4 text-xs text-zinc-300">
            <label class="flex items-center gap-1.5 cursor-pointer">
              <span>Theme:</span>
              <select v-model="htmlTheme" class="bg-zinc-950 border border-zinc-700 rounded px-2 py-1 text-zinc-200 focus:outline-none focus:border-emerald-500">
                <option value="dark">Dark Theme</option>
                <option value="light">Light Theme</option>
              </select>
            </label>

            <label class="flex items-center gap-1.5 cursor-pointer">
              <input type="checkbox" v-model="includeDiagram" class="rounded bg-zinc-950 border-zinc-700 text-emerald-500 focus:ring-0" />
              <span>Diagram</span>
            </label>

            <label class="flex items-center gap-1.5 cursor-pointer">
              <input type="checkbox" v-model="includeComments" class="rounded bg-zinc-950 border-zinc-700 text-emerald-500 focus:ring-0" />
              <span>Comments</span>
            </label>

            <label class="flex items-center gap-1.5 cursor-pointer">
              <input type="checkbox" v-model="includeVariations" class="rounded bg-zinc-950 border-zinc-700 text-emerald-500 focus:ring-0" />
              <span>Variations</span>
            </label>
          </div>

          <!-- LaTeX Specific Settings -->
          <div v-if="format === 'latex'" class="flex flex-wrap items-center gap-4 text-xs text-zinc-300">
            <label class="flex items-center gap-1.5 cursor-pointer">
              <span>Doc Class:</span>
              <select v-model="latexDocClass" class="bg-zinc-950 border border-zinc-700 rounded px-2 py-1 text-zinc-200 focus:outline-none focus:border-emerald-500">
                <option value="article">article</option>
                <option value="report">report</option>
              </select>
            </label>

            <label class="flex items-center gap-1.5 cursor-pointer">
              <input type="checkbox" v-model="includeDiagram" class="rounded bg-zinc-950 border-zinc-700 text-emerald-500 focus:ring-0" />
              <span>Diagram (\chessboard)</span>
            </label>
          </div>
        </div>
      </div>

      <!-- Content Tabs & Action Header -->
      <div class="px-6 py-2 bg-zinc-950/40 border-b border-zinc-800/80 flex items-center justify-between text-xs">
        <div class="flex gap-2">
          <button
            @click="activeTab = 'preview'"
            :class="activeTab === 'preview' ? 'text-emerald-400 border-b-2 border-emerald-500 font-semibold' : 'text-zinc-400 hover:text-zinc-200'"
            class="pb-1 px-1 transition-all"
          >
            Live Preview
          </button>
          <button
            @click="activeTab = 'code'"
            :class="activeTab === 'code' ? 'text-emerald-400 border-b-2 border-emerald-500 font-semibold' : 'text-zinc-400 hover:text-zinc-200'"
            class="pb-1 px-1 transition-all"
          >
            Source Code
          </button>
        </div>

        <div class="flex items-center gap-2">
          <button
            @click="copyToClipboard"
            class="px-3 py-1 bg-zinc-800 hover:bg-zinc-700 border border-zinc-700 rounded text-zinc-200 transition-colors flex items-center gap-1.5"
            :title="'Copy full ' + format.toUpperCase() + ' to clipboard'"
          >
            <span v-if="copied" class="text-emerald-400">✓ Copied!</span>
            <span v-else>📋 Copy Code</span>
          </button>
          <button
            @click="downloadFile"
            class="px-3 py-1 bg-emerald-600 hover:bg-emerald-500 text-white rounded font-medium transition-colors flex items-center gap-1.5"
          >
            <span>💾 Save File</span>
          </button>
        </div>
      </div>

      <!-- Main Body / Preview Pane -->
      <div class="flex-1 overflow-auto p-4 bg-zinc-950 flex flex-col">
        <div v-if="loading" class="flex-1 flex flex-col items-center justify-center text-zinc-400 gap-2">
          <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-emerald-500"></div>
          <span class="text-sm">Generating export...</span>
        </div>

        <div v-else-if="errorMessage" class="flex-1 flex flex-col items-center justify-center text-red-400 p-6 text-center">
          <p class="font-semibold mb-1">Failed to export game</p>
          <p class="text-xs text-red-300/80">{{ errorMessage }}</p>
        </div>

        <template v-else>
          <!-- HTML Preview Tab -->
          <div v-if="activeTab === 'preview' && format === 'html'" class="flex-1 w-full h-[450px] bg-white rounded-lg overflow-hidden border border-zinc-800">
            <iframe
              :srcdoc="exportedContent"
              class="w-full h-full border-0"
              sandbox="allow-same-origin"
              title="Export Preview"
            ></iframe>
          </div>

          <!-- LaTeX Preview (Formatted preview) -->
          <div v-else-if="activeTab === 'preview' && format === 'latex'" class="flex-1 font-mono text-xs bg-zinc-900/90 text-zinc-300 p-4 rounded-lg overflow-auto border border-zinc-800 whitespace-pre leading-relaxed select-text">
            {{ exportedContent }}
          </div>

          <!-- Raw Code Tab -->
          <div v-else class="flex-1 flex flex-col">
            <textarea
              readonly
              :value="exportedContent"
              class="w-full flex-1 font-mono text-xs bg-zinc-900 text-emerald-300 p-4 rounded-lg border border-zinc-800 focus:outline-none resize-none leading-relaxed select-all"
              rows="18"
            ></textarea>
          </div>
        </template>
      </div>

      <!-- Footer info -->
      <div class="px-6 py-3 border-t border-zinc-800 bg-zinc-950/70 flex items-center justify-between text-xs text-zinc-400">
        <div>
          Format: <span class="text-zinc-200 font-semibold uppercase">{{ format }}</span> |
          Characters: <span class="text-zinc-200">{{ exportedContent.length }}</span>
        </div>
        <button
          @click="close"
          class="px-4 py-1.5 bg-zinc-800 hover:bg-zinc-700 text-zinc-200 rounded-md transition-colors"
        >
          Close
        </button>
      </div>

    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onMounted } from 'vue'
import { ExportCurrentGameHTML, ExportCurrentGameLaTeX } from '../../wailsjs/go/main/App'

const props = defineProps<{
  isOpen: boolean
}>()

const emit = defineEmits<{
  (e: 'close'): void
}>()

const format = ref<'html' | 'latex'>('html')
const activeTab = ref<'preview' | 'code'>('preview')
const htmlTheme = ref<'dark' | 'light'>('dark')
const includeDiagram = ref(true)
const includeComments = ref(true)
const includeVariations = ref(true)
const latexDocClass = ref('article')

const exportedContent = ref('')
const loading = ref(false)
const errorMessage = ref('')
const copied = ref(false)

async function generateExport() {
  if (!props.isOpen) return
  loading.value = true
  errorMessage.value = ''

  try {
    if (format.value === 'html') {
      const opts = {
        theme: htmlTheme.value,
        includeDiagram: includeDiagram.value,
        diagramPly: -1,
        includeComments: includeComments.value,
        includeVariations: includeVariations.value,
      }
      exportedContent.value = await ExportCurrentGameHTML(opts)
    } else {
      const opts = {
        documentClass: latexDocClass.value,
        package: 'chessboard',
        includeDiagram: includeDiagram.value,
        diagramPly: -1,
      }
      exportedContent.value = await ExportCurrentGameLaTeX(opts)
    }
  } catch (err: any) {
    errorMessage.value = err?.message || String(err)
  } finally {
    loading.value = false
  }
}

watch([() => props.isOpen, format, htmlTheme, includeDiagram, includeComments, includeVariations, latexDocClass], () => {
  if (props.isOpen) {
    generateExport()
  }
})

function close() {
  emit('close')
}

async function copyToClipboard() {
  if (!exportedContent.value) return
  try {
    await navigator.clipboard.writeText(exportedContent.value)
    copied.value = true
    setTimeout(() => {
      copied.value = false
    }, 2000)
  } catch (err) {
    console.error('Failed to copy', err)
  }
}

function downloadFile() {
  if (!exportedContent.value) return
  const ext = format.value === 'html' ? 'html' : 'tex'
  const mime = format.value === 'html' ? 'text/html' : 'text/x-tex'
  const blob = new Blob([exportedContent.value], { type: mime })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `chess_game_export.${ext}`
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
}

onMounted(() => {
  if (props.isOpen) {
    generateExport()
  }
})
</script>
