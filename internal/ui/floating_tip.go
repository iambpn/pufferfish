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

// showNear displays text just below target, aligned to target's left edge -
// or, when there isn't enough room below for it to fit without spilling
// past floatingTip's own bottom edge (where Fyne would clip it, making it
// invisible), above target instead. target may sit anywhere in the canvas,
// not necessarily as a direct row alongside floatingTip - both positions
// are resolved to absolute canvas coordinates first, so nesting target
// inside other layout containers (a VBox, padding, ...) can't misposition
// the tip.
func (t *floatingTip) showNear(text string, target fyne.CanvasObject) {
	t.label.SetText(text)

	driver := fyne.CurrentApp().Driver()
	targetPos := driver.AbsolutePositionForObject(target)
	tipPos := driver.AbsolutePositionForObject(t)
	rel := targetPos.Subtract(tipPos)

	boxHeight := t.wrappedHeight() + floatingTipPadH
	below := rel.Y + target.Size().Height + 4
	y := below
	if below+boxHeight > t.Size().Height {
		if above := rel.Y - 4 - boxHeight; above >= 0 {
			y = above
		}
	}

	t.offset = fyne.NewPos(rel.X, y)
	t.active = true
	t.Refresh()
}

// wrappedHeight resizes the label to floatingTipWidth, forcing a rewrap,
// and reports the height that wrap needs.
func (t *floatingTip) wrappedHeight() float32 {
	t.label.Resize(fyne.NewSize(floatingTipWidth, t.label.MinSize().Height))
	return t.label.MinSize().Height
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
	wrappedHeight := r.tip.wrappedHeight()
	r.tip.label.Resize(fyne.NewSize(floatingTipWidth, wrappedHeight))

	boxSize := fyne.NewSize(floatingTipWidth+floatingTipPadW, wrappedHeight+floatingTipPadH)
	r.tip.bg.Resize(boxSize)

	pos := offscreen
	if r.tip.active {
		pos = r.tip.offset
	}
	r.tip.bg.Move(pos)
	r.tip.label.Move(pos.AddXY(floatingTipPadW/2, floatingTipPadH/2))
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
