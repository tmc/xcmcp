// Package xcmcp documents the xcmcp module.
//
// xcmcp is a macOS-focused toolkit for Xcode, Simulator, Accessibility, and
// AppleScript automation. The module is organized around command packages and
// reusable libraries rather than a single top-level API.
//
// # Commands
//
// The main entry points are:
//
//   - cmd/xcmcp, a stdio MCP server for project inspection, build and test,
//     simulator control, device control, UI inspection, and Xcode integration
//   - cmd/xc, a direct CLI built on the same packages
//   - cmd/ax and cmd/axmcp, tools for the macOS Accessibility API
//   - cmd/ascript and cmd/ascriptmcp, tools for scriptable macOS applications
//
// # Core Packages
//
// The main reusable packages are:
//
//   - project, for discovering Xcode projects and schemes
//   - xcodebuild, for build and test execution
//   - simctl, for simulator management through xcrun simctl
//   - devicectl, for physical device management
//   - ui, for macOS Accessibility access and UI screenshots
//   - screen, for screen capture helpers
//   - crash, for crash report inspection
//   - resources, for MCP resource registration
//
// # Environment
//
// xcmcp targets macOS and assumes Xcode and the simulator tooling are
// installed. Packages that drive the UI require Accessibility permission.
// Simulator and device features also depend on the corresponding runtime state,
// such as a booted simulator or a connected device.
//
// This package exists to document the module as a whole. Most functionality
// lives in the command packages and subpackages listed above.
package xcmcp
