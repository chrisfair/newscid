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
