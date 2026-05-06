# marg — terminal surface

Go + Bubble Tea + Lip Gloss. Lives in `tui/`.

## project layout

- `main.go` — tiny shim that calls `marg.Run`
- `internal/marg/` — flat package, all the actual code
  - `app.go` — root Bubble Tea model, view dispatch, picker overlay
  - `cli.go` — argument parsing
  - `buffer.go` — line-based text buffer + cursor-safe edits
  - `editor.go` — modal vim editor, soft-wrap, status bar
  - `tree.go` — `:Ex`-style markdown-only file browser
  - `picker.go` — `ctrl+p` fuzzy file picker
  - `markdown.go` — line + inline markdown styling
  - `config.go` — `~/.config/marg/config.toml` loader
  - `styles.go` — single source of truth for colors / lipgloss styles
  - `themes.go` — palettes (dark / light / sepia)

## code style

- Expressive, clear, written-out Go. Not clever. Not abstracted ahead of time.
- Flat layout inside `internal/marg/`. Resist splitting into more sub-packages.
- Full words over abbreviations.
- No comments unless WHY is non-obvious.
- No trailing summary docstrings or feature explanations in code — those go in README, USP doc, or commit messages.

## build / run

```bash
cd tui
go build -o marg .
./marg                # super mode
./marg .              # tree on cwd
./marg some-file.md   # open file directly
```

Tests: `go test ./...` from `tui/`.

## the iniva brand

Clean and mean: one accent color, no emoji, no splash, every key has one job. Do **not** import the rizz aesthetic. Polish lives in the repo / README / install flow / landing page, not in the TUI chrome itself.
