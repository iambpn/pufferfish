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
//
// The section follows the store while it is open, so items copied while the
// flyout is showing appear straight away. The returned detach function
// unsubscribes it and must be called when the window closes.
func NewHistorySection(store *clipboard.Store, onSelect func(clipboard.Item), onClose func(), onClearAll func()) (content fyne.CanvasObject, detach func()) {
	placeholder := widget.NewLabel("Clipboard history is empty")
	placeholder.Alignment = fyne.TextAlignCenter

	list := newHistoryList(store, onSelect)
	listNoShadow := container.NewThemeOverride(list, noScrollShadowTheme{Theme: theme.DefaultTheme()})

	refresh := func() {
		list.Refresh()
		if store.Len() == 0 {
			placeholder.Show()
			list.Hide()
		} else {
			placeholder.Hide()
			list.Show()
		}
	}
	refresh()
	detach = store.AddListener(refresh)

	titleRow := newTitleRow(onClose, onClearAll)
	dividerRow := newDividerRow()

	content = container.NewBorder(
		container.NewVBox(titleRow, dividerRow),
		nil, nil, nil,
		container.NewStack(listNoShadow, container.NewCenter(placeholder)),
	)
	return content, detach
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
// Removing an item notifies the store, which refreshes the list through the
// section's listener.
func newHistoryList(store *clipboard.Store, onSelect func(clipboard.Item)) *widget.List {
	list := widget.NewList(
		func() int { return store.Len() },
		func() fyne.CanvasObject { return newHistoryCard() },
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			item, ok := store.ItemAt(id)
			if !ok {
				return
			}

			imagePath, _ := store.ThumbPath(item)
			card := obj.(*historyCard)
			card.setItem(item, imagePath)
			card.onTap = func() { onSelect(item) }
			card.deleteBtn.OnTapped = func() { store.RemoveAt(id) }
		},
	)
	list.HideSeparators = true
	return list
}

// noScrollShadowTheme hides the drop shadow that Fyne's scroll container
// draws at the edge where more content can be scrolled into view. The
// cards already reach the flyout's edges, so that shadow looks like a
// smudge over the top card instead of a scroll hint. Making the shadow
// color transparent removes it.
type noScrollShadowTheme struct {
	fyne.Theme
}

func (t noScrollShadowTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	if name == theme.ColorNameShadow {
		return color.Transparent
	}
	return t.Theme.Color(name, variant)
}
