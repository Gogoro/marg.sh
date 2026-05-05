package marg

import (
	"fmt"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/charmbracelet/lipgloss"
)

// codeBlockSpans maps a logical row to the highlighted token spans for that
// row, when the row sits inside a fenced code block. Computed once per
// render via scanCodeBlocks.
type codeBlockSpans struct {
	spans  map[int][]tokenSpan
	inCode map[int]bool // includes opening/closing fences
}

type tokenSpan struct {
	startCol int
	endCol   int // exclusive
	style    lipgloss.Style
}

// chromaStyle is the active Chroma palette. setCodeTheme swaps it once at
// startup based on the user's `code_theme` config.
var chromaStyle = mustLoadStyle("monokai")

func setCodeTheme(name string) {
	chromaStyle = mustLoadStyle(name)
}

func mustLoadStyle(name string) *chroma.Style {
	if s := styles.Get(name); s != nil {
		return s
	}
	if s := styles.Get("monokai"); s != nil {
		return s
	}
	return styles.Fallback
}

// scanCodeBlocks walks the buffer, finds every ``` … ``` fenced block, and
// runs Chroma's lexer on each. Result is per-row token spans keyed by
// logical row.
func (e *editor) scanCodeBlocks() codeBlockSpans {
	out := codeBlockSpans{
		spans:  map[int][]tokenSpan{},
		inCode: map[int]bool{},
	}

	inBlock := false
	startRow := 0
	lang := ""
	for r := 0; r < e.buf.lineCount(); r++ {
		line := string(e.buf.line(r))
		trimmed := strings.TrimLeft(line, " \t")
		if strings.HasPrefix(trimmed, "```") {
			if !inBlock {
				inBlock = true
				startRow = r + 1
				lang = strings.TrimSpace(strings.TrimPrefix(trimmed, "```"))
				out.inCode[r] = true
			} else {
				if startRow <= r-1 {
					tokenizeBlock(out.spans, e.buf, startRow, r-1, lang)
				}
				out.inCode[r] = true
				inBlock = false
			}
			continue
		}
		if inBlock {
			out.inCode[r] = true
		}
	}
	// Unclosed block at EOF — still highlight what we've got.
	if inBlock && startRow < e.buf.lineCount() {
		tokenizeBlock(out.spans, e.buf, startRow, e.buf.lineCount()-1, lang)
	}
	return out
}

// tokenizeBlock joins lines [startRow..endRow] inclusive, picks a lexer
// (explicit language wins; otherwise auto-detect), and writes per-row token
// spans into spans.
func tokenizeBlock(spans map[int][]tokenSpan, buf *buffer, startRow, endRow int, lang string) {
	var content strings.Builder
	for r := startRow; r <= endRow; r++ {
		if r > startRow {
			content.WriteByte('\n')
		}
		content.WriteString(string(buf.line(r)))
	}
	text := content.String()

	var lexer chroma.Lexer
	if lang != "" {
		lexer = lexers.Get(lang)
	}
	if lexer == nil {
		lexer = lexers.Analyse(text)
	}
	if lexer == nil {
		lexer = lexers.Fallback
	}

	iter, err := lexer.Tokenise(nil, text)
	if err != nil {
		return
	}

	row := startRow
	col := 0
	for token := iter(); token != chroma.EOF; token = iter() {
		style := lipglossStyleFor(token.Type)
		runes := []rune(token.Value)
		i := 0
		for i < len(runes) {
			nl := -1
			for j := i; j < len(runes); j++ {
				if runes[j] == '\n' {
					nl = j
					break
				}
			}
			if nl == -1 {
				if len(runes)-i > 0 {
					spans[row] = append(spans[row], tokenSpan{
						startCol: col, endCol: col + len(runes) - i, style: style,
					})
					col += len(runes) - i
				}
				break
			}
			if nl > i {
				spans[row] = append(spans[row], tokenSpan{
					startCol: col, endCol: col + nl - i, style: style,
				})
			}
			row++
			col = 0
			i = nl + 1
		}
	}
}

// lipglossStyleFor reads Chroma's style for the given token type and returns
// an equivalent lipgloss.Style. Falls back to plain text if no entry exists.
func lipglossStyleFor(t chroma.TokenType) lipgloss.Style {
	entry := chromaStyle.Get(t)
	s := lipgloss.NewStyle()
	if entry.Colour.IsSet() {
		s = s.Foreground(lipgloss.Color(fmt.Sprintf("#%06x", entry.Colour)))
	}
	if entry.Bold == chroma.Yes {
		s = s.Bold(true)
	}
	if entry.Italic == chroma.Yes {
		s = s.Italic(true)
	}
	if entry.Underline == chroma.Yes {
		s = s.Underline(true)
	}
	return s
}

// styleAtCol returns the token-span style covering `col` in the given list,
// or `fallback` if no span matches.
func styleAtCol(spans []tokenSpan, col int, fallback lipgloss.Style) lipgloss.Style {
	for _, sp := range spans {
		if col >= sp.startCol && col < sp.endCol {
			return sp.style
		}
	}
	return fallback
}
