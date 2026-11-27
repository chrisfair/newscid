package training

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"newscid/pkg/chess"
	"newscid/pkg/tablebase"
)

func TestEndgameCategoryEnumAndCatalog(t *testing.T) {
	for _, tc := range []struct {
		c EndgameCategory
		s string
	}{
		{EndgameCatPawn, "pawn"},
		{EndgameCatRook, "rook"},
		{EndgameCatMinor, "minor"},
		{EndgameCatQueen, "queen"},
	} {
		assert.Equal(t, tc.s, tc.c.String())
		parsed, ok := ParseEndgameCategory(tc.s)
		assert.True(t, ok)
		assert.Equal(t, tc.c, parsed)
		raw, err := json.Marshal(tc.c)
		require.NoError(t, err)
		var out EndgameCategory
		require.NoError(t, json.Unmarshal(raw, &out))
		assert.Equal(t, tc.c, out)
	}
	assert.Equal(t, "pawn", EndgameCategory(99).String())
	_, ok := ParseEndgameCategory("unknown")
	assert.False(t, ok)
	var bad EndgameCategory
	assert.Error(t, json.Unmarshal([]byte(`123`), &bad))

	all := GetEndgameDrills("all")
	assert.GreaterOrEqual(t, len(all), 14)

	// Verify every curated endgame FEN parses and its first move is legal!
	for _, d := range all {
		b, err := chess.ParseFEN(d.FEN)
		require.NoError(t, err, "bad FEN in %s", d.ID)
		mv, err := b.ParseUCI(d.FirstMoveUCI)
		require.NoError(t, err, "illegal FirstMoveUCI %s in %s", d.FirstMoveUCI, d.ID)
		assert.NotEmpty(t, b.MoveToSAN(mv))
	}

	rookDrills := GetEndgameDrills("rook")
	assert.NotEmpty(t, rookDrills)
	for _, d := range rookDrills {
		assert.Equal(t, EndgameCatRook, d.Category)
	}

	d, found := GetEndgameDrill("eg-lucena")
	assert.True(t, found)
	assert.Equal(t, "win", d.Objective)
	_, found = GetEndgameDrill("missing")
	assert.False(t, found)
}

func TestEvaluateEndgameMove(t *testing.T) {
	// Invalid FEN
	_, err := EvaluateEndgameMove("invalid", "win", "white", nil)
	assert.Error(t, err)

	// Checkmate (user White wins: Qg7#)
	fb, err := EvaluateEndgameMove("7k/6Q1/6K1/8/8/8/8/8 b - - 0 1", "win", "white", nil)
	require.NoError(t, err)
	assert.Equal(t, "won", fb.Status)

	// Stalemate in winning endgame => blunder
	fbStale, err := EvaluateEndgameMove("7k/5Q2/5K2/8/8/8/8/8 b - - 0 1", "win", "white", nil)
	require.NoError(t, err)
	assert.True(t, fbStale.LostObjective)
	assert.Equal(t, "blunder", fbStale.Status)

	// Stalemate in drawing endgame => drawn
	fbDraw, err := EvaluateEndgameMove("7k/5Q2/5K2/8/8/8/8/8 b - - 0 1", "draw", "black", nil)
	require.NoError(t, err)
	assert.False(t, fbDraw.LostObjective)
	assert.Equal(t, "drawn", fbDraw.Status)

	// Mock Syzygy tablebase response (opponent loss => user winning)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"category":"loss","dtz":-12,"moves":[{"uci":"d8e8","san":"Ke8"}]}`))
	}))
	defer srv.Close()

	tb := tablebase.NewClient()
	tb.SetBaseURL(srv.URL)

	fbTB, err := EvaluateEndgameMove("1K1k4/1P6/8/8/8/8/r7/3R4 b - - 1 1", "win", "white", tb)
	require.NoError(t, err)
	assert.True(t, fbTB.TablebaseBacked)
	assert.Equal(t, "optimal", fbTB.Status)
	assert.Equal(t, "d8e8", fbTB.BestReplyUCI)

	// Offline fallback (nil tablebase)
	fbOff, err := EvaluateEndgameMove("1K1k4/1P6/8/8/8/8/r7/3R4 b - - 1 1", "win", "white", nil)
	require.NoError(t, err)
	assert.False(t, fbOff.TablebaseBacked)
	assert.NotEmpty(t, fbOff.BestReplyUCI)
}
