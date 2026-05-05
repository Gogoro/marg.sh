package marg

import (
	"strconv"
	"strings"
)

// listLineInfo describes a markdown list line's prefix so we can continue it
// on the next line when the user presses Enter at the end.
type listLineInfo struct {
	// prefixRunes is how many runes the existing line uses for its bullet
	// or number prefix (including trailing space).
	prefixRunes int
	// nextPrefix is the literal string to insert on the new line below.
	nextPrefix string
}

// parseListLine inspects a single line and returns its list-prefix info if
// the line is a markdown list item (`- `, `* `, `+ `, or `<n>. `, with any
// leading whitespace). For numbered lists the next prefix increments.
func parseListLine(line []rune) (listLineInfo, bool) {
	i := 0
	for i < len(line) && (line[i] == ' ' || line[i] == '\t') {
		i++
	}
	indent := string(line[:i])

	if i+1 < len(line) {
		c := line[i]
		if (c == '-' || c == '*' || c == '+') && line[i+1] == ' ' {
			prefix := i + 2
			return listLineInfo{
				prefixRunes: prefix,
				nextPrefix:  indent + string(c) + " ",
			}, true
		}
	}

	j := i
	for j < len(line) && line[j] >= '0' && line[j] <= '9' {
		j++
	}
	if j > i && j+1 < len(line) && line[j] == '.' && line[j+1] == ' ' {
		n, _ := strconv.Atoi(string(line[i:j]))
		return listLineInfo{
			prefixRunes: j + 2,
			nextPrefix:  indent + strconv.Itoa(n+1) + ". ",
		}, true
	}

	return listLineInfo{}, false
}

// applyHeadingLevel rewrites e.buf.lines[row] so its heading level matches
// `level` (0 strips heading entirely, 1..6 replaces or sets `#`-prefix).
// Indentation is preserved.
func (e *editor) applyHeadingLevel(level int) {
	line := string(e.buf.line(e.row))
	i := 0
	for i < len(line) && (line[i] == ' ' || line[i] == '\t') {
		i++
	}
	indent := line[:i]
	rest := line[i:]

	hashes := 0
	for hashes < len(rest) && rest[hashes] == '#' {
		hashes++
	}
	if hashes > 0 && hashes < len(rest) && rest[hashes] == ' ' {
		rest = rest[hashes+1:]
	}

	if level == 0 {
		e.buf.lines[e.row] = []rune(indent + rest)
	} else {
		e.buf.lines[e.row] = []rune(indent + strings.Repeat("#", level) + " " + rest)
	}
	e.dirty = true
	e.recordChange()
}
