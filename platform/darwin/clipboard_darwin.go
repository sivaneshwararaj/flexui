//go:build darwin

package darwin

import "github.com/ebitengine/purego/objc"

// Clipboard is the general pasteboard (plain text).
type Clipboard struct{}

const pasteboardTypeString = "public.utf8-plain-text"

func pasteboard() objc.ID { return class("NSPasteboard").Send(sel("generalPasteboard")) }

// ReadText implements platform.Clipboard.
func (Clipboard) ReadText() (string, error) {
	s := pasteboard().Send(sel("stringForType:"), nsString(pasteboardTypeString))
	debugf("pasteboard read %d bytes", len(goString(s)))
	return goString(s), nil
}

// WriteText implements platform.Clipboard.
func (Clipboard) WriteText(s string) error {
	pb := pasteboard()
	pb.Send(sel("clearContents"))
	pb.Send(sel("setString:forType:"), nsString(s), nsString(pasteboardTypeString))
	debugf("pasteboard wrote %d bytes", len(s))
	return nil
}
