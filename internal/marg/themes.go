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
	bg         string    // editor background (empty = terminal default)
	fg         string    // body text foreground
	boldFg     string    // **bold** text — picked to read distinctly on top of fg even when the terminal's bold-weight rendering is subtle
	muted      string    // status bar, line numbers, blockquote text
	dim        string    // very low-contrast accents (line gutters)
	accent     string    // mode indicator, accent strokes — one accent only
	headings   [6]string // # / ## / ### / #### / ##### / ###### colors, H1..H6
	codeInline string    // inline `code`
	link       string    // [text](url)
	quote      string    // > blockquote text
	selection  string    // visual-mode selection background
	matchBg    string    // /search match background
	matchFg    string    // /search match foreground
	treeCursor string    // file tree current row background
	cursorLine string    // background tint behind the line with the cursor
	statusFg   string    // status-bar foreground (mode label etc)
}

// Catppuccin Mocha tones for the dark theme. Headings cycle through the
// "rainbow" mapping catppuccin uses for H1..H6 in markdown — red, peach,
// yellow, green, sapphire, lavender — so document hierarchy reads at a
// glance instead of every heading being one flat color.
var paletteDark = palette{
	bg:     "", // honor the terminal background
	fg:     "#cdd6f4",
	boldFg: "#ffffff", // pure white so **bold** stands out even when the terminal's bold-weight rendering is weak
	muted:  "#7f849c",
	dim:    "#45475a",
	accent: "#89b4fa",
	headings: [6]string{
		"#f38ba8", // H1 red
		"#fab387", // H2 peach
		"#f9e2af", // H3 yellow
		"#a6e3a1", // H4 green
		"#74c7ec", // H5 sapphire
		"#b4befe", // H6 lavender
	},
	codeInline: "#f2cdcd", // flamingo — what catppuccin uses for inline code
	link:       "#89b4fa", // blue, matching catppuccin's markdownLinkText
	quote:      "#7f849c",
	selection:  "#414559",
	matchBg:    "#494d64",
	matchFg:    "#f9e2af",
	treeCursor: "#585b70",
	cursorLine: "#1e1e2e",
	statusFg:   "#89b4fa",
}

// Solarized-leaning palette. Headings cycle through solarized's accent
// colors for the same H1..H6 effect as the dark theme.
var paletteLight = palette{
	bg:     "#fdf6e3",
	fg:     "#586e75",
	boldFg: "#073642", // solarized base02 — clearly darker than fg
	muted:  "#93a1a1",
	dim:    "#eee8d5",
	accent: "#268bd2",
	headings: [6]string{
		"#dc322f", // H1 red
		"#cb4b16", // H2 orange
		"#b58900", // H3 yellow
		"#859900", // H4 green
		"#268bd2", // H5 blue
		"#6c71c4", // H6 violet
	},
	codeInline: "#859900",
	link:       "#2aa198",
	quote:      "#93a1a1",
	selection:  "#eee8d5",
	matchBg:    "#f5e8b8",
	matchFg:    "#586e75",
	treeCursor: "#ddd6c1",
	cursorLine: "#f7f0d8",
	statusFg:   "#268bd2",
}

// Warm earthy palette. Heading rainbow uses tones that read on the cream
// background without competing with each other.
var paletteSepia = palette{
	bg:     "#f4ecd8",
	fg:     "#5c4d3c",
	boldFg: "#2a1f15", // deep coffee — clearly darker than fg on the cream bg
	muted:  "#93826a",
	dim:    "#c5b898",
	accent: "#1f6feb",
	headings: [6]string{
		"#8b3a3a", // H1 oxblood
		"#9c4a1a", // H2 rust
		"#8b5a2b", // H3 amber
		"#6b6b3a", // H4 olive
		"#1f4e79", // H5 navy
		"#5b3e7a", // H6 plum
	},
	codeInline: "#4f7942",
	link:       "#1f6feb",
	quote:      "#93826a",
	selection:  "#e8dcc0",
	matchBg:    "#f0e1b0",
	matchFg:    "#5c4d3c",
	treeCursor: "#d6c8a8",
	cursorLine: "#f0e5c7",
	statusFg:   "#1f6feb",
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
