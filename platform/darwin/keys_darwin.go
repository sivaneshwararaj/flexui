//go:build darwin

package darwin

import "github.com/sivaneshwararaj/flexui/platform"

// keyNames maps macOS virtual key codes (Carbon kVK_*) of non-character
// keys to flexui key names. Character keys use the event's characters.
var keyNames = map[uint16]string{
	0x24: "Enter", 0x4C: "Enter", 0x30: "Tab", 0x31: " ", 0x33: "Backspace",
	0x75: "Delete", 0x35: "Escape", 0x7B: "ArrowLeft", 0x7C: "ArrowRight",
	0x7D: "ArrowDown", 0x7E: "ArrowUp", 0x73: "Home", 0x77: "End",
	0x74: "PageUp", 0x79: "PageDown",
}

// NSEventModifierFlags bits.
const (
	nsShift   = 1 << 17
	nsControl = 1 << 18
	nsOption  = 1 << 19
	nsCommand = 1 << 20
)

func mods(flags uint64) uint8 {
	var m uint8
	if flags&nsShift != 0 {
		m |= platform.ModShift
	}
	if flags&nsControl != 0 {
		m |= platform.ModCtrl
	}
	if flags&nsOption != 0 {
		m |= platform.ModAlt
	}
	if flags&nsCommand != 0 {
		m |= platform.ModMeta
	}
	return m
}
