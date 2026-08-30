package ui

import (
	"image/color"

	"fyne.io/fyne/v2/theme"
)

// hoverBackground tracks whether the pointer is over a widget and resolves
// that to the fill color its background should show, shared by every
// widget that swaps between the theme's input-background and hover colors
// on MouseIn/MouseOut (circleIconButton, smallButton).
type hoverBackground struct {
	hovered bool
}

// fillColor is the background color for the current hover state.
func (h *hoverBackground) fillColor() color.Color {
	if h.hovered {
		return theme.Color(theme.ColorNameHover)
	}
	return theme.Color(theme.ColorNameInputBackground)
}
