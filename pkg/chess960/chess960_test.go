package chess960

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGeneratePosition_Classic(t *testing.T) {
	// Standard chess is position 518
	pos, err := GeneratePosition(518)
	require.NoError(t, err)
	assert.Equal(t, 518, pos.ID)
	assert.Equal(t, "RNBQKBNR", pos.Pieces)
	assert.True(t, pos.IsClassic)
	assert.Equal(t, "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1", pos.FEN)
}

func TestGeneratePosition_All960(t *testing.T) {
	seenPieces := make(map[string]int)

	for id := 0; id < 960; id++ {
		pos, err := GeneratePosition(id)
		require.NoError(t, err, "generating position %d", id)
		require.NotNil(t, pos)
		assert.Equal(t, id, pos.ID)

		p := pos.Pieces
		require.Len(t, p, 8)

		// 1. Verify exact piece counts
		assert.Equal(t, 2, strings.Count(p, "R"), "id %d should have 2 rooks", id)
		assert.Equal(t, 2, strings.Count(p, "N"), "id %d should have 2 knights", id)
		assert.Equal(t, 2, strings.Count(p, "B"), "id %d should have 2 bishops", id)
		assert.Equal(t, 1, strings.Count(p, "Q"), "id %d should have 1 queen", id)
		assert.Equal(t, 1, strings.Count(p, "K"), "id %d should have 1 king", id)

		// 2. Bishops must be on opposite colored squares (one on odd index, one on even index)
		b1 := strings.Index(p, "B")
		b2 := strings.LastIndex(p, "B")
		assert.NotEqual(t, b1%2, b2%2, "id %d bishops must be on opposite colors", id)

		// 3. King must be between the two Rooks
		r1 := strings.Index(p, "R")
		r2 := strings.LastIndex(p, "R")
		k := strings.Index(p, "K")
		assert.True(t, r1 < k && k < r2, "id %d King must be between the two Rooks (r1=%d, k=%d, r2=%d)", id, r1, k, r2)

		// 4. Verify uniqueness of all 960 positions
		seenPieces[p]++
	}

	assert.Len(t, seenPieces, 960, "all 960 positions must be uniquely generated")
}

func TestGeneratePosition_Bounds(t *testing.T) {
	_, err := GeneratePosition(-1)
	assert.Error(t, err)

	_, err = GeneratePosition(960)
	assert.Error(t, err)
}

func TestRandomPosition(t *testing.T) {
	pos := RandomPosition()
	require.NotNil(t, pos)
	assert.GreaterOrEqual(t, pos.ID, 0)
	assert.Less(t, pos.ID, 960)
	assert.NotEmpty(t, pos.FEN)
}
