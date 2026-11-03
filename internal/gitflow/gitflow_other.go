//go:build !windows

package gitflow

import "os/exec"

func hideConsole(cmd *exec.Cmd) {}
