package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"newscid/pkg/chess"
	"newscid/pkg/database/scid"
	"newscid/pkg/database/sciddriver"
	"newscid/pkg/eco"
	"newscid/pkg/engine"
	"newscid/pkg/puzzleminer"
	"newscid/pkg/repertoire"
	"newscid/pkg/sets"
	"newscid/pkg/tree"
	"newscid/pkg/version"
)

// DatabaseInfo holds metadata about the currently opened database.
type DatabaseInfo struct {
	BasePath    string `json:"basePath"`
	Name        string `json:"name"`
	NumGames    int    `json:"numGames"`
	Description string `json:"description"`
}

// BoardState represents board status sent to the frontend.
type BoardState struct {
	FEN         string   `json:"fen"`
	Turn        string   `json:"turn"`
	InCheck     bool     `json:"inCheck"`
	IsCheckmate bool     `json:"isCheckmate"`
	IsStalemate bool     `json:"isStalemate"`
	LegalMoves  []string `json:"legalMoves"` // UCI moves
	FullMove    int      `json:"fullMove"`
	HalfMove    int      `json:"halfMove"`
}

// GamesPage encapsulates paginated game results.
type GamesPage struct {
	Total int                 `json:"total"`
	Games []*scid.GameSummary `json:"games"`
}

// RepertoireCheckResult contains feedback on whether a move aligns with repertoire.
type RepertoireCheckResult struct {
	IsRepertoire bool                        `json:"isRepertoire"`
	PlayedMove   *repertoire.RepertoireMove  `json:"playedMove,omitempty"`
	Alternatives []repertoire.RepertoireMove `json:"alternatives"`
}

// App struct
type App struct {
	ctx        context.Context
	mu         sync.RWMutex
	db         *scid.Database
	eco        *eco.Classifier
	driver     *sciddriver.Driver
	engine     *engine.Engine
	activeGame *chess.Game
	repMgr     *repertoire.Manager
}

// NewApp creates a new App application struct
func NewApp() *App {
	rand.Seed(time.Now().UnixNano())
	classifier := eco.NewClassifier()
	// Attempt to load system scid.eco if available
	if _, err := os.Stat("/usr/share/scid/data/scid.eco"); err == nil {
		_ = classifier.LoadFromFile("/usr/share/scid/data/scid.eco")
	}

	return &App{
		db:     scid.NewDatabase(),
		eco:    classifier,
		driver: sciddriver.NewDriver(),
		repMgr: repertoire.NewManager(""),
	}
}


// startup is called when the app starts.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// ==========================================
// Database Operations
// ==========================================

func (a *App) OpenDatabase(path string) (*DatabaseInfo, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// If path is a PGN file, convert/cache into temporary Scid DB
	if strings.HasSuffix(strings.ToLower(path), ".pgn") {
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		games, err := chess.ParsePGNDatabase(strings.NewReader(string(content)))
		if err != nil {
			return nil, fmt.Errorf("failed to parse PGN: %w", err)
		}
		tmpBase := filepath.Join(os.TempDir(), "newscid_pgn_"+strings.TrimSuffix(filepath.Base(path), ".pgn"))
		_ = os.Remove(tmpBase + ".si4")
		_ = os.Remove(tmpBase + ".sg4")
		_ = os.Remove(tmpBase + ".sn4")
		if err := a.db.Create(tmpBase, "Imported from "+filepath.Base(path)); err != nil {
			return nil, err
		}
		for _, g := range games {
			_, _ = a.db.SaveGame(g)
		}
		return &DatabaseInfo{
			BasePath:    a.db.BasePath,
			Name:        filepath.Base(path),
			NumGames:    a.db.NumGames(),
			Description: "Imported PGN: " + filepath.Base(path),
		}, nil
	}

	if err := a.db.Open(path); err != nil {
		return nil, err
	}

	desc := ""
	if a.db.Header != nil {
		desc = a.db.Header.Description
	}

	return &DatabaseInfo{
		BasePath:    a.db.BasePath,
		Name:        filepath.Base(a.db.BasePath),
		NumGames:    a.db.NumGames(),
		Description: desc,
	}, nil
}

func (a *App) CreateDatabase(path string, description string) (*DatabaseInfo, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if err := a.db.Create(path, description); err != nil {
		return nil, err
	}

	return &DatabaseInfo{
		BasePath:    a.db.BasePath,
		Name:        filepath.Base(a.db.BasePath),
		NumGames:    0,
		Description: description,
	}, nil
}

func (a *App) CloseDatabase() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.db.Close()
}

func (a *App) GetDatabaseInfo() (*DatabaseInfo, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if !a.db.IsOpen {
		return nil, nil
	}

	desc := ""
	if a.db.Header != nil {
		desc = a.db.Header.Description
	}

	return &DatabaseInfo{
		BasePath:    a.db.BasePath,
		Name:        filepath.Base(a.db.BasePath),
		NumGames:    a.db.NumGames(),
		Description: desc,
	}, nil
}

func (a *App) GetGames(offset, limit int, criteria scid.SearchCriteria) (*GamesPage, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if !a.db.IsOpen {
		return &GamesPage{Total: 0, Games: []*scid.GameSummary{}}, nil
	}

	games, total, err := a.db.GetGameSummaries(offset, limit, criteria)
	if err != nil {
		return nil, err
	}

	return &GamesPage{Total: total, Games: games}, nil
}

// CountSearchMatches returns the number of games matching the specified search criteria.
func (a *App) CountSearchMatches(criteria scid.SearchCriteria) (int, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if !a.db.IsOpen {
		return 0, nil
	}

	return a.db.CountMatches(criteria), nil
}

func (a *App) GetGame(gameID int) (*chess.Game, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.db.IsOpen {
		return nil, fmt.Errorf("no database open")
	}

	game, err := a.db.GetGame(gameID)
	if err != nil {
		return nil, err
	}
	a.activeGame = game
	return game, nil
}

func (a *App) SaveGame(game *chess.Game) (int, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.db.IsOpen {
		return 0, fmt.Errorf("no database open")
	}

	return a.db.SaveGame(game)
}

func (a *App) DeleteGame(gameID int, deleted bool) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.db.IsOpen {
		return fmt.Errorf("no database open")
	}

	return a.db.DeleteGame(gameID, deleted)
}

func (a *App) OpenFileDialog() (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Open Chess Database",
		Filters: []runtime.FileFilter{
			{DisplayName: "SCID Databases (*.si4)", Pattern: "*.si4"},
			{DisplayName: "PGN Databases (*.pgn)", Pattern: "*.pgn"},
			{DisplayName: "All Files (*.*)", Pattern: "*.*"},
		},
	})
}

func (a *App) SaveFileDialog(defaultName string) (string, error) {
	return runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Create Chess Database",
		DefaultFilename: defaultName,
		Filters: []runtime.FileFilter{
			{DisplayName: "SCID Database (*.si4)", Pattern: "*.si4"},
		},
	})
}

// ==========================================
// Chess Board & Move Operations
// ==========================================

func (a *App) ParseFEN(fen string) (*BoardState, error) {
	b, err := chess.ParseFEN(fen)
	if err != nil {
		return nil, err
	}
	return makeBoardState(b), nil
}

func (a *App) MakeMove(fen string, moveUCI string) (*BoardState, error) {
	b, err := chess.ParseFEN(fen)
	if err != nil {
		return nil, err
	}

	m, err := b.ParseUCI(moveUCI)
	if err != nil {
		return nil, err
	}

	b.MakeMove(m)
	return makeBoardState(b), nil
}

func (a *App) GetLegalMoves(fen string) ([]string, error) {
	b, err := chess.ParseFEN(fen)
	if err != nil {
		return nil, err
	}

	moves := b.GenerateLegalMoves()
	res := make([]string, len(moves))
	for i, m := range moves {
		res[i] = m.String()
	}
	return res, nil
}

func (a *App) ParsePGN(pgnContent string) (*chess.Game, error) {
	g, err := chess.ParsePGN(pgnContent)
	if err == nil {
		a.mu.Lock()
		a.activeGame = g
		a.mu.Unlock()
	}
	return g, err
}

func (a *App) ExportPGN(game *chess.Game) (string, error) {
	if game == nil {
		return "", fmt.Errorf("nil game")
	}
	return game.ExportPGN(), nil
}

func (a *App) GetOpeningTree(fen string) (*tree.TreeResult, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return tree.BuildTreeFromDatabase(a.db, fen, 1000)
}

func (a *App) SearchPosition(fen string) ([]*scid.GameSummary, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if !a.db.IsOpen {
		return nil, fmt.Errorf("no database open")
	}

	targetBoard, err := chess.ParseFEN(fen)
	if err != nil {
		return nil, err
	}
	targetFENCore := tree.CoreFEN(targetBoard.FEN())

	numGames := a.db.NumGames()
	limit := numGames
	if limit > 2000 {
		limit = 2000
	}

	var results []*scid.GameSummary
	for i := 1; i <= limit; i++ {
		g, err := a.db.GetGame(i)
		if err != nil {
			continue
		}

		b := chess.NewBoard()
		if g.StartFEN != "" && g.StartFEN != chess.StartingFEN {
			if parsed, err := chess.ParseFEN(g.StartFEN); err == nil {
				b = parsed
			}
		}

		matched := (tree.CoreFEN(b.FEN()) == targetFENCore)
		curr := g.RootMove
		for !matched && curr != nil {
			b.MakeMove(curr.Move)
			if tree.CoreFEN(b.FEN()) == targetFENCore {
				matched = true
				break
			}
			curr = curr.Next
		}

		if matched {
			summary, err := a.db.GetGameSummary(i)
			if err == nil {
				results = append(results, summary)
			}
		}
	}

	return results, nil
}

func (a *App) ClassifyOpening(moves []string) *eco.Entry {
	return a.eco.Classify(moves)
}


// ==========================================
// UCI Engine Operations & Scid Configuration
// ==========================================

func (a *App) GetDiscoveredEngines() []engine.EngineConfig {
	return engine.DiscoverScidEngines()
}

func (a *App) GetAvailableEngines() []string {
	engines := engine.DiscoverScidEngines()
	var result []string
	for _, e := range engines {
		if e.Available {
			label := e.Name
			if label == "" {
				label = filepath.Base(e.Cmd)
			}
			result = append(result, label)
		}
	}
	if len(result) == 0 {
		return engine.FindAvailableEngines()
	}
	return result
}

func (a *App) startEngineLocked(idOrPath string) error {
	if a.engine != nil {
		_ = a.engine.Quit()
	}

	execPath := idOrPath
	// Match against discovered engines
	for _, e := range engine.DiscoverScidEngines() {
		if (e.Name == idOrPath || e.Cmd == idOrPath || filepath.Base(e.Cmd) == idOrPath) && e.Available {
			if e.ResolvedCmd != "" {
				execPath = e.ResolvedCmd
			} else {
				execPath = e.Cmd
			}
			break
		}
	}

	eng := engine.NewEngine(execPath)
	if err := eng.Start(); err != nil {
		return err
	}

	eng.SetOnEval(func(info engine.EvalInfo) {
		if a.ctx != nil && a.ctx.Value("events") != nil {
			runtime.EventsEmit(a.ctx, "engine:eval", info)
		}
	})

	a.engine = eng
	return nil
}

func (a *App) StartEngine(idOrPath string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.startEngineLocked(idOrPath)
}

func (a *App) ensureEngineLocked() error {
	if a.engine != nil {
		return nil
	}
	discovered := engine.DiscoverScidEngines()
	var chosen string
	for _, e := range discovered {
		if e.Available && e.UCI {
			chosen = e.Name
			break
		}
	}
	if chosen == "" {
		return fmt.Errorf("no UCI chess engine available on system")
	}
	return a.startEngineLocked(chosen)
}

func (a *App) StopEngine() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.engine != nil {
		return a.engine.Stop()
	}
	return nil
}

func (a *App) AnalyzePosition(fen string, moves []string, maxDepth int) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if err := a.ensureEngineLocked(); err != nil {
		return err
	}

	return a.engine.Analyze(fen, moves, maxDepth)
}

func (a *App) GetEngineEval() engine.EvalInfo {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.engine != nil {
		return a.engine.GetLastEval()
	}
	return engine.EvalInfo{}
}

func (a *App) GetBestMove(fen string, moves []string, movetimeMs int) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if err := a.ensureEngineLocked(); err != nil {
		return "", err
	}

	best, _, err := a.engine.SearchBestMove(fen, moves, movetimeMs)
	return best, err
}

func (a *App) AnnotateCurrentGame(movetimeMs int) (*engine.GameAnalysis, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.activeGame == nil {
		return nil, fmt.Errorf("no active game to annotate")
	}

	if err := a.ensureEngineLocked(); err != nil {
		return nil, err
	}

	return engine.AnnotateGame(a.activeGame, a.engine, movetimeMs)
}

func (a *App) AnnotatePGN(pgnContent string, movetimeMs int) (*engine.GameAnalysis, error) {
	g, err := chess.ParsePGN(pgnContent)
	if err != nil {
		return nil, err
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	a.activeGame = g
	if err := a.ensureEngineLocked(); err != nil {
		return nil, err
	}

	return engine.AnnotateGame(g, a.engine, movetimeMs)
}


// ==========================================
// Piece Sets & Board Themes
// ==========================================

func (a *App) GetPieceSets() []sets.PieceSetInfo {
	return sets.GetAvailablePieceSets()
}

func (a *App) GetBoardThemes() []sets.BoardThemeInfo {
	return sets.GetAvailableBoardThemes()
}

func (a *App) GetDiscoveredDatabases() []scid.DiscoveredDatabase {
	return scid.DiscoverScidDatabases()
}

func (a *App) GetScidSystemBases() []string {
	var found []string
	candidates := []string{
		"/usr/share/scid/bases/tactics",
		"/usr/share/scid/bases/matein1",
		"/usr/share/scid/bases/matein2",
		"/usr/share/scid/bases/matein3",
		"/usr/share/scid/bases/matein4andmore",
		"/usr/share/scid/bases/endings",
	}
	for _, c := range candidates {
		if _, err := os.Stat(c + ".si4"); err == nil {
			found = append(found, c)
		}
	}
	return found
}

// TacticsPuzzle contains puzzle setup and solution sequence.
type TacticsPuzzle struct {
	GameID     int      `json:"gameId"`
	TotalGames int      `json:"totalGames"`
	BaseName   string   `json:"baseName"`
	Event      string   `json:"event"`
	White      string   `json:"white"`
	Black      string   `json:"black"`
	StartFEN   string   `json:"startFen"`
	Moves      []string `json:"moves"` // Solution UCI moves in order
	SANMoves   []string `json:"sanMoves"`
	Turn       string   `json:"turn"` // "w" or "b"
	Theme      string   `json:"theme"`
}

func (a *App) LoadTacticsPuzzle(basePath string, puzzleNum int) (*TacticsPuzzle, error) {
	if basePath == "" {
		basePath = "/usr/share/scid/bases/tactics"
	}
	basePath = strings.TrimSuffix(basePath, ".si4")
	basePath = strings.TrimSuffix(basePath, ".sg4")
	basePath = strings.TrimSuffix(basePath, ".sn4")

	db := scid.NewDatabase()
	if err := db.Open(basePath); err != nil {
		return nil, fmt.Errorf("failed to open puzzle database: %w", err)
	}
	defer db.Close()

	total := db.NumGames()
	if total == 0 {
		return nil, fmt.Errorf("database has no games")
	}

	if puzzleNum <= 0 {
		puzzleNum = rand.Intn(total) + 1
	} else if puzzleNum > total {
		puzzleNum = total
	}

	game, err := db.GetGame(puzzleNum)
	if err != nil {
		return nil, fmt.Errorf("failed to load puzzle %d: %w", puzzleNum, err)
	}

	startFen := game.StartFEN
	if startFen == "" {
		startFen = chess.StartingFEN
	}

	turn := "w"
	parts := strings.Split(startFen, " ")
	if len(parts) > 1 && parts[1] == "b" {
		turn = "b"
	}

	var moves []string
	var sanMoves []string
	curr := game.RootMove
	for curr != nil {
		moves = append(moves, curr.UCI)
		sanMoves = append(sanMoves, curr.SAN)
		curr = curr.Next
	}

	theme := game.Tag("Event")
	if theme == "?" || theme == "" {
		theme = filepath.Base(basePath)
	}

	return &TacticsPuzzle{
		GameID:     puzzleNum,
		TotalGames: total,
		BaseName:   filepath.Base(basePath),
		Event:      game.Tag("Event"),
		White:      game.Tag("White"),
		Black:      game.Tag("Black"),
		StartFEN:   startFen,
		Moves:      moves,
		SANMoves:   sanMoves,
		Turn:       turn,
		Theme:      theme,
	}, nil
}

// GetProgressivePuzzle delivers a puzzle scaled dynamically to the user's level.
func (a *App) GetProgressivePuzzle(level int) (*TacticsPuzzle, error) {
	if level <= 0 {
		level = 1
	}

	var base string
	switch {
	case level <= 5:
		base = "/usr/share/scid/bases/matein1"
	case level <= 15:
		base = "/usr/share/scid/bases/matein2"
	case level <= 25:
		base = "/usr/share/scid/bases/tactics"
	case level <= 35:
		base = "/usr/share/scid/bases/matein3"
	case level <= 45:
		base = "/usr/share/scid/bases/matein4andmore"
	default:
		base = "/usr/share/scid/bases/endings"
	}

	if _, err := os.Stat(base + ".si4"); err != nil {
		base = "/usr/share/scid/bases/tactics"
	}

	puzzle, err := a.LoadTacticsPuzzle(base, 0)
	if err != nil {
		return nil, err
	}
	puzzle.Theme = fmt.Sprintf("Level %d (%s)", level, filepath.Base(base))
	return puzzle, nil
}

// ==========================================
// Opening Repertoire Operations
// ==========================================

func (a *App) GetRepertoires() []repertoire.RepertoireSummary {
	return a.repMgr.ListRepertoires()
}

func (a *App) GetRepertoire(id string) (*repertoire.Repertoire, error) {
	return a.repMgr.GetRepertoire(id)
}

func (a *App) SaveRepertoire(rep *repertoire.Repertoire) error {
	return a.repMgr.SaveRepertoire(rep)
}

func (a *App) DeleteRepertoire(id string) error {
	return a.repMgr.DeleteRepertoire(id)
}

func (a *App) VerifyRepertoireMove(repID, fen, uci string) (*RepertoireCheckResult, error) {
	isMatch, move, alts, err := a.repMgr.VerifyMove(repID, fen, uci)
	if err != nil {
		return nil, err
	}
	return &RepertoireCheckResult{
		IsRepertoire: isMatch,
		PlayedMove:   move,
		Alternatives: alts,
	}, nil
}

func (a *App) RecordRepertoirePractice(repID, fen, uci string, wasCorrect bool) {
	a.repMgr.RecordPractice(repID, fen, uci, wasCorrect)
}

func (a *App) AddRepertoireMove(repID, fen, uci, san, nextFen, comment string) error {
	rep, err := a.repMgr.GetRepertoire(repID)
	if err != nil {
		return err
	}
	core := repertoire.CoreFEN(fen)
	existing := rep.Positions[core]
	for _, m := range existing {
		if strings.EqualFold(m.UCI, uci) {
			return nil
		}
	}
	rep.Positions[core] = append(existing, repertoire.RepertoireMove{
		UCI:     uci,
		SAN:     san,
		NextFEN: repertoire.CoreFEN(nextFen),
		Comment: comment,
	})
	return a.repMgr.SaveRepertoire(rep)
}

// ==========================================
// Database Puzzle Miner Operations
// ==========================================

func (a *App) MinePuzzlesFromDatabase(playerName string, maxGames int) ([]puzzleminer.MinedPuzzle, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if !a.db.IsOpen {
		return nil, fmt.Errorf("no database open to mine puzzles from")
	}

	filter := puzzleminer.MiningFilter{
		PlayerName: playerName,
		MaxGames:   maxGames,
		MinDeltaCp: 250,
	}

	return puzzleminer.MineFromDatabase(a.db, a.engine, filter)
}

func (a *App) IsScidDriverAvailable() bool {
	return a.driver.IsAvailable()
}

// GetVersionInfo returns current application version and runtime metadata.
func (a *App) GetVersionInfo() version.Info {
	return version.Get()
}



func makeBoardState(b *chess.Board) *BoardState {
	legal := b.GenerateLegalMoves()
	legalUCI := make([]string, len(legal))
	for i, m := range legal {
		legalUCI[i] = m.String()
	}

	turnStr := "w"
	if b.Turn == chess.Black {
		turnStr = "b"
	}

	return &BoardState{
		FEN:         b.FEN(),
		Turn:        turnStr,
		InCheck:     b.InCheck(b.Turn),
		IsCheckmate: b.IsCheckmate(),
		IsStalemate: b.IsStalemate(),
		LegalMoves:  legalUCI,
		FullMove:    b.FullMove,
		HalfMove:    b.HalfMove,
	}
}
