package stats

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"newscid/pkg/chess"
	"newscid/pkg/database/scid"
)

func TestAnalyzeDatabaseTimeManagementParallel(t *testing.T) {
	base := filepath.Join(t.TempDir(), "tm_parallel")
	db := scid.NewDatabase()
	require.NoError(t, db.Create(base, "TM Parallel"))
	defer db.Close()

	pgn := `[Event "Blitz"]
[Site "Online"]
[Date "2026.01.01"]
[White "ParallelHero"]
[Black "Rival"]
[Result "1-0"]
[TimeControl "180+0"]

1. e4 {[%clk 0:02:58]} e5 {[%clk 0:02:57]} 2. Nf3 {[%clk 0:02:50]} Nc6 {[%clk 0:02:45]} 3. Bb5 {[%clk 0:02:40]} a6 {[%clk 0:02:30]} 1-0`

	for i := 0; i < 8; i++ {
		g, err := chess.ParsePGN(pgn)
		require.NoError(t, err)
		_, err = db.SaveGame(g)
		require.NoError(t, err)
	}

	dossier, err := AnalyzePlayerTimeManagement(db, "ParallelHero", 20)
	require.NoError(t, err)
	assert.Equal(t, 8, dossier.GamesAnalyzed)
	assert.Equal(t, 8, dossier.GamesWithClock)
	assert.Greater(t, dossier.AverageMoveTimeSec, 0.0)
}
