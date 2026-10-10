package imbalance

import (
	"math"
	"newscid/pkg/chess"
)

// PawnStructureStats holds pawn details for one side.
type PawnStructureStats struct {
	DoubledPawns  int `json:"doubledPawns"`
	IsolatedPawns int `json:"isolatedPawns"`
	PassedPawns   int `json:"passedPawns"`
	MaxPassedRank int `json:"maxPassedRank"` // highest rank reached by passed pawn
}

// SideEvaluation contains detailed positional metrics for one side.
type SideEvaluation struct {
	MaterialPoints int                `json:"materialPoints"`
	HasBishopPair  bool               `json:"hasBishopPair"`
	PawnStructure  PawnStructureStats `json:"pawnStructure"`
	KingSafety     int                `json:"kingSafety"`     // 0 - 100
	ControlledTerritory int           `json:"controlledTerritory"` // squares attacked in enemy half
	OpenFilesControlled int          `json:"openFilesControlled"`
}

// Report holds the comparative positional and material imbalance between White and Black.
type Report struct {
	FEN           string         `json:"fen"`
	White         SideEvaluation `json:"white"`
	Black         SideEvaluation `json:"black"`
	MaterialDelta int            `json:"materialDelta"` // White - Black (in pawns: Q=9, R=5, B=3, N=3, P=1)
	PositionalSummary string     `json:"positionalSummary"`
}

// AnalyzeBoard computes the imbalance report for a given chess board.
func AnalyzeBoard(board *chess.Board) *Report {
	if board == nil {
		return &Report{}
	}

	wEval := analyzeSide(board, chess.White)
	bEval := analyzeSide(board, chess.Black)

	delta := wEval.MaterialPoints - bEval.MaterialPoints

	summary := generateSummary(wEval, bEval, delta)

	return &Report{
		FEN:               board.FEN(),
		White:             wEval,
		Black:             bEval,
		MaterialDelta:     delta,
		PositionalSummary: summary,
	}
}

// analyzeSide evaluates a single side's positional characteristics.
func analyzeSide(board *chess.Board, color chess.Color) SideEvaluation {
	var eval SideEvaluation

	// 1. Material Points & Bishop Pair
	bishops := 0
	rooks := 0
	queens := 0
	knights := 0
	pawns := 0

	for sq := 0; sq < 64; sq++ {
		p := board.Squares[sq]
		if p.Color() == color {
			switch p.Type() {
			case chess.Pawn:
				pawns++
				eval.MaterialPoints += 1
			case chess.Knight:
				knights++
				eval.MaterialPoints += 3
			case chess.Bishop:
				bishops++
				eval.MaterialPoints += 3
			case chess.Rook:
				rooks++
				eval.MaterialPoints += 5
			case chess.Queen:
				queens++
				eval.MaterialPoints += 9
			}
		}
	}

	eval.HasBishopPair = (bishops >= 2)

	// 2. Pawn Structure Analysis
	eval.PawnStructure = analyzePawns(board, color)

	// 3. King Safety Shield
	eval.KingSafety = analyzeKingSafety(board, color)

	// 4. Territory Control & Open Files
	eval.ControlledTerritory = analyzeTerritory(board, color)
	eval.OpenFilesControlled = analyzeOpenFiles(board, color)

	return eval
}

func analyzePawns(board *chess.Board, color chess.Color) PawnStructureStats {
	var stats PawnStructureStats
	pawnsPerFile := make([]int, 8)
	pawnRanks := make([][]int, 8)

	for sq := 0; sq < 64; sq++ {
		p := board.Squares[sq]
		if p.Type() == chess.Pawn && p.Color() == color {
			f := sq % 8
			r := sq / 8
			pawnsPerFile[f]++
			pawnRanks[f] = append(pawnRanks[f], r)
		}
	}

	enemyColor := chess.Black
	if color == chess.Black {
		enemyColor = chess.White
	}

	// Doubled & Isolated
	for f := 0; f < 8; f++ {
		if pawnsPerFile[f] > 1 {
			stats.DoubledPawns += (pawnsPerFile[f] - 1)
		}

		if pawnsPerFile[f] > 0 {
			hasNeighbor := false
			if f > 0 && pawnsPerFile[f-1] > 0 {
				hasNeighbor = true
			}
			if f < 7 && pawnsPerFile[f+1] > 0 {
				hasNeighbor = true
			}
			if !hasNeighbor {
				stats.IsolatedPawns += pawnsPerFile[f]
			}
		}
	}

	// Passed Pawns
	for f := 0; f < 8; f++ {
		for _, r := range pawnRanks[f] {
			isPassed := true
			// Check own file and adjacent files ahead of the pawn for enemy pawns
			for enemyFile := math.Max(0, float64(f-1)); enemyFile <= math.Min(7, float64(f+1)); enemyFile++ {
				ef := int(enemyFile)
				for sq := 0; sq < 64; sq++ {
					p := board.Squares[sq]
					if p.Type() == chess.Pawn && p.Color() == enemyColor {
						sqFile := sq % 8
						sqRank := sq / 8
						if sqFile == ef {
							if color == chess.White && sqRank > r {
								isPassed = false
								break
							} else if color == chess.Black && sqRank < r {
								isPassed = false
								break
							}
						}
					}
				}
				if !isPassed {
					break
				}
			}

			if isPassed {
				stats.PassedPawns++
				effectiveRank := r + 1
				if color == chess.Black {
					effectiveRank = 8 - r
				}
				if effectiveRank > stats.MaxPassedRank {
					stats.MaxPassedRank = effectiveRank
				}
			}
		}
	}

	return stats
}

func analyzeKingSafety(board *chess.Board, color chess.Color) int {
	kingSq := -1
	for sq := 0; sq < 64; sq++ {
		p := board.Squares[sq]
		if p.Type() == chess.King && p.Color() == color {
			kingSq = sq
			break
		}
	}

	if kingSq == -1 {
		return 50
	}

	kf := kingSq % 8
	kr := kingSq / 8

	safety := 100

	// Deduct if king is on open file
	hasPawnOnFile := false
	for r := 0; r < 8; r++ {
		if board.Squares[r*8+kf].Type() == chess.Pawn {
			hasPawnOnFile = true
			break
		}
	}
	if !hasPawnOnFile {
		safety -= 25
	}

	// Shield pawns ahead of king
	shieldPawns := 0
	forwardRank := kr + 1
	if color == chess.Black {
		forwardRank = kr - 1
	}

	if forwardRank >= 0 && forwardRank < 8 {
		for f := math.Max(0, float64(kf-1)); f <= math.Min(7, float64(kf+1)); f++ {
			sq := forwardRank*8 + int(f)
			p := board.Squares[sq]
			if p.Type() == chess.Pawn && p.Color() == color {
				shieldPawns++
			}
		}
	}

	// If missing shield pawns in front of castled king
	if (kf <= 2 || kf >= 6) && shieldPawns < 2 {
		safety -= (2 - shieldPawns) * 18
	}

	if safety < 10 {
		safety = 10
	}
	if safety > 100 {
		safety = 100
	}
	return safety
}

func analyzeTerritory(board *chess.Board, color chess.Color) int {
	controlled := 0
	targetMinRank := 4 // ranks 5-8 (indices 4..7) for White
	targetMaxRank := 7
	if color == chess.Black {
		targetMinRank = 0 // ranks 1-4 (indices 0..3) for Black
		targetMaxRank = 3
	}

	for sq := targetMinRank * 8; sq <= (targetMaxRank*8 + 7); sq++ {
		if board.IsSquareAttacked(chess.Square(sq), color) {
			controlled++
		}
	}
	return controlled
}

func analyzeOpenFiles(board *chess.Board, color chess.Color) int {
	filesControlled := 0
	for f := 0; f < 8; f++ {
		// Is this file open (no pawns of either color)?
		isOpen := true
		for r := 0; r < 8; r++ {
			if board.Squares[r*8+f].Type() == chess.Pawn {
				isOpen = false
				break
			}
		}
		if isOpen {
			// Does this side have a rook or queen on it?
			for r := 0; r < 8; r++ {
				p := board.Squares[r*8+f]
				if p.Color() == color && (p.Type() == chess.Rook || p.Type() == chess.Queen) {
					filesControlled++
					break
				}
			}
		}
	}
	return filesControlled
}

func generateSummary(w, b SideEvaluation, delta int) string {
	if delta > 2 {
		return "White holds a decisive material advantage."
	} else if delta < -2 {
		return "Black holds a decisive material advantage."
	}

	if w.HasBishopPair && !b.HasBishopPair {
		return "White holds the bishop pair advantage on open diagonals."
	} else if b.HasBishopPair && !w.HasBishopPair {
		return "Black holds the bishop pair advantage on open diagonals."
	}

	if w.PawnStructure.PassedPawns > b.PawnStructure.PassedPawns {
		return "White controls dangerous passed pawn advances."
	} else if b.PawnStructure.PassedPawns > w.PawnStructure.PassedPawns {
		return "Black controls dangerous passed pawn advances."
	}

	if w.KingSafety > b.KingSafety+20 {
		return "White enjoys significantly greater king shelter safety."
	} else if b.KingSafety > w.KingSafety+20 {
		return "Black enjoys significantly greater king shelter safety."
	}

	if delta == 0 {
		return "Material is equal; dynamic play revolves around space and activity."
	}
	return "Slight material and positional tension balance."
}
