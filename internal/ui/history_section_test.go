package ui

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/iambpn/pufferfish/internal/clipboard"
)

// cards walks the rendered tree and returns every history card in it.
func cards(obj fyne.CanvasObject) []*historyCard {
	var found []*historyCard
	var walk func(fyne.CanvasObject)
	walk = func(o fyne.CanvasObject) {
		if c, ok := o.(*historyCard); ok {
			found = append(found, c)
			return
		}
		switch t := o.(type) {
		case *fyne.Container:
			for _, child := range t.Objects {
				walk(child)
			}
		case fyne.Widget:
			for _, child := range test.WidgetRenderer(t).Objects() {
				walk(child)
			}
		}
	}
	walk(obj)
	return found
}

func newTestSection(t *testing.T, store *clipboard.Store) (fyne.Window, func()) {
	t.Helper()

	content, detach := NewHistorySection(store, func(clipboard.Item) {}, func() {}, store.Clear)
	w := test.NewWindow(content)
	w.Resize(fyne.NewSize(380, 400))
	return w, detach
}

func TestSectionShowsCapturedItems(t *testing.T) {
	test.NewTempApp(t)

	store := clipboard.NewStore(t.TempDir())
	t.Cleanup(store.Flush)
	store.Add(clipboard.NewTextItem("hello"))

	w, detach := newTestSection(t, store)
	defer w.Close()
	defer detach()

	got := cards(w.Content())
	if len(got) == 0 {
		t.Fatal("no history card rendered")
	}
	if text := got[0].content.Text; text != "hello" {
		t.Fatalf("card shows %q", text)
	}
}

func TestSectionFollowsStoreWhileOpen(t *testing.T) {
	test.NewTempApp(t)

	store := clipboard.NewStore(t.TempDir())
	t.Cleanup(store.Flush)
	w, detach := newTestSection(t, store)
	defer w.Close()
	defer detach()

	if len(cards(w.Content())) != 0 {
		t.Fatal("empty store should render no cards")
	}

	// A copy made while the flyout is open must appear without reopening it.
	store.Add(clipboard.NewTextItem("copied later"))

	got := cards(w.Content())
	if len(got) == 0 {
		t.Fatal("card did not appear after the store changed")
	}
	if text := got[0].content.Text; text != "copied later" {
		t.Fatalf("card shows %q", text)
	}
}

func TestDetachStopsFollowingTheStore(t *testing.T) {
	test.NewTempApp(t)

	store := clipboard.NewStore(t.TempDir())
	t.Cleanup(store.Flush)
	content, detach := NewHistorySection(store, func(clipboard.Item) {}, func() {}, store.Clear)
	w := test.NewWindow(content)
	defer w.Close()

	detach()
	// Would panic on a refresh of the closed window's tree if still attached.
	store.Add(clipboard.NewTextItem("after close"))
}

func TestDeleteButtonRemovesTheItem(t *testing.T) {
	test.NewTempApp(t)

	store := clipboard.NewStore(t.TempDir())
	t.Cleanup(store.Flush)
	store.Add(clipboard.NewTextItem("doomed"))

	w, detach := newTestSection(t, store)
	defer w.Close()
	defer detach()

	got := cards(w.Content())
	if len(got) == 0 {
		t.Fatal("no history card rendered")
	}
	test.Tap(got[0].deleteBtn)

	if items := store.Items(); len(items) != 0 {
		t.Fatalf("item survived deletion: %+v", items)
	}
}

func TestTappingACardSelectsIt(t *testing.T) {
	test.NewTempApp(t)

	store := clipboard.NewStore(t.TempDir())
	t.Cleanup(store.Flush)
	store.Add(clipboard.NewTextItem("pick me"))

	var picked clipboard.Item
	content, detach := NewHistorySection(store, func(i clipboard.Item) { picked = i }, func() {}, store.Clear)
	defer detach()

	w := test.NewWindow(content)
	w.Resize(fyne.NewSize(380, 400))
	defer w.Close()

	got := cards(w.Content())
	if len(got) == 0 {
		t.Fatal("no history card rendered")
	}
	test.Tap(got[0])

	if picked.Text != "pick me" {
		t.Fatalf("selected %+v", picked)
	}
}

func TestImageItemShowsThumbnail(t *testing.T) {
	test.NewTempApp(t)

	var buf bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, 10, 6))
	img.Set(0, 0, color.White)
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}

	store := clipboard.NewStore(t.TempDir())
	t.Cleanup(store.Flush)
	if !store.AddImage(buf.Bytes()) {
		t.Fatal("AddImage failed")
	}

	w, detach := newTestSection(t, store)
	defer w.Close()
	defer detach()

	got := cards(w.Content())
	if len(got) == 0 {
		t.Fatal("no history card rendered")
	}
	card := got[0]
	if card.thumb.Hidden {
		t.Fatal("image item rendered without its thumbnail")
	}
	if card.thumb.File == "" {
		t.Fatal("thumbnail has no file backing it")
	}
	if card.content.Text != "Image 10 × 6" {
		t.Fatalf("card shows %q", card.content.Text)
	}
}

func TestTextItemHidesTheThumbnail(t *testing.T) {
	test.NewTempApp(t)

	store := clipboard.NewStore(t.TempDir())
	t.Cleanup(store.Flush)
	store.Add(clipboard.NewTextItem("just text"))

	w, detach := newTestSection(t, store)
	defer w.Close()
	defer detach()

	got := cards(w.Content())
	if len(got) == 0 {
		t.Fatal("no history card rendered")
	}
	if !got[0].thumb.Hidden {
		t.Fatal("text item rendered a thumbnail")
	}
}

func TestNoScrollShadowThemeHidesOnlyTheShadowColor(t *testing.T) {
	base := theme.DefaultTheme()
	wrapped := noScrollShadowTheme{Theme: base}

	for _, variant := range []fyne.ThemeVariant{theme.VariantLight, theme.VariantDark} {
		if got := wrapped.Color(theme.ColorNameShadow, variant); got != color.Transparent {
			t.Fatalf("shadow color = %v, want transparent", got)
		}
		if got, want := wrapped.Color(theme.ColorNameForeground, variant), base.Color(theme.ColorNameForeground, variant); got != want {
			t.Fatalf("foreground color changed: got %v, want %v", got, want)
		}
	}
}

func TestHistorySectionListHasNoScrollShadow(t *testing.T) {
	test.NewTempApp(t)

	store := clipboard.NewStore(t.TempDir())
	t.Cleanup(store.Flush)
	store.Add(clipboard.NewTextItem("one"))

	w, detach := newTestSection(t, store)
	defer w.Close()
	defer detach()

	list := findList(w.Content())
	if list == nil {
		t.Fatal("no list rendered")
	}
	if got := theme.CurrentForWidget(list).Color(theme.ColorNameShadow, theme.VariantDark); got != color.Transparent {
		t.Fatalf("list's shadow color = %v, want transparent", got)
	}
}

func TestClearAllButtonInvokesCallback(t *testing.T) {
	test.NewTempApp(t)

	store := clipboard.NewStore(t.TempDir())
	t.Cleanup(store.Flush)
	store.Add(clipboard.NewTextItem("one"))

	var cleared bool
	content, detach := NewHistorySection(store, func(clipboard.Item) {}, func() {}, func() { cleared = true })
	w := test.NewWindow(content)
	w.Resize(fyne.NewSize(380, 400))
	defer w.Close()
	defer detach()

	btn := findSmallButton(w.Content())
	if btn == nil {
		t.Fatal("no clear-all button rendered")
	}
	test.Tap(btn)

	if !cleared {
		t.Fatal("tapping clear all did not invoke the callback")
	}
}

// findSmallButton walks the rendered tree and returns the first smallButton
// found, if any.
func findSmallButton(obj fyne.CanvasObject) *smallButton {
	var found *smallButton
	var walk func(fyne.CanvasObject)
	walk = func(o fyne.CanvasObject) {
		if found != nil {
			return
		}
		if b, ok := o.(*smallButton); ok {
			found = b
			return
		}
		switch t := o.(type) {
		case *fyne.Container:
			for _, child := range t.Objects {
				walk(child)
			}
		case fyne.Widget:
			for _, child := range test.WidgetRenderer(t).Objects() {
				walk(child)
			}
		}
	}
	walk(obj)
	return found
}

// findList walks the rendered tree and returns the history list, if any.
func findList(obj fyne.CanvasObject) *widget.List {
	var found *widget.List
	var walk func(fyne.CanvasObject)
	walk = func(o fyne.CanvasObject) {
		if found != nil {
			return
		}
		if l, ok := o.(*widget.List); ok {
			found = l
			return
		}
		switch t := o.(type) {
		case *fyne.Container:
			for _, child := range t.Objects {
				walk(child)
			}
		case fyne.Widget:
			for _, child := range test.WidgetRenderer(t).Objects() {
				walk(child)
			}
		}
	}
	walk(obj)
	return found
}
