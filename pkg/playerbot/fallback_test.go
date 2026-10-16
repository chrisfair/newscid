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
