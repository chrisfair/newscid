package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func TestAppFileDialogWrappersAndWorkstationImport(t *testing.T) {
	origOpen := openFileDialogFunc
	origSave := saveFileDialogFunc
	defer func() {
		openFileDialogFunc = origOpen
		saveFileDialogFunc = origSave
	}()

	var lastOpenTitle, lastSaveTitle, lastDefaultName string
	openFileDialogFunc = func(ctx context.Context, opts runtime.OpenDialogOptions) (string, error) {
		lastOpenTitle = opts.Title
		return "/tmp/selected.pgn", nil
	}
	saveFileDialogFunc = func(ctx context.Context, opts runtime.SaveDialogOptions) (string, error) {
		lastSaveTitle = opts.Title
		lastDefaultName = opts.DefaultFilename
		return "/tmp/" + opts.DefaultFilename, nil
	}

	app := NewApp()
	app.ctx = context.Background()

	p, err := app.OpenPGNFileDialog()
	require.NoError(t, err)
	assert.Equal(t, "/tmp/selected.pgn", p)
	assert.Contains(t, lastOpenTitle, "PGN")

	p, err = app.OpenFileDialog()
	require.NoError(t, err)
	assert.Equal(t, "/tmp/selected.pgn", p)
	assert.Contains(t, lastOpenTitle, "Database")

	p, err = app.SaveFileDialog("games.si4")
	require.NoError(t, err)
	assert.Equal(t, "/tmp/games.si4", p)
	assert.Equal(t, "games.si4", lastDefaultName)
	assert.Contains(t, lastSaveTitle, "Database")

	p, err = app.SaveBookFileDialog("openings.bin")
	require.NoError(t, err)
	assert.Equal(t, "/tmp/openings.bin", p)
	assert.Contains(t, lastSaveTitle, "Book")

	p, err = app.SavePGNFileDialog("rep.pgn")
	require.NoError(t, err)
	assert.Equal(t, "/tmp/rep.pgn", p)
	assert.Contains(t, lastSaveTitle, "PGN")

	// SelectAndImportWorkstationAsset with nil ctx
	appNoCtx := NewApp()
	_, err = appNoCtx.SelectAndImportWorkstationAsset("books")
	assert.Error(t, err)

	// SelectAndImportWorkstationAsset with empty selection
	openFileDialogFunc = func(ctx context.Context, opts runtime.OpenDialogOptions) (string, error) {
		return "", nil
	}
	res, err := app.SelectAndImportWorkstationAsset("databases")
	require.NoError(t, err)
	assert.Nil(t, res)

	// SelectAndImportWorkstationAsset with real temp file
	tmpDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", filepath.Join(tmpDir, "data"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, "config"))
	srcFile := filepath.Join(tmpDir, "sample.pgn")
	require.NoError(t, os.WriteFile(srcFile, []byte("[Event \"T\"]\n\n1. e4 e5 *\n"), 0o644))
	openFileDialogFunc = func(ctx context.Context, opts runtime.OpenDialogOptions) (string, error) {
		return srcFile, nil
	}
	imported, err := app.SelectAndImportWorkstationAsset("databases")
	require.NoError(t, err)
	require.NotNil(t, imported)
}

func TestAppRemoteSupportAndFairPlayTelemetryStubs(t *testing.T) {
	app := NewApp()
	app.ctx = context.Background()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	// Remote endpoints with empty/offline arguments should fail gracefully or return fallback intel
	_, _ = app.SyncRemoteSupportStatus("")
	intel := app.FetchFairPlayCommunityIntel("lichess", "", 12, false)
	assert.Equal(t, "lichess", intel.Platform)

	_, err := app.SubmitFairPlayCommunityFlag("lichess", "", 50, false, "")
	assert.Error(t, err)

	_, _ = app.FetchLichessDatabaseFeed("standard", 1)
}
