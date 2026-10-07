package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"newscid/pkg/chess"
	"newscid/pkg/database/scid"
	"newscid/pkg/database/sciddriver"
	"newscid/pkg/eco"
	"newscid/pkg/engine"
	"newscid/pkg/sets"
	"newscid/pkg/tree"
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

// App struct
type App struct {
	ctx        context.Context
	mu         sync.RWMutex
	db         *scid.Database
	eco        *eco.Classifier
	driver     *sciddriver.Driver
	engine     *engine.Engine
	activeGame *chess.Game
}

// NewApp creates a new App application struct
func NewApp() *App {
	classifier := eco.NewClassifier()
	// Attempt to load system scid.eco if available
	if _, err := os.Stat("/usr/share/scid/data/scid.eco"); err == nil {
		_ = classifier.LoadFromFile("/usr/share/scid/data/scid.eco")
	}

	return &App{
		db:     scid.NewDatabase(),
		eco:    classifier,
		driver: sciddriver.NewDriver(),
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
	return chess.ParsePGN(pgnContent)
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

func (a *App) StartEngine(idOrPath string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

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
		runtime.EventsEmit(a.ctx, "engine:eval", info)
	})

	a.engine = eng
	return nil
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
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.engine == nil {
		// Auto-launch first available UCI engine
		discovered := engine.DiscoverScidEngines()
		var chosen string
		for _, e := range discovered {
			if e.Available && e.UCI {
				chosen = e.Name
				break
			}
		}
		if chosen != "" {
			a.mu.RUnlock()
			err := a.StartEngine(chosen)
			a.mu.RLock()
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("no chess engine installed on system")
		}
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

func (a *App) IsScidDriverAvailable() bool {
	return a.driver.IsAvailable()
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
