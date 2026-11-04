//go:build windows

package windows

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Clipboard is the Windows clipboard (CF_UNICODETEXT).
type Clipboard struct{ hwnd uintptr }

// ReadText implements platform.Clipboard.
func (c Clipboard) ReadText() (string, error) {
	if r, _, err := pOpenClipboard.Call(c.hwnd); r == 0 {
		return "", fmt.Errorf("windows: OpenClipboard: %v", err)
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
	s := windows.UTF16PtrToString((*uint16)(ptr(p)))
	debugf("clipboard read %d bytes", len(s))
	return s, nil
}

// WriteText implements platform.Clipboard.
func (c Clipboard) WriteText(s string) error {
	u, err := windows.UTF16FromString(s)
	if err != nil {
		return err
	}
	if r, _, err := pOpenClipboard.Call(c.hwnd); r == 0 {
		return fmt.Errorf("windows: OpenClipboard: %v", err)
	}
	defer pCloseClipboard.Call()
	pEmptyClipboard.Call()
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
		return fmt.Errorf("windows: SetClipboardData: %v", err)
	}
	debugf("clipboard wrote %d bytes", len(s))
	return nil
}
