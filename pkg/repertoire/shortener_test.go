package repertoire

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShortenRepertoire_Pareto(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "repertoire-test-pareto-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	mgr := NewManager(tempDir)
	reps := mgr.ListRepertoires()
	require.NotEmpty(t, reps)

	repID := reps[0].ID

	opts := ShortenerOptions{
		RepertoireID:      repID,
		TargetCoveragePct: 80.0,
		MinFrequencyPct:   5.0,
		MaxDepth:          6,
		PreserveAnnotated: true,
		SaveAsNew:         true,
	}

	result, err := mgr.ShortenRepertoire(opts, nil)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, repID, result.OriginalID)
	assert.Contains(t, result.PrunedRepertoireID, "pareto-80")
	assert.True(t, result.PrunedMoveCount <= result.OriginalMoveCount)
	assert.NotNil(t, result.PrunedRepertoire)

	// Check that pruned repertoire was saved in memory
	saved, err := mgr.GetRepertoire(result.PrunedRepertoireID)
	require.NoError(t, err)
	assert.Equal(t, result.PrunedRepertoireID, saved.ID)
}

func TestShortenRepertoire_MaxDepth(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "repertoire-test-depth-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	mgr := NewManager(tempDir)
	reps := mgr.ListRepertoires()
	require.NotEmpty(t, reps)

	opts := ShortenerOptions{
		RepertoireID:      reps[0].ID,
		TargetCoveragePct: 100.0,
		MaxDepth:          2, // Shallow depth prune
		PreserveAnnotated: false,
		SaveAsNew:         false,
	}

	result, err := mgr.ShortenRepertoire(opts, nil)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.True(t, len(result.PrunedBranches) > 0)
}
