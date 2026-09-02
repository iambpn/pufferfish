package clipboard

import (
	"image"
	"image/color"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadWithNoStorageDirIsANoOp(t *testing.T) {
	s := NewStore("")
	s.Load()

	if got := s.Items(); len(got) != 0 {
		t.Fatalf("got %+v", got)
	}
}

func TestLoadWithMissingHistoryFileStartsEmpty(t *testing.T) {
	s := NewStore(t.TempDir())
	s.Load()

	if got := s.Items(); len(got) != 0 {
		t.Fatalf("got %+v", got)
	}
}

func TestLoadDiscardsUnreadableHistory(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, historyFileName), []byte("not json"), 0o600); err != nil {
		t.Fatal(err)
	}

	s := NewStore(dir)
	s.Load()

	if got := s.Items(); len(got) != 0 {
		t.Fatalf("got %+v", got)
	}
}

func TestSaveLockedWithNoStorageDirWritesNoFile(t *testing.T) {
	s := NewStore("")
	s.Add(NewTextItem("a"))
	// Nothing to assert on disk since there is no dir; Add must simply not
	// panic when saveLocked runs with s.dir == "".
}

func TestLoadPersistsAcrossStores(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)
	s.Add(NewTextItem("one"))
	s.Add(NewTextItem("two"))

	s.Flush()
	reloaded := NewStore(dir)
	reloaded.Load()

	items := reloaded.Items()
	if len(items) != 2 || items[0].Text != "two" || items[1].Text != "one" {
		t.Fatalf("got %+v", items)
	}
}

func TestFlushBlocksUntilTheBackgroundWriteLands(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)
	s.Add(NewTextItem("a"))

	s.Flush()

	if _, err := os.Stat(filepath.Join(dir, historyFileName)); err != nil {
		t.Fatalf("history file missing after Flush: %v", err)
	}
}

// TestStaleSaveNeverClobbersANewerOne exercises writeSaveAsync's ordering
// guard directly: saveLocked's background writes can otherwise finish in
// whatever order the goroutine scheduler picks, and an older save landing
// after a newer one would silently leave stale data on disk.
func TestStaleSaveNeverClobbersANewerOne(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)

	s.saveWG.Add(1)
	s.writeSaveAsync(2, []byte(`"newer"`))
	s.saveWG.Add(1)
	s.writeSaveAsync(1, []byte(`"older"`)) // stale: must be dropped

	got, err := os.ReadFile(filepath.Join(dir, historyFileName))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != `"newer"` {
		t.Fatalf("got %q, want the newer save to still be on disk", got)
	}
}

// TestThumbnailIsSmallerThanTheOriginal pins why thumbnails exist: the
// history card must never point Fyne at a full-resolution screenshot,
// which it would decode and hold as a raster far larger than the file.
func TestThumbnailIsWrittenAndDownscaled(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)
	t.Cleanup(s.Flush)

	if !s.AddImage(pngBytes(t, 1200, 900, color.White)) {
		t.Fatal("AddImage failed")
	}

	item := s.Items()[0]
	if item.Width != 1200 || item.Height != 900 {
		t.Fatalf("item records %dx%d, want the original dimensions", item.Width, item.Height)
	}
	if item.ThumbFile == "" {
		t.Fatal("no thumbnail was written")
	}

	path, ok := s.ThumbPath(item)
	if !ok {
		t.Fatal("ThumbPath found nothing")
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Width > thumbMaxPixels || cfg.Height > thumbMaxPixels {
		t.Fatalf("thumbnail is %dx%d, want both edges <= %d", cfg.Width, cfg.Height, thumbMaxPixels)
	}
	if cfg.Width != thumbMaxPixels {
		t.Fatalf("thumbnail is %dx%d, want the long edge scaled to %d", cfg.Width, cfg.Height, thumbMaxPixels)
	}
}

// TestSmallImageKeepsItsSize checks an image already under the cap is not
// scaled up on the way into the thumbnail.
func TestSmallImageThumbnailIsNotUpscaled(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)
	t.Cleanup(s.Flush)
	s.AddImage(pngBytes(t, 8, 4, color.Black))

	path, _ := s.ThumbPath(s.Items()[0])
	f, _ := os.Open(path)
	defer f.Close()
	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Width != 8 || cfg.Height != 4 {
		t.Fatalf("got %dx%d, want 8x4 unchanged", cfg.Width, cfg.Height)
	}
}

// TestRemovingAnItemRemovesItsThumbnail keeps the thumbnails from
// outliving the history and quietly filling the storage directory.
func TestRemovingAnItemRemovesItsThumbnail(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)
	t.Cleanup(s.Flush)
	s.AddImage(pngBytes(t, 200, 200, color.Black))

	item := s.Items()[0]
	thumb := filepath.Join(dir, item.ThumbFile)
	if _, err := os.Stat(thumb); err != nil {
		t.Fatalf("thumbnail missing before removal: %v", err)
	}

	s.RemoveAt(0)

	if _, err := os.Stat(thumb); !os.IsNotExist(err) {
		t.Fatalf("thumbnail outlived its item: %v", err)
	}
}

// TestThumbPathFallsBackToTheOriginal covers entries captured before
// thumbnails existed, which must still render rather than showing nothing.
func TestThumbPathFallsBackToTheOriginal(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)
	t.Cleanup(s.Flush)
	s.AddImage(pngBytes(t, 2, 2, color.Black))

	item := s.Items()[0]
	os.Remove(filepath.Join(dir, item.ThumbFile))

	path, ok := s.ThumbPath(item)
	if !ok {
		t.Fatal("ThumbPath gave up instead of falling back")
	}
	if filepath.Base(path) != item.ImageFile {
		t.Fatalf("got %q, want the original %q", filepath.Base(path), item.ImageFile)
	}
}
