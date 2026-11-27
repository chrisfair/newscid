package scid

import (
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
	"sync"

	"newscid/pkg/chess"
)

// MaterialPattern identifies a named material imbalance or pawn-structure motif.
type MaterialPattern uint8

const (
	PatternAny MaterialPattern = iota
	PatternOppositeBishops
	PatternSameBishops
	PatternQueenVsRookMinor
	PatternExchangeImbalance
	PatternTwoBishops
	PatternIQP
	PatternCarlsbad
	PatternMaroczyBind
	PatternWinningSacrifice
)

var materialPatternNames = [...]string{
	PatternAny:               "any",
	PatternOppositeBishops:   "opposite_bishops",
	PatternSameBishops:       "same_bishops",
	PatternQueenVsRookMinor:  "queen_vs_rook_minor",
	PatternExchangeImbalance: "exchange_imbalance",
	PatternTwoBishops:        "two_bishops",
	PatternIQP:               "iqp",
	PatternCarlsbad:          "carlsbad",
	PatternMaroczyBind:       "maroczy_bind",
	PatternWinningSacrifice:  "winning_sacrifice",
}

func (p MaterialPattern) String() string {
	if int(p) < len(materialPatternNames) {
		return materialPatternNames[p]
	}
	return "any"
}

// ParseMaterialPattern parses a material/structure pattern identifier.
func ParseMaterialPattern(s string) MaterialPattern {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "opposite_bishops":
		return PatternOppositeBishops
	case "same_bishops":
		return PatternSameBishops
	case "queen_vs_rook_minor":
		return PatternQueenVsRookMinor
	case "exchange_imbalance":
		return PatternExchangeImbalance
	case "two_bishops":
		return PatternTwoBishops
	case "iqp":
		return PatternIQP
	case "carlsbad":
		return PatternCarlsbad
	case "maroczy_bind":
		return PatternMaroczyBind
	case "winning_sacrifice":
		return PatternWinningSacrifice
	default:
		return PatternAny
	}
}

func (p MaterialPattern) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.String())
}

func (p *MaterialPattern) UnmarshalJSON(data []byte) error {
	var raw string
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*p = ParseMaterialPattern(raw)
	return nil
}

// PieceCountFilter constrains piece counts for one side (-1 means unconstrained).
type PieceCountFilter struct {
	MinQ int `json:"minQ"`
	MaxQ int `json:"maxQ"`
	MinR int `json:"minR"`
	MaxR int `json:"maxR"`
	MinB int `json:"minB"`
	MaxB int `json:"maxB"`
	MinN int `json:"minN"`
	MaxN int `json:"maxN"`
	MinP int `json:"minP"`
	MaxP int `json:"maxP"`
}

// UnconstrainedPieceFilter returns a filter that imposes no bounds.
func UnconstrainedPieceFilter() PieceCountFilter {
	return PieceCountFilter{-1, -1, -1, -1, -1, -1, -1, -1, -1, -1}
}

// MaterialSearchQuery configures a material imbalance and pawn-structure search.
type MaterialSearchQuery struct {
	Pattern        MaterialPattern  `json:"pattern"`
	White          PieceCountFilter `json:"white"`
	Black          PieceCountFilter `json:"black"`
	UsePieceCounts bool             `json:"usePieceCounts"`
	MinPly         int              `json:"minPly"`
	MaxPly         int              `json:"maxPly"`
	MinSacrificeCp int              `json:"minSacrificeCp"`
	MaxResults     int              `json:"maxResults"`
	MaxGamesToScan int              `json:"maxGamesToScan"`
}

// MaterialSearchHit describes a database game matching a material/structure query.
type MaterialSearchHit struct {
	GameID   int    `json:"gameId"`
	White    string `json:"white"`
	Black    string `json:"black"`
	Event    string `json:"event"`
	Date     string `json:"date"`
	Result   string `json:"result"`
	ECO      string `json:"eco"`
	MatchPly int    `json:"matchPly"`
	MatchFEN string `json:"matchFen"`
	Summary  string `json:"summary"`
}

type sideMaterial struct {
	q, r, b, n, p int
	lightBishops  int
	darkBishops   int
	pawnFiles     [8]int
}

func (m sideMaterial) centipawns() int {
	return m.q*900 + m.r*500 + m.b*330 + m.n*320 + m.p*100
}

func (m sideMaterial) matches(f PieceCountFilter) bool {
	return inRange(m.q, f.MinQ, f.MaxQ) &&
		inRange(m.r, f.MinR, f.MaxR) &&
		inRange(m.b, f.MinB, f.MaxB) &&
		inRange(m.n, f.MinN, f.MaxN) &&
		inRange(m.p, f.MinP, f.MaxP)
}

func inRange(v, lo, hi int) bool {
	if lo >= 0 && v < lo {
		return false
	}
	if hi >= 0 && v > hi {
		return false
	}
	return true
}

func countMaterial(b *chess.Board) (w, bl sideMaterial) {
	for sq := chess.A1; sq <= chess.H8; sq++ {
		p := b.Squares[sq]
		if p == chess.Empty {
			continue
		}
		isLight := (sq.File()+sq.Rank())%2 != 0
		dst := &w
		if p.Color() == chess.Black {
			dst = &bl
		}
		switch p.Type() {
		case chess.Queen:
			dst.q++
		case chess.Rook:
			dst.r++
		case chess.Bishop:
			dst.b++
			if isLight {
				dst.lightBishops++
			} else {
				dst.darkBishops++
			}
		case chess.Knight:
			dst.n++
		case chess.Pawn:
			dst.p++
			dst.pawnFiles[sq.File()]++
		}
	}
	return w, bl
}

func matchesBoardPattern(b *chess.Board, w, bl sideMaterial, pat MaterialPattern) (bool, string) {
	switch pat {
	case PatternAny:
		return true, "Custom material match"
	case PatternOppositeBishops:
		if w.b == 1 && bl.b == 1 && w.q == 0 && bl.q == 0 && w.r == 0 && bl.r == 0 && w.n == 0 && bl.n == 0 {
			if (w.lightBishops == 1 && bl.darkBishops == 1) || (w.darkBishops == 1 && bl.lightBishops == 1) {
				return true, "Opposite-colored bishop endgame"
			}
		}
	case PatternSameBishops:
		if w.b == 1 && bl.b == 1 && w.q == 0 && bl.q == 0 && w.r == 0 && bl.r == 0 && w.n == 0 && bl.n == 0 {
			if (w.lightBishops == 1 && bl.lightBishops == 1) || (w.darkBishops == 1 && bl.darkBishops == 1) {
				return true, "Same-colored bishop endgame"
			}
		}
	case PatternQueenVsRookMinor:
		if (w.q == 1 && bl.q == 0 && bl.r >= 1 && (bl.b+bl.n) >= 1) ||
			(bl.q == 1 && w.q == 0 && w.r >= 1 && (w.b+w.n) >= 1) {
			return true, "Queen vs. Rook + Minor imbalance"
		}
	case PatternExchangeImbalance:
		if (w.r == bl.r+1 && (bl.b+bl.n) == (w.b+w.n)+1) ||
			(bl.r == w.r+1 && (w.b+w.n) == (bl.b+bl.n)+1) {
			return true, "Exchange imbalance (Rook vs. Minor)"
		}
	case PatternTwoBishops:
		if w.b == 2 && bl.b < 2 {
			return true, "White holds the Bishop Pair"
		}
		if bl.b == 2 && w.b < 2 {
			return true, "Black holds the Bishop Pair"
		}
	case PatternIQP:
		if w.pawnFiles[3] == 1 && w.pawnFiles[2] == 0 && w.pawnFiles[4] == 0 && b.Squares[chess.D4] == chess.WP {
			return true, "White Isolated Queen's Pawn (d4)"
		}
		if bl.pawnFiles[3] == 1 && bl.pawnFiles[2] == 0 && bl.pawnFiles[4] == 0 && b.Squares[chess.D5] == chess.BP {
			return true, "Black Isolated Queen's Pawn (d5)"
		}
	case PatternCarlsbad:
		if b.Squares[chess.D4] == chess.WP && b.Squares[chess.E3] == chess.WP && w.pawnFiles[2] == 0 &&
			b.Squares[chess.D5] == chess.BP && b.Squares[chess.C6] == chess.BP && b.Squares[chess.E6] == chess.BP {
			return true, "Carlsbad pawn structure"
		}
	case PatternMaroczyBind:
		if b.Squares[chess.C4] == chess.WP && b.Squares[chess.E4] == chess.WP && w.pawnFiles[3] == 0 &&
			bl.pawnFiles[2] == 0 && b.Squares[chess.D6] == chess.BP {
			return true, "Maroczy Bind structure"
		}
	}
	return false, ""
}

func scanGameForMaterial(gameID int, g *chess.Game, q MaterialSearchQuery) (MaterialSearchHit, bool) {
	if g == nil || g.RootMove == nil {
		return MaterialSearchHit{}, false
	}

	b := chess.NewBoard()
	if g.StartFEN != "" && g.StartFEN != chess.StartingFEN {
		if parsed, err := chess.ParseFEN(g.StartFEN); err == nil {
			b = parsed
		}
	}

	if q.Pattern == PatternWinningSacrifice {
		if g.Result != chess.ResultWhiteWins && g.Result != chess.ResultBlackWins {
			return MaterialSearchHit{}, false
		}
		winnerIsWhite := g.Result == chess.ResultWhiteWins
		deficitStreak := 0
		var sacPly int
		var sacFEN string
		var maxDeficit int

		ply := 0
		for curr := g.RootMove; curr != nil; curr = curr.Next {
			b.MakeMove(curr.Move)
			ply++
			if q.MaxPly > 0 && ply > q.MaxPly {
				break
			}
			if ply < q.MinPly {
				continue
			}
			w, bl := countMaterial(b)
			diff := w.centipawns() - bl.centipawns()
			if !winnerIsWhite {
				diff = -diff
			}
			if diff <= -q.MinSacrificeCp {
				deficitStreak++
				if -diff > maxDeficit {
					maxDeficit = -diff
					sacPly = ply
					sacFEN = b.FEN()
				}
				if deficitStreak >= 2 {
					side := "White"
					if !winnerIsWhite {
						side = "Black"
					}
					return buildMaterialHit(gameID, g, sacPly, sacFEN,
						fmt.Sprintf("%s sacrificed %.1f pawns and won", side, float64(maxDeficit)/100.0)), true
				}
			} else {
				deficitStreak = 0
			}
		}
		return MaterialSearchHit{}, false
	}

	ply := 0
	for curr := g.RootMove; curr != nil; curr = curr.Next {
		b.MakeMove(curr.Move)
		ply++
		if q.MaxPly > 0 && ply > q.MaxPly {
			break
		}
		if ply < q.MinPly {
			continue
		}
		w, bl := countMaterial(b)
		if q.UsePieceCounts && (!w.matches(q.White) || !bl.matches(q.Black)) {
			continue
		}
		if ok, label := matchesBoardPattern(b, w, bl, q.Pattern); ok {
			return buildMaterialHit(gameID, g, ply, b.FEN(), label), true
		}
	}
	return MaterialSearchHit{}, false
}

func buildMaterialHit(gameID int, g *chess.Game, ply int, fen, summary string) MaterialSearchHit {
	return MaterialSearchHit{
		GameID:   gameID,
		White:    g.Tag("White"),
		Black:    g.Tag("Black"),
		Event:    g.Tag("Event"),
		Date:     g.Tag("Date"),
		Result:   string(g.Result),
		ECO:      g.Tag("ECO"),
		MatchPly: ply,
		MatchFEN: fen,
		Summary:  summary,
	}
}

// SearchMaterialAndStructure scans the open database in parallel for games reaching
// the requested material imbalance, custom piece counts, or pawn structure.
func (db *Database) SearchMaterialAndStructure(q MaterialSearchQuery) ([]MaterialSearchHit, error) {
	if db == nil || !db.IsOpen {
		return nil, fmt.Errorf("database is not open")
	}

	if q.MinPly <= 0 {
		q.MinPly = 6
	}
	if q.MinSacrificeCp <= 0 {
		q.MinSacrificeCp = 250
	}
	if q.MaxResults <= 0 {
		q.MaxResults = 100
	}
	if q.MaxGamesToScan <= 0 {
		q.MaxGamesToScan = 5000
	}

	type loadedGame struct {
		id   int
		game *chess.Game
	}

	numGames := db.NumGames()
	limit := min(numGames, q.MaxGamesToScan)
	loaded := make([]loadedGame, 0, min(limit, 256))

	for i := 1; i <= numGames && len(loaded) < limit; i++ {
		db.RLock()
		if i-1 >= len(db.Entries) {
			db.RUnlock()
			break
		}
		entry := db.Entries[i-1]
		deleted := entry == nil || (entry.Flags&FlagDelete) != 0
		db.RUnlock()
		if deleted {
			continue
		}
		g, err := db.GetGame(i)
		if err != nil || g == nil || g.RootMove == nil {
			continue
		}
		loaded = append(loaded, loadedGame{id: i, game: g})
	}

	if len(loaded) == 0 {
		return []MaterialSearchHit{}, nil
	}

	workers := min(runtime.GOMAXPROCS(0), len(loaded))
	perWorker := make([][]MaterialSearchHit, workers)
	chunk := (len(loaded) + workers - 1) / workers

	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		start := w * chunk
		if start >= len(loaded) {
			break
		}
		end := min(start+chunk, len(loaded))
		wg.Add(1)
		go func(idx, lo, hi int) {
			defer wg.Done()
			var local []MaterialSearchHit
			for i := lo; i < hi; i++ {
				if hit, ok := scanGameForMaterial(loaded[i].id, loaded[i].game, q); ok {
					local = append(local, hit)
				}
			}
			perWorker[idx] = local
		}(w, start, end)
	}
	wg.Wait()

	hits := make([]MaterialSearchHit, 0, q.MaxResults)
	for _, local := range perWorker {
		for _, h := range local {
			hits = append(hits, h)
			if len(hits) >= q.MaxResults {
				return hits, nil
			}
		}
	}
	return hits, nil
}
