package clipboard

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func pngBytes(t *testing.T, w, h int, c color.Color) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	img.Set(0, 0, c)
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestAddKeepsNewestFirst(t *testing.T) {
	s := NewStore(t.TempDir())
	s.Add(NewTextItem("one"))
	s.Add(NewTextItem("two"))

	items := s.Items()
	if len(items) != 2 || items[0].Text != "two" || items[1].Text != "one" {
		t.Fatalf("got %+v", items)
	}
}

func TestAddMovesDuplicateToFront(t *testing.T) {
	s := NewStore(t.TempDir())
	s.Add(NewTextItem("one"))
	s.Add(NewTextItem("two"))
	s.Add(NewTextItem("one"))

	items := s.Items()
	if len(items) != 2 {
		t.Fatalf("want 2 items, got %d: %+v", len(items), items)
	}
	if items[0].Text != "one" || items[1].Text != "two" {
		t.Fatalf("got %+v", items)
	}
}

func TestSetLimitTrimsOldest(t *testing.T) {
	s := NewStore(t.TempDir())
	for _, text := range []string{"a", "b", "c"} {
		s.Add(NewTextItem(text))
	}
	s.SetLimit(2)

	items := s.Items()
	if len(items) != 2 || items[0].Text != "c" || items[1].Text != "b" {
		t.Fatalf("got %+v", items)
	}
}

func TestAddRespectsLimit(t *testing.T) {
	s := NewStore(t.TempDir())
	s.SetLimit(2)
	for _, text := range []string{"a", "b", "c"} {
		s.Add(NewTextItem(text))
	}

	if items := s.Items(); len(items) != 2 || items[0].Text != "c" {
		t.Fatalf("got %+v", items)
	}
}

func TestImageIsStoredAndReloaded(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)
	if !s.AddImage(pngBytes(t, 4, 3, color.White)) {
		t.Fatal("AddImage reported failure")
	}

	item := s.Items()[0]
	if item.Kind != KindImage || item.Width != 4 || item.Height != 3 {
		t.Fatalf("got %+v", item)
	}
	if _, ok := s.ImagePath(item); !ok {
		t.Fatal("image file was not written")
	}

	reloaded := NewStore(dir)
	reloaded.Load()
	if got := reloaded.Items(); len(got) != 1 || got[0].Hash != item.Hash {
		t.Fatalf("got %+v", got)
	}
}

func TestRemoveDeletesImageFile(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)
	s.AddImage(pngBytes(t, 2, 2, color.Black))
	path, _ := s.ImagePath(s.Items()[0])

	s.RemoveAt(0)

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("image file still present: %v", err)
	}
}

func TestDuplicateImageKeepsItsFile(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)
	data := pngBytes(t, 2, 2, color.Black)
	s.AddImage(data)
	s.AddImage(data)

	items := s.Items()
	if len(items) != 1 {
		t.Fatalf("want 1 item, got %d", len(items))
	}
	if _, ok := s.ImagePath(items[0]); !ok {
		t.Fatal("re-copying an image deleted its file")
	}
}

func TestLoadDropsItemsWithMissingImages(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)
	s.Add(NewTextItem("kept"))
	s.AddImage(pngBytes(t, 2, 2, color.Black))

	path, _ := s.ImagePath(s.Items()[0])
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}

	reloaded := NewStore(dir)
	reloaded.Load()
	if got := reloaded.Items(); len(got) != 1 || got[0].Text != "kept" {
		t.Fatalf("got %+v", got)
	}
}

func TestClearRemovesEverything(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)
	s.Add(NewTextItem("a"))
	s.AddImage(pngBytes(t, 2, 2, color.Black))

	s.Clear()

	if got := s.Items(); len(got) != 0 {
		t.Fatalf("got %+v", got)
	}
	matches, _ := filepath.Glob(filepath.Join(dir, "*.png"))
	if len(matches) != 0 {
		t.Fatalf("image files left behind: %v", matches)
	}
}

func TestListenersFireOnChange(t *testing.T) {
	s := NewStore(t.TempDir())
	calls := 0
	remove := s.AddListener(func() {
		// A listener must be able to read the store without deadlocking.
		_ = s.Items()
		calls++
	})

	s.Add(NewTextItem("a"))
	if calls != 1 {
		t.Fatalf("want 1 call, got %d", calls)
	}

	remove()
	s.Add(NewTextItem("b"))
	if calls != 1 {
		t.Fatalf("listener fired after removal: %d", calls)
	}
}

func TestInMemoryStoreSkipsImages(t *testing.T) {
	s := NewStore("")
	if s.AddImage(pngBytes(t, 2, 2, color.Black)) {
		t.Fatal("AddImage should fail without a storage directory")
	}
	if got := s.Items(); len(got) != 0 {
		t.Fatalf("got %+v", got)
	}
}

func TestPreview(t *testing.T) {
	multiline := NewTextItem("first line\nsecond line")
	if got := multiline.Preview(); got != "first line …" {
		t.Fatalf("got %q", got)
	}

	img := Item{Kind: KindImage, Width: 800, Height: 600}
	if got := img.Preview(); got != "Image 800 × 600" {
		t.Fatalf("got %q", got)
	}
}
