package chess

import (
	"fmt"
	"strconv"
	"strings"
)

const StartingFEN = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"

// FEN generates the standard Forsyth-Edwards Notation string for the board.
func (b *Board) FEN() string {
	var sb strings.Builder

	// Piece placement
	for rank := 7; rank >= 0; rank-- {
		emptyCount := 0
		for file := 0; file < 8; file++ {
			p := b.Squares[MakeSquare(file, rank)]
			if p == Empty {
				emptyCount++
			} else {
				if emptyCount > 0 {
					sb.WriteByte('0' + byte(emptyCount))
					emptyCount = 0
				}
				sb.WriteByte(p.Byte())
			}
		}
		if emptyCount > 0 {
			sb.WriteByte('0' + byte(emptyCount))
		}
		if rank > 0 {
			sb.WriteByte('/')
		}
	}

	// Active color
	sb.WriteByte(' ')
	if b.Turn == White {
		sb.WriteByte('w')
	} else {
		sb.WriteByte('b')
	}

	// Castling availability
	sb.WriteByte(' ')
	if b.Castling == 0 {
		sb.WriteByte('-')
	} else {
		if (b.Castling & CastlingWhiteKing) != 0 {
			sb.WriteByte('K')
		}
		if (b.Castling & CastlingWhiteQueen) != 0 {
			sb.WriteByte('Q')
		}
		if (b.Castling & CastlingBlackKing) != 0 {
			sb.WriteByte('k')
		}
		if (b.Castling & CastlingBlackQueen) != 0 {
			sb.WriteByte('q')
		}
	}

	// En passant target square
	sb.WriteByte(' ')
	if b.EPSquare != NoSquare {
		sb.WriteString(b.EPSquare.String())
	} else {
		sb.WriteByte('-')
	}

	// Halfmove clock
	sb.WriteByte(' ')
	sb.WriteString(strconv.Itoa(b.HalfMove))

	// Fullmove number
	sb.WriteByte(' ')
	sb.WriteString(strconv.Itoa(b.FullMove))

	return sb.String()
}

// ParseFEN parses a FEN string into a Board.
func ParseFEN(fen string) (*Board, error) {
	parts := strings.Fields(strings.TrimSpace(fen))
	if len(parts) < 1 {
		return nil, fmt.Errorf("empty FEN string")
	}

	b := &Board{
		EPSquare: NoSquare,
		HalfMove: 0,
		FullMove: 1,
	}

	// Piece placement
	ranks := strings.Split(parts[0], "/")
	if len(ranks) != 8 {
		return nil, fmt.Errorf("invalid FEN: expected 8 ranks, got %d", len(ranks))
	}

	for rIdx, rankStr := range ranks {
		rank := 7 - rIdx
		file := 0
		for _, ch := range rankStr {
			if ch >= '1' && ch <= '8' {
				count := int(ch - '0')
				for i := 0; i < count; i++ {
					if file > 7 {
						return nil, fmt.Errorf("invalid FEN: rank %d overflow", rank)
					}
					b.Squares[MakeSquare(file, rank)] = Empty
					file++
				}
			} else {
				if file > 7 {
					return nil, fmt.Errorf("invalid FEN: rank %d overflow", rank)
				}
				piece := pieceFromChar(ch)
				if piece == Empty {
					return nil, fmt.Errorf("invalid piece char in FEN: %c", ch)
				}
				b.Squares[MakeSquare(file, rank)] = piece
				file++
			}
		}
		if file != 8 {
			return nil, fmt.Errorf("invalid FEN: rank %d had %d squares instead of 8", rank, file)
		}
	}

	// Active color
	if len(parts) > 1 {
		if len(parts[1]) == 1 && (parts[1][0] == 'w' || parts[1][0] == 'W') {
			b.Turn = White
		} else if len(parts[1]) == 1 && (parts[1][0] == 'b' || parts[1][0] == 'B') {
			b.Turn = Black
		} else {
			return nil, fmt.Errorf("invalid turn in FEN: %s", parts[1])
		}
	} else {
		b.Turn = White
	}

	// Castling
	b.Castling = 0
	if len(parts) > 2 {
		for _, ch := range parts[2] {
			switch ch {
			case 'K':
				b.Castling |= CastlingWhiteKing
			case 'Q':
				b.Castling |= CastlingWhiteQueen
			case 'k':
				b.Castling |= CastlingBlackKing
			case 'q':
				b.Castling |= CastlingBlackQueen
			case '-':
				// No castling
			}
		}
	}

	// En passant
	if len(parts) > 3 && parts[3] != "-" {
		b.EPSquare = SquareFromString(parts[3])
	}

	// Halfmove clock
	if len(parts) > 4 {
		if hm, err := strconv.Atoi(parts[4]); err == nil {
			b.HalfMove = hm
		}
	}

	// Fullmove number
	if len(parts) > 5 {
		if fm, err := strconv.Atoi(parts[5]); err == nil {
			b.FullMove = fm
		}
	}

	// Rebuild PieceList following exact Scid FEN loading order:
	// Squares scanned rank 7 down to 0, file 0 to 7.
	for i := range b.PiecePos {
		b.PiecePos[i] = -1
	}
	b.PieceCount[0] = 0
	b.PieceCount[1] = 0

	for rank := 7; rank >= 0; rank-- {
		for file := 0; file < 8; file++ {
			sq := MakeSquare(file, rank)
			p := b.Squares[sq]
			if p == Empty {
				continue
			}
			c := p.Color()
			pt := p.Type()

			if pt == King {
				if b.PieceCount[c] > 0 && b.PieceCount[c] < 16 {
					oldSq := b.PieceList[c][0]
					b.PieceList[c][b.PieceCount[c]] = oldSq
					b.PiecePos[oldSq] = b.PieceCount[c]
				}
				b.PieceList[c][0] = sq
				b.PiecePos[sq] = 0
				if b.PieceCount[c] < 16 {
					b.PieceCount[c]++
				}
			} else if b.PieceCount[c] < 16 {
				b.PiecePos[sq] = b.PieceCount[c]
				b.PieceList[c][b.PieceCount[c]] = sq
				b.PieceCount[c]++
			}
		}
	}

	return b, nil
}

func pieceFromChar(ch rune) Piece {
	switch ch {
	case 'K':
		return WK
	case 'Q':
		return WQ
	case 'R':
		return WR
	case 'B':
		return WB
	case 'N':
		return WN
	case 'P':
		return WP
	case 'k':
		return BK
	case 'q':
		return BQ
	case 'r':
		return BR
	case 'b':
		return BB
	case 'n':
		return BN
	case 'p':
		return BP
	default:
		return Empty
	}
}
