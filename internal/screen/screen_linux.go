//go:build (linux || freebsd || openbsd || netbsd) && !android

package screen

import (
	"encoding/binary"
	"io"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	x11 "golang.design/x/x11"
)

// x11DialTimeout bounds a single connection attempt to the X server, so a
// socket that never accepts can't block the caller indefinitely.
const x11DialTimeout = 500 * time.Millisecond

// getGeometryOpcode is the X11 core protocol request code for GetGeometry.
const getGeometryOpcode = 14

// Size asks the X server for the root window's geometry, which is the
// primary screen's resolution in pixels. This reaches the real display size
// even in a Wayland session, as long as its compositor runs XWayland -
// which GNOME and KDE both do by default.
//
// It reports ok=false when no X server is reachable at all (a pure Wayland
// session with no XWayland running) or the request fails for any other
// reason, so the caller can fall back to a fixed default position.
func Size() (width, height int, ok bool) {
	conn, root, err := dial()
	if err != nil {
		return 0, 0, false
	}
	defer conn.Close()

	req := make([]byte, 8)
	req[0] = getGeometryOpcode
	binary.LittleEndian.PutUint16(req[2:], uint16(len(req)/4))
	binary.LittleEndian.PutUint32(req[4:], root)
	if _, err := conn.Write(req); err != nil {
		return 0, 0, false
	}

	reply := make([]byte, 32)
	if _, err := io.ReadFull(conn, reply); err != nil {
		return 0, 0, false
	}
	if reply[0] != 1 { // not a normal reply - e.g. an X11 error packet
		return 0, 0, false
	}

	w := binary.LittleEndian.Uint16(reply[16:])
	h := binary.LittleEndian.Uint16(reply[18:])
	return int(w), int(h), true
}

// dial connects to the X server named by $DISPLAY and completes the
// connection setup handshake, returning the root window id GetGeometry
// targets.
func dial() (net.Conn, uint32, error) {
	d, err := x11.ParseDisplay(os.Getenv("DISPLAY"))
	if err != nil {
		return nil, 0, err
	}
	name, data := xauthCookie(d.Num)

	conn, root, err := dialOnce(d, name, data)
	if err != nil && name != "" {
		// Retry with no authorization: common for local unix sockets, which
		// often don't check it.
		conn, root, err = dialOnce(d, "", nil)
	}
	return conn, root, err
}

func dialOnce(d x11.Display, name string, data []byte) (net.Conn, uint32, error) {
	conn, err := net.DialTimeout(d.Net, d.Addr, x11DialTimeout)
	if err != nil && d.Net == "unix" && runtime.GOOS == "linux" {
		// Fall back to the Linux abstract-namespace socket some setups use
		// instead of the filesystem path.
		conn, err = net.DialTimeout("unix", "@/tmp/.X11-unix/X"+strconv.Itoa(d.Num), x11DialTimeout)
	}
	if err != nil {
		return nil, 0, err
	}

	if _, err := conn.Write(x11.SetupRequest(name, data)); err != nil {
		conn.Close()
		return nil, 0, err
	}
	setup, err := x11.ReadSetup(conn)
	if err != nil {
		conn.Close()
		return nil, 0, err
	}
	return conn, setup.Root, nil
}

// xauthCookie reads the MIT-MAGIC-COOKIE-1 for the given display from
// $XAUTHORITY (or ~/.Xauthority). It returns ("", nil) when none applies, in
// which case dial connects without authorization.
func xauthCookie(displayNum int) (string, []byte) {
	path := os.Getenv("XAUTHORITY")
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", nil
		}
		path = filepath.Join(home, ".Xauthority")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "", nil
	}
	entries, err := x11.ParseXauthority(b)
	if err != nil {
		return "", nil
	}
	host, _ := os.Hostname()
	return x11.ChooseCookie(entries, displayNum, host)
}
