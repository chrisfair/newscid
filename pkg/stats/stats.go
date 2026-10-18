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

type eloBracket uint8

const (
	eloUnder1600 eloBracket = iota
	elo1600To1799
	elo1800To1999
	elo2000To2199
	elo2200To2399
	elo2400To2599
	elo2600Plus
	numEloBrackets
)

var orderedEloLabels = [numEloBrackets]string{
	eloUnder1600:  "< 1600",
	elo1600To1799: "1600 - 1799",
	elo1800To1999: "1800 - 1999",
	elo2000To2199: "2000 - 2199",
	elo2200To2399: "2200 - 2399",
	elo2400To2599: "2400 - 2599",
	elo2600Plus:   "2600+",
}

type moveBracket uint8

const (
	moveBracket1To20 moveBracket = iota
	moveBracket21To40
	moveBracket41To60
	moveBracket61To80
	moveBracket81Plus
	numMoveBrackets
)

var orderedMoveLabels = [numMoveBrackets]string{
	moveBracket1To20:  "1 - 20",
	moveBracket21To40: "21 - 40",
	moveBracket41To60: "41 - 60",
	moveBracket61To80: "61 - 80",
	moveBracket81Plus: "81+",
}

type ecoSection uint8

const (
	ecoSectionA ecoSection = iota
	ecoSectionB
	ecoSectionC
	ecoSectionD
	ecoSectionE
	numECOSections
)

var orderedECOSections = [numECOSections]string{
	ecoSectionA: "A (Flank)",
	ecoSectionB: "B (Semi-Open)",
	ecoSectionC: "C (Open & French)",
	ecoSectionD: "D (Closed)",
	ecoSectionE: "E (Indian)",
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
		ratedCount     int
		totalEloSum    int64
		totalHalfMoves int64
		whiteWins      int
		draws          int
		blackWins      int
	)

	// Integer-keyed bins and fixed-size arrays for zero-allocation hot loop counting
	decadeCounts := make(map[int]int)
	yearCounts := make(map[int]int)
	var ecoSectionCounts [numECOSections]int
	ecoSpecificCounts := make(map[string]int)
	var eloBrackets [numEloBrackets]int
	var moveBrackets [numMoveBrackets]int

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
			moveBrackets[moveBracket1To20]++
		case moves <= 40:
			moveBrackets[moveBracket21To40]++
		case moves <= 60:
			moveBrackets[moveBracket41To60]++
		case moves <= 80:
			moveBrackets[moveBracket61To80]++
		default:
			moveBrackets[moveBracket81Plus]++
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
				eloBrackets[eloUnder1600]++
			case avgEntryElo < 1800:
				eloBrackets[elo1600To1799]++
			case avgEntryElo < 2000:
				eloBrackets[elo1800To1999]++
			case avgEntryElo < 2200:
				eloBrackets[elo2000To2199]++
			case avgEntryElo < 2400:
				eloBrackets[elo2200To2399]++
			case avgEntryElo < 2600:
				eloBrackets[elo2400To2599]++
			default:
				eloBrackets[elo2600Plus]++
			}
		}

		// Date & Decades
		if len(entry.Date) >= 4 {
			if yr, err := strconv.Atoi(entry.Date[:4]); err == nil && yr > 1400 && yr < 2100 {
				yearCounts[yr]++
				decade := (yr / 10) * 10
				decadeCounts[decade]++
			}
		}

		// ECO
		ecoStr := scid.FormatECO(entry.EcoCode)
		if len(ecoStr) >= 1 {
			switch ecoStr[0] {
			case 'A':
				ecoSectionCounts[ecoSectionA]++
			case 'B':
				ecoSectionCounts[ecoSectionB]++
			case 'C':
				ecoSectionCounts[ecoSectionC]++
			case 'D':
				ecoSectionCounts[ecoSectionD]++
			case 'E':
				ecoSectionCounts[ecoSectionE]++
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
	decades := make([]int, 0, len(decadeCounts))
	for d := range decadeCounts {
		decades = append(decades, d)
	}
	sort.Ints(decades)
	for _, d := range decades {
		c := decadeCounts[d]
		decadeBins = append(decadeBins, HistogramBin{
			Label: fmt.Sprintf("%ds", d),
			Count: c,
			Pct:   float64(c) / float64(activeGames) * 100,
		})
	}

	// Format Top Years
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
	eloBins := make([]HistogramBin, 0, numEloBrackets)
	for b := eloBracket(0); b < numEloBrackets; b++ {
		c := eloBrackets[b]
		pct := 0.0
		if ratedCount > 0 {
			pct = float64(c) / float64(ratedCount) * 100
		}
		eloBins = append(eloBins, HistogramBin{
			Label: orderedEloLabels[b],
			Count: c,
			Pct:   pct,
		})
	}

	// Format ECO Section Dist
	ecoSectionBins := make([]HistogramBin, 0, numECOSections)
	for s := ecoSection(0); s < numECOSections; s++ {
		c := ecoSectionCounts[s]
		ecoSectionBins = append(ecoSectionBins, HistogramBin{
			Label: orderedECOSections[s],
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
	moveBins := make([]HistogramBin, 0, numMoveBrackets)
	for b := moveBracket(0); b < numMoveBrackets; b++ {
		c := moveBrackets[b]
		moveBins = append(moveBins, HistogramBin{
			Label: orderedMoveLabels[b],
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

