package playerbot

import (
	"fmt"
	"sort"
	"strings"

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
	Name             string                `json:"name"`
	TotalGames       int                   `json:"totalGames"`
	WhiteGames       int                   `json:"whiteGames"`
	BlackGames       int                   `json:"blackGames"`
	Wins             int                   `json:"wins"`
	Draws            int                   `json:"draws"`
	Losses           int                   `json:"losses"`
	WinRate          float64               `json:"winRate"`
	EstimatedElo     int                   `json:"estimatedElo"`
	PeakElo          int                   `json:"peakElo"`
	TopWhiteECOs     []string              `json:"topWhiteECOs"`
	TopBlackECOs     []string              `json:"topBlackECOs"`
	WhiteBook        map[string][]BookMove `json:"whiteBook"`
	BlackBook        map[string][]BookMove `json:"blackBook"`
	AvgGamePlies     float64               `json:"avgGamePlies"`
	AggressionRating float64               `json:"aggressionRating"` // 0.0 (ultra-solid) to 100.0 (tactical firebrand)
	ContemptValue    int                   `json:"contemptValue"`    // Calibrated UCI Contempt (-50 to +50)
	PlayStyle        string                `json:"playStyle"`        // "Tactical Attacker", "Solid Positional", etc.
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
	return BuildPersonalityFromIdentifiers(db, playerName, []string{playerName})
}

// BuildPersonalityFromIdentifiers extracts games matching any identifier in identifiers
// and constructs a unified opening repertoire and statistical playing model for the persona.
func BuildPersonalityFromIdentifiers(db *scid.Database, displayName string, identifiers []string) (*PlayerPersonality, error) {
	if db == nil || !db.IsOpen {
		return nil, fmt.Errorf("database is not open")
	}
	cleanName := strings.TrimSpace(displayName)
	if cleanName == "" {
		return nil, fmt.Errorf("player display name cannot be empty")
	}

	aliasLookup := make(map[string]bool)
	for _, id := range identifiers {
		clean := strings.ToLower(strings.TrimSpace(id))
		if clean != "" {
			aliasLookup[clean] = true
		}
	}
	if len(aliasLookup) == 0 {
		aliasLookup[strings.ToLower(cleanName)] = true
	}

	p := &PlayerPersonality{
		Name:      cleanName,
		WhiteBook: make(map[string][]BookMove),
		BlackBook: make(map[string][]BookMove),
	}

	totalEloSum := 0
	eloCount := 0
	totalPlies := 0
	gamesWithPlies := 0

	whiteECOCounts := make(map[string]int)
	blackECOCounts := make(map[string]int)

	numGames := db.NumGames()
	for i := 0; i < numGames; i++ {
		db.RLock()
		if i >= len(db.Entries) {
			db.RUnlock()
			break
		}
		entry := db.Entries[i]
		if entry == nil || (entry.Flags&scid.FlagDelete) != 0 {
			db.RUnlock()
			continue
		}

		whiteName := strings.ToLower(db.Names.GetName(scid.NamePlayer, entry.WhiteID))
		blackName := strings.ToLower(db.Names.GetName(scid.NamePlayer, entry.BlackID))
		entryWhiteElo := int(entry.WhiteElo)
		entryBlackElo := int(entry.BlackElo)
		entryResult := entry.Result
		entryEcoCode := entry.EcoCode
		db.RUnlock()

		isWhite := false
		isBlack := false
		for alias := range aliasLookup {
			if strings.Contains(whiteName, alias) {
				isWhite = true
			}
			if strings.Contains(blackName, alias) {
				isBlack = true
			}
		}

		if !isWhite && !isBlack {
			continue
		}

		p.TotalGames++
		if isWhite {
			p.WhiteGames++
			if entryWhiteElo > 0 {
				totalEloSum += entryWhiteElo
				eloCount++
				if entryWhiteElo > p.PeakElo {
					p.PeakElo = entryWhiteElo
				}
			}
			switch entryResult {
			case chess.ResultWhiteWins:
				p.Wins++
			case chess.ResultDraw:
				p.Draws++
			case chess.ResultBlackWins:
				p.Losses++
			}
			ecoStr := scid.FormatECO(entryEcoCode)
			if ecoStr != "" {
				whiteECOCounts[ecoStr]++
			}
		} else {
			p.BlackGames++
			if entryBlackElo > 0 {
				totalEloSum += entryBlackElo
				eloCount++
				if entryBlackElo > p.PeakElo {
					p.PeakElo = entryBlackElo
				}
			}
			switch entryResult {
			case chess.ResultBlackWins:
				p.Wins++
			case chess.ResultDraw:
				p.Draws++
			case chess.ResultWhiteWins:
				p.Losses++
			}
			ecoStr := scid.FormatECO(entryEcoCode)
			if ecoStr != "" {
				blackECOCounts[ecoStr]++
			}
		}

		// Parse full game moves (up to 250 plies) to populate personal move tree and calculate average game length
		game, err := db.GetGame(i + 1)
		if err != nil || game == nil || game.RootMove == nil {
			continue
		}

		curr := game.RootMove
		board := chess.NewBoard()
		if game.StartFEN != "" && game.StartFEN != chess.StartingFEN {
			if parsed, err := chess.ParseFEN(game.StartFEN); err == nil {
				board = parsed
			}
		}
		ply := 0

		for curr != nil && ply < 250 {
			posKey := normalizeFEN(board.FEN())
			moverIsWhite := (board.Turn == chess.White)

			if (moverIsWhite && isWhite) || (!moverIsWhite && isBlack) {
				bookMap := p.WhiteBook
				if !moverIsWhite {
					bookMap = p.BlackBook
				}

				recordBookMove(bookMap, posKey, curr.UCI, curr.SAN, entryResult, moverIsWhite)
			}

			m, err := board.ParseUCI(curr.UCI)
			if err != nil {
				break
			}
			board.MakeMove(m)
			curr = curr.Next
			ply++
		}

		if ply > 0 {
			totalPlies += ply
			gamesWithPlies++
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

	if gamesWithPlies > 0 {
		p.AvgGamePlies = float64(totalPlies) / float64(gamesWithPlies)
	}

	// Classify play style, aggression, and calibrated contempt
	decisiveRate := float64(p.Wins+p.Losses) / float64(p.TotalGames) * 100.0
	drawRate := float64(p.Draws) / float64(p.TotalGames) * 100.0

	if p.AvgGamePlies > 0 && p.AvgGamePlies < 60 && decisiveRate > 80 {
		p.PlayStyle = "Tactical Attacker"
		p.AggressionRating = 85.0
		p.ContemptValue = 40
	} else if drawRate > 35 || (p.AvgGamePlies > 80 && decisiveRate < 70) {
		p.PlayStyle = "Solid Positional"
		p.AggressionRating = 30.0
		p.ContemptValue = -10
	} else if p.AvgGamePlies > 85 {
		p.PlayStyle = "Grinding Endgame Specialist"
		p.AggressionRating = 45.0
		p.ContemptValue = 10
	} else if decisiveRate > 75 {
		p.PlayStyle = "Dynamic Fighter"
		p.AggressionRating = 70.0
		p.ContemptValue = 25
	} else {
		p.PlayStyle = "Balanced All-Rounder"
		p.AggressionRating = 50.0
		p.ContemptValue = 15
	}

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

// SelectSimulatedMove returns the move the player would choose from the given position.
// Per persona design: if the exact position exists in their historical games, it plays their
// most common response (or sole response), faithfully replicating their real play style.
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

	totalGamesAtPos := 0
	var chosen BookMove
	maxCount := -1

	for _, c := range candidates {
		totalGamesAtPos += c.PlayedCount
		if c.PlayedCount > maxCount || (c.PlayedCount == maxCount && c.WinRate > chosen.WinRate) {
			maxCount = c.PlayedCount
			chosen = c
		}
	}

	if chosen.UCI == "" && len(candidates) > 0 {
		chosen = candidates[0]
	}

	var comment string
	if len(candidates) == 1 {
		comment = fmt.Sprintf("📖 %s's sole historical response: %s (%d game, %.1f%% score)",
			p.Name, chosen.SAN, chosen.PlayedCount, chosen.WinRate)
	} else {
		pct := 0.0
		if totalGamesAtPos > 0 {
			pct = float64(chosen.PlayedCount) * 100.0 / float64(totalGamesAtPos)
		}
		comment = fmt.Sprintf("📖 %s's most frequent response: %s (played in %d of %d games, %.0f%% freq, %.1f%% score)",
			p.Name, chosen.SAN, chosen.PlayedCount, totalGamesAtPos, pct, chosen.WinRate)
	}

	return &SimulatedMove{
		UCI:         chosen.UCI,
		SAN:         chosen.SAN,
		IsBook:      true,
		PlayedCount: chosen.PlayedCount,
		WinRate:     chosen.WinRate,
		EloTarget:   p.EstimatedElo,
		Commentary:  comment,
	}, true
}

// CalibrateEngineSettingsForPersona configures UCI engine options tuned to the player's Elo,
// style tendencies (contempt, skill level), and aggression rating.
func (p *PlayerPersonality) CalibrateEngineSettingsForPersona() (limitStrength bool, uciElo int, skillLevel int, contempt int) {
	limitStrength, uciElo, skillLevel = CalibrateEngineSettings(p.EstimatedElo)
	contempt = p.ContemptValue
	return
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
