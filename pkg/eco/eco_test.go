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

	sections := c.GetSections()
	assert.Len(t, sections, 5)
	for _, s := range sections {
		assert.Greater(t, s.Count, 0)
	}

	// Search by query
	sicilians := c.Search("B", "Najdorf", 10)
	assert.NotEmpty(t, sicilians)
	assert.Equal(t, "B", string(sicilians[0].Code[0]))

	all := c.GetAllEntries()
	assert.Greater(t, len(all), 100)
}

func TestECOSearchAndSections(t *testing.T) {
	c := eco.NewClassifier()

	sections := c.GetSections()
	assert.Len(t, sections, 5)
	assert.Equal(t, "A", sections[0].Letter)
	assert.Equal(t, "B", sections[1].Letter)
	assert.Equal(t, "C", sections[2].Letter)
	assert.Equal(t, "D", sections[3].Letter)
	assert.Equal(t, "E", sections[4].Letter)

	// Search in section B
	bRes := c.Search("B", "Sicilian", 5)
	require.NotEmpty(t, bRes)
	assert.Contains(t, bRes[0].Name, "Sicilian")

	// Search all sections
	allSic := c.Search("ALL", "Sicilian", 0)
	assert.GreaterOrEqual(t, len(allSic), 2)

	// Search code prefix
	cRes := c.Search("", "B90", 0)
	require.Len(t, cRes, 1)
	assert.Equal(t, "B90", cRes[0].Code)

	allEntries := c.GetAllEntries()
	assert.NotEmpty(t, allEntries)

	// Unknown classification
	unknown := c.Classify([]string{"a4", "h5", "a5", "h4"})
	assert.Equal(t, "A00", unknown.Code)
}

func TestLoadFromFileEdgeCases(t *testing.T) {
	c := eco.NewClassifier()
	err := c.LoadFromFile("/nonexistent/path/scid.eco")
	assert.Error(t, err)

	tmpFile := t.TempDir() + "/custom.eco"
	content := `# Comment line
invalid_line_without_space
A01 NoQuotes 1. b3 *
A02 "UnclosedQuote 1. f4 *
A03 "Bird Opening" 1. f4 d5 *
`
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0644))
	require.NoError(t, c.LoadFromFile(tmpFile))

	res := c.Classify([]string{"f4", "d5"})
	assert.Equal(t, "A03", res.Code)
	assert.Equal(t, "Bird Opening", res.Name)
}
