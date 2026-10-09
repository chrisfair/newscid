package scid

import (
	"fmt"
	"io"
	"os"
)

// CompactStats holds metrics about a database compaction operation.
type CompactStats struct {
	OriginalGames    int   `json:"originalGames"`
	CompactedGames   int   `json:"compactedGames"`
	PurgedGames      int   `json:"purgedGames"`
	BytesReclaimed   int64 `json:"bytesReclaimed"`
	OriginalSg4Size  int64 `json:"originalSg4Size"`
	CompactedSg4Size int64 `json:"compactedSg4Size"`
}

// Compact physically removes deleted games from the database files (.si4 and .sg4),
// defragmenting game data offsets and reclaiming disk space.
func (db *Database) Compact() (*CompactStats, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	if !db.IsOpen {
		return nil, fmt.Errorf("database not open")
	}

	si4Path := db.BasePath + ".si4"
	sg4Path := db.BasePath + ".sg4"

	// Stat original files
	sgStat, err := os.Stat(sg4Path)
	if err != nil {
		return nil, fmt.Errorf("reading sg4 stat: %w", err)
	}
	originalSgSize := sgStat.Size()
	originalGames := len(db.Entries)

	// Check how many games are marked deleted
	deletedCount := 0
	for _, entry := range db.Entries {
		if (entry.Flags & FlagDelete) != 0 {
			deletedCount++
		}
	}

	if deletedCount == 0 {
		return &CompactStats{
			OriginalGames:    originalGames,
			CompactedGames:   originalGames,
			PurgedGames:      0,
			BytesReclaimed:   0,
			OriginalSg4Size:  originalSgSize,
			CompactedSg4Size: originalSgSize,
		}, nil
	}

	tmpSgPath := db.BasePath + ".sg4.tmp"
	tmpSiPath := db.BasePath + ".si4.tmp"

	// Open original sg4 for reading
	origSg, err := os.Open(sg4Path)
	if err != nil {
		return nil, fmt.Errorf("opening original sg4: %w", err)
	}
	defer origSg.Close()

	// Create temporary sg4 for compacted game data
	tmpSg, err := os.Create(tmpSgPath)
	if err != nil {
		return nil, fmt.Errorf("creating temporary sg4: %w", err)
	}

	var newEntries []*IndexEntry
	var currentOffset uint32 = 0

	// Copy only non-deleted games
	for _, entry := range db.Entries {
		if (entry.Flags & FlagDelete) != 0 {
			continue
		}

		rawBytes := make([]byte, entry.Length)
		if _, err := origSg.ReadAt(rawBytes, int64(entry.Offset)); err != nil && err != io.EOF {
			tmpSg.Close()
			os.Remove(tmpSgPath)
			return nil, fmt.Errorf("reading game at offset %d: %w", entry.Offset, err)
		}

		if _, err := tmpSg.Write(rawBytes); err != nil {
			tmpSg.Close()
			os.Remove(tmpSgPath)
			return nil, fmt.Errorf("writing compacted game data: %w", err)
		}

		// Create cloned entry with updated offset
		newEntry := *entry
		newEntry.Offset = currentOffset
		newEntries = append(newEntries, &newEntry)

		currentOffset += entry.Length
	}

	if err := tmpSg.Close(); err != nil {
		os.Remove(tmpSgPath)
		return nil, fmt.Errorf("closing temporary sg4: %w", err)
	}

	// Create temporary si4 index file
	tmpSi, err := os.Create(tmpSiPath)
	if err != nil {
		os.Remove(tmpSgPath)
		return nil, fmt.Errorf("creating temporary si4: %w", err)
	}

	// Update header
	newHeader := *db.Header
	newHeader.NumGames = uint32(len(newEntries))

	if err := WriteIndexHeader(tmpSi, &newHeader); err != nil {
		tmpSi.Close()
		os.Remove(tmpSgPath)
		os.Remove(tmpSiPath)
		return nil, fmt.Errorf("writing compacted header: %w", err)
	}

	for _, entry := range newEntries {
		if err := WriteIndexEntry(tmpSi, entry); err != nil {
			tmpSi.Close()
			os.Remove(tmpSgPath)
			os.Remove(tmpSiPath)
			return nil, fmt.Errorf("writing compacted index entry: %w", err)
		}
	}

	if err := tmpSi.Close(); err != nil {
		os.Remove(tmpSgPath)
		os.Remove(tmpSiPath)
		return nil, fmt.Errorf("closing temporary si4: %w", err)
	}

	// Safely replace original files
	if err := os.Rename(tmpSgPath, sg4Path); err != nil {
		return nil, fmt.Errorf("replacing sg4 file: %w", err)
	}
	if err := os.Rename(tmpSiPath, si4Path); err != nil {
		return nil, fmt.Errorf("replacing si4 file: %w", err)
	}

	compactedSgSize := int64(currentOffset)
	bytesReclaimed := originalSgSize - compactedSgSize
	if bytesReclaimed < 0 {
		bytesReclaimed = 0
	}

	// Update in-memory database state
	db.Header = &newHeader
	db.Entries = newEntries

	return &CompactStats{
		OriginalGames:    originalGames,
		CompactedGames:   len(newEntries),
		PurgedGames:      deletedCount,
		BytesReclaimed:   bytesReclaimed,
		OriginalSg4Size:  originalSgSize,
		CompactedSg4Size: compactedSgSize,
	}, nil
}
