# What makes marg different

A running list of things marg does that other tools (Neovim, Vim, Helix, Obsidian, Typora, Word) typically don't. Add to this list whenever something genuinely differentiating goes in.

## Prose-shaped, not code-shaped

- **Soft-wrap that stops at the terminal width — and an optional `max_width` cap below that.** Almost every code-oriented TUI editor lets text run to the right edge of a wide monitor; marg is designed so paragraphs stay at a comfortable reading length even on a 200-column terminal. `max_width = 80` in config gives you Word-doc line lengths regardless of window size.
- **Prose narrow, code and tables wide — automatically.** Prose wraps at `max_width` for readability, but fenced code blocks and table rows use a separate `code_max_width` (default: full terminal width) so they don't get squished into the prose column. Same left margin, just allowed to extend further right. Other TUI editors force a single wrap width on every kind of content; marg lets the typography of the line follow the shape of the content.
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

- **Tree-sitter inside fenced blocks for sql / go / javascript / typescript / rust / python / bash / yaml.** marg statically links the tree-sitter parsers for those eight languages and ships highlight queries that drive the same `@keyword`/`@function`/`@type`/`@variable.member` taxonomy nvim+catppuccin uses. The result: SQL keywords get one color, table names another, column names a third, strings their own. Function calls light up green-blue, struct fields lavender — grammar-aware, not regex-aware. As far as I know, no other terminal markdown reader bundles tree-sitter for inline rendering.
- **Catppuccin Mocha colours by default.** The bundled tree-sitter palette is the same one catppuccin nvim uses, so SQL/Go/JS/Rust blocks read identically to your editor without configuration.
- **Chroma fallback for everything else.** Languages we don't ship a tree-sitter parser for (Python, TypeScript, shell, YAML, JSON, …) fall through to Chroma, which is regex-tokenizer-good. Auto-detect handles untagged blocks. So you never see "uniform green" — every code block gets the best highlighting we can give it.

## Reading-grade typography in a terminal

- **Hanging indents on wrapped list items.** When a bullet wraps, the continuation aligns under the content, not the dash. Most TUIs (and most code editors) wrap to column 0; marg wraps to where the prose actually started.
- **Vertical rhythm around headings.** A synthetic blank line is rendered above every heading so sections breathe — even if the source markdown didn't include one.
- **Subtle code-block backgrounds.** Fenced blocks get a tinted region across the full row width so they read as a separate kind of content. Cursor / selection / search-match still layer cleanly on top.
- **Cursorline.** A barely-perceptible background tint on the cursor's row, like Obsidian and good IDEs.
- **`:zen`** drops the status bar so the page is just text.
- **Three curated themes** (dark, light, sepia) — each palette designed independently, not just inverted. Light and sepia paint an opaque background so they read correctly even in a dark terminal window.

## Auto-reload on external edits

- **The open file reloads itself when something else writes to it** (Claude Code in another tmux pane, a save from your IDE, a `git checkout`, etc.). marg watches the file via fsnotify; if your buffer is clean, the reload is silent. If you have unsaved changes it just flashes a warning instead of clobbering — you choose with `:e!` (discard) or `:w` (overwrite). Same idea as Neovim's `autoread`, on by default.

## Markdown-native edit shortcuts

- **`*` / `_` / `` ` `` in visual mode wrap the selection in `**bold**`, `_italic_`, `` `code` ``.** No surround-plugin to install. The keys you'd reach for already do the right thing.
- **`:H1` … `:H6` (and `:H0` to remove) toggle the current line's heading level**, preserving indentation. Most editors make you select-the-line-and-prepend manually.
- **List continuation on Enter** carries `-`, `*`, `+`, or auto-incremented numbered bullets, including `- [ ]` / `- [x]` checkbox items. Pressing Enter on an empty bullet exits the list cleanly. Standard in GUI editors; rare in TUIs.
- **Hanging indent under the content of `- [ ]` checkbox items.** A wrapped todo continues under the text after the `]`, not under the dash — so long todos read as a single visual paragraph aligned to the content, the way Notion or a polished GUI editor does it.
- **Frontmatter rendered as a calm block.** YAML frontmatter at the top of a file (`---` … `---`) renders with muted keys and body-color values, fences dimmed. So skill files, blog posts, and other markdown-with-frontmatter open with the metadata visually pushed back instead of competing with the content.

## Built for the dictation + AI workflow

- **Designed to pair with Odett (speech-to-text).** Roadmap: pipe `odett ... | marg` straight into the buffer. Most editors treat dictation as an external paste; marg wants it as a first-class input.

---

*If you build something into marg that doesn't show up in the list above and isn't standard in other editors, add a bullet. We use this doc to figure out the marketing story over time.*
