package clipboard

import (
	"context"
	"errors"

	system "golang.design/x/clipboard"
)

// x11Images is true when the library uses its X11 backend. That backend
// cannot serve an image that does not fit in one X11 request, so
// Pufferfish serves images with its own X11 selection owner then.
var x11Images bool

func initImageWriter() {
	x11Images = libraryUsesX11()
}

// writeImage places png on the system clipboard. The returned channel
// closes when another writer replaces the image.
func writeImage(png []byte) (<-chan struct{}, error) {
	if x11Images {
		replaced, err := serveX11Image(x11ClipboardSelection, png)
		if !errors.Is(err, errX11Unavailable) {
			return replaced, err
		}
		// The check in libraryUsesX11 can give a wrong result, for
		// example when the compositor did not answer in time. Without an
		// X server the library cannot use X11 either, so the image goes
		// through the library.
	}
	return system.Write(context.Background(), system.FmtImage, png)
}
