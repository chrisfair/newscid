package repertoire

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"newscid/pkg/chess"
)

// DeviationIncident records a single game where the user broke out of their opening book.
type DeviationIncident struct {
	GameIndex   int    `json:"gameIndex"`
	White       string `json:"white"`
	Black       string `json:"black"`
	Opponent    string `json:"opponent"`
	Event       string `json:"event"`
	Site        string `json:"site"`
	Date        string `json:"date"`
	Result      string `json:"result"`
	UserOutcome string `json:"userOutcome"` // "win", "draw", "loss"
	Ply         int    `json:"ply"`
	MoveNumber  int    `json:"moveNumber"`
	PlayedSAN   string `json:"playedSan"`
	BookSAN     string `json:"bookSan"`
}

// RecurringBookDeviation aggregates all instances where the user played a specific non-book move at a known book position.
type RecurringBookDeviation struct {
	Key               string              `json:"key"`
	FEN               string              `json:"fen"`
	CoreFEN           string              `json:"coreFen"`
	Ply               int                 `json:"ply"`
	MoveNumber        int                 `json:"moveNumber"`
	LineBeforeSAN     string              `json:"lineBeforeSan"`
	PlayedMoveUCI     string              `json:"playedMoveUci"`
	PlayedMoveSAN     string              `json:"playedMoveSan"`
	ExpectedMoves     []RepertoireMove    `json:"expectedMoves"`
	PrimaryBookUCI    string              `json:"primaryBookUci"`
	PrimaryBookSAN    string              `json:"primaryBookSan"`
	BookComment       string              `json:"bookComment,omitempty"`
	Occurrences       int                 `json:"occurrences"`
	Wins              int                 `json:"wins"`
	Draws             int                 `json:"draws"`
	Losses            int                 `json:"losses"`
	WinRatePct        float64             `json:"winRatePct"`
	EvalBookCp        int                 `json:"evalBookCp"`
	EvalPlayedCp      int                 `json:"evalPlayedCp"`
	EvalLossCp        int                 `json:"evalLossCp"`
	PunishingReplyUCI string              `json:"punishingReplyUci,omitempty"`
	PunishingReplySAN string              `json:"punishingReplySan,omitempty"`
	AvoidanceAdvice   string              `json:"avoidanceAdvice"`
	SampleGames       []DeviationIncident `json:"sampleGames"`
}

// UncoveredOpponentSideline aggregates opponent moves that deviated from the repertoire's covered responses.
type UncoveredOpponentSideline struct {
	Key                 string  `json:"key"`
	FEN                 string  `json:"fen"`
	CoreFEN             string  `json:"coreFen"`
	AfterOpponentFEN    string  `json:"afterOpponentFen"`
	Ply                 int     `json:"ply"`
	MoveNumber          int     `json:"moveNumber"`
	LineBeforeSAN       string  `json:"lineBeforeSan"`
	OpponentMoveUCI     string  `json:"opponentMoveUci"`
	OpponentMoveSAN     string  `json:"opponentMoveSan"`
	FullLineWithReply   string  `json:"fullLineWithReply"`
	Occurrences         int     `json:"occurrences"`
	Wins                int     `json:"wins"`
	Draws               int     `json:"draws"`
	Losses              int     `json:"losses"`
	WinRatePct          float64 `json:"winRatePct"`
	RecommendedReplyUCI string  `json:"recommendedReplyUci"`
	RecommendedReplySAN string  `json:"recommendedReplySan"`
	RecommendedNextFEN  string  `json:"recommendedNextFen"`
	RemediationAdvice   string  `json:"remediationAdvice"`
}

// BookDeviationReport summarizes book compliance, user book breaks, and uncovered opponent sidelines across a set of games.
type BookDeviationReport struct {
	RepertoireID          string                      `json:"repertoireId"`
	RepertoireName        string                      `json:"repertoireName"`
	Color                 string                      `json:"color"`
	PlayerIdentifiers     []string                    `json:"playerIdentifiers"`
	TotalGamesScanned     int                         `json:"totalGamesScanned"`
	MatchingColorGames    int                         `json:"matchingColorGames"`
	GamesEnteredBook      int                         `json:"gamesEnteredBook"`
	CompliantGames        int                         `json:"compliantGames"`
	UserDeviatedGames     int                         `json:"userDeviatedGames"`
	OpponentSidelineGames int                         `json:"opponentSidelineGames"`
	CompliancePct         float64                     `json:"compliancePct"`
	UserDeviations        []RecurringBookDeviation    `json:"userDeviations"`
	OpponentSidelines     []UncoveredOpponentSideline `json:"opponentSidelines"`
}

// ScanGamesForBookDeviations compares a slice of games against the specified repertoire,
// identifying every game where the user broke book or faced an uncovered opponent sideline.
func (m *Manager) ScanGamesForBookDeviations(repID string, games []*chess.Game, playerIdentifiers []string) (*BookDeviationReport, error) {
	rep, err := m.GetRepertoire(repID)
	if err != nil {
		return nil, err
	}
	return AnalyzeBookDeviations(rep, games, playerIdentifiers), nil
}

// AnalyzeBookDeviations performs book compliance and deviation analysis on a repertoire and game list.
func AnalyzeBookDeviations(rep *Repertoire, games []*chess.Game, playerIdentifiers []string) *BookDeviationReport {
	var cleanIDs []string
	seenID := make(map[string]bool)
	for _, raw := range playerIdentifiers {
		trimmed := strings.TrimSpace(raw)
		if trimmed != "" {
			lower := strings.ToLower(trimmed)
			if !seenID[lower] {
				seenID[lower] = true
				cleanIDs = append(cleanIDs, trimmed)
			}
		}
	}

	report := &BookDeviationReport{
		RepertoireID:      rep.ID,
		RepertoireName:    rep.Name,
		Color:             rep.Color,
		PlayerIdentifiers: cleanIDs,
		TotalGamesScanned: len(games),
		UserDeviations:    make([]RecurringBookDeviation, 0),
		OpponentSidelines: make([]UncoveredOpponentSideline, 0),
	}

	repColor := parseRepColor(rep.Color)
	userDevMap := make(map[string]*RecurringBookDeviation)
	oppSideMap := make(map[string]*UncoveredOpponentSideline)

	for gameIdx, g := range games {
		if g == nil {
			continue
		}
		whiteName := strings.TrimSpace(g.Tags["White"])
		blackName := strings.TrimSpace(g.Tags["Black"])

		if len(cleanIDs) > 0 {
			if repColor == chess.White && !matchesAnyIdentifier(whiteName, cleanIDs) {
				continue
			}
			if repColor == chess.Black && !matchesAnyIdentifier(blackName, cleanIDs) {
				continue
			}
		}

		var mainMoves []chess.Move
		for node := g.RootMove; node != nil; node = node.Next {
			mainMoves = append(mainMoves, node.Move)
		}
		if len(mainMoves) == 0 {
			continue
		}

		report.MatchingColorGames++
		opponentName := blackName
		if repColor == chess.Black {
			opponentName = whiteName
		}
		resStr := string(g.Result)
		if resStr == "" || resStr == "?" {
			resStr = g.Tags["Result"]
		}
		outcome := classifyGameOutcome(resStr, repColor)

		b := chess.NewBoard()
		var sanHistory []string
		matchedBookPlies := 0
		enteredBook := false
		deviatedByUser := false
		deviatedByOpp := false

		maxPlies := len(mainMoves)
		if maxPlies > 32 {
			maxPlies = 32
		}

		for plyIdx := 0; plyIdx < maxPlies; plyIdx++ {
			mv := mainMoves[plyIdx]
			legal := false
			for _, lm := range b.GenerateLegalMoves() {
				if lm.Equals(mv) {
					mv = lm
					legal = true
					break
				}
			}
			if !legal {
				break
			}

			fenBefore := b.FEN()
			coreBefore := CoreFEN(fenBefore)
			isUserTurn := b.Turn == repColor
			expectedMoves := rep.FindMovesForBoard(b)

			if len(expectedMoves) == 0 {
				// Reached end of book line without breaking book
				break
			}

			enteredBook = true
			playedUCI := mv.String()
			playedSAN := b.MoveToSAN(mv)

			inBook := false
			for _, em := range expectedMoves {
				if strings.EqualFold(em.UCI, playedUCI) || strings.EqualFold(em.SAN, playedSAN) {
					inBook = true
					break
				}
			}

			if inBook {
				matchedBookPlies++
				sanHistory = append(sanHistory, formatPlySAN(plyIdx+1, playedSAN))
				b.MakeMove(mv)
				continue
			}

			moveNum := (plyIdx / 2) + 1
			lineBefore := strings.Join(sanHistory, " ")

			if isUserTurn {
				deviatedByUser = true
				devKey := coreBefore + "|" + strings.ToLower(playedUCI)
				rec, exists := userDevMap[devKey]
				if !exists {
					primaryBook := expectedMoves[0]
					evalBook, evalPlayed, evalLoss, punishUCI, punishSAN := evaluateDeviationImpact(b, primaryBook, mv, repColor)
					advice := buildAvoidanceAdvice(moveNum, lineBefore, playedSAN, expectedMoves, evalLoss, punishSAN)
					rec = &RecurringBookDeviation{
						Key:               devKey,
						FEN:               fenBefore,
						CoreFEN:           coreBefore,
						Ply:               plyIdx + 1,
						MoveNumber:        moveNum,
						LineBeforeSAN:     lineBefore,
						PlayedMoveUCI:     playedUCI,
						PlayedMoveSAN:     playedSAN,
						ExpectedMoves:     expectedMoves,
						PrimaryBookUCI:    primaryBook.UCI,
						PrimaryBookSAN:    primaryBook.SAN,
						BookComment:       primaryBook.Comment,
						EvalBookCp:        evalBook,
						EvalPlayedCp:      evalPlayed,
						EvalLossCp:        evalLoss,
						PunishingReplyUCI: punishUCI,
						PunishingReplySAN: punishSAN,
						AvoidanceAdvice:   advice,
						SampleGames:       make([]DeviationIncident, 0),
					}
					userDevMap[devKey] = rec
				}

				rec.Occurrences++
				switch outcome {
				case "win":
					rec.Wins++
				case "draw":
					rec.Draws++
				case "loss":
					rec.Losses++
				}

				if len(rec.SampleGames) < 10 {
					rec.SampleGames = append(rec.SampleGames, DeviationIncident{
						GameIndex:   gameIdx + 1,
						White:       whiteName,
						Black:       blackName,
						Opponent:    opponentName,
						Event:       g.Tags["Event"],
						Site:        g.Tags["Site"],
						Date:        g.Tags["Date"],
						Result:      resStr,
						UserOutcome: outcome,
						Ply:         plyIdx + 1,
						MoveNumber:  moveNum,
						PlayedSAN:   playedSAN,
						BookSAN:     rec.PrimaryBookSAN,
					})
				}
				break
			} else {
				// Opponent deviated from the book's covered responses
				if matchedBookPlies > 0 || repColor == chess.Black {
					deviatedByOpp = true
					sideKey := coreBefore + "|" + strings.ToLower(playedUCI)
					side, exists := oppSideMap[sideKey]
					if !exists {
						afterOppBoard := b.Clone()
						afterOppBoard.MakeMove(mv)
						afterFEN := afterOppBoard.FEN()
						recUCI, recSAN, recNextFEN := suggestBestUserReply(afterOppBoard, repColor)
						fullWithReply := strings.TrimSpace(lineBefore + " " + formatPlySAN(plyIdx+1, playedSAN))
						if recSAN != "" {
							fullWithReply = strings.TrimSpace(fullWithReply + " " + formatPlySAN(plyIdx+2, recSAN))
						}
						remAdvice := fmt.Sprintf("Opponents played %d...%s (%s) leaving your prepared book.", moveNum, playedSAN, playedUCI)
						if b.Turn == chess.White {
							remAdvice = fmt.Sprintf("Opponents played %d.%s (%s) leaving your prepared book.", moveNum, playedSAN, playedUCI)
						}
						if recSAN != "" {
							remAdvice += fmt.Sprintf(" Add %s to your book to cover this sideline.", recSAN)
						}

						side = &UncoveredOpponentSideline{
							Key:                 sideKey,
							FEN:                 fenBefore,
							CoreFEN:             coreBefore,
							AfterOpponentFEN:    afterFEN,
							Ply:                 plyIdx + 1,
							MoveNumber:          moveNum,
							LineBeforeSAN:       lineBefore,
							OpponentMoveUCI:     playedUCI,
							OpponentMoveSAN:     playedSAN,
							FullLineWithReply:   fullWithReply,
							RecommendedReplyUCI: recUCI,
							RecommendedReplySAN: recSAN,
							RecommendedNextFEN:  recNextFEN,
							RemediationAdvice:   remAdvice,
						}
						oppSideMap[sideKey] = side
					}

					side.Occurrences++
					switch outcome {
					case "win":
						side.Wins++
					case "draw":
						side.Draws++
					case "loss":
						side.Losses++
					}
				}
				break
			}
		}

		if enteredBook {
			report.GamesEnteredBook++
			if deviatedByUser {
				report.UserDeviatedGames++
			} else if deviatedByOpp {
				report.OpponentSidelineGames++
				report.CompliantGames++
			} else {
				report.CompliantGames++
			}
		}
	}

	if report.GamesEnteredBook > 0 {
		report.CompliancePct = math.Round((float64(report.CompliantGames)/float64(report.GamesEnteredBook))*1000.0) / 10.0
	}

	for _, dev := range userDevMap {
		if dev.Occurrences > 0 {
			score := float64(dev.Wins) + 0.5*float64(dev.Draws)
			dev.WinRatePct = math.Round((score/float64(dev.Occurrences))*1000.0) / 10.0
		}
		report.UserDeviations = append(report.UserDeviations, *dev)
	}

	sort.Slice(report.UserDeviations, func(i, j int) bool {
		a := report.UserDeviations[i]
		b := report.UserDeviations[j]
		if a.Occurrences != b.Occurrences {
			return a.Occurrences > b.Occurrences
		}
		if a.Losses != b.Losses {
			return a.Losses > b.Losses
		}
		return a.EvalLossCp > b.EvalLossCp
	})

	for _, side := range oppSideMap {
		if side.Occurrences > 0 {
			score := float64(side.Wins) + 0.5*float64(side.Draws)
			side.WinRatePct = math.Round((score/float64(side.Occurrences))*1000.0) / 10.0
		}
		report.OpponentSidelines = append(report.OpponentSidelines, *side)
	}

	sort.Slice(report.OpponentSidelines, func(i, j int) bool {
		a := report.OpponentSidelines[i]
		b := report.OpponentSidelines[j]
		if a.Occurrences != b.Occurrences {
			return a.Occurrences > b.Occurrences
		}
		return a.Losses > b.Losses
	})

	return report
}

func matchesAnyIdentifier(playerName string, identifiers []string) bool {
	pLower := strings.ToLower(strings.TrimSpace(playerName))
	if pLower == "" {
		return false
	}
	for _, id := range identifiers {
		idLower := strings.ToLower(strings.TrimSpace(id))
		if idLower == "" {
			continue
		}
		if pLower == idLower || strings.Contains(pLower, idLower) || strings.Contains(idLower, pLower) {
			return true
		}
	}
	return false
}

func classifyGameOutcome(result string, userColor chess.Color) string {
	res := strings.TrimSpace(result)
	switch res {
	case "1-0":
		if userColor == chess.White {
			return "win"
		}
		return "loss"
	case "0-1":
		if userColor == chess.Black {
			return "win"
		}
		return "loss"
	case "1/2-1/2":
		return "draw"
	default:
		return "draw"
	}
}

func formatPlySAN(ply int, san string) string {
	moveNum := (ply + 1) / 2
	if ply%2 == 1 {
		return fmt.Sprintf("%d.%s", moveNum, san)
	}
	return san
}

// evaluateDeviationImpact compares the book move against the played move using 2-ply tactical/positional evaluation.
func evaluateDeviationImpact(b *chess.Board, bookMove RepertoireMove, playedMove chess.Move, userColor chess.Color) (evalBook int, evalPlayed int, evalLoss int, punishUCI string, punishSAN string) {
	// Evaluate position after book move
	bBook := b.Clone()
	if bm, err := bBook.ParseUCI(bookMove.UCI); err == nil {
		bBook.MakeMove(bm)
		evalBook = evaluateAfterUserMove(bBook, userColor)
	}

	// Evaluate position after played move and find opponent's strongest punishing reply
	bPlayed := b.Clone()
	bPlayed.MakeMove(playedMove)
	oppMoves := bPlayed.GenerateLegalMoves()
	if len(oppMoves) == 0 {
		evalPlayed = staticEvalForColor(bPlayed, userColor)
	} else {
		worstForUser := 99999
		for _, om := range oppMoves {
			bReply := bPlayed.Clone()
			bReply.MakeMove(om)
			// Best user recapture/reply at ply 2
			score := bestUserReplyScore(bReply, userColor)
			if score < worstForUser {
				worstForUser = score
				punishUCI = om.String()
				punishSAN = bPlayed.MoveToSAN(om)
			}
		}
		evalPlayed = worstForUser
	}

	evalLoss = evalBook - evalPlayed
	if evalLoss < 0 {
		evalLoss = 0
	}
	return evalBook, evalPlayed, evalLoss, punishUCI, punishSAN
}

func evaluateAfterUserMove(bAfterUser *chess.Board, userColor chess.Color) int {
	oppMoves := bAfterUser.GenerateLegalMoves()
	if len(oppMoves) == 0 {
		return staticEvalForColor(bAfterUser, userColor)
	}
	worstForUser := 99999
	for _, om := range oppMoves {
		bReply := bAfterUser.Clone()
		bReply.MakeMove(om)
		score := bestUserReplyScore(bReply, userColor)
		if score < worstForUser {
			worstForUser = score
		}
	}
	return worstForUser
}

func bestUserReplyScore(bUserTurn *chess.Board, userColor chess.Color) int {
	userMoves := bUserTurn.GenerateLegalMoves()
	if len(userMoves) == 0 {
		if bUserTurn.InCheck(userColor) {
			return -10000
		}
		return 0
	}
	best := -99999
	for _, um := range userMoves {
		nb := bUserTurn.Clone()
		nb.MakeMove(um)
		sc := staticEvalForColor(nb, userColor)
		if sc > best {
			best = sc
		}
	}
	return best
}

func suggestBestUserReply(bUserTurn *chess.Board, userColor chess.Color) (bestUCI string, bestSAN string, nextFEN string) {
	legal := bUserTurn.GenerateLegalMoves()
	if len(legal) == 0 {
		return "", "", ""
	}
	bestScore := -99999
	var bestMove chess.Move
	for _, mv := range legal {
		nb := bUserTurn.Clone()
		nb.MakeMove(mv)
		sc := evaluateAfterUserMove(nb, userColor)
		if sc > bestScore {
			bestScore = sc
			bestMove = mv
		}
	}
	san := bUserTurn.MoveToSAN(bestMove)
	nb := bUserTurn.Clone()
	nb.MakeMove(bestMove)
	return bestMove.String(), san, CoreFEN(nb.FEN())
}

func staticEvalForColor(b *chess.Board, perspective chess.Color) int {
	whiteScore := 0
	for sq := 0; sq < 64; sq++ {
		p := b.Squares[sq]
		if p == chess.Empty {
			continue
		}
		val := pieceBaseCp(p.Type()) + centerControlBonus(chess.Square(sq), p.Type(), p.Color())
		if p.Color() == chess.White {
			whiteScore += val
		} else {
			whiteScore -= val
		}
	}
	if perspective == chess.Black {
		return -whiteScore
	}
	return whiteScore
}

func pieceBaseCp(pt chess.PieceType) int {
	switch pt {
	case chess.Pawn:
		return 100
	case chess.Knight:
		return 320
	case chess.Bishop:
		return 330
	case chess.Rook:
		return 500
	case chess.Queen:
		return 900
	default:
		return 0
	}
}

func centerControlBonus(sq chess.Square, pt chess.PieceType, col chess.Color) int {
	file := int(sq) % 8
	rank := int(sq) / 8
	if col == chess.Black {
		rank = 7 - rank
	}

	bonus := 0
	// Central files d, e (3, 4) and ranks 3, 4 (indices 2, 3, 4)
	if (file == 3 || file == 4) && (rank >= 2 && rank <= 4) {
		if pt == chess.Pawn {
			bonus += 25
		} else if pt == chess.Knight || pt == chess.Bishop {
			bonus += 20
		}
	} else if (file >= 2 && file <= 5) && (rank >= 2 && rank <= 5) {
		if pt == chess.Knight || pt == chess.Bishop {
			bonus += 12
		}
	}
	// Penalize undeveloped minor pieces on back rank
	if rank == 0 && (pt == chess.Knight || pt == chess.Bishop) {
		bonus -= 15
	}
	return bonus
}

func buildAvoidanceAdvice(moveNum int, lineBefore string, playedSAN string, expected []RepertoireMove, evalLoss int, punishSAN string) string {
	var bookSANs []string
	var primaryComment string
	for i, em := range expected {
		bookSANs = append(bookSANs, em.SAN)
		if i == 0 && em.Comment != "" {
			primaryComment = em.Comment
		}
	}
	bookStr := strings.Join(bookSANs, " or ")

	prefix := fmt.Sprintf("On move %d", moveNum)
	if lineBefore != "" {
		prefix = fmt.Sprintf("On move %d (after %s)", moveNum, lineBefore)
	}

	var msg strings.Builder
	fmt.Fprintf(&msg, "%s, you played %s instead of your book move %s.", prefix, playedSAN, bookStr)

	if evalLoss >= 80 && punishSAN != "" {
		fmt.Fprintf(&msg, " Avoid %s: it concedes %d cp and allows the opponent to respond with %s.", playedSAN, evalLoss, punishSAN)
	} else if punishSAN != "" {
		fmt.Fprintf(&msg, " After %s, the opponent can counter with %s.", playedSAN, punishSAN)
	}

	if primaryComment != "" {
		fmt.Fprintf(&msg, " Remember your book plan for %s: %s.", expected[0].SAN, primaryComment)
	} else {
		fmt.Fprintf(&msg, " Drill this position so %s becomes automatic.", expected[0].SAN)
	}

	return msg.String()
}
