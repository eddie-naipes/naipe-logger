// Package logging configura o log/slog do aplicativo: grava em
// ~/.teamwork-logger/logs/app.log com rotação por tamanho, espelha no stderr
// em modo de desenvolvimento e mascara segredos (o token de API) em toda
// mensagem antes de ela sair do processo.
package logging

import (
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"logTime-go/backend/internal/fsutil"
)

const (
	// EnvDebug liga o nível Debug. Desligado por padrão: um lote de
	// lançamentos gerava uma linha por tarefa×dia e despejava corpos de
	// resposta inteiros no log.
	EnvDebug = "TEAMWORK_LOGGER_DEBUG"

	// DefaultMaxSize e DefaultMaxBackups limitam o log a ~20 MB no disco do
	// usuário (atual + 3 anteriores de 5 MB).
	DefaultMaxSize    int64 = 5 << 20
	DefaultMaxBackups       = 3

	// LogFileName é o nome do arquivo de log atual dentro de LogsDir.
	LogFileName = "app.log"
)

// IsTruthy interpreta valores de variável de ambiente como ligado/desligado.
// Só valores explícitos ligam: TEAMWORK_LOGGER_DEBUG=0 ou vazio ficam
// desligados.
func IsTruthy(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on", "sim":
		return true
	default:
		return false
	}
}

// DebugFromEnv informa se TEAMWORK_LOGGER_DEBUG está ligada.
func DebugFromEnv() bool {
	return IsTruthy(os.Getenv(EnvDebug))
}

// DevBuild informa se o binário foi compilado por `wails dev` (build tag
// "dev"), em que o log também vai para o terminal.
func DevBuild() bool { return devBuild }

// DefaultDir devolve ~/.teamwork-logger/logs.
func DefaultDir() (string, error) {
	appDir, err := fsutil.AppDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(appDir, "logs"), nil
}

// Options controla Setup. Campos zerados usam os padrões.
type Options struct {
	Dir        string    // vazio: DefaultDir()
	Debug      bool      // nível Debug em vez de Info
	Stderr     bool      // espelha no stderr (modo dev)
	StderrOut  io.Writer // destino do espelho; nil = os.Stderr (testes trocam)
	MaxSize    int64
	MaxBackups int
}

// Logger é o resultado de Setup: o logger já instalado como padrão do slog e o
// arquivo que precisa ser fechado no encerramento.
type Logger struct {
	*slog.Logger
	dir  string
	file *RotatingFile
}

// Dir devolve a pasta dos logs ("" se o arquivo não pôde ser aberto).
func (l *Logger) Dir() string { return l.dir }

// Close fecha o arquivo de log.
func (l *Logger) Close() error {
	if l == nil || l.file == nil {
		return nil
	}
	return l.file.Close()
}

// Setup monta o logger e o instala com slog.SetDefault (o que também redireciona
// o pacote log). Se o arquivo não puder ser aberto, o app segue logando só no
// stderr e o erro é devolvido para ser registrado; não é motivo para não abrir.
func Setup(opts Options) (*Logger, error) {
	level := slog.LevelInfo
	if opts.Debug {
		level = slog.LevelDebug
	}
	if opts.MaxSize <= 0 {
		opts.MaxSize = DefaultMaxSize
	}
	if opts.MaxBackups <= 0 {
		opts.MaxBackups = DefaultMaxBackups
	}

	var setupErr error
	dir := opts.Dir
	if dir == "" {
		dir, setupErr = DefaultDir()
	}

	var writers []io.Writer
	var file *RotatingFile
	if setupErr == nil {
		if err := os.MkdirAll(dir, fsutil.DirPerm); err != nil {
			setupErr = err
		} else {
			// MkdirAll não corrige um diretório antigo com permissão aberta.
			_ = os.Chmod(dir, fsutil.DirPerm)
			file, setupErr = OpenRotatingFile(filepath.Join(dir, LogFileName), opts.MaxSize, opts.MaxBackups)
			if setupErr == nil {
				writers = append(writers, file)
			}
		}
	}
	if setupErr != nil {
		dir = ""
	}

	if opts.Stderr || len(writers) == 0 {
		out := opts.StderrOut
		if out == nil {
			out = os.Stderr
		}
		writers = append(writers, out)
	}

	handler := NewRedactingHandler(slog.NewTextHandler(io.MultiWriter(writers...), &slog.HandlerOptions{Level: level}))
	logger := slog.New(handler)
	slog.SetDefault(logger)

	if setupErr != nil {
		setupErr = errors.Join(errors.New("não foi possível abrir o arquivo de log; usando só o stderr"), setupErr)
	}
	return &Logger{Logger: logger, dir: dir, file: file}, setupErr
}
