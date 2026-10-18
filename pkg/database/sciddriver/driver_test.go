package sciddriver

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDriverAvailabilityAndExecution(t *testing.T) {
	// Test unavailable driver error paths
	unavail := &Driver{tcscidPath: ""}
	assert.False(t, unavail.IsAvailable())
	_, err := unavail.ExecuteTcl("puts {hi}")
	assert.Error(t, err)
	_, err = unavail.GetBaseGameCount("/nonexistent")
	assert.Error(t, err)

	// Test NewDriver with empty PATH fallback
	origPath := os.Getenv("PATH")
	_ = os.Setenv("PATH", "")
	_ = NewDriver()
	_ = os.Setenv("PATH", origPath)

	// Test command execution failure with false binary
	failDriver := &Driver{tcscidPath: "/bin/false"}
	_, err = failDriver.ExecuteTcl("puts {fail}")
	assert.Error(t, err)

	d := NewDriver()
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
