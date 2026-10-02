//go:build !windows

package legacy

import "errors"

// Detect: instalações antigas em Program Files só existiram no Windows.
func Detect() (Install, error) {
	return Install{}, nil
}

// RunUninstaller não se aplica fora do Windows.
func RunUninstaller(Install) error {
	return errors.New("desinstalação de versão antiga só se aplica ao Windows")
}
