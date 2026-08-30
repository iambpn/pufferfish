package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/iambpn/pufferfish/internal/clipboard"
)

const (
	cardCornerRadius float32 = 8
	cardThumbSize    float32 = 28
)

// historyCard is a single rounded, tappable row showing a captured item's
// content and a button to remove it. Image items also show a thumbnail
// ahead of their label; the thumbnail is sized the same for every card, so
// text and image rows keep a uniform height in the list.
type historyCard struct {
	widget.BaseWidget

	bg        *canvas.Rectangle
	thumb     *canvas.Image
	content   *widget.Label
	deleteBtn *widget.Button
	onTap     func()
}

func newHistoryCard() *historyCard {
	c := &historyCard{
		bg:      canvas.NewRectangle(theme.Color(theme.ColorNameOverlayBackground)),
		thumb:   &canvas.Image{FillMode: canvas.ImageFillContain},
		content: widget.NewLabel(""),
	}
	c.bg.CornerRadius = cardCornerRadius
	c.thumb.SetMinSize(fyne.NewSquareSize(cardThumbSize))
	c.content.Truncation = fyne.TextTruncateEllipsis
	c.deleteBtn = widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {})
	c.deleteBtn.Importance = widget.LowImportance
	c.ExtendBaseWidget(c)
	return c
}

// setItem points the card at an item. imagePath is the file backing an
// image item, or empty for text.
func (c *historyCard) setItem(item clipboard.Item, imagePath string) {
	c.content.SetText(item.Preview())

	if imagePath == "" {
		c.thumb.Hide()
		return
	}
	c.thumb.File = imagePath
	c.thumb.Show()
	c.thumb.Refresh()
}

func (c *historyCard) CreateRenderer() fyne.WidgetRenderer {
	inner := container.NewBorder(nil, nil, c.thumb, c.deleteBtn, c.content)
	padded := container.NewPadded(inner)
	stack := container.NewStack(c.bg, padded)
	outer := container.NewPadded(stack)
	return widget.NewSimpleRenderer(outer)
}

func (c *historyCard) Tapped(*fyne.PointEvent) {
	if c.onTap != nil {
		c.onTap()
	}
}
