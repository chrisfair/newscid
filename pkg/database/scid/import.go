package scid

import (
	"fmt"
	"io"
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

	if _, err := fSi.Seek(0, io.SeekEnd); err != nil {
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
	if _, err := fSi.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("seeking si4 to update header: %w", err)
	}
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

// GameSignatureSet holds fast O(1) fingerprints of existing games in a database
// so online syncs and database updates can skip already-present games without counting them.
type GameSignatureSet struct {
	sigs  map[string]struct{}
	count int
}

// NewGameSignatureSet creates an empty signature set.
func NewGameSignatureSet() *GameSignatureSet {
	return &GameSignatureSet{
		sigs: make(map[string]struct{}),
	}
}

// Count returns the number of games indexed in the signature set.
func (s *GameSignatureSet) Count() int {
	if s == nil {
		return 0
	}
	return s.count
}

func normalizeScidDate(dateStr string) string {
	clean := strings.ReplaceAll(strings.TrimSpace(dateStr), "-", ".")
	val := encodeDate(clean) & 0x000FFFFF
	year := int(val >> 9)
	month := int((val >> 5) & 15)
	day := int(val & 31)
	return formatDate(year, month, day)
}

func makeCanonicalGameKey(white, black, date, result string, halfMoves int) string {
	return fmt.Sprintf(
		"g:%s|%s|%s|%s|%d",
		normalizeScidDate(date),
		strings.ToLower(strings.TrimSpace(white)),
		strings.ToLower(strings.TrimSpace(black)),
		strings.TrimSpace(result),
		halfMoves,
	)
}

func makeSiteURLKey(site string) string {
	clean := strings.ToLower(strings.TrimSpace(site))
	if strings.HasPrefix(clean, "http://") || strings.HasPrefix(clean, "https://") {
		return "site:" + clean
	}
	return ""
}

// AddEntry registers an IndexEntry and its resolved names into the signature set.
func (s *GameSignatureSet) AddEntry(white, black, site, date, result string, halfMoves int) {
	if s == nil {
		return
	}
	if s.sigs == nil {
		s.sigs = make(map[string]struct{})
	}
	key := makeCanonicalGameKey(white, black, date, result, halfMoves)
	if _, exists := s.sigs[key]; !exists {
		s.sigs[key] = struct{}{}
		s.count++
	}
	if siteKey := makeSiteURLKey(site); siteKey != "" {
		s.sigs[siteKey] = struct{}{}
	}
}

// ContainsEntry checks whether an IndexEntry with the given metadata already exists in the set.
func (s *GameSignatureSet) ContainsEntry(white, black, site, date, result string, halfMoves int) bool {
	if s == nil || len(s.sigs) == 0 {
		return false
	}
	if siteKey := makeSiteURLKey(site); siteKey != "" {
		if _, ok := s.sigs[siteKey]; ok {
			return true
		}
	}
	key := makeCanonicalGameKey(white, black, date, result, halfMoves)
	_, ok := s.sigs[key]
	return ok
}

// ContainsGame checks whether a parsed chess.Game is already present in the signature set.
func (s *GameSignatureSet) ContainsGame(g *chess.Game) bool {
	if s == nil || g == nil || len(s.sigs) == 0 {
		return false
	}
	dateStr := g.Tag("Date")
	if (dateStr == "" || strings.Contains(dateStr, "?")) && g.Tag("UTCDate") != "" {
		dateStr = g.Tag("UTCDate")
	}
	site := g.Tag("Site")
	if link := g.Tag("Link"); strings.HasPrefix(strings.ToLower(link), "http") {
		if _, ok := s.sigs["site:"+strings.ToLower(strings.TrimSpace(link))]; ok {
			return true
		}
	}
	res := string(g.Result)
	if res == "" {
		res = g.Tag("Result")
	}
	return s.ContainsEntry(g.Tag("White"), g.Tag("Black"), site, dateStr, res, g.PlyCount)
}

// AddGame adds a parsed chess.Game to the signature set.
func (s *GameSignatureSet) AddGame(g *chess.Game) {
	if s == nil || g == nil {
		return
	}
	dateStr := g.Tag("Date")
	if (dateStr == "" || strings.Contains(dateStr, "?")) && g.Tag("UTCDate") != "" {
		dateStr = g.Tag("UTCDate")
	}
	res := string(g.Result)
	if res == "" {
		res = g.Tag("Result")
	}
	s.AddEntry(g.Tag("White"), g.Tag("Black"), g.Tag("Site"), dateStr, res, g.PlyCount)
	if link := g.Tag("Link"); strings.HasPrefix(strings.ToLower(link), "http") {
		s.sigs["site:"+strings.ToLower(strings.TrimSpace(link))] = struct{}{}
	}
}

// FilterNewGames filters a slice of games, returning only games not already in the signature set
// and updating the signature set with each newly accepted game.
func (s *GameSignatureSet) FilterNewGames(games []*chess.Game) ([]*chess.Game, int) {
	if s == nil {
		return games, 0
	}
	out := make([]*chess.Game, 0, len(games))
	skipped := 0
	for _, g := range games {
		if g == nil {
			continue
		}
		if s.ContainsGame(g) {
			skipped++
			continue
		}
		s.AddGame(g)
		out = append(out, g)
	}
	return out, skipped
}

// BuildGameSignatureSet constructs a fast O(1) lookup set of all non-deleted games in the database.
func (db *Database) BuildGameSignatureSet() *GameSignatureSet {
	set := NewGameSignatureSet()
	if db == nil {
		return set
	}
	db.mu.RLock()
	defer db.mu.RUnlock()
	if !db.IsOpen || len(db.Entries) == 0 {
		return set
	}
	set.sigs = make(map[string]struct{}, len(db.Entries)*2)
	for _, entry := range db.Entries {
		if entry == nil || (entry.Flags&FlagDelete) != 0 {
			continue
		}
		w := db.Names.GetName(NamePlayer, entry.WhiteID)
		b := db.Names.GetName(NamePlayer, entry.BlackID)
		site := db.Names.GetName(NameSite, entry.SiteID)
		set.AddEntry(w, b, site, entry.Date, string(entry.Result), int(entry.NumHalfMoves))
	}
	return set
}

// MergeStats summarizes updating/merging a database with one or more local databases.
type MergeStats struct {
	SourcesProcessed  int      `json:"sourcesProcessed"`
	SourceNames       []string `json:"sourceNames"`
	TotalScanned      int      `json:"totalScanned"`
	Imported          int      `json:"imported"`
	DuplicatesSkipped int      `json:"duplicatesSkipped"`
	Errors            int      `json:"errors"`
	DurationMs        int64    `json:"durationMs"`
	TargetTotalGames  int      `json:"targetTotalGames"`
}

// MergeFromDatabases updates db (even multi-million-game large databases) with all new, non-duplicate
// games from the specified local .si4 or .pgn source paths.
// For .si4 sources, raw .sg4 move streams are copied directly without move-tree decoding/re-encoding.
func (db *Database) MergeFromDatabases(
	sourcePaths []string,
	progressFn func(sourceName string, scanned, imported, skipped int),
) (*MergeStats, error) {
	if db == nil || !db.IsOpen {
		return nil, fmt.Errorf("target database is not open")
	}

	start := time.Now()
	existingSigs := db.BuildGameSignatureSet()
	targetClean := strings.TrimSuffix(strings.TrimSuffix(db.BasePath, ".si4"), ".pgn")

	res := &MergeStats{
		SourceNames: make([]string, 0, len(sourcePaths)),
	}

	db.mu.Lock()
	defer db.mu.Unlock()

	sg4Path := db.BasePath + ".sg4"
	si4Path := db.BasePath + ".si4"

	fSg, err := os.OpenFile(sg4Path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o644)
	if err != nil {
		return nil, fmt.Errorf("opening target sg4: %w", err)
	}
	defer fSg.Close()

	stat, err := fSg.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat target sg4: %w", err)
	}
	currentOffset := uint32(stat.Size())

	fSi, err := os.OpenFile(si4Path, os.O_RDWR, 0o644)
	if err != nil {
		return nil, fmt.Errorf("opening target si4: %w", err)
	}
	defer fSi.Close()

	if _, err := fSi.Seek(0, io.SeekEnd); err != nil {
		return nil, fmt.Errorf("seeking target si4: %w", err)
	}

	for _, rawSrc := range sourcePaths {
		srcClean := strings.TrimSpace(rawSrc)
		if srcClean == "" {
			continue
		}
		srcBase := strings.TrimSuffix(strings.TrimSuffix(srcClean, ".si4"), ".si3")
		if srcBase == targetClean {
			continue // Never merge a database into itself
		}

		// Case 1: PGN file source
		if strings.HasSuffix(strings.ToLower(srcClean), ".pgn") {
			fPgn, err := os.Open(srcClean)
			if err != nil {
				res.Errors++
				continue
			}
			games, err := chess.ParsePGNDatabase(fPgn)
			_ = fPgn.Close()
			if err != nil {
				res.Errors++
				continue
			}
			srcName := srcClean
			if idx := strings.LastIndexByte(srcClean, '/'); idx >= 0 {
				srcName = srcClean[idx+1:]
			}
			res.SourcesProcessed++
			res.SourceNames = append(res.SourceNames, srcName)

			for _, g := range games {
				if g == nil {
					continue
				}
				res.TotalScanned++
				if existingSigs.ContainsGame(g) {
					res.DuplicatesSkipped++
					continue
				}
				rawBytes, err := EncodeGameData(g)
				if err != nil {
					res.Errors++
					continue
				}
				length := uint32(len(rawBytes))
				if _, err := fSg.Write(rawBytes); err != nil {
					res.Errors++
					continue
				}

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
					res.Errors++
					continue
				}
				db.Entries = append(db.Entries, entry)
				currentOffset += length
				res.Imported++
				existingSigs.AddGame(g)
			}
			if progressFn != nil {
				progressFn(srcName, res.TotalScanned, res.Imported, res.DuplicatesSkipped)
			}
			continue
		}

		// Case 2: Scid .si4 database source (zero move-tree re-encoding!)
		if _, err := os.Stat(srcBase + ".si4"); err != nil {
			continue
		}
		srcDB := NewDatabase()
		if err := srcDB.Open(srcBase); err != nil {
			res.Errors++
			continue
		}

		srcSg, err := os.Open(srcBase + ".sg4")
		if err != nil {
			_ = srcDB.Close()
			res.Errors++
			continue
		}

		srcName := srcBase
		if idx := strings.LastIndexByte(srcBase, '/'); idx >= 0 {
			srcName = srcBase[idx+1:]
		}
		res.SourcesProcessed++
		res.SourceNames = append(res.SourceNames, srcName)

		for idx, srcEntry := range srcDB.Entries {
			if srcEntry == nil || (srcEntry.Flags&FlagDelete) != 0 {
				continue
			}
			res.TotalScanned++

			wName := srcDB.Names.GetName(NamePlayer, srcEntry.WhiteID)
			bName := srcDB.Names.GetName(NamePlayer, srcEntry.BlackID)
			eName := srcDB.Names.GetName(NameEvent, srcEntry.EventID)
			sName := srcDB.Names.GetName(NameSite, srcEntry.SiteID)
			rName := srcDB.Names.GetName(NameRound, srcEntry.RoundID)

			if existingSigs.ContainsEntry(wName, bName, sName, srcEntry.Date, string(srcEntry.Result), int(srcEntry.NumHalfMoves)) {
				res.DuplicatesSkipped++
				if idx%2000 == 0 && progressFn != nil {
					progressFn(srcName, res.TotalScanned, res.Imported, res.DuplicatesSkipped)
				}
				continue
			}

			rawBytes := make([]byte, srcEntry.Length)
			if _, err := srcSg.ReadAt(rawBytes, int64(srcEntry.Offset)); err != nil {
				res.Errors++
				continue
			}
			if _, err := fSg.Write(rawBytes); err != nil {
				res.Errors++
				continue
			}

			newEntry := &IndexEntry{
				Offset:       currentOffset,
				Length:       srcEntry.Length,
				Flags:        srcEntry.Flags &^ FlagDelete,
				WhiteID:      db.Names.AddName(NamePlayer, wName),
				BlackID:      db.Names.AddName(NamePlayer, bName),
				EventID:      db.Names.AddName(NameEvent, eName),
				SiteID:       db.Names.AddName(NameSite, sName),
				RoundID:      db.Names.AddName(NameRound, rName),
				Result:       srcEntry.Result,
				VarCounts:    srcEntry.VarCounts,
				EcoCode:      srcEntry.EcoCode,
				Date:         srcEntry.Date,
				EventDate:    srcEntry.EventDate,
				WhiteElo:     srcEntry.WhiteElo,
				BlackElo:     srcEntry.BlackElo,
				FinalMatSig:  srcEntry.FinalMatSig,
				NumHalfMoves: srcEntry.NumHalfMoves,
			}

			if err := WriteIndexEntry(fSi, newEntry); err != nil {
				res.Errors++
				continue
			}

			db.Entries = append(db.Entries, newEntry)
			currentOffset += srcEntry.Length
			res.Imported++
			existingSigs.AddEntry(wName, bName, sName, srcEntry.Date, string(srcEntry.Result), int(srcEntry.NumHalfMoves))

			if idx%1000 == 0 && progressFn != nil {
				progressFn(srcName, res.TotalScanned, res.Imported, res.DuplicatesSkipped)
			}
		}

		_ = srcSg.Close()
		_ = srcDB.Close()

		if progressFn != nil {
			progressFn(srcName, res.TotalScanned, res.Imported, res.DuplicatesSkipped)
		}
	}

	if res.Imported > 0 {
		if _, err := fSi.Seek(0, io.SeekStart); err == nil {
			db.Header.NumGames = uint32(len(db.Entries))
			_ = WriteIndexHeader(fSi, db.Header)
		}
		_ = db.Names.WriteToFile(db.BasePath + ".sn4")
	}

	res.TargetTotalGames = len(db.Entries)
	res.DurationMs = time.Since(start).Milliseconds()
	return res, nil
}

