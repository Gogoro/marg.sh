# Todos

Big and small things to attack. Not a roadmap, just a parking lot.

## Ole's pile

- [ ] **Super mode** — `marg` with no args (not even `.`) should open from the root of the machine and show all `.md` files everywhere. SCREAM. Lets us work SUPER fast cross-project. Use tools that can handle the scale (e.g. fd / ripgrep) for the search.
- [ ] **`/` for search in `:Ex` mode** — incremental filter on the tree, like rizz's filter.

## big

- [ ] **undo / redo** — currently you can lose work in normal mode if you `dd` the wrong line.
- [ ] **find / replace** — `/` for forward search at minimum, then `n` / `N`.
- [ ] **dictation pipe** — `odett ... | marg` should drop transcribed text into the current buffer at the cursor. README lists it as planned; nothing built yet.
- [ ] **markdown helpers** — bullet-list continuation on enter, toggle heading level, wrap selection in `**bold**` / `*italic*` / `` `code` ``, smart link insert.

## medium

- [ ] **release flow** — `deploy.sh` + GoReleaser + `install.sh` modeled on rizz, so we can publish properly when this goes public.
- [ ] **theme support** — let `theme = "..."` in config swap palettes. Minimum: a light theme.
- [ ] **soft-wrap edge case** — a single word wider than `max_width` gets hard-wrapped silently; should at least not crash, ideally show on the next visual line cleanly.
- [ ] **better cursor highlight in the file tree** — current dim background is too subtle; the cursor row should pop more clearly.

## small

- [ ] **status-bar transient messages** — `flash` lingers until next keypress; should auto-clear after ~2s.
- [ ] **`:set max_width 100` runtime toggle** — change config without editing the file.
- [ ] **double-press `esc` to clear state** — a stuck `pendingKey` (`g` / `d` / `y`) is only cleared by the next press.
- [ ] **README typography** — once we have a logo, swap the plain `# marg` for the centered logo block (rizz pattern).

## ideas (not yet decided)

- [ ] outline view (`ctrl+o`?) showing all headings in current file, jump to one
- [ ] daily notes shortcut — `marg --today` opens or creates `journal/YYYY-MM-DD.md`
- [ ] frontmatter awareness — collapse YAML frontmatter, jump past it on `gg`
- [ ] split view (two files side by side)
- [ ] export current file to PDF via pandoc — one-shot `:export pdf`
