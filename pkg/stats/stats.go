package stats

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"newscid/pkg/chess"
	"newscid/pkg/database/scid"
)

// RatingPoint tracks a player's rating at a specific game in time.
type RatingPoint struct {
	Date        string `json:"date"`
	Year        int    `json:"year"`
	Rating      int    `json:"rating"`
	Opponent    string `json:"opponent"`
	OpponentElo int    `json:"opponentElo"`
	Result      string `json:"result"` // "1-0", "0-1", "1/2-1/2"
	Event       string `json:"event"`
	ECO         string `json:"eco"`
	GameID      int    `json:"gameId"`
}

// RatingTrackerResult contains historical rating progression and statistics.
type RatingTrackerResult struct {
	Player      string        `json:"player"`
	TotalGames  int           `json:"totalGames"`
	RatedGames  int           `json:"ratedGames"`
	StartRating int           `json:"startRating"`
	EndRating   int           `json:"endRating"`
	PeakRating  int           `json:"peakRating"`
	PeakDate    string        `json:"peakDate"`
	MinRating   int           `json:"minRating"`
	MaxRating   int           `json:"maxRating"`
	Points      []RatingPoint `json:"points"`
}

// HistogramBin represents a bucket in a demographic chart.
type HistogramBin struct {
	Label string  `json:"label"`
	Count int     `json:"count"`
	Pct   float64 `json:"pct"`
}

// DatabaseDemographics represents aggregate statistics for an entire database.
type DatabaseDemographics struct {
	TotalGames     int            `json:"totalGames"`
	RatedGames     int            `json:"ratedGames"`
	AvgRating      int            `json:"avgRating"`
	AvgMoves       float64        `json:"avgMoves"`
	WhiteWins      int            `json:"whiteWins"`
	Draws          int            `json:"draws"`
	BlackWins      int            `json:"blackWins"`
	WhiteWinPct    float64        `json:"whiteWinPct"`
	DrawPct        float64        `json:"drawPct"`
	BlackWinPct    float64        `json:"blackWinPct"`
	DecadeDist     []HistogramBin `json:"decadeDist"`
	YearDist       []HistogramBin `json:"yearDist"`
	EloDist        []HistogramBin `json:"eloDist"`
	ECOSectionDist []HistogramBin `json:"ecoSectionDist"`
	TopECOs        []HistogramBin `json:"topEcos"`
	MoveLengthDist []HistogramBin `json:"moveLengthDist"`
}

// CalculateRatingTracker builds a chronological rating progression curve for a player.
func CalculateRatingTracker(db *scid.Database, playerAliases []string) (*RatingTrackerResult, error) {
	if db == nil || !db.IsOpen {
		return nil, fmt.Errorf("database not open")
	}
	if len(playerAliases) == 0 {
		return nil, fmt.Errorf("no player names provided")
	}

	db.RLock()
	defer db.RUnlock()

	// Normalize aliases for fast matching
	aliasSet := make(map[string]bool)
	aliasIDs := make(map[uint32]bool)
	for _, alias := range playerAliases {
		clean := strings.ToLower(strings.TrimSpace(alias))
		if clean != "" {
			aliasSet[clean] = true
			if id, ok := db.Names.GetID(scid.NamePlayer, alias); ok {
				aliasIDs[id] = true
			}
		}
	}

	var points []RatingPoint
	totalGames := 0

	for idx, entry := range db.Entries {
		if entry == nil || (entry.Flags&scid.FlagDelete) != 0 {
			continue
		}

		isWhite := aliasIDs[entry.WhiteID]
		isBlack := aliasIDs[entry.BlackID]

		if !isWhite && !isBlack {
			wName := strings.ToLower(db.Names.GetName(scid.NamePlayer, entry.WhiteID))
			bName := strings.ToLower(db.Names.GetName(scid.NamePlayer, entry.BlackID))
			if aliasSet[wName] {
				isWhite = true
			} else if aliasSet[bName] {
				isBlack = true
			}
		}

		if !isWhite && !isBlack {
			continue
		}

		totalGames++
		rating := int(entry.WhiteElo)
		oppElo := int(entry.BlackElo)
		oppName := db.Names.GetName(scid.NamePlayer, entry.BlackID)
		if isBlack {
			rating = int(entry.BlackElo)
			oppElo = int(entry.WhiteElo)
			oppName = db.Names.GetName(scid.NamePlayer, entry.WhiteID)
		}

		if rating <= 0 {
			continue
		}

		resStr := string(entry.Result)
		if resStr == "" {
			resStr = "*"
		}

		year := 0
		if len(entry.Date) >= 4 {
			year, _ = strconv.Atoi(entry.Date[:4])
		}

		points = append(points, RatingPoint{
			Date:        entry.Date,
			Year:        year,
			Rating:      rating,
			Opponent:    oppName,
			OpponentElo: oppElo,
			Result:      resStr,
			Event:       db.Names.GetName(scid.NameEvent, entry.EventID),
			ECO:         scid.FormatECO(entry.EcoCode),
			GameID:      idx + 1,
		})
	}

	// Sort chronologically by date
	sort.SliceStable(points, func(i, j int) bool {
		if points[i].Date != points[j].Date {
			return points[i].Date < points[j].Date
		}
		return points[i].GameID < points[j].GameID
	})

	primaryName := playerAliases[0]
	if len(points) == 0 {
		return &RatingTrackerResult{
			Player:      primaryName,
			TotalGames:  totalGames,
			RatedGames:  0,
			StartRating: 0,
			EndRating:   0,
			PeakRating:  0,
			MinRating:   0,
			MaxRating:   0,
			Points:      []RatingPoint{},
		}, nil
	}

	startRating := points[0].Rating
	endRating := points[len(points)-1].Rating
	peakRating := points[0].Rating
	peakDate := points[0].Date
	minRating := points[0].Rating
	maxRating := points[0].Rating

	for _, pt := range points {
		if pt.Rating > peakRating {
			peakRating = pt.Rating
			peakDate = pt.Date
		}
		if pt.Rating > maxRating {
			maxRating = pt.Rating
		}
		if pt.Rating < minRating {
			minRating = pt.Rating
		}
	}

	return &RatingTrackerResult{
		Player:      primaryName,
		TotalGames:  totalGames,
		RatedGames:  len(points),
		StartRating: startRating,
		EndRating:   endRating,
		PeakRating:  peakRating,
		PeakDate:    peakDate,
		MinRating:   minRating,
		MaxRating:   maxRating,
		Points:      points,
	}, nil
}

// CalculateDatabaseDemographics scans the database and calculates distributions.
func CalculateDatabaseDemographics(db *scid.Database) (*DatabaseDemographics, error) {
	if db == nil || !db.IsOpen {
		return nil, fmt.Errorf("database not open")
	}

	db.RLock()
	defer db.RUnlock()

	total := len(db.Entries)
	if total == 0 {
		return &DatabaseDemographics{}, nil
	}

	var (
		ratedCount    int
		totalEloSum   int64
		totalHalfMoves int64
		whiteWins     int
		draws         int
		blackWins     int
	)

	// Bins
	decadeCounts := make(map[string]int)
	yearCounts := make(map[int]int)
	ecoSectionCounts := make(map[string]int)
	ecoSpecificCounts := make(map[string]int)

	// Elo brackets: <1600, 1600-1799, 1800-1999, 2000-2199, 2200-2399, 2400-2599, 2600+
	eloBrackets := map[string]int{
		"< 1600":      0,
		"1600 - 1799": 0,
		"1800 - 1999": 0,
		"2000 - 2199": 0,
		"2200 - 2399": 0,
		"2400 - 2599": 0,
		"2600+":       0,
	}

	// Move lengths: 1-20, 21-40, 41-60, 61-80, 81+
	moveBrackets := map[string]int{
		"1 - 20":  0,
		"21 - 40": 0,
		"41 - 60": 0,
		"61 - 80": 0,
		"81+":     0,
	}

	activeGames := 0

	for _, entry := range db.Entries {
		if entry == nil || (entry.Flags&scid.FlagDelete) != 0 {
			continue
		}
		activeGames++

		// Results
		switch entry.Result {
		case chess.ResultWhiteWins:
			whiteWins++
		case chess.ResultDraw:
			draws++
		case chess.ResultBlackWins:
			blackWins++
		}

		// Move counts
		moves := int((entry.NumHalfMoves + 1) / 2)
		totalHalfMoves += int64(moves)

		switch {
		case moves <= 20:
			moveBrackets["1 - 20"]++
		case moves <= 40:
			moveBrackets["21 - 40"]++
		case moves <= 60:
			moveBrackets["41 - 60"]++
		case moves <= 80:
			moveBrackets["61 - 80"]++
		default:
			moveBrackets["81+"]++
		}

		// Elo
		wElo := int(entry.WhiteElo)
		bElo := int(entry.BlackElo)
		avgEntryElo := 0
		if wElo > 0 && bElo > 0 {
			avgEntryElo = (wElo + bElo) / 2
		} else if wElo > 0 {
			avgEntryElo = wElo
		} else if bElo > 0 {
			avgEntryElo = bElo
		}

		if avgEntryElo > 0 {
			ratedCount++
			totalEloSum += int64(avgEntryElo)
			switch {
			case avgEntryElo < 1600:
				eloBrackets["< 1600"]++
			case avgEntryElo < 1800:
				eloBrackets["1600 - 1799"]++
			case avgEntryElo < 2000:
				eloBrackets["1800 - 1999"]++
			case avgEntryElo < 2200:
				eloBrackets["2000 - 2199"]++
			case avgEntryElo < 2400:
				eloBrackets["2200 - 2399"]++
			case avgEntryElo < 2600:
				eloBrackets["2400 - 2599"]++
			default:
				eloBrackets["2600+"]++
			}
		}

		// Date & Decades
		if len(entry.Date) >= 4 {
			if yr, err := strconv.Atoi(entry.Date[:4]); err == nil && yr > 1400 && yr < 2100 {
				yearCounts[yr]++
				decade := (yr / 10) * 10
				decadeLabel := fmt.Sprintf("%ds", decade)
				decadeCounts[decadeLabel]++
			}
		}

		// ECO
		ecoStr := scid.FormatECO(entry.EcoCode)
		if len(ecoStr) >= 1 {
			sec := string(ecoStr[0])
			switch sec {
			case "A":
				ecoSectionCounts["A (Flank)"]++
			case "B":
				ecoSectionCounts["B (Semi-Open)"]++
			case "C":
				ecoSectionCounts["C (Open & French)"]++
			case "D":
				ecoSectionCounts["D (Closed)"]++
			case "E":
				ecoSectionCounts["E (Indian)"]++
			}
			ecoSpecificCounts[ecoStr]++
		}
	}

	if activeGames == 0 {
		return &DatabaseDemographics{}, nil
	}

	avgElo := 0
	if ratedCount > 0 {
		avgElo = int(totalEloSum / int64(ratedCount))
	}
	avgMoves := float64(totalHalfMoves) / float64(activeGames)

	whiteWinPct := float64(whiteWins) / float64(activeGames) * 100
	drawPct := float64(draws) / float64(activeGames) * 100
	blackWinPct := float64(blackWins) / float64(activeGames) * 100

	// Format Decade Dist
	var decadeBins []HistogramBin
	var decades []string
	for d := range decadeCounts {
		decades = append(decades, d)
	}
	sort.Strings(decades)
	for _, d := range decades {
		c := decadeCounts[d]
		decadeBins = append(decadeBins, HistogramBin{
			Label: d,
			Count: c,
			Pct:   float64(c) / float64(activeGames) * 100,
		})
	}

	// Format Top Years (up to 15 top years)
	type yrPair struct {
		yr int
		c  int
	}
	var yrPairs []yrPair
	for y, c := range yearCounts {
		yrPairs = append(yrPairs, yrPair{y, c})
	}
	sort.Slice(yrPairs, func(i, j int) bool {
		return yrPairs[i].yr < yrPairs[j].yr
	})
	var yearBins []HistogramBin
	for _, yp := range yrPairs {
		yearBins = append(yearBins, HistogramBin{
			Label: strconv.Itoa(yp.yr),
			Count: yp.c,
			Pct:   float64(yp.c) / float64(activeGames) * 100,
		})
	}

	// Format Elo Dist
	orderedEloLabels := []string{"< 1600", "1600 - 1799", "1800 - 1999", "2000 - 2199", "2200 - 2399", "2400 - 2599", "2600+"}
	var eloBins []HistogramBin
	for _, lbl := range orderedEloLabels {
		c := eloBrackets[lbl]
		pct := 0.0
		if ratedCount > 0 {
			pct = float64(c) / float64(ratedCount) * 100
		}
		eloBins = append(eloBins, HistogramBin{
			Label: lbl,
			Count: c,
			Pct:   pct,
		})
	}

	// Format ECO Section Dist
	orderedECOSections := []string{"A (Flank)", "B (Semi-Open)", "C (Open & French)", "D (Closed)", "E (Indian)"}
	var ecoSectionBins []HistogramBin
	for _, sec := range orderedECOSections {
		c := ecoSectionCounts[sec]
		ecoSectionBins = append(ecoSectionBins, HistogramBin{
			Label: sec,
			Count: c,
			Pct:   float64(c) / float64(activeGames) * 100,
		})
	}

	// Format Top ECOs
	type ecoPair struct {
		code string
		c    int
	}
	var topPairs []ecoPair
	for code, count := range ecoSpecificCounts {
		topPairs = append(topPairs, ecoPair{code, count})
	}
	sort.Slice(topPairs, func(i, j int) bool {
		return topPairs[i].c > topPairs[j].c
	})
	if len(topPairs) > 10 {
		topPairs = topPairs[:10]
	}
	var topEcoBins []HistogramBin
	for _, tp := range topPairs {
		topEcoBins = append(topEcoBins, HistogramBin{
			Label: tp.code,
			Count: tp.c,
			Pct:   float64(tp.c) / float64(activeGames) * 100,
		})
	}

	// Format Move Length Dist
	orderedMoveLabels := []string{"1 - 20", "21 - 40", "41 - 60", "61 - 80", "81+"}
	var moveBins []HistogramBin
	for _, lbl := range orderedMoveLabels {
		c := moveBrackets[lbl]
		moveBins = append(moveBins, HistogramBin{
			Label: lbl,
			Count: c,
			Pct:   float64(c) / float64(activeGames) * 100,
		})
	}

	return &DatabaseDemographics{
		TotalGames:     activeGames,
		RatedGames:     ratedCount,
		AvgRating:      avgElo,
		AvgMoves:       avgMoves,
		WhiteWins:      whiteWins,
		Draws:          draws,
		BlackWins:      blackWins,
		WhiteWinPct:    whiteWinPct,
		DrawPct:        drawPct,
		BlackWinPct:    blackWinPct,
		DecadeDist:     decadeBins,
		YearDist:       yearBins,
		EloDist:        eloBins,
		ECOSectionDist: ecoSectionBins,
		TopECOs:        topEcoBins,
		MoveLengthDist: moveBins,
	}, nil
}
