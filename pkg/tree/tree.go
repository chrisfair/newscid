package tree

import (
	"sort"
	"strings"

	"newscid/pkg/chess"
	"newscid/pkg/database/scid"
)

// MoveStat represents statistics for a move in the current position.
type MoveStat struct {
	Move      string  `json:"move"`      // SAN
	UCI       string  `json:"uci"`       // e.g. "e2e4"
	Count     int     `json:"count"`     // Number of games
	WhiteWins int     `json:"whiteWins"`
	Draws     int     `json:"draws"`
	BlackWins int     `json:"blackWins"`
	ScorePct  float64 `json:"scorePct"`  // White score % (or moving side %)
	AvgElo    int     `json:"avgElo"`
	ECO       string  `json:"eco,omitempty"`
}

// TreeResult contains the opening tree stats for a position.
type TreeResult struct {
	FEN        string      `json:"fen"`
	TotalGames int         `json:"totalGames"`
	Moves      []*MoveStat `json:"moves"`
}

// BuildTreeFromGames generates tree statistics for a position given a set of games.
func BuildTreeFromGames(fen string, games []*chess.Game) *TreeResult {
	targetBoard, err := chess.ParseFEN(fen)
	if err != nil {
		targetBoard = chess.NewBoard()
		fen = targetBoard.FEN()
	}

	targetFENCore := coreFEN(targetBoard.FEN())
	moveStatsMap := make(map[string]*MoveStat)
	totalGamesMatching := 0

	for _, g := range games {
		// Replay game and check if it reaches targetFENCore
		b := chess.NewBoard()
		if g.StartFEN != "" && g.StartFEN != chess.StartingFEN {
			if parsed, err := chess.ParseFEN(g.StartFEN); err == nil {
				b = parsed
			}
		}

		matched := false
		curr := g.RootMove
		for curr != nil {
			if coreFEN(b.FEN()) == targetFENCore {
				matched = true
				totalGamesMatching++

				san := curr.SAN
				stat, exists := moveStatsMap[san]
				if !exists {
					stat = &MoveStat{
						Move: san,
						UCI:  curr.UCI,
					}
					moveStatsMap[san] = stat
				}
				stat.Count++
				switch g.Result {
				case chess.ResultWhiteWins:
					stat.WhiteWins++
				case chess.ResultBlackWins:
					stat.BlackWins++
				case chess.ResultDraw:
					stat.Draws++
				}
				break
			}

			b.MakeMove(curr.Move)
			curr = curr.Next
		}

		if !matched && coreFEN(b.FEN()) == targetFENCore {
			totalGamesMatching++
		}
	}

	var moveStats []*MoveStat
	for _, stat := range moveStatsMap {
		if stat.Count > 0 {
			points := float64(stat.WhiteWins) + 0.5*float64(stat.Draws)
			stat.ScorePct = (points / float64(stat.Count)) * 100.0
		}
		moveStats = append(moveStats, stat)
	}

	sort.Slice(moveStats, func(i, j int) bool {
		return moveStats[i].Count > moveStats[j].Count
	})

	if moveStats == nil {
		moveStats = []*MoveStat{}
	}

	return &TreeResult{
		FEN:        fen,
		TotalGames: totalGamesMatching,
		Moves:      moveStats,
	}
}

// BuildTreeFromDatabase searches the database for games matching the position and aggregates move stats.
func BuildTreeFromDatabase(db *scid.Database, fen string, maxGames int) (*TreeResult, error) {
	if !db.IsOpen {
		return &TreeResult{FEN: fen, Moves: []*MoveStat{}}, nil
	}

	var sampledGames []*chess.Game
	numGames := db.NumGames()
	limit := numGames
	if maxGames > 0 && limit > maxGames {
		limit = maxGames
	}

	for i := 1; i <= limit; i++ {
		g, err := db.GetGame(i)
		if err == nil {
			sampledGames = append(sampledGames, g)
		}
	}

	return BuildTreeFromGames(fen, sampledGames), nil
}

// CoreFEN strips the halfmove clock and fullmove number for position matching.
func CoreFEN(fen string) string {
	return coreFEN(fen)
}

// SearchPositionInGames returns the 0-based indices of games that reach the target position.
func SearchPositionInGames(fen string, games []*chess.Game) []int {
	targetBoard, err := chess.ParseFEN(fen)
	if err != nil {
		targetBoard = chess.NewBoard()
		fen = targetBoard.FEN()
	}
	targetCore := coreFEN(targetBoard.FEN())
	var matched []int

	for i, g := range games {
		b := chess.NewBoard()
		if g.StartFEN != "" && g.StartFEN != chess.StartingFEN {
			if parsed, err := chess.ParseFEN(g.StartFEN); err == nil {
				b = parsed
			}
		}

		isMatch := (coreFEN(b.FEN()) == targetCore)
		curr := g.RootMove
		for !isMatch && curr != nil {
			b.MakeMove(curr.Move)
			if coreFEN(b.FEN()) == targetCore {
				isMatch = true
				break
			}
			curr = curr.Next
		}
		if isMatch {
			matched = append(matched, i)
		}
	}
	return matched
}

// coreFEN strips the halfmove clock, fullmove number, and ephemeral en-passant square for robust position matching.
func coreFEN(fen string) string {
	parts := strings.Split(strings.TrimSpace(fen), " ")
	if len(parts) >= 3 {
		return parts[0] + " " + parts[1] + " " + parts[2]
	}
	return fen
}

