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

func TestGetProphylaxisDrillBatch(t *testing.T) {
	batch := GetProphylaxisDrillBatch(5, "all")
	assert.NotEmpty(t, batch)
	assert.True(t, len(batch) <= 5)

	// Filter by king safety
	kingBatch := GetProphylaxisDrillBatch(10, string(CategoryKingSafety))
	assert.NotEmpty(t, kingBatch)
	for _, d := range kingBatch {
		assert.Equal(t, CategoryKingSafety, d.Category)
	}
}

func TestSubmitProphylaxisAnswer_Correct(t *testing.T) {
	drill, ok := GetProphylaxisDrill("pro-01-karpov-kh1")
	require.True(t, ok)

	res, err := SubmitProphylaxisAnswer(drill.ID, "g1h1")
	require.NoError(t, err)
	assert.True(t, res.IsCorrect)
	assert.Equal(t, 150, res.PointsAwarded)
	assert.Equal(t, "Kh1", res.BestMoveSAN)
	assert.NotEmpty(t, res.Explanation)
}

func TestSubmitProphylaxisAnswer_Wrong(t *testing.T) {
	drill, ok := GetProphylaxisDrill("pro-01-karpov-kh1")
	require.True(t, ok)

	res, err := SubmitProphylaxisAnswer(drill.ID, "a2a4")
	require.NoError(t, err)
	assert.False(t, res.IsCorrect)
	assert.Equal(t, 0, res.PointsAwarded)
	assert.Equal(t, "Kh1", res.BestMoveSAN)
}

func TestSubmitProphylaxisAnswer_NotFound(t *testing.T) {
	_, err := SubmitProphylaxisAnswer("non-existent-drill", "e2e4")
	assert.Error(t, err)
}

func TestGenerateProphylaxisDrillsFromDatabase(t *testing.T) {
	// 1. Nil DB
	_, err := GenerateProphylaxisDrillsFromDatabase(nil, 5)
	assert.Error(t, err)

	// 2. Open DB with games
	tmpDir, err := os.MkdirTemp("", "scid_pro_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "probase")
	db := scid.NewDatabase()
	err = db.Create(dbPath, "Prophylaxis Base")
	require.NoError(t, err)

	g := chess.NewGame()
	g.SetTag("White", "Karpov, Anatoly")
	g.SetTag("Black", "Spassky, Boris")
	g.SetTag("Event", "Candidates 1974")

	b := chess.NewBoard()
	var root *chess.MoveNode
	var prev *chess.MoveNode
	for i := 0; i < 20; i++ {
		san := "Nf3"
		if i%2 == 1 {
			san = "Nc6"
		}
		if i == 16 {
			san = "a3"
		}
		m, err := b.ParseSAN(san)
		if err != nil {
			// Fallback to legal moves
			legal := b.GenerateLegalMoves()
			if len(legal) == 0 {
				break
			}
			m = legal[0]
			san = b.MoveToSAN(m)
		}
		b.MakeMove(m)
		node := &chess.MoveNode{Move: m, UCI: m.String(), SAN: san}
		if root == nil {
			root = node
		} else {
			prev.Next = node
		}
		prev = node
	}
	g.RootMove = root
	g.PlyCount = 20

	_, err = db.SaveGame(g)
	require.NoError(t, err)

	drills, err := GenerateProphylaxisDrillsFromDatabase(db, 3)
	require.NoError(t, err)
	assert.NotEmpty(t, drills)

	// Alternative UCI answer ("pro-03-nimzowitsch-blockade" accepts "a2a3")
	resAlt, err := SubmitProphylaxisAnswer("pro-03-nimzowitsch-blockade", "a2a3")
	require.NoError(t, err)
	assert.True(t, resAlt.IsCorrect)
	assert.Equal(t, 150, resAlt.PointsAwarded)

	// 3. Empty DB falls back to batch
	emptyDir := t.TempDir()
	emptyDB := scid.NewDatabase()
	err = emptyDB.Create(filepath.Join(emptyDir, "pro_empty"), "Pro Empty")
	require.NoError(t, err)

	fallbackDrills, err := GenerateProphylaxisDrillsFromDatabase(emptyDB, 2)
	require.NoError(t, err)
	assert.NotEmpty(t, fallbackDrills)

	// 4. DB with quiet king move (Kh8 by Black at ply >= 14) and custom StartFEN
	kingDB := scid.NewDatabase()
	require.NoError(t, kingDB.Create(filepath.Join(t.TempDir(), "pro_king"), "Pro King"))
	gKing, err := chess.ParsePGN("1. e4 e5 2. Nf3 Nc6 3. Bc4 Bc5 4. O-O Nf6 5. d3 O-O 6. c3 d6 7. b4 Bb6 8. a4 a6 9. Kh1 Kh8 1-0")
	require.NoError(t, err)
	gKing.StartFEN = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 1 1"
	_, err = kingDB.SaveGame(gKing)
	require.NoError(t, err)

	kingDrills, err := GenerateProphylaxisDrillsFromDatabase(kingDB, 5)
	require.NoError(t, err)
	assert.NotEmpty(t, kingDrills)

	// 5. DB with short game (<14 plies) that yields 0 drills -> fallback to batch
	shortDB := scid.NewDatabase()
	require.NoError(t, shortDB.Create(filepath.Join(t.TempDir(), "pro_short"), "Pro Short"))
	gShort, _ := chess.ParsePGN("1. e4 e5 1-0")
	_, _ = shortDB.SaveGame(gShort)
	fbShort, err := GenerateProphylaxisDrillsFromDatabase(shortDB, 2)
	require.NoError(t, err)
	assert.NotEmpty(t, fbShort)
}

