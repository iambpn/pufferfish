package window

import "fyne.io/fyne/v2"

// singleWindow guards a lazily-built window so repeated calls to Open reuse
// and focus the one already showing instead of building a second one.
type singleWindow struct {
	win fyne.Window
}

// Open shows the window, building and showing a fresh one via build the
// first time (or once the previous one has closed), and just focusing the
// existing one on repeat calls.
//
// build must wire its own cleanup onto onClosed, rather than calling
// w.SetOnClosed itself, since Open needs that same hook to reset the guard.
func (s *singleWindow) Open(build func(onClosed func()) fyne.Window) {
	if s.win != nil {
		s.win.RequestFocus()
		return
	}
	s.win = build(func() { s.win = nil })
}
