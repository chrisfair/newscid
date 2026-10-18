package tournament

import (
	"context"
	"fmt"
	"strings"
	"time"

	"newscid/pkg/chess"
	"newscid/pkg/database/scid"
	"newscid/pkg/engine"
)

// EnginePlayer is an abstraction for chess engines in matches and tournaments.
type EnginePlayer interface {
	GetName() string
	SearchMove(ctx context.Context, fen string, moves []string, movetimeMs int) (string, engine.EvalInfo, error)
	Close() error
}

// UCIEnginePlayer implements EnginePlayer by driving an external UCI engine process.
type UCIEnginePlayer struct {
	Name   string
	Path   string
	engine *engine.Engine
}

// NewUCIEnginePlayer launches and initializes an external UCI engine.
func NewUCIEnginePlayer(name, path string, options map[string]string) (*UCIEnginePlayer, error) {
	eng := engine.NewEngine(path)
	if err := eng.Start(); err != nil {
		return nil, fmt.Errorf("starting engine %s (%s): %w", name, path, err)
	}

	for k, v := range options {
		_ = eng.SetOption(k, v)
	}

	displayName := name
	if displayName == "" {
		displayName = eng.Name
	}
	if displayName == "" {
		displayName = path
	}

	return &UCIEnginePlayer{
		Name:   displayName,
		Path:   path,
		engine: eng,
	}, nil
}

func (p *UCIEnginePlayer) GetName() string {
	return p.Name
}

func (p *UCIEnginePlayer) SearchMove(ctx context.Context, fen string, moves []string, movetimeMs int) (string, engine.EvalInfo, error) {
	return p.engine.SearchBestMove(fen, moves, movetimeMs)
}

func (p *UCIEnginePlayer) Close() error {
	if p.engine != nil {
		return p.engine.Quit()
	}
	return nil
}

// MockEnginePlayer is an in-memory test player for automated testing and simulation.
type MockEnginePlayer struct {
	Name       string
	FixedMoves []string
	MoveIndex  int
}

func NewMockEnginePlayer(name string, fixedMoves ...string) *MockEnginePlayer {
	return &MockEnginePlayer{
		Name:       name,
		FixedMoves: fixedMoves,
	}
}

func (p *MockEnginePlayer) GetName() string {
	return p.Name
}

func (p *MockEnginePlayer) SearchMove(ctx context.Context, fen string, moves []string, movetimeMs int) (string, engine.EvalInfo, error) {
	select {
	case <-ctx.Done():
		return "", engine.EvalInfo{}, ctx.Err()
	default:
	}

	if p.MoveIndex < len(p.FixedMoves) {
		m := p.FixedMoves[p.MoveIndex]
		p.MoveIndex++
		return m, engine.EvalInfo{Depth: 5, ScoreCp: 25}, nil
	}

	// Default fallback: play first legal move on board
	b, err := chess.ParseFEN(fen)
	if err != nil {
		return "", engine.EvalInfo{}, err
	}
	legal := b.GenerateLegalMoves()
	if len(legal) == 0 {
		return "", engine.EvalInfo{}, fmt.Errorf("no legal moves available")
	}

	firstMove := legal[0]
	uci := firstMove.String()

	return uci, engine.EvalInfo{Depth: 5, ScoreCp: 10, BestMove: uci}, nil
}

func (p *MockEnginePlayer) Close() error {
	return nil
}

// EngineConfig holds configuration for one engine participant.
type EngineConfig struct {
	Name    string            `json:"name"`
	Path    string            `json:"path"`
	Options map[string]string `json:"options,omitempty"`
}

// EngineMatchConfig configures an automated engine vs engine match.
type EngineMatchConfig struct {
	Engine1          EngineConfig `json:"engine1"`
	Engine2          EngineConfig `json:"engine2"`
	Rounds           int          `json:"rounds"`
	TimeControlMs    int          `json:"timeControlMs"`
	MaxPlies         int          `json:"maxPlies"`
	AlternateColors  bool         `json:"alternateColors"`
	OpeningFENs      []string     `json:"openingFens,omitempty"`
	AdjudicateEvalCp int          `json:"adjudicateEvalCp,omitempty"`
	AutoSaveToDB     bool         `json:"autoSaveToDb"`
}

// EngineGameResult describes the outcome of a single round in an engine match.
type EngineGameResult struct {
	Round       int      `json:"round"`
	WhiteEngine string   `json:"whiteEngine"`
	BlackEngine string   `json:"blackEngine"`
	Result      string   `json:"result"` // "1-0", "0-1", "1/2-1/2"
	Termination string   `json:"termination"`
	Plies       int      `json:"plies"`
	MovesUCI    []string `json:"movesUci"`
	MovesSAN    []string `json:"movesSan"`
	FinalFEN    string   `json:"finalFen"`
	PGN         string   `json:"pgn"`
	DurationMs  int64    `json:"durationMs"`
	ECO         string   `json:"eco,omitempty"`
}

// EngineMatchProgress provides live telemetry during an active match.
type EngineMatchProgress struct {
	CurrentRound int             `json:"currentRound"`
	TotalRounds  int             `json:"totalRounds"`
	CurrentPly   int             `json:"currentPly"`
	LastMoveUCI  string          `json:"lastMoveUci"`
	LastMoveSAN  string          `json:"lastMoveSan"`
	CurrentFEN   string          `json:"currentFen"`
	WhiteEngine  string          `json:"whiteEngine"`
	BlackEngine  string          `json:"blackEngine"`
	Engine1Score float64         `json:"engine1Score"`
	Engine2Score float64         `json:"engine2Score"`
	LastEval     engine.EvalInfo `json:"lastEval"`
}

// EngineMatchSummary contains the final complete outcome of an engine match.
type EngineMatchSummary struct {
	Engine1Name  string              `json:"engine1Name"`
	Engine2Name  string              `json:"engine2Name"`
	Rounds       int                 `json:"rounds"`
	Engine1Score float64             `json:"engine1Score"`
	Engine2Score float64             `json:"engine2Score"`
	Engine1Wins  int                 `json:"engine1Wins"`
	Engine2Wins  int                 `json:"engine2Wins"`
	Draws        int                 `json:"draws"`
	Games        []*EngineGameResult `json:"games"`
	DurationMs   int64               `json:"durationMs"`
}

// EngineMatchRunner executes an engine vs engine match.
type EngineMatchRunner struct {
	Player1  EnginePlayer
	Player2  EnginePlayer
	Config   EngineMatchConfig
	TargetDB *scid.Database
}

// NewEngineMatchRunner creates a runner with custom or pre-instantiated players.
func NewEngineMatchRunner(p1, p2 EnginePlayer, cfg EngineMatchConfig, targetDB *scid.Database) *EngineMatchRunner {
	if cfg.Rounds <= 0 {
		cfg.Rounds = 2
	}
	if cfg.TimeControlMs <= 0 {
		cfg.TimeControlMs = 500
	}
	if cfg.MaxPlies <= 0 {
		cfg.MaxPlies = 150
	}
	return &EngineMatchRunner{
		Player1:  p1,
		Player2:  p2,
		Config:   cfg,
		TargetDB: targetDB,
	}
}

// Run executes the match rounds and returns a full summary.
func (r *EngineMatchRunner) Run(
	ctx context.Context,
	onProgress func(prog EngineMatchProgress),
	onGameEnd func(game *EngineGameResult, p1Score, p2Score float64),
) (*EngineMatchSummary, error) {
	startTime := time.Now()

	summary := &EngineMatchSummary{
		Engine1Name: r.Player1.GetName(),
		Engine2Name: r.Player2.GetName(),
		Rounds:      r.Config.Rounds,
		Games:       make([]*EngineGameResult, 0, r.Config.Rounds),
	}

	for round := 1; round <= r.Config.Rounds; round++ {
		select {
		case <-ctx.Done():
			summary.DurationMs = time.Since(startTime).Milliseconds()
			return summary, ctx.Err()
		default:
		}

		// Determine colors for this round
		isP1White := true
		if r.Config.AlternateColors && (round%2 == 0) {
			isP1White = false
		}

		whitePlayer := r.Player1
		blackPlayer := r.Player2
		if !isP1White {
			whitePlayer = r.Player2
			blackPlayer = r.Player1
		}

		// Choose opening FEN if provided
		startFEN := chess.StartingFEN
		if len(r.Config.OpeningFENs) > 0 {
			fenIdx := (round - 1) % len(r.Config.OpeningFENs)
			startFEN = r.Config.OpeningFENs[fenIdx]
		}

		gameRes, err := r.playSingleGame(
			ctx,
			round,
			whitePlayer,
			blackPlayer,
			startFEN,
			isP1White,
			summary,
			onProgress,
		)
		if err != nil && gameRes == nil {
			return summary, err
		}

		summary.Games = append(summary.Games, gameRes)

		// Update scores
		switch gameRes.Result {
		case "1-0":
			if isP1White {
				summary.Engine1Score += 1.0
				summary.Engine1Wins++
			} else {
				summary.Engine2Score += 1.0
				summary.Engine2Wins++
			}
		case "0-1":
			if isP1White {
				summary.Engine2Score += 1.0
				summary.Engine2Wins++
			} else {
				summary.Engine1Score += 1.0
				summary.Engine1Wins++
			}
		case "1/2-1/2":
			summary.Engine1Score += 0.5
			summary.Engine2Score += 0.5
			summary.Draws++
		}

		// Optional: auto-save game into target database
		if r.Config.AutoSaveToDB && r.TargetDB != nil && r.TargetDB.IsOpen {
			_ = r.saveGameToDB(gameRes)
		}

		if onGameEnd != nil {
			onGameEnd(gameRes, summary.Engine1Score, summary.Engine2Score)
		}
	}

	summary.DurationMs = time.Since(startTime).Milliseconds()
	return summary, nil
}

type evalSide int8

const (
	evalSideNeutral evalSide = 0
	evalSideWhite   evalSide = 1
	evalSideBlack   evalSide = -1
)

func outcomeToPGNString(r gameOutcome) string {
	switch r {
	case outcomeWhiteWins:
		return "1-0"
	case outcomeBlackWins:
		return "0-1"
	case outcomeDraw:
		return "1/2-1/2"
	default:
		return "*"
	}
}

func (r *EngineMatchRunner) playSingleGame(
	ctx context.Context,
	round int,
	whitePlayer, blackPlayer EnginePlayer,
	startFEN string,
	isP1White bool,
	currentSummary *EngineMatchSummary,
	onProgress func(prog EngineMatchProgress),
) (*EngineGameResult, error) {
	board, err := chess.ParseFEN(startFEN)
	if err != nil {
		return nil, fmt.Errorf("parsing start FEN for round %d: %w", round, err)
	}

	gameStart := time.Now()
	outcome := outcomeUnknown
	res := &EngineGameResult{
		Round:       round,
		WhiteEngine: whitePlayer.GetName(),
		BlackEngine: blackPlayer.GetName(),
		Result:      outcomeToPGNString(outcome),
		MovesUCI:    make([]string, 0),
		MovesSAN:    make([]string, 0),
	}

	seenPositions := make(map[string]int)
	seenPositions[positionRepKey(board.FEN())]++

	ply := 0
	consecutiveEvalCount := 0
	lastEvalSide := evalSideNeutral

	for ply < r.Config.MaxPlies {
		select {
		case <-ctx.Done():
			res.Termination = "cancelled"
			res.Result = outcomeToPGNString(outcomeUnknown)
			res.FinalFEN = board.FEN()
			res.DurationMs = time.Since(gameStart).Milliseconds()
			res.Plies = ply
			res.PGN = r.buildPGN(res)
			return res, ctx.Err()
		default:
		}

		// 1. Check checkmate / stalemate
		if board.IsCheckmate() {
			res.Termination = "checkmate"
			if board.Turn == chess.White {
				outcome = outcomeBlackWins
			} else {
				outcome = outcomeWhiteWins
			}
			break
		}
		if board.IsStalemate() {
			res.Termination = "stalemate"
			outcome = outcomeDraw
			break
		}

		// 2. 50-move rule
		if board.HalfMove >= 100 {
			res.Termination = "50-move rule"
			outcome = outcomeDraw
			break
		}

		// 3. Whose turn?
		var activePlayer EnginePlayer
		if board.Turn == chess.White {
			activePlayer = whitePlayer
		} else {
			activePlayer = blackPlayer
		}

		// 4. Request move from active engine (board.FEN() already includes all moves played so far)
		moveUCI, evalInfo, err := activePlayer.SearchMove(ctx, board.FEN(), nil, r.Config.TimeControlMs)
		if err != nil {
			res.Termination = fmt.Sprintf("engine error: %v", err)
			if board.Turn == chess.White {
				outcome = outcomeBlackWins
			} else {
				outcome = outcomeWhiteWins
			}
			break
		}

		moveUCI = strings.TrimSpace(moveUCI)
		if moveUCI == "" {
			res.Termination = "engine forfeited (empty move)"
			if board.Turn == chess.White {
				outcome = outcomeBlackWins
			} else {
				outcome = outcomeWhiteWins
			}
			break
		}

		// 5. Parse and apply move
		parsedMove, err := board.ParseUCI(moveUCI)
		if err != nil {
			res.Termination = fmt.Sprintf("engine illegal move %s: %v", moveUCI, err)
			if board.Turn == chess.White {
				outcome = outcomeBlackWins
			} else {
				outcome = outcomeWhiteWins
			}
			break
		}

		moveSAN := board.MoveToSAN(parsedMove)
		board.MakeMove(parsedMove)

		ply++
		res.MovesUCI = append(res.MovesUCI, moveUCI)
		res.MovesSAN = append(res.MovesSAN, moveSAN)

		// 6. 3-fold repetition check
		repKey := positionRepKey(board.FEN())
		seenPositions[repKey]++
		if seenPositions[repKey] >= 3 {
			res.Termination = "3-fold repetition"
			outcome = outcomeDraw
			break
		}

		// 7. Continuous Eval Adjudication (evalInfo.ScoreCp is already normalized to White's perspective)
		if r.Config.AdjudicateEvalCp > 0 && evalInfo.Depth >= 6 {
			score := evalInfo.ScoreCp
			if score >= r.Config.AdjudicateEvalCp {
				if lastEvalSide == evalSideWhite {
					consecutiveEvalCount++
				} else {
					lastEvalSide = evalSideWhite
					consecutiveEvalCount = 1
				}
			} else if score <= -r.Config.AdjudicateEvalCp {
				if lastEvalSide == evalSideBlack {
					consecutiveEvalCount++
				} else {
					lastEvalSide = evalSideBlack
					consecutiveEvalCount = 1
				}
			} else {
				consecutiveEvalCount = 0
				lastEvalSide = evalSideNeutral
			}

			if consecutiveEvalCount >= 4 {
				if lastEvalSide == evalSideWhite {
					res.Termination = fmt.Sprintf("adjudication (+%d cp for White)", r.Config.AdjudicateEvalCp)
					outcome = outcomeWhiteWins
				} else {
					res.Termination = fmt.Sprintf("adjudication (+%d cp for Black)", r.Config.AdjudicateEvalCp)
					outcome = outcomeBlackWins
				}
				break
			}
		}

		// Progress telemetry
		if onProgress != nil {
			p1Score := currentSummary.Engine1Score
			p2Score := currentSummary.Engine2Score
			onProgress(EngineMatchProgress{
				CurrentRound: round,
				TotalRounds:  r.Config.Rounds,
				CurrentPly:   ply,
				LastMoveUCI:  moveUCI,
				LastMoveSAN:  moveSAN,
				CurrentFEN:   board.FEN(),
				WhiteEngine:  whitePlayer.GetName(),
				BlackEngine:  blackPlayer.GetName(),
				Engine1Score: p1Score,
				Engine2Score: p2Score,
				LastEval:     evalInfo,
			})
		}
	}

	if outcome == outcomeUnknown {
		res.Termination = fmt.Sprintf("max plies reached (%d)", r.Config.MaxPlies)
		outcome = outcomeDraw
	}

	res.Result = outcomeToPGNString(outcome)
	res.Plies = ply
	res.FinalFEN = board.FEN()
	res.DurationMs = time.Since(gameStart).Milliseconds()
	res.PGN = r.buildPGN(res)

	return res, nil
}

func positionRepKey(fen string) string {
	parts := strings.Split(fen, " ")
	if len(parts) >= 4 {
		return parts[0] + " " + parts[1] + " " + parts[2] + " " + parts[3]
	}
	return fen
}

func (r *EngineMatchRunner) buildPGN(g *EngineGameResult) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("[Event \"Engine Match: %s vs %s\"]\n", r.Player1.GetName(), r.Player2.GetName()))
	sb.WriteString("[Site \"NewScid\"]\n")
	sb.WriteString(fmt.Sprintf("[Date \"%s\"]\n", time.Now().Format("2006.01.02")))
	sb.WriteString(fmt.Sprintf("[Round \"%d\"]\n", g.Round))
	sb.WriteString(fmt.Sprintf("[White \"%s\"]\n", g.WhiteEngine))
	sb.WriteString(fmt.Sprintf("[Black \"%s\"]\n", g.BlackEngine))
	sb.WriteString(fmt.Sprintf("[Result \"%s\"]\n", g.Result))
	if g.Termination != "" {
		sb.WriteString(fmt.Sprintf("[Termination \"%s\"]\n", g.Termination))
	}
	sb.WriteString(fmt.Sprintf("[PlyCount \"%d\"]\n\n", g.Plies))

	for i := 0; i < len(g.MovesSAN); i++ {
		if i%2 == 0 {
			moveNum := (i / 2) + 1
			sb.WriteString(fmt.Sprintf("%d. ", moveNum))
		}
		sb.WriteString(g.MovesSAN[i])
		sb.WriteString(" ")
	}

	sb.WriteString(g.Result)
	sb.WriteString("\n\n")
	return sb.String()
}

func (r *EngineMatchRunner) saveGameToDB(g *EngineGameResult) error {
	if r.TargetDB == nil || !r.TargetDB.IsOpen {
		return fmt.Errorf("target database is not open")
	}

	games, err := chess.ParsePGNDatabase(strings.NewReader(g.PGN))
	if err != nil || len(games) == 0 {
		return fmt.Errorf("failed to parse generated game PGN: %w", err)
	}

	_, err = r.TargetDB.ImportGames(games)
	return err
}
