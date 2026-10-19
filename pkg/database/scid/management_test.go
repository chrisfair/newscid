package scid

import (
	"os"
	"path/filepath"
	"testing"

	"newscid/pkg/chess"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWritePoliciesAndInference(t *testing.T) {
	assert.Equal(t, "editable", WritePolicyEditable.String())
	assert.Equal(t, "sync_only", WritePolicySyncOnly.String())
	assert.Equal(t, "locked", WritePolicyLocked.String())
	assert.Equal(t, "editable", WritePolicyID(99).String())

	assert.Equal(t, WritePolicyEditable, ParseWritePolicy("editable"))
	assert.Equal(t, WritePolicyEditable, ParseWritePolicy(""))
	assert.Equal(t, WritePolicyEditable, ParseWritePolicy("unknown"))
	assert.Equal(t, WritePolicySyncOnly, ParseWritePolicy("sync_only"))
	assert.Equal(t, WritePolicySyncOnly, ParseWritePolicy("  SyncOnly "))
	assert.Equal(t, WritePolicyLocked, ParseWritePolicy("locked"))
	assert.Equal(t, WritePolicyLocked, ParseWritePolicy("readonly"))
	assert.Equal(t, WritePolicyLocked, ParseWritePolicy("READ_ONLY"))

	assert.True(t, WritePolicyEditable.AllowsDirectEdit())
	assert.False(t, WritePolicySyncOnly.AllowsDirectEdit())
	assert.False(t, WritePolicyLocked.AllowsDirectEdit())

	assert.True(t, WritePolicyEditable.AllowsOnlineSync())
	assert.True(t, WritePolicySyncOnly.AllowsOnlineSync())
	assert.False(t, WritePolicyLocked.AllowsOnlineSync())

	// Inference checks
	assert.Equal(t, WritePolicyLocked, InferDefaultWritePolicy("/usr/share/scid/bases/tactics", 500, "Tactics", false))
	assert.Equal(t, WritePolicySyncOnly, InferDefaultWritePolicy("/home/user/dbs/custom", 10, "Custom", true))
	assert.Equal(t, WritePolicyEditable, InferDefaultWritePolicy("/home/user/.config/newscid/personas/hikaru/games", 50000, "Persona DB", false))
	assert.Equal(t, WritePolicyEditable, InferDefaultWritePolicy("/home/user/Clipbase", 100, "Temporary", false))
	assert.Equal(t, WritePolicySyncOnly, InferDefaultWritePolicy("/home/user/ChessDataBases/lumbrasgigabase", 100, "", false))
	assert.Equal(t, WritePolicySyncOnly, InferDefaultWritePolicy("/home/user/ChessDataBases/my_archive", 100, "Mega Reference Collection", false))
	assert.Equal(t, WritePolicySyncOnly, InferDefaultWritePolicy("/home/user/ChessDataBases/big_collection", 25000, "Games", false))
	assert.Equal(t, WritePolicyEditable, InferDefaultWritePolicy("/home/user/ChessDataBases/my_repertoire", 450, "My Sicilian Repertoire", false))

	// InspectDatabase integration with WritePolicy and IsLocked
	tmpDir := t.TempDir()
	personalBase := filepath.Join(tmpDir, "my_study")
	db := NewDatabase()
	require.NoError(t, db.Create(personalBase, "Personal Study"))
	_ = db.Close()

	discPersonal := InspectDatabase(personalBase, "User Databases")
	assert.True(t, discPersonal.Exists)
	assert.Equal(t, "editable", discPersonal.WritePolicy)
	assert.False(t, discPersonal.IsLocked)

	megaBase := filepath.Join(tmpDir, "lumbrasgigabase_test")
	dbMega := NewDatabase()
	require.NoError(t, dbMega.Create(megaBase, "Lumbra GigaBase"))
	_ = dbMega.Close()

	discMega := InspectDatabase(megaBase, "User Databases")
	assert.True(t, discMega.Exists)
	assert.Equal(t, "sync_only", discMega.WritePolicy)
	assert.True(t, discMega.IsLocked)

	pgnPath := filepath.Join(tmpDir, "twic_archive.pgn")
	require.NoError(t, os.WriteFile(pgnPath, []byte("[Event \"Test\"]\n\n1. e4 e5 *\n"), 0o644))
	discPGN := InspectDatabase(pgnPath, "User Databases")
	assert.True(t, discPGN.Exists)
	assert.Equal(t, "sync_only", discPGN.WritePolicy)
	assert.True(t, discPGN.IsLocked)

	discSys := InspectDatabase("/usr/share/scid/bases/nonexistent_tactics", "System")
	assert.Equal(t, "locked", discSys.WritePolicy)
	assert.True(t, discSys.IsLocked)
}

func TestSubDatabaseHelpers(t *testing.T) {
	assert.Equal(t, "focused_subdb", sanitizeFileName("   !!!   "))
	assert.Equal(t, "Fischer_Najdorf_1970", sanitizeFileName("Fischer: Najdorf (1970)!"))

	moves := normalizeOpeningMoves([]string{"1. e4 c5", "2.Nf3!", "2...d6?", "3.", "a.b"})
	assert.Equal(t, []string{"e4", "c5", "Nf3", "d6", "a.b"}, moves)

	assert.True(t, matchesOpeningMoves(nil, nil))
	assert.False(t, matchesOpeningMoves(nil, []string{"e4"}))
	root := &chess.MoveNode{SAN: "e4", UCI: "e2e4", Next: &chess.MoveNode{SAN: "c5", UCI: "c7c5"}}
	assert.True(t, matchesOpeningMoves(root, []string{"e4", "c5"}))
	assert.False(t, matchesOpeningMoves(root, []string{"e4", "c5", "Nf3"}))
	assert.False(t, matchesOpeningMoves(root, []string{"d4"}))

	assert.True(t, matchesECOFilter("B90", ""))
	assert.False(t, matchesECOFilter("", "B90"))
	assert.False(t, matchesECOFilter("?", "B90"))
	assert.True(t, matchesECOFilter("B90", "B9"))
	assert.True(t, matchesECOFilter("B90a", "B20-B99"))
	assert.False(t, matchesECOFilter("C42", "B20-B99"))

	assert.False(t, playerMatchesSingle("?", "Fischer"))
	assert.False(t, playerMatchesSingle("Fischer, Bobby", ""))
	assert.True(t, playerMatchesSingle("Fischer, Robert James", "Fischer, R."))
	assert.True(t, playerMatchesSingle("Fischer, Bobby", "Bobby Fischer"))
	assert.True(t, playerMatchesSingle("Bobby Fischer", "Fischer, Bobby"))
	assert.False(t, playerMatchesSingle("Carlsen, Magnus", "Kasparov, Garry"))
	assert.True(t, matchesPlayerFilter("A", "B", nil, "any"))

	assert.False(t, matchesMetadataFilter("W", "B", "B90", "1970.01.01", "1-0", 2500, 2500, SubDatabaseFilter{YearFrom: 1980}))
	assert.False(t, matchesMetadataFilter("W", "B", "B90", "1970.01.01", "1-0", 2500, 2500, SubDatabaseFilter{YearTo: 1960}))
	assert.False(t, matchesMetadataFilter("W", "B", "B90", "1970.01.01", "1-0", 2500, 2500, SubDatabaseFilter{ECOFilter: "C42"}))

	assert.False(t, matchesCatalogEntryName("?", FamousPlayerEntry{Name: "Bobby Fischer"}))
	assert.True(t, matchesCatalogEntryName("Fischer, R", FamousPlayerEntry{Name: "Bobby Fischer", Aliases: []string{"", "Fischer, R."}}))
	assert.True(t, matchesCatalogEntryName("Carlsen, Magnus (NOR)", FamousPlayerEntry{Name: "Magnus Carlsen", Aliases: []string{"Carlsen, Magnus"}}))
	assert.True(t, matchesCatalogEntryName("Fischer, Robert James", FamousPlayerEntry{Name: "Bobby Fischer", Aliases: []string{"Fischer, Robert J"}}))
	assert.False(t, matchesCatalogEntryName("Fischer, Rolf", FamousPlayerEntry{Name: "Bobby Fischer", Aliases: []string{"Fischer, R."}}))

	assert.Equal(t, "", gameDedupSignature(nil))
	g := chess.NewGame()
	g.RootMove = &chess.MoveNode{SAN: "e4", UCI: ""}
	sig := gameDedupSignature(g)
	assert.Contains(t, sig, "e4,")
}

func TestBuildFocusedSubDatabase(t *testing.T) {
	tmpDir := t.TempDir()

	// Validation errors
	_, err := BuildFocusedSubDatabase(SubDatabaseFilter{}, tmpDir, nil)
	require.Error(t, err)

	_, err = BuildFocusedSubDatabase(SubDatabaseFilter{
		SourceBasePaths: []string{"some_path"},
	}, "", nil)
	require.Error(t, err)

	// Create Source 1 (.si4) with 4 games
	src1Base := filepath.Join(tmpDir, "source1")
	db1 := NewDatabase()
	require.NoError(t, db1.Create(src1Base, "Source 1"))

	pgn1 := `[Event "Siegen Olympiad"]
[Site "Siegen"]
[Date "1970.09.15"]
[Round "1"]
[White "Fischer, Robert James"]
[Black "Spassky, Boris V"]
[Result "1-0"]
[WhiteElo "2760"]
[BlackElo "2670"]
[ECO "B90"]

1. e4 c5 2. Nf3 d6 3. d4 cxd4 4. Nxd4 Nf6 5. Nc3 a6 1-0

[Event "Reykjavik WCh"]
[Site "Reykjavik"]
[Date "1972.07.23"]
[Round "6"]
[White "Fischer, Bobby"]
[Black "Spassky, Boris"]
[Result "1-0"]
[WhiteElo "2785"]
[BlackElo "2660"]
[ECO "D59"]

1. c4 e6 2. Nf3 d5 3. d4 Nf6 4. Nc3 Be7 1-0

[Event "Bled Candidates"]
[Site "Bled"]
[Date "1959.10.01"]
[Round "2"]
[White "Tal, Mikhail"]
[Black "Fischer, R."]
[Result "1-0"]
[WhiteElo "2700"]
[BlackElo "2580"]
[ECO "B87"]

1. e4 c5 2. Nf3 d6 3. d4 cxd4 4. Nxd4 Nf6 5. Nc3 a6 6. Bc4 1-0

[Event "Club Casual"]
[Site "Local"]
[Date "1965.05.01"]
[Round "1"]
[White "Amateur, A"]
[Black "Amateur, B"]
[Result "1/2-1/2"]
[WhiteElo "1800"]
[BlackElo "1820"]
[ECO "C50"]

1. e4 e5 2. Nf3 Nc6 3. Bc4 Bc5 1/2-1/2
`
	_, err = db1.ImportPGNText(pgn1)
	require.NoError(t, err)
	// Also add a deleted game to verify deleted games are skipped
	_, err = db1.ImportPGNText(`[Event "Deleted"]
[White "Fischer, Robert James"]
[Black "Tal, Mikhail"]
[Date "1970.01.01"]
[Result "1-0"]
[WhiteElo "2750"]
[ECO "B90"]

1. e4 c5 2. Nf3 d6 1-0
`)
	require.NoError(t, err)
	require.NoError(t, db1.DeleteGame(5, true))
	require.NoError(t, db1.Close())

	// Create Source 2 (.pgn) with a duplicate of Game 1 plus another Najdorf game
	src2PGN := filepath.Join(tmpDir, "source2.pgn")
	pgn2 := `[Event "Siegen Olympiad"]
[Site "Siegen"]
[Date "1970.09.15"]
[Round "1"]
[White "Fischer, Robert James"]
[Black "Spassky, Boris V"]
[Result "1-0"]
[WhiteElo "2760"]
[BlackElo "2670"]
[ECO "B90"]

1. e4 c5 2. Nf3 d6 3. d4 cxd4 4. Nxd4 Nf6 5. Nc3 a6 1-0

[Event "Buenos Aires"]
[Site "Buenos Aires"]
[Date "1970.07.19"]
[Round "3"]
[White "Fischer, Robert J"]
[Black "Najdorf, Miguel"]
[Result "1-0"]
[WhiteElo "2740"]
[BlackElo "2590"]
[ECO "B92"]

1. e4 c5 2. Nf3 d6 3. d4 cxd4 4. Nxd4 Nf6 5. Nc3 a6 6. Be2 e5 1-0
`
	require.NoError(t, os.WriteFile(src2PGN, []byte(pgn2), 0o644))

	// 1. Build sub-database for Bobby Fischer in Sicilian (B80-B99) with 1.e4 c5 2.Nf3 and deduplication
	var progressCalls int
	outDir := filepath.Join(tmpDir, "output")
	res, err := BuildFocusedSubDatabase(SubDatabaseFilter{
		TargetName:        "Fischer Sicilian Study",
		TargetDescription: "Fischer Sicilian Games",
		SourceBasePaths:   []string{"", src1Base, src2PGN, filepath.Join(tmpDir, "missing.si4"), filepath.Join(tmpDir, "missing.pgn")},
		Players:           []string{"Bobby Fischer", "Fischer, R."},
		PlayerColor:       "any",
		ECOFilter:         "B80-B99",
		OpeningMoves:      []string{"1.e4", "c5", "2.Nf3"},
		MinElo:            2600,
		YearFrom:          1955,
		YearTo:            1975,
		ResultFilter:      "1-0",
		Deduplicate:       true,
	}, outDir, func(source string, scanned, matched int) {
		progressCalls++
	})
	require.NoError(t, err)
	assert.Equal(t, 2, res.SourcesScanned)
	assert.Equal(t, 6, res.TotalGamesScanned)
	assert.Equal(t, 3, res.GamesMatched) // Game 1 (Siegen), Game 3 (Bled), Game 2 of PGN (Buenos Aires)
	assert.Equal(t, 1, res.DuplicatesSkipped)
	assert.GreaterOrEqual(t, progressCalls, 2)

	// Open built sub-database and verify contents
	builtDB := NewDatabase()
	require.NoError(t, builtDB.Open(res.TargetBasePath))
	assert.Equal(t, 3, builtDB.NumGames())
	require.NoError(t, builtDB.Close())

	// 2. Test PlayerColor = "black", UCI opening moves, explicit TargetBasePath, and MaxGames = 1
	explicitTarget := filepath.Join(tmpDir, "explicit_target.si4")
	res2, err := BuildFocusedSubDatabase(SubDatabaseFilter{
		TargetBasePath:  explicitTarget,
		SourceBasePaths: []string{src1Base, src2PGN},
		Players:         []string{"Fischer"},
		PlayerColor:     "black",
		OpeningMoves:    []string{"e2e4", "c7c5"},
		MaxGames:        1,
	}, "", nil)
	require.NoError(t, err)
	assert.Equal(t, 1, res2.GamesMatched)
	assert.Equal(t, "explicit_target", res2.TargetName)

	// 3. Test PlayerColor = "white" and MaxGames reached across PGN
	res3, err := BuildFocusedSubDatabase(SubDatabaseFilter{
		SourceBasePaths: []string{src2PGN, src1Base},
		Players:         []string{"Fischer"},
		PlayerColor:     "white",
		Deduplicate:     true,
		MaxGames:        1,
	}, outDir, nil)
	require.NoError(t, err)
	assert.Equal(t, 1, res3.GamesMatched)
}

func TestDiscoverNotablePlayersInDatabase(t *testing.T) {
	catalog := DefaultFamousPlayersCatalog()
	assert.GreaterOrEqual(t, len(catalog), 24)

	// Verify nil/closed database returns catalog + custom entries sorted with custom first
	custom := []FamousPlayerEntry{
		{
			Name:  "My Coach",
			Title: "IM & Trainer",
			Era:   "2010-present",
		},
	}
	notableNil := DiscoverNotablePlayersInDatabase(nil, custom, []string{"Paul Morphy"})
	require.NotEmpty(t, notableNil)
	assert.Equal(t, "My Coach", notableNil[0].Name)
	assert.True(t, notableNil[0].IsCustom)
	for _, p := range notableNil {
		assert.NotEqual(t, "Paul Morphy", p.Name)
	}

	// Create open database with games from famous players and custom player
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "notable_test")
	db := NewDatabase()
	require.NoError(t, db.Create(dbPath, "Notable Players Test"))
	defer db.Close()

	pgn := `[Event "Game 1"]
[Date "1972.07.23"]
[White "Fischer, Robert James"]
[Black "Spassky, Boris V"]
[Result "1-0"]
[WhiteElo "2785"]
[BlackElo "2660"]

1. c4 e6 1-0

[Event "Game 2"]
[Date "1959.10.01"]
[White "Tal, Mikhail"]
[Black "Fischer, R."]
[Result "1-0"]
[WhiteElo "2705"]
[BlackElo "2585"]

1. e4 c5 1-0

[Event "Game 3"]
[Date "2020.05.10"]
[White "DrNykterstein"]
[Black "Hikaru"]
[Result "1/2-1/2"]
[WhiteElo "2882"]
[BlackElo "2829"]

1. d4 Nf6 1/2-1/2

[Event "Game 4"]
[Date "2023.01.15"]
[White "Coach, My"]
[Black "Coach, My"]
[Result "1/2-1/2"]
[WhiteElo "2410"]
[BlackElo "2450"]

1. e4 e5 1/2-1/2
`
	_, err := db.ImportPGNText(pgn)
	require.NoError(t, err)

	discovered := DiscoverNotablePlayersInDatabase(db, custom, []string{"Emanuel Lasker"})
	require.NotEmpty(t, discovered)

	// Top entry should be Bobby Fischer (2 games)
	assert.Equal(t, "Bobby Fischer", discovered[0].Name)
	assert.Equal(t, 2, discovered[0].GameCount)
	assert.Equal(t, 2785, discovered[0].PeakElo)
	assert.Equal(t, 1959, discovered[0].YearStart)
	assert.Equal(t, 1972, discovered[0].YearEnd)
	assert.Equal(t, []string{"Fischer, R.", "Fischer, Robert James"}, discovered[0].MatchedNames)

	// Verify Magnus Carlsen matched "DrNykterstein"
	var foundCarlsen, foundCoach bool
	for _, p := range discovered {
		if p.Name == "Magnus Carlsen" {
			foundCarlsen = true
			assert.Equal(t, 1, p.GameCount)
			assert.Equal(t, 2882, p.PeakElo)
			assert.Equal(t, []string{"DrNykterstein"}, p.MatchedNames)
		}
		if p.Name == "My Coach" {
			foundCoach = true
			assert.Equal(t, 1, p.GameCount)
			assert.Equal(t, 2450, p.PeakElo)
			assert.Equal(t, 2023, p.YearStart)
			assert.Equal(t, 2023, p.YearEnd)
		}
	}
	assert.True(t, foundCarlsen)
	assert.True(t, foundCoach)
}

func TestExtractOnlineGameIDAndDeduplicator(t *testing.T) {
	assert.Equal(t, "", ExtractOnlineGameID(nil))

	gLichess := &chess.Game{
		Tags: map[string]string{
			"Site": "https://lichess.org/AbCd1234/white#25",
		},
	}
	assert.Equal(t, "lichess:abcd1234", ExtractOnlineGameID(gLichess))

	gLichess12 := &chess.Game{
		Tags: map[string]string{
			"Link": "https://lichess.org/game/export/ZyXw9876aaaa",
		},
	}
	assert.Equal(t, "lichess:zyxw9876", ExtractOnlineGameID(gLichess12))

	gLichessBroadcast := &chess.Game{
		Tags: map[string]string{
			"Site": "https://lichess.org/broadcast/tata-steel/round-1/BcDe5678",
		},
	}
	assert.Equal(t, "lichess:bcde5678", ExtractOnlineGameID(gLichessBroadcast))

	gChessCom := &chess.Game{
		Tags: map[string]string{
			"Site": "Chess.com",
			"Link": "https://www.chess.com/game/live/1092837465?username=test",
		},
	}
	assert.Equal(t, "chesscom:1092837465", ExtractOnlineGameID(gChessCom))

	gExplicitID := &chess.Game{
		Tags: map[string]string{
			"GameId": "FIDE-2026-0091",
		},
	}
	assert.Equal(t, "id:fide-2026-0091", ExtractOnlineGameID(gExplicitID))

	gNonAlpha := &chess.Game{
		Tags: map[string]string{
			"Site": "https://lichess.org/ab$def12",
			"Link": "https://chess.com/game/live/12$456",
		},
	}
	assert.Equal(t, "", ExtractOnlineGameID(gNonAlpha))

	var nilDedup *GameDeduplicator
	dup, byID := nilDedup.CheckAndAdd(gExplicitID)
	assert.False(t, dup)
	assert.False(t, byID)
	nilDedup.SeedFromDatabase(nil)

	dedup := NewGameDeduplicator()
	shortGame := &chess.Game{
		Tags: map[string]string{
			"White":  "Fischer, Bobby",
			"Black":  "Tal, Mikhail",
			"Date":   "1960.01.01",
			"Result": "1-0",
		},
		RootMove: &chess.MoveNode{UCI: "e2e4"},
	}
	dup1, _ := dedup.CheckAndAdd(shortGame)
	assert.False(t, dup1)
	dup2, byID2 := dedup.CheckAndAdd(shortGame)
	assert.True(t, dup2)
	assert.False(t, byID2)

	// Test canonicalPlayerKey
	assert.Equal(t, canonicalPlayerKey("GM Fischer, Robert James"), canonicalPlayerKey("Robert James Fischer"))
	assert.Equal(t, "?", canonicalPlayerKey("?"))
}

func TestConsolidateDatabases(t *testing.T) {
	tmpDir := t.TempDir()

	// Create Source DB 1 (.si4) with 3 games
	src1Path := filepath.Join(tmpDir, "online_lichess")
	src1 := NewDatabase()
	require.NoError(t, src1.Create(src1Path, "Lichess Games"))
	pgn1 := `
[Event "Rated Blitz"]
[Site "https://lichess.org/AbCdEf12"]
[Date "2026.02.10"]
[White "Carlsen, Magnus"]
[Black "Nakamura, Hikaru"]
[Result "1-0"]
[WhiteElo "2880"]
[BlackElo "2870"]

1. e4 e5 2. Nf3 Nc6 3. Bb5 a6 4. Ba4 Nf6 5. O-O Be7 1-0

[Event "World Championship"]
[Site "Reykjavik"]
[Date "1972.07.23"]
[White "Fischer, Robert James"]
[Black "Spassky, Boris V"]
[Result "1-0"]
[WhiteElo "2785"]
[BlackElo "2660"]

1. c4 e6 2. Nf3 d5 3. d4 Nf6 4. Nc3 Be7 5. Bg5 O-O 1-0
`
	_, err := src1.ImportPGNText(pgn1)
	require.NoError(t, err)
	require.NoError(t, src1.Close())

	// Create Source 2 (.pgn) containing:
	// - Game 1: Duplicate by Lichess ID (AbCdEf12) even with different White/Black handles
	// - Game 2: Duplicate by identical 10-ply move sequence & canonical players ("Robert James Fischer" vs "Boris V Spassky")
	// - Game 3: Unique new game on Chess.com
	src2PGNPath := filepath.Join(tmpDir, "online_chesscom.pgn")
	pgn2 := `
[Event "Titled Arena"]
[Site "https://lichess.org/AbCdEf12"]
[Date "2026.02.10"]
[White "DrNykterstein"]
[Black "Hikaru"]
[Result "1-0"]
[WhiteElo "2880"]
[BlackElo "2870"]

1. e4 e5 2. Nf3 Nc6 3. Bb5 a6 1-0

[Event "WCh 1972"]
[Site "Iceland"]
[Date "1972.07.23"]
[White "Robert James Fischer"]
[Black "Boris V Spassky"]
[Result "1-0"]
[WhiteElo "2785"]
[BlackElo "2660"]

1. c4 e6 2. Nf3 d5 3. d4 Nf6 4. Nc3 Be7 5. Bg5 O-O 1-0

[Event "Titled Tuesday"]
[Site "Chess.com"]
[Link "https://www.chess.com/game/live/9988776655"]
[Date "2026.03.01"]
[White "Caruana, Fabiano"]
[Black "Firouzja, Alireza"]
[Result "1/2-1/2"]
[WhiteElo "2805"]
[BlackElo "2790"]

1. d4 Nf6 2. c4 g6 3. Nc3 Bg7 4. e4 d6 5. Nf3 O-O 1/2-1/2
`
	require.NoError(t, os.WriteFile(src2PGNPath, []byte(pgn2), 0o644))

	var progressCalls int
	res, err := ConsolidateDatabases(ConsolidateDatabaseOptions{
		TargetName:      "Consolidated_Online_Master",
		SourceBasePaths: []string{src1Path, src2PGNPath},
		MinElo:          2500,
	}, tmpDir, func(source string, scanned, saved, dupID, dupContent int) {
		progressCalls++
	})
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Equal(t, 2, res.SourcesScanned)
	assert.Equal(t, 5, res.TotalGamesScanned)
	assert.Equal(t, 3, res.UniqueGamesSaved)
	assert.Equal(t, 2, res.DuplicatesRemoved)
	assert.Equal(t, 1, res.DuplicatesByID)
	assert.Equal(t, 1, res.DuplicatesByContent)
	assert.GreaterOrEqual(t, progressCalls, 2)

	// Re-running consolidation into the same target database should skip all 3 existing games!
	res2, err := ConsolidateDatabases(ConsolidateDatabaseOptions{
		TargetBasePath:  res.TargetBasePath,
		SourceBasePaths: []string{"", src1Path, src2PGNPath, res.TargetBasePath},
		MinElo:          2900, // Also test MinElo filtering on .si4 and .pgn
	}, tmpDir, nil)
	require.NoError(t, err)
	assert.Equal(t, 0, res2.UniqueGamesSaved)

	// Test custom StartFEN & short header-only game in CanonicalGameIdentity
	idKey, movesKey, hdrKey := CanonicalGameIdentity(nil)
	assert.Empty(t, idKey)
	assert.Empty(t, movesKey)
	assert.Empty(t, hdrKey)

	customFenGame := &chess.Game{
		StartFEN: "rnbqkbnr/pppp1ppp/8/4p3/4P3/8/PPPP1PPP/RNBQKBNR w KQkq - 0 2",
		Tags: map[string]string{
			"White":  "GM",
			"Black":  "?",
			"Date":   "2026.01.01",
			"Result": "1-0",
		},
		RootMove: &chess.MoveNode{SAN: "Nf3"},
	}
	_, _, hdrKey2 := CanonicalGameIdentity(customFenGame)
	assert.NotEmpty(t, hdrKey2)

	// Mark game 1 deleted in target DB and re-run with default TargetName into a temp dir
	reopenDB := NewDatabase()
	require.NoError(t, reopenDB.Open(res.TargetBasePath))
	require.NoError(t, reopenDB.DeleteGame(1, true))
	require.NoError(t, reopenDB.Close())

	res3, err := ConsolidateDatabases(ConsolidateDatabaseOptions{
		TargetBasePath:  res.TargetBasePath,
		SourceBasePaths: []string{src1Path},
	}, tmpDir, nil)
	require.NoError(t, err)
	assert.Equal(t, 1, res3.UniqueGamesSaved)

	// Verify error when no source paths provided or missing source file
	_, err = ConsolidateDatabases(ConsolidateDatabaseOptions{}, tmpDir, nil)
	assert.Error(t, err)
	_, err = ConsolidateDatabases(ConsolidateDatabaseOptions{
		SourceBasePaths: []string{filepath.Join(tmpDir, "missing.pgn")},
	}, "", nil)
	assert.Error(t, err)
	_, err = ConsolidateDatabases(ConsolidateDatabaseOptions{
		SourceBasePaths: []string{filepath.Join(tmpDir, "missing_scid")},
	}, tmpDir, nil)
	assert.Error(t, err)
}

func TestArchiveFilteringAndCancellation(t *testing.T) {
	assert.False(t, IsLegacyArchivePath(""))
	assert.False(t, IsLegacyArchivePath("/data/lumbras_otb_2026"))
	assert.False(t, IsLegacyArchivePath("/data/lumbras_otb_complete"))
	assert.False(t, IsLegacyArchivePath("/data/lichess_elite_2025_11"))
	assert.True(t, IsLegacyArchivePath("/data/lumbras_otb_2025"))
	assert.True(t, IsLegacyArchivePath("/data/lumbras_otb_2020-2024"))

	assert.Equal(t, []string{"/data/only_one_2025"}, FilterMostRecentDatabasePaths([]string{"/data/only_one_2025"}))
	assert.Equal(t, []string{"/data/a_2025", "/data/b_2024"}, FilterMostRecentDatabasePaths([]string{"/data/a_2025", "/data/b_2024"}))
	assert.Equal(t, []string{"/data/lumbras_otb_2026"}, FilterMostRecentDatabasePaths([]string{"/data/lumbras_otb_2026", "/data/lumbras_otb_2025"}))

	tmpDir := t.TempDir()
	db2026 := filepath.Join(tmpDir, "archive_2026")
	db2025 := filepath.Join(tmpDir, "archive_2025")
	pgnPath := filepath.Join(tmpDir, "games_2026.pgn")

	for _, p := range []string{db2026, db2025} {
		db := NewDatabase()
		require.NoError(t, db.Create(p, "Test"))
		g, err := chess.ParsePGN(`[White "A"] [Black "B"] [Date "2026.01.01"] [Result "1-0"] 1. e4 e5 2. Nf3 Nc6 3. Bb5 a6 4. Ba4 Nf6 1-0`)
		require.NoError(t, err)
		_, err = db.SaveGame(g)
		require.NoError(t, err)
		require.NoError(t, db.Close())
	}
	require.NoError(t, os.WriteFile(pgnPath, []byte("[White \"C\"]\n[Black \"D\"]\n[Date \"2026.02.01\"]\n[Result \"0-1\"]\n\n1. d4 d5 2. c4 e6 3. Nc3 Nf6 4. cxd5 exd5 0-1\n"), 0o644))

	// Consolidate with OnlyMostRecentArchives: true -> should skip db2025!
	resLatest, err := ConsolidateDatabases(ConsolidateDatabaseOptions{
		TargetName:             "Latest_Only",
		SourceBasePaths:        []string{db2026, db2025},
		OnlyMostRecentArchives: true,
	}, tmpDir, nil)
	require.NoError(t, err)
	assert.Equal(t, 1, resLatest.SourcesScanned)

	// Consolidate with immediate cancellation
	resCancel1, err := ConsolidateDatabases(ConsolidateDatabaseOptions{
		TargetName:      "Cancelled_Immediate",
		SourceBasePaths: []string{db2026},
		ShouldCancel:    func() bool { return true },
	}, tmpDir, nil)
	require.NoError(t, err)
	assert.True(t, resCancel1.Cancelled)
	assert.Equal(t, 0, resCancel1.SourcesScanned)

	// Consolidate with mid-stream cancellation inside PGN and Scid loops
	calls := 0
	resCancelMid, err := ConsolidateDatabases(ConsolidateDatabaseOptions{
		TargetName:      "Cancelled_Mid",
		SourceBasePaths: []string{pgnPath, db2026},
		ShouldCancel: func() bool {
			calls++
			return calls == 2 || calls == 4
		},
	}, tmpDir, nil)
	require.NoError(t, err)
	assert.True(t, resCancelMid.Cancelled)

	// BuildFocusedSubDatabase with immediate and mid-loop cancellation
	subCancel1, err := BuildFocusedSubDatabase(SubDatabaseFilter{
		TargetName:      "Sub_Cancel_1",
		SourceBasePaths: []string{db2026},
		ShouldCancel:    func() bool { return true },
	}, tmpDir, nil)
	require.NoError(t, err)
	assert.True(t, subCancel1.Cancelled)

	subCalls := 0
	subCancel2, err := BuildFocusedSubDatabase(SubDatabaseFilter{
		TargetName:      "Sub_Cancel_2",
		SourceBasePaths: []string{pgnPath, db2026},
		ShouldCancel: func() bool {
			subCalls++
			return subCalls == 2 || subCalls == 4
		},
	}, tmpDir, nil)
	require.NoError(t, err)
	assert.True(t, subCancel2.Cancelled)
}


