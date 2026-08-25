package clipboard

import (
	"context"
	"errors"
	"os"

	system "golang.design/x/clipboard"
)

// ErrUnavailable reports that the system clipboard could not be reached, so
// tracking and restoring are both impossible for this run.
var ErrUnavailable = errors.New("clipboard: system clipboard unavailable")

// Init prepares access to the system clipboard. It must succeed before any
// watcher is started or any item restored.
func Init() error {
	if err := system.Init(); err != nil {
		return errors.Join(ErrUnavailable, err)
	}
	return nil
}

// Put places item back on the system clipboard. The write is registered as
// Pufferfish's own, so the watcher does not recapture it.
func (w *Watcher) Put(item Item) error {
	format := system.FmtText
	buf := []byte(item.Text)

	if item.Kind == KindImage {
		path, ok := w.store.ImagePath(item)
		if !ok {
			return errors.New("clipboard: image file is missing")
		}
		png, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		format, buf = system.FmtImage, png
	}

	w.expectSelfWrite(hashBytes(buf))
	_, err := system.Write(context.Background(), format, buf)
	return err
}

// Clear empties the system clipboard so a paste after "clear all" doesn't
// bring back an item that was just removed from the history.
func (w *Watcher) Clear() error {
	_, err := system.Write(context.Background(), system.FmtText, []byte{})
	return err
}
