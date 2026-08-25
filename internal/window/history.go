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
	var win fyne.Window

	return func() {
		if win != nil {
			win.RequestFocus()
			return
		}

		w := a.Driver().(desktop.Driver).CreateSplashWindow()
		win = w

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

		clearAll := func() {
			store.Clear()
			if err := watcher.Clear(); err != nil {
				fyne.LogError("could not clear the system clipboard", err)
			}
		}

		content, detach := ui.NewHistorySection(store, selectItem, w.Close, clearAll)
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

		w.Resize(fyne.NewSize(historyWindowWidth, historyWindowHeight))
		w.Show()
		// A splash window centers itself the first time it's shown, which
		// would override a position requested beforehand - so position it
		// only now that centering has already happened.
		positionHistoryWindow(w, prefs.HistoryPosition)
	}
}

// positionHistoryWindow moves w to match pos. Center uses Fyne's own
// CenterOnScreen; the other anchors need the screen size, which Fyne has no
// portable API for, so they go through RequestPosition instead.
//
// Both are still only requests: Fyne's own docs note a window manager may
// ignore RequestPosition outright, and on GNOME's Mutter (tested under
// Wayland with XWayland windows) the delivered position can also come out
// scaled from what was asked, for reasons outside Pufferfish's control. On
// window managers that honor positioning requests as given, this places the
// window exactly at the requested anchor.
func positionHistoryWindow(w fyne.Window, pos preferences.HistoryPosition) {
	if pos == preferences.HistoryPositionCenter || pos == "" {
		w.CenterOnScreen()
		return
	}

	dw, ok := w.(desktop.Window)
	if !ok {
		return
	}

	screenW, screenH, ok := screen.Size()
	if !ok {
		dw.RequestPosition(historyWindowFallbackX, historyWindowFallbackY)
		return
	}

	x, y := historyWindowOrigin(pos, screenW, screenH)
	dw.RequestPosition(x, y)
}

// historyWindowOrigin computes the top-left corner for the history window
// under pos, given the primary screen's size.
func historyWindowOrigin(pos preferences.HistoryPosition, screenW, screenH int) (x, y int) {
	left := historyWindowMargin
	right := screenW - historyWindowWidth - historyWindowMargin
	centerX := (screenW - historyWindowWidth) / 2
	top := historyWindowMargin
	bottom := screenH - historyWindowHeight - historyWindowMargin
	centerY := (screenH - historyWindowHeight) / 2

	switch pos {
	case preferences.HistoryPositionTopLeft:
		return left, top
	case preferences.HistoryPositionTopCenter:
		return centerX, top
	case preferences.HistoryPositionTopRight:
		return right, top
	case preferences.HistoryPositionCenterLeft:
		return left, centerY
	case preferences.HistoryPositionCenterRight:
		return right, centerY
	case preferences.HistoryPositionBottomLeft:
		return left, bottom
	case preferences.HistoryPositionBottomCenter:
		return centerX, bottom
	case preferences.HistoryPositionBottomRight:
		return right, bottom
	default:
		return historyWindowFallbackX, historyWindowFallbackY
	}
}
