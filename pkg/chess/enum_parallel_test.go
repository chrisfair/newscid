package chess

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotationStyleEnumJSONAndString(t *testing.T) {
	styles := []struct {
		style NotationStyle
		text  string
	}{
		{NotationSAN, "san"},
		{NotationFAN, "fan"},
		{NotationLAN, "lan"},
		{NotationDescriptive, "descriptive"},
		{NotationUCI, "uci"},
		{NotationICCF, "iccf"},
	}

	for _, tc := range styles {
		assert.Equal(t, tc.text, tc.style.String())
		assert.Equal(t, tc.style, ParseNotationStyle(strings.ToUpper(tc.text)))

		raw, err := json.Marshal(tc.style)
		require.NoError(t, err)
		assert.Equal(t, `"`+tc.text+`"`, string(raw))

		var decoded NotationStyle
		require.NoError(t, json.Unmarshal(raw, &decoded))
		assert.Equal(t, tc.style, decoded)
	}

	assert.Equal(t, "", NotationStyle(99).String())
	assert.Equal(t, NotationInvalid, ParseNotationStyle("bogus"))

	var bad NotationStyle
	assert.Error(t, json.Unmarshal([]byte(`123`), &bad))
}

func TestParsePGNDatabaseParallel(t *testing.T) {
	var sb strings.Builder
	for i := 0; i < 24; i++ {
		sb.WriteString("[Event \"Batch\"]\n[White \"W\"]\n[Black \"B\"]\n[Result \"1-0\"]\n\n1. e4 e5 2. Nf3 Nc6 1-0\n\n")
	}

	games, err := ParsePGNDatabase(strings.NewReader(sb.String()))
	require.NoError(t, err)
	require.Len(t, games, 24)
	for _, g := range games {
		assert.Equal(t, 4, g.PlyCount)
		assert.Equal(t, ResultWhiteWins, g.Result)
	}
}
