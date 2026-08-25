//go:build (linux || freebsd || openbsd || netbsd) && !android

package screen

import (
	"os"
	"testing"
)

// TestSizeAgainstRealXServer exercises the actual X11 wire protocol against
// whatever display $DISPLAY names. It's an integration check rather than a
// unit test - there's no server to fake without reimplementing the protocol
// on both ends - so it skips cleanly wherever no display is reachable
// instead of failing the build.
func TestSizeAgainstRealXServer(t *testing.T) {
	if os.Getenv("DISPLAY") == "" {
		t.Skip("no $DISPLAY: no X server to query")
	}

	w, h, ok := Size()
	if !ok {
		t.Skip("no X server reachable at $DISPLAY")
	}
	if w <= 0 || h <= 0 {
		t.Fatalf("got non-positive size (%d, %d)", w, h)
	}
	t.Logf("screen size (%d, %d)", w, h)
}

func TestSizeFailsGracefullyWithoutADisplay(t *testing.T) {
	t.Setenv("DISPLAY", "")
	if _, _, ok := Size(); ok {
		t.Fatal("want ok=false with no $DISPLAY set")
	}
}
