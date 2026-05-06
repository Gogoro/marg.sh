package marg

import (
	"context"
	"embed"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/bash"
	"github.com/smacker/go-tree-sitter/golang"
	"github.com/smacker/go-tree-sitter/javascript"
	"github.com/smacker/go-tree-sitter/python"
	"github.com/smacker/go-tree-sitter/rust"
	"github.com/smacker/go-tree-sitter/sql"
	"github.com/smacker/go-tree-sitter/typescript/typescript"
	"github.com/smacker/go-tree-sitter/yaml"
)

//go:embed queries/*.scm
var highlightQueries embed.FS

// tsLanguage bundles a parser language pointer with its compiled highlight
// query and the fence-tag aliases users might write. typePrefixes is a
// fallback for grammars (notably SQL) that expose every keyword as its own
// node type — `keyword_select`, `keyword_from`, etc. Tree-sitter queries
// can only filter captures by node text or by anonymous-string match, not
// by node-type prefix, so we walk the AST after the query and emit spans
// for any node whose type matches one of the registered prefixes.
type tsLanguage struct {
	name         string
	sitter       *sitter.Language
	query        *sitter.Query
	aliases      []string
	typePrefixes map[string]string // node-type-prefix → capture name
}

var tsLanguages []*tsLanguage

func init() {
	tsLanguages = []*tsLanguage{
		loadTSLanguage("sql", sql.GetLanguage(), []string{"postgres", "postgresql", "mysql", "sqlite"},
			map[string]string{"keyword_": "keyword"}),
		loadTSLanguage("go", golang.GetLanguage(), []string{"golang"}, nil),
		loadTSLanguage("javascript", javascript.GetLanguage(), []string{"js", "jsx", "node"}, nil),
		loadTSLanguage("typescript", typescript.GetLanguage(), []string{"ts", "tsx"}, nil),
		loadTSLanguage("rust", rust.GetLanguage(), []string{"rs"}, nil),
		loadTSLanguage("python", python.GetLanguage(), []string{"py", "py3", "python3"}, nil),
		loadTSLanguage("bash", bash.GetLanguage(), []string{"sh", "shell", "zsh"}, nil),
		loadTSLanguage("yaml", yaml.GetLanguage(), []string{"yml"}, nil),
	}
}

func loadTSLanguage(name string, lang *sitter.Language, aliases []string, typePrefixes map[string]string) *tsLanguage {
	data, err := highlightQueries.ReadFile("queries/" + name + ".scm")
	if err != nil {
		return nil
	}
	q, err := sitter.NewQuery(data, lang)
	if err != nil {
		return nil
	}
	return &tsLanguage{name: name, sitter: lang, query: q, aliases: aliases, typePrefixes: typePrefixes}
}

// tsLanguageFor looks up the bundled tree-sitter language matching the fence
// tag (`go`, `js`, `postgres`, etc.) and returns nil if we don't ship it.
func tsLanguageFor(hint string) *tsLanguage {
	if hint == "" {
		return nil
	}
	h := strings.ToLower(strings.TrimSpace(hint))
	for _, l := range tsLanguages {
		if l == nil {
			continue
		}
		if l.name == h {
			return l
		}
		for _, a := range l.aliases {
			if a == h {
				return l
			}
		}
	}
	return nil
}

// tsTokenizeBlock parses `text` with the language's tree-sitter parser, runs
// the embedded highlight query, and writes per-row tokenSpan results into
// `spans` keyed by buffer row (`startRow` is the buffer row corresponding to
// the first line of `text`). Returns true if highlighting was applied.
func tsTokenizeBlock(spans map[int][]tokenSpan, lang *tsLanguage, text string, startRow int) bool {
	if lang == nil {
		return false
	}
	source := []byte(text)
	tree, err := sitter.ParseCtx(context.Background(), source, lang.sitter)
	if err != nil || tree == nil {
		return false
	}

	cursor := sitter.NewQueryCursor()
	defer cursor.Close()
	cursor.Exec(lang.query, tree)

	lineByteOffsets := computeLineByteOffsets(source)

	for {
		match, ok := cursor.NextMatch()
		if !ok {
			break
		}
		match = cursor.FilterPredicates(match, source)
		for _, capture := range match.Captures {
			captureName := lang.query.CaptureNameForId(capture.Index)
			style, ok := tsStyleFor(captureName)
			if !ok {
				continue
			}
			emitCaptureSpans(spans, source, lineByteOffsets, capture.Node, style, startRow)
		}
	}

	if len(lang.typePrefixes) > 0 {
		walkForTypePrefixes(spans, source, lineByteOffsets, tree, lang.typePrefixes, startRow)
	}
	return true
}

// walkForTypePrefixes is the type-based fallback for grammars where
// keywords (and similar leaf categories) live in the node-type taxonomy
// rather than inside named structural nodes — most prominently SQL,
// where `SELECT` is `keyword_select`, `FROM` is `keyword_from`, and so
// on for ~350 variants. We walk the tree once and emit a span for every
// node whose type starts with one of the registered prefixes.
func walkForTypePrefixes(
	spans map[int][]tokenSpan,
	source []byte,
	lineByteOffsets []int,
	root *sitter.Node,
	prefixes map[string]string,
	startRow int,
) {
	var visit func(n *sitter.Node)
	visit = func(n *sitter.Node) {
		nodeType := n.Type()
		for prefix, captureName := range prefixes {
			if strings.HasPrefix(nodeType, prefix) {
				if style, ok := tsStyleFor(captureName); ok {
					emitCaptureSpans(spans, source, lineByteOffsets, n, style, startRow)
				}
				break
			}
		}
		count := int(n.NamedChildCount())
		for i := 0; i < count; i++ {
			visit(n.NamedChild(i))
		}
	}
	visit(root)
}

// emitCaptureSpans writes one tokenSpan per row the capture covers,
// translating tree-sitter's byte-column points into rune-column indices
// that match the renderer's expectations.
func emitCaptureSpans(
	spans map[int][]tokenSpan,
	source []byte,
	lineByteOffsets []int,
	node *sitter.Node,
	style lipgloss.Style,
	startRow int,
) {
	startPoint := node.StartPoint()
	endPoint := node.EndPoint()
	startSourceRow := int(startPoint.Row)
	endSourceRow := int(endPoint.Row)

	for row := startSourceRow; row <= endSourceRow; row++ {
		lineBytes := lineBytesAt(source, lineByteOffsets, row)
		var fromByte, toByte int
		if row == startSourceRow {
			fromByte = int(startPoint.Column)
		} else {
			fromByte = 0
		}
		if row == endSourceRow {
			toByte = int(endPoint.Column)
		} else {
			toByte = len(lineBytes)
		}
		startCol := byteToRuneIndex(lineBytes, fromByte)
		endCol := byteToRuneIndex(lineBytes, toByte)
		if endCol <= startCol {
			continue
		}
		bufferRow := startRow + row
		spans[bufferRow] = append(spans[bufferRow], tokenSpan{
			startCol: startCol,
			endCol:   endCol,
			style:    style,
		})
	}
}

func computeLineByteOffsets(source []byte) []int {
	offsets := []int{0}
	for i, b := range source {
		if b == '\n' {
			offsets = append(offsets, i+1)
		}
	}
	return offsets
}

func lineBytesAt(source []byte, offsets []int, row int) []byte {
	if row < 0 || row >= len(offsets) {
		return nil
	}
	start := offsets[row]
	end := len(source)
	if row+1 < len(offsets) {
		end = offsets[row+1] - 1 // drop the newline byte
		if end < start {
			end = start
		}
	}
	return source[start:end]
}

func byteToRuneIndex(line []byte, byteIndex int) int {
	if byteIndex >= len(line) {
		return utf8.RuneCount(line)
	}
	return utf8.RuneCount(line[:byteIndex])
}

// tsStyleFor maps a tree-sitter capture name (e.g. "keyword",
// "function", "string.escape") to the marg lipgloss style. Falls back
// from a dotted name to its parent (so `function.builtin` reuses the
// `function` color if no specific override is registered).
func tsStyleFor(name string) (lipgloss.Style, bool) {
	for {
		if s, ok := tsStyleTable[name]; ok {
			return s, true
		}
		dot := strings.LastIndex(name, ".")
		if dot < 0 {
			return lipgloss.Style{}, false
		}
		name = name[:dot]
	}
}

// tsStyleTable maps capture names to lipgloss styles using the catppuccin
// mocha tones. Built once and reused; recolours when the user switches
// theme via rebuildTSStyles.
var tsStyleTable map[string]lipgloss.Style

func init() {
	rebuildTSStyles()
}

// catppuccin mocha tones. Used directly here because the syntax palette
// is independent of marg's prose palette — these are the colours nvim
// uses inside fenced code blocks regardless of the surrounding theme.
const (
	tsPink      = "#f5c2e7"
	tsMauve     = "#cba6f7"
	tsRed       = "#f38ba8"
	tsMaroon    = "#eba0ac"
	tsPeach     = "#fab387"
	tsYellow    = "#f9e2af"
	tsGreen     = "#a6e3a1"
	tsTeal      = "#94e2d5"
	tsSky       = "#89dceb"
	tsBlue      = "#89b4fa"
	tsLavender  = "#b4befe"
	tsTextColor = "#cdd6f4"
	tsOverlay2  = "#9399b2"
)

func rebuildTSStyles() {
	tsStyleTable = map[string]lipgloss.Style{
		"keyword":            lipgloss.NewStyle().Foreground(lipgloss.Color(tsMauve)).Bold(true),
		"function":           lipgloss.NewStyle().Foreground(lipgloss.Color(tsBlue)),
		"function.builtin":   lipgloss.NewStyle().Foreground(lipgloss.Color(tsPeach)),
		"function.macro":     lipgloss.NewStyle().Foreground(lipgloss.Color(tsTeal)),
		"type":               lipgloss.NewStyle().Foreground(lipgloss.Color(tsYellow)),
		"type.builtin":       lipgloss.NewStyle().Foreground(lipgloss.Color(tsYellow)).Italic(true),
		"string":             lipgloss.NewStyle().Foreground(lipgloss.Color(tsGreen)),
		"string.escape":      lipgloss.NewStyle().Foreground(lipgloss.Color(tsPink)),
		"number":             lipgloss.NewStyle().Foreground(lipgloss.Color(tsPeach)),
		"boolean":            lipgloss.NewStyle().Foreground(lipgloss.Color(tsPeach)).Italic(true),
		"comment":            lipgloss.NewStyle().Foreground(lipgloss.Color(tsOverlay2)).Italic(true),
		"constant":           lipgloss.NewStyle().Foreground(lipgloss.Color(tsPeach)),
		"constant.builtin":   lipgloss.NewStyle().Foreground(lipgloss.Color(tsPeach)).Italic(true),
		"operator":           lipgloss.NewStyle().Foreground(lipgloss.Color(tsSky)),
		"variable":           lipgloss.NewStyle().Foreground(lipgloss.Color(tsTextColor)),
		"variable.builtin":   lipgloss.NewStyle().Foreground(lipgloss.Color(tsRed)),
		"variable.member":    lipgloss.NewStyle().Foreground(lipgloss.Color(tsLavender)),
		"variable.parameter": lipgloss.NewStyle().Foreground(lipgloss.Color(tsMaroon)),
		"property":           lipgloss.NewStyle().Foreground(lipgloss.Color(tsLavender)),
		"attribute":          lipgloss.NewStyle().Foreground(lipgloss.Color(tsYellow)),
		"tag":                lipgloss.NewStyle().Foreground(lipgloss.Color(tsRed)),
	}
}
