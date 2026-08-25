package clipboard

// Store holds captured clipboard items in memory, newest first.
type Store struct {
	items []Item
}

// NewStore creates an empty clipboard history store.
func NewStore() *Store {
	return &Store{}
}

// Items returns the captured items, newest first.
func (s *Store) Items() []Item {
	return s.items
}

// Add records a newly captured item at the front of the history.
func (s *Store) Add(item Item) {
	s.items = append([]Item{item}, s.items...)
}

// RemoveAt deletes the item at index i.
func (s *Store) RemoveAt(i int) {
	s.items = append(s.items[:i], s.items[i+1:]...)
}

// Clear removes all captured items.
func (s *Store) Clear() {
	s.items = nil
}
