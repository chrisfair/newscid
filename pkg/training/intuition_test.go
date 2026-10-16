package training

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
