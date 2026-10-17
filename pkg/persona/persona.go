package persona

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"newscid/pkg/chess"
	"newscid/pkg/database/scid"
)

// Persona represents a unified user or player identity mapping together multiple
// database designators, usernames, and tournament name formats.
type Persona struct {
	ID               string   `json:"id"`
	DisplayName      string   `json:"displayName"`                // e.g. "Christopher Fair"
	Identifiers      []string `json:"identifiers"`                // e.g. ["Fair, Christopher", "ChristopherFair", "QuenyenHighElf"]
	DatabasePath     string   `json:"databasePath"`               // Dedicated .si4 database path for this persona
	Title            string   `json:"title,omitempty"`            // e.g. "GM", "IM", "FM", "NM", "Club Player"
	Bio              string   `json:"bio,omitempty"`              // Description or personal notes
	AvatarColor      string   `json:"avatarColor,omitempty"`      // CSS color or theme
	EstimatedElo     int      `json:"estimatedElo,omitempty"`     // User-specified or auto-calculated Elo
	ChessComUsername string   `json:"chessComUsername,omitempty"` // Linked Chess.com handle
	LichessUsername  string   `json:"lichessUsername,omitempty"`  // Linked Lichess handle
	FideID           string   `json:"fideId,omitempty"`           // Linked FIDE ID
	CreatedAt        int64    `json:"createdAt"`                  // Unix ms
	UpdatedAt        int64    `json:"updatedAt"`                  // Unix ms
}

// AggregateResult summarizes the result of extracting/merging games into the persona's dedicated database.
type AggregateResult struct {
	PersonaID             string `json:"personaId"`
	DisplayName           string `json:"displayName"`
	DedicatedDatabasePath string `json:"dedicatedDatabasePath"`
	TotalFound            int    `json:"totalFound"`
	NewlyAdded            int    `json:"newlyAdded"`
	DuplicatesSkipped     int    `json:"duplicatesSkipped"`
	TotalPersonaGames     int    `json:"totalPersonaGames"`
}

// IdentifierStat summarizes game presence for a single designator within a database.
type IdentifierStat struct {
	Identifier string  `json:"identifier"`
	GameCount  int     `json:"gameCount"`
	Wins       int     `json:"wins"`
	Draws      int     `json:"draws"`
	Losses     int     `json:"losses"`
	WinRate    float64 `json:"winRate"`
}

// PersonaStatistics summarizes aggregated historical performance across all designators.
type PersonaStatistics struct {
	PersonaID       string           `json:"personaId"`
	DisplayName     string           `json:"displayName"`
	TotalGames      int              `json:"totalGames"`
	WhiteGames      int              `json:"whiteGames"`
	BlackGames      int              `json:"blackGames"`
	Wins            int              `json:"wins"`
	Draws           int              `json:"draws"`
	Losses          int              `json:"losses"`
	WinRate         float64          `json:"winRate"` // 0.0 - 100.0%
	PeakElo         int              `json:"peakElo"`
	AverageElo      int              `json:"averageElo"`
	TopWhiteECOs    []string         `json:"topWhiteECOs"`
	TopBlackECOs    []string         `json:"topBlackECOs"`
	IdentifierStats []IdentifierStat `json:"identifierStats"`
	EarliestDate    string           `json:"earliestDate"`
	LatestDate      string           `json:"latestDate"`
}

// PersonaManager manages loading, saving, and querying unified personas.
type PersonaManager struct {
	mu         sync.RWMutex
	configPath string
	personas   map[string]*Persona
}

// DefaultConfigPath returns the standard user configuration path for NewScid personas.
func DefaultConfigPath() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = os.TempDir()
	}
	return filepath.Join(configDir, "newscid", "personas.json")
}

// NewPersonaManager creates a PersonaManager persisting personas at configPath.
func NewPersonaManager(configPath string) (*PersonaManager, error) {
	if configPath == "" {
		configPath = DefaultConfigPath()
	}

	pm := &PersonaManager{
		configPath: configPath,
		personas:   make(map[string]*Persona),
	}

	if err := pm.load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("loading personas: %w", err)
	}

	return pm, nil
}

func (pm *PersonaManager) load() error {
	data, err := os.ReadFile(pm.configPath)
	if err != nil {
		return err
	}

	var list []*Persona
	if err := json.Unmarshal(data, &list); err != nil {
		return err
	}

	pm.personas = make(map[string]*Persona)
	for _, p := range list {
		if p != nil && p.ID != "" {
			pm.personas[p.ID] = p
		}
	}
	return nil
}

func (pm *PersonaManager) save() error {
	dir := filepath.Dir(pm.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	list := make([]*Persona, 0, len(pm.personas))
	for _, p := range pm.personas {
		list = append(list, p)
	}
	sort.Slice(list, func(i, j int) bool {
		return strings.ToLower(list[i].DisplayName) < strings.ToLower(list[j].DisplayName)
	})

	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling personas: %w", err)
	}

	return os.WriteFile(pm.configPath, data, 0644)
}

// GetPersonas returns all configured personas sorted alphabetically by display name.
func (pm *PersonaManager) GetPersonas() []*Persona {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	result := make([]*Persona, 0, len(pm.personas))
	for _, p := range pm.personas {
		// return clone to prevent external mutation
		clone := *p
		clone.Identifiers = append([]string(nil), p.Identifiers...)
		result = append(result, &clone)
	}

	sort.Slice(result, func(i, j int) bool {
		return strings.ToLower(result[i].DisplayName) < strings.ToLower(result[j].DisplayName)
	})

	return result
}

// GetPersona retrieves a persona by its unique ID.
func (pm *PersonaManager) GetPersona(id string) (*Persona, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	p, exists := pm.personas[id]
	if !exists {
		return nil, fmt.Errorf("persona %q not found", id)
	}

	clone := *p
	clone.Identifiers = append([]string(nil), p.Identifiers...)
	return &clone, nil
}

// GetPersonaByDisplayName retrieves a persona by its exact display name (case-insensitive).
func (pm *PersonaManager) GetPersonaByDisplayName(name string) (*Persona, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	target := strings.ToLower(strings.TrimSpace(name))
	for _, p := range pm.personas {
		if strings.ToLower(p.DisplayName) == target {
			clone := *p
			clone.Identifiers = append([]string(nil), p.Identifiers...)
			return &clone, nil
		}
	}
	return nil, fmt.Errorf("persona %q not found", name)
}

// FindPersonaForIdentifier returns the persona that has designator in its identifiers, or nil.
func (pm *PersonaManager) FindPersonaForIdentifier(designator string) *Persona {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	clean := strings.ToLower(strings.TrimSpace(designator))
	if clean == "" {
		return nil
	}

	for _, p := range pm.personas {
		for _, id := range p.Identifiers {
			if strings.ToLower(strings.TrimSpace(id)) == clean {
				clone := *p
				clone.Identifiers = append([]string(nil), p.Identifiers...)
				return &clone
			}
		}
	}
	return nil
}

// SavePersona creates or updates a persona.
func (pm *PersonaManager) SavePersona(p *Persona) (*Persona, error) {
	if p == nil {
		return nil, fmt.Errorf("nil persona")
	}

	cleanDisplayName := strings.TrimSpace(p.DisplayName)
	if cleanDisplayName == "" {
		return nil, fmt.Errorf("persona display name cannot be empty")
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	now := time.Now().UnixMilli()

	id := p.ID
	if id == "" {
		id = generateID()
	}

	cleanChessCom := strings.TrimSpace(p.ChessComUsername)
	cleanLichess := strings.TrimSpace(p.LichessUsername)

	// Clean and deduplicate identifiers, ensuring online usernames are included
	ids := append([]string(nil), p.Identifiers...)
	if cleanChessCom != "" {
		ids = append(ids, cleanChessCom)
	}
	if cleanLichess != "" {
		ids = append(ids, cleanLichess)
	}
	cleanIdentifiers := CleanIdentifiers(ids)

	// Default avatar color if not provided
	avatarColor := p.AvatarColor
	if avatarColor == "" {
		avatarColor = pickAvatarColor(cleanDisplayName)
	}

	existing, exists := pm.personas[id]
	createdAt := now
	if exists && existing.CreatedAt > 0 {
		createdAt = existing.CreatedAt
	} else if p.CreatedAt > 0 {
		createdAt = p.CreatedAt
	}

	stored := &Persona{
		ID:               id,
		DisplayName:      cleanDisplayName,
		Identifiers:      cleanIdentifiers,
		DatabasePath:     strings.TrimSpace(p.DatabasePath),
		Title:            strings.TrimSpace(p.Title),
		Bio:              strings.TrimSpace(p.Bio),
		AvatarColor:      avatarColor,
		EstimatedElo:     p.EstimatedElo,
		ChessComUsername: cleanChessCom,
		LichessUsername:  cleanLichess,
		FideID:           strings.TrimSpace(p.FideID),
		CreatedAt:        createdAt,
		UpdatedAt:        now,
	}

	pm.personas[id] = stored

	if err := pm.save(); err != nil {
		return nil, err
	}

	clone := *stored
	clone.Identifiers = append([]string(nil), stored.Identifiers...)
	return &clone, nil
}

// DeletePersona removes a persona by ID.
func (pm *PersonaManager) DeletePersona(id string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if _, exists := pm.personas[id]; !exists {
		return fmt.Errorf("persona %q not found", id)
	}

	delete(pm.personas, id)
	return pm.save()
}

// DefaultPersonaDirectory returns the standard folder for persona databases.
func DefaultPersonaDirectory() string {
	homeDir, err := os.UserHomeDir()
	if err == nil {
		chessBasesDir := filepath.Join(homeDir, "ChessDataBases", "Personas")
		if _, err := os.Stat(filepath.Join(homeDir, "ChessDataBases")); err == nil {
			return chessBasesDir
		}
	}
	dataDir, err := os.UserConfigDir()
	if err != nil {
		dataDir = os.TempDir()
	}
	return filepath.Join(dataDir, "newscid", "personas")
}

// DefaultPersonaDatabasePath returns the default .si4 base path for a persona by display name.
func DefaultPersonaDatabasePath(displayName string) string {
	slug := sanitizeFileName(displayName)
	if slug == "" {
		slug = "persona"
	}
	return filepath.Join(DefaultPersonaDirectory(), slug)
}

func sanitizeFileName(name string) string {
	var sb strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			sb.WriteRune(r)
		} else if r == ' ' {
			sb.WriteRune('_')
		}
	}
	return sb.String()
}

// EnsurePersonaDatabase guarantees that a dedicated SCID database exists on disk for the persona.
func (pm *PersonaManager) EnsurePersonaDatabase(p *Persona) (string, error) {
	if p == nil {
		return "", fmt.Errorf("nil persona")
	}

	targetPath := p.DatabasePath
	if targetPath == "" {
		targetPath = DefaultPersonaDatabasePath(p.DisplayName)
	}

	dir := filepath.Dir(targetPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("creating persona database directory: %w", err)
	}

	si4Path := targetPath + ".si4"
	if _, err := os.Stat(si4Path); os.IsNotExist(err) {
		db := &scid.Database{}
		desc := fmt.Sprintf("%s Unified Games Database", p.DisplayName)
		if err := db.Create(targetPath, desc); err != nil {
			return "", fmt.Errorf("creating persona database %q: %w", targetPath, err)
		}
		_ = db.Close()
	}

	pm.mu.Lock()
	if stored, ok := pm.personas[p.ID]; ok && stored.DatabasePath != targetPath {
		stored.DatabasePath = targetPath
		stored.UpdatedAt = time.Now().UnixMilli()
		_ = pm.save()
	}
	p.DatabasePath = targetPath
	pm.mu.Unlock()

	return targetPath, nil
}

// AggregateGames extracts games matching any of the persona's identifiers from sourceDB
// and appends new/unique games into the persona's dedicated database.
func (pm *PersonaManager) AggregateGames(sourceDB *scid.Database, personaID string) (*AggregateResult, error) {
	if sourceDB == nil || !sourceDB.IsOpen {
		return nil, fmt.Errorf("source database is not open")
	}

	pm.mu.RLock()
	p, exists := pm.personas[personaID]
	pm.mu.RUnlock()
	if !exists {
		return nil, fmt.Errorf("persona %q not found", personaID)
	}

	targetBasePath, err := pm.EnsurePersonaDatabase(p)
	if err != nil {
		return nil, err
	}

	targetDB := &scid.Database{}
	if err := targetDB.Open(targetBasePath); err != nil {
		return nil, fmt.Errorf("opening target persona database: %w", err)
	}
	defer func() { _ = targetDB.Close() }()

	aliasLookup := make(map[string]bool)
	for _, id := range p.Identifiers {
		trimmed := strings.ToLower(strings.TrimSpace(id))
		if trimmed != "" {
			aliasLookup[trimmed] = true
		}
	}
	if len(aliasLookup) == 0 {
		aliasLookup[strings.ToLower(strings.TrimSpace(p.DisplayName))] = true
	}

	// Read existing signatures in target database for deduplication
	existingSigs := make(map[string]bool)
	for i := 0; i < len(targetDB.Entries); i++ {
		entry := targetDB.Entries[i]
		w := strings.ToLower(targetDB.Names.GetName(scid.NamePlayer, entry.WhiteID))
		b := strings.ToLower(targetDB.Names.GetName(scid.NamePlayer, entry.BlackID))
		sig := fmt.Sprintf("%s|%s|%s|%s|%d", entry.Date, w, b, entry.Result, entry.NumHalfMoves)
		existingSigs[sig] = true
	}

	result := &AggregateResult{
		PersonaID:             p.ID,
		DisplayName:           p.DisplayName,
		DedicatedDatabasePath: targetBasePath,
	}

	numGames := sourceDB.NumGames()
	for i := 0; i < numGames; i++ {
		sourceDB.RLock()
		if i >= len(sourceDB.Entries) {
			sourceDB.RUnlock()
			break
		}
		entry := sourceDB.Entries[i]
		if entry == nil || (entry.Flags&scid.FlagDelete) != 0 {
			sourceDB.RUnlock()
			continue
		}

		whiteName := strings.ToLower(sourceDB.Names.GetName(scid.NamePlayer, entry.WhiteID))
		blackName := strings.ToLower(sourceDB.Names.GetName(scid.NamePlayer, entry.BlackID))
		entryDate := entry.Date
		entryResult := entry.Result
		entryHalfMoves := entry.NumHalfMoves
		sourceDB.RUnlock()

		matched := false
		for alias := range aliasLookup {
			if strings.Contains(whiteName, alias) || strings.Contains(blackName, alias) {
				matched = true
				break
			}
		}

		if !matched {
			continue
		}

		result.TotalFound++

		sig := fmt.Sprintf("%s|%s|%s|%s|%d", entryDate, whiteName, blackName, entryResult, entryHalfMoves)
		if existingSigs[sig] {
			result.DuplicatesSkipped++
			continue
		}

		game, err := sourceDB.GetGame(i + 1)
		if err != nil {
			continue
		}

		if _, err := targetDB.SaveGame(game); err != nil {
			continue
		}

		existingSigs[sig] = true
		result.NewlyAdded++
	}

	result.TotalPersonaGames = targetDB.NumGames()
	return result, nil
}

// AddIdentifier appends a new designator/identifier to an existing persona.
func (pm *PersonaManager) AddIdentifier(personaID string, identifier string) (*Persona, error) {
	clean := strings.TrimSpace(identifier)
	if clean == "" {
		return nil, fmt.Errorf("identifier cannot be empty")
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	p, exists := pm.personas[personaID]
	if !exists {
		return nil, fmt.Errorf("persona %q not found", personaID)
	}

	for _, existing := range p.Identifiers {
		if strings.EqualFold(existing, clean) {
			clone := *p
			clone.Identifiers = append([]string(nil), p.Identifiers...)
			return &clone, nil // Already present
		}
	}

	p.Identifiers = append(p.Identifiers, clean)
	p.UpdatedAt = time.Now().UnixMilli()

	if err := pm.save(); err != nil {
		return nil, err
	}

	clone := *p
	clone.Identifiers = append([]string(nil), p.Identifiers...)
	return &clone, nil
}

// RemoveIdentifier deletes a designator from an existing persona.
func (pm *PersonaManager) RemoveIdentifier(personaID string, identifier string) (*Persona, error) {
	clean := strings.TrimSpace(identifier)
	if clean == "" {
		return nil, fmt.Errorf("identifier cannot be empty")
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	p, exists := pm.personas[personaID]
	if !exists {
		return nil, fmt.Errorf("persona %q not found", personaID)
	}

	newIDs := make([]string, 0, len(p.Identifiers))
	found := false
	for _, id := range p.Identifiers {
		if strings.EqualFold(id, clean) {
			found = true
			continue
		}
		newIDs = append(newIDs, id)
	}

	if !found {
		return nil, fmt.Errorf("identifier %q not found on persona %q", identifier, personaID)
	}

	p.Identifiers = newIDs
	p.UpdatedAt = time.Now().UnixMilli()

	if err := pm.save(); err != nil {
		return nil, err
	}

	clone := *p
	clone.Identifiers = append([]string(nil), p.Identifiers...)
	return &clone, nil
}

// CleanIdentifiers trims whitespace, drops empty strings, and deduplicates case-insensitively.
func CleanIdentifiers(ids []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(ids))
	for _, id := range ids {
		trimmed := strings.TrimSpace(id)
		if trimmed == "" {
			continue
		}
		lower := strings.ToLower(trimmed)
		if !seen[lower] {
			seen[lower] = true
			result = append(result, trimmed)
		}
	}
	return result
}

// SuggestIdentifiers searches the database for player names that may belong to this persona.
// It searches for exact or substring matches against query or persona name tokens, excluding existing.
func SuggestIdentifiers(db *scid.Database, query string, existingIdentifiers []string) []string {
	if db == nil || !db.IsOpen {
		return []string{}
	}
	db.RLock()
	defer db.RUnlock()

	existingSet := make(map[string]bool)
	for _, id := range existingIdentifiers {
		existingSet[strings.ToLower(strings.TrimSpace(id))] = true
	}

	cleanQuery := strings.ToLower(strings.TrimSpace(query))

	// Collect tokens if query contains spaces or commas
	tokens := make([]string, 0)
	if cleanQuery != "" {
		words := strings.FieldsFunc(cleanQuery, func(r rune) bool {
			return r == ' ' || r == ',' || r == '.'
		})
		for _, w := range words {
			if len(w) >= 2 {
				tokens = append(tokens, strings.ToLower(w))
			}
		}
	}

	results := make([]string, 0)
	seen := make(map[string]bool)

	for _, name := range db.Names.NamesByID[scid.NamePlayer] {
		if name == "" || name == "?" {
			continue
		}
		lowerName := strings.ToLower(name)
		if existingSet[lowerName] || seen[lowerName] {
			continue
		}

		matched := false
		if cleanQuery != "" {
			if strings.Contains(lowerName, cleanQuery) {
				matched = true
			} else {
				// Check if any token matches
				for _, tok := range tokens {
					if strings.Contains(lowerName, tok) {
						matched = true
						break
					}
				}
			}
		} else {
			matched = true
		}

		if matched {
			seen[lowerName] = true
			results = append(results, name)
			if len(results) >= 50 {
				break
			}
		}
	}

	sort.Strings(results)
	return results
}

// CalculatePersonaStatistics aggregates games matching any identifier in the persona from the database.
func CalculatePersonaStatistics(db *scid.Database, p *Persona) *PersonaStatistics {
	if db == nil || !db.IsOpen || p == nil {
		return &PersonaStatistics{
			PersonaID:   "",
			DisplayName: "",
		}
	}
	db.RLock()
	defer db.RUnlock()

	stats := &PersonaStatistics{
		PersonaID:       p.ID,
		DisplayName:     p.DisplayName,
		TopWhiteECOs:    []string{},
		TopBlackECOs:    []string{},
		IdentifierStats: make([]IdentifierStat, 0, len(p.Identifiers)),
	}

	if len(p.Identifiers) == 0 {
		return stats
	}

	// Prepare quick lookup structures
	type idAgg struct {
		games  int
		wins   int
		draws  int
		losses int
	}
	idMap := make(map[string]*idAgg)
	aliasLookup := make(map[string]string) // lowerName -> exact original identifier

	for _, id := range p.Identifiers {
		orig := strings.TrimSpace(id)
		idMap[orig] = &idAgg{}
		aliasLookup[strings.ToLower(orig)] = orig
	}

	whiteECOCounts := make(map[string]int)
	blackECOCounts := make(map[string]int)
	totalEloSum := 0
	eloCount := 0

	numGames := len(db.Entries)
	for i := 0; i < numGames; i++ {
		entry := db.Entries[i]
		if entry == nil || (entry.Flags&scid.FlagDelete) != 0 {
			continue
		}

		whiteName := db.Names.GetName(scid.NamePlayer, entry.WhiteID)
		blackName := db.Names.GetName(scid.NamePlayer, entry.BlackID)

		matchedWhite := matchIdentifier(whiteName, aliasLookup)
		matchedBlack := matchIdentifier(blackName, aliasLookup)

		if matchedWhite == "" && matchedBlack == "" {
			continue
		}

		stats.TotalGames++

		// Track dates
		if entry.Date != "" && len(entry.Date) >= 4 && entry.Date != "????.??.??" {
			if stats.EarliestDate == "" || entry.Date < stats.EarliestDate {
				stats.EarliestDate = entry.Date
			}
			if stats.LatestDate == "" || entry.Date > stats.LatestDate {
				stats.LatestDate = entry.Date
			}
		}

		// As White
		if matchedWhite != "" {
			stats.WhiteGames++
			agg := idMap[matchedWhite]
			agg.games++

			if entry.WhiteElo > 0 {
				totalEloSum += int(entry.WhiteElo)
				eloCount++
				if int(entry.WhiteElo) > stats.PeakElo {
					stats.PeakElo = int(entry.WhiteElo)
				}
			}

			ecoStr := scid.FormatECO(entry.EcoCode)
			if ecoStr != "" {
				whiteECOCounts[ecoStr]++
			}

			switch entry.Result {
			case chess.ResultWhiteWins:
				stats.Wins++
				agg.wins++
			case chess.ResultDraw:
				stats.Draws++
				agg.draws++
			case chess.ResultBlackWins:
				stats.Losses++
				agg.losses++
			}
		}

		// As Black
		if matchedBlack != "" {
			stats.BlackGames++
			agg := idMap[matchedBlack]
			agg.games++

			if entry.BlackElo > 0 {
				totalEloSum += int(entry.BlackElo)
				eloCount++
				if int(entry.BlackElo) > stats.PeakElo {
					stats.PeakElo = int(entry.BlackElo)
				}
			}

			ecoStr := scid.FormatECO(entry.EcoCode)
			if ecoStr != "" {
				blackECOCounts[ecoStr]++
			}

			switch entry.Result {
			case chess.ResultBlackWins:
				stats.Wins++
				agg.wins++
			case chess.ResultDraw:
				stats.Draws++
				agg.draws++
			case chess.ResultWhiteWins:
				stats.Losses++
				agg.losses++
			}
		}
	}

	if stats.TotalGames > 0 {
		stats.WinRate = float64(stats.Wins) / float64(stats.TotalGames) * 100.0
	}
	if eloCount > 0 {
		stats.AverageElo = totalEloSum / eloCount
	}

	// Sort top ECOs
	stats.TopWhiteECOs = topECOs(whiteECOCounts, 5)
	stats.TopBlackECOs = topECOs(blackECOCounts, 5)

	// Build per-identifier stats
	for _, id := range p.Identifiers {
		orig := strings.TrimSpace(id)
		agg := idMap[orig]
		wr := 0.0
		if agg.games > 0 {
			wr = float64(agg.wins) / float64(agg.games) * 100.0
		}
		stats.IdentifierStats = append(stats.IdentifierStats, IdentifierStat{
			Identifier: orig,
			GameCount:  agg.games,
			Wins:       agg.wins,
			Draws:      agg.draws,
			Losses:     agg.losses,
			WinRate:    wr,
		})
	}

	return stats
}

func matchIdentifier(playerName string, lookup map[string]string) string {
	lower := strings.ToLower(strings.TrimSpace(playerName))
	if orig, ok := lookup[lower]; ok {
		return orig
	}
	// Also check substring contains
	for k, orig := range lookup {
		if strings.Contains(lower, k) || strings.Contains(k, lower) {
			return orig
		}
	}
	return ""
}

func topECOs(counts map[string]int, limit int) []string {
	type kv struct {
		eco   string
		count int
	}
	var pairs []kv
	for k, v := range counts {
		pairs = append(pairs, kv{k, v})
	}
	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].count > pairs[j].count
	})
	res := make([]string, 0, limit)
	for i := 0; i < len(pairs) && i < limit; i++ {
		res = append(res, pairs[i].eco)
	}
	return res
}

func generateID() string {
	bytes := make([]byte, 8)
	_, _ = rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func pickAvatarColor(name string) string {
	colors := []string{
		"#10b981", // emerald
		"#6366f1", // indigo
		"#8b5cf6", // purple
		"#ec4899", // pink
		"#f59e0b", // amber
		"#06b6d4", // cyan
		"#3b82f6", // blue
		"#ef4444", // red
	}
	hash := 0
	for _, ch := range name {
		hash = (hash*31 + int(ch)) % len(colors)
	}
	return colors[hash]
}
