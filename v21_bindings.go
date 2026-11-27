package main

import (
	"context"
	"fmt"
	"time"

	"newscid/pkg/database/scid"
	"newscid/pkg/online"
	"newscid/pkg/tablebase"
	"newscid/pkg/training"
)

// SearchMaterialAndStructure scans the active database in parallel for games matching
// a material imbalance, custom piece counts, or pawn structure motif.
func (a *App) SearchMaterialAndStructure(q scid.MaterialSearchQuery) ([]scid.MaterialSearchHit, error) {
	a.mu.RLock()
	db := a.db
	a.mu.RUnlock()
	if db == nil || !db.IsOpen {
		return nil, fmt.Errorf("no database is currently open")
	}
	return db.SearchMaterialAndStructure(q)
}

// GetEndgameDrills returns curated theoretical endgame sparring drills.
func (a *App) GetEndgameDrills(category string) []training.EndgameDrill {
	return training.GetEndgameDrills(category)
}

// EvaluateEndgameMove evaluates a theoretical endgame sparring move with Syzygy tablebase
// when available (<= 7 pieces) and returns WDL/DTZ feedback plus an opponent reply.
func (a *App) EvaluateEndgameMove(fen string, objective string, userSide string, useTablebase bool) (*training.EndgameEvalFeedback, error) {
	var tb *tablebase.Client
	if useTablebase {
		tb = tablebase.NewClient()
	}
	return training.EvaluateEndgameMove(fen, objective, userSide, tb)
}

// ImportLichessStudy fetches all chapters from a public (or token-authenticated) Lichess Study
// and optionally saves them directly into the currently open database.
func (a *App) ImportLichessStudy(studyURLOrID string, token string, importIntoDB bool) (*online.LichessStudyImportResult, error) {
	client := online.NewLichessClient()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	a.mu.RLock()
	db := a.db
	a.mu.RUnlock()

	return client.FetchLichessStudy(ctx, studyURLOrID, token, db, importIntoDB)
}
