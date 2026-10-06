/*
The X11 selection owner that serves a copied image.

One X11 request holds a limited number of bytes, usually 262140, and
golang.design/x/clipboard sends a selection in one request. An image above
that limit damages its connection and no application can paste the image.

This owner follows the ICCCM instead. It sends an image that fits in one
request directly. For a larger image it answers with the INCR type and then
sends the image in parts. The requestor deletes the property each time it
has read a part, and each deletion triggers the next part. An empty part
ends the transfer.
*/
package clipboard

import (
	"encoding/binary"
	"errors"
	"time"

	x11wire "golang.design/x/x11"
)

const (
	x11ClipboardSelection = "CLIPBOARD"

	// x11TransferTimeout is how long the transfers that are in progress can
	// continue after another owner took the selection.
	x11TransferTimeout = 5 * time.Second
)

// errX11Unavailable reports that there is no connection to an X server.
var errX11Unavailable = errors.New("clipboard: X server unavailable")

// x11Transfer identifies an INCR transfer by the window and the property
// that receive it.
type x11Transfer struct {
	requestor uint32
	property  uint32
}

type x11ImageOwner struct {
	conn *x11Conn
	png  []byte

	selection uint32
	targets   uint32
	pngType   uint32
	incr      uint32

	// partSize is the largest number of bytes sent in one request.
	partSize int

	// sent holds the number of bytes already sent for each INCR transfer in
	// progress. A requestor that stops in the middle of a transfer leaves
	// its entry here until the owner ends.
	sent map[x11Transfer]int

	// replaced closes when the selection has a different owner, or no
	// owner. released records that it is closed.
	replaced chan struct{}
	released bool
}

// serveX11Image takes ownership of the named selection and serves png to
// each application that asks for it. The returned channel closes when
// another owner replaces the selection, or when the connection ends.
func serveX11Image(selection string, png []byte) (<-chan struct{}, error) {
	conn, err := dialX11()
	if err != nil {
		return nil, errors.Join(errX11Unavailable, err)
	}
	owner, err := newX11ImageOwner(conn, selection, png)
	if err != nil {
		conn.Close()
		return nil, err
	}

	go func() {
		defer conn.Close()
		owner.serve()
		// The last part of a transfer must reach the server before the
		// connection closes.
		conn.flush()
	}()
	return owner.replaced, nil
}

func newX11ImageOwner(conn *x11Conn, selection string, png []byte) (*x11ImageOwner, error) {
	owner := &x11ImageOwner{
		conn:     conn,
		png:      png,
		partSize: conn.maxPropertyData(),
		sent:     map[x11Transfer]int{},
		replaced: make(chan struct{}),
	}

	conn.conn.SetDeadline(time.Now().Add(x11SetupTimeout))
	defer conn.conn.SetDeadline(time.Time{})

	atoms := []struct {
		name string
		atom *uint32
	}{
		{selection, &owner.selection},
		{"TARGETS", &owner.targets},
		{"image/png", &owner.pngType},
		{"INCR", &owner.incr},
	}
	for _, entry := range atoms {
		atom, err := conn.intern(entry.name)
		if err != nil {
			return nil, err
		}
		*entry.atom = atom
	}

	setOwner := x11wire.SetSelectionOwner(conn.window, owner.selection, x11wire.CurrentTime)
	if _, err := conn.send(setOwner); err != nil {
		return nil, err
	}
	seq, err := conn.send(x11wire.GetSelectionOwner(owner.selection))
	if err != nil {
		return nil, err
	}
	reply, err := conn.reply(seq)
	if err != nil {
		return nil, err
	}
	if reply.SelectionOwner() != conn.window {
		return nil, errors.New("clipboard: could not take the X11 selection")
	}
	return owner, nil
}

// serve answers requests until another owner takes the selection. The
// transfers in progress at that time continue until they are complete or
// x11TransferTimeout ends.
func (o *x11ImageOwner) serve() {
	// The selection has no owner when the connection ends.
	defer o.release()

	for {
		if o.released && len(o.sent) == 0 {
			return
		}
		packet, err := o.conn.nextEvent()
		if err != nil {
			return
		}

		switch packet.EventCode() {
		case x11wire.EventSelectionClear:
			o.release()
			o.conn.conn.SetReadDeadline(time.Now().Add(x11TransferTimeout))
		case x11wire.EventSelectionRequest:
			err = o.answer(packet.SelectionRequest())
		case x11EventPropertyNotify:
			err = o.sendNextPart(x11PropertyNotify(packet))
		}
		if err != nil {
			return
		}
	}
}

// release reports that this owner does not hold the selection now.
func (o *x11ImageOwner) release() {
	if o.released {
		return
	}
	o.released = true
	close(o.replaced)
}

// answer replies to one request for the selection.
func (o *x11ImageOwner) answer(request x11wire.SelectionRequestEvent) error {
	if request.Selection != o.selection {
		return nil
	}
	property := request.Property
	if property == x11wire.None {
		// An obsolete requestor gives no property. The ICCCM tells the
		// owner to use the target atom as the property then.
		property = request.Target
	}
	notify := x11wire.SelectionNotify{
		Time:      request.Time,
		Requestor: request.Requestor,
		Selection: request.Selection,
		Target:    request.Target,
		Property:  property,
	}

	var err error
	switch request.Target {
	case o.targets:
		list := x11wire.AtomList(o.targets, o.pngType)
		_, err = o.conn.send(x11wire.ChangeProperty(request.Requestor, property, x11wire.AtomATOM, 32, list))
	case o.pngType:
		err = o.startImage(x11Transfer{requestor: request.Requestor, property: property})
	default:
		// An empty property tells the requestor that this target is not
		// available.
		notify.Property = x11wire.None
	}
	if err != nil {
		return err
	}

	_, err = o.conn.send(x11wire.SendSelectionNotify(notify))
	return err
}

// startImage writes the image to the property of the requestor, or starts
// an INCR transfer when the image does not fit in one request.
func (o *x11ImageOwner) startImage(transfer x11Transfer) error {
	if len(o.png) <= o.partSize {
		_, err := o.conn.send(x11wire.ChangeProperty(transfer.requestor, transfer.property, o.pngType, 8, o.png))
		return err
	}

	// The deletions of the property pace the transfer, so the owner must
	// receive them before it announces the transfer.
	if _, err := o.conn.send(x11SelectEvents(transfer.requestor, x11PropertyChangeMask)); err != nil {
		return err
	}
	// The value of an INCR property is the size of the data in bytes.
	size := binary.LittleEndian.AppendUint32(nil, uint32(len(o.png)))
	if _, err := o.conn.send(x11wire.ChangeProperty(transfer.requestor, transfer.property, o.incr, 32, size)); err != nil {
		return err
	}
	o.sent[transfer] = 0
	return nil
}

// sendNextPart sends the next part of an INCR transfer after the requestor
// deleted the property.
func (o *x11ImageOwner) sendNextPart(event x11PropertyEvent) error {
	transfer := x11Transfer{requestor: event.Window, property: event.Atom}
	offset, running := o.sent[transfer]
	if !running || !event.Deleted {
		return nil
	}

	end := min(offset+o.partSize, len(o.png))
	part := o.png[offset:end]
	if _, err := o.conn.send(x11wire.ChangeProperty(transfer.requestor, transfer.property, o.pngType, 8, part)); err != nil {
		return err
	}
	if len(part) > 0 {
		o.sent[transfer] = end
		return nil
	}

	// The empty part told the requestor that the image is complete.
	delete(o.sent, transfer)
	return o.stopEvents(transfer.requestor)
}

// stopEvents stops the property events of window, unless a different
// transfer to the same window still needs them.
func (o *x11ImageOwner) stopEvents(window uint32) error {
	for transfer := range o.sent {
		if transfer.requestor == window {
			return nil
		}
	}
	_, err := o.conn.send(x11SelectEvents(window, 0))
	return err
}
