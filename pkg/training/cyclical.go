package training

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"newscid/pkg/blundervault"
	"newscid/pkg/chess"
	"newscid/pkg/puzzleminer"
)

// CyclicalMethodAttribution provides transparent, nominative fair-use attribution for cyclical tactical training.
const CyclicalMethodAttribution = "Inspired by the cyclical tactical repetition method popularized by Grandmasters Axel Smith and Hans Tikkanen (commonly known as the Woodpecker training method), combined with Lichess-style tactical motif classification and proficiency analytics. NewScid is an independent project and is not affiliated with or endorsed by Quality Chess or the book's authors."

// DifficultyTierID is a hot-path uint8 identifier for puzzle difficulty ordering (Easy -> Medium -> Hard).
type DifficultyTierID uint8

const (
	TierEasy DifficultyTierID = iota
	TierMedium
	TierHard
)

var difficultyTierLabels = [3]string{
	TierEasy:   "Easy",
	TierMedium: "Medium",
	TierHard:   "Hard",
}

// String returns the canonical label ("Easy", "Medium", "Hard") for the DifficultyTierID.
func (d DifficultyTierID) String() string {
	if int(d) < len(difficultyTierLabels) {
		return difficultyTierLabels[d]
	}
	return "Medium"
}

// ParseDifficultyTierID converts a tier string into its hot-path DifficultyTierID constant.
func ParseDifficultyTierID(label string) DifficultyTierID {
	switch strings.ToLower(strings.TrimSpace(label)) {
	case "easy", "beginner":
		return TierEasy
	case "hard", "advanced":
		return TierHard
	default:
		return TierMedium
	}
}

// ThemeID is a hot-path uint8 identifier for Lichess-compatible tactical and phase themes.
type ThemeID uint8

const (
	// Motifs
	ThemeHangingPiece ThemeID = iota
	ThemeFork
	ThemePin
	ThemeSkewer
	ThemeDiscoveredAttack
	ThemeDoubleCheck
	ThemeTrappedPiece
	ThemeExposedKing
	ThemeAttackingF2F7
	ThemeKingsideAttack
	ThemeQueensideAttack

	// Advanced
	ThemeSacrifice
	ThemeDeflection
	ThemeAttraction
	ThemeClearance
	ThemeIntermezzo
	ThemeQuietMove
	ThemeDefensiveMove
	ThemeAdvancedPawn
	ThemePromotion
	ThemeEnPassant
	ThemeXRayAttack

	// Mates
	ThemeMateIn1
	ThemeMateIn2
	ThemeMateIn3
	ThemeBackRankMate
	ThemeSmotheredMate
	ThemeHookMate
	ThemeArabianMate

	// Phases
	ThemeOpening
	ThemeMiddlegame
	ThemeEndgame
	ThemeRookEndgame
	ThemePawnEndgame
	ThemeMinorPieceEndgame
	ThemeQueenEndgame

	themeCount
)

// ThemeMask is a 64-bit bitmask of ThemeID values for zero-allocation bitwise theme filtering in hot loops.
type ThemeMask uint64

// Has returns true if the given ThemeID bit is set in the mask.
func (m ThemeMask) Has(id ThemeID) bool {
	if id >= themeCount {
		return false
	}
	return (m & (ThemeMask(1) << id)) != 0
}

// Set sets the given ThemeID bit in the mask.
func (m *ThemeMask) Set(id ThemeID) {
	if id < themeCount {
		*m |= (ThemeMask(1) << id)
	}
}

// MatchesFilter returns true if filter is empty (0) or shares at least one set bit with m.
func (m ThemeMask) MatchesFilter(filter ThemeMask) bool {
	return filter == 0 || (m&filter) != 0
}

// LichessThemeInfo describes a single Lichess-compatible tactical theme in the catalog.
type LichessThemeInfo struct {
	ID          ThemeID `json:"id"`
	Key         string  `json:"key"`
	Name        string  `json:"name"`
	Category    string  `json:"category"` // "Motifs", "Advanced", "Mates", "Phases"
	Description string  `json:"description"`
	Actionable  bool    `json:"actionable"`
}

var lichessThemeCatalog = [themeCount]LichessThemeInfo{
	// Motifs
	ThemeHangingPiece: {
		ID:          ThemeHangingPiece,
		Key:         "hangingPiece",
		Name:        "Hanging Piece",
		Category:    "Motifs",
		Description: "Tactics involving an undefended or insufficiently defended piece ready to be captured.",
		Actionable:  true,
	},
	ThemeFork: {
		ID:          ThemeFork,
		Key:         "fork",
		Name:        "Fork / Double Attack",
		Category:    "Motifs",
		Description: "A single piece attacks two or more valuable opponent targets simultaneously.",
		Actionable:  true,
	},
	ThemePin: {
		ID:          ThemePin,
		Key:         "pin",
		Name:        "Pin",
		Category:    "Motifs",
		Description: "A defending piece cannot move without exposing a higher-value piece or the King behind it.",
		Actionable:  true,
	},
	ThemeSkewer: {
		ID:          ThemeSkewer,
		Key:         "skewer",
		Name:        "Skewer",
		Category:    "Motifs",
		Description: "A high-value piece is attacked and forced to move, exposing a piece behind it to capture.",
		Actionable:  true,
	},
	ThemeDiscoveredAttack: {
		ID:          ThemeDiscoveredAttack,
		Key:         "discoveredAttack",
		Name:        "Discovered Attack",
		Category:    "Motifs",
		Description: "Moving one piece unblocks a line of attack for a friendly sliding piece behind it.",
		Actionable:  true,
	},
	ThemeDoubleCheck: {
		ID:          ThemeDoubleCheck,
		Key:         "doubleCheck",
		Name:        "Double Check",
		Category:    "Motifs",
		Description: "Two pieces deliver check simultaneously via a discovered check, forcing the enemy King to move.",
		Actionable:  true,
	},
	ThemeTrappedPiece: {
		ID:          ThemeTrappedPiece,
		Key:         "trappedPiece",
		Name:        "Trapped Piece",
		Category:    "Motifs",
		Description: "An opponent piece has no safe escape squares and can be won.",
		Actionable:  true,
	},
	ThemeExposedKing: {
		ID:          ThemeExposedKing,
		Key:         "exposedKing",
		Name:        "Exposed King",
		Category:    "Motifs",
		Description: "An enemy King stripped of its pawn shield and vulnerable to direct tactical assault.",
		Actionable:  true,
	},
	ThemeAttackingF2F7: {
		ID:          ThemeAttackingF2F7,
		Key:         "attackingF2F7",
		Name:        "Attacking f2 / f7",
		Category:    "Motifs",
		Description: "Targeting the weakest starting square near the opponent's King (f2 or f7).",
		Actionable:  true,
	},
	ThemeKingsideAttack: {
		ID:          ThemeKingsideAttack,
		Key:         "kingsideAttack",
		Name:        "Kingside Attack",
		Category:    "Motifs",
		Description: "A coordinated tactical strike against the opponent's kingside castled position.",
		Actionable:  true,
	},
	ThemeQueensideAttack: {
		ID:          ThemeQueensideAttack,
		Key:         "queensideAttack",
		Name:        "Queenside Attack",
		Category:    "Motifs",
		Description: "A tactical strike aimed at the opponent's queenside King or pieces.",
		Actionable:  true,
	},

	// Advanced
	ThemeSacrifice: {
		ID:          ThemeSacrifice,
		Key:         "sacrifice",
		Name:        "Sacrifice",
		Category:    "Advanced",
		Description: "Intentionally giving up material to achieve checkmate or a decisive material return.",
		Actionable:  true,
	},
	ThemeDeflection: {
		ID:          ThemeDeflection,
		Key:         "deflection",
		Name:        "Deflection",
		Category:    "Advanced",
		Description: "Forcing an enemy defender away from a critical square or piece it was protecting.",
		Actionable:  true,
	},
	ThemeAttraction: {
		ID:          ThemeAttraction,
		Key:         "attraction",
		Name:        "Attraction",
		Category:    "Advanced",
		Description: "Luring an enemy King or Queen onto a vulnerable square where it falls to a tactic.",
		Actionable:  true,
	},
	ThemeClearance: {
		ID:          ThemeClearance,
		Key:         "clearance",
		Name:        "Clearance",
		Category:    "Advanced",
		Description: "Vacating a square, file, rank, or diagonal so another friendly piece can use it.",
		Actionable:  true,
	},
	ThemeIntermezzo: {
		ID:          ThemeIntermezzo,
		Key:         "intermezzo",
		Name:        "Intermezzo (Zwischenzug)",
		Category:    "Advanced",
		Description: "Playing an unexpected intermediate check or threat before making an expected recapture.",
		Actionable:  true,
	},
	ThemeQuietMove: {
		ID:          ThemeQuietMove,
		Key:         "quietMove",
		Name:        "Quiet Move",
		Category:    "Advanced",
		Description: "A subtle non-capture, non-check move that prepares an unstoppable tactical threat.",
		Actionable:  true,
	},
	ThemeDefensiveMove: {
		ID:          ThemeDefensiveMove,
		Key:         "defensiveMove",
		Name:        "Defensive Resource",
		Category:    "Advanced",
		Description: "An accurate defensive move or counter-tactic that neutralizes an opponent's threat.",
		Actionable:  true,
	},
	ThemeAdvancedPawn: {
		ID:          ThemeAdvancedPawn,
		Key:         "advancedPawn",
		Name:        "Advanced Pawn",
		Category:    "Advanced",
		Description: "Exploiting a pawn deep in enemy territory (6th or 7th rank) threatening promotion.",
		Actionable:  true,
	},
	ThemePromotion: {
		ID:          ThemePromotion,
		Key:         "promotion",
		Name:        "Promotion / Underpromotion",
		Category:    "Advanced",
		Description: "Promoting a pawn to a Queen or tactical Knight/Rook to decide the game.",
		Actionable:  true,
	},
	ThemeEnPassant: {
		ID:          ThemeEnPassant,
		Key:         "enPassant",
		Name:        "En Passant",
		Category:    "Advanced",
		Description: "Tactical execution involving the en passant pawn capture rule.",
		Actionable:  true,
	},
	ThemeXRayAttack: {
		ID:          ThemeXRayAttack,
		Key:         "xRayAttack",
		Name:        "X-Ray Attack",
		Category:    "Advanced",
		Description: "A sliding piece exerts tactical influence through an enemy piece along a shared line.",
		Actionable:  true,
	},

	// Mates
	ThemeMateIn1: {
		ID:          ThemeMateIn1,
		Key:         "mateIn1",
		Name:        "Mate in 1",
		Category:    "Mates",
		Description: "Deliver checkmate on the very first move.",
		Actionable:  true,
	},
	ThemeMateIn2: {
		ID:          ThemeMateIn2,
		Key:         "mateIn2",
		Name:        "Mate in 2",
		Category:    "Mates",
		Description: "Force checkmate in two moves against any defense.",
		Actionable:  true,
	},
	ThemeMateIn3: {
		ID:          ThemeMateIn3,
		Key:         "mateIn3",
		Name:        "Mate in 3+",
		Category:    "Mates",
		Description: "Calculate a multi-move forcing mating net of three or more moves.",
		Actionable:  true,
	},
	ThemeBackRankMate: {
		ID:          ThemeBackRankMate,
		Key:         "backRankMate",
		Name:        "Back Rank Mate",
		Category:    "Mates",
		Description: "Checkmate delivered on the 1st or 8th rank where the King is trapped by its own pawns.",
		Actionable:  true,
	},
	ThemeSmotheredMate: {
		ID:          ThemeSmotheredMate,
		Key:         "smotheredMate",
		Name:        "Smothered Mate",
		Category:    "Mates",
		Description: "A Knight checkmates a King that is completely surrounded by its own pieces.",
		Actionable:  true,
	},
	ThemeHookMate: {
		ID:          ThemeHookMate,
		Key:         "hookMate",
		Name:        "Hook Mate",
		Category:    "Mates",
		Description: "Checkmate with a Rook protected by a Knight while an enemy pawn blocks the King's escape.",
		Actionable:  true,
	},
	ThemeArabianMate: {
		ID:          ThemeArabianMate,
		Key:         "arabianMate",
		Name:        "Arabian Mate",
		Category:    "Mates",
		Description: "A Rook and Knight team up to trap the opposing King on a corner square.",
		Actionable:  true,
	},

	// Phases
	ThemeOpening: {
		ID:          ThemeOpening,
		Key:         "opening",
		Name:        "Opening Tactic",
		Category:    "Phases",
		Description: "Tactical punishment occurring during the opening phase.",
		Actionable:  false,
	},
	ThemeMiddlegame: {
		ID:          ThemeMiddlegame,
		Key:         "middlegame",
		Name:        "Middlegame Tactic",
		Category:    "Phases",
		Description: "Complex tactical combat with most pieces still on the board.",
		Actionable:  false,
	},
	ThemeEndgame: {
		ID:          ThemeEndgame,
		Key:         "endgame",
		Name:        "Endgame Tactic",
		Category:    "Phases",
		Description: "Sharp tactical calculation in simplified endgame positions.",
		Actionable:  true,
	},
	ThemeRookEndgame: {
		ID:          ThemeRookEndgame,
		Key:         "rookEndgame",
		Name:        "Rook Endgame",
		Category:    "Phases",
		Description: "Tactical motifs specific to Rook and Pawn endgames.",
		Actionable:  true,
	},
	ThemePawnEndgame: {
		ID:          ThemePawnEndgame,
		Key:         "pawnEndgame",
		Name:        "Pawn Endgame",
		Category:    "Phases",
		Description: "King and Pawn endgame breakthroughs, opposition, and pawn races.",
		Actionable:  true,
	},
	ThemeMinorPieceEndgame: {
		ID:          ThemeMinorPieceEndgame,
		Key:         "minorPieceEndgame",
		Name:        "Minor Piece Endgame",
		Category:    "Phases",
		Description: "Bishop and Knight endgame tactics.",
		Actionable:  true,
	},
	ThemeQueenEndgame: {
		ID:          ThemeQueenEndgame,
		Key:         "queenEndgame",
		Name:        "Queen Endgame",
		Category:    "Phases",
		Description: "Queen and Pawn endgame checks, forks, and cross-checks.",
		Actionable:  true,
	},
}

var themeKeyToID map[string]ThemeID

func init() {
	themeKeyToID = make(map[string]ThemeID, int(themeCount)*2)
	for i := ThemeID(0); i < themeCount; i++ {
		info := lichessThemeCatalog[i]
		themeKeyToID[info.Key] = i
		themeKeyToID[strings.ToLower(info.Key)] = i
	}
}

// GetLichessThemeCatalog returns the full catalog of Lichess-compatible tactical themes.
func GetLichessThemeCatalog() []LichessThemeInfo {
	out := make([]LichessThemeInfo, themeCount)
	copy(out, lichessThemeCatalog[:])
	return out
}

// LookupThemeID resolves a theme key string to its ThemeID and true if found.
func LookupThemeID(key string) (ThemeID, bool) {
	id, ok := themeKeyToID[strings.TrimSpace(key)]
	if ok {
		return id, true
	}
	id, ok = themeKeyToID[strings.ToLower(strings.TrimSpace(key))]
	return id, ok
}

// ThemeMaskFromKeys converts a slice of theme key strings into a ThemeMask bitmask.
func ThemeMaskFromKeys(keys []string) ThemeMask {
	var mask ThemeMask
	for _, k := range keys {
		if id, ok := LookupThemeID(k); ok {
			mask.Set(id)
		}
	}
	return mask
}

// Keys converts a ThemeMask into an ordered slice of canonical Lichess theme keys.
func (m ThemeMask) Keys() []string {
	var out []string
	for id := ThemeID(0); id < themeCount; id++ {
		if m.Has(id) {
			out = append(out, lichessThemeCatalog[id].Key)
		}
	}
	return out
}

// PrimaryThemeFromMask selects the most specific actionable tactical motif from a ThemeMask.
func PrimaryThemeFromMask(m ThemeMask) string {
	// Priority order: specific mates -> advanced motifs -> core motifs -> phases
	priority := []ThemeID{
		ThemeSmotheredMate, ThemeBackRankMate, ThemeArabianMate, ThemeHookMate,
		ThemeDoubleCheck, ThemeDiscoveredAttack, ThemeDeflection, ThemeAttraction,
		ThemeSacrifice, ThemeIntermezzo, ThemeQuietMove, ThemeDefensiveMove,
		ThemeSkewer, ThemePin, ThemeFork, ThemeTrappedPiece,
		ThemePromotion, ThemeEnPassant, ThemeAdvancedPawn, ThemeXRayAttack,
		ThemeClearance, ThemeAttackingF2F7, ThemeExposedKing,
		ThemeKingsideAttack, ThemeQueensideAttack, ThemeHangingPiece,
		ThemeMateIn3, ThemeMateIn2, ThemeMateIn1,
		ThemeRookEndgame, ThemePawnEndgame, ThemeMinorPieceEndgame, ThemeQueenEndgame,
		ThemeEndgame, ThemeOpening, ThemeMiddlegame,
	}
	for _, id := range priority {
		if m.Has(id) {
			return lichessThemeCatalog[id].Key
		}
	}
	return "middlegame"
}

// ClassifyPositionThemes inspects a tactical position and its solution line to tag all matching
// Lichess themes, compute its difficulty tier (Easy/Medium/Hard), and estimate its puzzle rating (1000..2500).
func ClassifyPositionThemes(startFEN string, solutionUCI []string, evalDeltaCp int) ([]string, DifficultyTierID, int) {
	b, err := chess.ParseFEN(startFEN)
	if err != nil {
		return []string{"middlegame"}, TierEasy, 1200
	}

	var mask ThemeMask
	solverColor := b.Turn
	enemyColor := solverColor.Other()

	// Classify Game Phase
	classifyPhaseThemes(b, &mask)

	// Check if solver starts in check (defensive / check escape resource)
	startedInCheck := b.InCheck(solverColor)

	// Check initial pin/trapped/exposed king state
	if isKingExposed(b, enemyColor) {
		mask.Set(ThemeExposedKing)
	}
	if hasTrappedEnemyPiece(b, solverColor) {
		mask.Set(ThemeTrappedPiece)
	}

	solverMovesCount := (len(solutionUCI) + 1) / 2
	if solverMovesCount <= 0 {
		solverMovesCount = 1
	}

	simBoard := b.Clone()
	var prevSolverFrom, prevSolverTo chess.Square = chess.NoSquare, chess.NoSquare

	for idx, uci := range solutionUCI {
		m, parseErr := parseMoveFlexible(simBoard, uci)
		if parseErr != nil {
			break
		}

		isSolverTurn := (idx % 2) == 0
		moverColor := simBoard.Turn
		oppColor := moverColor.Other()
		movingPiece := simBoard.Squares[m.From]
		movingType := movingPiece.Type()
		capturedPiece := simBoard.Squares[m.To]
		epBefore := simBoard.EPSquare

		if isSolverTurn {
			// Check En Passant
			if movingType == chess.Pawn && m.To == epBefore && epBefore != chess.NoSquare {
				mask.Set(ThemeEnPassant)
				capturedPiece = chess.MakePiece(oppColor, chess.Pawn)
			}

			// Check Promotion
			if m.Promo != chess.None {
				mask.Set(ThemePromotion)
			}

			// Check Advanced Pawn (pawn moving onto or already on 6th/7th/8th rank for White, 3rd/2nd/1st for Black)
			if movingType == chess.Pawn {
				toRank := m.To.Rank()
				if (moverColor == chess.White && toRank >= 5) || (moverColor == chess.Black && toRank <= 2) {
					mask.Set(ThemeAdvancedPawn)
				}
			}

			// Check Attacking f2/f7
			if (moverColor == chess.White && m.To == chess.F7) || (moverColor == chess.Black && m.To == chess.F2) {
				mask.Set(ThemeAttackingF2F7)
			}

			// Check Clearance (if second+ solver move uses square or ray vacated by previous solver move)
			if prevSolverFrom != chess.NoSquare && (m.To == prevSolverFrom || passesThroughSquare(m.From, m.To, prevSolverFrom)) {
				mask.Set(ThemeClearance)
			}

			// Check Hanging Piece capture on first move
			if idx == 0 && capturedPiece != chess.Empty {
				capVal := pieceValues[capturedPiece.Type()]
				moverVal := pieceValues[movingType]
				defended := isSquareDefendedBy(simBoard, m.To, oppColor)
				if !defended || capVal >= moverVal+150 {
					mask.Set(ThemeHangingPiece)
				}
			}

			// Record pre-move sliding attacks for discovered attack detection
			enemyKingSqBefore := simBoard.KingSquare(oppColor)
			slidingAttackersBefore := countSlidingAttacksOnTargets(simBoard, moverColor)

			// Execute move
			simBoard.MakeMove(m)

			givesCheck := simBoard.InCheck(oppColor)
			isMate := simBoard.IsCheckmate()

			// Discovered Attack & Double Check detection
			slidingAttackersAfter := countSlidingAttacksOnTargets(simBoard, moverColor)
			if slidingAttackersAfter > slidingAttackersBefore && !isSlidingPieceType(movingType) {
				mask.Set(ThemeDiscoveredAttack)
			} else if enemyKingSqBefore != chess.NoSquare && isRayUnblockedByMove(simBoard, m.From, m.To, moverColor) {
				mask.Set(ThemeDiscoveredAttack)
			}

			if givesCheck {
				checkers := findAttackersOfSquare(simBoard, simBoard.KingSquare(oppColor), moverColor)
				if len(checkers) >= 2 {
					mask.Set(ThemeDoubleCheck)
					mask.Set(ThemeDiscoveredAttack)
				}
			}

			// Fork detection: moved piece on m.To simultaneously attacks >= 2 valuable or undefended enemy targets
			if detectsForkFromSquare(simBoard, m.To, moverColor) {
				mask.Set(ThemeFork)
			}

			// Pin, Skewer, and X-Ray detection
			detectPinSkewerXRay(simBoard, moverColor, &mask)

			// Sacrifice detection: moved piece is placed on a square defended by enemy and is higher value than what it captured
			if !isMate && isSquareDefendedBy(simBoard, m.To, oppColor) {
				moverVal := pieceValues[movingType]
				capVal := pieceValues[capturedPiece.Type()]
				if moverVal > capVal+100 {
					mask.Set(ThemeSacrifice)
				}
			}

			// Quiet Move & Defensive Move detection
			if capturedPiece == chess.Empty && !givesCheck && m.Promo == chess.None {
				if idx == 0 && startedInCheck {
					mask.Set(ThemeDefensiveMove)
				} else if len(solutionUCI) >= 3 {
					mask.Set(ThemeQuietMove)
				} else {
					mask.Set(ThemeDefensiveMove)
				}
			} else if idx == 0 && startedInCheck && !isMate && solverMovesCount == 1 && evalDeltaCp < 350 {
				mask.Set(ThemeDefensiveMove)
			}

			// Intermezzo detection: intermediate check in a multi-move sequence before a capture
			if givesCheck && !isMate && idx == 0 && len(solutionUCI) >= 3 && capturedPiece == chess.Empty {
				if nextUCI := solutionUCI[2]; len(nextUCI) >= 4 {
					mask.Set(ThemeIntermezzo)
				}
			}

			// Kingside / Queenside attack detection when attacking near enemy King
			classifyKingZoneAttack(simBoard, moverColor, m.To, givesCheck, &mask)

			// Checkmate classification
			if isMate {
				switch solverMovesCount {
				case 1:
					mask.Set(ThemeMateIn1)
				case 2:
					mask.Set(ThemeMateIn2)
				default:
					mask.Set(ThemeMateIn3)
				}
				classifySpecificMatePattern(simBoard, moverColor, m, &mask)
			}

			prevSolverFrom = m.From
			prevSolverTo = m.To
		} else {
			// Opponent reply move (odd index)
			oppMovingPiece := simBoard.Squares[m.From]
			oppCaptured := simBoard.Squares[m.To]

			// If opponent captures our just-moved piece of higher value, that confirms a sacrifice!
			if m.To == prevSolverTo && oppCaptured != chess.Empty {
				if pieceValues[oppCaptured.Type()] >= pieceValues[ chess.Knight ] {
					mask.Set(ThemeSacrifice)
				}
				// If the opponent's King or Queen was drawn to prevSolverTo, tag Attraction; otherwise Deflection
				if oppMovingPiece.Type() == chess.King || oppMovingPiece.Type() == chess.Queen {
					mask.Set(ThemeAttraction)
				} else {
					mask.Set(ThemeDeflection)
				}
			} else if idx+1 < len(solutionUCI) {
				// Check if opponent's piece moved away from defending the next solver target square
				nextDest := chess.SquareFromString(solutionUCI[idx+1][2:4])
				if nextDest != chess.NoSquare && pieceAttacksSquare(simBoard, m.From, nextDest) {
					mask.Set(ThemeDeflection)
				}
			}

			simBoard.MakeMove(m)
		}
	}

	// If the tactical liquidation simplified into a specific endgame, tag that endgame phase as well
	classifyPhaseThemes(simBoard, &mask)

	// Ensure at least one actionable motif is present alongside the phase theme
	hasActionable := false
	for id := ThemeID(0); id < ThemeOpening; id++ {
		if mask.Has(id) {
			hasActionable = true
			break
		}
	}
	if !hasActionable {
		if solverMovesCount == 1 {
			mask.Set(ThemeHangingPiece)
		} else {
			mask.Set(ThemeFork)
		}
	}

	rating, tier := computeDifficultyAndRating(mask, solverMovesCount, evalDeltaCp)
	return mask.Keys(), tier, rating
}

func parseMoveFlexible(b *chess.Board, uci string) (chess.Move, error) {
	if m, err := b.ParseUCI(uci); err == nil {
		return m, nil
	}
	if m, err := b.ParseSAN(uci); err == nil {
		return m, nil
	}
	clean := strings.TrimSpace(strings.ToLower(uci))
	if len(clean) >= 4 {
		from := chess.SquareFromString(clean[0:2])
		to := chess.SquareFromString(clean[2:4])
		if from != chess.NoSquare && to != chess.NoSquare && b.Squares[from] != chess.Empty {
			promo := chess.None
			if len(clean) >= 5 {
				switch clean[4] {
				case 'q':
					promo = chess.Queen
				case 'r':
					promo = chess.Rook
				case 'b':
					promo = chess.Bishop
				case 'n':
					promo = chess.Knight
				}
			}
			return chess.Move{
				From:    from,
				To:      to,
				Promo:   promo,
				Capture: b.Squares[to].Type(),
			}, nil
		}
	}
	return chess.Move{}, fmt.Errorf("cannot parse move %s", uci)
}

func classifyPhaseThemes(b *chess.Board, mask *ThemeMask) {
	var queens, rooks, minors, totalNonPawn int
	for sq := chess.A1; sq <= chess.H8; sq++ {
		p := b.Squares[sq]
		if p == chess.Empty {
			continue
		}
		switch p.Type() {
		case chess.Queen:
			queens++
			totalNonPawn++
		case chess.Rook:
			rooks++
			totalNonPawn++
		case chess.Bishop, chess.Knight:
			minors++
			totalNonPawn++
		}
	}

	if totalNonPawn <= 6 {
		mask.Set(ThemeEndgame)
		switch {
		case totalNonPawn == 0:
			mask.Set(ThemePawnEndgame)
		case rooks > 0 && queens == 0 && minors == 0:
			mask.Set(ThemeRookEndgame)
		case minors > 0 && queens == 0 && rooks == 0:
			mask.Set(ThemeMinorPieceEndgame)
		case queens > 0 && rooks == 0 && minors == 0:
			mask.Set(ThemeQueenEndgame)
		}
	} else if b.FullMove <= 11 && totalNonPawn >= 10 {
		mask.Set(ThemeOpening)
	} else {
		mask.Set(ThemeMiddlegame)
	}
}

func isKingExposed(b *chess.Board, kingColor chess.Color) bool {
	ksq := b.KingSquare(kingColor)
	if ksq == chess.NoSquare {
		return false
	}
	r := ksq.Rank()
	f := ksq.File()
	// King out in the middle ranks (ranks 2..5, 0-indexed)
	if r >= 2 && r <= 5 {
		return true
	}
	// Count friendly pawns on adjacent squares in front of or around the king
	friendlyPawn := chess.MakePiece(kingColor, chess.Pawn)
	shieldCount := 0
	for df := -1; df <= 1; df++ {
		for dr := -1; dr <= 1; dr++ {
			sq := chess.MakeSquare(f+df, r+dr)
			if sq != chess.NoSquare && b.Squares[sq] == friendlyPawn {
				shieldCount++
			}
		}
	}
	return shieldCount == 0
}

func hasTrappedEnemyPiece(b *chess.Board, solverColor chess.Color) bool {
	enemy := solverColor.Other()
	for sq := chess.A1; sq <= chess.H8; sq++ {
		p := b.Squares[sq]
		if p == chess.Empty || p.Color() != enemy {
			continue
		}
		pt := p.Type()
		if pt != chess.Knight && pt != chess.Bishop && pt != chess.Rook && pt != chess.Queen {
			continue
		}
		if !b.IsSquareAttacked(sq, solverColor) {
			continue
		}
		// Check if every square this piece can move to is attacked by a lower-value solver piece
		safeExits := 0
		pieceVal := pieceValues[pt]
		for to := chess.A1; to <= chess.H8; to++ {
			if !pieceAttacksSquare(b, sq, to) {
				continue
			}
			destP := b.Squares[to]
			if destP != chess.Empty && destP.Color() == enemy {
				continue
			}
			if !isAttackedByLowerValue(b, to, solverColor, pieceVal) {
				safeExits++
				break
			}
		}
		if safeExits == 0 {
			return true
		}
	}
	return false
}

func isAttackedByLowerValue(b *chess.Board, sq chess.Square, byColor chess.Color, targetVal int) bool {
	attackers := findAttackersOfSquare(b, sq, byColor)
	for _, attSq := range attackers {
		attP := b.Squares[attSq]
		if pieceValues[attP.Type()] < targetVal {
			return true
		}
	}
	return false
}

func isSquareDefendedBy(b *chess.Board, sq chess.Square, byColor chess.Color) bool {
	return len(findAttackersOfSquare(b, sq, byColor)) > 0
}

func isSlidingPieceType(pt chess.PieceType) bool {
	return pt == chess.Bishop || pt == chess.Rook || pt == chess.Queen
}

func passesThroughSquare(from, to, mid chess.Square) bool {
	if from == mid || to == mid {
		return false
	}
	df := to.File() - from.File()
	dr := to.Rank() - from.Rank()
	stepF := signInt(df)
	stepR := signInt(dr)
	if df != 0 && dr != 0 && absInt(df) != absInt(dr) {
		return false
	}
	cf, cr := from.File()+stepF, from.Rank()+stepR
	for cf >= 0 && cf <= 7 && cr >= 0 && cr <= 7 {
		sq := chess.MakeSquare(cf, cr)
		if sq == to {
			break
		}
		if sq == mid {
			return true
		}
		cf += stepF
		cr += stepR
	}
	return false
}

func countSlidingAttacksOnTargets(b *chess.Board, side chess.Color) int {
	enemy := side.Other()
	count := 0
	for sq := chess.A1; sq <= chess.H8; sq++ {
		p := b.Squares[sq]
		if p == chess.Empty || p.Color() != side || !isSlidingPieceType(p.Type()) {
			continue
		}
		for target := chess.A1; target <= chess.H8; target++ {
			tp := b.Squares[target]
			if tp != chess.Empty && tp.Color() == enemy && tp.Type() != chess.Pawn {
				if pieceAttacksSquare(b, sq, target) {
					count++
				}
			}
		}
	}
	return count
}

func isRayUnblockedByMove(bAfter *chess.Board, from, to chess.Square, side chess.Color) bool {
	enemy := side.Other()
	for sq := chess.A1; sq <= chess.H8; sq++ {
		if sq == to {
			continue
		}
		p := bAfter.Squares[sq]
		if p == chess.Empty || p.Color() != side || !isSlidingPieceType(p.Type()) {
			continue
		}
		for target := chess.A1; target <= chess.H8; target++ {
			tp := bAfter.Squares[target]
			if tp != chess.Empty && tp.Color() == enemy && pieceValues[tp.Type()] >= pieceValues[chess.Knight] {
				if passesThroughSquare(sq, target, from) && pieceAttacksSquare(bAfter, sq, target) {
					return true
				}
			}
		}
	}
	return false
}

func detectsForkFromSquare(b *chess.Board, from chess.Square, side chess.Color) bool {
	p := b.Squares[from]
	if p == chess.Empty || p.Color() != side {
		return false
	}
	enemy := side.Other()
	moverVal := pieceValues[p.Type()]
	attackedTargets := 0

	for target := chess.A1; target <= chess.H8; target++ {
		tp := b.Squares[target]
		if tp == chess.Empty || tp.Color() != enemy {
			continue
		}
		if !pieceAttacksSquare(b, from, target) {
			continue
		}
		tVal := pieceValues[tp.Type()]
		if tp.Type() == chess.King || tVal > moverVal || (tVal >= pieceValues[chess.Knight] && !isSquareDefendedBy(b, target, enemy)) {
			attackedTargets++
		}
	}
	return attackedTargets >= 2
}

func detectPinSkewerXRay(b *chess.Board, side chess.Color, mask *ThemeMask) {
	enemy := side.Other()
	dirs := [8][2]int{
		{0, 1}, {0, -1}, {1, 0}, {-1, 0},
		{1, 1}, {1, -1}, {-1, 1}, {-1, -1},
	}

	for sq := chess.A1; sq <= chess.H8; sq++ {
		p := b.Squares[sq]
		if p == chess.Empty || p.Color() != side || !isSlidingPieceType(p.Type()) {
			continue
		}
		pt := p.Type()

		for dIdx, d := range dirs {
			isDiag := dIdx >= 4
			if pt == chess.Rook && isDiag {
				continue
			}
			if pt == chess.Bishop && !isDiag {
				continue
			}

			var firstSq, secondSq chess.Square = chess.NoSquare, chess.NoSquare
			cf, cr := sq.File()+d[0], sq.Rank()+d[1]
			for cf >= 0 && cf <= 7 && cr >= 0 && cr <= 7 {
				cur := chess.MakeSquare(cf, cr)
				if b.Squares[cur] != chess.Empty {
					if firstSq == chess.NoSquare {
						firstSq = cur
					} else {
						secondSq = cur
						break
					}
				}
				cf += d[0]
				cr += d[1]
			}

			if firstSq == chess.NoSquare || secondSq == chess.NoSquare {
				continue
			}
			p1 := b.Squares[firstSq]
			p2 := b.Squares[secondSq]

			if p1.Color() == enemy && p2.Color() == enemy {
				v1 := pieceValues[p1.Type()]
				v2 := pieceValues[p2.Type()]
				// Pin: lower-value enemy piece in front of King or significantly higher-value piece
				if p1.Type() != chess.King && (p2.Type() == chess.King || v2 >= v1+200) {
					mask.Set(ThemePin)
				}
				// Skewer: higher-value enemy piece (King, Queen, Rook) in front of another valuable piece
				if (p1.Type() == chess.King || v1 >= v2) && v2 >= pieceValues[chess.Knight] && v1 >= pieceValues[chess.Rook] {
					mask.Set(ThemeSkewer)
				}
			} else if p1.Color() == enemy && p2.Color() == side {
				// X-Ray: attacking/supporting through an enemy piece along the same line
				if isSlidingPieceType(p2.Type()) || p1.Type() == chess.Queen || p1.Type() == chess.Rook {
					mask.Set(ThemeXRayAttack)
				}
			}
		}
	}
}

func classifyKingZoneAttack(b *chess.Board, side chess.Color, moveTo chess.Square, givesCheck bool, mask *ThemeMask) {
	enemy := side.Other()
	ksq := b.KingSquare(enemy)
	if ksq == chess.NoSquare {
		return
	}
	kf, kr := ksq.File(), ksq.Rank()
	distF := absInt(moveTo.File() - kf)
	distR := absInt(moveTo.Rank() - kr)
	if !givesCheck && (distF > 2 || distR > 2) {
		return
	}
	if kf >= 4 {
		mask.Set(ThemeKingsideAttack)
	} else if kf <= 3 {
		mask.Set(ThemeQueensideAttack)
	}
}

func classifySpecificMatePattern(b *chess.Board, winnerColor chess.Color, matingMove chess.Move, mask *ThemeMask) {
	loserColor := winnerColor.Other()
	ksq := b.KingSquare(loserColor)
	if ksq == chess.NoSquare {
		return
	}
	kf, kr := ksq.File(), ksq.Rank()
	matingPiece := b.Squares[matingMove.To].Type()

	// Smothered Mate: Knight delivers mate and all adjacent squares around the King are occupied by friendly pieces
	if matingPiece == chess.Knight {
		allBlockedByOwn := true
		for df := -1; df <= 1; df++ {
			for dr := -1; dr <= 1; dr++ {
				if df == 0 && dr == 0 {
					continue
				}
				adj := chess.MakeSquare(kf+df, kr+dr)
				if adj == chess.NoSquare {
					continue
				}
				p := b.Squares[adj]
				if p == chess.Empty || p.Color() != loserColor {
					allBlockedByOwn = false
					break
				}
			}
		}
		if allBlockedByOwn {
			mask.Set(ThemeSmotheredMate)
		}
	}

	// Back Rank Mate: King on rank 0 (White) or rank 7 (Black), mated along back rank by Rook/Queen, blocked by own pawns
	backRank := 0
	pawnShieldRank := 1
	if loserColor == chess.Black {
		backRank = 7
		pawnShieldRank = 6
	}
	if kr == backRank && (matingPiece == chess.Rook || matingPiece == chess.Queen) && matingMove.To.Rank() == backRank {
		ownPawnsBlocking := 0
		for df := -1; df <= 1; df++ {
			sq := chess.MakeSquare(kf+df, pawnShieldRank)
			if sq != chess.NoSquare && b.Squares[sq].Color() == loserColor && b.Squares[sq].Type() == chess.Pawn {
				ownPawnsBlocking++
			}
		}
		if ownPawnsBlocking >= 2 {
			mask.Set(ThemeBackRankMate)
		}
	}

	// Arabian Mate & Hook Mate: Rook delivers checkmate Adjacent/Orthogonal to King, defended by a friendly Knight
	if matingPiece == chess.Rook {
		defendedByKnight := false
		for sq := chess.A1; sq <= chess.H8; sq++ {
			p := b.Squares[sq]
			if p.Color() == winnerColor && p.Type() == chess.Knight && pieceAttacksSquare(b, sq, matingMove.To) {
				defendedByKnight = true
				break
			}
		}
		if defendedByKnight {
			isCorner := (kf == 0 || kf == 7) && (kr == 0 || kr == 7)
			if isCorner {
				mask.Set(ThemeArabianMate)
			} else {
				mask.Set(ThemeHookMate)
			}
		}
	}
}

func computeDifficultyAndRating(mask ThemeMask, solverMoves int, evalDeltaCp int) (int, DifficultyTierID) {
	rating := 1150
	switch {
	case solverMoves == 2:
		rating = 1520
	case solverMoves >= 3:
		rating = 1920
	}

	if mask.Has(ThemeMateIn1) || mask.Has(ThemeHangingPiece) {
		rating -= 110
	}
	if mask.Has(ThemeFork) || mask.Has(ThemePin) || mask.Has(ThemeBackRankMate) {
		rating += 55
	}
	if mask.Has(ThemeDiscoveredAttack) || mask.Has(ThemeSkewer) || mask.Has(ThemeDoubleCheck) ||
		mask.Has(ThemeSmotheredMate) || mask.Has(ThemeArabianMate) || mask.Has(ThemeHookMate) || mask.Has(ThemeTrappedPiece) {
		rating += 130
	}
	if mask.Has(ThemeSacrifice) || mask.Has(ThemeDeflection) || mask.Has(ThemeAttraction) ||
		mask.Has(ThemeIntermezzo) || mask.Has(ThemeClearance) || mask.Has(ThemeXRayAttack) {
		rating += 210
	}
	if mask.Has(ThemeQuietMove) || mask.Has(ThemeDefensiveMove) || mask.Has(ThemeMateIn3) {
		rating += 270
	}
	if evalDeltaCp >= 600 && solverMoves >= 2 {
		rating += 60
	}

	if rating < 1000 {
		rating = 1000
	} else if rating > 2500 {
		rating = 2500
	}

	var tier DifficultyTierID
	switch {
	case rating < 1450:
		tier = TierEasy
	case rating < 1850:
		tier = TierMedium
	default:
		tier = TierHard
	}
	return rating, tier
}

func findAttackersOfSquare(b *chess.Board, sq chess.Square, byColor chess.Color) []chess.Square {
	return FindDefenders(b, sq, byColor)
}

func pieceAttacksSquare(b *chess.Board, from, to chess.Square) bool {
	if from == to || from >= 64 || to >= 64 {
		return false
	}
	p := b.Squares[from]
	if p == chess.Empty {
		return false
	}
	for _, sq := range FindDefenders(b, to, p.Color()) {
		if sq == from {
			return true
		}
	}
	return false
}

func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func signInt(x int) int {
	if x < 0 {
		return -1
	}
	if x > 0 {
		return 1
	}
	return 0
}

// CyclicalPuzzle represents a single tactical exercise in a cyclical repetition set.
type CyclicalPuzzle struct {
	ID               string   `json:"id"`
	SourceGameID     int      `json:"sourceGameId,omitempty"`
	White            string   `json:"white"`
	Black            string   `json:"black"`
	Event            string   `json:"event"`
	Date             string   `json:"date"`
	Ply              int      `json:"ply"`
	FEN              string   `json:"fen"`
	Turn             string   `json:"turn"` // "w" or "b"
	SolutionUCI      []string `json:"solutionUci"`
	SolutionSAN      []string `json:"solutionSan"`
	PlayedBlunderUCI string   `json:"playedBlunderUci,omitempty"`
	Themes           []string `json:"themes"`
	PrimaryTheme     string   `json:"primaryTheme"`
	DifficultyTier   string   `json:"difficultyTier"`  // "Easy", "Medium", "Hard"
	DifficultyOrder  int      `json:"difficultyOrder"` // 1..N (sorted Easy -> Hard)
	EstimatedRating  int      `json:"estimatedRating"`
	EvalDeltaCp      int      `json:"evalDeltaCp"`
	Explanation      string   `json:"explanation"`
}

var turnCodeByColor = [2]string{
	chess.White: "w",
	chess.Black: "b",
}

// formatSolutionSAN converts a sequence of UCI moves from startFEN into SAN moves.
func formatSolutionSAN(startFEN string, solutionUCI []string) []string {
	b, err := chess.ParseFEN(startFEN)
	if err != nil {
		return append([]string(nil), solutionUCI...)
	}
	sanList := make([]string, 0, len(solutionUCI))
	for _, uci := range solutionUCI {
		m, err := parseMoveFlexible(b, uci)
		if err != nil {
			sanList = append(sanList, uci)
			continue
		}
		sanList = append(sanList, b.MoveToSAN(m))
		b.MakeMove(m)
	}
	return sanList
}

// SortPuzzlesEasyToHard sorts puzzles strictly from Easy (TierEasy, lowest EstimatedRating)
// -> Medium (TierMedium) -> Hard (TierHard, highest EstimatedRating) and numbers DifficultyOrder 1..N.
func SortPuzzlesEasyToHard(puzzles []CyclicalPuzzle) []CyclicalPuzzle {
	sorted := make([]CyclicalPuzzle, len(puzzles))
	copy(sorted, puzzles)

	sort.SliceStable(sorted, func(i, j int) bool {
		ti := ParseDifficultyTierID(sorted[i].DifficultyTier)
		tj := ParseDifficultyTierID(sorted[j].DifficultyTier)
		if ti != tj {
			return ti < tj
		}
		if sorted[i].EstimatedRating != sorted[j].EstimatedRating {
			return sorted[i].EstimatedRating < sorted[j].EstimatedRating
		}
		return sorted[i].ID < sorted[j].ID
	})

	for i := range sorted {
		sorted[i].DifficultyOrder = i + 1
	}
	return sorted
}

// ConvertBlunderVaultPuzzle converts a personal Blunder Vault flashcard into a classified CyclicalPuzzle.
func ConvertBlunderVaultPuzzle(vp *blundervault.Puzzle) (CyclicalPuzzle, bool) {
	if vp == nil || strings.TrimSpace(vp.FEN) == "" || strings.TrimSpace(vp.BestMove) == "" {
		return CyclicalPuzzle{}, false
	}
	b, err := chess.ParseFEN(vp.FEN)
	if err != nil {
		return CyclicalPuzzle{}, false
	}
	m, err := parseMoveFlexible(b, vp.BestMove)
	if err != nil {
		return CyclicalPuzzle{}, false
	}
	uci := m.String()
	solUCI := []string{uci}
	solSAN := formatSolutionSAN(vp.FEN, solUCI)
	delta := vp.DeltaCp
	if delta <= 0 {
		delta = 300
	}

	themes, tierID, rating := ClassifyPositionThemes(vp.FEN, solUCI, delta)
	mask := ThemeMaskFromKeys(themes)
	primary := PrimaryThemeFromMask(mask)

	white := vp.PlayerName
	black := "Opponent"
	if strings.EqualFold(vp.UserColor, "black") {
		white = "Opponent"
		black = vp.PlayerName
	}

	expl := vp.Explanation
	if expl == "" {
		expl = fmt.Sprintf("Best tactical continuation is %s (%s motif).", solSAN[0], primary)
	}

	return CyclicalPuzzle{
		ID:               "vault-" + vp.ID,
		White:            white,
		Black:            black,
		Event:            vp.Event,
		Date:             vp.Date,
		Ply:              b.FullMove * 2,
		FEN:              vp.FEN,
		Turn:             turnCodeByColor[b.Turn&1],
		SolutionUCI:      solUCI,
		SolutionSAN:      solSAN,
		PlayedBlunderUCI: vp.PlayedMove,
		Themes:           themes,
		PrimaryTheme:     primary,
		DifficultyTier:   tierID.String(),
		EstimatedRating:  rating,
		EvalDeltaCp:      delta,
		Explanation:      expl,
	}, true
}

// ConvertMinedPuzzle converts a puzzleminer.MinedPuzzle into a classified CyclicalPuzzle.
func ConvertMinedPuzzle(mp puzzleminer.MinedPuzzle) (CyclicalPuzzle, bool) {
	if strings.TrimSpace(mp.StartFEN) == "" {
		return CyclicalPuzzle{}, false
	}
	solUCI := mp.Solution
	if len(solUCI) == 0 && mp.BestMoveUCI != "" {
		solUCI = []string{mp.BestMoveUCI}
	}
	if len(solUCI) == 0 {
		return CyclicalPuzzle{}, false
	}
	b, err := chess.ParseFEN(mp.StartFEN)
	if err != nil {
		return CyclicalPuzzle{}, false
	}

	solSAN := formatSolutionSAN(mp.StartFEN, solUCI)
	themes, tierID, rating := ClassifyPositionThemes(mp.StartFEN, solUCI, mp.EvalDeltaCp)
	mask := ThemeMaskFromKeys(themes)
	primary := PrimaryThemeFromMask(mask)

	return CyclicalPuzzle{
		ID:              mp.ID,
		SourceGameID:    mp.GameID,
		White:           mp.White,
		Black:           mp.Black,
		Event:           mp.Event,
		Date:            mp.Date,
		Ply:             mp.Ply,
		FEN:             mp.StartFEN,
		Turn:            turnCodeByColor[b.Turn&1],
		SolutionUCI:     solUCI,
		SolutionSAN:     solSAN,
		Themes:          themes,
		PrimaryTheme:    primary,
		DifficultyTier:  tierID.String(),
		EstimatedRating: rating,
		EvalDeltaCp:     mp.EvalDeltaCp,
		Explanation:     mp.Description,
	}, true
}

var evalCommentRegex = regexp.MustCompile(`(?i)\[%eval\s+([+-]?\d+(?:\.\d+)?|#[+-]?\d+)\]`)

// ExtractPuzzlesFromPlayerGames scans the player's games for missed tactics, annotated blunders/brilliancies,
// variations, checkmates, and tactical turns, supplements with built-in puzzles when needed, and orders Easy -> Hard.
func ExtractPuzzlesFromPlayerGames(games []*chess.Game, playerNames []string, themeFilter []string, maxPuzzles int) []CyclicalPuzzle {
	if maxPuzzles <= 0 {
		maxPuzzles = 24
	}

	filterMask := ThemeMaskFromKeys(themeFilter)
	var extracted []CyclicalPuzzle
	seenFENs := make(map[string]bool)

	var normPlayers []string
	for _, n := range playerNames {
		if clean := strings.ToLower(strings.TrimSpace(n)); clean != "" {
			normPlayers = append(normPlayers, clean)
		}
	}

	// Pass 1: Scan matching player games (or all games if no player filter matches)
	candidateGames := filterGamesByPlayers(games, normPlayers)
	for gIdx, g := range candidateGames {
		if g == nil || g.RootMove == nil {
			continue
		}
		gamePuzzles := extractFromSingleGame(g, gIdx+1, filterMask, seenFENs)
		extracted = append(extracted, gamePuzzles...)
		if len(extracted) >= maxPuzzles {
			break
		}
	}

	// Supplement from built-in multi-theme tactical library if needed
	if len(extracted) < maxPuzzles {
		builtIn := GetBuiltInCyclicalPuzzles()
		for _, bp := range builtIn {
			if len(extracted) >= maxPuzzles {
				break
			}
			if seenFENs[bp.FEN] {
				continue
			}
			pMask := ThemeMaskFromKeys(bp.Themes)
			if !pMask.MatchesFilter(filterMask) {
				continue
			}
			seenFENs[bp.FEN] = true
			extracted = append(extracted, bp)
		}
	}

	if len(extracted) > maxPuzzles {
		extracted = extracted[:maxPuzzles]
	}

	return SortPuzzlesEasyToHard(extracted)
}

func filterGamesByPlayers(games []*chess.Game, normPlayers []string) []*chess.Game {
	if len(normPlayers) == 0 {
		return games
	}
	var matched []*chess.Game
	for _, g := range games {
		if g == nil {
			continue
		}
		w := strings.ToLower(g.Tag("White"))
		b := strings.ToLower(g.Tag("Black"))
		for _, p := range normPlayers {
			if strings.Contains(w, p) || strings.Contains(b, p) {
				matched = append(matched, g)
				break
			}
		}
	}
	if len(matched) > 0 {
		return matched
	}
	return games
}

func extractFromSingleGame(g *chess.Game, defaultGameID int, filterMask ThemeMask, seenFENs map[string]bool) []CyclicalPuzzle {
	var out []CyclicalPuzzle
	b := chess.NewBoard()
	if g.StartFEN != "" && g.StartFEN != chess.StartingFEN {
		if parsed, err := chess.ParseFEN(g.StartFEN); err == nil {
			b = parsed
		}
	}

	gameID := g.ID
	if gameID <= 0 {
		gameID = defaultGameID
	}

	curr := g.RootMove
	ply := 1
	var prevEvalCp *int

	for curr != nil {
		fenBefore := b.FEN()
		turnColor := b.Turn

		if !seenFENs[fenBefore] {
			var solUCI []string
			var blunderUCI string
			evalDelta := 320

			// Check if move has a variation (refutation / better line)
			if len(curr.Variations) > 0 && curr.Variations[0] != nil {
				vNode := curr.Variations[0]
				for vNode != nil && len(solUCI) < 5 {
					if vNode.UCI != "" {
						solUCI = append(solUCI, vNode.UCI)
					} else {
						solUCI = append(solUCI, vNode.Move.String())
					}
					vNode = vNode.Next
				}
				blunderUCI = curr.UCI
				evalDelta = 420
			}

			// Check for [%eval] swing or NAG annotations (?, ??, !, !!)
			curEvalCp := parseEvalCommentCp(curr.Comment, turnColor)
			hasBlunderNAG := hasAnyNAG(curr.NAGs, 2, 4, 6) || strings.Contains(curr.Comment, "?")
			hasBrilliancyNAG := hasAnyNAG(curr.NAGs, 1, 3) || strings.Contains(curr.Comment, "!")

			if prevEvalCp != nil && curEvalCp != nil {
				swing := *prevEvalCp - *curEvalCp
				if swing >= 200 {
					hasBlunderNAG = true
					evalDelta = swing
				}
			}
			if curEvalCp != nil {
				prevEvalCp = curEvalCp
			}

			// Look for a tactical best move in the position before curr.Move
			if len(solUCI) == 0 && (hasBlunderNAG || hasBrilliancyNAG || ply >= 4) {
				bestTactic, tacticDelta, isMissed := findBestTacticalSequence(b, curr)
				if len(bestTactic) > 0 {
					solUCI = bestTactic
					if tacticDelta > evalDelta {
						evalDelta = tacticDelta
					}
					if isMissed {
						blunderUCI = curr.UCI
					}
				}
			}

			if len(solUCI) > 0 {
				themes, tierID, rating := ClassifyPositionThemes(fenBefore, solUCI, evalDelta)
				pMask := ThemeMaskFromKeys(themes)
				if pMask.MatchesFilter(filterMask) {
					seenFENs[fenBefore] = true
					primary := PrimaryThemeFromMask(pMask)
					solSAN := formatSolutionSAN(fenBefore, solUCI)
					expl := fmt.Sprintf("Tactical %s motif from %s vs %s (%s).", primary, g.Tag("White"), g.Tag("Black"), solSAN[0])
					if blunderUCI != "" {
						expl = fmt.Sprintf("Punish/correct game move %s with %s (%s motif).", curr.SAN, solSAN[0], primary)
					}
					out = append(out, CyclicalPuzzle{
						ID:               fmt.Sprintf("game-%d-ply-%d", gameID, ply),
						SourceGameID:     gameID,
						White:            g.Tag("White"),
						Black:            g.Tag("Black"),
						Event:            g.Tag("Event"),
						Date:             g.Tag("Date"),
						Ply:              ply,
						FEN:              fenBefore,
						Turn:             turnCodeByColor[turnColor&1],
						SolutionUCI:      solUCI,
						SolutionSAN:      solSAN,
						PlayedBlunderUCI: blunderUCI,
						Themes:           themes,
						PrimaryTheme:     primary,
						DifficultyTier:   tierID.String(),
						EstimatedRating:  rating,
						EvalDeltaCp:      evalDelta,
						Explanation:      expl,
					})
				}
			}
		}

		b.MakeMove(curr.Move)
		curr = curr.Next
		ply++
	}

	return out
}

func hasAnyNAG(nags []int, targets ...int) bool {
	for _, n := range nags {
		for _, t := range targets {
			if n == t {
				return true
			}
		}
	}
	return false
}

func parseEvalCommentCp(comment string, sideToMove chess.Color) *int {
	if comment == "" {
		return nil
	}
	m := evalCommentRegex.FindStringSubmatch(comment)
	if len(m) < 2 {
		return nil
	}
	raw := m[1]
	var cp int
	if strings.HasPrefix(raw, "#") {
		if strings.Contains(raw, "-") {
			cp = -1000
		} else {
			cp = 1000
		}
	} else if val, err := strconv.ParseFloat(raw, 64); err == nil {
		cp = int(math.Round(val * 100))
	} else {
		return nil
	}
	if sideToMove == chess.Black {
		cp = -cp
	}
	return &cp
}

// findBestTacticalSequence searches legal moves in b for checkmate, winning captures, forks, or promotions,
// or extracts the played tactical continuation if the game itself executed a sharp tactic.
func findBestTacticalSequence(b *chess.Board, curr *chess.MoveNode) ([]string, int, bool) {
	us := b.Turn
	them := us.Other()
	legal := b.GenerateLegalMoves()

	var bestMove chess.Move
	bestScore := 0

	for _, m := range legal {
		score := scoreTacticalCandidate(b, m, us, them)
		if score > bestScore {
			bestScore = score
			bestMove = m
		}
	}

	if bestScore < 280 {
		return nil, 0, false
	}

	playedScore := scoreTacticalCandidate(b, curr.Move, us, them)
	isMissed := !curr.Move.Equals(bestMove) && (bestScore-playedScore >= 200)

	// If the game move itself was the winning tactic and there are follow-up moves in the game, include up to 3 plies if forcing
	if curr.Move.Equals(bestMove) {
		seq := []string{bestMove.String()}
		if curr.Next != nil && curr.Next.Next != nil {
			nextSolver := curr.Next.Next
			bCopy := b.Clone()
			bCopy.MakeMove(curr.Move)
			bCopy.MakeMove(curr.Next.Move)
			if scoreTacticalCandidate(bCopy, nextSolver.Move, us, them) >= 280 {
				seq = append(seq, curr.Next.Move.String(), nextSolver.Move.String())
			}
		}
		return seq, bestScore, false
	}

	return []string{bestMove.String()}, bestScore, isMissed
}

func scoreTacticalCandidate(b *chess.Board, m chess.Move, us, them chess.Color) int {
	movingPiece := b.Squares[m.From]
	if movingPiece == chess.Empty {
		return 0
	}
	capturedPiece := b.Squares[m.To]
	if movingPiece.Type() == chess.Pawn && m.To == b.EPSquare && b.EPSquare != chess.NoSquare {
		capturedPiece = chess.MakePiece(them, chess.Pawn)
	}

	sim := b.Clone()
	sim.MakeMove(m)

	if sim.IsCheckmate() {
		return 1200
	}

	score := 0
	if m.Promo == chess.Queen {
		score += 750
	} else if m.Promo != chess.None {
		score += 450
	}

	if capturedPiece != chess.Empty {
		capVal := pieceValues[capturedPiece.Type()]
		moverVal := pieceValues[movingPiece.Type()]
		if !isSquareDefendedBy(sim, m.To, them) {
			score += capVal
		} else if capVal > moverVal {
			score += capVal - moverVal
		}
	}

	if detectsForkFromSquare(sim, m.To, us) && (!isSquareDefendedBy(sim, m.To, them) || sim.InCheck(them)) {
		score += 340
	}

	return score
}

// builtInPuzzleSeed defines a curated tactical position in the built-in 24-puzzle library.
type builtInPuzzleSeed struct {
	id          string
	white       string
	black       string
	event       string
	date        string
	ply         int
	fen         string
	solutionUCI []string
	blunderUCI  string
	extraThemes []string
	tier        DifficultyTierID
	rating      int
	deltaCp     int
	explanation string
}

// GetBuiltInCyclicalPuzzles returns 24 curated multi-theme tactical puzzles spanning Easy, Medium, and Hard tiers
// and covering the Lichess tactical theme catalog.
func GetBuiltInCyclicalPuzzles() []CyclicalPuzzle {
	seeds := []builtInPuzzleSeed{
		// --- EASY (8 Puzzles: Mate in 1, Back Rank, Smothered, Arabian, Hook, Hanging Piece, Knight Fork, Promotion) ---
		{
			id:          "cyc-builtin-01",
			white:       "Club Player",
			black:       "Tactical Bot",
			event:       "Cyclical Pattern Core",
			date:        "2025.01.01",
			ply:         35,
			fen:         "6k1/5ppp/8/8/8/8/5PPP/4R1K1 w - - 0 18",
			solutionUCI: []string{"e1e8"},
			blunderUCI:  "e1e2",
			extraThemes: []string{"backRankMate", "mateIn1", "endgame", "rookEndgame"},
			tier:        TierEasy,
			rating:      1050,
			deltaCp:     1000,
			explanation: "Re8# exploits the unguarded 8th rank where Black's own f7/g7/h7 pawns trap the King.",
		},
		{
			id:          "cyc-builtin-02",
			white:       "Morphy Study",
			black:       "Consultants",
			event:       "Smothered Pattern",
			date:        "2025.01.02",
			ply:         29,
			fen:         "6rk/6pp/7N/8/8/8/5PPP/6K1 w - - 0 15",
			solutionUCI: []string{"h6f7"},
			blunderUCI:  "g1f1",
			extraThemes: []string{"smotheredMate", "mateIn1", "endgame"},
			tier:        TierEasy,
			rating:      1100,
			deltaCp:     1000,
			explanation: "Nf7# delivers a classic smothered checkmate: the Black King on h8 is boxed in by Rg8, g7, and h7.",
		},
		{
			id:          "cyc-builtin-03",
			white:       "Al-Adli Pattern",
			black:       "Defender",
			event:       "Arabian Corner Net",
			date:        "2025.01.03",
			ply:         41,
			fen:         "7k/2R5/5N2/8/8/8/6PP/6K1 w - - 0 21",
			solutionUCI: []string{"c7h7"},
			blunderUCI:  "c7c8",
			extraThemes: []string{"arabianMate", "mateIn1", "endgame"},
			tier:        TierEasy,
			rating:      1140,
			deltaCp:     1000,
			explanation: "Rh7# is the Arabian Mate: the Knight on f6 protects the Rook on h7 while controlling g8.",
		},
		{
			id:          "cyc-builtin-04",
			white:       "Hook Pattern",
			black:       "Defender",
			event:       "Hook Mate Drill",
			date:        "2025.01.04",
			ply:         39,
			fen:         "5k2/4ppp1/4N3/8/8/8/5PPP/3R2K1 w - - 0 20",
			solutionUCI: []string{"d1d8"},
			blunderUCI:  "d1d7",
			extraThemes: []string{"hookMate", "kingsideAttack", "endgame"},
			tier:        TierEasy,
			rating:      1180,
			deltaCp:     1000,
			explanation: "Rd8# coordinates with the Knight on e6 and Black's own pawns to form a Hook Mate.",
		},
		{
			id:          "cyc-builtin-05",
			white:       "Italian Opening",
			black:       "Amateur",
			event:       "Attacking f7 Drill",
			date:        "2025.01.05",
			ply:         11,
			fen:         "r1bqk1nr/pppp1ppp/2n5/2b1p1N1/2B1P3/8/PPPP1PPP/RNBQK2R w KQkq - 4 6",
			solutionUCI: []string{"c4f7", "e8f8", "d1f3"},
			blunderUCI:  "d2d3",
			extraThemes: []string{"attackingF2F7", "exposedKing", "opening"},
			tier:        TierEasy,
			rating:      1220,
			deltaCp:     450,
			explanation: "Bxf7+! destroys Black's castling rights and exposes the King on f8.",
		},
		{
			id:          "cyc-builtin-06",
			white:       "Open Tactical",
			black:       "Loose Bishop",
			event:       "Hanging Piece Audit",
			date:        "2025.01.06",
			ply:         17,
			fen:         "r2qkbnr/ppp2ppp/2np4/4p3/2B1P1b1/5N2/PPPP1PPP/RNBQ1RK1 w kq - 2 6",
			solutionUCI: []string{"c4f7", "e8f7", "f3g5", "f7e8", "d1g4"},
			blunderUCI:  "h2h3",
			extraThemes: []string{"hangingPiece", "attraction", "discoveredAttack", "attackingF2F7"},
			tier:        TierEasy,
			rating:      1280,
			deltaCp:     360,
			explanation: "Bxf7+! Kxf7 Ng5+ followed by Qxg4 wins the undefended Bishop on g4 with an extra pawn.",
		},
		{
			id:          "cyc-builtin-07",
			white:       "Royal Fork",
			black:       "Center King",
			event:       "Knight Fork Drill",
			date:        "2025.01.07",
			ply:         23,
			fen:         "r3k2r/ppp2ppp/8/3N4/1q6/8/PPP2PPP/R2Q2K1 w kq - 0 12",
			solutionUCI: []string{"d5c7", "e8f8", "c7a8"},
			blunderUCI:  "d1e2",
			extraThemes: []string{"fork", "queensideAttack", "middlegame"},
			tier:        TierEasy,
			rating:      1320,
			deltaCp:     520,
			explanation: "Nxc7+ forks the Black King on e8, Queen on b4, and Rook on a8.",
		},
		{
			id:          "cyc-builtin-08",
			white:       "7th Rank Sprint",
			black:       "Late Rook",
			event:       "Advanced Pawn & Promotion",
			date:        "2025.01.08",
			ply:         55,
			fen:         "8/1P4pk/8/8/8/8/5PPP/6K1 w - - 0 28",
			solutionUCI: []string{"b7b8q"},
			blunderUCI:  "g1f1",
			extraThemes: []string{"promotion", "advancedPawn", "endgame", "pawnEndgame"},
			tier:        TierEasy,
			rating:      1380,
			deltaCp:     800,
			explanation: "b8=Q promotes the unstoppable 7th-rank passed pawn immediately.",
		},

		// --- MEDIUM (8 Puzzles: Pin, Skewer, Discovered Attack, Double Check, Trapped Piece, En Passant, X-Ray, Mate in 2) ---
		{
			id:          "cyc-builtin-09",
			white:       "Absolute Pin",
			black:       "Pinned Knight",
			event:       "Pin Exploitation",
			date:        "2025.01.09",
			ply:         25,
			fen:         "r3k2r/ppp2ppp/2n5/4P3/1b6/2N5/PPP2PPP/R1B1K2R w KQkq - 0 13",
			solutionUCI: []string{"c1d2", "e8g8", "e1c1"},
			blunderUCI:  "a2a3",
			extraThemes: []string{"pin", "defensiveMove", "middlegame"},
			tier:        TierMedium,
			rating:      1480,
			deltaCp:     320,
			explanation: "Bd2 breaks the pin on the c3 Knight while preparing queenside castling.",
		},
		{
			id:          "cyc-builtin-10",
			white:       "Diagonal Skewer",
			black:       "Long Diagonal",
			event:       "Skewer Motif",
			date:        "2025.01.10",
			ply:         45,
			fen:         "q7/8/2k5/8/4B3/8/6PP/6K1 w - - 0 23",
			solutionUCI: []string{"e4f3", "c6c5", "f3a8"},
			blunderUCI:  "g1f2",
			extraThemes: []string{"skewer", "endgame", "minorPieceEndgame"},
			tier:        TierMedium,
			rating:      1520,
			deltaCp:     900,
			explanation: "Bf3+ skewers the Black King on c6 and the undefended Queen on a8 along the h1-a8 diagonal.",
		},
		{
			id:          "cyc-builtin-11",
			white:       "Battery Unmask",
			black:       "Unprotected Queen",
			event:       "Discovered Attack",
			date:        "2025.01.11",
			ply:         27,
			fen:         "3qk2r/ppp2ppp/8/3B4/8/8/PPP2PPP/3Q1RK1 w k - 0 14",
			solutionUCI: []string{"d5f7", "e8e7", "d1d8"},
			blunderUCI:  "f1e1",
			extraThemes: []string{"discoveredAttack", "attackingF2F7", "deflection", "middlegame"},
			tier:        TierMedium,
			rating:      1560,
			deltaCp:     850,
			explanation: "Bxf7+! delivers a discovered attack that unmasks the d-file battery against Black's Queen on d8.",
		},
		{
			id:          "cyc-builtin-12",
			white:       "Reti Idea",
			black:       "Tartakower",
			event:       "Double Check Mate in 2",
			date:        "2025.01.12",
			ply:         21,
			fen:         "2kr3r/ppp2ppp/2p5/8/8/1B6/PPP2PPP/3R2K1 w - - 0 11",
			solutionUCI: []string{"b3e6", "c8b8", "d1d8"},
			blunderUCI:  "d1d8",
			extraThemes: []string{"discoveredAttack", "queensideAttack", "backRankMate", "mateIn2"},
			tier:        TierMedium,
			rating:      1620,
			deltaCp:     1000,
			explanation: "Be6+ forces Kb8, after which Rd8# executes a back-rank checkmate.",
		},
		{
			id:          "cyc-builtin-13",
			white:       "Noah's Ark",
			black:       "Trapped Bishop",
			event:       "Trapped Piece Motif",
			date:        "2025.01.13",
			ply:         19,
			fen:         "r1bqk1nr/1ppp1ppp/p1n5/4p3/Bb2P3/5N2/PPPP1PPP/RNBQK2R w KQkq - 0 5",
			solutionUCI: []string{"c2c3", "b4a5", "b2b4"},
			blunderUCI:  "e1g1",
			extraThemes: []string{"trappedPiece", "queensideAttack", "opening"},
			tier:        TierMedium,
			rating:      1680,
			deltaCp:     330,
			explanation: "c3 followed by b4 traps Black's Bishop on the queenside wing.",
		},
		{
			id:          "cyc-builtin-14",
			white:       "En Passant Strike",
			black:       "King in Center",
			event:       "En Passant Discovery",
			date:        "2025.01.14",
			ply:         31,
			fen:         "4k3/ppp3pp/8/4pP2/8/8/PPP3PP/4R1K1 w - e6 0 16",
			solutionUCI: []string{"f5e6", "e8e7", "e1e5"},
			blunderUCI:  "f5f6",
			extraThemes: []string{"enPassant", "discoveredAttack", "endgame", "rookEndgame"},
			tier:        TierMedium,
			rating:      1720,
			deltaCp:     350,
			explanation: "fxe6 e.p.! opens the e-file for the Rook on e1 and creates a dangerous advanced e6 pawn.",
		},
		{
			id:          "cyc-builtin-15",
			white:       "X-Ray Battery",
			black:       "Back Rank Defender",
			event:       "X-Ray Tactical Strike",
			date:        "2025.01.15",
			ply:         37,
			fen:         "3r2k1/5ppp/8/8/8/8/3Q1PPP/3R2K1 w - - 0 19",
			solutionUCI: []string{"d2d8", "d8d8", "d1d8"},
			blunderUCI:  "d2e2",
			extraThemes: []string{"xRayAttack", "backRankMate", "sacrifice", "mateIn2"},
			tier:        TierMedium,
			rating:      1760,
			deltaCp:     1000,
			explanation: "Qxd8+! Rxd8 Rxd8# uses an X-Ray battery along the d-file to force back-rank mate.",
		},
		{
			id:          "cyc-builtin-16",
			white:       "Double Check",
			black:       "Royal Trap",
			event:       "Double Check Finisher",
			date:        "2025.01.16",
			ply:         33,
			fen:         "4k3/3n1ppp/4N3/8/8/8/5PPP/4R1K1 w - - 0 17",
			solutionUCI: []string{"e6c7", "e8f8", "e1e8"},
			blunderUCI:  "e6g7",
			extraThemes: []string{"doubleCheck", "discoveredAttack", "mateIn2", "endgame"},
			tier:        TierMedium,
			rating:      1810,
			deltaCp:     1000,
			explanation: "Nc7+!! delivers a devastating double check (Knight on c7 + Rook on e1), forcing Kf8 Re8#.",
		},

		// --- HARD (8 Puzzles: Sacrifice, Deflection, Attraction, Clearance, Intermezzo, Quiet Move, Mate in 3, Endgame Breakthrough) ---
		{
			id:          "cyc-builtin-17",
			white:       "Opera Game",
			black:       "Duke & Count",
			event:       "Queen Sacrifice & Deflection",
			date:        "2025.01.17",
			ply:         31,
			fen:         "4kb1r/p2n1ppp/4q3/4p1B1/4P3/1Q6/PPP2PPP/2KR4 w k - 0 16",
			solutionUCI: []string{"b3b8", "d7b8", "d1d8"},
			blunderUCI:  "b3e6",
			extraThemes: []string{"sacrifice", "deflection", "backRankMate", "mateIn2"},
			tier:        TierHard,
			rating:      1900,
			deltaCp:     1000,
			explanation: "Qb8+!! Nxb8 Rd8# sacrifices the Queen to deflect the d7 Knight from guarding d8.",
		},
		{
			id:          "cyc-builtin-18",
			white:       "Magnetic King",
			black:       "Exposed Monarch",
			event:       "Attraction & Royal Fork",
			date:        "2025.01.18",
			ply:         35,
			fen:         "3q2k1/5ppp/8/8/8/5N2/5PPP/3R2K1 w - - 0 18",
			solutionUCI: []string{"d1d8", "g8f8", "f3e5"},
			blunderUCI:  "h2h3",
			extraThemes: []string{"attraction", "backRankMate", "endgame"},
			tier:        TierHard,
			rating:      1960,
			deltaCp:     900,
			explanation: "Rxd8+ wins the unguarded Queen and traps Black on the back rank.",
		},
		{
			id:          "cyc-builtin-19",
			white:       "Square Vacator",
			black:       "Fortress",
			event:       "Clearance Sacrifice",
			date:        "2025.01.19",
			ply:         41,
			fen:         "6k1/5ppp/8/8/8/5R2/4QPPP/6K1 w - - 0 21",
			solutionUCI: []string{"f3f7", "g8f7", "e2e8", "f7f6", "e8f8"},
			blunderUCI:  "e2e8",
			extraThemes: []string{"clearance", "sacrifice", "attraction", "kingsideAttack"},
			tier:        TierHard,
			rating:      2040,
			deltaCp:     650,
			explanation: "Rxf7+! clears the e-file/f-file coordination and attracts the Black King into the open.",
		},
		{
			id:          "cyc-builtin-20",
			white:       "Zwischenzug Master",
			black:       "Recapture Expecter",
			event:       "Intermezzo Check",
			date:        "2025.01.20",
			ply:         33,
			fen:         "3r2k1/5ppp/8/3q4/8/8/5PPP/2QR2K1 w - - 0 17",
			solutionUCI: []string{"d1d5", "d8d5", "c1c8", "d5d8", "c8d8"},
			blunderUCI:  "c1c2",
			extraThemes: []string{"intermezzo", "backRankMate", "deflection", "mateIn3"},
			tier:        TierHard,
			rating:      2110,
			deltaCp:     1000,
			explanation: "Rxd5! Rxd5 Qc8+ Rd8 Qxd8# exploits the back-rank weakness after liquidating the Queen.",
		},
		{
			id:          "cyc-builtin-21",
			white:       "Silent Assassin",
			black:       "Pinned Defense",
			event:       "Quiet Move Preparation",
			date:        "2025.01.21",
			ply:         39,
			fen:         "6k1/5p1p/6p1/8/8/5N2/4QPPP/6K1 w - - 0 20",
			solutionUCI: []string{"e2e5", "h7h5", "e5e8", "g8g7", "f3g5"},
			blunderUCI:  "e2e7",
			extraThemes: []string{"quietMove", "kingsideAttack", "endgame"},
			tier:        TierHard,
			rating:      2180,
			deltaCp:     580,
			explanation: "Qe5! is a quiet centralizing move that prepares invasion on the 8th rank and kingside dark squares.",
		},
		{
			id:          "cyc-builtin-22",
			white:       "Cold Defense",
			black:       "Counter-Attacker",
			event:       "Defensive Resource",
			date:        "2025.01.22",
			ply:         43,
			fen:         "6k1/5ppp/8/8/8/8/4rPPP/3R2K1 w - - 0 22",
			solutionUCI: []string{"d1d8", "e2e8", "d8e8"},
			blunderUCI:  "g1f1",
			extraThemes: []string{"defensiveMove", "backRankMate", "mateIn2", "rookEndgame"},
			tier:        TierHard,
			rating:      2240,
			deltaCp:     1000,
			explanation: "Rd8+! Re8 Rxe8# turns tables immediately with a counter-mating attack.",
		},
		{
			id:          "cyc-builtin-23",
			white:       "Corridor Net",
			black:       "Cornered King",
			event:       "Mate in 3 Calculation",
			date:        "2025.01.23",
			ply:         47,
			fen:         "2q1r1k1/1p3ppp/8/8/8/8/2Q2PPP/3R1RK1 w - - 0 24",
			solutionUCI: []string{"c2c8", "e8c8", "d1d8", "c8d8", "f1d8"},
			blunderUCI:  "c2b3",
			extraThemes: []string{"mateIn3", "backRankMate", "sacrifice", "deflection"},
			tier:        TierHard,
			rating:      2320,
			deltaCp:     1000,
			explanation: "Qxc8! Rxc8 Rd8+! Rxd8 Rxd8# is a forcing 3-move corridor deflection mate.",
		},
		{
			id:          "cyc-builtin-24",
			white:       "Queen Endgame Maestro",
			black:       "Cross-Check Study",
			event:       "Queen Endgame Precision",
			date:        "2025.01.24",
			ply:         59,
			fen:         "6k1/5qp1/7p/8/8/6PP/5PK1/3Q4 w - - 0 30",
			solutionUCI: []string{"d1d8", "f7f8", "d8d5", "g8h8", "d5f3"},
			blunderUCI:  "d1d2",
			extraThemes: []string{"queenEndgame", "endgame", "skewer", "quietMove"},
			tier:        TierHard,
			rating:      2410,
			deltaCp:     620,
			explanation: "Qd8+ Qf8 Qd5+ Kh8 Qf3 forces a winning Queen trade into a decisive endgame.",
		},
	}

	puzzles := make([]CyclicalPuzzle, 0, len(seeds))
	for _, s := range seeds {
		b, err := chess.ParseFEN(s.fen)
		turnStr := "w"
		if err == nil {
			turnStr = turnCodeByColor[b.Turn&1]
		}
		autoThemes, _, _ := ClassifyPositionThemes(s.fen, s.solutionUCI, s.deltaCp)
		mask := ThemeMaskFromKeys(autoThemes)
		for _, et := range s.extraThemes {
			if id, ok := LookupThemeID(et); ok {
				mask.Set(id)
			}
		}
		themes := mask.Keys()
		primary := s.extraThemes[0]
		san := formatSolutionSAN(s.fen, s.solutionUCI)

		puzzles = append(puzzles, CyclicalPuzzle{
			ID:               s.id,
			White:            s.white,
			Black:            s.black,
			Event:            s.event,
			Date:             s.date,
			Ply:              s.ply,
			FEN:              s.fen,
			Turn:             turnStr,
			SolutionUCI:      s.solutionUCI,
			SolutionSAN:      san,
			PlayedBlunderUCI: s.blunderUCI,
			Themes:           themes,
			PrimaryTheme:     primary,
			DifficultyTier:   s.tier.String(),
			EstimatedRating:  s.rating,
			EvalDeltaCp:      s.deltaCp,
			Explanation:      s.explanation,
		})
	}

	return SortPuzzlesEasyToHard(puzzles)
}

// PuzzleAttemptResult records the user's performance on one puzzle within a cycle.
type PuzzleAttemptResult struct {
	PuzzleID       string `json:"puzzleId"`
	SolvedFirstTry bool   `json:"solvedFirstTry"`
	TimeTakenSec   int    `json:"timeTakenSec"`
	AttemptedAt    int64  `json:"attemptedAt"`
}

// ActiveCycleProgress tracks live state within the current repetition cycle.
type ActiveCycleProgress struct {
	CycleNumber     int                            `json:"cycleNumber"`
	CurrentIndex    int                            `json:"currentIndex"`
	ElapsedSec      int                            `json:"elapsedSec"`
	FirstTryCorrect int                            `json:"firstTryCorrect"`
	TotalAttempted  int                            `json:"totalAttempted"`
	Results         map[string]PuzzleAttemptResult `json:"results"`
}

// CycleRun summarizes a completed repetition pass (Cycles 1..7) over a CycleSet.
type CycleRun struct {
	CycleNumber       int     `json:"cycleNumber"`
	CompletedAt       string  `json:"completedAt"`
	TotalPuzzles      int     `json:"totalPuzzles"`
	FirstTryCorrect   int     `json:"firstTryCorrect"`
	TotalSolved       int     `json:"totalSolved"`
	AccuracyPct       float64 `json:"accuracyPct"`
	TotalTimeSec      int     `json:"totalTimeSec"`
	TargetTimeSec     int     `json:"targetTimeSec"`
	AvgSecPerPuzzle   float64 `json:"avgSecPerPuzzle"`
	SpeedupVsCycle1   float64 `json:"speedupVsCycle1"`
	EasyAccuracyPct   float64 `json:"easyAccuracyPct"`
	MediumAccuracyPct float64 `json:"mediumAccuracyPct"`
	HardAccuracyPct   float64 `json:"hardAccuracyPct"`
}

// CycleSet represents a fixed set of tactical puzzles solved repeatedly across diminishing time cycles.
type CycleSet struct {
	ID             string               `json:"id"`
	Name           string               `json:"name"`
	PlayerName     string               `json:"playerName"`
	ThemeFilter    []string             `json:"themeFilter"`
	CreatedAt      int64                `json:"createdAt"`
	CurrentCycle   int                  `json:"currentCycle"` // Starts at 1, up to 7
	TargetDays     int                  `json:"targetDays"`   // Halves each cycle (e.g. 28 -> 14 -> 7 -> 4 -> 2 -> 1 -> 1)
	Puzzles        []CyclicalPuzzle     `json:"puzzles"`
	Cycles         []CycleRun           `json:"cycles"`
	ActiveProgress *ActiveCycleProgress `json:"activeProgress"`
}

// ThemePerformanceStat models Lichess's PuzzleDashboard.scala proficiency metrics per tactical theme.
type ThemePerformanceStat struct {
	ThemeKey          string  `json:"themeKey"`
	ThemeName         string  `json:"themeName"`
	Category          string  `json:"category"`
	Attempts          int     `json:"attempts"`
	FirstWins         int     `json:"firstWins"`
	Fixed             int     `json:"fixed"`
	Failed            int     `json:"failed"`
	AccuracyPct       float64 `json:"accuracyPct"`
	AvgTimeSec        float64 `json:"avgTimeSec"`
	RatingAvg         int     `json:"ratingAvg"`
	PerformanceRating int     `json:"performanceRating"`
	HistoryRatings    []int   `json:"historyRatings"`

	// Internal accumulators persisted in JSON for exact incremental updates
	TotalTimeSec   int `json:"totalTimeSec"`
	TotalRatingSum int `json:"totalRatingSum"`
}

// CyclicalDashboard aggregates sets, Lichess radar chart axes, strengths/weaknesses, and improvement history.
type CyclicalDashboard struct {
	Sets                    []*CycleSet            `json:"sets"`
	ThemeStats              []ThemePerformanceStat `json:"themeStats"`
	RadarAxes               []ThemePerformanceStat `json:"radarAxes"`
	StrongThemes            []ThemePerformanceStat `json:"strongThemes"`
	WeakThemes              []ThemePerformanceStat `json:"weakThemes"`
	GlobalPerformanceRating int                    `json:"globalPerformanceRating"`
	TotalPuzzlesAttempted   int                    `json:"totalPuzzlesAttempted"`
	TotalCyclesCompleted    int                    `json:"totalCyclesCompleted"`
	ImprovementSummary      string                 `json:"improvementSummary"`
	MethodAttribution       string                 `json:"methodAttribution"`
}

type persistedCyclicalState struct {
	Sets       []*CycleSet                     `json:"sets"`
	ThemeStats map[string]ThemePerformanceStat `json:"themeStats"`
}

// CyclicalManager persists and orchestrates Cyclical Tactical Drill sets and Lichess theme proficiency stats.
type CyclicalManager struct {
	mu         sync.RWMutex
	filePath   string
	sets       []*CycleSet
	themeStats map[string]ThemePerformanceStat
}

// NewCyclicalManager initializes a CyclicalManager backed by filePath (defaults to ~/.config/newscid/cyclical_drill.json).
func NewCyclicalManager(filePath string) (*CyclicalManager, error) {
	if strings.TrimSpace(filePath) == "" {
		cfgDir, err := os.UserConfigDir()
		if err != nil {
			cfgDir = os.TempDir()
		}
		filePath = filepath.Join(cfgDir, "newscid", "cyclical_drill.json")
	}

	cm := &CyclicalManager{
		filePath:   filePath,
		sets:       make([]*CycleSet, 0),
		themeStats: make(map[string]ThemePerformanceStat),
	}

	_ = cm.load()
	return cm, nil
}

func (cm *CyclicalManager) load() error {
	data, err := os.ReadFile(cm.filePath)
	if err != nil {
		return err
	}
	var state persistedCyclicalState
	if err := json.Unmarshal(data, &state); err != nil {
		return err
	}
	if state.Sets != nil {
		cm.sets = state.Sets
	}
	if state.ThemeStats != nil {
		cm.themeStats = state.ThemeStats
	}
	return nil
}

func (cm *CyclicalManager) saveLocked() error {
	if err := os.MkdirAll(filepath.Dir(cm.filePath), 0o755); err != nil {
		return err
	}
	state := persistedCyclicalState{
		Sets:       cm.sets,
		ThemeStats: cm.themeStats,
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(cm.filePath, data, 0o644)
}

// CreateCycleSet builds a new cyclical drill set from the player's Blunder Vault and Game History
// (supplemented by built-in puzzles if needed), sorted strictly Easy -> Medium -> Hard.
func (cm *CyclicalManager) CreateCycleSet(
	name, playerName string,
	games []*chess.Game,
	vaultPuzzles []*blundervault.Puzzle,
	themeFilter []string,
	setSize int,
) (*CycleSet, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if setSize <= 0 {
		setSize = 24
	}
	cleanName := strings.TrimSpace(name)
	if cleanName == "" {
		if strings.TrimSpace(playerName) != "" {
			cleanName = fmt.Sprintf("%s's Cyclical Drill Set", strings.TrimSpace(playerName))
		} else {
			cleanName = fmt.Sprintf("Cyclical Tactical Set #%d", len(cm.sets)+1)
		}
	}

	filterMask := ThemeMaskFromKeys(themeFilter)
	var combined []CyclicalPuzzle
	seenFENs := make(map[string]bool)

	// Convert matching Blunder Vault flashcards first (personal blunders)
	for _, vp := range vaultPuzzles {
		cp, ok := ConvertBlunderVaultPuzzle(vp)
		if !ok || seenFENs[cp.FEN] {
			continue
		}
		if !ThemeMaskFromKeys(cp.Themes).MatchesFilter(filterMask) {
			continue
		}
		seenFENs[cp.FEN] = true
		combined = append(combined, cp)
		if len(combined) >= setSize {
			break
		}
	}

	// Extract from player games and built-in library for the remaining slots
	if len(combined) < setSize {
		var playerNames []string
		if strings.TrimSpace(playerName) != "" {
			playerNames = []string{playerName}
		}
		more := ExtractPuzzlesFromPlayerGames(games, playerNames, themeFilter, setSize)
		for _, p := range more {
			if len(combined) >= setSize {
				break
			}
			if seenFENs[p.FEN] {
				continue
			}
			seenFENs[p.FEN] = true
			combined = append(combined, p)
		}
	}

	sortedPuzzles := SortPuzzlesEasyToHard(combined)

	setID := fmt.Sprintf("cycleset-%d-%d", time.Now().UnixMilli(), len(cm.sets)+1)
	newSet := &CycleSet{
		ID:           setID,
		Name:         cleanName,
		PlayerName:   strings.TrimSpace(playerName),
		ThemeFilter:  append([]string(nil), themeFilter...),
		CreatedAt:    time.Now().UnixMilli(),
		CurrentCycle: 1,
		TargetDays:   28,
		Puzzles:      sortedPuzzles,
		Cycles:       make([]CycleRun, 0),
		ActiveProgress: &ActiveCycleProgress{
			CycleNumber:  1,
			CurrentIndex: 0,
			Results:      make(map[string]PuzzleAttemptResult),
		},
	}

	cm.sets = append(cm.sets, newSet)
	err := cm.saveLocked()
	return newSet, err
}

// GetCycleSets returns all persisted CycleSets.
func (cm *CyclicalManager) GetCycleSets() []*CycleSet {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	out := make([]*CycleSet, len(cm.sets))
	copy(out, cm.sets)
	return out
}

// GetCycleSet retrieves a single CycleSet by ID.
func (cm *CyclicalManager) GetCycleSet(setID string) (*CycleSet, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	for _, s := range cm.sets {
		if s.ID == setID {
			return s, nil
		}
	}
	return nil, fmt.Errorf("cycle set %s not found", setID)
}

// DeleteCycleSet removes a CycleSet by ID.
func (cm *CyclicalManager) DeleteCycleSet(setID string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	for i, s := range cm.sets {
		if s.ID == setID {
			cm.sets = append(cm.sets[:i], cm.sets[i+1:]...)
			return cm.saveLocked()
		}
	}
	return fmt.Errorf("cycle set %s not found", setID)
}

// RecordCyclePuzzleAttempt records an attempt on a puzzle in the active cycle, updates per-theme Lichess
// proficiency ratings, and finalizes the CycleRun when all puzzles in the cycle have been completed.
func (cm *CyclicalManager) RecordCyclePuzzleAttempt(
	setID, puzzleID string,
	solvedFirstTry bool,
	timeTakenSec int,
) (*CycleSet, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	var targetSet *CycleSet
	for _, s := range cm.sets {
		if s.ID == setID {
			targetSet = s
			break
		}
	}
	if targetSet == nil {
		return nil, fmt.Errorf("cycle set %s not found", setID)
	}

	if timeTakenSec <= 0 {
		timeTakenSec = 1
	}

	var targetPuzzle *CyclicalPuzzle
	puzzleIdx := -1
	for i := range targetSet.Puzzles {
		if targetSet.Puzzles[i].ID == puzzleID {
			targetPuzzle = &targetSet.Puzzles[i]
			puzzleIdx = i
			break
		}
	}
	if targetPuzzle == nil {
		return nil, fmt.Errorf("puzzle %s not found in set %s", puzzleID, setID)
	}

	if targetSet.ActiveProgress == nil {
		targetSet.ActiveProgress = &ActiveCycleProgress{
			CycleNumber:  targetSet.CurrentCycle,
			CurrentIndex: 0,
			Results:      make(map[string]PuzzleAttemptResult),
		}
	}
	if targetSet.ActiveProgress.Results == nil {
		targetSet.ActiveProgress.Results = make(map[string]PuzzleAttemptResult)
	}

	prog := targetSet.ActiveProgress
	if _, alreadyRecorded := prog.Results[puzzleID]; !alreadyRecorded {
		prog.TotalAttempted++
		if solvedFirstTry {
			prog.FirstTryCorrect++
		}
	} else {
		prev := prog.Results[puzzleID]
		prog.ElapsedSec -= prev.TimeTakenSec
		if prev.SolvedFirstTry && !solvedFirstTry {
			prog.FirstTryCorrect--
		} else if !prev.SolvedFirstTry && solvedFirstTry {
			prog.FirstTryCorrect++
		}
	}

	prog.ElapsedSec += timeTakenSec
	prog.Results[puzzleID] = PuzzleAttemptResult{
		PuzzleID:       puzzleID,
		SolvedFirstTry: solvedFirstTry,
		TimeTakenSec:   timeTakenSec,
		AttemptedAt:    time.Now().UnixMilli(),
	}

	if puzzleIdx+1 > prog.CurrentIndex {
		prog.CurrentIndex = puzzleIdx + 1
	}

	// Update Lichess-style per-theme performance statistics
	cm.updateThemeStatsLocked(*targetPuzzle, solvedFirstTry, timeTakenSec)

	// Check if all puzzles in the cycle are now completed
	if len(prog.Results) >= len(targetSet.Puzzles) && len(targetSet.Puzzles) > 0 {
		cm.finalizeCycleLocked(targetSet)
	}

	err := cm.saveLocked()
	return targetSet, err
}

func (cm *CyclicalManager) updateThemeStatsLocked(p CyclicalPuzzle, solvedFirstTry bool, timeTakenSec int) {
	for _, themeKey := range p.Themes {
		id, ok := LookupThemeID(themeKey)
		if !ok {
			continue
		}
		info := lichessThemeCatalog[id]
		st, exists := cm.themeStats[info.Key]
		if !exists {
			st = ThemePerformanceStat{
				ThemeKey:  info.Key,
				ThemeName: info.Name,
				Category:  info.Category,
			}
		}

		st.Attempts++
		if solvedFirstTry {
			st.FirstWins++
		} else {
			st.Failed++
			st.Fixed++
		}
		st.TotalTimeSec += timeTakenSec
		st.TotalRatingSum += p.EstimatedRating

		st.AccuracyPct = math.Round((float64(st.FirstWins)/float64(st.Attempts))*1000.0) / 10.0
		st.AvgTimeSec = math.Round((float64(st.TotalTimeSec)/float64(st.Attempts))*10.0) / 10.0
		st.RatingAvg = int(math.Round(float64(st.TotalRatingSum) / float64(st.Attempts)))

		// Exact Lichess PuzzleDashboard.scala performance rating formula:
		//RatingAvg - 500 + round(1000 * FirstWins / Attempts)
		st.PerformanceRating = st.RatingAvg - 500 + int(math.Round(1000.0*float64(st.FirstWins)/float64(st.Attempts)))
		st.HistoryRatings = append(st.HistoryRatings, st.PerformanceRating)
		if len(st.HistoryRatings) > 20 {
			st.HistoryRatings = st.HistoryRatings[len(st.HistoryRatings)-20:]
		}

		cm.themeStats[info.Key] = st
	}
}

func (cm *CyclicalManager) finalizeCycleLocked(s *CycleSet) {
	prog := s.ActiveProgress
	totalPuzzles := len(s.Puzzles)
	if totalPuzzles == 0 || prog == nil {
		return
	}

	var easyTotal, easyCorrect int
	var medTotal, medCorrect int
	var hardTotal, hardCorrect int
	totalTime := 0
	firstTry := 0

	for _, p := range s.Puzzles {
		res, ok := prog.Results[p.ID]
		if !ok {
			continue
		}
		totalTime += res.TimeTakenSec
		if res.SolvedFirstTry {
			firstTry++
		}
		switch ParseDifficultyTierID(p.DifficultyTier) {
		case TierEasy:
			easyTotal++
			if res.SolvedFirstTry {
				easyCorrect++
			}
		case TierMedium:
			medTotal++
			if res.SolvedFirstTry {
				medCorrect++
			}
		case TierHard:
			hardTotal++
			if res.SolvedFirstTry {
				hardCorrect++
			}
		}
	}

	if totalTime <= 0 {
		totalTime = totalPuzzles
	}

	targetTime := totalTime
	speedup := 1.0
	if len(s.Cycles) > 0 {
		prev := s.Cycles[len(s.Cycles)-1]
		targetTime = prev.TotalTimeSec / 2
		if targetTime < 60 {
			targetTime = 60
		}
		cycle1Time := s.Cycles[0].TotalTimeSec
		if cycle1Time > 0 && totalTime > 0 {
			speedup = math.Round((float64(cycle1Time)/float64(totalTime))*100.0) / 100.0
		}
	}

	accPct := math.Round((float64(firstTry)/float64(totalPuzzles))*1000.0) / 10.0
	avgSec := math.Round((float64(totalTime)/float64(totalPuzzles))*10.0) / 10.0

	run := CycleRun{
		CycleNumber:       s.CurrentCycle,
		CompletedAt:       time.Now().UTC().Format(time.RFC3339),
		TotalPuzzles:      totalPuzzles,
		FirstTryCorrect:   firstTry,
		TotalSolved:       totalPuzzles,
		AccuracyPct:       accPct,
		TotalTimeSec:      totalTime,
		TargetTimeSec:     targetTime,
		AvgSecPerPuzzle:   avgSec,
		SpeedupVsCycle1:   speedup,
		EasyAccuracyPct:   calcPct(easyCorrect, easyTotal),
		MediumAccuracyPct: calcPct(medCorrect, medTotal),
		HardAccuracyPct:   calcPct(hardCorrect, hardTotal),
	}

	s.Cycles = append(s.Cycles, run)
	if s.CurrentCycle < 7 {
		s.CurrentCycle++
	}
	s.TargetDays = maxInt(1, s.TargetDays/2)
	s.ActiveProgress = &ActiveCycleProgress{
		CycleNumber:  s.CurrentCycle,
		CurrentIndex: 0,
		Results:      make(map[string]PuzzleAttemptResult),
	}
}

func calcPct(correct, total int) float64 {
	if total <= 0 {
		return 0
	}
	return math.Round((float64(correct)/float64(total))*1000.0) / 10.0
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// StartNextCycle manually advances or resets a CycleSet to start the next repetition pass.
func (cm *CyclicalManager) StartNextCycle(setID string) (*CycleSet, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	for _, s := range cm.sets {
		if s.ID == setID {
			if s.ActiveProgress != nil && len(s.ActiveProgress.Results) > 0 {
				cm.finalizeCycleLocked(s)
			} else {
				if s.CurrentCycle < 7 {
					s.CurrentCycle++
				}
				s.TargetDays = maxInt(1, s.TargetDays/2)
				s.ActiveProgress = &ActiveCycleProgress{
					CycleNumber:  s.CurrentCycle,
					CurrentIndex: 0,
					Results:      make(map[string]PuzzleAttemptResult),
				}
			}
			err := cm.saveLocked()
			return s, err
		}
	}
	return nil, fmt.Errorf("cycle set %s not found", setID)
}

// ResetCurrentCycle clears active progress for the current cycle so the user can restart the pass cleanly.
func (cm *CyclicalManager) ResetCurrentCycle(setID string) (*CycleSet, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	for _, s := range cm.sets {
		if s.ID == setID {
			s.ActiveProgress = &ActiveCycleProgress{
				CycleNumber:  s.CurrentCycle,
				CurrentIndex: 0,
				Results:      make(map[string]PuzzleAttemptResult),
			}
			err := cm.saveLocked()
			return s, err
		}
	}
	return nil, fmt.Errorf("cycle set %s not found", setID)
}

// GetDashboard computes the complete Lichess-style tactical proficiency radar and cyclical speedup summary.
func (cm *CyclicalManager) GetDashboard() CyclicalDashboard {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	var allStats []ThemePerformanceStat
	var actionableStats []ThemePerformanceStat
	totalAttempts := 0
	totalFirstWins := 0
	totalRatingSum := 0

	for _, st := range cm.themeStats {
		allStats = append(allStats, st)
		id, ok := LookupThemeID(st.ThemeKey)
		if ok && lichessThemeCatalog[id].Actionable {
			actionableStats = append(actionableStats, st)
			totalAttempts += st.Attempts
			totalFirstWins += st.FirstWins
			totalRatingSum += st.TotalRatingSum
		}
	}

	sort.Slice(allStats, func(i, j int) bool {
		if allStats[i].Attempts != allStats[j].Attempts {
			return allStats[i].Attempts > allStats[j].Attempts
		}
		return allStats[i].ThemeKey < allStats[j].ThemeKey
	})

	sort.Slice(actionableStats, func(i, j int) bool {
		if actionableStats[i].Attempts != actionableStats[j].Attempts {
			return actionableStats[i].Attempts > actionableStats[j].Attempts
		}
		return actionableStats[i].ThemeKey < actionableStats[j].ThemeKey
	})

	// RadarAxes: top 9 played actionable themes (or seeded defaults if fewer than 9 have been played yet)
	radarAxes := make([]ThemePerformanceStat, 0, 9)
	seenRadar := make(map[string]bool)
	for _, st := range actionableStats {
		if len(radarAxes) >= 9 {
			break
		}
		radarAxes = append(radarAxes, st)
		seenRadar[st.ThemeKey] = true
	}
	defaultRadarKeys := []ThemeID{
		ThemeFork, ThemePin, ThemeSkewer, ThemeDiscoveredAttack,
		ThemeSacrifice, ThemeDeflection, ThemeBackRankMate, ThemeHangingPiece, ThemeEndgame,
	}
	for _, id := range defaultRadarKeys {
		if len(radarAxes) >= 9 {
			break
		}
		info := lichessThemeCatalog[id]
		if seenRadar[info.Key] {
			continue
		}
		radarAxes = append(radarAxes, ThemePerformanceStat{
			ThemeKey:          info.Key,
			ThemeName:         info.Name,
			Category:          info.Category,
			RatingAvg:         1500,
			PerformanceRating: 1500,
		})
	}

	// Strongest & Weakest actionable themes
	byPerf := make([]ThemePerformanceStat, len(actionableStats))
	copy(byPerf, actionableStats)
	sort.Slice(byPerf, func(i, j int) bool {
		if byPerf[i].PerformanceRating != byPerf[j].PerformanceRating {
			return byPerf[i].PerformanceRating > byPerf[j].PerformanceRating
		}
		return byPerf[i].ThemeKey < byPerf[j].ThemeKey
	})

	var strongThemes, weakThemes []ThemePerformanceStat
	for i := 0; i < len(byPerf) && i < 3; i++ {
		strongThemes = append(strongThemes, byPerf[i])
	}
	for i := len(byPerf) - 1; i >= 0 && len(weakThemes) < 3; i-- {
		weakThemes = append(weakThemes, byPerf[i])
	}

	globalRating := 1500
	if totalAttempts > 0 {
		avgRating := int(math.Round(float64(totalRatingSum) / float64(totalAttempts)))
		globalRating = avgRating - 500 + int(math.Round(1000.0*float64(totalFirstWins)/float64(totalAttempts)))
	}

	totalCyclesCompleted := 0
	totalPuzzlesAttempted := 0
	bestSpeedup := 1.0
	bestAccDelta := 0.0

	setsCopy := make([]*CycleSet, len(cm.sets))
	copy(setsCopy, cm.sets)
	for _, s := range setsCopy {
		totalCyclesCompleted += len(s.Cycles)
		for _, c := range s.Cycles {
			totalPuzzlesAttempted += c.TotalPuzzles
			if c.SpeedupVsCycle1 > bestSpeedup {
				bestSpeedup = c.SpeedupVsCycle1
			}
		}
		if len(s.Cycles) >= 2 {
			delta := s.Cycles[len(s.Cycles)-1].AccuracyPct - s.Cycles[0].AccuracyPct
			if delta > bestAccDelta {
				bestAccDelta = delta
			}
		}
		if s.ActiveProgress != nil {
			totalPuzzlesAttempted += s.ActiveProgress.TotalAttempted
		}
	}

	summary := "Start Cycle 1 to establish your baseline tactical pattern recognition speed and accuracy."
	if totalCyclesCompleted > 0 {
		summary = fmt.Sprintf(
			"Completed %d cycle(s) (%d puzzle attempts) with %.2fx peak speed improvement and %+.1f%% accuracy gain.",
			totalCyclesCompleted, totalPuzzlesAttempted, bestSpeedup, bestAccDelta,
		)
	}

	return CyclicalDashboard{
		Sets:                    setsCopy,
		ThemeStats:              allStats,
		RadarAxes:               radarAxes,
		StrongThemes:            strongThemes,
		WeakThemes:              weakThemes,
		GlobalPerformanceRating: globalRating,
		TotalPuzzlesAttempted:   totalPuzzlesAttempted,
		TotalCyclesCompleted:    totalCyclesCompleted,
		ImprovementSummary:      summary,
		MethodAttribution:       CyclicalMethodAttribution,
	}
}
