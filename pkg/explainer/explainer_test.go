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

func TestExplainMove_ForkPromotionAndOpenFiles(t *testing.T) {
	// 1. Pawn Promotion
	fenPromo := "8/4P1k1/8/8/8/8/8/4K3 w - - 0 1"
	expPromo, err := ExplainMove(fenPromo, "e7e8q", 900, "e7e8q")
	if err != nil {
		t.Fatalf("promotion error: %v", err)
	}
	if !contains(expPromo.TacticalMotifs, "Pawn Promotion") {
		t.Errorf("expected Pawn Promotion motif, got %v", expPromo.TacticalMotifs)
	}

	// 2. Knight on Outpost & Central Control
	fenOutpost := "r1bqk2r/pp1p1ppp/2n1pn2/8/1bPN4/2N5/PP2PPPP/R1BQKB1R w KQkq - 1 6"
	expOutpost, err := ExplainMove(fenOutpost, "d4b5", 50, "d4b5")
	if err != nil {
		t.Fatalf("outpost error: %v", err)
	}
	if expOutpost.Assessment == "" {
		t.Errorf("expected assessment")
	}

	// 3. Rook on open file
	fenRook := "4k3/8/8/8/8/8/8/R3K3 w - - 0 1"
	expRook, err := ExplainMove(fenRook, "a1d1", 100, "a1d1")
	if err != nil {
		t.Fatalf("rook open file error: %v", err)
	}
	if !contains(expRook.StrategicConcepts, "Open File Control") {
		t.Errorf("expected Open File Control, got %v", expRook.StrategicConcepts)
	}

	// 4. Inaccuracy / Mistake detection
	fenMistake := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
	expMistake, err := ExplainMove(fenMistake, "h2h4", -200, "e2e4")
	if err != nil {
		t.Fatalf("mistake error: %v", err)
	}
	if expMistake.Assessment != "Inaccuracy / Mistake" {
		t.Errorf("expected Inaccuracy / Mistake, got %s", expMistake.Assessment)
	}

	// 5. Black queenside castle
	fenBlackCastle := "r3k2r/pppppppp/8/8/8/8/PPPPPPPP/R3K2R b KQkq - 0 1"
	expBlackCastle, err := ExplainMove(fenBlackCastle, "e8c8", 0, "e8c8")
	if err != nil {
		t.Fatalf("black castle error: %v", err)
	}
	if !contains(expBlackCastle.TacticalMotifs, "Castling") {
		t.Errorf("expected Castling motif for black queenside")
	}

	// 6. Error handling
	_, err = ExplainMove("invalid-fen", "e2e4", 0, "")
	if err == nil {
		t.Errorf("expected error for invalid FEN")
	}
	_, err = ExplainMove("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1", "invalid-move", 0, "")
	if err == nil {
		t.Errorf("expected error for invalid move")
	}
}

