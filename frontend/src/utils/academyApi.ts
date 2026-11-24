// Typed bridge to Go Academy, Career Ladder, Chess University, Spy Mode & Heatmap bindings.

export interface PersonalitySliders {
  aggression: number
  tradeTendency: number
  tacticalSharpness: number
  positionalSolidity: number
  blunderRate: number
}

export interface AcademyPersonality {
  id: string
  name: string
  title: string
  tier: number
  isBoss: boolean
  elo: number
  avatarColor: string
  avatarSymbol: string
  styleTag: string
  bio: string
  source: 'builtin' | 'custom' | 'database' | string
  sourcePlayer?: string
  sourceGames?: number
  sliders: PersonalitySliders
  openingBook?: Record<string, string[]>
  recommendedLessons?: string[]
}

export interface TierInfo {
  tier: number
  name: string
  ratingRange: string
  targetElo: number
  description: string
  bossId: string
}

export interface LessonArrow {
  from: string
  to: string
  color: string
}

export interface LessonStep {
  fen: string
  title: string
  instruction: string
  expectedMoves?: string[]
  hint?: string
  successFeedback?: string
  arrows?: LessonArrow[]
  highlights?: string[]
}

export interface AcademyLesson {
  id: string
  tier: number
  order: number
  title: string
  category: string
  estimatedMinutes: number
  summary: string
  steps: LessonStep[]
}

export interface OpponentRecord {
  wins: number
  draws: number
  losses: number
  lastPlayed?: string
}

export interface BossExamState {
  active: boolean
  gamesPlayed: number
  score: number
  passed: boolean
  nextColor: string
  lastFinished?: string
}

export interface CareerProfile {
  playerName: string
  rating: number
  peakRating: number
  currentTier: number
  unlockedTiers: number[]
  records: Record<string, OpponentRecord>
  completedLessons: Record<string, boolean>
  bossExams: Record<number, BossExamState>
  customPersonalities: AcademyPersonality[]
}

export interface CareerOverview {
  profile: CareerProfile
  tiers: TierInfo[]
  personalities: AcademyPersonality[]
  lessons: AcademyLesson[]
}

export interface GameResultUpdate {
  oldRating: number
  newRating: number
  ratingDelta: number
  promoted: boolean
  newTier: number
  examMessage?: string
  profile: CareerProfile
}

export interface DatabasePlayerSuggestion {
  name: string
  gameCount: number
  peakElo: number
}

export interface PersonalityMoveResult {
  uci: string
  san: string
  fromBook: boolean
  commentary: string
}

export interface SquareControlInfo {
  square: string
  whiteAttackers: number
  blackAttackers: number
  netControl: number
  piece?: string
  isHanging: boolean
}

export interface SpyThreatInfo {
  hasThreat: boolean
  from?: string
  to?: string
  moveUci?: string
  moveSan?: string
  reason?: string
  severity?: 'mate' | 'high' | 'medium' | string
}

export interface CoachReport {
  headline: string
  openingNote: string
  middlegameNote: string
  endgameNote: string
  keyMistakes: string[]
  strengths: string[]
  recommendedLessons: string[]
}

function goApp(): Record<string, any> | null {
  const w = typeof window !== 'undefined' ? (window as any) : null
  return w?.go?.main?.App || null
}

export async function getAcademyOverview(): Promise<CareerOverview | null> {
  const app = goApp()
  if (!app?.GetAcademyOverview) return null
  return await app.GetAcademyOverview()
}

export async function recordAcademyGameResult(personalityID: string, outcome: 'win' | 'draw' | 'loss'): Promise<GameResultUpdate | null> {
  const app = goApp()
  if (!app?.RecordAcademyGameResult) return null
  return await app.RecordAcademyGameResult(personalityID, outcome)
}

export async function completeAcademyLesson(lessonID: string): Promise<CareerProfile | null> {
  const app = goApp()
  if (!app?.CompleteAcademyLesson) return null
  return await app.CompleteAcademyLesson(lessonID)
}

export async function testOutAcademyTier(targetTier: number): Promise<CareerProfile | null> {
  const app = goApp()
  if (!app?.TestOutAcademyTier) return null
  return await app.TestOutAcademyTier(targetTier)
}

export async function saveCustomPersonality(p: AcademyPersonality): Promise<AcademyPersonality | null> {
  const app = goApp()
  if (!app?.SaveCustomPersonality) return null
  return await app.SaveCustomPersonality(p)
}

export async function deleteCustomPersonality(id: string): Promise<void> {
  const app = goApp()
  if (!app?.DeleteCustomPersonality) return
  await app.DeleteCustomPersonality(id)
}

export async function buildAcademyPersonalityFromDB(playerName: string): Promise<AcademyPersonality | null> {
  const app = goApp()
  if (!app?.BuildAcademyPersonalityFromDB) return null
  return await app.BuildAcademyPersonalityFromDB(playerName)
}

export async function searchDatabasePlayers(query: string, limit = 25): Promise<DatabasePlayerSuggestion[]> {
  const app = goApp()
  if (!app?.SearchDatabasePlayers) return []
  const res = await app.SearchDatabasePlayers(query, limit)
  return Array.isArray(res) ? res : []
}

export async function getAcademyPersonalityMove(personalityID: string, fen: string): Promise<PersonalityMoveResult | null> {
  const app = goApp()
  if (!app?.GetAcademyPersonalityMove) return null
  return await app.GetAcademyPersonalityMove(personalityID, fen)
}

export async function computeSquareControlHeatmap(fen: string): Promise<SquareControlInfo[]> {
  const app = goApp()
  if (!app?.ComputeSquareControlHeatmap) return []
  const res = await app.ComputeSquareControlHeatmap(fen)
  return Array.isArray(res) ? res : []
}

export async function computeSpyThreatOverlay(fen: string): Promise<SpyThreatInfo | null> {
  const app = goApp()
  if (!app?.ComputeSpyThreatOverlay) return null
  return await app.ComputeSpyThreatOverlay(fen)
}

export async function diagnoseGameForAcademyCoach(pgn: string, playerColor: 'white' | 'black'): Promise<CoachReport | null> {
  const app = goApp()
  if (!app?.DiagnoseGameForAcademyCoach) return null
  return await app.DiagnoseGameForAcademyCoach(pgn, playerColor)
}
