//go:build !windows

package scenraview

import "os/exec"

// hideConsole does nothing where programs open no console window.
func hideConsole(*exec.Cmd) {}
