//go:build windows

package windows

import (
	"fmt"
	"os"
	"runtime"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Debug enables a log of every Win32 step, message and frame on stderr
// (the stages set it with -debug).
var Debug bool

var start = time.Now()

func debugf(format string, args ...any) {
	if Debug {
		fmt.Fprintf(os.Stderr, "[windows %7.3fs] %s\n", time.Since(start).Seconds(), fmt.Sprintf(format, args...))
	}
}

func init() {
	// Windows and their message loop belong to the thread that created
	// them; package init runs on the main thread.
	runtime.LockOSThread()
}

var (
	user32   = windows.NewLazySystemDLL("user32.dll")
	gdi32    = windows.NewLazySystemDLL("gdi32.dll")
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")

	pRegisterClassExW              = user32.NewProc("RegisterClassExW")
	pCreateWindowExW               = user32.NewProc("CreateWindowExW")
	pDefWindowProcW                = user32.NewProc("DefWindowProcW")
	pDestroyWindow                 = user32.NewProc("DestroyWindow")
	pShowWindow                    = user32.NewProc("ShowWindow")
	pUpdateWindow                  = user32.NewProc("UpdateWindow")
	pGetMessageW                   = user32.NewProc("GetMessageW")
	pTranslateMessage              = user32.NewProc("TranslateMessage")
	pDispatchMessageW              = user32.NewProc("DispatchMessageW")
	pPostQuitMessage               = user32.NewProc("PostQuitMessage")
	pGetClientRect                 = user32.NewProc("GetClientRect")
	pLoadCursorW                   = user32.NewProc("LoadCursorW")
	pSetProcessDpiAwarenessContext = user32.NewProc("SetProcessDpiAwarenessContext")
	pGetDpiForWindow               = user32.NewProc("GetDpiForWindow")
	pAdjustWindowRectExForDpi      = user32.NewProc("AdjustWindowRectExForDpi")
	pSetWindowPos                  = user32.NewProc("SetWindowPos")
	pInvalidateRect                = user32.NewProc("InvalidateRect")
	pBeginPaint                    = user32.NewProc("BeginPaint")
	pEndPaint                      = user32.NewProc("EndPaint")
	pTrackMouseEvent               = user32.NewProc("TrackMouseEvent")
	pScreenToClient                = user32.NewProc("ScreenToClient")
	pSetCapture                    = user32.NewProc("SetCapture")
	pReleaseCapture                = user32.NewProc("ReleaseCapture")
	pGetKeyState                   = user32.NewProc("GetKeyState")
	pOpenClipboard                 = user32.NewProc("OpenClipboard")
	pCloseClipboard                = user32.NewProc("CloseClipboard")
	pEmptyClipboard                = user32.NewProc("EmptyClipboard")
	pGetClipboardData              = user32.NewProc("GetClipboardData")
	pSetClipboardData              = user32.NewProc("SetClipboardData")
	pSetWindowTextW                = user32.NewProc("SetWindowTextW")
	pSetDIBitsToDevice             = gdi32.NewProc("SetDIBitsToDevice")
	pGlobalAlloc                   = kernel32.NewProc("GlobalAlloc")
	pGlobalFree                    = kernel32.NewProc("GlobalFree")
	pGlobalLock                    = kernel32.NewProc("GlobalLock")
	pGlobalUnlock                  = kernel32.NewProc("GlobalUnlock")
	pGlobalSize                    = kernel32.NewProc("GlobalSize")
	pGetModuleHandleW              = kernel32.NewProc("GetModuleHandleW")
)

// Load resolves every Win32 function used, so that a missing one is
// reported up front (stage 1). SetProcessDpiAwarenessContext and the
// *ForDpi functions need Windows 10 1703 or later.
func Load() error {
	for _, p := range []*windows.LazyProc{
		pRegisterClassExW, pCreateWindowExW, pDefWindowProcW, pDestroyWindow, pShowWindow,
		pUpdateWindow, pGetMessageW, pTranslateMessage, pDispatchMessageW, pPostQuitMessage,
		pGetClientRect, pLoadCursorW, pSetProcessDpiAwarenessContext, pGetDpiForWindow,
		pAdjustWindowRectExForDpi, pSetWindowPos, pInvalidateRect, pBeginPaint, pEndPaint,
		pTrackMouseEvent, pScreenToClient, pSetCapture, pReleaseCapture, pGetKeyState,
		pOpenClipboard, pCloseClipboard, pEmptyClipboard, pGetClipboardData, pSetClipboardData,
		pSetWindowTextW, pSetDIBitsToDevice, pGlobalAlloc, pGlobalFree, pGlobalLock,
		pGlobalUnlock, pGlobalSize, pGetModuleHandleW,
	} {
		if err := p.Find(); err != nil {
			return fmt.Errorf("windows: %w", err)
		}
		debugf("resolved %s at %#x", p.Name, p.Addr())
	}
	return nil
}

// OSVersion returns the Windows version (for the stage 1 check).
func OSVersion() string {
	v := windows.RtlGetVersion()
	return fmt.Sprintf("Windows %d.%d build %d", v.MajorVersion, v.MinorVersion, v.BuildNumber)
}

// Win32 structures (64-bit layouts; x/sys does not define these).

type wndClassExW struct {
	cbSize        uint32
	style         uint32
	lpfnWndProc   uintptr
	cbClsExtra    int32
	cbWndExtra    int32
	hInstance     uintptr
	hIcon         uintptr
	hCursor       uintptr
	hbrBackground uintptr
	lpszMenuName  *uint16
	lpszClassName *uint16
	hIconSm       uintptr
}

type point struct{ X, Y int32 }

type msg struct {
	hwnd     uintptr
	message  uint32
	wParam   uintptr
	lParam   uintptr
	time     uint32
	pt       point
	lPrivate uint32
}

type rect struct{ Left, Top, Right, Bottom int32 }

type paintStruct struct {
	hdc         uintptr
	fErase      int32
	rcPaint     rect
	fRestore    int32
	fIncUpdate  int32
	rgbReserved [32]byte
}

type bitmapInfoHeader struct {
	biSize          uint32
	biWidth         int32
	biHeight        int32
	biPlanes        uint16
	biBitCount      uint16
	biCompression   uint32
	biSizeImage     uint32
	biXPelsPerMeter int32
	biYPelsPerMeter int32
	biClrUsed       uint32
	biClrImportant  uint32
}

type bitmapInfo struct {
	header bitmapInfoHeader
	colors [1]uint32
}

type trackMouseEvent struct {
	cbSize      uint32
	dwFlags     uint32
	hwndTrack   uintptr
	dwHoverTime uint32
}

// Messages and constants.
const (
	wmDestroy       = 0x0002
	wmSize          = 0x0005
	wmPaint         = 0x000F
	wmClose         = 0x0010
	wmEraseBkgnd    = 0x0014
	wmKeyDown       = 0x0100
	wmKeyUp         = 0x0101
	wmChar          = 0x0102
	wmSysKeyDown    = 0x0104
	wmSysKeyUp      = 0x0105
	wmMouseMove     = 0x0200
	wmLButtonDown   = 0x0201
	wmLButtonUp     = 0x0202
	wmMouseWheel    = 0x020A
	wmMouseHWheel   = 0x020E
	wmMouseLeave    = 0x02A3
	wmDpiChanged    = 0x02E0
	csHRedraw       = 0x0002
	csVRedraw       = 0x0001
	wsOverlappedWin = 0x00CF0000
	cwUseDefault    = 0x80000000
	swShow          = 5
	idcArrow        = 32512
	swpNoZOrder     = 0x0004
	swpNoActivate   = 0x0010
	swpNoMove       = 0x0002
	tmeLeave        = 0x00000002
	dibRGBColors    = 0
	biRGB           = 0
	cfUnicodeText   = 13
	gmemMoveable    = 0x0002
	// DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2 is the handle value -4.
	dpiAwarenessPerMonitorV2 = ^uintptr(3)
)

// ptr converts an address returned by a Win32 call into a pointer (the
// memory is not Go's; this form keeps vet's unsafeptr check quiet).
func ptr(addr uintptr) unsafe.Pointer { return *(*unsafe.Pointer)(unsafe.Pointer(&addr)) }
