package clipboard

import "testing"

func TestNewTextItemSetsKindHashAndTimestamp(t *testing.T) {
	item := NewTextItem("hello")

	if item.Kind != KindText {
		t.Fatalf("kind = %v", item.Kind)
	}
	if item.Text != "hello" {
		t.Fatalf("text = %q", item.Text)
	}
	if item.Hash != hashBytes([]byte("hello")) {
		t.Fatalf("hash = %q", item.Hash)
	}
	if item.CapturedAt.IsZero() {
		t.Fatal("CapturedAt was not set")
	}
}

func TestHashBytesIsStableAndDistinguishesContent(t *testing.T) {
	if hashBytes([]byte("a")) != hashBytes([]byte("a")) {
		t.Fatal("same bytes hashed differently")
	}
	if hashBytes([]byte("a")) == hashBytes([]byte("b")) {
		t.Fatal("different bytes hashed the same")
	}
}

func TestPreviewTrimsSurroundingWhitespace(t *testing.T) {
	item := NewTextItem("  padded  ")
	if got := item.Preview(); got != "padded" {
		t.Fatalf("got %q", got)
	}
}

func TestPreviewOfEmptyText(t *testing.T) {
	item := NewTextItem("")
	if got := item.Preview(); got != "" {
		t.Fatalf("got %q", got)
	}
}

func TestPreviewImageWithoutDimensions(t *testing.T) {
	item := Item{Kind: KindImage}
	if got := item.Preview(); got != "Image" {
		t.Fatalf("got %q", got)
	}
}

func TestPreviewOnlyTruncatesAtTheFirstLineBreak(t *testing.T) {
	item := NewTextItem("one\ntwo\nthree")
	if got := item.Preview(); got != "one …" {
		t.Fatalf("got %q", got)
	}
}
