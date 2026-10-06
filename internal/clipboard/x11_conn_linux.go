/*
A small X11 client for the selection owner in x11_image_linux.go.

It uses the request encoders and packet decoders of golang.design/x/x11, the
wire package that golang.design/x/clipboard is built on, and adds what that
package leaves to its caller: the connection, the sequence numbers and the
two messages that an INCR transfer needs.
*/
package clipboard

import (
	"bufio"
	"encoding/binary"
	"errors"
	"net"
	"os"
	"path/filepath"
	"time"

	x11wire "golang.design/x/x11"
)

const (
	x11OpChangeWindowAttributes = 2
	x11EventPropertyNotify      = 28

	// x11EventMaskAttribute selects the event mask in the value list of a
	// ChangeWindowAttributes request.
	x11EventMaskAttribute uint32 = 0x0800
	x11PropertyChangeMask uint32 = 0x00400000

	// x11PropertyDeleted is the state of a PropertyNotify event for a
	// property that was deleted.
	x11PropertyDeleted = 1

	// x11ChangePropertyHeader is the size in bytes of a ChangeProperty
	// request without its data.
	x11ChangePropertyHeader = 24

	// The X server can reset a connection during setup when clients connect
	// and disconnect quickly, so the setup is tried more than once.
	x11DialAttempts   = 3
	x11DialRetryDelay = 5 * time.Millisecond

	x11WindowAttempts = 8

	// x11SetupTimeout bounds the setup, so a server that does not answer
	// gives an error instead of a caller that waits forever.
	x11SetupTimeout = 5 * time.Second
)

var errX11Request = errors.New("clipboard: X server rejected the request")

// x11Conn is a connection to the X server with one window of its own.
type x11Conn struct {
	conn   net.Conn
	reader *bufio.Reader
	setup  x11wire.Setup
	ids    *x11wire.IDGen
	window uint32
	seq    uint16

	// events holds the events that arrived while reply or requestFailed
	// waited for a reply. nextEvent returns them first.
	events []x11wire.Packet
}

// x11PropertyEvent is the part of a PropertyNotify event that an INCR
// transfer needs.
type x11PropertyEvent struct {
	Window  uint32
	Atom    uint32
	Deleted bool
}

// dialX11 connects to the display named by DISPLAY and creates the window.
func dialX11() (*x11Conn, error) {
	display, err := x11wire.ParseDisplay(os.Getenv("DISPLAY"))
	if err != nil {
		return nil, err
	}
	authName, authData := x11Cookie(display.Num)

	var lastErr error
	for range x11DialAttempts {
		x, err := connectX11(display, authName, authData)
		if err == nil {
			return x, nil
		}
		lastErr = err
		time.Sleep(x11DialRetryDelay)
	}
	return nil, lastErr
}

func connectX11(display x11wire.Display, authName string, authData []byte) (*x11Conn, error) {
	x, err := handshakeX11(display, authName, authData)
	if err != nil && authName != "" {
		// A local server can refuse the cookie and still accept a
		// connection without authorization.
		x, err = handshakeX11(display, "", nil)
	}
	if err != nil {
		return nil, err
	}
	if err := x.createWindow(); err != nil {
		x.Close()
		return nil, err
	}
	// The setup is complete. From here the connection waits for events
	// without a time limit.
	x.conn.SetDeadline(time.Time{})
	return x, nil
}

func handshakeX11(display x11wire.Display, authName string, authData []byte) (*x11Conn, error) {
	conn, err := dialX11Socket(display)
	if err != nil {
		return nil, err
	}
	conn.SetDeadline(time.Now().Add(x11SetupTimeout))

	if _, err := conn.Write(x11wire.SetupRequest(authName, authData)); err != nil {
		conn.Close()
		return nil, err
	}
	reader := bufio.NewReader(conn)
	setup, err := x11wire.ReadSetup(reader)
	if err != nil {
		conn.Close()
		return nil, err
	}
	return &x11Conn{conn: conn, reader: reader, setup: setup, ids: x11wire.NewIDGen(setup)}, nil
}

func dialX11Socket(display x11wire.Display) (net.Conn, error) {
	if display.Net != "unix" {
		return net.Dial(display.Net, display.Addr)
	}
	conn, err := net.Dial("unix", display.Addr)
	if err == nil {
		return conn, nil
	}
	// Some servers listen only on the abstract socket of the same name.
	return net.Dial("unix", "@"+display.Addr)
}

// x11Cookie reads the authorization cookie for the display from XAUTHORITY,
// or from ~/.Xauthority. It returns an empty name when there is no cookie.
func x11Cookie(displayNum int) (name string, data []byte) {
	path := os.Getenv("XAUTHORITY")
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", nil
		}
		path = filepath.Join(home, ".Xauthority")
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return "", nil
	}
	entries, err := x11wire.ParseXauthority(content)
	if err != nil {
		return "", nil
	}
	host, _ := os.Hostname()
	return x11wire.ChooseCookie(entries, displayNum, host)
}

func (x *x11Conn) Close() error { return x.conn.Close() }

// maxPropertyData is the largest property value, in bytes, that fits in one
// ChangeProperty request on this connection.
func (x *x11Conn) maxPropertyData() int {
	maxRequest := int(x.setup.MaxReqLen) * 4
	return maxRequest - x11ChangePropertyHeader
}

// send writes one request and returns the sequence number that the server
// gives it.
func (x *x11Conn) send(request []byte) (uint16, error) {
	if _, err := x.conn.Write(request); err != nil {
		return 0, err
	}
	x.seq++
	return x.seq, nil
}

// reply reads packets until the reply to request seq arrives. Errors that
// belong to other requests are dropped.
func (x *x11Conn) reply(seq uint16) (x11wire.Packet, error) {
	for {
		packet, err := x11wire.ReadPacket(x.reader)
		if err != nil {
			return x11wire.Packet{}, err
		}
		if packet.IsEvent() {
			x.events = append(x.events, packet)
			continue
		}
		if packet.Sequence() != seq {
			continue
		}
		if packet.IsError() {
			return x11wire.Packet{}, errX11Request
		}
		return packet, nil
	}
}

// requestFailed reports whether the server answered request seq with an
// error. A request without a reply gives no answer when it succeeds, so
// this sends a second request that always has a reply and reads up to it.
func (x *x11Conn) requestFailed(seq uint16) (bool, error) {
	syncSeq, err := x.send(x11wire.GetInputFocus())
	if err != nil {
		return false, err
	}
	failed := false
	for {
		packet, err := x11wire.ReadPacket(x.reader)
		if err != nil {
			return false, err
		}
		if packet.IsEvent() {
			x.events = append(x.events, packet)
			continue
		}
		if packet.IsError() && packet.Sequence() == seq {
			failed = true
			continue
		}
		if packet.Sequence() == syncSeq {
			return failed, nil
		}
	}
}

// flush waits until the server has processed each request sent so far. The
// server can drop the last requests of a connection that closes before
// that.
func (x *x11Conn) flush() error {
	seq, err := x.send(x11wire.GetInputFocus())
	if err != nil {
		return err
	}
	_, err = x.reply(seq)
	return err
}

// nextEvent returns the next event from the server.
func (x *x11Conn) nextEvent() (x11wire.Packet, error) {
	if len(x.events) > 0 {
		packet := x.events[0]
		x.events = x.events[1:]
		return packet, nil
	}
	return x11wire.NextEvent(x.reader)
}

// createWindow creates the window that owns the selection. A new connection
// can get the resource ids of a connection that just closed, and the server
// rejects an id that it has not released yet, so a rejected id is replaced
// by the next one.
func (x *x11Conn) createWindow() error {
	for range x11WindowAttempts {
		window := x.ids.Next()
		createSeq, err := x.send(x11wire.CreateWindow(window, x.setup.Root))
		if err != nil {
			return err
		}
		failed, err := x.requestFailed(createSeq)
		if err != nil {
			return err
		}
		if !failed {
			x.window = window
			return nil
		}
	}
	return errors.New("clipboard: X server rejected every window id")
}

// intern returns the atom for name.
func (x *x11Conn) intern(name string) (uint32, error) {
	seq, err := x.send(x11wire.InternAtom(name, false))
	if err != nil {
		return 0, err
	}
	packet, err := x.reply(seq)
	if err != nil {
		return 0, err
	}
	return packet.Atom(), nil
}

// x11SelectEvents builds a ChangeWindowAttributes request that sets the
// events this connection receives for window.
func x11SelectEvents(window uint32, mask uint32) []byte {
	request := make([]byte, 16)
	request[0] = x11OpChangeWindowAttributes
	binary.LittleEndian.PutUint16(request[2:], uint16(len(request)/4))
	binary.LittleEndian.PutUint32(request[4:], window)
	binary.LittleEndian.PutUint32(request[8:], x11EventMaskAttribute)
	binary.LittleEndian.PutUint32(request[12:], mask)
	return request
}

// x11PropertyNotify decodes a PropertyNotify event.
func x11PropertyNotify(packet x11wire.Packet) x11PropertyEvent {
	raw := packet.Raw
	return x11PropertyEvent{
		Window:  binary.LittleEndian.Uint32(raw[4:]),
		Atom:    binary.LittleEndian.Uint32(raw[8:]),
		Deleted: raw[16] == x11PropertyDeleted,
	}
}
