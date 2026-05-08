# Todos

- [x] Auto-format markdown tables on save: pad each column to the widest cell, honor `:---` / `---:` / `:---:` alignment, skip tables inside fenced code blocks. See `internal/marg/tables.go`.
  - [ ] Horizontal scroll for tables wider than the viewport. Today they wrap at `code_max_width`; for nvim-style "table flows off to the right", we'd need a real horizontal-scroll path in the renderer.

- [~] Highlight a section and chat with an AI about it.
  - [x] MVP: visual-select → `K` opens centered overlay. Sends full doc + selection in `<document>` / `<selection>` XML envelopes plus `<question>`. Multi-turn conversation, esc to dismiss. Uses `smart_model` from `[ai]` config. See `internal/marg/chat.go`.
  - [x] Scrollback in the transcript. `↑` / `↓` scroll one line, `ctrl+u` / `ctrl+d` half-page, `pgup` / `pgdn` full page. Snaps back to tail when a new turn lands.
  - [ ] Streaming responses (today the overlay shows "thinking…" and blocks until the full reply lands; for long answers this feels slow).
  - [ ] Auto-attach supporting documents. Open question: should it be explicit (`@other-file.md` syntax in the input), implicit (follow markdown links from the current doc), or config-driven (a `context_dirs` list)? Pick one before building.
  - [ ] Copy a reply (or part of a reply) back into the document. Today you read the answer and re-type or remember it. A `y` in a "browse-the-transcript" sub-mode could yank the focused message; or an `:apply` that drops it under the selection.
  - [ ] Open the overlay without a selection (whole-doc question) — e.g. normal-mode `gK` or a `:ask` command.
  - [ ] Multi-line input. Enter currently sends; sometimes a question wants a paragraph.
  - [ ] Markdown rendering inside the transcript (today it's plain text — code blocks and bullets read flat).
  - [ ] Cancel an in-flight request. Esc closes the overlay but the request keeps running in the background; the result is just dropped on arrival.
  - [ ] Persist the conversation across opens for the same selection (or at least make it possible to reopen the last chat with `gK` or similar).
  - [ ] Have the possibility to get edit suggests as well from the chat, that I can accept :o Huge boost. 

- [ ] yank to clippboard y + " (i think, same as vim) 


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
