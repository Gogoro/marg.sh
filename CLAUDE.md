# marg

A terminal markdown reader/editor under the iniva brand. Go + Bubble Tea + Lip Gloss.

Codename — eventual product name TBD.

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

## the iniva brand

This is an iniva tool, not a personal-labs tool. Polish to the level Odett is polished at: clean, restrained, designer-quality README and chrome.

Clean and mean: one accent color, no emoji, no splash, every key has one job. Do **not** import the rizz aesthetic (cheek, vibes, confetti). Polish lives in the repo / README / install flow / landing page, not in the TUI chrome itself.

## unique-selling-points.md — keep this updated

Whenever a change goes in that gives marg a feature or behavior most other markdown / prose / TUI editors don't have, add a bullet to `unique-selling-points.md`. This is the marketing-narrative scratchpad — we use it to figure out what story to tell when this goes public.

Rules of thumb for what goes in:

- **Yes**: anything that's a genuine *and surprising* difference vs Neovim / Vim / Helix / Obsidian / Typora / Word for the prose use case.
- **No**: standard editor features (it has search, it saves files, it has vim keys). Those aren't differentiators.
- **Frame each bullet user-first**: lead with the user-visible behavior, then briefly say *why* it matters or what other tools do instead.

If you remove a feature, remove its bullet. The list should always reflect what marg actually does today.

## commit messages

Short, lowercase, no conventional-commit prefix. Match the iniva style: `add max_width config`, `fix soft-wrap on empty line`, `wire up dd and yy`.

## todos

Make sure to remind me about the `todos.md` file. There we have big and small things we should attack.
