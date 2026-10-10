package playerbot_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"newscid/pkg/chess"
	"newscid/pkg/database/scid"
	"newscid/pkg/playerbot"
)

func createTestDatabaseWithGames(t *testing.T) (*scid.Database, string) {
	tmpDir := t.TempDir()
	basePath := filepath.Join(tmpDir, "bot_test_base")

	db := scid.NewDatabase()
	err := db.Create(basePath, "Bot Test Database")
	require.NoError(t, err)

	// Add 3 sample games featuring "Kasparov, Garry"
	pgnGames := []string{
		`[Event "World Championship"]
[Site "Moscow"]
[Date "1985.10.15"]
[Round "16"]
[White "Karpov, Anatoly"]
[Black "Kasparov, Garry"]
[WhiteElo "2720"]
[BlackElo "2700"]
[Result "0-1"]
[ECO "B44"]

1. e4 c5 2. Nf3 e6 3. d4 cxd4 4. Nxd4 Nc6 5. Nb5 d6 6. c4 Nf6 7. N1c3 a6 0-1`,

		`[Event "Linares"]
[Site "Linares"]
[Date "1990.03.01"]
[Round "1"]
[White "Kasparov, Garry"]
[Black "Anand, Viswanathan"]
[WhiteElo "2800"]
[BlackElo "2650"]
[Result "1-0"]
[ECO "C88"]

1. e4 e5 2. Nf3 Nc6 3. Bb5 a6 4. Ba4 Nf6 5. O-O Be7 1-0`,

		`[Event "Novgorod"]
[Site "Novgorod"]
[Date "1997.05.01"]
[Round "2"]
[White "Kasparov, Garry"]
[Black "Short, Nigel D"]
[WhiteElo "2820"]
[BlackElo "2690"]
[Result "1/2-1/2"]
[ECO "C88"]

1. e4 e5 2. Nf3 Nc6 3. Bb5 a6 4. Ba4 Nf6 5. O-O Be7 1/2-1/2`,

		`[Event "Wijk aan Zee"]
[Site "Wijk aan Zee"]
[Date "1999.01.20"]
[Round "4"]
[White "Kasparov, Garry"]
[Black "Topalov, Veselin"]
[WhiteElo "2812"]
[BlackElo "2700"]
[Result "1-0"]
[ECO "E97"]

1. d4 Nf6 2. c4 g6 3. Nc3 Bg7 1-0`,
	}

	for _, pgn := range pgnGames {
		g, err := chess.ParsePGN(pgn)
		require.NoError(t, err)
		_, err = db.SaveGame(g)
		require.NoError(t, err)
	}

	return db, basePath
}

func TestBuildPersonality(t *testing.T) {
	db, _ := createTestDatabaseWithGames(t)
	defer db.Close()

	// 1. Build personality for Kasparov
	personality, err := playerbot.BuildPersonality(db, "Kasparov")
	require.NoError(t, err)
	assert.Equal(t, "Kasparov", personality.Name)
	assert.Equal(t, 4, personality.TotalGames)
	assert.Equal(t, 3, personality.WhiteGames)
	assert.Equal(t, 1, personality.BlackGames)
	assert.Equal(t, 3, personality.Wins)   // 2 as White, 1 as Black
	assert.Equal(t, 1, personality.Draws)  // 1 as White
	assert.Equal(t, 0, personality.Losses)
	assert.Greater(t, personality.EstimatedElo, 2750)
	assert.Equal(t, 2820, personality.PeakElo)
	assert.Contains(t, personality.TopWhiteECOs, "C88")
	assert.Contains(t, personality.TopBlackECOs, "B44")

	// Verify White opening book contains 1. e4
	startFEN := chess.StartingFEN
	simWhiteMove, inBook := personality.SelectSimulatedMove(startFEN, true)
	assert.True(t, inBook)
	require.NotNil(t, simWhiteMove)
	assert.Equal(t, "e2e4", simWhiteMove.UCI)
	assert.Equal(t, "e4", simWhiteMove.SAN)
	assert.Equal(t, 2, simWhiteMove.PlayedCount)
	assert.Contains(t, simWhiteMove.Commentary, "Kasparov")

	// Verify Black opening book plays 1... c5 against 1. e4
	afterE4 := "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1"
	simBlackMove, inBookBlack := personality.SelectSimulatedMove(afterE4, false)
	assert.True(t, inBookBlack)
	require.NotNil(t, simBlackMove)
	assert.Equal(t, "c7c5", simBlackMove.UCI)
	assert.Equal(t, "c5", simBlackMove.SAN)
	assert.Equal(t, 1, simBlackMove.PlayedCount)

	// Verify Out of book position returns false
	weirdFEN := "8/8/8/8/8/8/8/4K2k w - - 0 1"
	_, inBookOut := personality.SelectSimulatedMove(weirdFEN, true)
	assert.False(t, inBookOut)
}

func TestBuildPersonalityErrors(t *testing.T) {
	db, _ := createTestDatabaseWithGames(t)
	defer db.Close()

	// 1. Nil database
	_, err := playerbot.BuildPersonality(nil, "Kasparov")
	assert.Error(t, err)

	// 2. Empty player name
	_, err = playerbot.BuildPersonality(db, "   ")
	assert.Error(t, err)

	// 3. Player not found
	_, err = playerbot.BuildPersonality(db, "NonExistentPlayer12345")
	assert.Error(t, err)
}

func TestCalibrateEngineSettings(t *testing.T) {
	// Low Elo (< 1320)
	limit, uciElo, skill := playerbot.CalibrateEngineSettings(900)
	assert.True(t, limit)
	assert.Equal(t, 1320, uciElo)
	assert.Equal(t, 1, skill)

	// Intermediate Elo
	limit, uciElo, skill = playerbot.CalibrateEngineSettings(1800)
	assert.True(t, limit)
	assert.Equal(t, 1800, uciElo)
	assert.Equal(t, 10, skill)

	// High GM Elo
	limit, uciElo, skill = playerbot.CalibrateEngineSettings(2850)
	assert.True(t, limit)
	assert.Equal(t, 2850, uciElo)
	assert.Equal(t, 20, skill)

	// Super-human clamped at 3190
	limit, uciElo, skill = playerbot.CalibrateEngineSettings(3500)
	assert.True(t, limit)
	assert.Equal(t, 3190, uciElo)
	assert.Equal(t, 20, skill)

	// Under 800 Elo
	_, _, skillMin := playerbot.CalibrateEngineSettings(600)
	assert.Equal(t, 0, skillMin)
}

func TestBuildPersonalityFromIdentifiers(t *testing.T) {
	db, _ := createTestDatabaseWithGames(t)
	defer db.Close()

	// Multi-alias: "Kasparov" and "Karpov" combined persona
	personality, err := playerbot.BuildPersonalityFromIdentifiers(db, "Soviet Champions", []string{"Kasparov", "Karpov"})
	require.NoError(t, err)
	assert.Equal(t, "Soviet Champions", personality.Name)
	assert.Equal(t, 4, personality.TotalGames)

	// Single alias fallback
	pSingle, err := playerbot.BuildPersonalityFromIdentifiers(db, "Karpov", []string{"Karpov"})
	require.NoError(t, err)
	assert.Equal(t, 1, pSingle.TotalGames)

	// Empty identifiers fallback to displayName
	pEmpty, err := playerbot.BuildPersonalityFromIdentifiers(db, "Kasparov", []string{})
	require.NoError(t, err)
	assert.Equal(t, 4, pEmpty.TotalGames)

	// Empty display name error
	_, err = playerbot.BuildPersonalityFromIdentifiers(db, "   ", []string{"Kasparov"})
	assert.Error(t, err)

	// Style modeling and calibration
	assert.NotEmpty(t, personality.PlayStyle)
	assert.True(t, personality.AggressionRating > 0)
	limit, uciElo, skill, contempt := personality.CalibrateEngineSettingsForPersona()
	assert.True(t, limit)
	assert.True(t, uciElo >= 1320)
	assert.True(t, skill >= 0)
	assert.Equal(t, personality.ContemptValue, contempt)
}
