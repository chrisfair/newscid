package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"newscid/pkg/blundervault"
	"newscid/pkg/database/scid"
	"newscid/pkg/online"
	"newscid/pkg/persona"
	"newscid/pkg/playerbot"
	"newscid/pkg/polyglot"
	"newscid/pkg/repertoire"
	"newscid/pkg/training"
	"newscid/pkg/workstation"
)

// SavedFloaterRect stores floating panel coordinates in user config.
type SavedFloaterRect struct {
	X         int  `json:"x"`
	Y         int  `json:"y"`
	Width     int  `json:"width"`
	Height    int  `json:"height"`
	Minimized bool `json:"minimized"`
	ZIndex    int  `json:"zIndex"`
}

// SavedPanelConfig stores a panel's dock/float/detach state and assigned workspace in user config.
type SavedPanelConfig struct {
	Mode            string           `json:"mode"`            // "docked" | "floater" | "detached" | "hidden"
	DockZone        string           `json:"dockZone"`        // "left" | "right" | "bottom"
	TargetWorkspace string           `json:"targetWorkspace"` // "" | "1" .. "10"
	FloaterRect     SavedFloaterRect `json:"floaterRect"`
}

// AppPreferences stores persistent user preferences in ~/.config/newscid/preferences.json (and ~/.newscid/preferences.json).
type AppPreferences struct {
	DefaultDatabasePath         string                      `json:"defaultDatabasePath"`
	ReferenceDatabasePath       string                      `json:"referenceDatabasePath"`
	AutoOpenDefaultDB           bool                        `json:"autoOpenDefaultDb"`
	UseReferenceDBForTree       bool                        `json:"useReferenceDbForTree"`
	DefaultPieceSet             string                      `json:"defaultPieceSet"`
	DefaultBoardTheme           string                      `json:"defaultBoardTheme"`
	DefaultUITheme              string                      `json:"defaultUITheme"`
	DefaultNotationStyle        string                      `json:"defaultNotationStyle"`
	SoundEnabled                bool                        `json:"soundEnabled"`
	ShowEvalBar                 bool                        `json:"showEvalBar"`
	ThreatModeDefault           bool                        `json:"threatModeDefault"`
	BlindfoldDefault            bool                        `json:"blindfoldDefault"`
	BoardFlippedDefault         bool                        `json:"boardFlippedDefault"`
	CompactLayoutMode           string                      `json:"compactLayoutMode"` // "auto" | "compact" | "expanded"
	LeftColWidth                int                         `json:"leftColWidth"`
	BottomDrawerHeight          int                         `json:"bottomDrawerHeight"`
	BottomDrawerOpen            bool                        `json:"bottomDrawerOpen"`
	MainWindowWorkspace         string                      `json:"mainWindowWorkspace"`
	AutoRestoreWorkspaceWindows bool                        `json:"autoRestoreWorkspaceWindows"`
	PanelConfigs                map[string]SavedPanelConfig `json:"panelConfigs"`
	DefaultEngine               string                      `json:"defaultEngine"`
	DefaultMultiPV              int                         `json:"defaultMultiPv"`
	DefaultThreads              int                         `json:"defaultThreads"`
	DefaultHashMB               int                         `json:"defaultHashMb"`
	AnnotateMoveTimeMs          int                         `json:"annotateMoveTimeMs"`
	LichessUsername             string                      `json:"lichessUsername"`
	ChessComUsername            string                      `json:"chessComUsername"`
	DefaultPortalPlatform       string                      `json:"defaultPortalPlatform"` // "lichess" | "chesscom"
	PortalWindowMode            string                      `json:"portalWindowMode"`      // "browser-app" | "native-window" | "floater" | "docked"
	CustomBrowserCmd            string                      `json:"customBrowserCmd"`
	DatabasePolicies            map[string]string           `json:"databasePolicies,omitempty"`
	CustomNotablePlayers        []scid.FamousPlayerEntry    `json:"customNotablePlayers,omitempty"`
	HiddenFamousPlayers         []string                    `json:"hiddenFamousPlayers,omitempty"`
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
	PanelID     string `json:"panelId"`     // e.g. "board", "tree", "engine", "notation", "games", "imbalance", "info", "portal", "db-roles"
	Workspace   string `json:"workspace"`   // e.g. "1" .. "10"
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
	windowStartupCfg  = WindowStartupConfig{Mode: "main"}
	prefsMu           sync.Mutex
	workspaceSyncMu   sync.Mutex
	detachedPanelPIDs = make(map[string]int)
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

func xdgPreferencesFilePath() string {
	cfgDir, err := os.UserConfigDir()
	if err != nil || cfgDir == "" {
		home, _ := os.UserHomeDir()
		if home == "" {
			return preferencesFilePath()
		}
		cfgDir = filepath.Join(home, ".config")
	}
	dir := filepath.Join(cfgDir, "newscid")
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
		DefaultDatabasePath:         "",
		ReferenceDatabasePath:       "",
		AutoOpenDefaultDB:           true,
		UseReferenceDBForTree:       true,
		DefaultPieceSet:             "cburnett",
		DefaultBoardTheme:           "classic-wood",
		DefaultUITheme:              "dark",
		DefaultNotationStyle:        "san",
		SoundEnabled:                true,
		ShowEvalBar:                 true,
		ThreatModeDefault:           false,
		BlindfoldDefault:            false,
		BoardFlippedDefault:         false,
		CompactLayoutMode:           "auto",
		LeftColWidth:                0,
		BottomDrawerHeight:          210,
		BottomDrawerOpen:            true,
		MainWindowWorkspace:         "",
		AutoRestoreWorkspaceWindows: false,
		PanelConfigs:                make(map[string]SavedPanelConfig),
		DefaultEngine:               "",
		DefaultMultiPV:              1,
		DefaultThreads:              2,
		DefaultHashMB:               64,
		AnnotateMoveTimeMs:          150,
		LichessUsername:             "",
		ChessComUsername:            "",
		DefaultPortalPlatform:       "lichess",
		PortalWindowMode:            "browser-app",
		CustomBrowserCmd:            "",
		DatabasePolicies:            make(map[string]string),
	}
}

// GetAppPreferences loads the user's preferences from ~/.config/newscid/preferences.json (or ~/.newscid/preferences.json).
func (a *App) GetAppPreferences() (*AppPreferences, error) {
	prefsMu.Lock()
	defer prefsMu.Unlock()

	prefs := defaultAppPreferences()
	xdgPath := xdgPreferencesFilePath()
	legacyPath := preferencesFilePath()

	data, err := os.ReadFile(xdgPath)
	if err != nil {
		data, err = os.ReadFile(legacyPath)
		if err != nil {
			return prefs, nil
		}
	}
	_ = json.Unmarshal(data, prefs)
	if prefs.PanelConfigs == nil {
		prefs.PanelConfigs = make(map[string]SavedPanelConfig)
	}
	if prefs.DatabasePolicies == nil {
		prefs.DatabasePolicies = make(map[string]string)
	}
	if prefs.CompactLayoutMode == "" {
		prefs.CompactLayoutMode = "auto"
	}
	if prefs.BottomDrawerHeight <= 0 {
		prefs.BottomDrawerHeight = 210
	}
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

// SaveAppPreferences persists the user's preferences to ~/.config/newscid/preferences.json and ~/.newscid/preferences.json.
func (a *App) SaveAppPreferences(prefs AppPreferences) (*AppPreferences, error) {
	prefsMu.Lock()
	prefs.DefaultDatabasePath = normalizeScidBasePath(prefs.DefaultDatabasePath)
	prefs.ReferenceDatabasePath = normalizeScidBasePath(prefs.ReferenceDatabasePath)
	if prefs.PanelConfigs == nil {
		prefs.PanelConfigs = make(map[string]SavedPanelConfig)
	}
	if prefs.DatabasePolicies == nil {
		prefs.DatabasePolicies = make(map[string]string)
	}
	data, err := json.MarshalIndent(prefs, "", "  ")
	if err != nil {
		prefsMu.Unlock()
		return nil, err
	}
	xdgPath := xdgPreferencesFilePath()
	legacyPath := preferencesFilePath()
	if err := os.WriteFile(xdgPath, data, 0o644); err != nil {
		prefsMu.Unlock()
		return nil, err
	}
	_ = os.WriteFile(legacyPath, data, 0o644)
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

func (a *App) resolveDatabaseWritePolicy(basePath string, numGames int, desc string) scid.WritePolicyID {
	clean := normalizeScidBasePath(basePath)
	if clean == "" {
		return scid.WritePolicyEditable
	}
	prefs, _ := a.GetAppPreferences()
	if prefs != nil && prefs.DatabasePolicies != nil {
		if pol, ok := prefs.DatabasePolicies[clean]; ok && strings.TrimSpace(pol) != "" {
			return scid.ParseWritePolicy(pol)
		}
	}
	isRef := prefs != nil && normalizeScidBasePath(prefs.ReferenceDatabasePath) == clean && clean != ""
	return scid.InferDefaultWritePolicy(clean, numGames, desc, isRef)
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
		numG := int(a.db.Header.NumGames)
		desc := a.db.Header.Description
		pol := a.resolveDatabaseWritePolicy(activeClean, numG, desc)
		activeInfo = &DatabaseInfo{
			BasePath:    a.db.BasePath,
			Name:        filepath.Base(a.db.BasePath),
			NumGames:    numG,
			Description: desc,
			WritePolicy: pol.String(),
			IsLocked:    !pol.AllowsDirectEdit(),
		}
	}
	var refInfo *DatabaseInfo
	if a.refDB != nil && a.refDB.IsOpen {
		numG := int(a.refDB.Header.NumGames)
		desc := a.refDB.Header.Description
		pol := a.resolveDatabaseWritePolicy(a.refDB.BasePath, numG, desc)
		refInfo = &DatabaseInfo{
			BasePath:    a.refDB.BasePath,
			Name:        filepath.Base(a.refDB.BasePath),
			NumGames:    numG,
			Description: desc,
			WritePolicy: pol.String(),
			IsLocked:    !pol.AllowsDirectEdit(),
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
	prefs, _ := a.GetAppPreferences()
	ws := ""
	if cfg, ok := prefs.PanelConfigs[panelID]; ok {
		ws = cfg.TargetWorkspace
	}
	return a.LaunchPanelOnWorkspace(panelID, ws, title, currentFEN, dbPath)
}

// LaunchPanelOnWorkspace spawns or relocates a detached panel window onto a specific Hyprland workspace (1..10)
// and persists the workspace assignment in the user's configuration.
func (a *App) LaunchPanelOnWorkspace(panelID string, workspace string, title string, currentFEN string, dbPath string) error {
	panelID = strings.TrimSpace(panelID)
	if panelID == "" {
		panelID = "tree"
	}
	workspace = strings.TrimSpace(workspace)

	// Persist panel workspace & detached mode in user config immediately
	prefs, _ := a.GetAppPreferences()
	if prefs.PanelConfigs == nil {
		prefs.PanelConfigs = make(map[string]SavedPanelConfig)
	}
	cfg := prefs.PanelConfigs[panelID]
	cfg.Mode = "detached"
	cfg.TargetWorkspace = workspace
	prefs.PanelConfigs[panelID] = cfg
	_, _ = a.SaveAppPreferences(*prefs)

	// Check if a detached window for this panel is already alive
	prefsMu.Lock()
	existingPID := detachedPanelPIDs[panelID]
	prefsMu.Unlock()

	if existingPID > 0 && syscall.Kill(existingPID, 0) == nil {
		if workspace != "" {
			dispatchHyprlandWorkspace(existingPID, workspace, false)
		}
		return nil
	}

	exe, err := os.Executable()
	if err != nil || exe == "" {
		exe = "newscid"
	}
	if strings.TrimSpace(title) == "" {
		title = "NewScid Panel — " + strings.ToUpper(panelID)
	}

	args := []string{
		"--window-panel", panelID,
		"--title", title,
	}
	if workspace != "" {
		args = append(args, "--workspace", workspace)
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
	if err := cmd.Start(); err != nil {
		return err
	}

	if cmd.Process != nil {
		pid := cmd.Process.Pid
		prefsMu.Lock()
		detachedPanelPIDs[panelID] = pid
		prefsMu.Unlock()
		if workspace != "" {
			go func(targetPID int, ws string) {
				time.Sleep(400 * time.Millisecond)
				dispatchHyprlandWorkspace(targetPID, ws, true)
				time.Sleep(550 * time.Millisecond)
				dispatchHyprlandWorkspace(targetPID, ws, true)
			}(pid, workspace)
		}
	}
	return nil
}

// MoveCurrentWindowToWorkspace moves the current OS window to the specified Hyprland workspace (1..10).
func (a *App) MoveCurrentWindowToWorkspace(workspace string) error {
	workspace = strings.TrimSpace(workspace)
	if workspace == "" {
		return nil
	}
	if windowStartupCfg.Mode == "panel" && windowStartupCfg.PanelID != "" {
		prefs, _ := a.GetAppPreferences()
		if prefs.PanelConfigs == nil {
			prefs.PanelConfigs = make(map[string]SavedPanelConfig)
		}
		cfg := prefs.PanelConfigs[windowStartupCfg.PanelID]
		cfg.TargetWorkspace = workspace
		prefs.PanelConfigs[windowStartupCfg.PanelID] = cfg
		_, _ = a.SaveAppPreferences(*prefs)
	} else if windowStartupCfg.Mode == "main" {
		prefs, _ := a.GetAppPreferences()
		prefs.MainWindowWorkspace = workspace
		_, _ = a.SaveAppPreferences(*prefs)
	}
	dispatchHyprlandWorkspace(os.Getpid(), workspace, false)
	return nil
}

func dispatchHyprlandWorkspace(pid int, workspace string, silent bool) {
	if pid <= 0 || workspace == "" {
		return
	}
	hyprctl, err := exec.LookPath("hyprctl")
	if err != nil || hyprctl == "" {
		return
	}
	action := "movetoworkspace"
	if silent {
		action = "movetoworkspacesilent"
	}
	target := fmt.Sprintf("%s,pid:%d", workspace, pid)
	_ = exec.Command(hyprctl, "dispatch", action, target).Run()
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

// UpdateDatabaseFromLocal updates any target Scid database (even multi-million-game Reference/Mega databases,
// Default DB, or Active DB) with all non-duplicate games from local .si4 / .pgn databases currently on the machine.
func (a *App) UpdateDatabaseFromLocal(targetBasePath string, sourceBasePaths []string) (*scid.MergeStats, error) {
	targetClean := normalizeScidBasePath(targetBasePath)

	a.mu.RLock()
	activeClean := ""
	if a.db != nil && a.db.IsOpen {
		activeClean = normalizeScidBasePath(a.db.BasePath)
	}
	refClean := ""
	if a.refDB != nil && a.refDB.IsOpen {
		refClean = normalizeScidBasePath(a.refDB.BasePath)
	}
	a.mu.RUnlock()

	if targetClean == "" {
		if activeClean != "" {
			targetClean = activeClean
		} else {
			prefs, _ := a.GetAppPreferences()
			targetClean = normalizeScidBasePath(prefs.DefaultDatabasePath)
		}
	}
	if targetClean == "" {
		return nil, fmt.Errorf("no target database selected or open")
	}

	targetPolicy := a.resolveDatabaseWritePolicy(targetClean, 0, "")
	if !targetPolicy.AllowsOnlineSync() {
		return nil, fmt.Errorf("database %q is strictly locked (%s); unlock it or set it to Sync-Only first", filepath.Base(targetClean), targetPolicy.String())
	}

	if len(sourceBasePaths) == 0 {
		discovered := scid.DiscoverScidDatabases()
		for _, d := range discovered {
			if d.Category == "Tutorial & Tactics" {
				continue
			}
			srcClean := normalizeScidBasePath(d.BasePath)
			if srcClean == "" || srcClean == targetClean {
				continue
			}
			sourceBasePaths = append(sourceBasePaths, d.BasePath)
		}
	}

	progressCb := func(sourceName string, processed int, added int, skipped int) {
		if a.ctx != nil {
			runtime.EventsEmit(a.ctx, "db:merge-progress", map[string]interface{}{
				"source":    sourceName,
				"processed": processed,
				"added":     added,
				"skipped":   skipped,
			})
		}
	}

	if targetClean == activeClean {
		a.mu.Lock()
		defer a.mu.Unlock()
		if a.db == nil || !a.db.IsOpen {
			return nil, fmt.Errorf("active database is not open")
		}
		return a.db.MergeFromDatabases(sourceBasePaths, progressCb)
	}

	if targetClean == refClean {
		a.mu.Lock()
		defer a.mu.Unlock()
		if a.refDB == nil || !a.refDB.IsOpen {
			return nil, fmt.Errorf("reference database is not open")
		}
		return a.refDB.MergeFromDatabases(sourceBasePaths, progressCb)
	}

	targetDB := scid.NewDatabase()
	if err := targetDB.Open(targetClean); err != nil {
		return nil, fmt.Errorf("failed to open target database %s: %w", filepath.Base(targetClean), err)
	}
	defer targetDB.Close()

	return targetDB.MergeFromDatabases(sourceBasePaths, progressCb)
}

// ==========================================
// Database Locking & Write Policies
// ==========================================

// GetDatabaseWritePolicy returns the current write policy ("editable", "sync_only", or "locked") for basePath.
func (a *App) GetDatabaseWritePolicy(basePath string) string {
	clean := normalizeScidBasePath(basePath)
	if clean == "" {
		a.mu.RLock()
		if a.db != nil && a.db.IsOpen {
			clean = normalizeScidBasePath(a.db.BasePath)
			numG := a.db.NumGames()
			desc := ""
			if a.db.Header != nil {
				desc = a.db.Header.Description
			}
			a.mu.RUnlock()
			return a.resolveDatabaseWritePolicy(clean, numG, desc).String()
		}
		a.mu.RUnlock()
		return scid.WritePolicyEditable.String()
	}
	meta := scid.InspectDatabase(clean, "")
	return a.resolveDatabaseWritePolicy(clean, meta.NumGames, meta.Description).String()
}

// SetDatabaseWritePolicy sets and persists the write policy ("editable", "sync_only", or "locked") for basePath.
func (a *App) SetDatabaseWritePolicy(basePath string, policy string) (*DatabaseInfo, error) {
	clean := normalizeScidBasePath(basePath)
	if clean == "" {
		a.mu.RLock()
		if a.db != nil && a.db.IsOpen {
			clean = normalizeScidBasePath(a.db.BasePath)
		}
		a.mu.RUnlock()
	}
	if clean == "" {
		return nil, fmt.Errorf("no database path specified")
	}

	parsed := scid.ParseWritePolicy(policy)
	prefs, _ := a.GetAppPreferences()
	if prefs.DatabasePolicies == nil {
		prefs.DatabasePolicies = make(map[string]string)
	}
	prefs.DatabasePolicies[clean] = parsed.String()
	if _, err := a.SaveAppPreferences(*prefs); err != nil {
		return nil, err
	}

	return a.GetDatabaseInfo()
}

// ==========================================
// Active Long-Running Operation Cancellation
// ==========================================

func (a *App) startCancellableOperation(name string) (context.Context, func()) {
	a.opMu.Lock()
	if a.activeOpCancel != nil {
		a.activeOpCancel()
	}
	baseCtx := a.ctx
	if baseCtx == nil {
		baseCtx = context.Background()
	}
	ctx, cancel := context.WithCancel(baseCtx)
	a.activeOpCancel = cancel
	a.activeOpName = name
	a.opMu.Unlock()

	a.emitEvent("operation:started", map[string]interface{}{
		"operation": name,
	})

	finish := func() {
		a.opMu.Lock()
		cancel()
		if a.activeOpName == name {
			a.activeOpCancel = nil
			a.activeOpName = ""
		}
		a.opMu.Unlock()
		a.emitEvent("operation:finished", map[string]interface{}{
			"operation": name,
		})
	}
	return ctx, finish
}

// CancelActiveOperation cancels any currently running long action (database import, stream, consolidation, sub-database build, TWIC download, or engine match).
func (a *App) CancelActiveOperation() bool {
	a.opMu.Lock()
	cancel := a.activeOpCancel
	name := a.activeOpName
	a.activeOpCancel = nil
	a.activeOpName = ""
	a.opMu.Unlock()

	if cancel != nil {
		cancel()
		a.emitEvent("operation:cancelled", map[string]interface{}{
			"operation": name,
		})
		return true
	}

	if a.StopEngineMatch() {
		a.emitEvent("operation:cancelled", map[string]interface{}{
			"operation": "engine-match",
		})
		return true
	}
	return false
}

// GetActiveOperation returns the name of the currently executing long-running operation, or "" if idle.
func (a *App) GetActiveOperation() string {
	a.opMu.Lock()
	name := a.activeOpName
	a.opMu.Unlock()
	if name != "" {
		return name
	}
	if a.IsEngineMatchRunning() {
		return "engine-match"
	}
	return ""
}

// ==========================================
// Multi-Database Focused Sub-Database Builder
// ==========================================

// BuildFocusedDatabase extracts games matching opening lines, ECO codes, or players from multiple databases
// into a new focused Scid database in NewScid's managed databases folder.
func (a *App) BuildFocusedDatabase(filter scid.SubDatabaseFilter) (*scid.SubDatabaseResult, error) {
	dataRoot, _ := workstation.DefaultWorkstationDirs()
	outputDir := filepath.Join(dataRoot, "databases")
	_ = os.MkdirAll(outputDir, 0o755)

	if filter.ShouldCancel == nil {
		opCtx, finishOp := a.startCancellableOperation("build-focused-db")
		defer finishOp()
		filter.ShouldCancel = func() bool {
			return opCtx.Err() != nil
		}
	}

	// If no source databases were explicitly passed, use active DB (or all discovered user DBs)
	if len(filter.SourceBasePaths) == 0 {
		a.mu.RLock()
		if a.db != nil && a.db.IsOpen {
			filter.SourceBasePaths = append(filter.SourceBasePaths, a.db.BasePath)
		}
		if a.refDB != nil && a.refDB.IsOpen && a.refDB.BasePath != "" {
			filter.SourceBasePaths = append(filter.SourceBasePaths, a.refDB.BasePath)
		}
		a.mu.RUnlock()
	}

	res, err := scid.BuildFocusedSubDatabase(filter, outputDir, func(source string, scanned, matched int) {
		a.emitEvent("subdb:progress", map[string]interface{}{
			"source":  source,
			"scanned": scanned,
			"matched": matched,
		})
	})
	if err != nil {
		return nil, err
	}

	// Mark the newly created personal sub-database as editable
	if res != nil && res.TargetBasePath != "" {
		_, _ = a.SetDatabaseWritePolicy(res.TargetBasePath, scid.WritePolicyEditable.String())
	}
	return res, nil
}

// ==========================================
// Consolidated Online & Master Database Builder
// ==========================================

// BuildConsolidatedDatabase combines all specified (or all discovered online/local) databases
// into a single unified Scid (.si4) database, removing all duplicate games whether they match
// by Online Game ID/URL or by identical move sequence / game content.
// Optionally sets the resulting database as the Default Reference Database or Default User Database.
func (a *App) BuildConsolidatedDatabase(opts scid.ConsolidateDatabaseOptions) (*scid.ConsolidateDatabaseResult, error) {
	dataRoot, _ := workstation.DefaultWorkstationDirs()
	outputDir := filepath.Join(dataRoot, "databases")
	_ = os.MkdirAll(outputDir, 0o755)

	if opts.ShouldCancel == nil {
		opCtx, finishOp := a.startCancellableOperation("consolidate-databases")
		defer finishOp()
		opts.ShouldCancel = func() bool {
			return opCtx.Err() != nil
		}
	}

	if len(opts.SourceBasePaths) == 0 {
		// Default to only the most recent (2026 / latest) archives when auto-discovering sources
		opts.OnlyMostRecentArchives = true
		seenSrc := make(map[string]bool)
		addSrc := func(p string) {
			clean := normalizeScidBasePath(p)
			if clean == "" || seenSrc[clean] {
				return
			}
			baseLower := strings.ToLower(filepath.Base(clean))
			if baseLower == "clipbase" {
				return
			}
			seenSrc[clean] = true
			opts.SourceBasePaths = append(opts.SourceBasePaths, clean)
		}

		for _, d := range a.GetDiscoveredDatabases() {
			if d.Category == "Tutorial & Tactics" {
				continue
			}
			addSrc(d.BasePath)
		}

		a.mu.RLock()
		if a.db != nil && a.db.IsOpen {
			addSrc(a.db.BasePath)
		}
		if a.refDB != nil && a.refDB.IsOpen {
			addSrc(a.refDB.BasePath)
		}
		a.mu.RUnlock()
	}

	res, err := scid.ConsolidateDatabases(opts, outputDir, func(source string, scanned, added, skippedID, skippedMove int) {
		a.emitEvent("consolidate:progress", map[string]interface{}{
			"source":      source,
			"scanned":     scanned,
			"added":       added,
			"skippedId":   skippedID,
			"skippedMove": skippedMove,
		})
	})
	if err != nil {
		return nil, err
	}

	if res != nil && res.TargetBasePath != "" {
		targetClean := normalizeScidBasePath(res.TargetBasePath)
		// Consolidated reference databases default to Sync-Only to protect against accidental manual edits
		_, _ = a.SetDatabaseWritePolicy(targetClean, scid.WritePolicySyncOnly.String())

		if opts.SetAsReference {
			_, _ = a.SetReferenceDatabase(targetClean, true)
		}
		if opts.SetAsDefaultUser {
			_, _ = a.SetDefaultDatabase(targetClean)
		}

		// Refresh in-memory handles if the consolidated target is currently open as Active or Reference DB
		a.mu.Lock()
		if a.refDB != nil && a.refDB.IsOpen && normalizeScidBasePath(a.refDB.BasePath) == targetClean {
			_ = a.refDB.Close()
			_ = a.refDB.Open(targetClean)
		}
		if a.db != nil && a.db.IsOpen && normalizeScidBasePath(a.db.BasePath) == targetClean {
			_ = a.db.Close()
			_ = a.db.Open(targetClean)
		}
		a.mu.Unlock()
	}

	return res, nil
}

// ConsolidateOnlineCatalogDatabases downloads/streams selected online databases from the catalog
// (defaulting to the 2026 / most recent archives) and consolidates them (along with existing local
// databases if includeLocal is true) into a single deduplicated Consolidated Reference Database.
func (a *App) ConsolidateOnlineCatalogDatabases(
	sourceIDs []string,
	includeLocal bool,
	targetName string,
	maxGamesPerSource int,
	minElo int,
	setAsReference bool,
) (*scid.ConsolidateDatabaseResult, error) {
	dataRoot, _ := workstation.DefaultWorkstationDirs()
	outputDir := filepath.Join(dataRoot, "databases")
	_ = os.MkdirAll(outputDir, 0o755)

	if strings.TrimSpace(targetName) == "" {
		targetName = "Consolidated_Reference"
	}
	if maxGamesPerSource <= 0 {
		maxGamesPerSource = 2500
	}

	var sourcePaths []string
	seen := make(map[string]bool)
	addSource := func(p string) {
		clean := normalizeScidBasePath(p)
		if clean == "" || seen[clean] {
			return
		}
		if strings.EqualFold(filepath.Base(clean), "clipbase") {
			return
		}
		seen[clean] = true
		sourcePaths = append(sourcePaths, clean)
	}

	opCtx, finishOp := a.startCancellableOperation("consolidate-online")
	defer finishOp()

	client := online.NewMegaDatabaseClient()

	for _, id := range sourceIDs {
		if opCtx.Err() != nil {
			break
		}
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		stageName := "online_stage_" + strings.ReplaceAll(strings.ToLower(id), "-", "_")
		stageBase := filepath.Join(outputDir, stageName)
		stageDB := scid.NewDatabase()
		if _, err := os.Stat(stageBase + ".si4"); err == nil {
			_ = stageDB.Open(stageBase)
		} else {
			_ = stageDB.Create(stageBase, "Online Stage: "+id)
		}
		res, err := client.DownloadAndImport(opCtx, online.MegaImportOptions{
			SourceID:          id,
			MaxGames:          maxGamesPerSource,
			MinElo:            minElo,
			CreateNewDatabase: stageName,
		}, stageDB, outputDir, func(prog online.MegaImportProgress) {
			a.emitEvent("megadb:progress", prog)
		})
		_ = stageDB.Close()
		if err == nil && res != nil && res.DatabasePath != "" {
			addSource(res.DatabasePath)
		} else if _, statErr := os.Stat(stageBase + ".si4"); statErr == nil {
			addSource(stageBase)
		}
	}

	if includeLocal || len(sourcePaths) == 0 {
		for _, d := range a.GetDiscoveredDatabases() {
			if d.Category == "Tutorial & Tactics" {
				continue
			}
			addSource(d.BasePath)
		}
		a.mu.RLock()
		if a.db != nil && a.db.IsOpen {
			addSource(a.db.BasePath)
		}
		if a.refDB != nil && a.refDB.IsOpen {
			addSource(a.refDB.BasePath)
		}
		a.mu.RUnlock()
	}

	return a.BuildConsolidatedDatabase(scid.ConsolidateDatabaseOptions{
		TargetName:             targetName,
		TargetDescription:      "Consolidated Online & Master Reference Database (Deduplicated by ID & Move Sequence)",
		SourceBasePaths:        sourcePaths,
		MinElo:                 minElo,
		SetAsReference:         setAsReference,
		OnlyMostRecentArchives: true,
		ShouldCancel: func() bool {
			return opCtx.Err() != nil
		},
	})
}

// ==========================================
// Famous & Custom Tracked Players Discovery
// ==========================================

// GetNotablePlayersDropdown returns famous grandmasters and user-configured players discovered in the active database.
func (a *App) GetNotablePlayersDropdown() ([]scid.DiscoveredNotablePlayer, error) {
	prefs, _ := a.GetAppPreferences()
	var custom []scid.FamousPlayerEntry
	var hidden []string
	if prefs != nil {
		custom = prefs.CustomNotablePlayers
		hidden = prefs.HiddenFamousPlayers
	}

	a.mu.RLock()
	db := a.db
	a.mu.RUnlock()

	return scid.DiscoverNotablePlayersInDatabase(db, custom, hidden), nil
}

// AddCustomNotablePlayer adds a user-selected player of interest to the Notable Players dropdown.
func (a *App) AddCustomNotablePlayer(name string, aliases []string, title string, era string) ([]scid.DiscoveredNotablePlayer, error) {
	cleanName := strings.TrimSpace(name)
	if cleanName == "" {
		return nil, fmt.Errorf("player name is required")
	}

	prefs, _ := a.GetAppPreferences()
	// Remove from hidden list if present
	var newHidden []string
	for _, h := range prefs.HiddenFamousPlayers {
		if !strings.EqualFold(strings.TrimSpace(h), cleanName) {
			newHidden = append(newHidden, h)
		}
	}
	prefs.HiddenFamousPlayers = newHidden

	// Update or append custom entry
	var cleanAliases []string
	for _, al := range aliases {
		if t := strings.TrimSpace(al); t != "" {
			cleanAliases = append(cleanAliases, t)
		}
	}
	if len(cleanAliases) == 0 {
		cleanAliases = []string{cleanName}
	}

	entry := scid.FamousPlayerEntry{
		Name:     cleanName,
		Title:    strings.TrimSpace(title),
		Era:      strings.TrimSpace(era),
		Aliases:  cleanAliases,
		IsCustom: true,
	}
	if entry.Title == "" {
		entry.Title = "Tracked Player"
	}

	updated := false
	for i, existing := range prefs.CustomNotablePlayers {
		if strings.EqualFold(existing.Name, cleanName) {
			prefs.CustomNotablePlayers[i] = entry
			updated = true
			break
		}
	}
	if !updated {
		prefs.CustomNotablePlayers = append(prefs.CustomNotablePlayers, entry)
	}

	if _, err := a.SaveAppPreferences(*prefs); err != nil {
		return nil, err
	}
	return a.GetNotablePlayersDropdown()
}

// RemoveNotablePlayer removes a custom player or hides a built-in famous player from the dropdown.
func (a *App) RemoveNotablePlayer(name string) ([]scid.DiscoveredNotablePlayer, error) {
	cleanName := strings.TrimSpace(name)
	if cleanName == "" {
		return a.GetNotablePlayersDropdown()
	}

	prefs, _ := a.GetAppPreferences()
	var remaining []scid.FamousPlayerEntry
	removedCustom := false
	for _, c := range prefs.CustomNotablePlayers {
		if strings.EqualFold(strings.TrimSpace(c.Name), cleanName) {
			removedCustom = true
			continue
		}
		remaining = append(remaining, c)
	}
	prefs.CustomNotablePlayers = remaining

	if !removedCustom {
		alreadyHidden := false
		for _, h := range prefs.HiddenFamousPlayers {
			if strings.EqualFold(strings.TrimSpace(h), cleanName) {
				alreadyHidden = true
				break
			}
		}
		if !alreadyHidden {
			prefs.HiddenFamousPlayers = append(prefs.HiddenFamousPlayers, cleanName)
		}
	}

	if _, err := a.SaveAppPreferences(*prefs); err != nil {
		return nil, err
	}
	return a.GetNotablePlayersDropdown()
}

// ResetNotablePlayersToDefault restores the default Famous Players list.
func (a *App) ResetNotablePlayersToDefault() ([]scid.DiscoveredNotablePlayer, error) {
	prefs, _ := a.GetAppPreferences()
	prefs.CustomNotablePlayers = nil
	prefs.HiddenFamousPlayers = nil
	if _, err := a.SaveAppPreferences(*prefs); err != nil {
		return nil, err
	}
	return a.GetNotablePlayersDropdown()
}

// ==========================================
// Workstation Storage, Asset Copying & Factory Reset
// ==========================================

// GetWorkstationStorageReport returns the exact locations and contents of NewScid's managed directories
// (databases, personas, engines, opening books, endgame tablebases, repertoires, config, and backups).
func (a *App) GetWorkstationStorageReport() (*workstation.StorageReport, error) {
	dataRoot, configRoot := workstation.DefaultWorkstationDirs()
	return workstation.InspectStorage(dataRoot, configRoot)
}

// ImportAssetToWorkstation copies an external database (.si4/.pgn), UCI engine binary, Polyglot opening book (.bin),
// or endgame tablebase (.rtbw/.rtbz) into NewScid's managed directory.
func (a *App) ImportAssetToWorkstation(category string, sourcePath string) (*workstation.StoredAssetInfo, error) {
	dataRoot, configRoot := workstation.DefaultWorkstationDirs()
	return workstation.ImportAssetToWorkstation(category, sourcePath, dataRoot, configRoot)
}

// SelectAndImportWorkstationAsset opens a native file picker and copies the selected asset into NewScid's storage.
func (a *App) SelectAndImportWorkstationAsset(category string) (*workstation.StoredAssetInfo, error) {
	if a.ctx == nil {
		return nil, fmt.Errorf("file dialog requires desktop GUI runtime")
	}
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Asset to Copy into NewScid Workstation Storage",
	})
	if err != nil || strings.TrimSpace(path) == "" {
		return nil, err
	}
	return a.ImportAssetToWorkstation(category, path)
}

// FactoryResetNewScid completely resets all user databases, personas, repertoires, training vaults,
// and saved configurations back to clean FACTORY SETTINGS, with an opt-in .tar.gz backup before wiping.
func (a *App) FactoryResetNewScid(opts workstation.FactoryResetOptions) (*workstation.FactoryResetResult, error) {
	// Close any open databases and stop running engines first
	a.mu.Lock()
	if a.engine != nil {
		_ = a.engine.Quit()
		a.engine = nil
	}
	if a.db != nil && a.db.IsOpen {
		_ = a.db.Close()
	}
	if a.refDB != nil && a.refDB.IsOpen {
		_ = a.refDB.Close()
		a.refDB = nil
	}
	a.db = scid.NewDatabase()
	a.activeGame = nil
	a.personaBotCache = make(map[string]*playerbot.PlayerPersonality)
	a.polyglotBooks = make(map[string]*polyglot.Book)
	a.solitaireSession = nil
	a.mu.Unlock()

	dataRoot, configRoot := workstation.DefaultWorkstationDirs()
	legacyDir := workstation.DefaultLegacyConfigDir()
	res, err := workstation.FactoryResetWorkstation(opts, dataRoot, configRoot, legacyDir)
	if err != nil {
		return nil, err
	}

	// Reinitialize clean managers pointing to the fresh empty config directory
	a.mu.Lock()
	a.profileMgr, _ = online.NewProfileManager("")
	a.syncMgr = online.NewSyncManager(a.profileMgr)
	a.personaMgr, _ = persona.NewPersonaManager("")
	a.repMgr = repertoire.NewManager("")
	a.blunderVault, _ = blundervault.NewVault(filepath.Join(configRoot, "blunder_vault.json"))
	a.cyclicalMgr, _ = training.NewCyclicalManager(filepath.Join(configRoot, "cyclical_drill.json"))
	a.mu.Unlock()

	return res, nil
}
