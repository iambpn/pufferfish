/*
Paste synthesises the Ctrl+V (Cmd+V on macOS) that "automatically paste
selected item" promises, by sending the keystroke to whichever window holds
focus once the history flyout has closed.

Injecting a keystroke is not something a Fyne app can do to windows it does
not own, so this shells out to whichever helper the desktop provides. The
candidates are tried in order of how widely they work and the first one
present on PATH is used for the rest of the run:

  - ydotool works on both Wayland and X11 because it injects at the kernel
    level through its daemon, which is what makes it the only one of the
    three that works on GNOME and KDE.
  - wtype speaks the Wayland virtual-keyboard protocol, which wlroots
    compositors (Sway, Hyprland) support and GNOME and KDE do not.
  - xdotool is X11-only. Under XWayland it can reach XWayland windows but
    not native Wayland ones.

macOS and Windows each have one always-present helper, so there is nothing
to choose between there.

A helper that is installed can still fail - ydotool needs its daemon, macOS
needs the accessibility permission - so each carries a hint explaining the
fix, and ydotool additionally carries a recovery step that starts the
daemon. Recovery is attempted at most once per run: a setup that is broken
for some other reason must not re-run it on every paste.
*/
package clipboard

import (
	"context"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"time"

	"fyne.io/fyne/v2"
)

const (
	// pasteFocusDelay gives the window manager time to move focus back to
	// the application being pasted into after the flyout closes. Sending
	// the keystroke too early delivers it to a window that is on its way
	// out.
	pasteFocusDelay = 150 * time.Millisecond

	// pasteTimeout bounds a helper that hangs, e.g. ydotool waiting on a
	// daemon socket that nothing is listening on.
	pasteTimeout = 5 * time.Second

	// pasteRecoverCooldown is the shortest gap between two attempts to
	// repair the helper.
	pasteRecoverCooldown = 30 * time.Second
)

// pasteHelper is one external command able to send the paste shortcut.
type pasteHelper struct {
	name string
	args []string

	// env is added to the helper's environment.
	env []string

	// recover tries to make the helper usable after it failed. A nil
	// recover means the failure is not something Pufferfish can fix.
	recover func() error

	// hint explains what the user has to do when the helper keeps failing.
	hint string
}

// pasteHelpers lists the candidates for this OS, most broadly compatible
// first.
func pasteHelpers() []pasteHelper {
	switch runtime.GOOS {
	case "darwin":
		return []pasteHelper{{
			name: "osascript",
			args: []string{"-e", `tell application "System Events" to keystroke "v" using command down`},
			hint: "macOS blocks synthetic keystrokes until the app is allowed to control the computer: " +
				"add Pufferfish under System Settings > Privacy & Security > Accessibility",
		}}
	case "windows":
		return []pasteHelper{{
			name: "powershell",
			args: []string{"-NoProfile", "-Command",
				`(New-Object -ComObject wscript.shell).SendKeys('^v')`},
			hint: "Windows only lets a program send keys to windows at or below its own privilege level, " +
				"so pasting into an app running as administrator needs Pufferfish to run as administrator too",
		}}
	default:
		return []pasteHelper{
			{
				// Linux input event codes: 29 is left control, 47 is V.
				// ":1" presses a key and ":0" releases it.
				name:    "ydotool",
				args:    []string{"key", "29:1", "47:1", "47:0", "29:0"},
				env:     []string{"YDOTOOL_SOCKET=" + ydotoolSocket()},
				recover: startYdotoold,
				hint: "ydotool needs its daemon running and write access to /dev/uinput: " +
					"start it with `systemctl --user start ydotoold`, and if that fails see the README udev rule",
			},
			{
				name: "wtype",
				args: []string{"-M", "ctrl", "v", "-m", "ctrl"},
				hint: "wtype needs a compositor with the virtual-keyboard protocol (Sway, Hyprland); " +
					"on GNOME or KDE install ydotool instead",
			},
			{
				name: "xdotool",
				args: []string{"key", "--clearmodifiers", "ctrl+v"},
				hint: "xdotool cannot reach native Wayland windows; install ydotool for a Wayland session",
			},
		}
	}
}

var (
	pasteOnce   sync.Once
	pastePath   string
	pasteChosen pasteHelper

	recoverMu   sync.Mutex
	lastRecover time.Time
)

// findPasteHelper resolves the helper to use, once per run.
func findPasteHelper() {
	pasteOnce.Do(func() {
		for _, helper := range pasteHelpers() {
			path, err := exec.LookPath(helper.name)
			if err != nil {
				continue
			}
			pastePath, pasteChosen = path, helper
			return
		}
		fyne.LogError("automatic paste needs one of ydotool, wtype or xdotool on PATH", nil)
	})
}

// Paste sends the paste shortcut to the focused window. It is a no-op when
// no helper is installed.
func Paste() {
	go func() {
		findPasteHelper()
		if pastePath == "" {
			return
		}

		time.Sleep(pasteFocusDelay)

		err := runPasteHelper()
		if err == nil {
			return
		}

		// The helper is installed but did not work. If the reason is
		// something Pufferfish can fix - a daemon that is not running -
		// fix it and send the keystroke again.
		if pasteChosen.recover != nil && attemptPasteRecovery() {
			if err = runPasteHelper(); err == nil {
				return
			}
		}

		fyne.LogError("automatic paste failed: "+pasteChosen.hint, err)
	}()
}

func runPasteHelper() error {
	ctx, cancel := context.WithTimeout(context.Background(), pasteTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, pastePath, pasteChosen.args...)
	cmd.Env = append(os.Environ(), pasteChosen.env...)
	hideConsole(cmd)

	return cmd.Run()
}

// attemptPasteRecovery runs the chosen helper's recovery step, reporting
// whether the helper is now worth retrying.
//
// The cooldown is what keeps a setup that is broken for some other reason
// from re-running recovery on every single paste, while still letting a
// daemon that dies later in the session be brought back - which a
// once-per-run guard would not.
func attemptPasteRecovery() bool {
	recoverMu.Lock()
	defer recoverMu.Unlock()

	if time.Since(lastRecover) < pasteRecoverCooldown {
		return false
	}
	lastRecover = time.Now()

	if err := pasteChosen.recover(); err != nil {
		fyne.LogError("could not prepare automatic paste", err)
		return false
	}
	return true
}
