package repertoire_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"newscid/pkg/chess"
	"newscid/pkg/repertoire"
)

func TestRepertoireManager(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "repertoire_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	mgr := repertoire.NewManager(tmpDir)
	list := mgr.ListRepertoires()
	require.NotEmpty(t, list)
	assert.GreaterOrEqual(t, len(list), 2) // White & Black starter repertoires

	// Find the white repertoire
	var whiteRep *repertoire.RepertoireSummary
	for i := range list {
		if list[i].Color == "w" {
			whiteRep = &list[i]
			break
		}
	}
	require.NotNil(t, whiteRep)

	rep, err := mgr.GetRepertoire(whiteRep.ID)
	require.NoError(t, err)
	assert.Equal(t, "w", rep.Color)
	assert.NotEmpty(t, rep.Positions)

	// In initial position, White's move e4 (e2e4) should be valid
	isMatch, move, _, err := mgr.VerifyMove(whiteRep.ID, chess.StartingFEN, "e2e4")
	require.NoError(t, err)
	assert.True(t, isMatch)
	require.NotNil(t, move)
	assert.Equal(t, "e2e4", move.UCI)

	// An off-repertoire move like a2a3 should return false
	isMatchOff, _, options, err := mgr.VerifyMove(whiteRep.ID, chess.StartingFEN, "a2a3")
	require.NoError(t, err)
	assert.False(t, isMatchOff)
	assert.NotEmpty(t, options)

	// Record practice
	mgr.RecordPractice(whiteRep.ID, chess.StartingFEN, "e2e4", true)
	repAfter, _ := mgr.GetRepertoire(whiteRep.ID)
	core := repertoire.CoreFEN(chess.StartingFEN)
	assert.Equal(t, 1, repAfter.Positions[core][0].Practice)
	assert.Equal(t, 1, repAfter.Positions[core][0].Correct)

	// Test saving a custom repertoire
	custom := &repertoire.Repertoire{
		ID:          "custom-caro-kann",
		Name:        "Custom Caro-Kann",
		Color:       "b",
		Description: "Solid defense with c6",
		Positions:   make(map[string][]repertoire.RepertoireMove),
	}
	err = mgr.SaveRepertoire(custom)
	require.NoError(t, err)

	// Verify persistence from disk
	mgr2 := repertoire.NewManager(tmpDir)
	loaded, err := mgr2.GetRepertoire("custom-caro-kann")
	require.NoError(t, err)
	assert.Equal(t, "Custom Caro-Kann", loaded.Name)

	// Test deletion
	err = mgr2.DeleteRepertoire("custom-caro-kann")
	require.NoError(t, err)
	_, err = mgr2.GetRepertoire("custom-caro-kann")
	assert.Error(t, err)

	// Test auto-slug ID and nil Positions on SaveRepertoire
	autoRep := &repertoire.Repertoire{
		Name:  "My Sicilian Lines",
		Color: "b",
	}
	require.NoError(t, mgr2.SaveRepertoire(autoRep))
	assert.Equal(t, "b-my-sicilian-lines", autoRep.ID)
	assert.NotNil(t, autoRep.Positions)

	// VerifyMove and RecordPractice on missing ID
	_, _, _, err = mgr2.VerifyMove("missing-id", chess.StartingFEN, "e2e4")
	assert.Error(t, err)
	mgr2.RecordPractice("missing-id", chess.StartingFEN, "e2e4", false)

	// CoreFEN with short string and leading/trailing spaces
	assert.Equal(t, "short fen", repertoire.CoreFEN("  short fen  "))

	// Test NewManager with empty storageDir (using temporary HOME)
	t.Setenv("HOME", t.TempDir())
	mgrDefault := repertoire.NewManager("")
	assert.NotEmpty(t, mgrDefault.ListRepertoires())
}

func TestSystematicMasteryAndEditing(t *testing.T) {
	mgr := repertoire.NewManager(t.TempDir())

	rep := &repertoire.Repertoire{
		ID:          "white-test-book",
		Name:        "White Test Book",
		Color:       "w",
		Description: "Test systematic mastery",
		Positions:   make(map[string][]repertoire.RepertoireMove),
	}
	require.NoError(t, mgr.SaveRepertoire(rep))

	// Add two lines via AddLineSAN
	updated, err := mgr.AddLineSAN("white-test-book", "1. e4 e5 2. Nf3 Nc6 3. Bc4 Bc5", "Giuoco Piano")
	require.NoError(t, err)
	require.NotNil(t, updated)

	updated, err = mgr.AddLineSAN("white-test-book", "1. e4 c5 2. Nf3 d6", "Sicilian Defense")
	require.NoError(t, err)

	// Check initial mastery report (0% mastery, 2 lines)
	report, err := mgr.GetMasteryReport("white-test-book")
	require.NoError(t, err)
	assert.Equal(t, 2, report.TotalLines)
	assert.Equal(t, 0, report.MasteredLines)
	assert.Equal(t, 0.0, report.MasteryPct)
	assert.Greater(t, report.TotalUserNodes, 0)

	// Practice every user step in Line 0 and Line 1
	for _, line := range report.Lines {
		for _, step := range line.Steps {
			if step.IsUserMove {
				mgr.RecordPractice("white-test-book", step.FEN, step.UCI, true)
			}
		}
	}

	// Mastery should now be 100%!
	report100, err := mgr.GetMasteryReport("white-test-book")
	require.NoError(t, err)
	assert.Equal(t, 100.0, report100.MasteryPct)
	assert.Equal(t, report100.TotalLines, report100.MasteredLines)

	// Make a mistake on 1.e4 (playing a2a3 instead) -> should drop mastery below 100% and require 2 streak to re-master
	mgr.RecordPractice("white-test-book", chess.StartingFEN, "a2a3", false)
	reportAfterMiss, err := mgr.GetMasteryReport("white-test-book")
	require.NoError(t, err)
	assert.Less(t, reportAfterMiss.MasteryPct, 100.0)

	// One correct answer sets Streak=1 (not yet 2 after a miss)
	mgr.RecordPractice("white-test-book", chess.StartingFEN, "e2e4", true)
	reportStreak1, err := mgr.GetMasteryReport("white-test-book")
	require.NoError(t, err)
	assert.Less(t, reportStreak1.MasteryPct, 100.0)

	// Second consecutive correct answer restores Streak=2 -> back to 100%!
	mgr.RecordPractice("white-test-book", chess.StartingFEN, "e2e4", true)
	reportRestored, err := mgr.GetMasteryReport("white-test-book")
	require.NoError(t, err)
	assert.Equal(t, 100.0, reportRestored.MasteryPct)

	// Update comment on 1.e4
	_, err = mgr.UpdateMoveComment("white-test-book", chess.StartingFEN, "e2e4", "Control d5 and f5")
	require.NoError(t, err)
	repCheck, _ := mgr.GetRepertoire("white-test-book")
	assert.Equal(t, "Control d5 and f5", repCheck.Positions[repertoire.CoreFEN(chess.StartingFEN)][0].Comment)

	// Reset mastery -> back to 0%
	_, err = mgr.ResetMastery("white-test-book")
	require.NoError(t, err)
	reportReset, err := mgr.GetMasteryReport("white-test-book")
	require.NoError(t, err)
	assert.Equal(t, 0.0, reportReset.MasteryPct)

	// Compile to Polyglot book
	binPath := t.TempDir() + "/test_book.bin"
	entriesCount, err := mgr.CompileToPolyglotBook("white-test-book", binPath)
	require.NoError(t, err)
	assert.Greater(t, entriesCount, 0)

	// Delete a move
	_, err = mgr.DeleteMove("white-test-book", chess.StartingFEN, "e2e4")
	require.NoError(t, err)
	repAfterDel, _ := mgr.GetRepertoire("white-test-book")
	assert.Empty(t, repAfterDel.Positions[repertoire.CoreFEN(chess.StartingFEN)])
}

func TestScanGamesForBookDeviations(t *testing.T) {
	mgr := repertoire.NewManager(t.TempDir())

	// Use starter white Italian repertoire: 1.e4 e5 2.Nf3 Nc6 3.Bc4 Bc5 4.c3 Nf6 5.d4
	// Game 1: User follows book
	g1, err := chess.ParsePGN(`[White "HeroUser"]
[Black "Opponent1"]
[Result "1-0"]

1. e4 e5 2. Nf3 Nc6 3. Bc4 Bc5 4. c3 Nf6 5. d4 exd4 1-0`)
	require.NoError(t, err)

	// Game 2: User breaks book on move 3 with 3.Bb5 instead of 3.Bc4 and loses
	g2, err := chess.ParsePGN(`[White "HeroUser"]
[Black "Opponent2"]
[Result "0-1"]

1. e4 e5 2. Nf3 Nc6 3. Bb5 a6 4. Ba4 Nf6 0-1`)
	require.NoError(t, err)

	// Game 3: User breaks book on move 3 with 3.Nxe5?? (blundering the knight) and loses
	g3, err := chess.ParsePGN(`[White "HeroUser"]
[Black "Opponent3"]
[Result "0-1"]

1. e4 e5 2. Nf3 Nc6 3. Nxe5 Nxe5 0-1`)
	require.NoError(t, err)

	// Game 4: Opponent plays uncovered sideline 1.e4 d5 (Scandinavian)
	g4, err := chess.ParsePGN(`[White "HeroUser"]
[Black "Opponent4"]
[Result "1/2-1/2"]

1. e4 d5 2. exd5 Qxd5 1/2-1/2`)
	require.NoError(t, err)

	report, err := mgr.ScanGamesForBookDeviations(
		"white-openings-italian",
		[]*chess.Game{g1, g2, g3, g4},
		[]string{"HeroUser"},
	)
	require.NoError(t, err)
	require.NotNil(t, report)

	assert.Equal(t, 4, report.TotalGamesScanned)
	assert.Equal(t, 4, report.MatchingColorGames)
	assert.Equal(t, 4, report.GamesEnteredBook)
	assert.Equal(t, 2, report.CompliantGames) // g1 (compliant) + g4 (opponent deviated, user didn't break book)
	assert.Equal(t, 2, report.UserDeviatedGames)
	assert.Equal(t, 1, report.OpponentSidelineGames)
	assert.Equal(t, 50.0, report.CompliancePct)

	require.Len(t, report.UserDeviations, 2)
	// Verify avoidance advice is populated and references book move Bc4
	for _, dev := range report.UserDeviations {
		assert.Equal(t, "Bc4", dev.PrimaryBookSAN)
		assert.Contains(t, dev.AvoidanceAdvice, "Bc4")
		if dev.PlayedMoveSAN == "Nxe5" {
			assert.Greater(t, dev.EvalLossCp, 200)
			assert.Equal(t, "Nxe5", dev.PunishingReplySAN)
		}
	}

	require.Len(t, report.OpponentSidelines, 1)
	assert.Equal(t, "d5", report.OpponentSidelines[0].OpponentMoveSAN)
	assert.NotEmpty(t, report.OpponentSidelines[0].RecommendedReplySAN)
	assert.NotEmpty(t, report.OpponentSidelines[0].FullLineWithReply)
}
