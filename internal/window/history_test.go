package window

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/test"

	"github.com/iambpn/pufferfish/internal/clipboard"
	"github.com/iambpn/pufferfish/internal/preferences"
)

// desktopDriver adapts the test driver to desktop.Driver, which
// NewHistoryWindow requires for its splash window. The test driver has no
// desktop.Driver of its own, so CreateSplashWindow is delegated to a plain
// CreateWindow: the closure under test does not depend on the splash
// window's real borderless/centered behaviour, only that it gets a window
// back.
type desktopDriver struct {
	fyne.Driver
}

func (d desktopDriver) CreateSplashWindow() fyne.Window       { return d.CreateWindow("") }
func (d desktopDriver) CurrentKeyModifiers() fyne.KeyModifier { return 0 }
func (d desktopDriver) HasSecondaryDisplay() bool             { return false }

type appWithDesktopDriver struct {
	fyne.App
	driver fyne.Driver // concrete type also satisfies desktop.Driver
}

func (a *appWithDesktopDriver) Driver() fyne.Driver { return a.driver }

func newTestApp(t *testing.T) fyne.App {
	t.Helper()
	base := test.NewTempApp(t)
	return &appWithDesktopDriver{App: base, driver: desktopDriver{Driver: base.Driver()}}
}

// The test driver always keeps one dummy window alive for rendering (see
// fyne's test.NewDriver), so window counts in these tests are compared
// against a baseline rather than an absolute number.
func windowCount(a fyne.App) int { return len(a.Driver().AllWindows()) }

// newestWindow returns the most recently created window, which is always
// what NewHistoryWindow/NewPreferencesWindow just opened.
func newestWindow(a fyne.App) fyne.Window {
	windows := a.Driver().AllWindows()
	return windows[len(windows)-1]
}

func newTestHistoryWindow(t *testing.T) (a fyne.App, show func()) {
	t.Helper()
	a = newTestApp(t)
	store := clipboard.NewStore(t.TempDir())
	t.Cleanup(store.Flush)
	watcher := clipboard.NewWatcher(store)
	prefs := preferences.LoadClipboardPreferences(a)
	return a, NewHistoryWindow(a, store, watcher, prefs)
}

func TestHistoryWindowOpensWithContent(t *testing.T) {
	a, show := newTestHistoryWindow(t)
	baseline := windowCount(a)

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

func TestHistoryWindowReopenFocusesTheExistingWindow(t *testing.T) {
	a, show := newTestHistoryWindow(t)
	baseline := windowCount(a)

	show()
	show()

	if got := windowCount(a); got != baseline+1 {
		t.Fatalf("calling show twice should reuse the window, got %d windows", got)
	}
	newestWindow(a).Close()
}

func TestHistoryWindowCanReopenAfterClosing(t *testing.T) {
	a, show := newTestHistoryWindow(t)
	baseline := windowCount(a)

	show()
	newestWindow(a).Close()

	show()
	if got := windowCount(a); got != baseline+1 {
		t.Fatalf("want a fresh window after closing, got %d", got)
	}
	newestWindow(a).Close()
}

func TestHistoryWindowEscapeKeyCloses(t *testing.T) {
	a, show := newTestHistoryWindow(t)
	baseline := windowCount(a)

	show()
	win := newestWindow(a)

	onKey := win.Canvas().OnTypedKey()
	if onKey == nil {
		t.Fatal("no key handler was registered")
	}
	onKey(&fyne.KeyEvent{Name: fyne.KeyEscape})

	if got := windowCount(a); got != baseline {
		t.Fatalf("Escape should close the window, got %d windows still open", got)
	}
}

func TestHistoryWindowSelectingAnItemRestoresItAndCloses(t *testing.T) {
	if err := clipboard.Init(); err != nil {
		t.Skipf("system clipboard unavailable: %v", err)
	}

	a := newTestApp(t)
	baseline := windowCount(a)

	store := clipboard.NewStore(t.TempDir())
	t.Cleanup(store.Flush)
	store.Add(clipboard.NewTextItem("pick me"))
	watcher := clipboard.NewWatcher(store)
	prefs := preferences.LoadClipboardPreferences(a)
	prefs.SetAutoPaste(false)

	show := NewHistoryWindow(a, store, watcher, prefs)
	show()

	win := newestWindow(a)
	tapFirstCard(t, win.Content())

	if got := windowCount(a); got != baseline {
		t.Fatalf("selecting an item should close the window, got %d windows still open", got)
	}
}

func TestHistoryWindowSelectingAnOlderItemMovesItToTheFront(t *testing.T) {
	if err := clipboard.Init(); err != nil {
		t.Skipf("system clipboard unavailable: %v", err)
	}

	a := newTestApp(t)

	store := clipboard.NewStore(t.TempDir())
	t.Cleanup(store.Flush)
	store.Add(clipboard.NewTextItem("older"))
	store.Add(clipboard.NewTextItem("newest"))
	watcher := clipboard.NewWatcher(store)
	prefs := preferences.LoadClipboardPreferences(a)
	prefs.SetAutoPaste(false)

	show := NewHistoryWindow(a, store, watcher, prefs)
	show()

	win := newestWindow(a)
	// Items() is newest-first, so "older" renders as the second card.
	tapCardAt(t, win.Content(), 1)

	items := store.Items()
	if len(items) != 2 || items[0].Text != "older" || items[1].Text != "newest" {
		t.Fatalf("want [older, newest] after selecting the older item, got %#v", items)
	}
}

// tapFirstCard finds the first tappable, non-hoverable widget in the
// rendered history section (i.e. a history card, as opposed to the
// hoverable clear-all/close/delete buttons) and taps it.
func tapFirstCard(t *testing.T, obj fyne.CanvasObject) {
	t.Helper()
	tapCardAt(t, obj, 0)
}

// tapCardAt taps the nth tappable, non-hoverable widget (i.e. history card,
// in list order) found in the rendered history section.
func tapCardAt(t *testing.T, obj fyne.CanvasObject, n int) {
	t.Helper()
	cards := findCards(obj)
	if n >= len(cards) {
		t.Fatalf("want at least %d cards, found %d", n+1, len(cards))
	}
	test.Tap(cards[n])
}

// findCards walks the rendered history section and returns every tappable,
// non-hoverable widget (i.e. history card, as opposed to the hoverable
// clear-all/close/delete buttons) in list order.
func findCards(obj fyne.CanvasObject) []fyne.Tappable {
	var cards []fyne.Tappable
	var walk func(fyne.CanvasObject)
	walk = func(o fyne.CanvasObject) {
		if tp, ok := o.(fyne.Tappable); ok {
			if _, hoverable := o.(desktop.Hoverable); !hoverable {
				cards = append(cards, tp)
				return
			}
			// Tappable but also hoverable: a button, or the list's own
			// row wrapper. Keep descending into it so a non-hoverable
			// tappable underneath - the card itself - can still be found.
		}
		switch c := o.(type) {
		case *fyne.Container:
			for _, child := range c.Objects {
				walk(child)
			}
		case fyne.Widget:
			for _, child := range test.WidgetRenderer(c).Objects() {
				walk(child)
			}
		}
	}
	walk(obj)
	return cards
}
