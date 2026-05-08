# proofreading & ai notes — plan

Living doc for the AI proofreading feature. Roadmap, design constraints, open
questions. Update as we ship.

## What this is

An AI-assisted proofreader that lives inside marg. It marks weak spots in
prose with a calm visual, lets the user navigate between them with vim-style
keys, and accepts or rejects each one with a single chord. It's tuned for
**technical writing**.

Identity rule: marg = the page. Suggestions are visible enough to trust,
quiet enough to ignore. If a user writes for an hour and never engages with
any suggestion, the page should still feel like the page.

## Layered design

Three tiers, smallest first.

1. **Marker (always-on).** A subtle dotted/desaturated underline under the
   affected span. Optional gutter sign per paragraph. The page reads
   normally; you only see the marks if you look for them.
2. **Reveal (on cursor).** When the cursor lands on or next to a marker,
   the suggestion text appears — adapted to terminal width (see below).
3. **Action.** `]s` / `[s` to navigate; `gA` accept; `gX` reject (dismiss).

## Reveal modes (width-adaptive)

Same suggestion data, three rendering paths picked by `e.width`:

- **Wide (≥ ~110 cols):** suggestion text floats in the **right margin**
  aligned to the affected row. This is the "marg = margin" play.
- **Medium (~80–110 cols):** one-line callout below the paragraph.
- **Narrow (< 80 cols):** status bar — `[1/3] their → there  =accept -reject`.

Status-bar reveal is the MVP fallback that works at every width.

## Trigger

- **MVP:** explicit `:proof` command on the whole document (or
  `:proof` in visual mode = selection scope, future).
- **Phase 2:** paragraph-level idle trigger. Track which paragraph the
  cursor last edited, hash it, fire a request 2–3s after typing stops if
  the hash changed. Cancel in-flight on resumed typing.
- **Phase 3:** `:proof %` for whole-doc heavy pass with deeper notes.

## Suggestion taxonomy

- **mechanical** — spelling, dropped words, agreement, basic grammar.
  Auto-trigger candidate. Single-span replace.
- **stylistic** — passive voice, wordiness, weak verbs, redundancy.
  Auto-trigger candidate, lighter visual weight.
- **substantive** — paragraph-level "this buries the lede". On-demand
  only (`:proof %`). Renders as advisory, not a one-key replace.

MVP ships **mechanical + stylistic**. Substantive is Phase 3.

## Anchoring & staleness

- Each `suggestion` carries the verbatim `original` string from the model
  plus `replacement`, `reason`, `kind`.
- We anchor by `strings.Index(line, original)` per line — first match wins.
  Cheap. Robust against the user editing other parts of the doc.
- If the user edits the line containing a suggestion, the next render fails
  to find `original` → suggestion is silently dropped.
- Suggestions are **transient**. They never persist to disk. Reload, save,
  or reopen clears them.

## Keys

- `:proof` — run proofread on the whole doc.
- `:proof!` — clear all suggestions.
- `]s` / `[s` — next / previous suggestion (matches vim's spell-error nav).
- `gA` — accept suggestion at cursor (replaces `original` with `replacement`).
- `gX` — reject / dismiss suggestion at cursor.
- `g=` (Phase 2) — show full reason in status bar.
- `]A` (Phase 2) — accept all in current paragraph.

## Model & API

- **SDK:** official `github.com/anthropics/anthropic-sdk-go`.
- **Auth:** `[ai] api_key` in `~/.config/marg/config.toml` wins; empty
  falls back to the `ANTHROPIC_API_KEY` env var; if neither is set, AI
  features are off and `:proof` flashes a hint.
- **Two model slots, role-based.** Each feature picks one, so adding
  features doesn't bloat config.
  - `fast_model` (default `claude-haiku-4-5`) — inline mechanical work.
    Used by today's `:proof` and Phase 2 paragraph idle trigger.
  - `smart_model` (default `claude-sonnet-4-6`) — substantive passes.
    Reserved for `:proof %` and paragraph rewrite (Phase 3).
- **Cost shape:** whole-doc `:proof` ≈ one Haiku call per invocation.
  Paragraph-scoped Phase 2 ≈ one Haiku call per paragraph you edit.
- **Output:** strict JSON array; we parse it with `encoding/json`. The
  prompt forbids quoting code blocks, URLs, file paths, or jargon.

## Cost / safety controls (Phase 2+)

- Per-session call counter shown in status bar when over a threshold.
- Config `[ai] proof_kinds = ["mechanical"]` to disable the stylistic pass.
- Config `[ai] auto_proof = false` to disable idle triggers.

## What we're explicitly NOT doing

- No red squiggles. No "X errors found!" panels. No spinning indicators.
- No persisted suggestion state. No diff view.
- No multi-agent chorus. One proofreader. Phase 1 ships single voice; we
  revisit personas only if there's pull.
- No grammar engine fallback. If there's no API key, the feature is off.

## Roadmap

### MVP (shipped)

- [x] `:proof` on whole document
- [x] mechanical + stylistic categories
- [x] underline marker on each suggestion span
- [x] right-margin **per-suggestion boxes with Z-order stacking** —
      each suggestion gets its own bordered box anchored to the visual
      row where its span starts; boxes never rearrange. Where they
      overlap, the hovered suggestion (cursor inside its span) wins on
      depth — it renders last so its cells overwrite back-boxes' rows,
      which peek out at the edges of the front box like a stack of
      cards. Hovered uses the **accent color** + bold heading;
      non-hovered uses a kind-based color (mechanical = warm H3,
      stylistic = cool H6 from the heading palette) so the type reads
      at a glance. Bottom-edge aware: a box whose anchor is near the
      viewport bottom gets shifted up just enough to fit, so the
      bottom border lands on the viewport's last row. Top-edge: a box
      whose anchor is above the viewport is hidden. Long replacements
      wrap. Shown when there's ≥ 30 cols of right-side room.
- [x] inline `→ replacement` annotation — used as the fallback at narrow
      terminal widths where the box can't fit.
- [x] status-bar reveal — reason + key hints (`gA accept · gX reject`)
      when the cursor sits on a suggestion. Stays compact since the box
      already shows the replacement.
- [x] `]s` / `[s` nav, `gA` accept, `gX` reject
- [x] suggestions transient — never persisted, dropped silently when the
      anchored line text changes
- [x] config: `[ai] api_key`, `fast_model`, `smart_model` in
      `~/.config/marg/config.toml`; `ANTHROPIC_API_KEY` env var fallback

### Phase 2

- [ ] paragraph-level idle trigger with debounce + hash
- [ ] in-flight cancellation on resumed typing
- [ ] below-paragraph callout at medium widths (when neither margin nor
      narrow status-bar fits cleanly)
- [ ] `g=` show full reason in flash (when reason is truncated)
- [ ] config: `[ai] auto_proof`, `proof_kinds`

### Phase 3

- [ ] `:proof %` substantive pass (paragraph-level advisory notes)
- [ ] `]A` accept all in paragraph
- [ ] visual-mode `:proof` to scope to selection
- [ ] streaming: render markers as they arrive

## Open questions

- **Margin rendering** — needs careful interaction with the existing
  `centerAbove` / `leftMargin()` math. Probably a separate trailing column
  past `wrapWidth()`.
- **Line-spanning suggestions** — current MVP requires the `original` to
  fit on one line. If the model returns a span across a line break, we drop
  it. May need to expand to multi-line spans for paragraph-level rewrites.
- **Voice preservation** — Haiku tends to suggest "improvements" that
  flatten voice. Watch for over-correction in technical-blog drafts and
  tune the prompt.
- **Streaming** — per-paragraph requests are small enough that streaming
  doesn't help. Keep it batched.
