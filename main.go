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
	showPreferences := window.NewPreferencesWindow(a)
	showHistory := window.NewHistoryWindow(a, store)

	// Set up the system tray icon and menu.
	if desk, ok := a.(desktop.App); ok {
		menu := fyne.NewMenu("Pufferfish",
			fyne.NewMenuItem("Open History", showHistory),
			fyne.NewMenuItem("Clear History", store.Clear),
			fyne.NewMenuItem("Preferences", showPreferences),
		)

		desk.SetSystemTrayMenu(menu)
		desk.SetSystemTrayIcon(resourceIconPng)
	}

	// Start hidden in the system tray; windows are opened from the tray menu.
	a.Run()
}
