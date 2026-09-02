package clipboard

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

// Kind distinguishes the sorts of content Pufferfish captures.
type Kind string

const (
	KindText  Kind = "text"
	KindImage Kind = "image"
)

// Item is a single captured clipboard entry. Image bytes are not held here:
// they are written to a file in the store's directory and referenced by
// ImageFile, so a long history stays cheap in memory and on reload.
//
// ThumbFile is a small downscaled copy saved next to the image. History
// cards draw this instead of the original, so Fyne never has to decode a
// full-size image just to show a tiny thumbnail. It is empty for images
// saved before thumbnails existed, or when writing the thumbnail failed;
// ThumbPath then falls back to the original.
type Item struct {
	Kind       Kind      `json:"kind"`
	Text       string    `json:"text,omitempty"`
	ImageFile  string    `json:"imageFile,omitempty"`
	ThumbFile  string    `json:"thumbFile,omitempty"`
	Width      int       `json:"width,omitempty"`
	Height     int       `json:"height,omitempty"`
	Hash       string    `json:"hash"`
	CapturedAt time.Time `json:"capturedAt"`
}

// NewTextItem builds a text entry captured now.
func NewTextItem(text string) Item {
	return Item{
		Kind:       KindText,
		Text:       text,
		Hash:       hashBytes([]byte(text)),
		CapturedAt: time.Now(),
	}
}

// Preview is the single line shown for this item in the history list.
func (i Item) Preview() string {
	if i.Kind == KindImage {
		if i.Width > 0 && i.Height > 0 {
			return fmt.Sprintf("Image %d × %d", i.Width, i.Height)
		}
		return "Image"
	}
	text := strings.TrimSpace(i.Text)
	if idx := strings.IndexByte(text, '\n'); idx >= 0 {
		text = strings.TrimSpace(text[:idx]) + " …"
	}
	return text
}

func hashBytes(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
