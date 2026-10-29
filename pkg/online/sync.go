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

// SyncProfile executes a full incremental + backfill sync for the given profile into targetDB,
// fetching the most recent games first and continuing into older games if necessary without counting games already in targetDB.
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

	existingSigs := targetDB.BuildGameSignatureSet()
	var allGames []*chess.Game

	// Fetch from Lichess.org (newest first, continuing into older games, skipping existing)
	if strings.TrimSpace(p.LichessUsername) != "" {
		games, skipped, err := sm.Lichess.FetchUserGamesIncremental(
			p.LichessUsername,
			p.LichessToken,
			p.RatedOnly,
			0,
			existingSigs,
		)
		if err != nil {
			res.Errors = append(res.Errors, fmt.Sprintf("Lichess error: %v", err))
		} else {
			res.LichessGames = len(games)
			res.DuplicatesSkipped += skipped
			allGames = append(allGames, games...)
		}
	}

	// Fetch from Chess.com (newest archives first, continuing into older archives, skipping existing)
	if strings.TrimSpace(p.ChessComUsername) != "" {
		games, skipped, err := sm.ChessCom.FetchUserGamesIncremental(
			p.ChessComUsername,
			0,
			existingSigs,
			nil,
		)
		if err != nil {
			res.Errors = append(res.Errors, fmt.Sprintf("Chess.com error: %v", err))
		} else {
			res.ChessComGames = len(games)
			res.DuplicatesSkipped += skipped
			allGames = append(allGames, games...)
		}
	}

	res.TotalDownloaded = len(allGames)
	if len(allGames) == 0 {
		p.LastSyncTime = startTime.UnixMilli()
		p.LastSyncGames = 0
		if sm.Profiles != nil {
			_ = sm.Profiles.SaveProfile(p)
		}
		res.DurationMs = time.Since(startTime).Milliseconds()
		return res, nil
	}

	// Batch import into target database
	importStats, err := targetDB.ImportGames(allGames)
	if err != nil {
		return nil, fmt.Errorf("importing games to database: %w", err)
	}

	res.Imported = importStats.Imported
	if importStats.Errors > 0 {
		res.Errors = append(res.Errors, fmt.Sprintf("%d games failed during binary encoding", importStats.Errors))
	}

	// Auto-deduplication if enabled
	if p.AutoDeduplicate {
		deletedDups, err := targetDB.DeleteDuplicates(scid.DuplicateCriteria{ExactMoves: true})
		if err != nil {
			res.Errors = append(res.Errors, fmt.Sprintf("Deduplication error: %v", err))
		} else {
			res.DuplicatesSkipped += deletedDups
		}
	}

	// Blended database synchronization if configured
	if p.TargetMode == TargetBlendedBase && strings.TrimSpace(p.BlendedDatabase) != "" {
		blendedDB, err := openOrCreateDatabase(p.BlendedDatabase, fmt.Sprintf("Blended Master Database for %s", p.Name))
		if err != nil {
			res.Errors = append(res.Errors, fmt.Sprintf("Blended database error: %v", err))
		} else if blendedDB != nil {
			defer blendedDB.Close()
			blendedSigs := blendedDB.BuildGameSignatureSet()
			newForBlended, _ := blendedSigs.FilterNewGames(allGames)
			if len(newForBlended) > 0 {
				_, _ = blendedDB.ImportGames(newForBlended)
			}
			if p.AutoDeduplicate {
				_, _ = blendedDB.DeleteDuplicates(scid.DuplicateCriteria{ExactMoves: true})
			}
		}
	}

	// Update Profile sync metrics
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
	TargetMode       DatabaseTargetMode `json:"targetMode"`    // "active", "dedicated", "blended"
	DedicatedPath    string             `json:"dedicatedPath"` // Path for dedicated DB
	BlendedPath      string             `json:"blendedPath"`   // Path for blended master DB
	RatedOnly        bool               `json:"ratedOnly"`
	MaxGames         int                `json:"maxGames"`
	AutoDeduplicate  bool               `json:"autoDeduplicate"`
	CreateProfile    bool               `json:"createProfile"` // Save as persistent profile
	ProgressFn       ProgressFn         `json:"-"`
}

// ImportPlayerGames downloads games for any player or persona from Lichess or Chess.com,
// starting from the most recent games and continuing into older games if necessary,
// skipping games already present in the target database WITHOUT counting them toward MaxGames.
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

	mode := opts.TargetMode
	if mode == "" {
		if activeDB != nil && activeDB.IsOpen {
			mode = TargetActiveBase
		} else {
			mode = TargetDedicatedBase
		}
	}

	// Build signature set of games ALREADY in the target database so we never count existing games toward MaxGames!
	existingSigs := scid.NewGameSignatureSet()
	switch mode {
	case TargetActiveBase:
		if activeDB != nil && activeDB.IsOpen {
			existingSigs = activeDB.BuildGameSignatureSet()
		}
	case TargetDedicatedBase, TargetBlendedBase:
		if strings.TrimSpace(opts.DedicatedPath) != "" {
			cleanPath := strings.TrimSpace(opts.DedicatedPath)
			base := strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(cleanPath, ".si4"), ".sn4"), ".sg4")
			if activeDB != nil && activeDB.IsOpen && strings.TrimSuffix(activeDB.BasePath, ".si4") == base {
				existingSigs = activeDB.BuildGameSignatureSet()
			} else if _, err := os.Stat(base + ".si4"); err == nil {
				tmpDB := scid.NewDatabase()
				if err := tmpDB.Open(base); err == nil {
					existingSigs = tmpDB.BuildGameSignatureSet()
					_ = tmpDB.Close()
				}
			}
		} else if activeDB != nil && activeDB.IsOpen {
			existingSigs = activeDB.BuildGameSignatureSet()
		}
	}

	var allGames []*chess.Game

	// Fetch from Lichess (newest first, continuing into older games, skipping existing)
	if strings.TrimSpace(opts.LichessUsername) != "" {
		if opts.ProgressFn != nil {
			opts.ProgressFn("connecting", 0, 0, fmt.Sprintf("Connecting to Lichess for %s (%d existing games in DB)...", opts.LichessUsername, existingSigs.Count()))
		}
		games, skipped, err := sm.Lichess.FetchUserGamesIncremental(
			opts.LichessUsername,
			opts.LichessToken,
			opts.RatedOnly,
			opts.MaxGames,
			existingSigs,
			opts.ProgressFn,
		)
		if err != nil {
			res.Errors = append(res.Errors, fmt.Sprintf("Lichess error: %v", err))
		} else {
			res.LichessGames = len(games)
			res.DuplicatesSkipped += skipped
			allGames = append(allGames, games...)
		}
	}

	// Fetch from Chess.com (newest archives first, continuing into older archives, skipping existing)
	if strings.TrimSpace(opts.ChessComUsername) != "" {
		if opts.ProgressFn != nil {
			opts.ProgressFn("connecting", 0, 0, fmt.Sprintf("Connecting to Chess.com for %s (%d existing games in DB)...", opts.ChessComUsername, existingSigs.Count()))
		}
		games, skipped, err := sm.ChessCom.FetchUserGamesIncremental(
			opts.ChessComUsername,
			opts.MaxGames,
			existingSigs,
			opts.ProgressFn,
		)
		if err != nil {
			res.Errors = append(res.Errors, fmt.Sprintf("Chess.com error: %v", err))
		} else {
			res.ChessComGames = len(games)
			res.DuplicatesSkipped += skipped
			allGames = append(allGames, games...)
		}
	}

	res.TotalDownloaded = len(allGames)
	if len(allGames) == 0 {
		res.DurationMs = time.Since(startTime).Milliseconds()
		return res, nil
	}

	// Helper to import only new games into a database
	importIntoDB := func(db *scid.Database, countTowardPrimary bool) error {
		dbSigs := db.BuildGameSignatureSet()
		newForDB, skipped := dbSigs.FilterNewGames(allGames)
		if countTowardPrimary {
			res.DuplicatesSkipped += skipped
		}
		if len(newForDB) == 0 {
			return nil
		}
		if opts.ProgressFn != nil {
			opts.ProgressFn("importing", len(newForDB), len(newForDB), fmt.Sprintf("Encoding and importing %d new games into database...", len(newForDB)))
		}
		stats, err := db.ImportGames(newForDB)
		if err != nil {
			return err
		}
		res.Imported += stats.Imported
		if opts.AutoDeduplicate {
			if opts.ProgressFn != nil {
				opts.ProgressFn("deduplicating", 0, 0, "Scanning and removing duplicate games...")
			}
			dups, err := db.DeleteDuplicates(scid.DuplicateCriteria{ExactMoves: true})
			if err == nil && countTowardPrimary {
				res.DuplicatesSkipped += dups
			}
		}
		return nil
	}

	// Import according to TargetMode
	switch mode {
	case TargetActiveBase:
		if activeDB == nil || !activeDB.IsOpen {
			return nil, fmt.Errorf("active database is not open")
		}
		if err := importIntoDB(activeDB, true); err != nil {
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
			if err := importIntoDB(activeDB, true); err != nil {
				return nil, fmt.Errorf("importing to dedicated database: %w", err)
			}
			break
		}

		db, err := openOrCreateDatabase(opts.DedicatedPath, fmt.Sprintf("Dedicated database for %s", displayName))
		if err != nil {
			return nil, err
		}
		defer db.Close()
		if err := importIntoDB(db, true); err != nil {
			return nil, fmt.Errorf("importing to dedicated database: %w", err)
		}

	case TargetBlendedBase:
		countedPrimary := false
		// Save to dedicated DB
		if strings.TrimSpace(opts.DedicatedPath) != "" {
			db, err := openOrCreateDatabase(opts.DedicatedPath, fmt.Sprintf("Dedicated database for %s", displayName))
			if err != nil {
				res.Errors = append(res.Errors, fmt.Sprintf("Dedicated database error: %v", err))
			} else {
				defer db.Close()
				_ = importIntoDB(db, true)
				countedPrimary = true
			}
		}

		// Also blend into master database
		if strings.TrimSpace(opts.BlendedPath) != "" {
			blendedDB, err := openOrCreateDatabase(opts.BlendedPath, "Blended Master Database")
			if err != nil {
				res.Errors = append(res.Errors, fmt.Sprintf("Blended database error: %v", err))
			} else {
				defer blendedDB.Close()
				_ = importIntoDB(blendedDB, !countedPrimary)
			}
		} else if activeDB != nil && activeDB.IsOpen {
			_ = importIntoDB(activeDB, !countedPrimary)
		}
	}

	// Optionally register as tracked study profile
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
