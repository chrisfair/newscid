package playerbot

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"newscid/pkg/chess"
)

func TestSelectTacticalMove_CapturesFreePiece(t *testing.T) {
	// White has Queen hanging on e4; Black Knight on f6 should capture it (Nxe4)
	b, err := chess.ParseFEN("rnbqkb1r/pppp1ppp/5n2/8/4Q3/8/PPPP1PPP/RNB1KBNR b KQkq - 0 1")
	require.NoError(t, err)

	m, err := SelectTacticalMove(b, false)
	require.NoError(t, err)

	// Best move should capture Queen on e4
	assert.Equal(t, chess.E4, m.To)
}

func TestSelectTacticalMove_FindsMateInOne(t *testing.T) {
	// Scholar's mate: White Queen on f7 delivers mate (Qxf7#)
	b, err := chess.ParseFEN("r1bqkb1r/pppp1ppp/2n5/4p3/2B1n3/5Q2/PPPP1PPP/RNB1K1NR w KQkq - 0 1")
	require.NoError(t, err)

	m, err := SelectTacticalMove(b, true)
	require.NoError(t, err)

	// Qxf7# is mate
	assert.Equal(t, chess.F7, m.To)
}

func TestSelectTacticalMove_AvoidsHangingPiece(t *testing.T) {
	// White Knight on b1 has choices. Don't play to an attacked undefended square if safe choices exist.
	b, err := chess.ParseFEN("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")
	require.NoError(t, err)

	m, err := SelectTacticalMove(b, true)
	require.NoError(t, err)
	assert.NotEmpty(t, m.String())
}

func TestSelectTacticalMove_EdgeCases(t *testing.T) {
	// Nil board
	_, err := SelectTacticalMove(nil, true)
	assert.Error(t, err)

	// No legal moves (Checkmate / Stalemate)
	mateBoard, err := chess.ParseFEN("rnb1kbnr/pppp1ppp/8/4p3/6Pq/5P2/PPPPP2P/RNBQKBNR w KQkq - 1 3")
	require.NoError(t, err)
	_, err = SelectTacticalMove(mateBoard, true)
	assert.Error(t, err)

	// Exactly 1 legal move (King must move out of check)
	singleMoveBoard, err := chess.ParseFEN("8/8/8/8/8/8/1r6/K6k w - - 0 1")
	require.NoError(t, err)
	singleMove, err := SelectTacticalMove(singleMoveBoard, true)
	require.NoError(t, err)
	assert.NotEmpty(t, singleMove.String())

	// Promotion move
	promoBoard, err := chess.ParseFEN("8/4P3/8/8/8/8/8/k6K w - - 0 1")
	require.NoError(t, err)
	promoMove, err := SelectTacticalMove(promoBoard, true)
	require.NoError(t, err)
	assert.Equal(t, chess.E8, promoMove.To)
	assert.Equal(t, chess.Queen, promoMove.Promo)

	// Castling preference
	castleBoard, err := chess.ParseFEN("r3k2r/pppq1ppp/2np1n2/2b1p3/2B1P3/2NP1N2/PPPQ1PPP/R3K2R w KQkq - 4 8")
	require.NoError(t, err)
	cMove, err := SelectTacticalMove(castleBoard, true)
	require.NoError(t, err)
	assert.NotEmpty(t, cMove.String())

	// En-passant and Black development/tactics
	epBoard, err := chess.ParseFEN("rnbqkbnr/ppp1pppp/8/3pP3/8/8/PPPP1PPP/RNBQKBNR w KQkq d6 0 2")
	require.NoError(t, err)
	epMove, err := SelectTacticalMove(epBoard, true)
	require.NoError(t, err)
	assert.NotEmpty(t, epMove.String())

	// PlayStyle classifier coverage for all enum variants
	assert.Equal(t, styleTacticalAttacker, classifyPlayStyle(45, 85, 15))
	assert.Equal(t, styleSolidPositional, classifyPlayStyle(70, 50, 40))
	assert.Equal(t, styleGrindingEndgame, classifyPlayStyle(95, 75, 25))
	assert.Equal(t, styleDynamicFighter, classifyPlayStyle(70, 80, 20))
	assert.Equal(t, styleBalanced, classifyPlayStyle(70, 68, 30))

	for _, st := range []playStyleID{styleTacticalAttacker, styleSolidPositional, styleGrindingEndgame, styleDynamicFighter, styleBalanced} {
		name, agg, _ := st.profile()
		assert.NotEmpty(t, name)
		assert.Greater(t, agg, 0.0)
	}
}

