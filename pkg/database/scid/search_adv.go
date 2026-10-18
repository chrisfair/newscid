package scid

import (
	"strings"

	"newscid/pkg/chess"
)

// MaterialCriteria specifies piece balance and endgame material filters.
type MaterialCriteria struct {
	// Exact piece counts (-1 = ignore)
	WhiteQ int `json:"whiteQ"`
	WhiteR int `json:"whiteR"`
	WhiteB int `json:"whiteB"`
	WhiteN int `json:"whiteN"`
	WhiteP int `json:"whiteP"`

	BlackQ int `json:"blackQ"`
	BlackR int `json:"blackR"`
	BlackB int `json:"blackB"`
	BlackN int `json:"blackN"`
	BlackP int `json:"blackP"`

	// Material imbalance / differences (-99 = ignore)
	DiffQueens  int `json:"diffQueens"`  // White Q - Black Q
	DiffRooks   int `json:"diffRooks"`   // White R - Black R
	DiffBishops int `json:"diffBishops"` // White B - Black B
	DiffKnights int `json:"diffKnights"` // White N - Black N
	DiffPawns   int `json:"diffPawns"`   // White P - Black P

	// Specific tactical/endgame patterns
	OppositeBishops bool `json:"oppositeBishops"` // Both sides have bishops on opposite colored squares
	MatchEndOnly    bool `json:"matchEndOnly"`    // Check only the final position of the game
}

// DefaultMaterialCriteria creates an unconstrained material filter.
func DefaultMaterialCriteria() MaterialCriteria {
	return MaterialCriteria{
		WhiteQ: -1, WhiteR: -1, WhiteB: -1, WhiteN: -1, WhiteP: -1,
		BlackQ: -1, BlackR: -1, BlackB: -1, BlackN: -1, BlackP: -1,
		DiffQueens: -99, DiffRooks: -99, DiffBishops: -99, DiffKnights: -99, DiffPawns: -99,
	}
}

// IsActive returns true if at least one material constraint is specified.
func (mc *MaterialCriteria) IsActive() bool {
	if mc == nil {
		return false
	}
	if mc.WhiteQ >= 0 || mc.WhiteR >= 0 || mc.WhiteB >= 0 || mc.WhiteN >= 0 || mc.WhiteP >= 0 {
		return true
	}
	if mc.BlackQ >= 0 || mc.BlackR >= 0 || mc.BlackB >= 0 || mc.BlackN >= 0 || mc.BlackP >= 0 {
		return true
	}
	if mc.DiffQueens != -99 || mc.DiffRooks != -99 || mc.DiffBishops != -99 || mc.DiffKnights != -99 || mc.DiffPawns != -99 {
		return true
	}
	if mc.OppositeBishops {
		return true
	}
	return false
}

// MatchesBoard checks if a chessboard position matches the material criteria.
func (mc *MaterialCriteria) MatchesBoard(b *chess.Board) bool {
	if !mc.IsActive() {
		return true
	}

	var wQ, wR, wB, wN, wP int
	var bQ, bR, bB, bN, bP int
	var whiteBishopSquares []chess.Square
	var blackBishopSquares []chess.Square

	for sq := 0; sq < 64; sq++ {
		p := b.Squares[sq]
		switch p {
		case chess.WQ:
			wQ++
		case chess.WR:
			wR++
		case chess.WB:
			wB++
			whiteBishopSquares = append(whiteBishopSquares, chess.Square(sq))
		case chess.WN:
			wN++
		case chess.WP:
			wP++
		case chess.BQ:
			bQ++
		case chess.BR:
			bR++
		case chess.BB:
			bB++
			blackBishopSquares = append(blackBishopSquares, chess.Square(sq))
		case chess.BN:
			bN++
		case chess.BP:
			bP++
		}
	}

	// Piece count checks
	if mc.WhiteQ >= 0 && wQ != mc.WhiteQ {
		return false
	}
	if mc.WhiteR >= 0 && wR != mc.WhiteR {
		return false
	}
	if mc.WhiteB >= 0 && wB != mc.WhiteB {
		return false
	}
	if mc.WhiteN >= 0 && wN != mc.WhiteN {
		return false
	}
	if mc.WhiteP >= 0 && wP != mc.WhiteP {
		return false
	}

	if mc.BlackQ >= 0 && bQ != mc.BlackQ {
		return false
	}
	if mc.BlackR >= 0 && bR != mc.BlackR {
		return false
	}
	if mc.BlackB >= 0 && bB != mc.BlackB {
		return false
	}
	if mc.BlackN >= 0 && bN != mc.BlackN {
		return false
	}
	if mc.BlackP >= 0 && bP != mc.BlackP {
		return false
	}

	// Difference checks
	if mc.DiffQueens != -99 && (wQ-bQ) != mc.DiffQueens {
		return false
	}
	if mc.DiffRooks != -99 && (wR-bR) != mc.DiffRooks {
		return false
	}
	if mc.DiffBishops != -99 && (wB-bB) != mc.DiffBishops {
		return false
	}
	if mc.DiffKnights != -99 && (wN-bN) != mc.DiffKnights {
		return false
	}
	if mc.DiffPawns != -99 && (wP-bP) != mc.DiffPawns {
		return false
	}

	// Opposite-colored bishops check
	if mc.OppositeBishops {
		if len(whiteBishopSquares) == 0 || len(blackBishopSquares) == 0 {
			return false
		}
		// Check if any White bishop is on opposite color of any Black bishop
		foundOpposite := false
		for _, wSq := range whiteBishopSquares {
			wColor := (int(wSq%8) + int(wSq/8)) % 2
			for _, bSq := range blackBishopSquares {
				bColor := (int(bSq%8) + int(bSq/8)) % 2
				if wColor != bColor {
					foundOpposite = true
					break
				}
			}
			if foundOpposite {
				break
			}
		}
		if !foundOpposite {
			return false
		}
	}

	return true
}

// PawnStructureType represents a recognized pawn structure pattern as an integer constant.
type PawnStructureType uint8

const (
	PawnStructNone PawnStructureType = iota
	PawnStructCarlsbad
	PawnStructIsolani
	PawnStructFrenchChain
	PawnStructMaroczy
	PawnStructHedgehog
	PawnStructDragon
	PawnStructStonewall
	PawnStructUnknown
)

func parsePawnStructureType(structure string) PawnStructureType {
	switch strings.ToLower(strings.TrimSpace(structure)) {
	case "":
		return PawnStructNone
	case "carlsbad":
		return PawnStructCarlsbad
	case "isolani", "iqp":
		return PawnStructIsolani
	case "french_chain", "french":
		return PawnStructFrenchChain
	case "maroczy", "maroczy_bind":
		return PawnStructMaroczy
	case "hedgehog":
		return PawnStructHedgehog
	case "dragon":
		return PawnStructDragon
	case "stonewall":
		return PawnStructStonewall
	default:
		return PawnStructUnknown
	}
}

// MatchesPawnStructure verifies if a board matches common historical pawn formations.
func MatchesPawnStructure(b *chess.Board, structure string) bool {
	return matchesPawnStructureByType(b, parsePawnStructureType(structure))
}

func matchesPawnStructureByType(b *chess.Board, pst PawnStructureType) bool {
	if pst == PawnStructNone {
		return true
	}
	if pst == PawnStructUnknown {
		return false
	}

	hasPawn := func(sq chess.Square, c chess.Color) bool {
		p := b.Squares[sq]
		if c == chess.White {
			return p == chess.WP
		}
		return p == chess.BP
	}

	fileEmptyOfPawns := func(file int, c chess.Color) bool {
		targetPiece := chess.WP
		if c == chess.Black {
			targetPiece = chess.BP
		}
		for rank := 0; rank < 8; rank++ {
			sq := rank*8 + file
			if b.Squares[sq] == targetPiece {
				return false
			}
		}
		return true
	}

	switch pst {
	case PawnStructCarlsbad:
		// White c3, d4, e3 vs Black c6, d5, e6
		return hasPawn(chess.C3, chess.White) && hasPawn(chess.D4, chess.White) && hasPawn(chess.E3, chess.White) &&
			hasPawn(chess.C6, chess.Black) && hasPawn(chess.D5, chess.Black) && hasPawn(chess.E6, chess.Black)

	case PawnStructIsolani:
		// White isolated queen's pawn on d4 (no white c or e pawns)
		return hasPawn(chess.D4, chess.White) && fileEmptyOfPawns(2, chess.White) && fileEmptyOfPawns(4, chess.White)

	case PawnStructFrenchChain:
		// White d4, e5 vs Black d5, e6
		return hasPawn(chess.D4, chess.White) && hasPawn(chess.E5, chess.White) &&
			hasPawn(chess.D5, chess.Black) && hasPawn(chess.E6, chess.Black)

	case PawnStructMaroczy:
		// White c4, e4 vs Black d6
		return hasPawn(chess.C4, chess.White) && hasPawn(chess.E4, chess.White) && hasPawn(chess.D6, chess.Black)

	case PawnStructHedgehog:
		// Black pawns on a6, b6, d6, e6 with c file open for Black
		return hasPawn(chess.A6, chess.Black) && hasPawn(chess.B6, chess.Black) &&
			hasPawn(chess.D6, chess.Black) && hasPawn(chess.E6, chess.Black) && fileEmptyOfPawns(2, chess.Black)

	case PawnStructDragon:
		// Black pawns on d6, g6, h7 with c file open for Black
		return hasPawn(chess.D6, chess.Black) && hasPawn(chess.G6, chess.Black) &&
			hasPawn(chess.H7, chess.Black) && fileEmptyOfPawns(2, chess.Black)

	default: // PawnStructStonewall
		// White c3, d4, e3, f4
		return hasPawn(chess.C3, chess.White) && hasPawn(chess.D4, chess.White) &&
			hasPawn(chess.E3, chess.White) && hasPawn(chess.F4, chess.White)
	}
}

// MatchesGameAnnotation searches game commentary and tags for text keywords.
func MatchesGameAnnotation(g *chess.Game, keyword string, annotator string) bool {
	kw := strings.ToLower(strings.TrimSpace(keyword))
	ann := strings.ToLower(strings.TrimSpace(annotator))

	if kw == "" && ann == "" {
		return true
	}

	if ann != "" {
		tagAnn := strings.ToLower(g.Tag("Annotator"))
		if !strings.Contains(tagAnn, ann) {
			return false
		}
	}

	if kw == "" {
		return true
	}

	// Search move commentary
	matched := false
	var walk func(n *chess.MoveNode)
	walk = func(n *chess.MoveNode) {
		if n == nil || matched {
			return
		}
		if n.Comment != "" && strings.Contains(strings.ToLower(n.Comment), kw) {
			matched = true
			return
		}
		for _, v := range n.Variations {
			walk(v)
			if matched {
				return
			}
		}
		walk(n.Next)
	}

	walk(g.RootMove)
	return matched
}
