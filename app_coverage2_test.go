package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"newscid/pkg/chess"
	"newscid/pkg/database/scid"
	"newscid/pkg/fileformat"
	"newscid/pkg/online"
	"newscid/pkg/persona"
	"newscid/pkg/workstation"
)

func createDummyBin(t *testing.T, dir, name string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(p, []byte("#!/bin/sh\nexit 0\n"), 0o755))
	return p
}

func TestCovWorkspacePrefsWindowAndSync(t *testing.T) {
	app, tmp := newIsolatedApp(t)

	// OpenClipbase creates then re-opens
	clip1, err := app.OpenClipbase()
	require.NoError(t, err)
	assert.NotNil(t, clip1)
	clip2, err := app.OpenClipbase()
	require.NoError(t, err)
	assert.Equal(t, clip1.BasePath, clip2.BasePath)

	// GetWindowStartupConfig
	cfg := app.GetWindowStartupConfig()
	assert.NotEmpty(t, cfg.Mode)

	// Fake hyprctl on PATH so dispatchHyprlandWorkspace runs cleanly
	binDir := filepath.Join(tmp, "bin")
	require.NoError(t, os.MkdirAll(binDir, 0o755))
	createDummyBin(t, binDir, "hyprctl")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	dispatchHyprlandWorkspace(0, "2", false)
	dispatchHyprlandWorkspace(os.Getpid(), "2", false)
	dispatchHyprlandWorkspace(os.Getpid(), "3", true)

	// MoveCurrentWindowToWorkspace: empty, main mode, panel mode
	origCfg := windowStartupCfg
	defer func() { windowStartupCfg = origCfg }()

	require.NoError(t, app.MoveCurrentWindowToWorkspace(""))
	windowStartupCfg = WindowStartupConfig{Mode: "main"}
	require.NoError(t, app.MoveCurrentWindowToWorkspace("2"))

	windowStartupCfg = WindowStartupConfig{Mode: "panel", PanelID: "tree"}
	require.NoError(t, app.MoveCurrentWindowToWorkspace("4"))

	// LaunchPanelOnWorkspace / LaunchDetachedPanelWindow using existing alive PID (os.Getpid())
	prefsMu.Lock()
	detachedPanelPIDs["tree"] = os.Getpid()
	prefsMu.Unlock()
	defer func() {
		prefsMu.Lock()
		delete(detachedPanelPIDs, "tree")
		prefsMu.Unlock()
	}()

	require.NoError(t, app.LaunchDetachedPanelWindow("tree", "Tree Panel", chess.StartingFEN, ""))
	require.NoError(t, app.LaunchPanelOnWorkspace("", "3", "", chess.StartingFEN, ""))

	// LaunchPortalWindow: custom browser cmd (/bin/true) + chromium dummy on PATH
	dummyTrue := createDummyBin(t, binDir, "dummy-browser")
	prefs, err := app.GetAppPreferences()
	require.NoError(t, err)
	prefs.AutoCapturePortalGames = false
	prefs.CustomBrowserCmd = dummyTrue
	_, err = app.SaveAppPreferences(*prefs)
	require.NoError(t, err)

	require.NoError(t, app.LaunchPortalWindow("lichess", "", "app"))
	require.NoError(t, app.LaunchPortalWindow("chesscom", "", "app"))

	// Clear custom browser cmd and put dummy chromium on PATH
	prefs.CustomBrowserCmd = ""
	_, err = app.SaveAppPreferences(*prefs)
	require.NoError(t, err)
	createDummyBin(t, binDir, "chromium")
	t.Setenv("PATH", binDir)
	require.NoError(t, app.LaunchPortalWindow("lichess", "https://lichess.org", ""))

	// Replace chromium with firefox dummy
	require.NoError(t, os.Remove(filepath.Join(binDir, "chromium")))
	createDummyBin(t, binDir, "firefox")
	require.NoError(t, app.LaunchPortalWindow("chesscom", "https://www.chess.com", ""))

	// Shared workspace state: not exist -> publish -> read
	st0, err := app.GetSharedWorkspaceState()
	require.NoError(t, err)
	assert.Nil(t, st0)

	require.NoError(t, app.PublishWorkspaceState(SharedWorkspaceState{
		FEN:          chess.StartingFEN,
		ActiveGameID: 1,
		CurrentPly:   0,
		DBPath:       clip1.BasePath,
	}))
	st1, err := app.GetSharedWorkspaceState()
	require.NoError(t, err)
	require.NotNil(t, st1)
	assert.Equal(t, chess.StartingFEN, st1.FEN)
}

func TestCovWorkspaceLocalMergePoliciesAndConsolidation(t *testing.T) {
	app, tmp := newIsolatedApp(t)

	// GetDatabaseWritePolicy / SetDatabaseWritePolicy when no DB open
	assert.Equal(t, scid.WritePolicyEditable.String(), app.GetDatabaseWritePolicy(""))
	_, err := app.SetDatabaseWritePolicy("", "locked")
	assert.Error(t, err)

	src1 := createTempScidDB(t, tmp, "merge_src1", covSamplePGN)
	src2 := createTempScidDB(t, tmp, "merge_src2", covBlunderPGN)
	target := createTempScidDB(t, tmp, "merge_target", covSamplePGN)

	// Open target as active DB and test policy resolution
	_, err = app.OpenDatabase(target)
	require.NoError(t, err)
	assert.Equal(t, scid.WritePolicyEditable.String(), app.GetDatabaseWritePolicy(""))
	assert.Equal(t, scid.WritePolicyEditable.String(), app.GetDatabaseWritePolicy(src1))

	// UpdateDatabaseFromLocal: locked target error
	_, err = app.SetDatabaseWritePolicy(target, "locked")
	require.NoError(t, err)
	_, err = app.UpdateDatabaseFromLocal(target, []string{src2})
	assert.Error(t, err)

	// Unlock and merge into active DB
	_, err = app.SetDatabaseWritePolicy(target, "editable")
	require.NoError(t, err)
	statsActive, err := app.UpdateDatabaseFromLocal("", []string{src2})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, statsActive.Imported, 1)

	// Merge into reference DB
	refTarget := createTempScidDB(t, tmp, "merge_ref", covSamplePGN)
	_, err = app.SetReferenceDatabase(refTarget, true)
	require.NoError(t, err)
	require.NoError(t, app.openReferenceDatabaseInternal(refTarget))
	statsRef, err := app.UpdateDatabaseFromLocal(refTarget, []string{src2})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, statsRef.Imported, 1)

	// Merge into closed target DB on disk
	diskTarget := createTempScidDB(t, tmp, "merge_disk", covSamplePGN)
	statsDisk, err := app.UpdateDatabaseFromLocal(diskTarget, []string{src2})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, statsDisk.Imported, 1)

	// Auto-discover sources (empty sourceBasePaths)
	statsAuto, err := app.UpdateDatabaseFromLocal(diskTarget, nil)
	require.NoError(t, err)
	assert.NotNil(t, statsAuto)

	// Error when no target open and no default DB
	require.NoError(t, app.CloseDatabase())
	_, err = app.UpdateDatabaseFromLocal("", nil)
	assert.Error(t, err)

	// copyScidBundleFiles: same path, valid copy, missing source error
	require.NoError(t, copyScidBundleFiles(src1, src1))
	copiedBase := filepath.Join(tmp, "copied_bundle")
	require.NoError(t, copyScidBundleFiles(src1, copiedBase))
	assert.FileExists(t, fileformat.ScidIndexPath(copiedBase))
	assert.Error(t, copyScidBundleFiles(filepath.Join(tmp, "does_not_exist"), filepath.Join(tmp, "out_err")))

	// ImportStandardDatabase with .si4 source (original_only, original_and_master, master_only, user_db, invalid mode)
	resOrig, err := app.ImportStandardDatabase(StandardImportRequest{
		SourcePath:      src1 + ".si4",
		ImportMode:      "original_only",
		ArchiveName:     "StdOrigSI4",
		TargetDir:       filepath.Join(tmp, "std_out"),
		OpenAfterImport: true,
	})
	require.NoError(t, err)
	assert.Greater(t, resOrig.OriginalGames, 0)

	resMaster, err := app.ImportStandardDatabase(StandardImportRequest{
		SourcePath:      src2 + ".si4",
		ImportMode:      "master_only",
		MasterDBName:    "StdMaster",
		TargetDir:       filepath.Join(tmp, "std_out"),
		SetMasterAsRef:  true,
		OpenAfterImport: true,
	})
	require.NoError(t, err)
	assert.Greater(t, resMaster.MasterTotalGames, 0)

	resUser, err := app.ImportStandardDatabase(StandardImportRequest{
		PGNText:          covSamplePGN,
		ImportMode:       "user_db",
		UserDBName:       "StdUser",
		TargetDir:        filepath.Join(tmp, "std_out"),
		SetUserAsDefault: true,
		OpenAfterImport:  true,
	})
	require.NoError(t, err)
	assert.Greater(t, resUser.UserTotalGames, 0)

	_, err = app.ImportStandardDatabase(StandardImportRequest{
		SourcePath: src1 + ".si4",
		ImportMode: "bogus_mode",
	})
	assert.Error(t, err)
	_, err = app.ImportStandardDatabase(StandardImportRequest{})
	assert.Error(t, err)

	// BuildConsolidatedDatabase with SetAsReference + SetAsDefaultUser + auto-discover
	consRes, err := app.BuildConsolidatedDatabase(scid.ConsolidateDatabaseOptions{
		TargetName:       "CovConsolidated",
		SetAsReference:   true,
		SetAsDefaultUser: true,
	})
	require.NoError(t, err)
	assert.NotNil(t, consRes)

	// ConsolidateOnlineCatalogDatabases with local sources (empty sourceIDs so no network calls)
	onlineConsRes, err := app.ConsolidateOnlineCatalogDatabases(nil, true, "", 50, 0, true)
	require.NoError(t, err)
	assert.NotNil(t, onlineConsRes)

	// Notable players: empty name error, add, update existing, remove custom, remove famous, reset
	_, err = app.AddCustomNotablePlayer("   ", nil, "", "")
	assert.Error(t, err)
	_, err = app.AddCustomNotablePlayer("Cov Notable", []string{"CovN"}, "GM", "Modern")
	require.NoError(t, err)
	_, err = app.AddCustomNotablePlayer("Cov Notable", []string{"CovN2"}, "GM", "Modern")
	require.NoError(t, err)
	_, err = app.RemoveNotablePlayer("Cov Notable")
	require.NoError(t, err)
	_, err = app.RemoveNotablePlayer("Magnus Carlsen")
	require.NoError(t, err)
	_, err = app.ResetNotablePlayersToDefault()
	require.NoError(t, err)

	// ImportAssetToWorkstation
	pgnAsset := filepath.Join(tmp, "asset_test.pgn")
	require.NoError(t, os.WriteFile(pgnAsset, []byte(covSamplePGN), 0o644))
	_, err = app.ImportAssetToWorkstation("databases", pgnAsset)
	require.NoError(t, err)
}

func TestCovWorkspaceUserProfileAndHomeSync(t *testing.T) {
	app, _ := newIsolatedApp(t)
	srv := newMockOnlineServer(t, covSamplePGN)
	app.syncMgr.Lichess.BaseURL = srv.URL
	app.syncMgr.ChessCom.BaseURL = srv.URL

	// SyncUserHomeGames fails before profile usernames are configured
	_, err := app.SyncUserHomeGames(10)
	assert.Error(t, err)

	// SaveUserProfileSetup
	setupRes, err := app.SaveUserProfileSetup(UserProfileSetupRequest{
		DisplayName:        "Cov Hero",
		Title:              "FM",
		EstimatedElo:       2150,
		LichessUsername:    "CovHero",
		ChessComUsername:   "CovHero",
		FideID:             "1234567",
		Aliases:            []string{"Hero, C."},
		HomeDatabaseName:   "Cov_Home_DB",
		EnsureHomeDB:       true,
		CompleteOnboarding: true,
	})
	require.NoError(t, err)
	assert.Equal(t, "Cov Hero", setupRes.DisplayName)
	assert.NotEmpty(t, setupRes.HomeDBPath)

	// GetUserProfileStatus when prefs fields are cleared so it falls back to userPersona
	prefs, err := app.GetAppPreferences()
	require.NoError(t, err)
	prefs.PlayerDisplayName = ""
	prefs.PlayerTitle = ""
	prefs.PlayerEstimatedElo = 0
	prefs.LichessUsername = ""
	prefs.ChessComUsername = ""
	prefs.PlayerFideID = ""
	prefs.PlayerAliases = nil
	prefs.FocusWhiteRepertoireID = ""
	prefs.FocusBlackRepertoireID = ""
	prefs.FocusPolyglotBookPath = ""
	_, err = app.SaveAppPreferences(*prefs)
	require.NoError(t, err)

	statusFB, err := app.GetUserProfileStatus()
	require.NoError(t, err)
	assert.Equal(t, "Cov Hero", statusFB.DisplayName)
	assert.Equal(t, "CovHero", statusFB.LichessUsername)

	// SyncUserHomeGames succeeds via mock httptest server
	syncRes, err := app.SyncUserHomeGames(0)
	require.NoError(t, err)
	assert.NotNil(t, syncRes)
	assert.GreaterOrEqual(t, syncRes.TotalDownloaded, 1)

	// EnsurePersonalUserDatabase(false)
	info, err := app.EnsurePersonalUserDatabase(false)
	require.NoError(t, err)
	assert.NotEmpty(t, info.BasePath)
}

func TestCovPortalAutoCaptureFull(t *testing.T) {
	app, tmp := newIsolatedApp(t)
	srv := newMockOnlineServer(t, covSamplePGN)
	app.syncMgr.Lichess.BaseURL = srv.URL
	app.syncMgr.ChessCom.BaseURL = srv.URL

	// Reset global portal watcher state safely after test
	origInterval := portalPollInterval
	portalPollInterval = 80 * time.Millisecond
	defer func() {
		_, _ = app.StopLivePortalWatcher()
		portalPollInterval = origInterval
	}()

	// Helper unit checks: isLikelyLichessGameID, extractLichessGameID, formatCapturedGameSummary
	assert.True(t, isLikelyLichessGameID("abcd1234"))
	assert.True(t, isLikelyLichessGameID("abcd1234efgh"))
	assert.False(t, isLikelyLichessGameID("short"))
	assert.False(t, isLikelyLichessGameID("abcd-123"))

	assert.Equal(t, "abcd1234", extractLichessGameID("https://lichess.org/game/export/abcd1234efgh/white"))
	assert.Equal(t, "abcd1234", extractLichessGameID("https://lichess.org/abcd1234"))
	assert.Equal(t, "abcd1234", extractLichessGameID("abcd1234efgh"))
	assert.Equal(t, "abcd1234", extractLichessGameID("abcd1234"))
	assert.Equal(t, "", extractLichessGameID("https://example.com/nope"))

	assert.Equal(t, "Online Game", formatCapturedGameSummary(nil))
	assert.Equal(t, "? vs ? (*)", formatCapturedGameSummary(&chess.Game{Tags: map[string]string{}}))
	assert.Contains(t, formatCapturedGameSummary(&chess.Game{Tags: map[string]string{
		"White": "A", "Black": "B", "Result": "1-0", "Event": "Blitz",
	}}), "Blitz")

	// CheckAndCaptureLivePortalGames with no usernames configured -> error
	_, err := app.CheckAndCaptureLivePortalGames()
	assert.Error(t, err)

	// CaptureOnlineGameFromURLOrPGN error paths
	_, err = app.CaptureOnlineGameFromURLOrPGN("   ")
	assert.Error(t, err)
	_, err = app.CaptureOnlineGameFromURLOrPGN("not a valid url or pgn")
	assert.Error(t, err)
	_, err = app.CaptureOnlineGameFromURLOrPGN("https://lichess.org/short")
	assert.Error(t, err)

	// Set ChessComUsername in preferences so Chess.com URL capture can fetch from player archive
	pPrefs, err := app.GetAppPreferences()
	require.NoError(t, err)
	pPrefs.ChessComUsername = "CovHero"
	_, err = app.SaveAppPreferences(*pPrefs)
	require.NoError(t, err)

	// Capture from Lichess URL, bare Lichess ID, Chess.com URL, and raw PGN (plus duplicate re-capture)
	capLichess, err := app.CaptureOnlineGameFromURLOrPGN("https://lichess.org/cov12345")
	require.NoError(t, err)
	assert.NotNil(t, capLichess)

	capLichessID, err := app.CaptureOnlineGameFromURLOrPGN("cov12345")
	require.NoError(t, err)
	assert.Equal(t, 0, capLichessID.NewGamesImported) // duplicate returns PGN immediately
	assert.NotEmpty(t, capLichessID.LatestGamePGN)

	capCC, err := app.CaptureOnlineGameFromURLOrPGN("https://www.chess.com/game/live/999888777")
	require.NoError(t, err)
	assert.NotNil(t, capCC)

	capPGN, err := app.CaptureOnlineGameFromURLOrPGN(covBlunderPGN)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, capPGN.NewGamesImported, 1)

	// ConfigureLivePortalAutoCapture enables watcher, creates profile, then updates existing profile
	stOn, err := app.ConfigureLivePortalAutoCapture("CovHero", "CovHero", true, true)
	require.NoError(t, err)
	assert.True(t, stOn.WatcherActive)

	// Calling StartLivePortalWatcher again when already active is a no-op
	stAlready, err := app.StartLivePortalWatcher()
	require.NoError(t, err)
	assert.True(t, stAlready.WatcherActive)

	// Calling ConfigureLivePortalAutoCapture again updates existing profile and disables watcher
	stOff, err := app.ConfigureLivePortalAutoCapture("CovHero", "CovHero", false, false)
	require.NoError(t, err)
	assert.False(t, stOff.WatcherActive)

	// Also test resolvePortalCredentials fallback from profileMgr
	app2, _ := newIsolatedApp(t)
	require.NoError(t, app2.SaveSyncProfile(&online.Profile{
		ID:               "portal-cred-prof",
		Name:             "PortalProfile",
		LichessUsername:  "p_li",
		LichessToken:     "tok_123",
		ChessComUsername: "p_cc",
	}))
	li, tok, cc := app2.resolvePortalCredentials()
	assert.Equal(t, "p_li", li)
	assert.Equal(t, "tok_123", tok)
	assert.Equal(t, "p_cc", cc)

	// Test openOrCreatePortalTargetDB on existing DB vs new DB
	dbPath := filepath.Join(tmp, "portal_target_test")
	db1, err := openOrCreatePortalTargetDB(dbPath, "Portal Test")
	require.NoError(t, err)
	require.NoError(t, db1.Close())
	db2, err := openOrCreatePortalTargetDB(dbPath, "Portal Test")
	require.NoError(t, err)
	require.NoError(t, db2.Close())
}

func TestCovWorkspaceAndPortalDeepBranches(t *testing.T) {
	app, tmp := newIsolatedApp(t)
	srv := newMockOnlineServer(t, covSamplePGN)
	app.syncMgr.Lichess.BaseURL = srv.URL
	app.syncMgr.ChessCom.BaseURL = srv.URL

	// 1. LaunchPanelOnWorkspace spawning real subprocess (test binary exits immediately on unknown flag)
	require.NoError(t, app.LaunchPanelOnWorkspace("notation", "", "", chess.StartingFEN, filepath.Join(tmp, "dummy.si4")))
	prefsMu.Lock()
	delete(detachedPanelPIDs, "notation")
	prefsMu.Unlock()

	// 2. LaunchPortalWindow native-window mode
	prefs, err := app.GetAppPreferences()
	require.NoError(t, err)
	prefs.AutoCapturePortalGames = false
	_, err = app.SaveAppPreferences(*prefs)
	require.NoError(t, err)
	require.NoError(t, app.LaunchPortalWindow("lichess", "https://lichess.org", "native-window"))

	// 3. GetAppPreferences corrupt JSON fallback & SaveAppPreferences normalization
	prefFile := preferencesFilePath()
	require.NoError(t, os.WriteFile(prefFile, []byte("{corrupt json"), 0o644))
	pFallback, err := app.GetAppPreferences()
	require.NoError(t, err)
	assert.NotNil(t, pFallback)

	// Write legacy XDG file and remove primary to test fallback read
	require.NoError(t, os.Remove(prefFile))
	xdgFile := xdgPreferencesFilePath()
	require.NoError(t, os.MkdirAll(filepath.Dir(xdgFile), 0o755))
	require.NoError(t, os.WriteFile(xdgFile, []byte(`{"lichessUsername":"xdg_user"}`), 0o644))
	pXdg, err := app.GetAppPreferences()
	require.NoError(t, err)
	assert.Equal(t, "xdg_user", pXdg.LichessUsername)

	// 4. ImportStandardDatabase additional branches: PGN original_and_master (overwriting existing origBase),
	//    missing source file, user_db with default DB path, and mergeIntoTargetDB when active+ref point to target
	pgnFile := filepath.Join(tmp, "std_import.pgn")
	require.NoError(t, os.WriteFile(pgnFile, []byte(covSamplePGN), 0o644))
	outDir := filepath.Join(tmp, "std_deep")

	res1, err := app.ImportStandardDatabase(StandardImportRequest{
		SourcePath:      pgnFile,
		ImportMode:      "original_and_master",
		ArchiveName:     "OrigPGN",
		MasterDBName:    "MasterPGN",
		TargetDir:       outDir,
		SetMasterAsRef:  true,
		OpenAfterImport: true,
	})
	require.NoError(t, err)
	assert.Greater(t, res1.OriginalGames, 0)

	// Run again while active DB AND ref DB point to MasterPGN, and OrigPGN already exists on disk!
	require.NoError(t, app.openReferenceDatabaseInternal(res1.MasterBasePath))
	res2, err := app.ImportStandardDatabase(StandardImportRequest{
		SourcePath:      pgnFile,
		ImportMode:      "original_and_master",
		ArchiveName:     "OrigPGN",
		MasterDBName:    "MasterPGN",
		TargetDir:       outDir,
		SetMasterAsRef:  true,
		OpenAfterImport: true,
	})
	require.NoError(t, err)
	assert.NotNil(t, res2)

	// Missing .si4 source error
	_, err = app.ImportStandardDatabase(StandardImportRequest{
		SourcePath: filepath.Join(tmp, "missing.si4"),
		ImportMode: "original_only",
		TargetDir:  outDir,
	})
	assert.Error(t, err)

	// user_db with empty UserDBName and TargetDir using DefaultDatabasePath
	_, err = app.EnsurePersonalUserDatabase(false)
	require.NoError(t, err)
	resUserDef, err := app.ImportStandardDatabase(StandardImportRequest{
		PGNText:    covBlunderPGN,
		ImportMode: "user_db",
	})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, resUserDef.UserTotalGames, 1)

	// 5. UpdateDatabaseFromLocal: closed active DB -> uses DefaultDatabasePath, plus non-existent target error
	require.NoError(t, app.CloseDatabase())
	_, err = app.UpdateDatabaseFromLocal(filepath.Join(tmp, "no_such_target"), []string{res1.OriginalBasePath})
	assert.Error(t, err)
	_, err = app.UpdateDatabaseFromLocal("", []string{res1.OriginalBasePath})
	require.NoError(t, err)

	// 6. ConsolidateOnlineCatalogDatabases with non-empty sourceIDs (unknown ID fails fast with no network)
	cRes, err := app.ConsolidateOnlineCatalogDatabases([]string{"", "nonexistent-local-id"}, true, "CovOnlineCons", 0, 0, false)
	require.NoError(t, err)
	assert.NotNil(t, cRes)

	// 7. SaveUserProfileSetup edge branches: empty DisplayName/Title/Elo, explicit Focus fields, no open DB
	require.NoError(t, app.CloseDatabase())
	setup2, err := app.SaveUserProfileSetup(UserProfileSetupRequest{
		DisplayName:            "Second Hero",
		Title:                  "",
		EstimatedElo:           0,
		LichessUsername:        "CovHero",
		EnsureHomeDB:           true,
		FocusWhiteRepertoireID: "white-openings-italian",
		FocusBlackRepertoireID: "black-openings-sicilian-kid",
		FocusPolyglotBookPath:  filepath.Join(tmp, "focus.bin"),
	})
	require.NoError(t, err)
	assert.Equal(t, "Club Player", setup2.Title)
	assert.Equal(t, 1500, setup2.EstimatedElo)

	// 8. Portal auto-capture: CheckAndCaptureLivePortalGames full paths (in-flight, activeMatches, non-active, error, watcher tick)
	origInterval := portalPollInterval
	portalPollInterval = 60 * time.Millisecond
	defer func() {
		_, _ = app.StopLivePortalWatcher()
		portalPollInterval = origInterval
	}()

	// Test in-flight guard
	portalWatchMu.Lock()
	portalCheckInFlight = true
	portalWatchMu.Unlock()
	inFlightRes, err := app.CheckAndCaptureLivePortalGames()
	require.NoError(t, err)
	assert.NotNil(t, inFlightRes)
	portalWatchMu.Lock()
	portalCheckInFlight = false
	portalWatchMu.Unlock()

	// Configure usernames and capture into active DB (activeMatches = true)
	_, err = app.ConfigureLivePortalAutoCapture("CovHero", "CovHero", false, true)
	require.NoError(t, err)
	capRes1, err := app.CheckAndCaptureLivePortalGames()
	require.NoError(t, err)
	assert.NotNil(t, capRes1)

	// Second call -> 0 new games (all duplicates skipped)
	capRes2, err := app.CheckAndCaptureLivePortalGames()
	require.NoError(t, err)
	assert.Equal(t, 0, capRes2.NewGamesImported)

	// Close active DB and open a different DB so commitCapturedGamesToDatabase takes the non-matching DB path
	otherDB := createTempScidDB(t, tmp, "other_portal_db", covBlunderPGN)
	_, err = app.OpenDatabase(otherDB)
	require.NoError(t, err)
	_, err = app.CaptureOnlineGameFromURLOrPGN(covSamplePGN)
	require.NoError(t, err)

	// Close DB completely so noActiveDB == true path executes
	require.NoError(t, app.CloseDatabase())
	_, err = app.CaptureOnlineGameFromURLOrPGN(`[Event "Unique Portal Game"]
[Site "NewScid"]
[Date "2026.05.09"]
[Round "1"]
[White "UniqueW"]
[Black "UniqueB"]
[Result "1-0"]

1. c4 e5 2. Nc3 Nf6 3. g3 d5 1-0`)
	require.NoError(t, err)

	// Error branch in CheckAndCaptureLivePortalGames when both servers fail
	app.syncMgr.Lichess.BaseURL = "http://127.0.0.1:1"
	app.syncMgr.ChessCom.BaseURL = "http://127.0.0.1:1"
	errRes, err := app.CheckAndCaptureLivePortalGames()
	require.NoError(t, err)
	assert.NotNil(t, errRes)
	app.syncMgr.Lichess.BaseURL = srv.URL
	app.syncMgr.ChessCom.BaseURL = srv.URL

	// StartLivePortalWatcher and let initial timer (1.5s) fire once so goroutine body is covered
	_, err = app.ConfigureLivePortalAutoCapture("CovHero", "CovHero", true, false)
	require.NoError(t, err)
	time.Sleep(1650 * time.Millisecond)
	_, err = app.StopLivePortalWatcher()
	require.NoError(t, err)

	// 9. FactoryResetNewScid
	resetRes, err := app.FactoryResetNewScid(workstation.FactoryResetOptions{})
	require.NoError(t, err)
	assert.NotNil(t, resetRes)
}

func TestCovWorkspaceAndPortalFinalPush90(t *testing.T) {
	app, tmp := newIsolatedApp(t)
	srv := newMockOnlineServer(t, covSamplePGN)
	app.syncMgr.Lichess.BaseURL = srv.URL
	app.syncMgr.ChessCom.BaseURL = srv.URL

	// 1. GetAppPreferences normalization of zero/empty fields & SaveAppPreferences clearing refDB
	refDB := createTempScidDB(t, tmp, "prefs_ref_db", covSamplePGN)
	require.NoError(t, app.openReferenceDatabaseInternal(refDB))
	_, err := app.SaveAppPreferences(AppPreferences{
		ReferenceDatabasePath: "",
		PanelConfigs:          nil,
		DatabasePolicies:      nil,
	})
	require.NoError(t, err)

	zeroPrefsJSON := `{"compactLayoutMode":"","bottomDrawerHeight":0,"defaultMultiPV":0,"defaultThreads":0,"defaultHashMB":0,"annotateMoveTimeMs":0,"defaultPortalPlatform":"","portalWindowMode":"","panelConfigs":null,"databasePolicies":null}`
	require.NoError(t, os.WriteFile(xdgPreferencesFilePath(), []byte(zeroPrefsJSON), 0o644))
	normPrefs, err := app.GetAppPreferences()
	require.NoError(t, err)
	assert.Equal(t, "auto", normPrefs.CompactLayoutMode)
	assert.Equal(t, 210, normPrefs.BottomDrawerHeight)
	assert.Equal(t, 1, normPrefs.DefaultMultiPV)
	assert.Equal(t, 2, normPrefs.DefaultThreads)
	assert.Equal(t, 64, normPrefs.DefaultHashMB)
	assert.Equal(t, 150, normPrefs.AnnotateMoveTimeMs)
	assert.Equal(t, "lichess", normPrefs.DefaultPortalPlatform)
	assert.Equal(t, "browser-app", normPrefs.PortalWindowMode)

	// 2. Notable players: empty remove, double hide famous, re-add hidden with empty aliases/title, remove one of two customs
	_, err = app.RemoveNotablePlayer("   ")
	require.NoError(t, err)
	_, err = app.RemoveNotablePlayer("Bobby Fischer")
	require.NoError(t, err)
	_, err = app.RemoveNotablePlayer("Bobby Fischer") // alreadyHidden = true
	require.NoError(t, err)
	_, err = app.AddCustomNotablePlayer("Bobby Fischer", nil, "", "") // removes from hidden + default alias + default title
	require.NoError(t, err)
	_, err = app.AddCustomNotablePlayer("Second Custom", []string{"SC"}, "IM", "2020s")
	require.NoError(t, err)
	_, err = app.RemoveNotablePlayer("Bobby Fischer") // leaves Second Custom in remaining
	require.NoError(t, err)

	// 3. CancelActiveOperation & GetActiveOperation when engine-match is running
	app.mu.Lock()
	app.matchRunning = true
	app.matchCancel = func() {}
	app.mu.Unlock()
	assert.Equal(t, "engine-match", app.GetActiveOperation())
	assert.True(t, app.CancelActiveOperation())
	assert.Equal(t, "", app.GetActiveOperation())

	// 4. LaunchPanelOnWorkspace with workspace != "" (spawns goroutine) & LaunchPortalWindow fallback with empty PATH
	t.Setenv("PATH", tmp)
	require.NoError(t, app.LaunchPanelOnWorkspace("info", "2", "", chess.StartingFEN, ""))
	prefsMu.Lock()
	delete(detachedPanelPIDs, "info")
	prefsMu.Unlock()
	require.NoError(t, app.LaunchPortalWindow("lichess", "https://lichess.org", "browser-app"))

	// 5. UpdateDatabaseFromLocal auto-discover with managed DBs (editable + locked)
	managedDir := filepath.Join(tmp, ".local", "share", fileformat.AppDirName, fileformat.DirDatabases)
	require.NoError(t, os.MkdirAll(managedDir, 0o755))
	edDB := createTempScidDB(t, managedDir, "auto_editable", covBlunderPGN)
	lkDB := createTempScidDB(t, managedDir, "auto_locked", covSamplePGN)
	tgDB := createTempScidDB(t, managedDir, "auto_target", covSamplePGN)
	_, err = app.SetDatabaseWritePolicy(lkDB, "locked")
	require.NoError(t, err)
	_, err = app.SetDatabaseWritePolicy(edDB, "editable")
	require.NoError(t, err)
	_, err = app.SetDatabaseWritePolicy(tgDB, "editable")
	require.NoError(t, err)
	mergeAuto, err := app.UpdateDatabaseFromLocal(tgDB, nil)
	require.NoError(t, err)
	assert.NotNil(t, mergeAuto)

	// 6. BuildConsolidatedDatabase refreshing open a.db and a.refDB when they match TargetBasePath
	consTarget := filepath.Join(managedDir, "RefreshedCons")
	_ = createTempScidDB(t, managedDir, "RefreshedCons", covSamplePGN)
	_, err = app.OpenDatabase(consTarget)
	require.NoError(t, err)
	require.NoError(t, app.openReferenceDatabaseInternal(consTarget))
	_, err = app.BuildConsolidatedDatabase(scid.ConsolidateDatabaseOptions{
		TargetName:      "RefreshedCons",
		SourceBasePaths: []string{edDB},
	})
	require.NoError(t, err)

	// 7. SaveUserProfileSetup with empty DisplayName -> falls back to lichess then chesscom for sync profile name
	_, err = app.SaveUserProfileSetup(UserProfileSetupRequest{
		DisplayName:     "",
		LichessUsername: "only_li_user",
	})
	require.NoError(t, err)
	_, err = app.SaveUserProfileSetup(UserProfileSetupRequest{
		DisplayName:      "",
		LichessUsername:  "",
		ChessComUsername: "only_cc_user",
	})
	require.NoError(t, err)

	// 8. Portal auto-capture: resolveAutoCaptureTargetPath (active editable DB, then My_Games fallback),
	//    resolvePortalCredentials from profileMgr, CaptureOnlineGameFromURLOrPGN nil syncMgr + single-line PGN + fetch errors
	p, err := app.GetAppPreferences()
	require.NoError(t, err)
	p.DefaultDatabasePath = ""
	p.LichessUsername = ""
	p.ChessComUsername = ""
	_, err = app.SaveAppPreferences(*p)
	require.NoError(t, err)

	_, err = app.OpenDatabase(edDB)
	require.NoError(t, err)
	assert.Equal(t, edDB, app.resolveAutoCaptureTargetPath())

	require.NoError(t, app.CloseDatabase())
	assert.Contains(t, app.resolveAutoCaptureTargetPath(), fileformat.DefaultMyGamesDBName)

	// CaptureOnlineGameFromURLOrPGN with bare move text ("1. e4 e5") and nil syncMgr errors
	_, err = app.CaptureOnlineGameFromURLOrPGN("1. e4 e5 2. Nf3 Nc6 3. d4 exd4 *")
	require.NoError(t, err)

	savedSync := app.syncMgr
	app.syncMgr = nil
	_, err = app.CaptureOnlineGameFromURLOrPGN("https://www.chess.com/game/live/123456")
	assert.Error(t, err)
	_, err = app.CaptureOnlineGameFromURLOrPGN("https://lichess.org/cov12345")
	assert.Error(t, err)
	app.syncMgr = savedSync

	// Fetch errors on unreachable URL
	app.syncMgr.Lichess.BaseURL = "http://127.0.0.1:1"
	_, err = app.CaptureOnlineGameFromURLOrPGN("https://lichess.org/cov12345")
	assert.Error(t, err)
	app.syncMgr.Lichess.BaseURL = srv.URL

	// 9. StartLivePortalWatcher auto-stop branch when AutoCapturePortalGames becomes false
	origInterval := portalPollInterval
	portalPollInterval = 60 * time.Millisecond
	defer func() {
		_, _ = app.StopLivePortalWatcher()
		portalPollInterval = origInterval
	}()
	_, err = app.StartLivePortalWatcher()
	require.NoError(t, err)
	p2, _ := app.GetAppPreferences()
	p2.AutoCapturePortalGames = false
	_, _ = app.SaveAppPreferences(*p2)
	time.Sleep(1650 * time.Millisecond)

	// 10. FactoryResetNewScid while both active DB and refDB are open
	_, err = app.OpenDatabase(edDB)
	require.NoError(t, err)
	require.NoError(t, app.openReferenceDatabaseInternal(tgDB))
	_, err = app.FactoryResetNewScid(workstation.FactoryResetOptions{})
	require.NoError(t, err)
}

func TestCovWorkspaceOver90Surgical(t *testing.T) {
	app, tmp := newIsolatedApp(t)

	// 1. ImportStandardDatabase edge paths: user_db with .si4 source + empty UserDBName/DefaultDatabasePath,
	//    missing PGN file, and invalid TargetDir
	srcDB := createTempScidDB(t, tmp, "std_src_si4", covSamplePGN)
	resUserSI4, err := app.ImportStandardDatabase(StandardImportRequest{
		SourcePath: srcDB + ".si4",
		ImportMode: "user_db",
	})
	require.NoError(t, err)
	assert.Greater(t, resUserSI4.UserTotalGames, 0)

	_, err = app.ImportStandardDatabase(StandardImportRequest{
		SourcePath: filepath.Join(tmp, "missing_file.pgn"),
		ImportMode: "original_only",
	})
	assert.Error(t, err)

	blockingFile := filepath.Join(tmp, "not_a_dir")
	require.NoError(t, os.WriteFile(blockingFile, []byte("x"), 0o644))
	_, err = app.ImportStandardDatabase(StandardImportRequest{
		PGNText:    covSamplePGN,
		ImportMode: "user_db",
		TargetDir:  blockingFile,
	})
	assert.Error(t, err)

	// 2. SaveUserProfileSetup with empty DefaultDatabasePath + EnsureHomeDB=true, and matching existing persona by DisplayName
	_, err = app.SavePersona(&persona.Persona{
		DisplayName: "Matched Profile",
	})
	require.NoError(t, err)
	p, _ := app.GetAppPreferences()
	p.DefaultDatabasePath = ""
	p.PlayerPersonaID = ""
	p.DatabasePolicies = nil
	_, _ = app.SaveAppPreferences(*p)

	resProf, err := app.SaveUserProfileSetup(UserProfileSetupRequest{
		DisplayName:  "Matched Profile",
		EnsureHomeDB: true,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, resProf.PersonaID)

	// 3. BuildConsolidatedDatabase & ConsolidateOnlineCatalogDatabases with Clipbase open + pre-existing stage DB
	_, err = app.OpenClipbase()
	require.NoError(t, err)
	_, err = app.BuildConsolidatedDatabase(scid.ConsolidateDatabaseOptions{
		TargetName: "CovWithClipbase",
	})
	require.NoError(t, err)

	managedDir := filepath.Join(tmp, ".local", "share", fileformat.AppDirName, fileformat.DirDatabases)
	_ = createTempScidDB(t, managedDir, "online_stage_preexisting", covSamplePGN)
	_, err = app.ConsolidateOnlineCatalogDatabases([]string{"preexisting"}, true, "CovWithPreStage", 10, 0, false)
	require.NoError(t, err)

	// 4. xdgPreferencesFilePath / preferencesFilePath / workspaceSyncFilePath / clipbaseDefaultPath fallbacks when HOME/XDG_CONFIG_HOME empty
	t.Run("EmptyEnvPathFallbacks", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "")
		t.Setenv("HOME", "")
		assert.NotEmpty(t, preferencesFilePath())
		assert.NotEmpty(t, xdgPreferencesFilePath())
		assert.NotEmpty(t, workspaceSyncFilePath())
		assert.NotEmpty(t, clipbaseDefaultPath())
	})
}
