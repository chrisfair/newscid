package main

import (
	"path/filepath"
	"testing"

	"newscid/pkg/academy"
	"newscid/pkg/chess"
	"newscid/pkg/database/scid"
)

func TestAcademyAppBindings(t *testing.T) {
	dir := t.TempDir()
	app := NewApp()
	app.academyMgr = academy.NewManager(filepath.Join(dir, "career.json"))

	ov := app.GetAcademyOverview()
	if len(ov.Tiers) != 6 || len(ov.Personalities) < 24 || len(ov.Lessons) < 18 {
		t.Fatalf("unexpected overview: tiers=%d pers=%d lessons=%d", len(ov.Tiers), len(ov.Personalities), len(ov.Lessons))
	}

	// Build from DB when no DB is open should error cleanly
	if _, err := app.BuildAcademyPersonalityFromDB("Fischer"); err == nil {
		t.Fatalf("expected error when no DB open")
	}
	if list := app.SearchDatabasePlayers("fisch", 10); len(list) != 0 {
		t.Fatalf("expected empty list when no DB open")
	}

	// Open temp DB with a sample game
	db := scid.NewDatabase()
	if err := db.Create(filepath.Join(dir, "sample"), "Sample DB"); err != nil {
		t.Fatalf("db.Create failed: %v", err)
	}
	defer db.Close()
	pgn := `[Event "Open"]
[White "Morphy, Paul"]
[Black "Club, Player"]
[WhiteElo "2550"]
[BlackElo "2000"]
[Result "1-0"]

1. e4 e5 2. Nf3 Nc6 3. Bc4 Bc5 4. b4 Bxb4 5. c3 Ba5 1-0`
	g, err := chess.ParsePGN(pgn)
	if err != nil {
		t.Fatalf("ParsePGN failed: %v", err)
	}
	if _, err := db.SaveGame(g); err != nil {
		t.Fatalf("SaveGame failed: %v", err)
	}
	app.db = db

	suggestions := app.SearchDatabasePlayers("morph", 10)
	if len(suggestions) != 1 || suggestions[0].Name != "Morphy, Paul" || suggestions[0].GameCount != 1 {
		t.Fatalf("unexpected player suggestions: %+v", suggestions)
	}
	allTop := app.SearchDatabasePlayers("", 0)
	if len(allTop) != 2 {
		t.Fatalf("expected 2 players in DB, got %d", len(allTop))
	}

	pers, err := app.BuildAcademyPersonalityFromDB("Morphy, Paul")
	if err != nil || pers.Source != "database" {
		t.Fatalf("BuildAcademyPersonalityFromDB failed: %+v err=%v", pers, err)
	}

	mv, err := app.GetAcademyPersonalityMove(pers.ID, "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")
	if err != nil || mv.UCI == "" {
		t.Fatalf("GetAcademyPersonalityMove failed: %+v err=%v", mv, err)
	}

	if _, err := app.RecordAcademyGameResult("t1_toby", "win"); err != nil {
		t.Fatalf("RecordAcademyGameResult failed: %v", err)
	}
	if _, err := app.CompleteAcademyLesson("t1_piece_safety"); err != nil {
		t.Fatalf("CompleteAcademyLesson failed: %v", err)
	}
	if _, err := app.TestOutAcademyTier(3); err != nil {
		t.Fatalf("TestOutAcademyTier failed: %v", err)
	}
	saved, err := app.SaveCustomPersonality(academy.Personality{Name: "Test Bot", Elo: 1400})
	if err != nil {
		t.Fatalf("SaveCustomPersonality failed: %v", err)
	}
	if err := app.DeleteCustomPersonality(saved.ID); err != nil {
		t.Fatalf("DeleteCustomPersonality failed: %v", err)
	}

	if ctrl, err := app.ComputeSquareControlHeatmap("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"); err != nil || len(ctrl) != 64 {
		t.Fatalf("ComputeSquareControlHeatmap failed: %v", err)
	}
	if _, err := app.ComputeSpyThreatOverlay("r3k3/8/8/8/N7/8/8/4K3 w - - 0 1"); err != nil {
		t.Fatalf("ComputeSpyThreatOverlay failed: %v", err)
	}
	if rep, err := app.DiagnoseGameForAcademyCoach(pgn, "white"); err != nil || rep.Headline == "" {
		t.Fatalf("DiagnoseGameForAcademyCoach failed: %v", err)
	}
}
