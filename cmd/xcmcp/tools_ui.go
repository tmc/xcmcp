package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/tmc/xcmcp/internal/purego/coresim"
	"github.com/tmc/xcmcp/ui"
)

// UI Interaction
type UITargetInput struct {
	ID       string  `json:"id,omitempty"`
	Label    string  `json:"label,omitempty"`
	X        float64 `json:"x,omitempty" description:"X coordinate (for coordinate-based tap)"`
	Y        float64 `json:"y,omitempty" description:"Y coordinate (for coordinate-based tap)"`
	UDID     string  `json:"udid,omitempty" description:"Simulator UDID (optional)"`
	Platform string  `json:"platform,omitempty" description:"Target platform: 'mac' or 'ios' (default: auto)"`
}

type UISwipeInput struct {
	Direction string `json:"direction" description:"Swipe direction (left, right, up, down)"`
}

type UITypeInput struct {
	Text string `json:"text"`
}

func registerUITools(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "ui_tap",
		Description: "Tap an element or coordinate. Examples: ui_tap(id='login'), ui_tap(udid='booted', x=100, y=200)",
	}, SafeTool("ui_tap", func(ctx context.Context, req *mcp.CallToolRequest, args UITargetInput) (*mcp.CallToolResult, SimulatorActionOutput, error) {
		// Determine Platform
		isIOS := args.Platform == "ios" || args.UDID != ""
		if !isIOS && args.Platform != "mac" {
			// Auto-detect: if args.ID looks like a UUID or specific pattern?
			// For now default to Mac unless UDID is present.
			// However, if we are in a 'sim' context we might default to iOS.
		}

		if isIOS {
			udid := args.UDID
			if udid == "" {
				udid = "booted"
			}
			device, ok := coresim.FindDeviceByUDID(udid)
			if !ok {
				return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: "Device not found"}}}, SimulatorActionOutput{}, nil
			}

			// Coordinate Tap
			if args.X != 0 || args.Y != 0 {
				err := device.Tap(args.X, args.Y)
				if err != nil {
					return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("iOS Tap failed: %v", err)}}}, SimulatorActionOutput{}, nil
				}
				return &mcp.CallToolResult{}, SimulatorActionOutput{Message: fmt.Sprintf("Tapped at %.1f, %.1f on %s", args.X, args.Y, udid)}, nil
			}

			// ID Tap (Requires Element Lookup)
			// TODO: Implement Element Lookup via Traversing/Search
			// For now, fail if not coordinate.
			return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: "iOS Tap by ID not yet supported (requires deep tree traversal). Use coordinates (X, Y)."}}}, SimulatorActionOutput{}, nil
		}

		// Mac Implementation (Existing)
		// Currently generic tap or ID or Label stub
		if args.ID != "" {
			// ... (rest of mac logic)
			el := ui.ElementByID(args.ID)
			if el == nil && args.Label != "" {
				matches := ui.Application().Element().Query(ui.QueryParams{Label: args.Label})
				if len(matches) > 0 {
					el = matches[0]
				}
			}
			// ...
			if el == nil {
				// Fallback: Try finding by label if ID passed as ID (legacy behavior)
				matches := ui.Application().Element().Query(ui.QueryParams{Label: args.ID})
				if len(matches) > 0 {
					el = matches[0]
				}
			}

			if el == nil {
				// Fallback: Try finding by Title
				matches := ui.Application().Element().Query(ui.QueryParams{Title: args.ID})
				if len(matches) > 0 {
					el = matches[0]
				}
			}

			if el == nil {
				return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Element with ID/Label/Title '%s' not found", args.ID)}}}, SimulatorActionOutput{}, nil
			}
			el.Tap()
		} else {
			// Provide default fallback or error
			app := ui.Application()
			if !app.Exists() {
				return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: "Simulator app not found"}}}, SimulatorActionOutput{}, nil
			}
			el := app.Element()
			if el == nil {
				return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: "Simulator element not accessible"}}}, SimulatorActionOutput{}, nil
			}
			el.Tap() // Taps app center/frame
		}
		return &mcp.CallToolResult{}, SimulatorActionOutput{Message: "Tapped"}, nil
	}))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "ui_type",
		Description: "Type text into focused element. Example: ui_type(text='hello')",
	}, SafeTool("ui_type", func(ctx context.Context, req *mcp.CallToolRequest, args UITypeInput) (*mcp.CallToolResult, SimulatorActionOutput, error) {
		ui.FocusedElement().TypeText(args.Text)
		return &mcp.CallToolResult{}, SimulatorActionOutput{Message: "Typed"}, nil
	}))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "ui_swipe",
		Description: "Swipe on screen/app. Example: ui_swipe(direction='left')",
	}, SafeTool("ui_swipe", func(ctx context.Context, req *mcp.CallToolRequest, args UISwipeInput) (*mcp.CallToolResult, SimulatorActionOutput, error) {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{&mcp.TextContent{Text: "ui_swipe is not implemented"}},
		}, SimulatorActionOutput{}, nil
	}))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "ui_tree",
		Description: "Get UI hierarchy debug description. Example: ui_tree(bundle_id='com.apple.finder')",
	}, SafeTool("ui_tree", func(ctx context.Context, req *mcp.CallToolRequest, args struct {
		BundleID string `json:"bundle_id,omitempty"`
	}) (*mcp.CallToolResult, UITreeOutput, error) {
		bundleID := args.BundleID
		if bundleID == "" {
			bundleID = "com.apple.iphonesimulator"
		}
		app := ui.ApplicationWithBundleID(bundleID)
		return &mcp.CallToolResult{}, UITreeOutput{Tree: app.Tree()}, nil
	}))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "ui_inspect",
		Description: "Inspect element attributes. Examples: ui_inspect(id='my_button'), ui_inspect()",
	}, SafeTool("ui_inspect", func(ctx context.Context, req *mcp.CallToolRequest, args UITargetInput) (*mcp.CallToolResult, map[string]interface{}, error) {
		// Re-using UITargetInput as it has optional ID
		var el *ui.Element
		if args.ID != "" {
			el = ui.ElementByID(args.ID)
		} else {
			el = ui.Application().Element()
		}

		if el == nil {
			return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: "Element not found"}}}, nil, nil
		}

		attrs := el.Attributes()
		return &mcp.CallToolResult{}, map[string]interface{}{
			"label":      attrs.Label,
			"identifier": attrs.Identifier,
			"title":      attrs.Title,
			"value":      attrs.Value,
			"frame":      attrs.Frame,
			"enabled":    attrs.Enabled,
			"selected":   attrs.Selected,
			"has_focus":  attrs.HasFocus,
		}, nil
	}))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "ui_double_tap",
		Description: "Double tap an element. Example: ui_double_tap(id='like_button')",
	}, SafeTool("ui_double_tap", func(ctx context.Context, req *mcp.CallToolRequest, args UITargetInput) (*mcp.CallToolResult, SimulatorActionOutput, error) {
		var el *ui.Element
		if args.ID != "" {
			el = ui.ElementByID(args.ID)
		} else {
			el = ui.Application().Element()
		}

		if el == nil {
			return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: "Element not found"}}}, SimulatorActionOutput{}, nil
		}

		el.DoubleTap()
		return &mcp.CallToolResult{}, SimulatorActionOutput{Message: "Double tapped"}, nil
	}))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "ui_long_press",
		Description: "Long press an element. Example: ui_long_press(id='record', duration=2.0)",
	}, SafeTool("ui_long_press", func(ctx context.Context, req *mcp.CallToolRequest, args struct {
		ID       string  `json:"id,omitempty"`
		Duration float64 `json:"duration"`
	}) (*mcp.CallToolResult, SimulatorActionOutput, error) {
		var el *ui.Element
		if args.ID != "" {
			el = ui.ElementByID(args.ID)
		} else {
			el = ui.Application().Element()
		}

		if el == nil {
			return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: "Element not found"}}}, SimulatorActionOutput{}, nil
		}

		duration := args.Duration
		if duration == 0 {
			duration = 1.0
		}
		el.Press(duration)
		return &mcp.CallToolResult{}, SimulatorActionOutput{Message: "Long pressed"}, nil
	}))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "ui_wait",
		Description: "Wait for element to exist. Example: ui_wait(id='loaded_content', timeout=10.0)",
	}, SafeTool("ui_wait", func(ctx context.Context, req *mcp.CallToolRequest, args struct {
		ID      string  `json:"id,omitempty"`
		Timeout float64 `json:"timeout"`
	}) (*mcp.CallToolResult, SimulatorActionOutput, error) {
		var el *ui.Element
		if args.ID != "" {
			el = ui.ElementByID(args.ID)
		} else {
			el = ui.Application().Element()
		}

		if el == nil {
			return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: "Element context not found"}}}, SimulatorActionOutput{}, nil
		}

		timeout := args.Timeout
		if timeout == 0 {
			timeout = 5.0
		}
		exists := el.WaitForExistence(timeout)
		if !exists {
			return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: "Element did not appear within timeout"}}}, SimulatorActionOutput{}, nil
		}
		return &mcp.CallToolResult{}, SimulatorActionOutput{Message: "Element exists"}, nil
	}))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "ui_slider",
		Description: "Set slider value (0.0 - 1.0). Example: ui_slider(id='volume', value=0.5)",
	}, SafeTool("ui_slider", func(ctx context.Context, req *mcp.CallToolRequest, args struct {
		ID    string  `json:"id,omitempty"`
		Value float64 `json:"value"`
	}) (*mcp.CallToolResult, SimulatorActionOutput, error) {
		var el *ui.Element
		if args.ID != "" {
			el = ui.ElementByID(args.ID)
		} else {
			el = ui.Application().Element()
		}

		if el == nil {
			return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: "Element not found"}}}, SimulatorActionOutput{}, nil
		}

		el.AdjustToNormalizedSliderPosition(args.Value)
		return &mcp.CallToolResult{}, SimulatorActionOutput{Message: fmt.Sprintf("Set slider to %f", args.Value)}, nil
	}))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "ui_drag",
		Description: "Drag from one element to another. Example: ui_drag(from_id='a', to_id='b')",
	}, SafeTool("ui_drag", func(ctx context.Context, req *mcp.CallToolRequest, args struct {
		FromID   string  `json:"from_id,omitempty"`
		ToID     string  `json:"to_id,omitempty"`
		Duration float64 `json:"duration"`
	}) (*mcp.CallToolResult, SimulatorActionOutput, error) {
		fromEl := ui.ElementByID(args.FromID)
		toEl := ui.ElementByID(args.ToID)

		if fromEl == nil || toEl == nil {
			return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: "One or both elements not found"}}}, SimulatorActionOutput{}, nil
		}

		duration := args.Duration
		if duration == 0 {
			duration = 1.0
		}
		fromEl.DragTo(toEl, duration)
		return &mcp.CallToolResult{}, SimulatorActionOutput{Message: "Dragged"}, nil
	}))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "ui_list_windows",
		Description: "List windows of an application",
	}, SafeTool("ui_list_windows", func(ctx context.Context, req *mcp.CallToolRequest, args struct {
		BundleID string `json:"bundle_id"`
	}) (*mcp.CallToolResult, map[string]interface{}, error) {
		app := ui.ApplicationWithBundleID(args.BundleID)
		windows := app.Element().Windows()
		var windowList []string
		for _, w := range windows {
			windowList = append(windowList, w.Title())
		}
		return &mcp.CallToolResult{}, map[string]interface{}{"windows": windowList}, nil
	}))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "ui_list_buttons",
		Description: "List buttons in a window/element",
	}, SafeTool("ui_list_buttons", func(ctx context.Context, req *mcp.CallToolRequest, args struct {
		ID string `json:"id"`
	}) (*mcp.CallToolResult, map[string]interface{}, error) {
		el := ui.ElementByID(args.ID)
		if el == nil {
			return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: "Element not found"}}}, nil, nil
		}
		buttons := el.Buttons()
		var buttonList []string
		for _, b := range buttons {
			buttonList = append(buttonList, b.Title())
		}
		return &mcp.CallToolResult{}, map[string]interface{}{"buttons": buttonList}, nil
	}))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "ui_query",
		Description: "Query UI elements (generic)",
	}, SafeTool("ui_query", func(ctx context.Context, req *mcp.CallToolRequest, args struct {
		BundleID string `json:"bundle_id,omitempty"`
		Role     string `json:"role,omitempty"`
		Label    string `json:"label,omitempty"`
		Title    string `json:"title,omitempty"`
		ID       string `json:"id,omitempty"`
	}) (*mcp.CallToolResult, SimulatorActionOutput, error) {
		bundleID := args.BundleID
		if bundleID == "" {
			bundleID = "com.apple.iphonesimulator"
		}
		app := ui.ApplicationWithBundleID(bundleID)
		matches := app.Element().Query(ui.QueryParams{
			Role:       args.Role,
			Label:      args.Label,
			Title:      args.Title,
			Identifier: args.ID,
		})

		// Convert to basic info
		var resultMatches []map[string]interface{}
		for _, m := range matches {
			resultMatches = append(resultMatches, map[string]interface{}{
				"role":       m.Role(),
				"label":      m.Label(),
				"title":      m.Title(),
				"identifier": m.Identifier(),
				"frame":      m.Frame(),
			})
		}

		jsonBytes, _ := json.Marshal(map[string]interface{}{
			"count":   len(matches),
			"matches": resultMatches,
		})

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: string(jsonBytes)},
			},
		}, SimulatorActionOutput{}, nil
	}))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "ui_screenshot",
		Description: "Take a screenshot of an element. Example: ui_screenshot(id='login')",
	}, SafeTool("ui_screenshot", func(ctx context.Context, req *mcp.CallToolRequest, args struct {
		BundleID string `json:"bundle_id,omitempty"`
		ID       string `json:"id,omitempty"`
	}) (*mcp.CallToolResult, map[string]string, error) {
		bundleID := args.BundleID
		id := args.ID
		var el *ui.Element

		if bundleID != "" {
			app := ui.ApplicationWithBundleID(bundleID)
			if !app.Exists() {
				return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: "App not found or not running"}}}, map[string]string{}, nil
			}
			if id != "" {
				matches := app.Element().Query(ui.QueryParams{Identifier: id})
				if len(matches) > 0 {
					el = matches[0]
				}
			} else {
				el = app.Element()
			}
		} else if id != "" {
			el = ui.ElementByID(id)
		} else {
			el = ui.Application().Element()
		}

		if el == nil {
			return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: "Element not found"}}}, map[string]string{}, nil
		}

		// Use a persistent path for the user to view
		filename := fmt.Sprintf("/tmp/xcmcp_screenshot_%s.png", id)
		if id == "" {
			filename = "/tmp/xcmcp_screenshot_app.png"
		}

		// Create file to ensure permissions? screencapture creates it.
		// remove old
		os.Remove(filename)

		// We need to implement ScreenshotToFile in Element because Screenshot() reads it into memory.
		// Or just write 'data' to file.
		data, err := el.Screenshot()
		if err != nil {
			return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}}}, map[string]string{}, nil
		}

		if err := os.WriteFile(filename, data, 0644); err != nil {
			return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: "Failed to write screenshot file: " + err.Error()}}}, nil, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Screenshot saved to: %s", filename)},
				// Optional: Send a small preview or just the text
			},
		}, map[string]string{"message": "Screenshot taken", "path": filename}, nil
	}))
}
