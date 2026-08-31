package main

import (
	"flag"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/driver/desktop"

	"github.com/iambpn/pufferfish/internal/clipboard"
	"github.com/iambpn/pufferfish/internal/ipc"
	"github.com/iambpn/pufferfish/internal/preferences"
	"github.com/iambpn/pufferfish/internal/window"
)

func main() {
	dev := flag.Bool("dev", false, "open the history window on startup")
	history := flag.Bool("history", false, "open the history window, or focus it in the already-running instance")
	flag.Parse()

	// If another instance is already running, hand it the command and stop
	// instead of starting a second copy of the app.
	if *history && ipc.Send(ipc.ShowHistoryCmd) {
		return
	}

	a := app.New()
	// Set the runtime application icon.
	// this is an embedded icon resource
	a.SetIcon(resourceIconPng)

	// Claim the IPC port before setting up any store/watcher/tray state, so
	// a launch that loses the race to an already-running instance exits
	// immediately instead of running a second, uncoordinated copy of the
	// app against the same history.json.
	var showHistory func()
	ln, ok := ipc.Listen(func(cmd string) {
		if cmd == ipc.ShowHistoryCmd && showHistory != nil {
			fyne.Do(showHistory)
		}
	})
	if !ok {
		fyne.LogError("another instance is already running", nil)
		return
	}
	defer ln.Close()

	clipboardReady := true
	if err := clipboard.Init(); err != nil {
		// Without the system clipboard the app still runs, so the saved
		// history stays readable; only tracking and restoring are lost.
		fyne.LogError("clipboard tracking is disabled", err)
		clipboardReady = false
	}

	store := clipboard.NewStore(storageDir(a))
	store.Load()

	// A save triggered by the last change before quitting is written in the
	// background; without waiting for it here, it could still be in flight
	// when the process exits and never make it to disk.
	a.Lifecycle().SetOnStopped(store.Flush)

	prefs := preferences.LoadClipboardPreferences(a)
	watcher := clipboard.NewWatcher(store)

	apply := func() {
		store.SetLimit(prefs.RecentItems)
		watcher.SetCaptureImages(prefs.AddImages)
		watcher.SetEnabled(clipboardReady && prefs.UseClipboard)
	}
	prefs.AddListener(apply)
	apply()

	if clipboardReady && prefs.KeepContent {
		restoreLastItem(store, watcher)
	}

	showPreferences := window.NewPreferencesWindow(a, prefs)
	showHistory = window.NewHistoryWindow(a, store, watcher, prefs)

	// Set up the system tray icon and menu.
	if desk, ok := a.(desktop.App); ok {
		menu := fyne.NewMenu("Pufferfish",
			fyne.NewMenuItem("Open History", showHistory),
			fyne.NewMenuItem("Clear History", func() { clipboard.ClearAll(store, watcher) }),
			fyne.NewMenuItem("Preferences", showPreferences),
		)

		desk.SetSystemTrayMenu(menu)
		desk.SetSystemTrayIcon(resourceIconPng)
	}

	// With --dev or --history, open the history window right away instead
	// of starting hidden in the tray.
	if *dev || *history {
		showHistory()
	}

	// Start hidden in the system tray; windows are opened from the tray menu.
	a.Run()
}

// storageDir is where the history index and its image files are kept. An
// app without writable storage falls back to an in-memory history, which
// also means no image capture.
func storageDir(a fyne.App) string {
	root := a.Storage().RootURI()
	if root == nil {
		return ""
	}
	return root.Path()
}

// restoreLastItem puts the newest history entry back on the system
// clipboard, so "keep clipboard content" carries the last copy across a
// restart.
func restoreLastItem(store *clipboard.Store, watcher *clipboard.Watcher) {
	item, ok := store.Newest()
	if !ok {
		return
	}
	if err := watcher.Put(item); err != nil {
		fyne.LogError("could not restore the last clipboard item", err)
	}
}
