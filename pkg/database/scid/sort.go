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

type sortKeyID uint8

const (
	sortKeyDate sortKeyID = iota
	sortKeyECO
	sortKeyWhite
	sortKeyBlack
	sortKeyPlayer
	sortKeyWhiteElo
	sortKeyBlackElo
	sortKeyMaxElo
	sortKeyEvent
	sortKeySite
	sortKeyRound
	sortKeyResult
	sortKeyMoves
)

func parseSortKeyID(field SortField) sortKeyID {
	switch strings.ToLower(strings.TrimSpace(string(field))) {
	case "eco":
		return sortKeyECO
	case "white":
		return sortKeyWhite
	case "black":
		return sortKeyBlack
	case "player":
		return sortKeyPlayer
	case "whiteelo":
		return sortKeyWhiteElo
	case "blackelo":
		return sortKeyBlackElo
	case "maxelo":
		return sortKeyMaxElo
	case "event":
		return sortKeyEvent
	case "site":
		return sortKeySite
	case "round":
		return sortKeyRound
	case "result":
		return sortKeyResult
	case "moves":
		return sortKeyMoves
	default:
		return sortKeyDate
	}
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

	keyID := parseSortKeyID(criteria.Field)

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
		switch keyID {
		case sortKeyECO:
			less = a.EcoCode < b.EcoCode
		case sortKeyWhite:
			less = getPlayer(a.WhiteID) < getPlayer(b.WhiteID)
		case sortKeyBlack:
			less = getPlayer(a.BlackID) < getPlayer(b.BlackID)
		case sortKeyPlayer:
			pA := getPlayer(a.WhiteID)
			pB := getPlayer(b.WhiteID)
			if pA != pB {
				less = pA < pB
			} else {
				less = getPlayer(a.BlackID) < getPlayer(b.BlackID)
			}
		case sortKeyWhiteElo:
			less = a.WhiteElo < b.WhiteElo
		case sortKeyBlackElo:
			less = a.BlackElo < b.BlackElo
		case sortKeyMaxElo:
			less = maxElo(a.WhiteElo, a.BlackElo) < maxElo(b.WhiteElo, b.BlackElo)
		case sortKeyEvent:
			less = getEvent(a.EventID) < getEvent(b.EventID)
		case sortKeySite:
			less = getSite(a.SiteID) < getSite(b.SiteID)
		case sortKeyRound:
			less = getRound(a.RoundID) < getRound(b.RoundID)
		case sortKeyResult:
			less = a.Result < b.Result
		case sortKeyMoves:
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
