package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"newscid/pkg/blundervault"
	"newscid/pkg/chess"
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
	AutoCapturePortalGames      bool                        `json:"autoCapturePortalGames"`
	AutoLoadCapturedGame        bool                        `json:"autoLoadCapturedGame"`
	DefaultPortalPlatform       string                      `json:"defaultPortalPlatform"` // "lichess" | "chesscom"
	PortalWindowMode            string                      `json:"portalWindowMode"`      // "browser-app" | "native-window" | "floater" | "docked"
	CustomBrowserCmd            string                      `json:"customBrowserCmd"`
	DatabasePolicies            map[string]string           `json:"databasePolicies,omitempty"`
	CustomNotablePlayers        []scid.FamousPlayerEntry    `json:"customNotablePlayers,omitempty"`
	HiddenFamousPlayers         []string                    `json:"hiddenFamousPlayers,omitempty"`
	PlayerDisplayName           string                      `json:"playerDisplayName"`
	PlayerPersonaID             string                      `json:"playerPersonaId"`
	PlayerTitle                 string                      `json:"playerTitle"`
	PlayerEstimatedElo          int                         `json:"playerEstimatedElo"`
	PlayerFideID                string                      `json:"playerFideId"`
	PlayerAliases               []string                    `json:"playerAliases,omitempty"`
	FocusPolyglotBookPath       string                      `json:"focusPolyglotBookPath"`
	FocusWhiteRepertoireID      string                      `json:"focusWhiteRepertoireId"`
	FocusBlackRepertoireID      string                      `json:"focusBlackRepertoireId"`
	OnboardingCompleted         bool                        `json:"onboardingCompleted"`
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
		AutoCapturePortalGames:      true,
		AutoLoadCapturedGame:        true,
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
	if prefs == nil || prefs.AutoCapturePortalGames {
		_, _ = a.StartLivePortalWatcher()
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

	ctx, finish := a.startCancellableOperation("local-merge")
	defer finish()
	cancelCheck := func() bool {
		select {
		case <-ctx.Done():
			return true
		default:
			return false
		}
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
			// Automatically exclude massive reference/archive databases (>100k games)
			// and databases locked/sync_only to prevent accidentally freezing the UI or
			// sucking multi-gigabyte master archives into personal databases unless explicitly chosen by path.
			if d.NumGames > 100000 {
				continue
			}
			policy := a.resolveDatabaseWritePolicy(srcClean, d.NumGames, d.Description)
			if policy == scid.WritePolicyLocked || policy == scid.WritePolicySyncOnly {
				continue
			}
			sourceBasePaths = append(sourceBasePaths, d.BasePath)
		}
	}

	if len(sourceBasePaths) == 0 {
		return &scid.MergeStats{
			TargetTotalGames: 0,
		}, nil
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
		return a.db.MergeFromDatabasesWithCancel(sourceBasePaths, progressCb, cancelCheck)
	}

	if targetClean == refClean {
		a.mu.Lock()
		defer a.mu.Unlock()
		if a.refDB == nil || !a.refDB.IsOpen {
			return nil, fmt.Errorf("reference database is not open")
		}
		return a.refDB.MergeFromDatabasesWithCancel(sourceBasePaths, progressCb, cancelCheck)
	}

	targetDB := scid.NewDatabase()
	if err := targetDB.Open(targetClean); err != nil {
		return nil, fmt.Errorf("failed to open target database %s: %w", filepath.Base(targetClean), err)
	}
	defer targetDB.Close()

	return targetDB.MergeFromDatabasesWithCancel(sourceBasePaths, progressCb, cancelCheck)
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

	// Persist clean default preferences with empty DefaultDatabasePath & ReferenceDatabasePath
	freshPrefs := defaultAppPreferences()
	_, _ = a.SaveAppPreferences(*freshPrefs)

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

// ==========================================
// Standardized 3-Tier Database Import Pipeline
// (Original Archive vs. Master Reference vs. Personal User DB)
// ==========================================

// StandardImportRequest defines how an external database (.pgn or .si4) or PGN text
// should be imported into NewScid's 3-tier database architecture:
//   - "original_and_master": Save untouched Original Archive (Locked) + Merge & Deduplicate into Master Reference DB (Sync-Only)
//   - "original_only":       Save untouched Original Archive (Locked) as a standalone database
//   - "master_only":         Merge & Deduplicate directly into Master Reference DB (Sync-Only)
//   - "user_db":             Import & Deduplicate into Personal User Database (Editable, Default Startup DB)
type StandardImportRequest struct {
	SourcePath       string `json:"sourcePath"`
	PGNText          string `json:"pgnText"`
	ArchiveName      string `json:"archiveName"`
	ImportMode       string `json:"importMode"` // "original_and_master" | "original_only" | "master_only" | "user_db"
	MasterDBName     string `json:"masterDbName"`
	UserDBName       string `json:"userDbName"`
	TargetDir        string `json:"targetDir,omitempty"`
	SetMasterAsRef   bool   `json:"setMasterAsRef"`
	SetUserAsDefault bool   `json:"setUserAsDefault"`
	OpenAfterImport  bool   `json:"openAfterImport"`
}

// StandardImportResult reports the outcome of a standardized database import across Original, Master, and User tiers.
type StandardImportResult struct {
	ImportMode         string `json:"importMode"`
	OriginalBasePath   string `json:"originalBasePath,omitempty"`
	OriginalName       string `json:"originalName,omitempty"`
	OriginalGames      int    `json:"originalGames"`
	MasterBasePath     string `json:"masterBasePath,omitempty"`
	MasterName         string `json:"masterName,omitempty"`
	MasterTotalGames   int    `json:"masterTotalGames"`
	MasterAddedGames   int    `json:"masterAddedGames"`
	MasterSkippedDupes int    `json:"masterSkippedDupes"`
	UserBasePath       string `json:"userBasePath,omitempty"`
	UserName           string `json:"userName,omitempty"`
	UserTotalGames     int    `json:"userTotalGames"`
	UserAddedGames     int    `json:"userAddedGames"`
	UserSkippedDupes   int    `json:"userSkippedDupes"`
	ActiveBasePath     string `json:"activeBasePath,omitempty"`
	Summary            string `json:"summary"`
}

func sanitizeDBFilename(raw string, fallback string) string {
	s := strings.TrimSpace(raw)
	for _, ext := range []string{".si4", ".sg4", ".sn4", ".pgn"} {
		if strings.HasSuffix(strings.ToLower(s), ext) {
			s = s[:len(s)-len(ext)]
		}
	}
	s = filepath.Base(s)
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			b.WriteRune(r)
		} else if r == ' ' || r == '.' {
			b.WriteRune('_')
		}
	}
	out := strings.Trim(b.String(), "_-")
	if out == "" || out == "." {
		return fallback
	}
	return out
}

func copyScidBundleFiles(srcBase, dstBase string) error {
	cleanSrc, _ := filepath.Abs(srcBase)
	cleanDst, _ := filepath.Abs(dstBase)
	if cleanSrc == cleanDst {
		return nil
	}
	for _, ext := range []string{".si4", ".sg4", ".sn4"} {
		in, err := os.Open(cleanSrc + ext)
		if err != nil {
			return fmt.Errorf("opening %s%s: %w", srcBase, ext, err)
		}
		out, err := os.OpenFile(cleanDst+ext, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		if err != nil {
			_ = in.Close()
			return fmt.Errorf("creating %s%s: %w", dstBase, ext, err)
		}
		_, copyErr := io.Copy(out, in)
		_ = out.Close()
		_ = in.Close()
		if copyErr != nil {
			return fmt.Errorf("copying %s: %w", ext, copyErr)
		}
	}
	return nil
}

// EnsurePersonalUserDatabase ensures the user has a dedicated editable Personal User Database
// (defaulting to ~/.local/share/newscid/databases/My_Games), marks it Editable, sets it as the
// Default Startup User DB, and optionally opens it as the Active Working DB.
func (a *App) EnsurePersonalUserDatabase(openActive bool) (*DatabaseInfo, error) {
	prefs, _ := a.GetAppPreferences()
	dataRoot, _ := workstation.DefaultWorkstationDirs()
	dbDir := filepath.Join(dataRoot, "databases")
	_ = os.MkdirAll(dbDir, 0o755)

	targetBase := ""
	if prefs != nil && strings.TrimSpace(prefs.DefaultDatabasePath) != "" {
		cand := normalizeScidBasePath(prefs.DefaultDatabasePath)
		if _, err := os.Stat(cand + ".si4"); err == nil {
			targetBase = cand
		}
	}
	if targetBase == "" {
		targetBase = filepath.Join(dbDir, "My_Games")
	}

	if _, err := os.Stat(targetBase + ".si4"); err != nil {
		db := scid.NewDatabase()
		if err := db.Create(targetBase, "Personal User Database (My Games, Annotations & Repertoire)"); err != nil {
			return nil, fmt.Errorf("creating personal user database: %w", err)
		}
		_ = db.Close()
	}

	_, _ = a.SetDatabaseWritePolicy(targetBase, scid.WritePolicyEditable.String())
	_, _ = a.SetDefaultDatabase(targetBase)

	if openActive {
		return a.OpenDatabase(targetBase)
	}
	meta := scid.InspectDatabase(targetBase, "User Databases")
	return &DatabaseInfo{
		BasePath:    targetBase,
		Name:        filepath.Base(targetBase),
		NumGames:    meta.NumGames,
		Description: meta.Description,
		WritePolicy: scid.WritePolicyEditable.String(),
		IsLocked:    false,
	}, nil
}

// ImportStandardDatabase imports a .pgn file, .si4 database, or raw PGN text according to NewScid's
// standardized 3-tier database model (Original Archive -> Master Reference DB, or Personal User DB).
func (a *App) ImportStandardDatabase(req StandardImportRequest) (*StandardImportResult, error) {
	mode := strings.ToLower(strings.TrimSpace(req.ImportMode))
	if mode == "" {
		mode = "original_and_master"
	}

	targetDir := strings.TrimSpace(req.TargetDir)
	if targetDir == "" {
		dataRoot, _ := workstation.DefaultWorkstationDirs()
		targetDir = filepath.Join(dataRoot, "databases")
	}
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating databases directory: %w", err)
	}

	srcPath := strings.TrimSpace(req.SourcePath)
	pgnText := strings.TrimSpace(req.PGNText)
	if srcPath == "" && pgnText == "" {
		return nil, fmt.Errorf("either a database/PGN file path or PGN text is required")
	}

	opCtx, finishOp := a.startCancellableOperation("standard-import")
	defer finishOp()
	_ = opCtx

	res := &StandardImportResult{
		ImportMode: mode,
	}

	// Determine archive name
	defaultArchive := "Imported_Archive"
	if srcPath != "" {
		defaultArchive = sanitizeDBFilename(srcPath, "Imported_Archive")
	}
	archiveName := sanitizeDBFilename(req.ArchiveName, defaultArchive)

	// Helper to open or create a Scid database and merge from source
	mergeIntoTargetDB := func(dbBasePath, description, policy string, sources []string) (*scid.MergeStats, error) {
		cleanTarget := normalizeScidBasePath(dbBasePath)

		// Close active or ref handle temporarily if it points to cleanTarget so disk writes are consistent
		a.mu.Lock()
		wasActive := a.db != nil && a.db.IsOpen && normalizeScidBasePath(a.db.BasePath) == cleanTarget
		wasRef := a.refDB != nil && a.refDB.IsOpen && normalizeScidBasePath(a.refDB.BasePath) == cleanTarget
		if wasActive {
			_ = a.db.Close()
		}
		if wasRef {
			_ = a.refDB.Close()
		}
		a.mu.Unlock()

		tDB := scid.NewDatabase()
		if _, err := os.Stat(cleanTarget + ".si4"); err == nil {
			if err := tDB.Open(cleanTarget); err != nil {
				return nil, fmt.Errorf("opening %s: %w", filepath.Base(cleanTarget), err)
			}
		} else {
			if err := tDB.Create(cleanTarget, description); err != nil {
				return nil, fmt.Errorf("creating %s: %w", filepath.Base(cleanTarget), err)
			}
		}

		mStats, mergeErr := tDB.MergeFromDatabases(sources, func(sourceName string, scanned, imported, skipped int) {
			a.emitEvent("import:progress", map[string]interface{}{
				"target":   filepath.Base(cleanTarget),
				"source":   sourceName,
				"scanned":  scanned,
				"imported": imported,
				"skipped":  skipped,
			})
		})
		_ = tDB.Close()

		_, _ = a.SetDatabaseWritePolicy(cleanTarget, policy)

		a.mu.Lock()
		if wasActive {
			_ = a.db.Open(cleanTarget)
		}
		if wasRef {
			_ = a.refDB.Open(cleanTarget)
		}
		a.mu.Unlock()

		return mStats, mergeErr
	}

	// Step 1: If raw PGN text was provided without a file path, write it to a temporary PGN file for uniform processing
	effectiveSrc := srcPath
	if effectiveSrc == "" && pgnText != "" {
		tmpPGN := filepath.Join(os.TempDir(), fmt.Sprintf("newscid_import_%d.pgn", time.Now().UnixNano()))
		if err := os.WriteFile(tmpPGN, []byte(pgnText), 0o644); err != nil {
			return nil, fmt.Errorf("staging PGN text: %w", err)
		}
		defer os.Remove(tmpPGN)
		effectiveSrc = tmpPGN
	}

	isPGNSource := strings.HasSuffix(strings.ToLower(effectiveSrc), ".pgn")
	srcScidBase := normalizeScidBasePath(effectiveSrc)
	if !isPGNSource {
		if _, err := os.Stat(srcScidBase + ".si4"); err != nil {
			return nil, fmt.Errorf("source database not found: %s", effectiveSrc)
		}
	}

	// Step 2: Execute according to ImportMode
	switch mode {
	case "original_only", "original_and_master":
		origBase := filepath.Join(targetDir, archiveName)
		origDesc := fmt.Sprintf("Original Source Archive (%s)", archiveName)

		if isPGNSource {
			// Create or update Original Archive from PGN
			if _, err := os.Stat(origBase + ".si4"); err == nil {
				for _, ext := range []string{".si4", ".sg4", ".sn4"} {
					_ = os.Remove(origBase + ext)
				}
			}
			origDB := scid.NewDatabase()
			if err := origDB.Create(origBase, origDesc); err != nil {
				return nil, fmt.Errorf("creating original archive %s: %w", archiveName, err)
			}
			fPgn, err := os.Open(effectiveSrc)
			if err != nil {
				_ = origDB.Close()
				return nil, fmt.Errorf("opening PGN source: %w", err)
			}
			games, parseErr := chess.ParsePGNDatabase(fPgn)
			_ = fPgn.Close()
			if parseErr != nil {
				_ = origDB.Close()
				return nil, fmt.Errorf("parsing PGN source: %w", parseErr)
			}
			if _, err := origDB.ImportGames(games); err != nil {
				_ = origDB.Close()
				return nil, fmt.Errorf("importing games into original archive: %w", err)
			}
			res.OriginalGames = origDB.NumGames()
			_ = origDB.Close()
		} else {
			if err := copyScidBundleFiles(srcScidBase, origBase); err != nil {
				return nil, fmt.Errorf("copying original Scid archive: %w", err)
			}
			meta := scid.InspectDatabase(origBase, "")
			res.OriginalGames = meta.NumGames
		}

		// Protect Original Archive as locked so it stays a pristine source copy
		_, _ = a.SetDatabaseWritePolicy(origBase, scid.WritePolicyLocked.String())
		res.OriginalBasePath = origBase
		res.OriginalName = filepath.Base(origBase)

		if mode == "original_and_master" {
			masterName := sanitizeDBFilename(req.MasterDBName, "Master_Reference")
			masterBase := filepath.Join(targetDir, masterName)
			masterDesc := "Master Reference Database (Consolidated & Deduplicated Theory Base)"

			mStats, err := mergeIntoTargetDB(masterBase, masterDesc, scid.WritePolicySyncOnly.String(), []string{origBase})
			if err != nil {
				return nil, fmt.Errorf("merging original into master reference database: %w", err)
			}
			res.MasterBasePath = masterBase
			res.MasterName = filepath.Base(masterBase)
			res.MasterTotalGames = mStats.TargetTotalGames
			res.MasterAddedGames = mStats.Imported
			res.MasterSkippedDupes = mStats.DuplicatesSkipped

			if req.SetMasterAsRef {
				_, _ = a.SetReferenceDatabase(masterBase, true)
			}
			if req.OpenAfterImport {
				if info, err := a.OpenDatabase(masterBase); err == nil && info != nil {
					res.ActiveBasePath = info.BasePath
				}
			}
			res.Summary = fmt.Sprintf(
				"Saved untouched Original Archive '%s' (%d games, Locked) and merged +%d new games (%d duplicates skipped) into Master Reference '%s' (%d total games).",
				res.OriginalName, res.OriginalGames, res.MasterAddedGames, res.MasterSkippedDupes, res.MasterName, res.MasterTotalGames,
			)
		} else {
			if req.OpenAfterImport {
				if info, err := a.OpenDatabase(origBase); err == nil && info != nil {
					res.ActiveBasePath = info.BasePath
				}
			}
			res.Summary = fmt.Sprintf(
				"Imported standalone Original Archive '%s' (%d games, Locked against accidental edits).",
				res.OriginalName, res.OriginalGames,
			)
		}

	case "master_only":
		masterName := sanitizeDBFilename(req.MasterDBName, "Master_Reference")
		masterBase := filepath.Join(targetDir, masterName)
		masterDesc := "Master Reference Database (Consolidated & Deduplicated Theory Base)"

		srcForMerge := effectiveSrc
		if !isPGNSource {
			srcForMerge = srcScidBase
		}
		mStats, err := mergeIntoTargetDB(masterBase, masterDesc, scid.WritePolicySyncOnly.String(), []string{srcForMerge})
		if err != nil {
			return nil, fmt.Errorf("merging into master reference database: %w", err)
		}
		res.MasterBasePath = masterBase
		res.MasterName = filepath.Base(masterBase)
		res.MasterTotalGames = mStats.TargetTotalGames
		res.MasterAddedGames = mStats.Imported
		res.MasterSkippedDupes = mStats.DuplicatesSkipped

		if req.SetMasterAsRef {
			_, _ = a.SetReferenceDatabase(masterBase, true)
		}
		if req.OpenAfterImport {
			if info, err := a.OpenDatabase(masterBase); err == nil && info != nil {
				res.ActiveBasePath = info.BasePath
			}
		}
		res.Summary = fmt.Sprintf(
			"Merged +%d new games (%d duplicates skipped) into Master Reference '%s' (%d total games).",
			res.MasterAddedGames, res.MasterSkippedDupes, res.MasterName, res.MasterTotalGames,
		)

	case "user_db":
		prefs, _ := a.GetAppPreferences()
		userBase := ""
		if strings.TrimSpace(req.UserDBName) != "" {
			userBase = filepath.Join(targetDir, sanitizeDBFilename(req.UserDBName, "My_Games"))
		} else if prefs != nil && strings.TrimSpace(prefs.DefaultDatabasePath) != "" && req.TargetDir == "" {
			userBase = normalizeScidBasePath(prefs.DefaultDatabasePath)
		} else {
			userBase = filepath.Join(targetDir, "My_Games")
		}
		userDesc := "Personal User Database (My Games, Annotations & Repertoire)"

		srcForMerge := effectiveSrc
		if !isPGNSource {
			srcForMerge = srcScidBase
		}
		mStats, err := mergeIntoTargetDB(userBase, userDesc, scid.WritePolicyEditable.String(), []string{srcForMerge})
		if err != nil {
			return nil, fmt.Errorf("importing into personal user database: %w", err)
		}
		res.UserBasePath = userBase
		res.UserName = filepath.Base(userBase)
		res.UserTotalGames = mStats.TargetTotalGames
		res.UserAddedGames = mStats.Imported
		res.UserSkippedDupes = mStats.DuplicatesSkipped

		if req.SetUserAsDefault || (prefs != nil && strings.TrimSpace(prefs.DefaultDatabasePath) == "") {
			_, _ = a.SetDefaultDatabase(userBase)
		}
		if req.OpenAfterImport {
			if info, err := a.OpenDatabase(userBase); err == nil && info != nil {
				res.ActiveBasePath = info.BasePath
			}
		}
		res.Summary = fmt.Sprintf(
			"Imported +%d new games (%d duplicates skipped) into Personal User DB '%s' (%d total games, Editable).",
			res.UserAddedGames, res.UserSkippedDupes, res.UserName, res.UserTotalGames,
		)

	default:
		return nil, fmt.Errorf("unsupported import mode %q", req.ImportMode)
	}

	return res, nil
}

// ==========================================
// Unified User Profile, Home DB & Focus Book Setup Wizard
// ==========================================

// UserProfileSetupRequest defines the user's identity, home database, and focus books/repertoires.
type UserProfileSetupRequest struct {
	DisplayName            string   `json:"displayName"`
	Title                  string   `json:"title"`
	EstimatedElo           int      `json:"estimatedElo"`
	LichessUsername        string   `json:"lichessUsername"`
	ChessComUsername       string   `json:"chessComUsername"`
	FideID                 string   `json:"fideId"`
	Aliases                []string `json:"aliases"`
	HomeDatabaseName       string   `json:"homeDatabaseName"`
	EnsureHomeDB           bool     `json:"ensureHomeDb"`
	FocusWhiteRepertoireID string   `json:"focusWhiteRepertoireId"`
	FocusBlackRepertoireID string   `json:"focusBlackRepertoireId"`
	FocusPolyglotBookPath  string   `json:"focusPolyglotBookPath"`
	CompleteOnboarding     bool     `json:"completeOnboarding"`
}

// UserProfileSetupResult reports the current unified state of the user's profile, Home DB, Reference DB, and Focus Books.
type UserProfileSetupResult struct {
	DisplayName            string   `json:"displayName"`
	Title                  string   `json:"title"`
	EstimatedElo           int      `json:"estimatedElo"`
	LichessUsername        string   `json:"lichessUsername"`
	ChessComUsername       string   `json:"chessComUsername"`
	FideID                 string   `json:"fideId"`
	Aliases                []string `json:"aliases"`
	PersonaID              string   `json:"personaId"`
	SyncProfileID          string   `json:"syncProfileId"`
	HomeDBPath             string   `json:"homeDbPath"`
	HomeDBName             string   `json:"homeDbName"`
	HomeDBGames            int      `json:"homeDbGames"`
	ReferenceDBPath        string   `json:"referenceDbPath"`
	ReferenceDBName        string   `json:"referenceDbName"`
	ReferenceDBGames       int      `json:"referenceDbGames"`
	FocusWhiteRepertoireID string   `json:"focusWhiteRepertoireId"`
	FocusBlackRepertoireID string   `json:"focusBlackRepertoireId"`
	FocusPolyglotBookPath  string   `json:"focusPolyglotBookPath"`
	OnboardingCompleted    bool     `json:"onboardingCompleted"`
}

func buildAutomaticPlayerAliases(displayName, lichessUser, chessComUser string, extra []string) []string {
	var raw []string
	name := strings.TrimSpace(displayName)
	if name != "" {
		raw = append(raw, name)
		// If "First Last" format, automatically add "Last, First" PGN standard format
		if !strings.Contains(name, ",") {
			parts := strings.Fields(name)
			if len(parts) >= 2 {
				last := parts[len(parts)-1]
				first := strings.Join(parts[:len(parts)-1], " ")
				raw = append(raw, fmt.Sprintf("%s, %s", last, first))
				if len(first) > 0 {
					raw = append(raw, fmt.Sprintf("%s, %c", last, first[0]))
				}
			}
		}
	}
	if strings.TrimSpace(lichessUser) != "" {
		raw = append(raw, strings.TrimSpace(lichessUser))
	}
	if strings.TrimSpace(chessComUser) != "" {
		raw = append(raw, strings.TrimSpace(chessComUser))
	}
	raw = append(raw, extra...)
	return persona.CleanIdentifiers(raw)
}

// GetUserProfileStatus returns the current state of the user's identity, Home DB, Reference DB, Persona, and Focus Books.
func (a *App) GetUserProfileStatus() (*UserProfileSetupResult, error) {
	prefs, _ := a.GetAppPreferences()
	if prefs == nil {
		prefs = defaultAppPreferences()
	}

	// Check if a user profile persona already exists in personaMgr
	var userPersona *persona.Persona
	if a.personaMgr != nil {
		for _, p := range a.personaMgr.GetPersonas() {
			if p.IsUserProfile || (prefs.PlayerPersonaID != "" && p.ID == prefs.PlayerPersonaID) {
				userPersona = p
				break
			}
		}
	}

	displayName := strings.TrimSpace(prefs.PlayerDisplayName)
	title := strings.TrimSpace(prefs.PlayerTitle)
	elo := prefs.PlayerEstimatedElo
	lichess := strings.TrimSpace(prefs.LichessUsername)
	chesscom := strings.TrimSpace(prefs.ChessComUsername)
	fide := strings.TrimSpace(prefs.PlayerFideID)
	aliases := prefs.PlayerAliases
	personaID := strings.TrimSpace(prefs.PlayerPersonaID)
	whiteRep := strings.TrimSpace(prefs.FocusWhiteRepertoireID)
	blackRep := strings.TrimSpace(prefs.FocusBlackRepertoireID)
	bookPath := strings.TrimSpace(prefs.FocusPolyglotBookPath)

	if userPersona != nil {
		personaID = userPersona.ID
		if displayName == "" {
			displayName = userPersona.DisplayName
		}
		if title == "" {
			title = userPersona.Title
		}
		if elo <= 0 {
			elo = userPersona.EstimatedElo
		}
		if lichess == "" {
			lichess = userPersona.LichessUsername
		}
		if chesscom == "" {
			chesscom = userPersona.ChessComUsername
		}
		if fide == "" {
			fide = userPersona.FideID
		}
		if len(aliases) == 0 {
			aliases = userPersona.Identifiers
		}
		if whiteRep == "" {
			whiteRep = userPersona.WhiteRepertoireID
		}
		if blackRep == "" {
			blackRep = userPersona.BlackRepertoireID
		}
		if bookPath == "" {
			bookPath = userPersona.BookPath
		}
	}

	if len(aliases) == 0 && (displayName != "" || lichess != "" || chesscom != "") {
		aliases = buildAutomaticPlayerAliases(displayName, lichess, chesscom, nil)
	}

	// Default starter repertoires if none explicitly picked yet
	if whiteRep == "" {
		whiteRep = "white-openings-italian"
	}
	if blackRep == "" {
		blackRep = "black-openings-sicilian-kid"
	}

	homePath := normalizeScidBasePath(prefs.DefaultDatabasePath)
	homeName := ""
	homeGames := 0
	if homePath != "" {
		homeName = filepath.Base(homePath)
		if _, err := os.Stat(homePath + ".si4"); err == nil {
			meta := scid.InspectDatabase(homePath, "")
			homeGames = meta.NumGames
		}
	}

	refPath := normalizeScidBasePath(prefs.ReferenceDatabasePath)
	refName := ""
	refGames := 0
	if refPath != "" {
		refName = filepath.Base(refPath)
		if _, err := os.Stat(refPath + ".si4"); err == nil {
			meta := scid.InspectDatabase(refPath, "")
			refGames = meta.NumGames
		}
	}

	syncID := ""
	if pm := a.ensureProfileMgr(); pm != nil {
		for _, sp := range pm.GetProfiles() {
			if sp.ID == "user_home_sync" || !sp.IsStudyTarget {
				syncID = sp.ID
				break
			}
		}
	}

	return &UserProfileSetupResult{
		DisplayName:            displayName,
		Title:                  title,
		EstimatedElo:           elo,
		LichessUsername:        lichess,
		ChessComUsername:       chesscom,
		FideID:                 fide,
		Aliases:                aliases,
		PersonaID:              personaID,
		SyncProfileID:          syncID,
		HomeDBPath:             homePath,
		HomeDBName:             homeName,
		HomeDBGames:            homeGames,
		ReferenceDBPath:        refPath,
		ReferenceDBName:        refName,
		ReferenceDBGames:       refGames,
		FocusWhiteRepertoireID: whiteRep,
		FocusBlackRepertoireID: blackRep,
		FocusPolyglotBookPath:  bookPath,
		OnboardingCompleted:    prefs.OnboardingCompleted,
	}, nil
}

// SaveUserProfileSetup configures the user's identity, creates/links their Home Database,
// unifies their Personal Persona across all PGN aliases, sets up their Online Sync Profile,
// and saves their Focus Repertoires and Polyglot Opening Book in one step.
func (a *App) SaveUserProfileSetup(req UserProfileSetupRequest) (*UserProfileSetupResult, error) {
	prefs, _ := a.GetAppPreferences()
	if prefs == nil {
		prefs = defaultAppPreferences()
	}

	displayName := strings.TrimSpace(req.DisplayName)
	if displayName == "" {
		displayName = strings.TrimSpace(prefs.PlayerDisplayName)
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = "Club Player"
	}
	elo := req.EstimatedElo
	if elo <= 0 {
		elo = prefs.PlayerEstimatedElo
	}
	if elo <= 0 {
		elo = 1500
	}
	lichess := strings.TrimSpace(req.LichessUsername)
	chesscom := strings.TrimSpace(req.ChessComUsername)
	fide := strings.TrimSpace(req.FideID)

	aliases := buildAutomaticPlayerAliases(displayName, lichess, chesscom, req.Aliases)

	// 1. Ensure Home Database (defaulting to My_Games) if requested or if none configured
	dataRoot, _ := workstation.DefaultWorkstationDirs()
	dbDir := filepath.Join(dataRoot, "databases")
	_ = os.MkdirAll(dbDir, 0o755)

	homeBase := normalizeScidBasePath(prefs.DefaultDatabasePath)
	if strings.TrimSpace(req.HomeDatabaseName) != "" {
		homeBase = filepath.Join(dbDir, sanitizeDBFilename(req.HomeDatabaseName, "My_Games"))
	} else if homeBase == "" && (req.EnsureHomeDB || displayName != "") {
		homeBase = filepath.Join(dbDir, "My_Games")
	}

	if homeBase != "" {
		if _, err := os.Stat(homeBase + ".si4"); err != nil {
			db := scid.NewDatabase()
			desc := "Personal User Database (My Games, Annotations & Repertoire)"
			if displayName != "" {
				desc = fmt.Sprintf("%s — Home Database (My Games, Annotations & Repertoire)", displayName)
			}
			if err := db.Create(homeBase, desc); err != nil {
				return nil, fmt.Errorf("creating home database: %w", err)
			}
			_ = db.Close()
		}
		prefs.DefaultDatabasePath = homeBase
		prefs.AutoOpenDefaultDB = true
		if prefs.DatabasePolicies == nil {
			prefs.DatabasePolicies = make(map[string]string)
		}
		prefs.DatabasePolicies[homeBase] = scid.WritePolicyEditable.String()
	}

	// 2. Save User Profile fields onto AppPreferences
	prefs.PlayerDisplayName = displayName
	prefs.PlayerTitle = title
	prefs.PlayerEstimatedElo = elo
	prefs.LichessUsername = lichess
	prefs.ChessComUsername = chesscom
	prefs.PlayerFideID = fide
	prefs.PlayerAliases = aliases
	if strings.TrimSpace(req.FocusWhiteRepertoireID) != "" {
		prefs.FocusWhiteRepertoireID = strings.TrimSpace(req.FocusWhiteRepertoireID)
	} else if prefs.FocusWhiteRepertoireID == "" {
		prefs.FocusWhiteRepertoireID = "white-openings-italian"
	}
	if strings.TrimSpace(req.FocusBlackRepertoireID) != "" {
		prefs.FocusBlackRepertoireID = strings.TrimSpace(req.FocusBlackRepertoireID)
	} else if prefs.FocusBlackRepertoireID == "" {
		prefs.FocusBlackRepertoireID = "black-openings-sicilian-kid"
	}
	if strings.TrimSpace(req.FocusPolyglotBookPath) != "" {
		prefs.FocusPolyglotBookPath = strings.TrimSpace(req.FocusPolyglotBookPath)
	}
	if req.CompleteOnboarding {
		prefs.OnboardingCompleted = true
	}

	// 3. Create or update Personal Persona in PersonaManager
	if displayName != "" && a.personaMgr != nil {
		personaID := strings.TrimSpace(prefs.PlayerPersonaID)
		if personaID == "" {
			for _, existing := range a.personaMgr.GetPersonas() {
				if existing.IsUserProfile || strings.EqualFold(existing.DisplayName, displayName) {
					personaID = existing.ID
					break
				}
			}
		}
		pObj := &persona.Persona{
			ID:                personaID,
			DisplayName:       displayName,
			Identifiers:       aliases,
			DatabasePath:      homeBase,
			BookPath:          prefs.FocusPolyglotBookPath,
			WhiteRepertoireID: prefs.FocusWhiteRepertoireID,
			BlackRepertoireID: prefs.FocusBlackRepertoireID,
			IsUserProfile:     true,
			Title:             title,
			Bio:               "Primary Local Player Profile & Home Repertoire",
			EstimatedElo:      elo,
			ChessComUsername:  chesscom,
			LichessUsername:   lichess,
			FideID:            fide,
		}
		if savedP, err := a.SavePersona(pObj); err == nil && savedP != nil {
			prefs.PlayerPersonaID = savedP.ID
		}
	}

	// 4. Create or update Online Sync Profile linked to Home DB
	if (lichess != "" || chesscom != "") && a.ensureProfileMgr() != nil {
		profName := displayName
		if profName == "" {
			if lichess != "" {
				profName = lichess
			} else {
				profName = chesscom
			}
		}
		_ = a.profileMgr.SaveProfile(&online.Profile{
			ID:               "user_home_sync",
			Name:             profName + " (Home DB Sync)",
			TargetDatabase:   homeBase,
			TargetMode:       online.TargetDedicatedBase,
			IsStudyTarget:    false,
			LichessUsername:  lichess,
			ChessComUsername: chesscom,
			AutoDeduplicate:  true,
		})
	}

	// 5. Save preferences and add user to Notable Players dropdown
	_, err := a.SaveAppPreferences(*prefs)
	if err != nil {
		return nil, err
	}
	if displayName != "" {
		_, _ = a.AddCustomNotablePlayer(displayName, aliases, title, "My Profile")
	}

	// 6. If no database is currently open and homeBase exists, open it as Active DB
	a.mu.RLock()
	hasOpenDB := a.db != nil && a.db.IsOpen
	a.mu.RUnlock()
	if !hasOpenDB && homeBase != "" {
		_, _ = a.OpenDatabase(homeBase)
	}

	return a.GetUserProfileStatus()
}

// SyncUserHomeGames fetches the user's latest Lichess and Chess.com games directly into their Home Database.
func (a *App) SyncUserHomeGames(maxGames int) (*online.SyncResult, error) {
	status, err := a.GetUserProfileStatus()
	if err != nil {
		return nil, err
	}
	if status.LichessUsername == "" && status.ChessComUsername == "" {
		return nil, fmt.Errorf("no Lichess or Chess.com username configured in your profile")
	}
	if maxGames <= 0 {
		maxGames = 250
	}

	// Ensure Home DB exists
	homeInfo, err := a.EnsurePersonalUserDatabase(false)
	if err != nil {
		return nil, err
	}

	_, finish := a.startCancellableOperation("home-online-sync")
	defer finish()

	opts := online.ImportPlayerOptions{
		PlayerName:       status.DisplayName,
		LichessUsername:  status.LichessUsername,
		ChessComUsername: status.ChessComUsername,
		TargetMode:       online.TargetDedicatedBase,
		DedicatedPath:    homeInfo.BasePath,
		AutoDeduplicate:  true,
		MaxGames:         maxGames,
		ProgressFn: func(stage string, current, total int, message string) {
			a.emitEvent("online:fetch-progress", map[string]interface{}{
				"stage":   stage,
				"current": current,
				"total":   total,
				"message": message,
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

	// Re-open Home DB if it is currently active so new games appear immediately
	a.mu.RLock()
	isHomeActive := a.db != nil && a.db.IsOpen && normalizeScidBasePath(a.db.BasePath) == normalizeScidBasePath(homeInfo.BasePath)
	a.mu.RUnlock()
	if isHomeActive || activeDB == nil || !activeDB.IsOpen {
		_, _ = a.OpenDatabase(homeInfo.BasePath)
	}

	return res, nil
}

// BuildPersonaOpeningBook compiles a Polyglot (.bin) opening book representing the moves played
// by the specified persona across their dedicated and active databases, saves it to NewScid's opening books
// directory, and attaches it to the persona's BookPath.
func (a *App) BuildPersonaOpeningBook(personaID string, targetColor string, maxDepth int) (string, int, error) {
	if a.personaMgr == nil {
		return "", 0, fmt.Errorf("persona manager not initialized")
	}
	p, err := a.personaMgr.GetPersona(personaID)
	if err != nil {
		return "", 0, err
	}

	a.mu.RLock()
	activeDB := a.db
	a.mu.RUnlock()

	var allGames []*chess.Game
	seenSigs := make(map[string]bool)

	collectFromDB := func(db *scid.Database) {
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

		numGames := db.NumGames()
		for i := 0; i < numGames; i++ {
			db.RLock()
			if i >= len(db.Entries) {
				db.RUnlock()
				break
			}
			entry := db.Entries[i]
			if entry == nil || (entry.Flags&scid.FlagDelete) != 0 {
				db.RUnlock()
				continue
			}
			w := strings.ToLower(db.Names.GetName(scid.NamePlayer, entry.WhiteID))
			b := strings.ToLower(db.Names.GetName(scid.NamePlayer, entry.BlackID))
			entryDate := entry.Date
			entryResult := entry.Result
			entryHalfMoves := entry.NumHalfMoves
			db.RUnlock()

			if !matchPlayerLookup(w, nameLookup) && !matchPlayerLookup(b, nameLookup) {
				continue
			}
			sig := fmt.Sprintf("%s|%s|%s|%s|%d", entryDate, w, b, entryResult, entryHalfMoves)
			if seenSigs[sig] {
				continue
			}
			seenSigs[sig] = true
			if g, err := db.GetGame(i + 1); err == nil && g != nil {
				allGames = append(allGames, g)
			}
		}
	}

	if activeDB != nil && activeDB.IsOpen {
		collectFromDB(activeDB)
	}
	if p.DatabasePath != "" {
		dedDB := &scid.Database{}
		if err := dedDB.Open(p.DatabasePath); err == nil {
			defer dedDB.Close()
			collectFromDB(dedDB)
		}
	}

	if len(allGames) == 0 {
		return "", 0, fmt.Errorf("no games found for persona %q to compile a book from — import or fetch games first", p.DisplayName)
	}

	dataRoot, _ := workstation.DefaultWorkstationDirs()
	booksDir := filepath.Join(dataRoot, "books", "opening")
	_ = os.MkdirAll(booksDir, 0o755)

	slug := sanitizeDBFilename(p.DisplayName, "Persona")
	targetPath := filepath.Join(booksDir, fmt.Sprintf("%s_Book.bin", slug))

	ids := append([]string{p.DisplayName}, p.Identifiers...)
	count, err := persona.CompilePersonaPolyglotBook(allGames, ids, targetColor, maxDepth, targetPath)
	if err != nil {
		return "", 0, err
	}

	p.BookPath = targetPath
	_, _ = a.SavePersona(p)

	a.mu.Lock()
	if a.polyglotBooks != nil {
		delete(a.polyglotBooks, targetPath)
	}
	a.mu.Unlock()

	return targetPath, count, nil
}

// AssignPersonaBooks assigns a Polyglot (.bin) opening book, anti-book, or White/Black repertoires to a Persona.
func (a *App) AssignPersonaBooks(personaID string, bookPath string, antiBookPath string, whiteRepID string, blackRepID string) (*persona.Persona, error) {
	if a.personaMgr == nil {
		return nil, fmt.Errorf("persona manager not initialized")
	}
	p, err := a.personaMgr.GetPersona(personaID)
	if err != nil {
		return nil, err
	}
	p.BookPath = strings.TrimSpace(bookPath)
	p.AntiBookPath = strings.TrimSpace(antiBookPath)
	p.WhiteRepertoireID = strings.TrimSpace(whiteRepID)
	p.BlackRepertoireID = strings.TrimSpace(blackRepID)
	return a.SavePersona(p)
}

// CompileRepertoireToPolyglotBook compiles a curated Opening Repertoire into a Polyglot (.bin) book in NewScid's opening books folder.
func (a *App) CompileRepertoireToPolyglotBook(repID string, setAsFocusBook bool) (string, int, error) {
	if a.repMgr == nil {
		return "", 0, fmt.Errorf("repertoire manager not initialized")
	}
	rep, err := a.repMgr.GetRepertoire(repID)
	if err != nil {
		return "", 0, err
	}

	dataRoot, _ := workstation.DefaultWorkstationDirs()
	booksDir := filepath.Join(dataRoot, "books", "opening")
	_ = os.MkdirAll(booksDir, 0o755)

	slug := sanitizeDBFilename(rep.Name, rep.ID)
	targetPath := filepath.Join(booksDir, fmt.Sprintf("Repertoire_%s.bin", slug))

	count, err := a.repMgr.CompileToPolyglotBook(repID, targetPath)
	if err != nil {
		return "", 0, err
	}

	a.mu.Lock()
	if a.polyglotBooks != nil {
		delete(a.polyglotBooks, targetPath)
	}
	a.mu.Unlock()

	if setAsFocusBook {
		prefs, _ := a.GetAppPreferences()
		if prefs != nil {
			prefs.FocusPolyglotBookPath = targetPath
			_, _ = a.SaveAppPreferences(*prefs)
		}
	}

	return targetPath, count, nil
}


