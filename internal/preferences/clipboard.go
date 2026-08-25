package preferences

import "fyne.io/fyne/v2"

const (
	prefKeyUseClipboard    = "clipboard.useClipboard"
	prefKeyAddImages       = "clipboard.addImages"
	prefKeyKeepContent     = "clipboard.keepContent"
	prefKeyAutoPaste       = "clipboard.autoPaste"
	prefKeyRecentItems     = "clipboard.recentItems"
	prefKeyHistoryPosition = "clipboard.historyPosition"
)

// MinRecentItems and MaxRecentItems bound the number of items the history
// keeps.
const (
	MinRecentItems = 1
	MaxRecentItems = 100
)

const defaultRecentItems = 20

// HistoryPosition names a screen anchor the history window can open at.
type HistoryPosition string

const (
	HistoryPositionTopLeft      HistoryPosition = "top-left"
	HistoryPositionTopCenter    HistoryPosition = "top-center"
	HistoryPositionTopRight     HistoryPosition = "top-right"
	HistoryPositionCenterLeft   HistoryPosition = "center-left"
	HistoryPositionCenter       HistoryPosition = "center"
	HistoryPositionCenterRight  HistoryPosition = "center-right"
	HistoryPositionBottomLeft   HistoryPosition = "bottom-left"
	HistoryPositionBottomCenter HistoryPosition = "bottom-center"
	HistoryPositionBottomRight  HistoryPosition = "bottom-right"
)

const defaultHistoryPosition = HistoryPositionCenter

// ClipboardPreferences holds the clipboard settings shown in the
// preferences window. Every setter writes through to the app's persistent
// preference store, so changes survive a restart, and notifies the
// registered listeners so the running app follows the new setting at once.
type ClipboardPreferences struct {
	UseClipboard    bool
	AddImages       bool
	KeepContent     bool
	AutoPaste       bool
	RecentItems     int
	HistoryPosition HistoryPosition

	store     fyne.Preferences
	listeners []func()
}

// LoadClipboardPreferences reads the saved clipboard preferences, falling
// back to defaults the first time the app runs.
func LoadClipboardPreferences(a fyne.App) *ClipboardPreferences {
	store := a.Preferences()
	return &ClipboardPreferences{
		UseClipboard:    store.BoolWithFallback(prefKeyUseClipboard, true),
		AddImages:       store.BoolWithFallback(prefKeyAddImages, true),
		KeepContent:     store.BoolWithFallback(prefKeyKeepContent, true),
		AutoPaste:       store.BoolWithFallback(prefKeyAutoPaste, true),
		RecentItems:     store.IntWithFallback(prefKeyRecentItems, defaultRecentItems),
		HistoryPosition: HistoryPosition(store.StringWithFallback(prefKeyHistoryPosition, string(defaultHistoryPosition))),
		store:           store,
	}
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
	p.RecentItems = v
	p.store.SetInt(prefKeyRecentItems, v)
	p.notify()
}

func (p *ClipboardPreferences) SetHistoryPosition(v HistoryPosition) {
	p.HistoryPosition = v
	p.store.SetString(prefKeyHistoryPosition, string(v))
	p.notify()
}
