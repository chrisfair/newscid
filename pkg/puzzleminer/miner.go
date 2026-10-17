package puzzleminer

import (
	"fmt"
	"math"
	"strings"

	"newscid/pkg/chess"
	"newscid/pkg/database/scid"
	"newscid/pkg/engine"
)

// MinedPuzzle represents a tactical puzzle extracted from database games.
type MinedPuzzle struct {
	ID          string   `json:"id"`
	GameID      int      `json:"gameId"`
	White       string   `json:"white"`
	Black       string   `json:"black"`
	Date        string   `json:"date"`
	Event       string   `json:"event"`
	Ply         int      `json:"ply"`
	StartFEN    string   `json:"startFen"`
	Turn        string   `json:"turn"` // "w" or "b"
	BestMoveUCI string   `json:"bestMoveUci"`
	BestMoveSAN string   `json:"bestMoveSan"`
	Solution    []string `json:"solution"` // Solution UCI moves
	Theme       string   `json:"theme"`    // e.g. "Checkmate", "Queen Trap", "Blunder Punishment", "Material Gain"
	EvalDeltaCp int      `json:"evalDeltaCp"`
	Description string   `json:"description"`
}

// MiningFilter specifies criteria for extracting puzzles from games.
type MiningFilter struct {
	PlayerName   string // If set, only games containing this player
	MaxGames     int    // Max games to scan (e.g. 50-200)
	MinDeltaCp   int    // Minimum centipawn advantage swing (e.g. 250)
	OnlyCheckmate bool
}

// MineFromGames extracts tactical puzzles from a slice of games.
func MineFromGames(games []*chess.Game, eng *engine.Engine, filter MiningFilter) []MinedPuzzle {
	var puzzles []MinedPuzzle

	maxGames := filter.MaxGames
	if maxGames <= 0 || maxGames > len(games) {
		maxGames = len(games)
	}

	for i := 0; i < maxGames; i++ {
		g := games[i]
		if g == nil {
			continue
		}

		// Filter by player name if specified
		if filter.PlayerName != "" {
			nameLower := strings.ToLower(filter.PlayerName)
			whiteLower := strings.ToLower(g.Tag("White"))
			blackLower := strings.ToLower(g.Tag("Black"))
			if !strings.Contains(whiteLower, nameLower) && !strings.Contains(blackLower, nameLower) {
				continue
			}
		}

		gamePuzzles := mineGame(g, eng, filter)
		puzzles = append(puzzles, gamePuzzles...)
		if len(puzzles) >= 50 {
			break
		}
	}

	return puzzles
}

// MineFromDatabase scans games in an open SCID database and mines puzzles.
func MineFromDatabase(db *scid.Database, eng *engine.Engine, filter MiningFilter) ([]MinedPuzzle, error) {
	if db == nil || !db.IsOpen {
		return nil, fmt.Errorf("database is not open")
	}

	numGames := db.NumGames()
	limit := filter.MaxGames
	if limit <= 0 || limit > numGames {
		limit = numGames
	}
	if limit > 200 {
		limit = 200 // reasonable limit for responsiveness
	}

	var sampledGames []*chess.Game
	for i := 1; i <= limit; i++ {
		g, err := db.GetGame(i)
		if err == nil {
			g.ID = i
			sampledGames = append(sampledGames, g)
		}
	}

	return MineFromGames(sampledGames, eng, filter), nil
}

func mineGame(g *chess.Game, eng *engine.Engine, filter MiningFilter) []MinedPuzzle {
	var results []MinedPuzzle
	b := chess.NewBoard()
	if g.StartFEN != "" && g.StartFEN != chess.StartingFEN {
		if parsed, err := chess.ParseFEN(g.StartFEN); err == nil {
			b = parsed
		}
	}

	curr := g.RootMove
	ply := 1

	for curr != nil {
		fenBefore := b.FEN()
		isWhite := (b.Turn == chess.White)
		turnStr := "w"
		if !isWhite {
			turnStr = "b"
		}

		// Check for checkmate finish in game
		if curr.Next == nil && (strings.HasSuffix(curr.SAN, "#") || isCheckmatingMove(b, curr.Move)) {
			results = append(results, MinedPuzzle{
				ID:          fmt.Sprintf("puzzle-g%d-p%d", g.ID, ply),
				GameID:      g.ID,
				White:       g.Tag("White"),
				Black:       g.Tag("Black"),
				Date:        g.Tag("Date"),
				Event:       g.Tag("Event"),
				Ply:         ply,
				StartFEN:    fenBefore,
				Turn:        turnStr,
				BestMoveUCI: curr.UCI,
				BestMoveSAN: curr.SAN,
				Solution:    []string{curr.UCI},
				Theme:       "Checkmate",
				EvalDeltaCp: 1000,
				Description: fmt.Sprintf("%s found checkmate with %s!", getMoverName(g, isWhite), curr.SAN),
			})
			break
		}

		// If engine is available and game has tactical blunders/punishments
		if eng != nil && ply > 6 && !filter.OnlyCheckmate {
			// Check if current position has a decisive tactical move for the side to move
			bestMove, eval, err := eng.SearchBestMove(fenBefore, nil, 80)
			moverScoreCp := eval.ScoreCp
			moverMateIn := eval.MateIn
			if !isWhite {
				moverScoreCp = -moverScoreCp
				moverMateIn = -moverMateIn
			}
			if err == nil && (moverScoreCp >= 300 || moverMateIn > 0) {
				// Significant advantage detected!
				theme := "Tactical Advantage"
				if moverMateIn > 0 {
					theme = fmt.Sprintf("Mate in %d", moverMateIn)
				} else if moverScoreCp >= 500 {
					theme = "Decisive Advantage (+5.0)"
				}

				results = append(results, MinedPuzzle{
					ID:          fmt.Sprintf("puzzle-g%d-p%d", g.ID, ply),
					GameID:      g.ID,
					White:       g.Tag("White"),
					Black:       g.Tag("Black"),
					Date:        g.Tag("Date"),
					Event:       g.Tag("Event"),
					Ply:         ply,
					StartFEN:    fenBefore,
					Turn:        turnStr,
					BestMoveUCI: bestMove,
					BestMoveSAN: curr.SAN,
					Solution:    []string{bestMove},
					Theme:       theme,
					EvalDeltaCp: int(math.Abs(float64(eval.ScoreCp))),
					Description: fmt.Sprintf("%s has a winning tactic in %s vs %s.", getMoverName(g, isWhite), g.Tag("White"), g.Tag("Black")),
				})

				// Skip ahead 4 plies to avoid clustering puzzles in the same game
				for s := 0; s < 4 && curr != nil; s++ {
					b.MakeMove(curr.Move)
					curr = curr.Next
					ply++
				}
				continue
			}
		}

		b.MakeMove(curr.Move)
		curr = curr.Next
		ply++
	}

	return results
}

func isCheckmatingMove(b *chess.Board, m chess.Move) bool {
	copyB := b.Clone()
	copyB.MakeMove(m)
	return copyB.IsCheckmate()
}

func getMoverName(g *chess.Game, isWhite bool) string {
	if isWhite {
		w := g.Tag("White")
		if w != "?" && w != "" {
			return w
		}
		return "White"
	}
	bl := g.Tag("Black")
	if bl != "?" && bl != "" {
		return bl
	}
	return "Black"
}
