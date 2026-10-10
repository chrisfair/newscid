package scid

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCleanPlayerName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"GM Garry Kasparov", "Garry Kasparov"},
		{"Magnus Carlsen, GM", "Magnus Carlsen"},
		{"Anand, Viswanathan, IM", "Anand, Viswanathan"},
		{"  Karpov ,   Anatoly  ", "Karpov, Anatoly"},
		{"Robert  Fischer", "Robert Fischer"},
	}

	for _, tc := range tests {
		actual := CleanPlayerName(tc.input)
		assert.Equal(t, tc.expected, actual, "for input: %s", tc.input)
	}
}

func TestSpellChecker_StandardizeName(t *testing.T) {
	sc := NewSpellChecker()

	tests := []struct {
		input    string
		expected string
	}{
		{"Kasparov, Garry", "Kasparov, Garry"},
		{"Kasparov, G.", "Kasparov, Garry"},
		{"Garry Kasparov", "Kasparov, Garry"},
		{"GM Garry Kasparov", "Kasparov, Garry"},
		{"Carlsen, M.", "Carlsen, Magnus"},
		{"Magnus Carlsen", "Carlsen, Magnus"},
		{"Fischer, Bobby", "Fischer, Robert J"},
		{"Hikaru Nakamura", "Nakamura, Hikaru"},
		{"John Doe", "Doe, John"}, // Invert First Last
		{"Smith, Alice", "Smith, Alice"},
	}

	for _, tc := range tests {
		res := sc.StandardizeName(tc.input)
		assert.Equal(t, tc.expected, res, "for input: %s", tc.input)
	}
}

func TestSpellChecker_LoadDictionary(t *testing.T) {
	sc := NewSpellChecker()

	sspData := `# Custom master spelling dictionary
Polgar, Judit / Polgar, J. / Judit Polgar
Short, Nigel / Short, N. / Nigel Short
`
	err := sc.LoadDictionary(strings.NewReader(sspData))
	require.NoError(t, err)

	assert.Equal(t, "Polgar, Judit", sc.StandardizeName("Polgar, J."))
	assert.Equal(t, "Polgar, Judit", sc.StandardizeName("Judit Polgar"))
	assert.Equal(t, "Short, Nigel", sc.StandardizeName("Short, N."))
}

func TestSpellChecker_FindSuggestions(t *testing.T) {
	sc := NewSpellChecker()

	suggestions := sc.FindSuggestions("Kasparv, Gary", 3)
	require.NotEmpty(t, suggestions)
	assert.Contains(t, suggestions, "Kasparov, Garry")

	carlsenSug := sc.FindSuggestions("Carlson, Magnus", 3)
	require.NotEmpty(t, carlsenSug)
	assert.Contains(t, carlsenSug, "Carlsen, Magnus")
}

func TestSpellChecker_StandardizeDatabase(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "spell_db")
	db := NewDatabase()
	err := db.Create(dbPath, "Spell Test Base")
	require.NoError(t, err)
	defer db.Close()

	// Add games with non-standard names
	w1 := db.Names.AddName(NamePlayer, "Kasparov, G.")
	b1 := db.Names.AddName(NamePlayer, "Magnus Carlsen")

	w2 := db.Names.AddName(NamePlayer, "Carlsen, Magnus") // already standard
	b2 := db.Names.AddName(NamePlayer, "Karpov, A.")

	db.Entries = append(db.Entries, &IndexEntry{
		WhiteID: w1,
		BlackID: b1,
		Date:    "2000.01.01",
	})
	db.Entries = append(db.Entries, &IndexEntry{
		WhiteID: w2,
		BlackID: b2,
		Date:    "2010.01.01",
	})

	sc := NewSpellChecker()
	summary, err := sc.StandardizeDatabase(db)
	require.NoError(t, err)

	assert.Equal(t, 3, summary.CorrectionsMade) // Kasparov, G. -> Kasparov, Garry; Magnus Carlsen -> Carlsen, Magnus; Karpov, A. -> Karpov, Anatoly
	assert.Equal(t, 2, summary.GamesUpdated)

	// Verify entries remapped
	carlsenID, ok := db.Names.GetID(NamePlayer, "Carlsen, Magnus")
	require.True(t, ok)
	assert.Equal(t, carlsenID, db.Entries[0].BlackID)
	assert.Equal(t, carlsenID, db.Entries[1].WhiteID)

	kasparovID, ok := db.Names.GetID(NamePlayer, "Kasparov, Garry")
	require.True(t, ok)
	assert.Equal(t, kasparovID, db.Entries[0].WhiteID)
}
