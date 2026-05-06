package marg

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func baseStyleForLine(line []rune) lipgloss.Style {
	s := strings.TrimLeft(string(line), " \t")
	if level := headingLevel(s); level > 0 {
		return styleHeadings[level-1]
	}
	switch {
	case strings.HasPrefix(s, "```"):
		return styleCode
	case strings.HasPrefix(s, "> "):
		return styleQuote
	}
	return lipgloss.NewStyle()
}

func isHeadingLine(s string) bool {
	return headingLevel(s) > 0
}

// headingLevel returns 1..6 for `#`..`######` lines and 0 for anything
// else. `#tag` (no space after the hashes) is not a heading.
func headingLevel(s string) int {
	n := 0
	for n < len(s) && s[n] == '#' {
		n++
	}
	if n == 0 || n > 6 {
		return 0
	}
	if n < len(s) && (s[n] == ' ' || s[n] == '\t') {
		return n
	}
	return 0
}

// frontmatterValueRanges returns the value-side range of a `key: value`
// frontmatter line so renderers can paint the value portion in body color
// while the key/colon stay muted via the base style.
func frontmatterValueRanges(line []rune) []inlineRange {
	for i, r := range line {
		if r == ':' {
			return []inlineRange{{
				start: i + 1,
				end:   len(line),
				style: styleFrontmatterValue,
			}}
		}
	}
	return nil
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
