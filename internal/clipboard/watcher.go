/*
Watcher tracks the system clipboard and feeds what it sees into the Store.

Both formats are watched for the whole lifetime of the watcher and image
events are dropped when capturing images is switched off, so toggling that
preference never has to tear the watch down and rebuild it.

Writes Pufferfish makes itself - restoring an item the user picked from the
history - come back through the same watch. Their hash is remembered in
selfWrites so they are recognised and skipped instead of being recaptured.
*/
package clipboard

import (
	"context"
	"sync"

	"fyne.io/fyne/v2"
	system "golang.design/x/clipboard"
)

// Watcher observes the system clipboard and records what is copied.
type Watcher struct {
	store *Store

	mu            sync.Mutex
	cancel        context.CancelFunc
	captureImages bool
	selfWrites    map[string]bool
}

// NewWatcher creates a stopped watcher writing into store.
func NewWatcher(store *Store) *Watcher {
	return &Watcher{store: store, selfWrites: map[string]bool{}}
}

// SetCaptureImages controls whether copied images join the history.
func (w *Watcher) SetCaptureImages(v bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.captureImages = v
}

// Start begins watching the clipboard. Calling it while already running
// does nothing.
func (w *Watcher) Start() {
	w.mu.Lock()
	if w.cancel != nil {
		w.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	w.cancel = cancel
	w.mu.Unlock()

	go w.run(ctx)
}

// Stop ends watching. Calling it while already stopped does nothing.
func (w *Watcher) Stop() {
	w.mu.Lock()
	cancel := w.cancel
	w.cancel = nil
	w.mu.Unlock()

	if cancel != nil {
		cancel()
	}
}

// SetEnabled starts or stops the watcher to match the preference.
func (w *Watcher) SetEnabled(v bool) {
	if v {
		w.Start()
	} else {
		w.Stop()
	}
}

func (w *Watcher) run(ctx context.Context) {
	for data := range system.Watch(ctx, system.FmtText, system.FmtImage) {
		w.handle(data)
	}
}

func (w *Watcher) handle(data system.Data) {
	if len(data.Bytes) == 0 {
		return
	}
	if w.consumeSelfWrite(hashBytes(data.Bytes)) {
		return
	}

	// The store is only ever mutated on the UI goroutine, so the windows
	// bound to it can refresh straight from its change notification.
	switch data.Format {
	case system.FmtText:
		text := string(data.Bytes)
		fyne.Do(func() { w.store.Add(NewTextItem(text)) })
	case system.FmtImage:
		if !w.capturesImages() {
			return
		}
		png := data.Bytes
		fyne.Do(func() { w.store.AddImage(png) })
	}
}

// ClearAll empties store and the system clipboard together - the shared
// behavior behind every "clear history" entry point (the history window's
// button, the tray menu item), so they can't silently diverge.
func ClearAll(store *Store, watcher *Watcher) {
	store.Clear()
	if err := watcher.Clear(); err != nil {
		fyne.LogError("could not clear the system clipboard", err)
	}
}

func (w *Watcher) capturesImages() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.captureImages
}

// expectSelfWrite marks a hash as written by Pufferfish, so the watch event
// it causes is ignored rather than recaptured.
func (w *Watcher) expectSelfWrite(hash string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.selfWrites[hash] = true
}

func (w *Watcher) consumeSelfWrite(hash string) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.selfWrites[hash] {
		return false
	}
	delete(w.selfWrites, hash)
	return true
}
