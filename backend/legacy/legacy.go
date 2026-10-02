// Package legacy detecta instalações antigas do app no Windows.
//
// Até a 1.x o instalador (installer.nsi / teamwork-installer.nsi, e depois o
// project.nsi padrão do Wails) rodava como administrador, instalava em
// Program Files e registrava a desinstalação em HKLM:
//
//   - HKLM\...\Uninstall\Naipe Logger                     (DisplayName "Naipe Logger",
//     UninstallString sem aspas "$INSTDIR\uninst.exe", sem InstallLocation)
//   - HKLM\...\Uninstall\Naipe Sync SolutionsTeamwork Logger (DisplayName
//     "Teamwork Logger", UninstallString "\"$INSTDIR\uninstall.exe\"")
//
// O instalador atual é por usuário (HKCU, %LOCALAPPDATA%\Programs). Quem
// atualiza fica com duas cópias e dois atalhos; este pacote encontra a cópia
// antiga para a UI oferecer a remoção.
package legacy

import (
	"errors"
	"path/filepath"
	"strings"
)

// UninstallKey é a chave (em HKLM) onde o Windows lista os programas.
const UninstallKey = `Software\Microsoft\Windows\CurrentVersion\Uninstall`

// Nomes de exibição usados pelos instaladores antigos.
var legacyDisplayNames = []string{"teamwork logger", "naipe logger"}

// ErrNotFound indica que não há instalação antiga a remover.
var ErrNotFound = errors.New("nenhuma instalação antiga encontrada")

// Install descreve uma instalação antiga encontrada, no formato do frontend.
type Install struct {
	Found           bool   `json:"found"`
	DisplayName     string `json:"displayName"`
	InstallLocation string `json:"installLocation"`
	UninstallString string `json:"uninstallString"`
}

// Entry é uma subchave de Uninstall lida do registro.
type Entry struct {
	KeyName         string
	DisplayName     string
	InstallLocation string
	UninstallString string
}

// Registry lista as entradas de desinstalação de HKLM. Interface para que a
// seleção seja testável sem registro real (e fora do Windows).
type Registry interface {
	UninstallEntries() ([]Entry, error)
}

// Select escolhe, entre as entradas, a instalação antiga do app: nome
// conhecido, desinstalador registrado e pasta diferente da do executável em
// execução (se o próprio app rodando é o de Program Files, removê-lo seria
// apagar a si mesmo).
func Select(entries []Entry, currentExeDir string) Install {
	current := normalizeDir(currentExeDir)
	for _, e := range entries {
		if !matchesName(e.DisplayName) || strings.TrimSpace(e.UninstallString) == "" {
			continue
		}
		location := strings.TrimSpace(e.InstallLocation)
		if location == "" {
			// O installer.nsi antigo não gravava InstallLocation; a pasta é a
			// do desinstalador.
			exe, _ := SplitCommand(e.UninstallString)
			if exe != "" {
				location = windowsDir(exe)
			}
		}
		if location == "" || (current != "" && normalizeDir(location) == current) {
			continue
		}
		return Install{
			Found:           true,
			DisplayName:     e.DisplayName,
			InstallLocation: location,
			UninstallString: e.UninstallString,
		}
	}
	return Install{}
}

func matchesName(displayName string) bool {
	name := strings.ToLower(displayName)
	for _, known := range legacyDisplayNames {
		if strings.Contains(name, known) {
			return true
		}
	}
	return false
}

// normalizeDir compara pastas do Windows: sem diferença de caixa, barra final
// ou aspas.
func normalizeDir(dir string) string {
	dir = strings.Trim(strings.TrimSpace(dir), `"`)
	if dir == "" {
		return ""
	}
	dir = strings.ReplaceAll(dir, "/", `\`)
	dir = strings.TrimRight(dir, `\`)
	return strings.ToLower(dir)
}

// SplitCommand separa o executável dos argumentos de um UninstallString.
// Aceita a forma com aspas ("C:\...\uninstall.exe" /S) e a forma sem aspas
// com espaços no caminho (C:\Program Files\Naipe Logger\uninst.exe), que é a
// que o installer.nsi antigo gravava.
func SplitCommand(cmd string) (exe, args string) {
	cmd = strings.TrimSpace(cmd)
	if strings.HasPrefix(cmd, `"`) {
		end := strings.Index(cmd[1:], `"`)
		if end < 0 {
			return "", ""
		}
		return cmd[1 : end+1], strings.TrimSpace(cmd[end+2:])
	}
	lower := strings.ToLower(cmd)
	if i := strings.Index(lower, ".exe"); i >= 0 {
		return cmd[:i+4], strings.TrimSpace(cmd[i+4:])
	}
	return "", ""
}

// validateUninstaller confere que o comando aponta para um .exe com caminho
// absoluto, antes de pedir elevação ao Windows para executá-lo.
func validateUninstaller(uninstallString string) (exe, args string, err error) {
	exe, args = SplitCommand(uninstallString)
	if exe == "" || !strings.EqualFold(filepath.Ext(exe), ".exe") || !isAbsWindowsPath(exe) {
		return "", "", errors.New("desinstalador antigo com caminho inválido")
	}
	return exe, args, nil
}

// isAbsWindowsPath não usa filepath.IsAbs para funcionar igual em qualquer SO
// (os testes rodam no CI Linux).
func isAbsWindowsPath(p string) bool {
	return len(p) >= 3 && p[1] == ':' && (p[2] == '\\' || p[2] == '/')
}

// windowsDir devolve a pasta de um caminho do Windows. Não usa filepath.Dir
// porque, no CI Linux, "\" não é separador e o resultado seria ".".
func windowsDir(p string) string {
	i := strings.LastIndexAny(p, `\/`)
	if i < 0 {
		return ""
	}
	if i == 2 && p[1] == ':' {
		// Raiz do volume: mantém a barra ("C:\").
		return p[:3]
	}
	return p[:i]
}
