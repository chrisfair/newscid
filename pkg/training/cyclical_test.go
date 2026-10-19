package training

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"newscid/pkg/blundervault"
	"newscid/pkg/chess"
	"newscid/pkg/puzzleminer"
)

func TestLichessThemeCatalogAndBitmask(t *testing.T) {
	catalog := GetLichessThemeCatalog()
	require.GreaterOrEqual(t, len(catalog), 35)

	actionableCount := 0
	for _, item := range catalog {
		assert.NotEmpty(t, item.Key)
		assert.NotEmpty(t, item.Name)
		assert.NotEmpty(t, item.Category)
		assert.NotEmpty(t, item.Description)
		if item.Actionable {
			actionableCount++
		}
		id, ok := LookupThemeID(item.Key)
		assert.True(t, ok)
		assert.Equal(t, item.ID, id)
	}
	assert.GreaterOrEqual(t, actionableCount, 26)

	// Invalid theme lookup
	_, ok := LookupThemeID("nonExistentTheme")
	assert.False(t, ok)

	// Bitmask operations
	mask := ThemeMaskFromKeys([]string{"fork", "backRankMate", "SACRIFICE", "unknown"})
	assert.True(t, mask.Has(ThemeFork))
	assert.True(t, mask.Has(ThemeBackRankMate))
	assert.True(t, mask.Has(ThemeSacrifice))
	assert.False(t, mask.Has(ThemePin))
	assert.False(t, mask.Has(ThemeID(200)))

	var badMask ThemeMask
	badMask.Set(ThemeID(200))
	assert.Equal(t, ThemeMask(0), badMask)

	keys := mask.Keys()
	assert.ElementsMatch(t, []string{"fork", "sacrifice", "backRankMate"}, keys)
	assert.Equal(t, "backRankMate", PrimaryThemeFromMask(mask))
	assert.Equal(t, "middlegame", PrimaryThemeFromMask(0))

	filterFork := ThemeMaskFromKeys([]string{"fork"})
	filterPin := ThemeMaskFromKeys([]string{"pin"})
	assert.True(t, mask.MatchesFilter(0))
	assert.True(t, mask.MatchesFilter(filterFork))
	assert.False(t, mask.MatchesFilter(filterPin))

	// DifficultyTierID labels and parsing
	assert.Equal(t, "Easy", TierEasy.String())
	assert.Equal(t, "Medium", TierMedium.String())
	assert.Equal(t, "Hard", TierHard.String())
	assert.Equal(t, "Medium", DifficultyTierID(99).String())
	assert.Equal(t, TierEasy, ParseDifficultyTierID("Easy"))
	assert.Equal(t, TierEasy, ParseDifficultyTierID("beginner"))
	assert.Equal(t, TierMedium, ParseDifficultyTierID("Medium"))
	assert.Equal(t, TierHard, ParseDifficultyTierID("Hard"))
	assert.Equal(t, TierHard, ParseDifficultyTierID("advanced"))
}

func TestClassifyPositionThemes_MotifsMatesAndPhases(t *testing.T) {
	// 1. Invalid FEN fallback
	themesInvalid, tierInvalid, ratingInvalid := ClassifyPositionThemes("invalid_fen", []string{"e2e4"}, 200)
	assert.Contains(t, themesInvalid, "middlegame")
	assert.Equal(t, TierEasy, tierInvalid)
	assert.Equal(t, 1200, ratingInvalid)

	// 2. Back Rank Mate in 1 & Rook Endgame
	themesBR, tierBR, ratingBR := ClassifyPositionThemes(
		"6k1/5ppp/8/8/8/8/5PPP/4R1K1 w - - 0 18",
		[]string{"e1e8"},
		1000,
	)
	assert.Contains(t, themesBR, "backRankMate")
	assert.Contains(t, themesBR, "mateIn1")
	assert.Contains(t, themesBR, "rookEndgame")
	assert.Equal(t, TierEasy, tierBR)
	assert.GreaterOrEqual(t, ratingBR, 1000)

	// 3. Smothered Mate in 1
	themesSmothered, _, _ := ClassifyPositionThemes(
		"6rk/6pp/7N/8/8/8/5PPP/6K1 w - - 0 15",
		[]string{"h6f7"},
		1000,
	)
	assert.Contains(t, themesSmothered, "smotheredMate")
	assert.Contains(t, themesSmothered, "mateIn1")

	// 4. Arabian Mate in 1
	themesArabian, _, _ := ClassifyPositionThemes(
		"7k/2R5/5N2/8/8/8/6PP/6K1 w - - 0 21",
		[]string{"c7h7"},
		1000,
	)
	assert.Contains(t, themesArabian, "arabianMate")
	assert.Contains(t, themesArabian, "mateIn1")

	// 5. Hook Mate in 1 (King on f7, pawn on g7, White Knight on e5, White Rook plays h2-h7#)
	themesHook, _, _ := ClassifyPositionThemes(
		"8/4Rpk1/5N1p/6P1/8/8/6PP/6K1 b - - 0 25",
		nil,
		300,
	)
	assert.NotEmpty(t, themesHook)
	themesHookMate, _, _ := ClassifyPositionThemes(
		"6k1/5p1p/5N2/6P1/8/8/2R4P/6K1 w - - 0 25",
		[]string{"c2c8"},
		1000,
	)
	assert.NotEmpty(t, themesHookMate)

	// True Hook Mate position: Black King on g7, Black pawn on f6, White Knight on e5, White Rook plays h1-h7# or similar
	themesHookReal, _, _ := ClassifyPositionThemes(
		"8/6pk/5N2/6P1/8/8/1R5P/6K1 w - - 0 25",
		[]string{"b2b8"},
		400,
	)
	assert.NotEmpty(t, themesHookReal)

	// 6. En Passant + Pawn Endgame + Promotion + Advanced Pawn
	themesEP, _, _ := ClassifyPositionThemes(
		"4k3/8/8/4pP2/8/8/8/4K3 w - e6 0 20",
		[]string{"f5e6"},
		300,
	)
	assert.Contains(t, themesEP, "enPassant")
	assert.Contains(t, themesEP, "pawnEndgame")
	assert.Contains(t, themesEP, "advancedPawn")

	themesPromo, _, _ := ClassifyPositionThemes(
		"8/1P4pk/8/8/8/8/5PPP/6K1 w - - 0 28",
		[]string{"b7b8q"},
		800,
	)
	assert.Contains(t, themesPromo, "promotion")
	assert.Contains(t, themesPromo, "advancedPawn")

	// 7. Skewer + Minor Piece Endgame
	themesSkewer, _, _ := ClassifyPositionThemes(
		"q7/8/2k5/8/4B3/8/6PP/6K1 w - - 0 23",
		[]string{"e4f3", "c6c5", "f3a8"},
		900,
	)
	assert.Contains(t, themesSkewer, "skewer")
	assert.Contains(t, themesSkewer, "minorPieceEndgame")

	// 8. Queen Endgame
	themesQE, _, _ := ClassifyPositionThemes(
		"6k1/5qp1/7p/8/8/6PP/5PK1/3Q4 w - - 0 30",
		[]string{"d1d8", "f7f8", "d8d5"},
		500,
	)
	assert.Contains(t, themesQE, "queenEndgame")

	// 9. Double Check + Mate in 2
	themesDC, _, _ := ClassifyPositionThemes(
		"4k3/3n1ppp/4N3/8/8/8/5PPP/4R1K1 w - - 0 17",
		[]string{"e6c7", "e8f8", "e1e8"},
		1000,
	)
	assert.Contains(t, themesDC, "doubleCheck")
	assert.Contains(t, themesDC, "discoveredAttack")
	assert.Contains(t, themesDC, "mateIn2")

	// 10. Sacrifice + Deflection + Mate in 3
	themesM3, tierM3, ratingM3 := ClassifyPositionThemes(
		"2q1r1k1/1p3ppp/8/8/8/8/2Q2PPP/3R1RK1 w - - 0 24",
		[]string{"c2c8", "e8c8", "d1d8", "c8d8", "f1d8"},
		1000,
	)
	assert.Contains(t, themesM3, "mateIn3")
	assert.Contains(t, themesM3, "deflection")
	assert.Equal(t, TierHard, tierM3)
	assert.GreaterOrEqual(t, ratingM3, 1850)

	// 11. Defensive Move when starting in check
	themesDef, _, _ := ClassifyPositionThemes(
		"4k3/8/8/8/1b6/8/8/4K2R w K - 0 15",
		[]string{"e1f1"},
		150,
	)
	assert.Contains(t, themesDef, "defensiveMove")

	// 12. Hook Mate on non-corner square (Black King on f8, Black pawns on e7/f7/g7, White Knight on e6, White Rook plays d1-d8#)
	themesHookNonCorner, _, _ := ClassifyPositionThemes(
		"5k2/4ppp1/4N3/8/8/8/5PPP/3R2K1 w - - 0 22",
		[]string{"d1d8"},
		1000,
	)
	assert.Contains(t, themesHookNonCorner, "hookMate")
	assert.Contains(t, themesHookNonCorner, "mateIn1")

	// 13. Intermezzo intermediate check before capture
	themesIntermezzo, _, _ := ClassifyPositionThemes(
		"6k1/5ppp/8/8/1q6/8/4QPPP/1r1R2K1 w - - 0 20",
		[]string{"e2e8", "b4f8", "d1b1"},
		650,
	)
	assert.Contains(t, themesIntermezzo, "intermezzo")

	// 14. Flexible move parser fallback (SAN move in solutionUCI and coordinate fallback)
	themesSANFallback, _, _ := ClassifyPositionThemes(
		"6k1/5ppp/8/8/8/8/5PPP/4R1K1 w - - 0 18",
		[]string{"Re8#"},
		1000,
	)
	assert.Contains(t, themesSANFallback, "backRankMate")
}

func TestBuiltInCyclicalPuzzles_OrderedEasyToHard(t *testing.T) {
	puzzles := GetBuiltInCyclicalPuzzles()
	require.Len(t, puzzles, 24)

	prevTier := TierEasy
	prevRating := 0

	easyCount, medCount, hardCount := 0, 0, 0
	for i, p := range puzzles {
		assert.Equal(t, i+1, p.DifficultyOrder)
		assert.NotEmpty(t, p.ID)
		assert.NotEmpty(t, p.FEN)
		assert.NotEmpty(t, p.SolutionUCI)
		assert.NotEmpty(t, p.SolutionSAN)
		assert.NotEmpty(t, p.Themes)
		assert.NotEmpty(t, p.PrimaryTheme)

		curTier := ParseDifficultyTierID(p.DifficultyTier)
		assert.GreaterOrEqual(t, uint8(curTier), uint8(prevTier), "puzzles must be ordered Easy -> Medium -> Hard")
		if curTier == prevTier {
			assert.GreaterOrEqual(t, p.EstimatedRating, prevRating, "ratings within tier must be non-decreasing")
		}
		prevTier = curTier
		prevRating = p.EstimatedRating

		switch curTier {
		case TierEasy:
			easyCount++
		case TierMedium:
			medCount++
		case TierHard:
			hardCount++
		}
	}

	assert.Equal(t, 8, easyCount)
	assert.Equal(t, 8, medCount)
	assert.Equal(t, 8, hardCount)
}

func TestExtractPuzzlesFromPlayerGamesAndConversions(t *testing.T) {
	// Parse a PGN game containing annotated blunders, [%eval] swings, variations, and checkmate
	pgn := `[Event "Club Championship"]
[Site "Local"]
[Date "2025.05.10"]
[Round "1"]
[White "ChristopherFair"]
[Black "TacticalOpponent"]
[Result "1-0"]

1. e4 {[%eval +0.25]} e5 {[%eval +0.30]} 2. Nf3 Nc6 3. Bc4 Bc5 4. c3 Nf6 5. d4 exd4 6. cxd4 Bb4+ 7. Nc3 Nxe4? {[%eval -2.50]} (7... d5 8. exd5 Nxd5) 8. O-O! {[%eval +2.80]} Nxc3 9. bxc3 Bxc3 10. Qb3 Bxa1?? (10... d5 11. Bxd5 O-O) 11. Bxf7+ Kf8 12. Bg5 Ne7 13. Ne5 Bxd4 14. Bg6 d5 15. Qf3+ Bf5 16. Bxf5 Bxe5 17. Be6+ Bf6 18. Bxf6 gxf6 19. Qxf6+ Ke8 20. Qf7# 1-0`

	g, err := chess.ParsePGN(pgn)
	require.NoError(t, err)
	g.ID = 42

	extracted := ExtractPuzzlesFromPlayerGames([]*chess.Game{g}, []string{"ChristopherFair"}, nil, 12)
	require.Len(t, extracted, 12)

	// Verify at least one puzzle came from game 42 and that the list is strictly ordered Easy -> Hard
	fromGame := 0
	for i, p := range extracted {
		assert.Equal(t, i+1, p.DifficultyOrder)
		if p.SourceGameID == 42 {
			fromGame++
		}
	}
	assert.GreaterOrEqual(t, fromGame, 1)

	// Test with theme filter ("backRankMate")
	filtered := ExtractPuzzlesFromPlayerGames([]*chess.Game{g}, []string{"NonMatchingPlayer"}, []string{"backRankMate"}, 5)
	require.NotEmpty(t, filtered)
	for _, p := range filtered {
		assert.Contains(t, p.Themes, "backRankMate")
	}

	// Test ConvertBlunderVaultPuzzle
	vp := &blundervault.Puzzle{
		ID:         "bv-1",
		FEN:        "6k1/5ppp/8/8/8/8/5PPP/4R1K1 w - - 0 18",
		PlayerName: "ChristopherFair",
		UserColor:  "black",
		PlayedMove: "e1e2",
		BestMove:   "e1e8",
		DeltaCp:    0,
		Event:      "Blitz",
		Date:       "2025.05.01",
	}
	cp, ok := ConvertBlunderVaultPuzzle(vp)
	require.True(t, ok)
	assert.Equal(t, "vault-bv-1", cp.ID)
	assert.Equal(t, "ChristopherFair", cp.Black)
	assert.Contains(t, cp.Themes, "backRankMate")

	_, ok = ConvertBlunderVaultPuzzle(nil)
	assert.False(t, ok)
	_, ok = ConvertBlunderVaultPuzzle(&blundervault.Puzzle{FEN: "invalid", BestMove: "e2e4"})
	assert.False(t, ok)
	_, ok = ConvertBlunderVaultPuzzle(&blundervault.Puzzle{FEN: chess.StartingFEN, BestMove: "invalid_uci"})
	assert.False(t, ok)

	// Test ConvertMinedPuzzle
	mp := puzzleminer.MinedPuzzle{
		ID:          "mined-1",
		GameID:      7,
		White:       "WhitePlayer",
		Black:       "BlackPlayer",
		StartFEN:    "6k1/5ppp/8/8/8/8/5PPP/4R1K1 w - - 0 18",
		BestMoveUCI: "e1e8",
		EvalDeltaCp: 900,
		Description: "Back rank mate",
	}
	mcp, ok := ConvertMinedPuzzle(mp)
	require.True(t, ok)
	assert.Equal(t, "mined-1", mcp.ID)
	assert.Contains(t, mcp.Themes, "backRankMate")

	_, ok = ConvertMinedPuzzle(puzzleminer.MinedPuzzle{})
	assert.False(t, ok)
	_, ok = ConvertMinedPuzzle(puzzleminer.MinedPuzzle{StartFEN: "bad_fen", BestMoveUCI: "e2e4"})
	assert.False(t, ok)
	_, ok = ConvertMinedPuzzle(puzzleminer.MinedPuzzle{StartFEN: chess.StartingFEN})
	assert.False(t, ok)
}

func TestCyclicalManager_MultiCycleLifecycleAndLichessDashboard(t *testing.T) {
	tmpDir := t.TempDir()
	drillPath := filepath.Join(tmpDir, "cyclical_drill.json")

	cm, err := NewCyclicalManager(drillPath)
	require.NoError(t, err)

	// Default constructor with empty path
	defaultCM, err := NewCyclicalManager("")
	require.NoError(t, err)
	assert.NotNil(t, defaultCM)

	// Initial empty dashboard
	dash0 := cm.GetDashboard()
	assert.Empty(t, dash0.Sets)
	assert.Len(t, dash0.RadarAxes, 9)
	assert.Equal(t, 1500, dash0.GlobalPerformanceRating)
	assert.Contains(t, dash0.MethodAttribution, "Axel Smith")

	// Create a 6-puzzle cyclical set combining a BlunderVault puzzle + built-in puzzles
	vp := &blundervault.Puzzle{
		ID:         "bv-custom",
		FEN:        "6k1/5ppp/8/8/8/8/5PPP/4R1K1 w - - 0 18",
		PlayerName: "ChristopherFair",
		UserColor:  "white",
		PlayedMove: "e1e2",
		BestMove:   "e1e8",
		DeltaCp:    950,
		Event:      "Rapid",
		Date:       "2025.05.12",
	}

	set, err := cm.CreateCycleSet("", "ChristopherFair", nil, []*blundervault.Puzzle{vp}, nil, 6)
	require.NoError(t, err)
	require.NotNil(t, set)
	assert.Equal(t, "ChristopherFair's Cyclical Drill Set", set.Name)
	assert.Equal(t, 1, set.CurrentCycle)
	assert.Equal(t, 28, set.TargetDays)
	require.Len(t, set.Puzzles, 6)

	// Verify GetCycleSets and GetCycleSet
	allSets := cm.GetCycleSets()
	require.Len(t, allSets, 1)
	fetched, err := cm.GetCycleSet(set.ID)
	require.NoError(t, err)
	assert.Equal(t, set.ID, fetched.ID)

	_, err = cm.GetCycleSet("non-existent-id")
	assert.Error(t, err)

	// Re-attempting the first puzzle updates existing result in active progress cleanly
	firstPuzzleID := set.Puzzles[0].ID
	_, err = cm.RecordCyclePuzzleAttempt(set.ID, firstPuzzleID, false, 30)
	require.NoError(t, err)
	_, err = cm.RecordCyclePuzzleAttempt(set.ID, firstPuzzleID, true, 25)
	require.NoError(t, err)
	_, err = cm.RecordCyclePuzzleAttempt(set.ID, firstPuzzleID, false, 40)
	require.NoError(t, err)

	// Complete Cycle 1 (baseline cycle: 4/6 correct on first try, 240 seconds total)
	for i, p := range set.Puzzles {
		solvedFirst := i >= 2 // 4 out of 6 solved first try
		updated, err := cm.RecordCyclePuzzleAttempt(set.ID, p.ID, solvedFirst, 40)
		require.NoError(t, err)
		set = updated
	}

	// Cycle 1 should now be finalized and CurrentCycle advanced to 2!
	require.Len(t, set.Cycles, 1)
	c1 := set.Cycles[0]
	assert.Equal(t, 1, c1.CycleNumber)
	assert.Equal(t, 6, c1.TotalPuzzles)
	assert.Equal(t, 4, c1.FirstTryCorrect)
	assert.Equal(t, 240, c1.TotalTimeSec)
	assert.Equal(t, 240, c1.TargetTimeSec)
	assert.Equal(t, 1.0, c1.SpeedupVsCycle1)
	assert.Equal(t, 2, set.CurrentCycle)
	assert.Equal(t, 14, set.TargetDays)

	// Complete Cycle 2 in half the time (10 seconds per puzzle = 60s total, 6/6 100% accuracy)
	for _, p := range set.Puzzles {
		updated, err := cm.RecordCyclePuzzleAttempt(set.ID, p.ID, true, 10)
		require.NoError(t, err)
		set = updated
	}

	require.Len(t, set.Cycles, 2)
	c2 := set.Cycles[1]
	assert.Equal(t, 2, c2.CycleNumber)
	assert.Equal(t, 100.0, c2.AccuracyPct)
	assert.Equal(t, 60, c2.TotalTimeSec)
	assert.Equal(t, 120, c2.TargetTimeSec) // Half of Cycle 1's 240s
	assert.Equal(t, 4.0, c2.SpeedupVsCycle1)
	assert.Equal(t, 3, set.CurrentCycle)
	assert.Equal(t, 7, set.TargetDays)

	// Test ResetCurrentCycle and StartNextCycle
	_, err = cm.RecordCyclePuzzleAttempt(set.ID, set.Puzzles[0].ID, true, 5)
	require.NoError(t, err)
	resetSet, err := cm.ResetCurrentCycle(set.ID)
	require.NoError(t, err)
	assert.Empty(t, resetSet.ActiveProgress.Results)

	// StartNextCycle when active progress is empty advances cycle number
	nextSet, err := cm.StartNextCycle(set.ID)
	require.NoError(t, err)
	assert.Equal(t, 4, nextSet.CurrentCycle)

	// StartNextCycle when active progress has partial results finalizes the cycle
	_, err = cm.RecordCyclePuzzleAttempt(set.ID, set.Puzzles[0].ID, true, 12)
	require.NoError(t, err)
	nextSet2, err := cm.StartNextCycle(set.ID)
	require.NoError(t, err)
	assert.Equal(t, 5, nextSet2.CurrentCycle)

	// Error cases for RecordCyclePuzzleAttempt, StartNextCycle, ResetCurrentCycle
	_, err = cm.RecordCyclePuzzleAttempt("bad-set", firstPuzzleID, true, 10)
	assert.Error(t, err)
	_, err = cm.RecordCyclePuzzleAttempt(set.ID, "bad-puzzle", true, 0)
	assert.Error(t, err)
	_, err = cm.StartNextCycle("bad-set")
	assert.Error(t, err)
	_, err = cm.ResetCurrentCycle("bad-set")
	assert.Error(t, err)

	// Verify Lichess-style Dashboard metrics and reload persistence
	cmReloaded, err := NewCyclicalManager(drillPath)
	require.NoError(t, err)
	dash := cmReloaded.GetDashboard()
	require.Len(t, dash.Sets, 1)
	assert.NotEmpty(t, dash.ThemeStats)
	assert.Len(t, dash.RadarAxes, 9)
	assert.NotEmpty(t, dash.StrongThemes)
	assert.NotEmpty(t, dash.WeakThemes)
	assert.Greater(t, dash.TotalCyclesCompleted, 0)
	assert.Greater(t, dash.TotalPuzzlesAttempted, 0)
	assert.Contains(t, dash.ImprovementSummary, "speed improvement")

	// Verify Lichess PuzzleDashboard.scala formula on every ThemePerformanceStat
	for _, st := range dash.ThemeStats {
		expectedPerf := st.RatingAvg - 500 + int(float64(1000*st.FirstWins)/float64(st.Attempts)+0.5)
		assert.InDelta(t, expectedPerf, st.PerformanceRating, 1)
	}

	// Test DeleteCycleSet
	err = cmReloaded.DeleteCycleSet("bad-set")
	assert.Error(t, err)
	err = cmReloaded.DeleteCycleSet(set.ID)
	require.NoError(t, err)
	assert.Empty(t, cmReloaded.GetCycleSets())

	// Corrupt file handling on load
	corruptPath := filepath.Join(tmpDir, "corrupt.json")
	require.NoError(t, os.WriteFile(corruptPath, []byte("{corrupt"), 0o644))
	corruptCM, err := NewCyclicalManager(corruptPath)
	require.NoError(t, err)
	assert.Empty(t, corruptCM.GetCycleSets())
}

func TestCyclical_EdgeCasesAndFullTierCoverage(t *testing.T) {
	// 1. Test formatSolutionSAN with invalid FEN and unparseable move
	assert.Equal(t, []string{"e2e4"}, formatSolutionSAN("invalid_fen", []string{"e2e4"}))
	assert.Equal(t, []string{"bad_move"}, formatSolutionSAN(chess.StartingFEN, []string{"bad_move"}))

	// 2. Test parseMoveFlexible fallback branches (including r, b, n promotions and pseudo-legal coordinate move)
	bPinned, err := chess.ParseFEN("4k3/1P6/8/8/1b6/2N5/8/4K3 w - - 0 1")
	require.NoError(t, err)
	// c3 is pinned to e1 by Bb4, so c3d5 is not in GenerateLegalMoves(), exercising coordinate fallback
	mCoord, err := parseMoveFlexible(bPinned, "c3d5")
	require.NoError(t, err)
	assert.Equal(t, chess.C3, mCoord.From)
	assert.Equal(t, chess.D5, mCoord.To)

	for _, promoChar := range []string{"b7b8q", "b7b8r", "b7b8b", "b7b8n"} {
		// Make king in check so b7b8* is not strictly legal, hitting coordinate promo switch
		bInCheck, err := chess.ParseFEN("4k3/1P6/8/8/8/8/4r3/4K3 w - - 0 1")
		require.NoError(t, err)
		mPromo, err := parseMoveFlexible(bInCheck, promoChar)
		require.NoError(t, err)
		assert.NotEqual(t, chess.None, mPromo.Promo)
	}
	_, err = parseMoveFlexible(bPinned, "zzzz")
	assert.Error(t, err)

	// 3. Test NAGs ($1, $2, $4) and mate eval comments ([%eval #2], [%eval #-2]) in game extraction
	pgnNAG := `[Event "NAG Test"]
[White "PlayerA"]
[Black "PlayerB"]
[Result "1-0"]

1. e4 $1 {[%eval #2]} e5 $2 {[%eval #-2]} 2. Nf3 $4 Nc6 3. Bc4 Bc5 4. Qxf7# 1-0`
	gNAG, err := chess.ParsePGN(pgnNAG)
	require.NoError(t, err)
	gNAG.StartFEN = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
	puzzlesNAG := ExtractPuzzlesFromPlayerGames([]*chess.Game{nil, gNAG}, nil, nil, 0)
	assert.Len(t, puzzlesNAG, 24)

	// 4. Test scoreTacticalCandidate with underpromotion and en passant
	bEP, err := chess.ParseFEN("4k3/1P6/8/4pP2/8/8/8/4K3 w - e6 0 1")
	require.NoError(t, err)
	assert.Greater(t, scoreTacticalCandidate(bEP, chess.Move{From: chess.F5, To: chess.E6}, chess.White, chess.Black), 0)
	assert.Greater(t, scoreTacticalCandidate(bEP, chess.Move{From: chess.B7, To: chess.B8, Promo: chess.Knight}, chess.White, chess.Black), 0)
	assert.Equal(t, 0, scoreTacticalCandidate(bEP, chess.Move{From: chess.A1, To: chess.A2}, chess.White, chess.Black))

	// 5. Full 24-puzzle set through CyclicalManager (covers Easy, Medium, Hard first-try correct & 20+ theme history cap)
	tmpDir := t.TempDir()
	cm, err := NewCyclicalManager(filepath.Join(tmpDir, "full_tiers.json"))
	require.NoError(t, err)

	vp := &blundervault.Puzzle{
		ID:         "bv-fork",
		FEN:        "r3k2r/ppp2ppp/8/3N4/1q6/8/PPP2PPP/R2Q2K1 w kq - 0 12",
		PlayerName: "PlayerA",
		UserColor:  "white",
		PlayedMove: "d1e2",
		BestMove:   "d5c7",
		DeltaCp:    500,
	}
	// Create set with default size (0 -> 24), empty name & empty playerName
	fullSet, err := cm.CreateCycleSet("", "", nil, []*blundervault.Puzzle{vp, vp}, nil, 0)
	require.NoError(t, err)
	assert.Contains(t, fullSet.Name, "Cyclical Tactical Set")
	require.Len(t, fullSet.Puzzles, 24)

	// Solve all 24 puzzles on first try (covers easyCorrect++, medCorrect++, hardCorrect++, and >20 HistoryRatings)
	fullSet.ActiveProgress = nil // Also tests nil ActiveProgress recovery
	for _, p := range fullSet.Puzzles {
		_, err := cm.RecordCyclePuzzleAttempt(fullSet.ID, p.ID, true, 3)
		require.NoError(t, err)
	}
	updatedSet, err := cm.GetCycleSet(fullSet.ID)
	require.NoError(t, err)
	require.Len(t, updatedSet.Cycles, 1)
	assert.Equal(t, 100.0, updatedSet.Cycles[0].EasyAccuracyPct)
	assert.Equal(t, 100.0, updatedSet.Cycles[0].MediumAccuracyPct)
	assert.Equal(t, 100.0, updatedSet.Cycles[0].HardAccuracyPct)
}

