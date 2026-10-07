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
