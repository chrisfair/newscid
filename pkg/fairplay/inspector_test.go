package fairplay

import (
	"strings"
	"testing"

	"newscid/pkg/chess"
	"newscid/pkg/database/scid"
)

func TestClassifyTimeControl(t *testing.T) {
	cases := []struct {
		tc    string
		event string
		want  string
	}{
		{"60+0", "Rated game", "Bullet"},
		{"120+1", "Online", "Bullet"},
		{"180+0", "Rated game", "Blitz"},
		{"300+3", "Rated game", "Blitz"},
		{"600+0", "Rated game", "Rapid"},
		{"900+10", "Rated game", "Rapid"},
		{"1800+30", "Tournament", "Classical"},
		{"-", "Rated Rapid game", "Rapid"},
	}

	for _, tc := range cases {
		got := ClassifyTimeControl(tc.tc, tc.event)
		if got != tc.want {
			t.Errorf("ClassifyTimeControl(%q, %q) = %q, want %q", tc.tc, tc.event, got, tc.want)
		}
	}
}

func TestBuildPlayerFairPlayDossier_CrossControlConsistency(t *testing.T) {
	// Create 2 Bullet games (human variance, fast/variable move times, normal eval drops)
	// and 2 Rapid games across 2 days (flat 5s robotic move times, near-zero eval drops)
	pgnText := `
[Event "Rated Bullet game"]
[Site "https://lichess.org/b1"]
[Date "2026.10.14"]
[White "SuspectPlayer"]
[Black "OpponentA"]
[Result "0-1"]
[WhiteElo "1650"]
[BlackElo "1660"]
[TimeControl "60+0"]

1. e4 { [%eval 0.2] [%clk 0:01:00] } e5 { [%eval 0.2] [%clk 0:01:00] }
2. Nf3 { [%eval 0.2] [%clk 0:00:59] } Nc6 { [%eval 0.2] [%clk 0:00:59] }
3. Bc4 { [%eval 0.2] [%clk 0:00:58] } Bc5 { [%eval 0.2] [%clk 0:00:58] }
4. c3 { [%eval 0.1] [%clk 0:00:57] } Nf6 { [%eval 0.1] [%clk 0:00:57] }
5. d4 { [%eval 0.1] [%clk 0:00:56] } exd4 { [%eval 0.1] [%clk 0:00:56] }
6. cxd4 { [%eval 0.1] [%clk 0:00:55.5] } Bb4+ { [%eval 0.1] [%clk 0:00:55] }
7. Nc3 { [%eval -0.8] [%clk 0:00:46] } Nxe4 { [%eval -0.8] [%clk 0:00:54] }
8. O-O { [%eval -1.4] [%clk 0:00:45.2] } Bxc3 { [%eval -1.4] [%clk 0:00:53] }
9. bxc3 { [%eval -1.5] [%clk 0:00:44.8] } d5 { [%eval -1.5] [%clk 0:00:52] }
10. Bd3 { [%eval -2.2] [%clk 0:00:32.0] } O-O { [%eval -2.2] [%clk 0:00:50] }
11. Qc2 { [%eval -2.9] [%clk 0:00:31.2] } Bf5 { [%eval -2.9] [%clk 0:00:48] }
12. Ba3 { [%eval -3.5] [%clk 0:00:19.0] } Re8 { [%eval -3.5] [%clk 0:00:46] }
13. Rfe1 { [%eval -4.2] [%clk 0:00:18.4] } Bg6 { [%eval -4.2] [%clk 0:00:44] } 0-1

[Event "Rated Rapid game"]
[Site "https://lichess.org/r1"]
[Date "2026.10.15"]
[White "SuspectPlayer"]
[Black "OpponentB"]
[Result "1-0"]
[WhiteElo "1650"]
[BlackElo "1700"]
[TimeControl "600+0"]

1. e4 { [%eval 0.2] [%clk 0:10:00] } e5 { [%eval 0.2] [%clk 0:10:00] }
2. Nf3 { [%eval 0.2] [%clk 0:09:55] } Nc6 { [%eval 0.2] [%clk 0:09:55] }
3. Bb5 { [%eval 0.3] [%clk 0:09:50] } a6 { [%eval 0.3] [%clk 0:09:50] }
4. Ba4 { [%eval 0.3] [%clk 0:09:45] } Nf6 { [%eval 0.3] [%clk 0:09:45] }
5. O-O { [%eval 0.3] [%clk 0:09:40] } Be7 { [%eval 0.3] [%clk 0:09:40] }
6. Re1 { [%eval 0.35] [%clk 0:09:35] } b5 { [%eval 0.4] [%clk 0:09:35] }
7. Bb3 { [%eval 0.4] [%clk 0:09:30] } d6 { [%eval 0.45] [%clk 0:09:30] }
8. c3 { [%eval 0.45] [%clk 0:09:25] } O-O { [%eval 0.5] [%clk 0:09:25] }
9. h3 { [%eval 0.5] [%clk 0:09:20] } Na5 { [%eval 0.6] [%clk 0:09:20] }
10. Bc2 { [%eval 0.6] [%clk 0:09:15] } c5 { [%eval 0.7] [%clk 0:09:15] }
11. d4 { [%eval 0.7] [%clk 0:09:10] } Qc7 { [%eval 0.8] [%clk 0:09:10] }
12. Nbd2 { [%eval 0.8] [%clk 0:09:05] } cxd4 { [%eval 0.9] [%clk 0:09:05] }
13. cxd4 { [%eval 0.9] [%clk 0:09:00] } Nc6 { [%eval 1.1] [%clk 0:09:00] }
14. Nb3 { [%eval 1.1] [%clk 0:08:55] } a5 { [%eval 1.4] [%clk 0:08:55] }
15. Be3 { [%eval 1.4] [%clk 0:08:50] } a4 { [%eval 1.8] [%clk 0:08:50] }
16. Nbd2 { [%eval 1.8] [%clk 0:08:45] } Bd7 { [%eval 2.2] [%clk 0:08:45] }
17. Rc1 { [%eval 2.2] [%clk 0:08:40] } Qb7 { [%eval 2.8] [%clk 0:08:40] }
18. Bb1 { [%eval 2.8] [%clk 0:08:35] } Rfc8 { [%eval 3.4] [%clk 0:08:35] } 1-0

[Event "Rated Rapid game"]
[Site "https://lichess.org/r2"]
[Date "2026.10.16"]
[White "SuspectPlayer"]
[Black "OpponentC"]
[Result "1-0"]
[WhiteElo "1660"]
[BlackElo "1720"]
[TimeControl "600+0"]

1. d4 { [%eval 0.2] [%clk 0:10:00] } Nf6 { [%eval 0.2] [%clk 0:10:00] }
2. c4 { [%eval 0.25] [%clk 0:09:55] } e6 { [%eval 0.25] [%clk 0:09:55] }
3. Nc3 { [%eval 0.3] [%clk 0:09:50] } Bb4 { [%eval 0.3] [%clk 0:09:50] }
4. e3 { [%eval 0.3] [%clk 0:09:45] } O-O { [%eval 0.3] [%clk 0:09:45] }
5. Bd3 { [%eval 0.35] [%clk 0:09:40] } d5 { [%eval 0.35] [%clk 0:09:40] }
6. Nf3 { [%eval 0.4] [%clk 0:09:35] } c5 { [%eval 0.4] [%clk 0:09:35] }
7. O-O { [%eval 0.45] [%clk 0:09:30] } cxd4 { [%eval 0.45] [%clk 0:09:30] }
8. exd4 { [%eval 0.45] [%clk 0:09:25] } dxc4 { [%eval 0.5] [%clk 0:09:25] }
9. Bxc4 { [%eval 0.5] [%clk 0:09:20] } b6 { [%eval 0.6] [%clk 0:09:20] }
10. Bg5 { [%eval 0.6] [%clk 0:09:15] } Bb7 { [%eval 0.7] [%clk 0:09:15] }
11. Qe2 { [%eval 0.7] [%clk 0:09:10] } Nbd7 { [%eval 0.85] [%clk 0:09:10] }
12. Rac1 { [%eval 0.85] [%clk 0:09:05] } Rc8 { [%eval 1.1] [%clk 0:09:05] }
13. Rfd1 { [%eval 1.1] [%clk 0:09:00] } Bxc3 { [%eval 1.4] [%clk 0:09:00] }
14. bxc3 { [%eval 1.4] [%clk 0:08:55] } Qc7 { [%eval 1.7] [%clk 0:08:55] }
15. Bd3 { [%eval 1.7] [%clk 0:08:50] } Rfe8 { [%eval 2.1] [%clk 0:08:50] }
16. c4 { [%eval 2.1] [%clk 0:08:45] } h6 { [%eval 2.6] [%clk 0:08:45] }
17. Bh4 { [%eval 2.6] [%clk 0:08:40] } Nh5 { [%eval 3.2] [%clk 0:08:40] }
18. Bg3 { [%eval 3.2] [%clk 0:08:35] } Nxg3 { [%eval 3.8] [%clk 0:08:35] } 1-0
`

	games, err := chess.ParsePGNDatabase(strings.NewReader(pgnText))
	if err != nil {
		t.Fatalf("ParsePGNDatabase failed: %v", err)
	}

	dossier := BuildPlayerFairPlayDossier("SuspectPlayer", games, []int{1, 2, 3})
	if dossier.TotalGamesAnalyzed != 3 {
		t.Fatalf("expected 3 games analyzed, got %d", dossier.TotalGamesAnalyzed)
	}
	if dossier.TotalDaysActive != 3 {
		t.Fatalf("expected 3 active days, got %d", dossier.TotalDaysActive)
	}
	if dossier.HighestRiskTimeControl != "Rapid" {
		t.Errorf("expected HighestRiskTimeControl = Rapid, got %q", dossier.HighestRiskTimeControl)
	}
	if !dossier.CrossDivergence.HasMultipleControls {
		t.Errorf("expected CrossDivergence.HasMultipleControls = true")
	}
	if dossier.CrossDivergence.SuspectControl != "Rapid" || dossier.CrossDivergence.BaselineControl != "Bullet" {
		t.Errorf("unexpected CrossDivergence controls: %+v", dossier.CrossDivergence)
	}
	if dossier.HighestControlIssueScore < 70.0 {
		t.Errorf("expected Rapid ConsistentIssueScore >= 70.0, got %.1f", dossier.HighestControlIssueScore)
	}
	if !strings.Contains(dossier.ReportMarkdown, "Per-Time-Control Consistent Issue Matrix") {
		t.Errorf("expected ReportMarkdown to include Per-Time-Control Consistent Issue Matrix")
	}
}

func TestAnalyzePlayerFairPlay_SingleGameDatabaseAndPartialEval(t *testing.T) {
	// Regression test for 1-based database game indexing and partially annotated games
	pgnText := `
[Event "Rated Rapid game"]
[Site "https://lichess.org/single1"]
[Date "2026.10.16"]
[White "SingleDBPlayer"]
[Black "OpponentX"]
[Result "1-0"]
[WhiteElo "1800"]
[BlackElo "1820"]
[TimeControl "600+0"]

1. e4 { [%eval 0.2] [%clk 0:10:00] } e5 { [%eval 0.2] [%clk 0:10:00] }
2. Nf3 { [%eval 0.3] [%clk 0:09:55] } Nc6 { [%eval 0.3] [%clk 0:09:55] }
3. Bb5 a6 4. Ba4 Nf6 5. O-O Be7 6. Re1 b5 7. Bb3 d6 8. c3 O-O 9. h3 Na5 10. Bc2 c5 11. d4 Qc7 12. Nbd2 cxd4 13. cxd4 1-0
`
	games, err := chess.ParsePGNDatabase(strings.NewReader(pgnText))
	if err != nil {
		t.Fatalf("ParsePGNDatabase failed: %v", err)
	}
	dossier := BuildPlayerFairPlayDossier("SingleDBPlayer", games, []int{1})
	if dossier.TotalGamesAnalyzed != 1 {
		t.Fatalf("expected 1 game analyzed, got %d", dossier.TotalGamesAnalyzed)
	}
	if dossier.AllGames[0].GameID != 1 {
		t.Errorf("expected GameID = 1, got %d", dossier.AllGames[0].GameID)
	}
}

func TestUnannotatedHumanGames_NotFalselyFlagged(t *testing.T) {
	// Regression test for unannotated games (no [%eval] or [%clk] tags):
	// Previously, evaluateBoardHeuristicWhiteCp only checked raw material after the player's own move,
	// resulting in 0.8 cp ACPL, 100% Top-1 match, 3010 IPR, and 99/100 Issue Score for every player!
	pgnText := `
[Event "Live Chess"]
[Site "Chess.com"]
[Date "2026.09.10"]
[White "QuenyenHighElf"]
[Black "ClubOpponent1"]
[Result "0-1"]
[WhiteElo "1282"]
[BlackElo "1290"]
[TimeControl "600"]

1. e4 e5 2. Nf3 Nc6 3. Bc4 Nf6 4. Ng5 d5 5. exd5 Nxd5 6. Nxf7 Kxf7 7. Qf3+ Ke6 8. Nc3 Nce7 9. d4 c6 10. Bg5 h6 11. Bxe7 Bxe7 12. O-O-O Rf8 13. Qe4 Rf4 14. Qxe5+ Kf7 15. Nxd5 cxd5 16. Bxd5+ Kf8 17. Qxf4+ Bf6 18. Rhe1 Qxd5 19. Kb1 Be6 20. b3 Rc8 0-1

[Event "Live Chess"]
[Site "Chess.com"]
[Date "2026.09.11"]
[White "ClubOpponent2"]
[Black "QuenyenHighElf"]
[Result "1-0"]
[WhiteElo "1310"]
[BlackElo "1278"]
[TimeControl "180"]

1. d4 d5 2. c4 e6 3. Nc3 Nf6 4. Bg5 Be7 5. e3 O-O 6. Nf3 b6 7. cxd5 exd5 8. Bd3 Bb7 9. O-O Nbd7 10. Rc1 c5 11. Qe2 Ne4 12. Bf4 Ndf6 13. Rfd1 Bd6 14. Bxd6 Qxd6 15. dxc5 Nxc5 16. Bb1 Rfd8 17. Nd4 Rac8 18. Qf3 Qe5 19. Qh3 g6 20. Nf3 Qe7 1-0

[Event "Live Chess"]
[Site "Chess.com"]
[Date "2026.09.12"]
[White "QuenyenHighElf"]
[Black "ClubOpponent3"]
[Result "1-0"]
[WhiteElo "1285"]
[BlackElo "1260"]
[TimeControl "60"]

1. e4 c5 2. Nf3 d6 3. d4 cxd4 4. Nxd4 Nf6 5. Nc3 a6 6. Be2 e5 7. Nb3 Be7 8. O-O O-O 9. Be3 Be6 10. f3 Nbd7 11. Qd2 b5 12. Rfd1 Nb6 13. Qe1 Qc7 14. Qf2 Nc4 15. Bxc4 bxc4 16. Nd2 Rab8 17. Rab1 Rfc8 18. Nf1 Bf8 19. Bg5 Nd7 20. Ne3 f6 1-0
`
	games, err := chess.ParsePGNDatabase(strings.NewReader(pgnText))
	if err != nil {
		t.Fatalf("ParsePGNDatabase failed: %v", err)
	}
	dossier := BuildPlayerFairPlayDossier("QuenyenHighElf", games, []int{1, 2, 3})
	if dossier.OverallIssueScore >= 38.0 {
		t.Errorf("expected unannotated normal human play to have OverallIssueScore < 38.0, got %.1f (%s)",
			dossier.OverallIssueScore, dossier.OverallClassification)
	}
	if dossier.OverallClassification != "Normal Human Variance" {
		t.Errorf("expected Normal Human Variance, got %q", dossier.OverallClassification)
	}
	for _, p := range dossier.TimeControlProfiles {
		if p.AvgACPL < 10.0 {
			t.Errorf("expected realistic human ACPL >= 10.0 cp on unannotated games for %s, got %.1f cp", p.Category, p.AvgACPL)
		}
		if p.Top1AgreementPct >= 90.0 {
			t.Errorf("expected Top1AgreementPct < 90%% on unannotated games for %s, got %.1f%%", p.Category, p.Top1AgreementPct)
		}
		if p.IntrinsicPerformanceElo > 1700 {
			t.Errorf("expected realistic IPR <= 1700 for ~1280 player in %s, got %d", p.Category, p.IntrinsicPerformanceElo)
		}
	}
	if dossier.CrossDivergence.HasMultipleControls && dossier.CrossDivergence.BaselineControl == dossier.CrossDivergence.SuspectControl {
		t.Errorf("expected distinct BaselineControl and SuspectControl when multiple controls exist, got %s vs %s",
			dossier.CrossDivergence.BaselineControl, dossier.CrossDivergence.SuspectControl)
	}
}

func TestChristopherFairDB_IfPresent(t *testing.T) {
	dbPath := "/home/christopher/ChessDataBases/ChristopherFair/ChristopherFair"
	db := scid.NewDatabase()
	if err := db.Open(dbPath); err != nil {
		t.Skipf("ChristopherFair database not present: %v", err)
	}
	defer db.Close()

	if g1, err := db.GetGame(1); err == nil && g1.RootMove != nil {
		t.Logf("Game 1: White=%s Black=%s PlyCount=%d RootSAN=%s RootClk=%q RootComment=%q",
			g1.Tag("White"), g1.Tag("Black"), g1.PlyCount, g1.RootMove.SAN, g1.RootMove.Clock, g1.RootMove.Comment)
	}
	for _, targetPlayer := range []string{"QuenyenHighElf", "ChristopherFair"} {
		dossier, err := AnalyzePlayerFairPlay(db, targetPlayer, 300)
		if err != nil {
			t.Fatalf("AnalyzePlayerFairPlay(%s) failed: %v", targetPlayer, err)
		}
		t.Logf("%s: Games=%d, OverallScore=%.1f (%s), HighestRiskTC=%s (%.1f), CrossDiv=%s vs %s (%.1f)",
			targetPlayer, dossier.TotalGamesAnalyzed, dossier.OverallIssueScore, dossier.OverallClassification,
			dossier.HighestRiskTimeControl, dossier.HighestControlIssueScore,
			dossier.CrossDivergence.BaselineControl, dossier.CrossDivergence.SuspectControl, dossier.CrossDivergence.DivergenceScore)
		for _, p := range dossier.TimeControlProfiles {
			t.Logf("  TC=%s Games=%d Win=%.1f%% OfficialElo=%d IPR=%d (%+d, %+.1fσ) ACPL=%.1f Top1=%.1f%% CritAcc=%.1f%% AvgTime=%.1fs CV=%.2f Score=%.1f",
				p.Category, p.GamesAnalyzed, p.WinRatePct, p.AvgOfficialElo, p.IntrinsicPerformanceElo, p.EloOverperformanceDelta, p.ZScore,
				p.AvgACPL, p.Top1AgreementPct, p.CriticalAccuracyPct, p.AvgMoveTimeSec, p.AvgMoveTimeCV, p.ConsistentIssueScore)
		}
		if dossier.OverallIssueScore >= 40.0 {
			t.Errorf("expected %s to have normal human score < 40.0, got %.1f (%s)",
				targetPlayer, dossier.OverallIssueScore, dossier.OverallClassification)
		}
	}
}



