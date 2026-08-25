package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/iambpn/pufferfish/internal/clipboard"
)

var dividerColor = color.NRGBA{R: 128, G: 128, B: 128, A: 60}

// NewHistorySection builds the clipboard history flyout: a drag handle and
// close button, a "Clipboard" header with a "Clear all" action, and a
// scrollable list of captured items rendered as cards.
func NewHistorySection(store *clipboard.Store, onSelect func(clipboard.Item), onClose func(), onDragged func(dx, dy float32), onDragEnd func()) fyne.CanvasObject {
	list := newHistoryList(store, onSelect)
	dragRow := newDragRow(func() {
		list.ScrollToTop()
		onClose()
	}, onDragged, onDragEnd)
	titleRow := newTitleRow(func() {
		store.Clear()
		list.Refresh()
	})
	dividerRow := newDividerRow()

	return container.NewBorder(
		container.NewVBox(dragRow, titleRow, dividerRow),
		nil, nil, nil,
		list,
	)
}

// newDragRow builds the top row: a drag handle standing in for a title bar,
// and a close button.
func newDragRow(onClose func(), onDragged func(dx, dy float32), onDragEnd func()) fyne.CanvasObject {
	handle := newDragHandle(onDragged, onDragEnd)
	closeBtn := newCircleIconButton(theme.CancelIcon(), onClose)

	row := container.NewBorder(nil, nil, nil, closeBtn,
		container.New(layout.NewCustomPaddedLayout(6, 0, 0, 0),
			container.NewVBox(container.NewCenter(handle)),
		),
	)
	return container.New(layout.NewCustomPaddedLayout(0, 0, 0, 0), row)
}

// newTitleRow builds the "Clipboard" header with a "Clear all" action.
func newTitleRow(onClearAll func()) fyne.CanvasObject {
	title := widget.NewLabelWithStyle("Clipboard", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	clearAll := newSmallButton("Clear all", onClearAll)

	return container.New(layout.NewCustomPaddedLayout(6, 0, 4, 4),
		container.NewBorder(nil, nil, title, container.NewCenter(clearAll)),
	)
}

// newDividerRow builds a thin inset divider under the header.
func newDividerRow() fyne.CanvasObject {
	divider := canvas.NewRectangle(dividerColor)
	divider.SetMinSize(fyne.NewSize(0, 1))
	return container.New(layout.NewCustomPaddedLayout(0, 4, 8, 8), divider)
}

// newHistoryList builds the scrollable list of captured items rendered as
// cards, wired to copy an item on tap and delete it via its delete button.
func newHistoryList(store *clipboard.Store, onSelect func(clipboard.Item)) *widget.List {
	var list *widget.List
	list = widget.NewList(
		func() int { return len(store.Items()) },
		func() fyne.CanvasObject { return newHistoryCard() },
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			item := store.Items()[id]
			card := obj.(*historyCard)
			card.content.SetText(item.Content)
			card.onTap = func() { onSelect(item) }
			card.deleteBtn.OnTapped = func() {
				store.RemoveAt(id)
				list.Refresh()
			}
		},
	)
	list.HideSeparators = true
	return list
}
