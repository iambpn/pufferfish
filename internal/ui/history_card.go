package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const cardCornerRadius float32 = 8

// historyCard is a single rounded, tappable row showing a captured item's
// content and a menu button to remove it.
type historyCard struct {
	widget.BaseWidget

	bg        *canvas.Rectangle
	content   *widget.Label
	deleteBtn *widget.Button
	onTap     func()
}

func newHistoryCard() *historyCard {
	c := &historyCard{
		bg:      canvas.NewRectangle(theme.Color(theme.ColorNameOverlayBackground)),
		content: widget.NewLabel(""),
	}
	c.bg.CornerRadius = cardCornerRadius
	c.content.Truncation = fyne.TextTruncateEllipsis
	c.deleteBtn = widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {})
	c.deleteBtn.Importance = widget.LowImportance
	c.ExtendBaseWidget(c)
	return c
}

func (c *historyCard) CreateRenderer() fyne.WidgetRenderer {
	inner := container.NewBorder(nil, nil, nil, c.deleteBtn, c.content)
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
