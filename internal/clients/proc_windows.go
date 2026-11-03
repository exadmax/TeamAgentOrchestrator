//go:build windows

package clients

import (
	"os/exec"
	"syscall"
)

// hideConsole keeps "<exe> --version" and "npm view" from flashing a console window behind the GUI.
func hideConsole(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000} // CREATE_NO_WINDOW
}
