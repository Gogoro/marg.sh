#!/usr/bin/env bash
# Builds a throwaway markdown notes vault for marg screenshots.
# Usage:   ./demo/setup.sh [target-dir]
# Default: /tmp/marg-demo

set -euo pipefail

TARGET="${1:-/tmp/marg-demo}"

rm -rf "$TARGET"
mkdir -p "$TARGET"/{journal,notes,projects/sidekick,projects/personal-site}

cat > "$TARGET/README.md" <<'EOF'
# notes

A small slice of a second brain. Journal lives in `journal/`, loose notes
in `notes/`, ongoing projects in `projects/`.
EOF

cat > "$TARGET/journal/2026-04-12.md" <<'EOF'
# Sunday, April 12

The morning was slow in the best way. Coffee on the porch, no phone, just the
sound of the neighbour's sprinkler clicking through its cycle. I started reading
*The Pragmatic Programmer* again, the parts about orthogonality, and was reminded
how much of our work is really about reducing the surface area where mistakes can
happen.

Later I sat down to think about what I want the rest of this quarter to look
like. Three things kept coming back:

- ship the alpha by the end of the month, even if the polish isn't perfect
- close out the long tail of half-finished side notes — they weigh on me
- find one full afternoon a week with no meetings, no tickets, just a long
  walk and a notebook

> "Slow is smooth, smooth is fast." — heard this somewhere, can't remember
> where, but it's been rattling around all week.

Tomorrow: review the migration plan with the team. Make sure the rollout
sequence actually matches what's in the runbook.
EOF

cat > "$TARGET/journal/2026-04-15.md" <<'EOF'
# Wednesday, April 15

Big day. The streaming pipeline finally hit sub-second latency end-to-end on
a real machine without the fan kicking on. Felt like crossing a threshold —
until today the demo always had a "wait, let me start over" moment that
pulled people out of the magic. Now it just *works*.

## what got us here

1. Switched the chunker to a smaller window
2. Cached the language detection result for the session
3. Stopped re-tokenizing the whole transcript on every partial result
4. Moved the decode off the main thread

The third one was the surprise. We thought it was a model latency problem
all the way down, and it turned out half the budget was being spent in
JavaScript string manipulation.

## tomorrow

- record a fresh demo video before anyone notices it's broken again
- write up the perf notes in the team channel
- groceries: olive oil, lemons, parsley, the good salt
EOF

cat > "$TARGET/notes/ideas.md" <<'EOF'
# ideas worth chewing on

A loose pile. Most of these will never become anything. That's fine.

## tools

- A terminal markdown editor that doesn't make prose feel like code. Soft-wrap,
  no line numbers, vim keys for the muscle memory but arrows for the brain on
  bad days. *(this is what marg is)*
- A tiny CLI that reads my journal and gives me one sentence of feedback per
  week. Not advice, not analysis — just a mirror.
- A timer that knows when I'm in flow and refuses to interrupt me, even when
  I told it to.

## writing

- Essay: "What changed when I stopped editing in Neovim". The thesis: the
  affordances of your editor shape what you write, and code-shaped affordances
  produce code-shaped prose.
- Short story: a man wakes up with the ability to undo only the last thing
  he said. Each chapter is one conversation.

## product

- The dictation tool needs a "speak the punctuation" mode. Half my recordings
  end up with no commas because I dictate naturally and forget the model can
  hear me.
EOF

cat > "$TARGET/notes/reading-list.md" <<'EOF'
# reading list

## now

- *The Pragmatic Programmer* — re-read, going slow
- *A Pattern Language* — Christopher Alexander, dipping in and out

## next

- *The Master and His Emissary* — Iain McGilchrist
- *Working in Public* — Nadia Eghbal
- *Seeing Like a State* — James C. Scott

## abandoned (and why)

- *Infinite Jest* — couldn't make it stick. Maybe in five years.
- *Crime and Punishment* — translation was working against me. Try a
  different one next time.
EOF

cat > "$TARGET/notes/long-form.md" <<'EOF'
# A note about thinking on the page

This note is one continuous paragraph on purpose, written without any manual line breaks, because the way most editors handle a paragraph like this on a wide monitor is what got me to build marg in the first place. When you open something like Neovim and the window is two hundred columns across, your eye has to track all the way to the right edge before it can come back, and reading prose like that for any length of time is exhausting in a way you only notice after you stop. A book never asks that of you. A Word document never asks that of you. The reason book typography settled on something like sixty to eighty characters per line a few centuries ago is that human eyes are not very good at finding the start of the next line when the lines are too long. So this note is a test: if you are reading it in marg with `max_width = 80` set in your config, the words above and below this sentence should sit in a comfortable column even if your terminal is enormous, and it should feel like reading rather than scanning. If you are reading it without the cap set, the lines run all the way to the right edge of whatever window you happen to have open, and you will probably find yourself slightly tense without quite knowing why.
EOF

cat > "$TARGET/notes/quick-thoughts.md" <<'EOF'
# quick thoughts

A scratchpad. Don't expect coherence.

- The best teams I've been on all had a shared sense of humour about their
  work. Not jokes *about* the work — jokes that **only made sense** because
  you were doing the work.
- "Make it boring" is underrated advice. Boring software runs for a decade.
  Interesting software gets rewritten every two years.
- I keep noticing that the bugs I dread most are always smaller than I
  feared. The dread is the bug. Fixing it takes ten minutes.
EOF

cat > "$TARGET/projects/sidekick/snippets.md" <<'EOF'
# Useful snippets

## Go: simple HTTP server

```go
package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "hello, world")
	})
	log.Fatal(http.ListenAndServe(":8080", nil))
}
```

## Python: read a JSON file

```python
import json
from pathlib import Path

def load(path: str) -> dict:
    """Read a JSON file and return its parsed contents."""
    with Path(path).open("r", encoding="utf-8") as f:
        return json.load(f)

if __name__ == "__main__":
    print(load("data.json"))
```

## A no-language fence — should auto-detect

```
SELECT id, name, created_at
FROM users
WHERE last_seen_at > NOW() - INTERVAL '7 days'
ORDER BY created_at DESC
LIMIT 100;
```
EOF

cat > "$TARGET/projects/sidekick/launch-checklist.md" <<'EOF'
# Sidekick alpha launch checklist

> Working doc. Updated daily.

## must-have

- [x] streaming latency under 1s on a laptop
- [x] magic link auth flow on web + desktop
- [ ] payments tested end-to-end with a real card
- [ ] crash reporting wired up on web and native
- [ ] privacy policy reviewed

## should-have

- [ ] onboarding tour for first-time users
- [ ] in-app feedback button
- [ ] hotkeys customizable

## nice-to-have

- [ ] dark mode for the marketing site
- [ ] referral codes
EOF

cat > "$TARGET/projects/sidekick/roadmap.md" <<'EOF'
# Sidekick roadmap

## Q2 2026

- alpha launch (this month)
- 100 hand-picked alpha users
- weekly feedback sessions

## Q3 2026

- public beta
- pricing tiers finalized
- team plans

## Q4 2026

- general availability
- mobile app
- API for third-party integrations
EOF

cat > "$TARGET/projects/personal-site/copy-draft.md" <<'EOF'
# personal site — homepage copy

## hero

**Tools that get out of your way.**

Small, focused software for people who care about how their work *feels*.
Dictation that doesn't lag. Markdown editing that doesn't fight you. Code
review that ships you faster.

## about

A one-person studio. The tools shipped here are the tools used here, every
day. If something works for you the same way, that's the win.

## projects

### Sidekick
Speech to text that feels native. *In private alpha.*

### marg
A markdown editor for the terminal. *Open source.*
EOF

echo "marg demo vault built at $TARGET"
