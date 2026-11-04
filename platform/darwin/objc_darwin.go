//go:build darwin

package darwin

import (
	"fmt"
	"os"
	"runtime"
	"time"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

// Debug enables a log of every AppKit call and event on stderr (the
// stages set it with -debug).
var Debug bool

var start = time.Now()

func debugf(format string, args ...any) {
	if Debug {
		fmt.Fprintf(os.Stderr, "[darwin %7.3fs] %s\n", time.Since(start).Seconds(), fmt.Sprintf(format, args...))
	}
}

func init() {
	// AppKit must be used from the main thread; package init runs on it.
	runtime.LockOSThread()
}

// NSPoint, NSSize and NSRect mirror the CoreGraphics structs (CGFloat is
// float64 on 64-bit macOS).
type NSPoint struct{ X, Y float64 }
type NSSize struct{ Width, Height float64 }
type NSRect struct {
	Origin NSPoint
	Size   NSSize
}

var (
	loaded  bool
	loadErr error

	// CoreFoundation / CoreGraphics functions.
	cfDataCreate                   func(alloc uintptr, bytes *byte, length int) uintptr
	cfRelease                      func(ref uintptr)
	cgColorSpaceCreateWithName     func(name uintptr) uintptr
	cgDataProviderCreateWithCFData func(data uintptr) uintptr
	cgImageCreate                  func(w, h, bpc, bpp, stride int, space uintptr, info uint32, provider uintptr, decode uintptr, interpolate bool, intent int32) uintptr
	kCGColorSpaceSRGB              uintptr
)

// Load opens AppKit, CoreFoundation and CoreGraphics and resolves the C
// functions used. It is idempotent.
func Load() error {
	if loaded {
		return loadErr
	}
	loaded = true
	open := func(path string) uintptr {
		if loadErr != nil {
			return 0
		}
		h, err := purego.Dlopen(path, purego.RTLD_GLOBAL|purego.RTLD_NOW)
		if err != nil {
			loadErr = fmt.Errorf("darwin: dlopen %s: %w", path, err)
		}
		debugf("dlopen %s -> %#x", path, h)
		return h
	}
	open("/System/Library/Frameworks/AppKit.framework/AppKit")
	open("/System/Library/Frameworks/QuartzCore.framework/QuartzCore")
	cf := open("/System/Library/Frameworks/CoreFoundation.framework/CoreFoundation")
	cg := open("/System/Library/Frameworks/CoreGraphics.framework/CoreGraphics")
	if loadErr != nil {
		return loadErr
	}
	purego.RegisterLibFunc(&cfDataCreate, cf, "CFDataCreate")
	purego.RegisterLibFunc(&cfRelease, cf, "CFRelease")
	purego.RegisterLibFunc(&cgColorSpaceCreateWithName, cg, "CGColorSpaceCreateWithName")
	purego.RegisterLibFunc(&cgDataProviderCreateWithCFData, cg, "CGDataProviderCreateWithCFData")
	purego.RegisterLibFunc(&cgImageCreate, cg, "CGImageCreate")
	sym, err := purego.Dlsym(cg, "kCGColorSpaceSRGB")
	if err != nil {
		loadErr = fmt.Errorf("darwin: kCGColorSpaceSRGB: %w", err)
		return loadErr
	}
	// The symbol is a CFStringRef variable: read the pointer it holds.
	kCGColorSpaceSRGB = **(**uintptr)(unsafe.Pointer(&sym))
	debugf("kCGColorSpaceSRGB = %#x", kCGColorSpaceSRGB)
	return nil
}

// Selector and class helpers.

var selCache = map[string]objc.SEL{}

func sel(name string) objc.SEL {
	s, ok := selCache[name]
	if !ok {
		s = objc.RegisterName(name)
		selCache[name] = s
	}
	return s
}

func class(name string) objc.ID { return objc.ID(objc.GetClass(name)) }

// nsString makes an autoreleased NSString.
func nsString(s string) objc.ID {
	return class("NSString").Send(sel("stringWithUTF8String:"), s)
}

// goString copies an NSString into a Go string.
func goString(s objc.ID) string {
	if s == 0 {
		return ""
	}
	p := objc.Send[*byte](s, sel("UTF8String"))
	if p == nil {
		return ""
	}
	n := 0
	for *(*byte)(unsafe.Add(unsafe.Pointer(p), n)) != 0 {
		n++
	}
	return string(unsafe.Slice(p, n))
}

// OSVersion returns the macOS version string (for the stage 1 check).
func OSVersion() string {
	pi := class("NSProcessInfo").Send(sel("processInfo"))
	return goString(pi.Send(sel("operatingSystemVersionString")))
}
