# marg

A markdown reader/editor for people who write a lot of `.md` everywhere — Claude Code skills, Cursor rules, project plans, READMEs, ADRs — and want to jump across all of them without leaving a vim-style keyboard flow.

Local-first. No cloud. No vault setup. No workspace.

Two surfaces, one product:

- **[`tui/`](./tui)** — the terminal version. Go + Bubble Tea. Lives in your shell, alongside Claude Code and tmux. `marg` from any prompt fuzzy-finds every `.md` on the machine.
- **[`mac/`](./mac)** — the native macOS version. Swift + AppKit. For when you want the typography ceiling a TUI can't reach: real bold/italic weights, smooth scroll, generous margins, system fonts.

Codename — eventual product name TBD.

---

See each subdirectory's README for build/install/usage:

- [tui/README.md](./tui/README.md)
- [mac/README.md](./mac/README.md)

And [`unique-selling-points.md`](./unique-selling-points.md) for what marg does that other markdown tools don't.
