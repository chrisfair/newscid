package chess_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"newscid/pkg/chess"
)

func TestInitialBoardAndFEN(t *testing.T) {
	b := chess.NewBoard()
	assert.Equal(t, chess.White, b.Turn)
	assert.Equal(t, chess.StartingFEN, b.FEN())

	moves := b.GenerateLegalMoves()
	// Initial position has 20 legal moves (16 pawn moves + 4 knight moves)
	assert.Equal(t, 20, len(moves))
}

func TestFENParsingRoundtrip(t *testing.T) {
	fens := []string{
		chess.StartingFEN,
		"r1bqkbnr/pppp1ppp/2n5/4p3/4P3/5N2/PPPP1PPP/RNBQKB1R w KQkq - 2 3",
		"r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1",
		"8/2p5/3p4/KP5r/1R3p1k/8/4P1P1/8 w - - 0 1",
	}

	for _, fen := range fens {
		b, err := chess.ParseFEN(fen)
		require.NoError(t, err, "Failed parsing FEN: %s", fen)
		assert.Equal(t, fen, b.FEN(), "Roundtrip FEN mismatch")
	}
}

func TestCastlingAndAttacks(t *testing.T) {
	// Position where White can castle Kingside and Queenside
	fen := "r3k2r/8/8/8/8/8/8/R3K2R w KQkq - 0 1"
	b, err := chess.ParseFEN(fen)
	require.NoError(t, err)

	moves := b.GenerateLegalMoves()
	canCastleK := false
	canCastleQ := false
	for _, m := range moves {
		if m.From == chess.E1 && m.To == chess.G1 {
			canCastleK = true
		}
		if m.From == chess.E1 && m.To == chess.C1 {
			canCastleQ = true
		}
	}
	assert.True(t, canCastleK, "Should be able to castle kingside")
	assert.True(t, canCastleQ, "Should be able to castle queenside")

	// Make kingside castle move
	san := b.MoveToSAN(chess.Move{From: chess.E1, To: chess.G1})
	assert.Equal(t, "O-O", san)
}

func TestEnPassant(t *testing.T) {
	// White pawn on e5, Black pawn plays d7-d5 setting EP square d6
	b := chess.NewBoard()
	m1, _ := b.ParseSAN("e4")
	b.MakeMove(m1)
	m2, _ := b.ParseSAN("a6")
	b.MakeMove(m2)
	m3, _ := b.ParseSAN("e5")
	b.MakeMove(m3)
	m4, _ := b.ParseSAN("d5")
	b.MakeMove(m4)

	assert.Equal(t, chess.D6, b.EPSquare)

	// White should be able to capture exd6 e.p.
	epMove, err := b.ParseSAN("exd6")
	require.NoError(t, err)
	assert.Equal(t, chess.E5, epMove.From)
	assert.Equal(t, chess.D6, epMove.To)

	b.MakeMove(epMove)
	// Black pawn on d5 should now be gone
	assert.Equal(t, chess.Empty, b.PieceAt(chess.D5))
	assert.Equal(t, chess.WP, b.PieceAt(chess.D6))
}

func TestCheckmateAndStalemate(t *testing.T) {
	// Scholar's mate
	b := chess.NewBoard()
	moves := []string{"e4", "e5", "Qh5", "Nc6", "Bc4", "Nf6", "Qxf7#"}
	for i, san := range moves {
		m, err := b.ParseSAN(san)
		require.NoError(t, err, "Move %d (%s) failed", i+1, san)
		b.MakeMove(m)
	}

	assert.True(t, b.IsCheckmate(), "Should be checkmate")
	assert.False(t, b.IsStalemate(), "Should not be stalemate")

	// Stalemate position: King on a8, White King on c7, White Queen on b6
	staleFEN := "k7/2K5/1Q6/8/8/8/8/8 b - - 0 1"
	sb, err := chess.ParseFEN(staleFEN)
	require.NoError(t, err)
	assert.True(t, sb.IsStalemate(), "Position should be stalemate")
	assert.False(t, sb.IsCheckmate(), "Position should not be checkmate")
}

func TestPGNParsingAndExport(t *testing.T) {
	pgnSample := `[Event "World Championship"]
[Site "Reykjavik"]
[Date "1972.07.11"]
[Round "1"]
[White "Spassky, Boris V"]
[Black "Fischer, Robert J"]
[Result "1-0"]

1. d4 Nf6 2. c4 e6 3. Nf3 d5 4. Nc3 Bb4 5. e3 O-O 6. Bd3 c5 7. O-O Nc6 8. a3 Ba5 1-0`

	game, err := chess.ParsePGN(pgnSample)
	require.NoError(t, err)
	assert.Equal(t, "Spassky, Boris V", game.Tag("White"))
	assert.Equal(t, "Fischer, Robert J", game.Tag("Black"))
	assert.Equal(t, chess.ResultWhiteWins, game.Result)
	assert.Equal(t, 16, game.PlyCount)

	exported := game.ExportPGN()
	assert.Contains(t, exported, "[White \"Spassky, Boris V\"]")
	assert.Contains(t, exported, "1. d4 Nf6 2. c4 e6")
	assert.Contains(t, exported, "1-0")
}

func TestBoardEdgeCasesAndCastlingUnmake(t *testing.T) {
	b := chess.NewBoard()
	b.Clear()
	assert.Equal(t, chess.NoSquare, b.KingSquare(chess.White))
	assert.False(t, b.InCheck(chess.White))
	assert.False(t, b.IsSquareAttacked(chess.NoSquare, chess.White))
	b.SetPiece(chess.NoSquare, chess.WP)
	assert.Equal(t, chess.Empty, b.PieceAt(chess.NoSquare))

	// Invalid Piece, Square, Move, Game Tag helpers
	assert.Equal(t, ".", chess.Piece(99).String())
	assert.Equal(t, chess.NoSquare, chess.SquareFromString(""))
	assert.Equal(t, "0000", chess.Move{From: chess.NoSquare, To: chess.E4}.String())
	assert.Equal(t, "e7e8q", chess.Move{From: chess.E7, To: chess.E8, Promo: chess.Queen}.String())

	g := &chess.Game{}
	assert.Equal(t, "?", g.Tag("Missing"))
	g.SetTag("ECO", "B90")
	assert.Equal(t, "B90", g.Tag("ECO"))

	// Test all 4 castling MakeMove & UnmakeMove + Black attacks
	fen := "r3k2r/ppp1pppp/8/3p4/3P4/8/PPP1PPPP/R3K2R w KQkq - 0 1"
	cb, err := chess.ParseFEN(fen)
	require.NoError(t, err)

	// White O-O
	h1 := cb.MakeMove(chess.Move{From: chess.E1, To: chess.G1})
	assert.Equal(t, chess.WR, cb.PieceAt(chess.F1))
	cb.UnmakeMove(h1)
	assert.Equal(t, chess.WR, cb.PieceAt(chess.H1))

	// White O-O-O
	h2 := cb.MakeMove(chess.Move{From: chess.E1, To: chess.C1})
	assert.Equal(t, chess.WR, cb.PieceAt(chess.D1))
	cb.UnmakeMove(h2)
	assert.Equal(t, chess.WR, cb.PieceAt(chess.A1))

	// Switch to Black turn and test Black O-O and O-O-O
	cb.Turn = chess.Black
	bMoves := cb.GenerateLegalMoves()
	hasBlackOO, hasBlackOOO := false, false
	for _, m := range bMoves {
		if m.From == chess.E8 && m.To == chess.G8 {
			hasBlackOO = true
		}
		if m.From == chess.E8 && m.To == chess.C8 {
			hasBlackOOO = true
		}
	}
	assert.True(t, hasBlackOO)
	assert.True(t, hasBlackOOO)

	h3 := cb.MakeMove(chess.Move{From: chess.E8, To: chess.G8})
	assert.Equal(t, chess.BR, cb.PieceAt(chess.F8))
	cb.UnmakeMove(h3)
	assert.Equal(t, chess.BR, cb.PieceAt(chess.H8))

	h4 := cb.MakeMove(chess.Move{From: chess.E8, To: chess.C8})
	assert.Equal(t, chess.BR, cb.PieceAt(chess.D8))
	cb.UnmakeMove(h4)
	assert.Equal(t, chess.BR, cb.PieceAt(chess.A8))

	// Black En Passant and Black Pawn Promotion
	epBoard, err := chess.ParseFEN("4k3/8/8/8/3p4/8/2P1P3/4K3 w - - 0 1")
	require.NoError(t, err)
	epBoard.MakeMove(chess.Move{From: chess.C2, To: chess.C4})
	assert.Equal(t, chess.C3, epBoard.EPSquare)
	hEP := epBoard.MakeMove(chess.Move{From: chess.D4, To: chess.C3, Capture: chess.Pawn})
	assert.Equal(t, chess.Empty, epBoard.PieceAt(chess.C4))
	epBoard.UnmakeMove(hEP)
	assert.Equal(t, chess.WP, epBoard.PieceAt(chess.C4))

	// Black pawn promotion push & capture
	promoBoard, err := chess.ParseFEN("4k3/8/8/8/8/8/1p6/R1R1K3 b Q - 0 1")
	require.NoError(t, err)
	pMoves := promoBoard.GenerateLegalMoves()
	assert.NotEmpty(t, pMoves)
	hPromo := promoBoard.MakeMove(chess.Move{From: chess.B2, To: chess.B1, Promo: chess.Queen})
	assert.Equal(t, chess.BQ, promoBoard.PieceAt(chess.B1))
	promoBoard.UnmakeMove(hPromo)
	assert.Equal(t, chess.BP, promoBoard.PieceAt(chess.B2))
}

func TestSANDisambiguationAndErrors(t *testing.T) {
	// Rank disambiguation: two rooks on same file a1 and a7 moving to a4
	bRank, err := chess.ParseFEN("4k3/R7/8/8/8/8/8/R3K3 w - - 0 1")
	require.NoError(t, err)
	sanR1 := bRank.MoveToSAN(chess.Move{From: chess.A1, To: chess.A4})
	assert.Equal(t, "R1a4", sanR1)
	parsedR1, err := bRank.ParseSAN("R1a4")
	require.NoError(t, err)
	assert.Equal(t, chess.A1, parsedR1.From)

	// File + Rank disambiguation: 3 queens on a1, a4, d1 moving to d4
	bFileRank, err := chess.ParseFEN("6k1/8/8/8/Q7/8/8/Q2QK3 w - - 0 1")
	require.NoError(t, err)
	sanQa1 := bFileRank.MoveToSAN(chess.Move{From: chess.A1, To: chess.D4})
	assert.Equal(t, "Qa1d4", sanQa1)
	parsedQa1, err := bFileRank.ParseSAN("Qa1d4")
	require.NoError(t, err)
	assert.Equal(t, chess.A1, parsedQa1.From)

	// ParseSAN promotions to R, B, N and King move
	bPromo, err := chess.ParseFEN("4k3/P7/8/8/8/8/8/4K3 w - - 0 1")
	require.NoError(t, err)
	for _, sanStr := range []string{"a8=Q", "a8=R", "a8=B", "a8=N", "Ke2"} {
		_, err := bPromo.ParseSAN(sanStr)
		assert.NoError(t, err, "Expected %s to parse", sanStr)
	}

	// ParseSAN error cases
	bInit := chess.NewBoard()
	_, err = bInit.ParseSAN("O-O")
	assert.Error(t, err)
	_, err = bInit.ParseSAN("O-O-O")
	assert.Error(t, err)
	_, err = bInit.ParseSAN("a")
	assert.Error(t, err)
	_, err = bInit.ParseSAN("Nz9")
	assert.Error(t, err)
	_, err = bInit.ParseSAN("Ke2")
	assert.Error(t, err)

	// Ambiguous SAN
	_, err = bRank.ParseSAN("Ra4")
	assert.Error(t, err)

	// ParseUCI error cases
	_, err = bInit.ParseUCI("e2")
	assert.Error(t, err)
	_, err = bInit.ParseUCI("z9e4")
	assert.Error(t, err)
	_, err = bInit.ParseUCI("e2e5")
	assert.Error(t, err)
}

func TestFENErrorsAndEdgeCases(t *testing.T) {
	_, err := chess.ParseFEN("")
	assert.Error(t, err)
	_, err = chess.ParseFEN("8/8/8/8/8/8/8")
	assert.Error(t, err)
	_, err = chess.ParseFEN("9/8/8/8/8/8/8/8 w - - 0 1")
	assert.Error(t, err)
	_, err = chess.ParseFEN("ppppppppp/8/8/8/8/8/8/8 w - - 0 1")
	assert.Error(t, err)
	_, err = chess.ParseFEN("7x/8/8/8/8/8/8/8 w - - 0 1")
	assert.Error(t, err)
	_, err = chess.ParseFEN("7/8/8/8/8/8/8/8 w - - 0 1")
	assert.Error(t, err)
	_, err = chess.ParseFEN("8/8/8/8/8/8/8/8 x - - 0 1")
	assert.Error(t, err)

	// 1-field FEN defaults to White turn
	b1, err := chess.ParseFEN("4k3/8/8/8/8/8/8/4K3")
	require.NoError(t, err)
	assert.Equal(t, chess.White, b1.Turn)

	// Descriptive piece capture disambiguation & Black PV formatting
	bDesc, err := chess.ParseFEN("6k1/8/8/8/3p4/8/8/2R1R1K1 w - - 0 1")
	require.NoError(t, err)
	// Two rooks can play to d1
	mQR, err := bDesc.ParseUCI("c1d1")
	require.NoError(t, err)
	assert.Equal(t, "QR-Q1", bDesc.MoveToDescriptive(mQR))
	mKR, err := bDesc.ParseUCI("e1d1")
	require.NoError(t, err)
	assert.Equal(t, "KR-Q1", bDesc.MoveToDescriptive(mKR))

	bDescCap, err := chess.ParseFEN("6k1/8/8/8/8/8/8/2RpR1K1 w - - 0 1")
	require.NoError(t, err)
	mCapQ, err := bDescCap.ParseUCI("c1d1")
	require.NoError(t, err)
	assert.Equal(t, "QRxP", bDescCap.MoveToDescriptive(mCapQ))
	mCapK, err := bDescCap.ParseUCI("e1d1")
	require.NoError(t, err)
	assert.Equal(t, "KRxP", bDescCap.MoveToDescriptive(mCapK))

	// FormatPVWithStyle from Black turn & fallback on invalid move
	blackFEN := "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq - 0 1"
	pvStr := chess.FormatPVWithStyle(blackFEN, []string{"e7e5", "g1f3", "invalidmove"}, chess.NotationSAN)
	assert.Equal(t, "1... e5 2. Nf3 invalidmove", pvStr)
}

func TestPGNMultiGameAndCustomFEN(t *testing.T) {
	_, err := chess.ParsePGN("   ")
	assert.Error(t, err)

	multiPGN := `[Event "Game 1"]
[Annotator "Tester"]
[Result "0-1"]

1. e4 c5 { First comment } { Second comment } (1... e5 2. Nf3) 2. Nf3 d6 0-1

[Event "Game 2"]
[FEN "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq - 0 1"]
[Result "1/2-1/2"]

1... c5 1/2-1/2`

	g1, err := chess.ParsePGN(multiPGN)
	require.NoError(t, err)
	assert.Equal(t, "First comment Second comment", g1.RootMove.Next.Comment)
	exported := g1.ExportPGN()
	assert.Contains(t, exported, "[Annotator \"Tester\"]")
	assert.Contains(t, exported, "(1... e5 2. Nf3)")
}
