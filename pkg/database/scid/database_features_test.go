package scid

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"newscid/pkg/chess"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestDatabase(t *testing.T, numGames int) (*Database, string) {
	tmpDir := t.TempDir()
	basePath := filepath.Join(tmpDir, "testdb")

	db := NewDatabase()
	err := db.Create(basePath, "Test Database for Features")
	require.NoError(t, err)

	players := []string{"Carlsen, M.", "Kasparov, G.", "Fischer, R.", "Capablanca, J.", "Tal, M."}
	events := []string{"World Championship", "Candidates", "Olympiad", "Tata Steel", "Speed Chess"}

	for i := 0; i < numGames; i++ {
		g := chess.NewGame()
		g.SetTag("White", players[i%len(players)])
		g.SetTag("Black", players[(i+1)%len(players)])
		g.SetTag("Event", events[i%len(events)])
		g.SetTag("Site", "Test City")
		g.SetTag("Round", "1")
		g.SetTag("Date", "2020.01.15")

		// Play simple moves
		b := chess.NewBoard()
		m1, _ := b.ParseSAN("e4")
		b.MakeMove(m1)
		m2, _ := b.ParseSAN("e5")
		b.MakeMove(m2)
		m3, _ := b.ParseSAN("Nf3")
		b.MakeMove(m3)

		g.RootMove = &chess.MoveNode{
			Move: m1,
			SAN:  "e4",
			Next: &chess.MoveNode{
				Move: m2,
				SAN:  "e5",
				Next: &chess.MoveNode{
					Move: m3,
					SAN:  "Nf3",
				},
			},
		}
		g.PlyCount = 3
		g.Result = chess.ResultWhiteWins
		g.SetTag("Result", "1-0")

		_, err := db.SaveGame(g)
		require.NoError(t, err)
	}

	return db, basePath
}

func TestDatabaseCompact(t *testing.T) {
	db, basePath := createTestDatabase(t, 10)
	defer db.Close()

	// 1. Initial compaction when 0 deleted games
	stats, err := db.Compact()
	require.NoError(t, err)
	assert.Equal(t, 10, stats.OriginalGames)
	assert.Equal(t, 10, stats.CompactedGames)
	assert.Equal(t, 0, stats.PurgedGames)

	// 2. Mark 3 games deleted: games 2, 5, 8
	err = db.DeleteGame(2, true)
	require.NoError(t, err)
	err = db.DeleteGame(5, true)
	require.NoError(t, err)
	err = db.DeleteGame(8, true)
	require.NoError(t, err)

	// Verify soft deletion
	s2, err := db.GetGameSummary(2)
	require.NoError(t, err)
	assert.True(t, s2.Deleted)

	// 3. Compact database
	compactStats, err := db.Compact()
	require.NoError(t, err)
	assert.Equal(t, 10, compactStats.OriginalGames)
	assert.Equal(t, 7, compactStats.CompactedGames)
	assert.Equal(t, 3, compactStats.PurgedGames)
	assert.Greater(t, compactStats.BytesReclaimed, int64(0))
	assert.Equal(t, 7, db.NumGames())

	// 4. Verify re-opening compacted database
	reopened := NewDatabase()
	err = reopened.Open(basePath)
	require.NoError(t, err)
	defer reopened.Close()
	assert.Equal(t, 7, reopened.NumGames())

	// Every remaining game should be readable and not deleted
	for i := 1; i <= reopened.NumGames(); i++ {
		sum, err := reopened.GetGameSummary(i)
		require.NoError(t, err)
		assert.False(t, sum.Deleted)

		g, err := reopened.GetGame(i)
		require.NoError(t, err)
		assert.NotNil(t, g)
		assert.Equal(t, 3, g.PlyCount)
	}
}

func TestDatabaseSort(t *testing.T) {
	db, _ := createTestDatabase(t, 6)
	defer db.Close()

	// Set dates manually
	dates := []string{"2022.05.01", "2019.01.01", "2024.12.31", "2015.03.15", "2021.08.20", "2018.06.10"}
	for i, d := range dates {
		db.Entries[i].Date = d
	}

	// 1. Sort by Date Ascending
	err := db.Sort(SortCriteria{Field: SortByDate, Ascending: true})
	require.NoError(t, err)
	assert.Equal(t, "2015.03.15", db.Entries[0].Date)
	assert.Equal(t, "2024.12.31", db.Entries[5].Date)

	// 2. Sort by Date Descending
	err = db.Sort(SortCriteria{Field: SortByDate, Ascending: false})
	require.NoError(t, err)
	assert.Equal(t, "2024.12.31", db.Entries[0].Date)
	assert.Equal(t, "2015.03.15", db.Entries[5].Date)

	// 3. Sort by White player
	err = db.Sort(SortCriteria{Field: SortByWhite, Ascending: true})
	require.NoError(t, err)
	w0 := db.Names.GetName(NamePlayer, db.Entries[0].WhiteID)
	wLast := db.Names.GetName(NamePlayer, db.Entries[5].WhiteID)
	assert.LessOrEqual(t, w0, wLast)

	// 4. Test other sort fields for full coverage
	fields := []SortField{
		SortByECO, SortByBlack, SortByPlayer, SortByWhiteElo, SortByBlackElo,
		SortByMaxElo, SortByEvent, SortBySite, SortByRound, SortByResult, SortByMoves,
	}
	for _, f := range fields {
		err := db.Sort(SortCriteria{Field: f, Ascending: true})
		assert.NoError(t, err, "Sorting by field %s should succeed", f)
	}
}

func TestDatabaseDuplicates(t *testing.T) {
	db, _ := createTestDatabase(t, 4)
	defer db.Close()

	// Add an identical clone of game 1
	g1, err := db.GetGame(1)
	require.NoError(t, err)
	_, err = db.SaveGame(g1)
	require.NoError(t, err)

	assert.Equal(t, 5, db.NumGames())

	// 1. Find exact move duplicates
	dupGroups, err := db.FindDuplicates(DuplicateCriteria{ExactMoves: true})
	require.NoError(t, err)
	assert.NotEmpty(t, dupGroups)
	assert.GreaterOrEqual(t, len(dupGroups[0].GameIDs), 2)

	// 2. Find metadata duplicates
	metaGroups, err := db.FindDuplicates(DuplicateCriteria{
		MatchPlayers: true,
		MatchDate:    true,
		MatchResult:  true,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, metaGroups)

	// 3. Delete duplicates
	deleted, err := db.DeleteDuplicates(DuplicateCriteria{ExactMoves: true})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, deleted, 1)

	// Compacting should now purge the duplicates
	compactStats, err := db.Compact()
	require.NoError(t, err)
	assert.GreaterOrEqual(t, compactStats.PurgedGames, 1)
}

func TestDatabaseVerify(t *testing.T) {
	db, _ := createTestDatabase(t, 5)
	defer db.Close()

	report, err := db.Verify()
	require.NoError(t, err)
	assert.Equal(t, 5, report.TotalGames)
	assert.Equal(t, 5, report.ActiveGames)
	assert.Equal(t, 0, report.DeletedGames)
	assert.Equal(t, 0, report.CorruptedGames)
	assert.Equal(t, "OK", report.Status)
	assert.Empty(t, report.Errors)
	assert.NotEmpty(t, report.FileSizes["si4"])
	assert.NotEmpty(t, report.FileSizes["sn4"])
	assert.NotEmpty(t, report.FileSizes["sg4"])
}

func TestDatabaseStatistics(t *testing.T) {
	db, _ := createTestDatabase(t, 5)
	defer db.Close()

	stats, err := db.GetStatistics()
	require.NoError(t, err)
	assert.Equal(t, 5, stats.TotalGames)
	assert.Equal(t, 5, stats.ActiveGames)
	assert.Equal(t, 5, stats.WhiteWins)
	assert.Equal(t, 2020, stats.EarliestYear)
	assert.Equal(t, 2020, stats.LatestYear)
	assert.NotEmpty(t, stats.TopPlayers)
	assert.Greater(t, stats.TotalSizeBytes, int64(0))
}

func TestDatabaseBatchImport(t *testing.T) {
	db, _ := createTestDatabase(t, 2)
	defer db.Close()

	pgnData := `
[Event "Hastings 1895"]
[Site "Hastings ENG"]
[Date "1895.08.17"]
[Round "11"]
[White "Steinitz, William"]
[Black "Von Bardeleben, Curt"]
[Result "1-0"]

1. e4 e5 2. Nf3 Nc6 3. Bc4 Bc5 4. c3 Nf6 5. d4 exd4 1-0

[Event "Baden-Baden"]
[Site "Baden-Baden GER"]
[Date "1925.04.16"]
[Round "1"]
[White "Reti, Richard"]
[Black "Tartakower, Savielly"]
[Result "1-0"]

1. Nf3 d5 2. c4 c6 3. g3 Nf6 4. Bg2 Bf5 1-0
`

	// 1. Test ImportPGNText
	importStats, err := db.ImportPGNText(pgnData)
	require.NoError(t, err)
	assert.Equal(t, 2, importStats.TotalParsed)
	assert.Equal(t, 2, importStats.Imported)
	assert.Equal(t, 0, importStats.Errors)
	assert.Equal(t, 4, db.NumGames())

	// Verify imported game
	g3, err := db.GetGame(3)
	require.NoError(t, err)
	assert.Equal(t, "Steinitz, William", g3.Tag("White"))
	assert.Equal(t, "Von Bardeleben, Curt", g3.Tag("Black"))

	// 2. Test ImportPGNFile
	tmpFile := filepath.Join(t.TempDir(), "batch.pgn")
	err = os.WriteFile(tmpFile, []byte(pgnData), 0644)
	require.NoError(t, err)

	fileStats, err := db.ImportPGNFile(tmpFile)
	require.NoError(t, err)
	assert.Equal(t, 2, fileStats.Imported)
	assert.Equal(t, 6, db.NumGames())
}

func TestSortAllFieldsAndEdgeCases(t *testing.T) {
	// Closed database test
	closedDb := NewDatabase()
	err := closedDb.Sort(SortCriteria{Field: SortByDate, Ascending: true})
	assert.Error(t, err)

	// Single game database
	db1, _ := createTestDatabase(t, 1)
	defer db1.Close()
	err = db1.Sort(SortCriteria{Field: SortByDate, Ascending: true})
	assert.NoError(t, err)

	// Multi-game database with varied fields
	db, _ := createTestDatabase(t, 4)
	defer db.Close()

	db.Entries[0].Date = "2021.05.01"
	db.Entries[0].EcoCode = 100 // B00
	db.Entries[0].WhiteElo = 2600
	db.Entries[0].BlackElo = 2500
	db.Entries[0].NumHalfMoves = 60
	db.Entries[0].Result = chess.ResultWhiteWins

	db.Entries[1].Date = "2019.03.10"
	db.Entries[1].EcoCode = 50 // A50
	db.Entries[1].WhiteID = db.Entries[0].WhiteID // Identical WhiteID to test secondary player sort
	db.Entries[1].EventID = db.Entries[0].EventID // Identical EventID to test event cache
	db.Entries[1].SiteID = db.Entries[0].SiteID   // Identical SiteID to test site cache
	db.Entries[1].RoundID = db.Entries[0].RoundID // Identical RoundID to test round cache
	db.Entries[1].WhiteElo = 2400
	db.Entries[1].BlackElo = 2700 // BlackElo > WhiteElo to test maxElo return b
	db.Entries[1].NumHalfMoves = 20
	db.Entries[1].Result = chess.ResultBlackWins

	db.Entries[2].Date = "2023.11.20"
	db.Entries[2].EcoCode = 250 // C50
	db.Entries[2].WhiteElo = 2800
	db.Entries[2].BlackElo = 2800
	db.Entries[2].NumHalfMoves = 80
	db.Entries[2].Result = chess.ResultDraw

	db.Entries[3].Date = "2020.08.15"
	db.Entries[3].EcoCode = 400 // E00
	db.Entries[3].WhiteElo = 2300
	db.Entries[3].BlackElo = 2300
	db.Entries[3].NumHalfMoves = 40
	db.Entries[3].Result = chess.ResultUnderway

	// Sort fields to test
	fields := []SortField{
		SortByECO,
		SortByWhite,
		SortByBlack,
		SortByPlayer,
		SortByWhiteElo,
		SortByBlackElo,
		SortByMaxElo,
		SortByEvent,
		SortBySite,
		SortByRound,
		SortByResult,
		SortByMoves,
		SortByDate,
		SortField("unknownField"),
	}

	for _, f := range fields {
		err := db.Sort(SortCriteria{Field: f, Ascending: true})
		assert.NoError(t, err, "Sorting by %s ascending failed", f)
		err = db.Sort(SortCriteria{Field: f, Ascending: false})
		assert.NoError(t, err, "Sorting by %s descending failed", f)
	}
}

func TestCompactEdgeCases(t *testing.T) {
	// Closed database
	closedDb := NewDatabase()
	_, err := closedDb.Compact()
	assert.Error(t, err)

	// All games deleted
	db, basePath := createTestDatabase(t, 3)
	defer db.Close()

	for i := 1; i <= 3; i++ {
		err := db.DeleteGame(i, true)
		require.NoError(t, err)
	}

	stats, err := db.Compact()
	require.NoError(t, err)
	assert.Equal(t, 3, stats.OriginalGames)
	assert.Equal(t, 0, stats.CompactedGames)
	assert.Equal(t, 3, stats.PurgedGames)
	assert.Equal(t, 0, db.NumGames())

	// Reopen empty compacted database
	reopened := NewDatabase()
	err = reopened.Open(basePath)
	require.NoError(t, err)
	defer reopened.Close()
	assert.Equal(t, 0, reopened.NumGames())

	// Compact on database with missing .sg4
	dbMissing, missingBase := createTestDatabase(t, 2)
	defer dbMissing.Close()
	os.Remove(missingBase + ".sg4")
	_, err = dbMissing.Compact()
	assert.Error(t, err)
}

func TestDuplicatesMetadataAndEdgeCases(t *testing.T) {
	// Closed database
	closedDb := NewDatabase()
	_, err := closedDb.FindDuplicates(DuplicateCriteria{ExactMoves: true})
	assert.Error(t, err)
	_, err = closedDb.DeleteDuplicates(DuplicateCriteria{ExactMoves: true})
	assert.Error(t, err)

	// Database with metadata duplicate criteria
	db, _ := createTestDatabase(t, 4)
	defer db.Close()

	// Make game 1 and game 2 have exact same metadata
	db.Entries[1].WhiteID = db.Entries[0].WhiteID
	db.Entries[1].BlackID = db.Entries[0].BlackID
	db.Entries[1].Date = db.Entries[0].Date
	db.Entries[1].EventID = db.Entries[0].EventID
	db.Entries[1].Result = db.Entries[0].Result
	db.Entries[1].NumHalfMoves = db.Entries[0].NumHalfMoves

	crit := DuplicateCriteria{
		ExactMoves:   false,
		MatchPlayers: true,
		MatchDate:    true,
		MatchEvent:   true,
		MatchResult:  true,
	}

	dups, err := db.FindDuplicates(crit)
	require.NoError(t, err)
	assert.NotEmpty(t, dups)

	// Delete metadata duplicates
	deletedCount, err := db.DeleteDuplicates(crit)
	require.NoError(t, err)
	assert.Equal(t, 1, deletedCount)

	// Finding duplicates again should return 0 since deleted games are excluded
	dupsAfter, err := db.FindDuplicates(crit)
	require.NoError(t, err)
	assert.Empty(t, dupsAfter)
}

func TestVerifyCorruptionsAndWarnings(t *testing.T) {
	// Closed database
	closedDb := NewDatabase()
	_, err := closedDb.Verify()
	assert.Error(t, err)

	db, _ := createTestDatabase(t, 3)
	defer db.Close()

	// Soft-delete one game
	err = db.DeleteGame(2, true)
	require.NoError(t, err)

	// Introduce a version warning
	db.Header.Version = 399

	// Introduce an invalid player ID
	db.Entries[0].WhiteID = 999999

	// Introduce an offset corruption (beyond file size)
	db.Entries[2].Offset = 99999999

	report, err := db.Verify()
	require.NoError(t, err)
	assert.Equal(t, 3, report.TotalGames)
	assert.Equal(t, 2, report.ActiveGames)
	assert.Equal(t, 1, report.DeletedGames)
	assert.Greater(t, report.CorruptedGames, 0)
	assert.Equal(t, "CORRUPTED", report.Status)
	assert.NotEmpty(t, report.Errors)
	assert.NotEmpty(t, report.Warnings)
}

func TestStatisticsEmptyAndVaried(t *testing.T) {
	// Closed database
	closedDb := NewDatabase()
	_, err := closedDb.GetStatistics()
	assert.Error(t, err)

	// Empty database
	emptyDir := t.TempDir()
	emptyBase := filepath.Join(emptyDir, "empty")
	emptyDb := NewDatabase()
	err = emptyDb.Create(emptyBase, "Empty Test DB")
	require.NoError(t, err)
	defer emptyDb.Close()

	statsEmpty, err := emptyDb.GetStatistics()
	require.NoError(t, err)
	assert.Equal(t, 0, statsEmpty.TotalGames)
	assert.Equal(t, 0, statsEmpty.ActiveGames)

	// Multi-game database with varied results, years, and elos
	db, _ := createTestDatabase(t, 4)
	defer db.Close()

	db.Entries[0].Result = chess.ResultWhiteWins
	db.Entries[0].WhiteElo = 2600
	db.Entries[0].BlackElo = 2500
	db.Entries[0].Date = "1995.04.10"
	db.Entries[0].EcoCode = 100

	db.Entries[1].Result = chess.ResultBlackWins
	db.Entries[1].WhiteElo = 2400
	db.Entries[1].BlackElo = 2700
	db.Entries[1].Date = "2005.08.12"
	db.Entries[1].EcoCode = 100

	db.Entries[2].Result = chess.ResultDraw
	db.Entries[2].WhiteElo = 2200
	db.Entries[2].BlackElo = 2200
	db.Entries[2].Date = "2022.01.01"
	db.Entries[2].EcoCode = 200

	db.Entries[3].Result = chess.ResultUnderway
	db.Entries[3].WhiteElo = 0
	db.Entries[3].BlackElo = 0
	db.Entries[3].Date = "????"
	db.Entries[3].EcoCode = 300
	_ = db.DeleteGame(4, true)

	stats, err := db.GetStatistics()
	require.NoError(t, err)
	assert.Equal(t, 4, stats.TotalGames)
	assert.Equal(t, 3, stats.ActiveGames)
	assert.Equal(t, 1, stats.DeletedGames)
	assert.Equal(t, 1, stats.WhiteWins)
	assert.Equal(t, 1, stats.BlackWins)
	assert.Equal(t, 1, stats.Draws)
	assert.Equal(t, 1, stats.UnknownResults)
	assert.Equal(t, 1995, stats.EarliestYear)
	assert.Equal(t, 2022, stats.LatestYear)
	assert.Equal(t, 2700, stats.MaxElo)
	assert.Equal(t, 2200, stats.MinElo)
	assert.Greater(t, stats.AverageElo, 0)
	assert.NotEmpty(t, stats.TopOpenings)
}

func TestBatchImportEdgeCases(t *testing.T) {
	// Closed database
	closedDb := NewDatabase()
	_, err := closedDb.ImportPGNText("1. e4")
	assert.Error(t, err)
	_, err = closedDb.ImportPGNFile("nonexistent.pgn")
	assert.Error(t, err)
	_, err = closedDb.ImportGames([]*chess.Game{})
	assert.Error(t, err)

	db, _ := createTestDatabase(t, 1)
	defer db.Close()

	// Empty games
	stats, err := db.ImportGames([]*chess.Game{})
	require.NoError(t, err)
	assert.Equal(t, 0, stats.TotalParsed)

	// Non-existent file
	_, err = db.ImportPGNFile("does_not_exist_file.pgn")
	assert.Error(t, err)
}

func TestSearchCriteriaComprehensive(t *testing.T) {
	db, _ := createTestDatabase(t, 4)
	defer db.Close()

	db.Entries[0].Date = "2021.05.01"
	db.Entries[0].EcoCode = 100 // B00
	db.Entries[0].WhiteElo = 2650
	db.Entries[0].BlackElo = 2550
	db.Entries[0].NumHalfMoves = 60
	db.Entries[0].Result = chess.ResultWhiteWins
	db.Entries[0].VarCounts = 5 // Has variations

	db.Entries[1].Date = "2019.03.10"
	db.Entries[1].EcoCode = 50 // A50
	db.Entries[1].WhiteElo = 2400
	db.Entries[1].BlackElo = 2700
	db.Entries[1].NumHalfMoves = 20
	db.Entries[1].Result = chess.ResultBlackWins

	db.Entries[2].Date = "2023.11.20"
	db.Entries[2].EcoCode = 250 // C50
	db.Entries[2].WhiteElo = 2800
	db.Entries[2].BlackElo = 2800
	db.Entries[2].NumHalfMoves = 80
	db.Entries[2].Result = chess.ResultDraw

	db.Entries[3].Date = "2020.08.15"
	db.Entries[3].EcoCode = 400 // E00
	db.Entries[3].WhiteElo = 2300
	db.Entries[3].BlackElo = 2300
	db.Entries[3].NumHalfMoves = 40
	db.Entries[3].Result = chess.ResultUnderway
	_ = db.DeleteGame(4, true)

	// Search tests for criteria fields
	tests := []struct {
		name     string
		crit     SearchCriteria
		expected int
	}{
		{"All active games", SearchCriteria{IgnoreDeleted: true}, 3},
		{"Only deleted", SearchCriteria{OnlyDeleted: true}, 1},
		{"Result 1-0", SearchCriteria{Result: "1-0"}, 1},
		{"MinElo 2600", SearchCriteria{MinElo: 2600}, 3},
		{"MaxElo 2500", SearchCriteria{MaxElo: 2500}, 1},
		{"WhiteEloMin 2500", SearchCriteria{WhiteEloMin: 2500}, 2},
		{"WhiteEloMax 2500", SearchCriteria{WhiteEloMax: 2500}, 2},
		{"BlackEloMin 2600", SearchCriteria{BlackEloMin: 2600}, 2},
		{"BlackEloMax 2600", SearchCriteria{BlackEloMax: 2600}, 2},
		{"MinMoves 35", SearchCriteria{MinMoves: 35}, 1},
		{"MaxMoves 25", SearchCriteria{MaxMoves: 25}, 2},
		{"HasVariations", SearchCriteria{HasVariations: true}, 1},
		{"ECO B", SearchCriteria{ECO: "B"}, 1},
		{"EcoFrom B00", SearchCriteria{EcoFrom: "B00"}, 3},
		{"EcoTo B99", SearchCriteria{EcoTo: "B99"}, 2},
		{"YearFrom 2021", SearchCriteria{YearFrom: 2021}, 2},
		{"YearTo 2020", SearchCriteria{YearTo: 2020}, 2},
		{"DateFrom 2020.01.01", SearchCriteria{DateFrom: "2020.01.01"}, 3},
		{"DateTo 2021.01.01", SearchCriteria{DateTo: "2021.01.01"}, 2},
		{"Player filter", SearchCriteria{Player: "Kasparov"}, 2},
		{"White filter", SearchCriteria{White: "Kasparov"}, 1},
		{"Black filter", SearchCriteria{Black: "Kasparov"}, 1},
		{"Event filter", SearchCriteria{Event: "Candidates"}, 1},
		{"Site filter", SearchCriteria{Site: "Test City"}, 4},
		{"Round filter", SearchCriteria{Round: "1"}, 4},
		{"Players multi-alias matching", SearchCriteria{Players: []string{"Kasparov", "Fischer"}}, 3},
		{"Players no match", SearchCriteria{Players: []string{"UnknownGrandmaster", "GhostPlayer"}}, 0},
		{"Players empty string element", SearchCriteria{Players: []string{"", "Kasparov"}}, 2},
	}

	for _, tc := range tests {
		matches := db.CountMatches(tc.crit)
		assert.Equal(t, tc.expected, matches, "Mismatch in search test '%s'", tc.name)
	}
}

func TestNameBaseMethods(t *testing.T) {
	nb := NewNameBase()
	assert.Equal(t, "?", nb.GetName(-1, 0))
	assert.Equal(t, "?", nb.GetName(99, 0))
	assert.Equal(t, "?", nb.GetName(NamePlayer, 999))

	_, ok := nb.GetID(-1, "test")
	assert.False(t, ok)

	id1 := nb.AddName(NamePlayer, "Carlsen, Magnus")
	assert.Equal(t, uint32(0), id1)

	// Add again to test frequency increase
	id2 := nb.AddName(NamePlayer, "Carlsen, Magnus")
	assert.Equal(t, id1, id2)

	assert.Equal(t, "Carlsen, Magnus", nb.GetName(NamePlayer, id1))
	gotID, ok := nb.GetID(NamePlayer, "Carlsen, Magnus")
	assert.True(t, ok)
	assert.Equal(t, id1, gotID)

	// Add multiple names across all name types and shared prefixes
	nb.AddName(NamePlayer, "Carlsen, Henrik")
	for i := 0; i < 300; i++ {
		nb.AddName(NameEvent, "Super Tournament")
	}
	nb.AddName(NameSite, "Reykjavik ISL")
	nb.AddName(NameRound, "Finals")

	// Roundtrip via file
	tmpFile := filepath.Join(t.TempDir(), "test.sn4")
	err := nb.WriteToFile(tmpFile)
	require.NoError(t, err)

	nb2 := NewNameBase()
	err = nb2.ReadFromFile(tmpFile)
	require.NoError(t, err)
	assert.Equal(t, "Carlsen, Magnus", nb2.GetName(NamePlayer, id1))
	assert.Equal(t, "Carlsen, Henrik", nb2.GetName(NamePlayer, 1))
	assert.Equal(t, "Super Tournament", nb2.GetName(NameEvent, 0))
	assert.Equal(t, "Reykjavik ISL", nb2.GetName(NameSite, 0))
	assert.Equal(t, "Finals", nb2.GetName(NameRound, 0))
}

func TestGameEncodingDecodingVariationsAndPieces(t *testing.T) {
	// PGN with Castling, Queen moves, Rook moves, Bishop moves, Pawn promotions, Variations, NAGs
	pgnText := `[Event "Candidates Tournament"]
[Site "Madrid ESP"]
[Date "2022.06.18"]
[Round "2"]
[White "Nakamura, Hikaru"]
[Black "Radjabov, Teimour"]
[Result "1-0"]
[WhiteElo "2760"]
[BlackElo "2753"]
[ECO "C88"]
[Annotator "Stockfish"]
[CustomTag "CustomValue"]

1. e4 e5 2. Nf3 Nc6 3. Bb5 a6 4. Ba4 Nf6 5. O-O Be7 6. Re1 b5 7. Bb3 d6 8. c3 O-O 9. h3 Na5 10. Bc2 c5 11. d4 Qc7 12. Nbd2 cxd4 13. cxd4 Bd7 14. Nf1 Rac8 15. Ne3 Nc6 1-0`

	db, _ := createTestDatabase(t, 0)
	defer db.Close()

	stats, err := db.ImportPGNText(pgnText)
	require.NoError(t, err)
	assert.Equal(t, 1, stats.Imported)
	assert.Equal(t, 1, db.NumGames())

	game, err := db.GetGame(1)
	require.NoError(t, err)
	assert.Equal(t, "Nakamura, Hikaru", game.Tag("White"))
	assert.Equal(t, "Radjabov, Teimour", game.Tag("Black"))
	assert.Equal(t, "Madrid ESP", game.Tag("Site"))
	assert.Equal(t, "2760", game.Tag("WhiteElo"))
	assert.Equal(t, "CustomValue", game.Tag("CustomTag"))
	assert.Equal(t, "Stockfish", game.Tag("Annotator"))
	assert.Equal(t, 30, game.PlyCount)

	// Test saving and retrieving a game with custom FEN and promotion
	promoFEN := "8/4P3/8/8/8/8/k7/4K3 w - - 0 1"
	bPromo, err := chess.ParseFEN(promoFEN)
	require.NoError(t, err)

	promoMove, err := bPromo.ParseSAN("e8=Q")
	require.NoError(t, err)

	gPromo := chess.NewGame()
	gPromo.StartFEN = promoFEN
	gPromo.SetTag("White", "Pawn")
	gPromo.SetTag("Black", "King")
	gPromo.SetTag("Result", "1-0")
	gPromo.Result = chess.ResultWhiteWins
	gPromo.RootMove = &chess.MoveNode{
		Move: promoMove,
		SAN:  "e8=Q",
		NAGs: []int{1}, // $1 = good move
		Variations: []*chess.MoveNode{
			{
				Move: chess.Move{From: promoMove.From, To: promoMove.To, Promo: chess.Rook},
				SAN:  "e8=R",
			},
		},
	}
	gPromo.PlyCount = 1

	savedID, err := db.SaveGame(gPromo)
	require.NoError(t, err)
	assert.Equal(t, 2, savedID)

	retrievedPromo, err := db.GetGame(2)
	require.NoError(t, err)
	assert.Equal(t, promoFEN, retrievedPromo.StartFEN)
	assert.NotNil(t, retrievedPromo.RootMove)
	assert.Equal(t, "e8=Q", retrievedPromo.RootMove.SAN)
	assert.Contains(t, retrievedPromo.RootMove.NAGs, 1)
	assert.NotEmpty(t, retrievedPromo.RootMove.Variations)
	assert.Equal(t, "e8=R", retrievedPromo.RootMove.Variations[0].SAN)

	// Test ReadGameBytes directly
	sg4Path := db.BasePath + ".sg4"
	entry := db.Entries[1]
	raw, err := ReadGameBytes(sg4Path, entry.Offset, entry.Length)
	require.NoError(t, err)
	assert.NotEmpty(t, raw)

	_, err = ReadGameBytes("nonexistent.sg4", 0, 10)
	assert.Error(t, err)
}

func TestDatabaseErrorHandling(t *testing.T) {
	db := NewDatabase()

	// Operations on closed database
	_, err := db.SaveGame(chess.NewGame())
	assert.Error(t, err)
	assert.Error(t, db.DeleteGame(1, true))
	_, err = db.GetGame(1)
	assert.Error(t, err)
	_, err = db.GetGameSummary(1)
	assert.Error(t, err)

	// Open nonexistent
	assert.Error(t, db.Open("/nonexistent/path/db"))

	// Create invalid directory
	assert.Error(t, db.Create("/proc/invalid_path/testdb", "desc"))

	// Valid database out of bounds checks
	dbValid, _ := createTestDatabase(t, 2)
	defer dbValid.Close()

	_, err = dbValid.GetGame(0)
	assert.Error(t, err)
	_, err = dbValid.GetGame(99)
	assert.Error(t, err)

	_, err = dbValid.GetGameSummary(0)
	assert.Error(t, err)
	_, err = dbValid.GetGameSummary(99)
	assert.Error(t, err)

	assert.Error(t, dbValid.DeleteGame(0, true))
	assert.Error(t, dbValid.DeleteGame(99, true))
}

func TestIndexHeaderAndEntriesErrors(t *testing.T) {
	// 1. ReadIndexHeader with invalid magic
	badMagic := bytes.NewReader([]byte("InvalidMagicBytesHere1234567890"))
	_, err := ReadIndexHeader(badMagic)
	assert.Error(t, err)

	// 2. ReadIndexHeader with truncated stream
	truncated := bytes.NewReader([]byte("Scid.si\x00"))
	_, err = ReadIndexHeader(truncated)
	assert.Error(t, err)

	// Truncated at version/baseType/numGames
	_, err = ReadIndexHeader(bytes.NewReader([]byte("Scid.si\x00\x01\x90\x00")))
	assert.Error(t, err)

	// 3. ReadIndexEntry with truncated stream
	_, err = ReadIndexEntry(bytes.NewReader([]byte{1, 2, 3}))
	assert.Error(t, err)

	// Test read3 error
	_, err = read3(bytes.NewReader([]byte{1, 2}))
	assert.Error(t, err)

	// 4. LoadAllEntries with nonexistent file
	_, _, err = LoadAllEntries("/nonexistent/file.si4")
	assert.Error(t, err)

	// 5. LoadAllEntries with valid file
	db, basePath := createTestDatabase(t, 2)
	db.Close()

	hdr, entries, err := LoadAllEntries(basePath + ".si4")
	require.NoError(t, err)
	assert.NotNil(t, hdr)
	assert.Len(t, entries, 2)

	// 6. LoadAllEntries where entry reading fails (truncated file after header)
	truncPath := filepath.Join(t.TempDir(), "trunc.si4")
	var buf bytes.Buffer
	err = WriteIndexHeader(&buf, hdr)
	require.NoError(t, err)
	// Write partial entry
	buf.Write([]byte{1, 2, 3})
	err = os.WriteFile(truncPath, buf.Bytes(), 0644)
	require.NoError(t, err)

	_, _, err = LoadAllEntries(truncPath)
	assert.Error(t, err)
}

func TestVerifyMissingFiles(t *testing.T) {
	db, basePath := createTestDatabase(t, 2)
	defer db.Close()

	// Remove .sg4 to trigger missing file detection
	err := os.Remove(basePath + ".sg4")
	require.NoError(t, err)

	report, err := db.Verify()
	require.NoError(t, err)
	assert.NotEmpty(t, report.Errors)
	assert.Equal(t, "CORRUPTED", report.Status)
}

func TestNameBaseErrors(t *testing.T) {
	nb := NewNameBase()

	// Invalid magic
	badReader := bytes.NewReader([]byte("BadMagicBytes"))
	err := nb.Read(badReader)
	assert.Error(t, err)

	// Truncated reader
	truncReader := bytes.NewReader([]byte("Scid.sn\x00"))
	err = nb.Read(truncReader)
	assert.Error(t, err)

	// ReadFromFile on nonexistent
	err = nb.ReadFromFile("/nonexistent/namebase.sn4")
	assert.Error(t, err)

	// WriteToFile on unwritable path
	err = nb.WriteToFile("/proc/unwritable/namebase.sn4")
	assert.Error(t, err)
}

type failWriter struct{}

func (f failWriter) Write(p []byte) (n int, err error) {
	return 0, os.ErrPermission
}

func TestNameBaseHighFrequencyAndVolume(t *testing.T) {
	nb := NewNameBase()
	nb.Counts[NamePlayer] = 1
	nb.MaxFreq[NamePlayer] = 70000
	nb.NamesByID[NamePlayer][0] = "HighFreqPlayer"
	nb.IDByName[NamePlayer]["HighFreqPlayer"] = 0
	nb.Frequencies[NamePlayer][0] = 70000

	var buf bytes.Buffer
	err := nb.Write(&buf)
	require.NoError(t, err)

	nbRead := NewNameBase()
	err = nbRead.Read(&buf)
	require.NoError(t, err)
	assert.Equal(t, "HighFreqPlayer", nbRead.GetName(NamePlayer, 0))
	assert.Equal(t, uint32(70000), nbRead.Frequencies[NamePlayer][0])

	// Test mid-frequency (between 256 and 65536) to cover uint16 encoding
	nbMid := NewNameBase()
	nbMid.Counts[NamePlayer] = 1
	nbMid.MaxFreq[NamePlayer] = 500
	nbMid.NamesByID[NamePlayer][0] = "MidFreqPlayer"
	nbMid.IDByName[NamePlayer]["MidFreqPlayer"] = 0
	nbMid.Frequencies[NamePlayer][0] = 500

	var bufMid bytes.Buffer
	err = nbMid.Write(&bufMid)
	require.NoError(t, err)

	nbMidRead := NewNameBase()
	err = nbMidRead.Read(&bufMid)
	require.NoError(t, err)
	assert.Equal(t, "MidFreqPlayer", nbMidRead.GetName(NamePlayer, 0))
	assert.Equal(t, uint32(500), nbMidRead.Frequencies[NamePlayer][0])

	// Test write errors
	assert.Error(t, nb.Write(failWriter{}))
	for maxW := 0; maxW <= 12; maxW++ {
		sfw := &stepFailWriter{maxWrites: maxW}
		_ = nb.Write(sfw)
	}
}

func TestNameBaseLargeCount(t *testing.T) {
	nb := NewNameBase()
	nb.Counts[NamePlayer] = 65536
	nb.MaxFreq[NamePlayer] = 70000
	for id := uint32(0); id < 65536; id++ {
		nb.NamesByID[NamePlayer][id] = "P"
		nb.Frequencies[NamePlayer][id] = 70000
	}
	var buf bytes.Buffer
	err := nb.Write(&buf)
	require.NoError(t, err)

	nbRead := NewNameBase()
	err = nbRead.Read(&buf)
	require.NoError(t, err)
	assert.Equal(t, uint32(65536), nbRead.Counts[NamePlayer])
}

func TestIndexAndGamedataEdgeCases(t *testing.T) {
	hdr := &IndexHeader{
		Version:  400,
		BaseType: 1,
		NumGames: 1,
	}
	assert.Error(t, WriteIndexHeader(failWriter{}, hdr))

	entry := &IndexEntry{
		Offset: 0,
		Length: 10,
	}
	assert.Error(t, WriteIndexEntry(failWriter{}, entry))

	// DecodePieceMove edge cases
	b := chess.NewBoard()

	// Invalid piece number: reduce piece count so pieceNum 10 is out of bounds
	b.PieceCount[b.Turn] = 5
	_, err := decodePieceMove(bytes.NewReader(nil), b, 0xA0) // pieceNum 10 >= 5
	assert.Error(t, err)

	// Reset piece count
	b.PieceCount[b.Turn] = 16

	// Invalid knight move code
	_, err = decodePieceMove(bytes.NewReader(nil), b, 0x20) // pieceNum 2 (knight B1), code 0
	assert.Error(t, err)

	// White pawn a2-a4
	m, err := decodePieceMove(bytes.NewReader(nil), b, 0x8F) // Piece 8 (A2 pawn) code 15 = double step
	assert.NoError(t, err)
	assert.Equal(t, chess.A2, m.From)
	assert.Equal(t, chess.A4, m.To)

	// ParseECO coverage
	assert.Equal(t, uint16(0), parseECO(""))
	assert.Equal(t, uint16(0), parseECO("X01"))
	assert.Equal(t, uint16(0), parseECO("A"))
	assert.Equal(t, uint16(288), parseECO("c88"))

	// Step failure writer on WriteIndexHeader to cover all error lines
	for maxW := 0; maxW <= 6; maxW++ {
		sfw := &stepFailWriter{maxWrites: maxW}
		_ = WriteIndexHeader(sfw, hdr)
	}

	// Direct DecodeGameData error and custom tag branches
	entryDummy := &IndexEntry{}
	nbDummy := NewNameBase()

	// 1. Unknown tag code > 240
	rawCustomTag := []byte{251, 4, 't', 'e', 's', 't', 0, 0, 0}
	gDecoded, err := DecodeGameData(rawCustomTag, entryDummy, nbDummy)
	require.NoError(t, err)
	assert.Equal(t, "test", gDecoded.Tag("Tag_251"))

	// 2. Truncated custom tag name
	_, err = DecodeGameData([]byte{5, 'a', 'b'}, entryDummy, nbDummy)
	assert.Error(t, err)

	// 3. Truncated value length
	_, err = DecodeGameData([]byte{4, 't', 'e', 's', 't'}, entryDummy, nbDummy)
	assert.Error(t, err)

	// 4. Truncated value bytes
	_, err = DecodeGameData([]byte{4, 't', 'e', 's', 't', 5, 'a'}, entryDummy, nbDummy)
	assert.Error(t, err)

	// 5. Truncated flags byte
	_, err = DecodeGameData([]byte{0}, entryDummy, nbDummy)
	assert.Error(t, err)
}

type stepFailWriter struct {
	maxWrites int
	writes    int
}

func (s *stepFailWriter) Write(p []byte) (n int, err error) {
	s.writes++
	if s.writes > s.maxWrites {
		return 0, os.ErrPermission
	}
	return len(p), nil
}

func TestDatabaseOpenEdgeCasesAndUndelete(t *testing.T) {
	db, basePath := createTestDatabase(t, 3)
	defer db.Close()

	// Test soft-deleting and un-deleting
	err := db.DeleteGame(2, true)
	require.NoError(t, err)
	sum, err := db.GetGameSummary(2)
	require.NoError(t, err)
	assert.True(t, sum.Deleted)

	err = db.DeleteGame(2, false)
	require.NoError(t, err)
	sum, err = db.GetGameSummary(2)
	require.NoError(t, err)
	assert.False(t, sum.Deleted)

	// Missing .sn4 file during Open
	sn4Path := basePath + ".sn4"
	sn4Backup := basePath + ".sn4.bak"
	err = os.Rename(sn4Path, sn4Backup)
	require.NoError(t, err)

	dbMissingSn := NewDatabase()
	err = dbMissingSn.Open(basePath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "names file not found")

	// Restore sn4
	err = os.Rename(sn4Backup, sn4Path)
	require.NoError(t, err)

	// Missing .sg4 file during Open
	sg4Path := basePath + ".sg4"
	sg4Backup := basePath + ".sg4.bak"
	err = os.Rename(sg4Path, sg4Backup)
	require.NoError(t, err)

	dbMissingSg := NewDatabase()
	err = dbMissingSg.Open(basePath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "game data file not found")

	// Restore sg4
	err = os.Rename(sg4Backup, sg4Path)
	require.NoError(t, err)
}

func TestInspectDatabaseAndFileSizes(t *testing.T) {
	// 1. Nonexistent database
	res := InspectDatabase("/nonexistent/path/db", "User")
	assert.False(t, res.Exists)

	// 2. PGN file inspection
	tmpDir := t.TempDir()
	pgnPath := filepath.Join(tmpDir, "test.pgn")
	err := os.WriteFile(pgnPath, []byte("[Event \"Test\"]\n\n1. e4 *"), 0644)
	require.NoError(t, err)

	resPgn := InspectDatabase(pgnPath, "User")
	assert.True(t, resPgn.Exists)
	assert.Equal(t, "pgn", resPgn.Type)

	// 3. FormatFileSize coverage
	assert.Equal(t, "500 B", FormatFileSize(500))
	assert.Equal(t, "1.5 KB", FormatFileSize(1536))
	assert.Equal(t, "2.0 MB", FormatFileSize(2*1024*1024))
	assert.Equal(t, "1.0 GB", FormatFileSize(1024*1024*1024))
}

func TestSaveGameAndImportErrors(t *testing.T) {
	db, _ := createTestDatabase(t, 1)
	defer db.Close()

	// Game with invalid start FEN triggers EncodeGameData error in SaveGame
	invalidGame := chess.NewGame()
	invalidGame.StartFEN = "totally invalid fen format"
	_, err := db.SaveGame(invalidGame)
	assert.Error(t, err)

	// Game with invalid FEN triggers stats.Errors++ in ImportGames
	stats, err := db.ImportGames([]*chess.Game{invalidGame})
	require.NoError(t, err)
	assert.Equal(t, 1, stats.Errors)
	assert.Equal(t, 0, stats.Imported)

	// GetGame with corrupted game bytes
	corruptedDb, basePath := createTestDatabase(t, 1)
	defer corruptedDb.Close()

	// Overwrite game data bytes in .sg4 with garbage
	sg4Path := basePath + ".sg4"
	err = os.WriteFile(sg4Path, []byte{0x00, 0x00, 0xFF, 0xFF, 0x00}, 0644)
	require.NoError(t, err)

	_, err = corruptedDb.GetGame(1)
	assert.Error(t, err)
}

func TestDatabaseUpdateGameTagsAndSpellcheck(t *testing.T) {
	db, _ := createTestDatabase(t, 3)
	defer db.Close()

	// 1. FormatECO
	ecoStr := FormatECO(150)
	assert.Equal(t, "B50", ecoStr)

	// 2. RLock & RUnlock
	db.RLock()
	cnt := db.NumGames()
	db.RUnlock()
	assert.Equal(t, 3, cnt)

	// 3. UpdateGameTags valid
	tags := map[string]string{
		"White":    "Fischer, Robert J.",
		"Black":    "Spassky, Boris V.",
		"Event":    "World Championship Match",
		"Site":     "Reykjavik",
		"Round":    "6",
		"Date":     "1972.07.23",
		"WhiteElo": "2785",
		"BlackElo": "2660",
		"Result":   "1-0",
		"ECO":      "D59",
	}
	err := db.UpdateGameTags(1, tags)
	require.NoError(t, err)

	summary, err := db.GetGameSummary(1)
	require.NoError(t, err)
	assert.Equal(t, "Fischer, Robert J.", summary.White)
	assert.Equal(t, "Spassky, Boris V.", summary.Black)
	assert.Equal(t, "Reykjavik", summary.Site)
	assert.Equal(t, 2785, summary.WhiteElo)

	// 4. UpdateGameTags error cases
	assert.Error(t, db.UpdateGameTags(0, tags))
	assert.Error(t, db.UpdateGameTags(999, tags))

	closedDB := NewDatabase()
	assert.Error(t, closedDB.UpdateGameTags(1, tags))

	// 5. StandardizePlayerNames on Database
	spellSummary, err := db.StandardizePlayerNames()
	require.NoError(t, err)
	assert.NotNil(t, spellSummary)

	// Database with already-canonical names has 0 corrections needed
	cleanDB, _ := createTestDatabase(t, 0)
	defer cleanDB.Close()
	cleanDB.Names.AddName(NamePlayer, "Carlsen, Magnus")
	cleanSummary, err := cleanDB.StandardizePlayerNames()
	require.NoError(t, err)
	assert.Equal(t, 0, cleanSummary.CorrectionsMade)

	// Nil Names in StandardizeDatabase
	cleanDB.Names = nil
	nilNamesSummary, err := cleanDB.StandardizePlayerNames()
	require.NoError(t, err)
	assert.Equal(t, 0, nilNamesSummary.CorrectionsMade)
}

func TestSignatureSetAndMergeEdgeCases(t *testing.T) {
	var nilSet *GameSignatureSet
	assert.Equal(t, 0, nilSet.Count())
	nilSet.AddEntry("W", "B", "S", "2024.01.01", "1-0", 40)
	assert.False(t, nilSet.ContainsEntry("W", "B", "S", "2024.01.01", "1-0", 40))
	assert.False(t, nilSet.ContainsGame(nil))
	nilSet.AddGame(nil)
	gamesIn := []*chess.Game{chess.NewGame()}
	filteredNil, skippedNil := nilSet.FilterNewGames(gamesIn)
	assert.Len(t, filteredNil, 1)
	assert.Equal(t, 0, skippedNil)

	var nilDB *Database
	emptySet := nilDB.BuildGameSignatureSet()
	assert.Equal(t, 0, emptySet.Count())

	set := &GameSignatureSet{} // uninitialized map
	set.AddEntry("Carlsen", "Nakamura", "https://lichess.org/abc", "2024-05-01", "1-0", 60)
	assert.Equal(t, 1, set.Count())
	assert.True(t, set.ContainsEntry("Carlsen", "Nakamura", "Local", "2024.05.01", "1-0", 60))
	assert.True(t, set.ContainsEntry("Other", "Player", "https://lichess.org/abc", "2020.01.01", "*", 10))

	gLink := chess.NewGame()
	gLink.SetTag("White", "Player1")
	gLink.SetTag("Black", "Player2")
	gLink.SetTag("Date", "????.??.??")
	gLink.SetTag("UTCDate", "2024.06.01")
	gLink.SetTag("Link", "https://chess.com/game/123")
	gLink.Result = ""
	gLink.SetTag("Result", "0-1")
	gLink.Result = "" // test fallback to Tag("Result")

	gDup := chess.NewGame()
	gDup.SetTag("Link", "https://chess.com/game/123")

	accepted, skipped := set.FilterNewGames([]*chess.Game{nil, gLink, gDup})
	assert.Len(t, accepted, 1)
	assert.Equal(t, 1, skipped)
	assert.True(t, set.ContainsGame(gLink))

	// MergeFromDatabases edge cases
	closedDB := NewDatabase()
	_, err := closedDB.MergeFromDatabases([]string{"a"}, nil)
	assert.Error(t, err)

	db, _ := createTestDatabase(t, 2)
	defer db.Close()
	_ = db.DeleteGame(2, true) // cover deleted entry in BuildGameSignatureSet
	db.Lock()
	db.Unlock()

	srcDir := t.TempDir()
	srcSi4 := filepath.Join(srcDir, "srcbase")
	sDB := NewDatabase()
	require.NoError(t, sDB.Create(srcSi4, "Src"))
	gNew, _ := chess.ParsePGN("[White \"NewW\"]\n[Black \"NewB\"]\n[WhiteElo \"2500\"]\n[BlackElo \"2400\"]\n[Result \"1-0\"]\n\n1. d4 d5 1-0")
	_, _ = sDB.SaveGame(gNew)
	_ = sDB.Close()

	srcPgn := filepath.Join(srcDir, "src.pgn")
	require.NoError(t, os.WriteFile(srcPgn, []byte("[White \"PgnW\"]\n[Black \"PgnB\"]\n[WhiteElo \"2600\"]\n[BlackElo \"2550\"]\n[Result \"0-1\"]\n\n1. c4 e5 0-1\n"), 0644))

	progressCalled := 0
	badPgn := filepath.Join(srcDir, "nonexistent.pgn")
	ms, err := db.MergeFromDatabases([]string{"   ", badPgn, "/nonexistent/base.si4", srcSi4, srcPgn}, func(src string, sc, imp, sk int) {
		progressCalled++
	})
	require.NoError(t, err)
	assert.Equal(t, 1, ms.Errors)
	assert.Equal(t, 2, ms.Imported)
	assert.Greater(t, progressCalled, 0)
}

func TestGamedataAllPieceMovesAndComments(t *testing.T) {
	db, _ := createTestDatabase(t, 0)
	defer db.Close()

	// Build a game exercising Queen horiz/vert/diag, Rook horiz/vert, Bishop both diags,
	// King moves, Knight moves, Black pawn moves, Clock/EMT comments, long tags, and PlyCount=0 calculation
	g, err := chess.ParsePGN(`[Event "All Piece Moves"]
[White "Alpha"]
[Black "Beta"]
[WhiteElo "2800"]
[BlackElo "2750"]
[Result "1/2-1/2"]

1. e4 {[%clk 0:05:00] [%emt 0:00:02] Opening} 1... e5 2. d4 exd4 3. Qxd4 Nc6 4. Qd1 Bc5 5. Qd3 Qe7 6. Be3 Bb4+ 7. Ke2 Qe5 8. Nf3 Qxb2 9. Nbd2 Bxd2 10. Rb1 Qc3 11. Rb3 Qa5 1/2-1/2`)
	require.NoError(t, err)
	g.PlyCount = 0 // Force SaveGame to count RootMove plies
	g.Tags[""] = "skip_empty_key"
	g.Tags[strings.Repeat("K", 250)] = strings.Repeat("V", 260)

	id, err := db.SaveGame(g)
	require.NoError(t, err)
	assert.Equal(t, 1, id)

	decoded, err := db.GetGame(1)
	require.NoError(t, err)
	assert.Equal(t, "0:05:00", decoded.RootMove.Clock)
	assert.Equal(t, "0:00:02", decoded.RootMove.EMT)
	assert.Contains(t, decoded.RootMove.Comment, "Opening")

	// Test DecodeGameData truncation at every byte offset
	rawBytes, err := EncodeGameData(g)
	require.NoError(t, err)
	for cut := 0; cut < len(rawBytes); cut++ {
		_, _ = DecodeGameData(rawBytes[:cut], db.Entries[0], db.Names)
	}

	// Test ReadIndexHeader truncation at every byte offset
	var hdrBuf bytes.Buffer
	require.NoError(t, WriteIndexHeader(&hdrBuf, db.Header))
	hdrBytes := hdrBuf.Bytes()
	for cut := 0; cut < len(hdrBytes); cut++ {
		_, _ = ReadIndexHeader(bytes.NewReader(hdrBytes[:cut]))
	}

	// Test NameBase.Read truncation at every byte offset
	var nbBuf bytes.Buffer
	require.NoError(t, db.Names.Write(&nbBuf))
	nbBytes := nbBuf.Bytes()
	for cut := 0; cut < len(nbBytes); cut++ {
		tmpNB := NewNameBase()
		_ = tmpNB.Read(bytes.NewReader(nbBytes[:cut]))
	}

	// Test gameResult helper branches
	assert.Equal(t, chess.ResultUnderway, gameResult(nil))
	gRes := chess.NewGame()
	gRes.Result = ""
	gRes.Tags["Result"] = "1-0"
	assert.Equal(t, chess.ResultWhiteWins, gameResult(gRes))
	gRes.Tags["Result"] = "*"
	assert.Equal(t, chess.ResultUnderway, gameResult(gRes))

	// GetGameSummaries offset >= total
	sums, total, err := db.GetGameSummaries(10, 5, SearchCriteria{})
	require.NoError(t, err)
	assert.Empty(t, sums)
	assert.Equal(t, 1, total)
}

func TestSearchAdvAndSpellCheckRemainingBranches(t *testing.T) {
	// MaterialCriteria branches
	var nilMC *MaterialCriteria
	assert.False(t, nilMC.IsActive())

	b := chess.NewBoard()
	mcBlack := DefaultMaterialCriteria()
	mcBlack.BlackQ = 1
	assert.True(t, mcBlack.IsActive())

	mcDiff := DefaultMaterialCriteria()
	mcDiff.DiffRooks = 0
	assert.True(t, mcDiff.IsActive())

	// Individual mismatch checks
	for _, fn := range []func(*MaterialCriteria){
		func(m *MaterialCriteria) { m.WhiteB = 9 },
		func(m *MaterialCriteria) { m.WhiteN = 9 },
		func(m *MaterialCriteria) { m.WhiteP = 9 },
		func(m *MaterialCriteria) { m.BlackB = 9 },
		func(m *MaterialCriteria) { m.BlackN = 9 },
		func(m *MaterialCriteria) { m.BlackP = 9 },
		func(m *MaterialCriteria) { m.DiffRooks = 5 },
		func(m *MaterialCriteria) { m.DiffBishops = 5 },
		func(m *MaterialCriteria) { m.DiffKnights = 5 },
		func(m *MaterialCriteria) { m.DiffPawns = 5 },
	} {
		mc := DefaultMaterialCriteria()
		fn(&mc)
		assert.False(t, mc.MatchesBoard(b))
	}

	// OppositeBishops with no bishops
	emptyB := chess.NewBoard()
	emptyB.Clear()
	mcOpp := DefaultMaterialCriteria()
	mcOpp.OppositeBishops = true
	assert.True(t, mcOpp.IsActive())
	assert.False(t, mcOpp.MatchesBoard(emptyB))

	// Hedgehog and Dragon pawn structures
	bHedge := chess.NewBoard()
	bHedge.Clear()
	for _, sq := range []chess.Square{chess.A6, chess.B6, chess.D6, chess.E6} {
		bHedge.SetPiece(sq, chess.BP)
	}
	assert.True(t, MatchesPawnStructure(bHedge, "hedgehog"))
	bHedge.SetPiece(chess.C7, chess.BP) // block c-file
	assert.False(t, MatchesPawnStructure(bHedge, "hedgehog"))

	bDragon := chess.NewBoard()
	bDragon.Clear()
	for _, sq := range []chess.Square{chess.D6, chess.G6, chess.H7} {
		bDragon.SetPiece(sq, chess.BP)
	}
	assert.True(t, MatchesPawnStructure(bDragon, "dragon"))

	// MatchesGameAnnotation variations & empty
	g := chess.NewGame()
	assert.True(t, MatchesGameAnnotation(g, "", ""))
	g.RootMove = &chess.MoveNode{
		SAN: "e4",
		Variations: []*chess.MoveNode{
			{SAN: "d4", Comment: "secret variation note"},
		},
	}
	assert.True(t, MatchesGameAnnotation(g, "secret variation", ""))

	// MatchEndOnly in database search
	db, _ := createTestDatabase(t, 1)
	defer db.Close()
	mcEnd := DefaultMaterialCriteria()
	mcEnd.WhiteQ = 1
	mcEnd.MatchEndOnly = true
	assert.Equal(t, 1, db.CountMatches(SearchCriteria{Material: &mcEnd}))

	// Verify WARNING status (version mismatch, no errors)
	db.Header.Version = 300
	rep, err := db.Verify()
	require.NoError(t, err)
	assert.Equal(t, "WARNING", rep.Status)

	// SpellChecker edge cases
	sc := NewSpellChecker()
	sc.AddCanonical("")
	sc.AddCanonical("Doe, Jane", "Doe, Jane")
	assert.Equal(t, "Doe, Jane", sc.StandardizeName("Jane Doe"))
	assert.Equal(t, "Garry Kasparov", CleanPlayerName("Garry Kasparov GM"))
	assert.Equal(t, "", CleanPlayerName("   "))
	assert.Equal(t, "   ", sc.StandardizeName("   "))
	assert.Nil(t, sc.FindSuggestions("", 0))
	assert.NotEmpty(t, sc.FindSuggestions("Kasparov, Garry", 1))
	assert.Equal(t, 3, LevenshteinDistance("", "abc"))
	assert.Equal(t, 3, LevenshteinDistance("abc", ""))
	_, err = sc.StandardizeDatabase(nil)
	assert.Error(t, err)
	_, err = sc.StandardizeDatabase(NewDatabase())
	assert.Error(t, err)
}

func TestAdditionalErrorAndEdgeBranches(t *testing.T) {
	// 1. NameBase low/mid/high frequency & long name > 255 write/read & stepFailWriter
	nbLow := NewNameBase()
	nbLow.AddName(NamePlayer, strings.Repeat("A", 260))
	nbLow.AddName(NamePlayer, "SecondPlayer")
	for maxW := 0; maxW <= 20; maxW++ {
		_ = nbLow.Write(&stepFailWriter{maxWrites: maxW})
	}

	nbMid := NewNameBase()
	nbMid.Counts[NamePlayer] = 1
	nbMid.MaxFreq[NamePlayer] = 500
	nbMid.NamesByID[NamePlayer][0] = "Mid"
	nbMid.Frequencies[NamePlayer][0] = 500
	for maxW := 0; maxW <= 15; maxW++ {
		_ = nbMid.Write(&stepFailWriter{maxWrites: maxW})
	}
	var midBuf bytes.Buffer
	require.NoError(t, nbMid.Write(&midBuf))
	for cut := 0; cut < midBuf.Len(); cut++ {
		_ = NewNameBase().Read(bytes.NewReader(midBuf.Bytes()[:cut]))
	}

	nbHigh := NewNameBase()
	nbHigh.Counts[NamePlayer] = 65536
	nbHigh.MaxFreq[NamePlayer] = 70000
	nbHigh.NamesByID[NamePlayer][0] = "H"
	nbHigh.Frequencies[NamePlayer][0] = 70000
	for maxW := 0; maxW <= 15; maxW++ {
		_ = nbHigh.Write(&stepFailWriter{maxWrites: maxW})
	}
	var highBuf bytes.Buffer
	_ = nbHigh.Write(&highBuf)
	for cut := 30; cut <= 45 && cut < highBuf.Len(); cut++ {
		_ = NewNameBase().Read(bytes.NewReader(highBuf.Bytes()[:cut]))
	}

	// 2. MaterialCriteria WhiteR and BlackQ mismatches
	b := chess.NewBoard()
	mcWR := DefaultMaterialCriteria()
	mcWR.WhiteR = 9
	assert.False(t, mcWR.MatchesBoard(b))
	mcBQ := DefaultMaterialCriteria()
	mcBQ.BlackQ = 9
	assert.False(t, mcBQ.MatchesBoard(b))

	// 3. Statistics with > 8 ECOs and > 8 Players, plus invalid short date for parseYear
	dbStats, _ := createTestDatabase(t, 10)
	defer dbStats.Close()
	for i := 0; i < 10; i++ {
		pName := "Player_" + string(rune('A'+i))
		pid := dbStats.Names.AddName(NamePlayer, pName)
		dbStats.Entries[i].WhiteID = pid
		dbStats.Entries[i].EcoCode = uint16(100 + i)
	}
	dbStats.Entries[0].Date = "99"
	st, err := dbStats.GetStatistics()
	require.NoError(t, err)
	assert.Len(t, st.TopOpenings, 8)
	assert.Len(t, st.TopPlayers, 8)
	assert.Equal(t, 0, dbStats.CountMatches(SearchCriteria{YearFrom: 2000, ECO: "A00"}))

	// 4. Verify with invalid BlackID, size mismatch, and missing si4/sn4
	dbVer, verPath := createTestDatabase(t, 2)
	defer dbVer.Close()
	dbVer.Entries[0].BlackID = 999999
	fSiAppend, err := os.OpenFile(verPath+".si4", os.O_WRONLY|os.O_APPEND, 0644)
	require.NoError(t, err)
	_, _ = fSiAppend.Write([]byte{1, 2, 3})
	_ = fSiAppend.Close()
	rep1, err := dbVer.Verify()
	require.NoError(t, err)
	assert.NotEmpty(t, rep1.Warnings)
	assert.NotEmpty(t, rep1.Errors)

	_ = os.Remove(verPath + ".si4")
	_ = os.Remove(verPath + ".sn4")
	rep2, err := dbVer.Verify()
	require.NoError(t, err)
	assert.Equal(t, "CORRUPTED", rep2.Status)

	// 5. InspectDatabase fallback when si4 header is invalid but size > IndexHeaderSize
	corruptSiDir := t.TempDir()
	corruptSiBase := filepath.Join(corruptSiDir, "badhdr")
	require.NoError(t, os.WriteFile(corruptSiBase+".si4", make([]byte, IndexHeaderSize+IndexEntrySize*3), 0644))
	require.NoError(t, os.WriteFile(corruptSiBase+".sn4", []byte("bad"), 0644))
	require.NoError(t, os.WriteFile(corruptSiBase+".sg4", []byte("bad"), 0644))
	insp := InspectDatabase(corruptSiBase, "User")
	assert.True(t, insp.Exists)
	assert.Equal(t, 3, insp.NumGames)

	// 6. Open with corrupt .sn4 and corrupt .si4
	require.Error(t, NewDatabase().Open(corruptSiBase))
	validNB := NewNameBase()
	require.NoError(t, validNB.WriteToFile(corruptSiBase+".sn4"))
	require.Error(t, NewDatabase().Open(corruptSiBase))

	// 7. File I/O error branches when BasePath points to invalid path on open DB
	dbBadIO, _ := createTestDatabase(t, 2)
	defer dbBadIO.Close()
	_ = dbBadIO.DeleteGame(1, true)
	dbBadIO.BasePath = "/proc/invalid_scid_dir/db"

	_, err = dbBadIO.SaveGame(chess.NewGame())
	assert.Error(t, err)
	assert.Error(t, dbBadIO.DeleteGame(1, false))
	assert.Error(t, dbBadIO.UpdateGameTags(1, map[string]string{"White": "New Name"}))
	assert.Error(t, dbBadIO.UpdateGameTags(1, map[string]string{"Date": "2025.01.01"}))
	assert.Error(t, dbBadIO.Sort(SortCriteria{Field: SortByDate}))
	_, err = dbBadIO.ImportGames([]*chess.Game{chess.NewGame()})
	assert.Error(t, err)
	_, err = dbBadIO.MergeFromDatabases([]string{"src"}, nil)
	assert.Error(t, err)
	_, err = dbBadIO.FindDuplicates(DuplicateCriteria{ExactMoves: true})
	assert.Error(t, err)
	_, err = dbBadIO.DeleteDuplicates(DuplicateCriteria{ExactMoves: false})
	assert.Error(t, err)

	dbBadIO.Names.AddName(NamePlayer, "Kasparov, G.")
	dbBadIO.Entries[1].WhiteID = 0
	_, err = dbBadIO.StandardizePlayerNames()
	assert.Error(t, err)

	// 8. DecodeGameData with non-standard start FEN (valid & invalid)
	gCustomFEN := chess.NewGame()
	gCustomFEN.StartFEN = "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq - 0 1"
	rawCustom, err := EncodeGameData(gCustomFEN)
	require.NoError(t, err)
	for cut := 0; cut < len(rawCustom); cut++ {
		_, _ = DecodeGameData(rawCustom[:cut], &IndexEntry{}, NewNameBase())
	}
	// Corrupt start FEN inside encoded bytes
	badFENBytes := append([]byte{0, 1}, []byte("invalid_fen\x00")...)
	_, err = DecodeGameData(badFENBytes, &IndexEntry{}, NewNameBase())
	assert.Error(t, err)

	// 9. decodePieceMove invalid King code & empty square, Queen horizontal move, dual EncodeComment
	bKing := chess.NewBoard()
	_, err = decodePieceMove(bytes.NewReader(nil), bKing, 0x0B) // King code 11 out of bounds
	assert.Error(t, err)
	bKing.Squares[bKing.PieceList[chess.White][0]] = chess.Empty
	_, err = decodePieceMove(bytes.NewReader(nil), bKing, 0x01) // Empty square at From
	assert.Error(t, err)

	// Queen horizontal move (Qd4-a4)
	gQHoriz, err := chess.ParsePGN("[White \"W\"]\n[Black \"B\"]\n\n1. d4 d5 2. Qd3 Nc6 3. Qa3 *")
	require.NoError(t, err)
	rawQH, err := EncodeGameData(gQHoriz)
	require.NoError(t, err)
	_, err = DecodeGameData(rawQH, &IndexEntry{}, NewNameBase())
	require.NoError(t, err)

	// Dual EncodeComment on same move in raw stream: 0 tags, 0 flags, e2e4 (0xC1), EncodeComment, EncodeComment, EncodeEndGame, "c1\x00c2\x00"
	dualCommBytes := append([]byte{0, 0, 0xC1, EncodeComment, EncodeComment, EncodeEndGame}, []byte("c1\x00c2\x00")...)
	gDual, err := DecodeGameData(dualCommBytes, &IndexEntry{}, NewNameBase())
	require.NoError(t, err)
	assert.Equal(t, "c1 c2", gDual.RootMove.Comment)

	// 10. Missing .si4 error paths while .sg4/.sn4 exist
	dbNoSi, noSiBase := createTestDatabase(t, 2)
	defer dbNoSi.Close()
	// Save a custom FEN game for matchesCriteria custom FEN + bad offset
	_, _ = dbNoSi.SaveGame(gCustomFEN)
	mcAny := DefaultMaterialCriteria()
	mcAny.WhiteQ = 1
	assert.Equal(t, 3, dbNoSi.CountMatches(SearchCriteria{Material: &mcAny}))

	// Bad offset on entry 1: covers FindDuplicates ReadAt fallback and matchesCriteria getGameLocked error
	dbNoSi.Entries[0].Offset = 99999999
	_, err = dbNoSi.FindDuplicates(DuplicateCriteria{ExactMoves: true})
	require.NoError(t, err)
	assert.Equal(t, 0, dbNoSi.CountMatches(SearchCriteria{CommentText: "missing"}))

	// Remove .si4 and test SaveGame, ImportGames, MergeFromDatabases, StandardizePlayerNames
	_ = os.Remove(noSiBase + ".si4")
	_, err = dbNoSi.SaveGame(chess.NewGame())
	assert.Error(t, err)
	_, err = dbNoSi.ImportGames([]*chess.Game{chess.NewGame()})
	assert.Error(t, err)
	_, err = dbNoSi.MergeFromDatabases([]string{"src"}, nil)
	assert.Error(t, err)
	dbNoSi.Names.AddName(NamePlayer, "Kasparov, G.")
	_, err = dbNoSi.StandardizePlayerNames()
	assert.Error(t, err)

	// 11. MergeFromDatabases with corrupt .si4 source and source with deleted + bad-offset entries
	dbMergeTarget, _ := createTestDatabase(t, 1)
	defer dbMergeTarget.Close()
	dbSrcWithErrs, srcErrBase := createTestDatabase(t, 2)
	_ = dbSrcWithErrs.DeleteGame(1, true)
	dbSrcWithErrs.Entries[1].Offset = 99999999
	// Rewrite entry 2 with bad offset to disk so reopened srcDB has bad offset
	fSiSrc, err := os.OpenFile(srcErrBase+".si4", os.O_RDWR, 0644)
	require.NoError(t, err)
	_, _ = fSiSrc.Seek(int64(IndexHeaderSize+IndexEntrySize), 0)
	_ = WriteIndexEntry(fSiSrc, dbSrcWithErrs.Entries[1])
	_ = fSiSrc.Close()
	_ = dbSrcWithErrs.Close()

	ms2, err := dbMergeTarget.MergeFromDatabases([]string{corruptSiBase, srcErrBase}, nil)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, ms2.Errors, 2)

	// BuildGameSignatureSet on closed DB
	assert.Equal(t, 0, NewDatabase().BuildGameSignatureSet().Count())
}






