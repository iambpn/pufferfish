package preferences

import "fyne.io/fyne/v2"

const (
	prefKeyUseClipboard = "clipboard.useClipboard"
	prefKeyAddImages    = "clipboard.addImages"
	prefKeyKeepContent  = "clipboard.keepContent"
	prefKeyAutoPaste    = "clipboard.autoPaste"
	prefKeyRecentItems  = "clipboard.recentItems"
	prefKeyHistoryX     = "clipboard.historyX"
	prefKeyHistoryY     = "clipboard.historyY"
)

// defaultHistoryX/Y is where the history window opens before the user has
// dragged it anywhere.
const (
	defaultHistoryX = 80
	defaultHistoryY = 80
)

// MinRecentItems and MaxRecentItems bound the number of items the history
// keeps.
const (
	MinRecentItems = 1
	MaxRecentItems = 100
)

const defaultRecentItems = 20

// ClipboardPreferences holds the clipboard settings shown in the
// preferences window. Every setter writes through to the app's persistent
// preference store, so changes survive a restart, and notifies the
// registered listeners so the running app follows the new setting at once.
type ClipboardPreferences struct {
	UseClipboard bool
	AddImages    bool
	KeepContent  bool
	AutoPaste    bool
	RecentItems  int
	// HistoryX/Y is the last position the user dragged the history window
	// to, so it reopens where they left it.
	HistoryX, HistoryY int

	store     fyne.Preferences
	listeners []func()
}

// LoadClipboardPreferences reads the saved clipboard preferences, falling
// back to defaults the first time the app runs.
func LoadClipboardPreferences(a fyne.App) *ClipboardPreferences {
	store := a.Preferences()
	return &ClipboardPreferences{
		UseClipboard: store.BoolWithFallback(prefKeyUseClipboard, true),
		AddImages:    store.BoolWithFallback(prefKeyAddImages, true),
		KeepContent:  store.BoolWithFallback(prefKeyKeepContent, true),
		AutoPaste:    store.BoolWithFallback(prefKeyAutoPaste, true),
		RecentItems:  clampRecentItems(store.IntWithFallback(prefKeyRecentItems, defaultRecentItems)),
		HistoryX:     store.IntWithFallback(prefKeyHistoryX, defaultHistoryX),
		HistoryY:     store.IntWithFallback(prefKeyHistoryY, defaultHistoryY),
		store:        store,
	}
}

// clampRecentItems keeps v within [MinRecentItems, MaxRecentItems], so a
// stale or hand-edited preference value can't push the history limit
// outside the documented bounds.
func clampRecentItems(v int) int {
	if v < MinRecentItems {
		return MinRecentItems
	}
	if v > MaxRecentItems {
		return MaxRecentItems
	}
	return v
}

// AddListener registers fn to run after any preference changes. Listeners
// are expected to re-apply the whole set, so they stay correct whichever
// setting moved.
func (p *ClipboardPreferences) AddListener(fn func()) {
	p.listeners = append(p.listeners, fn)
}

func (p *ClipboardPreferences) notify() {
	for _, fn := range p.listeners {
		fn()
	}
}

func (p *ClipboardPreferences) SetUseClipboard(v bool) {
	p.UseClipboard = v
	p.store.SetBool(prefKeyUseClipboard, v)
	p.notify()
}

func (p *ClipboardPreferences) SetAddImages(v bool) {
	p.AddImages = v
	p.store.SetBool(prefKeyAddImages, v)
	p.notify()
}

func (p *ClipboardPreferences) SetKeepContent(v bool) {
	p.KeepContent = v
	p.store.SetBool(prefKeyKeepContent, v)
	p.notify()
}

func (p *ClipboardPreferences) SetAutoPaste(v bool) {
	p.AutoPaste = v
	p.store.SetBool(prefKeyAutoPaste, v)
	p.notify()
}

func (p *ClipboardPreferences) SetRecentItems(v int) {
	v = clampRecentItems(v)
	p.RecentItems = v
	p.store.SetInt(prefKeyRecentItems, v)
	p.notify()
}

// SetHistoryCoords records where the user dragged the history window. No
// listener re-applies a window position, so it doesn't notify.
func (p *ClipboardPreferences) SetHistoryCoords(x, y int) {
	p.HistoryX, p.HistoryY = x, y
	p.store.SetInt(prefKeyHistoryX, x)
	p.store.SetInt(prefKeyHistoryY, y)
}
