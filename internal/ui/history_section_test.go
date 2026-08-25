package ui

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

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

	content, detach := NewHistorySection(store, func(clipboard.Item) {}, func() {})
	w := test.NewWindow(content)
	w.Resize(fyne.NewSize(380, 400))
	return w, detach
}

func TestSectionShowsCapturedItems(t *testing.T) {
	test.NewTempApp(t)

	store := clipboard.NewStore(t.TempDir())
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
	content, detach := NewHistorySection(store, func(clipboard.Item) {}, func() {})
	w := test.NewWindow(content)
	defer w.Close()

	detach()
	// Would panic on a refresh of the closed window's tree if still attached.
	store.Add(clipboard.NewTextItem("after close"))
}

func TestDeleteButtonRemovesTheItem(t *testing.T) {
	test.NewTempApp(t)

	store := clipboard.NewStore(t.TempDir())
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
	store.Add(clipboard.NewTextItem("pick me"))

	var picked clipboard.Item
	content, detach := NewHistorySection(store, func(i clipboard.Item) { picked = i }, func() {})
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
