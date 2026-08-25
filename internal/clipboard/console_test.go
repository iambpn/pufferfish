package clipboard

import (
	"os/exec"
	"testing"
)

// hideConsole has one implementation per OS (console_other.go,
// console_windows.go); only the one matching GOOS is compiled in, so this
// exercises whichever is active without needing a build tag itself.
func TestHideConsoleDoesNotPanic(t *testing.T) {
	cmd := exec.Command("true")
	hideConsole(cmd)
}
