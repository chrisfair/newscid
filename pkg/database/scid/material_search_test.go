package scid

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"newscid/pkg/chess"
)

func TestMaterialPatternEnumJSON(t *testing.T) {
	patterns := []struct {
		p MaterialPattern
		s string
	}{
		{PatternAny, "any"},
		{PatternOppositeBishops, "opposite_bishops"},
		{PatternSameBishops, "same_bishops"},
		{PatternQueenVsRookMinor, "queen_vs_rook_minor"},
		{PatternExchangeImbalance, "exchange_imbalance"},
		{PatternTwoBishops, "two_bishops"},
		{PatternIQP, "iqp"},
		{PatternCarlsbad, "carlsbad"},
		{PatternMaroczyBind, "maroczy_bind"},
		{PatternWinningSacrifice, "winning_sacrifice"},
	}

	for _, tc := range patterns {
		assert.Equal(t, tc.s, tc.p.String())
		assert.Equal(t, tc.p, ParseMaterialPattern(tc.s))
		raw, err := json.Marshal(tc.p)
		require.NoError(t, err)
		var decoded MaterialPattern
		require.NoError(t, json.Unmarshal(raw, &decoded))
		assert.Equal(t, tc.p, decoded)
	}
	assert.Equal(t, "any", MaterialPattern(99).String())
	var bad MaterialPattern
	assert.Error(t, json.Unmarshal([]byte(`123`), &bad))
}

func TestSearchMaterialAndStructure(t *testing.T) {
	db := NewDatabase()
	_, err := db.SearchMaterialAndStructure(MaterialSearchQuery{})
	assert.Error(t, err)

	require.NoError(t, db.Create(filepath.Join(t.TempDir(), "mat"), "Mat"))
	defer db.Close()

	// Game 1: Winning Queen sacrifice / exchange drop by White (drops knight + wins)
	gSac, err := chess.ParsePGN("[White \"Tal\"]\n[Black \"Club\"]\n[Result \"1-0\"]\n\n1. e4 e5 2. Nf3 Nc6 3. Bc4 Bc5 4. Bxf7+ Kxf7 5. Nxe5+ Nxe5 6. Qh5+ Ng6 7. Qxc5 d6 1-0")
	require.NoError(t, err)
	_, err = db.SaveGame(gSac)
	require.NoError(t, err)

	// Game 2: Opposite-colored bishop endgame from custom StartFEN (Bc4 light vs bc1 dark)
	gOCB, err := chess.ParsePGN("[White \"EndW\"]\n[Black \"EndB\"]\n[Result \"1/2-1/2\"]\n[FEN \"4k3/5pp1/8/8/2B5/8/5PP1/2b3K1 w - - 0 40\"]\n\n40. Kf1 Ke7 41. Ke1 Ba3 1/2-1/2")
	require.NoError(t, err)
	_, err = db.SaveGame(gOCB)
	require.NoError(t, err)

	// Game 3: IQP (White d4 isolated pawn)
	gIQP, err := chess.ParsePGN("[White \"IQPW\"]\n[Black \"IQPB\"]\n[Result \"1-0\"]\n[FEN \"r1bq1rk1/pp3ppp/2n1pn2/8/3P4/2N2N2/PP3PPP/R1BQ1RK1 w - - 0 12\"]\n\n12. Bg5 h6 13. Bh4 Be7 1-0")
	require.NoError(t, err)
	_, err = db.SaveGame(gIQP)
	require.NoError(t, err)

	// Opposite-colored bishops
	hits, err := db.SearchMaterialAndStructure(MaterialSearchQuery{
		Pattern: PatternOppositeBishops,
		MinPly:  1,
	})
	require.NoError(t, err)
	require.Len(t, hits, 1)
	assert.Equal(t, "EndW", hits[0].White)

	// IQP
	iqpHits, err := db.SearchMaterialAndStructure(MaterialSearchQuery{
		Pattern: PatternIQP,
		MinPly:  1,
	})
	require.NoError(t, err)
	require.Len(t, iqpHits, 1)
	assert.Equal(t, "IQPW", iqpHits[0].White)

	// Winning sacrifice
	sacHits, err := db.SearchMaterialAndStructure(MaterialSearchQuery{
		Pattern:        PatternWinningSacrifice,
		MinPly:         4,
		MinSacrificeCp: 200,
	})
	require.NoError(t, err)
	require.Len(t, sacHits, 1)
	assert.Equal(t, "Tal", sacHits[0].White)
	assert.Contains(t, sacHits[0].Summary, "sacrificed")

	// Custom piece count filter (exact 1 bishop, 0 queens, 0 rooks for both sides)
	f := UnconstrainedPieceFilter()
	f.MinQ, f.MaxQ = 0, 0
	f.MinR, f.MaxR = 0, 0
	f.MinB, f.MaxB = 1, 1
	customHits, err := db.SearchMaterialAndStructure(MaterialSearchQuery{
		Pattern:        PatternAny,
		White:          f,
		Black:          f,
		UsePieceCounts: true,
		MinPly:         1,
	})
	require.NoError(t, err)
	require.Len(t, customHits, 1)
	assert.Equal(t, "EndW", customHits[0].White)
}
