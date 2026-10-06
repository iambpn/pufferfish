package clipboard

import (
	"bytes"
	"encoding/binary"
	"io"
	"testing"
)

// waylandEvent builds one event from the compositor.
func waylandEvent(object uint32, opcode uint16, body []byte) []byte {
	event := make([]byte, waylandHeaderSize, waylandHeaderSize+len(body))
	binary.LittleEndian.PutUint32(event[0:], object)
	binary.LittleEndian.PutUint16(event[4:], opcode)
	binary.LittleEndian.PutUint16(event[6:], uint16(waylandHeaderSize+len(body)))
	return append(event, body...)
}

// waylandGlobal builds a wl_registry.global event for iface.
func waylandGlobal(name uint32, iface string) []byte {
	text := append([]byte(iface), 0)
	for len(text)%4 != 0 {
		text = append(text, 0)
	}
	body := binary.LittleEndian.AppendUint32(nil, name)
	body = binary.LittleEndian.AppendUint32(body, uint32(len(iface)+1))
	body = append(body, text...)
	body = binary.LittleEndian.AppendUint32(body, 1) // version
	return waylandEvent(waylandRegistryID, 0, body)
}

// waylandDone builds the wl_callback.done event that ends the list.
func waylandDone() []byte {
	return waylandEvent(waylandCallbackID, 0, make([]byte, 4))
}

// fakeCompositor answers a probe with events and records the requests.
type fakeCompositor struct {
	io.Reader
	requests bytes.Buffer
}

func (c *fakeCompositor) Write(p []byte) (int, error) { return c.requests.Write(p) }

func newFakeCompositor(events ...[]byte) *fakeCompositor {
	return &fakeCompositor{Reader: bytes.NewReader(bytes.Join(events, nil))}
}

func TestWaylandProbeFindsADataControlManager(t *testing.T) {
	for _, manager := range waylandDataControlManagers {
		compositor := newFakeCompositor(
			waylandGlobal(1, "wl_compositor"),
			waylandGlobal(2, manager),
			waylandDone(),
		)

		found, err := waylandListsDataControl(compositor)
		if err != nil || !found {
			t.Fatalf("%s: got found=%v err=%v", manager, found, err)
		}
	}
}

func TestWaylandProbeReportsNoDataControlManager(t *testing.T) {
	compositor := newFakeCompositor(
		waylandGlobal(1, "wl_compositor"),
		waylandGlobal(2, "wl_data_device_manager"),
		waylandDone(),
	)

	found, err := waylandListsDataControl(compositor)
	if err != nil || found {
		t.Fatalf("got found=%v err=%v", found, err)
	}
}

func TestWaylandProbeSendsGetRegistryAndSync(t *testing.T) {
	compositor := newFakeCompositor(waylandDone())
	if _, err := waylandListsDataControl(compositor); err != nil {
		t.Fatal(err)
	}

	want := []byte{
		1, 0, 0, 0, 1, 0, 12, 0, 2, 0, 0, 0, // wl_display.get_registry(2)
		1, 0, 0, 0, 0, 0, 12, 0, 3, 0, 0, 0, // wl_display.sync(3)
	}
	if got := compositor.requests.Bytes(); !bytes.Equal(got, want) {
		t.Fatalf("got % x, want % x", got, want)
	}
}

func TestWaylandProbeFailsWhenTheListIsIncomplete(t *testing.T) {
	compositor := newFakeCompositor(waylandGlobal(1, "wl_compositor"))

	if _, err := waylandListsDataControl(compositor); err == nil {
		t.Fatal("want an error when the compositor closes before the end of the list")
	}
}

func TestWaylandProbeFailsOnADisplayError(t *testing.T) {
	compositor := newFakeCompositor(waylandEvent(waylandDisplayID, 0, make([]byte, 12)))

	if _, err := waylandListsDataControl(compositor); err == nil {
		t.Fatal("want an error for a wl_display.error event")
	}
}

func TestWaylandGlobalInterfaceRejectsAShortBody(t *testing.T) {
	if got := waylandGlobalInterface([]byte{1, 0, 0, 0}); got != "" {
		t.Fatalf("got %q", got)
	}
	// The string length is larger than the body.
	body := []byte{1, 0, 0, 0, 200, 0, 0, 0, 'w', 'l', 0, 0}
	if got := waylandGlobalInterface(body); got != "" {
		t.Fatalf("got %q", got)
	}
}
