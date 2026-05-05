package marg

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// picker is the Ctrl+P fuzzy file picker. Modal overlay; centered box; input
// at bottom; list of matched files above sorted by hierarchical path.
type picker struct {
	root  string
	all   []string // absolute paths of every markdown file under root
	view  []string // paths matching current query, sorted hierarchically
	query string

	cursor int
	width  int
	height int

	cancelled bool
	chosen    string
}

func newPicker() picker {
	return picker{}
}

func (p *picker) open(root string) {
	p.root = root
	p.all = collectMarkdownFiles(root)
	p.query = ""
	p.cursor = 0
	p.cancelled = false
	p.chosen = ""
	p.applyQuery()
}

func (p *picker) resize(w, h int) {
	p.width = w
	p.height = h
}

func (p picker) update(msg tea.KeyMsg) (picker, tea.Cmd) {
	switch msg.String() {
	case "esc", "ctrl+c":
		p.cancelled = true
	case "enter":
		if p.cursor >= 0 && p.cursor < len(p.view) {
			p.chosen = p.view[p.cursor]
		} else {
			p.cancelled = true
		}
	case "up", "ctrl+k", "ctrl+p":
		if p.cursor > 0 {
			p.cursor--
		}
	case "down", "ctrl+j", "ctrl+n":
		if p.cursor+1 < len(p.view) {
			p.cursor++
		}
	case "backspace":
		if len(p.query) > 0 {
			p.query = p.query[:len(p.query)-1]
			p.applyQuery()
		}
	default:
		if len(msg.Runes) > 0 {
			p.query += string(msg.Runes)
			p.applyQuery()
		}
	}
	return p, nil
}

func (p *picker) applyQuery() {
	q := strings.ToLower(strings.TrimSpace(p.query))
	if q == "" {
		p.view = append([]string{}, p.all...)
	} else {
		p.view = nil
		for _, path := range p.all {
			rel := relPathLower(p.root, path)
			if subsequenceMatch(rel, q) {
				p.view = append(p.view, path)
			}
		}
	}
	// Hierarchical sort: by full relative path, lowercased.
	sort.Slice(p.view, func(i, j int) bool {
		return strings.ToLower(p.view[i]) < strings.ToLower(p.view[j])
	})
	if p.cursor >= len(p.view) {
		p.cursor = len(p.view) - 1
	}
	if p.cursor < 0 {
		p.cursor = 0
	}
}

// overlay paints the picker box on top of the existing rendered view.
// We pad the box to a centered position roughly 60% width.
func (p *picker) overlay(below string) string {
	if p.width == 0 || p.height == 0 {
		return below
	}
	boxW := p.width * 6 / 10
	if boxW < 40 {
		boxW = 40
	}
	if boxW > p.width-4 {
		boxW = p.width - 4
	}
	boxH := p.height * 6 / 10
	if boxH < 10 {
		boxH = 10
	}
	if boxH > p.height-2 {
		boxH = p.height - 2
	}

	listH := boxH - 2 // input row + a separator row
	if listH < 1 {
		listH = 1
	}

	// Sliding window so the cursor stays visible.
	start := 0
	if p.cursor >= listH {
		start = p.cursor - listH + 1
	}
	end := start + listH
	if end > len(p.view) {
		end = len(p.view)
	}

	var rows []string
	for i := start; i < end; i++ {
		rel := relPath(p.root, p.view[i])
		row := truncate(rel, boxW-2)
		if i == p.cursor {
			row = stylePickerCursor.Render(padRight(row, boxW-2))
		} else {
			row = padRight(row, boxW-2)
		}
		rows = append(rows, row)
	}
	for len(rows) < listH {
		rows = append(rows, padRight("", boxW-2))
	}

	prompt := "› " + p.query + "_"
	prompt = padRight(prompt, boxW-2)

	body := strings.Join(rows, "\n") + "\n" + lipgloss.NewStyle().Foreground(colorMuted).Render(strings.Repeat("─", boxW-2)) + "\n" + prompt
	box := stylePickerBox.Width(boxW).Render(body)

	return placeOver(below, box, p.width, p.height)
}

func placeOver(below, box string, w, h int) string {
	belowLines := strings.Split(below, "\n")
	for len(belowLines) < h {
		belowLines = append(belowLines, "")
	}
	boxLines := strings.Split(box, "\n")
	boxH := len(boxLines)
	boxW := 0
	for _, l := range boxLines {
		if lw := lipgloss.Width(l); lw > boxW {
			boxW = lw
		}
	}
	startY := (h - boxH) / 2
	startX := (w - boxW) / 2
	if startY < 0 {
		startY = 0
	}
	if startX < 0 {
		startX = 0
	}

	for i, bl := range boxLines {
		y := startY + i
		if y >= len(belowLines) {
			break
		}
		belowLines[y] = overlayLine(belowLines[y], bl, startX, w)
	}
	return strings.Join(belowLines, "\n")
}

// overlayLine pastes `top` onto `base` at column `x`. Both may contain ANSI;
// for v1 we render conservatively: pad base to width with spaces, then take
// base[:x] + top + base[x+visualWidth(top):]. ANSI in base before x might
// bleed; acceptable trade-off to keep this simple.
func overlayLine(base, top string, x, w int) string {
	// Strip ANSI from base for safe slicing, then re-pad. Loses color under
	// the picker, which is fine because the picker covers it.
	plain := stripStyle(base)
	// Pad with spaces to width.
	if cw := lipgloss.Width(plain); cw < w {
		plain += strings.Repeat(" ", w-cw)
	}
	runes := []rune(plain)
	left := string(runes[:min(x, len(runes))])
	topW := lipgloss.Width(top)
	rightStart := x + topW
	right := ""
	if rightStart < len(runes) {
		right = string(runes[rightStart:])
	}
	return left + top + right
}

func truncate(s string, w int) string {
	if w <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= w {
		return s
	}
	return string(r[:w-1]) + "…"
}

func padRight(s string, w int) string {
	cw := lipgloss.Width(s)
	if cw >= w {
		return s
	}
	return s + strings.Repeat(" ", w-cw)
}

func relPath(root, path string) string {
	r, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return r
}

func relPathLower(root, path string) string {
	return strings.ToLower(relPath(root, path))
}

// subsequenceMatch returns true if every rune of `query` appears in `s`
// in order (not necessarily contiguous). Cheap fuzzy.
func subsequenceMatch(s, query string) bool {
	if query == "" {
		return true
	}
	si := 0
	sr := []rune(s)
	qr := []rune(query)
	for _, q := range qr {
		found := false
		for si < len(sr) {
			if sr[si] == q {
				si++
				found = true
				break
			}
			si++
		}
		if !found {
			return false
		}
	}
	return true
}

func collectMarkdownFiles(root string) []string {
	var out []string
	filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		name := filepath.Base(p)
		if strings.HasPrefix(name, ".") && p != root {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if isMarkdownPath(name) {
			out = append(out, p)
		}
		return nil
	})
	return out
}
