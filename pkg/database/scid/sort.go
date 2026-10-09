package scid

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

// SortField defines the attribute used to order games in a database.
type SortField string

const (
	SortByDate     SortField = "date"
	SortByECO      SortField = "eco"
	SortByWhite    SortField = "white"
	SortByBlack    SortField = "black"
	SortByPlayer   SortField = "player"
	SortByWhiteElo SortField = "whiteElo"
	SortByBlackElo SortField = "blackElo"
	SortByMaxElo   SortField = "maxElo"
	SortByEvent    SortField = "event"
	SortBySite     SortField = "site"
	SortByRound    SortField = "round"
	SortByResult   SortField = "result"
	SortByMoves    SortField = "moves"
)

// SortCriteria specifies the sorting field and order direction.
type SortCriteria struct {
	Field     SortField `json:"field"`
	Ascending bool      `json:"ascending"`
}

// Sort reorders the games in the database index (.si4) according to the specified criteria.
// Since each index entry contains its absolute byte offset into .sg4, sorting only rewrites
// the lightweight index file, achieving sub-second sorting across large databases.
func (db *Database) Sort(criteria SortCriteria) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if !db.IsOpen {
		return fmt.Errorf("database not open")
	}

	if len(db.Entries) <= 1 {
		return nil
	}

	field := SortField(strings.ToLower(string(criteria.Field)))
	if field == "" {
		field = SortByDate
	}

	// Pre-cache name lookups for fast string comparisons
	playerNameCache := make(map[uint32]string)
	eventNameCache := make(map[uint32]string)
	siteNameCache := make(map[uint32]string)
	roundNameCache := make(map[uint32]string)

	getPlayer := func(id uint32) string {
		if s, ok := playerNameCache[id]; ok {
			return s
		}
		s := db.Names.GetName(NamePlayer, id)
		playerNameCache[id] = s
		return s
	}

	getEvent := func(id uint32) string {
		if s, ok := eventNameCache[id]; ok {
			return s
		}
		s := db.Names.GetName(NameEvent, id)
		eventNameCache[id] = s
		return s
	}

	getSite := func(id uint32) string {
		if s, ok := siteNameCache[id]; ok {
			return s
		}
		s := db.Names.GetName(NameSite, id)
		siteNameCache[id] = s
		return s
	}

	getRound := func(id uint32) string {
		if s, ok := roundNameCache[id]; ok {
			return s
		}
		s := db.Names.GetName(NameRound, id)
		roundNameCache[id] = s
		return s
	}

	maxElo := func(w, b uint16) uint16 {
		if w > b {
			return w
		}
		return b
	}

	sort.SliceStable(db.Entries, func(i, j int) bool {
		a := db.Entries[i]
		b := db.Entries[j]

		var less bool
		switch field {
		case SortByDate:
			less = a.Date < b.Date
		case SortByECO:
			less = a.EcoCode < b.EcoCode
		case SortByWhite:
			less = strings.ToLower(getPlayer(a.WhiteID)) < strings.ToLower(getPlayer(b.WhiteID))
		case SortByBlack:
			less = strings.ToLower(getPlayer(a.BlackID)) < strings.ToLower(getPlayer(b.BlackID))
		case SortByPlayer:
			pA := strings.ToLower(getPlayer(a.WhiteID))
			pB := strings.ToLower(getPlayer(b.WhiteID))
			if pA != pB {
				less = pA < pB
			} else {
				less = strings.ToLower(getPlayer(a.BlackID)) < strings.ToLower(getPlayer(b.BlackID))
			}
		case SortByWhiteElo:
			less = a.WhiteElo < b.WhiteElo
		case SortByBlackElo:
			less = a.BlackElo < b.BlackElo
		case SortByMaxElo:
			less = maxElo(a.WhiteElo, a.BlackElo) < maxElo(b.WhiteElo, b.BlackElo)
		case SortByEvent:
			less = strings.ToLower(getEvent(a.EventID)) < strings.ToLower(getEvent(b.EventID))
		case SortBySite:
			less = strings.ToLower(getSite(a.SiteID)) < strings.ToLower(getSite(b.SiteID))
		case SortByRound:
			less = strings.ToLower(getRound(a.RoundID)) < strings.ToLower(getRound(b.RoundID))
		case SortByResult:
			less = a.Result < b.Result
		case SortByMoves:
			less = a.NumHalfMoves < b.NumHalfMoves
		default:
			less = a.Date < b.Date
		}

		if !criteria.Ascending {
			return !less
		}
		return less
	})

	// Rewrite .si4 index file
	si4Path := db.BasePath + ".si4"
	tmpSiPath := db.BasePath + ".si4.tmp"

	fSi, err := os.Create(tmpSiPath)
	if err != nil {
		return fmt.Errorf("creating temporary index: %w", err)
	}

	if err := WriteIndexHeader(fSi, db.Header); err != nil {
		fSi.Close()
		os.Remove(tmpSiPath)
		return fmt.Errorf("writing sorted index header: %w", err)
	}

	for _, entry := range db.Entries {
		if err := WriteIndexEntry(fSi, entry); err != nil {
			fSi.Close()
			os.Remove(tmpSiPath)
			return fmt.Errorf("writing sorted index entry: %w", err)
		}
	}

	if err := fSi.Close(); err != nil {
		os.Remove(tmpSiPath)
		return fmt.Errorf("closing sorted index: %w", err)
	}

	if err := os.Rename(tmpSiPath, si4Path); err != nil {
		return fmt.Errorf("updating index file: %w", err)
	}

	return nil
}
