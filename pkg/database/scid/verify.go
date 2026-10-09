package scid

import (
	"fmt"
	"os"
)

// VerificationReport contains detailed diagnostics on database health and file integrity.
type VerificationReport struct {
	TotalGames     int              `json:"totalGames"`
	ActiveGames    int              `json:"activeGames"`
	DeletedGames   int              `json:"deletedGames"`
	CorruptedGames int              `json:"corruptedGames"`
	Status         string           `json:"status"` // "OK", "WARNING", "CORRUPTED"
	Errors         []string         `json:"errors"`
	Warnings       []string         `json:"warnings"`
	FileSizes      map[string]int64 `json:"fileSizes"`
	NameCounts     map[string]int   `json:"nameCounts"`
}

// Verify runs comprehensive integrity checks across .si4 (index), .sn4 (names), and .sg4 (gamedata).
func (db *Database) Verify() (*VerificationReport, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if !db.IsOpen {
		return nil, fmt.Errorf("database not open")
	}

	report := &VerificationReport{
		TotalGames: len(db.Entries),
		Status:     "OK",
		Errors:     make([]string, 0),
		Warnings:   make([]string, 0),
		FileSizes:  make(map[string]int64),
		NameCounts: make(map[string]int),
	}

	si4Path := db.BasePath + ".si4"
	sn4Path := db.BasePath + ".sn4"
	sg4Path := db.BasePath + ".sg4"

	// 1. Stat files
	siStat, err := os.Stat(si4Path)
	if err != nil {
		report.Errors = append(report.Errors, fmt.Sprintf("Missing .si4 file: %v", err))
	} else {
		report.FileSizes["si4"] = siStat.Size()
	}

	snStat, err := os.Stat(sn4Path)
	if err != nil {
		report.Errors = append(report.Errors, fmt.Sprintf("Missing .sn4 file: %v", err))
	} else {
		report.FileSizes["sn4"] = snStat.Size()
	}

	sgStat, err := os.Stat(sg4Path)
	if err != nil {
		report.Errors = append(report.Errors, fmt.Sprintf("Missing .sg4 file: %v", err))
	} else {
		report.FileSizes["sg4"] = sgStat.Size()
	}

	// 2. Validate .si4 header and file length
	if siStat != nil && db.Header != nil {
		expectedSiSize := int64(IndexHeaderSize + len(db.Entries)*IndexEntrySize)
		if siStat.Size() != expectedSiSize {
			report.Warnings = append(report.Warnings,
				fmt.Sprintf("Index file size mismatch: actual %d bytes, expected %d bytes (%d games)",
					siStat.Size(), expectedSiSize, len(db.Entries)))
		}
		if db.Header.Version != 400 {
			report.Warnings = append(report.Warnings, fmt.Sprintf("Unexpected SCID version header: %d (expected 400)", db.Header.Version))
		}
	}

	// 3. Validate NameBase
	if db.Names != nil {
		pCount := len(db.Names.NamesByID[NamePlayer])
		eCount := len(db.Names.NamesByID[NameEvent])
		sCount := len(db.Names.NamesByID[NameSite])
		rCount := len(db.Names.NamesByID[NameRound])

		report.NameCounts["players"] = pCount
		report.NameCounts["events"] = eCount
		report.NameCounts["sites"] = sCount
		report.NameCounts["rounds"] = rCount
	}

	// 4. Validate entries and .sg4 offsets
	var sgSize int64 = 0
	if sgStat != nil {
		sgSize = sgStat.Size()
	}

	for i, entry := range db.Entries {
		if (entry.Flags & FlagDelete) != 0 {
			report.DeletedGames++
		} else {
			report.ActiveGames++
		}

		// Offset check
		endOffset := int64(entry.Offset) + int64(entry.Length)
		if sgSize > 0 && endOffset > sgSize {
			report.CorruptedGames++
			if len(report.Errors) < 10 {
				report.Errors = append(report.Errors,
					fmt.Sprintf("Game %d offset [%d, %d] exceeds .sg4 size (%d bytes)",
						i+1, entry.Offset, endOffset, sgSize))
			}
		}

		// Player name check
		if db.Names != nil {
			if _, exists := db.Names.NamesByID[NamePlayer][entry.WhiteID]; !exists && len(db.Names.NamesByID[NamePlayer]) > 0 {
				report.CorruptedGames++
				if len(report.Errors) < 10 {
					report.Errors = append(report.Errors, fmt.Sprintf("Game %d invalid WhiteID %d", i+1, entry.WhiteID))
				}
			}
			if _, exists := db.Names.NamesByID[NamePlayer][entry.BlackID]; !exists && len(db.Names.NamesByID[NamePlayer]) > 0 {
				report.CorruptedGames++
				if len(report.Errors) < 10 {
					report.Errors = append(report.Errors, fmt.Sprintf("Game %d invalid BlackID %d", i+1, entry.BlackID))
				}
			}
		}
	}

	if len(report.Errors) > 0 {
		report.Status = "CORRUPTED"
	} else if len(report.Warnings) > 0 {
		report.Status = "WARNING"
	}

	return report, nil
}
