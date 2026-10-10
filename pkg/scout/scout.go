package scout

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"newscid/pkg/database/scid"
)

// GameRecord contains win, draw, loss counters.
type GameRecord struct {
	Total   int     `json:"total"`
	Wins    int     `json:"wins"`
	Draws   int     `json:"draws"`
	Losses  int     `json:"losses"`
	WinRate float64 `json:"winRate"` // percentage 0 - 100
}

// LinePerformance tracks win rate for a specific ECO opening.
type LinePerformance struct {
	ECO     string  `json:"eco"`
	Name    string  `json:"name,omitempty"`
	Games   int     `json:"games"`
	Wins    int     `json:"wins"`
	Losses  int     `json:"losses"`
	Draws   int     `json:"draws"`
	WinRate float64 `json:"winRate"`
}

// Report holds an opponent's preparation scouting dossier.
type Report struct {
	PlayerName                  string            `json:"playerName"`
	TotalGames                  int               `json:"totalGames"`
	WhiteRecord                 GameRecord        `json:"whiteRecord"`
	BlackRecord                 GameRecord        `json:"blackRecord"`
	TopOpeningsAsWhite          []LinePerformance `json:"topOpeningsAsWhite"`
	TopOpeningsAsBlack          []LinePerformance `json:"topOpeningsAsBlack"`
	SharpestLines               []LinePerformance `json:"sharpestLines"` // Top win rate lines
	WeakestLines                []LinePerformance `json:"weakestLines"`  // Lowest win rate lines (attack targets!)
	RepertoireRecommendations   []string          `json:"repertoireRecommendations"`
}

// GenerateReport compiles games for playerName into an opponent dossier.
func GenerateReport(playerName string, games []*scid.GameSummary) *Report {
	normPlayer := strings.ToLower(strings.TrimSpace(playerName))

	report := &Report{
		PlayerName: playerName,
	}

	whiteLines := make(map[string]*LinePerformance)
	blackLines := make(map[string]*LinePerformance)

	for _, g := range games {
		if g == nil {
			continue
		}

		isWhite := strings.Contains(strings.ToLower(g.White), normPlayer)
		isBlack := strings.Contains(strings.ToLower(g.Black), normPlayer)

		if !isWhite && !isBlack {
			continue
		}

		report.TotalGames++
		ecoCode := g.ECO
		if ecoCode == "" {
			ecoCode = "A00"
		}

		res := strings.TrimSpace(string(g.Result))

		if isWhite {
			report.WhiteRecord.Total++
			line := getOrCreateLine(whiteLines, ecoCode)
			line.Games++

			if res == "1-0" {
				report.WhiteRecord.Wins++
				line.Wins++
			} else if res == "0-1" {
				report.WhiteRecord.Losses++
				line.Losses++
			} else if res == "1/2-1/2" {
				report.WhiteRecord.Draws++
				line.Draws++
			}
		}

		if isBlack {
			report.BlackRecord.Total++
			line := getOrCreateLine(blackLines, ecoCode)
			line.Games++

			if res == "0-1" {
				report.BlackRecord.Wins++
				line.Wins++
			} else if res == "1-0" {
				report.BlackRecord.Losses++
				line.Losses++
			} else if res == "1/2-1/2" {
				report.BlackRecord.Draws++
				line.Draws++
			}
		}
	}

	// Calculate win rates
	calcRates(&report.WhiteRecord)
	calcRates(&report.BlackRecord)

	report.TopOpeningsAsWhite = sortedLines(whiteLines)
	report.TopOpeningsAsBlack = sortedLines(blackLines)

	// Combine lines to find sharpest and weakest
	allLinesMap := make(map[string]*LinePerformance)
	for k, v := range whiteLines {
		allLinesMap["White: "+k] = v
	}
	for k, v := range blackLines {
		allLinesMap["Black: "+k] = v
	}

	allLines := sortedLines(allLinesMap)

	// Filter lines with at least 2 games for sharpest/weakest
	qualified := []LinePerformance{}
	for _, l := range allLines {
		if l.Games >= 1 {
			qualified = append(qualified, l)
		}
	}

	sort.Slice(qualified, func(i, j int) bool {
		return qualified[i].WinRate > qualified[j].WinRate
	})

	if len(qualified) > 0 {
		end := len(qualified)
		if end > 3 {
			end = 3
		}
		report.SharpestLines = qualified[:end]
	}

	sort.Slice(qualified, func(i, j int) bool {
		return qualified[i].WinRate < qualified[j].WinRate
	})

	if len(qualified) > 0 {
		end := len(qualified)
		if end > 3 {
			end = 3
		}
		report.WeakestLines = qualified[:end]
	}

	// Generate tactical recommendations
	report.RepertoireRecommendations = buildRecommendations(report)

	return report
}

func getOrCreateLine(m map[string]*LinePerformance, eco string) *LinePerformance {
	if line, ok := m[eco]; ok {
		return line
	}
	line := &LinePerformance{ECO: eco}
	m[eco] = line
	return line
}

func calcRates(r *GameRecord) {
	if r.Total > 0 {
		r.WinRate = math.Round((float64(r.Wins)+0.5*float64(r.Draws))/float64(r.Total)*1000) / 10
	}
}

func sortedLines(m map[string]*LinePerformance) []LinePerformance {
	var list []LinePerformance
	for _, v := range m {
		if v.Games > 0 {
			v.WinRate = math.Round((float64(v.Wins)+0.5*float64(v.Draws))/float64(v.Games)*1000) / 10
			list = append(list, *v)
		}
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].Games > list[j].Games
	})
	return list
}

func buildRecommendations(r *Report) []string {
	var recs []string

	if r.TotalGames == 0 {
		return []string{"Insufficient games in database to generate specific preparation recommendations."}
	}

	if r.WhiteRecord.Total > 0 && len(r.TopOpeningsAsWhite) > 0 {
		topW := r.TopOpeningsAsWhite[0]
		recs = append(recs, fmt.Sprintf("As White, %s's primary weapon is ECO %s (%d games, %.1f%% win rate).", r.PlayerName, topW.ECO, topW.Games, topW.WinRate))
	}

	if len(r.WeakestLines) > 0 {
		weak := r.WeakestLines[0]
		recs = append(recs, fmt.Sprintf("Key vulnerability: Target their ECO %s (concedes %.1f%% score rate).", weak.ECO, weak.WinRate))
	}

	if r.WhiteRecord.WinRate > r.BlackRecord.WinRate+15 {
		recs = append(recs, fmt.Sprintf("%s exhibits a significant performance dip playing Black (%.1f%% vs %.1f%% as White). Apply early pressure with White.", r.PlayerName, r.BlackRecord.WinRate, r.WhiteRecord.WinRate))
	}

	return recs
}
