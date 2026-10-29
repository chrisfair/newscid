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

func TestAnalyzeBoard_ComprehensiveImbalances(t *testing.T) {
	// Nil board
	if r := AnalyzeBoard(nil); r == nil || r.FEN != "" {
		t.Errorf("expected empty report for nil board")
	}

	// Board without kings (kingSq == -1) & doubled/isolated pawns & corner files (a/h)
	bNoKing := &chess.Board{}
	bNoKing.Squares[chess.A2] = chess.WP
	bNoKing.Squares[chess.A3] = chess.WP // doubled & isolated on a-file
	bNoKing.Squares[chess.H7] = chess.BP
	bNoKing.Squares[chess.H2] = chess.BP // passed black pawn on h-file
	rNoKing := AnalyzeBoard(bNoKing)
	if rNoKing.White.KingSafety != 50 {
		t.Errorf("expected king safety 50 when no king, got %d", rNoKing.White.KingSafety)
	}
	if rNoKing.White.PawnStructure.DoubledPawns != 1 {
		t.Errorf("expected 1 doubled pawn, got %d", rNoKing.White.PawnStructure.DoubledPawns)
	}

	// King on a1 and h8 to cover kf-1 < 0 and kf+1 > 7
	bCornerKings, _ := chess.ParseFEN("7k/8/8/8/8/8/8/K7 w - - 0 1")
	_ = AnalyzeBoard(bCornerKings)

	// All summary branches & bishop pairs
	cases := []struct {
		fen  string
		want string
	}{
		{"4k3/8/8/8/8/8/8/3QK3 w - - 0 1", "White holds a decisive material advantage."},
		{"3qk3/8/8/8/8/8/8/4K3 w - - 0 1", "Black holds a decisive material advantage."},
		{"4k3/8/8/8/8/8/4P3/4K3 w - - 0 1", "White holds a material advantage."},
		{"4k3/4p3/8/8/8/8/8/4K3 w - - 0 1", "Black holds a material advantage."},
		{"2n1kn2/8/8/8/8/8/8/2B1KB2 w - - 0 1", "White holds the bishop pair advantage on open diagonals."},
		{"2b1kb2/8/8/8/8/8/8/2N1KN2 w - - 0 1", "Black holds the bishop pair advantage on open diagonals."},
		{"4k3/p7/p3P3/8/8/8/P7/4K3 w - - 0 1", "White controls dangerous passed pawn advances."},
		{"4k3/p7/8/8/8/P3p3/P7/4K3 w - - 0 1", "Black controls dangerous passed pawn advances."},
		{"6k1/8/8/8/8/8/5PPP/6K1 w - - 0 1", ""}, // tested directly via generateSummary below
	}
	for _, tc := range cases {
		if tc.want == "" {
			continue
		}
		b, err := chess.ParseFEN(tc.fen)
		if err != nil {
			t.Fatalf("ParseFEN(%s): %v", tc.fen, err)
		}
		rep := AnalyzeBoard(b)
		if rep.PositionalSummary != tc.want {
			t.Errorf("FEN %s: got summary %q, want %q", tc.fen, rep.PositionalSummary, tc.want)
		}
	}

	// King safety summary branches (equal material)
	wSafe := SideEvaluation{KingSafety: 100}
	bUnsafe := SideEvaluation{KingSafety: 60}
	if s := generateSummary(wSafe, bUnsafe, 0); s != "White enjoys significantly greater king shelter safety." {
		t.Errorf("got %q", s)
	}
	if s := generateSummary(bUnsafe, wSafe, 0); s != "Black enjoys significantly greater king shelter safety." {
		t.Errorf("got %q", s)
	}
}
