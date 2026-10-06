/*
Watcher tracks the system clipboard and feeds what it sees into the Store.

Both formats are watched for the whole lifetime of the watcher and image
events are dropped when capturing images is switched off, so toggling that
preference never has to tear the watch down and rebuild it.

The watch is paused while content that Pufferfish wrote itself - an item the
user picked from the history - is on the system clipboard. Pufferfish thus
does not capture its own writes, and does not read its own content back
again and again. The watch continues when another application replaces that
content.
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
	captureImages bool

	// enabled is true when the user wants the clipboard watched.
	enabled bool
	// cancel stops the watch. It is nil while no watch runs.
	cancel context.CancelFunc
	// ownWrites counts the writes that Pufferfish made itself, and gives
	// each write its number.
	ownWrites uint64
	// held is the number of the own write whose content is on the
	// clipboard. It is zero when the clipboard holds no own content.
	held uint64
	// writing is true while an own write is in progress.
	writing bool

	// writeMu lets one own write run at a time.
	writeMu sync.Mutex
}

// NewWatcher creates a stopped watcher writing into store.
func NewWatcher(store *Store) *Watcher {
	return &Watcher{store: store}
}

// SetCaptureImages controls whether copied images join the history.
func (w *Watcher) SetCaptureImages(v bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.captureImages = v
}

// Start begins watching the clipboard. Calling it while already running
// does nothing. While Pufferfish's own content is on the clipboard, the
// watch begins when another application replaces that content.
func (w *Watcher) Start() {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.enabled = true
	w.startLocked(false)
}

// Stop ends watching. Calling it while already stopped does nothing.
func (w *Watcher) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.enabled = false
	w.stopLocked()
}

// startLocked starts the watch unless it runs already or Pufferfish's own
// content is on the clipboard. The caller must hold mu.
func (w *Watcher) startLocked(catchUp bool) {
	paused := w.writing || w.held != 0
	if w.cancel != nil || paused {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	w.cancel = cancel

	go w.run(ctx, catchUp)
}

// stopLocked stops the watch if it runs. The caller must hold mu.
func (w *Watcher) stopLocked() {
	if w.cancel == nil {
		return
	}
	w.cancel()
	w.cancel = nil
}

// SetEnabled starts or stops the watcher to match the preference.
func (w *Watcher) SetEnabled(v bool) {
	if v {
		w.Start()
	} else {
		w.Stop()
	}
}

// run feeds clipboard changes into the store until ctx ends. With catchUp
// it first records the content that is on the clipboard now. The library
// reports only the changes after the watch starts, so without this step the
// copy that replaced Pufferfish's own content would be lost.
func (w *Watcher) run(ctx context.Context, catchUp bool) {
	events := system.Watch(ctx, system.FmtText, system.FmtImage)

	if catchUp {
		for _, format := range []system.Format{system.FmtText, system.FmtImage} {
			content, err := system.Read(ctx, format)
			if err != nil || ctx.Err() != nil {
				continue
			}
			w.handle(system.Data{Format: format, Bytes: content})
		}
	}

	for data := range events {
		// An event that arrives after the watch stopped can hold the
		// content that Pufferfish wrote itself.
		if ctx.Err() != nil {
			return
		}
		w.handle(data)
	}
}

// ownClipboard runs write, which places Pufferfish's own content on the
// system clipboard and returns a channel that closes when another writer
// replaces that content. The watch is paused from before the write until
// that channel closes.
func (w *Watcher) ownClipboard(write func() (<-chan struct{}, error)) error {
	w.writeMu.Lock()
	defer w.writeMu.Unlock()

	w.mu.Lock()
	w.ownWrites++
	id := w.ownWrites
	w.writing = true
	w.stopLocked()
	w.mu.Unlock()

	replaced, err := write()
	written := err == nil && replaced != nil

	w.mu.Lock()
	w.writing = false
	// A failed write leaves the earlier content on the clipboard, so only
	// a write that worked changes held.
	if written {
		w.held = id
	}
	w.resumeLocked()
	w.mu.Unlock()

	if !written {
		return err
	}
	go func() {
		<-replaced
		w.releaseClipboard(id)
	}()
	return nil
}

// releaseClipboard records that the content of own write id is not on the
// clipboard now, and continues the watch. It does nothing for an old write,
// because a newer own write then holds the clipboard.
func (w *Watcher) releaseClipboard(id uint64) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if id != w.held {
		return
	}
	w.held = 0
	w.resumeLocked()
}

// resumeLocked continues the watch after an own write, if the user wants
// the clipboard watched. The caller must hold mu.
func (w *Watcher) resumeLocked() {
	if w.enabled {
		w.startLocked(true)
	}
}

func (w *Watcher) handle(data system.Data) {
	if len(data.Bytes) == 0 {
		return
	}

	// The store is only ever changed on the UI goroutine, so windows bound
	// to it can refresh straight from its change notification. Work that
	// does not touch the store - decoding and resizing an image - stays
	// here on the watch goroutine.
	switch data.Format {
	case system.FmtText:
		text := string(data.Bytes)
		fyne.Do(func() { w.store.Add(NewTextItem(text)) })
	case system.FmtImage:
		if !w.capturesImages() {
			return
		}
		item, ok := w.store.PrepareImage(data.Bytes)
		if !ok {
			return
		}
		fyne.Do(func() { w.store.Add(item) })
	}
}

// ClearAll empties store and the system clipboard together - the shared
// behavior behind every "clear history" entry point (the history window's
// button, the tray menu item), so they can't silently diverge.
//
// The write to the system clipboard can wait for other applications, and
// the callers are on the UI goroutine, so that write runs in the background.
func ClearAll(store *Store, watcher *Watcher) {
	store.Clear()
	go func() {
		if err := watcher.Clear(); err != nil {
			fyne.LogError("could not clear the system clipboard", err)
		}
	}()
}

func (w *Watcher) capturesImages() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.captureImages
}
