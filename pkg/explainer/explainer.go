package explainer

import (
	"fmt"
	"strings"

	"newscid/pkg/chess"
)

type motifID uint8

const (
	motifCheckmate motifID = iota
	motifCheck
	motifCastling
	motifCapture
	motifFavorableExchange
	motifPromotion
	motifPassedPawn
	motifFork
	motifCount
)

var motifLabels = [motifCount]string{
	motifCheckmate:         "Checkmate",
	motifCheck:             "Check",
	motifCastling:          "Castling",
	motifCapture:           "Capture",
	motifFavorableExchange: "Favorable Exchange",
	motifPromotion:         "Pawn Promotion",
	motifPassedPawn:        "Passed Pawn Advance",
	motifFork:              "Fork / Double Attack",
}

type assessmentID uint8

const (
	assessGoodMove assessmentID = iota
	assessTacticalShot
	assessPositionalImprovement
	assessInaccuracyMistake
)

var assessmentLabels = [4]string{
	assessGoodMove:              "Good Move",
	assessTacticalShot:          "Tactical Shot",
	assessPositionalImprovement: "Positional Improvement",
	assessInaccuracyMistake:     "Inaccuracy / Mistake",
}

var pieceValues = [7]int{
	chess.None:   0,
	chess.King:   0,
	chess.Queen:  9,
	chess.Rook:   5,
	chess.Bishop: 3,
	chess.Knight: 3,
	chess.Pawn:   1,
}

var pieceNames = [7]string{
	chess.None:   "piece",
	chess.King:   "king",
	chess.Queen:  "queen",
	chess.Rook:   "rook",
	chess.Bishop: "bishop",
	chess.Knight: "knight",
	chess.Pawn:   "pawn",
}

var colorNames = [2]string{
	chess.White: "White",
	chess.Black: "Black",
}

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

	// Make move on a clone to inspect resulting position
	bAfter := bBefore.Clone()
	bAfter.MakeMove(m)

	var motifMask uint16
	motifs := make([]string, 0, 4)
	addMotif := func(id motifID) {
		bit := uint16(1) << id
		if motifMask&bit == 0 {
			motifMask |= bit
			motifs = append(motifs, motifLabels[id])
		}
	}

	strategic := []string{}
	explanations := []string{}

	moverColor := movingPiece.Color()
	colorName := colorNames[moverColor&1]
	enemyName := colorNames[moverColor.Other()&1]

	// Check & Checkmate
	if bAfter.IsCheckmate() {
		addMotif(motifCheckmate)
		explanations = append(explanations, fmt.Sprintf("%s delivers checkmate, concluding the game with decisive victory.", colorName))
	} else if bAfter.InCheck(bAfter.Turn) {
		addMotif(motifCheck)
		explanations = append(explanations, fmt.Sprintf("%s gives check, forcing the %s king to respond.", colorName, enemyName))
	}

	// Castling
	if movingPiece.Type() == chess.King && (m.From == 4 && (m.To == 6 || m.To == 2) || m.From == 60 && (m.To == 62 || m.To == 58)) {
		addMotif(motifCastling)
		strategic = append(strategic, "King Safety", "Rook Activation")
		side := "kingside"
		if m.To%8 < m.From%8 {
			side = "queenside"
		}
		explanations = append(explanations, fmt.Sprintf("%s castles %s, tucking the king into pawn shelter and activating the rook.", colorName, side))
	}

	// Captures & Material Trades
	if capturedPiece.Type() != chess.None {
		valFrom := pieceValue(movingPiece.Type())
		valTo := pieceValue(capturedPiece.Type())
		addMotif(motifCapture)

		if valTo > valFrom {
			addMotif(motifFavorableExchange)
			explanations = append(explanations, fmt.Sprintf("%s wins material by taking the %s with a lower-value piece.", colorName, pieceName(capturedPiece.Type())))
		} else if valTo == valFrom {
			explanations = append(explanations, fmt.Sprintf("%s trades off the %s, altering the piece coordination.", colorName, pieceName(capturedPiece.Type())))
		} else {
			explanations = append(explanations, fmt.Sprintf("%s eliminates the %s on %s.", colorName, pieceName(capturedPiece.Type()), squareName(m.To)))
		}
	}

	// Pawn Promotions & Passed Pawns
	if m.Promo != chess.None {
		addMotif(motifPromotion)
		explanations = append(explanations, fmt.Sprintf("%s successfully promotes a pawn to a %s.", colorName, pieceName(m.Promo)))
	} else if movingPiece.Type() == chess.Pawn {
		toRank := int(m.To / 8)
		if (moverColor == chess.White && toRank >= 5) || (moverColor == chess.Black && toRank <= 2) {
			addMotif(motifPassedPawn)
			strategic = append(strategic, "Promotion Threat")
			explanations = append(explanations, fmt.Sprintf("%s advances a dangerous passed pawn closer to the promotion rank.", colorName))
		}
	}

	// Piece Fork (Attacking 2+ enemy pieces)
	attackedPieces := countAttackedPieces(bAfter, m.To, moverColor)
	if attackedPieces >= 2 {
		addMotif(motifFork)
		explanations = append(explanations, fmt.Sprintf("The %s on %s creates a double attack against multiple %s targets.", pieceName(movingPiece.Type()), squareName(m.To), enemyName))
	}

	// Central Outpost
	if isCentralOutpost(int(m.To)) {
		strategic = append(strategic, fmt.Sprintf("Control of %s Outpost", squareName(m.To)))
		if movingPiece.Type() == chess.Knight || movingPiece.Type() == chess.Bishop {
			explanations = append(explanations, fmt.Sprintf("Positions the %s on a dominant central outpost square at %s.", pieceName(movingPiece.Type()), squareName(m.To)))
		}
	}

	// Rook on Open File
	if movingPiece.Type() == chess.Rook && isOpenFile(bAfter, int(m.To%8)) {
		strategic = append(strategic, "Open File Control")
		explanations = append(explanations, fmt.Sprintf("%s seizes control of the open %c-file with the rook.", colorName, 'a'+(m.To%8)))
	}

	// Assessment logic using integer bitmask and enum
	const tacticalShotMask = (1 << motifCheckmate) | (1 << motifFavorableExchange) | (1 << motifFork)
	assess := assessGoodMove
	if motifMask&tacticalShotMask != 0 {
		assess = assessTacticalShot
	} else if len(strategic) > 0 {
		assess = assessPositionalImprovement
	}

	if bestMove != "" && bestMove != moveUCI && (strings.HasPrefix(bestMove, "e") || len(bestMove) >= 4) {
		if (scoreCp < -150 && moverColor == chess.White) || (scoreCp > 150 && moverColor == chess.Black) {
			assess = assessInaccuracyMistake
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
		Assessment:        assessmentLabels[assess],
		Summary:           summary,
		TacticalMotifs:    motifs,
		StrategicConcepts: unique(strategic),
		DetailedText:      detailed,
		BestAlternative:   bestMove,
	}, nil
}

func pieceValue(pt chess.PieceType) int {
	if int(pt) < len(pieceValues) {
		return pieceValues[pt]
	}
	return 0
}

func pieceName(pt chess.PieceType) string {
	if int(pt) < len(pieceNames) {
		return pieceNames[pt]
	}
	return "piece"
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

func countAttackedPieces(b *chess.Board, _ chess.Square, friendlyColor chess.Color) int {
	enemyColor := friendlyColor.Other()

	count := 0
	for sq := chess.A1; sq <= chess.H8; sq++ {
		p := b.Squares[sq]
		if p.Color() == enemyColor && p.Type() != chess.None && p.Type() != chess.Pawn {
			if b.IsSquareAttacked(sq, friendlyColor) {
				count++
			}
		}
	}
	return count
}

func unique(slice []string) []string {
	keys := make(map[string]bool, len(slice))
	list := make([]string, 0, len(slice))
	for _, entry := range slice {
		if !keys[entry] {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}
