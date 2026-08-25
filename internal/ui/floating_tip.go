/*
floatingTip is a single shared floating label shown near whichever row is
hovered. It lives alongside the row content (not as a canvas overlay), so
showing it never disrupts hover dispatch on the rows underneath.

bg/label are never Hide()/Show()-ed: a Fyne canvas object that starts
hidden isn't registered for repaint until some unrelated redraw discovers
it, so an early Refresh() on it silently does nothing. Instead they stay
permanently Visible and sized correctly at all times; "hidden" is done by
moving them to offscreen rather than resizing to zero, since repeatedly
resizing a rounded rect to/from zero has shown visual glitches - Move is
the more reliably-redrawn operation.
*/
package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const (
	floatingTipWidth float32 = 300
	floatingTipPadW  float32 = 12
	floatingTipPadH  float32 = 6
)

type floatingTip struct {
	widget.BaseWidget

	bg     *canvas.Rectangle
	label  *widget.Label
	offset fyne.Position
	active bool
}

func newFloatingTip() *floatingTip {
	t := &floatingTip{
		bg:    canvas.NewRectangle(theme.Color(theme.ColorNameMenuBackground)),
		label: widget.NewLabel(""),
	}
	t.bg.CornerRadius = 4
	t.label.Wrapping = fyne.TextWrapWord
	t.ExtendBaseWidget(t)
	return t
}

// showNear displays text just below target, aligned to target's left edge.
// target must share floatingTip's coordinate space (i.e. be a direct row in
// the same stack).
func (t *floatingTip) showNear(text string, target fyne.CanvasObject) {
	t.label.SetText(text)
	pos := target.Position()
	t.offset = fyne.NewPos(pos.X, pos.Y+target.Size().Height+4)
	t.active = true
	t.Refresh()
}

func (t *floatingTip) hide() {
	t.active = false
	t.Refresh()
}

func (t *floatingTip) CreateRenderer() fyne.WidgetRenderer {
	return &floatingTipRenderer{tip: t, objects: []fyne.CanvasObject{t.bg, t.label}}
}

type floatingTipRenderer struct {
	tip     *floatingTip
	objects []fyne.CanvasObject
}

// offscreen is where bg/label sit while inactive.
var offscreen = fyne.NewPos(-10000, -10000)

func (r *floatingTipRenderer) Layout(fyne.Size) {
	label := r.tip.label

	// Resize forces a rewrap at floatingTipWidth; MinSize afterward reports
	// the wrapped height for that width.
	label.Resize(fyne.NewSize(floatingTipWidth, label.MinSize().Height))
	wrappedHeight := label.MinSize().Height
	label.Resize(fyne.NewSize(floatingTipWidth, wrappedHeight))

	boxSize := fyne.NewSize(floatingTipWidth+floatingTipPadW, wrappedHeight+floatingTipPadH)
	r.tip.bg.Resize(boxSize)

	pos := offscreen
	if r.tip.active {
		pos = r.tip.offset
	}
	r.tip.bg.Move(pos)
	label.Move(pos.AddXY(floatingTipPadW/2, floatingTipPadH/2))
}

func (r *floatingTipRenderer) MinSize() fyne.Size {
	return fyne.NewSize(0, 0)
}

func (r *floatingTipRenderer) Refresh() {
	r.Layout(fyne.Size{})
	r.tip.bg.Refresh()
	r.tip.label.Refresh()
}

func (r *floatingTipRenderer) Objects() []fyne.CanvasObject {
	return r.objects
}

func (r *floatingTipRenderer) Destroy() {}
