package chess

import (
	"fmt"
	"strings"
)

// MoveToSAN formats a legal move into Standard Algebraic Notation (SAN).
func (b *Board) MoveToSAN(m Move) string {
	fromPiece := b.Squares[m.From]
	pt := fromPiece.Type()

	// 1. Castling
	if pt == King {
		if m.From == E1 && m.To == G1 || m.From == E8 && m.To == G8 {
			san := "O-O"
			return b.appendCheckSuffix(m, san)
		}
		if m.From == E1 && m.To == C1 || m.From == E8 && m.To == C8 {
			san := "O-O-O"
			return b.appendCheckSuffix(m, san)
		}
	}

	var sb strings.Builder

	// 2. Pawn moves
	if pt == Pawn {
		if m.From.File() != m.To.File() {
			// Pawn capture
			sb.WriteByte(byte('a' + m.From.File()))
			sb.WriteByte('x')
		}
		sb.WriteString(m.To.String())
		if m.Promo != None {
			sb.WriteByte('=')
			sb.WriteString(m.Promo.String())
		}
		return b.appendCheckSuffix(m, sb.String())
	}

	// 3. Piece moves: King, Queen, Rook, Bishop, Knight
	sb.WriteString(pt.String())

	// Disambiguation
	candidates := b.findDisambiguationCandidates(m)
	if len(candidates) > 0 {
		sameFile := false
		sameRank := false
		for _, cand := range candidates {
			if cand.File() == m.From.File() {
				sameFile = true
			}
			if cand.Rank() == m.From.Rank() {
				sameRank = true
			}
		}

		if !sameFile {
			sb.WriteByte(byte('a' + m.From.File()))
		} else if !sameRank {
			sb.WriteByte(byte('1' + m.From.Rank()))
		} else {
			sb.WriteByte(byte('a' + m.From.File()))
			sb.WriteByte(byte('1' + m.From.Rank()))
		}
	}

	// Capture indicator
	if b.Squares[m.To] != Empty {
		sb.WriteByte('x')
	}

	sb.WriteString(m.To.String())

	return b.appendCheckSuffix(m, sb.String())
}

func (b *Board) findDisambiguationCandidates(m Move) []Square {
	pt := b.Squares[m.From].Type()
	var candidates []Square

	allMoves := b.GenerateLegalMoves()
	for _, legalMove := range allMoves {
		if legalMove.From != m.From && legalMove.To == m.To {
			if b.Squares[legalMove.From].Type() == pt {
				candidates = append(candidates, legalMove.From)
			}
		}
	}
	return candidates
}

func (b *Board) appendCheckSuffix(m Move, san string) string {
	h := b.MakeMove(m)
	defer b.UnmakeMove(h)

	if b.IsCheckmate() {
		return san + "#"
	}
	if b.InCheck(b.Turn) {
		return san + "+"
	}
	return san
}

// ParseSAN parses a SAN move string in the current position and returns the matching legal Move.
func (b *Board) ParseSAN(san string) (Move, error) {
	clean := strings.Trim(san, "+#?! \t\r\n")
	legalMoves := b.GenerateLegalMoves()

	// Fast check for castling
	if clean == "O-O" || clean == "0-0" {
		for _, m := range legalMoves {
			if b.Squares[m.From].Type() == King && (m.To.File()-m.From.File() == 2) {
				return m, nil
			}
		}
		return Move{}, fmt.Errorf("illegal kingside castle: %s", san)
	}
	if clean == "O-O-O" || clean == "0-0-0" {
		for _, m := range legalMoves {
			if b.Squares[m.From].Type() == King && (m.From.File()-m.To.File() == 2) {
				return m, nil
			}
		}
		return Move{}, fmt.Errorf("illegal queenside castle: %s", san)
	}

	// Parse promotion if present (e.g. e8=Q or e8Q)
	promo := None
	if idx := strings.IndexAny(clean, "=QRBNqrbn"); idx != -1 {
		lastChar := clean[len(clean)-1]
		switch strings.ToUpper(string(lastChar)) {
		case "Q":
			promo = Queen
		case "R":
			promo = Rook
		case "B":
			promo = Bishop
		case "N":
			promo = Knight
		}
		clean = strings.TrimRight(clean, "=QRBNqrbn")
	}

	clean = strings.ReplaceAll(clean, "x", "")

	// Last 2 chars are the destination square
	if len(clean) < 2 {
		return Move{}, fmt.Errorf("invalid SAN move: %s", san)
	}
	toStr := clean[len(clean)-2:]
	toSq := SquareFromString(toStr)
	if toSq == NoSquare {
		return Move{}, fmt.Errorf("invalid destination square in SAN: %s", san)
	}

	prefix := clean[:len(clean)-2]
	var movingPieceType = Pawn
	var disFile = -1
	var disRank = -1

	if len(prefix) > 0 {
		first := prefix[0]
		switch first {
		case 'K':
			movingPieceType = King
			prefix = prefix[1:]
		case 'Q':
			movingPieceType = Queen
			prefix = prefix[1:]
		case 'R':
			movingPieceType = Rook
			prefix = prefix[1:]
		case 'B':
			movingPieceType = Bishop
			prefix = prefix[1:]
		case 'N':
			movingPieceType = Knight
			prefix = prefix[1:]
		default:
			movingPieceType = Pawn
		}
	}

	// Disambiguation file / rank in remaining prefix
	for _, ch := range prefix {
		if ch >= 'a' && ch <= 'h' {
			disFile = int(ch - 'a')
		} else if ch >= '1' && ch <= '8' {
			disRank = int(ch - '1')
		}
	}

	// Find matching legal move
	var matched []Move
	for _, m := range legalMoves {
		if m.To != toSq {
			continue
		}
		if b.Squares[m.From].Type() != movingPieceType {
			continue
		}
		if promo != None && m.Promo != promo {
			continue
		}
		if disFile != -1 && m.From.File() != disFile {
			continue
		}
		if disRank != -1 && m.From.Rank() != disRank {
			continue
		}
		matched = append(matched, m)
	}

	if len(matched) == 1 {
		return matched[0], nil
	}
	if len(matched) == 0 {
		return Move{}, fmt.Errorf("no legal move matches SAN: %s", san)
	}
	return Move{}, fmt.Errorf("ambiguous SAN move: %s (matches %d moves)", san, len(matched))
}

// ParseUCI parses a coordinate move like "e2e4" or "e7e8q".
func (b *Board) ParseUCI(uci string) (Move, error) {
	uci = strings.TrimSpace(strings.ToLower(uci))
	if len(uci) < 4 {
		return Move{}, fmt.Errorf("invalid UCI move: %s", uci)
	}

	from := SquareFromString(uci[0:2])
	to := SquareFromString(uci[2:4])
	if from == NoSquare || to == NoSquare {
		return Move{}, fmt.Errorf("invalid squares in UCI: %s", uci)
	}

	promo := None
	if len(uci) >= 5 {
		switch uci[4] {
		case 'q':
			promo = Queen
		case 'r':
			promo = Rook
		case 'b':
			promo = Bishop
		case 'n':
			promo = Knight
		}
	}

	legalMoves := b.GenerateLegalMoves()
	for _, m := range legalMoves {
		if m.From == from && m.To == to && m.Promo == promo {
			return m, nil
		}
	}

	return Move{}, fmt.Errorf("illegal UCI move for position: %s", uci)
}
