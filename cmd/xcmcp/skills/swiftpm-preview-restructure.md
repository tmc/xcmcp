# Skill: Restructure SwiftPM Package for SwiftUI Preview Support

## Problem

SwiftUI Previews in executable targets require `ENABLE_DEBUG_DYLIB=YES`, which cannot
be set via `Package.swift`. The only SwiftPM-native solution is to move SwiftUI views
into a library target (libraries support previews by default).

## When to Use

- User reports "ENABLE_DEBUG_DYLIB" errors when trying to preview SwiftUI views
- User has a SwiftPM package with SwiftUI views in an executable target
- User wants working Xcode previews in a SwiftPM-only project (no .xcodeproj)

## Workflow

### 1. Identify the executable target containing SwiftUI views

Read `Package.swift` and find `.executableTarget` entries that contain SwiftUI view
files. Look for files with `struct ... : View`, `#Preview`, `@main struct ... : App`.

### 2. Separate the App entry point from the UI code

Split the executable target into:
- **A new library target** (e.g. `FooUI`) — contains all SwiftUI views, view models,
  and supporting UI types
- **The executable target** (e.g. `FooApp`) — becomes a thin `@main` entry point that
  imports the library

The `@main App` struct stays in the executable. Everything else moves to the library.

### 3. Create the library target directory and move files

```bash
mkdir -p Sources/<NewLibraryName>
```

Move all view files, view model files, and UI-supporting types. Keep only the file
containing `@main struct ... : App` in the executable target directory.

### 4. Update Package.swift

Add the new library target and wire up dependencies:

```swift
.library(name: "FooUI", targets: ["FooUI"]),
// ...
.target(
    name: "FooUI",
    dependencies: ["FooCore"],  // whatever the UI depends on
    path: "Sources/FooUI"
),
.executableTarget(
    name: "FooApp",
    dependencies: ["FooCore", "FooUI"],
    path: "Sources/FooApp"
),
```

### 5. Make types public

Since the UI code is now in a separate module, add `public` to:
- All types accessed from the executable target (the App struct references them)
- All initializers used cross-module (add explicit `public init(...)`)
- All properties and methods accessed from outside the module

### 6. Update the App entry point

Add `import FooUI` and ensure the `@main` struct references types from the new module.

### 7. Build and verify

```bash
swift build
```

### 8. Switch Xcode scheme and test previews

**Critical**: After restructuring, the active Xcode scheme must be set to the **library
target** (e.g. `FooUI`), not the executable target. Previews still fail if the
executable scheme is selected.

Use the Xcode menu: Product > Scheme > FooUI

Then open any SwiftUI view file in the library target — previews should work without
any build settings changes.

## Example

### Before

```
Sources/
├── MyAppCore/       # Library
├── MyApp/           # Executable (views + app entry point — previews broken)
│   ├── AppModel.swift
│   ├── ContentView.swift
│   └── MyApp.swift
```

### After

```
Sources/
├── MyAppCore/       # Library (unchanged)
├── MyAppUI/         # NEW library (views + view models — previews work)
│   ├── AppModel.swift
│   └── ContentView.swift
├── MyApp/           # Executable (thin entry point)
│   └── MyApp.swift
```

## Notes

- Library targets don't require `ENABLE_DEBUG_DYLIB` for previews
- This pattern is standard in large SwiftUI apps (separating UI into frameworks)
- The `#Preview` macro in library targets works out of the box
- No Xcode project file or build settings modifications are needed
- If using xcmcp tools: use `RenderPreview` after switching scheme to verify
