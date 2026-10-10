package scid

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"newscid/pkg/chess"
)

func TestMaterialSearch(t *testing.T) {
	// Test Board starting position
	b := chess.NewBoard()

	mc := DefaultMaterialCriteria()
	assert.False(t, mc.IsActive())
	assert.True(t, mc.MatchesBoard(b))

	// White Queen = 1, Black Queen = 1
	mc.WhiteQ = 1
	mc.BlackQ = 1
	assert.True(t, mc.IsActive())
	assert.True(t, mc.MatchesBoard(b))

	// Mismatched Queen count
	mc.WhiteQ = 2
	assert.False(t, mc.MatchesBoard(b))
	mc.WhiteQ = 1

	// Rook count
	mc.WhiteR = 2
	mc.BlackR = 2
	assert.True(t, mc.MatchesBoard(b))
	mc.BlackR = 0
	assert.False(t, mc.MatchesBoard(b))
	mc.BlackR = 2

	// Knight and Bishop count
	mc.WhiteB = 2
	mc.BlackB = 2
	mc.WhiteN = 2
	mc.BlackN = 2
	mc.WhiteP = 8
	mc.BlackP = 8
	assert.True(t, mc.MatchesBoard(b))

	// Imbalance: DiffQueens
	mcImb := DefaultMaterialCriteria()
	mcImb.DiffQueens = 0
	assert.True(t, mcImb.MatchesBoard(b))
	mcImb.DiffQueens = 1
	assert.False(t, mcImb.MatchesBoard(b))

	// Opposite-colored bishops test
	// Board with White bishop on c1 (c1 is dark: 2+0=2%2=0) and Black bishop on c8 (c8 is light: 2+7=9%2=1)
	opBoard := chess.NewBoard()
	opBoard.Clear()
	opBoard.SetPiece(chess.E1, chess.WK)
	opBoard.SetPiece(chess.E8, chess.BK)
	opBoard.SetPiece(chess.C1, chess.WB) // dark square
	opBoard.SetPiece(chess.C8, chess.BB) // light square

	mcOpp := DefaultMaterialCriteria()
	mcOpp.OppositeBishops = true
	assert.True(t, mcOpp.MatchesBoard(opBoard))

	// Same color bishops (c1 dark, f8 dark: 5+7=12%2=0)
	sameBoard := chess.NewBoard()
	sameBoard.Clear()
	sameBoard.SetPiece(chess.E1, chess.WK)
	sameBoard.SetPiece(chess.E8, chess.BK)
	sameBoard.SetPiece(chess.C1, chess.WB)
	sameBoard.SetPiece(chess.F8, chess.BB)
	assert.False(t, mcOpp.MatchesBoard(sameBoard))
}

func TestPawnStructureSearch(t *testing.T) {
	// 1. Carlsbad structure
	bCarlsbad := chess.NewBoard()
	bCarlsbad.Clear()
	bCarlsbad.SetPiece(chess.C3, chess.WP)
	bCarlsbad.SetPiece(chess.D4, chess.WP)
	bCarlsbad.SetPiece(chess.E3, chess.WP)
	bCarlsbad.SetPiece(chess.C6, chess.BP)
	bCarlsbad.SetPiece(chess.D5, chess.BP)
	bCarlsbad.SetPiece(chess.E6, chess.BP)

	assert.True(t, MatchesPawnStructure(bCarlsbad, "carlsbad"))
	assert.False(t, MatchesPawnStructure(bCarlsbad, "french_chain"))

	// 2. Isolani (IQP) on d4
	bIQP := chess.NewBoard()
	bIQP.Clear()
	bIQP.SetPiece(chess.D4, chess.WP)
	// c and e files are empty
	assert.True(t, MatchesPawnStructure(bIQP, "isolani"))
	assert.True(t, MatchesPawnStructure(bIQP, "iqp"))

	// Not isolated if c-pawn exists
	bIQP.SetPiece(chess.C2, chess.WP)
	assert.False(t, MatchesPawnStructure(bIQP, "isolani"))

	// 3. French Chain (d4, e5 vs d5, e6)
	bFrench := chess.NewBoard()
	bFrench.Clear()
	bFrench.SetPiece(chess.D4, chess.WP)
	bFrench.SetPiece(chess.E5, chess.WP)
	bFrench.SetPiece(chess.D5, chess.BP)
	bFrench.SetPiece(chess.E6, chess.BP)
	assert.True(t, MatchesPawnStructure(bFrench, "french_chain"))
	assert.True(t, MatchesPawnStructure(bFrench, "french"))

	// 4. Maroczy Bind (c4, e4 vs d6)
	bMaroczy := chess.NewBoard()
	bMaroczy.Clear()
	bMaroczy.SetPiece(chess.C4, chess.WP)
	bMaroczy.SetPiece(chess.E4, chess.WP)
	bMaroczy.SetPiece(chess.D6, chess.BP)
	assert.True(t, MatchesPawnStructure(bMaroczy, "maroczy"))
	assert.True(t, MatchesPawnStructure(bMaroczy, "maroczy_bind"))

	// 5. Stonewall (c3, d4, e3, f4)
	bStonewall := chess.NewBoard()
	bStonewall.Clear()
	bStonewall.SetPiece(chess.C3, chess.WP)
	bStonewall.SetPiece(chess.D4, chess.WP)
	bStonewall.SetPiece(chess.E3, chess.WP)
	bStonewall.SetPiece(chess.F4, chess.WP)
	assert.True(t, MatchesPawnStructure(bStonewall, "stonewall"))

	// 6. Unknown / empty
	assert.True(t, MatchesPawnStructure(bStonewall, ""))
	assert.False(t, MatchesPawnStructure(bStonewall, "unknown_structure"))
}

func TestAnnotationSearch(t *testing.T) {
	g := chess.NewGame()
	g.SetTag("Annotator", "Kasparov, G.")

	node1 := &chess.MoveNode{
		SAN:     "e4",
		Comment: "A standard king's pawn opening with tactical possibilities.",
	}
	node2 := &chess.MoveNode{
		SAN:     "c5",
		Comment: "The fighting Sicilian Defense.",
	}
	node1.Next = node2
	g.RootMove = node1

	// 1. Keyword search in commentary
	assert.True(t, MatchesGameAnnotation(g, "tactical", ""))
	assert.True(t, MatchesGameAnnotation(g, "sicilian", ""))
	assert.False(t, MatchesGameAnnotation(g, "blunder", ""))

	// 2. Annotator tag search
	assert.True(t, MatchesGameAnnotation(g, "", "Kasparov"))
	assert.False(t, MatchesGameAnnotation(g, "", "Fischer"))

	// 3. Combined keyword + annotator
	assert.True(t, MatchesGameAnnotation(g, "tactical", "Kasparov"))
	assert.False(t, MatchesGameAnnotation(g, "tactical", "Carlsen"))
}

func TestDatabaseAdvancedSearchIntegration(t *testing.T) {
	tmpDir := t.TempDir()
	basePath := filepath.Join(tmpDir, "adv_search_test")

	db := NewDatabase()
	require.NoError(t, db.Create(basePath, "Advanced Search Test DB"))
	defer db.Close()

	// Game 1: French Defense with commentary
	pgn1 := `[Event "World Cup"]
[Site "Khanty-Mansiysk"]
[Date "2011.09.01"]
[Round "1"]
[White "Grischuk, Alexander"]
[Black "Potkin, Vladimir"]
[Result "1-0"]
[Annotator "Shipov, S."]

1. e4 e6 2. d4 d5 3. e5 c5 4. c3 {Critical pawn chain defense} 1-0`

	// Game 2: Ruy Lopez with Opposite-Colored Bishops endgame
	pgn2 := `[Event "Candidates"]
[Site "London"]
[Date "2013.03.15"]
[Round "2"]
[White "Carlsen, Magnus"]
[Black "Aronian, Levon"]
[Result "1/2-1/2"]

1. e4 e5 2. Nf3 Nc6 3. Bb5 a6 4. Ba4 Nf6 5. O-O Be7 6. d4 exd4 1/2-1/2`

	g1, err := chess.ParsePGN(pgn1)
	require.NoError(t, err)
	_, err = db.SaveGame(g1)
	require.NoError(t, err)

	g2, err := chess.ParsePGN(pgn2)
	require.NoError(t, err)
	_, err = db.SaveGame(g2)
	require.NoError(t, err)

	// 1. Search by Pawn Structure: "french_chain"
	resFrench, countFrench, err := db.GetGameSummaries(0, 10, SearchCriteria{
		PawnStructure: "french_chain",
	})
	require.NoError(t, err)
	assert.Equal(t, 1, countFrench)
	require.Len(t, resFrench, 1)
	assert.Contains(t, resFrench[0].White, "Grischuk")

	// 2. Search by Annotation keyword: "pawn chain"
	resComm, countComm, err := db.GetGameSummaries(0, 10, SearchCriteria{
		CommentText: "pawn chain",
	})
	require.NoError(t, err)
	assert.Equal(t, 1, countComm)
	assert.Contains(t, resComm[0].White, "Grischuk")

	// 3. Search by Annotator: "Shipov"
	resAnn, countAnn, err := db.GetGameSummaries(0, 10, SearchCriteria{
		Annotator: "Shipov",
	})
	require.NoError(t, err)
	assert.Equal(t, 1, countAnn)
	require.Len(t, resAnn, 1)
	assert.Contains(t, resAnn[0].White, "Grischuk")

	// 4. Search by Material: 2 Queens (1 White, 1 Black)
	matCrit := DefaultMaterialCriteria()
	matCrit.WhiteQ = 1
	matCrit.BlackQ = 1
	resMat, countMat, err := db.GetGameSummaries(0, 10, SearchCriteria{
		Material: &matCrit,
	})
	require.NoError(t, err)
	assert.Equal(t, 2, countMat)
	assert.Len(t, resMat, 2)
}
