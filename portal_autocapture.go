package main

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"newscid/pkg/chess"
	"newscid/pkg/database/scid"
	"newscid/pkg/online"
	"newscid/pkg/workstation"
)

// LivePortalCaptureStatus describes the current state of automatic game capture from Lichess.org and Chess.com.
type LivePortalCaptureStatus struct {
	Enabled                 bool   `json:"enabled"`
	AutoLoadToBoard         bool   `json:"autoLoadToBoard"`
	WatcherActive           bool   `json:"watcherActive"`
	IntervalSeconds         int    `json:"intervalSeconds"`
	LichessUsername         string `json:"lichessUsername"`
	ChessComUsername        string `json:"chessComUsername"`
	TargetDatabasePath      string `json:"targetDatabasePath"`
	TargetDatabaseName      string `json:"targetDatabaseName"`
	LastCheckAtMs           int64  `json:"lastCheckAtMs"`
	LastCapturedAtMs        int64  `json:"lastCapturedAtMs"`
	LastCapturedGameID      int    `json:"lastCapturedGameId"`
	LastCapturedGameSummary string `json:"lastCapturedGameSummary"`
	SessionCapturedCount    int    `json:"sessionCapturedCount"`
	LastError               string `json:"lastError,omitempty"`
}

// LivePortalCaptureResult reports the outcome of a live portal game check or direct game URL capture.
type LivePortalCaptureResult struct {
	CheckedAtMs        int64                    `json:"checkedAtMs"`
	NewGamesImported   int                      `json:"newGamesImported"`
	LichessImported    int                      `json:"lichessImported"`
	ChessComImported   int                      `json:"chessComImported"`
	DuplicatesSkipped  int                      `json:"duplicatesSkipped"`
	LatestGameID       int                      `json:"latestGameId"`
	LatestGameSummary  string                   `json:"latestGameSummary"`
	LatestGamePGN      string                   `json:"latestGamePgn,omitempty"`
	TargetDatabasePath string                   `json:"targetDatabasePath"`
	TargetDatabaseName string                   `json:"targetDatabaseName"`
	AutoLoadToBoard    bool                     `json:"autoLoadToBoard"`
	Status             *LivePortalCaptureStatus `json:"status"`
}

var (
	portalWatchMu          sync.Mutex
	portalWatcherStopCh    chan struct{}
	portalWatcherActive    bool
	portalLastCheckMs      int64
	portalLastCapturedMs   int64
	portalLastCapturedID   int
	portalLastCapturedDesc string
	portalSessionCaptured  int
	portalLastError        string
	portalCheckInFlight    bool
	portalPollInterval     = 25 * time.Second
)

// resolvePortalCredentials discovers the user's Lichess and Chess.com usernames (and optional Lichess token)
// from AppPreferences and falls back to any configured OnlineSync profile.
func (a *App) resolvePortalCredentials() (lichessUser, lichessToken, chessComUser string) {
	prefs, _ := a.GetAppPreferences()
	if prefs != nil {
		lichessUser = strings.TrimSpace(prefs.LichessUsername)
		chessComUser = strings.TrimSpace(prefs.ChessComUsername)
	}
	if pm := a.ensureProfileMgr(); pm != nil {
		profiles := pm.GetProfiles()
		for _, p := range profiles {
			if p == nil {
				continue
			}
			if lichessUser == "" && strings.TrimSpace(p.LichessUsername) != "" {
				lichessUser = strings.TrimSpace(p.LichessUsername)
			}
			if lichessToken == "" && strings.TrimSpace(p.LichessToken) != "" {
				lichessToken = strings.TrimSpace(p.LichessToken)
			}
			if chessComUser == "" && strings.TrimSpace(p.ChessComUsername) != "" {
				chessComUser = strings.TrimSpace(p.ChessComUsername)
			}
		}
	}
	return lichessUser, lichessToken, chessComUser
}

// resolveAutoCaptureTargetPath determines which database should receive auto-captured live games.
// Preference order:
// 1. Default Personal User Database (if configured and not Locked)
// 2. Active Database (if open, editable, and not a Reference/Mega archive)
// 3. Managed Personal User Database (~/.local/share/newscid/databases/My_Games)
func (a *App) resolveAutoCaptureTargetPath() string {
	prefs, _ := a.GetAppPreferences()
	if prefs != nil && strings.TrimSpace(prefs.DefaultDatabasePath) != "" {
		defClean := normalizeScidBasePath(prefs.DefaultDatabasePath)
		pol := a.resolveDatabaseWritePolicy(defClean, 0, "User Databases")
		if pol.AllowsOnlineSync() {
			return defClean
		}
	}

	a.mu.RLock()
	activeOpen := a.db != nil && a.db.IsOpen
	activeBase := ""
	activeCount := 0
	if activeOpen {
		activeBase = normalizeScidBasePath(a.db.BasePath)
		activeCount = a.db.NumGames()
	}
	a.mu.RUnlock()

	if activeOpen && activeBase != "" {
		pol := a.resolveDatabaseWritePolicy(activeBase, activeCount, "")
		if pol == scid.WritePolicyEditable {
			return activeBase
		}
	}

	if dataRoot, _ := workstation.DefaultWorkstationDirs(); dataRoot != "" {
		dbDir := filepath.Join(dataRoot, "databases")
		_ = os.MkdirAll(dbDir, 0o755)
		return filepath.Join(dbDir, "My_Games")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "newscid", "databases", "My_Games")
}

// GetLivePortalCaptureStatus returns the current configuration and live status of portal game auto-capture.
func (a *App) GetLivePortalCaptureStatus() (*LivePortalCaptureStatus, error) {
	prefs, _ := a.GetAppPreferences()
	enabled := true
	autoLoad := true
	if prefs != nil {
		enabled = prefs.AutoCapturePortalGames
		autoLoad = prefs.AutoLoadCapturedGame
	}

	lichessUser, _, chessComUser := a.resolvePortalCredentials()
	targetPath := a.resolveAutoCaptureTargetPath()
	targetName := filepath.Base(targetPath)

	portalWatchMu.Lock()
	defer portalWatchMu.Unlock()

	return &LivePortalCaptureStatus{
		Enabled:                 enabled,
		AutoLoadToBoard:         autoLoad,
		WatcherActive:           portalWatcherActive,
		IntervalSeconds:         int(portalPollInterval.Seconds()),
		LichessUsername:         lichessUser,
		ChessComUsername:        chessComUser,
		TargetDatabasePath:      targetPath,
		TargetDatabaseName:      targetName,
		LastCheckAtMs:           portalLastCheckMs,
		LastCapturedAtMs:        portalLastCapturedMs,
		LastCapturedGameID:      portalLastCapturedID,
		LastCapturedGameSummary: portalLastCapturedDesc,
		SessionCapturedCount:    portalSessionCaptured,
		LastError:               portalLastError,
	}, nil
}

// ConfigureLivePortalAutoCapture updates the user's Lichess/Chess.com usernames and auto-capture preferences,
// saves them to AppPreferences (and syncs with the default OnlineSync profile), and starts/stops the live watcher.
func (a *App) ConfigureLivePortalAutoCapture(
	lichessUsername string,
	chessComUsername string,
	enabled bool,
	autoLoadToBoard bool,
) (*LivePortalCaptureStatus, error) {
	prefs, err := a.GetAppPreferences()
	if err != nil || prefs == nil {
		prefs = defaultAppPreferences()
	}

	prefs.LichessUsername = strings.TrimSpace(lichessUsername)
	prefs.ChessComUsername = strings.TrimSpace(chessComUsername)
	prefs.AutoCapturePortalGames = enabled
	prefs.AutoLoadCapturedGame = autoLoadToBoard
	_, _ = a.SaveAppPreferences(*prefs)

	// Also keep the primary OnlineSync profile in sync if ProfileManager is available
	if pm := a.ensureProfileMgr(); pm != nil && (prefs.LichessUsername != "" || prefs.ChessComUsername != "") {
		profiles := pm.GetProfiles()
		if len(profiles) > 0 && profiles[0] != nil {
			p := profiles[0]
			if prefs.LichessUsername != "" {
				p.LichessUsername = prefs.LichessUsername
			}
			if prefs.ChessComUsername != "" {
				p.ChessComUsername = prefs.ChessComUsername
			}
			_ = pm.SaveProfile(p)
		} else {
			name := prefs.LichessUsername
			if name == "" {
				name = prefs.ChessComUsername
			}
			_ = pm.SaveProfile(&online.Profile{
				ID:               "portal-default",
				Name:             name,
				LichessUsername:  prefs.LichessUsername,
				ChessComUsername: prefs.ChessComUsername,
				TargetMode:       online.TargetActiveBase,
				AutoDeduplicate:  true,
			})
		}
	}

	if enabled && (prefs.LichessUsername != "" || prefs.ChessComUsername != "") {
		_, _ = a.StartLivePortalWatcher()
	} else if !enabled {
		_, _ = a.StopLivePortalWatcher()
	}

	return a.GetLivePortalCaptureStatus()
}

// StartLivePortalWatcher starts the background goroutine that automatically checks Lichess.org and Chess.com
// for newly completed games and adds them to the user's Personal Database.
func (a *App) StartLivePortalWatcher() (*LivePortalCaptureStatus, error) {
	portalWatchMu.Lock()
	if portalWatcherActive {
		portalWatchMu.Unlock()
		return a.GetLivePortalCaptureStatus()
	}
	stopCh := make(chan struct{})
	portalWatcherStopCh = stopCh
	portalWatcherActive = true
	interval := portalPollInterval
	if interval < 50*time.Millisecond {
		interval = 25 * time.Second
	}
	portalWatchMu.Unlock()

	go func(stop <-chan struct{}, pollEvery time.Duration) {
		// Perform an initial check shortly after starting
		timer := time.NewTimer(1500 * time.Millisecond)
		defer timer.Stop()

		for {
			select {
			case <-stop:
				return
			case <-timer.C:
				prefs, _ := a.GetAppPreferences()
				if prefs != nil && !prefs.AutoCapturePortalGames {
					_, _ = a.StopLivePortalWatcher()
					return
				}
				lichessUser, _, chessComUser := a.resolvePortalCredentials()
				if lichessUser != "" || chessComUser != "" {
					_, _ = a.CheckAndCaptureLivePortalGames()
				}
				timer.Reset(pollEvery)
			}
		}
	}(stopCh, interval)

	return a.GetLivePortalCaptureStatus()
}

// StopLivePortalWatcher stops the background live game auto-capture loop.
func (a *App) StopLivePortalWatcher() (*LivePortalCaptureStatus, error) {
	portalWatchMu.Lock()
	if portalWatcherActive && portalWatcherStopCh != nil {
		close(portalWatcherStopCh)
		portalWatcherStopCh = nil
	}
	portalWatcherActive = false
	portalWatchMu.Unlock()

	return a.GetLivePortalCaptureStatus()
}

// CheckAndCaptureLivePortalGames immediately polls the user's configured Lichess.org and Chess.com accounts
// for any newly finished games (inspecting only the most recent games) and imports them into the Personal User Database.
func (a *App) CheckAndCaptureLivePortalGames() (*LivePortalCaptureResult, error) {
	portalWatchMu.Lock()
	if portalCheckInFlight {
		portalWatchMu.Unlock()
		st, _ := a.GetLivePortalCaptureStatus()
		return &LivePortalCaptureResult{
			CheckedAtMs: time.Now().UnixMilli(),
			Status:      st,
		}, nil
	}
	portalCheckInFlight = true
	portalWatchMu.Unlock()

	defer func() {
		portalWatchMu.Lock()
		portalCheckInFlight = false
		portalWatchMu.Unlock()
	}()

	nowMs := time.Now().UnixMilli()
	lichessUser, lichessToken, chessComUser := a.resolvePortalCredentials()
	if lichessUser == "" && chessComUser == "" {
		portalWatchMu.Lock()
		portalLastCheckMs = nowMs
		portalWatchMu.Unlock()
		st, _ := a.GetLivePortalCaptureStatus()
		return &LivePortalCaptureResult{
			CheckedAtMs: nowMs,
			Status:      st,
		}, fmt.Errorf("configure your Lichess or Chess.com username in the Live Portal bar to enable automatic game capture")
	}

	targetPath := a.resolveAutoCaptureTargetPath()
	existingSigs, err := a.buildTargetSignatureSet(targetPath)
	if err != nil {
		existingSigs = scid.NewGameSignatureSet()
	}

	var newGames []*chess.Game
	lichessAdded := 0
	chessComAdded := 0
	skippedTotal := 0
	var errs []string

	if lichessUser != "" && a.syncMgr != nil && a.syncMgr.Lichess != nil {
		lgames, skipped, lerr := a.syncMgr.Lichess.FetchRecentNewGames(lichessUser, lichessToken, 8, existingSigs)
		if lerr != nil {
			errs = append(errs, fmt.Sprintf("Lichess: %v", lerr))
		} else {
			lichessAdded = len(lgames)
			skippedTotal += skipped
			newGames = append(newGames, lgames...)
		}
	}

	if chessComUser != "" && a.syncMgr != nil && a.syncMgr.ChessCom != nil {
		cgames, skipped, cerr := a.syncMgr.ChessCom.FetchRecentNewGames(chessComUser, 8, existingSigs)
		if cerr != nil {
			errs = append(errs, fmt.Sprintf("Chess.com: %v", cerr))
		} else {
			chessComAdded = len(cgames)
			skippedTotal += skipped
			newGames = append(newGames, cgames...)
		}
	}

	portalWatchMu.Lock()
	portalLastCheckMs = nowMs
	if len(errs) > 0 && len(newGames) == 0 {
		portalLastError = strings.Join(errs, "; ")
	} else {
		portalLastError = ""
	}
	portalWatchMu.Unlock()

	if len(newGames) == 0 {
		st, _ := a.GetLivePortalCaptureStatus()
		return &LivePortalCaptureResult{
			CheckedAtMs:        nowMs,
			DuplicatesSkipped:  skippedTotal,
			TargetDatabasePath: targetPath,
			TargetDatabaseName: filepath.Base(targetPath),
			AutoLoadToBoard:    st.AutoLoadToBoard,
			Status:             st,
		}, nil
	}

	return a.commitCapturedGamesToDatabase(targetPath, newGames, lichessAdded, chessComAdded, skippedTotal)
}

var (
	lichessURLRegex  = regexp.MustCompile(`(?i)(?:lichess\.org/)?(?:game/export/|embed/)?([a-zA-Z0-9]{8,12})(?:/(?:white|black))?`)
	chessComURLRegex = regexp.MustCompile(`(?i)chess\.com/(?:game/(?:live|daily)|live/game)/(\d+)`)
)

// CaptureOnlineGameFromURLOrPGN captures a single game from a Lichess URL/ID, Chess.com URL/ID, or raw PGN string,
// automatically saves it into the user's Personal Database (deduplicating against existing games), and returns its DB ID.
func (a *App) CaptureOnlineGameFromURLOrPGN(input string) (*LivePortalCaptureResult, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return nil, fmt.Errorf("provide a Lichess URL, Chess.com URL, or PGN text to capture")
	}

	var parsedGames []*chess.Game
	lichessAdded := 0
	chessComAdded := 0

	_, lichessToken, chessComUser := a.resolvePortalCredentials()

	if strings.Contains(trimmed, "[Event ") || strings.HasPrefix(trimmed, "1.") {
		games, err := chess.ParsePGNDatabase(strings.NewReader(trimmed))
		if err != nil || len(games) == 0 {
			g, singleErr := chess.ParsePGN(trimmed)
			if singleErr != nil {
				return nil, fmt.Errorf("failed to parse PGN: %w", singleErr)
			}
			games = []*chess.Game{g}
		}
		parsedGames = games
	} else if m := chessComURLRegex.FindStringSubmatch(trimmed); len(m) >= 2 {
		gameID := m[1]
		if a.syncMgr == nil || a.syncMgr.ChessCom == nil {
			return nil, fmt.Errorf("chess.com client is not initialized")
		}
		g, err := a.syncMgr.ChessCom.FetchGameByID(gameID, chessComUser)
		if err != nil {
			return nil, err
		}
		parsedGames = []*chess.Game{g}
		chessComAdded = 1
	} else if strings.Contains(strings.ToLower(trimmed), "lichess.org") || isLikelyLichessGameID(trimmed) {
		gameID := extractLichessGameID(trimmed)
		if gameID == "" {
			return nil, fmt.Errorf("could not extract Lichess 8-character game ID from %q", trimmed)
		}
		if a.syncMgr == nil || a.syncMgr.Lichess == nil {
			return nil, fmt.Errorf("lichess client is not initialized")
		}
		g, err := a.syncMgr.Lichess.FetchGameByID(gameID, lichessToken)
		if err != nil {
			return nil, err
		}
		parsedGames = []*chess.Game{g}
		lichessAdded = 1
	} else {
		return nil, fmt.Errorf("unrecognized game URL or PGN input %q", trimmed)
	}

	targetPath := a.resolveAutoCaptureTargetPath()
	existingSigs, _ := a.buildTargetSignatureSet(targetPath)

	var newGames []*chess.Game
	skipped := 0
	for _, g := range parsedGames {
		if g == nil {
			continue
		}
		if existingSigs != nil && existingSigs.ContainsGame(g) {
			skipped++
			continue
		}
		if existingSigs != nil {
			existingSigs.AddGame(g)
		}
		newGames = append(newGames, g)
	}

	// Even if the game was already in the database, return its PGN so the board can load it immediately
	if len(newGames) == 0 && len(parsedGames) > 0 && parsedGames[0] != nil {
		lastG := parsedGames[len(parsedGames)-1]
		summary := formatCapturedGameSummary(lastG)
		st, _ := a.GetLivePortalCaptureStatus()
		return &LivePortalCaptureResult{
			CheckedAtMs:        time.Now().UnixMilli(),
			NewGamesImported:   0,
			DuplicatesSkipped:  skipped,
			LatestGameSummary:  summary,
			LatestGamePGN:      lastG.ExportPGN(),
			TargetDatabasePath: targetPath,
			TargetDatabaseName: filepath.Base(targetPath),
			AutoLoadToBoard:    true,
			Status:             st,
		}, nil
	}

	return a.commitCapturedGamesToDatabase(targetPath, newGames, lichessAdded, chessComAdded, skipped)
}

func isLikelyLichessGameID(s string) bool {
	if len(s) != 8 && len(s) != 12 {
		return false
	}
	for _, ch := range s {
		if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9')) {
			return false
		}
	}
	return true
}

func extractLichessGameID(raw string) string {
	s := strings.TrimSpace(raw)
	if u, err := url.Parse(s); err == nil && u.Host != "" {
		segments := strings.Split(strings.Trim(u.Path, "/"), "/")
		for _, seg := range segments {
			low := strings.ToLower(seg)
			if low == "game" || low == "export" || low == "embed" || low == "white" || low == "black" || low == "analysis" {
				continue
			}
			if len(seg) == 12 {
				return seg[:8]
			}
			if len(seg) == 8 {
				return seg
			}
		}
	}
	if m := lichessURLRegex.FindStringSubmatch(s); len(m) >= 2 {
		id := m[1]
		if len(id) == 12 {
			return id[:8]
		}
		if len(id) == 8 {
			return id
		}
	}
	return ""
}

func formatCapturedGameSummary(g *chess.Game) string {
	if g == nil || g.Tags == nil {
		return "Online Game"
	}
	w := g.Tags["White"]
	if w == "" {
		w = "?"
	}
	b := g.Tags["Black"]
	if b == "" {
		b = "?"
	}
	res := g.Tags["Result"]
	if res == "" {
		res = string(g.Result)
	}
	if res == "" {
		res = "*"
	}
	ev := g.Tags["Event"]
	if ev != "" && ev != "?" {
		return fmt.Sprintf("%s vs %s (%s) • %s", w, b, res, ev)
	}
	return fmt.Sprintf("%s vs %s (%s)", w, b, res)
}

func openOrCreatePortalTargetDB(basePath string, desc string) (*scid.Database, error) {
	db := scid.NewDatabase()
	if _, err := os.Stat(basePath + ".si4"); err == nil {
		if err := db.Open(basePath); err != nil {
			return nil, err
		}
		return db, nil
	}
	if err := db.Create(basePath, desc); err != nil {
		return nil, err
	}
	return db, nil
}

func (a *App) buildTargetSignatureSet(targetBasePath string) (*scid.GameSignatureSet, error) {
	cleanTarget := normalizeScidBasePath(targetBasePath)

	a.mu.RLock()
	if a.db != nil && a.db.IsOpen && normalizeScidBasePath(a.db.BasePath) == cleanTarget {
		sigs := a.db.BuildGameSignatureSet()
		a.mu.RUnlock()
		return sigs, nil
	}
	a.mu.RUnlock()

	if _, err := os.Stat(cleanTarget + ".si4"); err != nil {
		return scid.NewGameSignatureSet(), nil
	}

	tmpDB := scid.NewDatabase()
	if err := tmpDB.Open(cleanTarget); err != nil {
		return scid.NewGameSignatureSet(), err
	}
	defer tmpDB.Close()
	return tmpDB.BuildGameSignatureSet(), nil
}

func (a *App) commitCapturedGamesToDatabase(
	targetBasePath string,
	newGames []*chess.Game,
	lichessCount int,
	chessComCount int,
	skippedCount int,
) (*LivePortalCaptureResult, error) {
	cleanTarget := normalizeScidBasePath(targetBasePath)
	_ = os.MkdirAll(filepath.Dir(cleanTarget), 0o755)

	var latestGameID int
	lastGame := newGames[len(newGames)-1]
	summary := formatCapturedGameSummary(lastGame)
	latestPGN := ""
	if lastGame != nil {
		latestPGN = lastGame.ExportPGN()
	}

	a.mu.Lock()
	activeMatches := a.db != nil && a.db.IsOpen && normalizeScidBasePath(a.db.BasePath) == cleanTarget
	noActiveDB := a.db == nil || !a.db.IsOpen
	if activeMatches {
		if _, err := a.db.ImportGames(newGames); err != nil {
			a.mu.Unlock()
			return nil, err
		}
		latestGameID = a.db.NumGames()
		a.activeGame = lastGame
		a.mu.Unlock()
	} else {
		a.mu.Unlock()
		targetDB, err := openOrCreatePortalTargetDB(cleanTarget, "Personal User Database (Auto-Captured Online Games)")
		if err != nil {
			return nil, err
		}
		if _, err := targetDB.ImportGames(newGames); err != nil {
			_ = targetDB.Close()
			return nil, err
		}
		latestGameID = targetDB.NumGames()
		_ = targetDB.Close()

		// Ensure the Personal User DB is marked editable and set as Default User DB if none was set
		_, _ = a.SetDatabaseWritePolicy(cleanTarget, string(scid.WritePolicyEditable))
		prefs, _ := a.GetAppPreferences()
		if prefs != nil && strings.TrimSpace(prefs.DefaultDatabasePath) == "" {
			prefs.DefaultDatabasePath = cleanTarget
			_, _ = a.SaveAppPreferences(*prefs)
		}

		// If no database was currently open in the workspace, open the Personal User DB automatically
		if noActiveDB {
			_, _ = a.OpenDatabase(cleanTarget)
		}
	}

	nowMs := time.Now().UnixMilli()
	portalWatchMu.Lock()
	portalLastCheckMs = nowMs
	portalLastCapturedMs = nowMs
	portalLastCapturedID = latestGameID
	portalLastCapturedDesc = summary
	portalSessionCaptured += len(newGames)
	portalLastError = ""
	portalWatchMu.Unlock()

	st, _ := a.GetLivePortalCaptureStatus()
	res := &LivePortalCaptureResult{
		CheckedAtMs:        nowMs,
		NewGamesImported:   len(newGames),
		LichessImported:    lichessCount,
		ChessComImported:   chessComCount,
		DuplicatesSkipped:  skippedCount,
		LatestGameID:       latestGameID,
		LatestGameSummary:  summary,
		LatestGamePGN:      latestPGN,
		TargetDatabasePath: cleanTarget,
		TargetDatabaseName: filepath.Base(cleanTarget),
		AutoLoadToBoard:    st.AutoLoadToBoard,
		Status:             st,
	}

	// Emit real-time event to frontend so Game Browser & Analysis Board update automatically
	a.emitEvent("portal:game-captured", res)

	// Also broadcast across multi-screen Hyprland windows via SharedWorkspaceState
	if st.AutoLoadToBoard && latestGameID > 0 {
		_ = a.PublishWorkspaceState(SharedWorkspaceState{
			UpdatedAtMs:   nowMs,
			SourceID:      "portal-autocapture",
			DBPath:        cleanTarget,
			DBName:        filepath.Base(cleanTarget),
			ActiveGameID:  latestGameID,
			PendingLoadID: latestGameID,
		})
	}

	return res, nil
}
