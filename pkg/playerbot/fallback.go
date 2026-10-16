package playerbot

import (
	"fmt"
	"math/rand"
	"sort"

	"newscid/pkg/chess"
	"newscid/pkg/training"
)

// Piece values in centipawns for heuristic tactical evaluation
var pieceValues = map[chess.PieceType]int{
	chess.Pawn:   100,
	chess.Knight: 320,
	chess.Bishop: 330,
	chess.Rook:   500,
	chess.Queen:  900,
	chess.King:   20000,
}

// ScoredMove pairs a legal move with its tactical/heuristic score.
type ScoredMove struct {
	Move  chess.Move
	Score int
}

// SelectTacticalMove selects a strong, non-blundering move using tactical heuristics
// (SEE captures, checks, defense, center control, development) when no UCI engine is connected.
func SelectTacticalMove(b *chess.Board, isWhite bool) (chess.Move, error) {
	if b == nil {
		return chess.Move{}, fmt.Errorf("board cannot be nil")
	}

	legal := b.GenerateLegalMoves()
	if len(legal) == 0 {
		return chess.Move{}, fmt.Errorf("no legal moves available")
	}

	if len(legal) == 1 {
		return legal[0], nil
	}

	scored := make([]ScoredMove, 0, len(legal))

	for _, m := range legal {
		score := evaluateMoveTactics(b, m, isWhite)
		scored = append(scored, ScoredMove{Move: m, Score: score})
	}

	// Sort descending by score
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].Score > scored[j].Score
	})

	// Pick among the top moves (top 1-3 if within 30 centipawns) to avoid 100% rigidity
	topScore := scored[0].Score
	bestCandidates := make([]chess.Move, 0, 3)
	for _, sm := range scored {
		if sm.Score >= topScore-30 {
			bestCandidates = append(bestCandidates, sm.Move)
		} else {
			break
		}
	}

	chosen := bestCandidates[rand.Intn(len(bestCandidates))]
	return chosen, nil
}

func evaluateMoveTactics(b *chess.Board, m chess.Move, isWhite bool) int {
	movingPiece := b.Squares[m.From]
	targetPiece := b.Squares[m.To]
	pt := movingPiece.Type()
	score := 0

	// 1. Captures & Material evaluation via Static Exchange Evaluation (SEE)
	if targetPiece != chess.Empty || m.To == b.EPSquare {
		see := training.CalculateSEE(b, m.From, m.To)
		victimVal := pieceValues[targetPiece.Type()]
		if m.To == b.EPSquare {
			victimVal = pieceValues[chess.Pawn]
		}
		attackerVal := pieceValues[pt]

		if victimVal >= attackerVal {
			// Winning or equal trade
			score += victimVal*10 + (victimVal - attackerVal)*5
		} else {
			score += see * 5
		}
	}

	// 2. Promotion bonus
	if m.Promo != chess.None {
		score += pieceValues[m.Promo] * 8
	}

	// 3. Make move on cloned board to check resulting tactical state
	nextB := b.Clone()
	nextB.MakeMove(m)

	// Checkmate check: next player has no legal moves and is in check
	nextLegal := nextB.GenerateLegalMoves()
	inCheck := nextB.InCheck(nextB.Turn)
	if len(nextLegal) == 0 && inCheck {
		return 1000000 // Checkmate!
	}

	// Giving check bonus
	if inCheck {
		score += 65
	}

	// Self piece safety: check if the destination square is attacked and underdefended
	defColor := "white"
	if !isWhite {
		defColor = "black"
	}
	targetSquare := m.To.String()
	if safetyReport, err := training.AnalyzeBoardSafety(nextB.FEN()); err == nil && safetyReport != nil {
		for _, ps := range safetyReport.Pieces {
			if ps.Square == targetSquare && ps.Color == defColor {
				if ps.Status == training.StatusHanging {
					score -= pieceValues[pt] * 12 // Severe penalty for hanging moved piece!
				} else if ps.Status == training.StatusUnderdefended {
					score -= pieceValues[pt] * 6
				}
			}
		}
	}

	// 4. Center control (e4, d4, e5, d5, c4, f4, c5, f5)
	toFile := m.To.File()
	toRank := m.To.Rank()
	if (toFile == 3 || toFile == 4) && (toRank == 3 || toRank == 4) {
		score += 35 // Sweet center
	} else if (toFile >= 2 && toFile <= 5) && (toRank >= 2 && toRank <= 5) {
		score += 15 // Extended center
	}

	// 5. Castling & King safety
	if pt == chess.King {
		fromFile := m.From.File()
		if fromFile == 4 && (toFile == 6 || toFile == 2) {
			score += 90 // Castling is heavily favored
		}
	}

	// 6. Minor piece development in opening
	if pt == chess.Knight || pt == chess.Bishop {
		if isWhite && m.From.Rank() == 0 {
			score += 30 // Moving off back rank
		} else if !isWhite && m.From.Rank() == 7 {
			score += 30
		}
	}

	return score
}
