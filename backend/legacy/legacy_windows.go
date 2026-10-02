//go:build windows

package legacy

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// hklmRegistry lê HKLM nas visões de 64 e 32 bits: o installer.nsi antigo
// usava SetRegView 64, mas um instalador 32 bits teria gravado em
// WOW6432Node.
type hklmRegistry struct{}

func (hklmRegistry) UninstallEntries() ([]Entry, error) {
	var entries []Entry
	seen := map[string]bool{}
	var errs []error
	for _, view := range []uint32{registry.WOW64_64KEY, registry.WOW64_32KEY} {
		list, err := readUninstall(view)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		for _, e := range list {
			if !seen[e.KeyName] {
				seen[e.KeyName] = true
				entries = append(entries, e)
			}
		}
	}
	if len(entries) == 0 && len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return entries, nil
}

func readUninstall(view uint32) ([]Entry, error) {
	root, err := registry.OpenKey(registry.LOCAL_MACHINE, UninstallKey, registry.ENUMERATE_SUB_KEYS|registry.QUERY_VALUE|view)
	if err != nil {
		return nil, err
	}
	defer root.Close()

	names, err := root.ReadSubKeyNames(-1)
	if err != nil {
		return nil, err
	}
	entries := make([]Entry, 0, len(names))
	for _, name := range names {
		k, err := registry.OpenKey(root, name, registry.QUERY_VALUE|view)
		if err != nil {
			continue
		}
		e := Entry{KeyName: name}
		e.DisplayName, _, _ = k.GetStringValue("DisplayName")
		e.InstallLocation, _, _ = k.GetStringValue("InstallLocation")
		e.UninstallString, _, _ = k.GetStringValue("UninstallString")
		k.Close()
		entries = append(entries, e)
	}
	return entries, nil
}

func currentExeDir() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	return filepath.Dir(exe)
}

// Detect procura uma instalação antiga em HKLM.
func Detect() (Install, error) {
	entries, err := hklmRegistry{}.UninstallEntries()
	if err != nil {
		return Install{}, err
	}
	return Select(entries, currentExeDir()), nil
}

// RunUninstaller executa o desinstalador da instalação antiga com o verbo
// "runas": ele foi instalado como administrador e o Windows pede UAC. Um
// CreateProcess comum falharia com ERROR_ELEVATION_REQUIRED.
func RunUninstaller(install Install) error {
	if !install.Found {
		return ErrNotFound
	}
	exe, args, err := validateUninstaller(install.UninstallString)
	if err != nil {
		return err
	}
	if _, err := os.Stat(exe); err != nil {
		return fmt.Errorf("desinstalador antigo não encontrado em %s", exe)
	}

	verb, _ := windows.UTF16PtrFromString("runas")
	file, err := windows.UTF16PtrFromString(exe)
	if err != nil {
		return err
	}
	var argsPtr *uint16
	if args != "" {
		if argsPtr, err = windows.UTF16PtrFromString(args); err != nil {
			return err
		}
	}
	dir, _ := windows.UTF16PtrFromString(filepath.Dir(exe))
	if err := windows.ShellExecute(0, verb, file, argsPtr, dir, windows.SW_SHOWNORMAL); err != nil {
		// ERROR_CANCELLED: o usuário recusou o UAC.
		if errors.Is(err, windows.ERROR_CANCELLED) {
			return errors.New("desinstalação cancelada no aviso do Windows (UAC)")
		}
		return fmt.Errorf("não foi possível executar o desinstalador antigo: %v", err)
	}
	return nil
}
