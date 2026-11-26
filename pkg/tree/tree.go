package tree

import (
	"runtime"
	"sort"
	"strings"
	"sync"

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

// boardsMatchCore compares board piece placement, side to move, and castling rights directly
// without allocating or parsing FEN strings in the per-move hot loop.
func boardsMatchCore(a, b *chess.Board) bool {
	return a.Turn == b.Turn && a.Castling == b.Castling && a.Squares == b.Squares
}

// initGameBoard initializes a board to a game's starting position.
func initGameBoard(g *chess.Game) *chess.Board {
	if g.StartFEN != "" && g.StartFEN != chess.StartingFEN {
		if parsed, err := chess.ParseFEN(g.StartFEN); err == nil {
			return parsed
		}
	}
	return chess.NewBoard()
}

func scanTreeChunk(targetBoard *chess.Board, games []*chess.Game) (map[string]*MoveStat, int) {
	moveStatsMap := make(map[string]*MoveStat)
	totalMatching := 0

	for _, g := range games {
		b := initGameBoard(g)
		matched := false
		curr := g.RootMove
		for curr != nil {
			if boardsMatchCore(b, targetBoard) {
				matched = true
				totalMatching++

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

		if !matched && boardsMatchCore(b, targetBoard) {
			totalMatching++
		}
	}

	return moveStatsMap, totalMatching
}

// BuildTreeFromGames generates tree statistics for a position given a set of games.
// Replays games concurrently across worker goroutines for larger slices.
func BuildTreeFromGames(fen string, games []*chess.Game) *TreeResult {
	targetBoard, err := chess.ParseFEN(fen)
	if err != nil {
		targetBoard = chess.NewBoard()
		fen = targetBoard.FEN()
	}

	var moveStatsMap map[string]*MoveStat
	totalGamesMatching := 0

	workers := min(runtime.GOMAXPROCS(0), len(games))
	if len(games) < 32 || workers <= 1 {
		moveStatsMap, totalGamesMatching = scanTreeChunk(targetBoard, games)
	} else {
		type chunkResult struct {
			stats   map[string]*MoveStat
			matched int
		}
		results := make([]chunkResult, workers)
		chunk := (len(games) + workers - 1) / workers
		var wg sync.WaitGroup
		for w := 0; w < workers; w++ {
			start := w * chunk
			if start >= len(games) {
				break
			}
			end := min(start+chunk, len(games))
			wg.Add(1)
			go func(idx, lo, hi int) {
				defer wg.Done()
				s, m := scanTreeChunk(targetBoard, games[lo:hi])
				results[idx] = chunkResult{stats: s, matched: m}
			}(w, start, end)
		}
		wg.Wait()

		moveStatsMap = make(map[string]*MoveStat)
		for _, res := range results {
			totalGamesMatching += res.matched
			for san, s := range res.stats {
				dst, exists := moveStatsMap[san]
				if !exists {
					moveStatsMap[san] = s
					continue
				}
				dst.Count += s.Count
				dst.WhiteWins += s.WhiteWins
				dst.Draws += s.Draws
				dst.BlackWins += s.BlackWins
			}
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
		if moveStats[i].Count != moveStats[j].Count {
			return moveStats[i].Count > moveStats[j].Count
		}
		return moveStats[i].Move < moveStats[j].Move
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
	if db == nil || !db.IsOpen {
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

func gameReachesPosition(g *chess.Game, targetBoard *chess.Board) bool {
	b := initGameBoard(g)
	if boardsMatchCore(b, targetBoard) {
		return true
	}
	for curr := g.RootMove; curr != nil; curr = curr.Next {
		b.MakeMove(curr.Move)
		if boardsMatchCore(b, targetBoard) {
			return true
		}
	}
	return false
}

// SearchPositionInGames returns the 0-based indices of games that reach the target position.
func SearchPositionInGames(fen string, games []*chess.Game) []int {
	targetBoard, err := chess.ParseFEN(fen)
	if err != nil {
		targetBoard = chess.NewBoard()
	}

	workers := min(runtime.GOMAXPROCS(0), len(games))
	if len(games) < 32 || workers <= 1 {
		var matched []int
		for i, g := range games {
			if gameReachesPosition(g, targetBoard) {
				matched = append(matched, i)
			}
		}
		return matched
	}

	perWorker := make([][]int, workers)
	chunk := (len(games) + workers - 1) / workers
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		start := w * chunk
		if start >= len(games) {
			break
		}
		end := min(start+chunk, len(games))
		wg.Add(1)
		go func(idx, lo, hi int) {
			defer wg.Done()
			var local []int
			for i := lo; i < hi; i++ {
				if gameReachesPosition(games[i], targetBoard) {
					local = append(local, i)
				}
			}
			perWorker[idx] = local
		}(w, start, end)
	}
	wg.Wait()

	var matched []int
	for _, local := range perWorker {
		matched = append(matched, local...)
	}
	return matched
}

// coreFEN strips the halfmove clock, fullmove number, and ephemeral en-passant square for robust position matching.
func coreFEN(fen string) string {
	parts := strings.Fields(strings.TrimSpace(fen))
	if len(parts) >= 3 {
		return parts[0] + " " + parts[1] + " " + parts[2]
	}
	return fen
}

