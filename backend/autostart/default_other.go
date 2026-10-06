//go:build !windows && !darwin && !linux

package autostart

func defaultLauncher() (Launcher, error) {
	return nil, ErrUnsupported
}
