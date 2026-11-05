//go:build windows

package windows

import (
	"fmt"
	"time"
	"unsafe"
)

// Clipboard is the Windows clipboard (CF_UNICODETEXT).
type Clipboard struct{ hwnd uintptr }

// open opens the clipboard, retrying for up to about 100 ms: another
// process (a clipboard manager, Windows clipboard history, the app that
// just wrote) can hold it open briefly, and OpenClipboard then fails.
func (c Clipboard) open() error {
	var err error
	for i := 0; i < 10; i++ {
		var r uintptr
		if r, _, err = pOpenClipboard.Call(c.hwnd); r != 0 {
			return nil
		}
		debugf("OpenClipboard attempt %d failed: %v", i+1, err)
		time.Sleep(10 * time.Millisecond)
	}
	return fmt.Errorf("windows: OpenClipboard: %v", err)
}

// ReadText implements platform.Clipboard.
func (c Clipboard) ReadText() (string, error) {
	if err := c.open(); err != nil {
		debugf("clipboard read: %v", err)
		return "", err
	}
	defer pCloseClipboard.Call()
	h, _, _ := pGetClipboardData.Call(cfUnicodeText)
	if h == 0 {
		return "", nil // no text on the clipboard
	}
	p, _, _ := pGlobalLock.Call(h)
	if p == 0 {
		return "", fmt.Errorf("windows: GlobalLock failed")
	}
	defer pGlobalUnlock.Call(h)
	// Bound the read by the block size: text without a terminating NUL
	// must not be over-read.
	size, _, _ := pGlobalSize.Call(h)
	s := decodeClipText(unsafe.Slice((*uint16)(ptr(p)), size/2))
	debugf("clipboard read %d bytes", len(s))
	return s, nil
}

// WriteText implements platform.Clipboard.
func (c Clipboard) WriteText(s string) error {
	u, err := encodeClipText(s)
	if err != nil {
		return err
	}
	if err := c.open(); err != nil {
		debugf("clipboard write: %v", err)
		return err
	}
	defer pCloseClipboard.Call()
	if r, _, err := pEmptyClipboard.Call(); r == 0 {
		debugf("clipboard write: EmptyClipboard: %v", err)
		return fmt.Errorf("windows: EmptyClipboard: %v", err)
	}
	size := uintptr(len(u)) * 2
	h, _, _ := pGlobalAlloc.Call(gmemMoveable, size)
	if h == 0 {
		return fmt.Errorf("windows: GlobalAlloc failed")
	}
	p, _, _ := pGlobalLock.Call(h)
	if p == 0 {
		pGlobalFree.Call(h)
		return fmt.Errorf("windows: GlobalLock failed")
	}
	copy(unsafe.Slice((*uint16)(ptr(p)), len(u)), u)
	pGlobalUnlock.Call(h)
	if r, _, err := pSetClipboardData.Call(cfUnicodeText, h); r == 0 {
		pGlobalFree.Call(h) // ownership was not transferred
		debugf("clipboard write: SetClipboardData: %v", err)
		return fmt.Errorf("windows: SetClipboardData: %v", err)
	}
	debugf("clipboard wrote %d bytes", len(s))
	return nil
}
