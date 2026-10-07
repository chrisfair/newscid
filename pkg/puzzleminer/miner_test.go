package puzzleminer_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"newscid/pkg/chess"
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
