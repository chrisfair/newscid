package engine_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
