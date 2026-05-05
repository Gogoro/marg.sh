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

// treeNode is one row in the rendered tree: a folder or a markdown file.
type treeNode struct {
	path     string // absolute
	name     string
	depth    int
	isDir    bool
	expanded bool // only meaningful for folders
}

// tree is the full-screen file browser. It walks the project root, filters
// to markdown files (showing only folders that contain them transitively),
// and lets the user navigate / open / create / delete.
type tree struct {
	root   string
	nodes  []treeNode
	cursor int
	width  int
	height int
	scroll int

	// expand state keyed by absolute path. Default: all expanded.
	expanded map[string]bool

	// filter is the active substring query for `/` filter mode (lowercased).
	// When non-empty the tree renders `filtered` instead of `nodes`.
	filter        string
	filterEditing bool
	filtered      []treeNode

	// inline prompt shown at the bottom of the tree (e.g. "new file: foo.md").
	prompt    string
	promptIn  string
	promptCmd promptCmd
	confirm   string // pending y/N confirmation message

	flash string

	// set when user presses esc/q with intent to return to the editor.
	backRequested bool
}

type promptCmd int

const (
	promptNone promptCmd = iota
	promptNewFile
	promptNewDir
	promptDelete
)

func newTree(root string) tree {
	t := tree{root: root, expanded: map[string]bool{root: true}}
	t.refresh()
	return t
}

// newTreeLazy is for super mode: same root, but the initial walk is
// deferred until the user actually opens `:Ex`. Walking $HOME eagerly was
// adding 10+ seconds of wall time before the picker could render.
func newTreeLazy(root string) tree {
	return tree{root: root, expanded: map[string]bool{root: true}}
}

func (t *tree) refresh() {
	t.nodes = walkMarkdown(t.root, t.expanded)
	if t.cursor >= len(t.nodes) {
		t.cursor = len(t.nodes) - 1
	}
	if t.cursor < 0 {
		t.cursor = 0
	}
}

func (t *tree) resize(w, h int) {
	t.width = w
	t.height = h
}

func (t tree) update(msg tea.KeyMsg) (tree, tea.Cmd, string) {
	t.flash = ""
	if t.prompt != "" {
		return t.updatePrompt(msg)
	}
	if t.confirm != "" {
		return t.updateConfirm(msg)
	}
	if t.filterEditing {
		return t.updateFilter(msg)
	}

	key := msg.String()
	nodes := t.visibleNodes()
	switch key {
	case "j", "down":
		if t.cursor+1 < len(nodes) {
			t.cursor++
		}
	case "k", "up":
		if t.cursor > 0 {
			t.cursor--
		}
	case "g":
		t.cursor = 0
	case "G":
		t.cursor = len(nodes) - 1
	case "enter", "l", "right":
		if len(nodes) == 0 {
			return t, nil, ""
		}
		n := nodes[t.cursor]
		if n.isDir {
			t.expanded[n.path] = !t.expanded[n.path]
			t.refresh()
			t.recomputeFiltered()
		} else {
			return t, nil, n.path
		}
	case "h", "left":
		if len(nodes) == 0 {
			return t, nil, ""
		}
		n := nodes[t.cursor]
		if n.isDir && t.expanded[n.path] {
			t.expanded[n.path] = false
			t.refresh()
			t.recomputeFiltered()
		} else if n.depth > 0 {
			parent := filepath.Dir(n.path)
			for i, m := range nodes {
				if m.path == parent {
					t.cursor = i
					break
				}
			}
		}
	case "%":
		t.prompt = "new file"
		t.promptIn = ""
		t.promptCmd = promptNewFile
	case "d":
		t.prompt = "new dir"
		t.promptIn = ""
		t.promptCmd = promptNewDir
	case "D":
		if len(nodes) > 0 {
			n := nodes[t.cursor]
			t.confirm = fmt.Sprintf("delete %s? (y/N)", n.name)
		}
	case "R":
		t.refresh()
		t.flash = "refreshed"
	case "/":
		t.filterEditing = true
	case "esc", "q", "ctrl+e":
		if t.filter != "" {
			t.filter = ""
			t.filtered = nil
			t.cursor = 0
			t.scroll = 0
			return t, nil, ""
		}
		t.backRequested = true
	}

	t.scrollToCursor()
	return t, nil, ""
}

func (t tree) updateFilter(msg tea.KeyMsg) (tree, tea.Cmd, string) {
	switch msg.String() {
	case "esc":
		t.filterEditing = false
		t.filter = ""
		t.filtered = nil
		t.cursor = 0
		t.scroll = 0
	case "enter":
		t.filterEditing = false
		// Keep the filter and let the user navigate the narrowed list.
	case "backspace":
		if len(t.filter) > 0 {
			t.filter = t.filter[:len(t.filter)-1]
			t.recomputeFiltered()
		} else {
			t.filterEditing = false
		}
	default:
		if len(msg.Runes) > 0 {
			t.filter += string(msg.Runes)
			t.recomputeFiltered()
		}
	}
	return t, nil, ""
}

// recomputeFiltered walks the tree from root, picks every markdown file whose
// path contains the lowercased filter, and includes their ancestor folders so
// the tree structure still reads correctly.
func (t *tree) recomputeFiltered() {
	if t.filter == "" {
		t.filtered = nil
		return
	}
	q := strings.ToLower(t.filter)

	matchingFile := map[string]bool{}
	matchingDir := map[string]bool{}
	filepath.WalkDir(t.root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		name := filepath.Base(p)
		if strings.HasPrefix(name, ".") && p != t.root {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() || !isMarkdownPath(name) {
			return nil
		}
		rel, _ := filepath.Rel(t.root, p)
		if !strings.Contains(strings.ToLower(rel), q) {
			return nil
		}
		matchingFile[p] = true
		dir := filepath.Dir(p)
		for dir != t.root && len(dir) > len(t.root) {
			matchingDir[dir] = true
			dir = filepath.Dir(dir)
		}
		return nil
	})

	t.filtered = nil
	var rec func(dir string, depth int)
	rec = func(dir string, depth int) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return
		}
		var dirs, files []os.DirEntry
		for _, e := range entries {
			name := e.Name()
			if strings.HasPrefix(name, ".") {
				continue
			}
			full := filepath.Join(dir, name)
			if e.IsDir() {
				if matchingDir[full] {
					dirs = append(dirs, e)
				}
			} else if matchingFile[full] {
				files = append(files, e)
			}
		}
		sort.Slice(dirs, func(i, j int) bool { return dirs[i].Name() < dirs[j].Name() })
		sort.Slice(files, func(i, j int) bool { return files[i].Name() < files[j].Name() })
		for _, d := range dirs {
			path := filepath.Join(dir, d.Name())
			t.filtered = append(t.filtered, treeNode{path: path, name: d.Name(), depth: depth, isDir: true})
			rec(path, depth+1)
		}
		for _, f := range files {
			path := filepath.Join(dir, f.Name())
			t.filtered = append(t.filtered, treeNode{path: path, name: f.Name(), depth: depth, isDir: false})
		}
	}
	rec(t.root, 0)

	if t.cursor >= len(t.filtered) {
		t.cursor = len(t.filtered) - 1
	}
	if t.cursor < 0 {
		t.cursor = 0
	}
}

// visibleNodes returns the slice the tree is currently rendering — the
// filtered view if a filter is active, otherwise the full tree.
func (t *tree) visibleNodes() []treeNode {
	if t.filter != "" {
		return t.filtered
	}
	return t.nodes
}

func (t tree) updatePrompt(msg tea.KeyMsg) (tree, tea.Cmd, string) {
	switch msg.String() {
	case "esc":
		t.prompt = ""
		t.promptIn = ""
	case "enter":
		name := strings.TrimSpace(t.promptIn)
		t.prompt = ""
		t.promptIn = ""
		if name == "" {
			return t, nil, ""
		}
		switch t.promptCmd {
		case promptNewFile:
			path := t.targetPathForNew(name)
			if filepath.Ext(path) == "" {
				path += ".md"
			}
			if err := os.WriteFile(path, []byte{}, 0o644); err != nil {
				t.flash = "create failed: " + err.Error()
			} else {
				t.refresh()
				t.cursorToPath(path)
				return t, nil, path
			}
		case promptNewDir:
			path := t.targetPathForNew(name)
			if err := os.MkdirAll(path, 0o755); err != nil {
				t.flash = "mkdir failed: " + err.Error()
			} else {
				t.expanded[path] = true
				t.refresh()
				t.cursorToPath(path)
			}
		}
	case "backspace":
		if len(t.promptIn) > 0 {
			t.promptIn = t.promptIn[:len(t.promptIn)-1]
		}
	default:
		if len(msg.Runes) > 0 {
			t.promptIn += string(msg.Runes)
		}
	}
	return t, nil, ""
}

func (t tree) updateConfirm(msg tea.KeyMsg) (tree, tea.Cmd, string) {
	switch msg.String() {
	case "y", "Y":
		nodes := t.visibleNodes()
		if len(nodes) > 0 {
			n := nodes[t.cursor]
			var err error
			if n.isDir {
				err = os.RemoveAll(n.path)
			} else {
				err = os.Remove(n.path)
			}
			if err != nil {
				t.flash = "delete failed: " + err.Error()
			} else {
				t.flash = "deleted"
				t.refresh()
				t.recomputeFiltered()
			}
		}
	}
	t.confirm = ""
	return t, nil, ""
}

func (t *tree) cursorToPath(path string) {
	for i, n := range t.visibleNodes() {
		if n.path == path {
			t.cursor = i
			t.scrollToCursor()
			return
		}
	}
}

// targetPathForNew returns where a new file/folder should be created relative
// to the currently selected node. If a folder is selected, create inside it;
// if a file is selected, create alongside it.
func (t *tree) targetPathForNew(name string) string {
	nodes := t.visibleNodes()
	if len(nodes) == 0 {
		return filepath.Join(t.root, name)
	}
	n := nodes[t.cursor]
	parent := n.path
	if !n.isDir {
		parent = filepath.Dir(n.path)
	}
	return filepath.Join(parent, name)
}

func (t *tree) scrollToCursor() {
	visible := t.height - 2
	if visible < 1 {
		visible = 1
	}
	if t.cursor < t.scroll {
		t.scroll = t.cursor
	} else if t.cursor >= t.scroll+visible {
		t.scroll = t.cursor - visible + 1
	}
	if t.scroll < 0 {
		t.scroll = 0
	}
}

func (t *tree) view() string {
	if t.height == 0 {
		return ""
	}
	var b strings.Builder
	header := styleStatusMode.Render("FILES") + "  " + relRoot(t.root)
	if t.filter != "" {
		header += "  " + lipgloss.NewStyle().Foreground(colorMuted).Render("filter: "+t.filter)
	}
	b.WriteString(header)
	b.WriteString("\n")

	nodes := t.visibleNodes()
	visible := t.height - 2
	end := t.scroll + visible
	if end > len(nodes) {
		end = len(nodes)
	}
	for i := t.scroll; i < end; i++ {
		n := nodes[i]
		b.WriteString(t.renderNode(n, i == t.cursor))
		b.WriteString("\n")
	}
	for i := 0; i < visible-(end-t.scroll); i++ {
		b.WriteString("\n")
	}
	return b.String()
}

func (t *tree) renderNode(n treeNode, isCursor bool) string {
	indent := strings.Repeat("  ", n.depth)
	var glyph, name string
	if n.isDir {
		if t.expanded[n.path] {
			glyph = "▾ "
		} else {
			glyph = "▸ "
		}
		name = n.name + "/"
		line := indent + glyph + styleTreeFolder.Render(name)
		if isCursor {
			return styleTreeCursor.Render(stripStyle(line))
		}
		return line
	}
	glyph = "  "
	name = n.name
	line := indent + glyph + styleTreeFile.Render(name)
	if isCursor {
		return styleTreeCursor.Render(stripStyle(line))
	}
	return line
}

// stripStyle is a hack: when highlighting the cursor row, we want a single
// background across the row, so we render without the inner colors. For v1
// just take the visible characters; lipgloss handles ANSI safely.
func stripStyle(s string) string {
	// Lip Gloss styles are ANSI-wrapped; rendering plain text preserves layout.
	// Strip ANSI by walking out of escape sequences.
	var b strings.Builder
	in := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if in {
			if c == 'm' {
				in = false
			}
			continue
		}
		if c == 0x1b && i+1 < len(s) && s[i+1] == '[' {
			in = true
			i++
			continue
		}
		b.WriteByte(c)
	}
	return b.String()
}

func relRoot(root string) string {
	if cwd, err := os.Getwd(); err == nil {
		if r, err := filepath.Rel(cwd, root); err == nil && !strings.HasPrefix(r, "..") {
			if r == "." {
				return root
			}
			return r
		}
	}
	return root
}

func (t *tree) statusBar(width int, transient string) string {
	if t.prompt != "" {
		return styleStatusBar.Render(t.prompt + ": " + t.promptIn + "_")
	}
	if t.confirm != "" {
		return styleStatusBar.Render(t.confirm)
	}
	if t.filterEditing {
		return styleStatusBar.Render("/" + t.filter + "_")
	}
	if t.flash != "" {
		return styleStatusBar.Render(t.flash)
	}
	if transient != "" {
		return styleStatusBar.Render(transient)
	}
	help := "j/k move · enter open/expand · h collapse · / filter · % new file · d new dir · D delete · ctrl+p find"
	return styleStatusBar.Render(help)
}

// --- walking ---

// walkMarkdown returns a flattened, ordered list of tree nodes rooted at
// `root`. Folders only appear if they contain markdown files (transitively).
// Folders are listed before files at each depth, alphabetically.
func walkMarkdown(root string, expanded map[string]bool) []treeNode {
	var out []treeNode
	var rec func(dir string, depth int)
	rec = func(dir string, depth int) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return
		}
		var dirs, files []os.DirEntry
		for _, e := range entries {
			name := e.Name()
			if strings.HasPrefix(name, ".") {
				continue
			}
			if e.IsDir() {
				if dirContainsMarkdown(filepath.Join(dir, name)) {
					dirs = append(dirs, e)
				}
			} else if isMarkdownPath(name) {
				files = append(files, e)
			}
		}
		sort.Slice(dirs, func(i, j int) bool { return dirs[i].Name() < dirs[j].Name() })
		sort.Slice(files, func(i, j int) bool { return files[i].Name() < files[j].Name() })

		for _, d := range dirs {
			path := filepath.Join(dir, d.Name())
			node := treeNode{
				path:     path,
				name:     d.Name(),
				depth:    depth,
				isDir:    true,
				expanded: expanded[path],
			}
			out = append(out, node)
			if expanded[path] {
				rec(path, depth+1)
			}
		}
		for _, f := range files {
			path := filepath.Join(dir, f.Name())
			out = append(out, treeNode{
				path:  path,
				name:  f.Name(),
				depth: depth,
				isDir: false,
			})
		}
	}
	rec(root, 0)
	return out
}

func dirContainsMarkdown(dir string) bool {
	found := false
	filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		name := filepath.Base(p)
		if d != nil && d.IsDir() {
			if p != dir && (strings.HasPrefix(name, ".") || noiseDirs[name]) {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.IsDir() && isMarkdownPath(name) {
			found = true
			return filepath.SkipAll
		}
		return nil
	})
	return found
}
