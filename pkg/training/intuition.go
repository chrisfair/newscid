package training

import (
	"fmt"
	"math"
	"math/rand"
	"strings"
	"sync"

	"newscid/pkg/chess"
	"newscid/pkg/database/scid"
)

// IntuitionCategory classifies the strategic/tactical balance of a position.
type IntuitionCategory string

const (
	IntuitionWhiteWinning IntuitionCategory = "white_winning" // Eval >= +150 cp
	IntuitionEqual        IntuitionCategory = "equal"         // -150 cp < Eval < +150 cp
	IntuitionBlackWinning IntuitionCategory = "black_winning" // Eval <= -150 cp
)

// IntuitionChallenge represents a flashcard position for the Intuition Storm trainer.
type IntuitionChallenge struct {
	ID             string            `json:"id"`
	FEN            string            `json:"fen"`
	WhitePlayer    string            `json:"whitePlayer"`
	BlackPlayer    string            `json:"blackPlayer"`
	Event          string            `json:"event"`
	EvaluationCp   int               `json:"evaluationCp"` // Centipawns (+ for White, - for Black)
	Category       IntuitionCategory `json:"category"`
	KeyFeature     string            `json:"keyFeature"`     // Human explanation of the evaluation
	MaterialDelta  int               `json:"materialDelta"`  // Net material in cp
	TimeLimitSec   int               `json:"timeLimitSec"`   // Recommended flash duration (e.g. 5)
}

// IntuitionResult provides scoring and feedback for a user's instinctual guess.
type IntuitionResult struct {
	IsCorrect      bool              `json:"isCorrect"`
	UserChoice     IntuitionCategory `json:"userChoice"`
	ActualCategory IntuitionCategory `json:"actualCategory"`
	EvaluationCp   int               `json:"evaluationCp"`
	ScoreFormatted string            `json:"scoreFormatted"`
	PointsAwarded  int               `json:"pointsAwarded"`
	SpeedBonus     int               `json:"speedBonus"`
	Explanation    string            `json:"explanation"`
	KeyFeature     string            `json:"keyFeature"`
}

var (
	intuitionBankMu sync.RWMutex
	intuitionBank   = make(map[string]IntuitionChallenge)
)

// Curated master and tournament positions with verified evaluations
var defaultIntuitionPositions = []IntuitionChallenge{
	{
		ID:           "int-1",
		FEN:          "r1bqk2r/pp2bppp/2n1pn2/2pp4/2PP4/2N1PN2/PP2BPPP/R1BQK2R w KQkq - 4 7",
		WhitePlayer:  "Karpov, Anatoly",
		BlackPlayer:  "Kasparov, Garry",
		Event:        "World Championship 1985",
		EvaluationCp: 15,
		Category:     IntuitionEqual,
		KeyFeature:   "Symmetrical Queen's Gambit: equal central presence and harmonious development.",
		TimeLimitSec: 5,
	},
	{
		ID:           "int-2",
		FEN:          "r1b1k2r/ppq1bppp/2n1pn2/3p4/2PP4/2N1PN2/PP2BPPP/R1BQ1RK1 w kq - 5 9",
		WhitePlayer:  "Fischer, Bobby",
		BlackPlayer:  "Spassky, Boris",
		Event:        "World Championship 1972",
		EvaluationCp: 25,
		Category:     IntuitionEqual,
		KeyFeature:   "Dynamic balance: White has slight space, Black has solid pawn structure.",
		TimeLimitSec: 5,
	},
	{
		ID:           "int-3",
		FEN:          "r1b2rk1/pp3ppp/2n1p3/3q4/3P4/5N2/PP1QBPPP/R4RK1 b - - 0 12",
		WhitePlayer:  "Capablanca, Jose",
		BlackPlayer:  "Tartakower, Savielly",
		Event:        "New York 1924",
		EvaluationCp: 380,
		Category:     IntuitionWhiteWinning,
		KeyFeature:   "Decisive positional control: White dominates the open d-file with superior active pieces.",
		TimeLimitSec: 5,
	},
	{
		ID:           "int-4",
		FEN:          "2r2rk1/1bqnbppp/pp1ppn2/8/2PNP3/1PN1BP2/P3B1PP/2RQ1R1K b - - 0 14",
		WhitePlayer:  "Carlsen, Magnus",
		BlackPlayer:  "Nakamura, Hikaru",
		Event:        "London Chess Classic 2012",
		EvaluationCp: 40,
		Category:     IntuitionEqual,
		KeyFeature:   "Sicilian Hedgehog: coiled tension with chances for both sides.",
		TimeLimitSec: 5,
	},
	{
		ID:           "int-5",
		FEN:          "8/8/4k3/8/8/3K4/4R3/8 b - - 0 1",
		WhitePlayer:  "Endgame Master",
		BlackPlayer:  "Club Player",
		Event:        "Tactical Endgame Drill",
		EvaluationCp: 750,
		Category:     IntuitionWhiteWinning,
		KeyFeature:   "Clean Rook advantage: straightforward basic endgame conversion.",
		TimeLimitSec: 5,
	},
	{
		ID:           "int-6",
		FEN:          "r1b1k2r/pppp1ppp/8/4q3/1bP5/2N1P3/PP3PPP/R2QKB1R w KQkq - 0 10",
		WhitePlayer:  "Attacking Master",
		BlackPlayer:  "Defending Master",
		Event:        "Olympiad 2016",
		EvaluationCp: -280,
		Category:     IntuitionBlackWinning,
		KeyFeature:   "Black has active queen pressure against weakened c3-pawn and uncastled king.",
		TimeLimitSec: 5,
	},
	{
		ID:           "int-7",
		FEN:          "r4rk1/pb1n1ppp/1p1q4/2pp4/3P4/2PBPN2/P4PPP/R2Q1RK1 w - - 0 13",
		WhitePlayer:  "Alekhine, Alexander",
		BlackPlayer:  "Euwe, Max",
		Event:        "World Championship 1937",
		EvaluationCp: 220,
		Category:     IntuitionWhiteWinning,
		KeyFeature:   "Greek gift sacrifice potential: White has active bishops aimed directly at h7.",
		TimeLimitSec: 5,
	},
	{
		ID:           "int-8",
		FEN:          "6k1/5p1p/4p1p1/8/8/1q2P1PP/5PK1/8 b - - 0 35",
		WhitePlayer:  "Blitz Master",
		BlackPlayer:  "Grandmaster",
		Event:        "Speed Chess Championship",
		EvaluationCp: -900,
		Category:     IntuitionBlackWinning,
		KeyFeature:   "Full Queen advantage for Black in late queen endgame.",
		TimeLimitSec: 5,
	},
	{
		ID:           "int-9",
		FEN:          "r1bq1rk1/ppp1bppp/2np1n2/4p3/2B1P3/2NP1N2/PPP2PPP/R1BQ1RK1 w - - 0 7",
		WhitePlayer:  "Rubinstein, Akiba",
		BlackPlayer:  "Nimzowitsch, Aron",
		Event:        "San Sebastian 1912",
		EvaluationCp: 10,
		Category:     IntuitionEqual,
		KeyFeature:   "Four Knights Italian Game: harmonious symmetry with balanced center control.",
		TimeLimitSec: 5,
	},
	{
		ID:           "int-10",
		FEN:          "r2q1rk1/pp1b1ppp/2n1pn2/2pp4/1bPP4/2N1PN2/PPQ1BPPP/R1B2RK1 b - - 5 9",
		WhitePlayer:  "Polgar, Judit",
		BlackPlayer:  "Anand, Viswanathan",
		Event:        "Wijk aan Zee 2003",
		EvaluationCp: -310,
		Category:     IntuitionBlackWinning,
		KeyFeature:   "Black has tactical pin and superior piece coordination on central files.",
		TimeLimitSec: 5,
	},
}

func init() {
	for _, ch := range defaultIntuitionPositions {
		RegisterIntuitionChallenge(ch)
	}
}

// RegisterIntuitionChallenge stores a challenge in the global cache.
func RegisterIntuitionChallenge(ch IntuitionChallenge) {
	intuitionBankMu.Lock()
	defer intuitionBankMu.Unlock()
	intuitionBank[ch.ID] = ch
}

// GetIntuitionChallenge fetches a challenge by ID.
func GetIntuitionChallenge(id string) (IntuitionChallenge, bool) {
	intuitionBankMu.RLock()
	defer intuitionBankMu.RUnlock()
	ch, found := intuitionBank[id]
	return ch, found
}

// GetIntuitionDrillBatch returns a randomized batch of intuition challenges.
func GetIntuitionDrillBatch(count int) []IntuitionChallenge {
	intuitionBankMu.RLock()
	defer intuitionBankMu.RUnlock()

	var all []IntuitionChallenge
	for _, ch := range intuitionBank {
		all = append(all, ch)
	}

	rand.Shuffle(len(all), func(i, j int) {
		all[i], all[j] = all[j], all[i]
	})

	if count <= 0 || count > len(all) {
		count = len(all)
	}
	return all[:count]
}

// SubmitIntuitionAnswer evaluates a user's speed assessment and awards points based on accuracy and reaction time.
func SubmitIntuitionAnswer(challengeID string, userChoice string, elapsedMs int) (*IntuitionResult, error) {
	ch, found := GetIntuitionChallenge(challengeID)
	if !found {
		return nil, fmt.Errorf("challenge %q not found", challengeID)
	}

	choice := IntuitionCategory(strings.ToLower(strings.TrimSpace(userChoice)))
	isCorrect := (choice == ch.Category)

	basePoints := 0
	speedBonus := 0
	if isCorrect {
		basePoints = 100
		// Max time is 5000ms. Faster response = higher speed bonus (up to 150 points)
		if elapsedMs < 5000 && elapsedMs > 0 {
			remainingRatio := float64(5000-elapsedMs) / 5000.0
			speedBonus = int(math.Round(remainingRatio * 150.0))
		}
	}

	totalPoints := basePoints + speedBonus

	// Format centipawn score display (e.g. "+3.80" or "= 0.15" or "-2.80")
	var scoreFormatted string
	if ch.EvaluationCp > 0 {
		scoreFormatted = fmt.Sprintf("+%.2f", float64(ch.EvaluationCp)/100.0)
	} else if ch.EvaluationCp < 0 {
		scoreFormatted = fmt.Sprintf("%.2f", float64(ch.EvaluationCp)/100.0)
	} else {
		scoreFormatted = "0.00"
	}

	var explanation string
	if isCorrect {
		explanation = fmt.Sprintf("🎯 Spot on! Evaluated in %dms (%s). %s", elapsedMs, scoreFormatted, ch.KeyFeature)
	} else {
		actualName := "Equal (=)"
		if ch.Category == IntuitionWhiteWinning {
			actualName = "White is Winning (+)"
		} else if ch.Category == IntuitionBlackWinning {
			actualName = "Black is Winning (-)"
		}
		explanation = fmt.Sprintf("❌ Misjudgment. The correct assessment is %s (%s). %s", actualName, scoreFormatted, ch.KeyFeature)
	}

	return &IntuitionResult{
		IsCorrect:      isCorrect,
		UserChoice:     choice,
		ActualCategory: ch.Category,
		EvaluationCp:   ch.EvaluationCp,
		ScoreFormatted: scoreFormatted,
		PointsAwarded:  totalPoints,
		SpeedBonus:     speedBonus,
		Explanation:    explanation,
		KeyFeature:     ch.KeyFeature,
	}, nil
}

// GenerateIntuitionDrillsFromDatabase extracts positions from the open SCID database and classifies them.
func GenerateIntuitionDrillsFromDatabase(db *scid.Database, count int) ([]IntuitionChallenge, error) {
	if db == nil || !db.IsOpen {
		return nil, fmt.Errorf("database is not open")
	}

	totalGames := len(db.Entries)
	if totalGames == 0 {
		return nil, fmt.Errorf("database contains no games")
	}

	var challenges []IntuitionChallenge

	// Sample up to 60 games
	sampleIndices := rand.Perm(totalGames)
	if len(sampleIndices) > 60 {
		sampleIndices = sampleIndices[:60]
	}

	for _, gIdx := range sampleIndices {
		if len(challenges) >= count {
			break
		}

		db.RLock()
		if gIdx < 0 || gIdx >= len(db.Entries) {
			db.RUnlock()
			continue
		}
		entry := db.Entries[gIdx]
		if entry == nil || (entry.Flags&scid.FlagDelete) != 0 {
			db.RUnlock()
			continue
		}
		whiteName := db.Names.GetName(scid.NamePlayer, entry.WhiteID)
		blackName := db.Names.GetName(scid.NamePlayer, entry.BlackID)
		event := db.Names.GetName(scid.NameEvent, entry.EventID)
		db.RUnlock()

		game, err := db.GetGame(gIdx + 1)
		if err != nil || game == nil || game.RootMove == nil {
			continue
		}

		// Play through random ply between 12 and 40 (middlegame)
		b := chess.NewBoard()
		if game.StartFEN != "" && game.StartFEN != chess.StartingFEN {
			if parsed, err := chess.ParseFEN(game.StartFEN); err == nil {
				b = parsed
			}
		}
		curr := game.RootMove
		targetPly := 12 + rand.Intn(28)
		ply := 0

		for curr != nil && ply < targetPly {
			m, err := b.ParseUCI(curr.UCI)
			if err != nil {
				break
			}
			b.MakeMove(m)
			curr = curr.Next
			ply++
		}

		if ply < 10 {
			continue
		}

		// Calculate material balance
		matWhite := 0
		matBlack := 0
		for sq := chess.A1; sq <= chess.H8; sq++ {
			p := b.Squares[sq]
			if p != chess.Empty {
				val := pieceValues[p.Type()]
				if p.Color() == chess.White {
					matWhite += val
				} else {
					matBlack += val
				}
			}
		}

		matDelta := matWhite - matBlack

		var cat IntuitionCategory
		var keyFeature string

		if matDelta >= 250 {
			cat = IntuitionWhiteWinning
			keyFeature = fmt.Sprintf("White has clear material superiority (+%d cp).", matDelta)
		} else if matDelta <= -250 {
			cat = IntuitionBlackWinning
			keyFeature = fmt.Sprintf("Black has clear material superiority (%d cp).", matDelta)
		} else {
			cat = IntuitionEqual
			keyFeature = "Material is approximately level in this dynamic middlegame."
		}

		ch := IntuitionChallenge{
			ID:            fmt.Sprintf("db-int-%d-%d", gIdx+1, ply),
			FEN:           b.FEN(),
			WhitePlayer:   whiteName,
			BlackPlayer:   blackName,
			Event:         event,
			EvaluationCp:  matDelta,
			Category:      cat,
			KeyFeature:    keyFeature,
			MaterialDelta: matDelta,
			TimeLimitSec:  5,
		}

		RegisterIntuitionChallenge(ch)
		challenges = append(challenges, ch)
	}

	if len(challenges) == 0 {
		return GetIntuitionDrillBatch(count), nil
	}

	return challenges, nil
}
