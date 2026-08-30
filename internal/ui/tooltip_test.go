package ui

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

// These tests avoid letting the real tooltipDelay timer fire: its callback
// runs fyne.Do on its own goroutine, and the test driver's DoFromGoroutine
// executes inline rather than marshalling onto one thread, so a test that
// raced a live timer against later assertions on the same tip would be a
// genuine, race-detector-visible data race. Behaviour that depends on the
// timer actually firing is covered by floating_tip_test.go instead; these
// tests exercise scheduling/cancellation and the wiring to floatingTip.

// mouseEvent is a minimal non-nil event; hoverCatcher/showNear both read
// its Position to hit-test/place things, so a nil *desktop.MouseEvent
// (valid for the widget.Hoverable methods generally, but not here) would
// panic.
func mouseEvent() *desktop.MouseEvent {
	return &desktop.MouseEvent{PointEvent: fyne.PointEvent{Position: fyne.NewPos(0, 0)}}
}

func newTestTooltip(t *testing.T, text string) (*tooltip, *hoverCatcher) {
	t.Helper()
	tt := withTooltip(newFloatingTip(), widget.NewLabel("row"), text).(*tooltip)
	test.NewTempWindow(t, tt)
	return tt, tooltipCatcher(tt)
}

func TestTooltipMouseInSchedulesATimerWithoutShowingImmediately(t *testing.T) {
	test.NewTempApp(t)
	tt, catcher := newTestTooltip(t, "explains the row")

	genBefore := tt.showGen
	catcher.MouseIn(mouseEvent())

	if tt.showGen != genBefore+1 {
		t.Fatal("MouseIn should schedule a show")
	}
	if tt.tip.active {
		t.Fatal("the tip must not show before the delay elapses")
	}
	tt.cancelAndHide() // invalidate the pending timer so it can't fire after this test ends
}

func TestTooltipMouseOutCancelsThePendingTimer(t *testing.T) {
	test.NewTempApp(t)
	tt, catcher := newTestTooltip(t, "explains the row")

	genBefore := tt.showGen
	catcher.MouseIn(mouseEvent())
	catcher.MouseOut()

	if tt.showGen != genBefore+2 {
		t.Fatal("MouseOut should cancel the pending show")
	}
	if tt.tip.active {
		t.Fatal("the tip must not be active after a mouse-out")
	}
}

func TestTooltipMouseOutHidesAnAlreadyShowingTip(t *testing.T) {
	test.NewTempApp(t)
	tt, catcher := newTestTooltip(t, "explains the row")

	// Simulate the timer having already fired, without waiting on the real
	// background timer.
	tt.tip.showNear(tt.text, tt)
	if !tt.tip.active {
		t.Fatal("precondition: tip should be showing")
	}

	catcher.MouseOut()
	if tt.tip.active {
		t.Fatal("MouseOut should hide an already-showing tip")
	}
}

// tooltipCatcher pulls the hoverCatcher out of the tooltip's rendered tree:
// a single Stack container holding [content, catcher].
func tooltipCatcher(tt *tooltip) *hoverCatcher {
	stack := test.WidgetRenderer(tt).Objects()[0].(*fyne.Container)
	for _, o := range stack.Objects {
		if c, ok := o.(*hoverCatcher); ok {
			return c
		}
	}
	return nil
}
