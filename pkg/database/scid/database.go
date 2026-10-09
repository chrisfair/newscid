package scid

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"newscid/pkg/chess"
)

// GameSummary represents lightweight metadata for fast table display.
type GameSummary struct {
	ID        int          `json:"id"`
	White     string       `json:"white"`
	Black     string       `json:"black"`
	Result    chess.Result `json:"result"`
	Date      string       `json:"date"`
	Event     string       `json:"event"`
	Site      string       `json:"site"`
	Round     string       `json:"round"`
	WhiteElo  int          `json:"whiteElo"`
	BlackElo  int          `json:"blackElo"`
	ECO       string       `json:"eco"`
	Moves     int          `json:"moves"`
	Deleted   bool         `json:"deleted"`
}

// SearchCriteria defines filters for querying the database.
type SearchCriteria struct {
	White         string `json:"white,omitempty"`
	Black         string `json:"black,omitempty"`
	Player        string `json:"player,omitempty"` // White or Black
	Event         string `json:"event,omitempty"`
	Site          string `json:"site,omitempty"`
	Round         string `json:"round,omitempty"`
	DateFrom      string `json:"dateFrom,omitempty"`
	DateTo        string `json:"dateTo,omitempty"`
	YearFrom      int    `json:"yearFrom,omitempty"`
	YearTo        int    `json:"yearTo,omitempty"`
	Result        string `json:"result,omitempty"`
	ECO           string `json:"eco,omitempty"`
	EcoFrom       string `json:"ecoFrom,omitempty"`
	EcoTo         string `json:"ecoTo,omitempty"`
	MinElo        int    `json:"minElo,omitempty"`
	MaxElo        int    `json:"maxElo,omitempty"`
	WhiteEloMin   int    `json:"whiteEloMin,omitempty"`
	WhiteEloMax   int    `json:"whiteEloMax,omitempty"`
	BlackEloMin   int    `json:"blackEloMin,omitempty"`
	BlackEloMax   int    `json:"blackEloMax,omitempty"`
	MinMoves      int    `json:"minMoves,omitempty"`
	MaxMoves      int    `json:"maxMoves,omitempty"`
	HasVariations bool   `json:"hasVariations,omitempty"`
	IgnoreDeleted bool   `json:"ignoreDeleted,omitempty"`
	OnlyDeleted   bool   `json:"onlyDeleted,omitempty"`
}

// Database represents an open SCID database.
type Database struct {
	mu       sync.RWMutex
	BasePath string
	Header   *IndexHeader
	Names    *NameBase
	Entries  []*IndexEntry
	IsOpen   bool
}

func NewDatabase() *Database {
	return &Database{
		Names: NewNameBase(),
	}
}

// Open loads a SCID database from basePath (e.g. "path/to/mygames" or "mygames.si4").
func (db *Database) Open(path string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	base := strings.TrimSuffix(path, ".si4")
	base = strings.TrimSuffix(base, ".sn4")
	base = strings.TrimSuffix(base, ".sg4")
	db.BasePath = base

	si4Path := base + ".si4"
	sn4Path := base + ".sn4"
	sg4Path := base + ".sg4"

	if _, err := os.Stat(si4Path); err != nil {
		return fmt.Errorf("index file not found: %s", si4Path)
	}
	if _, err := os.Stat(sn4Path); err != nil {
		return fmt.Errorf("names file not found: %s", sn4Path)
	}
	if _, err := os.Stat(sg4Path); err != nil {
		return fmt.Errorf("game data file not found: %s", sg4Path)
	}

	// 1. Read names
	nb := NewNameBase()
	if err := nb.ReadFromFile(sn4Path); err != nil {
		return fmt.Errorf("reading namebase: %w", err)
	}
	db.Names = nb

	// 2. Read index
	header, entries, err := LoadAllEntries(si4Path)
	if err != nil {
		return fmt.Errorf("reading index: %w", err)
	}
	db.Header = header
	db.Entries = entries
	db.IsOpen = true

	return nil
}

// Create creates a new empty SCID database.
func (db *Database) Create(path string, description string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	base := strings.TrimSuffix(path, ".si4")
	base = strings.TrimSuffix(base, ".sn4")
	base = strings.TrimSuffix(base, ".sg4")
	db.BasePath = base

	db.Names = NewNameBase()
	db.Header = &IndexHeader{
		Version:     400,
		BaseType:    0,
		NumGames:    0,
		AutoLoad:    2,
		Description: description,
	}
	db.Entries = nil

	// Write empty files
	if err := db.Names.WriteToFile(base + ".sn4"); err != nil {
		return err
	}

	fSi, err := os.Create(base + ".si4")
	if err != nil {
		return err
	}
	if err := WriteIndexHeader(fSi, db.Header); err != nil {
		fSi.Close()
		return err
	}
	fSi.Close()

	fSg, err := os.Create(base + ".sg4")
	if err != nil {
		return err
	}
	fSg.Close()

	db.IsOpen = true
	return nil
}

// Close closes the database.
func (db *Database) Close() error {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.IsOpen = false
	db.Header = nil
	db.Names = NewNameBase()
	db.Entries = nil
	db.BasePath = ""
	return nil
}

func (db *Database) NumGames() int {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return len(db.Entries)
}

func (db *Database) GetGameSummary(gameNum int) (*GameSummary, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if gameNum < 1 || gameNum > len(db.Entries) {
		return nil, fmt.Errorf("game %d out of range (total %d)", gameNum, len(db.Entries))
	}

	entry := db.Entries[gameNum-1]
	summary := &GameSummary{
		ID:       gameNum,
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
	}
	return summary, nil
}

func formatECO(eco uint16) string {
	if eco == 0 {
		return ""
	}
	letter := byte('A' + (eco / 100))
	num := eco % 100
	return fmt.Sprintf("%c%02d", letter, num)
}

// matchesCriteria checks if an entry matches the given search criteria.
func (db *Database) matchesCriteria(entry *IndexEntry, criteria *SearchCriteria) bool {
	// 1. Deleted check
	isDeleted := (entry.Flags & FlagDelete) != 0
	if criteria.OnlyDeleted && !isDeleted {
		return false
	}
	if criteria.IgnoreDeleted && isDeleted {
		return false
	}

	// 2. Result check
	if criteria.Result != "" && string(entry.Result) != criteria.Result {
		return false
	}

	// 3. Ratings check
	wElo := int(entry.WhiteElo)
	bElo := int(entry.BlackElo)

	if criteria.MinElo > 0 && (wElo < criteria.MinElo && bElo < criteria.MinElo) {
		return false
	}
	if criteria.MaxElo > 0 && ((wElo > 0 && wElo > criteria.MaxElo) || (bElo > 0 && bElo > criteria.MaxElo)) {
		return false
	}
	if criteria.WhiteEloMin > 0 && wElo < criteria.WhiteEloMin {
		return false
	}
	if criteria.WhiteEloMax > 0 && wElo > 0 && wElo > criteria.WhiteEloMax {
		return false
	}
	if criteria.BlackEloMin > 0 && bElo < criteria.BlackEloMin {
		return false
	}
	if criteria.BlackEloMax > 0 && bElo > 0 && bElo > criteria.BlackEloMax {
		return false
	}

	// 4. Move count check
	moves := int((entry.NumHalfMoves + 1) / 2)
	if criteria.MinMoves > 0 && moves < criteria.MinMoves {
		return false
	}
	if criteria.MaxMoves > 0 && moves > criteria.MaxMoves {
		return false
	}

	// 5. Variations check
	if criteria.HasVariations && (entry.VarCounts&0x0FFF) == 0 {
		return false
	}

	// 6. ECO check
	ecoStr := formatECO(entry.EcoCode)
	if criteria.ECO != "" {
		critECO := strings.ToUpper(strings.TrimSpace(criteria.ECO))
		if !strings.HasPrefix(ecoStr, critECO) {
			return false
		}
	}
	if criteria.EcoFrom != "" {
		critFrom := strings.ToUpper(strings.TrimSpace(criteria.EcoFrom))
		if ecoStr < critFrom {
			return false
		}
	}
	if criteria.EcoTo != "" {
		critTo := strings.ToUpper(strings.TrimSpace(criteria.EcoTo))
		if ecoStr > critTo {
			return false
		}
	}

	// 7. Date & Year check
	var year int
	if len(entry.Date) >= 4 {
		fmt.Sscanf(entry.Date[:4], "%d", &year)
	}
	if criteria.YearFrom > 0 && (year == 0 || year < criteria.YearFrom) {
		return false
	}
	if criteria.YearTo > 0 && (year == 0 || year > criteria.YearTo) {
		return false
	}
	if criteria.DateFrom != "" {
		cleanFrom := strings.TrimSpace(criteria.DateFrom)
		if entry.Date != "" && entry.Date < cleanFrom {
			return false
		}
	}
	if criteria.DateTo != "" {
		cleanTo := strings.TrimSpace(criteria.DateTo)
		if entry.Date != "" && entry.Date > cleanTo {
			return false
		}
	}

	// 8. Name criteria (only lookup from namebase if requested)
	if criteria.Player != "" {
		white := db.Names.GetName(NamePlayer, entry.WhiteID)
		black := db.Names.GetName(NamePlayer, entry.BlackID)
		critP := strings.ToLower(criteria.Player)
		if !strings.Contains(strings.ToLower(white), critP) && !strings.Contains(strings.ToLower(black), critP) {
			return false
		}
	}
	if criteria.White != "" {
		white := db.Names.GetName(NamePlayer, entry.WhiteID)
		if !strings.Contains(strings.ToLower(white), strings.ToLower(criteria.White)) {
			return false
		}
	}
	if criteria.Black != "" {
		black := db.Names.GetName(NamePlayer, entry.BlackID)
		if !strings.Contains(strings.ToLower(black), strings.ToLower(criteria.Black)) {
			return false
		}
	}
	if criteria.Event != "" {
		event := db.Names.GetName(NameEvent, entry.EventID)
		if !strings.Contains(strings.ToLower(event), strings.ToLower(criteria.Event)) {
			return false
		}
	}
	if criteria.Site != "" {
		site := db.Names.GetName(NameSite, entry.SiteID)
		if !strings.Contains(strings.ToLower(site), strings.ToLower(criteria.Site)) {
			return false
		}
	}
	if criteria.Round != "" {
		round := db.Names.GetName(NameRound, entry.RoundID)
		if !strings.Contains(strings.ToLower(round), strings.ToLower(criteria.Round)) {
			return false
		}
	}

	return true
}

// CountMatches returns the total number of games matching the criteria.
func (db *Database) CountMatches(criteria SearchCriteria) int {
	db.mu.RLock()
	defer db.mu.RUnlock()

	count := 0
	for _, entry := range db.Entries {
		if db.matchesCriteria(entry, &criteria) {
			count++
		}
	}
	return count
}

// GetGameSummaries returns filtered and paginated game summaries.
func (db *Database) GetGameSummaries(offset, limit int, criteria SearchCriteria) ([]*GameSummary, int, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	var matched []*GameSummary
	for i, entry := range db.Entries {
		if !db.matchesCriteria(entry, &criteria) {
			continue
		}

		gameNum := i + 1
		s := &GameSummary{
			ID:       gameNum,
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
		}
		matched = append(matched, s)
	}

	total := len(matched)
	if offset >= total {
		return []*GameSummary{}, total, nil
	}
	end := offset + limit
	if end > total || limit <= 0 {
		end = total
	}

	return matched[offset:end], total, nil
}

// GetGame retrieves and decodes a game by 1-based index.
func (db *Database) GetGame(gameNum int) (*chess.Game, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if gameNum < 1 || gameNum > len(db.Entries) {
		return nil, fmt.Errorf("game %d out of range", gameNum)
	}

	entry := db.Entries[gameNum-1]
	sg4Path := db.BasePath + ".sg4"
	rawBytes, err := ReadGameBytes(sg4Path, entry.Offset, entry.Length)
	if err != nil {
		return nil, fmt.Errorf("reading raw game bytes: %w", err)
	}

	game, err := DecodeGameData(rawBytes, entry, db.Names)
	if err != nil {
		return nil, fmt.Errorf("decoding game: %w", err)
	}

	game.ID = gameNum
	return game, nil
}

// SaveGame appends a new game to the database.
func (db *Database) SaveGame(game *chess.Game) (int, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	if !db.IsOpen {
		return 0, fmt.Errorf("database not open")
	}

	// 1. Encode game data
	rawBytes, err := EncodeGameData(game)
	if err != nil {
		return 0, fmt.Errorf("encoding game data: %w", err)
	}

	// 2. Append to .sg4
	sg4Path := db.BasePath + ".sg4"
	fSg, err := os.OpenFile(sg4Path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return 0, err
	}
	stat, err := fSg.Stat()
	if err != nil {
		fSg.Close()
		return 0, err
	}
	offset := uint32(stat.Size())
	if _, err := fSg.Write(rawBytes); err != nil {
		fSg.Close()
		return 0, err
	}
	fSg.Close()

	// 3. Register names in NameBase
	wID := db.Names.AddName(NamePlayer, game.Tag("White"))
	bID := db.Names.AddName(NamePlayer, game.Tag("Black"))
	eID := db.Names.AddName(NameEvent, game.Tag("Event"))
	sID := db.Names.AddName(NameSite, game.Tag("Site"))
	rID := db.Names.AddName(NameRound, game.Tag("Round"))

	// Save updated NameBase
	if err := db.Names.WriteToFile(db.BasePath + ".sn4"); err != nil {
		return 0, fmt.Errorf("updating namebase: %w", err)
	}

	// 4. Create IndexEntry
	entry := &IndexEntry{
		Offset:       offset,
		Length:       uint32(len(rawBytes)),
		WhiteID:      wID,
		BlackID:      bID,
		EventID:      eID,
		SiteID:       sID,
		RoundID:      rID,
		Result:       game.Result,
		Date:         game.Tag("Date"),
		NumHalfMoves: uint16(game.PlyCount),
	}

	db.Entries = append(db.Entries, entry)
	db.Header.NumGames = uint32(len(db.Entries))

	// 5. Append to .si4 and update header
	si4Path := db.BasePath + ".si4"
	fSi, err := os.OpenFile(si4Path, os.O_RDWR, 0644)
	if err != nil {
		return 0, err
	}
	defer fSi.Close()

	// Rewrite header
	if err := WriteIndexHeader(fSi, db.Header); err != nil {
		return 0, err
	}

	// Seek to end of file to append entry
	if _, err := fSi.Seek(0, os.SEEK_END); err != nil {
		return 0, err
	}
	if err := WriteIndexEntry(fSi, entry); err != nil {
		return 0, err
	}

	return len(db.Entries), nil
}

// DeleteGame toggles the deleted flag of a game.
func (db *Database) DeleteGame(gameNum int, deleted bool) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if gameNum < 1 || gameNum > len(db.Entries) {
		return fmt.Errorf("game %d out of range", gameNum)
	}

	entry := db.Entries[gameNum-1]
	if deleted {
		entry.Flags |= FlagDelete
	} else {
		entry.Flags &= ^uint16(FlagDelete)
	}

	// Write entry back to .si4
	si4Path := db.BasePath + ".si4"
	fSi, err := os.OpenFile(si4Path, os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer fSi.Close()

	entryOffset := int64(IndexHeaderSize + (gameNum-1)*IndexEntrySize)
	if _, err := fSi.Seek(entryOffset, os.SEEK_SET); err != nil {
		return err
	}
	return WriteIndexEntry(fSi, entry)
}
