package main

import (
	"path/filepath"
	"testing"

	"newscid/pkg/database/scid"
)

func TestWorkspacePreferencesAndDatabaseRoles(t *testing.T) {
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

	// Save original preferences so we don't clobber user config
	origPrefs, _ := app.GetAppPreferences()
	if origPrefs != nil {
		defer func() {
			_, _ = app.SaveAppPreferences(*origPrefs)
		}()
	}

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
