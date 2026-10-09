package scid

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
)

// DuplicateCriteria specifies rules for identifying duplicate games.
type DuplicateCriteria struct {
	ExactMoves   bool `json:"exactMoves"`
	MatchPlayers bool `json:"matchPlayers"`
	MatchDate    bool `json:"matchDate"`
	MatchEvent   bool `json:"matchEvent"`
	MatchResult  bool `json:"matchResult"`
}

// DuplicateGroup represents a set of games identified as duplicates of each other.
type DuplicateGroup struct {
	Key     string         `json:"key"`
	GameIDs []int          `json:"gameIds"`
	Count   int            `json:"count"`
	Games   []*GameSummary `json:"games"`
}

// FindDuplicates scans the database and groups identical or duplicate games based on the criteria.
func (db *Database) FindDuplicates(criteria DuplicateCriteria) ([]DuplicateGroup, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if !db.IsOpen {
		return nil, fmt.Errorf("database not open")
	}

	groupsMap := make(map[string][]int)
	sg4Path := db.BasePath + ".sg4"

	var sgFile *os.File
	var err error
	if criteria.ExactMoves {
		sgFile, err = os.Open(sg4Path)
		if err != nil {
			return nil, fmt.Errorf("opening sg4 for duplicate scan: %w", err)
		}
		defer sgFile.Close()
	}

	for i, entry := range db.Entries {
		// Skip already deleted games from duplicate grouping
		if (entry.Flags & FlagDelete) != 0 {
			continue
		}

		gameNum := i + 1
		var key string

		if criteria.ExactMoves {
			// Read raw compressed move bytes from .sg4 and compute SHA-256 hash
			buf := make([]byte, entry.Length)
			if _, err := sgFile.ReadAt(buf, int64(entry.Offset)); err == nil {
				h := sha256.Sum256(buf)
				key = "moves:" + hex.EncodeToString(h[:])
			} else {
				key = fmt.Sprintf("meta:%d:%d:%s:%d", entry.WhiteID, entry.BlackID, entry.Date, entry.NumHalfMoves)
			}
		} else {
			// Construct metadata signature key
			key = fmt.Sprintf("p:%d:%d:m:%d", entry.WhiteID, entry.BlackID, entry.NumHalfMoves)
			if criteria.MatchDate {
				key += ":d:" + entry.Date
			}
			if criteria.MatchEvent {
				key += fmt.Sprintf(":e:%d", entry.EventID)
			}
			if criteria.MatchResult {
				key += ":r:" + string(entry.Result)
			}
		}

		groupsMap[key] = append(groupsMap[key], gameNum)
	}

	var duplicateGroups []DuplicateGroup
	for key, ids := range groupsMap {
		if len(ids) > 1 {
			var summaries []*GameSummary
			for _, id := range ids {
				entry := db.Entries[id-1]
				summaries = append(summaries, &GameSummary{
					ID:       id,
					White:    db.Names.GetName(NamePlayer, entry.WhiteID),
					Black:    db.Names.GetName(NamePlayer, entry.BlackID),
					Result:   entry.Result,
					Date:     entry.Date,
					Event:    db.Names.GetName(NameEvent, entry.EventID),
					Site:     db.Names.GetName(NameSite, entry.SiteID),
					Round:    db.Names.GetName(NameRound, entry.RoundID),
					WhiteElo: int(entry.WhiteElo),
					BlackElo: int(entry.BlackElo),
					ECO:      formatECO(entry.EcoCode),
					Moves:    int((entry.NumHalfMoves + 1) / 2),
					Deleted:  (entry.Flags & FlagDelete) != 0,
				})
			}

			duplicateGroups = append(duplicateGroups, DuplicateGroup{
				Key:     key,
				GameIDs: ids,
				Count:   len(ids),
				Games:   summaries,
			})
		}
	}

	return duplicateGroups, nil
}

// DeleteDuplicates finds duplicate games, preserves the first game in each group,
// and marks all redundant duplicate games as deleted (FlagDelete) in .si4.
func (db *Database) DeleteDuplicates(criteria DuplicateCriteria) (int, error) {
	groups, err := db.FindDuplicates(criteria)
	if err != nil {
		return 0, err
	}

	db.mu.Lock()
	defer db.mu.Unlock()

	deletedCount := 0
	si4Path := db.BasePath + ".si4"

	fSi, err := os.OpenFile(si4Path, os.O_RDWR, 0644)
	if err != nil {
		return 0, fmt.Errorf("opening index for duplicate removal: %w", err)
	}
	defer fSi.Close()

	for _, group := range groups {
		// Keep group.GameIDs[0], delete all subsequent games
		for i := 1; i < len(group.GameIDs); i++ {
			gameNum := group.GameIDs[i]
			entry := db.Entries[gameNum-1]
			if (entry.Flags & FlagDelete) == 0 {
				entry.Flags |= FlagDelete
				deletedCount++

				// Update on disk
				entryOffset := int64(IndexHeaderSize + (gameNum-1)*IndexEntrySize)
				if _, err := fSi.Seek(entryOffset, os.SEEK_SET); err == nil {
					_ = WriteIndexEntry(fSi, entry)
				}
			}
		}
	}

	return deletedCount, nil
}
