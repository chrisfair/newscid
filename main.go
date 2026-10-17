package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"os"
	"runtime"
	"strconv"

	"newscid/pkg/server"
	"newscid/pkg/version"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func runServer(app *App, host string, port int, dataDir string) {
	defer app.shutdown(context.Background())
	srv := server.New(server.ServerConfig{
		Host:     host,
		Port:     port,
		DataDir:  dataDir,
		AssetsFS: assets,
	}, app)

	fmt.Printf("NewScid Web / Mobile Server running on http://%s:%d\n", host, port)
	if err := srv.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}

func main() {
	app := NewApp()
	isAndroid := runtime.GOOS == "android"

	if len(os.Args) > 1 {
		arg := os.Args[1]
		if arg == "-v" || arg == "--version" || arg == "version" {
			fmt.Println(version.String())
			os.Exit(0)
		}
		if arg == "-h" || arg == "--help" || arg == "help" {
			fmt.Printf("NewScid - Modern Chess Information Database (%s)\n", version.Short())
			fmt.Println("Usage: newscid [options] [database.si4 | file.pgn]")
			fmt.Println("       newscid --window-panel <tree|engine|notation|games|board|portal|db-hub> [--title <title>]")
			fmt.Println("       newscid --portal-url <https://lichess.org | https://www.chess.com> [--title <title>]")
			fmt.Println("       newscid serve [--host 127.0.0.1] [--port 8080] [--data-dir /path]")
			fmt.Println("\nOptions:")
			fmt.Println("  -v, --version         Print application version details")
			fmt.Println("  -h, --help            Display this help message")
			fmt.Println("  --window-panel <id>   Launch a detached multi-screen panel window (for Hyprland / multi-monitor)")
			fmt.Println("  --portal-url <url>    Launch a dedicated live portal window into Lichess.org or Chess.com")
			fmt.Println("  serve, --server       Run in standalone web / mobile server mode (for Android / browser)")
			os.Exit(0)
		}
		if arg == "serve" || arg == "--server" || arg == "-server" {
			fs := flag.NewFlagSet("serve", flag.ExitOnError)
			host := fs.String("host", "127.0.0.1", "Host address to bind to (e.g. 127.0.0.1 or 0.0.0.0)")
			port := fs.Int("port", 8080, "Port to listen on")
			dataDir := fs.String("data-dir", "", "Data directory for databases and uploads")
			_ = fs.Parse(os.Args[2:])

			runServer(app, *host, *port, *dataDir)
			return
		}
	}

	windowTitle := "NewScid - Modern Chess Information Database"
	windowWidth := 1280
	windowHeight := 850
	minWidth := 460
	minHeight := 360

	for i := 1; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "--window-panel":
			if i+1 < len(os.Args) {
				windowStartupCfg.Mode = "panel"
				windowStartupCfg.PanelID = os.Args[i+1]
				windowTitle = fmt.Sprintf("NewScid — %s [Screen Panel]", os.Args[i+1])
				windowWidth = 820
				windowHeight = 680
				i++
			}
		case "--portal-url":
			if i+1 < len(os.Args) {
				windowStartupCfg.Mode = "portal"
				windowStartupCfg.PortalURL = os.Args[i+1]
				windowTitle = "NewScid Live Portal"
				windowWidth = 1120
				windowHeight = 800
				i++
			}
		case "--title":
			if i+1 < len(os.Args) {
				windowStartupCfg.PortalTitle = os.Args[i+1]
				windowTitle = os.Args[i+1]
				i++
			}
		case "--fen":
			if i+1 < len(os.Args) {
				windowStartupCfg.InitialFEN = os.Args[i+1]
				i++
			}
		case "--db":
			if i+1 < len(os.Args) {
				windowStartupCfg.InitialDB = os.Args[i+1]
				i++
			}
		}
	}

	// On Android, always run the server mode for WebView or Termux
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
		runServer(app, host, port, dataDir)
		return
	}

	// Create application with options
	err := wails.Run(&options.App{
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
		println("Error:", err.Error())
	}
}
