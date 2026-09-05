package clipboard

import (
	"os"
	"path/filepath"
	"sync"
)

const historyFileName = "history.json"

// DefaultLimit is used until preferences supply the configured value.
const DefaultLimit = 20

// MaxLimit mirrors preferences.MaxRecentItems: SetLimit enforces it
// directly so the store can never silently exceed the documented maximum,
// even if a caller passes through an unclamped or stale preference value.
const MaxLimit = 100

// Store holds captured clipboard items, newest first, persisted to dir so
// the history survives a restart. Image bytes live in their own files
// beside the index; removing an item removes its file too.
//
// Every mutation notifies the registered listeners after the lock is
// released, so a listener is free to call back into the store.
type Store struct {
	mu    sync.RWMutex
	items []Item
	limit int
	dir   string

	listenerMu sync.Mutex
	nextID     int
	listeners  map[int]func()

	// saveMu, saveSeq and savedSeq coordinate the background disk writes
	// started by saveLocked. They keep the writes off the caller's
	// goroutine (usually the UI one, via fyne.Do), keep them in the order
	// the state changed, and stop a slow write from overwriting a newer
	// one. See saveLocked.
	saveMu   sync.Mutex
	saveSeq  uint64
	savedSeq uint64
	saveWG   sync.WaitGroup
}

// NewStore creates a store persisting to dir. An empty dir keeps the
// history in memory only, which is the fallback when the app has no
// writable storage.
func NewStore(dir string) *Store {
	return &Store{
		limit:     DefaultLimit,
		dir:       dir,
		listeners: map[int]func(){},
	}
}

// AddListener registers fn to be called after every change, and returns a
// function that unregisters it again.
func (s *Store) AddListener(fn func()) (remove func()) {
	s.listenerMu.Lock()
	defer s.listenerMu.Unlock()

	id := s.nextID
	s.nextID++
	s.listeners[id] = fn

	return func() {
		s.listenerMu.Lock()
		defer s.listenerMu.Unlock()
		delete(s.listeners, id)
	}
}

// notify runs each registered listener so the UI can refresh itself after
// the history changes. Callers invoke it after releasing the lock, so a
// listener may call back into the store.
func (s *Store) notify() {
	s.listenerMu.Lock()
	fns := make([]func(), 0, len(s.listeners))
	for _, fn := range s.listeners {
		fns = append(fns, fn)
	}
	s.listenerMu.Unlock()

	for _, fn := range fns {
		fn()
	}
}

// Items returns a snapshot of the captured items, newest first.
func (s *Store) Items() []Item {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]Item(nil), s.items...)
}

// Len reports how many items are in the history, without copying the
// underlying slice the way Items does - cheaper for a caller that only
// needs the count, such as a list widget's length callback.
func (s *Store) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.items)
}

// ItemAt returns the item at index i without copying the rest of the
// slice the way Items does, reporting false if i is out of range.
func (s *Store) ItemAt(i int) (Item, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if i < 0 || i >= len(s.items) {
		return Item{}, false
	}
	return s.items[i], true
}

// Newest returns the most recent item, if there is one.
func (s *Store) Newest() (Item, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.items) == 0 {
		return Item{}, false
	}
	return s.items[0], true
}

// ImagePath resolves an image item's file to an absolute path. It reports
// false for text items and for images whose file is gone.
func (s *Store) ImagePath(item Item) (string, bool) {
	if item.Kind != KindImage || item.ImageFile == "" || s.dir == "" {
		return "", false
	}
	path := filepath.Join(s.dir, item.ImageFile)
	if _, err := os.Stat(path); err != nil {
		return "", false
	}
	return path, true
}

// ThumbPath resolves the small image a history card should draw for item.
// It falls back to the full-resolution file for entries with no thumbnail,
// and reports false for text items and for images whose file is gone.
func (s *Store) ThumbPath(item Item) (string, bool) {
	if item.Kind != KindImage || item.ThumbFile == "" || s.dir == "" {
		return s.ImagePath(item)
	}
	path := filepath.Join(s.dir, item.ThumbFile)
	if _, err := os.Stat(path); err != nil {
		return s.ImagePath(item)
	}
	return path, true
}

// Add records a newly captured item at the front of the history. Re-copying
// something already in the history moves it back to the front rather than
// duplicating it.
func (s *Store) Add(item Item) {
	s.mu.Lock()
	for i, existing := range s.items {
		if existing.Hash != item.Hash {
			continue
		}
		// The same content copied again: keep the file already on disk,
		// discard the freshly written duplicate, and move the entry back
		// to the front with the new capture time.
		if item.ImageFile != existing.ImageFile {
			s.removeFileFor(item)
		}
		// An entry created before thumbnails were available won't have one, while
		// the newly prepared copy does. Use it instead of discarding it, which
		// would otherwise leave an unreferenced file behind and keep the card
		// rendering the full-resolution image.
		if existing.ThumbFile == "" {
			existing.ThumbFile = item.ThumbFile
		}
		existing.CapturedAt = item.CapturedAt
		item = existing
		s.items = append(s.items[:i], s.items[i+1:]...)
		break
	}
	s.items = append([]Item{item}, s.items...)
	s.trimLocked()
	s.saveLocked()
	s.mu.Unlock()

	s.notify()
}

// RemoveAt deletes the item at index i.
func (s *Store) RemoveAt(i int) {
	s.mu.Lock()
	if i < 0 || i >= len(s.items) {
		s.mu.Unlock()
		return
	}
	s.removeFileFor(s.items[i])
	s.items = append(s.items[:i], s.items[i+1:]...)
	s.saveLocked()
	s.mu.Unlock()

	s.notify()
}

// Clear removes all captured items and their image files.
func (s *Store) Clear() {
	s.mu.Lock()
	for _, item := range s.items {
		s.removeFileFor(item)
	}
	s.items = nil
	s.saveLocked()
	s.mu.Unlock()

	s.notify()
}

// SetLimit caps how many items are kept, dropping the oldest beyond it.
func (s *Store) SetLimit(n int) {
	if n < 1 {
		n = 1
	}

	if n > MaxLimit {
		n = MaxLimit
	}

	s.mu.Lock()
	if s.limit == n {
		s.mu.Unlock()
		return
	}
	s.limit = n
	trimmed := s.trimLocked()
	if trimmed {
		s.saveLocked()
	}
	s.mu.Unlock()

	if trimmed {
		s.notify()
	}
}

// trimLocked drops the oldest items beyond the limit, reporting whether it
// removed anything. The caller must hold the write lock.
func (s *Store) trimLocked() bool {
	if len(s.items) <= s.limit {
		return false
	}
	for _, item := range s.items[s.limit:] {
		s.removeFileFor(item)
	}
	s.items = s.items[:s.limit]
	return true
}

func (s *Store) removeFileFor(item Item) {
	if item.Kind != KindImage || s.dir == "" {
		return
	}
	if item.ImageFile != "" {
		os.Remove(filepath.Join(s.dir, item.ImageFile))
	}
	if item.ThumbFile != "" {
		os.Remove(filepath.Join(s.dir, item.ThumbFile))
	}
}
