package window

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// positioner is the slice of desktop.Window the mover needs: a request to
// place the native window at an on-screen coordinate.
type positioner interface {
	RequestPosition(x, y int)
}

// windowMover wraps the content of an undecorated window and turns a drag
// on any non-interactive part of it into a window move, standing in for the
// title bar the window doesn't have.
//
// Fyne reports drag positions relative to the window, so as soon as the
// window starts following the pointer those coordinates shift under it.
// Adding up the per-event delta then makes the window stutter or oscillate.
// Instead the mover pins the point where the drag was grabbed and, each
// event, shifts the window by how far the pointer has strayed from that
// point; once the window catches up the difference is back to zero, so it
// follows the pointer without chasing it.
type windowMover struct {
	widget.BaseWidget
	content fyne.CanvasObject
	win     positioner

	// x, y is the window's current origin in the coordinate space
	// RequestPosition expects. Seeded by the caller because Fyne can't read
	// a window's position back.
	x, y float32

	grabX, grabY float32 // pointer offset within the window when the drag began
	dragging     bool

	onMoved func(x, y int)
}

func newWindowMover(content fyne.CanvasObject, win positioner, x, y int, onMoved func(x, y int)) *windowMover {
	m := &windowMover{
		content: content,
		win:     win,
		x:       float32(x),
		y:       float32(y),
		onMoved: onMoved,
	}
	m.ExtendBaseWidget(m)
	return m
}

func (m *windowMover) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(m.content)
}

// scale converts Fyne's device-independent coordinates to the pixels
// RequestPosition works in. It's 1 on a standard-DPI display.
func (m *windowMover) scale() float32 {
	app := fyne.CurrentApp()
	if app == nil || app.Driver() == nil {
		return 1
	}
	if c := app.Driver().CanvasForObject(m); c != nil {
		if s := c.Scale(); s > 0 {
			return s
		}
	}
	return 1
}

func (m *windowMover) Dragged(e *fyne.DragEvent) {
	px := e.AbsolutePosition.X * m.scale()
	py := e.AbsolutePosition.Y * m.scale()

	if !m.dragging {
		// First event of this drag: remember where on the window the user
		// grabbed and don't move yet.
		m.grabX, m.grabY = px, py
		m.dragging = true
		return
	}

	m.x += px - m.grabX
	m.y += py - m.grabY
	m.win.RequestPosition(int(m.x), int(m.y))
}

func (m *windowMover) DragEnd() {
	m.dragging = false
	if m.onMoved != nil {
		m.onMoved(int(m.x), int(m.y))
	}
}
