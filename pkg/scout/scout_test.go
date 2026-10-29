package scout

import (
	"testing"
	"newscid/pkg/database/scid"
)

func TestGenerateReport(t *testing.T) {
	games := []*scid.GameSummary{
		{
			ID:     1,
			White:  "Kasparov, Garry",
			Black:  "Karpov, Anatoly",
			Result: "1-0",
			ECO:    "B90",
		},
		{
			ID:     2,
			White:  "Karpov, Anatoly",
			Black:  "Kasparov, Garry",
			Result: "0-1",
			ECO:    "E15",
		},
		{
			ID:     3,
			White:  "Kasparov, Garry",
			Black:  "Anand, Viswanathan",
			Result: "1/2-1/2",
			ECO:    "B90",
		},
	}

	report := GenerateReport("Kasparov", games)
	if report == nil {
		t.Fatal("expected report")
	}

	if report.TotalGames != 3 {
		t.Errorf("expected 3 games, got %d", report.TotalGames)
	}

	if report.WhiteRecord.Total != 2 {
		t.Errorf("expected 2 games as White, got %d", report.WhiteRecord.Total)
	}

	if report.BlackRecord.Total != 1 {
		t.Errorf("expected 1 game as Black, got %d", report.BlackRecord.Total)
	}

	if len(report.TopOpeningsAsWhite) == 0 || report.TopOpeningsAsWhite[0].ECO != "B90" {
		t.Errorf("expected B90 as top White opening, got %v", report.TopOpeningsAsWhite)
	}

	if len(report.RepertoireRecommendations) == 0 {
		t.Errorf("expected recommendations, got none")
	}
}

func TestGenerateReport_Comprehensive(t *testing.T) {
	// Empty games list
	emptyReport := GenerateReport("Carlsen", nil)
	if emptyReport.TotalGames != 0 {
		t.Errorf("expected 0 games, got %d", emptyReport.TotalGames)
	}
	if len(emptyReport.RepertoireRecommendations) != 1 {
		t.Errorf("expected insufficient games recommendation, got %v", emptyReport.RepertoireRecommendations)
	}

	// Diverse games with nil entries, non-matching players, empty ECO, losses, draws, >3 lines, and padded results
	games := []*scid.GameSummary{
		nil,
		{ID: 1, White: "Other1", Black: "Other2", Result: "1-0", ECO: "A00"},
		{ID: 2, White: "Carlsen, Magnus", Black: "PlayerA", Result: " 1-0 ", ECO: ""},
		{ID: 3, White: "Carlsen, Magnus", Black: "PlayerB", Result: " 0-1 ", ECO: "C50"},
		{ID: 4, White: "Carlsen, Magnus", Black: "PlayerC", Result: " 1/2-1/2 ", ECO: "D30"},
		{ID: 5, White: "PlayerD", Black: "Carlsen, Magnus", Result: " 1-0 ", ECO: "B12"},
		{ID: 6, White: "PlayerE", Black: "Carlsen, Magnus", Result: " 1/2-1/2 ", ECO: "E60"},
		{ID: 7, White: "PlayerF", Black: "Carlsen, Magnus", Result: "*", ECO: "A45"},
		{ID: 8, White: "Carlsen, Magnus", Black: "PlayerG", Result: "1-0", ECO: "C65"},
		{ID: 9, White: "Carlsen, Magnus", Black: "PlayerH", Result: "1-0", ECO: "C65"},
	}

	rep := GenerateReport("Carlsen", games)
	if rep.TotalGames != 8 {
		t.Errorf("expected 8 games, got %d", rep.TotalGames)
	}
	if rep.WhiteRecord.Wins != 3 || rep.WhiteRecord.Losses != 1 || rep.WhiteRecord.Draws != 1 {
		t.Errorf("unexpected white record: %+v", rep.WhiteRecord)
	}
	if rep.BlackRecord.Wins != 0 || rep.BlackRecord.Losses != 1 || rep.BlackRecord.Draws != 1 {
		t.Errorf("unexpected black record: %+v", rep.BlackRecord)
	}
	if len(rep.SharpestLines) != 3 || len(rep.WeakestLines) != 3 {
		t.Errorf("expected 3 sharpest and 3 weakest lines, got %d and %d", len(rep.SharpestLines), len(rep.WeakestLines))
	}
	if len(rep.RepertoireRecommendations) < 3 {
		t.Errorf("expected at least 3 recommendations (including black dip), got %v", rep.RepertoireRecommendations)
	}
}

