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

// useAnsiPalette is true when the user wants marg to color code blocks with
// the terminal's own ANSI bright colors (the default — guaranteed readable
// because the terminal theme controls how vibrant they look). When false we
// fall back to one of Chroma's curated styles.
var useAnsiPalette = true
var chromaStyle = mustLoadStyle("monokai")

func setCodeTheme(name string) {
	if name == "ansi" || name == "" {
		useAnsiPalette = true
		return
	}
	useAnsiPalette = false
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

// lipglossStyleFor returns the style used to render `t`. In ANSI mode we map
// token categories to the terminal's bright 8-color palette (which the
// terminal's own theme decides how to display, so it always looks "right"
// regardless of tmux truecolor config). In Chroma mode we read the loaded
// style.
func lipglossStyleFor(t chroma.TokenType) lipgloss.Style {
	if useAnsiPalette {
		return ansiStyleFor(t)
	}
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

// ansiStyleFor maps a Chroma token type to a lipgloss style using ANSI
// bright color slots. lipgloss.Color("9") through ("15") select the
// terminal's bright variants, which every modern terminal theme defines
// to be highly visible — much more reliable than truecolor escapes that
// tmux might downsample.
func ansiStyleFor(t chroma.TokenType) lipgloss.Style {
	const (
		brightRed     = "9"
		brightGreen   = "10"
		brightYellow  = "11"
		brightBlue    = "12"
		brightMagenta = "13"
		brightCyan    = "14"
		brightGray    = "8"
	)

	switch t.Category() {
	case chroma.Keyword:
		// Keyword.Type and Keyword.Pseudo get cyan; everything else magenta.
		switch t {
		case chroma.KeywordType, chroma.KeywordPseudo, chroma.KeywordReserved:
			return lipgloss.NewStyle().Foreground(lipgloss.Color(brightCyan)).Bold(true)
		}
		return lipgloss.NewStyle().Foreground(lipgloss.Color(brightMagenta)).Bold(true)
	case chroma.LiteralString:
		return lipgloss.NewStyle().Foreground(lipgloss.Color(brightYellow))
	case chroma.LiteralNumber, chroma.Literal:
		return lipgloss.NewStyle().Foreground(lipgloss.Color(brightMagenta))
	case chroma.Comment:
		return lipgloss.NewStyle().Foreground(lipgloss.Color(brightGray)).Italic(true)
	case chroma.Operator:
		return lipgloss.NewStyle().Foreground(lipgloss.Color(brightMagenta))
	case chroma.Name:
		switch t {
		case chroma.NameFunction, chroma.NameClass, chroma.NameDecorator:
			return lipgloss.NewStyle().Foreground(lipgloss.Color(brightGreen))
		case chroma.NameBuiltin, chroma.NameBuiltinPseudo, chroma.NameConstant:
			return lipgloss.NewStyle().Foreground(lipgloss.Color(brightCyan))
		case chroma.NameTag:
			return lipgloss.NewStyle().Foreground(lipgloss.Color(brightRed))
		}
		return lipgloss.NewStyle()
	case chroma.GenericInserted:
		return lipgloss.NewStyle().Foreground(lipgloss.Color(brightGreen))
	case chroma.GenericDeleted:
		return lipgloss.NewStyle().Foreground(lipgloss.Color(brightRed))
	}
	_ = brightBlue
	return lipgloss.NewStyle()
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
