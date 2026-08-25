// Package screen reports the primary monitor's size in pixels.
//
// Fyne has no portable API for this - Window.CenterOnScreen exists, but it
// doesn't expose the numbers it centers with - so each platform gets its
// own small backend calling straight into the OS.
package screen
