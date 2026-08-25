package clipboard

import (
	"os"
	"path/filepath"
	"sync"
)

const historyFileName = "history.json"

// DefaultLimit is used until preferences supply the configured value.
const DefaultLimit = 20

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
	if item.Kind != KindImage || item.ImageFile == "" || s.dir == "" {
		return
	}
	os.Remove(filepath.Join(s.dir, item.ImageFile))
}
