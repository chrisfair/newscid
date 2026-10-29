package main

import (
	"path/filepath"
	"testing"

	"newscid/pkg/chess"
	"newscid/pkg/database/scid"
	"newscid/pkg/persona"
	"newscid/pkg/workstation"
)

func TestWorkspacePreferencesAndDatabaseRoles(t *testing.T) {
	fakeHome := t.TempDir()
	t.Setenv("HOME", fakeHome)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(fakeHome, ".config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(fakeHome, ".local", "share"))

	app := NewApp()
	defer app.shutdown(nil)

	tmpDir := t.TempDir()
	db1Path := filepath.Join(tmpDir, "MyDefaultBase")
	db2Path := filepath.Join(tmpDir, "MyReferenceMega")

	db1 := scid.NewDatabase()
	if err := db1.Create(db1Path, "Default DB"); err != nil {
		t.Fatalf("failed to create db1: %v", err)
	}
	_ = db1.Close()

	db2 := scid.NewDatabase()
	if err := db2.Create(db2Path, "Reference DB"); err != nil {
		t.Fatalf("failed to create db2: %v", err)
	}
	_ = db2.Close()

	status, err := app.SetDefaultDatabase(db1Path + ".si4")
	if err != nil {
		t.Fatalf("SetDefaultDatabase failed: %v", err)
	}
	if status.DefaultDBName != "MyDefaultBase" {
		t.Errorf("expected DefaultDBName=MyDefaultBase, got %q", status.DefaultDBName)
	}

	info, err := app.OpenDefaultDatabase()
	if err != nil {
		t.Fatalf("OpenDefaultDatabase failed: %v", err)
	}
	if info.Name != "MyDefaultBase" {
		t.Errorf("expected active DB name MyDefaultBase, got %q", info.Name)
	}

	status, err = app.SetReferenceDatabase(db2Path, true)
	if err != nil {
		t.Fatalf("SetReferenceDatabase failed: %v", err)
	}
	if !status.IsActiveDefault {
		t.Errorf("expected IsActiveDefault=true when Default DB is open")
	}
	if status.ReferenceDBName != "MyReferenceMega" || !status.UseRefForTree {
		t.Errorf("unexpected reference DB status: %+v", status)
	}

	// Test shared workspace state sync
	testFEN := "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq - 0 1"
	err = app.PublishWorkspaceState(SharedWorkspaceState{
		SourceID:     "test-main",
		FEN:          testFEN,
		LastMoveFrom: "e2",
		LastMoveTo:   "e4",
		CurrentPly:   1,
		DBPath:       db1Path,
		DBName:       "MyDefaultBase",
	})
	if err != nil {
		t.Fatalf("PublishWorkspaceState failed: %v", err)
	}

	shared, err := app.GetSharedWorkspaceState()
	if err != nil || shared == nil {
		t.Fatalf("GetSharedWorkspaceState failed: %v", err)
	}
	if shared.FEN != testFEN || shared.CurrentPly != 1 {
		t.Errorf("unexpected shared workspace state: %+v", shared)
	}
}

func TestDatabaseLockingNotablePlayersAndWorkstationReset(t *testing.T) {
	fakeHome := t.TempDir()
	t.Setenv("HOME", fakeHome)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(fakeHome, ".config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(fakeHome, ".local", "share"))

	app := NewApp()
	defer app.shutdown(nil)

	dbPath := filepath.Join(t.TempDir(), "TestGamesDB")
	_, err := app.CreateDatabase(dbPath, "Personal Base")
	if err != nil {
		t.Fatalf("CreateDatabase failed: %v", err)
	}

	g, err := chess.ParsePGN(`[Event "World Championship"]
[White "Fischer, Robert James"]
[Black "Spassky, Boris V"]
[Result "1-0"]
[ECO "B90"]

1. e4 c5 2. Nf3 d6 1-0`)
	if err != nil {
		t.Fatalf("ParsePGN failed: %v", err)
	}
	if _, err := app.SaveGame(g); err != nil {
		t.Fatalf("SaveGame on editable DB failed: %v", err)
	}

	// Lock the database and verify SaveGame is blocked
	lockedInfo, err := app.SetDatabaseWritePolicy(dbPath, "locked")
	if err != nil || lockedInfo == nil || !lockedInfo.IsLocked || lockedInfo.WritePolicy != "locked" {
		t.Fatalf("SetDatabaseWritePolicy(locked) failed: %+v, err=%v", lockedInfo, err)
	}
	if _, err := app.SaveGame(g); err == nil {
		t.Fatalf("expected SaveGame to fail when database is locked")
	}

	// Set to sync_only: manual SaveGame blocked, online sync allowed
	syncInfo, err := app.SetDatabaseWritePolicy(dbPath, "sync_only")
	if err != nil || syncInfo == nil || syncInfo.WritePolicy != "sync_only" {
		t.Fatalf("SetDatabaseWritePolicy(sync_only) failed: %+v, err=%v", syncInfo, err)
	}
	if _, err := app.SaveGame(g); err == nil {
		t.Fatalf("expected manual SaveGame to fail when database is sync_only")
	}

	// Unlock back to editable
	_, err = app.SetDatabaseWritePolicy(dbPath, "editable")
	if err != nil {
		t.Fatalf("SetDatabaseWritePolicy(editable) failed: %v", err)
	}

	// Famous & custom notable players discovery
	notables, err := app.GetNotablePlayersDropdown()
	if err != nil || len(notables) == 0 {
		t.Fatalf("GetNotablePlayersDropdown failed: %v", err)
	}
	if notables[0].GameCount == 0 {
		t.Errorf("expected Bobby Fischer / Boris Spassky to have GameCount > 0, got %+v", notables[0])
	}

	notables, err = app.AddCustomNotablePlayer("Morphy, Paul", []string{"Paul Morphy"}, "Legend", "1857-1859")
	if err != nil {
		t.Fatalf("AddCustomNotablePlayer failed: %v", err)
	}
	notables, err = app.RemoveNotablePlayer("Morphy, Paul")
	if err != nil {
		t.Fatalf("RemoveNotablePlayer failed: %v", err)
	}
	_, err = app.ResetNotablePlayersToDefault()
	if err != nil {
		t.Fatalf("ResetNotablePlayersToDefault failed: %v", err)
	}

	// Build focused sub-database from active database
	subRes, err := app.BuildFocusedDatabase(scid.SubDatabaseFilter{
		TargetName:   "Fischer_Najdorf",
		Players:      []string{"Bobby Fischer"},
		ECOFilter:    "B90",
		OpeningMoves: []string{"1.e4", "c5", "2.Nf3"},
		Deduplicate:  true,
	})
	if err != nil || subRes == nil || subRes.GamesMatched != 1 {
		t.Fatalf("BuildFocusedDatabase failed: res=%+v, err=%v", subRes, err)
	}

	// Workstation storage report & asset copy
	report, err := app.GetWorkstationStorageReport()
	if err != nil || report == nil || len(report.Categories) == 0 {
		t.Fatalf("GetWorkstationStorageReport failed: %v", err)
	}

	// Factory reset with opt-in backup
	resetRes, err := app.FactoryResetNewScid(workstation.FactoryResetOptions{
		CreateOptInBackup: true,
	})
	if err != nil || resetRes == nil || !resetRes.BackupCreated || resetRes.BackupArchivePath == "" {
		t.Fatalf("FactoryResetNewScid failed: res=%+v, err=%v", resetRes, err)
	}
}

func TestBuildConsolidatedDatabaseAndPersonaDatabaseDiscovery(t *testing.T) {
	fakeHome := t.TempDir()
	t.Setenv("HOME", fakeHome)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(fakeHome, ".config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(fakeHome, ".local", "share"))

	app := NewApp()
	defer app.shutdown(nil)

	// Save a Persona and verify it automatically gets a dedicated .si4 database in GetDiscoveredDatabases
	savedPersona, err := app.SavePersona(&persona.Persona{
		DisplayName:      "Magnus Study Persona",
		Identifiers:      []string{"Carlsen, Magnus", "DrNykterstein"},
		LichessUsername:  "DrNykterstein",
		ChessComUsername: "MagnusCarlsen",
	})
	if err != nil || savedPersona == nil || savedPersona.DatabasePath == "" {
		t.Fatalf("SavePersona failed or did not assign DatabasePath: %+v, err=%v", savedPersona, err)
	}

	discovered := app.GetDiscoveredDatabases()
	foundPersonaDB := false
	for _, d := range discovered {
		if d.Category == "Persona Databases" {
			foundPersonaDB = true
			break
		}
	}
	if !foundPersonaDB {
		t.Fatalf("expected GetDiscoveredDatabases to include Persona Databases category, got %+v", discovered)
	}

	// Create two source online databases with duplicate games (by online ID and by identical moves)
	tmpDir := t.TempDir()
	src1 := filepath.Join(tmpDir, "Lichess_Broadcasts")
	src2 := filepath.Join(tmpDir, "Lichess_Elite")

	db1 := scid.NewDatabase()
	if err := db1.Create(src1, "Broadcasts"); err != nil {
		t.Fatalf("create src1: %v", err)
	}
	g1, _ := chess.ParsePGN(`[Event "Titled Arena"]
[Site "https://lichess.org/AbCdEf12"]
[White "Carlsen, Magnus"]
[Black "Nakamura, Hikaru"]
[Result "1-0"]

1. e4 e5 2. Nf3 Nc6 3. Bb5 a6 4. Ba4 Nf6 5. O-O Be7 1-0`)
	_, _ = db1.SaveGame(g1)
	_ = db1.Close()

	db2 := scid.NewDatabase()
	if err := db2.Create(src2, "Elite"); err != nil {
		t.Fatalf("create src2: %v", err)
	}
	// Duplicate 1: Same online ID AbCdEf12
	g1Dup, _ := chess.ParsePGN(`[Event "Lichess Elite"]
[Site "https://lichess.org/AbCdEf12/black"]
[White "DrNykterstein"]
[Black "Hikaru"]
[Result "1-0"]

1. e4 e5 2. Nf3 Nc6 3. Bb5 a6 4. Ba4 Nf6 5. O-O Be7 1-0`)
	// Duplicate 2: Different site tag, but identical 10-ply move sequence & players
	g1MoveDup, _ := chess.ParsePGN(`[Event "OTB Mirror"]
[Site "Oslo"]
[White "Magnus Carlsen"]
[Black "Hikaru Nakamura"]
[Result "1-0"]

1. e4 e5 2. Nf3 Nc6 3. Bb5 a6 4. Ba4 Nf6 5. O-O Be7 1-0`)
	// Unique game 2
	g2Unique, _ := chess.ParsePGN(`[Event "Candidates"]
[Site "https://www.chess.com/game/live/9988776655"]
[White "Caruana, Fabiano"]
[Black "Gukesh, D"]
[Result "1/2-1/2"]

1. d4 Nf6 2. c4 e6 3. Nc3 Bb4 4. e3 O-O 5. Bd3 d5 1/2-1/2`)
	_, _ = db2.SaveGame(g1Dup)
	_, _ = db2.SaveGame(g1MoveDup)
	_, _ = db2.SaveGame(g2Unique)
	_ = db2.Close()

	// Build Consolidated Database and set it as Default Reference Database
	consRes, err := app.BuildConsolidatedDatabase(scid.ConsolidateDatabaseOptions{
		TargetName:      "Consolidated_Online_Master",
		SourceBasePaths: []string{src1, src2},
		SetAsReference:  true,
	})
	if err != nil {
		t.Fatalf("BuildConsolidatedDatabase failed: %v", err)
	}
	if consRes.UniqueGamesSaved != 2 || consRes.DuplicatesByID != 1 || consRes.DuplicatesByContent != 1 {
		t.Fatalf("unexpected consolidation stats: %+v", consRes)
	}

	roles, err := app.GetDatabaseRolesStatus()
	if err != nil {
		t.Fatalf("GetDatabaseRolesStatus failed: %v", err)
	}
	if roles.ReferenceDBName != "Consolidated_Online_Master" {
		t.Errorf("expected ReferenceDBName=Consolidated_Online_Master, got %q", roles.ReferenceDBName)
	}
	if pol := app.GetDatabaseWritePolicy(consRes.TargetBasePath); pol != "sync_only" {
		t.Errorf("expected consolidated database policy=sync_only, got %q", pol)
	}

	// Verify GetLatestMegaDatabaseCatalog and CancelActiveOperation / GetActiveOperation
	latestCat := app.GetLatestMegaDatabaseCatalog()
	if len(latestCat) == 0 {
		t.Fatalf("expected non-empty latest mega database catalog")
	}
	for _, s := range latestCat {
		if !s.IsLatest && s.Year < 2026 {
			t.Errorf("expected only 2026/latest sources in GetLatestMegaDatabaseCatalog, got %+v", s)
		}
	}

	if op := app.GetActiveOperation(); op != "" {
		t.Errorf("expected idle active operation, got %q", op)
	}
	if app.CancelActiveOperation() {
		t.Errorf("expected CancelActiveOperation to return false when idle")
	}

	opCtx, finish := app.startCancellableOperation("test-import")
	if op := app.GetActiveOperation(); op != "test-import" {
		t.Errorf("expected active operation test-import, got %q", op)
	}
	if !app.CancelActiveOperation() {
		t.Errorf("expected CancelActiveOperation to return true when active")
	}
	if opCtx.Err() == nil {
		t.Errorf("expected opCtx to be cancelled")
	}
	finish()
}

func TestStandardDatabaseImportThreeTierModel(t *testing.T) {
	fakeHome := t.TempDir()
	t.Setenv("HOME", fakeHome)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(fakeHome, ".config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(fakeHome, ".local", "share"))

	app := NewApp()
	defer app.shutdown(nil)

	// EnsurePersonalUserDatabase creates My_Games (Editable) and sets it as Default User DB
	userInfo, err := app.EnsurePersonalUserDatabase(true)
	if err != nil || userInfo == nil {
		t.Fatalf("EnsurePersonalUserDatabase failed: %v", err)
	}
	if userInfo.Name != "My_Games" || userInfo.WritePolicy != "editable" || userInfo.IsLocked {
		t.Fatalf("unexpected user database info: %+v", userInfo)
	}

	// Import an external PGN archive using "original_and_master" (Standard Reference Import)
	samplePGN := `[Event "Candidates 2026"]
[Site "Toronto"]
[Date "2026.04.10"]
[Round "1"]
[White "Caruana, Fabiano"]
[Black "Nakamura, Hikaru"]
[Result "1/2-1/2"]

1. e4 e5 2. Nf3 Nc6 3. Bb5 a6 1/2-1/2

[Event "Candidates 2026"]
[Site "Toronto"]
[Date "2026.04.11"]
[Round "2"]
[White "Gukesh, D"]
[Black "Firouzja, Alireza"]
[Result "1-0"]

1. d4 Nf6 2. c4 g6 3. Nc3 Bg7 1-0
`
	res1, err := app.ImportStandardDatabase(StandardImportRequest{
		PGNText:         samplePGN,
		ArchiveName:     "Candidates_2026_Original",
		ImportMode:      "original_and_master",
		MasterDBName:    "Master_Reference",
		SetMasterAsRef:  true,
		OpenAfterImport: false,
	})
	if err != nil || res1 == nil {
		t.Fatalf("ImportStandardDatabase(original_and_master) failed: %v", err)
	}
	if res1.OriginalName != "Candidates_2026_Original" || res1.OriginalGames != 2 {
		t.Fatalf("unexpected original archive stats: %+v", res1)
	}
	if res1.MasterName != "Master_Reference" || res1.MasterAddedGames != 2 || res1.MasterTotalGames != 2 {
		t.Fatalf("unexpected master reference stats: %+v", res1)
	}
	// Verify Original Archive is locked and Master Reference is sync_only
	if pol := app.GetDatabaseWritePolicy(res1.OriginalBasePath); pol != "locked" {
		t.Errorf("expected Original Archive policy=locked, got %q", pol)
	}
	if pol := app.GetDatabaseWritePolicy(res1.MasterBasePath); pol != "sync_only" {
		t.Errorf("expected Master Reference policy=sync_only, got %q", pol)
	}

	// Re-importing the same games into Master should skip 2 duplicates
	resDup, err := app.ImportStandardDatabase(StandardImportRequest{
		PGNText:      samplePGN,
		ImportMode:   "master_only",
		MasterDBName: "Master_Reference",
	})
	if err != nil || resDup.MasterAddedGames != 0 || resDup.MasterSkippedDupes != 2 {
		t.Fatalf("expected 0 added and 2 skipped duplicates on re-import, got %+v (err=%v)", resDup, err)
	}

	// Import personal games into User DB ("user_db")
	myGamePGN := `[Event "Club Championship"]
[Site "Local Club"]
[Date "2026.05.01"]
[Round "1"]
[White "Christopher"]
[Black "Opponent"]
[Result "1-0"]

1. e4 c5 2. Nf3 d6 3. d4 cxd4 4. Nxd4 Nf6 5. Nc3 a6 1-0
`
	resUser, err := app.ImportStandardDatabase(StandardImportRequest{
		PGNText:          myGamePGN,
		ImportMode:       "user_db",
		SetUserAsDefault: true,
		OpenAfterImport:  true,
	})
	if err != nil || resUser == nil {
		t.Fatalf("ImportStandardDatabase(user_db) failed: %v", err)
	}
	if resUser.UserName != "My_Games" || resUser.UserAddedGames != 1 || resUser.UserTotalGames != 1 {
		t.Fatalf("unexpected user_db import stats: %+v", resUser)
	}
	if pol := app.GetDatabaseWritePolicy(resUser.UserBasePath); pol != "editable" {
		t.Errorf("expected User DB policy=editable, got %q", pol)
	}
}

func TestUserProfileSetupWizardAndPersonaBooks(t *testing.T) {
	fakeHome := t.TempDir()
	t.Setenv("HOME", fakeHome)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(fakeHome, ".config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(fakeHome, ".local", "share"))

	app := NewApp()
	defer app.shutdown(nil)

	// Initial status should have OnboardingCompleted = false
	initStatus, err := app.GetUserProfileStatus()
	if err != nil || initStatus == nil {
		t.Fatalf("GetUserProfileStatus failed: %v", err)
	}
	if initStatus.OnboardingCompleted {
		t.Errorf("expected initial OnboardingCompleted=false")
	}

	// Compile starter White repertoire into a Polyglot .bin book
	repBookPath, entryCount, err := app.CompileRepertoireToPolyglotBook("white-openings-italian", true)
	if err != nil || entryCount == 0 || repBookPath == "" {
		t.Fatalf("CompileRepertoireToPolyglotBook failed: path=%q count=%d err=%v", repBookPath, entryCount, err)
	}

	// Save User Profile via wizard setup RPC
	setupRes, err := app.SaveUserProfileSetup(UserProfileSetupRequest{
		DisplayName:            "Christopher Fair",
		Title:                  "Club Player",
		EstimatedElo:           1750,
		LichessUsername:        "chrisfair",
		ChessComUsername:       "chrisfair_chess",
		FideID:                 "12345678",
		Aliases:                []string{"C. Fair"},
		HomeDatabaseName:       "My_Games",
		EnsureHomeDB:           true,
		FocusWhiteRepertoireID: "white-openings-italian",
		FocusBlackRepertoireID: "black-openings-sicilian-kid",
		FocusPolyglotBookPath:  repBookPath,
		CompleteOnboarding:     true,
	})
	if err != nil || setupRes == nil {
		t.Fatalf("SaveUserProfileSetup failed: %v", err)
	}
	if !setupRes.OnboardingCompleted || setupRes.DisplayName != "Christopher Fair" || setupRes.HomeDBName != "My_Games" {
		t.Fatalf("unexpected setup result: %+v", setupRes)
	}
	if setupRes.PersonaID == "" || setupRes.SyncProfileID == "" {
		t.Fatalf("expected PersonaID and SyncProfileID to be created, got %+v", setupRes)
	}
	// Verify auto-generated "Last, First" alias was included
	foundLastFirst := false
	for _, al := range setupRes.Aliases {
		if al == "Fair, Christopher" {
			foundLastFirst = true
			break
		}
	}
	if !foundLastFirst {
		t.Errorf("expected 'Fair, Christopher' in auto-generated aliases, got %v", setupRes.Aliases)
	}

	// Add a game for this user into My_Games and compile their Persona Opening Book (.bin)
	myGame, err := chess.ParsePGN(`[Event "Rapid Match"]
[Site "Lichess"]
[Date "2026.10.20"]
[White "Fair, Christopher"]
[Black "RivalPlayer"]
[Result "1-0"]

1. e4 e5 2. Nf3 Nc6 3. Bc4 Bc5 4. c3 Nf6 5. d4 exd4 1-0`)
	if err != nil {
		t.Fatalf("ParsePGN failed: %v", err)
	}
	if _, err := app.SaveGame(myGame); err != nil {
		t.Fatalf("SaveGame to Home DB failed: %v", err)
	}

	personaBookPath, pBookCount, err := app.BuildPersonaOpeningBook(setupRes.PersonaID, "both", 12)
	if err != nil || pBookCount == 0 || personaBookPath == "" {
		t.Fatalf("BuildPersonaOpeningBook failed: path=%q count=%d err=%v", personaBookPath, pBookCount, err)
	}

	// Create a brand new opponent Persona with 0 games in DB, assign the compiled book & repertoire,
	// and verify GetPersonaSimulatedMove plays from the assigned book/repertoire immediately!
	oppPersona, err := app.SavePersona(&persona.Persona{
		DisplayName:  "Club Rival Bot",
		EstimatedElo: 1650,
	})
	if err != nil || oppPersona == nil {
		t.Fatalf("SavePersona(opponent) failed: %v", err)
	}
	oppPersona, err = app.AssignPersonaBooks(oppPersona.ID, personaBookPath, "", "white-openings-italian", "black-openings-sicilian-kid")
	if err != nil || oppPersona.BookPath != personaBookPath {
		t.Fatalf("AssignPersonaBooks failed: %+v, err=%v", oppPersona, err)
	}

	simMove, err := app.GetPersonaSimulatedMove(oppPersona.ID, chess.StartingFEN, true)
	if err != nil || simMove == nil || !simMove.IsBook || simMove.SAN != "e4" {
		t.Fatalf("expected simulated move e4 from assigned repertoire/book, got %+v, err=%v", simMove, err)
	}
}




