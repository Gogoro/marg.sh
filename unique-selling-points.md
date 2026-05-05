# What makes marg different

A running list of things marg does that other tools (Neovim, Vim, Helix, Obsidian, Typora, Word) typically don't. Add to this list whenever something genuinely differentiating goes in.

## Prose-shaped, not code-shaped

- **Soft-wrap that stops at the terminal width — and an optional `max_width` cap below that.** Almost every code-oriented TUI editor lets text run to the right edge of a wide monitor; marg is designed so paragraphs stay at a comfortable reading length even on a 200-column terminal. `max_width = 80` in config gives you Word-doc line lengths regardless of window size.
- **`j` / `k` move by visual line, not logical line.** In Neovim, pressing `j` on a wrapped paragraph jumps over the entire paragraph. In marg, it moves down one visible row — the way prose readers expect.
- **No line numbers by default.** Code editors lead with line numbers; for prose they're noise. marg's editor area is just text.

## Markdown-aware, terminal-native navigation

- **Markdown-only file tree.** `:Ex` walks recursively but hides folders that contain no `.md` files. Your notes vault is the whole tree; the build artifacts and config files don't get in the way.
- **VS Code-style fuzzy picker (`ctrl+p`) inside a TUI.** Centered modal, hierarchical sort, subsequence match. Most terminal editors make you install a plugin for this; in marg it's the default and the only file picker.
- **One tool that opens a folder *or* a single file.** `marg`, `marg ./notes`, `marg foo.md`, even `marg new-file.md` (creates it). No "set up a workspace first" step.

## Keybindings that bridge two audiences

- **Vim modal keys *and* arrow keys both work, in every mode.** Most modal editors force a choice; marg lets a vim user do `dd` and a non-vim user press `delete` on the same line.

## Built for the dictation + AI workflow

- **Designed to pair with Odett (speech-to-text).** Roadmap: pipe `odett ... | marg` straight into the buffer. Most editors treat dictation as an external paste; marg wants it as a first-class input.

---

*If you build something into marg that doesn't show up in the list above and isn't standard in other editors, add a bullet. We use this doc to figure out the marketing story over time.*
