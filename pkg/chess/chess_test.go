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
