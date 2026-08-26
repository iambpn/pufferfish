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
	resetSizeCache()
	t.Cleanup(resetSizeCache)

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
	// A prior test may have already cached a real result; without clearing
	// it here, Size would answer from that cache instead of exercising the
	// no-$DISPLAY path this test means to check.
	resetSizeCache()
	t.Cleanup(resetSizeCache)

	t.Setenv("DISPLAY", "")
	if _, _, ok := Size(); ok {
		t.Fatal("want ok=false with no $DISPLAY set")
	}
}

func TestSizeReturnsTheCachedValueWithoutQuerying(t *testing.T) {
	resetSizeCache()
	t.Cleanup(resetSizeCache)

	sizeCacheMu.Lock()
	sizeCacheValid, sizeCacheW, sizeCacheH = true, 1234, 5678
	sizeCacheMu.Unlock()

	// No $DISPLAY: a fresh query would fail, so a successful result here
	// can only have come from the cache.
	t.Setenv("DISPLAY", "")
	w, h, ok := Size()
	if !ok || w != 1234 || h != 5678 {
		t.Fatalf("got (%d, %d, %v), want the cached (1234, 5678, true)", w, h, ok)
	}
}

// TestPrimaryMonitorSizeAgainstRealXServer exercises the RandR path
// specifically (Size falls back to the root window's own geometry when
// this fails, which TestSizeAgainstRealXServer alone wouldn't catch a
// regression in - a broken RandR path could still report a plausible
// number by degrading silently to that fallback).
func TestPrimaryMonitorSizeAgainstRealXServer(t *testing.T) {
	if os.Getenv("DISPLAY") == "" {
		t.Skip("no $DISPLAY: no X server to query")
	}

	conn, root, err := dial()
	if err != nil {
		t.Skipf("no X server reachable at $DISPLAY: %v", err)
	}
	defer conn.Close()

	w, h, ok := primaryMonitorSize(conn, root)
	if !ok {
		t.Skip("RandR primary monitor size unavailable on this display")
	}
	if w <= 0 || h <= 0 {
		t.Fatalf("got non-positive size (%d, %d)", w, h)
	}
	t.Logf("primary monitor size (%d, %d)", w, h)
}

func TestQueryExtensionReportsAbsentForAnUnknownName(t *testing.T) {
	if os.Getenv("DISPLAY") == "" {
		t.Skip("no $DISPLAY: no X server to query")
	}

	conn, _, err := dial()
	if err != nil {
		t.Skipf("no X server reachable at $DISPLAY: %v", err)
	}
	defer conn.Close()

	if _, ok := queryExtension(conn, "NOT-A-REAL-EXTENSION"); ok {
		t.Fatal("want ok=false for an extension the server doesn't have")
	}
}
