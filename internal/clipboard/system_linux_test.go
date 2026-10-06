package clipboard

import (
	"bytes"
	"image"
	"image/png"
	"math/rand"
	"testing"
)

// noisePNG returns a PNG that stays large, because random pixels do not
// compress.
func noisePNG(t *testing.T, width, height int) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	source := rand.New(rand.NewSource(1))
	source.Read(img.Pix)
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestPutServesAnImageAboveTheX11RequestLimit(t *testing.T) {
	if err := Init(); err != nil {
		t.Skipf("system clipboard unavailable: %v", err)
	}
	if !x11Images {
		t.Skip("the clipboard library does not use its X11 backend here")
	}
	reader := newX11TestReader(t)
	reader.selection = x11ClipboardSelection

	store := NewStore(t.TempDir())
	t.Cleanup(store.Flush)
	image := noisePNG(t, 700, 700)
	if len(image) <= reader.conn.maxPropertyData() {
		t.Fatalf("the test image has only %d bytes", len(image))
	}
	if !store.AddImage(image) {
		t.Fatal("AddImage failed")
	}

	w := NewWatcher(store)
	if err := w.Put(store.Items()[0]); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	got, parts := reader.readImage()
	if !bytes.Equal(got, image) {
		t.Fatalf("got %d bytes that differ from the %d bytes of the image", len(got), len(image))
	}
	if parts < 2 {
		t.Fatalf("want the image in more than one part, got %d", parts)
	}
}
