//go:build windows

package gitlog

import (
	"os/exec"
	"syscall"
)

// createNoWindow é CREATE_NO_WINDOW: sem ele, cada chamada ao git a partir do
// app (que não tem console) abriria uma janela de terminal por um instante.
const createNoWindow = 0x08000000

func hideWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}
}
