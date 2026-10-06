package main

import (
	"embed"
	"log/slog"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	"logTime-go/backend"
	"logTime-go/backend/autostart"
	"logTime-go/backend/logging"
)

//go:embed all:frontend/dist
var assets embed.FS

// wailsJSON é embutido para que o binário saiba a própria versão: o CI grava a
// versão da tag em info.productVersion antes do `wails build`.
//
//go:embed wails.json
var wailsJSON []byte

func main() {
	// O log é configurado antes de tudo para que a carga da configuração (e
	// seus avisos de arquivo corrompido ou credencial antiga) já vá para o
	// arquivo. Em `wails dev` ou com TEAMWORK_LOGGER_DEBUG ligado, também
	// sai no terminal.
	debug := logging.DebugFromEnv()
	logger, err := logging.Setup(logging.Options{
		Debug:  debug,
		Stderr: debug || logging.DevBuild(),
	})
	if err != nil {
		slog.Warn("Log em arquivo indisponível", "err", err)
	}
	defer logger.Close()

	version := backend.MarkDevVersion(backend.ParseProductVersion(wailsJSON), logging.DevBuild())

	app, err := backend.NewApp(nil, backend.Options{
		LogsDir: logger.Dir(),
		Version: version,
	})
	if err != nil {
		fatal("Erro ao inicializar a aplicação", err, logger)
	}

	slog.Info("Aplicação iniciada", "versao", version)

	// Aberto pelo item "iniciar com o sistema": começa minimizado (na barra de
	// tarefas), com lembretes e cronômetro rodando. StartHidden não serve: o
	// app não tem ícone na bandeja e a janela ficaria inalcançável.
	startState := options.Normal
	if autostart.HasMinimizedFlag(os.Args[1:]) {
		startState = options.Minimised
	}

	if err := wails.Run(&options.App{
		Title:  "Teamwork Time Logger",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 255, G: 255, B: 255, A: 1},
		WindowStartState: startState,
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
		fatal("Erro ao executar a aplicação", err, logger)
	}
}

// fatal registra o erro e encerra. os.Exit pula os defers, então o arquivo de
// log é fechado aqui para não perder a última linha.
func fatal(msg string, err error, logger *logging.Logger) {
	slog.Error(msg, "err", err)
	_ = logger.Close()
	os.Exit(1)
}
