package stats

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"newscid/pkg/chess"
	"newscid/pkg/database/scid"
)

func TestParseClockToSeconds(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
		hasErr   bool
	}{
		{"1:30:00", 5400, false},
		{"0:05:22", 322, false},
		{"5:30", 330, false},
		{"0:45", 45, false},
		{"45s", 45, false},
		{"12.5", 12.5, false},
		{"", 0, true},
		{"invalid", 0, true},
	}

	for _, tc := range tests {
		res, err := ParseClockToSeconds(tc.input)
		if tc.hasErr {
			assert.Error(t, err, "expected error for %q", tc.input)
		} else {
			assert.NoError(t, err, "unexpected error for %q", tc.input)
			assert.Equal(t, tc.expected, res)
		}
	}
}

func TestParseEvalToCentipawns(t *testing.T) {
	val, ok := ParseEvalToCentipawns("+1.25")
	assert.True(t, ok)
	assert.Equal(t, 125, val)

	val, ok = ParseEvalToCentipawns("-0.80")
	assert.True(t, ok)
	assert.Equal(t, -80, val)

	val, ok = ParseEvalToCentipawns("#2")
	assert.True(t, ok)
	assert.Equal(t, 9999, val)

	val, ok = ParseEvalToCentipawns("-#1")
	assert.True(t, ok)
	assert.Equal(t, -9999, val)

	_, ok = ParseEvalToCentipawns("not_an_eval")
	assert.False(t, ok)
}

func TestAnalyzeGameTimeManagement(t *testing.T) {
	// Build a synthetic game with clock and evaluation comments
	g := chess.NewGame()
	g.ID = 42
	g.Tags["White"] = "Carlsen, Magnus"
	g.Tags["Black"] = "Nakamura, Hikaru"
	g.Tags["Event"] = "Speed Chess Championship 2024"
	g.Tags["TimeControl"] = "300+0"

	// 1. e4 {[%clk 0:05:00] [%eval 0.20]}
	// 1... c5 {[%clk 0:05:00] [%eval 0.25]}
	// 2. Nf3 {[%clk 0:04:58] [%eval 0.22]}
	// 2... d6 {[%clk 0:04:55] [%eval 0.30]}
	// 3. d4 {[%clk 0:02:40] [%eval 0.35]}  <- White long tank of 138 seconds!
	// 3... cxd4 {[%clk 0:04:54] [%eval 0.35]}
	// 4. Nxd4 {[%clk 0:02:38] [%eval 0.35]}
	// 4... Nf6 {[%clk 0:04:50] [%eval 0.35]}

	n1 := &chess.MoveNode{
		SAN:     "e4",
		UCI:     "e2e4",
		Clock:   "0:05:00",
		Comment: "[%eval 0.20]",
	}
	n2 := &chess.MoveNode{
		SAN:     "c5",
		UCI:     "c7c5",
		Clock:   "0:05:00",
		Comment: "[%eval 0.25]",
	}
	n3 := &chess.MoveNode{
		SAN:     "Nf3",
		UCI:     "g1f3",
		Clock:   "0:04:58",
		Comment: "[%eval 0.22]",
	}
	n4 := &chess.MoveNode{
		SAN:     "d6",
		UCI:     "d7d6",
		Clock:   "0:04:55",
		Comment: "[%eval 0.30]",
	}
	n5 := &chess.MoveNode{
		SAN:     "d4",
		UCI:     "d2d4",
		Clock:   "0:02:40", // 138s tank from 0:04:58
		Comment: "[%eval 0.35]",
	}
	n6 := &chess.MoveNode{
		SAN:     "cxd4",
		UCI:     "c5d4",
		Clock:   "0:04:54",
		Comment: "[%eval 0.35]",
	}
	n7 := &chess.MoveNode{
		SAN:     "Nxd4",
		UCI:     "f3d4",
		Clock:   "0:02:38",
		Comment: "[%eval 0.35]",
	}
	n8 := &chess.MoveNode{
		SAN:     "Nf6",
		UCI:     "g8f6",
		Clock:   "0:00:20", // Black enters scramble: 20s left!
		Comment: "[%eval 0.40]",
	}

	n1.Next = n2
	n2.Next = n3
	n3.Next = n4
	n4.Next = n5
	n5.Next = n6
	n6.Next = n7
	n7.Next = n8
	g.RootMove = n1

	report, err := AnalyzeGameTimeManagement(g)
	require.NoError(t, err)
	require.NotNil(t, report)

	assert.Equal(t, 42, report.GameID)
	assert.Equal(t, "Carlsen, Magnus", report.WhitePlayer)
	assert.Equal(t, "Nakamura, Hikaru", report.BlackPlayer)
	assert.True(t, report.WhiteHasClock)
	assert.True(t, report.BlackHasClock)
	assert.Equal(t, 8, len(report.Moves))

	// White move 3 (ply 5) was a long tank
	assert.Equal(t, 1, report.WhiteStats.TankCount)
	assert.Equal(t, 138.0, report.WhiteStats.LongestTankSec)
	assert.Equal(t, 5, report.WhiteStats.LongestTankPly)
	assert.True(t, len(report.TankDossier) >= 1)
	assert.Equal(t, "d4", report.TankDossier[0].SAN)
	assert.Equal(t, "Masterstroke", report.TankDossier[0].Outcome)

	// Black move 4 (ply 8) was in scramble (<= 30s)
	assert.Equal(t, 8, report.ScrambleSummary.BlackEnteredScramblePly)
	assert.Equal(t, 1, report.ScrambleSummary.BlackScrambleMoves)
	assert.Equal(t, 1, report.BlackStats.TimeTroubleMoves)

	// Check recommendations generated
	assert.NotEmpty(t, report.Recommendations)
}

func TestAnalyzeGameTimeManagement_Nil(t *testing.T) {
	_, err := AnalyzeGameTimeManagement(nil)
	assert.Error(t, err)
}

func TestCalculatePearson(t *testing.T) {
	// Perfect positive correlation
	pairs := [][2]float64{
		{1, 1},
		{2, 2},
		{3, 3},
		{4, 4},
		{5, 5},
	}
	r := calculatePearson(pairs)
	assert.InDelta(t, 1.0, r, 0.001)

	// Flat slope
	pairsFlat := [][2]float64{
		{1, 5},
		{2, 5},
		{3, 5},
	}
	rFlat := calculatePearson(pairsFlat)
	assert.Equal(t, 0.0, rFlat)
}

func TestAnalyzePlayerTimeManagement(t *testing.T) {
	// 1. Closed or nil DB
	_, err := AnalyzePlayerTimeManagement(nil, "Carlsen", 50)
	assert.Error(t, err)

	// 2. Open DB with clock games
	tmpDir, err := os.MkdirTemp("", "scid_time_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "timebase")
	db := scid.NewDatabase()
	err = db.Create(dbPath, "Time Base")
	require.NoError(t, err)

	g := chess.NewGame()
	g.SetTag("White", "Carlsen, Magnus")
	g.SetTag("Black", "Nakamura, Hikaru")
	g.SetTag("Event", "World Blitz")

	b := chess.NewBoard()
	m1, _ := b.ParseSAN("e4"); b.MakeMove(m1)
	m2, _ := b.ParseSAN("e5"); b.MakeMove(m2)
	m3, _ := b.ParseSAN("Nf3"); b.MakeMove(m3)
	m4, _ := b.ParseSAN("Nc6"); b.MakeMove(m4)

	n1 := &chess.MoveNode{Move: m1, SAN: "e4", UCI: m1.String(), Clock: "0:05:00"}
	n2 := &chess.MoveNode{Move: m2, SAN: "e5", UCI: m2.String(), Clock: "0:04:58"}
	n3 := &chess.MoveNode{Move: m3, SAN: "Nf3", UCI: m3.String(), Clock: "0:02:45"} // 135s tank
	n4 := &chess.MoveNode{Move: m4, SAN: "Nc6", UCI: m4.String(), Clock: "0:04:50"}
	n1.Next = n2
	n2.Next = n3
	n3.Next = n4
	g.RootMove = n1
	g.PlyCount = 4

	_, err = db.SaveGame(g)
	require.NoError(t, err)

	dossier, err := AnalyzePlayerTimeManagement(db, "Carlsen", 50)
	require.NoError(t, err)
	assert.Equal(t, "Carlsen", dossier.PlayerName)
	assert.Equal(t, 1, dossier.GamesWithClock)
	assert.True(t, dossier.TotalTanks >= 1)
	assert.NotEmpty(t, dossier.CoachingDossier)

	// Analyze Black player
	blackDossier, err := AnalyzePlayerTimeManagement(db, "Nakamura", 0)
	require.NoError(t, err)
	assert.Equal(t, "Nakamura", blackDossier.PlayerName)
	assert.Equal(t, 1, blackDossier.GamesWithClock)
}

func TestFormatSeconds(t *testing.T) {
	assert.Equal(t, "45.0s", formatSeconds(45))
	assert.Equal(t, "5m 30s", formatSeconds(330))
	assert.Equal(t, "75m 00s", formatSeconds(4500))
}

func TestGenerateRecommendationsAndScrambleAlerts(t *testing.T) {
	rep := &GameTimeManagementReport{
		WhiteStats: SideTimeStats{
			TankBlunderCount: 2,
			PanicMoveCount:   1,
		},
		BlackStats: SideTimeStats{
			TankBlunderCount: 1,
			PanicMoveCount:   2,
		},
		ScrambleSummary: ScrambleSummary{
			WhiteEnteredScramblePly: 35,
			WhiteScrambleMoves:      10,
			WhiteScrambleBlunders:   2,
			BlackEnteredScramblePly: 36,
			BlackScrambleMoves:      8,
			BlackScrambleBlunders:   1,
		},
	}
	generateRecommendations(rep)
	assert.NotEmpty(t, rep.Recommendations)
	assert.Len(t, rep.Recommendations, 6)

	// Empty clean report
	cleanRep := &GameTimeManagementReport{}
	generateRecommendations(cleanRep)
	assert.Len(t, cleanRep.Recommendations, 1)
	assert.Contains(t, cleanRep.Recommendations[0], "Clean clock discipline")
}

func TestPanicAndScrambleDetection(t *testing.T) {
	g := chess.NewGame()
	b := chess.NewBoard()
	var root, prev *chess.MoveNode

	// Generate 20 plies so ply > 16
	for i := 1; i <= 20; i++ {
		legal := b.GenerateLegalMoves()
		require.NotEmpty(t, legal)
		m := legal[0]
		b.MakeMove(m)
		clk := "0:05:00"
		comment := ""
		if i == 18 { // Panic move: blitzed in 2s with severe blunder (?? NAG 4)
			clk = "0:04:58"
			comment = "[%eval -4.50]"
		} else if i == 19 { // Scramble move: clock <= 30s with blunder
			clk = "0:00:20"
			comment = "[%eval -5.00]"
		}
		node := &chess.MoveNode{
			Move:    m,
			UCI:     m.String(),
			SAN:     b.MoveToSAN(m),
			Clock:   clk,
			Comment: comment,
		}
		if i == 18 {
			node.NAGs = []int{4} // Blunder ??
		}
		if root == nil {
			root = node
		} else {
			prev.Next = node
		}
		prev = node
	}
	g.RootMove = root
	g.PlyCount = 20

	rep, err := AnalyzeGameTimeManagement(g)
	require.NoError(t, err)
	assert.NotNil(t, rep)
}

func TestPlayerBehavioralHabits(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "habitbase")
	db := scid.NewDatabase()
	err := db.Create(dbPath, "Habit Base")
	require.NoError(t, err)

	g := chess.NewGame()
	g.SetTag("White", "HabitPlayer")
	g.SetTag("Black", "Opponent")

	b := chess.NewBoard()
	var root, prev *chess.MoveNode

	// Generate 30 plies with:
	// - Opening moves taking 30s (OpeningAvgSec > 25)
	// - 3 Tanks with blunders (TotalTanks >= 3, TankSuccessRate < 60)
	// - 3 Panic moves (PanicMoveCount >= 3)
	// - Scramble clock < 30s
	for i := 1; i <= 30; i++ {
		legal := b.GenerateLegalMoves()
		require.NotEmpty(t, legal)
		m := legal[0]
		b.MakeMove(m)

		clk := "0:05:00"
		comment := ""
		if i == 1 {
			clk = "0:04:30" // 30s think in opening
		} else if i == 3 {
			clk = "0:02:10" // 140s tank
			comment = "[%eval -3.00]"
		} else if i == 5 {
			clk = "0:00:25" // 105s tank into scramble!
			comment = "[%eval -4.00]"
		} else if i == 17 {
			clk = "0:00:24" // 1s move in scramble with blunder (panic)
			comment = "[%eval -5.00]"
		} else if i == 19 {
			clk = "0:00:22" // 2s move with blunder (panic)
			comment = "[%eval -6.00]"
		} else if i == 21 {
			clk = "0:00:20" // 2s move with blunder (panic)
			comment = "[%eval -7.00]"
		}

		node := &chess.MoveNode{
			Move:    m,
			UCI:     m.String(),
			SAN:     b.MoveToSAN(m),
			Clock:   clk,
			Comment: comment,
		}
		if i == 3 || i == 5 || i == 17 || i == 19 || i == 21 {
			node.NAGs = []int{4} // ??
		}
		if root == nil {
			root = node
		} else {
			prev.Next = node
		}
		prev = node
	}
	g.RootMove = root
	g.PlyCount = 30

	_, err = db.SaveGame(g)
	require.NoError(t, err)

	dossier, err := AnalyzePlayerTimeManagement(db, "HabitPlayer", 10)
	require.NoError(t, err)
	assert.NotEmpty(t, dossier.HabitFlags)
	assert.NotEmpty(t, dossier.CoachingDossier)
}

