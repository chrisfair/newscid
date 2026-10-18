package epd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseEPD(t *testing.T) {
	sample := `1k1r4/pp1b1R2/1q1bp2p/3p4/8/1NBQ4/PPP3PP/7K w - - bm Bd4; id "Tactics-01"; ce 250; c0 "White wins";`

	rec, err := Parse(sample)
	require.NoError(t, err)
	require.NotNil(t, rec)

	assert.Equal(t, "1k1r4/pp1b1R2/1q1bp2p/3p4/8/1NBQ4/PPP3PP/7K w - -", rec.FENCore)
	assert.Equal(t, "1k1r4/pp1b1R2/1q1bp2p/3p4/8/1NBQ4/PPP3PP/7K w - - 0 1", rec.FullFEN)
	assert.Equal(t, "Bd4", rec.BestMove)
	assert.Equal(t, "Tactics-01", rec.ID)
	assert.Equal(t, "White wins", rec.Comment)
	require.NotNil(t, rec.EvalCP)
	assert.Equal(t, 250, *rec.EvalCP)
}

func TestParseEPD_Bare(t *testing.T) {
	bare := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq -"
	rec, err := Parse(bare)
	require.NoError(t, err)
	assert.Equal(t, bare, rec.FENCore)
	assert.Empty(t, rec.Opcodes)
}

func TestParseEPD_Errors(t *testing.T) {
	_, err := Parse("")
	assert.Error(t, err)

	_, err = Parse("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w")
	assert.Error(t, err)
}

func TestFormatEPD(t *testing.T) {
	fen := "1k1r4/pp1b1R2/1q1bp2p/3p4/8/1NBQ4/PPP3PP/7K w - - 0 1"
	opcodes := map[string]string{
		"bm": "Bd4",
		"id": "Tactics-01",
	}

	res := Format(fen, opcodes)
	assert.Contains(t, res, "1k1r4/pp1b1R2/1q1bp2p/3p4/8/1NBQ4/PPP3PP/7K w - -")
	assert.Contains(t, res, "bm Bd4;")
	assert.Contains(t, res, "id \"Tactics-01\";")

	// Empty opcodes & bare opcode without operand & am opcode
	assert.Equal(t, "1k1r4/pp1b1R2/1q1bp2p/3p4/8/1NBQ4/PPP3PP/7K w - -", Format(fen, nil))

	rec2, err := Parse(`rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - am h4; flagonly;`)
	require.NoError(t, err)
	assert.Equal(t, "h4", rec2.AvoidMove)
	assert.Equal(t, "", rec2.Opcodes["flagonly"])

	formatted2 := Format("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq -", map[string]string{
		"flagonly": "",
		"c0":       "comment text",
	})
	assert.Contains(t, formatted2, "flagonly;")
	assert.Contains(t, formatted2, `c0 "comment text";`)
}
