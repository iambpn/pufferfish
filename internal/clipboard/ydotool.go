/*
ydotool talks to a long-running daemon over a unix socket, so a machine
with ydotool installed can still fail to paste simply because ydotoold is
not up. Starting it is worth doing because the daemon is what holds the
/dev/uinput handle, and it is safe to do because both ways of starting it
run as the user: nothing here escalates privileges. A setup where uinput
itself is out of reach is left to the README's udev rule rather than to a
password prompt from a tray app.

The daemon is deliberately never stopped again. It is a shared service -
dictation and automation tools use the same one - so shutting down a daemon
Pufferfish did not start, or that something else has since begun using,
would break them mid-session.
*/
package clipboard

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

const (
	// ydotoolSocketWait is how long the daemon is given to start serving.
	// Sending the keystroke before it does fails silently, so the retry
	// has to wait for it rather than race it.
	ydotoolSocketWait = 2 * time.Second
	ydotoolPollEvery  = 50 * time.Millisecond

	// ydotoolDialTimeout bounds a single readiness probe.
	ydotoolDialTimeout = 250 * time.Millisecond

	ydotoolStartTimeout = 5 * time.Second
)

// ydotoolSocket is where ydotool and ydotoold agree to meet. Respecting
// YDOTOOL_SOCKET keeps a custom setup working; otherwise this is the
// modern default, and passing it explicitly to both sides means a
// package whose built-in default differs cannot put them out of step.
func ydotoolSocket() string {
	if custom := os.Getenv("YDOTOOL_SOCKET"); custom != "" {
		return custom
	}
	return filepath.Join("/run/user", fmt.Sprint(os.Getuid()), ".ydotool_socket")
}

// startYdotoold brings the daemon up and waits for its socket to appear.
func startYdotoold() error {
	socket := ydotoolSocket()

	// Prefer systemd: it owns the daemon's lifetime, restarts it if it
	// dies, and starting an already-running unit is a no-op.
	err := runQuietly(ydotoolStartTimeout, "systemctl", "--user", "start", "ydotoold")
	if err != nil {
		// No unit installed, or no systemd at all. Run the daemon
		// directly instead, pointed at the same socket.
		if err = spawnYdotoold(socket); err != nil {
			return err
		}
	}

	if !waitForSocket(socket, ydotoolSocketWait) {
		return fmt.Errorf("ydotoold is not serving on %s", socket)
	}
	return nil
}

// spawnYdotoold starts the daemon detached, so it outlives Pufferfish the
// way the systemd-managed one does.
func spawnYdotoold(socket string) error {
	path, err := exec.LookPath("ydotoold")
	if err != nil {
		return errors.New("ydotoold is not installed")
	}

	cmd := exec.Command(path, "--socket-path="+socket)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = nil, nil, nil
	if err := cmd.Start(); err != nil {
		return err
	}

	// Nothing waits on the daemon, so reap the handle and let it run on.
	go cmd.Process.Release()
	return nil
}

// waitForSocket reports whether the daemon is serving. It connects rather
// than checking that the socket file exists, because ydotoold leaves its
// socket behind when it stops: an existing file proves only that the
// daemon ran at some point, not that anything is there now.
func waitForSocket(path string, within time.Duration) bool {
	deadline := time.Now().Add(within)
	for {
		if daemonListening(path) {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(ydotoolPollEvery)
	}
}

// daemonListening probes the socket. ydotoold serves datagrams, so that is
// what gets dialled; a build serving a stream socket answers "protocol
// wrong type" instead, which still means a daemon is bound to the path.
// Only a refused connection - or no socket at all - means nothing is there.
func daemonListening(path string) bool {
	conn, err := net.DialTimeout("unixgram", path, ydotoolDialTimeout)
	if err == nil {
		conn.Close()
		return true
	}
	if errors.Is(err, os.ErrNotExist) || errors.Is(err, syscall.ECONNREFUSED) {
		return false
	}
	return true
}

func runQuietly(timeout time.Duration, name string, args ...string) error {
	path, err := exec.LookPath(name)
	if err != nil {
		return err
	}

	cmd := exec.Command(path, args...)
	hideConsole(cmd)
	if err := cmd.Start(); err != nil {
		return err
	}

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	select {
	case err := <-done:
		return err
	case <-time.After(timeout):
		cmd.Process.Kill()
		return fmt.Errorf("%s timed out", name)
	}
}
