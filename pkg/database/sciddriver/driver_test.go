package sciddriver_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"newscid/pkg/database/sciddriver"
)

func TestDriverAvailabilityAndExecution(t *testing.T) {
	d := sciddriver.NewDriver()
	if !d.IsAvailable() {
		t.Skip("tcscid not installed, skipping driver test")
	}

	assert.True(t, d.IsAvailable())

	out, err := d.ExecuteTcl("puts {Hello SCID}")
	require.NoError(t, err)
	assert.Equal(t, "Hello SCID", out)

	tacticsPath := "/usr/share/scid/bases/tactics"
	if _, err := os.Stat(tacticsPath + ".si4"); err == nil {
		count, err := d.GetBaseGameCount(tacticsPath)
		require.NoError(t, err)
		assert.Equal(t, 928, count)
	}
}
