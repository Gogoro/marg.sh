# Todos

- [ ] Keybinds
  - [x] c+w for cutting out a word is dearly missed
  - [ ] g+d to jump to an link in the markdown. internal link jump to doc, external link opens browser
- [x] Make links look nice. Only show the full part when having cursor over. I think this will make the docs a lot easier and nicer to read. (collapsed `[text](url)` to just the link-styled text; full markdown shows on cursor-over and in insert mode)
- [ ] Fix the searching in the cmd+p. It doesnt hit the most accurate file


## Ideas
- [~] Add AI help into the system. For like proof reading, hints on things I could adjust and so on. Augment my flow. Should be in the marg (!!!)
  - [x] MVP: `:proof` runs Haiku 4.5 over the document, marks suggestions inline with an underline, right-margin `→ replacement` reveal, status-bar reason on cursor, `]s`/`[s` nav, `gA`/`gX` accept/reject. See `proofreading-plan.md` for the full roadmap.
  - [ ] Phase 2: paragraph-level idle trigger, right-margin reveal at wide widths, below-paragraph callout at medium widths
  - [ ] Phase 3: `:proof %` substantive pass, `]A` accept-all-in-paragraph, visual-mode selection scope


TUI
- [ ] add more languages to the treesitter
  - [ ] JSON
  - [ ] Dockerfile
  - [ ] TOML
  - [ ] Lua
  - [ ] HTML
  - [ ] CSS
