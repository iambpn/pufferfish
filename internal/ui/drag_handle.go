package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// dragHandle is a small pill-shaped bar that reports drag deltas. It stands
// in for an OS title bar so an undecorated window can still be moved.
type dragHandle struct {
	widget.BaseWidget

	bar       *canvas.Rectangle
	onDragged func(dx, dy float32)
	onDragEnd func()
}

func newDragHandle(onDragged func(dx, dy float32), onDragEnd func()) *dragHandle {
	h := &dragHandle{onDragged: onDragged, onDragEnd: onDragEnd}
	h.bar = canvas.NewRectangle(theme.Color(theme.ColorNameDisabled))
	h.bar.CornerRadius = 2
	h.bar.SetMinSize(fyne.NewSize(50, 4))
	h.ExtendBaseWidget(h)
	return h
}

func (h *dragHandle) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(h.bar)
}

func (h *dragHandle) Dragged(ev *fyne.DragEvent) {
	if h.onDragged != nil {
		h.onDragged(ev.Dragged.DX, ev.Dragged.DY)
	}
}

func (h *dragHandle) DragEnd() {
	if h.onDragEnd != nil {
		h.onDragEnd()
	}
}
