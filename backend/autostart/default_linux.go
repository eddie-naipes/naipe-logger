//go:build linux

package autostart

import (
	"os"
	"path/filepath"
)

// os.UserConfigDir respeita XDG_CONFIG_HOME (padrão ~/.config).
func defaultLauncher() (Launcher, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	return NewDesktopEntry(filepath.Join(dir, "autostart")), nil
}
