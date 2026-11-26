package training

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTrainingEnumsJSONAndString(t *testing.T) {
	// ProphylaxisCategory
	for _, tc := range []struct {
		c ProphylaxisCategory
		s string
	}{
		{CategoryKingSafety, "king_safety"},
		{CategoryPieceRestraint, "piece_restraint"},
		{CategoryOverprotection, "overprotection"},
		{CategoryBlockade, "blockade"},
		{CategoryQuietRegroup, "quiet_regroup"},
	} {
		assert.Equal(t, tc.s, tc.c.String())
		parsed, ok := ParseProphylaxisCategory(tc.s)
		assert.True(t, ok)
		assert.Equal(t, tc.c, parsed)
		raw, err := json.Marshal(tc.c)
		require.NoError(t, err)
		var out ProphylaxisCategory
		require.NoError(t, json.Unmarshal(raw, &out))
		assert.Equal(t, tc.c, out)
	}
	assert.Equal(t, "king_safety", ProphylaxisCategory(99).String())
	_, ok := ParseProphylaxisCategory("unknown")
	assert.False(t, ok)
	var badP ProphylaxisCategory
	assert.Error(t, json.Unmarshal([]byte(`123`), &badP))

	// PieceSafetyStatus
	for _, tc := range []struct {
		st PieceSafetyStatus
		s  string
	}{
		{StatusDefended, "defended"},
		{StatusLoose, "loose"},
		{StatusUnderdefended, "underdefended"},
		{StatusHanging, "hanging"},
	} {
		assert.Equal(t, tc.s, tc.st.String())
		assert.Equal(t, tc.st, ParsePieceSafetyStatus(tc.s))
		raw, err := json.Marshal(tc.st)
		require.NoError(t, err)
		var out PieceSafetyStatus
		require.NoError(t, json.Unmarshal(raw, &out))
		assert.Equal(t, tc.st, out)
	}
	assert.Equal(t, "defended", PieceSafetyStatus(99).String())
	var badS PieceSafetyStatus
	assert.Error(t, json.Unmarshal([]byte(`123`), &badS))

	// SafetyDrillMode
	for _, tc := range []struct {
		m SafetyDrillMode
		s string
	}{
		{ModeFindAllHanging, "find_all_hanging"},
		{ModeDefendYourPieces, "defend_your_pieces"},
		{ModePunishBlunder, "punish_blunder"},
		{ModeCaptureHanging, "capture_hanging"},
	} {
		assert.Equal(t, tc.s, tc.m.String())
		assert.Equal(t, tc.m, ParseSafetyDrillMode(tc.s))
		raw, err := json.Marshal(tc.m)
		require.NoError(t, err)
		var out SafetyDrillMode
		require.NoError(t, json.Unmarshal(raw, &out))
		assert.Equal(t, tc.m, out)
	}
	assert.Equal(t, "find_all_hanging", SafetyDrillMode(99).String())
	var badM SafetyDrillMode
	assert.Error(t, json.Unmarshal([]byte(`123`), &badM))

	// IntuitionCategory
	for _, tc := range []struct {
		c IntuitionCategory
		s string
	}{
		{IntuitionEqual, "equal"},
		{IntuitionWhiteWinning, "white_winning"},
		{IntuitionBlackWinning, "black_winning"},
	} {
		assert.Equal(t, tc.s, tc.c.String())
		parsed, ok := ParseIntuitionCategory(tc.s)
		assert.True(t, ok)
		assert.Equal(t, tc.c, parsed)
		raw, err := json.Marshal(tc.c)
		require.NoError(t, err)
		var out IntuitionCategory
		require.NoError(t, json.Unmarshal(raw, &out))
		assert.Equal(t, tc.c, out)
	}
	assert.Equal(t, "equal", IntuitionCategory(99).String())
	_, ok = ParseIntuitionCategory("unknown")
	assert.False(t, ok)
	var badI IntuitionCategory
	assert.Error(t, json.Unmarshal([]byte(`123`), &badI))
}
