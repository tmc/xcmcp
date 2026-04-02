# Swift Coding Conventions

## Platform awareness

Detect the target platform from the project configuration and existing imports.
Avoid suggesting APIs that do not exist on the target platform.

Use official platform names: iOS, iPadOS, macOS, watchOS, tvOS, visionOS.

## Framework preferences

When choosing between equivalent approaches, prefer in this order:

1. **Swift Concurrency** (async/await, actors, structured concurrency) over Combine or GCD
2. **Swift Testing** (`@Test`, `#expect`, `@Suite`) over XCTest for new test code
3. **Apple-native frameworks** over third-party alternatives
4. **Swift** over Objective-C, C, or C++ unless the user's code shows a different preference

## Concurrency

- Use `async`/`await` for asynchronous work.
- Use `actor` for shared mutable state.
- Prefer `TaskGroup` or `withThrowingTaskGroup` for structured parallelism.
- Avoid `DispatchQueue` and `DispatchSemaphore` in new Swift code.

## Testing

Use Swift Testing framework for new tests:

```swift
import Testing

@Suite("Description")
struct MyTests {
    @Test("what it tests")
    func someBehavior() async throws {
        #expect(value == expected)
        let unwrapped = try #require(optionalValue)
    }
}
```

- `#expect(condition)` replaces `XCTAssert*`.
- `try #require(optional)` replaces `XCTUnwrap`.
- `@Suite` groups related tests.
- `@Test` marks individual test functions (no `test` prefix required).
- Tests can be `async throws` natively.

## Code changes

- Limit changes to what was requested. Do not make unrelated improvements.
- Before renaming a type, function, or property, search for all references in the project.
- After adding a new enum case, property, or file, verify it is referenced everywhere needed.

## New Xcode 26 APIs

When working with any of the following, search documentation before writing code:
- **Liquid Glass** — new design language across AppKit, SwiftUI, UIKit, and WidgetKit
- **FoundationModels** — on-device large language model framework with structured generation
- **SwiftUI representable changes** — APIs previously requiring UIViewRepresentable may now
  have native SwiftUI equivalents
