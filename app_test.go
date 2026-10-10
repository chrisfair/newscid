package main

import (
	"archive/zip"
	"bytes"
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
	"newscid/pkg/gameexport"
	"newscid/pkg/online"
	"newscid/pkg/persona"
	"newscid/pkg/tournament"
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

func TestAppNotationAndAnnotations(t *testing.T) {
	app := NewApp()
	app.startup(context.Background())

	// 1. Notation style getter/setter
	assert.Equal(t, "san", app.GetNotationStyle())
	err := app.SetNotationStyle("fan")
	require.NoError(t, err)
	assert.Equal(t, "fan", app.GetNotationStyle())

	err = app.SetNotationStyle("invalid_style")
	assert.Error(t, err)

	// 2. FormatMove
	formatted, err := app.FormatMove(chess.StartingFEN, "g1f3", "")
	require.NoError(t, err)
	assert.Equal(t, "♘f3", formatted)

	// 3. FormatMoveSequence
	seq, err := app.FormatMoveSequence(chess.StartingFEN, []string{"e2e4", "e7e5"}, "lan")
	require.NoError(t, err)
	assert.Equal(t, "1. e2-e4 e7-e5", seq)

	// 4. Game Comments and NAGs
	pgnSample := `[Event "World Championship"]
[Site "Reykjavik"]
[Date "1972.07.11"]
[Round "1"]
[White "Spassky, Boris V"]
[Black "Fischer, Robert J"]
[Result "1-0"]

1. d4 Nf6 2. c4 e6 1-0`

	_, err = app.ParsePGN(pgnSample)
	require.NoError(t, err)

	err = app.AddGameComment(1, "Strong queen pawn start")
	require.NoError(t, err)

	err = app.SetGameNAG(1, 1) // Good move !
	require.NoError(t, err)

	err = app.ClearGameNAGs(1)
	require.NoError(t, err)

	// Errors on non-existent ply
	err = app.AddGameComment(99, "Error ply")
	assert.Error(t, err)
	err = app.SetGameNAG(99, 1)
	assert.Error(t, err)
	err = app.ClearGameNAGs(99)
	assert.Error(t, err)
}

func TestAppPlayerImportAndSimulation(t *testing.T) {
	app := NewApp()
	app.startup(context.Background())

	tmpDir := t.TempDir()
	basePath := filepath.Join(tmpDir, "sim_test_db")
	db := scid.NewDatabase()
	err := db.Create(basePath, "Sim DB")
	require.NoError(t, err)

	// Add games for Kasparov
	g1, _ := chess.ParsePGN(`[Event "Match"]
[White "Kasparov, Garry"]
[Black "Deep Blue"]
[Result "1-0"]
[WhiteElo "2820"]

1. e4 c5 2. Nf3 d6 1-0`)
	_, _ = db.SaveGame(g1)
	db.Close()

	_, err = app.OpenDatabase(basePath)
	require.NoError(t, err)

	// Test GetDatabasePlayers
	players, err := app.GetDatabasePlayers()
	require.NoError(t, err)
	assert.Contains(t, players, "Kasparov, Garry")

	// Test GetPlayerPersonality
	personality, err := app.GetPlayerPersonality("Kasparov")
	require.NoError(t, err)
	assert.Equal(t, "Kasparov", personality.Name)
	assert.Equal(t, 2820, personality.PeakElo)

	// Test GetSimulatedMove (book move)
	simMove, err := app.GetSimulatedMove("Kasparov", chess.StartingFEN, true)
	require.NoError(t, err)
	assert.True(t, simMove.IsBook)
	assert.Equal(t, "e2e4", simMove.UCI)

	// Test GetSimulatedMove (out of book / fallback)
	simFallback, err := app.GetSimulatedMove("Kasparov", "8/8/8/8/8/8/8/4K2k w - - 0 1", true)
	require.NoError(t, err)
	assert.False(t, simFallback.IsBook)

	// Test ImportPlayerGames through mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-chess-pgn")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[Event "Online"]
[White "Carlsen"]
[Black "Nakamura"]
[Result "1-0"]

1. e4 e5 1-0`))
	}))
	defer server.Close()

	app.syncMgr.Lichess.BaseURL = server.URL
	resImport, err := app.ImportPlayerGames(online.ImportPlayerOptions{
		PlayerName:      "Magnus Carlsen",
		LichessUsername: "magnuscarlsen",
		TargetMode:      online.TargetActiveBase,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, resImport.Imported)
}

func TestAppPersonaOperations(t *testing.T) {
	app := NewApp()
	app.startup(context.Background())

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "personas.json")
	pMgr, err := persona.NewPersonaManager(configPath)
	require.NoError(t, err)
	app.personaMgr = pMgr

	// 1. Initial state
	personas, err := app.GetPersonas()
	require.NoError(t, err)
	assert.Empty(t, personas)

	// 2. SavePersona
	p, err := app.SavePersona(&persona.Persona{
		DisplayName:  "Christopher Fair",
		Identifiers:  []string{"Fair, Christopher", "ChristopherFair"},
		DatabasePath: filepath.Join(tempDir, "dedicated_persona"),
		Title:        "Club Champion",
		EstimatedElo: 2400,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, p.ID)
	assert.Equal(t, "Christopher Fair", p.DisplayName)
	assert.NotEmpty(t, p.DatabasePath)

	// 3. GetPersonas and GetPersona
	list, err := app.GetPersonas()
	require.NoError(t, err)
	assert.Len(t, list, 1)

	fetched, err := app.GetPersona(p.ID)
	require.NoError(t, err)
	assert.Equal(t, p.DisplayName, fetched.DisplayName)

	_, err = app.GetPersona("non-existent")
	assert.Error(t, err)

	// 4. AddPersonaIdentifier & RemovePersonaIdentifier
	pUpdated, err := app.AddPersonaIdentifier(p.ID, "QuenyenHighElf")
	require.NoError(t, err)
	assert.Contains(t, pUpdated.Identifiers, "QuenyenHighElf")

	pRemoved, err := app.RemovePersonaIdentifier(p.ID, "QuenyenHighElf")
	require.NoError(t, err)
	assert.NotContains(t, pRemoved.Identifiers, "QuenyenHighElf")

	// 5. Create active database with games for this persona
	dbPath := filepath.Join(tempDir, "active_db")
	err = app.db.Create(dbPath, "Active Base")
	require.NoError(t, err)

	g1 := chess.NewGame()
	g1.SetTag("White", "Fair, Christopher")
	g1.SetTag("Black", "Opponent 1")
	g1.SetTag("Date", "2023.01.10")
	g1.SetTag("Result", "1-0")
	g1.SetTag("WhiteElo", "2380")
	g1.SetTag("ECO", "B90")
	g1.RootMove = &chess.MoveNode{
		UCI: "e2e4",
		SAN: "e4",
		FEN: "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1",
	}
	_, err = app.db.SaveGame(g1)
	require.NoError(t, err)

	g2 := chess.NewGame()
	g2.SetTag("White", "Opponent 2")
	g2.SetTag("Black", "ChristopherFair")
	g2.SetTag("Date", "2023.02.15")
	g2.SetTag("Result", "1/2-1/2")
	g2.SetTag("BlackElo", "2420")
	g2.SetTag("ECO", "C50")
	g2.RootMove = &chess.MoveNode{
		UCI: "e2e4",
		SAN: "e4",
		FEN: "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1",
	}
	_, err = app.db.SaveGame(g2)
	require.NoError(t, err)

	// 6. SuggestPersonaIdentifiers
	suggestions, err := app.SuggestPersonaIdentifiers("Fair", []string{})
	require.NoError(t, err)
	assert.Contains(t, suggestions, "Fair, Christopher")
	assert.Contains(t, suggestions, "ChristopherFair")

	// 7. GetPersonaStatistics
	stats, err := app.GetPersonaStatistics(p.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, stats.TotalGames)
	assert.Equal(t, 1, stats.WhiteGames)
	assert.Equal(t, 1, stats.BlackGames)
	assert.Equal(t, 1, stats.Wins)
	assert.Equal(t, 1, stats.Draws)
	assert.Equal(t, 2420, stats.PeakElo)

	// 8. AggregatePersonaGames into dedicated persona database
	aggRes, err := app.AggregatePersonaGames(p.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, aggRes.TotalFound)
	assert.Equal(t, 2, aggRes.NewlyAdded)
	assert.Equal(t, 2, aggRes.TotalPersonaGames)

	// Aggregate again to verify deduplication
	aggRes2, err := app.AggregatePersonaGames(p.ID)
	require.NoError(t, err)
	assert.Equal(t, 0, aggRes2.NewlyAdded)
	assert.Equal(t, 2, aggRes2.DuplicatesSkipped)

	// 9. GetPersonaPersonality
	personality, err := app.GetPersonaPersonality(p.ID)
	require.NoError(t, err)
	assert.Equal(t, "Christopher Fair", personality.Name)
	assert.Equal(t, 2, personality.TotalGames)
	assert.Equal(t, 2400, personality.EstimatedElo)

	// 10. OpenPersonaDatabase
	info, err := app.OpenPersonaDatabase(p.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, info.NumGames)

	// 11. DeletePersona
	err = app.DeletePersona(p.ID)
	require.NoError(t, err)
	personasAfter, _ := app.GetPersonas()
	assert.Empty(t, personasAfter)

	// 12. Test FetchPersonaOnlineGames with mock Lichess server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-chess-pgn")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[Event "Online Persona Game"]
[Site "lichess.org"]
[Date "2024.03.15"]
[White "OnlineUser123"]
[Black "Grandmaster"]
[Result "1-0"]
[WhiteElo "2450"]
[BlackElo "2500"]

1. e4 e5 2. Nf3 1-0`))
	}))
	defer server.Close()

	app.syncMgr.Lichess.BaseURL = server.URL

	pOnline, err := app.SavePersona(&persona.Persona{
		DisplayName:     "Online Persona",
		LichessUsername: "OnlineUser123",
		DatabasePath:    filepath.Join(tempDir, "online_persona_db"),
	})
	require.NoError(t, err)
	assert.Contains(t, pOnline.Identifiers, "OnlineUser123")

	fetchRes, err := app.FetchPersonaOnlineGames(pOnline.ID, 10)
	require.NoError(t, err)
	assert.Equal(t, 1, fetchRes.TotalDownloaded)
	assert.Equal(t, 1, fetchRes.Imported)
	assert.Equal(t, 1, fetchRes.LichessGames)

	// Verify stats can now be calculated from dedicated database even if active DB has no games for this persona
	statsOnline, err := app.GetPersonaStatistics(pOnline.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, statsOnline.TotalGames)
	assert.Equal(t, 1, statsOnline.Wins)
	assert.Equal(t, 2450, statsOnline.PeakElo)

	// Error when persona has no online usernames
	pNoOnline, err := app.SavePersona(&persona.Persona{
		DisplayName: "No Online User",
	})
	require.NoError(t, err)
	_, err = app.FetchPersonaOnlineGames(pNoOnline.ID, 10)
	assert.Error(t, err)

	// Error when persona does not exist
	_, err = app.FetchPersonaOnlineGames("invalid-id", 10)
	assert.Error(t, err)
}

func TestAppPolyglotAndECO(t *testing.T) {
	app := NewApp()
	app.startup(context.Background())

	// Polyglot Books
	books := app.GetPolyglotBooks()
	t.Logf("Discovered polyglot books: %v", books)

	if len(books) > 0 {
		firstBook := books[0]
		moves, err := app.QueryPolyglotBook(chess.StartingFEN, firstBook)
		require.NoError(t, err)
		assert.NotEmpty(t, moves)

		pickBest, err := app.GetPolyglotBookMove(chess.StartingFEN, firstBook, true)
		require.NoError(t, err)
		require.NotNil(t, pickBest)
		assert.NotEmpty(t, pickBest.SAN)

		pickWeighted, err := app.GetPolyglotBookMove(chess.StartingFEN, firstBook, false)
		require.NoError(t, err)
		require.NotNil(t, pickWeighted)
	}

	// Empty / invalid book path
	emptyMoves, err := app.QueryPolyglotBook(chess.StartingFEN, "")
	require.NoError(t, err)
	assert.Empty(t, emptyMoves)

	_, err = app.QueryPolyglotBook("invalid fen string", "some_book.bin")
	assert.Error(t, err)

	// ECO Sections
	sections := app.GetECOSections()
	assert.Len(t, sections, 5)

	// ECO Search
	bEntries := app.SearchECO("B", "Sicilian", 10)
	assert.NotEmpty(t, bEntries)

	// ECO Classify
	classified := app.ClassifyMoves([]string{"e4", "c5", "Nf3", "d6", "d4", "cxd4", "Nxd4", "Nf6", "Nc3", "a6"})
	assert.True(t, strings.HasPrefix(classified.Code, "B90"))
}

func TestAppTablebaseAndSolitaire(t *testing.T) {
	app := NewApp()
	app.startup(context.Background())

	// 1. Test Tablebase probe (too many pieces error)
	_, err := app.ProbeTablebase(chess.StartingFEN)
	assert.Error(t, err) // 32 pieces > 7

	// 2. Solitaire with no game loaded
	_, err = app.StartGuessTheMove(0, "white")
	assert.Error(t, err)

	_, err = app.GetGuessTheMoveState()
	assert.Error(t, err)

	_, err = app.SubmitGuessTheMove("e4")
	assert.Error(t, err)

	_, _, err = app.GetGuessTheMoveHint()
	assert.Error(t, err)

	_, err = app.SkipGuessTheMove()
	assert.Error(t, err)

	// Load a game into activeGame via PGN
	pgn := `[Event "Paris Opera"]
[Site "Paris"]
[Date "1858.11.02"]
[Round "1"]
[White "Morphy, Paul"]
[Black "Duke of Brunswick"]
[Result "1-0"]

1. e4 e5 2. Nf3 d6 3. d4 Bg4 1-0`
	game, err := chess.ParsePGN(pgn)
	require.NoError(t, err)
	require.NotNil(t, game)
	app.activeGame = game

	// Start Guess-the-Move as White
	sess, err := app.StartGuessTheMove(0, "white")
	require.NoError(t, err)
	assert.NotNil(t, sess)
	assert.Equal(t, "Morphy, Paul", sess.White)
	assert.Equal(t, 3, sess.TotalTurns)

	// State check
	state, err := app.GetGuessTheMoveState()
	require.NoError(t, err)
	assert.Equal(t, sess.White, state.White)

	// Hint
	hint, level, err := app.GetGuessTheMoveHint()
	require.NoError(t, err)
	assert.Equal(t, 1, level)
	assert.Contains(t, hint, "Pawn")

	// Wrong guess
	fb, err := app.SubmitGuessTheMove("d2d4")
	require.NoError(t, err)
	assert.False(t, fb.IsMatch)
	assert.Equal(t, 0, fb.PointsEarned)

	// Correct guess (White plays e4, opponent replies e5)
	fb, err = app.SubmitGuessTheMove("e4")
	require.NoError(t, err)
	assert.True(t, fb.IsMatch)
	assert.Equal(t, 7, fb.PointsEarned) // 7 points because 1 hint used
	assert.Equal(t, "e5", fb.OpponentReplySAN)

	// Skip next turn (Nf3)
	skipFb, err := app.SkipGuessTheMove()
	require.NoError(t, err)
	assert.False(t, skipFb.IsMatch)
	assert.Equal(t, "Nf3", skipFb.ActualMoveSAN)
	assert.Equal(t, "d6", skipFb.OpponentReplySAN)

	// Final turn (d4)
	fb, err = app.SubmitGuessTheMove("d4")
	require.NoError(t, err)
	assert.True(t, fb.IsMatch)
	assert.True(t, fb.IsFinished)
	require.NotNil(t, fb.Summary)
	assert.Greater(t, fb.Summary.Score, 0)
}

func TestAppRatingTrackerAndDemographics(t *testing.T) {
	app := NewApp()
	app.startup(context.Background())

	// Error when no database is open
	_, err := app.GetRatingEvolution("Morphy, Paul")
	assert.Error(t, err)

	_, err = app.GetDatabaseDemographics()
	assert.Error(t, err)

	// Mock database
	db := scid.NewDatabase()
	db.IsOpen = true
	db.Names = scid.NewNameBase()
	w1 := db.Names.AddName(scid.NamePlayer, "Morphy, Paul")
	b1 := db.Names.AddName(scid.NamePlayer, "Anderssen, Adolf")
	ev := db.Names.AddName(scid.NameEvent, "Paris Match")

	db.Entries = append(db.Entries, &scid.IndexEntry{
		WhiteID:      w1,
		BlackID:      b1,
		EventID:      ev,
		WhiteElo:     2600,
		BlackElo:     2550,
		Result:       chess.ResultWhiteWins,
		Date:         "1858.12.20",
		EcoCode:      250,
		NumHalfMoves: 60,
	})

	app.db = db

	// Rating evolution
	evo, err := app.GetRatingEvolution("Morphy, Paul")
	require.NoError(t, err)
	assert.Equal(t, 1, evo.TotalGames)
	assert.Equal(t, 2600, evo.PeakRating)

	// Demographics
	demo, err := app.GetDatabaseDemographics()
	require.NoError(t, err)
	assert.Equal(t, 1, demo.TotalGames)
	assert.Equal(t, 1, demo.WhiteWins)
	assert.Equal(t, 2575, demo.AvgRating)
}

func TestAppUpdateGameHeaders(t *testing.T) {
	app := NewApp()
	app.startup(context.Background())

	_, err := app.UpdateGameHeaders(1, map[string]string{"White": "New Name"})
	assert.Error(t, err)

	db := scid.NewDatabase()
	db.IsOpen = true
	db.Names = scid.NewNameBase()
	w := db.Names.AddName(scid.NamePlayer, "Player A")
	b := db.Names.AddName(scid.NamePlayer, "Player B")
	db.Entries = append(db.Entries, &scid.IndexEntry{
		WhiteID: w,
		BlackID: b,
		Date:    "2020.01.01",
	})
	app.db = db

	// Update headers
	sum, err := app.UpdateGameHeaders(1, map[string]string{
		"White":    "Carlsen, Magnus",
		"WhiteElo": "2882",
		"ECO":      "C50",
		"Result":   "1-0",
	})
	require.NoError(t, err)
	assert.Equal(t, "Carlsen, Magnus", sum.White)
	assert.Equal(t, 2882, sum.WhiteElo)
	assert.Equal(t, "C50", sum.ECO)
	assert.Equal(t, chess.Result("1-0"), sum.Result)
}

func makeAppTWICZip(t *testing.T, filename, content string) []byte {
	t.Helper()
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)
	w, err := zw.Create(filename)
	require.NoError(t, err)
	_, err = w.Write([]byte(content))
	require.NoError(t, err)
	require.NoError(t, zw.Close())
	return buf.Bytes()
}

func TestAppTWICDownloader(t *testing.T) {
	app := NewApp()
	app.startup(context.Background())

	// 1. Errors when db not open
	_, err := app.DownloadTWICIssue(100)
	assert.Error(t, err)

	_, err = app.DownloadTWICRange(100, 102)
	assert.Error(t, err)

	// 2. Open test db
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "app_twic_test")
	db := scid.NewDatabase()
	err = db.Create(dbPath, "App TWIC Test")
	require.NoError(t, err)
	defer db.Close()
	app.db = db

	// 3. Mock TWIC server
	samplePGN := `[Event "World Cup"]
[Site "Baku"]
[Date "2023.08.15"]
[Round "1"]
[White "Carlsen, Magnus"]
[Black "Praggnanandhaa, R"]
[Result "1-0"]

1. d4 Nf6 2. c4 e6 1-0`

	zipData := makeAppTWICZip(t, "twic1500.pgn", samplePGN)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.Header().Set("Content-Type", "application/zip")
		w.WriteHeader(http.StatusOK)
		w.Write(zipData)
	}))
	defer server.Close()

	app.twicClient = &online.TWICClient{
		BaseURLTemplate: server.URL + "/twic%dg.zip",
		HTTPClient:      server.Client(),
		UserAgent:       "AppTest/1.0",
	}

	// Single issue download
	res, err := app.DownloadTWICIssue(1500)
	require.NoError(t, err)
	assert.Equal(t, 1500, res.IssueNumber)
	assert.Equal(t, 1, res.GamesImported)

	// Check issue exists
	exists, err := app.CheckTWICIssue(1500)
	require.NoError(t, err)
	assert.True(t, exists)

	// Range download
	batch, err := app.DownloadTWICRange(1501, 1502)
	require.NoError(t, err)
	assert.Equal(t, 2, batch.TotalImported)
	assert.Equal(t, 3, app.db.NumGames())
}

func TestAppEngineMatch(t *testing.T) {
	app := NewApp()
	app.startup(context.Background())

	assert.False(t, app.IsEngineMatchRunning())
	assert.False(t, app.StopEngineMatch())

	// Test DB for auto-save
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "engine_match_app_test")
	db := scid.NewDatabase()
	err := db.Create(dbPath, "Engine Match Test Base")
	require.NoError(t, err)
	defer db.Close()
	app.db = db

	cfg := tournament.EngineMatchConfig{
		Engine1: tournament.EngineConfig{
			Name: "EngineOne",
		},
		Engine2: tournament.EngineConfig{
			Name: "EngineTwo",
		},
		Rounds:          2,
		TimeControlMs:   50,
		MaxPlies:        8,
		AlternateColors: true,
		AutoSaveToDB:    true,
	}

	summary, err := app.StartEngineMatch(cfg)
	require.NoError(t, err)
	assert.Equal(t, 2, len(summary.Games))
	assert.Equal(t, 2, summary.Rounds)
	assert.Equal(t, 2, app.db.NumGames())
	assert.False(t, app.IsEngineMatchRunning())
}

func TestAppStandardizePlayerNames(t *testing.T) {
	app := NewApp()
	app.startup(context.Background())

	// No database open
	_, err := app.StandardizePlayerNames()
	assert.Error(t, err)

	// Create temporary database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "spellcheck_app_test")
	db := scid.NewDatabase()
	err = db.Create(dbPath, "Spellcheck Test Base")
	require.NoError(t, err)
	defer db.Close()
	app.db = db

	// Import game with non-standard names
	pgn := `[Event "World Championship"]
[Site "London"]
[Date "2018.11.09"]
[Round "1"]
[White "Magnus Carlsen"]
[Black "Caruana, F."]
[Result "1/2-1/2"]

1. e4 e5 1/2-1/2`

	_, err = app.ImportPGNBatch(pgn)
	require.NoError(t, err)

	summary, err := app.StandardizePlayerNames()
	require.NoError(t, err)
	assert.NotNil(t, summary)
	assert.True(t, summary.CorrectionsMade >= 2)
	assert.Equal(t, 1, summary.GamesUpdated)
}

func TestAppExportGameHTMLAndLaTeX(t *testing.T) {
	app := NewApp()
	app.startup(context.Background())

	// Errors when no game / DB
	_, err := app.ExportCurrentGameHTML(gameexport.HTMLExportOptions{})
	assert.Error(t, err)
	_, err = app.ExportCurrentGameLaTeX(gameexport.LaTeXExportOptions{})
	assert.Error(t, err)
	_, err = app.ExportGameHTML(1, gameexport.HTMLExportOptions{})
	assert.Error(t, err)
	_, err = app.ExportGameLaTeX(1, gameexport.LaTeXExportOptions{})
	assert.Error(t, err)

	// Create test DB
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "export_app_test")
	db := scid.NewDatabase()
	err = db.Create(dbPath, "Export Test Base")
	require.NoError(t, err)
	defer db.Close()
	app.db = db

	pgn := `[Event "Candidates Tournament"]
[Site "Madrid"]
[Date "2022.06.17"]
[Round "1"]
[White "Nepomniachtchi, Ian"]
[Black "Ding, Liren"]
[Result "1-0"]

1. e4 e5 2. Nf3 Nc6 3. Bb5 a6 1-0`

	_, err = app.ImportPGNBatch(pgn)
	require.NoError(t, err)

	_, err = app.GetGame(1)
	require.NoError(t, err)

	// Test ExportCurrentGameHTML
	htmlCur, err := app.ExportCurrentGameHTML(gameexport.HTMLExportOptions{
		Theme:          "light",
		IncludeDiagram: true,
	})
	require.NoError(t, err)
	assert.Contains(t, htmlCur, "Nepomniachtchi, Ian")
	assert.Contains(t, htmlCur, "Ding, Liren")
	assert.Contains(t, htmlCur, "<svg")

	// Test ExportGameHTML by number
	htmlNum, err := app.ExportGameHTML(1, gameexport.HTMLExportOptions{
		Theme:          "dark",
		IncludeDiagram: false,
	})
	require.NoError(t, err)
	assert.Contains(t, htmlNum, "Nepomniachtchi, Ian")

	// Test ExportCurrentGameLaTeX
	latexCur, err := app.ExportCurrentGameLaTeX(gameexport.LaTeXExportOptions{
		IncludeDiagram: true,
	})
	require.NoError(t, err)
	assert.Contains(t, latexCur, "\\documentclass")
	assert.Contains(t, latexCur, "Nepomniachtchi, Ian")
	assert.Contains(t, latexCur, "\\chessboard")

	// Test ExportGameLaTeX by number
	latexNum, err := app.ExportGameLaTeX(1, gameexport.LaTeXExportOptions{
		IncludeDiagram: false,
	})
	require.NoError(t, err)
	assert.Contains(t, latexNum, "Nepomniachtchi, Ian")

	// Test SaveExportFile
	exportFilePath := filepath.Join(tmpDir, "exported_game.html")
	err = app.SaveExportFile(exportFilePath, htmlCur)
	require.NoError(t, err)
	savedBytes, err := os.ReadFile(exportFilePath)
	require.NoError(t, err)
	assert.Equal(t, htmlCur, string(savedBytes))

	// SaveExportFile error with empty path
	err = app.SaveExportFile("", htmlCur)
	assert.Error(t, err)
}

