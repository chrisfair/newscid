package online

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDatabaseTargetModeEnumJSONAndString(t *testing.T) {
	cases := []struct {
		mode DatabaseTargetMode
		text string
	}{
		{TargetModeUnset, ""},
		{TargetActiveBase, "active"},
		{TargetDedicatedBase, "dedicated"},
		{TargetBlendedBase, "blended"},
	}

	for _, tc := range cases {
		assert.Equal(t, tc.text, tc.mode.String())
		assert.Equal(t, tc.mode, ParseDatabaseTargetMode(tc.text))

		raw, err := json.Marshal(tc.mode)
		require.NoError(t, err)
		assert.Equal(t, `"`+tc.text+`"`, string(raw))

		var decoded DatabaseTargetMode
		require.NoError(t, json.Unmarshal(raw, &decoded))
		assert.Equal(t, tc.mode, decoded)
	}

	assert.Equal(t, "", DatabaseTargetMode(99).String())
	assert.Equal(t, TargetModeUnset, ParseDatabaseTargetMode("unknown"))

	var bad DatabaseTargetMode
	assert.Error(t, json.Unmarshal([]byte(`123`), &bad))
}
