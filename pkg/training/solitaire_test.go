package training

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"newscid/pkg/chess"
)

func TestSolitaireSessionWhite(t *testing.T) {
	pgn := `[Event "World Championship"]
[Site "Reykjavik"]
[Date "1972.07.23"]
[White "Fischer, Robert J."]
[Black "Spassky, Boris V."]
[Result "1-0"]

1. c4 e6 2. Nf3 d5 3. d4 Nf6 4. Nc3 Be7 1-0`

	game, err := chess.ParsePGN(pgn)
	require.NoError(t, err)

	sess, err := NewSolitaireSession(game, "white")
	require.NoError(t, err)
	assert.Equal(t, 4, sess.TotalTurns)
	assert.Equal(t, "Fischer, Robert J.", sess.White)
	assert.False(t, sess.IsFinished)

	// Turn 1: User needs to guess 1. c4
	rec1 := sess.CurrentTurnRecord()
	require.NotNil(t, rec1)
	assert.Equal(t, "c4", rec1.SAN)

	// Wrong guess: e4
	fbWrong := sess.SubmitGuess("e4")
	assert.False(t, fbWrong.IsMatch)
	assert.Equal(t, 0, fbWrong.PointsEarned)
	assert.Equal(t, 0, sess.Score)

	// Request hint
	hint1, lvl1 := sess.GetHint()
	assert.Equal(t, 1, lvl1)
	assert.Contains(t, hint1, "Pawn")

	// Correct guess with 1 hint -> +7 points
	fbCorrect := sess.SubmitGuess("c4")
	assert.True(t, fbCorrect.IsMatch)
	assert.Equal(t, 7, fbCorrect.PointsEarned)
	assert.Equal(t, 7, sess.Score)
	assert.Equal(t, "e6", fbCorrect.OpponentReplySAN) // Spassky's 1... e6 auto-played!

	// Turn 2: White to guess 2. Nf3. Test progressive hints
	hintA, lvlA := sess.GetHint()
	assert.Equal(t, 1, lvlA)
	assert.Contains(t, hintA, "Knight")

	hintB, lvlB := sess.GetHint()
	assert.Equal(t, 2, lvlB)
	assert.Contains(t, hintB, "f-file")

	hintC, lvlC := sess.GetHint()
	assert.Equal(t, 3, lvlC)
	assert.Contains(t, hintC, "f3")

	// Guess Nf3 with 3 hints -> +2 points
	fbNf3 := sess.SubmitGuess("Nf3")
	assert.True(t, fbNf3.IsMatch)
	assert.Equal(t, 2, fbNf3.PointsEarned)
	assert.Equal(t, "d5", fbNf3.OpponentReplySAN) // 2... d5 auto-played!

	// Turn 3: 3. d4. Test Reveal and Skip
	fbSkip := sess.RevealAndSkip()
	assert.False(t, fbSkip.IsMatch)
	assert.Equal(t, "d4", fbSkip.ActualMoveSAN)
	assert.Equal(t, "Nf6", fbSkip.OpponentReplySAN)

	// Turn 4: 4. Nc3. Correct guess without hints -> +10 points
	fbNc3 := sess.SubmitGuess("Nc3")
	assert.True(t, fbNc3.IsMatch)
	assert.Equal(t, 10, fbNc3.PointsEarned)
	assert.Equal(t, "Be7", fbNc3.OpponentReplySAN)
	assert.True(t, fbNc3.IsFinished)
	require.NotNil(t, fbNc3.Summary)

	assert.Equal(t, 19, fbNc3.Summary.Score)
	assert.Equal(t, 40, fbNc3.Summary.MaxScore)
	assert.Equal(t, 3, fbNc3.Summary.CorrectGuesses)
	assert.NotEmpty(t, fbNc3.Summary.RatingTitle)
}

func TestSolitaireSessionBlack(t *testing.T) {
	pgn := `[Event "Candidates"]
[White "Player1"]
[Black "Player2"]
[Result "0-1"]

1. e4 c5 2. Nf3 d6 0-1`

	game, err := chess.ParsePGN(pgn)
	require.NoError(t, err)

	sess, err := NewSolitaireSession(game, "black")
	require.NoError(t, err)
	assert.Equal(t, 2, sess.TotalTurns)

	// Since user is Black, move 1 (1. e4) was auto-played! Current board FEN has e4 already made!
	rec := sess.CurrentTurnRecord()
	require.NotNil(t, rec)
	assert.Equal(t, "c5", rec.SAN)

	fb := sess.SubmitGuess("c5")
	assert.True(t, fb.IsMatch)
	assert.Equal(t, 10, fb.PointsEarned)
	assert.Equal(t, "Nf3", fb.OpponentReplySAN) // 2. Nf3 auto-played!

	// Next turn: 2... d6
	fb2 := sess.SubmitGuess("d6")
	assert.True(t, fb2.IsMatch)
	assert.True(t, fb2.IsFinished)
	require.NotNil(t, fb2.Summary)
	assert.Equal(t, 20, fb2.Summary.Score)
	assert.Equal(t, 20, fb2.Summary.MaxScore)
	assert.Equal(t, 100.0, fb2.Summary.Percentage)
	assert.Contains(t, fb2.Summary.RatingTitle, "Grandmaster")
}

func TestSolitaireEdgeCases(t *testing.T) {
	// Nil game
	_, err := NewSolitaireSession(nil, "white")
	assert.Error(t, err)

	// Empty game
	gEmpty := chess.NewGame()
	_, err = NewSolitaireSession(gEmpty, "white")
	assert.Error(t, err)

	// Custom start FEN
	pgnFEN := `[Event "FEN Game"]
[FEN "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1"]

1... e5 2. Nf3 1-0`
	gFEN, err := chess.ParsePGN(pgnFEN)
	require.NoError(t, err)
	sFEN, err := NewSolitaireSession(gFEN, "black")
	require.NoError(t, err)
	assert.NotEmpty(t, sFEN.CurrentFEN())

	// Test nil board FEN fallback
	sNil := &SolitaireSession{}
	assert.Equal(t, chess.StartingFEN, sNil.CurrentFEN())

	// Test post-completion calls on sFEN
	fb1 := sFEN.SubmitGuess("e5")
	assert.True(t, fb1.IsMatch)
	assert.True(t, sFEN.IsFinished)

	// Calling GetHint on finished session
	hFin, lFin := sFEN.GetHint()
	assert.Contains(t, hFin, "completed")
	assert.Equal(t, 0, lFin)

	// Calling SubmitGuess on finished session
	fbFin := sFEN.SubmitGuess("a6")
	assert.True(t, fbFin.IsFinished)

	// Calling RevealAndSkip on finished session
	fbRevFin := sFEN.RevealAndSkip()
	assert.True(t, fbRevFin.IsFinished)

	// Test all PerformanceSummary rating brackets
	sTiers := &SolitaireSession{TotalTurns: 10}

	sTiers.Score = 85
	assert.Contains(t, sTiers.GetSummary().RatingTitle, "International Master")

	sTiers.Score = 75
	assert.Contains(t, sTiers.GetSummary().RatingTitle, "FIDE Master")

	sTiers.Score = 60
	assert.Contains(t, sTiers.GetSummary().RatingTitle, "Candidate Master")

	sTiers.Score = 45
	assert.Contains(t, sTiers.GetSummary().RatingTitle, "Club Player")

	sTiers.Score = 20
	assert.Contains(t, sTiers.GetSummary().RatingTitle, "Developing Player")

	// Test 2 hints -> 4 points, and RevealAndSkip on final move
	gShort, err := chess.ParsePGN("1. e4 e5 2. Nf3 1-0")
	require.NoError(t, err)
	sShort, err := NewSolitaireSession(gShort, "white")
	require.NoError(t, err)
	_, _ = sShort.GetHint()
	_, _ = sShort.GetHint()
	fb2Hints := sShort.SubmitGuess("e4")
	assert.Equal(t, 4, fb2Hints.PointsEarned)
	// Final move (2. Nf3) skipped via RevealAndSkip -> finishes session
	fbSkipFinal := sShort.RevealAndSkip()
	assert.True(t, fbSkipFinal.IsFinished)
	assert.NotNil(t, fbSkipFinal.Summary)
}
