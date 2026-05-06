# marg

A markdown reader/editor under the iniva brand. Two surfaces, one product.

- `tui/` — the original Go + Bubble Tea + Lip Gloss terminal version
- `mac/` — native macOS app in Swift (SwiftUI + AppKit), Swift Package Manager build

Both surfaces share the same product idea: jump across every markdown file on the machine, vim keys, no cloud, no workspace setup. The TUI lives where developers work (terminal, tmux, alongside Claude Code). The Mac app gives the typography ceiling a TUI can't reach.

Codename — eventual product name TBD.

## brand & feel

This is an iniva tool, not a personal-labs tool. Polish to the level Odett is polished at: clean, restrained, designer-quality README and chrome.

Do **not** import the rizz aesthetic (cheek, vibes, confetti). Different brand on purpose.

## working in this repo

Each surface has its own per-stack guide:

- `tui/CLAUDE.md` — Go code style, project layout, build/test commands
- `mac/CLAUDE.md` — Swift code style, project layout, build/test commands

The shared cross-surface concerns live here.

## unique-selling-points.md — keep this updated

Whenever a change goes in that gives marg a feature or behavior most other markdown / prose / TUI editors don't have, add a bullet to `unique-selling-points.md`. This is the marketing-narrative scratchpad — we use it to figure out what story to tell when this goes public.

Rules of thumb for what goes in:

- **Yes**: anything that's a genuine *and surprising* difference vs Neovim / Vim / Helix / Obsidian / Typora / Word for the prose use case.
- **No**: standard editor features (it has search, it saves files, it has vim keys). Those aren't differentiators.
- **Frame each bullet user-first**: lead with the user-visible behavior, then briefly say *why* it matters or what other tools do instead.

If you remove a feature, remove its bullet. The list should always reflect what marg actually does today (across either surface).

## code style (cross-surface)

- Expressive, clear, written-out code. Not clever. Not abstracted ahead of time.
- Full words over abbreviations.
- No comments unless WHY is non-obvious.
- No trailing summary docstrings or feature explanations in code — those go in README, USP doc, or commit messages.
- Flat layouts inside each surface's source dir. Resist splitting into more sub-packages/modules.

## commit messages

Short, lowercase, no conventional-commit prefix. Match the iniva style: `add max_width config`, `fix soft-wrap on empty line`, `wire up dd and yy`.

## todos

Make sure to remind me about the `todos.md` file. There we have big and small things we should attack across both surfaces.
