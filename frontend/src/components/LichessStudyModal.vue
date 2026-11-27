<script setup lang="ts">
import { ref } from 'vue'
import { X, BookOpen, Download, ExternalLink } from '@lucide/vue'
import { importLichessStudy, type LichessStudyImportResult } from '../utils/v21Api'

const props = defineProps<{
  show: boolean
  dbName?: string
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'loadGameId', gameId: number): void
  (e: 'loadPGN', pgn: string): void
}>()

const studyInput = ref('')
const token = ref('')
const saveToDB = ref(true)
const loading = ref(false)
const errorMsg = ref('')
const result = ref<LichessStudyImportResult | null>(null)

async function handleFetch() {
  loading.value = true
  errorMsg.value = ''
  result.value = null
  try {
    result.value = await importLichessStudy(studyInput.value, token.value, saveToDB.value)
  } catch (err: any) {
    errorMsg.value = err?.message || String(err)
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div v-if="show" class="fixed inset-0 z-50 bg-black/70 backdrop-blur-sm flex items-center justify-center p-4">
    <div class="bg-zinc-900 border border-zinc-700 rounded-xl shadow-2xl w-full max-w-2xl max-h-[85vh] flex flex-col overflow-hidden">
      <div class="flex items-center justify-between px-5 py-3.5 border-b border-zinc-800 bg-zinc-950/60">
        <div class="flex items-center gap-2.5">
          <BookOpen class="w-5 h-5 text-emerald-400" />
          <div>
            <h2 class="text-sm font-bold text-zinc-100">Lichess Study Importer</h2>
            <p class="text-[11px] text-zinc-400">Import multi-chapter studies with comments and variations</p>
          </div>
        </div>
        <button class="p-1.5 rounded hover:bg-zinc-800 text-zinc-400 hover:text-zinc-100 cursor-pointer" @click="emit('close')">
          <X class="w-4 h-4" />
        </button>
      </div>

      <div class="p-5 overflow-y-auto space-y-4">
        <div class="space-y-3">
          <label class="block text-xs text-zinc-300">
            Lichess Study URL or 8-Character ID
            <input
              v-model="studyInput"
              type="text"
              placeholder="https://lichess.org/study/AbCd1234"
              class="mt-1 w-full bg-zinc-950 border border-zinc-700 rounded-lg px-3 py-2 text-xs text-zinc-100 focus:border-emerald-500 outline-none"
              @keyup.enter="handleFetch"
            />
          </label>

          <label class="block text-xs text-zinc-400">
            Lichess Personal Token (optional — only needed for unlisted/private studies)
            <input
              v-model="token"
              type="password"
              placeholder="lip_..."
              class="mt-1 w-full bg-zinc-950 border border-zinc-800 rounded-lg px-3 py-1.5 text-xs text-zinc-100 outline-none"
            />
          </label>

          <div class="flex items-center justify-between gap-3">
            <label class="flex items-center gap-2 text-xs text-zinc-300 cursor-pointer">
              <input v-model="saveToDB" type="checkbox" class="accent-emerald-500" />
              <span>Save chapters into active database{{ dbName ? ` (${dbName})` : '' }}</span>
            </label>
            <button
              class="px-4 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-500 disabled:opacity-50 text-white text-xs font-semibold flex items-center gap-1.5 cursor-pointer"
              :disabled="loading || !studyInput.trim()"
              @click="handleFetch"
            >
              <Download class="w-3.5 h-3.5" />
              {{ loading ? 'Fetching...' : 'Fetch Study' }}
            </button>
          </div>
        </div>

        <div v-if="errorMsg" class="text-xs text-red-400 bg-red-950/40 border border-red-800/50 rounded-lg p-3">
          {{ errorMsg }}
        </div>

        <div v-if="result" class="space-y-2">
          <div class="flex items-center justify-between text-xs text-zinc-300 bg-emerald-950/30 border border-emerald-800/40 rounded-lg px-3 py-2">
            <span>
              Fetched <strong>{{ result.totalChapters }}</strong> chapter(s)
              <span v-if="result.importedCount > 0"> · Imported <strong>{{ result.importedCount }}</strong> into database</span>
            </span>
            <span class="text-[11px] text-zinc-400 flex items-center gap-1 font-mono">
              <ExternalLink class="w-3 h-3" /> {{ result.studyId }}
            </span>
          </div>

          <div class="border border-zinc-800 rounded-lg divide-y divide-zinc-800 max-h-64 overflow-y-auto">
            <button
              v-for="ch in result.chapters"
              :key="ch.chapterIndex"
              class="w-full text-left px-3.5 py-2 hover:bg-zinc-800/70 flex items-center justify-between gap-3 cursor-pointer"
              @click="ch.gameId ? emit('loadGameId', ch.gameId) : emit('loadPGN', ch.pgn); emit('close')"
            >
              <div class="min-w-0">
                <div class="text-xs font-semibold text-zinc-100 truncate">
                  {{ ch.chapterIndex }}. {{ ch.title }}
                </div>
                <div class="text-[11px] text-zinc-400 truncate">
                  {{ ch.white || '?' }} vs {{ ch.black || '?' }} · {{ ch.plyCount }} plies
                </div>
              </div>
              <span class="text-[11px] text-emerald-400 shrink-0">Open →</span>
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
