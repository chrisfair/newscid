package chess

import (
	"fmt"
	"strings"
)

// Color represents White or Black.
type Color uint8

const (
	White Color = 0
	Black Color = 1
	NoColor Color = 2
)

func (c Color) Other() Color {
	if c == White {
		return Black
	}
	return White
}

func (c Color) String() string {
	if c == White {
		return "White"
	} else if c == Black {
		return "Black"
	}
	return "NoColor"
}

// PieceType represents King, Queen, Rook, Bishop, Knight, Pawn, Empty.
type PieceType uint8

const (
	None   PieceType = 0
	King   PieceType = 1
	Queen  PieceType = 2
	Rook   PieceType = 3
	Bishop PieceType = 4
	Knight PieceType = 5
	Pawn   PieceType = 6
)

func (pt PieceType) String() string {
	switch pt {
	case King:
		return "K"
	case Queen:
		return "Q"
	case Rook:
		return "R"
	case Bishop:
		return "B"
	case Knight:
		return "N"
	case Pawn:
		return "P"
	default:
		return ""
	}
}

// Piece represents a colored piece or empty square.
type Piece uint8

const (
	Empty Piece = 0
	WK    Piece = 1
	WQ    Piece = 2
	WR    Piece = 3
	WB    Piece = 4
	WN    Piece = 5
	WP    Piece = 6

	BK Piece = 9
	BQ Piece = 10
	BR Piece = 11
	BB Piece = 12
	BN Piece = 13
	BP Piece = 14
)

func MakePiece(c Color, pt PieceType) Piece {
	if pt == None {
		return Empty
	}
	if c == White {
		return Piece(pt)
	}
	return Piece(pt | 8)
}

func (p Piece) Type() PieceType {
	return PieceType(p & 7)
}

func (p Piece) Color() Color {
	if p == Empty {
		return NoColor
	}
	if (p & 8) != 0 {
		return Black
	}
	return White
}

func (p Piece) String() string {
	chars := " KQRBNP  kqrbnp"
	if int(p) < len(chars) {
		return string(chars[p])
	}
	return "."
}

// Square represents squares 0..63 (A1=0, B1=1 ... H8=63).
type Square uint8

const (
	A1 Square = 0; B1 Square = 1; C1 Square = 2; D1 Square = 3; E1 Square = 4; F1 Square = 5; G1 Square = 6; H1 Square = 7
	A2 Square = 8; B2 Square = 9; C2 Square = 10; D2 Square = 11; E2 Square = 12; F2 Square = 13; G2 Square = 14; H2 Square = 15
	A3 Square = 16; B3 Square = 17; C3 Square = 18; D3 Square = 19; E3 Square = 20; F3 Square = 21; G3 Square = 22; H3 Square = 23
	A4 Square = 24; B4 Square = 25; C4 Square = 26; D4 Square = 27; E4 Square = 28; F4 Square = 29; G4 Square = 30; H4 Square = 31
	A5 Square = 32; B5 Square = 33; C5 Square = 34; D5 Square = 35; E5 Square = 36; F5 Square = 37; G5 Square = 38; H5 Square = 39
	A6 Square = 40; B6 Square = 41; C6 Square = 42; D6 Square = 43; E6 Square = 44; F6 Square = 45; G6 Square = 46; H6 Square = 47
	A7 Square = 48; B7 Square = 49; C7 Square = 50; D7 Square = 51; E7 Square = 52; F7 Square = 53; G7 Square = 54; H7 Square = 55
	A8 Square = 56; B8 Square = 57; C8 Square = 58; D8 Square = 59; E8 Square = 60; F8 Square = 61; G8 Square = 62; H8 Square = 63
	NoSquare Square = 64
)

func MakeSquare(file, rank int) Square {
	if file < 0 || file > 7 || rank < 0 || rank > 7 {
		return NoSquare
	}
	return Square(rank*8 + file)
}

func (s Square) File() int {
	return int(s % 8)
}

func (s Square) Rank() int {
	return int(s / 8)
}

func (s Square) String() string {
	if s >= 64 {
		return "-"
	}
	f := byte('a' + s.File())
	r := byte('1' + s.Rank())
	return string([]byte{f, r})
}

func SquareFromString(str string) Square {
	str = strings.ToLower(strings.TrimSpace(str))
	if len(str) < 2 {
		return NoSquare
	}
	file := int(str[0] - 'a')
	rank := int(str[1] - '1')
	return MakeSquare(file, rank)
}

// Castling rights bitmask.
const (
	CastlingWhiteKing  uint8 = 1
	CastlingWhiteQueen uint8 = 2
	CastlingBlackKing  uint8 = 4
	CastlingBlackQueen uint8 = 8
)

// Move represents a chess move with optional promotion.
type Move struct {
	From    Square
	To      Square
	Promo   PieceType
	Capture PieceType
}

func (m Move) String() string {
	if m.From >= 64 || m.To >= 64 {
		return "0000"
	}
	res := m.From.String() + m.To.String()
	if m.Promo != None {
		res += strings.ToLower(m.Promo.String())
	}
	return res
}

func (m Move) Equals(other Move) bool {
	return m.From == other.From && m.To == other.To && m.Promo == other.Promo
}

// Result represents game outcome.
type Result string

const (
	ResultWhiteWins Result = "1-0"
	ResultBlackWins Result = "0-1"
	ResultDraw      Result = "1/2-1/2"
	ResultUnderway  Result = "*"
)

// MoveNode represents a node in a game move tree with variations and comments.
type MoveNode struct {
	Move       Move        `json:"move"`
	SAN        string      `json:"san"`
	UCI        string      `json:"uci"`
	FEN        string      `json:"fen"`
	Comment    string      `json:"comment,omitempty"`
	NAGs       []int       `json:"nags,omitempty"`
	Variations []*MoveNode `json:"variations,omitempty"`
	Next       *MoveNode   `json:"next,omitempty"`
}

// Game represents a complete chess game with metadata and move tree.
type Game struct {
	ID        int               `json:"id"`
	Tags      map[string]string `json:"tags"`
	StartFEN  string            `json:"startFen"`
	RootMove  *MoveNode         `json:"rootMove"`
	Result    Result            `json:"result"`
	PlyCount  int               `json:"plyCount"`
}

func NewGame() *Game {
	return &Game{
		Tags: map[string]string{
			"Event": "?",
			"Site":  "?",
			"Date":  "????.??.??",
			"Round": "?",
			"White": "?",
			"Black": "?",
			"Result": "*",
		},
		Result: ResultUnderway,
	}
}

func (g *Game) Tag(key string) string {
	if val, ok := g.Tags[key]; ok {
		return val
	}
	return "?"
}

func (g *Game) SetTag(key, value string) {
	if g.Tags == nil {
		g.Tags = make(map[string]string)
	}
	g.Tags[key] = value
	if key == "Result" {
		g.Result = Result(value)
	}
}

func FormatNAG(nag int) string {
	switch nag {
	case 1:
		return "!"
	case 2:
		return "?"
	case 3:
		return "!!"
	case 4:
		return "??"
	case 5:
		return "!?"
	case 6:
		return "?!"
	case 10:
		return "="
	case 14:
		return "+="
	case 15:
		return "=+"
	case 16:
		return "+/-"
	case 17:
		return "-/+"
	case 18:
		return "+-"
	case 19:
		return "-+"
	default:
		return fmt.Sprintf("$%d", nag)
	}
}
