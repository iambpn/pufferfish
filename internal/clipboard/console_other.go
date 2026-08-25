//go:build !windows

package clipboard

import "os/exec"

// hideConsole is a no-op: only Windows shows a console for a helper.
func hideConsole(*exec.Cmd) {}
