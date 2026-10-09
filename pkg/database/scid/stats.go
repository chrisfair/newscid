package scid

import (
	"fmt"
	"os"
	"sort"
)

// NameCountPair represents an item and its frequency in a ranking.
type NameCountPair struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// DatabaseStatistics provides high-level demographic, analytical, and storage metrics.
type DatabaseStatistics struct {
	TotalGames     int              `json:"totalGames"`
	ActiveGames    int              `json:"activeGames"`
	DeletedGames   int              `json:"deletedGames"`
	EarliestYear   int              `json:"earliestYear"`
	LatestYear     int              `json:"latestYear"`
	WhiteWins      int              `json:"whiteWins"`
	BlackWins      int              `json:"blackWins"`
	Draws          int              `json:"draws"`
	UnknownResults int              `json:"unknownResults"`
	AverageElo     int              `json:"averageElo"`
	MaxElo         int              `json:"maxElo"`
	MinElo         int              `json:"minElo"`
	TopOpenings    []NameCountPair  `json:"topOpenings"`
	TopPlayers     []NameCountPair  `json:"topPlayers"`
	TotalSizeBytes int64            `json:"totalSizeBytes"`
	FileSizes      map[string]int64 `json:"fileSizes"`
}

// GetStatistics aggregates demographic, rating, result, and storage metrics for the active database.
func (db *Database) GetStatistics() (*DatabaseStatistics, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if !db.IsOpen {
		return nil, fmt.Errorf("database not open")
	}

	stats := &DatabaseStatistics{
		TotalGames:  len(db.Entries),
		FileSizes:   make(map[string]int64),
		TopOpenings: make([]NameCountPair, 0),
		TopPlayers:  make([]NameCountPair, 0),
	}

	siStat, _ := os.Stat(db.BasePath + ".si4")
	snStat, _ := os.Stat(db.BasePath + ".sn4")
	sgStat, _ := os.Stat(db.BasePath + ".sg4")

	if siStat != nil {
		stats.FileSizes["si4"] = siStat.Size()
		stats.TotalSizeBytes += siStat.Size()
	}
	if snStat != nil {
		stats.FileSizes["sn4"] = snStat.Size()
		stats.TotalSizeBytes += snStat.Size()
	}
	if sgStat != nil {
		stats.FileSizes["sg4"] = sgStat.Size()
		stats.TotalSizeBytes += sgStat.Size()
	}

	if len(db.Entries) == 0 {
		return stats, nil
	}

	var eloSum int64
	var eloCount int
	minElo := 9999
	maxElo := 0

	earliestYear := 9999
	latestYear := 0

	ecoCounts := make(map[string]int)
	playerCounts := make(map[string]int)

	for _, entry := range db.Entries {
		if (entry.Flags & FlagDelete) != 0 {
			stats.DeletedGames++
		} else {
			stats.ActiveGames++
		}

		// Results
		switch string(entry.Result) {
		case "1-0":
			stats.WhiteWins++
		case "0-1":
			stats.BlackWins++
		case "1/2-1/2":
			stats.Draws++
		default:
			stats.UnknownResults++
		}

		// Ratings
		wElo := int(entry.WhiteElo)
		bElo := int(entry.BlackElo)

		if wElo > 0 {
			eloSum += int64(wElo)
			eloCount++
			if wElo > maxElo {
				maxElo = wElo
			}
			if wElo < minElo {
				minElo = wElo
			}
		}
		if bElo > 0 {
			eloSum += int64(bElo)
			eloCount++
			if bElo > maxElo {
				maxElo = bElo
			}
			if bElo < minElo {
				minElo = bElo
			}
		}

		// Years
		if len(entry.Date) >= 4 {
			var y int
			if _, err := fmt.Sscanf(entry.Date[:4], "%d", &y); err == nil && y > 1000 && y < 3000 {
				if y < earliestYear {
					earliestYear = y
				}
				if y > latestYear {
					latestYear = y
				}
			}
		}

		// ECO Openings
		ecoStr := formatECO(entry.EcoCode)
		if ecoStr != "" {
			ecoCounts[ecoStr]++
		}

		// Players
		if db.Names != nil {
			wName := db.Names.GetName(NamePlayer, entry.WhiteID)
			bName := db.Names.GetName(NamePlayer, entry.BlackID)
			if wName != "" && wName != "?" {
				playerCounts[wName]++
			}
			if bName != "" && bName != "?" {
				playerCounts[bName]++
			}
		}
	}

	if eloCount > 0 {
		stats.AverageElo = int(eloSum / int64(eloCount))
		stats.MaxElo = maxElo
		stats.MinElo = minElo
	}
	if earliestYear != 9999 {
		stats.EarliestYear = earliestYear
		stats.LatestYear = latestYear
	}

	// Top ECOs
	var ecoList []NameCountPair
	for eco, count := range ecoCounts {
		ecoList = append(ecoList, NameCountPair{Name: eco, Count: count})
	}
	sort.Slice(ecoList, func(i, j int) bool {
		return ecoList[i].Count > ecoList[j].Count
	})
	if len(ecoList) > 8 {
		ecoList = ecoList[:8]
	}
	stats.TopOpenings = ecoList

	// Top Players
	var playerList []NameCountPair
	for player, count := range playerCounts {
		playerList = append(playerList, NameCountPair{Name: player, Count: count})
	}
	sort.Slice(playerList, func(i, j int) bool {
		return playerList[i].Count > playerList[j].Count
	})
	if len(playerList) > 8 {
		playerList = playerList[:8]
	}
	stats.TopPlayers = playerList

	return stats, nil
}
