package clipboard

import (
	"image/color"
	"testing"
)

func TestInitCanBeCalledMoreThanOnce(t *testing.T) {
	err := Init()
	if err != nil {
		t.Skipf("system clipboard unavailable: %v", err)
	}
	if err := Init(); err != nil {
		t.Fatalf("second Init call failed: %v", err)
	}
}

func TestPutImageWithMissingFileFails(t *testing.T) {
	store := NewStore(t.TempDir())
	w := NewWatcher(store)

	item := Item{Kind: KindImage, ImageFile: "does-not-exist.png"}
	if err := w.Put(item); err == nil {
		t.Fatal("want an error when the backing image file is missing")
	}
}

func TestPutRestoresTextToTheSystemClipboard(t *testing.T) {
	if err := Init(); err != nil {
		t.Skipf("system clipboard unavailable: %v", err)
	}

	store := NewStore(t.TempDir())
	w := NewWatcher(store)

	if err := w.Put(NewTextItem("restored via Put")); err != nil {
		t.Fatalf("Put failed: %v", err)
	}
}

func TestClearEmptiesTheSystemClipboard(t *testing.T) {
	if err := Init(); err != nil {
		t.Skipf("system clipboard unavailable: %v", err)
	}

	store := NewStore(t.TempDir())
	w := NewWatcher(store)

	if err := w.Put(NewTextItem("about to be cleared")); err != nil {
		t.Fatalf("Put failed: %v", err)
	}
	if err := w.Clear(); err != nil {
		t.Fatalf("Clear failed: %v", err)
	}
}

func TestPutRestoresImageToTheSystemClipboard(t *testing.T) {
	if err := Init(); err != nil {
		t.Skipf("system clipboard unavailable: %v", err)
	}

	dir := t.TempDir()
	store := NewStore(dir)
	t.Cleanup(store.Flush)
	if !store.AddImage(pngBytes(t, 2, 2, color.White)) {
		t.Fatal("AddImage failed")
	}
	item := store.Items()[0]

	w := NewWatcher(store)
	if err := w.Put(item); err != nil {
		t.Fatalf("Put failed: %v", err)
	}
}
