//go:build windows

package screen

import "syscall"

var (
	user32               = syscall.NewLazyDLL("user32.dll")
	procGetSystemMetrics = user32.NewProc("GetSystemMetrics")
)

const (
	smCXScreen = 0
	smCYScreen = 1
)

// Size asks Windows for the primary display's resolution via
// GetSystemMetrics.
func Size() (width, height int, ok bool) {
	w, _, _ := procGetSystemMetrics.Call(uintptr(smCXScreen))
	h, _, _ := procGetSystemMetrics.Call(uintptr(smCYScreen))
	if w == 0 || h == 0 {
		return 0, 0, false
	}
	return int(w), int(h), true
}
