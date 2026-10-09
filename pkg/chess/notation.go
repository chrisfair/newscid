package chess

import (
	"fmt"
	"strconv"
	"strings"
)

// NotationStyle represents the chess move notation format.
type NotationStyle string

const (
	NotationSAN         NotationStyle = "san"         // Standard Algebraic: Nf3, e4, Bxf7+
	NotationFAN         NotationStyle = "fan"         // Figurine Algebraic: ♘f3, e4, ♗xf7+
	NotationLAN         NotationStyle = "lan"         // Long Algebraic: Ng1-f3, e2-e4, Bf1xf7+
	NotationDescriptive NotationStyle = "descriptive" // English Descriptive / Classical: N-KB3, P-K4, BxP+
	NotationUCI         NotationStyle = "uci"         // Coordinate / Engine: g1f3, e2e4, f1f7
	NotationICCF        NotationStyle = "iccf"        // Numeric Correspondence: 7163, 5254, 6167
)

// ValidNotationStyle returns whether the given string is a valid notation style.
func ValidNotationStyle(s string) bool {
	switch NotationStyle(strings.ToLower(s)) {
	case NotationSAN, NotationFAN, NotationLAN, NotationDescriptive, NotationUCI, NotationICCF:
		return true
	default:
		return false
	}
}

// MoveToNotation formats a legal move according to the specified notation style.
func (b *Board) MoveToNotation(m Move, style NotationStyle) string {
	switch style {
	case NotationFAN:
		return b.MoveToFAN(m)
	case NotationLAN:
		return b.MoveToLAN(m)
	case NotationDescriptive:
		return b.MoveToDescriptive(m)
	case NotationUCI:
		return m.String()
	case NotationICCF:
		return b.MoveToICCF(m)
	case NotationSAN:
		fallthrough
	default:
		return b.MoveToSAN(m)
	}
}

// MoveToFAN formats a move into Figurine Algebraic Notation using Unicode chess glyphs.
func (b *Board) MoveToFAN(m Move) string {
	san := b.MoveToSAN(m)
	return SANToFAN(san)
}

// SANToFAN converts standard algebraic notation into figurine algebraic notation.
func SANToFAN(san string) string {
	if len(san) == 0 {
		return san
	}

	// Do not alter castling
	if strings.HasPrefix(san, "O-O") || strings.HasPrefix(san, "0-0") {
		return san
	}

	res := san
	// Replace leading piece character
	firstChar := res[0]
	switch firstChar {
	case 'K':
		res = "♔" + res[1:]
	case 'Q':
		res = "♕" + res[1:]
	case 'R':
		res = "♖" + res[1:]
	case 'B':
		res = "♗" + res[1:]
	case 'N':
		res = "♘" + res[1:]
	}

	// Replace promotion piece if present
	res = strings.ReplaceAll(res, "=Q", "=♕")
	res = strings.ReplaceAll(res, "=R", "=♖")
	res = strings.ReplaceAll(res, "=B", "=♗")
	res = strings.ReplaceAll(res, "=N", "=♘")

	return res
}

// MoveToLAN formats a move into Long Algebraic Notation (e.g. "Ng1-f3", "e2-e4", "Bf1xc4").
func (b *Board) MoveToLAN(m Move) string {
	fromPiece := b.Squares[m.From]
	pt := fromPiece.Type()

	// Castling
	if pt == King {
		if (m.From == E1 && m.To == G1) || (m.From == E8 && m.To == G8) {
			return b.appendCheckSuffix(m, "O-O")
		}
		if (m.From == E1 && m.To == C1) || (m.From == E8 && m.To == C8) {
			return b.appendCheckSuffix(m, "O-O-O")
		}
	}

	var sb strings.Builder
	if pt != Pawn {
		sb.WriteString(pt.String())
	}

	sb.WriteString(m.From.String())

	isCapture := b.Squares[m.To] != Empty || (pt == Pawn && m.To == b.EPSquare)
	if isCapture {
		sb.WriteByte('x')
	} else {
		sb.WriteByte('-')
	}

	sb.WriteString(m.To.String())

	if m.Promo != None {
		sb.WriteByte('=')
		sb.WriteString(m.Promo.String())
	}

	return b.appendCheckSuffix(m, sb.String())
}

// MoveToICCF formats a move into 4-digit (or 5-digit for promotion) ICCF Numeric notation.
func (b *Board) MoveToICCF(m Move) string {
	fromCol := m.From.File() + 1
	fromRow := m.From.Rank() + 1
	toCol := m.To.File() + 1
	toRow := m.To.Rank() + 1

	base := fmt.Sprintf("%d%d%d%d", fromCol, fromRow, toCol, toRow)
	if m.Promo != None {
		switch m.Promo {
		case Queen:
			base += "1"
		case Rook:
			base += "2"
		case Bishop:
			base += "3"
		case Knight:
			base += "4"
		}
	}
	return base
}

// MoveToDescriptive formats a move into English Classical / Descriptive notation.
func (b *Board) MoveToDescriptive(m Move) string {
	fromPiece := b.Squares[m.From]
	pt := fromPiece.Type()

	// 1. Castling
	if pt == King {
		if (m.From == E1 && m.To == G1) || (m.From == E8 && m.To == G8) {
			return b.appendCheckSuffix(m, "O-O")
		}
		if (m.From == E1 && m.To == C1) || (m.From == E8 && m.To == C8) {
			return b.appendCheckSuffix(m, "O-O-O")
		}
	}

	fileDescs := [8]string{"QR", "QN", "QB", "Q", "K", "KB", "KN", "KR"}
	targetFileDesc := fileDescs[m.To.File()]

	targetRank := m.To.Rank() + 1
	if b.Turn == Black {
		targetRank = 8 - m.To.Rank()
	}

	isCapture := b.Squares[m.To] != Empty || (pt == Pawn && m.To == b.EPSquare)

	var sb strings.Builder

	if isCapture {
		// Captured piece name
		capturedPiece := b.Squares[m.To].Type()
		if pt == Pawn && m.To == b.EPSquare {
			capturedPiece = Pawn
		}
		capName := capturedPiece.String()

		if pt == Pawn {
			// Check if multiple pawns could capture at target
			candPawns := 0
			for _, lm := range b.GenerateLegalMoves() {
				if lm.To == m.To && b.Squares[lm.From].Type() == Pawn {
					candPawns++
				}
			}
			if candPawns > 1 {
				fromFile := fileDescs[m.From.File()]
				sb.WriteString(string(fromFile[0]))
			}
			sb.WriteString("PxP")
			if pt == Pawn && m.To == b.EPSquare {
				sb.WriteString(" e.p.")
			}
		} else {
			// Piece capture
			candPieces := 0
			for _, lm := range b.GenerateLegalMoves() {
				if lm.To == m.To && b.Squares[lm.From].Type() == pt {
					candPieces++
				}
			}
			if candPieces > 1 {
				side := "K"
				if m.From.File() < 4 {
					side = "Q"
				}
				sb.WriteString(side)
			}
			sb.WriteString(pt.String())
			sb.WriteByte('x')
			sb.WriteString(capName)
		}
	} else {
		// Non-capture move
		if pt == Pawn {
			sb.WriteString(fmt.Sprintf("P-%s%d", targetFileDesc, targetRank))
		} else {
			candPieces := 0
			for _, lm := range b.GenerateLegalMoves() {
				if lm.To == m.To && b.Squares[lm.From].Type() == pt {
					candPieces++
				}
			}
			if candPieces > 1 {
				side := "K"
				if m.From.File() < 4 {
					side = "Q"
				}
				sb.WriteString(side)
			}
			sb.WriteString(pt.String())
			sb.WriteByte('-')
			sb.WriteString(targetFileDesc)
			sb.WriteString(strconv.Itoa(targetRank))
		}
	}

	// Promotion suffix
	if m.Promo != None {
		sb.WriteString(fmt.Sprintf("(%s)", m.Promo.String()))
	}

	return b.appendCheckSuffix(m, sb.String())
}

// FormatMoveWithFEN formats a UCI move string given the position FEN into the target notation style.
func FormatMoveWithFEN(fen string, moveUCI string, style NotationStyle) (string, error) {
	b, err := ParseFEN(fen)
	if err != nil {
		return "", err
	}
	m, err := b.ParseUCI(moveUCI)
	if err != nil {
		return "", err
	}
	return b.MoveToNotation(m, style), nil
}

// FormatPVWithStyle formats a slice of UCI moves into a move sequence string using the given notation style.
func FormatPVWithStyle(startFEN string, pvMoves []string, style NotationStyle) string {
	if len(pvMoves) == 0 {
		return ""
	}

	b, err := ParseFEN(startFEN)
	if err != nil {
		return strings.Join(pvMoves, " ")
	}

	var sb strings.Builder
	moveNum := b.FullMove
	isWhite := b.Turn == White

	for i, uci := range pvMoves {
		m, err := b.ParseUCI(uci)
		if err != nil {
			// Fallback: append remaining as UCI
			if i > 0 {
				sb.WriteByte(' ')
			}
			sb.WriteString(strings.Join(pvMoves[i:], " "))
			break
		}

		if i > 0 {
			sb.WriteByte(' ')
		}

		if isWhite {
			sb.WriteString(fmt.Sprintf("%d. ", moveNum))
		} else if i == 0 {
			sb.WriteString(fmt.Sprintf("%d... ", moveNum))
		}

		notated := b.MoveToNotation(m, style)
		sb.WriteString(notated)

		b.MakeMove(m)
		if !isWhite {
			moveNum++
		}
		isWhite = !isWhite
	}

	return sb.String()
}
