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

// NewHistorySection builds the clipboard history flyout: a "Clipboard"
// header with a "Clear all" action and a close button, and a scrollable
// list of captured items rendered as cards, or a placeholder when the
// history is empty.
func NewHistorySection(store *clipboard.Store, onSelect func(clipboard.Item), onClose func()) fyne.CanvasObject {
	placeholder := widget.NewLabel("Clipboard history is empty")
	placeholder.Alignment = fyne.TextAlignCenter

	var updateEmptyState func()
	list := newHistoryList(store, onSelect, func() { updateEmptyState() })
	updateEmptyState = func() {
		if len(store.Items()) == 0 {
			placeholder.Show()
			list.Hide()
		} else {
			placeholder.Hide()
			list.Show()
		}
	}
	updateEmptyState()

	titleRow := newTitleRow(onClose, func() {
		store.Clear()
		list.Refresh()
		updateEmptyState()
	})
	dividerRow := newDividerRow()

	return container.NewBorder(
		container.NewVBox(titleRow, dividerRow),
		nil, nil, nil,
		container.NewStack(list, container.NewCenter(placeholder)),
	)
}

// newTitleRow builds the "Clipboard" header with a "Clear all" action and a
// close button.
func newTitleRow(onClose func(), onClearAll func()) fyne.CanvasObject {
	title := widget.NewLabelWithStyle("Clipboard", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	clearAll := newSmallButton("Clear all", onClearAll)
	closeBtn := newCircleIconButton(theme.CancelIcon(), onClose)

	right := container.NewHBox(container.NewCenter(clearAll), container.NewCenter(closeBtn))
	return container.New(layout.NewCustomPaddedLayout(6, 0, 4, 4),
		container.NewBorder(nil, nil, title, right),
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
// onChange is called after an item is removed, so callers can react to the
// list becoming empty.
func newHistoryList(store *clipboard.Store, onSelect func(clipboard.Item), onChange func()) *widget.List {
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
				onChange()
			}
		},
	)
	list.HideSeparators = true
	return list
}
