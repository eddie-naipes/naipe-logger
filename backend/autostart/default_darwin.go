//go:build darwin

package autostart

import (
	"os"
	"path/filepath"
)

func defaultLauncher() (Launcher, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	return NewLaunchAgent(filepath.Join(home, "Library", "LaunchAgents"), BundleID), nil
}
