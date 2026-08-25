package clipboard

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadWithNoStorageDirIsANoOp(t *testing.T) {
	s := NewStore("")
	s.Load()

	if got := s.Items(); len(got) != 0 {
		t.Fatalf("got %+v", got)
	}
}

func TestLoadWithMissingHistoryFileStartsEmpty(t *testing.T) {
	s := NewStore(t.TempDir())
	s.Load()

	if got := s.Items(); len(got) != 0 {
		t.Fatalf("got %+v", got)
	}
}

func TestLoadDiscardsUnreadableHistory(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, historyFileName), []byte("not json"), 0o600); err != nil {
		t.Fatal(err)
	}

	s := NewStore(dir)
	s.Load()

	if got := s.Items(); len(got) != 0 {
		t.Fatalf("got %+v", got)
	}
}

func TestSaveLockedWithNoStorageDirWritesNoFile(t *testing.T) {
	s := NewStore("")
	s.Add(NewTextItem("a"))
	// Nothing to assert on disk since there is no dir; Add must simply not
	// panic when saveLocked runs with s.dir == "".
}

func TestLoadPersistsAcrossStores(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)
	s.Add(NewTextItem("one"))
	s.Add(NewTextItem("two"))

	reloaded := NewStore(dir)
	reloaded.Load()

	items := reloaded.Items()
	if len(items) != 2 || items[0].Text != "two" || items[1].Text != "one" {
		t.Fatalf("got %+v", items)
	}
}
