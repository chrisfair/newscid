package tournament

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTournamentTypeEnumJSONAndString(t *testing.T) {
	cases := []struct {
		typ  TournamentType
		text string
	}{
		{TypeRoundRobin, "Round-Robin"},
		{TypeSwiss, "Swiss"},
		{TypeMatch, "Match"},
	}

	for _, tc := range cases {
		assert.Equal(t, tc.text, tc.typ.String())
		assert.Equal(t, tc.typ, ParseTournamentType(tc.text))

		raw, err := json.Marshal(tc.typ)
		require.NoError(t, err)
		assert.Equal(t, `"`+tc.text+`"`, string(raw))

		var decoded TournamentType
		require.NoError(t, json.Unmarshal(raw, &decoded))
		assert.Equal(t, tc.typ, decoded)
	}

	assert.Equal(t, "Round-Robin", TournamentType(99).String())
	assert.Equal(t, TypeRoundRobin, ParseTournamentType("unknown"))

	var bad TournamentType
	assert.Error(t, json.Unmarshal([]byte(`123`), &bad))
}
