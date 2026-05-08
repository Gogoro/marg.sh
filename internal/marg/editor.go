package marg

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// mode is a vim-style modal indicator.
type mode int

const (
	modeNormal mode = iota
	modeInsert
	modeCommand
	modeVisual
	modeVisualLine
	modeSearch
)

// register holds the most recently yanked or deleted text. lineWise indicates
// whether the source was line-oriented (dd/yy/V…d) and changes paste behavior.
type register struct {
	text     string
	lineWise bool
}

// editor owns the buffer being edited, the cursor, the viewport, and key
// handling for both normal and insert modes.
type editor struct {
	filepath string // absolute path; "" means an unsaved buffer
	buf      *buffer

	mode  mode
	dirty bool

	// Logical cursor position into the buffer (rune index for col).
	row int
	col int
	// preferredCol remembers the column the user last set explicitly with
	// a horizontal motion; vertical moves try to land at this column.
	preferredCol int

	// scroll is the index of the first visual line currently displayed.
	scroll int

	// width and height of the editor viewport.
	width  int
	height int

	// maxWidth optionally caps the wrap width below the terminal width.
	// 0 means "no cap, use terminal width".
	maxWidth int

	// codeMaxWidth optionally caps the wrap width for fenced code blocks
	// and table rows. 0 means "use the full available width to the right
	// of the prose left margin", so code/tables get more room than prose.
	codeMaxWidth int

	// centerAbove is the terminal width at which the text block starts
	// being horizontally centered. 0 disables centering.
	centerAbove int

	// command-mode line state (`:` prefix).
	cmdInput string

	// search state. searchInput is the in-flight query while typing in
	// modeSearch; lastSearch is the last committed query that `n` / `N`
	// repeat against.
	searchInput string
	lastSearch  string

	// pendingKey holds a buffered prefix for two-key motions like `gg`,
	// `dd`, `yy`.
	pendingKey string

	// reg is the unnamed yank/delete register.
	reg register

	// Selection anchor for visual modes (the position v/V was pressed at).
	anchorRow int
	anchorCol int

	// Undo / redo history. history[historyIdx] is the current state.
	history    []snapshot
	historyIdx int

	// insertSnapshot is the state captured when the user entered insert mode.
	// It's compared against the buffer on exit so a no-op insert session
	// doesn't add a redundant history entry.
	insertSnapshot snapshot

	// Flags inspected by the root model after Update.
	openTreeRequested bool
	quitRequested     bool

	// transient one-shot status (e.g. "saved").
	flash    string
	flashGen int

	// AI proofreading state. proofItems is the canonical list received
	// from the model — it doesn't move when the user edits. suggestions
	// is the live, anchored view of that list against the current buffer
	// — refreshed on every update() so the boxes track edits (insert,
	// delete, paste). A proofItem with dismissed=true is hidden from
	// anchoring without losing the underlying raw, so accept/reject
	// don't have to surgically remove entries from the list.
	// proofRunning is true while a `:proof` request is in flight.
	proofItems   []proofItem
	suggestions  []suggestion
	proofRunning bool

	// ai holds the model + auth config copied from the global Config when
	// the editor was opened. The editor reads from it on `:proof` so users
	// can swap models or rotate keys by editing config.toml without code
	// changes.
	ai AIConfig
}

// flashTickMsg is delivered by a tea.Tick a couple of seconds after a flash
// was set, so the status bar can clear itself even if the user doesn't press
// another key.
type flashTickMsg struct{ gen int }

func newEditor(path string) editor {
	e := editor{filepath: path, buf: newBuffer(), mode: modeNormal}
	e.initHistory()
	return e
}

func loadEditor(path string) (editor, error) {
	buf, err := loadBufferFromFile(path)
	if err != nil {
		return editor{}, err
	}
	e := editor{filepath: path, buf: buf, mode: modeNormal}
	e.initHistory()
	return e, nil
}

func (e *editor) resize(w, h int) {
	e.width = w
	e.height = h
	e.clampScroll()
}

// --- key handling ---

func (e editor) update(msg tea.KeyMsg) (editor, tea.Cmd) {
	e.flash = ""
	// Refresh suggestion anchors against the live buffer before any
	// handler reads them — this is what makes the proof boxes track the
	// user's edits (delete, insert, paste). Cheap: anchoring scans lines
	// for each item's `original` text, and the item list is small.
	if len(e.proofItems) > 0 {
		e.suggestions = anchorSuggestions(e.buf, e.proofItems)
	}
	var (
		next editor
		cmd  tea.Cmd
	)
	switch e.mode {
	case modeNormal:
		next, cmd = e.updateNormal(msg)
	case modeInsert:
		next, cmd = e.updateInsert(msg)
	case modeCommand:
		next, cmd = e.updateCommand(msg)
	case modeVisual, modeVisualLine:
		next, cmd = e.updateVisual(msg)
	case modeSearch:
		next, cmd = e.updateSearch(msg)
	default:
		return e, nil
	}
	// And re-anchor again on the post-handler buffer so the upcoming
	// view() renders boxes against the latest edits made in this tick
	// (insert mode keystrokes, paste, delete, accept replacements).
	if len(next.proofItems) > 0 {
		next.suggestions = anchorSuggestions(next.buf, next.proofItems)
	}
	if next.flash != "" {
		next.flashGen++
		gen := next.flashGen
		clear := tea.Tick(2*time.Second, func(time.Time) tea.Msg {
			return flashTickMsg{gen: gen}
		})
		if cmd != nil {
			cmd = tea.Batch(cmd, clear)
		} else {
			cmd = clear
		}
	}
	return next, cmd
}

// onFlashTick is called by the root model with the gen of an expiring
// timer. If the editor's gen still matches, the flash is cleared.
func (e editor) onFlashTick(gen int) editor {
	if e.flashGen == gen {
		e.flash = ""
	}
	return e
}

func (e editor) updateNormal(msg tea.KeyMsg) (editor, tea.Cmd) {
	key := msg.String()

	// Two-key sequences first.
	if e.pendingKey != "" {
		pending := e.pendingKey
		e.pendingKey = ""
		switch {
		case pending == "g" && key == "g":
			e.row = 0
			e.col = 0
			e.preferredCol = 0
			e.clampCursor()
			e.scrollToCursor()
			return e, nil
		case pending == "d" && key == "d":
			e.yankCurrentLine()
			e.deleteCurrentLine()
			e.dirty = true
			e.recordChange()
			e.clampCursor()
			e.scrollToCursor()
			return e, nil
		case pending == "y" && key == "y":
			e.yankCurrentLine()
			e.flash = "1 line yanked"
			return e, nil
		case pending == "c" && key == "w":
			line := e.buf.line(e.row)
			if e.col >= len(line) {
				return e, nil
			}
			end := e.col
			// If on a word break, skip past whitespace first (matches `dw`/`cw`
			// when the cursor sits on a space).
			if isWordBreak(line[end]) {
				for end < len(line) && isWordBreak(line[end]) {
					end++
				}
			}
			// Then skip across the word itself, stopping at the next break.
			// Unlike `dw`, `cw` does NOT include trailing whitespace — that's
			// the long-standing vim convention.
			for end < len(line) && !isWordBreak(line[end]) {
				end++
			}
			if end <= e.col {
				return e, nil
			}
			// Snapshot must be taken BEFORE the deletion so a later esc
			// records one undo step covering the whole `cw + typed text`.
			e.insertSnapshot = e.captureSnapshot()
			e.reg = register{text: e.buf.textRange(e.row, e.col, e.row, end-1), lineWise: false}
			e.row, e.col = e.buf.deleteRange(e.row, e.col, e.row, end-1)
			e.dirty = true
			e.mode = modeInsert
			return e, nil
		case pending == "r":
			if key == "esc" {
				return e, nil
			}
			if len(msg.Runes) == 1 && e.col < e.buf.lineLen(e.row) {
				e.buf.lines[e.row][e.col] = msg.Runes[0]
				e.dirty = true
				e.recordChange()
			}
			return e, nil
		case pending == "]" && key == "s":
			e.jumpToNextSuggestion()
			e.clampCursor()
			e.scrollToCursor()
			return e, nil
		case pending == "[" && key == "s":
			e.jumpToPrevSuggestion()
			e.clampCursor()
			e.scrollToCursor()
			return e, nil
		case pending == "g" && key == "A":
			e.acceptSuggestionAtCursor()
			return e, nil
		case pending == "g" && key == "X":
			e.rejectSuggestionAtCursor()
			return e, nil
		case pending == "g" && key == "d":
			return e, e.followLinkAtCursor()
		}
		// pending didn't match — fall through and handle key normally.
	}

	switch key {
	case "esc":
		// Discard any half-typed sequence so it doesn't lurk waiting for
		// the next key.
		e.pendingKey = ""
		e.flash = ""
	// motions
	case "h", "left":
		e.moveLeft()
	case "l", "right":
		e.moveRight()
	case "j", "down":
		e.moveVisualDown()
	case "k", "up":
		e.moveVisualUp()
	case "0", "home":
		e.col = 0
		e.preferredCol = 0
	case "_", "^":
		e.col = firstNonBlank(e.buf.line(e.row))
		e.preferredCol = e.col
	case "$", "end":
		e.col = e.visualLineEnd()
		e.preferredCol = e.col
	case "g":
		e.pendingKey = "g"
	case "]":
		e.pendingKey = "]"
	case "[":
		e.pendingKey = "["
	case "G":
		e.row = e.buf.lineCount() - 1
		e.col = 0
		e.preferredCol = 0
	case "w":
		e.moveWordForward()
	case "b":
		e.moveWordBackward()
	case "ctrl+d", "pgdown":
		e.jumpVisualLines(e.halfPage())
	case "ctrl+u", "pgup":
		e.jumpVisualLines(-e.halfPage())
	case "ctrl+f":
		e.jumpVisualLines(e.height)
	case "ctrl+b":
		e.jumpVisualLines(-e.height)

	// mode switches
	case "i":
		e.enterInsertMode()
	case "a":
		e.col = min(e.col+1, e.buf.lineLen(e.row))
		e.enterInsertMode()
	case "I":
		e.col = firstNonBlank(e.buf.line(e.row))
		e.enterInsertMode()
	case "A":
		e.col = e.buf.lineLen(e.row)
		e.enterInsertMode()
	case "o":
		e.insertSnapshot = e.captureSnapshot()
		e.buf.insertLineBelow(e.row)
		e.row++
		e.col = 0
		e.mode = modeInsert
		e.dirty = true
	case "O":
		e.insertSnapshot = e.captureSnapshot()
		e.buf.insertLineAbove(e.row)
		e.col = 0
		e.mode = modeInsert
		e.dirty = true

	// edits
	case "x":
		e.row, e.col = e.buf.deleteRuneAt(e.row, e.col)
		e.dirty = true
		e.recordChange()
	case "r":
		e.pendingKey = "r"
	case "d":
		e.pendingKey = "d"
	case "c":
		e.pendingKey = "c"
	case "y":
		e.pendingKey = "y"
	case "Y":
		// Vim convention: Y yanks the current line (same as yy).
		e.yankCurrentLine()
		e.flash = "1 line yanked"
	case "p":
		e.pasteAfterCursor()
		e.recordChange()
	case "P":
		e.pasteBeforeCursor()
		e.recordChange()
	case "u":
		if e.undo() {
			e.flash = "undo"
		} else {
			e.flash = "already at oldest change"
		}
	case "ctrl+r":
		if e.redo() {
			e.flash = "redo"
		} else {
			e.flash = "already at newest change"
		}
	case "v":
		e.mode = modeVisual
		e.anchorRow = e.row
		e.anchorCol = e.col
	case "V":
		e.mode = modeVisualLine
		e.anchorRow = e.row
		e.anchorCol = e.col

	// command line
	case ":":
		e.mode = modeCommand
		e.cmdInput = ""

	// search
	case "/":
		e.mode = modeSearch
		e.searchInput = ""
	case "n":
		if e.lastSearch == "" {
			e.flash = "no recent search"
		} else if r, c, ok, wrapped := e.findNextMatch(e.row, e.col, e.lastSearch, true); ok {
			e.row, e.col = r, c
			e.preferredCol = c
			if wrapped {
				e.flash = "search hit BOTTOM, continuing at TOP"
			}
		} else {
			e.flash = "pattern not found: " + e.lastSearch
		}
	case "N":
		if e.lastSearch == "" {
			e.flash = "no recent search"
		} else if r, c, ok, wrapped := e.findNextMatch(e.row, e.col, e.lastSearch, false); ok {
			e.row, e.col = r, c
			e.preferredCol = c
			if wrapped {
				e.flash = "search hit TOP, continuing at BOTTOM"
			}
		} else {
			e.flash = "pattern not found: " + e.lastSearch
		}

	// save
	case "ctrl+s":
		if cmd := e.save(); cmd != nil {
			return e, cmd
		}

	// open file tree
	case "ctrl+e":
		e.openTreeRequested = true
	}

	e.clampCursor()
	e.scrollToCursor()
	return e, nil
}

func (e editor) updateInsert(msg tea.KeyMsg) (editor, tea.Cmd) {
	key := msg.String()
	switch key {
	case "esc":
		e.mode = modeNormal
		// Vim convention: stepping out of insert moves cursor left by one.
		if e.col > 0 {
			e.col--
		}
		e.preferredCol = e.col
		// Record an undo entry for the whole insert session if it changed
		// the buffer.
		if !snapshotsEqual(e.insertSnapshot, e.captureSnapshot()) {
			e.recordChange()
		}
	case "left":
		e.moveLeft()
	case "right":
		e.moveRight()
	case "up":
		e.moveVisualUp()
	case "down":
		e.moveVisualDown()
	case "pgup":
		e.jumpVisualLines(-e.halfPage())
	case "pgdown":
		e.jumpVisualLines(e.halfPage())
	case "home":
		e.col = 0
	case "end":
		e.col = e.visualLineEnd() + 1
		if e.col > e.buf.lineLen(e.row) {
			e.col = e.buf.lineLen(e.row)
		}
	case "backspace":
		e.row, e.col = e.buf.deleteRuneBefore(e.row, e.col)
		e.dirty = true
	case "delete":
		e.row, e.col = e.buf.deleteRuneAt(e.row, e.col)
		e.dirty = true
	case "enter":
		// Markdown list continuation: pressing Enter at the end of a
		// list item carries the bullet (or auto-incremented number) onto
		// the new line. An empty list item exits the list cleanly.
		line := e.buf.line(e.row)
		if info, ok := parseListLine(line); ok && e.col == len(line) {
			rest := strings.TrimSpace(string(line[info.prefixRunes:]))
			if rest == "" {
				e.buf.lines[e.row] = []rune{}
				e.buf.insertNewline(e.row, 0)
				e.row++
				e.col = 0
				e.dirty = true
				break
			}
			e.buf.insertNewline(e.row, e.col)
			e.row++
			e.col = 0
			for _, r := range info.nextPrefix {
				e.buf.insertRune(e.row, e.col, r)
				e.col++
			}
			e.dirty = true
			break
		}
		e.buf.insertNewline(e.row, e.col)
		e.row++
		e.col = 0
		e.dirty = true
	case "tab":
		// insert two spaces — markdown convention
		e.buf.insertRune(e.row, e.col, ' ')
		e.col++
		e.buf.insertRune(e.row, e.col, ' ')
		e.col++
		e.dirty = true
	case "ctrl+s":
		if cmd := e.save(); cmd != nil {
			return e, cmd
		}
	default:
		// Plain text input. Bubble Tea reports printable runes via Runes.
		if len(msg.Runes) > 0 {
			for _, r := range msg.Runes {
				e.buf.insertRune(e.row, e.col, r)
				e.col++
			}
			e.dirty = true
		}
	}
	e.clampCursor()
	e.scrollToCursor()
	return e, nil
}

func (e editor) updateVisual(msg tea.KeyMsg) (editor, tea.Cmd) {
	key := msg.String()

	// gg in visual mode keeps the selection extending to the top.
	if e.pendingKey == "g" {
		e.pendingKey = ""
		if key == "g" {
			e.row = 0
			e.col = 0
			e.clampCursor()
			e.scrollToCursor()
			return e, nil
		}
	}

	switch key {
	case "esc":
		e.mode = modeNormal
	case "v":
		if e.mode == modeVisual {
			e.mode = modeNormal
		} else {
			e.mode = modeVisual
		}
	case "V":
		if e.mode == modeVisualLine {
			e.mode = modeNormal
		} else {
			e.mode = modeVisualLine
		}

	// motions reuse normal-mode helpers
	case "h", "left":
		e.moveLeft()
	case "l", "right":
		e.moveRight()
	case "j", "down":
		e.moveVisualDown()
	case "k", "up":
		e.moveVisualUp()
	case "0", "home":
		e.col = 0
		e.preferredCol = 0
	case "^":
		e.col = firstNonBlank(e.buf.line(e.row))
		e.preferredCol = e.col
	case "$", "end":
		e.col = e.visualLineEnd()
		e.preferredCol = e.col
	case "w":
		e.moveWordForward()
	case "b":
		e.moveWordBackward()
	case "g":
		e.pendingKey = "g"
	case "G":
		e.row = e.buf.lineCount() - 1
		e.col = 0
		e.preferredCol = 0
	case "ctrl+d", "pgdown":
		e.jumpVisualLines(e.halfPage())
	case "ctrl+u", "pgup":
		e.jumpVisualLines(-e.halfPage())
	case "ctrl+f":
		e.jumpVisualLines(e.height)
	case "ctrl+b":
		e.jumpVisualLines(-e.height)

	// operators end the visual mode
	case "y":
		e.yankSelection()
		e.mode = modeNormal
		e.flash = "yanked"
	case "d", "x":
		e.yankSelection()
		e.deleteSelection()
		e.dirty = true
		e.mode = modeNormal
		e.recordChange()
	case "c":
		e.insertSnapshot = e.captureSnapshot()
		e.yankSelection()
		e.deleteSelection()
		e.dirty = true
		e.mode = modeInsert
	case "p":
		// Replace selection with register contents.
		e.deleteSelection()
		e.pasteHere()
		e.dirty = true
		e.mode = modeNormal
		e.recordChange()
	case "*":
		e.wrapSelection("**")
	case "`":
		e.wrapSelection("`")
	case "_":
		e.wrapSelection("_")
	}

	e.clampCursor()
	e.scrollToCursor()
	return e, nil
}

func (e editor) updateCommand(msg tea.KeyMsg) (editor, tea.Cmd) {
	key := msg.String()
	switch key {
	case "esc":
		e.mode = modeNormal
		e.cmdInput = ""
	case "enter":
		cmd := e.runCommand(e.cmdInput)
		e.cmdInput = ""
		e.mode = modeNormal
		return e, cmd
	case "backspace":
		if len(e.cmdInput) > 0 {
			e.cmdInput = e.cmdInput[:len(e.cmdInput)-1]
		} else {
			e.mode = modeNormal
		}
	default:
		if len(msg.Runes) > 0 {
			e.cmdInput += string(msg.Runes)
		}
	}
	return e, nil
}

func (e *editor) runCommand(cmd string) tea.Cmd {
	cmd = strings.TrimSpace(cmd)
	switch cmd {
	case "w":
		return e.save()
	case "q":
		if e.dirty {
			e.flash = "unsaved changes — :q! to force"
			return nil
		}
		e.quitRequested = true
		return func() tea.Msg { return quitMsg{} }
	case "q!":
		e.quitRequested = true
		return func() tea.Msg { return quitMsg{} }
	case "wq", "x":
		// save() writes synchronously; its returned status message is
		// discarded because we're quitting anyway.
		_ = e.save()
		e.quitRequested = true
		return func() tea.Msg { return quitMsg{} }
	}
	if strings.HasPrefix(cmd, "Ex") || cmd == "E" {
		e.openTreeRequested = true
		return nil
	}
	if len(cmd) == 2 && cmd[0] == 'H' && cmd[1] >= '0' && cmd[1] <= '6' {
		e.applyHeadingLevel(int(cmd[1] - '0'))
		return nil
	}
	if strings.HasPrefix(cmd, "s/") {
		e.runSubstitute(cmd[2:], e.row, e.row)
		return nil
	}
	if cmd == "e" {
		e.reloadFromDisk(false)
		return nil
	}
	if cmd == "e!" {
		e.reloadFromDisk(true)
		return nil
	}
	if cmd == "zen" {
		return func() tea.Msg { return zenToggleMsg{} }
	}
	if strings.HasPrefix(cmd, "%s/") {
		e.runSubstitute(cmd[3:], 0, e.buf.lineCount()-1)
		return nil
	}
	if strings.HasPrefix(cmd, "rename ") || cmd == "rename" {
		e.runRename(strings.TrimSpace(strings.TrimPrefix(cmd, "rename")))
		return nil
	}
	if cmd == "proof" {
		return e.runProofread()
	}
	if cmd == "proof!" {
		e.suggestions = nil
		e.flash = "suggestions cleared"
		return nil
	}
	e.flash = "unknown: :" + cmd
	return nil
}

// runRename renames the open file on disk and updates the buffer's path.
// A bare name renames in the same directory; a name containing a separator
// is treated as the full destination path. ".md" is appended if no extension.
func (e *editor) runRename(arg string) {
	if e.filepath == "" {
		e.flash = "no file to rename"
		return
	}
	if arg == "" {
		e.flash = "usage: :rename <new-name>"
		return
	}
	dest := arg
	if !strings.ContainsRune(dest, filepath.Separator) {
		dest = filepath.Join(filepath.Dir(e.filepath), dest)
	}
	if filepath.Ext(dest) == "" {
		dest += ".md"
	}
	if dest == e.filepath {
		e.flash = "same path — nothing to do"
		return
	}
	if _, err := os.Stat(dest); err == nil {
		e.flash = "destination exists: " + dest
		return
	}
	if err := os.Rename(e.filepath, dest); err != nil {
		e.flash = "rename failed: " + err.Error()
		return
	}
	e.filepath = dest
	e.flash = "renamed to " + filepath.Base(dest)
}

// runSubstitute handles :s/foo/bar/[g] and :%s/foo/bar/[g]. The argument
// passed in is everything after the leading `s/` or `%s/`, so it looks like
// `foo/bar` or `foo/bar/g`.
func (e *editor) runSubstitute(arg string, startRow, endRow int) {
	parts := strings.SplitN(arg, "/", 3)
	if len(parts) < 2 {
		e.flash = "usage: :s/find/replace[/g]"
		return
	}
	find := parts[0]
	replace := parts[1]
	flags := ""
	if len(parts) == 3 {
		flags = parts[2]
	}
	if find == "" {
		e.flash = "empty pattern"
		return
	}
	count := -1
	if !strings.Contains(flags, "g") {
		count = 1
	}
	changed := 0
	for r := startRow; r <= endRow && r < e.buf.lineCount(); r++ {
		original := string(e.buf.line(r))
		modified := strings.Replace(original, find, replace, count)
		if modified != original {
			e.buf.lines[r] = []rune(modified)
			changed++
		}
	}
	if changed == 0 {
		e.flash = "pattern not found: " + find
		return
	}
	e.dirty = true
	e.clampCursor()
	e.recordChange()
	e.flash = fmt.Sprintf("substituted in %d line(s)", changed)
}

// reloadFromDisk replaces the buffer with the file's current on-disk
// content. Without `force`, it refuses if there are unsaved changes.
func (e *editor) reloadFromDisk(force bool) {
	if e.filepath == "" {
		e.flash = "no file to reload"
		return
	}
	data, err := os.ReadFile(e.filepath)
	if err != nil {
		e.flash = "reload failed: " + err.Error()
		return
	}
	if !force && e.dirty {
		e.flash = "no write since change (use :e! to force)"
		return
	}
	e.buf = bufferFromString(string(data))
	e.dirty = false
	e.clampCursor()
	e.scrollToCursor()
	e.recordChange()
	e.flash = "reloaded"
}

// --- proofread ---

// runProofread fires off an async proofreading pass over the whole document
// and returns a tea.Cmd that delivers a proofResultMsg when the API responds.
// While a request is in flight, further `:proof` calls are no-ops so we
// don't pile up redundant work.
func (e *editor) runProofread() tea.Cmd {
	if e.proofRunning {
		e.flash = "proofread already running"
		return nil
	}
	e.proofRunning = true
	// proofRunning drives the persistent status-bar indicator; no flash
	// needed (and a flash would auto-clear after 2s, which would mislead
	// the user since the API call typically takes 5–30s).
	doc := e.buf.toString()
	model := e.ai.FastModel
	apiKey := e.ai.APIKey
	return func() tea.Msg {
		raws, err := requestProofread(context.Background(), doc, model, apiKey)
		return proofResultMsg{suggestions: raws, err: err}
	}
}

// onProofResult ingests the model's response, builds the canonical
// proofItems list, anchors them against the current buffer, and updates
// the status flash with the count or error. Subsequent edits re-anchor
// from proofItems on each update() so suggestions track the buffer.
func (e editor) onProofResult(m proofResultMsg) editor {
	e.proofRunning = false
	if m.err != nil {
		e.flash = "proof failed: " + m.err.Error()
		return e
	}
	e.proofItems = make([]proofItem, 0, len(m.suggestions))
	for _, r := range m.suggestions {
		e.proofItems = append(e.proofItems, proofItem{raw: r})
	}
	e.suggestions = anchorSuggestions(e.buf, e.proofItems)
	if len(e.suggestions) == 0 {
		e.flash = "no suggestions"
	} else {
		e.flash = fmt.Sprintf("%d suggestion(s) — ]s next, gA accept, gX reject", len(e.suggestions))
	}
	return e
}

// suggestionAtCursor returns a pointer to the suggestion the cursor is
// currently inside, or nil. The cursor counts as "inside" anywhere in the
// half-open span [startCol, endCol).
func (e *editor) suggestionAtCursor() *suggestion {
	for i := range e.suggestions {
		s := &e.suggestions[i]
		if s.row == e.row && e.col >= s.startCol && e.col < s.endCol {
			return s
		}
	}
	return nil
}

// jumpToNextSuggestion moves the cursor to the start of the next suggestion
// after the current cursor position. Wraps around to the top.
func (e *editor) jumpToNextSuggestion() {
	if len(e.suggestions) == 0 {
		e.flash = "no suggestions"
		return
	}
	for _, s := range e.suggestions {
		if s.row > e.row || (s.row == e.row && s.startCol > e.col) {
			e.row, e.col = s.row, s.startCol
			e.preferredCol = e.col
			return
		}
	}
	first := e.suggestions[0]
	e.row, e.col = first.row, first.startCol
	e.preferredCol = e.col
	e.flash = "wrapped to first suggestion"
}

// jumpToPrevSuggestion is the mirror of jumpToNextSuggestion.
func (e *editor) jumpToPrevSuggestion() {
	if len(e.suggestions) == 0 {
		e.flash = "no suggestions"
		return
	}
	for i := len(e.suggestions) - 1; i >= 0; i-- {
		s := e.suggestions[i]
		if s.row < e.row || (s.row == e.row && s.startCol < e.col) {
			e.row, e.col = s.row, s.startCol
			e.preferredCol = e.col
			return
		}
	}
	last := e.suggestions[len(e.suggestions)-1]
	e.row, e.col = last.row, last.startCol
	e.preferredCol = e.col
	e.flash = "wrapped to last suggestion"
}

// acceptSuggestionAtCursor replaces the cursor's suggestion with the
// model's proposed text and dismisses the corresponding proofItem.
// All other suggestions are re-anchored against the new buffer state so
// any that shifted (positions on the same line, suggestions on later
// lines after a multi-char replacement) get fresh coordinates.
func (e *editor) acceptSuggestionAtCursor() {
	s := e.suggestionAtCursor()
	if s == nil {
		e.flash = "no suggestion at cursor"
		return
	}
	if s.replacement == "" {
		e.flash = "advisory only — no replacement to accept"
		return
	}
	row, startCol, endCol := s.row, s.startCol, s.endCol
	itemIdx := s.itemIdx
	replacement := s.replacement

	e.buf.deleteRange(row, startCol, row, endCol-1)
	e.row, e.col = e.buf.insertText(row, startCol, replacement)
	e.dirty = true
	e.recordChange()
	if itemIdx >= 0 && itemIdx < len(e.proofItems) {
		e.proofItems[itemIdx].dismissed = true
	}
	e.suggestions = anchorSuggestions(e.buf, e.proofItems)
	e.flash = "accepted"
	e.clampCursor()
	e.scrollToCursor()
}

// rejectSuggestionAtCursor dismisses the cursor's suggestion without
// changing the buffer. The item stays in proofItems flagged dismissed
// so anchoring skips it on every refresh.
func (e *editor) rejectSuggestionAtCursor() {
	s := e.suggestionAtCursor()
	if s == nil {
		e.flash = "no suggestion at cursor"
		return
	}
	if s.itemIdx >= 0 && s.itemIdx < len(e.proofItems) {
		e.proofItems[s.itemIdx].dismissed = true
	}
	e.suggestions = anchorSuggestions(e.buf, e.proofItems)
	e.flash = "rejected"
}

// --- save ---

func (e *editor) save() tea.Cmd {
	if e.filepath == "" {
		e.flash = "no filename — use :w <path>"
		return nil
	}
	data := []byte(e.buf.toString())
	if err := os.WriteFile(e.filepath, data, 0o644); err != nil {
		msg := "save failed: " + err.Error()
		return func() tea.Msg { return statusMsg(msg) }
	}
	e.dirty = false
	rel := e.filepath
	if cwd, err := os.Getwd(); err == nil {
		if r, err := filepath.Rel(cwd, e.filepath); err == nil {
			rel = r
		}
	}
	msg := fmt.Sprintf("saved %s", rel)
	return func() tea.Msg { return statusMsg(msg) }
}

// --- motions ---

func (e *editor) moveLeft() {
	if e.col > 0 {
		e.col--
	} else if e.row > 0 {
		e.row--
		e.col = e.buf.lineLen(e.row)
	}
	e.preferredCol = e.col
}

func (e *editor) moveRight() {
	if e.col < e.buf.lineLen(e.row) {
		e.col++
	} else if e.row+1 < e.buf.lineCount() {
		e.row++
		e.col = 0
	}
	e.preferredCol = e.col
}

// moveVisualDown moves the cursor one visual line down. Visual lines are
// produced by soft-wrap; this is the prose-friendly behavior.
func (e *editor) moveVisualDown() {
	visuals := e.allVisualLines()
	idx := e.cursorVisualIndex(visuals)
	next := idx + 1
	for next < len(visuals) && visuals[next].synthetic {
		next++
	}
	if next >= len(visuals) {
		return
	}
	target := visuals[next]
	cur := visuals[idx]
	offset := e.col - cur.startCol
	if e.preferredCol > e.col {
		offset = e.preferredCol - cur.startCol
	}
	newCol := target.startCol + offset
	maxCol := target.endCol
	if maxCol > e.buf.lineLen(target.row) {
		maxCol = e.buf.lineLen(target.row)
	}
	if newCol > maxCol {
		newCol = maxCol
	}
	e.row = target.row
	e.col = newCol
}

func (e *editor) moveVisualUp() {
	visuals := e.allVisualLines()
	idx := e.cursorVisualIndex(visuals)
	prev := idx - 1
	for prev >= 0 && visuals[prev].synthetic {
		prev--
	}
	if prev < 0 {
		return
	}
	target := visuals[prev]
	cur := visuals[idx]
	offset := e.col - cur.startCol
	if e.preferredCol > e.col {
		offset = e.preferredCol - cur.startCol
	}
	newCol := target.startCol + offset
	maxCol := target.endCol
	if maxCol > e.buf.lineLen(target.row) {
		maxCol = e.buf.lineLen(target.row)
	}
	if newCol > maxCol {
		newCol = maxCol
	}
	e.row = target.row
	e.col = newCol
}

// jumpVisualLines moves the cursor by n visual lines (positive = down).
// Tries to land at the same column offset within the target visual line.
// After moving, scrolls so the cursor stays roughly mid-screen.
func (e *editor) jumpVisualLines(n int) {
	if n == 0 {
		return
	}
	visuals := e.allVisualLines()
	if len(visuals) == 0 {
		return
	}
	idx := e.cursorVisualIndex(visuals)
	target := idx + n
	if target < 0 {
		target = 0
	}
	if target >= len(visuals) {
		target = len(visuals) - 1
	}
	dir := 1
	if n < 0 {
		dir = -1
	}
	for target >= 0 && target < len(visuals) && visuals[target].synthetic {
		target += dir
	}
	if target < 0 || target >= len(visuals) {
		// walked off the end stepping past synthetic — bounce back the other way
		dir = -dir
		target = idx + n
		if target < 0 {
			target = 0
		}
		if target >= len(visuals) {
			target = len(visuals) - 1
		}
		for target >= 0 && target < len(visuals) && visuals[target].synthetic {
			target += dir
		}
	}
	if target < 0 || target >= len(visuals) {
		return
	}

	cur := visuals[idx]
	dest := visuals[target]
	offset := e.col - cur.startCol
	if e.preferredCol > e.col {
		offset = e.preferredCol - cur.startCol
	}
	newCol := dest.startCol + offset
	maxCol := dest.endCol
	if maxCol > e.buf.lineLen(dest.row) {
		maxCol = e.buf.lineLen(dest.row)
	}
	if newCol > maxCol {
		newCol = maxCol
	}
	e.row = dest.row
	e.col = newCol

	// Center the cursor vertically — gives a real "jump" feel rather than
	// the cursor sticking to an edge.
	e.scroll = target - e.height/2
	if e.scroll < 0 {
		e.scroll = 0
	}
	maxScroll := len(visuals) - e.height
	if maxScroll < 0 {
		maxScroll = 0
	}
	if e.scroll > maxScroll {
		e.scroll = maxScroll
	}
}

// visualLineEnd returns the column of the last visible rune on the visual
// line the cursor is currently on. For unwrapped lines this is the same as
// the logical end of line; for wrapped lines it's the end of the segment.
func (e *editor) visualLineEnd() int {
	visuals := e.allVisualLines()
	if len(visuals) == 0 {
		return 0
	}
	v := visuals[e.cursorVisualIndex(visuals)]
	if v.endCol > v.startCol {
		return v.endCol - 1
	}
	return v.startCol
}

func (e *editor) halfPage() int {
	h := e.height / 2
	if h < 1 {
		return 1
	}
	return h
}

func (e *editor) moveWordForward() {
	line := e.buf.line(e.row)
	col := e.col
	// skip current word
	for col < len(line) && !isWordBreak(line[col]) {
		col++
	}
	// skip whitespace
	for col < len(line) && isWordBreak(line[col]) {
		col++
	}
	if col >= len(line) && e.row+1 < e.buf.lineCount() {
		e.row++
		e.col = 0
	} else {
		e.col = col
	}
	e.preferredCol = e.col
}

func (e *editor) moveWordBackward() {
	col := e.col
	if col == 0 {
		if e.row > 0 {
			e.row--
			e.col = e.buf.lineLen(e.row)
		}
		e.preferredCol = e.col
		return
	}
	line := e.buf.line(e.row)
	col--
	for col > 0 && isWordBreak(line[col]) {
		col--
	}
	for col > 0 && !isWordBreak(line[col-1]) {
		col--
	}
	e.col = col
	e.preferredCol = e.col
}

func (e editor) updateSearch(msg tea.KeyMsg) (editor, tea.Cmd) {
	switch msg.String() {
	case "esc":
		e.mode = modeNormal
		e.searchInput = ""
	case "enter":
		e.lastSearch = e.searchInput
		e.searchInput = ""
		e.mode = modeNormal
		if e.lastSearch != "" {
			if r, c, ok, wrapped := e.findNextMatch(e.row, e.col, e.lastSearch, true); ok {
				e.row, e.col = r, c
				e.preferredCol = c
				if wrapped {
					e.flash = "search hit BOTTOM, continuing at TOP"
				}
			} else {
				e.flash = "pattern not found: " + e.lastSearch
			}
		}
	case "backspace":
		if len(e.searchInput) > 0 {
			e.searchInput = e.searchInput[:len(e.searchInput)-1]
		} else {
			e.mode = modeNormal
		}
	default:
		if len(msg.Runes) > 0 {
			e.searchInput += string(msg.Runes)
		}
	}
	e.clampCursor()
	e.scrollToCursor()
	return e, nil
}

// enterInsertMode switches to insert and remembers the pre-insert state so
// the whole session collapses into a single undo entry on exit.
func (e *editor) enterInsertMode() {
	e.insertSnapshot = e.captureSnapshot()
	e.mode = modeInsert
}

// --- yank / delete / paste ---

func (e *editor) yankCurrentLine() {
	e.reg = register{text: string(e.buf.line(e.row)), lineWise: true}
}

func (e *editor) deleteCurrentLine() {
	if e.buf.lineCount() == 1 {
		e.buf.lines[0] = []rune{}
		e.col = 0
		return
	}
	e.buf.deleteLines(e.row, e.row)
	if e.row >= e.buf.lineCount() {
		e.row = e.buf.lineCount() - 1
	}
	e.col = 0
}

// selectionRange returns the normalized inclusive selection (sr, sc, er, ec).
// For visual line mode, columns span the full lines.
func (e *editor) selectionRange() (int, int, int, int) {
	sr, sc, er, ec := e.anchorRow, e.anchorCol, e.row, e.col
	if e.mode == modeVisualLine {
		if sr > er {
			sr, er = er, sr
		}
		return sr, 0, er, e.buf.lineLen(er) - 1
	}
	sr, sc, er, ec = normalizeRange(sr, sc, er, ec)
	return sr, sc, er, ec
}

func (e *editor) yankSelection() {
	if e.mode != modeVisual && e.mode != modeVisualLine {
		return
	}
	sr, sc, er, ec := e.selectionRange()
	if e.mode == modeVisualLine {
		text := ""
		for r := sr; r <= er; r++ {
			if r > sr {
				text += "\n"
			}
			text += string(e.buf.line(r))
		}
		e.reg = register{text: text, lineWise: true}
		return
	}
	e.reg = register{text: e.buf.textRange(sr, sc, er, ec), lineWise: false}
}

func (e *editor) deleteSelection() {
	if e.mode != modeVisual && e.mode != modeVisualLine {
		return
	}
	sr, sc, er, ec := e.selectionRange()
	if e.mode == modeVisualLine {
		e.buf.deleteLines(sr, er)
		e.row = sr
		if e.row >= e.buf.lineCount() {
			e.row = e.buf.lineCount() - 1
		}
		e.col = 0
		return
	}
	e.row, e.col = e.buf.deleteRange(sr, sc, er, ec)
}

// pasteAfterCursor implements `p`: line-wise inserts a new line below the
// current one; char-wise inserts after the cursor.
func (e *editor) pasteAfterCursor() {
	if e.reg.text == "" {
		return
	}
	if e.reg.lineWise {
		row := e.buf.insertLinesBelow(e.row, e.reg.text)
		e.row = row
		e.col = 0
	} else {
		col := e.col + 1
		if col > e.buf.lineLen(e.row) {
			col = e.buf.lineLen(e.row)
		}
		// Empty line: insert at col 0 instead.
		if e.buf.lineLen(e.row) == 0 {
			col = 0
		}
		nr, nc := e.buf.insertText(e.row, col, e.reg.text)
		e.row = nr
		e.col = nc - 1
		if e.col < 0 {
			e.col = 0
		}
	}
	e.dirty = true
}

// pasteBeforeCursor implements `P`: line-wise above; char-wise at cursor.
func (e *editor) pasteBeforeCursor() {
	if e.reg.text == "" {
		return
	}
	if e.reg.lineWise {
		row := e.buf.insertLinesAbove(e.row, e.reg.text)
		e.row = row
		e.col = 0
	} else {
		nr, nc := e.buf.insertText(e.row, e.col, e.reg.text)
		e.row = nr
		e.col = nc - 1
		if e.col < 0 {
			e.col = 0
		}
	}
	e.dirty = true
}

// pasteHere is used in visual mode after the selection has been deleted.
// Cursor sits at the start of where the selection was.
func (e *editor) pasteHere() {
	if e.reg.text == "" {
		return
	}
	if e.reg.lineWise {
		// Inserting line-wise content where a char selection was: drop a
		// blank line at the cursor and put the lines in.
		row := e.buf.insertLinesAbove(e.row, e.reg.text)
		e.row = row
		e.col = 0
		return
	}
	nr, nc := e.buf.insertText(e.row, e.col, e.reg.text)
	e.row = nr
	if nc > 0 {
		e.col = nc - 1
	}
}

// wrapSelection brackets the current visual-mode selection with `marker`
// on each side (e.g. `**` for bold, `` ` `` for inline code). Visual-line
// mode wraps the joined block but doesn't try to wrap each line individually.
func (e *editor) wrapSelection(marker string) {
	if e.mode != modeVisual && e.mode != modeVisualLine {
		return
	}
	sr, sc, er, ec := e.selectionRange()
	if e.mode == modeVisualLine {
		// Prepend the marker to the first line; append to the last line.
		mr := []rune(marker)
		first := append([]rune{}, mr...)
		first = append(first, e.buf.lines[sr]...)
		e.buf.lines[sr] = first
		e.buf.lines[er] = append(e.buf.lines[er], mr...)
		e.row = er
		e.col = e.buf.lineLen(er) - 1
		if e.col < 0 {
			e.col = 0
		}
	} else {
		text := e.buf.textRange(sr, sc, er, ec)
		nr, nc := e.buf.deleteRange(sr, sc, er, ec)
		nr2, nc2 := e.buf.insertText(nr, nc, marker+text+marker)
		e.row = nr2
		e.col = nc2 - 1
		if e.col < 0 {
			e.col = 0
		}
	}
	e.dirty = true
	e.mode = modeNormal
	e.recordChange()
}

// isSelected returns true if the rune at (row, col) is part of the current
// visual-mode selection. Always false outside visual modes.
func (e *editor) isSelected(row, col int) bool {
	if e.mode != modeVisual && e.mode != modeVisualLine {
		return false
	}
	sr, sc, er, ec := e.selectionRange()
	if row < sr || row > er {
		return false
	}
	if e.mode == modeVisualLine {
		return true
	}
	if sr == er {
		return col >= sc && col <= ec
	}
	if row == sr {
		return col >= sc
	}
	if row == er {
		return col <= ec
	}
	return true
}

func isWordBreak(r rune) bool {
	return r == ' ' || r == '\t' || r == '.' || r == ',' || r == ';' || r == ':' || r == '!' || r == '?'
}

func firstNonBlank(line []rune) int {
	for i, r := range line {
		if r != ' ' && r != '\t' {
			return i
		}
	}
	return 0
}

func (e *editor) clampCursor() {
	if e.row < 0 {
		e.row = 0
	}
	if e.row >= e.buf.lineCount() {
		e.row = e.buf.lineCount() - 1
	}
	maxCol := e.buf.lineLen(e.row)
	if e.mode != modeInsert && maxCol > 0 {
		// Normal mode: cursor sits on a character, not past end.
		// Allow col == 0 for empty lines.
		if e.col >= maxCol {
			e.col = maxCol - 1
		}
	}
	if e.col < 0 {
		e.col = 0
	}
	if e.col > maxCol {
		e.col = maxCol
	}
}

// --- viewport / soft-wrap ---

// visualLine describes one wrapped segment of a logical buffer line.
//
// hangIndent > 0 means this is a continuation segment of a list item: the
// renderer prepends that many spaces so the wrapped text aligns under the
// item content rather than the bullet glyph.
//
// synthetic = true means this visual line has no source rune content — it's
// padding inserted for typographic rhythm (e.g. above a heading).
type visualLine struct {
	row        int
	startCol   int
	endCol     int // exclusive
	text       []rune
	hangIndent int
	synthetic  bool
}

// wrapWidth is the number of columns available for text content. We subtract
// margins for breathing room and, if the user has set a max_width in config,
// cap there too — useful in wide terminals where prose becomes a horizontal
// blur otherwise.
func (e *editor) wrapWidth() int {
	w := e.width - 2
	if e.maxWidth > 0 && w > e.maxWidth {
		w = e.maxWidth
	}
	if w < 10 {
		return 10
	}
	return w
}

// codeWrapWidth is the wrap width used for fenced code blocks and table rows.
// Code/tables anchor at the same left margin as prose but get to extend
// further to the right — the goal being that prose stays comfortable to
// read while code and tables don't get squished. 0 codeMaxWidth means
// "fill all available room to the right of the left margin".
func (e *editor) codeWrapWidth() int {
	available := e.width - e.leftMargin() - 1
	if e.codeMaxWidth > 0 && available > e.codeMaxWidth {
		available = e.codeMaxWidth
	}
	if available < 10 {
		return 10
	}
	return available
}

// leftMargin is the number of blank columns to the left of the text block.
// Centering only kicks in once the terminal grows past `centerAbove`; below
// that threshold the text stays left-aligned with a single-column gutter.
func (e *editor) leftMargin() int {
	if e.width <= 0 {
		return 1
	}
	if e.centerAbove == 0 || e.width < e.centerAbove {
		return 1
	}
	pad := (e.width - e.wrapWidth()) / 2
	if pad < 1 {
		return 1
	}
	return pad
}

// wrapLine breaks one logical line into visual lines using word-aware wrap.
// For markdown list items, continuation segments carry a hangIndent so the
// wrapped text aligns under the item content rather than the bullet. Code
// block rows and table rows use a wider wrap so they don't get squished by
// the prose max_width.
func (e *editor) wrapLine(row int, code codeBlockSpans) []visualLine {
	line := e.buf.line(row)
	w := e.wrapWidth()
	if code.inCode[row] || isTableRow(line) {
		w = e.codeWrapWidth()
	}

	hang := 0
	if info, ok := parseListLine(line); ok {
		hang = info.prefixRunes
	}

	if len(line) == 0 {
		return []visualLine{{row: row, startCol: 0, endCol: 0}}
	}

	var out []visualLine
	start := 0
	first := true
	for start < len(line) {
		segW := w
		indent := 0
		if !first {
			indent = hang
			segW = w - indent
			if segW < 10 {
				segW = 10
			}
		}
		end := start + segW
		if end >= len(line) {
			out = append(out, visualLine{
				row: row, startCol: start, endCol: len(line),
				text: line[start:], hangIndent: indent,
			})
			break
		}
		brk := -1
		for i := end; i > start; i-- {
			if line[i] == ' ' {
				brk = i
				break
			}
		}
		if brk == -1 {
			out = append(out, visualLine{
				row: row, startCol: start, endCol: end,
				text: line[start:end], hangIndent: indent,
			})
			start = end
			first = false
			continue
		}
		out = append(out, visualLine{
			row: row, startCol: start, endCol: brk,
			text: line[start:brk], hangIndent: indent,
		})
		start = brk + 1
		first = false
	}
	return out
}

func (e *editor) allVisualLines() []visualLine {
	code := e.scanCodeBlocks()
	var out []visualLine
	for r := 0; r < e.buf.lineCount(); r++ {
		if r > 0 && startsAsHeading(e.buf.line(r)) {
			prev := strings.TrimSpace(string(e.buf.line(r - 1)))
			if prev != "" {
				out = append(out, visualLine{row: -1, synthetic: true})
			}
		}
		out = append(out, e.wrapLine(r, code)...)
	}
	return out
}

func startsAsHeading(line []rune) bool {
	s := strings.TrimLeft(string(line), " \t")
	if !strings.HasPrefix(s, "#") {
		return false
	}
	return isHeadingLine(s)
}

func (e *editor) cursorVisualIndex(visuals []visualLine) int {
	for i, v := range visuals {
		if v.row != e.row {
			continue
		}
		if e.col >= v.startCol && e.col <= v.endCol {
			// Prefer the segment that doesn't push cursor past its end,
			// unless it's the last one for the row.
			if e.col == v.endCol && i+1 < len(visuals) && visuals[i+1].row == e.row && visuals[i+1].startCol == v.endCol {
				continue
			}
			return i
		}
	}
	return 0
}

func (e *editor) scrollToCursor() {
	visuals := e.allVisualLines()
	idx := e.cursorVisualIndex(visuals)
	if idx < e.scroll {
		e.scroll = idx
	} else if idx >= e.scroll+e.height {
		e.scroll = idx - e.height + 1
	}
	e.clampScroll()
}

func (e *editor) clampScroll() {
	if e.scroll < 0 {
		e.scroll = 0
	}
}

// --- view ---

func (e *editor) view() string {
	if e.width == 0 || e.height == 0 {
		return ""
	}
	visuals := e.allVisualLines()
	cursorIdx := e.cursorVisualIndex(visuals)
	codeSpans := e.scanCodeBlocks()
	frontmatterEnd := e.scanFrontmatter()

	rows := make([]string, 0, e.height)
	end := e.scroll + e.height
	if end > len(visuals) {
		end = len(visuals)
	}
	boxRows := e.renderProofBox(visuals, e.height)
	pickSidebar := func(row int) string {
		if boxRows == nil {
			return ""
		}
		if row < 0 || row >= len(boxRows) {
			return ""
		}
		return boxRows[row]
	}
	for i := e.scroll; i < end; i++ {
		v := visuals[i]
		isCursorLine := i == cursorIdx
		rows = append(rows, e.renderVisualLine(v, isCursorLine, codeSpans, frontmatterEnd, pickSidebar(i-e.scroll)))
	}
	for len(rows) < e.height {
		rows = append(rows, e.renderEmptyRow(false, pickSidebar(len(rows))))
	}
	return strings.Join(rows, "\n")
}

// scanFrontmatter returns the row index just past the closing `---` of a
// YAML frontmatter block at the very top of the document, or 0 if there is
// no frontmatter. `---` only counts as opening when it's literally line 0;
// otherwise it's a horizontal rule.
func (e *editor) scanFrontmatter() int {
	if e.buf.lineCount() == 0 {
		return 0
	}
	if strings.TrimSpace(string(e.buf.line(0))) != "---" {
		return 0
	}
	for r := 1; r < e.buf.lineCount(); r++ {
		if strings.TrimSpace(string(e.buf.line(r))) == "---" {
			return r + 1
		}
	}
	return 0
}

func (e *editor) renderVisualLine(v visualLine, hasCursor bool, code codeBlockSpans, frontmatterEnd int, sidebar string) string {
	if v.synthetic {
		return e.renderEmptyRow(false, sidebar)
	}

	line := e.buf.line(v.row)
	inCode := code.inCode[v.row]
	codeRowSpans := code.spans[v.row]
	inFrontmatter := v.row < frontmatterEnd

	base := baseStyleForLine(line)
	var inlines []inlineRange
	if inFrontmatter {
		if strings.TrimSpace(string(line)) == "---" {
			base = styleFrontmatterFence
		} else {
			base = styleFrontmatterKey
			inlines = frontmatterValueRanges(line)
		}
	} else if !inCode {
		inlines = inlineRanges(line)
	}
	hiddenCols := e.collapsedSyntaxCols(v.row, line, inFrontmatter || inCode)
	searchMatches := findMatchesInLine(line, e.lastSearch)

	rowBg, paintRow := e.rowBackground(hasCursor)

	cursorRel := -1
	if hasCursor {
		cursorRel = e.col - v.startCol
	}

	cursorStyle := lipgloss.NewStyle().Reverse(true)
	if e.mode == modeInsert {
		cursorStyle = lipgloss.NewStyle().Foreground(colorAccent).Underline(true)
	}

	type run struct {
		style lipgloss.Style
		text  []rune
	}
	var current run
	var b strings.Builder
	flush := func() {
		if len(current.text) > 0 {
			b.WriteString(current.style.Render(string(current.text)))
			current.text = current.text[:0]
		}
	}

	segLen := v.endCol - v.startCol
	hiddenInSeg := 0
	for i := 0; i < segLen; i++ {
		col := v.startCol + i
		// Collapsed-link chars: skip the `[`, `]`, and `(url)` so the link
		// reads as plain styled text. Cursor never lands on these because a
		// link containing the cursor is force-expanded above.
		if hiddenCols[col] {
			hiddenInSeg++
			continue
		}
		var s lipgloss.Style
		if inCode && len(codeRowSpans) > 0 {
			s = styleAtCol(codeRowSpans, col, base)
		} else {
			s = base
			for _, ir := range inlines {
				if col >= ir.start && col < ir.end {
					s = ir.style
					break
				}
			}
		}
		// Proofread suggestions: a subtle underline under the affected span.
		// Layered after inline markdown styling so it composes with bold /
		// italic / code colors rather than replacing them. Selection, search
		// match, and cursor still override below.
		for _, sg := range e.suggestions {
			if sg.row == v.row && col >= sg.startCol && col < sg.endCol {
				s = s.Underline(true)
				break
			}
		}
		if paintRow {
			s = s.Background(rowBg)
		}
		if e.isSelected(v.row, col) {
			s = s.Background(colorSelection)
		}
		if isInRanges(col, searchMatches) {
			s = s.Background(colorMatch).Foreground(colorMatchFg)
		}
		if i == cursorRel {
			s = cursorStyle
		}
		if !sameStyle(s, current.style) {
			flush()
			current.style = s
		}
		current.text = append(current.text, line[col])
	}
	flush()

	contentW := segLen - hiddenInSeg

	if hasCursor && cursorRel == segLen {
		b.WriteString(cursorStyle.Render(" "))
		contentW++
	}

	if e.mode == modeVisualLine && segLen == 0 && e.isSelected(v.row, 0) {
		b.WriteString(lipgloss.NewStyle().Background(colorSelection).Render(" "))
		contentW++
	}

	prefixWidth := e.leftMargin() + v.hangIndent
	prefixSpaces := strings.Repeat(" ", prefixWidth)
	if inCode && e.leftMargin() > 0 {
		railCol := e.leftMargin() - 1
		leftSpaces := strings.Repeat(" ", railCol)
		rightSpaces := strings.Repeat(" ", prefixWidth-railCol-1)
		bgStyle := lipgloss.NewStyle()
		railStyle := lipgloss.NewStyle().Foreground(colorMuted)
		if paintRow {
			bgStyle = bgStyle.Background(rowBg)
			railStyle = railStyle.Background(rowBg)
		}
		prefixSpaces = bgStyle.Render(leftSpaces) + railStyle.Render("▎") + bgStyle.Render(rightSpaces)
	} else if paintRow {
		prefixSpaces = lipgloss.NewStyle().Background(rowBg).Render(prefixSpaces)
	}

	filled := e.leftMargin() + v.hangIndent + contentW
	sidebarW := lipgloss.Width(sidebar)
	// Box anchors at a fixed offset past the prose's wrap edge — not at
	// the far right of the terminal — so on a wide terminal the box sits
	// snug next to the text instead of floating in the void.
	const sidebarGap = 4
	sidebarLeft := e.leftMargin() + e.wrapWidth() + sidebarGap
	innerGap := sidebarLeft - filled
	if innerGap < 0 {
		innerGap = 0
	}
	trailingFill := e.width - sidebarLeft - sidebarW
	if trailingFill < 0 {
		trailingFill = 0
	}
	gap := e.width - filled - sidebarW
	if gap < 0 {
		gap = 0
	}

	tail := ""
	if sidebar != "" {
		// Box mode: bg-paint the gap between prose and sidebar (so the
		// cursor-line highlight extends right up to the box), then append
		// the sidebar row, then bg-pad the trailing region out to the
		// terminal edge so the row reads as a continuous strip.
		inner := strings.Repeat(" ", innerGap)
		trailing := strings.Repeat(" ", trailingFill)
		if paintRow {
			bg := lipgloss.NewStyle().Background(rowBg)
			inner = bg.Render(inner)
			trailing = bg.Render(trailing)
		}
		tail = inner + sidebar + trailing
	} else {
		// No box (terminal too narrow, no suggestions): fall back to the
		// inline right-margin note for whichever suggestion starts here.
		if !inCode {
			if note := e.suggestionNoteForVisualLine(v); note != "" {
				const leadGap = 2
				noteRunes := []rune(note)
				if avail := gap - leadGap; avail > 4 {
					if len(noteRunes) > avail {
						noteRunes = noteRunes[:avail]
					}
					noteStr := string(noteRunes)
					lead := strings.Repeat(" ", leadGap)
					trail := strings.Repeat(" ", avail-len(noteRunes))
					noteStyle := lipgloss.NewStyle().Foreground(colorMuted)
					if paintRow {
						bg := lipgloss.NewStyle().Background(rowBg)
						tail = bg.Render(lead) + noteStyle.Background(rowBg).Render(noteStr) + bg.Render(trail)
					} else {
						tail = lead + noteStyle.Render(noteStr)
					}
				}
			}
		}
		if tail == "" && paintRow && gap > 0 {
			tail = lipgloss.NewStyle().Background(rowBg).Render(strings.Repeat(" ", gap))
		}
	}

	return prefixSpaces + b.String() + tail
}

// collapsedSyntaxCols returns the set of rune-indexed cols on `line` that
// the renderer should skip in reading mode — link brackets + URL, plus
// inline markers like `**`, `*`, `_`, `~~`, `` ` ``. The styled run still
// renders, just without its surrounding syntax. A span is left fully
// expanded (no skipping) when:
//   - we're in insert mode (editing — show the raw markdown)
//   - the cursor is on this row and inside the span
//   - this is a frontmatter or code-block line (no inline rendering applies)
func (e *editor) collapsedSyntaxCols(row int, line []rune, skipAll bool) map[int]bool {
	if skipAll {
		return nil
	}
	out := make(map[int]bool)
	if e.mode != modeInsert {
		for _, m := range inlineMarkerSpans(line) {
			if e.row == row && e.col >= m.start && e.col < m.end {
				continue
			}
			for c := m.start; c < m.start+m.leadLen; c++ {
				out[c] = true
			}
			for c := m.end - m.tailLen; c < m.end; c++ {
				out[c] = true
			}
		}
	}
	for _, lk := range linkSpans(line) {
		expanded := e.mode == modeInsert
		if !expanded && e.row == row && e.col >= lk.start && e.col < lk.end {
			expanded = true
		}
		if expanded {
			continue
		}
		for c := lk.start; c < lk.textStart; c++ {
			out[c] = true
		}
		for c := lk.textEnd; c < lk.end; c++ {
			out[c] = true
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// suggestionNoteForVisualLine picks the suggestion to annotate at the right
// margin of one visual line, if any. A suggestion is shown on the visual
// line where its span starts; long phrases that wrap don't double-annotate.
// `→ replacement` for replacements, `· reason` for advisory notes.
func (e *editor) suggestionNoteForVisualLine(v visualLine) string {
	if v.synthetic {
		return ""
	}
	for _, sg := range e.suggestions {
		if sg.row != v.row {
			continue
		}
		// Annotate on the wrap segment that contains the span's start.
		// Empty-line suggestions (rare) anchor at startCol=0 endCol=0.
		if sg.startCol < v.startCol {
			continue
		}
		if sg.startCol >= v.endCol && !(v.startCol == 0 && v.endCol == 0) {
			continue
		}
		if sg.replacement != "" {
			return "→ " + sg.replacement
		}
		if sg.reason != "" {
			return "· " + sg.reason
		}
	}
	return ""
}

// rowBackground returns the background colour to paint behind the entire
// visual line (including margins and trailing fill). The boolean is false
// when the row should stay transparent (default state on dark theme).
func (e *editor) rowBackground(hasCursor bool) (lipgloss.Color, bool) {
	if hasCursor {
		return colorCursorLine, true
	}
	if active.bg != "" {
		return colorBg, true
	}
	return "", false
}

// renderEmptyRow draws a blank visual line — used for synthetic spacing
// (above headings) and for trailing rows that pad the editor area.
// `sidebar` is the proof-box column for this row; an empty string means
// no box on this row. Sidebar position matches renderVisualLine: fixed
// offset past the prose's wrap edge, not at the terminal's right edge.
func (e *editor) renderEmptyRow(hasCursor bool, sidebar string) string {
	rowBg, paint := e.rowBackground(hasCursor)
	sidebarW := lipgloss.Width(sidebar)
	const sidebarGap = 4
	sidebarLeft := e.leftMargin() + e.wrapWidth() + sidebarGap
	if sidebar != "" {
		leadW := sidebarLeft
		if leadW < 0 {
			leadW = 0
		}
		trailW := e.width - sidebarLeft - sidebarW
		if trailW < 0 {
			trailW = 0
		}
		lead := strings.Repeat(" ", leadW)
		trail := strings.Repeat(" ", trailW)
		if paint {
			bg := lipgloss.NewStyle().Background(rowBg)
			lead = bg.Render(lead)
			trail = bg.Render(trail)
		}
		return lead + sidebar + trail
	}
	if !paint {
		return ""
	}
	return lipgloss.NewStyle().Background(rowBg).Render(strings.Repeat(" ", e.width))
}

func (e *editor) statusBar(width int, transient string) string {
	left := e.statusLeft()
	right := e.statusRight()
	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}

	// Command-mode line replaces the normal status when active.
	if e.mode == modeCommand {
		return styleStatusBar.Render(":" + e.cmdInput + "_")
	}
	if e.mode == modeSearch {
		return styleStatusBar.Render("/" + e.searchInput + "_")
	}
	// Persistent proofread indicator — wins over the regular flash slot
	// because the API call can take 5-30s and the user needs to know it's
	// still running (a 2-second auto-clearing flash would lie to them).
	if e.proofRunning {
		return styleStatusBar.Render(lipgloss.NewStyle().Foreground(colorAccent).Render("proofreading…"))
	}
	if e.flash != "" {
		return styleStatusBar.Render(e.flash)
	}
	if s := e.suggestionAtCursor(); s != nil {
		return styleStatusBar.Render(formatSuggestionReveal(s))
	}
	if transient != "" {
		return styleStatusBar.Render(transient)
	}

	return styleStatusBar.Render(left + strings.Repeat(" ", gap) + right)
}

// formatSuggestionReveal builds the status-line preview of one suggestion.
// The margin annotation already shows `→ replacement` on the same row, so
// the status bar focuses on the reason + the keys you act on it with.
func formatSuggestionReveal(s *suggestion) string {
	reason := s.reason
	if reason == "" {
		reason = "suggestion"
	}
	return fmt.Sprintf("%s  ·  gA accept  gX reject", reason)
}

func (e *editor) statusLeft() string {
	mode := "NORMAL"
	switch e.mode {
	case modeInsert:
		mode = "INSERT"
	case modeCommand:
		mode = "COMMAND"
	case modeVisual:
		mode = "VISUAL"
	case modeVisualLine:
		mode = "V-LINE"
	}

	name := e.filepath
	if name == "" {
		name = "[no name]"
	} else if cwd, err := os.Getwd(); err == nil {
		if r, err := filepath.Rel(cwd, e.filepath); err == nil {
			name = r
		}
	}
	dirty := ""
	if e.dirty {
		dirty = styleStatusDirty.Render(" [+]")
	}
	return styleStatusMode.Render(mode) + "  " + name + dirty
}

func (e *editor) statusRight() string {
	return fmt.Sprintf("%d:%d  ·  %d lines  ·  %d words  ·  %d chars",
		e.row+1, e.col+1,
		e.buf.lineCount(), e.buf.wordCount(), e.buf.charCount())
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
