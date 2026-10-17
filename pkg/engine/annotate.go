package engine

import (
	"fmt"
	"math"

	"newscid/pkg/chess"
)

// MoveAnalysis contains analysis data for an individual ply.
type MoveAnalysis struct {
	Ply      int    `json:"ply"`
	SAN      string `json:"san"`
	UCI      string `json:"uci"`
	ScoreCp  int    `json:"scoreCp"` // from White's perspective
	MateIn   int    `json:"mateIn"`
	BestMove string `json:"bestMove"`
	DeltaCp  int    `json:"deltaCp"` // loss for the player who made the move
	NAG      int    `json:"nag"`
	Comment  string `json:"comment"`
}

// GameAnalysis encapsulates full-game annotation results.
type GameAnalysis struct {
	WhiteAccuracy float64        `json:"whiteAccuracy"`
	BlackAccuracy float64        `json:"blackAccuracy"`
	WhiteBlunders int            `json:"whiteBlunders"`
	WhiteMistakes int            `json:"whiteMistakes"`
	WhiteInaccs   int            `json:"whiteInaccs"`
	BlackBlunders int            `json:"blackBlunders"`
	BlackMistakes int            `json:"blackMistakes"`
	BlackInaccs   int            `json:"blackInaccs"`
	MoveAnalyses  []MoveAnalysis `json:"moveAnalyses"`
}

// AnnotateGame analyzes all moves in a game using a running UCI engine.
func AnnotateGame(game *chess.Game, eng *Engine, movetimeMs int, progressCb ...func(ply, total int, ma MoveAnalysis)) (*GameAnalysis, error) {
	if eng == nil {
		return nil, fmt.Errorf("engine not provided")
	}
	if game == nil {
		return nil, fmt.Errorf("game not provided")
	}

	analysis := &GameAnalysis{
		MoveAnalyses:  make([]MoveAnalysis, 0),
		WhiteAccuracy: 0.0,
		BlackAccuracy: 0.0,
	}

	if game.RootMove == nil {
		return analysis, nil
	}

	totalPlies := 0
	for countNode := game.RootMove; countNode != nil; countNode = countNode.Next {
		totalPlies++
	}

	board := chess.NewBoard()
	if game.StartFEN != "" && game.StartFEN != chess.StartingFEN {
		b, err := chess.ParseFEN(game.StartFEN)
		if err == nil {
			board = b
		}
	}

	var whiteLosses, blackLosses []float64

	curr := game.RootMove
	ply := 1

	// Evaluate initial position before ply 1
	bestMoveBefore, scoreBefore, _ := evalPositionWhitePOV(eng, board, movetimeMs)

	for curr != nil {
		isWhite := (board.Turn == chess.White)
		bestMove := bestMoveBefore
		if bestMove == "" {
			bestMove = curr.UCI
		}

		// Advance board with the played move
		board.MakeMove(curr.Move)

		// Evaluate position after the move (which also serves as before-eval for next ply)
		nextBestMove, scoreAfter, mateAfter := evalPositionWhitePOV(eng, board, movetimeMs)

		// Calculate centipawn loss from mover's perspective
		var loss int
		if curr.UCI != bestMove {
			if isWhite {
				// White wants White-perspective score to stay high
				loss = scoreBefore - scoreAfter
			} else {
				// Black wants White-perspective score to stay low
				loss = scoreAfter - scoreBefore
			}
		}
		if loss < 0 {
			loss = 0
		}
		if loss > 1000 {
			loss = 1000
		}

		if isWhite {
			whiteLosses = append(whiteLosses, float64(loss))
		} else {
			blackLosses = append(blackLosses, float64(loss))
		}

		// Classify move with NAG and comment
		var nag int
		var comment string

		if loss >= 200 {
			nag = 4 // ?? Blunder
			if isWhite {
				analysis.WhiteBlunders++
			} else {
				analysis.BlackBlunders++
			}
			comment = fmt.Sprintf("%.2f (blunder, best was %s)", float64(scoreAfter)/100.0, bestMove)
		} else if loss >= 100 {
			nag = 2 // ? Mistake
			if isWhite {
				analysis.WhiteMistakes++
			} else {
				analysis.BlackMistakes++
			}
			comment = fmt.Sprintf("%.2f (mistake, better was %s)", float64(scoreAfter)/100.0, bestMove)
		} else if loss >= 50 {
			nag = 6 // ?! Inaccuracy
			if isWhite {
				analysis.WhiteInaccs++
			} else {
				analysis.BlackInaccs++
			}
			comment = fmt.Sprintf("%.2f (inaccuracy, better was %s)", float64(scoreAfter)/100.0, bestMove)
		} else if curr.UCI == bestMove && (loss == 0) && math.Abs(float64(scoreAfter)) < 500 {
			nag = 1 // ! Good move
			comment = fmt.Sprintf("%.2f (best)", float64(scoreAfter)/100.0)
		} else {
			comment = fmt.Sprintf("%.2f", float64(scoreAfter)/100.0)
		}

		// Apply NAG and comment to game node
		if nag > 0 {
			curr.NAGs = []int{nag}
		}
		curr.Comment = comment

		ma := MoveAnalysis{
			Ply:      ply,
			SAN:      curr.SAN,
			UCI:      curr.UCI,
			ScoreCp:  scoreAfter,
			MateIn:   mateAfter,
			BestMove: bestMove,
			DeltaCp:  loss,
			NAG:      nag,
			Comment:  comment,
		}
		analysis.MoveAnalyses = append(analysis.MoveAnalyses, ma)

		if len(progressCb) > 0 && progressCb[0] != nil {
			progressCb[0](ply, totalPlies, ma)
		}

		bestMoveBefore = nextBestMove
		scoreBefore = scoreAfter

		curr = curr.Next
		ply++
	}

	// Calculate accuracy percentage (Lichess/Chess.com polynomial formula)
	analysis.WhiteAccuracy = calculateAccuracy(whiteLosses)
	analysis.BlackAccuracy = calculateAccuracy(blackLosses)

	// Update game tags
	if game.Tags == nil {
		game.Tags = make(map[string]string)
	}
	game.Tags["Annotator"] = "NewScid Analysis"
	game.Tags["WhiteAccuracy"] = fmt.Sprintf("%.1f%%", analysis.WhiteAccuracy)
	game.Tags["BlackAccuracy"] = fmt.Sprintf("%.1f%%", analysis.BlackAccuracy)

	return analysis, nil
}

func calculateAccuracy(losses []float64) float64 {
	if len(losses) == 0 {
		return 0.0
	}
	var total float64
	for _, l := range losses {
		total += l
	}
	acpl := total / float64(len(losses))

	// Standard formula: 103.1668 * exp(-0.04354 * acpl) - 3.1669
	acc := 103.1668*math.Exp(-0.04354*acpl) - 3.1669
	if acc > 100.0 {
		acc = 100.0
	}
	if acc < 0.0 {
		acc = 0.0
	}
	return math.Round(acc*10) / 10
}

func evalPositionWhitePOV(eng *Engine, b *chess.Board, movetimeMs int) (bestMove string, scoreCp int, mateIn int) {
	if len(b.GenerateLegalMoves()) == 0 {
		if b.InCheck(b.Turn) {
			if b.Turn == chess.White {
				return "", -3000, -1
			}
			return "", 3000, 1
		}
		return "", 0, 0
	}

	best, eval, err := eng.SearchBestMove(b.FEN(), nil, movetimeMs)
	if err != nil {
		return "", 0, 0
	}

	// SearchBestMove / ParseInfoLine already normalizes ScoreCp and MateIn to White's perspective
	scoreCp = eval.ScoreCp
	mateIn = eval.MateIn
	if mateIn > 0 {
		scoreCp = 3000 - (mateIn * 50)
	} else if mateIn < 0 {
		scoreCp = -3000 - (mateIn * 50)
	}
	return best, scoreCp, mateIn
}
