package tournament

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"newscid/pkg/chess"
	"newscid/pkg/database/scid"
)

func TestFideCategory(t *testing.T) {
	assert.Equal(t, 0, CalculateFideCategory(2200))
	assert.Equal(t, 0, CalculateFideCategory(2250))
	assert.Equal(t, 1, CalculateFideCategory(2251))
	assert.Equal(t, 1, CalculateFideCategory(2275))
	assert.Equal(t, 2, CalculateFideCategory(2276))
	assert.Equal(t, 20, CalculateFideCategory(2750))
	assert.Equal(t, 21, CalculateFideCategory(2751))
}

func TestFideRatingDifference(t *testing.T) {
	assert.Equal(t, 800, GetFideRatingDifference(1.0))
	assert.Equal(t, 800, GetFideRatingDifference(1.05))
	assert.Equal(t, -800, GetFideRatingDifference(0.0))
	assert.Equal(t, -800, GetFideRatingDifference(-0.1))
	assert.Equal(t, 0, GetFideRatingDifference(0.50))
	assert.Equal(t, 193, GetFideRatingDifference(0.75))
	assert.Equal(t, -193, GetFideRatingDifference(0.25))
}

func TestCalculatePerformanceRating(t *testing.T) {
	// Zero/negative tests
	assert.Equal(t, 0, CalculatePerformanceRating(0, 0, 2500))
	assert.Equal(t, 0, CalculatePerformanceRating(3, 5, 0))
	assert.Equal(t, 1700, CalculatePerformanceRating(0, 5, 2500)) // 2500 - 800
	assert.Equal(t, 3300, CalculatePerformanceRating(5, 5, 2500)) // 2500 + 800

	// 50% score
	assert.Equal(t, 2600, CalculatePerformanceRating(5, 10, 2600))
	// 75% score (dp = +193)
	assert.Equal(t, 2793, CalculatePerformanceRating(7.5, 10, 2600))
	// 25% score (dp = -193)
	assert.Equal(t, 2407, CalculatePerformanceRating(2.5, 10, 2600))

	// Very low rating ceiling
	assert.Equal(t, 0, CalculatePerformanceRating(0, 5, 500))
}

func TestBuildCrosstableFromGames_RoundRobin(t *testing.T) {
	// 4 Players: GM Carlsen (2850), GM Nakamura (2780), IM Praggnanandhaa (2680), FM Smith (2300)
	games := []*scid.GameSummary{
		{
			ID: 1, White: "GM Carlsen, Magnus", Black: "GM Nakamura, Hikaru",
			Result: chess.ResultWhiteWins, WhiteElo: 2850, BlackElo: 2780, Round: "1", Event: "Super Masters", Site: "Oslo", Date: "2024.01.10",
		},
		{
			ID: 2, White: "GM Carlsen, Magnus", Black: "IM Praggnanandhaa, R",
			Result: chess.ResultDraw, WhiteElo: 2850, BlackElo: 2680, Round: "2", Event: "Super Masters", Site: "Oslo", Date: "2024.01.11",
		},
		{
			ID: 3, White: "FM Smith, John", Black: "GM Carlsen, Magnus",
			Result: chess.ResultBlackWins, WhiteElo: 2300, BlackElo: 2850, Round: "3", Event: "Super Masters", Site: "Oslo", Date: "2024.01.12",
		},
		{
			ID: 4, White: "GM Nakamura, Hikaru", Black: "IM Praggnanandhaa, R",
			Result: chess.ResultWhiteWins, WhiteElo: 2780, BlackElo: 2680, Round: "3", Event: "Super Masters", Site: "Oslo", Date: "2024.01.12",
		},
		{
			ID: 5, White: "GM Nakamura, Hikaru", Black: "FM Smith, John",
			Result: chess.ResultWhiteWins, WhiteElo: 2780, BlackElo: 2300, Round: "2", Event: "Super Masters", Site: "Oslo", Date: "2024.01.11",
		},
		{
			ID: 6, White: "IM Praggnanandhaa, R", Black: "FM Smith, John",
			Result: chess.ResultWhiteWins, WhiteElo: 2680, BlackElo: 2300, Round: "1", Event: "Super Masters", Site: "Oslo", Date: "2024.01.10",
		},
	}

	ct := BuildCrosstableFromGames(games, "Super Masters", "Oslo", "2024.01.10")
	require.NotNil(t, ct)
	assert.Equal(t, "Super Masters", ct.Event)
	assert.Equal(t, 4, ct.NumPlayers)
	assert.Equal(t, 6, ct.NumGames)
	assert.Equal(t, TypeRoundRobin, ct.Type)
	assert.True(t, ct.Category > 0)
	assert.True(t, ct.AverageElo > 2500)

	// Standings verification
	// 1. Carlsen: 2.5 pts (W vs Nakamura, W vs Smith, D vs Pragg)
	// 2. Nakamura: 2.0 pts (W vs Pragg, W vs Smith, L vs Carlsen)
	// 3. Pragg: 1.5 pts (W vs Smith, D vs Carlsen, L vs Nakamura)
	// 4. Smith: 0.0 pts
	require.Len(t, ct.Standings, 4)
	assert.Equal(t, "GM Carlsen, Magnus", ct.Standings[0].PlayerName)
	assert.Equal(t, 2.5, ct.Standings[0].Points)
	assert.Equal(t, 2, ct.Standings[0].Wins)
	assert.Equal(t, 1, ct.Standings[0].Draws)
	assert.Equal(t, 0, ct.Standings[0].Losses)
	assert.Equal(t, "GM", ct.Standings[0].Title)

	assert.Equal(t, "GM Nakamura, Hikaru", ct.Standings[1].PlayerName)
	assert.Equal(t, 2.0, ct.Standings[1].Points)

	assert.Equal(t, "IM Praggnanandhaa, R", ct.Standings[2].PlayerName)
	assert.Equal(t, 1.5, ct.Standings[2].Points)
	assert.Equal(t, "IM", ct.Standings[2].Title)

	assert.Equal(t, "FM Smith, John", ct.Standings[3].PlayerName)
	assert.Equal(t, 0.0, ct.Standings[3].Points)
	assert.Equal(t, "FM", ct.Standings[3].Title)

	// Sonneborn-Berger checks:
	// Carlsen beat Nakamura (2.0) and Smith (0.0), drew Pragg (1.5 * 0.5 = 0.75) => 2.75
	assert.InDelta(t, 2.75, ct.Standings[0].SonnebornBerger, 0.01)

	// Round-robin matrix check
	assert.Len(t, ct.Matrix, 4)
	assert.Equal(t, "X", ct.Matrix[0][0].Result)
	// Carlsen vs Nakamura (row 0, col 1): Carlsen won as White
	assert.Equal(t, "1", ct.Matrix[0][1].Result)
	assert.Equal(t, "w", ct.Matrix[0][1].Color)
	assert.Equal(t, 1, ct.Matrix[0][1].GameID)
	// Nakamura vs Carlsen (row 1, col 0): Nakamura lost as Black
	assert.Equal(t, "0", ct.Matrix[1][0].Result)
	assert.Equal(t, "b", ct.Matrix[1][0].Color)

	// Text cross-table
	assert.Contains(t, ct.CrossTableText, "Super Masters")
	assert.Contains(t, ct.CrossTableText, "GM Carlsen, Magnus")
	assert.Contains(t, ct.CrossTableText, "Pts")
}

func TestBuildCrosstableFromGames_Swiss(t *testing.T) {
	// 6 players, 2 rounds (Swiss pairing)
	games := []*scid.GameSummary{
		{ID: 1, White: "Alice", Black: "Bob", Result: chess.ResultWhiteWins, WhiteElo: 2200, BlackElo: 2100, Round: "1", Event: "Open Swiss"},
		{ID: 2, White: "Charlie", Black: "David", Result: chess.ResultWhiteWins, WhiteElo: 2000, BlackElo: 1900, Round: "1", Event: "Open Swiss"},
		{ID: 3, White: "Eve", Black: "Frank", Result: chess.ResultDraw, WhiteElo: 1800, BlackElo: 1850, Round: "1", Event: "Open Swiss"},
		// Round 2: Winners play winners, etc.
		{ID: 4, White: "Alice", Black: "Charlie", Result: chess.ResultWhiteWins, WhiteElo: 2200, BlackElo: 2000, Round: "2", Event: "Open Swiss"},
		{ID: 5, White: "Bob", Black: "Eve", Result: chess.ResultWhiteWins, WhiteElo: 2100, BlackElo: 1800, Round: "2", Event: "Open Swiss"},
		{ID: 6, White: "David", Black: "Frank", Result: chess.ResultBlackWins, WhiteElo: 1900, BlackElo: 1850, Round: "2", Event: "Open Swiss"},
	}

	ct := BuildCrosstableFromGames(games, "Open Swiss", "", "")
	require.NotNil(t, ct)
	assert.Equal(t, TypeSwiss, ct.Type)
	assert.Equal(t, 6, ct.NumPlayers)
	assert.Equal(t, 6, ct.NumGames)
	assert.Equal(t, 2, ct.NumRounds)

	// Alice is 2.0 (beat Bob and Charlie)
	assert.Equal(t, "Alice", ct.Standings[0].PlayerName)
	assert.Equal(t, 2.0, ct.Standings[0].Points)
	// Bob: 1.0 pt, Charlie: 1.0 pt, Frank: 1.5 pt
	// Check Swiss round results
	require.Len(t, ct.Standings[0].RoundResults, 2)
	assert.Equal(t, "1", ct.Standings[0].RoundResults[0].Round)
	assert.Equal(t, "1", ct.Standings[0].RoundResults[0].Result)
	assert.Equal(t, "w", ct.Standings[0].RoundResults[0].Color)

	assert.Contains(t, ct.CrossTableText, "Open Swiss")
}

func TestBuildCrosstableFromGames_Match(t *testing.T) {
	// 2 players match
	games := []*scid.GameSummary{
		{ID: 1, White: "Kasparov", Black: "Karpov", Result: chess.ResultWhiteWins, Round: "1", Event: "World Championship Match"},
		{ID: 2, White: "Karpov", Black: "Kasparov", Result: chess.ResultDraw, Round: "2", Event: "World Championship Match"},
	}

	ct := BuildCrosstableFromGames(games, "World Championship Match", "", "")
	assert.Equal(t, TypeMatch, ct.Type)
	assert.Equal(t, 2, ct.NumPlayers)
	assert.Equal(t, 2, ct.NumGames)
}

func TestBuildCrosstableFromGames_EmptyAndDeleted(t *testing.T) {
	ctEmpty := BuildCrosstableFromGames(nil, "", "", "")
	assert.Equal(t, 0, ctEmpty.NumGames)
	assert.Equal(t, 0, ctEmpty.NumPlayers)

	deletedGames := []*scid.GameSummary{
		{ID: 1, White: "A", Black: "B", Result: chess.ResultWhiteWins, Deleted: true, Event: "Test"},
	}
	ctDel := BuildCrosstableFromGames(deletedGames, "Test", "", "")
	assert.Equal(t, 0, ctDel.NumGames)
}

func TestDiscoverEventsAndCalculate_Database(t *testing.T) {
	// Test error when db is nil or closed
	events, err := DiscoverEvents(nil)
	assert.Error(t, err)
	assert.Nil(t, events)

	ct, err := CalculateCrosstable(nil, "Event", "", "")
	assert.Error(t, err)
	assert.Nil(t, ct)

	// Create temporary SCID database
	tmpDir, err := os.MkdirTemp("", "scid_tournament_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	basePath := filepath.Join(tmpDir, "tourney_test")
	db := scid.NewDatabase()
	err = db.Create(basePath, "Tournament Test DB")
	require.NoError(t, err)

	// Populate entries directly into db
	w1 := db.Names.AddName(scid.NamePlayer, "Magnus Carlsen")
	b1 := db.Names.AddName(scid.NamePlayer, "Hikaru Nakamura")
	ev1 := db.Names.AddName(scid.NameEvent, "Tata Steel Masters 2024")
	site1 := db.Names.AddName(scid.NameSite, "Wijk aan Zee NED")
	r1 := db.Names.AddName(scid.NameRound, "1")

	db.Entries = append(db.Entries, &scid.IndexEntry{
		WhiteID:   w1,
		BlackID:   b1,
		EventID:   ev1,
		SiteID:    site1,
		RoundID:   r1,
		Date:      "2024.01.15",
		Result:    chess.ResultWhiteWins,
		WhiteElo:  2830,
		BlackElo:  2780,
	})

	// Add second game in same event
	r2 := db.Names.AddName(scid.NameRound, "2")
	db.Entries = append(db.Entries, &scid.IndexEntry{
		WhiteID:   b1,
		BlackID:   w1,
		EventID:   ev1,
		SiteID:    site1,
		RoundID:   r2,
		Date:      "2024.01.16",
		Result:    chess.ResultDraw,
		WhiteElo:  2780,
		BlackElo:  2830,
	})

	// Add game in second event
	ev2 := db.Names.AddName(scid.NameEvent, "World Blitz 2023")
	site2 := db.Names.AddName(scid.NameSite, "Samarkand UZB")
	db.Entries = append(db.Entries, &scid.IndexEntry{
		WhiteID:   w1,
		BlackID:   b1,
		EventID:   ev2,
		SiteID:    site2,
		RoundID:   r1,
		Date:      "2023.12.30",
		Result:    chess.ResultBlackWins,
		WhiteElo:  2880,
		BlackElo:  2875,
	})

	// Discover events
	discovered, err := DiscoverEvents(db)
	require.NoError(t, err)
	require.Len(t, discovered, 2)
	assert.Equal(t, "Tata Steel Masters 2024", discovered[0].Event)
	assert.Equal(t, 2, discovered[0].NumGames)
	assert.Equal(t, 2, discovered[0].NumPlayers)
	assert.Equal(t, "World Blitz 2023", discovered[1].Event)

	// Calculate crosstable for Tata Steel
	crosstable, err := CalculateCrosstable(db, "Tata Steel Masters 2024", "Wijk aan Zee NED", "2024.01.15")
	require.NoError(t, err)
	assert.Equal(t, "Tata Steel Masters 2024", crosstable.Event)
	assert.Equal(t, 2, crosstable.NumPlayers)
	assert.Equal(t, 2, crosstable.NumGames)
	assert.Equal(t, "Magnus Carlsen", crosstable.Standings[0].PlayerName)
	assert.Equal(t, 1.5, crosstable.Standings[0].Points)
}

func TestDoubleRoundRobinAndDirectEncounter(t *testing.T) {
	// 3 players, double round-robin
	// Player A, Player B, Player C
	// A vs B: A wins as White (game 1), B draws as White against A (game 2)
	// A vs C: A draws both (games 3 & 4)
	// B vs C: B wins as White (game 5), C wins as White (game 6)
	games := []*scid.GameSummary{
		{ID: 1, White: "Player A", Black: "Player B", Result: chess.ResultWhiteWins, WhiteElo: 2400, BlackElo: 2400, Round: "1", Event: "Triangular Double RR"},
		{ID: 2, White: "Player B", Black: "Player A", Result: chess.ResultDraw, WhiteElo: 2400, BlackElo: 2400, Round: "4", Event: "Triangular Double RR"},
		{ID: 3, White: "Player A", Black: "Player C", Result: chess.ResultDraw, WhiteElo: 2400, BlackElo: 2400, Round: "2", Event: "Triangular Double RR"},
		{ID: 4, White: "Player C", Black: "Player A", Result: chess.ResultDraw, WhiteElo: 2400, BlackElo: 2400, Round: "5", Event: "Triangular Double RR"},
		{ID: 5, White: "Player B", Black: "Player C", Result: chess.ResultWhiteWins, WhiteElo: 2400, BlackElo: 2400, Round: "3", Event: "Triangular Double RR"},
		{ID: 6, White: "Player C", Black: "Player B", Result: chess.ResultWhiteWins, WhiteElo: 2400, BlackElo: 2400, Round: "6", Event: "Triangular Double RR"},
	}

	ct := BuildCrosstableFromGames(games, "Triangular Double RR", "Local Club", "2024.03.01")
	require.NotNil(t, ct)
	assert.Equal(t, 3, ct.NumPlayers)
	assert.Equal(t, 6, ct.NumGames)
	assert.Equal(t, TypeRoundRobin, ct.Type)

	// Standings:
	// A: 1.0 (vs B) + 0.5 (vs B) + 0.5 (vs C) + 0.5 (vs C) = 2.5 pts
	// B: 0.0 (vs A) + 0.5 (vs A) + 1.0 (vs C) + 0.0 (vs C) = 1.5 pts
	// C: 0.5 (vs A) + 0.5 (vs A) + 0.0 (vs B) + 1.0 (vs B) = 2.0 pts
	// Rank 1: A (2.5)
	// Rank 2: C (2.0)
	// Rank 3: B (1.5)
	assert.Equal(t, "Player A", ct.Standings[0].PlayerName)
	assert.Equal(t, 2.5, ct.Standings[0].Points)
	assert.Equal(t, "Player C", ct.Standings[1].PlayerName)
	assert.Equal(t, 2.0, ct.Standings[1].Points)
	assert.Equal(t, "Player B", ct.Standings[2].PlayerName)
	assert.Equal(t, 1.5, ct.Standings[2].Points)

	// Double round-robin matrix cell string
	// A vs B is row 0, col 2: "1 ½"
	assert.Equal(t, "1 ½", ct.Matrix[0][2].Result)
	// B vs A is row 2, col 0: "0 ½"
	assert.Equal(t, "0 ½", ct.Matrix[2][0].Result)
}

func TestParseRoundNumberAndTitles(t *testing.T) {
	assert.Equal(t, 0, parseRoundNumber(""))
	assert.Equal(t, 5, parseRoundNumber("5"))
	assert.Equal(t, 12, parseRoundNumber("Round 12"))
	assert.Equal(t, 3, parseRoundNumber("3.1"))
	assert.Equal(t, 0, parseRoundNumber("Preliminary"))

	assert.Equal(t, "GM", extractTitle("GM Magnus Carlsen"))
	assert.Equal(t, "WGM", extractTitle("WGM Ju Wenjun"))
	assert.Equal(t, "IM", extractTitle("Praggnanandhaa, R (IM)"))
	assert.Equal(t, "FM", extractTitle("Smith, John [FM]"))
	assert.Equal(t, "", extractTitle("Casual Player"))
}

