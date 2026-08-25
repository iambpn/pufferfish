package window

import (
	"fyne.io/fyne/v2"

	"github.com/iambpn/pufferfish/internal/preferences"
	"github.com/iambpn/pufferfish/internal/ui"
)

// NewPreferencesWindow builds the Pufferfish preferences window.
func NewPreferencesWindow(a fyne.App) fyne.Window {
	w := a.NewWindow("Pufferfish Preferences")

	prefs := preferences.LoadClipboardPreferences(a)
	w.SetContent(ui.NewClipboardSection(prefs))
	w.Resize(fyne.NewSize(380, 0))
	return w
}
