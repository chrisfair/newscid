package explainer

import (
	"strings"
	"testing"

	"newscid/pkg/chess"
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
	// Pawn Promotion
	fenPromo := "8/4P1k1/8/8/8/8/8/4K3 w - - 0 1"
	expPromo, err := ExplainMove(fenPromo, "e7e8q", 900, "e7e8q")
	if err != nil {
		t.Fatalf("promotion error: %v", err)
	}
	if !contains(expPromo.TacticalMotifs, "Pawn Promotion") {
		t.Errorf("expected Pawn Promotion motif, got %v", expPromo.TacticalMotifs)
	}

	// Knight on Outpost & Central Control
	fenOutpost := "r1bqk2r/pp1p1ppp/2n1pn2/8/1bPN4/2N5/PP2PPPP/R1BQKB1R w KQkq - 1 6"
	expOutpost, err := ExplainMove(fenOutpost, "d4b5", 50, "d4b5")
	if err != nil {
		t.Fatalf("outpost error: %v", err)
	}
	if expOutpost.Assessment == "" {
		t.Errorf("expected assessment")
	}

	// Rook on open file
	fenRook := "4k3/8/8/8/8/8/8/R3K3 w - - 0 1"
	expRook, err := ExplainMove(fenRook, "a1d1", 100, "a1d1")
	if err != nil {
		t.Fatalf("rook open file error: %v", err)
	}
	if !contains(expRook.StrategicConcepts, "Open File Control") {
		t.Errorf("expected Open File Control, got %v", expRook.StrategicConcepts)
	}

	// Inaccuracy / Mistake detection
	fenMistake := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
	expMistake, err := ExplainMove(fenMistake, "h2h4", -200, "e2e4")
	if err != nil {
		t.Fatalf("mistake error: %v", err)
	}
	if expMistake.Assessment != "Inaccuracy / Mistake" {
		t.Errorf("expected Inaccuracy / Mistake, got %s", expMistake.Assessment)
	}

	// Black queenside castle
	fenBlackCastle := "r3k2r/pppppppp/8/8/8/8/PPPPPPPP/R3K2R b KQkq - 0 1"
	expBlackCastle, err := ExplainMove(fenBlackCastle, "e8c8", 0, "e8c8")
	if err != nil {
		t.Fatalf("black castle error: %v", err)
	}
	if !contains(expBlackCastle.TacticalMotifs, "Castling") {
		t.Errorf("expected Castling motif for black queenside")
	}

	// Error handling
	_, err = ExplainMove("invalid-fen", "e2e4", 0, "")
	if err == nil {
		t.Errorf("expected error for invalid FEN")
	}
	_, err = ExplainMove("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1", "invalid-move", 0, "")
	if err == nil {
		t.Errorf("expected error for invalid move")
	}

	// Equal trade, lower-value capture, Check, Passed Pawn, Central Outpost, Closed File Rook
	fenCheckTrade := "4k3/8/8/3n4/8/5N2/8/4K2R w - - 0 1"
	expEqual, err := ExplainMove(fenCheckTrade, "f3d4", 0, "f3d4") // Knight to central outpost d4
	if err != nil || !strings.Contains(expEqual.DetailedText, "central outpost") {
		t.Errorf("expected central outpost explanation, got %v / %v", expEqual, err)
	}

	// Equal capture: Nxd5 where d5 is knight from c3
	fenEqualCap := "4k3/8/8/3n4/8/2N5/8/4K3 w - - 0 1"
	expEqCap, err := ExplainMove(fenEqualCap, "c3d5", 0, "c3d5")
	if err != nil || !strings.Contains(expEqCap.DetailedText, "trades off") {
		t.Errorf("expected trades off, got %v", expEqCap)
	}

	// Higher value piece captures lower value piece + Check
	fenHighCap := "4k3/4p3/8/8/8/8/8/4Q2K w - - 0 1"
	expHighCap, err := ExplainMove(fenHighCap, "e1e7", 100, "e1e7")
	if err != nil || !contains(expHighCap.TacticalMotifs, "Check") || !strings.Contains(expHighCap.DetailedText, "eliminates") {
		t.Errorf("expected Check and eliminates, got %v", expHighCap)
	}

	// Passed pawn advance to rank >= 5 (e.g. e5->e6, index 5)
	fenPassed := "4k3/8/8/4P3/8/8/8/4K3 w - - 0 1"
	expPassed, err := ExplainMove(fenPassed, "e5e6", 150, "e5e6")
	if err != nil || !contains(expPassed.TacticalMotifs, "Passed Pawn Advance") {
		t.Errorf("expected Passed Pawn Advance, got %v", expPassed)
	}

	// Rook on closed file (pawn present)
	fenClosedRook := "4k3/3p4/8/8/8/8/8/R3K3 w - - 0 1"
	expClosedRook, err := ExplainMove(fenClosedRook, "a1d1", 0, "a1d1")
	if err != nil || contains(expClosedRook.StrategicConcepts, "Open File Control") {
		t.Errorf("did not expect Open File Control on file with pawn")
	}

	// Out-of-bounds pieceType helper fallback
	if pieceValue(chess.PieceType(99)) != 0 {
		t.Errorf("expected 0 for unknown piece type")
	}
	if pieceName(chess.PieceType(99)) != "piece" {
		t.Errorf("expected 'piece' for unknown piece type")
	}
}

func contains(slice []string, val string) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}

