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

// NewHistoryWindow builds the undecorated window listing captured clipboard
// items. Selecting an item copies it back to the system clipboard. Since the
// window has no OS title bar, it exposes a drag handle whose movement is
// applied to the window position and remembered across launches.
func NewHistoryWindow(a fyne.App, store *clipboard.Store) fyne.Window {
	w := a.Driver().(desktop.Driver).CreateSplashWindow()
	dw := w.(desktop.Window)

	prefs := a.Preferences()
	posX := prefs.IntWithFallback(prefKeyHistoryPosX, 100)
	posY := prefs.IntWithFallback(prefKeyHistoryPosY, 100)
	dw.RequestPosition(posX, posY)

	onDragged := func(dx, dy float32) {
		posX += int(dx)
		posY += int(dy)
		dw.RequestPosition(posX, posY)
	}
	onDragEnd := func() {
		prefs.SetInt(prefKeyHistoryPosX, posX)
		prefs.SetInt(prefKeyHistoryPosY, posY)
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
				w.Hide,
				onDragged,
				onDragEnd,
			),
		),
	)

	w.Resize(fyne.NewSize(380, 400))
	return w
}
