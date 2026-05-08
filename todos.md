# Todos

- [x] Auto-format markdown tables on save: pad each column to the widest cell, honor `:---` / `---:` / `:---:` alignment, skip tables inside fenced code blocks. See `internal/marg/tables.go`.
  - [ ] Horizontal scroll for tables wider than the viewport. Today they wrap at `code_max_width`; for nvim-style "table flows off to the right", we'd need a real horizontal-scroll path in the renderer.

- [~] Highlight a section and chat with an AI about it.
  - [x] MVP: visual-select → `K` opens centered overlay. Sends full doc + selection in `<document>` / `<selection>` XML envelopes plus `<question>`. Multi-turn conversation, esc to dismiss. Uses `smart_model` from `[ai]` config. See `internal/marg/chat.go`.
  - [x] Scrollback in the transcript. `↑` / `↓` scroll one line, `ctrl+u` / `ctrl+d` half-page, `pgup` / `pgdn` full page. Snaps back to tail when a new turn lands.
  - [x] Streaming responses. Replies render as they arrive, chunk by chunk, instead of blocking on a "thinking…" placeholder.
  - [x] Cancel in-flight request. Esc cancels the streaming request via `context.WithCancel`, so closing the overlay no longer leaks a pending API call.
  - [x] Persist the conversation across opens. `gK` (normal mode) or `:ask` reopens the previous chat; `K` (visual) starts fresh against the new selection.
  - [x] Whole-doc question. `gK` opens chat with no selection; `:ask <text>` opens, pre-fills the question, and submits.
  - [x] `@file.md` mentions in chat input attach the file's contents in an `<attachments>` block. Resolves relative to the current document, supports `~/` and absolute paths. Missing files are silently ignored.
  - [x] Multi-line input. `alt+enter` / `shift+enter` / `ctrl+j` insert a newline; `enter` still sends. The input area grows up to 5 rows, transcript shrinks to fit.
  - [x] Yank reply to clipboard. `ctrl+y` inside the chat copies the latest assistant reply to the OS clipboard.
  - [x] Edit suggestions from chat. `ctrl+e` arms the next send for structured edits — model returns JSON, parsed into the proof pipeline, anchored on the buffer with the same `gA` / `gX` flow. The chat closes once the marks are live.
  - [ ] Markdown rendering inside the transcript (today it's plain text — code blocks and bullets read flat).
  - [ ] Refresh the document context on follow-up turns. Today only the first user turn carries `<document>`; if the doc changes mid-conversation the model still sees the original snapshot.

- [x] Yank/paste through the OS clipboard via `"+y` / `"+yy` / `"+p` / `"+P`. Visual-mode `"+y` copies the selection. Shells out to `pbcopy`/`pbpaste` (macOS) or `wl-copy`/`xclip`/`xsel` (Linux).

- [ ] Table formating gets strange when its linkes inside. Because they get prettied to not have [](), so that needs a fix. Dynamic formating between view and edit view? or something? Maybe view mode is prettied with table format as well, but not stored witht those spaces in the real file. that the edit is the source of truth.

- [ ] go to link, should work for links on page as well, for scroll to heading (#) links

- [ ] Colors are still a bit anoying in the text/code blocks. In nvim its a lot lighter of a green for code blocks wihthout syntax, but we have a very dark green on a dark surface


## Ideas
- [~] Add AI help into the system. For like proof reading, hints on things I could adjust and so on. Augment my flow. Should be in the marg (!!!)
  - [x] MVP: `:proof` runs Haiku 4.5 over the document, marks suggestions inline with an underline, right-margin `→ replacement` reveal, status-bar reason on cursor, `]s`/`[s` nav, `gA`/`gX` accept/reject. See `proofreading-plan.md` for the full roadmap.
  - [ ] Phase 2: paragraph-level idle trigger, right-margin reveal at wide widths, below-paragraph callout at medium widths
  - [ ] Phase 3: `:proof %` substantive pass, `]A` accept-all-in-paragraph, visual-mode selection scope


TUI
- [~] add more languages to the treesitter
  - [ ] JSON — no Go binding in `smacker/go-tree-sitter`; falls through to Chroma (which handles JSON cleanly already)
  - [x] Dockerfile (basic: comments, strings, image specs, env/label keys, expansions — keywords like `FROM`/`RUN` aren't anonymous strings in this grammar, so they're not highlighted)
  - [x] TOML
  - [x] Lua (limited: no keyword highlighting because the grammar doesn't expose them as queryable anon strings — comments, strings, numbers, booleans, function names, identifiers all work)
  - [x] HTML
  - [x] CSS
