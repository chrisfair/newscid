<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import {
  FileEdit,
  X,
  Check,
  Calendar,
  Award,
  Globe,
  Tag,
  Hash
} from '@lucide/vue'
import { UpdateGameHeaders } from '../../wailsjs/go/main/App'
import { scid } from '../../wailsjs/go/models'

const props = withDefaults(defineProps<{
  gameId: number
  initialTags?: Record<string, string>
}>(), {
  gameId: 1,
  initialTags: () => ({
    White: '',
    Black: '',
    WhiteElo: '',
    BlackElo: '',
    Event: '',
    Site: '',
    Date: '',
    Round: '',
    Result: '*',
    ECO: ''
  })
})

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'tagsUpdated', summary: scid.GameSummary): void
}>()

const form = reactive({
  White: props.initialTags.White || '',
  Black: props.initialTags.Black || '',
  WhiteElo: props.initialTags.WhiteElo || '',
  BlackElo: props.initialTags.BlackElo || '',
  Event: props.initialTags.Event || '',
  Site: props.initialTags.Site || '',
  Date: props.initialTags.Date || '',
  Round: props.initialTags.Round || '',
  Result: props.initialTags.Result || '*',
  ECO: props.initialTags.ECO || ''
})

const isSaving = ref(false)
const errorMessage = ref('')
const successMessage = ref('')

async function handleSave() {
  isSaving.value = true
  errorMessage.value = ''
  successMessage.value = ''

  try {
    const payload: Record<string, string> = {
      White: String(form.White ?? '').trim(),
      Black: String(form.Black ?? '').trim(),
      WhiteElo: String(form.WhiteElo ?? '').trim(),
      BlackElo: String(form.BlackElo ?? '').trim(),
      Event: String(form.Event ?? '').trim(),
      Site: String(form.Site ?? '').trim(),
      Date: String(form.Date ?? '').trim(),
      Round: String(form.Round ?? '').trim(),
      Result: String(form.Result ?? '').trim(),
      ECO: String(form.ECO ?? '').trim()
    }

    const updatedSummary = await UpdateGameHeaders(props.gameId, payload)
    successMessage.value = 'Game headers updated successfully!'
    emit('tagsUpdated', updatedSummary)
    setTimeout(() => {
      emit('close')
    }, 400)
  } catch (err: any) {
    errorMessage.value = `Failed to update tags: ${err}`
  } finally {
    isSaving.value = false
  }
}
</script>

<template>
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/75 backdrop-blur-sm p-4 animate-fade-in">
    <div class="bg-zinc-900 border border-zinc-800 rounded-xl shadow-2xl w-full max-w-xl flex flex-col overflow-hidden">
      <!-- Modal Header -->
      <div class="px-5 py-3 border-b border-zinc-800 flex items-center justify-between bg-zinc-950/60">
        <div class="flex items-center gap-2.5">
          <div class="p-2 rounded-lg bg-emerald-500/10 border border-emerald-500/20 text-emerald-400">
            <FileEdit class="w-5 h-5" />
          </div>
          <div>
            <h2 class="text-base font-semibold text-zinc-100 flex items-center gap-2">
              Edit Game Headers & Tags
              <span class="text-xs px-2 py-0.5 rounded-full bg-zinc-800 text-zinc-300 font-mono">
                Game #{{ gameId }}
              </span>
            </h2>
            <p class="text-xs text-zinc-400">
              Update player identities, ratings, date, event, round, and theoretical ECO classification.
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

      <!-- Main Form -->
      <form @submit.prevent="handleSave" class="p-5 flex flex-col gap-4 text-xs">
        <!-- White & White Elo -->
        <div class="grid grid-cols-3 gap-3">
          <div class="col-span-2 flex flex-col gap-1">
            <label class="text-[11px] font-medium text-zinc-300 flex items-center gap-1.5">
              <span>White Player Name</span>
            </label>
            <input
              v-model="form.White"
              type="text"
              placeholder="e.g. Kasparov, Garry"
              class="w-full bg-zinc-950 border border-zinc-700 rounded-lg px-3 py-1.5 text-zinc-100 placeholder-zinc-500 focus:outline-none focus:border-emerald-500 font-medium"
              required
            />
          </div>
          <div class="flex flex-col gap-1">
            <label class="text-[11px] font-medium text-zinc-300">White Elo</label>
            <input
              v-model="form.WhiteElo"
              type="number"
              placeholder="e.g. 2800"
              class="w-full bg-zinc-950 border border-zinc-700 rounded-lg px-3 py-1.5 text-zinc-100 placeholder-zinc-500 focus:outline-none focus:border-emerald-500 font-mono"
            />
          </div>
        </div>

        <!-- Black & Black Elo -->
        <div class="grid grid-cols-3 gap-3">
          <div class="col-span-2 flex flex-col gap-1">
            <label class="text-[11px] font-medium text-zinc-300 flex items-center gap-1.5">
              <span>Black Player Name</span>
            </label>
            <input
              v-model="form.Black"
              type="text"
              placeholder="e.g. Karpov, Anatoly"
              class="w-full bg-zinc-950 border border-zinc-700 rounded-lg px-3 py-1.5 text-zinc-100 placeholder-zinc-500 focus:outline-none focus:border-emerald-500 font-medium"
              required
            />
          </div>
          <div class="flex flex-col gap-1">
            <label class="text-[11px] font-medium text-zinc-300">Black Elo</label>
            <input
              v-model="form.BlackElo"
              type="number"
              placeholder="e.g. 2750"
              class="w-full bg-zinc-950 border border-zinc-700 rounded-lg px-3 py-1.5 text-zinc-100 placeholder-zinc-500 focus:outline-none focus:border-emerald-500 font-mono"
            />
          </div>
        </div>

        <!-- Event & Site -->
        <div class="grid grid-cols-2 gap-3">
          <div class="flex flex-col gap-1">
            <label class="text-[11px] font-medium text-zinc-300">Tournament / Event</label>
            <input
              v-model="form.Event"
              type="text"
              placeholder="e.g. World Championship Match"
              class="w-full bg-zinc-950 border border-zinc-700 rounded-lg px-3 py-1.5 text-zinc-100 placeholder-zinc-500 focus:outline-none focus:border-emerald-500"
            />
          </div>
          <div class="flex flex-col gap-1">
            <label class="text-[11px] font-medium text-zinc-300">Site / City</label>
            <input
              v-model="form.Site"
              type="text"
              placeholder="e.g. Moscow, RUS"
              class="w-full bg-zinc-950 border border-zinc-700 rounded-lg px-3 py-1.5 text-zinc-100 placeholder-zinc-500 focus:outline-none focus:border-emerald-500"
            />
          </div>
        </div>

        <!-- Date & Round & ECO -->
        <div class="grid grid-cols-3 gap-3">
          <div class="flex flex-col gap-1">
            <label class="text-[11px] font-medium text-zinc-300 flex items-center gap-1">
              <Calendar class="w-3 h-3 text-zinc-400" />
              <span>Date (YYYY.MM.DD)</span>
            </label>
            <input
              v-model="form.Date"
              type="text"
              placeholder="1985.11.09"
              class="w-full bg-zinc-950 border border-zinc-700 rounded-lg px-3 py-1.5 text-zinc-100 placeholder-zinc-500 focus:outline-none focus:border-emerald-500 font-mono"
            />
          </div>
          <div class="flex flex-col gap-1">
            <label class="text-[11px] font-medium text-zinc-300 flex items-center gap-1">
              <Hash class="w-3 h-3 text-zinc-400" />
              <span>Round</span>
            </label>
            <input
              v-model="form.Round"
              type="text"
              placeholder="e.g. 24"
              class="w-full bg-zinc-950 border border-zinc-700 rounded-lg px-3 py-1.5 text-zinc-100 placeholder-zinc-500 focus:outline-none focus:border-emerald-500 font-mono"
            />
          </div>
          <div class="flex flex-col gap-1">
            <label class="text-[11px] font-medium text-zinc-300 flex items-center gap-1">
              <Tag class="w-3 h-3 text-zinc-400" />
              <span>ECO Code</span>
            </label>
            <input
              v-model="form.ECO"
              type="text"
              placeholder="e.g. B44"
              class="w-full bg-zinc-950 border border-zinc-700 rounded-lg px-3 py-1.5 text-zinc-100 placeholder-zinc-500 focus:outline-none focus:border-emerald-500 font-mono uppercase"
            />
          </div>
        </div>

        <!-- Result Selector -->
        <div class="flex flex-col gap-1.5">
          <label class="text-[11px] font-medium text-zinc-300">Game Result</label>
          <div class="flex items-center gap-2">
            <button
              v-for="res in ['1-0', '0-1', '1/2-1/2', '*']"
              :key="res"
              type="button"
              class="flex-1 py-1.5 rounded-lg border font-mono font-bold text-xs transition-colors cursor-pointer"
              :class="form.Result === res
                ? 'bg-emerald-600 text-white border-emerald-500 shadow-xs'
                : 'bg-zinc-950 text-zinc-400 border-zinc-700 hover:bg-zinc-800 hover:text-zinc-200'"
              @click="form.Result = res"
            >
              {{ res === '*' ? '* (Underway)' : res }}
            </button>
          </div>
        </div>

        <!-- Feedback Alerts -->
        <div v-if="errorMessage" class="p-2.5 rounded-lg bg-rose-950/60 border border-rose-700 text-rose-200 text-xs">
          {{ errorMessage }}
        </div>
        <div v-if="successMessage" class="p-2.5 rounded-lg bg-emerald-950/60 border border-emerald-700 text-emerald-200 text-xs flex items-center gap-2">
          <Check class="w-4 h-4 text-emerald-400" />
          <span>{{ successMessage }}</span>
        </div>

        <!-- Footer Actions -->
        <div class="flex items-center justify-end gap-3 pt-2 border-t border-zinc-800 mt-2">
          <button
            type="button"
            class="px-4 py-2 rounded-lg bg-zinc-800 hover:bg-zinc-700 text-zinc-300 font-medium text-xs transition-colors cursor-pointer"
            @click="emit('close')"
          >
            Cancel
          </button>
          <button
            type="submit"
            class="px-5 py-2 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white font-medium text-xs flex items-center gap-1.5 transition-colors cursor-pointer disabled:opacity-50"
            :disabled="isSaving"
            @click="handleSave"
          >
            <Check class="w-4 h-4" />
            <span>{{ isSaving ? 'Saving...' : 'Save Headers' }}</span>
          </button>
        </div>
      </form>
    </div>
  </div>
</template>
