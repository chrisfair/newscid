package fairplay

import (
	"fmt"
	"math"
	"strings"
)

// FormatAccuracyBand defines expected accuracy bounds and flagging threshold for a platform/format/rating band.
type FormatAccuracyBand struct {
	AvgLo     float64 `json:"avgLo"`
	AvgHi     float64 `json:"avgHi"`
	Expected  float64 `json:"expected"`
	Threshold float64 `json:"threshold"`
}

type accuracyBandRow struct {
	maxRating int
	band      FormatAccuracyBand
}

// ChessComFormatAccuracy matches FairPlay Radar's calibrated Chess.com accuracy norms.
var ChessComFormatAccuracy = map[string][]accuracyBandRow{
	"bullet": {
		{700, FormatAccuracyBand{65, 75, 70, 84}},
		{1000, FormatAccuracyBand{73, 80, 76, 85}},
		{1500, FormatAccuracyBand{77, 82, 79, 87}},
		{2000, FormatAccuracyBand{80, 85, 82, 89}},
		{2500, FormatAccuracyBand{83, 87, 85, 91}},
		{3000, FormatAccuracyBand{86, 90, 88, 93}},
		{99999, FormatAccuracyBand{87, 91, 89, 94}},
	},
	"blitz": {
		{700, FormatAccuracyBand{65, 74, 70, 85}},
		{1000, FormatAccuracyBand{73, 80, 76, 86}},
		{1500, FormatAccuracyBand{78, 83, 80, 88}},
		{2000, FormatAccuracyBand{81, 86, 84, 90}},
		{2500, FormatAccuracyBand{84, 88, 86, 91}},
		{3000, FormatAccuracyBand{86, 90, 88, 93}},
		{99999, FormatAccuracyBand{87, 91, 89, 94}},
	},
	"rapid": {
		{700, FormatAccuracyBand{66, 75, 71, 85}},
		{1000, FormatAccuracyBand{72, 79, 76, 86}},
		{1500, FormatAccuracyBand{78, 83, 80, 89}},
		{2000, FormatAccuracyBand{82, 87, 85, 91}},
		{2500, FormatAccuracyBand{86, 90, 88, 93}},
		{3000, FormatAccuracyBand{88, 92, 90, 95}},
		{99999, FormatAccuracyBand{90, 93, 92, 97}},
	},
}

// LichessFormatAccuracy matches FairPlay Radar's calibrated Lichess/Lila accuracy norms.
var LichessFormatAccuracy = map[string][]accuracyBandRow{
	"bullet": {
		{1000, FormatAccuracyBand{55, 68, 64, 80}},
		{1300, FormatAccuracyBand{65, 75, 70, 82}},
		{1800, FormatAccuracyBand{72, 80, 76, 84}},
		{2300, FormatAccuracyBand{76, 83, 80, 86}},
		{2800, FormatAccuracyBand{79, 85, 82, 88}},
		{99999, FormatAccuracyBand{84, 90, 87, 93}},
	},
	"blitz": {
		{1000, FormatAccuracyBand{58, 70, 66, 82}},
		{1300, FormatAccuracyBand{68, 77, 73, 84}},
		{1800, FormatAccuracyBand{74, 82, 78, 85}},
		{2300, FormatAccuracyBand{79, 86, 83, 89}},
		{2800, FormatAccuracyBand{84, 91, 88, 94}},
		{99999, FormatAccuracyBand{86, 92, 89, 95}},
	},
	"rapid": {
		{1000, FormatAccuracyBand{60, 72, 68, 83}},
		{1300, FormatAccuracyBand{70, 78, 74, 85}},
		{1800, FormatAccuracyBand{76, 83, 80, 86}},
		{2300, FormatAccuracyBand{80, 87, 84, 90}},
		{2800, FormatAccuracyBand{85, 91, 88, 94}},
		{99999, FormatAccuracyBand{87, 93, 90, 96}},
	},
}

// GetFormatAccuracyBand returns the calibrated accuracy band for a platform, format, and rating.
func GetFormatAccuracyBand(format string, rating int, platform string) FormatAccuracyBand {
	p := strings.ToLower(strings.TrimSpace(platform))
	isLila := p == "lichess" || p == "playstrategy" || p == "pychess"
	tables := ChessComFormatAccuracy
	if isLila {
		tables = LichessFormatAccuracy
	}
	f := strings.ToLower(strings.TrimSpace(format))
	if f == "classical" {
		f = "rapid"
	}
	rows, ok := tables[f]
	if !ok {
		rows = tables["blitz"]
	}
	for _, r := range rows {
		if rating < r.maxRating {
			return r.band
		}
	}
	return rows[len(rows)-1].band
}

// GetCRSAccountAgeFactor returns the FairPlay Radar account-age risk multiplier.
func GetCRSAccountAgeFactor(ageDays int) float64 {
	switch {
	case ageDays <= 0:
		return 1.0 // unknown age (e.g. local DB) -> neutral
	case ageDays <= 7:
		return 5.0
	case ageDays <= 14:
		return 4.0
	case ageDays <= 30:
		return 3.5
	case ageDays <= 60:
		return 2.5
	case ageDays <= 90:
		return 2.0
	case ageDays <= 180:
		return 1.5
	case ageDays <= 365:
		return 1.2
	default:
		return 1.0
	}
}

// GetEloPoolWinRateOffset accounts for natural above-50% win rates among upper-tail (>1750)
// and verified titled players who pair downward on average.
func GetEloPoolWinRateOffset(rating, accountAgeDays, overallGames int, title string) float64 {
	titleFloorOffset := 0.0
	t := strings.ToUpper(strings.TrimSpace(title))
	switch t {
	case "GM", "WGM":
		titleFloorOffset = 0.25
	case "IM", "WIM", "LM":
		titleFloorOffset = 0.21
	case "FM", "WFM", "NM", "WNM", "CM", "WCM":
		titleFloorOffset = 0.18
	}
	if rating <= 1750 || (accountAgeDays > 0 && accountAgeDays < 180 && overallGames < 150 && titleFloorOffset == 0) {
		return titleFloorOffset
	}
	poolOffset := math.Min(0.28, math.Max(0, float64(rating-1750)*0.00038))
	return math.Max(titleFloorOffset, poolOffset)
}

// ACPLToAccuracyPct converts Average Centipawn Loss to an approximate Lichess/Chess.com 0-100 accuracy %.
func ACPLToAccuracyPct(acpl float64) float64 {
	if acpl <= 0 {
		return 99.5
	}
	acc := 103.1668*math.Exp(-0.006*acpl) - 3.1669
	if acc > 99.5 {
		return 99.5
	}
	if acc < 10.0 {
		return 10.0
	}
	return round1(acc)
}

// RadarProfileHints holds optional online profile metadata (from Lichess/Chess.com APIs)
// to enrich local database fair-play analysis to full FairPlay Radar parity.
type RadarProfileHints struct {
	Platform        string `json:"platform"` // "lichess" | "chesscom" | "otb"
	PlayerTitle     string `json:"playerTitle,omitempty"`
	AccountAgeDays  int    `json:"accountAgeDays,omitempty"`
	OverallGames    int    `json:"overallGames,omitempty"`
	PuzzleRating    int    `json:"puzzleRating,omitempty"`
	PuzzleCount     int    `json:"puzzleCount,omitempty"`
	OfficialBanned  bool   `json:"officialBanned,omitempty"`
	OfficialBanNote string `json:"officialBanNote,omitempty"`
	SevenDayEloGain int    `json:"sevenDayEloGain,omitempty"`
}

// RadarSignal represents one of the 6 core FairPlay Radar forensic sub-scores.
type RadarSignal struct {
	ID       string  `json:"id"` // official_flag | engine_accuracy | puzzle_footprint | crs_composite | cross_control_gap | cadence_flatness
	Name     string  `json:"name"`
	Score    float64 `json:"score"`
	MaxScore float64 `json:"maxScore"`
	Flagged  bool    `json:"flagged"`
	Detail   string  `json:"detail"`
}

// RadarRiskAssessment provides full FairPlay Radar 6-signal parity (CRS 0-100, Cheat Likelihood %, Risk Tier).
type RadarRiskAssessment struct {
	Platform              string        `json:"platform"`
	CRSScore              float64       `json:"crsScore"`           // 0-100 Composite Risk Score
	CheatLikelihoodPct    int           `json:"cheatLikelihoodPct"` // 0-99
	RiskTier              string        `json:"riskTier"`           // "low" | "moderate" | "elevated" | "high" | "critical"
	HighAccuracyGameCount int           `json:"highAccuracyGameCount"`
	MaxHighAccuracyStreak int           `json:"maxHighAccuracyStreak"`
	AvgEstimatedAccuracy  float64       `json:"avgEstimatedAccuracy"`
	ExpectedBandThreshold float64       `json:"expectedBandThreshold"`
	TitleDampener         float64       `json:"titleDampener"`
	AccountAgeMultiplier  float64       `json:"accountAgeMultiplier"`
	Signals               []RadarSignal `json:"signals"`
	Recommendation        string        `json:"recommendation"`
}

func inferPlatformFromDossier(d *PlayerFairPlayDossier) string {
	lichessHits, chesscomHits := 0, 0
	for _, g := range d.AllGames {
		s := strings.ToLower(g.Site + " " + g.Event)
		if strings.Contains(s, "lichess") {
			lichessHits++
		} else if strings.Contains(s, "chess.com") || strings.Contains(s, "chesscom") {
			chesscomHits++
		}
	}
	if lichessHits > chesscomHits {
		return "lichess"
	}
	if chesscomHits > 0 {
		return "chesscom"
	}
	return "lichess"
}

// BuildRadarAssessment computes the 6-signal FairPlay Radar assessment from a NewScid PlayerFairPlayDossier
// plus optional online profile hints (account age, puzzle footprint, official TOS ban status).
func BuildRadarAssessment(d *PlayerFairPlayDossier, hints RadarProfileHints) *RadarRiskAssessment {
	if d == nil {
		return nil
	}
	platform := strings.ToLower(strings.TrimSpace(hints.Platform))
	if platform == "" {
		platform = inferPlatformFromDossier(d)
	}

	title := strings.TrimSpace(hints.PlayerTitle)
	if title == "" {
		for _, g := range d.AllGames {
			if strings.TrimSpace(g.PlayerTitle) != "" {
				title = g.PlayerTitle
				break
			}
		}
	}
	titleDamp := verifiedMasterTitleDampener(title)
	ageMult := GetCRSAccountAgeFactor(hints.AccountAgeDays)

	// Determine primary time control and average Elo
	primaryTC := strings.ToLower(d.HighestRiskTimeControl)
	if primaryTC == "" {
		primaryTC = "blitz"
	}
	avgElo := 1500
	winRate := 0.50
	if len(d.TimeControlProfiles) > 0 {
		var best TimeControlIssueProfile
		for _, tc := range d.TimeControlProfiles {
			if tc.GamesAnalyzed > best.GamesAnalyzed {
				best = tc
			}
		}
		if best.AvgOfficialElo > 0 {
			avgElo = best.AvgOfficialElo
		}
		if best.GamesAnalyzed > 0 {
			winRate = best.WinRatePct / 100.0
		}
	}
	if floor := verifiedMasterRatingFloor(title); floor > avgElo {
		avgElo = floor
	}

	band := GetFormatAccuracyBand(primaryTC, avgElo, platform)

	// Signal 1: Official TOS / Fair-Play Flag (0-100)
	sigOfficial := RadarSignal{
		ID:       "official_flag",
		Name:     "Official Fair-Play / TOS Status",
		MaxScore: 100,
		Detail:   "No official platform TOS/fair-play closure recorded.",
	}
	if hints.OfficialBanned {
		sigOfficial.Score = 100
		sigOfficial.Flagged = true
		if hints.OfficialBanNote != "" {
			sigOfficial.Detail = hints.OfficialBanNote
		} else {
			sigOfficial.Detail = "Confirmed official platform Fair Play / TOS closure."
		}
	}

	// Signal 2: Platform-Calibrated Engine Accuracy & High-Accuracy Streaks (0-30)
	highAccGames := 0
	curStreak := 0
	maxStreak := 0
	accSum := 0.0
	evalGames := 0
	for _, g := range d.AllGames {
		if !g.HasEvalData && g.ACPL <= 0 {
			continue
		}
		evalGames++
		acc := ACPLToAccuracyPct(g.ACPL)
		accSum += acc
		gBand := GetFormatAccuracyBand(g.TimeControlCategory, maxIntVal(g.PlayerElo, avgElo), platform)
		if acc >= gBand.Threshold && g.NonBookMoves >= 12 {
			highAccGames++
			curStreak++
			if curStreak > maxStreak {
				maxStreak = curStreak
			}
		} else {
			curStreak = 0
		}
	}
	avgAcc := 0.0
	if evalGames > 0 {
		avgAcc = round1(accSum / float64(evalGames))
	}
	accScore := 0.0
	if evalGames > 0 {
		if avgAcc > band.Expected {
			accScore += math.Min(16, (avgAcc-band.Expected)*1.8)
		}
		if highAccGames >= 2 {
			accScore += math.Min(8, float64(highAccGames)*1.8)
		}
		if maxStreak >= 2 {
			accScore += math.Min(6, float64(maxStreak)*2.0)
		}
		accScore = math.Min(30, accScore*titleDamp)
	}
	sigAccuracy := RadarSignal{
		ID:       "engine_accuracy",
		Name:     "Platform-Calibrated Accuracy & Streaks",
		Score:    round1(accScore),
		MaxScore: 30,
		Flagged:  accScore >= 14 || maxStreak >= 3,
		Detail: fmt.Sprintf(
			"Avg est. accuracy %.1f%% vs %.0f%% band expected (threshold %.0f%%); %d high-accuracy games (max streak %d).",
			avgAcc, band.Expected, band.Threshold, highAccGames, maxStreak,
		),
	}

	// Signal 3: Puzzle Training Footprint vs Live Rating (0-15)
	puzScore := 0.0
	puzDetail := "Puzzle profile not supplied (local database mode)."
	if hints.PuzzleRating > 0 || hints.PuzzleCount > 0 {
		gap := avgElo - hints.PuzzleRating
		if hints.PuzzleCount < 15 && avgElo >= 1700 {
			puzScore += 7
		}
		if hints.PuzzleRating > 0 && gap >= 250 {
			puzScore += math.Min(8, float64(gap-200)*0.025)
		}
		puzScore = math.Min(15, puzScore*titleDamp)
		puzDetail = fmt.Sprintf("Puzzle rating %d (%d puzzles) vs %d live rating (gap %+d).", hints.PuzzleRating, hints.PuzzleCount, avgElo, gap)
	}
	sigPuzzle := RadarSignal{
		ID:       "puzzle_footprint",
		Name:     "Puzzle Training Footprint",
		Score:    round1(puzScore),
		MaxScore: 15,
		Flagged:  puzScore >= 8,
		Detail:   puzDetail,
	}

	// Signal 4: CRS Composite (Account Age Multiplier + Elo-Pool Win-Rate Offset + Sample Weighting) (0-25)
	nGames := d.TotalGamesAnalyzed
	if hints.OverallGames > nGames {
		nGames = hints.OverallGames
	}
	sampleW := float64(d.TotalGamesAnalyzed) / (float64(d.TotalGamesAnalyzed) + 20.0)
	expectedWR := 0.50 + GetEloPoolWinRateOffset(avgElo, hints.AccountAgeDays, nGames, title)
	excessWR := math.Max(0, winRate-expectedWR)
	crsSub := (excessWR * 65.0 * sampleW)
	if hints.SevenDayEloGain >= 120 {
		crsSub += math.Min(8, float64(hints.SevenDayEloGain-100)*0.04)
	}
	if ageMult > 1.0 && (excessWR > 0.08 || accScore >= 10) {
		crsSub *= math.Min(1.8, 1.0+(ageMult-1.0)*0.22)
	}
	crsSub = math.Min(25, crsSub*titleDamp)
	sigCRS := RadarSignal{
		ID:       "crs_composite",
		Name:     "Win-Rate / Age / Elo-Pool CRS",
		Score:    round1(crsSub),
		MaxScore: 25,
		Flagged:  crsSub >= 12,
		Detail: fmt.Sprintf(
			"Win rate %.1f%% vs %.1f%% Elo-pool expected (age mult %.1fx, title dampener %.2fx).",
			winRate*100, expectedWR*100, ageMult, titleDamp,
		),
	}

	// Signal 5: Cross-Control Divergence (Slow vs Bullet/Blitz) (0-15)
	divScore := 0.0
	divDetail := "Single time-control sample; no cross-control divergence measured."
	if d.CrossDivergence.HasMultipleControls {
		divScore = math.Min(15, d.CrossDivergence.DivergenceScore*0.18*titleDamp)
		divDetail = d.CrossDivergence.Explanation
	}
	sigCross := RadarSignal{
		ID:       "cross_control_gap",
		Name:     "Cross-Control Divergence",
		Score:    round1(divScore),
		MaxScore: 15,
		Flagged:  divScore >= 8,
		Detail:   divDetail,
	}

	// Signal 6: Cadence Flatness, Micro-Bursts & Forced-Move Hesitation (0-15)
	cadScore := 0.0
	roboticGames := 0
	microBurstGames := 0
	hesitationSum := 0
	clockGames := 0
	for _, g := range d.AllGames {
		if !g.HasClockData {
			continue
		}
		clockGames++
		if g.MoveTimeCV > 0 && g.MoveTimeCV < 0.38 {
			roboticGames++
		}
		if g.MicroBurstWindowCV > 0 && g.MicroBurstWindowCV < 0.24 {
			microBurstGames++
		}
		hesitationSum += g.ForcedMoveHesitationCount
	}
	cadDetail := "No clock telemetry ([%clk]/[%emt]) present in sample."
	if clockGames > 0 {
		robPct := float64(roboticGames) / float64(clockGames)
		mbPct := float64(microBurstGames) / float64(clockGames)
		cadScore = math.Min(15, robPct*11.0+mbPct*6.0+math.Min(4, float64(hesitationSum)*0.6))
		cadDetail = fmt.Sprintf(
			"%d/%d clocked games showed flat robotic CV (<0.38); %d micro-burst windows; %d forced-move hesitations.",
			roboticGames, clockGames, microBurstGames, hesitationSum,
		)
	}
	sigCadence := RadarSignal{
		ID:       "cadence_flatness",
		Name:     "Move-Time Cadence & Micro-Bursts",
		Score:    round1(cadScore),
		MaxScore: 15,
		Flagged:  cadScore >= 7.5,
		Detail:   cadDetail,
	}

	signals := []RadarSignal{sigOfficial, sigAccuracy, sigPuzzle, sigCRS, sigCross, sigCadence}

	// Combine CRS (0-100) and Cheat Likelihood % (0-99)
	crsTotal := 0.0
	if hints.OfficialBanned {
		crsTotal = 99.0
	} else {
		rawSum := sigAccuracy.Score + sigPuzzle.Score + sigCRS.Score + sigCross.Score + sigCadence.Score
		// Blend with NewScid's longitudinal OverallIssueScore so both engines reinforce cleanly.
		crsTotal = math.Min(99.0, round1(math.Max(rawSum, rawSum*0.65+d.OverallIssueScore*0.35)))
	}
	likelihood := int(math.Round(crsTotal))
	if likelihood > 99 {
		likelihood = 99
	}
	if likelihood < 0 {
		likelihood = 0
	}

	tier := "low"
	rec := "Profile metrics fall within normal human variance for this rating band."
	switch {
	case hints.OfficialBanned || likelihood >= 82:
		tier = "critical"
		rec = "Critical anomaly profile: multiple independent signals exceed human thresholds. Review games with Stockfish and avoid rated rematches."
	case likelihood >= 65:
		tier = "high"
		rec = "High fair-play risk: elevated accuracy streaks, pacing flatness, or cross-control divergence warrant deep engine inspection."
	case likelihood >= 45:
		tier = "elevated"
		rec = "Elevated anomaly signals detected: inspect flagged games and move-time cadence before drawing conclusions."
	case likelihood >= 25:
		tier = "moderate"
		rec = "Mild statistical variance observed; collect a larger sample before taking action."
	}

	return &RadarRiskAssessment{
		Platform:              platform,
		CRSScore:              round1(crsTotal),
		CheatLikelihoodPct:    likelihood,
		RiskTier:              tier,
		HighAccuracyGameCount: highAccGames,
		MaxHighAccuracyStreak: maxStreak,
		AvgEstimatedAccuracy:  avgAcc,
		ExpectedBandThreshold: band.Threshold,
		TitleDampener:         titleDamp,
		AccountAgeMultiplier:  ageMult,
		Signals:               signals,
		Recommendation:        rec,
	}
}

func maxIntVal(a, b int) int {
	if a > b {
		return a
	}
	return b
}
