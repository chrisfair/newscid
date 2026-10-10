<script setup lang="ts">
import { HelpCircle, X } from '@lucide/vue'

defineEmits<{
  (e: 'close'): void
}>()

const shortcuts = [
  { group: 'Game Navigation', items: [
    { key: 'Left / Right', desc: 'Step backward / forward one move' },
    { key: 'Home / End', desc: 'Jump to beginning / end of game' },
    { key: 'Space', desc: 'Toggle game autoplay' },
    { key: 'F', desc: 'Flip board orientation' },
  ]},
  { group: 'Database & Files', items: [
    { key: 'Ctrl+O', desc: 'Open database (.si4, .pgn)' },
    { key: 'Ctrl+T', desc: 'Edit game headers & tags' },
    { key: 'Ctrl+H', desc: 'Header / General database search' },
    { key: 'Ctrl+D', desc: 'Database Demographics & Rating Tracker' },
    { key: 'Ctrl+N', desc: 'Start a new game' },
    { key: 'Ctrl+S', desc: 'Save current game to open database' },
    { key: 'Ctrl+Del', desc: 'Delete current game from database' },
    { key: 'Ctrl+I', desc: 'Import PGN game' },
    { key: 'Ctrl+E', desc: 'Export PGN game to clipboard' },
    { key: 'Ctrl+Shift+E', desc: 'Export Game (HTML & LaTeX with diagrams)' },
    { key: 'Ctrl+C / Ctrl+V', desc: 'Copy / Paste FEN' },
  ]},
  { group: 'Analysis & Power Tools', items: [
    { key: 'Ctrl+B', desc: 'Setup Board / Visual FEN Editor' },
    { key: 'Ctrl+Space', desc: 'Toggle Trial Mode (explore moves without dirtying game)' },
    { key: 'Ctrl+Z / Ctrl+Y', desc: 'Undo / Redo move' },
    { key: 'Ctrl+G', desc: 'Guess the Move (Solitaire Chess training)' },
    { key: 'F2', desc: 'Toggle live UCI engine analysis' },
    { key: 'F6', desc: 'ECO Opening Classification Browser' },
    { key: 'F7', desc: 'Tournament Crosstables & Standings (Swiss & Round-Robin)' },
    { key: 'F1', desc: 'User Manual & Online Help' },
  ]},
]
</script>

<template>
  <div class="fixed inset-0 bg-black/75 backdrop-blur-xs flex items-center justify-center z-50 p-4 animate-fade-in select-none">
    <div class="bg-zinc-900 border border-zinc-700/80 rounded-2xl w-full max-w-lg shadow-2xl overflow-hidden flex flex-col max-h-[85vh]">
      <!-- Header -->
      <div class="flex items-center justify-between px-5 py-3.5 bg-zinc-950 border-b border-zinc-800">
        <div class="flex items-center gap-2">
          <HelpCircle class="w-5 h-5 text-emerald-400" />
          <h2 class="text-sm font-bold text-zinc-100">Keyboard Shortcuts</h2>
          <span class="text-[11px] text-zinc-500 font-mono">Scid vs. PC Keymap</span>
        </div>
        <button
          class="text-zinc-400 hover:text-white p-1 rounded-lg hover:bg-zinc-800 transition-colors"
          @click="$emit('close')"
        >
          <X class="w-5 h-5" />
        </button>
      </div>

      <!-- Content -->
      <div class="p-5 overflow-y-auto space-y-4">
        <div v-for="section in shortcuts" :key="section.group" class="space-y-2">
          <h3 class="text-xs font-bold text-emerald-400 uppercase tracking-wider">{{ section.group }}</h3>
          <div class="bg-zinc-950 rounded-xl border border-zinc-800 divide-y divide-zinc-800/80">
            <div
              v-for="s in section.items"
              :key="s.key"
              class="flex items-center justify-between px-3.5 py-2 text-xs"
            >
              <span class="text-zinc-300">{{ s.desc }}</span>
              <kbd class="px-2 py-0.5 rounded bg-zinc-900 border border-zinc-700 font-mono text-[11px] text-amber-300 font-semibold shadow-xs">
                {{ s.key }}
              </kbd>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
