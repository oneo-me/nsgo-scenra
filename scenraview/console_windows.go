//go:build windows

package scenraview

import (
	"os/exec"
	"syscall"
)

// hideConsole keeps a console program the editor runs, such as Git, from
// opening a console window.
func hideConsole(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	const createNoWindow = 0x08000000 // CREATE_NO_WINDOW
	cmd.SysProcAttr.HideWindow = true
	cmd.SysProcAttr.CreationFlags |= createNoWindow
}
