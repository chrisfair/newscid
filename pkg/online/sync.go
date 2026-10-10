package online

import (
	"fmt"
	"os"
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

	// 5. Blended database synchronization if configured
	if p.TargetMode == TargetBlendedBase && strings.TrimSpace(p.BlendedDatabase) != "" {
		blendedDB, err := openOrCreateDatabase(p.BlendedDatabase, fmt.Sprintf("Blended Master Database for %s", p.Name))
		if err != nil {
			res.Errors = append(res.Errors, fmt.Sprintf("Blended database error: %v", err))
		} else if blendedDB != nil {
			defer blendedDB.Close()
			_, _ = blendedDB.ImportGames(allGames)
			if p.AutoDeduplicate {
				_, _ = blendedDB.DeleteDuplicates(scid.DuplicateCriteria{ExactMoves: true})
			}
		}
	}

	// 6. Update Profile sync metrics
	p.LastSyncTime = startTime.UnixMilli()
	p.LastSyncGames = res.Imported
	if sm.Profiles != nil {
		_ = sm.Profiles.SaveProfile(p)
	}

	res.DurationMs = time.Since(startTime).Milliseconds()
	return res, nil
}

// ImportPlayerOptions configures downloading games for another player/opponent/GM
// and merging them into dedicated, active, or blended databases.
type ImportPlayerOptions struct {
	PlayerName       string             `json:"playerName"`
	ChessComUsername string             `json:"chessComUsername"`
	LichessUsername  string             `json:"lichessUsername"`
	LichessToken     string             `json:"lichessToken,omitempty"`
	TargetMode       DatabaseTargetMode `json:"targetMode"`       // "active", "dedicated", "blended"
	DedicatedPath    string             `json:"dedicatedPath"`    // Path for dedicated DB
	BlendedPath      string             `json:"blendedPath"`      // Path for blended master DB
	RatedOnly        bool               `json:"ratedOnly"`
	MaxGames         int                `json:"maxGames"`
	AutoDeduplicate  bool               `json:"autoDeduplicate"`
	CreateProfile    bool               `json:"createProfile"`    // Save as persistent profile
}

// ImportPlayerGames downloads games for any player from Lichess or Chess.com and merges them
// according to the selected destination mode (active, dedicated standalone, or blended).
func (sm *SyncManager) ImportPlayerGames(opts ImportPlayerOptions, activeDB *scid.Database) (*SyncResult, error) {
	if strings.TrimSpace(opts.ChessComUsername) == "" && strings.TrimSpace(opts.LichessUsername) == "" {
		return nil, fmt.Errorf("at least one username (Chess.com or Lichess) must be specified")
	}

	startTime := time.Now()
	displayName := opts.PlayerName
	if displayName == "" {
		if opts.ChessComUsername != "" {
			displayName = opts.ChessComUsername
		} else {
			displayName = opts.LichessUsername
		}
	}

	res := &SyncResult{
		ProfileName: displayName,
		Timestamp:   startTime.UnixMilli(),
		Errors:      make([]string, 0),
	}

	var allGames []*chess.Game

	// 1. Fetch from Lichess
	if strings.TrimSpace(opts.LichessUsername) != "" {
		lichessPGN, err := sm.Lichess.FetchUserGames(
			opts.LichessUsername,
			0,
			opts.LichessToken,
			opts.RatedOnly,
			opts.MaxGames,
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
	if strings.TrimSpace(opts.ChessComUsername) != "" {
		chessComPGN, err := sm.ChessCom.FetchUserGames(opts.ChessComUsername, 0, 0)
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

	mode := opts.TargetMode
	if mode == "" {
		if activeDB != nil && activeDB.IsOpen {
			mode = TargetActiveBase
		} else {
			mode = TargetDedicatedBase
		}
	}

	// Helper to import into a database
	importIntoDB := func(db *scid.Database) error {
		stats, err := db.ImportGames(allGames)
		if err != nil {
			return err
		}
		res.Imported += stats.Imported
		if opts.AutoDeduplicate {
			dups, err := db.DeleteDuplicates(scid.DuplicateCriteria{ExactMoves: true})
			if err == nil {
				res.DuplicatesSkipped += dups
			}
		}
		return nil
	}

	// 3. Import according to TargetMode
	switch mode {
	case TargetActiveBase:
		if activeDB == nil || !activeDB.IsOpen {
			return nil, fmt.Errorf("active database is not open")
		}
		if err := importIntoDB(activeDB); err != nil {
			return nil, fmt.Errorf("importing to active database: %w", err)
		}

	case TargetDedicatedBase:
		if strings.TrimSpace(opts.DedicatedPath) == "" {
			return nil, fmt.Errorf("dedicated database path must be specified")
		}
		cleanPath := strings.TrimSpace(opts.DedicatedPath)
		base := strings.TrimSuffix(cleanPath, ".si4")
		base = strings.TrimSuffix(base, ".sn4")
		base = strings.TrimSuffix(base, ".sg4")

		if activeDB != nil && activeDB.IsOpen && strings.TrimSuffix(activeDB.BasePath, ".si4") == base {
			if err := importIntoDB(activeDB); err != nil {
				return nil, fmt.Errorf("importing to dedicated database: %w", err)
			}
			break
		}

		db, err := openOrCreateDatabase(opts.DedicatedPath, fmt.Sprintf("Dedicated database for %s", displayName))
		if err != nil {
			return nil, err
		}
		defer db.Close()
		if err := importIntoDB(db); err != nil {
			return nil, fmt.Errorf("importing to dedicated database: %w", err)
		}

	case TargetBlendedBase:
		// Save to dedicated DB
		if strings.TrimSpace(opts.DedicatedPath) != "" {
			db, err := openOrCreateDatabase(opts.DedicatedPath, fmt.Sprintf("Dedicated database for %s", displayName))
			if err != nil {
				res.Errors = append(res.Errors, fmt.Sprintf("Dedicated database error: %v", err))
			} else {
				defer db.Close()
				_ = importIntoDB(db)
			}
		}

		// Also blend into master database
		if strings.TrimSpace(opts.BlendedPath) != "" {
			blendedDB, err := openOrCreateDatabase(opts.BlendedPath, "Blended Master Database")
			if err != nil {
				res.Errors = append(res.Errors, fmt.Sprintf("Blended database error: %v", err))
			} else {
				defer blendedDB.Close()
				_ = importIntoDB(blendedDB)
			}
		} else if activeDB != nil && activeDB.IsOpen {
			_ = importIntoDB(activeDB)
		}
	}

	// 4. Optionally register as tracked study profile
	if opts.CreateProfile && sm.Profiles != nil {
		prof := &Profile{
			Name:             displayName,
			IsStudyTarget:    true,
			TargetMode:       mode,
			TargetDatabase:   opts.DedicatedPath,
			BlendedDatabase:  opts.BlendedPath,
			ChessComUsername: opts.ChessComUsername,
			LichessUsername:  opts.LichessUsername,
			LichessToken:     opts.LichessToken,
			RatedOnly:        opts.RatedOnly,
			AutoDeduplicate:  opts.AutoDeduplicate,
			LastSyncTime:     startTime.UnixMilli(),
			LastSyncGames:    res.Imported,
		}
		_ = sm.Profiles.SaveProfile(prof)
	}

	res.DurationMs = time.Since(startTime).Milliseconds()
	return res, nil
}

func openOrCreateDatabase(path string, desc string) (*scid.Database, error) {
	cleanPath := strings.TrimSpace(path)
	if cleanPath == "" {
		return nil, fmt.Errorf("database path is empty")
	}
	base := strings.TrimSuffix(cleanPath, ".si4")
	base = strings.TrimSuffix(base, ".sn4")
	base = strings.TrimSuffix(base, ".sg4")

	db := scid.NewDatabase()
	if _, err := os.Stat(base + ".si4"); os.IsNotExist(err) {
		if err := db.Create(base, desc); err != nil {
			return nil, fmt.Errorf("creating database %s: %w", base, err)
		}
		return db, nil
	}
	if err := db.Open(base); err != nil {
		return nil, fmt.Errorf("opening database %s: %w", base, err)
	}
	return db, nil
}
