//go:build windows

package runner

import (
	"os/exec"
	"strconv"
	"syscall"
)

const createNoWindow = 0x08000000

// hideWindow keeps the CLI from flashing a console window behind the GUI.
func hideWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}
}

// killTree ends the CLI and every process it started (node children, shells).
func killTree(cmd *exec.Cmd) error {
	kill := exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(cmd.Process.Pid))
	hideWindow(kill)
	if err := kill.Run(); err != nil {
		return cmd.Process.Kill()
	}
	return nil
}
