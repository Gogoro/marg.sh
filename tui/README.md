# marg

A terminal markdown editor. Word-doc feel, vim keys, no forever-long lines.

> Codename — eventual product name TBD.

<p align="center">
  <img src="assets/screenshots/editor-prose.png" alt="marg editing a journal entry with soft-wrapped prose" width="900"/>
</p>

## why

Most of my work lives in the terminal these days. Neovim is great for code, less great for prose: lines run to the horizon, no obvious file picker for "all my notes", no easy "browse folder of markdown" entry point. marg is a small, focused TUI that does just that:

- soft-wrap that stops at the edge of your terminal — paragraphs read like a Word doc
- vim keybindings (or arrow keys, both work)
- VS Code-style fuzzy file picker (`ctrl+p`)
- netrw-style file tree (`:Ex`) — markdown-only, recursive
- pipe-friendly: `odett ... | marg` works (planned)

## install

```bash
cd tui
go build -o marg .
mv marg ~/.local/bin/   # or anywhere on $PATH
```

Requires Go 1.24+.

### strongly recommended: install `fd` or `rg`

Super mode (`marg` with no arguments) walks your home directory looking for every `.md` file. Without help, Go's built-in walker takes 10+ seconds across a real `$HOME`. With **`fd`** ([install](https://github.com/sharkdp/fd#installation)) or **`rg`** ([install](https://github.com/BurntSushi/ripgrep#installation)) on `$PATH`, marg shells out to whichever it finds first and the same walk takes about a second.

```bash
# macOS
brew install fd            # preferred (a touch faster)
# or:
brew install ripgrep

# Debian / Ubuntu
sudo apt install fd-find ripgrep

# Arch
sudo pacman -S fd ripgrep
```

Marg works without either tool installed — you'll just see `indexing…` for noticeably longer the first time you launch super mode.

## usage

```bash
marg                  # super mode: fuzzy-find any .md anywhere under super_roots (default: $HOME)
marg .                # open the file tree on the current directory
marg path/to/dir      # open the file tree on that directory
marg path/to/file.md  # open that file directly
```

Super mode is the fastest way to jump to any note across all your projects — `marg` from any terminal, type a few letters, hit enter.

### file tree

`marg` (or `marg .`) drops you straight into a recursive markdown-only tree. Folders without any `.md` files don't show up — your notes vault is the whole tree, the build artifacts and config files don't get in the way.

<p align="center">
  <img src="assets/screenshots/file-tree.png" alt="marg file tree showing journal, notes, and projects folders" width="900"/>
</p>

### fuzzy file picker (`ctrl+p`)

`ctrl+p` from anywhere opens a centered overlay. Type to filter, up/down to navigate, enter to open, esc to cancel. Subsequence match (so `oroad` finds `odett/roadmap.md`).

<p align="center">
  <img src="assets/screenshots/fuzzy-picker.png" alt="marg fuzzy file picker filtering on the query 'launch'" width="900"/>
</p>

### markdown styling

Headings, blockquotes, sub-headings, bullet lists, **bold**, *italic*, `inline code`, and [links](https://example.com) are all styled inline as you type — soft-wrapped like a Word document, no line numbers, nothing in your way.

<p align="center">
  <img src="assets/screenshots/markdown-styling.png" alt="a launch checklist styled with heading, blockquote and bullets" width="900"/>
</p>

### code blocks

Fenced code blocks with an explicit language (e.g. ` ```go `) get full syntax highlighting via [Chroma](https://github.com/alecthomas/chroma) — keywords, strings, comments, numbers, function names, all colored. Blocks without a language tag are auto-detected when Chroma is confident; otherwise they render as plain text.

<p align="center">
  <img src="assets/screenshots/code-highlight.png" alt="syntax-highlighted Go and Python code blocks inside a markdown file" width="900"/>
</p>

### visual selection

`v` for character selection, `V` for line selection. Extend with any motion, then `y` to yank, `d`/`x` to cut, `p` to replace.

<p align="center">
  <img src="assets/screenshots/visual-line-select.png" alt="V-LINE selection across several lines of a journal entry" width="900"/>
</p>

## keybindings

### everywhere

| key      | action                       |
|----------|------------------------------|
| `ctrl+p` | fuzzy file picker            |
| `ctrl+e` | toggle file tree             |

### editor — normal mode

| key            | action                                |
|----------------|---------------------------------------|
| `h j k l` / arrows | move cursor (j/k = visual line)   |
| `0` / `home`   | start of line                         |
| `_` / `^`      | first non-blank character             |
| `$` / `end`    | end of line                           |
| `gg`           | top of buffer                         |
| `G`            | bottom of buffer                      |
| `w` / `b`      | word forward / backward               |
| `ctrl+d` / `ctrl+u` | jump half page down / up         |
| `ctrl+f` / `ctrl+b` | jump full page down / up         |
| `i` / `a`      | insert before / after cursor          |
| `I` / `A`      | insert at line start / end            |
| `o` / `O`      | open new line below / above           |
| `x`            | delete char under cursor              |
| `u` / `ctrl+r` | undo / redo                           |
| `dd`           | cut (delete) current line             |
| `yy` / `Y`     | yank (copy) current line              |
| `p` / `P`      | paste after / before cursor (or below/above for line-wise) |
| `v` / `V`      | visual / visual-line selection mode   |
| `/`            | search forward (then `n` / `N` to repeat) |
| `:`            | command line (`:w`, `:q`, `:wq`, `:e`, `:e!`, `:zen`, `:Ex`, `:H1..:H6`, `:s/…`, `:%s/…`) |
| `ctrl+s`       | save                                  |

### editor — visual mode (`v` / `V`)

Move with motions to extend the selection. Then:

| key            | action                                |
|----------------|---------------------------------------|
| `y`            | yank selection                        |
| `d` / `x`      | cut selection                         |
| `p`            | replace selection with register       |
| `*`            | wrap selection in `**bold**`          |
| `_`            | wrap selection in `_italic_`          |
| `` ` ``        | wrap selection in `` `code` ``        |
| `esc`          | leave visual mode                     |

### editor — insert mode

Type freely. Arrow keys, backspace, delete, enter, tab all behave naturally.
`esc` returns to normal mode.

Markdown list continuation: pressing Enter on a line that starts with `- `, `* `, `+ `, or `<n>. ` carries the bullet (or auto-incremented number) onto the new line. An empty bullet on Enter exits the list cleanly.

### file tree (`:Ex`)

| key            | action                                |
|----------------|---------------------------------------|
| `j k` / arrows | move cursor                           |
| `enter` / `l`  | open file or expand folder            |
| `h`            | collapse folder / jump to parent      |
| `g` / `G`      | top / bottom                          |
| `%`            | new file (creates `.md` if no ext)    |
| `d`            | new directory                         |
| `D`            | delete (with confirmation)            |
| `/`            | filter tree (substring match across paths) |
| `R`            | refresh                               |
| `esc` / `q`    | clear filter, then back to editor     |

### file picker (`ctrl+p`)

Type to filter (subsequence match). Up/down to navigate. Enter to open. Esc to cancel.

## config

Optional config at `~/.config/marg/config.toml`. Format is `key = value` per line, `#` for comments.

```toml
# Editor palette. Three curated themes ship with the binary.
#   "dark"  → soft off-white on the terminal background (default)
#   "light" → Solarized Light: dark warm grey on a cream background
#   "sepia" → warmer paper-like cream that's easy on the eyes
# Light and sepia paint an opaque background so they look right even when
# launched from a dark terminal.
theme = "dark"

# Cap the wrap width below the terminal width — useful in wide terminals
# where prose otherwise stretches out into a horizontal blur.
max_width = 80

# Terminal width at which the text block starts being horizontally centered.
# 0 disables centering (text hugs the left edge — default). A typical value
# on a wide monitor: 160.
center_above = 0

# Theme for syntax highlighting inside fenced code blocks.
#   "ansi"  → uses your terminal's bright ANSI colors (default).
#            Keywords get bright magenta, strings yellow, comments grey,
#            etc. Always readable because YOUR terminal theme picks the
#            actual shades.
#   "<name>" → any Chroma style name ("dracula", "monokai", "tokyo-night",
#            "github-dark", "onedark", "gruvbox", "nord", "rose-pine", …).
#            More color variety but depends on truecolor reaching marg
#            cleanly (some tmux configs strip it).
code_theme = "ansi"

# Where super mode (running `marg` with no args) walks for markdown files.
# Defaults to your home directory. Use `~` for HOME, or any absolute path.
super_roots = ["~"]

# Extra directory basenames to skip (added to the built-in defaults like
# node_modules, go, Library, target, build, dist, Pods, Carthage,
# DerivedData, coverage, Applications). Useful for personal noisy folders.
ignore_dirs = ["Downloads", "Dropbox"]

# Whitelisted directories that should be searched even though the default
# rules would skip them — usually dot-prefixed dirs that contain notes.
# marg only descends through visible, non-noise ancestors when looking
# for these, so a `.claude` inside `node_modules` won't be surfaced.
include_dirs = [".claude", ".obsidian"]
```

### `max_width` in action

A wide terminal without `max_width` lets a single paragraph spread across the whole screen — readable in theory, exhausting in practice.

<p align="center">
  <img src="assets/screenshots/wrap-without-cap.png" alt="long paragraph stretched across a 200-column terminal" width="900"/>
  <br/>
  <i>Without <code>max_width</code> — text fills the entire terminal width.</i>
</p>

The same paragraph with `max_width = 80` stays at a comfortable book-page column even on a huge monitor.

<p align="center">
  <img src="assets/screenshots/wrap-with-cap.png" alt="same paragraph held at 80 columns regardless of terminal width" width="900"/>
  <br/>
  <i>With <code>max_width = 80</code> — text held at a comfortable reading width.</i>
</p>

## reading-friendly typography

A few small touches that add up:

- **Hanging indents on wrapped list items.** When `- a long bullet` soft-wraps, the continuation aligns under the content, not under the dash.
- **Vertical rhythm around headings.** A blank visual line is rendered above every heading so sections breathe.
- **Subtle code-block backgrounds.** Fenced code blocks get a tinted region so they read as a separate kind of content without screaming.
- **Cursorline.** The line under the cursor gets a barely-perceptible background tint so you don't lose your place when looking away.
- **Centered text on wide monitors** via `center_above` (see config below).
- **`:zen`** toggles a reading mode that hides the status bar — only text remains.

## recommended fonts

Marg can't pick your terminal font for you, but a few work especially well for long-form prose:

- **[iA Writer Mono S](https://github.com/iaolo/iA-Fonts)** — designed for prose. Slightly looser tracking, real italics, very calm.
- **[Monaspace Neon](https://monaspace.githubnext.com/)** — GitHub's prose-leaning monospace. Pairs well with Krypton for code.
- **[JetBrains Mono](https://www.jetbrains.com/lp/mono/)** — popular default; ligatures off if reading prose.
- **[Berkeley Mono](https://berkeleygraphics.com/typefaces/berkeley-mono/)** — premium, but the typography is a step up if you live in a terminal.
- **[IBM Plex Mono](https://www.ibm.com/plex/)** — free, calm, pairs with Plex Serif if you want a matched display font.

Fira Code and Cascadia Code work too; both are oriented more toward code than prose, but they're widely installed.

## regenerating the screenshots

The screenshots above are generated from a [VHS](https://github.com/charmbracelet/vhs) tape and a throwaway notes vault.

```bash
brew install vhs            # one-time
bash demo/setup.sh          # builds /tmp/marg-demo
vhs demo/tape/main.tape     # main screenshot set
vhs demo/tape/wrap.tape     # max_width comparison
```

PNGs land in `assets/screenshots/`.

## status

v1 — works for me. Lots to add (undo, find/replace, list helpers, dictation pipe).
