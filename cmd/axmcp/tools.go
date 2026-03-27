package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/tmc/apple/x/axuiautomation"
	"github.com/tmc/xcmcp/internal/ui"
)

func registerAXTools(s *mcp.Server) {
	registerAXApps(s)
	registerAXTree(s)
	registerAXFind(s)
	registerAXPipe(s)
	registerAXClick(s)
	registerAXType(s)
	registerAXMenu(s)
	registerAXFocus(s)
	registerAXListWindows(s)
	registerAXScreenshot(s)
}

// openApp opens an application by bundle ID or numeric PID string.
func openApp(arg string) (*axuiautomation.Application, error) {
	if pid, err := strconv.ParseInt(arg, 10, 32); err == nil {
		app := axuiautomation.NewApplicationFromPID(int32(pid))
		if app == nil {
			return nil, fmt.Errorf("cannot connect to PID %d", pid)
		}
		return app, nil
	}
	return axuiautomation.NewApplication(arg)
}

// spinAndOpen opens an app and spins the run loop to prime AX IPC.
func spinAndOpen(arg string) (*axuiautomation.Application, error) {
	app, err := openApp(arg)
	if err != nil {
		return nil, err
	}
	axuiautomation.SpinRunLoop(200 * time.Millisecond)
	return app, nil
}

func elementAttrs(e *axuiautomation.Element) map[string]any {
	x, y := e.Position()
	w, h := e.Size()
	return map[string]any{
		"role": e.Role(), "title": e.Title(), "value": e.Value(),
		"subrole": e.Subrole(), "enabled": e.IsEnabled(),
		"role_desc": e.RoleDescription(),
		"desc":      e.Description(), "identifier": e.Identifier(),
		"x": x, "y": y, "w": w, "h": h,
	}
}

func elementSummary(e *axuiautomation.Element) string {
	return formatSnapshot(snapshotElement(e, 0, 0))
}

func treeText(e *axuiautomation.Element, indent, maxDepth int) string {
	if e == nil || indent > maxDepth {
		return ""
	}
	var b strings.Builder
	b.WriteString(strings.Repeat("  ", indent))
	b.WriteString(elementSummary(e))
	b.WriteString("\n")
	for _, c := range e.Children() {
		b.WriteString(treeText(c, indent+1, maxDepth))
	}
	return b.String()
}

func textResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
	}
}

// ── ax_apps ──────────────────────────────────────────────────────────────────

type axAppsInput struct{}

func registerAXApps(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "ax_apps",
		Description: "List running macOS applications with their bundle IDs and PIDs",
	}, func(_ context.Context, _ *mcp.CallToolRequest, _ axAppsInput) (*mcp.CallToolResult, any, error) {
		out, err := exec.Command("lsappinfo", "list").Output()
		if err != nil {
			return nil, nil, fmt.Errorf("lsappinfo: %w", err)
		}
		type appInfo struct {
			Name     string
			BundleID string
			PID      int
		}
		var apps []appInfo
		var cur appInfo
		for _, line := range strings.Split(string(out), "\n") {
			line = strings.TrimSpace(line)
			if strings.Contains(line, ") \"") && strings.Contains(line, "ASN:") {
				if cur.BundleID != "" {
					apps = append(apps, cur)
				}
				cur = appInfo{}
				s := line[strings.Index(line, "\"")+1:]
				cur.Name = s[:strings.Index(s, "\"")]
			} else if strings.HasPrefix(line, "bundleID=") {
				id := strings.Trim(strings.TrimPrefix(line, "bundleID="), `"`)
				if id != "[ NULL ]" {
					cur.BundleID = id
				}
			} else if strings.HasPrefix(line, "pid = ") {
				rest := strings.TrimPrefix(line, "pid = ")
				if i := strings.IndexAny(rest, " \t"); i > 0 {
					rest = rest[:i]
				}
				cur.PID, _ = strconv.Atoi(rest)
			}
		}
		if cur.BundleID != "" {
			apps = append(apps, cur)
		}
		var buf bytes.Buffer
		for _, a := range apps {
			fmt.Fprintf(&buf, "%-45s  pid=%-6d  %s\n", a.BundleID, a.PID, a.Name)
		}
		return textResult(buf.String()), nil, nil
	})
}

// ── ax_tree ───────────────────────────────────────────────────────────────────

type axTreeInput struct {
	App   string `json:"app"`
	Depth int    `json:"depth,omitempty"`
}

func registerAXTree(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "ax_tree",
		Description: "Print the AX element tree for a running application. Returns role/title hierarchy.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, args axTreeInput) (*mcp.CallToolResult, any, error) {
		app, err := spinAndOpen(args.App)
		if err != nil {
			return nil, nil, err
		}
		defer app.Close()
		depth := args.Depth
		if depth <= 0 {
			depth = 4
		}
		return textResult(treeText(app.Root(), 0, depth)), nil, nil
	})
}

// ── ax_find ───────────────────────────────────────────────────────────────────

type axFindInput struct {
	App      string `json:"app"`
	Role     string `json:"role,omitempty"`
	Title    string `json:"title,omitempty"`
	Contains string `json:"contains,omitempty"`
	Limit    int    `json:"limit,omitempty"`
}

func registerAXFind(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "ax_find",
		Description: "Find AX elements in an app by role, exact text, or substring across title, description, value, and identifier.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, args axFindInput) (*mcp.CallToolResult, any, error) {
		app, err := spinAndOpen(args.App)
		if err != nil {
			return nil, nil, err
		}
		defer app.Close()

		limit := args.Limit
		if limit <= 0 {
			limit = 50
		}
		result := findElements(app.Root(), searchOptions{
			Role:     args.Role,
			Title:    args.Title,
			Contains: args.Contains,
			Limit:    limit,
		})
		var buf bytes.Buffer
		if len(result.matches) == 0 {
			buf.WriteString(noMatchMessage(result))
			return textResult(buf.String()), nil, nil
		}
		if note := selectionReason(result); note != "" {
			buf.WriteString(note)
			buf.WriteByte('\n')
		}
		for i, match := range result.matches {
			fmt.Fprintf(&buf, "[%d] %s\n", i, formatMatch(match))
		}
		return textResult(buf.String()), nil, nil
	})
}

// ── ax_pipe ───────────────────────────────────────────────────────────────────

type axPipeInput struct {
	// Pipeline is a //-separated sequence of stages, e.g.:
	//   "app com.apple.dt.Xcode // windows"
	//   "app com.apple.finder // focus // attr AXTitle"
	Pipeline string `json:"pipeline"`
}

func registerAXPipe(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name: "ax_pipe",
		Description: `Execute an AX pipeline and return its text output.

Stages separated by // (double-slash):
  app <bundle-id|pid>              open application
  window [substr]                  focus first matching window
  windows                          list all windows
  focus                            get focused element
  children                         get children
  first                            take first element from list
  find [--role R] [--title T]      search descendants using normalized text matching
  .                                print current context (default)
  tree [--depth N]                 element tree
  list                             element list
  json                             JSON output
  click                            click element
  type <text>                      type text
  attr <AXAttr>                    print attribute
  click-menu <A->B->C>             click menu path

Examples:
  app com.apple.dt.Xcode // windows
  app com.apple.dt.Xcode // window // find --role AXButton // list
  app com.apple.finder // focus // attr AXTitle`,
	}, func(_ context.Context, _ *mcp.CallToolRequest, args axPipeInput) (*mcp.CallToolResult, any, error) {
		out, err := execPipeline(args.Pipeline)
		if err != nil {
			return nil, nil, err
		}
		return textResult(out), nil, nil
	})
}

// ── ax_click ──────────────────────────────────────────────────────────────────

type axClickInput struct {
	App      string `json:"app"`
	Contains string `json:"contains"`
	Role     string `json:"role,omitempty"`
	XOffset  *int   `json:"x_offset,omitempty"`
	YOffset  *int   `json:"y_offset,omitempty"`
}

func registerAXClick(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "ax_click",
		Description: "Click an element in an app found by normalized text lookup across title, description, value, and identifier. Provide x_offset and y_offset to click at a specific point relative to the element's top-left corner.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, args axClickInput) (*mcp.CallToolResult, any, error) {
		app, err := spinAndOpen(args.App)
		if err != nil {
			return nil, nil, err
		}
		defer app.Close()

		result := findElements(app.Root(), searchOptions{
			Role:     args.Role,
			Contains: args.Contains,
			Limit:    100,
		})
		if len(result.matches) == 0 {
			return nil, nil, fmt.Errorf("%s", noMatchMessage(result))
		}

		match := result.matches[0]
		resolution := resolveClickTarget(match, 50)
		target := resolution.target.element
		if target == nil {
			return nil, nil, fmt.Errorf("click target disappeared: %s", formatMatch(match))
		}

		if args.XOffset != nil && args.YOffset != nil {
			if err := target.ClickAt(*args.XOffset, *args.YOffset); err != nil {
				return nil, nil, fmt.Errorf("click_at %s: %w", formatSnapshot(resolution.target), err)
			}
			var buf bytes.Buffer
			fmt.Fprintf(&buf, "clicked %s at offset %d,%d", formatSnapshot(resolution.target), *args.XOffset, *args.YOffset)
			if note := selectionReason(result); note != "" {
				fmt.Fprintf(&buf, "\n%s", note)
			}
			if resolution.reason != "" {
				fmt.Fprintf(&buf, "\n%s", resolution.reason)
			}
			return textResult(buf.String()), nil, nil
		}

		if err := target.Click(); err != nil {
			if !match.snapshot.record.actionable && len(resolution.actionableDescendants) > 1 {
				var b strings.Builder
				fmt.Fprintf(&b, "click %s: %v\n", formatMatch(match), err)
				b.WriteString("actionable descendants:\n")
				for _, descendant := range resolution.actionableDescendants[:min(len(resolution.actionableDescendants), 5)] {
					fmt.Fprintf(&b, "  - %s\n", formatSnapshot(descendant))
				}
				return nil, nil, fmt.Errorf("%s", strings.TrimRight(b.String(), "\n"))
			}
			return nil, nil, fmt.Errorf("click %s: %w", formatSnapshot(resolution.target), err)
		}
		var buf bytes.Buffer
		fmt.Fprintf(&buf, "clicked %s", formatSnapshot(resolution.target))
		if note := selectionReason(result); note != "" {
			fmt.Fprintf(&buf, "\n%s", note)
		}
		if resolution.reason != "" {
			fmt.Fprintf(&buf, "\n%s", resolution.reason)
		}
		return textResult(buf.String()), nil, nil
	})
}

// ── ax_type ───────────────────────────────────────────────────────────────────

type axTypeInput struct {
	App      string `json:"app"`
	Contains string `json:"contains"`
	Text     string `json:"text"`
}

func registerAXType(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "ax_type",
		Description: "Type text into an element found by normalized text lookup across title, description, value, and identifier.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, args axTypeInput) (*mcp.CallToolResult, any, error) {
		app, err := spinAndOpen(args.App)
		if err != nil {
			return nil, nil, err
		}
		defer app.Close()

		result := findElements(app.Root(), searchOptions{
			Contains: args.Contains,
			Limit:    100,
		})
		if len(result.matches) == 0 {
			return nil, nil, fmt.Errorf("%s", noMatchMessage(result))
		}
		el := result.matches[0].snapshot.element
		if el == nil {
			return nil, nil, fmt.Errorf("type target disappeared: %s", formatMatch(result.matches[0]))
		}
		if err := el.Click(); err != nil {
			return nil, nil, fmt.Errorf("focus %s: %w", formatMatch(result.matches[0]), err)
		}
		if err := el.TypeText(args.Text); err != nil {
			return nil, nil, fmt.Errorf("type %s: %w", formatMatch(result.matches[0]), err)
		}
		var buf bytes.Buffer
		fmt.Fprintf(&buf, "typed into %s", formatMatch(result.matches[0]))
		if note := selectionReason(result); note != "" {
			fmt.Fprintf(&buf, "\n%s", note)
		}
		return textResult(buf.String()), nil, nil
	})
}

// ── ax_menu ───────────────────────────────────────────────────────────────────

type axMenuInput struct {
	App  string   `json:"app"`
	Path []string `json:"path"`
}

func registerAXMenu(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "ax_menu",
		Description: `Click a menu item by path array, e.g. ["File", "New", "Target..."]`,
	}, func(_ context.Context, _ *mcp.CallToolRequest, args axMenuInput) (*mcp.CallToolResult, any, error) {
		app, err := spinAndOpen(args.App)
		if err != nil {
			return nil, nil, err
		}
		defer app.Close()

		if err := app.ClickMenuItem(args.Path); err != nil {
			return nil, nil, fmt.Errorf("menu: %w", err)
		}
		return textResult("clicked menu: " + strings.Join(args.Path, " > ")), nil, nil
	})
}

// ── ax_list_windows ───────────────────────────────────────────────────────────

type axListWindowsInput struct {
	App string `json:"app"`
}

func registerAXListWindows(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "ax_list_windows",
		Description: "List windows for an application by name or bundle ID. Returns window IDs, titles, and bounds.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, args axListWindowsInput) (*mcp.CallToolResult, any, error) {
		windows, err := listAppWindows(args.App)
		if err != nil {
			return nil, nil, err
		}
		data, err := json.Marshal(windows)
		if err != nil {
			return nil, nil, fmt.Errorf("marshal: %w", err)
		}
		return textResult(string(data)), nil, nil
	})
}

// ── ax_screenshot ─────────────────────────────────────────────────────────────

type axScreenshotInput struct {
	App      string `json:"app"`
	Contains string `json:"contains,omitempty"`
	Role     string `json:"role,omitempty"`
}

func registerAXScreenshot(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "ax_screenshot",
		Description: "Capture a screenshot of an app window or specific element. Omit contains/role to capture the first window.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, args axScreenshotInput) (*mcp.CallToolResult, any, error) {
		app, err := spinAndOpen(args.App)
		if err != nil {
			return nil, nil, err
		}
		defer app.Close()

		var el *axuiautomation.Element
		if args.Contains != "" || args.Role != "" {
			result := findElements(app.Root(), searchOptions{
				Role:     args.Role,
				Contains: args.Contains,
				Limit:    100,
			})
			if len(result.matches) == 0 {
				return nil, nil, fmt.Errorf("%s", noMatchMessage(result))
			}
			el = result.matches[0].snapshot.element
			// Capture specific element
			png, err := captureElementOrWindow(args.App, true, el)
			if err != nil {
				return nil, nil, err
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.ImageContent{Data: png, MIMEType: "image/png"}},
			}, nil, nil
		} else {
			wins := app.WindowList()
			if len(wins) == 0 {
				return nil, nil, fmt.Errorf("no windows found to screenshot")
			}
			el = wins[0]
			// Capture the whole window using screen-capture and list-app-windows
			png, err := captureElementOrWindow(args.App, false, el)
			if err != nil {
				return nil, nil, err
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.ImageContent{Data: png, MIMEType: "image/png"}},
			}, nil, nil
		}
	})
}

// captureElementOrWindow abstracts the logic to capture a screenshot.
// If isElement is true, it attempts an element screenshot.
// Otherwise it tries ScreenCaptureKit for a robust window capture.
func captureElementOrWindow(appName string, isElement bool, el *axuiautomation.Element) ([]byte, error) {
	if !ui.IsScreenRecordingTrusted() {
		go ui.CheckScreenCapture()
		return nil, fmt.Errorf("screenshot failed: Screen Recording permission required — grant access in System Settings > Privacy & Security")
	}

	if !isElement {
		// Try native SCK capture for full windows.
		windows, err := listAppWindows(appName)
		if err == nil && len(windows) > 0 {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if png, err := captureWindowSCK(ctx, windows[0].WindowID); err == nil {
				return png, nil
			}
		}
	}

	// Fallback to accessibility element screenshot.
	png, err := el.Screenshot()
	if err != nil {
		return nil, fmt.Errorf("screenshot: %w", err)
	}

	return png, nil
}

// ── ax_focus ──────────────────────────────────────────────────────────────────

type axFocusInput struct {
	App string `json:"app"`
}

func registerAXFocus(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "ax_focus",
		Description: "Get the currently focused AX element in an application",
	}, func(_ context.Context, _ *mcp.CallToolRequest, args axFocusInput) (*mcp.CallToolResult, any, error) {
		app, err := spinAndOpen(args.App)
		if err != nil {
			return nil, nil, err
		}
		defer app.Close()

		el := app.FocusedElement()
		if el == nil {
			// Fallback: perhaps the app has no focused element, but has a main window
			win := app.MainWindow()
			if win != nil {
				return textResult(fmt.Sprintf("no focused element; main window is: %s", elementSummary(win))), nil, nil
			}
			return nil, nil, fmt.Errorf("no focused element and no main window found (app might be in background or has no standard UI)")
		}
		return textResult(elementSummary(el)), nil, nil
	})
}
