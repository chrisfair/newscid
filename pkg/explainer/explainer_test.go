package explainer

import (
	"strings"
	"testing"
)

func TestExplainMove_Castling(t *testing.T) {
	// White ready to castle kingside
	fen := "r1bqk2r/pppp1ppp/2n2n2/2b1p3/2B1P3/3P1N2/PPP2PPP/RNBQK2R w KQkq - 1 5"
	exp, err := ExplainMove(fen, "e1g1", 40, "e1g1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if exp.MoveSAN != "O-O" {
		t.Errorf("expected move SAN O-O, got %s", exp.MoveSAN)
	}

	if !contains(exp.TacticalMotifs, "Castling") {
		t.Errorf("expected Castling motif, got %v", exp.TacticalMotifs)
	}

	if !strings.Contains(exp.DetailedText, "castles") {
		t.Errorf("expected detailed text to mention castles, got %s", exp.DetailedText)
	}
}

func TestExplainMove_Checkmate(t *testing.T) {
	// Scholar's Mate: Qh5xf7#
	fen := "r1bqkb1r/pppp1ppp/2n5/4p3/2B1n3/5Q2/PPPP1PPP/RNB1K1NR w KQkq - 0 4"
	exp, err := ExplainMove(fen, "f3f7", 1000, "f3f7")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !contains(exp.TacticalMotifs, "Checkmate") {
		t.Errorf("expected Checkmate motif, got %v", exp.TacticalMotifs)
	}

	if !strings.Contains(exp.Summary, "checkmate") {
		t.Errorf("expected checkmate in summary, got %s", exp.Summary)
	}
}

func TestExplainMove_FavorableCapture(t *testing.T) {
	// White pawn takes black queen on e5
	fen := "rnb1kbnr/pppp1ppp/8/4q3/3P4/8/PPP2PPP/RNBQKBNR w KQkq - 0 1"
	exp, err := ExplainMove(fen, "d4e5", 900, "d4e5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !contains(exp.TacticalMotifs, "Favorable Exchange") {
		t.Errorf("expected Favorable Exchange motif, got %v", exp.TacticalMotifs)
	}
}
