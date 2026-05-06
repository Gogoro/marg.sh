package marg

// snapshot captures the buffer state and cursor at a point in time so we
// can restore it for undo / redo. Lines are deep-copied so later edits
// don't leak through.
type snapshot struct {
	lines [][]rune
	row   int
	col   int
	dirty bool
}

const historyCap = 200

// captureSnapshot copies the editor's current buffer + cursor into a snapshot.
func (e *editor) captureSnapshot() snapshot {
	lines := make([][]rune, len(e.buf.lines))
	for i, l := range e.buf.lines {
		lines[i] = append([]rune{}, l...)
	}
	return snapshot{lines: lines, row: e.row, col: e.col, dirty: e.dirty}
}

func (e *editor) restoreSnapshot(s snapshot) {
	lines := make([][]rune, len(s.lines))
	for i, l := range s.lines {
		lines[i] = append([]rune{}, l...)
	}
	e.buf.lines = lines
	e.row = s.row
	e.col = s.col
	e.dirty = s.dirty
	e.clampCursor()
	e.scrollToCursor()
}

// recordChange must be called after a meaningful mutation (one normal-mode
// edit, one completed insert session, one substitute, one visual-mode
// operator). It drops any pending redo, appends the new state, and caps
// the stack length.
func (e *editor) recordChange() {
	snap := e.captureSnapshot()
	if len(e.history) > 0 && snapshotsEqual(e.history[e.historyIdx], snap) {
		return
	}
	if e.historyIdx < len(e.history)-1 {
		e.history = e.history[:e.historyIdx+1]
	}
	e.history = append(e.history, snap)
	if len(e.history) > historyCap {
		e.history = e.history[len(e.history)-historyCap:]
	}
	e.historyIdx = len(e.history) - 1
}

// initHistory plants the initial snapshot so the first undo can return to
// the loaded-from-disk state instead of failing silently.
func (e *editor) initHistory() {
	if len(e.history) == 0 {
		e.history = append(e.history, e.captureSnapshot())
		e.historyIdx = 0
	}
}

func (e *editor) undo() bool {
	if e.historyIdx <= 0 {
		return false
	}
	e.historyIdx--
	e.restoreSnapshot(e.history[e.historyIdx])
	return true
}

func (e *editor) redo() bool {
	if e.historyIdx >= len(e.history)-1 {
		return false
	}
	e.historyIdx++
	e.restoreSnapshot(e.history[e.historyIdx])
	return true
}

func snapshotsEqual(a, b snapshot) bool {
	if len(a.lines) != len(b.lines) {
		return false
	}
	for i := range a.lines {
		if len(a.lines[i]) != len(b.lines[i]) {
			return false
		}
		for j := range a.lines[i] {
			if a.lines[i][j] != b.lines[i][j] {
				return false
			}
		}
	}
	return true
}
