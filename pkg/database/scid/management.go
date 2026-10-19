package scid

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"newscid/pkg/chess"
)

// WritePolicyID defines the write and lock policy for a database.
type WritePolicyID uint8

const (
	WritePolicyEditable WritePolicyID = iota
	WritePolicySyncOnly
	WritePolicyLocked
)

// String returns the canonical string identifier for the write policy.
func (p WritePolicyID) String() string {
	switch p {
	case WritePolicySyncOnly:
		return "sync_only"
	case WritePolicyLocked:
		return "locked"
	default:
		return "editable"
	}
}

// ParseWritePolicy parses a string into a WritePolicyID, defaulting to WritePolicyEditable.
func ParseWritePolicy(s string) WritePolicyID {
	clean := strings.ToLower(strings.TrimSpace(s))
	switch clean {
	case "sync_only", "synconly":
		return WritePolicySyncOnly
	case "locked", "readonly", "read_only":
		return WritePolicyLocked
	default:
		return WritePolicyEditable
	}
}

// AllowsDirectEdit returns true if manual user edits (tag edits, move edits, manual game saves/deletes) are permitted.
func (p WritePolicyID) AllowsDirectEdit() bool {
	return p == WritePolicyEditable
}

// AllowsOnlineSync returns true if automated online sync or reference database updates are permitted.
func (p WritePolicyID) AllowsOnlineSync() bool {
	return p == WritePolicyEditable || p == WritePolicySyncOnly
}

var referenceDatabaseKeywords = []string{
	"mega",
	"giga",
	"lumbra",
	"caissabase",
	"lichess",
	"twic",
	"million",
	"master",
	"reference",
}

// InferDefaultWritePolicy determines the default write policy for a database based on its location, size, and metadata.
func InferDefaultWritePolicy(basePath string, numGames int, description string, isReference bool) WritePolicyID {
	cleanPath := strings.TrimSpace(basePath)
	lowerPath := strings.ToLower(cleanPath)

	if strings.HasPrefix(lowerPath, "/usr/share") {
		return WritePolicyLocked
	}
	if isReference {
		return WritePolicySyncOnly
	}
	if strings.Contains(lowerPath, "personas") || strings.Contains(lowerPath, "clipbase") {
		return WritePolicyEditable
	}

	baseName := strings.ToLower(filepath.Base(cleanPath))
	lowerDesc := strings.ToLower(strings.TrimSpace(description))
	for _, kw := range referenceDatabaseKeywords {
		if strings.Contains(baseName, kw) || strings.Contains(lowerDesc, kw) {
			return WritePolicySyncOnly
		}
	}

	if numGames >= 25000 {
		return WritePolicySyncOnly
	}

	return WritePolicyEditable
}

// SubDatabaseFilter specifies criteria for extracting a focused sub-database across multiple sources.
type SubDatabaseFilter struct {
	TargetName        string   `json:"targetName"`
	TargetDescription string   `json:"targetDescription"`
	TargetBasePath    string   `json:"targetBasePath,omitempty"`
	SourceBasePaths   []string `json:"sourceBasePaths"`
	Players           []string `json:"players,omitempty"`      // Matches White or Black across any of these player names/aliases
	PlayerColor       string   `json:"playerColor,omitempty"`  // "any", "white", "black"
	ECOFilter         string   `json:"ecoFilter,omitempty"`    // e.g. "B90", "C42", or range "B20-B99"
	OpeningMoves      []string `json:"openingMoves,omitempty"` // SAN or UCI move prefix sequence e.g. ["e4", "c5", "Nf3"] or ["1.e4", "c5"]
	MinElo            int      `json:"minElo,omitempty"`
	YearFrom          int      `json:"yearFrom,omitempty"`
	YearTo            int      `json:"yearTo,omitempty"`
	ResultFilter      string   `json:"resultFilter,omitempty"` // "", "1-0", "0-1", "1/2-1/2"
	Deduplicate       bool     `json:"deduplicate"`
	MaxGames          int      `json:"maxGames,omitempty"`
}

// SubDatabaseResult summarizes the outcome of building a focused sub-database.
type SubDatabaseResult struct {
	TargetBasePath    string `json:"targetBasePath"`
	TargetName        string `json:"targetName"`
	SourcesScanned    int    `json:"sourcesScanned"`
	TotalGamesScanned int    `json:"totalGamesScanned"`
	GamesMatched      int    `json:"gamesMatched"`
	DuplicatesSkipped int    `json:"duplicatesSkipped"`
}

func sanitizeFileName(name string) string {
	trimmed := strings.TrimSpace(name)
	var sb strings.Builder
	lastUnderscore := false
	for _, r := range trimmed {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' {
			sb.WriteRune(r)
			lastUnderscore = false
		} else if !lastUnderscore {
			sb.WriteByte('_')
			lastUnderscore = true
		}
	}
	res := strings.Trim(sb.String(), "_-")
	if res == "" {
		return "focused_subdb"
	}
	return res
}

func normalizeOpeningMoves(rawMoves []string) []string {
	var out []string
	for _, raw := range rawMoves {
		for _, tok := range strings.Fields(raw) {
			clean := strings.TrimSpace(tok)
			if dotIdx := strings.LastIndexByte(clean, '.'); dotIdx >= 0 {
				prefix := clean[:dotIdx]
				allDigitsOrDots := len(prefix) > 0
				for i := 0; i < len(prefix); i++ {
					if (prefix[i] < '0' || prefix[i] > '9') && prefix[i] != '.' {
						allDigitsOrDots = false
						break
					}
				}
				if allDigitsOrDots {
					clean = strings.TrimSpace(clean[dotIdx+1:])
				}
			}
			clean = strings.TrimRight(clean, "!?")
			if clean != "" {
				out = append(out, clean)
			}
		}
	}
	return out
}

func matchesOpeningMoves(root *chess.MoveNode, normMoves []string) bool {
	if len(normMoves) == 0 {
		return true
	}
	curr := root
	for _, want := range normMoves {
		if curr == nil {
			return false
		}
		sanClean := strings.TrimRight(curr.SAN, "+#!?")
		wantClean := strings.TrimRight(want, "+#!?")
		if !strings.EqualFold(curr.SAN, want) &&
			!strings.EqualFold(sanClean, wantClean) &&
			!strings.EqualFold(curr.UCI, want) {
			return false
		}
		curr = curr.Next
	}
	return true
}

func matchesECOFilter(gameECO, ecoFilter string) bool {
	crit := strings.ToUpper(strings.TrimSpace(ecoFilter))
	if crit == "" {
		return true
	}
	eco := strings.ToUpper(strings.TrimSpace(gameECO))
	if eco == "" || eco == "?" {
		return false
	}
	if strings.Contains(crit, "-") {
		parts := strings.SplitN(crit, "-", 2)
		from := strings.TrimSpace(parts[0])
		to := strings.TrimSpace(parts[1])
		cmpEco := eco
		if len(cmpEco) > 3 {
			cmpEco = cmpEco[:3]
		}
		return cmpEco >= from && cmpEco <= to
	}
	return strings.HasPrefix(eco, crit)
}

func playerMatchesDirect(gp, fp string) bool {
	if gp == "" || gp == "?" || fp == "" {
		return false
	}
	if strings.Contains(gp, fp) {
		return true
	}
	fpNoDot := strings.TrimRight(fp, ".")
	if len(fpNoDot) >= 3 && strings.Contains(gp, fpNoDot) {
		return true
	}
	if strings.Contains(fp, ",") {
		parts := strings.SplitN(fp, ",", 2)
		last := strings.TrimSpace(parts[0])
		first := strings.TrimSpace(strings.TrimRight(parts[1], "."))
		if first != "" && last != "" {
			firstLast := first + " " + last
			if strings.Contains(gp, firstLast) {
				return true
			}
		}
	} else {
		fields := strings.Fields(fp)
		if len(fields) >= 2 {
			last := fields[len(fields)-1]
			first := strings.Join(fields[:len(fields)-1], " ")
			lastFirst := last + ", " + first
			if strings.Contains(gp, lastFirst) {
				return true
			}
		}
	}
	return false
}

func playerMatchesSingle(gamePlayer, filterPlayer string) bool {
	gp := strings.ToLower(strings.TrimSpace(gamePlayer))
	fp := strings.ToLower(strings.TrimSpace(filterPlayer))
	if playerMatchesDirect(gp, fp) {
		return true
	}
	// Also check if filterPlayer matches a Famous Player's display name or alias (e.g. "Bobby Fischer" -> "Fischer, Robert James")
	for _, famous := range DefaultFamousPlayersCatalog() {
		if strings.EqualFold(famous.Name, fp) {
			for _, al := range famous.Aliases {
				if playerMatchesDirect(gp, strings.ToLower(strings.TrimSpace(al))) {
					return true
				}
			}
		}
	}
	return false
}

func playerMatchesAny(playerName string, filterPlayers []string) bool {
	for _, fp := range filterPlayers {
		if playerMatchesSingle(playerName, fp) {
			return true
		}
	}
	return false
}

func matchesPlayerFilter(white, black string, players []string, playerColor string) bool {
	if len(players) == 0 {
		return true
	}
	color := strings.ToLower(strings.TrimSpace(playerColor))
	switch color {
	case "white":
		return playerMatchesAny(white, players)
	case "black":
		return playerMatchesAny(black, players)
	default:
		return playerMatchesAny(white, players) || playerMatchesAny(black, players)
	}
}

func gameDedupSignature(g *chess.Game) string {
	if g == nil {
		return ""
	}
	white := strings.ToLower(strings.TrimSpace(g.Tag("White")))
	black := strings.ToLower(strings.TrimSpace(g.Tag("Black")))
	date := normalizeScidDate(g.Tag("Date"))
	res := string(gameResult(g))

	var movesBuilder strings.Builder
	for curr, ply := g.RootMove, 0; curr != nil && ply < 60; curr, ply = curr.Next, ply+1 {
		if curr.UCI != "" {
			movesBuilder.WriteString(strings.ToLower(curr.UCI))
		} else {
			movesBuilder.WriteString(curr.SAN)
		}
		movesBuilder.WriteByte(',')
	}
	return fmt.Sprintf("%s|%s|%s|%s|%s", white, black, date, res, movesBuilder.String())
}

func matchesMetadataFilter(white, black, eco, date, result string, whiteElo, blackElo int, filter SubDatabaseFilter) bool {
	if strings.TrimSpace(filter.ResultFilter) != "" && strings.TrimSpace(result) != strings.TrimSpace(filter.ResultFilter) {
		return false
	}
	if filter.MinElo > 0 && whiteElo < filter.MinElo && blackElo < filter.MinElo {
		return false
	}
	if filter.YearFrom > 0 || filter.YearTo > 0 {
		year := parseYear(date)
		if filter.YearFrom > 0 && (year == 0 || year < filter.YearFrom) {
			return false
		}
		if filter.YearTo > 0 && (year == 0 || year > filter.YearTo) {
			return false
		}
	}
	if !matchesECOFilter(eco, filter.ECOFilter) {
		return false
	}
	if !matchesPlayerFilter(white, black, filter.Players, filter.PlayerColor) {
		return false
	}
	return true
}

// BuildFocusedSubDatabase extracts matching games from one or more Scid or PGN databases into a new Scid database.
func BuildFocusedSubDatabase(
	filter SubDatabaseFilter,
	outputDir string,
	progressFn func(source string, scanned, matched int),
) (*SubDatabaseResult, error) {
	if len(filter.SourceBasePaths) == 0 {
		return nil, fmt.Errorf("at least one source database path is required")
	}

	targetName := strings.TrimSpace(filter.TargetName)
	targetBasePath := strings.TrimSpace(filter.TargetBasePath)
	if targetBasePath != "" {
		targetBasePath = strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(targetBasePath, ".si4"), ".sn4"), ".sg4")
		if targetName == "" {
			targetName = filepath.Base(targetBasePath)
		}
	} else {
		if targetName == "" {
			targetName = "focused_subdb"
		}
		if strings.TrimSpace(outputDir) == "" {
			return nil, fmt.Errorf("outputDir or targetBasePath is required")
		}
		targetBasePath = filepath.Join(outputDir, sanitizeFileName(targetName))
	}

	if err := os.MkdirAll(filepath.Dir(targetBasePath), 0o755); err != nil {
		return nil, fmt.Errorf("creating target directory: %w", err)
	}

	desc := strings.TrimSpace(filter.TargetDescription)
	if desc == "" {
		desc = targetName
	}

	targetDB := NewDatabase()
	if err := targetDB.Create(targetBasePath, desc); err != nil {
		return nil, fmt.Errorf("creating target database: %w", err)
	}
	defer targetDB.Close()

	normMoves := normalizeOpeningMoves(filter.OpeningMoves)
	seenSigs := make(map[string]struct{})

	result := &SubDatabaseResult{
		TargetBasePath: targetBasePath,
		TargetName:     targetName,
	}

	for _, rawSrc := range filter.SourceBasePaths {
		src := strings.TrimSpace(rawSrc)
		if src == "" {
			continue
		}
		if filter.MaxGames > 0 && result.GamesMatched >= filter.MaxGames {
			break
		}

		srcLabel := filepath.Base(src)

		if strings.HasSuffix(strings.ToLower(src), ".pgn") {
			f, err := os.Open(src)
			if err != nil {
				continue
			}
			games, err := chess.ParsePGNDatabase(f)
			_ = f.Close()
			if err != nil {
				continue
			}
			result.SourcesScanned++

			for _, g := range games {
				if g == nil {
					continue
				}
				if filter.MaxGames > 0 && result.GamesMatched >= filter.MaxGames {
					break
				}
				result.TotalGamesScanned++

				wElo, _ := strconv.Atoi(strings.TrimSpace(g.Tag("WhiteElo")))
				bElo, _ := strconv.Atoi(strings.TrimSpace(g.Tag("BlackElo")))
				resStr := string(gameResult(g))

				if !matchesMetadataFilter(g.Tag("White"), g.Tag("Black"), g.Tag("ECO"), g.Tag("Date"), resStr, wElo, bElo, filter) {
					continue
				}
				if !matchesOpeningMoves(g.RootMove, normMoves) {
					continue
				}
				if filter.Deduplicate {
					sig := gameDedupSignature(g)
					if _, exists := seenSigs[sig]; exists {
						result.DuplicatesSkipped++
						continue
					}
					seenSigs[sig] = struct{}{}
				}

				if _, err := targetDB.SaveGame(g); err == nil {
					result.GamesMatched++
				}
			}

			if progressFn != nil {
				progressFn(srcLabel, result.TotalGamesScanned, result.GamesMatched)
			}
			continue
		}

		cleanSrc := strings.TrimSuffix(strings.TrimSuffix(src, ".si4"), ".si3")
		if cleanSrc == targetBasePath {
			continue
		}
		srcDB := NewDatabase()
		if err := srcDB.Open(cleanSrc); err != nil {
			continue
		}
		result.SourcesScanned++

		for idx, entry := range srcDB.Entries {
			if entry == nil || (entry.Flags&FlagDelete) != 0 {
				continue
			}
			if filter.MaxGames > 0 && result.GamesMatched >= filter.MaxGames {
				break
			}
			result.TotalGamesScanned++

			white := srcDB.Names.GetName(NamePlayer, entry.WhiteID)
			black := srcDB.Names.GetName(NamePlayer, entry.BlackID)
			ecoStr := formatECO(entry.EcoCode)
			resStr := string(entry.Result)

			if !matchesMetadataFilter(white, black, ecoStr, entry.Date, resStr, int(entry.WhiteElo), int(entry.BlackElo), filter) {
				continue
			}

			g, err := srcDB.GetGame(idx + 1)
			if err != nil || g == nil {
				continue
			}
			if (g.Tag("ECO") == "" || g.Tag("ECO") == "?") && ecoStr != "" {
				g.SetTag("ECO", ecoStr)
			}

			if !matchesOpeningMoves(g.RootMove, normMoves) {
				continue
			}
			if filter.Deduplicate {
				sig := gameDedupSignature(g)
				if _, exists := seenSigs[sig]; exists {
					result.DuplicatesSkipped++
					continue
				}
				seenSigs[sig] = struct{}{}
			}

			if _, err := targetDB.SaveGame(g); err == nil {
				result.GamesMatched++
			}
		}

		_ = srcDB.Close()
		if progressFn != nil {
			progressFn(srcLabel, result.TotalGamesScanned, result.GamesMatched)
		}
	}

	return result, nil
}

// FamousPlayerEntry represents a catalog entry for a notable historical, elite, or custom player.
type FamousPlayerEntry struct {
	Name     string   `json:"name"`     // Display name, e.g. "Bobby Fischer"
	Title    string   `json:"title"`    // e.g. "World Champion (1972-1975)"
	Era      string   `json:"era"`      // e.g. "1955-1992"
	Aliases  []string `json:"aliases"`  // e.g. ["Fischer, Robert James", "Fischer, Robert J", "Fischer, Bobby", "Fischer, R."]
	IsCustom bool     `json:"isCustom"`
}

// DiscoveredNotablePlayer holds database statistics and matched aliases for a notable player.
type DiscoveredNotablePlayer struct {
	Name         string   `json:"name"`
	Title        string   `json:"title"`
	Era          string   `json:"era"`
	Aliases      []string `json:"aliases"`
	MatchedNames []string `json:"matchedNames"` // Exact player strings found in the open database
	GameCount    int      `json:"gameCount"`
	PeakElo      int      `json:"peakElo"`
	YearStart    int      `json:"yearStart"`
	YearEnd      int      `json:"yearEnd"`
	IsCustom     bool     `json:"isCustom"`
}

// DefaultFamousPlayersCatalog returns the curated catalog of 24+ world-famous historical and modern players.
func DefaultFamousPlayersCatalog() []FamousPlayerEntry {
	return []FamousPlayerEntry{
		{
			Name:    "Bobby Fischer",
			Title:   "11th World Champion (1972-1975)",
			Era:     "1955-1992",
			Aliases: []string{"Fischer, Robert James", "Fischer, Robert J", "Fischer, Bobby", "Fischer, R."},
		},
		{
			Name:    "Magnus Carlsen",
			Title:   "16th World Champion (2013-2023)",
			Era:     "2000-present",
			Aliases: []string{"Carlsen, Magnus", "Carlsen, M.", "DrNykterstein", "MagnusCarlsen"},
		},
		{
			Name:    "Garry Kasparov",
			Title:   "13th World Champion (1985-2000)",
			Era:     "1976-2005",
			Aliases: []string{"Kasparov, Garry", "Kasparov, Gary", "Kasparov, G."},
		},
		{
			Name:    "Anatoly Karpov",
			Title:   "12th World Champion (1975-1985)",
			Era:     "1966-present",
			Aliases: []string{"Karpov, Anatoly", "Karpov, A."},
		},
		{
			Name:    "Mikhail Tal",
			Title:   "8th World Champion (1960-1961)",
			Era:     "1950-1992",
			Aliases: []string{"Tal, Mikhail", "Tal, Mihail", "Tal, M."},
		},
		{
			Name:    "Jose Raul Capablanca",
			Title:   "3rd World Champion (1921-1927)",
			Era:     "1901-1939",
			Aliases: []string{"Capablanca, Jose Raul", "Capablanca, Jose", "Capablanca, J."},
		},
		{
			Name:    "Alexander Alekhine",
			Title:   "4th World Champion (1927-1935, 1937-1946)",
			Era:     "1907-1946",
			Aliases: []string{"Alekhine, Alexander", "Alekhine, A."},
		},
		{
			Name:    "Mikhail Botvinnik",
			Title:   "6th World Champion (1948-1963)",
			Era:     "1924-1970",
			Aliases: []string{"Botvinnik, Mikhail", "Botvinnik, M."},
		},
		{
			Name:    "Emanuel Lasker",
			Title:   "2nd World Champion (1894-1921)",
			Era:     "1889-1940",
			Aliases: []string{"Lasker, Emanuel"},
		},
		{
			Name:    "Paul Morphy",
			Title:   "Unofficial World Champion (1858-1859)",
			Era:     "1849-1864",
			Aliases: []string{"Morphy, Paul"},
		},
		{
			Name:    "Viswanathan Anand",
			Title:   "15th World Champion (2007-2013)",
			Era:     "1984-present",
			Aliases: []string{"Anand, Viswanathan", "Anand, V."},
		},
		{
			Name:    "Vladimir Kramnik",
			Title:   "14th World Champion (2000-2007)",
			Era:     "1989-2019",
			Aliases: []string{"Kramnik, Vladimir", "Kramnik, V."},
		},
		{
			Name:    "Hikaru Nakamura",
			Title:   "5-Time US Champion & Elite GM",
			Era:     "1998-present",
			Aliases: []string{"Nakamura, Hikaru", "Hikaru"},
		},
		{
			Name:    "Fabiano Caruana",
			Title:   "World Championship Challenger (2018)",
			Era:     "2005-present",
			Aliases: []string{"Caruana, Fabiano"},
		},
		{
			Name:    "Judit Polgar",
			Title:   "Strongest Female Player in History",
			Era:     "1988-2014",
			Aliases: []string{"Polgar, Judit"},
		},
		{
			Name:    "Boris Spassky",
			Title:   "10th World Champion (1969-1972)",
			Era:     "1952-2010",
			Aliases: []string{"Spassky, Boris V", "Spassky, Boris"},
		},
		{
			Name:    "Tigran Petrosian",
			Title:   "9th World Champion (1963-1969)",
			Era:     "1945-1984",
			Aliases: []string{"Petrosian, Tigran V", "Petrosian, Tigran"},
		},
		{
			Name:    "Viktor Korchnoi",
			Title:   "Two-Time World Championship Challenger",
			Era:     "1951-2015",
			Aliases: []string{"Korchnoi, Viktor", "Kortschnoj, Viktor"},
		},
		{
			Name:    "Vasily Smyslov",
			Title:   "7th World Champion (1957-1958)",
			Era:     "1938-2001",
			Aliases: []string{"Smyslov, Vasily", "Smyslov, Vassily"},
		},
		{
			Name:    "Ding Liren",
			Title:   "17th World Champion (2023-2024)",
			Era:     "2009-present",
			Aliases: []string{"Ding, Liren", "Ding Liren"},
		},
		{
			Name:    "Dommaraju Gukesh",
			Title:   "18th World Champion (2024-present)",
			Era:     "2017-present",
			Aliases: []string{"Gukesh, D", "Gukesh D", "Dommaraju, Gukesh"},
		},
		{
			Name:    "Alireza Firouzja",
			Title:   "Youngest 2800+ Rated Player",
			Era:     "2015-present",
			Aliases: []string{"Firouzja, Alireza"},
		},
		{
			Name:    "Akiba Rubinstein",
			Title:   "Classical Grandmaster & Endgame Virtuoso",
			Era:     "1903-1932",
			Aliases: []string{"Rubinstein, Akiba"},
		},
		{
			Name:    "Aron Nimzowitsch",
			Title:   "Hypermodern Pioneer & Author of My System",
			Era:     "1904-1934",
			Aliases: []string{"Nimzowitsch, Aaron", "Nimzowitsch, Aron"},
		},
	}
}

func matchesCatalogEntryName(dbName string, entry FamousPlayerEntry) bool {
	lowerDB := strings.ToLower(strings.TrimSpace(dbName))
	if lowerDB == "" || lowerDB == "?" {
		return false
	}

	candidates := make([]string, 0, len(entry.Aliases)+2)
	candidates = append(candidates, entry.Aliases...)
	if strings.TrimSpace(entry.Name) != "" {
		candidates = append(candidates, entry.Name)
		fields := strings.Fields(entry.Name)
		if len(fields) >= 2 && !strings.Contains(entry.Name, ",") {
			lastFirst := fields[len(fields)-1] + ", " + strings.Join(fields[:len(fields)-1], " ")
			candidates = append(candidates, lastFirst)
		}
	}

	for _, alias := range candidates {
		lowerAlias := strings.ToLower(strings.TrimSpace(alias))
		if lowerAlias == "" {
			continue
		}
		if lowerDB == lowerAlias {
			return true
		}
		prefix := strings.TrimRight(lowerAlias, ".")
		if lowerDB == prefix {
			return true
		}
		if strings.Contains(prefix, ",") && strings.HasPrefix(lowerDB, prefix) {
			rest := lowerDB[len(prefix):]
			if rest == "" || rest[0] == ' ' || rest[0] == '.' || rest[0] == ',' || rest[0] == '(' {
				return true
			}
			// Handle middle initial prefix like "fischer, robert j" matching "fischer, robert james"
			commaIdx := strings.IndexByte(prefix, ',')
			afterComma := strings.TrimSpace(prefix[commaIdx+1:])
			if len(afterComma) >= 3 && afterComma[len(afterComma)-2] == ' ' {
				return true
			}
		}
	}
	return false
}

// DiscoverNotablePlayersInDatabase scans the active database's NameBase and IndexEntries in a single fast pass
// to aggregate game counts, peak Elos, and active year spans for famous and custom players.
func DiscoverNotablePlayersInDatabase(
	db *Database,
	customEntries []FamousPlayerEntry,
	hiddenNames []string,
) []DiscoveredNotablePlayer {
	hiddenSet := make(map[string]bool, len(hiddenNames))
	for _, h := range hiddenNames {
		clean := strings.ToLower(strings.TrimSpace(h))
		if clean != "" {
			hiddenSet[clean] = true
		}
	}

	var combined []FamousPlayerEntry
	for _, defEntry := range DefaultFamousPlayersCatalog() {
		if hiddenSet[strings.ToLower(strings.TrimSpace(defEntry.Name))] {
			continue
		}
		combined = append(combined, defEntry)
	}
	for _, ce := range customEntries {
		name := strings.TrimSpace(ce.Name)
		if name == "" || hiddenSet[strings.ToLower(name)] {
			continue
		}
		ce.Name = name
		ce.IsCustom = true
		if len(ce.Aliases) == 0 {
			ce.Aliases = []string{name}
		}
		combined = append(combined, ce)
	}

	results := make([]DiscoveredNotablePlayer, len(combined))
	matchedSets := make([]map[string]struct{}, len(combined))
	for i, c := range combined {
		aliasesCopy := make([]string, len(c.Aliases))
		copy(aliasesCopy, c.Aliases)
		results[i] = DiscoveredNotablePlayer{
			Name:         c.Name,
			Title:        c.Title,
			Era:          c.Era,
			Aliases:      aliasesCopy,
			MatchedNames: []string{},
			IsCustom:     c.IsCustom,
		}
		matchedSets[i] = make(map[string]struct{})
	}

	if db != nil {
		db.mu.RLock()
		if db.IsOpen && db.Names != nil {
			// 1. Scan NamePlayer dictionary once to map player NameID -> catalog entry indices
			idToCatalogIdx := make(map[uint32][]int)
			for nameID, playerName := range db.Names.NamesByID[NamePlayer] {
				for idx, entry := range combined {
					if matchesCatalogEntryName(playerName, entry) {
						idToCatalogIdx[nameID] = append(idToCatalogIdx[nameID], idx)
					}
				}
			}

			// 2. Scan db.Entries in a single fast pass using integer NameIDs
			if len(idToCatalogIdx) > 0 {
				for _, entry := range db.Entries {
					if entry == nil || (entry.Flags&FlagDelete) != 0 {
						continue
					}
					wMatches := idToCatalogIdx[entry.WhiteID]
					bMatches := idToCatalogIdx[entry.BlackID]
					if len(wMatches) == 0 && len(bMatches) == 0 {
						continue
					}

					year := parseYear(entry.Date)
					validYear := year > 1000 && year < 3000

					for _, cIdx := range wMatches {
						p := &results[cIdx]
						p.GameCount++
						if int(entry.WhiteElo) > p.PeakElo {
							p.PeakElo = int(entry.WhiteElo)
						}
						if validYear {
							if p.YearStart == 0 || year < p.YearStart {
								p.YearStart = year
							}
							if year > p.YearEnd {
								p.YearEnd = year
							}
						}
						wName := db.Names.GetName(NamePlayer, entry.WhiteID)
						matchedSets[cIdx][wName] = struct{}{}
					}

					for _, cIdx := range bMatches {
						p := &results[cIdx]
						alreadyCounted := false
						for _, wIdx := range wMatches {
							if wIdx == cIdx {
								alreadyCounted = true
								break
							}
						}
						if !alreadyCounted {
							p.GameCount++
						}
						if int(entry.BlackElo) > p.PeakElo {
							p.PeakElo = int(entry.BlackElo)
						}
						if validYear {
							if p.YearStart == 0 || year < p.YearStart {
								p.YearStart = year
							}
							if year > p.YearEnd {
								p.YearEnd = year
							}
						}
						bName := db.Names.GetName(NamePlayer, entry.BlackID)
						matchedSets[cIdx][bName] = struct{}{}
					}
				}
			}
		}
		db.mu.RUnlock()
	}

	for i := range results {
		if len(matchedSets[i]) > 0 {
			names := make([]string, 0, len(matchedSets[i]))
			for n := range matchedSets[i] {
				names = append(names, n)
			}
			sort.Strings(names)
			results[i].MatchedNames = names
		}
	}

	sort.Slice(results, func(i, j int) bool {
		a, b := results[i], results[j]
		aProminent := a.GameCount > 0 || a.IsCustom
		bProminent := b.GameCount > 0 || b.IsCustom
		if aProminent != bProminent {
			return aProminent
		}
		if a.GameCount != b.GameCount {
			return a.GameCount > b.GameCount
		}
		if a.IsCustom != b.IsCustom {
			return a.IsCustom
		}
		return a.Name < b.Name
	})

	return results
}
