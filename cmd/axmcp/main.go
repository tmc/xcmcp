// Command axmcp is an MCP server for macOS Accessibility API automation.
//
// It exposes the AX element tree, querying, and interaction tools over the
// Model Context Protocol, running as a macOS app bundle for Accessibility TCC.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"runtime"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/tmc/apple/appkit"
	"github.com/tmc/macgo"
	"github.com/tmc/xcmcp/internal/ui"
)

var verbose = flag.Bool("v", false, "enable verbose debug logging")

const (
	permissionWaitTimeout  = 10 * time.Second
	permissionPollInterval = 250 * time.Millisecond
)

func permissionPane(service string) string {
	switch service {
	case "Screen Recording":
		return "Screen Recording"
	default:
		return service
	}
}

func waitForPermission(service string, timeout, interval time.Duration, check func() bool) error {
	if check() {
		return nil
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		time.Sleep(interval)
		if check() {
			return nil
		}
	}
	return fmt.Errorf("%s permission not granted for axmcp.app; grant access in System Settings > Privacy & Security > %s", service, permissionPane(service))
}

func failPermission(err error) {
	fmt.Fprintf(os.Stderr, "axmcp: %v\n", err)
	os.Exit(1)
}

func main() {
	// Handle -h/--help before macgo.Start to avoid app bundle relaunch.
	for _, arg := range os.Args[1:] {
		if arg == "-h" || arg == "--help" || arg == "-help" {
			fmt.Fprintf(os.Stderr, "Usage: axmcp [flags]\n\naxmcp is an MCP server for macOS Accessibility API automation.\n\nFlags:\n")
			flag.PrintDefaults()
			os.Exit(0)
		}
	}

	runtime.LockOSThread()
	flag.Parse()

	cfg := macgo.NewConfig().
		WithAppName("axmcp").
		WithPermissions(macgo.Accessibility, macgo.ScreenRecording).
		WithUsageDescription("NSScreenCaptureUsageDescription", "axmcp needs to capture screenshots of specific UI elements and windows.").
		WithAdHocSign()
	if *verbose {
		cfg = cfg.WithDebug()
	}
	cfg.BundleID = "dev.tmc.axmcp"
	ui.ConfigureIdentity("axmcp", cfg.BundleID)

	if err := macgo.Start(cfg); err != nil {
		log.Fatalf("macgo start failed: %v", err)
	}

	logLevel := slog.LevelWarn
	if *verbose {
		logLevel = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel})))

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "axmcp",
		Version: "0.1.0",
	}, &mcp.ServerOptions{
		Capabilities: &mcp.ServerCapabilities{
			Tools: &mcp.ToolCapabilities{ListChanged: true},
		},
	})

	registerAXTools(server)

	// Initialize AppKit — required for NSWindow, buttons, and DispatchMainSafe.
	app := appkit.GetNSApplicationClass().SharedApplication()

	ui.CheckTrust()

	// Only check screen recording eagerly for the CLI screenshot subcommand.
	// MCP server mode defers the check until a screenshot tool is actually called.
	if len(os.Args) >= 2 && os.Args[1] == "screenshot" {
		ui.CheckScreenCapture()
	}

	if isTTY() || len(os.Args) > 1 {
		// Run CLI in goroutine so main thread can drive the AppKit run loop.
		go func() {
			time.Sleep(100 * time.Millisecond)
			if err := waitForPermission("Accessibility", permissionWaitTimeout, permissionPollInterval, ui.IsTrusted); err != nil {
				failPermission(err)
			}
			// Wait for Screen Recording if screenshotting.
			if len(os.Args) >= 2 && os.Args[1] == "screenshot" {
				if err := waitForPermission("Screen Recording", permissionWaitTimeout, permissionPollInterval, ui.IsScreenRecordingTrusted); err != nil {
					failPermission(err)
				}
			}
			runCLI()
			// runCLI calls os.Exit on completion, so this goroutine won't return
		}()
	} else {
		// Run MCP server in goroutine so main thread can drive the AppKit run loop.
		go func() {
			if err := waitForPermission("Accessibility", permissionWaitTimeout, permissionPollInterval, ui.IsTrusted); err != nil {
				failPermission(err)
			}
			if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
				log.Printf("server error: %v", err)
			}
			ui.WaitForWindows()
			os.Exit(0)
		}()
	}

	// Run the AppKit event loop on the main thread. This drains CFRunLoop,
	// the GCD main queue, and AppKit UI events (buttons, windows, etc.).
	app.Run()
	_ = 42
}
