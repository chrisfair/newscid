package puzzleminer_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"newscid/pkg/chess"
	"newscid/pkg/database/scid"
	"newscid/pkg/puzzleminer"
)

func TestPuzzleMinerCheckmateDetection(t *testing.T) {
	// Game ending with Scholar's Mate
	pgn := `[Event "Casual Game"]
[Site "Local"]
[Date "2026.10.07"]
[White "Christopher"]
[Black "Opponent"]
[Result "1-0"]

1. e4 e5 2. Qh5 Nc6 3. Bc4 Nf6 4. Qxf7# 1-0`

	game, err := chess.ParsePGN(pgn)
	require.NoError(t, err)

	puzzles := puzzleminer.MineFromGames([]*chess.Game{game}, nil, puzzleminer.MiningFilter{
		MaxGames:      10,
		OnlyCheckmate: true,
	})

	require.Len(t, puzzles, 1)
	p := puzzles[0]
	assert.Equal(t, "Checkmate", p.Theme)
	assert.Equal(t, "w", p.Turn)
	assert.Equal(t, "f7", p.BestMoveUCI[2:4])
	assert.Contains(t, p.Description, "Christopher")
	assert.NotEmpty(t, p.StartFEN)
}

func TestPuzzleMinerBlackCheckmateAndFilters(t *testing.T) {
	// Fool's mate with Black winning
	pgn := `[Event "Fool's Mate"]
[Date "2026.01.01"]
[White "?"]
[Black "BlackPlayer"]
[Result "0-1"]

1. f3 e5 2. g4 Qh4# 0-1`

	game, err := chess.ParsePGN(pgn)
	require.NoError(t, err)

	// Filter by non-matching player
	puzzlesNoMatch := puzzleminer.MineFromGames([]*chess.Game{game, nil}, nil, puzzleminer.MiningFilter{
		PlayerName: "NonExistentPlayer",
	})
	assert.Empty(t, puzzlesNoMatch)

	// Filter matching BlackPlayer
	puzzlesMatch := puzzleminer.MineFromGames([]*chess.Game{game}, nil, puzzleminer.MiningFilter{
		PlayerName: "BlackPlayer",
	})
	require.Len(t, puzzlesMatch, 1)
	assert.Equal(t, "b", puzzlesMatch[0].Turn)
	assert.Contains(t, puzzlesMatch[0].Description, "BlackPlayer")
}

func TestPuzzleMinerMineFromDatabase(t *testing.T) {
	// 1. Closed database
	_, err := puzzleminer.MineFromDatabase(nil, nil, puzzleminer.MiningFilter{})
	assert.Error(t, err)

	// 2. Open database with games
	db := scid.NewDatabase()
	db.IsOpen = true
	db.Names = scid.NewNameBase()

	puzzles, err := puzzleminer.MineFromDatabase(db, nil, puzzleminer.MiningFilter{
		MaxGames: 10,
	})
	require.NoError(t, err)
	assert.Empty(t, puzzles)
}

