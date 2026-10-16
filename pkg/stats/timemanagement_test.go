package stats

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"newscid/pkg/chess"
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
