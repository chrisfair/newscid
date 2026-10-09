package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"newscid/pkg/chess"
	"newscid/pkg/database/scid"
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




