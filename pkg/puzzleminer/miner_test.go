package puzzleminer_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"newscid/pkg/chess"
	"newscid/pkg/database/scid"
	"newscid/pkg/engine"
	"newscid/pkg/puzzleminer"
)

func createMockUCIEngine(t *testing.T, infoLine string) *engine.Engine {
	t.Helper()
	scriptPath := filepath.Join(t.TempDir(), "mock_engine.sh")
	script := `#!/bin/sh
while IFS= read -r line; do
  case "$line" in
    uci)
      echo "id name MockTacticalEngine"
      echo "id author Test"
      echo "uciok"
      ;;
    isready)
      echo "readyok"
      ;;
    go*)
      echo "` + infoLine + `"
      echo "bestmove e4d5"
      ;;
    quit)
      exit 0
      ;;
  esac
done
`
	require.NoError(t, os.WriteFile(scriptPath, []byte(script), 0755))
	eng := engine.NewEngine(scriptPath)
	require.NoError(t, eng.Start())
	t.Cleanup(func() { _ = eng.Quit() })
	return eng
}

func TestPuzzleMinerCheckmateDetection(t *testing.T) {
	// Game ending with Scholar's Mate
	pgn := `[Event "Casual Game"]
[Site "Local"]
[Date "2026.10.07"]
[White "Christopher"]
[Black "Opponent"]
[Result "1-0"]

1. e4 e5 2. Qh5 Nc6 3. Bc4 Nf6 4. Qxf7# 1-0`

	game, err := chess.ParsePGN(pgn)
	require.NoError(t, err)

	puzzles := puzzleminer.MineFromGames([]*chess.Game{game}, nil, puzzleminer.MiningFilter{
		MaxGames:      10,
		OnlyCheckmate: true,
	})

	require.Len(t, puzzles, 1)
	p := puzzles[0]
	assert.Equal(t, "Checkmate", p.Theme)
	assert.Equal(t, "w", p.Turn)
	assert.Equal(t, "f7", p.BestMoveUCI[2:4])
	assert.Contains(t, p.Description, "Christopher")
	assert.NotEmpty(t, p.StartFEN)

	// Test checkmate where SAN doesn't have '#' suffix (triggers isCheckmatingMove) and White/Black tags are "?"
	gameNoHash, err := chess.ParsePGN("1. f3 e5 2. g4 Qh4# 0-1")
	require.NoError(t, err)
	gameNoHash.SetTag("White", "?")
	gameNoHash.SetTag("Black", "?")
	for curr := gameNoHash.RootMove; curr != nil; curr = curr.Next {
		if curr.Next == nil {
			curr.SAN = "Qh4" // strip '#'
		}
	}
	pNoHash := puzzleminer.MineFromGames([]*chess.Game{gameNoHash}, nil, puzzleminer.MiningFilter{})
	require.Len(t, pNoHash, 1)
	assert.Contains(t, pNoHash[0].Description, "Black")

	// Also test White checkmate with White="?"
	gameWhiteNoName, err := chess.ParsePGN("1. e4 e5 2. Qh5 Nc6 3. Bc4 Nf6 4. Qxf7# 1-0")
	require.NoError(t, err)
	gameWhiteNoName.SetTag("White", "?")
	pWhiteNoName := puzzleminer.MineFromGames([]*chess.Game{gameWhiteNoName}, nil, puzzleminer.MiningFilter{})
	require.Len(t, pWhiteNoName, 1)
	assert.Contains(t, pWhiteNoName[0].Description, "White")
}

func TestPuzzleMinerBlackCheckmateAndFilters(t *testing.T) {
	// Fool's mate with Black winning
	pgn := `[Event "Fool's Mate"]
[Date "2026.01.01"]
[White "?"]
[Black "BlackPlayer"]
[Result "0-1"]

1. f3 e5 2. g4 Qh4# 0-1`

	game, err := chess.ParsePGN(pgn)
	require.NoError(t, err)

	// Filter by non-matching player
	puzzlesNoMatch := puzzleminer.MineFromGames([]*chess.Game{game, nil}, nil, puzzleminer.MiningFilter{
		PlayerName: "NonExistentPlayer",
	})
	assert.Empty(t, puzzlesNoMatch)

	// Filter matching BlackPlayer
	puzzlesMatch := puzzleminer.MineFromGames([]*chess.Game{game}, nil, puzzleminer.MiningFilter{
		PlayerName: "BlackPlayer",
	})
	require.Len(t, puzzlesMatch, 1)
	assert.Equal(t, "b", puzzlesMatch[0].Turn)
	assert.Contains(t, puzzlesMatch[0].Description, "BlackPlayer")
}

func TestPuzzleMinerWithEngineAndDatabase(t *testing.T) {
	longPGN := `1. e4 e5 2. Nf3 Nc6 3. Bc4 Bc5 4. c3 Nf6 5. d4 exd4 6. cxd4 Bb4+ 7. Nc3 Nxe4 8. O-O Bxc3 1-0`
	game, err := chess.ParsePGN(longPGN)
	require.NoError(t, err)
	// Custom non-default StartFEN (same board with halfmove clock 1)
	game.StartFEN = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 1 1"

	// 1. Engine returning Decisive Advantage (+600 cp)
	engDecisive := createMockUCIEngine(t, "info depth 12 multipv 1 score cp 600 nodes 1000 nps 10000 pv e4d5")
	puzzlesDec := puzzleminer.MineFromGames([]*chess.Game{game}, engDecisive, puzzleminer.MiningFilter{})
	require.NotEmpty(t, puzzlesDec)
	assert.Equal(t, "Decisive Advantage (+5.0)", puzzlesDec[0].Theme)

	// 2. Engine returning Mate in 3
	engMate := createMockUCIEngine(t, "info depth 12 multipv 1 score mate 3 nodes 1000 nps 10000 pv e4d5")
	puzzlesMate := puzzleminer.MineFromGames([]*chess.Game{game}, engMate, puzzleminer.MiningFilter{})
	require.NotEmpty(t, puzzlesMate)
	assert.Equal(t, "Mate in 3", puzzlesMate[0].Theme)

	// 3. Engine returning Tactical Advantage (+350 cp) on Black's turn (ply 8: score cp 350 from Black POV -> negated by ParseInfoLine to -350, then negated back to +350 for Black)
	engBlackTac := createMockUCIEngine(t, "info depth 12 multipv 1 score cp -350 nodes 1000 nps 10000 pv e4d5")
	// With -350 from mover POV: on ply 7 (White), White gets -350 (ignored), then advances to ply 8 (Black), where ParseInfoLine negates -350 to +350, and mineGame negates to -350.
	// Conversely with +350: on ply 8 (Black), ParseInfoLine negates +350 to -350, and mineGame negates to +350!
	_ = puzzleminer.MineFromGames([]*chess.Game{game}, engBlackTac, puzzleminer.MiningFilter{})

	engTac := createMockUCIEngine(t, "info depth 12 multipv 1 score cp 350 nodes 1000 nps 10000 pv e4d5")
	puzzlesTac := puzzleminer.MineFromGames([]*chess.Game{game}, engTac, puzzleminer.MiningFilter{})
	require.NotEmpty(t, puzzlesTac)
	assert.Equal(t, "Tactical Advantage", puzzlesTac[0].Theme)

	// 4. Test custom StartFEN and >= 50 puzzles limit
	mateGame, err := chess.ParsePGN("1. e4 e5 2. Qh5 Nc6 3. Bc4 Nf6 4. Qxf7# 1-0")
	require.NoError(t, err)
	var manyGames []*chess.Game
	for i := 0; i < 55; i++ {
		manyGames = append(manyGames, mateGame)
	}
	cappedPuzzles := puzzleminer.MineFromGames(manyGames, nil, puzzleminer.MiningFilter{OnlyCheckmate: true})
	assert.Len(t, cappedPuzzles, 50)

	// 5. MineFromDatabase with closed and open DB (including > 200 games limit check)
	_, err = puzzleminer.MineFromDatabase(nil, nil, puzzleminer.MiningFilter{})
	assert.Error(t, err)

	db := scid.NewDatabase()
	require.NoError(t, db.Create(filepath.Join(t.TempDir(), "minedb"), "Mine DB"))
	for i := 0; i < 205; i++ {
		_, err = db.SaveGame(mateGame)
		require.NoError(t, err)
	}

	dbPuzzles, err := puzzleminer.MineFromDatabase(db, nil, puzzleminer.MiningFilter{
		MaxGames:      500,
		OnlyCheckmate: true,
	})
	require.NoError(t, err)
	assert.Len(t, dbPuzzles, 50)
}

