package scid_test

import (
	"os"
	"path/filepath"
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
