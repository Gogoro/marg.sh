package marg

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
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

	key := msg.String()
	switch key {
	case "j", "down":
		if t.cursor+1 < len(t.nodes) {
			t.cursor++
		}
	case "k", "up":
		if t.cursor > 0 {
			t.cursor--
		}
	case "g":
		t.cursor = 0
	case "G":
		t.cursor = len(t.nodes) - 1
	case "enter", "l", "right":
		if len(t.nodes) == 0 {
			return t, nil, ""
		}
		n := t.nodes[t.cursor]
		if n.isDir {
			t.expanded[n.path] = !t.expanded[n.path]
			t.refresh()
		} else {
			return t, nil, n.path
		}
	case "h", "left":
		if len(t.nodes) == 0 {
			return t, nil, ""
		}
		n := t.nodes[t.cursor]
		if n.isDir && t.expanded[n.path] {
			t.expanded[n.path] = false
			t.refresh()
		} else if n.depth > 0 {
			// jump to parent
			parent := filepath.Dir(n.path)
			for i, m := range t.nodes {
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
		if len(t.nodes) > 0 {
			n := t.nodes[t.cursor]
			t.confirm = fmt.Sprintf("delete %s? (y/N)", n.name)
		}
	case "R":
		t.refresh()
		t.flash = "refreshed"
	case "esc", "q", "ctrl+e":
		t.backRequested = true
	}

	t.scrollToCursor()
	return t, nil, ""
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
		if len(t.nodes) > 0 {
			n := t.nodes[t.cursor]
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
			}
		}
	}
	t.confirm = ""
	return t, nil, ""
}

func (t *tree) cursorToPath(path string) {
	for i, n := range t.nodes {
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
	if len(t.nodes) == 0 {
		return filepath.Join(t.root, name)
	}
	n := t.nodes[t.cursor]
	parent := n.path
	if !n.isDir {
		parent = filepath.Dir(n.path)
	}
	return filepath.Join(parent, name)
}

func (t *tree) scrollToCursor() {
	visible := t.height - 2 // header + prompt-or-help row
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
	b.WriteString(header)
	b.WriteString("\n")

	visible := t.height - 2
	end := t.scroll + visible
	if end > len(t.nodes) {
		end = len(t.nodes)
	}
	for i := t.scroll; i < end; i++ {
		n := t.nodes[i]
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
	if t.flash != "" {
		return styleStatusBar.Render(t.flash)
	}
	if transient != "" {
		return styleStatusBar.Render(transient)
	}
	help := "j/k move · enter open/expand · h collapse · % new file · d new dir · D delete · ctrl+p find"
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
		if err != nil || found {
			return nil
		}
		name := filepath.Base(p)
		if strings.HasPrefix(name, ".") && p != dir {
			if d != nil && d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d != nil && !d.IsDir() && isMarkdownPath(name) {
			found = true
		}
		return nil
	})
	return found
}
