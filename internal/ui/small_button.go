package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const smallButtonTextSize float32 = 11

// smallButton is a compact, pill-shaped text button, smaller than the
// theme's default widget.Button, for secondary actions like "Clear all".
type smallButton struct {
	widget.BaseWidget
	hoverBackground

	text *canvas.Text
	bg   *canvas.Rectangle

	onTapped func()
}

func newSmallButton(label string, onTapped func()) *smallButton {
	b := &smallButton{onTapped: onTapped}
	b.text = canvas.NewText(label, theme.Color(theme.ColorNameForeground))
	b.text.TextSize = smallButtonTextSize
	b.bg = canvas.NewRectangle(theme.Color(theme.ColorNameInputBackground))
	b.bg.CornerRadius = 8
	b.ExtendBaseWidget(b)
	return b
}

func (b *smallButton) CreateRenderer() fyne.WidgetRenderer {
	padded := container.New(layout.NewCustomPaddedLayout(4, 4, 8, 8), b.text)
	return widget.NewSimpleRenderer(container.NewStack(b.bg, padded))
}

func (b *smallButton) Tapped(*fyne.PointEvent) {
	if b.onTapped != nil {
		b.onTapped()
	}
}

func (b *smallButton) Cursor() desktop.Cursor {
	return desktop.PointerCursor
}

func (b *smallButton) MouseIn(*desktop.MouseEvent) {
	b.hovered = true
	b.bg.FillColor = b.fillColor()
	b.bg.Refresh()
}

func (b *smallButton) MouseMoved(*desktop.MouseEvent) {}

func (b *smallButton) MouseOut() {
	b.hovered = false
	b.bg.FillColor = b.fillColor()
	b.bg.Refresh()
}
