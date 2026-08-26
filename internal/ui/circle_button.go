package ui

import (
	"image/color"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const (
	circleButtonDiameter    float32       = 30
	circleButtonHoldDelay   time.Duration = 400 * time.Millisecond
	circleButtonRepeatEvery time.Duration = 100 * time.Millisecond
)

// circleIconButton is a round icon button with a circular ripple on tap and
// press-and-hold repeat for its action.
type circleIconButton struct {
	widget.BaseWidget
	hoverBackground

	icon   fyne.Resource
	action func()

	repeating bool
	pressGen  int

	bg     *canvas.Circle
	ripple *canvas.Circle
}

func newCircleIconButton(icon fyne.Resource, action func()) *circleIconButton {
	b := &circleIconButton{icon: icon, action: action}
	b.ExtendBaseWidget(b)
	return b
}

func (b *circleIconButton) CreateRenderer() fyne.WidgetRenderer {
	b.bg = canvas.NewCircle(theme.Color(theme.ColorNameInputBackground))
	b.ripple = canvas.NewCircle(color.Transparent)
	b.ripple.Hidden = true

	iconObj := canvas.NewImageFromResource(b.icon)
	iconObj.FillMode = canvas.ImageFillContain

	return &circleIconButtonRenderer{
		button:  b,
		icon:    iconObj,
		objects: []fyne.CanvasObject{b.bg, b.ripple, iconObj},
	}
}

func (b *circleIconButton) MinSize() fyne.Size {
	return fyne.NewSquareSize(circleButtonDiameter)
}

func (b *circleIconButton) Tapped(pe *fyne.PointEvent) {
	center := fyne.NewPos(circleButtonDiameter/2, circleButtonDiameter/2)
	if pe != nil {
		center = pe.Position
	}
	b.startRipple(center)

	if b.repeating {
		b.repeating = false
		return
	}
	b.action()
}

func (b *circleIconButton) startRipple(center fyne.Position) {
	b.ripple.Hidden = false
	maxRadius := circleButtonDiameter / 2

	base := color.NRGBAModel.Convert(theme.Color(theme.ColorNamePressed)).(color.NRGBA)

	anim := fyne.NewAnimation(canvas.DurationStandard, func(done float32) {
		radius := maxRadius * done
		b.ripple.Position1 = center.SubtractXY(radius, radius)
		b.ripple.Position2 = center.AddXY(radius, radius)
		fade := uint8(float32(base.A) * (1 - done))
		b.ripple.FillColor = color.NRGBA{R: base.R, G: base.G, B: base.B, A: fade}
		canvas.Refresh(b.ripple)
		if done == 1 {
			b.ripple.Hidden = true
			canvas.Refresh(b.ripple)
		}
	})
	anim.Curve = fyne.AnimationEaseOut
	anim.Start()
}

func (b *circleIconButton) MouseDown(*desktop.MouseEvent) {
	b.repeating = false
	b.pressGen++
	gen := b.pressGen
	time.AfterFunc(circleButtonHoldDelay, func() {
		fyne.Do(func() { b.fireRepeat(gen) })
	})
}

func (b *circleIconButton) MouseUp(*desktop.MouseEvent) {
	b.pressGen++
}

func (b *circleIconButton) fireRepeat(gen int) {
	if gen != b.pressGen {
		return
	}
	b.repeating = true
	b.action()
	time.AfterFunc(circleButtonRepeatEvery, func() {
		fyne.Do(func() { b.fireRepeat(gen) })
	})
}

func (b *circleIconButton) MouseIn(*desktop.MouseEvent) {
	b.hovered = true
	b.Refresh()
}

func (b *circleIconButton) MouseMoved(*desktop.MouseEvent) {}

func (b *circleIconButton) MouseOut() {
	b.hovered = false
	b.pressGen++
	b.Refresh()
}

func (b *circleIconButton) Cursor() desktop.Cursor {
	return desktop.PointerCursor
}

type circleIconButtonRenderer struct {
	button  *circleIconButton
	icon    *canvas.Image
	objects []fyne.CanvasObject
}

func (r *circleIconButtonRenderer) Layout(size fyne.Size) {
	d := circleButtonDiameter
	bgOff := fyne.NewPos((size.Width-d)/2, (size.Height-d)/2)
	r.button.bg.Position1 = bgOff
	r.button.bg.Position2 = bgOff.AddXY(d, d)

	iconSize := d * 0.45
	iconOff := bgOff.AddXY((d-iconSize)/2, (d-iconSize)/2)
	r.icon.Resize(fyne.NewSquareSize(iconSize))
	r.icon.Move(iconOff)
}

func (r *circleIconButtonRenderer) MinSize() fyne.Size {
	return r.button.MinSize()
}

func (r *circleIconButtonRenderer) Refresh() {
	r.button.bg.FillColor = r.button.fillColor()
	canvas.Refresh(r.button.bg)
	r.icon.Refresh()
}

func (r *circleIconButtonRenderer) Objects() []fyne.CanvasObject {
	return r.objects
}

func (r *circleIconButtonRenderer) Destroy() {}
