package clipboard

import (
	"errors"
	"image/color"
	"testing"
	"time"

	"fyne.io/fyne/v2/test"
	system "golang.design/x/clipboard"
)

func TestNewWatcherStartsWithoutCapturingImages(t *testing.T) {
	w := NewWatcher(NewStore(t.TempDir()))
	if w.capturesImages() {
		t.Fatal("a new watcher should not capture images by default")
	}
}

func TestSetCaptureImagesTogglesTheFlag(t *testing.T) {
	w := NewWatcher(NewStore(t.TempDir()))

	w.SetCaptureImages(true)
	if !w.capturesImages() {
		t.Fatal("want captureImages = true")
	}

	w.SetCaptureImages(false)
	if w.capturesImages() {
		t.Fatal("want captureImages = false")
	}
}

func TestHandleIgnoresEmptyData(t *testing.T) {
	test.NewTempApp(t)
	store := NewStore(t.TempDir())
	w := NewWatcher(store)

	w.handle(system.Data{Format: system.FmtText, Bytes: nil})

	if got := store.Items(); len(got) != 0 {
		t.Fatalf("got %+v", got)
	}
}

func TestHandleAddsCapturedText(t *testing.T) {
	test.NewTempApp(t)
	store := NewStore(t.TempDir())
	t.Cleanup(store.Flush)
	w := NewWatcher(store)

	w.handle(system.Data{Format: system.FmtText, Bytes: []byte("copied text")})

	items := store.Items()
	if len(items) != 1 || items[0].Text != "copied text" {
		t.Fatalf("got %+v", items)
	}
}

func TestHandleDropsImagesWhenCaptureIsOff(t *testing.T) {
	test.NewTempApp(t)
	store := NewStore(t.TempDir())
	w := NewWatcher(store)
	w.SetCaptureImages(false)

	w.handle(system.Data{Format: system.FmtImage, Bytes: pngBytes(t, 2, 2, color.White)})

	if got := store.Items(); len(got) != 0 {
		t.Fatalf("got %+v", got)
	}
}

func TestHandleAddsImagesWhenCaptureIsOn(t *testing.T) {
	test.NewTempApp(t)
	store := NewStore(t.TempDir())
	t.Cleanup(store.Flush)
	w := NewWatcher(store)
	w.SetCaptureImages(true)

	w.handle(system.Data{Format: system.FmtImage, Bytes: pngBytes(t, 3, 3, color.White)})

	items := store.Items()
	if len(items) != 1 || items[0].Kind != KindImage {
		t.Fatalf("got %+v", items)
	}
}

func TestWatcherStartStopLifecycle(t *testing.T) {
	if err := Init(); err != nil {
		t.Skipf("system clipboard unavailable: %v", err)
	}

	w := NewWatcher(NewStore(t.TempDir()))

	w.Start()
	w.Start() // starting an already-running watcher must be a no-op
	time.Sleep(20 * time.Millisecond)

	w.Stop()
	w.Stop() // stopping an already-stopped watcher must be a no-op
}

func TestSetEnabledStartsAndStopsTheWatcher(t *testing.T) {
	if err := Init(); err != nil {
		t.Skipf("system clipboard unavailable: %v", err)
	}

	w := NewWatcher(NewStore(t.TempDir()))

	w.SetEnabled(true)
	time.Sleep(20 * time.Millisecond)
	w.SetEnabled(false)
}

// watcherState reports whether Pufferfish's own content is on the clipboard
// for w, and whether the watch of w runs.
func watcherState(w *Watcher) (owning, running bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.held != 0, w.cancel != nil
}

// waitUntil fails the test when cond does not become true in time.
func waitUntil(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("condition was not met in time")
}

func TestOwnWriteHoldsTheClipboardUntilItIsReplaced(t *testing.T) {
	w := NewWatcher(NewStore(t.TempDir()))

	replaced := make(chan struct{})
	err := w.ownClipboard(func() (<-chan struct{}, error) { return replaced, nil })
	if err != nil {
		t.Fatalf("ownClipboard failed: %v", err)
	}
	if owning, _ := watcherState(w); !owning {
		t.Fatal("want the watcher to hold the clipboard after its own write")
	}

	close(replaced)
	waitUntil(t, func() bool {
		owning, _ := watcherState(w)
		return !owning
	})
}

func TestReplacedOlderWriteDoesNotReleaseANewerWrite(t *testing.T) {
	w := NewWatcher(NewStore(t.TempDir()))

	first := make(chan struct{})
	second := make(chan struct{})
	w.ownClipboard(func() (<-chan struct{}, error) { return first, nil })
	w.ownClipboard(func() (<-chan struct{}, error) { return second, nil })

	// The second write replaced the first one. The clipboard still holds
	// Pufferfish's own content.
	close(first)
	time.Sleep(50 * time.Millisecond)
	if owning, _ := watcherState(w); !owning {
		t.Fatal("the end of an older write must not release the clipboard")
	}

	close(second)
	waitUntil(t, func() bool {
		owning, _ := watcherState(w)
		return !owning
	})
}

func TestFailedOwnWriteReleasesTheClipboard(t *testing.T) {
	w := NewWatcher(NewStore(t.TempDir()))

	want := errors.New("write failed")
	err := w.ownClipboard(func() (<-chan struct{}, error) { return nil, want })
	if !errors.Is(err, want) {
		t.Fatalf("want the write error, got %v", err)
	}
	if owning, _ := watcherState(w); owning {
		t.Fatal("a failed write must not hold the clipboard")
	}
}

func TestFailedOwnWriteKeepsTheEarlierWriteOnTheClipboard(t *testing.T) {
	w := NewWatcher(NewStore(t.TempDir()))

	first := make(chan struct{})
	w.ownClipboard(func() (<-chan struct{}, error) { return first, nil })

	// The second write fails, so the content of the first write is still
	// on the clipboard.
	w.ownClipboard(func() (<-chan struct{}, error) { return nil, errors.New("write failed") })
	if owning, _ := watcherState(w); !owning {
		t.Fatal("a failed write must not release the earlier write")
	}

	close(first)
	waitUntil(t, func() bool {
		owning, _ := watcherState(w)
		return !owning
	})
}

func TestFailedOwnWriteReleasesAWriteThatWasReplacedMeanwhile(t *testing.T) {
	w := NewWatcher(NewStore(t.TempDir()))

	first := make(chan struct{})
	w.ownClipboard(func() (<-chan struct{}, error) { return first, nil })

	// Another application replaces the first write while the second write
	// runs, and the second write then fails.
	w.ownClipboard(func() (<-chan struct{}, error) {
		close(first)
		waitUntil(t, func() bool {
			owning, _ := watcherState(w)
			return !owning
		})
		return nil, errors.New("write failed")
	})

	if owning, _ := watcherState(w); owning {
		t.Fatal("no own content is on the clipboard after the failed write")
	}
}

func TestStartWaitsWhileOwnContentIsOnTheClipboard(t *testing.T) {
	w := NewWatcher(NewStore(t.TempDir()))

	replaced := make(chan struct{})
	w.ownClipboard(func() (<-chan struct{}, error) { return replaced, nil })

	w.Start()
	if _, running := watcherState(w); running {
		t.Fatal("the watch must not run while own content is on the clipboard")
	}

	// A stopped watcher stays stopped when the own content is replaced.
	w.Stop()
	close(replaced)
	waitUntil(t, func() bool {
		owning, _ := watcherState(w)
		return !owning
	})
	if _, running := watcherState(w); running {
		t.Fatal("a stopped watcher must not start again")
	}
}

func TestOwnWritePausesTheWatchAndReplacementContinuesIt(t *testing.T) {
	if err := Init(); err != nil {
		t.Skipf("system clipboard unavailable: %v", err)
	}
	test.NewTempApp(t)

	store := NewStore(t.TempDir())
	t.Cleanup(store.Flush)
	w := NewWatcher(store)
	w.Start()
	t.Cleanup(w.Stop)

	replaced := make(chan struct{})
	w.ownClipboard(func() (<-chan struct{}, error) { return replaced, nil })
	if _, running := watcherState(w); running {
		t.Fatal("the watch must pause for an own write")
	}

	close(replaced)
	waitUntil(t, func() bool {
		_, running := watcherState(w)
		return running
	})
}
