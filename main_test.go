package main

import (
	"testing"

	"fyne.io/fyne/v2/test"

	"github.com/iambpn/pufferfish/internal/clipboard"
)

func TestStorageDirUsesTheAppsRootURI(t *testing.T) {
	a := test.NewTempApp(t)

	dir := storageDir(a)
	if dir == "" {
		t.Fatal("want a non-empty storage dir for a test app with storage")
	}
}

func TestRestoreLastItemWithEmptyStoreDoesNothing(t *testing.T) {
	store := clipboard.NewStore(t.TempDir())
	watcher := clipboard.NewWatcher(store)

	restoreLastItem(store, watcher) // must not panic when there is nothing to restore
}

func TestRestoreLastItemPutsTheNewestItemBack(t *testing.T) {
	if err := clipboard.Init(); err != nil {
		t.Skipf("system clipboard unavailable: %v", err)
	}

	store := clipboard.NewStore(t.TempDir())
	store.Add(clipboard.NewTextItem("older"))
	store.Add(clipboard.NewTextItem("newest"))
	watcher := clipboard.NewWatcher(store)

	restoreLastItem(store, watcher)
}
