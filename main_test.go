package main

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wailsapp/wails/v2/pkg/options"
)

func TestRunMainDispatchBranches(t *testing.T) {
	origTUI := runTUIFunc
	origSrv := runServerFunc
	origWails := wailsRunFunc
	origCfg := windowStartupCfg
	defer func() {
		runTUIFunc = origTUI
		runServerFunc = origSrv
		wailsRunFunc = origWails
		windowStartupCfg = origCfg
	}()

	var tuiGot []string
	runTUIFunc = func(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
		tuiGot = append([]string(nil), args...)
		return 0
	}

	var srvHost, srvData string
	var srvPort int
	var srvErr error
	runServerFunc = func(app *App, host string, port int, dataDir string) error {
		srvHost, srvPort, srvData = host, port, dataDir
		return srvErr
	}

	var wailsOpts *options.App
	var wailsErr error
	wailsRunFunc = func(opts *options.App) error {
		wailsOpts = opts
		return wailsErr
	}

	var out, errBuf bytes.Buffer

	// --version
	assert.Equal(t, 0, runMain([]string{"newscid", "--version"}, "linux", strings.NewReader(""), &out, &errBuf))
	assert.NotEmpty(t, out.String())

	// --help
	out.Reset()
	assert.Equal(t, 0, runMain([]string{"newscid", "--help"}, "linux", strings.NewReader(""), &out, &errBuf))
	assert.Contains(t, out.String(), "NewScid")

	// tui subcommand & flags anywhere
	assert.Equal(t, 0, runMain([]string{"newscid", "tui", "games.si4"}, "linux", strings.NewReader(""), &out, &errBuf))
	assert.Equal(t, []string{"games.si4"}, tuiGot)

	assert.Equal(t, 0, runMain([]string{"newscid", "games.si4", "--tui"}, "linux", strings.NewReader(""), &out, &errBuf))
	assert.Equal(t, []string{"games.si4"}, tuiGot)

	assert.Equal(t, 0, runMain([]string{"newscid", "games.si4", "--kitty"}, "linux", strings.NewReader(""), &out, &errBuf))
	assert.Equal(t, []string{"--kitty", "games.si4"}, tuiGot)

	assert.Equal(t, 0, runMain([]string{"newscid", "--sixel", "games.si4"}, "linux", strings.NewReader(""), &out, &errBuf))
	assert.Equal(t, []string{"--sixel", "games.si4"}, tuiGot)

	// serve mode (success, error, bad flag)
	assert.Equal(t, 0, runMain([]string{"newscid", "serve", "--host", "0.0.0.0", "--port", "9090", "--data-dir", "/tmp/d"}, "linux", strings.NewReader(""), &out, &errBuf))
	assert.Equal(t, "0.0.0.0", srvHost)
	assert.Equal(t, 9090, srvPort)
	assert.Equal(t, "/tmp/d", srvData)

	srvErr = errors.New("bind failed")
	assert.Equal(t, 1, runMain([]string{"newscid", "serve"}, "linux", strings.NewReader(""), &out, &errBuf))
	srvErr = nil

	assert.Equal(t, 2, runMain([]string{"newscid", "serve", "--unknown-flag"}, "linux", strings.NewReader(""), &out, &errBuf))

	// Headless Linux fallback to TUI
	t.Setenv("DISPLAY", "")
	t.Setenv("WAYLAND_DISPLAY", "")
	assert.Equal(t, 0, runMain([]string{"newscid", "headless.si4"}, "linux", strings.NewReader(""), &out, &errBuf))
	assert.Equal(t, []string{"headless.si4"}, tuiGot)

	// Desktop GUI with DISPLAY set + panel/portal/title/fen/db flags
	t.Setenv("DISPLAY", ":0")
	windowStartupCfg = WindowStartupConfig{}
	assert.Equal(t, 0, runMain([]string{"newscid", "--window-panel", "board", "--fen", "startpos", "--db", "a.si4"}, "linux", strings.NewReader(""), &out, &errBuf))
	assert.Equal(t, 680, wailsOpts.Width)
	assert.Equal(t, "board", windowStartupCfg.PanelID)

	windowStartupCfg = WindowStartupConfig{}
	assert.Equal(t, 0, runMain([]string{"newscid", "--window-panel", "tree", "--title", "Custom Title"}, "linux", strings.NewReader(""), &out, &errBuf))
	assert.Equal(t, "Custom Title", wailsOpts.Title)
	assert.Equal(t, 820, wailsOpts.Width)

	windowStartupCfg = WindowStartupConfig{}
	assert.Equal(t, 0, runMain([]string{"newscid", "--portal-url", "https://lichess.org"}, "linux", strings.NewReader(""), &out, &errBuf))
	assert.Equal(t, "portal", windowStartupCfg.Mode)
	assert.Equal(t, 1120, wailsOpts.Width)

	// Wails error path
	wailsErr = errors.New("gui init failed")
	assert.Equal(t, 1, runMain([]string{"newscid"}, "linux", strings.NewReader(""), &out, &errBuf))
	wailsErr = nil

	// Android server mode
	t.Setenv("NEWSCID_PORT", "8123")
	t.Setenv("NEWSCID_HOST", "127.0.0.1")
	t.Setenv("NEWSCID_DATA_DIR", "/tmp/android")
	assert.Equal(t, 0, runMain([]string{"newscid"}, "android", strings.NewReader(""), &out, &errBuf))
	assert.Equal(t, 8123, srvPort)
	assert.Equal(t, "/tmp/android", srvData)

	srvErr = errors.New("android bind failed")
	assert.Equal(t, 1, runMain([]string{"newscid"}, "android", strings.NewReader(""), &out, &errBuf))
}
