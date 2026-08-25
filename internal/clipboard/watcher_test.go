package clipboard

import (
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

func TestSelfWriteIsConsumedOnce(t *testing.T) {
	w := NewWatcher(NewStore(t.TempDir()))
	w.expectSelfWrite("hash-a")

	if !w.consumeSelfWrite("hash-a") {
		t.Fatal("expected self write was not recognised")
	}
	if w.consumeSelfWrite("hash-a") {
		t.Fatal("self write should only be consumed once")
	}
}

func TestConsumeSelfWriteReportsFalseForUnknownHash(t *testing.T) {
	w := NewWatcher(NewStore(t.TempDir()))
	if w.consumeSelfWrite("never-written") {
		t.Fatal("an unrecorded hash must not be treated as a self write")
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
	w := NewWatcher(store)

	w.handle(system.Data{Format: system.FmtText, Bytes: []byte("copied text")})

	items := store.Items()
	if len(items) != 1 || items[0].Text != "copied text" {
		t.Fatalf("got %+v", items)
	}
}

func TestHandleSkipsItsOwnWrite(t *testing.T) {
	test.NewTempApp(t)
	store := NewStore(t.TempDir())
	w := NewWatcher(store)

	data := []byte("restored text")
	w.expectSelfWrite(hashBytes(data))
	w.handle(system.Data{Format: system.FmtText, Bytes: data})

	if got := store.Items(); len(got) != 0 {
		t.Fatalf("a self write should not be recaptured, got %+v", got)
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
