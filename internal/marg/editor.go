package marg

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

	// command-mode line state (`:` prefix).
	cmdInput string

	// pendingKey holds a buffered prefix for two-key motions like `gg`,
	// `dd`, `yy`.
	pendingKey string

	// reg is the unnamed yank/delete register.
	reg register

	// Selection anchor for visual modes (the position v/V was pressed at).
	anchorRow int
	anchorCol int

	// Flags inspected by the root model after Update.
	openTreeRequested bool
	quitRequested     bool

	// transient one-shot status (e.g. "saved").
	flash string
}

func newEditor(path string) editor {
	return editor{filepath: path, buf: newBuffer(), mode: modeNormal}
}

func loadEditor(path string) (editor, error) {
	buf, err := loadBufferFromFile(path)
	if err != nil {
		return editor{}, err
	}
	return editor{filepath: path, buf: buf, mode: modeNormal}, nil
}

func (e *editor) resize(w, h int) {
	e.width = w
	e.height = h
	e.clampScroll()
}

// --- key handling ---

func (e editor) update(msg tea.KeyMsg) (editor, tea.Cmd) {
	e.flash = ""
	switch e.mode {
	case modeNormal:
		return e.updateNormal(msg)
	case modeInsert:
		return e.updateInsert(msg)
	case modeCommand:
		return e.updateCommand(msg)
	case modeVisual, modeVisualLine:
		return e.updateVisual(msg)
	}
	return e, nil
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
			e.clampCursor()
			e.scrollToCursor()
			return e, nil
		case pending == "y" && key == "y":
			e.yankCurrentLine()
			e.flash = "1 line yanked"
			return e, nil
		}
		// pending didn't match — fall through and handle key normally.
	}

	switch key {
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
		e.mode = modeInsert
	case "a":
		e.col = min(e.col+1, e.buf.lineLen(e.row))
		e.mode = modeInsert
	case "I":
		e.col = firstNonBlank(e.buf.line(e.row))
		e.mode = modeInsert
	case "A":
		e.col = e.buf.lineLen(e.row)
		e.mode = modeInsert
	case "o":
		e.buf.insertLineBelow(e.row)
		e.row++
		e.col = 0
		e.mode = modeInsert
		e.dirty = true
	case "O":
		e.buf.insertLineAbove(e.row)
		e.col = 0
		e.mode = modeInsert
		e.dirty = true

	// edits
	case "x":
		e.row, e.col = e.buf.deleteRuneAt(e.row, e.col)
		e.dirty = true
	case "d":
		e.pendingKey = "d"
	case "y":
		e.pendingKey = "y"
	case "Y":
		// Vim convention: Y yanks the current line (same as yy).
		e.yankCurrentLine()
		e.flash = "1 line yanked"
	case "p":
		e.pasteAfterCursor()
	case "P":
		e.pasteBeforeCursor()
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
	case "_", "^":
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
	case "p":
		// Replace selection with register contents.
		e.deleteSelection()
		e.pasteHere()
		e.dirty = true
		e.mode = modeNormal
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
	e.flash = "unknown: :" + cmd
	return nil
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
	if idx+1 >= len(visuals) {
		return
	}
	target := visuals[idx+1]
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
	if idx <= 0 {
		return
	}
	target := visuals[idx-1]
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
type visualLine struct {
	row      int
	startCol int
	endCol   int // exclusive
	text     []rune
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

// wrapLine breaks one logical line into visual lines using word-aware wrap.
func (e *editor) wrapLine(row int) []visualLine {
	line := e.buf.line(row)
	w := e.wrapWidth()
	if len(line) == 0 {
		return []visualLine{{row: row, startCol: 0, endCol: 0, text: nil}}
	}

	var out []visualLine
	start := 0
	for start < len(line) {
		end := start + w
		if end >= len(line) {
			out = append(out, visualLine{row: row, startCol: start, endCol: len(line), text: line[start:]})
			break
		}
		// Try to break at last space at or before end.
		brk := -1
		for i := end; i > start; i-- {
			if line[i] == ' ' {
				brk = i
				break
			}
		}
		if brk == -1 {
			// No space — hard wrap at width.
			out = append(out, visualLine{row: row, startCol: start, endCol: end, text: line[start:end]})
			start = end
			continue
		}
		out = append(out, visualLine{row: row, startCol: start, endCol: brk, text: line[start:brk]})
		// Skip the space itself when starting next visual line.
		start = brk + 1
	}
	return out
}

func (e *editor) allVisualLines() []visualLine {
	var out []visualLine
	for r := 0; r < e.buf.lineCount(); r++ {
		out = append(out, e.wrapLine(r)...)
	}
	return out
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

	rows := make([]string, 0, e.height)
	end := e.scroll + e.height
	if end > len(visuals) {
		end = len(visuals)
	}
	for i := e.scroll; i < end; i++ {
		v := visuals[i]
		isCursorLine := i == cursorIdx
		rows = append(rows, e.renderVisualLine(v, isCursorLine))
	}
	// Pad to fill height.
	for len(rows) < e.height {
		rows = append(rows, "")
	}
	return strings.Join(rows, "\n")
}

func (e *editor) renderVisualLine(v visualLine, hasCursor bool) string {
	line := e.buf.line(v.row)
	base := baseStyleForLine(line)
	inlines := inlineRanges(line)

	cursorRel := -1
	if hasCursor {
		cursorRel = e.col - v.startCol
	}

	cursorStyle := lipgloss.NewStyle().Reverse(true)
	if e.mode == modeInsert {
		cursorStyle = lipgloss.NewStyle().Foreground(colorAccent).Underline(true)
	}

	// Build the segment by grouping runs of equal effective style.
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
	for i := 0; i < segLen; i++ {
		col := v.startCol + i
		s := base
		for _, ir := range inlines {
			if col >= ir.start && col < ir.end {
				s = ir.style
				break
			}
		}
		if e.isSelected(v.row, col) {
			s = s.Background(colorSelection)
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

	// Cursor sitting past the end of this segment (e.g. end-of-line in insert).
	if hasCursor && cursorRel == segLen {
		b.WriteString(cursorStyle.Render(" "))
	}

	// In visual line mode, give an explicit hint that an empty line is
	// part of the selection by trailing one selected space.
	if e.mode == modeVisualLine && segLen == 0 && e.isSelected(v.row, 0) {
		b.WriteString(lipgloss.NewStyle().Background(colorSelection).Render(" "))
	}

	return " " + b.String()
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
	if e.flash != "" {
		return styleStatusBar.Render(e.flash)
	}
	if transient != "" {
		return styleStatusBar.Render(transient)
	}

	return styleStatusBar.Render(left + strings.Repeat(" ", gap) + right)
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
	return fmt.Sprintf("%d:%d  %d words", e.row+1, e.col+1, e.buf.wordCount())
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
