package marg

import (
	"fmt"
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
	root       string   // single-root mode (project picker)
	roots      []string // multi-root mode (super mode)
	useTilde   bool     // when true, render paths under HOME with `~/`
	all        []string // absolute paths of every markdown file under the roots
	view       []string // paths matching current query, sorted hierarchically
	query      string

	cursor int
	width  int
	height int

	cancelled bool
	chosen    string

	// indexing is true while a super-mode walk is in flight. The picker
	// renders an "indexing…" message until the index lands.
	indexing bool
}

func newPicker() picker {
	return picker{}
}

func (p *picker) open(root string) {
	p.root = root
	p.roots = nil
	p.useTilde = false
	p.all = collectMarkdownFiles(root)
	p.resetState()
}

// openSuper sets up the picker for super-mode but does NOT walk yet — that
// happens asynchronously via indexCmd so the UI is responsive even on a
// huge home directory.
func (p *picker) openSuper(roots []string) {
	p.root = ""
	p.roots = roots
	p.useTilde = true
	p.all = nil
	p.indexing = true
	p.resetState()
}

// setIndex completes a super-mode walk and refreshes the visible list.
func (p *picker) setIndex(files []string) {
	p.all = files
	p.indexing = false
	p.applyQuery()
}

func (p *picker) resetState() {
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
		sort.Slice(p.view, func(i, j int) bool {
			return strings.ToLower(p.view[i]) < strings.ToLower(p.view[j])
		})
	} else {
		type scored struct {
			path  string
			score int
		}
		var hits []scored
		for _, path := range p.all {
			candidate := strings.ToLower(p.displayPath(path))
			score, ok := fuzzyScore(candidate, q)
			if !ok {
				continue
			}
			hits = append(hits, scored{path: path, score: score})
		}
		// Higher score first; alphabetical as a stable tiebreaker so equal
		// matches still feel orderly.
		sort.SliceStable(hits, func(i, j int) bool {
			if hits[i].score != hits[j].score {
				return hits[i].score > hits[j].score
			}
			return strings.ToLower(hits[i].path) < strings.ToLower(hits[j].path)
		})
		p.view = nil
		for _, h := range hits {
			p.view = append(p.view, h.path)
		}
	}
	if p.cursor >= len(p.view) {
		p.cursor = len(p.view) - 1
	}
	if p.cursor < 0 {
		p.cursor = 0
	}
}

// fuzzyScore rates how well candidate matches query. Both are lowercased;
// query is non-empty. Returns (score, true) if every rune of query
// appears in candidate in order, and (0, false) otherwise.
//
// Scoring favors matches that fall in the basename, are contiguous with
// the previous match, and land at the start of the basename. Path length
// is a mild tiebreaker so a shorter path edges out a longer one when the
// match quality is otherwise equal.
func fuzzyScore(candidate, query string) (int, bool) {
	cr := []rune(candidate)
	qr := []rune(query)

	baseStart := 0
	for i, r := range cr {
		if r == '/' {
			baseStart = i + 1
		}
	}

	score := 0
	si := 0
	prevMatch := -2
	for _, qc := range qr {
		for si < len(cr) && cr[si] != qc {
			si++
		}
		if si >= len(cr) {
			return 0, false
		}
		score += 10
		if si >= baseStart {
			score += 20
		}
		if si == baseStart {
			score += 30
		}
		if si == prevMatch+1 {
			score += 15
		}
		prevMatch = si
		si++
	}
	score -= len(cr) / 4
	return score, true
}

// displayPath returns the user-visible form of path: relative to root when
// the picker was opened on a single root, or `~/...` when in super mode.
func (p *picker) displayPath(path string) string {
	if p.useTilde {
		if home, err := os.UserHomeDir(); err == nil && strings.HasPrefix(path, home+string(os.PathSeparator)) {
			return "~" + path[len(home):]
		}
		return path
	}
	return relPath(p.root, path)
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
	if p.indexing {
		rows = append(rows, padRight("indexing…", boxW-2))
	} else if len(p.view) == 0 && p.query == "" {
		rows = append(rows, padRight("(no markdown files found)", boxW-2))
	}
	for i := start; i < end; i++ {
		rel := p.displayPath(p.view[i])
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

	count := ""
	if !p.indexing {
		count = fmt.Sprintf("  %d", len(p.view))
	}
	prompt := "› " + p.query + "_" + count
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
	if root == "" {
		return path
	}
	r, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return r
}

func collectMarkdownFiles(root string) []string {
	var out []string
	filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		name := filepath.Base(p)
		if d.IsDir() {
			if p != root && (strings.HasPrefix(name, ".") || isIgnoredDir(name)) {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(name, ".") {
			return nil
		}
		if isMarkdownPath(name) {
			out = append(out, p)
		}
		return nil
	})
	return out
}
