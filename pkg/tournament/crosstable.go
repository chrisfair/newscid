package tournament

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"newscid/pkg/chess"
	"newscid/pkg/database/scid"
)

type rawPlayer struct {
	name          string
	elo           int
	wins          int
	draws         int
	losses        int
	points        float64
	games         []playerGame
	scoresAgainst map[string]float64
}

type playerGame struct {
	opponent     string
	opponentElo  int
	color        string // "w" or "b"
	resultStr    string // "1", "0", "½", "*"
	score        float64
	round        string
	roundNum     int
	gameID       int
}

// DiscoverEvents scans the database index and aggregates games into tournament event summaries.
func DiscoverEvents(db *scid.Database) ([]*EventSummary, error) {
	if db == nil || !db.IsOpen {
		return nil, fmt.Errorf("database is not open")
	}

	type eventKey struct {
		event string
		site  string
		year  int
	}

	type eventData struct {
		event         string
		site          string
		date          string
		year          int
		numGames      int
		players       map[string]struct{}
		maxRound      int
		gameSummaries []*scid.GameSummary
	}

	eventsMap := make(map[eventKey]*eventData)

	db.RLock()
	defer db.RUnlock()

	for i, entry := range db.Entries {
		if (entry.Flags & scid.FlagDelete) != 0 {
			continue
		}

		evName := strings.TrimSpace(db.Names.GetName(scid.NameEvent, entry.EventID))
		if evName == "" || evName == "?" {
			continue
		}

		siteName := strings.TrimSpace(db.Names.GetName(scid.NameSite, entry.SiteID))
		dateStr := strings.TrimSpace(entry.Date)
		wName := strings.TrimSpace(db.Names.GetName(scid.NamePlayer, entry.WhiteID))
		bName := strings.TrimSpace(db.Names.GetName(scid.NamePlayer, entry.BlackID))
		roundStr := strings.TrimSpace(db.Names.GetName(scid.NameRound, entry.RoundID))

		var year int
		if len(dateStr) >= 4 {
			fmt.Sscanf(dateStr[:4], "%d", &year)
		}

		k := eventKey{
			event: evName,
			site:  siteName,
			year:  year,
		}

		ed, exists := eventsMap[k]
		if !exists {
			ed = &eventData{
				event:   evName,
				site:    siteName,
				date:    dateStr,
				year:    year,
				players: make(map[string]struct{}),
			}
			eventsMap[k] = ed
		} else {
			if ed.date == "" || (dateStr != "" && dateStr < ed.date) {
				ed.date = dateStr
			}
			if ed.site == "" && siteName != "" {
				ed.site = siteName
			}
		}

		ed.numGames++
		if wName != "" {
			ed.players[wName] = struct{}{}
		}
		if bName != "" {
			ed.players[bName] = struct{}{}
		}

		rNum := parseRoundNumber(roundStr)
		if rNum > ed.maxRound {
			ed.maxRound = rNum
		}

		// Save minimal summary for type detection
		summary := &scid.GameSummary{
			ID:       i + 1,
			White:    wName,
			Black:    bName,
			Result:   entry.Result,
			Round:    roundStr,
			WhiteElo: int(entry.WhiteElo),
			BlackElo: int(entry.BlackElo),
		}
		ed.gameSummaries = append(ed.gameSummaries, summary)
	}

	var results []*EventSummary
	for _, ed := range eventsMap {
		numPlayers := len(ed.players)
		ttype := DetectTournamentType(numPlayers, ed.maxRound, ed.gameSummaries)

		results = append(results, &EventSummary{
			Event:      ed.event,
			Site:       ed.site,
			Date:       ed.date,
			Year:       ed.year,
			NumGames:   ed.numGames,
			NumPlayers: numPlayers,
			Rounds:     ed.maxRound,
			Type:       ttype,
		})
	}

	// Sort events: by Date descending, then by NumGames descending, then Name
	sort.Slice(results, func(i, j int) bool {
		if results[i].Date != results[j].Date {
			return results[i].Date > results[j].Date
		}
		if results[i].NumGames != results[j].NumGames {
			return results[i].NumGames > results[j].NumGames
		}
		return results[i].Event < results[j].Event
	})

	return results, nil
}

// CalculateCrosstable extracts games matching an event from an open database and computes the crosstable.
func CalculateCrosstable(db *scid.Database, eventName string, site string, date string) (*TournamentCrosstable, error) {
	if db == nil || !db.IsOpen {
		return nil, fmt.Errorf("database is not open")
	}

	criteria := scid.SearchCriteria{
		Event:         eventName,
		Site:          site,
		IgnoreDeleted: true,
	}

	games, _, err := db.GetGameSummaries(0, 10000, criteria)
	if err != nil {
		return nil, fmt.Errorf("querying tournament games: %w", err)
	}

	return BuildCrosstableFromGames(games, eventName, site, date), nil
}

// BuildCrosstableFromGames computes full tournament standings, tiebreaks, matrix, and ASCII formatting.
func BuildCrosstableFromGames(games []*scid.GameSummary, eventName, site, date string) *TournamentCrosstable {
	ct := &TournamentCrosstable{
		Event: eventName,
		Site:  site,
		Date:  date,
	}

	// Filter matching games
	var matchedGames []*scid.GameSummary
	for _, g := range games {
		if g.Deleted {
			continue
		}
		if eventName != "" && !strings.EqualFold(strings.TrimSpace(g.Event), strings.TrimSpace(eventName)) {
			continue
		}
		matchedGames = append(matchedGames, g)
	}

	ct.NumGames = len(matchedGames)
	if len(matchedGames) == 0 {
		ct.Type = TypeRoundRobin
		ct.Standings = []*PlayerStanding{}
		ct.Matrix = [][]CrosstableCell{}
		ct.CrossTableText = "No games found for this tournament."
		return ct
	}

	if ct.Event == "" && len(matchedGames) > 0 {
		ct.Event = matchedGames[0].Event
	}
	if ct.Site == "" && len(matchedGames) > 0 {
		ct.Site = matchedGames[0].Site
	}
	if ct.Date == "" && len(matchedGames) > 0 {
		ct.Date = matchedGames[0].Date
	}

	playersMap := make(map[string]*rawPlayer)
	maxRound := 0

	for _, g := range matchedGames {
		wName := strings.TrimSpace(g.White)
		bName := strings.TrimSpace(g.Black)
		if wName == "" || bName == "" {
			continue
		}

		if _, exists := playersMap[wName]; !exists {
			playersMap[wName] = &rawPlayer{
				name:          wName,
				elo:           g.WhiteElo,
				scoresAgainst: make(map[string]float64),
			}
		} else if g.WhiteElo > playersMap[wName].elo {
			playersMap[wName].elo = g.WhiteElo
		}

		if _, exists := playersMap[bName]; !exists {
			playersMap[bName] = &rawPlayer{
				name:          bName,
				elo:           g.BlackElo,
				scoresAgainst: make(map[string]float64),
			}
		} else if g.BlackElo > playersMap[bName].elo {
			playersMap[bName].elo = g.BlackElo
		}

		rNum := parseRoundNumber(g.Round)
		if rNum > maxRound {
			maxRound = rNum
		}

		var wScore, bScore float64
		var wRes, bRes string

		switch g.Result {
		case chess.ResultWhiteWins:
			wScore, bScore = 1.0, 0.0
			wRes, bRes = "1", "0"
		case chess.ResultBlackWins:
			wScore, bScore = 0.0, 1.0
			wRes, bRes = "0", "1"
		case chess.ResultDraw:
			wScore, bScore = 0.5, 0.5
			wRes, bRes = "½", "½"
		default:
			wScore, bScore = 0.0, 0.0
			wRes, bRes = "*", "*"
		}

		wp := playersMap[wName]
		bp := playersMap[bName]

		wp.points += wScore
		wp.scoresAgainst[bName] += wScore
		wp.games = append(wp.games, playerGame{
			opponent:    bName,
			opponentElo: g.BlackElo,
			color:       "w",
			resultStr:   wRes,
			score:       wScore,
			round:       g.Round,
			roundNum:    rNum,
			gameID:      g.ID,
		})

		bp.points += bScore
		bp.scoresAgainst[wName] += bScore
		bp.games = append(bp.games, playerGame{
			opponent:    wName,
			opponentElo: g.WhiteElo,
			color:       "b",
			resultStr:   bRes,
			score:       bScore,
			round:       g.Round,
			roundNum:    rNum,
			gameID:      g.ID,
		})

		if g.Result == chess.ResultWhiteWins {
			wp.wins++
			bp.losses++
		} else if g.Result == chess.ResultBlackWins {
			bp.wins++
			wp.losses++
		} else if g.Result == chess.ResultDraw {
			wp.draws++
			bp.draws++
		}
	}

	ct.NumPlayers = len(playersMap)
	ct.NumRounds = maxRound

	// Calculate Average Elo and Category
	totalElo := 0
	eloCount := 0
	for _, p := range playersMap {
		if p.elo > 0 {
			totalElo += p.elo
			eloCount++
		}
	}
	if eloCount > 0 {
		ct.AverageElo = totalElo / eloCount
		ct.Category = CalculateFideCategory(ct.AverageElo)
	}

	// Tiebreak Calculations (Sonneborn-Berger, Buchholz, Buchholz Cut 1, TPR)
	type preliminaryStanding struct {
		raw *rawPlayer
		sb  float64
		bh  float64
		bh1 float64
		tpr int
	}

	var preStandings []*preliminaryStanding
	for _, p := range playersMap {
		var sb float64
		var bh float64
		var opponentScores []float64

		totalOppElo := 0
		ratedOppCount := 0
		scoreAgainstRated := 0.0

		for _, pg := range p.games {
			opp := playersMap[pg.opponent]
			if opp == nil {
				continue
			}

			// Sonneborn-Berger: score * opponent's total tournament points
			sb += pg.score * opp.points
			bh += opp.points
			opponentScores = append(opponentScores, opp.points)

			if pg.opponentElo > 0 {
				totalOppElo += pg.opponentElo
				ratedOppCount++
				scoreAgainstRated += pg.score
			}
		}

		// Buchholz Cut 1 (minus lowest opponent score)
		bh1 := bh
		if len(opponentScores) >= 2 {
			sort.Float64s(opponentScores)
			bh1 = bh - opponentScores[0]
		}

		avgOppElo := 0
		if ratedOppCount > 0 {
			avgOppElo = totalOppElo / ratedOppCount
		}
		tpr := CalculatePerformanceRating(scoreAgainstRated, ratedOppCount, avgOppElo)

		preStandings = append(preStandings, &preliminaryStanding{
			raw: p,
			sb:  sb,
			bh:  bh,
			bh1: bh1,
			tpr: tpr,
		})
	}

	// Sort standings by chess ranking criteria:
	// 1. Total Points (descending)
	// 2. Direct encounter (head-to-head) if 2 tied
	// 3. Sonneborn-Berger (descending)
	// 4. Buchholz (descending)
	// 5. Total Wins (descending)
	// 6. Name (ascending)
	sort.Slice(preStandings, func(i, j int) bool {
		pi := preStandings[i]
		pj := preStandings[j]

		if math.Abs(pi.raw.points-pj.raw.points) > 0.001 {
			return pi.raw.points > pj.raw.points
		}

		// Direct encounter
		directScoreI := pi.raw.scoresAgainst[pj.raw.name]
		directScoreJ := pj.raw.scoresAgainst[pi.raw.name]
		if math.Abs(directScoreI-directScoreJ) > 0.001 {
			return directScoreI > directScoreJ
		}

		// Sonneborn-Berger
		if math.Abs(pi.sb-pj.sb) > 0.001 {
			return pi.sb > pj.sb
		}

		// Buchholz
		if math.Abs(pi.bh-pj.bh) > 0.001 {
			return pi.bh > pj.bh
		}

		// Wins
		if pi.raw.wins != pj.raw.wins {
			return pi.raw.wins > pj.raw.wins
		}

		return pi.raw.name < pj.raw.name
	})

	// Name to Rank map
	nameToRank := make(map[string]int)
	for idx, ps := range preStandings {
		nameToRank[ps.raw.name] = idx + 1
	}

	ct.Standings = make([]*PlayerStanding, len(preStandings))
	for idx, ps := range preStandings {
		rank := idx + 1

		// Swiss round results sorted by round number
		var roundResults []*SwissRoundResult
		// Sort raw player games by roundNum
		sort.Slice(ps.raw.games, func(a, b int) bool {
			return ps.raw.games[a].roundNum < ps.raw.games[b].roundNum
		})

		for _, pg := range ps.raw.games {
			roundResults = append(roundResults, &SwissRoundResult{
				Round:        pg.round,
				OpponentRank: nameToRank[pg.opponent],
				OpponentName: pg.opponent,
				Color:        pg.color,
				Result:       pg.resultStr,
				Score:        pg.score,
				GameID:       pg.gameID,
			})
		}

		ct.Standings[idx] = &PlayerStanding{
			Rank:              rank,
			PlayerName:        ps.raw.name,
			Title:             extractTitle(ps.raw.name),
			Elo:               ps.raw.elo,
			Points:            ps.raw.points,
			Played:            len(ps.raw.games),
			Wins:              ps.raw.wins,
			Draws:             ps.raw.draws,
			Losses:            ps.raw.losses,
			SonnebornBerger:   math.Round(ps.sb*100) / 100,
			Buchholz:          math.Round(ps.bh*100) / 100,
			BuchholzCut1:      math.Round(ps.bh1*100) / 100,
			PerformanceRating: ps.tpr,
			RoundResults:      roundResults,
			ScoresAgainst:     ps.raw.scoresAgainst,
		}
	}

	// Detect tournament type
	ct.Type = DetectTournamentType(ct.NumPlayers, ct.NumRounds, matchedGames)

	// Build Round-Robin Matrix (NxN)
	n := len(ct.Standings)
	matrix := make([][]CrosstableCell, n)
	for i := 0; i < n; i++ {
		matrix[i] = make([]CrosstableCell, n)
		playerA := ct.Standings[i].PlayerName

		for j := 0; j < n; j++ {
			if i == j {
				matrix[i][j] = CrosstableCell{Result: "X"}
				continue
			}

			playerB := ct.Standings[j].PlayerName

			// Find games between playerA and playerB
			var resList []string
			var totalScore float64
			var primaryGameID int
			var primaryColor string

			for _, g := range matchedGames {
				if strings.EqualFold(g.White, playerA) && strings.EqualFold(g.Black, playerB) {
					if primaryGameID == 0 {
						primaryGameID = g.ID
						primaryColor = "w"
					}
					switch g.Result {
					case chess.ResultWhiteWins:
						resList = append(resList, "1")
						totalScore += 1.0
					case chess.ResultBlackWins:
						resList = append(resList, "0")
					case chess.ResultDraw:
						resList = append(resList, "½")
						totalScore += 0.5
					default:
						resList = append(resList, "*")
					}
				} else if strings.EqualFold(g.Black, playerA) && strings.EqualFold(g.White, playerB) {
					if primaryGameID == 0 {
						primaryGameID = g.ID
						primaryColor = "b"
					}
					switch g.Result {
					case chess.ResultWhiteWins:
						resList = append(resList, "0")
					case chess.ResultBlackWins:
						resList = append(resList, "1")
						totalScore += 1.0
					case chess.ResultDraw:
						resList = append(resList, "½")
						totalScore += 0.5
					default:
						resList = append(resList, "*")
					}
				}
			}

			resultStr := strings.Join(resList, " ")
			matrix[i][j] = CrosstableCell{
				Result: resultStr,
				Score:  totalScore,
				Color:  primaryColor,
				GameID: primaryGameID,
			}
		}
	}
	ct.Matrix = matrix

	// Format text crosstable
	ct.CrossTableText = formatTextCrosstable(ct)

	return ct
}

// DetectTournamentType checks game counts and pairing patterns to determine Swiss vs Round-Robin vs Match.
func DetectTournamentType(numPlayers int, rounds int, games []*scid.GameSummary) TournamentType {
	if numPlayers <= 2 && len(games) >= 1 {
		return TypeMatch
	}

	if numPlayers <= 0 {
		return TypeRoundRobin
	}

	// Check pairing graph
	pairings := make(map[string]int)
	for _, g := range games {
		w := strings.ToLower(strings.TrimSpace(g.White))
		b := strings.ToLower(strings.TrimSpace(g.Black))
		if w == "" || b == "" {
			continue
		}
		if w > b {
			w, b = b, w
		}
		pairings[w+"#"+b]++
	}

	possiblePairs := (numPlayers * (numPlayers - 1)) / 2
	actualPairs := len(pairings)

	// If at least 70% of all possible pairings occurred, it is a Round-Robin
	if possiblePairs > 0 && float64(actualPairs)/float64(possiblePairs) >= 0.70 {
		return TypeRoundRobin
	}

	return TypeSwiss
}

func parseRoundNumber(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}

	var digits []rune
	for _, r := range s {
		if unicode.IsDigit(r) {
			digits = append(digits, r)
		} else if len(digits) > 0 {
			break
		}
	}
	if len(digits) == 0 {
		return 0
	}
	n, _ := strconv.Atoi(string(digits))
	return n
}

func extractTitle(name string) string {
	nameUpper := strings.ToUpper(name)
	for _, t := range []string{"GM", "WGM", "IM", "WIM", "FM", "WFM", "CM", "WCM"} {
		if strings.HasPrefix(nameUpper, t+" ") || strings.Contains(nameUpper, "("+t+")") || strings.Contains(nameUpper, "["+t+"]") {
			return t
		}
	}
	return ""
}

func formatPoints(pts float64) string {
	if math.Mod(pts, 1.0) == 0 {
		return fmt.Sprintf("%.0f", pts)
	}
	return fmt.Sprintf("%.1f", pts)
}

func formatTextCrosstable(ct *TournamentCrosstable) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("Event: %s\n", ct.Event))
	if ct.Site != "" || ct.Date != "" {
		b.WriteString(fmt.Sprintf("Site:  %-28s Date: %s\n", ct.Site, ct.Date))
	}
	if ct.AverageElo > 0 {
		if ct.Category > 0 {
			b.WriteString(fmt.Sprintf("Category: %d (Avg Elo: %d) | Type: %s | Games: %d | Players: %d\n",
				ct.Category, ct.AverageElo, ct.Type, ct.NumGames, ct.NumPlayers))
		} else {
			b.WriteString(fmt.Sprintf("Avg Elo: %d | Type: %s | Games: %d | Players: %d\n",
				ct.AverageElo, ct.Type, ct.NumGames, ct.NumPlayers))
		}
	} else {
		b.WriteString(fmt.Sprintf("Type: %s | Games: %d | Players: %d\n",
			ct.Type, ct.NumGames, ct.NumPlayers))
	}

	if ct.Type == TypeRoundRobin && len(ct.Standings) <= 24 {
		// Round-Robin Matrix
		sepLen := 45 + len(ct.Standings)*4 + 20
		sep := strings.Repeat("-", sepLen)
		b.WriteString(sep + "\n")

		b.WriteString(fmt.Sprintf("%-3s  %-24s %-5s", "#", "Player", "Elo"))
		for i := 1; i <= len(ct.Standings); i++ {
			b.WriteString(fmt.Sprintf("%3d ", i))
		}
		b.WriteString(fmt.Sprintf(" %-5s %-6s %-5s\n", "Pts", "SB", "TPR"))
		b.WriteString(sep + "\n")

		for i, p := range ct.Standings {
			b.WriteString(fmt.Sprintf("%2d   %-24.24s ", p.Rank, p.PlayerName))
			if p.Elo > 0 {
				b.WriteString(fmt.Sprintf("%-5d", p.Elo))
			} else {
				b.WriteString(" ----")
			}

			for j := 0; j < len(ct.Standings); j++ {
				cell := ct.Matrix[i][j].Result
				if cell == "" {
					cell = "."
				}
				b.WriteString(fmt.Sprintf("%3s ", cell))
			}

			tprStr := "----"
			if p.PerformanceRating > 0 {
				tprStr = fmt.Sprintf("%d", p.PerformanceRating)
			}

			b.WriteString(fmt.Sprintf(" %4s %6.2f %5s\n",
				formatPoints(p.Points), p.SonnebornBerger, tprStr))
		}
		b.WriteString(sep + "\n")

	} else {
		// Swiss / Standings List
		sep := strings.Repeat("-", 76)
		b.WriteString(sep + "\n")
		b.WriteString(fmt.Sprintf("%-3s  %-26s %-5s %-6s %-6s %-6s %-6s %-5s\n",
			"#", "Player", "Elo", "Score", "Played", "Buch", "SB", "TPR"))
		b.WriteString(sep + "\n")

		for _, p := range ct.Standings {
			eloStr := "----"
			if p.Elo > 0 {
				eloStr = fmt.Sprintf("%d", p.Elo)
			}
			tprStr := "----"
			if p.PerformanceRating > 0 {
				tprStr = fmt.Sprintf("%d", p.PerformanceRating)
			}

			b.WriteString(fmt.Sprintf("%2d   %-26.26s %-5s %-6s %-6d %-6.1f %-6.2f %-5s\n",
				p.Rank, p.PlayerName, eloStr, formatPoints(p.Points),
				p.Played, p.Buchholz, p.SonnebornBerger, tprStr))
		}
		b.WriteString(sep + "\n")
	}

	return b.String()
}
