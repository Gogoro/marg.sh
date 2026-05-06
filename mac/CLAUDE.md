# marg — macOS surface

Swift + SwiftUI + AppKit. Built with Swift Package Manager. Lives in `mac/`.

The Mac surface exists because typography is the ceiling we couldn't break in the terminal. CoreText, system fonts, smooth scroll, real bold/italic weights, generous margins. Everything below leans into that — keep the chrome quiet and the text room generous.

## project layout

`Package.swift` at `mac/`. Source files live flat in `mac/Sources/Marg/`:

- `App.swift` — `@main MargApp`, top-level Scene + commands (cmd-P, cmd-S)
- `AppState.swift` — observable state (open file, dirty flag, file tree, picker visibility, vim mode)
- `ContentView.swift` — `NavigationSplitView` shell
- `FileTreeView.swift` / `FileTreeWalker.swift` / `FileNode.swift` — sidebar + recursive walker
- `MarkdownEditor.swift` — `NSViewRepresentable` wrapping `NSTextView`
- `MarkdownStyler.swift` — text → `NSAttributedString` for prose-grade rendering
- `FuzzyPicker.swift` / `FuzzyMatcher.swift` — cmd-P sheet + subsequence scoring
- `FileWatcher.swift` — `DispatchSource` file watcher; reload on external write
- `VimMode.swift` — modal state machine (normal / insert / visual / command line)
- `Theme.swift` — fonts, sizes, spacing, color tokens

## code style

- Expressive, clear Swift. Written out. Not clever.
- Flat file layout inside `Sources/Marg/`. Resist sub-folders.
- Full words over abbreviations (`fileTreeWalker`, not `ftw`).
- No comments unless WHY is non-obvious.
- No backwards-compat shims; greenfield.
- Prefer `struct` + value types where SwiftUI doesn't force `class`.
- Only `class` for `ObservableObject` state and reference-semantic helpers (file watcher).

## build / run

```bash
cd mac
swift run Marg
```

The executable target uses SPM directly; you can also `open Package.swift` in Xcode.

## what's intentionally NOT here yet

Not a missing-feature list — a do-not-over-build list:
- Multi-document tabs
- Per-file search (cmd-F inside the buffer)
- Markdown shortcut keys (`*` / `_` wrap in visual)
- Heading toggles (`:H1`)
- List continuation
- Themes / config file
- Super-mode `fd`/`rg` index
- Markdown table rendering / image previews

Add when there's a real reason. The point of v1 is the *feel* of opening, reading, editing markdown.

## vim mode

Vim mode is implemented in `VimMode.swift` but **gated off** via `appState.vimEnabled = false`. Default behavior is plain NSTextView editing. Vim becomes a setting we expose later, once the visual feel is dialed in. Don't reference vim mode in onboarding or status chrome — keep the default surface clean.
