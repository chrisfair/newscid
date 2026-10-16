package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"newscid/pkg/blundervault"
	"newscid/pkg/chess"
	"newscid/pkg/chess960"
	"newscid/pkg/database/scid"
	"newscid/pkg/database/sciddriver"
	"newscid/pkg/eco"
	"newscid/pkg/engine"
	"newscid/pkg/epd"
	"newscid/pkg/explainer"
	"newscid/pkg/fairplay"
	"newscid/pkg/gameexport"
	"newscid/pkg/imbalance"
	"newscid/pkg/online"
	"newscid/pkg/persona"
	"newscid/pkg/playerbot"
	"newscid/pkg/polyglot"
	"newscid/pkg/puzzleminer"
	"newscid/pkg/repertoire"
	"newscid/pkg/scout"
	"newscid/pkg/sets"
	"newscid/pkg/stats"
	"newscid/pkg/support"
	"newscid/pkg/tablebase"
	"newscid/pkg/tournament"
	"newscid/pkg/training"
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
	MoveUCI     string   `json:"moveUci,omitempty"`
	MoveSAN     string   `json:"moveSan,omitempty"`
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

// EventListener receives events emitted by App when running in server or mobile mode.
type EventListener = func(eventName string, optionalData ...interface{})

// App struct
type App struct {
	ctx              context.Context
	mu               sync.RWMutex
	listenerMu       sync.RWMutex
	eventListener    EventListener
	db               *scid.Database
	eco              *eco.Classifier
	driver           *sciddriver.Driver
	engine           *engine.Engine
	activeGame       *chess.Game
	repMgr           *repertoire.Manager
	profileMgr       *online.ProfileManager
	syncMgr          *online.SyncManager
	personaMgr       *persona.PersonaManager
	personaBotCache  map[string]*playerbot.PlayerPersonality
	polyglotBooks    map[string]*polyglot.Book
	tablebaseClient  *tablebase.Client
	solitaireSession *training.SolitaireSession
	twicClient       *online.TWICClient
	matchCancel      context.CancelFunc
	matchRunning     bool
	engineMultiPV    int
	engineThreads    int
	engineHashMB     int
	notationStyle    chess.NotationStyle
	blunderVault     *blundervault.Vault
	supportMgr       *support.Manager
}

// NewApp creates a new App application struct
func NewApp() *App {
	classifier := eco.NewClassifier()
	// Attempt to load system scid.eco if available
	if _, err := os.Stat("/usr/share/scid/data/scid.eco"); err == nil {
		_ = classifier.LoadFromFile("/usr/share/scid/data/scid.eco")
	}

	profileMgr, _ := online.NewProfileManager("")
	personaMgr, _ := persona.NewPersonaManager("")

	configDir, _ := os.UserConfigDir()
	vaultPath := filepath.Join(configDir, "newscid", "blunder_vault.json")
	bVault, _ := blundervault.NewVault(vaultPath)
	supportPath := filepath.Join(configDir, "newscid", "support.json")
	supMgr, _ := support.NewManager(supportPath)

	return &App{
		db:              scid.NewDatabase(),
		eco:             classifier,
		driver:          sciddriver.NewDriver(),
		repMgr:          repertoire.NewManager(""),
		profileMgr:      profileMgr,
		syncMgr:         online.NewSyncManager(profileMgr),
		twicClient:      online.NewTWICClient(),
		personaMgr:      personaMgr,
		personaBotCache: make(map[string]*playerbot.PlayerPersonality),
		polyglotBooks:   make(map[string]*polyglot.Book),
		tablebaseClient: tablebase.NewClient(),
		engineMultiPV:   1,
		blunderVault:    bVault,
		supportMgr:      supMgr,
	}
}

// startup is called when the app starts.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// shutdown is called when the app terminates.
func (a *App) shutdown(ctx context.Context) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.engine != nil {
		_ = a.engine.Quit()
		a.engine = nil
	}
	if a.db != nil && a.db.IsOpen {
		_ = a.db.Close()
	}
}

// SetEventListener attaches a callback to receive runtime events (for SSE / WebSockets).
func (a *App) SetEventListener(l EventListener) {
	a.listenerMu.Lock()
	defer a.listenerMu.Unlock()
	a.eventListener = l
}

func (a *App) emitEvent(eventName string, optionalData ...interface{}) {
	if a.ctx != nil && a.ctx.Value("events") != nil {
		runtime.EventsEmit(a.ctx, eventName, optionalData...)
	}
	a.listenerMu.RLock()
	listener := a.eventListener
	a.listenerMu.RUnlock()
	if listener != nil {
		listener(eventName, optionalData...)
	}
}

// DispatchRPC invokes a method on App by name with JSON-encoded parameters.
// This allows full RPC communication over HTTP/WebSocket for mobile/Android/browser clients.
func (a *App) DispatchRPC(method string, params []json.RawMessage) (interface{}, error) {
	val := reflect.ValueOf(a)
	m := val.MethodByName(method)
	if !m.IsValid() {
		return nil, fmt.Errorf("method not found: %s", method)
	}

	mType := m.Type()
	numIn := mType.NumIn()

	if mType.IsVariadic() {
		if len(params) < numIn-1 {
			return nil, fmt.Errorf("insufficient parameters for variadic method %s: expected at least %d, got %d", method, numIn-1, len(params))
		}
	} else if len(params) != numIn {
		return nil, fmt.Errorf("parameter count mismatch for %s: expected %d, got %d", method, numIn, len(params))
	}

	inArgs := make([]reflect.Value, numIn)
	for i := 0; i < numIn; i++ {
		targetType := mType.In(i)
		elemVal := reflect.New(targetType)
		if i < len(params) && len(params[i]) > 0 && string(params[i]) != "null" {
			if err := json.Unmarshal(params[i], elemVal.Interface()); err != nil {
				return nil, fmt.Errorf("invalid argument %d for %s: %w", i, method, err)
			}
		}
		inArgs[i] = elemVal.Elem()
	}

	out := m.Call(inArgs)
	if len(out) == 0 {
		return nil, nil
	}

	// If last return value implements error, inspect it
	lastOut := out[len(out)-1]
	if lastOut.Type().Implements(reflect.TypeOf((*error)(nil)).Elem()) {
		if !lastOut.IsNil() {
			return nil, lastOut.Interface().(error)
		}
		out = out[:len(out)-1]
	}

	if len(out) == 1 {
		return out[0].Interface(), nil
	} else if len(out) > 1 {
		results := make([]interface{}, len(out))
		for i, v := range out {
			results[i] = v.Interface()
		}
		return results, nil
	}

	return nil, nil
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
	if a.db == nil {
		return nil
	}
	return a.db.Close()
}

func (a *App) GetDatabaseInfo() (*DatabaseInfo, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.db == nil || !a.db.IsOpen {
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

	if a.db == nil || !a.db.IsOpen {
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

	if a.db == nil || !a.db.IsOpen {
		return 0, nil
	}

	return a.db.CountMatches(criteria), nil
}

func (a *App) GetGame(gameID int) (*chess.Game, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.db == nil || !a.db.IsOpen {
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

	if a.db == nil || !a.db.IsOpen {
		return 0, fmt.Errorf("no database open")
	}

	return a.db.SaveGame(game)
}

func (a *App) DeleteGame(gameID int, deleted bool) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.db == nil || !a.db.IsOpen {
		return fmt.Errorf("no database open")
	}

	return a.db.DeleteGame(gameID, deleted)
}

// UpdateGameHeaders updates game metadata tags in the database and active game.
func (a *App) UpdateGameHeaders(gameID int, tags map[string]string) (*scid.GameSummary, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.db == nil || !a.db.IsOpen {
		return nil, fmt.Errorf("no database open")
	}

	if err := a.db.UpdateGameTags(gameID, tags); err != nil {
		return nil, err
	}

	if a.activeGame != nil && a.activeGame.ID == gameID {
		for k, v := range tags {
			a.activeGame.SetTag(k, v)
		}
		if res, ok := tags["Result"]; ok && res != "" {
			a.activeGame.Result = chess.Result(res)
		}
	}

	return a.db.GetGameSummary(gameID)
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

func (a *App) MakeMove(fen string, moveStr string) (*BoardState, error) {
	b, err := chess.ParseFEN(fen)
	if err != nil {
		return nil, err
	}

	m, err := b.ParseUCI(moveStr)
	var moveSAN string
	if err != nil {
		m, err = b.ParseSAN(moveStr)
		if err != nil {
			return nil, fmt.Errorf("invalid move %q: %w", moveStr, err)
		}
		moveSAN = moveStr
	} else {
		moveSAN = b.MoveToSAN(m)
	}

	b.MakeMove(m)
	state := makeBoardState(b)
	state.MoveUCI = m.String()
	state.MoveSAN = moveSAN
	return state, nil
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
	db := a.db
	a.mu.RUnlock()

	var res *tree.TreeResult
	var err error
	if db != nil && db.IsOpen {
		res, err = tree.BuildTreeFromDatabase(db, fen, 1000)
		if err != nil {
			return nil, err
		}
	}

	// If database is open and has matching moves, return it
	if res != nil && len(res.Moves) > 0 {
		return res, nil
	}

	// Fallback to Polyglot Master Opening Book (Performance.bin or Elo2400.bin) if DB has 0 moves or is not open
	books := polyglot.DiscoverBooks()
	if len(books) > 0 {
		preferredBook := books[0]
		for _, b := range books {
			if strings.Contains(b, "Performance.bin") {
				preferredBook = b
				break
			}
			if strings.Contains(b, "Elo2400.bin") {
				preferredBook = b
			}
		}

		bMoves, err := a.QueryPolyglotBook(fen, preferredBook)
		if err == nil && len(bMoves) > 0 {
			var treeMoves []*tree.MoveStat
			totalWeight := 0
			for _, bm := range bMoves {
				totalWeight += int(bm.Weight)
			}
			for _, bm := range bMoves {
				pct := 0.0
				if totalWeight > 0 {
					pct = (float64(bm.Weight) / float64(totalWeight)) * 100.0
				}
				treeMoves = append(treeMoves, &tree.MoveStat{
					Move:     bm.SAN,
					UCI:      bm.UCI,
					Count:    int(bm.Weight),
					ScorePct: pct,
				})
			}
			return &tree.TreeResult{
				FEN:        fen,
				TotalGames: totalWeight,
				Moves:      treeMoves,
			}, nil
		}
	}

	if res != nil {
		return res, nil
	}
	return &tree.TreeResult{FEN: fen, Moves: []*tree.MoveStat{}}, nil
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

	results := make([]*scid.GameSummary, 0)
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
	if a.activeGame == nil {
		a.mu.Unlock()
		return nil, fmt.Errorf("no active game to annotate")
	}
	if err := a.ensureEngineLocked(); err != nil {
		a.mu.Unlock()
		return nil, err
	}
	eng := a.engine
	g := a.activeGame
	a.mu.Unlock()

	return engine.AnnotateGame(g, eng, movetimeMs, func(ply, total int, ma engine.MoveAnalysis) {
		pct := 0
		if total > 0 {
			pct = int(float64(ply) / float64(total) * 100)
		}
		a.emitEvent("annotate:progress", map[string]interface{}{
			"currentPly": ply,
			"totalPlies": total,
			"percent":    pct,
			"san":        ma.SAN,
			"scoreCp":    ma.ScoreCp,
		})
	})
}

func (a *App) AnnotatePGN(pgnContent string, movetimeMs int) (*engine.GameAnalysis, error) {
	g, err := chess.ParsePGN(pgnContent)
	if err != nil {
		return nil, err
	}

	a.mu.Lock()
	a.activeGame = g
	if err := a.ensureEngineLocked(); err != nil {
		a.mu.Unlock()
		return nil, err
	}
	eng := a.engine
	a.mu.Unlock()

	return engine.AnnotateGame(g, eng, movetimeMs, func(ply, total int, ma engine.MoveAnalysis) {
		pct := 0
		if total > 0 {
			pct = int(float64(ply) / float64(total) * 100)
		}
		a.emitEvent("annotate:progress", map[string]interface{}{
			"currentPly": ply,
			"totalPlies": total,
			"percent":    pct,
			"san":        ma.SAN,
			"scoreCp":    ma.ScoreCp,
		})
	})
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

// ShortenRepertoire applies the Pareto 80/20 pruning algorithm to condense opening repertoires.
func (a *App) ShortenRepertoire(opts repertoire.ShortenerOptions) (*repertoire.ShortenerResult, error) {
	a.mu.RLock()
	db := a.db
	a.mu.RUnlock()

	return a.repMgr.ShortenRepertoire(opts, db)
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
// TWIC (The Week In Chess) Bulletins
// ==========================================

// DownloadTWICIssue downloads and imports a single TWIC issue into the currently active database.
func (a *App) DownloadTWICIssue(issue int) (*online.TWICIssueResult, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.db == nil || !a.db.IsOpen {
		return nil, fmt.Errorf("no database is currently open; open or create a database first")
	}

	return a.twicClient.ImportIssue(context.Background(), issue, a.db)
}

// DownloadTWICRange downloads and imports a range of TWIC issues [startIssue..endIssue] into the active database.
func (a *App) DownloadTWICRange(startIssue, endIssue int) (*online.TWICBatchResult, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.db == nil || !a.db.IsOpen {
		return nil, fmt.Errorf("no database is currently open; open or create a database first")
	}

	progressFn := func(currentIssue, processedIssues, totalIssues, gamesImported int) {
		a.emitEvent("twic:progress", map[string]interface{}{
			"currentIssue":    currentIssue,
			"processedIssues": processedIssues,
			"totalIssues":     totalIssues,
			"gamesImported":   gamesImported,
		})
	}

	return a.twicClient.ImportRange(context.Background(), startIssue, endIssue, a.db, progressFn)
}

// CheckTWICIssue checks if a particular TWIC issue exists on the server.
func (a *App) CheckTWICIssue(issue int) (bool, error) {
	return a.twicClient.CheckIssueExists(context.Background(), issue)
}

// GetLatestTWICIssue automatically detects the latest published TWIC issue number.
func (a *App) GetLatestTWICIssue() (int, error) {
	return a.twicClient.FindLatestIssue(context.Background(), 1560)
}

// ==========================================
// Engine vs. Engine Match & Tournament
// ==========================================

// StartEngineMatch launches an automated engine vs engine match.
func (a *App) StartEngineMatch(cfg tournament.EngineMatchConfig) (*tournament.EngineMatchSummary, error) {
	a.mu.Lock()
	if a.matchRunning {
		a.mu.Unlock()
		return nil, fmt.Errorf("an engine match is already currently running")
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.matchCancel = cancel
	a.matchRunning = true
	targetDB := a.db
	a.mu.Unlock()

	defer func() {
		a.mu.Lock()
		a.matchRunning = false
		a.matchCancel = nil
		a.mu.Unlock()
	}()

	// 1. Resolve Player 1
	var p1 tournament.EnginePlayer
	var err error
	if cfg.Engine1.Path != "" {
		p1, err = tournament.NewUCIEnginePlayer(cfg.Engine1.Name, cfg.Engine1.Path, cfg.Engine1.Options)
		if err != nil {
			return nil, fmt.Errorf("initializing Engine 1: %w", err)
		}
		defer p1.Close()
	} else {
		p1 = tournament.NewMockEnginePlayer(cfg.Engine1.Name)
	}

	// 2. Resolve Player 2
	var p2 tournament.EnginePlayer
	if cfg.Engine2.Path != "" {
		p2, err = tournament.NewUCIEnginePlayer(cfg.Engine2.Name, cfg.Engine2.Path, cfg.Engine2.Options)
		if err != nil {
			return nil, fmt.Errorf("initializing Engine 2: %w", err)
		}
		defer p2.Close()
	} else {
		p2 = tournament.NewMockEnginePlayer(cfg.Engine2.Name)
	}

	runner := tournament.NewEngineMatchRunner(p1, p2, cfg, targetDB)

	onProgress := func(prog tournament.EngineMatchProgress) {
		a.emitEvent("enginematch:progress", prog)
	}
	onGameEnd := func(game *tournament.EngineGameResult, p1Score, p2Score float64) {
		a.emitEvent("enginematch:gameend", map[string]interface{}{
			"game":         game,
			"engine1Score": p1Score,
			"engine2Score": p2Score,
		})
	}

	return runner.Run(ctx, onProgress, onGameEnd)
}

// StopEngineMatch aborts any currently active engine match.
func (a *App) StopEngineMatch() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.matchCancel != nil {
		a.matchCancel()
		a.matchRunning = false
		a.matchCancel = nil
		return true
	}
	return false
}

// IsEngineMatchRunning reports whether an automated engine match is currently executing.
func (a *App) IsEngineMatchRunning() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.matchRunning
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

	// 2. Out of book: ensure engine is started and calibrate engine with player's estimated Elo and style
	limit, uciElo, skill, contempt := personality.CalibrateEngineSettingsForPersona()

	a.mu.Lock()
	_ = a.ensureEngineLocked()
	eng := a.engine
	a.mu.Unlock()

	movetimeMs := 800
	if personality.EstimatedElo >= 2000 {
		movetimeMs = 1200
	} else if personality.EstimatedElo >= 1600 {
		movetimeMs = 1000
	}

	if eng != nil {
		if limit {
			_ = eng.SetOption("UCI_LimitStrength", "true")
			_ = eng.SetOption("UCI_Elo", strconv.Itoa(uciElo))
		}
		_ = eng.SetOption("Skill Level", strconv.Itoa(skill))
		if contempt != 0 {
			_ = eng.SetOption("Contempt", strconv.Itoa(contempt))
		}

		bestUCI, _, err := eng.SearchBestMove(fen, nil, movetimeMs)
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
				Commentary: fmt.Sprintf("🧠 %s (%s, %d Elo) calculated move", personality.Name, personality.PlayStyle, personality.EstimatedElo),
			}, nil
		}
	}

	// Smart tactical fallback if engine is unavailable: evaluates checks, captures, defenses, and center control
	b, err := chess.ParseFEN(fen)
	if err != nil {
		return nil, err
	}
	tacticalMove, err := playerbot.SelectTacticalMove(b, isWhite)
	if err != nil {
		return nil, err
	}
	return &playerbot.SimulatedMove{
		UCI:        tacticalMove.String(),
		SAN:        b.MoveToSAN(tacticalMove),
		IsBook:     false,
		EloTarget:  personality.EstimatedElo,
		Commentary: fmt.Sprintf("🛡️ %s (%d Elo) played tactical heuristic move", personality.Name, personality.EstimatedElo),
	}, nil
}

// GetPersonaSimulatedMove queries the persona's full game repertoire and returns their most common
// historical move. If out of their historical games, it computes an engine move calibrated to their
// human Elo, contempt rating, and tactical aggression style.
func (a *App) GetPersonaSimulatedMove(personaID string, fen string, isWhite bool) (*playerbot.SimulatedMove, error) {
	a.mu.Lock()
	if a.personaBotCache == nil {
		a.personaBotCache = make(map[string]*playerbot.PlayerPersonality)
	}
	personality := a.personaBotCache[personaID]
	a.mu.Unlock()

	var err error
	if personality == nil {
		personality, err = a.GetPersonaPersonality(personaID)
		if err != nil {
			return nil, err
		}
		a.mu.Lock()
		a.personaBotCache[personaID] = personality
		a.mu.Unlock()
	}

	// 1. Check personal full-game tree (plays most common response or sole response)
	if simMove, inBook := personality.SelectSimulatedMove(fen, isWhite); inBook {
		return simMove, nil
	}

	// 2. Out of personal game tree: ensure engine is started and calibrate settings with style, contempt, and Elo
	limit, uciElo, skill, contempt := personality.CalibrateEngineSettingsForPersona()

	a.mu.Lock()
	_ = a.ensureEngineLocked()
	eng := a.engine
	a.mu.Unlock()

	movetimeMs := 800
	if personality.EstimatedElo >= 2000 {
		movetimeMs = 1200
	} else if personality.EstimatedElo >= 1600 {
		movetimeMs = 1000
	}

	if eng != nil {
		if limit {
			_ = eng.SetOption("UCI_LimitStrength", "true")
			_ = eng.SetOption("UCI_Elo", strconv.Itoa(uciElo))
		}
		_ = eng.SetOption("Skill Level", strconv.Itoa(skill))
		if contempt != 0 {
			_ = eng.SetOption("Contempt", strconv.Itoa(contempt))
		}

		bestUCI, _, err := eng.SearchBestMove(fen, nil, movetimeMs)
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
				Commentary: fmt.Sprintf("🧠 %s (%s, %d Elo) calculated move", personality.Name, personality.PlayStyle, personality.EstimatedElo),
			}, nil
		}
	}

	// Smart tactical fallback if engine is unavailable: evaluates checks, captures, defenses, and center control
	b, err := chess.ParseFEN(fen)
	if err != nil {
		return nil, err
	}
	tacticalMove, err := playerbot.SelectTacticalMove(b, isWhite)
	if err != nil {
		return nil, err
	}
	return &playerbot.SimulatedMove{
		UCI:        tacticalMove.String(),
		SAN:        b.MoveToSAN(tacticalMove),
		IsBook:     false,
		EloTarget:  personality.EstimatedElo,
		Commentary: fmt.Sprintf("🛡️ %s (%d Elo) played tactical heuristic move", personality.Name, personality.EstimatedElo),
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
	a.mu.Lock()
	delete(a.personaBotCache, saved.ID)
	a.mu.Unlock()
	_, _ = a.personaMgr.EnsurePersonaDatabase(saved)
	return saved, nil
}

// DeletePersona removes a persona by ID.
func (a *App) DeletePersona(id string) error {
	if a.personaMgr == nil {
		return fmt.Errorf("persona manager not initialized")
	}
	a.mu.Lock()
	delete(a.personaBotCache, id)
	a.mu.Unlock()
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

	a.mu.Lock()
	delete(a.personaBotCache, personaID)
	a.mu.Unlock()

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

	res, err := a.personaMgr.AggregateGames(db, personaID)
	if err != nil {
		return nil, err
	}

	a.mu.Lock()
	delete(a.personaBotCache, personaID)
	a.mu.Unlock()

	return res, nil
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

// GetPolyglotBooks returns discovered Polyglot opening books (.bin) on the system.
func (a *App) GetPolyglotBooks() []string {
	return polyglot.DiscoverBooks()
}

// QueryPolyglotBook returns moves from a Polyglot book for the specified position FEN.
func (a *App) QueryPolyglotBook(fen string, bookPath string) ([]polyglot.BookMove, error) {
	if strings.TrimSpace(bookPath) == "" {
		return []polyglot.BookMove{}, nil
	}

	a.mu.Lock()
	if a.polyglotBooks == nil {
		a.polyglotBooks = make(map[string]*polyglot.Book)
	}
	book, ok := a.polyglotBooks[bookPath]
	var err error
	if !ok {
		book, err = polyglot.OpenBook(bookPath)
		if err != nil {
			a.mu.Unlock()
			return nil, err
		}
		a.polyglotBooks[bookPath] = book
	}
	a.mu.Unlock()

	var b *chess.Board
	if strings.TrimSpace(fen) == "" || fen == chess.StartingFEN {
		b = chess.NewBoard()
	} else {
		parsed, err := chess.ParseFEN(fen)
		if err != nil {
			return nil, fmt.Errorf("parsing FEN %q: %w", fen, err)
		}
		b = parsed
	}

	return book.FindMoves(b)
}

// GetPolyglotBookMove selects a book move according to weights or chooses the best move.
func (a *App) GetPolyglotBookMove(fen string, bookPath string, pickBest bool) (*polyglot.BookMove, error) {
	moves, err := a.QueryPolyglotBook(fen, bookPath)
	if err != nil {
		return nil, err
	}
	if len(moves) == 0 {
		return nil, nil
	}
	if pickBest || len(moves) == 1 {
		return &moves[0], nil
	}

	totalWeight := 0
	for _, m := range moves {
		totalWeight += m.Weight
	}
	if totalWeight <= 0 {
		return &moves[0], nil
	}

	r := rand.Intn(totalWeight)
	running := 0
	for i := range moves {
		running += moves[i].Weight
		if r < running {
			return &moves[i], nil
		}
	}
	return &moves[0], nil
}

// GetECOSections returns the 5 primary ECO classifications with game/line counts.
func (a *App) GetECOSections() []eco.SectionSummary {
	a.mu.RLock()
	classifier := a.eco
	a.mu.RUnlock()

	if classifier == nil {
		return nil
	}
	return classifier.GetSections()
}

// SearchECO queries ECO openings by section (A-E or ALL) and optional keywords.
func (a *App) SearchECO(section string, query string, limit int) []*eco.Entry {
	a.mu.RLock()
	classifier := a.eco
	a.mu.RUnlock()

	if classifier == nil {
		return nil
	}
	return classifier.Search(section, query, limit)
}

// ClassifyMoves identifies the ECO code and opening name for a sequence of moves.
func (a *App) ClassifyMoves(sans []string) *eco.Entry {
	a.mu.RLock()
	classifier := a.eco
	a.mu.RUnlock()

	if classifier == nil {
		return &eco.Entry{Code: "A00", Name: "Unknown"}
	}
	return classifier.Classify(sans)
}

// ProbeTablebase queries Syzygy endgame tablebases for a position (up to 7 pieces).
func (a *App) ProbeTablebase(fen string) (*tablebase.TablebaseResult, error) {
	a.mu.RLock()
	client := a.tablebaseClient
	a.mu.RUnlock()
	if client == nil {
		client = tablebase.NewClient()
	}
	return client.Probe(fen)
}

// StartGuessTheMove initializes a Solitaire Chess session for the given game and side.
func (a *App) StartGuessTheMove(gameID int, userSide string) (*training.SolitaireSession, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	var g *chess.Game
	if a.db != nil && a.db.IsOpen && gameID > 0 {
		game, err := a.db.GetGame(gameID)
		if err != nil {
			return nil, fmt.Errorf("loading game %d: %w", gameID, err)
		}
		g = game
	} else if a.activeGame != nil {
		g = a.activeGame
	} else {
		return nil, fmt.Errorf("no game loaded for Guess-the-Move training")
	}

	session, err := training.NewSolitaireSession(g, userSide)
	if err != nil {
		return nil, err
	}
	a.solitaireSession = session
	return session, nil
}

// GetGuessTheMoveState returns the current Solitaire Chess session if active.
func (a *App) GetGuessTheMoveState() (*training.SolitaireSession, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.solitaireSession == nil {
		return nil, fmt.Errorf("no active guess-the-move session")
	}
	return a.solitaireSession, nil
}

// SubmitGuessTheMove submits the user's move guess and returns evaluation feedback.
func (a *App) SubmitGuessTheMove(guess string) (*training.GuessFeedback, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.solitaireSession == nil {
		return nil, fmt.Errorf("no active guess-the-move session")
	}
	feedback := a.solitaireSession.SubmitGuess(guess)
	return feedback, nil
}

// GetGuessTheMoveHint requests a progressive hint for the current turn.
func (a *App) GetGuessTheMoveHint() (string, int, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.solitaireSession == nil {
		return "", 0, fmt.Errorf("no active guess-the-move session")
	}
	hintText, level := a.solitaireSession.GetHint()
	return hintText, level, nil
}

// SkipGuessTheMove reveals the actual move and advances to the next turn.
func (a *App) SkipGuessTheMove() (*training.GuessFeedback, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.solitaireSession == nil {
		return nil, fmt.Errorf("no active guess-the-move session")
	}
	feedback := a.solitaireSession.RevealAndSkip()
	return feedback, nil
}

// GetRatingEvolution calculates chronological rating progression for a player or persona.
func (a *App) GetRatingEvolution(playerOrPersona string) (*stats.RatingTrackerResult, error) {
	a.mu.RLock()
	db := a.db
	pMgr := a.personaMgr
	a.mu.RUnlock()

	if db == nil || !db.IsOpen {
		return nil, fmt.Errorf("no database open")
	}

	aliases := []string{playerOrPersona}
	if pMgr != nil {
		if p, err := pMgr.GetPersona(playerOrPersona); err == nil && p != nil {
			aliases = append([]string{p.DisplayName}, p.Identifiers...)
			if p.ChessComUsername != "" {
				aliases = append(aliases, p.ChessComUsername)
			}
			if p.LichessUsername != "" {
				aliases = append(aliases, p.LichessUsername)
			}
		}
	}

	return stats.CalculateRatingTracker(db, aliases)
}

// GetDatabaseDemographics returns demographic distributions for the active database.
func (a *App) GetDatabaseDemographics() (*stats.DatabaseDemographics, error) {
	a.mu.RLock()
	db := a.db
	a.mu.RUnlock()

	if db == nil || !db.IsOpen {
		return nil, fmt.Errorf("no database open")
	}

	return stats.CalculateDatabaseDemographics(db)
}

// StandardizePlayerNames applies spelling standardizations across the active database.
func (a *App) StandardizePlayerNames() (*scid.SpellCheckSummary, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.db == nil || !a.db.IsOpen {
		return nil, fmt.Errorf("no database open")
	}

	return a.db.StandardizePlayerNames()
}

// ExportCurrentGameHTML exports the currently loaded game as standalone HTML.
func (a *App) ExportCurrentGameHTML(opts gameexport.HTMLExportOptions) (string, error) {
	a.mu.RLock()
	g := a.activeGame
	a.mu.RUnlock()

	if g == nil {
		return "", fmt.Errorf("no active game loaded")
	}

	return gameexport.ExportGameHTML(g, opts)
}

// ExportGameHTML exports a specific game by index as standalone HTML.
func (a *App) ExportGameHTML(gameNum int, opts gameexport.HTMLExportOptions) (string, error) {
	a.mu.RLock()
	db := a.db
	a.mu.RUnlock()

	if db == nil || !db.IsOpen {
		return "", fmt.Errorf("no database open")
	}

	g, err := db.GetGame(gameNum)
	if err != nil {
		return "", fmt.Errorf("failed to load game %d: %w", gameNum, err)
	}

	return gameexport.ExportGameHTML(g, opts)
}

// ExportCurrentGameLaTeX exports the currently loaded game as a LaTeX document.
func (a *App) ExportCurrentGameLaTeX(opts gameexport.LaTeXExportOptions) (string, error) {
	a.mu.RLock()
	g := a.activeGame
	a.mu.RUnlock()

	if g == nil {
		return "", fmt.Errorf("no active game loaded")
	}

	return gameexport.ExportGameLaTeX(g, opts)
}

// ExportGameLaTeX exports a specific game by index as a LaTeX document.
func (a *App) ExportGameLaTeX(gameNum int, opts gameexport.LaTeXExportOptions) (string, error) {
	a.mu.RLock()
	db := a.db
	a.mu.RUnlock()

	if db == nil || !db.IsOpen {
		return "", fmt.Errorf("no database open")
	}

	g, err := db.GetGame(gameNum)
	if err != nil {
		return "", fmt.Errorf("failed to load game %d: %w", gameNum, err)
	}

	return gameexport.ExportGameLaTeX(g, opts)
}

// SaveExportFile writes exported content (HTML, LaTeX, PGN, etc.) to a file on disk.
func (a *App) SaveExportFile(filePath, content string) error {
	if filePath == "" {
		return fmt.Errorf("file path cannot be empty")
	}
	return os.WriteFile(filePath, []byte(content), 0644)
}

// ==========================================
// Chess960 (Fischer Random Chess)
// ==========================================

// GetChess960Position generates the Chess960 position for a given index (0-959).
func (a *App) GetChess960Position(id int) (*chess960.Position, error) {
	return chess960.GeneratePosition(id)
}

// GenerateRandomChess960 returns a randomly rolled Chess960 starting position.
func (a *App) GenerateRandomChess960() (*chess960.Position, error) {
	pos := chess960.RandomPosition()
	return pos, nil
}

// ==========================================
// EPD (Extended Position Description)
// ==========================================

// ParseEPD parses an EPD string into structured fields with opcodes.
func (a *App) ParseEPD(epdString string) (*epd.EPDRecord, error) {
	return epd.Parse(epdString)
}

// ExportEPD formats a position and opcodes into standard EPD format.
func (a *App) ExportEPD(fen string, opcodes map[string]string) (string, error) {
	return epd.Format(fen, opcodes), nil
}

// ==========================================
// Positional & Material Imbalance
// ==========================================

// GetPositionalImbalance analyzes the material, pawn structure, king safety, and territory control.
func (a *App) GetPositionalImbalance(fen string) (*imbalance.Report, error) {
	b, err := chess.ParseFEN(fen)
	if err != nil {
		return nil, err
	}
	return imbalance.AnalyzeBoard(b), nil
}

// ==========================================
// Natural Language Move Explainer
// ==========================================

// ExplainMove generates plain-English coaching explanations of a move and its motifs.
func (a *App) ExplainMove(beforeFEN, moveUCI string, scoreCp int, bestMove string) (*explainer.Explanation, error) {
	return explainer.ExplainMove(beforeFEN, moveUCI, scoreCp, bestMove)
}

// ==========================================
// Personal Blunder Spaced-Repetition Vault
// ==========================================

// GetDueBlunderPuzzles returns blunder flashcards due for spaced-repetition review.
func (a *App) GetDueBlunderPuzzles() ([]*blundervault.Puzzle, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.blunderVault == nil {
		return []*blundervault.Puzzle{}, nil
	}
	return a.blunderVault.GetDuePuzzles(), nil
}

// GetAllBlunderPuzzles returns all blunder flashcards in the vault.
func (a *App) GetAllBlunderPuzzles() ([]*blundervault.Puzzle, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.blunderVault == nil {
		return []*blundervault.Puzzle{}, nil
	}
	return a.blunderVault.GetAllPuzzles(), nil
}

// RecordBlunderReview records a review grade (0-5) and recalculates SM-2 interval.
func (a *App) RecordBlunderReview(id string, quality int) (*blundervault.Puzzle, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.blunderVault == nil {
		return nil, fmt.Errorf("blunder vault not initialized")
	}
	return a.blunderVault.RecordReview(id, quality)
}

// AddBlunderToVault adds a new blunder puzzle into the vault.
func (a *App) AddBlunderToVault(p *blundervault.Puzzle) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.blunderVault == nil {
		return fmt.Errorf("blunder vault not initialized")
	}
	return a.blunderVault.AddPuzzle(p)
}

// ==========================================
// Opponent Scouting & Prep Matrix
// ==========================================

// GenerateOpponentScoutingReport builds an in-depth dossier on a player from the current database.
func (a *App) GenerateOpponentScoutingReport(playerName string) (*scout.Report, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.db == nil || !a.db.IsOpen {
		return nil, fmt.Errorf("no database is currently open")
	}

	games, _, err := a.db.GetGameSummaries(0, 100000, scid.SearchCriteria{})
	if err != nil {
		return nil, err
	}
	return scout.GenerateReport(playerName, games), nil
}

// ==========================================
// Opponent Opening Exploiter & Custom Book Generator
// ==========================================

// EvaluateOpponentWeaknesses performs opening tree evaluation for any player in the active database.
func (a *App) EvaluateOpponentWeaknesses(playerName string, targetColor string, maxDepth int, minGames int) (*persona.AntiPersonaReport, error) {
	a.mu.RLock()
	db := a.db
	a.mu.RUnlock()

	if db == nil || !db.IsOpen {
		return nil, fmt.Errorf("no database is currently open")
	}

	cleanName := strings.TrimSpace(playerName)
	if cleanName == "" {
		return nil, fmt.Errorf("player name cannot be empty")
	}

	cfg := persona.AntiPersonaConfig{
		DisplayName:          cleanName,
		Identifiers:          []string{cleanName},
		TargetColor:          targetColor,
		MaxDepth:             maxDepth,
		MinGames:             minGames,
		MinVulnerabilityRate: 45.0,
	}

	return persona.EvaluatePersonaDatabase(db, cfg)
}

// EvaluatePersonaWeaknesses performs deep opening tree evaluation for a persona across active and dedicated databases.
func (a *App) EvaluatePersonaWeaknesses(personaID string, targetColor string, maxDepth int, minGames int) (*persona.AntiPersonaReport, error) {
	if a.personaMgr == nil {
		return nil, fmt.Errorf("persona manager not initialized")
	}

	p, err := a.personaMgr.GetPersona(personaID)
	if err != nil {
		return nil, err
	}

	cfg := persona.AntiPersonaConfig{
		PersonaID:            p.ID,
		DisplayName:          p.DisplayName,
		Identifiers:          p.Identifiers,
		TargetColor:          targetColor,
		MaxDepth:             maxDepth,
		MinGames:             minGames,
		MinVulnerabilityRate: 45.0,
	}

	a.mu.RLock()
	activeDB := a.db
	a.mu.RUnlock()

	var allGames []*chess.Game
	seenSigs := make(map[string]bool)

	addGamesFromDB := func(db *scid.Database) {
		if db == nil || !db.IsOpen {
			return
		}
		nameLookup := make(map[string]bool)
		for _, id := range p.Identifiers {
			t := strings.ToLower(strings.TrimSpace(id))
			if t != "" {
				nameLookup[t] = true
			}
		}
		if p.DisplayName != "" {
			nameLookup[strings.ToLower(strings.TrimSpace(p.DisplayName))] = true
		}

		for i := 0; i < len(db.Entries); i++ {
			entry := db.Entries[i]
			if (entry.Flags & scid.FlagDelete) != 0 {
				continue
			}

			w := strings.ToLower(db.Names.GetName(scid.NamePlayer, entry.WhiteID))
			b := strings.ToLower(db.Names.GetName(scid.NamePlayer, entry.BlackID))

			isW := matchPlayerLookup(w, nameLookup)
			isB := matchPlayerLookup(b, nameLookup)
			if !isW && !isB {
				continue
			}

			sig := fmt.Sprintf("%s|%s|%s|%s|%d", entry.Date, w, b, entry.Result, entry.NumHalfMoves)
			if seenSigs[sig] {
				continue
			}
			seenSigs[sig] = true

			g, err := db.GetGame(i + 1)
			if err == nil && g != nil {
				allGames = append(allGames, g)
			}
		}
	}

	if activeDB != nil && activeDB.IsOpen {
		addGamesFromDB(activeDB)
	}

	if p.DatabasePath != "" {
		dedicatedDB := &scid.Database{}
		if err := dedicatedDB.Open(p.DatabasePath); err == nil {
			defer dedicatedDB.Close()
			addGamesFromDB(dedicatedDB)
		}
	}

	return persona.EvaluateGames(allGames, cfg)
}

func matchPlayerLookup(name string, lookup map[string]bool) bool {
	if lookup[name] {
		return true
	}
	for id := range lookup {
		if strings.Contains(name, id) || strings.Contains(id, name) {
			return true
		}
	}
	return false
}

// ExportAntiPersonaPolyglotBook writes the generated Polyglot opening book to disk.
func (a *App) ExportAntiPersonaPolyglotBook(report *persona.AntiPersonaReport, filePath string) (string, error) {
	if report == nil {
		return "", fmt.Errorf("nil report")
	}

	target := strings.TrimSpace(filePath)
	if target == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			homeDir = os.TempDir()
		}
		booksDir := filepath.Join(homeDir, ".local", "share", "newscid", "books")
		_ = os.MkdirAll(booksDir, 0755)

		slug := strings.ReplaceAll(report.DisplayName, " ", "_")
		slug = strings.ReplaceAll(slug, ",", "")
		target = filepath.Join(booksDir, fmt.Sprintf("Anti_%s.bin", slug))
	}

	if err := persona.SaveAntiPersonaBook(report, target); err != nil {
		return "", err
	}

	return target, nil
}

// ExportAntiPersonaPGN writes the generated PGN repertoire to disk.
func (a *App) ExportAntiPersonaPGN(report *persona.AntiPersonaReport, filePath string) (string, error) {
	if report == nil {
		return "", fmt.Errorf("nil report")
	}

	target := strings.TrimSpace(filePath)
	if target == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			homeDir = os.TempDir()
		}
		repDir := filepath.Join(homeDir, ".local", "share", "newscid", "repertoires")
		_ = os.MkdirAll(repDir, 0755)

		slug := strings.ReplaceAll(report.DisplayName, " ", "_")
		slug = strings.ReplaceAll(slug, ",", "")
		target = filepath.Join(repDir, fmt.Sprintf("Anti_%s.pgn", slug))
	}

	dir := filepath.Dir(target)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	if err := os.WriteFile(target, []byte(report.PGNRepertoire), 0644); err != nil {
		return "", err
	}

	return target, nil
}

// SaveBookFileDialog prompts user to choose a destination for opening book (.bin).
func (a *App) SaveBookFileDialog(defaultFilename string) (string, error) {
	return runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Save Polyglot Opening Book",
		DefaultFilename: defaultFilename,
		Filters: []runtime.FileFilter{
			{DisplayName: "Polyglot Opening Book (*.bin)", Pattern: "*.bin"},
		},
	})
}

// SavePGNFileDialog prompts user to choose a destination for PGN repertoire.
func (a *App) SavePGNFileDialog(defaultFilename string) (string, error) {
	return runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Save Repertoire PGN",
		DefaultFilename: defaultFilename,
		Filters: []runtime.FileFilter{
			{DisplayName: "PGN Chess Repertoire (*.pgn)", Pattern: "*.pgn"},
		},
	})
}

// ==========================================
// Piece Safety Scanner & Drill Module
// ==========================================

// AnalyzeBoardSafety calculates square safety, attackers, defenders, and hanging pieces for the given FEN.
func (a *App) AnalyzeBoardSafety(fen string) (*training.BoardSafetyReport, error) {
	if strings.TrimSpace(fen) == "" {
		a.mu.RLock()
		if a.activeGame != nil {
			b := chess.NewBoard()
			if a.activeGame.StartFEN != "" {
				if parsed, err := chess.ParseFEN(a.activeGame.StartFEN); err == nil {
					b = parsed
				}
			}
			for curr := a.activeGame.RootMove; curr != nil; curr = curr.Next {
				b.MakeMove(curr.Move)
			}
			fen = b.FEN()
		}
		a.mu.RUnlock()
	}
	if fen == "" {
		fen = chess.StartingFEN
	}
	return training.AnalyzeBoardSafety(fen)
}

// GetSafetyDrillBatch retrieves a set of curated safety challenges.
func (a *App) GetSafetyDrillBatch(count int, difficulty string) ([]training.SafetyDrill, error) {
	drills := training.GetSafetyDrillBatch(count, difficulty)
	return drills, nil
}

// GenerateSafetyDrillsFromDatabase mines blunder positions from the active database.
func (a *App) GenerateSafetyDrillsFromDatabase(count int) ([]training.SafetyDrill, error) {
	a.mu.RLock()
	db := a.db
	a.mu.RUnlock()

	if db == nil || !db.IsOpen {
		return training.GetSafetyDrillBatch(count, "all"), nil
	}
	return training.ExtractSafetyDrillsFromDatabase(db, count), nil
}

// SubmitSafetyDrillAnswer evaluates the user's selected squares or move against a safety drill.
func (a *App) SubmitSafetyDrillAnswer(drill training.SafetyDrill, selectedSquares []string, moveUCI string) (*training.SafetyDrillFeedback, error) {
	feedback := training.EvaluateSafetySubmission(&drill, selectedSquares, moveUCI)
	return feedback, nil
}

// GetIntuitionDrillBatch returns a batch of curated rapid evaluation challenges.
func (a *App) GetIntuitionDrillBatch(count int) ([]training.IntuitionChallenge, error) {
	return training.GetIntuitionDrillBatch(count), nil
}

// GenerateIntuitionDrillsFromDatabase extracts positions from the active database for speed assessment.
func (a *App) GenerateIntuitionDrillsFromDatabase(count int) ([]training.IntuitionChallenge, error) {
	a.mu.RLock()
	db := a.db
	a.mu.RUnlock()

	if db == nil || !db.IsOpen {
		return training.GetIntuitionDrillBatch(count), nil
	}
	return training.GenerateIntuitionDrillsFromDatabase(db, count)
}

// SubmitIntuitionAnswer evaluates the user's rapid judgment and returns score and speed bonus.
func (a *App) SubmitIntuitionAnswer(challengeID string, userChoice string, elapsedMs int) (*training.IntuitionResult, error) {
	return training.SubmitIntuitionAnswer(challengeID, userChoice, elapsedMs)
}

// AnalyzeGameTimeManagement analyzes clock progression, tanks, panics, and scrambles for a game.
func (a *App) AnalyzeGameTimeManagement(gameID int) (*stats.GameTimeManagementReport, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var targetGame *chess.Game
	if gameID <= 0 || a.db == nil || !a.db.IsOpen {
		targetGame = a.activeGame
	} else {
		g, err := a.db.GetGame(gameID)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch game %d: %w", gameID, err)
		}
		targetGame = g
	}

	if targetGame == nil {
		return nil, fmt.Errorf("no game loaded or found to analyze")
	}

	return stats.AnalyzeGameTimeManagement(targetGame)
}

// AnalyzePlayerTimeManagement generates an aggregate time management and tanking profile for a player.
func (a *App) AnalyzePlayerTimeManagement(playerName string, maxGames int) (*stats.PlayerTimeManagementDossier, error) {
	a.mu.RLock()
	db := a.db
	a.mu.RUnlock()

	if db == nil || !db.IsOpen {
		return nil, fmt.Errorf("database is not open")
	}

	return stats.AnalyzePlayerTimeManagement(db, playerName, maxGames)
}

// GetProphylaxisDrillBatch returns curated quiet move / prophylaxis drill challenges.
func (a *App) GetProphylaxisDrillBatch(count int, category string) ([]training.ProphylaxisDrill, error) {
	return training.GetProphylaxisDrillBatch(count, category), nil
}

// GenerateProphylaxisDrillsFromDatabase mines quiet prophylactic moves from the active database.
func (a *App) GenerateProphylaxisDrillsFromDatabase(count int) ([]training.ProphylaxisDrill, error) {
	a.mu.RLock()
	db := a.db
	a.mu.RUnlock()

	if db == nil || !db.IsOpen {
		return training.GetProphylaxisDrillBatch(count, "all"), nil
	}
	return training.GenerateProphylaxisDrillsFromDatabase(db, count)
}

// SubmitProphylaxisAnswer checks the user's quiet prophylactic move.
func (a *App) SubmitProphylaxisAnswer(drillID string, moveUCI string) (*training.ProphylaxisFeedback, error) {
	return training.SubmitProphylaxisAnswer(drillID, moveUCI)
}

func (a *App) collectLinkedChessAndLichessIDs() ([]string, []string) {
	var chessComIDs []string
	var lichessIDs []string

	if a.profileMgr != nil {
		for _, p := range a.profileMgr.GetProfiles() {
			if strings.TrimSpace(p.ChessComUsername) != "" {
				chessComIDs = append(chessComIDs, p.ChessComUsername)
			}
			if strings.TrimSpace(p.LichessUsername) != "" {
				lichessIDs = append(lichessIDs, p.LichessUsername)
			}
			if strings.TrimSpace(p.Name) != "" {
				chessComIDs = append(chessComIDs, p.Name)
				lichessIDs = append(lichessIDs, p.Name)
			}
		}
	}

	if a.personaMgr != nil {
		for _, p := range a.personaMgr.GetPersonas() {
			if strings.TrimSpace(p.ChessComUsername) != "" {
				chessComIDs = append(chessComIDs, p.ChessComUsername)
			}
			if strings.TrimSpace(p.LichessUsername) != "" {
				lichessIDs = append(lichessIDs, p.LichessUsername)
			}
			for _, id := range p.Identifiers {
				chessComIDs = append(chessComIDs, id)
				lichessIDs = append(lichessIDs, id)
			}
		}
	}

	return chessComIDs, lichessIDs
}

// GetSupportStatus returns the current voluntary support status, Stripe URL, and whether the startup prompt should display.
// It automatically exempts the author's computer using the linked Chess.com and Lichess.org IDs.
func (a *App) GetSupportStatus() support.SupportStatus {
	if a.supportMgr == nil {
		a.supportMgr, _ = support.NewManager("")
	}
	chessComIDs, lichessIDs := a.collectLinkedChessAndLichessIDs()
	return a.supportMgr.GetStatus(chessComIDs, lichessIDs)
}

// RecordUserSupport records that the user has supported NewScid (or verified an author/supporter ID),
// permanently disabling the startup nag dialog.
func (a *App) RecordUserSupport(tier string, reference string) (support.SupportStatus, error) {
	if a.supportMgr == nil {
		a.supportMgr, _ = support.NewManager("")
	}
	return a.supportMgr.RecordSupport(tier, reference)
}

// DismissSupportNag dismisses the non-obligatory support prompt for the current session.
func (a *App) DismissSupportNag() (support.SupportStatus, error) {
	if a.supportMgr == nil {
		a.supportMgr, _ = support.NewManager("")
	}
	return a.supportMgr.DismissNag()
}

// OpenSupportLink opens the Stripe support checkout URL in the user's default system browser
// with client_reference_id=newscid_<amount>usd so payments from NewScid are distinguishable from Bughouse Online in Stripe.
func (a *App) OpenSupportLink(amountUSD int) string {
	st := a.GetSupportStatus()
	if amountUSD < support.DefaultMinSupportUSD {
		amountUSD = support.DefaultMinSupportUSD
	}
	baseURL := support.StripeURLForAmount(amountUSD, st.StripeURL)
	targetURL := support.BuildCheckoutURL(baseURL, fmt.Sprintf("newscid_%dusd", amountUSD))
	_, _ = a.RecordUserSupport(fmt.Sprintf("$%d Support", amountUSD), "stripe_checkout")
	if a.ctx != nil {
		runtime.BrowserOpenURL(a.ctx, targetURL)
	}
	return targetURL
}

// SetSupportStripeURL updates the Stripe payment link URL used by the Support form.
func (a *App) SetSupportStripeURL(url string) error {
	if a.supportMgr == nil {
		a.supportMgr, _ = support.NewManager("")
	}
	return a.supportMgr.SetStripeURL(url)
}

// GetMegaDatabaseCatalog returns the curated catalog of big online chess databases
// (Lumbra's GigaBase, Lichess Open, Lichess Broadcasts, and Lichess Elite).
func (a *App) GetMegaDatabaseCatalog() []online.MegaDatabaseSource {
	client := online.NewMegaDatabaseClient()
	return client.GetCatalog()
}

// FetchLichessDatabaseFeed fetches the latest monthly .pgn.zst archive URLs from database.lichess.org.
func (a *App) FetchLichessDatabaseFeed(feedType string, limit int) ([]string, error) {
	client := online.NewMegaDatabaseClient()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return client.FetchLichessFeed(ctx, feedType, limit)
}

// ImportMegaDatabase downloads or streams a big online database (LumbrasGigaBase, Lichess Open, Lichess Elite, etc.)
// and imports the games into the active Scid database (or creates a new .si4 database if requested).
func (a *App) ImportMegaDatabase(opts online.MegaImportOptions) (*online.MegaImportResult, error) {
	client := online.NewMegaDatabaseClient()
	home, _ := os.UserHomeDir()
	userDBDir := filepath.Join(home, ".local", "share", "newscid", "databases")
	_ = os.MkdirAll(userDBDir, 0o755)

	// Determine whether we need to create/open a new database first
	a.mu.Lock()
	targetDB := a.db
	isOpen := targetDB != nil && targetDB.IsOpen
	a.mu.Unlock()

	src, hasSrc := client.FindSource(opts.SourceID)
	isNativeSI4 := hasSrc && src.Format == "7z-si4"

	if !isNativeSI4 {
		newDBName := strings.TrimSpace(opts.CreateNewDatabase)
		if newDBName == "" && !isOpen {
			if hasSrc {
				newDBName = strings.ReplaceAll(strings.ToLower(src.ID), "-", "_")
			} else {
				newDBName = "mega_imported"
			}
		}
		if newDBName != "" {
			cleanBase := filepath.Base(newDBName)
			cleanBase = strings.TrimSuffix(cleanBase, ".si4")
			newDBPath := filepath.Join(userDBDir, cleanBase)
			if _, err := os.Stat(newDBPath + ".si4"); err == nil {
				_, _ = a.OpenDatabase(newDBPath)
			} else {
				_, _ = a.CreateDatabase(newDBPath, cleanBase)
			}
			a.mu.Lock()
			targetDB = a.db
			a.mu.Unlock()
		}
	}

	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}

	res, err := client.DownloadAndImport(ctx, opts, targetDB, userDBDir, func(prog online.MegaImportProgress) {
		a.emitEvent("megadb:progress", prog)
	})
	if err != nil {
		return res, err
	}

	// If a native .si4 archive was extracted, open it automatically!
	if res != nil && res.DatabasePath != "" {
		_, _ = a.OpenDatabase(res.DatabasePath)
	}
	return res, nil
}

// AnalyzePlayerFairPlay runs the Longitudinal Fair-Play & Time-Control Consistency Inspector
// across all games of playerName in the currently open database.
func (a *App) AnalyzePlayerFairPlay(playerName string, maxGames int) (*fairplay.PlayerFairPlayDossier, error) {
	a.mu.Lock()
	db := a.db
	a.mu.Unlock()
	return fairplay.AnalyzePlayerFairPlay(db, playerName, maxGames)
}

// DownloadAndAnalyzePlayerFairPlay fetches a player's recent games across all time controls from
// Lichess.org and/or Chess.com, imports them into the active (or dedicated) database, and returns
// the complete Longitudinal Multi-Time-Control & Day-by-Day Fair-Play Dossier.
func (a *App) DownloadAndAnalyzePlayerFairPlay(platform string, username string, maxGames int) (*fairplay.PlayerFairPlayDossier, error) {
	user := strings.TrimSpace(username)
	if user == "" {
		return nil, fmt.Errorf("username is required")
	}
	if maxGames <= 0 {
		maxGames = 200
	}

	// Ensure a database is open to store and inspect the downloaded games
	a.mu.Lock()
	db := a.db
	isOpen := db != nil && db.IsOpen
	a.mu.Unlock()

	if !isOpen {
		home, _ := os.UserHomeDir()
		userDBDir := filepath.Join(home, ".local", "share", "newscid", "databases")
		_ = os.MkdirAll(userDBDir, 0o755)
		safeName := "fairplay_" + strings.ToLower(regexp.MustCompile(`[^a-zA-Z0-9_-]`).ReplaceAllString(user, "_"))
		dbPath := filepath.Join(userDBDir, safeName)
		if _, err := os.Stat(dbPath + ".si4"); err == nil {
			_, _ = a.OpenDatabase(dbPath)
		} else {
			_, _ = a.CreateDatabase(dbPath, "Fair-Play Dossier: "+user)
		}
	}

	plat := strings.ToLower(strings.TrimSpace(platform))
	opts := online.ImportPlayerOptions{
		PlayerName:      user,
		TargetMode:      online.TargetActiveBase,
		MaxGames:        maxGames,
		AutoDeduplicate: true,
	}
	switch plat {
	case "chesscom", "chess.com":
		opts.ChessComUsername = user
	case "both":
		opts.LichessUsername = user
		opts.ChessComUsername = user
	default:
		opts.LichessUsername = user
	}

	_, err := a.ImportPlayerGames(opts)
	if err != nil {
		return nil, err
	}

	a.mu.Lock()
	db = a.db
	a.mu.Unlock()
	return fairplay.AnalyzePlayerFairPlay(db, user, maxGames)
}


