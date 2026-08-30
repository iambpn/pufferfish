package ui

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/test"
)

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

// newShowNearTestWindow builds a window whose content mirrors
// clipboard_section.go's layout: target sits inside a VBox ("rows") which
// is itself nested a level deeper than tip - behind an offsetting padded
// layer - rather than as tip's own direct Stack sibling. showNear must
// resolve both to absolute canvas coordinates to place the tip correctly
// here; using target's raw Position() (relative to the VBox) would ignore
// the padded layer's own offset entirely.
func newShowNearTestWindow(tip *floatingTip, target fyne.CanvasObject) fyne.Window {
	spacer := canvas.NewRectangle(nil)
	spacer.SetMinSize(fyne.NewSize(10, 10))
	rows := container.NewVBox(spacer, target)
	nested := container.New(layout.NewCustomPaddedLayout(30, 0, 20, 0), rows)

	w := test.NewWindow(container.NewStack(nested, tip))
	w.Resize(fyne.NewSize(300, 300))
	return w
}

func TestShowNearPositionsBelowTheTarget(t *testing.T) {
	test.NewTempApp(t)
	tip := newFloatingTip()

	target := canvas.NewRectangle(nil)
	target.SetMinSize(fyne.NewSize(100, 30))

	w := newShowNearTestWindow(tip, target)
	defer w.Close()

	tip.showNear("hover text", target)

	if !tip.active {
		t.Fatal("showNear should activate the tip")
	}
	if tip.label.Text != "hover text" {
		t.Fatalf("label text = %q", tip.label.Text)
	}

	// The pre-fix formula (target.Position() alone) would place the tip at
	// the VBox-relative position, missing the padded layer's (20, 30)
	// offset entirely - assert against the true absolute placement instead.
	driver := fyne.CurrentApp().Driver()
	wantX := driver.AbsolutePositionForObject(target).X - driver.AbsolutePositionForObject(tip).X
	wantY := driver.AbsolutePositionForObject(target).Y - driver.AbsolutePositionForObject(tip).Y + target.Size().Height + 4
	want := fyne.NewPos(wantX, wantY)

	if tip.offset != want {
		t.Fatalf("offset = %v, want %v", tip.offset, want)
	}
	if naive := target.Position(); want.X == naive.X && want.Y == naive.Y+target.Size().Height+4 {
		t.Fatal("test is degenerate: nesting produced no offset from target's raw Position()")
	}
	if tip.bg.Position() != want {
		t.Fatalf("bg was not moved to the offset: got %v", tip.bg.Position())
	}
}

func TestHideMovesTheTipOffscreenAgain(t *testing.T) {
	test.NewTempApp(t)
	tip := newFloatingTip()

	target := canvas.NewRectangle(nil)
	target.SetMinSize(fyne.NewSize(50, 20))
	w := newShowNearTestWindow(tip, target)
	defer w.Close()

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
