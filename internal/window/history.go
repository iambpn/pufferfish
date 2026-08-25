package window

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"

	"github.com/iambpn/pufferfish/internal/clipboard"
	"github.com/iambpn/pufferfish/internal/ui"
)

const (
	prefKeyHistoryPosX = "history.posX"
	prefKeyHistoryPosY = "history.posY"
)

// NewHistoryWindow returns a function that opens the undecorated window
// listing captured clipboard items. The window is built fresh each time
// it's opened and destroyed when closed, so it holds no resources while not
// in use. Selecting an item copies it back to the system clipboard.
func NewHistoryWindow(a fyne.App, store *clipboard.Store) func() {
	var win fyne.Window

	return func() {
		if win != nil {
			win.RequestFocus()
			return
		}

		w := a.Driver().(desktop.Driver).CreateSplashWindow()
		win = w
		w.SetOnClosed(func() {
			win = nil
			a.Lifecycle().SetOnExitedForeground(nil)
		})
		a.Lifecycle().SetOnExitedForeground(w.Close)

		if dw, ok := w.(desktop.Window); ok {
			prefs := a.Preferences()
			posX := prefs.IntWithFallback(prefKeyHistoryPosX, 100)
			posY := prefs.IntWithFallback(prefKeyHistoryPosY, 100)
			dw.RequestPosition(posX, posY)
		}

		w.SetContent(
			container.New(
				layout.NewCustomPaddedLayout(
					ui.PaddingSmall,
					ui.PaddingSmall,
					ui.PaddingSmall,
					ui.PaddingSmall,
				),
				ui.NewHistorySection(
					store,
					func(item clipboard.Item) {
						a.Clipboard().SetContent(item.Content)
					},
					w.Close,
				),
			),
		)

		w.Canvas().SetOnTypedKey(func(ev *fyne.KeyEvent) {
			if ev.Name == fyne.KeyEscape {
				w.Close()
			}
		})

		w.Resize(fyne.NewSize(380, 400))
		w.Show()
	}
}
