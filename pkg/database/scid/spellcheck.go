package scid

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode"
)

// SpellCorrection records a single player name replacement.
type SpellCorrection struct {
	Original     string `json:"original"`
	Standardized string `json:"standardized"`
	GamesCount   int    `json:"gamesCount"`
}

// SpellCheckSummary details the results of standardizing player names across a database.
type SpellCheckSummary struct {
	TotalPlayersChecked int                `json:"totalPlayersChecked"`
	CorrectionsMade     int                `json:"correctionsMade"`
	GamesUpdated        int                `json:"gamesUpdated"`
	Corrections         []*SpellCorrection `json:"corrections"`
}

// SpellChecker normalizes player names and corrects spelling mistakes and format discrepancies.
type SpellChecker struct {
	CanonicalMap map[string]string // lowercase key -> Canonical Name e.g. "garry kasparov" -> "Kasparov, Garry"
	Aliases      map[string]string // lowercase key -> Canonical Name
}

// NewSpellChecker creates a SpellChecker with standard top master player defaults.
func NewSpellChecker() *SpellChecker {
	sc := &SpellChecker{
		CanonicalMap: make(map[string]string),
		Aliases:      make(map[string]string),
	}
	sc.loadBuiltinMasters()
	return sc
}

func (sc *SpellChecker) loadBuiltinMasters() {
	masters := []struct {
		Canonical string
		Aliases   []string
	}{
		{"Carlsen, Magnus", []string{"Carlsen, M.", "Carlsen, M", "Magnus Carlsen", "Carlsen Magnus"}},
		{"Kasparov, Garry", []string{"Kasparov, G.", "Kasparov, G", "Garry Kasparov", "Kasparov, Gary", "Kasparov Garry"}},
		{"Karpov, Anatoly", []string{"Karpov, A.", "Karpov, A", "Anatoly Karpov", "Karpov Anatoly"}},
		{"Fischer, Robert J", []string{"Fischer, R.", "Fischer, Bobby", "Bobby Fischer", "Robert James Fischer", "Fischer, Robert"}},
		{"Capablanca, Jose Raul", []string{"Capablanca, J.R.", "Capablanca, Jose", "Jose Raul Capablanca", "Capablanca, J."}},
		{"Alekhine, Alexander", []string{"Alekhine, A.", "Alexander Alekhine", "Aljechin, Alexander"}},
		{"Botvinnik, Mikhail", []string{"Botvinnik, M.", "Mikhail Botvinnik"}},
		{"Smyslov, Vasily", []string{"Smyslov, V.", "Vasily Smyslov"}},
		{"Tal, Mikhail", []string{"Tal, M.", "Mikhail Tal"}},
		{"Petrosian, Tigran", []string{"Petrosian, T.", "Tigran Petrosian"}},
		{"Spassky, Boris", []string{"Spassky, B.", "Boris Spassky"}},
		{"Kramnik, Vladimir", []string{"Kramnik, V.", "Vladimir Kramnik"}},
		{"Anand, Viswanathan", []string{"Anand, V.", "Viswanathan Anand"}},
		{"Topalov, Veselin", []string{"Topalov, V.", "Veselin Topalov"}},
		{"Ding, Liren", []string{"Ding Liren", "Ding, L.", "Liren Ding"}},
		{"Nepomniachtchi, Ian", []string{"Nepomniachtchi, I.", "Ian Nepomniachtchi", "Nepomniachtchi Ian"}},
		{"Nakamura, Hikaru", []string{"Nakamura, H.", "Hikaru Nakamura"}},
		{"Caruana, Fabiano", []string{"Caruana, F.", "Fabiano Caruana"}},
		{"So, Wesley", []string{"So, W.", "Wesley So"}},
		{"Aronian, Levon", []string{"Aronian, L.", "Levon Aronian"}},
		{"Giri, Anish", []string{"Giri, A.", "Anish Giri"}},
		{"Morphy, Paul", []string{"Morphy, P.", "Paul Morphy"}},
		{"Lasker, Emanuel", []string{"Lasker, E.", "Emanuel Lasker"}},
		{"Steinitz, William", []string{"Steinitz, W.", "Wilhelm Steinitz", "William Steinitz"}},
		{"Euwe, Max", []string{"Euwe, M.", "Max Euwe"}},
	}

	for _, m := range masters {
		sc.AddCanonical(m.Canonical, m.Aliases...)
	}
}

// AddCanonical registers a canonical player name along with known aliases.
func (sc *SpellChecker) AddCanonical(canonical string, aliases ...string) {
	canonical = strings.TrimSpace(canonical)
	if canonical == "" {
		return
	}
	key := strings.ToLower(canonical)
	sc.CanonicalMap[key] = canonical

	for _, a := range aliases {
		a = strings.TrimSpace(a)
		if a != "" {
			sc.Aliases[strings.ToLower(a)] = canonical
		}
	}
}

// LoadDictionary reads Scid format spelling dictionary lines (Canonical / Alias1 / Alias2...).
func (sc *SpellChecker) LoadDictionary(r io.Reader) error {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Split(line, "/")
		if len(parts) == 0 {
			continue
		}
		canonical := strings.TrimSpace(parts[0])
		var aliases []string
		for i := 1; i < len(parts); i++ {
			a := strings.TrimSpace(parts[i])
			if a != "" {
				aliases = append(aliases, a)
			}
		}
		sc.AddCanonical(canonical, aliases...)
	}
	return scanner.Err()
}

// StandardizeName formats a player name cleanly:
// 1. Strips titles (GM, IM, etc.)
// 2. Normalizes spacing, commas, and casing
// 3. Resolves known aliases and canonical dictionary matches
// 4. Converts "Firstname Lastname" to "Lastname, Firstname"
func (sc *SpellChecker) StandardizeName(raw string) string {
	cleaned := CleanPlayerName(raw)
	if cleaned == "" {
		return raw
	}

	// 1. Direct canonical match
	lower := strings.ToLower(cleaned)
	if canon, ok := sc.CanonicalMap[lower]; ok {
		return canon
	}

	// 2. Direct alias match
	if canon, ok := sc.Aliases[lower]; ok {
		return canon
	}

	// 3. Invert "Firstname Lastname" if no comma
	if !strings.Contains(cleaned, ",") {
		words := strings.Fields(cleaned)
		if len(words) == 2 {
			inverted := fmt.Sprintf("%s, %s", words[1], words[0])
			if canon, ok := sc.Aliases[strings.ToLower(inverted)]; ok {
				return canon
			}
			return inverted
		}
	}

	return cleaned
}

// CleanPlayerName removes title prefixes/suffixes, extra spaces, and formats commas.
func CleanPlayerName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}

	// Remove title prefixes e.g. "GM Garry Kasparov" -> "Garry Kasparov"
	titles := []string{"GM", "WGM", "IM", "WIM", "FM", "WFM", "CM", "WCM"}
	for _, t := range titles {
		prefix := t + " "
		if strings.HasPrefix(strings.ToUpper(name), prefix) {
			name = strings.TrimSpace(name[len(prefix):])
			break
		}
		suffix := " " + t
		if strings.HasSuffix(strings.ToUpper(name), suffix) {
			name = strings.TrimSpace(name[:len(name)-len(suffix)])
			break
		}
		commaSuffix := ", " + t
		if strings.HasSuffix(strings.ToUpper(name), commaSuffix) {
			name = strings.TrimSpace(name[:len(name)-len(commaSuffix)])
			break
		}
	}

	// Collapse multiple spaces
	words := strings.Fields(name)
	name = strings.Join(words, " ")

	// Ensure space after comma if comma present
	if strings.Contains(name, ",") {
		parts := strings.Split(name, ",")
		var cleanedParts []string
		for _, p := range parts {
			trimmed := strings.TrimSpace(p)
			if trimmed != "" {
				cleanedParts = append(cleanedParts, trimmed)
			}
		}
		name = strings.Join(cleanedParts, ", ")
	}

	return name
}

// FindSuggestions returns candidate canonical matches ranked by edit distance.
func (sc *SpellChecker) FindSuggestions(target string, maxSuggestions int) []string {
	target = strings.ToLower(CleanPlayerName(target))
	if target == "" {
		return nil
	}
	if maxSuggestions <= 0 {
		maxSuggestions = 5
	}

	type match struct {
		name string
		dist int
	}

	var matches []match
	for lower, canon := range sc.CanonicalMap {
		d := LevenshteinDistance(target, lower)
		if d <= 3 || (len(target) > 5 && d <= 4) {
			matches = append(matches, match{name: canon, dist: d})
		}
	}

	// Sort by smallest distance
	for i := 0; i < len(matches); i++ {
		for j := i + 1; j < len(matches); j++ {
			if matches[j].dist < matches[i].dist {
				matches[i], matches[j] = matches[j], matches[i]
			}
		}
	}

	var results []string
	seen := make(map[string]bool)
	for _, m := range matches {
		if !seen[m.name] {
			seen[m.name] = true
			results = append(results, m.name)
			if len(results) >= maxSuggestions {
				break
			}
		}
	}
	return results
}

// LevenshteinDistance calculates string edit distance.
func LevenshteinDistance(s, t string) int {
	sRunes := []rune(s)
	tRunes := []rune(t)
	sLen := len(sRunes)
	tLen := len(tRunes)

	if sLen == 0 {
		return tLen
	}
	if tLen == 0 {
		return sLen
	}

	d := make([][]int, sLen+1)
	for i := range d {
		d[i] = make([]int, tLen+1)
		d[i][0] = i
	}
	for j := 0; j <= tLen; j++ {
		d[0][j] = j
	}

	for i := 1; i <= sLen; i++ {
		for j := 1; j <= tLen; j++ {
			cost := 1
			if unicode.ToLower(sRunes[i-1]) == unicode.ToLower(tRunes[j-1]) {
				cost = 0
			}
			d[i][j] = min(
				d[i-1][j]+1,      // deletion
				d[i][j-1]+1,      // insertion
				d[i-1][j-1]+cost, // substitution
			)
		}
	}
	return d[sLen][tLen]
}

func min(a, b, c int) int {
	if a <= b && a <= c {
		return a
	}
	if b <= a && b <= c {
		return b
	}
	return c
}

// StandardizeDatabase scans all player names in the database, maps them to canonical
// standardized forms, updates IndexEntry pointers, and returns a detailed summary.
func (sc *SpellChecker) StandardizeDatabase(db *Database) (*SpellCheckSummary, error) {
	if db == nil {
		return nil, fmt.Errorf("database is not open")
	}
	db.mu.Lock()
	defer db.mu.Unlock()

	if !db.IsOpen {
		return nil, fmt.Errorf("database is not open")
	}

	summary := &SpellCheckSummary{
		Corrections: make([]*SpellCorrection, 0),
	}

	if db.Names == nil {
		return summary, nil
	}

	// 1. Identify all corrections needed
	idRemap := make(map[uint32]uint32)
	playerNames := db.Names.NamesByID[NamePlayer]
	summary.TotalPlayersChecked = len(playerNames)

	for id, original := range playerNames {
		standardized := sc.StandardizeName(original)
		if standardized != original {
			// Find or add standardized name
			targetID := db.Names.AddName(NamePlayer, standardized)
			idRemap[id] = targetID

			// Count occurrences in database
			freq := db.Names.Frequencies[NamePlayer][id]
			summary.Corrections = append(summary.Corrections, &SpellCorrection{
				Original:     original,
				Standardized: standardized,
				GamesCount:   int(freq),
			})
			summary.CorrectionsMade++
		}
	}

	if len(idRemap) == 0 {
		return summary, nil
	}

	// 2. Remap entries in database
	for _, entry := range db.Entries {
		if entry == nil || (entry.Flags&FlagDelete) != 0 {
			continue
		}
		updated := false
		if newWID, ok := idRemap[entry.WhiteID]; ok {
			entry.WhiteID = newWID
			updated = true
		}
		if newBID, ok := idRemap[entry.BlackID]; ok {
			entry.BlackID = newBID
			updated = true
		}
		if updated {
			summary.GamesUpdated++
		}
	}

	// 3. Write back updated index and namebase
	if db.BasePath != "" {
		sn4Path := db.BasePath + ".sn4"
		if err := db.Names.WriteToFile(sn4Path); err != nil {
			return summary, fmt.Errorf("saving namebase: %w", err)
		}

		si4Path := db.BasePath + ".si4"
		fSi, err := os.OpenFile(si4Path, os.O_RDWR, 0644)
		if err != nil {
			return summary, fmt.Errorf("opening index: %w", err)
		}
		defer fSi.Close()

		if _, err := fSi.Seek(int64(IndexHeaderSize), io.SeekStart); err != nil {
			return summary, fmt.Errorf("seeking index: %w", err)
		}

		for _, entry := range db.Entries {
			if err := WriteIndexEntry(fSi, entry); err != nil {
				return summary, fmt.Errorf("writing index entry: %w", err)
			}
		}
	}

	return summary, nil
}

// StandardizePlayerNames scans all player names in the database, normalizes and corrects them,
// updates index entries, writes back changes, and returns a detailed summary.
func (db *Database) StandardizePlayerNames() (*SpellCheckSummary, error) {
	sc := NewSpellChecker()
	return sc.StandardizeDatabase(db)
}

