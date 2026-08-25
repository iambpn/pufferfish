/*
tooltip wraps content so hovering it for a moment shows a short explanation
in the shared floating tip. An invisible hoverCatcher sits above the content
only to track the mouse, so it never steals taps from interactive content
underneath (e.g. a checkbox).

hoverCatcher exploits how Fyne resolves hover: when several Hoverable
objects overlap at the same point, the last one visited in the render tree
wins. Since it's placed after content in the Stack, it always wins hover
there - but because it does not implement Tappable, taps still fall through
to content beneath it.
*/
package ui

import (
	"image/color"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

const tooltipDelay = 500 * time.Millisecond

type tooltip struct {
	widget.BaseWidget

	content fyne.CanvasObject
	text    string
	tip     *floatingTip
	timer   *time.Timer
}

// withTooltip wraps content so hovering it shows text in the shared tip.
func withTooltip(tip *floatingTip, content fyne.CanvasObject, text string) fyne.CanvasObject {
	t := &tooltip{content: content, text: text, tip: tip}
	t.ExtendBaseWidget(t)
	return t
}

func (t *tooltip) CreateRenderer() fyne.WidgetRenderer {
	catcher := newHoverCatcher(t.scheduleShow, t.cancelAndHide)
	return widget.NewSimpleRenderer(container.NewStack(t.content, catcher))
}

func (t *tooltip) scheduleShow() {
	t.cancelTimer()
	t.timer = time.AfterFunc(tooltipDelay, func() {
		fyne.Do(func() {
			t.tip.showNear(t.text, t)
		})
	})
}

func (t *tooltip) cancelAndHide() {
	t.cancelTimer()
	t.tip.hide()
}

func (t *tooltip) cancelTimer() {
	if t.timer != nil {
		t.timer.Stop()
		t.timer = nil
	}
}

type hoverCatcher struct {
	widget.BaseWidget

	onEnter func()
	onExit  func()
}

func newHoverCatcher(onEnter, onExit func()) *hoverCatcher {
	h := &hoverCatcher{onEnter: onEnter, onExit: onExit}
	h.ExtendBaseWidget(h)
	return h
}

func (h *hoverCatcher) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(canvas.NewRectangle(color.Transparent))
}

func (h *hoverCatcher) MouseIn(*desktop.MouseEvent) {
	if h.onEnter != nil {
		h.onEnter()
	}
}

func (h *hoverCatcher) MouseMoved(*desktop.MouseEvent) {}

func (h *hoverCatcher) MouseOut() {
	if h.onExit != nil {
		h.onExit()
	}
}
