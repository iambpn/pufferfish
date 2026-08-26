package window

import (
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"

	"github.com/iambpn/pufferfish/internal/clipboard"
	"github.com/iambpn/pufferfish/internal/preferences"
	"github.com/iambpn/pufferfish/internal/screen"
	"github.com/iambpn/pufferfish/internal/ui"
)

const (
	historyWindowWidth  = 380
	historyWindowHeight = 400

	// historyWindowMargin keeps an edge-anchored window off the screen edge
	// it's anchored to.
	historyWindowMargin = 20

	// historyWindowFallbackX/Y is where the window opens when its position
	// can't be computed - HistoryPositionCenter, or no screen.Size backend
	// on this platform.
	historyWindowFallbackX = 100
	historyWindowFallbackY = 100
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

			w.Resize(fyne.NewSize(historyWindowWidth, historyWindowHeight))
			w.Show()
			// A splash window centers itself the first time it's shown, which
			// would override a position requested beforehand - so position it
			// only now that centering has already happened.
			positionHistoryWindow(w, prefs.HistoryPosition)
			return w
		})
	}
}

// positionHistoryWindow moves w to match pos. Every anchor, including
// Center, needs the screen size to compute its target position, which Fyne
// has no portable API for, so they all go through RequestPosition; only
// when the screen size can't be determined does Center fall back to Fyne's
// own CenterOnScreen (a documented no-op under Wayland, but still the best
// available option with nothing to compute a position from).
//
// RequestPosition is still only a request: Fyne's own docs note a window manager may
// ignore RequestPosition outright, and on GNOME's Mutter (tested under
// Wayland with XWayland windows) the delivered position can also come out
// scaled from what was asked, for reasons outside Pufferfish's control. On
// window managers that honor positioning requests as given, this places the
// window exactly at the requested anchor.
func positionHistoryWindow(w fyne.Window, pos preferences.HistoryPosition) {
	if pos == "" {
		pos = preferences.HistoryPositionCenter
	}

	dw, ok := w.(desktop.Window)
	if !ok {
		// No RequestPosition on this platform - CenterOnScreen is the best
		// available placement for every anchor.
		w.CenterOnScreen()
		return
	}

	screenW, screenH, ok := screen.Size()
	if !ok {
		// CenterOnScreen is itself a documented no-op under Wayland, but
		// with no screen size to compute a position from it's still the
		// best available fallback there, and works as intended elsewhere.
		if pos == preferences.HistoryPositionCenter {
			w.CenterOnScreen()
			return
		}
		dw.RequestPosition(historyWindowFallbackX, historyWindowFallbackY)
		return
	}

	x, y := historyWindowOrigin(pos, screenW, screenH)
	dw.RequestPosition(x, y)
}

// historyWindowOrigin computes the top-left corner for the history window
// under pos, given the primary screen's size. The 9 anchors themselves come
// from preferences.HistoryAnchors, the single shared source also used to
// label the position dropdown in the preferences UI.
func historyWindowOrigin(pos preferences.HistoryPosition, screenW, screenH int) (x, y int) {
	xs := [3]int{
		historyWindowMargin,
		(screenW - historyWindowWidth) / 2,
		screenW - historyWindowWidth - historyWindowMargin,
	}
	ys := [3]int{
		historyWindowMargin,
		(screenH - historyWindowHeight) / 2,
		screenH - historyWindowHeight - historyWindowMargin,
	}

	for _, a := range preferences.HistoryAnchors {
		if a.Position == pos {
			return xs[a.Col], ys[a.Row]
		}
	}
	return historyWindowFallbackX, historyWindowFallbackY
}
