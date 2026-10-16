<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, nextTick } from 'vue'
import {
  Search,
  FileText,
  FolderOpen,
  Save,
  Cpu,
  Sparkles,
  BookOpen,
  Trophy,
  BarChart3,
  Award,
  Zap,
  Target,
  Palette,
  Eye,
  Volume2,
  HelpCircle,
  Wrench,
  Dices,
  Globe,
  SlidersHorizontal,
  ArrowRight
} from '@lucide/vue'

export interface CommandItem {
  id: string
  title: string
  subtitle?: string
  category: 'Actions' | 'Openings' | 'Analysis' | 'Training' | 'Options' | 'Tools'
  icon: any
  shortcut?: string
  action: () => void
}

const props = defineProps<{
  isOpen: boolean
  commands: CommandItem[]
}>()

const emit = defineEmits<{
  (e: 'close'): void
}>()

const searchQuery = ref('')
const selectedIndex = ref(0)
const inputRef = ref<HTMLInputElement | null>(null)

const filteredCommands = computed(() => {
  const q = searchQuery.value.trim().toLowerCase()
  if (!q) {
    return props.commands
  }
  return props.commands.filter(
    (c) =>
      c.title.toLowerCase().includes(q) ||
      (c.subtitle && c.subtitle.toLowerCase().includes(q)) ||
      c.category.toLowerCase().includes(q)
  )
})

function close() {
  searchQuery.value = ''
  selectedIndex.value = 0
  emit('close')
}

function selectItem(item: CommandItem) {
  close()
  item.action()
}

function handleKeydown(e: KeyboardEvent) {
  if (!props.isOpen) return

  if (e.key === 'ArrowDown') {
    e.preventDefault()
    if (filteredCommands.value.length > 0) {
      selectedIndex.value = (selectedIndex.value + 1) % filteredCommands.value.length
    }
  } else if (e.key === 'ArrowUp') {
    e.preventDefault()
    if (filteredCommands.value.length > 0) {
      selectedIndex.value =
        (selectedIndex.value - 1 + filteredCommands.value.length) % filteredCommands.value.length
    }
  } else if (e.key === 'Enter') {
    e.preventDefault()
    if (filteredCommands.value.length > 0 && selectedIndex.value < filteredCommands.value.length) {
      selectItem(filteredCommands.value[selectedIndex.value])
    }
  } else if (e.key === 'Escape') {
    e.preventDefault()
    close()
  }
}

onMounted(() => {
  window.addEventListener('keydown', handleKeydown)
})

onUnmounted(() => {
  window.removeEventListener('keydown', handleKeydown)
})

function onOpen() {
  searchQuery.value = ''
  selectedIndex.value = 0
  nextTick(() => {
    inputRef.value?.focus()
  })
}

defineExpose({ onOpen })
</script>

<template>
  <div
    v-if="isOpen"
    class="fixed inset-0 bg-black/70 backdrop-blur-sm flex items-start justify-center pt-24 z-50 p-4 animate-fade-in"
    @click.self="close"
  >
    <div
      class="bg-zinc-900 border border-zinc-700/80 w-full max-w-xl rounded-2xl shadow-2xl overflow-hidden flex flex-col max-h-[70vh] border-t-zinc-600"
    >
      <!-- Search Input Header -->
      <div class="p-3.5 border-b border-zinc-800 flex items-center gap-3 bg-zinc-950/60">
        <Search class="w-5 h-5 text-emerald-400 shrink-0" />
        <input
          ref="inputRef"
          v-model="searchQuery"
          type="text"
          placeholder="Search commands, openings, players, tools... (or ↑↓ to navigate)"
          class="w-full bg-transparent text-sm text-zinc-100 placeholder-zinc-500 focus:outline-hidden"
          @keydown="handleKeydown"
        />
        <span class="px-2 py-0.5 rounded text-[10px] bg-zinc-800 text-zinc-400 font-mono">ESC</span>
      </div>

      <!-- Commands List -->
      <div class="overflow-y-auto p-2 divide-y divide-zinc-800/40 select-none">
        <div v-if="filteredCommands.length === 0" class="p-8 text-center text-zinc-500 text-xs">
          No matching commands found.
        </div>

        <div
          v-for="(cmd, idx) in filteredCommands"
          :key="cmd.id"
          class="flex items-center justify-between px-3.5 py-2.5 rounded-xl cursor-pointer transition-colors"
          :class="{
            'bg-emerald-500/15 text-white border border-emerald-500/30': selectedIndex === idx,
            'text-zinc-300 hover:bg-zinc-800/70 hover:text-white': selectedIndex !== idx
          }"
          @click="selectItem(cmd)"
          @mouseenter="selectedIndex = idx"
        >
          <div class="flex items-center gap-3 min-w-0">
            <div
              class="w-8 h-8 rounded-lg flex items-center justify-center shrink-0"
              :class="selectedIndex === idx ? 'bg-emerald-500 text-zinc-950 font-bold' : 'bg-zinc-800 text-zinc-400'"
            >
              <component :is="cmd.icon" class="w-4 h-4" />
            </div>
            <div class="truncate">
              <div class="flex items-center gap-2">
                <span class="font-medium text-xs truncate">{{ cmd.title }}</span>
                <span
                  class="text-[9px] px-1.5 py-0.2 rounded uppercase tracking-wider font-semibold"
                  :class="selectedIndex === idx ? 'bg-emerald-500/30 text-emerald-300' : 'bg-zinc-800 text-zinc-400'"
                >
                  {{ cmd.category }}
                </span>
              </div>
              <p v-if="cmd.subtitle" class="text-[11px] text-zinc-400 truncate mt-0.5">
                {{ cmd.subtitle }}
              </p>
            </div>
          </div>

          <div class="flex items-center gap-2 shrink-0 ml-3">
            <span v-if="cmd.shortcut" class="text-[10px] font-mono text-zinc-500 bg-zinc-800/80 px-2 py-0.5 rounded">
              {{ cmd.shortcut }}
            </span>
            <ArrowRight class="w-3.5 h-3.5 text-zinc-500" :class="{ 'text-emerald-400': selectedIndex === idx }" />
          </div>
        </div>
      </div>

      <!-- Footer navigation tips -->
      <div class="p-2 border-t border-zinc-800/80 bg-zinc-950/80 flex items-center justify-between text-[11px] text-zinc-500 px-4">
        <span>Navigation: <kbd class="px-1 rounded bg-zinc-800 text-zinc-300">↑</kbd> <kbd class="px-1 rounded bg-zinc-800 text-zinc-300">↓</kbd></span>
        <span>Execute: <kbd class="px-1 rounded bg-zinc-800 text-zinc-300">↵ Enter</kbd></span>
        <span>Close: <kbd class="px-1 rounded bg-zinc-800 text-zinc-300">ESC</kbd></span>
      </div>
    </div>
  </div>
</template>
