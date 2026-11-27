package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"newscid/pkg/academy"
	"newscid/pkg/blundervault"
	"newscid/pkg/chess"
	"newscid/pkg/database/scid"
	"newscid/pkg/database/sciddriver"
	"newscid/pkg/eco"
	"newscid/pkg/engine"
	"newscid/pkg/fileformat"
	"newscid/pkg/online"
	"newscid/pkg/persona"
	"newscid/pkg/playerbot"
	"newscid/pkg/polyglot"
	"newscid/pkg/puzzleminer"
	"newscid/pkg/repertoire"
	"newscid/pkg/sets"
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
	WritePolicy string `json:"writePolicy"`
	IsLocked    bool   `json:"isLocked"`
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

// RepertoireScanRequest configures an online and/or database scan for opening book deviations.
type RepertoireScanRequest struct {
	RepertoireID     string `json:"repertoireId"`
	Source           string `json:"source"` // "online", "database", "both"
	LichessUsername  string `json:"lichessUsername"`
	ChessComUsername string `json:"chessComUsername"`
	PlayerName       string `json:"playerName"`
	MaxGames         int    `json:"maxGames"`
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
	refDB            *scid.Database
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
	opMu             sync.Mutex
	activeOpCancel   context.CancelFunc
	activeOpName     string
	engineMultiPV    int
	engineThreads    int
	engineHashMB     int
	notationStyle    chess.NotationStyle
	blunderVault     *blundervault.Vault
	supportMgr       *support.Manager
	cyclicalMgr      *training.CyclicalManager
	coachMgr         *training.AdaptiveCoachManager
	academyMgr       *academy.Manager
}

// NewApp creates a new App application struct
func NewApp() *App {
	classifier := eco.NewClassifier()
	// Attempt to load system scid.eco if available
	if _, err := os.Stat(fileformat.SystemScidECOPath); err == nil {
		_ = classifier.LoadFromFile(fileformat.SystemScidECOPath)
	}

	profileMgr, _ := online.NewProfileManager("")
	personaMgr, _ := persona.NewPersonaManager("")

	configDir, _ := os.UserConfigDir()
	vaultPath := filepath.Join(configDir, fileformat.AppDirName, fileformat.FileBlunderVaultJSON)
	bVault, _ := blundervault.NewVault(vaultPath)
	supportPath := filepath.Join(configDir, fileformat.AppDirName, fileformat.FileSupportJSON)
	supMgr, _ := support.NewManager(supportPath)
	cyclicalPath := filepath.Join(configDir, fileformat.AppDirName, fileformat.FileCyclicalDrillJSON)
	cycMgr, _ := training.NewCyclicalManager(cyclicalPath)
	coachPath := filepath.Join(configDir, fileformat.AppDirName, fileformat.FileAdaptiveCoachJSON)
	coachMgr, _ := training.NewAdaptiveCoachManager(coachPath)
	academyPath := filepath.Join(configDir, fileformat.AppDirName, "career.json")
	acadMgr := academy.NewManager(academyPath)

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
		cyclicalMgr:     cycMgr,
		coachMgr:        coachMgr,
		academyMgr:      acadMgr,
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
	if a.refDB != nil && a.refDB.IsOpen {
		_ = a.refDB.Close()
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
// Database Operations
func (a *App) checkActiveDBEditableLocked() error {
	if a.db == nil || !a.db.IsOpen {
		return fmt.Errorf("no database open")
	}
	desc := ""
	if a.db.Header != nil {
		desc = a.db.Header.Description
	}
	pol := a.resolveDatabaseWritePolicy(a.db.BasePath, a.db.NumGames(), desc)
	if !pol.AllowsDirectEdit() {
		return fmt.Errorf("database %q is protected (%s policy) to prevent accidental contamination; unlock it or switch to a personal database first", filepath.Base(a.db.BasePath), pol.String())
	}
	return nil
}

func (a *App) checkActiveDBSyncAllowedLocked() error {
	if a.db == nil || !a.db.IsOpen {
		return fmt.Errorf("no database open")
	}
	desc := ""
	if a.db.Header != nil {
		desc = a.db.Header.Description
	}
	pol := a.resolveDatabaseWritePolicy(a.db.BasePath, a.db.NumGames(), desc)
	if !pol.AllowsOnlineSync() {
		return fmt.Errorf("database %q is strictly locked (%s); unlock it or set it to Sync-Only to allow online updates", filepath.Base(a.db.BasePath), pol.String())
	}
	return nil
}

func (a *App) OpenDatabase(path string) (*DatabaseInfo, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// If path is a PGN file, convert/cache into Scid DB
	if fileformat.IsPGNPath(path) {
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		games, err := chess.ParsePGNDatabase(strings.NewReader(string(content)))
		if err != nil {
			return nil, fmt.Errorf("failed to parse PGN: %w", err)
		}
		tmpBase := filepath.Join(os.TempDir(), "newscid_pgn_"+fileformat.TrimDatabaseExtensions(filepath.Base(path)))
		for _, ext := range fileformat.ScidV4Extensions() {
			_ = os.Remove(tmpBase + ext)
		}
		if err := a.db.Create(tmpBase, "Imported from "+filepath.Base(path)); err != nil {
			return nil, err
		}
		for _, g := range games {
			_, _ = a.db.SaveGame(g)
		}
		numG := a.db.NumGames()
		desc := "Imported PGN: " + filepath.Base(path)
		pol := a.resolveDatabaseWritePolicy(a.db.BasePath, numG, desc)
		return &DatabaseInfo{
			BasePath:    a.db.BasePath,
			Name:        filepath.Base(path),
			NumGames:    numG,
			Description: desc,
			WritePolicy: pol.String(),
			IsLocked:    !pol.AllowsDirectEdit(),
		}, nil
	}

	if err := a.db.Open(path); err != nil {
		return nil, err
	}

	desc := ""
	if a.db.Header != nil {
		desc = a.db.Header.Description
	}
	numG := a.db.NumGames()
	pol := a.resolveDatabaseWritePolicy(a.db.BasePath, numG, desc)

	return &DatabaseInfo{
		BasePath:    a.db.BasePath,
		Name:        filepath.Base(a.db.BasePath),
		NumGames:    numG,
		Description: desc,
		WritePolicy: pol.String(),
		IsLocked:    !pol.AllowsDirectEdit(),
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
		WritePolicy: scid.WritePolicyEditable.String(),
		IsLocked:    false,
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
	numG := a.db.NumGames()
	pol := a.resolveDatabaseWritePolicy(a.db.BasePath, numG, desc)

	return &DatabaseInfo{
		BasePath:    a.db.BasePath,
		Name:        filepath.Base(a.db.BasePath),
		NumGames:    numG,
		Description: desc,
		WritePolicy: pol.String(),
		IsLocked:    !pol.AllowsDirectEdit(),
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

	if err := a.checkActiveDBEditableLocked(); err != nil {
		return 0, err
	}

	if game != nil && game.RootMove == nil && a.activeGame != nil && a.activeGame.RootMove != nil {
		game.RootMove = a.activeGame.RootMove
	}
	if game != nil {
		a.activeGame = game
	}

	return a.db.SaveGame(game)
}

func (a *App) DeleteGame(gameID int, deleted bool) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if err := a.checkActiveDBEditableLocked(); err != nil {
		return err
	}

	return a.db.DeleteGame(gameID, deleted)
}

// UpdateGameHeaders updates game metadata tags in the database and active game.
func (a *App) UpdateGameHeaders(gameID int, tags map[string]string) (*scid.GameSummary, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if err := a.checkActiveDBEditableLocked(); err != nil {
		return nil, err
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

	if err := a.checkActiveDBEditableLocked(); err != nil {
		return nil, err
	}

	return a.db.Compact()
}

// SortDatabase reorders the database index based on the chosen field and direction.
func (a *App) SortDatabase(criteria scid.SortCriteria) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if err := a.checkActiveDBEditableLocked(); err != nil {
		return err
	}

	return a.db.Sort(criteria)
}

// FindDuplicateGames finds all groups of duplicate games in the current database.
func (a *App) FindDuplicateGames(criteria scid.DuplicateCriteria) ([]scid.DuplicateGroup, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.db == nil || !a.db.IsOpen {
		return nil, fmt.Errorf("no database open")
	}

	return a.db.FindDuplicates(criteria)
}

// DeleteDuplicateGames marks duplicate copies of games as deleted, keeping the primary instance.
func (a *App) DeleteDuplicateGames(criteria scid.DuplicateCriteria) (int, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if err := a.checkActiveDBEditableLocked(); err != nil {
		return 0, err
	}

	return a.db.DeleteDuplicates(criteria)
}

// VerifyDatabase performs a thorough integrity check on the active database files.
func (a *App) VerifyDatabase() (*scid.VerificationReport, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.db == nil || !a.db.IsOpen {
		return nil, fmt.Errorf("no database open")
	}

	return a.db.Verify()
}

// GetDatabaseStatistics aggregates demographic, rating, result, and storage metrics.
func (a *App) GetDatabaseStatistics() (*scid.DatabaseStatistics, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.db == nil || !a.db.IsOpen {
		return nil, fmt.Errorf("no database open")
	}

	return a.db.GetStatistics()
}

// ImportPGNBatch imports multiple PGN games from a raw string.
func (a *App) ImportPGNBatch(pgnText string) (*scid.ImportStats, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if err := a.checkActiveDBEditableLocked(); err != nil {
		return nil, err
	}

	return a.db.ImportPGNText(pgnText)
}

// ImportPGNFileBatch imports multiple PGN games from a file on disk.
func (a *App) ImportPGNFileBatch(filePath string) (*scid.ImportStats, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if err := a.checkActiveDBEditableLocked(); err != nil {
		return nil, err
	}

	return a.db.ImportPGNFile(filePath)
}

var (
	openFileDialogFunc = runtime.OpenFileDialog
	saveFileDialogFunc = runtime.SaveFileDialog
)

func (a *App) openFileDialog(title string, filters ...runtime.FileFilter) (string, error) {
	return openFileDialogFunc(a.ctx, runtime.OpenDialogOptions{
		Title:   title,
		Filters: filters,
	})
}

func (a *App) saveFileDialog(title, defaultName string, filters ...runtime.FileFilter) (string, error) {
	return saveFileDialogFunc(a.ctx, runtime.SaveDialogOptions{
		Title:           title,
		DefaultFilename: defaultName,
		Filters:         filters,
	})
}

func (a *App) OpenPGNFileDialog() (string, error) {
	return a.openFileDialog("Select PGN File to Import",
		runtime.FileFilter{DisplayName: "PGN Files (" + fileformat.GlobPGN + ")", Pattern: fileformat.GlobPGN},
		runtime.FileFilter{DisplayName: "All Files (*.*)", Pattern: "*.*"},
	)
}

func (a *App) OpenFileDialog() (string, error) {
	return a.openFileDialog("Open Chess Database",
		runtime.FileFilter{DisplayName: "SCID Databases (" + fileformat.GlobScidIndex + ")", Pattern: fileformat.GlobScidIndex},
		runtime.FileFilter{DisplayName: "PGN Databases (" + fileformat.GlobPGN + ")", Pattern: fileformat.GlobPGN},
		runtime.FileFilter{DisplayName: "All Files (*.*)", Pattern: "*.*"},
	)
}

func (a *App) SaveFileDialog(defaultName string) (string, error) {
	return a.saveFileDialog("Create Chess Database", defaultName,
		runtime.FileFilter{DisplayName: "SCID Database (" + fileformat.GlobScidIndex + ")", Pattern: fileformat.GlobScidIndex},
	)
}
// Chess Board & Move Operations
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
	prefs, _ := a.GetAppPreferences()
	if prefs != nil && prefs.ReferenceDatabasePath != "" && prefs.UseReferenceDBForTree {
		_ = a.openReferenceDatabaseInternal(prefs.ReferenceDatabasePath)
	}

	a.mu.RLock()
	db := a.db
	refDB := a.refDB
	a.mu.RUnlock()

	var res *tree.TreeResult
	var err error
	if prefs != nil && prefs.UseReferenceDBForTree && refDB != nil && refDB.IsOpen {
		res, _ = tree.BuildTreeFromDatabase(refDB, fen, 1000)
		if res != nil && len(res.Moves) > 0 {
			return res, nil
		}
	}
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
			if strings.Contains(b, fileformat.BookPerformanceBin) {
				preferredBook = b
				break
			}
			if strings.Contains(b, fileformat.BookElo2400Bin) {
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

	if a.db == nil || !a.db.IsOpen {
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
// UCI Engine Operations & Scid Configuration
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
// Piece Sets & Board Themes
func (a *App) GetPieceSets() []sets.PieceSetInfo {
	return sets.GetAvailablePieceSets()
}

func (a *App) GetBoardThemes() []sets.BoardThemeInfo {
	return sets.GetAvailableBoardThemes()
}

func (a *App) GetDiscoveredDatabases() []scid.DiscoveredDatabase {
	personaPaths := make(map[string]*persona.Persona)
	if a.personaMgr != nil {
		for _, p := range a.personaMgr.GetPersonas() {
			if basePath, err := a.personaMgr.EnsurePersonaDatabase(p); err == nil && basePath != "" {
				personaPaths[normalizeScidBasePath(basePath)] = p
			}
		}
	}

	dbs := scid.DiscoverScidDatabases()
	seen := make(map[string]bool, len(dbs))
	for i := range dbs {
		clean := normalizeScidBasePath(dbs[i].BasePath)
		seen[clean] = true
		if p, ok := personaPaths[clean]; ok {
			dbs[i].Category = "Persona Databases"
			if dbs[i].Description == "" {
				dbs[i].Description = fmt.Sprintf("%s Unified Games Database", p.DisplayName)
			}
		}
		pol := a.resolveDatabaseWritePolicy(dbs[i].BasePath, dbs[i].NumGames, dbs[i].Description)
		dbs[i].WritePolicy = pol.String()
		dbs[i].IsLocked = !pol.AllowsDirectEdit()
	}

	for clean, p := range personaPaths {
		if !seen[clean] {
			meta := scid.InspectDatabase(clean, "Persona Databases")
			if meta.Description == "" {
				meta.Description = fmt.Sprintf("%s Unified Games Database", p.DisplayName)
			}
			pol := a.resolveDatabaseWritePolicy(meta.BasePath, meta.NumGames, meta.Description)
			meta.WritePolicy = pol.String()
			meta.IsLocked = !pol.AllowsDirectEdit()
			dbs = append(dbs, meta)
		}
	}
	return dbs
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
		if _, err := os.Stat(c + fileformat.ExtScidIndex); err == nil {
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
	basePath = fileformat.TrimScidExtensions(basePath)

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

	if _, err := os.Stat(base + fileformat.ExtScidIndex); err != nil {
		base = "/usr/share/scid/bases/tactics"
	}

	puzzle, err := a.LoadTacticsPuzzle(base, 0)
	if err != nil {
		return nil, err
	}
	puzzle.Theme = fmt.Sprintf("Level %d (%s)", level, filepath.Base(base))
	return puzzle, nil
}
// Opening Repertoire Operations
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

// AddRepertoireLineSAN parses a PGN/SAN move sequence and adds all moves into the specified repertoire.
func (a *App) AddRepertoireLineSAN(repID string, moveText string, comment string) (*repertoire.Repertoire, error) {
	return a.repMgr.AddLineSAN(repID, moveText, comment)
}

// DeleteRepertoireMove removes a move from a position in the specified repertoire.
func (a *App) DeleteRepertoireMove(repID string, fen string, uci string) (*repertoire.Repertoire, error) {
	return a.repMgr.DeleteMove(repID, fen, uci)
}

// UpdateRepertoireMoveComment updates the strategic note for a move in the specified repertoire.
func (a *App) UpdateRepertoireMoveComment(repID string, fen string, uci string, comment string) (*repertoire.Repertoire, error) {
	return a.repMgr.UpdateMoveComment(repID, fen, uci, comment)
}

// GetRepertoireMastery returns the 0-100% systematic training progress and variation line checklist for a repertoire.
func (a *App) GetRepertoireMastery(repID string) (*repertoire.RepertoireMasteryReport, error) {
	return a.repMgr.GetMasteryReport(repID)
}

// ResetRepertoireMastery clears all practice counters and mastery flags for a repertoire and returns the fresh report.
func (a *App) ResetRepertoireMastery(repID string) (*repertoire.RepertoireMasteryReport, error) {
	if _, err := a.repMgr.ResetMastery(repID); err != nil {
		return nil, err
	}
	return a.repMgr.GetMasteryReport(repID)
}

// CompileRepertoireToPolyglot compiles a repertoire into a Polyglot (.bin) opening book and links it to the user profile if applicable.
func (a *App) CompileRepertoireToPolyglot(repID string) (string, error) {
	home, _ := os.UserHomeDir()
	bookDir := filepath.Join(home, ".local", "share", fileformat.AppDirName, fileformat.DirBooks, fileformat.DirOpeningBooks)
	if err := os.MkdirAll(bookDir, 0755); err != nil {
		return "", err
	}
	targetPath := filepath.Join(bookDir, repID+fileformat.ExtPolyglotBook)
	count, err := a.repMgr.CompileToPolyglotBook(repID, targetPath)
	if err != nil {
		return "", err
	}
	if a.personaMgr != nil {
		for _, p := range a.personaMgr.GetPersonas() {
			if p.IsUserProfile || p.WhiteRepertoireID == repID || p.BlackRepertoireID == repID {
				p.BookPath = targetPath
				_, _ = a.personaMgr.SavePersona(p)
			}
		}
	}
	a.mu.Lock()
	if a.polyglotBooks != nil {
		if b, bErr := polyglot.OpenBook(targetPath); bErr == nil {
			a.polyglotBooks[targetPath] = b
		}
	}
	a.mu.Unlock()
	return fmt.Sprintf("%s (%d positions)", targetPath, count), nil
}

// GetMainUserOnlineDefaults returns the primary user's configured Lichess/Chess.com handles and display name.
func (a *App) GetMainUserOnlineDefaults() map[string]string {
	res := map[string]string{
		"displayName":       "",
		"lichessUsername":   "",
		"chessComUsername":  "",
		"whiteRepertoireId": "",
		"blackRepertoireId": "",
	}
	if a.personaMgr != nil {
		personas := a.personaMgr.GetPersonas()
		for _, p := range personas {
			if p.IsUserProfile {
				res["displayName"] = p.DisplayName
				res["lichessUsername"] = p.LichessUsername
				res["chessComUsername"] = p.ChessComUsername
				res["whiteRepertoireId"] = p.WhiteRepertoireID
				res["blackRepertoireId"] = p.BlackRepertoireID
				break
			}
		}
		if res["lichessUsername"] == "" && res["chessComUsername"] == "" && len(personas) > 0 {
			p := personas[0]
			res["displayName"] = p.DisplayName
			res["lichessUsername"] = p.LichessUsername
			res["chessComUsername"] = p.ChessComUsername
			res["whiteRepertoireId"] = p.WhiteRepertoireID
			res["blackRepertoireId"] = p.BlackRepertoireID
		}
	}
	if a.profileMgr != nil {
		for _, prof := range a.profileMgr.GetProfiles() {
			if res["lichessUsername"] == "" && prof.LichessUsername != "" {
				res["lichessUsername"] = prof.LichessUsername
			}
			if res["chessComUsername"] == "" && prof.ChessComUsername != "" {
				res["chessComUsername"] = prof.ChessComUsername
			}
			if res["displayName"] == "" && prof.Name != "" {
				res["displayName"] = prof.Name
			}
		}
	}
	return res
}

// ScanRepertoireDeviations scans online accounts (Lichess / Chess.com) and/or local databases for times the opening book was broken.
func (a *App) ScanRepertoireDeviations(req RepertoireScanRequest) (*repertoire.BookDeviationReport, error) {
	if strings.TrimSpace(req.RepertoireID) == "" {
		return nil, fmt.Errorf("repertoire ID is required")
	}
	maxGames := req.MaxGames
	if maxGames <= 0 {
		maxGames = 100
	}
	if maxGames > 1000 {
		maxGames = 1000
	}

	source := strings.ToLower(strings.TrimSpace(req.Source))
	if source == "" {
		source = "both"
	}

	lichessUser := strings.TrimSpace(req.LichessUsername)
	chessComUser := strings.TrimSpace(req.ChessComUsername)
	playerName := strings.TrimSpace(req.PlayerName)

	var identifiers []string
	var personaDBPath string

	if a.personaMgr != nil {
		for _, p := range a.personaMgr.GetPersonas() {
			if p.IsUserProfile ||
				(playerName != "" && strings.EqualFold(p.DisplayName, playerName)) ||
				(lichessUser != "" && strings.EqualFold(p.LichessUsername, lichessUser)) ||
				(chessComUser != "" && strings.EqualFold(p.ChessComUsername, chessComUser)) {
				if lichessUser == "" {
					lichessUser = strings.TrimSpace(p.LichessUsername)
				}
				if chessComUser == "" {
					chessComUser = strings.TrimSpace(p.ChessComUsername)
				}
				if playerName == "" {
					playerName = strings.TrimSpace(p.DisplayName)
				}
				if personaDBPath == "" && p.DatabasePath != "" {
					personaDBPath = p.DatabasePath
				}
				identifiers = append(identifiers, p.DisplayName, p.LichessUsername, p.ChessComUsername)
				identifiers = append(identifiers, p.Identifiers...)
				break
			}
		}
	}

	if lichessUser != "" {
		identifiers = append(identifiers, lichessUser)
	}
	if chessComUser != "" {
		identifiers = append(identifiers, chessComUser)
	}
	if playerName != "" {
		identifiers = append(identifiers, playerName)
	}

	var games []*chess.Game

	// 1. Fetch from Online Accounts if requested
	if source == "online" || source == "both" {
		if a.syncMgr == nil {
			a.syncMgr = online.NewSyncManager(a.ensureProfileMgr())
		}
		if lichessUser != "" && a.syncMgr.Lichess != nil {
			if pgnText, err := a.syncMgr.Lichess.FetchUserGames(lichessUser, 0, "", false, maxGames); err == nil && strings.TrimSpace(pgnText) != "" {
				if parsed, pErr := chess.ParsePGNDatabase(strings.NewReader(pgnText)); pErr == nil {
					games = append(games, parsed...)
				}
			}
		}
		if chessComUser != "" && a.syncMgr.ChessCom != nil {
			if pgnText, err := a.syncMgr.ChessCom.FetchUserGames(chessComUser, 0, 0, maxGames); err == nil && strings.TrimSpace(pgnText) != "" {
				if parsed, pErr := chess.ParsePGNDatabase(strings.NewReader(pgnText)); pErr == nil {
					games = append(games, parsed...)
				}
			}
		}
	}

	// 2. Load from Local Databases if requested or if no online games were found
	if source == "database" || source == "both" || len(games) == 0 {
		// Check persona dedicated database first
		if personaDBPath != "" {
			pDB := scid.NewDatabase()
			if err := pDB.Open(personaDBPath); err == nil {
				total := pDB.NumGames()
				for id := total; id >= 1 && len(games) < maxGames*2; id-- {
					if g, gErr := pDB.GetGame(id); gErr == nil && g != nil {
						games = append(games, g)
					}
				}
				_ = pDB.Close()
			}
		}

		// Also check active database
		a.mu.RLock()
		db := a.db
		activeGame := a.activeGame
		a.mu.RUnlock()

		if db != nil && db.IsOpen {
			total := db.NumGames()
			for id := total; id >= 1 && len(games) < maxGames*2; id-- {
				if g, gErr := db.GetGame(id); gErr == nil && g != nil {
					games = append(games, g)
				}
			}
		}
		if len(games) == 0 && activeGame != nil {
			games = append(games, activeGame)
		}
	}

	return a.repMgr.ScanGamesForBookDeviations(req.RepertoireID, games, identifiers)
}
// Database Puzzle Miner Operations
func (a *App) MinePuzzlesFromDatabase(playerName string, maxGames int) ([]puzzleminer.MinedPuzzle, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.db == nil || !a.db.IsOpen {
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
// Online Account Sync (Chess.com & Lichess)
func (a *App) ensureProfileMgr() *online.ProfileManager {
	if a.profileMgr == nil {
		a.profileMgr, _ = online.NewProfileManager("")
	}
	return a.profileMgr
}

func (a *App) GetSyncProfiles() ([]*online.Profile, error) {
	pm := a.ensureProfileMgr()
	if pm == nil {
		return []*online.Profile{}, nil
	}
	return pm.GetProfiles(), nil
}

func (a *App) GetSyncProfile(id string) (*online.Profile, error) {
	pm := a.ensureProfileMgr()
	if pm == nil {
		return nil, fmt.Errorf("profile manager unavailable")
	}
	return pm.GetProfile(id)
}

func (a *App) SaveSyncProfile(p *online.Profile) error {
	pm := a.ensureProfileMgr()
	if pm == nil {
		return fmt.Errorf("profile manager unavailable")
	}
	return pm.SaveProfile(p)
}

func (a *App) DeleteSyncProfile(id string) error {
	pm := a.ensureProfileMgr()
	if pm == nil {
		return fmt.Errorf("profile manager unavailable")
	}
	return pm.DeleteProfile(id)
}

func (a *App) SyncOnlineProfile(profileID string) (*online.SyncResult, error) {
	pm := a.ensureProfileMgr()
	if pm == nil {
		return nil, fmt.Errorf("profile manager unavailable")
	}
	profile, err := pm.GetProfile(profileID)
	if err != nil {
		return nil, err
	}

	_, finish := a.startCancellableOperation("online-sync")
	defer finish()

	a.mu.Lock()
	defer a.mu.Unlock()

	var targetDB *scid.Database
	shouldCloseTarget := false

	targetPath := strings.TrimSpace(profile.TargetDatabase)
	if targetPath == "" {
		if a.db == nil || !a.db.IsOpen {
			return nil, fmt.Errorf("no target database specified in profile and no database currently open")
		}
		targetDB = a.db
	} else {
		basePath := fileformat.TrimScidExtensions(targetPath)
		targetPolicy := a.resolveDatabaseWritePolicy(basePath, 0, "")
		if !targetPolicy.AllowsOnlineSync() {
			return nil, fmt.Errorf("database %q is locked (%s); unlock or switch to Sync-Only", filepath.Base(basePath), targetPolicy.String())
		}
		if a.db != nil && a.db.IsOpen && filepath.Clean(a.db.BasePath) == filepath.Clean(basePath) {
			targetDB = a.db
		} else {
			targetDB = scid.NewDatabase()
			shouldCloseTarget = true

			if _, err := os.Stat(basePath + fileformat.ExtScidIndex); err == nil {
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
// TWIC (The Week In Chess) Bulletins
// DownloadTWICIssue downloads and imports a single TWIC issue into the currently active database.
func (a *App) DownloadTWICIssue(issue int) (*online.TWICIssueResult, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.db == nil || !a.db.IsOpen {
		return nil, fmt.Errorf("no database is currently open; open or create a database first")
	}
	if err := a.checkActiveDBSyncAllowedLocked(); err != nil {
		return nil, err
	}

	opCtx, finishOp := a.startCancellableOperation("twic-download")
	defer finishOp()

	return a.twicClient.ImportIssue(opCtx, issue, a.db)
}

// DownloadTWICRange downloads and imports a range of TWIC issues [startIssue..endIssue] into the active database.
func (a *App) DownloadTWICRange(startIssue, endIssue int) (*online.TWICBatchResult, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.db == nil || !a.db.IsOpen {
		return nil, fmt.Errorf("no database is currently open; open or create a database first")
	}
	if err := a.checkActiveDBSyncAllowedLocked(); err != nil {
		return nil, err
	}

	opCtx, finishOp := a.startCancellableOperation("twic-download")
	defer finishOp()

	progressFn := func(currentIssue, processedIssues, totalIssues, gamesImported int) {
		a.emitEvent("twic:progress", map[string]interface{}{
			"currentIssue":    currentIssue,
			"processedIssues": processedIssues,
			"totalIssues":     totalIssues,
			"gamesImported":   gamesImported,
		})
	}

	return a.twicClient.ImportRange(opCtx, startIssue, endIssue, a.db, progressFn)
}

// CheckTWICIssue checks if a particular TWIC issue exists on the server.
func (a *App) CheckTWICIssue(issue int) (bool, error) {
	return a.twicClient.CheckIssueExists(context.Background(), issue)
}

// GetLatestTWICIssue automatically detects the latest published TWIC issue number.
func (a *App) GetLatestTWICIssue() (int, error) {
	return a.twicClient.FindLatestIssue(context.Background(), 1560)
}
// Engine vs. Engine Match & Tournament
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

	// Resolve Player 1
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

	// Resolve Player 2
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
// Notation Styles & Formatting
// SetNotationStyle sets the active notation style (san, fan, lan, descriptive, uci, iccf).
func (a *App) SetNotationStyle(style string) error {
	parsed := chess.ParseNotationStyle(style)
	if parsed == chess.NotationInvalid {
		return fmt.Errorf("invalid notation style: %s", style)
	}
	a.mu.Lock()
	a.notationStyle = parsed
	a.mu.Unlock()
	return nil
}

// GetNotationStyle returns the active notation format.
func (a *App) GetNotationStyle() string {
	a.mu.RLock()
	st := a.notationStyle
	a.mu.RUnlock()
	if st == chess.NotationInvalid {
		return chess.NotationSAN.String()
	}
	return st.String()
}

func (a *App) resolveNotationStyle(style string) chess.NotationStyle {
	if st := chess.ParseNotationStyle(style); st != chess.NotationInvalid {
		return st
	}
	a.mu.RLock()
	st := a.notationStyle
	a.mu.RUnlock()
	if st == chess.NotationInvalid {
		return chess.NotationSAN
	}
	return st
}

// FormatMove formats a UCI move from a given FEN position into the target notation style.
func (a *App) FormatMove(fen string, moveUCI string, style string) (string, error) {
	return chess.FormatMoveWithFEN(fen, moveUCI, a.resolveNotationStyle(style))
}

// FormatMoveSequence formats a sequence of UCI moves into readable moves with full-move numbers.
func (a *App) FormatMoveSequence(fen string, movesUCI []string, style string) (string, error) {
	return chess.FormatPVWithStyle(fen, movesUCI, a.resolveNotationStyle(style)), nil
}
// Game Annotations & Variations
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
// Player Import & Dedicated / Blended Sync
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
// Player Simulation & Training Module
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

	if simMove, inBook := personality.SelectSimulatedMove(fen, isWhite); inBook {
		return simMove, nil
	}

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
		_ = eng.ResetStrengthOptions()
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
				Commentary: fmt.Sprintf("%s (%s, %d Elo) calculated move", personality.Name, personality.PlayStyle, personality.EstimatedElo),
			}, nil
		}
	}

	// Tactical fallback when no UCI engine is running
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
		Commentary: fmt.Sprintf("%s (%d Elo) played tactical heuristic move", personality.Name, personality.EstimatedElo),
	}, nil
}

// GetPersonaSimulatedMove queries the persona's full game repertoire and returns their most common
// historical move. If out of their historical games, it computes an engine move calibrated to their
// human Elo, contempt rating, and tactical aggression style.
func (a *App) GetPersonaSimulatedMove(personaID string, fen string, isWhite bool) (*playerbot.SimulatedMove, error) {
	var pObj *persona.Persona
	if a.personaMgr != nil {
		pObj, _ = a.personaMgr.GetPersona(personaID)
	}

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
			if pObj != nil {
				elo := pObj.EstimatedElo
				if elo <= 0 {
					elo = 1500
				}
				personality = &playerbot.PlayerPersonality{
					Name:         pObj.DisplayName,
					EstimatedElo: elo,
					PlayStyle:    "Universal",
				}
			} else {
				return nil, err
			}
		} else {
			a.mu.Lock()
			a.personaBotCache[personaID] = personality
			a.mu.Unlock()
		}
	}

	if pObj != nil && a.repMgr != nil {
		repID := strings.TrimSpace(pObj.BlackRepertoireID)
		if isWhite {
			repID = strings.TrimSpace(pObj.WhiteRepertoireID)
		}
		if repID != "" {
			if rep, rErr := a.repMgr.GetRepertoire(repID); rErr == nil && rep != nil {
				core := repertoire.CoreFEN(fen)
				if opts := rep.Positions[core]; len(opts) > 0 {
					chosen := opts[0]
					return &playerbot.SimulatedMove{
						UCI:        chosen.UCI,
						SAN:        chosen.SAN,
						IsBook:     true,
						EloTarget:  personality.EstimatedElo,
						Commentary: fmt.Sprintf("%s played repertoire move %s (%s)", personality.Name, chosen.SAN, rep.Name),
					}, nil
				}
			}
		}
	}

	if pObj != nil && strings.TrimSpace(pObj.BookPath) != "" {
		if bm, bErr := a.GetPolyglotBookMove(fen, pObj.BookPath, false); bErr == nil && bm != nil {
			return &playerbot.SimulatedMove{
				UCI:        bm.UCI,
				SAN:        bm.SAN,
				IsBook:     true,
				EloTarget:  personality.EstimatedElo,
				Commentary: fmt.Sprintf("%s played assigned book move %s (%.0f%% weight)", personality.Name, bm.SAN, bm.Percentage),
			}, nil
		}
	}

	if simMove, inBook := personality.SelectSimulatedMove(fen, isWhite); inBook {
		return simMove, nil
	}

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
		_ = eng.ResetStrengthOptions()
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
				Commentary: fmt.Sprintf("%s (%s, %d Elo) calculated move", personality.Name, personality.PlayStyle, personality.EstimatedElo),
			}, nil
		}
	}

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
		Commentary: fmt.Sprintf("%s (%d Elo) played tactical heuristic move", personality.Name, personality.EstimatedElo),
	}, nil
}
