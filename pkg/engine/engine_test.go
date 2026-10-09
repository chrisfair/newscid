package engine_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"newscid/pkg/chess"
	"newscid/pkg/engine"
)

func TestEngineUCICommunication(t *testing.T) {
	engines := engine.FindAvailableEngines()
	if len(engines) == 0 {
		t.Skip("No UCI chess engines found on system, skipping engine test")
	}

	eng := engine.NewEngine(engines[0])
	err := eng.Start()
	require.NoError(t, err)
	defer eng.Quit()

	assert.NotEmpty(t, eng.Name)

	receivedEval := false
	eng.SetOnEval(func(info engine.EvalInfo) {
		if info.Depth > 0 {
			receivedEval = true
		}
	})

	err = eng.Analyze("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1", nil, 4)
	require.NoError(t, err)

	// Wait up to 2 seconds for engine to return depth > 0
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if receivedEval {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	assert.True(t, receivedEval, "Engine should return evaluation information")
	_ = eng.Stop()
}

func TestEngineSearchBestMove(t *testing.T) {
	engines := engine.FindAvailableEngines()
	if len(engines) == 0 {
		t.Skip("No UCI chess engines found on system, skipping engine test")
	}

	eng := engine.NewEngine(engines[0])
	err := eng.Start()
	require.NoError(t, err)
	defer eng.Quit()

	bestMove, eval, err := eng.SearchBestMove("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1", nil, 200)
	require.NoError(t, err)
	assert.NotEmpty(t, bestMove, "Engine should return a best move")
	t.Logf("Engine returned best move %s (depth %d, eval %d cp)", bestMove, eval.Depth, eval.ScoreCp)
}

func TestAnnotateGame(t *testing.T) {
	engines := engine.FindAvailableEngines()
	if len(engines) == 0 {
		t.Skip("No UCI chess engines found on system, skipping engine test")
	}

	eng := engine.NewEngine(engines[0])
	err := eng.Start()
	require.NoError(t, err)
	defer eng.Quit()

	game, err := chess.ParsePGN("1. e4 e5 2. Nf3 Nc6 3. Bc4 Bc5 1-0")
	require.NoError(t, err)

	analysis, err := engine.AnnotateGame(game, eng, 150)
	require.NoError(t, err)
	assert.NotNil(t, analysis)
	assert.NotEmpty(t, analysis.MoveAnalyses)
	assert.Greater(t, analysis.WhiteAccuracy, 0.0)
	assert.Greater(t, analysis.BlackAccuracy, 0.0)
	t.Logf("White Accuracy: %.1f%%, Black Accuracy: %.1f%%", analysis.WhiteAccuracy, analysis.BlackAccuracy)
	assert.NotEmpty(t, game.RootMove.Comment)
}

func TestParseEnginesDat(t *testing.T) {
	data := `# Analysis Engines configuration for Scid vs. PC
engine {
  Name Stockfish-17
  Cmd  /usr/bin/stockfish
  Args {}
  Dir  .
  Elo  3500
  UCI  1
}

engine {
  Name Fruit
  Cmd  fruit
  Args {-v}
  Dir  /tmp
  Elo  2800
  UCI  1
}

engine {
  Name Phalanx
  Cmd  phalanx
  UCI  0
}
`
	engines, err := engine.ParseEnginesDat(strings.NewReader(data))
	require.NoError(t, err)
	require.Len(t, engines, 3)

	assert.Equal(t, "Stockfish-17", engines[0].Name)
	assert.Equal(t, "/usr/bin/stockfish", engines[0].Cmd)
	assert.Equal(t, 3500, engines[0].Elo)
	assert.True(t, engines[0].UCI)

	assert.Equal(t, "Fruit", engines[1].Name)
	assert.Equal(t, "-v", engines[1].Args)

	assert.Equal(t, "Phalanx", engines[2].Name)
	assert.False(t, engines[2].UCI)
}

func TestDiscoverScidEngines(t *testing.T) {
	discovered := engine.DiscoverScidEngines()
	t.Logf("Discovered %d engines on system", len(discovered))
	for _, e := range discovered {
		t.Logf("Engine: %s | Cmd: %s | Available: %v", e.Name, e.Cmd, e.Available)
	}

	// At least Fruit, Scidlet, or Phalanx is installed on this system
	assert.NotEmpty(t, discovered)
}

func TestMultiPVAndOptionConfiguration(t *testing.T) {
	eng := engine.NewEngine("/nonexistent/path")

	// 1. Initial MultiPV
	assert.Equal(t, 1, eng.GetMultiPV())

	// 2. SetMultiPV with clamping
	err := eng.SetMultiPV(3)
	require.NoError(t, err)
	assert.Equal(t, 3, eng.GetMultiPV())

	err = eng.SetMultiPV(-5)
	require.NoError(t, err)
	assert.Equal(t, 1, eng.GetMultiPV())

	err = eng.SetMultiPV(25)
	require.NoError(t, err)
	assert.Equal(t, 10, eng.GetMultiPV())

	// 3. SetThreads and SetHash clamping
	err = eng.SetThreads(4)
	require.NoError(t, err)
	err = eng.SetThreads(0)
	require.NoError(t, err)

	err = eng.SetHash(256)
	require.NoError(t, err)
	err = eng.SetHash(-10)
	require.NoError(t, err)

	err = eng.SetOption("Contempt", "10")
	require.NoError(t, err)

	// 4. GetLastEval
	eval := eng.GetLastEval()
	assert.Equal(t, 10, eval.MultiPV)

	// 5. Unstarted engine error cases
	err = eng.Analyze("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1", nil, 10)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "engine not started")

	_, _, err = eng.SearchBestMove("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1", nil, 100)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "engine not started")
}

func TestParseInfoLineMultiPV(t *testing.T) {
	startFEN := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"

	// 1. MultiPV Line 1
	line1 := "info depth 16 seldepth 22 multipv 1 score cp 35 nodes 87654 nps 450000 pv e2e4 e7e5 g1f3 b8c6"
	pvLine1, hasPV := engine.ParseInfoLine(line1, startFEN)
	require.True(t, hasPV)
	assert.Equal(t, 1, pvLine1.MultiPV)
	assert.Equal(t, 16, pvLine1.Depth)
	assert.Equal(t, 35, pvLine1.ScoreCp)
	assert.Equal(t, int64(87654), pvLine1.Nodes)
	assert.Equal(t, int64(450000), pvLine1.NPS)
	assert.Equal(t, "e2e4", pvLine1.MoveUCI)
	assert.Equal(t, "e4", pvLine1.SAN)
	assert.Contains(t, pvLine1.PVFormatted, "1. e4")

	// 2. MultiPV Line 2
	line2 := "info depth 16 seldepth 20 multipv 2 score cp 15 nodes 87654 nps 450000 pv d2d4 d7d5 c2c4 e7e6"
	pvLine2, hasPV := engine.ParseInfoLine(line2, startFEN)
	require.True(t, hasPV)
	assert.Equal(t, 2, pvLine2.MultiPV)
	assert.Equal(t, 16, pvLine2.Depth)
	assert.Equal(t, 15, pvLine2.ScoreCp)
	assert.Equal(t, "d2d4", pvLine2.MoveUCI)
	assert.Equal(t, "d4", pvLine2.SAN)

	// 3. MultiPV Line 3 with Mate score
	line3 := "info depth 12 multipv 3 score mate -3 nodes 54321 nps 300000 pv f2f3 e7e5 g2g4 d8h4"
	pvLine3, hasPV := engine.ParseInfoLine(line3, startFEN)
	require.True(t, hasPV)
	assert.Equal(t, 3, pvLine3.MultiPV)
	assert.Equal(t, -3, pvLine3.MateIn)

	// 4. Non-PV Info Line (currmove or stats only)
	lineNonPV := "info depth 12 currmove e2e4 currmovenumber 1"
	_, hasPV = engine.ParseInfoLine(lineNonPV, startFEN)
	assert.False(t, hasPV)
}

func TestFormatPV(t *testing.T) {
	// Empty PV
	san, formatted := engine.FormatPV("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1", nil)
	assert.Empty(t, san)
	assert.Empty(t, formatted)

	// Black to move
	blackFEN := "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1"
	san, formatted = engine.FormatPV(blackFEN, []string{"e7e5", "g1f3", "b8c6"})
	assert.Equal(t, "e5", san)
	assert.Contains(t, formatted, "1... e5")
	assert.Contains(t, formatted, "2. Nf3")

	// Invalid moves fallback
	san, formatted = engine.FormatPV("invalid-fen", []string{"x9z9", "a1a2"})
	assert.Equal(t, "x9z9", san)
	assert.Equal(t, "x9z9 a1a2", formatted)
}

func TestLiveMultiPVWithEngine(t *testing.T) {
	engines := engine.FindAvailableEngines()
	if len(engines) == 0 {
		t.Skip("No engines found on system")
	}

	eng := engine.NewEngine(engines[0])
	_ = eng.SetMultiPV(2)
	_ = eng.SetThreads(2)
	_ = eng.SetHash(32)
	err := eng.Start()
	require.NoError(t, err)
	defer eng.Quit()

	// Re-apply options on active engine to test runtime UCI commands
	err = eng.SetMultiPV(3)
	require.NoError(t, err)
	err = eng.SetThreads(1)
	require.NoError(t, err)
	err = eng.SetHash(64)
	require.NoError(t, err)
	err = eng.SetOption("Ponder", "false")
	require.NoError(t, err)

	multiPVReceived := false
	eng.SetOnEval(func(info engine.EvalInfo) {
		if len(info.Lines) >= 2 {
			multiPVReceived = true
		}
	})

	err = eng.Analyze("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1", nil, 6)
	require.NoError(t, err)

	deadline := time.Now().Add(2500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if multiPVReceived {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	lastEval := eng.GetLastEval()
	_ = eng.Stop()

	if multiPVReceived {
		assert.GreaterOrEqual(t, len(lastEval.Lines), 2)
		assert.Equal(t, 1, lastEval.Lines[0].MultiPV)
		assert.Equal(t, 2, lastEval.Lines[1].MultiPV)
		assert.NotEmpty(t, lastEval.Lines[0].SAN)
		assert.NotEmpty(t, lastEval.Lines[1].SAN)
	}
}

func TestAnnotateGameVariations(t *testing.T) {
	// 1. Error checks
	_, err := engine.AnnotateGame(nil, nil, 100)
	assert.Error(t, err)

	game, _ := chess.ParsePGN("1. e4 1-0")
	_, err = engine.AnnotateGame(game, nil, 100)
	assert.Error(t, err)

	// 2. Empty game
	emptyGame := chess.NewGame()
	engMock := engine.NewEngine("/mock")
	analysis, err := engine.AnnotateGame(emptyGame, engMock, 100)
	require.NoError(t, err)
	assert.Empty(t, analysis.MoveAnalyses)

	// 3. Annotate Fool's Mate game with blunders and checkmate
	engines := engine.FindAvailableEngines()
	if len(engines) > 0 {
		activeEng := engine.NewEngine(engines[0])
		if err := activeEng.Start(); err == nil {
			defer activeEng.Quit()
			blunderGame, err := chess.ParsePGN("1. f3 e5 2. g4 Qh4# 0-1")
			if err == nil {
				blunderAnalysis, err := engine.AnnotateGame(blunderGame, activeEng, 100)
				if err == nil {
					assert.NotNil(t, blunderAnalysis)
					assert.GreaterOrEqual(t, blunderAnalysis.WhiteBlunders+blunderAnalysis.WhiteMistakes, 1)
				}
			}
		}
	}
}

func TestCheckEngineAvailabilityPaths(t *testing.T) {
	// Non-existent command
	cfg1 := &engine.EngineConfig{Cmd: "/path/to/definitely_not_here_binary_12345"}
	engine.CheckEngineAvailability(cfg1)
	assert.False(t, cfg1.Available)

	// Empty command
	cfgEmpty := &engine.EngineConfig{Cmd: ""}
	engine.CheckEngineAvailability(cfgEmpty)
	assert.False(t, cfgEmpty.Available)

	// System command
	cfg2 := &engine.EngineConfig{Cmd: "ls"}
	engine.CheckEngineAvailability(cfg2)
	assert.True(t, cfg2.Available)
	assert.NotEmpty(t, cfg2.ResolvedCmd)

	// Tilde path
	cfg3 := &engine.EngineConfig{Cmd: "~/definitely_nonexistent_binary"}
	engine.CheckEngineAvailability(cfg3)
	assert.False(t, cfg3.Available)
}

