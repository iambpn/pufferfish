package window

import (
	"fyne.io/fyne/v2"

	"github.com/iambpn/pufferfish/internal/preferences"
	"github.com/iambpn/pufferfish/internal/ui"
)

// NewPreferencesWindow returns a function that opens the Pufferfish
// preferences window. The window is built fresh each time it's opened and
// destroyed when closed, so it holds no resources while not in use.
func NewPreferencesWindow(a fyne.App) func() {
	var w fyne.Window

	return func() {
		if w != nil {
			w.RequestFocus()
			return
		}

		w = a.NewWindow("Pufferfish Preferences")
		w.SetOnClosed(func() { w = nil })

		prefs := preferences.LoadClipboardPreferences(a)
		w.SetContent(ui.NewClipboardSection(prefs))
		w.Resize(fyne.NewSize(380, 0))
		w.Show()
	}
}
