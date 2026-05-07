# marg

A terminal markdown reader/editor. Go + Bubble Tea + Lip Gloss.

Personal open source project. Repo: `github.com/Gogoro/marg.sh`.

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
go build -o marg .
./marg                # super mode
./marg .              # tree on cwd
./marg some-file.md   # open file directly
```

Tests: `go test ./...` from the repo root.

## design philosophy

Clean and restrained: one accent color, no emoji, no splash, every key has one job. The polish lives in the repo, README, and install flow — the TUI itself stays calm and out of the way. No cheek, no vibes, no confetti. The reader should feel like they're looking at the page, not at the editor.

## unique-selling-points.md — keep this updated

Whenever a change goes in that gives marg a feature or behavior most other markdown / prose / TUI editors don't have, add a bullet to `unique-selling-points.md`. This is the marketing-narrative scratchpad — we use it to figure out what story to tell.

Rules of thumb for what goes in:

- **Yes**: anything that's a genuine *and surprising* difference vs Neovim / Vim / Helix / Obsidian / Typora / Word for the prose use case.
- **No**: standard editor features (it has search, it saves files, it has vim keys). Those aren't differentiators.
- **Frame each bullet user-first**: lead with the user-visible behavior, then briefly say *why* it matters or what other tools do instead.

If you remove a feature, remove its bullet. The list should always reflect what marg actually does today.

## commit messages

Short, lowercase, no conventional-commit prefix. Examples: `add max_width config`, `fix soft-wrap on empty line`, `wire up dd and yy`.

## todos

Remind me about the `todos.md` file. There we have big and small things we should attack.
