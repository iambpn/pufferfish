package clipboard

import "time"

// Item is a single captured clipboard entry.
type Item struct {
	Content    string
	CapturedAt time.Time
}
