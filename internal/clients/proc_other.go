//go:build !windows

package clients

import "os/exec"

func hideConsole(*exec.Cmd) {}
