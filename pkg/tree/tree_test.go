package tree_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"newscid/pkg/chess"
	"newscid/pkg/tree"
)

func TestOpeningTreeCalculation(t *testing.T) {
	// Sample PGN games
	pgn1 := `[White "A"][Black "B"][Result "1-0"] 1. e4 e5 2. Nf3 Nc6 1-0`
	pgn2 := `[White "C"][Black "D"][Result "0-1"] 1. e4 c5 2. Nf3 d6 0-1`
	pgn3 := `[White "E"][Black "F"][Result "1/2-1/2"] 1. d4 Nf6 2. c4 e6 1/2-1/2`
	pgn4 := `[White "G"][Black "H"][Result "1-0"] 1. e4 e5 2. Bc4 Nf6 1-0`

	var games []*chess.Game
	for _, p := range []string{pgn1, pgn2, pgn3, pgn4} {
		g, err := chess.ParsePGN(p)
		require.NoError(t, err)
		games = append(games, g)
	}

	// Initial position tree
	treeRes := tree.BuildTreeFromGames(chess.StartingFEN, games)
	assert.Equal(t, 4, treeRes.TotalGames)
	require.NotEmpty(t, treeRes.Moves)

	// e4 was played in 3 games, d4 in 1 game
	assert.Equal(t, "e4", treeRes.Moves[0].Move)
	assert.Equal(t, 3, treeRes.Moves[0].Count)
	assert.Equal(t, 2, treeRes.Moves[0].WhiteWins)
	assert.Equal(t, 1, treeRes.Moves[0].BlackWins)

	assert.Equal(t, "d4", treeRes.Moves[1].Move)
	assert.Equal(t, 1, treeRes.Moves[1].Count)
	assert.Equal(t, 1, treeRes.Moves[1].Draws)

	// Position after 1. e4
	b := chess.NewBoard()
	mE4, _ := b.ParseSAN("e4")
	b.MakeMove(mE4)

	e4Tree := tree.BuildTreeFromGames(b.FEN(), games)
	assert.Equal(t, 3, e4Tree.TotalGames)
	// Black answers: e5 (2 games), c5 (1 game)
	assert.Equal(t, "e5", e4Tree.Moves[0].Move)
	assert.Equal(t, 2, e4Tree.Moves[0].Count)
	assert.Equal(t, "c5", e4Tree.Moves[1].Move)
	assert.Equal(t, 1, e4Tree.Moves[1].Count)
}
