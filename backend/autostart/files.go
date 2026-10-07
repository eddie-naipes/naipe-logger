package autostart

import (
	"bytes"
	"encoding/xml"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"logTime-go/backend/internal/fsutil"
)

// BundleID é o identificador usado no LaunchAgent do macOS. É o padrão do
// Wails (com.wails.<name do wails.json>), o mesmo do Info.plist gerado.
const BundleID = "com.wails.TeamworkLogger"

// DesktopFileName é o nome da entrada de autostart no Linux.
const DesktopFileName = "teamwork-logger.desktop"

// fileLauncher grava/remove um arquivo; Enable é idempotente (reescreve).
type fileLauncher struct {
	path   string
	render func(exe string, args []string) []byte
}

func (f fileLauncher) Enable(exe string, args []string) error {
	if err := os.MkdirAll(filepath.Dir(f.path), 0o755); err != nil {
		return err
	}
	data := f.render(exe, args)
	if atual, err := os.ReadFile(f.path); err == nil && bytes.Equal(atual, data) {
		return nil
	}
	// 0644: launchd e o gerenciador de sessão precisam ler; não há segredo.
	return fsutil.WriteFileAtomic(f.path, data, 0o644)
}

func (f fileLauncher) Disable() error {
	err := os.Remove(f.path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (f fileLauncher) Enabled() (bool, error) {
	_, err := os.Stat(f.path)
	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, os.ErrNotExist):
		return false, nil
	default:
		return false, err
	}
}

// NewLaunchAgent é o Launcher do macOS: ~/Library/LaunchAgents/<label>.plist
// com RunAtLoad. dir é a pasta LaunchAgents (injetável nos testes).
func NewLaunchAgent(dir, label string) Launcher {
	return fileLauncher{
		path:   filepath.Join(dir, label+".plist"),
		render: func(exe string, args []string) []byte { return launchAgentPlist(label, exe, args) },
	}
}

func xmlEscape(s string) string {
	var b strings.Builder
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}

func launchAgentPlist(label, exe string, args []string) []byte {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">` + "\n")
	b.WriteString(`<plist version="1.0">` + "\n<dict>\n")
	b.WriteString("\t<key>Label</key>\n\t<string>" + xmlEscape(label) + "</string>\n")
	b.WriteString("\t<key>ProgramArguments</key>\n\t<array>\n")
	for _, a := range append([]string{exe}, args...) {
		b.WriteString("\t\t<string>" + xmlEscape(a) + "</string>\n")
	}
	b.WriteString("\t</array>\n")
	b.WriteString("\t<key>RunAtLoad</key>\n\t<true/>\n")
	b.WriteString("</dict>\n</plist>\n")
	return []byte(b.String())
}

// NewDesktopEntry é o Launcher do Linux: <dir>/teamwork-logger.desktop, com
// dir = ~/.config/autostart (injetável nos testes).
func NewDesktopEntry(dir string) Launcher {
	return fileLauncher{
		path:   filepath.Join(dir, DesktopFileName),
		render: desktopEntry,
	}
}

// desktopExecArg cita um argumento conforme a especificação Desktop Entry:
// entre aspas, escapando " ` $ e \ (e % vira %%).
func desktopExecArg(a string) string {
	a = strings.ReplaceAll(a, "%", "%%")
	if !strings.ContainsAny(a, " \t\n\"'\\><~|&;$*?#()`") {
		return a
	}
	r := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "`", "\\`", `$`, `\$`)
	return `"` + r.Replace(a) + `"`
}

func desktopEntry(exe string, args []string) []byte {
	parts := []string{desktopExecArg(exe)}
	for _, a := range args {
		parts = append(parts, desktopExecArg(a))
	}
	var b strings.Builder
	b.WriteString("[Desktop Entry]\n")
	b.WriteString("Type=Application\n")
	b.WriteString("Name=Teamwork Logger\n")
	b.WriteString("Comment=Lançamento de horas no Teamwork\n")
	b.WriteString("Exec=" + strings.Join(parts, " ") + "\n")
	b.WriteString("Terminal=false\n")
	b.WriteString("X-GNOME-Autostart-enabled=true\n")
	return []byte(b.String())
}
