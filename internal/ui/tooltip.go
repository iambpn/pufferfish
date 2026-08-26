/*
tooltip wraps content so hovering it for a moment shows a short explanation
in the shared floating tip. An invisible hoverCatcher sits above the content
only to track the mouse, so it never steals taps from interactive content
underneath (e.g. a checkbox).

hoverCatcher exploits how Fyne resolves hover: when several Hoverable
objects overlap at the same point, the last one visited in the render tree
wins. Since it's placed after content in the Stack, it always wins hover
there - but because it does not implement Tappable, taps still fall through
to content beneath it. Any Hoverable nested inside content (e.g. a button's
own hover highlight) would otherwise never fire while wrapped, so
hoverCatcher hit-tests content's subtree itself and forwards
MouseIn/MouseMoved/MouseOut to whichever Hoverable descendant sits under
the pointer, in addition to driving the tooltip.
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
	showGen int
}

// withTooltip wraps content so hovering it shows text in the shared tip.
func withTooltip(tip *floatingTip, content fyne.CanvasObject, text string) fyne.CanvasObject {
	t := &tooltip{content: content, text: text, tip: tip}
	t.ExtendBaseWidget(t)
	return t
}

func (t *tooltip) CreateRenderer() fyne.WidgetRenderer {
	catcher := newHoverCatcher(t.content, t.scheduleShow, t.cancelAndHide)
	return widget.NewSimpleRenderer(container.NewStack(t.content, catcher))
}

// scheduleShow arms the shared tip to appear after tooltipDelay. Stopping a
// time.AfterFunc timer can't reliably cancel a callback that has already
// started firing, so instead each call is tagged with a generation number;
// the callback checks it's still current before touching the shared tip,
// the same guard circleIconButton uses for its own delayed callback.
func (t *tooltip) scheduleShow() {
	t.showGen++
	gen := t.showGen
	time.AfterFunc(tooltipDelay, func() {
		fyne.Do(func() {
			if gen != t.showGen {
				return
			}
			t.tip.showNear(t.text, t)
		})
	})
}

func (t *tooltip) cancelAndHide() {
	t.showGen++
	t.tip.hide()
}

type hoverCatcher struct {
	widget.BaseWidget

	content fyne.CanvasObject
	onEnter func()
	onExit  func()

	hovered desktop.Hoverable
}

func newHoverCatcher(content fyne.CanvasObject, onEnter, onExit func()) *hoverCatcher {
	h := &hoverCatcher{content: content, onEnter: onEnter, onExit: onExit}
	h.ExtendBaseWidget(h)
	return h
}

func (h *hoverCatcher) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(canvas.NewRectangle(color.Transparent))
}

func (h *hoverCatcher) MouseIn(e *desktop.MouseEvent) {
	h.updateHovered(e)
	if h.onEnter != nil {
		h.onEnter()
	}
}

func (h *hoverCatcher) MouseMoved(e *desktop.MouseEvent) {
	h.updateHovered(e)
}

func (h *hoverCatcher) MouseOut() {
	if h.hovered != nil {
		h.hovered.MouseOut()
		h.hovered = nil
	}
	if h.onExit != nil {
		h.onExit()
	}
}

// updateHovered forwards MouseIn/MouseMoved/MouseOut to whichever Hoverable
// descendant of content sits under e, since hoverCatcher - not that
// descendant - is what Fyne actually dispatches hover events to.
func (h *hoverCatcher) updateHovered(e *desktop.MouseEvent) {
	found := hoverableAt(h.content, e.Position)
	if found != h.hovered {
		if h.hovered != nil {
			h.hovered.MouseOut()
		}
		h.hovered = found
		if found != nil {
			found.MouseIn(e)
		}
		return
	}
	if found != nil {
		found.MouseMoved(e)
	}
}

// hoverableAt returns the innermost Hoverable in obj's subtree whose bounds
// contain pos, which is given in obj's own coordinate space. It descends
// into *fyne.Container children (recomputing pos into each child's local
// space) so a Hoverable nested arbitrarily deep - not just a direct child -
// is still found.
func hoverableAt(obj fyne.CanvasObject, pos fyne.Position) desktop.Hoverable {
	if obj == nil || !obj.Visible() {
		return nil
	}
	size := obj.Size()
	if pos.X < 0 || pos.Y < 0 || pos.X > size.Width || pos.Y > size.Height {
		return nil
	}

	if c, ok := obj.(*fyne.Container); ok {
		for _, child := range c.Objects {
			if found := hoverableAt(child, pos.Subtract(child.Position())); found != nil {
				return found
			}
		}
	}

	if h, ok := obj.(desktop.Hoverable); ok {
		return h
	}
	return nil
}
