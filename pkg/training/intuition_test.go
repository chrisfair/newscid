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
	assert.Contains(t, res.Explanation, "Correct.")
}

func TestSubmitIntuitionAnswer_Wrong(t *testing.T) {
	res, err := SubmitIntuitionAnswer("int-3", "black_winning", 2000)
	require.NoError(t, err)
	assert.False(t, res.IsCorrect)
	assert.Equal(t, 0, res.PointsAwarded)
	assert.Equal(t, 0, res.SpeedBonus)
	assert.Contains(t, res.Explanation, "Incorrect.")
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
	// Nil or closed DB
	_, err := GenerateIntuitionDrillsFromDatabase(nil, 5)
	assert.Error(t, err)

	// Open DB with games
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

	// Test negative and 0 EvaluationCp & wrong answers for BlackWinning and Equal
	resWrongBlack, err := SubmitIntuitionAnswer("int-6", "white_winning", 1000)
	require.NoError(t, err)
	assert.False(t, resWrongBlack.IsCorrect)
	assert.Contains(t, resWrongBlack.Explanation, "Black is Winning")

	resWrongEq, err := SubmitIntuitionAnswer("int-1", "white_winning", 1000)
	require.NoError(t, err)
	assert.False(t, resWrongEq.IsCorrect)
	assert.Contains(t, resWrongEq.Explanation, "Equal")

	RegisterIntuitionChallenge(IntuitionChallenge{
		ID:           "int-zero",
		FEN:          chess.StartingFEN,
		EvaluationCp: 0,
		Category:     IntuitionEqual,
	})
	resZero, err := SubmitIntuitionAnswer("int-zero", "equal", 500)
	require.NoError(t, err)
	assert.Equal(t, "0.00", resZero.ScoreFormatted)

	// Add 40-ply games to db: equal, White material up, Black material up
	gEqual, err := chess.ParsePGN("1. e4 e5 2. Nf3 Nc6 3. Bc4 Bc5 4. c3 Nf6 5. d3 d6 6. O-O O-O 7. Re1 Re8 8. Nbd2 a6 9. Nf1 Ba7 10. Ng3 h6 11. h3 Be6 12. Bb3 Bxb3 13. Qxb3 Qd7 14. Be3 Bxe3 15. Rxe3 Rab8 16. Rd1 Qe6 17. Qc2 Rbd8 18. b3 d5 19. Rde1 dxe4 20. dxe4 Rd7 1-0")
	require.NoError(t, err)
	gEqual.StartFEN = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 1 1"
	_, err = db.SaveGame(gEqual)
	require.NoError(t, err)

	// Game where White wins Black's Queen early and plays 20 moves
	gWhiteUp, err := chess.ParsePGN("1. e4 d5 2. exd5 Qxd5 3. Nc3 Qd8 4. Bc4 Nf6 5. Nf3 Bg4 6. Bxf7+ Kxf7 7. Ne5+ Kg8 8. Nxg4 Nxg4 9. Qxg4 Nc6 10. Qe6# 1-0")
	require.NoError(t, err)
	// Also a longer game where White is up a Queen
	gWhiteQueenUp, err := chess.ParsePGN("1. e4 e5 2. Nf3 d5 3. Nxe5 dxe4 4. Bc4 Qd4 5. Bxf7+ Kd8 6. Bxg8 Rxg8 7. Nf7+ Ke8 8. Ng5 Qd5 9. d3 exd3 10. Qxd3 Qxg2 11. Qe4+ Qxe4+ 12. Nxe4 Bf5 13. Nbc3 Bb4 14. Bd2 Bxc3 15. Nxc3 Bxc2 16. O-O Nc6 17. Rfe1+ Kf7 18. Rac1 Bg6 19. Nd5 Rgd8 20. Nxc7 Rxd2 1-0")
	require.NoError(t, err)
	_, _ = db.SaveGame(gWhiteUp)
	_, _ = db.SaveGame(gWhiteQueenUp)

	// Game where White blunders Queen early and plays 20 moves (Black winning >= 250 cp)
	gBlackUp, err := chess.ParsePGN("1. e4 e5 2. Qh5 Nc6 3. Bc4 g6 4. Qxg6 hxg6 5. Nf3 Nf6 6. Nc3 Bc5 7. d3 d6 8. Bg5 Be6 9. Bxe6 fxe6 10. O-O O-O 11. Na4 Bb6 12. Nxb6 axb6 13. c3 Qd7 14. Rfe1 Nh7 15. Be3 Rf7 16. d4 exd4 17. Nxd4 Nxd4 18. Bxd4 e5 19. Be3 Raf8 20. f3 Qe6 0-1")
	require.NoError(t, err)
	for i := 0; i < 65; i++ {
		_, _ = db.SaveGame(gBlackUp)
	}

	drills, err := GenerateIntuitionDrillsFromDatabase(db, 100)
	require.NoError(t, err)
	assert.NotEmpty(t, drills)

	// DB with only short (<10 ply) game triggers len(challenges) == 0 fallback
	shortDB := scid.NewDatabase()
	require.NoError(t, shortDB.Create(filepath.Join(t.TempDir(), "short_int"), "Short Int"))
	_, _ = shortDB.SaveGame(g)
	fbDrills, err := GenerateIntuitionDrillsFromDatabase(shortDB, 3)
	require.NoError(t, err)
	assert.Len(t, fbDrills, 3)

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

