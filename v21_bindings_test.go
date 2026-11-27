package main

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"newscid/pkg/chess"
	"newscid/pkg/database/scid"
)

func TestV21Bindings(t *testing.T) {
	app := NewApp()

	// Material search without open DB returns error
	_, err := app.SearchMaterialAndStructure(scid.MaterialSearchQuery{Pattern: scid.PatternIQP})
	assert.Error(t, err)

	// Open DB and search IQP
	dbPath := filepath.Join(t.TempDir(), "v21")
	_, err = app.CreateDatabase(dbPath, "V21")
	require.NoError(t, err)
	g, err := chess.ParsePGN(`[White "W"][Black "B"][Result "1-0"]
[FEN "r1bq1rk1/pp3ppp/2n1pn2/8/3P4/2N2N2/PP3PPP/R1BQ1RK1 w - - 0 12"]
12. Bg5 h6 13. Bh4 Be7 1-0`)
	require.NoError(t, err)
	_, err = app.db.SaveGame(g)
	require.NoError(t, err)

	hits, err := app.SearchMaterialAndStructure(scid.MaterialSearchQuery{Pattern: scid.PatternIQP, MinPly: 1})
	require.NoError(t, err)
	require.Len(t, hits, 1)

	// Endgame drills & offline move evaluation
	drills := app.GetEndgameDrills("all")
	assert.GreaterOrEqual(t, len(drills), 14)

	fb, err := app.EvaluateEndgameMove("1K1k4/1P6/8/8/8/8/r7/3R4 b - - 1 1", "win", "white", false)
	require.NoError(t, err)
	assert.NotEmpty(t, fb.BestReplyUCI)

	// Lichess Study invalid URL returns clean error
	_, err = app.ImportLichessStudy("bad-study", "", false)
	assert.Error(t, err)
}
