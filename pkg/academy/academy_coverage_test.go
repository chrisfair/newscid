package academy

import (
	"math/rand"
	"os"
	"path/filepath"
	"testing"

	"newscid/pkg/chess"
	"newscid/pkg/database/scid"
)

func TestAcademyCoverageBranches(t *testing.T) {
	// DefaultCareerPath
	if p := DefaultCareerPath(); p == "" {
		t.Fatalf("expected non-empty DefaultCareerPath")
	}

	// Corrupt JSON on disk falls back cleanly to default profile
	dir := t.TempDir()
	corruptPath := filepath.Join(dir, "career.json")
	if err := os.WriteFile(corruptPath, []byte("{bad-json"), 0o644); err != nil {
		t.Fatal(err)
	}
	mgr := NewManager(corruptPath)
	prof := mgr.GetOverview().Profile
	if prof.CurrentTier != 1 {
		t.Fatalf("expected default tier 1 on corrupt json, got %d", prof.CurrentTier)
	}

	// eloToTier all tiers
	for _, tc := range []struct {
		elo  int
		tier int
	}{
		{800, 1}, {1100, 2}, {1400, 3}, {1750, 4}, {2050, 5}, {2500, 6},
	} {
		if got := eloToTier(tc.elo); got != tc.tier {
			t.Fatalf("eloToTier(%d) = %d, want %d", tc.elo, got, tc.tier)
		}
	}

	// deriveStyleTag all styles
	tags := []PersonalitySliders{
		{Aggression: 80},
		{TradeTendency: 80},
		{PositionalSolidity: 85},
		{TacticalSharpness: 80},
		{Aggression: 50, TradeTendency: 50, PositionalSolidity: 50, TacticalSharpness: 50},
	}
	for _, s := range tags {
		if deriveStyleTag(s) == "" {
			t.Fatalf("empty style tag for %+v", s)
		}
	}

	// clampInt bounds
	if clampInt(-5, 0, 10) != 0 || clampInt(25, 0, 10) != 10 || clampInt(5, 0, 10) != 5 {
		t.Fatalf("clampInt failed")
	}

	// pieceChar and pieceCharName including out-of-range
	for _, pt := range []chess.PieceType{chess.None, chess.Pawn, chess.Knight, chess.Bishop, chess.Rook, chess.Queen, chess.King, chess.PieceType(99)} {
		_ = pieceChar(pt)
		if pieceCharName(pt) == "" {
			t.Fatalf("empty pieceCharName for %v", pt)
		}
	}

	// BuildPersonalityFromDB with nil DB returns error
	if _, err := mgr.BuildPersonalityFromDB(nil, "Nobody"); err == nil {
		t.Fatalf("expected error for nil db")
	}

	// BuildPersonalityFromDB with low-Elo + long game + black ECOs
	dbPath := filepath.Join(dir, "longgames")
	db := scid.NewDatabase()
	if err := db.Create(dbPath, "Long"); err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	pgn := `[Event "Club"]
[Site "Home"]
[Date "2026.01.01"]
[White "Opponent"]
[Black "Grinder"]
[Result "0-1"]
[WhiteElo "1500"]
[BlackElo "0"]
[ECO "B20"]

1. e4 c5 2. Nf3 d6 3. d4 cxd4 4. Nxd4 Nf6 5. Nc3 a6 6. Be2 e5 7. Nb3 Be7 8. O-O O-O 0-1`
	g, err := chess.ParsePGN(pgn)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.SaveGame(g); err != nil {
		t.Fatal(err)
	}
	if pers, err := mgr.BuildPersonalityFromDB(db, "Grinder"); err != nil || pers.ID == "" {
		t.Fatalf("BuildPersonalityFromDB failed: %v", err)
	}

	// SelectPersonalityMove error on invalid FEN and checkmate position
	clara, _ := FindBuiltinPersonality("t1_clara")
	if _, err := SelectPersonalityMove("not-a-fen", clara, nil, rand.New(rand.NewSource(1))); err == nil {
		t.Fatalf("expected error for invalid FEN")
	}
	// Checkmate position (no legal moves)
	if _, err := SelectPersonalityMove("7k/5Q2/6K1/8/8/8/8/8 b - - 0 1", clara, nil, rand.New(rand.NewSource(1))); err == nil {
		t.Fatalf("expected error on checkmate position")
	}
	// High blunder rate sampling with multiple candidates + nil rng
	clara.Sliders.BlunderRate = 100
	clara.OpeningBook = nil
	res, err := SelectPersonalityMove(chess.StartingFEN, clara, []CandidateMove{
		{UCI: "e2e4", ScoreCp: 30},
		{UCI: "d2d4", ScoreCp: 25},
		{UCI: "g1f3", ScoreCp: 20},
		{UCI: "b1c3", ScoreCp: 10},
		{UCI: "invalid", ScoreCp: 999},
	}, nil)
	if err != nil || res.UCI == "" {
		t.Fatalf("expected valid move with blunder sampling, got %+v err=%v", res, err)
	}

	// DiagnoseGameForCoach leaving a knight hanging (hits pieceCharName!)
	hangPGN := `[Event "Hang"]
[White "Hero"]
[Black "Bot"]
[Result "0-1"]

1. e4 e5 2. Nf3 Nc6 3. Nxe5 0-1`
	rep, err := DiagnoseGameForCoach(hangPGN, "white")
	if err != nil || rep.Headline == "" {
		t.Fatalf("DiagnoseGameForCoach hanging piece failed: %+v err=%v", rep, err)
	}

	// Empty JSON object initializes all zero/nil career defaults cleanly
	emptyPath := filepath.Join(dir, "empty_career.json")
	if err := os.WriteFile(emptyPath, []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	emptyMgr := NewManager(emptyPath)
	if emptyMgr.GetOverview().Profile.CurrentTier != 1 {
		t.Fatalf("expected default tier 1 on empty json")
	}

	// Invalid FENs & PGN error branches
	if _, err := ComputeSquareControl("bad-fen"); err == nil {
		t.Fatalf("expected error on bad FEN in ComputeSquareControl")
	}
	if _, err := ComputeSpyThreat("bad-fen"); err == nil {
		t.Fatalf("expected error on bad FEN in ComputeSpyThreat")
	}
	if _, err := DiagnoseGameForCoach("", "white"); err == nil {
		t.Fatalf("expected error on empty PGN in DiagnoseGameForCoach")
	}

	// Spy Mode: opponent threatens checkmate on next move (Qa1#) and no-threat quiet position
	spyMate, err := ComputeSpyThreat("4k3/8/8/8/8/8/1q3PPP/5RK1 w - - 0 1")
	if err != nil {
		t.Fatalf("ComputeSpyThreat mate failed: %v", err)
	}
	_ = spyMate
	spyQuiet, err := ComputeSpyThreat("4k3/8/8/8/8/8/8/4K3 w - - 0 1")
	if err != nil || spyQuiet.HasThreat {
		t.Fatalf("expected no threat in bare kings, got %+v err=%v", spyQuiet, err)
	}
}
