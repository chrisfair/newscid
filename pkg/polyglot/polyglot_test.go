package polyglot

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"newscid/pkg/chess"
)

func TestPolyglotZobristHash(t *testing.T) {
	// Starting board
	b := chess.NewBoard()
	hash0 := HashBoard(b)
	assert.Equal(t, uint64(0x463b96181691fc9c), hash0)

	// 1. e4
	mE4, err := b.ParseSAN("e4")
	require.NoError(t, err)
	b.MakeMove(mE4)
	assert.Equal(t, uint64(0x823c9b50fd114196), HashBoard(b))

	// 1... e5
	mE5, err := b.ParseSAN("e5")
	require.NoError(t, err)
	b.MakeMove(mE5)
	assert.Equal(t, uint64(0x0844931a6ef4b9a0), HashBoard(b))

	// 2. Nf3
	mNf3, err := b.ParseSAN("Nf3")
	require.NoError(t, err)
	b.MakeMove(mNf3)
	assert.Equal(t, uint64(0xd3207fec0612d89d), HashBoard(b))

	// 2... Nc6
	mNc6, err := b.ParseSAN("Nc6")
	require.NoError(t, err)
	b.MakeMove(mNc6)
	assert.Equal(t, uint64(0x78cda70e17837d9e), HashBoard(b))

	// 3. Bb5
	mBb5, err := b.ParseSAN("Bb5")
	require.NoError(t, err)
	b.MakeMove(mBb5)
	assert.Equal(t, uint64(0x4b1376a17217ee1d), HashBoard(b))
}

func TestSynthesizedPolyglotBook(t *testing.T) {
	tmpDir := t.TempDir()
	bookPath := filepath.Join(tmpDir, "test_book.bin")

	b := chess.NewBoard()
	startKey := HashBoard(b)

	// Create binary book entries for starting position:
	// Entry 1: e2e4 (from e2=12, to e4=28, fromFile=4, fromRank=1, toFile=4, toRank=3)
	// rawMove = (toFile=4) | (toRank=3 << 3) | (fromFile=4 << 6) | (fromRank=1 << 9)
	// = 4 | 24 | 256 | 512 = 796 = 0x031c
	rawE4 := uint16(4 | (3 << 3) | (4 << 6) | (1 << 9))
	rawD4 := uint16(3 | (3 << 3) | (3 << 6) | (1 << 9))

	var fileData []byte

	// Write entries in sorted order by key
	writeEntry := func(key uint64, rawMove uint16, weight uint16, learn uint32) {
		buf := make([]byte, 16)
		binary.BigEndian.PutUint64(buf[0:8], key)
		binary.BigEndian.PutUint16(buf[8:10], rawMove)
		binary.BigEndian.PutUint16(buf[10:12], weight)
		binary.BigEndian.PutUint32(buf[12:16], learn)
		fileData = append(fileData, buf...)
	}

	writeEntry(startKey, rawE4, 80, 100)
	writeEntry(startKey, rawD4, 20, 50)

	require.NoError(t, os.WriteFile(bookPath, fileData, 0644))

	book, err := OpenBook(bookPath)
	require.NoError(t, err)
	assert.Equal(t, 2, book.entries)

	moves, err := book.FindMoves(b)
	require.NoError(t, err)
	require.Len(t, moves, 2)

	assert.Equal(t, "e2e4", moves[0].UCI)
	assert.Equal(t, "e4", moves[0].SAN)
	assert.Equal(t, 80, moves[0].Weight)
	assert.Equal(t, 80.0, moves[0].Percentage)

	assert.Equal(t, "d2d4", moves[1].UCI)
	assert.Equal(t, "d4", moves[1].SAN)
	assert.Equal(t, 20, moves[1].Weight)
	assert.Equal(t, 20.0, moves[1].Percentage)

	// Test PickMove best
	bestMove, err := book.PickMove(b, true)
	require.NoError(t, err)
	require.NotNil(t, bestMove)
	assert.Equal(t, "e2e4", bestMove.UCI)

	// Test PickMove weighted
	weightedMove, err := book.PickMove(b, false)
	require.NoError(t, err)
	require.NotNil(t, weightedMove)
	assert.True(t, weightedMove.UCI == "e2e4" || weightedMove.UCI == "d2d4")

	// Position not in book
	mE4, _ := b.ParseSAN("e4")
	b.MakeMove(mE4)
	movesE4, err := book.FindMoves(b)
	require.NoError(t, err)
	assert.Empty(t, movesE4)
}

func TestRealSystemPolyglotBooks(t *testing.T) {
	books := DiscoverBooks()
	t.Logf("Discovered %d polyglot books: %v", len(books), books)

	eloPath := "/usr/share/scid/books/Elo2400.bin"
	if _, err := os.Stat(eloPath); err == nil {
		book, err := OpenBook(eloPath)
		require.NoError(t, err)
		assert.Greater(t, book.entries, 1000)

		b := chess.NewBoard()
		moves, err := book.FindMoves(b)
		require.NoError(t, err)
		assert.NotEmpty(t, moves)

		// Verify e4 and d4 are prominent in Elo2400
		hasE4 := false
		hasD4 := false
		for _, m := range moves {
			if m.SAN == "e4" {
				hasE4 = true
			}
			if m.SAN == "d4" {
				hasD4 = true
			}
		}
		assert.True(t, hasE4, "Elo2400 should have e4")
		assert.True(t, hasD4, "Elo2400 should have d4")

		pick, err := book.PickMove(b, true)
		require.NoError(t, err)
		require.NotNil(t, pick)
		assert.NotEmpty(t, pick.SAN)
	}
}

func TestCastlingAndPromotionDecoding(t *testing.T) {
	b := chess.NewBoard()

	// Decode White Kingside Castling e1h1 -> e1g1
	rawCastleK := uint16(7 | (0 << 3) | (4 << 6) | (0 << 9))
	uciCastleK := decodeRawMove(b, rawCastleK)
	assert.Equal(t, "e1g1", uciCastleK)

	// Decode White Queenside Castling e1a1 -> e1c1
	rawCastleQ := uint16(0 | (0 << 3) | (4 << 6) | (0 << 9))
	uciCastleQ := decodeRawMove(b, rawCastleQ)
	assert.Equal(t, "e1c1", uciCastleQ)

	// Black Kingside Castling e8h8 -> e8g8
	b.Turn = chess.Black
	rawCastleBK := uint16(7 | (7 << 3) | (4 << 6) | (7 << 9))
	uciCastleBK := decodeRawMove(b, rawCastleBK)
	assert.Equal(t, "e8g8", uciCastleBK)

	// Black Queenside Castling e8a8 -> e8c8
	rawCastleBQ := uint16(0 | (7 << 3) | (4 << 6) | (7 << 9))
	uciCastleBQ := decodeRawMove(b, rawCastleBQ)
	assert.Equal(t, "e8c8", uciCastleBQ)

	// Promotion moves: knight, bishop, rook, queen
	// e7e8n
	rawPromoN := uint16(4 | (7 << 3) | (4 << 6) | (6 << 9) | (1 << 12))
	assert.Equal(t, "e7e8n", decodeRawMove(b, rawPromoN))

	// e7e8b
	rawPromoB := uint16(4 | (7 << 3) | (4 << 6) | (6 << 9) | (2 << 12))
	assert.Equal(t, "e7e8b", decodeRawMove(b, rawPromoB))

	// e7e8r
	rawPromoR := uint16(4 | (7 << 3) | (4 << 6) | (6 << 9) | (3 << 12))
	assert.Equal(t, "e7e8r", decodeRawMove(b, rawPromoR))

	// e7e8q (promo=4)
	rawPromoQ := uint16(4 | (7 << 3) | (4 << 6) | (6 << 9) | (4 << 12))
	assert.Equal(t, "e7e8q", decodeRawMove(b, rawPromoQ))
}

func TestPolyglotErrorsAndEdgeCases(t *testing.T) {
	// 1. Non-existent file
	_, err := OpenBook("/non/existent/file.bin")
	assert.Error(t, err)

	// 2. Invalid size
	tmpDir := t.TempDir()
	badSizeFile := filepath.Join(tmpDir, "bad_size.bin")
	require.NoError(t, os.WriteFile(badSizeFile, []byte{1, 2, 3}, 0644))
	_, err = OpenBook(badSizeFile)
	assert.Error(t, err)

	// 3. Empty book
	emptyFile := filepath.Join(tmpDir, "empty.bin")
	require.NoError(t, os.WriteFile(emptyFile, []byte{}, 0644))
	emptyBook, err := OpenBook(emptyFile)
	require.NoError(t, err)
	b := chess.NewBoard()
	moves, err := emptyBook.FindMoves(b)
	require.NoError(t, err)
	assert.Empty(t, moves)
	pick, err := emptyBook.PickMove(b, false)
	require.NoError(t, err)
	assert.Nil(t, pick)

	// 4. En passant on White turn (Black played f5, White pawn on e5 can capture)
	bEPWhite, err := chess.ParseFEN("rnbqkbnr/ppp1p1pp/8/3pPp2/8/8/PPPP1PPP/RNBQKBNR w KQkq f6 0 3")
	require.NoError(t, err)
	hashEPWhite := HashBoard(bEPWhite)
	assert.NotZero(t, hashEPWhite)

	// 5. En passant on Black turn (White played d4, Black pawn on e4 can capture)
	bEPBlack, err := chess.ParseFEN("rnbqkbnr/ppp1pppp/8/8/3pP3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 2")
	require.NoError(t, err)
	hashEPBlack := HashBoard(bEPBlack)
	assert.NotZero(t, hashEPBlack)

	// 6. DiscoverBooks with extra custom dir
	discovered := DiscoverBooks(tmpDir)
	assert.NotEmpty(t, discovered)

	// 7. polyglotPieceIndex for invalid piece type
	assert.Equal(t, -1, polyglotPieceIndex(chess.None, chess.White))
}
