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
func AnnotateGame(game *chess.Game, eng *Engine, movetimeMs int) (*GameAnalysis, error) {
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
	var prevScoreCp int

	for curr != nil {
		fenBefore := board.FEN()
		isWhite := (board.Turn == chess.White)

		// Search position before move to get best alternative and true evaluation
		bestMove, eval, err := eng.SearchBestMove(fenBefore, nil, movetimeMs)
		if err != nil {
			// fallback if search times out
			bestMove = curr.UCI
		}

		scoreCp := eval.ScoreCp
		if eval.MateIn > 0 {
			scoreCp = 3000 - (eval.MateIn * 50)
		} else if eval.MateIn < 0 {
			scoreCp = -3000 - (eval.MateIn * 50)
		}

		// Calculate centipawn loss from mover's perspective
		var loss int
		if ply > 1 {
			if isWhite {
				// White's turn: compare with previous position (White wants score to increase)
				loss = prevScoreCp - scoreCp
			} else {
				// Black's turn: compare with previous position (Black wants score to decrease)
				loss = scoreCp - prevScoreCp
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
			comment = fmt.Sprintf("%.2f (blunder, best was %s)", float64(scoreCp)/100.0, bestMove)
		} else if loss >= 100 {
			nag = 2 // ? Mistake
			if isWhite {
				analysis.WhiteMistakes++
			} else {
				analysis.BlackMistakes++
			}
			comment = fmt.Sprintf("%.2f (mistake, better was %s)", float64(scoreCp)/100.0, bestMove)
		} else if loss >= 50 {
			nag = 6 // ?! Inaccuracy
			if isWhite {
				analysis.WhiteInaccs++
			} else {
				analysis.BlackInaccs++
			}
			comment = fmt.Sprintf("%.2f (inaccuracy, better was %s)", float64(scoreCp)/100.0, bestMove)
		} else if curr.UCI == bestMove && (loss == 0) && math.Abs(float64(scoreCp)) < 500 {
			nag = 1 // ! Good move
			comment = fmt.Sprintf("%.2f (best)", float64(scoreCp)/100.0)
		} else {
			comment = fmt.Sprintf("%.2f", float64(scoreCp)/100.0)
		}

		// Apply NAG and comment to game node
		if nag > 0 {
			curr.NAGs = []int{nag}
		}
		curr.Comment = comment

		analysis.MoveAnalyses = append(analysis.MoveAnalyses, MoveAnalysis{
			Ply:      ply,
			SAN:      curr.SAN,
			UCI:      curr.UCI,
			ScoreCp:  scoreCp,
			MateIn:   eval.MateIn,
			BestMove: bestMove,
			DeltaCp:  loss,
			NAG:      nag,
			Comment:  comment,
		})

		prevScoreCp = scoreCp

		// Advance board
		board.MakeMove(curr.Move)
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
