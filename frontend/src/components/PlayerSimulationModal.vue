<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import {
  Bot,
  User,
  Users,
  Search,
  Swords,
  Trophy,
  BookOpen,
  Zap,
  Clock,
  Shield,
  Sparkles,
  X,
  Play,
  CheckCircle2,
  AlertCircle
} from '@lucide/vue'
import {
  GetDatabasePlayers,
  GetPlayerPersonality,
  GetPersonas,
  GetPersonaPersonality
} from '../../wailsjs/go/main/App'
import { playerbot, persona } from '../../wailsjs/go/models'

const props = defineProps<{
  activeDbName?: string
  initialPersona?: persona.Persona
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'startMatch', payload: {
    playerName: string
    personaId?: string
    userColor: 'white' | 'black'
    timeControlMinutes: number
    personality: playerbot.PlayerPersonality
  }): void
}>()

// Opponent selection tab: 'database' | 'personas'
const activeTab = ref<'database' | 'personas'>(props.initialPersona ? 'personas' : 'database')
const personasList = ref<persona.Persona[]>([])
const selectedPersona = ref<persona.Persona | null>(props.initialPersona || null)

const availablePlayers = ref<string[]>([])
const searchInput = ref('')
const selectedPlayer = ref('')
const loadingPlayers = ref(true)
const loadingPersonality = ref(false)
const errorMessage = ref('')

const personality = ref<playerbot.PlayerPersonality | null>(null)

// Match Configuration
const userColor = ref<'white' | 'black' | 'random'>('white')
const timeControl = ref<number>(10) // minutes, 0 = untimed
const bookStyle = ref<'weighted' | 'strict'>('weighted')

onMounted(async () => {
  try {
    const pList = await GetPersonas()
    personasList.value = pList || []
  } catch (e) {
    personasList.value = []
  }

  if (props.initialPersona) {
    activeTab.value = 'personas'
    await analyzePersona(props.initialPersona)
  }

  try {
    const list = await GetDatabasePlayers()
    availablePlayers.value = list || []
    if (!props.initialPersona && availablePlayers.value.length > 0) {
      // Pick a popular GM if available or first player
      const preferred = availablePlayers.value.find(p =>
        p.includes('Kasparov') || p.includes('Carlsen') || p.includes('Fischer')
      )
      if (preferred) {
        selectedPlayer.value = preferred
        searchInput.value = preferred
        await analyzePlayer(preferred)
      } else {
        selectedPlayer.value = availablePlayers.value[0]
        searchInput.value = availablePlayers.value[0]
        await analyzePlayer(availablePlayers.value[0])
      }
    }
  } catch (err: any) {
    if (!props.initialPersona) {
      errorMessage.value = `Failed to load database players: ${err?.message || err}`
    }
  } finally {
    loadingPlayers.value = false
  }
})

const filteredPlayers = computed(() => {
  const q = searchInput.value.toLowerCase().trim()
  if (!q) return availablePlayers.value.slice(0, 15)
  return availablePlayers.value
    .filter(p => p.toLowerCase().includes(q))
    .slice(0, 20)
})

async function analyzePlayer(name: string) {
  if (!name.trim()) return
  loadingPersonality.value = true
  errorMessage.value = ''
  selectedPersona.value = null
  try {
    const p = await GetPlayerPersonality(name)
    personality.value = p
    selectedPlayer.value = name
  } catch (err: any) {
    errorMessage.value = `Error analyzing player ${name}: ${err?.message || err}`
    personality.value = null
  } finally {
    loadingPersonality.value = false
  }
}

async function analyzePersona(p: persona.Persona) {
  if (!p || !p.id) return
  loadingPersonality.value = true
  errorMessage.value = ''
  selectedPersona.value = p
  try {
    const pers = await GetPersonaPersonality(p.id)
    personality.value = pers
    selectedPlayer.value = p.displayName
  } catch (err: any) {
    errorMessage.value = `Error analyzing persona ${p.displayName}: ${err?.message || err}`
    personality.value = null
  } finally {
    loadingPersonality.value = false
  }
}

function handleSelectPlayer(name: string) {
  searchInput.value = name
  analyzePlayer(name)
}

function handleSelectPersona(p: persona.Persona) {
  analyzePersona(p)
}

function handleStartMatch() {
  if (!personality.value) return
  let finalColor: 'white' | 'black' = 'white'
  if (userColor.value === 'random') {
    finalColor = Math.random() < 0.5 ? 'white' : 'black'
  } else {
    finalColor = userColor.value
  }

  emit('startMatch', {
    playerName: personality.value.name,
    personaId: selectedPersona.value?.id,
    userColor: finalColor,
    timeControlMinutes: timeControl.value,
    personality: personality.value
  })
}
</script>

<template>
  <div class="fixed inset-0 z-50 bg-black/70 backdrop-blur-sm flex items-center justify-center p-4">
    <div class="modal-card bg-zinc-900 border border-zinc-700/80 rounded-2xl w-full max-w-2xl shadow-2xl overflow-hidden flex flex-col max-h-[90vh]">
      <!-- Modal Header -->
      <div class="flex items-center justify-between p-4 bg-zinc-950 border-b border-zinc-800">
        <div class="flex items-center gap-2.5">
          <div class="w-8 h-8 rounded-xl bg-purple-500/20 border border-purple-500/40 flex items-center justify-center text-purple-400">
            <Bot class="w-4 h-4" />
          </div>
          <div>
            <h2 class="text-base font-bold text-zinc-100 flex items-center gap-2">
              <span>Player Simulation Training</span>
              <span class="px-2 py-0.5 rounded-full text-[10px] font-mono bg-purple-950/80 border border-purple-800/60 text-purple-300">
                AI Sparring Bot
              </span>
            </h2>
            <p class="text-xs text-zinc-400">
              Train against simulated opponents using their real database opening trees and calibrated strength.
            </p>
          </div>
        </div>

        <button
          class="text-zinc-400 hover:text-white p-1 rounded-lg hover:bg-zinc-800 transition-colors"
          @click="emit('close')"
        >
          <X class="w-5 h-5" />
        </button>
      </div>

      <!-- Modal Body -->
      <div class="flex-1 overflow-y-auto p-4 space-y-4">
        <!-- Error Alert -->
        <div
          v-if="errorMessage"
          class="p-3 rounded-xl bg-rose-950/60 border border-rose-800/60 text-xs text-rose-300 flex items-center gap-2"
        >
          <AlertCircle class="w-4 h-4 text-rose-400 shrink-0" />
          <span>{{ errorMessage }}</span>
        </div>

        <!-- Mode Tabs: Database Opponents vs Unified Personas -->
        <div class="flex items-center gap-2 p-1 bg-zinc-950 border border-zinc-800 rounded-xl text-xs">
          <button
            class="flex-1 py-1.5 px-3 rounded-lg font-semibold flex items-center justify-center gap-1.5 transition-colors cursor-pointer"
            :class="activeTab === 'database'
              ? 'bg-zinc-800 text-white shadow-xs'
              : 'text-zinc-400 hover:text-zinc-200'"
            @click="activeTab = 'database'"
          >
            <User class="w-3.5 h-3.5" />
            <span>Database Opponents</span>
          </button>
          <button
            class="flex-1 py-1.5 px-3 rounded-lg font-semibold flex items-center justify-center gap-1.5 transition-colors cursor-pointer"
            :class="activeTab === 'personas'
              ? 'bg-purple-600 text-white shadow-xs'
              : 'text-zinc-400 hover:text-zinc-200'"
            @click="activeTab = 'personas'"
          >
            <Users class="w-3.5 h-3.5" />
            <span>Unified Personas ({{ personasList.length }})</span>
          </button>
        </div>

        <!-- 1. Database Opponents Selection -->
        <div v-if="activeTab === 'database'" class="space-y-1.5">
          <label class="text-xs font-semibold text-zinc-300 flex items-center justify-between">
            <span>Select Player to Simulate:</span>
            <span class="text-[11px] text-zinc-500 font-normal">
              {{ availablePlayers.length }} players in {{ activeDbName || 'database' }}
            </span>
          </label>

          <div class="relative">
            <Search class="w-4 h-4 text-zinc-500 absolute left-3 top-3" />
            <input
              v-model="searchInput"
              type="text"
              class="w-full bg-zinc-950 border border-zinc-800 rounded-xl pl-9 pr-24 py-2 text-sm text-zinc-100 placeholder-zinc-500 focus:outline-none focus:ring-2 focus:ring-purple-500/50"
              placeholder="Search opponent or Grandmaster name..."
              @keydown.enter="analyzePlayer(searchInput)"
            />
            <button
              class="absolute right-1.5 top-1.5 px-3 py-1 bg-purple-600 hover:bg-purple-500 text-white rounded-lg text-xs font-semibold transition-colors"
              :disabled="loadingPersonality"
              @click="analyzePlayer(searchInput)"
            >
              Analyze
            </button>
          </div>

          <!-- Quick Suggestion Badges -->
          <div v-if="filteredPlayers.length > 0" class="flex flex-wrap gap-1 pt-1 max-h-24 overflow-y-auto">
            <button
              v-for="pName in filteredPlayers"
              :key="pName"
              class="px-2 py-0.5 rounded-md text-xs font-mono transition-colors"
              :class="selectedPlayer === pName
                ? 'bg-purple-900/60 text-purple-200 border border-purple-600/60 font-bold'
                : 'bg-zinc-950/60 text-zinc-400 hover:text-zinc-200 hover:bg-zinc-800 border border-zinc-800/60'"
              @click="handleSelectPlayer(pName)"
            >
              {{ pName }}
            </button>
          </div>
        </div>

        <!-- 2. Unified Personas Selection -->
        <div v-else class="space-y-2">
          <label class="text-xs font-semibold text-zinc-300 flex items-center justify-between">
            <span>Choose Persona to Spar Against:</span>
            <span class="text-[11px] text-purple-400 font-mono">Consolidated Multi-Alias Sparring</span>
          </label>

          <div v-if="personasList.length > 0" class="grid grid-cols-2 gap-2 max-h-44 overflow-y-auto p-1">
            <button
              v-for="p in personasList"
              :key="p.id"
              class="p-2.5 rounded-xl border text-left flex items-center gap-2.5 transition-all cursor-pointer"
              :class="selectedPersona?.id === p.id
                ? 'bg-purple-950/50 border-purple-500 text-white shadow-xs'
                : 'bg-zinc-950/80 border-zinc-800 hover:border-zinc-700 text-zinc-300'"
              @click="handleSelectPersona(p)"
            >
              <div
                class="w-8 h-8 rounded-lg flex items-center justify-center font-bold text-xs text-white shrink-0"
                :style="{ backgroundColor: p.avatarColor || '#8b5cf6' }"
              >
                {{ p.displayName.charAt(0).toUpperCase() }}
              </div>
              <div class="min-w-0 flex-1">
                <div class="font-bold text-xs truncate">{{ p.displayName }}</div>
                <div class="flex items-center gap-2 text-[10px] text-zinc-400 mt-0.5">
                  <span class="text-purple-300 font-mono">~{{ p.estimatedElo }} Elo</span>
                  <span>•</span>
                  <span>{{ p.identifiers?.length || 0 }} aliases</span>
                </div>
              </div>
            </button>
          </div>
          <div v-else class="p-6 text-center text-xs text-zinc-500 bg-zinc-950/50 rounded-xl border border-zinc-800/80">
            No personas configured yet. Create a persona in Tools -> Player Personas.
          </div>
        </div>

        <!-- Loading State -->
        <div v-if="loadingPersonality" class="py-12 flex flex-col items-center justify-center space-y-2 text-zinc-400">
          <div class="animate-spin w-6 h-6 border-2 border-purple-500 border-t-transparent rounded-full" />
          <span class="text-xs">Extracting historical games & opening repertoire...</span>
        </div>

        <!-- Player Personality Dossier Card -->
        <div
          v-else-if="personality"
          class="personality-card bg-zinc-950/90 border border-purple-500/30 rounded-2xl p-4 space-y-4 shadow-inner"
        >
          <!-- Dossier Header -->
          <div class="flex items-start justify-between">
            <div class="flex items-center gap-3">
              <div class="w-12 h-12 rounded-2xl bg-gradient-to-br from-purple-600 to-indigo-700 flex items-center justify-center text-white font-bold text-lg shadow-md shadow-purple-900/30">
                {{ personality.name.slice(0, 2).toUpperCase() }}
              </div>
              <div>
                <h3 class="text-base font-bold text-white flex items-center gap-2">
                  <span>{{ personality.name }}</span>
                  <span class="px-2 py-0.5 rounded-full text-xs font-mono font-bold bg-amber-500/20 text-amber-300 border border-amber-500/40">
                    {{ personality.estimatedElo }} Elo
                  </span>
                  <span v-if="personality.playStyle" class="px-2 py-0.5 rounded-full text-xs font-mono font-semibold bg-purple-500/20 text-purple-300 border border-purple-500/40">
                    {{ personality.playStyle }}
                  </span>
                </h3>
                <p class="text-xs text-zinc-400 flex items-center gap-2 mt-0.5">
                  <span>{{ personality.totalGames }} historical games</span>
                  <span>•</span>
                  <span>Peak {{ personality.peakElo }} Elo</span>
                  <span v-if="personality.avgGamePlies">•</span>
                  <span v-if="personality.avgGamePlies">Avg {{ (personality.avgGamePlies / 2).toFixed(0) }} moves</span>
                </p>
              </div>
            </div>

            <!-- Win Rate Badge -->
            <div class="text-right">
              <div class="text-xs text-zinc-400 font-medium">Historical Score</div>
              <div class="text-sm font-bold font-mono text-emerald-400">
                {{ personality.winRate.toFixed(1) }}%
              </div>
            </div>
          </div>

          <!-- Performance Breakdown Grid -->
          <div class="grid grid-cols-3 gap-2 text-center text-xs">
            <div class="p-2 rounded-xl bg-zinc-900/80 border border-zinc-800">
              <div class="text-emerald-400 font-bold font-mono text-sm">{{ personality.wins }}</div>
              <div class="text-zinc-500 text-[10px] uppercase font-semibold">Wins</div>
            </div>
            <div class="p-2 rounded-xl bg-zinc-900/80 border border-zinc-800">
              <div class="text-amber-400 font-bold font-mono text-sm">{{ personality.draws }}</div>
              <div class="text-zinc-500 text-[10px] uppercase font-semibold">Draws</div>
            </div>
            <div class="p-2 rounded-xl bg-zinc-900/80 border border-zinc-800">
              <div class="text-rose-400 font-bold font-mono text-sm">{{ personality.losses }}</div>
              <div class="text-zinc-500 text-[10px] uppercase font-semibold">Losses</div>
            </div>
          </div>

          <!-- Opening Repertoire DNA -->
          <div class="space-y-2 border-t border-zinc-800/80 pt-3">
            <div class="text-xs font-semibold text-zinc-300 flex items-center gap-1.5">
              <BookOpen class="w-3.5 h-3.5 text-purple-400" />
              <span>Opening Repertoire DNA</span>
            </div>

            <div class="grid grid-cols-2 gap-3 text-xs">
              <!-- White Repertoire -->
              <div class="p-2.5 rounded-xl bg-zinc-900/80 border border-zinc-800 space-y-1">
                <span class="text-zinc-400 font-semibold block text-[11px]">As White (Top ECOs):</span>
                <div v-if="personality.topWhiteECOs.length > 0" class="flex flex-wrap gap-1">
                  <span
                    v-for="eco in personality.topWhiteECOs"
                    :key="eco"
                    class="px-1.5 py-0.5 rounded bg-zinc-800 text-zinc-200 font-mono text-[10px]"
                  >
                    {{ eco }}
                  </span>
                </div>
                <div v-else class="text-zinc-500 italic text-[11px]">No White games</div>
              </div>

              <!-- Black Repertoire -->
              <div class="p-2.5 rounded-xl bg-zinc-900/80 border border-zinc-800 space-y-1">
                <span class="text-zinc-400 font-semibold block text-[11px]">As Black (Top ECOs):</span>
                <div v-if="personality.topBlackECOs.length > 0" class="flex flex-wrap gap-1">
                  <span
                    v-for="eco in personality.topBlackECOs"
                    :key="eco"
                    class="px-1.5 py-0.5 rounded bg-zinc-800 text-zinc-200 font-mono text-[10px]"
                  >
                    {{ eco }}
                  </span>
                </div>
                <div v-else class="text-zinc-500 italic text-[11px]">No Black games</div>
              </div>
            </div>
          </div>
        </div>

        <!-- Match Setup Options -->
        <div class="match-options grid grid-cols-2 gap-3 text-xs">
          <!-- Color Picker -->
          <div class="p-3 rounded-xl bg-zinc-950 border border-zinc-800 space-y-2">
            <label class="font-semibold text-zinc-300 block">Your Color:</label>
            <div class="grid grid-cols-3 gap-1">
              <button
                class="py-1.5 rounded-lg border font-semibold transition-all text-center"
                :class="userColor === 'white'
                  ? 'bg-zinc-200 text-zinc-950 border-white shadow-sm'
                  : 'bg-zinc-900 text-zinc-400 border-zinc-800 hover:text-white'"
                @click="userColor = 'white'"
              >
                White
              </button>
              <button
                class="py-1.5 rounded-lg border font-semibold transition-all text-center"
                :class="userColor === 'black'
                  ? 'bg-zinc-800 text-white border-zinc-600 shadow-sm'
                  : 'bg-zinc-900 text-zinc-400 border-zinc-800 hover:text-white'"
                @click="userColor = 'black'"
              >
                Black
              </button>
              <button
                class="py-1.5 rounded-lg border font-semibold transition-all text-center"
                :class="userColor === 'random'
                  ? 'bg-purple-900/60 text-purple-200 border-purple-500 shadow-sm'
                  : 'bg-zinc-900 text-zinc-400 border-zinc-800 hover:text-white'"
                @click="userColor = 'random'"
              >
                Random
              </button>
            </div>
          </div>

          <!-- Time Control -->
          <div class="p-3 rounded-xl bg-zinc-950 border border-zinc-800 space-y-2">
            <label class="font-semibold text-zinc-300 block">Time Control:</label>
            <div class="grid grid-cols-4 gap-1">
              <button
                v-for="tc in [0, 3, 5, 10]"
                :key="tc"
                class="py-1.5 rounded-lg border font-semibold transition-all text-center font-mono"
                :class="timeControl === tc
                  ? 'bg-purple-600 text-white border-purple-400 shadow-sm'
                  : 'bg-zinc-900 text-zinc-400 border-zinc-800 hover:text-white'"
                @click="timeControl = tc"
              >
                {{ tc === 0 ? 'Casual' : `${tc}m` }}
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- Modal Footer -->
      <div class="p-4 bg-zinc-950 border-t border-zinc-800 flex items-center justify-between">
        <div class="text-xs text-zinc-400 flex items-center gap-1.5">
          <Zap class="w-3.5 h-3.5 text-amber-400" />
          <span>Plays from historical opening book, calibrated midgame AI</span>
        </div>

        <div class="flex items-center gap-2">
          <button
            class="px-4 py-2 rounded-xl text-xs font-semibold text-zinc-400 hover:text-white hover:bg-zinc-800 transition-colors"
            @click="emit('close')"
          >
            Cancel
          </button>
          <button
            class="px-5 py-2 rounded-xl text-xs font-bold bg-gradient-to-r from-purple-600 to-indigo-600 hover:from-purple-500 hover:to-indigo-500 text-white shadow-lg shadow-purple-900/40 transition-all flex items-center gap-1.5 cursor-pointer disabled:opacity-50"
            :disabled="!personality || loadingPersonality"
            @click="handleStartMatch"
          >
            <Swords class="w-4 h-4" />
            <span>Start Sparring Match</span>
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
