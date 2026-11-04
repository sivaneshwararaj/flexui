//go:build windows

package windows

// Virtual-key codes.
const (
	vkBack    = 0x08
	vkTab     = 0x09
	vkReturn  = 0x0D
	vkShift   = 0x10
	vkControl = 0x11
	vkMenu    = 0x12
	vkEscape  = 0x1B
	vkSpace   = 0x20
	vkPrior   = 0x21
	vkNext    = 0x22
	vkEnd     = 0x23
	vkHome    = 0x24
	vkLeft    = 0x25
	vkUp      = 0x26
	vkRight   = 0x27
	vkDown    = 0x28
	vkDelete  = 0x2E
)

var vkNames = map[uintptr]string{
	vkBack: "Backspace", vkTab: "Tab", vkReturn: "Enter", vkEscape: "Escape",
	vkSpace: " ", vkPrior: "PageUp", vkNext: "PageDown", vkEnd: "End",
	vkHome: "Home", vkLeft: "ArrowLeft", vkUp: "ArrowUp", vkRight: "ArrowRight",
	vkDown: "ArrowDown", vkDelete: "Delete",
}

// keyName returns flexui's key name for a virtual key: special keys by
// name, letters as lower-case letters (for shortcuts such as Ctrl+A),
// digits as digits; "" for keys flexui ignores (modifiers, F-keys, ...).
// Typed text arrives separately through WM_CHAR.
func keyName(vk uintptr) string {
	if n, ok := vkNames[vk]; ok {
		return n
	}
	switch {
	case vk >= 'A' && vk <= 'Z':
		return string(rune(vk - 'A' + 'a'))
	case vk >= '0' && vk <= '9':
		return string(rune(vk))
	}
	return ""
}
