package ui

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

// fakeTarget is a minimal CanvasObject with a fixed position and size, so
// showNear can be exercised without a full render tree.
type fakeTarget struct {
	widget.BaseWidget
	pos  fyne.Position
	size fyne.Size
}

func (f *fakeTarget) Position() fyne.Position { return f.pos }
func (f *fakeTarget) Size() fyne.Size         { return f.size }

func TestFloatingTipStartsOffscreenAndHidden(t *testing.T) {
	test.NewTempApp(t)
	tip := newFloatingTip()
	test.WidgetRenderer(tip)
	tip.Refresh() // forces the renderer's initial Layout

	if tip.active {
		t.Fatal("a new tip should not be active")
	}
	if tip.bg.Position() != offscreen {
		t.Fatalf("bg position = %v, want offscreen", tip.bg.Position())
	}
}

func TestShowNearPositionsBelowTheTarget(t *testing.T) {
	test.NewTempApp(t)
	tip := newFloatingTip()
	test.WidgetRenderer(tip)

	target := &fakeTarget{pos: fyne.NewPos(10, 20), size: fyne.NewSize(100, 30)}
	tip.showNear("hover text", target)

	if !tip.active {
		t.Fatal("showNear should activate the tip")
	}
	if tip.label.Text != "hover text" {
		t.Fatalf("label text = %q", tip.label.Text)
	}
	want := fyne.NewPos(10, 54) // target.Y + target.Height + 4
	if tip.offset != want {
		t.Fatalf("offset = %v, want %v", tip.offset, want)
	}
	if tip.bg.Position() != want {
		t.Fatalf("bg was not moved to the offset: got %v", tip.bg.Position())
	}
}

func TestHideMovesTheTipOffscreenAgain(t *testing.T) {
	test.NewTempApp(t)
	tip := newFloatingTip()
	test.WidgetRenderer(tip)

	target := &fakeTarget{pos: fyne.NewPos(0, 0), size: fyne.NewSize(50, 20)}
	tip.showNear("visible", target)
	tip.hide()

	if tip.active {
		t.Fatal("hide should clear active")
	}
	if tip.bg.Position() != offscreen {
		t.Fatalf("bg position = %v, want offscreen", tip.bg.Position())
	}
}

func TestFloatingTipMinSizeIsZero(t *testing.T) {
	test.NewTempApp(t)
	tip := newFloatingTip()
	renderer := test.WidgetRenderer(tip)
	if got := renderer.MinSize(); got != (fyne.NewSize(0, 0)) {
		t.Fatalf("got %v", got)
	}
}
