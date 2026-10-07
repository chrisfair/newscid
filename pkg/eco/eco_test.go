package eco_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"newscid/pkg/eco"
)

func TestECOClassificationDefaults(t *testing.T) {
	c := eco.NewClassifier()

	// 1. e4 c5 -> Sicilian Defense
	entry1 := c.Classify([]string{"e4", "c5"})
	assert.Equal(t, "B20", entry1.Code)
	assert.Equal(t, "Sicilian Defense", entry1.Name)

	// 1. e4 c5 2. Nf3 d6 3. d4 cxd4 4. Nxd4 Nf6 5. Nc3 a6 -> Najdorf
	entry2 := c.Classify([]string{"e4", "c5", "Nf3", "d6", "d4", "cxd4", "Nxd4", "Nf6", "Nc3", "a6"})
	assert.Equal(t, "B90", entry2.Code)
	assert.Equal(t, "Sicilian, Najdorf", entry2.Name)

	// 1. d4 Nf6 2. c4 g6 -> King's Indian
	entry3 := c.Classify([]string{"d4", "Nf6", "c4", "g6"})
	assert.Equal(t, "E60", entry3.Code)
	assert.Equal(t, "King's Indian Defense", entry3.Name)
}

func TestLoadScidECOFile(t *testing.T) {
	ecoPath := "/usr/share/scid/data/scid.eco"
	if _, err := os.Stat(ecoPath); os.IsNotExist(err) {
		t.Skip("scid.eco not on system, skipping")
	}

	c := eco.NewClassifier()
	err := c.LoadFromFile(ecoPath)
	require.NoError(t, err)

	entry := c.Classify([]string{"e4", "e5", "Nf3", "Nc6", "Bb5"})
	assert.NotEmpty(t, entry.Code)
	assert.NotEmpty(t, entry.Name)
}
