package fairplay

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"newscid/pkg/chess"
)

func TestHardMoveBenchmarkDetectorAndSelectiveSimulation(t *testing.T) {
	// Mock Chess.com and Lichess status APIs to test confirmed banned opponent validation
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "suspectengine"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"closed:fair_play_violations"}`))
		default:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"basic","tosViolation":false}`))
		}
	}))
	defer srv.Close()

	// Construct 25 benchmark games for "HeroUser":
	// - 20 normal human games where HeroUser and HonestOpp have realistic centipawn losses on complex middlegame moves
	// - 5 opponent games where "SuspectEngine" finds every single 0.00cp top move across 12 middlegame plies
	var games []*chess.Game
	var gameIDs []int

	const normalHumanPGNTemplate = `[Event "Live Chess Rapid"]
[Site "Chess.com"]
[Date "2026.10.%02d"]
[White "HeroUser"]
[Black "HonestOpp%d"]
[Result "1-0"]
[WhiteElo "1650"]
[BlackElo "1645"]
[TimeControl "600"]

1. e4 {[%%eval 0.20]} e5 {[%%eval 0.25]} 2. Nf3 {[%%eval 0.22]} Nc6 {[%%eval 0.25]} 3. Bb5 {[%%eval 0.30]} a6 {[%%eval 0.32]} 4. Ba4 {[%%eval 0.30]} Nf6 {[%%eval 0.32]} 5. O-O {[%%eval 0.30]} Be7 {[%%eval 0.35]} 6. Re1 {[%%eval 0.10]} b5 {[%%eval 0.45]} 7. Bb3 {[%%eval 0.15]} d6 {[%%eval 0.50]} 8. c3 {[%%eval 0.18]} O-O {[%%eval 0.52]} 9. h3 {[%%eval 0.10]} Na5 {[%%eval 0.48]} 10. Bc2 {[%%eval 0.12]} c5 {[%%eval 0.45]} 11. d4 {[%%eval 0.05]} Qc7 {[%%eval 0.42]} 12. Nbd2 {[%%eval 0.02]} cxd4 {[%%eval 0.40]} 13. cxd4 {[%%eval 0.05]} Nc6 {[%%eval 0.44]} 14. Nb3 {[%%eval -0.15]} a5 {[%%eval 0.28]} 15. Be3 {[%%eval -0.05]} a4 {[%%eval 0.35]} 16. Nbd2 {[%%eval 0.00]} Bd7 {[%%eval 0.38]} 1-0
`

	const engineOpponentPGNTemplate = `[Event "Live Chess Rapid"]
[Site "Chess.com"]
[Date "2026.10.%02d"]
[White "HeroUser"]
[Black "SuspectEngine"]
[Result "0-1"]
[WhiteElo "1650"]
[BlackElo "1650"]
[TimeControl "600"]

1. e4 {[%%eval 0.20]} e5 {[%%eval 0.20]} 2. Nf3 {[%%eval 0.20]} Nc6 {[%%eval 0.20]} 3. Bb5 {[%%eval 0.25]} a6 {[%%eval 0.25]} 4. Ba4 {[%%eval 0.25]} Nf6 {[%%eval 0.25]} 5. O-O {[%%eval 0.25]} Be7 {[%%eval 0.25]} 6. Re1 {[%%eval 0.00]} b5 {[%%eval 0.00]} 7. Bb3 {[%%eval -0.35]} d6 {[%%eval -0.35]} 8. c3 {[%%eval -0.60]} O-O {[%%eval -0.60]} 9. h3 {[%%eval -0.90]} Na5 {[%%eval -0.90]} 10. Bc2 {[%%eval -1.20]} c5 {[%%eval -1.20]} 11. d4 {[%%eval -1.55]} Qc7 {[%%eval -1.55]} 12. Nbd2 {[%%eval -1.90]} cxd4 {[%%eval -1.90]} 13. cxd4 {[%%eval -2.30]} Nc6 {[%%eval -2.30]} 14. Nb3 {[%%eval -2.75]} a5 {[%%eval -2.75]} 15. Be3 {[%%eval -3.20]} a4 {[%%eval -3.20]} 16. Nbd2 {[%%eval -3.70]} Bd7 {[%%eval -3.70]} 0-1
`

	for i := 1; i <= 20; i++ {
		g, err := chess.ParsePGN(fmt.Sprintf(normalHumanPGNTemplate, (i%28)+1, i))
		if err != nil {
			t.Fatalf("failed to parse normal PGN: %v", err)
		}
		games = append(games, g)
		gameIDs = append(gameIDs, len(games))
	}

	for i := 1; i <= 5; i++ {
		g, err := chess.ParsePGN(fmt.Sprintf(engineOpponentPGNTemplate, i))
		if err != nil {
			t.Fatalf("failed to parse engine PGN: %v", err)
		}
		games = append(games, g)
		gameIDs = append(gameIDs, len(games))
	}

	report := BuildHardMoveBenchmarkReport("HeroUser", "ALL", games, gameIDs, true, srv.URL, srv.URL)
	if report == nil {
		t.Fatalf("expected non-nil HardMoveBenchmarkReport")
	}
	if report.SelfGamesQualified != 25 || report.OpponentGamesQualified != 25 {
		t.Fatalf("expected 25 qualified self and opponent games, got self=%d opp=%d", report.SelfGamesQualified, report.OpponentGamesQualified)
	}
	if report.TotalHardMovesAnalyzed <= 0 {
		t.Fatalf("expected positive TotalHardMovesAnalyzed, got %d", report.TotalHardMovesAnalyzed)
	}
	if report.OpponentInTopTwoBands < 5 {
		t.Fatalf("expected all 5 SuspectEngine games to land in Top Two Bands, got %d", report.OpponentInTopTwoBands)
	}
	if report.ExcessOpponentGames <= 0 {
		t.Fatalf("expected positive ExcessOpponentGames, got %.2f", report.ExcessOpponentGames)
	}
	if len(report.Bands) != 5 {
		t.Fatalf("expected 5 benchmark bands, got %d", len(report.Bands))
	}
	if len(report.SelectiveSimulation) != 5 {
		t.Fatalf("expected 5 selective simulation points, got %d", len(report.SelectiveSimulation))
	}
	// Verify selective cheating simulation increases Top-Two-Band detection rate monotonically
	if report.SelectiveSimulation[4].TopTwoBandsCaughtPct <= report.SelectiveSimulation[0].TopTwoBandsCaughtPct {
		t.Fatalf("expected 100%% swapped hard moves to have higher detection rate than 0 swaps, got %.1f%% vs %.1f%%",
			report.SelectiveSimulation[4].TopTwoBandsCaughtPct, report.SelectiveSimulation[0].TopTwoBandsCaughtPct)
	}
	// Verify confirmed banned opponent cross-check detected "suspectengine"
	if report.BannedValidation.ConfirmedBannedCount != 1 || report.BannedValidation.GamesVsBanned != 5 {
		t.Fatalf("expected 1 confirmed banned handle with 5 games, got %+v", report.BannedValidation)
	}
}
