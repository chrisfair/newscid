package main

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"time"

	"newscid/pkg/academy"
	"newscid/pkg/database/scid"
)

func (a *App) ensureAcademyMgr() *academy.Manager {
	if a.academyMgr == nil {
		a.academyMgr = academy.NewManager("")
	}
	return a.academyMgr
}

// GetAcademyOverview returns the full Career Ladder, Chess University curriculum,
// and all built-in + custom + database-derived personalities.
func (a *App) GetAcademyOverview() academy.CareerOverview {
	return a.ensureAcademyMgr().GetOverview()
}

// RecordAcademyGameResult updates Career Elo, opponent W/D/L record, and Tier Boss exam state.
func (a *App) RecordAcademyGameResult(personalityID, outcome string) (academy.GameResultUpdate, error) {
	return a.ensureAcademyMgr().RecordGameResult(personalityID, outcome)
}

// CompleteAcademyLesson marks a Chess University lesson as finished.
func (a *App) CompleteAcademyLesson(lessonID string) (academy.CareerProfile, error) {
	return a.ensureAcademyMgr().CompleteLesson(lessonID)
}

// TestOutAcademyTier unlocks up to targetTier (1..6) for experienced players.
func (a *App) TestOutAcademyTier(targetTier int) (academy.CareerProfile, error) {
	return a.ensureAcademyMgr().TestOutToTier(targetTier)
}

// SaveCustomPersonality creates or updates a custom slider-based personality.
func (a *App) SaveCustomPersonality(p academy.Personality) (academy.Personality, error) {
	return a.ensureAcademyMgr().SaveCustomPersonality(p)
}

// DeleteCustomPersonality removes a custom or database-built personality.
func (a *App) DeleteCustomPersonality(id string) error {
	return a.ensureAcademyMgr().DeleteCustomPersonality(id)
}

// BuildAcademyPersonalityFromDB builds a playable Academy personality directly
// from a player's games in the currently open database.
func (a *App) BuildAcademyPersonalityFromDB(playerName string) (academy.Personality, error) {
	a.mu.RLock()
	db := a.db
	a.mu.RUnlock()
	if db == nil || !db.IsOpen {
		return academy.Personality{}, fmt.Errorf("no database is open — open a .si4 or .pgn database first")
	}
	return a.ensureAcademyMgr().BuildPersonalityFromDB(db, playerName)
}

// DatabasePlayerSuggestion represents a player found in the currently open database.
type DatabasePlayerSuggestion struct {
	Name      string `json:"name"`
	GameCount int    `json:"gameCount"`
	PeakElo   int    `json:"peakElo"`
}

// SearchDatabasePlayers returns top matching players from the currently open database
// sorted by game count so users can easily look up players or build personalities.
func (a *App) SearchDatabasePlayers(query string, limit int) []DatabasePlayerSuggestion {
	a.mu.RLock()
	db := a.db
	a.mu.RUnlock()
	if db == nil || !db.IsOpen {
		return []DatabasePlayerSuggestion{}
	}
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	q := strings.ToLower(strings.TrimSpace(query))

	type agg struct {
		name    string
		count   int
		peakElo int
	}
	counts := make(map[string]*agg)

	db.RLock()
	for _, entry := range db.Entries {
		if entry == nil || (entry.Flags&scid.FlagDelete) != 0 {
			continue
		}
		wName := strings.TrimSpace(db.Names.GetName(scid.NamePlayer, entry.WhiteID))
		bName := strings.TrimSpace(db.Names.GetName(scid.NamePlayer, entry.BlackID))

		if wName != "" && wName != "?" && (q == "" || strings.Contains(strings.ToLower(wName), q)) {
			if st, ok := counts[wName]; ok {
				st.count++
				if int(entry.WhiteElo) > st.peakElo {
					st.peakElo = int(entry.WhiteElo)
				}
			} else {
				counts[wName] = &agg{name: wName, count: 1, peakElo: int(entry.WhiteElo)}
			}
		}
		if bName != "" && bName != "?" && (q == "" || strings.Contains(strings.ToLower(bName), q)) {
			if st, ok := counts[bName]; ok {
				st.count++
				if int(entry.BlackElo) > st.peakElo {
					st.peakElo = int(entry.BlackElo)
				}
			} else {
				counts[bName] = &agg{name: bName, count: 1, peakElo: int(entry.BlackElo)}
			}
		}
	}
	db.RUnlock()

	list := make([]DatabasePlayerSuggestion, 0, len(counts))
	for _, st := range counts {
		list = append(list, DatabasePlayerSuggestion{
			Name:      st.name,
			GameCount: st.count,
			PeakElo:   st.peakElo,
		})
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].GameCount != list[j].GameCount {
			return list[i].GameCount > list[j].GameCount
		}
		return list[i].Name < list[j].Name
	})
	if len(list) > limit {
		list = list[:limit]
	}
	return list
}

// GetAcademyPersonalityMove selects the next move for a personality using its opening book,
// sliders, and running UCI engine (if available) or heuristic fallback.
func (a *App) GetAcademyPersonalityMove(personalityID, fen string) (academy.PersonalityMoveResult, error) {
	p, ok := a.ensureAcademyMgr().FindPersonality(personalityID)
	if !ok {
		return academy.PersonalityMoveResult{}, fmt.Errorf("unknown personality: %s", personalityID)
	}
	var candidates []academy.CandidateMove
	if a.engine != nil {
		if bestMove, err := a.GetBestMove(fen, nil, 10); err == nil && bestMove != "" {
			candidates = append(candidates, academy.CandidateMove{UCI: bestMove, ScoreCp: 50})
		}
	}
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	return academy.SelectPersonalityMove(fen, p, candidates, rng)
}

// ComputeSquareControlHeatmap returns attack/defense counts and hanging pieces for all 64 squares.
func (a *App) ComputeSquareControlHeatmap(fen string) ([]academy.SquareControlInfo, error) {
	return academy.ComputeSquareControl(fen)
}

// ComputeSpyThreatOverlay returns the opponent's sharpest immediate threat (PyChess Spy Mode).
func (a *App) ComputeSpyThreatOverlay(fen string) (academy.SpyThreatInfo, error) {
	return academy.ComputeSpyThreat(fen)
}

// DiagnoseGameForAcademyCoach analyzes a completed PGN offline and recommends University lessons.
func (a *App) DiagnoseGameForAcademyCoach(pgn, playerColor string) (academy.CoachReport, error) {
	return academy.DiagnoseGameForCoach(pgn, playerColor)
}
