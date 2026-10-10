<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { X, Sparkles, AlertCircle, Lightbulb, Tag, ArrowRight } from '@lucide/vue'
import { ExplainMove } from '../../wailsjs/go/main/App'
import type { explainer } from '../../wailsjs/go/models'

const props = defineProps<{
  beforeFen: string
  moveUci: string
  scoreCp?: number
  bestMove?: string
}>()

const emit = defineEmits<{
  (e: 'close'): void
}>()

const loading = ref(true)
const explanation = ref<explainer.Explanation | null>(null)
const error = ref('')

async function fetchExplanation() {
  loading.value = true
  error.value = ''
  try {
    const res = await ExplainMove(
      props.beforeFen,
      props.moveUci,
      props.scoreCp ?? 0,
      props.bestMove ?? ''
    )
    explanation.value = res
  } catch (err: unknown) {
    error.value = String(err)
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchExplanation()
})
</script>

<template>
  <div
    class="fixed inset-0 bg-black/70 backdrop-blur-xs flex items-center justify-center z-50 p-4 animate-fade-in"
    @click.self="emit('close')"
  >
    <div
      class="bg-zinc-900 border border-zinc-700/80 w-full max-w-lg rounded-2xl p-5 shadow-2xl flex flex-col gap-4 text-xs select-none"
    >
      <!-- Header -->
      <div class="flex items-center justify-between border-b border-zinc-800 pb-3">
        <div class="flex items-center gap-2.5">
          <div class="w-8 h-8 rounded-lg bg-amber-500/20 border border-amber-500/40 flex items-center justify-center text-amber-300">
            <Sparkles class="w-4 h-4" />
          </div>
          <div>
            <h3 class="font-bold text-sm text-zinc-100 flex items-center gap-2">
              Explain Move: <span class="font-mono text-emerald-400">{{ explanation?.moveSan || moveUci }}</span>
            </h3>
            <p class="text-[11px] text-zinc-400">Natural language tactical & positional breakdown</p>
          </div>
        </div>
        <button
          class="p-1.5 rounded-lg text-zinc-400 hover:text-zinc-200 hover:bg-zinc-800 transition-colors cursor-pointer"
          @click="emit('close')"
        >
          <X class="w-4 h-4" />
        </button>
      </div>

      <!-- Loading State -->
      <div v-if="loading" class="py-8 text-center text-zinc-400 animate-pulse">
        Analyzing position and evaluating motifs...
      </div>

      <!-- Error State -->
      <div v-else-if="error" class="p-3 bg-red-950/40 border border-red-800/60 rounded-xl text-red-300 text-xs">
        {{ error }}
      </div>

      <!-- Content -->
      <div v-else-if="explanation" class="flex flex-col gap-3.5">
        <!-- Assessment Badge & Summary -->
        <div class="p-3 bg-zinc-950/80 rounded-xl border border-zinc-800/80 flex flex-col gap-1.5">
          <div class="flex items-center justify-between">
            <span
              class="px-2 py-0.5 rounded text-[10px] font-bold uppercase tracking-wider"
              :class="{
                'bg-emerald-500/20 text-emerald-300 border border-emerald-500/30': explanation.assessment.includes('Good') || explanation.assessment.includes('Tactical'),
                'bg-cyan-500/20 text-cyan-300 border border-cyan-500/30': explanation.assessment.includes('Positional'),
                'bg-amber-500/20 text-amber-300 border border-amber-500/30': explanation.assessment.includes('Inaccuracy'),
                'bg-red-500/20 text-red-300 border border-red-500/30': explanation.assessment.includes('Mistake')
              }"
            >
              {{ explanation.assessment }}
            </span>
            <span v-if="explanation.bestAlternative && explanation.bestAlternative !== moveUci" class="text-[11px] text-amber-400 font-mono">
              Engine Preferred: {{ explanation.bestAlternative }}
            </span>
          </div>
          <p class="font-semibold text-zinc-200 text-sm mt-1">
            {{ explanation.summary }}
          </p>
        </div>

        <!-- Tactical Motifs & Strategic Concepts Badges -->
        <div v-if="explanation.tacticalMotifs.length > 0 || explanation.strategicConcepts.length > 0" class="flex flex-wrap gap-1.5">
          <span
            v-for="motif in explanation.tacticalMotifs"
            :key="motif"
            class="px-2 py-1 rounded-md bg-amber-500/15 border border-amber-500/30 text-amber-300 text-[11px] font-medium flex items-center gap-1"
          >
            <Tag class="w-3 h-3" /> {{ motif }}
          </span>
          <span
            v-for="concept in explanation.strategicConcepts"
            :key="concept"
            class="px-2 py-1 rounded-md bg-blue-500/15 border border-blue-500/30 text-blue-300 text-[11px] font-medium flex items-center gap-1"
          >
            <Lightbulb class="w-3 h-3" /> {{ concept }}
          </span>
        </div>

        <!-- Detailed Description Paragraph -->
        <div class="p-3.5 bg-zinc-950/60 border border-zinc-800 rounded-xl text-zinc-300 text-xs leading-relaxed">
          {{ explanation.detailedText }}
        </div>
      </div>

      <!-- Footer -->
      <div class="flex justify-end pt-2 border-t border-zinc-800">
        <button
          class="px-4 py-1.5 rounded-lg bg-zinc-800 hover:bg-zinc-700 text-zinc-200 text-xs font-medium cursor-pointer transition-colors"
          @click="emit('close')"
        >
          Close
        </button>
      </div>
    </div>
  </div>
</template>
