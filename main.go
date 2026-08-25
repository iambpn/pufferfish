package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/driver/desktop"

	"github.com/iambpn/pufferfish/internal/clipboard"
	"github.com/iambpn/pufferfish/internal/window"
)

func main() {
	a := app.New()
	// Set the runtime application icon.
	// this is an embedded icon resource
	a.SetIcon(resourceIconPng)

	store := clipboard.NewStore()
	w := window.NewPreferencesWindow(a)
	history := window.NewHistoryWindow(a, store)

	// Set up the system tray icon and menu.
	if desk, ok := a.(desktop.App); ok {
		menu := fyne.NewMenu("Pufferfish",
			// clipboard items
			fyne.NewMenuItem("Clear", func() {}),
			fyne.NewMenuItem("Clipboard History", history.Show),
			fyne.NewMenuItem("Preferences", w.Show),
		)

		desk.SetSystemTrayMenu(menu)
		desk.SetSystemTrayIcon(resourceIconPng)
	}

	w.ShowAndRun()
}
