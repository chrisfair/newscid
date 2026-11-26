package scid

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"newscid/pkg/fileformat"
)

// SortField defines the attribute used to order games in a database.
type SortField uint8

const (
	SortByDate SortField = iota
	SortByECO
	SortByWhite
	SortByBlack
	SortByPlayer
	SortByWhiteElo
	SortByBlackElo
	SortByMaxElo
	SortByEvent
	SortBySite
	SortByRound
	SortByResult
	SortByMoves
)

var sortFieldNames = [...]string{
	SortByDate:     "date",
	SortByECO:      "eco",
	SortByWhite:    "white",
	SortByBlack:    "black",
	SortByPlayer:   "player",
	SortByWhiteElo: "whiteElo",
	SortByBlackElo: "blackElo",
	SortByMaxElo:   "maxElo",
	SortByEvent:    "event",
	SortBySite:     "site",
	SortByRound:    "round",
	SortByResult:   "result",
	SortByMoves:    "moves",
}

// String returns the canonical JSON name of the sort field.
func (f SortField) String() string {
	if int(f) < len(sortFieldNames) {
		return sortFieldNames[f]
	}
	return "date"
}

// ParseSortField parses a sort field name (case-insensitive), defaulting to SortByDate.
func ParseSortField(s string) SortField {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "eco":
		return SortByECO
	case "white":
		return SortByWhite
	case "black":
		return SortByBlack
	case "player":
		return SortByPlayer
	case "whiteelo":
		return SortByWhiteElo
	case "blackelo":
		return SortByBlackElo
	case "maxelo":
		return SortByMaxElo
	case "event":
		return SortByEvent
	case "site":
		return SortBySite
	case "round":
		return SortByRound
	case "result":
		return SortByResult
	case "moves":
		return SortByMoves
	default:
		return SortByDate
	}
}

// MarshalJSON encodes SortField as its JSON string name.
func (f SortField) MarshalJSON() ([]byte, error) {
	return json.Marshal(f.String())
}

// UnmarshalJSON decodes SortField from its JSON string name.
func (f *SortField) UnmarshalJSON(data []byte) error {
	var raw string
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*f = ParseSortField(raw)
	return nil
}

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

	// Pre-cache lowercased name lookups for fast string comparisons without O(N log N) allocations
	playerNameCache := make(map[uint32]string)
	eventNameCache := make(map[uint32]string)
	siteNameCache := make(map[uint32]string)
	roundNameCache := make(map[uint32]string)

	getPlayer := func(id uint32) string {
		if s, ok := playerNameCache[id]; ok {
			return s
		}
		s := strings.ToLower(db.Names.GetName(NamePlayer, id))
		playerNameCache[id] = s
		return s
	}

	getEvent := func(id uint32) string {
		if s, ok := eventNameCache[id]; ok {
			return s
		}
		s := strings.ToLower(db.Names.GetName(NameEvent, id))
		eventNameCache[id] = s
		return s
	}

	getSite := func(id uint32) string {
		if s, ok := siteNameCache[id]; ok {
			return s
		}
		s := strings.ToLower(db.Names.GetName(NameSite, id))
		siteNameCache[id] = s
		return s
	}

	getRound := func(id uint32) string {
		if s, ok := roundNameCache[id]; ok {
			return s
		}
		s := strings.ToLower(db.Names.GetName(NameRound, id))
		roundNameCache[id] = s
		return s
	}

	sort.SliceStable(db.Entries, func(i, j int) bool {
		a := db.Entries[i]
		b := db.Entries[j]

		var less bool
		switch criteria.Field {
		case SortByECO:
			less = a.EcoCode < b.EcoCode
		case SortByWhite:
			less = getPlayer(a.WhiteID) < getPlayer(b.WhiteID)
		case SortByBlack:
			less = getPlayer(a.BlackID) < getPlayer(b.BlackID)
		case SortByPlayer:
			pA := getPlayer(a.WhiteID)
			pB := getPlayer(b.WhiteID)
			if pA != pB {
				less = pA < pB
			} else {
				less = getPlayer(a.BlackID) < getPlayer(b.BlackID)
			}
		case SortByWhiteElo:
			less = a.WhiteElo < b.WhiteElo
		case SortByBlackElo:
			less = a.BlackElo < b.BlackElo
		case SortByMaxElo:
			less = max(a.WhiteElo, a.BlackElo) < max(b.WhiteElo, b.BlackElo)
		case SortByEvent:
			less = getEvent(a.EventID) < getEvent(b.EventID)
		case SortBySite:
			less = getSite(a.SiteID) < getSite(b.SiteID)
		case SortByRound:
			less = getRound(a.RoundID) < getRound(b.RoundID)
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
	si4Path := fileformat.ScidIndexPath(db.BasePath)
	tmpSiPath := db.BasePath + fileformat.ExtScidIndexTemp

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
