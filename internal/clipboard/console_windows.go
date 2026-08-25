//go:build windows

package clipboard

import (
	"os/exec"
	"syscall"
)

// hideConsole keeps the helper from flashing a console window on screen.
// Without it every automatic paste blinks a PowerShell window over the app
// being pasted into.
func hideConsole(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
}
