package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"newscid/pkg/chess"
	"newscid/pkg/database/scid"
	"newscid/pkg/database/sciddriver"
	"newscid/pkg/eco"
	"newscid/pkg/engine"
	"newscid/pkg/online"
	"newscid/pkg/persona"
	"newscid/pkg/playerbot"
	"newscid/pkg/puzzleminer"
	"newscid/pkg/repertoire"
	"newscid/pkg/sets"
	"newscid/pkg/tournament"
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
	activeGame    *chess.Game
	repMgr        *repertoire.Manager
	profileMgr    *online.ProfileManager
	syncMgr       *online.SyncManager
	personaMgr    *persona.PersonaManager
	engineMultiPV int
	engineThreads int
	engineHashMB  int
	notationStyle chess.NotationStyle
}

// NewApp creates a new App application struct
func NewApp() *App {
	rand.Seed(time.Now().UnixNano())
	classifier := eco.NewClassifier()
	// Attempt to load system scid.eco if available
	if _, err := os.Stat("/usr/share/scid/data/scid.eco"); err == nil {
		_ = classifier.LoadFromFile("/usr/share/scid/data/scid.eco")
	}

	profileMgr, _ := online.NewProfileManager("")
	personaMgr, _ := persona.NewPersonaManager("")

	return &App{
		db:            scid.NewDatabase(),
		eco:           classifier,
		driver:        sciddriver.NewDriver(),
		repMgr:        repertoire.NewManager(""),
		profileMgr:    profileMgr,
		syncMgr:       online.NewSyncManager(profileMgr),
		personaMgr:    personaMgr,
		engineMultiPV: 1,
	}
}


// startup is called when the app starts.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) emitEvent(eventName string, optionalData ...interface{}) {
	if a.ctx != nil && a.ctx.Value("events") != nil {
		runtime.EventsEmit(a.ctx, eventName, optionalData...)
	}
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

// CompactDatabase physically cleans deleted games from disk and defragments game data offsets.
func (a *App) CompactDatabase() (*scid.CompactStats, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.db.IsOpen {
		return nil, fmt.Errorf("no database open")
	}

	return a.db.Compact()
}

// SortDatabase reorders the database index based on the chosen field and direction.
func (a *App) SortDatabase(criteria scid.SortCriteria) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.db.IsOpen {
		return fmt.Errorf("no database open")
	}

	return a.db.Sort(criteria)
}

// FindDuplicateGames finds all groups of duplicate games in the current database.
func (a *App) FindDuplicateGames(criteria scid.DuplicateCriteria) ([]scid.DuplicateGroup, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if !a.db.IsOpen {
		return nil, fmt.Errorf("no database open")
	}

	return a.db.FindDuplicates(criteria)
}

// DeleteDuplicateGames marks duplicate copies of games as deleted, keeping the primary instance.
func (a *App) DeleteDuplicateGames(criteria scid.DuplicateCriteria) (int, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.db.IsOpen {
		return 0, fmt.Errorf("no database open")
	}

	return a.db.DeleteDuplicates(criteria)
}

// VerifyDatabase performs a thorough integrity check on the active database files.
func (a *App) VerifyDatabase() (*scid.VerificationReport, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if !a.db.IsOpen {
		return nil, fmt.Errorf("no database open")
	}

	return a.db.Verify()
}

// GetDatabaseStatistics aggregates demographic, rating, result, and storage metrics.
func (a *App) GetDatabaseStatistics() (*scid.DatabaseStatistics, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if !a.db.IsOpen {
		return nil, fmt.Errorf("no database open")
	}

	return a.db.GetStatistics()
}

// ImportPGNBatch imports multiple PGN games from a raw string.
func (a *App) ImportPGNBatch(pgnText string) (*scid.ImportStats, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.db.IsOpen {
		return nil, fmt.Errorf("no database open")
	}

	return a.db.ImportPGNText(pgnText)
}

// ImportPGNFileBatch imports multiple PGN games from a file on disk.
func (a *App) ImportPGNFileBatch(filePath string) (*scid.ImportStats, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.db.IsOpen {
		return nil, fmt.Errorf("no database open")
	}

	return a.db.ImportPGNFile(filePath)
}

func (a *App) OpenPGNFileDialog() (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select PGN File to Import",
		Filters: []runtime.FileFilter{
			{DisplayName: "PGN Files (*.pgn)", Pattern: "*.pgn"},
			{DisplayName: "All Files (*.*)", Pattern: "*.*"},
		},
	})
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
	if a.engineMultiPV > 1 {
		_ = eng.SetMultiPV(a.engineMultiPV)
	}
	if a.engineThreads > 0 {
		_ = eng.SetThreads(a.engineThreads)
	}
	if a.engineHashMB > 0 {
		_ = eng.SetHash(a.engineHashMB)
	}

	if err := eng.Start(); err != nil {
		return err
	}

	eng.SetOnEval(func(info engine.EvalInfo) {
		a.emitEvent("engine:eval", info)
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

func (a *App) SetEngineMultiPV(lines int) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if lines < 1 {
		lines = 1
	} else if lines > 10 {
		lines = 10
	}
	a.engineMultiPV = lines

	if a.engine != nil {
		return a.engine.SetMultiPV(lines)
	}
	return nil
}

func (a *App) GetEngineMultiPV() int {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.engine != nil {
		return a.engine.GetMultiPV()
	}
	if a.engineMultiPV > 0 {
		return a.engineMultiPV
	}
	return 1
}

func (a *App) SetEngineThreads(threads int) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.engineThreads = threads
	if a.engine != nil {
		return a.engine.SetThreads(threads)
	}
	return nil
}

func (a *App) SetEngineHash(mb int) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.engineHashMB = mb
	if a.engine != nil {
		return a.engine.SetHash(mb)
	}
	return nil
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

// ==========================================
// Online Account Sync (Chess.com & Lichess)
// ==========================================

func (a *App) GetSyncProfiles() ([]*online.Profile, error) {
	return a.profileMgr.GetProfiles(), nil
}

func (a *App) GetSyncProfile(id string) (*online.Profile, error) {
	return a.profileMgr.GetProfile(id)
}

func (a *App) SaveSyncProfile(p *online.Profile) error {
	return a.profileMgr.SaveProfile(p)
}

func (a *App) DeleteSyncProfile(id string) error {
	return a.profileMgr.DeleteProfile(id)
}

func (a *App) SyncOnlineProfile(profileID string) (*online.SyncResult, error) {
	profile, err := a.profileMgr.GetProfile(profileID)
	if err != nil {
		return nil, err
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	var targetDB *scid.Database
	shouldCloseTarget := false

	targetPath := strings.TrimSpace(profile.TargetDatabase)
	if targetPath == "" {
		if !a.db.IsOpen {
			return nil, fmt.Errorf("no target database specified in profile and no database currently open")
		}
		targetDB = a.db
	} else {
		basePath := strings.TrimSuffix(targetPath, ".si4")
		if a.db.IsOpen && filepath.Clean(a.db.BasePath) == filepath.Clean(basePath) {
			targetDB = a.db
		} else {
			targetDB = scid.NewDatabase()
			shouldCloseTarget = true

			if _, err := os.Stat(basePath + ".si4"); err == nil {
				if err := targetDB.Open(basePath); err != nil {
					return nil, fmt.Errorf("failed to open target database: %w", err)
				}
			} else {
				if err := targetDB.Create(basePath, fmt.Sprintf("Online sync for %s", profile.Name)); err != nil {
					return nil, fmt.Errorf("failed to create target database: %w", err)
				}
			}
		}
	}

	if shouldCloseTarget {
		defer targetDB.Close()
	}

	return a.syncMgr.SyncProfile(profile, targetDB)
}

func (a *App) FetchLichessPreview(username string, count int) (string, error) {
	if count <= 0 {
		count = 5
	}
	return a.syncMgr.Lichess.FetchUserGames(username, 0, "", false, count)
}

func (a *App) FetchChessComPreview(username string) ([]string, error) {
	return a.syncMgr.ChessCom.FetchArchives(username)
}

// ==========================================
// Notation Styles & Formatting
// ==========================================

// SetNotationStyle sets the active notation style (san, fan, lan, descriptive, uci, iccf).
func (a *App) SetNotationStyle(style string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if !chess.ValidNotationStyle(style) {
		return fmt.Errorf("invalid notation style: %s", style)
	}
	a.notationStyle = chess.NotationStyle(strings.ToLower(style))
	return nil
}

// GetNotationStyle returns the active notation format.
func (a *App) GetNotationStyle() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.notationStyle == "" {
		return string(chess.NotationSAN)
	}
	return string(a.notationStyle)
}

// FormatMove formats a UCI move from a given FEN position into the target notation style.
func (a *App) FormatMove(fen string, moveUCI string, style string) (string, error) {
	st := chess.NotationStyle(strings.ToLower(style))
	if !chess.ValidNotationStyle(style) {
		a.mu.RLock()
		st = a.notationStyle
		a.mu.RUnlock()
		if st == "" {
			st = chess.NotationSAN
		}
	}
	return chess.FormatMoveWithFEN(fen, moveUCI, st)
}

// FormatMoveSequence formats a sequence of UCI moves into readable moves with full-move numbers.
func (a *App) FormatMoveSequence(fen string, movesUCI []string, style string) (string, error) {
	st := chess.NotationStyle(strings.ToLower(style))
	if !chess.ValidNotationStyle(style) {
		a.mu.RLock()
		st = a.notationStyle
		a.mu.RUnlock()
		if st == "" {
			st = chess.NotationSAN
		}
	}
	return chess.FormatPVWithStyle(fen, movesUCI, st), nil
}

// ==========================================
// Game Annotations & Variations
// ==========================================

// AddGameComment attaches a comment to the move node at the given 1-based ply in activeGame.
func (a *App) AddGameComment(ply int, comment string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.activeGame == nil || a.activeGame.RootMove == nil {
		return fmt.Errorf("no active game")
	}
	curr := a.activeGame.RootMove
	for p := 1; curr != nil && p < ply; p++ {
		curr = curr.Next
	}
	if curr == nil {
		return fmt.Errorf("ply %d not found in active game", ply)
	}
	curr.Comment = comment
	return nil
}

// SetGameNAG sets or adds a Numeric Annotation Glyph to the move at the given 1-based ply.
func (a *App) SetGameNAG(ply int, nag int) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.activeGame == nil || a.activeGame.RootMove == nil {
		return fmt.Errorf("no active game")
	}
	curr := a.activeGame.RootMove
	for p := 1; curr != nil && p < ply; p++ {
		curr = curr.Next
	}
	if curr == nil {
		return fmt.Errorf("ply %d not found in active game", ply)
	}
	for _, existing := range curr.NAGs {
		if existing == nag {
			return nil
		}
	}
	curr.NAGs = append(curr.NAGs, nag)
	return nil
}

// ClearGameNAGs removes all NAGs from the move at the given 1-based ply.
func (a *App) ClearGameNAGs(ply int) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.activeGame == nil || a.activeGame.RootMove == nil {
		return fmt.Errorf("no active game")
	}
	curr := a.activeGame.RootMove
	for p := 1; curr != nil && p < ply; p++ {
		curr = curr.Next
	}
	if curr == nil {
		return fmt.Errorf("ply %d not found in active game", ply)
	}
	curr.NAGs = nil
	return nil
}

// PromoteVariation promotes a variation at the specified ply to become the main line.
func (a *App) PromoteVariation(ply int, varIndex int) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.activeGame == nil || a.activeGame.RootMove == nil {
		return fmt.Errorf("no active game")
	}
	if ply <= 1 {
		if varIndex < 0 || varIndex >= len(a.activeGame.RootMove.Variations) {
			return fmt.Errorf("variation index %d out of range", varIndex)
		}
		oldMain := a.activeGame.RootMove
		newMain := oldMain.Variations[varIndex]
		oldMain.Variations = append(oldMain.Variations[:varIndex], oldMain.Variations[varIndex+1:]...)
		newMain.Variations = append(newMain.Variations, oldMain)
		a.activeGame.RootMove = newMain
		return nil
	}

	parent := a.activeGame.RootMove
	for p := 1; parent != nil && p < ply-1; p++ {
		parent = parent.Next
	}
	if parent == nil || parent.Next == nil {
		return fmt.Errorf("ply %d not found", ply)
	}
	targetNode := parent.Next
	if varIndex < 0 || varIndex >= len(targetNode.Variations) {
		return fmt.Errorf("variation index %d out of range", varIndex)
	}
	promotedNode := targetNode.Variations[varIndex]
	targetNode.Variations = append(targetNode.Variations[:varIndex], targetNode.Variations[varIndex+1:]...)
	promotedNode.Variations = append(promotedNode.Variations, targetNode)
	parent.Next = promotedNode
	return nil
}

// DeleteVariation removes the specified variation branch at ply.
func (a *App) DeleteVariation(ply int, varIndex int) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.activeGame == nil || a.activeGame.RootMove == nil {
		return fmt.Errorf("no active game")
	}
	curr := a.activeGame.RootMove
	for p := 1; curr != nil && p < ply; p++ {
		curr = curr.Next
	}
	if curr == nil {
		return fmt.Errorf("ply %d not found", ply)
	}
	if varIndex < 0 || varIndex >= len(curr.Variations) {
		return fmt.Errorf("variation index %d out of range", varIndex)
	}
	curr.Variations = append(curr.Variations[:varIndex], curr.Variations[varIndex+1:]...)
	return nil
}

// ==========================================
// Player Import & Dedicated / Blended Sync
// ==========================================

// ImportPlayerGames downloads games for any player and merges them into dedicated, active, or blended databases.
func (a *App) ImportPlayerGames(opts online.ImportPlayerOptions) (*online.SyncResult, error) {
	a.mu.Lock()
	activeDB := a.db
	a.mu.Unlock()

	opts.ProgressFn = func(stage string, current, total int, message string) {
		a.emitEvent("online:fetch-progress", map[string]interface{}{
			"stage":   stage,
			"current": current,
			"total":   total,
			"message": message,
		})
	}

	return a.syncMgr.ImportPlayerGames(opts, activeDB)
}

// ==========================================
// Player Simulation & Training Module
// ==========================================

// GetDatabasePlayers returns a sorted list of unique player names in the active database.
func (a *App) GetDatabasePlayers() ([]string, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.db == nil || !a.db.IsOpen {
		return nil, fmt.Errorf("no database open")
	}

	players := make([]string, 0, len(a.db.Names.NamesByID[scid.NamePlayer]))
	for _, name := range a.db.Names.NamesByID[scid.NamePlayer] {
		if name != "" && name != "?" {
			players = append(players, name)
		}
	}
	sort.Strings(players)
	return players, nil
}

// GetPlayerPersonality builds a repertoire and calibrated personality model for playerName from the database.
func (a *App) GetPlayerPersonality(playerName string) (*playerbot.PlayerPersonality, error) {
	a.mu.RLock()
	db := a.db
	a.mu.RUnlock()

	if db == nil || !db.IsOpen {
		return nil, fmt.Errorf("no database open")
	}

	return playerbot.BuildPersonality(db, playerName)
}

// GetSimulatedMove queries the player's personal opening book or generates an Elo-calibrated move.
func (a *App) GetSimulatedMove(playerName string, fen string, isWhite bool) (*playerbot.SimulatedMove, error) {
	personality, err := a.GetPlayerPersonality(playerName)
	if err != nil {
		return nil, err
	}

	// 1. Check personal opening book
	if simMove, inBook := personality.SelectSimulatedMove(fen, isWhite); inBook {
		return simMove, nil
	}

	// 2. Out of book: calibrate engine with player's estimated Elo
	limit, uciElo, skill := playerbot.CalibrateEngineSettings(personality.EstimatedElo)

	a.mu.Lock()
	eng := a.engine
	a.mu.Unlock()

	if eng != nil {
		if limit {
			_ = eng.SetOption("UCI_LimitStrength", "true")
			_ = eng.SetOption("UCI_Elo", strconv.Itoa(uciElo))
		}
		_ = eng.SetOption("Skill Level", strconv.Itoa(skill))

		bestUCI, _, err := eng.SearchBestMove(fen, nil, 800)
		if err == nil && bestUCI != "" {
			board, _ := chess.ParseFEN(fen)
			san := bestUCI
			if board != nil {
				if m, err := board.ParseUCI(bestUCI); err == nil {
					san = board.MoveToSAN(m)
				}
			}
			return &playerbot.SimulatedMove{
				UCI:        bestUCI,
				SAN:        san,
				IsBook:     false,
				EloTarget:  personality.EstimatedElo,
				Commentary: fmt.Sprintf("🧠 %s calculated move at calibrated %d Elo", personality.Name, personality.EstimatedElo),
			}, nil
		}
	}

	// Fallback if engine is not running: pick a legal move
	b, err := chess.ParseFEN(fen)
	if err != nil {
		return nil, err
	}
	moves := b.GenerateLegalMoves()
	if len(moves) == 0 {
		return nil, fmt.Errorf("no legal moves available")
	}
	m := moves[rand.Intn(len(moves))]
	return &playerbot.SimulatedMove{
		UCI:        m.String(),
		SAN:        b.MoveToSAN(m),
		IsBook:     false,
		EloTarget:  personality.EstimatedElo,
		Commentary: fmt.Sprintf("🎲 %s played fallback move", personality.Name),
	}, nil
}

// ==========================================
// Persona Management & Multi-Alias Aggregation
// ==========================================

// GetPersonas returns all configured personas sorted by display name.
func (a *App) GetPersonas() ([]*persona.Persona, error) {
	if a.personaMgr == nil {
		return nil, fmt.Errorf("persona manager not initialized")
	}
	return a.personaMgr.GetPersonas(), nil
}

// GetPersona retrieves a persona by its unique ID.
func (a *App) GetPersona(id string) (*persona.Persona, error) {
	if a.personaMgr == nil {
		return nil, fmt.Errorf("persona manager not initialized")
	}
	return a.personaMgr.GetPersona(id)
}

// SavePersona creates or updates a unified persona.
func (a *App) SavePersona(p *persona.Persona) (*persona.Persona, error) {
	if a.personaMgr == nil {
		return nil, fmt.Errorf("persona manager not initialized")
	}
	saved, err := a.personaMgr.SavePersona(p)
	if err != nil {
		return nil, err
	}
	_, _ = a.personaMgr.EnsurePersonaDatabase(saved)
	return saved, nil
}

// DeletePersona removes a persona by ID.
func (a *App) DeletePersona(id string) error {
	if a.personaMgr == nil {
		return fmt.Errorf("persona manager not initialized")
	}
	return a.personaMgr.DeletePersona(id)
}

// AddPersonaIdentifier adds a new database designator to the persona.
func (a *App) AddPersonaIdentifier(personaID string, identifier string) (*persona.Persona, error) {
	if a.personaMgr == nil {
		return nil, fmt.Errorf("persona manager not initialized")
	}
	return a.personaMgr.AddIdentifier(personaID, identifier)
}

// RemovePersonaIdentifier removes a database designator from the persona.
func (a *App) RemovePersonaIdentifier(personaID string, identifier string) (*persona.Persona, error) {
	if a.personaMgr == nil {
		return nil, fmt.Errorf("persona manager not initialized")
	}
	return a.personaMgr.RemoveIdentifier(personaID, identifier)
}

// SuggestPersonaIdentifiers queries the active database for player names matching query.
func (a *App) SuggestPersonaIdentifiers(query string, existing []string) ([]string, error) {
	a.mu.RLock()
	db := a.db
	a.mu.RUnlock()

	if db == nil || !db.IsOpen {
		return []string{}, nil
	}
	return persona.SuggestIdentifiers(db, query, existing), nil
}

// GetPersonaStatistics calculates aggregated statistics across all identifiers for a persona from active DB or dedicated base.
func (a *App) GetPersonaStatistics(personaID string) (*persona.PersonaStatistics, error) {
	if a.personaMgr == nil {
		return nil, fmt.Errorf("persona manager not initialized")
	}
	p, err := a.personaMgr.GetPersona(personaID)
	if err != nil {
		return nil, err
	}

	a.mu.RLock()
	db := a.db
	a.mu.RUnlock()

	// If no database is open, try opening persona's dedicated database
	if (db == nil || !db.IsOpen) && p.DatabasePath != "" {
		dedicatedDB := &scid.Database{}
		if err := dedicatedDB.Open(p.DatabasePath); err == nil {
			defer dedicatedDB.Close()
			return persona.CalculatePersonaStatistics(dedicatedDB, p), nil
		}
	}

	if db == nil || !db.IsOpen {
		return &persona.PersonaStatistics{
			PersonaID:   p.ID,
			DisplayName: p.DisplayName,
		}, nil
	}

	stats := persona.CalculatePersonaStatistics(db, p)
	// If the active database has no games for this persona, check dedicated database
	if stats.TotalGames == 0 && p.DatabasePath != "" {
		dedicatedDB := &scid.Database{}
		if err := dedicatedDB.Open(p.DatabasePath); err == nil {
			defer dedicatedDB.Close()
			dedStats := persona.CalculatePersonaStatistics(dedicatedDB, p)
			if dedStats.TotalGames > 0 {
				return dedStats, nil
			}
		}
	}

	return stats, nil
}

// FetchPersonaOnlineGames fetches games for a persona from Chess.com and/or Lichess directly into their dedicated database.
func (a *App) FetchPersonaOnlineGames(personaID string, maxGames int) (*online.SyncResult, error) {
	if a.personaMgr == nil {
		return nil, fmt.Errorf("persona manager not initialized")
	}
	if a.syncMgr == nil {
		return nil, fmt.Errorf("sync manager not initialized")
	}

	p, err := a.personaMgr.GetPersona(personaID)
	if err != nil {
		return nil, err
	}

	chessComUser := strings.TrimSpace(p.ChessComUsername)
	lichessUser := strings.TrimSpace(p.LichessUsername)

	if chessComUser == "" && lichessUser == "" {
		return nil, fmt.Errorf("no Chess.com or Lichess username specified for persona %q", p.DisplayName)
	}

	basePath, err := a.personaMgr.EnsurePersonaDatabase(p)
	if err != nil {
		return nil, fmt.Errorf("ensuring persona database: %w", err)
	}

	opts := online.ImportPlayerOptions{
		PlayerName:       p.DisplayName,
		ChessComUsername: chessComUser,
		LichessUsername:  lichessUser,
		TargetMode:       online.TargetDedicatedBase,
		DedicatedPath:    basePath,
		AutoDeduplicate:  true,
		MaxGames:         maxGames,
		ProgressFn: func(stage string, current, total int, message string) {
			a.emitEvent("online:fetch-progress", map[string]interface{}{
				"personaId": personaID,
				"stage":     stage,
				"current":   current,
				"total":     total,
				"message":   message,
			})
		},
	}

	a.mu.RLock()
	activeDB := a.db
	a.mu.RUnlock()

	res, err := a.syncMgr.ImportPlayerGames(opts, activeDB)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// AggregatePersonaGames extracts all matching games from the active database into the persona's dedicated database.
func (a *App) AggregatePersonaGames(personaID string) (*persona.AggregateResult, error) {
	if a.personaMgr == nil {
		return nil, fmt.Errorf("persona manager not initialized")
	}

	a.mu.RLock()
	db := a.db
	a.mu.RUnlock()

	if db == nil || !db.IsOpen {
		return nil, fmt.Errorf("no database open to aggregate games from")
	}

	return a.personaMgr.AggregateGames(db, personaID)
}

// OpenPersonaDatabase opens the persona's dedicated database as the active database in NewScid.
func (a *App) OpenPersonaDatabase(personaID string) (*DatabaseInfo, error) {
	if a.personaMgr == nil {
		return nil, fmt.Errorf("persona manager not initialized")
	}
	p, err := a.personaMgr.GetPersona(personaID)
	if err != nil {
		return nil, err
	}
	basePath, err := a.personaMgr.EnsurePersonaDatabase(p)
	if err != nil {
		return nil, err
	}
	return a.OpenDatabase(basePath)
}

// GetPersonaPersonality builds a playerbot personality using all identifiers of the persona.
func (a *App) GetPersonaPersonality(personaID string) (*playerbot.PlayerPersonality, error) {
	if a.personaMgr == nil {
		return nil, fmt.Errorf("persona manager not initialized")
	}
	p, err := a.personaMgr.GetPersona(personaID)
	if err != nil {
		return nil, err
	}

	a.mu.RLock()
	db := a.db
	a.mu.RUnlock()

	// If active database is open and contains games for this persona, use it
	if db != nil && db.IsOpen {
		personality, err := playerbot.BuildPersonalityFromIdentifiers(db, p.DisplayName, p.Identifiers)
		if err == nil && personality.TotalGames > 0 {
			if p.EstimatedElo > 0 {
				personality.EstimatedElo = p.EstimatedElo
			}
			return personality, nil
		}
	}

	// Otherwise, check if dedicated persona database has games
	if p.DatabasePath != "" {
		dedicatedDB := &scid.Database{}
		if err := dedicatedDB.Open(p.DatabasePath); err == nil {
			defer dedicatedDB.Close()
			personality, err := playerbot.BuildPersonalityFromIdentifiers(dedicatedDB, p.DisplayName, p.Identifiers)
			if err == nil && personality.TotalGames > 0 {
				if p.EstimatedElo > 0 {
					personality.EstimatedElo = p.EstimatedElo
				}
				return personality, nil
			}
		}
	}

	if db != nil && db.IsOpen {
		return playerbot.BuildPersonalityFromIdentifiers(db, p.DisplayName, p.Identifiers)
	}

	return nil, fmt.Errorf("no games found for persona %q in open or dedicated databases", p.DisplayName)
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

// GetTournamentEvents returns discovered tournament events in the currently open database.
func (a *App) GetTournamentEvents() ([]*tournament.EventSummary, error) {
	a.mu.RLock()
	db := a.db
	a.mu.RUnlock()

	if db == nil || !db.IsOpen {
		return nil, fmt.Errorf("no database is currently open")
	}

	return tournament.DiscoverEvents(db)
}

// GetTournamentCrosstable calculates standings, matrix, and tiebreak metrics for a given event in the open database.
func (a *App) GetTournamentCrosstable(eventName string, site string, date string) (*tournament.TournamentCrosstable, error) {
	a.mu.RLock()
	db := a.db
	a.mu.RUnlock()

	if db == nil || !db.IsOpen {
		return nil, fmt.Errorf("no database is currently open")
	}

	return tournament.CalculateCrosstable(db, eventName, site, date)
}

// ExportCrosstableText returns the ASCII crosstable text for an event in the open database.
func (a *App) ExportCrosstableText(eventName string, site string, date string) (string, error) {
	ct, err := a.GetTournamentCrosstable(eventName, site, date)
	if err != nil {
		return "", err
	}
	return ct.CrossTableText, nil
}

