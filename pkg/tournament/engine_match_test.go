package tournament

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"newscid/pkg/chess"
	"newscid/pkg/database/scid"
	"newscid/pkg/engine"
)

func TestMockEnginePlayer_Basics(t *testing.T) {
	p := NewMockEnginePlayer("Bot1", "e2e4", "e7e5")
	assert.Equal(t, "Bot1", p.GetName())

	ctx := context.Background()
	m1, _, err := p.SearchMove(ctx, chess.StartingFEN, nil, 100)
	require.NoError(t, err)
	assert.Equal(t, "e2e4", m1)

	m2, _, err := p.SearchMove(ctx, chess.StartingFEN, nil, 100)
	require.NoError(t, err)
	assert.Equal(t, "e7e5", m2)

	// Fallback to legal move
	m3, _, err := p.SearchMove(ctx, chess.StartingFEN, nil, 100)
	require.NoError(t, err)
	assert.NotEmpty(t, m3)

	assert.NoError(t, p.Close())
}

func TestEngineMatch_CheckmateFoolMate(t *testing.T) {
	// Round 1:
	// White (p1): f2f3, g2g4
	// Black (p2): e7e5, d8h4# (Checkmate!)
	p1 := NewMockEnginePlayer("EngineA", "f2f3", "g2g4")
	p2 := NewMockEnginePlayer("EngineB", "e7e5", "d8h4")

	cfg := EngineMatchConfig{
		Rounds:          1,
		TimeControlMs:   100,
		MaxPlies:        50,
		AlternateColors: false,
	}

	runner := NewEngineMatchRunner(p1, p2, cfg, nil)
	summary, err := runner.Run(context.Background(), nil, nil)
	require.NoError(t, err)

	assert.Equal(t, 1, len(summary.Games))
	g := summary.Games[0]
	assert.Equal(t, "checkmate", g.Termination)
	assert.Equal(t, "0-1", g.Result)
	assert.Equal(t, 0.0, summary.Engine1Score)
	assert.Equal(t, 1.0, summary.Engine2Score)
	assert.Equal(t, 1, summary.Engine2Wins)
	assert.Contains(t, g.PGN, "Qh4#")
}

func TestEngineMatch_ColorAlternationAndRounds(t *testing.T) {
	// 2 rounds with auto color alternation
	p1 := NewMockEnginePlayer("BotAlpha")
	p2 := NewMockEnginePlayer("BotBeta")

	cfg := EngineMatchConfig{
		Rounds:          2,
		TimeControlMs:   50,
		MaxPlies:        10, // Short game limit for testing
		AlternateColors: true,
	}

	var progressEvents int
	runner := NewEngineMatchRunner(p1, p2, cfg, nil)
	summary, err := runner.Run(
		context.Background(),
		func(prog EngineMatchProgress) {
			progressEvents++
		},
		func(game *EngineGameResult, p1Score, p2Score float64) {
			assert.NotEmpty(t, game.Result)
		},
	)
	require.NoError(t, err)

	assert.Equal(t, 2, len(summary.Games))
	assert.Equal(t, "BotAlpha", summary.Games[0].WhiteEngine)
	assert.Equal(t, "BotBeta", summary.Games[0].BlackEngine)

	assert.Equal(t, "BotBeta", summary.Games[1].WhiteEngine)
	assert.Equal(t, "BotAlpha", summary.Games[1].BlackEngine)

	assert.Greater(t, progressEvents, 0)
}

func TestEngineMatch_RepetitionDraw(t *testing.T) {
	// Knights moving back and forth -> 3-fold repetition
	// 1. Nf3 Nf6 2. Ng1 Ng8 3. Nf3 Nf6 4. Ng1 Ng8
	p1 := NewMockEnginePlayer("KnightBotW", "g1f3", "f3g1", "g1f3", "f3g1")
	p2 := NewMockEnginePlayer("KnightBotB", "g8f6", "f6g8", "g8f6", "f6g8")

	cfg := EngineMatchConfig{
		Rounds:          1,
		TimeControlMs:   50,
		MaxPlies:        50,
		AlternateColors: false,
	}

	runner := NewEngineMatchRunner(p1, p2, cfg, nil)
	summary, err := runner.Run(context.Background(), nil, nil)
	require.NoError(t, err)

	assert.Equal(t, 1, len(summary.Games))
	assert.Equal(t, "3-fold repetition", summary.Games[0].Termination)
	assert.Equal(t, "1/2-1/2", summary.Games[0].Result)
	assert.Equal(t, 0.5, summary.Engine1Score)
	assert.Equal(t, 0.5, summary.Engine2Score)
	assert.Equal(t, 1, summary.Draws)
}

func TestEngineMatch_AutoSaveToDatabase(t *testing.T) {
	p1 := NewMockEnginePlayer("Stockfish1", "e2e4", "g1f3")
	p2 := NewMockEnginePlayer("Stockfish2", "e7e5", "b8c6")

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "engine_match_db")
	db := scid.NewDatabase()
	err := db.Create(dbPath, "Engine Tournament Base")
	require.NoError(t, err)
	defer db.Close()

	cfg := EngineMatchConfig{
		Rounds:          1,
		TimeControlMs:   50,
		MaxPlies:        4,
		AlternateColors: false,
		AutoSaveToDB:    true,
	}

	runner := NewEngineMatchRunner(p1, p2, cfg, db)
	summary, err := runner.Run(context.Background(), nil, nil)
	require.NoError(t, err)

	assert.Equal(t, 1, len(summary.Games))
	assert.Equal(t, 1, db.NumGames())
}

func TestEngineMatch_ContextCancellation(t *testing.T) {
	p1 := NewMockEnginePlayer("SlowBot1")
	p2 := NewMockEnginePlayer("SlowBot2")

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	cfg := EngineMatchConfig{
		Rounds:        5,
		TimeControlMs: 50,
		MaxPlies:      50,
	}

	runner := NewEngineMatchRunner(p1, p2, cfg, nil)
	_, err := runner.Run(ctx, nil, nil)
	assert.Error(t, err)
}

type evalPlayer struct {
	name    string
	evalCp  int
	moves   []string
	moveIdx int
}

func (p *evalPlayer) GetName() string { return p.name }
func (p *evalPlayer) Close() error   { return nil }
func (p *evalPlayer) SearchMove(ctx context.Context, fen string, moves []string, movetimeMs int) (string, engine.EvalInfo, error) {
	move := "e2e4"
	if p.moveIdx < len(p.moves) {
		move = p.moves[p.moveIdx]
		p.moveIdx++
	}
	return move, engine.EvalInfo{Depth: 10, ScoreCp: p.evalCp}, nil
}

func TestEngineMatch_EvalAdjudication(t *testing.T) {
	// White has huge winning advantage (+1200 cp) for 4 plies
	p1 := &evalPlayer{
		name:   "WhiteAdvantage",
		evalCp: 1200,
		moves:  []string{"e2e4", "g1f3", "f1c4", "e1g1"},
	}
	p2 := &evalPlayer{
		name:   "BlackDefense",
		evalCp: -1200,
		moves:  []string{"e7e5", "b8c6", "f8c5", "g8f6"},
	}

	cfg := EngineMatchConfig{
		Rounds:           1,
		TimeControlMs:    50,
		MaxPlies:         20,
		AdjudicateEvalCp: 1000,
	}

	runner := NewEngineMatchRunner(p1, p2, cfg, nil)
	summary, err := runner.Run(context.Background(), nil, nil)
	require.NoError(t, err)

	assert.Equal(t, 1, len(summary.Games))
	assert.Contains(t, summary.Games[0].Termination, "adjudication")
	assert.Equal(t, "1-0", summary.Games[0].Result)
}

func TestEngineMatch_IllegalMoveForfeit(t *testing.T) {
	p1 := NewMockEnginePlayer("IllegalBot", "invalid_move")
	p2 := NewMockEnginePlayer("HonestBot", "e7e5")

	cfg := EngineMatchConfig{
		Rounds:        1,
		TimeControlMs: 50,
		MaxPlies:      10,
	}

	runner := NewEngineMatchRunner(p1, p2, cfg, nil)
	summary, err := runner.Run(context.Background(), nil, nil)
	require.NoError(t, err)

	assert.Equal(t, 1, len(summary.Games))
	assert.Contains(t, summary.Games[0].Termination, "illegal move")
	assert.Equal(t, "0-1", summary.Games[0].Result)
}

