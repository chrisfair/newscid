package academy

import (
	"math/rand"
	"path/filepath"
	"testing"

	"newscid/pkg/chess"
	"newscid/pkg/database/scid"
)

func TestRosterAndCurriculumIntegrity(t *testing.T) {
	tiers := Tiers()
	if len(tiers) != 6 {
		t.Fatalf("expected 6 tiers, got %d", len(tiers))
	}
	pers := BuiltinPersonalities()
	if len(pers) != 24 {
		t.Fatalf("expected 24 builtin personalities, got %d", len(pers))
	}
	for _, tier := range tiers {
		boss, ok := FindBuiltinPersonality(tier.BossID)
		if !ok || !boss.IsBoss || boss.Tier != tier.Tier {
			t.Fatalf("invalid boss for tier %d: %s", tier.Tier, tier.BossID)
		}
	}
	if _, ok := FindBuiltinPersonality("nonexistent"); ok {
		t.Fatalf("expected nonexistent personality to fail lookup")
	}

	lessons := Curriculum()
	if len(lessons) != 18 {
		t.Fatalf("expected 18 lessons, got %d", len(lessons))
	}
	for tier := 1; tier <= 6; tier++ {
		if len(LessonsForTier(tier)) != 3 {
			t.Fatalf("expected 3 lessons for tier %d", tier)
		}
	}
	if _, ok := FindLesson("missing"); ok {
		t.Fatalf("expected missing lesson to fail lookup")
	}

	// Verify EVERY lesson step has a valid FEN and EVERY expected move is legal!
	for _, l := range lessons {
		if _, ok := FindLesson(l.ID); !ok {
			t.Fatalf("FindLesson failed for %s", l.ID)
		}
		for sIdx, step := range l.Steps {
			b, err := chess.ParseFEN(step.FEN)
			if err != nil {
				t.Fatalf("lesson %s step %d invalid FEN %q: %v", l.ID, sIdx, step.FEN, err)
			}
			legal := b.GenerateLegalMoves()
			for _, uci := range step.ExpectedMoves {
				m, err := b.ParseUCI(uci)
				if err != nil {
					t.Fatalf("lesson %s step %d invalid UCI %q: %v", l.ID, sIdx, uci, err)
				}
				found := false
				for _, lm := range legal {
					if lm == m {
						found = true
						break
					}
				}
				if !found {
					t.Fatalf("lesson %s step %d expected move %s is NOT legal in FEN %s", l.ID, sIdx, uci, step.FEN)
				}
			}
		}
	}
}

func TestCareerProgressionAndBossExam(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "career.json")
	mgr := NewManager(tmp)

	ov := mgr.GetOverview()
	if ov.Profile.Rating != 900 || ov.Profile.CurrentTier != 1 {
		t.Fatalf("unexpected default profile: %+v", ov.Profile)
	}

	// Win against Tier 1 opponent
	upd, err := mgr.RecordGameResult("t1_toby", "win")
	if err != nil {
		t.Fatalf("RecordGameResult failed: %v", err)
	}
	if upd.NewRating <= upd.OldRating {
		t.Fatalf("expected rating increase on win: %+v", upd)
	}

	// Draw and loss
	if _, err := mgr.RecordGameResult("t1_clara", "draw"); err != nil {
		t.Fatalf("draw failed: %v", err)
	}
	if _, err := mgr.RecordGameResult("t1_ben", "loss"); err != nil {
		t.Fatalf("loss failed: %v", err)
	}
	if _, err := mgr.RecordGameResult("t1_ben", "bogus"); err == nil {
		t.Fatalf("expected invalid outcome error")
	}
	if _, err := mgr.RecordGameResult("unknown_bot", "win"); err == nil {
		t.Fatalf("expected unknown personality error")
	}

	// Pass Tier 1 Boss 2-game exam (win + draw = 1.5/2)
	u1, err := mgr.RecordGameResult("t1_milo_boss", "win")
	if err != nil || u1.Promoted {
		t.Fatalf("expected game 1/2 not yet promoted: %+v err=%v", u1, err)
	}
	u2, err := mgr.RecordGameResult("t1_milo_boss", "draw")
	if err != nil || !u2.Promoted || u2.NewTier != 2 {
		t.Fatalf("expected promotion to Tier 2: %+v err=%v", u2, err)
	}

	// Fail Tier 2 Boss exam (loss + loss)
	_, _ = mgr.RecordGameResult("t2_vera_boss", "loss")
	uFail, err := mgr.RecordGameResult("t2_vera_boss", "loss")
	if err != nil || uFail.Promoted {
		t.Fatalf("expected failed exam not to promote: %+v", uFail)
	}

	// Complete lesson & TestOutToTier
	prof, err := mgr.CompleteLesson("t1_piece_safety")
	if err != nil || !prof.CompletedLessons["t1_piece_safety"] {
		t.Fatalf("CompleteLesson failed: %v", err)
	}
	if _, err := mgr.CompleteLesson("nope"); err == nil {
		t.Fatalf("expected unknown lesson error")
	}
	prof, err = mgr.TestOutToTier(4)
	if err != nil || prof.CurrentTier < 4 {
		t.Fatalf("TestOutToTier failed: %v", err)
	}
	if _, err := mgr.TestOutToTier(9); err == nil {
		t.Fatalf("expected out-of-range tier error")
	}

	// Custom personality CRUD + reload persistence
	saved, err := mgr.SaveCustomPersonality(Personality{
		Name:    "My Club Rival",
		Elo:     1450,
		Sliders: PersonalitySliders{Aggression: 80, TradeTendency: 40, TacticalSharpness: 70, PositionalSolidity: 60, BlunderRate: 20},
	})
	if err != nil || saved.ID == "" {
		t.Fatalf("SaveCustomPersonality failed: %v", err)
	}
	if _, ok := mgr.FindPersonality(saved.ID); !ok {
		t.Fatalf("expected custom personality to be found")
	}
	// Update existing
	saved.Elo = 1520
	if _, err := mgr.SaveCustomPersonality(saved); err != nil {
		t.Fatalf("update custom personality failed: %v", err)
	}

	// Reload from disk
	mgr2 := NewManager(tmp)
	if _, ok := mgr2.FindPersonality(saved.ID); !ok {
		t.Fatalf("reloaded manager missing custom personality")
	}
	if err := mgr2.DeleteCustomPersonality(saved.ID); err != nil {
		t.Fatalf("DeleteCustomPersonality failed: %v", err)
	}
	if err := mgr2.DeleteCustomPersonality("missing"); err == nil {
		t.Fatalf("expected missing delete error")
	}
	if _, err := mgr2.SaveCustomPersonality(Personality{Name: ""}); err == nil {
		t.Fatalf("expected empty name error")
	}
}

func TestBuildPersonalityFromDB(t *testing.T) {
	dir := t.TempDir()
	basePath := filepath.Join(dir, "testdb")
	db := scid.NewDatabase()
	if err := db.Create(basePath, "Test Academy DB"); err != nil {
		t.Fatalf("db.Create failed: %v", err)
	}
	defer db.Close()

	pgn := `[Event "Club Open"]
[Site "Ur"]
[Date "2026.01.01"]
[Round "1"]
[White "Kasparov, Garry"]
[Black "Rival, Club"]
[WhiteElo "2800"]
[BlackElo "2500"]
[Result "1-0"]

1. e4 c5 2. Nf3 d6 3. d4 cxd4 4. Nxd4 Nf6 5. Nc3 a6 1-0`
	g, err := chess.ParsePGN(pgn)
	if err != nil {
		t.Fatalf("ParsePGN failed: %v", err)
	}
	if _, err := db.SaveGame(g); err != nil {
		t.Fatalf("SaveGame failed: %v", err)
	}

	mgr := NewManager(filepath.Join(dir, "career.json"))
	pers, err := mgr.BuildPersonalityFromDB(db, "Kasparov")
	if err != nil {
		t.Fatalf("BuildPersonalityFromDB failed: %v", err)
	}
	if pers.Source != "database" || pers.SourceGames != 1 || pers.Elo != 2800 {
		t.Fatalf("unexpected DB personality: %+v", pers)
	}
	if len(pers.OpeningBook) == 0 {
		t.Fatalf("expected extracted opening book moves")
	}
}

func TestSelectPersonalityMoveSpyHeatmapAndCoach(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	clara, _ := FindBuiltinPersonality("t1_clara")

	// Book move
	res, err := SelectPersonalityMove("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1", clara, nil, rng)
	if err != nil || !res.FromBook || res.UCI != "e2e4" {
		t.Fatalf("expected book move e2e4, got %+v err=%v", res, err)
	}

	// Candidate selection with Multi-PV lines
	candidates := []CandidateMove{
		{UCI: "g1f3", ScoreCp: 35},
		{UCI: "d2d4", ScoreCp: 30},
		{UCI: "f1c4", ScoreCp: 28},
	}
	gm, _ := FindBuiltinPersonality("t6_grandmaster_boss")
	res2, err := SelectPersonalityMove("r1bqkbnr/pppp1ppp/2n5/4p3/4P3/8/PPPP1PPP/RNBQKBNR w KQkq - 2 3", gm, candidates, rng)
	if err != nil || res2.UCI == "" {
		t.Fatalf("SelectPersonalityMove with candidates failed: %v", err)
	}

	// Fallback heuristic without candidates
	res3, err := SelectPersonalityMove("6k1/5Q2/6K1/8/8/8/8/8 w - - 0 1", gm, nil, rng)
	if err != nil || res3.SAN == "" {
		t.Fatalf("SelectPersonalityMove fallback failed: %v", err)
	}

	// Square control heatmap + hanging piece detection
	ctrl, err := ComputeSquareControl("4k3/8/8/8/4r3/8/4N3/4K3 w - - 0 1")
	if err != nil || len(ctrl) != 64 {
		t.Fatalf("ComputeSquareControl failed: %v len=%d", err, len(ctrl))
	}
	// Undefended knight on a4 attacked by rook on a8
	ctrl2, err := ComputeSquareControl("r3k3/8/8/8/N7/8/8/4K3 w - - 0 1")
	if err != nil {
		t.Fatalf("ComputeSquareControl 2 failed: %v", err)
	}
	foundHanging := false
	for _, sq := range ctrl2 {
		if sq.Square == "a4" && sq.IsHanging {
			foundHanging = true
		}
	}
	if !foundHanging {
		t.Fatalf("expected knight on a4 to be flagged as hanging")
	}

	// Spy Mode: opponent threatens to capture undefended knight on a4
	spy, err := ComputeSpyThreat("r3k3/8/8/8/N7/8/8/4K3 w - - 0 1")
	if err != nil || !spy.HasThreat || spy.MoveUCI != "a8a4" {
		t.Fatalf("expected Spy threat a8a4, got %+v err=%v", spy, err)
	}
	// Spy Mode: immediate mate threat
	spyMate, err := ComputeSpyThreat("4k3/8/8/8/8/8/5ppp/3r2K1 w - - 0 1")
	if err != nil {
		t.Fatalf("Spy check test failed: %v", err)
	}
	if spyMate.HasThreat {
		t.Fatalf("expected no null-move threat when already in check")
	}

	// Offline Coach Report
	pgn := `[Event "Career"]
[White "Challenger"]
[Black "Toby"]
[Result "0-1"]

1. e4 e5 2. Qh5 Nc6 3. Qf3 Nf6 4. Nc3 Nd4 5. Qd1 d5 6. Nf3 Bg4 7. Nxd5 Nxd5 8. exd5 Qxd5 0-1`
	rep, err := DiagnoseGameForCoach(pgn, "white")
	if err != nil || rep.Headline == "" || len(rep.RecommendedLessons) == 0 {
		t.Fatalf("DiagnoseGameForCoach failed: %+v err=%v", rep, err)
	}

	// Additional coverage: castled game, black perspective, spy mate threat, and helpers
	pgn2 := `[Event "Solid"]
[White "A"]
[Black "B"]
[Result "1-0"]

1. e4 e5 2. Nf3 Nc6 3. Bc4 Bc5 4. O-O Nf6 5. d3 O-O 6. Nc3 d6 1-0`
	rep2, err := DiagnoseGameForCoach(pgn2, "black")
	if err != nil || len(rep2.Strengths) == 0 {
		t.Fatalf("DiagnoseGameForCoach solid game failed: %+v err=%v", rep2, err)
	}
	if _, err := ComputeSpyThreat("6k1/5ppp/8/8/8/8/5PPP/3R2K1 b - - 0 1"); err != nil {
		t.Fatalf("ComputeSpyThreat mate threat failed: %v", err)
	}
	if DefaultCareerPath() == "" {
		t.Fatalf("expected non-empty DefaultCareerPath")
	}
	for _, elo := range []int{700, 1100, 1400, 1750, 2050, 2400} {
		if eloToTier(elo) < 1 || eloToTier(elo) > 6 {
			t.Fatalf("bad tier for %d", elo)
		}
	}
	_ = deriveStyleTag(PersonalitySliders{Aggression: 85})
	_ = deriveStyleTag(PersonalitySliders{TradeTendency: 80})
	_ = deriveStyleTag(PersonalitySliders{PositionalSolidity: 85})
	_ = deriveStyleTag(PersonalitySliders{TacticalSharpness: 80})
	_ = deriveStyleTag(PersonalitySliders{})
}
