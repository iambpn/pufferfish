package window

import (
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"

	"github.com/iambpn/pufferfish/internal/clipboard"
	"github.com/iambpn/pufferfish/internal/preferences"
	"github.com/iambpn/pufferfish/internal/ui"
)

const (
	historyWindowWidth  = 380
	historyWindowHeight = 400
)

// NewHistoryWindow returns a function that opens the undecorated window
// listing captured clipboard items. The window is built fresh each time
// it's opened and destroyed when closed, so it holds no resources while not
// in use.
//
// Selecting an item puts it back on the system clipboard through the
// watcher, so the restore is not mistaken for a fresh copy, moves it to the
// front of the history as if it had just been copied, and closes the
// flyout. With "automatically paste" enabled the paste shortcut is then
// sent to whichever window regains focus.
func NewHistoryWindow(
	a fyne.App,
	store *clipboard.Store,
	watcher *clipboard.Watcher,
	prefs *preferences.ClipboardPreferences,
) func() {
	var sw singleWindow

	return func() {
		sw.Open(func(onClosed func()) fyne.Window {
			w := a.Driver().(desktop.Driver).CreateSplashWindow()

			selectItem := func(item clipboard.Item) {
				if err := watcher.Put(item); err != nil {
					fyne.LogError("could not restore the clipboard item", err)
					return
				}
				item.CapturedAt = time.Now()
				store.Add(item)
				w.Close()
				if prefs.AutoPaste {
					clipboard.Paste()
				}
			}

			clearAll := func() { clipboard.ClearAll(store, watcher) }

			content, detach := ui.NewHistorySection(store, selectItem, w.Close, clearAll)
			w.SetOnClosed(func() {
				detach()
				onClosed()
				a.Lifecycle().SetOnExitedForeground(nil)
			})
			a.Lifecycle().SetOnExitedForeground(w.Close)

			// The window has no title bar, so wrap its content in a mover
			// that drags the window when the pointer is dragged over any
			// non-interactive part of it.
			var root fyne.CanvasObject = content
			dw, movable := w.(desktop.Window)
			if movable {
				root = newWindowMover(content, dw, prefs.HistoryX, prefs.HistoryY, prefs.SetHistoryCoords)
			}

			w.SetContent(
				container.New(
					layout.NewCustomPaddedLayout(
						ui.PaddingSmall,
						ui.PaddingSmall,
						ui.PaddingSmall,
						ui.PaddingSmall,
					),
					root,
				),
			)

			w.Canvas().SetOnTypedKey(func(ev *fyne.KeyEvent) {
				if ev.Name == fyne.KeyEscape {
					w.Close()
				}
			})

			w.Resize(fyne.NewSize(historyWindowWidth, historyWindowHeight))
			w.Show()
			// A splash window centers itself the first time it's shown,
			// which would override an earlier RequestPosition, so restore
			// the saved position only now that centering has happened.
			if movable {
				dw.RequestPosition(prefs.HistoryX, prefs.HistoryY)
			}
			return w
		})
	}
}
