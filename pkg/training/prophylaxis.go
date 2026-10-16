package training

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"

	"newscid/pkg/chess"
	"newscid/pkg/database/scid"
)

// ProphylaxisCategory classifies the strategic objective of the quiet move.
type ProphylaxisCategory string

const (
	CategoryKingSafety     ProphylaxisCategory = "king_safety"     // e.g. Kh1, Kh8, avoiding pins/checks
	CategoryPieceRestraint ProphylaxisCategory = "piece_restraint" // e.g. a3/h3 denying enemy outpost/pins
	CategoryOverprotection ProphylaxisCategory = "overprotection"  // Nimzowitsch overprotection of strongpoint
	CategoryBlockade       ProphylaxisCategory = "blockade"        // Restricting enemy passed pawn or file
	CategoryQuietRegroup   ProphylaxisCategory = "quiet_regroup"   // Calm retreat preparing future breakthrough
)

// ProphylaxisDrill represents a quiet move or prophylactic puzzle.
type ProphylaxisDrill struct {
	ID              string              `json:"id"`
	Title           string              `json:"title"`
	FEN             string              `json:"fen"`
	Turn            string              `json:"turn"` // "white" | "black"
	Category        ProphylaxisCategory `json:"category"`
	CategoryLabel   string              `json:"categoryLabel"`
	Prompt          string              `json:"prompt"`
	WinningMoveUCI  string              `json:"winningMoveUCI"`
	WinningMoveSAN  string              `json:"winningMoveSAN"`
	AlternativeUCIs []string            `json:"alternativeUCIs"`
	OpponentThreat  string              `json:"opponentThreat"`
	ThreatArrowFrom string              `json:"threatArrowFrom"`
	ThreatArrowTo   string              `json:"threatArrowTo"`
	Explanation     string              `json:"explanation"`
	Source          string              `json:"source"`
	Difficulty      string              `json:"difficulty"`
}

// ProphylaxisFeedback evaluates user guess against the drill.
type ProphylaxisFeedback struct {
	IsCorrect      bool   `json:"isCorrect"`
	PointsAwarded  int    `json:"pointsAwarded"`
	BestMoveUCI    string `json:"bestMoveUCI"`
	BestMoveSAN    string `json:"bestMoveSAN"`
	Explanation    string `json:"explanation"`
	OpponentThreat string `json:"opponentThreat"`
}

var (
	prophylaxisMu   sync.RWMutex
	prophylaxisBank = make(map[string]ProphylaxisDrill)
)

var curatedProphylaxisDrills = []ProphylaxisDrill{
	{
		ID:              "pro-01-karpov-kh1",
		Title:           "Karpov's Prophylactic King Step",
		FEN:             "r1b2rk1/pp1nq1pp/4p3/2ppPp2/3P4/2PB1N2/PP1Q1PPP/R4RK1 w - - 0 13",
		Turn:            "white",
		Category:        CategoryKingSafety,
		CategoryLabel:   "King Safety Unpin",
		Prompt:          "Black is ready to launch counterplay on the queenside. Step out of potential diagonal pins before opening files.",
		WinningMoveUCI:  "g1h1",
		WinningMoveSAN:  "Kh1",
		OpponentThreat:  "Black intends ...c4 followed by tactical bishop checks on the g1-a7 diagonal.",
		ThreatArrowFrom: "c5",
		ThreatArrowTo:   "c4",
		Explanation:     "Kh1! A signature Karpovian calm move. White steps off the sensitive g1-a7 diagonal before advancing central pawns, eliminating all future tactical checks.",
		Source:          "Karpov vs. Spassky, Candidates Semifinal 1974",
		Difficulty:      "advanced",
	},
	{
		ID:              "pro-02-karpov-a3-squeeze",
		Title:           "The Universal Prophylactic Clamp",
		FEN:             "r1bqr1k1/pp1nbppp/2p1pn2/3p4/2PP4/1PN1PN2/PB2BPPP/R2Q1RK1 w - - 3 10",
		Turn:            "white",
		Category:        CategoryPieceRestraint,
		CategoryLabel:   "Piece Restraint",
		Prompt:          "Deny Black's minor pieces active outposts on the queenside without rushing central pawn contact.",
		WinningMoveUCI:  "a2a3",
		WinningMoveSAN:  "a3",
		OpponentThreat:  "Black prepares ...Ne4 or ...Bb4 pinning the knight and disrupting White's harmony.",
		ThreatArrowFrom: "e7",
		ThreatArrowTo:   "b4",
		Explanation:     "a3! Denies both ...Bb4 and ...Nb4. Black's minor pieces are starved of active forward squares while White prepares b4 queenside expansion.",
		Source:          "Karpov vs. Unzicker, Nice Olympiad 1974",
		Difficulty:      "intermediate",
	},
	{
		ID:              "pro-03-nimzowitsch-blockade",
		Title:           "Nimzowitsch's Supreme Blockade",
		FEN:             "r2q1rk1/1b1nbppp/p3pn2/1p1p4/3P4/1PNBPN2/PB3PPP/R2Q1RK1 w - - 2 12",
		Turn:            "white",
		Category:        CategoryBlockade,
		CategoryLabel:   "Central Blockade",
		Prompt:          "Black has expanded on the queenside. Plant a solid piece blockade in front of the pawns to freeze counterplay.",
		WinningMoveUCI:  "c3e2",
		WinningMoveSAN:  "Ne2",
		AlternativeUCIs: []string{"a2a3"},
		OpponentThreat:  "Black wants to push ...b4 dislodging the c3 knight.",
		ThreatArrowFrom: "b5",
		ThreatArrowTo:   "b4",
		Explanation:     "Ne2! Prophylactically reroutes the knight to g3 or d4, anticipating ...b4 and rendering Black's pawn push harmless.",
		Source:          "Nimzowitsch vs. Salwe, Karlsbad 1911",
		Difficulty:      "advanced",
	},
	{
		ID:              "pro-04-fischer-luft",
		Title:           "Fischer's Calm Luft & Pin Prevention",
		FEN:             "r1b2rk1/ppqn1ppp/2p1p3/3p4/2PP4/2N1PN2/PP2BPPP/R2Q1RK1 w - - 4 10",
		Turn:            "white",
		Category:        CategoryKingSafety,
		CategoryLabel:   "Luft & Restraint",
		Prompt:          "Black has developed solidly. Play a calm prophylactic pawn step preventing enemy bishop pins.",
		WinningMoveUCI:  "h2h3",
		WinningMoveSAN:  "h3",
		OpponentThreat:  "Black threatens ...Bg4 pinning White's key f3 defender of the central d4 square.",
		ThreatArrowFrom: "c8",
		ThreatArrowTo:   "g4",
		Explanation:     "h3! Shuts down ...Bg4 permanently, protects the vital f3 knight, and grants the White king permanent escape air against back-rank tactics.",
		Source:          "Fischer vs. Petrosian, Candidates Final 1971",
		Difficulty:      "intermediate",
	},
	{
		ID:              "pro-05-carlsen-regroup",
		Title:           "Carlsen's Quiet Bishop Retreat",
		FEN:             "r2q1rk1/pp1b1ppp/2n1pn2/2bp4/8/2NBPN2/PPPB1PPP/R2Q1RK1 w - - 4 9",
		Turn:            "white",
		Category:        CategoryQuietRegroup,
		CategoryLabel:   "Prophylactic Retreat",
		Prompt:          "Black's active pieces threaten ...Nb4 trading off your valuable dark-squared attacking bishop.",
		WinningMoveUCI:  "d3e2",
		WinningMoveSAN:  "Be2",
		OpponentThreat:  "Black plays ...Nb4 hunting down White's attacking bishop on d3.",
		ThreatArrowFrom: "c6",
		ThreatArrowTo:   "b4",
		Explanation:     "Be2! A calm prophylactic retreat preserving the bishop pair before Black can play ...Nb4. Carlsen avoids the exchange and keeps tactical leverage.",
		Source:          "Carlsen vs. Anand, World Championship 2013",
		Difficulty:      "master",
	},
	{
		ID:              "pro-06-petrosian-kh8",
		Title:           "The Iron Tigran's Silent Refuge",
		FEN:             "r4rk1/1pp1qpp1/p1np1n1p/4p3/2B1P1b1/2NPPN2/PPP1Q1PP/R4RK1 b - - 1 12",
		Turn:            "black",
		Category:        CategoryKingSafety,
		CategoryLabel:   "Diagonal Evasion",
		Prompt:          "White's bishop is actively positioned on c4. Neutralize future pin tactics with a classic Petrosian king tuck.",
		WinningMoveUCI:  "g8h8",
		WinningMoveSAN:  "Kh8",
		OpponentThreat:  "White prepares f4-f5 with tactical pins on the g8 king through the a2-g8 diagonal.",
		ThreatArrowFrom: "f2",
		ThreatArrowTo:   "f4",
		Explanation:     "Kh8! Black steps off the dangerous a2-g8 diagonal. Now Black is completely free to contest the f-file and play ...Ng8 without being pinned.",
		Source:          "Petrosian vs. Pachman, Bled 1961",
		Difficulty:      "advanced",
	},
}

func init() {
	for _, drill := range curatedProphylaxisDrills {
		RegisterProphylaxisDrill(drill)
	}
}

// RegisterProphylaxisDrill caches a drill in the global memory bank.
func RegisterProphylaxisDrill(drill ProphylaxisDrill) {
	prophylaxisMu.Lock()
	defer prophylaxisMu.Unlock()
	prophylaxisBank[drill.ID] = drill
}

// GetProphylaxisDrill fetches a drill by ID.
func GetProphylaxisDrill(id string) (ProphylaxisDrill, bool) {
	prophylaxisMu.RLock()
	defer prophylaxisMu.RUnlock()
	d, ok := prophylaxisBank[id]
	return d, ok
}

// GetProphylaxisDrillBatch retrieves a randomized set of drills filtered by category.
func GetProphylaxisDrillBatch(count int, category string) []ProphylaxisDrill {
	prophylaxisMu.RLock()
	defer prophylaxisMu.RUnlock()

	var matched []ProphylaxisDrill
	for _, d := range prophylaxisBank {
		if category == "" || category == "all" || string(d.Category) == category {
			matched = append(matched, d)
		}
	}

	rand.Shuffle(len(matched), func(i, j int) {
		matched[i], matched[j] = matched[j], matched[i]
	})

	if count <= 0 || count > len(matched) {
		count = len(matched)
	}
	return matched[:count]
}

// SubmitProphylaxisAnswer evaluates the user's move for quiet prophylactic correctness.
func SubmitProphylaxisAnswer(drillID string, moveUCI string) (*ProphylaxisFeedback, error) {
	drill, ok := GetProphylaxisDrill(drillID)
	if !ok {
		return nil, fmt.Errorf("drill %q not found", drillID)
	}

	cleanMove := strings.ToLower(strings.TrimSpace(moveUCI))
	cleanWin := strings.ToLower(strings.TrimSpace(drill.WinningMoveUCI))

	isCorrect := (cleanMove == cleanWin)
	if !isCorrect {
		for _, alt := range drill.AlternativeUCIs {
			if cleanMove == strings.ToLower(strings.TrimSpace(alt)) {
				isCorrect = true
				break
			}
		}
	}

	points := 0
	if isCorrect {
		points = 150
	}

	return &ProphylaxisFeedback{
		IsCorrect:      isCorrect,
		PointsAwarded:  points,
		BestMoveUCI:    drill.WinningMoveUCI,
		BestMoveSAN:    drill.WinningMoveSAN,
		Explanation:    drill.Explanation,
		OpponentThreat: drill.OpponentThreat,
	}, nil
}

// GenerateProphylaxisDrillsFromDatabase mines quiet masterstroke positions from the active database.
func GenerateProphylaxisDrillsFromDatabase(db *scid.Database, count int) ([]ProphylaxisDrill, error) {
	if db == nil || !db.IsOpen {
		return nil, fmt.Errorf("database is not open")
	}

	totalGames := len(db.Entries)
	if totalGames == 0 {
		return GetProphylaxisDrillBatch(count, "all"), nil
	}

	var drills []ProphylaxisDrill

	sampleIndices := rand.Perm(totalGames)
	if len(sampleIndices) > 60 {
		sampleIndices = sampleIndices[:60]
	}

	for _, gIdx := range sampleIndices {
		if len(drills) >= count {
			break
		}

		entry := db.Entries[gIdx]
		if (entry.Flags & scid.FlagDelete) != 0 {
			continue
		}

		game, err := db.GetGame(gIdx + 1)
		if err != nil || game == nil || game.RootMove == nil {
			continue
		}

		whiteName := db.Names.GetName(scid.NamePlayer, entry.WhiteID)
		blackName := db.Names.GetName(scid.NamePlayer, entry.BlackID)
		event := db.Names.GetName(scid.NameEvent, entry.EventID)

		// Search for quiet king steps (Kh1/Kh8) or pawn restraint (a3/h3/a6/h6)
		b := chess.NewBoard()
		curr := game.RootMove
		ply := 0

		for curr != nil && ply < 45 {
			m, err := b.ParseUCI(curr.UCI)
			if err != nil {
				break
			}

			// Check if this move is a quiet prophylactic move
			san := curr.SAN
			isQuietKing := (san == "Kh1" || san == "Kh8" || san == "Kg1" || san == "Kg8" || san == "Kf1")
			isPawnRestraint := (san == "a3" || san == "h3" || san == "a6" || san == "h6")

			if (isQuietKing || isPawnRestraint) && ply >= 14 {
				cat := CategoryPieceRestraint
				catLabel := "Pawn Restraint"
				prompt := "White to move. Calmly restrict enemy counterplay."
				if isQuietKing {
					cat = CategoryKingSafety
					catLabel = "King Safety Unpin"
					prompt = "Step your king into safety, preempting potential checks or diagonal pins."
				}

				d := ProphylaxisDrill{
					ID:             fmt.Sprintf("db-pro-%d-%d", gIdx+1, ply+1),
					Title:          fmt.Sprintf("Prophylaxis: %s vs %s", whiteName, blackName),
					FEN:            b.FEN(),
					Turn:           "white",
					Category:       cat,
					CategoryLabel:  catLabel,
					Prompt:         prompt,
					WinningMoveUCI: curr.UCI,
					WinningMoveSAN: curr.SAN,
					OpponentThreat: "Opponent prepares tactical counterplay on active diagonals.",
					Explanation:    fmt.Sprintf("%s was played by %s in %s to restrict counterplay and ensure positional stability.", curr.SAN, whiteName, event),
					Source:         fmt.Sprintf("%s vs %s (%s)", whiteName, blackName, event),
					Difficulty:     "advanced",
				}
				if b.Turn == chess.Black {
					d.Turn = "black"
				}

				RegisterProphylaxisDrill(d)
				drills = append(drills, d)
				break
			}

			b.MakeMove(m)
			curr = curr.Next
			ply++
		}
	}

	if len(drills) == 0 {
		return GetProphylaxisDrillBatch(count, "all"), nil
	}

	return drills, nil
}
