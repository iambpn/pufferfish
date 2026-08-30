package window

import (
	"fyne.io/fyne/v2"

	"github.com/iambpn/pufferfish/internal/preferences"
	"github.com/iambpn/pufferfish/internal/ui"
)

// NewPreferencesWindow returns a function that opens the Pufferfish
// preferences window. The window is built fresh each time it's opened and
// destroyed when closed, so it holds no resources while not in use. It
// edits the app's live preferences, so every change takes effect at once
// rather than on the next launch.
func NewPreferencesWindow(a fyne.App, prefs *preferences.ClipboardPreferences) func() {
	var sw singleWindow

	return func() {
		sw.Open(func(onClosed func()) fyne.Window {
			w := a.NewWindow("Pufferfish Preferences")
			w.SetOnClosed(onClosed)

			w.SetContent(ui.NewClipboardSection(prefs))
			w.Resize(fyne.NewSize(380, 0))
			w.Show()
			return w
		})
	}
}
