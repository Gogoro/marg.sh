package marg

// intRange is an inclusive-exclusive rune index range within a single line.
type intRange struct {
	start int
	end   int
}

func isInRanges(col int, ranges []intRange) bool {
	for _, r := range ranges {
		if col >= r.start && col < r.end {
			return true
		}
	}
	return false
}

// findMatchesInLine returns every position in line where query appears as a
// contiguous rune sequence. Case-sensitive, ASCII-overlap-safe.
func findMatchesInLine(line []rune, query string) []intRange {
	if query == "" {
		return nil
	}
	q := []rune(query)
	if len(q) > len(line) {
		return nil
	}
	var out []intRange
	for i := 0; i+len(q) <= len(line); i++ {
		match := true
		for j, r := range q {
			if line[i+j] != r {
				match = false
				break
			}
		}
		if match {
			out = append(out, intRange{start: i, end: i + len(q)})
			i += len(q) - 1
		}
	}
	return out
}

// findNextMatch walks the buffer from (row, col) and returns the first
// position where query matches, plus a wrapped flag. forward=false runs the
// search backwards. If no match is found, ok is false and (row, col) come
// back unchanged.
func (e *editor) findNextMatch(row, col int, query string, forward bool) (int, int, bool, bool) {
	if query == "" || e.buf.lineCount() == 0 {
		return row, col, false, false
	}

	lineCount := e.buf.lineCount()

	if forward {
		// Current line, after the cursor.
		matches := findMatchesInLine(e.buf.line(row), query)
		for _, m := range matches {
			if m.start > col {
				return row, m.start, true, false
			}
		}
		// Later lines.
		for r := row + 1; r < lineCount; r++ {
			matches := findMatchesInLine(e.buf.line(r), query)
			if len(matches) > 0 {
				return r, matches[0].start, true, false
			}
		}
		// Wrap to top.
		for r := 0; r < row; r++ {
			matches := findMatchesInLine(e.buf.line(r), query)
			if len(matches) > 0 {
				return r, matches[0].start, true, true
			}
		}
		// Match at or before cursor on the same line counts as a wrap.
		for _, m := range matches {
			if m.start <= col {
				return row, m.start, true, true
			}
		}
		return row, col, false, false
	}

	// Backwards.
	matches := findMatchesInLine(e.buf.line(row), query)
	for i := len(matches) - 1; i >= 0; i-- {
		if matches[i].start < col {
			return row, matches[i].start, true, false
		}
	}
	for r := row - 1; r >= 0; r-- {
		matches := findMatchesInLine(e.buf.line(r), query)
		if len(matches) > 0 {
			return r, matches[len(matches)-1].start, true, false
		}
	}
	for r := lineCount - 1; r > row; r-- {
		matches := findMatchesInLine(e.buf.line(r), query)
		if len(matches) > 0 {
			return r, matches[len(matches)-1].start, true, true
		}
	}
	for i := len(matches) - 1; i >= 0; i-- {
		if matches[i].start >= col {
			return row, matches[i].start, true, true
		}
	}
	return row, col, false, false
}
