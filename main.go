package main

import (
	"embed"
	"fmt"
	"os"

	"newscid/pkg/version"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	if len(os.Args) > 1 {
		arg := os.Args[1]
		if arg == "-v" || arg == "--version" || arg == "version" {
			fmt.Println(version.String())
			os.Exit(0)
		}
		if arg == "-h" || arg == "--help" || arg == "help" {
			fmt.Printf("NewScid - Modern Chess Information Database (%s)\n", version.Short())
			fmt.Println("Usage: newscid [options] [database.si4 | file.pgn]")
			fmt.Println("\nOptions:")
			fmt.Println("  -v, --version    Print application version details")
			fmt.Println("  -h, --help       Display this help message")
			os.Exit(0)
		}
	}

	// Create an instance of the app structure
	app := NewApp()

	// Create application with options
	err := wails.Run(&options.App{
		Title:     "NewScid - Modern Chess Information Database",
		Width:     1280,
		Height:    850,
		MinWidth:  960,
		MinHeight: 650,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
