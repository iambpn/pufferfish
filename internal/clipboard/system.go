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

// ready reports whether Init has succeeded. golang.design/x/clipboard's own
// docs warn that Write can panic when Init hasn't - so Put/Clear check it
// before ever calling into system.Write.
var ready bool

// Init prepares access to the system clipboard. It must succeed before any
// watcher is started or any item restored.
func Init() error {
	if err := system.Init(); err != nil {
		return errors.Join(ErrUnavailable, err)
	}
	initImageWriter()
	ready = true
	return nil
}

// Put places item back on the system clipboard. The watch is paused while
// the item is there, so the watcher does not recapture it.
//
// Put can wait for the system clipboard, so do not call it on the UI
// goroutine.
func (w *Watcher) Put(item Item) error {
	if !ready {
		return ErrUnavailable
	}

	if item.Kind != KindImage {
		return w.writeText(item.Text)
	}

	path, ok := w.store.ImagePath(item)
	if !ok {
		return errors.New("clipboard: image file is missing")
	}
	png, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return w.ownClipboard(func() (<-chan struct{}, error) {
		return writeImage(png)
	})
}

// Clear empties the system clipboard so a paste after "clear all" doesn't
// bring back an item that was just removed from the history.
//
// Clear can wait for the system clipboard, so do not call it on the UI
// goroutine.
func (w *Watcher) Clear() error {
	if !ready {
		return ErrUnavailable
	}
	return w.writeText("")
}

func (w *Watcher) writeText(text string) error {
	return w.ownClipboard(func() (<-chan struct{}, error) {
		return system.Write(context.Background(), system.FmtText, []byte(text))
	})
}
