package scid_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"newscid/pkg/chess"
	"newscid/pkg/database/scid"
)

func TestOpenTacticsDatabase(t *testing.T) {
	tacticsPath := "/usr/share/scid/bases/tactics"
	if _, err := os.Stat(tacticsPath + ".si4"); os.IsNotExist(err) {
		t.Skip("Tactics base not found on system, skipping real base test")
	}

	db := scid.NewDatabase()
	err := db.Open(tacticsPath)
	require.NoError(t, err)
	defer db.Close()

	assert.True(t, db.IsOpen)
	assert.Equal(t, 928, db.NumGames())

	// Test Game 1 summary
	sum1, err := db.GetGameSummary(1)
	require.NoError(t, err)
	assert.Equal(t, "Fischer", sum1.White)
	assert.Equal(t, "Andersson", sum1.Black)
	assert.Equal(t, chess.ResultWhiteWins, sum1.Result)

	// Test Game 1 full game decoding
	g1, err := db.GetGame(1)
	require.NoError(t, err)
	assert.Equal(t, "Fischer", g1.Tag("White"))
	assert.Equal(t, "Andersson", g1.Tag("Black"))
	assert.Equal(t, chess.ResultWhiteWins, g1.Result)
	assert.NotNil(t, g1.RootMove)
	// Verify Fischer's tactical move
	assert.Equal(t, "Qxe6", g1.RootMove.SAN)

	// Test Search games by player Fischer
	results, total, err := db.GetGameSummaries(0, 10, scid.SearchCriteria{
		Player: "Fischer",
	})
	require.NoError(t, err)
	assert.Greater(t, total, 0)
	assert.NotEmpty(t, results)
}

func TestOpenOtherSystemBases(t *testing.T) {
	bases := []string{
		"/usr/share/scid/bases/matein1",
		"/usr/share/scid/bases/matein2",
		"/usr/share/scid/bases/endings",
	}

	for _, basePath := range bases {
		if _, err := os.Stat(basePath + ".si4"); os.IsNotExist(err) {
			continue
		}
		db := scid.NewDatabase()
		err := db.Open(basePath)
		require.NoError(t, err, "Failed to open base %s", basePath)
		assert.Greater(t, db.NumGames(), 0)

		// Test loading first game
		g, err := db.GetGame(1)
		require.NoError(t, err, "Failed to load game 1 from base %s", basePath)
		assert.NotNil(t, g)
		db.Close()
	}
}

func TestCreateAndModifyDatabaseRoundtrip(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "scid_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "testbase")
	db := scid.NewDatabase()
	err = db.Create(dbPath, "Test Roundtrip Base")
	require.NoError(t, err)
	assert.Equal(t, 0, db.NumGames())

	// Create and save game 1
	g1 := chess.NewGame()
	g1.SetTag("White", "Carlsen, Magnus")
	g1.SetTag("Black", "Nakamura, Hikaru")
	g1.SetTag("Date", "2024.01.15")
	g1.SetTag("Event", "Speed Chess Championship")
	g1.SetTag("Site", "Internet")
	g1.Result = chess.ResultWhiteWins

	// Play 1. e4 e5 2. Nf3 Nc6
	b := chess.NewBoard()
	m1, _ := b.ParseSAN("e4"); b.MakeMove(m1)
	m2, _ := b.ParseSAN("e5"); b.MakeMove(m2)
	m3, _ := b.ParseSAN("Nf3"); b.MakeMove(m3)
	m4, _ := b.ParseSAN("Nc6"); b.MakeMove(m4)

	n1 := &chess.MoveNode{Move: m1, SAN: "e4", UCI: m1.String()}
	n2 := &chess.MoveNode{Move: m2, SAN: "e5", UCI: m2.String()}
	n3 := &chess.MoveNode{Move: m3, SAN: "Nf3", UCI: m3.String()}
	n4 := &chess.MoveNode{Move: m4, SAN: "Nc6", UCI: m4.String()}
	n1.Next = n2
	n2.Next = n3
	n3.Next = n4
	g1.RootMove = n1
	g1.PlyCount = 4

	id, err := db.SaveGame(g1)
	require.NoError(t, err)
	assert.Equal(t, 1, id)
	assert.Equal(t, 1, db.NumGames())

	// Close database and reopen from disk
	err = db.Close()
	require.NoError(t, err)

	db2 := scid.NewDatabase()
	err = db2.Open(dbPath)
	require.NoError(t, err)
	defer db2.Close()

	assert.Equal(t, 1, db2.NumGames())
	loadedGame, err := db2.GetGame(1)
	require.NoError(t, err)
	assert.Equal(t, "Carlsen, Magnus", loadedGame.Tag("White"))
	assert.Equal(t, "Nakamura, Hikaru", loadedGame.Tag("Black"))
	assert.Equal(t, chess.ResultWhiteWins, loadedGame.Result)
	assert.Equal(t, "e4", loadedGame.RootMove.SAN)
	assert.Equal(t, "e5", loadedGame.RootMove.Next.SAN)
	assert.Equal(t, "Nf3", loadedGame.RootMove.Next.Next.SAN)
	assert.Equal(t, "Nc6", loadedGame.RootMove.Next.Next.Next.SAN)

	// Test Deleting Game
	err = db2.DeleteGame(1, true)
	require.NoError(t, err)
	sum, err := db2.GetGameSummary(1)
	require.NoError(t, err)
	assert.True(t, sum.Deleted)
}

func TestOpenUserDatabases(t *testing.T) {
	userBase := "/home/christopher/ChessDataBases/ChristopherFair/ChristopherFair"
	if _, err := os.Stat(userBase + ".si4"); os.IsNotExist(err) {
		t.Skip("ChristopherFair database not found")
	}

	db := scid.NewDatabase()
	err := db.Open(userBase)
	require.NoError(t, err)
	defer db.Close()

	t.Logf("Successfully opened user base %s: %d games", userBase, db.NumGames())
	assert.Greater(t, db.NumGames(), 0)

	// Read summary of game 1
	sum, err := db.GetGameSummary(1)
	require.NoError(t, err)
	t.Logf("Game 1: %s vs %s [%s]", sum.White, sum.Black, sum.Result)
	assert.NotEmpty(t, sum.White)

	g1, err := db.GetGame(1)
	require.NoError(t, err)
	assert.NotNil(t, g1.RootMove)
	t.Logf("Game 1 first move: %s", g1.RootMove.SAN)
}

func TestParseRecentDat(t *testing.T) {
	data := `# Scid vs. PC recent files list
set recentFiles(limit) 10
set recentFiles(data) {/path/to/base1.si4 /path/to/base2.si4 /path/to/game.pgn}
`
	paths := scid.ParseRecentDat(strings.NewReader(data))
	require.Len(t, paths, 3)
	assert.Equal(t, "/path/to/base1.si4", paths[0])
	assert.Equal(t, "/path/to/base2.si4", paths[1])
	assert.Equal(t, "/path/to/game.pgn", paths[2])
}

func TestDiscoverScidDatabases(t *testing.T) {
	dbs := scid.DiscoverScidDatabases()
	t.Logf("Discovered %d databases", len(dbs))
	for _, db := range dbs {
		t.Logf("[%s] %s (%s, %d games): %s", db.Category, db.Name, db.SizeStr, db.NumGames, db.BasePath)
	}

	assert.NotEmpty(t, dbs)
}

func TestAdvancedHeaderSearch(t *testing.T) {
	tacticsPath := "/usr/share/scid/bases/tactics"
	if _, err := os.Stat(tacticsPath + ".si4"); os.IsNotExist(err) {
		t.Skip("Tactics base not found on system, skipping search test")
	}

	db := scid.NewDatabase()
	err := db.Open(tacticsPath)
	require.NoError(t, err)
	defer db.Close()

	// Search by Player
	results, total, err := db.GetGameSummaries(0, 50, scid.SearchCriteria{
		Player: "Fischer",
	})
	require.NoError(t, err)
	assert.Greater(t, total, 0)
	for _, g := range results {
		assert.True(t, strings.Contains(strings.ToLower(g.White), "fischer") || strings.Contains(strings.ToLower(g.Black), "fischer"))
	}
	assert.Equal(t, total, db.CountMatches(scid.SearchCriteria{Player: "Fischer"}))

	// Search by Specific Color (White = Fischer)
	resWhite, totalWhite, err := db.GetGameSummaries(0, 50, scid.SearchCriteria{
		White: "Fischer",
	})
	require.NoError(t, err)
	for _, g := range resWhite {
		assert.Contains(t, strings.ToLower(g.White), "fischer")
	}
	assert.LessOrEqual(t, totalWhite, total)

	// Search by Result (1-0)
	resWins, totalWins, err := db.GetGameSummaries(0, 10, scid.SearchCriteria{
		Result: "1-0",
	})
	require.NoError(t, err)
	assert.Greater(t, totalWins, 0)
	for _, g := range resWins {
		assert.Equal(t, chess.ResultWhiteWins, g.Result)
	}

	// Search by ECO Range (e.g. B00 to B99)
	resEco, totalEco, err := db.GetGameSummaries(0, 20, scid.SearchCriteria{
		EcoFrom: "B00",
		EcoTo:   "B99",
	})
	require.NoError(t, err)
	for _, g := range resEco {
		if g.ECO != "" {
			assert.True(t, g.ECO >= "B00" && g.ECO <= "B99", "ECO %s should be in [B00, B99]", g.ECO)
		}
	}
	assert.Equal(t, totalEco, db.CountMatches(scid.SearchCriteria{EcoFrom: "B00", EcoTo: "B99"}))

	// Search by Move count (MinMoves = 10, MaxMoves = 30)
	resMoves, totalMoves, err := db.GetGameSummaries(0, 20, scid.SearchCriteria{
		MinMoves: 10,
		MaxMoves: 30,
	})
	require.NoError(t, err)
	for _, g := range resMoves {
		assert.GreaterOrEqual(t, g.Moves, 10)
		assert.LessOrEqual(t, g.Moves, 30)
	}
	assert.Equal(t, totalMoves, db.CountMatches(scid.SearchCriteria{MinMoves: 10, MaxMoves: 30}))

	// Search by Year (YearFrom = 1970, YearTo = 1980)
	resYears, totalYears, err := db.GetGameSummaries(0, 20, scid.SearchCriteria{
		YearFrom: 1970,
		YearTo:   1980,
	})
	require.NoError(t, err)
	for _, g := range resYears {
		if len(g.Date) >= 4 {
			var y int
			fmt.Sscanf(g.Date[:4], "%d", &y)
			if y > 0 {
				assert.GreaterOrEqual(t, y, 1970)
				assert.LessOrEqual(t, y, 1980)
			}
		}
	}
	assert.Equal(t, totalYears, db.CountMatches(scid.SearchCriteria{YearFrom: 1970, YearTo: 1980}))
}

func TestMergeFromDatabases(t *testing.T) {
	tmpDir := t.TempDir()

	// Create target database with 1 game
	targetPath := filepath.Join(tmpDir, "target_mega")
	targetDB := scid.NewDatabase()
	require.NoError(t, targetDB.Create(targetPath, "Target Mega DB"))
	defer targetDB.Close()

	g1, err := chess.ParsePGN(`[Event "Wijk"]
[Site "https://lichess.org/gameA"]
[Date "2026.01.10"]
[White "Carlsen, Magnus"]
[Black "Nakamura, Hikaru"]
[Result "1-0"]

1. e4 e5 2. Nf3 Nc6 3. Bb5 a6 1-0`)
	require.NoError(t, err)
	_, err = targetDB.ImportGames([]*chess.Game{g1})
	require.NoError(t, err)
	assert.Equal(t, 1, targetDB.NumGames())

	// Create source .si4 database with 1 duplicate (g1) and 1 brand new game (g2)
	srcSi4Path := filepath.Join(tmpDir, "local_persona")
	srcDB := scid.NewDatabase()
	require.NoError(t, srcDB.Create(srcSi4Path, "Local Persona DB"))
	g2, err := chess.ParsePGN(`[Event "Candidates"]
[Site "https://lichess.org/gameB"]
[Date "2026.02.15"]
[White "Caruana, Fabiano"]
[Black "Carlsen, Magnus"]
[Result "0-1"]

1. d4 Nf6 2. c4 e6 3. Nc3 Bb4 0-1`)
	require.NoError(t, err)
	_, err = srcDB.ImportGames([]*chess.Game{g1, g2})
	require.NoError(t, err)
	require.NoError(t, srcDB.Close())

	// Create source .pgn file with 1 duplicate (g2) and 1 brand new game (g3)
	srcPgnPath := filepath.Join(tmpDir, "local_club.pgn")
	pgnContent := `[Event "Candidates"]
[Site "https://lichess.org/gameB"]
[Date "2026.02.15"]
[White "Caruana, Fabiano"]
[Black "Carlsen, Magnus"]
[Result "0-1"]

1. d4 Nf6 2. c4 e6 3. Nc3 Bb4 0-1

[Event "Club Championship"]
[Site "https://lichess.org/gameC"]
[Date "2026.03.01"]
[White "Firouzja, Alireza"]
[Black "Gukesh, D"]
[Result "1/2-1/2"]

1. c4 c5 2. Nf3 Nf6 1/2-1/2
`
	require.NoError(t, os.WriteFile(srcPgnPath, []byte(pgnContent), 0o644))

	// Merge both sources (plus self path which should be ignored) into targetDB
	mergeStats, err := targetDB.MergeFromDatabases([]string{targetPath, srcSi4Path, srcPgnPath}, nil)
	require.NoError(t, err)
	assert.Equal(t, 2, mergeStats.SourcesProcessed)
	assert.Equal(t, 2, mergeStats.Imported)
	assert.Equal(t, 2, mergeStats.DuplicatesSkipped)
	assert.Equal(t, 3, targetDB.NumGames())

	// Verify all 3 games load and decode cleanly from the updated target database
	reopened := scid.NewDatabase()
	require.NoError(t, reopened.Open(targetPath))
	defer reopened.Close()
	assert.Equal(t, 3, reopened.NumGames())
	for id := 1; id <= 3; id++ {
		gm, err := reopened.GetGame(id)
		require.NoError(t, err)
		assert.NotNil(t, gm.RootMove)
	}
}
