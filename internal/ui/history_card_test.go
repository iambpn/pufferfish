package ui

import (
	"testing"

	"fyne.io/fyne/v2/test"

	"github.com/iambpn/pufferfish/internal/clipboard"
)

func TestHistoryCardSetItemShowsTextPreview(t *testing.T) {
	test.NewTempApp(t)
	c := newHistoryCard()
	test.WidgetRenderer(c)

	c.setItem(clipboard.NewTextItem("hello"), "")

	if c.content.Text != "hello" {
		t.Fatalf("content text = %q", c.content.Text)
	}
	if !c.thumb.Hidden {
		t.Fatal("a text item must not show a thumbnail")
	}
}

func TestHistoryCardSetItemShowsImageThumbnail(t *testing.T) {
	test.NewTempApp(t)
	c := newHistoryCard()
	test.WidgetRenderer(c)

	item := clipboard.Item{Kind: clipboard.KindImage, Width: 10, Height: 6}
	c.setItem(item, "/tmp/some-image.png")

	if c.content.Text != "Image 10 × 6" {
		t.Fatalf("content text = %q", c.content.Text)
	}
	if c.thumb.Hidden {
		t.Fatal("an image item must show its thumbnail")
	}
	if c.thumb.File != "/tmp/some-image.png" {
		t.Fatalf("thumb file = %q", c.thumb.File)
	}
}

func TestHistoryCardTappedInvokesOnTap(t *testing.T) {
	test.NewTempApp(t)
	c := newHistoryCard()
	test.WidgetRenderer(c)

	calls := 0
	c.onTap = func() { calls++ }
	test.Tap(c)

	if calls != 1 {
		t.Fatalf("want 1 call, got %d", calls)
	}
}

func TestHistoryCardTappedWithNoHandlerDoesNotPanic(t *testing.T) {
	test.NewTempApp(t)
	c := newHistoryCard()
	test.WidgetRenderer(c)

	test.Tap(c)
}

func TestHistoryCardSwitchingBackToTextHidesThumbnail(t *testing.T) {
	test.NewTempApp(t)
	c := newHistoryCard()
	test.WidgetRenderer(c)

	c.setItem(clipboard.Item{Kind: clipboard.KindImage}, "/tmp/img.png")
	if c.thumb.Hidden {
		t.Fatal("precondition: thumbnail should be visible")
	}

	c.setItem(clipboard.NewTextItem("now text"), "")
	if !c.thumb.Hidden {
		t.Fatal("reusing the card for a text item must hide the thumbnail")
	}
}
