<script setup lang="ts">
import { computed } from 'vue'
import { useNotation, formatMoveNotation } from '../utils/notation'

export interface MoveStat {
  move: string
  uci: string
  count: number
  whiteWins: number
  draws: number
  blackWins: number
  scorePct: number
  avgElo?: number
  eco?: string
}

export interface TreeData {
  fen: string
  totalGames: number
  moves: MoveStat[]
}

const props = defineProps<{
  treeData: TreeData | null
  loading?: boolean
}>()

const emit = defineEmits<{
  (e: 'playMove', san: string): void
}>()

const movesList = computed(() => props.treeData?.moves || [])
const totalGames = computed(() => props.treeData?.totalGames || 0)
</script>

<template>
  <div class="opening-tree flex flex-col h-full bg-zinc-900 border border-zinc-800 rounded-xl overflow-hidden shadow-lg">
    <!-- Header -->
    <div class="tree-header flex items-center justify-between p-2.5 bg-zinc-950 border-b border-zinc-800">
      <div class="flex items-center gap-2">
        <span class="font-semibold text-xs text-zinc-300 uppercase tracking-wider">Opening Tree</span>
        <span class="px-2 py-0.5 rounded-full text-[11px] font-mono bg-zinc-800 text-emerald-400 font-medium">
          {{ totalGames }} games
        </span>
      </div>
      <div v-if="loading" class="animate-spin w-3.5 h-3.5 border-2 border-emerald-500 border-t-transparent rounded-full" />
    </div>

    <!-- Tree Table -->
    <div class="table-container flex-1 overflow-y-auto">
      <div v-if="movesList.length === 0" class="text-zinc-500 text-xs italic text-center py-8">
        No database games match this position.
      </div>

      <table v-else class="w-full text-left font-mono text-xs border-collapse">
        <thead class="bg-zinc-950/80 sticky top-0 text-zinc-400 select-none z-10 border-b border-zinc-800/80">
          <tr>
            <th class="py-2 px-3 font-semibold">Move</th>
            <th class="py-2 px-2 font-semibold text-right">Games</th>
            <th class="py-2 px-2 font-semibold">Score %</th>
            <th class="py-2 px-2 font-semibold text-right text-zinc-500">W / D / L</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-zinc-800/40">
          <tr
            v-for="m in movesList"
            :key="m.move"
            class="hover:bg-emerald-950/30 transition-colors cursor-pointer group"
            @click="emit('playMove', m.move)"
          >
            <!-- Move SAN / Styled -->
            <td class="py-2 px-3 font-bold text-emerald-400 group-hover:text-emerald-300 flex items-center gap-1.5">
              <span>{{ formatMoveNotation(m.move, m.uci || '') }}</span>
            </td>

            <!-- Games Count -->
            <td class="py-2 px-2 text-right text-zinc-300 font-medium">
              {{ m.count }}
            </td>

            <!-- Score % Bar -->
            <td class="py-2 px-2 min-w-[120px]">
              <div class="flex items-center gap-2">
                <span class="w-9 text-right text-[11px] text-zinc-400">
                  {{ m.scorePct.toFixed(1) }}%
                </span>
                <div class="flex-1 h-2 rounded-full overflow-hidden bg-zinc-800 flex">
                  <!-- White win bar -->
                  <div
                    class="bg-zinc-300 h-full"
                    :style="{ width: `${(m.whiteWins / m.count) * 100}%` }"
                    title="White Wins"
                  />
                  <!-- Draw bar -->
                  <div
                    class="bg-zinc-500 h-full"
                    :style="{ width: `${(m.draws / m.count) * 100}%` }"
                    title="Draws"
                  />
                  <!-- Black win bar -->
                  <div
                    class="bg-zinc-900 h-full border-l border-zinc-700"
                    :style="{ width: `${(m.blackWins / m.count) * 100}%` }"
                    title="Black Wins"
                  />
                </div>
              </div>
            </td>

            <!-- W / D / L Breakdown -->
            <td class="py-2 px-2 text-right text-[11px] text-zinc-400 whitespace-nowrap">
              <span class="text-zinc-200">{{ m.whiteWins }}</span>
              <span class="text-zinc-500">/</span>
              <span class="text-zinc-400">{{ m.draws }}</span>
              <span class="text-zinc-500">/</span>
              <span class="text-zinc-400">{{ m.blackWins }}</span>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<style scoped>
.table-container::-webkit-scrollbar {
  width: 6px;
}
.table-container::-webkit-scrollbar-thumb {
  background-color: rgba(255, 255, 255, 0.1);
  border-radius: 4px;
}
</style>
