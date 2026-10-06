package clipboard

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"slices"
	"testing"
	"time"

	x11wire "golang.design/x/x11"
)

// The tests use a selection of their own, so the content of the real
// clipboard does not change.
const x11TestSelection = "PUFFERFISH_TEST_SELECTION"

// x11TestReader is a requestor that follows the ICCCM, with INCR. It reads
// a selection the same way as an application that pastes.
type x11TestReader struct {
	t         *testing.T
	conn      *x11Conn
	selection string
	property  uint32
	incr      uint32
}

func newX11TestReader(t *testing.T) *x11TestReader {
	t.Helper()
	conn, err := dialX11()
	if err != nil {
		t.Skipf("X server unavailable: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	conn.conn.SetDeadline(time.Now().Add(20 * time.Second))

	r := &x11TestReader{t: t, conn: conn, selection: x11TestSelection}
	r.property = r.intern("PUFFERFISH_TEST_DATA")
	r.incr = r.intern("INCR")
	if _, err := conn.send(x11SelectEvents(conn.window, x11PropertyChangeMask)); err != nil {
		t.Fatal(err)
	}
	return r
}

func (r *x11TestReader) intern(name string) uint32 {
	r.t.Helper()
	atom, err := r.conn.intern(name)
	if err != nil {
		r.t.Fatal(err)
	}
	return atom
}

// request asks for target and returns the first answer. incr is true when
// the owner announced an INCR transfer, and readParts then returns the
// data.
func (r *x11TestReader) request(target string) (value []byte, incr bool, refused bool) {
	r.t.Helper()
	selection := r.intern(r.selection)
	convert := x11wire.ConvertSelection(r.conn.window, selection, r.intern(target), r.property, x11wire.CurrentTime)
	if _, err := r.conn.send(convert); err != nil {
		r.t.Fatal(err)
	}

	if r.waitNotify().Property == x11wire.None {
		return nil, false, true
	}
	propertyType, value := r.takeProperty()
	return value, propertyType == r.incr, false
}

// waitNotify returns the answer of the owner to a request.
func (r *x11TestReader) waitNotify() x11wire.SelectionNotifyEvent {
	r.t.Helper()
	for {
		packet, err := r.conn.nextEvent()
		if err != nil {
			r.t.Fatalf("no answer from the selection owner: %v", err)
		}
		if packet.EventCode() == x11wire.EventSelectionNotify {
			return packet.SelectionNotify()
		}
	}
}

// readParts reads the parts of an INCR transfer until the empty part.
func (r *x11TestReader) readParts() (data []byte, parts int) {
	r.t.Helper()
	for {
		packet, err := r.conn.nextEvent()
		if err != nil {
			r.t.Fatalf("the transfer stopped after %d parts: %v", parts, err)
		}
		if packet.EventCode() != x11EventPropertyNotify {
			continue
		}
		event := x11PropertyNotify(packet)
		if event.Atom != r.property || event.Deleted {
			continue
		}

		_, part := r.takeProperty()
		if len(part) == 0 {
			return data, parts
		}
		data = append(data, part...)
		parts++
	}
}

// takeProperty reads the property and deletes it.
func (r *x11TestReader) takeProperty() (propertyType uint32, value []byte) {
	r.t.Helper()
	seq, err := r.conn.send(x11wire.GetProperty(true, r.conn.window, r.property, 0, 0, 0xffffffff))
	if err != nil {
		r.t.Fatal(err)
	}
	reply, err := r.conn.reply(seq)
	if err != nil {
		r.t.Fatal(err)
	}
	propertyType = binary.LittleEndian.Uint32(reply.Raw[8:])
	// Copy the value, because the reply buffer is not kept.
	return propertyType, bytes.Clone(reply.PropertyValue())
}

// readImage reads image/png and reports the number of requests that
// carried the data.
func (r *x11TestReader) readImage() (data []byte, parts int) {
	r.t.Helper()
	value, incr, refused := r.request("image/png")
	if refused {
		r.t.Fatal("the owner refused image/png")
	}
	if !incr {
		return value, 1
	}
	return r.readParts()
}

func randomBytes(t *testing.T, size int) []byte {
	t.Helper()
	data := make([]byte, size)
	if _, err := rand.Read(data); err != nil {
		t.Fatal(err)
	}
	return data
}

func TestX11ImageOwnerServesImagesOfEachSize(t *testing.T) {
	reader := newX11TestReader(t)
	partSize := reader.conn.maxPropertyData()

	cases := []struct {
		name      string
		size      int
		wantParts int
	}{
		{"small image in one request", 1000, 1},
		{"largest image for one request", partSize, 1},
		{"one byte above the limit", partSize + 1, 2},
		{"exactly two full parts", 2 * partSize, 2},
		{"large image", 5*partSize + 12345, 6},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			image := randomBytes(t, tc.size)
			if _, err := serveX11Image(x11TestSelection, image); err != nil {
				t.Fatalf("serveX11Image failed: %v", err)
			}

			got, parts := reader.readImage()
			if !bytes.Equal(got, image) {
				t.Fatalf("got %d bytes that differ from the %d bytes served", len(got), len(image))
			}
			if parts != tc.wantParts {
				t.Fatalf("want %d parts, got %d", tc.wantParts, parts)
			}
		})
	}
}

func TestX11ImageOwnerServesTheSameImageAgain(t *testing.T) {
	reader := newX11TestReader(t)
	image := randomBytes(t, 3*reader.conn.maxPropertyData())
	if _, err := serveX11Image(x11TestSelection, image); err != nil {
		t.Fatalf("serveX11Image failed: %v", err)
	}

	for paste := 1; paste <= 3; paste++ {
		got, _ := reader.readImage()
		if !bytes.Equal(got, image) {
			t.Fatalf("paste %d returned different data", paste)
		}
	}
}

func TestX11ImageOwnerListsItsTargets(t *testing.T) {
	reader := newX11TestReader(t)
	if _, err := serveX11Image(x11TestSelection, randomBytes(t, 100)); err != nil {
		t.Fatalf("serveX11Image failed: %v", err)
	}

	value, _, refused := reader.request("TARGETS")
	if refused {
		t.Fatal("the owner refused TARGETS")
	}
	var atoms []uint32
	for offset := 0; offset+4 <= len(value); offset += 4 {
		atoms = append(atoms, binary.LittleEndian.Uint32(value[offset:]))
	}
	if !slices.Contains(atoms, reader.intern("image/png")) {
		t.Fatalf("TARGETS does not list image/png: %v", atoms)
	}
}

func TestX11ImageOwnerRefusesOtherTargets(t *testing.T) {
	reader := newX11TestReader(t)
	if _, err := serveX11Image(x11TestSelection, randomBytes(t, 100)); err != nil {
		t.Fatalf("serveX11Image failed: %v", err)
	}

	if _, _, refused := reader.request("UTF8_STRING"); !refused {
		t.Fatal("want a refusal for a target that the owner does not offer")
	}
}

func TestX11ImageOwnerUsesTheTargetAsPropertyForAnObsoleteRequestor(t *testing.T) {
	reader := newX11TestReader(t)
	image := randomBytes(t, 1000)
	if _, err := serveX11Image(x11TestSelection, image); err != nil {
		t.Fatalf("serveX11Image failed: %v", err)
	}

	// An obsolete requestor gives no property.
	selection := reader.intern(x11TestSelection)
	pngType := reader.intern("image/png")
	convert := x11wire.ConvertSelection(reader.conn.window, selection, pngType, x11wire.None, x11wire.CurrentTime)
	if _, err := reader.conn.send(convert); err != nil {
		t.Fatal(err)
	}

	if got := reader.waitNotify().Property; got != pngType {
		t.Fatalf("want the target atom %d as the property, got %d", pngType, got)
	}
	reader.property = pngType
	if _, got := reader.takeProperty(); !bytes.Equal(got, image) {
		t.Fatalf("got %d bytes that differ from the %d bytes served", len(got), len(image))
	}
}

func TestServeX11ImageReportsAMissingXServer(t *testing.T) {
	t.Setenv("DISPLAY", "")

	_, err := serveX11Image(x11TestSelection, []byte("image"))
	if !errors.Is(err, errX11Unavailable) {
		t.Fatalf("want errX11Unavailable, got %v", err)
	}
}

func TestX11ImageOwnerEndsWhenAnotherOwnerTakesTheSelection(t *testing.T) {
	newX11TestReader(t) // skips the test when there is no X server

	replaced, err := serveX11Image(x11TestSelection, randomBytes(t, 100))
	if err != nil {
		t.Fatalf("serveX11Image failed: %v", err)
	}
	select {
	case <-replaced:
		t.Fatal("the owner ended before another owner took the selection")
	case <-time.After(100 * time.Millisecond):
	}

	if _, err := serveX11Image(x11TestSelection, randomBytes(t, 100)); err != nil {
		t.Fatalf("serveX11Image failed: %v", err)
	}
	select {
	case <-replaced:
	case <-time.After(5 * time.Second):
		t.Fatal("the first owner did not end")
	}
}

func TestX11ImageOwnerCompletesATransferAfterItLostTheSelection(t *testing.T) {
	reader := newX11TestReader(t)
	image := randomBytes(t, 4*reader.conn.maxPropertyData())
	replaced, err := serveX11Image(x11TestSelection, image)
	if err != nil {
		t.Fatalf("serveX11Image failed: %v", err)
	}

	_, incr, refused := reader.request("image/png")
	if refused || !incr {
		t.Fatalf("want an INCR transfer, got incr=%v refused=%v", incr, refused)
	}

	// Another owner takes the selection while the transfer is in progress.
	if _, err := serveX11Image(x11TestSelection, randomBytes(t, 100)); err != nil {
		t.Fatalf("serveX11Image failed: %v", err)
	}

	got, _ := reader.readParts()
	if !bytes.Equal(got, image) {
		t.Fatalf("got %d bytes that differ from the %d bytes served", len(got), len(image))
	}
	select {
	case <-replaced:
	case <-time.After(x11TransferTimeout + 5*time.Second):
		t.Fatal("the first owner did not end after its transfer")
	}
}

func TestX11SelectEventsBuildsAChangeWindowAttributesRequest(t *testing.T) {
	got := x11SelectEvents(0x01020304, x11PropertyChangeMask)

	want := []byte{
		2, 0, 4, 0, // opcode, unused byte, length in 4-byte units
		0x04, 0x03, 0x02, 0x01, // window
		0x00, 0x08, 0x00, 0x00, // value mask: event mask
		0x00, 0x00, 0x40, 0x00, // event mask: property change
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("got % x, want % x", got, want)
	}
}

func TestX11PropertyNotifyDecodesTheEvent(t *testing.T) {
	raw := make([]byte, 32)
	raw[0] = x11EventPropertyNotify
	binary.LittleEndian.PutUint32(raw[4:], 77)
	binary.LittleEndian.PutUint32(raw[8:], 88)
	raw[16] = x11PropertyDeleted

	got := x11PropertyNotify(x11wire.Packet{Raw: raw})
	want := x11PropertyEvent{Window: 77, Atom: 88, Deleted: true}
	if got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}
