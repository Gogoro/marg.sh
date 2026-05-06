package marg

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func baseStyleForLine(line []rune) lipgloss.Style {
	s := strings.TrimLeft(string(line), " \t")
	switch {
	case strings.HasPrefix(s, "######"),
		strings.HasPrefix(s, "#####"),
		strings.HasPrefix(s, "####"),
		strings.HasPrefix(s, "###"),
		strings.HasPrefix(s, "##"),
		strings.HasPrefix(s, "#"):
		if isHeadingLine(s) {
			return styleHeading
		}
	case strings.HasPrefix(s, "```"):
		return styleCode
	case strings.HasPrefix(s, "> "):
		return styleQuote
	}
	return lipgloss.NewStyle()
}

func isHeadingLine(s string) bool {
	// `# foo` is a heading; `#tag` is not.
	for i, r := range s {
		if r != '#' {
			return i > 0 && (r == ' ' || r == '\t')
		}
	}
	return false
}

type inlineRange struct {
	start int
	end   int
	style lipgloss.Style
}

// inlineRanges scans a line for bold (**..**), italic (*..*/_.._), inline
// code (`..`), and links ([text](url)). Returns rune-index ranges.
func inlineRanges(line []rune) []inlineRange {
	var out []inlineRange
	i := 0
	for i < len(line) {
		switch {
		case i+1 < len(line) && line[i] == '*' && line[i+1] == '*':
			if end := findClose(line, i+2, "**"); end > 0 {
				out = append(out, inlineRange{start: i, end: end + 2, style: styleBold})
				i = end + 2
				continue
			}
		case line[i] == '*' || line[i] == '_':
			ch := line[i]
			if end := findCloseChar(line, i+1, ch); end > 0 {
				out = append(out, inlineRange{start: i, end: end + 1, style: styleItalic})
				i = end + 1
				continue
			}
		case line[i] == '`':
			if end := findCloseChar(line, i+1, '`'); end > 0 {
				out = append(out, inlineRange{start: i, end: end + 1, style: styleCode})
				i = end + 1
				continue
			}
		case line[i] == '[':
			if close := findCloseChar(line, i+1, ']'); close > 0 && close+1 < len(line) && line[close+1] == '(' {
				if rclose := findCloseChar(line, close+2, ')'); rclose > 0 {
					out = append(out, inlineRange{start: i, end: rclose + 1, style: styleLink})
					i = rclose + 1
					continue
				}
			}
		}
		i++
	}
	return out
}

func findClose(line []rune, from int, marker string) int {
	mr := []rune(marker)
	for i := from; i+len(mr) <= len(line); i++ {
		match := true
		for j, r := range mr {
			if line[i+j] != r {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

func findCloseChar(line []rune, from int, ch rune) int {
	for i := from; i < len(line); i++ {
		if line[i] == ch {
			return i
		}
	}
	return -1
}

// sameStyle approximates style equality via rendered output of a sentinel.
// Lip Gloss styles aren't directly comparable, but for our purposes this
// is reliable enough.
func sameStyle(a, b lipgloss.Style) bool {
	return a.Render("x") == b.Render("x")
}
