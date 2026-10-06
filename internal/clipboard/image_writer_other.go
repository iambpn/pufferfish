//go:build !linux

package clipboard

import (
	"context"

	system "golang.design/x/clipboard"
)

func initImageWriter() {}

// writeImage places png on the system clipboard. The returned channel
// closes when another writer replaces the image.
func writeImage(png []byte) (<-chan struct{}, error) {
	return system.Write(context.Background(), system.FmtImage, png)
}
