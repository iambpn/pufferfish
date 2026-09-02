package clipboard

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"time"

	"fyne.io/fyne/v2"
	"golang.org/x/image/draw"
)

// Load reads the persisted history from disk. A missing or unreadable
// history file is not an error: the app simply starts with an empty one.
func (s *Store) Load() {
	if s.dir == "" {
		return
	}

	data, err := os.ReadFile(filepath.Join(s.dir, historyFileName))
	if err != nil {
		return
	}

	var items []Item
	if err := json.Unmarshal(data, &items); err != nil {
		fyne.LogError("discarding unreadable clipboard history", err)
		return
	}

	s.mu.Lock()
	s.items = s.dropMissingImagesLocked(items)
	s.trimLocked()
	s.mu.Unlock()

	s.notify()
}

// dropMissingImagesLocked drops entries whose image file went missing while
// the app was closed, so the list never shows a broken card.
func (s *Store) dropMissingImagesLocked(items []Item) []Item {
	kept := items[:0]
	for _, item := range items {
		if item.Kind == KindImage {
			if _, ok := s.ImagePath(item); !ok {
				continue
			}
		}
		kept = append(kept, item)
	}
	return kept
}

// saveLocked encodes the history to JSON and starts a background write to
// disk. The caller must hold the write lock.
//
// The encoding runs here, under the lock, so it sees a consistent s.items.
// It is fast because the slice is already in memory. The file write is the
// slow part, so it runs on its own goroutine and does not block the caller
// - usually the UI goroutine, via fyne.Do.
func (s *Store) saveLocked() {
	if s.dir == "" {
		return
	}

	data, err := json.Marshal(s.items)
	if err != nil {
		fyne.LogError("could not encode clipboard history", err)
		return
	}

	s.saveSeq++
	seq := s.saveSeq
	s.saveWG.Add(1)
	go s.writeSaveAsync(seq, data)
}

// writeSaveAsync writes data to the history file. It skips the write when a
// newer save has already been written, so two quick saves cannot finish
// out of order and leave stale data on disk. saveMu lets only one of these
// run at a time, so their writes never interleave.
func (s *Store) writeSaveAsync(seq uint64, data []byte) {
	defer s.saveWG.Done()

	s.saveMu.Lock()
	defer s.saveMu.Unlock()

	if seq <= s.savedSeq {
		return
	}
	if err := os.WriteFile(filepath.Join(s.dir, historyFileName), data, 0o600); err != nil {
		fyne.LogError("could not save clipboard history", err)
		return
	}
	s.savedSeq = seq
}

// Flush blocks until every queued save has finished writing, or been
// replaced by a newer one. The app calls this on shutdown so the last
// change is not lost to a write still running when the process exits.
// Tests call it when they need a write on disk before reading it back.
func (s *Store) Flush() {
	s.saveWG.Wait()
}

// thumbMaxPixels is the longest edge, in pixels, of the thumbnail saved
// next to a captured image. History cards show it at 28pt, so 128px still
// looks sharp on a HiDPI screen while keeping each decoded thumbnail down
// to a few tens of KB in memory. Showing the original instead is far more
// expensive: a 4K screenshot is 385KB on disk but 32MB once decoded.
const thumbMaxPixels = 128

// AddImage stores png bytes as a new image item. It returns false when
// there is no directory to keep the file in, in which case nothing is
// captured.
//
// Decoding and resizing a large screenshot is slow, so it should not run
// on the UI goroutine. PrepareImage does that work; the watcher calls it
// in the background and then passes the finished item to Add.
func (s *Store) AddImage(png []byte) bool {
	item, ok := s.PrepareImage(png)
	if !ok {
		return false
	}
	s.Add(item)
	return true
}

// PrepareImage writes png and its thumbnail to the store's directory and
// returns the item describing them, without touching the history itself.
// It does no locking and is safe to call off the UI goroutine; pass the
// result to Add to actually record it.
func (s *Store) PrepareImage(png []byte) (Item, bool) {
	if s.dir == "" {
		return Item{}, false
	}

	hash := hashBytes(png)
	name := fmt.Sprintf("%s.png", hash[:16])
	path := filepath.Join(s.dir, name)

	// Re-copying an image the store already holds reuses its file; Add
	// then folds the entry back to the front by hash.
	if _, err := os.Stat(path); err != nil {
		if err := os.WriteFile(path, png, 0o600); err != nil {
			fyne.LogError("could not save clipboard image", err)
			return Item{}, false
		}
	}

	item := Item{
		Kind:       KindImage,
		ImageFile:  name,
		Hash:       hash,
		CapturedAt: time.Now(),
	}

	// Decode the image once here to read its size. The card shows the
	// thumbnail, so it never has to decode the full image itself.
	src, _, err := image.Decode(bytes.NewReader(png))
	if err != nil {
		fyne.LogError("could not decode clipboard image", err)
		return item, true
	}
	bounds := src.Bounds()
	item.Width, item.Height = bounds.Dx(), bounds.Dy()

	thumbName := fmt.Sprintf("%s_thumb.png", hash[:16])
	if err := writeThumb(src, filepath.Join(s.dir, thumbName)); err != nil {
		// The card falls back to the full image, which still renders.
		fyne.LogError("could not write clipboard image thumbnail", err)
		return item, true
	}
	item.ThumbFile = thumbName

	return item, true
}

// writeThumb scales src down to fit thumbMaxPixels and writes it to path as
// a PNG. An image already that small is written through unscaled.
func writeThumb(src image.Image, path string) error {
	bounds := src.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w <= 0 || h <= 0 {
		return errors.New("clipboard: image has no pixels")
	}

	if w > thumbMaxPixels || h > thumbMaxPixels {
		if w > h {
			w, h = thumbMaxPixels, max(h*thumbMaxPixels/w, 1)
		} else {
			w, h = max(w*thumbMaxPixels/h, 1), thumbMaxPixels
		}
		dst := image.NewNRGBA(image.Rect(0, 0, w, h))
		draw.ApproxBiLinear.Scale(dst, dst.Bounds(), src, bounds, draw.Src, nil)
		src = dst
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, src); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o600)
}
