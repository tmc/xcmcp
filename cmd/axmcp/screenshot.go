package main

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unsafe"

	"github.com/tmc/apple/appkit"
	"github.com/tmc/apple/corefoundation"
	"github.com/tmc/apple/coregraphics"
	"github.com/tmc/apple/objc"
	"github.com/tmc/apple/screencapturekit"
	"github.com/tmc/xcmcp/internal/ui"
)

type windowInfo struct {
	WindowID  uint32  `json:"window_id"`
	Title     string  `json:"title"`
	OwnerName string  `json:"owner_name"`
	OwnerPID  int64   `json:"owner_pid"`
	X         float64 `json:"x"`
	Y         float64 `json:"y"`
	Width     float64 `json:"width"`
	Height    float64 `json:"height"`
}

// cfStringToGo extracts a Go string from a CFStringRef.
func cfStringToGo(ref corefoundation.CFStringRef) string {
	if ref == 0 {
		return ""
	}
	buf := make([]byte, 1024)
	if corefoundation.CFStringGetCString(ref, &buf[0], len(buf), 0x08000100) {
		for i, b := range buf {
			if b == 0 {
				return string(buf[:i])
			}
		}
	}
	return ""
}

func makeCFString(s string) corefoundation.CFStringRef {
	return corefoundation.CFStringCreateWithCString(0, s, 0x08000100)
}

// cfPointer converts a CF reference (stored as uintptr) to unsafe.Pointer
// for use with CoreFoundation dictionary APIs.
func cfPointer(ref uintptr) unsafe.Pointer {
	return *(*unsafe.Pointer)(unsafe.Pointer(&ref))
}

func dictGetString(dict corefoundation.CFDictionaryRef, key string) string {
	k := makeCFString(key)
	defer corefoundation.CFRelease(corefoundation.CFTypeRef(k))
	v := corefoundation.CFDictionaryGetValue(dict, cfPointer(uintptr(k)))
	if v == nil {
		return ""
	}
	return cfStringToGo(corefoundation.CFStringRef(uintptr(v)))
}

func dictGetNumber(dict corefoundation.CFDictionaryRef, key string) (int64, bool) {
	k := makeCFString(key)
	defer corefoundation.CFRelease(corefoundation.CFTypeRef(k))
	v := corefoundation.CFDictionaryGetValue(dict, cfPointer(uintptr(k)))
	if v == nil {
		return 0, false
	}
	ref := corefoundation.CFNumberRef(uintptr(v))
	var n int64
	if corefoundation.CFNumberGetValue(ref, corefoundation.KCFNumberSInt64Type, unsafe.Pointer(&n)) {
		return n, true
	}
	var n32 int32
	if corefoundation.CFNumberGetValue(ref, corefoundation.KCFNumberSInt32Type, unsafe.Pointer(&n32)) {
		return int64(n32), true
	}
	return 0, false
}

// listAppWindows returns on-screen windows matching the given app identifier.
func listAppWindows(appIdentifier string) ([]windowInfo, error) {
	windowList := coregraphics.CGWindowListCopyWindowInfo(
		coregraphics.KCGWindowListOptionOnScreenOnly,
		0,
	)
	if windowList == 0 {
		return nil, fmt.Errorf("CGWindowListCopyWindowInfo returned nil")
	}
	defer corefoundation.CFRelease(corefoundation.CFTypeRef(windowList))

	count := corefoundation.CFArrayGetCount(windowList)
	var windows []windowInfo
	for i := range count {
		dictPtr := corefoundation.CFArrayGetValueAtIndex(windowList, i)
		dict := corefoundation.CFDictionaryRef(uintptr(dictPtr))

		ownerName := dictGetString(dict, coregraphics.KCGWindowOwnerName)
		windowName := dictGetString(dict, coregraphics.KCGWindowName)
		ownerPID, _ := dictGetNumber(dict, coregraphics.KCGWindowOwnerPID)
		windowID, _ := dictGetNumber(dict, coregraphics.KCGWindowNumber)

		if !strings.EqualFold(ownerName, appIdentifier) &&
			!strings.Contains(strings.ToLower(ownerName), strings.ToLower(appIdentifier)) {
			continue
		}

		windows = append(windows, windowInfo{
			WindowID:  uint32(windowID),
			Title:     windowName,
			OwnerName: ownerName,
			OwnerPID:  ownerPID,
		})
	}
	if len(windows) == 0 {
		return nil, fmt.Errorf("no windows found for %q", appIdentifier)
	}
	return windows, nil
}

// captureWindowSCK captures a window screenshot using ScreenCaptureKit.
func captureWindowSCK(ctx context.Context, windowID uint32) ([]byte, error) {
	diagf("captureWindowSCK: start windowID=%d\n", windowID)

	diagf("captureWindowSCK: calling GetShareableContent\n")
	t0 := time.Now()
	content, err := screencapturekit.GetSCShareableContentClass().GetShareableContent(ctx)
	diagf("captureWindowSCK: GetShareableContent returned elapsed=%v err=%v\n", time.Since(t0), err)
	if err != nil {
		return nil, fmt.Errorf("get shareable content: %w", err)
	}

	// Retain the content object to prevent premature deallocation
	// by the ObjC autorelease pool.
	objc.Send[objc.ID](content.ID, objc.Sel("retain"))
	defer objc.Send[objc.ID](content.ID, objc.Sel("release"))

	diagf("captureWindowSCK: getting windows list\n")
	windows := content.Windows()
	diagf("captureWindowSCK: got %d windows\n", len(windows))
	var target screencapturekit.SCWindow
	for i, w := range windows {
		wid := w.WindowID()
		diagf("captureWindowSCK: window[%d] id=%d\n", i, wid)
		if wid == windowID {
			target = w
			break
		}
	}
	if target.GetID() == 0 {
		return nil, fmt.Errorf("window %d not found in shareable content", windowID)
	}
	diagf("captureWindowSCK: found target window\n")

	filter := screencapturekit.NewContentFilterWithDesktopIndependentWindow(&target)

	config := screencapturekit.NewSCStreamConfiguration()
	frame := target.Frame()
	config.SetWidth(uintptr(frame.Size.Width * 2))   // retina 2x
	config.SetHeight(uintptr(frame.Size.Height * 2)) // retina 2x

	diagf("captureWindowSCK: calling CaptureImage %.0fx%.0f\n", frame.Size.Width*2, frame.Size.Height*2)
	t1 := time.Now()
	img, err := screencapturekit.GetSCScreenshotManagerClass().CaptureImageWithFilterConfiguration(ctx, &filter, &config)
	diagf("captureWindowSCK: CaptureImage returned elapsed=%v err=%v img=%v\n", time.Since(t1), err, img)
	if err != nil {
		return nil, fmt.Errorf("capture image: %w", err)
	}
	defer coregraphics.CGImageRelease(img)

	diagf("captureWindowSCK: converting to PNG\n")
	return cgImageToPNG(img)
}

// captureFullScreen captures the entire main display using ScreenCaptureKit.
// Requires explicit opt-in because the resulting image is large.
func captureFullScreen() ([]byte, error) {
	if !ui.IsScreenRecordingTrusted() {
		if !ui.WaitForScreenRecording(30 * time.Second) {
			return nil, fmt.Errorf("screenshot failed: Screen Recording permission required — grant access in System Settings > Privacy & Security")
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	content, err := screencapturekit.GetSCShareableContentClass().GetShareableContent(ctx)
	if err != nil {
		return nil, fmt.Errorf("get shareable content: %w", err)
	}

	displays := content.Displays()
	if len(displays) == 0 {
		return nil, fmt.Errorf("no displays found")
	}

	display := displays[0]
	filter := screencapturekit.NewContentFilterWithDisplayExcludingWindows(&display, nil)
	config := screencapturekit.NewSCStreamConfiguration()
	config.SetWidth(uintptr(display.Width() * 2))
	config.SetHeight(uintptr(display.Height() * 2))

	img, err := screencapturekit.GetSCScreenshotManagerClass().CaptureImageWithFilterConfiguration(ctx, &filter, &config)
	if err != nil {
		return nil, fmt.Errorf("capture display: %w", err)
	}
	defer coregraphics.CGImageRelease(img)

	return cgImageToPNG(img)
}

// cgImageToPNG converts a CGImageRef to PNG-encoded bytes.
func cgImageToPNG(img coregraphics.CGImageRef) ([]byte, error) {
	if img == 0 {
		return nil, fmt.Errorf("nil CGImage")
	}
	rep := appkit.NewBitmapImageRepWithCGImage(img)
	if rep.GetID() == 0 {
		return nil, fmt.Errorf("failed to create NSBitmapImageRep")
	}
	data := rep.RepresentationUsingTypeProperties(appkit.NSBitmapImageFileTypePNG, nil)
	if data == nil {
		return nil, fmt.Errorf("failed to create PNG representation")
	}
	length := data.Length()
	if length == 0 {
		return nil, fmt.Errorf("empty PNG data")
	}
	raw := unsafe.Slice((*byte)(data.Bytes()), length)
	result := make([]byte, length)
	copy(result, raw)
	return result, nil
}
