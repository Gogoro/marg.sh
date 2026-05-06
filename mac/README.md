# marg — macOS

Native markdown editor for macOS. Swift + SwiftUI + AppKit. Built with SwiftPM.

The terminal version of marg ([../tui](../tui)) lives where developers work. This Mac surface exists because typography is the ceiling a TUI can't reach: real serif body text, smooth scroll, system fonts, generous margins.

## requirements

- macOS 14 or later
- Swift 5.9+ / Xcode 15+

## run

```bash
cd mac
swift run Marg
```

Or open `Package.swift` in Xcode and hit ▶︎.

## features (v1 prototype)

- Sidebar file tree of every `.md` under `~` (folders without markdown are hidden)
- `cmd-P` fuzzy picker across every markdown file in the index
- NSTextView-backed editor with prose-grade markdown styling
- `cmd-S` save, dirty-state indicator
- File watcher reloads the open file on external writes (Claude Code, your IDE, `git checkout`)
- Modal vim keys: normal / insert / visual, motions (`hjkl`, `w/b`, `0/$`, `gg/G`), operators (`dd`, `yy`, `p`), and `:w` `:q`
- Arrow keys also work everywhere — modal or not, your call

## status

Greenfield prototype. Not yet at parity with the TUI surface. See `mac/CLAUDE.md` for the deliberate "not yet" list.
