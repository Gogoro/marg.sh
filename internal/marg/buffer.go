package marg

import (
	"os"
	"strings"
)

// buffer holds the text being edited as a slice of logical lines (without
// trailing newlines). Operations work in rune-space, not byte-space, so
// non-ASCII characters behave correctly.
type buffer struct {
	lines [][]rune
}

func newBuffer() *buffer {
	return &buffer{lines: [][]rune{{}}}
}

func bufferFromString(s string) *buffer {
	b := &buffer{}
	if s == "" {
		b.lines = [][]rune{{}}
		return b
	}
	for _, line := range strings.Split(s, "\n") {
		b.lines = append(b.lines, []rune(line))
	}
	return b
}

func loadBufferFromFile(path string) (*buffer, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return newBuffer(), nil
		}
		return nil, err
	}
	return bufferFromString(string(data)), nil
}

func (b *buffer) lineCount() int { return len(b.lines) }

func (b *buffer) lineLen(row int) int {
	if row < 0 || row >= len(b.lines) {
		return 0
	}
	return len(b.lines[row])
}

func (b *buffer) line(row int) []rune {
	if row < 0 || row >= len(b.lines) {
		return nil
	}
	return b.lines[row]
}

// insertRune inserts r at (row, col). Caller must ensure (row, col) is valid.
func (b *buffer) insertRune(row, col int, r rune) {
	line := b.lines[row]
	if col < 0 {
		col = 0
	}
	if col > len(line) {
		col = len(line)
	}
	out := make([]rune, len(line)+1)
	copy(out, line[:col])
	out[col] = r
	copy(out[col+1:], line[col:])
	b.lines[row] = out
}

// insertNewline splits line at (row, col) and produces two lines.
func (b *buffer) insertNewline(row, col int) {
	line := b.lines[row]
	if col < 0 {
		col = 0
	}
	if col > len(line) {
		col = len(line)
	}
	left := append([]rune{}, line[:col]...)
	right := append([]rune{}, line[col:]...)
	b.lines[row] = left
	// Insert right after row.
	b.lines = append(b.lines, nil)
	copy(b.lines[row+2:], b.lines[row+1:])
	b.lines[row+1] = right
}

// deleteRuneBefore removes the rune to the left of (row, col). Returns the
// new (row, col) cursor position.
func (b *buffer) deleteRuneBefore(row, col int) (int, int) {
	if row == 0 && col == 0 {
		return row, col
	}
	if col == 0 {
		// Join with previous line.
		prevLen := len(b.lines[row-1])
		b.lines[row-1] = append(b.lines[row-1], b.lines[row]...)
		b.lines = append(b.lines[:row], b.lines[row+1:]...)
		return row - 1, prevLen
	}
	line := b.lines[row]
	out := make([]rune, len(line)-1)
	copy(out, line[:col-1])
	copy(out[col-1:], line[col:])
	b.lines[row] = out
	return row, col - 1
}

// deleteRuneAt removes the rune at (row, col) (forward delete).
func (b *buffer) deleteRuneAt(row, col int) (int, int) {
	line := b.lines[row]
	if col >= len(line) {
		// Join with next line if any.
		if row+1 >= len(b.lines) {
			return row, col
		}
		b.lines[row] = append(line, b.lines[row+1]...)
		b.lines = append(b.lines[:row+1], b.lines[row+2:]...)
		return row, col
	}
	out := make([]rune, len(line)-1)
	copy(out, line[:col])
	copy(out[col:], line[col+1:])
	b.lines[row] = out
	return row, col
}

// insertLineBelow inserts an empty line after row.
func (b *buffer) insertLineBelow(row int) {
	b.lines = append(b.lines, nil)
	copy(b.lines[row+2:], b.lines[row+1:])
	b.lines[row+1] = []rune{}
}

// insertLineAbove inserts an empty line at row, shifting existing lines down.
func (b *buffer) insertLineAbove(row int) {
	b.lines = append(b.lines, nil)
	copy(b.lines[row+1:], b.lines[row:])
	b.lines[row] = []rune{}
}

// toString rebuilds the file contents with newline separators.
func (b *buffer) toString() string {
	parts := make([]string, len(b.lines))
	for i, line := range b.lines {
		parts[i] = string(line)
	}
	return strings.Join(parts, "\n")
}

// deleteRange removes runes in [startRow,startCol] .. [endRow,endCol] inclusive.
// Returns the cursor position after the deletion (the start of the range).
func (b *buffer) deleteRange(startRow, startCol, endRow, endCol int) (int, int) {
	startRow, startCol, endRow, endCol = normalizeRange(startRow, startCol, endRow, endCol)
	if startRow == endRow {
		line := b.lines[startRow]
		if endCol >= len(line) {
			endCol = len(line) - 1
		}
		if endCol < startCol {
			return startRow, startCol
		}
		out := make([]rune, 0, len(line)-(endCol-startCol+1))
		out = append(out, line[:startCol]...)
		out = append(out, line[endCol+1:]...)
		b.lines[startRow] = out
		return startRow, startCol
	}
	head := append([]rune{}, b.lines[startRow][:startCol]...)
	end := b.lines[endRow]
	tailStart := endCol + 1
	if tailStart > len(end) {
		tailStart = len(end)
	}
	tail := append([]rune{}, end[tailStart:]...)
	b.lines[startRow] = append(head, tail...)
	b.lines = append(b.lines[:startRow+1], b.lines[endRow+1:]...)
	return startRow, startCol
}

// textRange returns the runes covered by [startRow,startCol] .. [endRow,endCol] inclusive.
func (b *buffer) textRange(startRow, startCol, endRow, endCol int) string {
	startRow, startCol, endRow, endCol = normalizeRange(startRow, startCol, endRow, endCol)
	if startRow == endRow {
		line := b.lines[startRow]
		if endCol >= len(line) {
			endCol = len(line) - 1
		}
		if endCol < startCol {
			return ""
		}
		return string(line[startCol : endCol+1])
	}
	parts := []string{string(b.lines[startRow][startCol:])}
	for r := startRow + 1; r < endRow; r++ {
		parts = append(parts, string(b.lines[r]))
	}
	end := b.lines[endRow]
	tail := endCol + 1
	if tail > len(end) {
		tail = len(end)
	}
	parts = append(parts, string(end[:tail]))
	return strings.Join(parts, "\n")
}

// deleteLines removes lines [start, end] inclusive and returns their joined text.
func (b *buffer) deleteLines(start, end int) string {
	if start > end {
		start, end = end, start
	}
	if start < 0 {
		start = 0
	}
	if end >= len(b.lines) {
		end = len(b.lines) - 1
	}
	parts := make([]string, 0, end-start+1)
	for r := start; r <= end; r++ {
		parts = append(parts, string(b.lines[r]))
	}
	text := strings.Join(parts, "\n")
	b.lines = append(b.lines[:start], b.lines[end+1:]...)
	if len(b.lines) == 0 {
		b.lines = [][]rune{{}}
	}
	return text
}

// insertText inserts s at (row, col), splitting on embedded newlines.
// Returns the cursor position immediately after the last inserted rune.
func (b *buffer) insertText(row, col int, s string) (int, int) {
	if s == "" {
		return row, col
	}
	if col > len(b.lines[row]) {
		col = len(b.lines[row])
	}
	parts := strings.Split(s, "\n")
	if len(parts) == 1 {
		for _, r := range parts[0] {
			b.insertRune(row, col, r)
			col++
		}
		return row, col
	}
	line := b.lines[row]
	headRunes := append([]rune{}, line[:col]...)
	tailRunes := append([]rune{}, line[col:]...)
	first := append(headRunes, []rune(parts[0])...)
	b.lines[row] = first

	insertions := make([][]rune, 0, len(parts)-1)
	for _, p := range parts[1 : len(parts)-1] {
		insertions = append(insertions, []rune(p))
	}
	last := append([]rune(parts[len(parts)-1]), tailRunes...)
	insertions = append(insertions, last)

	newLines := make([][]rune, 0, len(b.lines)+len(insertions))
	newLines = append(newLines, b.lines[:row+1]...)
	newLines = append(newLines, insertions...)
	newLines = append(newLines, b.lines[row+1:]...)
	b.lines = newLines

	return row + len(parts) - 1, len(parts[len(parts)-1])
}

// insertLinesBelow places each line of `text` (split on \n) as a new line
// immediately after `row`. Returns the row of the first inserted line.
func (b *buffer) insertLinesBelow(row int, text string) int {
	parts := strings.Split(text, "\n")
	for i := 0; i < len(parts); i++ {
		b.insertLineBelow(row + i)
	}
	for i, p := range parts {
		b.lines[row+1+i] = []rune(p)
	}
	return row + 1
}

// insertLinesAbove places each line of `text` as a new line at `row`,
// shifting existing lines down. Returns the row of the first inserted line.
func (b *buffer) insertLinesAbove(row int, text string) int {
	parts := strings.Split(text, "\n")
	for i := 0; i < len(parts); i++ {
		b.insertLineAbove(row + i)
	}
	for i, p := range parts {
		b.lines[row+i] = []rune(p)
	}
	return row
}

func normalizeRange(sr, sc, er, ec int) (int, int, int, int) {
	if sr > er || (sr == er && sc > ec) {
		return er, ec, sr, sc
	}
	return sr, sc, er, ec
}

func (b *buffer) wordCount() int {
	n := 0
	for _, line := range b.lines {
		inWord := false
		for _, r := range line {
			isSpace := r == ' ' || r == '\t'
			if !isSpace && !inWord {
				n++
				inWord = true
			} else if isSpace {
				inWord = false
			}
		}
	}
	return n
}

func (b *buffer) charCount() int {
	n := 0
	for _, line := range b.lines {
		n += len(line)
	}
	return n
}
