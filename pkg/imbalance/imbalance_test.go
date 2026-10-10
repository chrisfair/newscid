package imbalance

import (
	"testing"
	"newscid/pkg/chess"
)

func TestAnalyzeBoard_Initial(t *testing.T) {
	b, err := chess.ParseFEN(chess.StartingFEN)
	if err != nil {
		t.Fatalf("ParseFEN failed: %v", err)
	}

	report := AnalyzeBoard(b)
	if report == nil {
		t.Fatal("expected report")
	}

	if report.MaterialDelta != 0 {
		t.Errorf("expected 0 material delta, got %d", report.MaterialDelta)
	}

	if !report.White.HasBishopPair || !report.Black.HasBishopPair {
		t.Errorf("expected both to have bishop pair initially")
	}

	if report.White.PawnStructure.IsolatedPawns != 0 {
		t.Errorf("expected 0 isolated pawns initially, got %d", report.White.PawnStructure.IsolatedPawns)
	}
}

func TestAnalyzeBoard_PassedPawnAndMaterial(t *testing.T) {
	// White has passed pawn on e7 and queen up
	fen := "4k3/4P3/8/8/8/8/8/4K2Q w - - 0 1"
	b, err := chess.ParseFEN(fen)
	if err != nil {
		t.Fatalf("ParseFEN failed: %v", err)
	}

	report := AnalyzeBoard(b)
	if report.MaterialDelta != 10 { // Q (9) + P (1) = 10
		t.Errorf("expected material delta 10, got %d", report.MaterialDelta)
	}

	if report.White.PawnStructure.PassedPawns != 1 {
		t.Errorf("expected 1 passed pawn, got %d", report.White.PawnStructure.PassedPawns)
	}

	if report.White.PawnStructure.MaxPassedRank != 7 {
		t.Errorf("expected max passed rank 7, got %d", report.White.PawnStructure.MaxPassedRank)
	}
}

func TestAnalyzeBoard_KingSafety(t *testing.T) {
	// Exposed white king on open e-file without pawns
	fen := "4r1k1/8/8/8/8/8/8/4K3 w - - 0 1"
	b, err := chess.ParseFEN(fen)
	if err != nil {
		t.Fatalf("ParseFEN failed: %v", err)
	}

	report := AnalyzeBoard(b)
	if report.White.KingSafety >= 100 {
		t.Errorf("expected white king safety < 100 on open file with enemy rook, got %d", report.White.KingSafety)
	}
}
