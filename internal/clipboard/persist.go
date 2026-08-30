package clipboard

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	_ "image/png"
	"os"
	"path/filepath"
	"time"

	"fyne.io/fyne/v2"
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
	s.items = s.keepReadableLocked(items)
	s.trimLocked()
	s.mu.Unlock()

	s.notify()
}

// keepReadableLocked drops entries whose image file went missing while the
// app was closed, so the list never shows a broken card.
func (s *Store) keepReadableLocked(items []Item) []Item {
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

// saveLocked marshals the history index and dispatches it to be written to
// disk in the background. The caller must hold the write lock.
//
// The write itself happens off whatever goroutine called Add/RemoveAt/
// Clear/SetLimit - typically the UI goroutine, wrapped in fyne.Do - so a
// slow disk doesn't stall it. Marshaling still happens here, synchronously,
// since it needs the lock the caller already holds to see a consistent
// s.items; encoding that (already in-memory) slice to JSON is cheap enough
// not to matter next to the actual file write.
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

// writeSaveAsync writes data to the history file, unless a later save has
// already landed - saveLocked's caller no longer holds the lock by the time
// this runs, so writes from back-to-back saves could otherwise finish out
// of order and leave a stale one on disk. saveMu serializes the writes
// themselves so two goroutines can't interleave partial ones.
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

// Flush blocks until every save queued so far has been written to disk (or
// superseded by a later one). The app calls this on shutdown so the very
// last change isn't lost to a save still in flight when the process exits;
// tests call it when they need a write to have landed before reading it
// back, e.g. reloading a store from disk right after mutating another one.
func (s *Store) Flush() {
	s.saveWG.Wait()
}

// AddImage stores png bytes as a new image item. It returns false when the
// history has nowhere to keep the file, in which case nothing is captured.
func (s *Store) AddImage(png []byte) bool {
	if s.dir == "" {
		return false
	}

	hash := hashBytes(png)
	name := fmt.Sprintf("%s.png", hash[:16])
	path := filepath.Join(s.dir, name)

	// Re-copying an image the store already holds reuses its file; Add
	// then folds the entry back to the front by hash.
	if _, err := os.Stat(path); err != nil {
		if err := os.WriteFile(path, png, 0o600); err != nil {
			fyne.LogError("could not save clipboard image", err)
			return false
		}
	}

	item := Item{
		Kind:       KindImage,
		ImageFile:  name,
		Hash:       hash,
		CapturedAt: time.Now(),
	}
	if cfg, _, err := image.DecodeConfig(bytes.NewReader(png)); err == nil {
		item.Width, item.Height = cfg.Width, cfg.Height
	}

	s.Add(item)
	return true
}
