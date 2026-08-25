package clipboard

import (
	"errors"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestPasteHelpersAreDefinedForThisOS(t *testing.T) {
	helpers := pasteHelpers()
	if len(helpers) == 0 {
		t.Fatalf("no paste helper defined for %s", runtime.GOOS)
	}
	for _, h := range helpers {
		if h.name == "" || len(h.args) == 0 {
			t.Fatalf("incomplete helper: %+v", h)
		}
	}
}

func TestFindPasteHelperPicksTheFirstInstalled(t *testing.T) {
	findPasteHelper()

	// Whatever was chosen must be the first candidate present on PATH, so
	// the preference order in pasteHelpers is what actually decides.
	var wantPath string
	for _, h := range pasteHelpers() {
		if path, err := exec.LookPath(h.name); err == nil {
			wantPath = path
			break
		}
	}

	if pastePath != wantPath {
		t.Fatalf("chose %q, want %q", pastePath, wantPath)
	}
	if wantPath == "" {
		t.Skip("no paste helper installed here; selection could not be exercised")
	}
	t.Logf("auto-paste will use %s", pastePath)
}

func TestYdotoolSocketRespectsTheEnvironment(t *testing.T) {
	t.Setenv("YDOTOOL_SOCKET", "/tmp/custom.sock")
	if got := ydotoolSocket(); got != "/tmp/custom.sock" {
		t.Fatalf("got %q", got)
	}

	t.Setenv("YDOTOOL_SOCKET", "")
	if got := ydotoolSocket(); got == "" || got == "/tmp/custom.sock" {
		t.Fatalf("fallback socket = %q", got)
	}
}

func TestWaitForSocketTimesOut(t *testing.T) {
	start := time.Now()
	if waitForSocket("/nonexistent/never.sock", 150*time.Millisecond) {
		t.Fatal("reported a socket that does not exist")
	}
	if elapsed := time.Since(start); elapsed < 100*time.Millisecond {
		t.Fatalf("returned after only %v, it did not wait", elapsed)
	}
}

func TestWaitForSocketFindsAListeningDaemon(t *testing.T) {
	path := filepath.Join(t.TempDir(), "present.sock")
	// ydotoold serves datagrams, so the probe has to find a dgram socket.
	conn, err := net.ListenPacket("unixgram", path)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	if !waitForSocket(path, time.Second) {
		t.Fatal("did not detect a listening daemon")
	}
}

func TestDaemonListeningRejectsAMissingSocket(t *testing.T) {
	if daemonListening(filepath.Join(t.TempDir(), "absent.sock")) {
		t.Fatal("reported a daemon on a path with no socket")
	}
}

// A stopped ydotoold leaves its socket file behind, so the file existing
// must not be mistaken for a daemon that is up.
func TestWaitForSocketRejectsAStaleSocketFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stale.sock")
	listener, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	// Leave the file behind on close, which is what a stopped ydotoold
	// does; Go would otherwise unlink it and hide the case under test.
	listener.(*net.UnixListener).SetUnlinkOnClose(false)
	listener.Close()

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("the socket file should still exist: %v", err)
	}
	if waitForSocket(path, 150*time.Millisecond) {
		t.Fatal("a leftover socket file was reported as a running daemon")
	}
}

func TestEveryHelperCarriesAHint(t *testing.T) {
	for _, h := range pasteHelpers() {
		if h.hint == "" {
			t.Fatalf("%s has no hint to show the user when it fails", h.name)
		}
	}
}

func TestRecoveryIsRateLimited(t *testing.T) {
	findPasteHelper()

	calls := 0
	restore := pasteChosen
	pasteChosen.recover = func() error { calls++; return nil }
	t.Cleanup(func() {
		pasteChosen = restore
		lastRecover = time.Time{}
	})

	lastRecover = time.Time{}
	if !attemptPasteRecovery() {
		t.Fatal("first attempt should run recovery")
	}
	if attemptPasteRecovery() {
		t.Fatal("second attempt should be held off by the cooldown")
	}
	if calls != 1 {
		t.Fatalf("recovery ran %d times, want 1", calls)
	}

	// Once the cooldown has passed a daemon that died later must be
	// recoverable again.
	lastRecover = time.Now().Add(-pasteRecoverCooldown - time.Second)
	if !attemptPasteRecovery() {
		t.Fatal("recovery should be allowed again after the cooldown")
	}
	if calls != 2 {
		t.Fatalf("recovery ran %d times, want 2", calls)
	}
}

func TestRecoveryFailureIsReportedAsSuch(t *testing.T) {
	findPasteHelper()

	restore := pasteChosen
	pasteChosen.recover = func() error { return errors.New("nope") }
	t.Cleanup(func() {
		pasteChosen = restore
		lastRecover = time.Time{}
	})

	lastRecover = time.Time{}
	if attemptPasteRecovery() {
		t.Fatal("a failed recovery must not report the helper as retryable")
	}
}
