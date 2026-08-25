package preferences

import (
	"testing"

	"fyne.io/fyne/v2/test"
)

func TestDefaultsOnFirstRun(t *testing.T) {
	p := LoadClipboardPreferences(test.NewTempApp(t))

	if !p.UseClipboard || !p.AddImages || !p.KeepContent || !p.AutoPaste {
		t.Fatalf("got %+v", p)
	}
	if p.RecentItems != defaultRecentItems {
		t.Fatalf("recent items = %d", p.RecentItems)
	}
	if p.HistoryPosition != defaultHistoryPosition {
		t.Fatalf("history position = %q", p.HistoryPosition)
	}
}

func TestSettersPersistAndNotify(t *testing.T) {
	a := test.NewTempApp(t)
	p := LoadClipboardPreferences(a)

	calls := 0
	p.AddListener(func() { calls++ })

	p.SetUseClipboard(false)
	p.SetAddImages(false)
	p.SetKeepContent(false)
	p.SetAutoPaste(false)
	p.SetRecentItems(7)
	p.SetHistoryPosition(HistoryPositionTopRight)

	if calls != 6 {
		t.Fatalf("want 6 notifications, got %d", calls)
	}

	// Reloading from the same app must see the written-through values.
	reloaded := LoadClipboardPreferences(a)
	if reloaded.UseClipboard || reloaded.AddImages || reloaded.KeepContent || reloaded.AutoPaste {
		t.Fatalf("got %+v", reloaded)
	}
	if reloaded.RecentItems != 7 {
		t.Fatalf("recent items = %d", reloaded.RecentItems)
	}
	if reloaded.HistoryPosition != HistoryPositionTopRight {
		t.Fatalf("history position = %q", reloaded.HistoryPosition)
	}
}
