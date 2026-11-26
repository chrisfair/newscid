package scid

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"newscid/pkg/chess"
)

func TestSortFieldEnumJSONAndString(t *testing.T) {
	fields := []struct {
		field SortField
		name  string
	}{
		{SortByDate, "date"},
		{SortByECO, "eco"},
		{SortByWhite, "white"},
		{SortByBlack, "black"},
		{SortByPlayer, "player"},
		{SortByWhiteElo, "whiteElo"},
		{SortByBlackElo, "blackElo"},
		{SortByMaxElo, "maxElo"},
		{SortByEvent, "event"},
		{SortBySite, "site"},
		{SortByRound, "round"},
		{SortByResult, "result"},
		{SortByMoves, "moves"},
	}

	for _, tc := range fields {
		assert.Equal(t, tc.name, tc.field.String())
		assert.Equal(t, tc.field, ParseSortField(tc.name))

		raw, err := json.Marshal(tc.field)
		require.NoError(t, err)
		assert.Equal(t, `"`+tc.name+`"`, string(raw))

		var decoded SortField
		require.NoError(t, json.Unmarshal(raw, &decoded))
		assert.Equal(t, tc.field, decoded)
	}

	assert.Equal(t, "date", SortField(99).String())
	assert.Equal(t, SortByDate, ParseSortField("unknown"))

	var bad SortField
	assert.Error(t, json.Unmarshal([]byte(`123`), &bad))
}

func TestImportGamesParallelBatch(t *testing.T) {
	base := filepath.Join(t.TempDir(), "batch")
	db := NewDatabase()
	require.NoError(t, db.Create(base, "Batch"))
	defer db.Close()

	var sb strings.Builder
	for i := 0; i < 24; i++ {
		sb.WriteString("[Event \"Batch\"]\n[White \"PlayerA\"]\n[Black \"PlayerB\"]\n[Result \"1-0\"]\n\n1. e4 e5 2. Nf3 Nc6 1-0\n\n")
	}

	games, err := chess.ParsePGNDatabase(strings.NewReader(sb.String()))
	require.NoError(t, err)
	stats, err := db.ImportGames(games)
	require.NoError(t, err)
	assert.Equal(t, 24, stats.Imported)
	assert.Equal(t, 24, db.NumGames())
}
