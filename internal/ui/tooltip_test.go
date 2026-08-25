package ui

import (
	"testing"

	"fyne.io/fyne/v2"
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

func newTestTooltip(text string) (*tooltip, *hoverCatcher) {
	tt := withTooltip(newFloatingTip(), widget.NewLabel("row"), text).(*tooltip)
	test.WidgetRenderer(tt)
	return tt, tooltipCatcher(tt)
}

func TestTooltipMouseInSchedulesATimerWithoutShowingImmediately(t *testing.T) {
	test.NewTempApp(t)
	tt, catcher := newTestTooltip("explains the row")

	catcher.MouseIn(nil)

	if tt.timer == nil {
		t.Fatal("MouseIn should schedule a timer")
	}
	if tt.tip.active {
		t.Fatal("the tip must not show before the delay elapses")
	}
	tt.cancelTimer()
}

func TestTooltipMouseOutCancelsThePendingTimer(t *testing.T) {
	test.NewTempApp(t)
	tt, catcher := newTestTooltip("explains the row")

	catcher.MouseIn(nil)
	catcher.MouseOut()

	if tt.timer != nil {
		t.Fatal("MouseOut should cancel the pending show")
	}
	if tt.tip.active {
		t.Fatal("the tip must not be active after a mouse-out")
	}
}

func TestTooltipMouseOutHidesAnAlreadyShowingTip(t *testing.T) {
	test.NewTempApp(t)
	tt, catcher := newTestTooltip("explains the row")

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
