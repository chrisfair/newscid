package chess_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"newscid/pkg/chess"
)

func TestValidNotationStyle(t *testing.T) {
	assert.True(t, chess.ValidNotationStyle("san"))
	assert.True(t, chess.ValidNotationStyle("SAN"))
	assert.True(t, chess.ValidNotationStyle("fan"))
	assert.True(t, chess.ValidNotationStyle("lan"))
	assert.True(t, chess.ValidNotationStyle("descriptive"))
	assert.True(t, chess.ValidNotationStyle("uci"))
	assert.True(t, chess.ValidNotationStyle("iccf"))
	assert.False(t, chess.ValidNotationStyle("unknown"))
	assert.False(t, chess.ValidNotationStyle(""))
}

func TestNotationStylesForStandardMoves(t *testing.T) {
	b := chess.NewBoard()

	// 1. e4 (e2e4)
	m1, err := b.ParseUCI("e2e4")
	require.NoError(t, err)

	assert.Equal(t, "e4", b.MoveToNotation(m1, chess.NotationSAN))
	assert.Equal(t, "e4", b.MoveToNotation(m1, chess.NotationFAN))
	assert.Equal(t, "e2-e4", b.MoveToNotation(m1, chess.NotationLAN))
	assert.Equal(t, "P-K4", b.MoveToNotation(m1, chess.NotationDescriptive))
	assert.Equal(t, "e2e4", b.MoveToNotation(m1, chess.NotationUCI))
	assert.Equal(t, "5254", b.MoveToNotation(m1, chess.NotationICCF))

	b.MakeMove(m1)

	// 1... e5 (e7e5)
	m2, err := b.ParseUCI("e7e5")
	require.NoError(t, err)

	assert.Equal(t, "e5", b.MoveToNotation(m2, chess.NotationSAN))
	assert.Equal(t, "e5", b.MoveToNotation(m2, chess.NotationFAN))
	assert.Equal(t, "e7-e5", b.MoveToNotation(m2, chess.NotationLAN))
	assert.Equal(t, "P-K4", b.MoveToNotation(m2, chess.NotationDescriptive))
	assert.Equal(t, "e7e5", b.MoveToNotation(m2, chess.NotationUCI))
	assert.Equal(t, "5755", b.MoveToNotation(m2, chess.NotationICCF))

	b.MakeMove(m2)

	// 2. Nf3 (g1f3)
	m3, err := b.ParseUCI("g1f3")
	require.NoError(t, err)

	assert.Equal(t, "Nf3", b.MoveToNotation(m3, chess.NotationSAN))
	assert.Equal(t, "♘f3", b.MoveToNotation(m3, chess.NotationFAN))
	assert.Equal(t, "Ng1-f3", b.MoveToNotation(m3, chess.NotationLAN))
	assert.Equal(t, "N-KB3", b.MoveToNotation(m3, chess.NotationDescriptive))
	assert.Equal(t, "g1f3", b.MoveToNotation(m3, chess.NotationUCI))
	assert.Equal(t, "7163", b.MoveToNotation(m3, chess.NotationICCF))

	b.MakeMove(m3)

	// 2... Nc6 (b8c6)
	m4, err := b.ParseUCI("b8c6")
	require.NoError(t, err)

	assert.Equal(t, "Nc6", b.MoveToNotation(m4, chess.NotationSAN))
	assert.Equal(t, "♘c6", b.MoveToNotation(m4, chess.NotationFAN))
	assert.Equal(t, "Nb8-c6", b.MoveToNotation(m4, chess.NotationLAN))
	assert.Equal(t, "N-QB3", b.MoveToNotation(m4, chess.NotationDescriptive))
	assert.Equal(t, "b8c6", b.MoveToNotation(m4, chess.NotationUCI))
	assert.Equal(t, "2836", b.MoveToNotation(m4, chess.NotationICCF))
}

func TestNotationCastlingAndCaptures(t *testing.T) {
	// Setup position with castling and captures
	fen := "r3k2r/pbppqpb1/1pn1pn2/8/2B1PB2/2NP1N2/PPP1QPPP/R3K2R w KQkq - 0 1"
	b, err := chess.ParseFEN(fen)
	require.NoError(t, err)

	// Kingside castle O-O (e1g1)
	mCastleK, err := b.ParseUCI("e1g1")
	require.NoError(t, err)
	assert.Equal(t, "O-O", b.MoveToNotation(mCastleK, chess.NotationSAN))
	assert.Equal(t, "O-O", b.MoveToNotation(mCastleK, chess.NotationFAN))
	assert.Equal(t, "O-O", b.MoveToNotation(mCastleK, chess.NotationLAN))
	assert.Equal(t, "O-O", b.MoveToNotation(mCastleK, chess.NotationDescriptive))
	assert.Equal(t, "e1g1", b.MoveToNotation(mCastleK, chess.NotationUCI))
	assert.Equal(t, "5171", b.MoveToNotation(mCastleK, chess.NotationICCF))

	// Queenside castle O-O-O (e1c1)
	mCastleQ, err := b.ParseUCI("e1c1")
	require.NoError(t, err)
	assert.Equal(t, "O-O-O", b.MoveToNotation(mCastleQ, chess.NotationSAN))
	assert.Equal(t, "O-O-O", b.MoveToNotation(mCastleQ, chess.NotationFAN))
	assert.Equal(t, "O-O-O", b.MoveToNotation(mCastleQ, chess.NotationLAN))
	assert.Equal(t, "O-O-O", b.MoveToNotation(mCastleQ, chess.NotationDescriptive))
	assert.Equal(t, "e1c1", b.MoveToNotation(mCastleQ, chess.NotationUCI))
	assert.Equal(t, "5131", b.MoveToNotation(mCastleQ, chess.NotationICCF))

	// Capture: Bxc7 (f4c7)
	mCap, err := b.ParseUCI("f4c7")
	require.NoError(t, err)
	assert.Equal(t, "Bxc7", b.MoveToNotation(mCap, chess.NotationSAN))
	assert.Equal(t, "♗xc7", b.MoveToNotation(mCap, chess.NotationFAN))
	assert.Equal(t, "Bf4xc7", b.MoveToNotation(mCap, chess.NotationLAN))
	assert.Equal(t, "BxP", b.MoveToNotation(mCap, chess.NotationDescriptive))
	assert.Equal(t, "f4c7", b.MoveToNotation(mCap, chess.NotationUCI))
	assert.Equal(t, "6437", b.MoveToNotation(mCap, chess.NotationICCF))
}

func TestNotationPromotionAndCheck(t *testing.T) {
	// White pawn on a7 about to promote with check: a7a8q, King on f8
	fen := "5k2/P7/8/8/8/8/8/4K3 w - - 0 1"
	b, err := chess.ParseFEN(fen)
	require.NoError(t, err)

	mPromoQ, err := b.ParseUCI("a7a8q")
	require.NoError(t, err)

	assert.Equal(t, "a8=Q+", b.MoveToNotation(mPromoQ, chess.NotationSAN))
	assert.Equal(t, "a8=♕+", b.MoveToNotation(mPromoQ, chess.NotationFAN))
	assert.Equal(t, "a7-a8=Q+", b.MoveToNotation(mPromoQ, chess.NotationLAN))
	assert.Equal(t, "P-QR8(Q)+", b.MoveToNotation(mPromoQ, chess.NotationDescriptive))
	assert.Equal(t, "a7a8q", b.MoveToNotation(mPromoQ, chess.NotationUCI))
	assert.Equal(t, "17181", b.MoveToNotation(mPromoQ, chess.NotationICCF))

	// Underpromotions for ICCF:
	mPromoR, _ := b.ParseUCI("a7a8r")
	assert.Equal(t, "17182", b.MoveToNotation(mPromoR, chess.NotationICCF))
	mPromoB, _ := b.ParseUCI("a7a8b")
	assert.Equal(t, "17183", b.MoveToNotation(mPromoB, chess.NotationICCF))
	mPromoN, _ := b.ParseUCI("a7a8n")
	assert.Equal(t, "17184", b.MoveToNotation(mPromoN, chess.NotationICCF))
}

func TestFormatMoveWithFEN(t *testing.T) {
	str, err := chess.FormatMoveWithFEN(chess.StartingFEN, "e2e4", chess.NotationFAN)
	require.NoError(t, err)
	assert.Equal(t, "e4", str)

	strKnight, err := chess.FormatMoveWithFEN(chess.StartingFEN, "g1f3", chess.NotationFAN)
	require.NoError(t, err)
	assert.Equal(t, "♘f3", strKnight)

	// Error handling
	_, err = chess.FormatMoveWithFEN("invalid fen", "e2e4", chess.NotationSAN)
	assert.Error(t, err)

	_, err = chess.FormatMoveWithFEN(chess.StartingFEN, "e2e5", chess.NotationSAN)
	assert.Error(t, err)
}

func TestFormatPVWithStyle(t *testing.T) {
	pv := []string{"e2e4", "e7e5", "g1f3", "b8c6"}

	sanPV := chess.FormatPVWithStyle(chess.StartingFEN, pv, chess.NotationSAN)
	assert.Equal(t, "1. e4 e5 2. Nf3 Nc6", sanPV)

	fanPV := chess.FormatPVWithStyle(chess.StartingFEN, pv, chess.NotationFAN)
	assert.Equal(t, "1. e4 e5 2. ♘f3 ♘c6", fanPV)

	lanPV := chess.FormatPVWithStyle(chess.StartingFEN, pv, chess.NotationLAN)
	assert.Equal(t, "1. e2-e4 e7-e5 2. Ng1-f3 Nb8-c6", lanPV)

	descPV := chess.FormatPVWithStyle(chess.StartingFEN, pv, chess.NotationDescriptive)
	assert.Equal(t, "1. P-K4 P-K4 2. N-KB3 N-QB3", descPV)

	uciPV := chess.FormatPVWithStyle(chess.StartingFEN, pv, chess.NotationUCI)
	assert.Equal(t, "1. e2e4 e7e5 2. g1f3 b8c6", uciPV)

	iccfPV := chess.FormatPVWithStyle(chess.StartingFEN, pv, chess.NotationICCF)
	assert.Equal(t, "1. 5254 5755 2. 7163 2836", iccfPV)

	// Empty PV
	assert.Empty(t, chess.FormatPVWithStyle(chess.StartingFEN, nil, chess.NotationSAN))

	// Invalid FEN fallback
	assert.Equal(t, "e2e4 e7e5", chess.FormatPVWithStyle("invalid fen", []string{"e2e4", "e7e5"}, chess.NotationSAN))
}

func TestSANToFAN(t *testing.T) {
	assert.Equal(t, "", chess.SANToFAN(""))
	assert.Equal(t, "O-O", chess.SANToFAN("O-O"))
	assert.Equal(t, "O-O-O", chess.SANToFAN("O-O-O"))
	assert.Equal(t, "♔e2", chess.SANToFAN("Ke2"))
	assert.Equal(t, "♕h5+", chess.SANToFAN("Qh5+"))
	assert.Equal(t, "♖e1", chess.SANToFAN("Re1"))
	assert.Equal(t, "♗c4", chess.SANToFAN("Bc4"))
	assert.Equal(t, "♘f3", chess.SANToFAN("Nf3"))
	assert.Equal(t, "e8=♕", chess.SANToFAN("e8=Q"))
	assert.Equal(t, "e8=♖", chess.SANToFAN("e8=R"))
	assert.Equal(t, "e8=♗", chess.SANToFAN("e8=B"))
	assert.Equal(t, "e8=♘", chess.SANToFAN("e8=N"))
}

func TestDescriptiveDisambiguationAndEnPassant(t *testing.T) {
	// Position with two knights that can move to d2: Nb1 and Nf3 (d4 pawn, d2 empty)
	b1, err := chess.ParseFEN("r1bqkbnr/pppppppp/8/8/3P4/5N2/PPP1PPPP/RNBQKB1R w KQkq - 0 1")
	require.NoError(t, err)

	// Nb1 to d2 (QN-Q2)
	mQN, err := b1.ParseUCI("b1d2")
	require.NoError(t, err)
	assert.Equal(t, "QN-Q2", b1.MoveToNotation(mQN, chess.NotationDescriptive))

	// Nf3 to d2 (KN-Q2)
	mKN, err := b1.ParseUCI("f3d2")
	require.NoError(t, err)
	assert.Equal(t, "KN-Q2", b1.MoveToNotation(mKN, chess.NotationDescriptive))

	// Position with en passant capture in descriptive
	// White pawn on e5, Black pawn on d5 (just played d7d5 with EP d6)
	b2 := chess.NewBoard()
	mE4, _ := b2.ParseUCI("e2e4")
	b2.MakeMove(mE4)
	mA6, _ := b2.ParseUCI("a7a6")
	b2.MakeMove(mA6)
	mE5, _ := b2.ParseUCI("e4e5")
	b2.MakeMove(mE5)
	mD5, _ := b2.ParseUCI("d7d5")
	b2.MakeMove(mD5)

	mEP, err := b2.ParseUCI("e5d6")
	require.NoError(t, err)
	assert.Equal(t, "PxP e.p.", b2.MoveToNotation(mEP, chess.NotationDescriptive))

	// Multiple pawns capturing on d5: white pawns on c4 and e4 capturing d5
	b3, err := chess.ParseFEN("rnbqkbnr/ppp1pppp/8/3p4/2P1P3/8/PP1P1PPP/RNBQKBNR w KQkq - 0 1")
	require.NoError(t, err)

	mCPxP, err := b3.ParseUCI("c4d5")
	require.NoError(t, err)
	assert.Equal(t, "QPxP", b3.MoveToNotation(mCPxP, chess.NotationDescriptive))
}

func TestFormatNAGAndTypes(t *testing.T) {
	// NAG formatting
	assert.Equal(t, "!", chess.FormatNAG(1))
	assert.Equal(t, "?", chess.FormatNAG(2))
	assert.Equal(t, "!!", chess.FormatNAG(3))
	assert.Equal(t, "??", chess.FormatNAG(4))
	assert.Equal(t, "!?", chess.FormatNAG(5))
	assert.Equal(t, "?!", chess.FormatNAG(6))
	assert.Equal(t, "=", chess.FormatNAG(10))
	assert.Equal(t, "+=", chess.FormatNAG(14))
	assert.Equal(t, "=+", chess.FormatNAG(15))
	assert.Equal(t, "+/-", chess.FormatNAG(16))
	assert.Equal(t, "-/+", chess.FormatNAG(17))
	assert.Equal(t, "+-", chess.FormatNAG(18))
	assert.Equal(t, "-+", chess.FormatNAG(19))
	assert.Equal(t, "$99", chess.FormatNAG(99))

	// Color string
	assert.Equal(t, "White", chess.White.String())
	assert.Equal(t, "Black", chess.Black.String())
	assert.Equal(t, "NoColor", chess.NoColor.String())

	// PieceType string
	assert.Equal(t, "K", chess.King.String())
	assert.Equal(t, "Q", chess.Queen.String())
	assert.Equal(t, "R", chess.Rook.String())
	assert.Equal(t, "B", chess.Bishop.String())
	assert.Equal(t, "N", chess.Knight.String())
	assert.Equal(t, "P", chess.Pawn.String())
	assert.Equal(t, "", chess.None.String())

	// Piece helpers
	wp := chess.MakePiece(chess.White, chess.Pawn)
	assert.Equal(t, chess.WP, wp)
	assert.Equal(t, "P", wp.String())
	assert.Equal(t, chess.White, wp.Color())
	emptyP := chess.MakePiece(chess.White, chess.None)
	assert.Equal(t, chess.Empty, emptyP)
	assert.Equal(t, chess.NoColor, emptyP.Color())

	// Square helpers
	invSq := chess.MakeSquare(-1, 0)
	assert.Equal(t, chess.NoSquare, invSq)
	assert.Equal(t, "-", invSq.String())

	// Move equality
	m1 := chess.Move{From: chess.E2, To: chess.E4}
	m2 := chess.Move{From: chess.E2, To: chess.E4}
	m3 := chess.Move{From: chess.E2, To: chess.E3}
	assert.True(t, m1.Equals(m2))
	assert.False(t, m1.Equals(m3))
}

func TestPGNWithCommentsNAGsAndVariations(t *testing.T) {
	pgnSample := `[Event "Test Tournament"]
[Site "Paris"]
[Date "2024.01.01"]
[Round "1"]
[White "Player A"]
[Black "Player B"]
[Result "1-0"]

1. e4 {Best by test} 1... e5 $1 2. Nf3 (2. f4 $5 {King's Gambit} 2... exf4) 2... Nc6 1-0`

	game, err := chess.ParsePGN(pgnSample)
	require.NoError(t, err)
	assert.Equal(t, "Player A", game.Tag("White"))
	assert.Equal(t, "Player B", game.Tag("Black"))

	exported := game.ExportPGN()
	assert.Contains(t, exported, "{Best by test}")
	assert.Contains(t, exported, "!")
	assert.Contains(t, exported, "2. f4")
}

func TestPGNWithClockAndEMT(t *testing.T) {
	pgnSample := `[Event "Rapid Match"]
[White "Carlsen"]
[Black "Nakamura"]
[Result "1-0"]

1. e4 {[%clk 0:10:00]} 1... c5 {Sicilian Defense [%clk 0:10:00] [%emt 0:00:03]} 2. Nf3 {[%clk 0:09:58]} 2... d6 {[%clk 0:09:59]} 1-0`

	game, err := chess.ParsePGN(pgnSample)
	require.NoError(t, err)

	// Move 1: e4 has clock 0:10:00 and NO comment box text
	e4 := game.RootMove
	assert.Equal(t, "e4", e4.SAN)
	assert.Equal(t, "0:10:00", e4.Clock)
	assert.Empty(t, e4.Comment)

	// Move 1... c5: has text comment "Sicilian Defense", clock 0:10:00, emt 0:00:03
	c5 := e4.Next
	assert.Equal(t, "c5", c5.SAN)
	assert.Equal(t, "Sicilian Defense", c5.Comment)
	assert.Equal(t, "0:10:00", c5.Clock)
	assert.Equal(t, "0:00:03", c5.EMT)

	exported := game.ExportPGN()
	assert.Contains(t, exported, "[%clk 0:10:00]")
	assert.Contains(t, exported, "Sicilian Defense")
}


