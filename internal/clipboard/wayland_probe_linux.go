/*
golang.design/x/clipboard selects its Linux backend when it starts. It uses
Wayland when the compositor offers a data-control protocol, and X11 in all
other cases. The library does not report its selection, so this file does
the same check: it asks the compositor for its list of globals and looks for
a data-control manager.
*/
package clipboard

import (
	"encoding/binary"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"slices"
	"time"
)

const (
	waylandDisplayID  = 1
	waylandRegistryID = 2
	waylandCallbackID = 3

	waylandHeaderSize = 8

	waylandProbeTimeout = 2 * time.Second
)

// waylandDataControlManagers are the interfaces that make the library use
// its Wayland backend.
var waylandDataControlManagers = []string{
	"ext_data_control_manager_v1",
	"zwlr_data_control_manager_v1",
}

// libraryUsesX11 reports whether golang.design/x/clipboard uses its X11
// backend in this session.
func libraryUsesX11() bool {
	return !waylandOffersDataControl()
}

func waylandOffersDataControl() bool {
	path := waylandSocketPath()
	if path == "" {
		return false
	}
	conn, err := net.Dial("unix", path)
	if err != nil {
		return false
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(waylandProbeTimeout))

	found, err := waylandListsDataControl(conn)
	return err == nil && found
}

// waylandSocketPath returns the path of the compositor socket, or an empty
// string when the process is not in a Wayland session.
func waylandSocketPath() string {
	display := os.Getenv("WAYLAND_DISPLAY")
	if display == "" {
		return ""
	}
	if filepath.IsAbs(display) {
		return display
	}
	dir := os.Getenv("XDG_RUNTIME_DIR")
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, display)
}

// waylandListsDataControl asks the compositor on conn for its globals and
// reports whether one of them is a data-control manager.
func waylandListsDataControl(conn io.ReadWriter) (bool, error) {
	// wl_display.get_registry is opcode 1. wl_display.sync is opcode 0, and
	// the compositor answers it after it has sent all globals.
	getRegistry := waylandRequest(waylandDisplayID, 1, waylandRegistryID)
	sync := waylandRequest(waylandDisplayID, 0, waylandCallbackID)
	if _, err := conn.Write(append(getRegistry, sync...)); err != nil {
		return false, err
	}

	header := make([]byte, waylandHeaderSize)
	for {
		if _, err := io.ReadFull(conn, header); err != nil {
			return false, err
		}
		object := binary.LittleEndian.Uint32(header[0:])
		opcode := binary.LittleEndian.Uint16(header[4:])
		size := int(binary.LittleEndian.Uint16(header[6:]))
		if size < waylandHeaderSize {
			return false, errors.New("clipboard: invalid Wayland message")
		}
		body := make([]byte, size-waylandHeaderSize)
		if _, err := io.ReadFull(conn, body); err != nil {
			return false, err
		}

		switch {
		case object == waylandCallbackID:
			return false, nil
		case object == waylandDisplayID && opcode == 0:
			return false, errors.New("clipboard: Wayland protocol error")
		case object == waylandRegistryID && opcode == 0:
			name := waylandGlobalInterface(body)
			if slices.Contains(waylandDataControlManagers, name) {
				return true, nil
			}
		}
	}
}

// waylandRequest builds a request that has one new object id as its
// argument.
func waylandRequest(object uint32, opcode uint16, newID uint32) []byte {
	request := make([]byte, waylandHeaderSize+4)
	binary.LittleEndian.PutUint32(request[0:], object)
	binary.LittleEndian.PutUint16(request[4:], opcode)
	binary.LittleEndian.PutUint16(request[6:], uint16(len(request)))
	binary.LittleEndian.PutUint32(request[8:], newID)
	return request
}

// waylandGlobalInterface returns the interface name from the body of a
// wl_registry.global event. The body holds the numeric name, then the
// interface as a string with its length and a NUL at the end, then the
// version.
func waylandGlobalInterface(body []byte) string {
	const stringStart = 8
	if len(body) < stringStart {
		return ""
	}
	length := int(binary.LittleEndian.Uint32(body[4:]))
	if length < 1 || stringStart+length > len(body) {
		return ""
	}
	return string(body[stringStart : stringStart+length-1])
}
