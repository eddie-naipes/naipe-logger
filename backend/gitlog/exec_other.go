//go:build !windows

package gitlog

import "os/exec"

func hideWindow(*exec.Cmd) {}
