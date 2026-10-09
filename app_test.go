package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"newscid/pkg/chess"
	"newscid/pkg/database/scid"
	"newscid/pkg/online"
)


func TestAppDatabaseAndEngineDiscovery(t *testing.T) {
	app := NewApp()
	app.startup(context.Background())

	// Test Engine Discovery
	engines := app.GetDiscoveredEngines()
	assert.NotEmpty(t, engines, "Should discover installed engines")
	t.Logf("App discovered %d engines", len(engines))
	var availableCount int
	for _, e := range engines {
		if e.Available {
			availableCount++
			t.Logf("Available engine: %s (%s)", e.Name, e.Cmd)
		}
	}
	assert.Greater(t, availableCount, 0, "Should have at least one available engine")

	// Test Database Discovery
	dbs := app.GetDiscoveredDatabases()
	assert.NotEmpty(t, dbs, "Should discover Scid databases")
	t.Logf("App discovered %d databases", len(dbs))
	for _, db := range dbs {
		t.Logf("Database: [%s] %s (%s, %d games)", db.Category, db.Name, db.SizeStr, db.NumGames)
	}

	// Test opening a discovered database (prefer ChristopherFair or tactics for fast test execution)
	var targetBase string
	for _, db := range dbs {
		if (db.Name == "ChristopherFair" || db.Name == "tactics") && db.Exists {
			targetBase = db.BasePath
			break
		}
	}
	if targetBase == "" {
		for _, db := range dbs {
			if db.Exists && db.Type == "scid" && db.NumGames > 0 && db.NumGames < 100000 {
				targetBase = db.BasePath
				break
			}
		}
	}

	if targetBase != "" {
		info, err := app.OpenDatabase(targetBase)
		require.NoError(t, err)
		assert.NotNil(t, info)
		assert.Greater(t, info.NumGames, 0)
		t.Logf("Successfully opened base %s with %d games", info.Name, info.NumGames)

		// Test games listing
		page, err := app.GetGames(0, 10, scid.SearchCriteria{})
		require.NoError(t, err)
		assert.NotEmpty(t, page.Games)

		// Test opening tree
		tree, err := app.GetOpeningTree("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")
		require.NoError(t, err)
		assert.NotNil(t, tree)
	}
}

func TestAppOpenPGNFile(t *testing.T) {
	app := NewApp()
	app.startup(context.Background())

	// Create a temporary PGN file
	tmpDir := t.TempDir()
	pgnPath := filepath.Join(tmpDir, "test_game.pgn")
	pgnContent := `[Event "Test Match"]
[Site "NewScid"]
[Date "2026.10.07"]
[Round "1"]
[White "Player 1"]
[Black "Player 2"]
[Result "1-0"]

1. e4 e5 2. Nf3 Nc6 3. Bb5 a6 4. Ba4 Nf6 5. O-O Be7 1-0
`
	err := os.WriteFile(pgnPath, []byte(pgnContent), 0644)
	require.NoError(t, err)

	info, err := app.OpenDatabase(pgnPath)
	require.NoError(t, err)
	assert.Equal(t, "test_game.pgn", info.Name)
	assert.Equal(t, 1, info.NumGames)

	game, err := app.GetGame(1)
	require.NoError(t, err)
	assert.Equal(t, "Player 1", game.Tag("White"))
	assert.Equal(t, "Player 2", game.Tag("Black"))
	assert.Equal(t, "e4", game.RootMove.SAN)
}

func TestAppPlayAndAnnotation(t *testing.T) {
	app := NewApp()
	app.startup(context.Background())

	// Test GetBestMove
	move, err := app.GetBestMove("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1", nil, 150)
	require.NoError(t, err)
	assert.NotEmpty(t, move)
	t.Logf("App GetBestMove: %s", move)

	// Test Annotation
	game, err := app.ParsePGN("1. e4 e5 2. Qh5 Nc6 3. Bc4 Nf6 4. Qxf7# 1-0")
	require.NoError(t, err)
	app.activeGame = game

	analysis, err := app.AnnotateCurrentGame(150)
	require.NoError(t, err)
	assert.NotNil(t, analysis)
	assert.NotEmpty(t, analysis.MoveAnalyses)
	t.Logf("Scholar's Mate analysis: White acc: %.1f%%, Black acc: %.1f%%", analysis.WhiteAccuracy, analysis.BlackAccuracy)
}

func TestAppTacticsAndSearchPosition(t *testing.T) {
	app := NewApp()
	app.startup(context.Background())

	// Test Tactics puzzle loading
	puzzle, err := app.LoadTacticsPuzzle("/usr/share/scid/bases/tactics", 1)
	if err == nil {
		assert.Equal(t, 1, puzzle.GameID)
		assert.Greater(t, puzzle.TotalGames, 0)
		assert.NotEmpty(t, puzzle.StartFEN)
		assert.NotEmpty(t, puzzle.Moves)
		t.Logf("Loaded puzzle #1: StartFEN=%s, Moves=%v", puzzle.StartFEN, puzzle.Moves)
	}

	// Test SearchPosition on user base
	userBase := "/home/christopher/ChessDataBases/ChristopherFair/ChristopherFair"
	if _, err := os.Stat(userBase + ".si4"); err == nil {
		_, err := app.OpenDatabase(userBase)
		require.NoError(t, err)

		results, err := app.SearchPosition("rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1")
		require.NoError(t, err)
		t.Logf("SearchPosition for 1. e4 found %d games in ChristopherFair", len(results))
		assert.NotEmpty(t, results)
	}
}

func TestAppNewFeatures(t *testing.T) {
	app := NewApp()
	app.startup(context.Background())

	// 1. Repertoires
	reps := app.GetRepertoires()
	require.NotEmpty(t, reps)
	assert.GreaterOrEqual(t, len(reps), 2)

	check, err := app.VerifyRepertoireMove(reps[0].ID, chess.StartingFEN, "e2e4")
	require.NoError(t, err)
	assert.NotNil(t, check)

	// 2. Progressive Tactics Puzzle
	progPuzzle, err := app.GetProgressivePuzzle(1)
	if err == nil {
		assert.NotEmpty(t, progPuzzle.StartFEN)
		assert.Contains(t, progPuzzle.Theme, "Level 1")
		t.Logf("Progressive Puzzle Level 1: %s (%s)", progPuzzle.Theme, progPuzzle.StartFEN)
	}

	// 3. Database Puzzle Mining
	userBase := "/home/christopher/ChessDataBases/ChristopherFair/ChristopherFair"
	if _, err := os.Stat(userBase + ".si4"); err == nil {
		_, err := app.OpenDatabase(userBase)
		require.NoError(t, err)

		mined, err := app.MinePuzzlesFromDatabase("", 15)
		require.NoError(t, err)
		t.Logf("Mined %d tactical puzzles from ChristopherFair", len(mined))
	}
}

func TestAppHeaderSearch(t *testing.T) {
	app := NewApp()
	app.startup(context.Background())

	tacticsPath := "/usr/share/scid/bases/tactics"
	if _, err := os.Stat(tacticsPath + ".si4"); os.IsNotExist(err) {
		t.Skip("Tactics base not found on system, skipping app search test")
	}

	_, err := app.OpenDatabase(tacticsPath)
	require.NoError(t, err)
	defer app.CloseDatabase()

	// Count Fischer matches
	fischerCount, err := app.CountSearchMatches(scid.SearchCriteria{
		Player: "Fischer",
	})
	require.NoError(t, err)
	assert.Greater(t, fischerCount, 0)

	// Fetch page with advanced criteria
	page, err := app.GetGames(0, 10, scid.SearchCriteria{
		Player:   "Fischer",
		Result:   "1-0",
		MinMoves: 5,
	})
	require.NoError(t, err)
	assert.NotNil(t, page)
	assert.LessOrEqual(t, len(page.Games), 10)
	t.Logf("App HeaderSearch found %d Fischer wins with MinMoves>=5 (page len: %d)", page.Total, len(page.Games))
}

func TestAppVersionInfo(t *testing.T) {
	app := NewApp()
	v := app.GetVersionInfo()
	assert.NotEmpty(t, v.Version)
	assert.NotEmpty(t, v.GoVersion)
	assert.NotEmpty(t, v.Platform)
	t.Logf("App Version: %s (Commit: %s, Platform: %s)", v.Version, v.GitCommit, v.Platform)
}

func TestAppDatabaseMaintenanceOperations(t *testing.T) {
	app := NewApp()
	app.startup(context.Background())

	// 1. Errors when no database open
	_, err := app.CompactDatabase()
	assert.Error(t, err)
	assert.Error(t, app.SortDatabase(scid.SortCriteria{Field: scid.SortByDate}))
	_, err = app.FindDuplicateGames(scid.DuplicateCriteria{ExactMoves: true})
	assert.Error(t, err)
	_, err = app.DeleteDuplicateGames(scid.DuplicateCriteria{ExactMoves: true})
	assert.Error(t, err)
	_, err = app.VerifyDatabase()
	assert.Error(t, err)
	_, err = app.GetDatabaseStatistics()
	assert.Error(t, err)
	_, err = app.ImportPGNBatch("1. e4")
	assert.Error(t, err)
	_, err = app.ImportPGNFileBatch("somefile.pgn")
	assert.Error(t, err)

	// 2. Create a test database
	tmpDir := t.TempDir()
	basePath := filepath.Join(tmpDir, "app_test_db")
	dbInfo, err := app.CreateDatabase(basePath, "App Maintenance Test DB")
	require.NoError(t, err)
	assert.Equal(t, 0, dbInfo.NumGames)

	// 3. Batch PGN Import
	pgnBatch := `[Event "Round 1"]
[Site "City A"]
[Date "2024.01.01"]
[Round "1"]
[White "Carlsen, Magnus"]
[Black "Nakamura, Hikaru"]
[Result "1-0"]
[WhiteElo "2830"]
[BlackElo "2780"]
[ECO "C88"]

1. e4 e5 2. Nf3 Nc6 3. Bb5 a6 1-0

[Event "Round 1"]
[Site "City A"]
[Date "2024.01.01"]
[Round "1"]
[White "Carlsen, Magnus"]
[Black "Nakamura, Hikaru"]
[Result "1-0"]
[WhiteElo "2830"]
[BlackElo "2780"]
[ECO "C88"]

1. e4 e5 2. Nf3 Nc6 3. Bb5 a6 1-0

[Event "Round 2"]
[Site "City B"]
[Date "2023.05.10"]
[Round "2"]
[White "Caruana, Fabiano"]
[Black "Nepomniachtchi, Ian"]
[Result "0-1"]
[WhiteElo "2790"]
[BlackElo "2770"]
[ECO "B90"]

1. e4 c5 2. Nf3 d6 3. d4 cxd4 0-1
`
	importStats, err := app.ImportPGNBatch(pgnBatch)
	require.NoError(t, err)
	assert.Equal(t, 3, importStats.Imported)

	// 4. Database Statistics
	stats, err := app.GetDatabaseStatistics()
	require.NoError(t, err)
	assert.Equal(t, 3, stats.TotalGames)
	assert.Equal(t, 2, stats.WhiteWins)
	assert.Equal(t, 1, stats.BlackWins)
	assert.Equal(t, 2830, stats.MaxElo)

	// 5. Duplicate Detection & Deletion
	dupGroups, err := app.FindDuplicateGames(scid.DuplicateCriteria{ExactMoves: true})
	require.NoError(t, err)
	assert.Len(t, dupGroups, 1)
	assert.Equal(t, 2, dupGroups[0].Count)

	deletedCount, err := app.DeleteDuplicateGames(scid.DuplicateCriteria{ExactMoves: true})
	require.NoError(t, err)
	assert.Equal(t, 1, deletedCount)

	// 6. Sort Database
	err = app.SortDatabase(scid.SortCriteria{Field: scid.SortByDate, Ascending: true})
	require.NoError(t, err)

	// 7. Verify Database
	report, err := app.VerifyDatabase()
	require.NoError(t, err)
	assert.Equal(t, 3, report.TotalGames)
	assert.Equal(t, 2, report.ActiveGames)
	assert.Equal(t, 1, report.DeletedGames)
	assert.Equal(t, "OK", report.Status)

	// 8. Compact Database
	compactStats, err := app.CompactDatabase()
	require.NoError(t, err)
	assert.Equal(t, 3, compactStats.OriginalGames)
	assert.Equal(t, 2, compactStats.CompactedGames)
	assert.Equal(t, 1, compactStats.PurgedGames)

	// 9. Batch PGN File Import
	filePGN := filepath.Join(tmpDir, "single.pgn")
	err = os.WriteFile(filePGN, []byte("[Event \"Extra\"]\n[White \"A\"]\n[Black \"B\"]\n[Result \"*\"]\n\n1. d4 *"), 0644)
	require.NoError(t, err)

	fileStats, err := app.ImportPGNFileBatch(filePGN)
	require.NoError(t, err)
	assert.Equal(t, 1, fileStats.Imported)
}

func TestAppOnlineSyncOperations(t *testing.T) {
	app := NewApp()
	app.startup(context.Background())

	tmpDir := t.TempDir()
	pm, err := online.NewProfileManager(filepath.Join(tmpDir, "profiles.json"))
	require.NoError(t, err)

	app.profileMgr = pm
	app.syncMgr = online.NewSyncManager(pm)

	// 1. Initial list should be empty
	profiles, err := app.GetSyncProfiles()
	require.NoError(t, err)
	assert.Empty(t, profiles)

	// 2. Save profile
	p := &online.Profile{
		Name:             "Grandmaster Sync",
		ChessComUsername: "testplayer",
		LichessUsername:  "testuser",
		AutoDeduplicate:  true,
	}
	err = app.SaveSyncProfile(p)
	require.NoError(t, err)
	assert.NotEmpty(t, p.ID)

	// 3. Query profiles
	profiles, err = app.GetSyncProfiles()
	require.NoError(t, err)
	assert.Len(t, profiles, 1)
	assert.Equal(t, "Grandmaster Sync", profiles[0].Name)

	retrieved, err := app.GetSyncProfile(p.ID)
	require.NoError(t, err)
	assert.Equal(t, p.Name, retrieved.Name)

	_, err = app.GetSyncProfile("nonexistent_id")
	assert.Error(t, err)

	// 4. Set up mock HTTP server for Lichess and Chess.com
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case strings.HasPrefix(path, "/api/games/user/"):
			w.Header().Set("Content-Type", "application/x-chess-pgn")
			_, _ = w.Write([]byte(`[Event "Lichess Round 1"]
[White "testuser"]
[Black "OpponentA"]
[Result "1-0"]

1. e4 e5 2. Nf3 Nc6 1-0

[Event "Lichess Round 2"]
[White "OpponentB"]
[Black "testuser"]
[Result "0-1"]

1. d4 d5 2. c4 e6 0-1
`))
		case strings.HasSuffix(path, "/games/archives"):
			w.Header().Set("Content-Type", "application/json")
			// Return one archive URL pointing back to this mock server
			archiveURL := fmt.Sprintf("http://%s/pub/player/testplayer/games/2026/10", r.Host)
			_, _ = w.Write([]byte(fmt.Sprintf(`{"archives":["%s"]}`, archiveURL)))
		case strings.HasSuffix(path, "/pgn"):
			w.Header().Set("Content-Type", "application/x-chess-pgn")
			_, _ = w.Write([]byte(`[Event "ChessCom Game"]
[White "testplayer"]
[Black "MasterC"]
[Result "1-0"]

1. e4 c5 2. Nf3 d6 1-0
`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	app.syncMgr.Lichess.BaseURL = server.URL
	app.syncMgr.ChessCom.BaseURL = server.URL

	// 5. Test Previews
	previewLichess, err := app.FetchLichessPreview("testuser", 2)
	require.NoError(t, err)
	assert.Contains(t, previewLichess, "Lichess Round 1")

	archivesChessCom, err := app.FetchChessComPreview("testplayer")
	require.NoError(t, err)
	assert.Len(t, archivesChessCom, 1)

	// 6. Test SyncOnlineProfile error cases
	_, err = app.SyncOnlineProfile("nonexistent_id")
	assert.Error(t, err)

	// When no database is open and no target path is set
	_, err = app.SyncOnlineProfile(p.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no target database specified")

	// 7. Sync into a new target database file
	targetDBPath := filepath.Join(tmpDir, "online_merged")
	p.TargetDatabase = targetDBPath
	err = app.SaveSyncProfile(p)
	require.NoError(t, err)

	syncRes, err := app.SyncOnlineProfile(p.ID)
	require.NoError(t, err)
	assert.Equal(t, 3, syncRes.TotalDownloaded)
	assert.Equal(t, 2, syncRes.LichessGames)
	assert.Equal(t, 1, syncRes.ChessComGames)
	assert.Equal(t, 3, syncRes.Imported)
	assert.Empty(t, syncRes.Errors)

	// Verify profile metrics were recorded
	updatedP, err := app.GetSyncProfile(p.ID)
	require.NoError(t, err)
	assert.Greater(t, updatedP.LastSyncTime, int64(0))
	assert.Equal(t, 3, updatedP.LastSyncGames)

	// Verify target database file actually exists and has 3 games
	verifyDB := scid.NewDatabase()
	err = verifyDB.Open(targetDBPath)
	require.NoError(t, err)
	assert.Equal(t, 3, verifyDB.NumGames())
	_ = verifyDB.Close()

	// 8. Sync into currently active open database
	activePath := filepath.Join(tmpDir, "active_base")
	_, err = app.CreateDatabase(activePath, "Active Test Base")
	require.NoError(t, err)

	pActive := &online.Profile{
		Name:            "Active Target Profile",
		LichessUsername: "testuser",
		TargetDatabase:  "", // should sync to app.db
	}
	err = app.SaveSyncProfile(pActive)
	require.NoError(t, err)

	activeRes, err := app.SyncOnlineProfile(pActive.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, activeRes.TotalDownloaded)
	assert.Equal(t, 2, app.db.NumGames())

	// 9. Delete Profile
	err = app.DeleteSyncProfile(p.ID)
	require.NoError(t, err)
	_, err = app.GetSyncProfile(p.ID)
	assert.Error(t, err)

	err = app.DeleteSyncProfile("nonexistent")
	assert.Error(t, err)
}

func TestAppEngineMultiPVAndOptions(t *testing.T) {
	app := NewApp()
	app.startup(context.Background())

	// Default MultiPV should be 1
	assert.Equal(t, 1, app.GetEngineMultiPV())

	// Set MultiPV before engine started
	err := app.SetEngineMultiPV(3)
	require.NoError(t, err)
	assert.Equal(t, 3, app.GetEngineMultiPV())

	err = app.SetEngineThreads(2)
	require.NoError(t, err)

	err = app.SetEngineHash(128)
	require.NoError(t, err)

	// If engine is available, test starting and applying settings
	engines := app.GetDiscoveredEngines()
	var availableEngine string
	for _, e := range engines {
		if e.Available && e.UCI {
			availableEngine = e.Name
			break
		}
	}

	if availableEngine != "" {
		err = app.StartEngine(availableEngine)
		require.NoError(t, err)
		defer func() { _ = app.StopEngine() }()

		assert.Equal(t, 3, app.GetEngineMultiPV())

		// Update MultiPV while engine is running
		err = app.SetEngineMultiPV(2)
		require.NoError(t, err)
		assert.Equal(t, 2, app.GetEngineMultiPV())

		err = app.SetEngineThreads(1)
		require.NoError(t, err)

		err = app.SetEngineHash(64)
		require.NoError(t, err)
	}
}


