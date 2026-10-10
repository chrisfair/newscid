package explainer

import (
	"fmt"
	"strings"
	"newscid/pkg/chess"
)

// Explanation contains human-readable analysis of a played move.
type Explanation struct {
	MoveUCI           string   `json:"moveUci"`
	MoveSAN           string   `json:"moveSan"`
	Assessment        string   `json:"assessment"`        // "Best Move", "Strong Tactical Shot", "Positional Improvement", "Inaccuracy", "Blunder"
	Summary           string   `json:"summary"`           // Primary 1-sentence headline
	TacticalMotifs    []string `json:"tacticalMotifs"`    // e.g. ["Fork", "Discovered Attack", "Passed Pawn Advance"]
	StrategicConcepts []string `json:"strategicConcepts"` // e.g. ["King Safety", "Control of d5 Outpost", "Open File Infiltration"]
	DetailedText      string   `json:"detailedText"`      // Full natural language explanation
	BestAlternative   string   `json:"bestAlternative,omitempty"`
}

// ExplainMove evaluates a move from beforeFEN to afterFEN with engine context.
func ExplainMove(beforeFEN, moveUCI string, scoreCp int, bestMove string) (*Explanation, error) {
	bBefore, err := chess.ParseFEN(beforeFEN)
	if err != nil {
		return nil, fmt.Errorf("invalid before FEN: %w", err)
	}

	m, err := bBefore.ParseUCI(moveUCI)
	if err != nil {
		return nil, fmt.Errorf("invalid move UCI: %w", err)
	}

	movingPiece := bBefore.Squares[m.From]
	capturedPiece := bBefore.Squares[m.To]

	// Format SAN
	moveSAN := bBefore.MoveToSAN(m)
	if moveSAN == "" {
		moveSAN = moveUCI
	}

	// Make move on a clone to inspect resulting position
	bAfter, _ := chess.ParseFEN(beforeFEN)
	bAfter.MakeMove(m)

	motifs := []string{}
	strategic := []string{}
	explanations := []string{}

	colorName := "White"
	enemyName := "Black"
	if movingPiece.Color() == chess.Black {
		colorName = "Black"
		enemyName = "White"
	}

	// 1. Check & Checkmate
	if bAfter.IsCheckmate() {
		motifs = append(motifs, "Checkmate")
		explanations = append(explanations, fmt.Sprintf("%s delivers checkmate, concluding the game with decisive victory.", colorName))
	} else if bAfter.InCheck(bAfter.Turn) {
		motifs = append(motifs, "Check")
		explanations = append(explanations, fmt.Sprintf("%s gives check, forcing the %s king to respond.", colorName, enemyName))
	}

	// 2. Castling
	if movingPiece.Type() == chess.King && (m.From == 4 && (m.To == 6 || m.To == 2) || m.From == 60 && (m.To == 62 || m.To == 58)) {
		motifs = append(motifs, "Castling")
		strategic = append(strategic, "King Safety", "Rook Activation")
		side := "kingside"
		if m.To%8 < m.From%8 {
			side = "queenside"
		}
		explanations = append(explanations, fmt.Sprintf("%s castles %s, tucking the king into pawn shelter and activating the rook.", colorName, side))
	}

	// 3. Captures & Material Trades
	if capturedPiece.Type() != chess.None {
		valFrom := pieceValue(movingPiece.Type())
		valTo := pieceValue(capturedPiece.Type())
		motifs = append(motifs, "Capture")

		if valTo > valFrom {
			motifs = append(motifs, "Favorable Exchange")
			explanations = append(explanations, fmt.Sprintf("%s wins material by taking the %s with a lower-value piece.", colorName, pieceName(capturedPiece.Type())))
		} else if valTo == valFrom {
			explanations = append(explanations, fmt.Sprintf("%s trades off the %s, altering the piece coordination.", colorName, pieceName(capturedPiece.Type())))
		} else {
			explanations = append(explanations, fmt.Sprintf("%s eliminates the %s on %s.", colorName, pieceName(capturedPiece.Type()), squareName(m.To)))
		}
	}

	// 4. Pawn Promotions & Passed Pawns
	if m.Promo != chess.None {
		motifs = append(motifs, "Pawn Promotion")
		explanations = append(explanations, fmt.Sprintf("%s successfully promotes a pawn to a %s.", colorName, pieceName(m.Promo)))
	} else if movingPiece.Type() == chess.Pawn {
		toRank := int(m.To / 8)
		if (movingPiece.Color() == chess.White && toRank >= 5) || (movingPiece.Color() == chess.Black && toRank <= 2) {
			motifs = append(motifs, "Passed Pawn Advance")
			strategic = append(strategic, "Promotion Threat")
			explanations = append(explanations, fmt.Sprintf("%s advances a dangerous passed pawn closer to the promotion rank.", colorName))
		}
	}

	// 5. Piece Fork (Attacking 2+ enemy pieces)
	attackedPieces := countAttackedPieces(bAfter, m.To, movingPiece.Color())
	if attackedPieces >= 2 {
		motifs = append(motifs, "Fork / Double Attack")
		explanations = append(explanations, fmt.Sprintf("The %s on %s creates a double attack against multiple %s targets.", pieceName(movingPiece.Type()), squareName(m.To), enemyName))
	}

	// 6. Central Outpost
	if isCentralOutpost(int(m.To)) {
		strategic = append(strategic, fmt.Sprintf("Control of %s Outpost", squareName(m.To)))
		if movingPiece.Type() == chess.Knight || movingPiece.Type() == chess.Bishop {
			explanations = append(explanations, fmt.Sprintf("Positions the %s on a dominant central outpost square at %s.", pieceName(movingPiece.Type()), squareName(m.To)))
		}
	}

	// 7. Rook on Open File
	if movingPiece.Type() == chess.Rook && isOpenFile(bAfter, int(m.To%8)) {
		strategic = append(strategic, "Open File Control")
		explanations = append(explanations, fmt.Sprintf("%s seizes control of the open %c-file with the rook.", colorName, 'a'+(m.To%8)))
	}

	// Assessment logic
	assessment := "Good Move"
	if len(motifs) > 0 && (contains(motifs, "Checkmate") || contains(motifs, "Favorable Exchange") || contains(motifs, "Fork / Double Attack")) {
		assessment = "Tactical Shot"
	} else if len(strategic) > 0 {
		assessment = "Positional Improvement"
	}

	if bestMove != "" && bestMove != moveUCI && (strings.HasPrefix(bestMove, "e") || len(bestMove) >= 4) {
		// If played move differed from best move
		if scoreCp < -150 && movingPiece.Color() == chess.White || scoreCp > 150 && movingPiece.Color() == chess.Black {
			assessment = "Inaccuracy / Mistake"
			explanations = append(explanations, fmt.Sprintf("Computer analysis suggests %s was stronger in this position.", bestMove))
		}
	}

	summary := fmt.Sprintf("%s played %s", colorName, moveSAN)
	if len(explanations) > 0 {
		summary = explanations[0]
	}

	detailed := strings.Join(explanations, " ")
	if detailed == "" {
		detailed = fmt.Sprintf("%s moves the %s from %s to %s to improve piece placement.", colorName, pieceName(movingPiece.Type()), squareName(m.From), squareName(m.To))
	}

	return &Explanation{
		MoveUCI:           moveUCI,
		MoveSAN:           moveSAN,
		Assessment:        assessment,
		Summary:           summary,
		TacticalMotifs:    unique(motifs),
		StrategicConcepts: unique(strategic),
		DetailedText:      detailed,
		BestAlternative:   bestMove,
	}, nil
}

func pieceValue(pt chess.PieceType) int {
	switch pt {
	case chess.Pawn:
		return 1
	case chess.Knight, chess.Bishop:
		return 3
	case chess.Rook:
		return 5
	case chess.Queen:
		return 9
	default:
		return 0
	}
}

func pieceName(pt chess.PieceType) string {
	switch pt {
	case chess.Pawn:
		return "pawn"
	case chess.Knight:
		return "knight"
	case chess.Bishop:
		return "bishop"
	case chess.Rook:
		return "rook"
	case chess.Queen:
		return "queen"
	case chess.King:
		return "king"
	default:
		return "piece"
	}
}

func squareName(sq chess.Square) string {
	f := rune('a' + (sq % 8))
	r := rune('1' + (sq / 8))
	return fmt.Sprintf("%c%c", f, r)
}

func isCentralOutpost(sq int) bool {
	f := sq % 8
	r := sq / 8
	return (f >= 2 && f <= 5) && (r >= 2 && r <= 5)
}

func isOpenFile(b *chess.Board, file int) bool {
	for r := 0; r < 8; r++ {
		if b.Squares[r*8+file].Type() == chess.Pawn {
			return false
		}
	}
	return true
}

func countAttackedPieces(b *chess.Board, fromSquare chess.Square, friendlyColor chess.Color) int {
	enemyColor := chess.Black
	if friendlyColor == chess.Black {
		enemyColor = chess.White
	}

	count := 0
	for sq := 0; sq < 64; sq++ {
		p := b.Squares[sq]
		if p.Color() == enemyColor && p.Type() != chess.None && p.Type() != chess.Pawn {
			if b.IsSquareAttacked(chess.Square(sq), friendlyColor) {
				count++
			}
		}
	}
	return count
}

func contains(slice []string, val string) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}

func unique(slice []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range slice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}
