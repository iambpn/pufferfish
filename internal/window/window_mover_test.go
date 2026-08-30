package window

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// fakeWindow stands in for the native window: it records the last requested
// origin so a test can feed it back as the window position.
type fakeWindow struct {
	x, y  int
	calls int
}

func (f *fakeWindow) RequestPosition(x, y int) {
	f.x, f.y = x, y
	f.calls++
}

// dragTo simulates the pointer being at screen coordinate (mx, my) with the
// window at its current origin. Fyne reports the position relative to the
// window, so that's what the mover receives.
func dragTo(m *windowMover, f *fakeWindow, mx, my float32) {
	m.Dragged(&fyne.DragEvent{PointEvent: fyne.PointEvent{
		AbsolutePosition: fyne.NewPos(mx-float32(f.x), my-float32(f.y)),
	}})
}

func TestWindowMoverFirstEventOnlyGrabs(t *testing.T) {
	f := &fakeWindow{x: 80, y: 80}
	m := newWindowMover(widget.NewLabel("x"), f, 80, 80, nil)

	dragTo(m, f, 200, 200)

	if f.calls != 0 {
		t.Fatalf("RequestPosition called %d times on the grab event, want 0", f.calls)
	}
}

func TestWindowMoverTracksPointerAcrossEvents(t *testing.T) {
	f := &fakeWindow{x: 80, y: 80}
	m := newWindowMover(widget.NewLabel("x"), f, 80, 80, nil)

	dragTo(m, f, 200, 200) // grab
	dragTo(m, f, 260, 230) // pointer moved +60, +30
	dragTo(m, f, 300, 230) // pointer moved another +40, +0

	// The window should have shifted by the total pointer displacement.
	if f.x != 180 || f.y != 110 {
		t.Fatalf("window origin = (%d, %d), want (180, 110)", f.x, f.y)
	}
}

func TestWindowMoverDoesNotDriftWhenPointerHolds(t *testing.T) {
	f := &fakeWindow{x: 80, y: 80}
	m := newWindowMover(widget.NewLabel("x"), f, 80, 80, nil)

	dragTo(m, f, 200, 200) // grab
	dragTo(m, f, 240, 240) // move
	dragTo(m, f, 240, 240) // same spot again
	dragTo(m, f, 240, 240)

	if f.x != 120 || f.y != 120 {
		t.Fatalf("window origin = (%d, %d), want (120, 120)", f.x, f.y)
	}
}

func TestWindowMoverReportsRestingPositionOnDragEnd(t *testing.T) {
	f := &fakeWindow{x: 80, y: 80}
	var gotX, gotY int
	called := false
	m := newWindowMover(widget.NewLabel("x"), f, 80, 80, func(x, y int) {
		gotX, gotY, called = x, y, true
	})

	dragTo(m, f, 200, 200)
	dragTo(m, f, 250, 170)
	m.DragEnd()

	if !called {
		t.Fatal("onMoved was not called")
	}
	if gotX != f.x || gotY != f.y {
		t.Fatalf("onMoved got (%d, %d), window origin (%d, %d)", gotX, gotY, f.x, f.y)
	}
}

func TestWindowMoverReGrabsForASecondDrag(t *testing.T) {
	f := &fakeWindow{x: 80, y: 80}
	m := newWindowMover(widget.NewLabel("x"), f, 80, 80, nil)

	dragTo(m, f, 200, 200)
	dragTo(m, f, 230, 200)
	m.DragEnd()
	firstX := f.x

	// A new drag starting somewhere else must not jump the window.
	dragTo(m, f, 500, 500) // grab only
	if f.x != firstX {
		t.Fatalf("window jumped on re-grab: %d -> %d", firstX, f.x)
	}
	dragTo(m, f, 510, 500)
	if f.x != firstX+10 {
		t.Fatalf("second drag moved to %d, want %d", f.x, firstX+10)
	}
}
