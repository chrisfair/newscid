package training

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"newscid/pkg/chess"
	"newscid/pkg/database/scid"
)

func TestFindAttackersAndDefenders(t *testing.T) {
	// Starting position
	b := chess.NewBoard()

	// e4 is attacked by White pawns on d2/f2? No, starting pawns are on rank 2, e4 is rank 4.
	// But d2 is defended by c1 (bishop), e1 (king), d1 (queen), b1 (knight).
	defendersD2 := FindDefenders(b, chess.D2, chess.White)
	assert.True(t, len(defendersD2) >= 2, "d2 should have multiple friendly defenders in initial setup")

	// Italian Game setup: 1. e4 e5 2. Nf3 Nc6 3. Bc4
	b2, err := chess.ParseFEN("r1bqkbnr/pppp1ppp/2n5/4p3/2B1P3/5N2/PPPP1PPP/RNBQK2R b KQkq - 3 3")
	require.NoError(t, err)

	// In this position, e5 is attacked by f3 (Knight) and defended by c6 (Knight)
	attackersE5 := FindAttackers(b2, chess.E5, chess.White)
	require.Len(t, attackersE5, 1)
	assert.Equal(t, chess.F3, attackersE5[0])

	defendersE5 := FindDefenders(b2, chess.E5, chess.Black)
	require.Len(t, defendersE5, 1)
	assert.Equal(t, chess.C6, defendersE5[0])

	// Bishop on c4 is undefended!
	defendersC4 := FindDefenders(b2, chess.C4, chess.White)
	assert.Empty(t, defendersC4, "Bc4 is loose with 0 defenders")
}

func TestCalculateSEE(t *testing.T) {
	// White Knight on d5 captured by Black Pawn on e6, recaptured by White Bishop
	// FEN where Black has undefended knight on e4 attacked by pawn on d3
	fen := "r1bqk2r/pppp1ppp/2n5/2b1p3/2B1n3/3P1N2/PPP2PPP/RNBQ1RK1 b kq - 0 6"
	b, err := chess.ParseFEN(fen)
	require.NoError(t, err)

	// dxe4: White pawn (100) takes Black knight (320)
	see := CalculateSEE(b, chess.D3, chess.E4)
	assert.Greater(t, see, 0, "Capturing undefended knight with pawn should have positive SEE")
}

func TestAnalyzeBoardSafety(t *testing.T) {
	// Initial board should have 0 hanging pieces
	initialReport, err := AnalyzeBoardSafety("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")
	require.NoError(t, err)
	assert.Equal(t, 0, initialReport.TotalHangingWhite)
	assert.Equal(t, 0, initialReport.TotalHangingBlack)

	// Hanging Queen position (Scandinavian blunder: 1. e4 d5 2. exd5 Qxd5 3. Nc3 Qa5 4. d4 - Queen on a5 is loose)
	scandiFEN := "rnb1kbnr/ppp1pppp/8/q7/3P4/2N5/PPP2PPP/R1BQKBNR b KQkq - 1 4"
	report, err := AnalyzeBoardSafety(scandiFEN)
	require.NoError(t, err)

	// Verify loose pieces detected
	var foundLooseQueen bool
	for _, lp := range report.LoosePieces {
		if lp.Square == "a5" && lp.Piece == "Q" {
			foundLooseQueen = true
			break
		}
	}
	assert.True(t, foundLooseQueen, "Queen on a5 should be flagged as loose")

	// Blundered Knight on e4
	blunderFEN := "r1bqk2r/pppp1ppp/2n5/2b1p3/2B1n3/3P1N2/PPP2PPP/RNBQ1RK1 b kq - 0 6"
	blunderReport, err := AnalyzeBoardSafety(blunderFEN)
	require.NoError(t, err)

	assert.Contains(t, blunderReport.HangingSquares, "e4")
	require.NotNil(t, blunderReport.CriticalThreat)
	assert.Equal(t, "e4", blunderReport.CriticalThreat.Square)
	assert.Equal(t, "N", blunderReport.CriticalThreat.Piece)
}

func TestEvaluateSafetySubmission(t *testing.T) {
	drill := &SafetyDrill{
		ID:             "test_drill",
		CorrectSquares: []string{"e4", "c5"},
		WinningMoveUCI: "d3e4",
		Explanation:    "Free piece on e4",
	}

	// Perfect match
	feedback := EvaluateSafetySubmission(drill, []string{"c5", "e4"}, "")
	assert.True(t, feedback.IsCorrect)
	assert.Equal(t, 100, feedback.Score)
	assert.Empty(t, feedback.MissedSquares)
	assert.Empty(t, feedback.WrongSquares)

	// Partial match (missed c5, added wrong a1)
	feedbackPartial := EvaluateSafetySubmission(drill, []string{"e4", "a1"}, "")
	assert.False(t, feedbackPartial.IsCorrect)
	assert.Contains(t, feedbackPartial.MissedSquares, "c5")
	assert.Contains(t, feedbackPartial.WrongSquares, "a1")

	// Move submission mode
	moveFeedback := EvaluateSafetySubmission(drill, nil, "d3e4")
	assert.True(t, moveFeedback.IsCorrect)
	assert.Equal(t, 100, moveFeedback.Score)
}

func TestGetSafetyDrillBatch(t *testing.T) {
	drills := GetSafetyDrillBatch(5, "beginner")
	assert.NotEmpty(t, drills)
	assert.LessOrEqual(t, len(drills), 5)
	for _, d := range drills {
		assert.Equal(t, "beginner", d.Difficulty)
		assert.NotEmpty(t, d.FEN)
		assert.NotEmpty(t, d.CorrectSquares)
	}

	all := GetBuiltinSafetyChallenges()
	assert.NotEmpty(t, all)

	batchAll := GetSafetyDrillBatch(0, "all")
	assert.NotEmpty(t, batchAll)

	batchUnknown := GetSafetyDrillBatch(2, "nonexistent_diff")
	assert.NotEmpty(t, batchUnknown)
}

func TestExtractSafetyDrillsFromDatabase(t *testing.T) {
	// Nil or empty DB returns fallback batch
	fb1 := ExtractSafetyDrillsFromDatabase(nil, 3)
	assert.Len(t, fb1, 3)

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "safety_test_db")
	db := scid.NewDatabase()
	err := db.Create(dbPath, "Safety Test DB")
	require.NoError(t, err)

	fb2 := ExtractSafetyDrillsFromDatabase(db, 2)
	assert.Len(t, fb2, 2)

	// Add game with hanging blunder
	g := chess.NewGame()
	g.SetTag("White", "Player A")
	g.SetTag("Black", "Player B")
	b := chess.NewBoard()
	moves := []string{"e4", "e5", "Nf3", "Nc6", "Bc4", "Nf6", "d3", "d5", "exd5", "Nxd5", "O-O", "Be7"}
	var prev *chess.MoveNode
	for _, san := range moves {
		m, err := b.ParseSAN(san)
		require.NoError(t, err)
		node := &chess.MoveNode{Move: m, SAN: san, UCI: m.String()}
		b.MakeMove(m)
		if g.RootMove == nil {
			g.RootMove = node
		} else {
			prev.Next = node
		}
		prev = node
	}
	_, err = db.SaveGame(g)
	require.NoError(t, err)

	drills := ExtractSafetyDrillsFromDatabase(db, 5)
	assert.NotEmpty(t, drills)
}
