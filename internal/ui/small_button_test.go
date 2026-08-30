package ui

import (
	"testing"

	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
)

func TestSmallButtonTapInvokesOnTapped(t *testing.T) {
	test.NewTempApp(t)
	calls := 0
	b := newSmallButton("Clear all", func() { calls++ })
	test.WidgetRenderer(b)

	test.Tap(b)

	if calls != 1 {
		t.Fatalf("want 1 call, got %d", calls)
	}
}

func TestSmallButtonTapWithNilCallbackDoesNotPanic(t *testing.T) {
	test.NewTempApp(t)
	b := newSmallButton("Clear all", nil)
	test.WidgetRenderer(b)

	test.Tap(b)
}

func TestSmallButtonCursorIsPointer(t *testing.T) {
	b := newSmallButton("Clear all", func() {})
	if b.Cursor() != desktop.PointerCursor {
		t.Fatalf("got %v", b.Cursor())
	}
}

func TestSmallButtonHoverChangesBackground(t *testing.T) {
	test.NewTempApp(t)
	b := newSmallButton("Clear all", func() {})
	test.WidgetRenderer(b)

	original := b.bg.FillColor
	b.MouseIn(nil)
	if b.bg.FillColor != theme.Color(theme.ColorNameHover) {
		t.Fatalf("background did not switch to the hover color: got %v", b.bg.FillColor)
	}

	b.MouseOut()
	if b.bg.FillColor != original {
		t.Fatalf("background did not restore after MouseOut: got %v, want %v", b.bg.FillColor, original)
	}
}
