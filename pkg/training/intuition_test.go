package training

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"newscid/pkg/chess"
	"newscid/pkg/database/scid"
)

func TestGetIntuitionDrillBatch(t *testing.T) {
	batch := GetIntuitionDrillBatch(5)
	assert.NotEmpty(t, batch)
	assert.LessOrEqual(t, len(batch), 5)

	for _, ch := range batch {
		assert.NotEmpty(t, ch.ID)
		assert.NotEmpty(t, ch.FEN)
		assert.NotEmpty(t, ch.Category)
		assert.NotEmpty(t, ch.KeyFeature)
	}
}

func TestSubmitIntuitionAnswer_Correct(t *testing.T) {
	ch, found := GetIntuitionChallenge("int-3") // Capablanca - Tartakower (White Winning)
	require.True(t, found)
	assert.Equal(t, IntuitionWhiteWinning, ch.Category)

	res, err := SubmitIntuitionAnswer("int-3", "white_winning", 1500)
	require.NoError(t, err)
	assert.True(t, res.IsCorrect)
	assert.Equal(t, IntuitionWhiteWinning, res.ActualCategory)
	assert.Greater(t, res.PointsAwarded, 100) // Base 100 + speed bonus
	assert.Greater(t, res.SpeedBonus, 0)
	assert.Contains(t, res.Explanation, "Spot on!")
}

func TestSubmitIntuitionAnswer_Wrong(t *testing.T) {
	res, err := SubmitIntuitionAnswer("int-3", "black_winning", 2000)
	require.NoError(t, err)
	assert.False(t, res.IsCorrect)
	assert.Equal(t, 0, res.PointsAwarded)
	assert.Equal(t, 0, res.SpeedBonus)
	assert.Contains(t, res.Explanation, "Misjudgment")
}

func TestSubmitIntuitionAnswer_Equal(t *testing.T) {
	res, err := SubmitIntuitionAnswer("int-1", "equal", 2200)
	require.NoError(t, err)
	assert.True(t, res.IsCorrect)
	assert.Equal(t, IntuitionEqual, res.ActualCategory)
}

func TestSubmitIntuitionAnswer_NotFound(t *testing.T) {
	_, err := SubmitIntuitionAnswer("non-existent-id", "equal", 1000)
	assert.Error(t, err)
}

func TestGenerateIntuitionDrillsFromDatabase(t *testing.T) {
	// 1. Nil or closed DB
	_, err := GenerateIntuitionDrillsFromDatabase(nil, 5)
	assert.Error(t, err)

	// 2. Open DB with games
	tmpDir, err := os.MkdirTemp("", "scid_int_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "intbase")
	db := scid.NewDatabase()
	err = db.Create(dbPath, "Intuition Base")
	require.NoError(t, err)

	g := chess.NewGame()
	g.SetTag("White", "Karpov, Anatoly")
	g.SetTag("Black", "Kasparov, Garry")
	g.SetTag("Event", "World Championship")

	b := chess.NewBoard()
	m1, _ := b.ParseSAN("e4"); b.MakeMove(m1)
	m2, _ := b.ParseSAN("e5"); b.MakeMove(m2)
	n1 := &chess.MoveNode{Move: m1, UCI: m1.String(), SAN: "e4"}
	n2 := &chess.MoveNode{Move: m2, UCI: m2.String(), SAN: "e5"}
	n1.Next = n2
	g.RootMove = n1
	g.PlyCount = 2

	_, err = db.SaveGame(g)
	require.NoError(t, err)

	drills, err := GenerateIntuitionDrillsFromDatabase(db, 3)
	require.NoError(t, err)
	assert.NotEmpty(t, drills)

	// Test batch
	b1 := GetIntuitionDrillBatch(2)
	assert.NotEmpty(t, b1)
	assert.LessOrEqual(t, len(b1), 2)
	bAll := GetIntuitionDrillBatch(0)
	assert.NotEmpty(t, bAll)

	// Test empty DB
	emptyDir := t.TempDir()
	emptyDB := scid.NewDatabase()
	err = emptyDB.Create(filepath.Join(emptyDir, "empty"), "Empty")
	require.NoError(t, err)
	_, err = GenerateIntuitionDrillsFromDatabase(emptyDB, 5)
	assert.Error(t, err)
}

