package repertoire_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"newscid/pkg/chess"
	"newscid/pkg/repertoire"
)

func TestRepertoireManager(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "repertoire_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	mgr := repertoire.NewManager(tmpDir)
	list := mgr.ListRepertoires()
	require.NotEmpty(t, list)
	assert.GreaterOrEqual(t, len(list), 2) // White & Black starter repertoires

	// Find the white repertoire
	var whiteRep *repertoire.RepertoireSummary
	for i := range list {
		if list[i].Color == "w" {
			whiteRep = &list[i]
			break
		}
	}
	require.NotNil(t, whiteRep)

	rep, err := mgr.GetRepertoire(whiteRep.ID)
	require.NoError(t, err)
	assert.Equal(t, "w", rep.Color)
	assert.NotEmpty(t, rep.Positions)

	// In initial position, White's move e4 (e2e4) should be valid
	isMatch, move, _, err := mgr.VerifyMove(whiteRep.ID, chess.StartingFEN, "e2e4")
	require.NoError(t, err)
	assert.True(t, isMatch)
	require.NotNil(t, move)
	assert.Equal(t, "e2e4", move.UCI)

	// An off-repertoire move like a2a3 should return false
	isMatchOff, _, options, err := mgr.VerifyMove(whiteRep.ID, chess.StartingFEN, "a2a3")
	require.NoError(t, err)
	assert.False(t, isMatchOff)
	assert.NotEmpty(t, options)

	// Record practice
	mgr.RecordPractice(whiteRep.ID, chess.StartingFEN, "e2e4", true)
	repAfter, _ := mgr.GetRepertoire(whiteRep.ID)
	core := repertoire.CoreFEN(chess.StartingFEN)
	assert.Equal(t, 1, repAfter.Positions[core][0].Practice)
	assert.Equal(t, 1, repAfter.Positions[core][0].Correct)

	// Test saving a custom repertoire
	custom := &repertoire.Repertoire{
		ID:          "custom-caro-kann",
		Name:        "Custom Caro-Kann",
		Color:       "b",
		Description: "Solid defense with c6",
		Positions:   make(map[string][]repertoire.RepertoireMove),
	}
	err = mgr.SaveRepertoire(custom)
	require.NoError(t, err)

	// Verify persistence from disk
	mgr2 := repertoire.NewManager(tmpDir)
	loaded, err := mgr2.GetRepertoire("custom-caro-kann")
	require.NoError(t, err)
	assert.Equal(t, "Custom Caro-Kann", loaded.Name)

	// Test deletion
	err = mgr2.DeleteRepertoire("custom-caro-kann")
	require.NoError(t, err)
	_, err = mgr2.GetRepertoire("custom-caro-kann")
	assert.Error(t, err)

	// Test auto-slug ID and nil Positions on SaveRepertoire
	autoRep := &repertoire.Repertoire{
		Name:  "My Sicilian Lines",
		Color: "b",
	}
	require.NoError(t, mgr2.SaveRepertoire(autoRep))
	assert.Equal(t, "b-my-sicilian-lines", autoRep.ID)
	assert.NotNil(t, autoRep.Positions)

	// VerifyMove and RecordPractice on missing ID
	_, _, _, err = mgr2.VerifyMove("missing-id", chess.StartingFEN, "e2e4")
	assert.Error(t, err)
	mgr2.RecordPractice("missing-id", chess.StartingFEN, "e2e4", false)

	// CoreFEN with short string and leading/trailing spaces
	assert.Equal(t, "short fen", repertoire.CoreFEN("  short fen  "))

	// Test NewManager with empty storageDir (using temporary HOME)
	t.Setenv("HOME", t.TempDir())
	mgrDefault := repertoire.NewManager("")
	assert.NotEmpty(t, mgrDefault.ListRepertoires())
}
