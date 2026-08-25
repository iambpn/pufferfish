package window

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"

	"github.com/iambpn/pufferfish/internal/clipboard"
	"github.com/iambpn/pufferfish/internal/preferences"
	"github.com/iambpn/pufferfish/internal/ui"
)

const (
	prefKeyHistoryPosX = "history.posX"
	prefKeyHistoryPosY = "history.posY"
)

// NewHistoryWindow returns a function that opens the undecorated window
// listing captured clipboard items. The window is built fresh each time
// it's opened and destroyed when closed, so it holds no resources while not
// in use.
//
// Selecting an item puts it back on the system clipboard through the
// watcher, so the restore is not mistaken for a fresh copy, and closes the
// flyout. With "automatically paste" enabled the paste shortcut is then
// sent to whichever window regains focus.
func NewHistoryWindow(
	a fyne.App,
	store *clipboard.Store,
	watcher *clipboard.Watcher,
	prefs *preferences.ClipboardPreferences,
) func() {
	var win fyne.Window

	return func() {
		if win != nil {
			win.RequestFocus()
			return
		}

		w := a.Driver().(desktop.Driver).CreateSplashWindow()
		win = w

		if dw, ok := w.(desktop.Window); ok {
			p := a.Preferences()
			posX := p.IntWithFallback(prefKeyHistoryPosX, 100)
			posY := p.IntWithFallback(prefKeyHistoryPosY, 100)
			dw.RequestPosition(posX, posY)
		}

		selectItem := func(item clipboard.Item) {
			if err := watcher.Put(item); err != nil {
				fyne.LogError("could not restore the clipboard item", err)
				return
			}
			w.Close()
			if prefs.AutoPaste {
				clipboard.Paste()
			}
		}

		content, detach := ui.NewHistorySection(store, selectItem, w.Close)
		w.SetOnClosed(func() {
			detach()
			win = nil
			a.Lifecycle().SetOnExitedForeground(nil)
		})
		a.Lifecycle().SetOnExitedForeground(w.Close)

		w.SetContent(
			container.New(
				layout.NewCustomPaddedLayout(
					ui.PaddingSmall,
					ui.PaddingSmall,
					ui.PaddingSmall,
					ui.PaddingSmall,
				),
				content,
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
