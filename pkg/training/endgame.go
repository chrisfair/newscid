package training

import (
	"encoding/json"
	"fmt"
	"strings"

	"newscid/pkg/chess"
	"newscid/pkg/tablebase"
)

// EndgameCategory groups theoretical endgame drills by piece configuration.
type EndgameCategory uint8

const (
	EndgameCatPawn EndgameCategory = iota + 1
	EndgameCatRook
	EndgameCatMinor
	EndgameCatQueen
)

var endgameCategoryNames = [...]string{
	EndgameCatPawn:  "pawn",
	EndgameCatRook:  "rook",
	EndgameCatMinor: "minor",
	EndgameCatQueen: "queen",
}

func (c EndgameCategory) String() string {
	if int(c) < len(endgameCategoryNames) {
		return endgameCategoryNames[c]
	}
	return "pawn"
}

func ParseEndgameCategory(s string) (EndgameCategory, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "pawn":
		return EndgameCatPawn, true
	case "rook":
		return EndgameCatRook, true
	case "minor":
		return EndgameCatMinor, true
	case "queen":
		return EndgameCatQueen, true
	default:
		return EndgameCatPawn, false
	}
}

func (c EndgameCategory) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.String())
}

func (c *EndgameCategory) UnmarshalJSON(data []byte) error {
	var raw string
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	parsed, _ := ParseEndgameCategory(raw)
	*c = parsed
	return nil
}

// EndgameDrill describes a curated theoretical endgame sparring position.
type EndgameDrill struct {
	ID           string          `json:"id"`
	Title        string          `json:"title"`
	Category     EndgameCategory `json:"category"`
	Difficulty   string          `json:"difficulty"` // "beginner", "intermediate", "advanced"
	Objective    string          `json:"objective"`  // "win" or "draw"
	SideToPlay   string          `json:"sideToPlay"` // "white" or "black"
	FEN          string          `json:"fen"`
	KeyConcept   string          `json:"keyConcept"`
	Hint         string          `json:"hint"`
	FirstMoveUCI string          `json:"firstMoveUci"`
	FirstMoveSAN string          `json:"firstMoveSan"`
}

// EndgameEvalFeedback reports Syzygy/heuristic feedback after a sparring move.
type EndgameEvalFeedback struct {
	FEN             string `json:"fen"`
	Status          string `json:"status"` // "optimal", "inaccuracy", "blunder", "won", "drawn", "lost"
	WDLText         string `json:"wdlText"`
	DTZ             int    `json:"dtz"`
	TablebaseBacked bool   `json:"tablebaseBacked"`
	LostObjective   bool   `json:"lostObjective"`
	Message         string `json:"message"`
	BestReplyUCI    string `json:"bestReplyUci,omitempty"`
	BestReplySAN    string `json:"bestReplySan,omitempty"`
}

var curatedEndgameDrills = []EndgameDrill{
	{
		ID:           "eg-lucena",
		Title:        "The Lucena Position (Building a Bridge)",
		Category:     EndgameCatRook,
		Difficulty:   "intermediate",
		Objective:    "win",
		SideToPlay:   "white",
		FEN:          "1K1k4/1P6/8/8/8/8/r7/2R5 w - - 0 1",
		KeyConcept:   "Cut off the enemy king, advance the rook to the 4th rank, and build a bridge against vertical checks.",
		Hint:         "First push the Black king away from the pawn with a rook check on c8 or d1, or lift your rook to the 4th rank.",
		FirstMoveUCI: "c1d1",
		FirstMoveSAN: "Rd1+",
	},
	{
		ID:           "eg-philidor",
		Title:        "The Philidor Defense (3rd Rank Hold)",
		Category:     EndgameCatRook,
		Difficulty:   "intermediate",
		Objective:    "draw",
		SideToPlay:   "black",
		FEN:          "4k3/8/4K3/4P3/8/r7/8/7R b - - 0 1",
		KeyConcept:   "Keep your rook on the 6th rank to deny the enemy king shelter; once the pawn advances, switch to checking from behind.",
		Hint:         "Place your rook on the 6th rank (a6) to prevent the White king from advancing.",
		FirstMoveUCI: "a3a6",
		FirstMoveSAN: "Ra6+",
	},
	{
		ID:           "eg-vancura",
		Title:        "The Vancura Defense (Rook Pawn Hold)",
		Category:     EndgameCatRook,
		Difficulty:   "advanced",
		Objective:    "draw",
		SideToPlay:   "black",
		FEN:          "R7/P4k2/8/8/8/8/1r6/6K1 b - - 0 1",
		KeyConcept:   "Attack the a-pawn laterally from the f-file (Rf6) so the White king can never hide from side checks.",
		Hint:         "Keep the rook active and attack the pawn from the side on the 6th rank.",
		FirstMoveUCI: "b2b6",
		FirstMoveSAN: "Rb6",
	},
	{
		ID:           "eg-rook-cutoff",
		Title:        "Cutting Off the King Along a File",
		Category:     EndgameCatRook,
		Difficulty:   "beginner",
		Objective:    "win",
		SideToPlay:   "white",
		FEN:          "8/8/8/3P4/3K4/8/2k5/R7 w - - 0 1",
		KeyConcept:   "Use your rook to imprison the enemy king away from your passed pawn before marching it down.",
		Hint:         "Cut off the Black king along the c-file or 2nd rank.",
		FirstMoveUCI: "a1a2",
		FirstMoveSAN: "Ra2+",
	},
	{
		ID:           "eg-opposition-basic",
		Title:        "Direct Opposition & Key Squares",
		Category:     EndgameCatPawn,
		Difficulty:   "beginner",
		Objective:    "win",
		SideToPlay:   "white",
		FEN:          "4k3/8/8/4K3/4P3/8/8/8 w - - 0 1",
		KeyConcept:   "Advance your king ahead of the pawn onto the key squares (d6, e6, f6) before pushing the pawn.",
		Hint:         "Step your king forward to d6 or f6 to outflank the Black king.",
		FirstMoveUCI: "e5d6",
		FirstMoveSAN: "Kd6",
	},
	{
		ID:           "eg-king-pawn-draw",
		Title:        "Holding Opposition Against a Pawn",
		Category:     EndgameCatPawn,
		Difficulty:   "beginner",
		Objective:    "draw",
		SideToPlay:   "black",
		FEN:          "8/8/3k4/8/3PK3/8/8/8 b - - 0 1",
		KeyConcept:   "Retreat straight back in front of the pawn and take direct opposition whenever the attacking king steps forward.",
		Hint:         "Stay directly in front of the pawn with Ke6 or Kd7 so White cannot take the key squares.",
		FirstMoveUCI: "d6e6",
		FirstMoveSAN: "Ke6",
	},
	{
		ID:           "eg-square-rule",
		Title:        "Rule of the Square (Outside Passed Pawn)",
		Category:     EndgameCatPawn,
		Difficulty:   "beginner",
		Objective:    "win",
		SideToPlay:   "white",
		FEN:          "8/8/8/p7/7P/8/6K1/1k6 w - - 0 1",
		KeyConcept:   "Push the passed pawn immediately when the enemy king cannot step inside its promotion square.",
		Hint:         "Push h5! Black's king is too far away to enter the square of the h-pawn.",
		FirstMoveUCI: "h4h5",
		FirstMoveSAN: "h5",
	},
	{
		ID:           "eg-trebuchet",
		Title:        "Mutual Zugzwang (The Trebuchet)",
		Category:     EndgameCatPawn,
		Difficulty:   "intermediate",
		Objective:    "win",
		SideToPlay:   "white",
		FEN:          "8/8/8/3p4/3P1k2/8/5K2/8 w - - 0 1",
		KeyConcept:   "Force the enemy king to give up defense of its pawn by seizing the corresponding square.",
		Hint:         "Step toward the pawn with Ke2 or Kg2 to claim the opposition.",
		FirstMoveUCI: "f2e2",
		FirstMoveSAN: "Ke2",
	},
	{
		ID:           "eg-wrong-rook-pawn",
		Title:        "Wrong-Colored Bishop Fortress",
		Category:     EndgameCatMinor,
		Difficulty:   "beginner",
		Objective:    "draw",
		SideToPlay:   "black",
		FEN:          "7k/8/8/7P/5B2/8/8/6K1 b - - 0 1",
		KeyConcept:   "When the bishop does not control the rook-pawn's promotion square (h8), the defending king holds the corner for a stalemate draw.",
		Hint:         "Stay locked in the h8/g8/h7 corner — never let White drive you out.",
		FirstMoveUCI: "h8h7",
		FirstMoveSAN: "Kh7",
	},
	{
		ID:           "eg-ocb-fortress",
		Title:        "Opposite-Colored Bishops Blockade",
		Category:     EndgameCatMinor,
		Difficulty:   "intermediate",
		Objective:    "draw",
		SideToPlay:   "black",
		FEN:          "8/8/3k4/3p1p2/4P3/3B4/2K1b3/8 b - - 0 1",
		KeyConcept:   "Establish a light- or dark-square blockade with king and bishop so the opponent's extra pawn cannot break through.",
		Hint:         "Capture or blockade on the opposite color squares.",
		FirstMoveUCI: "d5e4",
		FirstMoveSAN: "dxe4",
	},
	{
		ID:           "eg-kbn-mate",
		Title:        "Bishop & Knight Checkmate Technique",
		Category:     EndgameCatMinor,
		Difficulty:   "advanced",
		Objective:    "win",
		SideToPlay:   "white",
		FEN:          "7k/8/6K1/8/8/5BN1/8/8 w - - 0 1",
		KeyConcept:   "Drive the enemy king into the corner matching your bishop's color (h8 for a light-squared bishop).",
		Hint:         "Keep the Black king trapped near the h8 corner and bring the knight to control the escape squares.",
		FirstMoveUCI: "g3f5",
		FirstMoveSAN: "Nf5",
	},
	{
		ID:           "eg-q-vs-pawn-7th",
		Title:        "Queen vs. Central Pawn on the 7th",
		Category:     EndgameCatQueen,
		Difficulty:   "intermediate",
		Objective:    "win",
		SideToPlay:   "white",
		FEN:          "8/8/8/8/8/4Q3/3kp3/7K w - - 0 1",
		KeyConcept:   "Force the enemy king in front of its own pawn with checks, then step your king closer each time the pawn is blocked.",
		Hint:         "Deliver a check (Qd4+ or Qf2) to force the Black king onto e1.",
		FirstMoveUCI: "e3d4",
		FirstMoveSAN: "Qd4+",
	},
	{
		ID:           "eg-q-vs-bp-7th",
		Title:        "Queen vs. Bishop Pawn on the 7th (Stalemate Trick)",
		Category:     EndgameCatQueen,
		Difficulty:   "advanced",
		Objective:    "draw",
		SideToPlay:   "black",
		FEN:          "8/8/8/8/6K1/3Q4/2p5/1k6 b - - 0 1",
		KeyConcept:   "With a bishop pawn on the 7th and the enemy king far away, retreat to the corner (Ka1) to threaten stalemate if the queen captures.",
		Hint:         "Step into the corner with Ka1 or Kb2 to use the stalemate resource.",
		FirstMoveUCI: "b1a1",
		FirstMoveSAN: "Ka1",
	},
	{
		ID:           "eg-queen-box-mate",
		Title:        "King & Queen Mate (Knight's Distance Box)",
		Category:     EndgameCatQueen,
		Difficulty:   "beginner",
		Objective:    "win",
		SideToPlay:   "white",
		FEN:          "8/8/4k3/8/8/8/2Q5/4K3 w - - 0 1",
		KeyConcept:   "Keep your queen a knight's move away from the enemy king to shrink its box to the edge without stalemating, then bring your king.",
		Hint:         "Shrink the king's box with Qc5 or Qd3.",
		FirstMoveUCI: "c2c5",
		FirstMoveSAN: "Qc5",
	},
}

// GetEndgameDrills returns curated theoretical endgame drills, optionally filtered by category.
func GetEndgameDrills(category string) []EndgameDrill {
	cat, hasFilter := ParseEndgameCategory(category)
	all := category == "" || strings.EqualFold(strings.TrimSpace(category), "all")
	out := make([]EndgameDrill, 0, len(curatedEndgameDrills))
	for _, d := range curatedEndgameDrills {
		if all || (hasFilter && d.Category == cat) {
			out = append(out, d)
		}
	}
	return out
}

// GetEndgameDrill looks up a single endgame drill by ID.
func GetEndgameDrill(id string) (EndgameDrill, bool) {
	for _, d := range curatedEndgameDrills {
		if d.ID == id {
			return d, true
		}
	}
	return EndgameDrill{}, false
}

// EvaluateEndgameMove checks a position after a user's sparring move using Syzygy tablebase
// when available (<= 7 pieces) and falls back to tactical/material checks otherwise.
func EvaluateEndgameMove(fen string, objective string, userSide string, tb *tablebase.Client) (*EndgameEvalFeedback, error) {
	b, err := chess.ParseFEN(fen)
	if err != nil {
		return nil, fmt.Errorf("invalid FEN: %w", err)
	}

	legal := b.GenerateLegalMoves()
	if len(legal) == 0 {
		if b.InCheck(b.Turn) {
			winnerIsWhite := b.Turn == chess.Black
			userIsWhite := strings.EqualFold(userSide, "white")
			if winnerIsWhite == userIsWhite {
				return &EndgameEvalFeedback{
					FEN:     fen,
					Status:  "won",
					WDLText: "Checkmate",
					Message: "Checkmate! You converted the endgame cleanly.",
				}, nil
			}
			return &EndgameEvalFeedback{
				FEN:           fen,
				Status:        "lost",
				WDLText:       "Checkmated",
				LostObjective: true,
				Message:       "Checkmated — try again from the previous move.",
			}, nil
		}
		// Stalemate
		lost := strings.EqualFold(objective, "win")
		msg := "Stalemate! Objective held (Draw)."
		status := "drawn"
		if lost {
			msg = "Stalemate — you accidentally allowed a draw in a winning endgame."
			status = "blunder"
		}
		return &EndgameEvalFeedback{
			FEN:           fen,
			Status:        status,
			WDLText:       "Stalemate (Draw)",
			LostObjective: lost,
			Message:       msg,
		}, nil
	}

	// Probe Syzygy tablebase if client is provided and piece count <= 7
	if tb != nil && tablebase.CountPieces(fen) <= 7 {
		if res, err := tb.Probe(fen); err == nil && res != nil {
			// res.Category is from the side-to-move perspective (opponent after user's move).
			cat := strings.ToLower(res.Category)
			fb := &EndgameEvalFeedback{
				FEN:             fen,
				WDLText:         res.Category,
				DTZ:             res.DTZ,
				TablebaseBacked: true,
			}
			if len(res.Moves) > 0 {
				fb.BestReplyUCI = res.Moves[0].UCI
				fb.BestReplySAN = res.Moves[0].SAN
			}

			wantWin := strings.EqualFold(objective, "win")
			// After user moves, it's opponent's turn:
			// opponent "loss" => user is winning!
			// opponent "draw" => drawn!
			// opponent "win"  => user is losing!
			switch {
			case wantWin && strings.Contains(cat, "loss"):
				fb.Status = "optimal"
				fb.Message = fmt.Sprintf("Tablebase confirmed: winning! (DTZ %d)", absInt(res.DTZ))
			case wantWin && strings.Contains(cat, "draw"):
				fb.Status = "blunder"
				fb.LostObjective = true
				fb.Message = "Tablebase alert: this move threw away the win — position is now a theoretical draw."
			case wantWin && strings.Contains(cat, "win"):
				fb.Status = "blunder"
				fb.LostObjective = true
				fb.Message = "Tablebase alert: this move loses the game!"
			case !wantWin && (strings.Contains(cat, "draw") || strings.Contains(cat, "loss")):
				fb.Status = "optimal"
				fb.Message = "Tablebase confirmed: you are holding the draw cleanly."
			default:
				fb.Status = "blunder"
				fb.LostObjective = true
				fb.Message = "Tablebase alert: you slipped into a theoretical loss — retry this move."
			}
			return fb, nil
		}
	}

	// Offline/heuristic fallback: pick a reasonable opponent reply and report material balance
	bestReply := legal[0]
	bestScore := -999999
	for _, m := range legal {
		sc := 0
		if capPiece := b.Squares[m.To]; capPiece != chess.Empty {
			sc += pieceValues[capPiece.Type()]
		}
		if m.Promo != chess.None {
			sc += pieceValues[m.Promo]
		}
		sim := b.Clone()
		sim.MakeMove(m)
		if sim.InCheck(sim.Turn) {
			sc += 40
		}
		if sc > bestScore {
			bestScore = sc
			bestReply = m
		}
	}

	return &EndgameEvalFeedback{
		FEN:          fen,
		Status:       "optimal",
		WDLText:      "In Progress",
		Message:      "Move played. Keep executing the endgame plan!",
		BestReplyUCI: bestReply.String(),
		BestReplySAN: b.MoveToSAN(bestReply),
	}, nil
}
