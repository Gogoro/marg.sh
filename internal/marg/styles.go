package marg

import "github.com/charmbracelet/lipgloss"

// All renderable styles derive from the `active` palette in themes.go.
// They are populated once on startup via rebuildStyles() and again any
// time the user switches theme.

var (
	colorBg          lipgloss.Color
	colorFg          lipgloss.Color
	colorMuted       lipgloss.Color
	colorDim         lipgloss.Color
	colorAccent      lipgloss.Color
	colorHeadings    [6]lipgloss.Color
	colorCode        lipgloss.Color
	colorLink        lipgloss.Color
	colorQuote       lipgloss.Color
	colorSelection   lipgloss.Color
	colorMatch       lipgloss.Color
	colorMatchFg     lipgloss.Color
	colorTreeCursor  lipgloss.Color
	colorCursorLine  lipgloss.Color
	colorStatus      lipgloss.Color
	colorWarn        = lipgloss.Color("#E5C07B") // legacy — used for dirty marker
	colorText        lipgloss.Color              // alias for colorFg

	styleStatusBar        lipgloss.Style
	styleStatusMode       lipgloss.Style
	styleStatusDirty      lipgloss.Style
	styleLineNumber       lipgloss.Style
	styleCursorLineNumber lipgloss.Style

	styleHeadings [6]lipgloss.Style
	styleBold     lipgloss.Style
	styleItalic  lipgloss.Style
	styleCode    lipgloss.Style
	styleLink    lipgloss.Style
	styleQuote   lipgloss.Style
	styleListBul lipgloss.Style

	styleFrontmatterKey   lipgloss.Style
	styleFrontmatterValue lipgloss.Style
	styleFrontmatterFence lipgloss.Style

	styleTreeFolder lipgloss.Style
	styleTreeFile   lipgloss.Style
	styleTreeCursor lipgloss.Style

	stylePickerBox    lipgloss.Style
	stylePickerCursor lipgloss.Style
)

func init() {
	rebuildStyles()
}

func rebuildStyles() {
	colorBg = lipgloss.Color(active.bg)
	colorFg = lipgloss.Color(active.fg)
	colorText = colorFg
	colorMuted = lipgloss.Color(active.muted)
	colorDim = lipgloss.Color(active.dim)
	colorAccent = lipgloss.Color(active.accent)
	for i, hex := range active.headings {
		colorHeadings[i] = lipgloss.Color(hex)
	}
	colorCode = lipgloss.Color(active.codeInline)
	colorLink = lipgloss.Color(active.link)
	colorQuote = lipgloss.Color(active.quote)
	colorSelection = lipgloss.Color(active.selection)
	colorMatch = lipgloss.Color(active.matchBg)
	colorMatchFg = lipgloss.Color(active.matchFg)
	colorTreeCursor = lipgloss.Color(active.treeCursor)
	colorCursorLine = lipgloss.Color(active.cursorLine)
	colorStatus = lipgloss.Color(active.statusFg)

	styleStatusBar = lipgloss.NewStyle().Foreground(colorMuted).Padding(0, 1)
	styleStatusMode = lipgloss.NewStyle().Foreground(colorStatus).Bold(true)
	styleStatusDirty = lipgloss.NewStyle().Foreground(colorWarn)

	styleLineNumber = lipgloss.NewStyle().
		Foreground(colorDim).
		Width(4).
		Align(lipgloss.Right).
		MarginRight(1)

	styleCursorLineNumber = lipgloss.NewStyle().
		Foreground(colorMuted).
		Width(4).
		Align(lipgloss.Right).
		MarginRight(1)

	for i, c := range colorHeadings {
		styleHeadings[i] = lipgloss.NewStyle().Foreground(c).Bold(true)
	}
	styleBold = lipgloss.NewStyle().Foreground(colorFg).Bold(true)
	styleItalic = lipgloss.NewStyle().Foreground(colorFg).Italic(true)
	styleCode = lipgloss.NewStyle().Foreground(colorCode)
	styleLink = lipgloss.NewStyle().Foreground(colorLink).Underline(true)
	styleQuote = lipgloss.NewStyle().Foreground(colorQuote).Italic(true)
	styleListBul = lipgloss.NewStyle().Foreground(colorAccent)

	styleFrontmatterKey = lipgloss.NewStyle().Foreground(colorMuted)
	styleFrontmatterValue = lipgloss.NewStyle().Foreground(colorFg)
	styleFrontmatterFence = lipgloss.NewStyle().Foreground(colorDim)

	styleTreeFolder = lipgloss.NewStyle().Foreground(colorAccent)
	styleTreeFile = lipgloss.NewStyle().Foreground(colorFg)
	styleTreeCursor = lipgloss.NewStyle().Foreground(colorFg).Background(colorTreeCursor).Bold(true)

	stylePickerBox = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorMuted).
		Padding(0, 1)
	stylePickerCursor = lipgloss.NewStyle().Foreground(colorFg).Background(colorTreeCursor)
}

// withBg wraps the supplied style with the editor background, if the active
// theme defines one. Used so light / sepia themes paint a coherent
// background even on a dark terminal.
func withBg(s lipgloss.Style) lipgloss.Style {
	if active.bg == "" {
		return s
	}
	return s.Background(colorBg)
}
