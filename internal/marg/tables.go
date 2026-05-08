package marg

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// formatTablesInBuffer rewrites every markdown table in the buffer so each
// column is padded to the width of its widest cell. Called on save so the
// on-disk file always has a clean grid.
//
// Rules: a table block is a run of consecutive lines that match isTableRow
// and contains a valid separator row (`| --- | --- |` or with alignment
// markers `:---`, `---:`, `:---:`) at offset 1. Anything else is left alone.
// Lines inside fenced code blocks are skipped so markdown-inside-markdown
// examples don't get mangled.
func formatTablesInBuffer(b *buffer) {
	inCode := false
	row := 0
	for row < b.lineCount() {
		line := b.line(row)
		trimmed := strings.TrimSpace(string(line))
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inCode = !inCode
			row++
			continue
		}
		if inCode {
			row++
			continue
		}
		if !isTableRow(line) {
			row++
			continue
		}
		end := row
		for end+1 < b.lineCount() && isTableRow(b.line(end+1)) {
			end++
		}
		// A real markdown table needs at least header + separator. Anything
		// shorter is probably one-off pipe-art, leave it untouched.
		if end-row >= 1 && isSeparatorRow(b.line(row+1)) {
			block := make([]string, end-row+1)
			for i := range block {
				block[i] = string(b.line(row + i))
			}
			for i, line := range formatTableBlock(block) {
				b.lines[row+i] = []rune(line)
			}
		}
		row = end + 1
	}
}

type tableAlignment int

const (
	alignLeft tableAlignment = iota
	alignRight
	alignCenter
)

// formatTableBlock takes the source lines of one markdown table and returns
// the same number of lines with columns padded to the widest cell in each
// column. Alignment is inherited from the separator row.
func formatTableBlock(lines []string) []string {
	rows := make([][]string, len(lines))
	for i, line := range lines {
		rows[i] = parseTableRow(line)
	}
	const sepIdx = 1
	aligns := parseAlignments(rows[sepIdx])

	cols := 0
	for _, row := range rows {
		if len(row) > cols {
			cols = len(row)
		}
	}
	for i := range rows {
		for len(rows[i]) < cols {
			rows[i] = append(rows[i], "")
		}
	}
	for len(aligns) < cols {
		aligns = append(aligns, alignLeft)
	}

	widths := make([]int, cols)
	for i, row := range rows {
		if i == sepIdx {
			continue
		}
		for j, cell := range row {
			if w := lipgloss.Width(cell); w > widths[j] {
				widths[j] = w
			}
		}
	}
	// Separator needs at least three dashes plus optional colons, so the
	// minimum column width that renders a syntactically valid separator is
	// 3 ("---") or 4–5 with colons. Bump tiny columns up so the separator
	// always parses cleanly.
	for j, a := range aligns {
		min := 3
		if a == alignCenter {
			min = 5
		} else if a == alignLeft || a == alignRight {
			min = 3
		}
		if widths[j] < min {
			widths[j] = min
		}
	}

	out := make([]string, len(lines))
	for i, row := range rows {
		var sb strings.Builder
		sb.WriteByte('|')
		for j, cell := range row {
			sb.WriteByte(' ')
			if i == sepIdx {
				sb.WriteString(separatorCell(widths[j], aligns[j]))
			} else {
				sb.WriteString(padCell(cell, widths[j], aligns[j]))
			}
			sb.WriteByte(' ')
			sb.WriteByte('|')
		}
		out[i] = sb.String()
	}
	return out
}

// parseTableRow splits a line on `|` into trimmed cell contents. Leading
// and trailing pipes are stripped so a `| a | b |` line yields ["a", "b"].
// Backslash-escaped pipes (`\|`) inside cell text are preserved as a
// single literal pipe.
func parseTableRow(line string) []string {
	s := strings.TrimSpace(line)
	s = strings.TrimPrefix(s, "|")
	s = strings.TrimSuffix(s, "|")

	var cells []string
	var current strings.Builder
	runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if r == '\\' && i+1 < len(runes) && runes[i+1] == '|' {
			current.WriteRune('|')
			i++
			continue
		}
		if r == '|' {
			cells = append(cells, strings.TrimSpace(current.String()))
			current.Reset()
			continue
		}
		current.WriteRune(r)
	}
	cells = append(cells, strings.TrimSpace(current.String()))
	return cells
}

// isSeparatorRow reports whether the line is a valid markdown table
// separator — every cell shaped like `:?-+:?` after trimming.
func isSeparatorRow(line []rune) bool {
	cells := parseTableRow(string(line))
	if len(cells) == 0 {
		return false
	}
	for _, cell := range cells {
		if !isSeparatorCell(cell) {
			return false
		}
	}
	return true
}

func isSeparatorCell(cell string) bool {
	s := strings.TrimSpace(cell)
	s = strings.TrimPrefix(s, ":")
	s = strings.TrimSuffix(s, ":")
	if len(s) == 0 {
		return false
	}
	for _, r := range s {
		if r != '-' {
			return false
		}
	}
	return true
}

func parseAlignments(separatorCells []string) []tableAlignment {
	out := make([]tableAlignment, len(separatorCells))
	for i, cell := range separatorCells {
		s := strings.TrimSpace(cell)
		left := strings.HasPrefix(s, ":")
		right := strings.HasSuffix(s, ":") && len(s) > 1
		switch {
		case left && right:
			out[i] = alignCenter
		case right:
			out[i] = alignRight
		default:
			out[i] = alignLeft
		}
	}
	return out
}

func padCell(text string, width int, a tableAlignment) string {
	pad := width - lipgloss.Width(text)
	if pad <= 0 {
		return text
	}
	switch a {
	case alignRight:
		return strings.Repeat(" ", pad) + text
	case alignCenter:
		left := pad / 2
		right := pad - left
		return strings.Repeat(" ", left) + text + strings.Repeat(" ", right)
	default:
		return text + strings.Repeat(" ", pad)
	}
}

func separatorCell(width int, a tableAlignment) string {
	switch a {
	case alignRight:
		return strings.Repeat("-", width-1) + ":"
	case alignCenter:
		return ":" + strings.Repeat("-", width-2) + ":"
	default:
		return strings.Repeat("-", width)
	}
}
