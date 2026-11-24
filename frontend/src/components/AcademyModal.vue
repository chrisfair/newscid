<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import {
  X,
  Trophy,
  GraduationCap,
  Database,
  Sliders,
  Play,
  CheckCircle2,
  Lock,
  Sparkles,
  Search,
  Trash2,
  ArrowRight,
  HelpCircle,
  Award,
  FolderOpen
} from '@lucide/vue'
import Chessboard, { type BoardArrow } from './Chessboard.vue'
import {
  getAcademyOverview,
  completeAcademyLesson,
  testOutAcademyTier,
  saveCustomPersonality,
  deleteCustomPersonality,
  buildAcademyPersonalityFromDB,
  searchDatabasePlayers,
  type CareerOverview,
  type AcademyPersonality,
  type AcademyLesson,
  type DatabasePlayerSuggestion
} from '../utils/academyApi'

const props = defineProps<{
  show: boolean
  initialTab?: 'career' | 'build-db' | 'editor'
  initialLessonId?: string
  dbName?: string
  dbNumGames?: number
  pieceSet?: string
  boardTheme?: { light: string; dark: string; highlight: string }
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'startCareerGame', personality: AcademyPersonality, color: 'white' | 'black'): void
  (e: 'openDatabase'): void
}>()

const activeTab = ref<'career' | 'lesson' | 'build-db' | 'editor'>('career')
const loading = ref(false)
const errorMsg = ref('')
const statusMsg = ref('')
const overview = ref<CareerOverview | null>(null)
const selectedTier = ref(1)
const playColor = ref<'white' | 'black'>('white')

// Lesson Runner state
const activeLesson = ref<AcademyLesson | null>(null)
const lessonStepIdx = ref(0)
const lessonFeedback = ref('')
const lessonStepPassed = ref(false)
const showLessonHint = ref(false)

// Build from DB state
const dbPlayerQuery = ref('')
const dbSuggestions = ref<DatabasePlayerSuggestion[]>([])
const buildingFromDb = ref(false)
const lastBuiltPersonality = ref<AcademyPersonality | null>(null)

// Custom Personality Editor state
const customName = ref('')
const customElo = ref(1400)
const customBio = ref('')
const customAggression = ref(60)
const customTrade = ref(50)
const customSharpness = ref(60)
const customSolidity = ref(60)
const customBlunder = ref(25)

const unlockedSet = computed(() => new Set(overview.value?.profile?.unlockedTiers || [1]))
const completedLessonCount = computed(() => {
  const m = overview.value?.profile?.completedLessons || {}
  return Object.values(m).filter(Boolean).length
})
const totalLessonCount = computed(() => overview.value?.lessons?.length || 18)

const currentTierInfo = computed(() =>
  overview.value?.tiers.find(t => t.tier === selectedTier.value) || null
)

const tierPersonalities = computed(() =>
  (overview.value?.personalities || []).filter(p => p.tier === selectedTier.value && p.source === 'builtin')
)

const tierLessons = computed(() =>
  (overview.value?.lessons || []).filter(l => l.tier === selectedTier.value)
)

const customPersonalities = computed(() =>
  (overview.value?.personalities || []).filter(p => p.source !== 'builtin')
)

const activeLessonStep = computed(() => {
  if (!activeLesson.value) return null
  return activeLesson.value.steps[lessonStepIdx.value] || null
})

const lessonBoardArrows = computed<BoardArrow[]>(() => {
  const step = activeLessonStep.value
  if (!step?.arrows) return []
  return step.arrows.map(a => ({ from: a.from, to: a.to, color: a.color || '#10b981' }))
})

const lessonLegalMoves = computed<string[]>(() => {
  const step = activeLessonStep.value
  if (!step || !step.expectedMoves || step.expectedMoves.length === 0) return []
  // Allow all expected moves plus any standard square move attempt via Chessboard
  return step.expectedMoves
})

async function loadOverview() {
  loading.value = true
  errorMsg.value = ''
  try {
    const data = await getAcademyOverview()
    if (data) {
      overview.value = data
      if (data.profile?.currentTier) {
        selectedTier.value = data.profile.currentTier
      }
      if (props.initialLessonId) {
        const found = data.lessons.find(l => l.id === props.initialLessonId)
        if (found) {
          openLesson(found)
        }
      }
    }
  } catch (e: any) {
    errorMsg.value = e?.message || 'Failed to load Academy data'
  } finally {
    loading.value = false
  }
}

async function refreshDbPlayers() {
  if (!props.dbName) {
    dbSuggestions.value = []
    return
  }
  try {
    dbSuggestions.value = await searchDatabasePlayers(dbPlayerQuery.value, 20)
  } catch {
    dbSuggestions.value = []
  }
}

watch(() => props.show, (visible) => {
  if (visible) {
    activeTab.value = props.initialTab || 'career'
    statusMsg.value = ''
    errorMsg.value = ''
    loadOverview()
    refreshDbPlayers()
  }
})

watch(dbPlayerQuery, () => {
  refreshDbPlayers()
})

watch(() => props.dbName, () => {
  refreshDbPlayers()
})

onMounted(() => {
  if (props.show) {
    activeTab.value = props.initialTab || 'career'
    loadOverview()
    refreshDbPlayers()
  }
})

function openLesson(lesson: AcademyLesson) {
  activeLesson.value = lesson
  lessonStepIdx.value = 0
  lessonFeedback.value = ''
  showLessonHint.value = false
  lessonStepPassed.value = !(lesson.steps[0]?.expectedMoves?.length)
  activeTab.value = 'lesson'
}

function handleLessonBoardMove(uci: string) {
  const step = activeLessonStep.value
  if (!step) return
  if (!step.expectedMoves || step.expectedMoves.length === 0) {
    lessonStepPassed.value = true
    return
  }
  const clean = uci.toLowerCase().trim()
  if (step.expectedMoves.some(m => m.toLowerCase() === clean)) {
    lessonStepPassed.value = true
    lessonFeedback.value = step.successFeedback || 'Correct move!'
  } else {
    lessonFeedback.value = 'Not quite that move — try again or click "Show Hint".'
  }
}

async function nextLessonStep() {
  if (!activeLesson.value) return
  if (lessonStepIdx.value + 1 < activeLesson.value.steps.length) {
    lessonStepIdx.value++
    lessonFeedback.value = ''
    showLessonHint.value = false
    const nextStep = activeLesson.value.steps[lessonStepIdx.value]
    lessonStepPassed.value = !(nextStep?.expectedMoves?.length)
  } else {
    // Completed lesson!
    const updated = await completeAcademyLesson(activeLesson.value.id)
    if (updated && overview.value) {
      overview.value.profile = updated
    }
    statusMsg.value = `Completed lesson: ${activeLesson.value.title}!`
    activeLesson.value = null
    activeTab.value = 'career'
  }
}

async function handleTestOut(tier: number) {
  const updated = await testOutAcademyTier(tier)
  if (updated && overview.value) {
    overview.value.profile = updated
    selectedTier.value = tier
    statusMsg.value = `Unlocked up to Tier ${tier}!`
  }
}

async function handleBuildFromDb(playerName: string) {
  const clean = playerName.trim()
  if (!clean) {
    errorMsg.value = 'Enter or select a player name from the open database.'
    return
  }
  buildingFromDb.value = true
  errorMsg.value = ''
  statusMsg.value = ''
  try {
    const built = await buildAcademyPersonalityFromDB(clean)
    if (built) {
      lastBuiltPersonality.value = built
      statusMsg.value = `Built playable personality "${built.name}" (${built.elo} Elo) from ${built.sourceGames || 0} games!`
      await loadOverview()
    }
  } catch (e: any) {
    errorMsg.value = e?.message || String(e)
  } finally {
    buildingFromDb.value = false
  }
}

async function handleSaveCustom() {
  errorMsg.value = ''
  if (!customName.value.trim()) {
    errorMsg.value = 'Please give your custom opponent a name.'
    return
  }
  try {
    const saved = await saveCustomPersonality({
      id: '',
      name: customName.value.trim(),
      title: 'Custom Opponent',
      tier: 1,
      isBoss: false,
      elo: customElo.value,
      avatarColor: '#10b981',
      avatarSymbol: '♘',
      styleTag: '',
      bio: customBio.value.trim() || 'Custom-tuned sparring partner.',
      source: 'custom',
      sliders: {
        aggression: customAggression.value,
        tradeTendency: customTrade.value,
        tacticalSharpness: customSharpness.value,
        positionalSolidity: customSolidity.value,
        blunderRate: customBlunder.value
      }
    })
    if (saved) {
      statusMsg.value = `Saved custom opponent "${saved.name}" (${saved.elo} Elo)!`
      customName.value = ''
      customBio.value = ''
      await loadOverview()
    }
  } catch (e: any) {
    errorMsg.value = e?.message || String(e)
  }
}

async function handleDeleteCustom(id: string) {
  await deleteCustomPersonality(id)
  await loadOverview()
}

function recordString(id: string): string {
  const r = overview.value?.profile?.records?.[id]
  if (!r) return '0W · 0D · 0L'
  return `${r.wins}W · ${r.draws}D · ${r.losses}L`
}

function launchMatch(p: AcademyPersonality) {
  emit('startCareerGame', p, playColor.value)
  emit('close')
}
</script>

<template>
  <div v-if="show" class="fixed inset-0 z-50 flex items-center justify-center bg-black/75 backdrop-blur-sm p-4">
    <div class="bg-zinc-950 border border-zinc-800 rounded-2xl shadow-2xl w-full max-w-6xl max-h-[90vh] flex flex-col overflow-hidden text-zinc-100">
      <!-- Top Header -->
      <div class="px-6 py-4 bg-zinc-900/90 border-b border-zinc-800 flex items-center justify-between gap-4">
        <div class="flex items-center gap-3">
          <div class="w-10 h-10 rounded-xl bg-amber-500/20 border border-amber-500/40 flex items-center justify-center text-amber-400">
            <Trophy class="w-5 h-5" />
          </div>
          <div>
            <h2 class="text-lg font-extrabold tracking-tight flex items-center gap-2">
              NewScid Academy &amp; Career Ladder
              <span class="text-[11px] font-mono px-2 py-0.5 rounded bg-emerald-500/15 text-emerald-300 border border-emerald-500/30">
                v2.0
              </span>
            </h2>
            <p class="text-xs text-zinc-400">
              Climb 6 Career Tiers, complete Chess University lessons, or clone any player from your open database.
            </p>
          </div>
        </div>

        <div class="flex items-center gap-3">
          <div v-if="overview" class="hidden sm:flex items-center gap-3 px-3 py-1.5 rounded-lg bg-zinc-950 border border-zinc-800 text-xs">
            <div>
              <span class="text-zinc-500">Academy Elo:</span>
              <span class="ml-1 font-bold text-amber-400 font-mono">{{ overview.profile.rating }}</span>
            </div>
            <span class="text-zinc-700">|</span>
            <div>
              <span class="text-zinc-500">Lessons:</span>
              <span class="ml-1 font-bold text-emerald-400 font-mono">{{ completedLessonCount }}/{{ totalLessonCount }}</span>
            </div>
          </div>
          <button
            @click="emit('close')"
            class="p-2 rounded-lg text-zinc-400 hover:text-zinc-100 hover:bg-zinc-800 transition-colors"
            title="Close Academy"
          >
            <X class="w-5 h-5" />
          </button>
        </div>
      </div>

      <!-- Tab Bar -->
      <div class="px-6 pt-3 pb-0 bg-zinc-900/40 border-b border-zinc-800 flex flex-wrap items-center gap-2">
        <button
          @click="activeTab = 'career'"
          :class="[
            'px-4 py-2 text-xs font-semibold rounded-t-lg border-b-2 flex items-center gap-1.5 transition-colors',
            activeTab === 'career' || activeTab === 'lesson'
              ? 'border-amber-400 text-amber-300 bg-zinc-900/80'
              : 'border-transparent text-zinc-400 hover:text-zinc-200'
          ]"
        >
          <GraduationCap class="w-4 h-4" />
          Career Ladder &amp; University
        </button>
        <button
          @click="activeTab = 'build-db'; refreshDbPlayers()"
          :class="[
            'px-4 py-2 text-xs font-semibold rounded-t-lg border-b-2 flex items-center gap-1.5 transition-colors',
            activeTab === 'build-db'
              ? 'border-cyan-400 text-cyan-300 bg-zinc-900/80'
              : 'border-transparent text-zinc-400 hover:text-zinc-200'
          ]"
        >
          <Database class="w-4 h-4" />
          Build Opponent from Database
        </button>
        <button
          @click="activeTab = 'editor'"
          :class="[
            'px-4 py-2 text-xs font-semibold rounded-t-lg border-b-2 flex items-center gap-1.5 transition-colors',
            activeTab === 'editor'
              ? 'border-emerald-400 text-emerald-300 bg-zinc-900/80'
              : 'border-transparent text-zinc-400 hover:text-zinc-200'
          ]"
        >
          <Sliders class="w-4 h-4" />
          Custom Opponents ({{ customPersonalities.length }})
        </button>

        <div class="ml-auto pb-2 flex items-center gap-2 text-xs text-zinc-400">
          <span>Play as:</span>
          <button
            @click="playColor = 'white'"
            :class="['px-2.5 py-1 rounded border text-xs font-medium', playColor === 'white' ? 'bg-zinc-100 text-zinc-950 border-white' : 'bg-zinc-900 border-zinc-700 text-zinc-300']"
          >
            ♔ White
          </button>
          <button
            @click="playColor = 'black'"
            :class="['px-2.5 py-1 rounded border text-xs font-medium', playColor === 'black' ? 'bg-zinc-700 text-white border-zinc-400' : 'bg-zinc-900 border-zinc-700 text-zinc-300']"
          >
            ♚ Black
          </button>
        </div>
      </div>

      <!-- Feedback Banners -->
      <div v-if="errorMsg" class="mx-6 mt-3 px-4 py-2 rounded-lg bg-red-950/60 border border-red-800 text-xs text-red-200">
        {{ errorMsg }}
      </div>
      <div v-if="statusMsg" class="mx-6 mt-3 px-4 py-2 rounded-lg bg-emerald-950/60 border border-emerald-800 text-xs text-emerald-200">
        {{ statusMsg }}
      </div>

      <!-- Main Scrollable Content -->
      <div class="flex-1 overflow-y-auto p-6">
        <!-- TAB 1: CAREER LADDER & UNIVERSITY -->
        <div v-if="activeTab === 'career' && overview" class="grid grid-cols-1 lg:grid-cols-12 gap-6">
          <!-- Left: 6 Tiers Sidebar -->
          <div class="lg:col-span-4 space-y-2">
            <div class="text-xs font-bold uppercase tracking-wider text-zinc-400 mb-2">
              Career Tiers (Select a Tier)
            </div>
            <button
              v-for="t in overview.tiers"
              :key="t.tier"
              @click="selectedTier = t.tier"
              :class="[
                'w-full text-left p-3 rounded-xl border transition-all flex items-center justify-between gap-2',
                selectedTier === t.tier
                  ? 'bg-amber-500/15 border-amber-500/60 text-zinc-100'
                  : 'bg-zinc-900/60 border-zinc-800 text-zinc-300 hover:border-zinc-700'
              ]"
            >
              <div>
                <div class="text-sm font-bold flex items-center gap-1.5">
                  <span>Tier {{ t.tier }}: {{ t.name }}</span>
                </div>
                <div class="text-[11px] text-zinc-400">{{ t.ratingRange }} Elo</div>
              </div>
              <div class="flex items-center gap-1">
                <CheckCircle2 v-if="overview.profile.bossExams?.[t.tier]?.passed" class="w-4 h-4 text-emerald-400" />
                <Lock v-else-if="!unlockedSet.has(t.tier)" class="w-4 h-4 text-zinc-500" />
                <Award v-else class="w-4 h-4 text-amber-400" />
              </div>
            </button>

            <div class="mt-4 p-3 rounded-xl bg-zinc-900/70 border border-zinc-800 text-xs text-zinc-400 space-y-2">
              <div class="font-semibold text-zinc-200">Already an experienced player?</div>
              <p>Skip lower tiers directly—you can always revisit any lesson or opponent later.</p>
              <button
                v-if="!unlockedSet.has(selectedTier)"
                @click="handleTestOut(selectedTier)"
                class="w-full py-1.5 px-3 rounded-lg bg-amber-600 hover:bg-amber-500 text-white font-semibold text-xs"
              >
                Unlock Tier {{ selectedTier }} Now (Test Out)
              </button>
            </div>
          </div>

          <!-- Right: Tier University Lessons + Tier Personalities -->
          <div class="lg:col-span-8 space-y-6">
            <div v-if="currentTierInfo" class="p-4 rounded-xl bg-zinc-900/80 border border-zinc-800">
              <div class="flex items-center justify-between gap-2">
                <h3 class="text-base font-bold text-amber-300">
                  Tier {{ currentTierInfo.tier }}: {{ currentTierInfo.name }} ({{ currentTierInfo.ratingRange }} Elo)
                </h3>
                <span v-if="!unlockedSet.has(currentTierInfo.tier)" class="text-xs text-zinc-400 flex items-center gap-1">
                  <Lock class="w-3.5 h-3.5" /> Locked — Beat Tier {{ currentTierInfo.tier - 1 }} Boss or Test Out
                </span>
              </div>
              <p class="text-xs text-zinc-300 mt-1">{{ currentTierInfo.description }}</p>
            </div>

            <!-- University Lessons for this Tier -->
            <div>
              <h4 class="text-xs font-bold uppercase tracking-wider text-emerald-400 mb-3 flex items-center gap-1.5">
                <GraduationCap class="w-4 h-4" />
                Step 1 — Chess University Lessons (Tier {{ selectedTier }})
              </h4>
              <div class="grid grid-cols-1 md:grid-cols-3 gap-3">
                <div
                  v-for="lesson in tierLessons"
                  :key="lesson.id"
                  class="p-3.5 rounded-xl bg-zinc-900/70 border border-zinc-800 hover:border-emerald-500/50 transition-colors flex flex-col justify-between gap-3"
                >
                  <div>
                    <div class="flex items-center justify-between gap-1 mb-1">
                      <span class="text-[10px] uppercase font-bold px-2 py-0.5 rounded bg-zinc-800 text-zinc-300">
                        {{ lesson.category }}
                      </span>
                      <CheckCircle2 v-if="overview.profile.completedLessons?.[lesson.id]" class="w-4 h-4 text-emerald-400" />
                    </div>
                    <div class="text-sm font-bold text-zinc-100">{{ lesson.title }}</div>
                    <p class="text-[11px] text-zinc-400 mt-1 leading-relaxed">{{ lesson.summary }}</p>
                  </div>
                  <button
                    @click="openLesson(lesson)"
                    class="w-full py-1.5 px-3 rounded-lg bg-emerald-600/20 hover:bg-emerald-600/35 text-emerald-300 border border-emerald-500/40 text-xs font-semibold flex items-center justify-center gap-1"
                  >
                    {{ overview.profile.completedLessons?.[lesson.id] ? 'Review Lesson' : 'Start Lesson' }}
                    <ArrowRight class="w-3.5 h-3.5" />
                  </button>
                </div>
              </div>
            </div>

            <!-- Tier Personalities -->
            <div>
              <h4 class="text-xs font-bold uppercase tracking-wider text-amber-400 mb-3 flex items-center gap-1.5">
                <Trophy class="w-4 h-4" />
                Step 2 — Challenge Tier {{ selectedTier }} Opponents &amp; Boss
              </h4>
              <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
                <div
                  v-for="p in tierPersonalities"
                  :key="p.id"
                  :class="[
                    'p-4 rounded-xl border flex flex-col justify-between gap-3',
                    p.isBoss
                      ? 'bg-amber-950/25 border-amber-500/50'
                      : 'bg-zinc-900/70 border-zinc-800'
                  ]"
                >
                  <div class="flex items-start gap-3">
                    <div
                      class="w-11 h-11 rounded-xl flex items-center justify-center text-xl font-bold text-white shrink-0 shadow"
                      :style="{ backgroundColor: p.avatarColor }"
                    >
                      {{ p.avatarSymbol }}
                    </div>
                    <div class="min-w-0 flex-1">
                      <div class="flex items-center justify-between gap-2">
                        <div class="font-bold text-sm text-zinc-100 truncate">{{ p.name }}</div>
                        <span class="text-xs font-mono font-bold text-amber-300">{{ p.elo }}</span>
                      </div>
                      <div class="text-[11px] font-semibold text-zinc-400">
                        {{ p.title }} · <span class="text-emerald-400">{{ p.styleTag }}</span>
                      </div>
                      <p class="text-[11px] text-zinc-400 mt-1.5 leading-relaxed">{{ p.bio }}</p>
                    </div>
                  </div>

                  <div class="flex items-center justify-between pt-2 border-t border-zinc-800/80">
                    <span class="text-[11px] font-mono text-zinc-400">{{ recordString(p.id) }}</span>
                    <button
                      @click="launchMatch(p)"
                      :disabled="!unlockedSet.has(p.tier)"
                      :class="[
                        'px-3.5 py-1.5 rounded-lg text-xs font-bold flex items-center gap-1.5 transition-colors',
                        unlockedSet.has(p.tier)
                          ? (p.isBoss ? 'bg-amber-600 hover:bg-amber-500 text-white' : 'bg-emerald-600 hover:bg-emerald-500 text-white')
                          : 'bg-zinc-800 text-zinc-500 cursor-not-allowed'
                      ]"
                    >
                      <Play class="w-3.5 h-3.5" />
                      {{ p.isBoss ? 'Play Promotion Exam' : 'Challenge' }}
                    </button>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- TAB 2: INTERACTIVE LESSON RUNNER -->
        <div v-else-if="activeTab === 'lesson' && activeLesson && activeLessonStep" class="grid grid-cols-1 lg:grid-cols-12 gap-6 items-start">
          <div class="lg:col-span-6 flex flex-col items-center">
            <Chessboard
              :fen="activeLessonStep.fen"
              :legal-moves="lessonLegalMoves"
              :arrows="lessonBoardArrows"
              :piece-set="pieceSet || 'cburnett'"
              :board-theme="boardTheme"
              :board-size="440"
              :interactive="Boolean(activeLessonStep.expectedMoves && activeLessonStep.expectedMoves.length > 0)"
              @move="handleLessonBoardMove"
            />
          </div>
          <div class="lg:col-span-6 space-y-4">
            <div class="flex items-center justify-between">
              <span class="text-xs font-mono text-emerald-400">
                Step {{ lessonStepIdx + 1 }} of {{ activeLesson.steps.length }} · {{ activeLesson.title }}
              </span>
              <button
                @click="activeLesson = null; activeTab = 'career'"
                class="text-xs text-zinc-400 hover:text-zinc-200 underline"
              >
                Back to Tier Overview
              </button>
            </div>

            <div class="p-5 rounded-xl bg-zinc-900 border border-zinc-800 space-y-3">
              <h3 class="text-base font-bold text-zinc-100">{{ activeLessonStep.title }}</h3>
              <p class="text-sm text-zinc-300 leading-relaxed">{{ activeLessonStep.instruction }}</p>

              <div v-if="activeLessonStep.expectedMoves && activeLessonStep.expectedMoves.length > 0" class="pt-2">
                <div class="text-xs text-amber-300 font-semibold mb-2">
                  Play the correct move directly on the board to continue.
                </div>
                <button
                  v-if="activeLessonStep.hint && !showLessonHint"
                  @click="showLessonHint = true"
                  class="px-3 py-1.5 rounded-lg bg-zinc-800 hover:bg-zinc-700 text-xs text-zinc-200 flex items-center gap-1.5"
                >
                  <HelpCircle class="w-3.5 h-3.5 text-amber-400" />
                  Show Hint
                </button>
                <div v-if="showLessonHint && activeLessonStep.hint" class="p-3 rounded-lg bg-amber-950/40 border border-amber-800/60 text-xs text-amber-200">
                  Hint: {{ activeLessonStep.hint }}
                </div>
              </div>

              <div
                v-if="lessonFeedback"
                :class="[
                  'p-3 rounded-lg border text-xs font-medium',
                  lessonStepPassed
                    ? 'bg-emerald-950/60 border-emerald-700 text-emerald-200'
                    : 'bg-amber-950/60 border-amber-700 text-amber-200'
                ]"
              >
                {{ lessonFeedback }}
              </div>
            </div>

            <button
              @click="nextLessonStep"
              :disabled="!lessonStepPassed"
              :class="[
                'w-full py-2.5 px-4 rounded-xl font-bold text-sm flex items-center justify-center gap-2 transition-colors',
                lessonStepPassed
                  ? 'bg-emerald-600 hover:bg-emerald-500 text-white'
                  : 'bg-zinc-800 text-zinc-500 cursor-not-allowed'
              ]"
            >
              {{ lessonStepIdx + 1 < activeLesson.steps.length ? 'Next Step' : 'Complete Lesson & Save Progress' }}
              <ArrowRight class="w-4 h-4" />
            </button>
          </div>
        </div>

        <!-- TAB 3: BUILD OPPONENT FROM DATABASE (3-STEP GUIDED FLOW) -->
        <div v-else-if="activeTab === 'build-db'" class="max-w-3xl mx-auto space-y-5">
          <div class="p-4 rounded-xl bg-cyan-950/30 border border-cyan-800/50 space-y-1.5">
            <h3 class="text-sm font-bold text-cyan-300 flex items-center gap-2">
              <Database class="w-4 h-4" />
              How to Build a Playable Personality from Any Database
            </h3>
            <p class="text-xs text-zinc-300 leading-relaxed">
              NewScid scans every game played by your chosen player in the currently open database, extracts their exact White &amp; Black opening moves, estimates their Elo and aggression, and turns them into a playable opponent.
            </p>
          </div>

          <!-- Step 1: Open Database Status -->
          <div class="p-4 rounded-xl bg-zinc-900 border border-zinc-800 flex items-center justify-between gap-4">
            <div>
              <div class="text-[11px] uppercase font-bold text-zinc-400">Step 1 · Source Database</div>
              <div v-if="dbName" class="text-sm font-bold text-emerald-400 mt-0.5">
                Open: {{ dbName }} <span class="text-zinc-400 font-normal">({{ dbNumGames || 0 }} games)</span>
              </div>
              <div v-else class="text-sm font-semibold text-amber-300 mt-0.5">
                No database is currently open. Open a .si4 or .pgn file first!
              </div>
            </div>
            <button
              @click="emit('openDatabase')"
              class="px-3.5 py-2 rounded-lg bg-zinc-800 hover:bg-zinc-700 border border-zinc-700 text-xs font-semibold flex items-center gap-1.5"
            >
              <FolderOpen class="w-4 h-4 text-emerald-400" />
              {{ dbName ? 'Switch Database' : 'Open Database' }}
            </button>
          </div>

          <!-- Step 2: Search Player in Database -->
          <div class="p-4 rounded-xl bg-zinc-900 border border-zinc-800 space-y-3">
            <div class="text-[11px] uppercase font-bold text-zinc-400">Step 2 · Pick a Player from This Database</div>
            <div class="flex gap-2">
              <div class="relative flex-1">
                <Search class="w-4 h-4 text-zinc-500 absolute left-3 top-2.5" />
                <input
                  v-model="dbPlayerQuery"
                  type="text"
                  placeholder="Type player last name (e.g. Kasparov, Carlsen, your username)..."
                  class="w-full pl-9 pr-3 py-2 rounded-lg bg-zinc-950 border border-zinc-700 text-sm text-zinc-100 focus:outline-none focus:border-cyan-500"
                />
              </div>
              <button
                @click="handleBuildFromDb(dbPlayerQuery)"
                :disabled="!dbName || !dbPlayerQuery.trim() || buildingFromDb"
                class="px-4 py-2 rounded-lg bg-cyan-600 hover:bg-cyan-500 disabled:bg-zinc-800 disabled:text-zinc-500 text-white text-xs font-bold flex items-center gap-1.5"
              >
                <Sparkles class="w-4 h-4" />
                {{ buildingFromDb ? 'Building...' : 'Build Personality' }}
              </button>
            </div>

            <div v-if="dbSuggestions.length > 0" class="space-y-1.5">
              <div class="text-[11px] text-zinc-400">Top matching players in this database (click to build immediately):</div>
              <div class="grid grid-cols-1 sm:grid-cols-2 gap-2 max-h-56 overflow-y-auto pr-1">
                <button
                  v-for="s in dbSuggestions"
                  :key="s.name"
                  @click="dbPlayerQuery = s.name; handleBuildFromDb(s.name)"
                  class="p-2.5 rounded-lg bg-zinc-950 hover:bg-zinc-800 border border-zinc-800 hover:border-cyan-500/50 text-left flex items-center justify-between gap-2 transition-colors"
                >
                  <span class="text-xs font-semibold text-zinc-100 truncate">{{ s.name }}</span>
                  <span class="text-[11px] font-mono text-zinc-400 shrink-0">
                    {{ s.gameCount }}g{{ s.peakElo > 0 ? ` · ${s.peakElo}` : '' }}
                  </span>
                </button>
              </div>
            </div>
            <div v-else-if="dbName" class="text-xs text-zinc-500">
              No matching players found for "{{ dbPlayerQuery }}". Try clearing the filter to see the top players in this database.
            </div>
          </div>

          <!-- Step 3: Result Card -->
          <div v-if="lastBuiltPersonality" class="p-4 rounded-xl bg-emerald-950/35 border border-emerald-700/60 flex items-center justify-between gap-4">
            <div>
              <div class="text-xs font-bold uppercase text-emerald-400">Ready to Play!</div>
              <div class="text-base font-bold text-zinc-100 mt-0.5">
                {{ lastBuiltPersonality.name }} ({{ lastBuiltPersonality.elo }} Elo · {{ lastBuiltPersonality.styleTag }})
              </div>
              <p class="text-xs text-zinc-300 mt-1">{{ lastBuiltPersonality.bio }}</p>
            </div>
            <button
              @click="launchMatch(lastBuiltPersonality)"
              class="px-4 py-2 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white text-xs font-bold flex items-center gap-1.5 shrink-0"
            >
              <Play class="w-4 h-4" />
              Play Now
            </button>
          </div>
        </div>

        <!-- TAB 4: CUSTOM PERSONALITY EDITOR & SAVED ROSTER -->
        <div v-else-if="activeTab === 'editor'" class="grid grid-cols-1 lg:grid-cols-12 gap-6">
          <div class="lg:col-span-6 p-4 rounded-xl bg-zinc-900 border border-zinc-800 space-y-4">
            <h3 class="text-sm font-bold text-emerald-300">Create Custom Opponent</h3>
            <div class="space-y-3">
              <div>
                <label class="block text-xs text-zinc-400 mb-1">Opponent Name</label>
                <input
                  v-model="customName"
                  type="text"
                  placeholder="e.g. Friday Night Club Grinder"
                  class="w-full px-3 py-2 rounded-lg bg-zinc-950 border border-zinc-700 text-sm text-zinc-100"
                />
              </div>
              <div>
                <label class="block text-xs text-zinc-400 mb-1">Short Bio (Optional)</label>
                <input
                  v-model="customBio"
                  type="text"
                  placeholder="e.g. Plays sharp gambits and loves kingside attacks"
                  class="w-full px-3 py-2 rounded-lg bg-zinc-950 border border-zinc-700 text-sm text-zinc-100"
                />
              </div>
              <div class="grid grid-cols-2 gap-3 text-xs">
                <div>
                  <div class="flex justify-between mb-1"><span class="text-zinc-400">Target Elo</span><span class="font-mono text-amber-300">{{ customElo }}</span></div>
                  <input v-model.number="customElo" type="range" min="600" max="2600" step="50" class="w-full accent-amber-500" />
                </div>
                <div>
                  <div class="flex justify-between mb-1"><span class="text-zinc-400">Aggression</span><span class="font-mono text-red-300">{{ customAggression }}%</span></div>
                  <input v-model.number="customAggression" type="range" min="0" max="100" class="w-full accent-red-500" />
                </div>
                <div>
                  <div class="flex justify-between mb-1"><span class="text-zinc-400">Trade Tendency</span><span class="font-mono text-cyan-300">{{ customTrade }}%</span></div>
                  <input v-model.number="customTrade" type="range" min="0" max="100" class="w-full accent-cyan-500" />
                </div>
                <div>
                  <div class="flex justify-between mb-1"><span class="text-zinc-400">Positional Solidity</span><span class="font-mono text-emerald-300">{{ customSolidity }}%</span></div>
                  <input v-model.number="customSolidity" type="range" min="0" max="100" class="w-full accent-emerald-500" />
                </div>
                <div class="col-span-2">
                  <div class="flex justify-between mb-1"><span class="text-zinc-400">Human Blunder Rate</span><span class="font-mono text-purple-300">{{ customBlunder }}%</span></div>
                  <input v-model.number="customBlunder" type="range" min="0" max="90" class="w-full accent-purple-500" />
                </div>
              </div>
              <button
                @click="handleSaveCustom"
                class="w-full py-2 px-4 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white text-xs font-bold"
              >
                Save Custom Opponent
              </button>
            </div>
          </div>

          <div class="lg:col-span-6 space-y-3">
            <h3 class="text-sm font-bold text-zinc-200">Your Saved Custom &amp; Database Opponents</h3>
            <div v-if="customPersonalities.length === 0" class="p-6 rounded-xl bg-zinc-900/50 border border-dashed border-zinc-800 text-center text-xs text-zinc-400">
              No custom or database opponents yet. Build one from sliders on the left or switch to the "Build Opponent from Database" tab!
            </div>
            <div
              v-for="cp in customPersonalities"
              :key="cp.id"
              class="p-3.5 rounded-xl bg-zinc-900 border border-zinc-800 flex items-center justify-between gap-3"
            >
              <div class="min-w-0">
                <div class="text-sm font-bold text-zinc-100 flex items-center gap-2">
                  <span>{{ cp.name }}</span>
                  <span class="text-[10px] px-1.5 py-0.5 rounded bg-zinc-800 text-cyan-300 uppercase">{{ cp.source }}</span>
                  <span class="text-xs font-mono text-amber-300">{{ cp.elo }}</span>
                </div>
                <p class="text-[11px] text-zinc-400 mt-0.5 truncate">{{ cp.bio }}</p>
              </div>
              <div class="flex items-center gap-1.5 shrink-0">
                <button
                  @click="launchMatch(cp)"
                  class="px-3 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white text-xs font-bold flex items-center gap-1"
                >
                  <Play class="w-3.5 h-3.5" /> Play
                </button>
                <button
                  @click="handleDeleteCustom(cp.id)"
                  class="p-1.5 rounded-lg text-zinc-400 hover:text-red-400 hover:bg-zinc-800"
                  title="Delete Opponent"
                >
                  <Trash2 class="w-4 h-4" />
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
