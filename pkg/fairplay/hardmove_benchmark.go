package fairplay

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"newscid/pkg/chess"
	"newscid/pkg/database/scid"
)

// HardMoveMoment represents a single position in a game where a human of that rating
// would find the engine's top choice 30% of the time or less (p <= 0.30).
type HardMoveMoment struct {
	Ply              int     `json:"ply"`
	MoveNumber       int     `json:"moveNumber"`
	Side             string  `json:"side"` // "self" | "opponent"
	PlayedSAN        string  `json:"playedSan"`
	BestSAN          string  `json:"bestSan"`
	FoundTopChoice   bool    `json:"foundTopChoice"`
	HumanProb        float64 `json:"humanProb"`        // p_i <= 0.30
	EvalLossCp       float64 `json:"evalLossCp"`       // Centipawn loss on this move
	EvalGapTop2Cp    float64 `json:"evalGapTop2Cp"`    // Gap between #1 and #2 moves
	PlausibleOptions int     `json:"plausibleOptions"` // Number of reasonable human candidate moves
}

// GameHardMovePerformance holds the Hard-Move Detector score for one side (Self or Opponent) in a single game.
type GameHardMovePerformance struct {
	GameID             int              `json:"gameId"`
	Date               string           `json:"date"`
	TimeControl        string           `json:"timeControl"`
	Side               string           `json:"side"` // "self" | "opponent"
	PlayerName         string           `json:"playerName"`
	AnonymousLabel     string           `json:"anonymousLabel"`
	PlayerElo          int              `json:"playerElo"`
	OpponentName       string           `json:"opponentName"`
	Result             string           `json:"result"`
	TotalNonBookMoves  int              `json:"totalNonBookMoves"`
	HardMoveCount      int              `json:"hardMoveCount"`      // Moves with p_i <= 0.30
	HardMovesFound     int              `json:"hardMovesFound"`     // K_top found out of HardMoveCount
	HardMoveHitRatePct float64          `json:"hardMoveHitRatePct"` // 100 * K_top / HardMoveCount
	ExpectedHardHits   float64          `json:"expectedHardHits"`   // Sum of p_i across hard moves
	TailProbability    float64          `json:"tailProbability"`    // Exact Poisson-Binomial P(X >= K_top)
	UnusualnessScore   float64          `json:"unusualnessScore"`   // -log10(P) + tie-breaker
	ZScore             float64          `json:"zScore"`             // (K_top - E) / sigma
	BandIndex          int              `json:"bandIndex"`          // 1..5 (4 & 5 are the Top Two Bands / Top 1%)
	BandName           string           `json:"bandName"`
	IsBannedOpponent   bool             `json:"isBannedOpponent"`
	BanSource          string           `json:"banSource,omitempty"` // e.g. "Chess.com Closed: Fair Play" or "Lichess TOS Violation"
	HardMoments        []HardMoveMoment `json:"hardMoments,omitempty"`
}

// BenchmarkBandSummary compares Self vs. Opponent game counts and percentages in one Unusualness Band.
type BenchmarkBandSummary struct {
	BandIndex        int     `json:"bandIndex"` // 1..5
	BandName         string  `json:"bandName"`
	Description      string  `json:"description"`
	ThresholdScore   float64 `json:"thresholdScore"`
	SelfCount        int     `json:"selfCount"`
	SelfPct          float64 `json:"selfPct"`
	OpponentCount    int     `json:"opponentCount"`
	OpponentPct      float64 `json:"opponentPct"`
	ExpectedOppCount float64 `json:"expectedOppCount"`
	ExcessOppCount   float64 `json:"excessOppCount"`
	BannedOppCount   int     `json:"bannedOppCount"`
	BannedOppPct     float64 `json:"bannedOppPct"`
}

// SelectiveCheatingPoint shows what percentage of games land in the Top 1% (Top Two Bands)
// and Top 5% when swapping k hard moves per game with the engine's top choice in your own games.
type SelectiveCheatingPoint struct {
	Label                string  `json:"label"`
	SwappedMovesSpec     string  `json:"swappedMovesSpec"`
	AvgSwappedPerGame    float64 `json:"avgSwappedPerGame"`
	TopTwoBandsCaughtPct float64 `json:"topTwoBandsCaughtPct"` // Caught in Top 1% benchmark
	TopFivePctCaughtPct  float64 `json:"topFivePctCaughtPct"`  // Caught in Top 5% benchmark
	AvgUnusualnessScore  float64 `json:"avgUnusualnessScore"`
}

// BannedValidationSummary summarizes how the detector performs on opponents confirmed banned/closed online.
type BannedValidationSummary struct {
	AccountsChecked       int     `json:"accountsChecked"`
	ConfirmedBannedCount  int     `json:"confirmedBannedCount"`
	GamesVsBanned         int     `json:"gamesVsBanned"`
	BannedInTopTwoBands   int     `json:"bannedInTopTwoBands"`
	BannedTopTwoRatePct   float64 `json:"bannedTopTwoRatePct"`
	OpenTopTwoRatePct     float64 `json:"openTopTwoRatePct"`
	MultiplierVsOpen      float64 `json:"multiplierVsOpen"`
	ConfirmedBannedHandle []string `json:"confirmedBannedHandles,omitempty"`
}

// HardMoveBenchmarkReport is the full "Man vs Chess" Self-Benchmarked Hard-Move Excess Dossier.
type HardMoveBenchmarkReport struct {
	BenchmarkPlayer           string                    `json:"benchmarkPlayer"`
	TimeControlFilter         string                    `json:"timeControlFilter"`
	HardMoveProbCutoff        float64                   `json:"hardMoveProbCutoff"` // 0.30 (30%)
	TotalGamesScanned         int                       `json:"totalGamesScanned"`
	SelfGamesQualified        int                       `json:"selfGamesQualified"`
	OpponentGamesQualified    int                       `json:"opponentGamesQualified"`
	TotalHardMovesAnalyzed    int                       `json:"totalHardMovesAnalyzed"`
	SelfAvgHardHitRatePct     float64                   `json:"selfAvgHardHitRatePct"`
	OpponentAvgHardHitRatePct float64                   `json:"opponentAvgHardHitRatePct"`
	SelfExpectedHitRatePct    float64                   `json:"selfExpectedHitRatePct"`
	OpponentExpectedHitRatePct float64                  `json:"opponentExpectedHitRatePct"`
	TopTwoBandThreshold       float64                   `json:"topTwoBandThreshold"` // 99th percentile of Self games
	TopHalfPctThreshold       float64                   `json:"topHalfPctThreshold"` // 99.5th percentile of Self games
	SelfInTopTwoBands         int                       `json:"selfInTopTwoBands"`
	SelfTopTwoRatePct         float64                   `json:"selfTopTwoRatePct"`
	OpponentInTopTwoBands     int                       `json:"opponentInTopTwoBands"`
	OpponentTopTwoRatePct     float64                   `json:"opponentTopTwoRatePct"`
	ExpectedOpponentTopTwo    float64                   `json:"expectedOpponentTopTwo"`
	ExcessOpponentGames       float64                   `json:"excessOpponentGames"`
	ExcessGamesLowEstimate    int                       `json:"excessGamesLowEstimate"`
	ExcessGamesHighEstimate   int                       `json:"excessGamesHighEstimate"`
	ExcessPctOfAllGames       float64                   `json:"excessPctOfAllGames"`
	ExcessPValue              float64                   `json:"excessPValue"`
	OddsOneIn                 int                       `json:"oddsOneIn"` // e.g. 1000 for "1 in 1,000"
	SelfInTop100Count         int                       `json:"selfInTop100Count"`
	OpponentInTop100Count     int                       `json:"opponentInTop100Count"`
	Bands                     []BenchmarkBandSummary    `json:"bands"`
	SelectiveSimulation       []SelectiveCheatingPoint  `json:"selectiveSimulation"`
	BannedValidation          BannedValidationSummary   `json:"bannedValidation"`
	Top100Games               []GameHardMovePerformance `json:"top100Games"`
	VerdictHeadline           string                    `json:"verdictHeadline"`
	VerdictFindings           []string                  `json:"verdictFindings"`
}

// AnalyzeHardMoveBenchmark runs the "Man vs Chess" Self-Benchmarked Hard-Move Excess Detector
// across up to maxGames for benchmarkPlayer in db.
func AnalyzeHardMoveBenchmark(
	db *scid.Database,
	benchmarkPlayer string,
	timeControlFilter string,
	maxGames int,
	checkOnlineBans bool,
) (*HardMoveBenchmarkReport, error) {
	if db == nil || !db.IsOpen {
		return nil, fmt.Errorf("database is not open")
	}
	target := strings.TrimSpace(benchmarkPlayer)
	if target == "" {
		return nil, fmt.Errorf("benchmark player name is required")
	}
	if maxGames <= 0 {
		maxGames = 2000
	}

	targetLower := strings.ToLower(target)
	tcFilter := strings.TrimSpace(timeControlFilter)
	if tcFilter == "" || strings.EqualFold(tcFilter, "ALL") {
		tcFilter = "ALL"
	}

	totalInDB := db.NumGames()
	var games []*chess.Game
	var gameIDs []int

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
		if tcFilter != "ALL" {
			tcCat := ClassifyTimeControl(g.Tag("TimeControl"), g.Tag("Event"))
			if !strings.EqualFold(tcCat, tcFilter) {
				continue
			}
		}
		games = append(games, g)
		gameIDs = append(gameIDs, gameNum)
	}

	if len(games) == 0 {
		return nil, fmt.Errorf("no matching games found for benchmark player %q", benchmarkPlayer)
	}

	return BuildHardMoveBenchmarkReport(target, tcFilter, games, gameIDs, checkOnlineBans, "", ""), nil
}

// BuildHardMoveBenchmarkReport computes the complete self-benchmarked hard-move excess analysis from parsed games.
func BuildHardMoveBenchmarkReport(
	benchmarkPlayer string,
	timeControlFilter string,
	games []*chess.Game,
	gameIDs []int,
	checkOnlineBans bool,
	chessComBaseURL string,
	lichessBaseURL string,
) *HardMoveBenchmarkReport {
	targetLower := strings.ToLower(strings.TrimSpace(benchmarkPlayer))
	if timeControlFilter == "" {
		timeControlFilter = "ALL"
	}

	var selfPerfs []GameHardMovePerformance
	var oppPerfs []GameHardMovePerformance
	totalHardMoves := 0

	var selfHardSum, selfFoundSum, selfExpSum float64
	var oppHardSum, oppFoundSum, oppExpSum float64

	anonMap := make(map[string]string)
	anonCounter := 0
	getAnonLabel := func(handle string) string {
		key := strings.ToLower(strings.TrimSpace(handle))
		if lbl, ok := anonMap[key]; ok {
			return lbl
		}
		anonCounter++
		lbl := fmt.Sprintf("Opponent #%03d", anonCounter)
		anonMap[key] = lbl
		return lbl
	}

	for i, g := range games {
		if g == nil {
			continue
		}
		gid := i + 1
		if i < len(gameIDs) && gameIDs[i] > 0 {
			gid = gameIDs[i]
		}

		sPerf, oPerf, ok := evaluateGameHardMovesPair(g, gid, targetLower)
		if !ok {
			continue
		}

		oPerf.AnonymousLabel = getAnonLabel(oPerf.PlayerName)
		sPerf.AnonymousLabel = "You (Benchmark)"

		if sPerf.HardMoveCount >= 2 {
			selfPerfs = append(selfPerfs, sPerf)
			totalHardMoves += sPerf.HardMoveCount
			selfHardSum += float64(sPerf.HardMoveCount)
			selfFoundSum += float64(sPerf.HardMovesFound)
			selfExpSum += sPerf.ExpectedHardHits
		}
		if oPerf.HardMoveCount >= 2 {
			oppPerfs = append(oppPerfs, oPerf)
			totalHardMoves += oPerf.HardMoveCount
			oppHardSum += float64(oPerf.HardMoveCount)
			oppFoundSum += float64(oPerf.HardMovesFound)
			oppExpSum += oPerf.ExpectedHardHits
		}
	}

	report := &HardMoveBenchmarkReport{
		BenchmarkPlayer:        benchmarkPlayer,
		TimeControlFilter:      timeControlFilter,
		HardMoveProbCutoff:     0.30,
		TotalGamesScanned:      len(games),
		SelfGamesQualified:     len(selfPerfs),
		OpponentGamesQualified: len(oppPerfs),
		TotalHardMovesAnalyzed: totalHardMoves,
	}

	if selfHardSum > 0 {
		report.SelfAvgHardHitRatePct = round1(100.0 * selfFoundSum / selfHardSum)
		report.SelfExpectedHitRatePct = round1(100.0 * selfExpSum / selfHardSum)
	}
	if oppHardSum > 0 {
		report.OpponentAvgHardHitRatePct = round1(100.0 * oppFoundSum / oppHardSum)
		report.OpponentExpectedHitRatePct = round1(100.0 * oppExpSum / oppHardSum)
	}

	if len(selfPerfs) == 0 {
		report.VerdictHeadline = "Insufficient games with Hard Moves (p <= 30%) to calibrate personal benchmark."
		return report
	}

	// Calibrate Empirical Benchmark Bands from Your Own Games
	selfScores := make([]float64, len(selfPerfs))
	for i, p := range selfPerfs {
		selfScores[i] = p.UnusualnessScore
	}
	sort.Float64s(selfScores)

	// Empirical percentiles on Self benchmark (with floor thresholds so tiny sample sizes still have sensible bands)
	p90 := math.Max(0.85, empiricalQuantile(selfScores, 0.90))
	p95 := math.Max(p90+0.18, empiricalQuantile(selfScores, 0.95))
	p99 := math.Max(p95+0.35, empiricalQuantile(selfScores, 0.99))
	p995 := math.Max(p99+0.30, empiricalQuantile(selfScores, 0.995))

	report.TopTwoBandThreshold = round2(p99)
	report.TopHalfPctThreshold = round2(p995)

	classifyBand := func(score float64) (int, string) {
		switch {
		case score >= p995:
			return 5, "Band 5: Extreme Outlier (Top 0.5%)"
		case score >= p99:
			return 4, "Band 4: High Unusualness (Top 0.5%–1.0%)"
		case score >= p95:
			return 3, "Band 3: Unusual (95th–99th %ile)"
		case score >= p90:
			return 2, "Band 2: Strong Game (90th–95th %ile)"
		default:
			return 1, "Band 1: Normal Human Range (<90th %ile)"
		}
	}

	for i := range selfPerfs {
		idx, name := classifyBand(selfPerfs[i].UnusualnessScore)
		selfPerfs[i].BandIndex = idx
		selfPerfs[i].BandName = name
	}
	for i := range oppPerfs {
		idx, name := classifyBand(oppPerfs[i].UnusualnessScore)
		oppPerfs[i].BandIndex = idx
		oppPerfs[i].BandName = name
	}

	// Optional Online Ban / Closed-Account Verification for Unique Opponents
	bannedHandles := make(map[string]string)
	checkedAccounts := 0
	if checkOnlineBans && len(oppPerfs) > 0 {
		checkedAccounts, bannedHandles = checkOpponentAccountsBanStatus(oppPerfs, chessComBaseURL, lichessBaseURL)
	}
	// Also check if any game tag explicitly flags a fair-play closure/ban
	for i := range oppPerfs {
		key := strings.ToLower(strings.TrimSpace(oppPerfs[i].PlayerName))
		if src, ok := bannedHandles[key]; ok {
			oppPerfs[i].IsBannedOpponent = true
			oppPerfs[i].BanSource = src
		}
	}

	// Aggregate 5-Band Distribution & Top-Two-Band Excess
	bandDefs := []struct {
		idx   int
		name  string
		desc  string
		thres float64
	}{
		{1, "Band 1: Normal (<90%)", "Everyday human games matching expected rating accuracy on hard moves", 0},
		{2, "Band 2: Sharp (90–95%)", "Strong tactical games (1 in 10 benchmark games)", round2(p90)},
		{3, "Band 3: Unusual (95–99%)", "Inspired streaks on hard moves (1 in 25 benchmark games)", round2(p95)},
		{4, "Band 4: Top 1% Band A", "Second highest unusualness band (Top 0.5%–1.0% of your benchmark)", round2(p99)},
		{5, "Band 5: Top 0.5% Band B", "Highest unusualness band — extreme engine-like hard-move precision", round2(p995)},
	}

	var selfBandCounts [6]int
	var oppBandCounts [6]int
	var bannedBandCounts [6]int
	gamesVsBanned := 0
	gamesVsOpen := 0
	openTopTwo := 0

	for _, p := range selfPerfs {
		selfBandCounts[p.BandIndex]++
	}
	for _, p := range oppPerfs {
		oppBandCounts[p.BandIndex]++
		if p.IsBannedOpponent {
			gamesVsBanned++
			bannedBandCounts[p.BandIndex]++
		} else {
			gamesVsOpen++
			if p.BandIndex >= 4 {
				openTopTwo++
			}
		}
	}

	nSelf := float64(len(selfPerfs))
	nOpp := float64(len(oppPerfs))

	bands := make([]BenchmarkBandSummary, 0, 5)
	for _, bd := range bandDefs {
		sCnt := selfBandCounts[bd.idx]
		oCnt := oppBandCounts[bd.idx]
		bCnt := bannedBandCounts[bd.idx]

		sPct := 0.0
		if nSelf > 0 {
			sPct = 100.0 * float64(sCnt) / nSelf
		}
		// If sample size is small and empirical top bands have 0 self games, use nominal benchmark rate (0.5% each for Band 4 & 5)
		nominalSelfRate := float64(sCnt) / math.Max(1.0, nSelf)
		if bd.idx == 4 && sCnt == 0 {
			nominalSelfRate = 0.005
		} else if bd.idx == 5 && sCnt == 0 {
			nominalSelfRate = 0.005
		}

		oPct := 0.0
		if nOpp > 0 {
			oPct = 100.0 * float64(oCnt) / nOpp
		}
		expOpp := nOpp * nominalSelfRate
		excess := float64(oCnt) - expOpp
		if excess < 0 {
			excess = 0
		}
		bPct := 0.0
		if gamesVsBanned > 0 {
			bPct = 100.0 * float64(bCnt) / float64(gamesVsBanned)
		}

		bands = append(bands, BenchmarkBandSummary{
			BandIndex:        bd.idx,
			BandName:         bd.name,
			Description:      bd.desc,
			ThresholdScore:   bd.thres,
			SelfCount:        sCnt,
			SelfPct:          round2(sPct),
			OpponentCount:    oCnt,
			OpponentPct:      round2(oPct),
			ExpectedOppCount: round1(expOpp),
			ExcessOppCount:   round1(excess),
			BannedOppCount:   bCnt,
			BannedOppPct:     round2(bPct),
		})
	}
	report.Bands = bands

	selfTop2 := selfBandCounts[4] + selfBandCounts[5]
	oppTop2 := oppBandCounts[4] + oppBandCounts[5]
	report.SelfInTopTwoBands = selfTop2
	report.OpponentInTopTwoBands = oppTop2
	if nSelf > 0 {
		report.SelfTopTwoRatePct = round2(100.0 * float64(selfTop2) / nSelf)
	}
	if nOpp > 0 {
		report.OpponentTopTwoRatePct = round2(100.0 * float64(oppTop2) / nOpp)
	}

	benchmarkTop2Rate := float64(selfTop2) / math.Max(1.0, nSelf)
	if benchmarkTop2Rate < 0.01 {
		benchmarkTop2Rate = 0.01 // Standard 1.0% top-two-band benchmark
	}
	expTop2 := nOpp * benchmarkTop2Rate
	excessTop2 := math.Max(0, float64(oppTop2)-expTop2)
	report.ExpectedOpponentTopTwo = round1(expTop2)
	report.ExcessOpponentGames = round1(excessTop2)

	// Confidence range for excess games (+/- 1 SD of Poisson/Binomial count)
	sdTop2 := math.Sqrt(math.Max(1.0, float64(oppTop2)))
	lowEst := int(math.Round(math.Max(0, excessTop2-0.85*sdTop2)))
	highEst := int(math.Round(excessTop2 + 0.85*sdTop2))
	if excessTop2 > 0 && highEst == 0 {
		highEst = 1
	}
	report.ExcessGamesLowEstimate = lowEst
	report.ExcessGamesHighEstimate = highEst
	if nOpp > 0 {
		report.ExcessPctOfAllGames = round2(100.0 * excessTop2 / nOpp)
	}

	pVal := poissonTailProb(oppTop2, expTop2)
	report.ExcessPValue = pVal
	if pVal > 0 && pVal < 1.0 {
		odds := int(math.Round(1.0 / pVal))
		if odds < 1 {
			odds = 1
		}
		if odds > 1000000 {
			odds = 1000000
		}
		report.OddsOneIn = odds
	} else {
		report.OddsOneIn = 1
	}

	// Banned Opponent Validation Summary
	bannedTop2 := bannedBandCounts[4] + bannedBandCounts[5]
	bannedRate := 0.0
	if gamesVsBanned > 0 {
		bannedRate = round2(100.0 * float64(bannedTop2) / float64(gamesVsBanned))
	}
	openRate := report.SelfTopTwoRatePct
	if gamesVsOpen > 0 {
		openRate = round2(100.0 * float64(openTopTwo) / float64(gamesVsOpen))
	}
	mult := 0.0
	if openRate > 0 && bannedRate > 0 {
		mult = round1(bannedRate / openRate)
	} else if bannedRate > 0 {
		mult = round1(bannedRate / 1.0)
	}
	var confirmedList []string
	for h := range bannedHandles {
		confirmedList = append(confirmedList, h)
	}
	sort.Strings(confirmedList)
	report.BannedValidation = BannedValidationSummary{
		AccountsChecked:       checkedAccounts,
		ConfirmedBannedCount:  len(confirmedList),
		GamesVsBanned:         gamesVsBanned,
		BannedInTopTwoBands:   bannedTop2,
		BannedTopTwoRatePct:   bannedRate,
		OpenTopTwoRatePct:     openRate,
		MultiplierVsOpen:      mult,
		ConfirmedBannedHandle: confirmedList,
	}

	// Selective Cheating Simulation (Swapping Engine Top Choices into Your Own Games)
	report.SelectiveSimulation = simulateSelectiveCheatingOnSelfGames(selfPerfs, p95, p99)

	// Top 100 Most Unusual Games Leaderboard (Both Self and Opponents)
	allCombined := make([]GameHardMovePerformance, 0, len(selfPerfs)+len(oppPerfs))
	allCombined = append(allCombined, selfPerfs...)
	allCombined = append(allCombined, oppPerfs...)
	sort.Slice(allCombined, func(i, j int) bool {
		if allCombined[i].UnusualnessScore == allCombined[j].UnusualnessScore {
			return allCombined[i].ZScore > allCombined[j].ZScore
		}
		return allCombined[i].UnusualnessScore > allCombined[j].UnusualnessScore
	})
	limit100 := 100
	if len(allCombined) < limit100 {
		limit100 = len(allCombined)
	}
	top100 := allCombined[:limit100]
	selfInTop100 := 0
	oppInTop100 := 0
	for _, item := range top100 {
		if item.Side == "self" {
			selfInTop100++
		} else {
			oppInTop100++
		}
	}
	report.SelfInTop100Count = selfInTop100
	report.OpponentInTop100Count = oppInTop100
	report.Top100Games = top100

	// Build "Man vs Chess" Verdict Headline & Key Findings
	buildHardMoveVerdict(report)

	return report
}

func evaluateGameHardMovesPair(
	g *chess.Game,
	gameID int,
	targetLower string,
) (selfPerf GameHardMovePerformance, oppPerf GameHardMovePerformance, ok bool) {
	whiteName := g.Tag("White")
	blackName := g.Tag("Black")
	isWhite := strings.Contains(strings.ToLower(whiteName), targetLower)
	isBlack := strings.Contains(strings.ToLower(blackName), targetLower)
	if !isWhite && !isBlack {
		return GameHardMovePerformance{}, GameHardMovePerformance{}, false
	}

	selfColor := chess.White
	selfName := whiteName
	oppName := blackName
	selfElo, _ := strconv.Atoi(g.Tag("WhiteElo"))
	oppElo, _ := strconv.Atoi(g.Tag("BlackElo"))
	if !isWhite && isBlack {
		selfColor = chess.Black
		selfName = blackName
		oppName = whiteName
		selfElo, _ = strconv.Atoi(g.Tag("BlackElo"))
		oppElo, _ = strconv.Atoi(g.Tag("WhiteElo"))
	}
	if selfElo <= 0 {
		selfElo = 1500
	}
	if oppElo <= 0 {
		oppElo = selfElo
	}

	tcCat := ClassifyTimeControl(g.Tag("TimeControl"), g.Tag("Event"))
	dateStr := normalizeDateStr(g.Tag("Date"))
	resStr := g.Tag("Result")

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

	selfPerf = GameHardMovePerformance{
		GameID:       gameID,
		Date:         dateStr,
		TimeControl:  tcCat,
		Side:         "self",
		PlayerName:   selfName,
		PlayerElo:    selfElo,
		OpponentName: oppName,
		Result:       resStr,
	}
	oppPerf = GameHardMovePerformance{
		GameID:       gameID,
		Date:         dateStr,
		TimeControl:  tcCat,
		Side:         "opponent",
		PlayerName:   oppName,
		PlayerElo:    oppElo,
		OpponentName: selfName,
		Result:       resStr,
	}

	// Check if PGN tags mention Fair Play closure (for offline tests / annotated PGNs)
	terminationLower := strings.ToLower(g.Tag("Termination") + " " + g.Tag("Annotator") + " " + g.Tag("Comment"))
	if strings.Contains(terminationLower, "fair play") || strings.Contains(terminationLower, "cheating") || strings.Contains(terminationLower, "tosviolation") {
		oppPerf.IsBannedOpponent = true
		oppPerf.BanSource = "PGN Termination / Fair-Play Tag"
	}

	prevEvalWhiteCp := 20.0
	prevHasEval := false
	prevWasCapture := false

	node := g.RootMove
	ply := 0

	for node != nil {
		ply++
		turnColor := board.Turn
		isSelf := turnColor == selfColor
		elo := oppElo
		if isSelf {
			elo = selfElo
		}

		currEvalWhiteCp, currHasEval := parseEvalWhiteCp(node.Comment)
		isCapture := strings.Contains(node.SAN, "x")

		if ply > 10 {
			if isSelf {
				selfPerf.TotalNonBookMoves++
			} else {
				oppPerf.TotalNonBookMoves++
			}

			isHard, humanProb, foundTop, lossCp, gapTop2, plausibleCnt, bestSAN := evaluateHardMoveCandidate(
				board,
				node,
				elo,
				prevHasEval && currHasEval,
				prevEvalWhiteCp,
				currEvalWhiteCp,
				prevWasCapture && isCapture,
			)

			if isHard {
				moment := HardMoveMoment{
					Ply:              ply,
					MoveNumber:       (ply + 1) / 2,
					Side:             map[bool]string{true: "self", false: "opponent"}[isSelf],
					PlayedSAN:        node.SAN,
					BestSAN:          bestSAN,
					FoundTopChoice:   foundTop,
					HumanProb:        round2(humanProb),
					EvalLossCp:       round1(lossCp),
					EvalGapTop2Cp:    round1(gapTop2),
					PlausibleOptions: plausibleCnt,
				}
				if isSelf {
					selfPerf.HardMoveCount++
					selfPerf.ExpectedHardHits += humanProb
					if foundTop {
						selfPerf.HardMovesFound++
					}
					if len(selfPerf.HardMoments) < 12 {
						selfPerf.HardMoments = append(selfPerf.HardMoments, moment)
					}
				} else {
					oppPerf.HardMoveCount++
					oppPerf.ExpectedHardHits += humanProb
					if foundTop {
						oppPerf.HardMovesFound++
					}
					if len(oppPerf.HardMoments) < 12 {
						oppPerf.HardMoments = append(oppPerf.HardMoments, moment)
					}
				}
			}
		}

		board.MakeMove(node.Move)
		if currHasEval {
			prevEvalWhiteCp = currEvalWhiteCp
			prevHasEval = true
		} else {
			prevHasEval = false
		}
		prevWasCapture = isCapture
		node = node.Next
	}

	finalizeHardMovePerformance(&selfPerf)
	finalizeHardMovePerformance(&oppPerf)
	return selfPerf, oppPerf, true
}

// evaluateHardMoveCandidate determines whether the current board position is a "Hard Move"
// (expected human probability of finding the engine's #1 choice is <= 30% at the player's Elo)
// and whether the played move matched the top choice.
func evaluateHardMoveCandidate(
	board *chess.Board,
	node *chess.MoveNode,
	playerElo int,
	hasExactEvalPair bool,
	prevEvalWhiteCp float64,
	currEvalWhiteCp float64,
	isImmediateRecapture bool,
) (isHard bool, humanProb float64, foundTop bool, lossCp float64, gapTop2Cp float64, plausibleCnt int, bestSAN string) {
	legalMoves := board.GenerateLegalMoves()
	nLegal := len(legalMoves)
	if nLegal <= 3 {
		return false, 0.90, true, 0, 0, nLegal, node.SAN
	}

	mover := board.Turn
	type scoredMove struct {
		m     chess.Move
		score float64
	}
	scored := make([]scoredMove, 0, nLegal)
	var playedHeurScore float64
	foundPlayed := false

	for _, lm := range legalMoves {
		sc := evaluateCandidateMove(board, lm, mover)
		scored = append(scored, scoredMove{m: lm, score: sc})
		if lm.From == node.Move.From && lm.To == node.Move.To && lm.Promo == node.Move.Promo {
			playedHeurScore = sc
			foundPlayed = true
		}
	}

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	bestMove := scored[0].m
	bestScore := scored[0].score
	secondScore := scored[1].score
	gapTop2Cp = math.Max(0, bestScore-secondScore)
	bestSAN = board.MoveToSAN(bestMove)

	// Count plausible human alternatives (moves within 75cp of the best move)
	plausibleCnt = 0
	for _, sm := range scored {
		if bestScore-sm.score <= 75.0 {
			plausibleCnt++
		}
	}
	if plausibleCnt < 1 {
		plausibleCnt = 1
	}

	// Obvious recaptures where #1 is >220cp better than #2 are easy moves (p > 0.75), not hard moves
	if isImmediateRecapture && gapTop2Cp > 200.0 {
		return false, 0.85, true, 0, gapTop2Cp, 1, bestSAN
	}

	// Estimate human probability p_i of finding the #1 move at playerElo
	// Factors that make a move HARD (lower p_i):
	// Many legal moves (nLegal >= 22) and several plausible competing candidates (plausibleCnt >= 3)
	// Quiet best move (not a check or obvious capture) or non-trivial tactical gap
	// Moderate player Elo
	eloFactor := (float64(playerElo) - 1500.0) / 2000.0 // e.g. 0.0 at 1500, +0.25 at 2000, -0.15 at 1200
	baseProb := 1.0 / math.Max(1.6, float64(plausibleCnt)*0.95+float64(nLegal)*0.055)

	bestPiece := board.Squares[bestMove.From]
	bestCaptured := board.Squares[bestMove.To]
	isQuietBest := bestCaptured == chess.Empty && bestMove.Promo == chess.None
	if isQuietBest {
		baseProb *= 0.72 // Quiet top choices in complex positions are harder for humans to spot
	}
	if bestPiece.Type() == chess.Knight || bestPiece.Type() == chess.Pawn {
		// Subtle knight/pawn maneuvers
		baseProb *= 0.90
	}
	if gapTop2Cp >= 35.0 && gapTop2Cp <= 165.0 {
		// Narrow tactical/positional needle where #1 is distinctly best but not an obvious queen hang
		baseProb *= 0.82
	}

	humanProb = baseProb + eloFactor*0.08
	if humanProb < 0.06 {
		humanProb = 0.06
	}
	if humanProb > 0.92 {
		humanProb = 0.92
	}

	// DetermineEvalLoss and whether the player found the top choice
	if hasExactEvalPair {
		if mover == chess.White {
			lossCp = prevEvalWhiteCp - currEvalWhiteCp
		} else {
			lossCp = currEvalWhiteCp - prevEvalWhiteCp
		}
		if lossCp < 0 {
			lossCp = 0
		}
		foundTop = lossCp <= 10.0
		if foundTop {
			bestSAN = node.SAN
		}
	} else {
		if !foundPlayed {
			playedHeurScore = bestScore - 35.0
		}
		lossCp = math.Max(0, bestScore-playedHeurScore)
		isExactBest := bestMove.From == node.Move.From && bestMove.To == node.Move.To && bestMove.Promo == node.Move.Promo
		foundTop = isExactBest || (lossCp <= 6.0 && gapTop2Cp <= 22.0)
	}

	// Hard Move criterion from the video: human probability of finding Stockfish's top choice is <= 30% (0.30)
	isHard = humanProb <= 0.30 && nLegal >= 16
	return isHard, humanProb, foundTop, lossCp, gapTop2Cp, plausibleCnt, bestSAN
}

func finalizeHardMovePerformance(perf *GameHardMovePerformance) {
	n := perf.HardMoveCount
	if n == 0 {
		return
	}
	k := perf.HardMovesFound
	perf.HardMoveHitRatePct = round1(100.0 * float64(k) / float64(n))
	perf.ExpectedHardHits = round2(perf.ExpectedHardHits)

	// Compute exact Poisson-Binomial tail probability P(X >= k) using the per-move probabilities p_i
	probs := make([]float64, 0, n)
	for _, m := range perf.HardMoments {
		probs = append(probs, m.HumanProb)
	}
	// If there were more than len(HardMoments) hard moves, fill remaining with mean p
	meanP := perf.ExpectedHardHits / float64(n)
	if meanP <= 0.05 {
		meanP = 0.20
	}
	for len(probs) < n {
		probs = append(probs, meanP)
	}

	tailP := poissonBinomialTail(probs, k)
	if tailP < 1e-9 {
		tailP = 1e-9
	}
	if tailP > 1.0 {
		tailP = 1.0
	}
	perf.TailProbability = tailP

	var variance float64
	for _, p := range probs {
		variance += p * (1.0 - p)
	}
	sd := math.Sqrt(math.Max(0.25, variance))
	z := (float64(k) - perf.ExpectedHardHits) / sd
	perf.ZScore = round2(z)

	// Unusualness score = -log10(P(X >= k)), smoothly interpolated so ties break by hit ratio
	negLog10P := -math.Log10(tailP)
	perf.UnusualnessScore = round2(negLog10P)
}

// poissonBinomialTail computes the exact probability P(X >= k) for independent Bernoulli trials with probabilities probs.
func poissonBinomialTail(probs []float64, k int) float64 {
	if k <= 0 {
		return 1.0
	}
	n := len(probs)
	if k > n {
		return 0.0
	}
	dp := make([]float64, n+1)
	dp[0] = 1.0
	for i, p := range probs {
		if p < 0 {
			p = 0
		}
		if p > 1 {
			p = 1
		}
		for j := i + 1; j >= 1; j-- {
			dp[j] = dp[j]*(1.0-p) + dp[j-1]*p
		}
		dp[0] *= (1.0 - p)
	}
	var tail float64
	for j := k; j <= n; j++ {
		tail += dp[j]
	}
	return tail
}

func poissonTailProb(observed int, expected float64) float64 {
	if observed <= 0 || expected <= 0 {
		return 1.0
	}
	if float64(observed) <= expected {
		return 0.5
	}
	// Normal approximation with continuity correction for Poisson/Binomial tail
	z := (float64(observed) - 0.5 - expected) / math.Sqrt(expected)
	return 0.5 * math.Erfc(z/math.Sqrt2)
}

func empiricalQuantile(sorted []float64, q float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	if q <= 0 {
		return sorted[0]
	}
	if q >= 1 {
		return sorted[len(sorted)-1]
	}
	pos := q * float64(len(sorted)-1)
	lo := int(math.Floor(pos))
	hi := int(math.Ceil(pos))
	if lo == hi || hi >= len(sorted) {
		return sorted[lo]
	}
	frac := pos - float64(lo)
	return sorted[lo]*(1.0-frac) + sorted[hi]*frac
}

// simulateSelectiveCheatingOnSelfGames emulates the "Selective Cheating Simulation" from the video:
// taking your own real games and swapping Stockfish's top choice into 1, 2, 50%, or 100% of your missed hard moves.
func simulateSelectiveCheatingOnSelfGames(
	selfPerfs []GameHardMovePerformance,
	p95Threshold float64,
	p99Threshold float64,
) []SelectiveCheatingPoint {
	configs := []struct {
		label string
		spec  string
		mode  string
		count int
	}{
		{"0 Swaps (Your Real Benchmark)", "0 engine moves", "fixed", 0},
		{"1 Hard Move Swapped / Game", "+1 engine move in critical spot", "fixed", 1},
		{"2 Hard Moves Swapped / Game", "+2 engine moves in critical spots", "fixed", 2},
		{"50% of Missed Hard Moves Swapped", "Half of hard moves engine-assisted", "half", 0},
		{"100% of Hard Moves Swapped", "Full engine assistance on hard moves", "all", 0},
	}

	points := make([]SelectiveCheatingPoint, 0, len(configs))
	if len(selfPerfs) == 0 {
		return points
	}

	for _, cfg := range configs {
		caughtTop2 := 0
		caughtTop5 := 0
		var totalSwapped float64
		var scoreSum float64

		for _, perf := range selfPerfs {
			n := perf.HardMoveCount
			missed := n - perf.HardMovesFound
			if missed < 0 {
				missed = 0
			}

			addHits := 0
			switch cfg.mode {
			case "fixed":
				addHits = cfg.count
				if addHits > missed {
					addHits = missed
				}
			case "half":
				addHits = int(math.Ceil(float64(missed) * 0.5))
			case "all":
				addHits = missed
			}

			totalSwapped += float64(addHits)
			simK := perf.HardMovesFound + addHits

			probs := make([]float64, 0, n)
			for _, m := range perf.HardMoments {
				probs = append(probs, m.HumanProb)
			}
			meanP := perf.ExpectedHardHits / math.Max(1.0, float64(n))
			if meanP <= 0.05 {
				meanP = 0.20
			}
			for len(probs) < n {
				probs = append(probs, meanP)
			}

			tailP := poissonBinomialTail(probs, simK)
			if tailP < 1e-9 {
				tailP = 1e-9
			}
			simScore := -math.Log10(tailP)
			scoreSum += simScore

			if simScore >= p99Threshold {
				caughtTop2++
			}
			if simScore >= p95Threshold {
				caughtTop5++
			}
		}

		denom := float64(len(selfPerfs))
		points = append(points, SelectiveCheatingPoint{
			Label:                cfg.label,
			SwappedMovesSpec:     cfg.spec,
			AvgSwappedPerGame:    round2(totalSwapped / denom),
			TopTwoBandsCaughtPct: round1(100.0 * float64(caughtTop2) / denom),
			TopFivePctCaughtPct:  round1(100.0 * float64(caughtTop5) / denom),
			AvgUnusualnessScore:  round2(scoreSum / denom),
		})
	}

	return points
}

func checkOpponentAccountsBanStatus(
	oppPerfs []GameHardMovePerformance,
	chessComBaseURL string,
	lichessBaseURL string,
) (int, map[string]string) {
	if chessComBaseURL == "" {
		chessComBaseURL = "https://api.chess.com"
	}
	if lichessBaseURL == "" {
		lichessBaseURL = "https://lichess.org"
	}

	// Prioritize checking opponents who scored highest on unusualness first (up to 40 unique accounts so it's fast)
	sorted := make([]GameHardMovePerformance, len(oppPerfs))
	copy(sorted, oppPerfs)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].UnusualnessScore > sorted[j].UnusualnessScore
	})

	var uniqueHandles []string
	seen := make(map[string]bool)
	for _, p := range sorted {
		h := strings.TrimSpace(p.PlayerName)
		low := strings.ToLower(h)
		if h == "" || h == "?" || strings.Contains(h, " ") || seen[low] {
			continue
		}
		seen[low] = true
		uniqueHandles = append(uniqueHandles, h)
		if len(uniqueHandles) >= 35 {
			break
		}
	}

	banned := make(map[string]string)
	if len(uniqueHandles) == 0 {
		return 0, banned
	}

	client := &http.Client{Timeout: 4 * time.Second}
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, 6)

	for _, handle := range uniqueHandles {
		wg.Add(1)
		go func(username string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			low := strings.ToLower(username)
			// Check Chess.com public player status
			ccEndpoint := fmt.Sprintf("%s/pub/player/%s", strings.TrimRight(chessComBaseURL, "/"), url.PathEscape(low))
			if req, err := http.NewRequest(http.MethodGet, ccEndpoint, nil); err == nil {
				req.Header.Set("User-Agent", "NewScid/1.0")
				if resp, err := client.Do(req); err == nil {
					var ccData struct {
						Status string `json:"status"`
					}
					_ = json.NewDecoder(resp.Body).Decode(&ccData)
					resp.Body.Close()
					stLower := strings.ToLower(ccData.Status)
					if strings.Contains(stLower, "fair_play") || stLower == "closed:fair_play_violations" {
						mu.Lock()
						banned[low] = "Chess.com Closed: Fair Play"
						mu.Unlock()
						return
					}
				}
			}

			// Check Lichess public user status
			liEndpoint := fmt.Sprintf("%s/api/user/%s", strings.TrimRight(lichessBaseURL, "/"), url.PathEscape(username))
			if req, err := http.NewRequest(http.MethodGet, liEndpoint, nil); err == nil {
				req.Header.Set("User-Agent", "NewScid/1.0")
				if resp, err := client.Do(req); err == nil {
					var liData struct {
						TosViolation bool `json:"tosViolation"`
						Disabled     bool `json:"disabled"`
					}
					_ = json.NewDecoder(resp.Body).Decode(&liData)
					resp.Body.Close()
					if liData.TosViolation {
						mu.Lock()
						banned[low] = "Lichess TOS / Fair Play Violation"
						mu.Unlock()
						return
					}
				}
			}
		}(handle)
	}
	wg.Wait()

	return len(uniqueHandles), banned
}

func buildHardMoveVerdict(r *HardMoveBenchmarkReport) {
	if r.OpponentGamesQualified == 0 {
		r.VerdictHeadline = "No qualified opponent games with sufficient Hard Moves."
		return
	}

	if r.ExcessOpponentGames >= 1.5 && r.OddsOneIn >= 20 {
		r.VerdictHeadline = fmt.Sprintf(
			"Your opponents' games are more unusual than your benchmark by ~%.0f games beyond chance (%d–%d est., %.2f%% of games; odds of luck ~1 in %d).",
			r.ExcessOpponentGames,
			r.ExcessGamesLowEstimate,
			r.ExcessGamesHighEstimate,
			r.ExcessPctOfAllGames,
			r.OddsOneIn,
		)
	} else {
		r.VerdictHeadline = fmt.Sprintf(
			"Across %d games, opponent top-band unusualness (%.2f%%) tracks closely with your own human benchmark (%.2f%%) — well under 1%% excess.",
			r.OpponentGamesQualified,
			r.OpponentTopTwoRatePct,
			r.SelfTopTwoRatePct,
		)
	}

	r.VerdictFindings = []string{
		fmt.Sprintf(
			"Hard-Move Filter (p <= 30%%): Isolated %d non-obvious positions across %d games. Your hit rate on hard moves is %.1f%% (expected %.1f%%) vs. Opponents' %.1f%% (expected %.1f%%).",
			r.TotalHardMovesAnalyzed,
			r.TotalGamesScanned,
			r.SelfAvgHardHitRatePct,
			r.SelfExpectedHitRatePct,
			r.OpponentAvgHardHitRatePct,
			r.OpponentExpectedHitRatePct,
		),
		fmt.Sprintf(
			"Top Two Bands (Your Most Unusual 1%%, Score >= %.2f): %d of your opponent games (%.2f%%) landed in the Top Two Bands vs. %.1f expected by chance — an estimated %d to %d excess games.",
			r.TopTwoBandThreshold,
			r.OpponentInTopTwoBands,
			r.OpponentTopTwoRatePct,
			r.ExpectedOpponentTopTwo,
			r.ExcessGamesLowEstimate,
			r.ExcessGamesHighEstimate,
		),
		fmt.Sprintf(
			"Why You Also Appear in the Top 100: %d of the Top %d most unusual performances are your own games (%d are opponents') — showing why a single unusual game is never proof on its own.",
			r.SelfInTop100Count,
			len(r.Top100Games),
			r.OpponentInTop100Count,
		),
	}

	if r.BannedValidation.GamesVsBanned > 0 {
		r.VerdictFindings = append(r.VerdictFindings, fmt.Sprintf(
			"Banned Opponent Validation: Across %d games against %d confirmed closed/banned opponents, %.1f%% landed in the Top Two Bands (%.1fx the rate of open accounts).",
			r.BannedValidation.GamesVsBanned,
			r.BannedValidation.ConfirmedBannedCount,
			r.BannedValidation.BannedTopTwoRatePct,
			r.BannedValidation.MultiplierVsOpen,
		))
	}
}
