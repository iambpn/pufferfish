//go:build (linux || freebsd || openbsd || netbsd) && !android

package screen

import (
	"encoding/binary"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"time"

	x11 "golang.design/x/x11"
)

// x11DialTimeout bounds a single connection attempt to the X server, so a
// socket that never accepts can't block the caller indefinitely.
const x11DialTimeout = 500 * time.Millisecond

// getGeometryOpcode is the X11 core protocol request code for GetGeometry.
const getGeometryOpcode = 14

// queryExtensionOpcode is the X11 core protocol request code for
// QueryExtension, used to learn RandR's assigned major opcode.
const queryExtensionOpcode = 98

// randrExtensionName is what the X server registers the RandR extension
// under.
const randrExtensionName = "RANDR"

// RandR request minor-opcodes, stable since RandR 1.2 (GetOutputInfo,
// GetCrtcInfo) and 1.5 (GetOutputPrimary) - old enough that any RandR
// server encountered today supports them.
const (
	randrGetOutputInfo    = 9
	randrGetCrtcInfo      = 20
	randrGetOutputPrimary = 31
)

// sizeCache holds the result of the first successful Size query. The
// screen resolution essentially never changes mid-session, so every call
// after the first is answered from here instead of repeating the blocking
// X11 round trip (up to ~1s on a slow/unreachable DISPLAY) on the calling
// goroutine - typically the single Fyne UI goroutine, every time the
// history window is positioned.
var (
	sizeCacheMu    sync.Mutex
	sizeCacheValid bool
	sizeCacheW     int
	sizeCacheH     int
)

// Size reports the primary monitor's resolution in pixels, via the RandR
// extension. This reaches the real display size even in a Wayland session,
// as long as its compositor runs XWayland - which GNOME and KDE both do by
// default.
//
// When RandR isn't available, or no primary output is configured, it falls
// back to the root window's geometry instead - the union of every output on
// a multi-monitor desktop rather than any one monitor's own resolution, but
// still the best information available.
//
// It reports ok=false when no X server is reachable at all (a pure Wayland
// session with no XWayland running) or every attempt above fails, so the
// caller can fall back to a fixed default position. A successful result is
// cached; see sizeCache.
func Size() (width, height int, ok bool) {
	sizeCacheMu.Lock()
	if sizeCacheValid {
		w, h := sizeCacheW, sizeCacheH
		sizeCacheMu.Unlock()
		return w, h, true
	}
	sizeCacheMu.Unlock()

	w, h, ok := querySize()
	if ok {
		sizeCacheMu.Lock()
		sizeCacheValid, sizeCacheW, sizeCacheH = true, w, h
		sizeCacheMu.Unlock()
	}
	return w, h, ok
}

// querySize performs the X11 round trip Size caches the result of.
func querySize() (width, height int, ok bool) {
	conn, root, err := dial()
	if err != nil {
		return 0, 0, false
	}
	defer conn.Close()

	if w, h, ok := primaryMonitorSize(conn, root); ok {
		return w, h, true
	}
	return rootGeometry(conn, root)
}

// resetSizeCache clears the cached screen size. It exists for tests that
// need Size to perform a fresh query regardless of what an earlier test
// already cached.
func resetSizeCache() {
	sizeCacheMu.Lock()
	sizeCacheValid = false
	sizeCacheMu.Unlock()
}

// rootGeometry asks the X server for the root window's geometry - the
// union of every output's resolution on a multi-monitor RandR setup, not
// any single monitor's own resolution.
func rootGeometry(conn net.Conn, root uint32) (width, height int, ok bool) {
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

// primaryMonitorSize asks RandR for the primary output's actual monitor
// geometry. It reports ok=false for any reason that can't be determined -
// no RandR extension, no primary output configured, that output not
// currently driving a CRTC, or any request failing outright - so the
// caller can fall back to the root window's geometry instead.
func primaryMonitorSize(conn net.Conn, root uint32) (width, height int, ok bool) {
	major, ok := queryExtension(conn, randrExtensionName)
	if !ok {
		return 0, 0, false
	}

	output, ok := randrGetOutputPrimaryReply(conn, major, root)
	if !ok || output == 0 { // 0 is None: no primary output configured
		return 0, 0, false
	}

	crtc, ok := randrGetOutputInfoReply(conn, major, output)
	if !ok || crtc == 0 { // 0 is None: output isn't driving any CRTC
		return 0, 0, false
	}

	w, h, ok := randrGetCrtcInfoReply(conn, major, crtc)
	if !ok || w == 0 || h == 0 {
		return 0, 0, false
	}
	return int(w), int(h), true
}

func randrGetOutputPrimaryReply(conn net.Conn, major byte, window uint32) (output uint32, ok bool) {
	reply, err := randrRoundTrip(conn, major, randrGetOutputPrimary, window)
	if err != nil {
		return 0, false
	}
	return binary.LittleEndian.Uint32(reply[8:]), true
}

func randrGetOutputInfoReply(conn net.Conn, major byte, output uint32) (crtc uint32, ok bool) {
	reply, err := randrRoundTrip(conn, major, randrGetOutputInfo, output, 0)
	if err != nil {
		return 0, false
	}
	return binary.LittleEndian.Uint32(reply[12:]), true
}

func randrGetCrtcInfoReply(conn net.Conn, major byte, crtc uint32) (width, height uint16, ok bool) {
	reply, err := randrRoundTrip(conn, major, randrGetCrtcInfo, crtc, 0)
	if err != nil {
		return 0, 0, false
	}
	w := binary.LittleEndian.Uint16(reply[16:])
	h := binary.LittleEndian.Uint16(reply[18:])
	return w, h, true
}

// randrRoundTrip sends a RandR extension request - byte 0 the extension's
// own major opcode learned from queryExtension, byte 1 the RandR
// minor-opcode naming the specific call, followed by its CARD32 arguments -
// and returns the reply, fully drained of whatever trailing data it
// carries so the connection's request/reply stream stays in sync for
// whichever request comes next.
func randrRoundTrip(conn net.Conn, major byte, minorOpcode byte, args ...uint32) ([]byte, error) {
	req := make([]byte, 4+4*len(args))
	req[0] = major
	req[1] = minorOpcode
	binary.LittleEndian.PutUint16(req[2:], uint16(len(req)/4))
	for i, v := range args {
		binary.LittleEndian.PutUint32(req[4+4*i:], v)
	}
	if _, err := conn.Write(req); err != nil {
		return nil, err
	}

	reply, err := readReply(conn)
	if err != nil {
		return nil, err
	}
	if reply[0] != 1 { // not a normal reply - e.g. an X11 error packet
		return nil, errNotAReply
	}
	return reply, nil
}

// errNotAReply marks an X11 response that parsed cleanly but wasn't a
// normal Reply packet (e.g. an Error packet instead).
var errNotAReply = errors.New("screen: not a reply packet")

// readReply reads one X11 reply or error packet off conn: the fixed
// 32-byte header, followed by whatever extra data its length field (bytes
// 4:8, a count of 4-byte units) says to expect. Reading exactly this much
// even when the extra data goes unused keeps the connection's
// request/reply stream in sync for whatever request comes next.
func readReply(conn net.Conn) ([]byte, error) {
	head := make([]byte, 32)
	if _, err := io.ReadFull(conn, head); err != nil {
		return nil, err
	}
	extra := binary.LittleEndian.Uint32(head[4:8])
	if extra == 0 {
		return head, nil
	}
	tail := make([]byte, extra*4)
	if _, err := io.ReadFull(conn, tail); err != nil {
		return nil, err
	}
	return append(head, tail...), nil
}

// queryExtension asks the X server for name's assigned major request
// opcode via the core QueryExtension request, reporting ok=false when the
// extension isn't present (or the query fails outright).
func queryExtension(conn net.Conn, name string) (majorOpcode byte, ok bool) {
	pad := (4 - len(name)%4) % 4
	req := make([]byte, 8+len(name)+pad)
	req[0] = queryExtensionOpcode
	binary.LittleEndian.PutUint16(req[2:], uint16(len(req)/4))
	binary.LittleEndian.PutUint16(req[4:], uint16(len(name)))
	copy(req[8:], name)
	if _, err := conn.Write(req); err != nil {
		return 0, false
	}

	reply, err := readReply(conn)
	if err != nil {
		return 0, false
	}
	// Every X11 reply's first 8 bytes are [type, detail, sequence(2),
	// length(4)]; QueryExtension leaves "detail" unused and starts its own
	// fields at byte 8: present(1), major-opcode(1), first-event(1),
	// first-error(1).
	if reply[0] != 1 || reply[8] == 0 { // not a reply, or extension absent
		return 0, false
	}
	return reply[9], true
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
