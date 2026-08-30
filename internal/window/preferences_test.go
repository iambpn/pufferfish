package window

import (
	"testing"

	"fyne.io/fyne/v2/test"

	"github.com/iambpn/pufferfish/internal/preferences"
)

func TestPreferencesWindowOpensWithTheClipboardSection(t *testing.T) {
	a := test.NewTempApp(t)
	prefs := preferences.LoadClipboardPreferences(a)
	baseline := windowCount(a)

	show := NewPreferencesWindow(a, prefs)
	show()

	if got := windowCount(a); got != baseline+1 {
		t.Fatalf("want %d windows open, got %d", baseline+1, got)
	}
	win := newestWindow(a)
	if win.Content() == nil {
		t.Fatal("window has no content")
	}
	win.Close()
}

func TestPreferencesWindowReopenFocusesTheExistingWindow(t *testing.T) {
	a := test.NewTempApp(t)
	prefs := preferences.LoadClipboardPreferences(a)
	baseline := windowCount(a)

	show := NewPreferencesWindow(a, prefs)
	show()
	show()

	if got := windowCount(a); got != baseline+1 {
		t.Fatalf("calling show twice should reuse the window, got %d windows", got)
	}
	newestWindow(a).Close()
}

func TestPreferencesWindowCanReopenAfterClosing(t *testing.T) {
	a := test.NewTempApp(t)
	prefs := preferences.LoadClipboardPreferences(a)
	baseline := windowCount(a)

	show := NewPreferencesWindow(a, prefs)
	show()
	newestWindow(a).Close()

	show()
	if got := windowCount(a); got != baseline+1 {
		t.Fatalf("want a fresh window after closing, got %d", got)
	}
	newestWindow(a).Close()
}
