package ui

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
)

// newTestCircleButton forces CreateRenderer so the ripple/background fields
// used by Tapped and the hover handlers are populated.
func newTestCircleButton(action func()) *circleIconButton {
	b := newCircleIconButton(theme.CancelIcon(), action)
	test.WidgetRenderer(b)
	return b
}

func TestCircleButtonTapInvokesAction(t *testing.T) {
	test.NewTempApp(t)
	calls := 0
	b := newTestCircleButton(func() { calls++ })

	test.Tap(b)

	if calls != 1 {
		t.Fatalf("want 1 call, got %d", calls)
	}
}

func TestCircleButtonHoverTracksState(t *testing.T) {
	test.NewTempApp(t)
	b := newTestCircleButton(func() {})

	b.MouseIn(nil)
	if !b.hovered {
		t.Fatal("hovered should be true after MouseIn")
	}

	b.MouseOut()
	if b.hovered {
		t.Fatal("hovered should be false after MouseOut")
	}
}

func TestCircleButtonCursorIsPointer(t *testing.T) {
	b := newCircleIconButton(theme.CancelIcon(), func() {})
	if b.Cursor() != desktop.PointerCursor {
		t.Fatalf("got %v", b.Cursor())
	}
}

func TestCircleButtonMinSizeIsTheDiameter(t *testing.T) {
	b := newCircleIconButton(theme.CancelIcon(), func() {})
	want := fyne.NewSquareSize(circleButtonDiameter)
	if got := b.MinSize(); got != want {
		t.Fatalf("got %v, want %v", got, want)
	}
}

// The hold-and-repeat behaviour is driven by MouseDown scheduling
// time.AfterFunc callbacks that call back into fyne.Do on their own
// goroutine. The test driver's DoFromGoroutine runs those inline rather
// than marshalling them onto one thread the way a real driver does, so a
// test that let the real timer fire and then read the button's state from
// the test goroutine would be a genuine, race-detector-visible data race.
// These tests instead call fireRepeat directly - the same code the timer
// would call - to exercise the logic deterministically.

func TestFireRepeatInvokesTheActionAndFlagsRepeating(t *testing.T) {
	test.NewTempApp(t)
	calls := 0
	b := newTestCircleButton(func() { calls++ })

	b.MouseDown(nil) // sets pressGen to the generation fireRepeat expects
	b.fireRepeat(b.pressGen)

	if calls != 1 {
		t.Fatalf("want 1 call, got %d", calls)
	}
	if !b.repeating {
		t.Fatal("fireRepeat should flag the button as repeating")
	}

	// fireRepeat reschedules itself; advance pressGen so that pending
	// callback becomes a no-op instead of leaking a live repeat loop.
	b.MouseUp(nil)
}

func TestTapEndingAHoldDoesNotFireTheActionAgain(t *testing.T) {
	test.NewTempApp(t)
	calls := 0
	b := newTestCircleButton(func() { calls++ })

	b.MouseDown(nil)
	b.fireRepeat(b.pressGen) // simulates the hold delay having elapsed

	before := calls
	test.Tap(b) // the mouse-up ending a hold delivers a Tapped event
	if calls != before {
		t.Fatalf("tap ending a hold fired the action again: %d -> %d", before, calls)
	}
	if b.repeating {
		t.Fatal("the tap ending a hold should clear the repeating flag")
	}

	// fireRepeat reschedules itself; advance pressGen so that pending
	// callback becomes a no-op instead of leaking a live repeat loop.
	b.MouseUp(nil)
}

func TestMouseUpMakesALaterFireRepeatANoOp(t *testing.T) {
	test.NewTempApp(t)
	calls := 0
	b := newTestCircleButton(func() { calls++ })

	b.MouseDown(nil)
	gen := b.pressGen
	b.MouseUp(nil) // advances pressGen, invalidating the scheduled callback

	b.fireRepeat(gen) // what the now-stale timer callback would run
	if calls != 0 {
		t.Fatalf("a repeat scheduled before MouseUp must not fire, got %d calls", calls)
	}
}

func TestMouseOutMakesAPendingHoldANoOp(t *testing.T) {
	test.NewTempApp(t)
	calls := 0
	b := newTestCircleButton(func() { calls++ })

	b.MouseDown(nil)
	gen := b.pressGen
	b.MouseOut() // advances pressGen, invalidating the scheduled callback

	b.fireRepeat(gen) // what the now-stale timer callback would run
	if calls != 0 {
		t.Fatalf("moving out before the hold delay must cancel the repeat, got %d calls", calls)
	}
}
