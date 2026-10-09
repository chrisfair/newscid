package playerbot

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"time"

	"newscid/pkg/chess"
	"newscid/pkg/database/scid"
)

// BookMove represents a move historically played by a player from a given position.
type BookMove struct {
	UCI         string  `json:"uci"`
	SAN         string  `json:"san"`
	PlayedCount int     `json:"playedCount"`
	Wins        int     `json:"wins"`
	Draws       int     `json:"draws"`
	Losses      int     `json:"losses"`
	WinRate     float64 `json:"winRate"` // 0.0 to 100.0%
}

// PlayerPersonality models a player's real-world play style, historical opening repertoire,
// and calibrated playing strength derived from their games.
type PlayerPersonality struct {
	Name         string              `json:"name"`
	TotalGames   int                 `json:"totalGames"`
	WhiteGames   int                 `json:"whiteGames"`
	BlackGames   int                 `json:"blackGames"`
	Wins         int                 `json:"wins"`
	Draws        int                 `json:"draws"`
	Losses       int                 `json:"losses"`
	WinRate      float64             `json:"winRate"`
	EstimatedElo int                 `json:"estimatedElo"`
	PeakElo      int                 `json:"peakElo"`
	TopWhiteECOs []string            `json:"topWhiteECOs"`
	TopBlackECOs []string            `json:"topBlackECOs"`
	WhiteBook    map[string][]BookMove `json:"whiteBook"`
	BlackBook    map[string][]BookMove `json:"blackBook"`
}

// SimulatedMove represents a decision made by the simulated player.
type SimulatedMove struct {
	UCI         string  `json:"uci"`
	SAN         string  `json:"san"`
	IsBook      bool    `json:"isBook"`
	PlayedCount int     `json:"playedCount"`
	WinRate     float64 `json:"winRate"`
	EloTarget   int     `json:"eloTarget"`
	Commentary  string  `json:"commentary"`
}

// BuildPersonality extracts games for playerName from the database and constructs a complete
// opening repertoire and statistical playing model.
func BuildPersonality(db *scid.Database, playerName string) (*PlayerPersonality, error) {
	if db == nil || !db.IsOpen {
		return nil, fmt.Errorf("database is not open")
	}
	cleanName := strings.TrimSpace(playerName)
	if cleanName == "" {
		return nil, fmt.Errorf("player name cannot be empty")
	}

	p := &PlayerPersonality{
		Name:      cleanName,
		WhiteBook: make(map[string][]BookMove),
		BlackBook: make(map[string][]BookMove),
	}

	lowerTarget := strings.ToLower(cleanName)
	totalEloSum := 0
	eloCount := 0

	whiteECOCounts := make(map[string]int)
	blackECOCounts := make(map[string]int)

	numGames := len(db.Entries)
	for i := 0; i < numGames; i++ {
		entry := db.Entries[i]
		if (entry.Flags & scid.FlagDelete) != 0 {
			continue
		}

		whiteName := db.Names.GetName(scid.NamePlayer, entry.WhiteID)
		blackName := db.Names.GetName(scid.NamePlayer, entry.BlackID)

		isWhite := strings.Contains(strings.ToLower(whiteName), lowerTarget)
		isBlack := strings.Contains(strings.ToLower(blackName), lowerTarget)

		if !isWhite && !isBlack {
			continue
		}

		p.TotalGames++
		if isWhite {
			p.WhiteGames++
			if entry.WhiteElo > 0 {
				totalEloSum += int(entry.WhiteElo)
				eloCount++
				if int(entry.WhiteElo) > p.PeakElo {
					p.PeakElo = int(entry.WhiteElo)
				}
			}
			switch entry.Result {
			case chess.ResultWhiteWins:
				p.Wins++
			case chess.ResultDraw:
				p.Draws++
			case chess.ResultBlackWins:
				p.Losses++
			}
			ecoStr := scid.FormatECO(entry.EcoCode)
			if ecoStr != "" {
				whiteECOCounts[ecoStr]++
			}
		} else {
			p.BlackGames++
			if entry.BlackElo > 0 {
				totalEloSum += int(entry.BlackElo)
				eloCount++
				if int(entry.BlackElo) > p.PeakElo {
					p.PeakElo = int(entry.BlackElo)
				}
			}
			switch entry.Result {
			case chess.ResultBlackWins:
				p.Wins++
			case chess.ResultDraw:
				p.Draws++
			case chess.ResultWhiteWins:
				p.Losses++
			}
			ecoStr := scid.FormatECO(entry.EcoCode)
			if ecoStr != "" {
				blackECOCounts[ecoStr]++
			}
		}

		// Parse moves up to ply 30 to populate personal opening tree
		game, err := db.GetGame(i + 1)
		if err != nil || game == nil || game.RootMove == nil {
			continue
		}

		curr := game.RootMove
		board := chess.NewBoard()
		ply := 0

		for curr != nil && ply < 30 {
			posKey := normalizeFEN(board.FEN())
			moverIsWhite := (board.Turn == chess.White)

			if (moverIsWhite && isWhite) || (!moverIsWhite && isBlack) {
				bookMap := p.WhiteBook
				if !moverIsWhite {
					bookMap = p.BlackBook
				}

				recordBookMove(bookMap, posKey, curr.UCI, curr.SAN, entry.Result, moverIsWhite)
			}

			m, err := board.ParseUCI(curr.UCI)
			if err != nil {
				break
			}
			board.MakeMove(m)
			curr = curr.Next
			ply++
		}
	}

	if p.TotalGames == 0 {
		return nil, fmt.Errorf("no games found for player %q in database", cleanName)
	}

	if eloCount > 0 {
		p.EstimatedElo = totalEloSum / eloCount
	} else {
		p.EstimatedElo = 1500 // Sensible default if database lacks Elo tags
	}

	p.WinRate = float64(p.Wins) / float64(p.TotalGames) * 100.0

	// Sort and aggregate top ECOs
	p.TopWhiteECOs = topKeys(whiteECOCounts, 5)
	p.TopBlackECOs = topKeys(blackECOCounts, 5)

	return p, nil
}

func recordBookMove(book map[string][]BookMove, posKey, uci, san string, res chess.Result, moverIsWhite bool) {
	moves := book[posKey]
	foundIdx := -1
	for i, m := range moves {
		if m.UCI == uci {
			foundIdx = i
			break
		}
	}

	var bm BookMove
	if foundIdx >= 0 {
		bm = moves[foundIdx]
	} else {
		bm = BookMove{UCI: uci, SAN: san}
	}

	bm.PlayedCount++
	switch res {
	case chess.ResultWhiteWins:
		if moverIsWhite {
			bm.Wins++
		} else {
			bm.Losses++
		}
	case chess.ResultBlackWins:
		if !moverIsWhite {
			bm.Wins++
		} else {
			bm.Losses++
		}
	case chess.ResultDraw:
		bm.Draws++
	}

	if bm.PlayedCount > 0 {
		bm.WinRate = (float64(bm.Wins) + 0.5*float64(bm.Draws)) / float64(bm.PlayedCount) * 100.0
	}

	if foundIdx >= 0 {
		moves[foundIdx] = bm
	} else {
		moves = append(moves, bm)
	}
	book[posKey] = moves
}

// SelectSimulatedMove checks if the position is in the player's personal book.
// If found, it samples a candidate move weighted by how often the player actually played it.
func (p *PlayerPersonality) SelectSimulatedMove(fen string, isWhite bool) (*SimulatedMove, bool) {
	key := normalizeFEN(fen)
	bookMap := p.WhiteBook
	if !isWhite {
		bookMap = p.BlackBook
	}

	candidates, found := bookMap[key]
	if !found || len(candidates) == 0 {
		return nil, false
	}

	// Calculate total weight
	totalWeight := 0
	for _, c := range candidates {
		totalWeight += c.PlayedCount
	}

	if totalWeight <= 0 {
		return nil, false
	}

	// Weighted random selection
	r := rand.Intn(totalWeight)
	current := 0
	var chosen BookMove
	for _, c := range candidates {
		current += c.PlayedCount
		if r < current {
			chosen = c
			break
		}
	}
	if chosen.UCI == "" && len(candidates) > 0 {
		chosen = candidates[0]
	}

	return &SimulatedMove{
		UCI:         chosen.UCI,
		SAN:         chosen.SAN,
		IsBook:      true,
		PlayedCount: chosen.PlayedCount,
		WinRate:     chosen.WinRate,
		EloTarget:   p.EstimatedElo,
		Commentary: fmt.Sprintf("📖 %s played %s in %d games (%.1f%% score)",
			p.Name, chosen.SAN, chosen.PlayedCount, chosen.WinRate),
	}, true
}

// CalibrateEngineSettings maps an estimated player Elo (e.g. 1000 - 2850)
// to standard UCI options (UCI_LimitStrength, UCI_Elo, Skill Level 0-20).
func CalibrateEngineSettings(elo int) (limitStrength bool, uciElo int, skillLevel int) {
	limitStrength = true

	// Stockfish UCI_Elo is clamped between 1320 and 3190
	if elo < 1320 {
		uciElo = 1320
	} else if elo > 3190 {
		uciElo = 3190
	} else {
		uciElo = elo
	}

	// Linear interpolation for Skill Level 0-20 (from 800 to 2800 Elo)
	if elo <= 800 {
		skillLevel = 0
	} else if elo >= 2800 {
		skillLevel = 20
	} else {
		skillLevel = int(float64(elo-800) / 2000.0 * 20.0)
	}

	return limitStrength, uciElo, skillLevel
}

// normalizeFEN strips halfmove and fullmove counters so positions match transpositionally.
func normalizeFEN(fen string) string {
	parts := strings.Fields(fen)
	if len(parts) >= 4 {
		return strings.Join(parts[:4], " ")
	}
	return fen
}

func topKeys(counts map[string]int, limit int) []string {
	type kv struct {
		k string
		v int
	}
	var pairs []kv
	for k, v := range counts {
		pairs = append(pairs, kv{k, v})
	}
	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].v > pairs[j].v
	})
	var res []string
	for i := 0; i < len(pairs) && i < limit; i++ {
		res = append(res, pairs[i].k)
	}
	return res
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
