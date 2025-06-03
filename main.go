package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	"logTime-go/backend"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app, err := backend.NewApp(nil)
	if err != nil {
		log.Fatalf("Erro ao inicializar a aplicação: %v", err)
	}

	if err := wails.Run(&options.App{
		Title:  "Teamwork Time Logger",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 255, G: 255, B: 255, A: 1},
		OnStartup:        app.Startup,
		OnShutdown:       app.Shutdown,
		Bind: []interface{}{
			app,
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			BackdropType:         windows.Mica,
		},
	}); err != nil {
		log.Fatalf("Erro ao executar a aplicação: %v", err)
	}

}
