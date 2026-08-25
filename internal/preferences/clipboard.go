package preferences

import "fyne.io/fyne/v2"

const (
	prefKeyUseClipboard = "clipboard.useClipboard"
	prefKeyAddImages    = "clipboard.addImages"
	prefKeyKeepContent  = "clipboard.keepContent"
	prefKeyAutoPaste    = "clipboard.autoPaste"
	prefKeyRecentItems  = "clipboard.recentItems"
)

const defaultRecentItems = 20

// ClipboardPreferences holds the clipboard settings shown in the
// preferences window. Every setter writes through to the app's persistent
// preference store, so changes survive a restart.
type ClipboardPreferences struct {
	UseClipboard bool
	AddImages    bool
	KeepContent  bool
	AutoPaste    bool
	RecentItems  int

	store fyne.Preferences
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
		RecentItems:  store.IntWithFallback(prefKeyRecentItems, defaultRecentItems),
		store:        store,
	}
}

func (p *ClipboardPreferences) SetUseClipboard(v bool) {
	p.UseClipboard = v
	p.store.SetBool(prefKeyUseClipboard, v)
}

func (p *ClipboardPreferences) SetAddImages(v bool) {
	p.AddImages = v
	p.store.SetBool(prefKeyAddImages, v)
}

func (p *ClipboardPreferences) SetKeepContent(v bool) {
	p.KeepContent = v
	p.store.SetBool(prefKeyKeepContent, v)
}

func (p *ClipboardPreferences) SetAutoPaste(v bool) {
	p.AutoPaste = v
	p.store.SetBool(prefKeyAutoPaste, v)
}

func (p *ClipboardPreferences) SetRecentItems(v int) {
	p.RecentItems = v
	p.store.SetInt(prefKeyRecentItems, v)
}
