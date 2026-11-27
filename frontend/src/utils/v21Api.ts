// Typed bridge to NewScid v2.1 Go bindings:
// Material & Pawn-Structure Search, Endgame Sparring Trainer, and Lichess Study Importer.

export type MaterialPattern =
  | 'any'
  | 'opposite_bishops'
  | 'same_bishops'
  | 'queen_vs_rook_minor'
  | 'exchange_imbalance'
  | 'two_bishops'
  | 'iqp'
  | 'carlsbad'
  | 'maroczy_bind'
  | 'winning_sacrifice'

export interface PieceCountFilter {
  minQ: number
  maxQ: number
  minR: number
  maxR: number
  minB: number
  maxB: number
  minN: number
  maxN: number
  minP: number
  maxP: number
}

export interface MaterialSearchQuery {
  pattern: MaterialPattern
  white: PieceCountFilter
  black: PieceCountFilter
  usePieceCounts: boolean
  minPly: number
  maxPly: number
  minSacrificeCp: number
  maxResults: number
  maxGamesToScan: number
}

export interface MaterialSearchHit {
  gameId: number
  white: string
  black: string
  event: string
  date: string
  result: string
  eco: string
  matchPly: number
  matchFen: string
  summary: string
}

export interface EndgameDrill {
  id: string
  title: string
  category: 'pawn' | 'rook' | 'minor' | 'queen'
  difficulty: string
  objective: 'win' | 'draw'
  sideToPlay: 'white' | 'black'
  fen: string
  keyConcept: string
  hint: string
  firstMoveUci: string
  firstMoveSan: string
}

export interface EndgameEvalFeedback {
  fen: string
  status: 'optimal' | 'inaccuracy' | 'blunder' | 'won' | 'drawn' | 'lost'
  wdlText: string
  dtz: number
  tablebaseBacked: boolean
  lostObjective: boolean
  message: string
  bestReplyUci?: string
  bestReplySan?: string
}

export interface StudyChapterSummary {
  chapterIndex: number
  title: string
  white: string
  black: string
  fen: string
  plyCount: number
  pgn: string
  gameId?: number
}

export interface LichessStudyImportResult {
  studyId: string
  studyUrl: string
  totalChapters: number
  importedCount: number
  chapters: StudyChapterSummary[]
}

function getApp(): any {
  return (window as any)?.go?.main?.App
}

export function defaultPieceFilter(): PieceCountFilter {
  return {
    minQ: -1, maxQ: -1,
    minR: -1, maxR: -1,
    minB: -1, maxB: -1,
    minN: -1, maxN: -1,
    minP: -1, maxP: -1,
  }
}

export async function searchMaterialAndStructure(q: MaterialSearchQuery): Promise<MaterialSearchHit[]> {
  const app = getApp()
  if (!app?.SearchMaterialAndStructure) return []
  return await app.SearchMaterialAndStructure(q)
}

export async function getEndgameDrills(category = 'all'): Promise<EndgameDrill[]> {
  const app = getApp()
  if (!app?.GetEndgameDrills) return []
  return await app.GetEndgameDrills(category)
}

export async function evaluateEndgameMove(
  fen: string,
  objective: string,
  userSide: string,
  useTablebase = true
): Promise<EndgameEvalFeedback | null> {
  const app = getApp()
  if (!app?.EvaluateEndgameMove) return null
  return await app.EvaluateEndgameMove(fen, objective, userSide, useTablebase)
}

export async function importLichessStudy(
  studyURLOrID: string,
  token: string,
  importIntoDB: boolean
): Promise<LichessStudyImportResult> {
  const app = getApp()
  if (!app?.ImportLichessStudy) {
    throw new Error('Lichess Study importer requires the NewScid backend')
  }
  return await app.ImportLichessStudy(studyURLOrID, token, importIntoDB)
}
