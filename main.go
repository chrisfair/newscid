package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"

	"newscid/pkg/server"
	"newscid/pkg/tui"
	"newscid/pkg/version"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

var (
	runTUIFunc    = tui.RunCLI
	runServerFunc = runServer
	wailsRunFunc  = wails.Run
)

func runServer(app *App, host string, port int, dataDir string) error {
	defer app.shutdown(context.Background())
	srv := server.New(server.ServerConfig{
		Host:     host,
		Port:     port,
		DataDir:  dataDir,
		AssetsFS: assets,
	}, app)

	fmt.Printf("NewScid Web / Mobile Server running on http://%s:%d\n", host, port)
	return srv.Start()
}

func printCLIHelp(w io.Writer) {
	fmt.Fprintf(w, "NewScid - Modern Chess Information Database (%s)\n", version.Short())
	fmt.Fprintln(w, "Usage: newscid [options] [database.si4 | file.pgn]")
	fmt.Fprintln(w, "       newscid tui [--kitty|--unicode] [--theme <id>] [--piece-set <id>] [database.si4 | file.pgn]")
	fmt.Fprintln(w, "       newscid --window-panel <board|tree|engine|notation|games|imbalance|info|portal|db-roles> [--workspace <1..10>] [--title <title>]")
	fmt.Fprintln(w, "       newscid --portal-url <https://lichess.org | https://www.chess.com> [--title <title>]")
	fmt.Fprintln(w, "       newscid serve [--host 127.0.0.1] [--port 8080] [--data-dir /path]")
	fmt.Fprintln(w, "\nOptions:")
	fmt.Fprintln(w, "  -v, --version         Print application version details")
	fmt.Fprintln(w, "  -h, --help            Display this help message")
	fmt.Fprintln(w, "  tui, --tui            Launch the Kitty / Sixel / TrueColor terminal workstation (TUI)")
	fmt.Fprintln(w, "  --kitty               Launch the terminal workstation with HD pixel board (Kitty, or Sixel on foot)")
	fmt.Fprintln(w, "  --sixel               Launch the terminal workstation with DEC Sixel HD pixel board (foot, xterm)")
	fmt.Fprintln(w, "  --window-panel <id>   Launch a detached multi-screen panel window (for Hyprland / multi-monitor)")
	fmt.Fprintln(w, "  --workspace <1..10>   Place the window onto a specific Hyprland workspace")
	fmt.Fprintln(w, "  --portal-url <url>    Launch a dedicated live portal window into Lichess.org or Chess.com")
	fmt.Fprintln(w, "  serve, --server       Run in standalone web / mobile server mode (for Android / browser)")
}

// extractTUIArgs detects `tui`, `--tui`, `--kitty`, or `--sixel` anywhere in args
// and returns the normalized argument slice to pass to tui.RunCLI.
func extractTUIArgs(args []string) ([]string, bool) {
	if len(args) > 1 && args[1] == "tui" {
		return args[2:], true
	}
	for i := 1; i < len(args); i++ {
		var prefix string
		switch args[i] {
		case "--tui", "-tui":
		case "--kitty", "-kitty":
			prefix = "--kitty"
		case "--sixel", "-sixel":
			prefix = "--sixel"
		default:
			continue
		}
		var rest []string
		if prefix != "" {
			rest = append(rest, prefix)
		}
		rest = append(rest, args[1:i]...)
		rest = append(rest, args[i+1:]...)
		return rest, true
	}
	return nil, false
}

func runMain(args []string, goos string, stdin io.Reader, stdout, stderr io.Writer) int {
	isAndroid := goos == "android"

	if len(args) > 1 {
		switch args[1] {
		case "-v", "--version", "version":
			fmt.Fprintln(stdout, version.String())
			return 0
		case "-h", "--help", "help":
			printCLIHelp(stdout)
			return 0
		}
	}

	if tuiArgs, ok := extractTUIArgs(args); ok {
		return runTUIFunc(tuiArgs, stdin, stdout, stderr)
	}

	app := NewApp()

	if len(args) > 1 {
		switch args[1] {
		case "serve", "--server", "-server":
			fs := flag.NewFlagSet("serve", flag.ContinueOnError)
			fs.SetOutput(stderr)
			host := fs.String("host", "127.0.0.1", "Host address to bind to (e.g. 127.0.0.1 or 0.0.0.0)")
			port := fs.Int("port", 8080, "Port to listen on")
			dataDir := fs.String("data-dir", "", "Data directory for databases and uploads")
			if err := fs.Parse(args[2:]); err != nil {
				return 2
			}
			if err := runServerFunc(app, *host, *port, *dataDir); err != nil {
				fmt.Fprintf(stderr, "Server error: %v\n", err)
				return 1
			}
			return 0
		}
	}

	// Headless Linux SSH / console fallback: when neither X11 nor Wayland is active, launch TUI directly.
	if goos == "linux" && !isAndroid && os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
		return runTUIFunc(args[1:], stdin, stdout, stderr)
	}

	windowTitle := "NewScid - Modern Chess Information Database"
	windowWidth := 1280
	windowHeight := 850
	minWidth := 460
	minHeight := 360

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--window-panel":
			if i+1 < len(args) {
				windowStartupCfg.Mode = "panel"
				windowStartupCfg.PanelID = args[i+1]
				windowTitle = fmt.Sprintf("NewScid — %s [Screen Panel]", args[i+1])
				if windowStartupCfg.PanelID == "board" {
					windowWidth = 680
					windowHeight = 720
				} else {
					windowWidth = 820
					windowHeight = 680
				}
				i++
			}
		case "--workspace":
			if i+1 < len(args) {
				windowStartupCfg.Workspace = args[i+1]
				i++
			}
		case "--portal-url":
			if i+1 < len(args) {
				windowStartupCfg.Mode = "portal"
				windowStartupCfg.PortalURL = args[i+1]
				windowTitle = "NewScid Live Portal"
				windowWidth = 1120
				windowHeight = 800
				i++
			}
		case "--title":
			if i+1 < len(args) {
				windowStartupCfg.PortalTitle = args[i+1]
				windowTitle = args[i+1]
				i++
			}
		case "--fen":
			if i+1 < len(args) {
				windowStartupCfg.InitialFEN = args[i+1]
				i++
			}
		case "--db":
			if i+1 < len(args) {
				windowStartupCfg.InitialDB = args[i+1]
				i++
			}
		}
	}

	if windowStartupCfg.Workspace != "" {
		go func(ws string) {
			dispatchHyprlandWorkspace(os.Getpid(), ws, true)
		}(windowStartupCfg.Workspace)
	}

	// On Android, always run the server mode for WebView or Termux.
	if isAndroid {
		port := 8080
		if portEnv := os.Getenv("NEWSCID_PORT"); portEnv != "" {
			if p, err := strconv.Atoi(portEnv); err == nil && p > 0 {
				port = p
			}
		}
		host := "127.0.0.1"
		if hostEnv := os.Getenv("NEWSCID_HOST"); hostEnv != "" {
			host = hostEnv
		}
		dataDir := os.Getenv("NEWSCID_DATA_DIR")
		if err := runServerFunc(app, host, port, dataDir); err != nil {
			fmt.Fprintf(stderr, "Server error: %v\n", err)
			return 1
		}
		return 0
	}

	err := wailsRunFunc(&options.App{
		Title:     windowTitle,
		Width:     windowWidth,
		Height:    windowHeight,
		MinWidth:  minWidth,
		MinHeight: minHeight,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return 1
	}
	return 0
}

func main() {
	if code := runMain(os.Args, runtime.GOOS, os.Stdin, os.Stdout, os.Stderr); code != 0 {
		os.Exit(code)
	}
}
