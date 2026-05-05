package marg

import "github.com/charmbracelet/lipgloss"

// One restrained palette. Adjust here, nowhere else.
var (
	colorAccent     = lipgloss.Color("#7AA2F7") // soft blue
	colorSelection  = lipgloss.Color("#3B4252")
	colorTreeCursor = lipgloss.Color("#585B70") // brighter than dim, clearly highlights the row
	colorMatch      = lipgloss.Color("#735C00") // muted amber for search matches
	colorMuted   = lipgloss.Color("#5C6370")
	colorDim     = lipgloss.Color("#3B4048")
	colorText    = lipgloss.Color("#C8CCD4")
	colorWarn    = lipgloss.Color("#E5C07B")
	colorHeading = lipgloss.Color("#E5C07B")
	colorCode    = lipgloss.Color("#98C379")
	colorLink    = lipgloss.Color("#61AFEF")
)

var (
	styleStatusBar = lipgloss.NewStyle().
			Foreground(colorMuted).
			Padding(0, 1)

	styleStatusMode = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true)

	styleStatusDirty = lipgloss.NewStyle().
				Foreground(colorWarn)

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

	styleHeading = lipgloss.NewStyle().Foreground(colorHeading).Bold(true)
	styleBold    = lipgloss.NewStyle().Foreground(colorText).Bold(true)
	styleItalic  = lipgloss.NewStyle().Foreground(colorText).Italic(true)
	styleCode    = lipgloss.NewStyle().Foreground(colorCode)
	styleLink    = lipgloss.NewStyle().Foreground(colorLink).Underline(true)
	styleQuote   = lipgloss.NewStyle().Foreground(colorMuted).Italic(true)
	styleListBul = lipgloss.NewStyle().Foreground(colorAccent)

	styleTreeFolder = lipgloss.NewStyle().Foreground(colorAccent)
	styleTreeFile   = lipgloss.NewStyle().Foreground(colorText)
	styleTreeCursor = lipgloss.NewStyle().Foreground(colorText).Background(colorTreeCursor).Bold(true)

	stylePickerBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorMuted).
			Padding(0, 1)
	stylePickerCursor = lipgloss.NewStyle().Foreground(colorText).Background(colorDim)
)
