package tree_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"newscid/pkg/chess"
	"newscid/pkg/database/scid"
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

func TestSearchPositionInGames(t *testing.T) {
	pgn1 := `[White "A"][Black "B"][Result "1-0"] 1. e4 e5 2. Nf3 Nc6 1-0`
	pgn2 := `[White "C"][Black "D"][Result "0-1"] 1. d4 d5 2. c4 e6 0-1`
	pgn3 := `[White "E"][Black "F"][Result "1/2-1/2"] 1. e4 e5 2. Nf3 Nf6 1/2-1/2`

	var games []*chess.Game
	for _, p := range []string{pgn1, pgn2, pgn3} {
		g, err := chess.ParsePGN(p)
		require.NoError(t, err)
		games = append(games, g)
	}

	// Position after 1. e4 e5 2. Nf3
	b := chess.NewBoard()
	for _, san := range []string{"e4", "e5", "Nf3"} {
		m, err := b.ParseSAN(san)
		require.NoError(t, err)
		b.MakeMove(m)
	}

	matches := tree.SearchPositionInGames(b.FEN(), games)
	assert.Equal(t, []int{0, 2}, matches)
}

func TestBuildTreeFromDatabaseAndCoreFEN(t *testing.T) {
	// Closed database
	resClosed, err := tree.BuildTreeFromDatabase(nil, chess.StartingFEN, 50)
	require.NoError(t, err)
	assert.Empty(t, resClosed.Moves)

	// Open empty database
	db := scid.NewDatabase()
	db.IsOpen = true
	resOpen, err := tree.BuildTreeFromDatabase(db, chess.StartingFEN, 50)
	require.NoError(t, err)
	assert.Empty(t, resOpen.Moves)

	// Invalid FEN passed
	resInvalid := tree.BuildTreeFromGames("invalid fen string", nil)
	assert.NotNil(t, resInvalid)
	assert.Empty(t, resInvalid.Moves)

	matchesInvalid := tree.SearchPositionInGames("invalid fen string", nil)
	assert.Empty(t, matchesInvalid)

	// CoreFEN helper
	core := tree.CoreFEN(chess.StartingFEN)
	assert.Equal(t, "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq", core)
	shortFEN := "rnbqkbnr/pppppppp"
	assert.Equal(t, shortFEN, tree.CoreFEN(shortFEN))

	// Populated database with maxGames limit + Custom StartFEN + Terminal position match
	tmpBase := t.TempDir() + "/treetest"
	realDB := scid.NewDatabase()
	require.NoError(t, realDB.Create(tmpBase, "Tree Test"))
	defer realDB.Close()

	customFEN := "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq - 0 1"
	g1, err := chess.ParsePGN("[FEN \"" + customFEN + "\"]\n[Result \"1-0\"]\n\n1... e5 1-0")
	require.NoError(t, err)
	g2, err := chess.ParsePGN("[White \"A\"]\n[Black \"B\"]\n[Result \"0-1\"]\n\n1. e4 c5 0-1")
	require.NoError(t, err)

	_, err = realDB.SaveGame(g1)
	require.NoError(t, err)
	_, err = realDB.SaveGame(g2)
	require.NoError(t, err)

	dbTree, err := tree.BuildTreeFromDatabase(realDB, customFEN, 1)
	require.NoError(t, err)
	assert.Equal(t, 1, dbTree.TotalGames)
	require.Len(t, dbTree.Moves, 1)
	assert.Equal(t, "e5", dbTree.Moves[0].Move)

	// Terminal position match (after 1... e5 in g1)
	bEnd, _ := chess.ParseFEN(customFEN)
	mE5, _ := bEnd.ParseSAN("e5")
	bEnd.MakeMove(mE5)
	endTree := tree.BuildTreeFromGames(bEnd.FEN(), []*chess.Game{g1})
	assert.Equal(t, 1, endTree.TotalGames)
	assert.Empty(t, endTree.Moves)

	foundIndices := tree.SearchPositionInGames(customFEN, []*chess.Game{g1, g2})
	assert.Equal(t, []int{0, 1}, foundIndices)
}


