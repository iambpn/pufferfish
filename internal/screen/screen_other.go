//go:build !windows && !((linux || freebsd || openbsd || netbsd) && !android)

package screen

// Size reports ok=false: Pufferfish has no screen-size backend for this
// platform yet, so callers fall back to a fixed default position.
func Size() (width, height int, ok bool) {
	return 0, 0, false
}
