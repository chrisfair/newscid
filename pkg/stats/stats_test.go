package stats

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"newscid/pkg/chess"
	"newscid/pkg/database/scid"
)

func createTestDatabase() *scid.Database {
	db := scid.NewDatabase()
	db.IsOpen = true
	db.Names = scid.NewNameBase()

	// Register names
	w1 := db.Names.AddName(scid.NamePlayer, "Kasparov, Garry")
	b1 := db.Names.AddName(scid.NamePlayer, "Karpov, Anatoly")
	w2 := db.Names.AddName(scid.NamePlayer, "Short, Nigel")
	b2 := db.Names.AddName(scid.NamePlayer, "Kasparov, Garry")
	w3 := db.Names.AddName(scid.NamePlayer, "Deep Blue")
	b3 := db.Names.AddName(scid.NamePlayer, "Kasparov, Garry")
	ev := db.Names.AddName(scid.NameEvent, "World Championship")

	// 1. Kasparov vs Karpov: 1985.11.09, 1-0, B44, 40 moves, 2700 vs 2720
	db.Entries = append(db.Entries, &scid.IndexEntry{
		WhiteID:      w1,
		BlackID:      b1,
		EventID:      ev,
		WhiteElo:     2700,
		BlackElo:     2720,
		Result:       chess.ResultWhiteWins,
		Date:         "1985.11.09",
		EcoCode:      144, // B44
		NumHalfMoves: 80,  // 40 moves
	})

	// 2. Short vs Kasparov: 1993.09.07, 0-1, C88, 38 moves, 2655 vs 2815
	db.Entries = append(db.Entries, &scid.IndexEntry{
		WhiteID:      w2,
		BlackID:      b2,
		EventID:      ev,
		WhiteElo:     2655,
		BlackElo:     2815,
		Result:       chess.ResultBlackWins,
		Date:         "1993.09.07",
		EcoCode:      288, // C88
		NumHalfMoves: 76,  // 38 moves
	})

	// 3. Deep Blue vs Kasparov: 1997.05.11, 1-0, B17, 19 moves, 0 vs 2820
	db.Entries = append(db.Entries, &scid.IndexEntry{
		WhiteID:      w3,
		BlackID:      b3,
		EventID:      ev,
		WhiteElo:     0,
		BlackElo:     2820,
		Result:       chess.ResultWhiteWins,
		Date:         "1997.05.11",
		EcoCode:      117, // B17
		NumHalfMoves: 38,  // 19 moves
	})

	// 4. Deleted game (should be skipped)
	db.Entries = append(db.Entries, &scid.IndexEntry{
		WhiteID:      w1,
		BlackID:      b1,
		Flags:        scid.FlagDelete,
		WhiteElo:     2750,
		BlackElo:     2730,
		Result:       chess.ResultDraw,
		Date:         "1986.10.01",
		EcoCode:      300,
		NumHalfMoves: 60,
	})

	// 5. Draw game: 2000.10.20, 1/2-1/2, E04, 55 moves, 2849 vs 2770
	db.Entries = append(db.Entries, &scid.IndexEntry{
		WhiteID:      w1,
		BlackID:      b1,
		EventID:      ev,
		WhiteElo:     2849,
		BlackElo:     2770,
		Result:       chess.ResultDraw,
		Date:         "2000.10.20",
		EcoCode:      404, // E04
		NumHalfMoves: 110, // 55 moves
	})

	return db
}

func TestCalculateRatingTracker(t *testing.T) {
	// 1. Errors on closed db or empty aliases
	_, err := CalculateRatingTracker(nil, []string{"Kasparov, Garry"})
	assert.Error(t, err)

	closedDB := scid.NewDatabase()
	_, err = CalculateRatingTracker(closedDB, []string{"Kasparov, Garry"})
	assert.Error(t, err)

	db := createTestDatabase()
	_, err = CalculateRatingTracker(db, nil)
	assert.Error(t, err)

	// 2. Player with no games
	resEmpty, err := CalculateRatingTracker(db, []string{"Unknown Player"})
	require.NoError(t, err)
	assert.Equal(t, 0, resEmpty.TotalGames)
	assert.Equal(t, 0, resEmpty.RatedGames)

	// 3. Kasparov progression
	res, err := CalculateRatingTracker(db, []string{"Kasparov, Garry", "Garry Kasparov"})
	require.NoError(t, err)
	assert.Equal(t, "Kasparov, Garry", res.Player)
	assert.Equal(t, 4, res.TotalGames)
	assert.Equal(t, 4, res.RatedGames)

	assert.Equal(t, 2700, res.StartRating)
	assert.Equal(t, 2849, res.EndRating)
	assert.Equal(t, 2849, res.PeakRating)
	assert.Equal(t, "2000.10.20", res.PeakDate)
	assert.Equal(t, 2700, res.MinRating)
	assert.Equal(t, 2849, res.MaxRating)

	require.Len(t, res.Points, 4)
	assert.Equal(t, "1985.11.09", res.Points[0].Date)
	assert.Equal(t, 2700, res.Points[0].Rating)
	assert.Equal(t, "Karpov, Anatoly", res.Points[0].Opponent)
	assert.Equal(t, 2720, res.Points[0].OpponentElo)
	assert.Equal(t, "1-0", res.Points[0].Result)

	assert.Equal(t, "1993.09.07", res.Points[1].Date)
	assert.Equal(t, 2815, res.Points[1].Rating)
	assert.Equal(t, "Short, Nigel", res.Points[1].Opponent)
	assert.Equal(t, "0-1", res.Points[1].Result)

	assert.Equal(t, "1997.05.11", res.Points[2].Date)
	assert.Equal(t, 2820, res.Points[2].Rating)
	assert.Equal(t, "Deep Blue", res.Points[2].Opponent)
	assert.Equal(t, "1-0", res.Points[2].Result)

	assert.Equal(t, "2000.10.20", res.Points[3].Date)
	assert.Equal(t, 2849, res.Points[3].Rating)
	assert.Equal(t, "1/2-1/2", res.Points[3].Result)
}

func TestCalculateDatabaseDemographics(t *testing.T) {
	// 1. Errors on closed db
	_, err := CalculateDatabaseDemographics(nil)
	assert.Error(t, err)

	closedDB := scid.NewDatabase()
	_, err = CalculateDatabaseDemographics(closedDB)
	assert.Error(t, err)

	// 2. Empty db
	emptyDB := scid.NewDatabase()
	emptyDB.IsOpen = true
	resEmpty, err := CalculateDatabaseDemographics(emptyDB)
	require.NoError(t, err)
	assert.Equal(t, 0, resEmpty.TotalGames)

	// 3. Populated db
	db := createTestDatabase()
	demo, err := CalculateDatabaseDemographics(db)
	require.NoError(t, err)

	assert.Equal(t, 4, demo.TotalGames) // 4 active games (1 deleted was skipped)
	assert.Equal(t, 4, demo.RatedGames)
	assert.Greater(t, demo.AvgRating, 2600)
	assert.Greater(t, demo.AvgMoves, 30.0)

	// Results breakdown: 2 white wins (1985 & 1997), 1 black win (1993), 1 draw (2000)
	assert.Equal(t, 2, demo.WhiteWins)
	assert.Equal(t, 1, demo.Draws)
	assert.Equal(t, 1, demo.BlackWins)
	assert.InDelta(t, 50.0, demo.WhiteWinPct, 0.1)
	assert.InDelta(t, 25.0, demo.DrawPct, 0.1)
	assert.InDelta(t, 25.0, demo.BlackWinPct, 0.1)

	// Decades
	assert.NotEmpty(t, demo.DecadeDist)
	// Elo Dist
	assert.NotEmpty(t, demo.EloDist)
	// ECO sections: B (2), C (1), E (1)
	assert.NotEmpty(t, demo.ECOSectionDist)
	// Top ECOs
	assert.NotEmpty(t, demo.TopECOs)
	// Move lengths
	assert.NotEmpty(t, demo.MoveLengthDist)
}
