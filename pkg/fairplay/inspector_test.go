package fairplay

import (
	"strings"
	"testing"

	"newscid/pkg/chess"
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

