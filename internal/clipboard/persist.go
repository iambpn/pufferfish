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

// saveLocked writes the history index to disk. The caller must hold the
// write lock.
func (s *Store) saveLocked() {
	if s.dir == "" {
		return
	}

	data, err := json.Marshal(s.items)
	if err != nil {
		fyne.LogError("could not encode clipboard history", err)
		return
	}
	if err := os.WriteFile(filepath.Join(s.dir, historyFileName), data, 0o600); err != nil {
		fyne.LogError("could not save clipboard history", err)
	}
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
