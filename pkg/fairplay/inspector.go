package fairplay

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"newscid/pkg/chess"
	"newscid/pkg/database/scid"
)

var (
	clkTagRegex  = regexp.MustCompile(`(?i)\[%clk\s+([^\]]+)\]`)
	emtTagRegex  = regexp.MustCompile(`(?i)\[%emt\s+([^\]]+)\]`)
	evalTagRegex = regexp.MustCompile(`(?i)\[%eval\s+([^\]]+)\]`)
)

// GameFairPlayRecord holds objective forensic metrics for a single game played by the target player.
type GameFairPlayRecord struct {
	GameID                  int      `json:"gameId"`
	Date                    string   `json:"date"`
	Event                   string   `json:"event"`
	Site                    string   `json:"site"`
	TimeControlRaw          string   `json:"timeControlRaw"`
	TimeControlCategory     string   `json:"timeControlCategory"` // "Bullet" | "Blitz" | "Rapid" | "Classical"
	PlayerColor             string   `json:"playerColor"`         // "white" | "black"
	PlayerElo               int      `json:"playerElo"`
	OpponentName            string   `json:"opponentName"`
	OpponentElo             int      `json:"opponentElo"`
	Result                  string   `json:"result"`
	PlayerScore             float64  `json:"playerScore"` // 1.0, 0.5, 0.0
	TotalPlies              int      `json:"totalPlies"`
	NonBookMoves            int      `json:"nonBookMoves"`
	HasClockData            bool     `json:"hasClockData"`
	HasEvalData             bool     `json:"hasEvalData"`
	AvgMoveTimeSec          float64  `json:"avgMoveTimeSec"`
	MoveTimeStdDevSec       float64  `json:"moveTimeStdDevSec"`
	MoveTimeCV              float64  `json:"moveTimeCV"`              // StdDev / Mean; < 0.38 indicates flat robotic pacing
	RoboticCadencePct       float64  `json:"roboticCadencePct"`       // % of moves in narrow [2.5s, 8.5s] band
	DelayedRecaptureCount   int      `json:"delayedRecaptureCount"`   // Recaptures taking > 3.0s
	ACPL                    float64  `json:"acpl"`                    // Average Centipawn Loss on non-book moves
	Top1AgreementPct        float64  `json:"top1AgreementPct"`        // % of moves with <= 10cp loss
	CriticalAccuracyPct     float64  `json:"criticalAccuracyPct"`     // % accuracy in sharp/high-tension positions
	BlunderCount            int      `json:"blunderCount"`            // Moves with >= 180cp loss
	IntrinsicPerformanceElo int      `json:"intrinsicPerformanceElo"` // Ken Regan-style IPR estimate
	GameIssueScore          float64  `json:"gameIssueScore"`          // 0-100 anomaly index for this game
	Flags                   []string `json:"flags"`
}

// TimeControlIssueProfile aggregates multi-game and multi-day fair-play & consistency metrics for one time control category.
type TimeControlIssueProfile struct {
	Category                  string  `json:"category"` // "Bullet" | "Blitz" | "Rapid" | "Classical"
	GamesAnalyzed             int     `json:"gamesAnalyzed"`
	DaysActive                int     `json:"daysActive"`
	DaysWithElevatedIssue     int     `json:"daysWithElevatedIssue"`
	DailyConsistencyPct       float64 `json:"dailyConsistencyPct"` // % of active days showing anomaly in this time control
	WinRatePct                float64 `json:"winRatePct"`
	AvgOfficialElo            int     `json:"avgOfficialElo"`
	IntrinsicPerformanceElo   int     `json:"intrinsicPerformanceElo"`
	EloOverperformanceDelta   int     `json:"eloOverperformanceDelta"`
	AvgACPL                   float64 `json:"avgAcpl"`
	Top1AgreementPct          float64 `json:"top1AgreementPct"`
	CriticalAccuracyPct       float64 `json:"criticalAccuracyPct"`
	BlunderFreeGamePct        float64 `json:"blunderFreeGamePct"`
	AvgMoveTimeSec            float64 `json:"avgMoveTimeSec"`
	AvgMoveTimeCV             float64 `json:"avgMoveTimeCV"`
	RoboticCadenceGamePct     float64 `json:"roboticCadenceGamePct"`
	ConsistentIssueScore      float64 `json:"consistentIssueScore"`      // 0-100 longitudinal issue score for this time control
	ZScore                    float64 `json:"zScore"`                    // Statistical sigma score
	ConsistencyClassification string  `json:"consistencyClassification"` // "Normal Human Variance" | "Mild Anomaly" | "Elevated Consistent Issue" | "High Consistent Time-Control Anomaly"
	Summary                   string  `json:"summary"`
}

// DailyFairPlaySummary tracks everyday consistency for a player on a specific calendar date.
type DailyFairPlaySummary struct {
	Date                  string   `json:"date"`
	GamesCount            int      `json:"gamesCount"`
	TimeControls          []string `json:"timeControls"`
	PrimaryTimeControl    string   `json:"primaryTimeControl"`
	WinRatePct            float64  `json:"winRatePct"`
	AvgACPL               float64  `json:"avgAcpl"`
	Top1AgreementPct      float64  `json:"top1AgreementPct"`
	AvgMoveTimeCV         float64  `json:"avgMoveTimeCV"`
	IntrinsicPerformance  int      `json:"intrinsicPerformance"`
	DailyIssueScore       float64  `json:"dailyIssueScore"`
	PeakGameID            int      `json:"peakGameId"`
	Classification        string   `json:"classification"`
}

// CrossControlDivergence measures performance & pacing discrepancies between time controls (e.g. Rapid vs Bullet/Blitz).
type CrossControlDivergence struct {
	HasMultipleControls bool    `json:"hasMultipleControls"`
	BaselineControl     string  `json:"baselineControl"`
	SuspectControl      string  `json:"suspectControl"`
	IPRDelta            int     `json:"iprDelta"`
	ACPLDelta           float64 `json:"acplDelta"`
	CadenceCVDelta      float64 `json:"cadenceCvDelta"`
	DivergenceScore     float64 `json:"divergenceScore"` // 0-100
	Explanation         string  `json:"explanation"`
}

// PlayerFairPlayDossier is the complete longitudinal fair-play & time-control consistency report for a player.
type PlayerFairPlayDossier struct {
	PlayerName               string                    `json:"playerName"`
	TotalGamesAnalyzed       int                       `json:"totalGamesAnalyzed"`
	TotalDaysActive          int                       `json:"totalDaysActive"`
	OverallIssueScore        float64                   `json:"overallIssueScore"` // 0-100
	HighestRiskTimeControl   string                    `json:"highestRiskTimeControl"`
	HighestControlIssueScore float64                   `json:"highestControlIssueScore"`
	OverallClassification    string                    `json:"overallClassification"`
	TimeControlProfiles      []TimeControlIssueProfile `json:"timeControlProfiles"`
	DailyTimeline            []DailyFairPlaySummary    `json:"dailyTimeline"`
	CrossDivergence          CrossControlDivergence    `json:"crossDivergence"`
	FlaggedGames             []GameFairPlayRecord      `json:"flaggedGames"`
	AllGames                 []GameFairPlayRecord      `json:"allGames"`
	KeyFindings              []string                  `json:"keyFindings"`
	ReportMarkdown           string                    `json:"reportMarkdown"`
}

// TimeControlID is a compact integer constant representing time control buckets in hot paths.
type TimeControlID uint8

const (
	TCBullet TimeControlID = iota
	TCBlitz
	TCRapid
	TCClassical
	tcCount = 4
)

var tcNames = [tcCount]string{"Bullet", "Blitz", "Rapid", "Classical"}

func (tc TimeControlID) String() string {
	if tc < tcCount {
		return tcNames[tc]
	}
	return "Classical"
}

// ParseTimeControlID converts a category name to its integer constant.
func ParseTimeControlID(cat string) TimeControlID {
	switch cat {
	case "Bullet":
		return TCBullet
	case "Blitz":
		return TCBlitz
	case "Rapid":
		return TCRapid
	default:
		return TCClassical
	}
}

// ClassificationID is an integer constant for statistical anomaly classification levels.
type ClassificationID uint8

const (
	ClassNormal ClassificationID = iota
	ClassMild
	ClassElevated
	ClassHighAnomaly
)

var classNames = [4]string{
	"Normal Human Variance",
	"Mild / Situational Anomaly",
	"Elevated Consistent Issue",
	"High Consistent Time-Control Anomaly",
}

func (c ClassificationID) String() string {
	if int(c) < len(classNames) {
		return classNames[c]
	}
	return classNames[0]
}

// ClassifyTimeControlID maps PGN TimeControl headers and Event tags to an integer TimeControlID constant.
func ClassifyTimeControlID(tcHeader, eventHeader string) TimeControlID {
	evLower := strings.ToLower(eventHeader)
	if strings.Contains(evLower, "bullet") || strings.Contains(evLower, "ultrabullet") || strings.Contains(evLower, "hyperbullet") {
		return TCBullet
	}
	if strings.Contains(evLower, "blitz") {
		return TCBlitz
	}
	if strings.Contains(evLower, "rapid") {
		return TCRapid
	}
	if strings.Contains(evLower, "classical") || strings.Contains(evLower, "correspondence") || strings.Contains(evLower, "daily") {
		return TCClassical
	}

	tc := strings.TrimSpace(tcHeader)
	if tc == "" || tc == "-" || tc == "?" || strings.Contains(tc, "/") {
		return TCClassical
	}
	parts := strings.Split(tc, "+")
	baseSec, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return TCBlitz
	}
	incSec := 0
	if len(parts) > 1 {
		incSec, _ = strconv.Atoi(strings.TrimSpace(parts[1]))
	}
	estTotalSec := baseSec + 40*incSec
	switch {
	case estTotalSec < 180:
		return TCBullet
	case estTotalSec < 480:
		return TCBlitz
	case estTotalSec <= 1500:
		return TCRapid
	default:
		return TCClassical
	}
}

// ClassifyTimeControl maps PGN TimeControl headers and Event tags to "Bullet", "Blitz", "Rapid", or "Classical".
func ClassifyTimeControl(tcHeader, eventHeader string) string {
	return ClassifyTimeControlID(tcHeader, eventHeader).String()
}

// AnalyzePlayerFairPlay scans up to maxGames for playerName in db and builds a full longitudinal dossier
// across every time control and every active day.
func AnalyzePlayerFairPlay(db *scid.Database, playerName string, maxGames int) (*PlayerFairPlayDossier, error) {
	if db == nil || !db.IsOpen {
		return nil, fmt.Errorf("database is not open")
	}
	target := strings.TrimSpace(playerName)
	if target == "" {
		return nil, fmt.Errorf("player name is required")
	}
	if maxGames <= 0 {
		maxGames = 300
	}

	targetLower := strings.ToLower(target)
	totalInDB := db.NumGames()

	var games []*chess.Game
	var gameIDs []int

	// Scan newest games first (1-based gameNum) so recent platform downloads are prioritized
	for gameNum := totalInDB; gameNum >= 1 && len(games) < maxGames; gameNum-- {
		summary, err := db.GetGameSummary(gameNum)
		if err != nil {
			continue
		}
		wMatch := strings.Contains(strings.ToLower(summary.White), targetLower)
		bMatch := strings.Contains(strings.ToLower(summary.Black), targetLower)
		if !wMatch && !bMatch {
			continue
		}
		g, err := db.GetGame(gameNum)
		if err != nil || g == nil {
			continue
		}
		games = append(games, g)
		gameIDs = append(gameIDs, gameNum)
	}

	if len(games) == 0 {
		return nil, fmt.Errorf("no games found for player %q in the active database", playerName)
	}

	return BuildPlayerFairPlayDossier(target, games, gameIDs), nil
}

// BuildPlayerFairPlayDossier computes the complete multi-time-control and day-by-day fair-play analysis from parsed games.
func BuildPlayerFairPlayDossier(playerName string, games []*chess.Game, gameIDs []int) *PlayerFairPlayDossier {
	targetLower := strings.ToLower(strings.TrimSpace(playerName))
	records := make([]GameFairPlayRecord, 0, len(games))

	for i, g := range games {
		if g == nil {
			continue
		}
		gid := i + 1
		if i < len(gameIDs) && gameIDs[i] > 0 {
			gid = gameIDs[i]
		}
		rec, ok := analyzeSingleGameForPlayer(g, gid, targetLower)
		if ok && rec.NonBookMoves >= 4 {
			records = append(records, rec)
		}
	}

	dossier := &PlayerFairPlayDossier{
		PlayerName:         playerName,
		TotalGamesAnalyzed: len(records),
		AllGames:           records,
		KeyFindings:        make([]string, 0),
	}

	if len(records) == 0 {
		dossier.OverallClassification = "Insufficient Non-Book Games"
		return dossier
	}

	// Group by Calendar Date (Daily Timeline)
	byDate := make(map[string][]GameFairPlayRecord)
	for _, r := range records {
		d := normalizeDateStr(r.Date)
		byDate[d] = append(byDate[d], r)
	}

	dailyList := make([]DailyFairPlaySummary, 0, len(byDate))
	for dateStr, dayGames := range byDate {
		dailyList = append(dailyList, summarizeDailyGames(dateStr, dayGames))
	}
	sort.Slice(dailyList, func(i, j int) bool {
		return dailyList[i].Date > dailyList[j].Date
	})
	dossier.DailyTimeline = dailyList
	dossier.TotalDaysActive = len(dailyList)

	// Group by Time Control using integer TimeControlID array indexing (TCBullet..TCClassical)
	var byTC [tcCount][]GameFairPlayRecord
	for _, r := range records {
		tcID := ParseTimeControlID(r.TimeControlCategory)
		byTC[tcID] = append(byTC[tcID], r)
	}

	profiles := make([]TimeControlIssueProfile, 0, int(tcCount))
	for tcID := TimeControlID(0); tcID < tcCount; tcID++ {
		tcGames := byTC[tcID]
		if len(tcGames) == 0 {
			continue
		}
		prof := buildTimeControlProfile(tcID.String(), tcGames)
		profiles = append(profiles, prof)
	}

	// Compute Cross-Time-Control Divergence (e.g., Rapid vs Bullet/Blitz baseline)
	div := computeCrossControlDivergence(profiles)
	dossier.CrossDivergence = div

	// Boost the suspect time control's ConsistentIssueScore if there is a massive cross-time-control jump
	if div.HasMultipleControls && div.DivergenceScore >= 35 {
		for i := range profiles {
			if profiles[i].Category == div.SuspectControl {
				boost := (div.DivergenceScore - 30) * 0.25
				profiles[i].ConsistentIssueScore = round1(math.Min(99.0, profiles[i].ConsistentIssueScore+boost))
				profiles[i].ConsistencyClassification = classifyIssueScore(profiles[i].ConsistentIssueScore)
			}
		}
	}

	dossier.TimeControlProfiles = profiles

	// Identify Highest Risk Time Control & Overall Score
	bestTC := ""
	bestTCScore := 0.0
	weightedSum := 0.0
	totalWeight := 0.0
	for _, p := range profiles {
		if p.ConsistentIssueScore > bestTCScore {
			bestTCScore = p.ConsistentIssueScore
			bestTC = p.Category
		}
		w := float64(p.GamesAnalyzed)
		weightedSum += p.ConsistentIssueScore * w
		totalWeight += w
	}
	dossier.HighestRiskTimeControl = bestTC
	dossier.HighestControlIssueScore = round1(bestTCScore)

	overall := bestTCScore
	if totalWeight > 0 {
		avgScore := weightedSum / totalWeight
		// Weight peak time control 65% and overall average 35% because selective time-control cheating happens in 1 control
		overall = round1(0.65*bestTCScore + 0.35*avgScore)
	}
	dossier.OverallIssueScore = overall
	dossier.OverallClassification = classifyIssueScore(overall)

	// Collect Flagged Games (sorted by GameIssueScore descending)
	sortedGames := make([]GameFairPlayRecord, len(records))
	copy(sortedGames, records)
	sort.Slice(sortedGames, func(i, j int) bool {
		return sortedGames[i].GameIssueScore > sortedGames[j].GameIssueScore
	})
	for _, g := range sortedGames {
		if g.GameIssueScore >= 50.0 || len(g.Flags) > 0 {
			dossier.FlaggedGames = append(dossier.FlaggedGames, g)
		}
		if len(dossier.FlaggedGames) >= 25 {
			break
		}
	}
	if len(dossier.FlaggedGames) == 0 && len(sortedGames) > 0 {
		limit := 5
		if len(sortedGames) < limit {
			limit = len(sortedGames)
		}
		dossier.FlaggedGames = sortedGames[:limit]
	}

	// Synthesize Key Findings & Official Markdown Report
	dossier.KeyFindings = buildKeyFindings(dossier)
	dossier.ReportMarkdown = generateFairPlayMarkdown(dossier)

	return dossier
}

func analyzeSingleGameForPlayer(g *chess.Game, gameID int, targetLower string) (GameFairPlayRecord, bool) {
	whiteName := g.Tag("White")
	blackName := g.Tag("Black")
	isWhite := strings.Contains(strings.ToLower(whiteName), targetLower)
	isBlack := strings.Contains(strings.ToLower(blackName), targetLower)
	if !isWhite && !isBlack {
		return GameFairPlayRecord{}, false
	}

	playerColor := chess.White
	playerColorStr := "white"
	opponentName := blackName
	playerElo, _ := strconv.Atoi(g.Tag("WhiteElo"))
	opponentElo, _ := strconv.Atoi(g.Tag("BlackElo"))
	if !isWhite && isBlack {
		playerColor = chess.Black
		playerColorStr = "black"
		opponentName = whiteName
		playerElo, _ = strconv.Atoi(g.Tag("BlackElo"))
		opponentElo, _ = strconv.Atoi(g.Tag("WhiteElo"))
	}
	if playerElo <= 0 {
		playerElo = 1500
	}
	if opponentElo <= 0 {
		opponentElo = playerElo
	}

	resStr := g.Tag("Result")
	playerScore := 0.5
	switch resStr {
	case "1-0":
		if playerColor == chess.White {
			playerScore = 1.0
		} else {
			playerScore = 0.0
		}
	case "0-1":
		if playerColor == chess.Black {
			playerScore = 1.0
		} else {
			playerScore = 0.0
		}
	}

	tcRaw := g.Tag("TimeControl")
	eventStr := g.Tag("Event")
	tcCat := ClassifyTimeControl(tcRaw, eventStr)
	incSec := parseIncrementSeconds(tcRaw)

	// Traverse mainline to extract clocks, evaluations, and board complexity
	board := chess.NewBoard()
	startFEN := strings.TrimSpace(g.StartFEN)
	if startFEN == "" || startFEN == "?" {
		startFEN = strings.TrimSpace(g.Tag("FEN"))
	}
	if startFEN != "" && startFEN != "?" && startFEN != chess.StartingFEN {
		if parsed, err := chess.ParseFEN(startFEN); err == nil {
			board = parsed
		}
	}

	var moveTimes []float64
	var cpLosses []float64
	var criticalLosses []float64
	delayedRecaptures := 0
	top1Count := 0
	criticalTop1Count := 0
	blunders := 0
	hasClock := false
	hasEval := false

	prevWhiteClk := -1.0
	prevBlackClk := -1.0
	prevEvalWhiteCp := 20.0
	prevHasEval := false
	prevWasCapture := false

	node := g.RootMove
	ply := 0
	nonBookMoves := 0

	for node != nil {
		ply++
		turnColor := board.Turn

		isCapture := strings.Contains(node.SAN, "x")
		complexity := estimateBoardComplexity(board)

		// Extract clock time spent
		var spentSec float64 = -1
		if node.EMT != "" {
			if emt, ok := parseRawClockSeconds(node.EMT); ok {
				spentSec = emt
				hasClock = true
			}
		} else if emt, ok := parseTagSeconds(emtTagRegex, node.Comment); ok {
			spentSec = emt
			hasClock = true
		} else {
			var clk float64
			var okClk bool
			if node.Clock != "" {
				clk, okClk = parseRawClockSeconds(node.Clock)
			}
			if !okClk {
				clk, okClk = parseTagSeconds(clkTagRegex, node.Comment)
			}
			if okClk {
				hasClock = true
				if turnColor == chess.White {
					if prevWhiteClk >= 0 {
						spentSec = prevWhiteClk - clk + float64(incSec)
					}
					prevWhiteClk = clk
				} else {
					if prevBlackClk >= 0 {
						spentSec = prevBlackClk - clk + float64(incSec)
					}
					prevBlackClk = clk
				}
			}
		}
		if spentSec < 0 && hasClock {
			spentSec = 0
		}

		currEvalWhiteCp, currHasEval := parseEvalWhiteCp(node.Comment)
		if currHasEval {
			hasEval = true
		}

		// Compute heuristic 2-ply tactical + positional move loss BEFORE mutating board if [%eval] is absent
		var heurLossCp float64
		var heurIsTop1 bool
		if turnColor == playerColor && ply > 10 && (!currHasEval || !prevHasEval) {
			heurLossCp, heurIsTop1 = evaluateMoveLossHeuristic(board, node, ply, playerElo, playerScore)
		}

		// Advance board with the played move
		board.MakeMove(node.Move)

		// Record metrics for target player on non-book plies (ply > 10)
		if turnColor == playerColor && ply > 10 {
			nonBookMoves++
			if hasClock && spentSec >= 0 && spentSec < 1800 {
				moveTimes = append(moveTimes, spentSec)
				if prevWasCapture && isCapture && spentSec >= 3.2 {
					delayedRecaptures++
				}
			}

			var lossCp float64
			var isTop1 bool
			if currHasEval && prevHasEval {
				if turnColor == chess.White {
					lossCp = prevEvalWhiteCp - currEvalWhiteCp
				} else {
					lossCp = currEvalWhiteCp - prevEvalWhiteCp
				}
				if lossCp < 0 {
					lossCp = 0
				}
				if lossCp > 500 {
					lossCp = 500
				}
				isTop1 = lossCp <= 12.0
			} else {
				lossCp = heurLossCp
				isTop1 = heurIsTop1
			}

			cpLosses = append(cpLosses, lossCp)
			if isTop1 {
				top1Count++
			}
			if lossCp >= 160.0 {
				blunders++
			}
			if complexity >= 10 {
				criticalLosses = append(criticalLosses, lossCp)
				if isTop1 || lossCp <= 15.0 {
					criticalTop1Count++
				}
			}
		}

		if currHasEval {
			prevEvalWhiteCp = currEvalWhiteCp
			prevHasEval = true
		} else {
			prevHasEval = false
		}
		prevWasCapture = isCapture
		node = node.Next
	}

	avgTime, stdTime, cvTime, roboticPct := computeCadenceStats(moveTimes)
	acpl := meanFloat64(cpLosses)
	top1Pct := 0.0
	if len(cpLosses) > 0 {
		top1Pct = round1(100.0 * float64(top1Count) / float64(len(cpLosses)))
	}
	critAccPct := top1Pct
	if len(criticalLosses) > 0 {
		critAccPct = round1(100.0 * float64(criticalTop1Count) / float64(len(criticalLosses)))
	}

	hasValidClock := hasClock && len(moveTimes) >= 4
	if !hasValidClock {
		avgTime = 0
		stdTime = 0
		cvTime = 0
		roboticPct = 0
	}

	// Estimate Ken Regan-style Intrinsic Performance Rating (IPR) from ACPL + Critical Accuracy + Game Outcome
	ipr := estimateIPR(acpl, critAccPct, playerElo, playerScore, opponentElo, hasEval, blunders, nonBookMoves)

	// Calculate composite GameIssueScore (0-100) and diagnostic flags
	issueScore, flags := scoreSingleGameAnomaly(
		tcCat,
		playerElo,
		ipr,
		playerScore,
		acpl,
		top1Pct,
		critAccPct,
		blunders,
		nonBookMoves,
		hasEval,
		hasValidClock,
		avgTime,
		cvTime,
		roboticPct,
		delayedRecaptures,
	)

	return GameFairPlayRecord{
		GameID:                  gameID,
		Date:                    normalizeDateStr(g.Tag("Date")),
		Event:                   eventStr,
		Site:                    g.Tag("Site"),
		TimeControlRaw:          tcRaw,
		TimeControlCategory:     tcCat,
		PlayerColor:             playerColorStr,
		PlayerElo:               playerElo,
		OpponentName:            opponentName,
		OpponentElo:             opponentElo,
		Result:                  resStr,
		PlayerScore:             playerScore,
		TotalPlies:              ply,
		NonBookMoves:            nonBookMoves,
		HasClockData:            hasValidClock,
		HasEvalData:             hasEval,
		AvgMoveTimeSec:          round2(avgTime),
		MoveTimeStdDevSec:       round2(stdTime),
		MoveTimeCV:              round2(cvTime),
		RoboticCadencePct:       round1(roboticPct),
		DelayedRecaptureCount:   delayedRecaptures,
		ACPL:                    round1(acpl),
		Top1AgreementPct:        top1Pct,
		CriticalAccuracyPct:     critAccPct,
		BlunderCount:            blunders,
		IntrinsicPerformanceElo: ipr,
		GameIssueScore:          round1(issueScore),
		Flags:                   flags,
	}, true
}

func buildTimeControlProfile(category string, games []GameFairPlayRecord) TimeControlIssueProfile {
	n := len(games)
	if n == 0 {
		return TimeControlIssueProfile{Category: category}
	}

	var (
		winsSum         float64
		eloSum          int
		iprSum          int
		acplSum         float64
		top1Sum         float64
		critSum         float64
		blunderFreeCnt  int
		timeSum         float64
		cvSum           float64
		clockGames      int
		evalGames       int
		roboticGamesCnt int
		issueSum        float64
	)

	dayIssues := make(map[string][]float64)
	for _, g := range games {
		winsSum += g.PlayerScore
		eloSum += g.PlayerElo
		iprSum += g.IntrinsicPerformanceElo
		acplSum += g.ACPL
		top1Sum += g.Top1AgreementPct
		critSum += g.CriticalAccuracyPct
		if g.BlunderCount == 0 {
			blunderFreeCnt++
		}
		if g.HasEvalData {
			evalGames++
		}
		if g.HasClockData {
			clockGames++
			timeSum += g.AvgMoveTimeSec
			cvSum += g.MoveTimeCV
			if g.MoveTimeCV > 0 && g.MoveTimeCV <= 0.38 && g.RoboticCadencePct >= 60.0 {
				roboticGamesCnt++
			}
		}
		issueSum += g.GameIssueScore
		dayIssues[g.Date] = append(dayIssues[g.Date], g.GameIssueScore)
	}

	daysActive := len(dayIssues)
	daysElevated := 0
	for _, scores := range dayIssues {
		if meanFloat64(scores) >= 48.0 {
			daysElevated++
		}
	}
	dailyConsistencyPct := 0.0
	if daysActive > 0 {
		dailyConsistencyPct = round1(100.0 * float64(daysElevated) / float64(daysActive))
	}

	avgElo := eloSum / n
	avgIPR := iprSum / n
	eloDelta := avgIPR - avgElo
	avgACPL := round1(acplSum / float64(n))
	avgTop1 := round1(top1Sum / float64(n))
	avgCrit := round1(critSum / float64(n))
	blunderFreePct := round1(100.0 * float64(blunderFreeCnt) / float64(n))

	avgMoveTime := 0.0
	avgCV := 0.0
	roboticGamePct := 0.0
	if clockGames > 0 {
		avgMoveTime = round2(timeSum / float64(clockGames))
		avgCV = round2(cvSum / float64(clockGames))
		roboticGamePct = round1(100.0 * float64(roboticGamesCnt) / float64(clockGames))
	}

	baseAvgIssue := issueSum / float64(n)
	// Multi-game & multi-day consistency bonus: if elevated across multiple days/games in this time control
	consistencyBonus := 0.0
	if n >= 3 && dailyConsistencyPct >= 50.0 {
		consistencyBonus += (dailyConsistencyPct - 40.0) * 0.22
	}
	if clockGames >= 2 && roboticGamePct >= 50.0 {
		consistencyBonus += (roboticGamePct - 40.0) * 0.18
	}
	if eloDelta > 280 && (evalGames > 0 || clockGames > 0 || (winsSum/float64(n)) >= 0.65) {
		consistencyBonus += math.Min(16.0, float64(eloDelta-280)*0.035)
	}

	consistentScore := round1(math.Min(99.0, math.Max(0.0, baseAvgIssue+consistencyBonus)))
	zScore := round2((consistentScore - 24.0) / 15.0)
	if zScore < -2.0 {
		zScore = -2.0
	}

	classification := classifyIssueScore(consistentScore)
	cvStr := "no clock data"
	if clockGames > 0 {
		cvStr = fmt.Sprintf("Move-Time CV %.2f (%.1fs avg)", avgCV, avgMoveTime)
	}
	summary := fmt.Sprintf(
		"%s (%d games across %d day(s)): Official %d vs. %d IPR (%+d), %.1f ACPL, %.1f%% Top-1 match, %s — %s (Score %.1f/100, Z=%+.2fσ).",
		category, n, daysActive, avgElo, avgIPR, eloDelta, avgACPL, avgTop1, cvStr, classification, consistentScore, zScore,
	)

	return TimeControlIssueProfile{
		Category:                  category,
		GamesAnalyzed:             n,
		DaysActive:                daysActive,
		DaysWithElevatedIssue:     daysElevated,
		DailyConsistencyPct:       dailyConsistencyPct,
		WinRatePct:                round1(100.0 * winsSum / float64(n)),
		AvgOfficialElo:            avgElo,
		IntrinsicPerformanceElo:   avgIPR,
		EloOverperformanceDelta:   eloDelta,
		AvgACPL:                   avgACPL,
		Top1AgreementPct:          avgTop1,
		CriticalAccuracyPct:       avgCrit,
		BlunderFreeGamePct:        blunderFreePct,
		AvgMoveTimeSec:            avgMoveTime,
		AvgMoveTimeCV:             avgCV,
		RoboticCadenceGamePct:     roboticGamePct,
		ConsistentIssueScore:      consistentScore,
		ZScore:                    zScore,
		ConsistencyClassification: classification,
		Summary:                   summary,
	}
}

func summarizeDailyGames(dateStr string, games []GameFairPlayRecord) DailyFairPlaySummary {
	n := len(games)
	if n == 0 {
		return DailyFairPlaySummary{Date: dateStr}
	}
	tcCount := make(map[string]int)
	var wins, acplSum, top1Sum, cvSum, issueSum float64
	var iprSum, clockCnt int
	peakScore := -1.0
	peakID := games[0].GameID

	for _, g := range games {
		tcCount[g.TimeControlCategory]++
		wins += g.PlayerScore
		acplSum += g.ACPL
		top1Sum += g.Top1AgreementPct
		iprSum += g.IntrinsicPerformanceElo
		issueSum += g.GameIssueScore
		if g.HasClockData {
			cvSum += g.MoveTimeCV
			clockCnt++
		}
		if g.GameIssueScore > peakScore {
			peakScore = g.GameIssueScore
			peakID = g.GameID
		}
	}

	var tcs []string
	primaryTC := ""
	bestCnt := 0
	for tc, cnt := range tcCount {
		tcs = append(tcs, tc)
		if cnt > bestCnt {
			bestCnt = cnt
			primaryTC = tc
		}
	}
	sort.Strings(tcs)

	avgCV := 0.0
	if clockCnt > 0 {
		avgCV = round2(cvSum / float64(clockCnt))
	}
	dailyScore := round1(issueSum / float64(n))

	return DailyFairPlaySummary{
		Date:                 dateStr,
		GamesCount:           n,
		TimeControls:         tcs,
		PrimaryTimeControl:   primaryTC,
		WinRatePct:           round1(100.0 * wins / float64(n)),
		AvgACPL:              round1(acplSum / float64(n)),
		Top1AgreementPct:     round1(top1Sum / float64(n)),
		AvgMoveTimeCV:        avgCV,
		IntrinsicPerformance: iprSum / n,
		DailyIssueScore:      dailyScore,
		PeakGameID:           peakID,
		Classification:       classifyIssueScore(dailyScore),
	}
}

func computeCrossControlDivergence(profiles []TimeControlIssueProfile) CrossControlDivergence {
	if len(profiles) < 2 {
		return CrossControlDivergence{
			HasMultipleControls: false,
			Explanation:         "Only one time control present in sample; import additional time controls (Bullet, Blitz, Rapid) to measure cross-control divergence.",
		}
	}

	// Pick lowest-issue (or fastest) control as human baseline
	baselineIdx := 0
	for i := 1; i < len(profiles); i++ {
		if profiles[i].ConsistentIssueScore < profiles[baselineIdx].ConsistentIssueScore ||
			(profiles[i].ConsistentIssueScore == profiles[baselineIdx].ConsistentIssueScore &&
				profiles[i].EloOverperformanceDelta < profiles[baselineIdx].EloOverperformanceDelta) {
			baselineIdx = i
		}
	}

	// Pick highest-issue distinct control as suspect comparison control
	suspectIdx := -1
	for i := 0; i < len(profiles); i++ {
		if i == baselineIdx {
			continue
		}
		if suspectIdx == -1 ||
			profiles[i].ConsistentIssueScore > profiles[suspectIdx].ConsistentIssueScore ||
			(profiles[i].ConsistentIssueScore == profiles[suspectIdx].ConsistentIssueScore &&
				profiles[i].EloOverperformanceDelta > profiles[suspectIdx].EloOverperformanceDelta) {
			suspectIdx = i
		}
	}
	if suspectIdx == -1 {
		suspectIdx = 0
	}

	base := profiles[baselineIdx]
	susp := profiles[suspectIdx]

	iprDelta := susp.IntrinsicPerformanceElo - base.IntrinsicPerformanceElo
	acplDelta := round1(base.AvgACPL - susp.AvgACPL) // Positive means suspect control has lower centipawn loss
	cvDelta := 0.0
	if base.AvgMoveTimeSec > 0 && susp.AvgMoveTimeSec > 0 {
		cvDelta = round2(base.AvgMoveTimeCV - susp.AvgMoveTimeCV)
	}

	score := 0.0
	if iprDelta > 150 {
		score += math.Min(45.0, float64(iprDelta-150)*0.09)
	}
	if acplDelta > 10.0 {
		score += math.Min(30.0, (acplDelta-10.0)*1.1)
	}
	if cvDelta > 0.20 {
		score += math.Min(25.0, (cvDelta-0.15)*50.0)
	}
	score = round1(math.Min(100.0, math.Max(0.0, score)))

	var explanation string
	if score < 15.0 {
		explanation = fmt.Sprintf(
			"Consistent human performance profile across time controls (%s IPR %d / %.1f ACPL vs. %s baseline IPR %d / %.1f ACPL).",
			susp.Category, susp.IntrinsicPerformanceElo, susp.AvgACPL, base.Category, base.IntrinsicPerformanceElo, base.AvgACPL,
		)
	} else {
		explanation = fmt.Sprintf(
			"Comparing %s vs. %s baseline: Intrinsic Performance jumps by %+d Elo (%d in %s vs. %d in %s), ACPL drops by %.1f cp (%.1f vs. %.1f), and Move-Time CV shifts by %+.2f.",
			susp.Category, base.Category, iprDelta, susp.IntrinsicPerformanceElo, susp.Category, base.IntrinsicPerformanceElo, base.Category,
			acplDelta, susp.AvgACPL, base.AvgACPL, -cvDelta,
		)
	}

	return CrossControlDivergence{
		HasMultipleControls: true,
		BaselineControl:     base.Category,
		SuspectControl:      susp.Category,
		IPRDelta:            iprDelta,
		ACPLDelta:           acplDelta,
		CadenceCVDelta:      cvDelta,
		DivergenceScore:     score,
		Explanation:         explanation,
	}
}

func scoreSingleGameAnomaly(
	tcCat string,
	playerElo, ipr int,
	playerScore float64,
	acpl, top1Pct, critAccPct float64,
	blunders, nonBookMoves int,
	hasEval, hasClock bool,
	avgTime, cvTime, roboticPct float64,
	delayedRecaptures int,
) (float64, []string) {
	score := 8.0
	var flags []string

	eloDiff := ipr - playerElo

	// Move Accuracy & ACPL vs Official Rating
	if hasEval {
		// Engine-annotated moves ([%eval] tags present)
		if acpl <= 14.0 && nonBookMoves >= 12 {
			score += 34.0
			flags = append(flags, fmt.Sprintf("Ultra-low engine ACPL (%.1f cp) over %d non-book moves", acpl, nonBookMoves))
		} else if acpl <= 21.0 && nonBookMoves >= 12 {
			score += 18.0
		}
		if critAccPct >= 88.0 && nonBookMoves >= 10 && acpl <= 22.0 {
			score += 14.0
			flags = append(flags, fmt.Sprintf("%.0f%% critical-position accuracy in high-tension middlegame", critAccPct))
		}
	} else {
		// Heuristic tactical/positional screening when [%eval] tags are absent
		if acpl <= 15.0 && blunders == 0 && playerScore >= 0.75 && nonBookMoves >= 16 {
			score += 22.0
			flags = append(flags, fmt.Sprintf("Near-zero tactical/positional loss (%.1f cp) over %d moves", acpl, nonBookMoves))
		} else if acpl <= 21.0 && blunders == 0 && playerScore == 1.0 && nonBookMoves >= 16 {
			score += 12.0
		}
	}

	if eloDiff >= 400 && (hasEval || playerScore >= 0.75) {
		score += math.Min(26.0, float64(eloDiff-300)*0.045)
		flags = append(flags, fmt.Sprintf("Intrinsic Performance %d (+%d above %d rating)", ipr, eloDiff, playerElo))
	}

	if blunders == 0 && nonBookMoves >= 22 && acpl <= 20.0 && playerScore >= 0.5 {
		score += 8.0
	}

	// Move-Time Cadence ("Robotic Pacing" / External Consultation Signature)
	if hasClock {
		if cvTime > 0 && cvTime <= 0.35 && avgTime >= 2.2 {
			score += 28.0
			flags = append(flags, fmt.Sprintf("Robotic flat move-time cadence (%.1fs ± %.2f CV, %.0f%% in narrow band)", avgTime, cvTime, roboticPct))
		} else if cvTime > 0 && cvTime <= 0.45 && roboticPct >= 65.0 {
			score += 16.0
			flags = append(flags, fmt.Sprintf("Low move-time variance (CV %.2f, %.0f%% moves in 2.5s–8.5s band)", cvTime, roboticPct))
		}
		if delayedRecaptures >= 2 {
			score += 8.0
			flags = append(flags, fmt.Sprintf("%d obvious recaptures delayed >3.2s (tab-switch signature)", delayedRecaptures))
		}
	}

	if score > 99.0 {
		score = 99.0
	}
	return score, flags
}

func estimateIPR(acpl, critAccPct float64, officialElo int, score float64, oppElo int, hasEval bool, blunders int, nonBookMoves int) int {
	if oppElo <= 0 {
		oppElo = officialElo
	}
	if officialElo <= 0 {
		officialElo = 1500
	}
	perfResultElo := float64(oppElo) + (score-0.5)*650.0

	if hasEval {
		// Empirical Regan/Lichess mapping when engine [%eval] annotations are present
		rawIPR := 3250.0 - (acpl * 31.0) + ((critAccPct - 50.0) * 3.5)
		if rawIPR > 3350 {
			rawIPR = 3350
		}
		if rawIPR < 750 {
			rawIPR = 750
		}
		blended := 0.78*rawIPR + 0.22*perfResultElo
		return int(math.Round(blended))
	}

	// When no [%eval] tags are embedded in the PGN, combine 2-ply tactical/positional move quality
	// with empirical opponent-adjusted performance rating so normal human games are never inflated to 3000+.
	expectedACPLForElo := math.Max(14.0, 58.0-float64(officialElo-1000)*0.022)
	acplDelta := expectedACPLForElo - acpl
	blunderPenalty := float64(blunders) * 28.0
	moveQualityElo := float64(officialElo) + acplDelta*8.5 + (critAccPct-45.0)*2.0 - blunderPenalty
	if moveQualityElo < float64(officialElo)-350 {
		moveQualityElo = float64(officialElo) - 350
	}
	if moveQualityElo > float64(officialElo)+650 {
		moveQualityElo = float64(officialElo) + 650
	}
	blended := 0.50*moveQualityElo + 0.50*perfResultElo
	if blended < 800 {
		blended = 800
	}
	if blended > 3100 {
		blended = 3100
	}
	return int(math.Round(blended))
}

// evaluateMoveLossHeuristic evaluates a move in an unannotated game by comparing it against all legal
// candidate moves using Piece-Square Tables (PST), king safety, development, and 1-ply opponent tactical
// capture replies (hanging pieces / SEE).
func evaluateMoveLossHeuristic(b *chess.Board, node *chess.MoveNode, ply int, playerElo int, playerScore float64) (lossCp float64, isTop1 bool) {
	if b == nil || node == nil {
		return 25.0, false
	}
	mover := b.Turn
	legalMoves := b.GenerateLegalMoves()
	if len(legalMoves) <= 1 {
		return 0.0, true
	}

	playedScore := evaluateCandidateMove(b, node.Move, mover)
	bestScore := playedScore
	strictlyBetter := 0

	for _, cand := range legalMoves {
		sc := evaluateCandidateMove(b, cand, mover)
		if sc > bestScore {
			bestScore = sc
		}
		if cand != node.Move && sc > playedScore+3.5 {
			strictlyBetter++
		}
	}

	tacticalDiff := bestScore - playedScore
	if tacticalDiff < 0 {
		tacticalDiff = 0
	}
	if tacticalDiff > 450 {
		tacticalDiff = 450
	}

	isTop1 = strictlyBetter == 0 && tacticalDiff <= 8.0
	return tacticalDiff, isTop1
}

func evaluateCandidateMove(b *chess.Board, m chess.Move, mover chess.Color) float64 {
	next := b.Clone()
	next.MakeMove(m)
	base := evaluatePositionForColor(next, mover)
	oppColor := chess.White
	if mover == chess.White {
		oppColor = chess.Black
	}
	// Subtract opponent's best immediate net capture gain (hanging piece / unfavorable trade detection)
	oppGain := bestCaptureGain(next, oppColor)
	return base - oppGain*0.85
}

func bestCaptureGain(b *chess.Board, side chess.Color) float64 {
	if b.Turn != side {
		return 0
	}
	moves := b.GenerateLegalMoves()
	bestGain := 0.0
	for _, m := range moves {
		target := b.Squares[m.To]
		if target == chess.Empty && m.Promo == chess.None {
			continue
		}
		valCaptured := pieceValueCp(target.Type())
		if m.Promo != chess.None {
			valCaptured += pieceValueCp(m.Promo) - 100.0
		}
		if valCaptured <= 0 {
			continue
		}
		attackerVal := pieceValueCp(b.Squares[m.From].Type())
		// Check if opponent can recapture on m.To
		afterCap := b.Clone()
		afterCap.MakeMove(m)
		recaptured := false
		oppMoves := afterCap.GenerateLegalMoves()
		for _, rm := range oppMoves {
			if rm.To == m.To {
				recaptured = true
				break
			}
		}
		gain := valCaptured
		if recaptured {
			gain = valCaptured - attackerVal
		}
		if gain > bestGain {
			bestGain = gain
		}
	}
	return bestGain
}

func pieceValueCp(pt chess.PieceType) float64 {
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
	case chess.King:
		return 20000
	default:
		return 0
	}
}

func evaluatePositionForColor(b *chess.Board, perspective chess.Color) float64 {
	wCp := evaluateBoardHeuristicWhiteCp(b)
	if perspective == chess.Black {
		return -wCp
	}
	return wCp
}

func evaluateBoardHeuristicWhiteCp(b *chess.Board) float64 {
	if b == nil {
		return 0
	}
	var score float64
	wBishops := 0
	bBishops := 0

	for sq := chess.Square(0); sq < 64; sq++ {
		p := b.Squares[sq]
		if p == chess.Empty {
			continue
		}
		file := int(sq) % 8
		rank := int(sq) / 8
		relRank := rank
		if p.Color() == chess.Black {
			relRank = 7 - rank
		}
		centerDist := math.Abs(float64(file)-3.5) + math.Abs(float64(rank)-3.5)

		val := pieceValueCp(p.Type())
		switch p.Type() {
		case chess.Pawn:
			// Advancement + central pawn bonus
			val += float64(relRank-1) * 8.0
			if (file == 3 || file == 4) && (relRank == 3 || relRank == 4) {
				val += 18.0
			}
		case chess.Knight:
			// Knights strongly prefer central squares and dislike the rim
			val += (4.5 - centerDist) * 7.5
			if relRank == 0 {
				val -= 12.0 // Undeveloped back-rank penalty
			}
		case chess.Bishop:
			val += (4.5 - centerDist) * 4.5
			if relRank == 0 {
				val -= 10.0
			}
			if p.Color() == chess.White {
				wBishops++
			} else {
				bBishops++
			}
		case chess.Rook:
			if relRank == 6 {
				val += 22.0 // 7th rank bonus
			}
			if file == 3 || file == 4 {
				val += 8.0
			}
		case chess.Queen:
			val += (4.5 - centerDist) * 3.0
		case chess.King:
			val = 0 // Do not add 20000 king material to static balance
			// Castled king safety bonus on g1/c1 or g8/c8
			if relRank == 0 && (file == 6 || file == 2) {
				val += 28.0
			} else if relRank >= 2 {
				val -= 25.0
			}
		}

		if p.Color() == chess.White {
			score += val
		} else {
			score -= val
		}
	}
	if wBishops >= 2 {
		score += 25.0
	}
	if bBishops >= 2 {
		score -= 25.0
	}
	return score
}

func estimateBoardComplexity(b *chess.Board) int {
	if b == nil {
		return 8
	}
	nonPawnPieces := 0
	for sq := chess.Square(0); sq < 64; sq++ {
		p := b.Squares[sq]
		if p != chess.Empty && p.Type() != chess.Pawn && p.Type() != chess.King {
			nonPawnPieces++
		}
	}
	return nonPawnPieces
}

func computeCadenceStats(times []float64) (mean, stdDev, cv, roboticPct float64) {
	if len(times) == 0 {
		return 0, 0, 0.85, 0
	}
	mean = meanFloat64(times)
	if mean <= 0.05 {
		return mean, 0, 0.85, 0
	}
	var sqDiffSum float64
	narrowBandCnt := 0
	for _, t := range times {
		d := t - mean
		sqDiffSum += d * d
		if t >= 2.3 && t <= 8.8 {
			narrowBandCnt++
		}
	}
	stdDev = math.Sqrt(sqDiffSum / float64(len(times)))
	cv = stdDev / mean
	roboticPct = 100.0 * float64(narrowBandCnt) / float64(len(times))
	return mean, stdDev, cv, roboticPct
}

func meanFloat64(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	var sum float64
	for _, v := range vals {
		sum += v
	}
	return sum / float64(len(vals))
}

func normalizeDateStr(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" || strings.Contains(s, "?") {
		return "Unknown-Date"
	}
	return strings.ReplaceAll(s, ".", "-")
}

func round1(v float64) float64 {
	return math.Round(v*10.0) / 10.0
}

func round2(v float64) float64 {
	return math.Round(v*100.0) / 100.0
}

// ClassifyIssueScoreID returns the compact integer ClassificationID for a 0-100 issue score.
func ClassifyIssueScoreID(score float64) ClassificationID {
	switch {
	case score >= 75.0:
		return ClassHighAnomaly
	case score >= 55.0:
		return ClassElevated
	case score >= 38.0:
		return ClassMild
	default:
		return ClassNormal
	}
}

func classifyIssueScore(score float64) string {
	return ClassifyIssueScoreID(score).String()
}

func buildKeyFindings(d *PlayerFairPlayDossier) []string {
	var findings []string
	for _, p := range d.TimeControlProfiles {
		findings = append(findings, p.Summary)
	}
	if d.CrossDivergence.HasMultipleControls && d.CrossDivergence.DivergenceScore >= 25.0 {
		findings = append(findings, "Cross-Control Divergence: "+d.CrossDivergence.Explanation)
	}
	if len(d.FlaggedGames) > 0 {
		top := d.FlaggedGames[0]
		findings = append(findings, fmt.Sprintf(
			"Peak Anomaly Game #%d (%s, %s vs %s): Score %.1f/100, %.1f ACPL, %d IPR, Move-Time CV %.2f.",
			top.GameID, top.Date, top.TimeControlCategory, top.OpponentName, top.GameIssueScore, top.ACPL, top.IntrinsicPerformanceElo, top.MoveTimeCV,
		))
	}
	return findings
}

func generateFairPlayMarkdown(d *PlayerFairPlayDossier) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Longitudinal Fair-Play & Time-Control Issue Report: %s\n\n", d.PlayerName))
	sb.WriteString(fmt.Sprintf("- **Games Analyzed:** %d across %d active calendar day(s)\n", d.TotalGamesAnalyzed, d.TotalDaysActive))
	sb.WriteString(fmt.Sprintf("- **Highest-Risk Time Control:** %s (Consistent Issue Score: **%.1f / 100**)\n", d.HighestRiskTimeControl, d.HighestControlIssueScore))
	sb.WriteString(fmt.Sprintf("- **Overall Statistical Classification:** %s (**%.1f / 100**)\n\n", d.OverallClassification, d.OverallIssueScore))

	sb.WriteString("## 1. Per-Time-Control Consistent Issue Matrix\n")
	sb.WriteString("| Time Control | Games | Days | Official Elo | Intrinsic IPR | ACPL | Top-1 % | Move-Time CV | Issue Score | Z-Score |\n")
	sb.WriteString("|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|\n")
	for _, p := range d.TimeControlProfiles {
		sb.WriteString(fmt.Sprintf(
			"| **%s** | %d | %d | %d | %d (%+d) | %.1f | %.1f%% | %.2f | **%.1f** | %+.2fσ |\n",
			p.Category, p.GamesAnalyzed, p.DaysActive, p.AvgOfficialElo, p.IntrinsicPerformanceElo,
			p.EloOverperformanceDelta, p.AvgACPL, p.Top1AgreementPct, p.AvgMoveTimeCV, p.ConsistentIssueScore, p.ZScore,
		))
	}
	sb.WriteString("\n")

	if d.CrossDivergence.HasMultipleControls {
		sb.WriteString("## 2. Cross-Time-Control Divergence\n")
		sb.WriteString(fmt.Sprintf("- %s (Divergence Index: **%.1f / 100**)\n\n", d.CrossDivergence.Explanation, d.CrossDivergence.DivergenceScore))
	}

	if len(d.FlaggedGames) > 0 {
		sb.WriteString("## 3. Top Flagged Games\n")
		for i, g := range d.FlaggedGames {
			if i >= 8 {
				break
			}
			sb.WriteString(fmt.Sprintf(
				"- **Game #%d (%s • %s vs. %s)**: Issue Score **%.1f/100** | ACPL %.1f | IPR %d | Top-1 %.1f%% | Move-Time CV %.2f %v\n",
				g.GameID, g.Date, g.TimeControlCategory, g.OpponentName, g.GameIssueScore, g.ACPL, g.IntrinsicPerformanceElo, g.Top1AgreementPct, g.MoveTimeCV, g.Flags,
			))
		}
	}
	return sb.String()
}

func parseIncrementSeconds(tcRaw string) int {
	parts := strings.Split(strings.TrimSpace(tcRaw), "+")
	if len(parts) > 1 {
		inc, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
		return inc
	}
	return 0
}

func parseTagSeconds(re *regexp.Regexp, comment string) (float64, bool) {
	if comment == "" {
		return 0, false
	}
	m := re.FindStringSubmatch(comment)
	if len(m) < 2 {
		return 0, false
	}
	return parseRawClockSeconds(m[1])
}

func parseRawClockSeconds(raw string) (float64, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, false
	}
	parts := strings.Split(raw, ":")
	switch len(parts) {
	case 3:
		h, _ := strconv.ParseFloat(parts[0], 64)
		mins, _ := strconv.ParseFloat(parts[1], 64)
		secs, _ := strconv.ParseFloat(parts[2], 64)
		return h*3600 + mins*60 + secs, true
	case 2:
		mins, _ := strconv.ParseFloat(parts[0], 64)
		secs, _ := strconv.ParseFloat(parts[1], 64)
		return mins*60 + secs, true
	case 1:
		secs, err := strconv.ParseFloat(strings.TrimSuffix(parts[0], "s"), 64)
		return secs, err == nil
	}
	return 0, false
}

func parseEvalWhiteCp(comment string) (float64, bool) {
	if comment == "" {
		return 0, false
	}
	m := evalTagRegex.FindStringSubmatch(comment)
	if len(m) < 2 {
		return 0, false
	}
	valStr := strings.TrimSpace(m[1])
	if idx := strings.Index(valStr, ","); idx != -1 {
		valStr = valStr[:idx]
	}
	if strings.HasPrefix(valStr, "#") {
		if strings.HasPrefix(valStr, "#-") {
			return -900.0, true
		}
		return 900.0, true
	}
	pawns, err := strconv.ParseFloat(valStr, 64)
	if err != nil {
		return 0, false
	}
	cp := pawns * 100.0
	if cp > 900 {
		cp = 900
	}
	if cp < -900 {
		cp = -900
	}
	return cp, true
}


