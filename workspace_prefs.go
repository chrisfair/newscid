package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"newscid/pkg/database/scid"
)

// AppPreferences stores persistent user preferences in ~/.newscid/preferences.json.
type AppPreferences struct {
	DefaultDatabasePath   string `json:"defaultDatabasePath"`
	ReferenceDatabasePath string `json:"referenceDatabasePath"`
	AutoOpenDefaultDB     bool   `json:"autoOpenDefaultDb"`
	UseReferenceDBForTree bool   `json:"useReferenceDbForTree"`
	DefaultPieceSet       string `json:"defaultPieceSet"`
	DefaultBoardTheme     string `json:"defaultBoardTheme"`
	DefaultUITheme        string `json:"defaultUITheme"`
	DefaultNotationStyle  string `json:"defaultNotationStyle"`
	SoundEnabled          bool   `json:"soundEnabled"`
	ShowEvalBar           bool   `json:"showEvalBar"`
	ThreatModeDefault     bool   `json:"threatModeDefault"`
	DefaultEngine         string `json:"defaultEngine"`
	DefaultMultiPV        int    `json:"defaultMultiPv"`
	DefaultThreads        int    `json:"defaultThreads"`
	DefaultHashMB         int    `json:"defaultHashMb"`
	AnnotateMoveTimeMs    int    `json:"annotateMoveTimeMs"`
	LichessUsername       string `json:"lichessUsername"`
	ChessComUsername      string `json:"chessComUsername"`
	DefaultPortalPlatform string `json:"defaultPortalPlatform"` // "lichess" | "chesscom"
	PortalWindowMode      string `json:"portalWindowMode"`      // "browser-app" | "native-window" | "floater" | "docked"
	CustomBrowserCmd      string `json:"customBrowserCmd"`
}

// DatabaseRolesStatus describes how the user's databases are currently assigned and related.
type DatabaseRolesStatus struct {
	ActiveDB        *DatabaseInfo `json:"activeDb"`
	DefaultDBPath   string        `json:"defaultDbPath"`
	DefaultDBName   string        `json:"defaultDbName"`
	IsActiveDefault bool          `json:"isActiveDefault"`
	ReferenceDB     *DatabaseInfo `json:"referenceDb"`
	ReferenceDBPath string        `json:"referenceDbPath"`
	ReferenceDBName string        `json:"referenceDbName"`
	UseRefForTree   bool          `json:"useRefForTree"`
	ClipbasePath    string        `json:"clipbasePath"`
	IsClipbase      bool          `json:"isClipbase"`
}

// WindowStartupConfig tells the frontend whether this window was launched as the primary
// workspace, a detached multi-screen panel for Hyprland, or a dedicated live portal window.
type WindowStartupConfig struct {
	Mode        string `json:"mode"`        // "main" | "panel" | "portal"
	PanelID     string `json:"panelId"`     // e.g. "tree", "engine", "notation", "games", "board", "portal", "db-hub"
	PortalURL   string `json:"portalUrl"`   // e.g. "https://lichess.org" or "https://www.chess.com"
	PortalTitle string `json:"portalTitle"` // Window title
	InitialFEN  string `json:"initialFen"`
	InitialDB   string `json:"initialDb"`
}

// SharedWorkspaceState synchronizes live board/database state across multiple OS windows on Hyprland.
type SharedWorkspaceState struct {
	UpdatedAtMs    int64             `json:"updatedAtMs"`
	SourceID       string            `json:"sourceId"`
	FEN            string            `json:"fen"`
	LastMoveFrom   string            `json:"lastMoveFrom"`
	LastMoveTo     string            `json:"lastMoveTo"`
	CurrentPly     int               `json:"currentPly"`
	DBPath         string            `json:"dbPath"`
	DBName         string            `json:"dbName"`
	ActiveGameID   int               `json:"activeGameId"`
	GameTags       map[string]string `json:"gameTags"`
	PendingMoveUCI string            `json:"pendingMoveUci,omitempty"`
	PendingLoadID  int               `json:"pendingLoadId,omitempty"`
}

var (
	windowStartupCfg = WindowStartupConfig{Mode: "main"}
	prefsMu          sync.Mutex
	workspaceSyncMu  sync.Mutex
)

func preferencesFilePath() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return filepath.Join(os.TempDir(), "newscid_preferences.json")
	}
	dir := filepath.Join(home, ".newscid")
	_ = os.MkdirAll(dir, 0o755)
	return filepath.Join(dir, "preferences.json")
}

func workspaceSyncFilePath() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return filepath.Join(os.TempDir(), "newscid_workspace_sync.json")
	}
	dir := filepath.Join(home, ".newscid")
	_ = os.MkdirAll(dir, 0o755)
	return filepath.Join(dir, "workspace_sync.json")
}

func clipbaseDefaultPath() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return filepath.Join(os.TempDir(), "Clipbase")
	}
	dir := filepath.Join(home, ".local", "share", "newscid", "databases")
	_ = os.MkdirAll(dir, 0o755)
	return filepath.Join(dir, "Clipbase")
}

func normalizeScidBasePath(p string) string {
	p = strings.TrimSpace(p)
	for _, ext := range []string{".si4", ".sg4", ".sn4", ".pgn"} {
		if strings.HasSuffix(strings.ToLower(p), ext) {
			return p[:len(p)-len(ext)]
		}
	}
	return p
}

func defaultAppPreferences() *AppPreferences {
	return &AppPreferences{
		DefaultDatabasePath:   "",
		ReferenceDatabasePath: "",
		AutoOpenDefaultDB:     true,
		UseReferenceDBForTree: true,
		DefaultPieceSet:       "cburnett",
		DefaultBoardTheme:     "classic-wood",
		DefaultUITheme:        "dark",
		DefaultNotationStyle:  "san",
		SoundEnabled:          true,
		ShowEvalBar:           true,
		ThreatModeDefault:     false,
		DefaultEngine:         "",
		DefaultMultiPV:        1,
		DefaultThreads:        2,
		DefaultHashMB:         64,
		AnnotateMoveTimeMs:    150,
		LichessUsername:       "",
		ChessComUsername:      "",
		DefaultPortalPlatform: "lichess",
		PortalWindowMode:      "browser-app",
		CustomBrowserCmd:      "",
	}
}

// GetAppPreferences loads the user's preferences from ~/.newscid/preferences.json.
func (a *App) GetAppPreferences() (*AppPreferences, error) {
	prefsMu.Lock()
	defer prefsMu.Unlock()

	prefs := defaultAppPreferences()
	path := preferencesFilePath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return prefs, nil
		}
		return prefs, nil
	}
	_ = json.Unmarshal(data, prefs)
	if prefs.DefaultMultiPV <= 0 {
		prefs.DefaultMultiPV = 1
	}
	if prefs.DefaultThreads <= 0 {
		prefs.DefaultThreads = 2
	}
	if prefs.DefaultHashMB <= 0 {
		prefs.DefaultHashMB = 64
	}
	if prefs.AnnotateMoveTimeMs <= 0 {
		prefs.AnnotateMoveTimeMs = 150
	}
	if prefs.DefaultPortalPlatform == "" {
		prefs.DefaultPortalPlatform = "lichess"
	}
	if prefs.PortalWindowMode == "" {
		prefs.PortalWindowMode = "browser-app"
	}
	return prefs, nil
}

// SaveAppPreferences persists the user's preferences to ~/.newscid/preferences.json.
func (a *App) SaveAppPreferences(prefs AppPreferences) (*AppPreferences, error) {
	prefsMu.Lock()
	prefs.DefaultDatabasePath = normalizeScidBasePath(prefs.DefaultDatabasePath)
	prefs.ReferenceDatabasePath = normalizeScidBasePath(prefs.ReferenceDatabasePath)
	path := preferencesFilePath()
	data, err := json.MarshalIndent(prefs, "", "  ")
	if err != nil {
		prefsMu.Unlock()
		return nil, err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		prefsMu.Unlock()
		return nil, err
	}
	prefsMu.Unlock()

	// Sync reference DB if changed
	if prefs.ReferenceDatabasePath != "" {
		_ = a.openReferenceDatabaseInternal(prefs.ReferenceDatabasePath)
	} else {
		a.mu.Lock()
		if a.refDB != nil {
			_ = a.refDB.Close()
			a.refDB = nil
		}
		a.mu.Unlock()
	}

	return &prefs, nil
}

func (a *App) openReferenceDatabaseInternal(basePath string) error {
	clean := normalizeScidBasePath(basePath)
	if clean == "" {
		return nil
	}
	a.mu.Lock()
	if a.refDB != nil && a.refDB.IsOpen && normalizeScidBasePath(a.refDB.BasePath) == clean {
		a.mu.Unlock()
		return nil
	}
	if a.refDB != nil {
		_ = a.refDB.Close()
		a.refDB = nil
	}
	a.mu.Unlock()

	ref := scid.NewDatabase()
	if err := ref.Open(clean); err != nil {
		return err
	}
	a.mu.Lock()
	a.refDB = ref
	a.mu.Unlock()
	return nil
}

// SetDefaultDatabase sets (or clears if empty) the Default Startup Database path.
func (a *App) SetDefaultDatabase(basePath string) (*DatabaseRolesStatus, error) {
	prefs, _ := a.GetAppPreferences()
	prefs.DefaultDatabasePath = normalizeScidBasePath(basePath)
	if prefs.DefaultDatabasePath != "" {
		prefs.AutoOpenDefaultDB = true
	}
	_, err := a.SaveAppPreferences(*prefs)
	if err != nil {
		return nil, err
	}
	return a.GetDatabaseRolesStatus()
}

// SetReferenceDatabase sets (or clears if empty) the Reference Database used for Opening Tree and Opponent Prep.
func (a *App) SetReferenceDatabase(basePath string, useForTree bool) (*DatabaseRolesStatus, error) {
	prefs, _ := a.GetAppPreferences()
	prefs.ReferenceDatabasePath = normalizeScidBasePath(basePath)
	prefs.UseReferenceDBForTree = useForTree
	_, err := a.SaveAppPreferences(*prefs)
	if err != nil {
		return nil, err
	}
	return a.GetDatabaseRolesStatus()
}

// OpenDefaultDatabase opens the user's configured Default Database as the Active Working Database.
func (a *App) OpenDefaultDatabase() (*DatabaseInfo, error) {
	prefs, _ := a.GetAppPreferences()
	if prefs.DefaultDatabasePath == "" {
		return nil, fmt.Errorf("no default database is configured yet — select any database and click 'Set as Default'")
	}
	return a.OpenDatabase(prefs.DefaultDatabasePath)
}

// OpenClipbase opens (creating if necessary) the temporary Clipbase scratchpad database.
func (a *App) OpenClipbase() (*DatabaseInfo, error) {
	clipPath := clipbaseDefaultPath()
	if _, err := os.Stat(clipPath + ".si4"); err == nil {
		return a.OpenDatabase(clipPath)
	}
	return a.CreateDatabase(clipPath, "Clipbase (Session Scratchpad & Temporary Filter Base)")
}

// GetDatabaseRolesStatus returns the current relationship state of Active, Default, Reference, and Clipbase databases.
func (a *App) GetDatabaseRolesStatus() (*DatabaseRolesStatus, error) {
	prefs, _ := a.GetAppPreferences()
	if prefs.ReferenceDatabasePath != "" {
		_ = a.openReferenceDatabaseInternal(prefs.ReferenceDatabasePath)
	}

	a.mu.RLock()
	var activeInfo *DatabaseInfo
	activeClean := ""
	if a.db != nil && a.db.IsOpen {
		activeClean = normalizeScidBasePath(a.db.BasePath)
		activeInfo = &DatabaseInfo{
			BasePath:    a.db.BasePath,
			Name:        filepath.Base(a.db.BasePath),
			NumGames:    int(a.db.Header.NumGames),
			Description: a.db.Header.Description,
		}
	}
	var refInfo *DatabaseInfo
	if a.refDB != nil && a.refDB.IsOpen {
		refInfo = &DatabaseInfo{
			BasePath:    a.refDB.BasePath,
			Name:        filepath.Base(a.refDB.BasePath),
			NumGames:    int(a.refDB.Header.NumGames),
			Description: a.refDB.Header.Description,
		}
	}
	a.mu.RUnlock()

	defClean := normalizeScidBasePath(prefs.DefaultDatabasePath)
	defName := ""
	if defClean != "" {
		defName = filepath.Base(defClean)
	}

	refClean := normalizeScidBasePath(prefs.ReferenceDatabasePath)
	refName := ""
	if refClean != "" {
		refName = filepath.Base(refClean)
	}

	clipPath := clipbaseDefaultPath()
	clipClean := normalizeScidBasePath(clipPath)

	return &DatabaseRolesStatus{
		ActiveDB:        activeInfo,
		DefaultDBPath:   defClean,
		DefaultDBName:   defName,
		IsActiveDefault: activeClean != "" && defClean != "" && activeClean == defClean,
		ReferenceDB:     refInfo,
		ReferenceDBPath: refClean,
		ReferenceDBName: refName,
		UseRefForTree:   prefs.UseReferenceDBForTree,
		ClipbasePath:    clipClean,
		IsClipbase:      activeClean != "" && activeClean == clipClean,
	}, nil
}

// GetWindowStartupConfig returns the startup mode for this OS window instance.
func (a *App) GetWindowStartupConfig() WindowStartupConfig {
	return windowStartupCfg
}

// LaunchDetachedPanelWindow spawns a secondary native NewScid OS window dedicated to a specific panel
// so tiling compositors like Hyprland can place different portions of activity on separate monitors/workspaces.
func (a *App) LaunchDetachedPanelWindow(panelID string, title string, currentFEN string, dbPath string) error {
	exe, err := os.Executable()
	if err != nil || exe == "" {
		exe = "newscid"
	}
	if strings.TrimSpace(panelID) == "" {
		panelID = "tree"
	}
	if strings.TrimSpace(title) == "" {
		title = "NewScid Panel — " + strings.ToUpper(panelID)
	}

	args := []string{
		"--window-panel", panelID,
		"--title", title,
	}
	if strings.TrimSpace(currentFEN) != "" {
		args = append(args, "--fen", currentFEN)
	}
	if strings.TrimSpace(dbPath) != "" {
		args = append(args, "--db", dbPath)
	}

	cmd := exec.Command(exe, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Start()
}

// LaunchPortalWindow opens a dedicated window right into Lichess.org or Chess.com (suitable for Hyprland multi-screen).
func (a *App) LaunchPortalWindow(platform string, targetURL string, mode string) error {
	plat := strings.ToLower(strings.TrimSpace(platform))
	url := strings.TrimSpace(targetURL)
	if url == "" {
		if plat == "chesscom" || plat == "chess.com" {
			url = "https://www.chess.com/play/online"
		} else {
			url = "https://lichess.org"
		}
	}

	prefs, _ := a.GetAppPreferences()
	if mode == "" {
		mode = prefs.PortalWindowMode
	}

	title := "NewScid Live Portal — Lichess.org"
	if strings.Contains(strings.ToLower(url), "chess.com") || plat == "chesscom" {
		title = "NewScid Live Portal — Chess.com"
	}

	if mode == "native-window" {
		exe, err := os.Executable()
		if err == nil && exe != "" {
			cmd := exec.Command(exe, "--portal-url", url, "--title", title)
			if err := cmd.Start(); err == nil {
				return nil
			}
		}
	}

	// Check custom browser command first
	if prefs.CustomBrowserCmd != "" {
		parts := strings.Fields(prefs.CustomBrowserCmd)
		if len(parts) > 0 {
			args := append(parts[1:], url)
			cmd := exec.Command(parts[0], args...)
			if err := cmd.Start(); err == nil {
				return nil
			}
		}
	}

	// Prefer chromeless standalone --app=<url> window for Hyprland tiling if a Chromium-based browser is installed
	appBrowsers := []string{
		"chromium",
		"brave",
		"brave-browser",
		"google-chrome-stable",
		"google-chrome",
		"vivaldi",
		"microsoft-edge",
	}
	for _, bin := range appBrowsers {
		if path, err := exec.LookPath(bin); err == nil && path != "" {
			cmd := exec.Command(path, "--app="+url, "--class=newscid-portal-"+plat)
			if err := cmd.Start(); err == nil {
				return nil
			}
		}
	}

	// Try firefox --new-window
	if path, err := exec.LookPath("firefox"); err == nil && path != "" {
		cmd := exec.Command(path, "--new-window", url)
		if err := cmd.Start(); err == nil {
			return nil
		}
	}

	// Fallback to native NewScid portal window or xdg-open
	if exe, err := os.Executable(); err == nil && exe != "" {
		cmd := exec.Command(exe, "--portal-url", url, "--title", title)
		if err := cmd.Start(); err == nil {
			return nil
		}
	}
	if path, err := exec.LookPath("xdg-open"); err == nil && path != "" {
		return exec.Command(path, url).Start()
	}
	return fmt.Errorf("no suitable browser or window launcher found for %s", url)
}

// PublishWorkspaceState writes the current board/database state to ~/.newscid/workspace_sync.json
// so detached Hyprland screen windows stay synchronized in real time.
func (a *App) PublishWorkspaceState(state SharedWorkspaceState) error {
	workspaceSyncMu.Lock()
	defer workspaceSyncMu.Unlock()

	if state.UpdatedAtMs == 0 {
		state.UpdatedAtMs = time.Now().UnixMilli()
	}
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return os.WriteFile(workspaceSyncFilePath(), data, 0o644)
}

// GetSharedWorkspaceState reads the shared workspace state for multi-screen synchronization.
func (a *App) GetSharedWorkspaceState() (*SharedWorkspaceState, error) {
	workspaceSyncMu.Lock()
	defer workspaceSyncMu.Unlock()

	data, err := os.ReadFile(workspaceSyncFilePath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var state SharedWorkspaceState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}
