# xcmcp

`xcmcp` is a macOS-focused toolkit for Xcode, Simulator, Accessibility, and AppleScript automation.

The repository has two main faces:

- `cmd/xcmcp` is a stdio MCP server for project inspection, build and test, simulator control, device control, UI inspection, and Xcode integration.
- `cmd/xc` is a direct CLI built on the same packages for local use from a terminal.

It also includes focused tools for macOS automation:

- `cmd/ax` and `cmd/axmcp` for the macOS Accessibility API.
- `cmd/ascript` and `cmd/ascriptmcp` for AppleScript dictionaries and scriptable applications.

## Requirements

- macOS with Xcode installed.
- Command Line Tools available through `xcrun`.
- Go 1.26 or newer to build from source.
- Accessibility permission for commands that drive the UI, including `xcmcp`, `xc`, `ax`, and `axmcp`.
- A booted simulator or connected device for simulator and device workflows.

## Install

Build the commands you need:

```sh
go install ./cmd/xcmcp ./cmd/xc ./cmd/ax ./cmd/axmcp ./cmd/ascript ./cmd/ascriptmcp
```

Or build everything in the module:

```sh
go build ./...
```

## Quick Start

Run the MCP server:

```sh
xcmcp
```

`xcmcp` starts with:

- core project, build, test, simulator, and Xcode target tools
- MCP resources
- Xcode bridge tools via `xcrun mcpbridge`

Optional toolsets can be enabled at startup:

```sh
xcmcp --enable-ui-tools --enable-device-tools --enable-ios-tools
```

Or all at once:

```sh
xcmcp --enable-all
```

An MCP client configuration usually looks like this:

```json
{
  "mcpServers": {
    "xcmcp": {
      "command": "/absolute/path/to/xcmcp",
      "args": [
        "--enable-ui-tools",
        "--enable-device-tools",
        "--enable-ios-tools"
      ]
    }
  }
}
```

Within a session, optional toolsets can also be enabled dynamically with `list_toolsets` and `enable_toolset`.

## Commands

### `xcmcp`

`xcmcp` serves tools and resources over stdio.

Always-on native tools include:

- `discover_projects`, `list_schemes`, `show_build_settings`
- `build`, `test`
- `list_simulators`, `boot_simulator`, `shutdown_simulator`
- `xcode_add_target`
- `list_toolsets`, `enable_toolset`

The Xcode bridge toolset is enabled by default and adds IDE-facing tools discovered from `xcrun mcpbridge`.

Optional toolsets include:

- `app`: app lifecycle, install, uninstall, logs, and app listing
- `ui`: UI tree, tap, inspect, query, screenshot, and wait
- `device`: simulator orientation, privacy, location, appearance, and screenshots
- `ios`: direct CoreSimulator-based accessibility tree and hit-testing
- `simulator_extras`: open URL, add media, and app container lookup
- `physical_device`: connected device inspection and app lifecycle actions
- `video`: simulator recording
- `crash`: crash report listing and reading
- `filesystem`: file access helpers
- `dependency`: Swift Package Manager helpers
- `asc`: App Store Connect and `altool` helpers

### `xc`

`xc` exposes the same building blocks as a direct CLI.

Examples:

```sh
xc sims list
xc build --scheme MyApp
xc test --scheme MyApp
xc app launch com.example.MyApp --udid booted
xc ui tree --bundle-id com.apple.finder
xc ios tree --udid booted
xc xcode add-target --template "Widget Extension" --product MyWidget
```

### `ax` and `axmcp`

These commands target the macOS Accessibility API directly.

Examples:

```sh
ax apps
ax tree com.apple.finder
ax find com.apple.dt.Xcode --role AXButton --title Build
```

`axmcp` exposes similar functionality as MCP tools.

### `ascript` and `ascriptmcp`

These commands inspect scriptable applications and run AppleScript-backed operations.

Examples:

```sh
ascript list /Applications/Xcode.app
ascript classes /Applications/Finder.app
ascript script /Applications/Finder.app activate
```

## Resources

`xcmcp` currently registers these MCP resources by default:

- `xcmcp://project`
- `xcmcp://simulators`
- `xcmcp://apps`
- `xcmcp://apps/{bundle_id}/tree`
- `xcmcp://apps/{bundle_id}/logs`

## Internal Layout

This module is command-first. The reusable helper packages live under `internal/`
and are not intended as a public import surface.

The main internal packages are:

- `internal/project`: discover Xcode projects and inspect schemes and build settings
- `internal/xcodebuild`: build and test wrappers
- `internal/simctl`: simulator management through `xcrun simctl`
- `internal/devicectl`: physical device management
- `internal/ui`: macOS Accessibility access and UI screenshots
- `internal/screen`: screen capture helpers
- `internal/crash`: crash report listing and reading
- `internal/resources`: MCP resource registration
- `internal/sdef`: parser for AppleScript scripting definitions
- `internal/altool` and `internal/asc`: App Store Connect helpers

## Notes

- This repository targets macOS. Many packages use AppKit, Accessibility, or Apple developer tools directly.
- UI automation depends on macOS Accessibility permission and on the target app being reachable through the Accessibility tree.
- Some simulator and Xcode automation features rely on private or implementation-defined behavior and are best treated as developer tooling rather than a stable public protocol.
- The supported entry points are the commands in `cmd/`. The internal packages may change without notice.
