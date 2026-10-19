<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import {
  X,
  Flame,
  Target,
  Activity,
  Award,
  Clock,
  CheckCircle,
  AlertTriangle,
  RotateCcw,
  ArrowRight,
  Lightbulb,
  Plus,
  Trash2,
  Play,
  FastForward,
  Sparkles,
  ShieldCheck,
  Search,
  TrendingUp,
  Layers,
} from '@lucide/vue'
import Chessboard, { type BoardArrow } from './Chessboard.vue'
import * as AppBindings from '../../wailsjs/go/main/App'
import { playCaptureSound, playMoveSound } from '../utils/sound'

export interface FeatureEntitlement {
  featureId: string
  unlocked: boolean
  enforceGate: boolean
  supporterActive: boolean
  bughouseOnlineSync: boolean
  tierRequired: string
  message: string
}

export interface LichessThemeInfo {
  key: string
  name: string
  category: string
  description: string
}

export interface CyclicalPuzzle {
  id: string
  sourceGameId?: number
  white: string
  black: string
  event: string
  date: string
  ply: number
  fen: string
  turn: string // "w" | "b"
  solutionUci: string[]
  solutionSan: string[]
  playedBlunderUci?: string
  themes: string[]
  primaryTheme: string
  difficultyTier: string // "Easy" | "Medium" | "Hard"
  difficultyOrder: number // 1..N (sorted Easy -> Hard)
  estimatedRating: number
  evalDeltaCp: number
  explanation: string
}

export interface PuzzleAttemptResult {
  puzzleId?: string
  solvedFirstTry: boolean
  attempts?: number
  timeTakenSec: number
  completedAt?: string
}

export interface ActiveCycleProgress {
  cycleNumber: number
  currentIndex: number
  elapsedSec: number
  firstTryCorrect: number
  totalAttempted: number
  results?: Record<string, PuzzleAttemptResult>
}

export interface CycleRun {
  cycleNumber: number
  completedAt: string
  totalPuzzles: number
  firstTryCorrect: number
  totalSolved: number
  accuracyPct: number
  totalTimeSec: number
  targetTimeSec: number
  avgSecPerPuzzle: number
  speedupVsCycle1: number
  easyAccuracyPct: number
  mediumAccuracyPct: number
  hardAccuracyPct: number
}

export interface CycleSet {
  id: string
  name: string
  playerName: string
  themeFilter: string[]
  createdAt: number
  currentCycle: number
  targetDays: number
  puzzles: CyclicalPuzzle[]
  cycles: CycleRun[]
  activeProgress?: ActiveCycleProgress | null
}

export interface ThemePerformanceStat {
  themeKey: string
  themeName: string
  category: string
  attempts: number
  firstWins: number
  fixed: number
  failed: number
  accuracyPct: number
  avgTimeSec: number
  ratingAvg: number
  performanceRating: number
  historyRatings?: number[]
}

export interface CyclicalDashboard {
  sets: CycleSet[]
  themeStats: ThemePerformanceStat[]
  radarAxes: ThemePerformanceStat[]
  strongThemes: ThemePerformanceStat[]
  weakThemes: ThemePerformanceStat[]
  globalPerformanceRating: number
  totalPuzzlesAttempted: number
  totalCyclesCompleted: number
  improvementSummary: string
  methodAttribution: string
}

interface PersonaItem {
  id: string
  displayName: string
  identifiers?: string[]
}

const props = withDefaults(defineProps<{
  boardTheme?: {
    light: string
    dark: string
    highlight: string
  }
  pieceSet?: string
  currentPieceSet?: string
  activeDbName?: string
  initialPlayer?: string
}>(), {
  boardTheme: () => ({
    light: '#e3c896',
    dark: '#ab7a4e',
    highlight: '#f7e597',
  }),
  pieceSet: 'cburnett',
  currentPieceSet: 'cburnett',
  activeDbName: '',
  initialPlayer: '',
})

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'load-into-app', fen: string): void
}>()

const effectivePieceSet = computed(() => props.pieceSet || props.currentPieceSet || 'cburnett')

// Safe Wails RPC bridge supporting both mocked imports in Vitest and live Wails runtime
const wailsMod = AppBindings as Record<string, any>
function callWails<T = any>(fnName: string, ...args: any[]): Promise<T> {
  if (typeof wailsMod[fnName] === 'function') {
    return wailsMod[fnName](...args)
  }
  const winApp = (window as any)?.go?.main?.App
  if (winApp && typeof winApp[fnName] === 'function') {
    return winApp[fnName](...args)
  }
  return Promise.resolve(undefined as unknown as T)
}

const DEFAULT_THEME_CATALOG: LichessThemeInfo[] = [
  { key: 'fork', name: 'Fork / Double Attack', category: 'Motifs', description: 'A single piece attacks two or more opponent targets simultaneously.' },
  { key: 'pin', name: 'Pin', category: 'Motifs', description: 'An attacked piece cannot move without exposing a more valuable piece behind it.' },
  { key: 'skewer', name: 'Skewer', category: 'Motifs', description: 'A high-value piece is attacked and forced to move, exposing a piece behind it.' },
  { key: 'discoveredAttack', name: 'Discovered Attack', category: 'Motifs', description: 'Moving one piece unmasks a line attack by another piece.' },
  { key: 'doubleCheck', name: 'Double Check', category: 'Motifs', description: 'Two pieces deliver check simultaneously, forcing a king move.' },
  { key: 'hangingPiece', name: 'Hanging Piece', category: 'Motifs', description: 'Tactical capture of an undefended or underdefended enemy piece.' },
  { key: 'trappedPiece', name: 'Trapped Piece', category: 'Motifs', description: 'An enemy piece has no safe escape squares and is won.' },
  { key: 'exposedKing', name: 'Exposed King', category: 'Motifs', description: 'Punishing a king stripped of its defensive pawn shield.' },
  { key: 'kingsideAttack', name: 'Kingside Attack', category: 'Motifs', description: 'Direct tactical assault against the castled kingside.' },
  { key: 'queensideAttack', name: 'Queenside Attack', category: 'Motifs', description: 'Tactical assault targeting the queenside.' },
  { key: 'attackingF2F7', name: 'Attacking f2/f7', category: 'Motifs', description: 'Targeting the vulnerable f2 or f7 square near the king.' },
  { key: 'sacrifice', name: 'Sacrifice', category: 'Advanced', description: 'Giving up material temporarily for a decisive tactical or mating continuation.' },
  { key: 'deflection', name: 'Deflection', category: 'Advanced', description: 'Luring a key defending piece away from a critical square or duty.' },
  { key: 'attraction', name: 'Attraction', category: 'Advanced', description: 'Decoying a king or high-value piece onto a vulnerable square.' },
  { key: 'clearance', name: 'Clearance', category: 'Advanced', description: 'Vacating a square or line so another friendly piece can use it.' },
  { key: 'intermezzo', name: 'Intermezzo / Zwischenzug', category: 'Advanced', description: 'Playing an intermediate forcing threat before an expected recapture.' },
  { key: 'quietMove', name: 'Quiet Move', category: 'Advanced', description: 'A non-capture, non-check move that prepares an unstoppable threat.' },
  { key: 'defensiveMove', name: 'Defensive Resource', category: 'Advanced', description: 'A precise defensive move that refutes the opponent threat.' },
  { key: 'advancedPawn', name: 'Advanced Pawn', category: 'Advanced', description: 'Exploiting a deep passed pawn on the 6th or 7th rank.' },
  { key: 'promotion', name: 'Pawn Promotion', category: 'Advanced', description: 'Forcing pawn promotion or underpromotion tactics.' },
  { key: 'xRayAttack', name: 'X-Ray Attack', category: 'Advanced', description: 'Indirectly defending or attacking a square through an enemy piece.' },
  { key: 'mateIn1', name: 'Mate in 1', category: 'Mates', description: 'Immediate one-move checkmate pattern.' },
  { key: 'mateIn2', name: 'Mate in 2', category: 'Mates', description: 'Two-move forced checkmate calculation.' },
  { key: 'mateIn3', name: 'Mate in 3+', category: 'Mates', description: 'Deep multi-move forced checkmate sequence.' },
  { key: 'backRankMate', name: 'Back Rank Mate', category: 'Mates', description: 'Checkmating a king trapped behind its own pawn wall on the back rank.' },
  { key: 'smotheredMate', name: 'Smothered Mate', category: 'Mates', description: 'Knight checkmate against a king boxed in by its own pieces.' },
  { key: 'hookMate', name: 'Hook Mate', category: 'Mates', description: 'Checkmate using a rook protected by a knight and pawn.' },
  { key: 'arabianMate', name: 'Arabian Mate', category: 'Mates', description: 'Corner checkmate executed by a rook and knight working together.' },
  { key: 'opening', name: 'Opening Phase', category: 'Phases', description: 'Tactical traps and punishments arising in the opening.' },
  { key: 'middlegame', name: 'Middlegame Phase', category: 'Phases', description: 'Complex middlegame tactical combinations.' },
  { key: 'endgame', name: 'Endgame Phase', category: 'Phases', description: 'Technical and tactical endgame calculation.' },
  { key: 'rookEndgame', name: 'Rook Endgame', category: 'Phases', description: 'Tactical motifs in rook and pawn endgames.' },
  { key: 'pawnEndgame', name: 'Pawn Endgame', category: 'Phases', description: 'King and pawn races, opposition, and breakthrough sacrifices.' },
]

// Navigation & Loading State
const activeTab = ref<'drill' | 'sets' | 'themes'>('drill')
const loading = ref(true)
const creatingSet = ref(false)
const errorMessage = ref<string | null>(null)

// Data from Backend
const dashboard = ref<CyclicalDashboard>({
  sets: [],
  themeStats: [],
  radarAxes: [],
  strongThemes: [],
  weakThemes: [],
  globalPerformanceRating: 1500,
  totalPuzzlesAttempted: 0,
  totalCyclesCompleted: 0,
  improvementSummary: '',
  methodAttribution:
    "Implements High-Frequency Cyclical Pattern Repetition—similar to the training method popularized by GM Axel Smith and IM Hans Tikkanen's 'The Woodpecker Method'—where you solve a fixed set of puzzles from your own game history ordered Easy → Medium → Hard across repeated cycles, halving your target completion time each cycle until tactical patterns become automatic reflexes.",
})
const themesCatalog = ref<LichessThemeInfo[]>(DEFAULT_THEME_CATALOG)
const personas = ref<PersonaItem[]>([])
const entitlement = ref<FeatureEntitlement>({
  featureId: 'cyclical_tactical_drill',
  unlocked: true,
  enforceGate: false,
  supporterActive: false,
  bughouseOnlineSync: true,
  tierRequired: 'Supporter',
  message: 'Included in NewScid Workstation (bughouse-online Supporter Ready)',
})

// Active CycleSet & Drill Board State
const selectedSetId = ref<string>('')
const currentPuzzleIndex = ref(0)
const moveStepIndex = ref(0)
const currentBoardFEN = ref('rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1')
const lastMove = ref<{ from: string; to: string } | null>(null)
const hintArrows = ref<BoardArrow[]>([])
const hintText = ref<string | null>(null)
const userFeedback = ref<'correct' | 'step-correct' | 'wrong' | null>(null)
const feedbackMessage = ref('')
const failedOnCurrentPuzzle = ref(false)
const puzzleSolved = ref(false)
const puzzleElapsedSec = ref(0)
const sessionElapsedSec = ref(0)
let timerHandle: ReturnType<typeof setInterval> | null = null

// Build New Cycle Set Form State
const newSetName = ref('My Game History Tactical Cycle')
const selectedPersonaOrPlayer = ref(props.initialPlayer || '')
const customPlayerName = ref(props.initialPlayer || '')
const selectedSetSize = ref<number>(20)
const selectedThemes = ref<string[]>([])

// Theme Table Filter State
const themeSearchQuery = ref('')
const selectedThemeCategory = ref<string>('All')

const activeSet = computed<CycleSet | null>(() => {
  if (!dashboard.value.sets || dashboard.value.sets.length === 0) return null
  const found = dashboard.value.sets.find(s => s.id === selectedSetId.value)
  return found || dashboard.value.sets[0] || null
})

const currentPuzzle = computed<CyclicalPuzzle | null>(() => {
  if (!activeSet.value || !activeSet.value.puzzles || activeSet.value.puzzles.length === 0) {
    return null
  }
  const idx = Math.max(0, Math.min(currentPuzzleIndex.value, activeSet.value.puzzles.length - 1))
  return activeSet.value.puzzles[idx] || null
})

const activeCycleNumber = computed(() => {
  return activeSet.value?.activeProgress?.cycleNumber || activeSet.value?.currentCycle || 1
})

const firstTryCorrectCount = computed(() => {
  return activeSet.value?.activeProgress?.firstTryCorrect ?? 0
})

const attemptedCount = computed(() => {
  return activeSet.value?.activeProgress?.totalAttempted ?? 0
})

const liveAccuracyPct = computed(() => {
  if (attemptedCount.value === 0) return 100
  return Math.round((firstTryCorrectCount.value / attemptedCount.value) * 100)
})

const totalCycleElapsedSec = computed(() => {
  const saved = activeSet.value?.activeProgress?.elapsedSec ?? 0
  return saved + sessionElapsedSec.value
})

const targetCycleSec = computed(() => {
  if (!activeSet.value) return 1800
  const cycles = activeSet.value.cycles || []
  if (cycles.length > 0) {
    const lastRun = cycles[cycles.length - 1]
    if (lastRun && lastRun.totalTimeSec > 0) {
      return Math.max(60, Math.floor(lastRun.totalTimeSec / 2))
    }
  }
  return Math.max(120, (activeSet.value.puzzles?.length || 20) * 45)
})

const cycleTimeProgressPct = computed(() => {
  if (targetCycleSec.value <= 0) return 0
  return Math.min(100, Math.round((totalCycleElapsedSec.value / targetCycleSec.value) * 100))
})

// Pure TypeScript FEN updater so board moves & opponent replies work immediately in both Wails and Vitest
function applyUciToFen(fen: string, uci: string): string {
  if (!fen || !uci || uci.length < 4) return fen
  const parts = fen.split(' ')
  const rows = (parts[0] || '').split('/')
  if (rows.length !== 8) return fen

  const grid: string[][] = []
  for (let r = 0; r < 8; r++) {
    const rowArr: string[] = []
    for (const ch of rows[r] || '') {
      if (ch >= '1' && ch <= '8') {
        const emptyCount = parseInt(ch, 10)
        for (let k = 0; k < emptyCount; k++) rowArr.push('')
      } else {
        rowArr.push(ch)
      }
    }
    while (rowArr.length < 8) rowArr.push('')
    grid.push(rowArr)
  }

  const fromFile = uci.charCodeAt(0) - 97
  const fromRank = 8 - parseInt(uci[1] || '1', 10)
  const toFile = uci.charCodeAt(2) - 97
  const toRank = 8 - parseInt(uci[3] || '1', 10)
  const promo = uci[4]

  if (fromFile < 0 || fromFile > 7 || fromRank < 0 || fromRank > 7) return fen
  if (toFile < 0 || toFile > 7 || toRank < 0 || toRank > 7) return fen

  let movingPiece = grid[fromRank]?.[fromFile] || 'P'
  grid[fromRank][fromFile] = ''

  if (promo) {
    const isWhite = movingPiece === movingPiece.toUpperCase()
    movingPiece = isWhite ? promo.toUpperCase() : promo.toLowerCase()
  }
  grid[toRank][toFile] = movingPiece

  const newRows = grid.map(row => {
    let str = ''
    let empty = 0
    for (const cell of row) {
      if (!cell) {
        empty++
      } else {
        if (empty > 0) {
          str += String(empty)
          empty = 0
        }
        str += cell
      }
    }
    if (empty > 0) str += String(empty)
    return str
  })

  const nextTurn = (parts[1] || 'w') === 'w' ? 'b' : 'w'
  const castling = parts[2] || '-'
  const ep = '-'
  const halfmove = parts[4] || '0'
  const fullmove = parts[5] || '1'
  return `${newRows.join('/')} ${nextTurn} ${castling} ${ep} ${halfmove} ${fullmove}`
}

function loadActivePuzzleState() {
  moveStepIndex.value = 0
  userFeedback.value = null
  feedbackMessage.value = ''
  failedOnCurrentPuzzle.value = false
  puzzleSolved.value = false
  hintArrows.value = []
  hintText.value = null
  puzzleElapsedSec.value = 0

  if (currentPuzzle.value) {
    currentBoardFEN.value = currentPuzzle.value.fen
    lastMove.value = null
  }
}

function syncActiveSetSelection(preferredSetId?: string) {
  const sets = dashboard.value.sets || []
  if (sets.length === 0) {
    selectedSetId.value = ''
    currentPuzzleIndex.value = 0
    return
  }

  const target = (preferredSetId && sets.find(s => s.id === preferredSetId)) || sets[0]
  if (target) {
    selectedSetId.value = target.id
    const savedIdx = target.activeProgress?.currentIndex ?? 0
    currentPuzzleIndex.value =
      savedIdx >= 0 && savedIdx < (target.puzzles?.length || 1) ? savedIdx : 0
    loadActivePuzzleState()
  }
}

async function fetchInitialData() {
  loading.value = true
  errorMessage.value = null
  try {
    const [dashRes, catalogRes, entRes, personasRes] = await Promise.all([
      callWails<CyclicalDashboard>('GetCyclicalDashboard'),
      callWails<LichessThemeInfo[]>('GetLichessThemesCatalog'),
      callWails<FeatureEntitlement>('CheckCyclicalEntitlement'),
      callWails<PersonaItem[]>('GetPersonas'),
    ])

    if (catalogRes && Array.isArray(catalogRes) && catalogRes.length > 0) {
      themesCatalog.value = catalogRes
    }
    if (entRes && entRes.featureId) {
      entitlement.value = entRes
    }
    if (personasRes && Array.isArray(personasRes)) {
      personas.value = personasRes
      if (!selectedPersonaOrPlayer.value && personasRes.length > 0) {
        selectedPersonaOrPlayer.value = personasRes[0].displayName
      }
    }
    if (dashRes) {
      dashboard.value = {
        ...dashboard.value,
        ...dashRes,
        sets: dashRes.sets || [],
        themeStats: dashRes.themeStats || [],
        radarAxes: dashRes.radarAxes || [],
        strongThemes: dashRes.strongThemes || [],
        weakThemes: dashRes.weakThemes || [],
      }
      syncActiveSetSelection(selectedSetId.value)
    }
  } catch (err) {
    errorMessage.value = `Failed to load cyclical drill dashboard: ${err}`
  } finally {
    loading.value = false
  }
}

watch(selectedSetId, (newId) => {
  const found = dashboard.value.sets.find(s => s.id === newId)
  if (found) {
    const savedIdx = found.activeProgress?.currentIndex ?? 0
    currentPuzzleIndex.value =
      savedIdx >= 0 && savedIdx < (found.puzzles?.length || 1) ? savedIdx : 0
    loadActivePuzzleState()
  }
})

async function handleBoardMove(uci: string) {
  if (!currentPuzzle.value || !activeSet.value || puzzleSolved.value) return

  const solution = currentPuzzle.value.solutionUci || []
  const expectedUci = solution[moveStepIndex.value]
  if (!expectedUci) return

  if (uci.trim().toLowerCase() === expectedUci.trim().toLowerCase()) {
    hintArrows.value = []
    hintText.value = null
    currentBoardFEN.value = applyUciToFen(currentBoardFEN.value, expectedUci)
    lastMove.value = { from: expectedUci.slice(0, 2), to: expectedUci.slice(2, 4) }
    moveStepIndex.value++
    playCaptureSound()

    // Check if there is an automatic opponent reply move next
    if (moveStepIndex.value < solution.length) {
      const opponentReplyUci = solution[moveStepIndex.value]
      if (opponentReplyUci) {
        userFeedback.value = 'step-correct'
        feedbackMessage.value = `Good move! Opponent replied ${opponentReplyUci} — find the follow-up continuation!`
        currentBoardFEN.value = applyUciToFen(currentBoardFEN.value, opponentReplyUci)
        lastMove.value = { from: opponentReplyUci.slice(0, 2), to: opponentReplyUci.slice(2, 4) }
        moveStepIndex.value++
      }
    }

    // Check if the entire puzzle solution is now complete
    if (moveStepIndex.value >= solution.length) {
      puzzleSolved.value = true
      userFeedback.value = 'correct'
      const solvedCleanly = !failedOnCurrentPuzzle.value
      const timeSpent = Math.max(1, puzzleElapsedSec.value)
      feedbackMessage.value = solvedCleanly
        ? `Solved cleanly on the 1st try in ${timeSpent}s!`
        : `Puzzle solved after retry (${timeSpent}s).`

      try {
        const updatedSet = await callWails<CycleSet>(
          'RecordCyclicalPuzzleAttempt',
          activeSet.value.id,
          currentPuzzle.value.id,
          solvedCleanly,
          timeSpent,
        )
        if (updatedSet && updatedSet.id) {
          const idx = dashboard.value.sets.findIndex(s => s.id === updatedSet.id)
          if (idx >= 0) {
            dashboard.value.sets[idx] = updatedSet
          }
        }
        const freshDash = await callWails<CyclicalDashboard>('GetCyclicalDashboard')
        if (freshDash && freshDash.sets) {
          dashboard.value = { ...dashboard.value, ...freshDash }
        }
      } catch (err) {
        console.error('Error recording cyclical puzzle attempt:', err)
      }
    }
  } else {
    failedOnCurrentPuzzle.value = true
    userFeedback.value = 'wrong'
    feedbackMessage.value = `Not the sharpest tactical continuation (${uci}). Look for checks, captures, or threats and try again!`
    playMoveSound()
  }
}

function triggerHint() {
  if (!currentPuzzle.value || puzzleSolved.value) return
  const uci = currentPuzzle.value.solutionUci?.[moveStepIndex.value]
  const san = currentPuzzle.value.solutionSan?.[moveStepIndex.value] || uci
  if (!uci || uci.length < 4) return

  const from = uci.slice(0, 2)
  const to = uci.slice(2, 4)
  hintArrows.value = [{ from, to, color: '#eab308' }]
  hintText.value = `Hint: Look at moving from ${from.toUpperCase()} (${san})`
}

function retryCurrentPuzzle() {
  loadActivePuzzleState()
}

function goToNextPuzzle() {
  if (!activeSet.value || !activeSet.value.puzzles.length) return
  if (currentPuzzleIndex.value < activeSet.value.puzzles.length - 1) {
    currentPuzzleIndex.value++
  } else {
    currentPuzzleIndex.value = 0
  }
  loadActivePuzzleState()
}

function selectSetForDrill(setId: string) {
  selectedSetId.value = setId
  const found = dashboard.value.sets.find(s => s.id === setId)
  if (found) {
    const savedIdx = found.activeProgress?.currentIndex ?? 0
    currentPuzzleIndex.value =
      savedIdx >= 0 && savedIdx < (found.puzzles?.length || 1) ? savedIdx : 0
  }
  loadActivePuzzleState()
  activeTab.value = 'drill'
}

async function handleStartNextCycle(setId: string) {
  try {
    const updatedSet = await callWails<CycleSet>('StartNextCyclicalCycle', setId)
    if (updatedSet && updatedSet.id) {
      const idx = dashboard.value.sets.findIndex(s => s.id === updatedSet.id)
      if (idx >= 0) {
        dashboard.value.sets[idx] = updatedSet
      }
      selectedSetId.value = updatedSet.id
      currentPuzzleIndex.value = 0
      loadActivePuzzleState()
      activeTab.value = 'drill'
    }
  } catch (err) {
    errorMessage.value = `Failed to start next cycle: ${err}`
  }
}

function toggleThemeFilter(themeKey: string) {
  const idx = selectedThemes.value.indexOf(themeKey)
  if (idx >= 0) {
    selectedThemes.value.splice(idx, 1)
  } else {
    selectedThemes.value.push(themeKey)
  }
}

function selectWeakestThemes() {
  const weakKeys = (dashboard.value.weakThemes || []).map(t => t.themeKey)
  if (weakKeys.length > 0) {
    selectedThemes.value = [...new Set(weakKeys)]
  } else {
    selectedThemes.value = ['fork', 'pin', 'backRankMate', 'quietMove']
  }
}

async function handleCreateCycleSet() {
  creatingSet.value = true
  errorMessage.value = null
  const effectivePlayer = (customPlayerName.value || selectedPersonaOrPlayer.value || '').trim()
  const name = (newSetName.value || 'Custom Tactical Cycle').trim()
  try {
    const created = await callWails<CycleSet>(
      'CreateCyclicalDrillSet',
      name,
      effectivePlayer,
      [...selectedThemes.value],
      selectedSetSize.value,
    )
    if (created && created.id) {
      const exists = dashboard.value.sets.some(s => s.id === created.id)
      if (!exists) {
        dashboard.value.sets.unshift(created)
      }
      selectedSetId.value = created.id
      currentPuzzleIndex.value = 0
      loadActivePuzzleState()
    }
    const freshDash = await callWails<CyclicalDashboard>('GetCyclicalDashboard')
    if (freshDash && freshDash.sets) {
      dashboard.value = { ...dashboard.value, ...freshDash }
      if (created?.id) {
        selectedSetId.value = created.id
      }
    }
    activeTab.value = 'drill'
  } catch (err) {
    errorMessage.value = `Could not build cycle set: ${err}`
  } finally {
    creatingSet.value = false
  }
}

async function handleDeleteCycleSet(setId: string) {
  try {
    await callWails('DeleteCyclicalDrillSet', setId)
    dashboard.value.sets = dashboard.value.sets.filter(s => s.id !== setId)
    if (selectedSetId.value === setId) {
      syncActiveSetSelection()
    }
  } catch (err) {
    errorMessage.value = `Could not delete cycle set: ${err}`
  }
}

async function buildThemeCycleSet(themeKey: string, themeName: string) {
  selectedThemes.value = [themeKey]
  newSetName.value = `${themeName} Mastery Cycle`
  await handleCreateCycleSet()
}

// Tier counts helper for any CycleSet
function getSetTierCounts(set: CycleSet | null) {
  if (!set || !set.puzzles) return { easy: 0, medium: 0, hard: 0 }
  let easy = 0
  let medium = 0
  let hard = 0
  for (const p of set.puzzles) {
    if (p.difficultyTier === 'Easy') easy++
    else if (p.difficultyTier === 'Medium') medium++
    else hard++
  }
  return { easy, medium, hard }
}

function getThemeName(themeKey: string): string {
  const found = themesCatalog.value.find(t => t.key === themeKey)
  return found ? found.name : themeKey
}

function formatDuration(totalSec: number): string {
  const sec = Math.max(0, Math.floor(totalSec || 0))
  const m = Math.floor(sec / 60)
  const s = sec % 60
  if (m >= 60) {
    const h = Math.floor(m / 60)
    const remM = m % 60
    return `${h}h ${remM}m ${s}s`
  }
  return `${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`
}

// 9-Axis SVG Spider / Radar Chart Geometry (Lichess PuzzleDashboard style)
const radarDisplayAxes = computed<ThemePerformanceStat[]>(() => {
  if (dashboard.value.radarAxes && dashboard.value.radarAxes.length >= 3) {
    return dashboard.value.radarAxes.slice(0, 9)
  }
  if (dashboard.value.themeStats && dashboard.value.themeStats.length >= 3) {
    return dashboard.value.themeStats.slice(0, 9)
  }
  // Fallback 9 core Lichess axes when user has not attempted puzzles yet
  return DEFAULT_THEME_CATALOG.slice(0, 9).map(t => ({
    themeKey: t.key,
    themeName: t.name,
    category: t.category,
    attempts: 0,
    firstWins: 0,
    fixed: 0,
    failed: 0,
    accuracyPct: 50,
    avgTimeSec: 0,
    ratingAvg: 1500,
    performanceRating: 1500,
  }))
})

const RADAR_CENTER_X = 190
const RADAR_CENTER_Y = 165
const RADAR_RADIUS = 112
const RADAR_MIN_RATING = 800
const RADAR_MAX_RATING = 2400

function getRadarPoint(index: number, total: number, rating: number) {
  const angle = (Math.PI * 2 * index) / total - Math.PI / 2
  const clamped = Math.max(RADAR_MIN_RATING, Math.min(RADAR_MAX_RATING, rating || 1500))
  const normalized = (clamped - RADAR_MIN_RATING) / (RADAR_MAX_RATING - RADAR_MIN_RATING)
  const r = Math.max(0.15, normalized) * RADAR_RADIUS
  return {
    x: Number((RADAR_CENTER_X + r * Math.cos(angle)).toFixed(1)),
    y: Number((RADAR_CENTER_Y + r * Math.sin(angle)).toFixed(1)),
    axisX: Number((RADAR_CENTER_X + RADAR_RADIUS * Math.cos(angle)).toFixed(1)),
    axisY: Number((RADAR_CENTER_Y + RADAR_RADIUS * Math.sin(angle)).toFixed(1)),
    labelX: Number((RADAR_CENTER_X + (RADAR_RADIUS + 34) * Math.cos(angle)).toFixed(1)),
    labelY: Number((RADAR_CENTER_Y + (RADAR_RADIUS + 24) * Math.sin(angle)).toFixed(1)),
  }
}

const radarPolygonPoints = computed(() => {
  const axes = radarDisplayAxes.value
  return axes
    .map((stat, idx) => {
      const pt = getRadarPoint(idx, axes.length, stat.performanceRating)
      return `${pt.x},${pt.y}`
    })
    .join(' ')
})

function getRadarRingPoints(scale: number): string {
  const axes = radarDisplayAxes.value
  const total = axes.length || 9
  const pts: string[] = []
  for (let i = 0; i < total; i++) {
    const angle = (Math.PI * 2 * i) / total - Math.PI / 2
    const x = (RADAR_CENTER_X + RADAR_RADIUS * scale * Math.cos(angle)).toFixed(1)
    const y = (RADAR_CENTER_Y + RADAR_RADIUS * scale * Math.sin(angle)).toFixed(1)
    pts.push(`${x},${y}`)
  }
  return pts.join(' ')
}

// Merged & filtered theme proficiency table rows
const filteredThemeRows = computed<ThemePerformanceStat[]>(() => {
  const statsMap = new Map<string, ThemePerformanceStat>()
  for (const s of dashboard.value.themeStats || []) {
    statsMap.set(s.themeKey, s)
  }

  const combined: ThemePerformanceStat[] = []
  for (const info of themesCatalog.value) {
    const existing = statsMap.get(info.key)
    if (existing) {
      combined.push(existing)
      statsMap.delete(info.key)
    } else {
      combined.push({
        themeKey: info.key,
        themeName: info.name,
        category: info.category,
        attempts: 0,
        firstWins: 0,
        fixed: 0,
        failed: 0,
        accuracyPct: 0,
        avgTimeSec: 0,
        ratingAvg: 1500,
        performanceRating: 1500,
      })
    }
  }
  for (const leftover of statsMap.values()) {
    combined.push(leftover)
  }

  const q = themeSearchQuery.value.trim().toLowerCase()
  return combined.filter(row => {
    if (selectedThemeCategory.value !== 'All' && row.category !== selectedThemeCategory.value) {
      return false
    }
    if (!q) return true
    return (
      row.themeName.toLowerCase().includes(q) ||
      row.themeKey.toLowerCase().includes(q) ||
      row.category.toLowerCase().includes(q)
    )
  })
})

onMounted(() => {
  fetchInitialData()
  timerHandle = setInterval(() => {
    if (activeTab.value === 'drill' && currentPuzzle.value && !puzzleSolved.value) {
      puzzleElapsedSec.value++
      sessionElapsedSec.value++
    }
  }, 1000)
})

onUnmounted(() => {
  if (timerHandle) clearInterval(timerHandle)
})
</script>

<template>
  <div
    class="fixed inset-0 bg-black/80 backdrop-blur-xs flex items-center justify-center z-50 p-4 animate-fade-in select-none"
    @click.self="emit('close')"
  >
    <div
      class="bg-zinc-900 border border-zinc-700/80 rounded-2xl w-full max-w-6xl shadow-2xl overflow-hidden flex flex-col max-h-[94vh] text-xs"
    >
      <!-- Top Header -->
      <div class="flex items-center justify-between px-5 py-3.5 bg-zinc-950 border-b border-zinc-800">
        <div class="flex items-center gap-3">
          <div class="w-9 h-9 rounded-xl bg-amber-500/20 border border-amber-500/40 flex items-center justify-center text-amber-400">
            <Flame class="w-5 h-5" />
          </div>
          <div>
            <div class="flex items-center gap-2.5 flex-wrap">
              <h2 data-testid="cyclical-modal-title" class="text-sm font-bold text-zinc-100">
                Cyclical Tactical Drill &amp; Theme Mastery
              </h2>
              <span
                data-testid="entitlement-badge"
                class="text-[10px] px-2.5 py-0.5 rounded-full bg-emerald-500/15 border border-emerald-500/30 text-emerald-300 font-medium flex items-center gap-1"
              >
                <ShieldCheck class="w-3 h-3" />
                {{ entitlement.message || 'bughouse-online Supporter Ready' }}
              </span>
            </div>
            <p class="text-[11px] text-zinc-400">
              Multi-Cycle Easy → Hard Pattern Repetition from Your Game History + Lichess Theme Proficiency Radar
            </p>
          </div>
        </div>

        <button
          data-testid="close-cyclical-btn"
          class="text-zinc-400 hover:text-white p-1.5 rounded-lg hover:bg-zinc-800 transition-colors cursor-pointer"
          @click="emit('close')"
        >
          <X class="w-5 h-5" />
        </button>
      </div>

      <!-- Legal Naming & Method Attribution Banner + Global KPI Summary -->
      <div
        data-testid="method-attribution-banner"
        class="px-5 py-2.5 bg-zinc-950/70 border-b border-zinc-800/80 flex flex-col lg:flex-row lg:items-center justify-between gap-3"
      >
        <p class="text-[11px] text-zinc-300 leading-relaxed max-w-3xl">
          <span class="font-bold text-amber-300">High-Frequency Cyclical Pattern Repetition:</span>
          Similar to the training method popularized by GM Axel Smith &amp; IM Hans Tikkanen's
          <em>The Woodpecker Method</em>—solve a fixed set of tactical puzzles extracted from your own game history
          ordered <strong class="text-emerald-300">Easy</strong> → <strong class="text-amber-300">Medium</strong> →
          <strong class="text-rose-300">Hard</strong> across repeated cycles, halving your target completion time each
          cycle until tactical patterns become automatic reflexes.
          <span class="text-zinc-500">
            (Independent open-source implementation; not affiliated with Quality Chess or Chessable.)
          </span>
        </p>

        <div class="flex items-center gap-3 shrink-0">
          <div class="px-3 py-1.5 rounded-xl bg-zinc-900 border border-zinc-800 text-center">
            <div class="text-[10px] text-zinc-500 uppercase">Theme Perf</div>
            <div data-testid="global-perf-rating" class="text-xs font-bold text-amber-400 font-mono">
              {{ dashboard.globalPerformanceRating }} Elo
            </div>
          </div>
          <div class="px-3 py-1.5 rounded-xl bg-zinc-900 border border-zinc-800 text-center">
            <div class="text-[10px] text-zinc-500 uppercase">Puzzles Drilled</div>
            <div data-testid="total-puzzles-attempted" class="text-xs font-bold text-zinc-200 font-mono">
              {{ dashboard.totalPuzzlesAttempted }}
            </div>
          </div>
          <div class="px-3 py-1.5 rounded-xl bg-zinc-900 border border-zinc-800 text-center">
            <div class="text-[10px] text-zinc-500 uppercase">Cycles Done</div>
            <div data-testid="total-cycles-completed" class="text-xs font-bold text-emerald-400 font-mono">
              {{ dashboard.totalCyclesCompleted }}
            </div>
          </div>
        </div>
      </div>

      <!-- Navigation Tabs -->
      <div class="flex items-center justify-between px-5 py-2 bg-zinc-950/40 border-b border-zinc-800 gap-2 flex-wrap">
        <div class="flex items-center gap-2">
          <button
            data-testid="tab-drill"
            class="px-3.5 py-1.5 rounded-xl font-semibold flex items-center gap-1.5 transition-colors cursor-pointer"
            :class="activeTab === 'drill'
              ? 'bg-amber-500/20 text-amber-300 border border-amber-500/40'
              : 'text-zinc-400 hover:text-zinc-200 hover:bg-zinc-800/60'"
            @click="activeTab = 'drill'"
          >
            <Play class="w-3.5 h-3.5" />
            Active Drill Board
            <span v-if="activeSet" class="text-[10px] px-1.5 py-0.2 rounded-full bg-zinc-800 text-zinc-300 font-mono">
              Cycle {{ activeCycleNumber }}
            </span>
          </button>

          <button
            data-testid="tab-sets"
            class="px-3.5 py-1.5 rounded-xl font-semibold flex items-center gap-1.5 transition-colors cursor-pointer"
            :class="activeTab === 'sets'
              ? 'bg-amber-500/20 text-amber-300 border border-amber-500/40'
              : 'text-zinc-400 hover:text-zinc-200 hover:bg-zinc-800/60'"
            @click="activeTab = 'sets'"
          >
            <Layers class="w-3.5 h-3.5" />
            Cycle Sets &amp; Improvement History
            <span class="text-[10px] px-1.5 py-0.2 rounded-full bg-zinc-800 text-zinc-300 font-mono">
              {{ dashboard.sets.length }}
            </span>
          </button>

          <button
            data-testid="tab-themes"
            class="px-3.5 py-1.5 rounded-xl font-semibold flex items-center gap-1.5 transition-colors cursor-pointer"
            :class="activeTab === 'themes'
              ? 'bg-amber-500/20 text-amber-300 border border-amber-500/40'
              : 'text-zinc-400 hover:text-zinc-200 hover:bg-zinc-800/60'"
            @click="activeTab = 'themes'"
          >
            <Target class="w-3.5 h-3.5" />
            Lichess Theme Radar &amp; Proficiency
          </button>
        </div>

        <!-- Active Set Switcher -->
        <div v-if="dashboard.sets.length > 0" class="flex items-center gap-2">
          <span class="text-[11px] text-zinc-400">Active Set:</span>
          <select
            v-model="selectedSetId"
            data-testid="active-set-select"
            class="bg-zinc-900 border border-zinc-700 rounded-lg px-2.5 py-1 text-xs text-zinc-200"
          >
            <option v-for="s in dashboard.sets" :key="s.id" :value="s.id">
              {{ s.name }} (Cycle {{ s.currentCycle }} • {{ s.puzzles?.length || 0 }} puzzles)
            </option>
          </select>
        </div>
      </div>

      <!-- Error Banner -->
      <div
        v-if="errorMessage"
        class="mx-5 mt-3 p-3 rounded-xl bg-rose-500/15 border border-rose-500/40 text-rose-300 flex items-center justify-between"
      >
        <span>{{ errorMessage }}</span>
        <button class="text-rose-200 hover:text-white cursor-pointer" @click="errorMessage = null">Dismiss</button>
      </div>

      <!-- Main Body -->
      <div class="p-5 flex-1 overflow-y-auto">
        <div v-if="loading" class="py-16 text-center text-zinc-400 animate-pulse">
          Loading cyclical tactical drill sets and Lichess theme radar...
        </div>

        <!-- =============================================================== -->
        <!-- TAB 1: ACTIVE DRILL BOARD (Easy -> Medium -> Hard Progression) -->
        <!-- =============================================================== -->
        <div v-else-if="activeTab === 'drill'">
          <div
            v-if="!activeSet || !currentPuzzle"
            class="py-12 flex flex-col items-center justify-center gap-3 text-center bg-zinc-950/50 rounded-2xl border border-zinc-800 p-6"
          >
            <Flame class="w-10 h-10 text-amber-400" />
            <h3 class="text-sm font-bold text-zinc-100">No Active Tactical Cycle Set Yet</h3>
            <p class="text-xs text-zinc-400 max-w-md">
              Build your first personalized cycle set from your game history and Lichess tactical themes to start
              training from Easy → Medium → Hard.
            </p>
            <button
              data-testid="empty-go-to-sets-btn"
              class="mt-2 px-4 py-2 rounded-xl bg-amber-500 hover:bg-amber-400 text-zinc-950 font-bold cursor-pointer"
              @click="activeTab = 'sets'"
            >
              Build Cycle Set from Game History
            </button>
          </div>

          <div v-else class="grid grid-cols-1 lg:grid-cols-12 gap-6 items-start">
            <!-- Left Column: Chessboard & Timers (7 cols) -->
            <div class="lg:col-span-6 flex flex-col items-center gap-3">
              <!-- Puzzle Difficulty & Source Game Header -->
              <div
                data-testid="puzzle-header"
                class="w-full bg-zinc-950 border border-zinc-800 rounded-xl p-3 flex flex-col gap-2"
              >
                <div class="flex items-center justify-between gap-2 flex-wrap">
                  <div class="flex items-center gap-2">
                    <span
                      data-testid="puzzle-difficulty-badge"
                      class="px-2.5 py-0.5 rounded-full text-[11px] font-bold border"
                      :class="{
                        'bg-emerald-500/20 text-emerald-300 border-emerald-500/40': currentPuzzle.difficultyTier === 'Easy',
                        'bg-amber-500/20 text-amber-300 border-amber-500/40': currentPuzzle.difficultyTier === 'Medium',
                        'bg-rose-500/20 text-rose-300 border-rose-500/40': currentPuzzle.difficultyTier === 'Hard',
                      }"
                    >
                      {{ currentPuzzle.difficultyTier }}
                    </span>
                    <span data-testid="puzzle-order-indicator" class="font-mono font-bold text-zinc-200">
                      #{{ currentPuzzle.difficultyOrder || currentPuzzleIndex + 1 }} / {{ activeSet.puzzles.length }}
                      <span class="text-zinc-500 font-normal">(Easy → Hard)</span>
                    </span>
                    <span class="text-[11px] font-mono text-amber-400">
                      Est. {{ currentPuzzle.estimatedRating }} Elo
                    </span>
                  </div>

                  <!-- Live Puzzle Timer -->
                  <div
                    data-testid="puzzle-timer"
                    class="flex items-center gap-1.5 px-2.5 py-1 rounded-lg bg-zinc-900 border border-zinc-800 font-mono text-zinc-200"
                  >
                    <Clock class="w-3.5 h-3.5 text-amber-400" />
                    <span>{{ formatDuration(puzzleElapsedSec) }}</span>
                  </div>
                </div>

                <div class="flex items-center justify-between text-[11px] text-zinc-400">
                  <span class="truncate font-medium text-zinc-300">
                    {{ currentPuzzle.white }} vs {{ currentPuzzle.black }}
                  </span>
                  <span class="truncate text-zinc-500">
                    {{ currentPuzzle.event }} ({{ currentPuzzle.date }})
                  </span>
                </div>
              </div>

              <!-- Interactive Chessboard -->
              <div class="w-80 h-80 sm:w-96 sm:h-96 flex items-center justify-center">
                <Chessboard
                  :fen="currentBoardFEN"
                  :board-theme="boardTheme"
                  :piece-set="effectivePieceSet"
                  :flipped="currentPuzzle.turn === 'b'"
                  :last-move="lastMove"
                  :arrows="hintArrows"
                  @move="handleBoardMove"
                />
              </div>

              <!-- Cycle Timer vs Target Halved Time Progress Bar -->
              <div
                data-testid="cycle-time-bar"
                class="w-full bg-zinc-950 border border-zinc-800 rounded-xl p-3 flex flex-col gap-1.5"
              >
                <div class="flex items-center justify-between text-[11px]">
                  <span class="text-zinc-400 font-medium">
                    Cycle {{ activeCycleNumber }} Elapsed vs. Halved Target Pace
                  </span>
                  <span class="font-mono text-zinc-200 font-bold">
                    {{ formatDuration(totalCycleElapsedSec) }} / {{ formatDuration(targetCycleSec) }}
                  </span>
                </div>
                <div class="w-full h-2 bg-zinc-800 rounded-full overflow-hidden">
                  <div
                    class="h-full transition-all duration-300"
                    :class="cycleTimeProgressPct <= 85 ? 'bg-emerald-500' : 'bg-amber-500'"
                    :style="{ width: `${cycleTimeProgressPct}%` }"
                  />
                </div>
              </div>
            </div>

            <!-- Right Column: Move Prompt, Theme Reveal & Cycle Scoreboard (6 cols) -->
            <div class="lg:col-span-6 flex flex-col gap-4">
              <!-- Move Prompt & Solution Card -->
              <div
                data-testid="move-prompt"
                class="bg-zinc-950 border border-zinc-800 rounded-2xl p-4 flex flex-col gap-3"
              >
                <div class="flex items-center justify-between">
                  <div class="flex items-center gap-2">
                    <span
                      class="w-3 h-3 rounded-full"
                      :class="currentPuzzle.turn === 'b' ? 'bg-zinc-900 border border-zinc-400' : 'bg-white'"
                    />
                    <span class="font-bold text-sm text-zinc-100">
                      Find the best tactical continuation for {{ currentPuzzle.turn === 'b' ? 'Black' : 'White' }}
                    </span>
                  </div>
                  <span class="text-[11px] font-mono text-zinc-400">
                    Move {{ Math.min(Math.floor(moveStepIndex / 2) + 1, Math.max(1, Math.ceil((currentPuzzle.solutionUci?.length || 1) / 2))) }}
                    of {{ Math.max(1, Math.ceil((currentPuzzle.solutionUci?.length || 1) / 2)) }}
                  </span>
                </div>

                <div
                  v-if="currentPuzzle.playedBlunderUci"
                  class="text-[11px] text-zinc-400 bg-zinc-900/80 px-3 py-1.5 rounded-lg border border-zinc-800"
                >
                  Original game move / blunder:
                  <span class="font-mono text-rose-400 font-bold">{{ currentPuzzle.playedBlunderUci }}</span>
                  <span v-if="currentPuzzle.evalDeltaCp" class="text-zinc-500 ml-1">
                    ({{ currentPuzzle.evalDeltaCp }} cp swing)
                  </span>
                </div>

                <!-- Hint Callout -->
                <div
                  v-if="hintText"
                  data-testid="hint-callout"
                  class="p-2.5 rounded-xl bg-amber-500/15 border border-amber-500/40 text-amber-300 flex items-center gap-2"
                >
                  <Lightbulb class="w-4 h-4 shrink-0" />
                  <span>{{ hintText }}</span>
                </div>

                <!-- Feedback Callout -->
                <div
                  v-if="userFeedback === 'wrong'"
                  data-testid="feedback-wrong"
                  class="p-3 rounded-xl bg-rose-500/15 border border-rose-500/40 text-rose-300 flex items-center gap-2"
                >
                  <AlertTriangle class="w-4 h-4 shrink-0" />
                  <span>{{ feedbackMessage }}</span>
                </div>

                <div
                  v-else-if="userFeedback === 'step-correct'"
                  data-testid="feedback-step"
                  class="p-3 rounded-xl bg-sky-500/15 border border-sky-500/40 text-sky-300 flex items-center gap-2"
                >
                  <CheckCircle class="w-4 h-4 shrink-0" />
                  <span>{{ feedbackMessage }}</span>
                </div>

                <div
                  v-else-if="userFeedback === 'correct'"
                  data-testid="feedback-correct"
                  class="p-3.5 rounded-xl bg-emerald-500/15 border border-emerald-500/40 text-emerald-200 flex flex-col gap-2"
                >
                  <div class="flex items-center justify-between">
                    <div class="flex items-center gap-2 font-bold text-emerald-300">
                      <CheckCircle class="w-4 h-4" />
                      <span>{{ feedbackMessage }}</span>
                    </div>
                    <span class="font-mono text-[11px] text-emerald-300">
                      {{ (currentPuzzle.solutionSan || currentPuzzle.solutionUci).join(' ') }}
                    </span>
                  </div>

                  <p data-testid="puzzle-explanation" class="text-zinc-300 text-[11px]">
                    {{ currentPuzzle.explanation }}
                  </p>

                  <!-- Revealed Lichess Theme Tags -->
                  <div class="flex items-center gap-1.5 flex-wrap pt-1">
                    <span class="text-[10px] text-zinc-400 uppercase font-semibold">Lichess Themes:</span>
                    <span
                      v-for="th in currentPuzzle.themes"
                      :key="th"
                      class="px-2 py-0.5 rounded-full bg-zinc-900 border border-emerald-500/30 text-emerald-300 text-[10px] font-mono"
                    >
                      {{ getThemeName(th) }}
                    </span>
                  </div>
                </div>

                <!-- Action Controls -->
                <div class="flex items-center gap-2 flex-wrap pt-1">
                  <button
                    data-testid="hint-btn"
                    :disabled="puzzleSolved"
                    class="px-3 py-2 rounded-xl bg-zinc-800 hover:bg-zinc-700 disabled:opacity-40 text-amber-300 font-semibold flex items-center gap-1.5 cursor-pointer"
                    @click="triggerHint"
                  >
                    <Lightbulb class="w-3.5 h-3.5" />
                    Hint
                  </button>

                  <button
                    data-testid="retry-puzzle-btn"
                    class="px-3 py-2 rounded-xl bg-zinc-800 hover:bg-zinc-700 text-zinc-200 font-semibold flex items-center gap-1.5 cursor-pointer"
                    @click="retryCurrentPuzzle"
                  >
                    <RotateCcw class="w-3.5 h-3.5" />
                    Retry Puzzle
                  </button>

                  <button
                    data-testid="next-puzzle-btn"
                    class="ml-auto px-4 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-500 text-white font-bold flex items-center gap-1.5 cursor-pointer shadow-lg"
                    @click="goToNextPuzzle"
                  >
                    Next Puzzle
                    <ArrowRight class="w-3.5 h-3.5" />
                  </button>
                </div>
              </div>

              <!-- Live Cycle Scoreboard & Easy -> Medium -> Hard Ramp -->
              <div
                data-testid="cycle-scoreboard"
                class="bg-zinc-950 border border-zinc-800 rounded-2xl p-4 flex flex-col gap-3"
              >
                <div class="flex items-center justify-between">
                  <div>
                    <h4 class="font-bold text-zinc-100 text-xs flex items-center gap-2">
                      <Award class="w-4 h-4 text-amber-400" />
                      {{ activeSet.name }} — Cycle {{ activeCycleNumber }} of 7
                    </h4>
                    <p class="text-[11px] text-zinc-400">
                      Each subsequent cycle halves your target time to build instant tactical intuition.
                    </p>
                  </div>
                  <button
                    data-testid="start-next-cycle-btn"
                    class="px-3 py-1.5 rounded-xl bg-amber-500/20 hover:bg-amber-500/30 border border-amber-500/40 text-amber-300 font-semibold flex items-center gap-1 cursor-pointer"
                    @click="handleStartNextCycle(activeSet.id)"
                  >
                    <FastForward class="w-3.5 h-3.5" />
                    Start Next Faster Cycle
                  </button>
                </div>

                <!-- Scoreboard KPI Grid -->
                <div class="grid grid-cols-3 gap-2.5">
                  <div class="p-2.5 rounded-xl bg-zinc-900 border border-zinc-800">
                    <div class="text-[10px] text-zinc-500 uppercase">Solved Cleanly (1st Try)</div>
                    <div data-testid="scoreboard-first-try" class="text-sm font-bold text-emerald-400 font-mono">
                      {{ firstTryCorrectCount }} / {{ attemptedCount }}
                    </div>
                  </div>
                  <div class="p-2.5 rounded-xl bg-zinc-900 border border-zinc-800">
                    <div class="text-[10px] text-zinc-500 uppercase">Cycle Accuracy</div>
                    <div data-testid="scoreboard-accuracy" class="text-sm font-bold text-amber-400 font-mono">
                      {{ liveAccuracyPct }}%
                    </div>
                  </div>
                  <div class="p-2.5 rounded-xl bg-zinc-900 border border-zinc-800">
                    <div class="text-[10px] text-zinc-500 uppercase">Completed Cycles</div>
                    <div class="text-sm font-bold text-zinc-200 font-mono">
                      {{ activeSet.cycles?.length || 0 }}
                    </div>
                  </div>
                </div>

                <!-- Easy -> Medium -> Hard Progression Selector Strip -->
                <div class="flex flex-col gap-1.5">
                  <div class="flex items-center justify-between text-[11px]">
                    <span class="text-zinc-400 font-semibold">Set Difficulty Ramp (Easy → Medium → Hard)</span>
                    <div class="flex items-center gap-2 text-[10px]">
                      <span class="text-emerald-400">Easy: {{ getSetTierCounts(activeSet).easy }}</span>
                      <span class="text-amber-400">Medium: {{ getSetTierCounts(activeSet).medium }}</span>
                      <span class="text-rose-400">Hard: {{ getSetTierCounts(activeSet).hard }}</span>
                    </div>
                  </div>
                  <div class="flex items-center gap-1 flex-wrap">
                    <button
                      v-for="(puz, pIdx) in activeSet.puzzles"
                      :key="puz.id"
                      :data-testid="'puzzle-pill-' + pIdx"
                      class="w-6 h-6 rounded-md font-mono text-[10px] font-bold flex items-center justify-center border transition-all cursor-pointer"
                      :class="[
                        pIdx === currentPuzzleIndex ? 'ring-2 ring-white scale-105' : 'opacity-80 hover:opacity-100',
                        puz.difficultyTier === 'Easy'
                          ? 'bg-emerald-500/20 border-emerald-500/50 text-emerald-300'
                          : puz.difficultyTier === 'Medium'
                            ? 'bg-amber-500/20 border-amber-500/50 text-amber-300'
                            : 'bg-rose-500/20 border-rose-500/50 text-rose-300',
                      ]"
                      @click="currentPuzzleIndex = pIdx; loadActivePuzzleState()"
                    >
                      {{ puz.difficultyOrder || pIdx + 1 }}
                    </button>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- =============================================================== -->
        <!-- TAB 2: CYCLE SETS & IMPROVEMENT HISTORY                         -->
        <!-- =============================================================== -->
        <div v-else-if="activeTab === 'sets'" class="flex flex-col gap-6">
          <!-- Create New Cycle Set Form -->
          <div
            data-testid="create-set-form"
            class="bg-zinc-950 border border-zinc-800 rounded-2xl p-5 flex flex-col gap-4"
          >
            <div class="flex items-center justify-between flex-wrap gap-2">
              <div>
                <h3 class="text-sm font-bold text-zinc-100 flex items-center gap-2">
                  <Sparkles class="w-4 h-4 text-amber-400" />
                  Build New Cyclical Set from Player Game History
                </h3>
                <p class="text-[11px] text-zinc-400">
                  Extracts missed tactics &amp; blunders from your games, classifies Lichess tactical motifs, and sorts
                  them strictly Easy → Medium → Hard.
                </p>
              </div>
            </div>

            <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
              <!-- Set Name -->
              <div class="flex flex-col gap-1">
                <label class="text-[11px] font-semibold text-zinc-300">Cycle Set Name</label>
                <input
                  v-model="newSetName"
                  data-testid="set-name-input"
                  type="text"
                  placeholder="e.g. My Autumn Blunders & Tactics"
                  class="bg-zinc-900 border border-zinc-700 rounded-xl px-3 py-2 text-xs text-zinc-100"
                />
              </div>

              <!-- Player / Persona Selector -->
              <div class="flex flex-col gap-1">
                <label class="text-[11px] font-semibold text-zinc-300">Player History / Persona</label>
                <div class="flex items-center gap-2">
                  <select
                    v-if="personas.length > 0"
                    v-model="selectedPersonaOrPlayer"
                    data-testid="player-select"
                    class="bg-zinc-900 border border-zinc-700 rounded-xl px-2.5 py-2 text-xs text-zinc-100"
                    @change="customPlayerName = selectedPersonaOrPlayer"
                  >
                    <option value="">All Active Database Games</option>
                    <option v-for="p in personas" :key="p.id" :value="p.displayName">
                      Persona: {{ p.displayName }}
                    </option>
                  </select>
                  <input
                    v-model="customPlayerName"
                    data-testid="player-name-input"
                    type="text"
                    placeholder="Player name (or leave blank for DB)"
                    class="flex-1 bg-zinc-900 border border-zinc-700 rounded-xl px-3 py-2 text-xs text-zinc-100"
                  />
                </div>
              </div>

              <!-- Set Size -->
              <div class="flex flex-col gap-1">
                <label class="text-[11px] font-semibold text-zinc-300">Cycle Set Size (Puzzles)</label>
                <div data-testid="set-size-select" class="flex items-center gap-1.5">
                  <button
                    v-for="sz in [10, 20, 32, 64, 128]"
                    :key="sz"
                    type="button"
                    :data-testid="'set-size-' + sz"
                    class="flex-1 py-2 rounded-xl font-mono font-bold border transition-colors cursor-pointer"
                    :class="selectedSetSize === sz
                      ? 'bg-amber-500/20 border-amber-500/50 text-amber-300'
                      : 'bg-zinc-900 border-zinc-800 text-zinc-400 hover:text-zinc-200'"
                    @click="selectedSetSize = sz"
                  >
                    {{ sz }}
                  </button>
                </div>
              </div>
            </div>

            <!-- Lichess Theme Filter Chips -->
            <div class="flex flex-col gap-2">
              <div class="flex items-center justify-between flex-wrap gap-2">
                <span class="text-[11px] font-semibold text-zinc-300">
                  Target Lichess Tactical Themes
                  <span class="text-zinc-500 font-normal">
                    ({{ selectedThemes.length === 0 ? 'All Themes Included' : `${selectedThemes.length} selected` }})
                  </span>
                </span>
                <div class="flex items-center gap-2">
                  <button
                    type="button"
                    data-testid="select-weak-themes-btn"
                    class="px-2.5 py-1 rounded-lg bg-rose-500/15 hover:bg-rose-500/25 border border-rose-500/30 text-rose-300 text-[11px] font-semibold cursor-pointer"
                    @click="selectWeakestThemes"
                  >
                    Select My Weakest Themes
                  </button>
                  <button
                    v-if="selectedThemes.length > 0"
                    type="button"
                    data-testid="clear-themes-btn"
                    class="px-2.5 py-1 rounded-lg bg-zinc-800 hover:bg-zinc-700 text-zinc-300 text-[11px] cursor-pointer"
                    @click="selectedThemes = []"
                  >
                    Clear Filter
                  </button>
                </div>
              </div>

              <div class="flex items-center gap-1.5 flex-wrap max-h-28 overflow-y-auto p-2 rounded-xl bg-zinc-900/70 border border-zinc-800">
                <button
                  v-for="th in themesCatalog"
                  :key="th.key"
                  type="button"
                  :data-testid="'theme-chip-' + th.key"
                  class="px-2.5 py-1 rounded-full text-[11px] font-medium border transition-colors cursor-pointer"
                  :class="selectedThemes.includes(th.key)
                    ? 'bg-amber-500/25 border-amber-500/60 text-amber-200'
                    : 'bg-zinc-900 border-zinc-800 text-zinc-400 hover:text-zinc-200'"
                  @click="toggleThemeFilter(th.key)"
                >
                  {{ th.name }}
                </button>
              </div>
            </div>

            <div class="flex justify-end">
              <button
                type="button"
                data-testid="create-set-submit-btn"
                :disabled="creatingSet"
                class="px-5 py-2.5 rounded-xl bg-amber-500 hover:bg-amber-400 disabled:opacity-50 text-zinc-950 font-bold flex items-center gap-2 cursor-pointer shadow-lg"
                @click="handleCreateCycleSet"
              >
                <Plus class="w-4 h-4" />
                {{ creatingSet ? 'Building Set...' : 'Build Easy → Hard Cycle Set' }}
              </button>
            </div>
          </div>

          <!-- Existing Cycle Sets & Multi-Cycle History of Improvement -->
          <div data-testid="cycle-sets-list" class="flex flex-col gap-4">
            <h3 class="text-sm font-bold text-zinc-100 flex items-center gap-2">
              <TrendingUp class="w-4 h-4 text-emerald-400" />
              Your Cycle Sets &amp; Multi-Cycle History of Improvement
            </h3>

            <div
              v-if="dashboard.sets.length === 0"
              class="p-8 rounded-2xl bg-zinc-950/60 border border-zinc-800 text-center text-zinc-400"
            >
              No cycle sets created yet. Use the builder above to generate your first personalized Easy → Hard tactical
              set!
            </div>

            <div
              v-for="set in dashboard.sets"
              :key="set.id"
              :data-testid="'cycle-set-card-' + set.id"
              class="bg-zinc-950 border border-zinc-800 rounded-2xl p-4 flex flex-col gap-4"
            >
              <div class="flex items-center justify-between flex-wrap gap-3">
                <div class="flex flex-col gap-1">
                  <div class="flex items-center gap-2.5 flex-wrap">
                    <h4 class="text-sm font-bold text-zinc-100">{{ set.name }}</h4>
                    <span class="px-2.5 py-0.5 rounded-full bg-amber-500/20 border border-amber-500/40 text-amber-300 font-mono text-[11px] font-bold">
                      Cycle {{ set.currentCycle }} / 7
                    </span>
                    <span class="text-[11px] text-zinc-400 font-mono">
                      {{ set.puzzles?.length || 0 }} puzzles ({{ getSetTierCounts(set).easy }} Easy • {{ getSetTierCounts(set).medium }} Medium • {{ getSetTierCounts(set).hard }} Hard)
                    </span>
                  </div>
                  <div class="flex items-center gap-2 text-[11px] text-zinc-400 flex-wrap">
                    <span>Player Source: <strong class="text-zinc-200">{{ set.playerName || 'Active Database' }}</strong></span>
                    <span v-if="set.themeFilter && set.themeFilter.length > 0" class="flex items-center gap-1">
                      • Themes:
                      <span
                        v-for="tk in set.themeFilter"
                        :key="tk"
                        class="px-2 py-0.2 rounded-full bg-zinc-900 border border-zinc-700 text-amber-300 font-mono text-[10px]"
                      >
                        {{ getThemeName(tk) }}
                      </span>
                    </span>
                  </div>
                </div>

                <div class="flex items-center gap-2">
                  <button
                    :data-testid="'start-set-btn-' + set.id"
                    class="px-3.5 py-1.5 rounded-xl bg-emerald-600 hover:bg-emerald-500 text-white font-bold flex items-center gap-1.5 cursor-pointer"
                    @click="selectSetForDrill(set.id)"
                  >
                    <Play class="w-3.5 h-3.5" />
                    Start / Resume Cycle
                  </button>
                  <button
                    :data-testid="'next-cycle-set-btn-' + set.id"
                    class="px-3 py-1.5 rounded-xl bg-amber-500/20 hover:bg-amber-500/30 border border-amber-500/40 text-amber-300 font-semibold flex items-center gap-1 cursor-pointer"
                    @click="handleStartNextCycle(set.id)"
                  >
                    <FastForward class="w-3.5 h-3.5" />
                    Next Faster Cycle
                  </button>
                  <button
                    :data-testid="'delete-set-btn-' + set.id"
                    class="p-1.5 rounded-xl bg-zinc-900 hover:bg-rose-500/20 text-zinc-400 hover:text-rose-300 border border-zinc-800 cursor-pointer"
                    @click="handleDeleteCycleSet(set.id)"
                  >
                    <Trash2 class="w-4 h-4" />
                  </button>
                </div>
              </div>

              <!-- History of Improvement Table across Cycles -->
              <div
                v-if="set.cycles && set.cycles.length > 0"
                :data-testid="'improvement-history-' + set.id"
                class="overflow-x-auto border border-zinc-800/80 rounded-xl"
              >
                <table class="w-full text-left border-collapse">
                  <thead>
                    <tr class="bg-zinc-900/90 text-[10px] text-zinc-400 uppercase border-b border-zinc-800">
                      <th class="py-2 px-3">Cycle</th>
                      <th class="py-2 px-3">Total Time</th>
                      <th class="py-2 px-3">Target Time</th>
                      <th class="py-2 px-3">Speedup vs Cycle 1</th>
                      <th class="py-2 px-3">1st-Try Accuracy</th>
                      <th class="py-2 px-3">Avg Sec/Puzzle</th>
                      <th class="py-2 px-3">Tier Accuracy (Easy / Med / Hard)</th>
                    </tr>
                  </thead>
                  <tbody class="divide-y divide-zinc-800/60 text-xs">
                    <tr v-for="run in set.cycles" :key="run.cycleNumber" class="hover:bg-zinc-900/40">
                      <td class="py-2 px-3 font-bold text-amber-300 font-mono">
                        Cycle {{ run.cycleNumber }}
                      </td>
                      <td class="py-2 px-3 font-mono text-zinc-100">
                        {{ formatDuration(run.totalTimeSec) }}
                      </td>
                      <td class="py-2 px-3 font-mono text-zinc-400">
                        {{ formatDuration(run.targetTimeSec) }}
                      </td>
                      <td class="py-2 px-3 font-mono">
                        <span
                          class="px-2 py-0.5 rounded-full font-bold text-[11px]"
                          :class="run.speedupVsCycle1 >= 1.5
                            ? 'bg-emerald-500/20 text-emerald-300'
                            : 'bg-zinc-800 text-zinc-300'"
                        >
                          {{ (run.speedupVsCycle1 || 1).toFixed(2) }}x faster
                        </span>
                      </td>
                      <td class="py-2 px-3 font-mono font-bold text-emerald-400">
                        {{ run.accuracyPct.toFixed(1) }}% ({{ run.firstTryCorrect }}/{{ run.totalPuzzles }})
                      </td>
                      <td class="py-2 px-3 font-mono text-zinc-300">
                        {{ run.avgSecPerPuzzle.toFixed(1) }}s
                      </td>
                      <td class="py-2 px-3 font-mono text-[11px]">
                        <span class="text-emerald-400">{{ run.easyAccuracyPct.toFixed(0) }}%</span> /
                        <span class="text-amber-400">{{ run.mediumAccuracyPct.toFixed(0) }}%</span> /
                        <span class="text-rose-400">{{ run.hardAccuracyPct.toFixed(0) }}%</span>
                      </td>
                    </tr>
                  </tbody>
                </table>
              </div>
              <div v-else class="text-[11px] text-zinc-500 italic">
                Complete Cycle 1 to establish your baseline time and unlock halved-time speedup tracking across Cycles 2–7.
              </div>
            </div>
          </div>
        </div>

        <!-- =============================================================== -->
        <!-- TAB 3: LICHESS THEME RADAR & PROFICIENCY                        -->
        <!-- =============================================================== -->
        <div v-else-if="activeTab === 'themes'" class="flex flex-col gap-6">
          <!-- Top Grid: 9-Axis SVG Spider Radar + Strengths & Weaknesses -->
          <div class="grid grid-cols-1 lg:grid-cols-12 gap-6 items-stretch">
            <!-- 9-Axis SVG Radar Chart Card (5 cols) -->
            <div class="lg:col-span-5 bg-zinc-950 border border-zinc-800 rounded-2xl p-4 flex flex-col items-center justify-between">
              <div class="w-full flex items-center justify-between mb-1">
                <div>
                  <h3 class="text-sm font-bold text-zinc-100 flex items-center gap-2">
                    <Activity class="w-4 h-4 text-emerald-400" />
                    Lichess 9-Axis Theme Radar
                  </h3>
                  <p class="text-[11px] text-zinc-400">
                    Performance Rating = RatingAvg − 500 + 1000 × (1stWins / Attempts)
                  </p>
                </div>
                <span class="px-2.5 py-1 rounded-lg bg-zinc-900 border border-zinc-800 font-mono font-bold text-amber-400">
                  {{ dashboard.globalPerformanceRating }} Elo
                </span>
              </div>

              <svg
                data-testid="lichess-radar-svg"
                viewBox="0 0 380 330"
                class="w-full max-w-[360px] h-auto select-none"
              >
                <!-- Concentric Radar Polygons -->
                <polygon
                  v-for="scale in [0.25, 0.5, 0.75, 1.0]"
                  :key="scale"
                  :points="getRadarRingPoints(scale)"
                  fill="none"
                  stroke="#27272a"
                  stroke-width="1"
                />

                <!-- Axis Spokes -->
                <line
                  v-for="(stat, idx) in radarDisplayAxes"
                  :key="'spoke-' + stat.themeKey"
                  :x1="RADAR_CENTER_X"
                  :y1="RADAR_CENTER_Y"
                  :x2="getRadarPoint(idx, radarDisplayAxes.length, RADAR_MAX_RATING).axisX"
                  :y2="getRadarPoint(idx, radarDisplayAxes.length, RADAR_MAX_RATING).axisY"
                  stroke="#3f3f46"
                  stroke-width="1"
                  stroke-dasharray="2,2"
                />

                <!-- Player Performance Polygon -->
                <polygon
                  data-testid="radar-performance-polygon"
                  :points="radarPolygonPoints"
                  fill="rgba(16, 185, 129, 0.25)"
                  stroke="#10b981"
                  stroke-width="2.2"
                />

                <!-- Vertex Dots & Axis Labels -->
                <g v-for="(stat, idx) in radarDisplayAxes" :key="'vertex-' + stat.themeKey">
                  <circle
                    :cx="getRadarPoint(idx, radarDisplayAxes.length, stat.performanceRating).x"
                    :cy="getRadarPoint(idx, radarDisplayAxes.length, stat.performanceRating).y"
                    r="4"
                    fill="#10b981"
                    stroke="#09090b"
                    stroke-width="1.5"
                  />
                  <text
                    :x="getRadarPoint(idx, radarDisplayAxes.length, stat.performanceRating).labelX"
                    :y="getRadarPoint(idx, radarDisplayAxes.length, stat.performanceRating).labelY"
                    text-anchor="middle"
                    fill="#e4e4e7"
                    font-size="9.5"
                    font-weight="600"
                  >
                    {{ stat.themeName }} ({{ stat.performanceRating }})
                  </text>
                </g>
              </svg>

              <p class="text-[11px] text-zinc-400 text-center">
                {{ dashboard.improvementSummary || 'Solve puzzles across themes to calibrate your 9-axis Lichess proficiency profile.' }}
              </p>
            </div>

            <!-- Side-by-Side Strengths & Weaknesses Panels (7 cols) -->
            <div class="lg:col-span-7 grid grid-cols-1 md:grid-cols-2 gap-4">
              <!-- Top Strengths -->
              <div
                data-testid="strong-themes-panel"
                class="bg-zinc-950 border border-emerald-500/30 rounded-2xl p-4 flex flex-col gap-3"
              >
                <div>
                  <h4 class="text-xs font-bold text-emerald-300 flex items-center gap-1.5">
                    <CheckCircle class="w-4 h-4" />
                    Top Tactical Strengths
                  </h4>
                  <p class="text-[11px] text-zinc-400">Highest Lichess theme performance ratings</p>
                </div>

                <div v-if="dashboard.strongThemes.length === 0" class="text-zinc-500 text-[11px] py-6 text-center">
                  Complete more themed puzzles to identify your strongest tactical motifs.
                </div>

                <div
                  v-for="stat in dashboard.strongThemes"
                  :key="stat.themeKey"
                  :data-testid="'strong-theme-card-' + stat.themeKey"
                  class="p-3 rounded-xl bg-zinc-900/90 border border-emerald-500/25 flex items-center justify-between gap-2"
                >
                  <div>
                    <div class="font-bold text-zinc-100">{{ stat.themeName }}</div>
                    <div class="text-[11px] text-zinc-400 font-mono">
                      <span class="text-emerald-400 font-bold">{{ stat.performanceRating }} Elo</span> •
                      {{ stat.accuracyPct.toFixed(0) }}% ({{ stat.firstWins }}/{{ stat.attempts }})
                    </div>
                  </div>
                  <button
                    :data-testid="'drill-strong-theme-' + stat.themeKey"
                    class="px-2.5 py-1.5 rounded-lg bg-emerald-500/20 hover:bg-emerald-500/30 border border-emerald-500/40 text-emerald-300 text-[11px] font-semibold cursor-pointer shrink-0"
                    @click="buildThemeCycleSet(stat.themeKey, stat.themeName)"
                  >
                    Drill Theme
                  </button>
                </div>
              </div>

              <!-- Top Weaknesses -->
              <div
                data-testid="weak-themes-panel"
                class="bg-zinc-950 border border-rose-500/30 rounded-2xl p-4 flex flex-col gap-3"
              >
                <div>
                  <h4 class="text-xs font-bold text-rose-300 flex items-center gap-1.5">
                    <AlertTriangle class="w-4 h-4" />
                    Priority Weaknesses to Drill
                  </h4>
                  <p class="text-[11px] text-zinc-400">1-Click build a targeted cyclical set for any weak motif</p>
                </div>

                <div v-if="dashboard.weakThemes.length === 0" class="text-zinc-500 text-[11px] py-6 text-center">
                  No major theme weaknesses flagged yet.
                </div>

                <div
                  v-for="stat in dashboard.weakThemes"
                  :key="stat.themeKey"
                  :data-testid="'weak-theme-card-' + stat.themeKey"
                  class="p-3 rounded-xl bg-zinc-900/90 border border-rose-500/25 flex items-center justify-between gap-2"
                >
                  <div>
                    <div class="font-bold text-zinc-100">{{ stat.themeName }}</div>
                    <div class="text-[11px] text-zinc-400 font-mono">
                      <span class="text-rose-400 font-bold">{{ stat.performanceRating }} Elo</span> •
                      {{ stat.accuracyPct.toFixed(0) }}% ({{ stat.failed }} missed)
                    </div>
                  </div>
                  <button
                    :data-testid="'drill-weak-theme-' + stat.themeKey"
                    class="px-2.5 py-1.5 rounded-lg bg-rose-500/20 hover:bg-rose-500/30 border border-rose-500/40 text-rose-200 text-[11px] font-semibold cursor-pointer shrink-0"
                    @click="buildThemeCycleSet(stat.themeKey, stat.themeName)"
                  >
                    Build Cycle Set for This Theme
                  </button>
                </div>
              </div>
            </div>
          </div>

          <!-- Full Searchable / Filterable Lichess Theme Table -->
          <div
            data-testid="theme-proficiency-table"
            class="bg-zinc-950 border border-zinc-800 rounded-2xl p-4 flex flex-col gap-3"
          >
            <div class="flex items-center justify-between flex-wrap gap-3">
              <div class="flex items-center gap-2">
                <h4 class="text-sm font-bold text-zinc-100">All Lichess Tactical &amp; Phase Themes</h4>
                <div class="flex items-center gap-1">
                  <button
                    v-for="cat in ['All', 'Motifs', 'Advanced', 'Mates', 'Phases']"
                    :key="cat"
                    class="px-2.5 py-1 rounded-lg text-[11px] font-semibold cursor-pointer"
                    :class="selectedThemeCategory === cat
                      ? 'bg-amber-500/20 text-amber-300 border border-amber-500/40'
                      : 'bg-zinc-900 text-zinc-400 hover:text-zinc-200'"
                    @click="selectedThemeCategory = cat"
                  >
                    {{ cat }}
                  </button>
                </div>
              </div>

              <div class="relative w-64">
                <Search class="w-3.5 h-3.5 text-zinc-500 absolute left-3 top-2.5" />
                <input
                  v-model="themeSearchQuery"
                  data-testid="theme-search-input"
                  type="text"
                  placeholder="Search Lichess themes..."
                  class="w-full bg-zinc-900 border border-zinc-700 rounded-xl pl-8 pr-3 py-1.5 text-xs text-zinc-100"
                />
              </div>
            </div>

            <div class="overflow-x-auto border border-zinc-800/80 rounded-xl">
              <table class="w-full text-left border-collapse">
                <thead>
                  <tr class="bg-zinc-900/90 text-[10px] text-zinc-400 uppercase border-b border-zinc-800">
                    <th class="py-2 px-3">Theme</th>
                    <th class="py-2 px-3">Category</th>
                    <th class="py-2 px-3">Performance Rating</th>
                    <th class="py-2 px-3">1st-Try Accuracy</th>
                    <th class="py-2 px-3">Solved / Failed</th>
                    <th class="py-2 px-3">Avg Time</th>
                    <th class="py-2 px-3 text-right">Action</th>
                  </tr>
                </thead>
                <tbody class="divide-y divide-zinc-800/60 text-xs">
                  <tr
                    v-for="row in filteredThemeRows"
                    :key="row.themeKey"
                    :data-testid="'theme-row-' + row.themeKey"
                    class="hover:bg-zinc-900/40"
                  >
                    <td class="py-2 px-3 font-bold text-zinc-100">
                      {{ row.themeName }}
                    </td>
                    <td class="py-2 px-3">
                      <span class="px-2 py-0.5 rounded-full bg-zinc-900 border border-zinc-700 text-zinc-300 text-[10px]">
                        {{ row.category }}
                      </span>
                    </td>
                    <td class="py-2 px-3 font-mono font-bold text-amber-400">
                      {{ row.performanceRating }} Elo
                    </td>
                    <td class="py-2 px-3 font-mono">
                      <span :class="row.accuracyPct >= 70 ? 'text-emerald-400' : row.attempts > 0 ? 'text-amber-300' : 'text-zinc-500'">
                        {{ row.attempts > 0 ? `${row.accuracyPct.toFixed(1)}%` : 'Unplayed' }}
                      </span>
                    </td>
                    <td class="py-2 px-3 font-mono text-zinc-300">
                      {{ row.firstWins }} solved / {{ row.failed }} failed
                    </td>
                    <td class="py-2 px-3 font-mono text-zinc-400">
                      {{ row.attempts > 0 ? `${row.avgTimeSec.toFixed(1)}s` : '—' }}
                    </td>
                    <td class="py-2 px-3 text-right">
                      <button
                        :data-testid="'drill-table-theme-' + row.themeKey"
                        class="px-2.5 py-1 rounded-lg bg-zinc-800 hover:bg-amber-500/20 hover:text-amber-300 border border-zinc-700 text-zinc-200 text-[11px] font-semibold cursor-pointer"
                        @click="buildThemeCycleSet(row.themeKey, row.themeName)"
                      >
                        Drill Theme
                      </button>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
