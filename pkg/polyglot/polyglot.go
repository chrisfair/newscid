package polyglot

import (
	"encoding/binary"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"newscid/pkg/chess"
)

// HashBoard calculates the standard 64-bit Polyglot Zobrist hash of a chess board.
func HashBoard(b *chess.Board) uint64 {
	var hash uint64

	// 1. Pieces on squares
	for sq := 0; sq < 64; sq++ {
		p := b.Squares[sq]
		if p == chess.Empty {
			continue
		}
		pt := p.Type()
		col := p.Color()
		pIdx := polyglotPieceIndex(pt, col)
		if pIdx >= 0 {
			hash ^= PolyglotRandomArray[64*pIdx+sq]
		}
	}

	// 2. Castling flags
	if (b.Castling & chess.CastlingWhiteKing) != 0 {
		hash ^= PolyglotRandomArray[768]
	}
	if (b.Castling & chess.CastlingWhiteQueen) != 0 {
		hash ^= PolyglotRandomArray[769]
	}
	if (b.Castling & chess.CastlingBlackKing) != 0 {
		hash ^= PolyglotRandomArray[770]
	}
	if (b.Castling & chess.CastlingBlackQueen) != 0 {
		hash ^= PolyglotRandomArray[771]
	}

	// 3. En passant square (only if an opposing pawn can capture it)
	if b.EPSquare != chess.NoSquare {
		file := int(b.EPSquare % 8)
		canCapture := false
		if b.Turn == chess.White {
			// White to move, EPSquare on rank 6 (index 5), White pawns on rank 5 (index 4)
			if file > 0 && b.Squares[4*8+file-1] == chess.WP {
				canCapture = true
			}
			if file < 7 && b.Squares[4*8+file+1] == chess.WP {
				canCapture = true
			}
		} else {
			// Black to move, EPSquare on rank 3 (index 2), Black pawns on rank 4 (index 3)
			if file > 0 && b.Squares[3*8+file-1] == chess.BP {
				canCapture = true
			}
			if file < 7 && b.Squares[3*8+file+1] == chess.BP {
				canCapture = true
			}
		}
		if canCapture {
			hash ^= PolyglotRandomArray[772+file]
		}
	}

	// 4. Turn (White to move)
	if b.Turn == chess.White {
		hash ^= PolyglotRandomArray[780]
	}

	return hash
}

func polyglotPieceIndex(pt chess.PieceType, color chess.Color) int {
	var base int
	switch pt {
	case chess.Pawn:
		base = 0
	case chess.Knight:
		base = 2
	case chess.Bishop:
		base = 4
	case chess.Rook:
		base = 6
	case chess.Queen:
		base = 8
	case chess.King:
		base = 10
	default:
		return -1
	}
	if color == chess.White {
		base++
	}
	return base
}

// BookEntry represents a raw 16-byte entry in a Polyglot .bin book.
type BookEntry struct {
	Key     uint64
	RawMove uint16
	Weight  uint16
	Learn   uint32
}

// BookMove represents a decoded move from a Polyglot book.
type BookMove struct {
	UCI        string  `json:"uci"`
	SAN        string  `json:"san"`
	Weight     int     `json:"weight"`
	Percentage float64 `json:"percentage"`
	Learn      uint32  `json:"learn"`
}

// Book represents an opened Polyglot opening book file (.bin).
type Book struct {
	Path    string
	data    []byte
	entries int
}

// OpenBook loads a Polyglot .bin file into memory.
func OpenBook(path string) (*Book, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading polyglot book %s: %w", path, err)
	}

	if len(data)%16 != 0 {
		return nil, fmt.Errorf("invalid polyglot book size %d (must be multiple of 16)", len(data))
	}

	return &Book{
		Path:    path,
		data:    data,
		entries: len(data) / 16,
	}, nil
}

// NumEntries returns the total number of 16-byte entries in the book.
func (b *Book) NumEntries() int {
	if b == nil {
		return 0
	}
	return b.entries
}

// FindMoves returns all book moves for the given board position.
func (b *Book) FindMoves(board *chess.Board) ([]BookMove, error) {
	if b.entries == 0 {
		return []BookMove{}, nil
	}

	key := HashBoard(board)

	// Binary search for first occurrence of key
	low := 0
	high := b.entries - 1
	firstIdx := -1

	for low <= high {
		mid := low + (high-low)/2
		midKey := b.entryKey(mid)
		if midKey >= key {
			if midKey == key {
				firstIdx = mid
			}
			high = mid - 1
		} else {
			low = mid + 1
		}
	}

	if firstIdx == -1 {
		return []BookMove{}, nil
	}

	// Collect all entries with this key
	var moves []BookMove
	totalWeight := 0

	legalMoves := board.GenerateLegalMoves()

	for i := firstIdx; i < b.entries; i++ {
		entry := b.getEntry(i)
		if entry.Key != key {
			break
		}

		if entry.Weight == 0 {
			continue
		}

		decoded := decodeRawMoveToMove(board, entry.RawMove)
		var legMove chess.Move
		isLegal := false
		for _, lm := range legalMoves {
			if lm.Equals(decoded) {
				legMove = lm
				isLegal = true
				break
			}
		}
		if !isLegal {
			continue
		}

		san := board.MoveToSAN(legMove)

		bm := BookMove{
			UCI:    legMove.String(),
			SAN:    san,
			Weight: int(entry.Weight),
			Learn:  entry.Learn,
		}
		moves = append(moves, bm)
		totalWeight += int(entry.Weight)
	}

	// Calculate percentages
	if totalWeight > 0 {
		for i := range moves {
			moves[i].Percentage = float64(moves[i].Weight) / float64(totalWeight) * 100.0
		}
	}

	// Sort descending by weight
	sort.Slice(moves, func(i, j int) bool {
		return moves[i].Weight > moves[j].Weight
	})

	return moves, nil
}

// PickMove selects a move according to weights or chooses the highest-weight move.
func (b *Book) PickMove(board *chess.Board, pickBest bool) (*BookMove, error) {
	moves, err := b.FindMoves(board)
	if err != nil {
		return nil, err
	}
	if len(moves) == 0 {
		return nil, nil
	}

	if pickBest || len(moves) == 1 {
		return &moves[0], nil
	}

	totalWeight := 0
	for _, m := range moves {
		totalWeight += m.Weight
	}

	if totalWeight <= 0 {
		return &moves[0], nil
	}

	r := rand.Intn(totalWeight)
	running := 0
	for i := range moves {
		running += moves[i].Weight
		if r < running {
			return &moves[i], nil
		}
	}

	return &moves[0], nil
}

func (b *Book) entryKey(idx int) uint64 {
	offset := idx * 16
	return binary.BigEndian.Uint64(b.data[offset : offset+8])
}

func (b *Book) getEntry(idx int) BookEntry {
	offset := idx * 16
	return BookEntry{
		Key:     binary.BigEndian.Uint64(b.data[offset : offset+8]),
		RawMove: binary.BigEndian.Uint16(b.data[offset+8 : offset+10]),
		Weight:  binary.BigEndian.Uint16(b.data[offset+10 : offset+12]),
		Learn:   binary.BigEndian.Uint32(b.data[offset+12 : offset+16]),
	}
}

func decodeRawMoveToMove(board *chess.Board, raw uint16) chess.Move {
	toFile := int(raw & 0x7)
	toRank := int((raw >> 3) & 0x7)
	fromFile := int((raw >> 6) & 0x7)
	fromRank := int((raw >> 9) & 0x7)
	promo := int((raw >> 12) & 0x7)

	fromSq := chess.Square(fromRank*8 + fromFile)
	toSq := chess.Square(toRank*8 + toFile)

	// Handle Castling:
	// Polyglot represents White O-O as e1h1, White O-O-O as e1a1, Black O-O as e8h8, Black O-O-O as e8a8
	if promo == 0 {
		if fromSq == chess.E1 && board.Squares[chess.E1] == chess.WK {
			if toSq == chess.H1 {
				return chess.Move{From: chess.E1, To: chess.G1}
			} else if toSq == chess.A1 {
				return chess.Move{From: chess.E1, To: chess.C1}
			}
		} else if fromSq == chess.E8 && board.Squares[chess.E8] == chess.BK {
			if toSq == chess.H8 {
				return chess.Move{From: chess.E8, To: chess.G8}
			} else if toSq == chess.A8 {
				return chess.Move{From: chess.E8, To: chess.C8}
			}
		}
	}

	var promoPiece chess.PieceType
	switch promo {
	case 1:
		promoPiece = chess.Knight
	case 2:
		promoPiece = chess.Bishop
	case 3:
		promoPiece = chess.Rook
	case 4:
		promoPiece = chess.Queen
	}

	return chess.Move{From: fromSq, To: toSq, Promo: promoPiece}
}

func decodeRawMove(board *chess.Board, raw uint16) string {
	return decodeRawMoveToMove(board, raw).String()
}

// EncodeRawMove encodes a chess.Move into the standard 16-bit Polyglot move format.
func EncodeRawMove(board *chess.Board, m chess.Move) uint16 {
	fromSq := int(m.From)
	toSq := int(m.To)

	// Castling conversion (King moves to Rook square in Polyglot)
	if m.Promo == chess.None {
		if (fromSq == int(chess.E1) && (board == nil || board.Squares[chess.E1] == chess.WK)) {
			if toSq == int(chess.G1) {
				toSq = int(chess.H1)
			} else if toSq == int(chess.C1) {
				toSq = int(chess.A1)
			}
		} else if (fromSq == int(chess.E8) && (board == nil || board.Squares[chess.E8] == chess.BK)) {
			if toSq == int(chess.G8) {
				toSq = int(chess.H8)
			} else if toSq == int(chess.C8) {
				toSq = int(chess.A8)
			}
		}
	}

	fromFile := fromSq % 8
	fromRank := fromSq / 8
	toFile := toSq % 8
	toRank := toSq / 8

	promo := 0
	switch m.Promo {
	case chess.Knight:
		promo = 1
	case chess.Bishop:
		promo = 2
	case chess.Rook:
		promo = 3
	case chess.Queen:
		promo = 4
	}

	return uint16(toFile | (toRank << 3) | (fromFile << 6) | (fromRank << 9) | (promo << 12))
}

// EncodeBook sorts entries ascending by Key and encodes them into standard Polyglot binary bytes.
func EncodeBook(entries []BookEntry) []byte {
	// Polyglot specification requires entries to be sorted primarily by Key ascending.
	// Secondarily, sort by Weight descending.
	sorted := make([]BookEntry, len(entries))
	copy(sorted, entries)

	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Key != sorted[j].Key {
			return sorted[i].Key < sorted[j].Key
		}
		if sorted[i].Weight != sorted[j].Weight {
			return sorted[i].Weight > sorted[j].Weight
		}
		return sorted[i].RawMove < sorted[j].RawMove
	})

	buf := make([]byte, len(sorted)*16)
	for i, e := range sorted {
		offset := i * 16
		binary.BigEndian.PutUint64(buf[offset:offset+8], e.Key)
		binary.BigEndian.PutUint16(buf[offset+8:offset+10], e.RawMove)
		binary.BigEndian.PutUint16(buf[offset+10:offset+12], e.Weight)
		binary.BigEndian.PutUint32(buf[offset+12:offset+16], e.Learn)
	}
	return buf
}

// WriteBook encodes entries and writes them to a file at path.
func WriteBook(path string, entries []BookEntry) error {
	data := EncodeBook(entries)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// DiscoverBooks looks for opening books (.bin) across standard Scid vs PC and user directories.
func DiscoverBooks(extraDirs ...string) []string {
	dirs := []string{
		filepath.Join(os.Getenv("HOME"), ".local", "share", "newscid", "books", "opening"),
		filepath.Join(os.Getenv("HOME"), ".local", "share", "newscid", "books"),
		"/usr/share/scid/books",
		"/usr/share/games/scid/books",
		"/usr/local/share/scid/books",
		filepath.Join(os.Getenv("HOME"), ".scidvspc", "books"),
		"books",
	}
	dirs = append(dirs, extraDirs...)

	seen := make(map[string]bool)
	var books []string

	for _, d := range dirs {
		if strings.TrimSpace(d) == "" {
			continue
		}
		entries, err := os.ReadDir(d)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".bin") {
				full := filepath.Join(d, e.Name())
				if !seen[full] {
					seen[full] = true
					books = append(books, full)
				}
			}
		}
	}

	sort.Strings(books)
	return books
}
