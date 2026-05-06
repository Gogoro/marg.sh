package marg

// palette holds every colour marg renders with. Three curated palettes
// ship with the binary; the user picks one via `theme = "..."` in config.
//
// Each colour is a hex string. A `bg` colour of "" means "let the terminal
// background show through" — that's how the dark theme works on a dark
// terminal without forcing an opaque rectangle behind the text. Light /
// sepia themes set bg explicitly so they look right even in a dark
// terminal window.
type palette struct {
	bg          string // editor background (empty = terminal default)
	fg          string // body text foreground
	muted       string // status bar, line numbers, blockquote text
	dim         string // very low-contrast accents (line gutters)
	accent      string // mode indicator, accent strokes — one accent only
	heading     string // # / ## / ### lines
	codeInline  string // inline `code`
	link        string // [text](url)
	quote       string // > blockquote text
	selection   string // visual-mode selection background
	matchBg     string // /search match background
	matchFg     string // /search match foreground
	treeCursor  string // file tree current row background
	codeBlockBg string // background tint for fenced code-block rows
	cursorLine  string // background tint behind the line with the cursor
	statusFg    string // status-bar foreground (mode label etc)
}

var paletteDark = palette{
	bg:          "", // honor the terminal background
	fg:          "#cdd6f4",
	muted:       "#7f849c",
	dim:         "#45475a",
	accent:      "#89b4fa",
	heading:     "#fab387",
	codeInline:  "#a6e3a1",
	link:        "#89dceb",
	quote:       "#7f849c",
	selection:   "#414559",
	matchBg:     "#494d64",
	matchFg:     "#f9e2af",
	treeCursor:  "#585b70",
	codeBlockBg: "#181825",
	cursorLine:  "#1e1e2e",
	statusFg:    "#89b4fa",
}

var paletteLight = palette{
	bg:          "#fdf6e3",
	fg:          "#586e75",
	muted:       "#93a1a1",
	dim:         "#eee8d5",
	accent:      "#268bd2",
	heading:     "#b58900",
	codeInline:  "#859900",
	link:        "#2aa198",
	quote:       "#93a1a1",
	selection:   "#eee8d5",
	matchBg:     "#f5e8b8",
	matchFg:     "#586e75",
	treeCursor:  "#ddd6c1",
	codeBlockBg: "#f3ecd5",
	cursorLine:  "#f7f0d8",
	statusFg:    "#268bd2",
}

var paletteSepia = palette{
	bg:          "#f4ecd8",
	fg:          "#5c4d3c",
	muted:       "#93826a",
	dim:         "#c5b898",
	accent:      "#1f6feb",
	heading:     "#8b5a2b",
	codeInline:  "#4f7942",
	link:        "#1f6feb",
	quote:       "#93826a",
	selection:   "#e8dcc0",
	matchBg:     "#f0e1b0",
	matchFg:     "#5c4d3c",
	treeCursor:  "#d6c8a8",
	codeBlockBg: "#efe5c9",
	cursorLine:  "#f0e5c7",
	statusFg:    "#1f6feb",
}

// active is the currently selected palette. Reassigned by applyTheme.
var active = paletteDark

func applyTheme(name string) {
	switch name {
	case "light":
		active = paletteLight
	case "sepia":
		active = paletteSepia
	case "", "dark":
		active = paletteDark
	default:
		active = paletteDark
	}
	rebuildStyles()
}
