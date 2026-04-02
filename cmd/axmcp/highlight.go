package main

import (
	"fmt"
	"time"

	"github.com/tmc/apple/appkit"
	"github.com/tmc/apple/corefoundation"
	"github.com/tmc/apple/dispatch"
	"github.com/tmc/apple/foundation"
)

const (
	highlightDuration     = 2 * time.Second
	highlightFadeDuration = 180 * time.Millisecond
	highlightFadeSteps    = 6
	highlightBoxPadding   = 4
	highlightMaxMatches   = 8
)

func highlightCollectionBehavior() appkit.NSWindowCollectionBehavior {
	return appkit.NSWindowCollectionBehaviorCanJoinAllSpaces |
		appkit.NSWindowCollectionBehaviorTransient |
		appkit.NSWindowCollectionBehaviorIgnoresCycle |
		appkit.NSWindowCollectionBehaviorFullScreenAuxiliary
}

func runOnMain(work func()) {
	if foundation.GetThreadClass().CurrentThread().IsMainThread() {
		work()
		return
	}
	done := make(chan struct{})
	dispatch.MainQueue().Async(func() {
		defer close(done)
		work()
	})
	<-done
}

func uniqueOCRMatches(matches []ocrResult, limit int) []ocrResult {
	if limit <= 0 {
		limit = len(matches)
	}
	out := make([]ocrResult, 0, min(len(matches), limit))
	seen := make(map[[4]int]bool, len(matches))
	for _, match := range matches {
		if match.W <= 0 || match.H <= 0 {
			continue
		}
		key := [4]int{match.X, match.Y, match.W, match.H}
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, match)
		if len(out) >= limit {
			break
		}
	}
	return out
}

func highlightRectForMatch(match ocrResult, imgW, imgH int) corefoundation.CGRect {
	x0 := max(0, match.X-highlightBoxPadding)
	y0 := max(0, match.Y-highlightBoxPadding)
	x1 := min(imgW, match.X+match.W+highlightBoxPadding)
	y1 := min(imgH, match.Y+match.H+highlightBoxPadding)
	if x1 < x0 {
		x1 = x0
	}
	if y1 < y0 {
		y1 = y0
	}
	return corefoundation.CGRect{
		Origin: corefoundation.CGPoint{
			X: float64(x0),
			Y: float64(imgH - y1),
		},
		Size: corefoundation.CGSize{
			Width:  float64(x1 - x0),
			Height: float64(y1 - y0),
		},
	}
}

func animateOverlayWindow(win appkit.NSWindow, duration time.Duration) {
	if duration <= 0 {
		duration = highlightDuration
	}
	fade := highlightFadeDuration
	if duration < 2*fade {
		fade = duration / 2
	}
	if fade <= 0 {
		runOnMain(func() {
			win.SetAlphaValue(1)
		})
		return
	}
	stepSleep := fade / highlightFadeSteps
	if stepSleep <= 0 {
		stepSleep = time.Millisecond
	}
	for i := 1; i <= highlightFadeSteps; i++ {
		alpha := float64(i) / float64(highlightFadeSteps)
		runOnMain(func() {
			win.SetAlphaValue(alpha)
		})
		time.Sleep(stepSleep)
	}
	hold := duration - (2 * fade)
	if hold > 0 {
		time.Sleep(hold)
	}
	for i := highlightFadeSteps - 1; i >= 0; i-- {
		alpha := float64(i) / float64(highlightFadeSteps)
		runOnMain(func() {
			win.SetAlphaValue(alpha)
		})
		time.Sleep(stepSleep)
	}
}

func highlightOCRMatches(capture *ocrCapture, matches []ocrResult, duration time.Duration) (int, error) {
	if capture == nil || capture.target == nil {
		return 0, fmt.Errorf("highlight target disappeared")
	}
	if capture.imgW <= 0 || capture.imgH <= 0 {
		return 0, fmt.Errorf("highlight target has invalid size")
	}
	matches = uniqueOCRMatches(matches, highlightMaxMatches)
	if len(matches) == 0 {
		return 0, fmt.Errorf("no visible OCR boxes to highlight")
	}

	targetFrame := capture.target.Frame()
	frame := corefoundation.CGRect{
		Origin: corefoundation.CGPoint{
			X: targetFrame.Origin.X,
			Y: targetFrame.Origin.Y,
		},
		Size: corefoundation.CGSize{
			Width:  float64(capture.imgW),
			Height: float64(capture.imgH),
		},
	}

	var win appkit.NSWindow
	runOnMain(func() {
		win = appkit.NewWindowWithContentRectStyleMaskBackingDefer(
			frame,
			appkit.NSWindowStyleMaskBorderless,
			appkit.NSBackingStoreBuffered,
			false,
		)
		win.SetOpaque(false)
		win.SetBackgroundColor(appkit.NewColorWithSRGBRedGreenBlueAlpha(0, 0, 0, 0))
		win.SetHasShadow(false)
		win.SetIgnoresMouseEvents(true)
		win.SetReleasedWhenClosed(false)
		win.SetLevel(appkit.StatusWindowLevel)
		win.SetCollectionBehavior(highlightCollectionBehavior())

		content := appkit.NSViewFromID(win.ContentView().GetID())
		border := appkit.NewColorWithSRGBRedGreenBlueAlpha(1.0, 0.44, 0.08, 0.98)
		fill := appkit.NewColorWithSRGBRedGreenBlueAlpha(1.0, 0.44, 0.08, 0.18)
		for _, match := range matches {
			box := appkit.NewBoxWithFrame(highlightRectForMatch(match, capture.imgW, capture.imgH))
			box.SetBoxType(appkit.NSBoxCustom)
			box.SetTitlePosition(appkit.NSNoTitle)
			box.SetBorderColor(border)
			box.SetBorderWidth(3)
			box.SetCornerRadius(8)
			box.SetFillColor(fill)
			content.AddSubview(box)
		}

		win.SetAlphaValue(0)
		win.OrderFrontRegardless()
	})

	animateOverlayWindow(win, duration)
	runOnMain(func() {
		win.OrderOut(nil)
		win.Close()
	})
	return len(matches), nil
}
