package scid

import (
	"fmt"
	"os"
	"strings"
	"time"

	"newscid/pkg/chess"
)

// ImportStats contains the summary of a batch PGN import.
type ImportStats struct {
	TotalParsed int   `json:"totalParsed"`
	Imported    int   `json:"imported"`
	Skipped     int   `json:"skipped"`
	Errors      int   `json:"errors"`
	DurationMs  int64 `json:"durationMs"`
	FirstGameID int   `json:"firstGameId"`
	LastGameID  int   `json:"lastGameId"`
}

// ImportPGNText parses and imports multiple PGN games from a string into the active database.
func (db *Database) ImportPGNText(pgnText string) (*ImportStats, error) {
	reader := strings.NewReader(pgnText)
	games, err := chess.ParsePGNDatabase(reader)
	if err != nil {
		return nil, fmt.Errorf("parsing PGN database: %w", err)
	}

	return db.ImportGames(games)
}

// ImportPGNFile parses and imports a PGN file directly into the active database.
func (db *Database) ImportPGNFile(filePath string) (*ImportStats, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("opening PGN file: %w", err)
	}
	defer file.Close()

	games, err := chess.ParsePGNDatabase(file)
	if err != nil {
		return nil, fmt.Errorf("parsing PGN file: %w", err)
	}

	return db.ImportGames(games)
}

// ImportGames batch-appends a slice of decoded games into the database with high performance.
func (db *Database) ImportGames(games []*chess.Game) (*ImportStats, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	if !db.IsOpen {
		return nil, fmt.Errorf("database not open")
	}

	startTime := time.Now()
	stats := &ImportStats{
		TotalParsed: len(games),
	}

	if len(games) == 0 {
		return stats, nil
	}

	sg4Path := db.BasePath + ".sg4"
	si4Path := db.BasePath + ".si4"

	// Open .sg4 for appending
	fSg, err := os.OpenFile(sg4Path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return nil, fmt.Errorf("opening sg4: %w", err)
	}
	defer fSg.Close()

	stat, err := fSg.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat sg4: %w", err)
	}
	currentOffset := uint32(stat.Size())

	// Open .si4 for appending entries
	fSi, err := os.OpenFile(si4Path, os.O_RDWR, 0644)
	if err != nil {
		return nil, fmt.Errorf("opening si4: %w", err)
	}
	defer fSi.Close()

	if _, err := fSi.Seek(0, os.SEEK_END); err != nil {
		return nil, fmt.Errorf("seeking si4: %w", err)
	}

	firstID := len(db.Entries) + 1
	var lastID int

	for _, g := range games {
		// Encode game
		rawBytes, err := EncodeGameData(g)
		if err != nil {
			stats.Errors++
			continue
		}

		length := uint32(len(rawBytes))
		if _, err := fSg.Write(rawBytes); err != nil {
			stats.Errors++
			continue
		}

		// Register names in NameBase
		wID := db.Names.AddName(NamePlayer, g.Tag("White"))
		bID := db.Names.AddName(NamePlayer, g.Tag("Black"))
		eID := db.Names.AddName(NameEvent, g.Tag("Event"))
		sID := db.Names.AddName(NameSite, g.Tag("Site"))
		rID := db.Names.AddName(NameRound, g.Tag("Round"))

		var wElo, bElo int
		if val := g.Tag("WhiteElo"); val != "" && val != "?" {
			fmt.Sscanf(val, "%d", &wElo)
		}
		if val := g.Tag("BlackElo"); val != "" && val != "?" {
			fmt.Sscanf(val, "%d", &bElo)
		}

		entry := &IndexEntry{
			Offset:       currentOffset,
			Length:       length,
			WhiteID:      wID,
			BlackID:      bID,
			EventID:      eID,
			SiteID:       sID,
			RoundID:      rID,
			WhiteElo:     uint16(wElo),
			BlackElo:     uint16(bElo),
			EcoCode:      parseECO(g.Tag("ECO")),
			Result:       g.Result,
			Date:         g.Tag("Date"),
			NumHalfMoves: uint16(g.PlyCount),
		}

		if err := WriteIndexEntry(fSi, entry); err != nil {
			stats.Errors++
			continue
		}

		db.Entries = append(db.Entries, entry)
		currentOffset += length
		stats.Imported++
		lastID = len(db.Entries)
	}

	// Update header in .si4
	db.Header.NumGames = uint32(len(db.Entries))
	if err := WriteIndexHeader(fSi, db.Header); err != nil {
		return nil, fmt.Errorf("updating index header: %w", err)
	}

	// Save updated NameBase to .sn4
	sn4Path := db.BasePath + ".sn4"
	if err := db.Names.WriteToFile(sn4Path); err != nil {
		return nil, fmt.Errorf("updating namebase: %w", err)
	}

	stats.DurationMs = time.Since(startTime).Milliseconds()
	if stats.Imported > 0 {
		stats.FirstGameID = firstID
		stats.LastGameID = lastID
	}

	return stats, nil
}
