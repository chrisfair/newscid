package online

import (
	"fmt"
	"strings"
	"time"

	"newscid/pkg/chess"
	"newscid/pkg/database/scid"
)

// SyncResult details the outcome of an online database sync operation.
type SyncResult struct {
	ProfileID         string   `json:"profileId"`
	ProfileName       string   `json:"profileName"`
	TotalDownloaded   int      `json:"totalDownloaded"`
	LichessGames      int      `json:"lichessGames"`
	ChessComGames     int      `json:"chessComGames"`
	Imported          int      `json:"imported"`
	DuplicatesSkipped int      `json:"duplicatesSkipped"`
	DurationMs        int64    `json:"durationMs"`
	Timestamp         int64    `json:"timestamp"`
	Errors            []string `json:"errors"`
}

// SyncManager orchestrates downloading online games and merging them into Scid databases.
type SyncManager struct {
	Profiles *ProfileManager
	Lichess  *LichessClient
	ChessCom *ChessComClient
}

// NewSyncManager creates a sync manager with default clients.
func NewSyncManager(pm *ProfileManager) *SyncManager {
	return &SyncManager{
		Profiles: pm,
		Lichess:  NewLichessClient(),
		ChessCom: NewChessComClient(),
	}
}

// SyncProfile executes a full sync for the given profile into targetDB.
func (sm *SyncManager) SyncProfile(p *Profile, targetDB *scid.Database) (*SyncResult, error) {
	if p == nil {
		return nil, fmt.Errorf("profile cannot be nil")
	}
	if targetDB == nil || !targetDB.IsOpen {
		return nil, fmt.Errorf("target database is not open")
	}

	startTime := time.Now()
	res := &SyncResult{
		ProfileID:   p.ID,
		ProfileName: p.Name,
		Timestamp:   startTime.UnixMilli(),
		Errors:      make([]string, 0),
	}

	var allGames []*chess.Game

	// 1. Fetch from Lichess.org
	if strings.TrimSpace(p.LichessUsername) != "" {
		lichessPGN, err := sm.Lichess.FetchUserGames(
			p.LichessUsername,
			p.LastSyncTime,
			p.LichessToken,
			p.RatedOnly,
			0,
		)
		if err != nil {
			res.Errors = append(res.Errors, fmt.Sprintf("Lichess error: %v", err))
		} else if strings.TrimSpace(lichessPGN) != "" {
			games, err := chess.ParsePGNDatabase(strings.NewReader(lichessPGN))
			if err != nil {
				res.Errors = append(res.Errors, fmt.Sprintf("Parsing Lichess PGN: %v", err))
			} else {
				res.LichessGames = len(games)
				allGames = append(allGames, games...)
			}
		}
	}

	// 2. Fetch from Chess.com
	if strings.TrimSpace(p.ChessComUsername) != "" {
		var sinceYear, sinceMonth int
		if p.LastSyncTime > 0 {
			lastSyncDate := time.UnixMilli(p.LastSyncTime)
			sinceYear = lastSyncDate.Year()
			sinceMonth = int(lastSyncDate.Month())
		}

		chessComPGN, err := sm.ChessCom.FetchUserGames(p.ChessComUsername, sinceYear, sinceMonth)
		if err != nil {
			res.Errors = append(res.Errors, fmt.Sprintf("Chess.com error: %v", err))
		} else if strings.TrimSpace(chessComPGN) != "" {
			games, err := chess.ParsePGNDatabase(strings.NewReader(chessComPGN))
			if err != nil {
				res.Errors = append(res.Errors, fmt.Sprintf("Parsing Chess.com PGN: %v", err))
			} else {
				res.ChessComGames = len(games)
				allGames = append(allGames, games...)
			}
		}
	}

	res.TotalDownloaded = len(allGames)
	if len(allGames) == 0 {
		res.DurationMs = time.Since(startTime).Milliseconds()
		return res, nil
	}

	// 3. Batch import into target database
	importStats, err := targetDB.ImportGames(allGames)
	if err != nil {
		return nil, fmt.Errorf("importing games to database: %w", err)
	}

	res.Imported = importStats.Imported
	if importStats.Errors > 0 {
		res.Errors = append(res.Errors, fmt.Sprintf("%d games failed during binary encoding", importStats.Errors))
	}

	// 4. Auto-deduplication if enabled
	if p.AutoDeduplicate {
		deletedDups, err := targetDB.DeleteDuplicates(scid.DuplicateCriteria{ExactMoves: true})
		if err != nil {
			res.Errors = append(res.Errors, fmt.Sprintf("Deduplication error: %v", err))
		} else {
			res.DuplicatesSkipped = deletedDups
		}
	}

	// 5. Update Profile sync metrics
	p.LastSyncTime = startTime.UnixMilli()
	p.LastSyncGames = res.Imported
	if sm.Profiles != nil {
		_ = sm.Profiles.SaveProfile(p)
	}

	res.DurationMs = time.Since(startTime).Milliseconds()
	return res, nil
}
