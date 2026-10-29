package imbalance

import (
	"newscid/pkg/chess"
)

var piecePointValues = [7]int{
	chess.None:   0,
	chess.King:   0,
	chess.Queen:  9,
	chess.Rook:   5,
	chess.Bishop: 3,
	chess.Knight: 3,
	chess.Pawn:   1,
}

type summaryReason uint8

const (
	summaryEqual summaryReason = iota
	summaryWhiteDecisiveMaterial
	summaryBlackDecisiveMaterial
	summaryWhiteMaterial
	summaryBlackMaterial
	summaryWhiteBishopPair
	summaryBlackBishopPair
	summaryWhitePassedPawns
	summaryBlackPassedPawns
	summaryWhiteKingSafety
	summaryBlackKingSafety
)

var summaryMessages = [11]string{
	summaryEqual:                 "Material is equal; dynamic play revolves around space and activity.",
	summaryWhiteDecisiveMaterial: "White holds a decisive material advantage.",
	summaryBlackDecisiveMaterial: "Black holds a decisive material advantage.",
	summaryWhiteMaterial:         "White holds a material advantage.",
	summaryBlackMaterial:         "Black holds a material advantage.",
	summaryWhiteBishopPair:       "White holds the bishop pair advantage on open diagonals.",
	summaryBlackBishopPair:       "Black holds the bishop pair advantage on open diagonals.",
	summaryWhitePassedPawns:      "White controls dangerous passed pawn advances.",
	summaryBlackPassedPawns:      "Black controls dangerous passed pawn advances.",
	summaryWhiteKingSafety:       "White enjoys significantly greater king shelter safety.",
	summaryBlackKingSafety:       "Black enjoys significantly greater king shelter safety.",
}

// PawnStructureStats holds pawn details for one side.
type PawnStructureStats struct {
	DoubledPawns  int `json:"doubledPawns"`
	IsolatedPawns int `json:"isolatedPawns"`
	PassedPawns   int `json:"passedPawns"`
	MaxPassedRank int `json:"maxPassedRank"` // highest rank reached by passed pawn
}

// SideEvaluation contains detailed positional metrics for one side.
type SideEvaluation struct {
	MaterialPoints      int                `json:"materialPoints"`
	HasBishopPair       bool               `json:"hasBishopPair"`
	PawnStructure       PawnStructureStats `json:"pawnStructure"`
	KingSafety          int                `json:"kingSafety"`          // 0 - 100
	ControlledTerritory int                `json:"controlledTerritory"` // squares attacked in enemy half
	OpenFilesControlled int                `json:"openFilesControlled"`
}

// Report holds the comparative positional and material imbalance between White and Black.
type Report struct {
	FEN               string         `json:"fen"`
	White             SideEvaluation `json:"white"`
	Black             SideEvaluation `json:"black"`
	MaterialDelta     int            `json:"materialDelta"`   // White - Black (in pawns: Q=9, R=5, B=3, N=3, P=1)
	EstimatedEvalCp   int            `json:"estimatedEvalCp"` // White - Black centipawn estimate combining material & positional factors
	PositionalSummary string         `json:"positionalSummary"`
}

// AnalyzeBoard computes the imbalance report for a given chess board.
func AnalyzeBoard(board *chess.Board) *Report {
	if board == nil {
		return &Report{}
	}

	wEval := analyzeSide(board, chess.White)
	bEval := analyzeSide(board, chess.Black)

	delta := wEval.MaterialPoints - bEval.MaterialPoints
	estCp := computeEstimatedEvalCp(wEval, bEval, delta)
	summary := generateSummary(wEval, bEval, delta)

	return &Report{
		FEN:               board.FEN(),
		White:             wEval,
		Black:             bEval,
		MaterialDelta:     delta,
		EstimatedEvalCp:   estCp,
		PositionalSummary: summary,
	}
}

// analyzeSide evaluates a single side's positional characteristics.
func analyzeSide(board *chess.Board, color chess.Color) SideEvaluation {
	var eval SideEvaluation

	// Material Points & Bishop Pair
	bishops := 0
	for sq := chess.A1; sq <= chess.H8; sq++ {
		p := board.Squares[sq]
		if p != chess.Empty && p.Color() == color {
			pt := p.Type()
			eval.MaterialPoints += piecePointValues[pt]
			if pt == chess.Bishop {
				bishops++
			}
		}
	}

	eval.HasBishopPair = (bishops >= 2)

	// Pawn Structure Analysis
	eval.PawnStructure = analyzePawns(board, color)

	// King Safety Shield
	eval.KingSafety = analyzeKingSafety(board, color)

	// Territory Control & Open Files
	eval.ControlledTerritory = analyzeTerritory(board, color)
	eval.OpenFilesControlled = analyzeOpenFiles(board, color)

	return eval
}

func analyzePawns(board *chess.Board, color chess.Color) PawnStructureStats {
	var stats PawnStructureStats
	var pawnsPerFile [8]int
	var ownPawnRanks [8]uint8
	var enemyPawnRanks [8]uint8
	enemyColor := color.Other()

	for sq := 0; sq < 64; sq++ {
		p := board.Squares[sq]
		if p.Type() == chess.Pawn {
			f := sq & 7
			r := sq >> 3
			if p.Color() == color {
				pawnsPerFile[f]++
				ownPawnRanks[f] |= 1 << uint(r)
			} else if p.Color() == enemyColor {
				enemyPawnRanks[f] |= 1 << uint(r)
			}
		}
	}

	// Doubled & Isolated
	for f := 0; f < 8; f++ {
		if pawnsPerFile[f] > 1 {
			stats.DoubledPawns += (pawnsPerFile[f] - 1)
		}

		if pawnsPerFile[f] > 0 {
			hasNeighbor := (f > 0 && pawnsPerFile[f-1] > 0) || (f < 7 && pawnsPerFile[f+1] > 0)
			if !hasNeighbor {
				stats.IsolatedPawns += pawnsPerFile[f]
			}
		}
	}

	// Passed Pawns using rank bitmasks
	for f := 0; f < 8; f++ {
		mask := ownPawnRanks[f]
		if mask == 0 {
			continue
		}
		minEF := f - 1
		if minEF < 0 {
			minEF = 0
		}
		maxEF := f + 1
		if maxEF > 7 {
			maxEF = 7
		}

		for r := 0; r < 8; r++ {
			if mask&(1<<uint(r)) == 0 {
				continue
			}
			var aheadMask uint8
			if color == chess.White {
				// ranks r+1..7
				aheadMask = ^uint8((1 << uint(r+1)) - 1)
			} else {
				// ranks 0..r-1
				aheadMask = uint8((1 << uint(r)) - 1)
			}

			isPassed := true
			for ef := minEF; ef <= maxEF; ef++ {
				if enemyPawnRanks[ef]&aheadMask != 0 {
					isPassed = false
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

	kf := kingSq & 7
	kr := kingSq >> 3

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
		minF := kf - 1
		if minF < 0 {
			minF = 0
		}
		maxF := kf + 1
		if maxF > 7 {
			maxF = 7
		}
		for f := minF; f <= maxF; f++ {
			sq := forwardRank*8 + f
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

func computeEstimatedEvalCp(w, b SideEvaluation, delta int) int {
	cp := delta * 100
	if w.HasBishopPair && !b.HasBishopPair {
		cp += 35
	} else if b.HasBishopPair && !w.HasBishopPair {
		cp -= 35
	}
	cp += (w.PawnStructure.PassedPawns - b.PawnStructure.PassedPawns) * 25
	cp -= (w.PawnStructure.IsolatedPawns - b.PawnStructure.IsolatedPawns) * 12
	cp -= (w.PawnStructure.DoubledPawns - b.PawnStructure.DoubledPawns) * 12
	cp += (w.KingSafety - b.KingSafety) / 2
	cp += (w.ControlledTerritory - b.ControlledTerritory) * 3
	return cp
}

func classifySummary(w, b SideEvaluation, delta int) summaryReason {
	switch {
	case delta > 2:
		return summaryWhiteDecisiveMaterial
	case delta < -2:
		return summaryBlackDecisiveMaterial
	case delta > 0:
		return summaryWhiteMaterial
	case delta < 0:
		return summaryBlackMaterial
	case w.HasBishopPair && !b.HasBishopPair:
		return summaryWhiteBishopPair
	case b.HasBishopPair && !w.HasBishopPair:
		return summaryBlackBishopPair
	case w.PawnStructure.PassedPawns > b.PawnStructure.PassedPawns:
		return summaryWhitePassedPawns
	case b.PawnStructure.PassedPawns > w.PawnStructure.PassedPawns:
		return summaryBlackPassedPawns
	case w.KingSafety > b.KingSafety+20:
		return summaryWhiteKingSafety
	case b.KingSafety > w.KingSafety+20:
		return summaryBlackKingSafety
	default:
		return summaryEqual
	}
}

func generateSummary(w, b SideEvaluation, delta int) string {
	return summaryMessages[classifySummary(w, b, delta)]
}
