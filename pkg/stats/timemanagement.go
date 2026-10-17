package stats

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
	clkRegex  = regexp.MustCompile(`(?i)\[%clk\s+([^\]]+)\]`)
	emtRegex  = regexp.MustCompile(`(?i)\[%emt\s+([^\]]+)\]`)
	evalRegex = regexp.MustCompile(`(?i)\[%eval\s+([^\]]+)\]`)
)

// MoveTimeRecord captures clock and evaluation data for a single ply.
type MoveTimeRecord struct {
	Ply                        int     `json:"ply"`
	MoveNumber                 int     `json:"moveNumber"`
	Turn                       string  `json:"turn"` // "white" | "black"
	SAN                        string  `json:"san"`
	UCI                        string  `json:"uci"`
	ClockRemainingSec          float64 `json:"clockRemainingSec"`
	TimeSpentSec               float64 `json:"timeSpentSec"`
	ClockStr                   string  `json:"clockStr"`
	Phase                      string  `json:"phase"` // "opening" | "middlegame" | "endgame"
	IsTank                     bool    `json:"isTank"`
	IsPanic                    bool    `json:"isPanic"`
	IsScramble                 bool    `json:"isScramble"`
	EvalCp                     int     `json:"evalCp"`
	EvalDropCp                 int     `json:"evalDropCp"`
	EvaluationClassification   string  `json:"evaluationClassification"` // "best" | "good" | "inaccuracy" | "mistake" | "blunder" | "unknown"
	Flag                       string  `json:"flag"`                     // "tank_success" | "tank_blunder" | "panic_blunder" | "scramble_blunder" | "normal"
}

// SideTimeStats summarizes time management metrics for one color.
type SideTimeStats struct {
	TotalMoves             int     `json:"totalMoves"`
	TotalTimeSpentSec      float64 `json:"totalTimeSpentSec"`
	AvgMoveTimeSec         float64 `json:"avgMoveTimeSec"`
	MedianMoveTimeSec      float64 `json:"medianMoveTimeSec"`
	OpeningAvgTimeSec      float64 `json:"openingAvgTimeSec"`
	MiddlegameAvgTimeSec   float64 `json:"middlegameAvgTimeSec"`
	EndgameAvgTimeSec      float64 `json:"endgameAvgTimeSec"`
	LongestTankSec         float64 `json:"longestTankSec"`
	LongestTankPly         int     `json:"longestTankPly"`
	TankCount              int     `json:"tankCount"`
	TankBlunderCount       int     `json:"tankBlunderCount"`
	PanicMoveCount         int     `json:"panicMoveCount"`
	TimeTroubleMoves       int     `json:"timeTroubleMoves"`
	TimeTroubleBlunders    int     `json:"timeTroubleBlunders"`
	CorrelationTimeVsLoss  float64 `json:"correlationTimeVsLoss"`
}

// TankIncident details a deep think episode and its outcome.
type TankIncident struct {
	Ply             int     `json:"ply"`
	MoveNumber      int     `json:"moveNumber"`
	Player          string  `json:"player"`
	Color           string  `json:"color"`
	SAN             string  `json:"san"`
	TimeSpentSec    float64 `json:"timeSpentSec"`
	ClockBeforeSec  float64 `json:"clockBeforeSec"`
	ClockAfterSec   float64 `json:"clockAfterSec"`
	EvalDropCp      int     `json:"evalDropCp"`
	Outcome         string  `json:"outcome"` // "Masterstroke" | "Solid" | "Inaccuracy" | "Blunder"
	Description     string  `json:"description"`
}

// PanicIncident tracks impulsive moves in critical positions.
type PanicIncident struct {
	Ply               int     `json:"ply"`
	MoveNumber        int     `json:"moveNumber"`
	Player            string  `json:"player"`
	Color             string  `json:"color"`
	SAN               string  `json:"san"`
	TimeSpentSec      float64 `json:"timeSpentSec"`
	ClockRemainingSec float64 `json:"clockRemainingSec"`
	EvalDropCp        int     `json:"evalDropCp"`
	Description       string  `json:"description"`
}

// ScrambleSummary provides comparative time scramble data.
type ScrambleSummary struct {
	WhiteEnteredScramblePly int     `json:"whiteEnteredScramblePly"`
	BlackEnteredScramblePly int     `json:"blackEnteredScramblePly"`
	WhiteScrambleMoves      int     `json:"whiteScrambleMoves"`
	BlackScrambleMoves      int     `json:"blackScrambleMoves"`
	WhiteScrambleBlunders   int     `json:"whiteScrambleBlunders"`
	BlackScrambleBlunders   int     `json:"blackScrambleBlunders"`
}

// GameTimeManagementReport contains full dossier for a single game.
type GameTimeManagementReport struct {
	GameID            int               `json:"gameId"`
	WhitePlayer       string            `json:"whitePlayer"`
	BlackPlayer       string            `json:"blackPlayer"`
	Event             string            `json:"event"`
	Date              string            `json:"date"`
	TimeControl       string            `json:"timeControl"`
	WhiteHasClock     bool              `json:"whiteHasClock"`
	BlackHasClock     bool              `json:"blackHasClock"`
	Moves             []MoveTimeRecord  `json:"moves"`
	WhiteStats        SideTimeStats     `json:"whiteStats"`
	BlackStats        SideTimeStats     `json:"blackStats"`
	TankDossier       []TankIncident    `json:"tankDossier"`
	PanicIncidents    []PanicIncident   `json:"panicIncidents"`
	ScrambleSummary   ScrambleSummary   `json:"scrambleSummary"`
	Recommendations   []string          `json:"recommendations"`
}

// PlayerTimeManagementDossier provides aggregate historical time management analysis for a player.
type PlayerTimeManagementDossier struct {
	PlayerName            string   `json:"playerName"`
	GamesAnalyzed         int      `json:"gamesAnalyzed"`
	GamesWithClock        int      `json:"gamesWithClock"`
	AverageMoveTimeSec    float64  `json:"averageMoveTimeSec"`
	OpeningAvgSec         float64  `json:"openingAvgSec"`
	MiddlegameAvgSec      float64  `json:"middlegameAvgSec"`
	EndgameAvgSec         float64  `json:"endgameAvgSec"`
	TotalTanks            int      `json:"totalTanks"`
	TankSuccessRate       float64  `json:"tankSuccessRate"` // percentage of tanks that did not blunder
	PanicMoveCount        int      `json:"panicMoveCount"`
	PanicErrorRate        float64  `json:"panicErrorRate"`
	ScrambleFrequencyPct  float64  `json:"scrambleFrequencyPct"`
	ScrambleBlunderCount  int      `json:"scrambleBlunderCount"`
	HabitFlags            []string `json:"habitFlags"`
	CoachingDossier       []string `json:"coachingDossier"`
}

// ParseClockToSeconds parses strings like "1:30:00", "0:05:22", "45", "12.4" into seconds.
func ParseClockToSeconds(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty clock string")
	}

	parts := strings.Split(s, ":")
	if len(parts) == 3 {
		h, err1 := strconv.ParseFloat(parts[0], 64)
		m, err2 := strconv.ParseFloat(parts[1], 64)
		sec, err3 := strconv.ParseFloat(parts[2], 64)
		if err1 != nil || err2 != nil || err3 != nil {
			return 0, fmt.Errorf("invalid H:M:S format: %s", s)
		}
		return h*3600 + m*60 + sec, nil
	} else if len(parts) == 2 {
		m, err1 := strconv.ParseFloat(parts[0], 64)
		sec, err2 := strconv.ParseFloat(parts[1], 64)
		if err1 != nil || err2 != nil {
			return 0, fmt.Errorf("invalid M:S format: %s", s)
		}
		return m*60 + sec, nil
	} else if len(parts) == 1 {
		secStr := strings.TrimSuffix(parts[0], "s")
		sec, err := strconv.ParseFloat(secStr, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid S format: %s", s)
		}
		return sec, nil
	}
	return 0, fmt.Errorf("unknown clock format: %s", s)
}

// ParseEvalToCentipawns parses strings like "+1.25", "-0.80", "#2", "-#1".
func ParseEvalToCentipawns(s string) (int, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	if strings.HasPrefix(s, "#") || strings.HasPrefix(s, "+#") {
		return 9999, true
	}
	if strings.HasPrefix(s, "-#") {
		return -9999, true
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, false
	}
	return int(math.Round(f * 100)), true
}

// AnalyzeGameTimeManagement produces a thorough time and tanking dossier for a chess game.
func AnalyzeGameTimeManagement(game *chess.Game) (*GameTimeManagementReport, error) {
	if game == nil {
		return nil, fmt.Errorf("game is nil")
	}

	report := &GameTimeManagementReport{
		GameID:          game.ID,
		WhitePlayer:     game.Tag("White"),
		BlackPlayer:     game.Tag("Black"),
		Event:           game.Tag("Event"),
		Date:            game.Tag("Date"),
		TimeControl:     game.Tag("TimeControl"),
		Moves:           make([]MoveTimeRecord, 0),
		TankDossier:     make([]TankIncident, 0),
		PanicIncidents:  make([]PanicIncident, 0),
		Recommendations: make([]string, 0),
	}

	if game.RootMove == nil {
		return report, nil
	}

	// First pass: extract clock, EMT, and eval for all plies
	type rawPly struct {
		ply      int
		moveNum  int
		turn     string
		san      string
		uci      string
		clockSec float64
		hasClock bool
		emtSec   float64
		hasEMT   bool
		clockStr string
		evalCp   int
		hasEval  bool
		nags     []int
	}

	var rawPlies []rawPly
	curr := game.RootMove
	plyIdx := 1
	var prevEvalCp int
	var hadPrevEval bool

	for curr != nil {
		moveNum := (plyIdx + 1) / 2
		turn := "white"
		if plyIdx%2 == 0 {
			turn = "black"
		}

		rp := rawPly{
			ply:      plyIdx,
			moveNum:  moveNum,
			turn:     turn,
			san:      curr.SAN,
			uci:      curr.UCI,
			nags:     curr.NAGs,
			clockStr: curr.Clock,
		}

		// Parse clock
		clkStr := curr.Clock
		if clkStr == "" && curr.Comment != "" {
			if m := clkRegex.FindStringSubmatch(curr.Comment); len(m) > 1 {
				clkStr = strings.TrimSpace(m[1])
				rp.clockStr = clkStr
			}
		}
		if clkStr != "" {
			if sec, err := ParseClockToSeconds(clkStr); err == nil {
				rp.clockSec = sec
				rp.hasClock = true
				if turn == "white" {
					report.WhiteHasClock = true
				} else {
					report.BlackHasClock = true
				}
			}
		}

		// Parse EMT (Elapsed Move Time)
		emtStr := curr.EMT
		if emtStr == "" && curr.Comment != "" {
			if m := emtRegex.FindStringSubmatch(curr.Comment); len(m) > 1 {
				emtStr = strings.TrimSpace(m[1])
			}
		}
		if emtStr != "" {
			if sec, err := ParseClockToSeconds(emtStr); err == nil {
				rp.emtSec = sec
				rp.hasEMT = true
			}
		}

		// Parse Eval
		if curr.Comment != "" {
			if m := evalRegex.FindStringSubmatch(curr.Comment); len(m) > 1 {
				if cp, ok := ParseEvalToCentipawns(m[1]); ok {
					rp.evalCp = cp
					rp.hasEval = true
				}
			}
		}

		rawPlies = append(rawPlies, rp)
		curr = curr.Next
		plyIdx++
	}

	totalPlies := len(rawPlies)
	var prevWhiteClock, prevBlackClock float64
	var hadPrevWhiteClock, hadPrevBlackClock bool

	for i, rp := range rawPlies {
		// Determine phase
		phase := "opening"
		if rp.ply > 24 && rp.ply <= 60 {
			phase = "middlegame"
		} else if rp.ply > 60 {
			phase = "endgame"
		}

		var timeSpent float64
		if rp.hasEMT {
			timeSpent = rp.emtSec
		} else if rp.hasClock {
			if rp.turn == "white" {
				if hadPrevWhiteClock {
					// Time spent = previous white clock - current white clock
					diff := prevWhiteClock - rp.clockSec
					if diff >= 0 {
						timeSpent = diff
					}
				}
				prevWhiteClock = rp.clockSec
				hadPrevWhiteClock = true
			} else {
				if hadPrevBlackClock {
					diff := prevBlackClock - rp.clockSec
					if diff >= 0 {
						timeSpent = diff
					}
				}
				prevBlackClock = rp.clockSec
				hadPrevBlackClock = true
			}
		}

		// Determine Eval Drop
		evalDrop := 0
		classification := "unknown"
		if rp.hasEval && hadPrevEval {
			if rp.turn == "white" {
				// For white, drop is previous eval - current eval
				drop := prevEvalCp - rp.evalCp
				if drop > 0 {
					evalDrop = drop
				}
			} else {
				// For black, drop is current eval - previous eval
				drop := rp.evalCp - prevEvalCp
				if drop > 0 {
					evalDrop = drop
				}
			}
		}

		// Check NAGs for blunder/mistake classification
		for _, nag := range rp.nags {
			switch nag {
			case 1: // !
				classification = "best"
			case 2: // ?
				classification = "mistake"
				if evalDrop < 100 {
					evalDrop = 150
				}
			case 4: // ??
				classification = "blunder"
				if evalDrop < 250 {
					evalDrop = 300
				}
			case 5: // !?
				classification = "good"
			case 6: // ?!
				classification = "inaccuracy"
				if evalDrop < 50 {
					evalDrop = 60
				}
			}
		}

		if classification == "unknown" && rp.hasEval {
			if evalDrop >= 250 {
				classification = "blunder"
			} else if evalDrop >= 120 {
				classification = "mistake"
			} else if evalDrop >= 50 {
				classification = "inaccuracy"
			} else {
				classification = "good"
			}
		}

		if rp.hasEval {
			prevEvalCp = rp.evalCp
			hadPrevEval = true
		}

		// Flag Long Tanks: >= 120s or (> 60s and > 25% of remaining clock)
		isTank := false
		if timeSpent >= 120.0 || (timeSpent >= 60.0 && rp.hasClock && rp.clockSec > 0 && (timeSpent/(timeSpent+rp.clockSec)) > 0.25) {
			isTank = true
		}

		// Flag Panic Moves: <= 5s in non-opening with significant blunder
		isPanic := false
		if timeSpent > 0 && timeSpent <= 5.0 && rp.ply > 16 && (classification == "blunder" || classification == "mistake" || evalDrop >= 150) {
			isPanic = true
		}

		// Flag Time Scramble: clock <= 30 seconds
		isScramble := false
		if rp.hasClock && rp.clockSec <= 30.0 {
			isScramble = true
		}

		flag := "normal"
		if isTank {
			if classification == "blunder" || classification == "mistake" || evalDrop >= 150 {
				flag = "tank_blunder"
			} else {
				flag = "tank_success"
			}
		} else if isPanic {
			flag = "panic_blunder"
		} else if isScramble && (classification == "blunder" || classification == "mistake") {
			flag = "scramble_blunder"
		}

		rec := MoveTimeRecord{
			Ply:                      rp.ply,
			MoveNumber:               rp.moveNum,
			Turn:                     rp.turn,
			SAN:                      rp.san,
			UCI:                      rp.uci,
			ClockRemainingSec:        rp.clockSec,
			TimeSpentSec:             timeSpent,
			ClockStr:                 rp.clockStr,
			Phase:                    phase,
			IsTank:                   isTank,
			IsPanic:                  isPanic,
			IsScramble:               isScramble,
			EvalCp:                   rp.evalCp,
			EvalDropCp:               evalDrop,
			EvaluationClassification: classification,
			Flag:                     flag,
		}

		report.Moves = append(report.Moves, rec)

		// Record incidents
		if isTank {
			outcome := "Solid"
			if classification == "best" || (evalDrop == 0 && rp.hasEval) {
				outcome = "Masterstroke"
			} else if classification == "blunder" || evalDrop >= 250 {
				outcome = "Blunder"
			} else if classification == "mistake" || evalDrop >= 120 {
				outcome = "Inaccuracy"
			}

			playerName := report.WhitePlayer
			if rp.turn == "black" {
				playerName = report.BlackPlayer
			}

			tankInc := TankIncident{
				Ply:            rp.ply,
				MoveNumber:     rp.moveNum,
				Player:         playerName,
				Color:          rp.turn,
				SAN:            rp.san,
				TimeSpentSec:   timeSpent,
				ClockBeforeSec: rp.clockSec + timeSpent,
				ClockAfterSec:  rp.clockSec,
				EvalDropCp:     evalDrop,
				Outcome:        outcome,
				Description:    fmt.Sprintf("%s spent %.1fs (%s) on move %d...%s. Outcome: %s.", playerName, timeSpent, formatSeconds(timeSpent), rp.moveNum, rp.san, outcome),
			}
			report.TankDossier = append(report.TankDossier, tankInc)
		}

		if isPanic {
			playerName := report.WhitePlayer
			if rp.turn == "black" {
				playerName = report.BlackPlayer
			}
			panicInc := PanicIncident{
				Ply:               rp.ply,
				MoveNumber:        rp.moveNum,
				Player:            playerName,
				Color:             rp.turn,
				SAN:               rp.san,
				TimeSpentSec:      timeSpent,
				ClockRemainingSec: rp.clockSec,
				EvalDropCp:        evalDrop,
				Description:       fmt.Sprintf("%s played %d...%s hastily in %.1fs with %.1fs remaining on clock, incurring %d cp loss.", playerName, rp.moveNum, rp.san, timeSpent, rp.clockSec, evalDrop),
			}
			report.PanicIncidents = append(report.PanicIncidents, panicInc)
		}

		// Track Scramble entry
		if isScramble {
			if rp.turn == "white" {
				report.ScrambleSummary.WhiteScrambleMoves++
				if report.ScrambleSummary.WhiteEnteredScramblePly == 0 {
					report.ScrambleSummary.WhiteEnteredScramblePly = rp.ply
				}
				if classification == "blunder" || classification == "mistake" || evalDrop >= 150 {
					report.ScrambleSummary.WhiteScrambleBlunders++
				}
			} else {
				report.ScrambleSummary.BlackScrambleMoves++
				if report.ScrambleSummary.BlackEnteredScramblePly == 0 {
					report.ScrambleSummary.BlackEnteredScramblePly = rp.ply
				}
				if classification == "blunder" || classification == "mistake" || evalDrop >= 150 {
					report.ScrambleSummary.BlackScrambleBlunders++
				}
			}
		}
		_ = totalPlies
		_ = i
	}

	// Compute side stats
	report.WhiteStats = computeSideStats(report.Moves, "white")
	report.BlackStats = computeSideStats(report.Moves, "black")

	// Generate tactical and practical coaching recommendations
	generateRecommendations(report)

	return report, nil
}

func computeSideStats(moves []MoveTimeRecord, side string) SideTimeStats {
	var stats SideTimeStats
	var times []float64
	var openingTimes, middlegameTimes, endgameTimes []float64
	var timeLossPairs [][2]float64

	for _, m := range moves {
		if m.Turn != side {
			continue
		}
		stats.TotalMoves++
		stats.TotalTimeSpentSec += m.TimeSpentSec
		times = append(times, m.TimeSpentSec)

		switch m.Phase {
		case "opening":
			openingTimes = append(openingTimes, m.TimeSpentSec)
		case "middlegame":
			middlegameTimes = append(middlegameTimes, m.TimeSpentSec)
		case "endgame":
			endgameTimes = append(endgameTimes, m.TimeSpentSec)
		}

		if m.TimeSpentSec > stats.LongestTankSec {
			stats.LongestTankSec = m.TimeSpentSec
			stats.LongestTankPly = m.Ply
		}

		if m.IsTank {
			stats.TankCount++
			if m.Flag == "tank_blunder" {
				stats.TankBlunderCount++
			}
		}

		if m.IsPanic {
			stats.PanicMoveCount++
		}

		if m.IsScramble {
			stats.TimeTroubleMoves++
			if m.EvaluationClassification == "blunder" || m.EvaluationClassification == "mistake" || m.EvalDropCp >= 150 {
				stats.TimeTroubleBlunders++
			}
		}

		if m.EvalDropCp > 0 || m.EvaluationClassification != "unknown" {
			timeLossPairs = append(timeLossPairs, [2]float64{m.TimeSpentSec, float64(m.EvalDropCp)})
		}
	}

	if stats.TotalMoves > 0 {
		stats.AvgMoveTimeSec = stats.TotalTimeSpentSec / float64(stats.TotalMoves)
	}

	if len(times) > 0 {
		sortedTimes := make([]float64, len(times))
		copy(sortedTimes, times)
		sort.Float64s(sortedTimes)
		mid := len(sortedTimes) / 2
		if len(sortedTimes)%2 == 0 {
			stats.MedianMoveTimeSec = (sortedTimes[mid-1] + sortedTimes[mid]) / 2.0
		} else {
			stats.MedianMoveTimeSec = sortedTimes[mid]
		}
	}

	if len(openingTimes) > 0 {
		stats.OpeningAvgTimeSec = sum(openingTimes) / float64(len(openingTimes))
	}
	if len(middlegameTimes) > 0 {
		stats.MiddlegameAvgTimeSec = sum(middlegameTimes) / float64(len(middlegameTimes))
	}
	if len(endgameTimes) > 0 {
		stats.EndgameAvgTimeSec = sum(endgameTimes) / float64(len(endgameTimes))
	}

	// Calculate Pearson Correlation between time spent and centipawn loss
	if len(timeLossPairs) >= 5 {
		stats.CorrelationTimeVsLoss = calculatePearson(timeLossPairs)
	}

	return stats
}

func sum(vals []float64) float64 {
	total := 0.0
	for _, v := range vals {
		total += v
	}
	return total
}

func calculatePearson(pairs [][2]float64) float64 {
	n := float64(len(pairs))
	var sumX, sumY, sumXY, sumX2, sumY2 float64
	for _, p := range pairs {
		x := p[0]
		y := p[1]
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
		sumY2 += y * y
	}

	denom := math.Sqrt((n*sumX2 - sumX*sumX) * (n*sumY2 - sumY*sumY))
	if denom == 0 {
		return 0
	}
	return (n*sumXY - sumX*sumY) / denom
}

func formatSeconds(sec float64) string {
	m := int(sec) / 60
	s := int(sec) % 60
	if m > 0 {
		return fmt.Sprintf("%dm %02ds", m, s)
	}
	return fmt.Sprintf("%.1fs", sec)
}

func generateRecommendations(report *GameTimeManagementReport) {
	// White recommendations
	if report.WhiteStats.TankBlunderCount > 0 {
		report.Recommendations = append(report.Recommendations,
			fmt.Sprintf("⚠️ White suffered %d tank blunders (deep thinking episodes that still resulted in mistakes). Practice identifying 'candidate move exhaustion' when thinking beyond 3 minutes.", report.WhiteStats.TankBlunderCount),
		)
	}

	if report.WhiteStats.PanicMoveCount > 0 {
		report.Recommendations = append(report.Recommendations,
			fmt.Sprintf("⚡ White played %d panic moves (<5s with severe drops). Maintain a 10-second minimum 'blunder check' breath in critical positions.", report.WhiteStats.PanicMoveCount),
		)
	}

	if report.ScrambleSummary.WhiteEnteredScramblePly > 0 {
		report.Recommendations = append(report.Recommendations,
			fmt.Sprintf("⏱️ White entered severe time trouble at ply %d (%s moves in zeitnot, %d blunders). Aim to bank at least 2 minutes for the middlegame transition.",
				report.ScrambleSummary.WhiteEnteredScramblePly,
				formatSeconds(float64(report.ScrambleSummary.WhiteScrambleMoves)),
				report.ScrambleSummary.WhiteScrambleBlunders,
			),
		)
	}

	// Black recommendations
	if report.BlackStats.TankBlunderCount > 0 {
		report.Recommendations = append(report.Recommendations,
			fmt.Sprintf("⚠️ Black suffered %d tank blunders. Long tanks often correlate with emotional fatigue or trying to find wins in dry positions.", report.BlackStats.TankBlunderCount),
		)
	}

	if report.BlackStats.PanicMoveCount > 0 {
		report.Recommendations = append(report.Recommendations,
			fmt.Sprintf("⚡ Black rushed %d critical decisions under 5 seconds. Resist the urge to blitz back instantly when under pressure.", report.BlackStats.PanicMoveCount),
		)
	}

	if report.ScrambleSummary.BlackEnteredScramblePly > 0 {
		report.Recommendations = append(report.Recommendations,
			fmt.Sprintf("⏱️ Black fought in time trouble for %d moves with %d errors. Shift to solid, prophylactic moves early to avoid clock bleed.",
				report.ScrambleSummary.BlackScrambleMoves,
				report.ScrambleSummary.BlackScrambleBlunders,
			),
		)
	}

	if len(report.Recommendations) == 0 {
		report.Recommendations = append(report.Recommendations,
			"✅ Clean clock discipline demonstrated by both sides! Moves were paced evenly with no chronic zeitnot or impulsive blunder bursts.",
		)
	}
}

// AnalyzePlayerTimeManagement inspects all games for a specific player across a database.
func AnalyzePlayerTimeManagement(db *scid.Database, playerName string, maxGames int) (*PlayerTimeManagementDossier, error) {
	if db == nil || !db.IsOpen {
		return nil, fmt.Errorf("database is not open")
	}

	if maxGames <= 0 || maxGames > 300 {
		maxGames = 300
	}

	dossier := &PlayerTimeManagementDossier{
		PlayerName:      playerName,
		HabitFlags:      make([]string, 0),
		CoachingDossier: make([]string, 0),
	}

	cleanTarget := strings.ToLower(strings.TrimSpace(playerName))

	var totalMovesAll int
	var totalTimeAll float64
	var openingTimesAll, middlegameTimesAll, endgameTimesAll []float64
	var tankCountAll int
	var successfulTanksAll int
	var panicCountAll int
	var scrambleGames int
	var scrambleBlundersAll int

	gamesSampled := 0

	numGames := db.NumGames()
	for i := 0; i < numGames; i++ {
		if gamesSampled >= maxGames {
			break
		}

		db.RLock()
		if i >= len(db.Entries) {
			db.RUnlock()
			break
		}
		entry := db.Entries[i]
		if entry == nil || (entry.Flags&scid.FlagDelete) != 0 {
			db.RUnlock()
			continue
		}

		white := db.Names.GetName(scid.NamePlayer, entry.WhiteID)
		black := db.Names.GetName(scid.NamePlayer, entry.BlackID)
		db.RUnlock()

		isWhite := strings.Contains(strings.ToLower(white), cleanTarget)
		isBlack := strings.Contains(strings.ToLower(black), cleanTarget)

		if !isWhite && !isBlack {
			continue
		}

		game, err := db.GetGame(i + 1)
		if err != nil || game == nil || game.RootMove == nil {
			continue
		}

		dossier.GamesAnalyzed++
		gamesSampled++

		report, err := AnalyzeGameTimeManagement(game)
		if err != nil {
			continue
		}

		targetColor := "white"
		if isBlack {
			targetColor = "black"
		}

		hasClock := (targetColor == "white" && report.WhiteHasClock) || (targetColor == "black" && report.BlackHasClock)
		if !hasClock {
			continue
		}

		dossier.GamesWithClock++
		sideStats := report.WhiteStats
		if targetColor == "black" {
			sideStats = report.BlackStats
		}

		totalMovesAll += sideStats.TotalMoves
		totalTimeAll += sideStats.TotalTimeSpentSec
		tankCountAll += sideStats.TankCount
		successfulTanksAll += (sideStats.TankCount - sideStats.TankBlunderCount)
		panicCountAll += sideStats.PanicMoveCount

		for _, m := range report.Moves {
			if m.Turn != targetColor {
				continue
			}
			switch m.Phase {
			case "opening":
				openingTimesAll = append(openingTimesAll, m.TimeSpentSec)
			case "middlegame":
				middlegameTimesAll = append(middlegameTimesAll, m.TimeSpentSec)
			case "endgame":
				endgameTimesAll = append(endgameTimesAll, m.TimeSpentSec)
			}
		}

		inScramble := (targetColor == "white" && report.ScrambleSummary.WhiteEnteredScramblePly > 0) ||
			(targetColor == "black" && report.ScrambleSummary.BlackEnteredScramblePly > 0)
		if inScramble {
			scrambleGames++
			if targetColor == "white" {
				scrambleBlundersAll += report.ScrambleSummary.WhiteScrambleBlunders
			} else {
				scrambleBlundersAll += report.ScrambleSummary.BlackScrambleBlunders
			}
		}
	}

	if dossier.GamesWithClock > 0 {
		if totalMovesAll > 0 {
			dossier.AverageMoveTimeSec = totalTimeAll / float64(totalMovesAll)
		}
		if len(openingTimesAll) > 0 {
			dossier.OpeningAvgSec = sum(openingTimesAll) / float64(len(openingTimesAll))
		}
		if len(middlegameTimesAll) > 0 {
			dossier.MiddlegameAvgSec = sum(middlegameTimesAll) / float64(len(middlegameTimesAll))
		}
		if len(endgameTimesAll) > 0 {
			dossier.EndgameAvgSec = sum(endgameTimesAll) / float64(len(endgameTimesAll))
		}

		dossier.TotalTanks = tankCountAll
		if tankCountAll > 0 {
			dossier.TankSuccessRate = (float64(successfulTanksAll) / float64(tankCountAll)) * 100.0
		}
		dossier.PanicMoveCount = panicCountAll
		if totalMovesAll > 0 {
			dossier.PanicErrorRate = (float64(panicCountAll) / float64(totalMovesAll)) * 100.0
		}

		dossier.ScrambleFrequencyPct = (float64(scrambleGames) / float64(dossier.GamesWithClock)) * 100.0
		dossier.ScrambleBlunderCount = scrambleBlundersAll

		// Diagnose behavioral habits
		if dossier.OpeningAvgSec > 25.0 {
			dossier.HabitFlags = append(dossier.HabitFlags, "Chronic Opening Tanker: Consumes heavy clock before move 12")
			dossier.CoachingDossier = append(dossier.CoachingDossier, "Prepare repertoire fundamentals with NewScid's Opening Trainer to avoid early clock bleed.")
		}
		if dossier.ScrambleFrequencyPct >= 35.0 {
			dossier.HabitFlags = append(dossier.HabitFlags, "High Zeitnot Risk: Reaches <30s remaining in over a third of games")
			dossier.CoachingDossier = append(dossier.CoachingDossier, "Implement a hard 3-minute cap on non-forcing positional calculations.")
		}
		if dossier.TankSuccessRate < 60.0 && dossier.TotalTanks >= 3 {
			dossier.HabitFlags = append(dossier.HabitFlags, "Low Tank ROI: Tanks frequently lead to inaccurate decisions or blunders")
			dossier.CoachingDossier = append(dossier.CoachingDossier, "When a deep calculation goes beyond 4 minutes, stop looking for perfection and play a solid, improving move.")
		}
		if dossier.PanicMoveCount >= 3 {
			dossier.HabitFlags = append(dossier.HabitFlags, "Impulsive Responder: Rushes complex moves under 5 seconds")
			dossier.CoachingDossier = append(dossier.CoachingDossier, "Take a breath and verify piece safety before releasing pieces in tactical skirmishes.")
		}
	}

	if len(dossier.CoachingDossier) == 0 {
		dossier.CoachingDossier = append(dossier.CoachingDossier, "Solid, balanced pacing observed across all phases of the game.")
	}

	return dossier, nil
}
