package persona

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"newscid/pkg/chess"
	"newscid/pkg/database/scid"
)

func TestPersonaManagerCRUD(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test_personas.json")

	// Initialize PersonaManager
	pm, err := NewPersonaManager(configPath)
	require.NoError(t, err)
	assert.Empty(t, pm.GetPersonas())

	// SavePersona validation: empty display name should fail
	_, err = pm.SavePersona(&Persona{DisplayName: "  "})
	assert.Error(t, err)
	_, err = pm.SavePersona(nil)
	assert.Error(t, err)

	// Create Persona 1
	p1, err := pm.SavePersona(&Persona{
		DisplayName: "Christopher Fair",
		Identifiers: []string{"Fair, Christopher", "ChristopherFair", "QuenyenHighElf", "Fair, Christopher"}, // duplicate should be cleaned
		Title:       "Club Master",
		Bio:         "Primary playing persona across OTB and online",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, p1.ID)
	assert.Equal(t, "Christopher Fair", p1.DisplayName)
	assert.Len(t, p1.Identifiers, 3) // deduplicated
	assert.Equal(t, []string{"Fair, Christopher", "ChristopherFair", "QuenyenHighElf"}, p1.Identifiers)
	assert.NotEmpty(t, p1.AvatarColor)
	assert.Greater(t, p1.CreatedAt, int64(0))

	// Create Persona 2
	p2, err := pm.SavePersona(&Persona{
		DisplayName: "Magnus Carlsen",
		Identifiers: []string{"Carlsen, Magnus", "MagnusCarlsen", "DrNykterstein"},
		Title:       "GM",
		AvatarColor: "#6366f1",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, p2.ID)

	// GetPersonas should return 2 sorted alphabetically
	personas := pm.GetPersonas()
	require.Len(t, personas, 2)
	assert.Equal(t, "Christopher Fair", personas[0].DisplayName)
	assert.Equal(t, "Magnus Carlsen", personas[1].DisplayName)

	// GetPersona by ID
	fetched, err := pm.GetPersona(p1.ID)
	require.NoError(t, err)
	assert.Equal(t, p1.DisplayName, fetched.DisplayName)

	_, err = pm.GetPersona("non-existent-id")
	assert.Error(t, err)

	// GetPersonaByDisplayName
	byName, err := pm.GetPersonaByDisplayName("christopher fair")
	require.NoError(t, err)
	assert.Equal(t, p1.ID, byName.ID)

	_, err = pm.GetPersonaByDisplayName("Unknown Player")
	assert.Error(t, err)

	// FindPersonaForIdentifier
	found1 := pm.FindPersonaForIdentifier("quenyenhighelf")
	require.NotNil(t, found1)
	assert.Equal(t, p1.ID, found1.ID)

	found2 := pm.FindPersonaForIdentifier("DrNykterstein")
	require.NotNil(t, found2)
	assert.Equal(t, p2.ID, found2.ID)

	assert.Nil(t, pm.FindPersonaForIdentifier("UnknownAlias"))
	assert.Nil(t, pm.FindPersonaForIdentifier(""))

	// AddIdentifier
	updated, err := pm.AddIdentifier(p1.ID, "Fair, C.")
	require.NoError(t, err)
	assert.Contains(t, updated.Identifiers, "Fair, C.")

	// Adding duplicate identifier should be a no-op and not error
	sameAgain, err := pm.AddIdentifier(p1.ID, "fair, c.")
	require.NoError(t, err)
	assert.Len(t, sameAgain.Identifiers, len(updated.Identifiers))

	_, err = pm.AddIdentifier(p1.ID, "  ")
	assert.Error(t, err)
	_, err = pm.AddIdentifier("bad-id", "some_alias")
	assert.Error(t, err)

	// RemoveIdentifier
	removed, err := pm.RemoveIdentifier(p1.ID, "Fair, C.")
	require.NoError(t, err)
	assert.NotContains(t, removed.Identifiers, "Fair, C.")

	_, err = pm.RemoveIdentifier(p1.ID, "NonExistentAlias")
	assert.Error(t, err)
	_, err = pm.RemoveIdentifier(p1.ID, "")
	assert.Error(t, err)
	_, err = pm.RemoveIdentifier("bad-id", "some_alias")
	assert.Error(t, err)

	// Persistence verification: re-open manager from disk
	pm2, err := NewPersonaManager(configPath)
	require.NoError(t, err)
	assert.Len(t, pm2.GetPersonas(), 2)
	persistedP1, err := pm2.GetPersona(p1.ID)
	require.NoError(t, err)
	assert.Equal(t, "Christopher Fair", persistedP1.DisplayName)

	// DeletePersona
	err = pm.DeletePersona(p2.ID)
	require.NoError(t, err)
	assert.Len(t, pm.GetPersonas(), 1)

	err = pm.DeletePersona("bad-id")
	assert.Error(t, err)

	// DefaultConfigPath should not be empty
	assert.NotEmpty(t, DefaultConfigPath())
}

func TestPersonaStatisticsAndSuggestions(t *testing.T) {
	// Create a mock SCID database in a temporary directory
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "personas_test_db")

	db := &scid.Database{}
	err := db.Create(dbPath, "Persona Testing Base")
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	// Add games with multiple aliases for a single person:
	// "Fair, Christopher" and "ChristopherFair"
	game1 := &chess.Game{
		Tags: map[string]string{
			"Event": "City Open",
			"Site":  "Dallas",
			"Date":  "2023.04.10",
			"White": "Fair, Christopher",
			"Black": "Kasparov, Garry",
			"Result": "1-0",
			"WhiteElo": "2350",
			"BlackElo": "2812",
			"ECO": "B90",
		},
		RootMove: &chess.MoveNode{
			UCI: "e2e4",
			SAN: "e4",
			FEN: "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1",
		},
	}
	_, err = db.SaveGame(game1)
	require.NoError(t, err)

	game2 := &chess.Game{
		Tags: map[string]string{
			"Event": "Chess.com Blitz",
			"Site":  "chess.com",
			"Date":  "2023.08.20",
			"White": "Carlsen, Magnus",
			"Black": "ChristopherFair",
			"Result": "1/2-1/2",
			"WhiteElo": "2850",
			"BlackElo": "2400",
			"ECO": "C50",
		},
		RootMove: &chess.MoveNode{
			UCI: "e2e4",
			SAN: "e4",
			FEN: "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1",
		},
	}
	_, err = db.SaveGame(game2)
	require.NoError(t, err)

	game3 := &chess.Game{
		Tags: map[string]string{
			"Event": "Lichess Rapid",
			"Site":  "lichess.org",
			"Date":  "2024.01.15",
			"White": "QuenyenHighElf",
			"Black": "RandomPlayer",
			"Result": "1-0",
			"WhiteElo": "2450",
			"BlackElo": "2100",
			"ECO": "B90",
		},
		RootMove: &chess.MoveNode{
			UCI: "e2e4",
			SAN: "e4",
			FEN: "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1",
		},
	}
	_, err = db.SaveGame(game3)
	require.NoError(t, err)

	// Create Persona
	persona := &Persona{
		ID:          "p-chris",
		DisplayName: "Christopher Fair",
		Identifiers: []string{"Fair, Christopher", "ChristopherFair", "QuenyenHighElf"},
	}

	// CalculatePersonaStatistics
	stats := CalculatePersonaStatistics(db, persona)
	require.NotNil(t, stats)
	assert.Equal(t, "p-chris", stats.PersonaID)
	assert.Equal(t, 3, stats.TotalGames)
	assert.Equal(t, 2, stats.WhiteGames)
	assert.Equal(t, 1, stats.BlackGames)
	assert.Equal(t, 2, stats.Wins)
	assert.Equal(t, 1, stats.Draws)
	assert.Equal(t, 0, stats.Losses)
	assert.InDelta(t, 66.66, stats.WinRate, 0.5)
	assert.Equal(t, 2450, stats.PeakElo)
	assert.Equal(t, 2400, stats.AverageElo) // (2350 + 2400 + 2450) / 3 = 2400
	assert.Equal(t, "2023.04.10", stats.EarliestDate)
	assert.Equal(t, "2024.01.15", stats.LatestDate)
	assert.Contains(t, stats.TopWhiteECOs, "B90")
	assert.Contains(t, stats.TopBlackECOs, "C50")

	// Verify per-identifier breakdown
	require.Len(t, stats.IdentifierStats, 3)
	statMap := make(map[string]IdentifierStat)
	for _, s := range stats.IdentifierStats {
		statMap[s.Identifier] = s
	}
	assert.Equal(t, 1, statMap["Fair, Christopher"].GameCount)
	assert.Equal(t, 1, statMap["Fair, Christopher"].Wins)
	assert.Equal(t, 1, statMap["ChristopherFair"].GameCount)
	assert.Equal(t, 1, statMap["ChristopherFair"].Draws)
	assert.Equal(t, 1, statMap["QuenyenHighElf"].GameCount)
	assert.Equal(t, 1, statMap["QuenyenHighElf"].Wins)

	// SuggestIdentifiers
	// Query "Fair" should suggest "Fair, Christopher" and "ChristopherFair"
	suggestions := SuggestIdentifiers(db, "Fair", []string{})
	assert.Contains(t, suggestions, "Fair, Christopher")
	assert.Contains(t, suggestions, "ChristopherFair")

	// If already in existingIdentifiers, should be excluded
	suggestionsFiltered := SuggestIdentifiers(db, "Fair", []string{"Fair, Christopher"})
	assert.NotContains(t, suggestionsFiltered, "Fair, Christopher")
	assert.Contains(t, suggestionsFiltered, "ChristopherFair")

	// Empty query suggests all valid players
	allSuggestions := SuggestIdentifiers(db, "", []string{})
	assert.GreaterOrEqual(t, len(allSuggestions), 3)

	// Tokenized query ("Last, First" where second token matches)
	tokenSuggestions := SuggestIdentifiers(db, "Nonmatching, Kasparov", []string{})
	assert.Contains(t, tokenSuggestions, "Kasparov, Garry")

	// Add losses for White and Black to test all result branches in CalculatePersonaStatistics
	game4 := &chess.Game{
		Tags: map[string]string{
			"White":  "Fair, Christopher",
			"Black":  "EngineBot",
			"Result": "0-1",
			"Date":   "2021.01.01",
			"ECO":    "A00",
		},
	}
	_, _ = db.SaveGame(game4)
	game5 := &chess.Game{
		Tags: map[string]string{
			"White":  "EngineBot",
			"Black":  "Fair, Christopher",
			"Result": "1-0",
			"Date":   "2025.01.01",
			"ECO":    "D00",
		},
	}
	_, _ = db.SaveGame(game5)
	game6 := &chess.Game{
		Tags: map[string]string{
			"White":  "Fair, Christopher",
			"Black":  "EngineBot",
			"Result": "1/2-1/2",
			"Date":   "????.??.??",
		},
	}
	_, _ = db.SaveGame(game6)
	statsWithLosses := CalculatePersonaStatistics(db, persona)
	assert.Equal(t, 2, statsWithLosses.Losses)
	assert.Equal(t, 2, statsWithLosses.Draws)
	assert.Equal(t, "2021.01.01", statsWithLosses.EarliestDate)
	assert.Equal(t, "2025.01.01", statsWithLosses.LatestDate)

	// Nil database or nil persona handling
	nilStats := CalculatePersonaStatistics(nil, persona)
	assert.Empty(t, nilStats.DisplayName)
	nilStats2 := CalculatePersonaStatistics(db, nil)
	assert.Empty(t, nilStats2.DisplayName)
	emptyPersonaStats := CalculatePersonaStatistics(db, &Persona{})
	assert.Equal(t, 0, emptyPersonaStats.TotalGames)

	nilSuggestions := SuggestIdentifiers(nil, "Fair", nil)
	assert.Empty(t, nilSuggestions)
}

func TestPersonaEdgeCasesAndCorruptConfig(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "corrupt.json")

	// Write invalid json
	err := os.WriteFile(configPath, []byte("invalid-json{"), 0644)
	require.NoError(t, err)

	_, err = NewPersonaManager(configPath)
	assert.Error(t, err)

	// CleanIdentifiers edge cases
	cleaned := CleanIdentifiers([]string{"   ", "alice", "Alice", "bob", "", "BOB"})
	assert.Equal(t, []string{"alice", "bob"}, cleaned)
}

func TestPersonaDedicatedDatabaseAndAggregation(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "personas.json")
	pm, err := NewPersonaManager(configPath)
	require.NoError(t, err)

	// Create a source database with games
	sourceBasePath := filepath.Join(tempDir, "source_db")
	sourceDB := &scid.Database{}
	err = sourceDB.Create(sourceBasePath, "Source Big Base")
	require.NoError(t, err)
	defer sourceDB.Close()

	// Add 3 games to sourceDB
	g1 := chess.NewGame()
	g1.SetTag("White", "Fair, Christopher")
	g1.SetTag("Black", "Karpov, Anatoly")
	g1.SetTag("Date", "2022.05.01")
	g1.SetTag("Result", "1-0")
	_, err = sourceDB.SaveGame(g1)
	require.NoError(t, err)

	g2 := chess.NewGame()
	g2.SetTag("White", "Kasparov, Garry")
	g2.SetTag("Black", "ChristopherFair")
	g2.SetTag("Date", "2023.06.15")
	g2.SetTag("Result", "0-1")
	_, err = sourceDB.SaveGame(g2)
	require.NoError(t, err)

	g3 := chess.NewGame()
	g3.SetTag("White", "Unrelated Player 1")
	g3.SetTag("Black", "Unrelated Player 2")
	g3.SetTag("Date", "2024.01.01")
	g3.SetTag("Result", "1/2-1/2")
	_, err = sourceDB.SaveGame(g3)
	require.NoError(t, err)

	// Create Persona with dedicated database path in tempDir
	personaBasePath := filepath.Join(tempDir, "ChristopherFair_Dedicated")
	p, err := pm.SavePersona(&Persona{
		DisplayName:  "Christopher Fair",
		Identifiers:  []string{"Fair, Christopher", "ChristopherFair"},
		DatabasePath: personaBasePath,
	})
	require.NoError(t, err)

	// EnsurePersonaDatabase creates .si4 if not present
	ensuredPath, err := pm.EnsurePersonaDatabase(p)
	require.NoError(t, err)
	assert.Equal(t, personaBasePath, ensuredPath)
	assert.FileExists(t, personaBasePath+".si4")

	// AggregateGames: extracts 2 matching games into dedicated base
	res, err := pm.AggregateGames(sourceDB, p.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, res.TotalFound)
	assert.Equal(t, 2, res.NewlyAdded)
	assert.Equal(t, 0, res.DuplicatesSkipped)
	assert.Equal(t, 2, res.TotalPersonaGames)

	// AggregateGames again: should skip both as duplicates!
	res2, err := pm.AggregateGames(sourceDB, p.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, res2.TotalFound)
	assert.Equal(t, 0, res2.NewlyAdded)
	assert.Equal(t, 2, res2.DuplicatesSkipped)
	assert.Equal(t, 2, res2.TotalPersonaGames)

	// Aggregate errors
	_, err = pm.AggregateGames(nil, p.ID)
	assert.Error(t, err)
	_, err = pm.AggregateGames(sourceDB, "non-existent-id")
	assert.Error(t, err)
	_, err = pm.EnsurePersonaDatabase(nil)
	assert.Error(t, err)

	// Test directory helpers
	dir := DefaultPersonaDirectory()
	assert.NotEmpty(t, dir)
	dbPath := DefaultPersonaDatabasePath("Garry Kasparov")
	assert.Contains(t, dbPath, "Garry_Kasparov")
	assert.Contains(t, DefaultPersonaDatabasePath(""), "persona")

	// EnsurePersonaDatabase with empty DatabasePath
	pEmptyPath, err := pm.SavePersona(&Persona{
		DisplayName: "Temporary Player",
	})
	require.NoError(t, err)
	ensuredEmpty, err := pm.EnsurePersonaDatabase(pEmptyPath)
	require.NoError(t, err)
	assert.NotEmpty(t, ensuredEmpty)

	// NewPersonaManager with empty configPath (uses default)
	pmDefault, err := NewPersonaManager("")
	require.NoError(t, err)
	assert.NotNil(t, pmDefault)
}

func TestPersonaOnlineUsernamesAutoIncluded(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "online_test_personas.json")
	pm, err := NewPersonaManager(configPath)
	require.NoError(t, err)

	// Save persona with ChessCom and Lichess usernames but empty Identifiers
	p, err := pm.SavePersona(&Persona{
		DisplayName:      "Hikaru Nakamura",
		ChessComUsername: "Hikaru",
		LichessUsername:  "HikaruNakamura",
	})
	require.NoError(t, err)
	assert.Contains(t, p.Identifiers, "Hikaru")
	assert.Contains(t, p.Identifiers, "HikaruNakamura")

	// Ensure FindPersonaForIdentifier works for both usernames
	foundChessCom := pm.FindPersonaForIdentifier("hikaru")
	require.NotNil(t, foundChessCom)
	assert.Equal(t, p.ID, foundChessCom.ID)

	foundLichess := pm.FindPersonaForIdentifier("hikarunakamura")
	require.NotNil(t, foundLichess)
	assert.Equal(t, p.ID, foundLichess.ID)

	// Ensure updating persona doesn't duplicate usernames
	p.DisplayName = "Hikaru Nakamura (GM)"
	updated, err := pm.SavePersona(p)
	require.NoError(t, err)
	assert.Len(t, updated.Identifiers, 2)

	// SavePersona with preset CreatedAt and ID, and AggregateGames with empty Identifiers + deleted source game
	pPreset, err := pm.SavePersona(&Persona{
		ID:           "preset-id",
		DisplayName:  "Karpov",
		CreatedAt:    123456,
		DatabasePath: filepath.Join(tempDir, "karpov_db"),
	})
	require.NoError(t, err)
	assert.Equal(t, int64(123456), pPreset.CreatedAt)
	// Clear identifiers on stored persona to test DisplayName fallback in AggregateGames
	pm.personas[pPreset.ID].Identifiers = nil

	srcDB := &scid.Database{}
	require.NoError(t, srcDB.Create(filepath.Join(tempDir, "src2"), "Src2"))
	defer srcDB.Close()
	gK1 := chess.NewGame()
	gK1.SetTag("White", "Karpov, Anatoly")
	gK1.SetTag("Black", "Someone")
	_, _ = srcDB.SaveGame(gK1)
	gK2 := chess.NewGame()
	gK2.SetTag("White", "Deleted")
	gK2.SetTag("Black", "Player")
	_, _ = srcDB.SaveGame(gK2)
	srcDB.Entries[1].Flags |= scid.FlagDelete

	aggRes, err := pm.AggregateGames(srcDB, pPreset.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, aggRes.NewlyAdded)

	// Substring match and deleted entry in CalculatePersonaStatistics
	statsK := CalculatePersonaStatistics(srcDB, &Persona{
		ID:          "k",
		DisplayName: "Karpov",
		Identifiers: []string{"Karpov"},
	})
	assert.Equal(t, 1, statsK.TotalGames)
}

