# SwiftUI Preview Generation Rules

## Macro syntax

Always use `#Preview { }` macro. Never use the deprecated `PreviewProvider` protocol.

```swift
#Preview("Label") {
    MyView()
}
```

Only add `@available` if using `@Previewable`.

## When to wrap in NavigationStack

Embed in NavigationStack **only** if the view uses:
- `.navigation*` modifiers (`.navigationTitle`, `.navigationBarTitleDisplayMode`, etc.)
- `NavigationLink`
- `.toolbar*` modifiers
- `.customizationBehavior` / `.defaultCustomization`

Otherwise, preview the view directly without a NavigationStack.

## When to wrap in List

Embed in List **only** if the view uses list-specific modifiers or has a "Row" suffix:
- `.listItemTint`, `.listItemPlatterColor`, `.listRowBackground`, `.listRowInsets`
- `.listRowPlatterColor`, `.listRowSeparatorTint`, `.listRowSpacing`
- `.listSectionSeparatorTint`, `.listSectionSpacing`, `.selectionDisabled`

## Mock data

- If the view takes a list of model types, provide 5 sample entries.
- For `@Binding` parameters, define the binding inline using `@Previewable @State`.
- Prefer existing static vars or globals of the needed type over constructing new mock instances.
- For `Image` / `CGImage` / `NSImage` / `UIImage` parameters, look for existing globals or
  static vars in the project first. Fall back to system images (`Image(systemName:)`).

## Multiple views

If the file defines multiple View types, generate a separate `#Preview` block for each.

## Playground macro

For quick inline experimentation (not UI previews), use the new `#Playground { }` macro:

```swift
#Playground {
    let result = MyType.compute(input: 42)
    print(result)
}
```
