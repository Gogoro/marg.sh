# What makes marg different

A running list of things marg does that other tools (Neovim, Vim, Helix, Obsidian, Typora, Word) typically don't. Add to this list whenever something genuinely differentiating goes in.

## Prose-shaped, not code-shaped

- **Soft-wrap that stops at the terminal width — and an optional `max_width` cap below that.** Almost every code-oriented TUI editor lets text run to the right edge of a wide monitor; marg is designed so paragraphs stay at a comfortable reading length even on a 200-column terminal. `max_width = 80` in config gives you Word-doc line lengths regardless of window size.
- **`j` / `k` move by visual line, not logical line.** In Neovim, pressing `j` on a wrapped paragraph jumps over the entire paragraph. In marg, it moves down one visible row — the way prose readers expect.
- **No line numbers by default.** Code editors lead with line numbers; for prose they're noise. marg's editor area is just text.

## Markdown-aware, terminal-native navigation

- **Super mode (`marg` with no args).** Drops you straight into a fuzzy picker over every `.md` file under `super_roots` (defaults to `$HOME`). Cross-project navigation in 2 keystrokes — type `marg` from any terminal, `roa` for "roadmap", enter. No editor I know of treats your machine-wide notes vault as a first-class search target.
- **Markdown-only file tree.** `:Ex` walks recursively but hides folders that contain no `.md` files. Your notes vault is the whole tree; the build artifacts and config files don't get in the way.
- **`/` filter inside the file tree.** Live substring narrow over the whole tree (across all unexpanded folders too). Most terminal trees make you toggle into a separate "find file" command; in marg it's just `/`.
- **VS Code-style fuzzy picker (`ctrl+p`) inside a TUI.** Centered modal, hierarchical sort, subsequence match. Most terminal editors make you install a plugin for this; in marg it's the default and the only file picker.
- **One tool that opens nothing, a folder, or a single file.** `marg`, `marg ./notes`, `marg foo.md`, even `marg new-file.md` (creates it). No "set up a workspace first" step.

## Keybindings that bridge two audiences

- **Vim modal keys *and* arrow keys both work, in every mode.** Most modal editors force a choice; marg lets a vim user do `dd` and a non-vim user press `delete` on the same line.

## Real syntax highlighting inside fenced code blocks

- **Code in markdown gets the full Chroma highlighter treatment.** ` ```go ` colors keywords, strings, comments, function names. Blocks without a language tag are auto-detected. Most TUI markdown editors color the fenced block uniformly green; marg actually parses the code.

## Auto-reload on external edits

- **The open file reloads itself when something else writes to it** (Claude Code in another tmux pane, a save from your IDE, a `git checkout`, etc.). marg watches the file via fsnotify; if your buffer is clean, the reload is silent. If you have unsaved changes it just flashes a warning instead of clobbering — you choose with `:e!` (discard) or `:w` (overwrite). Same idea as Neovim's `autoread`, on by default.

## Markdown-native edit shortcuts

- **`*` / `_` / `` ` `` in visual mode wrap the selection in `**bold**`, `_italic_`, `` `code` ``.** No surround-plugin to install. The keys you'd reach for already do the right thing.
- **`:H1` … `:H6` (and `:H0` to remove) toggle the current line's heading level**, preserving indentation. Most editors make you select-the-line-and-prepend manually.
- **List continuation on Enter** carries `-`, `*`, `+`, or auto-incremented numbered bullets. Pressing Enter on an empty bullet exits the list cleanly. Standard in GUI editors; rare in TUIs.

## Built for the dictation + AI workflow

- **Designed to pair with Odett (speech-to-text).** Roadmap: pipe `odett ... | marg` straight into the buffer. Most editors treat dictation as an external paste; marg wants it as a first-class input.

---

*If you build something into marg that doesn't show up in the list above and isn't standard in other editors, add a bullet. We use this doc to figure out the marketing story over time.*
