package tournament

import (
	"context"
	"os"
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
		evalCp: 1200,
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

type customMovePlayer struct {
	name  string
	fn    func(ctx context.Context, fen string, ply int) (string, engine.EvalInfo, error)
	calls int
}

func (p *customMovePlayer) GetName() string { return p.name }
func (p *customMovePlayer) Close() error    { return nil }
func (p *customMovePlayer) SearchMove(ctx context.Context, fen string, moves []string, movetimeMs int) (string, engine.EvalInfo, error) {
	p.calls++
	return p.fn(ctx, fen, p.calls)
}

func TestUCIEnginePlayer_LifecycleAndErrors(t *testing.T) {
	// 1. Invalid path error
	_, err := NewUCIEnginePlayer("BadEngine", "/nonexistent/path/to/engine", nil)
	assert.Error(t, err)

	// Nil engine Close
	var emptyPlayer UCIEnginePlayer
	assert.NoError(t, emptyPlayer.Close())

	// 2. Fake UCI engine script in TempDir
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "fake_uci.sh")
	script := `#!/bin/sh
while read -r line; do
  case "$line" in
    uci)
      echo "id name FakeChessEngine 1.0"
      echo "uciok"
      ;;
    isready)
      echo "readyok"
      ;;
    go*)
      echo "info depth 6 score cp 42 pv e2e4"
      echo "bestmove e2e4"
      ;;
    quit)
      exit 0
      ;;
  esac
done
`
	require.NoError(t, writeExecutableScript(scriptPath, script))

	p, err := NewUCIEnginePlayer("", scriptPath, map[string]string{"Hash": "16"})
	require.NoError(t, err)
	assert.Equal(t, "FakeChessEngine 1.0", p.GetName())

	bestMove, info, err := p.SearchMove(context.Background(), chess.StartingFEN, nil, 50)
	require.NoError(t, err)
	assert.Equal(t, "e2e4", bestMove)
	assert.Equal(t, 6, info.Depth)
	assert.NoError(t, p.Close())
}

func writeExecutableScript(path, content string) error {
	return os.WriteFile(path, []byte(content), 0755)
}

func TestEngineMatch_AdditionalTerminationsAndBranches(t *testing.T) {
	// 1. Default config values in NewEngineMatchRunner
	rDefault := NewEngineMatchRunner(NewMockEnginePlayer("P1"), NewMockEnginePlayer("P2"), EngineMatchConfig{}, nil)
	assert.Equal(t, 2, rDefault.Config.Rounds)
	assert.Equal(t, 500, rDefault.Config.TimeControlMs)
	assert.Equal(t, 150, rDefault.Config.MaxPlies)

	// 2. Invalid OpeningFEN returns error
	rBadFEN := NewEngineMatchRunner(NewMockEnginePlayer("P1"), NewMockEnginePlayer("P2"), EngineMatchConfig{
		Rounds:      1,
		OpeningFENs: []string{"invalid_fen"},
	}, nil)
	_, err := rBadFEN.Run(context.Background(), nil, nil)
	assert.Error(t, err)

	// 3. Stalemate starting from a stalemate FEN (Black to move, stalemated)
	stalemateFEN := "7k/5Q2/6K1/8/8/8/8/8 b - - 0 1"
	rStalemate := NewEngineMatchRunner(NewMockEnginePlayer("P1"), NewMockEnginePlayer("P2"), EngineMatchConfig{
		Rounds:      1,
		OpeningFENs: []string{stalemateFEN},
	}, nil)
	sumStale, err := rStalemate.Run(context.Background(), nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "stalemate", sumStale.Games[0].Termination)
	assert.Equal(t, "1/2-1/2", sumStale.Games[0].Result)

	// 4. 50-move rule (HalfMove >= 100)
	fiftyMoveFEN := "4k3/8/8/8/8/8/8/4K2R w - - 100 60"
	rFifty := NewEngineMatchRunner(NewMockEnginePlayer("P1"), NewMockEnginePlayer("P2"), EngineMatchConfig{
		Rounds:      1,
		OpeningFENs: []string{fiftyMoveFEN},
	}, nil)
	sumFifty, err := rFifty.Run(context.Background(), nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "50-move rule", sumFifty.Games[0].Termination)
	assert.Equal(t, "1/2-1/2", sumFifty.Games[0].Result)

	// 5. White wins by checkmate (protected Queen on g7) in 2 rounds with AlternateColors
	whiteMateFEN := "6k1/6Qp/6K1/8/8/8/8/8 b - - 0 1"
	rWhiteMate := NewEngineMatchRunner(NewMockEnginePlayer("P1"), NewMockEnginePlayer("P2"), EngineMatchConfig{
		Rounds:          2,
		AlternateColors: true,
		OpeningFENs:     []string{whiteMateFEN},
	}, nil)
	sumWM, err := rWhiteMate.Run(context.Background(), nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "1-0", sumWM.Games[0].Result)
	assert.Equal(t, 1, sumWM.Engine1Wins)
	assert.Equal(t, 1, sumWM.Engine2Wins)

	// 6. Black wins in Round 2 when AlternateColors is true (so P1 is Black and wins)
	blackMateFEN := "8/8/8/8/8/6k1/5qpp/6K1 w - - 0 1"
	rBlackMate := NewEngineMatchRunner(NewMockEnginePlayer("P1"), NewMockEnginePlayer("P2"), EngineMatchConfig{
		Rounds:          2,
		AlternateColors: true,
		OpeningFENs:     []string{blackMateFEN},
	}, nil)
	sumBM, err := rBlackMate.Run(context.Background(), nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "0-1", sumBM.Games[1].Result)
	assert.Equal(t, 1, sumBM.Engine1Wins)
	assert.Equal(t, 1, sumBM.Engine2Wins)

	// 7. Engine error, empty move forfeit, and illegal move by Black
	errWhite := &customMovePlayer{name: "ErrW", fn: func(ctx context.Context, fen string, ply int) (string, engine.EvalInfo, error) {
		return "", engine.EvalInfo{}, assert.AnError
	}}
	sumErrW, err := NewEngineMatchRunner(errWhite, NewMockEnginePlayer("P2"), EngineMatchConfig{Rounds: 1}, nil).Run(context.Background(), nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "0-1", sumErrW.Games[0].Result)

	errBlack := &customMovePlayer{name: "ErrB", fn: func(ctx context.Context, fen string, ply int) (string, engine.EvalInfo, error) {
		return "", engine.EvalInfo{}, assert.AnError
	}}
	sumErrB, err := NewEngineMatchRunner(NewMockEnginePlayer("P1", "e2e4"), errBlack, EngineMatchConfig{Rounds: 1}, nil).Run(context.Background(), nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "1-0", sumErrB.Games[0].Result)

	emptyWhite := NewMockEnginePlayer("EmptyW", "   ")
	sumEmptyW, err := NewEngineMatchRunner(emptyWhite, NewMockEnginePlayer("P2"), EngineMatchConfig{Rounds: 1}, nil).Run(context.Background(), nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "0-1", sumEmptyW.Games[0].Result)

	emptyBlack := NewMockEnginePlayer("EmptyB", "   ")
	sumEmptyB, err := NewEngineMatchRunner(NewMockEnginePlayer("P1", "e2e4"), emptyBlack, EngineMatchConfig{Rounds: 1}, nil).Run(context.Background(), nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "1-0", sumEmptyB.Games[0].Result)

	illegalBlack := NewMockEnginePlayer("IllegalB", "invalid_uci")
	sumIllB, err := NewEngineMatchRunner(NewMockEnginePlayer("P1", "e2e4"), illegalBlack, EngineMatchConfig{Rounds: 1}, nil).Run(context.Background(), nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "1-0", sumIllB.Games[0].Result)

	// 8. Eval adjudication for Black (and score reset before 4 consecutive plies)
	blackAdjW := &customMovePlayer{name: "W", fn: func(ctx context.Context, fen string, call int) (string, engine.EvalInfo, error) {
		moves := []string{"e2e4", "g1f3", "f1c4", "d2d3"}
		cp := -1200
		if call == 1 {
			cp = 0 // neutral first ply to test reset branch
		}
		return moves[call-1], engine.EvalInfo{Depth: 10, ScoreCp: cp}, nil
	}}
	blackAdjB := &customMovePlayer{name: "B", fn: func(ctx context.Context, fen string, call int) (string, engine.EvalInfo, error) {
		moves := []string{"e7e5", "b8c6", "f8c5"}
		return moves[call-1], engine.EvalInfo{Depth: 10, ScoreCp: -1200}, nil
	}}
	sumAdjB, err := NewEngineMatchRunner(blackAdjW, blackAdjB, EngineMatchConfig{
		Rounds:           1,
		AdjudicateEvalCp: 1000,
	}, nil).Run(context.Background(), nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "0-1", sumAdjB.Games[0].Result)
	assert.Contains(t, sumAdjB.Games[0].Termination, "for Black")

	// 9. Context cancelled mid-game inside playSingleGame
	ctxMid, cancelMid := context.WithCancel(context.Background())
	cancelOnMove := &customMovePlayer{name: "CancelBot", fn: func(ctx context.Context, fen string, call int) (string, engine.EvalInfo, error) {
		cancelMid()
		return "e2e4", engine.EvalInfo{}, nil
	}}
	_, err = NewEngineMatchRunner(cancelOnMove, NewMockEnginePlayer("P2"), EngineMatchConfig{Rounds: 1}, nil).Run(ctxMid, nil, nil)
	assert.NoError(t, err) // playSingleGame returns non-nil res on mid-game cancel

	// 10. MockEnginePlayer edge cases (cancelled ctx, invalid FEN, no legal moves)
	mockP := NewMockEnginePlayer("Mock")
	_, _, err = mockP.SearchMove(ctxMid, chess.StartingFEN, nil, 10)
	assert.Error(t, err)
	_, _, err = mockP.SearchMove(context.Background(), "bad_fen", nil, 10)
	assert.Error(t, err)
	_, _, err = mockP.SearchMove(context.Background(), stalemateFEN, nil, 10)
	assert.Error(t, err)

	// 11. positionRepKey short FEN & saveGameToDB errors
	assert.Equal(t, "short", positionRepKey("short"))
	assert.Error(t, rDefault.saveGameToDB(&EngineGameResult{PGN: ""}))
}


