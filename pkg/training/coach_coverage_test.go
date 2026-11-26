package training

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"newscid/pkg/chess"
)

func TestCoachHelpersCoverage(t *testing.T) {
	// pieceTypeLabel all cases
	for _, pt := range []chess.PieceType{chess.Pawn, chess.Knight, chess.Bishop, chess.Rook, chess.Queen, chess.King, chess.None} {
		assert.NotEmpty(t, pieceTypeLabel(pt))
	}

	// coachingHabitForTheme all themes
	themes := []string{
		"hangingPiece", "fork", "pin", "skewer",
		"discoveredAttack", "doubleCheck", "backRankMate",
		"mateIn1", "deflection", "attraction", "unknownTheme",
	}
	for _, th := range themes {
		assert.NotEmpty(t, coachingHabitForTheme(th))
		assert.NotEmpty(t, buildStepByStepCoachText(th, "Theme", "Nf3", "a3"))
		assert.NotEmpty(t, buildStepByStepCoachText(th, "Theme", "Nf3", ""))
	}

	// buildPostSolveTakeaway fallback when FEN/move invalid
	assert.Contains(t, buildPostSolveTakeaway("invalid-fen", "z9z9", 200, "fork", "Fork"), "Fork")

	// buildWhatWentWrongText branches
	b, err := chess.ParseFEN("4k3/8/8/8/4r3/8/4R3/4K3 w - - 0 1")
	require.NoError(t, err)
	bestMove, err := b.ParseUCI("e2e4")
	require.NoError(t, err)
	playedMove, err := b.ParseUCI("e1f1")
	require.NoError(t, err)
	bAfter := b.Clone()
	bAfter.MakeMove(playedMove)
	oppCapture, err := bAfter.ParseUCI("e4e2")
	require.NoError(t, err)

	// Undefended capture missed
	txt1 := buildWhatWentWrongText(b, playedMove, "Kf1", bestMove, "Rxe4", bAfter, oppCapture, 500, "Hanging Piece", 12)
	assert.Contains(t, txt1, "undefended")

	// Opponent capture refutation when best move is quiet
	b2, err := chess.ParseFEN("4k3/8/8/8/4r3/8/4R3/4K3 w - - 0 1")
	require.NoError(t, err)
	quietBest, err := b2.ParseUCI("e2e3")
	require.NoError(t, err)
	txt2 := buildWhatWentWrongText(b2, playedMove, "Kf1", quietBest, "Re3", bAfter, oppCapture, 500, "Hanging Piece", 12)
	assert.Contains(t, txt2, "capturing")

	// Opponent quiet tactical refutation (non-capture)
	oppQuiet, err := bAfter.ParseUCI("e4e5")
	require.NoError(t, err)
	txt3 := buildWhatWentWrongText(b2, playedMove, "Kf1", quietBest, "Re3", bAfter, oppQuiet, 300, "Initiative", 12)
	assert.Contains(t, txt3, "refutation")

	// Generic fallback (oppPunishScore < 240)
	txt4 := buildWhatWentWrongText(b2, playedMove, "Kf1", quietBest, "Re3", bAfter, oppQuiet, 50, "Strategy", 12)
	assert.Contains(t, txt4, "Strategy")
}
